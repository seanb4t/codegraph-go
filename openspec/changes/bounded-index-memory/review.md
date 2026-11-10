---
verdict: approve-with-notes
phase: plan
reviewed_at: 2026-11-10T00:49:20+00:00
head: 78125cde530dbeea8896a31cef87b6b016c44959
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
| fit-1 | fit | should-fix | D6 moves the pebble.Open retry loop into one helper that open, LockWriter and the exchange all call. In graphstore, lock-held classification is documented as valid only for errors that come from pebble.Open (isLockHeldOS, classifyOpenError, ErrStoreLocked). LockWriter's flock fails with an error from flock, not from Pebble. The design does not name that rule, does not say how the shared helper classifies a flock failure, and gives no reason to depart from the rule. | internal/graphstore/pebble_store.go:98 — "// Classification happens exactly once, inside Open's retry loop — the only" | In D6 and task 1.4, make the helper take the attempt and its own classifier: classifyOpenError for pebble.Open and pebble.LockDirectory, and a flock classifier (EWOULDBLOCK from LOCK_NB) for LockWriter. Otherwise, name the single-seam rule and say why it no longer holds. Either way, update the 'Callers' comments of isLockHeldOS and ErrStoreLocked. |
| fit-2 | fit | should-fix | Duplication (change-vs-repo): D3 builds one barrier-2 member index to replace the two member maps of retryConformanceCalls and synthesizeOverrides. But synthesizeGoImplements runs at the same barrier and builds its own struct-to-methods map and interface-embeds map from the same structural edges. After the change, barrier 2 still has two member indexes and two supertype views. | internal/indexer/resolve/helpers.go:80 — "typeMethods := make(map[string]map[string]string) // typeID -> methodName -> methodID" | In D3 and task 3.4, have synthesizeGoImplements read the struct methods from the member index, with arity from the view. Have it read the interface-to-interface embeds from the hierarchy's supertypes, filtered by kind. Otherwise, state why it keeps its own maps. |
| fit-3 | fit | should-fix | Duplication (change-vs-repo): Three copies of 'open the store, export it, zero the meta last-sync time, re-encode' exist today: the inline copy in indexAndExportWith, exportNormalized, and indexAndExport in the phpextract resolution test. D8 merges the first two. Task 1.1 rewrites the phpextract test's frame code onto recordio, but keeps its normalization copy, and D8 says no export helper is added. The phpextract comment justified the copy by the unexported framing, which this change exports. | internal/indexer/mainstream/phpextract/resolution_test.go:116 — "func indexAndExport(t *testing.T, root string) []byte {" | In D8 and task 1.1, move the normalization into the one shared function and make all three callers and the comparisons of task 5.6 use it. Otherwise, state in D8 why the phpextract test keeps its own normalization once recordio and the exported kind tags exist. |
| fit-4+spe-1 | fit,spec-lint | should-fix | Duplication (change-internal): Task 3.3 describes a golden recorder, and task 5.3 describes TestLinkUnitsMatchGolden. Both run the indexer into a temporary store, export it, keep the edge and file frames in key order and append the file-index entries. The repository's update-flag pattern is one test that writes under the flag and compares without it. The tasks describe two separate procedures. | tasks.md > 3. Symbols and resolution — "The recorder runs `Run` into a temporary store, exports it, and keeps the edge and file frames in key order." | In task 3.3, add TestLinkUnitsMatchGolden itself, with the flag as its write branch, and record the goldens with it before Link changes. In task 5.3, keep only its positive assertions on the kinds of edges the goldens hold. |
| inv-1+inv-3 | invariants | should-fix | The rule that no batch is empty has no enforcement point, and the D4 argument for it breaks at two edges. Pebble's Batch.Len() returns max(batchrepr.HeaderLen, len(data)), which is 12 for an empty batch (pebble v2.1.7 batch.go:1237). With a cap below 12, for example the cap 0 or 1 that a test can set through batchCap or importWith to force many commits, the check before the first add finds 12 above the cap and commits an empty batch. Close(sync), as task 4.1 states it, commits the open batch without condition, so a writer that got no record commits an empty batch too. Pebble accepts both commits without an error. The probe probe-batch-len under the handoff directory follows the planned rule literally: cap 0 or 11 with two records commits batches with 0, 1 and 1 records, cap 12 commits 1 and 1, and a Close with no record commits one empty batch. TestBatchWriterCapProperty draws the cap from 1 KiB, so it never reaches the edge. Only the empty-stream row of TestImportEdgeStreams sees the Close case, and only for Import. | design.md > D4. One writer, a token bound and a size cap checked on the batch length — "If that length is already above the cap, it commits the batch and starts a new one, and then adds the record." | In D4 and task 4.1, guard both commit sites on a non-empty batch (Batch.Count() > 0, or !Batch.Empty()): the cap check before an add and Close. Alternatively, make NewBoundedWriter reject a cap at or below the 12-byte batch header. Add rows to TestBatchWriterCapEdges: a cap of 0 with two records commits two batches of one record each, and a Close with no record commits no batch. Widen TestBatchWriterCapProperty to draw the cap from 0 and the record count from 0. |
| inv-2 | invariants | should-fix | The spill reads depend on an invariant that the plan never states: under concurrent appends from the W workers of pass 1, the bytes of each record stay contiguous at the offset that the writer returns. Every pass 1 worker writes its own result into the one spill file. D2 states only that ReadAt is safe from many goroutines, and task 2.3 gives the spill writer no serialization point, for example a mutex around the offset counter and the write. The spill package tests only concurrent reads, so unserialized appends can interleave frames or hand out duplicate offsets with no test of the spill package failing. Only TestWorkInFlightBound (W=2) and the race step would catch it, indirectly. | design.md > D1. Three parallel passes with two barriers — "A worker runs `extractOne` on one file, then writes the result to the spill file (D2)." | In D2 and task 2.3, state that the spill writer is safe for concurrent appends, and name the point that serializes the offset assignment and the write, for example one mutex. Add TestSpillConcurrentAppends: 8 goroutines append 1,000 records of drawn sizes, the empty frame included. Each returned offset must be distinct, and a read at each offset must return its own record. Run the test under the race step of task 5.9. |
| fit-5 | fit | note | Duplication (third-copy): Issue #274 tracks the collapse of the full-scan edge indexes, and D11 names three of them. query.buildContainsIndex is a fourth full scan of IterateEdges filtered by kind, and task 6.1 widens the sync's copy to contains edges too. D11 does not name buildContainsIndex, so it is outside the tracked collapse. | internal/indexer/sync.go:614 — "func buildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {" | Name buildContainsIndex in the D11 sentence and in #274, or state why it stays. It indexes by source and folds overload sets. |
| fit-6 | fit | note | D10 shares only the help text of --workers. The four commands still each register the same flag line with name, default 0 and that text. The only part that varies is the variable the flag binds, and a helper can take it as its parameter. | internal/cli/index.go:170 — "cmd.Flags().IntVar(&workers, "workers", 0, "bound the extraction worker pool (default: number of CPUs)")" | Optionally, in D10 and task 1.5, add addWorkersFlag(cmd, &workers) in internal/cli, in the style of addColorFlag. Its parameter is the bound variable. All four commands then call it. |

