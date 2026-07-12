package cextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Extract walks a single C or C++ file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-cpp's
// external C scanner (tree-sitter-c carries none).
//
// moduleKey is the discovery-time path-identity placeholder
// (languages_c.go/languages_cpp.go's LanguageSpec.ModuleKey) — C/C++ have no
// in-source module declaration, so Extract never overrides
// FileResult.ImportPath (see types.go).
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	language := languageForExt(relPath)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    language,
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
		result.Err = fmt.Errorf("cextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("cextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      language,
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	decls := topLevelDecls(root)

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		language:        language,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Types are collected first so an out-of-line method's qualifier lookup
	// and a base-class-clause's embeds ref always see every struct/class
	// this file declares, regardless of declaration order (mirrors
	// goextract/rustextract's ordering discipline).
	ex.collectIncludes(decls)
	ex.collectTypes(decls)
	ex.collectTypedefs(decls)
	ex.collectFunctions(decls)

	return result, nil
}

// languageForExt determines "c" or "cpp" from relPath's own extension —
// cextract.Extract has no other signal available (LanguageSpec.Extract's
// shared cross-language signature carries no language field). ".h" always
// resolves to "c" (the documented default disposition, types.go); this
// default also covers any extension not claimed by either
// languages_c.go/languages_cpp.go, which never happens in practice since
// Extract is only ever invoked for a registered extension.
func languageForExt(relPath string) string {
	lower := strings.ToLower(relPath)
	switch {
	case strings.HasSuffix(lower, ".cpp"), strings.HasSuffix(lower, ".cc"),
		strings.HasSuffix(lower, ".cxx"), strings.HasSuffix(lower, ".hpp"),
		strings.HasSuffix(lower, ".hh"):
		return "cpp"
	default:
		return "c"
	}
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract/rustextract's extractor shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	language        string
	result          *goextract.FileResult
	typeNodesByName map[string]string
}

// topLevelDecls returns the effective list of "top-level" declaration nodes
// to walk: root's own direct named children, with a C++ namespace_definition
// TRANSPARENTLY expanded (recursively, so nested namespaces flatten fully)
// in its place — see types.go for why no namespace node is ever emitted.
func topLevelDecls(root *tree_sitter.Node) []*tree_sitter.Node {
	var decls []*tree_sitter.Node
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		for i := uint(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Kind() == "namespace_definition" {
				if body := c.ChildByFieldName("body"); body != nil {
					walk(body)
				}
				continue
			}
			decls = append(decls, c)
		}
	}
	walk(root)
	return decls
}

// --- #include ---

func (ex *extractor) collectIncludes(decls []*tree_sitter.Node) {
	for _, d := range decls {
		if d.Kind() != "preproc_include" {
			continue
		}
		ex.emitInclude(d)
	}
}

func (ex *extractor) emitInclude(decl *tree_sitter.Node) {
	pathNode := decl.ChildByFieldName("path")
	if pathNode == nil {
		return
	}
	trimmed := strings.Trim(pathNode.Utf8Text(ex.src), `"<>`)
	if trimmed == "" {
		return
	}
	pos := decl.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: trimmed, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// --- struct_specifier / class_specifier ---

func (ex *extractor) collectTypes(decls []*tree_sitter.Node) {
	for _, d := range decls {
		switch d.Kind() {
		case "struct_specifier":
			ex.emitType(d, false)
		case "class_specifier":
			ex.emitType(d, true)
		}
	}
}

func (ex *extractor) emitType(decl *tree_sitter.Node, isClass bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "type_identifier" {
		// An anonymous struct/class, or one whose name is a template_type/
		// qualified_identifier — documented gap (types.go).
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(goextract.KindStruct, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindStruct,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    "public",
		IsExported:    true,
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.typeNodesByName[name] = id

	if !isClass {
		return
	}
	ex.collectSupertypes(id, decl)
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectInlineMethods(id, name, body)
	}
}

func (ex *extractor) collectSupertypes(id string, decl *tree_sitter.Node) {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() != "base_class_clause" {
			continue
		}
		for j := uint(0); j < c.NamedChildCount(); j++ {
			n := c.NamedChild(j)
			if n.Kind() != "type_identifier" {
				// A qualified_identifier base (`ns::Base`) — documented gap.
				continue
			}
			pos := n.StartPosition()
			ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
				FromID: id, Name: n.Utf8Text(ex.src), Kind: goextract.RefKindEmbeds,
				Line: int32(pos.Row) + 1, Col: int32(pos.Column),
			})
		}
	}
}

