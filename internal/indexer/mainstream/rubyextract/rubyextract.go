package rubyextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Extract walks a single Ruby file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-ruby's
// external heredoc C scanner.
//
// moduleKey is computed entirely at discovery time by languages_ruby.go's
// LanguageSpec.ModuleKey (an unconditional, directory-relative path
// identity) — Extract never overrides FileResult.ImportPath.
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "ruby",
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
		result.Err = fmt.Errorf("rubyextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("rubyextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "ruby",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	ex := &extractor{
		src:              src,
		relPath:          relPath,
		fileID:           fileID,
		result:           &result,
		typeNodesByStart: make(map[uint]string),
		typeNodeByName:   make(map[string]int),
		alloc:            nodeid.NewAllocator(relPath),
	}

	// Requires are collected first so a same-file class's superclass check
	// (below) can see every require_relative-populated Imports entry
	// regardless of declaration order in the source.
	ex.collectRequires(root)
	ex.collectClasses(root)
	ex.collectMethods(root)
	ex.collectTopLevelFunctions(root)

	goextract.FinalizeIDs(ex.alloc, root, &result)
	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract/pyextract's extractor shape.
type extractor struct {
	src     []byte
	relPath string
	fileID  string
	result  *goextract.FileResult
	alloc   *nodeid.Allocator

	// env is the type environment of the function body under extraction
	// (design D17), or nil outside a body.
	env *goextract.TypeEnv

	// typeNodesByStart maps the start byte of each type declaration to its
	// node id, so that each copy of a repeated type gets its own methods.
	typeNodesByStart map[uint]string

	// typeNodeByName maps a class or module name to the index in
	// result.Nodes of its node: a reopened class or module is one node
	// (design D16), whose span grows to cover every block.
	typeNodeByName map[string]int
}

// --- require / require_relative (ordinary calls in Ruby's own grammar) ---

func (ex *extractor) collectRequires(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() == "call" {
			ex.maybeEmitRequire(n)
		}
		return true
	})
}

// isRequireCall reports whether call is a receiverless
// require("...")/require_relative("...") shape, returning the recognized
// method name.
func isRequireCall(call *tree_sitter.Node, src []byte) (string, bool) {
	if call.ChildByFieldName("receiver") != nil {
		return "", false
	}
	methodNode := call.ChildByFieldName("method")
	if methodNode == nil {
		return "", false
	}
	method := methodNode.Utf8Text(src)
	if method != "require" && method != "require_relative" {
		return "", false
	}
	return method, true
}

func (ex *extractor) maybeEmitRequire(call *tree_sitter.Node) {
	method, ok := isRequireCall(call, ex.src)
	if !ok {
		return
	}
	args := call.ChildByFieldName("arguments")
	if args == nil || args.NamedChildCount() == 0 {
		return
	}
	strNode := args.NamedChild(0)
	if strNode.Kind() != "string" {
		// A dynamic/interpolated require path is not resolvable statically
		// (documented gap, see types.go).
		return
	}
	requirePath, ok := rubyStringContent(strNode, ex.src)
	if !ok {
		return
	}

	pos := call.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: requirePath, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})

	if method == "require_relative" {
		target := ex.resolveRequireRelative(requirePath)
		alias := target
		if i := strings.LastIndexByte(alias, '/'); i >= 0 {
			alias = alias[i+1:]
		}
		ex.result.Imports[alias] = target
	}
}

// resolveRequireRelative resolves requirePath against this file's own
// directory — require_relative is always directory-relative, unlike
// require's $LOAD_PATH-relative resolution (languages_ruby.go's own
// unconditional, extension-stripped-relPath ModuleKey format).
func (ex *extractor) resolveRequireRelative(requirePath string) string {
	dir := path.Dir(ex.relPath)
	target := requirePath
	if dir != "." {
		target = path.Join(dir, requirePath)
	}
	return strings.TrimSuffix(path.Clean(target), ".rb")
}

// rubyStringContent extracts a Ruby "string" node's literal, non-interpolated
// content. Returns ok=false if the string contains any interpolation (a
// dynamic require path this extractor cannot resolve statically).
func rubyStringContent(n *tree_sitter.Node, src []byte) (string, bool) {
	var sb strings.Builder
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		switch c.Kind() {
		case "string_content", "escape_sequence":
			sb.WriteString(c.Utf8Text(src))
		default:
			return "", false
		}
	}
	return sb.String(), true
}

// --- classes / modules ---

func (ex *extractor) collectClasses(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class", "module":
			ex.emitClassOrModule(n)
		}
		return true
	})
}

func (ex *extractor) emitClassOrModule(node *tree_sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := rubyConstantName(nameNode, ex.src)
	if name == "" {
		return
	}
	kind := goextract.KindStruct
	start := node.StartPosition()
	end := node.EndPosition()
	if i, reopened := ex.typeNodeByName[name]; reopened {
		// A reopened class or module adds to the one node (design D16):
		// its methods join the node, and the span grows to cover the
		// new block.
		prev := ex.result.Nodes[i].Node
		ex.typeNodesByStart[node.StartByte()] = prev.Id
		if int32(end.Row)+1 > prev.EndLine || (int32(end.Row)+1 == prev.EndLine && int32(end.Column) > prev.EndCol) {
			prev.EndLine, prev.EndCol = int32(end.Row)+1, int32(end.Column)
		}
		if node.Kind() == "class" {
			if sc := node.ChildByFieldName("superclass"); sc != nil {
				ex.emitSuperclass(prev.Id, sc)
			}
		}
		return
	}
	id := ex.alloc.Declare(kind, name, "", node.StartByte())
	ex.typeNodeByName[name] = len(ex.result.Nodes)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "ruby",
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
	ex.typeNodesByStart[node.StartByte()] = id

	if node.Kind() == "class" {
		if sc := node.ChildByFieldName("superclass"); sc != nil {
			ex.emitSuperclass(id, sc)
		}
	}
}

