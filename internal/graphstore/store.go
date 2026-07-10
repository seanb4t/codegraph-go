package graphstore

import (
	"io"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// GraphStore is the only door onto the embedded key-value engine (D-04).
// No package outside internal/graphstore (and its own subpackages) may
// import the engine directly — archtest.TestNoPackageBypassesGraphStore
// enforces this boundary at test time (D-04a).
//
// Concurrency model: many lock-free readers via Snapshot, plus one
// coordinated writer via NewWriter, with no external locking required
// (INDX-05).
type GraphStore interface {
	// Snapshot returns a consistent, point-in-time Reader. Multiple
	// snapshots may be open concurrently with an in-flight writer; Pebble
	// coordinates this without pinning memtables or blocking readers.
	Snapshot() (Reader, error)

	// NewWriter returns a batched Writer scoped to one file-change /
	// debounce window. Callers commit once; the write path is not meant
	// for one engine write per symbol (D-04).
	NewWriter() (Writer, error)

	// Export streams every record (meta, nodes, edges, files) in
	// schema-versioned form from a consistent snapshot (ARCH-01).
	Export(w io.Writer) error

	// Close releases the underlying engine handle.
	Close() error
}

// Reader is a consistent, point-in-time view of the graph. A Reader must
// be Closed when no longer needed, to release its underlying snapshot.
type Reader interface {
	// GetNode looks up a single node by id. Returns ErrNotFound if absent.
	GetNode(id string) (*schema.Node, error)

	// GetFile looks up a single file record by path. Returns ErrNotFound
	// if absent.
	GetFile(path string) (*schema.File, error)

	// GetMeta returns the store-wide Meta record (schema version,
	// aggregate counts, index health). Returns ErrNotFound if the store
	// has never had a Meta record written.
	GetMeta() (*schema.Meta, error)

	// IterateEdges returns an EdgeIterator over every edge whose source
	// is srcPrefix, ordered by (src, kind, dst) — a single contiguous
	// range scan (D-03), suitable for callers/callees/impact queries.
	IterateEdges(srcPrefix string) (EdgeIterator, error)

	// Close releases the Reader's underlying snapshot.
	Close() error
}

// EdgeIterator walks a contiguous range of Edge records. Callers must call
// Next before the first call to Edge, and check Err after Next returns
// false to distinguish end-of-range from an error.
type EdgeIterator interface {
	// Next advances the iterator and reports whether a record is
	// available.
	Next() bool

	// Edge returns the record at the iterator's current position. Only
	// valid after a call to Next that returned true.
	Edge() *schema.Edge

	// Err returns the first error encountered during iteration, if any.
	Err() error

	// Close releases the iterator's resources.
	Close() error
}

// Writer batches graph mutations for one file-change / debounce window. A
// Writer commits atomically: either every staged Put/Delete is applied, or
// none is (D-04).
type Writer interface {
	// PutNode stages n for write. n.Id determines its key (D-03).
	PutNode(n *schema.Node) error

	// PutEdge stages e for write. (e.Source, e.Kind, e.Target) determine
	// its key (D-03); a second PutEdge with the same triple overwrites
	// the first — see keys.go's edgeKey doc for the deliberate dedup
	// behavior this implies.
	PutEdge(e *schema.Edge) error

	// PutFile stages f for write. f.Path determines its key (D-03).
	PutFile(f *schema.File) error

	// PutMeta stages the store-wide Meta record for write.
	PutMeta(m *schema.Meta) error

	// DeleteFileSubgraph stages a single range-delete over path's own
	// file record (D-03) — the mechanism Phase 4's rename/delete pruning
	// binds to.
	DeleteFileSubgraph(path string) error

	// Commit atomically applies every staged mutation. Do not reuse a
	// Writer after Commit.
	Commit() error
}
