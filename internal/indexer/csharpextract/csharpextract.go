package csharpextract

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

// Extract walks a single C# file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch.
//
// moduleKey is the discovery-time, path-based placeholder computed by
// languages_csharp.go's LanguageSpec.ModuleKey (which cannot see file
// content). C#'s real cross-file identity is declared IN the source
// (`namespace Foo.Bar;` or `namespace Foo.Bar { ... }`, independent of
// directory layout — RESEARCH Pitfall 2) — once this file's own namespace
// declaration is parsed below, it OVERRIDES moduleKey as the returned
// FileResult.ImportPath (the symbolIndex's outer key, per symbolindex.go's
// byModuleKeyAndName). A file with no namespace declaration keeps the
// passed-in path-based moduleKey.
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "csharp",
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
		result.Err = fmt.Errorf("csharpextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("csharpextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "csharp",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	if ns := findNamespace(root, src); ns != "" {
		result.ImportPath = ns
	}

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Usings are collected first purely for consistency with Go/Java's
	// ordering discipline (imports-before-types) — unlike Java, a plain
	// `using` doesn't populate an alias this extractor's call/embeds
	// resolution consults (see types.go's package doc); only the alias
	// form (`using X = Y;`) does.
	ex.collectUsings(root)
	// Types are collected before methods (05-PATTERNS.md's ordering
	// discipline) so a method's enclosing-type lookup (typeNodesByName)
	// always succeeds regardless of declaration order in the source.
	ex.collectTypes(root)
	ex.collectMethods(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring javaextract.extractor's shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *goextract.FileResult
	typeNodesByName map[string]string
}

// findNamespace returns the file's top-level declared namespace (block-form
// `namespace Foo.Bar { ... }` or file-scoped `namespace Foo.Bar;`), or ""
// if none is declared (the global/unnamed namespace — legal, if unusual,
// C#). Only a TOP-LEVEL namespace declaration is consulted (RESEARCH
// Pitfall 2's "one moduleKey per file" simplification, mirroring
// javaextract's single `package` statement assumption) — a file with
// multiple or deeply nested namespace blocks is an accepted, documented
// simplification, not a crash: every type in the file is still attributed
// to this one detected namespace (or the path-based placeholder if none).
func findNamespace(root *tree_sitter.Node, src []byte) string {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		switch c.Kind() {
		case "namespace_declaration", "file_scoped_namespace_declaration":
			if n := c.ChildByFieldName("name"); n != nil {
				return n.Utf8Text(src)
			}
		}
	}
	return ""
}

// --- using directives ---

func (ex *extractor) collectUsings(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		if c := root.NamedChild(i); c.Kind() == "using_directive" {
			ex.emitUsingDirective(c)
		}
	}
}

func (ex *extractor) emitUsingDirective(decl *tree_sitter.Node) {
	// using_directive: optional('global') 'using' choice(
	//   seq(optional('unsafe'), field('name', identifier), '=', type),  // alias form
	//   seq(repeat(choice('static','unsafe')), _name),                 // plain/static form
	// ) ';'
	// aliasNode is only non-nil for the alias form; the plain/static form
	// never sets the "name" field at all.
	aliasNode := decl.ChildByFieldName("name")
	var target *tree_sitter.Node
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if aliasNode != nil && c.StartByte() == aliasNode.StartByte() && c.EndByte() == aliasNode.EndByte() {
			// This IS the alias identifier itself (decl's first named
			// child in the alias form) — skip it, we want the type/name
			// node that follows.
			continue
		}
		switch c.Kind() {
		case "identifier", "qualified_name", "generic_name", "alias_qualified_name":
			target = c
		}
	}
	if target == nil {
		return
	}
	fqn := target.Utf8Text(ex.src)
	pos := decl.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: fqn, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})

	// `using X = Y.Z;` is a genuine alias (unlike a plain namespace-level
	// `using`, which names no single simple type this extractor could key
	// an Imports entry by) — record it exactly as Go/Java record a real
	// import alias.
	if aliasNode != nil {
		ex.result.Imports[aliasNode.Utf8Text(ex.src)] = fqn
	}
}

