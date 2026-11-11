---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-11T16:50:10+00:00
head: ecc22e0725b1141c90741d90122b734ccf6d210e
lenses:
  spec-lint: not-applicable
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: partial
  invariants: ok
---
# Review

## Incomplete lenses

- test-honesty: partial, gaps: 1

The verdict covers only what the lenses checked.

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | D2 rejects singleflight because a waiter cannot take its count in the same step as the end of the open, so an entry can close between the open and the count. The chosen code has the same window. A waiter wakes on the opened channel outside the table mutex, locks again, and can find the entry of the opener already closing. The code handles that with errHandleClosing and one retry sleep, and a test pins that outcome. The first reason of the rejection is not a difference between the two designs. Only the second reason, the missing closing state, still holds. | design.md > D2: Each open is one `retryLocked` loop over the table — "It does not let a waiter take its count in the same step under the table mutex. An entry can then reach zero and close b" | Remove the first reason from the singleflight alternative of D2, or restate it. Base the rejection on the missing closing state, and say that both designs have the window between the end of the open and the count and handle it with a retry. |
| app-2 | approach | should-fix | D2 says that a caller that waits on an opening entry either takes one count or returns the error of that open. The code has a third outcome. When the opener closes its handle before the waiter locks the table again, the waiter returns errHandleClosing and uses one retry sleep of its own budget. design.md does not record this outcome. | design.md > D2: Each open is one `retryLocked` loop over the table — "For an opening entry, it waits for that one `pebble.Open` call to end. It then takes one count, or it returns the error " | Add to the opening-entry case of D2 that a waiter that finds the handle closing after the wait returns ErrStoreLocked through the closing state and retries within its own attempts. |
| fit-1 | fit | should-fix | setSyncBeforeCommitHook in package indexer re-does the generic setHook test helper that the same package already has in pipeline_test.go. | internal/indexer/sync_shared_test.go:16 — "func setSyncBeforeCommitHook(t interface{ Cleanup(func()) }, hook func()) {" | Widen the t parameter of setHook to an interface with Helper and Cleanup, so that *rapid.T fits, and call setHook(t, &syncBeforeCommitHook, ...) in place of setSyncBeforeCommitHook. |
| fit-2 | fit | should-fix | The change adds three seam setters in shared_test.go that differ only in the variable they set, where the repository already uses one generic setHook for that job. | internal/graphstore/shared_test.go:84 — "func setSharedOpenHook(t *testing.T, hook func()) {" | Add one setHook[T any](t, hook *T, value T) to the graphstore tests, in the shape of the indexer helper, and call it for sharedOpenHook, sharedWaitHook and sharedCloseHook. |
| fit-3 | fit | should-fix | runHeld in TestRunCatchUpSyncOnHeldStore repeats the awaitEpisode loop of TestDaemonFlushLockRequeueGivesUpPerEpisode: the injected ErrStoreLocked syncFn, the receive loop with a 10 s timer and the 1 s quiet gap. | internal/daemon/catchup_test.go:145 — "return indexer.Stats{}, fmt.Errorf("%w: injected contention", graphstore.ErrStoreLocked)" | Hoist one helper beside startFakeRun that receives want sync results, checks which are ErrStoreLocked, and then checks the quiet gap. Build its receive on the existing receive helper of testbudget_test.go. Call it from awaitEpisode and from runHeld. |
| fit-4 | fit | should-fix | TestOpenOfSecondPathDoesNotWait copies the body of the blockFirstOpen closure inline, because that closure is local to TestSharedHandleWaitersShareOpen. | internal/graphstore/shared_test.go:349 — "blockFirstOpen := func(t *testing.T) (inOpen <-chan struct{}, unblock func()) {" | Move blockFirstOpen to file level in shared_test.go and call it from TestOpenOfSecondPathDoesNotWait. |
| inv-1 | invariants | should-fix | The shared handle depends on each user closing its snapshots before its own Close, and nothing enforces it. A pebbleReader holds no tie to its pebbleStore, and a Close that is not the last returns nil, so a snapshot that one user leaks stays usable or breaks depending on whether another user is still open, and Pebble reports the leak on the Close of a different caller. | design.md > Risks / Trade-offs — "A tool call is short. A snapshot is released at `Close`." | Count open readers per pebbleStore (Snapshot adds one, pebbleReader.Close removes one) and make the Close of that pebbleStore report its own leaked reader, or state in D1 that a user closes its readers before Close. Add a test with two users in which one closes with an open snapshot and the error lands on that user, not on the last closer. |
| tes-1 | test-coverage | should-fix | Scenario 'A short lock' is proven at graphstore.Open level only; no tool call is shown returning a result without isError after a mid-retry release. | tasks.md > 3. Readers and the sync of one process — "`TestOpenConvergesWhenHolderCloses` covers the spec scenario "A short lock": the holder releases between two attempts, a" | Add to internal/mcp/held_store_test.go a test that holds locktest.HoldStoreLock, releases it from the openLockRetrySleep seam (or a timer shorter than the retry window) during a codegraph_explore call, and asserts result.IsError is false and the text contains Alpha. If the seam is unexported from graphstore, use a short timer. |
| ctl-lens | test-honesty | should-fix | lens test-honesty reported partial, gaps: 1 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A short lock | internal/graphstore/open_lock_test.go: TestOpenConvergesWhenHolderCloses | partial | pass |
| A long lock | internal/mcp/held_store_test.go: TestToolCallOnHeldStore | covered | pass |
| A sync of the same server longer than the retry window | internal/mcp/held_store_test.go: TestToolCallWhileProcessHasStoreOpen; internal/query/open_shared_test.go: TestOpenAtDuringSync | covered | pass |
| A call during one sync | internal/indexer/sync_shared_test.go: TestSyncReaderSeesWholeSync | covered | pass |
| Tool calls back to back during a live edit | test/integration/testdata/watch_live_sync.txtar via TestScripts (mcpclient live-edit) | covered | pass |
| An edit right after the handshake | internal/daemon/catchup_test.go: TestRunSyncsEditMadeBeforeWatchOpen; test/integration/testdata/watch_live_sync.txtar | covered | pass |
| An edit during a takeover | internal/daemon/soak_test.go: TestConvergenceTwoSessions | covered | pass |
| Nothing changed while the watcher started | internal/daemon/catchup_test.go: TestRunCatchUpSyncWithNoChange | covered | pass |
| Another process holds the store while the watcher starts | internal/daemon/catchup_test.go: TestRunCatchUpSyncOnHeldStore | covered | pass |

