package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// lowCap is the batch cap of the tests that need a run to commit several
// batches over the tree of goTree.
const lowCap = 64 << 10

// goTree writes a module of n Go source files, each with one function that
// calls the function of the next file, and returns its root. With routes
// set, it adds three Express files, each of which registers one route, so
// that route detection has more than one file to work on.
func goTree(t *testing.T, n int, routes bool) string {
	t.Helper()
	files := make(map[string]string, n+4)
	for i := range n {
		files[fmt.Sprintf("pkg/f%04d.go", i)] = fmt.Sprintf("package pkg\n\n// F%04d calls the function of the next file.\nfunc F%04d() int {\n\treturn F%04d() + %d\n}\n", i, i, (i+1)%n, i)
	}
	if routes {
		files["package.json"] = "{\n  \"name\": \"tree\",\n  \"dependencies\": {\n    \"express\": \"^4.18.0\"\n  }\n}\n"
		for i := range 3 {
			files[fmt.Sprintf("web/app%d.ts", i)] = fmt.Sprintf("import express from \"express\";\n\nfunction handler%d() {}\n\nconst app = express();\napp.get(\"/r%d\", handler%d);\n", i, i, i)
		}
	}
	return writeFixture(t, files)
}

// storeNodes returns every node of the store at storeDir with the given
// kind and name.
func storeNodes(t *testing.T, storeDir, kind, name string) []*schema.Node {
	t.Helper()
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()
	var out []*schema.Node
	for it.Next() {
		if n := it.Node(); n.Kind == kind && n.Name == name {
			out = append(out, n)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("node iteration: %v", err)
	}
	return out
}

// recordCommits sets onCommit to record each commit of a run.
func recordCommits(t *testing.T) *[]graphstore.Commit {
	t.Helper()
	var commits []graphstore.Commit
	setHook(t, &onCommit, func(c graphstore.Commit) error {
		commits = append(commits, c)
		return nil
	})
	return &commits
}

func TestMergedPartialTypeStored(t *testing.T) {
	const csproj = "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <RootNamespace>Shop</RootNamespace>\n  </PropertyGroup>\n</Project>\n"
	part := func(modifier, base, method string) string {
		return fmt.Sprintf("namespace Shop;\n\n%spartial class Order : %s\n{\n    public int %s()\n    {\n        return 0;\n    }\n}\n", modifier, base, method)
	}
	for _, tc := range []struct {
		name           string
		modifierB      string
		wantVisibility string
		wantExported   bool
	}{
		{"B.cs is public", "public ", "public", true},
		{"no modifier on either part", "", "internal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFixture(t, map[string]string{
				"Shop.csproj": csproj,
				"IA.cs":       "namespace Shop;\n\npublic interface IA\n{\n}\n",
				"IB.cs":       "namespace Shop;\n\npublic interface IB\n{\n}\n",
				"A.cs":        part("", "IA", "Total"),
				"B.cs":        part(tc.modifierB, "IB", "Count"),
			})
			// A cap of 0 gives each record its own batch, so the shared
			// node, its base edges and the metadata are in different ones.
			setHook(t, &batchCap, 0)
			commits := recordCommits(t)
			storeDir := filepath.Join(t.TempDir(), "store")
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(*commits) < 3 {
				t.Fatalf("%d batches committed, want the records in several batches", len(*commits))
			}

			orders := storeNodes(t, storeDir, goextract.KindStruct, "Order")
			if len(orders) != 1 {
				t.Fatalf("the store holds %d Order nodes, want the one merged node", len(orders))
			}
			if got := orders[0]; got.Visibility != tc.wantVisibility || got.IsExported != tc.wantExported {
				t.Errorf("Order: visibility %q, exported %v; want %q, %v", got.Visibility, got.IsExported, tc.wantVisibility, tc.wantExported)
			}
			r, closeFn := openSnapshot(t, storeDir)
			for _, iface := range []string{"IA", "IB"} {
				target := storeNodesIn(t, r, goextract.KindInterface, iface)
				if len(target) != 1 || !hasEdge(t, r, orders[0].Id, goextract.EdgeKindImplements, target[0].Id) {
					t.Errorf("the store holds no implements edge from Order to %s, the base of one part", iface)
				}
			}
			closeFn()
			// The metadata counts equal the stored records for an id that
			// two files emit.
			requireInvariants(t, storeDir)
		})
	}
}

// storeNodesIn is storeNodes over an open reader.
func storeNodesIn(t *testing.T, r graphstore.Reader, kind, name string) []*schema.Node {
	t.Helper()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()
	var out []*schema.Node
	for it.Next() {
		if n := it.Node(); n.Kind == kind && n.Name == name {
			out = append(out, n)
		}
	}
	return out
}

