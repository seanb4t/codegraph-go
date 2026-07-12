package swiftextract

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

// Extract walks a single Swift file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for
// alex-pinkus/tree-sitter-swift's external C scanner (string
// interpolation).
//
// moduleKey is the discovery-time path-identity placeholder
// (languages_swift.go's LanguageSpec.ModuleKey — a best-effort Swift Package
// Manager target-directory guess, or path identity) — Extract never
// overrides FileResult.ImportPath; Swift has no in-source module
// declaration equivalent to PHP/C#'s namespace (see types.go).
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "swift",
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
		result.Err = fmt.Errorf("swiftextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("swiftextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "swift",
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

	// Types are collected first so a method's containing-type lookup always
	// sees every class/struct/protocol this file declares, regardless of
	// declaration order (mirrors rustextract/phpextract's ordering
	// discipline).
	ex.collectImports(root)
	ex.collectTypes(root)
	ex.collectTopLevelFunctions(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract/rustextract's extractor shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *goextract.FileResult
	typeNodesByName map[string]string
}

// --- import_declaration ---

func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := root.NamedChild(i)
		if d.Kind() != "import_declaration" {
			continue
		}
		ex.emitImport(d)
	}
}

func (ex *extractor) emitImport(decl *tree_sitter.Node) {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() != "identifier" {
			continue
		}
		name := c.Utf8Text(ex.src)
		if name == "" {
			continue
		}
		pos := decl.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: ex.fileID, Name: name, Kind: goextract.RefKindImports,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
		return
	}
}

// --- class_declaration (class/struct/enum/actor/extension) + protocol_declaration ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := root.NamedChild(i)
		switch d.Kind() {
		case "class_declaration":
			ex.emitClassLike(d)
		case "protocol_declaration":
			ex.emitProtocol(d)
		}
	}
}

// swiftDeclKind scans n's own (including anonymous) children for the
// declaration-kind keyword token ("class"/"struct"/"enum"/"actor"/
// "extension") that distinguishes what class_declaration actually declares
// (types.go's documented grammar-shape adaptation).
func swiftDeclKind(n *tree_sitter.Node) string {
	for i := uint(0); i < n.ChildCount(); i++ {
		switch n.Child(i).Kind() {
		case "class", "struct", "enum", "actor", "extension":
			return n.Child(i).Kind()
		}
	}
	return ""
}

func (ex *extractor) emitClassLike(decl *tree_sitter.Node) {
	kind := swiftDeclKind(decl)
	if kind == "extension" {
		// Recognized, but its own body is never walked (types.go's
		// documented gap) -- an extension's members would need to merge
		// into its (possibly cross-file) extended type's own method set,
		// which this tier does not implement.
		return
	}
	if kind == "" {
		return
	}

	nameNode := decl.ChildByFieldName("name")
	name, ok := swiftSimpleTypeName(nameNode, ex.src)
	if !ok {
		return
	}
	id := nodeid.NodeID(goextract.KindStruct, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindStruct,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "swift",
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

	ex.collectInheritance(id, decl)
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectMethods(id, name, body)
	}
}

func (ex *extractor) emitProtocol(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	name, ok := swiftSimpleTypeName(nameNode, ex.src)
	if !ok {
		return
	}
	id := nodeid.NodeID(goextract.KindInterface, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindInterface,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "swift",
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

	ex.collectInheritance(id, decl)
	// protocol_function_declaration (a protocol requirement, always
	// bodyless) is a DIFFERENT node kind never walked here -- documented
	// "no bodyless prototype node" skip (types.go).
}

// swiftSimpleTypeName resolves a class_declaration/protocol_declaration's
// own "name" field down to a simple type name: a plain type_identifier
// (the common case), or a user_type wrapping one (the shape an EXTENSION's
// own "name" field takes -- verified via a live parse-tree dump).
func swiftSimpleTypeName(n *tree_sitter.Node, src []byte) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind() {
	case "type_identifier":
		return n.Utf8Text(src), true
	case "user_type":
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c.Kind() == "type_identifier" {
				return c.Utf8Text(src), true
			}
		}
		return "", false
	default:
		return "", false
	}
}

// collectInheritance walks decl's own direct inheritance_specifier children
// (one per superclass/protocol conformance -- verified via a live
// parse-tree dump of a multi-conformance `class Foo: A, B`).
func (ex *extractor) collectInheritance(id string, decl *tree_sitter.Node) {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() != "inheritance_specifier" {
			continue
		}
		inheritsFrom := c.ChildByFieldName("inherits_from")
		name, ok := swiftSimpleTypeName(inheritsFrom, ex.src)
		if !ok && inheritsFrom != nil && inheritsFrom.Kind() == "type_identifier" {
			name, ok = inheritsFrom.Utf8Text(ex.src), true
		}
		if !ok {
			continue
		}
		pos := c.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: id, Name: name, Kind: goextract.RefKindEmbeds,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}
}

// --- function_declaration (methods and top-level functions) ---

func (ex *extractor) collectMethods(typeID, typeName string, body *tree_sitter.Node) {
	for i := uint(0); i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		if m.Kind() != "function_declaration" {
			continue
		}
		nameNode := m.ChildByFieldName("name")
		if nameNode == nil || nameNode.Kind() != "simple_identifier" {
			continue
		}
		name := nameNode.Utf8Text(ex.src)
		qualifiedName := typeName + "." + name
		id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, m)})
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		if fbody := m.ChildByFieldName("body"); fbody != nil {
			ex.collectCalls(id, fbody)
		}
	}
}

func (ex *extractor) collectTopLevelFunctions(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := root.NamedChild(i)
		if d.Kind() != "function_declaration" {
			continue
		}
		nameNode := d.ChildByFieldName("name")
		if nameNode == nil || nameNode.Kind() != "simple_identifier" {
			continue
		}
		name := nameNode.Utf8Text(ex.src)
		id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, d)})
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		if body := d.ChildByFieldName("body"); body != nil {
			ex.collectCalls(id, body)
		}
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
		Language:      "swift",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    "public",
		IsExported:    true,
	}
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
	if call.NamedChildCount() == 0 {
		return
	}
	callee := call.NamedChild(0)
	pos := call.StartPosition()
	line, col := int32(pos.Row)+1, int32(pos.Column)

	switch callee.Kind() {
	case "simple_identifier":
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: callee.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "navigation_expression":
		target := callee.ChildByFieldName("target")
		suffix := callee.ChildByFieldName("suffix")
		if target == nil || suffix == nil {
			return
		}
		memberNode := suffix.ChildByFieldName("suffix")
		if memberNode == nil || memberNode.Kind() != "simple_identifier" {
			return
		}
		name := memberNode.Utf8Text(ex.src)
		var pkgAlias string
		switch {
		case target.Kind() == "simple_identifier" && isLikelyTypeName(target.Utf8Text(ex.src)):
			// A rare PascalCase bare-name target -- same-module attempt.
		case target.Kind() == "simple_identifier":
			// A lowercase-leading target (including `self`, deliberately
			// not special-cased -- types.go) is forced through the WR-02
			// synthetic-non-matching-alias pattern.
			pkgAlias = "<local:" + target.Utf8Text(ex.src) + ">"
		default:
			pkgAlias = "<" + target.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter -- Swift's own near-universal naming convention distinguishing a
// type name (PascalCase) from a local variable/parameter/function name
// (camelCase), mirroring pyextract/rustextract/phpextract's identical
// heuristic.
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