### Not run

None.

## Verification runs

- exit 0: `mise exec -- go test ./internal/graphstore -run '^TestOpenConvergesWhenHolderCloses$' -count=1 -v; # covers: internal/graphstore/open_lock_test.go: TestOpenConvergesWhenHolderCloses`
- exit 0: `mise exec -- go test ./internal/mcp -run '^TestToolCallOnHeldStore$' -count=1 -v; # covers: internal/mcp/held_store_test.go: TestToolCallOnHeldStore`
- exit 0: `mise exec -- go test ./internal/mcp -run '^TestToolCallWhileProcessHasStoreOpen$' -count=1 -v; mise exec -- go test ./internal/query -run '^TestOpenAtDuringSync$' -count=1 -v; # covers: internal/mcp/held_store_test.go: TestToolCallWhileProcessHasStoreOpen; internal/query/open_shared_test.go: TestOpenAtDuringSync`
- exit 0: `mise exec -- go test ./internal/indexer -run '^TestSyncReaderSeesWholeSync$' -count=1 -v; # covers: internal/indexer/sync_shared_test.go: TestSyncReaderSeesWholeSync`
- exit 0: `mise exec -- go test ./test/integration -run '^TestScripts/watch_live_sync$' -count=1 -v; # covers: test/integration/testdata/watch_live_sync.txtar via TestScripts (mcpclient live-edit)`
- exit 0: `mise exec -- go test ./internal/daemon -run '^TestRunSyncsEditMadeBeforeWatchOpen$' -count=1 -v; mise exec -- go test ./test/integration -run '^TestScripts/watch_live_sync$' -count=1 -v; # covers: internal/daemon/catchup_test.go: TestRunSyncsEditMadeBeforeWatchOpen; test/integration/testdata/watch_live_sync.txtar`
- exit 0: `mise exec -- go test ./internal/daemon -run '^TestConvergenceTwoSessions$' -count=1 -v; # covers: internal/daemon/soak_test.go: TestConvergenceTwoSessions`
- exit 0: `mise exec -- go test ./internal/daemon -run '^TestRunCatchUpSyncWithNoChange$' -count=1 -v; # covers: internal/daemon/catchup_test.go: TestRunCatchUpSyncWithNoChange`
- exit 0: `mise exec -- go test ./internal/daemon -run '^TestRunCatchUpSyncOnHeldStore$' -count=1 -v; # covers: internal/daemon/catchup_test.go: TestRunCatchUpSyncOnHeldStore`

