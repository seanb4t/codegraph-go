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
	"math"
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
// (RESEARCH Pattern 6). isStale corroborates a live pid against StartedAt
// (T-04-07-02, WR-02), so a pid the OS recycled for an unrelated process
// after the original daemon died is caught where the OS reports start times
// (see isStale), even though Signal(0) reports it live. StartedAt is what
// selfStartedAt returns.
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

// isStale reports whether info's owning process is dead, or is alive but
// started more than procStartTimeSlack after info.StartedAt, which isStale
// treats as the OS having recycled the pid (WR-02); Signal(0) alone cannot
// tell those apart. Where processStartTime cannot read the start time (a
// platform other than Linux and macOS, or any read failure), liveness alone
// decides: "can't corroborate" is never itself a staleness signal.
func isStale(info lockInfo) bool {
	if !isProcessLive(info.PID) {
		return true
	}
	if actualStart, ok := startTimeOf(info.PID); ok {
		if !startTimesCorroborate(info.StartedAt, actualStart) {
			return true
		}
	}
	return false
}

// startTimeOf is processStartTime indirected behind a package-level func var,
// the same test-seam convention as stopSignal, so tests can make start times
// unavailable on a platform that reports them.
var startTimeOf = processStartTime

// procStartTimeSlack is how much later than the recorded StartedAt a live
// process may have started and still corroborate the record. It absorbs the
// whole-second rounding of Linux start times. A pid recycled within this
// slack of the original record's StartedAt is not detected.
const procStartTimeSlack = 5 * time.Second

// startTimesCorroborate reports whether a process that started at actual can
// own a record stamped recorded: it started no more than procStartTimeSlack
// after recorded. A recorded time later than actual MUST corroborate, by any
// amount, or a live daemon whose lock carries a wall-clock stamp loses it to
// a second writer (INDX-05).
func startTimesCorroborate(recorded, actual time.Time) bool {
	return actual.Sub(recorded) <= procStartTimeSlack
}

// selfStartedAt returns the StartedAt this process records in any lockfile or
// registry record naming its own pid: its start time as the OS reports it, the
// same value isStale later reads for that pid. Where the OS reports no start
// time it returns time.Now().UTC(), which is later than the start and so
// corroborates.
func selfStartedAt() time.Time {
	if actual, ok := startTimeOf(os.Getpid()); ok {
		return actual
	}
	return time.Now().UTC()
}

// validPID reports whether pid is in the range of a process id. kill(2) takes
// a 32-bit pid_t and truncates a wider value, so 4294967295 would signal every
// process the user owns (pid -1) and 4294967296 the caller's process group
// (pid 0).
func validPID(pid int) bool {
	return pid > 0 && pid <= math.MaxInt32
}

// isProcessLive checks pid liveness via os.FindProcess + Signal(0)
// (RESEARCH Pattern 6, POSIX). A pid validPID rejects is never live and is
// never signaled.
func isProcessLive(pid int) bool {
	if !validPID(pid) {
		return false
	}
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
	data, err := json.Marshal(lockInfo{PID: os.Getpid(), StartedAt: selfStartedAt()})
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
		return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph daemon unlock` once it has exited", ErrLockLive, info.PID)
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

// Unlock implements `codegraph daemon unlock`'s engine (SYNC-05; moved
// under daemon in Phase 3 VERB-04): it removes the daemon lockfile at
// codegraphDir ONLY when it is genuinely stale
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
