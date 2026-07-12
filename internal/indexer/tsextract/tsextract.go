package tsextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// knownExtensions are the file extensions this extractor's specifier
// resolution recognizes and strips, across all three registered grammars.
var knownExtensions = []string{".tsx", ".ts", ".jsx", ".mjs", ".cjs", ".js"}

// Extract walks a single TypeScript, TSX, or JavaScript file's parsed
// syntax tree (the correct grammar was already selected by extract.go's
// per-worker, per-language-ID parser cache — this function does not itself
// select a grammar) and produces its Pass-1 intermediate, reproducing
// goextract.Extract's exact skip/error contract: a parse failure or
// unexpected tree shape sets FileResult.Err and Extract itself returns a
// nil error, so one bad file never aborts a caller's batch.
//
// moduleKey is languages_typescript.go's LanguageSpec.ModuleKey result for
// this file — always NormalizeModuleKey(relPath), the extension-stripped,
// slash-separated, repo-root-relative identity (types.go's package doc,
// "Cross-file module resolution").
func Extract(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error) {
	sum := sha256.Sum256(src)
	lang := languageForPath(relPath)
	result := goextract.FileResult{
		ImportPath:  moduleKey,
		RelPath:     relPath,
		Language:    lang,
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
		result.Err = fmt.Errorf("tsextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	if root == nil {
		result.Err = fmt.Errorf("tsextract: parser returned an empty tree for %s", relPath)
		return result, nil
	}

	fileID := nodeid.NodeID(goextract.KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            fileID,
		Kind:          goextract.KindFile,
		Name:          relPath,
		QualifiedName: relPath,
		FilePath:      relPath,
		Language:      lang,
		StartLine:     1,
		EndLine:       int32(root.EndPosition().Row) + 1,
	}})

	ex := &extractor{
		src:               src,
		relPath:           relPath,
		fileID:            fileID,
		language:          lang,
		result:            &result,
		typeNodesByName:   make(map[string]string),
		namedImportOrigin: make(map[string]string),
	}

	// Imports are collected first so every subsequent phase (types,
	// functions, methods — all of which may reference an imported name in
	// a heritage clause or call) sees a fully populated Imports/
	// namedImportOrigin table regardless of declaration order in the
	// source (ES imports are hoisted at the language level; this extractor
	// mirrors that by ordering its own collection the same way).
	ex.collectImports(root)
	ex.collectTypes(root)
	ex.collectMethods(root)
	ex.collectFunctions(root)
	ex.collectExportedConsts(root)

	return result, nil
}

// extractor carries the per-file state threaded through the tree-walk
// helpers below, mirroring csharpextract.extractor's shape plus the
// TS/JS-specific namedImportOrigin table (types.go's package doc, "Named-
// import call/heritage resolution").
type extractor struct {
	src             []byte
	relPath         string
	fileID          string
	language        string
	result          *goextract.FileResult
	typeNodesByName map[string]string

	// namedImportOrigin maps a default/named import's LOCAL alias to the
	// target module's OWN declared symbol name — see types.go's package
	// doc. Populated only for default and named imports; a namespace
	// import needs no entry (always referenced via member access, which
	// resolves through the ordinary Imports-membership check).
	namedImportOrigin map[string]string
}

// languageForPath derives the LanguageSpec ID this file was registered
// under purely from its extension — Extract's shared signature does not
// receive the ID extract.go's worker pool already dispatched on, so this
// re-derives it deterministically from the one thing Extract does receive.
func languageForPath(relPath string) string {
	switch {
	case strings.HasSuffix(relPath, ".tsx"):
		return "tsx"
	case strings.HasSuffix(relPath, ".ts"):
		return "typescript"
	default:
		return "javascript"
	}
}

