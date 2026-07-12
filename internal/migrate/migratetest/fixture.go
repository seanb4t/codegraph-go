// Package migratetest provides an in-Go SQLite fixture-reconstruction
// harness for internal/migrate tests. It materializes a real, readable
// SQLite .db from the committed TS DDL (testdata/golden/ts-schema.sql) and
// representative dump (testdata/golden/ts-schema.dump.sql) — no external
// sqlite3 binary, no live TS CodeGraph CLI. See RESEARCH.md §"The
// Real-Aged-.codegraph/ Fixture Question": no committed aged .db fixture
// exists, only the DDL + a determinism-stripped dump, so every downstream
// migrate test depends on this harness for something real to run against.
package migratetest

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "modernc.org/sqlite" // registers the "sqlite" driver name
)

// Variant selects which shape of TS-index fixture BuildTSIndex produces.
type Variant int

const (
	// VariantHappy is the full DDL + full seed rows: exercises the happy
	// path (file:-source contains edges, unistr docstrings, nullable
	// columns).
	VariantHappy Variant = iota
	// VariantAged drops later-added columns (nodes.return_type,
	// edges.provenance) after the happy build, so the resulting DB
	// predates those columns — proves D-09.4 defensive reads downstream.
	VariantAged
	// VariantDangling adds one extra edge whose target id is present in
	// no nodes row — the D-09.2 corruption fixture.
	VariantDangling
)

// fixedEpochMS replaces every <EPOCH_MS> token in the committed dump. The
// dump was determinism-stripped at capture time; any fixed valid
// millisecond timestamp is acceptable here — the exact value carries no
// semantic meaning for the migrator.
const fixedEpochMS = "1700000000000"

// BuildTSIndex reconstructs a real, readable TS-shaped SQLite .db in
// t.TempDir() from the committed testdata/golden/ts-schema.sql (DDL) and
// ts-schema.dump.sql (seed rows), entirely in-Go via the modernc.org/sqlite
// driver. It returns the path to the built .db file. t.Cleanup closes the
// underlying handle.
func BuildTSIndex(t testing.TB, v Variant) string {
	t.Helper()

	root := repoRoot(t)
	ddl := readFixtureFile(t, filepath.Join(root, "testdata", "golden", "ts-schema.sql"))
	dump := readFixtureFile(t, filepath.Join(root, "testdata", "golden", "ts-schema.dump.sql"))

	dbPath := filepath.Join(t.TempDir(), "ts-index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("migratetest: open db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("migratetest: close db: %v", err)
		}
	})

	if _, err := db.Exec(sanitizeDDL(ddl)); err != nil {
		t.Fatalf("migratetest: exec ddl: %v", err)
	}

	seed := strings.ReplaceAll(dump, "<EPOCH_MS>", fixedEpochMS)
	// unistr('...') docstrings in the dump are evaluated by SQLite itself
	// during this Exec — the migrator never sees the wrapper, only the
	// already-decoded column value (RESEARCH §"unistr(...) docstrings").
	if _, err := db.Exec(seed); err != nil {
		t.Fatalf("migratetest: exec seed: %v", err)
	}
	closeReferentialGaps(t, db)

	switch v {
	case VariantHappy:
		// Nothing further — the seeded happy-path DB is the fixture.
	case VariantAged:
		ageIndex(t, db)
	case VariantDangling:
		addDanglingEdge(t, db)
	default:
		t.Fatalf("migratetest: unknown variant %d", v)
	}

	return dbPath
}

// sqliteReservedTablePrefixes lists the exact CREATE TABLE line prefixes
// ts-schema.sql carries for tables SQLite creates and manages implicitly:
// sqlite_sequence (any AUTOINCREMENT column), sqlite_stat1 (ANALYZE), and
// the FTS5 shadow tables SQLite creates itself alongside the nodes_fts
// virtual table declaration (nodes_fts_data/idx/docsize/config).
// ts-schema.sql was captured via `.schema`, which dumps these
// auto-generated tables' definitions verbatim; replaying them as explicit
// CREATE TABLE statements conflicts with SQLite creating them itself
// (sqlite_* names are reserved; the virtual table statement already
// creates its own shadow tables). This is pure fixture-reconstruction
// plumbing, not a migration-code concern — the migrator never touches
// these tables either (D-03 drop list; Pitfall 3).
var sqliteReservedTablePrefixes = []string{
	"CREATE TABLE sqlite_sequence(",
	"CREATE TABLE sqlite_stat1(",
	"CREATE TABLE 'nodes_fts_data'(",
	"CREATE TABLE 'nodes_fts_idx'(",
	"CREATE TABLE 'nodes_fts_docsize'(",
	"CREATE TABLE 'nodes_fts_config'(",
}

