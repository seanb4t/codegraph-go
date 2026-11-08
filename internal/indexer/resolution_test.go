package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestDeferredTypeFillsFromAnotherFile checks design.md D17: a local
// initialized from a call into another file takes that call's return
// type in pass 2, and a call through the local resolves by it. The
// caller's file sorts before the callee's, so the order of the files
// cannot supply the type by accident.
func TestDeferredTypeFillsFromAnotherFile(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a/App.java", `package a;

import z.Factory;

public class App {
    public void run(String s) {
        var b = Factory.make();
        b.add(s);
    }
}
`)
	write("z/Builder.java", `package z;

public class Builder {
    public Builder add(int n) { return this; }

    public Builder add(String s) { return this; }
}
`)
	write("z/Factory.java", `package z;

public class Factory {
    public static Builder make() { return new Builder(); }
}
`)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	g := readGraph(t, storeDir)
	run := g.node(t, "a/App.java", goextract.KindMethod, "run", "")
	want := g.node(t, "z/Builder.java", goextract.KindMethod, "add", "(String s)")
	g.requireOnlyCallNamed(t, run, want)
}

// TestOverloadSetExistsOnlyForTwoOrMore checks design.md D14 over
// overloadfixture: every name with two or more overloads under one owner
// has one set that the owner contains and that contains each overload,
// and a name with one declaration has no set and stays under its owner.
// The partial subtest covers a C# partial type whose overloads span
// files: one set, with the type's sentinel location.
func TestOverloadSetExistsOnlyForTwoOrMore(t *testing.T) {
	g := loadOverloadGraph(t)
	byID := make(map[string]*schema.Node, len(g.nodes))
	for _, n := range g.nodes {
		byID[n.Id] = n
	}
	parents := make(map[string][]*schema.Node)
	for _, e := range g.edges {
		if e.Kind == goextract.RefKindContains {
			parents[e.Target] = append(parents[e.Target], byID[e.Source])
		}
	}
	type group struct{ owner, name string }
	groups := make(map[group][]*schema.Node)
	for _, n := range g.nodes {
		if n.Kind != goextract.KindMethod && n.Kind != goextract.KindFunction {
			continue
		}
		for _, p := range parents[n.Id] {
			if p == nil {
				t.Errorf("%s %s has a contains edge from a node the store lacks", n.Kind, n.QualifiedName)
				continue
			}
			owner := p
			if p.Kind == goextract.KindOverloadSet {
				if ps := parents[p.Id]; len(ps) == 1 && ps[0] != nil {
					owner = ps[0]
				} else {
					t.Errorf("set %s has %d parents, want 1", p.QualifiedName, len(ps))
					continue
				}
			}
			groups[group{owner.Id, n.Name}] = append(groups[group{owner.Id, n.Name}], n)
		}
	}
	sets := 0
	for gk, members := range groups {
		owner := byID[gk.owner]
		var direct, viaSet int
		var set *schema.Node
		for _, m := range members {
			for _, p := range parents[m.Id] {
				switch {
				case p.Id == owner.Id:
					direct++
				case p.Kind == goextract.KindOverloadSet:
					viaSet++
					set = p
				}
			}
		}
		switch {
		case len(members) >= 2 && (direct != 0 || viaSet != len(members) || set == nil):
			t.Errorf("%s.%s: %d overloads, %d under the owner and %d under a set, want all under one set", owner.QualifiedName, gk.name, len(members), direct, viaSet)
		case len(members) == 1 && (direct != 1 || viaSet != 0):
			t.Errorf("%s.%s: one declaration, %d under the owner and %d under a set, want it under the owner", owner.QualifiedName, gk.name, direct, viaSet)
		}
		if set != nil {
			sets++
			if set.Language != members[0].Language || set.Name != gk.name {
				t.Errorf("set %s: language %q name %q, want %q %q", set.Id, set.Language, set.Name, members[0].Language, gk.name)
			}
		}
	}
	if sets == 0 {
		t.Fatal("no overload set found in overloadfixture")
	}
	for _, n := range g.nodes {
		if n.Kind == goextract.KindOverloadSet && len(parents[n.Id]) != 1 {
			t.Errorf("set %s has %d parents, want 1", n.QualifiedName, len(parents[n.Id]))
		}
	}

	t.Run("partial type", func(t *testing.T) {
		var store *schema.Node
		for _, n := range g.nodes {
			if n.Kind == goextract.KindStruct && n.Name == "Store" && n.Language == "csharp" {
				store = n
			}
		}
		if store == nil {
			t.Fatal("no Store type")
		}
		var sets []*schema.Node
		for _, e := range g.edges {
			if e.Kind == goextract.RefKindContains && e.Source == store.Id && byID[e.Target] != nil && byID[e.Target].Kind == goextract.KindOverloadSet {
				sets = append(sets, byID[e.Target])
			}
		}
		if len(sets) != 1 || sets[0].Name != "Put" {
			t.Fatalf("Store contains %d sets, want one named Put", len(sets))
		}
		set := sets[0]
		if set.FilePath != "" || set.StartLine != 0 {
			t.Errorf("set Put spans files but has location %q:%d, want the sentinel", set.FilePath, set.StartLine)
		}
		var files []string
		for _, e := range g.edges {
			if e.Kind == goextract.RefKindContains && e.Source == set.Id {
				files = append(files, byID[e.Target].FilePath)
			}
		}
		if len(files) != 2 || files[0] == files[1] {
			t.Errorf("set Put contains members in %v, want two files", files)
		}
		fill := g.node(t, "csharp/StoreB.cs", goextract.KindMethod, "Fill", "")
		g.requireOnlyCallNamed(t, fill, g.node(t, "csharp/StoreB.cs", goextract.KindMethod, "Put", "(string value)"))
	})
}

// TestNestedDeclarationIsNotVisibleAcrossFiles checks design.md D15 over
// two TypeScript files. In a.ts, `helper` is declared twice, in two
// nested scopes, and never at module level. A call in the inner scope
// reaches the inner `helper`, a call in the outer function reaches the
// outer one, and a module-level call finds none. b.ts imports `helper`
// and calls it: the call stays unresolved, and neither nested `helper`
// has a caller from b.ts.
func TestNestedDeclarationIsNotVisibleAcrossFiles(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.ts", `export function outer() {
  function helper() {
    return 1;
  }
  function mid() {
    function helper() {
      return 2;
    }
    function inner() {
      return helper();
    }
    return inner();
  }
  function useOuter() {
    return helper();
  }
  return mid() + useOuter();
}

export function atModuleLevel() {
  return helper();
}
`)
	write("b.ts", `import { helper } from "./a";

export function fromB() {
  return helper();
}
`)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	requireInvariants(t, storeDir)
	g := readGraph(t, storeDir)
	fn := goextract.KindFunction
	var helpers []*schema.Node
	for _, n := range g.nodes {
		if n.Name == "helper" && n.FilePath == "a.ts" {
			helpers = append(helpers, n)
		}
	}
	if len(helpers) != 2 {
		t.Fatalf("a.ts has %d helper nodes, want 2", len(helpers))
	}
	outerHelper, innerHelper := helpers[0], helpers[1]
	if outerHelper.StartLine > innerHelper.StartLine {
		outerHelper, innerHelper = innerHelper, outerHelper
	}
	for _, h := range helpers {
		if !h.Nested {
			t.Errorf("helper at line %d is not marked nested", h.StartLine)
		}
	}
	g.requireOnlyCall(t, g.node(t, "a.ts", fn, "inner", ""), innerHelper)
	g.requireOnlyCall(t, g.node(t, "a.ts", fn, "useOuter", ""), outerHelper)
	for _, caller := range []string{"atModuleLevel", "fromB"} {
		file := "a.ts"
		if caller == "fromB" {
			file = "b.ts"
		}
		if targets := g.callTargets(g.node(t, file, fn, caller, "")); len(targets) != 0 {
			t.Errorf("%s calls %v, want no call edge: no helper is visible there", caller, targets)
		}
	}
}

// TestLastDefinitionWinsAtModuleLevel checks design.md D16 for Python,
// Ruby and JavaScript: when a file defines `f` twice at module level, a
// call to `f` reaches the second definition, as it does at runtime.
func TestLastDefinitionWinsAtModuleLevel(t *testing.T) {
	cases := []struct{ lang, file, src string }{
		{"python", "m.py", "def f():\n    return 1\n\n\ndef f():\n    return 2\n\n\ndef g():\n    return f()\n"},
		{"ruby", "m.rb", "def f\n  1\nend\n\ndef f\n  2\nend\n\ndef g\n  f()\nend\n"},
		{"javascript", "m.js", "function f() {\n  return 1;\n}\n\nfunction f() {\n  return 2;\n}\n\nfunction g() {\n  return f();\n}\n"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, c.file), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			storeDir := t.TempDir()
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			g := readGraph(t, storeDir)
			var fs []*schema.Node
			for _, n := range g.nodes {
				if n.Name == "f" && n.Kind == goextract.KindFunction {
					fs = append(fs, n)
				}
			}
			if len(fs) != 2 {
				t.Fatalf("%d f nodes, want 2", len(fs))
			}
			last := fs[0]
			if fs[1].StartLine > last.StartLine {
				last = fs[1]
			}
			g.requireOnlyCall(t, g.node(t, c.file, goextract.KindFunction, "g", ""), last)
		})
	}
}

// TestChainedCallsResolveEachLink checks that every link of a call chain
// gets its own deferred type: a.b().c() starts each link at the same
// position, and only the end of a call tells the links apart.
func TestChainedCallsResolveEachLink(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"a/App.java": `package a;

import z.Factory;

public class App {
    public void run(String s) {
        Factory.make().build().add(s);
    }
}
`,
		"z/Builder.java": `package z;

public class Builder {
    public Builder add(int n) { return this; }

    public Builder add(String s) { return this; }
}
`,
		"z/Maker.java": `package z;

public class Maker {
    public Builder build() { return new Builder(); }
}
`,
		"z/Factory.java": `package z;

public class Factory {
    public static Maker make() { return new Maker(); }
}
`,
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	g := readGraph(t, storeDir)
	run := g.node(t, "a/App.java", goextract.KindMethod, "run", "")
	g.requireOnlyCallNamed(t, run, g.node(t, "z/Builder.java", goextract.KindMethod, "add", "(String s)"))
	g.requireOnlyCallNamed(t, run, g.node(t, "z/Maker.java", goextract.KindMethod, "build", ""))
}

// TestRedefinitionWithAnotherArityWinsLast checks design D16 when the
// redefinition changes the parameter count: the last definition wins
// before any filter reads an arity, so a call that fits only the earlier
// copy still reaches the last one, and no overload set exists.
func TestRedefinitionWithAnotherArityWinsLast(t *testing.T) {
	cases := []struct{ lang, file, src string }{
		{"python", "m.py", "def f(a):\n    return 1\n\n\ndef f(a, b):\n    return 2\n\n\ndef g():\n    return f(1)\n"},
		{"javascript", "m.js", "function f(a) {\n  return 1;\n}\n\nfunction f(a, b) {\n  return 2;\n}\n\nfunction g() {\n  return f(1);\n}\n"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			root := writeFixture(t, map[string]string{c.file: c.src})
			storeDir := t.TempDir()
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			g := readGraph(t, storeDir)
			var last *schema.Node
			for _, n := range g.nodes {
				if n.Kind == goextract.KindOverloadSet {
					t.Errorf("found overload set %s, want none in a last-wins language", n.QualifiedName)
				}
				if n.Name == "f" && n.Kind == goextract.KindFunction && (last == nil || n.StartLine > last.StartLine) {
					last = n
				}
			}
			if last == nil {
				t.Fatal("no f node")
			}
			g.requireOnlyCall(t, g.node(t, c.file, goextract.KindFunction, "g", ""), last)
		})
	}
}

// TestLabeledCallsResolveByLabel checks labelsFit through resolution: a
// Swift call picks the function whose labels it uses, and a Kotlin named
// argument must name a parameter of the overload.
func TestLabeledCallsResolveByLabel(t *testing.T) {
	t.Run("swift", func(t *testing.T) {
		root := writeFixture(t, map[string]string{
			"app.swift": "func move(to point: Int) {}\nfunc move(by delta: Int) {}\nfunc run() {\n    move(by: 1)\n}\n",
		})
		storeDir := t.TempDir()
		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		g := readGraph(t, storeDir)
		var want *schema.Node
		for _, n := range g.nodes {
			if n.QualifiedName == "move(by:)" {
				want = n
			}
		}
		if want == nil {
			t.Fatal("no node for move(by:)")
		}
		g.requireOnlyCall(t, g.node(t, "app.swift", goextract.KindFunction, "run", ""), want)
	})
	t.Run("kotlin", func(t *testing.T) {
		root := writeFixture(t, map[string]string{
			"a.kt": "package fixture\n\nfun f(a: Int) {}\n\nfun f(b: Int, c: Int = 0) {}\n\nfun run() {\n    f(b = 1)\n}\n",
		})
		storeDir := t.TempDir()
		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		g := readGraph(t, storeDir)
		var want *schema.Node
		for _, n := range g.nodes {
			if n.Kind == goextract.KindFunction && n.Name == "f" && len(n.Params) == 2 {
				want = n
			}
		}
		if want == nil {
			t.Fatal("no node for the two-parameter f")
		}
		g.requireOnlyCall(t, g.node(t, "a.kt", goextract.KindFunction, "run", ""), want)
	})
}

// TestDeclarationIdSurvivesEditsAbove checks the graph-index scenario "an
// edit above a declaration keeps its id" through a sync: lines inserted
// above a function move it, and its id is unchanged.
func TestDeclarationIdSurvivesEditsAbove(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod": "module example.com/stable\n\ngo 1.26\n",
		"a.go":   "package stable\n\nfunc Keep() {}\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	before := readGraph(t, storeDir).node(t, "a.go", goextract.KindFunction, "Keep", "")
	writeFile(t, filepath.Join(root, "a.go"), "package stable\n\n// Keep stays.\n\nfunc Added() {}\n\nfunc Keep() {}\n")
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	after := readGraph(t, storeDir).node(t, "a.go", goextract.KindFunction, "Keep", "")
	if after.Id != before.Id {
		t.Errorf("Keep's id changed from %s to %s across an edit above it", before.Id, after.Id)
	}
	if after.StartLine != 7 {
		t.Errorf("Keep starts at line %d after the edit, want 7", after.StartLine)
	}
}

// TestAlternativesResolveToAmbiguousSet checks design D16 through
// resolution for C and PHP: two copies of one signature under
// conditional compilation or a function_exists guard are alternatives,
// and a call that fits both reaches their set, marked ambiguous.
func TestAlternativesResolveToAmbiguousSet(t *testing.T) {
	cases := []struct{ lang, file, src string }{
		{"c", "m.c", "#ifdef FAST\nint f(int a) { return 1; }\n#else\nint f(int a) { return 2; }\n#endif\nint g(void) { return f(1); }\n"},
		{"php", "h.php", "<?php\nif (!function_exists('f')) {\n    function f($s) { return $s; }\n}\nif (!function_exists('f')) {\n    function f($s) { return strtolower($s); }\n}\nfunction g() { return f(\"A\"); }\n"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			root := writeFixture(t, map[string]string{c.file: c.src})
			storeDir := t.TempDir()
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			g := readGraph(t, storeDir)
			set := g.node(t, c.file, goextract.KindOverloadSet, "f", "")
			targets := g.callTargets(g.node(t, c.file, goextract.KindFunction, "g", ""))
			if ambiguous, ok := targets[set.Id]; !ok || !ambiguous {
				t.Errorf("g calls %v, want one ambiguous edge to the f set", targets)
			}
			for id := range targets {
				if id != set.Id {
					t.Errorf("g also calls %s", g.describe(id))
				}
			}
		})
	}
}

// TestQualifiedParameterTypeMatchesBareArgumentType checks that a
// parameter written with a package qualifier, such as loggers.Logger,
// accepts an argument whose type the resolver filled as the bare name
// Logger: a Go package-qualified call reaches the package function and
// not a method of the same name whose parameters fit anything.
func TestQualifiedParameterTypeMatchesBareArgumentType(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod":             "module example.com/hx\n\ngo 1.26\n",
		"loggers/loggers.go": "package loggers\n\ntype Logger interface{ Info(string) }\n\ntype logger struct{}\n\nfunc (l logger) Info(string) {}\n\nfunc NewDefault() Logger { return logger{} }\n",
		"hexec/exec.go":      "package hexec\n\nimport \"example.com/hx/loggers\"\n\ntype Exec struct{}\n\nfunc New(workingDir string, log loggers.Logger) *Exec {\n\treturn &Exec{}\n}\n\nfunc (e *Exec) New(name string, arg ...any) error {\n\treturn nil\n}\n",
		"app/app.go":         "package app\n\nimport (\n\t\"example.com/hx/hexec\"\n\t\"example.com/hx/loggers\"\n)\n\nfunc Build() *hexec.Exec {\n\treturn hexec.New(\"\", loggers.NewDefault())\n}\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	g := readGraph(t, storeDir)
	g.requireOnlyCallNamed(t, g.node(t, "app/app.go", goextract.KindFunction, "Build", ""), g.node(t, "hexec/exec.go", goextract.KindFunction, "New", ""))
}
