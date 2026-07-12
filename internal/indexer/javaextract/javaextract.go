package javaextract

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

// Extract walks a single Java file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch.
//
// moduleKey is the discovery-time, path-based placeholder computed by
// languages_java.go's LanguageSpec.ModuleKey (which cannot see file
// content). Java's real cross-file identity is declared IN the source
// (`package com.foo.bar;`, independent of directory layout — RESEARCH
// Pitfall 2) — once this file's own package declaration is parsed below,
// it OVERRIDES moduleKey as the returned FileResult.ImportPath (the
// symbolIndex's outer key, per symbolindex.go's byModuleKeyAndName). A file
// with no package declaration keeps the passed-in path-based moduleKey.
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "java",
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
		result.Err = fmt.Errorf("javaextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("javaextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "java",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	if pkg := findPackageDeclaration(root, src); pkg != "" {
		result.ImportPath = pkg
	}

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Imports are collected first so extends/implements and call
	// resolution below can tell an imported simple name (routes through
	// resolveSelector via the Imports map, RESEARCH Pitfall 2) apart from
	// a same-package one (routes through resolveUnqualified) — RESEARCH
	// Pitfall 3's declared-import ambiguity.
	ex.collectImports(root)
	// Types are collected before methods (05-PATTERNS.md's ordering
	// discipline) so a method's enclosing-type lookup (typeNodesByName)
	// always succeeds regardless of declaration order in the source.
	ex.collectTypes(root)
	ex.collectMethods(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract.extractor's shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *goextract.FileResult
	typeNodesByName map[string]string
}

// findPackageDeclaration returns the file's declared `package a.b.c;` name,
// or "" if the file has no package declaration (the default/unnamed
// package — legal Java, though unusual outside toy examples).
func findPackageDeclaration(root *tree_sitter.Node, src []byte) string {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		if c.Kind() != "package_declaration" {
			continue
		}
		for j := uint(0); j < c.NamedChildCount(); j++ {
			nc := c.NamedChild(j)
			if nc.Kind() == "identifier" || nc.Kind() == "scoped_identifier" {
				return nc.Utf8Text(src)
			}
		}
	}
	return ""
}

// --- imports ---

// collectImports walks the file's top-level import_declaration nodes,
// populating result.Imports (simple imported name -> its declaring
// package's module key) and emitting a RefKindImports unresolved ref per
// import statement.
func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() == "import_declaration" {
			ex.emitImportDeclaration(decl)
		}
	}
}

func (ex *extractor) emitImportDeclaration(decl *tree_sitter.Node) {
	// import_declaration: 'import' optional('static') _name optional('.' asterisk) ';'
	// _name is a hidden supertype rule, so its concrete alternative
	// (identifier for a single-segment/default-package import, or the far
	// more common scoped_identifier for a dotted FQN) appears directly as
	// decl's named child.
	var nameNode *tree_sitter.Node
	wildcard := false
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		switch c.Kind() {
		case "identifier", "scoped_identifier":
			nameNode = c
		case "asterisk":
			wildcard = true
		}
	}
	if nameNode == nil {
		return
	}

	fqn := nameNode.Utf8Text(ex.src)
	pos := decl.StartPosition()
	line := int32(pos.Row) + 1
	col := int32(pos.Column)

	if wildcard {
		// A wildcard import (`import com.foo.*;`) names a whole package,
		// not a single simple class — there is no one simple name to key
		// result.Imports by, so this only records the RefKindImports ref
		// (Task 1's "import statements produce RefKindImports unresolved
		// refs"); it does not extend the alias->moduleKey map a qualified
		// call/extends/implements reference can route through.
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: ex.fileID, Name: fqn, Kind: goextract.RefKindImports, Line: line, Col: col,
		})
		return
	}

	simpleName, pkg := splitJavaFQN(nameNode, ex.src)
	if simpleName != "" {
		ex.result.Imports[simpleName] = pkg
	}

	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: fqn, Kind: goextract.RefKindImports, Line: line, Col: col,
	})
}

// splitJavaFQN splits a parsed _name node's fully-qualified text into its
// simple (last-segment) name and declaring-package prefix. scoped_identifier
// carries explicit "scope"/"name" fields regardless of nesting depth, so the
// scope field's own source span is exactly the package prefix text — no
// manual dot-splitting needed. A bare identifier (single-segment import,
// the unnamed/default package) has no package prefix at all.
func splitJavaFQN(nameNode *tree_sitter.Node, src []byte) (simpleName, pkg string) {
	if nameNode.Kind() == "scoped_identifier" {
		scope := nameNode.ChildByFieldName("scope")
		name := nameNode.ChildByFieldName("name")
		if scope != nil && name != nil {
			return name.Utf8Text(src), scope.Utf8Text(src)
		}
	}
	return nameNode.Utf8Text(src), ""
}

