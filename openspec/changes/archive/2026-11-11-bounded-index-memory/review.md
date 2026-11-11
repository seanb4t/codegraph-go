---
verdict: approve-with-notes
phase: implementation
reviewed_at: 2026-11-11T02:04:55+00:00
head: 2001e04a4b087a1f551d6ffbc4f619c3c07f807c
lenses:
  spec-lint: not-applicable
  approach: partial
  fit: ok
  test-coverage: ok
  test-honesty: partial
  invariants: partial
---
# Review

## Incomplete lenses

- approach: partial, gaps: 3
- test-honesty: partial, gaps: 1
- invariants: partial, gaps: 1

The verdict covers only what the lenses checked.

## Findings

| id | lens | severity | claim | evidence | action |
|---|---|---|---|---|---|
| app-1 | approach | should-fix | The per-file route nodes and route edges of route detection stay in runState until the run ends, after they were copied into the spilled route result. D5 lists every part of the run that stays in memory, drops the declarations and the structural lists at their barriers, and names no per-file route lists. | internal/indexer/fullindex.go:52 — "routeNodes [][]*schema.Node" | Set st.routeNodes and st.routeEdges to nil in detectRoutes once the route result is in the spill, as barrier 1 does for st.decls, or name the per-file route lists in D5 as run-wide state. |
| app-2 | approach | should-fix | D8 moves exportNormalized into determinism_test.go, adds no new export helper, and keeps the copy in phpextract. The code adds a new non-test package, internal/graphstore/storetest, with ExportNormalized, and phpextract and every indexer test call it. Neither design.md nor the proposal records this. | internal/graphstore/storetest/storetest.go:26 — "func ExportNormalized(t testing.TB, storeDir string, clearGeneration bool) []byte {" | Record the storetest package in D8 as a deviation with its reason, and correct the sentences on exportNormalized, the phpextract copy and the helper placement, plus the Impact list of proposal.md and task 1.1; or move the helper back as D8 states. |
| app-3 | approach | should-fix | stampCapabilities in a changed file names resolveRefsWithIndex as the function that stamps interface_methods on every node. No such function exists at head, and D1 moves that stamp into extract.File. | internal/indexer/resolve.go:60 — "// resolveRefsWithIndex stamps arity and interface_methods on every" | Rewrite the three comments in stampCapabilities to name the code that fills each field at head: extract.File for interface_methods, and the resolver functions that fill unresolved_names and declared_module_key. |
| app-4 | approach | should-fix | The comment of TestIndexForceRebuildBumpsCoverageGeneration says that codegraph index wipes the store with RemoveAll and MkdirAll before indexer.Run, and that the test drives that path. This change removed both calls from index.go (D6, task 5.5), so the comment describes a path that no longer exists. | internal/cli/index_test.go:49 — "actual RemoveAll+MkdirAll+indexer.Run path), then assert (a) the" | Rewrite the comment for the run that now reads the prior generation from the live store and replaces it with the exchange. |
| app-5 | approach | should-fix | The comment of TestRoute_DeterministicRebuild says that route edges flow through the collapseEdges/writeGraph path. This change deletes writeGraph (D4, task 5.3), and route edges now form their own unit in pass 3. | internal/indexer/routes_detect_test.go:152 — "// nodes/edges flow through the SAME collapseEdges/writeGraph path as" | Name the route unit of Global.Assemble and the writer of D4 in place of writeGraph. |
| ctl-lens | approach | should-fix | lens approach reported partial, gaps: 3 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| fit-1 | fit | should-fix | The new newBatch helper of the bounded writer repeats the close-guarded batch creation of NewWriter line for line, and NewWriter does not call it. | internal/graphstore/bounded.go:76 — "func (s *pebbleStore) newBatch() (*pebble.Batch, error) {" | Make NewWriter call newBatch and wrap the result in a pebbleWriter, so the WR-01 close guard has one body. |
| fit-2 | fit | should-fix | openExisting detects a store by matching the name of Pebble's private manifest marker file, where the pinned Pebble exports Peek, which reports whether a database exists without opening it. | internal/graphstore/pebble_store.go:245 — "const manifestMarkerPrefix = "marker.manifest."" | Use pebble.Peek(dir, vfs.Default) and its Exists field in openExisting, and map a missing directory to ErrNotFound. Otherwise name Peek in D6 with the reason it does not serve. |
| fit-3 | fit | should-fix | The change resolves the worker default with the same three-line block in run, Sync and forEachFile, although D10 sets out to give the default one place. | internal/indexer/fullindex.go:111 — "if workers <= 0 {" | Add one resolveWorkers(n int) int beside DefaultWorkers in extract.go and call it from the three sites. The alternative is to let forEachFile alone resolve and pass the raw value through. |
| fit-5 | fit | should-fix | The two new sync scans of IterateEdges(id) re-check e.Source == id, which the Reader contract already guarantees. The existing callers, such as calleeCallEdges, filter only by kind, and the design does not name a reason. | internal/graphstore/store.go:58 — "// IterateEdges returns an EdgeIterator over every edge whose source" | Drop the source check and filter by kind as calleeCallEdges does. If IterateEdges is a byte-prefix scan after all, fix its contract comment in store.go instead and say so in D11. |
| ctl-lens | invariants | should-fix | lens invariants reported partial, gaps: 1 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| inv-1 | invariants | should-fix | No test covers either side of the readAhead split in ReadFrameAt, which picks between one read and two. No test reads a whole frame of exactly 4096 bytes or of 4097 bytes. A probe of a copy shows that both read back correctly, so the gap is only in the tests. | internal/recordio/recordio.go:121 — "if uint64(len(have)) >= length {" | Add payload lengths 4093 and 4094 (whole frames of 4096 and 4097 bytes with a 3-byte header) to the weighted lengths of TestFrameRoundTripProperty. Or add them as rows of a ReadFrameAt table that also reads the frame that follows. |
| inv-2 | invariants | should-fix | ReplaceStore can return an error after the exchange or rename has already put the new store in place. This happens when the parent fsync of fsatomic fails, or when the deferred close of the Pebble lock or the swap lock fails. run then returns that error while the live store holds the new graph, which breaks the rule that a failed run leaves the previous records. | specs/index-build/spec.md > Requirement: A failed full index leaves the previous store unchanged — "A run that fails, is cancelled or stops halfway SHALL leave the previous store readable, with the same records it held b" | Choose one rule and enforce it. Option 1: give an error after the swap its own type (for example a wrapped sentinel that says the store was replaced), state in the requirement that it fails after the exchange, and have run report it apart. Option 2: log the post-swap fsync and close errors and return nil. Add a test that sets the syncDir or exchangeDirs seam to swap and then fail, and check that the result matches the chosen rule. |
| inv-3 | invariants | should-fix | After a successful exchange, run joins the error of its deferred scratch.Remove into its return value. A cleanup failure after the swap therefore returns an error for a run whose new store is already live. A caller cannot tell this from a failure that kept the previous store. | internal/indexer/fullindex.go:152 — "defer func() { err = errors.Join(err, scratch.Remove()) }()" | After ReplaceStore returns nil, report a failed scratch removal apart from the run error, for example in Stats or as a wrapped error that says the new store is in place. The next sweep removes the directory anyway. State the rule in the requirement on scratch data, and add a test that fails the removal after the exchange. |
| inv-4 | invariants | should-fix | With no ceiling, which is how the spill reads, ReadFrameAt panics on a length prefix above the maximum int ('makeslice: len out of range', from a probe of a copy). It also tries a huge allocation for any large length instead of returning an error. A corrupt spill record therefore crashes the process from a worker goroutine. The run does not return the first error, and its scratch directory stays until the next sweep. | internal/recordio/recordio.go:124 — "payload = make([]byte, length)" | In ReadFrameAt, compare the length with the bytes that remain in the source when it is an *os.File (size minus offset), or at least with math.MaxInt, and return an error. Add a row to TestReadFrameAtTruncated with a 10-byte uvarint prefix of 2^63 and ceiling 0, and assert an error and no panic. |
| tes-1 | test-coverage | should-fix | The default-worker-count test checks the token peak for extract, structure and resolve only. Route detection has no check on the default count, though the scenario says no more than 3 files in any pass. | internal/indexer/claim_test.go:99 — "for _, step := range []string{stepExtract, stepStructure, stepResolve} {" | In TestDefaultWorkersFollowsGOMAXPROCS, use a tree with Express files (goTree(t, n, true)) and add stepRoutes to the asserted steps, with peak at most wantPeak. |
| ctl-lens | test-honesty | should-fix | lens test-honesty reported partial, gaps: 1 | review > lens-status — "partial" | rerun the lens or accept the gap in the review |
| fit-4 | fit | note | The sync's buildReverseAdjacency, now with a kind parameter, is one more full-scan edge index beside query.BuildReverseAdjacency, BuildImplementsIndex and buildContainsIndex. Design D11 names all four, and issue #274 tracks their collapse. | internal/indexer/sync.go:769 — "func buildReverseAdjacency(r graphstore.Reader, kinds ...string) (map[string][]*schema.Edge, error) {" | None in this change. Issue #274 carries the shared builder, and the comment in sync.go cites it. |
| fit-6 | fit | note | The candidate order of split-indexer-layers D6 is spelled out twice, once per file in Assemble and once across results in CollapseNoFile, though D3 says the order has one implementation. | internal/indexer/resolve/units.go:397 — "{t.edges, &u.NoFile.types}," | Name the parameter: one ordered list of the eight categories (types, calls, deferred, implements, overrides, retried, retried deferred, sets), each with its accessor and a flag for path order. Assemble and CollapseNoFile would both walk it. |
| tes-2 | test-coverage | note | The resolve-pass release check in TestWorkInFlightBound reads only the file record. The scenario says the store holds the records of that file, nodes and edges included. | internal/indexer/fullindex_test.go:301 — "if _, err := r.GetFile(files[file].RelPath); err != nil {" | Also assert that the store holds a node of each noted file, for example through IterateNodes by FilePath. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two fresh indexes | internal/indexer/determinism_test.go: TestDeterministicRebuild | covered | pass |
| A different worker count | internal/indexer/determinism_test.go: TestRebuildIgnoresWorkerCount | covered | pass |
| A different claim order | internal/indexer/schedule_test.go: TestClaimOrderReverse | covered | pass |
| Duplicate edges at one position | internal/indexer/fullindex_test.go: TestDuplicateEdgesCollapse | covered | pass |
| A public part and a part with no modifier | internal/indexer/fullindex_test.go: TestMergedPartialTypeStored/B.cs is public | covered | pass |
| Every part has the default visibility | internal/indexer/fullindex_test.go: TestMergedPartialTypeStored/no modifier on either part | covered | pass |
| A route that one file registers twice | internal/indexer/fullindex_test.go: TestDuplicateRouteStored | covered | pass |
| A sync of one part | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part drops its public modifier | covered | pass |
| A sync adds a part to an indexed type | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part is added | covered | pass |
| A sync after one part leaves the type | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part leaves the type | covered | pass |
| A sync deletes the only part | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/the only part is deleted | covered | pass |
| A sync deletes the only part of a type that another file names | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/the only part is deleted, and another file names the type | covered | pass |
| A sync after one part is deleted | internal/indexer/sync_partial_test.go: TestSyncPartialTypeMatchesIndex/a part is deleted | covered | pass |
| Where working state ends in each pass | internal/indexer/fullindex_test.go: TestWorkInFlightBound | covered | pass |
| Two workers over many files | internal/indexer/fullindex_test.go: TestWorkInFlightBound | covered | pass |
| Default worker count follows the CPU limit | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS | partial | pass |
| One worker | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS/workers 1 over 30 files | covered | pass |
| More workers than files | internal/indexer/claim_test.go: TestDefaultWorkersFollowsGOMAXPROCS/workers 1099511627776 over 3 files | covered | pass |
| A large graph spans several batches | internal/indexer/fullindex_test.go: TestWriteBatchBound | covered | pass |
| A small graph still commits | internal/indexer/fullindex_test.go: TestSmallGraphStillCommits | covered | pass |
| An empty tree | internal/indexer/fullindex_test.go: TestEmptyTreeCommitsMetadataOnly | covered | pass |
| A large export | internal/graphstore/bounded_test.go: TestImportCommitsInBatches | covered | pass |
| An empty export | internal/graphstore/bounded_test.go: TestImportEdgeStreams/an empty stream | covered | pass |
| A commit error halfway through | internal/indexer/fullindex_fail_test.go: TestFailedIndexKeepsStore | covered | pass |
| The exchange cannot take the live store | internal/indexer/fullindex_fail_test.go: TestExchangeFailureKeepsStore | covered | pass |
| The process is killed before the exchange | internal/indexer/fullindex_fail_test.go: TestKilledIndexKeepsStore | covered | pass |
| The process is killed after the exchange | internal/indexer/fullindex_fail_test.go: TestKilledAfterExchange | covered | pass |
| A first index fails | internal/indexer/fullindex_fail_test.go: TestFirstIndexFailureLeavesNoGraph | covered | pass |
| A reader opens the store during a first index | internal/indexer/fullindex_fail_test.go: TestFirstIndexWithConcurrentOpen | covered | pass |
| A full index succeeds | internal/indexer/fullindex_fail_test.go: TestSuccessfulIndexDropsOldRecords | covered | pass |
| A sync during a full index | internal/indexer/claim_test.go: TestSyncDuringFullIndexFails | covered | pass |
| A read during a full index | internal/indexer/claim_test.go: TestReadDuringFullIndexSeesOldGraph | covered | pass |
| After a commit error | internal/indexer/fullindex_fail_test.go: TestIndexLeavesNoScratch/after a commit error | covered | pass |
| After a success | internal/indexer/fullindex_fail_test.go: TestIndexLeavesNoScratch/after a success | covered | pass |
| After a killed run | internal/indexer/fullindex_fail_test.go: TestKilledIndexKeepsStore | covered | pass |
| A run killed while it created its scratch directory | internal/indexer/fullindex_fail_test.go: TestIndexSweepsScratch/a pending directory whose lock no process holds | covered | pass |
| A scratch directory that another process holds | internal/indexer/fullindex_fail_test.go: TestIndexSweepsScratch/a directory whose lock the test holds | covered | pass |
| A writer error | internal/indexer/fullindex_stop_test.go: TestWriterErrorStopsRun | covered | pass |
| A worker error | internal/indexer/fullindex_stop_test.go: TestWorkerErrorStopsRun | covered | pass |
| A cancel while workers wait | internal/indexer/fullindex_stop_test.go: TestCancelStopsBlockedWorkers | covered | pass |

