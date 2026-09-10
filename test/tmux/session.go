//go:build tmux

package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"
)

// sessionWidth and sessionHeight are the fixed pane geometry every tmux
// session in this package is created with (D-12's capture instrument
// assumes a stable, known pane size).
const (
	sessionWidth  = 100
	sessionHeight = 30
)

// runTmux is the single choke point every tmux invocation in this package
// goes through: one exec.CommandContext-free exec.Command with "tmux" as
// argv[0] and every other element passed as a separate argument — never
// assembled into a shell string (T-08-01). stdout and stderr are captured
// into separate buffers so callers needing only one never accidentally
// misparse the other. On failure the returned error names the tmux
// subcommand and its args, following internal/gitmeta/worktree.go's
// exec.Command + wrapped-error idiom.
func runTmux(args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command("tmux", args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	if err != nil {
		return outBuf.String(), errBuf.String(), fmt.Errorf("tmux %v: %w: %s", args, err, errBuf.String())
	}
	return outBuf.String(), errBuf.String(), nil
}

// tmuxAvailable reports whether tmux is reachable on PATH and runnable, by
// actually invoking `tmux -V` through runTmux rather than merely checking
// exec.LookPath — a stale or broken tmux binary on PATH fails this the same
// way an absent one does.
func tmuxAvailable() error {
	_, _, err := runTmux("-V")
	return err
}

// requireTmux skips the calling test with a reason naming tmux and the
// underlying error when tmux is unavailable (D-04) — never a silent pass,
// never a bare t.Skip with no reason. Mirrors
// internal/mcp/markdown_test.go:116 and internal/githooks/githooks_test.go:31's
// skip-with-reason shape for a missing git.
func requireTmux(t *testing.T) {
	t.Helper()
	if err := tmuxAvailable(); err != nil {
		t.Skipf("tmux -V failed (tmux missing or unsupported here): %v", err)
	}
}

// sessionNameSanitizer replaces every character outside [A-Za-z0-9_-] with
// a dash when building a tmux session name from a test's own name (which
// may contain "/" for a subtest, or other characters tmux session names
// don't accept).
var sessionNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// newSession creates a new detached tmux session sized sessionWidth x
// sessionHeight and returns its name. A thin caller of newSessionSized —
// see that function's doc comment for the full contract. The three
// existing tests keep the exact pane geometry they were verified against.
func newSession(t *testing.T) string {
	t.Helper()
	return newSessionSized(t, sessionWidth, sessionHeight)
}

// newSessionSized creates a new detached tmux session sized width x height
// and returns its name. The name is "cgtmux-<pid>-<sanitized test name>" —
// unique per OS process AND per test function, which makes a session-name
// collision structurally impossible even though `go test` may run packages
// concurrently (TTY-05 concurrency edge). Teardown is registered via
// t.Cleanup immediately after creation succeeds, before returning, running
// `kill-session` and discarding its error: tmux returns exit 1 for an
// already-gone session or server, and an already-gone session is not a
// test failure.
func newSessionSized(t *testing.T, width, height int) string {
	t.Helper()

	sanitized := sessionNameSanitizer.ReplaceAllString(t.Name(), "-")
	name := fmt.Sprintf("cgtmux-%d-%s", os.Getpid(), sanitized)

	if _, stderr, err := runTmux("new-session", "-d", "-s", name,
		"-x", fmt.Sprintf("%d", width), "-y", fmt.Sprintf("%d", height)); err != nil {
		t.Fatalf("tmux new-session -d -s %s failed: %v: %s", name, err, stderr)
	}

	t.Cleanup(func() {
		_, _, _ = runTmux("kill-session", "-t", name) // already-gone is not a failure
	})

	return name
}

// sendLiteral sends text into session as literal input via
// `send-keys -l` — the form for arbitrary text (e.g. a shell command line),
// never interpreted as a tmux key name.
func sendLiteral(t *testing.T, session, text string) {
	t.Helper()
	if _, stderr, err := runTmux("send-keys", "-t", session, "-l", text); err != nil {
		t.Fatalf("tmux send-keys -l %q -t %s failed: %v: %s", text, session, err, stderr)
	}
}

// sendKey sends key into session as a tmux key name WITHOUT `-l` — for
// tmux's own key names (Enter, Space, Escape) and single barewords like
// "q", which tmux passes through as literal input when unrecognized as a
// named key.
func sendKey(t *testing.T, session, key string) {
	t.Helper()
	if _, stderr, err := runTmux("send-keys", "-t", session, key); err != nil {
		t.Fatalf("tmux send-keys %q -t %s failed: %v: %s", key, session, err, stderr)
	}
}
