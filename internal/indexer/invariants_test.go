package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/invariant"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

const repeatfixtureRoot = "testdata/repeatfixture"

// requireInvariants fails t unless the store at storeDir passes
// invariant.Check and holds at least one node and one edge.
func requireInvariants(t *testing.T, storeDir string) invariant.Counts {
	t.Helper()
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	c, err := invariant.Check(r)
	if err != nil {
		t.Fatalf("invariant.Check:\n%v", err)
	}
	if c.Nodes == 0 || c.Edges == 0 {
		t.Fatalf("invariant.Check scanned %d nodes and %d edges, want both above zero", c.Nodes, c.Edges)
	}
	return c
}

// declaredNodes returns the nodes of kind and name that path declares,
// in source order.
func declaredNodes(t *testing.T, r graphstore.Reader, path, kind, name string) []*schema.Node {
	t.Helper()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()
	var out []*schema.Node
	for it.Next() {
		n := it.Node()
		if n.FilePath == path && n.Kind == kind && n.Name == name {
			out = append(out, n)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartLine != out[j].StartLine {
			return out[i].StartLine < out[j].StartLine
		}
		return out[i].StartCol < out[j].StartCol
	})
	return out
}

// edgeTargets returns the names of the targets of src's edges of kind.
func edgeTargets(t *testing.T, r graphstore.Reader, src, kind string) []string {
	t.Helper()
	it, err := r.IterateEdges(src)
	if err != nil {
		t.Fatalf("IterateEdges(%s): %v", src, err)
	}
	defer it.Close()
	var names []string
	for it.Next() {
		e := it.Edge()
		if e.Kind != kind {
			continue
		}
		n, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		names = append(names, n.Name)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateEdges(%s): %v", src, err)
	}
	sort.Strings(names)
	return names
}

// TestRepeatedClassesOwnTheirMethods indexes two same-named classes in one
// file and checks that each class contains its own method only, and that
// the contains range rule of invariant.Check holds.
func TestRepeatedClassesOwnTheirMethods(t *testing.T) {
	storeDir := t.TempDir()
	if _, err := Run(repeatfixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	c := requireInvariants(t, storeDir)
	if c.Nesting == 0 {
		t.Fatal("invariant.Check checked no nesting contains edge")
	}

	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	models := declaredNodes(t, r, "a.ts", goextract.KindStruct, "Model")
	if len(models) != 2 {
		t.Fatalf("got %d Model nodes, want 2", len(models))
	}
	for i, want := range []string{"load", "save"} {
		got := edgeTargets(t, r, models[i].Id, goextract.RefKindContains)
		if len(got) != 1 || got[0] != want {
			t.Errorf("Model at line %d contains %v, want [%s]", models[i].StartLine, got, want)
		}
	}
}

// TestInvariantsHoldForGoMethodsOutsideTheirType checks that the contains
// range rule leaves out Go, whose methods are declared outside the type.
func TestInvariantsHoldForGoMethodsOutsideTheirType(t *testing.T) {
	storeDir := t.TempDir()
	if _, err := Run(fixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	requireInvariants(t, storeDir)
}

// TestReferenceResolvesByScope checks that a same-file call resolves by
// lexical scope and that only the exported copy is reachable from another
// file. a.ts exports a module-level `e` and declares another `e` inside a
// callback, next to `callNested`, which calls it. b.ts imports `e`. The
// check runs after a full index and after an incremental sync that
// reparses b.ts against the store-seeded symbol index.
func TestReferenceResolvesByScope(t *testing.T) {
	root := copyFixture(t, repeatfixtureRoot)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Run("index", func(t *testing.T) { requireScopedResolution(t, storeDir) })

	path := filepath.Join(root, "b.ts")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read b.ts: %v", err)
	}
	writeFile(t, path, string(src)+"\n// edited\n")
	stats, err := Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if stats.FilesReparsed != 1 {
		t.Fatalf("Sync reparsed %d files, want only b.ts", stats.FilesReparsed)
	}
	t.Run("sync", func(t *testing.T) { requireScopedResolution(t, storeDir) })
}

func requireScopedResolution(t *testing.T, storeDir string) {
	t.Helper()
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	copies := declaredNodes(t, r, "a.ts", goextract.KindFunction, "e")
	if len(copies) != 2 {
		t.Fatalf("got %d nodes for e in a.ts, want 2", len(copies))
	}
	exported, nested := copies[0], copies[1]
	inner := declaredNodes(t, r, "a.ts", goextract.KindFunction, "callNested")
	if len(inner) != 1 {
		t.Fatalf("got %d callNested nodes, want 1", len(inner))
	}
	if !hasEdge(t, r, inner[0].Id, goextract.RefKindCalls, nested.Id) {
		t.Errorf("callNested does not call the nested e (line %d)", nested.StartLine)
	}
	if hasEdge(t, r, inner[0].Id, goextract.RefKindCalls, exported.Id) {
		t.Errorf("callNested calls the exported e (line %d) instead of the nested one", exported.StartLine)
	}
	callers := declaredNodes(t, r, "b.ts", goextract.KindFunction, "useE")
	if len(callers) != 1 {
		t.Fatalf("got %d useE nodes, want 1", len(callers))
	}
	if !hasEdge(t, r, callers[0].Id, goextract.RefKindCalls, exported.Id) {
		t.Errorf("useE does not call the exported e (line %d)", exported.StartLine)
	}
	if hasEdge(t, r, callers[0].Id, goextract.RefKindCalls, nested.Id) {
		t.Errorf("useE calls the nested e (line %d)", nested.StartLine)
	}
}

// TestNodeCountMatchesStoreAfterSyncs edits and syncs a file with repeated
// declarations ten times and checks the invariants after each sync (#188).
func TestNodeCountMatchesStoreAfterSyncs(t *testing.T) {
	root := copyFixture(t, repeatfixtureRoot)
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	requireInvariants(t, storeDir)

	path := filepath.Join(root, "a.ts")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read a.ts: %v", err)
	}
	for i := 1; i <= 10; i++ {
		writeFile(t, path, string(src)+fmt.Sprintf("\nexport function added%d() {}\n", i))
		if _, err := Sync(root, storeDir, Options{}); err != nil {
			t.Fatalf("Sync %d: %v", i, err)
		}
		t.Run(fmt.Sprintf("sync %d", i), func(t *testing.T) { requireInvariants(t, storeDir) })
	}
}

