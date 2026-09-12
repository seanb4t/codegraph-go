//go:build tmux

package tmux

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// TestDaemonPickerEntersAltScreenAndRestoresMainBuffer is TTY-04: against a
// really-seeded running daemon, the picker enters the alternate screen
// (alternateOn reports true) while a plain converged capture — taken in the
// SAME window, never from a capture-pane -a capture — contains "Running
// daemons" plus the seeded record's own identifying text, so the assertion
// cannot pass against an empty picker. D-13 as corrected by
// 08-RESEARCH.md Pitfall 2: capture-pane -a's own stdout is the frozen MAIN
// buffer while alt-mode is active, not the alt-screen's rendered content —
// only alternateOn's binary signal (or -a's exit code) may be used to
// detect that alt-mode is active. After q, alternateOn reports false and
// the converged scrollback capture carries none of
// daemon_empty_test.go's decrqmResponseMarkers (reused, not re-declared).
func TestDaemonPickerEntersAltScreenAndRestoresMainBuffer(t *testing.T) {
	requireTmux(t)

	home := t.TempDir()
	repo := t.TempDir()

	seedRunningDaemon(t, home, repo)
	waitForRegistryRecord(t, home)

	session := newSession(t)
	sendLiteral(t, session, fmt.Sprintf("env HOME=%s USERPROFILE=%s %s daemon", home, home, binPath))
	sendKey(t, session, "Enter")

	capture := pollUntilStable(t, session, paneContains(t, "Running daemons"))

	if !alternateOn(t, session) {
		t.Fatalf("TTY-04: alternateOn is false while the daemon picker should be open:\n%s", capture)
	}
	if !strings.Contains(capture, "Running daemons") {
		t.Fatalf("TTY-04: converged capture does not contain \"Running daemons\":\n%s", capture)
	}
	// The daemon picker's row renders filepath.Base(record.RepoRoot), not
	// the full path (internal/cli/tui/daemonpicker.go's daemonDelegate.Render)
	// — assert on the basename actually rendered, not the full temp-dir path.
	repoBase := filepath.Base(repo)
	if !strings.Contains(capture, repoBase) {
		t.Fatalf("TTY-04: converged capture does not contain the seeded repo's own identifying text %q — the assertion could pass against an empty picker:\n%s", repoBase, capture)
	}

	sendKey(t, session, "q")
	capture = pollUntilStable(t, session, altScreenOff(t, session))

	if alternateOn(t, session) {
		t.Fatalf("TTY-04: alternateOn is still true after q — the main buffer was not restored:\n%s", capture)
	}
	for _, marker := range decrqmResponseMarkers {
		if strings.Contains(capture, marker) {
			t.Fatalf("TTY-04: converged capture after quit leaks DECRQM mode-query response marker %q:\n%s", marker, capture)
		}
	}
}
