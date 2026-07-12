package phpextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Extract walks a single PHP file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-php's
// external tag-switching (`<?php ?>`) C scanner.
//
// moduleKey is the discovery-time path-based placeholder (languages_php.go's
// LanguageSpec.ModuleKey — path identity, or a composer.json PSR-4 best
// effort). A file's own declared `namespace Foo\Bar;`/`namespace Foo\Bar {
// ... }` statement, when present, OVERRIDES it — mirroring csharpextract's
// identical parse-time-override pattern (C#'s `namespace` declaration is
// likewise only knowable after parsing).
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "php",
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
		result.Err = fmt.Errorf("phpextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("phpextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "php",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	if ns, ok := namespaceOverride(root, src); ok && ns != "" {
		result.ImportPath = ns
	}

	decls := topLevelDecls(root)

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Imports are collected first so supertype/call qualifier checks below
	// can tell an imported simple name apart from a same-module one,
	// mirroring javaextract/pyextract's ordering discipline.
	ex.collectUses(decls)
	ex.collectTypes(decls)
	ex.collectMethods(decls)
	ex.collectFunctions(decls)

	return result, nil
}

// namespaceOverride finds root's first top-level namespace_definition (both
// the unbraced `namespace Foo\Bar;` and braced `namespace Foo\Bar { ... }`
// forms share the same "name" field shape) and returns its fully-qualified
// namespace_name text verbatim (PHP's own "\"-separated convention).
// Returns ok=false if no namespace_definition exists, or the rare global
// (`namespace;`, no name) form is used.
func namespaceOverride(root *tree_sitter.Node, src []byte) (string, bool) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		if c.Kind() != "namespace_definition" {
			continue
		}
		if nameNode := c.ChildByFieldName("name"); nameNode != nil {
			return nameNode.Utf8Text(src), true
		}
		return "", false
	}
	return "", false
}

// topLevelDecls returns the effective list of "top-level" declaration nodes
// to walk: root's own direct named children, EXCEPT a braced
// `namespace Foo { ... }` block's own body children are substituted in its
// place (PHP's unbraced `namespace Foo;` form already leaves its
// declarations as root's own direct siblings — see the live parse-tree
// verification in this package's development notes — so only the braced
// form needs this one-level expansion).
func topLevelDecls(root *tree_sitter.Node) []*tree_sitter.Node {
	var decls []*tree_sitter.Node
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		if c.Kind() == "namespace_definition" {
			if body := c.ChildByFieldName("body"); body != nil {
				for j := uint(0); j < body.NamedChildCount(); j++ {
					decls = append(decls, body.NamedChild(j))
				}
			}
			continue
		}
		decls = append(decls, c)
	}
	return decls
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract/pyextract's extractor shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *goextract.FileResult
	typeNodesByName map[string]string
}

// --- namespace_use_declaration ---

func (ex *extractor) collectUses(decls []*tree_sitter.Node) {
	for _, decl := range decls {
		if decl.Kind() == "namespace_use_declaration" {
			ex.emitUseDeclaration(decl)
		}
	}
}

func (ex *extractor) emitUseDeclaration(decl *tree_sitter.Node) {
	var prefix string
	var group *tree_sitter.Node
	var clauses []*tree_sitter.Node
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		switch c.Kind() {
		case "namespace_name":
			prefix = c.Utf8Text(ex.src)
		case "namespace_use_group":
			group = c
		case "namespace_use_clause":
			clauses = append(clauses, c)
		}
	}
	if group != nil {
		for i := uint(0); i < group.NamedChildCount(); i++ {
			if c := group.NamedChild(i); c.Kind() == "namespace_use_clause" {
				ex.emitUseClause(prefix, c)
			}
		}
		return
	}
	for _, c := range clauses {
		ex.emitUseClause("", c)
	}
}

