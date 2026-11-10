package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/extract"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
)

// declOccurrence is one declaration as the tree-sitter table sees it, or
// one extracted node. Two copies of a name on one line differ by column.
type declOccurrence struct {
	Name, Kind string
	Line, Col  int32
}

func (o declOccurrence) String() string {
	return fmt.Sprintf("%s %s at %d:%d", o.Kind, o.Name, o.Line, o.Col)
}

// declarationExceptions are the constructs whose row in the table
// disagrees with an extractor on purpose. The test logs the count of each
// on every run, so an exception cannot hide.
var declarationExceptions = map[string]string{
	"go-local-declaration":                     "the Go extractor indexes package-level declarations only",
	"ts-method-outside-class-declaration":      "the TypeScript extractor emits methods of class declarations only, not of object literals or class expressions",
	"ts-function-value-outside-exported-const": "the TypeScript extractor emits a function value only in a top-level exported const",
	"java-enum-not-extracted":                  "the Java extractor emits no enum nodes",
	"java-member-outside-class-or-interface":   "the Java extractor emits methods of class and interface declarations only, not of enums, records or anonymous classes",
	"python-function-not-at-module-or-class":   "the Python extractor emits module-level functions and class methods only",
	"csharp-member-outside-type-body":          "the C# extractor reads the members of a type body, under #if branches included, so it skips a member declared outside any type body",
	"csharp-partial-type":                      "a C# partial type has one shared node with no location, so all its parts count as one declaration",
	"ruby-reopened-type":                       "a reopened Ruby class or module adds to the node of its first declaration",
	"ruby-dynamic-type-name":                   "the Ruby extractor skips a class or module whose name is not a constant, and the methods in it",
	"ruby-method-not-at-module-or-class":       "the Ruby extractor emits top-level methods and the methods of a class or module body only",
	"c-prototype-of-defined-function":          "a C prototype of a function the same file defines is a forward declaration of that definition",
	"c-typedef-without-plain-name":             "the C extractor emits a typedef whose declarator is a plain name, not a function pointer",
	"kotlin-object-not-extracted":              "the Kotlin extractor emits no node for an object declaration",
	"kotlin-nested-type-not-extracted":         "the Kotlin extractor emits top-level types only",
	"kotlin-function-not-at-top-or-class-body": "the Kotlin extractor emits top-level functions and the functions of a top-level class or interface body only",
	"swift-extension-not-extracted":            "the Swift extractor emits no node for an extension or its members",
	"swift-nested-type-not-extracted":          "the Swift extractor emits top-level types only",
	"swift-operator-function-not-extracted":    "the Swift extractor emits functions with a simple name only",
	"swift-function-not-at-top-or-type-body":   "the Swift extractor emits top-level functions and the functions of a top-level type body only",
	"rust-item-not-at-top-level":               "the Rust extractor emits top-level items only, not the items of a module block",
	"rust-function-not-at-top-level-or-impl":   "the Rust extractor emits top-level functions and the functions of an impl block only",
	"php-enum-not-extracted":                   "the PHP extractor emits no node for an enum or its methods",
	"php-method-outside-type-body":             "the PHP extractor emits the methods of a class, interface or trait declaration only",
	"php-function-not-at-top-level":            "the PHP extractor emits functions at the top level, under a namespace or under a top-level if only",
}

// declarationKinds lists, per language, the node kinds the table covers.
// Each extracted node of one of these kinds must match one expected
// occurrence.
var declarationKinds = map[string]map[string]bool{
	"go":         kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias, goextract.KindConstant, goextract.KindVariable),
	"typescript": kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias),
	"tsx":        kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias),
	"javascript": kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct),
	"java":       kindSet(goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
	"python":     kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct),
	"csharp":     kindSet(goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
	"ruby":       kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct),
	"c":          kindSet(goextract.KindFunction, goextract.KindStruct, goextract.KindTypeAlias),
	"kotlin":     kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
	"swift":      kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
	"rust":       kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
	"php":        kindSet(goextract.KindFunction, goextract.KindMethod, goextract.KindStruct, goextract.KindInterface),
}

