# Design

## Context

PR #374 added `query.IndexDirs(root)` in `internal/query/confine.go`. It resolves `<root>/.codegraph` and `<root>/.codegraph/store` with `realpath.Resolve`. When either one resolves outside `root`, it returns `*IndexOutsideRootError`. A failure to resolve, such as a link loop or `EACCES`, returns a plain wrapped error. Every command calls `IndexDirs` at start, through `ResolveIndex`, `indexDirs` in `internal/cli/init.go`, or `daemon.New`.

`daemon.New` (`internal/daemon/daemon.go`) stores the resolved paths in `Daemon.codegraphDir` and `Daemon.storeDir`. After that, the daemon uses the path strings and never resolves them again. `Daemon.Run` writes through them in three places:

- `acquire(codegraphDir)` reads `daemon.lock`. It removes a stale lock and writes its own. `RunWithRetry` calls `Run` again after each `ErrLockLive`, so a `serve` watcher that defers to a live daemon repeats this step, possibly for days.
- Each debounced `flush` calls `touchPending`, then `indexer.Sync(repoRoot, storeDir, opts)`, then `clearPending`.
- The deferred `release(codegraphDir)` calls `os.Remove` on `<codegraphDir>/daemon.lock`.

A resolved path such as `/r/.codegraph/store` contains no link at the time of `New`. When Git later replaces `/r/.codegraph` with a link, the same string goes through the link at the OS level.

`indexer.Sync` never swaps a store. Only a full index calls `graphstore.ReplaceStore`, and a watcher never runs a full index.

`serve` runs `daemon.RunWithRetry` in a goroutine inside `serveWatchStart` and prints any error that is not `ErrLockLive`, a cancel or `*watch.DisabledError` as `codegraph serve: watcher: <err>` on stderr. The MCP server keeps running. `daemon start` returns the error of `Run` from its `RunE`. `cmd/codegraph/main.go` prints any returned error on stderr and exits 1.

The refusal scenarios at start already have tests from #374. `serve refuses to start` has none.

## Goals / Non-Goals

**Goals:**

- A watcher writes no byte of the index, the pending sidecar or a lock file through a path that resolves differently from its start.
- The watcher stops with an error that says why, and the caller prints it through its existing error path.

**Non-Goals:**

- A watcher does not restart itself on the new paths. The user restarts it.
- No change to the check that commands run at start.
- No change for a `.codegraph` that disappears while the watcher runs, for example after `codegraph uninit`. `IndexDirs` then returns the same paths as at start.

## Decisions

### D1. One check, `checkIndex`, before the lock and before each flush

`Daemon` keeps its start paths as one field, `start query.IndexPaths`, in place of `codegraphDir` and `storeDir`. A new method `(*Daemon).checkIndex() error` calls `query.IndexDirs(d.repoRoot)`:

- On any error, refused or not, it returns that error unchanged. So `errors.Is(err, query.ErrIndexOutsideRoot)` still works for a refusal.
- When the result differs from `d.start` (struct equality), it returns `*IndexChangedError`.
- Otherwise, it returns `nil`.

`Run` calls `checkIndex` after the watch-policy gate and before `acquire`. On an error, `Run` returns it, and so does `RunWithRetry`, because the error is not `ErrLockLive`. The debounce callback calls `checkIndex` before `flush`. On an error, it calls `stopWith` (D3) and does not call `flush`. So `flush`, `touchPending`, `Sync` and `clearPending` run only after a passed check.

Alternatives:

- A watch on `.codegraph` that stops the watcher on a change to the directory. Rejected for this change. The recursive watcher skips `.codegraph`, an event can arrive late or never, and a check before the write is still needed.
- Open the store and the index directory once as file descriptors and write relative to them (`openat`). Rejected. Pebble opens its files by path through its VFS, so this needs a custom VFS or a change upstream.
- Stop only on a refusal, and log any other resolve error. Rejected: a link loop or `EACCES` then logs on every flush while the watcher writes nothing useful.

### D2. Stop on any change, not only on a refusal

A link that now resolves inside the repository passes `IndexDirs`. The watcher still stops. The paths the watcher holds are the resolved paths of its start. After a change, they no longer name the index that commands open: a path such as `<repo>/.codegraph/store` now passes through the new link. A write through them goes where the change points. Stopping keeps one writer for each index, and a restart costs one command. A change of the store alone counts as a change.

Alternative: switch to the new paths and continue. Rejected, because the switch must move the lock and the sidecar while another process can start.

### D3. A typed error and one stop path in `Run`

`internal/daemon` adds `IndexChangedError{Was, Now query.IndexPaths}` and the sentinel `ErrIndexChanged`. Its message names each path that differs, `Dir`, `Store` or both, at start and now, and says `restart the watcher`.

