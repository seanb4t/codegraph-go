package daemon

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

// startedAtFor returns the StartedAt a lockInfo/Record fixture naming pid
// carries: pid's start time as processStartTime reports it, the value the
// pid's own writer stamps (selfStartedAt). For a dead pid, or where
// processStartTime is unavailable, it falls back to time.Now(); isStale then
// decides on liveness alone and never compares the value.
func startedAtFor(pid int) time.Time {
	if actual, ok := processStartTime(pid); ok {
		return actual
	}
	return time.Now().UTC()
}

// writeLock writes a lockfile for pid directly, bypassing acquire, so
// tests can set up a specific (live or dead) pid without needing a real
// daemon process running.
func writeLock(t *testing.T, dir string, pid int) {
	t.Helper()
	writeLockInfo(t, dir, lockInfo{PID: pid, StartedAt: startedAtFor(pid)})
}

// writeLockInfo writes info as the lockfile at dir, bypassing acquire.
func writeLockInfo(t *testing.T, dir string, info lockInfo) {
	t.Helper()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal lockInfo: %v", err)
	}
	if err := os.WriteFile(lockPath(dir), data, 0o644); err != nil {
		t.Fatalf("writing lockfile: %v", err)
	}
}

// deadPID returns a pid that was valid a moment ago but is now guaranteed
// dead: it re-execs this test binary with a run filter matching nothing,
// so the child process starts and exits immediately (portable,
// dependency-free — no reliance on a system "true"/"sleep" binary being on
// PATH).
func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatalf("spawning throwaway process: %v", err)
	}
	return cmd.Process.Pid
}

// TestUnlockStaleOnly is the plan's primary acceptance gate (T-04-07-01):
// unlock clears a lock owned by a dead pid, but refuses — and leaves
// untouched — a lock owned by the current (live) process.
func TestUnlockStaleOnly(t *testing.T) {
	t.Run("dead pid lock is cleared", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, deadPID(t))

		msg, err := Unlock(dir)
		if err != nil {
			t.Fatalf("Unlock: unexpected error: %v", err)
		}
		if msg == "" {
			t.Fatal("Unlock: expected a non-empty confirmation message for a cleared stale lock")
		}
		if _, ok, rerr := readLock(dir); rerr != nil || ok {
			t.Fatalf("Unlock: lockfile still present after clearing a stale lock (ok=%v err=%v)", ok, rerr)
		}
	})

	t.Run("live pid lock is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, os.Getpid())

		_, err := Unlock(dir)
		if !errors.Is(err, ErrLockLive) {
			t.Fatalf("Unlock: got err %v, want ErrLockLive", err)
		}
		if _, ok, rerr := readLock(dir); rerr != nil || !ok {
			t.Fatalf("Unlock: live lock was removed (ok=%v err=%v) — must never clear a live daemon's lock", ok, rerr)
		}
	})
}

// TestUnlockAbsentIsNoOp proves Unlock on a directory with no lockfile at
// all is a clean, error-free no-op.
func TestUnlockAbsentIsNoOp(t *testing.T) {
	dir := t.TempDir()
	msg, err := Unlock(dir)
	if err != nil {
		t.Fatalf("Unlock: unexpected error on absent lock: %v", err)
	}
	if msg == "" {
		t.Fatal("Unlock: expected a non-empty no-op message for an absent lock")
	}
	if _, ok, rerr := readLock(dir); rerr != nil || ok {
		t.Fatalf("Unlock: unexpectedly created a lockfile (ok=%v err=%v)", ok, rerr)
	}
}

// TestAcquireRejectsLiveLock proves acquire never overwrites a live lock —
// the other half of the single-writer invariant (a second daemon must not
// start while one is already running).
func TestAcquireRejectsLiveLock(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, os.Getpid())

	err := acquire(dir)
	if !errors.Is(err, ErrLockLive) {
		t.Fatalf("acquire: got err %v, want ErrLockLive", err)
	}
	if _, ok, rerr := readLock(dir); rerr != nil || !ok {
		t.Fatalf("acquire: live lock was removed on a rejected acquire (ok=%v err=%v)", ok, rerr)
	}
}

