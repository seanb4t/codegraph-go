package graphstore

import (
	"errors"
	"fmt"
	"io"
	"sync"
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
//
// mu (WR-01) guards the closed-check-then-Pebble-call sequence in
// Snapshot/NewWriter/Export as one atomic critical section against a
// concurrent Close: Close takes mu's exclusive Lock, so it cannot mark the
// store closed and call db.Close() while any of those three calls is
// mid-flight holding RLock, and any call arriving after Close has released
// the lock is guaranteed to observe closed==true and return ErrClosed
// before ever touching db again — closing the plain atomic.Bool
// check-then-act race this file's own comments used to flag as latent
// (pebble/v2 panics, rather than returning an error, once its *pebble.DB
// is closed). Every current call site in this codebase opens/closes its
// own pebbleStore without sharing it across goroutines, so this was not
// exercised in practice, but the guard is cheap (RWMutex, uncontended in
// the common single-goroutine case) and closes the gap before a future
// long-lived shared store handle (e.g. an MCP-server-owned store) needs it.
//
// Commit on a Writer already obtained via NewWriter remains outside mu's
// protection — that race (a Writer whose Commit races a subsequent Close)
// is a harder problem this fix does not attempt, consistent with
// ErrClosed's existing doc.
type pebbleStore struct {
	db     *pebble.DB
	mu     sync.RWMutex
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
// lock-free with respect to any in-flight Writer. It IS guarded (RLock,
// WR-01) against a concurrent Close.
func (s *pebbleStore) Snapshot() (Reader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed.Load() {
		return nil, ErrClosed
	}
	return &pebbleReader{snap: s.db.NewSnapshot()}, nil
}

// NewWriter returns a batched Writer. It wraps a plain Pebble Batch, not an
// IndexedBatch: this write path needs no read-your-writes within the
// batch, and an indexed batch is slower for inserts. It IS guarded (RLock,
// WR-01) against a concurrent Close.
func (s *pebbleStore) NewWriter() (Writer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed.Load() {
		return nil, ErrClosed
	}
	return &pebbleWriter{batch: s.db.NewBatch()}, nil
}

// Close releases the underlying Pebble handle. Safe to call more than
// once: the first call marks the store closed (so subsequent
// Snapshot/NewWriter/Export calls observe ErrClosed) and delegates to
// pebble; every call after that is a no-op, since pebble.DB.Close()
// itself panics on a second invocation. Takes mu's exclusive Lock (WR-01)
// so it cannot run concurrently with any in-flight Snapshot/NewWriter/
// Export call.
func (s *pebbleStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Swap(true) {
		return nil
	}
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
	snap   *pebble.Snapshot
	closed atomic.Bool
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
	// srcPrefix == "" means "every edge" (D-04 — the reverse-adjacency
	// builder's single full-namespace scan), not "edges whose source is
	// the empty string". edgeSrcPrefix("") length-prefixes an empty src
	// segment, which — being a real, addressable (if never-written)
	// segment value in the appendSegment encoding — would otherwise
	// bound the scan to just that empty-src slice of the keyspace
	// instead of the whole e/ namespace. Mirror IterateNodes/
	// IterateFiles' whole-namespace-prefix pattern for this case.
	var lower []byte
	if srcPrefix == "" {
		lower = []byte{prefixEdge}
	} else {
		lower = edgeSrcPrefix(srcPrefix)
	}
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleEdgeIterator{iter: iter}, nil
}

func (r *pebbleReader) IterateNodes() (NodeIterator, error) {
	lower := []byte{prefixNode}
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleNodeIterator{iter: iter}, nil
}

func (r *pebbleReader) IterateFiles() (FileIterator, error) {
	lower := []byte{prefixFile}
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleFileIterator{iter: iter}, nil
}

// IterateFileIndex bounds a scan to exactly path's own x/ file-index
// entries — both its node and edge sub-ranges together (Phase 4 D-02).
func (r *pebbleReader) IterateFileIndex(path string) (FileIndexIterator, error) {
	lower := fileIndexPrefix(path)
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleFileIndexIterator{iter: iter}, nil
}

// Close releases the underlying Pebble snapshot. Safe to call more than
// once: *pebble.Snapshot.Close() panics on a second invocation, so repeat
// calls after the first are a no-op.
func (r *pebbleReader) Close() error {
	if r.closed.Swap(true) {
		return nil
	}
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

// pebbleNodeIterator adapts a *pebble.Iterator ranging over the whole n/
// namespace to the NodeIterator interface.
type pebbleNodeIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.Node
	err     error
}

func (it *pebbleNodeIterator) Next() bool {
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
	var n schema.Node
	if err := proto.Unmarshal(it.iter.Value(), &n); err != nil {
		it.err = err
		return false
	}
	it.cur = &n
	return true
}

func (it *pebbleNodeIterator) Node() *schema.Node { return it.cur }
func (it *pebbleNodeIterator) Err() error         { return it.err }
func (it *pebbleNodeIterator) Close() error       { return it.iter.Close() }

// pebbleFileIterator adapts a *pebble.Iterator ranging over the whole f/
// namespace to the FileIterator interface.
type pebbleFileIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.File
	err     error
}

func (it *pebbleFileIterator) Next() bool {
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
	var f schema.File
	if err := proto.Unmarshal(it.iter.Value(), &f); err != nil {
		it.err = err
		return false
	}
	it.cur = &f
	return true
}

func (it *pebbleFileIterator) File() *schema.File { return it.cur }
func (it *pebbleFileIterator) Err() error         { return it.err }
func (it *pebbleFileIterator) Close() error       { return it.iter.Close() }

// pebbleFileIndexIterator adapts a *pebble.Iterator ranging over one
// file's x/ index prefix to the FileIndexIterator interface. Unlike the
// other iterators above, the x/ namespace stores no value payload — every
// field of the decoded FileIndexEntry comes straight from the key bytes
// (decodeFileIndexKey), since the key itself is the reference (Phase 4
// D-02, mirrors pebbleNodeIterator's started/err/Next() discipline).
type pebbleFileIndexIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     FileIndexEntry
	err     error
}

