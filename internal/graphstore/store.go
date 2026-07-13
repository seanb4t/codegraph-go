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

	// GetMigration returns the migration-progress cursor blob (07-02) — an
	// opaque payload owned and encoded by the caller (internal/migrate),
	// never interpreted here. Lives under its own m/migration meta key,
	// distinct from the m/schema Meta record GetMeta returns. Returns
	// ErrNotFound if no migration progress record has ever been written.
	GetMigration() ([]byte, error)

	// IterateEdges returns an EdgeIterator over every edge whose source
	// is srcPrefix, ordered by (src, kind, dst) — a single contiguous
	// range scan (D-03), suitable for callers/callees/impact queries.
	IterateEdges(srcPrefix string) (EdgeIterator, error)

	// IterateNodes returns a NodeIterator over every node record in the
	// store — a single contiguous range scan over the whole n/ namespace
	// (D-03). Phase 3 v1 kind-filtering, if any, is applied in-memory by
	// the caller/query engine: the stored n/ key length-prefixes the
	// whole id (keys.go's appendSegment), so a byte-prefix scan cannot
	// cleanly isolate one kind (RESEARCH Pitfall 1).
	IterateNodes() (NodeIterator, error)

	// IterateFiles returns a FileIterator over every file record under
	// the f/ namespace — a single contiguous range scan (D-03).
	IterateFiles() (FileIterator, error)

	// IterateFileIndex returns a FileIndexIterator over every x/ entry
	// path owns — its node ids and outgoing-edge triples — as a single
	// contiguous range scan (Phase 4 D-02). Sync's prune step uses this
	// to find the exact n/e keys to point-delete via DeleteNode/
	// DeleteEdge for a changed/deleted file's scattered subgraph.
	IterateFileIndex(path string) (FileIndexIterator, error)

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

// NodeIterator walks a contiguous range of Node records. Callers must call
// Next before the first call to Node, and check Err after Next returns
// false to distinguish end-of-range from an error.
type NodeIterator interface {
	// Next advances the iterator and reports whether a record is
	// available.
	Next() bool

	// Node returns the record at the iterator's current position. Only
	// valid after a call to Next that returned true.
	Node() *schema.Node

	// Err returns the first error encountered during iteration, if any.
	Err() error

	// Close releases the iterator's resources.
	Close() error
}

// FileIterator walks a contiguous range of File records. Callers must call
// Next before the first call to File, and check Err after Next returns
// false to distinguish end-of-range from an error.
type FileIterator interface {
	// Next advances the iterator and reports whether a record is
	// available.
	Next() bool

	// File returns the record at the iterator's current position. Only
	// valid after a call to Next that returned true.
	File() *schema.File

	// Err returns the first error encountered during iteration, if any.
	Err() error

	// Close releases the iterator's resources.
	Close() error
}

// FileIndexEntry is one decoded x/ file-index record: either a node
// reference (IsNode true, NodeID set) or an edge reference (IsNode false,
// Source/Kind/Target set — the owning file's outgoing edge triple). The x/
// namespace stores no value payload; every field here is decoded straight
// from the key bytes (Phase 4 D-02).
type FileIndexEntry struct {
	IsNode bool
	NodeID string

	Source, Kind, Target string
}

// FileIndexIterator walks a contiguous range of one file's x/ index
// entries (both its owned node ids and outgoing edge triples). Callers
// must call Next before the first call to Entry, and check Err after Next
// returns false to distinguish end-of-range from an error.
type FileIndexIterator interface {
	// Next advances the iterator and reports whether a record is
	// available.
	Next() bool

	// Entry returns the decoded record at the iterator's current
	// position. Only valid after a call to Next that returned true.
	Entry() FileIndexEntry

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
	// behavior this implies. ownerPath is the file that owns e's source
	// node (Phase 4 D-02) — when non-empty, PutEdge also stages the
	// corresponding x/ file-index entry so the owning file's outgoing
	// edges are enumerable via IterateFileIndex.
	PutEdge(e *schema.Edge, ownerPath string) error

	// PutFile stages f for write. f.Path determines its key (D-03).
	PutFile(f *schema.File) error

	// PutMeta stages the store-wide Meta record for write.
	PutMeta(m *schema.Meta) error

	// PutMigration stages the migration-progress cursor blob for write
	// (07-02). data is an opaque payload owned and encoded by the caller
	// (internal/migrate) — this is a raw byte Set, not a proto-marshaled
	// record, and it is stored under its own m/migration meta key,
	// distinct from and never overwriting the m/schema Meta record PutMeta
	// writes.
	PutMigration(data []byte) error

	// DeleteNode stages a point-delete of the node record identified by
	// id (Phase 4 D-02) — the mechanism Sync's prune step uses after
	// finding id via IterateFileIndex.
	DeleteNode(id string) error

	// DeleteEdge stages a point-delete of the edge record identified by
	// (source, kind, target) (Phase 4 D-02) — the mechanism Sync's prune
	// step uses after finding the triple via IterateFileIndex.
	DeleteEdge(source, kind, target string) error

	// DeleteFileIndexEdge stages a point-delete of ownerPath's own x/
	// file-index entry for the outgoing edge (source, kind, target) (Phase
	// 4 D-02, CR-04) — WITHOUT touching the edge's own e/ record. Callers
	// pair this with DeleteEdge when discarding a single owned edge in
	// isolation (Sync's pruneOwnedEdgesOnly, for a dependent file whose
	// own nodes/File record survive): a plain DeleteEdge alone would leave
	// a stale x/<ownerPath>/e/<source>/<kind>/<target> entry with no
	// matching e/ record, which a later IterateFileIndex(ownerPath) scan
	// (e.g. a subsequent DIRECT prune of ownerPath) would enumerate and
	// phantom-count as a real removal. Callers doing a FULL file prune use
	// DeleteFileSubgraph instead, which range-deletes the owning file's
	// whole x/ region — including every edge entry — in one call; this
	// method exists for the narrower "just this one edge" case
	// DeleteFileSubgraph does not cover.
	DeleteFileIndexEdge(ownerPath, source, kind, target string) error

	// DeleteFileSubgraph stages a range-delete over path's own file
	// record AND every x/<path>/... file-index entry it owns (D-03,
	// extended by Phase 4 D-02) — the mechanism Phase 4's rename/delete
	// pruning binds to. Still one logical "prune this file entirely"
	// call from the caller's perspective; the node/edge records
	// themselves are pruned separately via DeleteNode/DeleteEdge, driven
	// by an IterateFileIndex scan taken before this call.
	DeleteFileSubgraph(path string) error

	// Commit atomically applies every staged mutation. Do not reuse a
	// Writer after Commit.
	Commit() error

	// Close releases the Writer's underlying batch without applying it.
	// Callers that abandon a Writer without calling Commit (e.g. after a
	// marshal error mid-stage) MUST call Close to release the batch.
	// Calling Close after a successful Commit is a safe no-op.
	Close() error
}
