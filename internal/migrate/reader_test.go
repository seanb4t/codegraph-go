package migrate

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/migrate/migratetest"
)

func TestOpenSource_HappyFixtureOpens(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)

	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestOpenSource_NonexistentPathErrors(t *testing.T) {
	_, err := OpenSource(filepath.Join(t.TempDir(), "does-not-exist.db"))
	if err == nil {
		t.Fatal("expected OpenSource to error on a nonexistent path")
	}
}

// TestSource_ByteIdentity proves the source .db file's bytes (and absence
// of -wal/-shm sidecars) are unchanged after a full OpenSource + ScanTable
// pass + Close (D-08 non-destructive-to-source).
func TestSource_ByteIdentity(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)

	before, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash before: %v", err)
	}

	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}

	for _, table := range []string{"files", "nodes", "edges"} {
		if err := src.ScanTable(table, 0, func(int64, map[string]any) error { return nil }); err != nil {
			t.Fatalf("ScanTable(%q): %v", table, err)
		}
	}

	if err := src.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	after, err := hashFile(dbPath)
	if err != nil {
		t.Fatalf("hash after: %v", err)
	}
	if before != after {
		t.Error("source db bytes changed after a read-only OpenSource+ScanTable pass")
	}

	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dbPath + suffix); err == nil {
			t.Errorf("unexpected sidecar file %s%s created by a read-only open", dbPath, suffix)
		}
	}
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func TestDetectTS_Happy(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	if err := src.DetectTS(); err != nil {
		t.Errorf("expected happy fixture to detect as a TS source: %v", err)
	}
}

func TestDetectTS_NotATSSource(t *testing.T) {
	dbPath := buildNonTSDB(t)
	src := openSourceT(t, dbPath)

	if err := src.DetectTS(); !errors.Is(err, ErrNotATSSource) {
		t.Errorf("expected ErrNotATSSource, got %v", err)
	}
}

func buildNonTSDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "not-ts.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("migrate_test: open %s: %v", dbPath, err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE unrelated (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("migrate_test: create unrelated table: %v", err)
	}
	return dbPath
}

func TestSchemaVersion_Happy(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	v, err := src.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if v != 7 {
		t.Errorf("expected schema version 7, got %d", v)
	}
}

func TestSchemaVersion_TooNewRejected(t *testing.T) {
	dbPath := buildSchemaVersionsDB(t, 1, 8)
	src := openSourceT(t, dbPath)

	if _, err := src.SchemaVersion(); !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("expected ErrUnsupportedSchemaVersion for version 8, got %v", err)
	}
}

func TestSchemaVersion_TooOldRejected(t *testing.T) {
	dbPath := buildSchemaVersionsDB(t, 0)
	src := openSourceT(t, dbPath)

	if _, err := src.SchemaVersion(); !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Errorf("expected ErrUnsupportedSchemaVersion for version 0, got %v", err)
	}
}

// buildSchemaVersionsDB creates a minimal DB with the three required TS
// tables (so OpenSource/DetectTS succeed) and a schema_versions table
// seeded with exactly the given version values — enough to exercise
// SchemaVersion's range guard without needing the full fixture harness.
func buildSchemaVersionsDB(t *testing.T, versions ...int) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sv.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("migrate_test: open %s: %v", dbPath, err)
	}
	defer db.Close()

	const ddl = `
CREATE TABLE schema_versions (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL, description TEXT);
CREATE TABLE nodes (id TEXT PRIMARY KEY);
CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, source TEXT, target TEXT, kind TEXT);
`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("migrate_test: exec ddl: %v", err)
	}
	for _, v := range versions {
		if _, err := db.Exec(`INSERT INTO schema_versions(version, applied_at, description) VALUES (?, 0, '')`, v); err != nil {
			t.Fatalf("migrate_test: insert schema_versions(%d): %v", v, err)
		}
	}
	return dbPath
}

func TestScanTable_AgedToleratesMissingColumns(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantAged)
	src := openSourceT(t, dbPath)

	for _, table := range []string{"files", "nodes", "edges"} {
		if err := src.ScanTable(table, 0, func(int64, map[string]any) error { return nil }); err != nil {
			t.Errorf("ScanTable(%q) on aged fixture: %v", table, err)
		}
	}
}

