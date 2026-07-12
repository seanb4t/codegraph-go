package pyextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Extract walks a single Python file's parsed syntax tree and produces its
// Pass-1 intermediate, reproducing goextract.Extract's exact skip/error
// contract (05-PATTERNS.md's "Per-file skip/error contract"): a parse
// failure or unexpected tree shape sets FileResult.Err and Extract itself
// returns a nil error, so one bad file never aborts a caller's batch. This
// is the front-line mitigation (threat T-05-DoS) for tree-sitter-python's
// INDENT/DEDENT external C scanner — parser.MaxSourceBytes is enforced by
// p.Parse BEFORE any backend-specific parsing runs (internal/parser/cgo's
// CGoParser.Parse), so an oversized file never reaches the scanner at all.
//
// Unlike javaextract/csharpextract, Python has no in-source declared
// cross-file identity (no `package`/`namespace` statement) — its dotted
// module path is entirely directory-structure-derived (RESEARCH "Don't
// Hand-Roll"), so moduleKey (computed by languages_python.go's
// LanguageSpec.ModuleKey, which already has everything it needs: the
// repo's resolved package root + relPath) is already authoritative.
// Extract never overrides FileResult.ImportPath the way Java/C# do.
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    "python",
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
		result.Err = fmt.Errorf("pyextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("pyextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      "python",
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	ex := &extractor{
		src:             src,
		relPath:         relPath,
		fileID:          fileID,
		result:          &result,
		typeNodesByName: make(map[string]string),
		ownPackage:      ownPackageFor(moduleKey, relPath),
	}

	// Imports are collected first so calls and base-class references below
	// can tell an imported/aliased simple name (routes through
	// resolveSelector via the Imports map) apart from a same-module one
	// (routes through resolveUnqualified) — mirrors javaextract/
	// csharpextract's ordering discipline.
	ex.collectImports(root)
	// Classes are collected before methods (05-PATTERNS.md's ordering
	// discipline) so a method's enclosing-class lookup (typeNodesByName)
	// always succeeds regardless of declaration order in the source.
	ex.collectClasses(root)
	ex.collectMethods(root)
	ex.collectTopLevelFunctions(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring goextract/javaextract's extractor shape.
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	result          *goextract.FileResult
	typeNodesByName map[string]string

	// ownPackage is this file's own enclosing dotted package path (its
	// moduleKey with the last segment stripped, or the moduleKey itself for
	// an __init__.py module) — the base a relative import's leading dots
	// resolve against (see resolveFromModule).
	ownPackage string
}

// ownPackageFor derives a file's enclosing dotted package from its own
// dotted moduleKey: an __init__.py module IS its own package (no stripping
// — Python's own semantics), any other module's enclosing package is its
// moduleKey with the trailing ".symbolname" segment removed. A top-level
// module with no dot in its moduleKey has no enclosing package at all (the
// dotted-path root itself).
func ownPackageFor(moduleKey, relPath string) string {
	if path.Base(relPath) == "__init__.py" {
		return moduleKey
	}
	if idx := strings.LastIndex(moduleKey, "."); idx >= 0 {
		return moduleKey[:idx]
	}
	return ""
}

// packageUp strips levels trailing dotted segments from pkg — used to walk
// a relative import's leading-dot count up the package hierarchy (one dot
// beyond the first level).
func packageUp(pkg string, levels int) string {
	for i := 0; i < levels && pkg != ""; i++ {
		if idx := strings.LastIndex(pkg, "."); idx >= 0 {
			pkg = pkg[:idx]
		} else {
			pkg = ""
		}
	}
	return pkg
}

// --- imports ---

func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		switch decl.Kind() {
		case "import_statement":
			ex.emitImportStatement(decl)
		case "import_from_statement":
			ex.emitImportFromStatement(decl)
		}
	}
}

