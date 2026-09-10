//go:build tmux

package tmux

import (
	"testing"
	"time"
)

// stabilityPollInterval and stabilityPollDeadline bound pollUntilStable
// (D-14): capture on stabilityPollInterval, fail if no two consecutive
// captures match by stabilityPollDeadline.
//
// stabilityPollInterval is deliberately larger than a naive "fast poll"
// choice: measured directly on this machine (three independent traces,
// 100ms sampling), the spawned codegraph process consistently takes
// 700-900ms between the shell echoing the typed command line and its own
// first stdout output appearing — a real subprocess-startup latency, not a
// rendering artifact (reproduced identically whether the typed command is
// long or short). A sub-second interval risks two consecutive samples both
// landing inside that pre-output window, which is byte-identical to itself
// and would make pollUntilStable converge on the wrong (pre-execution)
// stable frame. stabilityPollInterval exceeds the measured worst case with
// margin, so consecutive samples can never both land inside that window by
// construction — the comparison stays a literal two-sample
// immediate-predecessor check (D-14's contract, unmodified); only the
// sampling cadence was tuned to the real timing this machine exhibits.
const (
	stabilityPollInterval = 1 * time.Second
	stabilityPollDeadline = 10 * time.Second
)

// capturePane runs exactly `capture-pane -t <session> -p -e -C -S -` and
// returns stdout. This exact four-flag set is hard-coded here and ONLY
// here in the package (verified by this plan's own repo-wide grep gate),
// each flag load-bearing per D-12:
//
//   - -p prints the pane contents to stdout rather than to a buffer.
//   - -e keeps the escape sequences the default capture strips — without
//     this, a leaked ESC byte is invisible to any assertion.
//   - -C renders non-printable bytes as octal escapes, which is what makes
//     a leaked ESC byte matchable AS TEXT by a plain strings.Contains.
//   - -S - starts capture at the beginning of scrollback history rather
//     than only the visible pane, since some assertions in this package
//     (TTY-04) read content that has scrolled.
//
// Removing -e or -C makes the TTY-03 escape-hygiene assertion pass against
// every capture, fireable or not — the exact vacuity shape this whole
// phase exists to close (rule 84d1gfpywd).
func capturePane(t *testing.T, session string) string {
	t.Helper()
	stdout, stderr, err := runTmux("capture-pane", "-t", session, "-p", "-e", "-C", "-S", "-")
	if err != nil {
		t.Fatalf("tmux capture-pane -t %s -p -e -C -S - failed: %v: %s", session, err, stderr)
	}
	return stdout
}

// alternateOn reports whether session's pane currently has an alternate
// screen active, via `tmux display-message -t <session> -p #{alternate_on}`.
//
// Empirically (08-RESEARCH.md, corrected from D-13's original assumption):
// while the alt-screen is active, a PLAIN capture-pane returns the
// alt-screen's own rendered content, and a `capture-pane -a` returns the
// FROZEN MAIN buffer underneath — both exit 0. Only alternateOn's own
// signal (or -a's exit code) tells you whether alt-mode is active; content
// assertions must always come from the plain capturePane above, never from
// an -a capture.
func alternateOn(t *testing.T, session string) bool {
	t.Helper()
	stdout, stderr, err := runTmux("display-message", "-t", session, "-p", "#{alternate_on}")
	if err != nil {
		t.Fatalf("tmux display-message -t %s -p #{alternate_on} failed: %v: %s", session, err, stderr)
	}
	return trimNewline(stdout) == "1"
}

// trimNewline strips a single trailing "\n" (tmux display-message's output
// convention), without pulling in strings.TrimSpace's broader whitespace
// trimming that could mask a real content difference elsewhere in this
// package's raw-comparison discipline (D-15).
func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}

// pollUntilStable is the package's ONLY wait primitive (D-14): it captures
// session's pane on stabilityPollInterval, comparing each capture to its
// immediate predecessor with plain byte equality and no normalization
// (D-15 — the default capture already trims trailing whitespace, which
// removes the main source of false differences). It returns the first
// capture that is byte-identical to its predecessor.
//
// If stabilityPollDeadline elapses with no two consecutive captures
// matching, pollUntilStable calls t.Fatalf with a message containing both
// of the last two differing captures verbatim. Non-convergence is a
// FAILURE — never a skip, never a whole-case retry, and nothing in this
// package may pause for a fixed duration and then assert without going
// through this poll.
func pollUntilStable(t *testing.T, session string) string {
	t.Helper()

	deadline := time.Now().Add(stabilityPollDeadline)
	prev := capturePane(t, session)
	for {
		if time.Now().After(deadline) {
			cur := capturePane(t, session)
			t.Fatalf("pollUntilStable: capture-pane never converged within %s\n--- previous capture ---\n%s\n--- latest capture ---\n%s",
				stabilityPollDeadline, prev, cur)
		}
		<-time.After(stabilityPollInterval) // interval pacing via a channel wait, no blocking pause call
		cur := capturePane(t, session)
		if cur == prev {
			return cur
		}
		prev = cur
	}
}
