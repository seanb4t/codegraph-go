---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-10T13:03:26+00:00
head: b7cc138c98df58d9755e7a7bfa27a599687dee68
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
| app-1+inv-4 | approach,invariants | should-fix | D4 says the metadata record is the last record in Import. At head, Export writes the Meta frame first and Import puts records in stream order. So Import gets the metadata first, not last. Neither D9 nor task 4.2 says that Import must hold the Meta frame until EOF and put it after every other record. The spec requires that the metadata commit in the last batch, and TestImportCommitsInBatches expects GetMeta to return ErrNotFound after every commit but the last. | design.md > D4. One writer, a token bound and a size cap checked on the batch length — "The metadata record is the last record in `run` and in `Import`." | In TestSinkBackpressure, count the pass 3 bodies that returned after their send, or read the queue length through the run state of D8. After synctest.Wait, assert that onClaim fired exactly W times and that all W bodies returned, with W units queued. Only a token wait gives that state. A worker blocked on its send has not returned. |
| app-2+inv-1 | approach,invariants | should-fix | D6 and task 1.4 say that the `Callers` comments of isLockHeldOS, classifyOpenError and ErrStoreLocked name the new callers. Only isLockHeldOS has a `Callers:` line. classifyOpenError says that only Open may call it, and ErrStoreLocked says that classification happens only inside Open's retry loop. Both statements become false once the exchange passes classifyOpenError and LockWriter and the swap lock map flock errors to ErrStoreLocked. The plan does not name that prose. | design.md > D6. Scratch directory, writer lock and atomic exchange — "The `Callers` comments of `isLockHeldOS`, `classifyOpenError` and `ErrStoreLocked` name the new callers." | In D6 and task 1.3, state that the helper fsyncs the parent of each of the two paths, and only once when both paths share a parent. For ExchangeDirs that is .codegraph and the scratch directory. For RenameDir it is the old parent and the new parent. Add a seam that records which directories get the fsync. Then add a row to TestExchangeDirs and to TestRenameDir in which the two paths have different parents, and assert that both parents get the fsync. |
| app-3 | approach | should-fix | D1 says that about 20 tests of resolve_test.go call goextract.Extract directly. At head the file has 16 such calls. Task 3.2 sizes the move by this number. | design.md > D1. Three parallel passes with two barriers — "About 20 tests of `resolve_test.go` call `goextract.Extract` directly today." | Give the count as 16 calls, or describe them without a number. |
| app-4 | approach | should-fix | The rollback is only 'a revert of the change'. After a revert, the earlier release does not remove `.codegraph/store.scratch-*` directories that a killed run left. Each one can hold a whole copy of the old store. The earlier release also leaves the `store.write` and `store.swap` lock files. Its only sweep matches the wtindex seed prefix in the worktree root. The plan has no rollback step for these files. | design.md > Migration Plan — "Rollback is a revert of the change." | Add a rollback step: after a revert, remove `.codegraph/store.scratch-*`, `store.write` and `store.swap`, or state that the files are inert and give their disk cost. |
| fit-1 | fit | should-fix | Task 4.4 makes ExchangeDirs fail in a graphstore test through the seam of fsatomic. The seam then belongs to fsatomic but is set from package graphstore, so it must be exported. The repository's test seams follow one pattern: an unexported package variable with no exported setter, declared in the package that calls it. When a second package needs the same seam, it declares its own copy, as gitmeta does for indexer's gitExecLookPath and wtindex does with sameDevice. D6 does not name the seam, the pattern or a reason to depart from it. | tasks.md > 4. Store writes, locks and replacement — "`TestReplaceStoreReleasesLocksOnFailure`: `ExchangeDirs` returns `EINVAL` through the seam of `fsatomic`." | In D6 and task 4.4, give graphstore an unexported seam variable, for example `exchangeDirs = fsatomic.ExchangeDirs`, that ReplaceStore calls and TestReplaceStoreReleasesLocksOnFailure replaces. Keep the fsatomic seam unexported for TestExchangeDirsUnsupported. If you keep an exported fsatomic seam instead, name it in D6 and say why the per-package pattern does not serve. |
| fit-2 | fit | should-fix | This is a change-vs-repo duplication. TestSpillRoundTrip adds a second walk of internal/indexer/testdata with its own skip list, beside the inline walk of TestEveryDeclarationHasOneNode. Task 3.3 shows the cost: adding the `link` golden directory means editing every skip list, and the plan has to state that the walks agree. The next non-fixture directory under testdata will need the same edit in two places. | internal/indexer/declarations_test.go:533 — "if e.IsDir() && e.Name() != "rapid" {" | In tasks 2.2 and 3.3, move the walk of TestEveryDeclarationHasOneNode into one unexported test helper of package indexer that lists the fixture trees and skips `rapid` and `link`. Put TestSpillRoundTrip in package indexer, which already imports the spill package and runs Extract, and have it call that helper. Drop the sentence about the walks agreeing. |
| inv-2 | invariants | should-fix | The sync's reverse adjacency now holds both calls entries and contains entries, and each reader must keep only its own kind. The dependentPaths loop reads only calls entries. The shared-id step reads only contains entries. No planned test passes the edge that tells the two kinds apart: a pruned node whose only reverse edge from another file is a contains edge, or a shared id that a file in another unchanged file calls. A reader that drops its filter would reparse extra files. Every equality test (TestSyncPartialTypeMatchesIndex and the sync property) would still pass, because extra reparses do not change the store. | design.md > D11. A node id that several files emit becomes one merged node — "Each reader of the map filters by kind. The `dependentPaths` loop reads only the calls entries, as today, so a `contains" | In task 6.1, add a table test that asserts Stats.FilesReparsed, in the style of invariants_test.go line 151. Row one is a sync that prunes a node whose only reverse edge from another file is a contains edge. The file of that edge must not be reparsed. Row two is a sync with a shared id that an unchanged file only calls. The shared-id step must not reparse that caller. Add a positive control: a calls edge into the pruned node does add the caller's file as a dependent. |
| inv-3 | invariants | should-fix | The design still requires that a failed run releases its open batch. The writer goroutine of run and importWith each defer Close. But task 5.3 deletes TestSingleWriter_CloseOnStagingError, the only test that passes a staging error mid-batch and asserts one Close and no Commit. The row of TestBatchWriterCapEdges that calls Close without Commit tests what Close does. It does not test that run or importWith calls Close after an error. So no remaining test passes the error edge. | internal/indexer/resolve_test.go:425 — "t.Errorf("closeCalls = %d, want 1 (must release the abandoned batch)", w.closeCalls)" | Keep a stub-based version of the deleted test. Give the writer-goroutine body, or importWith, a graphstore.GraphStore parameter, so that the stubStore of resolve_test.go can drive it with a stubWriter that fails on PutEdge. Assert one Close, no Commit after the failure, and that the error is returned. Or add the same assertion as an importWith row in graphstore, with a stub store whose writer fails on its second add. |
| fit-3 | fit | note | After the change, the Sync backfill path reads the live store's Meta twice, through two open functions. needsFileIndexBackfill uses graphstore.Open, which creates a missing store, and maps ErrNotFound to a full index. Step 2 of run uses the new OpenExisting and maps ErrNotFound to generation 0. Both open the store by themselves, take a snapshot, call GetMeta, treat ErrNotFound as never indexed and close the store. They differ on purpose only in the open function and in what they derive from Meta. D6 says that only step 2 uses OpenExisting. | internal/indexer/sync.go:642 — "store, err := graphstore.Open(storeDir)" | Parameter: what the caller derives from Meta. One option is an unexported helper in package indexer that opens the store with OpenExisting and returns its Meta or ErrNotFound. needsFileIndexBackfill and step 2 of run would both call it, so a backfill check no longer creates an empty store. Otherwise, add a sentence to D6 that says why needsFileIndexBackfill keeps graphstore.Open. |