// NormalizeModuleKey strips a recognized TS/JS extension from p, if
// present — the canonical per-file identity languages_typescript.go's
// ModuleKey computes for every discovered file, and the SAME normalization
// resolveModuleSpecifier applies to a resolved (relative or paths-aliased)
// import target, so the two independently-computed values always match by
// construction (types.go's package doc, "Cross-file module resolution").
// Exported so languages_typescript.go (a different package) can share this
// single source of truth rather than risking drift from a duplicated copy.
func NormalizeModuleKey(p string) string {
	for _, ext := range knownExtensions {
		if strings.HasSuffix(p, ext) {
			return strings.TrimSuffix(p, ext)
		}
	}
	return p
}

// --- module specifier resolution ---

// resolveModuleSpecifier resolves specifier (as written in an import/
// re-export-from statement in the file at relPath) into the target file's
// canonical moduleKey, per types.go's documented resolution order. ok is
// false for a specifier this extractor cannot resolve to an intra-repo
// file (an external/node_modules package specifier).
func resolveModuleSpecifier(relPath, specifier string) (target string, ok bool) {
	if strings.HasPrefix(specifier, "./") || strings.HasPrefix(specifier, "../") {
		dir := path.Dir(relPath)
		joined := path.Join(dir, specifier)
		return NormalizeModuleKey(joined), true
	}

	cfg := getConfig()
	if rewritten, matched := resolvePathsAlias(cfg.Paths, specifier); matched {
		joined := rewritten
		if cfg.BaseURL != "" && cfg.BaseURL != "." {
			joined = path.Join(cfg.BaseURL, rewritten)
		}
		return NormalizeModuleKey(joined), true
	}

	if cfg.BaseURL != "" && cfg.BaseURL != "." {
		return NormalizeModuleKey(path.Join(cfg.BaseURL, specifier)), true
	}

	return "", false
}

// resolvePathsAlias attempts to match specifier against paths (tsconfig.
// json's compilerOptions.paths). Pattern keys are consulted in SORTED
// order — never Go's own nondeterministic map-iteration order — so a
// specifier matching more than one configured pattern always picks the
// same winner across runs (determinism is load-bearing project-wide).
func resolvePathsAlias(paths map[string][]string, specifier string) (string, bool) {
	if len(paths) == 0 {
		return "", false
	}
	patterns := make([]string, 0, len(paths))
	for k := range paths {
		patterns = append(patterns, k)
	}
	sort.Strings(patterns)

	for _, pattern := range patterns {
		targets := paths[pattern]
		if len(targets) == 0 {
			continue
		}
		target := targets[0]
		if !strings.Contains(pattern, "*") {
			if specifier == pattern {
				return target, true
			}
			continue
		}
		prefix, suffix, _ := strings.Cut(pattern, "*")
		if !strings.HasPrefix(specifier, prefix) || !strings.HasSuffix(specifier, suffix) {
			continue
		}
		matched := strings.TrimSuffix(strings.TrimPrefix(specifier, prefix), suffix)
		return strings.Replace(target, "*", matched, 1), true
	}
	return "", false
}

// --- imports ---

func (ex *extractor) collectImports(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		c := root.NamedChild(i)
		switch c.Kind() {
		case "import_statement":
			ex.emitImportStatement(c)
		case "export_statement":
			if c.ChildByFieldName("source") != nil {
				ex.emitReexportStatement(c)
			}
		}
	}
}

func (ex *extractor) emitImportStatement(stmt *tree_sitter.Node) {
	sourceNode := stmt.ChildByFieldName("source")
	if sourceNode == nil {
		return
	}
	specifier := unquoteJSString(sourceNode.Utf8Text(ex.src))
	pos := stmt.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: specifier, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})

	target, resolvable := resolveModuleSpecifier(ex.relPath, specifier)
	if !resolvable {
		return
	}

	var clause *tree_sitter.Node
	for i := uint(0); i < stmt.NamedChildCount(); i++ {
		if c := stmt.NamedChild(i); c.Kind() == "import_clause" {
			clause = c
			break
		}
	}
	if clause == nil {
		// A side-effect-only import (`import './styles.css'`) — the
		// RefKindImports dependency ref above already records it.
		return
	}
	ex.bindImportClause(clause, target)
}

