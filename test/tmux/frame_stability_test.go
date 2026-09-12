//go:build tmux

package tmux

import (
	"fmt"
	"testing"
	"time"
)

// frameStabilityCaptures is the number of further captures
// TestInstallPickerFrameStableWhileIdle takes after the picker's first
// settled frame, all of which must be byte-identical to that first frame.
const frameStabilityCaptures = 5

// frameStabilityRows is deliberately shorter than the package's default
// sessionHeight (30) so the agent list plus its help footer cannot fit the
// pane. That is not incidental: the alternate screen exists precisely
// because a full-height list that overflows the remaining space scrolls
// the main buffer every frame (internal/cli/tui/daemonpicker.go's View()
// doc comment) — a picker rendering inline in a 12-row pane genuinely
// flickers while one in a 30-row pane might not. Choosing the short pane
// is what makes this assertion non-vacuous and what makes plan 08-04's
// family (d) mutation (v.AltScreen = false) observable rather than a coin
// flip.
const frameStabilityRows = 12

// TestInstallPickerFrameStableWhileIdle is TTY-06: an idle checkbox
// picker, receiving no input, holds byte-identical across
// frameStabilityCaptures consecutive captures taken after its first
// settled frame — the flicker proxy. Comparison is raw, with no
// normalization and no masked regions (D-15). The test logs a line
// beginning exactly "TTY-06: frame-stable across N=" carrying the value
// frameStabilityCaptures — go test -json carries t.Logf output as
// Action=="output" regardless of -v, which is what lets task test:tmux
// echo the reported N into the CI job log (the literal prefix Taskfile.yml
// filters on).
func TestInstallPickerFrameStableWhileIdle(t *testing.T) {
	requireTmux(t)

	home := t.TempDir()
	session := newSessionSized(t, sessionWidth, frameStabilityRows)

	sendLiteral(t, session, fmt.Sprintf("env HOME=%s USERPROFILE=%s %s install", home, home, binPath))
	sendKey(t, session, "Enter")

	// Converge once to reach a settled first frame — this proxy measures
	// stability of an already-settled picker, not the transient during
	// startup.
	settled := pollUntilStable(t, session, paneContains(t, "[ ]"))

	for i := 1; i <= frameStabilityCaptures; i++ {
		<-time.After(stabilityPollInterval)
		cur := capturePane(t, session)
		if cur != settled {
			t.Fatalf("TTY-06: capture %d/%d diverged from the settled frame while idle:\n--- settled frame ---\n%s\n--- capture %d ---\n%s",
				i, frameStabilityCaptures, settled, i, cur)
		}
	}

	t.Logf("TTY-06: frame-stable across N=%d captures (pane %dx%d, install picker idle)", frameStabilityCaptures, sessionWidth, frameStabilityRows)

	sendKey(t, session, "q")
}
