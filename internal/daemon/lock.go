// Package daemon implements SYNC-04/SYNC-05/D-05: the long-lived local
// process (or in-process fallback) that owns a repo's watcher plus the
// single GraphStore.Writer so multiple agent sessions share one indexer.
// A pid+start-timestamp lockfile in .codegraph/ guards the single-writer
// invariant (INDX-05); Unlock clears ONLY a genuinely-stale lock, never a
// live daemon's.
//
// internal/daemon depends only on internal/indexer and internal/watch —
// indexer.Sync owns its own GraphStore.Open/Close/Writer lifecycle
// internally, so this package never imports internal/graphstore or Pebble
// directly.
package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// lockFileName is the daemon lockfile within .codegraph/ (D-05).
const lockFileName = "daemon.lock"

// ErrLockLive is returned by acquire and Unlock when the lockfile names a
// still-live process — the lock must never be silently cleared out from
// under a running daemon (T-04-07-01).
var ErrLockLive = errors.New("daemon: lock is held by a live process")

// lockInfo is the lockfile's JSON payload: pid + start timestamp
// (RESEARCH Pattern 6). StartedAt corroborates pid liveness to reduce the
// residual PID-reuse false-negative risk in containers (T-04-07-02, A3) —
// documented, not eliminated: v1's isStale does not read the OS's own
// process-start-time (e.g. /proc/<pid>/stat on Linux) to cross-check a
// live pid against StartedAt; it records this daemon's own wall-clock
// start so a future corroboration pass has the data available without a
// lockfile-format change.
type lockInfo struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
}

// lockPath returns the daemon lockfile's path under codegraphDir.
func lockPath(codegraphDir string) string {
	return filepath.Join(codegraphDir, lockFileName)
}

// readLock reads and decodes the lockfile at codegraphDir. ok is false
// (with a nil error) when no lockfile exists — the caller's "nothing to
// do" case, distinguished from a genuine read/decode error.
func readLock(codegraphDir string) (info lockInfo, ok bool, err error) {
	data, err := os.ReadFile(lockPath(codegraphDir))
	if err != nil {
		if os.IsNotExist(err) {
			return lockInfo{}, false, nil
		}
		return lockInfo{}, false, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return lockInfo{}, false, fmt.Errorf("daemon: decoding lockfile %s: %w", lockPath(codegraphDir), err)
	}
	return info, true, nil
}

// isStale reports whether info's owning process is dead.
func isStale(info lockInfo) bool {
	return !isProcessLive(info.PID)
}

// isProcessLive checks pid liveness via os.FindProcess + Signal(0)
// (RESEARCH Pattern 6, POSIX). Isolated in its own function — v1 targets
// POSIX only; a future Windows build-tag variant (an OpenProcess-based
// check) replaces just this function, not the surrounding acquire/Unlock
// logic (RESEARCH Environment Availability).
func isProcessLive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		// No such process — definitely stale (RESEARCH Pattern 6).
		return false
	}
	// On POSIX, FindProcess always succeeds; liveness is proven by
	// signal 0 (no-op signal, delivery-checked but never delivered).
	// ESRCH (no such process) or EPERM-on-a-dead-pid are both treated as
	// dead, matching the RESEARCH skeleton.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

// acquire creates the daemon lockfile at codegraphDir, recording the
// current process's pid + start time. If a live lock already exists,
// acquire fails with ErrLockLive without touching it (T-04-07-01 — never
// stomp a live daemon). If a stale lock is found (a crashed daemon), it is
// cleared first so a new daemon can start.
func acquire(codegraphDir string) error {
	info, ok, err := readLock(codegraphDir)
	if err != nil {
		return err
	}
	if ok {
		if !isStale(info) {
			return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
		}
		if err := os.Remove(lockPath(codegraphDir)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	data, err := json.Marshal(lockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		return err
	}
	return os.WriteFile(lockPath(codegraphDir), data, 0o644)
}

// release removes the daemon lockfile unconditionally. Only the process
// that successfully acquired the lock calls release, on clean shutdown —
// it is not the stale-only removal Unlock implements.
func release(codegraphDir string) error {
	err := os.Remove(lockPath(codegraphDir))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// Unlock implements `codegraph unlock`'s engine (SYNC-05): it removes the
// daemon lockfile at codegraphDir ONLY when it is genuinely stale
// (T-04-07-01). An absent lockfile is a clean no-op; a live lock is left
// untouched and reported via ErrLockLive. The returned message is a
// human-readable summary of what happened, for a thin CLI layer (Plan
// 04-08) to print verbatim.
func Unlock(codegraphDir string) (string, error) {
	info, ok, err := readLock(codegraphDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return fmt.Sprintf("no lock present at %s — nothing to do", lockPath(codegraphDir)), nil
	}
	if !isStale(info) {
		return "", fmt.Errorf("%w: pid=%d — daemon still running, stop it first", ErrLockLive, info.PID)
	}
	if err := os.Remove(lockPath(codegraphDir)); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return fmt.Sprintf("removed stale lock (pid=%d)", info.PID), nil
}
