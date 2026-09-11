//go:build tmux

package tmux

import (
	"strings"
	"testing"
	"time"
)

// stabilityPollInterval and stabilityPollDeadline bound pollUntilStable
// (D-14, amended by plan 08-05 to close G-08-1): capture on
// stabilityPollInterval, converge only once the caller's readiness
// predicate holds for a capture that is ALSO byte-identical to its
// predecessor, and fail if neither condition is jointly satisfied by
// stabilityPollDeadline.
//
// stabilityPollInterval is the sampling cadence ONLY: it bounds how
// finely the poll observes the pane and how quickly it can return once a
// frame has both settled and satisfied readiness. It is NOT what prevents
// convergence on a pre-output frame — the readiness predicate is. A
// pre-output frame (shell prompt plus echoed command, before the spawned
// process has written anything) is byte-identical to itself and therefore
// "stable" under any interval; no choice of interval alone can tell it
// apart from a genuine post-output frame.
//
// stabilityPollDeadline bounds the total wait and is the only timing
// value here that must exceed a spawned process's real startup latency:
// on overrun, the poll fails loudly (t.Fatalf) rather than passing
// vacuously.
//
// Measured on this machine (darwin/arm64, tmux 3.7c):
//   - 08-01's original 700-900ms trace was the shell-echo-to-first-output
//     latency of an ALREADY-WARM binary — a warm exec's startup, not a
//     cold one.
//   - The G-08-1 debug session measured the FIRST in-pane exec of a
//     freshly built ~80MB binary at 1.1-1.6s; six fresh builds timed at
//     the shell were 739-1230ms, five of six above 1s.
//   - Warm execs of the same binary measured 167-226ms.
//
// The 1s interval sits below the cold-exec figure above, which is why the
// interval alone could never have guaranteed a post-output frame — only
// the readiness predicate does. The constants stay 1s/10s: retuning the
// interval moves the cliff rather than removing it and is deliberately
// not the fix here.
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
//
// It is also the basis of the altScreenOff pollUntilStable readiness
// predicate below, which re-queries it fresh on every poll sample.
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

// pollUntilStable is the package's ONLY wait primitive (D-14, amended by
// plan 08-05 to close G-08-1): it captures session's pane on
// stabilityPollInterval, and converges only when the caller's readiness
// predicate ready holds for a capture AND that capture is byte-identical
// to its immediate predecessor, with no normalization (D-15 — the default
// capture already trims trailing whitespace, which removes the main
// source of false differences).
//
// Both halves are needed. A pre-output frame — the shell prompt plus the
// echoed command, before the spawned process has written anything — is
// byte-identical to itself and therefore "stable" to any plain equality
// check; only the readiness predicate tells it apart from a genuine
// post-output frame. This is G-08-1: the cold first exec of a freshly
// built binary can exceed stabilityPollInterval, leaving two consecutive
// samples on the identical pre-output frame, which a two-sample equality
// check with no readiness condition accepted as converged.
//
// ready is required: a nil ready is a programming error and fails
// immediately via t.Fatal, before any capture is taken. Callers state
// what "settled" means for their own case — paneContains for content a
// launch or keystroke adds, altScreenOff for a quit key whose only
// guaranteed change is leaving alternate mode. An anchor must be absent
// from the typed command line that precedes it, or the pre-output frame
// satisfies it too.
//
// If stabilityPollDeadline elapses with no capture satisfying both
// halves, pollUntilStable calls t.Fatalf naming the deadline, whether
// readiness ever held during the poll, and both of the last two captures
// verbatim. Non-convergence is a FAILURE — never a skip, never a
// whole-case retry, and nothing in this package may pause for a fixed
// duration and then assert without going through this poll.
func pollUntilStable(t *testing.T, session string, ready func(capture string) bool) string {
	t.Helper()
	if ready == nil {
		t.Fatal("pollUntilStable: ready is nil — every caller must state what a settled frame must contain or satisfy (see this function's doc comment); pass paneContains or altScreenOff, never a bare byte-equality wait")
	}

	start := time.Now()
	deadline := start.Add(stabilityPollDeadline)

	sample := 0
	prev := capturePane(t, session)
	readyEverHeld := ready(prev)
	readyFirstSample := -1
	if readyEverHeld {
		readyFirstSample = sample
	}

	for {
		if time.Now().After(deadline) {
			cur := capturePane(t, session)
			if readyEverHeld {
				t.Fatalf("pollUntilStable: capture-pane never converged within %s (readiness first held at sample %d)\n--- previous capture ---\n%s\n--- latest capture ---\n%s",
					stabilityPollDeadline, readyFirstSample, prev, cur)
			}
			t.Fatalf("pollUntilStable: capture-pane never converged within %s (readiness never held during the poll)\n--- previous capture ---\n%s\n--- latest capture ---\n%s",
				stabilityPollDeadline, prev, cur)
		}
		<-time.After(stabilityPollInterval) // interval pacing via a channel wait, no blocking pause call
		sample++
		cur := capturePane(t, session)
		curReady := ready(cur)
		if curReady && !readyEverHeld {
			readyEverHeld = true
			readyFirstSample = sample
		}
		if cur == prev && curReady {
			t.Logf("pollUntilStable: converged at sample %d after %s (ready first held at sample %d)", sample, time.Since(start), readyFirstSample)
			return cur
		}
		prev = cur
	}
}

// paneContains returns a pollUntilStable readiness predicate reporting
// whether a capture contains needle. Use it as the anchor after a command
// launch or a keystroke that adds content — content the caller's own next
// assertion already requires, so the anchor and the assertion agree on
// what "settled" means.
//
// An empty needle is rejected immediately: it is satisfied by every
// capture, including a pre-output one, which is exactly the vacuity this
// whole change removes.
func paneContains(t *testing.T, needle string) func(string) bool {
	t.Helper()
	if needle == "" {
		t.Fatal("paneContains: needle is empty — an empty needle is satisfied by every capture, including a pre-output one, defeating the readiness predicate's purpose")
	}
	return func(capture string) bool {
		return strings.Contains(capture, needle)
	}
}

// altScreenOff returns a pollUntilStable readiness predicate reporting
// whether session's pane has LEFT the alternate screen, re-querying
// alternateOn fresh on every evaluation (never cached). Use it as the
// anchor after a quit key, where the only guaranteed change is tmux
// leaving alternate mode — the main buffer's content underneath (an
// echoed command line that wraps at an unpredictable column) is not a
// usable content anchor.
func altScreenOff(t *testing.T, session string) func(string) bool {
	t.Helper()
	return func(string) bool {
		return !alternateOn(t, session)
	}
}
