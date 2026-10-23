# Tasks

## 1. Once-per-key logging

- [ ] 1.1 In `internal/daemon`, add a `logf` func var set to `log.Printf`. Add a once-per-key helper backed by a mutex-guarded set (design D3), and a test helper that resets the set and captures `logf` output with `t.Cleanup`. Verify with `TestLogOncePerKey`: the same key twice gives one line, and two keys give two lines.

## 2. Registry listing

- [ ] 2.1 Change `List` to `([]Record, []string, error)`. The second result holds the path of every `<pid>.json` that `List` leaves in place, unlisted, because a process holds its pid (design D1). Update `List`'s callers in `internal/daemon` tests. Verify:
  - `TestRegistryListKeepsMalformedUnderLivePid` asserts the skipped paths equal the left-in-place files.
  - `TestRegistryListRemovesUndecodableRecord`, `TestRegistryListRemovesPidMismatch`, `TestRegistryListRemovesNonPidName` and `TestRegistryListPrunesStale` assert no skipped paths.
- [ ] 2.2 In `List`, treat only `errors.Is(err, fs.ErrNotExist)` from `ReadFile` as the concurrent-removal race. Any other read error takes the malformed-record path (design D6). Verify:
  - `TestRegistryListRemovesUnreadableRecord`: `<dead pid>.json` is a symlink to a directory. The link is removed, the target directory still exists, and nothing is listed.
  - `TestRegistryListKeepsUnreadableUnderLivePid`: the same symlink named for the test's own pid is not listed, is still present, and is returned as skipped.
- [ ] 2.3 Route `List`'s removals through a `removeFile` func var set to `os.Remove` (design D7). Log every failure except not-exist once per path: `daemon: removing registry file <path>: <err>`. Verify with `TestRegistryListLogsFailedRemovalOnce`:
  - `removeFile` returns `EACCES` for a dead-pid record, and `List` runs twice. Neither listing lists the record or returns it as skipped, and the captured log holds exactly one line naming the path and the error.
  - A stub that returns `fs.ErrNotExist` logs nothing.

## 3. Stop

- [ ] 3.1 Change `StopAll` and `StopMatching` to `([]Record, []string, error)`. `stopTargets` passes `List`'s skipped paths through. Update the existing `StopAll` and `StopMatching` calls in `stop_test.go`. Verify with a new `TestStopAll` subtest: with only a malformed record under the test's own pid, nothing is signaled, the error is nil, and the skipped paths hold that file.

## 4. Liveness after a failed start-time read

- [ ] 4.1 In `isStale`, when `startTimeOf` returns `ok=false`, return `!isProcessLive(pid)` (design D4). Verify:
  - `TestRegistryListRemovesRecordWhoseProcessExitsDuringStartTimeRead`: the test writes a record for a `liveProcess` child. A `startTimeOf` stub counts its calls, kills and waits for the child, then returns `ok=false`. `List` does not list the record, its file is removed, and the stub ran exactly once (design D7).
  - `TestIsStaleWithoutStartTimes` still passes, so a live pid with no start time stays live.

## 5. macOS start-time read errors

- [ ] 5.1 In `procstart_darwin.go`, `EIO` from `kinfoProc` stays silent. Log any other error once per process through the once-helper: `daemon: reading process start times: <err>; a recycled pid cannot be detected` (design D5). Verify with darwin-tagged `TestRegistryListLogsDeniedStartTimeReadOnce` in `procstart_darwin_test.go`:
  - A `kinfoProc` stub returns `EPERM`, and the registry holds a valid record for the test's own pid. `List` runs twice. Both listings list the record, and the captured log holds exactly one line containing the error.
  - A stub that returns `EIO` logs nothing, and the record is still listed.
  - The test runs in the `daemon-darwin` workflow.

## 6. CLI warning

- [ ] 6.1 Move the `daemonList`, `daemonStopAll` and `daemonStopMatching` seams in `internal/cli/daemon.go`, and the `stopMatching` and `stopAll` seams in `internal/cli/tui/daemonpicker.go`, to the new signatures. `dispatchDaemonAction` and `RunDaemonPicker` return the stop action's skipped paths, and the `runDaemonPicker` seam takes the new signature (design D2). Update the stubs in `internal/cli/daemon_test.go` and `internal/cli/tui/daemonpicker_test.go`. Verify:
  - `TestDaemonPickerModel_StopAllDispatches` and `TestDaemonPickerModel_EnterDispatchesStopOne` assert that `dispatchDaemonAction` returns the stub's skipped paths.
  - `go test ./internal/cli/...` passes.
- [ ] 6.2 Add a helper in `internal/cli/daemon.go` that writes nothing for no skipped paths. Otherwise it writes the one `warning:` line from design D2 to `cmd.ErrOrStderr()`, with a singular and a plural form, and passes each path through `sanitizePathForDisplay`. Call it from the bare `daemon` command after the plain list or after the picker returns, joining both listings' paths without duplicates, and from both `daemon stop` branches after their own output. Verify in `internal/cli/daemon_test.go`:
  - `TestDaemonBareCmd_NonTTY_WarnsAboutSkippedRecords`: stdout equals the output without skipped files, and stderr holds one line naming both stubbed paths. One stubbed path holds an escape byte, and the line holds that path without it.
  - `TestDaemonStopCmd_All_WarnsAboutSkippedRecords`: stdout is `no running daemons`, stderr holds one line, and the error is nil.
  - `TestDaemonStopCmd_Matching_WarnsAboutSkippedRecords`: `daemon stop` without `--all`. Stdout is `no running daemon for <path>`, stderr holds one line naming every stubbed path, and the error is nil.
  - `TestDaemonBareCmd_InteractiveAllowed_WarnsAfterPicker`: stdout and stderr share one buffer. The listing stub skips path A, and the picker stub returns paths A and B. One warning line follows the picker stub's output and names A and B once each.
  - The existing bare and stop tests assert stderr is empty when nothing was skipped.

## 7. Verification

- [ ] 7.1 Check that each new test can fail. For each one, break the code it guards and confirm the test fails:
  - Restore `_ = os.Remove`.
  - Treat every read error as a race.
  - Drop the liveness re-check.
  - Log `EIO`.
  - Count a failed removal as a skipped path.
  - Drop the CLI warning call from each `daemon stop` branch and from the bare command.
  - Drop the picker's skipped paths from the bare command's warning.
  - Print a skipped path without `sanitizePathForDisplay`.
- [ ] 7.2 Run `GOTOOLCHAIN=go1.26.10 task test`, then `GOTOOLCHAIN=go1.26.10 task lint`, one after the other. Use the `go` line in `go.mod` as the toolchain. Verify both pass.
- [ ] 7.3 Run `openspec validate surface-daemon-registry-failures --strict` and verify it passes.

## 8. Changelog

- [ ] 8.1 Add the fragment: `task changie -- new --interactive=false -k Fixes -b "codegraph daemon and daemon stop warn on stderr about registry files they skip, log registry files they cannot remove and denied macOS start-time reads, and no longer treat a daemon that exits during the start-time read as running"`. Verify the file appears under `.changes/unreleased/`.

## Workflow follow-up

- The implementation PR says `Closes #155` and carries the code, the changelog fragment and the `openspec archive surface-daemon-registry-failures` result.
- Commit the archive first, or with nothing else unstaged, so hk's stash and restore leave no empty `openspec/changes/<name>/specs/*` directories.
