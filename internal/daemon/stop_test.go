//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestHelperProcess isn't a real test. It's re-exec'd by liveProcess as a
// throwaway subprocess that blocks until signaled, so TestSendStop has a
// real live pid to SIGTERM. Gated behind GO_WANT_HELPER_PROCESS so an
// ordinary `go test` run of this package (which also runs TestHelperProcess
// as a subtest name match) exits immediately instead of blocking forever.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	select {} // blocks until the parent test delivers a real signal
}

// liveProcess starts a real, currently-running subprocess (re-exec'ing this
// test binary into TestHelperProcess's blocking branch) so sendStop has a
// live pid to signal — portable and dependency-free, no reliance on a
// system "sleep" binary being on PATH (mirrors deadPID's re-exec idiom in
// lock_test.go).
func liveProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting live helper process: %v", err)
	}
	return cmd
}

// TestSendStop is the plan's primary acceptance gate for Task 1: sendStop
// delivers a real SIGTERM (not Signal(0)) that terminates a live process,
// and errors when the target pid is already dead.
func TestSendStop(t *testing.T) {
	t.Run("SIGTERM terminates a live process", func(t *testing.T) {
		cmd := liveProcess(t)

		if err := sendStop(cmd.Process.Pid); err != nil {
			t.Fatalf("sendStop: %v", err)
		}

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
			// SIGTERM's default disposition is termination — cmd.Wait
			// returning (even with a non-zero/signaled *exec.ExitError) is
			// exactly the expected outcome, not a failure.
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-done
			t.Fatal("sendStop: process did not exit within 5s of SIGTERM")
		}
	})

	t.Run("dead pid errors", func(t *testing.T) {
		if err := sendStop(deadPID(t)); err == nil {
			t.Fatal("sendStop: expected an error signaling a definitely-dead pid")
		}
	})
}
