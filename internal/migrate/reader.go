// Package migrate implements the one-way TS CodeGraph SQLite → codegraph-go
// Pebble migration (MIGR-01/MIGR-02). This file is the read half: it opens
// the source TS `.codegraph/*.db` read-only, detects a genuine TS source,
// guards the observed schema version, and streams rows from the
// allow-listed files/nodes/edges tables with defensive column enumeration
// (07-RESEARCH.md Pattern 1) and mandatory rows.Err() checks (07-RESEARCH.md
// Pitfall 2 — mirrors internal/graphstore/export.go's exportNamespace
// "return iter.Error() after the loop" idiom).
package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" driver name (NOT "sqlite3" — that's mattn/go-sqlite3)
)

// Supported source schema_versions range (observed dump range: 1..7,
// 07-RESEARCH.md §Validation D-09.3). Bump maxSupportedSchemaVersion when a
// newer TS schema version is confirmed compatible with this field mapping.
const (
	minSupportedSchemaVersion = 1
	maxSupportedSchemaVersion = 7
)

// ErrNotATSSource is returned by DetectTS when the opened SQLite file is
// missing one or more of the required TS tables (schema_versions, nodes,
// edges) — i.e. it is not a genuine TS CodeGraph index.
var ErrNotATSSource = errors.New("migrate: not a TS CodeGraph source (missing schema_versions/nodes/edges)")

// ErrUnsupportedSchemaVersion is returned by SchemaVersion when the source's
// max(schema_versions.version) falls outside
// [minSupportedSchemaVersion, maxSupportedSchemaVersion].
var ErrUnsupportedSchemaVersion = errors.New("migrate: unsupported source schema version")

// allowedTables is the fixed, code-controlled allow-list of tables ScanTable
// and CountRows may read as data (D-03). Table names are NEVER derived from
// DB contents — only these constants are ever interpolated into a query
// (07-RESEARCH.md Pitfall 3: FTS5 shadow tables and SQLite internals must
// never be enumerated as records).
var allowedTables = map[string]bool{
	"files": true,
	"nodes": true,
	"edges": true,
}

// wantedColumns lists, per allow-listed table, the columns the migrator
// wants to read — in the order translate.go's row-map consumers expect to
// reason about them. ScanTable intersects this list against the columns
// actually present in the opened DB (via presentColumns/PRAGMA table_info)
// so an aged DB missing a later-added column (edges.provenance,
// nodes.return_type) migrates instead of crashing (D-09.4).
var wantedColumns = map[string][]string{
	"files": {"path", "content_hash", "language", "size", "modified_at", "node_count", "errors"},
	"nodes": {
		"id", "kind", "name", "qualified_name", "file_path", "language",
		"start_line", "end_line", "start_column", "end_column",
		"docstring", "signature", "visibility", "is_exported", "return_type",
	},
	"edges": {"source", "target", "kind", "line", "col", "provenance", "metadata"},
}

// Source is a read-only handle onto a TS CodeGraph SQLite index.
type Source struct {
	db *sql.DB
}

// OpenSource opens dbPath read-only via the modernc.org/sqlite driver. The
// DSN sets mode=ro (SQLite URI open mode: read-only, fails if the file
// cannot be opened read-only) and _pragma=query_only(1) (rejects all writes
// at the SQLite layer) — belt-and-suspenders so the source can never be
// mutated and no -wal/-shm sidecar is created (D-08 non-destructive-to-
// source).
func OpenSource(dbPath string) (*Source, error) {
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("migrate: resolve source path %q: %w", dbPath, err)
	}
	dsn := "file:" + abs + "?mode=ro&_pragma=query_only(1)&_txlock=deferred"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("migrate: open source %q: %w", dbPath, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: open source %q: %w", dbPath, err)
	}
	return &Source{db: db}, nil
}

// Close releases the underlying database handle.
func (s *Source) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("migrate: close source: %w", err)
	}
	return nil
}

// DetectTS probes sqlite_master for the three tables that make a SQLite
// file a genuine TS CodeGraph source. Returns ErrNotATSSource if any are
// absent.
func (s *Source) DetectTS() error {
	required := []string{"schema_versions", "nodes", "edges"}
	for _, table := range required {
		var name string
		err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrNotATSSource
		case err != nil:
			return fmt.Errorf("migrate: detect ts source: %w", err)
		}
	}
	return nil
}

// SchemaVersion returns max(schema_versions.version), the observed source
// schema version. It returns ErrUnsupportedSchemaVersion (wrapping the
// observed value) when that version falls outside
// [minSupportedSchemaVersion, maxSupportedSchemaVersion], or when the table
// has no rows.
func (s *Source) SchemaVersion() (int, error) {
	var version sql.NullInt64
	if err := s.db.QueryRow(`SELECT max(version) FROM schema_versions`).Scan(&version); err != nil {
		return 0, fmt.Errorf("migrate: read schema version: %w", err)
	}
	if !version.Valid {
		return 0, fmt.Errorf("%w: schema_versions has no rows", ErrUnsupportedSchemaVersion)
	}
	v := int(version.Int64)
	if v < minSupportedSchemaVersion || v > maxSupportedSchemaVersion {
		return 0, fmt.Errorf("%w: %d (supported range [%d,%d])", ErrUnsupportedSchemaVersion, v, minSupportedSchemaVersion, maxSupportedSchemaVersion)
	}
	return v, nil
}