## Coverage matrix

| scenario | covered by | status | verified |
|---|---|---|---|
| Two fresh indexes | 7.1 | covered |  |
| A different worker count | 7.1 | covered |  |
| A different claim order | 7.1 | covered |  |
| Duplicate edges at one position | 5.3 | covered |  |
| A public part and a part with no modifier | 5.3 | covered |  |
| Every part has the default visibility | 5.3 | covered |  |
| A route that one file registers twice | 5.3 | covered |  |
| A sync of one part | 6.1 | covered |  |
| A sync adds a part to an indexed type | 6.1 | covered |  |
| A sync deletes the only part | 6.1 | covered |  |
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
| A cancel while workers wait | 5.4 | covered |  |

### Not run

Plan phase: no verification runs.

## Verification runs

None.

## Dropped

- inv-4 (invariants, confidence 0.6): D3 replaces the two member maps of helpers.go with one member index plus Hierarchy.OwnerOf, and it claims the retry and override results stay 'as today'. That is an equivalence over an unbounded domain, the lists of structural edges. The two owner rules differ: today's retry takes the last contains edge from any source (file nodes included), and OwnerOf keeps only a source of type kind. Only fixture goldens and three real trees check the equivalence.

## Lens notes

- approach: Every requirement of index-build and graph-index has a producing decision: work-in-flight bound (D4, D8, D10), batch cap (D4), Import (D4, D9), failed index and atomic replacement (D6, D7), only writer (D6 writer lock), no scratch data (D6 scratchdir and sweep), first error (D7), INDX-02 claim order (D3, D4, D8), merged node and its sync scenarios (D11).
- approach: Every scenario THEN is reachable through a named hook or artifact: onRelease carries step, file and run state; Commit carries Len, BeforeLast, Records and Sync; onExchange and onPass exist. The prior round's app-1 and app-2 are resolved in this revision, as are app-3 (overlay read by synthesizeOverrides), app-4 (overloadfixture named) and app-5 (generateSyntheticCorpus named).
- approach: Verified at head: Link order and withStoredEdges (link.go:98-121); retryConformanceCalls takes the last contains edge from any source and the last id per name (helpers.go:84-103); synthesizeOverrides filters KindMethod and reads implements as a supertype edge (helpers.go:185-200); synthesizeGoImplements reads struct-method contains and interface-interface embeds (helpers.go:278-293); NewHierarchy reads only contains, embeds, extends and implements (hierarchy.go:72-80).
- approach: Verified at head: EdgeOwnerID takes the target for contains and the source otherwise, CollapseEdges uses sort.Slice (collapse.go); writeGraph builds nodeFilePath from g.Nodes, which holds the view nodes plus set nodes, and aborts on a non-NotFound GetMeta error (resolve.go:118-142); stage returns a distinct count and Sync ignores it (sync.go:531).
- approach: Verified at head: Types sets InterfaceMethods (types.go:44) and appends IntraEdges to its edges; AddResults adds nodes of error results before the Err check (builder.go:171-187); loadStoreView calls AddInterfaceMethods (storeview.go:79) and runs before AddResults in Sync (sync.go:430-435); Sync builds buildReverseAdjacency only when goneIDs is non-empty, filtered to calls (sync.go:299, 614-633).
- approach: Verified at head: detectRoutesInFile creates a parser per file (routes_detect.go:121); every route edge has a route node as source (routes_detect.go:173); buildGlobalHandlerIndex reads only Err, Language and nodes, which Declarations carries.
- approach: Verified at head: run, Resolve, writeGraph, resolveFunc and the three runtime.NumCPU defaults (pipeline.go:160, sync.go:88, extract.go:80); the four --workers flag lines; codegraph index requires .codegraph, classifies the prior generation in three outcomes, then RemoveAll and MkdirAll (index.go:81-146); init creates the store directory (init.go:58); wtindex.Build.build removes the store before Run (wtindex.go:208); seeddir uses MkdirTemp before its lock and lockFile uses O_CREATE then Flock; place uses os.Rename.
- approach: Verified at head: open retries 5 times at 100 ms with openLockRetrySleep; classifyOpenError's only-Open comment; Checkpoint stats then calls open; Import commits once and Export writes meta first; maxImportRecordBytes is 64 MiB; deterministicMarshal is in batch.go; PutEdge writes no x/ entry for an empty owner path; batchsize.go holds setOverhead, fixedAllowance and maxSizeHint; query.OpenAt opens per call; BuildReverseAdjacency, BuildImplementsIndex and buildContainsIndex exist in query/traverse.go.
- approach: Verified at head: maxFlushLockRequeues is 5 (daemon.go:75), the debounce is 2000 ms (watch/debounce.go:13), the sidecar stays on a failed flush (daemon.go:341, 363); the daemon comment names a single-writer invariant (daemon.go:239).
- approach: Verified in the module cache: pebble v2.1.7 LockDirectory (open.go:1173); x/sys v0.48.0 Renameat2 and RENAME_EXCHANGE on Linux, RenamexNp and RENAME_SWAP on darwin. go.mod is go 1.26.10, so GOMAXPROCS follows container limits and testing/synctest is available; x/sync, x/sys, goleak and rapid are direct requirements.
- approach: Verified at head: overloadfixture is multi-file (Go, C# and Java parts); routesfixture/express exists; tools/bench/gencorpus is package main with Generate and ProductionFileCount; runner duplicates it as regressionFileCount and names a nonexistent buildGencorpus; generateSyntheticCorpus builds the command; the plain-golden flag, gitExecLookPath, requireInvariants, invariant.Check and the test names that tasks 5.1, 5.5, 5.8 and 7.2 cite exist; docs/architecture/indexer-layers.html, tools/bench/BASELINE.md and baseline.json exist.
- approach: For the controller, not filed (no approach class fits, sync path only): rewireSetEdges also dissolves a stored overload set that the pass no longer declares and emits owner-to-member contains edges for it (resolver.go:806-822). D3 lists the set edges and replaced member pairs as barrier 2 outputs but never names the dissolution edges, and no artifact mentions StoredSetIDs. If the barrier 2 function omits them, a sync that dissolves a stored set loses those contains edges.
- approach: For the controller, not filed: D8 gives forEachFile the signature (ctx, n, workers, setup, body), but the pass 3 token is released by the writer goroutine after staging. The design states that release point but does not say how the token or its release travels with the unit to the writer.
- approach: For the fit lens: tools/bench/runner/main_test.go:457 also reads regressionFileCount, which task 2.1 replaces.
- approach: generated_at is a nominal time on the date of the run, because no clock read was available to this lens.
- fit: Prior fit findings fit-1, fit-2, fit-4, fit-5, fit-6 and fit-7 of review.md are resolved in this revision: link is in the declaration-walk skip, gencorpus moves under internal/, D6 names the boundary with daemon.lock, the export kind tags are exported, and link_test.go's marshal moves onto recordio.
- fit: Checked and consistent: forEachFile keeps the persistent-worker, atomic-counter shape that extract.go documents against errgroup.SetLimit; the hook variables follow gitExecLookPath and openLockRetrySleep; internal/flock moves the only unix.Flock user (wtindex/lock_unix.go); RenameDir and ExchangeDirs extend fsatomic, the existing home of crash-safe file operations; scratchdir moves wtindex's seeddir; no other gob or semaphore use exists in the repo; the deterministic marshal has only the two copies D2 names (graphstore/batch.go:35, resolve/link_test.go:23).
- fit: D11's LanguageSpec.DefaultVisibility departs from symbols' language switch pattern (LastWins, Reachable), but the design names its reason: one place for the C# type default, shared with csharpVisibility. I raised no finding.
- fit: wtindex.seed still calls os.RemoveAll on an outdated checkpoint (wtindex.go:140). Task 5.5 removes only the RemoveAll in build (wtindex.go:208). That is consistent with the design.
- invariants: I1: at most W files hold a token at once in every step, W = --workers if > 0, else GOMAXPROCS(0) \| enforced at the planned semaphore.Weighted in forEachFile, released at the D8 release points (pass 3: after the writer stages the unit) \| boundary test task 5.3 TestWorkInFlightBound (W=2, release-point artifacts), task 5.8 table (-1, 0, 1, 1<<40 over 3 files), rapid TestForEachFileProperty (W -1..64, n 0..200)
- invariants: I2: a send to the writer never blocks once the worker holds a token (channel capacity min(W,n), n counts the route record) \| enforced at the planned channel size plus the release after staging \| boundary test task 5.4 TestSinkBackpressure; n=0 gives an unbuffered channel with no sender, which is covered by the 5.8 property at n=0
- invariants: I3: each batch without its last record is at or below the cap \| enforced at the planned Batch.Len() check before each add \| boundary test task 4.1 rows over the cap, exactly the cap (both sides of the equality), first record cap+1, and the rapid property
- invariants: I4: no batch is empty, and a run commits at least one batch \| enforced at none (the D4 argument fails for a cap below the 12-byte batch header and for a Close with no record; probe-batch-len) \| boundary test TestImportEdgeStreams empty row only (inv-1)
- invariants: I5: Import's metadata is in its last batch, which is its only pebble.Sync commit; a staging store commits only NoSync \| enforced at the planned Close(sync) and the deferral of the meta record (Export writes meta first, export.go) \| boundary test task 4.1 TestBatchWriterSyncLast, task 4.2 GetMeta-after-each-commit, task 4.3 TestOpenStaging Sync rejection
- invariants: I6: a frame above the ceiling is refused, a frame at the ceiling is read, and a ceiling of 0 means none \| enforced at the planned recordio reader (today export.go:185 `length > maxImportRecordBytes`) \| boundary test task 1.1 TestFrameReaderCeiling (1 KiB, 1 KiB+1), task 2.3 64 MiB+1 spill record, the rapid frame property at 127/128/16383/16384 and at 0, read at the offset (covers a short last frame at EOF)
- invariants: I7: the spill round-trips every FileResult field (empty equals nil, Err by text plus the ErrInternal flag) \| enforced at the planned codec \| boundary test task 2.2 reflection test, rapid property with error comparer and errors.Is; FileResult and nested structs have no unexported or interface field except Err (goextract/types.go:244-294)
- invariants: I8: the record at each spill offset is the record its writer appended, under W concurrent appenders \| enforced at none named \| boundary test none at the spill level (inv-2)
- invariants: I9: all candidates of one (source, kind, target) triple are in one unit \| enforced at the planned ErrInternal unit checks; EdgeOwnerID depends only on the triple (collapse.go:20-25); resolver edges are built from ref.FromID of the file (types.go:127-208, helpers.go:19,120,252), and every route edge has a route node as its source (routes_detect.go:173) \| boundary test task 3.5 TestAssemblyRejectsForeignEdge, TestCollapseByUnitProperty with distinct Provenance, the goldens
- invariants: I10: one node per shared id, first non-default visibility in path order, exported if any copy is exported, other fields from the first copy \| enforced at the planned Builder merge (today AddNode overwrites: builder.go:135) \| boundary test task 3.2 table plus rapid fold and idempotency property, task 5.3 stored-node tests. Ids embed the file path (nodeid), so only the C# partial sentinel (csharpextract.go:218-222) and same-file route copies collide, and the merge never changes FilePath
- invariants: I11: Types and Calls write no shared data in pass 3 \| enforced by removing types.go:44, the only node-field write in resolve and symbols (grep) \| boundary test task 5.9 race step over internal/indexer
- invariants: I12: Meta.NodeCount and EdgeCount equal the stored distinct nodes and edges \| enforced at the planned view-based count, checked by invariant.Check (invariant.go:101) \| boundary test none on a tree with an id that two files emit (inv-3)
- invariants: I13: the live store is never partial, and the exchange is one syscall \| enforced at the planned fsatomic.ExchangeDirs and ReplaceStore \| boundary test task 1.3 tables, task 4.4 rows (real store, empty dir, LOCK-only dir, missing path), task 5.6 kill and failure tests
- invariants: I14: no open creates the live store between Lstat and the exchange, and no handle is open during the exchange \| enforced at the swap lock taken before pebble.Open and before any directory is created, plus pebble.LockDirectory. Lock order is writer, then swap, then Pebble everywhere; needsFileIndexBackfill closes its handle before run (sync.go:43-56, 641-645); no production caller holds a handle across calls (query.OpenAt per call, uiserver SRV-04) \| boundary test task 4.4 TestOpenWaitsForSwapLock and the three wait tests
- invariants: I15: one writer per store at a time \| enforced at the planned LockWriter flock (per open file description, so it also excludes a second writer in the same process) \| boundary test task 4.5, task 5.7, TestIndexReportsHeldWriterLock
- invariants: I16: the sweep never removes a held scratch directory and never creates a lock file \| enforced at scratchdir.New (lock under a temporary name with O_EXCL, rename, then the directory rename) \| boundary test task 1.2 rows at PendingMaxAge plus and minus 1 s, the held-lock row, TestNewLocksBeforeItNames
- invariants: I17: the new coverage generation is the prior one plus 1, or 1 for a missing or unreadable store, read under the writer lock \| enforced at run step 2 \| boundary test moved generation test, task 5.7. Not weakened: the abort on an unreadable store that the change removes is reachable today only from the CLI, which already tolerated it, because Sync fails earlier in needsFileIndexBackfill and wtindex wipes the store
- invariants: I18: the first error stops every goroutine of the run \| enforced at errgroup.WithContext. x/sync v0.23.0 Acquire returns ctx.Err() when the context is already done, even when tokens are free (semaphore.go), so no claim runs after a cancel \| boundary test task 5.4 with goleak and synctest.Wait
- invariants: I19: the merged visibility and the exported flag stay consistent for C# (exported iff visibility is public) \| enforced for valid C# only: the merge rule decides each one separately, so conflicting modifiers (invalid C#, CS0262) can give private and exported \| boundary test none; not filed, because the source is invalid
- invariants: Probe: probe-batch-len under the handoff directory (go run against pebble v2.1.7 from the module cache) follows the D4 rule literally. An empty batch has Len 12, Empty true and Count 0. Commits per batch: cap 0 with 2 records gives [0 1 1], cap 11 gives [0 1 1], cap 12 gives [1 1], and cap 1024 with 0 records gives [0].
- invariants: Outside this change: graphstore.Import gives each edge the x/ owner of its source (export.go:225), while run uses EdgeOwnerID, which takes the target for contains. Import has no production caller, so the gap is not filed.
- invariants: Test-feasibility notes for the other lenses: (a) D8 says the onCommit callback sees the size and record count of each batch, but TestBatchWriterSyncLast and TestImportCommitsInBatches assert which commit used pebble.Sync, so the callback also needs the sync mode. (b) Tasks 5.4 and 5.6 fail the second or third commit and assert that a batch committed first, but they do not lower batchCap; with the default 16 MiB cap a fixture tree commits one batch, so those tests fail instead of passing.
- invariants: Prior-round invariants findings inv-1 (resolve tests moved onto extract.File), inv-2 (error comparer and errors.Is), inv-3 (at most its last record) and inv-6 (rapid property on forEachFile) are resolved in this revision.
- spec-lint: Checked: every requirement in both delta specs has at least one scenario, and every scenario has WHEN or GIVEN and THEN.
- spec-lint: Checked: no BREAKING marker in proposal.md or in either delta spec, so no breaking-mismatch finding.
- spec-lint: Checked: the build targets named in tasks.md exist (task test:golden, golden:regen, changie, test:race in Taskfile.yml; tools/bench/runner, gencorpus, BASELINE.md and baseline.json in tools/bench).
- spec-lint: Checked: each requirement heading has tasks with two or more matching distinctive words.
- spec-lint: design.md Open Questions says None, so no assumed-answer finding.
- spec-lint: generated_at is the date with a 00:00:00 time, because no clock read was available to this lens.
- test-coverage: All 36 scenarios map to a task that names a test asserting the THEN condition.
- test-coverage: Tasks name tests rather than literal go test commands; treated as verification.
- test-coverage: No BREAKING marker in the proposal.
- approach gap: Not verified (needs execution, which this lens does not run): the plan-review probes on Pebble ErrorIfNotExists creating the directory and LOCK, a rename onto an empty directory failing on macOS, go-cmp refusing to compare two error values, and Pebble refusing a Sync commit with the WAL off.
- approach gap: Not verified: the name of the Pebble manifest marker that openExisting checks.
- approach gap: Read sync.go lines 1-660 and storeview.go loadStoreView; did not read pruneFileSubgraph, filesWaitingOn or containedMethodFiles in full for the D11 sync path.
- fit gap: openspec/config.yaml context holds no library-preference rows, so only the generic rows applied.
- fit gap: The proposal names no next change beyond #269 and #274. The two other active changes (shell-language-support, symbol-and-reference-completeness) do not mention the components this change touches (AddResults, LanguageSpec, Link, extract.File, stage, FileResult), so I found no forward conflict. I did not read them in full.
- invariants gap: Not checked: whether a real Pebble staging store opens and closes inside a testing/synctest bubble (TestSinkBackpressure, TestCancelStopsBlockedWorkers). A probe needs a test binary.
- invariants gap: Not checked: renameat2(RENAME_EXCHANGE) on overlayfs or older kernels, which can return EXDEV or ENOSYS rather than EINVAL. The live store stays unchanged in every case, but the error that names the cause covers only EINVAL and ENOTSUP.
- invariants gap: Checked only the edge constructions of resolve and routes_detect for I9, not those of every mainstream extractor's IntraEdges. The goldens and task 8.1 would catch an edge owned by another file.
- invariants gap: Did not read the sync reverse-adjacency and shared-id code of D11 beyond loadStoreView.
- spec-lint gap: Scenario 'Where working state ends in each pass' (index-build spec) has a WHEN that says 'it reports that a file stopped holding working state', which names no concrete trigger beyond the hook. Not counted as a finding because the input tree and worker count are concrete.
