# Tasks

## 1. macOS process start time

- [ ] 1.1 Add `internal/daemon/procstart_darwin.go` implementing `processStartTime` over `unix.SysctlKinfoProc("kern.proc.pid", pid)` and narrow `procstart_other.go` to `!linux && !darwin`; run `go mod tidy` so `golang.org/x/sys` becomes a direct requirement with no version change. Verify with a new darwin-tagged `TestProcessStartTimeDarwin` that the start time of the test's own pid is non-zero and not after `time.Now()`, and that a pid no process holds returns `ok=false`.
- [ ] 1.2 Verify `requireAgedProcess` no longer skips on macOS: `go test ./internal/daemon -run 'TestIsStale|TestAcquireWritesLock' -v` shows the corroboration subtests running, not skipped.

## 2. One-sided corroboration

- [ ] 2.1 Change `startTimesCorroborate` to `actual.Sub(recorded) <= procStartTimeSlack` and rewrite the doc comments on `lockInfo`, `isStale`, `procStartTimeSlack` and `startTimesCorroborate` for the one-sided rule and the v0.15.0 transition. Verify with `TestStartTimesCorroborate` (the "recorded well after actual" case now corroborates; the pid-reuse case still does not) and `TestIsStaleStillDetectsPIDReuseOnAgedProcess` (the "+1h" case now live; the "-1h" and "just outside the slack" cases still stale).
- [ ] 2.2 Add `TestAcquireKeepsLockWithWallClockStartedAt` in `lock_test.go`: a lockfile for the test's own pid with `StartedAt` an hour after its actual start makes `acquire` return `ErrLockLive` and leaves the file in place.

## 3. Registry pruning

- [ ] 3.1 In `List`, parse each `*.json` name as a pid, remove files whose name does not parse, whose body does not decode, or whose `pid` differs from the name, and skip non-`.json` entries untouched. Verify with new cases in `registry_test.go`: `TestRegistryListRemovesNonPidName` (`d28d43d85835d89e.json`), `TestRegistryListRemovesUndecodableRecord`, `TestRegistryListRemovesPidMismatch`, `TestRegistryListLeavesForeignFile` (`notes.txt` survives).
- [ ] 3.2 Extend `TestRegistryListPrunesStale` with a record for the test's own pid whose `StartedAt` is an hour before the actual start, and assert it is removed on macOS and Linux (skip only where `processStartTime` is unavailable).
- [ ] 3.3 Run `task test` and then `task lint` sequentially (the daemon suite is timing-sensitive under concurrent load) and verify both pass.

## 4. Changelog

- [ ] 4.1 Add the fragment: `task changie -- new -k Fixes -b "codegraph daemon no longer lists a stale record whose pid was reused by another process, prunes malformed registry files, and corroborates daemon start times on macOS" --interactive=false` and verify the file appears under `.changes/unreleased/`.
