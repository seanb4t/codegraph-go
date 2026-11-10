package indexer

import (
	"errors"
	"io"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// fixtureResults returns Pass 1's results and module path for the shared
// multi-package testdata/gofixture tree, the same fixture 02-03's own
// tests exercise.
func fixtureResults(t *testing.T) ([]goextract.FileResult, string) {
	t.Helper()
	files, modulePath, err := discover.Discover(fixtureRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	results, err := Extract(files, 0)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	return results, modulePath
}

// errStubWrite is the sentinel error stubWriter returns for whichever
// Put* method its failOn field names.
var errStubWrite = errors.New("stub write error")

// stubWriter is an in-memory graphstore.Writer stand-in so writeGraph's
// single-writer/batched-commit contract can be tested without a real
// Pebble-backed store. failOn, when non-empty, names the Put* method that
// should fail (simulating a mid-write staging error).
type stubWriter struct {
	nodes    []*schema.Node
	edges    []*schema.Edge
	files    []*schema.File
	excluded []*schema.ExcludedFile
	meta     *schema.Meta

	commitCalls            int
	closeCalls             int
	deleteAllExcludedCalls int

	// order records the call sequence of DeleteAllExcludedFiles and
	// PutExcludedFile so a test can assert the range-delete happens
	// BEFORE the rewrite (Phase 10 D-07, TestWriteGraphRangeDeletesExcludedNamespaceBeforeRewriting).
	order []string

	failOn string
}

func (w *stubWriter) PutNode(n *schema.Node) error {
	if w.failOn == "PutNode" {
		return errStubWrite
	}
	w.nodes = append(w.nodes, n)
	return nil
}

func (w *stubWriter) PutEdge(e *schema.Edge, ownerPath string) error {
	if w.failOn == "PutEdge" {
		return errStubWrite
	}
	w.edges = append(w.edges, e)
	return nil
}

func (w *stubWriter) PutFile(f *schema.File) error {
	if w.failOn == "PutFile" {
		return errStubWrite
	}
	w.files = append(w.files, f)
	return nil
}

func (w *stubWriter) PutMeta(m *schema.Meta) error {
	if w.failOn == "PutMeta" {
		return errStubWrite
	}
	w.meta = m
	return nil
}

func (w *stubWriter) PutExcludedFile(x *schema.ExcludedFile) error {
	if w.failOn == "PutExcludedFile" {
		return errStubWrite
	}
	w.excluded = append(w.excluded, x)
	w.order = append(w.order, "PutExcludedFile")
	return nil
}

func (w *stubWriter) DeleteExcludedFile(path string) error {
	if w.failOn == "DeleteExcludedFile" {
		return errStubWrite
	}
	return nil
}

func (w *stubWriter) DeleteGraph() error {
	if w.failOn == "DeleteGraph" {
		return errStubWrite
	}
	w.order = append(w.order, "DeleteGraph")
	return nil
}

func (w *stubWriter) DeleteAllExcludedFiles() error {
	if w.failOn == "DeleteAllExcludedFiles" {
		return errStubWrite
	}
	w.deleteAllExcludedCalls++
	w.order = append(w.order, "DeleteAllExcludedFiles")
	return nil
}

func (w *stubWriter) DeleteFileSubgraph(path string) error { return nil }

func (w *stubWriter) DeleteNode(id string) error { return nil }

func (w *stubWriter) DeleteEdge(source, kind, target string) error { return nil }

func (w *stubWriter) DeleteFileIndexEdge(ownerPath, source, kind, target string) error { return nil }

func (w *stubWriter) Commit() error {
	w.commitCalls++
	return nil
}

func (w *stubWriter) Close() error {
	w.closeCalls++
	return nil
}

// stubStore is a graphstore.GraphStore stand-in whose NewWriter always
// returns the same injected stubWriter, so a test can inspect it after
// writeGraph returns.
//
// snapshotMeta/snapshotErr configure what Snapshot().GetMeta() reports to
// writeGraph's WR-01 (iteration 2) read-modify-write of CoverageGeneration.
// The zero value (both nil) mimics a genuinely fresh store — GetMeta
// returns graphstore.ErrNotFound, exactly as pebbleReader.GetMeta does for
// a store with no Meta record yet — so every pre-existing test in this
// file, none of which set these fields, exercises the "fresh store starts
// at generation 1" path unchanged.
type stubStore struct {
	writer *stubWriter

	snapshotMeta *schema.Meta
	snapshotErr  error
}

func (s *stubStore) Snapshot() (graphstore.Reader, error) {
	if s.snapshotMeta != nil {
		return &stubReader{meta: s.snapshotMeta}, nil
	}
	if s.snapshotErr != nil {
		return &stubReader{err: s.snapshotErr}, nil
	}
	return &stubReader{err: graphstore.ErrNotFound}, nil
}
func (s *stubStore) NewWriter() (graphstore.Writer, error) { return s.writer, nil }
func (s *stubStore) NewBoundedWriter(int, func(graphstore.Commit) error) (graphstore.Writer, error) {
	return s.writer, nil
}
func (s *stubStore) Export(w io.Writer) error { return nil }
func (s *stubStore) Close() error             { return nil }

// stubReader is a graphstore.Reader stand-in used ONLY for writeGraph's
// prior-Meta read (Snapshot().GetMeta()). Every other method panics if
// called — writeGraph never calls them, and a future caller that starts
// doing so should fail loudly rather than silently return a zero value.
type stubReader struct {
	meta *schema.Meta
	err  error
}

func (r *stubReader) GetMeta() (*schema.Meta, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.meta, nil
}
func (r *stubReader) GetNode(id string) (*schema.Node, error) {
	panic("stubReader.GetNode not implemented")
}
func (r *stubReader) GetFile(path string) (*schema.File, error) {
	panic("stubReader.GetFile not implemented")
}
func (r *stubReader) IterateEdges(srcPrefix string) (graphstore.EdgeIterator, error) {
	panic("stubReader.IterateEdges not implemented")
}
func (r *stubReader) IterateNodes() (graphstore.NodeIterator, error) {
	panic("stubReader.IterateNodes not implemented")
}
func (r *stubReader) IterateFiles() (graphstore.FileIterator, error) {
	panic("stubReader.IterateFiles not implemented")
}
func (r *stubReader) IterateFileIndex(path string) (graphstore.FileIndexIterator, error) {
	panic("stubReader.IterateFileIndex not implemented")
}
func (r *stubReader) IterateExcludedFiles() (graphstore.ExcludedFileIterator, error) {
	panic("stubReader.IterateExcludedFiles not implemented")
}
func (r *stubReader) Close() error { return nil }

// TestSingleWriter_CommitsOnce proves the entire resolved graph is written
// through exactly ONE GraphStore.Writer with a single Commit() — no
// per-symbol commits (D-04a) — and that Meta is stamped with correct
// node/edge counts and schema_version == 1.
func TestSingleWriter_CommitsOnce(t *testing.T) {
	w := &stubWriter{}
	store := &stubStore{writer: w}

	nodes := []*schema.Node{
		{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"},
		{Id: "fn:b", Kind: "function", Name: "b", FilePath: "pkg/a.go"},
	}
	packageNodes := []*schema.Node{
		{Id: "package:x", Kind: "package", Name: "x", QualifiedName: "example.com/x"},
	}
	edges := []*schema.Edge{
		{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 30, Col: 1, Provenance: "ast"},
		{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 12, Col: 1, Provenance: "ast"},
	}
	files := []*schema.File{
		{Path: "pkg/a.go", ContentHash: "deadbeef", Language: "go", NodeCount: 2, EdgeCount: 1},
	}

	if err := writeGraph(store, nodes, packageNodes, edges, files, "", nil, 0); err != nil {
		t.Fatalf("writeGraph returned error: %v", err)
	}

	if w.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", w.commitCalls)
	}
	if w.closeCalls != 0 {
		t.Errorf("closeCalls = %d, want 0 (Close is only for the error path)", w.closeCalls)
	}
	if len(w.nodes) != len(nodes)+len(packageNodes) {
		t.Errorf("staged %d nodes, want %d (symbol + package pseudo-nodes)", len(w.nodes), len(nodes)+len(packageNodes))
	}
	if len(w.files) != len(files) {
		t.Errorf("staged %d files, want %d", len(w.files), len(files))
	}
	if len(w.edges) != 1 {
		t.Fatalf("staged %d edges, want 1 (the duplicate (fn:a,calls,fn:b) pair must collapse)", len(w.edges))
	}
	if w.edges[0].Line != 12 {
		t.Errorf("collapsed edge Line = %d, want 12 (the (filePath,line,col)-sorted first candidate)", w.edges[0].Line)
	}
	if w.meta == nil {
		t.Fatal("PutMeta was never called")
	}
	if w.meta.SchemaVersion != schema.SchemaVersion {
		t.Errorf("Meta.SchemaVersion = %d, want %d", w.meta.SchemaVersion, schema.SchemaVersion)
	}
	if w.meta.NodeCount != int64(len(nodes)+len(packageNodes)) {
		t.Errorf("Meta.NodeCount = %d, want %d", w.meta.NodeCount, len(nodes)+len(packageNodes))
	}
	if w.meta.EdgeCount != 1 {
		t.Errorf("Meta.EdgeCount = %d, want 1", w.meta.EdgeCount)
	}
}

// TestWriteGraphStagesExcludedFilesInTheSameBatch proves writeGraph stages
// ExcludedFile records on the SAME Writer/Commit as every other record
// kind (Phase 10 D-07/T-10-10) — never a second commit — and stamps
// Meta.HasCoverage true.
func TestWriteGraphStagesExcludedFilesInTheSameBatch(t *testing.T) {
	w := &stubWriter{}
	store := &stubStore{writer: w}

	nodes := []*schema.Node{{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"}}
	files := []*schema.File{{Path: "pkg/a.go", ContentHash: "deadbeef", Language: "go"}}
	excluded := []*schema.ExcludedFile{
		{Path: "b.md", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".md"},
		{Path: "a/tagged.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, Detail: "linux/amd64"},
	}

	if err := writeGraph(store, nodes, nil, nil, files, "", excluded, 0); err != nil {
		t.Fatalf("writeGraph returned error: %v", err)
	}

	if w.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", w.commitCalls)
	}
	if w.closeCalls != 0 {
		t.Errorf("closeCalls = %d, want 0", w.closeCalls)
	}
	if len(w.excluded) != 2 {
		t.Fatalf("staged %d excluded records, want 2", len(w.excluded))
	}
	if w.excluded[0].GetPath() != "a/tagged.go" || w.excluded[1].GetPath() != "b.md" {
		t.Fatalf("excluded records not path-sorted: got [%q, %q], want [\"a/tagged.go\", \"b.md\"]", w.excluded[0].GetPath(), w.excluded[1].GetPath())
	}
	if w.meta == nil {
		t.Fatal("PutMeta was never called")
	}
	if !w.meta.GetHasCoverage() {
		t.Error("Meta.HasCoverage = false, want true")
	}
}

// TestWriteGraphStampsMonotonicCoverageGeneration proves the WR-01
// (iteration 2) fix: writeGraph read-modify-writes CoverageGeneration from
// whatever the store's PRIOR Meta recorded, incrementing by exactly 1 —
// never resetting to 1 unconditionally — so a from-scratch rewrite over an
// EXISTING store (Sync's D-02b backfill path delegates to writeGraph too)
// continues the same monotonic sequence a live CoverageRows page token may
// already be pinned to.
func TestWriteGraphStampsMonotonicCoverageGeneration(t *testing.T) {
	t.Run("fresh store (no prior Meta) starts at generation 1", func(t *testing.T) {
		w := &stubWriter{}
		store := &stubStore{writer: w} // zero value: Snapshot().GetMeta() -> ErrNotFound

		if err := writeGraph(store, nil, nil, nil, nil, "", nil, 0); err != nil {
			t.Fatalf("writeGraph returned error: %v", err)
		}
		if w.meta == nil {
			t.Fatal("PutMeta was never called")
		}
		if got := w.meta.GetCoverageGeneration(); got != 1 {
			t.Errorf("CoverageGeneration = %d, want 1 for a fresh store", got)
		}
	})

	t.Run("existing store continues from its prior generation", func(t *testing.T) {
		w := &stubWriter{}
		store := &stubStore{writer: w, snapshotMeta: &schema.Meta{CoverageGeneration: 41}}

		if err := writeGraph(store, nil, nil, nil, nil, "", nil, 0); err != nil {
			t.Fatalf("writeGraph returned error: %v", err)
		}
		if w.meta == nil {
			t.Fatal("PutMeta was never called")
		}
		if got := w.meta.GetCoverageGeneration(); got != 42 {
			t.Errorf("CoverageGeneration = %d, want 42 (41 + 1)", got)
		}
	})

	t.Run("a non-ErrNotFound Snapshot/GetMeta failure propagates", func(t *testing.T) {
		w := &stubWriter{}
		store := &stubStore{writer: w, snapshotErr: errStubWrite}

		if err := writeGraph(store, nil, nil, nil, nil, "", nil, 0); err == nil {
			t.Fatal("writeGraph returned nil error, want the propagated snapshot/GetMeta failure")
		}
		if w.commitCalls != 0 {
			t.Errorf("commitCalls = %d, want 0 — a prior-Meta read failure must abort before any Writer is opened", w.commitCalls)
		}
	})
}

// TestWriteGraphRangeDeletesExcludedNamespaceBeforeRewriting proves
// writeGraph clears the whole c/ namespace via DeleteAllExcludedFiles
// BEFORE staging any PutExcludedFile — the D-07 "range-delete then
// rewrite" sequence a from-scratch rewrite over an EXISTING store (e.g.
// Sync's D-02b backfill path) needs to avoid layering stale records on
// top of a fresh set — all in the SAME single-commit batch.
func TestWriteGraphRangeDeletesExcludedNamespaceBeforeRewriting(t *testing.T) {
	w := &stubWriter{}
	store := &stubStore{writer: w}

	excluded := []*schema.ExcludedFile{
		{Path: "a.md", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION},
		{Path: "b.md", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION},
	}

	if err := writeGraph(store, nil, nil, nil, nil, "", excluded, 0); err != nil {
		t.Fatalf("writeGraph returned error: %v", err)
	}

	if w.deleteAllExcludedCalls != 1 {
		t.Fatalf("deleteAllExcludedCalls = %d, want 1", w.deleteAllExcludedCalls)
	}
	if w.commitCalls != 1 {
		t.Fatalf("commitCalls = %d, want 1", w.commitCalls)
	}

	firstPutIdx := -1
	deleteAllIdx := -1
	for i, op := range w.order {
		if op == "DeleteAllExcludedFiles" && deleteAllIdx == -1 {
			deleteAllIdx = i
		}
		if op == "PutExcludedFile" && firstPutIdx == -1 {
			firstPutIdx = i
		}
	}
	if deleteAllIdx == -1 {
		t.Fatal("DeleteAllExcludedFiles was never recorded in call order")
	}
	if firstPutIdx == -1 {
		t.Fatal("PutExcludedFile was never recorded in call order")
	}
	if deleteAllIdx >= firstPutIdx {
		t.Fatalf("DeleteAllExcludedFiles (index %d) did not precede the first PutExcludedFile (index %d): order = %v", deleteAllIdx, firstPutIdx, w.order)
	}
}

// TestSingleWriter_CloseOnStagingError proves a staging error mid-write
// calls Writer.Close() (releasing the batch) instead of Commit(), and
// writeGraph returns the error — never a partial commit.
func TestSingleWriter_CloseOnStagingError(t *testing.T) {
	w := &stubWriter{failOn: "PutEdge"}
	store := &stubStore{writer: w}

	nodes := []*schema.Node{{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"}}
	edges := []*schema.Edge{{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 1, Provenance: "ast"}}
	files := []*schema.File{{Path: "pkg/a.go"}}

	err := writeGraph(store, nodes, nil, edges, files, "", nil, 0)
	if err == nil {
		t.Fatal("expected writeGraph to return the staging error, got nil")
	}
	if w.commitCalls != 0 {
		t.Errorf("commitCalls = %d, want 0 (must not commit after a staging error)", w.commitCalls)
	}
	if w.closeCalls != 1 {
		t.Errorf("closeCalls = %d, want 1 (must release the abandoned batch)", w.closeCalls)
	}
}

// TestResolve_EndToEnd proves the exported Resolve entry point runs the
// full Pass 2 pipeline against a REAL GraphStore (not a stub), over the
// shared multi-package fixture.
func TestResolve_EndToEnd(t *testing.T) {
	results, modulePath := fixtureResults(t)

	store, err := graphstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	if _, err := Resolve(store, results, modulePath, "", nil, 0); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	meta, err := (func() (*schema.Meta, error) {
		r, err := store.Snapshot()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return r.GetMeta()
	})()
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if meta.SchemaVersion != schema.SchemaVersion {
		t.Errorf("Meta.SchemaVersion = %d, want %d", meta.SchemaVersion, schema.SchemaVersion)
	}
	if meta.NodeCount == 0 {
		t.Error("Meta.NodeCount = 0, want > 0")
	}
	if meta.EdgeCount == 0 {
		t.Error("Meta.EdgeCount = 0, want > 0")
	}
}

// TestOverridesSynthesizedAcrossOverloadSets indexes a subclass that
// overrides two overloads of one name. The type contains an overload set
// that contains the methods, and the overrides synthesis still pairs
// each overload with the one it overrides.
func TestOverridesSynthesizedAcrossOverloadSets(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"Base.java": `package p;

public class Base {
    public void foo(int n) {}

    public void foo(String s) {}
}
`,
		"Derived.java": `package p;

public class Derived extends Base {
    @Override
    public void foo(int n) {}

    @Override
    public void foo(String s) {}
}
`,
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	g := readGraph(t, storeDir)
	for _, sig := range []string{"(int n)", "(String s)"} {
		from := g.node(t, "Derived.java", goextract.KindMethod, "foo", sig)
		to := g.node(t, "Base.java", goextract.KindMethod, "foo", sig)
		found := false
		for _, e := range g.edges {
			if e.Kind == goextract.EdgeKindOverrides && e.Source == from.Id {
				if e.Target != to.Id {
					t.Errorf("Derived.foo%s overrides %s, want Base.foo%s", sig, g.describe(e.Target), sig)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("no overrides edge from Derived.foo%s", sig)
		}
	}
}