func kindSet(kinds ...string) map[string]bool {
	m := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		m[k] = true
	}
	return m
}

// declCollector receives what a language walker finds.
type declCollector struct {
	src        []byte
	occs       []declOccurrence
	exceptions map[string]int
	partials   map[declOccurrence]bool
}

func (c *declCollector) add(name, kind string, pos *tree_sitter.Node) {
	p := pos.StartPosition()
	c.occs = append(c.occs, declOccurrence{Name: name, Kind: kind, Line: int32(p.Row) + 1, Col: int32(p.Column)})
}

func (c *declCollector) except(name string) {
	if _, ok := declarationExceptions[name]; !ok {
		panic("unnamed declaration exception " + name)
	}
	c.exceptions[name]++
}

func (c *declCollector) text(n *tree_sitter.Node) string { return n.Utf8Text(c.src) }

func (c *declCollector) name(n *tree_sitter.Node) (string, bool) {
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return "", false
	}
	return c.text(nameNode), true
}

// declWalkers holds the table: one walker per language that lists the
// declaration occurrences of a parsed file.
var declWalkers = map[string]func(c *declCollector, root *tree_sitter.Node){
	"go":         goDecls,
	"typescript": tsDecls,
	"tsx":        tsDecls,
	"javascript": tsDecls,
	"java":       javaDecls,
	"python":     pythonDecls,
	"csharp":     csharpDecls,
	"ruby":       rubyDecls,
	"c":          cDecls,
	"kotlin":     kotlinDecls,
	"swift":      swiftDecls,
	"rust":       rustDecls,
	"php":        phpDecls,
}

func walkTree(n *tree_sitter.Node, visit func(*tree_sitter.Node)) {
	visit(n)
	for i := uint(0); i < n.NamedChildCount(); i++ {
		walkTree(n.NamedChild(i), visit)
	}
}

func parentKind(n *tree_sitter.Node) string {
	if p := n.Parent(); p != nil {
		return p.Kind()
	}
	return ""
}

func goDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "function_declaration", "method_declaration":
			kind := goextract.KindFunction
			if n.Kind() == "method_declaration" {
				kind = goextract.KindMethod
			}
			if name, ok := c.name(n); ok {
				c.add(name, kind, n)
			}
		case "type_spec", "type_alias":
			if !goTopLevel(n) {
				c.except("go-local-declaration")
				return
			}
			nameNode := n.ChildByFieldName("name")
			if nameNode == nil {
				nameNode = n.NamedChild(0)
			}
			typeNode := n.ChildByFieldName("type")
			kind := goextract.KindTypeAlias
			if typeNode != nil {
				switch typeNode.Kind() {
				case "struct_type":
					kind = goextract.KindStruct
				case "interface_type":
					kind = goextract.KindInterface
				}
			}
			c.add(c.text(nameNode), kind, n)
		case "const_spec", "var_spec":
			if !goTopLevel(n) {
				c.except("go-local-declaration")
				return
			}
			kind := goextract.KindConstant
			if n.Kind() == "var_spec" {
				kind = goextract.KindVariable
			}
			for i := uint(0); i < n.NamedChildCount(); i++ {
				id := n.NamedChild(i)
				if id.Kind() != "identifier" {
					break
				}
				if name := c.text(id); name != "_" {
					c.add(name, kind, n)
				}
			}
		}
	})
}

// goTopLevel reports whether a spec belongs to a package-level declaration.
func goTopLevel(spec *tree_sitter.Node) bool {
	decl := spec.Parent()
	if decl != nil && decl.Kind() == "var_spec_list" {
		decl = decl.Parent()
	}
	return decl != nil && parentKind(decl) == "source_file"
}

func tsDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		var kind string
		switch n.Kind() {
		case "function_declaration", "generator_function_declaration":
			kind = goextract.KindFunction
		case "class_declaration", "abstract_class_declaration":
			kind = goextract.KindStruct
		case "interface_declaration":
			kind = goextract.KindInterface
		case "type_alias_declaration":
			kind = goextract.KindTypeAlias
		case "method_definition":
			body := n.Parent()
			if body == nil || body.Kind() != "class_body" || (parentKind(body) != "class_declaration" && parentKind(body) != "abstract_class_declaration") {
				c.except("ts-method-outside-class-declaration")
				return
			}
			kind = goextract.KindMethod
		case "arrow_function", "function_expression", "generator_function":
			d := n.Parent()
			if d == nil || d.Kind() != "variable_declarator" {
				return
			}
			nameNode := d.ChildByFieldName("name")
			if nameNode == nil || nameNode.Kind() != "identifier" {
				return
			}
			if !tsExportedTopLevelConst(c, d) {
				c.except("ts-function-value-outside-exported-const")
				return
			}
			c.add(c.text(nameNode), goextract.KindFunction, d)
			return
		default:
			return
		}
		if name, ok := c.name(n); ok {
			c.add(name, kind, n)
		}
	})
}

// tsExportedTopLevelConst reports whether declarator d is in an
// `export const` statement at the top of the file.
func tsExportedTopLevelConst(c *declCollector, d *tree_sitter.Node) bool {
	decl := d.Parent()
	if decl == nil || decl.Kind() != "lexical_declaration" {
		return false
	}
	if k := decl.ChildByFieldName("kind"); k == nil || c.text(k) != "const" {
		return false
	}
	stmt := decl.Parent()
	return stmt != nil && stmt.Kind() == "export_statement" && parentKind(stmt) == "program"
}

func javaDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		var kind string
		switch n.Kind() {
		case "class_declaration":
			kind = goextract.KindStruct
		case "interface_declaration":
			kind = goextract.KindInterface
		case "enum_declaration":
			c.except("java-enum-not-extracted")
			return
		case "method_declaration", "constructor_declaration":
			body := n.Parent()
			owner := ""
			if body != nil && (body.Kind() == "class_body" || body.Kind() == "interface_body") {
				owner = parentKind(body)
			}
			if owner != "class_declaration" && owner != "interface_declaration" {
				c.except("java-member-outside-class-or-interface")
				return
			}
			kind = goextract.KindMethod
		default:
			return
		}
		if name, ok := c.name(n); ok {
			c.add(name, kind, n)
		}
	})
}

func pythonDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "class_definition":
			if name, ok := c.name(n); ok {
				c.add(name, goextract.KindStruct, n)
			}
		case "function_definition":
			outer := n.Parent()
			if outer != nil && outer.Kind() == "decorated_definition" {
				outer = outer.Parent()
			}
			var kind string
			switch {
			case outer != nil && outer.Kind() == "module":
				kind = goextract.KindFunction
			case outer != nil && outer.Kind() == "block" && parentKind(outer) == "class_definition":
				kind = goextract.KindMethod
			default:
				c.except("python-function-not-at-module-or-class")
				return
			}
			if name, ok := c.name(n); ok {
				c.add(name, kind, n)
			}
		}
	})
}

func csharpDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		var kind string
		switch n.Kind() {
		case "class_declaration", "struct_declaration", "record_declaration":
			kind = goextract.KindStruct
		case "interface_declaration":
			kind = goextract.KindInterface
		case "method_declaration", "constructor_declaration":
			// A member under an `#if` branch is still a member of the
			// type body around the branch (design.md D16).
			body := n.Parent()
			for body != nil && goextract.IsPreprocBlock(body) {
				body = body.Parent()
			}
			if body == nil || body.Kind() != "declaration_list" || !csharpTypeKinds[parentKind(body)] {
				c.except("csharp-member-outside-type-body")
				return
			}
			kind = goextract.KindMethod
		default:
			return
		}
		name, ok := c.name(n)
		if !ok {
			return
		}
		if kind != goextract.KindMethod && csharpIsPartial(c, n) {
			// The extractor gives every part of a partial type one node
			// with a sentinel location, so the parts in a file count once.
			c.except("csharp-partial-type")
			occ := declOccurrence{Name: name, Kind: kind}
			if !c.partials[occ] {
				c.partials[occ] = true
				c.occs = append(c.occs, occ)
			}
			return
		}
		c.add(name, kind, n)
	})
}