// TestAcquireClearsStaleLock proves acquire itself recovers from a
// crashed daemon's stale lock rather than requiring a separate unlock
// step first.
func TestAcquireClearsStaleLock(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, deadPID(t))

	if err := acquire(dir); err != nil {
		t.Fatalf("acquire: unexpected error over a stale lock: %v", err)
	}

	info, ok, err := readLock(dir)
	if err != nil || !ok {
		t.Fatalf("acquire: no lockfile present after a successful acquire (ok=%v err=%v)", ok, err)
	}
	if info.PID != os.Getpid() {
		t.Fatalf("acquire: lockfile pid = %d, want this process's pid %d", info.PID, os.Getpid())
	}

	if err := release(dir); err != nil {
		t.Fatalf("release: %v", err)
	}
}

// TestIsStale proves isStale's pid-liveness classification directly:
// false for the current, definitely-live process; true for a definitely-
// dead one.
func TestIsStale(t *testing.T) {
	if isStale(lockInfo{PID: os.Getpid(), StartedAt: startedAtFor(os.Getpid())}) {
		t.Fatal("isStale: reported the current live process as stale")
	}

	dead := deadPID(t)
	if !isStale(lockInfo{PID: dead, StartedAt: startedAtFor(dead)}) {
		t.Fatalf("isStale: reported dead pid %d as live", dead)
	}
}

// TestIsProcessLive: this process is live; a dead pid and every value outside
// the range of a process id are not, including this process's negated process
// group id, which kill(2) would accept.
func TestIsProcessLive(t *testing.T) {
	if !isProcessLive(os.Getpid()) {
		t.Fatalf("isProcessLive(own pid %d) = false, want true", os.Getpid())
	}
	for _, pid := range []int{deadPID(t), 0, -1, -syscall.Getpgrp(), math.MaxInt32 + 1, 4294967295, 4294967296} {
		if isProcessLive(pid) {
			t.Errorf("isProcessLive(%d) = true, want false", pid)
		}
	}
}

// TestIsStaleWithoutStartTimes: where start times are unavailable, a live pid
// is not stale whatever its StartedAt says, and a dead pid still is.
func TestIsStaleWithoutStartTimes(t *testing.T) {
	prev := startTimeOf
	startTimeOf = func(int) (time.Time, bool) { return time.Time{}, false }
	t.Cleanup(func() { startTimeOf = prev })

	now := time.Now().UTC()
	for _, skew := range []time.Duration{-time.Hour, 0, time.Hour} {
		if isStale(lockInfo{PID: os.Getpid(), StartedAt: now.Add(skew)}) {
			t.Errorf("isStale(live pid, StartedAt=now%+v) = true without start times, want false", skew)
		}
	}
	dead := deadPID(t)
	if !isStale(lockInfo{PID: dead, StartedAt: now}) {
		t.Fatalf("isStale(dead pid %d) = false without start times, want true", dead)
	}
}

// TestAcquireConcurrentRaceOnlyOneWinner is the CR-02 regression: many
// goroutines racing acquire() against the SAME (initially absent)
// lockfile path must see exactly one success — the exclusive-create fix
// (O_EXCL) makes this deterministic; the prior read-then-unconditional-
// WriteFile implementation could let more than one "win" within the same
// race window, each believing it was the sole daemon.
func TestAcquireConcurrentRaceOnlyOneWinner(t *testing.T) {
	dir := t.TempDir()

	const racers = 32
	var wg sync.WaitGroup
	results := make(chan error, racers)
	// A barrier so every goroutine's acquire() call starts as close to
	// simultaneously as possible, maximizing the chance of exercising the
	// TOCTOU window if the fix regresses.
	start := make(chan struct{})

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- acquire(dir)
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, ErrLockLive) {
			t.Fatalf("acquire: unexpected error (want nil or ErrLockLive): %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("acquire: %d of %d concurrent racers succeeded, want exactly 1 — CR-02 regression (two daemons both believe they hold the lock)", successes, racers)
	}

	info, ok, err := readLock(dir)
	if err != nil || !ok {
		t.Fatalf("readLock after race: ok=%v err=%v, want a lockfile present", ok, err)
	}
	if info.PID != os.Getpid() {
		t.Fatalf("readLock after race: pid=%d, want this process's pid %d", info.PID, os.Getpid())
	}
}

// requireProcessStartTime returns this test binary's start time as
// processStartTime reports it, the value isStale corroborates a record for
// os.Getpid() against. It skips where processStartTime reports no start time,
// because isStale is liveness-only there; TestProcessStartTime fails on Linux
// and macOS when that happens.
func requireProcessStartTime(t *testing.T) time.Time {
	t.Helper()
	actual, ok := processStartTime(os.Getpid())
	if !ok {
		t.Skip("processStartTime unavailable on this platform — isStale is liveness-only, the corroboration branch is unreachable")
	}
	return actual
}

