package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
)

// TestStalenessSidecarPresent pins the D-04a watcher/daemon signal: a
// `.codegraph/.sync-pending` sidecar present (however fresh the graph
// itself is) always reports Stale, regardless of the mtime fallback.
func TestStalenessSidecarPresent(t *testing.T) {
	engine, dir := filesStatusFixture(t)

	sidecar := filepath.Join(dir, indexdir.Name, staleSidecarName)
	if err := os.WriteFile(sidecar, nil, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	got, err := engine.Status(context.Background())
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

	got, err := engine.Status(context.Background())
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

	got, err := engine.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: unexpected error: %v", err)
	}
	if got.Stale {
		t.Fatal("Status.Stale: got true, want false (freshly indexed, no sidecar, no newer files)")
	}
}

// TestNewestSourceMtimeSkipsTheIndexBehindALink pins the walk of the
// fallback to the sources: a store that links to a directory at the root,
// its lock files and its scratch directories all hold newer files, and
// none of them counts.
func TestNewestSourceMtimeSkipsTheIndexBehindALink(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	source := filepath.Join(root, "main.go")
	if err := os.WriteFile(source, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(source, old, old); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"store-real/000001.log", "store-real.swap", "store-real.write", "store-real.scratch-abc/store/000002.sst"} {
		path := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, indexdir.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "store-real"), filepath.Join(root, indexdir.Name, indexdir.StoreName)); err != nil {
		t.Fatal(err)
	}

	got, err := newestSourceMtime(root)
	if err != nil {
		t.Fatalf("newestSourceMtime: %v", err)
	}
	if !got.Equal(old) {
		t.Fatalf("newestSourceMtime = %v, want %v, the time of main.go", got, old)
	}
}