// emitImportStatement walks a top-level `import a.b[, c.d as e]` statement's
// field('name', ...) entries — each is either a bare dotted_name or an
// aliased_import.
func (ex *extractor) emitImportStatement(decl *tree_sitter.Node) {
	for i := uint32(0); i < uint32(decl.NamedChildCount()); i++ {
		if decl.FieldNameForNamedChild(i) != "name" {
			continue
		}
		ex.emitImportName(decl.NamedChild(uint(i)))
	}
}

func (ex *extractor) emitImportName(nameEntry *tree_sitter.Node) {
	var dotted *tree_sitter.Node
	var alias string
	switch nameEntry.Kind() {
	case "dotted_name":
		dotted = nameEntry
		// A plain "import foo.bar" binds only the top-level package name
		// ("foo"), never the full dotted path — see types.go's package doc
		// for why no Imports entry is populated here.
	case "aliased_import":
		dotted = nameEntry.ChildByFieldName("name")
		if a := nameEntry.ChildByFieldName("alias"); a != nil {
			alias = a.Utf8Text(ex.src)
		}
	default:
		return
	}
	if dotted == nil {
		return
	}
	fullPath := dotted.Utf8Text(ex.src)
	if alias != "" {
		ex.result.Imports[alias] = fullPath
	}
	pos := nameEntry.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: fullPath, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// emitImportFromStatement walks a top-level `from <module> import name1[,
// name2 as alias2][, *]` statement: one RefKindImports ref for the whole
// statement (Name = the resolved "from" module dotted path), plus one
// Imports[alias-or-name] = fromModule entry per non-wildcard imported name
// (mirroring javaextract's per-import-declaration handling).
func (ex *extractor) emitImportFromStatement(decl *tree_sitter.Node) {
	moduleNode := decl.ChildByFieldName("module_name")
	if moduleNode == nil {
		return
	}
	fromModule, ok := ex.resolveFromModule(moduleNode)
	if !ok {
		return
	}

	pos := decl.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: fromModule, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})

	for i := uint32(0); i < uint32(decl.NamedChildCount()); i++ {
		c := decl.NamedChild(uint(i))
		if c.Kind() == "wildcard_import" {
			// from foo import * — no single simple name to key
			// result.Imports by (mirrors javaextract's wildcard handling);
			// the RefKindImports ref above already records the dependency.
			continue
		}
		if decl.FieldNameForNamedChild(i) != "name" {
			continue
		}
		ex.emitFromImportName(fromModule, c)
	}
}

func (ex *extractor) emitFromImportName(fromModule string, entry *tree_sitter.Node) {
	var nameNode *tree_sitter.Node
	var alias string
	switch entry.Kind() {
	case "dotted_name":
		// `from x import y` — y is grammatically a (single-segment)
		// dotted_name; `from x import y.z` is not valid Python syntax.
		nameNode = entry
	case "aliased_import":
		nameNode = entry.ChildByFieldName("name")
		if a := entry.ChildByFieldName("alias"); a != nil {
			alias = a.Utf8Text(ex.src)
		}
	default:
		return
	}
	if nameNode == nil {
		return
	}
	simple := nameNode.Utf8Text(ex.src)
	if alias == "" {
		alias = simple
	}
	ex.result.Imports[alias] = fromModule
}

// resolveFromModule resolves an import_from_statement's module_name field
// (a dotted_name for an absolute import, or a relative_import for a
// leading-dot-prefixed one) into a fully-dotted module path. A relative
// import's dot count walks up ex.ownPackage: one dot means "this file's own
// enclosing package", each additional dot strips one more trailing segment
// (RESEARCH "Don't Hand-Roll" — Python's own relative-import-dot
// semantics).
func (ex *extractor) resolveFromModule(moduleNode *tree_sitter.Node) (string, bool) {
	switch moduleNode.Kind() {
	case "dotted_name":
		return moduleNode.Utf8Text(ex.src), true
	case "relative_import":
		var prefixNode, dottedNode *tree_sitter.Node
		for i := uint(0); i < moduleNode.NamedChildCount(); i++ {
			c := moduleNode.NamedChild(i)
			switch c.Kind() {
			case "import_prefix":
				prefixNode = c
			case "dotted_name":
				dottedNode = c
			}
		}
		dots := 1
		if prefixNode != nil {
			if n := len([]rune(prefixNode.Utf8Text(ex.src))); n > 0 {
				dots = n
			}
		}
		base := packageUp(ex.ownPackage, dots-1)
		if dottedNode == nil {
			return base, true
		}
		rest := dottedNode.Utf8Text(ex.src)
		if base == "" {
			return rest, true
		}
		return base + "." + rest, true
	default:
		return "", false
	}
}