## Coverage matrix

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

## Verification runs

None.

## Dropped

None.

## Lens notes

- approach: Every index-build and graph-index requirement maps to a decision: the work bound (D4, D8, D10), batches (D4), import (D4, D9), the failed index (D6), the only writer (D6), scratch (D6), the first error (D7), INDX-02 claim order (D8), and the merged node with its sync scenarios (D11).
- approach: These cited facts check out at head: writeGraph, Resolve, resolveFunc, NewWriterSized, BatchSize, DeleteGraph; the three NumCPU defaults; the --workers flags in index, init, sync and daemon; the CoverageGenerationFloor flow; the seeddir lockFile and sweep gaps; detectRoutesInFile making a parser per file; the route node id; setOf with FilePath empty; the methodOwner last-wins rule; rewireSetEdges' stored-set contains edges; buildReverseAdjacency filtering to calls; filesResolvingToChangedSets skipping FilePath-empty nodes; maxFlushLockRequeues=5 with a 2000 ms debounce; Batch.Len counting a 12-byte header; 'pebble: WAL disabled'; LockDirectory and pebble's in-process lock registry; renameat2 RENAME_EXCHANGE and RenamexNp RENAME_SWAP in x/sys v0.48.0; go 1.26; the two export frame decoder copies; the four writeGraph single-batch tests; gitExecLookPath; and -update-plain-goldens.
- approach: Not filed: D3 says that each edge's unit follows the owner path, but a route edge's owner, the route node, has the registering file as its FilePath (routes_detect.go:167). The route unit therefore follows from the route result, not from the owner-path rule. The design stays coherent because pass 3 handles the route result as its own record.
- fit: Every fit finding of the prior round is resolved at this head. Old fit-1: D4 now gives the memory reason against several writers. Old fit-2: D4 and task 5.3 remove DeleteGraph and rewrite the DeleteAllExcludedFiles doc. Old fit-3: task 1.1 moves the phpextract frame codec onto recordio and deletes its comment. Old fit-4: D3 names the supertype map of Hierarchy and says why its kinded table stays separate.
- fit: For the controller, outside the fit categories. tools/bench/cpudiag/main.go:7-8 still says the worker pool defaults to runtime.NumCPU() in extract.go, pipeline.go and sync.go. D10 replaces exactly those three defaults, and no task updates that comment.
- fit: For the controller, outside the fit categories. internal/graphstore/testdata/rapid/TestBatchSizeCoversStagedRecords/ still holds a rapid failure file, and task 4.1 deletes batchsize_test.go without removing it.
- fit: For the controller, outside the fit categories. The only caller of indexer.resolveRefs (internal/indexer/resolve.go:41) is Resolve (resolve.go:72), which task 5.3 removes. The design does not list resolveRefs among the removals, and its doc says 'as a full index does', which the new passes make false. resolveGraph stays for Sync (sync.go:438).
- fit: For the controller, outside the fit categories. D8 says that graphstore has no hook variables. But openLockRetrySleep (pebble_store.go:91) is one, and D6 keeps it as the seam of the retry helper. The sentence should say no new hook variables.
- fit: For the controller, an artifact inconsistency that is still open from the prior round. Proposal Impact says that scratchdir moves out so that graphstore can use it too. D6 says that graphstore does not depend on scratchdir, and run in internal/indexer is the code that creates the store scratch directory.
- fit: Checked and consistent. forEachFile keeps the persistent workers and atomic counter that extract.go:68-77 documents. Hooks are unexported indexer variables, in the style of gitExecLookPath. The test flag -update-link-goldens follows -update-plain-goldens (internal/cli/plain_golden_test.go:26). importWith follows extractWithFactory. addWorkersFlag follows addColorFlag (internal/cli/colorflag.go:52) and replaces the four flag lines (init.go:94, daemon.go:254, sync.go:64, index.go:170). DefaultWorkers replaces the only three NumCPU defaults (extract.go:80, pipeline.go:160, sync.go:88). recordio collects the only non-test deterministic marshal (graphstore/batch.go:35) and the uvarint frame copies (export.go, determinism_test.go, phpextract resolution_test.go).
- fit: Checked and not filed. protodelim in google.golang.org/protobuf provides uvarint-delimited proto framing, but the export frame starts with a kind byte and the spill payload is gob, so it cannot replace recordio. golang.org/x/sync/semaphore, errgroup, goleak, rapid and go-cmp are already in go.mod. x/sys/unix provides both Renameat2 and RenamexNp.
- fit: Checked and not filed. A third copy of the export normalization stays in phpextract indexAndExport, and D8 gives the reason. The change does not add a copy there, and the other two copies merge into exportNormalized. writeGitignoreHint (internal/cli/init.go:135) and the seed .gitignore that moves into scratchdir are two one-line writes. They predate this change, which moves one of them and adds none.
- invariants: I1: At most W files hold a token at once in each step; W is --workers above zero, else GOMAXPROCS \| enforced at the weighted semaphore of forEachFile (task 5.1) \| boundary test the 5.8 table (-1, 0, 1, 2^40), TestForEachFileProperty (n from 0, failing bodies), TestWorkInFlightBound
- invariants: I2: release acts once, and a failed body leaks no token \| enforced at the release guard of D8 \| boundary test TestForEachFileReleaseOnce and the failing-index draws of TestForEachFileProperty
- invariants: I3: A send to the writer never waits once a worker holds its token \| enforced at the writer channel of size min(W, n) (D4) \| boundary test none that tells a token wait from a send wait (inv-4)
- invariants: I4: A batch exceeds the cap by at most its last record \| enforced at the Batch.Len check before each add (task 4.1) \| boundary test TestBatchWriterCapEdges (above the cap, exactly the cap, cap plus one, cap 0) and TestBatchWriterCapProperty (caps 0, 11, 12, 13)
- invariants: I5: No batch is empty, the last record is in the last batch, and no record means no commit \| enforced at the Count() > 0 guard at both commit sites \| boundary test the no-record row, the empty-stream row and TestEmptyTreeCommitsMetadataOnly. Probe: an empty Pebble batch has Len 12 and Count 0, and Pebble commits it with no error, so the guard is the one enforcement point, as D4 says
- invariants: I6: Staging commits use NoSync, and the last commit on a store with the WAL on uses Sync \| enforced at the writer's commit mode \| boundary test TestOpenStaging, TestBatchWriterSyncLast and TestImportCommitsInBatches. Probe: a Sync commit on a store opened with DisableWAL returns 'pebble: WAL disabled', and a NoSync commit succeeds
- invariants: I7: A staging store keeps its committed records after Close \| enforced at the flush in the Close of OpenStaging \| boundary test the positive assertion of TestOpenStaging. Probe: with the WAL off, a Close without a flush loses a NoSync commit (a reopen gets 'pebble: not found'), so the flush is required
- invariants: I8: A frame longer than the ceiling is refused, and zero means no ceiling \| enforced at the recordio reader \| boundary test TestFrameReaderCeiling (1 KiB and 1 KiB plus one) and TestSpillReadsLargeRecord (64 MiB plus one)
- invariants: I9: A frame round-trips in every length class of the uvarint prefix \| enforced at the recordio writer and reader \| boundary test TestFrameRoundTripProperty (0, 127, 128, 16383, 16384)
- invariants: I10: Decoding an encoded spill record gives the same result, up to empty versus nil, and ErrInternal survives \| enforced at the spill codec \| boundary test TestSpillRoundTrip and TestSpillRoundTripProperty
- invariants: I11: Concurrent spill appends get distinct offsets and contiguous records \| enforced at the spill mutex \| boundary test TestSpillConcurrentAppends, empty frame included
- invariants: I12: Every candidate of one triple is in one unit, so a collapse per unit equals one global collapse \| enforced at EdgeOwnerID, which reads only kind, source and target (collapse.go lines 20 to 25), and at CollapseEdges, which builds each group in input order and sorts it apart from the other groups (collapse.go lines 34 to 68). So a unit's group equals the global group, unstable sort or not. The ErrInternal unit checks add to this \| boundary test TestAssemblyRejectsForeignEdge, TestCollapseByUnitProperty and TestLinkUnitsMatchGolden
- invariants: I13: Meta.NodeCount is view nodes plus package nodes plus set nodes, with no overlap \| enforced by construction \| boundary test requireInvariants in TestMergedPartialTypeStored and overload_test.go. The sync count (sync.go lines 510 to 528) counts a shared id that r0 already stores as not added
- invariants: I14: A merged node takes the first non-default visibility, is exported if any copy is, and takes its other fields from the first copy \| enforced at the builder merge (D11) \| boundary test TestBuilderMergesSharedID and TestBuilderMergeProperty (a copy added twice changes nothing). C# passes the default internal for every type, nested types included (csharpextract.go line 232)
- invariants: I15: No shared id carries InterfaceMethods \| enforced by construction: goextract.go line 298 is the only writer of FileResult.InterfaceMethods, and a Go interface lives in one file \| boundary test none needed
- invariants: I16: The stored records do not depend on W, the claim order or the batch cap \| enforced at the file-order slots, the merge and the per-unit collapse \| boundary test TestIndexScheduleIndependence and TestClaimOrderReverse
- invariants: I17: Only one writer commits to a store at a time \| enforced at LockWriter in run and Sync \| boundary test TestLockWriterExcludesSecondWriter and TestSyncDuringFullIndexFails
- invariants: I18: No open of this release creates the live store between the look and the exchange \| enforced at the shared swap lock that open takes before pebble.Open \| boundary test TestOpenWaitsForSwapLock
- invariants: I19: ReplaceStore releases the swap lock and the Pebble lock on every return path \| enforced at D6 step 7 \| boundary test TestReplaceStoreReleasesLocksOnFailure and TestReplaceStoreWaitsForOpenHandle. Probe: pebble.LockDirectory creates LOCK in an empty directory, and a second lock in the same process returns 'lock held by current process'
- invariants: I20: An exchange or a rename that returned is durable \| enforced at a fsync of 'the parent directory' \| boundary test none (inv-1). Probe: on APFS, renamex_np(RENAME_SWAP) swaps <scratch>/store with .codegraph/graph across two parents, including an empty live directory, and a missing target gives ENOENT. A directory Sync returns nil on APFS
- invariants: I21: A run leaves no scratch directory on any return path \| enforced at the cleanup of run \| boundary test TestIndexLeavesNoScratch, the rows of TestWorkerErrorStopsRun and TestExchangeFailureKeepsStore
- invariants: I22: The sweep never removes a held directory, never creates a lock file, and keeps a pending directory younger than PendingMaxAge \| enforced at the scratchdir sweep \| boundary test TestSweep (PendingMaxAge plus and minus one second, both with and without a temporary lock file) and TestIndexSweepsScratch
- invariants: I23: The coverage generation is the prior generation plus one, and a missing or unreadable record gives one \| enforced at step 2 of run \| boundary test the moved subtests of task 5.5 and TestSyncDuringFullIndexFails
- invariants: I24: The first error stops the run, and no goroutine remains \| enforced at the errgroup \| boundary test TestWriterErrorStopsRun, TestWorkerErrorStopsRun (four steps) and TestCancelStopsBlockedWorkers
- invariants: I25: A failed run or import releases its open batch \| enforced at the deferred Close in the writer goroutine and in importWith \| boundary test none after task 5.3 deletes TestSingleWriter_CloseOnStagingError (inv-3)
- invariants: I26: Each reader of the sync reverse adjacency keeps only its own edge kind \| enforced at the per-reader filter (D11) \| boundary test none (inv-2)
- invariants: I27: A sync gives the same merged node and the same no-file edges as a full index \| enforced at the shared-id step, the skip in loadStoreView and the delete step \| boundary test the rows of TestSyncPartialTypeMatchesIndex and the sync property with a C# partial type
- invariants: I28: The sync merges in the same path order as a full index \| enforced at the sort by RelPath, the same comparator as DiscoverAll (discoverexclusion.go line 282) \| boundary test the sync property
- invariants: Property candidates: every invariant over an unbounded domain already has a rapid property in the plan (frames, spill codec, cap, collapse by unit, forEachFile, merge, conformance index, schedule independence, sync). No property-candidate finding.
- spec-lint: Checked scenario shape: every requirement in specs/index-build/spec.md and specs/graph-index/spec.md has at least one scenario; every scenario has WHEN or GIVEN and THEN.
- spec-lint: Checked requirement-to-task coverage for all index-build and graph-index requirements; each subject has a matching task line with two or more distinctive words.
- spec-lint: Checked BREAKING markers: none in proposal.md or either delta spec, so no mismatch.
- spec-lint: Checked build targets named in tasks.md: task test:golden, golden:regen, test:race, test and changie exist in Taskfile.yml.
- spec-lint: Checked design.md: Open Questions states None, so no assumed-answer finding.
- spec-lint: THEN and WHEN steps of the delta scenarios name observables or concrete inputs; no finding raised.
- test-coverage: No BREAKING marker in any artifact, so no breaking-unproven finding.
- test-coverage: Tasks name tests and say 'make sure the tests pass' but give no literal go test command; I graded a named test with a positive assertion as the verification.
- test-coverage: Worker-count scenarios are written as CLI commands (--workers -1, 0) but task 5.8 drives Options.Workers; only 1.5 checks flag help text. No CLI-level parse test of negative values.
- fit gap: openspec/config.yaml context holds no library-preference rows, so only the generic rows of the rulebook applied.
- fit gap: The proposal names #269 and #274 as later work. I could not read issue #274, because this lens has no gh or Bash. I relied on the D11 text, which says the kind parameter of the widened buildReverseAdjacency serves that collapse.
- fit gap: I read the design of symbol-and-reference-completeness. Its competing-write rule for callable bindings and its binding dependencies need run-wide state across files before Calls runs. symbols.Declarations (D1) is the natural place to extend for that. I filed nothing, because the proposal does not name that change as the next one. I did not read the design of shell-language-support in full.
- fit gap: generated_at is the date of the run at 00:00:00Z, because this lens had no clock read.
- invariants gap: Plan phase: no enforcement point is written yet, so no enforcement script could be run. Probes under the handoff directory (inv-probe, inv-probe-swap, inv-probe-dirsync) ran library and syscall behaviour on edge inputs instead.
- invariants gap: Not probed: whether Pebble behaves inside a testing/synctest bubble, which TestSinkBackpressure and TestCancelStopsBlockedWorkers need. Background goroutines, global state created in a bubble, and goleak.IgnoreCurrent on the first Pebble use are all open. A probe needs a test command, and this lens may not run one. internal/graphstore/store_test.go lines 20 to 25 already avoid synctest with Pebble.
- invariants gap: Not probed: Linux renameat2(RENAME_EXCHANGE). The host is macOS, so only renamex_np(RENAME_SWAP) was probed.
- spec-lint gap: Repository code paths named in tasks (tools/bench/gencorpus, internal/corpora) were not checked for existence; only Taskfile targets were checked.
