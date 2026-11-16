package indexer

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// syncStep is one edit of a tree: src written to path, or path deleted
// when src is nil.
type syncStep struct {
	path string
	src  *string
}

func edit(path, src string) syncStep { return syncStep{path: path, src: &src} }
func drop(path string) syncStep      { return syncStep{path: path} }

func TestCIncludeSyncEquivalence(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		steps []syncStep
		check func(t *testing.T, g *overloadGraph)
	}{
		{
			name:  "A definition appears after the call",
			files: map[string]string{"add.h": headerPrototypeTree()["add.h"], "main.c": headerPrototypeTree()["main.c"]},
			steps: []syncStep{edit("add.c", headerPrototypeTree()["add.c"])},
			check: calls("main.c", "run", "add.c function add definition"),
		},
		{
			name:  "A header loses its prototype",
			files: headerPrototypeTree(),
			steps: []syncStep{edit("add.h", "\n")},
			check: calls("main.c", "run"),
		},
		{
			name:  "An include is removed",
			files: headerPrototypeTree(),
			steps: []syncStep{edit("main.c", "int run(void) { return add(1, 2); }\n")},
			check: func(t *testing.T, g *overloadGraph) {
				if got := includesFrom(g)["main.c"]; len(got) != 0 {
					t.Errorf("main.c includes %q, want nothing", got)
				}
			},
		},
		{
			name:  "A header appears after its include",
			files: map[string]string{"src/main.c": "#include \"add.h\"\nint run(void) { return add(1, 2); }\n"},
			steps: []syncStep{edit("src/add.h", "int add(int a, int b);\n")},
			check: calls("src/main.c", "run", "src/add.h function add prototype"),
		},
		{
			name: "An include cycle in a sync",
			files: map[string]string{
				"a.h": "#include \"b.h\"\n", "b.h": "#include \"a.h\"\nvoid f(void);\n",
				"main.c": "#include \"a.h\"\nvoid run(void) { f(); }\n",
			},
			steps: []syncStep{edit("b.h", "#include \"a.h\"\nvoid f(void);\nvoid g(void);\n")},
			check: calls("main.c", "run", "b.h function f prototype"),
		},
		{
			name:  "A header is deleted",
			files: headerPrototypeTree(),
			steps: []syncStep{drop("add.h")},
			check: func(t *testing.T, g *overloadGraph) {
				for file, targets := range includesFrom(g) {
					if slices.Contains(targets, "add.h") {
						t.Errorf("%s still includes the deleted add.h", file)
					}
				}
				calls("main.c", "run")(t, g)
			},
		},
		{
			name: "A header moves to the repository root",
			files: map[string]string{
				"src/main.c": "#include \"add.h\"\nint run(void) { return 0; }\n", "src/add.h": "int add(int a, int b);\n",
			},
			steps: []syncStep{drop("src/add.h"), edit("add.h", "int add(int a, int b);\n")},
			check: includes("src/main.c", "add.h"),
		},
		{
			name: "An indirect header is deleted",
			files: map[string]string{
				"main.c": "#include \"api.h\"\nint run(void) { return add(1, 2); }\n", "api.h": "#include \"add.h\"\n",
				"add.h": "int add(int a, int b);\n", "add.c": "#include \"add.h\"\nint add(int a, int b) { return a + b; }\n",
			},
			steps: []syncStep{drop("add.h")},
			check: calls("main.c", "run"),
		},
		{
			name:  "A header appears at the root",
			files: map[string]string{"src/main.c": "#include \"util/str.h\"\nint run(void) { return 0; }\n"},
			steps: []syncStep{edit("util/str.h", "int len(const char *s);\n")},
			check: includes("src/main.c", "util/str.h"),
		},
		{
			name: "A header's own include moves",
			files: map[string]string{
				"main.c":    "#include \"src/api.h\"\nint run(void) { return add(1, 2); }\n",
				"src/api.h": "#include \"add.h\"\n", "add.h": "int add(int a, int b);\n",
			},
			steps: []syncStep{edit("src/add.h", "int add(int a);\n")},
			check: calls("main.c", "run", "src/add.h function add prototype"),
		},
		{
			// A struct is no C function, so no changed set reaches main.cpp:
			// only the include walk of the second wave does.
			name: "A header's own include moves, for a C++ struct",
			files: map[string]string{
				"main.cpp":    "#include \"src/api.hpp\"\nvoid run() { S(1); }\n",
				"src/api.hpp": "#include \"s.hpp\"\n", "s.hpp": "struct S { int x; };\n",
			},
			steps: []syncStep{edit("src/s.hpp", "struct S { int y; };\n")},
			check: calls("main.cpp", "run", "src/s.hpp struct S"),
		},
		{
			name: "A nearer header appears",
			files: map[string]string{
				"src/main.c": "#include \"util/str.h\"\nint run(void) { return 0; }\n", "util/str.h": "int len(const char *s);\n",
			},
			steps: []syncStep{edit("src/util/str.h", "int len(const char *s);\n")},
			check: includes("src/main.c", "src/util/str.h"),
		},
		{
			name: "A definition loses its body",
			files: map[string]string{
				"p.h": "int add(int a, int b);\n", "d.c": "int add(int a, int b) { return a + b; }\n",
				"main.c": "#include \"p.h\"\nint run(void) { return add(1, 2); }\n",
			},
			steps: []syncStep{edit("d.c", "int add(int a, int b);\n")},
			check: calls("main.c", "run", "p.h function add prototype"),
		},
		{
			name: "A prototype gains its body",
			files: map[string]string{
				"p.h": "int add(int a, int b);\n", "d.c": "int add(int a, int b);\n",
				"main.c": "#include \"p.h\"\nint run(void) { return add(1, 2); }\n",
			},
			steps: []syncStep{edit("d.c", "int add(int a, int b) { return a + b; }\n")},
			check: calls("main.c", "run", "d.c function add definition"),
		},
		{
			name: "A header with no symbols",
			files: map[string]string{
				"main.c": "#include \"api.h\"\nint main(void) { return add(1, 2); }\n", "api.h": "#include \"add.h\"\n",
				"add.h": "int add(int a, int b);\n",
			},
			steps: []syncStep{edit("main.c", "#include \"api.h\"\nint main(void) { int y = 0; return add(1, 2) + y; }\n")},
			check: calls("main.c", "main", "add.h function add prototype"),
		},
		{
			name:  "A definition loses static",
			files: with(headerPrototypeTree(), "add.c", "#include \"add.h\"\nstatic int add(int a, int b) { return a + b; }\n"),
			steps: []syncStep{edit("add.c", "#include \"add.h\"\nint    add(int a, int b) { return a + b; }\n")},
			check: calls("main.c", "run", "add.c function add definition"),
		},
		{
			name:  "A second definition is deleted",
			files: with(headerPrototypeTree(), "add2.c", "int add(int a, int b) { return b + a; }\n"),
			steps: []syncStep{drop("add2.c")},
			check: calls("main.c", "run", "add.c function add definition"),
		},
		{
			name:  "A second definition appears",
			files: headerPrototypeTree(),
			steps: []syncStep{edit("add2.c", "int add(int a, int b) { return b + a; }\n")},
			check: calls("main.c", "run", "add.h function add prototype"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, storeDir := testtree.Write(t, tc.files), t.TempDir()
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			for _, s := range tc.steps {
				if s.src == nil {
					removeAt(t, root, s.path)
				} else {
					writeFile(t, filepath.Join(root, s.path), *s.src)
				}
			}
			if _, err := Sync(root, storeDir, Options{}); err != nil {
				t.Fatalf("Sync: %v", err)
			}
			requireSyncEqualsRun(t, root, storeDir)
			tc.check(t, readGraph(t, storeDir))
		})
	}
}

// calls checks the targets of the calls from the function name in file.
func calls(file, name string, want ...string) func(*testing.T, *overloadGraph) {
	return func(t *testing.T, g *overloadGraph) {
		t.Helper()
		if got := callsFrom(t, g, file, name); !slices.Equal(got, want) {
			t.Errorf("calls from %s = %q, want %q", name, got, want)
		}
	}
}

// includes checks the files that file includes.
func includes(file string, want ...string) func(*testing.T, *overloadGraph) {
	return func(t *testing.T, g *overloadGraph) {
		t.Helper()
		if got := includesFrom(g)[file]; !slices.Equal(got, want) {
			t.Errorf("%s includes %q, want %q", file, got, want)
		}
	}
}
