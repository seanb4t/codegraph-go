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
// residual PID-reuse false-negative risk in containers (T-04-07-02, A3,
// WR-02): isStale cross-checks a live pid's OS-reported actual start time
// (processStartTime — /proc/<pid>/stat on Linux; unavailable elsewhere)
// against StartedAt, so a pid recycled by an unrelated process after the
// original daemon crashed is caught even though Signal(0) alone reports it
// as live. Falls back to liveness-only wherever the OS corroboration data
// isn't available.
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

// isStale reports whether info's owning process is dead — or, when OS
// corroboration data is available, whether a live pid appears to have been
// reused by an unrelated process since the lockfile was written (WR-02).
// isProcessLive alone cannot distinguish "the original daemon is still
// running" from "the original daemon crashed and its pid was recycled by
// the OS for a different process" — both look identical to Signal(0). On
// platforms where processStartTime can read the OS's own record of when
// pid actually started (Linux today, via /proc), a mismatch against the
// StartedAt this daemon itself recorded at acquire time is a strong
// PID-reuse signal and is treated as stale despite passing the liveness
// check. Corroboration is skipped (never a false-positive-stale) whenever
// the data is unavailable — a different platform, missing /proc access, or
// any parse failure — falling back to the liveness-only check documented
// on lockInfo.
func isStale(info lockInfo) bool {
	if !isProcessLive(info.PID) {
		return true
	}
	if actualStart, ok := processStartTime(info.PID); ok {
		if !startTimesCorroborate(info.StartedAt, actualStart) {
			return true
		}
	}
	return false
}

// procStartTimeSlack is the tolerance for comparing info.StartedAt (our own
// time.Now().UTC() wall-clock reading at acquire time) against the OS's
// derived process-start wall-clock time (itself computed from a
// boot-relative monotonic tick count plus a whole-second boot time) —
// generous enough to absorb second-level rounding and scheduling jitter
// between the two independent clock sources without masking a genuine
// PID-reuse (which, in practice, is separated from the original process's
// start by at least the time it took the original process to crash and the
// OS to recycle the pid — far larger than this slack).
const procStartTimeSlack = 5 * time.Second

// startTimesCorroborate reports whether recorded and actual plausibly refer
// to the same process instance, within procStartTimeSlack.
func startTimesCorroborate(recorded, actual time.Time) bool {
	delta := recorded.Sub(actual)
	if delta < 0 {
		delta = -delta
	}
	return delta <= procStartTimeSlack
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
//
// CR-02: the initial attempt uses exclusive creation, not a read-then-
// write, so two processes racing acquire() for the very first lockfile can
// never both "win" — exactly one create succeeds; the other observes
// EEXIST and falls through to the ordinary read-and-check-staleness path
// below (which itself remains racy against a THIRD concurrent acquirer
// within the narrow stale-remove-then-recreate window — closing that
// window entirely would need a filesystem-level lock, out of scope for
// this fix; this closes the specific "nothing exists yet" TOCTOU the
// finding describes).
func acquire(codegraphDir string) error {
	data, err := json.Marshal(lockInfo{PID: os.Getpid(), StartedAt: time.Now().UTC()})
	if err != nil {
		return err
	}

	if err := createLockExclusive(codegraphDir, data); err == nil {
		return nil
	} else if !os.IsExist(err) {
		return err
	}

	// A lockfile already existed at the moment of the O_EXCL attempt above
	// — read it and apply the same staleness check acquire always has.
	info, ok, err := readLock(codegraphDir)
	if err != nil {
		return err
	}
	if !ok {
		// Vanished between the O_EXCL failure and this read (another
		// process's release() raced us) — nothing to remove; fall through
		// to the retry create below.
	} else if !isStale(info) {
		return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
	} else if err := os.Remove(lockPath(codegraphDir)); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := createLockExclusive(codegraphDir, data); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%w: another process acquired the lock during stale-lock recovery — try again", ErrLockLive)
		}
		return err
	}
	return nil
}

// createLockExclusive stages codegraphDir's lockfile so it can only ever be
// observed either fully-written or not-yet-existing — never partially
// written (CR-02).
//
// A plain os.OpenFile(..., O_EXCL) create-then-Write is NOT sufficient
// here: O_EXCL makes the CREATE step exclusive (only one racing caller's
// open succeeds), but the file exists — empty — for the whole interval
// between that open and the subsequent Write completing. A concurrent
// LOSING racer's readLock() can land in that interval and observe a
// present-but-empty file, failing with a JSON decode error instead of the
// EEXIST-driven staleness path acquire() expects (this was caught by
// TestAcquireConcurrentRaceOnlyOneWinner flaking under -race). Instead,
// this writes the complete payload to a uniquely-named temp file in the
// same directory (so an os.Rename fallback would be same-filesystem, if
// ever needed) and then os.Link()s it into place: Link is atomic with
// respect to both existence (fails EEXIST if the destination already
// exists — the same exclusivity O_EXCL gave us) and content (the linked
// name always resolves to the temp file's already-fully-written inode; a
// concurrent reader can never observe a partial write through it).
func createLockExclusive(codegraphDir string, data []byte) error {
	tmp, err := os.CreateTemp(codegraphDir, ".daemon.lock.tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // best-effort; a no-op once Link below moves past this point's need for it

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Link(tmpPath, lockPath(codegraphDir)); err != nil {
		return err
	}
	return nil
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
