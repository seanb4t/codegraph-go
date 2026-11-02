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
	discovery, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll(%s): %v", root, err)
	}
	var files []DiscoveredFile
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
			spec, _ := lookupLanguageByID(f.Language)
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

// TestEveryDeclarationHasOneNode compares, file by file, the declarations
// that a tree-sitter table lists with the nodes the extractors produce:
// each declaration has exactly one node, matched by name, kind, start line
// and start column, and no two nodes share an id. It runs over the
// in-repository fixtures, and over the pinned corpora when
// CODEGRAPH_CORPUS_DIR is set.
func TestEveryDeclarationHasOneNode(t *testing.T) {
	type tree struct{ name, root string }
	var trees []tree
	fixtures, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	for _, e := range fixtures {
		// testdata/rapid holds the failure files of rapid, not a fixture.
		if e.IsDir() && e.Name() != "rapid" {
			trees = append(trees, tree{e.Name(), filepath.Join("testdata", e.Name())})
		}
	}
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
			trees = append(trees, tree{e.Repo, dir})
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
