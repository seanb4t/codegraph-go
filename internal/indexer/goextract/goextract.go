package goextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Extract walks a single Go file's parsed syntax tree and produces its
// Pass-1 intermediate: every node it declares (LANG-01 vocabulary, D-06),
// every intra-file edge those nodes participate in, and every unresolved
// cross-file reference (calls/imports/embeds/contains) Pass 2 must later
// settle against a global symbol index.
//
// Size/skip contract (RESEARCH Pitfall 4, threat T-02-03): p.Parse owns
// the parser.MaxSourceBytes ceiling. If Parse fails for ANY reason —
// parser.ErrSourceTooLarge or otherwise — that failure is recorded on the
// returned FileResult.Err and Extract itself returns a nil error, so one
// bad file never aborts a caller's batch.
func Extract(p parser.Parser, importPath, relPath string, src []byte) (FileResult, error) {
	sum := sha256.Sum256(src)
	result := FileResult{
		ImportPath:       importPath,
		RelPath:          relPath,
		Language:         "go",
		ContentHash:      hex.EncodeToString(sum[:]),
		Imports:          make(map[string]string),
		InterfaceMethods: make(map[string][]MethodSpec),
		MethodArity:      make(map[string]int32),
	}

	tree, err := p.Parse(src, nil)
	if err != nil {
		result.Err = err
		return result, nil
	}
	defer tree.Close()

	native, ok := tree.Inner().(*tree_sitter.Tree)
	if !ok || native == nil {
		result.Err = fmt.Errorf("goextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("goextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "go",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Types are collected first so a same-file method's receiver-type
	// lookup (below) always sees every type this file declares,
	// regardless of declaration order in the source.
	ex.collectTypes(root)
	ex.collectConstsVars(root)
	ex.collectImports(root)
	ex.collectFuncsAndMethods(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *FileResult
	typeNodesByName map[string]string
}

// --- type declarations (struct / interface / type_alias) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() != "type_declaration" {
			continue
		}
		for j := uint(0); j < decl.NamedChildCount(); j++ {
			spec := decl.NamedChild(j)
			// A grouped or single type declaration wraps one or more
			// specs. A true alias ("type A = B") is its own node kind,
			// "type_alias" — distinct from a type definition's
			// "type_spec" — but both carry "name"/"type" fields and
			// both map to KindTypeAlias unless the underlying type is a
			// struct_type/interface_type (D-06's "type_alias" bucket
			// covers both shapes; this extractor does not distinguish
			// definition from true alias in the emitted node kind).
			if spec.Kind() == "type_spec" || spec.Kind() == "type_alias" {
				ex.emitTypeSpec(spec)
			}
		}
	}
}

func (ex *extractor) emitTypeSpec(spec *tree_sitter.Node) {
	nameNode := spec.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = spec.NamedChild(0)
	}
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)

	typeNode := spec.ChildByFieldName("type")
	if typeNode == nil && spec.NamedChildCount() > 1 {
		typeNode = spec.NamedChild(1)
	}

	kind := KindTypeAlias
	if typeNode != nil {
		switch typeNode.Kind() {
		case "struct_type":
			kind = KindStruct
		case "interface_type":
			kind = KindInterface
		}
	}

	id := nodeid.NodeID(kind, name, ex.relPath)
	start := spec.StartPosition()
	end := spec.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "go",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    visibilityFor(name),
		IsExported:    isExported(name),
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.typeNodesByName[name] = id

	switch kind {
	case KindStruct:
		if typeNode != nil {
			ex.collectStructEmbeds(id, typeNode)
		}
	case KindInterface:
		if typeNode != nil {
			ex.collectInterfaceEmbeds(id, typeNode)
			ex.collectInterfaceMethods(id, typeNode)
		}
	}
}

