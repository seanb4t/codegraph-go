package graphstore

import (
	"errors"

	"github.com/cockroachdb/pebble/v2"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// DefaultBatchCap is the batch cap of a full index and of Import: the
// length above which a bounded Writer commits its open batch before it
// adds the next record.
const DefaultBatchCap = 16 << 20

// Commit describes one batch that a bounded Writer committed.
type Commit struct {
	// Len is the length of the batch in bytes, the batch header included.
	Len int
	// BeforeLast is the length that the writer read before it added the
	// last record of the batch. A batch with more than one record has
	// BeforeLast at or below the cap, so the batch exceeds the cap by at
	// most its last record. A batch with one record has the length of an
	// empty batch here.
	BeforeLast int
	// Records is the number of Writer calls that the batch holds. One
	// call can put more than one Pebble entry.
	Records int
	// Sync reports whether the commit used pebble.Sync.
	Sync bool
}

// boundedWriter is a Writer that commits its batch whenever the batch
// passes a cap, so no batch holds the whole graph.
type boundedWriter struct {
	store    *pebbleStore
	inner    pebbleWriter
	cap      int
	onCommit func(Commit) error

	records    int
	beforeLast int
	// done is set by Commit and Close, and by a failed commit. The writer
	// then holds no batch.
	done bool
}

// NewBoundedWriter returns a Writer like NewWriter, with one difference:
// an add can commit a full batch first. Before it adds a record, the
// writer reads the length of its batch. If that length is above cap and
// the batch holds a record, it commits the batch and starts a new one.
// So the writer computes no record size, and a batch exceeds cap by at
// most its last record.
//
// Commit applies the open batch if it holds a record, and Close releases
// an open batch without a commit, as for every Writer. The commit that
// Commit makes is durable (pebble.Sync) unless the store came from
// OpenStaging, which has no write-ahead log. Every earlier commit uses
// pebble.NoSync. No commit is empty.
//
// onCommit, if not nil, runs after each commit with its description. If it
// returns an error, the add or the Commit that made the commit fails with
// that error. The writes of one bounded Writer are not atomic as a whole:
// a caller that needs all or nothing writes into a store that it places
// with ReplaceStore.
func (s *pebbleStore) NewBoundedWriter(cap int, onCommit func(Commit) error) (Writer, error) {
	batch, err := s.newBatch()
	if err != nil {
		return nil, err
	}
	return &boundedWriter{store: s, inner: pebbleWriter{batch: batch}, cap: cap, onCommit: onCommit}, nil
}

// newBatch returns a new batch, guarded against a concurrent Close like
// NewWriter (WR-01).
func (s *pebbleStore) newBatch() (*pebble.Batch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed.Load() {
		return nil, ErrClosed
	}
	return s.db.NewBatch(), nil
}

// add runs before each record. It commits a batch that is over the cap,
// then notes the length that the record is added at.
func (w *boundedWriter) add(put func() error) error {
	if w.done {
		return ErrClosed
	}
	// Batch.Len counts the batch header, so the length alone cannot tell
	// an empty batch. Without the Count check, a cap below the header
	// length commits an empty batch before the first record.
	if w.inner.batch.Len() > w.cap && w.inner.batch.Count() > 0 {
		if err := w.commit(false); err != nil {
			return err
		}
		batch, err := w.store.newBatch()
		if err != nil {
			w.done = true
			return err
		}
		w.inner.batch = batch
		w.records = 0
	}
	w.beforeLast = w.inner.batch.Len()
	w.records++
	return put()
}

// commit applies the open batch and releases it.
func (w *boundedWriter) commit(sync bool) error {
	c := Commit{Len: w.inner.batch.Len(), BeforeLast: w.beforeLast, Records: w.records, Sync: sync}
	err := w.inner.batch.Commit(syncOption(sync))
	if closeErr := w.inner.batch.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		w.done = true
		return err
	}
	if w.onCommit != nil {
		if err := w.onCommit(c); err != nil {
			w.done = true
			return err
		}
	}
	return nil
}

func (w *boundedWriter) Commit() error {
	if w.done {
		return ErrClosed
	}
	if w.inner.batch.Count() == 0 {
		w.done = true
		return w.inner.Close()
	}
	// A staging store has no write-ahead log, and Pebble refuses a Sync
	// commit on it. Its durability comes from the flush in its Close.
	err := w.commit(!w.store.staging)
	w.done = true
	return err
}

func (w *boundedWriter) Close() error {
	if w.done {
		return nil
	}
	w.done = true
	if err := w.inner.batch.Close(); err != nil && !errors.Is(err, pebble.ErrClosed) {
		return err
	}
	return nil
}

func (w *boundedWriter) PutNode(n *schema.Node) error {
	return w.add(func() error { return w.inner.PutNode(n) })
}

func (w *boundedWriter) PutEdge(e *schema.Edge, ownerPath string) error {
	return w.add(func() error { return w.inner.PutEdge(e, ownerPath) })
}

func (w *boundedWriter) PutFile(f *schema.File) error {
	return w.add(func() error { return w.inner.PutFile(f) })
}

func (w *boundedWriter) PutMeta(m *schema.Meta) error {
	return w.add(func() error { return w.inner.PutMeta(m) })
}

func (w *boundedWriter) PutExcludedFile(x *schema.ExcludedFile) error {
	return w.add(func() error { return w.inner.PutExcludedFile(x) })
}

func (w *boundedWriter) DeleteExcludedFile(path string) error {
	return w.add(func() error { return w.inner.DeleteExcludedFile(path) })
}

func (w *boundedWriter) DeleteAllExcludedFiles() error {
	return w.add(func() error { return w.inner.DeleteAllExcludedFiles() })
}

func (w *boundedWriter) DeleteNode(id string) error {
	return w.add(func() error { return w.inner.DeleteNode(id) })
}

func (w *boundedWriter) DeleteEdge(source, kind, target string) error {
	return w.add(func() error { return w.inner.DeleteEdge(source, kind, target) })
}

func (w *boundedWriter) DeleteFileIndexEdge(ownerPath, source, kind, target string) error {
	return w.add(func() error { return w.inner.DeleteFileIndexEdge(ownerPath, source, kind, target) })
}

func (w *boundedWriter) DeleteFileSubgraph(path string) error {
	return w.add(func() error { return w.inner.DeleteFileSubgraph(path) })
}

// syncOption returns the Pebble write option of a commit mode.
func syncOption(sync bool) *pebble.WriteOptions {
	if sync {
		return pebble.Sync
	}
	return pebble.NoSync
}