// --- type declarations (class / struct / record / interface) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration", "struct_declaration", "record_declaration":
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
	mods := modifiersOf(node, ex.src)
	partial := containsMod(mods, "partial")

	var id, filePath string
	var startLine, endLine, startCol, endCol int32
	if partial {
		// Pitfall 5 scheme (b) — see types.go's package doc comment for
		// the full rationale: shared node keyed by (qualifiedName,
		// namespace) only, deterministic sentinel location fields.
		id = nodeid.NodeID(kind, name, ex.result.ImportPath)
	} else {
		id = nodeid.NodeID(kind, name, ex.relPath)
		filePath = ex.relPath
		start := node.StartPosition()
		end := node.EndPosition()
		startLine, endLine = int32(start.Row)+1, int32(end.Row)+1
		startCol, endCol = int32(start.Column), int32(end.Column)
	}

	vis := csharpVisibility(mods, "internal")

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      filePath,
		Language:      "csharp",
		StartLine:     startLine,
		EndLine:       endLine,
		StartCol:      startCol,
		EndCol:        endCol,
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.typeNodesByName[name] = id

	ex.collectSupertypeRefs(id, node)
	if body := node.ChildByFieldName("body"); body != nil {
		ex.collectFieldTypeOfRefs(id, body)
	}
}

// collectSupertypeRefs emits one RefKindEmbeds unresolved ref per entry in
// a type's base_list (`: IFoo, BaseClass`) — RESEARCH Pattern 2: extends
// and implements are NOT distinguished at parse time; that promotion is
// Wave 6's job.
func (ex *extractor) collectSupertypeRefs(fromID string, node *tree_sitter.Node) {
	var baseList *tree_sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if c := node.NamedChild(i); c.Kind() == "base_list" {
			baseList = c
			break
		}
	}
	if baseList == nil {
		return
	}
	for i := uint(0); i < baseList.NamedChildCount(); i++ {
		ex.emitSupertypeRef(fromID, baseList.NamedChild(i))
	}
}

func (ex *extractor) emitSupertypeRef(fromID string, t *tree_sitter.Node) {
	if t.Kind() == "primary_constructor_base_type" {
		// `class Foo(int x) : Base(x)` — the base type itself is field
		// "type"; the trailing "(x)" is a call-argument list, not a type
		// reference.
		inner := t.ChildByFieldName("type")
		if inner == nil {
			return
		}
		t = inner
	}
	simpleName, prefix, ok := csharpQualifiedParts(t, ex.src)
	if !ok {
		return
	}
	pkgAlias := ex.namespacePrefixAlias(prefix, simpleName)
	pos := t.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: simpleName, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// csharpQualifiedParts extracts a type reference node's simple (rightmost)
// name and, when the reference is fully qualified, its declaring-namespace
// prefix text — the shapes a base_list entry or a call's qualified operand
// can take: a plain identifier, a predefined_type (`int`, `string`, ...), a
// generic_name (unwrap to its own base identifier, ignoring type
// arguments), or a qualified_name (whose "qualifier" field's own source
// span is exactly the namespace prefix text — no manual dot-splitting
// needed, mirroring javaextract's splitJavaFQN).
func csharpQualifiedParts(t *tree_sitter.Node, src []byte) (simpleName, prefix string, ok bool) {
	switch t.Kind() {
	case "identifier", "predefined_type":
		return t.Utf8Text(src), "", true
	case "generic_name":
		if t.NamedChildCount() == 0 {
			return "", "", false
		}
		return t.NamedChild(0).Utf8Text(src), "", true
	case "qualified_name":
		nameField := t.ChildByFieldName("name")
		qualifier := t.ChildByFieldName("qualifier")
		if nameField == nil || qualifier == nil {
			return "", "", false
		}
		simple := nameField.Utf8Text(src)
		if nameField.Kind() == "generic_name" && nameField.NamedChildCount() > 0 {
			simple = nameField.NamedChild(0).Utf8Text(src)
		}
		return simple, qualifier.Utf8Text(src), true
	default:
		return "", "", false
	}
}

// namespacePrefixAlias resolves the PkgAlias a call/embeds reference should
// carry — see types.go's package doc comment ("Cross-namespace call/embeds
// qualifier resolution") for the full, documented rationale. prefix is
// non-empty only when the reference's own AST shape already spells out its
// declaring namespace (a fully-qualified qualified_name); simpleName is
// only consulted when prefix is empty, to apply the PascalCase (type-like,
// same-namespace attempt) vs. camelCase (local-variable-like, synthetic
// non-matching alias) heuristic.
func (ex *extractor) namespacePrefixAlias(prefix, simpleName string) string {
	if prefix != "" {
		// Self-mapped: resolveSelector's exact-match lookup needs
		// result.Imports[pkgAlias] to resolve to a moduleKey, and a fully-
		// qualified reference's own namespace prefix text IS that
		// moduleKey (no `using` declaration required at all).
		ex.result.Imports[prefix] = prefix
		return prefix
	}
	if isLikelyTypeName(simpleName) {
		return ""
	}
	return "<local:" + simpleName + ">"
}

// --- methods / constructors / calls ---

func (ex *extractor) collectMethods(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration", "struct_declaration", "record_declaration", "interface_declaration":
			ex.emitMethodsForType(n)
		}
		return true
	})
}

