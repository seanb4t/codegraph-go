package kotlinextract

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

// Extract walks a single Kotlin file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-kotlin's
// external C scanner.
//
// moduleKey is the discovery-time path-based placeholder
// (languages_kotlin.go's LanguageSpec.ModuleKey). A file's own declared
// `package foo.bar` statement, when present, OVERRIDES it — mirroring
// phpextract's identical parse-time-override pattern (Kotlin's package,
// like PHP/C#'s namespace, is only knowable after parsing).
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "kotlin",
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
		result.Err = fmt.Errorf("kotlinextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("kotlinextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "kotlin",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	if pkg, ok := packageOverride(root, src); ok && pkg != "" {
		result.ImportPath = pkg
	}

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
	}

	// Imports are collected first for consistency with javaextract/
	// pyextract's ordering discipline, though this tier's calls/embeds
	// resolution does not currently consult FileResult.Imports for Kotlin.
	ex.collectImports(root)
	ex.collectTypes(root)
	ex.collectTopLevelFunctions(root)

	return result, nil
}

// packageOverride finds root's first top-level package_header and returns
// its own qualified_identifier's full dotted text verbatim.
func packageOverride(root *tree_sitter.Node, src []byte) (string, bool) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		if c.Kind() != "package_header" {
			continue
		}
		if qi := firstNamedChild(c); qi != nil && qi.Kind() == "qualified_identifier" {
			return qi.Utf8Text(src), true
		}
		return "", false
	}
	return "", false
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

// --- import ---

func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := root.NamedChild(i)
		if d.Kind() != "import" {
			continue
		}
		ex.emitImport(d)
	}
}

func (ex *extractor) emitImport(decl *tree_sitter.Node) {
	qi := firstNamedChild(decl)
	if qi == nil || qi.Kind() != "qualified_identifier" {
		return
	}
	full := qi.Utf8Text(ex.src)
	if full == "" {
		return
	}
	parts := strings.Split(full, ".")
	alias := parts[len(parts)-1]
	ex.result.Imports[alias] = full

	pos := decl.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: full, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// --- class_declaration (class/interface/enum/object) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := root.NamedChild(i)
		if d.Kind() == "class_declaration" {
			ex.emitType(d)
		}
	}
}

// kotlinDeclKind scans n's own (including anonymous) children for the
// declaration-kind keyword token ("class"/"interface"/"enum"/"object") that
// distinguishes what class_declaration actually declares -- this grammar
// carries NO field for it (types.go's documented grammar-shape adaptation).
func kotlinDeclKind(n *tree_sitter.Node) string {
	for i := uint(0); i < n.ChildCount(); i++ {
		switch n.Child(i).Kind() {
		case "class", "interface", "enum", "object":
			return n.Child(i).Kind()
		}
	}
	return ""
}

func (ex *extractor) emitType(decl *tree_sitter.Node) {
	kind := kotlinDeclKind(decl)
	var symbolKind string
	switch kind {
	case "class", "enum":
		symbolKind = goextract.KindStruct
	case "interface":
		symbolKind = goextract.KindInterface
	default:
		// "object" (or an unrecognized shape) -- documented gap.
		return
	}

	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "identifier" {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(symbolKind, name, ex.relPath)

	start := decl.StartPosition()
	end := decl.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          symbolKind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "kotlin",
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

	ex.collectDelegations(id, decl)
	if body := firstChildOfKind(decl, "class_body"); body != nil {
		ex.collectMethods(id, name, body)
	}
}

func (ex *extractor) collectDelegations(id string, decl *tree_sitter.Node) {
	specs := firstChildOfKind(decl, "delegation_specifiers")
	if specs == nil {
		return
	}
	for i := uint(0); i < specs.NamedChildCount(); i++ {
		spec := specs.NamedChild(i)
		if spec.Kind() != "delegation_specifier" {
			continue
		}
		name, ok := kotlinTypeName(spec, ex.src)
		if !ok {
			continue
		}
		pos := spec.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: id, Name: name, Kind: goextract.RefKindEmbeds,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column),
		})
	}
}

// kotlinTypeName resolves a delegation_specifier's own child down to a
// simple type name: a plain `user_type` (interface conformance), or a
// `constructor_invocation` wrapping one (an invoked superclass
// constructor) -- both verified via live parse-tree dumps.
func kotlinTypeName(n *tree_sitter.Node, src []byte) (string, bool) {
	switch n.Kind() {
	case "delegation_specifier", "constructor_invocation":
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if name, ok := kotlinTypeName(n.NamedChild(i), src); ok {
				return name, ok
			}
		}
		return "", false
	case "user_type":
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(i); c.Kind() == "identifier" {
				return c.Utf8Text(src), true
			}
		}
		return "", false
	case "identifier":
		return n.Utf8Text(src), true
	default:
		return "", false
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
		if nameNode == nil || nameNode.Kind() != "identifier" {
			continue
		}
		name := nameNode.Utf8Text(ex.src)
		qualifiedName := typeName + "." + name
		id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, m)})
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		// An interface's own method requirement has no function_body --
		// collectCalls is simply skipped (mirrors phpextract's identical
		// bodyless-interface-method handling).
		if fbody := firstChildOfKind(m, "function_body"); fbody != nil {
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
		if nameNode == nil || nameNode.Kind() != "identifier" {
			continue
		}
		name := nameNode.Utf8Text(ex.src)
		id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)
		ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, d)})
		ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
			Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
		}})
		if body := firstChildOfKind(d, "function_body"); body != nil {
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
		Language:      "kotlin",
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
	case "identifier":
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: callee.Utf8Text(ex.src), Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "navigation_expression":
		if callee.NamedChildCount() < 2 {
			return
		}
		receiver := callee.NamedChild(0)
		member := callee.NamedChild(callee.NamedChildCount() - 1)
		if member.Kind() != "identifier" {
			return
		}
		name := member.Utf8Text(ex.src)
		var pkgAlias string
		switch {
		case receiver.Kind() == "identifier" && isLikelyTypeName(receiver.Utf8Text(ex.src)):
			// A rare PascalCase bare-name receiver -- same-module attempt.
		case receiver.Kind() == "identifier":
			// A lowercase-leading receiver (including `this`, deliberately
			// not special-cased -- types.go) is forced through the WR-02
			// synthetic-non-matching-alias pattern.
			pkgAlias = "<local:" + receiver.Utf8Text(ex.src) + ">"
		default:
			pkgAlias = "<" + receiver.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter -- Kotlin's own near-universal naming convention distinguishing a
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

func firstNamedChild(n *tree_sitter.Node) *tree_sitter.Node {
	if n.NamedChildCount() == 0 {
		return nil
	}
	return n.NamedChild(0)
}

func firstChildOfKind(n *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if c := n.NamedChild(i); c.Kind() == kind {
			return c
		}
	}
	return nil
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
