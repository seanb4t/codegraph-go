package indexer

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

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

// readMeta returns the metadata record of the store at storeDir.
func readMeta(t *testing.T, storeDir string) *schema.Meta {
	t.Helper()
	meta, err := readStoreMeta(storeDir)
	if err != nil {
		t.Fatalf("readStoreMeta(%s): %v", storeDir, err)
	}
	return meta
}

// TestRunStampsMetadata checks the metadata record of a full index: the
// schema version, and node and edge counts that equal the records the
// store holds.
func TestRunStampsMetadata(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), "store")
	stats, err := Run(fixtureRoot, storeDir, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	meta := readMeta(t, storeDir)
	if meta.SchemaVersion != schema.SchemaVersion {
		t.Errorf("Meta.SchemaVersion = %d, want %d", meta.SchemaVersion, schema.SchemaVersion)
	}
	if !meta.GetHasCoverage() {
		t.Error("Meta.HasCoverage = false, want true")
	}
	// requireInvariants compares the counts of the metadata with the
	// records of the store, and fails when either is zero.
	counts := requireInvariants(t, storeDir)
	if int(meta.NodeCount) != counts.Nodes || int(meta.EdgeCount) != counts.Edges {
		t.Errorf("Meta counts %d nodes and %d edges, the store holds %d and %d", meta.NodeCount, meta.EdgeCount, counts.Nodes, counts.Edges)
	}
	if stats.Nodes != counts.Nodes || stats.Edges != counts.Edges {
		t.Errorf("Stats counts %d nodes and %d edges, the store holds %d and %d", stats.Nodes, stats.Edges, counts.Nodes, counts.Edges)
	}
}

// TestRunStampsMonotonicCoverageGeneration checks that a full index reads
// the coverage generation of the store it replaces and stamps one more, so
// a page token of an earlier generation can tell that the data changed
// (WR-01).
func TestRunStampsMonotonicCoverageGeneration(t *testing.T) {
	t.Run("fresh store (no prior Meta) starts at generation 1", func(t *testing.T) {
		storeDir := filepath.Join(t.TempDir(), "store")
		stats, err := Run(fixtureRoot, storeDir, Options{})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got := readMeta(t, storeDir).GetCoverageGeneration(); got != 1 {
			t.Errorf("CoverageGeneration = %d, want 1 for a fresh store", got)
		}
		if stats.PriorGenerationErr != nil {
			t.Errorf("Stats.PriorGenerationErr = %v, want nil for a store that was never indexed", stats.PriorGenerationErr)
		}
	})

	t.Run("existing store continues from its prior generation", func(t *testing.T) {
		storeDir := filepath.Join(t.TempDir(), "store")
		store, err := graphstore.Open(storeDir)
		if err != nil {
			t.Fatalf("graphstore.Open: %v", err)
		}
		w, err := store.NewWriter()
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		if err := w.PutMeta(&schema.Meta{CoverageGeneration: 41}); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}
		if err := w.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		if _, err := Run(fixtureRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		if got := readMeta(t, storeDir).GetCoverageGeneration(); got != 42 {
			t.Errorf("CoverageGeneration = %d, want 42 (41 + 1)", got)
		}
	})

	t.Run("an unreadable prior store is replaced, from generation 1", func(t *testing.T) {
		storeDir := filepath.Join(t.TempDir(), "store")
		if _, err := Run(fixtureRoot, storeDir, Options{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		// Overwrite the manifest of the store, so that it opens with an
		// error that is neither "locked" nor "not found".
		manifests, err := filepath.Glob(filepath.Join(storeDir, "MANIFEST-*"))
		if err != nil || len(manifests) != 1 {
			t.Fatalf("found the MANIFEST files %v, %v; want exactly one", manifests, err)
		}
		if err := os.WriteFile(manifests[0], []byte("not a manifest\n"), 0o644); err != nil {
			t.Fatalf("overwrite %s: %v", manifests[0], err)
		}
		if _, err := readStoreMeta(storeDir); err == nil || errors.Is(err, graphstore.ErrNotFound) || errors.Is(err, graphstore.ErrStoreLocked) {
			t.Fatalf("readStoreMeta of the damaged store = %v; want an error that is neither not-found nor locked, so the test reaches the third outcome", err)
		}

		stats, err := Run(fixtureRoot, storeDir, Options{})
		if err != nil {
			t.Fatalf("Run over an unreadable store: %v, want the run to replace it", err)
		}
		if stats.PriorGenerationErr == nil {
			t.Error("Stats.PriorGenerationErr = nil, want the error of the read")
		}
		if got := readMeta(t, storeDir).GetCoverageGeneration(); got != 1 {
			t.Errorf("CoverageGeneration = %d, want 1", got)
		}
		requireInvariants(t, storeDir)
	})
}

// TestWriterClosesBatchOnStageError checks the writer goroutine of a full
// index against a stub store: an error while it stages a unit makes it
// release its batch with Close, commit nothing, and return the error.
func TestWriterClosesBatchOnStageError(t *testing.T) {
	w := &stubWriter{failOn: "PutEdge"}
	store := &stubStore{writer: w}
	view := symbols.NewBuilder().Freeze()
	global := resolve.NewGlobal(view, symbols.NewHierarchy(view, nil), nil)

	released := 0
	units := make(chan stagedUnit, 1)
	units <- stagedUnit{
		unit: resolve.Unit{
			Nodes: []*schema.Node{{Id: "fn:a", Kind: "function", Name: "a", FilePath: "pkg/a.go"}},
			Edges: []*schema.Edge{{Source: "fn:a", Kind: "calls", Target: "fn:b", Line: 1, Provenance: "ast"}},
			File:  &schema.File{Path: "pkg/a.go"},
		},
		release: func() { released++ },
	}
	close(units)

	tailCalled := false
	_, err := writeUnits(context.Background(), store, units, global, func(graphstore.Writer, *stagedCounts) error {
		tailCalled = true
		return nil
	})
	if !errors.Is(err, errStubWrite) {
		t.Fatalf("writeUnits error = %v, want the staging error", err)
	}
	// Positive assertion: the writer staged a record before the failure.
	if len(w.nodes) != 1 {
		t.Fatalf("the stub recorded %d node puts before the failure, want 1", len(w.nodes))
	}
	if w.commitCalls != 0 {
		t.Errorf("commitCalls = %d, want 0 (no commit after a staging error)", w.commitCalls)
	}
	if w.closeCalls != 1 {
		t.Errorf("closeCalls = %d, want 1 (the batch must be released)", w.closeCalls)
	}
	if tailCalled || released != 0 {
		t.Errorf("tail called = %v, releases = %d; want the writer to stop at the failed unit", tailCalled, released)
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