// --- classes ---

// collectClasses walks the whole tree (module-level and nested classes
// alike) emitting a KindStruct node per class_definition found — mirrors
// javaextract's collectTypes shape. A decorated_definition wrapping a
// class_definition needs no special unwrapping here: walkDescendants
// already visits the wrapped class_definition as one of the wrapper's own
// descendants.
func (ex *extractor) collectClasses(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() == "class_definition" {
			ex.emitClass(n)
		}
		return true
	})
}

func (ex *extractor) emitClass(node *tree_sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(goextract.KindStruct, name, ex.relPath)

	start := node.StartPosition()
	end := node.EndPosition()
	vis := pythonVisibility(name)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindStruct,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "python",
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

	ex.collectBaseClasses(id, node)
}

// collectBaseClasses emits one RefKindEmbeds unresolved ref per positional
// entry in a class's "superclasses" argument_list — a simple identifier or
// a single `module.Attr`-shaped attribute chain. Keyword arguments
// (`metaclass=...`) and starred entries are skipped: they are not concrete
// supertype references (RESEARCH Pattern 2 — extends/implements are not
// distinguished at parse time; that is Wave 6's job, out of this plan's
// scope).
func (ex *extractor) collectBaseClasses(classID string, classNode *tree_sitter.Node) {
	superWrap := classNode.ChildByFieldName("superclasses")
	if superWrap == nil {
		return
	}
	for i := uint(0); i < superWrap.NamedChildCount(); i++ {
		arg := superWrap.NamedChild(i)
		switch arg.Kind() {
		case "identifier":
			ex.emitBaseClassRef(classID, arg, arg.Utf8Text(ex.src), "")
		case "attribute":
			field := arg.ChildByFieldName("attribute")
			object := arg.ChildByFieldName("object")
			if field == nil {
				continue
			}
			var pkgAliasCandidate string
			if object != nil && object.Kind() == "identifier" {
				pkgAliasCandidate = object.Utf8Text(ex.src)
			}
			ex.emitBaseClassRef(classID, arg, field.Utf8Text(ex.src), pkgAliasCandidate)
		default:
			// keyword_argument (metaclass=...), list_splat (*bases),
			// dictionary_splat (**kwargs), or any other non-type-reference
			// shape — not a concrete base-class reference, skipped.
		}
	}
}

func (ex *extractor) emitBaseClassRef(classID string, node *tree_sitter.Node, name, pkgAliasCandidate string) {
	pkgAlias := ""
	if pkgAliasCandidate != "" {
		if _, imported := ex.result.Imports[pkgAliasCandidate]; imported {
			pkgAlias = pkgAliasCandidate
		}
		// else: not a genuine import alias — fall through to an unqualified
		// same-module reference by simple name, mirroring javaextract's own
		// emitSupertypeRef tolerance for a same-package supertype needing
		// no import.
	}
	pos := node.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: classID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// pythonVisibility applies Python's own naming-convention visibility rule:
// a leading underscore ("_x", "__x") is the language's own private/internal
// convention (no enforced access control exists); no leading underscore is
// public.
func pythonVisibility(name string) string {
	if strings.HasPrefix(name, "_") {
		return "private"
	}
	return "public"
}

// --- methods / functions / calls ---

// collectMethods walks the whole tree; for every class_definition found, it
// processes that class's OWN direct body children — mirrors javaextract's
// collectMethods shape.
func (ex *extractor) collectMethods(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() == "class_definition" {
			ex.emitMethodsForClass(n)
		}
		return true
	})
}