func (ex *extractor) collectStructEmbeds(structID string, structType *tree_sitter.Node) {
	var fieldList *tree_sitter.Node
	for i := uint(0); i < structType.NamedChildCount(); i++ {
		if c := structType.NamedChild(i); c.Kind() == "field_declaration_list" {
			fieldList = c
			break
		}
	}
	if fieldList == nil {
		return
	}
	for i := uint(0); i < fieldList.NamedChildCount(); i++ {
		fd := fieldList.NamedChild(i)
		if fd.Kind() != "field_declaration" {
			continue
		}
		// An embedded field is a field_declaration with exactly one
		// named child that is itself a type reference (type_identifier,
		// qualified_type, or a pointer_type wrapping either) and NO
		// separate field-name child before it.
		if fd.NamedChildCount() != 1 {
			continue
		}
		t := fd.NamedChild(0)
		if t.Kind() == "pointer_type" {
			if t.NamedChildCount() == 0 {
				continue
			}
			t = t.NamedChild(0)
		}
		name, pkgAlias, ok := typeRefName(t, ex.src)
		if !ok {
			continue
		}
		pos := fd.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: structID, Name: name, PkgAlias: pkgAlias, Kind: RefKindEmbeds,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}
}

// collectInterfaceEmbeds finds embedded-type references among an
// interface_type's direct named children. This grammar version (verified
// via a live parse this session, not just the docs) represents each
// embedded entry as its own "type_elem" node — a sibling of the
// interface's own "method_elem" method-signature nodes, not a nested
// "type_list"/"method_declaration_list" pair.
func (ex *extractor) collectInterfaceEmbeds(interfaceID string, interfaceType *tree_sitter.Node) {
	for i := uint(0); i < interfaceType.NamedChildCount(); i++ {
		elem := interfaceType.NamedChild(i)
		if elem.Kind() != "type_elem" {
			continue
		}
		if elem.NamedChildCount() == 0 {
			continue
		}
		t := elem.NamedChild(0)
		name, pkgAlias, ok := typeRefName(t, ex.src)
		if !ok {
			continue
		}
		pos := elem.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: interfaceID, Name: name, PkgAlias: pkgAlias, Kind: RefKindEmbeds,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}
}

// collectInterfaceMethods records each interface_type's own "method_elem"
// method-signature nodes (siblings of the "type_elem" embed nodes
// collectInterfaceEmbeds walks — verified via the same live parse this
// session) into result.InterfaceMethods, keyed by interfaceID (Phase 5
// RES-02/Pattern 3). Embedded interfaces' method specs are intentionally
// NOT flattened in here — dispatch.SynthesizeImplements composes them at
// resolve time via the "embeds" edges collectInterfaceEmbeds' unresolved
// refs eventually become.
func (ex *extractor) collectInterfaceMethods(interfaceID string, interfaceType *tree_sitter.Node) {
	for i := uint(0); i < interfaceType.NamedChildCount(); i++ {
		elem := interfaceType.NamedChild(i)
		if elem.Kind() != "method_elem" {
			continue
		}
		nameNode := elem.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		spec := MethodSpec{
			Name:  nameNode.Utf8Text(ex.src),
			Arity: countParams(elem.ChildByFieldName("parameters"), ex.src),
		}
		ex.result.InterfaceMethods[interfaceID] = append(ex.result.InterfaceMethods[interfaceID], spec)
	}
}

// countParams counts a parameter_list's actual declared parameters (Phase
// 5 RES-02/Pattern 3's "arity" bound) — NOT its named-child count, since a
// single parameter_declaration node may group several identifiers under
// one shared type ("a, b int" is 2 parameters, one node). A parameter
// with no identifier children at all (an unnamed parameter, e.g. an
// interface method spec's "(int, string)") counts as exactly 1.
func countParams(paramsNode *tree_sitter.Node, src []byte) int32 {
	if paramsNode == nil {
		return 0
	}
	var count int32
	for i := uint(0); i < paramsNode.NamedChildCount(); i++ {
		p := paramsNode.NamedChild(i)
		switch p.Kind() {
		case "parameter_declaration", "variadic_parameter_declaration":
			names := int32(0)
			for j := uint(0); j < p.NamedChildCount(); j++ {
				if p.NamedChild(j).Kind() == "identifier" {
					names++
				}
			}
			if names == 0 {
				names = 1
			}
			count += names
		}
	}
	return count
}