// presentColumns queries PRAGMA table_info(table) to learn which columns
// actually exist in this (possibly aged) DB. table MUST be one of
// allowedTables's fixed constants — never DB-derived — since PRAGMA
// table_info is not parameterizable and is therefore string-interpolated
// (07-RESEARCH.md Pattern 1).
func (s *Source) presentColumns(table string) (map[string]bool, error) {
	if !allowedTables[table] {
		return nil, fmt.Errorf("migrate: table %q is not in the allow-list", table)
	}
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, fmt.Errorf("migrate: table_info(%s): %w", table, err)
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid         int
			name, ctype string
			notnull, pk int
			dflt        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("migrate: scan table_info(%s): %w", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: iterate table_info(%s): %w", table, err)
	}
	return cols, nil
}

// ScanTable streams rows from an allow-listed table (files/nodes/edges),
// ordered ascending by rowid, restricted to rowid > afterRowID (the resume
// cursor — D-06). Each row is scanned into a map[string]any keyed by
// column name, built from the intersection of wantedColumns[table] and the
// columns actually present (defensive read, D-09.4), then passed to fn.
// ScanTable returns rows.Err() after the loop — the mandatory fail-loud
// check mirroring internal/graphstore/export.go's exportNamespace
// "return iter.Error()" idiom (07-RESEARCH.md Pitfall 2).
func (s *Source) ScanTable(table string, afterRowID int64, fn func(rowid int64, row map[string]any) error) error {
	if !allowedTables[table] {
		return fmt.Errorf("migrate: table %q is not in the allow-list", table)
	}
	present, err := s.presentColumns(table)
	if err != nil {
		return err
	}
	wanted := wantedColumns[table]
	cols := make([]string, 0, len(wanted))
	for _, c := range wanted {
		if present[c] {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return fmt.Errorf("migrate: table %q has none of the wanted columns present", table)
	}

	query := fmt.Sprintf("SELECT rowid, %s FROM %s WHERE rowid > ? ORDER BY rowid", strings.Join(cols, ", "), table)
	rows, err := s.db.Query(query, afterRowID)
	if err != nil {
		return fmt.Errorf("migrate: scan %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		dest := make([]any, len(cols)+1)
		ptrs := make([]any, len(dest))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("migrate: scan row in %s: %w", table, err)
		}
		rowid, ok := dest[0].(int64)
		if !ok {
			return fmt.Errorf("migrate: unexpected rowid type %T in %s", dest[0], table)
		}
		row := make(map[string]any, len(cols))
		for i, c := range cols {
			row[c] = dest[i+1]
		}
		if err := fn(rowid, row); err != nil {
			return err
		}
	}
	return rows.Err()
}

// CountRows returns SELECT count(*) FROM table for an allow-listed table.
func (s *Source) CountRows(table string) (int64, error) {
	if !allowedTables[table] {
		return 0, fmt.Errorf("migrate: table %q is not in the allow-list", table)
	}
	var n int64
	if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		return 0, fmt.Errorf("migrate: count %s: %w", table, err)
	}
	return n, nil
}

// CountDistinctEdges returns the number of distinct (source, kind, target)
// tuples in edges — the count reconciliation (D-09.1) must compare against
// this, not raw row count, because two TS edges sharing (source,kind,target)
// but differing only in (line,col) collapse to one stored edge in the new
// format's key scheme (07-RESEARCH.md §Validation Invariants).
func (s *Source) CountDistinctEdges() (int64, error) {
	var n int64
	const query = `SELECT count(*) FROM (SELECT DISTINCT source, kind, target FROM edges)`
	if err := s.db.QueryRow(query).Scan(&n); err != nil {
		return 0, fmt.Errorf("migrate: count distinct edges: %w", err)
	}
	return n, nil
}

// FindDBFile autodetects the single *.db file inside a TS .codegraph/
// directory. Returns an error if zero or more than one is found.
func FindDBFile(codegraphDir string) (string, error) {
	entries, err := os.ReadDir(codegraphDir)
	if err != nil {
		return "", fmt.Errorf("migrate: read %s: %w", codegraphDir, err)
	}
	var found []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".db") {
			found = append(found, e.Name())
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("migrate: no *.db file found in %s", codegraphDir)
	case 1:
		return filepath.Join(codegraphDir, found[0]), nil
	default:
		return "", fmt.Errorf("migrate: multiple *.db files found in %s (ambiguous): %v", codegraphDir, found)
	}
}
