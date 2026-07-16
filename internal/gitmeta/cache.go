package gitmeta

import (
	"context"
	"os"
	"sync"
)

// maxCacheEntries bounds CachingDetector's map (WR-02): on a long-lived
// MCP server, startPath is derived from a client-supplied "path" argument,
// and query.OpenAt succeeds for paths that don't even exist (the upward
// .codegraph/ walk just keeps going) — a looping or malicious client can
// therefore mint an unbounded number of distinct, permanently-retained
// cache keys, each also costing a git spawn on first sight (defeating the
// cache's own purpose). Simplest bounded policy: once the cache is full,
// drop it and start over — an LRU would preserve more hot entries, but at
// this scale (one entry per distinct (startPath, indexRoot) pair actually
// seen) a full reset is a rare, cheap event, not a hot path.
const maxCacheEntries = 1024

// CachingDetector memoizes DetectIndexMismatch verdicts — POSITIVE and
// NEGATIVE — keyed on TS's own cache key (startPath + "\x00" + indexRoot,
// per D-13). Detection costs up to four git subprocesses (see
// DetectIndexMismatch's doc comment); on a long-lived MCP server that would
// otherwise re-pay that cost on every single tool call.
//
// The cache deliberately lives HERE, not on internal/query.Engine:
// internal/mcp's openEngine builds a FRESH Engine on every single tool call
// by design, so an Engine-scoped cache would yield zero cross-call benefit
// on the exact long-lived surface the cache exists for. internal/mcp
// constructs one CachingDetector per server and closes over it in every
// handler; the CLI constructs one per invocation (free — it's one-shot);
// both surfaces share this identical type (D-13, corrected 2026-07-15).
type CachingDetector struct {
	mu    sync.Mutex
	cache map[string]*Mismatch
}

// NewCachingDetector returns a ready-to-use, empty CachingDetector.
func NewCachingDetector() *CachingDetector {
	return &CachingDetector{cache: make(map[string]*Mismatch)}
}

// Detect returns the memoized DetectIndexMismatch verdict for
// (startPath, indexRoot), computing and caching it on first call. Negative
// verdicts (nil == "checked, no mismatch") are cached too — a bare nil
// lookup can't distinguish "not yet checked" from "checked, none found", so
// presence is tracked via the two-value map form, never a nil test (D-13).
//
// Detect is safe to call on a nil *CachingDetector: it falls through
// directly to DetectIndexMismatch, uncached, so every consumer can treat
// the detector as optional.
//
// WR-02: a startPath that is not an existing, statable directory can never
// be inside a working tree — DetectIndexMismatch's gate 1 (WorktreeRoot)
// would immediately return "" for it anyway, so this is a pure
// short-circuit, not a behavior change. Rejecting it BEFORE the cache
// lookup/store means a client that mints a fresh nonexistent "path" on
// every call (accidentally, via a stale reference, or a malicious/looping
// MCP client) cannot grow the cache at all, on top of the maxCacheEntries
// bound below for legitimate, existing paths.
func (d *CachingDetector) Detect(ctx context.Context, startPath, indexRoot string) *Mismatch {
	if d == nil {
		return DetectIndexMismatch(ctx, startPath, indexRoot)
	}

	if fi, err := os.Stat(startPath); err != nil || !fi.IsDir() {
		return nil
	}

	key := startPath + "\x00" + indexRoot

	d.mu.Lock()
	if v, ok := d.cache[key]; ok {
		d.mu.Unlock()
		return v
	}
	d.mu.Unlock()

	v := DetectIndexMismatch(ctx, startPath, indexRoot)

	d.mu.Lock()
	// WR-02: once the cache is full, reset it rather than growing without
	// bound — see maxCacheEntries' doc comment for the threat this guards
	// against and why a full reset (not an LRU) is an acceptable policy
	// here.
	if len(d.cache) >= maxCacheEntries {
		d.cache = make(map[string]*Mismatch)
	}
	d.cache[key] = v
	d.mu.Unlock()

	return v
}