// typeRefName extracts the (name, pkgAlias) of a type_identifier or
// qualified_type node — the two shapes a struct/interface embed or an
// interface's embedded-type list entry can take.
func typeRefName(t *tree_sitter.Node, src []byte) (name, pkgAlias string, ok bool) {
	switch t.Kind() {
	case "type_identifier":
		return t.Utf8Text(src), "", true
	case "qualified_type":
		pkg := t.ChildByFieldName("package")
		typ := t.ChildByFieldName("name")
		if pkg == nil {
			pkg = t.NamedChild(0)
		}
		if typ == nil && t.NamedChildCount() > 1 {
			typ = t.NamedChild(1)
		}
		if pkg == nil || typ == nil {
			return "", "", false
		}
		return typ.Utf8Text(src), pkg.Utf8Text(src), true
	default:
		return "", "", false
	}
}

// namedTypeRef resolves a type expression node to a (name, pkgAlias) pair
// IF it is a single named type reference — type_identifier or
// qualified_type, optionally wrapped in exactly one pointer_type level (a
// Go idiom: *T is still "the type T" for D-09 purposes). Any other shape
// (slice_type, map_type, array_type, generic_type, a multi-value
// parameter_list, etc.) returns ok=false — RESEARCH §B's documented
// precision note: these compound/unnamed type shapes are not resolved to
// a single target node; absence, not error.
func namedTypeRef(t *tree_sitter.Node, src []byte) (name, pkgAlias string, ok bool) {
	if t == nil {
		return "", "", false
	}
	if t.Kind() == "pointer_type" {
		if t.NamedChildCount() == 0 {
			return "", "", false
		}
		t = t.NamedChild(0)
	}
	name, pkgAlias, ok = typeRefName(t, src)
	if ok && pkgAlias == "" && isGoPredeclaredType(name) {
		// Go's tree-sitter grammar has no separate "primitive_type" node
		// kind — a built-in type name ("int", "string", "error", ...) is
		// syntactically indistinguishable from a user-defined
		// type_identifier. Filtering the predeclared-identifier set here
		// (rather than leaving it for Pass 2 to silently fail to resolve)
		// avoids inflating unresolvedCount with spurious noise for every
		// primitive-typed var/return in the whole indexed tree — a
		// resolve.go concern this extractor can cheaply short-circuit
		// instead.
		return "", "", false
	}
	return name, pkgAlias, ok
}

// isGoPredeclaredType reports whether name is one of Go's predeclared
// type identifiers (the Go spec's "Predeclared identifiers" §Types list)
// — never a user-declared, in-repo type this extractor's vocabulary could
// resolve.
func isGoPredeclaredType(name string) bool {
	switch name {
	case "bool", "byte", "complex64", "complex128", "error",
		"float32", "float64",
		"int", "int8", "int16", "int32", "int64",
		"rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"any", "comparable":
		return true
	default:
		return false
	}
}

// --- const / var declarations ---

func (ex *extractor) collectConstsVars(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		var kind string
		switch decl.Kind() {
		case "const_declaration":
			kind = KindConstant
		case "var_declaration":
			kind = KindVariable
		default:
			continue
		}
		for j := uint(0); j < decl.NamedChildCount(); j++ {
			spec := decl.NamedChild(j)
			if spec.Kind() == "const_spec" || spec.Kind() == "var_spec" {
				ex.emitConstVarSpec(kind, spec)
			}
		}
	}
}