var csharpTypeKinds = map[string]bool{
	"class_declaration":     true,
	"struct_declaration":    true,
	"record_declaration":    true,
	"interface_declaration": true,
}

func csharpIsPartial(c *declCollector, n *tree_sitter.Node) bool {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if m := n.NamedChild(i); m.Kind() == "modifier" && c.text(m) == "partial" {
			return true
		}
	}
	return false
}

// declarationReport is the outcome of checking one tree.
type declarationReport struct {
	matched    int
	exceptions map[string]int
	problems   []string
}

// checkDeclarations discovers and extracts root, lists each file's
// declarations with the table, and compares the two.
func checkDeclarations(t *testing.T, root string) declarationReport {
	t.Helper()
	discovery, err := discover.DiscoverAll(root)
	if err != nil {
		t.Fatalf("discover.DiscoverAll(%s): %v", root, err)
	}
	var files []discover.DiscoveredFile
	for _, f := range discovery.Files {
		if declWalkers[f.Language] != nil {
			files = append(files, f)
		}
	}
	results, err := Extract(files, 0)
	if err != nil {
		t.Fatalf("Extract(%s): %v", root, err)
	}

	rep := declarationReport{exceptions: make(map[string]int)}
	parsers := make(map[string]parser.Parser)
	defer func() {
		for _, p := range parsers {
			p.Close()
		}
	}()

	firstID := make(map[string]string)
	for i, f := range files {
		r := results[i]
		if r.Err != nil {
			continue
		}
		p, ok := parsers[f.Language]
		if !ok {
			spec, _ := extract.ByID(f.Language)
			if p, err = spec.NewParser(); err != nil {
				t.Fatalf("NewParser(%s): %v", f.Language, err)
			}
			parsers[f.Language] = p
		}
		src, err := os.ReadFile(f.AbsPath)
		if err != nil {
			t.Fatalf("read %s: %v", f.AbsPath, err)
		}
		tree, err := p.Parse(src, nil)
		if err != nil {
			t.Fatalf("parse %s: %v", f.RelPath, err)
		}
		native := tree.Inner().(*tree_sitter.Tree)
		c := &declCollector{src: src, exceptions: rep.exceptions, partials: make(map[declOccurrence]bool)}
		declWalkers[f.Language](c, native.RootNode())
		tree.Close()

		kinds := declarationKinds[f.Language]
		want := make(map[declOccurrence]int)
		for _, o := range c.occs {
			if kinds[o.Kind] {
				want[o]++
			}
		}
		got := make(map[declOccurrence]int)
		partialIDs := make(map[string]bool)
		for _, en := range r.Nodes {
			n := en.Node
			if n.Kind == goextract.KindFile {
				continue
			}
			if n.FilePath == "" {
				// A C# partial type: one shared node for every part.
				if partialIDs[n.Id] {
					continue
				}
				partialIDs[n.Id] = true
			} else if prev, dup := firstID[n.Id]; dup {
				rep.problems = append(rep.problems, fmt.Sprintf("%s: %s has id %s, the same as %s", f.RelPath,
					declOccurrence{n.Name, n.Kind, n.StartLine, n.StartCol}, n.Id, prev))
			} else {
				firstID[n.Id] = fmt.Sprintf("%s %s at %s:%d:%d", n.Kind, n.Name, f.RelPath, n.StartLine, n.StartCol)
			}
			if kinds[n.Kind] {
				got[declOccurrence{n.Name, n.Kind, n.StartLine, n.StartCol}]++
			}
		}
		for _, o := range sortedOccurrences(want, got) {
			w, g := want[o], got[o]
			switch {
			case w == g:
				rep.matched += w
			case g == 0:
				rep.problems = append(rep.problems, fmt.Sprintf("%s: %s has no extracted node", f.RelPath, o))
			case w == 0:
				rep.problems = append(rep.problems, fmt.Sprintf("%s: extracted node %s matches no declaration", f.RelPath, o))
			default:
				rep.problems = append(rep.problems, fmt.Sprintf("%s: %s is declared %d times and extracted %d times", f.RelPath, o, w, g))
			}
		}
	}
	return rep
}