// sanitizeDDL strips the reserved-table CREATE statements described above.
// Each one is captured as a single physical line in ts-schema.sql, so a
// line-based filter is sufficient and avoids the fragility of a multi-line
// regex over statements containing their own internal parentheses.
func sanitizeDDL(ddl string) string {
	lines := strings.Split(ddl, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isReservedTableLine(trimmed) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func isReservedTableLine(line string) bool {
	for _, prefix := range sqliteReservedTablePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// fixedEpochMSValue is fixedEpochMS as a bound-parameter int64 for
// synthetic INSERTs (closeReferentialGaps), rather than a string-substituted
// literal in the Exec text.
const fixedEpochMSValue int64 = 1700000000000

// closeReferentialGaps inserts minimal synthetic node rows for any edge
// target id the seeded dump references but does not itself include as a
// node row. ts-schema.dump.sql is a deliberately small, representative
// excerpt: it seeds 5 nodes plus 5 file:-source "contains" edges whose
// targets are import: node ids the excerpt does not include. Left
// unaddressed, the happy-path fixture would already contain "dangling"
// edges before VariantDangling ever adds its own, making a dangling-edge
// count meaningless as a distinguishing signal between variants. This is
// fixture-reconstruction plumbing to make the representative dump
// referentially self-consistent — not a statement about what the real
// migrator should assume about arbitrary TS indexes.
func closeReferentialGaps(t testing.TB, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT e.target FROM edges e WHERE NOT EXISTS (SELECT 1 FROM nodes n WHERE n.id = e.target)`)
	if err != nil {
		t.Fatalf("migratetest: query referential gaps: %v", err)
	}
	var missing []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatalf("migratetest: scan referential gap: %v", err)
		}
		missing = append(missing, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatalf("migratetest: iterate referential gaps: %v", err)
	}
	rows.Close()

	const insert = `INSERT INTO nodes (id, kind, name, qualified_name, file_path, language, start_line, end_line, start_column, end_column, is_exported, is_async, is_static, is_abstract, updated_at) VALUES (?, ?, ?, ?, 'synthetic', 'go', 0, 0, 0, 0, 0, 0, 0, 0, ?)`
	for _, id := range missing {
		kind, _, _ := strings.Cut(id, ":")
		if _, err := db.Exec(insert, id, kind, id, id, fixedEpochMSValue); err != nil {
			t.Fatalf("migratetest: insert synthetic node for %s: %v", id, err)
		}
	}
}

// ageIndex drops the later-added nodes.return_type and edges.provenance
// columns (both present in the current ts-schema.sql DDL with DEFAULT
// clauses per RESEARCH's aged-index-reality note) so the DB looks like a
// genuinely aged TS index that predates them. The provenance column's
// index must be dropped first — SQLite refuses ALTER TABLE ... DROP COLUMN
// on a column an index still references.
func ageIndex(t testing.TB, db *sql.DB) {
	t.Helper()
	stmts := []string{
		"ALTER TABLE nodes DROP COLUMN return_type",
		"DROP INDEX IF EXISTS idx_edges_provenance",
		"ALTER TABLE edges DROP COLUMN provenance",
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("migratetest: age index: exec %q: %v", stmt, err)
		}
	}
}

// danglingEdgeSource is a node id present in the seeded happy-path dump
// (the "EpicKey" constant) — used as the dangling edge's source so only
// its target is a corruption signal, not the source too.
const danglingEdgeSource = "constant:01228593622a5678b0879f06c50d971c"

// danglingEdgeTarget deliberately does not match any seeded node id.
const danglingEdgeTarget = "class:dangling0000000000000000000000"

// addDanglingEdge inserts one edge whose target resolves to no node —
// the D-09.2 corruption fixture. SQLite's FK enforcement defaults to off
// (PRAGMA foreign_keys is not enabled by this harness), so the insert
// succeeds despite the edges table's FOREIGN KEY (target) REFERENCES
// nodes(id) declaration.
func addDanglingEdge(t testing.TB, db *sql.DB) {
	t.Helper()
	const stmt = `INSERT INTO edges(source, target, kind) VALUES (?, ?, 'calls')`
	if _, err := db.Exec(stmt, danglingEdgeSource, danglingEdgeTarget); err != nil {
		t.Fatalf("migratetest: add dangling edge: %v", err)
	}
}

// repoRoot locates the module root by walking up from this file's own
// location (via runtime.Caller) until a go.mod is found. This lets
// BuildTSIndex resolve testdata/golden/* regardless of the calling test's
// own working directory.
func repoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("migratetest: runtime.Caller failed to report the calling file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("migratetest: could not locate go.mod above %s", filepath.Dir(file))
		}
		dir = parent
	}
}

func readFixtureFile(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("migratetest: read fixture %s: %v", path, err)
	}
	return string(data)
}