// TestFinalizedIDsReachResolution checks that the final id of a repeated
// declaration reaches the edges its body emits and the method metadata
// that the resolver stamps on its node. The two Shape.Area copies have
// different parameter lists, so they are two keys and get no ordinal.
func TestFinalizedIDsReachResolution(t *testing.T) {
	storeDir := t.TempDir()
	if _, err := Run(repeatfixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()

	copies := declaredNodes(t, r, "a.ts", goextract.KindFunction, "nextGeometry")
	if len(copies) != 3 {
		t.Fatalf("got %d nextGeometry nodes, want 3", len(copies))
	}
	if want := nodeid.OrdinalID(goextract.KindFunction, "nextGeometry", "a.ts", "()", 2); copies[1].Id != want {
		t.Fatalf("second nextGeometry id = %s, want %s", copies[1].Id, want)
	}
	measure := declaredNodes(t, r, "a.ts", goextract.KindFunction, "measure")
	if len(measure) != 1 {
		t.Fatalf("got %d measure nodes, want 1", len(measure))
	}
	if !hasEdge(t, r, copies[1].Id, goextract.RefKindCalls, measure[0].Id) {
		t.Error("the second nextGeometry has no calls edge to measure")
	}
	for _, i := range []int{0, 2} {
		if hasEdge(t, r, copies[i].Id, goextract.RefKindCalls, measure[0].Id) {
			t.Errorf("nextGeometry copy %d calls measure", i+1)
		}
	}

	areas := declaredNodes(t, r, "shape.go", goextract.KindMethod, "Area")
	if len(areas) != 2 {
		t.Fatalf("got %d Shape.Area nodes, want 2", len(areas))
	}
	wantIDs := []string{
		nodeid.DeclID(goextract.KindMethod, "Shape.Area", "shape.go", "(int)"),
		nodeid.DeclID(goextract.KindMethod, "Shape.Area", "shape.go", "(int,int)"),
	}
	for i, n := range areas {
		if n.Id != wantIDs[i] || n.Arity != int32(i+1) {
			t.Errorf("Shape.Area copy %d: id %s arity %d, want id %s arity %d", i+1, n.Id, n.Arity, wantIDs[i], i+1)
		}
	}
}

// TestBackfillReplacesOldGraph writes a store that looks like one from
// before unique declaration ids: it holds a node and an edge that no
// source file declares, lacks the has_unique_decl_ids flag and has a
// wrong node count. The first sync rebuilds it into the same graph as a
// fresh index. The next two syncs are incremental again.
func TestBackfillReplacesOldGraph(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"a.go": "package fixture\n\nfunc A() { B() }\n",
		"b.go": "package fixture\n\nfunc B() {}\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	meta, err := r.GetMeta()
	r.Close()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	ghost := &schema.Node{
		Id: nodeid.DeclID(goextract.KindFunction, "Ghost", "gone.go", "()"), Kind: goextract.KindFunction,
		Name: "Ghost", QualifiedName: "Ghost", FilePath: "gone.go", Language: "go", StartLine: 1, EndLine: 1,
	}
	bID := nodeid.DeclID(goextract.KindFunction, "B", "b.go", "()")
	if err := w.PutNode(ghost); err != nil {
		t.Fatalf("PutNode: %v", err)
	}
	if err := w.PutEdge(&schema.Edge{Source: ghost.Id, Target: bID, Kind: goextract.RefKindCalls}, "gone.go"); err != nil {
		t.Fatalf("PutEdge: %v", err)
	}
	meta.HasUniqueDeclIds = false
	meta.NodeCount = 999
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	store.Close()

	stats, err := Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync 1: %v", err)
	}
	if stats.FilesReparsed != stats.Files || stats.Files != 2 {
		t.Fatalf("Sync 1 reparsed %d of %d files, want a full rebuild of 2", stats.FilesReparsed, stats.Files)
	}
	requireInvariants(t, storeDir)
	freshDir := t.TempDir()
	if _, err := Run(root, freshDir, Options{}); err != nil {
		t.Fatalf("Run (fresh): %v", err)
	}
	synced, closeSynced := openSnapshot(t, storeDir)
	fresh, closeFresh := openSnapshot(t, freshDir)
	err = invariant.Equal(synced, fresh)
	closeSynced()
	closeFresh()
	if err != nil {
		t.Fatalf("after the rebuild, the store differs from a fresh index (first store: synced):\n%v", err)
	}

	for i, src := range []string{
		"package fixture\n\nfunc A() { B() }\n\nfunc C() {}\n",
		"package fixture\n\nfunc A() { B(); C() }\n\nfunc C() {}\n",
	} {
		writeFile(t, filepath.Join(root, "a.go"), src)
		stats, err := Sync(root, storeDir, Options{})
		if err != nil {
			t.Fatalf("Sync %d: %v", i+2, err)
		}
		if stats.FilesReparsed != 1 {
			t.Errorf("Sync %d reparsed %d files, want only a.go", i+2, stats.FilesReparsed)
		}
		requireInvariants(t, storeDir)
	}
}