func TestDuplicateRouteStored(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"package.json": "{\n  \"name\": \"dup\",\n  \"dependencies\": {\n    \"express\": \"^4.18.0\"\n  }\n}\n",
		// GET /x is registered on line 3 and again on line 7.
		"app.ts": "import express from \"express\";\nconst app = express();\napp.get(\"/x\", first);\n\nfunction first() {}\nfunction second() {}\napp.get(\"/x\", second);\n",
	})
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	routes := storeNodes(t, storeDir, goextract.KindRoute, "GET /x")
	if len(routes) != 1 {
		t.Fatalf("the store holds %d route nodes for GET /x, want 1", len(routes))
	}
	if routes[0].StartLine != 3 {
		t.Errorf("the route node has start line %d, want 3, the first registration", routes[0].StartLine)
	}
	requireInvariants(t, storeDir)
}

// inFlight counts, for each step of a run, the files that hold a token.
type inFlight struct {
	mu      sync.Mutex
	current map[string]int
	peak    map[string]int
	claims  map[string]map[int]bool
	// both is closed for a step once two files held a token at once.
	both map[string]chan struct{}
}

func newInFlight() *inFlight {
	f := &inFlight{current: map[string]int{}, peak: map[string]int{}, claims: map[string]map[int]bool{}, both: map[string]chan struct{}{}}
	for _, step := range []string{stepExtract, stepRoutes, stepStructure, stepResolve} {
		f.claims[step] = map[int]bool{}
		f.both[step] = make(chan struct{})
	}
	return f
}

// claim notes a claim. The first claim of each step then waits until a
// second file holds a token too, so the count reaches two in every run
// of a loop that lets two files in at once. A loop that lets only one
// file in never gets past its first claim, and the test times out.
func (f *inFlight) claim(step string, file int) {
	f.mu.Lock()
	f.current[step]++
	f.peak[step] = max(f.peak[step], f.current[step])
	first := len(f.claims[step]) == 0
	f.claims[step][file] = true
	if f.current[step] == 2 {
		select {
		case <-f.both[step]:
		default:
			close(f.both[step])
		}
	}
	both := f.both[step]
	f.mu.Unlock()
	if first {
		<-both
	}
}

func (f *inFlight) release(step string) {
	f.mu.Lock()
	f.current[step]--
	f.mu.Unlock()
}

func TestWorkInFlightBound(t *testing.T) {
	const workers = 2
	root := goTree(t, 500, true)
	discovery, err := discover.DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	files := discovery.Files
	routeFiles := 0
	for _, f := range files {
		if f.Language == "typescript" {
			routeFiles++
		}
	}
	if len(files) != 503 || routeFiles != 3 {
		t.Fatalf("the tree has %d files, %d of them TypeScript; want 503 and 3", len(files), routeFiles)
	}

	flight := newInFlight()
	var mu sync.Mutex
	var problems []string
	problem := func(format string, args ...any) {
		mu.Lock()
		problems = append(problems, fmt.Sprintf(format, args...))
		mu.Unlock()
	}
	staged := make(map[int]bool)
	setHook(t, &onClaim, func(step string, file int) error {
		flight.claim(step, file)
		return nil
	})
	// At each release, the artifact of the step for that file exists.
	setHook(t, &onRelease, func(step string, file int, st *runState) {
		flight.release(step)
		switch step {
		case stepExtract:
			r, err := st.spill.Read(st.offsets[file])
			if err != nil || r.RelPath != files[file].RelPath {
				problem("pass 1, file %d: the spill holds the record of %q, %v; want %q", file, r.RelPath, err, files[file].RelPath)
			}
		case stepRoutes:
			if len(st.routeNodes[file]) != 1 {
				problem("route detection, file %d: the route result holds %d route nodes of the file, want 1", file, len(st.routeNodes[file]))
			}
		case stepStructure:
			// The route result is one more record, after the files, and
			// it has no structural edge.
			if file < len(files) && len(st.structural[file]) == 0 {
				problem("pass 2, file %d: the structural edge list holds no edge of the file", file)
			}
		case stepResolve:
			mu.Lock()
			staged[file] = true
			mu.Unlock()
		}
	})

	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := run(context.Background(), root, storeDir, Options{Workers: workers}); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, p := range problems {
		t.Error(p)
	}
	// The route result is one more record of passes 2 and 3.
	want := map[string]int{stepExtract: len(files), stepRoutes: routeFiles, stepStructure: len(files) + 1, stepResolve: len(files) + 1}
	for step, n := range want {
		if got := flight.peak[step]; got != workers {
			t.Errorf("%s: at most %d files held a token at once, want exactly %d", step, got, workers)
		}
		if got := len(flight.claims[step]); got != n {
			t.Errorf("%s: %d files were claimed, want %d", step, got, n)
		}
	}
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	for file := range files {
		if !staged[file] {
			t.Errorf("pass 3 released no token for file %d", file)
			continue
		}
		if _, err := r.GetFile(files[file].RelPath); err != nil {
			t.Errorf("the store holds no file record for %s, which pass 3 staged: %v", files[file].RelPath, err)
		}
	}
}

