package rustextract

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

// Extract walks a single Rust file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-rust's
// external raw-string C scanner — parser.MaxSourceBytes is enforced by
// p.Parse BEFORE any backend-specific parsing runs.
//
// moduleKey is computed entirely at discovery time by languages_rust.go's
// LanguageSpec.ModuleKey (crate name + Cargo-convention module path) —
// Extract never overrides FileResult.ImportPath.
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "rust",
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
		result.Err = fmt.Errorf("rustextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("rustextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "rust",
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

	// Types are collected first so an impl block's receiver-type lookup
	// (below) always sees every struct/enum this file declares, regardless
	// of declaration order in the source (mirrors goextract/pyextract's
	// ordering discipline).
	ex.collectTypes(root)
	ex.collectUses(root)
	ex.collectImpls(root)
	ex.collectFunctions(root)

	return result, nil
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

// --- struct / enum / trait declarations (top-level only) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		switch decl.Kind() {
		case "struct_item":
			ex.emitTypeItem(goextract.KindStruct, decl)
		case "enum_item":
			ex.emitTypeItem(goextract.KindStruct, decl)
		case "trait_item":
			ex.emitTypeItem(goextract.KindInterface, decl)
		}
	}
}

func (ex *extractor) emitTypeItem(kind string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(kind, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	vis := rustVisibility(decl)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "rust",
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
}

// rustVisibility reports "public" when decl carries a visibility_modifier
// child (pub, pub(crate), pub(super), ...), "private" otherwise — Rust's
// own default-private-unless-pub rule.
func rustVisibility(decl *tree_sitter.Node) string {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		if decl.NamedChild(i).Kind() == "visibility_modifier" {
			return "public"
		}
	}
	return "private"
}

// --- use declarations (top-level only) ---

func (ex *extractor) collectUses(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() != "use_declaration" {
			continue
		}
		arg := decl.ChildByFieldName("argument")
		if arg == nil {
			continue
		}
		pos := decl.StartPosition()
		ex.emitUseTree("", arg, int32(pos.Row)+1, int32(pos.Column))
	}
}

// emitUseTree walks one use_declaration's argument tree — which may nest
// via use_list/scoped_use_list for `use foo::{bar, baz}` grouped imports —
// emitting one RefKindImports ref per resolved leaf path. See types.go for
// why this never populates FileResult.Imports.
func (ex *extractor) emitUseTree(prefix string, node *tree_sitter.Node, line, col int32) {
	switch node.Kind() {
	case "identifier", "crate", "self", "super", "metavariable":
		ex.emitImportRef(joinRustPath(prefix, node.Utf8Text(ex.src)), line, col)
	case "scoped_identifier":
		ex.emitImportRef(joinRustPath(prefix, node.Utf8Text(ex.src)), line, col)
	case "use_as_clause":
		if p := node.ChildByFieldName("path"); p != nil {
			ex.emitImportRef(joinRustPath(prefix, p.Utf8Text(ex.src)), line, col)
		}
	case "use_wildcard":
		base := prefix
		for i := uint(0); i < node.NamedChildCount(); i++ {
			base = joinRustPath(prefix, node.NamedChild(i).Utf8Text(ex.src))
		}
		ex.emitImportRef(base, line, col)
	case "use_list":
		for i := uint(0); i < node.NamedChildCount(); i++ {
			ex.emitUseTree(prefix, node.NamedChild(i), line, col)
		}
	case "scoped_use_list":
		newPrefix := prefix
		if p := node.ChildByFieldName("path"); p != nil {
			newPrefix = joinRustPath(prefix, p.Utf8Text(ex.src))
		}
		if l := node.ChildByFieldName("list"); l != nil {
			ex.emitUseTree(newPrefix, l, line, col)
		}
	}
}

func (ex *extractor) emitImportRef(path string, line, col int32) {
	if path == "" {
		return
	}
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: path, Kind: goextract.RefKindImports, Line: line, Col: col,
	})
}

func joinRustPath(prefix, seg string) string {
	if prefix == "" {
		return seg
	}
	return prefix + "::" + seg
}

// --- impl blocks (top-level only) ---

func (ex *extractor) collectImpls(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() != "impl_item" {
			continue
		}
		ex.emitImpl(decl)
	}
}