// rubyConstantName extracts a class/module declaration's simple name from
// either a bare "constant" node or a "scope_resolution" node (`Foo::Bar`),
// taking only the final segment in the latter case.
func rubyConstantName(n *tree_sitter.Node, src []byte) string {
	switch n.Kind() {
	case "constant":
		return n.Utf8Text(src)
	case "scope_resolution":
		if name := n.ChildByFieldName("name"); name != nil {
			return name.Utf8Text(src)
		}
	}
	return ""
}

// emitSuperclass emits a RefKindEmbeds unresolved ref for a class's
// declared superclass (the "superclass" node wraps `< Expr`; Expr is its
// sole named child).
func (ex *extractor) emitSuperclass(classID string, scNode *tree_sitter.Node) {
	if scNode.NamedChildCount() == 0 {
		return
	}
	expr := scNode.NamedChild(0)
	name := rubyConstantName(expr, ex.src)
	if name == "" {
		// A dynamic superclass expression (a local variable/method call
		// result) — not a concrete supertype reference this extractor can
		// resolve.
		return
	}
	pkgAlias := ""
	if _, imported := ex.result.Imports[name]; imported {
		pkgAlias = name
	}
	pos := scNode.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: classID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// --- methods / functions / calls ---

func (ex *extractor) collectMethods(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class", "module":
			ex.emitMethodsForScope(n)
		}
		return true
	})
}

func (ex *extractor) emitMethodsForScope(scopeNode *tree_sitter.Node) {
	nameNode := scopeNode.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	scopeName := rubyConstantName(nameNode, ex.src)
	scopeID, ok := ex.typeNodesByStart[scopeNode.StartByte()]
	if !ok {
		return
	}
	body := scopeNode.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		switch m.Kind() {
		case "method", "singleton_method":
			ex.emitScopedMethod(scopeID, scopeName, m)
		}
	}
}

func (ex *extractor) emitScopedMethod(scopeID, scopeName string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := scopeName + "." + name
	id := ex.alloc.Declare(goextract.KindMethod, qualifiedName, ex.sigKey(decl), decl.StartByte())

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: scopeID, Target: id, Kind: "contains", Provenance: "ast",
	}})

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.env = ex.bodyEnv(decl)
		ex.collectCalls(id, body)
		ex.env = nil
	}
}

// collectTopLevelFunctions walks ONLY root's direct named children (never
// nested inside a class/module) emitting a KindFunction node per top-level
// method/singleton_method found.
func (ex *extractor) collectTopLevelFunctions(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		m := root.NamedChild(i)
		switch m.Kind() {
		case "method", "singleton_method":
			ex.emitFunction(m)
		}
	}
}

func (ex *extractor) emitFunction(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := ex.alloc.Declare(goextract.KindFunction, name, ex.sigKey(decl), decl.StartByte())

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})

	if body := decl.ChildByFieldName("body"); body != nil {
		ex.env = ex.bodyEnv(decl)
		ex.collectCalls(id, body)
		ex.env = nil
	}
}

func (ex *extractor) buildFuncNode(kind, id, name, qualifiedName string, decl *tree_sitter.Node) *schema.Node {
	start := decl.StartPosition()
	end := decl.EndPosition()

	var signature string
	if params := decl.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	}

	n := &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "ruby",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Signature:     signature,
		Visibility:    "public",
		IsExported:    true,
	}
	goextract.FillSignature(n, collectParams(decl, ex.src), "")
	return n
}

func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "call" {
			if _, isRequire := isRequireCall(n, ex.src); !isRequire {
				ex.recordCall(fromID, n)
			}
		}
		return true
	})
}

func (ex *extractor) recordCall(fromID string, call *tree_sitter.Node) {
	methodNode := call.ChildByFieldName("method")
	if methodNode == nil {
		return
	}
	pos := call.StartPosition()
	line := int32(pos.Row) + 1
	col := int32(pos.Column)
	name := methodNode.Utf8Text(ex.src)
	args := ex.callArgs(ex.env, call.ChildByFieldName("arguments"))

	receiver := call.ChildByFieldName("receiver")
	if receiver == nil {
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, Kind: goextract.RefKindCalls, Line: line, Col: col,
			ArgCount: args.Count, HasSpread: args.Spread, ArgTypes: args.Types, ArgLabels: args.Labels,
		})
		return
	}

	pkgAlias, receiverType := ex.receiver(ex.env, receiver)
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		ReceiverType: receiverType, ArgCount: args.Count, HasSpread: args.Spread, ArgTypes: args.Types, ArgLabels: args.Labels,
	})
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