func (it *pebbleFileIndexIterator) Next() bool {
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
	entry, err := decodeFileIndexKey(it.iter.Key())
	if err != nil {
		it.err = err
		return false
	}
	it.cur = entry
	return true
}

func (it *pebbleFileIndexIterator) Entry() FileIndexEntry { return it.cur }
func (it *pebbleFileIndexIterator) Err() error            { return it.err }
func (it *pebbleFileIndexIterator) Close() error          { return it.iter.Close() }

// decodeFileIndexKey reconstructs a FileIndexEntry from a raw x/ namespace
// key: [prefixFileIndex][path segment][marker byte][node-id segment |
// src/kind/dst segments]. The path segment's value is not decoded into
// the result — IterateFileIndex's caller already supplied path as the
// scan bound — only skipped over to reach the marker byte.
func decodeFileIndexKey(key []byte) (FileIndexEntry, error) {
	if len(key) == 0 || key[0] != prefixFileIndex {
		return FileIndexEntry{}, fmt.Errorf("graphstore: file-index key missing prefixFileIndex: %x", key)
	}
	_, offset, err := decodeSegment(key, 1)
	if err != nil {
		return FileIndexEntry{}, fmt.Errorf("graphstore: decode file-index path segment: %w", err)
	}
	if offset >= len(key) {
		return FileIndexEntry{}, fmt.Errorf("graphstore: file-index key missing marker byte: %x", key)
	}
	marker := key[offset]
	offset++

	switch marker {
	case fileIndexKindNode:
		nodeID, _, err := decodeSegment(key, offset)
		if err != nil {
			return FileIndexEntry{}, fmt.Errorf("graphstore: decode file-index node id: %w", err)
		}
		return FileIndexEntry{IsNode: true, NodeID: nodeID}, nil
	case fileIndexKindEdge:
		src, offset, err := decodeSegment(key, offset)
		if err != nil {
			return FileIndexEntry{}, fmt.Errorf("graphstore: decode file-index edge source: %w", err)
		}
		kind, offset, err := decodeSegment(key, offset)
		if err != nil {
			return FileIndexEntry{}, fmt.Errorf("graphstore: decode file-index edge kind: %w", err)
		}
		dst, _, err := decodeSegment(key, offset)
		if err != nil {
			return FileIndexEntry{}, fmt.Errorf("graphstore: decode file-index edge target: %w", err)
		}
		return FileIndexEntry{Source: src, Kind: kind, Target: dst}, nil
	default:
		return FileIndexEntry{}, fmt.Errorf("graphstore: unknown file-index marker byte %#x", marker)
	}
}