// emitMethodsForType processes one type declaration's OWN direct body
// children (method_declaration/constructor_declaration) — not a nested
// type's methods, which this same collectMethods walk visits separately
// when it reaches that nested declaration node. property_declaration and
// field_declaration are deliberately absent from the switch below (the
// field-skip precedent, types.go's package doc).
func (ex *extractor) emitMethodsForType(typeNode *tree_sitter.Node) {
	nameNode := typeNode.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	typeName := nameNode.Utf8Text(ex.src)
	typeID, ok := ex.typeNodesByName[typeName]
	if !ok {
		return
	}
	body := typeNode.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		decl := body.NamedChild(i)
		switch decl.Kind() {
		case "method_declaration", "constructor_declaration":
			ex.emitMethod(typeID, typeName, decl)
		}
	}
}

func (ex *extractor) emitMethod(typeID, typeName string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	// A constructor's own "name" field text already equals typeName by
	// grammar rule (valid C# requires it) — no isConstructor flag needed,
	// unlike javaextract: ChildByFieldName("returns") below simply returns
	// nil for a constructor_declaration (that field doesn't exist on that
	// node type), so returnType naturally stays "".
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := typeName + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()

	var signature, returnType string
	var returnTypeNode *tree_sitter.Node
	if params := decl.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	}
	if rt := decl.ChildByFieldName("returns"); rt != nil {
		returnType = rt.Utf8Text(ex.src)
		returnTypeNode = rt
	}
	vis := csharpVisibility(modifiersOf(decl, ex.src), "private")

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindMethod,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "csharp",
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

	// D-09 (01-RESEARCH.md §B): reuses the already-parsed return-type node
	// (returnTypeNode, above) — emitNamedTypeRef isolates the type NAME for
	// node resolution via namedTypeRef, mirroring goextract's
	// collectReturnTypeRef. A primitive/void return (predefined_type) or a
	// constructor (returnTypeNode stays nil, C# constructor_declaration has
	// no "returns" field) emits no ref — absence, not error.
	ex.emitNamedTypeRef(id, returnTypeNode, goextract.RefKindReturns)

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
		ex.collectReferencesAndInstantiates(id, body)
	}
}

