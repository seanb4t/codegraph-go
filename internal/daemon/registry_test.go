package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withRegistryDir points the registryDir seam at a fresh t.TempDir() for the
// duration of the test, mirroring the onSync*/onWatchOpen test-seam
// convention (daemon_test.go) — restores the previous value in Cleanup so
// tests never touch the real ~/.codegraph/daemons/.
func withRegistryDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := registryDir
	registryDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { registryDir = prev })
	return dir
}

// TestRegistryRegisterDeregister is the plan's primary acceptance gate for
// Task 1: Register writes an atomically-readable <pid>.json round-tripping
// the Record, and Deregister removes it — a missing pid is a nil no-op
// (mirrors lock.go's release() shape, D-06).
func TestRegistryRegisterDeregister(t *testing.T) {
	dir := withRegistryDir(t)

	rec := Record{PID: 12345, StartedAt: time.Now().UTC().Truncate(time.Second), RepoRoot: "/repo/root"}
	if err := Register(rec); err != nil {
		t.Fatalf("Register: %v", err)
	}

	path := filepath.Join(dir, "12345.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading registered record: %v", err)
	}
	var got Record
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling registered record: %v", err)
	}
	if got != rec {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, rec)
	}

	if err := Deregister(rec.PID); err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected record file removed after Deregister, stat err=%v", err)
	}

	if err := Deregister(999999); err != nil {
		t.Fatalf("Deregister of a missing pid should be a nil no-op, got %v", err)
	}
}

// TestRegistrySameRepoRootDistinctFiles is the DMON-04 adjacency edge: two
// daemons for the same repoRoot are keyed by pid, never merged/overwritten.
func TestRegistrySameRepoRootDistinctFiles(t *testing.T) {
	dir := withRegistryDir(t)

	repoRoot := "/same/repo/root"
	recA := Record{PID: 111, StartedAt: time.Now().UTC(), RepoRoot: repoRoot}
	recB := Record{PID: 222, StartedAt: time.Now().UTC(), RepoRoot: repoRoot}

	if err := Register(recA); err != nil {
		t.Fatalf("Register recA: %v", err)
	}
	if err := Register(recB); err != nil {
		t.Fatalf("Register recB: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 distinct record files for the same repoRoot, got %d", len(entries))
	}
	if _, err := os.Stat(filepath.Join(dir, "111.json")); err != nil {
		t.Fatalf("expected 111.json to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "222.json")); err != nil {
		t.Fatalf("expected 222.json to exist: %v", err)
	}
}