// TestSelfRecordSurvivesStaleness: a lockfile or registry record naming this
// process and stamped by selfStartedAt carries the OS-reported start time
// exactly and is never classified stale.
func TestSelfRecordSurvivesStaleness(t *testing.T) {
	actual := requireProcessStartTime(t)

	self := os.Getpid()
	if stamped := selfStartedAt(); isStale(lockInfo{PID: self, StartedAt: stamped}) {
		t.Fatalf("isStale reported this live process stale: recorded StartedAt=%v, OS-reported start %v", stamped, actual)
	}

	// The same stamp routed through the registry self-heal in List.
	withRegistryDir(t)
	rec := Record{PID: self, StartedAt: selfStartedAt(), RepoRoot: "/live/repo"}
	if err := Register(rec); err != nil {
		t.Fatalf("Register: %v", err)
	}
	live, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(live) != 1 || live[0] != rec {
		t.Fatalf("List pruned this live process's own record: got %+v, want exactly %+v", live, rec)
	}
	if !live[0].StartedAt.Equal(actual) {
		t.Fatalf("listed record StartedAt=%v, want the OS-reported start %v exactly", live[0].StartedAt, actual)
	}
}

// TestIsStaleStillDetectsPIDReuse is the security counterpart of
// TestSelfRecordSurvivesStaleness: isStale treats a live pid whose
// OS-reported start is more than procStartTimeSlack after the recorded
// StartedAt as a recycled pid (WR-02 / T-04-07-01), and keeps a record
// stamped later than the start.
func TestIsStaleStillDetectsPIDReuse(t *testing.T) {
	actual := requireProcessStartTime(t)

	for _, tc := range []struct {
		name  string
		skew  time.Duration
		stale bool
	}{
		{"recorded an hour before actual start (pid recycled)", -time.Hour, true},
		{"recorded just beyond the slack before actual start", -(procStartTimeSlack + time.Second), true},
		{"recorded exactly the slack before actual start", -procStartTimeSlack, false},
		{"recorded exactly at the OS-reported start", 0, false},
		{"recorded exactly the slack after actual start", procStartTimeSlack, false},
		{"recorded just beyond the slack after actual start", procStartTimeSlack + time.Second, false},
		{"recorded an hour after actual start (wall-clock stamp)", time.Hour, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := isStale(lockInfo{PID: os.Getpid(), StartedAt: actual.Add(tc.skew)})
			if got != tc.stale {
				t.Fatalf("isStale(live pid, StartedAt=actual%+v) = %v, want %v", tc.skew, got, tc.stale)
			}
		})
	}
}

// TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck: acquire() stamps the
// lock with the process's OS-reported start time exactly, the lock is not
// stale, and it keeps a second acquire() out (INDX-05 / T-04-07-01).
func TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck(t *testing.T) {
	actualStart := requireProcessStartTime(t)

	dir := t.TempDir()
	if err := acquire(dir); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	info, ok, err := readLock(dir)
	if err != nil {
		t.Fatalf("readLock: %v", err)
	}
	if !ok {
		t.Fatal("readLock: acquire left no lockfile")
	}
	if !info.StartedAt.Equal(actualStart) {
		t.Fatalf("acquire stamped StartedAt=%v, want the OS-reported start %v exactly", info.StartedAt, actualStart)
	}
	if isStale(info) {
		t.Fatalf("acquire wrote a lock that is already stale: recorded StartedAt=%v, OS-reported start of pid %d=%v", info.StartedAt, info.PID, actualStart)
	}

	if err := acquire(dir); !errors.Is(err, ErrLockLive) {
		t.Fatalf("second acquire() against a lock this same live process holds = %v, want ErrLockLive — the single-writer invariant is defeated", err)
	}
}

