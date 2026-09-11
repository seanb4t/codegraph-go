//go:build tmux

package tmux

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// pollContractAnchor is the readiness anchor
// TestPollUntilStableDoesNotConvergeOnPreOutputFrame types into the pane as
// "cgtmux-''ready" (an empty-quote split) and looks for, unbroken, in the
// settled output line ("cgtmux-ready").
const pollContractAnchor = "cgtmux-ready"

// TestPollUntilStableDoesNotConvergeOnPreOutputFrame is a self-test of the
// harness's own wait primitive, not of the product under test.
//
// It reproduces G-08-1's cold-start race deterministically: a process whose
// first output arrives later than one stabilityPollInterval leaves two
// consecutive samples on the byte-identical pre-output frame, which a plain
// two-sample equality check accepts as converged even though the command
// has not finished writing anything.
//
// Rather than relying on the real codegraph binary's own cold-start latency
// (1.1-1.6s measured in-pane on this machine — see
// .planning/debug/tty03-cold-start-poll-race.md — which reproduces the race
// only on a cold machine, and only probabilistically), this test commands
// the pane's own shell to delay its only output by a fixed number of poll
// intervals. That reproduces the race on every machine, every run, so the
// property is checked deterministically instead of probabilistically.
//
// The anchor is typed as "cgtmux-''ready" — an empty-quote split every
// POSIX shell and fish concatenate into the unbroken output
// "cgtmux-ready" — because the echoed command line is itself part of every
// capture. An anchor that appeared unbroken in the typed command line too
// would be satisfied by the pre-output frame, making this test vacuous.
func TestPollUntilStableDoesNotConvergeOnPreOutputFrame(t *testing.T) {
	requireTmux(t)
	session := newSession(t)

	delaySeconds := int(math.Ceil((4 * stabilityPollInterval).Seconds()))
	delay := time.Duration(delaySeconds) * time.Second
	if delay <= 2*stabilityPollInterval || delay+3*stabilityPollInterval >= stabilityPollDeadline {
		t.Fatalf("test self-consistency: delay=%s must be > 2*stabilityPollInterval=%s and delay+3*stabilityPollInterval must be < stabilityPollDeadline=%s, got delay+3*stabilityPollInterval=%s",
			delay, 2*stabilityPollInterval, stabilityPollDeadline, delay+3*stabilityPollInterval)
	}

	sendLiteral(t, session, fmt.Sprintf("sleep %d; echo cgtmux-''ready", delaySeconds))
	start := time.Now()
	sendKey(t, session, "Enter")

	capture := pollUntilStable(t, session)
	elapsed := time.Since(start)

	if !strings.Contains(capture, pollContractAnchor) {
		t.Fatalf("pollUntilStable converged on a frame without the anchor %q after %s — the pre-output frame was accepted as stable:\n%s",
			pollContractAnchor, elapsed, capture)
	}
	if elapsed < delay {
		t.Fatalf("pollUntilStable returned after %s, before the commanded delay of %s had passed — it cannot have waited for the output it returned",
			elapsed, delay)
	}
}