func (ex *extractor) emitUseClause(prefix string, clause *tree_sitter.Node) {
	var nameNode *tree_sitter.Node
	for i := uint(0); i < clause.NamedChildCount(); i++ {
		c := clause.NamedChild(i)
		if c.Kind() == "name" || c.Kind() == "qualified_name" {
			nameNode = c
			break
		}
	}
	if nameNode == nil {
		return
	}
	simple, full := phpSimpleAndFullName(nameNode, ex.src)
	if simple == "" {
		return
	}
	fullPath := full
	if prefix != "" {
		fullPath = prefix + "\\" + full
	}
	alias := simple
	if a := clause.ChildByFieldName("alias"); a != nil {
		alias = a.Utf8Text(ex.src)
	}
	ex.result.Imports[alias] = fullPath

	pos := clause.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: fullPath, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// phpSimpleAndFullName extracts (simpleName, fullDottedPath) from a "name"
// leaf or a "qualified_name"/"relative_name" wrapper — the latter two
// always end with a trailing "name" child holding the final simple segment.
func phpSimpleAndFullName(n *tree_sitter.Node, src []byte) (simple, full string) {
	switch n.Kind() {
	case "name":
		t := n.Utf8Text(src)
		return t, t
	case "qualified_name", "relative_name":
		full = n.Utf8Text(src)
		if n.NamedChildCount() == 0 {
			return "", full
		}
		last := n.NamedChild(n.NamedChildCount() - 1)
		return last.Utf8Text(src), full
	default:
		return "", ""
	}
}

// --- class / interface / trait declarations ---

func (ex *extractor) collectTypes(decls []*tree_sitter.Node) {
	for _, decl := range decls {
		switch decl.Kind() {
		case "class_declaration":
			ex.emitTypeDecl(goextract.KindStruct, decl)
		case "interface_declaration":
			ex.emitTypeDecl(goextract.KindInterface, decl)
		case "trait_declaration":
			ex.emitTypeDecl(goextract.KindStruct, decl)
		}
	}
}

func (ex *extractor) emitTypeDecl(kind string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(kind, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "php",
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

	ex.collectSupertypes(id, decl)
}

func (ex *extractor) collectSupertypes(id string, decl *tree_sitter.Node) {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() == "base_clause" || c.Kind() == "class_interface_clause" {
			ex.emitSupertypeClause(id, c)
		}
	}
}

func (ex *extractor) emitSupertypeClause(id string, clause *tree_sitter.Node) {
	for i := uint(0); i < clause.NamedChildCount(); i++ {
		n := clause.NamedChild(i)
		switch n.Kind() {
		case "name", "qualified_name", "relative_name":
			ex.emitSupertypeRef(id, n)
		}
	}
}

func (ex *extractor) emitSupertypeRef(id string, n *tree_sitter.Node) {
	simple, _ := phpSimpleAndFullName(n, ex.src)
	if simple == "" {
		return
	}
	pkgAlias := ""
	if _, imported := ex.result.Imports[simple]; imported {
		pkgAlias = simple
	}
	pos := n.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: id, Name: simple, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// --- methods (inside class/interface/trait bodies) ---

func (ex *extractor) collectMethods(decls []*tree_sitter.Node) {
	for _, decl := range decls {
		switch decl.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration":
			ex.emitMethodsForType(decl)
		}
	}
}

func (ex *extractor) emitMethodsForType(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	typeName := nameNode.Utf8Text(ex.src)
	typeID, ok := ex.typeNodesByName[typeName]
	if !ok {
		return
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		if m := body.NamedChild(i); m.Kind() == "method_declaration" {
			ex.emitMethod(typeID, typeName, m)
		}
	}
}

func (ex *extractor) emitMethod(typeID, typeName string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := typeName + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
	}})

	// An interface method has no body (abstract) -- ChildByFieldName
	// returns nil, collectCalls is simply skipped.
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

// --- top-level free functions ---

func (ex *extractor) collectFunctions(decls []*tree_sitter.Node) {
	for _, decl := range decls {
		if decl.Kind() == "function_definition" {
			ex.emitFunction(decl)
		}
	}
}