func (ex *extractor) emitConstVarSpec(kind string, spec *tree_sitter.Node) {
	pos := spec.StartPosition()
	end := spec.EndPosition()
	// D-09's type_of ref (RESEARCH §B) applies only to KindVariable specs
	// with an explicit declared type ("var x T", not "var x = expr"),
	// per the plan's behavior spec. Constants are out of scope here
	// (not mentioned in the D-09 behavior list), and struct FIELD
	// type_of is a documented Go D-02 divergence — this extractor emits
	// no "field" node at all (the pre-existing ratified skip named at
	// this package's KindFile-block doc comment), so a field has no
	// FromID to anchor a type_of ref on.
	var typeNode *tree_sitter.Node
	if kind == KindVariable {
		typeNode = spec.ChildByFieldName("type")
	}
	for k := uint(0); k < spec.NamedChildCount(); k++ {
		c := spec.NamedChild(k)
		if c.Kind() != "identifier" {
			// The leading run of identifier children are the declared
			// names (a spec may declare several: "a, b = 1, 2"); the
			// first non-identifier child is the type or value list,
			// which ends that run.
			break
		}
		name := c.Utf8Text(ex.src)
		if name == "_" {
			continue
		}
		id := nodeid.NodeID(kind, name, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, ExtractedNode{Node: &schema.Node{
			Id:            id,
			Kind:          kind,
			Name:          name,
			QualifiedName: name,
			FilePath:      ex.relPath,
			Language:      "go",
			StartLine:     int32(pos.Row) + 1,
			EndLine:       int32(end.Row) + 1,
			StartCol:      int32(pos.Column),
			EndCol:        int32(end.Column),
			Visibility:    visibilityFor(name),
			IsExported:    isExported(name),
		}})
		ex.result.IntraEdges = append(ex.result.IntraEdges, IntraEdge{Edge: &schema.Edge{
			Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		if typeNode != nil {
			ex.emitTypeOfRef(id, typeNode)
		}
	}
}

// emitTypeOfRef emits a D-09 `type_of` Pass-1 ref (RESEARCH §B) from a
// variable's own id to its declared type's name, when that type resolves
// to a single named type reference (namedTypeRef). An un-annotated or
// compound/unnamed-type declaration emits no ref — absence, not error.
func (ex *extractor) emitTypeOfRef(fromID string, typeNode *tree_sitter.Node) {
	name, pkgAlias, ok := namedTypeRef(typeNode, ex.src)
	if !ok {
		return
	}
	pos := typeNode.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: RefKindTypeOf,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// --- imports ---

func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() != "import_declaration" {
			continue
		}
		for j := uint(0); j < decl.NamedChildCount(); j++ {
			c := decl.NamedChild(j)
			switch c.Kind() {
			case "import_spec":
				ex.emitImportSpec(c)
			case "import_spec_list":
				for k := uint(0); k < c.NamedChildCount(); k++ {
					if spec := c.NamedChild(k); spec.Kind() == "import_spec" {
						ex.emitImportSpec(spec)
					}
				}
			}
		}
	}
}

func (ex *extractor) emitImportSpec(spec *tree_sitter.Node) {
	var pathNode, aliasNode *tree_sitter.Node
	for i := uint(0); i < spec.NamedChildCount(); i++ {
		c := spec.NamedChild(i)
		switch c.Kind() {
		case "interpreted_string_literal", "raw_string_literal":
			pathNode = c
		case "package_identifier", "dot", "blank_identifier":
			aliasNode = c
		}
	}
	if pathNode == nil {
		return
	}
	path := unquoteImportPath(pathNode.Utf8Text(ex.src))

	var alias string
	switch {
	case aliasNode == nil:
		alias = defaultPackageAlias(path)
	case aliasNode.Kind() == "dot":
		alias = "."
	case aliasNode.Kind() == "blank_identifier":
		alias = "_"
	default:
		alias = aliasNode.Utf8Text(ex.src)
	}
	ex.result.Imports[alias] = path

	pos := spec.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
		FromID: ex.fileID, Name: path, PkgAlias: alias, Kind: RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

func defaultPackageAlias(importPath string) string {
	if i := strings.LastIndexByte(importPath, '/'); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}

func unquoteImportPath(text string) string {
	if len(text) >= 2 && text[0] == '`' {
		return text[1 : len(text)-1]
	}
	if unquoted, err := strconv.Unquote(text); err == nil {
		return unquoted
	}
	return text
}

// --- functions / methods / calls ---

func (ex *extractor) collectFuncsAndMethods(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		switch decl.Kind() {
		case "function_declaration":
			ex.emitFunction(decl)
		case "method_declaration":
			ex.emitMethod(decl)
		}
	}
}

func (ex *extractor) emitFunction(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(KindFunction, name, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, ExtractedNode{Node: ex.buildSymbolNode(KindFunction, id, name, name, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.collectReturnTypeRef(id, decl)

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
		ex.collectReferencesAndInstantiates(id, body)
	}
}

func (ex *extractor) emitMethod(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)

	recvType, hasRecv := receiverTypeName(decl, ex.src)
	qualifiedName := name
	if hasRecv {
		qualifiedName = recvType + "." + name
	}
	id := nodeid.NodeID(KindMethod, qualifiedName, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, ExtractedNode{Node: ex.buildSymbolNode(KindMethod, id, name, qualifiedName, decl)})
	ex.result.MethodArity[id] = countParams(decl.ChildByFieldName("parameters"), ex.src)

	switch {
	case hasRecv && ex.typeNodesByName[recvType] != "":
		// The receiver's type is declared in this same file: both ids
		// are known now, so this is an IntraEdge, not an UnresolvedRef.
		ex.result.IntraEdges = append(ex.result.IntraEdges, IntraEdge{Edge: &schema.Edge{
			Source: ex.typeNodesByName[recvType], Target: id, Kind: "contains", Provenance: "ast",
		}})
	case hasRecv:
		// The receiver's type lives in a different file. Pass 2 resolves
		// this once it has a global, cross-file symbol index.
		pos := decl.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: id, Name: recvType, Kind: RefKindContains,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	default:
		// No resolvable receiver type at all (malformed/unusual source);
		// still attach the method to the file so it is not orphaned.
		ex.result.IntraEdges = append(ex.result.IntraEdges, IntraEdge{Edge: &schema.Edge{
			Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
		}})
	}

	ex.collectReturnTypeRef(id, decl)

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
		ex.collectReferencesAndInstantiates(id, body)
	}
}

// receiverTypeName extracts a method_declaration's receiver type name,
// unwrapping a pointer_type first so both value and pointer receivers
// resolve to the same underlying type (RESEARCH §Method Receiver
// Extraction).
func receiverTypeName(method *tree_sitter.Node, src []byte) (string, bool) {
	receiver := method.ChildByFieldName("receiver")
	if receiver == nil {
		return "", false
	}
	var paramDecl *tree_sitter.Node
	for i := uint(0); i < receiver.NamedChildCount(); i++ {
		if c := receiver.NamedChild(i); c.Kind() == "parameter_declaration" {
			paramDecl = c
			break
		}
	}
	if paramDecl == nil {
		return "", false
	}
	n := paramDecl.NamedChildCount()
	if n == 0 {
		return "", false
	}
	// The type is always the LAST named child: the receiver name is
	// optional and, when present, comes first.
	typeNode := paramDecl.NamedChild(n - 1)
	if typeNode.Kind() == "pointer_type" {
		if typeNode.NamedChildCount() == 0 {
			return "", false
		}
		typeNode = typeNode.NamedChild(0)
	}
	if typeNode.Kind() != "type_identifier" {
		return "", false
	}
	return typeNode.Utf8Text(src), true
}

func (ex *extractor) buildSymbolNode(kind, id, name, qualifiedName string, decl *tree_sitter.Node) *schema.Node {
	start := decl.StartPosition()
	end := decl.EndPosition()

	var signature, returnType string
	if params := decl.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	}
	if result := decl.ChildByFieldName("result"); result != nil {
		returnType = result.Utf8Text(ex.src)
		if signature != "" {
			signature += " " + returnType
		} else {
			signature = returnType
		}
	}

	return &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "go",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Signature:     signature,
		Visibility:    visibilityFor(name),
		IsExported:    isExported(name),
		ReturnType:    returnType,
	}
}

