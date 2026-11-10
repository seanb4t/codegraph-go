---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-10T23:03:02+00:00
head: e6536a09c1b747032ff814bd6dc4c35b376b524b
lenses:
  spec-lint: not-applicable
  approach: partial
  fit: ok
  test-coverage: partial
  test-honesty: partial
  invariants: partial
---
# Review

## Incomplete lenses

- approach: partial, gaps: 2
- test-coverage: partial, gaps: 1
- test-honesty: partial, gaps: 2
- invariants: partial, gaps: 2

The verdict covers only what the lenses checked.

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | D7 puts the passes and the writer in one errgroup. In the code, run starts the writer only at pass 3, and pass 1, route detection and pass 2 each run earlier in their own forEachFile errgroup. design.md records no deviation. | design.md > D7. Cancellation and goroutine lifetime — "The passes and the writer run in one `errgroup.WithContext`. The first error cancels the context." | Either record in D7 that passes 1, route detection and 2 run in sequence, each in its own forEachFile errgroup, with only pass 3 and the writer sharing one errgroup, or change run to match D7. The doc comment of buildStore makes the same claim and needs the same fix. |
| app-2 | approach | should-fix | D11 says loadStoreView skips the stored edges whose source is a shared id. The code skips every edge whose owner id is a shared id, so it also drops the contains edges from a file node into that id. design.md records no deviation. | design.md > D11. A node id that several files emit becomes one merged node — "It also skips the stored edges whose owner path is "" and whose source is such an id." | Make D11 say that loadStoreView skips both groups of no-file-unit edges: the base edges from the shared id and the contains edges into it. These are the same two groups that the delete step names. Otherwise, narrow the check to edges whose source is the shared id. |
| app-3 | approach | should-fix | D8 names exactly two seam variables for graphstore, and the proposal says Checkpoint changes only by the swap lock. The code adds a third seam, checkpointHook, inside Checkpoint, and design.md does not record it. | design.md > D8. One claim loop and its test hooks — "Its two seam variables are `openLockRetrySleep` and `exchangeDirs` (D6)." | Add checkpointHook to the seam list of D8, saying that TestReplaceStoreWaitsForCheckpoint uses it to hold the source handle open, and update the Checkpoint line of the proposal Impact. Otherwise, rewrite that test without a seam. |
| ctl-lens | approach | should-fix | lens approach reported partial, gaps: 2 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| fit-1 | fit | should-fix | Sync collapses the edges of resolve.Link a second time, though Link now returns edges already collapsed in each unit and stage sorts them by triple itself. | internal/indexer/sync.go:577 — "collapsedEdges := resolve.CollapseEdges(edges, nodeFilePath)" | Drop the CollapseEdges call in Sync and use g.Edges as the collapsed edges. Then no production caller passes a nodeFilePath map, so remove that parameter of CollapseEdges as well. |
| fit-2 | fit | should-fix | Two mechanisms compute the owner path of an edge in one sync: Global.OwnerPath (view plus hierarchy sets) picks the unit inside Link, and ownerPathFor (nodeFilePath plus an r0 fallback) picks the x/ entry and the keep check. | internal/indexer/resolve/units.go:210 — "func (g *Global) OwnerPath(e *schema.Edge) string {" | Have Link return the owner-path function of its Global (for example as a field of resolve.Graph), and pass it to stage and the keep check in Sync, as the full index passes global.OwnerPath. Or name in D3 why the sync keeps ownerPathFor. |
| fit-3 | fit | should-fix | The phpextract test keeps its own copy of the export normalization of exportNormalized. The design's reason (no test-only production helper) does not hold, because the change itself adds test-only exports to recordio. | internal/indexer/mainstream/phpextract/resolution_test.go:85 — "frames, err := recordio.ReadAll(bytes.NewReader(buf.Bytes()), 0)" | Move the export-and-normalize helper into one importable package (a small test-helper package under internal/indexer, or beside ReadAll and AppendFrames) and call it from both tests. Or record the reason for the copy against recordio's test-only exports in D8. |
| fit-4 | fit | should-fix | The spill writer hard-codes 11 as the frame header length, a second name for recordio's unexported maxHeaderLen. | internal/indexer/spill/file.go:48 — "frame := recordio.AppendFrame(make([]byte, 0, len(payload)+11), frameKind, payload)" | Export the constant from recordio (MaxHeaderLen) and use it in spill.File.appendPayload. |
| ctl-lens | invariants | should-fix | lens invariants reported partial, gaps: 2 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| inv-1 | invariants | should-fix | PendingPrefix documents that a pending name never starts with the final prefix, but nothing enforces the trailing dash it relies on. A probe shows that the prefixes "foo" and "" break the rule, and Sweep(parent, "") would treat every unlocked directory in parent as its own. | internal/scratchdir/scratchdir.go:40 — "// finish yet. It never has prefix itself as a prefix, so a pending" | Make New and Sweep return an error (or panic) for a prefix that is empty or does not end in "-", or build the pending name so that it cannot share the final prefix. Add a TestSweep row or a small table test for such a prefix. |
| inv-2 | invariants | should-fix | D3 states that in a full index every unit has a result, so the whole-graph edges of a unit without a result come to nothing. buildStore never calls Global.Rest and checks nothing, so an edge of such a unit would be dropped without an error. | design.md > D3. The global passes run once, and each edge has one unit — "In a full index every unit has a result, so this step emits nothing." | After pass 3 in buildStore (or in stageTail), call global.Rest with the paths of the records and return an error that wraps goextract.ErrInternal if it is not empty. Add a resolve test in which Rest over a unit with no result is non-empty and the full-index check rejects it. |
| inv-3 | invariants | should-fix | D3 says that no unit other than the route unit makes a triple with a route source. Assemble checks only the route-unit side. A file-unit edge whose source is a route node passes when its owner path is the registering file, and it would be collapsed apart from the route unit. | design.md > D3. The global passes run once, and each edge has one unit — "Each of them has a route node as its source, so no other unit makes the same triple." | In the file-unit branch of Assemble, return goextract.ErrInternal for an edge whose source is of kind route. Add a row to TestAssemblyRejectsForeignEdge with a file-unit edge from a route node. |
| ctl-lens | test-coverage | should-fix | lens test-coverage reported partial, gaps: 1 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| ctl-lens | test-honesty | should-fix | lens test-honesty reported partial, gaps: 2 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| fit-5 | fit | note | The sync's buildReverseAdjacency stays a variant of query.BuildReverseAdjacency, now with an edge-kind parameter. The design names it, and issue #274 tracks the collapse. | internal/indexer/sync.go:763 — "binary (D-02a). Issue #274 tracks one shared builder for this map and" | None in this change. Keep the citation of #274 in the comment. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two fresh indexes | internal/indexer/determinism_test.go: TestDeterministicRebuild | covered | pass |
| A different worker count | internal/indexer/schedule_test.go: TestIndexScheduleIndependence | covered | pass |
| A different claim order | internal/indexer/schedule_test.go: TestClaimOrderReverse | covered | pass |
| Duplicate edges at one position | internal/indexer/fullindex_test.go: TestDuplicateEdgesCollapse | covered | pass |
| A public part and a part with no modifier | internal/indexer/fullindex_test.go: TestMergedPartialTypeStored | covered | pass |
| Every part has the default visibility | internal/indexer/fullindex_test.go: TestMergedPartialTypeStored | covered | pass |
| A route that one file registers twice | internal/indexer/fullindex_test.go: TestDuplicateRouteStored | covered | pass |
| A sync of one part | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part drops its public modifier | covered | pass |
| A sync adds a part to an indexed type | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part is added | covered | pass |
| A sync after one part leaves the type | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part leaves the type | covered | pass |
| A sync deletes the only part | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/the only part is deleted | covered | pass |
| A sync deletes the only part of a type that another file names | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/the only part is deleted, and another file names the type | covered | pass |
| A sync after one part is deleted | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part is deleted | covered | pass |
| Where working state ends in each pass | internal/indexer/fullindex_test.go: TestWorkInFlightBound | covered | pass |
| Two workers over many files | internal/indexer/fullindex_test.go: TestWorkInFlightBound | covered | pass |
| Default worker count follows the CPU limit | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS | covered | pass |
| One worker | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS | covered | pass |
| More workers than files | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS | covered | pass |
| A large graph spans several batches | internal/indexer/fullindex_test.go: TestWriteBatchBound | covered | pass |
| A small graph still commits | internal/indexer/fullindex_test.go: TestSmallGraphStillCommits | covered | pass |
| An empty tree | internal/indexer/fullindex_test.go: TestEmptyTreeCommitsMetadataOnly | covered | pass |
| A large export | internal/graphstore/bounded_test.go: TestImportCommitsInBatches | covered | pass |
| An empty export | internal/graphstore/bounded_test.go: TestImportEdgeStreams | covered | pass |
| A commit error halfway through | internal/indexer/fullindex_fail_test.go: TestFailedIndexKeepsStore | covered | pass |
| The exchange cannot take the live store | internal/indexer/fullindex_fail_test.go: TestExchangeFailureKeepsStore | covered | pass |
| The process is killed before the exchange | internal/indexer/fullindex_fail_test.go: TestKilledIndexKeepsStore | covered | pass |
| The process is killed after the exchange | internal/indexer/fullindex_fail_test.go: TestKilledAfterExchange | covered | pass |
| A first index fails | internal/indexer/fullindex_fail_test.go: TestFirstIndexFailureLeavesNoGraph | covered | pass |
| A reader opens the store during a first index | internal/indexer/fullindex_fail_test.go: TestFirstIndexWithConcurrentOpen | covered | pass |
| A full index succeeds | internal/indexer/fullindex_fail_test.go: TestSuccessfulIndexDropsOldRecords | covered | pass |
| A sync during a full index | internal/indexer/claim_test.go: TestSyncDuringFullIndexFails | covered | pass |
| A read during a full index | internal/indexer/claim_test.go: TestReadDuringFullIndexSeesOldGraph | covered | pass |
| After a commit error | internal/indexer/fullindex_fail_test.go: TestIndexLeavesNoScratch | covered | pass |
| After a success | internal/indexer/fullindex_fail_test.go: TestIndexLeavesNoScratch | covered | pass |
| After a killed run | internal/indexer/fullindex_fail_test.go: TestKilledIndexKeepsStore | covered | pass |
| A run killed while it created its scratch directory | internal/indexer/fullindex_fail_test.go: TestIndexSweepsScratch | covered | pass |
| A scratch directory that another process holds | internal/indexer/fullindex_fail_test.go: TestIndexSweepsScratch | covered | pass |
| A writer error | internal/indexer/fullindex_stop_test.go: TestWriterErrorStopsRun | covered | pass |
| A worker error | internal/indexer/fullindex_stop_test.go: TestWorkerErrorStopsRun | covered | pass |
| A cancel while workers wait | internal/indexer/fullindex_stop_test.go: TestCancelStopsBlockedWorkers | covered | pass |

