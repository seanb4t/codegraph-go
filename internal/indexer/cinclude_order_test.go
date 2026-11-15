package indexer

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
)

// TestCIncludeOrder resolves the same C files twice: with the include
// lines of each file in two orders, and with the files given to the
// resolver in two orders. Both give the same call edges, because the walk
// decides one depth at a time (design D3) and symbols.Pick takes the
// lowest id.
func TestCIncludeOrder(t *testing.T) {
	paths := []string{"main.c", "a.h", "b.h", "sub/c.h", "sub/d.c"}
	includes := []string{"a.h", "b.h", "c.h", "sub/c.h"}
	names := []string{"f", "g"}
	params := []string{"void", "int a", "int a, int b"}
	args := []string{"", "1", "1, 2"}
	base := t.TempDir()
	var linked int
	rapid.Check(t, func(t *rapid.T) {
		type cFile struct {
			includes []string
			body     string
		}
		files := make(map[string]cFile)
		for _, p := range paths {
			if p != "main.c" && !rapid.Bool().Draw(t, "present") {
				continue
			}
			var f cFile
			for _, inc := range includes {
				if rapid.Bool().Draw(t, "include") {
					f.includes = append(f.includes, inc)
				}
			}
			var b strings.Builder
			for range rapid.IntRange(0, 3).Draw(t, "decls") {
				if rapid.Bool().Draw(t, "static") {
					b.WriteString("static ")
				}
				name := rapid.SampledFrom(names).Draw(t, "name")
				fmt.Fprintf(&b, "int %s(%s)", name, rapid.SampledFrom(params).Draw(t, "params"))
				if rapid.Bool().Draw(t, "prototype") {
					b.WriteString(";\n")
				} else {
					b.WriteString(" { return 0; }\n")
				}
			}
			b.WriteString("int run_" + strings.NewReplacer("/", "_", ".", "_").Replace(p) + "(void) { return 0")
			for range rapid.IntRange(0, 3).Draw(t, "calls") {
				fmt.Fprintf(&b, " + %s(%s)", rapid.SampledFrom(names).Draw(t, "callee"), rapid.SampledFrom(args).Draw(t, "args"))
			}
			b.WriteString("; }\n")
			f.body = b.String()
			files[p] = f
		}

		resolveTree := func(label string, includeOrder func([]string) []string, order func([]goextract.FileResult) []goextract.FileResult) map[string]bool {
			dir, err := os.MkdirTemp(base, label)
			if err != nil {
				t.Fatalf("MkdirTemp: %v", err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			for p, f := range files {
				var src strings.Builder
				for _, inc := range includeOrder(f.includes) {
					fmt.Fprintf(&src, "#include %q\n", inc)
				}
				src.WriteString(f.body)
				full := filepath.Join(dir, p)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(full, []byte(src.String()), 0o644); err != nil {
					t.Fatalf("write %s: %v", p, err)
				}
			}
			found, modulePath, err := discover.Discover(dir)
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			results, err := Extract(found, 1)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			results = order(results)
			b := symbols.NewBuilder()
			b.AddResults(declarationsOf(results))
			g, err := resolveGraph(results, modulePath, b.Freeze())
			if err != nil {
				t.Fatalf("resolveGraph: %v", err)
			}
			pathOf := make(map[string]string)
			for _, n := range g.Nodes {
				pathOf[n.Id] = n.FilePath
			}
			calls := make(map[string]bool)
			for _, e := range g.Edges {
				if e.Kind != goextract.RefKindCalls {
					continue
				}
				calls[e.Source+" -> "+e.Target] = true
				if pathOf[e.Target] != "" && pathOf[e.Target] != pathOf[e.Source] {
					linked++
				}
			}
			return calls
		}

		asWritten := resolveTree("written", slices.Clone[[]string], func(r []goextract.FileResult) []goextract.FileResult { return r })
		shuffled := resolveTree("shuffled",
			func(inc []string) []string { return rapid.Permutation(inc).Draw(t, "includeOrder") },
			func(r []goextract.FileResult) []goextract.FileResult {
				return rapid.Permutation(r).Draw(t, "fileOrder")
			})
		if !maps.Equal(asWritten, shuffled) {
			t.Fatalf("the call edges differ between two orders:\nas written: %v\nshuffled:   %v", slices.Sorted(maps.Keys(asWritten)), slices.Sorted(maps.Keys(shuffled)))
		}
	})
	if linked == 0 {
		t.Fatal("no run resolved a call into another file")
	}
}