// --- types (class / interface) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration":
			ex.emitTypeDecl(n, goextract.KindStruct)
		case "interface_declaration":
			ex.emitTypeDecl(n, goextract.KindInterface)
		}
		return true
	})
}

func (ex *extractor) emitTypeDecl(node *tree_sitter.Node, kind string) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(kind, name, ex.relPath)

	start := node.StartPosition()
	end := node.EndPosition()
	vis := javaVisibility(findModifiersChild(node))

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "java",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.typeNodesByName[name] = id

	ex.collectSupertypeRefs(id, node, kind)
}

// collectSupertypeRefs emits one RefKindEmbeds unresolved ref per supertype
// a class's `extends`/`implements` clause or an interface's own `extends`
// clause names (RESEARCH Pattern 2 — extends/implements are NOT
// distinguished at parse time; that promotion is Wave 6's job).
func (ex *extractor) collectSupertypeRefs(fromID string, node *tree_sitter.Node, kind string) {
	if kind == goextract.KindStruct {
		// class_declaration: field('superclass', $.superclass) wraps
		// 'extends' <type> as its own node's single named child; field(
		// 'interfaces', $.super_interfaces) wraps 'implements' <type_list>
		// the same way.
		if superWrap := node.ChildByFieldName("superclass"); superWrap != nil && superWrap.NamedChildCount() > 0 {
			ex.emitSupertypeRef(fromID, superWrap.NamedChild(0))
		}
		if ifaceWrap := node.ChildByFieldName("interfaces"); ifaceWrap != nil && ifaceWrap.NamedChildCount() > 0 {
			ex.emitTypeListRefs(fromID, ifaceWrap.NamedChild(0))
		}
		return
	}

	// interface_declaration's own `extends` clause (extends_interfaces) is
	// NOT wrapped in a field name by the grammar — find it by node kind
	// among the interface_declaration's direct named children.
	for i := uint(0); i < node.NamedChildCount(); i++ {
		c := node.NamedChild(i)
		if c.Kind() == "extends_interfaces" && c.NamedChildCount() > 0 {
			ex.emitTypeListRefs(fromID, c.NamedChild(0))
		}
	}
}

func (ex *extractor) emitTypeListRefs(fromID string, typeList *tree_sitter.Node) {
	if typeList.Kind() != "type_list" {
		// A super_interfaces/extends_interfaces node's single named child
		// is always a type_list per the grammar; defensively tolerate a
		// lone type too rather than silently dropping it.
		ex.emitSupertypeRef(fromID, typeList)
		return
	}
	for i := uint(0); i < typeList.NamedChildCount(); i++ {
		ex.emitSupertypeRef(fromID, typeList.NamedChild(i))
	}
}

func (ex *extractor) emitSupertypeRef(fromID string, typeNode *tree_sitter.Node) {
	name, ok := simpleTypeName(typeNode, ex.src)
	if !ok {
		return
	}
	// A supertype reference is unambiguously a type by grammar shape (no
	// local-variable ambiguity the way a call's operand can have) — only
	// set PkgAlias when name is a genuine import; a same-package supertype
	// needs no import in Java, so an empty PkgAlias correctly routes
	// through resolveUnqualified against the caller's own moduleKey.
	pkgAlias := ""
	if _, imported := ex.result.Imports[name]; imported {
		pkgAlias = name
	}
	pos := typeNode.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// simpleTypeName extracts a type reference node's simple (unqualified)
// name — the shape a superclass/interfaces type or a type_list entry can
// take: a plain type_identifier, a generic_type (unwrap to its own base
// type, ignoring type arguments), or a scoped_type_identifier (take its
// "name" field, the rightmost segment).
func simpleTypeName(t *tree_sitter.Node, src []byte) (string, bool) {
	switch t.Kind() {
	case "type_identifier":
		return t.Utf8Text(src), true
	case "generic_type":
		if t.NamedChildCount() == 0 {
			return "", false
		}
		return simpleTypeName(t.NamedChild(0), src)
	case "scoped_type_identifier":
		if name := t.ChildByFieldName("name"); name != nil {
			return name.Utf8Text(src), true
		}
		return "", false
	default:
		return "", false
	}
}

func findModifiersChild(node *tree_sitter.Node) *tree_sitter.Node {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if c := node.NamedChild(i); c.Kind() == "modifiers" {
			return c
		}
	}
	return nil
}