func TestWriteBatchBound(t *testing.T) {
	root := goTree(t, 500, true)
	setHook(t, &batchCap, lowCap)
	commits := recordCommits(t)
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	several := 0
	for i, c := range *commits {
		if c.Records == 0 {
			t.Errorf("commit %d is empty", i)
		}
		if c.Records > 1 {
			several++
			if c.BeforeLast > lowCap {
				t.Errorf("commit %d: BeforeLast %d is above the cap %d", i, c.BeforeLast, lowCap)
			}
		}
		if c.Sync {
			t.Errorf("commit %d used Sync on the staging store", i)
		}
	}
	// Positive assertion: the cap was low enough to split the graph.
	if len(*commits) < 2 || several == 0 {
		t.Errorf("%d batches committed, %d of them with more than one record; want more than one batch and at least one with several records", len(*commits), several)
	}
	requireInvariants(t, storeDir)
}

func TestSmallGraphStillCommits(t *testing.T) {
	root := writeFixture(t, map[string]string{"one.go": "package one\n\nfunc Only() {}\n"})
	commits := recordCommits(t)
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(*commits) == 0 {
		t.Fatal("no batch committed for a tree with one file")
	}
	if nodes := storeNodes(t, storeDir, goextract.KindFunction, "Only"); len(nodes) != 1 {
		t.Errorf("the store holds %d nodes for the function of the file, want 1", len(nodes))
	}
}

func TestEmptyTreeCommitsMetadataOnly(t *testing.T) {
	// An empty directory: no source file, and no file for discovery to
	// record as excluded either.
	root := t.TempDir()
	commits := recordCommits(t)
	storeDir := filepath.Join(t.TempDir(), "store")
	stats, err := Run(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Files != 0 {
		t.Fatalf("the tree has %d source files, want none", stats.Files)
	}
	if len(*commits) != 1 || (*commits)[0].Records != 1 {
		t.Fatalf("commits = %+v, want exactly one batch with one record", *commits)
	}
	meta := readMeta(t, storeDir)
	if meta.NodeCount != 0 || meta.EdgeCount != 0 || meta.GetCoverageGeneration() != 1 {
		t.Errorf("metadata = %d nodes, %d edges, generation %d; want 0, 0 and 1", meta.NodeCount, meta.EdgeCount, meta.GetCoverageGeneration())
	}
}

// TestDuplicateEdgesCollapse indexes a call that two references make at one
// position: the store holds one edge for it, and two fresh indexes store
// the same edge.
func TestDuplicateEdgesCollapse(t *testing.T) {
	root := writeFixture(t, map[string]string{
		// The two calls of target on one line give two references with one
		// source, kind and target.
		"dup.go": "package dup\n\nfunc target() int { return 1 }\n\nfunc caller() int { return target() + target() }\n",
	})
	export := func() []byte {
		storeDir := filepath.Join(t.TempDir(), "store")
		if _, err := Run(root, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		caller := storeNodes(t, storeDir, goextract.KindFunction, "caller")
		target := storeNodes(t, storeDir, goextract.KindFunction, "target")
		if len(caller) != 1 || len(target) != 1 {
			t.Fatalf("the store holds %d caller and %d target nodes, want one of each", len(caller), len(target))
		}
		r, closeFn := openSnapshot(t, storeDir)
		it, err := r.IterateEdges(caller[0].Id)
		if err != nil {
			t.Fatalf("IterateEdges: %v", err)
		}
		calls := 0
		for it.Next() {
			if e := it.Edge(); e.Kind == goextract.RefKindCalls && e.Target == target[0].Id {
				calls++
			}
		}
		it.Close()
		closeFn()
		if calls != 1 {
			t.Fatalf("the store holds %d calls edges from caller to target, want the one collapsed edge", calls)
		}
		return exportNormalized(t, storeDir)
	}
	if first, second := export(), export(); string(first) != string(second) {
		t.Errorf("two fresh indexes of one tree store different records")
	}
}