// collectCalls walks a function/method body with an explicit stack (T-02-04
// depth guard — no unbounded Go-stack recursion over a pathologically deep
// AST) looking for call_expression nodes.
func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "call_expression" {
			ex.recordCall(fromID, n)
		}
		return true
	})
}

func (ex *extractor) recordCall(fromID string, call *tree_sitter.Node) {
	fn := call.ChildByFieldName("function")
	if fn == nil {
		return
	}
	for fn.Kind() == "parenthesized_expression" {
		if fn.NamedChildCount() == 0 {
			return
		}
		fn = fn.NamedChild(0)
	}

	pos := call.StartPosition()
	line := int32(pos.Row) + 1
	col := int32(pos.Column)

	switch fn.Kind() {
	case "identifier":
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: fromID, Name: fn.Utf8Text(ex.src), Kind: RefKindCalls, Line: line, Col: col,
		})
	case "selector_expression":
		operand := fn.ChildByFieldName("operand")
		field := fn.ChildByFieldName("field")
		if operand == nil {
			operand = fn.NamedChild(0)
		}
		if field == nil && fn.NamedChildCount() > 1 {
			field = fn.NamedChild(fn.NamedChildCount() - 1)
		}
		if field == nil {
			return
		}
		var pkgAlias string
		switch {
		case operand == nil:
			// No operand at all (malformed/unusual source) — fall through
			// to the pre-existing empty-PkgAlias (unqualified) shape;
			// unchanged from before this fix.
		case operand.Kind() == "identifier":
			pkgAlias = operand.Utf8Text(ex.src)
		default:
			// WR-02: a non-identifier operand (call_expression,
			// index_expression, etc. — e.g. `foo().Bar()` or
			// `arr[i].Bar()`) is never a package alias. Leaving PkgAlias
			// empty here would make resolveNameRef treat this as an
			// UNQUALIFIED same-package reference, which could wrongly
			// match an unrelated same-package symbol that happens to
			// share the bare field name (the exact WR-02 mis-resolution
			// bug). Force this ref through resolveSelector's
			// narrowest-safe-set alias-membership boundary instead, using
			// a synthetic alias that can never equal a real import alias
			// (a valid Go package_identifier never contains "<"/">"), so
			// it deterministically ends up "unresolved" — matching the
			// local-variable-receiver case's own fall-through behavior.
			pkgAlias = "<" + operand.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: RefKindCalls, Line: line, Col: col,
		})
	}
}

