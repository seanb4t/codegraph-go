package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestQueryStub covers VERB-03/D-05/D-06: the hidden `query` rename stub
// executes nothing, writes only to stderr, and returns a non-nil error —
// regardless of flags/args, which are ignored entirely.
func TestQueryStub(t *testing.T) {
	dir := setupIndexedFixture(t)

	wantErrSubstring := `renamed to "search --full"`
	wantStderr := "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\n" +
		"the \"query\" stub is removed in the next minor release (v0.15.0)\n"

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
			if !strings.Contains(err.Error(), wantErrSubstring) {
				t.Fatalf("the query stub (%s) error = %q, want it to contain %q", tc.name, err.Error(), wantErrSubstring)
			}
			if out != "" {
				t.Fatalf("the query stub (%s): expected empty stdout, got %q", tc.name, out)
			}
			if errOut != wantStderr {
				t.Fatalf("the query stub (%s) stderr = %q, want %q", tc.name, errOut, wantStderr)
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

// TestUnlockStub covers VERB-04/D-05/D-06: the hidden `unlock` rename stub
// executes nothing (a stale lockfile it would otherwise clear is left
// untouched), writes only to stderr, and returns a non-nil error.
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

	wantErrSubstring := `renamed to "daemon unlock"`
	wantStderr := "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\n" +
		"the \"unlock\" stub is removed in the next minor release (v0.15.0)\n"

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
			if !strings.Contains(err.Error(), wantErrSubstring) {
				t.Fatalf("the unlock stub (%s) error = %q, want it to contain %q", tc.name, err.Error(), wantErrSubstring)
			}
			if out != "" {
				t.Fatalf("the unlock stub (%s): expected empty stdout, got %q", tc.name, out)
			}
			if errOut != wantStderr {
				t.Fatalf("the unlock stub (%s) stderr = %q, want %q", tc.name, errOut, wantStderr)
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

