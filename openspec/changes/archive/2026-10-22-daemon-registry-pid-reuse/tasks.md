# Tasks

## 1. macOS process start time

- [x] 1.1 Add `internal/daemon/procstart_darwin.go` implementing `processStartTime` over `unix.SysctlKinfoProc("kern.proc.pid", pid)` and narrow `procstart_other.go` to `!linux && !darwin`; run `go mod tidy` so `golang.org/x/sys` becomes a direct requirement with no version change. Verify with a new darwin-tagged `TestProcessStartTimeDarwin` that the start time of the test's own pid is non-zero and not after `time.Now()`, and that a pid no process holds returns `ok=false`.
- [x] 1.2 Replace `requireAgedProcess` in `lock_test.go` with `requireProcessStartTime(t) time.Time` (skip only where `processStartTime` is unavailable, no wait), update its four callers in `lock_test.go` and `daemon_test.go`, and drop the "aged process" wording and the ~3 s effective-window comment from their names and doc comments. Verify `go test ./internal/daemon -run 'TestIsStale|TestAcquireWritesLock|TestSelfRecord|TestRunRegistersRecord' -v` on macOS shows the subtests running, not skipped, and `rg -n 'time.Sleep' internal/daemon/lock_test.go` finds nothing.

## 2. One-sided corroboration

- [x] 2.1 Change `startTimesCorroborate` to `actual.Sub(recorded) <= procStartTimeSlack` and rewrite the doc comments on `lockInfo`, `isStale`, `procStartTimeSlack` and `startTimesCorroborate` for the one-sided rule and the v0.15.0 transition. Verify with `TestStartTimesCorroborate` (the "recorded well after actual" case now corroborates; the pid-reuse case still does not; new cases: recorded 5 s before actual corroborates, 6 s before does not) and `TestIsStaleStillDetectsPIDReuse` (the "+1h" and "+6 s" cases now live; the "-1h" case still stale; new "-6 s" case stale and "-5 s" case live).
- [x] 2.2 Add `TestAcquireKeepsLockWithWallClockStartedAt` in `lock_test.go`: a lockfile for the test's own pid with `StartedAt` an hour after its actual start makes `acquire` return `ErrLockLive` and leaves the file in place.
- [x] 2.3 In `TestAcquireWritesLockThatOutlivesItsOwnStalenessCheck`, `TestSelfRecordSurvivesStaleness` and `TestRunRegistersRecordThatSurvivesPruning`, assert the written `StartedAt` equals `processStartTime(os.Getpid())` exactly (where available) in addition to the existing liveness assertions, so a writer that regresses to `time.Now()` fails without any process aging.

## 3. Registry pruning

- [x] 3.1 In `List`, parse each `*.json` name as a pid; remove files whose name does not parse; for files whose body does not decode or whose `pid` differs from the name, remove them only when `isProcessLive` is false for the name's pid and otherwise skip them unlisted; skip non-`.json` entries untouched. Verify with new cases in `registry_test.go`: `TestRegistryListRemovesNonPidName` (`d28d43d85835d89e.json`), `TestRegistryListRemovesUndecodableRecord` and `TestRegistryListRemovesPidMismatch` (both named for a pid no process holds), `TestRegistryListKeepsMalformedUnderLivePid` (the test's own pid with a garbage body: not listed, file still present), `TestRegistryListLeavesForeignFile` (`notes.txt` survives).
- [x] 3.2 Extend `TestRegistryListPrunesStale` with a record for the test's own pid whose `StartedAt` is an hour before the actual start, and assert it is removed on macOS and Linux (skip only where `processStartTime` is unavailable).
- [x] 3.3 Run `task test` and then `task lint` sequentially (the daemon suite is timing-sensitive under concurrent load) and verify both pass.

## 4. Changelog

- [x] 4.1 Add the fragment: `task changie -- new -k Fixes -b "codegraph daemon no longer lists a stale record whose pid was reused by another process, prunes malformed registry files, and corroborates daemon start times on macOS" --interactive=false` and verify the file appears under `.changes/unreleased/`.

## 5. Review fixes

- [x] 5.1 Add `validPID` (1 through `math.MaxInt32`) and make `isProcessLive` return false for any pid it rejects, so a record or lockfile naming 4294967295 or 4294967296 is stale and never reaches `kill(2)`; drop the now-redundant `pid <= 0` check from `List`'s name rule. Verify with `TestIsProcessLive` and `TestRegistryListRemovesOutOfRangePid`, and confirm both fail with the guard removed.
- [x] 5.2 Indirect `isStale` and `selfStartedAt` through a `startTimeOf` package var and add `TestIsStaleWithoutStartTimes`, so the "Start times are unavailable" scenario runs on Linux and macOS.
- [x] 5.3 Replace `TestProcessStartTimeDarwin` with `TestProcessStartTime` (Linux and macOS): a child's start lies between the instants around its start, and a dead pid yields ok=false.
- [x] 5.4 Add `TestRecycledPidLockIsCleared` (acquire replaces, and Unlock removes, a lock for a live pid stamped an hour before its start); extend `TestRegistryListKeepsMalformedUnderLivePid` with a body naming another pid; stamp `TestSelfRecordSurvivesStaleness` through `selfStartedAt`; rename the `TestStopAll` duplicate-file subtest and assert the second file is removed.
- [x] 5.5 Rewrite the doc comments and failure messages the review found inaccurate or narrating release history so they state current behaviour only.
- [x] 5.6 After the second review: add this process's negated process group id to `TestIsProcessLive` and the `TestStopAll` range subtest, so `validPID`'s lower bound has a case Go's own pid checks do not cover; add `TestAcquireWithoutStartTimeKeepsItsLock` for the `selfStartedAt` fallback; give `TestRegistryListKeepsMalformedUnderLivePid` and `TestRegistryListLeavesForeignFile` a positive assertion; record the seam and the pid range in design.md (D4, D5) and proposal.md. Verify each new case fails against a mutant that removes what it guards.
- [x] 5.7 Run `task test` and then `task lint` sequentially and verify both pass.
