package indexer

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// writeTree writes files under root, each path relative to it.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		writeFixtureFile(t, root, rel, src)
	}
}

// indexTree indexes files in a new directory and returns its graph.
func indexTree(t *testing.T, files map[string]string) *overloadGraph {
	t.Helper()
	root, storeDir := t.TempDir(), t.TempDir()
	writeTree(t, root, files)
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return readGraph(t, storeDir)
}

// includesFrom maps each file to the files its imports edges target,
// one entry per edge.
func includesFrom(g *overloadGraph) map[string][]string {
	byID := make(map[string]*schema.Node, len(g.nodes))
	for _, n := range g.nodes {
		byID[n.Id] = n
	}
	out := make(map[string][]string)
	for _, e := range g.edges {
		if e.Kind != goextract.RefKindImports {
			continue
		}
		from, to := byID[e.Source], byID[e.Target]
		target := e.Target
		if to != nil {
			target = to.FilePath
		}
		out[from.FilePath] = append(out[from.FilePath], target)
	}
	for _, targets := range out {
		slices.Sort(targets)
	}
	return out
}

// callsFrom returns "file kind name" of each target of the calls edges
// from the function name in file, sorted.
func callsFrom(t *testing.T, g *overloadGraph, file, name string) []string {
	t.Helper()
	src := g.node(t, file, goextract.KindFunction, name, "")
	byID := make(map[string]*schema.Node, len(g.nodes))
	for _, n := range g.nodes {
		byID[n.Id] = n
	}
	var out []string
	for _, e := range g.edges {
		if e.Kind != goextract.RefKindCalls || e.Source != src.Id {
			continue
		}
		n := byID[e.Target]
		desc := describeTarget(n)
		if n != nil && n.Kind == goextract.KindFunction {
			if n.Prototype {
				desc += " prototype"
			} else {
				desc += " definition"
			}
		}
		out = append(out, desc)
	}
	slices.Sort(out)
	return out
}

func describeTarget(n *schema.Node) string {
	if n == nil {
		return "<missing>"
	}
	return n.FilePath + " " + n.Kind + " " + n.Name
}

func TestCIncludeLinks(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  map[string][]string
	}{
		{
			name:  "A header beside the source file",
			files: map[string]string{"src/main.c": "#include \"add.h\"\n", "src/add.h": "int add(int a, int b);\n"},
			want:  map[string][]string{"src/main.c": {"src/add.h"}},
		},
		{
			name:  "A path from the repository root",
			files: map[string]string{"src/main.c": "#include \"util/str.h\"\n", "util/str.h": "int len(const char *s);\n"},
			want:  map[string][]string{"src/main.c": {"util/str.h"}},
		},
		{
			name: "A sibling include directory",
			files: map[string]string{
				"src/main.c":    "#include \"../include/add.h\"\n#include \"../../x.h\"\n",
				"include/add.h": "int add(int a, int b);\n", "x.h": "int x(void);\n",
			},
			want: map[string][]string{"src/main.c": {"include/add.h"}},
		},
		{
			name: "An absolute path",
			files: map[string]string{
				"src/main.c": "#include \"/usr/include/foo.h\"\n", "src/usr/include/foo.h": "int foo(void);\n",
			},
			want: map[string][]string{},
		},
		{
			name:  "A system header",
			files: map[string]string{"src/main.c": "#include <stdio.h>\n", "stdio.h": "int puts(const char *s);\n"},
			want:  map[string][]string{},
		},
		{
			name: "An inner dot segment",
			files: map[string]string{
				"src/main.c": "#include \"util/../add.h\"\n", "src/util/str.h": "int len(const char *s);\n",
				"src/add.h": "int add(int a, int b);\n",
			},
			want: map[string][]string{},
		},
		{
			name: "A header that includes itself",
			files: map[string]string{
				"a.h": "#include \"a.h\"\nvoid f(void);\n", "main.c": "#include \"a.h\"\nvoid run(void) { f(); }\n",
			},
			want: map[string][]string{"main.c": {"a.h"}},
		},
		{
			name: "two directives that name one header",
			files: map[string]string{
				"src/main.c": "#include \"add.h\"\n#include \"./add.h\"\n", "src/add.h": "int add(int a, int b);\n",
			},
			want: map[string][]string{"src/main.c": {"src/add.h"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := indexTree(t, tc.files)
			if got := includesFrom(g); !maps.EqualFunc(got, tc.want, slices.Equal[[]string]) {
				t.Errorf("imports edges = %q, want %q", got, tc.want)
			}
			for _, n := range g.nodes {
				if n.Kind == resolve.KindPackage {
					t.Errorf("package node %q, want none for a C include", n.QualifiedName)
				}
			}
			if tc.name == "A header that includes itself" {
				if got, want := callsFrom(t, g, "main.c", "run"), []string{"a.h function f prototype"}; !slices.Equal(got, want) {
					t.Errorf("calls from run = %q, want %q", got, want)
				}
			}
		})
	}
}

// removeAt deletes one file of root.
func removeAt(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, rel)); err != nil {
		t.Fatal(err)
	}
}