### Not run

None.

## Verification runs

- exit 0: `go test -count=1 -v -run 'TestImportCommitsInBatches' ./internal/graphstore`
- exit 0: `go test -count=1 -v -run 'TestImportEdgeStreams/an empty stream' ./internal/graphstore`
- exit 0: `go test -count=1 -v -run 'TestDefaultWorkersFollowsGOMAXPROCS' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestDefaultWorkersFollowsGOMAXPROCS/workers 1 over 30 files' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestDefaultWorkersFollowsGOMAXPROCS/workers 1099511627776 over 3 files' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestReadDuringFullIndexSeesOldGraph' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncDuringFullIndexFails' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestDeterministicRebuild' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestRebuildIgnoresWorkerCount' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestExchangeFailureKeepsStore' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestFailedIndexKeepsStore' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestFirstIndexFailureLeavesNoGraph' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestFirstIndexWithConcurrentOpen' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestIndexLeavesNoScratch/after a commit error' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestIndexLeavesNoScratch/after a success' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestIndexSweepsScratch/a directory whose lock the test holds' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestIndexSweepsScratch/a pending directory whose lock no process holds' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestKilledAfterExchange' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestKilledIndexKeepsStore' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSuccessfulIndexDropsOldRecords' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestCancelStopsBlockedWorkers' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestWorkerErrorStopsRun' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestWriterErrorStopsRun' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestDuplicateEdgesCollapse' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestDuplicateRouteStored' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestEmptyTreeCommitsMetadataOnly' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestMergedPartialTypeStored/B.cs is public' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestMergedPartialTypeStored/no modifier on either part' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSmallGraphStillCommits' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestWorkInFlightBound' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestWriteBatchBound' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestClaimOrderReverse' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part drops its public modifier' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part is added' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part is deleted' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/a part leaves the type' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/the only part is deleted' ./internal/indexer`
- exit 0: `go test -count=1 -v -run 'TestSyncPartialTypeMatchesIndex/the only part is deleted, and another file names the type' ./internal/indexer`

