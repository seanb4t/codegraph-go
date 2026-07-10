package graphstore

import (
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

// Commit applies the batch atomically and durably (pebble.Sync).
func (w *pebbleWriter) Commit() error {
	return w.batch.Commit(pebble.Sync)
}