// collectInlineMethods extracts only DIRECT function_definition children of
// a class_specifier's own field_declaration_list body — inline methods with
// a body written in the class. A field_declaration child (a member
// variable, or any bodyless/pure-virtual method prototype) is never
// extracted (types.go).
func (ex *extractor) collectInlineMethods(typeID, typeName string, body *tree_sitter.Node) {
	for i := uint(0); i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		if m.Kind() != "function_definition" {
			continue
		}
		declarator := m.ChildByFieldName("declarator")
		if declarator == nil {
			continue
		}
		fd := findFunctionDeclarator(declarator)
		if fd == nil {
			continue
		}
		nameDeclarator := fd.ChildByFieldName("declarator")
		if nameDeclarator == nil {
			continue
		}
		var name string
		switch nameDeclarator.Kind() {
		case "field_identifier", "identifier", "destructor_name":
			name = nameDeclarator.Utf8Text(ex.src)
		default:
			// operator_name, or any other shape — documented gap.
			continue
		}
		qualifiedName := typeName + "." + name
		id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, m)})
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		if mbody := m.ChildByFieldName("body"); mbody != nil {
			ex.collectCalls(id, mbody)
		}
	}
}

// --- typedef ---

func (ex *extractor) collectTypedefs(decls []*tree_sitter.Node) {
	for _, d := range decls {
		if d.Kind() != "type_definition" {
			continue
		}
		ex.emitTypedef(d)
	}
}

func (ex *extractor) emitTypedef(decl *tree_sitter.Node) {
	declarator := decl.ChildByFieldName("declarator")
	if declarator == nil {
		return
	}
	name, ok := cDeclaratorName(declarator, ex.src)
	if !ok {
		return
	}
	id := nodeid.NodeID(goextract.KindTypeAlias, name, ex.relPath)
	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindTypeAlias,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    "public",
		IsExported:    true,
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
}

// --- top-level functions, prototypes, and C++ out-of-line methods ---

func (ex *extractor) collectFunctions(decls []*tree_sitter.Node) {
	for _, d := range decls {
		switch d.Kind() {
		case "function_definition", "declaration":
			ex.emitFreeFunctionLike(d)
		}
	}
}

// emitFreeFunctionLike handles a root-level (or namespace-flattened)
// function_definition OR declaration node: a plain identifier declarator is
// a free function (or, for a bodyless "declaration", a prototype); a
// qualified_identifier declarator is a C++ out-of-line method definition.
func (ex *extractor) emitFreeFunctionLike(decl *tree_sitter.Node) {
	declarator := decl.ChildByFieldName("declarator")
	if declarator == nil {
		return
	}
	fd := findFunctionDeclarator(declarator)
	if fd == nil {
		// Not a function shape (a plain variable declaration/typedef-like
		// declaration) — nothing to extract.
		return
	}
	nameDeclarator := fd.ChildByFieldName("declarator")
	if nameDeclarator == nil {
		return
	}
	switch nameDeclarator.Kind() {
	case "identifier":
		ex.emitFunction(decl, nameDeclarator.Utf8Text(ex.src))
	case "qualified_identifier":
		if scope, name, ok := splitQualifiedMethod(nameDeclarator, ex.src); ok {
			ex.emitOutOfLineMethod(decl, scope, name)
		}
	default:
		// field_identifier/destructor_name/operator_name shouldn't normally
		// appear at top level — documented gap if they do.
	}
}

func (ex *extractor) emitFunction(decl *tree_sitter.Node, name string) {
	id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

func (ex *extractor) emitOutOfLineMethod(decl *tree_sitter.Node, scope, name string) {
	qualifiedName := scope + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl)})

	if typeID, found := ex.typeNodesByName[scope]; found {
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
		}})
	} else {
		// The qualifying type is declared in a different file (typically
		// the paired header) — Pass 2 resolves this once it has a global,
		// cross-file symbol index (mirrors rustextract's identical
		// cross-file impl-block handling).
		pos := decl.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: id, Name: scope, Kind: goextract.RefKindContains,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

func (ex *extractor) buildFuncNode(kind, id, name, qualifiedName string, decl *tree_sitter.Node) *schema.Node {
	start := decl.StartPosition()
	end := decl.EndPosition()
	return &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    "public",
		IsExported:    true,
	}
}

