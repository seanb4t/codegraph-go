package wireoracle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stderrJoinMarker is what the stub in TestCaptureJoinsBeforeReadingStderr
// writes to stderr AFTER its last stdout response. Nothing else in this
// package emits it, so its presence in Transcript.Stderr is unambiguous
// evidence that Capture joined the subprocess before reading the buffer.
const stderrJoinMarker = "codegraph: stderr-join-guard"

// stderrJoinStub is a stand-in for `codegraph serve --mcp` that reproduces
// the exact ordering the real server hits under CI load, deterministically
// rather than occasionally:
//
//	read one request  ->  write its stdout response  ->  PAUSE  ->  write
//	stderr  ->  exit
//
// The pause is what makes this a durable gate instead of a coin flip. With
// a deferred-only cmd.Wait(), Capture returns as soon as drainUntil has
// seen the response id — during the pause — and reads a stderrBuf the
// child has not written to yet, so Stderr comes back EMPTY every time.
// With the join before the read, Capture waits for exit and the marker is
// always present.
//
// TO SEE THIS GO RED (the check that this check works): delete the `join()`
// call immediately above the success-path `return Transcript{...}` in
// capture.go and re-run. This test must fail with an empty Stderr. If it
// still passes, the pause is too short for this machine, not the guard
// working.
// It is built with Sprintf from stderrJoinMarker rather than repeating the
// literal, so the string the stub writes and the string the assertion looks
// for cannot drift apart into a test that can never fail.
var stderrJoinStub = fmt.Sprintf(`#!/bin/sh
# Consume the request so Capture's stdin write cannot race a dead process.
read -r _request
printf '{"jsonrpc":"2.0","id":1,"result":{}}\n'
# The race window. Capture's completion condition is satisfied above; every
# byte written after this point is only observable to a caller that waits.
sleep 1
printf '%%s pause=1s\n' %q >&2
`, stderrJoinMarker)

// TestCaptureJoinsBeforeReadingStderr pins the fix for the CI-only failure
// of TestFrozenTranscriptsMatch/toolslist-narrowed ("stderr must contain
// exactly one \"codegraph: mcp-session\" line, found 0").
//
// The property under test is Transcript.Stderr's own documented promise —
// "the subprocess's COMPLETE captured stderr output". os/exec copies a
// child's stderr into a non-*os.File writer on a background goroutine that
// only cmd.Wait() joins, so reading the buffer before waiting yields
// whatever happened to have been copied. That truncation is bidirectional:
// it produced the loud failure above, and it silently makes
// assertNoSessionLine (which asserts stderr contains NO session line) pass
// VACUOUSLY. This test covers both, because both reduce to "is Stderr
// complete".
//
// It deliberately does NOT go through Scenarios(): it passes its own stub
// as binPath, so it neither adds to ExpectedScenarioCount nor needs a
// frozen transcript.
func TestCaptureJoinsBeforeReadingStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub is a /bin/sh script; this repository's harnesses target macOS and Linux only")
	}

	stubDir := t.TempDir()
	stubPath := filepath.Join(stubDir, "serve-stub")
	if err := os.WriteFile(stubPath, []byte(stderrJoinStub), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	sc := Scenario{
		Name: "stderr-join-guard",
		Requests: []map[string]any{
			{"jsonrpc": "2.0", "id": 1, "method": "ping"},
		},
	}

	tr, err := Capture(context.Background(), stubPath, t.TempDir(), t.TempDir(), sc)
	if err != nil {
		t.Fatalf("capture against stub: %v", err)
	}

	if !strings.Contains(tr.Stderr, stderrJoinMarker) {
		t.Fatalf("Transcript.Stderr is missing the stub's post-response marker %q — Capture read stderrBuf before joining the subprocess, so Stderr is truncated and every stderr assertion in this package is unsound (a present-line check fails loudly; an absent-line check passes vacuously). Got %d bytes: %q",
			stderrJoinMarker, len(tr.Stderr), tr.Stderr)
	}

	// The stdout half must be unaffected — it has its own explicitly
	// drained StdoutPipe and was never part of the race. Asserting it here
	// keeps a future "fix" that joins by discarding stdout from passing.
	if !strings.Contains(string(tr.Stdout), `"id":1`) {
		t.Fatalf("Transcript.Stdout lost the stub's response: %q", tr.Stdout)
	}
}
