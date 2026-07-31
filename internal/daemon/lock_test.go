package daemon

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// startedAtFor returns the StartedAt a lockInfo/Record naming pid must carry
// to be a physically POSSIBLE fixture — pid's actual start time as the OS
// reports it, which is exactly what isStale's WR-02 corroboration compares
// against (startTimesCorroborate, within procStartTimeSlack).
//
// Fixtures must NOT use time.Now() for a pid that is already running, above
// all not for os.Getpid(). Doing so describes a process that started just now
// under a pid that demonstrably started earlier — the precise signature of the
// PID reuse isStale exists to catch. It is harmless while the gap is small,
// so it passes on any platform without OS corroboration (procstart_other.go
// makes the branch unreachable off Linux) and on a young test binary; on Linux
// it silently turns every such fixture stale once the binary has been alive
// past the slack, which is what made TestStopAll/TestStopMatching/TestIsStale
// and friends fail on CI only, and only sometimes. /proc's start time is
// derived from a whole-second btime plus an integer tick division, so it reads
// up to ~2s earlier than reality and the effective window is ~3s of process
// lifetime, not the nominal 5s.
//
// A dead pid has no /proc entry, so this falls back to time.Now() there — the
// value is irrelevant in that case because isStale's liveness check rejects
// the record before corroboration is ever consulted.
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
	data, err := json.Marshal(lockInfo{PID: pid, StartedAt: startedAtFor(pid)})
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

// requireAgedProcess blocks until this test binary is demonstrably OLDER than
// procStartTimeSlack as measured against the very clock isStale corroborates
// against (processStartTime), and returns that OS-reported start time. It
// skips on any platform where that corroboration data is unavailable (every
// non-Linux target — procstart_other.go), because isStale degrades to
// liveness-only there and the condition under test cannot arise.
//
// This exists so the aged-process regression below fails DETERMINISTICALLY
// rather than depending on how long the preceding tests happened to take on a
// given runner — the exact non-determinism that let this bug reach CI as an
// intermittent "sometimes 9 tests fail, sometimes 2" flake.
func requireAgedProcess(t *testing.T) time.Time {
	t.Helper()
	actual, ok := processStartTime(os.Getpid())
	if !ok {
		t.Skip("processStartTime unavailable on this platform — isStale is liveness-only, the corroboration branch is unreachable")
	}
	const margin = time.Second
	deadline := time.Now().Add(procStartTimeSlack + 2*margin)
	for time.Since(actual) <= procStartTimeSlack+margin {
		if time.Now().After(deadline) {
			t.Fatalf("requireAgedProcess: binary never reached age %v (start=%v)", procStartTimeSlack+margin, actual)
		}
		time.Sleep(25 * time.Millisecond)
	}
	return actual
}

// TestSelfRecordSurvivesStalenessOnAgedProcess is the regression guard for the
// Linux-only daemon flake: a lockfile/registry record naming THIS process must
// never be classified stale, no matter how long this process has been running.
//
// isStale corroborates a live pid's recorded StartedAt against the OS-reported
// actual process start time (/proc/<pid>/stat) within procStartTimeSlack. A
// fixture that pairs os.Getpid() with time.Now() therefore describes a
// physically impossible process — one that started just now under a pid that
// demonstrably started earlier — and is correctly rejected as a PID-reuse
// forgery once the gap exceeds the slack. Because /proc's start time is
// derived from a whole-second btime plus an integer tick division, it reads up
// to ~2s EARLIER than reality, so the effective window is ~3s of process
// lifetime, not the nominal 5s.
func TestSelfRecordSurvivesStalenessOnAgedProcess(t *testing.T) {
	actual := requireAgedProcess(t)

	self := os.Getpid()
	if isStale(lockInfo{PID: self, StartedAt: startedAtFor(self)}) {
		t.Fatalf("isStale reported this live process stale after %v of uptime (OS-reported start %v): a self-referencing fixture must derive StartedAt via startedAtFor, not time.Now()", time.Since(actual).Round(time.Millisecond), actual)
	}

	// The same fixture routed through the two engines that consume it — the
	// registry self-heal (List) and the stop path (stopTargets) — since those
	// are where the flake actually surfaced, not in isStale directly.
	withRegistryDir(t)
	rec := Record{PID: self, StartedAt: startedAtFor(self), RepoRoot: "/live/repo"}
	if err := Register(rec); err != nil {
		t.Fatalf("Register: %v", err)
	}
	live, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(live) != 1 || live[0] != rec {
		t.Fatalf("List pruned this live process's own record after %v of uptime: got %+v, want exactly %+v", time.Since(actual).Round(time.Millisecond), live, rec)
	}
}

// TestIsStaleStillDetectsPIDReuseOnAgedProcess is the security counterpart of
// the regression above: whatever the fixtures do, isStale must STILL treat a
// live pid whose recorded StartedAt is far from the OS-reported start time as
// stale. This is WR-02 / T-04-07-01's PID-reuse mitigation, and it must pass
// both before and after the fixture fix — proof the fix narrows the fixtures,
// not the check.
func TestIsStaleStillDetectsPIDReuseOnAgedProcess(t *testing.T) {
	actual := requireAgedProcess(t)

	for _, tc := range []struct {
		name  string
		skew  time.Duration
		stale bool
	}{
		{"recorded an hour before actual start (pid recycled)", -time.Hour, true},
		{"recorded an hour after actual start", time.Hour, true},
		{"recorded just outside the slack window", procStartTimeSlack + time.Second, true},
		{"recorded exactly at the slack boundary", procStartTimeSlack, false},
		{"recorded exactly at the OS-reported start", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := isStale(lockInfo{PID: os.Getpid(), StartedAt: actual.Add(tc.skew)})
			if got != tc.stale {
				t.Fatalf("isStale(live pid, StartedAt=actual%+v) = %v, want %v", tc.skew, got, tc.stale)
			}
		})
	}
}

// TestStartTimesCorroborate proves the WR-02 slack-window comparison: times
// within procStartTimeSlack of each other (in either direction) corroborate,
// times further apart do not.
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
		{"recorded well after actual, beyond slack", base.Add(time.Hour), base, false},
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
