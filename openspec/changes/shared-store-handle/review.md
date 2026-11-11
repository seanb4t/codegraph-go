---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-11T15:26:42+00:00
head: 1e016455708823caf88c64b9da858cfd40e19748
lenses:
  spec-lint: ok
  approach: ok
  fit: ok
  test-coverage: ok
  test-honesty: not-applicable
  invariants: ok
---
# Review

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1+inv-2 | approach,invariants | should-fix | app-1: D2 describes an open at head as one retryLocked loop of 5 attempts around one pebble.Open. At head, open first takes the shared swap flock through lockBeside, which runs a retryLocked loop of its own. D1 puts the swap lock in the shared entry, but D2 does not say whether a new entry takes that flock inside the table attempt or with which retry. Its bounds (a waiter waits one pebble.Open, a caller has one open's budget) are therefore not shown to hold, also not for the exchange moment that the spec keeps on the bounded retry.; inv-2: D2 bounds each caller to its own openLockRetryAttempts and makes an opening wait last one pebble.Open call. It does not say where the opener takes the shared swap flock. Today lockBeside runs its own retryLocked loop. If that loop runs inside the table attempt, a held swap lock (the D4 exchange) gives nested retries, and waiters wait through them too. No test counts the sleeps under a held swap lock. | design.md > D2: Each open is one `retryLocked` loop over the table — "With no entry, it puts an entry in the opening state and calls `pebble.Open` one time, outside the mutex." | app-1: State in D2 where a new entry takes the swap flock. Either the attempt takes it as one non-retrying flock.Open, classified by classifyFlockError, or the flock is taken before the table with its own loop. Then restate the wait bound of a waiter and of a caller to match.; inv-2: In D2, state that the opener takes the swap flock with one attempt inside the table attempt, or before the loop, and never as a nested retryLocked. Add a case to TestOpenWaitsForSwapLock that holds the swap lock exclusive and asserts countRetrySleeps == openLockRetryAttempts-1. |
| app-2 | approach | should-fix | Task 3.5 lists the comments that say a second open in one process is refused. The list and its rg check miss three comments in internal/query/engine_worktree_test.go. Each says that two engines on one store cannot be open at once, which D1 makes false. The pattern 'second.*open\|lock-refused\|exclusive directory lock\|held the store open' does not match their text. | tasks.md > 3. Readers and the sync of one process — "Several comments say that a second open in one process is refused, and they are wrong after design D1." | Add internal/query/engine_worktree_test.go (near lines 130, 173 and 304) to task 3.5, and add 'exclusive lock per' to its rg pattern. |
| fit-1 | fit | should-fix | Task 2.2 puts a table accessor in internal/graphstore/export_test.go. In this package that file holds the tests of the Export feature. Every graphstore test is in package graphstore, so the tests can read the table without an accessor file. | tasks.md > 2. Shared store handle in `internal/graphstore` — "After all of them close, the table holds no entry, read through an accessor in `export_test.go`" | Remove the accessor from task 2.2 and have the in-package test read the table directly. If a helper is wanted, put it beside countRetrySleeps in retry_test.go, not in the Export test file. |
| fit-2 | fit | should-fix | Today open takes the shared swap lock through lockBeside, which has its own retryLocked loop, and only then runs a second retryLocked loop around pebble.Open. D1 puts the swap lock in the shared entry, and D2 makes one retryLocked loop over the table. Neither decision says where the opening attempt takes the swap lock, or whether it keeps the separate lockBeside retry. | internal/graphstore/pebble_store.go:187 — "swap, err := lockBeside(dir, swapLockSuffix, flock.Shared)" | Name the present open sequence (lockBeside, then retryLocked around pebble.Open) in D2. State whether the opening attempt makes one flock.Open attempt classified by classifyFlockError inside the table attempt, or keeps lockBeside and its own retry, and give the reason. |
| inv-1 | invariants | should-fix | HoldStoreLock locks only the cleaned and the symlink-resolved name, never the absolute name. A probe showed that a hold under ./store/ does not stop pebble.Open of the absolute path in the same process, and a hold under the absolute path does not stop an open of the relative spelling. The planned ./ case checks only that the helper takes the lock. | design.md > D8: A test holds the store through the Pebble lock, not through `Open` — "It locks the cleaned name, and it locks the resolved name too only when the two differ." | In D8 and task 1.1, have HoldStoreLock also lock filepath.Abs(dir) and its resolved form. In the ./ case, assert that Open of the absolute path returns ErrStoreLocked. |
| inv-3 | invariants | should-fix | A waiter on an opening entry must not take a count on an entry that has already moved to closing. The edge is an opener that returns and closes before the parked waiter wakes. No case reaches it: in 2.6 the waiter wakes while the opener still holds its count. Property 2.5 asserts only that each open succeeds, so it also passes when a waiter gets a handle on a closing DB. | design.md > D2: Each open is one `retryLocked` loop over the table — "For an opening entry, it waits for that one `pebble.Open` call to end. It then takes one count, or it returns the error " | In D2, state that a waiter re-checks the state under the mutex and returns ErrStoreLocked on closing. In 2.5, read a record through every handle before its close. In 2.6, add a case where the first opener closes while the waiter is still parked, and assert that the waiter's handle reads. |
| fit-3 | fit | note | The new countRetrySleeps differs from the existing noRetrySleep only in that it counts. TestCheckpointLockedSource, which task 1.2 rewrites, also has an inline copy of noRetrySleep. | internal/graphstore/retry_test.go:15 — "func noRetrySleep(t *testing.T) {" | The parameter is whether the caller reads the count. Write countRetrySleeps once, let noRetrySleep call it and drop the result, and have the rewritten TestCheckpointLockedSource call one of the two instead of its inline swap. |

## Coverage matrix

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

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: Every requirement has a producing decision: the same-process case of 'A briefly locked store does not fail or hang a call' through D1 to D4, 'An edit made while the watcher starts reaches the graph' through D6 (each Run start, so a takeover too), and the stale requirement through D6's last paragraph and the .sync-pending marker that query/status.go reads.
- approach: Verified at head: pebble v2.1.7 keys its in-process lock registry by the joined LOCK path and returns 'lock held by current process' (vfs/file_lock_unix.go:48-49). A second pebble.Lock Close panics (open.go:1220-1225). indexer.Sync commits one batch on each path (sync.go:249, sync.go:731). The debouncer fires only after Add. internal/flock uses flock(2). depguard graphstore-owns-pebble exempts internal/graphstore/**. golang.org/x/sync is in go.mod.
- approach: isLockHeldOS still cites pebble v2.1.6 and the test that D8 replaces. Task 2.10 already updates that comment.
- approach: The D1 rejected alternative (a serve-owned handle needs a parameter through four packages) is a claim about a design that does not exist. It was judged plausible from the open sites (query.OpenAt, indexer.Sync, wtindex), not traced.
- fit: The context in openspec/config.yaml has no library-preference rows, so only the generic rows apply.
- fit: D2 names golang.org/x/sync/singleflight as an alternative and gives a reason, so it is not a bespoke-over-library case. No other reference-counted registry exists in internal/ (searched for singleflight, refcount and sync.Map).
- fit: D6 reuses the existing debouncer sentinel and requeue wrapper (daemon.go:64, daemon.go:354) and matches the established pattern.
- fit: D8 puts locktest beside storetest and gives the reason: storetest imports graphstore, so in-package tests that imported it would form a cycle. The lint rule graphstore-owns-pebble already allows Pebble in subpackages (.golangci.yml:85).
- fit: D2 says its hooks follow checkpointHook (pebble_store.go:279), and they do.
- fit: D6 does not mention the synchronous startup reconcile in internal/cli/serve.go:315. It serves a different moment (before the first tool call), so it is not raised as a concept duplication.
- fit: The repository has a re-exec helper-process pattern for holders in another process (internal/daemon/stop_test.go:36, internal/indexer/fullindex_fail_test.go:70). D8 does not name it. It is not raised, because the closest pattern for a store holder is the in-process Open, and the design gives its reason for leaving that.
- invariants: I1: Two opens of one store path (same filepath.Abs) in one process share one *pebble.DB and the second does not retry \| enforced at none yet (planned table, task 2.1) \| boundary test TestOpenSecondOpenInProcessSharesHandle (2.1), TestOpenSpellingsShareHandle (2.2)
- invariants: I2: The process releases the Pebble lock and the swap lock exactly when the user count reaches zero \| enforced at none yet (planned Close, task 2.3) \| boundary test TestSharedHandleClose (2.3), counts 2 to 1 to 0 and a double Close
- invariants: I3: Close is idempotent for one caller and lowers the count once \| enforced at internal/graphstore/pebble_store.go:337 closed.Swap \| boundary test TestSharedHandleClose (2.3)
- invariants: I4: Each caller has its own openLockRetryAttempts budget \| enforced at retryLocked internal/graphstore/pebble_store.go:215 \| boundary test TestSharedHandleWaiterKeepsOwnBudget (2.7), TestOpenDuringLastCloseRetries case 4 (2.4); swap-lock path untested (inv-2)
- invariants: I5: A closing entry gives ErrStoreLocked and leaves the table on every return path of the close \| enforced at none yet (task 2.4) \| boundary test TestOpenDuringLastCloseRetries cases 1 to 3 (2.4). Pebble DB.Close releases its file lock before the leaked-iterator check (pebble db.go:1737), so case 3 is reachable
- invariants: I6: A waiter on an opening entry returns the opener's non-lock error at once and retries a lock error on its own budget \| enforced at none yet (task 2.6) \| boundary test TestSharedHandleWaitersShareOpen case 2, TestSharedHandleWaiterKeepsOwnBudget
- invariants: I7: No count is taken on a closing entry \| enforced at none stated \| boundary test none (inv-3)
- invariants: I8: A reader snapshot sees the graph before or after one Sync commit, never a part \| enforced at internal/indexer/sync.go:249 and :731, one Commit on each path, and internal/query/engine.go:200, one Snapshot per OpenAt \| boundary test TestSyncReaderSeesWholeSync (3.1) with both sides of the commit, property TestSyncSnapshotsAreWholeSyncs (3.2)
- invariants: I9: One writer at a time per store \| enforced at LockWriter internal/indexer/sync.go:66 and fullindex.go:122; the only production NewWriter calls on a live store are in sync.go \| boundary test existing TestDaemonSharedWriter
- invariants: I10: ReplaceStore refuses while any in-process handle is open \| enforced at the shared swap flock internal/graphstore/locks.go:95 \| boundary test TestReplaceStoreWaitsForSharedHandle (2.9), with one of two handles closed and then both
- invariants: I11: Each Daemon.Run start runs exactly one catch-up sync with no event \| enforced at none yet (task 4.1); flush ignores its path set (internal/daemon/daemon.go:490), so the sentinel gives a whole-tree sync \| boundary test TestRunSyncsEditMadeBeforeWatchOpen, TestRunCatchUpSyncWithNoChange
- invariants: I12: A catch-up flush that loses the lock through every requeue leaves .sync-pending in place \| enforced at internal/daemon/daemon.go:507-511 and the give-up reset at :362 \| boundary test TestRunCatchUpSyncOnHeldStore (4.3)
- invariants: I13: Lock-held errors of Pebble map to ErrStoreLocked \| enforced at isLockHeldOS internal/graphstore/locked_unix.go \| the replaced TestOpenSecondOpenInProcessReturnsErrStoreLocked pinned the real Pebble message; TestHoldStoreLockExcludesOpen (1.1) pins it again through pebble.LockDirectory, so the check is not weakened
- invariants: Probe: a Go program under inv-probe/ in the handoff directory ran pebble.LockDirectory then pebble.Open on three spellings. Same absolute name: refused with lock held by current process. Hold ./store/, open absolute: open succeeded. Hold absolute, open relative: open succeeded.
- invariants: I14: The catch-up sentinel merges with real events and never runs two syncs at once \| enforced at Debouncer.Add internal/watch/debounce.go:71 (one pending set, ctx gate) and d.syncMu in flush internal/daemon/daemon.go:503 \| boundary test TestRunSyncsEditMadeBeforeWatchOpen (4.1)
- invariants: I15: codegraph_status reports stale while .sync-pending exists \| enforced at computeStale internal/query/status.go:143-148 \| boundary test existing TestStalenessSidecarPresent, and TestRunCatchUpSyncOnHeldStore (4.3) for both sides (marker kept after give-up, removed after one lock loss then success)
- invariants: D2 says the closing entry leaves the table on every return path. A panic in pebble DB.Close (double close, or the metrics check of a race build at db.go:1788) is not a return path. A deferred removal would cover it. Recorded here as context, not as a finding.
- invariants: The replaced TestOpenSecondOpenInProcessReturnsErrStoreLocked also asserted that the full retry budget elapsed. Case 2 of TestSharedHandleWaiterKeepsOwnBudget (2.7) asserts the same budget for an outside holder through countRetrySleeps, so no check is lost.
- spec-lint: Checked all 3 requirements and their 9 scenarios in specs/mcp-server/spec.md: each has a WHEN or GIVEN line and a THEN line, and each THEN names a status, a payload, a count or a field.
- spec-lint: Checked each tracked task in tasks.md (groups 1 to 6): each names a go test, rg, task lint, task test or task changie command, and the file or observable it checks.
- spec-lint: Requirement subjects matched tasks 3.3, 3.4, 5.1, 4.1, 4.2, 4.3 and 4.4 with two distinctive words each.
- spec-lint: No BREAKING word appears in proposal.md or spec.md, so the BREAKING check passes in both directions.
- spec-lint: Build targets task lint, task test and task changie exist in Taskfile.yml. The file test/integration/testdata/watch_live_sync.txtar and the TestScripts entry exist.
- spec-lint: design.md has no open-questions section, so the assumed-answer check had nothing to test.
- spec-lint: The BREAKING grep also matched lines of review.md. I did not use them as evidence; the BREAKING result above rests on my own read of proposal.md and spec.md.
- spec-lint: The run ended before the mid-run checkpoint of the turn budget because the artifact pass completed early.
- test-coverage: No BREAKING requirement in the delta spec.
- test-coverage: Every scenario has a task naming the behavior and a go test command; implementation-phase assertions still need checking.
