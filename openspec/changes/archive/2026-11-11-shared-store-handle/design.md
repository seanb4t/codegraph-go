# Design

## Context

See proposal.md for the motivation and the measured failure rate.

`internal/graphstore` is the one package that holds a Pebble handle. `open` in `pebble_store.go` calls `pebble.Open` for each `Open`, `OpenExisting` and `Checkpoint`. Pebble takes an exclusive lock on the store directory for the life of a handle. It keeps a registry of those locks for the process, keyed by the name of the lock file. A second `pebble.Open` with the same name fails with `lock held by current process`. `retryLocked` tries 5 times with 100 ms between attempts and then returns `ErrStoreLocked`.

A `serve --mcp` process opens the store from three places:

- `query.OpenAt` (`internal/query/engine.go`), once for each tool call. It takes one snapshot and reads only that snapshot.
- `indexer.Sync` (`internal/indexer/sync.go`), once for each watcher flush and once at startup. It holds the writer lock `<store>.write` and commits one batch.
- `wtindex` (`internal/wtindex/wtindex.go`), for a worktree index.

Two flock files beside the store stay as they are (`locks.go`). Each handle holds `<store>.swap` shared, and `ReplaceStore` takes it exclusive. One writer holds `<store>.write`.

`Daemon.Run` (`internal/daemon/daemon.go`) takes the daemon lock, opens the watcher, creates the debouncer and starts the watch loop. It runs no sync of its own. Each flush goes through `Daemon.flush`, which writes the `.sync-pending` marker, runs the sync and removes the marker on success. The debouncer fires only after an `Add`. `Run` already adds the sentinel `flushRetryPath` to queue a flush again after a lost lock.

## Goals / Non-Goals

**Goals:**

- A second open of a store in one process shares the handle of the first. It does not fail on the lock of that process.
- A reader that opens during a sync of the same process sees the graph of one completed sync.
- The lock rules between processes stay as they are.
- An edit made before the watcher is ready is synced with no further edit.

**Non-Goals:**