### Not run

None.

## Verification runs

- exit 0: `go test -count=1 -v -run '^TestDeterministicRebuild$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestIndexScheduleIndependence$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestClaimOrderReverse$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestDuplicateEdgesCollapse$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestMergedPartialTypeStored$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestDuplicateRouteStored$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^a_part_drops_its_public_modifier$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^a_part_is_added$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^a_part_leaves_the_type$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^the_only_part_is_deleted$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^the_only_part_is_deleted,_and_another_file_names_the_type$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncPartialTypeMatchesIndex$/^a_part_is_deleted$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestWorkInFlightBound$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestDefaultWorkersFollowsGOMAXPROCS$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestWriteBatchBound$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSmallGraphStillCommits$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestEmptyTreeCommitsMetadataOnly$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestImportCommitsInBatches$' ./internal/graphstore`
- exit 0: `go test -count=1 -v -run '^TestImportEdgeStreams$' ./internal/graphstore`
- exit 0: `go test -count=1 -v -run '^TestFailedIndexKeepsStore$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestExchangeFailureKeepsStore$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestKilledIndexKeepsStore$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestKilledAfterExchange$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestFirstIndexFailureLeavesNoGraph$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestFirstIndexWithConcurrentOpen$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSuccessfulIndexDropsOldRecords$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestSyncDuringFullIndexFails$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestReadDuringFullIndexSeesOldGraph$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestIndexLeavesNoScratch$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestIndexSweepsScratch$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestWriterErrorStopsRun$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestWorkerErrorStopsRun$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run '^TestCancelStopsBlockedWorkers$' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part drops its public modifier' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part is added' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part leaves the type' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/the only part is deleted' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/the only part is deleted, and another file names the type' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part is deleted' ./internal/indexer`

