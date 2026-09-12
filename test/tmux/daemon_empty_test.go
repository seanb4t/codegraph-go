//go:build tmux

package tmux

import (
	"fmt"
	"strings"
	"testing"
)

// decrqmResponseMarkers holds the two DECRQM (mode-query) capability-probe
// response literals G-07-1 leaked, exactly as `capture-pane -p -e -C -S -`
// renders them: bubbletea's synchronized-output (mode 2026) and
// grapheme-clustering (mode 2027) probe replies, echoed visibly by the
// shell's own line editor once the Program has already quit and nothing
// consumes them. Sourced verbatim from 08-RESEARCH.md § Code Examples and
// independently reproduced on this machine (tmux 3.7c) against a binary
// built from the corrected two-file mutation (08-CONTEXT.md D-06 family
// (a), 08-RESEARCH.md Pitfall 1) before being reverted byte-clean.
var decrqmResponseMarkers = []string{
	"^[[?2026;2$y",
	"^[[?2027;0$y",
}

// TestDaemonEmptyRegistryLeaksNoModeQueryBytes is the phase's tracer
// assertion (TTY-03): on an empty daemon registry, bare `daemon` must print
// ONLY "no running daemons" — no leaked DECRQM capability-probe response
// bytes anywhere in the stabilized capture. Both halves matter (standing
// rule 84d1gfpywd): the positive half (the line IS present) proves the
// harness actually observed the real command run rather than an empty
// pane; the negative half (neither marker is present) proves the
// empty-registry short-circuit still holds.
//
// The poll anchors on the same "no running daemons" line the positive
// half asserts (D-14 as amended by plan 08-05), which is what keeps a
// cold first exec (G-08-1) from being converged on before the binary has
// written anything; the explicit assertion below stays as the documented
// positive half (rule 84d1gfpywd).
func TestDaemonEmptyRegistryLeaksNoModeQueryBytes(t *testing.T) {
	requireTmux(t)

	home := t.TempDir()
	session := newSession(t)

	sendLiteral(t, session, fmt.Sprintf("env HOME=%s USERPROFILE=%s %s daemon", home, home, binPath))
	sendKey(t, session, "Enter")

	capture := pollUntilStable(t, session, paneContains(t, "no running daemons"))

	if !strings.Contains(capture, "no running daemons") {
		t.Fatalf("TTY-03: stabilized capture does not contain \"no running daemons\" — the harness may not have observed the real command run:\n%s", capture)
	}

	for _, marker := range decrqmResponseMarkers {
		if strings.Contains(capture, marker) {
			t.Fatalf("TTY-03: stabilized capture leaks DECRQM mode-query response marker %q — empty-registry escape hygiene violated:\n%s", marker, capture)
		}
	}
}
