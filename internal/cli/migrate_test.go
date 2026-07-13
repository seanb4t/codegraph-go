package cli

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/migrate"
	"github.com/seanb4t/codegraph-go/internal/migrate/migratetest"
)

func TestMigrateCmdRegisteredAndFlags(t *testing.T) {
	root := newRootCmd()
	cmd, _, err := root.Find([]string{"migrate"})
	if err != nil {
		t.Fatalf("migrate command not found: %v", err)
	}
	for _, name := range []string{"from", "to", "force", "drop-dangling"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("expected --%s flag registered", name)
		}
	}
}

func TestMigrateCmdEndToEnd(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	fromDir := filepath.Dir(dbPath) // contains a single *.db — FindDBFile autodetect
	toDir := filepath.Join(t.TempDir(), codegraphDirName)

	out, _, err := execCmd("migrate", "--from", fromDir, "--to", toDir)
	if err != nil {
		t.Fatalf("migrate: unexpected error: %v", err)
	}
	if !strings.Contains(out, "migrated:") {
		t.Fatalf("expected report output to mention 'migrated:', got %q", out)
	}
	if !strings.Contains(out, "full re-index") {
		t.Fatalf("expected D-01 first-sync full-reindex note in output, got %q", out)
	}

	store, err := graphstore.Open(filepath.Join(toDir, storeDirName))
	if err != nil {
		t.Fatalf("open migrated store: %v", err)
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	if !meta.GetHealthy() {
		t.Fatalf("expected migrated store to be healthy, got Meta=%+v", meta)
	}
	if !meta.GetHasFileIndex() {
		t.Fatalf("expected HasFileIndex=true on the migrated store")
	}
}

func TestMigrateCmdFailsLoudOnNonTSSource(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "not-ts.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE foo (id INTEGER)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	toDir := filepath.Join(t.TempDir(), "out")
	_, _, err = execCmd("migrate", "--from", dir, "--to", toDir)
	if !errors.Is(err, migrate.ErrNotATSSource) {
		t.Fatalf("expected ErrNotATSSource, got: %v", err)
	}
	if _, statErr := os.Stat(toDir); !os.IsNotExist(statErr) {
		t.Fatalf("target should not have been created on a failed migration, stat err: %v", statErr)
	}
}

func TestMigrateCmdRefusesUnrecognizedTargetWithoutForce(t *testing.T) {
	dbPath := migratetest.BuildTSIndex(t, migratetest.VariantHappy)
	fromDir := filepath.Dir(dbPath)

	toDir := t.TempDir()
	existing := filepath.Join(toDir, "existing.txt")
	if err := os.WriteFile(existing, []byte("not a codegraph store"), 0o644); err != nil {
		t.Fatalf("seed non-empty target: %v", err)
	}

	out, _, err := execCmdWithInput("n\n", "migrate", "--from", fromDir, "--to", toDir)
	if err != nil {
		t.Fatalf("migrate (declined): unexpected error: %v", err)
	}
	if !strings.Contains(out, "aborted") {
		t.Fatalf("expected aborted message, got %q", out)
	}
	if _, statErr := os.Stat(existing); statErr != nil {
		t.Fatalf("existing target content should remain untouched: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(toDir, "store")); !os.IsNotExist(statErr) {
		t.Fatalf("no store should have been written on refusal, stat err: %v", statErr)
	}
}