// namedTypeRef resolves a type expression node to a (name, pkgAlias) pair
// via csharpQualifiedParts, filtering C#'s predefined/primitive types (int,
// string, void, ...) — mirrors goextract.namedTypeRef's Go predeclared-type
// filtering discipline (01-RESEARCH.md §B). A compound/unnamed type shape
// (array_type, nullable_type, tuple_type, implicit_type/"var", ...) is
// already excluded by csharpQualifiedParts' own switch.
func (ex *extractor) namedTypeRef(t *tree_sitter.Node) (name, pkgAlias string, ok bool) {
	if t == nil || t.Kind() == "predefined_type" {
		return "", "", false
	}
	simple, prefix, ok := csharpQualifiedParts(t, ex.src)
	if !ok {
		return "", "", false
	}
	return simple, ex.namespacePrefixAlias(prefix, simple), true
}

// emitNamedTypeRef emits a D-09 Pass-1 ref (01-RESEARCH.md §B) of the given
// kind (RefKindReturns/RefKindTypeOf) from fromID to typeNode's simple named
// type reference, when typeNode resolves to one via namedTypeRef. A
// primitive/void/compound-shaped type emits no ref — absence, not error.
func (ex *extractor) emitNamedTypeRef(fromID string, typeNode *tree_sitter.Node, kind string) {
	if typeNode == nil {
		return
	}
	name, pkgAlias, ok := ex.namedTypeRef(typeNode)
	if !ok {
		return
	}
	pos := typeNode.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: kind,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// recordInstantiate emits a D-09 `instantiates` Pass-1 ref (01-RESEARCH.md
// §B) for a `new T(...)` object_creation_expression whose "type" field is a
// single named type reference (namedTypeRef) — mirrors
// goextract.recordInstantiate's shape exactly. The resolved target's
// Kind-check disambiguation (must be a class, not an interface) happens at
// Pass 2 (resolve.go), not here.
func (ex *extractor) recordInstantiate(fromID string, creation *tree_sitter.Node) {
	t := creation.ChildByFieldName("type")
	if t == nil {
		return
	}
	name, pkgAlias, ok := ex.namedTypeRef(t)
	if !ok {
		return
	}
	pos := creation.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindInstantiates,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// collectReferencesAndInstantiates walks a method/constructor body for
// three D-09 Pass-1 capture kinds (01-RESEARCH.md §B): instantiates (via
// recordInstantiate), type_of for LOCAL variable declarations (anchored at
// the enclosing method id fromID — field type_of is handled separately by
// collectFieldTypeOfRefs, anchored at the enclosing type, since this
// extractor emits no field OR local-variable node either), and references
// (a value read of an identifier/member-access that is NOT an
// invocation_expression's own callee/qualifier — collectCalls already
// captures those via its own independent whole-body scan, and de-dup
// requires a called symbol never ALSO emit a references ref).
//
// C# D-02 precision note (mirrors goextract's/javaextract's own note
// exactly, per 01-RESEARCH.md §B): this walk is scoped to a bounded
// allow-list of unambiguous read positions — call/constructor arguments,
// return values, local-variable initializers, and assignment right-hand
// sides — plus the common compound-expression wrappers reachable from them
// via captureExprRead — rather than exhaustively covering every C#
// expression shape. This is a deliberate, bounded scope, not a silent drop
// of ground truth — an over-broad walk risks the exact false
// same-namespace-name-collision resolution recordCall's local-vs-type
// discipline (mirrored below in captureMemberAccessRead) already guards
// against for `calls`.
func (ex *extractor) collectReferencesAndInstantiates(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "invocation_expression":
			// De-dup: the callee ("function") is already captured by
			// collectCalls' own separate walk — only "arguments" are
			// additional read positions.
			if args := n.ChildByFieldName("arguments"); args != nil {
				for i := uint(0); i < args.NamedChildCount(); i++ {
					ex.captureExprRead(fromID, args.NamedChild(i))
				}
			}
			return false
		case "object_creation_expression":
			ex.recordInstantiate(fromID, n)
			if args := n.ChildByFieldName("arguments"); args != nil {
				for i := uint(0); i < args.NamedChildCount(); i++ {
					ex.captureExprRead(fromID, args.NamedChild(i))
				}
			}
			return false
		case "return_statement":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				ex.captureExprRead(fromID, n.NamedChild(i))
			}
			return false
		case "local_declaration_statement":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				vd := n.NamedChild(i)
				if vd.Kind() != "variable_declaration" {
					continue
				}
				ex.emitNamedTypeRef(fromID, vd.ChildByFieldName("type"), goextract.RefKindTypeOf)
				for j := uint(0); j < vd.NamedChildCount(); j++ {
					c := vd.NamedChild(j)
					if c.Kind() != "variable_declarator" {
						continue
					}
					// A variable_declarator's optional initializer is its
					// SECOND named child (index 1) with no field name — C#'s
					// grammar exposes only "name" as a field here, unlike
					// Java's variable_declarator which names its initializer
					// "value" (verified via a live parse this session, not
					// just the docs).
					if c.NamedChildCount() > 1 {
						ex.captureExprRead(fromID, c.NamedChild(1))
					}
				}
			}
			return false
		case "assignment_expression":
			if right := n.ChildByFieldName("right"); right != nil {
				ex.captureExprRead(fromID, right)
			}
			return false
		}
		return true
	})
}

