//go:build tmux

package tmux

import (
	"strings"
	"testing"
)

// TestRequireTmuxReportsSkipReasonWhenAbsent is TTY-01's named, always-run
// assertion (D-04): it proves tmuxAvailable's detection half works when
// tmux is genuinely absent from PATH, without itself skipping or spawning
// tmux. It deliberately does NOT call requireTmux(t) and does NOT create a
// session — it asserts only the detection that requireTmux's t.Skipf path
// depends on.
//
// This function passes IDENTICALLY on a machine with tmux and one without,
// because it overrides PATH locally via t.Setenv rather than depending on
// the ambient environment. That is what keeps
// TMUX_EXPECTED_TESTS (Taskfile.yml's test:tmux target) a deterministic
// constant in both environments: a test that skipped whenever tmux IS
// present would make the executed count environment-dependent and defeat
// D-02's exact-equality gate.
func TestRequireTmuxReportsSkipReasonWhenAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := tmuxAvailable()
	if err == nil {
		t.Fatal("tmuxAvailable() = nil error with PATH stripped of tmux, want a non-nil error")
	}
	if !strings.Contains(err.Error(), "tmux") {
		t.Fatalf("tmuxAvailable() error = %q, want it to contain \"tmux\" so requireTmux's t.Skipf reason is self-describing", err.Error())
	}
}
