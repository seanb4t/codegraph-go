package cli

import (
	"strings"
	"testing"
)

// TestDaemonCmdPolicyDisabledExitsCleanly pins `codegraph daemon`'s
// policy-disabled branch (03-REVIEW.md IN-06 — the only shipped consumer of
// round-1 WR-01's fix, previously exercised by nothing): with
// CODEGRAPH_NO_WATCH=1 exported, the command must exit cleanly (nil error —
// a policy-disabled watcher is a deliberate, explained state, not a
// failure) and print the D-12 guidance with the verbatim reason to stderr.
// The reason travels inside daemon.Run's typed watch.DisabledError (IN-05),
// so this also pins the errors.As extraction path.
//
// The env-driven disable keeps the test hermetic (no flags beyond --path),
// and Run's policy gate returns before any lockfile/watcher/store work, so
// the test is instant once the fixture is indexed. Uses the package's
// in-process cobra execution pattern (execCmd, cli_test.go) — no
// subprocess machinery.
func TestDaemonCmdPolicyDisabledExitsCleanly(t *testing.T) {
	dir := copyFixture(t)
	if _, _, err := execCmd("init", "--quiet", dir); err != nil {
		t.Fatalf("init fixture: %v", err)
	}

	t.Setenv("CODEGRAPH_NO_WATCH", "1")

	_, stderr, err := execCmd("daemon", "--path", dir)
	if err != nil {
		t.Fatalf("daemon with CODEGRAPH_NO_WATCH=1: want nil error (clean exit for a policy-disabled watcher), got: %v", err)
	}
	if want := "File watcher disabled — CODEGRAPH_NO_WATCH=1 is set"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q does not contain the verbatim disabled message %q", stderr, want)
	}
	if want := "run `codegraph sync`"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr %q does not contain the %s guidance", stderr, want)
	}
	// IN-05: the standalone daemon command deliberately drops serve's
	// "[CodeGraph MCP]" banner — it is not the MCP server.
	if strings.Contains(stderr, "[CodeGraph MCP]") {
		t.Fatalf("stderr %q carries the [CodeGraph MCP] banner; the standalone daemon command must not", stderr)
	}
}
