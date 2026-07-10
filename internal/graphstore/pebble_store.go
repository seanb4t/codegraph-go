package graphstore

import (
	"errors"
	"io"
	"sync/atomic"

	"github.com/cockroachdb/pebble/v2"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// ErrNotFound is returned by Reader lookups when the requested record does
// not exist. Callers compare against this sentinel rather than importing
// pebble to check pebble.ErrNotFound directly — reaching for the engine's
// own error type outside this package would itself be a D-04a bypass.
var ErrNotFound = errors.New("graphstore: not found")

// ErrClosed is returned by Snapshot, NewWriter, and Export once the store
// has been Closed, instead of letting the call reach pebble's own
// closed-DB panic path (pebble/v2 panics rather than returning an error
// once its *pebble.DB is closed — verified against db.go's NewSnapshot
// and applyInternal). Commit on a Writer obtained just before Close is a
// harder race to close entirely without additional coordination and is
// not guarded by this sentinel.
var ErrClosed = errors.New("graphstore: store is closed")

// metaRecordName is the single well-known meta/ key name under which the
// store's Meta record (schema version, aggregate counts, health) lives.
const metaRecordName = "schema"

// pebbleStore is the sole holder of a *pebble.DB in the whole module
// (D-04). Every other package reaches the graph exclusively through the
// GraphStore/Reader/Writer/EdgeIterator interfaces declared in store.go —
// archtest.TestNoPackageBypassesGraphStore enforces this at test time
// (D-04a).
type pebbleStore struct {
	db     *pebble.DB
	closed atomic.Bool
}

// Open opens (creating if necessary) a pebble/v2-backed GraphStore at dir.
func Open(dir string) (GraphStore, error) {
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &pebbleStore{db: db}, nil
}

// Snapshot returns a consistent, point-in-time Reader (INDX-05): Pebble
// snapshots do not pin memtables or block the writer, so this call is
// lock-free with respect to any in-flight Writer.
func (s *pebbleStore) Snapshot() (Reader, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}
	return &pebbleReader{snap: s.db.NewSnapshot()}, nil
}

// NewWriter returns a batched Writer. It wraps a plain Pebble Batch, not an
// IndexedBatch: this write path needs no read-your-writes within the
// batch, and an indexed batch is slower for inserts.
func (s *pebbleStore) NewWriter() (Writer, error) {
	if s.closed.Load() {
		return nil, ErrClosed
	}
	return &pebbleWriter{batch: s.db.NewBatch()}, nil
}

// Close releases the underlying Pebble handle. Safe to call once; marks
// the store closed before delegating to pebble so subsequent
// Snapshot/NewWriter/Export calls observe ErrClosed instead of racing
// into pebble's own closed-DB panic path.
func (s *pebbleStore) Close() error {
	s.closed.Store(true)
	return s.db.Close()
}

// getter is satisfied by both *pebble.DB and *pebble.Snapshot; it is the
// minimal shape getProto needs to look up a single key.
type getter interface {
	Get(key []byte) (value []byte, closer io.Closer, err error)
}

// getProto looks up key via g and unmarshals its value into msg. It
// returns ErrNotFound (never the underlying pebble.ErrNotFound) when the
// key is absent, so callers never need to import pebble to interpret a
// miss.
func getProto(g getter, key []byte, msg proto.Message) error {
	val, closer, err := g.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	defer closer.Close()
	return proto.Unmarshal(val, msg)
}

// pebbleReader is the Reader implementation. Every read goes through a
// single Pebble snapshot captured at construction time, so every
// GetX/IterateEdges call on one pebbleReader observes the same consistent,
// point-in-time view (INDX-05).
type pebbleReader struct {
	snap *pebble.Snapshot
}

func (r *pebbleReader) GetNode(id string) (*schema.Node, error) {
	var n schema.Node
	if err := getProto(r.snap, nodeKey(id), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *pebbleReader) GetFile(path string) (*schema.File, error) {
	var f schema.File
	if err := getProto(r.snap, fileKey(path), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *pebbleReader) GetMeta() (*schema.Meta, error) {
	var m schema.Meta
	if err := getProto(r.snap, metaKey(metaRecordName), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *pebbleReader) IterateEdges(srcPrefix string) (EdgeIterator, error) {
	lower := edgeSrcPrefix(srcPrefix)
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleEdgeIterator{iter: iter}, nil
}

func (r *pebbleReader) Close() error {
	return r.snap.Close()
}

// pebbleEdgeIterator adapts a *pebble.Iterator ranging over one edge
// namespace prefix to the EdgeIterator interface.
type pebbleEdgeIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.Edge
	err     error
}

func (it *pebbleEdgeIterator) Next() bool {
	if it.err != nil {
		return false
	}
	var ok bool
	if !it.started {
		it.started = true
		ok = it.iter.First()
	} else {
		ok = it.iter.Next()
	}
	if !ok {
		if err := it.iter.Error(); err != nil {
			it.err = err
		}
		return false
	}
	var e schema.Edge
	if err := proto.Unmarshal(it.iter.Value(), &e); err != nil {
		it.err = err
		return false
	}
	it.cur = &e
	return true
}

func (it *pebbleEdgeIterator) Edge() *schema.Edge { return it.cur }
func (it *pebbleEdgeIterator) Err() error         { return it.err }
func (it *pebbleEdgeIterator) Close() error       { return it.iter.Close() }
