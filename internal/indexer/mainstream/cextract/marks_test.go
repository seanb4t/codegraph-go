package cextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestCFunctionMarks pins the prototype mark, the internal linkage of a
// static free function and the quoted mark of an include (design D4).
func TestCFunctionMarks(t *testing.T) {
	c, cpp := newCTestParser(t), newCppTestParser(t)
	extract := func(t *testing.T, file, src string) []*schema.Node {
		t.Helper()
		p := c
		if languageForExt(file) == "cpp" {
			p = cpp
		}
		res, err := Extract(p, file, file, []byte(src))
		if err != nil || res.Err != nil {
			t.Fatalf("Extract(%s): %v / %v", file, err, res.Err)
		}
		var out []*schema.Node
		for _, en := range res.Nodes {
			out = append(out, en.Node)
		}
		return out
	}
	named := func(nodes []*schema.Node, kind, name string) []*schema.Node {
		var out []*schema.Node
		for _, n := range nodes {
			if n.Kind == kind && n.Name == name {
				out = append(out, n)
			}
		}
		return out
	}
	// want checks every node of kind and name, and that there is one.
	want := func(t *testing.T, nodes []*schema.Node, kind, name string, local bool) {
		t.Helper()
		got := named(nodes, kind, name)
		if len(got) == 0 {
			t.Fatalf("no %s node %s", kind, name)
		}
		vis, exported := "public", true
		if local {
			vis, exported = "module", false
		}
		for _, n := range got {
			if n.Visibility != vis || n.IsExported != exported {
				t.Errorf("%s at line %d: visibility %q, exported %v; want %q, %v", name, n.StartLine, n.Visibility, n.IsExported, vis, exported)
			}
		}
	}
	definition := func(t *testing.T, nodes []*schema.Node, name string) *schema.Node {
		t.Helper()
		for _, n := range named(nodes, goextract.KindFunction, name) {
			if !n.Prototype {
				return n
			}
		}
		t.Fatalf("no definition of %s", name)
		return nil
	}

	t.Run("A static helper", func(t *testing.T) {
		nodes := extract(t, "add.c", "static int clamp(int x) { return x; }\nint add(int a, int b) { return clamp(a) + b; }\n")
		want(t, nodes, goextract.KindFunction, "clamp", true)
		want(t, nodes, goextract.KindFunction, "add", false)
	})
	t.Run("A static forward declaration", func(t *testing.T) {
		nodes := extract(t, "util.c", "static int h(int);\nint h(int x) { return x; }\n")
		if got := named(nodes, goextract.KindFunction, "h"); len(got) != 1 {
			t.Fatalf("h has %d nodes, want 1", len(got))
		}
		want(t, nodes, goextract.KindFunction, "h", true)
	})
	t.Run("A static declaration outside the definition's #if", func(t *testing.T) {
		nodes := extract(t, "util.c", "static int h(int);\n#ifdef X\nint h(int x) { return x; }\n#endif\n")
		if d := definition(t, nodes, "h"); d.Visibility != "module" || d.IsExported {
			t.Errorf("definition of h: visibility %q, exported %v; want module, false", d.Visibility, d.IsExported)
		}
	})
	t.Run("A C++ static member function", func(t *testing.T) {
		nodes := extract(t, "s.cpp", "class S { public: static int f() { return 0; } };\n")
		want(t, nodes, goextract.KindMethod, "f", false)
	})
	t.Run("an earlier static declaration of the name in C", func(t *testing.T) {
		nodes := extract(t, "h.c", "static int h();\nint h(int x) { return x; }\n")
		if d := definition(t, nodes, "h"); d.Visibility != "module" || d.IsExported {
			t.Errorf("definition of h: visibility %q, exported %v; want module, false", d.Visibility, d.IsExported)
		}
	})
	t.Run("a static declaration of another name", func(t *testing.T) {
		nodes := extract(t, "h.c", "static int g(int);\nint h(int x) { return x; }\n")
		want(t, nodes, goextract.KindFunction, "h", false)
	})
	t.Run("a static C++ declaration with another key", func(t *testing.T) {
		nodes := extract(t, "s.cpp", "static int h(long);\nint h(int x) { return x; }\n")
		if d := definition(t, nodes, "h"); d.Visibility != "public" || !d.IsExported {
			t.Errorf("definition of h: visibility %q, exported %v; want public, true", d.Visibility, d.IsExported)
		}
	})
	t.Run("a prototype in another #if block", func(t *testing.T) {
		nodes := extract(t, "h.c", "static int h(int);\n#ifdef A\nint h(int);\n#endif\n")
		if got := named(nodes, goextract.KindFunction, "h"); len(got) != 2 {
			t.Fatalf("h has %d nodes, want 2", len(got))
		}
		want(t, nodes, goextract.KindFunction, "h", true)
	})
	t.Run("extern and inline keep their linkage", func(t *testing.T) {
		nodes := extract(t, "add.h", "extern int add(int a, int b);\ninline int f(void) { return 0; }\n")
		want(t, nodes, goextract.KindFunction, "add", false)
		want(t, nodes, goextract.KindFunction, "f", false)
	})
	t.Run("an out-of-line method after a static function of its name", func(t *testing.T) {
		nodes := extract(t, "s.cpp", "static int f(int);\nint S::f(int x) { return x; }\n")
		want(t, nodes, goextract.KindMethod, "f", false)
	})
	t.Run("a prototype and a definition", func(t *testing.T) {
		header := extract(t, "add.h", "int add(int a, int b);\n")
		if got := named(header, goextract.KindFunction, "add"); len(got) != 1 || !got[0].Prototype {
			t.Errorf("header add: %v, want one prototype", got)
		}
		source := extract(t, "add.c", "int add(int a, int b) { return a + b; }\n")
		if got := named(source, goextract.KindFunction, "add"); len(got) != 1 || got[0].Prototype {
			t.Errorf("source add: %v, want one definition", got)
		}
	})
	t.Run("only a quoted include is marked", func(t *testing.T) {
		res, err := Extract(c, "main.c", "main.c", []byte("#include <stdio.h>\n#include HEADER\n#include \"a.h\"\n"))
		if err != nil || res.Err != nil {
			t.Fatalf("Extract: %v / %v", err, res.Err)
		}
		quoted := map[string]bool{}
		for _, u := range res.Unresolved {
			if u.Kind == goextract.RefKindImports {
				quoted[u.Name] = u.Quoted
			}
		}
		want := map[string]bool{"stdio.h": false, "HEADER": false, "a.h": true}
		if len(quoted) != len(want) {
			t.Fatalf("includes = %v, want %v", quoted, want)
		}
		for name, q := range want {
			if got, ok := quoted[name]; !ok || got != q {
				t.Errorf("include %q: quoted %v (present %v), want %v", name, got, ok, q)
			}
		}
	})
}
