package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestQueryStub covers VERB-03/D-05/D-06/WR-01: the hidden `query` rename
// stub executes nothing, writes nothing directly to stderr, and returns a
// non-nil error whose Error() text carries the whole two-line D-06 message
// — regardless of flags/args, which are ignored entirely.
func TestQueryStub(t *testing.T) {
	dir := setupIndexedFixture(t)

	// wantErrText is the whole D-06 two-line message, joined by a single
	// newline, with no trailing newline and no "codegraph: " prefix — this
	// is now the ENTIRE Error() text (see renamed.go's doc comment): RunE
	// itself writes nothing to stderr, so cmd/codegraph/main.go's single
	// fmt.Fprintln(os.Stderr, err) is the only place these two lines are
	// ever printed, exactly once (WR-01).
	wantErrText := `"query" has been renamed to "search --full" — run: codegraph search --full <term>` + "\n" +
		`the "query" stub is removed in the next minor release (v0.15.0)`

	cases := []struct {
		name string
		args []string
	}{
		{"bare invocation", []string{"query", "main", "-p", dir}},
		{"with --json and args", []string{"query", "--json", "main", "-p", dir}},
		{"with no args at all", []string{"query"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, err := execCmd(tc.args...)
			if err == nil {
				t.Fatalf("the query stub (%s): expected a non-nil error, got nil", tc.name)
			}
			if err.Error() != wantErrText {
				t.Fatalf("the query stub (%s) error = %q, want %q", tc.name, err.Error(), wantErrText)
			}
			if out != "" {
				t.Fatalf("the query stub (%s): expected empty stdout, got %q", tc.name, out)
			}
			if errOut != "" {
				t.Fatalf("the query stub (%s): expected empty direct stderr (the message now travels solely via the returned error), got %q", tc.name, errOut)
			}
		})
	}

	t.Run("Hidden/DisableFlagParsing/no deprecation", func(t *testing.T) {
		root := newRootCmd()
		found, _, err := root.Find([]string{"query"})
		if err != nil {
			t.Fatalf("root.Find([query]): %v", err)
		}
		if !found.Hidden {
			t.Fatal("the query stub: expected Hidden == true")
		}
		if !found.DisableFlagParsing {
			t.Fatal("the query stub: expected DisableFlagParsing == true")
		}
		if found.Deprecated != "" {
			t.Fatalf("the query stub: expected no deprecation string, got %q", found.Deprecated)
		}
	})
}

// TestUnlockStub covers VERB-04/D-05/D-06/WR-01: the hidden `unlock`
// rename stub executes nothing (a stale lockfile it would otherwise clear
// is left untouched), writes nothing directly to stderr, and returns a
// non-nil error whose Error() text carries the whole two-line D-06
// message.
func TestUnlockStub(t *testing.T) {
	dir := t.TempDir()
	codegraphDir := filepath.Join(dir, codegraphDirName)
	if err := os.MkdirAll(codegraphDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	lockPath := filepath.Join(codegraphDir, "daemon.lock")
	data, err := json.Marshal(map[string]any{
		"pid":       deadPID(t),
		"startedAt": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal lock payload: %v", err)
	}
	if err := os.WriteFile(lockPath, data, 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	// wantErrText: see TestQueryStub's identical rationale (WR-01) — the
	// whole D-06 message is now the Error() text, and direct stderr is
	// empty.
	wantErrText := `"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]` + "\n" +
		`the "unlock" stub is removed in the next minor release (v0.15.0)`

	cases := []struct {
		name string
		args []string
	}{
		{"bare invocation", []string{"unlock", dir}},
		{"with --json", []string{"unlock", "--json", dir}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errOut, err := execCmd(tc.args...)
			if err == nil {
				t.Fatalf("the unlock stub (%s): expected a non-nil error, got nil", tc.name)
			}
			if err.Error() != wantErrText {
				t.Fatalf("the unlock stub (%s) error = %q, want %q", tc.name, err.Error(), wantErrText)
			}
			if out != "" {
				t.Fatalf("the unlock stub (%s): expected empty stdout, got %q", tc.name, out)
			}
			if errOut != "" {
				t.Fatalf("the unlock stub (%s): expected empty direct stderr (the message now travels solely via the returned error), got %q", tc.name, errOut)
			}
			if _, statErr := os.Stat(lockPath); statErr != nil {
				t.Fatalf("the unlock stub (%s): expected the lockfile to still exist (nothing executed), stat err: %v", tc.name, statErr)
			}
		})
	}

	t.Run("Hidden/DisableFlagParsing/no deprecation", func(t *testing.T) {
		root := newRootCmd()
		found, _, err := root.Find([]string{"unlock"})
		if err != nil {
			t.Fatalf("root.Find([unlock]): %v", err)
		}
		if !found.Hidden {
			t.Fatal("the unlock stub: expected Hidden == true")
		}
		if !found.DisableFlagParsing {
			t.Fatal("the unlock stub: expected DisableFlagParsing == true")
		}
		if found.Deprecated != "" {
			t.Fatalf("the unlock stub: expected no deprecation string, got %q", found.Deprecated)
		}
	})
}
