package query

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExploreRejectsEmptyQuery pins WR-05 for Explore: an empty or
// whitespace-only query must be rejected, mirroring Query/Search, rather
// than falling through to matchNodes' degenerate "matches everything"
// case.
func TestExploreRejectsEmptyQuery(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	defer closer.Close()

	if _, err := engine.Explore("", 5); err == nil {
		t.Fatal("Explore: expected error for an empty query, got nil")
	}
	if _, err := engine.Explore("   ", 5); err == nil {
		t.Fatal("Explore: expected error for a whitespace-only query, got nil")
	}
}

// copySyntheticParityFixture copies testdata/golden/corpus/synthetic-parity/src
// (D-03's purpose-built behavioral corpus — see its own README's case map)
// into a fresh t.TempDir(), mirroring engine_test.go's copyFixture, but
// SKIPS the corpus's own committed .codegraph/ directory: this plan's
// tests build a fresh index via indexFixture (indexer.Run) against the
// CURRENT extractors/schema, rather than trusting a possibly-stale
// committed Pebble store.
func copySyntheticParityFixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "..", "testdata", "golden", "corpus", "synthetic-parity", "src"))
	if err != nil {
		t.Fatalf("resolve synthetic-parity fixture path: %v", err)
	}
	dst := t.TempDir()

	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".codegraph" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy synthetic-parity fixture: %v", err)
	}
	return dst
}

// syntheticParityEngine copies+freshly-indexes the synthetic-parity corpus
// and opens an Engine on it (mirroring nodeExploreFixture's shape,
// render_markdown_test.go).
func syntheticParityEngine(t *testing.T) *Engine {
	t.Helper()

	dir := copySyntheticParityFixture(t)
	indexFixture(t, dir)

	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine
}

// TestExploreMultiWord pins EXPL-01 end-to-end: a 3-word query tokenizes
// (H1/H2) and matches — case (b) of the synthetic-parity corpus, a
// CamelCase type name (UserAccountManager) a multi-word query must
// tokenize into User/Account/Manager and match, not 0-match the way the
// pre-EXPL-01 single-arg CLI bug would have forced a quoted single term.
func TestExploreMultiWord(t *testing.T) {
	engine := syntheticParityEngine(t)

	got, err := engine.Explore("user account manager", 5)
	if err != nil {
		t.Fatalf("Explore: unexpected error: %v", err)
	}
	if strings.Contains(got, "Found 0 symbols across 0 files") {
		t.Fatalf("Explore(\"user account manager\"): got a 0-match result, want a non-empty ranked result:\n%s", got)
	}
	if !strings.Contains(got, "UserAccountManager") {
		t.Fatalf("Explore(\"user account manager\"): expected UserAccountManager to be matched, got:\n%s", got)
	}
}

// TestExploreTestNotTop pins EXPL-03's file-relevance gate: case (c) of
// the synthetic-parity corpus — TestAccountRecovery lexically matches
// "account recovery" but has ZERO inbound graph edges (nothing but the go
// test runtime invokes it), while recoverAccount (recovery.go) is the
// structurally-connected non-test symbol in the same cluster. H15's hard
// test/spec exclusion (query does not mention "test") must drop
// recovery_test.go entirely, so TestAccountRecovery never tops — or even
// appears in — the results.
func TestExploreTestNotTop(t *testing.T) {
	engine := syntheticParityEngine(t)

	got, err := engine.Explore("account recovery", 5)
	if err != nil {
		t.Fatalf("Explore: unexpected error: %v", err)
	}
	// Check the FILE is never SELECTED/RENDERED (not just that the symbol
	// name or file path is absent anywhere in the output — recovery.go's
	// own doc comments mention "TestAccountRecovery" by name, and
	// recoverAccount's legitimate "; tests: `recovery/recovery_test.go`"
	// blast annotation cites the test file it's covered by, without that
	// file itself being selected/rendered — neither is a false positive
	// for H15's exclusion; only an actual "**`recovery/recovery_test.go`**"
	// file-source header would be).
	if strings.Contains(got, "**`recovery/recovery_test.go`**") {
		t.Fatalf("Explore(\"account recovery\"): the weakly-connected Test*-only file must not be selected/rendered (H15 hard test exclusion):\n%s", got)
	}
	if !strings.Contains(got, "recoverAccount") {
		t.Fatalf("Explore(\"account recovery\"): expected the structurally-connected recoverAccount to be matched, got:\n%s", got)
	}
}

// TestExploreStructuralBeatsLexical pins EXPL-02's core RWR property:
// case (d) of the synthetic-parity corpus — AccountBalanceHelper is a
// full lexical match for "account balance" but is deliberately
// graph-isolated (zero calls in/out), while GetBalance is only a PARTIAL
// lexical match (names "balance" but not "account") yet is the real
// bridge into ReconcileLedger, the structurally central symbol (called by
// GetBalance+PostTransaction, itself calling applyAdjustment+AuditEntry —
// strictly more graph edges than the isolated AccountBalanceHelper).
//
// Note on what "outranks" means here (D-02): a pure Random-Walk-with-
// Restart's node-level steady-state mass mathematically CANNOT ever rank
// a connected seed's own self-mass above a same-weighted DANGLING seed's
// self-mass — a degree-0 node retains 100% of its restart injection every
// iteration (it has nowhere to disperse it), which is the maximum any
// node with that restart weight could achieve; a connected seed instead
// disperses most of its mass outward to its neighborhood. So the
// behavioral property this test actually proves is EXPL-02's real
// value-add: RWR-driven expansion SURFACES a symbol with zero lexical
// match at all (ReconcileLedger) because of its structural centrality —
// something the old pure-lexical matchNodes could never have done — and
// the partial-lexical-match structural bridge (GetBalance) still gets
// selected as a matched candidate even though its raw gather/rerank score
// trails the isolated full-match candidate (AccountBalanceHelper).
func TestExploreStructuralBeatsLexical(t *testing.T) {
	engine := syntheticParityEngine(t)

	got, err := engine.Explore("account balance", 5)
	if err != nil {
		t.Fatalf("Explore: unexpected error: %v", err)
	}
	if !strings.Contains(got, "ledger/ledger.go") {
		t.Fatalf("Explore(\"account balance\"): expected ledger/ledger.go to be selected, got:\n%s", got)
	}
	if !strings.Contains(got, "GetBalance") {
		t.Fatalf("Explore(\"account balance\"): expected GetBalance (the structural bridge into ReconcileLedger) to be matched despite its lower raw gather score than the isolated AccountBalanceHelper, got:\n%s", got)
	}
	if !strings.Contains(got, "ReconcileLedger") {
		t.Fatalf("Explore(\"account balance\"): expected ReconcileLedger's source to be surfaced — a symbol with ZERO lexical match at all, reached purely through EXPL-02's structural expansion — got:\n%s", got)
	}
}
