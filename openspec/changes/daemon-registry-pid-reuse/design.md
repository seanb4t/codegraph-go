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

The symmetric cases in `TestStartTimesCorroborate` and `TestIsStaleStillDetectsPIDReuseOnAgedProcess` ("recorded an hour after actual start") flip to "corroborates"; the pid-reuse cases keep their expected result. The doc comments on `lockInfo`, `isStale`, `procStartTimeSlack` and `startTimesCorroborate` are rewritten for the one-sided rule.

Alternative: keep the symmetric rule and accept a one-release window in which a new binary can clear a live old daemon's lock on macOS. Rejected: the failure is silent and corrupts the thing the lock protects.

### D3. Prune malformed files by name, decode and pid

`List` parses each `*.json` name as an integer pid before reading it. A name that does not parse, a body that does not decode, or a `pid` field that differs from the name is removed with the same best-effort `os.Remove` the stale path uses, and the file is skipped. Non-`.json` entries are skipped without removal, so a stray file someone drops in the directory is not destroyed. Records are written atomically through `fsatomic`, so a half-written record never carries the final name and this rule cannot race a concurrent `Register`.

### D4. No new seam

`processStartTime` stays a plain function with build-tagged implementations. Tests reach the corroboration branch through `requireAgedProcess`, which now runs on macOS instead of skipping, and through `isStale` with a skewed `StartedAt` on the test's own pid. The darwin implementation gets one direct test: the start time of `os.Getpid()` is non-zero and not after `time.Now()`, and a pid no process holds returns `ok=false`.

## Risks / Trade-offs

- [`SysctlKinfoProc` reports a start time for a zombie] → `isProcessLive` runs first; a zombie that still answers `Signal(0)` is listed today too and dies with its parent's `wait`. No change in exposure.
- [The OS-derived start time differs between write and check] → Both sides come from the same OS source for the same pid, so for the writer process the delta is zero; the slack absorbs second-level rounding, as on Linux today.
- [A registry file someone named `12345.json` by hand with a live pid] → Listed as today, and it also has to carry a corroborating start time. Not a new exposure.

## Migration Plan

None beyond the binary swap. The first `codegraph daemon` run with the new binary prunes the stale and malformed records; v0.15.0 daemons that are still running stay listed and keep their locks.
