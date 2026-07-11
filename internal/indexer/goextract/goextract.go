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
		ImportPath:  importPath,
		RelPath:     relPath,
		Language:    "go",
		ContentHash: hex.EncodeToString(sum[:]),
		Imports:     make(map[string]string),
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
	}
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

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
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

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
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
		if operand != nil && operand.Kind() == "identifier" {
			pkgAlias = operand.Utf8Text(ex.src)
		}
		ex.result.Unresolved = append(ex.result.Unresolved, UnresolvedRef{
			FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: RefKindCalls, Line: line, Col: col,
		})
	}
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