// captureExprRead classifies a single expression node reached from an
// allow-listed read position (see collectReferencesAndInstantiates) and
// emits a references ref for a bare identifier or member-access value read,
// recursing through common compound-expression wrappers so a nested read
// inside one of those is still found — mirrors goextract.captureExprRead's
// shape.
func (ex *extractor) captureExprRead(fromID string, expr *tree_sitter.Node) {
	if expr == nil {
		return
	}
	switch expr.Kind() {
	case "identifier":
		pos := expr.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: expr.Utf8Text(ex.src), Kind: goextract.RefKindReferences,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	case "member_access_expression":
		ex.captureMemberAccessRead(fromID, expr)
	case "parenthesized_expression":
		if expr.NamedChildCount() > 0 {
			ex.captureExprRead(fromID, expr.NamedChild(0))
		}
	case "cast_expression":
		if v := expr.ChildByFieldName("value"); v != nil {
			ex.captureExprRead(fromID, v)
		}
	case "unary_expression":
		if o := expr.ChildByFieldName("argument"); o != nil {
			ex.captureExprRead(fromID, o)
		}
	case "binary_expression":
		if l := expr.ChildByFieldName("left"); l != nil {
			ex.captureExprRead(fromID, l)
		}
		if r := expr.ChildByFieldName("right"); r != nil {
			ex.captureExprRead(fromID, r)
		}
	case "invocation_expression":
		// A nested call (e.g. an argument that is itself a call) — its own
		// callee is never a reference (de-dup, mirroring the outer
		// collectReferencesAndInstantiates rule); only its arguments are
		// walked here.
		if args := expr.ChildByFieldName("arguments"); args != nil {
			for i := uint(0); i < args.NamedChildCount(); i++ {
				ex.captureExprRead(fromID, args.NamedChild(i))
			}
		}
	case "object_creation_expression":
		ex.recordInstantiate(fromID, expr)
		if args := expr.ChildByFieldName("arguments"); args != nil {
			for i := uint(0); i < args.NamedChildCount(); i++ {
				ex.captureExprRead(fromID, args.NamedChild(i))
			}
		}
	}
	// Every other expression kind (literals, lambda_expression bodies,
	// is_pattern_expression, array_creation_expression, ...) is out of this
	// bounded allow-list per the C# D-02 precision note above — no
	// reference captured, no error.
}

