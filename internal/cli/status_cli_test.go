package cli

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
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
