# Proposal: Prune registry records whose pid belongs to another process

Issue: #146

## Why

`codegraph daemon` lists a daemon as running whenever a process with the recorded pid is alive. On macOS that is the whole check: the start-time corroboration that catches a recycled pid exists only on Linux, so on 2026-10-20 a registry of 886 records kept seven whose pids now belonged to unrelated programs and listed four of them as running daemons from August and September. A record whose file name is not a pid was never pruned either.

## What Changes

- Process start times are read on macOS as well as Linux, so a live pid is corroborated against the record's `startedAt` on both platforms, for the registry and for the `.codegraph/daemon.lock` lockfile alike. Records and locks written by this binary on macOS stamp the OS start time, as they already do on Linux.
- The corroboration rule becomes one-sided: a record is stale when the live process started later than the recorded time plus the slack. A recorded time that is later than the process start is kept. A recycled pid always starts after the original's record was written, so nothing real is lost, and the records and locks that v0.15.0 wrote on macOS with the wall clock at register time stay valid, so a new binary never clears a live older daemon's lock.
- Listing the registry removes a `.json` file whose name is not `<pid>.json`, whose content does not decode, or whose `pid` field differs from its name. Files without the `.json` suffix are left alone.
- `codegraph daemon` and `codegraph daemon stop` then see only daemons that are actually running. Their output shapes do not change.

## Capabilities

### New Capabilities

- `daemon-registry`: which daemon records the registry treats as live, how a live pid is corroborated against its record, and what happens to records that are stale or malformed. The registry's location stays in `user-directories`.

### Modified Capabilities

None.

## Out of scope

- Windows. Native Windows was dropped; WSL2 is Linux.
- A background reaper. Pruning stays on the same listing call, as D-05 decided.
- Cleaning the maintainer's registry by hand. The next `codegraph daemon` run prunes it.
- Changing the slack or the lockfile and record formats.

## Impact

- `internal/daemon`: a new `procstart_darwin.go` using `golang.org/x/sys/unix.SysctlKinfoProc` (the module is already an indirect dependency and becomes direct), `procstart_other.go` narrowed to `!linux && !darwin`, `startTimesCorroborate` in `lock.go` and `List` in `registry.go`.
- Tests: the corroboration cases in `lock_test.go`, `TestRegistryListPrunesStale` and new malformed-file cases in `registry_test.go`. `requireAgedProcess` stops skipping on macOS, so the lock-lifetime tests run there too.
- Changelog fragment, kind Fixes.