func (ex *extractor) emitImpl(impl *tree_sitter.Node) {
	typeNode := impl.ChildByFieldName("type")
	if typeNode == nil {
		return
	}
	implType, ok := rustTypeName(typeNode, ex.src)
	if !ok {
		return
	}

	if traitNode := impl.ChildByFieldName("trait"); traitNode != nil {
		if traitName, ok := rustTypeName(traitNode, ex.src); ok {
			if implID, found := ex.typeNodesByName[implType]; found {
				pos := traitNode.StartPosition()
				ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
					FromID: implID, Name: traitName, Kind: goextract.RefKindEmbeds,
					Line: int32(pos.Row) + 1, Col: int32(pos.Column),
				})
			}
		}
	}

	body := impl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		fn := body.NamedChild(i)
		if fn.Kind() == "function_item" {
			ex.emitMethod(implType, fn)
		}
	}
}

// rustTypeName extracts a simple type-reference name from the shapes a
// struct/enum/trait reference can take in an impl block's "type"/"trait"
// fields: a bare type_identifier, or the innermost name of a
// generic_type/scoped_type_identifier/scoped_identifier wrapper.
func rustTypeName(n *tree_sitter.Node, src []byte) (string, bool) {
	switch n.Kind() {
	case "type_identifier":
		return n.Utf8Text(src), true
	case "generic_type":
		if t := n.ChildByFieldName("type"); t != nil {
			return rustTypeName(t, src)
		}
		return "", false
	case "scoped_type_identifier", "scoped_identifier":
		if name := n.ChildByFieldName("name"); name != nil {
			return name.Utf8Text(src), true
		}
		return "", false
	default:
		return "", false
	}
}

func (ex *extractor) emitMethod(implType string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := implType + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl)})

	if implID, found := ex.typeNodesByName[implType]; found {
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: implID, Target: id, Kind: "contains", Provenance: "ast",
		}})
	} else {
		// The impl'd type lives in a different file (or was not otherwise
		// extractable) — Pass 2 resolves this once it has a global,
		// cross-file symbol index, mirroring goextract's own cross-file
		// receiver-type handling.
		pos := decl.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: id, Name: implType, Kind: goextract.RefKindContains,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

// --- top-level free functions ---

func (ex *extractor) collectFunctions(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() == "function_item" {
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
	vis := rustVisibility(decl)

	return &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "rust",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Signature:     signature,
		ReturnType:    returnType,
		Visibility:    vis,
		IsExported:    vis == "public",
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
	fn := call.ChildByFieldName("function")
	if fn == nil {
		return
	}
	pos := call.StartPosition()
	line := int32(pos.Row) + 1
	col := int32(pos.Column)

	switch fn.Kind() {
	case "identifier":
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: fn.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "field_expression":
		field := fn.ChildByFieldName("field")
		value := fn.ChildByFieldName("value")
		if field == nil {
			return
		}
		var pkgAlias string
		switch {
		case value == nil:
		case value.Kind() == "identifier" && isLikelyTypeName(value.Utf8Text(ex.src)):
			// A PascalCase-leading receiver that is NOT a real, tracked
			// local variable is, by Rust's own naming convention, very
			// likely a same-module type reference (rare shape — Rust's
			// idiomatic method calls almost always go through
			// scoped_identifier's Type::method() form, but this covers a
			// PascalCase-named binding holding a same-module value too).
		case value.Kind() == "identifier":
			// A lowercase-leading identifier is very likely a local
			// variable/parameter this extractor tracks no type for — force
			// it through resolveSelector's alias-membership boundary via a
			// synthetic non-matching alias (mirrors goextract's WR-02 fix).
			pkgAlias = "<local:" + value.Utf8Text(ex.src) + ">"
		default:
			pkgAlias = "<" + value.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "scoped_identifier":
		nameNode := fn.ChildByFieldName("name")
		if nameNode == nil {
			return
		}
		var pkgAlias string
		if pathNode := fn.ChildByFieldName("path"); pathNode != nil {
			switch {
			case pathNode.Kind() == "self", pathNode.Kind() == "crate":
				// Self::method() / crate::method() — treated as a
				// same-module attempt, empty PkgAlias.
			case pathNode.Kind() == "identifier" && isLikelyTypeName(pathNode.Utf8Text(ex.src)):
				// Type::associated_fn() — the dominant Rust idiom for a
				// same-module associated-function/constructor call.
			default:
				pkgAlias = "<" + pathNode.Kind() + ">"
			}
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: nameNode.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — Rust's own near-universal convention (RFC 430) distinguishing a
// type/trait name (PascalCase) from a local variable/parameter/function
// name (snake_case), mirroring pyextract/javaextract's identical heuristic.
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