## Dropped

None.

## Lens notes

- approach: Read against the code with no deviation found: D1 (passes, route result as one more record), D4 (bounded writer, Count guard, Sync mode, node count), D6 (writer lock, swap lock, OpenExisting, OpenStaging, ReplaceStore, ExchangeDirs, RenameDir, scratchdir), D9 (Import keeps metadata last), D10 (DefaultWorkers, addWorkersFlag), the claim loop and hooks of D8, and the sync steps of D11 (shared ids, emitters, deletes, RelPath sort).
- approach: The cited names hold at head: gitExecLookPath, addColorFlag, -update-plain-goldens, sameDevice, requireInvariants, exportNormalized, indexAndExportWith, query.BuildReverseAdjacency, BuildImplementsIndex, buildContainsIndex, View.StoredEdges, hierarchy setOf, maxFlushLockRequeues = 5, internal/corpora and tools/corpora, gencorpus.ProductionFileCount.
- approach: tasks.md leaves 1.3 and 4.4 unchecked, although fsatomic.ExchangeDirs, RenameDir and graphstore.ReplaceStore exist at head. That is outside this lens.
- approach: The doc comment of resolveGraph in internal/indexer/resolve.go still says that Run passes it a view, but run no longer calls it. This is a stale code comment, not a design citation.
- fit: The prompt names no path for repository library-preference rows, so only the generic rows of the rulebook apply.
- fit: The shared helpers fit the patterns the design names: internal/flock is the only flock caller, retryLocked takes one classifier per lock, recordio is the only frame codec, forEachFile is the only semaphore and errgroup claim loop, addWorkersFlag is the only --workers registration, and DefaultWorkers is the only GOMAXPROCS default.
- fit: Outside this change: graphstore.importRecord keys the x/ owner of every edge on its source (nodeFilePath[e.GetSource()], export.go:232), while resolve.EdgeOwnerID gives the target for a contains edge. That is a third owner-path rule beside the two in fit-2. The change moved Import onto the bounded writer but did not write this line.
- fit: The full index and Sync each build a fresh Meta with stampCapabilities and HasCoverage (fullindex.go:551, sync.go:226, sync.go:708). The comment at sync.go:232 says each meta-write site stamps HasCoverage on purpose, and the full-index copy replaces the one in the removed writeGraph, so the count of copies does not change.
- invariants: I1: At most W files hold working state in each step \| enforced at internal/indexer/claim.go:101 semaphore.NewWeighted, with the pass 3 release in writeUnits at fullindex.go:512 \| boundary test TestDefaultWorkersFollowsGOMAXPROCS rows -1, 0, 1 and 1<<40 over 3 files, TestForEachFileProperty W -1..64 and n 0..200
- invariants: I2: A pass 3 send never waits once the worker holds a token \| enforced at fullindex.go:298 channel capacity min(workers, records) \| boundary test TestSinkBackpressure
- invariants: I3: A batch exceeds the cap by at most its last record \| enforced at internal/graphstore/bounded.go:94 \| boundary test TestBatchWriterCapEdges rows cap-1, exactly the cap, cap 0, and TestBatchWriterCapProperty weighted on 0, 11, 12, 13
- invariants: I4: No committed batch is empty \| enforced at internal/graphstore/bounded.go:94 and :135 Count checks \| boundary test TestBatchWriterCapEdges rows for no record and cap 0, TestImportEdgeStreams empty stream
- invariants: I5: The metadata record is in the last batch, and the last commit is the only Sync one on a store with a WAL \| enforced at fullindex.go stageTail PutMeta last, export.go importWith meta held aside, bounded.go:141 \| boundary test TestBatchWriterSyncLast, TestImportCommitsInBatches
- invariants: I6: A frame above the ceiling is refused \| enforced at internal/recordio/recordio.go:124 \| boundary test TestFrameReaderCeiling at the ceiling and at ceiling+1; TestFrameRoundTripProperty weights 0, 127, 128, 16383, 16384
- invariants: I7: A pending scratch directory with no lock file stays until PendingMaxAge \| enforced at internal/scratchdir/scratchdir.go:139 \| boundary test TestSweep rows PendingMaxAge-1s and PendingMaxAge+1s
- invariants: I8: OpenExisting creates nothing for a missing or empty directory \| enforced at internal/graphstore/pebble_store.go:261 \| boundary test TestOpenExistingMissingStore
- invariants: I9: ReplaceStore exchanges whenever the live path exists, and renames otherwise \| enforced at internal/graphstore/locks.go:83 \| boundary test TestReplaceStoreExchanges rows real store, empty directory, LOCK-only directory; TestReplaceStoreRenamesFirstStore
- invariants: I10: An unsupported exchange leaves both directories unchanged \| enforced at internal/fsatomic/dirs.go:42 \| boundary test TestExchangeDirsUnsupported rows EINVAL and ENOTSUP
- invariants: I11: The lock retry classifies only its own lock errors \| enforced at pebble_store.go:210 retryLocked and :235 classifyFlockError \| boundary test TestLockRetryClassifies with EACCES rows and a held-once row
- invariants: I12: A file-unit edge has the file or an empty owner path, and a route-unit edge has a route source \| enforced at internal/indexer/resolve/units.go:344 and :350 \| boundary test TestAssemblyRejectsForeignEdge; the converse of the route rule is unenforced (inv-3)
- invariants: I13: The collapse of each unit equals one global collapse \| enforced at none (a property of the unit choice) \| boundary test TestCollapseByUnitProperty, which draws no route unit
- invariants: I14: The merged node takes the first non-default visibility and is exported if any copy is, and merging a copy twice changes nothing \| enforced at internal/indexer/symbols/declarations.go:71 mergeCopy \| boundary test TestBuilderMergesSharedID, TestBuilderMergeProperty
- invariants: I15: The store of a full index does not depend on W, the claim order or the batch cap \| enforced at none (follows from I3, I12 and the merge) \| boundary test TestIndexScheduleIndependence and TestClaimOrderReverse; they compare exports, which carry no x/ entries, and TestLinkUnitsMatchGolden covers x/ on fixtures
- invariants: I16: The spill round-trips each FileResult, including the ErrInternal flag \| enforced at internal/indexer/spill/codec.go:92 and :166 \| boundary test TestSpillRoundTripProperty, TestSpillReadsLargeRecord at 64 MiB + 1
- invariants: I17: A run removes its scratch directory on every return path, before it releases the writer lock \| enforced at fullindex.go:152 deferred scratch.Remove, registered after the writerLock defer at :126 \| boundary test TestIndexLeavesNoScratch, TestWorkerErrorStopsRun, TestExchangeFailureKeepsStore
- invariants: I18: PendingPrefix never starts with the final prefix \| enforced at none \| boundary test none; the probe in probe-pending/main.go of the handoff directory shows that it fails for the prefixes foo and empty (inv-1)
- invariants: Tasks 1.3 and 4.4 are unchecked in tasks.md, but their tests (TestExchangeDirs*, TestRenameDir, TestReplaceStore*, TestOpenWaitsForSwapLock) exist at head.
- test-coverage: No uncovered or weak scenario found in the rows whose bodies were read.
- test-coverage: Tasks 1.3, 4.4 and 8.1 to 8.4 are unchecked in tasks.md although tests for 1.3 and 4.4 exist; not a coverage finding.
- test-honesty: Read in full: fullindex_test.go (merge, work-in-flight, batch bound, empty tree, duplicate edges), fullindex_fail_test.go (failure, kill, exchange, sweep) and fullindex_stop_test.go (backpressure, cancel, writer and worker errors). Each asserts the spec THEN with positive guards, so I report no finding for them.
- test-honesty: internal/gencorpus/gen_test.go TestFileCountExceeds100k skips the 120k materialization under -short. It is a conditional skip behind a planned-count assertion, so I did not report it as a new disabled test.
- approach gap: The body of resolve.Global.Assemble and the per-unit edge order of D3 (internal/indexer/resolve/units.go) are not read line by line against the order of split-indexer-layers D6.
- approach gap: internal/recordio (D2 frame reader ceiling, exported framing) is not read.
- invariants gap: D11 sync path: filesResolvingToChangedSets and the shared-id wave scheduling in internal/indexer/sync.go are not read line by line.
- invariants gap: The bodies of TestSyncPartialTypeMatchesIndex, TestLinkUnitsMatchGolden and the fullindex_fail_test.go tests are not read; only their names and the scenarios in tasks.md are.
- test-coverage gap: Test bodies not read for every row; rows other than TestSyncPartialTypeMatchesIndex, TestWorkInFlightBound, TestWriteBatchBound and the stop tests are graded from names and task text.
- test-honesty gap: Not read: tests in internal/graphstore, flock, fsatomic, recordio, scratchdir, wtindex, spill, resolve, symbols, sync_* tests, cli tests, tools/bench runner test, golden files.
- test-honesty gap: Taskfile.yml and ci.yml diff hunks not inspected; only a grep for continue-on-error and \|\| true, whose hits look pre-existing.

