package cli

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// TestStatusCmdSections drives the REAL `codegraph status` command — not a
// renderer in isolation — and pins D-09's sectioned plain-text layout,
// replacing the terse `backend=… files=… stale=…` one-liner (02-07-PLAN.md
// Task 2). ★ REACHABILITY IS THE POINT (Phase 1's CR-02 lesson): STAT-01/02/03
// have been live/unit-tested since earlier plans in this phase — this test
// proves they are reachable through the real CLI entry point, not merely
// correct in internal/query.
func TestStatusCmdSections(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("Test 1: DB Size line is a well-formed two-decimal MB value (STAT-01 reachability)", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir)
		if err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
		m := regexp.MustCompile(`DB Size:\s+(\S+ MB)`).FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("status: expected a %q line, got: %q", "DB Size:", out)
		}
		if !regexp.MustCompile(`^\d+\.\d{2} MB$`).MatchString(m[1]) {
			t.Fatalf("status: DB Size value %q does not match ^\\d+\\.\\d{2} MB$", m[1])
		}
	})

	t.Run("Test 2: section headers present (STAT-02 reachability)", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir)
		if err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
		for _, want := range []string{"Index Statistics:", "Nodes by Kind:", "Files by Language:"} {
			if !strings.Contains(out, want) {
				t.Fatalf("status: expected output to contain %q, got: %q", want, out)
			}
		}
	})

	t.Run("Test 3: the live staleness/reindex advisory is reachable through the command (STAT-03 reachability)", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir)
		if err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
		if !strings.Contains(out, "up to date") && !strings.Contains(out, "Pending Changes:") {
			t.Fatalf("status: expected either the up-to-date line or the pending-sync advisory (driven by the live stale signal), got: %q", out)
		}
	})

	t.Run("Test 4: the terse backend=/files=/stale= one-liner is ABSENT (D-09)", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir)
		if err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
		for _, terse := range []string{"backend=", "files=", "stale="} {
			if strings.Contains(out, terse) {
				t.Fatalf("status: expected the terse one-liner token %q to be ABSENT from the sectioned layout, got: %q", terse, out)
			}
		}
	})

	t.Run("Test 5: --json still emits valid JSON containing dbSizeBytes", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("status --json: unexpected error: %v", err)
		}
		var result struct {
			DbSizeBytes int64 `json:"dbSizeBytes"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("status --json: invalid JSON: %v\noutput: %s", err, out)
		}
		if result.DbSizeBytes <= 0 {
			t.Fatalf("status --json: dbSizeBytes = %d, want > 0", result.DbSizeBytes)
		}
	})

	t.Run("Test 6: the verbose worktree warning appears from a borrowed-index worktree (WORK-02)", func(t *testing.T) {
		wt, _ := statusWorktreeMismatchFixture(t)

		out, _, err := execCmd("status", "-p", wt)
		if err != nil {
			t.Fatalf("status: unexpected error: %v", err)
		}
		if !strings.Contains(out, "This CodeGraph index belongs to a different git working tree.") {
			t.Fatalf("status: expected the verbose warning's opening line, got: %q", out)
		}
		if !strings.Contains(out, "Running in:") {
			t.Fatalf("status: expected a %q row, got: %q", "Running in:", out)
		}
		if !strings.Contains(out, "Index from:") {
			t.Fatalf("status: expected an %q row, got: %q", "Index from:", out)
		}
	})
}

// zeroEdgeRowRE matches a rendered breakdown row whose count is literally
// "0" (writeBreakdownText's "  <key padded to 15> <count>\n" shape) — used
// to assert the SPARSE flagless default carries no zero rows.
var zeroEdgeRowRE = regexp.MustCompile(`(?m)^\s+\S+\s+0\s*$`)

// edgesByKindSection extracts the substring between "Edges by Kind:" and
// the next section header ("Files by Language:") from human-text status
// output, failing the test if either marker is missing or out of order.
func edgesByKindSection(t *testing.T, out string) string {
	t.Helper()
	idxEdges := strings.Index(out, "Edges by Kind:")
	idxFiles := strings.Index(out, "Files by Language:")
	if idxEdges < 0 || idxFiles < 0 || idxFiles < idxEdges {
		t.Fatalf("status: expected %q to appear before %q, got: %q", "Edges by Kind:", "Files by Language:", out)
	}
	return out[idxEdges:idxFiles]
}

// TestStatusCmdEdgesByKindSection drives the REAL `codegraph status`
// command with no flag and asserts the Edges by Kind: section lists only
// kinds the index actually contains — no zero rows (D-04's sparse
// default).
func TestStatusCmdEdgesByKindSection(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("status", "-p", dir)
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}

	section := edgesByKindSection(t, out)
	if zeroEdgeRowRE.MatchString(section) {
		t.Errorf("status (flagless): Edges by Kind: section contains a zero-valued row, want only kinds the index actually contains\n--- section ---\n%s", section)
	}
}

// TestStatusCmdAllKindsFlag drives `codegraph status --all-kinds` and
// asserts the Edges by Kind: section lists all nine RankEdges kinds,
// including an explicit "0" row for a kind this fixture's idiomatic Go
// source produces none of (overrides — the same trap FIXT-01 exists to
// catch, per this repo's own v1.0 Phase 1 finding).
func TestStatusCmdAllKindsFlag(t *testing.T) {
	dir := setupIndexedFixture(t)

	out, _, err := execCmd("status", "-p", dir, "--all-kinds")
	if err != nil {
		t.Fatalf("status --all-kinds: unexpected error: %v", err)
	}

	section := edgesByKindSection(t, out)
	for k := range query.RankEdges {
		if !strings.Contains(section, k) {
			t.Errorf("status --all-kinds: Edges by Kind: section missing RankEdges member %q\n--- section ---\n%s", k, section)
		}
	}

	wantOverridesRow := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(goextract.EdgeKindOverrides) + `\s+0\s*$`)
	if !wantOverridesRow.MatchString(section) {
		t.Errorf("status --all-kinds: expected an explicit-zero %q row, got:\n--- section ---\n%s", goextract.EdgeKindOverrides, section)
	}

	// The flag must not alter nodesByKind or filesByLanguage.
	for _, unrelated := range []string{"Nodes by Kind:", "Files by Language:"} {
		if !strings.Contains(out, unrelated) {
			t.Errorf("status --all-kinds: expected unrelated section %q to still be present, got: %q", unrelated, out)
		}
	}
}

// TestStatusCmdJSONDense drives `codegraph status --json` (sparse) and
// `codegraph status --json --all-kinds` (dense) and asserts both shapes.
// The dense case compares the decoded edgesByKind key set against
// query.RankEdges MEMBER BY MEMBER, in both directions — every RankEdges
// member present in the decoded map, and every decoded key either a
// RankEdges member or an unranked kind carrying a positive count. A
// length comparison cannot establish this: dense mode preserves unranked
// kinds, so the map is a superset, and a count of nine or more is
// consistent with a missing ranked kind offset by an unranked one.
func TestStatusCmdJSONDense(t *testing.T) {
	dir := setupIndexedFixture(t)

	t.Run("sparse --json carries no zero-valued entries", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir, "--json")
		if err != nil {
			t.Fatalf("status --json: unexpected error: %v", err)
		}
		var result struct {
			EdgesByKind map[string]int64 `json:"edgesByKind"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("status --json: invalid JSON: %v\noutput: %s", err, out)
		}
		for k, v := range result.EdgesByKind {
			if v <= 0 {
				t.Errorf("status --json (sparse): edgesByKind[%q] = %d, want > 0 (sparse mode must never carry a zero or negative entry)", k, v)
			}
		}
	})

	t.Run("dense --json --all-kinds key set equals RankEdges member by member, both directions", func(t *testing.T) {
		out, _, err := execCmd("status", "-p", dir, "--json", "--all-kinds")
		if err != nil {
			t.Fatalf("status --json --all-kinds: unexpected error: %v", err)
		}
		var result struct {
			EdgesByKind map[string]int64 `json:"edgesByKind"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("status --json --all-kinds: invalid JSON: %v\noutput: %s", err, out)
		}

		for k := range query.RankEdges {
			if _, ok := result.EdgesByKind[k]; !ok {
				t.Errorf("status --json --all-kinds: edgesByKind missing RankEdges member %q", k)
			}
		}
		hasExplicitZero := false
		for k, v := range result.EdgesByKind {
			if !query.RankEdges[k] && v <= 0 {
				t.Errorf("status --json --all-kinds: edgesByKind has unranked key %q with non-positive count %d — every decoded key must be a RankEdges member or an unranked kind carrying a positive count", k, v)
			}
			if v == 0 {
				hasExplicitZero = true
			}
		}
		if !hasExplicitZero {
			t.Errorf("status --json --all-kinds: edgesByKind carries no explicit zero, want at least one — dense mode must carry at least one explicit zero")
		}
	})
}
