//go:build tmux

package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallPickerCancelWritesNoConfig is TTY-05: the checkbox picker
// renders the unchecked glyph before any input, a space keypress flips it
// to the checked glyph, and both q and esc terminate it. A whole-tree
// sha256 over the throwaway HOME, taken before the first picker launch and
// again after both cancel paths, is byte-identical — proving cancel writes
// zero config files.
//
// The space toggle before each cancel path is load-bearing, not
// decoration: on a throwaway HOME no agent is detected, so nothing starts
// checked, and without a toggle a cancel that wrongly resolved its targets
// would still write nothing — there would be nothing for the zero-write
// assertion to be wrong about.
func TestInstallPickerCancelWritesNoConfig(t *testing.T) {
	requireTmux(t)

	home := t.TempDir()
	before := hashConfigTree(t, home)

	session := newSession(t)

	// First cancel path: q.
	sendLiteral(t, session, fmt.Sprintf("env HOME=%s USERPROFILE=%s %s install", home, home, binPath))
	sendKey(t, session, "Enter")
	capture := pollUntilStable(t, session, paneContains(t, "[ ]"))
	if !strings.Contains(capture, "[ ]") {
		t.Fatalf("TTY-05: converged capture does not contain the unchecked checkbox glyph \"[ ]\":\n%s", capture)
	}
	// Positive control proving the pane isn't blank or crashed: the help
	// footer text and every registered target's display name. Since FIX-03
	// (D-24/D-25), each picker row costs exactly one line — the delegate's
	// own Render no longer writes a trailing newline, so bubbles/v2/list's
	// populatedView separator is the only newline per row — so all 8 rows
	// plus the footer now fit within this package's default 100x30 pane.
	// The footer is the positive control, not the picker's title, because
	// it also proves the fix: before FIX-03 the footer never appeared at
	// this geometry (verified via temporary, reverted debug
	// instrumentation; see 08-02-SUMMARY.md).
	if !strings.Contains(capture, "space: toggle") {
		t.Fatalf("TTY-05: converged capture does not contain the footer text \"space: toggle\":\n%s", capture)
	}
	for _, name := range []string{
		"Antigravity", "Claude Code", "Codex CLI", "Cursor",
		"Gemini CLI", "Hermes Agent", "Kiro", "opencode",
	} {
		if !strings.Contains(capture, name) {
			t.Fatalf("TTY-05: converged capture does not contain target display name %q:\n%s", name, capture)
		}
	}

	sendKey(t, session, "Space")
	capture = pollUntilStable(t, session, paneContains(t, "[x]"))
	if !strings.Contains(capture, "[x]") {
		t.Fatalf("TTY-05: converged capture after space does not contain the checked checkbox glyph \"[x]\":\n%s", capture)
	}

	sendKey(t, session, "q")
	capture = pollUntilStable(t, session, altScreenOff(t, session))
	if alternateOn(t, session) {
		t.Fatalf("TTY-05: alternateOn is still true after q — the picker did not exit:\n%s", capture)
	}

	// Second cancel path: esc, same session, same home.
	sendLiteral(t, session, fmt.Sprintf("env HOME=%s USERPROFILE=%s %s install", home, home, binPath))
	sendKey(t, session, "Enter")
	pollUntilStable(t, session, paneContains(t, "[ ]"))
	sendKey(t, session, "Space")
	pollUntilStable(t, session, paneContains(t, "[x]"))
	sendKey(t, session, "Escape")
	capture = pollUntilStable(t, session, altScreenOff(t, session))
	if alternateOn(t, session) {
		t.Fatalf("TTY-05: alternateOn is still true after Escape — the picker did not exit:\n%s", capture)
	}

	after := hashConfigTree(t, home)
	if after != before {
		var paths []string
		_ = filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})
		t.Fatalf("TTY-05: config-tree hash changed after cancel — before=%s after=%s; paths present under %s: %v", before, after, home, paths)
	}
}
