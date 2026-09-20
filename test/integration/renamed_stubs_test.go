package integration

import (
	"os/exec"
	"testing"
)

// TestRenamedStubsPrintExactlyOnce is WR-01's end-to-end proof: the
// real, spawned binary — not root.Execute() via execCmd, which
// internal/cli/renamed_test.go already covers, but the actual
// cmd/codegraph/main.go error-printing seam D-19 exists to exercise —
// prints the D-06 two-line rename message to stderr EXACTLY ONCE (not
// duplicated by main.go's fmt.Fprintln(os.Stderr, err)), writes nothing
// to stdout, and exits 1.
//
// Before the WR-01 fix, each stub's RunE both wrote the two D-06 lines to
// cmd.ErrOrStderr() itself AND returned a non-nil error whose own text
// main.go printed as a third line — three lines total, with the third
// partially duplicating the first. The fix (internal/cli/renamed.go)
// makes the returned error's Error() text the WHOLE two-line message and
// has RunE write nothing directly, so main.go's single print is the only
// one and the total is exactly two lines plus its own trailing newline.
func TestRenamedStubsPrintExactlyOnce(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	const queryWant = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\n" +
		"the \"query\" stub is removed in the next minor release (v0.15.0)\n"
	const unlockWant = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\n" +
		"the \"unlock\" stub is removed in the next minor release (v0.15.0)\n"

	cases := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{"query bare", []string{"query", "main"}, queryWant},
		{"query with --json flag", []string{"query", "--json", "main"}, queryWant},
		{"unlock bare", []string{"unlock", "-p", dir}, unlockWant},
		{"unlock with nonexistent path arg", []string{"unlock", "-p", "/nonexistent"}, unlockWant},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runBinary(t, dir, nil, tc.args...)

			if stdout != "" {
				t.Fatalf("%v: expected empty stdout, got %q", tc.args, stdout)
			}
			if stderr != tc.wantStderr {
				t.Fatalf("%v: stderr = %q, want exactly %q (the message printed exactly once by main.go, WR-01)", tc.args, stderr, tc.wantStderr)
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
}
