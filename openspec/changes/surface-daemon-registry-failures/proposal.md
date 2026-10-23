# Proposal: Report the daemon registry entries a listing skips or cannot remove

Issue: #155

## Why

Some daemon registry failures leave no trace. A listing ignores some entries, and some removals fail with no message. `codegraph daemon stop --all` can report success while a daemon gets no signal. The suite review of the #146 change found these paths. They follow design decision D3 of that change, so they are gaps, not regressions.

Issue #155 describes the exit race only on the macOS start-time path. The liveness re-check goes in `isStale`. Every platform and the lockfile use that function, so the re-check covers Linux and the lockfile too.

## What Changes

- A listing treats only "file does not exist" from reading a `<pid>.json` as a race with a concurrent removal. Any other read error is treated like a record that does not decode: the file is removed when no process holds its pid. Otherwise it is skipped.
- A listing that cannot remove a registry file logs the path and the error to stderr. The cause is anything except the file being gone already. Each path is logged at most once per process, so a file that can never be removed is not logged on every retry.
- `codegraph daemon` and `codegraph daemon stop` print one warning line to stderr when the listing skipped `<pid>.json` files under a live pid. The line names those files. Stdout and the exit status do not change.
- When a process start time cannot be read, liveness is checked again before liveness alone decides. A daemon that exits during the read is then stale, not live. This applies on Linux and macOS, for registry records and for the `.codegraph/daemon.lock` lockfile.
- On macOS, a start-time read that fails with any error except `EIO` is logged to stderr once per process. `EIO` means no process holds the pid. A denied read, for example under a sandbox, turns pid-reuse detection off for every record, and the log line makes that visible.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `daemon-registry`: the start-time fallback checks liveness again after a failed read. macOS logs a start-time read failure other than `EIO`. An unreadable `<pid>.json` is handled like one that does not decode. Two new requirements say how a listing logs removals that fail and reports the entries it skips.

## Out of scope

- A non-zero exit status for skipped entries. `daemon stop` still exits non-zero only when a signal fails.
- `migrateLegacyRegistry`. Its removals stay best-effort and silent, as its doc comment says.
- Logging start-time read failures on Linux. `/proc` read failures for a process the user owns are not a known problem. The liveness re-check applies on Linux as well.
- Changes to the record or lockfile formats, the slack, or which records count as live.
- A reaper, or removal of a malformed record under a live pid. D3 of the #146 change still holds.

## Impact

- `internal/daemon`:
  - `registry.go`: `List` returns the paths it skipped, treats only not-exist as a read race, and logs failed removals.
  - `stop.go`: `StopAll` and `StopMatching` pass the skipped paths through.
  - `lock.go`: `isStale` checks liveness again after a failed start-time read.
  - `procstart_darwin.go`: logs a non-`EIO` sysctl error once.
  - A once-per-key log helper, plus `logf` and `removeFile` test seams.
- `internal/cli/daemon.go`: the bare command and `daemon stop` print the skipped-entries warning to stderr. The `daemonList`, `daemonStopAll` and `daemonStopMatching` seams take the new signatures.
- `internal/cli/tui/daemonpicker.go`: the `stopMatching` and `stopAll` seams take the new signatures. `RunDaemonPicker` and `dispatchDaemonAction` return the stop action's skipped paths, so the bare command can name them in its warning. The picker's own output does not change.
- Tests:
  - `registry_test.go`: unreadable files, failed removals and the skipped paths.
  - `lock_test.go`: a process that exits during the start-time read.
  - `stop_test.go`: a `TestStopAll` subtest for a skipped record.
  - `internal/daemon/daemon_test.go`, `lock_test.go` and `stop_test.go`: existing calls move to the new `List`, `StopAll` and `StopMatching` signatures.
  - `procstart_darwin_test.go`: the logged sysctl error.
  - `internal/cli/daemon_test.go`: the stderr warning and unchanged stdout.
  - `daemonpicker_test.go`: stub signatures.
- Changelog fragment, kind Fixes.