// captureMemberAccessRead handles a member_access_expression VALUE read
// (`Type.Field`/`obj.Field` used as a value, not called — recordCall's own
// invocation_expression handling already captures the call-callee shape
// via collectCalls' separate walk). Mirrors recordCall's memberAccessAlias/
// local-vs-type alias discipline for PkgAlias, and mirrors
// goextract.captureSelectorRead's recursion discipline exactly: an
// identifier (or `this`) operand's own read is fully represented by the
// emitted ref's PkgAlias/Name pair (never re-walked as its own reference);
// any other operand shape (a nested member_access_expression chain, an
// invocation_expression result, ...) is walked further via captureExprRead.
func (ex *extractor) captureMemberAccessRead(fromID string, sel *tree_sitter.Node) {
	object := sel.ChildByFieldName("expression")
	field := sel.ChildByFieldName("name")
	if field == nil {
		return
	}
	name := field.Utf8Text(ex.src)
	if field.Kind() == "generic_name" && field.NamedChildCount() > 0 {
		name = field.NamedChild(0).Utf8Text(ex.src)
	}
	pkgAlias := ex.memberAccessAlias(object)
	if object != nil && object.Kind() != "identifier" && object.Kind() != "this" {
		ex.captureExprRead(fromID, object)
	}
	pos := sel.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindReferences,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// collectFieldTypeOfRefs emits a D-09 `type_of` Pass-1 ref (01-RESEARCH.md
// §B) for each of typeID's own type body's direct field_declaration
// children, anchored at typeID itself — a documented C# D-02 precision
// note (mirrors this package's pre-existing "no field node" skip, types.go's
// package doc): since this extractor never emits a field node, a field's
// declared-type ref has no field-level FromID to anchor on, so it is
// recorded against the enclosing type instead. A primitive-typed field
// (predefined_type) emits no ref via emitNamedTypeRef's own filtering.
func (ex *extractor) collectFieldTypeOfRefs(typeID string, body *tree_sitter.Node) {
	for i := uint(0); i < body.NamedChildCount(); i++ {
		fd := body.NamedChild(i)
		if fd.Kind() != "field_declaration" {
			continue
		}
		for j := uint(0); j < fd.NamedChildCount(); j++ {
			if vd := fd.NamedChild(j); vd.Kind() == "variable_declaration" {
				ex.emitNamedTypeRef(typeID, vd.ChildByFieldName("type"), goextract.RefKindTypeOf)
			}
		}
	}
}

func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "invocation_expression" {
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
		// No object at all — an implicit `this.Method()` call.
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: fn.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "member_access_expression":
		nameNode := fn.ChildByFieldName("name")
		if nameNode == nil {
			return
		}
		name := nameNode.Utf8Text(ex.src)
		if nameNode.Kind() == "generic_name" && nameNode.NamedChildCount() > 0 {
			name = nameNode.NamedChild(0).Utf8Text(ex.src)
		}
		pkgAlias := ex.memberAccessAlias(fn.ChildByFieldName("expression"))
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	default:
		// Other function shapes (conditional_access_expression,
		// element_access_expression, cast_expression,
		// parenthesized_expression, another invocation_expression — e.g. a
		// delegate/lambda call result) are left unrecorded rather than
		// guessed at.
	}
}

// memberAccessAlias resolves a member_access_expression's PkgAlias from its
// "expression" (object) operand.
func (ex *extractor) memberAccessAlias(object *tree_sitter.Node) string {
	if object == nil {
		// No object at all (malformed/unusual source) — implicit this.
		return ""
	}
	switch object.Kind() {
	case "this":
		return ""
	case "identifier", "predefined_type":
		return ex.namespacePrefixAlias("", object.Utf8Text(ex.src))
	case "qualified_name":
		if q := object.ChildByFieldName("qualifier"); q != nil {
			return ex.namespacePrefixAlias(q.Utf8Text(ex.src), "")
		}
		return "<qualified_name>"
	case "member_access_expression":
		// An EXPRESSION-position dotted chain (`Other.Namespace.Helper`,
		// invoking `.Assist()` on it, OR `obj.Field.Method()`, a plain
		// local-variable property/field access chain) parses as nested
		// member_access_expression nodes, not qualified_name (that grammar
		// rule is reserved for TYPE positions — base_list, using_directive
		// — see csharpQualifiedParts). These two shapes are only
		// distinguishable by walking down to the chain's ROOT identifier:
		// an uppercase-leading root (`Other...`) is a namespace/type-shaped
		// chain, so object's own "expression" field text (the chain MINUS
		// its last segment — here "Other.Namespace", the discarded
		// "Helper" type-name segment) is used as PkgAlias, mirroring the
		// qualified_name "qualifier" field's own text-based extraction
		// above. A lowercase-leading root (`obj...`) is a local-
		// variable/field access chain — never a namespace — routed through
		// the same synthetic non-matching alias as a direct local-variable
		// receiver (goextract's WR-02 fix).
		root := chainRoot(object)
		if root.Kind() != "identifier" {
			return "<" + object.Kind() + ">"
		}
		rootName := root.Utf8Text(ex.src)
		if !isLikelyTypeName(rootName) {
			return "<local:" + rootName + ">"
		}
		if inner := object.ChildByFieldName("expression"); inner != nil {
			return ex.namespacePrefixAlias(inner.Utf8Text(ex.src), "")
		}
		return "<member_access_expression>"
	default:
		// A non-identifier operand (another invocation_expression, an
		// element_access_expression, a conditional_access_expression, ...)
		// can never be a real namespace/type reference — same synthetic-
		// alias treatment as the local-variable case, mirroring
		// goextract.go's WR-02 fix: force this through resolveSelector's
		// narrowest-safe-set alias-membership boundary via an alias that
		// can never equal a real map key, so it deterministically ends up
		// "unresolved" instead of risking a same-namespace false match.
		return "<" + object.Kind() + ">"
	}
}

