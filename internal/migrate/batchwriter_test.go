package migrate

import (
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// countingStore wraps a real GraphStore and tracks Writer lifecycle: every
// NewWriter increments opened; every Writer terminal (Commit or Close)
// increments done exactly once. A leaked (abandoned, never-terminated) Writer
// shows up as opened-done > 0.
type countingStore struct {
	graphstore.GraphStore
	opened int
	done   int
}

func (c *countingStore) NewWriter() (graphstore.Writer, error) {
	w, err := c.GraphStore.NewWriter()
	if err != nil {
		return nil, err
	}
	c.opened++
	return &countingWriter{Writer: w, store: c}, nil
}

func (c *countingStore) outstanding() int { return c.opened - c.done }

type countingWriter struct {
	graphstore.Writer
	store    *countingStore
	finished bool
}

func (w *countingWriter) markDone() {
	if !w.finished {
		w.finished = true
		w.store.done++
	}
}

func (w *countingWriter) Commit() error {
	err := w.Writer.Commit()
	w.markDone()
	return err
}

func (w *countingWriter) Close() error {
	err := w.Writer.Close()
	w.markDone()
	return err
}

// TestBatchWriter_ClosesTrailingWriter proves WR-02: commitData eagerly opens
// a fresh Writer after committing, so the batchWriter always holds an open,
// never-committed Pebble batch once its data is flushed. Without batchWriter.
// Close(), that trailing batch is abandoned un-Closed on every run — a
// lifecycle leak and a violation of the graphstore.Writer contract (an
// abandoned Writer MUST be Closed to return its batch to Pebble's pool). This
// test drives the exact commit rhythm Run uses, confirms the trailing Writer
// is left open, and confirms Close() returns it.
func TestBatchWriter_ClosesTrailingWriter(t *testing.T) {
	real, err := graphstore.Open(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer real.Close()

	cs := &countingStore{GraphStore: real}

	bw, err := newBatchWriter(cs)
	if err != nil {
		t.Fatalf("newBatchWriter: %v", err)
	}

	// Mirror Run's rhythm: stage a Put, then commitData (which commits the
	// staged batch AND eagerly opens a fresh, empty Writer for the next Put).
	if err := bw.w.PutFile(&schema.File{Path: "pkg/a.go"}); err != nil {
		t.Fatalf("PutFile: %v", err)
	}
	bw.n++
	if err := bw.commitData(); err != nil {
		t.Fatalf("commitData: %v", err)
	}

	// The eagerly-opened fresh Writer is open and uncommitted — the leak.
	if got := cs.outstanding(); got != 1 {
		t.Fatalf("after commitData: outstanding writers = %d, want 1 (the eagerly-opened trailing batch)", got)
	}

	if err := bw.Close(); err != nil {
		t.Fatalf("batchWriter.Close: %v", err)
	}

	if got := cs.outstanding(); got != 0 {
		t.Errorf("after batchWriter.Close: outstanding writers = %d, want 0 (trailing batch must be returned)", got)
	}

	// Close is idempotent — a second call (e.g. an explicit call plus Run's
	// deferred one) must not double-count or error.
	if err := bw.Close(); err != nil {
		t.Fatalf("batchWriter.Close (second call): %v", err)
	}
	if got := cs.outstanding(); got != 0 {
		t.Errorf("after a second batchWriter.Close: outstanding writers = %d, want 0", got)
	}
}
