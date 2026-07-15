package query

import (
	"context"
	"io"
	"path/filepath"
	"sync"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
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
//
// repoRoot and startPath are NOT the same thing, and the distinction is
// the whole of D-14's worktree-detection bug: repoRoot is the RESOLVED
// INDEX root — the output of ResolveCodegraphDir's upward walk from
// whatever directory the caller started at, i.e. wherever the nearest
// ancestor .codegraph/ actually lives. startPath is where the caller
// ACTUALLY STOOD when they issued the query — OpenAt's start argument,
// absolutized. When start lives inside a linked worktree that has no
// .codegraph/ of its own, the upward walk resolves repoRoot to a
// DIFFERENT working tree (the main checkout), and worktree detection
// (WorktreeMismatch) is precisely the comparison between these two paths.
// Engines built via New/NewWithRoot carry no startPath (zero value "") and
// therefore have no context to detect a mismatch against — see
// WorktreeMismatch's degrade-safely contract.
type Engine struct {
	reader   graphstore.Reader
	repoRoot string

	// startPath is the caller's absolutized starting directory (D-14),
	// set only by OpenAt. Empty for Engines built via New/NewWithRoot.
	startPath string

	// detector is an optional, caller-injected shared cache (D-13,
	// corrected). See UseDetector's doc comment for why this cannot live
	// only on Engine. A nil detector is safe to use — CachingDetector.Detect
	// is nil-receiver-safe and falls through to an uncached
	// gitmeta.DetectIndexMismatch call.
	detector *gitmeta.CachingDetector

	mismatchOnce  sync.Once
	mismatchCache *gitmeta.Mismatch
}

// New wraps an already-open graphstore.Reader in an Engine with no repo
// root configured. Most callers should use OpenAt instead; New exists so
// tests (and OpenAt itself, via NewWithRoot) can construct an Engine from a
// Reader obtained however is convenient. An Engine built via New rejects
// Node (file mode) and Explore's disk reads with a clear error, since it
// has no repo root to confine them to, and reports no worktree mismatch
// (no startPath, no repoRoot — see WorktreeMismatch).
func New(r graphstore.Reader) *Engine {
	return &Engine{reader: r}
}

// NewWithRoot wraps reader together with repoRoot (03-06, D-05a) — the
// directory Node (file mode) and Explore resolve every on-disk source read
// against and confine it to (path-traversal defense, T-03-06-Path). No
// startPath is set, so WorktreeMismatch degrades to nil (D-14) — only
// OpenAt supplies the caller's original starting directory.
func NewWithRoot(r graphstore.Reader, repoRoot string) *Engine {
	return &Engine{reader: r, repoRoot: repoRoot}
}

// UseDetector installs a shared, caller-owned gitmeta.CachingDetector that
// WorktreeMismatch will route detection through instead of computing an
// uncached, Engine-private verdict.
//
// This exists because internal/mcp's openEngine builds a FRESH Engine on
// every single tool call by design (D-02/D-08b, RESEARCH Pitfall 2) — an
// Engine-scoped cache alone (the mismatchOnce/mismatchCache fields below)
// yields ZERO cross-call benefit on the exact long-lived surface
// (the MCP server process) the cache exists to help, since a brand-new
// Engine means a brand-new sync.Once every time. internal/mcp therefore
// constructs ONE CachingDetector per server (BuildServer) and calls
// UseDetector on every Engine it opens, so all tool calls within one
// server's lifetime share a single git-subprocess-avoiding cache (D-13,
// corrected). The CLI needs no such call — it constructs at most one
// Engine per invocation, so the per-Engine cache alone is already free.
func (e *Engine) UseDetector(d *gitmeta.CachingDetector) {
	e.detector = d
}

// WorktreeMismatch returns the live worktree/index-mismatch verdict for
// this Engine (D-14/WORK-01): whether startPath (where the caller stood)
// belongs to a DIFFERENT git working tree than repoRoot (the resolved
// index root). Detection runs at most once per Engine (guarded by
// mismatchOnce) — Status() and any render path that also calls this method
// within one request never re-spawn git.
//
// Returns nil, without ever panicking, when this Engine has no filesystem
// context to check: startPath == "" or repoRoot == "" (Engines built via
// New/NewWithRoot — the same degrade-safely shape computeStale already
// uses for e.repoRoot == ""). Otherwise delegates to e.detector.Detect,
// which is nil-receiver-safe (falls through to an uncached
// gitmeta.DetectIndexMismatch when no detector was injected via
// UseDetector) — so no nil branch is needed here.
func (e *Engine) WorktreeMismatch() *gitmeta.Mismatch {
	e.mismatchOnce.Do(func() {
		if e.startPath == "" || e.repoRoot == "" {
			return
		}
		e.mismatchCache = e.detector.Detect(context.Background(), e.startPath, e.repoRoot)
	})
	return e.mismatchCache
}

// OpenAt is the single read seam CLI commands and MCP tool handlers both
// call (D-08b): it resolves the nearest .codegraph/ at or above start
// (ResolveCodegraphDir), opens the GraphStore at its store subdirectory,
// takes a fresh Snapshot (D-02 — one snapshot per invocation, never
// reused across calls), and returns an Engine wrapping that snapshot plus
// an io.Closer that releases both the Reader and the underlying store.
//
// The returned Engine also retains start, absolutized, as startPath
// (D-14) — the side of worktree detection ResolveCodegraphDir's upward
// walk does NOT capture (its return value becomes repoRoot, the resolved
// index root, which may differ from where the caller actually stood). On
// a filepath.Abs error, startPath is left empty rather than failing
// OpenAt — a status query must not die because a path could not be
// absolutized (WORK-03); WorktreeMismatch simply degrades to nil in that
// case.
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

	eng := NewWithRoot(reader, dir)
	// Pitfall 3: resolveStartPath (internal/cli/query.go) may hand OpenAt
	// a RELATIVE --path value. Absolutizing here mirrors TS's own
	// path.resolve(pathArg || process.cwd()) at its CLI entry point and
	// ResolveCodegraphDir's own filepath.Abs call on the same input — a
	// relative startPath would not be byte-comparable against
	// EvalSymlinks-resolved paths downstream, breaking gate 2's equality
	// check in gitmeta.DetectIndexMismatch.
	if abs, err := filepath.Abs(start); err == nil {
		eng.startPath = abs
	}

	return eng, &engineCloser{reader: reader, store: store}, nil
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