// collectReturnTypeRef emits a D-09 `returns` Pass-1 ref (01-RESEARCH.md
// §B) from a function/method's own id to its declared return type's
// name, reusing the SAME "result" field buildSymbolNode already reads
// for Node.ReturnType — this does not re-parse anything, it just
// isolates the type NAME for node resolution via namedTypeRef. Only a
// SINGLE named return type (type_identifier/qualified_type, optionally
// pointer-wrapped) is captured; a primitive return (int, string, ...), a
// multi-value return (a parameter_list result), or no declared result at
// all emits no ref — absence, not error (RESEARCH §B's documented
// precision note: generics/unions/multi-returns resolve to nothing
// rather than guessing at an "outer type").
func (ex *extractor) collectReturnTypeRef(fromID string, decl *tree_sitter.Node) {
	result := decl.ChildByFieldName("result")
	if result == nil {
		return
	}
	name, pkgAlias, ok := namedTypeRef(result, ex.src)
	if !ok {
		return
	}
	pos := result.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: RefKindReturns,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// recordInstantiate emits a D-09 `instantiates` Pass-1 ref (01-RESEARCH.md
// §B) for a composite_literal (`T{}`; `&T{}` is a unary_expression
// wrapping the SAME composite_literal shape, so no separate handling is
// needed) whose "type" field is a single named type reference. A nested
// literal with no explicit type (e.g. the inner `{...}` elements of
// `[]T{{...}, {...}}`, which inherit T from the outer slice type) and a
// literal typed by a compound/unnamed shape (slice_type, map_type,
// array_type, generic_type) both emit no ref — absence, not error; the
// Kind-check disambiguation that the resolved target must actually be a
// type-Kind node (not, say, a package) happens at Pass 2/resolve.go, not
// here.
func (ex *extractor) recordInstantiate(fromID string, lit *tree_sitter.Node) {
	t := lit.ChildByFieldName("type")
	if t == nil {
		return
	}
	name, pkgAlias, ok := namedTypeRef(t, ex.src)
	if !ok {
		return
	}
	pos := lit.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: RefKindInstantiates,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// collectReferencesAndInstantiates walks a function/method body for two
// new D-09 Pass-1 capture kinds (01-RESEARCH.md §B): instantiates (via
// recordInstantiate, above) and references (a value read of an
// identifier or package-qualified selector that is NOT a call's own
// callee — RefKindCalls already captures those via collectCalls'
// independent whole-body scan, and de-dup requires a called symbol never
// ALSO emit a references ref).
//
// Go D-02 precision note (RESEARCH §B): this walk is scoped to a bounded
// allow-list of unambiguous read positions — call arguments, return
// values, assignment/short-var-declaration right-hand sides, and
// composite-literal element values, plus the common compound-expression
// wrappers reachable from them via captureExprRead (parenthesized/
// binary/unary/index expressions, nested composite literals, nested
// calls) — rather than exhaustively covering every Go expression-
// statement shape (e.g. a bare expression_statement, a condition
// expression, a go/defer/select argument are not captured). This is a
// deliberate, bounded scope, not a silent drop of ground truth: TS's own
// "references" semantic is already a broad, heuristic "identifier use"
// signal, and Go's syntax makes exhaustive coverage categorically harder
// to bound safely than calls/embeds — an over-broad walk risks the exact
// false same-package-name-collision resolution the codebase's existing
// WR-02 discipline (recordCall's selector handling, mirrored below in
// captureSelectorRead) exists to avoid.
func (ex *extractor) collectReferencesAndInstantiates(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "call_expression":
			// Walk only the arguments — never the callee subtree (de-dup:
			// RefKindCalls already captures the callee via collectCalls'
			// separate walk, and a called symbol must never ALSO emit a
			// references ref).
			if args := n.ChildByFieldName("arguments"); args != nil {
				for i := uint(0); i < args.NamedChildCount(); i++ {
					ex.captureExprRead(fromID, args.NamedChild(i))
				}
			}
			return false
		case "composite_literal":
			ex.captureExprRead(fromID, n) // handles recordInstantiate + element values
			return false
		case "return_statement":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				ex.captureExprRead(fromID, n.NamedChild(i))
			}
			return false
		case "short_var_declaration":
			// The "left" side is a declaration (new local names), never a
			// reference — only "right" (the initializer values) is walked.
			if right := n.ChildByFieldName("right"); right != nil {
				for i := uint(0); i < right.NamedChildCount(); i++ {
					ex.captureExprRead(fromID, right.NamedChild(i))
				}
			}
			return false
		case "assignment_statement":
			// The "left" side is a write target, never a reference — only
			// "right" (the assigned values) is walked.
			if right := n.ChildByFieldName("right"); right != nil {
				for i := uint(0); i < right.NamedChildCount(); i++ {
					ex.captureExprRead(fromID, right.NamedChild(i))
				}
			}
			return false
		}
		return true
	})
}

