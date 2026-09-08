---
status: complete
phase: 01-engine-seam-wire-protocol-secure-transport
source: 01-01-SUMMARY.md,01-02-SUMMARY.md 01-03-SUMMARY.md,01-04-SUMMARY.md 01-05-SUMMARY.md,01-06-SUMMARY.md 01-07-SUMMARY.md,01-08-SUMMARY.md 01-09-SUMMARY.md,01-10-SUMMARY.md 01-11-SUMMARY.md
started: 2026-08-23T18:35:55Z
updated: 2026-08-23T18:56:16Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: |
  Kill any running codegraph daemon/serve process. From the repo root run
  `codegraph ui`. It binds 127.0.0.1 on an ephemeral port, prints the URL,
  and (on a TTY, non-CI) attempts to open a browser. A GetStatus call against
  that port returns the indexed commit and node/edge counts. Ctrl-C exits
  cleanly with no orphaned lock.
result: issue
reported: "Ran ./codegraph ui — printed http://127.0.0.1:52724, browser opened to that URL and showed `404 page not found` (screenshots)."
severity: minor
injected: cold-start-smoke
note: |
  Injected by judgment, not by the spec's literal pattern list — that list is
  JS/TS-centric (server.ts, app.js, migrations/, Dockerfile) and matched nothing
  here. This phase's whole deliverable is a new bootable server process, and no
  human has run it end to end.