// javaVisibility scans a modifiers node's (unnamed keyword token) children
// for the standard Java access modifiers. No explicit modifier (or no
// modifiers node at all) is Java's own default access level, "package"
// (a.k.a. package-private).
func javaVisibility(modifiers *tree_sitter.Node) string {
	if modifiers == nil {
		return "package"
	}
	for i := uint(0); i < modifiers.ChildCount(); i++ {
		switch modifiers.Child(i).Kind() {
		case "public":
			return "public"
		case "private":
			return "private"
		case "protected":
			return "protected"
		}
	}
	return "package"
}

// --- methods / constructors / calls ---

func (ex *extractor) collectMethods(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration", "interface_declaration":
			ex.emitMethodsForType(n)
		}
		return true
	})
}

// emitMethodsForType processes one class/interface declaration's OWN
// direct body children (method_declaration/constructor_declaration) — not
// a nested class/interface's methods, which this same collectMethods walk
// visits separately when it reaches that nested declaration node.
func (ex *extractor) emitMethodsForType(typeNode *tree_sitter.Node) {
	nameNode := typeNode.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	typeName := nameNode.Utf8Text(ex.src)
	typeID, ok := ex.typeNodesByName[typeName]
	if !ok {
		// collectTypes (pass 1) skipped this declaration (e.g. a missing
		// name node) — nothing to attach methods to.
		return
	}
	body := typeNode.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		decl := body.NamedChild(i)
		switch decl.Kind() {
		case "method_declaration":
			ex.emitMethod(typeID, typeName, decl, false)
		case "constructor_declaration":
			ex.emitMethod(typeID, typeName, decl, true)
		}
	}
}

func (ex *extractor) emitMethod(typeID, typeName string, decl *tree_sitter.Node, isConstructor bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := typeName + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()

	var signature, returnType string
	if params := decl.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	}
	if !isConstructor {
		if rt := decl.ChildByFieldName("type"); rt != nil {
			returnType = rt.Utf8Text(ex.src)
		}
	}
	vis := javaVisibility(findModifiersChild(decl))

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindMethod,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "java",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Signature:     signature,
		ReturnType:    returnType,
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
	}})

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "method_invocation" {
			ex.recordCall(fromID, n)
		}
		return true
	})
}

func (ex *extractor) recordCall(fromID string, call *tree_sitter.Node) {
	nameNode := call.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	pos := call.StartPosition()
	line := int32(pos.Row) + 1
	col := int32(pos.Column)

	object := call.ChildByFieldName("object")
	var pkgAlias string
	switch {
	case object == nil:
		// No object at all — an implicit same-class call, resolved
		// same-package/same-file via an empty PkgAlias
		// (resolveUnqualified).
	case object.Kind() == "this":
		// An explicit `this.method()` — same as the no-object case
		// above.
	case object.Kind() == "identifier":
		ident := object.Utf8Text(ex.src)
		_, isImport := ex.result.Imports[ident]
		switch {
		case isImport:
			// A real import alias — routes through resolveSelector to
			// the imported class's own declaring package.
			pkgAlias = ident
		case isLikelyTypeName(ident):
			// An uppercase-leading identifier that is NOT a real import
			// is, by Java naming convention, very likely a same-package
			// class reference (`Helper.assist()` needs no import when
			// Helper shares Caller's package) — an empty PkgAlias
			// correctly routes through resolveUnqualified against the
			// caller's own moduleKey (RESEARCH Pitfall 3's declared-
			// import ambiguity, resolved here by convention since this
			// extractor tracks no per-file local-variable type table).
		default:
			// A lowercase-leading identifier is very likely a local
			// variable or field receiver, not a type — force it through
			// resolveSelector's alias-membership boundary via a
			// synthetic non-matching alias so it deterministically falls
			// through to "unresolved" instead of risking a same-package
			// false match (mirrors goextract.go's WR-02 fix).
			pkgAlias = "<local:" + ident + ">"
		}
	default:
		// A non-identifier operand (`super`, another method_invocation, a
		// field_access chain, ...) can never be a real import alias —
		// same synthetic-alias treatment as the local-variable case
		// above, mirroring goextract.go's WR-02 fix.
		pkgAlias = "<" + object.Kind() + ">"
	}

	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls,
		Line: line, Col: col,
	})
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — the near-universal Java convention distinguishing a type name
// (PascalCase) from a local variable/field/parameter name (camelCase).
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract.go's own
// walkDescendants (T-02-04 depth guard — no unbounded Go-stack recursion
// over a pathologically deep AST; the tree is already size-bounded by
// parser.MaxSourceBytes).
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
