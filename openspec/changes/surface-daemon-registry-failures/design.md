# Design: Report the daemon registry entries a listing skips or cannot remove

## Context

`List` in `internal/daemon/registry.go` reads every `*.json` in the registry directory. It removes stale and malformed files on the same call, and returns only live records. It has three silent paths:

- Every removal is `_ = os.Remove(path)`.
- Every `ReadFile` error is treated as "the file vanished".
- A malformed `<pid>.json` under a live pid is skipped, and nothing reports it.

`StopAll` and `StopMatching` in `stop.go` start from `List`, so `daemon stop` inherits all three paths. The only callers of `List`, `StopAll` and `StopMatching` are in `internal/cli/daemon.go` and `internal/cli/tui/daemonpicker.go`. Each caller reaches the functions through a package-level func var, so tests can replace them.

`isStale` in `lock.go` decides on liveness alone when `startTimeOf` returns `ok=false`. It does not check liveness again after the failed read. On macOS, `processStartTime` maps every `unix.SysctlKinfoProc` error to `ok=false`. `SysctlKinfoProc` returns `EIO` when the kernel returns a short buffer, and the kernel does that for a pid no process holds. Every other error comes from `sysctl(3)` itself, for example a denied read under a sandbox.

`internal/daemon` is in the closure of `TestNoStdoutNoiseInServeReachablePackages`. Its production code must never write to stdout or call `log.SetOutput`. The package already logs through `log.Printf` (`daemon.go`). The stdlib logger writes to stderr.

## Goals / Non-Goals

**Goals:**

- Each silent path in the proposal gets one visible signal. The signal is not repeated within one process.
- Stdout, exit status and the record format stay the same.

**Non-Goals:**

- No new logging framework or log levels. The package keeps `log.Printf`.
- No change to which records count as live, except in the exit race (D4).

## Decisions

### D1. `List` returns the paths it skipped as a second result

`List` becomes `func List() ([]Record, []string, error)`. `StopAll` and `StopMatching` become `([]Record, []string, error)` and pass `List`'s skipped paths through unchanged.

A skipped path is a `<pid>.json` that `List` left in place without listing it because a process holds its pid. Such a file does not decode, names another pid, or cannot be read. Files that `List` failed to remove are not skipped paths. `List` had already judged each of them stale or malformed:

- its pid is dead,
- its pid is live but belongs to a recycled process (start-time corroboration failed), or
- its name is not a pid.

None of them is a running daemon that `List` could not act on. Their removal failures are logged instead (D3).

Alternatives considered:

- A `Listing` struct. It is the same data with more churn at every call site.
- A typed non-fatal error. Every current caller returns any non-nil error, so the listing would fail, and a later caller could make the same mistake.
- A package-level "last skipped" value. It is global state, and nothing ties it to the listing that set it.

### D2. The CLI prints the warning; `internal/daemon` only logs

`internal/cli` prints the warning to `cmd.ErrOrStderr()` with the existing `warning:` prefix (`githooks.go`, `index.go`). The line names the count and the full path of each skipped file, joined by `, `:

```text
warning: skipped 1 daemon registry file that names a running process but is not a valid record: /home/u/.local/state/codegraph/daemons/4242.json
```

- Bare `codegraph daemon` prints the warning after the plain list, or after the picker returns. A line written before the picker starts could be overwritten by the inline renderer or hidden by the alternate screen.
- `daemon stop` prints the warning after its `stopped pid …` or `no running daemon…` lines.
- The picker's stop action runs a second listing through `StopAll` or `StopMatching`. The picker can stay open for minutes, so that listing can skip a file the first one did not. `RunDaemonPicker` therefore returns the stop action's skipped paths: its signature becomes `([]string, error)`, and `dispatchDaemonAction` returns them as well. The bare command joins them with its own listing's paths. It keeps the first listing's order, adds each new path once, and prints one line.

The decision about what to print belongs to the CLI. `internal/daemon` is also reached from `serve --mcp`, where only stderr logging is safe.

### D3. Failures are logged once per key, per process

A small helper in `internal/daemon` keeps a mutex-guarded set of keys. It logs through `logf` only the first time it sees a key. `logf` is a package-level func var set to `log.Printf`, so tests can capture the output. That follows the package's func-var seam convention (`startTimeOf`, `stopSignal`) and leaves the global logger alone. Keys:

- A failed removal uses the file's path: `daemon: removing registry file <path>: <err>`.
- The macOS start-time read uses one fixed key (D5).

One `codegraph daemon` run can list the registry twice: the bare listing, then the picker's stop-all. Without the set, the same failure would print twice. A long-lived caller would print it on every listing.

`sync.Once` was rejected because it cannot key by path. A test helper resets the set with `t.Cleanup`, as `resetWSLCacheForTests` does in `internal/watch`.

### D4. `isStale` checks liveness again after a failed start-time read

When `startTimeOf` returns `ok=false`, `isStale` returns `!isProcessLive(pid)` and does not return `false` at once. The check sits in `isStale`, not in the macOS reader, so it covers all of these:

- A Linux process whose `/proc/<pid>/stat` vanished.
- The macOS `EIO`.
- The `procstart_other.go` stub.
- Registry records and the lockfile path (`acquire`, `Unlock`).

The cost is one extra `Signal(0)`, paid only when the read fails.

A process can still exit after the second check. That window is the same one every liveness check has, and `stopTargets` checks `isStale` again before it signals.

### D5. macOS logs every sysctl error except `EIO`

`processStartTime` tests `errors.Is(err, unix.EIO)`.

- `EIO` stays silent, because it means no process holds the pid.
- Any other error goes through the once-helper (D3): `daemon: reading process start times: <err>; a recycled pid cannot be detected`. One line per process is enough, because the cause, a denied sysctl, applies to every pid alike.

### D6. An unreadable `<pid>.json` is handled like one that does not decode

`List` treats only `errors.Is(err, fs.ErrNotExist)` from `ReadFile` as the concurrent-removal race. Any other read error takes the malformed-record path:

- The file is removed when `isProcessLive(pid)` is false.
- Otherwise the file is skipped and reported (D1).

Removing it is safe when the pid is dead. A dead pid's record is stale whatever its content (D-05), so nothing is lost by not reading it.

### D7. Test inputs that work as any user

- **Failed removal:** `os.Remove` in `List` goes through a `removeFile` func var. A test can then fail a removal with `EACCES`. A read-only directory would not fail the removal for root, and development containers often run as root.
- **Unreadable record:** the test writes `<pid>.json` as a symlink to a directory.
  - Its `DirEntry` is a symlink, so `List` does not skip it as a directory.
  - `ReadFile` fails with `EISDIR` on Linux and macOS, with no permission change.
  - Removing it removes the link, not the target.
- **Exit race (D4):** the test writes a record for a `liveProcess` helper child and replaces `startTimeOf` with a stub. The stub kills and waits for the child, then returns `ok=false`, and it counts its calls. `List` must then remove the record and not list it, and the stub must have run exactly once.
  - The count proves the race happened. Without it, a child that exits before the first liveness check would pass the test with no re-check in place.
  - The stub must wait for the child, not only kill it. A child that is killed but not waited for stays a zombie, and `Signal(0)` still reports a zombie live.
  - There is no sleep.
- **Denied macOS read (D5):** a darwin-only test replaces `kinfoProc` with a stub that returns `EPERM`, writes a valid record for the test's own pid, and lists the registry twice. Both listings must list the record, and the captured log must hold one line. Testing through `List` checks the outcome the spec scenario states, not only `processStartTime`'s return value.

## Risks / Trade-offs

- [The exported signatures of `List`, `StopAll` and `StopMatching` change] → The package is internal, its only callers are in `internal/cli` and `internal/cli/tui`, and this change updates them. CLI and MCP output is the compatibility surface. That output does not change.
- [Stderr gains a line that a script does not expect] → The line appears only when a skipped file exists, which is already an abnormal state. Stdout and the exit status do not change.
- [A malformed file under a recycled pid keeps the warning on every run until that process exits] → D3 of the #146 change does not allow removal while the pid is alive. The warning names the file so the user can remove it.
- [Dedup per process hides a repeat of the same failure in a long-lived process] → No long-lived process lists the registry today, and the first failure is always logged.
- [`log.Printf` adds the stdlib date and time prefix to CLI stderr] → The existing `daemon:` log lines have the same prefix.

## Migration Plan

No migration. The record and lockfile formats do not change. To roll back, revert the change.