func sortedOccurrences(a, b map[declOccurrence]int) []declOccurrence {
	keys := make([]declOccurrence, 0, len(a)+len(b))
	for o := range a {
		keys = append(keys, o)
	}
	for o := range b {
		if _, ok := a[o]; !ok {
			keys = append(keys, o)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Line != keys[j].Line {
			return keys[i].Line < keys[j].Line
		}
		if keys[i].Col != keys[j].Col {
			return keys[i].Col < keys[j].Col
		}
		return keys[i].Kind+keys[i].Name < keys[j].Kind+keys[j].Name
	})
	return keys
}

// fixtureTree is one source tree that a test indexes.
type fixtureTree struct{ name, root string }

// fixtureTrees lists the fixture trees under testdata. It is the one walk
// of that directory, so every test that runs over "each fixture" sees the
// same set.
func fixtureTrees(t *testing.T) []fixtureTree {
	t.Helper()
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	var trees []fixtureTree
	for _, e := range entries {
		// testdata/rapid holds the failure files of rapid, not a fixture.
		if e.IsDir() && e.Name() != "rapid" {
			trees = append(trees, fixtureTree{e.Name(), filepath.Join("testdata", e.Name())})
		}
	}
	if len(trees) == 0 {
		t.Fatal("testdata holds no fixture tree")
	}
	return trees
}

// TestEveryDeclarationHasOneNode compares, file by file, the declarations
// that a tree-sitter table lists with the nodes the extractors produce:
// each declaration has exactly one node, matched by name, kind, start line
// and start column, and no two nodes share an id. It runs over the
// in-repository fixtures, and over the pinned corpora when
// CODEGRAPH_CORPUS_DIR is set.
func TestEveryDeclarationHasOneNode(t *testing.T) {
	trees := fixtureTrees(t)
	if os.Getenv("CODEGRAPH_CORPUS_DIR") != "" {
		m, err := corpora.Load(filepath.Join("..", "..", "corpora", "manifest.json"))
		if err != nil {
			t.Fatalf("load corpora manifest: %v", err)
		}
		root, err := corpora.CorpusRoot()
		if err != nil {
			t.Fatalf("corpus root: %v", err)
		}
		locked := corpora.LockedEntries(m)
		if len(locked) == 0 {
			t.Fatal("the corpora manifest locks no corpus")
		}
		for _, e := range locked {
			dir := e.Dir(root)
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("corpus %s is not fetched at %s: %v", e.Repo, dir, err)
			}
			trees = append(trees, fixtureTree{e.Repo, dir})
		}
	}

	for _, tr := range trees {
		t.Run(tr.name, func(t *testing.T) {
			rep := checkDeclarations(t, tr.root)
			names := make([]string, 0, len(rep.exceptions))
			for name := range rep.exceptions {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				t.Logf("exception %s (%d): %s", name, rep.exceptions[name], declarationExceptions[name])
			}
			t.Logf("%d declarations matched", rep.matched)
			if rep.matched == 0 {
				t.Errorf("matched no declaration in %s", tr.root)
			}
			if len(rep.problems) > 0 {
				shown := rep.problems
				if len(shown) > 40 {
					shown = shown[:40]
				}
				t.Errorf("%d problems, the first %d:\n%s", len(rep.problems), len(shown), strings.Join(shown, "\n"))
			}
		})
	}
}

// rubyConstant mirrors rubyextract.rubyConstantName: the text of a
// constant, or the name part of a scope resolution.
func rubyConstant(c *declCollector, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "constant":
		return c.text(n)
	case "scope_resolution":
		if name := n.ChildByFieldName("name"); name != nil {
			return c.text(name)
		}
	}
	return ""
}

