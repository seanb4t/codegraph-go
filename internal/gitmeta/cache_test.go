package gitmeta

import (
	"context"
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
