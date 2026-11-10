package resolve

import (
	"os"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/extract"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// fixtureRoot is the shared Go fixture module of internal/indexer.
const fixtureRoot = "../testdata/gofixture"

// resolveRefs resolves results alone, through Types, NewHierarchy, Calls
// and Link, as a full index does.
func resolveRefs(results []goextract.FileResult, modulePath string) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolved int) {
	b := symbols.NewBuilder()
	b.AddResults(declarationsOf(results))
	view := b.Freeze()
	types := make([]TypeOutput, len(results))
	var typeEdges []*schema.Edge
	for i := range results {
		types[i] = Types(view, modulePath, results[i])
		typeEdges = append(typeEdges, types[i].Edges()...)
	}
	h := symbols.NewHierarchy(view, typeEdges)
	calls := make([]CallOutput, len(results))
	for i := range results {
		calls[i] = Calls(view, h, results[i])
	}
	g, err := Link(view, h, types, calls)
	if err != nil {
		panic(err)
	}
	return g.Nodes, g.PackageNodes, g.Edges, g.Files, g.Unresolved
}

// declarationsOf returns the declarations of each result, with the default
// visibility of its language, as package indexer builds them.
func declarationsOf(results []goextract.FileResult) []symbols.Declarations {
	decls := make([]symbols.Declarations, len(results))
	for i, r := range results {
		spec, _ := extract.ByID(r.Language)
		decls[i] = symbols.DeclarationsOf(r, spec.DefaultVisibility)
	}
	return decls
}

// extractGo extracts one Go source file under the module example.com/p
// through extract.File, the path of a full index, so its nodes carry what
// that function stamps on them.
func extractGo(t testing.TB, p parser.Parser, relPath, src string) goextract.FileResult {
	t.Helper()
	return extractGoIn(t, p, "example.com/p", relPath, src)
}

// extractGoIn is extractGo under the module key moduleKey.
func extractGoIn(t testing.TB, p parser.Parser, moduleKey, relPath, src string) goextract.FileResult {
	t.Helper()
	spec, ok := extract.ByID("go")
	if !ok {
		t.Fatal("no registered language go")
	}
	r, err := extract.File(spec, p, moduleKey, relPath, []byte(src))
	if err != nil {
		t.Fatalf("extract.File(%s): %v", relPath, err)
	}
	return r
}

// fixtureResults returns pass 1's results and module path for the shared
// multi-package testdata/gofixture tree.
func fixtureResults(t *testing.T) ([]goextract.FileResult, string) {
	t.Helper()
	return extractTree(t, fixtureRoot)
}

// extractTree returns pass 1's results and module path for the tree at
// root, extracted one file at a time.
func extractTree(t testing.TB, root string) ([]goextract.FileResult, string) {
	t.Helper()
	files, modulePath, err := discover.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	results := make([]goextract.FileResult, len(files))
	for i, f := range files {
		spec, ok := extract.ByID(f.Language)
		if !ok {
			t.Fatalf("no registered language %q", f.Language)
		}
		p, err := spec.NewParser()
		if err != nil {
			t.Fatalf("NewParser(%s): %v", f.Language, err)
		}
		src, err := os.ReadFile(f.AbsPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		results[i], err = extract.File(spec, p, f.ImportPath, f.RelPath, src)
		p.Close()
		if err != nil {
			t.Fatalf("extract.File(%s): %v", f.RelPath, err)
		}
		results[i].MtimeUnixNs, results[i].SizeBytes = f.MtimeUnixNs, f.SizeBytes
	}
	return results, modulePath
}

// newTestResolver freezes b and returns a resolver over it, with the
// hierarchy of b's declarations and no type-level edges.
func newTestResolver(b *symbols.Builder) *resolver {
	view := b.Freeze()
	return newResolver(view, symbols.NewHierarchy(view, nil))
}

// unresolvedNameStrings renders the names f waits on as "module name".
func unresolvedNameStrings(f *schema.File) []string {
	var out []string
	for _, u := range f.GetUnresolvedNames() {
		out = append(out, u.GetModuleKey()+" "+u.GetName())
	}
	return out
}