func rubyDecls(c *declCollector, root *tree_sitter.Node) {
	seen := make(map[string]bool)
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "class", "module":
			nameNode := n.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := rubyConstant(c, nameNode)
			if name == "" {
				c.except("ruby-dynamic-type-name")
				return
			}
			if seen[name] {
				c.except("ruby-reopened-type")
				return
			}
			seen[name] = true
			c.add(name, goextract.KindStruct, n)
		case "method", "singleton_method":
			name, ok := c.name(n)
			if !ok {
				return
			}
			parent := n.Parent()
			switch {
			case parent != nil && parent.Kind() == "program":
				c.add(name, goextract.KindFunction, n)
			case parent != nil && parent.Kind() == "body_statement" && (parentKind(parent) == "class" || parentKind(parent) == "module"):
				owner := parent.Parent().ChildByFieldName("name")
				if owner == nil || rubyConstant(c, owner) == "" {
					c.except("ruby-dynamic-type-name")
					return
				}
				c.add(name, goextract.KindMethod, n)
			default:
				c.except("ruby-method-not-at-module-or-class")
			}
		}
	})
}

// cTopLevel mirrors cextract.topLevelDecls: the root's children, with
// namespace bodies and preprocessor branches flattened.
func cTopLevel(root *tree_sitter.Node) []*tree_sitter.Node {
	var decls []*tree_sitter.Node
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		for i := uint(0); i < n.NamedChildCount(); i++ {
			child := n.NamedChild(i)
			if child.Kind() == "namespace_definition" {
				if body := child.ChildByFieldName("body"); body != nil {
					walk(body)
				}
				continue
			}
			if goextract.IsPreprocBlock(child) {
				walk(child)
				continue
			}
			decls = append(decls, child)
		}
	}
	walk(root)
	return decls
}

// cFunctionName returns the plain name of the function that a
// function_definition or declaration declares.
func cFunctionName(c *declCollector, decl *tree_sitter.Node) (string, bool) {
	d := decl.ChildByFieldName("declarator")
	for d != nil && d.Kind() != "function_declarator" {
		switch d.Kind() {
		case "pointer_declarator", "init_declarator":
			d = d.ChildByFieldName("declarator")
		case "parenthesized_declarator":
			if d.NamedChildCount() == 0 {
				return "", false
			}
			d = d.NamedChild(0)
		default:
			return "", false
		}
	}
	if d == nil {
		return "", false
	}
	name := d.ChildByFieldName("declarator")
	if name == nil || name.Kind() != "identifier" {
		return "", false
	}
	return c.text(name), true
}

// cPlainName mirrors cextract.cDeclaratorName: the name under pointer,
// array and parenthesized declarators, and none under a function
// declarator.
func cPlainName(c *declCollector, d *tree_sitter.Node) (string, bool) {
	for d != nil {
		switch d.Kind() {
		case "identifier", "field_identifier", "type_identifier":
			return c.text(d), true
		case "pointer_declarator", "array_declarator", "reference_declarator", "init_declarator":
			d = d.ChildByFieldName("declarator")
		case "parenthesized_declarator":
			if d.NamedChildCount() == 0 {
				return "", false
			}
			d = d.NamedChild(0)
		default:
			return "", false
		}
	}
	return "", false
}

func cDecls(c *declCollector, root *tree_sitter.Node) {
	decls := cTopLevel(root)
	defined := make(map[string]bool)
	for _, d := range decls {
		if d.Kind() == "function_definition" {
			if name, ok := cFunctionName(c, d); ok {
				defined[name] = true
			}
		}
	}
	for _, d := range decls {
		switch d.Kind() {
		case "function_definition":
			if name, ok := cFunctionName(c, d); ok {
				c.add(name, goextract.KindFunction, d)
			}
		case "declaration":
			name, ok := cFunctionName(c, d)
			if !ok {
				continue
			}
			if defined[name] {
				c.except("c-prototype-of-defined-function")
				continue
			}
			c.add(name, goextract.KindFunction, d)
		case "struct_specifier":
			if name := d.ChildByFieldName("name"); name != nil && name.Kind() == "type_identifier" {
				c.add(c.text(name), goextract.KindStruct, d)
			}
		case "type_definition":
			name, ok := cPlainName(c, d.ChildByFieldName("declarator"))
			if !ok {
				c.except("c-typedef-without-plain-name")
				continue
			}
			c.add(name, goextract.KindTypeAlias, d)
		}
	}
}

