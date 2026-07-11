package query

import (
	"errors"
	"io"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// storeSubdir is the Pebble store's subdirectory under .codegraph/,
// matching internal/cli's storeDirName constant (D-01b). It is
// re-declared here (not imported from internal/cli) so internal/query has
// no dependency on internal/cli — commands import query, not the other
// way around.
const storeSubdir = "store"

// Engine is the read-only query engine over a single graphstore.Reader
// snapshot (D-02). It implements no query verbs yet (those land in
// 03-03/03-04/03-05/03-06) — this plan establishes the construction seam
// every subsequent query method hangs off of.
type Engine struct {
	reader graphstore.Reader
}

// New wraps an already-open graphstore.Reader in an Engine. Most callers
// should use OpenAt instead; New exists so tests (and OpenAt itself) can
// construct an Engine from a Reader obtained however is convenient.
func New(r graphstore.Reader) *Engine {
	return &Engine{reader: r}
}

// OpenAt is the single read seam CLI commands and MCP tool handlers both
// call (D-08b): it resolves the nearest .codegraph/ at or above start
// (ResolveCodegraphDir), opens the GraphStore at its store subdirectory,
// takes a fresh Snapshot (D-02 — one snapshot per invocation, never
// reused across calls), and returns an Engine wrapping that snapshot plus
// an io.Closer that releases both the Reader and the underlying store.
//
// The returned closer is idempotent.
//
// TODO(03-02 GREEN): unimplemented — RED stub.
func OpenAt(start string) (*Engine, io.Closer, error) {
	return nil, nil, errors.New("query: OpenAt not implemented")
}