`Run` makes its child context with `context.WithCancelCause` in place of `context.WithCancel`. A stop calls `cancel(err)`, and the first cause wins. The debounce callback calls `cancel(err)` on a failed `checkIndex`. The branch where the watch loop exits on its own calls `cancel(ErrWatcherClosed)`, so the `runErr` variable goes away and `Run` has one stop path. `startWatchdog` gets a function that calls `cancel(nil)`. After the join, `Run` returns `context.Cause(ctx)`, or `nil` when the cause is `context.Canceled`. `Run` also returns `nil` when the cause is the cause of the parent context, for example `context.DeadlineExceeded` or a cause that the caller set. Before this change, `Run` returned `nil` whenever its parent ended, and a stop that the caller asked for is not an error. A later watch on `.codegraph` can call the same `cancel` without a move of code.

A requeue for `ErrStoreLocked` stays as it is. Any other `Sync` error stays a logged error, and the watcher continues.

### D4. Remove the lock only when it is this process's lock inside the repository

The deferred release in `Run` calls a new `releaseOwnLock`. It resolves `<repoRoot>/.codegraph` alone with the new `query.IndexDir`, the first half of `IndexDirs`. `IndexDir` returns `*IndexOutsideRootError` for a result outside the repository, so a refusal has one error shape everywhere. When it is, it reads the lock with `readLock`. Only when `lockInfo.PID` equals `os.Getpid()` does it remove the file. In every other case, it logs the path and leaves the file.

The decision uses `.codegraph` alone, because the lock lives there. It accepts any directory inside the repository, not only the start directory. When the user moves `.codegraph` to `.cg-real` and links it, the watcher's own lock moves too. Leaving it would be wrong: under `serve`, it names the live `serve` pid, so `isStale` keeps it, and `daemon start` and `daemon unlock` refuse it until `serve` exits. A change of the store alone leaves `.codegraph` in place, so the own lock is removed in that case too.

The path check refuses a `.codegraph` that now leads to another project, even when the lock there names this pid by chance. The owner check matches `Unlock`, which also reads the lock before it removes it. It guards this sequence: someone removes the watcher's lock from outside while the watcher runs, for example by hand or through `uninit` and `init`, and another daemon then takes a new lock in the same directory.

`Deregister` removes the registry record under the user's state directory, not under `.codegraph`, so it does not change.

### D5. Tests through the existing seams, and two new seams in `internal/cli`

The daemon tests use `newFakeDaemon` with a `fakeWatcher` (`internal/daemon/fakewatcher_test.go`) and the `syncFn` seam. A test replaces `.codegraph` or the store with a link between start and a fake event. It then checks the error of `Run`, that `syncFn` never ran, and that the target directory is unchanged. No test runs Git: the scenario starts after the replacement that Git makes.

`internal/cli` gets two package-level function variables, in the style of `daemonStopMatching`:

- `runDaemonWithRetry`, which `serveWatchStart` calls in place of `daemon.RunWithRetry`.
- `runDaemon`, which `daemon start` calls in place of `d.Run`.

A test sets either one to return an error. It does not need a real watcher.

### D6. A property for D2

D2 stops the watcher on any inequality, so `IndexDirs` must return equal results for a tree that did not change. A table test in `internal/query` checks this for every pair of states that a pull can leave `.codegraph` and `store` in: absent, a real directory, a link inside, a link outside, a dangling link and a link to itself. That is 36 trees at most, so a table covers each one, and a random generator adds nothing.

## Risks / Trade-offs

- [The check and the open are not atomic] A link can appear after the check and before `Sync` opens the store. It can also appear while `Sync` runs. → The window shrinks from the life of the watcher to one sync, usually milliseconds. If a real attack needs it, `openat` from D1 can close the window.
- [`clearPending` runs after `Sync`] It removes `.sync-pending` through the start path. A link that appears during `Sync` makes it remove that file in the target. → Same window as above. The file is an empty marker.
- [`acquire` after the check] The same window exists between `checkIndex` and the writes of `acquire`. It covers all of `acquire`: a read of the process start time, a temporary file, a hard link and, for a stale lock, a read and a removal. → It is a few system calls long. A check inside `acquire` before each write can shorten it if needed.
- [A stopped watcher in `serve`] The MCP server keeps running with no auto-sync until a restart. → The stderr line says why. Tool calls already refuse a link that leads outside, so no tool reads the other index.
- [Behavior change for a move inside the repository] A user who moves the index on purpose while a watcher runs now sees the watcher stop. → The error says to restart it.

## Migration Plan

None. The change adds a check to a running process and stores nothing new.