func (ex *extractor) emitFunction(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

func (ex *extractor) buildFuncNode(kind, id, name, qualifiedName string, decl *tree_sitter.Node) *schema.Node {
	start := decl.StartPosition()
	end := decl.EndPosition()

	var signature, returnType string
	if params := decl.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	}
	if rt := decl.ChildByFieldName("return_type"); rt != nil {
		returnType = rt.Utf8Text(ex.src)
	}

	return &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "php",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Signature:     signature,
		ReturnType:    returnType,
		Visibility:    "public",
		IsExported:    true,
	}
}

// --- calls ---

func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "function_call_expression":
			ex.recordFunctionCall(fromID, n)
		case "member_call_expression", "nullsafe_member_call_expression":
			ex.recordMemberCall(fromID, n)
		case "scoped_call_expression":
			ex.recordScopedCall(fromID, n)
		}
		return true
	})
}

func (ex *extractor) recordFunctionCall(fromID string, call *tree_sitter.Node) {
	fn := call.ChildByFieldName("function")
	if fn == nil {
		return
	}
	pos := call.StartPosition()
	line, col := int32(pos.Row)+1, int32(pos.Column)

	switch fn.Kind() {
	case "name":
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: fn.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "qualified_name", "relative_name":
		simple, _ := phpSimpleAndFullName(fn, ex.src)
		if simple == "" {
			return
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: simple, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	default:
		// A dynamic callee (a variable holding a callable, an expression
		// result) is not resolvable -- documented gap.
	}
}

func (ex *extractor) recordMemberCall(fromID string, call *tree_sitter.Node) {
	nameNode := call.ChildByFieldName("name")
	object := call.ChildByFieldName("object")
	if nameNode == nil || nameNode.Kind() != "name" || object == nil {
		return
	}
	pos := call.StartPosition()
	line, col := int32(pos.Row)+1, int32(pos.Column)

	var pkgAlias string
	switch {
	case object.Kind() == "variable_name" && phpVariableName(object, ex.src) == "this":
		// $this->method() -- implicit same-class call.
	case object.Kind() == "variable_name":
		pkgAlias = "<local:" + phpVariableName(object, ex.src) + ">"
	case object.Kind() == "name" && isLikelyTypeName(object.Utf8Text(ex.src)):
		// A rare PascalCase bare-name receiver -- same-module attempt.
	default:
		pkgAlias = "<" + object.Kind() + ">"
	}
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: nameNode.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
	})
}

func (ex *extractor) recordScopedCall(fromID string, call *tree_sitter.Node) {
	nameNode := call.ChildByFieldName("name")
	scope := call.ChildByFieldName("scope")
	if nameNode == nil || nameNode.Kind() != "name" || scope == nil {
		return
	}
	pos := call.StartPosition()
	line, col := int32(pos.Row)+1, int32(pos.Column)

	var pkgAlias string
	switch scope.Kind() {
	case "name":
		scopeText := scope.Utf8Text(ex.src)
		if _, imported := ex.result.Imports[scopeText]; imported {
			pkgAlias = scopeText
		}
		// else: self/static/parent, or a same-module Type::method() attempt
		// (the dominant PHP static-call idiom; PHP class names are
		// conventionally PascalCase) -- empty PkgAlias either way, mirroring
		// pyextract/javaextract's naming-convention heuristic.
	case "variable_name":
		pkgAlias = "<local:" + phpVariableName(scope, ex.src) + ">"
	default:
		pkgAlias = "<" + scope.Kind() + ">"
	}
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: nameNode.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
	})
}

// phpVariableName extracts a variable_name node's own identifier text
// (without its leading "$") via its "name" named child.
func phpVariableName(n *tree_sitter.Node, src []byte) string {
	if n.NamedChildCount() == 0 {
		return ""
	}
	return n.NamedChild(0).Utf8Text(src)
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — PHP's own near-universal PSR convention distinguishing a
// class/interface/trait name (PascalCase) from a local variable/function
// name (camelCase/snake_case), mirroring pyextract/rustextract's identical
// heuristic.
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract/pyextract's own
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
