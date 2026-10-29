package wtindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

const (
	benchPackages        = 50
	benchFilesPerPackage = 20
	benchFuncsPerFile    = 20
)

// benchChangedFiles are the changed-file counts the benchmarks measure:
// 1%, 5%, 10% and 25% of the module's files.
var benchChangedFiles = []int{10, 50, 100, 250}

// benchFile returns the source of file f in package p: about 280 lines,
// the size of an average Go file in this repository. Each function calls a
// function in the next file of its package and one in the next package. A
// changed file renames its first function and calls one more function, so
// the files that call the old name become dependents of the change.
func benchFile(p, f int, changed bool) string {
	next := (p + 1) % benchPackages
	var b strings.Builder
	fmt.Fprintf(&b, "package p%02d\n\nimport \"example.com/bench/p%02d\"\n\n", p, next)
	fmt.Fprintf(&b, "// T%02d holds a running total.\ntype T%02d struct {\n\ttotal int\n}\n\n", f, f)
	fmt.Fprintf(&b, "// Add adds n to the total.\nfunc (t *T%02d) Add(n int) {\n\tt.total += F%02d_1(n, n)\n}\n\n", f, f)
	for k := range benchFuncsPerFile {
		name := fmt.Sprintf("F%02d_%d", f, k)
		if changed && k == 0 {
			name += "v2"
		}
		fmt.Fprintf(&b, "// %s combines a and b with the functions it calls.\n", name)
		fmt.Fprintf(&b, "func %s(a, b int) int {\n\ttotal := a + b\n\tfor i := 0; i < b; i++ {\n\t\ttotal += i * a\n\t}\n", name)
		fmt.Fprintf(&b, "\tif total > 100 {\n\t\ttotal -= F%02d_%d(a, b)\n\t}\n", (f+1)%benchFilesPerPackage, k)
		if changed {
			fmt.Fprintf(&b, "\ttotal += F%02d_%d(b, a)\n", (f+2)%benchFilesPerPackage, k)
		}
		fmt.Fprintf(&b, "\treturn total + p%02d.F%02d_%d(b, a)\n}\n\n", next, f, k)
	}
	return b.String()
}

// writeBenchModule writes the module to root, with the first changed files
// in their changed form.
func writeBenchModule(b *testing.B, root string, changed int) {
	b.Helper()
	files := map[string]string{"go.mod": "module example.com/bench\n\ngo 1.26\n"}
	n := 0
	for p := range benchPackages {
		for f := range benchFilesPerPackage {
			files[fmt.Sprintf("p%02d/f%02d.go", p, f)] = benchFile(p, f, n < changed)
			n++
		}
	}
	for rel, src := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			b.Fatal(err)
		}
	}
}

var benchBase struct {
	once  sync.Once
	dir   string
	store string
	err   error
}

// benchMainStore returns the store of an index of the unchanged module,
// built once per test binary.
func benchMainStore(b *testing.B) string {
	b.Helper()
	benchBase.once.Do(func() {
		benchBase.dir, benchBase.err = os.MkdirTemp("", "wtindex-bench-*")
		if benchBase.err != nil {
			return
		}
		root := filepath.Join(benchBase.dir, "main")
		writeBenchModule(b, root, 0)
		benchBase.store = filepath.Join(benchBase.dir, "store")
		_, benchBase.err = indexer.Run(root, benchBase.store, indexer.Options{Quiet: true})
	})
	if benchBase.err != nil {
		b.Fatal(benchBase.err)
	}
	return benchBase.store
}

func TestMain(m *testing.M) {
	code := m.Run()
	if benchBase.dir != "" {
		os.RemoveAll(benchBase.dir)
	}
	os.Exit(code)
}

// BenchmarkSeededSync measures Prepare's foreground work: checkpointing
// the main store and syncing the copy with a worktree whose files differ
// in some number of places.
func BenchmarkSeededSync(b *testing.B) {
	mainStore := benchMainStore(b)
	for _, changed := range benchChangedFiles {
		b.Run(fmt.Sprintf("changed=%d", changed), func(b *testing.B) {
			wt := b.TempDir()
			writeBenchModule(b, wt, changed)
			for i := 0; b.Loop(); i++ {
				b.StopTimer()
				seed := filepath.Join(b.TempDir(), fmt.Sprintf("seed-%d", i))
				b.StartTimer()
				if err := graphstore.Checkpoint(mainStore, seed); err != nil {
					b.Fatal(err)
				}
				if _, err := indexer.Sync(wt, seed, indexer.Options{Quiet: true}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFullIndex measures indexing the same worktrees from scratch.
func BenchmarkFullIndex(b *testing.B) {
	for _, changed := range benchChangedFiles {
		b.Run(fmt.Sprintf("changed=%d", changed), func(b *testing.B) {
			wt := b.TempDir()
			writeBenchModule(b, wt, changed)
			for i := 0; b.Loop(); i++ {
				b.StopTimer()
				store := filepath.Join(b.TempDir(), fmt.Sprintf("store-%d", i))
				b.StartTimer()
				if _, err := indexer.Run(wt, store, indexer.Options{Quiet: true}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