### 2. Mutation-Proof Procedure (D3, 01-05)
expected: |
  Review the full mutation-proof transcript in 01-05-SUMMARY.md
  (#task-3-d-04-mutation-proof-full-transcript). Both extracted seams —
  01-04's NodeDetail single-def builder and 01-05's ExploreResult builder —
  were each deliberately mutated, watched FAIL against the byte-identity
  golden oracle with counts and named failing sets recorded, then reverted by
  exact reverse patch leaving the tree clean.
result: pass
verified_by: live-re-derivation
coverage_id: D3
note: |
  NOT ratified from the transcript. The maintainer flagged transcript review as
  mechanical, so the proof was re-derived live with a DIFFERENT mutation than the
  recorded one (recorded: drop CalledBy / drop blasts assembly; used here: SWAP
  Calls and CalledBy at internal/query/detail.go:138 — nastier, since both fields
  stay non-empty and a non-emptiness assertion would not catch it).
  Procedure and result:
    - pre:      git status --porcelain internal/query/ empty
    - mutation: compiles cleanly (go build exit 0), so the oracle is what catches it
    - patch:    git diff captured BEFORE running anything
    - RED:      58 --- PASS / 3 --- FAIL
    - failing set (named, discriminating — not blanket):
        TestGoldensMatchLiveEngineOutput
        TestGoldensMatchLiveEngineOutput/requests/go-node.json
        TestGoldensMatchLiveEngineOutput/requests/go-node-mcp.json
      go-node-multi.json stayed GREEN — multi-def routes through
      buildMultiDefDetail, untouched by this mutation.
    - revert:   git apply -R (never git checkout --), exit 0
    - post:     git status --porcelain internal/query/ empty
    - GREEN:    61 --- PASS / 0 --- FAIL
  Scored by counting --- PASS/--- FAIL lines, never by go test exit status
  (which was 1 under mutation by design).
source_summary: 01-05-SUMMARY.md
human_judgment_reason: |
  D-04's own scoped exception to the pass/fail conjunction: mutation runs are
  EXPECTED to exit non-zero and are scored by --- FAIL/--- PASS counts and a
  named failing set, never by go test status. No single automated signal
  captures "the procedure was carried out correctly and both reverts left the
  tree clean" — that is review-time judgment over the transcript.

### 3. Real-Process Coexistence (D10, 01-11)
expected: |
  Review Task 3 Exercises A/B/C transcripts in 01-11-SUMMARY.md. A real
  codegraph daemon and `serve --mcp` run concurrently with `codegraph ui`
  against the same store, and a fresh Open succeeds immediately once no
  handle is retained.
result: pass
verified_by: live-re-derivation
coverage_id: D10
note: |
  NOT ratified from the transcript (same maintainer objection as D3). Re-derived
  live, and the run covered THREE concurrent processes rather than the two D10
  describes:
    - /opt/homebrew/bin/codegraph serve --mcp (PID 64748, started 2026-08-22
      19:14) — a pre-existing, independently-running installed binary, not
      spawned by this session
    - ./codegraph serve --mcp (locally built, spawned here) — answered a real
      JSON-RPC initialize with capabilities
    - ./codegraph ui (locally built) — HTTP 200 on GetStatus at the same moment
  All three against the same store, with .codegraph/daemon.lock held throughout.
  Both spawned processes confirmed alive concurrently (kill -0).
  INCIDENTAL FINDING, benign and explained: GetStatus returned {stale:true, no
  commitSha} at 14:36 and {commitSha:"c90efca...", no stale} at 14:55. Cause is
  NOT a UI bug — the pre-existing Homebrew MCP server auto-synced at 14:55:03
  (.codegraph/store/MANIFEST-006289 mtime), re-indexing at HEAD. This proves
  ENG-04's commit SHA is live-derived: c90efca did not exist until this session
  committed COVERAGE.md. proto3 omits false, which is why `stale` disappears
  rather than reading false.
  NOT RUN: `codegraph daemon start`. Deliberately skipped to avoid triggering an
  index write. Noted for honesty: an auto-sync write occurred anyway from the
  pre-existing server, so that caution did not achieve what it intended.
  ALSO OBSERVED (pre-existing, not a Phase 1 issue): two orphaned
  .codegraph/.daemon.lock.tmp-* files dated Aug 13 and Aug 15.
source_summary: 01-11-SUMMARY.md
human_judgment_reason: |
  Exercises A/B/C drive real codegraph binaries as separate OS processes
  (daemon start, serve --mcp, a throwaway holder) — outside what a go test
  invocation can assert. Transcripts are the evidence; confirming real-process
  behavior against a human's own judgment is the intended verification path.

### 4. Origin/Host exact-match guard rejects DNS-rebinding-shaped requests before any handler runs, admitted spellings never treated as interchangeable
expected: Origin/Host exact-match guard rejects DNS-rebinding-shaped requests before any handler runs, admitted spellings never treated as interchangeable
result: pass
source: automated
coverage_id: D1
source_summary: 01-01-SUMMARY.md
covered_by: internal/uiserver/originguard_test.go#TestOriginHostGuard, internal/uiserver/originguard_test.go#TestOriginHostGuardAdmitsOriginlessGET

### 5. codegraph.ui.v1.UIService protobuf/Connect wire surface with one rpc (GetStatus), generated via a new buf-driven codegen pipeline covering both proto surfaces in the repo
expected: codegraph.ui.v1.UIService protobuf/Connect wire surface with one rpc (GetStatus), generated via a new buf-driven codegen pipeline covering both proto surfaces in the repo
result: pass
source: automated
coverage_id: D2
source_summary: 01-01-SUMMARY.md
covered_by: internal/uiserver/server_test.go#TestUIServerServesGetStatusEndToEnd

### 6. uiserver.Listen binds and publishes a connectable URL before uiserver.Serve blocks; Serve returns nil (not http.ErrServerClosed) on context cancellation within a 5-second shutdown budget
expected: uiserver.Listen binds and publishes a connectable URL before uiserver.Serve blocks; Serve returns nil (not http.ErrServerClosed) on context cancellation within a 5-second shutdown budget
result: pass
source: automated
coverage_id: D3
source_summary: 01-01-SUMMARY.md
covered_by: internal/uiserver/server_test.go#TestListenPublishesABoundPortBeforeServe, internal/uiserver/server_test.go#TestServeReturnsNilOnContextCancellation

### 7. codegraph ui command: binds, prints a working URL, opens a browser only when a human is watching (D-09 suppression), serves in the foreground until Ctrl-C, exposes only --path/-p and --no-open
expected: codegraph ui command: binds, prints a working URL, opens a browser only when a human is watching (D-09 suppression), serves in the foreground until Ctrl-C, exposes only --path/-p and --no-open
result: pass
source: automated
coverage_id: D4
source_summary: 01-01-SUMMARY.md
covered_by: internal/cli/ui_test.go#TestUICommandFlagSetIsExactlyPathAndNoOpen, internal/cli/ui_test.go#TestShouldOpenBrowserSuppression, internal/cli/ui_test.go#TestUICommandPrintsConnectableURLBeforeServing, CI=1 codegraph ui --path <fixture> and the same command with stdout piped, run by hand: URL printed, no browser launched, exit 0 on SIGINT

### 8. Connect handlers mount on net/http via a per-call Engine open/close discipline — no store handle survives between rpc calls
expected: Connect handlers mount on net/http via a per-call Engine open/close discipline — no store handle survives between rpc calls
result: pass
source: automated
coverage_id: D5
source_summary: 01-01-SUMMARY.md
covered_by: internal/uiserver/server_test.go#TestUIServerHoldsNoStoreHandleBetweenCalls, internal/uiserver/server_test.go#TestUIServiceHoldsNoStoreTypedField

### 9. Shared capture-spec package (internal/goldenspec) eliminates the duplicate locked-args table and MCP call shape previously declared in both gocapture/main.go and behavioral_test.go
expected: Shared capture-spec package (internal/goldenspec) eliminates the duplicate locked-args table and MCP call shape previously declared in both gocapture/main.go and behavioral_test.go
result: pass
source: automated
coverage_id: D1
source_summary: 01-02-SUMMARY.md
covered_by: internal/goldenspec/spec_test.go#TestLockedCorpusArgsAreTheFrozenValues, testdata/golden/behavioral_test.go#TestExploreCLIMatchesMCP, testdata/golden/behavioral_test.go#TestNodeCLIMatchesMCP

### 10. Byte-identity oracle over all 26 frozen golden pairs, proven able to fail via a hand-run mutation with the failing set measured and every survivor confirmed called-by-free
expected: Byte-identity oracle over all 26 frozen golden pairs, proven able to fail via a hand-run mutation with the failing set measured and every survivor confirmed called-by-free
result: pass
source: automated
coverage_id: D2
source_summary: 01-02-SUMMARY.md
covered_by: testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput, testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutputIsNonVacuous

### 11. pendingWriter decrements only on complete outbound lines classified as JSON-RPC responses; a server-initiated notification never decrements
expected: pendingWriter decrements only on complete outbound lines classified as JSON-RPC responses; a server-initiated notification never decrements
result: pass
source: automated
coverage_id: D1
source_summary: 01-03-SUMMARY.md
covered_by: internal/mcp/pending_writer_test.go#TestPendingWriterDecrementsOnlyResponses, internal/mcp/pending_writer_test.go#TestPendingWriterDrainsEarlyWithoutFix

### 12. The classifier's input is only the bytes the underlying writer actually reported as written (b[:n]), never the full buffer on a short write
expected: The classifier's input is only the bytes the underlying writer actually reported as written (b[:n]), never the full buffer on a short write
result: pass
source: automated
coverage_id: D2
source_summary: 01-03-SUMMARY.md
covered_by: internal/mcp/pending_writer_test.go#TestPendingWriterClassifiesOnlyBytesActuallyWritten

### 13. pendingWriter forwards every byte before classifying, and forwarding + classification are one serialized critical section so wire order and classification order cannot diverge under concurrency
expected: pendingWriter forwards every byte before classifying, and forwarding + classification are one serialized critical section so wire order and classification order cannot diverge under concurrency
result: pass
source: automated
coverage_id: D3
source_summary: 01-03-SUMMARY.md
covered_by: internal/mcp/pending_writer_test.go#TestPendingWriterForwardsBeforeClassifying, internal/mcp/pending_writer_test.go#TestPendingWriterSerializesWriteAndClassification, internal/mcp/pending_writer_test.go#TestPendingWriterDecrementsPerLineAcrossWriteBoundaries

### 14. The pending counter never goes negative under any interleaving, and refused decrements are counted via pendingUnderflows rather than silently absorbed
expected: The pending counter never goes negative under any interleaving, and refused decrements are counted via pendingUnderflows rather than silently absorbed
result: pass
source: automated
coverage_id: D4
source_summary: 01-03-SUMMARY.md
covered_by: internal/mcp/pending_writer_test.go#TestPendingWriterNeverGoesNegative

### 15. The toolslist-repeat wire-oracle ordering flake is proven separable from FIX-01: with zero notification traffic present, two pipelined tools/list calls are both answered exactly once, matched by JSON-RPC id
expected: The toolslist-repeat wire-oracle ordering flake is proven separable from FIX-01: with zero notification traffic present, two pipelined tools/list calls are both answered exactly once, matched by JSON-RPC id
result: pass
source: automated
coverage_id: D5
source_summary: 01-03-SUMMARY.md
covered_by: internal/mcp/pending_writer_test.go#TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications

### 16. NodeDetail sum type extracted from Node(), covering all three shapes (file/single-def/multi-def) with the multi-def case kept lazy; Node() reduced to a thin wrapper; all 26 frozen goldens still match byte-for-byte
expected: NodeDetail sum type extracted from Node(), covering all three shapes (file/single-def/multi-def) with the multi-def case kept lazy; Node() reduced to a thin wrapper; all 26 frozen goldens still match byte-for-byte
result: pass
source: automated
coverage_id: D1
source_summary: 01-04-SUMMARY.md
covered_by: internal/query/detail_test.go#TestNodeDetailCoversAllThreeShapes, internal/query/detail_test.go#TestNodeDetailSingleVsMultiBoundary, internal/query/detail_test.go#TestNodeDetailPreservesFetcherNilness, internal/query/detail_test.go#TestNodeDetailErrorsMatchNode, internal/query/detail_test.go#TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates, internal/query/detail_test.go#TestMultiDefReverseAdjacencyBuiltOnce, internal/query/detail_test.go#TestNodeErrorStringsAreUnchanged, testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput

### 17. ExploreFileGroup/ExploreBlast exported package-wide; ExploreResult extracted covering the populated case and all five zero-match early returns; Explore() reduced to a thin wrapper; ExploreDetail exported as the ENG-02 seam; all 26 frozen goldens still match byte-for-byte
expected: ExploreFileGroup/ExploreBlast exported package-wide; ExploreResult extracted covering the populated case and all five zero-match early returns; Explore() reduced to a thin wrapper; ExploreDetail exported as the ENG-02 seam; all 26 frozen goldens still match byte-for-byte
result: pass
source: automated
coverage_id: D1
source_summary: 01-05-SUMMARY.md
covered_by: internal/query/detail_test.go#TestExploreResultZeroMatchModeCoversEveryBranch, internal/query/detail_test.go#TestExploreResultMaxFilesBoundary, internal/query/detail_test.go#TestExploreResultCarriesRenderInputsUnchanged, internal/query/detail_external_test.go#TestExploreResultComponentTypesAreExported, testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput

### 18. Every caller-reachable argument rejection and not-found error in internal/query is classifiable via errors.Is against ErrInvalidArgument/ErrNotFound, with byte-identical messages; the classification table is counted so it cannot silently shrink
expected: Every caller-reachable argument rejection and not-found error in internal/query is classifiable via errors.Is against ErrInvalidArgument/ErrNotFound, with byte-identical messages; the classification table is counted so it cannot silently shrink
result: pass
source: automated
coverage_id: D2
source_summary: 01-05-SUMMARY.md
covered_by: internal/query/errors_test.go#TestClassifiedErrorsPreserveTheirMessages, internal/query/errors_test.go#TestEveryReachableErrorIsClassified

### 19. schema.Meta carries the indexed commit SHA as additive field 8, following has_file_index's precedent; absent/empty degrades gracefully, never an error
expected: schema.Meta carries the indexed commit SHA as additive field 8, following has_file_index's precedent; absent/empty degrades gracefully, never an error
result: pass
source: automated
coverage_id: D1
source_summary: 01-06-SUMMARY.md
covered_by: internal/schema/meta_commit_test.go#TestMetaCommitSHARoundTrips, internal/schema/meta_commit_test.go#TestMetaCommitSHAAbsentDegrades, internal/schema/meta_commit_test.go#TestKnownMetaFieldNumbersAreStable

### 20. HEAD is resolved exactly once per operation (full index run, incremental sync) and stamped at all three PutMeta write sites; accepts both SHA-1 and SHA-256 lengths; degrades to empty on any failure, never an error
expected: HEAD is resolved exactly once per operation (full index run, incremental sync) and stamped at all three PutMeta write sites; accepts both SHA-1 and SHA-256 lengths; degrades to empty on any failure, never an error
result: pass
source: automated
coverage_id: D2
source_summary: 01-06-SUMMARY.md
covered_by: internal/indexer/commit_test.go#TestResolveHeadCommitSHA, internal/indexer/commit_test.go#TestResolveHeadCommitSHAOnNonGitTree, internal/indexer/commit_test.go#TestResolveHeadCommitSHAWithNoGitBinary, internal/indexer/commit_test.go#TestHeadIsResolvedOncePerOperation, internal/indexer/commit_test.go#TestIndexRunStampsHeadCommitSHA

### 21. (*Engine).IndexMeta gives the wire layer a real path to the stored Meta record, degrading to (nil, nil) on a graph with no Meta
expected: (*Engine).IndexMeta gives the wire layer a real path to the stored Meta record, degrading to (nil, nil) on a graph with no Meta
result: pass
source: automated
coverage_id: D3
source_summary: 01-06-SUMMARY.md
covered_by: internal/query/meta_test.go#TestIndexMetaCarriesTheStoredMeta, internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta

### 22. codegraph status output is unchanged, proven three independent ways: status.go untouched by diff, StatusResult's field set pinned, and a before/after --json capture diffed
expected: codegraph status output is unchanged, proven three independent ways: status.go untouched by diff, StatusResult's field set pinned, and a before/after --json capture diffed
result: pass
source: automated
coverage_id: D4
source_summary: 01-06-SUMMARY.md
covered_by: internal/query/meta_test.go#TestStatusResultFieldSetIsUnchanged, git diff --stat <merge-base HEAD main>..HEAD -- internal/query/status.go (empty, positive-controlled against internal/indexer/ non-empty over the same range), codegraph status --json captured for a fixture repo from both the phase-base binary and this plan's binary; diffed by hand (see below)

### 23. proto:drift regenerates both proto surfaces into a temporary tree and byte-compares against the committed files, reporting a positive count of files compared and failing on a zero/short count
expected: proto:drift regenerates both proto surfaces into a temporary tree and byte-compares against the committed files, reporting a positive count of files compared and failing on a zero/short count
result: pass
source: automated
coverage_id: D1
source_summary: 01-07-SUMMARY.md
covered_by: internal/upgrade/proto_task_test.go#TestProtoTasksExist, internal/upgrade/proto_task_test.go#TestProtoDriftGuardReportsAComparedCount, internal/upgrade/proto_task_test.go#TestProtoDriftGeneratesIntoATemporaryTree, task proto:drift (clean tree): exit 0, 'compared 3 generated files', 'all 3 generated files byte-identical...'

### 24. The drift guard has been watched fail independently against a stale internal/schema/graph.pb.go, a stale internal/uiproto/uiv1/ui.pb.go, and a header-only staleness, with every mutation restored via an exact reverse patch and the tree confirmed clean afterward
expected: The drift guard has been watched fail independently against a stale internal/schema/graph.pb.go, a stale internal/uiproto/uiv1/ui.pb.go, and a header-only staleness, with every mutation restored via an exact reverse patch and the tree confirmed clean afterward
result: pass
source: automated
coverage_id: D2
source_summary: 01-07-SUMMARY.md
covered_by: Three deliberate-RED exercises against task proto:drift, verbatim output recorded below (see '## Deliberate-RED Observations (Task 2)')

### 25. The guard runs in CI as a Taskfile-defined job body (task proto:drift), and TestWorkflowRunBodiesInvokeTask's single-definition property still holds
expected: The guard runs in CI as a Taskfile-defined job body (task proto:drift), and TestWorkflowRunBodiesInvokeTask's single-definition property still holds
result: pass
source: automated
coverage_id: D3
source_summary: 01-07-SUMMARY.md
covered_by: internal/upgrade/proto_task_test.go#TestProtoDriftTaskIsInvokedByCI, internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask

### 26. Search rpc answers the same Location matches internal/query.Engine.Search returns for the same term/kind/limit, compared element-by-element against an independently-computed Engine result
expected: Search rpc answers the same Location matches internal/query.Engine.Search returns for the same term/kind/limit, compared element-by-element against an independently-computed Engine result
result: pass
source: automated
coverage_id: D1
source_summary: 01-08-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceSearchMatchesEngine

### 27. GetStatusResponse carries the indexed commit SHA (ENG-04) through (*Engine).IndexMeta + schema.IndexedCommitSHA — non-empty for a git checkout (equal to that fixture's own git rev-parse HEAD), empty for a non-git tree, in both cases via a successful response
expected: GetStatusResponse carries the indexed commit SHA (ENG-04) through (*Engine).IndexMeta + schema.IndexedCommitSHA — non-empty for a git checkout (equal to that fixture's own git rev-parse HEAD), empty for a non-git tree, in both cases via a successful response
result: pass
source: automated
coverage_id: D2
source_summary: 01-08-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceStatusCarriesCommitSHA/git-checkout, internal/uiserver/handlers_test.go#TestUIServiceStatusCarriesCommitSHA/non-git-tree

### 28. mapEngineError classifies query.ErrNotFound as CodeNotFound and query.ErrInvalidArgument as CodeInvalidArgument, never by string match; every handler opens through the single withEngine/query.OpenAt seam (SRV-04)
expected: mapEngineError classifies query.ErrNotFound as CodeNotFound and query.ErrInvalidArgument as CodeInvalidArgument, never by string match; every handler opens through the single withEngine/query.OpenAt seam (SRV-04)
result: pass
source: automated
coverage_id: D3
source_summary: 01-08-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceErrorClassesAreTyped, internal/uiserver/handlers_test.go#TestUIServiceOpensThroughTheSingleSeam

### 29. Files rpc answers both the flat and tree formats internal/query.FilesResult supports, preserving the union contract (exactly one collection populated per format) and the tree's directory/leaf field asymmetry, and rejects an invalid format or over-maximum depth as CodeInvalidArgument
expected: Files rpc answers both the flat and tree formats internal/query.FilesResult supports, preserving the union contract (exactly one collection populated per format) and the tree's directory/leaf field asymmetry, and rejects an invalid format or over-maximum depth as CodeInvalidArgument
result: pass
source: automated
coverage_id: D4
source_summary: 01-08-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceFilesPreservesBothFormats, internal/uiserver/handlers_test.go#TestUIServiceFilesRejectsInvalidInput

### 30. Callers, Callees, Impact, and Affected rpcs each answer exactly what the corresponding Engine method returns for the same arguments; depth/limit bounds are the Engine's alone (clamped for depth, rejected for limit — never a second RPC-layer copy); Affected accepts a repeated file path; an unknown symbol classifies as CodeNotFound not CodeInternal
expected: Callers, Callees, Impact, and Affected rpcs each answer exactly what the corresponding Engine method returns for the same arguments; depth/limit bounds are the Engine's alone (clamped for depth, rejected for limit — never a second RPC-layer copy); Affected accepts a repeated file path; an unknown symbol classifies as CodeNotFound not CodeInternal
result: pass
source: automated
coverage_id: D5
source_summary: 01-08-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceTraversalsMatchEngine

### 31. GetNodeDetail answers all three internal/query.NodeDetail shapes over the wire (file/single-def/multi-def), with per-candidate multi-def calls/called-by preserved rather than flattened, going through (*Engine).NodeDetail exclusively
expected: GetNodeDetail answers all three internal/query.NodeDetail shapes over the wire (file/single-def/multi-def), with per-candidate multi-def calls/called-by preserved rather than flattened, going through (*Engine).NodeDetail exclusively
result: pass
source: automated
coverage_id: D1
source_summary: 01-09-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceNodeDetailCoversAllThreeModes, internal/uiserver/handlers_test.go#TestUIServiceNodeDetailGoesThroughEngineBuilder

### 32. The UI's own uiMultiDefCap bounds per-candidate gathering (never the reported total), reporting the TRUE total candidate count regardless of the cap, and out-of-cap candidates are listed without detail rather than read
expected: The UI's own uiMultiDefCap bounds per-candidate gathering (never the reported total), reporting the TRUE total candidate count regardless of the cap, and out-of-cap candidates are listed without detail rather than read
result: pass
source: automated
coverage_id: D2
source_summary: 01-09-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceNodeDetailCapsCandidatesAndReportsTheTotal

### 33. GetNodeDetail classifies every caller-reachable rejection with the mapped Connect code: not-found symbol -> CodeNotFound, neither symbol nor file -> CodeInvalidArgument, an unreadable within-cap candidate -> CodeInternal (failing the whole RPC, never a partial response)
expected: GetNodeDetail classifies every caller-reachable rejection with the mapped Connect code: not-found symbol -> CodeNotFound, neither symbol nor file -> CodeInvalidArgument, an unreadable within-cap candidate -> CodeInternal (failing the whole RPC, never a partial response)
result: pass
source: automated
coverage_id: D3
source_summary: 01-09-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceNodeDetailErrorClasses

### 34. Explore answers internal/query.ExploreResult over the wire through (*Engine).ExploreDetail exclusively: each group carries its own matched symbols and skeletonized flag keyed by path, blast entries map one-to-one onto ExploreResult.Blasts, and a zero-match query is a successful response (never CodeNotFound/CodeInternal)
expected: Explore answers internal/query.ExploreResult over the wire through (*Engine).ExploreDetail exclusively: each group carries its own matched symbols and skeletonized flag keyed by path, blast entries map one-to-one onto ExploreResult.Blasts, and a zero-match query is a successful response (never CodeNotFound/CodeInternal)
result: pass
source: automated
coverage_id: D4
source_summary: 01-09-SUMMARY.md
covered_by: internal/uiserver/handlers_test.go#TestUIServiceExploreGroupsCarryTheirOwnSources, internal/uiserver/handlers_test.go#TestUIServiceExploreZeroMatchIsNotAnError, internal/uiserver/handlers_test.go#TestUIServiceExploreRejectsEmptyQuery

### 35. No RPC can mutate the index: the nine-method surface is asserted by set equality in both directions (an added or removed method fails), and the field numbers are pinned by stability and coverage over the generated descriptor rather than by contiguity — both guards demonstrated failing under a deliberate mutation, then restored
expected: No RPC can mutate the index: the nine-method surface is asserted by set equality in both directions (an added or removed method fails), and the field numbers are pinned by stability and coverage over the generated descriptor rather than by contiguity — both guards demonstrated failing under a deliberate mutation, then restored
result: pass
source: automated
coverage_id: D5
source_summary: 01-09-SUMMARY.md
covered_by: internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet, internal/uiserver/readonly_test.go#TestUIServiceDeclaresNoMutatingMethod, internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique

### 36. Shared rune-boundary truncation helper (internal/textutil.TruncateOnRuneBoundary), consumed by both internal/mcp and internal/uiserver, replacing the duplicated unexported copy
expected: Shared rune-boundary truncation helper (internal/textutil.TruncateOnRuneBoundary), consumed by both internal/mcp and internal/uiserver, replacing the duplicated unexported copy
result: pass
source: automated
coverage_id: D1
source_summary: 01-10-SUMMARY.md
covered_by: internal/textutil/truncate_test.go#TestTruncateOnRuneBoundary, internal/mcp/... (go test -race ./internal/mcp/...)

### 37. Two-tier source truncation (line cap primary, byte cap secondary) with exact totals in both units, never signalled as an error
expected: Two-tier source truncation (line cap primary, byte cap secondary) with exact totals in both units, never signalled as an error
result: pass
source: automated
coverage_id: D2
source_summary: 01-10-SUMMARY.md
covered_by: internal/uiserver/truncate_test.go#TestTruncateSourceUnderBothCaps,TestTruncateSourceExceedsLineCap,TestTruncateSourceExceedsByteCap,TestTruncateSourceLineCapBoundary,TestTruncateSourceNeverSplitsARune,TestTruncateSourceEmptyInput,TestTruncateSourceTotalsAreExact,TestCountLinesSemantics

### 38. SourceBlob wired end-to-end at all three attachment points (GetNodeDetail single-def/file mode, multi-def candidates, Explore groups) through a real Connect client
expected: SourceBlob wired end-to-end at all three attachment points (GetNodeDetail single-def/file mode, multi-def candidates, Explore groups) through a real Connect client
result: pass
source: automated
coverage_id: D3
source_summary: 01-10-SUMMARY.md
covered_by: internal/uiserver/sourceblob_test.go#TestUIServiceSourceBlobTruncatesAnOversizedDefinitionSource,TestUIServiceSourceBlobOnEveryAttachmentPoint,TestUIServiceSourceBlobGroupSourceMatchesItsGroupPath,TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop,TestUIServiceSourceBlobFileModeUsesTheSameTruncationPath

### 39. Transport backstop (WithSendMaxBytes/WithReadMaxBytes) mounted on the UIService handler, strictly above the application byte cap
expected: Transport backstop (WithSendMaxBytes/WithReadMaxBytes) mounted on the UIService handler, strictly above the application byte cap
result: pass
source: automated
coverage_id: D4
source_summary: 01-10-SUMMARY.md
covered_by: internal/uiserver/truncate_test.go#TestTransportBackstopSitsAboveApplicationCap, internal/uiserver/sourceblob_test.go#TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop

### 40. 01-09's known-number field fixture extended by exactly 9 entries, every prior entry still resolving unchanged
expected: 01-09's known-number field fixture extended by exactly 9 entries, every prior entry still resolving unchanged
result: pass
source: automated
coverage_id: D5
source_summary: 01-10-SUMMARY.md
covered_by: internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique

### 41. A locked store renders as CodeUnavailable with a typed IndexingInProgress detail on every non-GetStatus RPC (D-14), never as a raw error or CodeFailedPrecondition
expected: A locked store renders as CodeUnavailable with a typed IndexingInProgress detail on every non-GetStatus RPC (D-14), never as a raw error or CodeFailedPrecondition
result: pass
source: automated
coverage_id: D1
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestNonStatusRPCsDegradeWhenAHolderNeverReleases

### 42. A holder that releases within graphstore.Open's bounded budget produces a normal successful result from every RPC, driven by a causal edge rather than a timer
expected: A holder that releases within graphstore.Open's bounded budget produces a normal successful result from every RPC, driven by a causal edge rather than a timer
result: pass
source: automated
coverage_id: D2
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget

### 43. No second retry layer exists above graphstore.Open's own bounded budget (D-15), proven by counting store opens, not by measuring time
expected: No second retry layer exists above graphstore.Open's own bounded budget (D-15), proven by counting store opens, not by measuring time
result: pass
source: automated
coverage_id: D3
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestDegradedRPCOpensTheStoreExactlyOnce

### 44. Status still answers when the store cannot be opened at all, reporting store-exists and indexing-in-progress from filesystem facts with graph-derived counts zeroed (D-16)
expected: Status still answers when the store cannot be opened at all, reporting store-exists and indexing-in-progress from filesystem facts with graph-derived counts zeroed (D-16)
result: pass
source: automated
coverage_id: D4
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestStatusDegradesOnLock

### 45. A repository with no .codegraph/ at all reports a distinct not-initialized state, never conflated with the locked state
expected: A repository with no .codegraph/ at all reports a distinct not-initialized state, never conflated with the locked state
result: pass
source: automated
coverage_id: D5
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestStatusOnUninitializedRepo

### 46. The degrade precedence (not-initialized outranks locked) is an explicit ordered check, tested with a synthetic error satisfying both sentinels since the two conditions cannot co-occur on a real filesystem
expected: The degrade precedence (not-initialized outranks locked) is an explicit ordered check, tested with a synthetic error satisfying both sentinels since the two conditions cannot co-occur on a real filesystem
result: pass
source: automated
coverage_id: D6
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestClassifyDegradePrecedence

### 47. Calling the same RPC twice against a locked store yields identical CodeUnavailable and detail both times, mutating nothing
expected: Calling the same RPC twice against a locked store yields identical CodeUnavailable and detail both times, mutating nothing
result: pass
source: automated
coverage_id: D7
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestDegradeIsIdempotent

### 48. The final-attempt boundary of graphstore.Open's bounded retry loop is deterministic and event-synchronized, distinct from the release-between-attempts case an existing test already covers
expected: The final-attempt boundary of graphstore.Open's bounded retry loop is deterministic and event-synchronized, distinct from the release-between-attempts case an existing test already covers
result: pass
source: automated
coverage_id: D8
source_summary: 01-11-SUMMARY.md
covered_by: internal/graphstore/open_lock_test.go#TestOpenSucceedsOnTheFinalAttempt

### 49. Concurrent RPCs each open, snapshot and close independently; a holder attempted only after every concurrent RPC completes acquires the store without needing a fairness guarantee
expected: Concurrent RPCs each open, snapshot and close independently; a holder attempted only after every concurrent RPC completes acquires the store without needing a fairness guarantee
result: pass
source: automated
coverage_id: D9
source_summary: 01-11-SUMMARY.md
covered_by: internal/uiserver/degrade_test.go#TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete

## Summary

total: 49
passed: 48
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-01-1
  truth: "`codegraph ui` boots and the URL it opens in the browser shows something meaningful"
  status: failed
  reason: "User reported: ran ./codegraph ui, browser opened to http://127.0.0.1:52724 and showed `404 page not found`."
  severity: minor
  test: 1
  root_cause: "internal/cli/ui.go:65 auto-opens srv.URL() (the root path) whenever TTY and non-CI, but internal/uiserver/server.go:93 mounts ONLY the Connect handler prefix on the mux — no root handler exists in Phase 1 by design (01-01-PLAN.md:350 'no SPA'). http.ServeMux therefore 404s GET /. Verified live: GET / -> 404, POST /codegraph.ui.v1.UIService/GetStatus -> 200 with real counts (nodeCount 5279, edgeCount 12281, fileCount 475), evil Origin -> 403."
  artifacts:
    - path: "internal/cli/ui.go"
      issue: "line 65 opens the root URL unconditionally; help text promises \"opens it in your default browser\""
    - path: "internal/uiserver/server.go"
      issue: "line 93 mounts only uiv1connect.NewUIServiceHandler; no root route"
  missing:
    - "Decide Phase 1 scope: suppress auto-open until an app shell exists, OR mount a minimal placeholder root, OR accept the 404 as a known Phase-1-only state and say so in the command help"
  phase_2_closes_by_construction: true
  note: "Phase 2 success criterion 1 mounts the embedded SPA on this same mux, which removes the 404 by construction. The open question is Phase-1-scoped: whether shipping auto-open pointed at an unbuilt root is acceptable in the interim."