// headerPrototypeTree is a header prototype, its definition in another
// file, and a caller that includes the header.
func headerPrototypeTree() map[string]string {
	return map[string]string{
		"add.h":  "int add(int a, int b);\n",
		"add.c":  "#include \"add.h\"\nint add(int a, int b) { return a + b; }\n",
		"main.c": "#include \"add.h\"\nint run(void) { return add(1, 2); }\n",
	}
}

func with(files map[string]string, more ...string) map[string]string {
	for i := 0; i+1 < len(more); i += 2 {
		files[more[i]] = more[i+1]
	}
	return files
}

func TestCIncludeCalls(t *testing.T) {
	cases := []struct {
		name   string
		files  map[string]string
		caller [2]string
		want   []string
	}{
		// A C call resolves through the files its file includes.
		{
			name: "A function defined in a header",
			files: map[string]string{
				"util.h": "static inline int twice(int x) { return x * 2; }\n",
				"main.c": "#include \"util.h\"\nint run(void) { return twice(2); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"util.h function twice definition"},
		},
		{
			name: "An include of an include",
			files: map[string]string{
				"main.c": "#include \"api.h\"\nint run(void) { return add(1, 2); }\n",
				"api.h":  "#include \"add.h\"\n", "add.h": "int add(int a, int b);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"add.h function add prototype"},
		},
		{
			name: "A struct tag in a nearer header",
			files: map[string]string{
				"main.c": "#include \"a.h\"\nstruct add { int x; };\nint run(void) { return add(1, 2); }\n",
				"a.h":    "#include \"b.h\"\nstruct add { int y; };\n", "b.h": "int add(int a, int b);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"b.h function add prototype"},
		},
		{
			name: "Include guards that form a cycle",
			files: map[string]string{
				"a.h": "#include \"b.h\"\n", "b.h": "#include \"a.h\"\nvoid f(void);\n",
				"main.c": "#include \"a.h\"\nvoid run(void) { f(); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"b.h function f prototype"},
		},
		{
			name: "The nearest header wins",
			files: map[string]string{
				"main.c": "#include \"a.h\"\n#include \"b.h\"\nint run(void) { return f(1); }\n",
				"a.h":    "int f(int x);\n", "b.h": "#include \"c.h\"\n", "c.h": "int f(int x);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"a.h function f prototype"},
		},
		{
			name: "Two headers at one depth",
			files: map[string]string{
				"main.c": "#include \"a.h\"\n#include \"b.h\"\nint run(void) { return f(1, 2); }\n",
				"a.h":    "int f(int x);\n", "b.h": "int f(int x, int y);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"b.h function f prototype"},
		},
		{
			name: "A header at depth 1 and depth 2",
			files: map[string]string{
				"main.c": "#include \"a.h\"\n#include \"b.h\"\nint run(void) { return f(1, 2); }\n",
				"a.h":    "#include \"b.h\"\nint f(int x);\n", "b.h": "int f(int x, int y);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"b.h function f prototype"},
		},
		{
			name: "The nearest header has the wrong arity",
			files: map[string]string{
				"main.c": "#include \"a.h\"\nint run(void) { return f(1, 2); }\n",
				"a.h":    "#include \"b.h\"\nint f(int x);\n", "b.h": "int f(int x, int y);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"a.h function f prototype"},
		},
		{
			name: "Every candidate at the first depth has the wrong arity",
			files: map[string]string{
				"main.c": "#include \"a.h\"\n#include \"b.h\"\nint run(void) { return f(1, 2); }\n",
				"a.h":    "#include \"c.h\"\nint f(int x);\n", "b.h": "int f(int x);\n", "c.h": "int f(int x, int y);\n",
			},
			caller: [2]string{"main.c", "run"}, want: nil,
		},
		{
			name: "A name the file does not include",
			files: map[string]string{
				"other.c": "int add(int a, int b) { return a + b; }\n",
				"main.c":  "int run(void) { return add(1, 2); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: nil,
		},

		// A C prototype resolves to its definition in another file.
		{
			name: "The repro from #259", files: headerPrototypeTree(),
			caller: [2]string{"main.c", "run"}, want: []string{"add.c function add definition"},
		},
		{
			name: "A library prototype",
			files: map[string]string{
				"vendor.h": "int vendor_init(void);\n",
				"main.c":   "#include \"vendor.h\"\nint run(void) { return vendor_init(); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"vendor.h function vendor_init prototype"},
		},
		{
			name: "One definition for each platform",
			files: map[string]string{
				"io.h":       "int open_port(int n);\n",
				"io_posix.c": "int open_port(int n) { return n; }\n", "io_win.c": "int open_port(int n) { return -n; }\n",
				"main.c": "#include \"io.h\"\nint run(void) { return open_port(1); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"io.h function open_port prototype"},
		},
		{
			name:   "A static definition of the same name",
			files:  with(headerPrototypeTree(), "add.c", "static int add(int a, int b) { return a + b; }\n"),
			caller: [2]string{"main.c", "run"}, want: []string{"add.h function add prototype"},
		},
		{
			name: "A prototype under an include guard",
			files: with(headerPrototypeTree(), "add.h", "#ifndef ADD_H\n#define ADD_H\nint add(int a, int b);\n#endif\n",
				"add.c", "int add(int a, int b) { return a + b; }\n"),
			caller: [2]string{"main.c", "run"}, want: []string{"add.c function add definition"},
		},
		{
			name: "Pointer spacing and unnamed parameters",
			files: map[string]string{
				"str.h": "int count(const char *);\n", "str.c": "int count(const char* s) { return s[0]; }\n",
				"main.c": "#include \"str.h\"\nint run(void) { return count(\"x\"); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"str.c function count definition"},
		},
		{
			name: "Two definitions under #if in one file",
			files: map[string]string{
				"io.h":   "int open_port(int n);\n",
				"io.c":   "#ifdef A\nint open_port(int n) { return n; }\n#else\nint open_port(int n) { return -n; }\n#endif\n",
				"main.c": "#include \"io.h\"\nint run(void) { return open_port(1); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"io.h function open_port prototype"},
		},
		{
			name: "A C++ default argument",
			files: map[string]string{
				"f.hpp": "int f(int a, int b = 0);\n", "f.cpp": "int f(int a, int b) { return a + b; }\n",
				"main.cpp": "#include \"f.hpp\"\nint run() { return f(1); }\n",
			},
			caller: [2]string{"main.cpp", "run"}, want: []string{"f.cpp function f definition"},
		},
		{
			name:   "A static prototype",
			files:  with(headerPrototypeTree(), "add.h", "static int add(int a, int b);\n"),
			caller: [2]string{"main.c", "run"}, want: []string{"add.h function add prototype"},
		},
		{
			name: "A definition in the prototype's own file",
			files: with(headerPrototypeTree(), "add.h",
				"int add(int a, int b);\n#ifdef X\nint add(int a, int b) { return a + b; }\n#endif\n"),
			caller: [2]string{"main.c", "run"}, want: []string{"add.h overload_set add"},
		},
		{
			name: "A C++ method of the same name",
			files: map[string]string{
				"f.hpp": "int f(int x);\n", "s.cpp": "class S { int f(int x); };\nint S::f(int x) { return x; }\n",
				"main.cpp": "#include \"f.hpp\"\nint run() { return f(1); }\n",
			},
			caller: [2]string{"main.cpp", "run"}, want: []string{"f.hpp function f prototype"},
		},
		{
			name:   "Another signature",
			files:  with(headerPrototypeTree(), "add.c", "int add(long a, long b) { return a + b; }\n"),
			caller: [2]string{"main.c", "run"}, want: []string{"add.h function add prototype"},
		},

		// A C prototype matches its definition by one rule.
		{
			name: "An array parameter with and without a name",
			files: map[string]string{
				"arr.h": "int sum(int[10]);\n", "arr.c": "int sum(int a[10]) { return a[0]; }\n",
				"main.c": "#include \"arr.h\"\nint v[10];\nint run(void) { return sum(v); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"arr.c function sum definition"},
		},
		{
			name: "A function-pointer parameter with and without names",
			files: map[string]string{
				"cb.h": "void on(void (*)(int));\n", "cb.c": "void on(void (*cb)(int x)) {}\n",
				"main.c": "#include \"cb.h\"\nvoid f(int x) {}\nvoid run(void) { on(f); }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"cb.c function on definition"},
		},

		// The edges of the walk.
		{
			name: "a C++ call of a struct name",
			files: map[string]string{
				"s.hpp": "struct S { int x; };\n", "main.cpp": "#include \"s.hpp\"\nvoid run() { S(1); }\n",
			},
			caller: [2]string{"main.cpp", "run"}, want: []string{"s.hpp struct S"},
		},
		{
			name: "a prototype in the calling file",
			files: map[string]string{
				"main.c": "int add(int a, int b);\nint run(void) { return add(1, 2); }\n",
				"add.c":  "int add(int a, int b) { return a + b; }\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"add.c function add definition"},
		},
		{
			name: "the own file decides before the walk",
			files: map[string]string{
				"main.c": "#include \"a.h\"\nint f(int x);\nint run(void) { return f(1, 2); }\n",
				"a.h":    "int f(int x, int y);\n",
			},
			caller: [2]string{"main.c", "run"}, want: []string{"main.c function f prototype"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := indexTree(t, tc.files)
			if got := callsFrom(t, g, tc.caller[0], tc.caller[1]); !slices.Equal(got, tc.want) {
				t.Errorf("calls from %s = %q, want %q", tc.caller[1], got, tc.want)
			}
		})
	}

	oneNode := []struct{ name, file, src, fn string }{
		{"Pointer spacing in one file", "str.c", "int count(const char *);\nint count(const char* s) { return s[0]; }\n", "count"},
		{"A default argument in one file", "s.cpp", "int f(int a, int b = 0);\nint f(int a, int b) { return a + b; }\n", "f"},
	}
	for _, tc := range oneNode {
		t.Run(tc.name, func(t *testing.T) {
			g := indexTree(t, map[string]string{tc.file: tc.src})
			g.node(t, tc.file, goextract.KindFunction, tc.fn, "")
		})
	}
}