// keywordKind returns the first anonymous child of n whose kind is one of
// kinds: the keyword that says what a declaration declares.
func keywordKind(n *tree_sitter.Node, kinds ...string) string {
	for i := uint(0); i < n.ChildCount(); i++ {
		k := n.Child(i).Kind()
		for _, want := range kinds {
			if k == want {
				return k
			}
		}
	}
	return ""
}

// kotlinTypeKind is the node kind the Kotlin extractor gives a top-level
// class_declaration, or "" for one it skips.
func kotlinTypeKind(decl *tree_sitter.Node) string {
	switch keywordKind(decl, "class", "interface", "enum", "object") {
	case "class", "enum":
		return goextract.KindStruct
	case "interface":
		return goextract.KindInterface
	}
	return ""
}

func kotlinDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "class_declaration":
			if parentKind(n) != "source_file" {
				c.except("kotlin-nested-type-not-extracted")
				return
			}
			kind := kotlinTypeKind(n)
			if kind == "" {
				c.except("kotlin-object-not-extracted")
				return
			}
			if name := n.ChildByFieldName("name"); name != nil && name.Kind() == "identifier" {
				c.add(c.text(name), kind, n)
			}
		case "function_declaration":
			name := n.ChildByFieldName("name")
			if name == nil || name.Kind() != "identifier" {
				return
			}
			parent := n.Parent()
			switch {
			case parent != nil && parent.Kind() == "source_file":
				c.add(c.text(name), goextract.KindFunction, n)
			case parent != nil && parent.Kind() == "class_body" && parentKind(parent) == "class_declaration" &&
				parentKind(parent.Parent()) == "source_file" && kotlinTypeKind(parent.Parent()) != "":
				c.add(c.text(name), goextract.KindMethod, n)
			default:
				c.except("kotlin-function-not-at-top-or-class-body")
			}
		}
	})
}

// swiftTypeName mirrors swiftextract.swiftSimpleTypeName.
func swiftTypeName(c *declCollector, n *tree_sitter.Node) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind() {
	case "type_identifier":
		return c.text(n), true
	case "user_type":
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if t := n.NamedChild(i); t.Kind() == "type_identifier" {
				return c.text(t), true
			}
		}
	}
	return "", false
}

// swiftEmittedType reports whether decl is a top-level class, struct,
// enum or actor declaration the Swift extractor emits.
func swiftEmittedType(c *declCollector, decl *tree_sitter.Node) bool {
	if decl == nil || decl.Kind() != "class_declaration" || parentKind(decl) != "source_file" {
		return false
	}
	switch keywordKind(decl, "class", "struct", "enum", "actor", "extension") {
	case "class", "struct", "enum", "actor":
		_, ok := swiftTypeName(c, decl.ChildByFieldName("name"))
		return ok
	}
	return false
}

func swiftDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "class_declaration":
			if parentKind(n) != "source_file" {
				c.except("swift-nested-type-not-extracted")
				return
			}
			if keywordKind(n, "class", "struct", "enum", "actor", "extension") == "extension" {
				c.except("swift-extension-not-extracted")
				return
			}
			if name, ok := swiftTypeName(c, n.ChildByFieldName("name")); ok {
				c.add(name, goextract.KindStruct, n)
			}
		case "protocol_declaration":
			if parentKind(n) != "source_file" {
				c.except("swift-nested-type-not-extracted")
				return
			}
			if name, ok := swiftTypeName(c, n.ChildByFieldName("name")); ok {
				c.add(name, goextract.KindInterface, n)
			}
		case "function_declaration":
			parent := n.Parent()
			atTop := parent != nil && parent.Kind() == "source_file"
			inType := parent != nil && parent.Kind() == "class_body" && swiftEmittedType(c, parent.Parent())
			if !atTop && !inType {
				c.except("swift-function-not-at-top-or-type-body")
				return
			}
			name := n.ChildByFieldName("name")
			if name == nil || name.Kind() != "simple_identifier" {
				c.except("swift-operator-function-not-extracted")
				return
			}
			if atTop {
				c.add(c.text(name), goextract.KindFunction, n)
			} else {
				c.add(c.text(name), goextract.KindMethod, n)
			}
		}
	})
}