func (ex *extractor) bindImportClause(clause *tree_sitter.Node, target string) {
	for i := uint(0); i < clause.NamedChildCount(); i++ {
		c := clause.NamedChild(i)
		switch c.Kind() {
		case "identifier":
			// Default import (`import Foo from './foo'`) — see types.go's
			// documented default-import gap: origin == local, resolving
			// correctly only when the local binding coincides with the
			// target's own declared name.
			local := c.Utf8Text(ex.src)
			ex.result.Imports[local] = target
			ex.namedImportOrigin[local] = local
		case "namespace_import":
			if c.NamedChildCount() == 0 {
				continue
			}
			local := c.NamedChild(0).Utf8Text(ex.src)
			ex.result.Imports[local] = target
		case "named_imports":
			ex.bindNamedImports(c, target)
		}
	}
}

func (ex *extractor) bindNamedImports(namedImports *tree_sitter.Node, target string) {
	for j := uint(0); j < namedImports.NamedChildCount(); j++ {
		spec := namedImports.NamedChild(j)
		if spec.Kind() != "import_specifier" {
			continue
		}
		nameNode := spec.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		origin := nameNode.Utf8Text(ex.src)
		local := origin
		if aliasNode := spec.ChildByFieldName("alias"); aliasNode != nil {
			local = aliasNode.Utf8Text(ex.src)
		}
		ex.result.Imports[local] = target
		ex.namedImportOrigin[local] = origin
	}
}

// emitReexportStatement handles `export { X } from '...'` / `export * from
// '...'` — an export_statement carrying a "source" field. Its own imported
// names populate Imports/namedImportOrigin identically to a regular named
// import (a re-export's local binding IS, briefly, in this file's own
// scope), so any same-file usage of X also resolves correctly.
func (ex *extractor) emitReexportStatement(stmt *tree_sitter.Node) {
	sourceNode := stmt.ChildByFieldName("source")
	if sourceNode == nil {
		return
	}
	specifier := unquoteJSString(sourceNode.Utf8Text(ex.src))
	pos := stmt.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: ex.fileID, Name: specifier, Kind: goextract.RefKindImports,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})

	target, resolvable := resolveModuleSpecifier(ex.relPath, specifier)
	if !resolvable {
		return
	}

	for i := uint(0); i < stmt.NamedChildCount(); i++ {
		c := stmt.NamedChild(i)
		if c.Kind() != "export_clause" {
			continue
		}
		for j := uint(0); j < c.NamedChildCount(); j++ {
			spec := c.NamedChild(j)
			if spec.Kind() != "export_specifier" {
				continue
			}
			nameNode := spec.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			origin := nameNode.Utf8Text(ex.src)
			local := origin
			if aliasNode := spec.ChildByFieldName("alias"); aliasNode != nil {
				local = aliasNode.Utf8Text(ex.src)
			}
			ex.result.Imports[local] = target
			ex.namedImportOrigin[local] = origin
		}
	}
}

func unquoteJSString(text string) string {
	if len(text) >= 2 {
		return text[1 : len(text)-1]
	}
	return text
}

// --- type declarations (class / interface / type alias) ---

func (ex *extractor) collectTypes(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration", "abstract_class_declaration":
			ex.emitTypeDecl(n, goextract.KindStruct)
		case "interface_declaration":
			ex.emitTypeDecl(n, goextract.KindInterface)
		case "type_alias_declaration":
			ex.emitTypeAlias(n)
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
	exported := isDirectlyExported(node)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    visibilityFor(exported),
		IsExported:    exported,
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	ex.typeNodesByName[name] = id

	ex.collectSupertypeRefs(id, node)
}

