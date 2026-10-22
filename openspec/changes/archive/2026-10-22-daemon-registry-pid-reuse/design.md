# Design

## Context

`isStale` (`internal/daemon/lock.go`) decides staleness for both the lockfile and the registry: dead pid, or a live pid whose OS-reported start time does not corroborate the recorded one. `processStartTime` has a Linux implementation over `/proc/<pid>/stat` and a stub for every other platform that returns `ok=false`, which makes corroboration a no-op on macOS, the maintainer's platform and the one the issue was observed on. `selfStartedAt` stamps the OS start time where available and `time.Now()` otherwise, so every record and lock written on macOS so far carries a wall-clock time taken somewhere in the startup path, after the process actually started.

`startTimesCorroborate` is symmetric today: recorded and actual must be within `procStartTimeSlack` (5 s) in either direction. `List` skips a record that does not decode but leaves the file in place, and reads every entry in the directory whatever its name.

The one CGo dependency is tree-sitter; the daemon package must stay pure Go.

## Goals / Non-Goals

**Goals:**
- Corroboration works on macOS with no CGo and no new module.
- A binary from this change never treats a live daemon from v0.15.0 as stale, on either platform.
- The registry directory holds only well-formed `<pid>.json` records after a listing.

**Non-Goals:**
- Verifying that a live pid is a codegraph process by name. Start-time corroboration is what the lockfile already relies on, and it needs no process-name heuristics.
- Changing the slack, the lock protocol or the record format.

## Decisions

### D1. macOS start time through `sysctl kern.proc.pid`

`procstart_darwin.go` calls `unix.SysctlKinfoProc("kern.proc.pid", pid)` from `golang.org/x/sys/unix` (v0.48.0, already in `go.mod` as an indirect dependency) and converts `Proc.P_starttime` (a `Timeval`) to a `time.Time`. An error or a zero start time returns `ok=false`, the same contract as Linux: never a false claim of corroboration. `procstart_other.go` becomes `//go:build !linux && !darwin`.

Alternative: `ps -o lstart= -p <pid>`. Spawns a process per record on every listing, parses locale-dependent text and still needs a seam for tests. Rejected.

Alternative: CGo `proc_pidinfo`. Breaks the one-CGo-dependency constraint. Rejected.

### D2. One-sided corroboration

`startTimesCorroborate(recorded, actual)` returns `actual.Sub(recorded) <= procStartTimeSlack`. A recycled pid's process always starts after the original process wrote its record, so `actual` is later than `recorded` by at least the original's remaining lifetime; the one-sided test catches it exactly as the symmetric one did. The other direction, `recorded` later than `actual`, is what every v0.15.0 record and lock on macOS looks like (`time.Now()` at register time, after process start), and what a Linux record from before `selfStartedAt` looked like. Keeping those live means a new binary cannot clear the lock of a v0.15.0 daemon that is still writing, which would otherwise break the single-writer invariant (INDX-05) for as long as both binaries are installed. The memory store records two daemon registries live on the maintainer's machine until the next release, so this transition is real.

In `TestStartTimesCorroborate` and `TestIsStaleStillDetectsPIDReuse`, every case where the record is later than the actual start ("recorded an hour after actual start", "recorded just outside the slack window" at +6 s) flips to "corroborates"; the pid-reuse cases (record earlier than the actual start by more than the slack) keep their expected result, and two new cases pin the rejecting boundary: a record 5 s before the actual start corroborates, 6 s before does not. The doc comments on `lockInfo`, `isStale`, `procStartTimeSlack` and `startTimesCorroborate` are rewritten for the one-sided rule.

Alternative: keep the symmetric rule and accept a one-release window in which a new binary can clear a live old daemon's lock on macOS. Rejected: the failure is silent and corrupts the thing the lock protects.

### D3. Prune malformed files by name, decode and pid, never under a live pid

`List` parses each `*.json` name as an integer pid before reading it. A name that does not parse is removed at once with the same best-effort `os.Remove` the stale path uses: `Register` only ever writes `<pid>.json`, so no writer can be racing for that name. A body that does not decode, or a `pid` field that differs from the name, is removed only when no process holds the pid in the name; while that pid is alive the file is skipped, not listed, and reconsidered on the next listing. That keeps `List` from deleting a record that a daemon with that pid wrote between the read and the remove, an interleaving atomic writes do not prevent (`fsatomic` makes a record all-or-nothing; it does not make read, validate and remove one step). The window that remains, a dead pid recycled between the liveness check and the remove, is the one the stale path has today and is not widened. Non-`.json` entries are skipped without removal, so a stray file someone drops in the directory is not destroyed.

### D4. One test seam, and no aging wait

`processStartTime` keeps its build-tagged implementations. `isStale` and `selfStartedAt` call it through a `startTimeOf` package var, the package's existing test-seam convention (`stopSignal`), so a test can make start times unavailable and run the "Start times are unavailable" scenario, and the writer's wall-clock fallback, on Linux and macOS. `requireAgedProcess` and its wall-clock wait are removed: it existed so that a `time.Now()` fixture or writer would be rejected deterministically under the symmetric rule, and under the one-sided rule a later-than-actual time never fails corroboration, so no amount of aging reveals a regression. Its callers take a new `requireProcessStartTime(t) time.Time` that returns the test's own OS-reported start or skips where `processStartTime` is unavailable. Pid reuse is reached through `isStale` with a `StartedAt` skewed earlier than that start; the writer side (`acquire`, `Register` through `Run`) is checked by asserting the written `StartedAt` equals `processStartTime(os.Getpid())` exactly, which holds because `selfStartedAt` reads the same OS value for the same pid and a `time.Now()` regression never matches it. `TestProcessStartTime` checks each implementation directly on Linux and macOS: a child process's start lies between `time.Now()` readings taken just before and just after it starts (opening two seconds earlier on Linux, where `btime` and the tick count are each truncated to whole seconds), and a pid no process holds returns `ok=false`.

### D5. A pid outside the range of a process id is never live

`isProcessLive` returns false for a pid outside 1 through `math.MaxInt32` before it calls `os.FindProcess`. `kill(2)` takes a 32-bit `pid_t`, so a wider value read from a record or a lockfile is truncated: 4294967295 becomes pid -1 (every process the user owns) and 4294967296 becomes pid 0 (the caller's process group), and a negative value names a process group outright. Every path that reaches a signal or `processStartTime` with a pid from disk (`List`, `isStale`, `acquire`, `Unlock`, `stopTargets`) passes through `isProcessLive` first, so one check covers them all, and `List`'s name rule only needs the name to be the canonical form of an integer.

## Risks / Trade-offs

- [`SysctlKinfoProc` reports a start time for a zombie] → `isProcessLive` runs first; a zombie that still answers `Signal(0)` is listed today too and dies with its parent's `wait`. No change in exposure.
- [The OS-derived start time differs between write and check] → Both sides come from the same OS source for the same pid, so for the writer process the delta is zero; the slack absorbs second-level rounding, as on Linux today.
- [A registry file someone named `12345.json` by hand with a live pid] → Listed as today, and it also has to carry a corroborating start time. Not a new exposure.
- [A malformed file sits under a pid that stays alive for days] → It is never listed and costs one decode per listing; it goes when the pid does.

## Migration Plan

None beyond the binary swap. The first `codegraph daemon` run with the new binary prunes the stale and malformed records; v0.15.0 daemons that are still running stay listed and keep their locks.