func TestScanTable_AllowListRejectsDisallowedTables(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	for _, table := range []string{"nodes_fts", "unresolved_refs", "sqlite_master", "schema_versions", "name_segment_vocab"} {
		t.Run(table, func(t *testing.T) {
			called := false
			err := src.ScanTable(table, 0, func(int64, map[string]any) error {
				called = true
				return nil
			})
			if err == nil {
				t.Errorf("expected ScanTable(%q) to error (not in allow-list)", table)
			}
			if called {
				t.Errorf("ScanTable(%q) invoked the callback despite being disallowed", table)
			}
		})
	}
}

func TestScanTable_ResumeCursor(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	var rowids []int64
	if err := src.ScanTable("edges", 2, func(rowid int64, _ map[string]any) error {
		rowids = append(rowids, rowid)
		return nil
	}); err != nil {
		t.Fatalf("ScanTable: %v", err)
	}
	if len(rowids) == 0 {
		t.Fatal("expected at least one edge row with rowid > 2")
	}
	for i, id := range rowids {
		if id <= 2 {
			t.Errorf("row %d: rowid %d is not > afterRowID 2", i, id)
		}
		if i > 0 && rowids[i-1] >= id {
			t.Errorf("rowids not strictly ascending: %v", rowids)
		}
	}
}

// TestScanTable_SurfacesErrorOnCorruptedSource proves a truncated/corrupted
// source surfaces as a loud ScanTable error rather than looking like a
// clean, merely-short end-of-rows (07-RESEARCH.md Pitfall 2 — the exact
// silent-I/O class deep review caught in prior phases). Truncating the
// fixture mid-file keeps the valid SQLite header intact (so OpenSource can
// still succeed) while corrupting the b-tree pages a full table scan must
// walk.
func TestScanTable_SurfacesErrorOnCorruptedSource(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read fixture db: %v", err)
	}
	if len(data) < 4096 {
		t.Fatalf("fixture db unexpectedly small (%d bytes) to truncate meaningfully", len(data))
	}
	truncated := filepath.Join(t.TempDir(), "truncated.db")
	if err := os.WriteFile(truncated, data[:len(data)/2], 0o600); err != nil {
		t.Fatalf("write truncated db: %v", err)
	}

	src, err := OpenSource(truncated)
	if err != nil {
		// Corruption detected as early as open/ping — also a valid
		// fail-loud outcome; nothing further to assert.
		return
	}
	defer src.Close()

	scanErr := src.ScanTable("nodes", 0, func(int64, map[string]any) error { return nil })
	if scanErr == nil {
		t.Error("expected ScanTable to fail loud on a truncated/corrupted source, not silently succeed")
	}
}

func TestCountRows(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	n, err := src.CountRows("nodes")
	if err != nil {
		t.Fatalf("CountRows(nodes): %v", err)
	}
	if n <= 0 {
		t.Errorf("expected nodes count > 0, got %d", n)
	}

	if _, err := src.CountRows("nodes_fts"); err == nil {
		t.Error("expected CountRows on a disallowed table to error")
	}
}

func TestCountDistinctEdges(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	src := openSourceT(t, dbPath)

	n, err := src.CountDistinctEdges()
	if err != nil {
		t.Fatalf("CountDistinctEdges: %v", err)
	}
	if n <= 0 {
		t.Errorf("expected distinct edge count > 0, got %d", n)
	}
}

func TestFindDBFile(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "index.db")
		if err := os.WriteFile(dbPath, []byte("x"), 0o600); err != nil {
			t.Fatalf("write fixture db: %v", err)
		}
		got, err := FindDBFile(dir)
		if err != nil {
			t.Fatalf("FindDBFile: %v", err)
		}
		if got != dbPath {
			t.Errorf("FindDBFile = %q, want %q", got, dbPath)
		}
	})

	t.Run("zero", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := FindDBFile(dir); err == nil {
			t.Error("expected FindDBFile to error when no *.db is present")
		}
	})

	t.Run("multiple", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"a.db", "b.db"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
				t.Fatalf("write fixture db %s: %v", name, err)
			}
		}
		if _, err := FindDBFile(dir); err == nil {
			t.Error("expected FindDBFile to error when multiple *.db files are present")
		}
	})
}

// openSourceT opens dbPath via OpenSource and registers a t.Cleanup to
// close it, so individual test bodies don't have to repeat the
// open-check-defer boilerplate.
func openSourceT(t *testing.T, dbPath string) *Source {
	t.Helper()
	src, err := OpenSource(dbPath)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	t.Cleanup(func() {
		if err := src.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return src
}