func (ex *extractor) emitMethodsForClass(classNode *tree_sitter.Node) {
	nameNode := classNode.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nameNode.Utf8Text(ex.src)
	classID, ok := ex.typeNodesByName[className]
	if !ok {
		// collectClasses (pass 1) skipped this declaration (e.g. a missing
		// name node) — nothing to attach methods to.
		return
	}
	body := classNode.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		fn := unwrapDecorated(body.NamedChild(i))
		if fn.Kind() == "function_definition" {
			ex.emitMethod(classID, className, fn)
		}
	}
}

// collectTopLevelFunctions walks ONLY root's direct named children (never
// nested inside a class or another function) emitting a KindFunction node
// per module-level function_definition — mirrors goextract's
// collectFuncsAndMethods top-level-only scope. A function nested inside
// another function is not extracted as its own symbol, an accepted scope
// boundary this project's Go/Java/C# extractors share.
func (ex *extractor) collectTopLevelFunctions(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		fn := unwrapDecorated(root.NamedChild(i))
		if fn.Kind() == "function_definition" {
			ex.emitFunction(fn)
		}
	}
}

// unwrapDecorated returns n's wrapped "definition" field when n is a
// decorated_definition, or n itself otherwise — decorators are not
// extracted as edges (out of scope for this plan).
func unwrapDecorated(n *tree_sitter.Node) *tree_sitter.Node {
	if n.Kind() != "decorated_definition" {
		return n
	}
	if d := n.ChildByFieldName("definition"); d != nil {
		return d
	}
	return n
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

func (ex *extractor) emitMethod(classID, className string, decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := className + "." + name
	id := nodeid.NodeID(goextract.KindMethod, qualifiedName, ex.relPath)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl)})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: classID, Target: id, Kind: "contains", Provenance: "ast",
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
	vis := pythonVisibility(name)

	return &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "python",
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

func (ex *extractor) collectCalls(fromID string, body *tree_sitter.Node) {
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "call" {
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
	case "attribute":
		object := fn.ChildByFieldName("object")
		field := fn.ChildByFieldName("attribute")
		if object == nil || field == nil {
			return
		}
		var pkgAlias string
		switch {
		case object.Kind() == "identifier":
			ident := object.Utf8Text(ex.src)
			_, isImport := ex.result.Imports[ident]
			switch {
			case ident == "self" || ident == "cls":
				// self.method()/cls.method() — implicit same-class call,
				// resolved same-module via an empty PkgAlias
				// (resolveUnqualified), mirroring Java's `this.method()`.
			case isImport:
				// A real import alias — routes through resolveSelector to
				// the imported module's own declaring moduleKey.
				pkgAlias = ident
			case isLikelyTypeName(ident):
				// An uppercase-leading identifier that is NOT a real import
				// is, by Python naming convention, very likely a
				// same-module class reference (a class defined earlier in
				// this same file) — an empty PkgAlias correctly routes
				// through resolveUnqualified (RESEARCH Pitfall 3's
				// declared-import ambiguity, resolved here by convention,
				// mirroring javaextract/csharpextract).
			default:
				// A lowercase-leading identifier is very likely a local
				// variable/parameter/module receiver this extractor tracks
				// no type for — force it through resolveSelector's
				// alias-membership boundary via a synthetic non-matching
				// alias so it deterministically falls through to
				// "unresolved" instead of risking a same-module false
				// match (mirrors goextract's WR-02 fix).
				pkgAlias = "<local:" + ident + ">"
			}
		default:
			// A non-identifier operand (another call, a chained attribute,
			// a subscript, ...) can never be a real import alias — same
			// synthetic-alias treatment as the local-variable case above.
			pkgAlias = "<" + object.Kind() + ">"
		}
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: field.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls,
			Line: line, Col: col,
		})
	}
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — the near-universal Python convention (PEP 8) distinguishing a
// class name (PascalCase) from a local variable/parameter/function name
// (snake_case), mirroring javaextract/csharpextract's identical heuristic.
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract/javaextract's own
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