// rustImplType mirrors rustextract.rustTypeName on an impl block's type.
func rustImplType(c *declCollector, n *tree_sitter.Node) (string, bool) {
	for n != nil {
		switch n.Kind() {
		case "type_identifier":
			return c.text(n), true
		case "generic_type":
			n = n.ChildByFieldName("type")
		case "scoped_type_identifier", "scoped_identifier":
			if name := n.ChildByFieldName("name"); name != nil {
				return c.text(name), true
			}
			return "", false
		default:
			return "", false
		}
	}
	return "", false
}

func rustDecls(c *declCollector, root *tree_sitter.Node) {
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "struct_item", "enum_item", "trait_item":
			if parentKind(n) != "source_file" {
				c.except("rust-item-not-at-top-level")
				return
			}
			kind := goextract.KindStruct
			if n.Kind() == "trait_item" {
				kind = goextract.KindInterface
			}
			if name, ok := c.name(n); ok {
				c.add(name, kind, n)
			}
		case "function_item":
			name, ok := c.name(n)
			if !ok {
				return
			}
			parent := n.Parent()
			switch {
			case parent != nil && parent.Kind() == "source_file":
				c.add(name, goextract.KindFunction, n)
			case parent != nil && parent.Kind() == "declaration_list" && parentKind(parent) == "impl_item" && parentKind(parent.Parent()) == "source_file":
				if _, ok := rustImplType(c, parent.Parent().ChildByFieldName("type")); ok {
					c.add(name, goextract.KindMethod, n)
				}
			default:
				c.except("rust-function-not-at-top-level-or-impl")
			}
		}
	})
}

// phpTopLevel mirrors phpextract.topLevelDecls: the root's children with
// namespace bodies, top-level if branches and blocks flattened.
func phpTopLevel(root *tree_sitter.Node) map[uintptr]bool {
	decls := make(map[uintptr]bool)
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		for i := uint(0); i < n.NamedChildCount(); i++ {
			child := n.NamedChild(i)
			switch child.Kind() {
			case "namespace_definition":
				if body := child.ChildByFieldName("body"); body != nil {
					walk(body)
				}
				continue
			case "if_statement":
				for j := uint(0); j < child.NamedChildCount(); j++ {
					if b := child.NamedChild(j); b.Kind() == "compound_statement" || b.Kind() == "else_clause" {
						walk(b)
					}
				}
				continue
			case "compound_statement":
				walk(child)
				continue
			}
			decls[child.Id()] = true
		}
	}
	walk(root)
	return decls
}

func phpDecls(c *declCollector, root *tree_sitter.Node) {
	top := phpTopLevel(root)
	walkTree(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "class_declaration", "trait_declaration", "interface_declaration":
			if !top[n.Id()] {
				return
			}
			kind := goextract.KindStruct
			if n.Kind() == "interface_declaration" {
				kind = goextract.KindInterface
			}
			if name, ok := c.name(n); ok {
				c.add(name, kind, n)
			}
		case "enum_declaration":
			if top[n.Id()] {
				c.except("php-enum-not-extracted")
			}
		case "method_declaration":
			parent := n.Parent()
			owner := ""
			if parent != nil && parent.Kind() == "declaration_list" && parent.Parent() != nil && top[parent.Parent().Id()] {
				owner = parentKind(parent)
			}
			switch owner {
			case "class_declaration", "trait_declaration", "interface_declaration":
				if name, ok := c.name(n); ok {
					c.add(name, goextract.KindMethod, n)
				}
			default:
				c.except("php-method-outside-type-body")
			}
		case "function_definition":
			if !top[n.Id()] {
				c.except("php-function-not-at-top-level")
				return
			}
			if name, ok := c.name(n); ok {
				c.add(name, goextract.KindFunction, n)
			}
		}
	})
}