// chainRoot walks down a member_access_expression chain's "expression"
// field until it reaches a non-member_access_expression node — the root
// operand the whole dotted chain is ultimately built on (e.g. "Other" in
// `Other.Namespace.Helper`, or "obj" in `obj.Field.Method`).
func chainRoot(n *tree_sitter.Node) *tree_sitter.Node {
	for n.Kind() == "member_access_expression" {
		e := n.ChildByFieldName("expression")
		if e == nil {
			return n
		}
		n = e
	}
	return n
}

// --- modifiers / visibility ---

// modifiersOf scans node's direct named children for C# "modifier" nodes
// (public/private/protected/internal/static/partial/...), returning their
// literal keyword text. Unlike Java's single wrapping "modifiers" node, C#
// modifiers are direct siblings of the declaration's other children — no
// wrapper node exists.
func modifiersOf(node *tree_sitter.Node, src []byte) []string {
	var mods []string
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if c := node.NamedChild(i); c.Kind() == "modifier" {
			mods = append(mods, c.Utf8Text(src))
		}
	}
	return mods
}

func containsMod(mods []string, want string) bool {
	for _, m := range mods {
		if m == want {
			return true
		}
	}
	return false
}

// csharpVisibility derives a declaration's access level from its scanned
// modifier tokens, handling the two legal compound-access modifiers
// ("protected internal", "private protected") before falling back to a
// single public/private/protected/internal token, or defaultVis when no
// access modifier is present at all. C# has two different implicit
// defaults depending on declaration context — a top-level type defaults to
// "internal"; a type MEMBER defaults to "private" — so callers pass the
// correct default explicitly rather than this function guessing from
// context.
func csharpVisibility(mods []string, defaultVis string) string {
	var hasPublic, hasPrivate, hasProtected, hasInternal bool
	for _, m := range mods {
		switch m {
		case "public":
			hasPublic = true
		case "private":
			hasPrivate = true
		case "protected":
			hasProtected = true
		case "internal":
			hasInternal = true
		}
	}
	switch {
	case hasProtected && hasInternal:
		return "protected internal"
	case hasPrivate && hasProtected:
		return "private protected"
	case hasPublic:
		return "public"
	case hasPrivate:
		return "private"
	case hasProtected:
		return "protected"
	case hasInternal:
		return "internal"
	default:
		return defaultVis
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — the near-universal C# convention distinguishing a type name
// (PascalCase) from a local variable/field/parameter name (camelCase),
// exactly mirroring javaextract's own heuristic.
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract.go/javaextract.go's
// own walkDescendants (T-02-04 depth guard — no unbounded Go-stack
// recursion over a pathologically deep AST; the tree is already
// size-bounded by parser.MaxSourceBytes).
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
