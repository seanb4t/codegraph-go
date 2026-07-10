package graphstore

import (
	"errors"

	"github.com/cockroachdb/pebble/v2"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// pebbleWriter is the Writer implementation. Every Put/Delete stages a
// mutation on a single Pebble Batch; nothing reaches the engine until
// Commit applies the whole batch atomically (D-04) — this is what makes
// TestConcurrentReadersSingleWriter's "no torn write" guarantee hold.
type pebbleWriter struct {
	batch *pebble.Batch
}

func (w *pebbleWriter) PutNode(n *schema.Node) error {
	data, err := proto.Marshal(n)
	if err != nil {
		return err
	}
	return w.batch.Set(nodeKey(n.GetId()), data, nil)
}

func (w *pebbleWriter) PutEdge(e *schema.Edge) error {
	data, err := proto.Marshal(e)
	if err != nil {
		return err
	}
	return w.batch.Set(edgeKey(e.GetSource(), e.GetKind(), e.GetTarget()), data, nil)
}

func (w *pebbleWriter) PutFile(f *schema.File) error {
	data, err := proto.Marshal(f)
	if err != nil {
		return err
	}
	return w.batch.Set(fileKey(f.GetPath()), data, nil)
}

func (w *pebbleWriter) PutMeta(m *schema.Meta) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	return w.batch.Set(metaKey(metaRecordName), data, nil)
}

// DeleteFileSubgraph stages a single Pebble range-delete over path's own
// file record (D-03): [fileSubgraphPrefix(path), rangeUpperBound(...)) is
// the exact byte range that isolates path's record from a
// lexicographically adjacent sibling (e.g. "foo" vs "foobar").
func (w *pebbleWriter) DeleteFileSubgraph(path string) error {
	start := fileSubgraphPrefix(path)
	end := rangeUpperBound(start)
	return w.batch.DeleteRange(start, end, nil)
}

// Commit applies the batch atomically and durably (pebble.Sync), then
// closes the batch to return it to Pebble's internal batch pool
// (pebble/v2's own idiom: NewBatch/NewIndexedBatch draw from a sync.Pool
// that Batch.Close returns the batch to).
func (w *pebbleWriter) Commit() error {
	err := w.batch.Commit(pebble.Sync)
	if closeErr := w.batch.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Close releases the batch's resources without applying it. Callers that
// abandon a Writer without calling Commit (e.g. after a marshal error
// while staging Puts) must call Close to return the batch to Pebble's
// pool; calling Close after a successful Commit is a safe no-op (Pebble's
// own pebble.ErrClosed from the redundant underlying Batch.Close is
// swallowed here rather than surfaced as a caller-facing error).
func (w *pebbleWriter) Close() error {
	if err := w.batch.Close(); err != nil && !errors.Is(err, pebble.ErrClosed) {
		return err
	}
	return nil
}
