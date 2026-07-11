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

// PutNode stages n's own n/ record, plus — when n has a FilePath (i.e. it
// is not the kindPackage pseudo-node, which has no owning file) — its
// x/<FilePath>/... file-index entry (Phase 4 D-02), so the owning file's
// nodes are enumerable via IterateFileIndex.
func (w *pebbleWriter) PutNode(n *schema.Node) error {
	data, err := proto.Marshal(n)
	if err != nil {
		return err
	}
	if err := w.batch.Set(nodeKey(n.GetId()), data, nil); err != nil {
		return err
	}
	if n.GetFilePath() == "" {
		return nil
	}
	return w.batch.Set(fileIndexNodeKey(n.GetFilePath(), n.GetId()), nil, nil)
}

// PutEdge stages e's own e/ record, plus — when ownerPath is non-empty —
// its x/<ownerPath>/... file-index entry (Phase 4 D-02), so the owning
// file's outgoing edges are enumerable via IterateFileIndex.
func (w *pebbleWriter) PutEdge(e *schema.Edge, ownerPath string) error {
	data, err := proto.Marshal(e)
	if err != nil {
		return err
	}
	if err := w.batch.Set(edgeKey(e.GetSource(), e.GetKind(), e.GetTarget()), data, nil); err != nil {
		return err
	}
	if ownerPath == "" {
		return nil
	}
	return w.batch.Set(fileIndexEdgeKey(ownerPath, e.GetSource(), e.GetKind(), e.GetTarget()), nil, nil)
}

// DeleteNode stages a point-delete of the node record identified by id
// (Phase 4 D-02). It does not touch that node's x/ file-index entry —
// callers driving a full-file prune use DeleteFileSubgraph instead, which
// range-deletes the whole x/<path>/... region in one call.
func (w *pebbleWriter) DeleteNode(id string) error {
	return w.batch.Delete(nodeKey(id), nil)
}

// DeleteEdge stages a point-delete of the edge record identified by
// (source, kind, target) (Phase 4 D-02). See DeleteNode's doc for why this
// does not also touch the x/ file-index entry.
func (w *pebbleWriter) DeleteEdge(source, kind, target string) error {
	return w.batch.Delete(edgeKey(source, kind, target), nil)
}

// DeleteFileIndexEdge stages a point-delete of ownerPath's own x/
// file-index entry for the outgoing edge (source, kind, target) (Phase 4
// D-02, CR-04). It does not touch the edge's own e/ record — callers pair
// this with DeleteEdge when discarding a single owned edge without a full
// DeleteFileSubgraph prune (Sync's pruneOwnedEdgesOnly).
func (w *pebbleWriter) DeleteFileIndexEdge(ownerPath, source, kind, target string) error {
	return w.batch.Delete(fileIndexEdgeKey(ownerPath, source, kind, target), nil)
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

// DeleteFileSubgraph stages two Pebble range-deletes — path's own file
// record under f/ (D-03: [fileSubgraphPrefix(path), rangeUpperBound(...)),
// the exact byte range that isolates path's record from a
// lexicographically adjacent sibling, e.g. "foo" vs "foobar") and path's
// whole file-index region under x/ (Phase 4 D-02: [fileIndexPrefix(path),
// rangeUpperBound(...)), covering both the node and edge sub-ranges
// together) — as one logical "prune this file entirely" call from the
// caller's perspective. It does not, by itself, prune the file's
// scattered n/e records elsewhere in the keyspace: callers enumerate
// IterateFileIndex(path) BEFORE calling this, and stage DeleteNode/
// DeleteEdge for each entry found, so the x/ range this deletes has
// already been read.
func (w *pebbleWriter) DeleteFileSubgraph(path string) error {
	start := fileSubgraphPrefix(path)
	end := rangeUpperBound(start)
	if err := w.batch.DeleteRange(start, end, nil); err != nil {
		return err
	}
	xStart := fileIndexPrefix(path)
	xEnd := rangeUpperBound(xStart)
	return w.batch.DeleteRange(xStart, xEnd, nil)
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