## Plan review (2026-11-10)

Verdict approve-with-notes at head b7cc138c98df58d9755e7a7bfa27a599687dee68.

### Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1+inv-4 | approach,invariants | should-fix | D4 says the metadata record is the last record in Import. At head, Export writes the Meta frame first and Import puts records in stream order. So Import gets the metadata first, not last. Neither D9 nor task 4.2 says that Import must hold the Meta frame until EOF and put it after every other record. The spec requires that the metadata commit in the last batch, and TestImportCommitsInBatches expects GetMeta to return ErrNotFound after every commit but the last. | design.md > D4. One writer, a token bound and a size cap checked on the batch length — "The metadata record is the last record in `run` and in `Import`." | In TestSinkBackpressure, count the pass 3 bodies that returned after their send, or read the queue length through the run state of D8. After synctest.Wait, assert that onClaim fired exactly W times and that all W bodies returned, with W units queued. Only a token wait gives that state. A worker blocked on its send has not returned. |
| app-2+inv-1 | approach,invariants | should-fix | D6 and task 1.4 say that the `Callers` comments of isLockHeldOS, classifyOpenError and ErrStoreLocked name the new callers. Only isLockHeldOS has a `Callers:` line. classifyOpenError says that only Open may call it, and ErrStoreLocked says that classification happens only inside Open's retry loop. Both statements become false once the exchange passes classifyOpenError and LockWriter and the swap lock map flock errors to ErrStoreLocked. The plan does not name that prose. | design.md > D6. Scratch directory, writer lock and atomic exchange — "The `Callers` comments of `isLockHeldOS`, `classifyOpenError` and `ErrStoreLocked` name the new callers." | In D6 and task 1.3, state that the helper fsyncs the parent of each of the two paths, and only once when both paths share a parent. For ExchangeDirs that is .codegraph and the scratch directory. For RenameDir it is the old parent and the new parent. Add a seam that records which directories get the fsync. Then add a row to TestExchangeDirs and to TestRenameDir in which the two paths have different parents, and assert that both parents get the fsync. |
| app-3 | approach | should-fix | D1 says that about 20 tests of resolve_test.go call goextract.Extract directly. At head the file has 16 such calls. Task 3.2 sizes the move by this number. | design.md > D1. Three parallel passes with two barriers — "About 20 tests of `resolve_test.go` call `goextract.Extract` directly today." | Give the count as 16 calls, or describe them without a number. |
| app-4 | approach | should-fix | The rollback is only 'a revert of the change'. After a revert, the earlier release does not remove `.codegraph/store.scratch-*` directories that a killed run left. Each one can hold a whole copy of the old store. The earlier release also leaves the `store.write` and `store.swap` lock files. Its only sweep matches the wtindex seed prefix in the worktree root. The plan has no rollback step for these files. | design.md > Migration Plan — "Rollback is a revert of the change." | Add a rollback step: after a revert, remove `.codegraph/store.scratch-*`, `store.write` and `store.swap`, or state that the files are inert and give their disk cost. |
| fit-1 | fit | should-fix | Task 4.4 makes ExchangeDirs fail in a graphstore test through the seam of fsatomic. The seam then belongs to fsatomic but is set from package graphstore, so it must be exported. The repository's test seams follow one pattern: an unexported package variable with no exported setter, declared in the package that calls it. When a second package needs the same seam, it declares its own copy, as gitmeta does for indexer's gitExecLookPath and wtindex does with sameDevice. D6 does not name the seam, the pattern or a reason to depart from it. | tasks.md > 4. Store writes, locks and replacement — "`TestReplaceStoreReleasesLocksOnFailure`: `ExchangeDirs` returns `EINVAL` through the seam of `fsatomic`." | In D6 and task 4.4, give graphstore an unexported seam variable, for example `exchangeDirs = fsatomic.ExchangeDirs`, that ReplaceStore calls and TestReplaceStoreReleasesLocksOnFailure replaces. Keep the fsatomic seam unexported for TestExchangeDirsUnsupported. If you keep an exported fsatomic seam instead, name it in D6 and say why the per-package pattern does not serve. |
| fit-2 | fit | should-fix | This is a change-vs-repo duplication. TestSpillRoundTrip adds a second walk of internal/indexer/testdata with its own skip list, beside the inline walk of TestEveryDeclarationHasOneNode. Task 3.3 shows the cost: adding the `link` golden directory means editing every skip list, and the plan has to state that the walks agree. The next non-fixture directory under testdata will need the same edit in two places. | internal/indexer/declarations_test.go:533 — "if e.IsDir() && e.Name() != "rapid" {" | In tasks 2.2 and 3.3, move the walk of TestEveryDeclarationHasOneNode into one unexported test helper of package indexer that lists the fixture trees and skips `rapid` and `link`. Put TestSpillRoundTrip in package indexer, which already imports the spill package and runs Extract, and have it call that helper. Drop the sentence about the walks agreeing. |
| inv-2 | invariants | should-fix | The sync's reverse adjacency now holds both calls entries and contains entries, and each reader must keep only its own kind. The dependentPaths loop reads only calls entries. The shared-id step reads only contains entries. No planned test passes the edge that tells the two kinds apart: a pruned node whose only reverse edge from another file is a contains edge, or a shared id that a file in another unchanged file calls. A reader that drops its filter would reparse extra files. Every equality test (TestSyncPartialTypeMatchesIndex and the sync property) would still pass, because extra reparses do not change the store. | design.md > D11. A node id that several files emit becomes one merged node — "Each reader of the map filters by kind. The `dependentPaths` loop reads only the calls entries, as today, so a `contains" | In task 6.1, add a table test that asserts Stats.FilesReparsed, in the style of invariants_test.go line 151. Row one is a sync that prunes a node whose only reverse edge from another file is a contains edge. The file of that edge must not be reparsed. Row two is a sync with a shared id that an unchanged file only calls. The shared-id step must not reparse that caller. Add a positive control: a calls edge into the pruned node does add the caller's file as a dependent. |
| inv-3 | invariants | should-fix | The design still requires that a failed run releases its open batch. The writer goroutine of run and importWith each defer Close. But task 5.3 deletes TestSingleWriter_CloseOnStagingError, the only test that passes a staging error mid-batch and asserts one Close and no Commit. The row of TestBatchWriterCapEdges that calls Close without Commit tests what Close does. It does not test that run or importWith calls Close after an error. So no remaining test passes the error edge. | internal/indexer/resolve_test.go:425 — "t.Errorf("closeCalls = %d, want 1 (must release the abandoned batch)", w.closeCalls)" | Keep a stub-based version of the deleted test. Give the writer-goroutine body, or importWith, a graphstore.GraphStore parameter, so that the stubStore of resolve_test.go can drive it with a stubWriter that fails on PutEdge. Assert one Close, no Commit after the failure, and that the error is returned. Or add the same assertion as an importWith row in graphstore, with a stub store whose writer fails on its second add. |
| fit-3 | fit | note | After the change, the Sync backfill path reads the live store's Meta twice, through two open functions. needsFileIndexBackfill uses graphstore.Open, which creates a missing store, and maps ErrNotFound to a full index. Step 2 of run uses the new OpenExisting and maps ErrNotFound to generation 0. Both open the store by themselves, take a snapshot, call GetMeta, treat ErrNotFound as never indexed and close the store. They differ on purpose only in the open function and in what they derive from Meta. D6 says that only step 2 uses OpenExisting. | internal/indexer/sync.go:642 — "store, err := graphstore.Open(storeDir)" | Parameter: what the caller derives from Meta. One option is an unexported helper in package indexer that opens the store with OpenExisting and returns its Meta or ErrNotFound. needsFileIndexBackfill and step 2 of run would both call it, so a backfill check no longer creates an empty store. Otherwise, add a sentence to D6 that says why needsFileIndexBackfill keeps graphstore.Open. |

### Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two fresh indexes | 7.2 | covered |  |
| A different worker count | 7.1 | covered |  |
| A different claim order | 7.1 | covered |  |
| Duplicate edges at one position | 5.3 | covered |  |
| A public part and a part with no modifier | 5.3 | covered |  |
| Every part has the default visibility | 5.3 | covered |  |
| A route that one file registers twice | 5.3 | covered |  |
| A sync of one part | 6.1 | covered |  |
| A sync adds a part to an indexed type | 6.1 | covered |  |
| A sync after one part leaves the type | 6.1 | covered |  |
| A sync deletes the only part | 6.1 | covered |  |
| A sync deletes the only part of a type that another file names | 6.1 | covered |  |
| A sync after one part is deleted | 6.1 | covered |  |
| Where working state ends in each pass | 5.3 | covered |  |
| Two workers over many files | 5.3 | covered |  |
| Default worker count follows the CPU limit | 5.8 | covered |  |
| One worker | 5.8 | covered |  |
| More workers than files | 5.8 | covered |  |
| A large graph spans several batches | 5.3 | covered |  |
| A small graph still commits | 5.3 | covered |  |
| An empty tree | 5.3 | covered |  |
| A large export | 4.2 | covered |  |
| An empty export | 4.2 | covered |  |
| A commit error halfway through | 5.6 | covered |  |
| The exchange cannot take the live store | 5.6 | covered |  |
| The process is killed before the exchange | 5.6 | covered |  |
| The process is killed after the exchange | 5.6 | covered |  |
| A first index fails | 5.6 | covered |  |
| A reader opens the store during a first index | 5.6 | covered |  |
| A full index succeeds | 5.6 | covered |  |
| A sync during a full index | 5.7 | covered |  |
| A read during a full index | 5.7 | covered |  |
| After a commit error | 5.6 | covered |  |
| After a success | 5.6 | covered |  |
| After a killed run | 5.6 | covered |  |
| A run killed while it created its scratch directory | 5.6 | covered |  |
| A scratch directory that another process holds | 5.6 | covered |  |
| A writer error | 5.4 | covered |  |
| A worker error | 5.4 | covered |  |
| A cancel while workers wait | 5.4 | covered |  |

### Not run

Plan phase: no verification runs.