- A handle that stays open between users. That design keeps other processes out of the store.
- A shorter sync (#197).
- A change to the retry window or to `ErrStoreLocked`.
- A different result for a store that another process holds.
- One commit for a burst of edits. The watcher decides which edits one sync holds.

## Decisions

### D1: A table of shared handles in `graphstore`, counted by user

`graphstore` keeps a table for the process. The key is the absolute store path (D5). The value is one shared entry: the `*pebble.DB`, the shared swap lock, a count of users and a state. The state is opening, ready or closing.

- `Open`, `OpenExisting` and `Checkpoint` look in the table first. For a ready entry, the caller gets a new `pebbleStore` that points at the entry, and the count goes up by one. With no entry, the caller opens Pebble with the present bounded retry and puts the entry in the table.
- `Close` on a `pebbleStore` marks that one handle closed and lowers the count by one. At zero, it closes the Pebble handle and the swap lock and removes the entry.
- One entry type owns the `*pebble.DB` and the swap lock for both kinds of handle. A staging entry (D4) never enters the table and has no count: its one `Close` closes it. `Close` then has one release path, which skips the count for an entry outside the table.
- Each caller keeps its own `pebbleStore` value. `ErrClosed` and the idempotent `Close` stay rules of one caller. A caller that closes its handle does not close the handle of another caller.

Alternatives considered:

- A longer retry window. It makes the failure less frequent and keeps the defect. A sync on a large repository lasts longer than any window that is short enough for a tool call (#197).
- A gate in the process that makes a second open wait for the first. A tool call then waits for the whole sync. The agent gets a slow answer where a snapshot can answer at once.
- One handle that `serve` owns for the whole session and passes to the query engine and the daemon. It needs a new parameter through four packages: `query`, `indexer`, `daemon` and `mcp`. It also keeps every other process out of the store for the life of the session. The CLI, a second session and a full index all open the same store.

### D2: Each open is one `retryLocked` loop over the table

One mutex guards the table. `pebble.Open` and the Pebble close can take time, so the mutex covers neither call.

Each `Open`, `OpenExisting` and `Checkpoint` runs the table lookup as the attempt function of `retryLocked`, the present helper that owns the bounded retry. Each caller therefore has its own 5 attempts, as today. One attempt does one of four things:

- For a ready entry, it takes one count and succeeds.
- With no entry, it puts an entry in the opening state and calls `pebble.Open` one time, outside the mutex. On success the entry becomes ready. On failure the entry leaves the table, and the attempt returns the error.
- For an opening entry, it waits for that one `pebble.Open` call to end. It then takes one count, or it returns the error of that call. The wait is part of the attempt and lasts as long as one `pebble.Open` call.
- For a closing entry, it returns `ErrStoreLocked`.

`retryLocked` sleeps and tries again only for `ErrStoreLocked`, which `classifyOpenError` gives for a held Pebble lock and the table gives for a closing entry. Every other error returns at once. A caller that waits on the open of another caller uses its own attempts and never the remaining attempts of that caller. Its total wait has the bound of one open today.

The last close uses the closing state. `Close` sets the state under the mutex, closes Pebble and the swap lock outside the mutex, and then removes the entry. The entry leaves the table on every return path of that close, also when Pebble returns an error. A Pebble close waits for its compactions, so an open does not wait for it without a limit. The spec names this moment as one of two that keep the bounded retry. The other is the exchange of a full index (D4).

Three test seams run in the pattern of `checkpointHook`: `sharedOpenHook` while an entry is in the opening state, `sharedCloseHook` between the state change and the Pebble close, and `sharedWaitHook` before and after the wait of a caller on an opening entry. `internal/indexer` has one more, `syncBeforeCommitHook`, which runs just before the one commit of a sync.

Alternative considered: `golang.org/x/sync/singleflight`, which is already a dependency. `Do` gives each waiter the result of one call. It does not let a waiter take its count in the same step under the table mutex. An entry can then reach zero and close between `Do` and the count. `singleflight` also has no closing state. The entry with three states is one small type.

### D3: A reader of the shared handle sees one commit or the next

`query.OpenAt` reads through one Pebble snapshot. `indexer.Sync` writes one batch and commits it one time on each path: the batch that only refreshes file times, or the batch of the incremental sync. A snapshot taken before the commit shows the old graph. A snapshot taken after it shows the new graph. The exclusive lock gave this property by force. The spec scenario "A call during one sync" now states it.

The property is about one sync. A burst of edits can reach the graph in two syncs, because a flush can fire during the burst. A reader between those syncs sees the first part of the burst. That graph is the graph of one completed sync, and each file in it is whole.

The writer lock keeps one writer at a time. Two syncs in one process take `<store>.write` through two file descriptors, and flock excludes the second one. The shared handle adds no second writer.

A sync that needs the file-index backfill runs a full index. That path uses `OpenStaging` and `ReplaceStore` and is covered by D4.

### D4: `OpenStaging` and `ReplaceStore` stay outside the table

`OpenStaging` opens a store that one full index owns, with the write-ahead log off. Its options differ from those of `Open`, and no second user exists. It keeps its own handle.

`ReplaceStore` takes `<store>.swap` exclusive. A shared entry holds that lock shared through its own file descriptor. `ReplaceStore` therefore waits for the bounded retry and then returns `ErrStoreLocked`. It does the same today for an open handle. This holds inside one process and between processes. After the last user closes, the entry releases the swap lock and `ReplaceStore` can proceed.

A full index reads the metadata of the live store with `OpenExisting` before it builds. With a handle open in the same process, that read now shares the handle. The run then stops at `ReplaceStore` with `ErrStoreLocked`, and the live store stays unchanged. `TestIndexForceRefusesWhileStoreIsHeld` in `internal/cli` covers this path and keeps its in-process holder.

During the exchange of a full index, `ReplaceStore` holds the swap lock exclusive and the Pebble lock of the live store. Both are outside the table. An open in the same process during that short time runs the bounded retry, as it does today.

### D5: The key is the absolute path, and link spellings stay as they are

The table key is `filepath.Abs` of the store path, which also cleans it. A relative spelling and an absolute spelling of one store share one entry. The lock files and `pebble.Open` get the path as the caller gave it, as they do today. The first opener of an entry fixes that spelling for the life of the entry.

The table does not resolve symbolic links. Today two link spellings of one store give two Pebble handles on one directory in one process. Pebble keys its registry by the name it was given, and its file lock does not exclude a second lock from the same process. After this change the same two spellings give two table entries and the same two handles. This change keeps that behavior and adds no new failure to it.

Alternative considered: resolve symbolic links for the key, the lock files and `pebble.Open`. It needs a resolver for a path that does not exist yet, and it moves the lock files of a linked store directory away from the place that an earlier release uses. That is a second change with its own compatibility rules, and #277 does not need it.

### D6: The catch-up sync is one queued flush

After the watch loop starts, `Daemon.Run` adds the present sentinel `flushRetryPath` to the debouncer one time. The sentinel gets the neutral name `flushSentinelPath`, because it now has two uses. The flush that follows runs the ordinary sync. From the moment `watch.Open` returns, each later edit gives an event. The catch-up sync finds each edit made before that moment, because a sync compares the tree with the stored file records. That comparison skips a file whose size and modification time both match its record. An edit of that kind is out of reach of every sync, and the spec names the exception.

The debouncer gives the catch-up the behavior of every other flush: the `.sync-pending` marker, the requeue on `ErrStoreLocked`, the join on shutdown and the merge with real events of the same window.

Alternative considered: call the sync directly in `Run` before the watch loop. It delays the loop start and needs its own lock-requeue handling.

The standalone `codegraph daemon` command runs the same `Run`, so it gets the same catch-up sync.

Each start of `Run` now gives one sync with no event. A daemon test that counts syncs waits for that first sync before it writes. One test helper beside `waitWatcherOpened`, `waitCatchUpSync`, does that wait. `TestDaemonFlushLockRequeueGivesUpPerEpisode` is the exception. Its store is locked from the start, so the catch-up flush is the first attempt of its first episode, and the test counts it as such.

The catch-up flush has the requeue budget of every other flush. If another process holds the store through `maxFlushLockRequeues` requeues, the flush stops. The `.sync-pending` marker stays, and the next event starts a new sync. During the hold, each tool call returns the `isError` result of a locked store. After the other process releases the store, `codegraph_status` reports the graph as stale. The spec states this exception.

### D7: The live-edit client stops rewriting its file

`mcpLiveEdit` writes its anchor file again each 500 ms because the watcher can miss the first burst. D6 removes the miss. The client writes its files one time and then calls `codegraph_explore` until the symbol appears. If the catch-up sync does not run, the test fails.

### D8: A test holds the store through the Pebble lock, not through `Open`

Sixteen tests use an in-process `Open` as the holder that stands for another process. After D1 that `Open` shares the handle, and the code under test no longer meets a lock.

A new package, `internal/graphstore/locktest`, gets one helper, `HoldStoreLock(t, dir)`. It takes the Pebble directory lock with `pebble.LockDirectory` and fails the test if the lock is not free. Pebble keys its registry by the cleaned name that it was given, which is not made absolute. The helper therefore locks three names of `dir`, where they differ: the cleaned path, the absolute path, and the absolute path with its symbolic links resolved. On macOS a temporary directory has both spellings, and the code under test can use either one. It returns a release function and also releases in `t.Cleanup`. A `sync.Once` guards the release, because a second close of a Pebble lock panics.

`locktest` imports Pebble, and it does not import `graphstore`. The tests of package `graphstore` can therefore import it with no cycle, and the tests of `internal/uiserver`, `internal/wtindex` and `internal/mcp` use the same helper. The lint rule `graphstore-owns-pebble` allows Pebble in the subpackages of `internal/graphstore`. `storetest` stays a package for tests that compare stores.

The lock is outside the table, so an `Open` meets `lock held by current process` and follows the path of a store that another process holds.

A fixture in `internal/uiserver/livepublish_test.go` keeps its own retry loop around `graphstore.Open` for a collision with the live publisher of the same process. After D1 that open shares the handle, so the loop goes away.

The helper `requireUsable` in the `graphstore` tests proves that the process released a store, and it does so with an `Open`. It moves to `locktest.HoldStoreLock` for the same reason as the probe above. One helper beside `noRetrySleep` in `retry_test.go`, `countRetrySleeps`, swaps the `openLockRetrySleep` seam and returns the count. The seam is unexported, so only the `graphstore` tests count sleeps.

The tests that move to the helper:

- `internal/graphstore`: `TestOpenConvergesWhenHolderCloses`, `TestOpenSucceedsOnTheFinalAttempt`, `TestCheckpointLockedSource`.
- `internal/uiserver`: `TestStatusDegradesOnLock`, `TestNonStatusRPCsDegradeWhenAHolderNeverReleases`, `TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget`, `TestDegradedRPCOpensTheStoreExactlyOnce`, `TestDegradeIsIdempotent`, `TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete`, `TestGetHealthDoesNotDegrade`, `TestFileGraphDoesNotDegrade`, `TestFileSymbolsDoesNotDegrade`.
- `internal/wtindex`: `TestPrepareMainStoreLocked`.

`TestUIServerHoldsNoStoreHandleBetweenCalls` in `internal/uiserver` uses a second in-process `Open` as a probe for a leaked handle. After D1 that probe succeeds on a leaked handle too. It moves to `locktest.HoldStoreLock`, which fails the test on a handle that is still open. With it, seventeen tests change or are examined.

`TestReplaceStoreWaitsForOpenHandle` has a case that holds the store with a raw `pebble.Open`. That case moves to the helper too, so the `graphstore` tests have one way to stand for an outside holder.

`TestOpenSecondOpenInProcessReturnsErrStoreLocked` is replaced by a test of the new rule: a second open succeeds with no retry sleep. Two tests keep their in-process holder, because `ReplaceStore` still refuses a store with an open handle (D4): `TestIndexForceRefusesWhileStoreIsHeld` in `internal/cli` and `TestExchangeFailureKeepsStore` in `internal/indexer`.

## Risks / Trade-offs

- [Users that overlap chain one hold of the store with no gap, and other processes stay out for that time] → This behavior is new, and the design accepts it. Today a second user in the process fails and does not extend the hold. Each tool call holds its handle only for that call, and the watcher holds its handle only for one sync. No handle outlives its last user, and another process keeps its bounded retry and its `isError` result.
- [A later sync with two commits shows a partial graph to a reader] → D3 states the rule. A test in `internal/indexer` reads through the shared handle during a sync and rejects a partial graph.
- [The catch-up sync adds one tree walk at each watcher start] → It runs off the handshake path. A sync with no changes stops after the walk and the file-record comparison. #197 tracks the cost of a sync.
- [`codegraph_status` reports `stale` for a short time after each watcher start] → The flush writes `.sync-pending`. The marker says that a sync is queued, which is true. The flush removes it after the sync.
- [Under heavy load, a tool call meets the last close of a sync handle and that close lasts longer than the retry] → The retry is the one of today, and the hold is one Pebble close in place of one whole sync. The load test of task 5.2 measures the result.
- [A leaked handle keeps an entry and the Pebble lock for the life of the process] → The same leak holds the Pebble lock today. The count adds no new way to leak.
- [A reader holds a snapshot through a long sync and Pebble keeps the old data for that time] → A tool call is short. A snapshot is released at `Close`.
- [A full index builds its staging store before `ReplaceStore` refuses it] → This occurs only with a handle open in the same process. In `serve`, the backfill path of a sync runs a full index one time for a store of an earlier release. The background build of a worktree index also runs one, on its own seed store, which no tool call opens. The live store stays unchanged, and the daemon queues the flush again.

## Migration Plan

No data migration. The store format and the lock files do not change. A process of an earlier release that opens the same store still meets the Pebble lock and the bounded retry. Rollback is a revert of the change.