// TestSyncReresolvesWhenANewSymbolWins adds, in a sync, a second
// declaration of a name in the same package whose id wins the resolver's
// tie-break. The file whose call resolved to the first declaration is not
// changed, but the sync must move its call to the new winner, as a fresh
// index does.
func TestSyncReresolvesWhenANewSymbolWins(t *testing.T) {
	name := ""
	for i := 0; i < 1000 && name == ""; i++ {
		n := fmt.Sprintf("Beta%d", i)
		if nodeid.DeclID(goextract.KindFunction, n, "b.go", "()") < nodeid.DeclID(goextract.KindFunction, n, "a.go", "()") {
			name = n
		}
	}
	root := writeFixture(t, map[string]string{
		"a.go":      fmt.Sprintf("package fixture\n\nfunc %s() {}\n", name),
		"caller.go": fmt.Sprintf("package fixture\n\nfunc Caller() { %s() }\n", name),
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	writeFile(t, filepath.Join(root, "b.go"), fmt.Sprintf("package fixture\n\nfunc %s() {}\n", name))
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	r, closeFn := openSnapshot(t, storeDir)
	callers := declaredNodes(t, r, "caller.go", goextract.KindFunction, "Caller")
	winner := nodeid.DeclID(goextract.KindFunction, name, "b.go", "()")
	if len(callers) != 1 || !hasEdge(t, r, callers[0].Id, goextract.RefKindCalls, winner) {
		t.Errorf("after the sync, Caller does not call %s in b.go", name)
	}
	closeFn()
	requireSyncEqualsRun(t, root, storeDir)
}

// TestSyncReresolvesWhenAnOverloadIsAdded is the spec's sync scenario
// for design.md D11, with Kotlin top-level functions so that the new
// overload arrives in a file the caller has no edge into. a.kt declares
// add(Any) and c.kt calls add(s) with a String. A sync that adds b.kt
// with add(String) reparses c.kt, whose call then targets add(String),
// as a fresh index resolves it.
func TestSyncReresolvesWhenAnOverloadIsAdded(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"a.kt": "package fixture\n\nfun add(o: Any) {}\n",
		"c.kt": "package fixture\n\nfun run(s: String) {\n    add(s)\n}\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	writeFile(t, filepath.Join(root, "b.kt"), "package fixture\n\nfun add(s: String) {}\n")
	if _, err := Sync(root, storeDir, Options{}); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	g := readGraph(t, storeDir)
	g.requireOnlyCallNamed(t, g.node(t, "c.kt", goextract.KindFunction, "run", ""), g.node(t, "b.kt", goextract.KindFunction, "add", ""))
	requireSyncEqualsRun(t, root, storeDir)
}
