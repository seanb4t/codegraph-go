package migratetest

import (
	"database/sql"
	"testing"
)

// TestBuildTSIndex_Variants builds each of the three fixture variants and
// asserts the distinguishing property each one exists to prove.
func TestBuildTSIndex_Variants(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		dbPath := BuildTSIndex(t, VariantHappy)
		db := openReadOnly(t, dbPath)

		assertCountPositive(t, db, "nodes")
		assertCountPositive(t, db, "edges")
		assertCountPositive(t, db, "files")

		var fileSourceEdges int
		const q = `SELECT count(*) FROM edges WHERE source LIKE 'file:%'`
		if err := db.QueryRow(q).Scan(&fileSourceEdges); err != nil {
			t.Fatalf("migratetest: query file-source edges: %v", err)
		}
		if fileSourceEdges == 0 {
			t.Error("migratetest: expected at least one file:-prefixed edge source in the happy fixture")
		}
	})

	t.Run("aged", func(t *testing.T) {
		dbPath := BuildTSIndex(t, VariantAged)
		db := openReadOnly(t, dbPath)

		if cols := presentColumns(t, db, "nodes"); cols["return_type"] {
			t.Error("migratetest: aged fixture should not have nodes.return_type")
		}
		if cols := presentColumns(t, db, "edges"); cols["provenance"] {
			t.Error("migratetest: aged fixture should not have edges.provenance")
		}
	})

	t.Run("dangling", func(t *testing.T) {
		dbPath := BuildTSIndex(t, VariantDangling)
		db := openReadOnly(t, dbPath)

		var danglingCount int
		const q = `SELECT count(*) FROM edges e WHERE NOT EXISTS (SELECT 1 FROM nodes n WHERE n.id = e.target)`
		if err := db.QueryRow(q).Scan(&danglingCount); err != nil {
			t.Fatalf("migratetest: query dangling edges: %v", err)
		}
		if danglingCount != 1 {
			t.Errorf("migratetest: expected exactly 1 dangling edge, got %d", danglingCount)
		}
	})
}

func openReadOnly(t testing.TB, dbPath string) *sql.DB {
	t.Helper()
	dsn := "file:" + dbPath + "?mode=ro&_pragma=query_only(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("migratetest: open read-only: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("migratetest: close read-only db: %v", err)
		}
	})
	return db
}

// assertCountPositive asserts a row-count is > 0. table is always a
// code-controlled literal from this test file (never external input), so
// string concatenation into the query is safe.
func assertCountPositive(t testing.TB, db *sql.DB, table string) {
	t.Helper()
	var cnt int
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&cnt); err != nil {
		t.Fatalf("migratetest: count %s: %v", table, err)
	}
	if cnt <= 0 {
		t.Errorf("migratetest: expected %s count > 0, got %d", table, cnt)
	}
}

// presentColumns returns the set of column names PRAGMA table_info reports
// for table. table is always a code-controlled literal (never external
// input) — PRAGMA table_info is not parameterizable via placeholders.
func presentColumns(t testing.TB, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("migratetest: table_info(%s): %v", table, err)
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("migratetest: scan table_info(%s): %v", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("migratetest: iterate table_info(%s): %v", table, err)
	}
	return cols
}
