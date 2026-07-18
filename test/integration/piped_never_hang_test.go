// Task 2 (D-17/TUI-04): piped/closed-stdio never-hang assertions for
// `daemon` (bare) and `install`, riding the same subprocess harness
// substrate main_test.go/watch_default_test.go already built — binPath,
// runBinary, the bounded context/time.After convention. Proves that
// adding bubbletea (07-01 through 07-07) never regresses the plain/CI
// invocation path: piped stdin+stdout must exit promptly with
// non-interactive output and never block on tea.NewProgram().
package integration

import (
	"strings"
	"testing"
	"time"
)

// TestPipedNeverHang spawns the real binary for `daemon` (bare, no args)
// and `install` with piped stdin+stdout — runBinary's bytes.Buffer stdio,
// no cmd.Stdin set (the child's stdin is /dev/null), the exact
// closed/piped shape D-17 needs. Each subprocess call runs in its own
// goroutine racing a bounded time.After select so a hang FAILS the test
// instead of blocking the whole `go test` run. Both commands run under a
// throwaway HOME so neither mutates the developer's real ~/.codegraph
// daemon registry or real agent configs (T-07-08-02).
func TestPipedNeverHang(t *testing.T) {
	t.Run("daemon_bare", func(t *testing.T) {
		home := t.TempDir()
		env := []string{"HOME=" + home, "USERPROFILE=" + home}
		dir := t.TempDir()

		stdout, stderr := runPipedNeverHang(t, dir, env, "daemon")

		if !strings.Contains(stdout, "no running daemons") {
			t.Fatalf("daemon (bare, piped, throwaway HOME) stdout = %q, want it to report \"no running daemons\"", stdout)
		}
		assertNoInteractiveEscape(t, "daemon", stdout, stderr)
	})

	t.Run("install_auto_fallback", func(t *testing.T) {
		home := t.TempDir()
		env := []string{"HOME=" + home, "USERPROFILE=" + home}
		dir := t.TempDir()

		stdout, stderr := runPipedNeverHang(t, dir, env, "install")

		if strings.TrimSpace(stdout) == "" {
			t.Fatalf("install (piped, off-TTY auto fallback) produced no stdout output; stderr: %q", stderr)
		}
		assertNoInteractiveEscape(t, "install", stdout, stderr)
	})
}

// runPipedNeverHang runs the real binary via runBinary (piped bytes.Buffer
// stdio, no Stdin set — off-TTY) inside a goroutine, racing a bounded
// time.After. A hang fails the test via t.Fatalf in the calling (test)
// goroutine rather than blocking go test itself — the exact never-hang
// contract D-17/TUI-04 requires.
func runPipedNeverHang(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string) {
	t.Helper()

	done := make(chan struct{})
	var runErr error
	go func() {
		stdout, stderr, runErr = runBinary(t, dir, env, args...)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("codegraph %v (piped stdio) did not exit within 10s — likely blocked on tea.NewProgram()", args)
	}

	if runErr != nil {
		t.Fatalf("codegraph %v (piped stdio) exited non-zero: %v\nstdout: %s\nstderr: %s", args, runErr, stdout, stderr)
	}
	return stdout, stderr
}

// assertNoInteractiveEscape fails the test if stdout/stderr carry an ANSI
// escape sequence — proof the off-TTY fallback path never launched an
// interactive bubbletea Program against piped stdio.
func assertNoInteractiveEscape(t *testing.T, cmdName, stdout, stderr string) {
	t.Helper()
	if strings.Contains(stdout, "\x1b[") {
		t.Fatalf("codegraph %s stdout contains an ANSI escape sequence (interactive TUI leaked onto piped stdout): %q", cmdName, stdout)
	}
	if strings.Contains(stderr, "\x1b[") {
		t.Fatalf("codegraph %s stderr contains an ANSI escape sequence (interactive TUI leaked onto piped stderr): %q", cmdName, stderr)
	}
}
