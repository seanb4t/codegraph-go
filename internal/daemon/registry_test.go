package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// writeRawRecord writes raw bytes directly under the registry dir,
// bypassing Register — lets tests seed a record for a pid Register would
// never be called with (a dead pid) or malformed content.
func writeRawRecord(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("writing raw record %s: %v", name, err)
	}
}

// TestRegistryListPrunesStale is the plan's primary acceptance gate for
// Task 2 (D-05): List() self-heals by pruning a dead-pid record from disk
// and excluding it from the result, keeps a live-pid record, and skips a
// malformed record without erroring.
func TestRegistryListPrunesStale(t *testing.T) {
	dir := withRegistryDir(t)

	dead := deadPID(t)
	deadName := fmt.Sprintf("%d.json", dead)
	deadData, err := json.Marshal(Record{PID: dead, StartedAt: time.Now().UTC(), RepoRoot: "/dead/repo"})
	if err != nil {
		t.Fatalf("marshal dead record: %v", err)
	}
	writeRawRecord(t, dir, deadName, deadData)

	live := Record{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid()), RepoRoot: "/live/repo"}
	if err := Register(live); err != nil {
		t.Fatalf("Register live: %v", err)
	}

	writeRawRecord(t, dir, "not-a-pid.json", []byte("not valid json"))

	got, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0] != live {
		t.Fatalf("List: got %+v, want exactly the live record %+v", got, live)
	}

	if _, err := os.Stat(filepath.Join(dir, deadName)); !os.IsNotExist(err) {
		t.Fatalf("expected stale record file removed by List, stat err=%v", err)
	}
}

// TestRegistryListMissingDir is the DMON-04 empty edge: a registry dir that
// has never been created is not an error — List returns (nil, nil).
func TestRegistryListMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	prev := registryDir
	registryDir = func() (string, error) { return missing, nil }
	t.Cleanup(func() { registryDir = prev })

	got, err := List()
	if err != nil || got != nil {
		t.Fatalf("List on missing dir: got %v, err %v; want nil, nil", got, err)
	}
}

// TestRegistryListEmptyDir covers the same empty-edge contract when the
// registry dir exists but has no records yet.
func TestRegistryListEmptyDir(t *testing.T) {
	withRegistryDir(t)

	got, err := List()
	if err != nil || got != nil {
		t.Fatalf("List on empty dir: got %v, err %v; want nil, nil", got, err)
	}
}

// listNames returns the sorted file names directly under dir, or nil when
// dir does not exist, so a test can assert the exact contents of both
// sides of a migration in one comparison.
func listNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("ReadDir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// legacyRegistry builds <home>/.codegraph/daemons with the given record
// file names and returns the daemons directory.
func legacyRegistry(t *testing.T, home string, names ...string) string {
	t.Helper()
	legacy := filepath.Join(home, legacyStateDirName, registryDirName)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(legacy, n), []byte(`{"pid":1}`), 0o644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
	return legacy
}

// TestDefaultRegistryDirUsesStateDir: the registry lives under
// userdirs.StateDir(), honoring XDG_STATE_HOME, never under ~/.codegraph.
func TestDefaultRegistryDirUsesStateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	t.Setenv("XDG_STATE_HOME", "")
	if err := os.Unsetenv("XDG_STATE_HOME"); err != nil {
		t.Fatalf("unsetenv: %v", err)
	}
	got, err := defaultRegistryDir()
	if err != nil {
		t.Fatalf("defaultRegistryDir: %v", err)
	}
	if want := filepath.Join(home, ".local", "state", "codegraph", "daemons"); got != want {
		t.Fatalf("defaultRegistryDir with XDG_STATE_HOME unset: got %q, want %q", got, want)
	}

	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	got, err = defaultRegistryDir()
	if err != nil {
		t.Fatalf("defaultRegistryDir: %v", err)
	}
	if want := filepath.Join(state, "codegraph", "daemons"); got != want {
		t.Fatalf("defaultRegistryDir with XDG_STATE_HOME set: got %q, want %q", got, want)
	}
}

// TestMigrateLegacyRegistryMovesRecords: with nothing else under
// ~/.codegraph, both records move and both old directories are gone.
func TestMigrateLegacyRegistryMovesRecords(t *testing.T) {
	home := t.TempDir()
	legacy := legacyRegistry(t, home, "100.json", "200.json")
	dir := filepath.Join(t.TempDir(), "codegraph", "daemons")

	if err := migrateLegacyRegistry(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyRegistry: %v", err)
	}
	if got, want := listNames(t, dir), []string{"100.json", "200.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("new registry holds %v, want %v", got, want)
	}
	if _, err := os.Stat(filepath.Join(home, legacyStateDirName)); !os.IsNotExist(err) {
		t.Fatalf("~/.codegraph still exists after migration (stat err %v)", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "100.json"))
	if err != nil || string(data) != `{"pid":1}` {
		t.Fatalf("migrated record content = %q, err %v; want the original bytes", data, err)
	}
}

// TestMigrateLegacyRegistryKeepsForeignFiles: a file codegraph did not write
// keeps ~/.codegraph alive; only daemons/ is removed.
func TestMigrateLegacyRegistryKeepsForeignFiles(t *testing.T) {
	home := t.TempDir()
	legacy := legacyRegistry(t, home, "100.json")
	foreign := filepath.Join(home, legacyStateDirName, "telemetry.json")
	if err := os.WriteFile(foreign, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write foreign: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "codegraph", "daemons")

	if err := migrateLegacyRegistry(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyRegistry: %v", err)
	}
	if got, want := listNames(t, dir), []string{"100.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("new registry holds %v, want %v", got, want)
	}
	if got, want := listNames(t, filepath.Join(home, legacyStateDirName)), []string{"telemetry.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("~/.codegraph holds %v, want %v", got, want)
	}
}

// TestMigrateLegacyRegistryNoopWhenNewDirExists: once the new registry
// exists, old records are left where they are and nothing is deleted.
func TestMigrateLegacyRegistryNoopWhenNewDirExists(t *testing.T) {
	home := t.TempDir()
	legacy := legacyRegistry(t, home, "100.json")
	dir := filepath.Join(t.TempDir(), "codegraph", "daemons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}

	if err := migrateLegacyRegistry(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyRegistry: %v", err)
	}
	if got := listNames(t, dir); len(got) != 0 {
		t.Fatalf("new registry holds %v, want nothing", got)
	}
	if got, want := listNames(t, legacy), []string{"100.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy registry holds %v, want %v", got, want)
	}
}

// TestMigrateLegacyRegistryNoopWithoutLegacy: no old registry means no
// filesystem change at all, and no new directory either.
func TestMigrateLegacyRegistryNoopWithoutLegacy(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, legacyStateDirName, registryDirName)
	dir := filepath.Join(t.TempDir(), "codegraph", "daemons")

	if err := migrateLegacyRegistry(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyRegistry: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("new registry dir was created by a no-op migration (stat err %v)", err)
	}
	if got := listNames(t, home); len(got) != 0 {
		t.Fatalf("home holds %v after a no-op migration, want nothing", got)
	}
}
