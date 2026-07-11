package daemon

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// writeLock writes a lockfile for pid directly, bypassing acquire, so
// tests can set up a specific (live or dead) pid without needing a real
// daemon process running.
func writeLock(t *testing.T, dir string, pid int) {
	t.Helper()
	data, err := json.Marshal(lockInfo{PID: pid, StartedAt: time.Now().UTC()})
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
	if isStale(lockInfo{PID: os.Getpid(), StartedAt: time.Now()}) {
		t.Fatal("isStale: reported the current live process as stale")
	}

	dead := deadPID(t)
	if !isStale(lockInfo{PID: dead, StartedAt: time.Now()}) {
		t.Fatalf("isStale: reported dead pid %d as live", dead)
	}
}