// captureExprRead classifies a single expression node reached from an
// allow-listed read position (see collectReferencesAndInstantiates) and
// emits a references UnresolvedRef for a bare identifier or
// package-qualified selector value-read, recursing through the common
// compound-expression wrappers (parenthesized/binary/unary/index/
// composite-literal/nested-call) so a nested read inside one of those is
// still found. This recursion is bounded by expression nesting depth (not
// overall AST size), consistent with typeRefName/receiverTypeName's
// existing single-hop ChildByFieldName access pattern elsewhere in this
// file — the whole-tree traversal itself stays on walkDescendants'
// explicit stack (T-02-04).
func (ex *extractor) captureExprRead(fromID string, expr *tree_sitter.Node) {
	if expr == nil {
		return
	}
	switch expr.Kind() {
	case "identifier":
		pos := expr.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: fromID, Name: expr.Utf8Text(ex.src), Kind: RefKindReferences,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	case "selector_expression":
		ex.captureSelectorRead(fromID, expr)
	case "parenthesized_expression":
		if expr.NamedChildCount() > 0 {
			ex.captureExprRead(fromID, expr.NamedChild(0))
		}
	case "unary_expression":
		if operand := expr.ChildByFieldName("operand"); operand != nil {
			ex.captureExprRead(fromID, operand)
		}
	case "binary_expression":
		if left := expr.ChildByFieldName("left"); left != nil {
			ex.captureExprRead(fromID, left)
		}
		if right := expr.ChildByFieldName("right"); right != nil {
			ex.captureExprRead(fromID, right)
		}
	case "index_expression":
		if operand := expr.ChildByFieldName("operand"); operand != nil {
			ex.captureExprRead(fromID, operand)
		}
		if index := expr.ChildByFieldName("index"); index != nil {
			ex.captureExprRead(fromID, index)
		}
	case "call_expression":
		// A nested call (e.g. an argument that is itself a call) — its
		// own callee is never a reference (de-dup, mirroring the outer
		// collectReferencesAndInstantiates rule); collectCalls' separate
		// whole-body walk already records the call itself regardless of
		// nesting position, so only its arguments are walked here.
		if args := expr.ChildByFieldName("arguments"); args != nil {
			for i := uint(0); i < args.NamedChildCount(); i++ {
				ex.captureExprRead(fromID, args.NamedChild(i))
			}
		}
	case "composite_literal":
		ex.recordInstantiate(fromID, expr)
		if elemBody := expr.ChildByFieldName("body"); elemBody != nil {
			for i := uint(0); i < elemBody.NamedChildCount(); i++ {
				ex.captureCompositeElement(fromID, elemBody.NamedChild(i))
			}
		}
	}
	// Every other expression kind (literals, function_literal bodies,
	// type_assertion_expression, etc.) is out of this bounded allow-list
	// per the Go D-02 precision note above — no reference captured, no
	// error.
}

