package integration

import (
	"fmt"
	"os/exec"
	"testing"
)

// TestRemovedVerbsAreUnknownCommands is VERB-09's end-to-end proof (D-01,
// D-02): the real, spawned binary — not root.Execute() via execCmd, but the
// actual cmd/codegraph/main.go error-printing seam — answers both removed
// verbs, `query` and `unlock`, with cobra's own unknown-command error and
// exits 1. Neither name may have any afterlife: no alias, no SuggestFor
// entry, no hidden command and no hint text naming a replacement may revive
// either one (D-01). This is the RED-first test for the removal (rule
// x1cjy9vyhq): it is committed while the stubs still exist and must fail on
// the stderr assertion against their still-registered rename text, then
// pass once the stubs, their registration and their allowlist entries are
// gone.
//
// Rule 84d1gfpywd — a guard must carry a positive assertion that it did its
// work — is enforced two ways: len(cases) is checked against the exact
// literal 4 before the loop runs, and a counter incremented at the start of
// each subtest is logged and checked against that same literal after the
// loop. A filtered `-run` naming only one subtest therefore fails the
// executed-count check by design: that is the guard doing its job, not a
// false alarm.
func TestRemovedVerbsAreUnknownCommands(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	cases := []struct {
		name string
		args []string
		verb string
	}{
		{"query bare", []string{"query", "main"}, "query"},
		{"query with --json flag", []string{"query", "--json", "main"}, "query"},
		{"unlock bare", []string{"unlock", "-p", dir}, "unlock"},
		{"unlock with nonexistent path arg", []string{"unlock", "-p", "/nonexistent"}, "unlock"},
	}

	if len(cases) != 4 {
		t.Fatalf("test table has %d cases, want 4", len(cases))
	}

	executed := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			executed++
			wantStderr := fmt.Sprintf("unknown command %q for %q\n", tc.verb, "codegraph")

			stdout, stderr, err := runBinary(t, dir, nil, tc.args...)

			if stdout != "" {
				t.Fatalf("%v: expected empty stdout, got %q", tc.args, stdout)
			}
			if stderr != wantStderr {
				t.Fatalf("%v: stderr = %q, want exactly %q (cobra's own unknown-command line, VERB-09/D-01)", tc.args, stderr, wantStderr)
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%v: err = %v (%T), want *exec.ExitError", tc.args, err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Fatalf("%v: exit code = %d, want 1", tc.args, exitErr.ExitCode())
			}
		})
	}

	t.Logf("removed-verb guard: executed %d of %d cases", executed, len(cases))
	if executed != 4 {
		t.Fatalf("removed-verb guard: executed %d of %d cases, want 4", executed, len(cases))
	}
}
