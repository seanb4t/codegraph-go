package gitmeta

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCachingDetectorMemoizesPositive(t *testing.T) {
	startPath, indexRoot := newLinkedWorktreeFixture(t)
	d := NewCachingDetector()

	v1 := d.Detect(context.Background(), startPath, indexRoot)
	v2 := d.Detect(context.Background(), startPath, indexRoot)

	if v1 == nil || v2 == nil {
		t.Fatalf("Detect() = %v, %v; want both non-nil", v1, v2)
	}
	if *v1 != *v2 {
		t.Fatalf("Detect() verdicts differ across calls: %+v vs %+v", v1, v2)
	}

	d.mu.Lock()
	n := len(d.cache)
	d.mu.Unlock()
	if n != 1 {
		t.Fatalf("cache has %d entries after two calls to the same key, want 1", n)
	}
}

func TestCachingDetectorMemoizesNegative(t *testing.T) {
	startPath, indexRoot := newNonGitFixture(t)
	d := NewCachingDetector()

	v1 := d.Detect(context.Background(), startPath, indexRoot)
	v2 := d.Detect(context.Background(), startPath, indexRoot)
	if v1 != nil || v2 != nil {
		t.Fatalf("Detect() = %v, %v; want both nil (no mismatch)", v1, v2)
	}

	key := startPath + "\x00" + indexRoot
	d.mu.Lock()
	v, ok := d.cache[key]
	n := len(d.cache)
	d.mu.Unlock()
	if !ok {
		t.Fatalf("cache has no entry for key %q after Detect — negative verdicts must be cached (D-13)", key)
	}
	if v != nil {
		t.Fatalf("cached value = %v, want nil", v)
	}
	if n != 1 {
		t.Fatalf("cache has %d entries, want 1", n)
	}
}

func TestCachingDetectorNilReceiverSafety(t *testing.T) {
	var d *CachingDetector
	startPath, indexRoot := newNonGitFixture(t)
	// Must not panic; falls through directly to DetectIndexMismatch.
	got := d.Detect(context.Background(), startPath, indexRoot)
	if got != nil {
		t.Fatalf("Detect() on nil receiver = %v, want nil", got)
	}
}

// TestCachingDetectorRejectsNonexistentStartPath is WR-02's regression pin:
// a startPath that does not exist on disk (or is not a directory) never
// grows the cache — query.OpenAt succeeds for nonexistent paths, so a
// looping/malicious MCP client minting a fresh "path" argument per call
// must not mint a fresh, permanently-retained cache entry per call.
func TestCachingDetectorRejectsNonexistentStartPath(t *testing.T) {
	_, indexRoot := newNonGitFixture(t)
	d := NewCachingDetector()

	for i := 0; i < 5; i++ {
		got := d.Detect(context.Background(), filepath.Join(indexRoot, "does-not-exist"), indexRoot)
		if got != nil {
			t.Fatalf("Detect() on a nonexistent startPath = %v, want nil", got)
		}
	}

	d.mu.Lock()
	n := len(d.cache)
	d.mu.Unlock()
	if n != 0 {
		t.Fatalf("cache has %d entries after repeated calls with a nonexistent startPath, want 0", n)
	}
}

// TestCachingDetectorCancelledContextNotCached is BL-01's unit-level pin:
// a Detect call made with an ALREADY-CANCELLED context must degrade to nil
// for THAT call (WORK-03: never block/error a read on a failed git probe)
// but must NOT write that nil into the cache — otherwise the very next
// call, even with a perfectly healthy context, would read back the stale
// "clean" verdict forever. startPath/indexRoot here are a REAL positive
// (linked-worktree) fixture specifically so a wrongly-cached nil is
// distinguishable from the correct, uncached positive verdict.
func TestCachingDetectorCancelledContextNotCached(t *testing.T) {
	startPath, indexRoot := newLinkedWorktreeFixture(t)
	d := NewCachingDetector()

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if got := d.Detect(cancelledCtx, startPath, indexRoot); got != nil {
		t.Fatalf("Detect() under a cancelled ctx = %v, want nil (degraded, not an error)", got)
	}

	d.mu.Lock()
	_, cached := d.cache[startPath+"\x00"+indexRoot]
	d.mu.Unlock()
	if cached {
		t.Fatal("BL-01 REGRESSION: a verdict computed under a cancelled context was written to the cache")
	}

	// The next call, with a HEALTHY context, must re-probe and find the
	// real positive verdict — not inherit a poisoned nil from the call above.
	got := d.Detect(context.Background(), startPath, indexRoot)
	if got == nil {
		t.Fatal("BL-01 REGRESSION: after one cancelled Detect call, a subsequent healthy call on the same detector still returns nil — the cancelled call poisoned the cache")
	}
}

// TestCachingDetectorBounded is WR-02's cache-growth pin: once the cache
// reaches maxCacheEntries, the NEXT Detect call resets it rather than
// growing further — a bound that survives a client minting an unbounded
// number of distinct, EXISTING startPath directories (the maxCacheEntries
// doc comment covers the nonexistent-path case, already rejected before
// this point by TestCachingDetectorRejectsNonexistentStartPath).
func TestCachingDetectorBounded(t *testing.T) {
	_, indexRoot := newNonGitFixture(t)
	d := NewCachingDetector()

	// Fill the cache past its bound with distinct, real, non-git
	// directories (each a legitimate, distinguishable cache key).
	for i := 0; i < maxCacheEntries+10; i++ {
		start := t.TempDir()
		d.Detect(context.Background(), start, indexRoot)
	}

	d.mu.Lock()
	n := len(d.cache)
	d.mu.Unlock()
	if n > maxCacheEntries {
		t.Fatalf("cache has %d entries, want at most maxCacheEntries (%d)", n, maxCacheEntries)
	}
}