// TestAcquireKeepsLockWithWallClockStartedAt: a lock naming a live pid whose
// StartedAt is later than the process's OS-reported start, as a wall-clock
// stamp is, belongs to a running daemon. acquire refuses it and leaves the
// file exactly as it was.
func TestAcquireKeepsLockWithWallClockStartedAt(t *testing.T) {
	actual := requireProcessStartTime(t)

	dir := t.TempDir()
	stamped := actual.Add(time.Hour)
	writeLockInfo(t, dir, lockInfo{PID: os.Getpid(), StartedAt: stamped})

	if err := acquire(dir); !errors.Is(err, ErrLockLive) {
		t.Fatalf("acquire over a live lock stamped %v after the process start = %v, want ErrLockLive", time.Hour, err)
	}
	info, ok, err := readLock(dir)
	if err != nil || !ok {
		t.Fatalf("readLock after refused acquire: ok=%v err=%v, want the original lockfile", ok, err)
	}
	if info.PID != os.Getpid() || !info.StartedAt.Equal(stamped) {
		t.Fatalf("lockfile after refused acquire = %+v, want pid %d StartedAt %v unchanged", info, os.Getpid(), stamped)
	}
}

// TestAcquireWithoutStartTimeKeepsItsLock: a lock acquire writes while this
// process's start time cannot be read still corroborates once it can, so the
// lock is not stale and a second acquire is refused.
func TestAcquireWithoutStartTimeKeepsItsLock(t *testing.T) {
	requireProcessStartTime(t)
	dir := t.TempDir()

	prev := startTimeOf
	t.Cleanup(func() { startTimeOf = prev })
	startTimeOf = func(int) (time.Time, bool) { return time.Time{}, false }
	err := acquire(dir)
	startTimeOf = prev
	if err != nil {
		t.Fatalf("acquire without a start time: %v", err)
	}

	info, ok, err := readLock(dir)
	if err != nil || !ok {
		t.Fatalf("readLock after acquire: ok=%v err=%v, want a lockfile", ok, err)
	}
	if isStale(info) {
		t.Fatalf("lock stamped StartedAt=%v without a start time reads stale once start times are available", info.StartedAt)
	}
	if err := acquire(dir); !errors.Is(err, ErrLockLive) {
		t.Fatalf("second acquire = %v, want ErrLockLive", err)
	}
}

// TestRecycledPidLockIsCleared: a lock naming a live pid whose OS-reported
// start is an hour after its StartedAt belongs to a recycled pid. acquire
// replaces it with a lock stamped with this process's start, and Unlock
// removes it.
func TestRecycledPidLockIsCleared(t *testing.T) {
	actual := requireProcessStartTime(t)
	recycled := lockInfo{PID: os.Getpid(), StartedAt: actual.Add(-time.Hour)}

	t.Run("acquire replaces it", func(t *testing.T) {
		dir := t.TempDir()
		writeLockInfo(t, dir, recycled)

		if err := acquire(dir); err != nil {
			t.Fatalf("acquire over a recycled-pid lock: %v, want nil", err)
		}
		info, ok, err := readLock(dir)
		if err != nil || !ok {
			t.Fatalf("readLock after acquire: ok=%v err=%v, want a lockfile", ok, err)
		}
		if !info.StartedAt.Equal(actual) {
			t.Fatalf("acquire stamped StartedAt=%v, want the OS-reported start %v", info.StartedAt, actual)
		}
	})

	t.Run("Unlock removes it", func(t *testing.T) {
		dir := t.TempDir()
		writeLockInfo(t, dir, recycled)

		if _, err := Unlock(dir); err != nil {
			t.Fatalf("Unlock of a recycled-pid lock: %v, want nil", err)
		}
		if _, ok, err := readLock(dir); err != nil || ok {
			t.Fatalf("readLock after Unlock: ok=%v err=%v, want no lockfile", ok, err)
		}
	})
}

// TestStartTimesCorroborate proves the WR-02 one-sided comparison: a recorded
// time up to procStartTimeSlack before actual, or any amount after it,
// corroborates; a recorded time further before actual does not.
func TestStartTimesCorroborate(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name             string
		recorded, actual time.Time
		want             bool
	}{
		{"identical", base, base, true},
		{"recorded slightly after actual, within slack", base.Add(2 * time.Second), base, true},
		{"recorded slightly before actual, within slack", base.Add(-2 * time.Second), base, true},
		{"recorded well after actual (wall-clock stamp)", base.Add(time.Hour), base, true},
		{"recorded 5s before actual, at the slack", base.Add(-5 * time.Second), base, true},
		{"recorded 6s before actual, beyond slack", base.Add(-6 * time.Second), base, false},
		{"recorded well before actual, beyond slack (PID reuse)", base.Add(-time.Hour), base, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := startTimesCorroborate(tc.recorded, tc.actual); got != tc.want {
				t.Errorf("startTimesCorroborate(%v, %v) = %v, want %v", tc.recorded, tc.actual, got, tc.want)
			}
		})
	}
}