func (ex *extractor) emitTypeAlias(node *tree_sitter.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(goextract.KindTypeAlias, name, ex.relPath)
	start := node.StartPosition()
	end := node.EndPosition()
	exported := isDirectlyExported(node)

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
		Visibility:    visibilityFor(exported),
		IsExported:    exported,
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
}

// collectSupertypeRefs emits one RefKindEmbeds unresolved ref per listed
// supertype in node's class_heritage (class_declaration/
// abstract_class_declaration) or extends_type_clause (interface_declaration)
// child — RESEARCH Pattern 2: extends and implements are NOT distinguished
// at parse time.
func (ex *extractor) collectSupertypeRefs(fromID string, node *tree_sitter.Node) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		c := node.NamedChild(i)
		switch c.Kind() {
		case "class_heritage":
			ex.collectClassHeritageRefs(fromID, c)
		case "extends_type_clause":
			for j := uint(0); j < c.NamedChildCount(); j++ {
				ex.emitSupertypeRef(fromID, c.NamedChild(j))
			}
		}
	}
}

// collectClassHeritageRefs handles class_heritage's two distinct shapes
// (types.go's package doc): the JavaScript grammar wraps a single, direct
// expression child (`extends Base` — JS has no `implements` keyword at
// all); the TypeScript/TSX grammar wraps extends_clause and/or
// implements_clause children instead. Detecting which shape is present is
// a simple child-kind check — no language-ID parameter needed.
func (ex *extractor) collectClassHeritageRefs(fromID string, heritage *tree_sitter.Node) {
	sawTSClause := false
	for i := uint(0); i < heritage.NamedChildCount(); i++ {
		c := heritage.NamedChild(i)
		switch c.Kind() {
		case "extends_clause", "implements_clause":
			sawTSClause = true
			for j := uint(0); j < c.NamedChildCount(); j++ {
				ex.emitSupertypeRef(fromID, c.NamedChild(j))
			}
		}
	}
	if sawTSClause {
		return
	}
	// JavaScript shape: class_heritage's only named child IS the base
	// expression itself.
	if heritage.NamedChildCount() > 0 {
		ex.emitSupertypeRef(fromID, heritage.NamedChild(0))
	}
}

func (ex *extractor) emitSupertypeRef(fromID string, t *tree_sitter.Node) {
	var name, pkgAlias string
	switch t.Kind() {
	case "member_expression":
		propNode := t.ChildByFieldName("property")
		if propNode == nil {
			return
		}
		name = propNode.Utf8Text(ex.src)
		pkgAlias = ex.memberAccessAlias(t.ChildByFieldName("object"))
	default:
		n, ok := typeRefFromExpr(t, ex.src)
		if !ok {
			return
		}
		pkgAlias, name = ex.resolveBareIdentifier(n)
	}
	pos := t.StartPosition()
	ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
		FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindEmbeds,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column),
	})
}

// typeRefFromExpr extracts a simple type-reference name from the shapes a
// heritage-clause/extends_type_clause entry can take: a plain identifier,
// a TS type_identifier, a generic_type (unwrapped to its own base name,
// ignoring type arguments), or a nested_type_identifier (its own "name"
// field — the rightmost segment; the qualifying "module" segment is
// deliberately NOT consulted here, since a nested_type_identifier's
// TYPE-position qualifier is a namespace path this bounded extractor does
// not resolve — see types.go's documented `implements ns.Foo` scope; the
// member_expression EXPRESSION-position case a plain `extends` target can
// also take is handled separately by emitSupertypeRef's own switch, via
// memberAccessAlias).
func typeRefFromExpr(t *tree_sitter.Node, src []byte) (name string, ok bool) {
	switch t.Kind() {
	case "identifier", "type_identifier":
		return t.Utf8Text(src), true
	case "generic_type":
		if nameField := t.ChildByFieldName("name"); nameField != nil {
			return typeRefFromExpr(nameField, src)
		}
		return "", false
	case "nested_type_identifier":
		if nameField := t.ChildByFieldName("name"); nameField != nil {
			return nameField.Utf8Text(src), true
		}
		return "", false
	default:
		return "", false
	}
}

