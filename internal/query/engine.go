package query

import (
	"io"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// storeSubdir is the Pebble store's subdirectory under .codegraph/,
// matching internal/cli's storeDirName constant (D-01b). It is
// re-declared here (not imported from internal/cli) so internal/query has
// no dependency on internal/cli — commands import query, not the other
// way around.
const storeSubdir = "store"

// Engine is the read-only query engine over a single graphstore.Reader
// snapshot (D-02), plus (03-06) the repo root Node (file mode) and Explore
// confine their on-disk source reads to (D-05a — source is read fresh from
// disk on every call, not from the stored Node/File record).
type Engine struct {
	reader   graphstore.Reader
	repoRoot string
}

// New wraps an already-open graphstore.Reader in an Engine with no repo
// root configured. Most callers should use OpenAt instead; New exists so
// tests (and OpenAt itself, via NewWithRoot) can construct an Engine from a
// Reader obtained however is convenient. An Engine built via New rejects
// Node (file mode) and Explore's disk reads with a clear error, since it
// has no repo root to confine them to.
func New(r graphstore.Reader) *Engine {
	return &Engine{reader: r}
}

// NewWithRoot wraps reader together with repoRoot (03-06, D-05a) — the
// directory Node (file mode) and Explore resolve every on-disk source read
// against and confine it to (path-traversal defense, T-03-06-Path).
func NewWithRoot(r graphstore.Reader, repoRoot string) *Engine {
	return &Engine{reader: r, repoRoot: repoRoot}
}

// OpenAt is the single read seam CLI commands and MCP tool handlers both
// call (D-08b): it resolves the nearest .codegraph/ at or above start
// (ResolveCodegraphDir), opens the GraphStore at its store subdirectory,
// takes a fresh Snapshot (D-02 — one snapshot per invocation, never
// reused across calls), and returns an Engine wrapping that snapshot plus
// an io.Closer that releases both the Reader and the underlying store.
//
// The returned closer is idempotent: calling Close more than once is
// safe and returns nil on the second and subsequent calls, so a defer
// alongside an early explicit Close (or vice versa) never double-frees
// the underlying Pebble handles.
func OpenAt(start string) (*Engine, io.Closer, error) {
	dir, err := ResolveCodegraphDir(start)
	if err != nil {
		return nil, nil, err
	}

	store, err := graphstore.Open(filepath.Join(dir, codegraphDirName, storeSubdir))
	if err != nil {
		return nil, nil, err
	}

	reader, err := store.Snapshot()
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}

	return NewWithRoot(reader, dir), &engineCloser{reader: reader, store: store}, nil
}

// engineCloser closes an Engine's Reader then its underlying GraphStore,
// exactly once, regardless of how many times Close is called.
type engineCloser struct {
	reader graphstore.Reader
	store  graphstore.GraphStore
	closed bool
}

// Close releases the Reader's snapshot and then the store's underlying
// engine handle. It is safe to call more than once — the second and
// subsequent calls are no-ops returning nil.
func (c *engineCloser) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

	var readerErr error
	if c.reader != nil {
		readerErr = c.reader.Close()
	}
	var storeErr error
	if c.store != nil {
		storeErr = c.store.Close()
	}
	if readerErr != nil {
		return readerErr
	}
	return storeErr
}
