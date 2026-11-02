package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestResolutionIgnoresInsertionOrder checks design.md D3: the overload a
// call reaches is a function of the candidate set, not of the order the
// candidates arrived in. It runs the resolver over every order of three
// Java overloads of `add` for one call with a String argument.
func TestResolutionIgnoresInsertionOrder(t *testing.T) {
	param := func(name, typ string) *schema.Param { return &schema.Param{Name: name, Type: typ} }
	owner := &schema.Node{Id: "struct:Sink", Kind: goextract.KindStruct, Name: "Sink", FilePath: "Sink.java", Language: "java"}
	overloads := []*schema.Node{
		{Id: "method:add:int", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(int n)", Params: []*schema.Param{param("n", "int")}},
		{Id: "method:add:String", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(String s)", Params: []*schema.Param{param("s", "String")}},
		{Id: "method:add:String,int", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.java", Language: "java",
			Signature: "(String s, int n)", Params: []*schema.Param{param("s", "String"), param("n", "int")}},
	}
	call := goextract.UnresolvedRef{
		Kind: goextract.RefKindCalls, Name: "add", FromID: "method:use",
		ReceiverType: "Sink", ArgCount: 1, ArgTypes: []string{"String"},
	}
	file := goextract.FileResult{RelPath: "Use.java", ImportPath: "com.example", Language: "java"}

	var permute func(order []*schema.Node, rest []*schema.Node)
	permute = func(order, rest []*schema.Node) {
		if len(rest) == 0 {
			idx := newEmptySymbolIndex()
			nodeByID := map[string]*schema.Node{owner.Id: owner}
			kinds := map[string]string{owner.Id: owner.Kind}
			idx.add(file.ImportPath, owner)
			for _, n := range order {
				idx.add(file.ImportPath, n)
				idx.addMember(owner.Id, n)
				nodeByID[n.Id] = n
				kinds[n.Id] = n.Kind
			}
			rs := newResolver(idx, kinds, nodeByID, nil, nil)
			rs.declareSets()
			out := rs.resolveCall(file, call, false)
			if out.target != "method:add:String" {
				t.Errorf("order %v: resolved to %q (set %v), want method:add:String", ids(order), out.target, out.set != nil)
			}
			return
		}
		for i, n := range rest {
			next := append(append([]*schema.Node(nil), rest[:i]...), rest[i+1:]...)
			permute(append(append([]*schema.Node(nil), order...), n), next)
		}
	}
	permute(nil, overloads)
}

func ids(nodes []*schema.Node) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Id
	}
	return out
}

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