// --- functions ---

func (ex *extractor) collectFunctions(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "function_declaration", "generator_function_declaration":
			ex.emitFunction(n)
		}
		return true
	})
}

func (ex *extractor) emitFunction(decl *tree_sitter.Node) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Utf8Text(ex.src)
	id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)
	exported := isDirectlyExported(decl)

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{
		Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, decl, decl, exported),
	})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

// --- exported const declarations ---

func (ex *extractor) collectExportedConsts(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		stmt := root.NamedChild(i)
		if stmt.Kind() != "export_statement" {
			continue
		}
		decl := stmt.ChildByFieldName("declaration")
		if decl == nil || decl.Kind() != "lexical_declaration" {
			continue
		}
		kindNode := decl.ChildByFieldName("kind")
		if kindNode == nil || kindNode.Utf8Text(ex.src) != "const" {
			continue
		}
		for j := uint(0); j < decl.NamedChildCount(); j++ {
			d := decl.NamedChild(j)
			if d.Kind() == "variable_declarator" {
				ex.emitExportedConstDeclarator(d)
			}
		}
	}
}

func (ex *extractor) emitExportedConstDeclarator(d *tree_sitter.Node) {
	nameNode := d.ChildByFieldName("name")
	if nameNode == nil || nameNode.Kind() != "identifier" {
		// A destructuring pattern target (`export const { a, b } = obj;`)
		// — out of this bounded extractor's scope.
		return
	}
	name := nameNode.Utf8Text(ex.src)
	valueNode := d.ChildByFieldName("value")

	if valueNode != nil {
		switch valueNode.Kind() {
		case "arrow_function", "function_expression", "generator_function":
			id := nodeid.NodeID(goextract.KindFunction, name, ex.relPath)
			ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{
				Node: ex.buildFuncNode(goextract.KindFunction, id, name, name, d, valueNode, true),
			})
			ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
				Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
			}})
			if body := valueNode.ChildByFieldName("body"); body != nil {
				ex.collectCalls(id, body)
			}
			return
		}
	}

	id := nodeid.NodeID(goextract.KindConstant, name, ex.relPath)
	start := d.StartPosition()
	end := d.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindConstant,
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

// buildFuncNode builds a function/method schema.Node. posNode supplies the
// node's own StartLine/EndLine/StartCol/EndCol span; sigNode supplies its
// parameters/return_type (the same node for an ordinary function_
// declaration/method_definition, but DELIBERATELY DIFFERENT for a `export
// const NAME = (...) => {...}` — posNode is the variable_declarator (so the
// recorded span covers the whole `NAME = (...) => {...}` text) while
// sigNode is the arrow_function/function_expression itself (which alone
// carries the parameters/return_type fields)).
func (ex *extractor) buildFuncNode(kind, id, name, qualifiedName string, posNode, sigNode *tree_sitter.Node, exported bool) *schema.Node {
	start := posNode.StartPosition()
	end := posNode.EndPosition()

	var signature, returnType string
	if params := sigNode.ChildByFieldName("parameters"); params != nil {
		signature = params.Utf8Text(ex.src)
	} else if param := sigNode.ChildByFieldName("parameter"); param != nil {
		// arrow_function's single-unparenthesized-parameter shape
		// (`x => x * 2`).
		signature = param.Utf8Text(ex.src)
	}
	if rt := sigNode.ChildByFieldName("return_type"); rt != nil {
		returnType = rt.Utf8Text(ex.src)
	}

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
		Signature:     signature,
		ReturnType:    returnType,
		Visibility:    visibilityFor(exported),
		IsExported:    exported,
	}
}

// --- methods ---