## Dropped

None.

## Lens notes

- approach: Requirement 'A briefly locked store does not fail or hang a call' is produced by D1, D2 and D3: internal/graphstore/shared.go shares one handle for each filepath.Abs key, and indexer.Sync commits one batch on each path (sync.go lines 254-257 and 739-742).
- approach: Requirement 'An edit made while the watcher starts reaches the graph' is produced by D6: daemon.go line 389 adds flushSentinelPath after the watch loop goroutine starts. watch.Open adds every directory (fsnotify) or starts the FSEvents stream before it returns.
- approach: Requirement 'A watcher-start sync that cannot get the store leaves the graph marked stale' is produced by D6: flush writes .sync-pending before the sync and clears it only on success, the give-up branch never requeues, and query/status.go reports stale while the sidecar exists.
- approach: Verified citations: retryLocked 5 attempts and 100 ms backoff (pebble_store.go 95-96); the Pebble lock registry keyed by the cleaned lock-file name and the message 'lock held by current process' (pebble v2.1.7 vfs/file_lock_unix.go 48-49, open.go 1174, vfs.go 347); Pebble Close waits for compactions (db.go 1713); internal/flock uses flock(2); golang.org/x/sync is a direct dependency; the depguard rule graphstore-owns-pebble excludes internal/graphstore/**.
- approach: D4 verified: fullindex.go reads metadata with OpenExisting (line 201) before OpenStaging and ReplaceStore; ReplaceStore takes the swap lock exclusive with the bounded retry; TestIndexForceRefusesWhileStoreIsHeld keeps its in-process holder.
- approach: D7 and D8 verified: mcpLiveEdit writes its files one time with no ticker; locktest.HoldStoreLock locks the cleaned, absolute and symlink-resolved names with sync.OnceFunc and t.Cleanup; the livepublish fixture opens once; every test that D8 lists exists and uses the helper.
- approach: Every risk mitigation names a task or a test that exists at head: TestSyncReaderSeesWholeSync and TestSyncSnapshotsAreWholeSyncs in internal/indexer/sync_shared_test.go, and task 5.2 for the load test. The worktree background build runs on its own seed directory, as the last risk says.
- approach: The migration plan has no deployed configuration to roll back; a revert is the rollback. design.md has no open questions.
- fit: The prompt names no path for repository library rows, and openspec/config.yaml holds none, so only the generic rows of the rulebook apply.
- fit: The production components fit their closest patterns: openShared runs as an attempt of the existing retryLocked with the alreadyClassified classifier, the three shared seams follow checkpointHook, and D2 names golang.org/x/sync/singleflight with a reason.
- fit: D6 reuses the existing flush sentinel and the debouncer requeue path instead of a second sync path, and waitCatchUpSync is built on the existing receive helper.
- fit: locktest is the one Pebble lock holder for tests: every remaining pebble.LockDirectory call outside graphstore production code is in internal/graphstore/locktest/locktest.go.
- fit: The proposal names no next change, so no forward-misfit check applies. #197 is out of scope and touches the sync cost, not the handle table.
- fit: Without git, the lens cannot tell new lines from old ones in a changed file. The inline time.After selects of TestRunUsesInjectedWatcher in fakewatcher_test.go may be older than this change, so they are not reported.
- invariants: I1: One process has at most one table entry and one Pebble handle per absolute store path \| enforced at internal/graphstore/shared.go acquireShared under sharedHandles.mu, key from filepath.Abs in openShared \| boundary test TestOpenSecondOpenInProcessSharesHandle, TestOpenSpellingsShareHandle (relative, ./, a/.., doubled and trailing separator, absolute)
- invariants: I2: users of a ready entry equals the count of open pebbleStore values on it, and a second Close of one pebbleStore never lowers it \| enforced at pebbleStore.Close closed.Swap and dbHandle.release \| boundary test TestSharedHandleClose (second Close returns nil, table still holds 1), TestSharedHandleConcurrentOpenClose, TestSharedHandleParallelOpenClose
- invariants: I3: An entry leaves the table on every return path of the last close, also on a Pebble close error \| enforced at shared.go release deferred delete \| boundary test TestOpenDuringLastCloseRetries case 'the close returns an error' (leaked iterator, then zero retry sleeps)
- invariants: I4: A closing entry takes no new user and gives ErrStoreLocked \| enforced at acquireShared errHandleClosing \| boundary test TestOpenDuringLastCloseRetries cases 1 and 2 (sleeps == openLockRetryAttempts-1)
- invariants: I5: A waiter on an opening entry never takes a count on a failed open, and returns the error of that open \| enforced at acquireShared openErr check after <-h.opened \| boundary test TestSharedHandleWaitersShareOpen (non-lock error, held-lock error, opener closes before waiter looks)
- invariants: I6: Each open has exactly openLockRetryAttempts attempts of its own, shared across swap lock, Pebble lock and closing state \| enforced at retryLocked with alreadyClassified over acquireShared \| boundary test TestSharedHandleWaiterKeepsOwnBudget, TestOpenDuringLastCloseRetries case 4, TestOpenSucceedsOnTheFinalAttempt (release before attempt 5, ordinal asserted)
- invariants: I7: A staging handle never enters the table and has one owner \| enforced at open(dir, true) with key empty \| boundary test TestStagingStoreHasOneOwner
- invariants: I8: ReplaceStore refuses while any shared user is open, inside one process \| enforced at locks.go ReplaceStore exclusive swap flock against the shared flock of the entry (internal/flock uses unix.Flock, per open file description) \| boundary test TestReplaceStoreWaitsForSharedHandle (one of two closed: refused; both closed: succeeds)
- invariants: I9: An incremental Sync commits one batch on each path, so a snapshot sees the graph of one completed sync \| enforced at internal/indexer/sync.go by structure only (one NewWriter and one Commit per path, no bounded writer) \| boundary test TestSyncReaderSeesWholeSync (concurrent poll plus hook before commit), TestSyncSnapshotsAreWholeSyncs (rapid, hookRuns <= 1)
- invariants: I10: One writer at a time per store, also inside one process \| enforced at graphstore.LockWriter flock before Open in Sync \| boundary test unchanged by this change (not re-examined)
- invariants: I11: Each Run gives exactly one catch-up flush with no event, and it shares the requeue budget of maxFlushLockRequeues \| enforced at daemon.go deb.Add(flushSentinelPath) after the watch goroutine starts \| boundary test runCatchUpSync asserts one sync; TestRunCatchUpSyncOnHeldStore asserts maxFlushLockRequeues+1 locked syncs then none, and 1 locked then success
- invariants: I12: flushSentinelPath never equals a watcher path \| enforced at the NUL byte in the constant, which no file path can hold \| boundary test none, none needed
- invariants: I13: A store that another process holds still gives isError with the store-lock text \| enforced at retryLocked and classifyOpenError \| boundary test TestToolCallOnHeldStore (held, then released)
- invariants: No enforcement point of this change is a script; all are Go code, so no probe ran.
- invariants: No weakened check found: the same-process string arm of isLockHeldOS stays pinned, by TestHoldStoreLockExcludesOpen with noRetrySleep in place of the removed TestOpenSecondOpenInProcessReturnsErrStoreLocked.
- invariants: Every production caller read (indexer.Sync, query.OpenAt engineCloser, wtindex seedUsable, readStoreMeta) closes its snapshot before its store today, so inv-1 is a latent invariant and not a present defect.
- test-coverage: Task 6.1 (changelog fragment) is unchecked in tasks.md; outside scenario coverage.
- test-coverage: The 0 of 72 load-test result in task 5.2 is a recorded run, not a repository test.
- test-honesty: Read in full or in part and found honest: indexer/sync_shared_test.go, mcp/held_store_test.go, query/open_shared_test.go, daemon/catchup_test.go, graphstore/hold_lock_test.go, shared_test.go lines 1-345. Each has assertions that touch the spec THEN.
- test-honesty: Grep of test files found no new Skip, disabled test or `\|\| true` in the change's diff files. The one Skip in a diff file, uiserver/filegraph_test.go:262, is an opt-in measurement gated on CODEGRAPH_GUAVA_STORE.
- test-honesty: Minor, no finding: TestSharedHandleConcurrentOpenClose discards the result of nodeVisible and writes no record, so its 'each open user still reads' check only fails on a snapshot or read error. The check is a liveness probe, not a data check.
- test-honesty gap: Not read: internal/uiserver tests, internal/wtindex/prepare_test.go, internal/cli/index_lock_test.go, remaining daemon tests (daemon_test, soak, testbudget, fakewatcher), integration test and txtar, graphstore tests other than shared_test.go and hold_lock_test.go, shared_test.go lines 345 onward, sync_property_test.go.

## Plan review (2026-11-11)

Verdict approve-with-notes at head 1e016455708823caf88c64b9da858cfd40e19748.

### Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1+inv-2 | approach,invariants | should-fix | app-1: D2 describes an open at head as one retryLocked loop of 5 attempts around one pebble.Open. At head, open first takes the shared swap flock through lockBeside, which runs a retryLocked loop of its own. D1 puts the swap lock in the shared entry, but D2 does not say whether a new entry takes that flock inside the table attempt or with which retry. Its bounds (a waiter waits one pebble.Open, a caller has one open's budget) are therefore not shown to hold, also not for the exchange moment that the spec keeps on the bounded retry.; inv-2: D2 bounds each caller to its own openLockRetryAttempts and makes an opening wait last one pebble.Open call. It does not say where the opener takes the shared swap flock. Today lockBeside runs its own retryLocked loop. If that loop runs inside the table attempt, a held swap lock (the D4 exchange) gives nested retries, and waiters wait through them too. No test counts the sleeps under a held swap lock. | design.md > D2: Each open is one `retryLocked` loop over the table — "With no entry, it puts an entry in the opening state and calls `pebble.Open` one time, outside the mutex." | app-1: State in D2 where a new entry takes the swap flock. Either the attempt takes it as one non-retrying flock.Open, classified by classifyFlockError, or the flock is taken before the table with its own loop. Then restate the wait bound of a waiter and of a caller to match.; inv-2: In D2, state that the opener takes the swap flock with one attempt inside the table attempt, or before the loop, and never as a nested retryLocked. Add a case to TestOpenWaitsForSwapLock that holds the swap lock exclusive and asserts countRetrySleeps == openLockRetryAttempts-1. |
| app-2 | approach | should-fix | Task 3.5 lists the comments that say a second open in one process is refused. The list and its rg check miss three comments in internal/query/engine_worktree_test.go. Each says that two engines on one store cannot be open at once, which D1 makes false. The pattern 'second.*open\|lock-refused\|exclusive directory lock\|held the store open' does not match their text. | tasks.md > 3. Readers and the sync of one process — "Several comments say that a second open in one process is refused, and they are wrong after design D1." | Add internal/query/engine_worktree_test.go (near lines 130, 173 and 304) to task 3.5, and add 'exclusive lock per' to its rg pattern. |
| fit-1 | fit | should-fix | Task 2.2 puts a table accessor in internal/graphstore/export_test.go. In this package that file holds the tests of the Export feature. Every graphstore test is in package graphstore, so the tests can read the table without an accessor file. | tasks.md > 2. Shared store handle in `internal/graphstore` — "After all of them close, the table holds no entry, read through an accessor in `export_test.go`" | Remove the accessor from task 2.2 and have the in-package test read the table directly. If a helper is wanted, put it beside countRetrySleeps in retry_test.go, not in the Export test file. |
| fit-2 | fit | should-fix | Today open takes the shared swap lock through lockBeside, which has its own retryLocked loop, and only then runs a second retryLocked loop around pebble.Open. D1 puts the swap lock in the shared entry, and D2 makes one retryLocked loop over the table. Neither decision says where the opening attempt takes the swap lock, or whether it keeps the separate lockBeside retry. | internal/graphstore/pebble_store.go:187 — "swap, err := lockBeside(dir, swapLockSuffix, flock.Shared)" | Name the present open sequence (lockBeside, then retryLocked around pebble.Open) in D2. State whether the opening attempt makes one flock.Open attempt classified by classifyFlockError inside the table attempt, or keeps lockBeside and its own retry, and give the reason. |
| inv-1 | invariants | should-fix | HoldStoreLock locks only the cleaned and the symlink-resolved name, never the absolute name. A probe showed that a hold under ./store/ does not stop pebble.Open of the absolute path in the same process, and a hold under the absolute path does not stop an open of the relative spelling. The planned ./ case checks only that the helper takes the lock. | design.md > D8: A test holds the store through the Pebble lock, not through `Open` — "It locks the cleaned name, and it locks the resolved name too only when the two differ." | In D8 and task 1.1, have HoldStoreLock also lock filepath.Abs(dir) and its resolved form. In the ./ case, assert that Open of the absolute path returns ErrStoreLocked. |
| inv-3 | invariants | should-fix | A waiter on an opening entry must not take a count on an entry that has already moved to closing. The edge is an opener that returns and closes before the parked waiter wakes. No case reaches it: in 2.6 the waiter wakes while the opener still holds its count. Property 2.5 asserts only that each open succeeds, so it also passes when a waiter gets a handle on a closing DB. | design.md > D2: Each open is one `retryLocked` loop over the table — "For an opening entry, it waits for that one `pebble.Open` call to end. It then takes one count, or it returns the error " | In D2, state that a waiter re-checks the state under the mutex and returns ErrStoreLocked on closing. In 2.5, read a record through every handle before its close. In 2.6, add a case where the first opener closes while the waiter is still parked, and assert that the waiter's handle reads. |
| fit-3 | fit | note | The new countRetrySleeps differs from the existing noRetrySleep only in that it counts. TestCheckpointLockedSource, which task 1.2 rewrites, also has an inline copy of noRetrySleep. | internal/graphstore/retry_test.go:15 — "func noRetrySleep(t *testing.T) {" | The parameter is whether the caller reads the count. Write countRetrySleeps once, let noRetrySleep call it and drop the result, and have the rewritten TestCheckpointLockedSource call one of the two instead of its inline swap. |

### Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| A short lock | 1.2, 3.4 | covered |  |
| A long lock | 3.4 | covered |  |
| A sync of the same server longer than the retry window | 3.3 | covered |  |
| A call during one sync | 3.1, 3.2 | covered |  |
| Tool calls back to back during a live edit | 5.1, 5.2 | covered |  |
| An edit right after the handshake | 4.1, 5.1 | covered |  |
| An edit during a takeover | 4.4 | covered |  |
| Nothing changed while the watcher started | 4.2 | covered |  |
| Another process holds the store while the watcher starts | 4.3 | covered |  |

### Not run

Plan phase: no verification runs.
