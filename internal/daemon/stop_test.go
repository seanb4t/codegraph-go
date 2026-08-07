//go:build !windows

package daemon

import (
	"encoding/json"
	"fmt"
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
		case <-time.After(testBudget(5 * time.Second)):
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

// captureStopSignal replaces the injectable stopSignal seam (stop.go) with
// a recording stub for the duration of the test — the same test-seam
// convention as withRegistryDir — so StopAll/StopMatching tests can assert
// exactly which pids were targeted without ever delivering a real OS
// signal to the test process itself.
func captureStopSignal(t *testing.T) *[]int {
	t.Helper()
	called := new([]int)
	prev := stopSignal
	stopSignal = func(pid int) error {
		*called = append(*called, pid)
		return nil
	}
	t.Cleanup(func() { stopSignal = prev })
	return called
}

// TestStopAll is the plan's primary acceptance gate for Task 2: StopAll
// signals only a live, corroborated record — a stale one is never passed
// to stopSignal (the daemon-stop safety invariant, T-07-04-01) — targets
// are de-duplicated by pid, and an empty registry is a clean no-op.
func TestStopAll(t *testing.T) {
	t.Run("signals only the live record, skips the stale one", func(t *testing.T) {
		dir := withRegistryDir(t)
		called := captureStopSignal(t)

		live := Record{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid()), RepoRoot: "/live/repo"}
		if err := Register(live); err != nil {
			t.Fatalf("Register live: %v", err)
		}

		stale := Record{PID: deadPID(t), StartedAt: time.Now().UTC(), RepoRoot: "/dead/repo"}
		staleData, err := json.Marshal(stale)
		if err != nil {
			t.Fatalf("marshal stale record: %v", err)
		}
		writeRawRecord(t, dir, fmt.Sprintf("%d.json", stale.PID), staleData)

		got, err := StopAll()
		if err != nil {
			t.Fatalf("StopAll: %v", err)
		}
		if len(got) != 1 || got[0] != live {
			t.Fatalf("StopAll: got %+v, want exactly the live record %+v", got, live)
		}
		if len(*called) != 1 || (*called)[0] != live.PID {
			t.Fatalf("StopAll: stopSignal calls = %v, want exactly [%d] (stale record must never be signaled)", *called, live.PID)
		}
	})

	t.Run("empty registry is a clean no-op", func(t *testing.T) {
		withRegistryDir(t)
		captureStopSignal(t)

		got, err := StopAll()
		if err != nil || got != nil {
			t.Fatalf("StopAll on empty registry: got %v, err %v; want nil, nil", got, err)
		}
	})

	t.Run("de-duplicates targets by pid", func(t *testing.T) {
		dir := withRegistryDir(t)
		called := captureStopSignal(t)

		live := Record{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid()), RepoRoot: "/live/repo"}
		data, err := json.Marshal(live)
		if err != nil {
			t.Fatalf("marshal live record: %v", err)
		}
		writeRawRecord(t, dir, "dup-a.json", data)
		writeRawRecord(t, dir, "dup-b.json", data)

		got, err := StopAll()
		if err != nil {
			t.Fatalf("StopAll: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("StopAll: got %d records, want exactly 1 (de-duplicated by pid): %+v", len(got), got)
		}
		if len(*called) != 1 {
			t.Fatalf("StopAll: stopSignal called %d times, want exactly 1 (de-duplicated by pid): %v", len(*called), *called)
		}
	})
}

// TestStopMatching is the plan's primary acceptance gate for Task 2's
// target-filtering half: StopMatching signals only records whose RepoRoot
// matches the requested one, and a no-match is a clean no-op.
func TestStopMatching(t *testing.T) {
	t.Run("signals only the matching repoRoot", func(t *testing.T) {
		withRegistryDir(t)
		called := captureStopSignal(t)

		match := Record{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid()), RepoRoot: "/repo/a"}
		if err := Register(match); err != nil {
			t.Fatalf("Register match: %v", err)
		}

		other := Record{PID: deadPID(t), StartedAt: time.Now().UTC(), RepoRoot: "/repo/b"}
		if err := Register(other); err != nil {
			t.Fatalf("Register other: %v", err)
		}

		got, err := StopMatching("/repo/a")
		if err != nil {
			t.Fatalf("StopMatching: %v", err)
		}
		if len(got) != 1 || got[0] != match {
			t.Fatalf("StopMatching: got %+v, want exactly %+v", got, match)
		}
		if len(*called) != 1 || (*called)[0] != match.PID {
			t.Fatalf("StopMatching: stopSignal calls = %v, want exactly [%d]", *called, match.PID)
		}
	})

	t.Run("no match is a clean no-op", func(t *testing.T) {
		withRegistryDir(t)
		captureStopSignal(t)

		live := Record{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid()), RepoRoot: "/repo/a"}
		if err := Register(live); err != nil {
			t.Fatalf("Register: %v", err)
		}

		got, err := StopMatching("/repo/does-not-match")
		if err != nil || got != nil {
			t.Fatalf("StopMatching with no match: got %v, err %v; want nil, nil", got, err)
		}
	})
}