func (ex *extractor) collectMethods(root *tree_sitter.Node) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "class_declaration", "abstract_class_declaration":
			ex.emitMethodsForType(n)
		}
		return true
	})
}

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
		m := body.NamedChild(i)
		if m.Kind() == "method_definition" {
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

	vis := methodVisibility(decl, ex.src)
	node := ex.buildFuncNode(goextract.KindMethod, id, name, qualifiedName, decl, decl, vis == "public")
	node.Visibility = vis

	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: node})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: typeID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	if body := decl.ChildByFieldName("body"); body != nil {
		ex.collectCalls(id, body)
	}
}

// methodVisibility scans decl's direct named children for a TS
// accessibility_modifier (public/private/protected) token, defaulting to
// "public" — TS/JS class members with no explicit modifier are public by
// convention (unlike C#'s "private" member default).
func methodVisibility(decl *tree_sitter.Node, src []byte) string {
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		if c := decl.NamedChild(i); c.Kind() == "accessibility_modifier" {
			return c.Utf8Text(src)
		}
	}
	return "public"
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
		pkgAlias, name := ex.resolveBareIdentifier(fn.Utf8Text(ex.src))
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	case "member_expression":
		propNode := fn.ChildByFieldName("property")
		if propNode == nil {
			return
		}
		pkgAlias := ex.memberAccessAlias(fn.ChildByFieldName("object"))
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: propNode.Utf8Text(ex.src), PkgAlias: pkgAlias, Kind: goextract.RefKindCalls, Line: line, Col: col,
		})
	}
}

// resolveBareIdentifier returns the (pkgAlias, name) pair a bare identifier
// reference (a call's callee, or a heritage-clause's bare base-type name)
// carries — see types.go's package doc, "Named-import call/heritage
// resolution".
func (ex *extractor) resolveBareIdentifier(ident string) (pkgAlias, name string) {
	if orig, ok := ex.namedImportOrigin[ident]; ok {
		return ident, orig
	}
	return "", ident
}

// memberAccessAlias resolves a member_expression call/heritage reference's
// PkgAlias from its "object" operand — see types.go's package doc,
// "Named-import call/heritage resolution", for the full disambiguation
// rationale (real import alias -> qualified; PascalCase non-import ->
// same-module attempt; camelCase non-import -> forced-unresolved local
// variable, mirroring goextract's WR-02 fix).
func (ex *extractor) memberAccessAlias(object *tree_sitter.Node) string {
	if object == nil {
		return ""
	}
	switch object.Kind() {
	case "this", "super":
		return ""
	case "identifier":
		name := object.Utf8Text(ex.src)
		if _, ok := ex.result.Imports[name]; ok {
			return name
		}
		if isLikelyTypeName(name) {
			return ""
		}
		return "<local:" + name + ">"
	default:
		// A non-identifier operand (another call_expression, a
		// subscript_expression, another member_expression chain, a `new`
		// expression, ...) can never be a real import alias — same
		// synthetic-alias treatment, mirroring goextract's WR-02 fix.
		return "<" + object.Kind() + ">"
	}
}

// --- shared helpers ---

// isDirectlyExported reports whether node's immediate parent is an
// export_statement (`export class X {}`, `export function f() {}`,
// `export interface I {}`, `export type T = ...`) — TS/JS's real
// "visibility" concept, unlike Go's identifier-case convention.
func isDirectlyExported(node *tree_sitter.Node) bool {
	parent := node.Parent()
	return parent != nil && parent.Kind() == "export_statement"
}

func visibilityFor(exported bool) string {
	if exported {
		return "public"
	}
	return "module"
}

// isLikelyTypeName reports whether name starts with an uppercase Unicode
// letter — the near-universal TS/JS class-naming convention, mirroring
// javaextract's/csharpextract's/pyextract's identical heuristic.
func isLikelyTypeName(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants, mirroring goextract.go's/
// javaextract.go's/csharpextract.go's own walkDescendants (T-02-04 depth
// guard — no unbounded Go-stack recursion over a pathologically deep AST;
// the tree is already size-bounded by parser.MaxSourceBytes across all
// three registered grammars, T-05-DoS).
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
