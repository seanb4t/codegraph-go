package query

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStalenessSidecarPresent pins the D-04a watcher/daemon signal: a
// `.codegraph/.sync-pending` sidecar present (however fresh the graph
// itself is) always reports Stale, regardless of the mtime fallback.
func TestStalenessSidecarPresent(t *testing.T) {
	engine, dir := filesStatusFixture(t)

	sidecar := filepath.Join(dir, codegraphDirName, staleSidecarName)
	if err := os.WriteFile(sidecar, nil, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	got, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if !got.Stale {
		t.Fatal("Status.Stale: got false, want true (sidecar present)")
	}
}

// TestStalenessMtimeFallback pins the D-04a no-daemon fallback: with no
// sidecar present, a source file whose mtime is newer than
// Meta.last_sync_unix_ms marks the graph stale.
func TestStalenessMtimeFallback(t *testing.T) {
	engine, dir := filesStatusFixture(t)

	future := time.Now().Add(time.Hour)
	target := filepath.Join(dir, "main.go")
	if err := os.Chtimes(target, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	got, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if !got.Stale {
		t.Fatal("Status.Stale: got false, want true (a file's mtime is newer than last_sync_unix_ms)")
	}
}

// TestStalenessFreshNotStale pins the not-stale steady state: a graph
// indexed after its fixture files were written (so last_sync_unix_ms is
// newer than every file's mtime) with no sidecar present reports not
// stale.
func TestStalenessFreshNotStale(t *testing.T) {
	engine, _ := filesStatusFixture(t)

	got, err := engine.Status()
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if got.Stale {
		t.Fatal("Status.Stale: got true, want false (freshly indexed, no sidecar, no newer files)")
	}
}