## Dropped

None.

## Lens notes

- approach: Every requirement of index-build and graph-index has a decision that produces it; no unaddressed or unproducible finding was found in the parts read.
- approach: D1 to D10 were read against fullindex.go, claim.go, extract.go, resolve/units.go, resolve/link.go, graphstore/bounded.go, locks.go, pebble_store.go, export.go, scratchdir, fsatomic, recordio, spill/file.go, cli/workersflag.go and Taskfile.yml, and match except for the findings.
- approach: The signature of forEachFile takes a claimLoop parameter that D8 does not name; it carries the step and state that the D8 hooks need, so it is not reported.
- approach: The CollapseEdges doc comment says two tied candidates keep the first in input order, while it sorts with sort.Slice, which D3 states is not stable; that is left to the invariants lens.
- fit: The prompt names no path for repository library rows, so only the generic rows of the rulebook were applied.
- fit: The implementation collapsed the export normalization of phpextract onto graphstore/storetest.ExportNormalized, while design D8 still says that third copy stays and that no test helper belongs in graphstore.
- fit: storetest.ExportNormalized re-encodes the Meta frame with proto.Marshal instead of recordio.Marshal. schema.Meta has no map field, so the bytes stay deterministic, and this is not raised as a finding.
- fit: Read against the code: recordio, spill codec and file, flock, scratchdir, wtindex/seeddir, fsatomic dirs, graphstore locks, bounded writer, open and retry helper, export Import, claim loop, fullindex run and buildStore, resolve.go, resolve/units.go Assemble and CollapseNoFile, symbols builder merge, csharpextract default, workersflag and its four call sites, gencorpus callers. stage.go, storeview.go, routes_detect.go and the tests were not read in depth.
- invariants: I1: At most W files hold a token at once \| enforced at internal/indexer/claim.go:101 semaphore.NewWeighted, with release in a sync.Once \| boundary test TestDefaultWorkersFollowsGOMAXPROCS (Workers -1, 0, 1, 1<<40), TestForEachFileReleaseOnce, TestForEachFileProperty
- invariants: I2: A batch with more than one record has BeforeLast at or below the cap \| enforced at internal/graphstore/bounded.go:94 (Len() > cap) \| boundary test TestBatchWriterCapEdges rows with cap full-1 and cap full (both sides of the equality), TestBatchWriterCapProperty with caps 0, 11, 12, 13
- invariants: I3: No committed batch is empty \| enforced at internal/graphstore/bounded.go:94 and :135 (Count() > 0, Count() == 0) \| boundary test TestBatchWriterCapEdges rows 'a cap of 0 and two records' and 'no record', TestImportEdgeStreams empty stream
- invariants: I4: The metadata record is in the last batch, and only that commit syncs on a store with a WAL \| enforced at internal/indexer/fullindex.go:592 (PutMeta last) and internal/graphstore/export.go:200 (meta held back) \| boundary test TestBatchWriterSyncLast, TestImportCommitsInBatches, TestEmptyTreeCommitsMetadataOnly
- invariants: I5: A frame payload above the ceiling is refused \| enforced at internal/recordio/recordio.go:133 \| boundary test TestFrameReaderCeiling at the ceiling and one byte over
- invariants: I6: A frame up to readAhead bytes takes one ReadAt, a longer one two \| enforced at internal/recordio/recordio.go:121 \| boundary test none (inv-1). A probe of a copy under the handoff directory (probe-recordio) read sizes 4093, 4094, 4095 and 0 correctly, with 1, 2, 2 and 1 calls
- invariants: I7: The scratch directory goes away on every return path of run \| enforced at internal/indexer/fullindex.go:152 (deferred scratch.Remove, which runs before the deferred writer-lock close) \| boundary test TestIndexLeavesNoScratch, TestWorkerErrorStopsRun, TestExchangeFailureKeepsStore
- invariants: I8: A pending scratch directory without a lock file stays until it is older than PendingMaxAge \| enforced at internal/scratchdir/scratchdir.go:158 (> PendingMaxAge) \| boundary test TestSweep rows at PendingMaxAge minus one second and plus one second
- invariants: I9: A lock file has its flock from the first moment it has its name \| enforced at internal/scratchdir/scratchdir.go:88-92 (Create with O_EXCL and a lock, then rename) \| boundary test TestNewLocksBeforeItNames
- invariants: I10: No open creates the live store between the Lstat and the exchange of ReplaceStore \| enforced at internal/graphstore/pebble_store.go open (shared swap lock before pebble.Open) and locks.go:77 (exclusive swap lock before Lstat) \| boundary test TestOpenWaitsForSwapLock, TestReplaceStoreExchanges (rows for an empty directory and for a LOCK-only directory)
- invariants: I11: ReplaceStore releases the swap lock and the Pebble lock on every return path \| enforced at internal/graphstore/locks.go:81 and :93 \| boundary test TestReplaceStoreReleasesLocksOnFailure, TestReplaceStoreWaitsForOpenHandle
- invariants: I12: A sync during a full index returns ErrStoreLocked \| enforced at internal/indexer/sync.go:66 LockWriter and fullindex.go:122 \| boundary test TestSyncDuringFullIndexFails, TestLockWriterExcludesSecondWriter
- invariants: I13: Every edge of a file unit has the file or '' as its owner path, and every edge of the route unit has a route source \| enforced at internal/indexer/resolve/units.go:343-360 (ErrInternal) \| boundary test TestAssemblyRejectsForeignEdge
- invariants: I14: A merged node takes the first visibility that differs from the default, and is exported if any copy is \| enforced at internal/indexer/symbols/declarations.go:71 mergeCopy \| boundary test TestBuilderMergesSharedID, with the property TestBuilderMergeProperty for the fold and idempotency
- invariants: I15: The pass 3 channel never blocks a worker that holds a token (capacity min(W, records)) \| enforced at internal/indexer/fullindex.go:307 \| boundary test TestSinkBackpressure (length W-1)
- invariants: I16: Import commits no batch for an empty stream \| enforced at internal/graphstore/bounded.go:135 \| boundary test TestImportEdgeStreams
- invariants: Import gave up its all-or-nothing commit, which the doc comment at main named ('the whole import is atomic (D-04)'). Design D9 and the new doc comment state the drop. Import has no production caller, and no test at the merge base relied on it, so this is not filed as weakened.
- invariants: The unbounded-domain invariants already have rapid properties: frame round trip, spill codec round trip, batch cap, merge fold, collapse by unit, claim loop, and schedule independence. No property-candidate is filed.
- test-coverage: Tasks 8.1 and 8.2 (hash comparison against main, 120,000-file benchmark) are unchecked in tasks.md. No scenario depends on them.
- test-coverage: The delta specs contain no BREAKING requirement, so no breaking-unproven finding applies.
- test-coverage: The test suite was not run. Rows rest on reading the assertions.
- test-honesty: Grep of skips over internal/**/_test.go found no new disabled test in the diff. The only skips in diff files are TestHelperKilledIndex (child-process helper, skips when its env is unset, which is legitimate) and a short-mode skip in internal/gencorpus/gen_test.go, which moved with the file.
- test-honesty: The Taskfile race step lists the four packages the spec names, with no ignored failures.
- test-honesty: Kill, failure and sweep tests assert on exports from before and after the run and carry positive assertions such as commit counts and a kill marker. TestWorkInFlightBound blocks the first claim until two files hold a token, so it can fail.
- approach gap: internal/indexer/doc.go and docs/architecture/indexer-layers.html are not read against D1, D4 and D6 (task 8.3).
- approach gap: The kill-process child helpers of D8 in fullindex_fail_test.go are not read against the onCommit and onExchange hook description.
- approach gap: The rest of sync.go after line 705 and the bodies of sharedIDsOfCycle, sharedIDEmitters, deleteNoFileEdges and filesResolvingToChangedSets are not read against D11.
- invariants gap: Requirement: One node merges the copies of an id that several files emit: the sync path of D11 (filesResolvingToChangedSets, the delete of the no-file-unit edges, loadStoreView skips) was read only in part, and no invariant of it was checked against a boundary test.
- test-honesty gap: Read in full: fullindex_fail_test.go, TestWorkInFlightBound and neighbours in fullindex_test.go, TestDefaultWorkersFollowsGOMAXPROCS in claim_test.go. Not read: graphstore, fsatomic, flock, scratchdir, recordio, spill, resolve, symbols, sync and cli tests, tools/bench/runner test, canary workflow body.

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
