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

A link that now resolves inside the repository passes `IndexDirs`. The watcher still stops. Its lock file and its pending sidecar are in the old directory. A second daemon started on the new path does not see that lock. Stopping keeps one writer for each index, and a restart costs one command. A change of the store alone counts as a change.

Alternative: switch to the new paths and continue. Rejected, because the switch must move the lock and the sidecar while another process can start.

### D3. A typed error and one stop path in `Run`

`internal/daemon` adds `IndexChangedError{Was, Now query.IndexPaths}` and the sentinel `ErrIndexChanged`. Its message names both paths and says `restart the watcher`.

`Run` declares one closure, `stopWith(err)`. It stores the first error it gets and calls the `cancel` of the child context that `Run` already makes. `Run` then joins as it does on a cancel and returns the stored error in place of `nil`. The debounce callback calls `stopWith` today. A later watch on `.codegraph` can call it without a move of code.

A requeue for `ErrStoreLocked` stays as it is. Any other `Sync` error stays a logged error, and the watcher continues.

### D4. Remove the lock only when it is this process's lock in the start directory

The deferred release in `Run` calls a new `releaseOwnLock`. It resolves `<repoRoot>/.codegraph` alone with `realpath.Resolve` and compares it with `d.start.Dir`. When they are equal, it reads the lock with `readLock`. Only when `lockInfo.PID` equals `os.Getpid()` does it remove the file. In every other case, it logs both paths and leaves the file.

The decision uses `.codegraph` alone, because the lock lives there. After a change of the store alone, the lock is still the watcher's own, and leaving it is wrong: under `serve`, it names the live `serve` pid, so `isStale` keeps it, and `daemon start` and `daemon unlock` refuse it until `serve` exits.

The owner check matches `Unlock`, which also reads the lock before it removes it. The path check alone is not enough. After the watcher stops, another process can take a stale lock in the same directory and rewrite it. The owner check refuses that lock.

`Deregister` removes the registry record under the user's state directory, not under `.codegraph`, so it does not change.

### D5. Tests through the existing seams, and two new seams in `internal/cli`

The daemon tests use `newFakeDaemon` with a `fakeWatcher` (`internal/daemon/fakewatcher_test.go`) and the `syncFn` seam. A test replaces `.codegraph` or the store with a link between start and a fake event. It then checks the error of `Run`, that `syncFn` never ran, and that the target directory is unchanged. No test runs Git: the scenario starts after the replacement that Git makes.

`internal/cli` gets two package-level function variables, in the style of `daemonStopMatching`:

- `runDaemonWithRetry`, which `serveWatchStart` calls in place of `daemon.RunWithRetry`.
- `runDaemon`, which `daemon start` calls in place of `d.Run`.

A test sets either one to return an error. It does not need a real watcher.

### D6. A property for D2

D2 stops the watcher on any inequality, so `IndexDirs` must return equal results for a tree that did not change. A `pgregory.net/rapid` property in `internal/query` checks this over random trees of links under `.codegraph` and `store`. It follows `TestResolve_AgreesWithTheOperatingSystem` in `internal/realpath/realpath_test.go`.

## Risks / Trade-offs

- [The check and the open are not atomic] A link can appear after the check and before `Sync` opens the store. It can also appear while `Sync` runs. → The window shrinks from the life of the watcher to one sync, usually milliseconds. If a real attack needs it, `openat` from D1 can close the window.
- [`clearPending` runs after `Sync`] It removes `.sync-pending` through the start path. A link that appears during `Sync` makes it remove that file in the target. → Same window as above. The file is an empty marker.
- [`acquire` after the check] The same window exists between `checkIndex` and `acquire` in `Run`. → It is one system call long.
- [A stopped watcher in `serve`] The MCP server keeps running with no auto-sync until a restart. → The stderr line says why. Tool calls already refuse a link that leads outside, so no tool reads the other index.
- [Behavior change for a move inside the repository] A user who moves the index on purpose while a watcher runs now sees the watcher stop. → The error says to restart it.

## Migration Plan

None. The change adds a check to a running process and stores nothing new.