// splitQualifiedMethod extracts (scope, name) from a C++ qualified_identifier
// declarator (`Type::method`), the shape a C++ out-of-line method
// definition's declarator takes. scope's node kind is "namespace_identifier"
// regardless of whether it's semantically a namespace or a class name (a
// genuine C++ grammar-level ambiguity at this node shape, not an oversight —
// see types.go).
func splitQualifiedMethod(n *tree_sitter.Node, src []byte) (scope, name string, ok bool) {
	scopeNode := n.ChildByFieldName("scope")
	nameNode := n.ChildByFieldName("name")
	if scopeNode == nil || nameNode == nil {
		return "", "", false
	}
	switch scopeNode.Kind() {
	case "namespace_identifier", "type_identifier":
		scope = scopeNode.Utf8Text(src)
	default:
		return "", "", false
	}
	switch nameNode.Kind() {
	case "identifier", "field_identifier", "destructor_name":
		name = nameNode.Utf8Text(src)
	default:
		return "", "", false
	}
	return scope, name, true
}

// --- declarator name resolution ---

// findFunctionDeclarator walks n (a _declarator/_field_declarator/
// _type_declarator node, as bound to a function_definition/declaration's own
// "declarator" field) through pointer/array/reference/parenthesized wrapping
// to find the innermost function_declarator node — needed both for its own
// "declarator" (the function's name-declarator) and "parameters" fields.
func findFunctionDeclarator(n *tree_sitter.Node) *tree_sitter.Node {
	for n != nil {
		if n.Kind() == "function_declarator" {
			return n
		}
		switch n.Kind() {
		case "pointer_declarator", "array_declarator", "reference_declarator":
			n = n.ChildByFieldName("declarator")
		case "parenthesized_declarator":
			n = firstNamedChild(n)
		default:
			return nil
		}
	}
	return nil
}

// cDeclaratorName walks n through the same wrapping shapes as
// findFunctionDeclarator, but resolves to a plain identifier/type name
// (used for typedef declarators, which never wrap a function_declarator).
func cDeclaratorName(n *tree_sitter.Node, src []byte) (string, bool) {
	for n != nil {
		switch n.Kind() {
		case "identifier", "field_identifier", "type_identifier":
			return n.Utf8Text(src), true
		case "pointer_declarator", "array_declarator", "reference_declarator":
			n = n.ChildByFieldName("declarator")
		case "parenthesized_declarator":
			n = firstNamedChild(n)
		case "init_declarator":
			n = n.ChildByFieldName("declarator")
		default:
			return "", false
		}
	}
	return "", false
}

func firstNamedChild(n *tree_sitter.Node) *tree_sitter.Node {
	if n.NamedChildCount() == 0 {
		return nil
	}
	return n.NamedChild(0)
}

// --- calls ---

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
	pos := call.StartPosition()
	line, col := int32(pos.Row)+1, int32(pos.Column)

	switch fn.Kind() {
	case "identifier":
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: fn.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "field_expression":
		field := fn.ChildByFieldName("field")
		arg := fn.ChildByFieldName("argument")
		if field == nil {
			return
		}
		var pkgAlias string
		switch {
		case arg == nil, arg.Kind() == "this":
			// this->method() -- implicit same-class call, empty PkgAlias.
		case arg.Kind() == "identifier" && isLikelyTypeName(arg.Utf8Text(ex.src)):
			// A rare PascalCase bare-name receiver -- same-module attempt.
		case arg.Kind() == "identifier":
			// A lowercase-leading identifier is very likely a local
			// variable/parameter this extractor tracks no type for -- force
			// it through the WR-02 synthetic-non-matching-alias pattern.
			pkgAlias = "<local:" + arg.Utf8Text(ex.src) + ">"
		default:
			pkgAlias = "<" + arg.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "qualified_identifier":
		nameNode := fn.ChildByFieldName("name")
		if nameNode == nil {
			return
		}
		var name string
		switch nameNode.Kind() {
		case "identifier", "field_identifier":
			name = nameNode.Utf8Text(ex.src)
		default:
			return
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter -- C++'s own near-universal naming convention distinguishing a
// type/class name (PascalCase) from a local variable/parameter/function
// name (snake_case/camelCase), mirroring pyextract/rustextract/phpextract's
// identical heuristic.
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract/rustextract's own
// walkDescendants (T-02-04 depth guard).
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