// captureCompositeElement handles one element of a composite_literal's
// element_list: a bare value, or a keyed_element{key, value} pair — only
// the VALUE side is a read (a keyed_element's key, when it names a
// struct field, is a declaration-adjacent label, not a value reference).
func (ex *extractor) captureCompositeElement(fromID string, elem *tree_sitter.Node) {
	if elem.Kind() == "keyed_element" {
		if v := elem.ChildByFieldName("value"); v != nil {
			ex.captureExprRead(fromID, v)
		} else if elem.NamedChildCount() > 1 {
			ex.captureExprRead(fromID, elem.NamedChild(1))
		}
		return
	}
	ex.captureExprRead(fromID, elem)
}

// captureSelectorRead handles a selector_expression VALUE read
// (pkg.Symbol used as a value, not called — recordCall's own selector
// handling already captures the call-callee shape via collectCalls'
// separate walk). Mirrors recordCall's WR-02 alias-safety discipline
// exactly: a non-identifier operand can never be a package alias, so it
// is forced through a synthetic alias that never matches a real import,
// keeping resolution deterministically unresolved rather than risking a
// same-package name collision — and the operand itself is still walked
// for further nested reads.
func (ex *extractor) captureSelectorRead(fromID string, sel *tree_sitter.Node) {
	operand := sel.ChildByFieldName("operand")
	field := sel.ChildByFieldName("field")
	if operand == nil {
		operand = sel.NamedChild(0)
	}
	if field == nil && sel.NamedChildCount() > 1 {
		field = sel.NamedChild(sel.NamedChildCount() - 1)
	}
	if field == nil {
		return
	}
	var pkgAlias string
	switch {
	case operand == nil:
		// No operand at all (malformed/unusual source) — fall through to
		// the empty-PkgAlias (unqualified) shape.
	case operand.Kind() == "identifier":
		pkgAlias = operand.Utf8Text(ex.src)
	default:
		pkgAlias = "<" + operand.Kind() + ">"
		ex.captureExprRead(fromID, operand)
	}
	pos := sel.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
		FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: RefKindReferences,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants. visit returns false to skip a node's
// children (not used today, but kept for callers that need to prune a
// subtree). Using an explicit stack rather than Go recursion means a
// pathologically deep AST cannot exhaust the Go stack (T-02-04) — the
// tree is already size-bounded by parser.MaxSourceBytes, but the walk
// itself makes no additional assumption on top of that ceiling.
func walkDescendants(root *tree_sitter.Node, visit func(*tree_sitter.Node) bool) {
	stack := make([]*tree_sitter.Node, 0, 64)
	stack = append(stack, root)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !visit(n) {
			continue
		}
		count := n.ChildCount()
		for i := int(count) - 1; i >= 0; i-- {
			if c := n.Child(uint(i)); c != nil {
				stack = append(stack, c)
			}
		}
	}
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func visibilityFor(name string) string {
	if isExported(name) {
		return "public"
	}
	return "package"
}
