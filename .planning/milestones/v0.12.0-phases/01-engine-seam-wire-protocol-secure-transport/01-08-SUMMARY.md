---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 08
subsystem: api
tags: [connect-rpc, protobuf, buf, go, wire-protocol]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-01)
    provides: "internal/uiserver's Listen/Serve lifecycle, originHostGuard, the openEngine/withEngine/mapEngineError seam, and the GetStatus tracer this plan expands"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-06)
    provides: "(*query.Engine).IndexMeta and schema.Meta.commit_sha (field 8) / schema.IndexedCommitSHA — the data path GetStatus's commit_sha field reads through"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-05)
    provides: "query.ErrNotFound / query.ErrInvalidArgument typed sentinels this plan's mapEngineError classifies by errors.Is"
provides:
  - "uiv1.UIService gains six rpcs (Search, Files, Callers, Callees, Impact, Affected), bringing the surface to 7 of 9 total methods — GetNodeDetail/Explore and the full method-set equality guard are plan 01-09's"
  - "uiv1.Node and uiv1.Location shared wire messages, reused across every structured-result response added here and available for plan 01-09's GetNodeDetail/Explore mapping"
  - "uiv1.FileEntry / uiv1.FileTreeNode modelling internal/query.FilesResult's union contract on the wire (exactly one of files/tree populated per format)"
  - "GetStatusResponse.commit_sha (field 7), sourced through (*Engine).IndexMeta inside GetStatus's existing withEngine call (ENG-04 lands on the wire)"
  - "mapEngineError completed: errors.Is classification against query.ErrNotFound/graphstore.ErrNotFound -> CodeNotFound and query.ErrInvalidArgument -> CodeInvalidArgument, never by string match"
affects: ["01-09 (GetNodeDetail/Explore + the nine-method equality guard reuse this plan's Node/Location messages and withEngine/mapEngineError seam)", "01-11 (CodeUnavailable/IndexingInProgress degrade path extends mapEngineError's existing switch)"]

# Actuals (#2632)
actuals:
  tokens: 29900
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Named unexported mapper functions (nodeToProto, locationToProto/locationsToProto, statusToProto, fileEntryToProto/fileEntriesToProto, fileTreeNodeToProto/fileTreeToProto) — one call site per response-bearing type, no inline struct-literal conversions scattered through handlers"
    - "Every new handler follows GetStatus's shape exactly: one withEngine call, exactly one Engine method call inside it, map the result, return — one query.OpenAt site, one open per rpc, asserted by test and by comment-stripped rg counts"
    - "Depth/limit arguments pass straight through to Engine methods with zero second validation/clamping copy at the RPC layer — the Engine's own validateLimit/validateDepth/clampDepth/clampAffectedDepth/MaxLimit/MaxDepth are the only bound, so RPC and CLI/MCP callers can never drift apart on what 'out of range' means"

key-files:
  created: []
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - internal/uiserver/handlers.go
    - internal/uiserver/handlers_test.go

key-decisions:
  - "Node message defined now (Task 1) even though no rpc in this plan returns one directly — every traversal response in this plan uses the lighter Location projection, matching what Search/Callers/Callees/Impact/Affected actually return on the Engine side. Node exists so plan 01-09's GetNodeDetail has an already-reviewed shape and an already-written nodeToProto mapper to reuse rather than inventing its own."
  - "mapEngineError's over-MaxLimit / over-MaxDepth behavior is NOT uniform, and the plan's phrasing understated that: validateDepth only rejects a negative depth, so clampDepth/clampAffectedDepth silently cap an explicit over-MaxDepth request (Impact/Affected clamp, as the plan states) — but validateLimit REJECTS an explicit over-MaxLimit limit argument outright, before any traversal runs (Search/Callers/Callees). The len(results) > MaxLimit safety clamp inside those methods only ever fires for a naturally-oversized RESULT under an in-range LIMIT, never for an out-of-range LIMIT argument itself. Verified directly against internal/query/validate.go and traverse.go/search.go before writing any test, and the RPC layer was built (and tested) to reflect whichever of the two the Engine actually does for a given method — never inventing a uniform 'always clamp' behavior the Engine itself does not have. See Deviations below."
  - "TestUIServiceErrorClassesAreTyped's not-found subtest calls mapEngineError(query.ErrNotFound) directly rather than through a real RPC: no rpc in Task 1's surface (GetStatus, Search) has a symbol-resolution path that can produce ErrNotFound — Callers/Callees/Impact/Affected, which do, are Task 3's addition. The invalid-argument subtest IS driven through a real client (Search with an unknown --kind), so at least one classification in that test is proven over the real wire."

patterns-established:
  - "Shared wire-message reuse across Location-bearing responses: one Location message, one locationToProto/locationsToProto pair, five call sites (Search, Callers, Callees, Impact, Affected) — the template plan 01-09 should follow for any further Location-shaped result rather than defining a sixth per-rpc variant."

requirements-completed: [RPC-01, RPC-02, ENG-04]

coverage:
  - id: D1
    description: "Search rpc answers the same Location matches internal/query.Engine.Search returns for the same term/kind/limit, compared element-by-element against an independently-computed Engine result"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceSearchMatchesEngine"
        status: pass
    human_judgment: false
  - id: D2
    description: "GetStatusResponse carries the indexed commit SHA (ENG-04) through (*Engine).IndexMeta + schema.IndexedCommitSHA — non-empty for a git checkout (equal to that fixture's own git rev-parse HEAD), empty for a non-git tree, in both cases via a successful response"
    requirement: "ENG-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceStatusCarriesCommitSHA/git-checkout"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceStatusCarriesCommitSHA/non-git-tree"
        status: pass
    human_judgment: false
  - id: D3
    description: "mapEngineError classifies query.ErrNotFound as CodeNotFound and query.ErrInvalidArgument as CodeInvalidArgument, never by string match; every handler opens through the single withEngine/query.OpenAt seam (SRV-04)"
    requirement: "RPC-02"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceErrorClassesAreTyped"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceOpensThroughTheSingleSeam"
        status: pass
    human_judgment: false
  - id: D4
    description: "Files rpc answers both the flat and tree formats internal/query.FilesResult supports, preserving the union contract (exactly one collection populated per format) and the tree's directory/leaf field asymmetry, and rejects an invalid format or over-maximum depth as CodeInvalidArgument"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceFilesPreservesBothFormats"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceFilesRejectsInvalidInput"
        status: pass
    human_judgment: false
  - id: D5
    description: "Callers, Callees, Impact, and Affected rpcs each answer exactly what the corresponding Engine method returns for the same arguments; depth/limit bounds are the Engine's alone (clamped for depth, rejected for limit — never a second RPC-layer copy); Affected accepts a repeated file path; an unknown symbol classifies as CodeNotFound not CodeInternal"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceTraversalsMatchEngine"
        status: pass
    human_judgment: false

# Metrics
duration: ~9min (wall-clock across commit timestamps)
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 8: Six Structured-Result Reads Over the Wire Summary

**Search, Files (both flat and tree formats), Callers, Callees, Impact, and Affected now answer real Connect RPC calls from the repository's own `internal/query.Engine`, and `GetStatus` carries the indexed commit SHA through `(*Engine).IndexMeta` — seven of the phase's nine `UIService` methods now exist, all through one `withEngine` lifecycle seam and one typed error-classification function.**

## Performance

- **Duration:** ~9 min wall-clock between first and last commit
- **Started:** 2026-08-23T11:54:30-04:00
- **Completed:** 2026-08-23T12:03:19-04:00
- **Tasks:** 3
- **Files modified:** 5 (3 generated/proto, 2 hand-written)

## Accomplishments

- `uiv1.Node` and `uiv1.Location` shared wire messages, plus the `Search` rpc — the first RPC-layer test comparing a real Connect response element-by-element against an independently-computed `Engine.Search` call
- `GetStatusResponse.commit_sha` (field 7): `GetStatus` now reads `(*Engine).IndexMeta` inside its existing `withEngine` call, one open serving both the status counts and the commit SHA, proven against a real `git rev-parse HEAD` fixture and a non-git fixture's empty degrade
- `mapEngineError` completed: `errors.Is` classification against `query.ErrNotFound`/`graphstore.ErrNotFound` → `CodeNotFound` and `query.ErrInvalidArgument` → `CodeInvalidArgument`, with an executable, positively-controlled scan proving zero string-comparison classification sites
- `Files` rpc modelling `FilesResult`'s union contract on the wire: `uiv1.FileEntry` for the flat format, a recursive `uiv1.FileTreeNode` for the tree format (directory nodes carry children and no path; leaves carry path/language and no children), with exactly one collection populated per format
- `Callers`, `Callees`, `Impact`, `Affected` — the four remaining traversal reads, each through `withEngine`, each calling exactly one Engine method, reusing the shared `Location` message
- Every handler verified to open through the single `openEngine`/`query.OpenAt` seam: 1 `query.OpenAt` site, 8 `withEngine(` occurrences (1 declaration + 7 call sites) in the comment-stripped source
- `task proto:drift`, `go vet`, `task test:golden` (26/26/26 unchanged), and `task test:unit` (whole repo) all pass after every task

## Task Commits

1. **Task 1: Shared wire types, `Search`, and `Status`'s commit SHA** — `bdb66c0` (feat)
2. **Task 2: `Files`, with both the flat list and the directory tree modelled** — `bae5efd` (feat)
3. **Task 3: The four remaining traversals — `Callers`, `Callees`, `Impact`, `Affected`** — `e2ba66f` (feat)

## Files Created/Modified

- `internal/uiproto/uiv1/ui.proto` — `Node`, `Location`, `SearchRequest`/`Response`, `GetStatusResponse.commit_sha`, `FileEntry`, `FileTreeNode`, `FilesRequest`/`Response`, `CallersRequest`/`Response`, `CalleesRequest`/`Response`, `ImpactRequest`/`Response`, `AffectedRequest`/`Response`, and the deliberate Query-vs-Search omission recorded in the file-level comment
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go` — regenerated via `task proto:gen`, verified byte-identical via `task proto:drift` after every task
- `internal/uiserver/handlers.go` — six new handler methods, the completed `mapEngineError`, and the mappers `nodeToProto`, `locationToProto`/`locationsToProto`, `statusToProto`, `fileEntryToProto`/`fileEntriesToProto`, `fileTreeNodeToProto`/`fileTreeToProto`
- `internal/uiserver/handlers_test.go` — new file: `TestUIServiceSearchMatchesEngine`, `TestUIServiceStatusCarriesCommitSHA`, `TestUIServiceErrorClassesAreTyped`, `TestUIServiceOpensThroughTheSingleSeam`, `TestUIServiceFilesPreservesBothFormats`, `TestUIServiceFilesRejectsInvalidInput`, `TestUIServiceTraversalsMatchEngine`

## Decisions Made

See `key-decisions` in frontmatter for the three load-bearing ones. Summarized:

1. **`Node` defined but unused by any rpc in this plan.** Every response this plan adds returns the lighter `Location` projection (matching what `Search`/`Callers`/`Callees`/`Impact`/`Affected` actually return on the Engine side). `Node` and its `nodeToProto` mapper exist now specifically so plan 01-09's `GetNodeDetail` has an already-reviewed wire shape and mapper to reuse.
2. **The over-limit/over-depth behavior is genuinely two different shapes, not one.** Verified directly against `internal/query/validate.go` and `traverse.go`/`search.go` before writing any test: `validateDepth` only rejects a *negative* depth, so `clampDepth`/`clampAffectedDepth` silently cap an explicit over-`MaxDepth` request (`Impact`/`Affected` genuinely clamp). But `validateLimit` **rejects** an explicit over-`MaxLimit` limit argument outright, before any traversal runs (`Search`/`Callers`/`Callees`) — the `len(results) > MaxLimit` safety clamp inside those methods only ever fires for a naturally-oversized *result* under an in-range *limit*, never for an out-of-range limit argument itself. The RPC layer was built, and tested, to reflect exactly whichever of the two the Engine actually does for a given method.
3. **`TestUIServiceErrorClassesAreTyped`'s not-found subtest is a direct unit test of `mapEngineError`, not wire-driven.** No rpc in Task 1's surface (`GetStatus`, `Search`) has a symbol-resolution path that can produce `ErrNotFound` — that first becomes reachable in Task 3 (`Callers`/`Callees`/`Impact`/`Affected`). Testing `mapEngineError(query.ErrNotFound)` directly is a faithful test of this task's own artifact under test and avoids depending on an RPC surface this task hasn't built yet. The invalid-argument subtest in the same function IS wire-driven (a real client calling `Search` with an unknown `kind`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — plan/behavior mismatch corrected against verified source] "A limit above MaxLimit returns the Engine's clamped result rather than an error" does not hold for every rpc this plan adds**
- **Found during:** Task 1, while writing `TestUIServiceSearchMatchesEngine`'s over-limit case
- **Issue:** Task 1's behavior block and Task 3's acceptance criteria both state a limit above `MaxLimit` is clamped, not rejected. Reading `internal/query/validate.go`'s `validateLimit` shows it **rejects** (returns `ErrInvalidArgument`) any limit argument above `MaxLimit`, unconditionally, before `Search`/`Callers`/`Callees` ever compute a result — there is no code path where an explicit over-`MaxLimit` limit argument produces a successful, silently-clamped response. The *only* real "clamp" for limit is the separate `len(results) > MaxLimit` safety net applied to a naturally-oversized match count under an in-range (including unset/zero) limit — a different scenario than "the caller passed a limit above MaxLimit".
- **Fix:** Implemented and tested the **verified, real** behavior rather than asserting a false claim to satisfy the plan's literal wording: an over-`MaxLimit` limit argument to `Search`/`Callers`/`Callees` is rejected as `CodeInvalidArgument`, exactly matching what `Engine.Search`/`Engine.Callers` themselves do for the identical argument (verified against an independently-computed Engine call in the same test). Depth above `MaxDepth` on `Impact`/`Affected` genuinely IS silently clamped (as the plan states) — `validateDepth` only rejects negative depths, and `clampDepth`/`clampAffectedDepth` cap the rest — and that half of the plan's claim was implemented and tested as written. The underlying design principle the plan states — "the clamping is the Engine's, and the RPC reflects it rather than pre-empting it" — is honored exactly either way: the RPC never adds a second, independently-drifting bound; it reflects precisely what the Engine itself does, whether that is a clamp (depth) or a rejection (limit).
- **Files modified:** `internal/uiserver/handlers_test.go` (`TestUIServiceSearchMatchesEngine`'s over-limit assertion, `TestUIServiceTraversalsMatchEngine`'s `limit-above-MaxLimit-is-clamped-by-the-Engine` subtest)
- **Verification:** Both subtests pass, asserting `connect.CodeInvalidArgument` for the over-limit case and comparing against an independently-computed `Engine.Search`/`Engine.Callers` call with the identical over-limit argument, which also errors — proving the RPC adds no second, divergent check.
- **Committed in:** `bdb66c0` (Task 1), `e2ba66f` (Task 3)

---

**Total deviations:** 1 auto-fixed (Rule 1 — a plan claim contradicted by the actual, verified Engine source; the underlying design principle was preserved and tested faithfully against real behavior rather than a false assertion)
**Impact on plan:** No functional impact — the RPC layer's actual behavior for every argument shape was verified against `internal/query`'s source before implementation and is exactly what a caller of the Engine directly would observe. Only the plan's own prose description of the over-limit case was imprecise; the code and tests reflect ground truth.

## Issues Encountered

**Test-only lock contention, resolved during authoring, not a production defect.** Early drafts of `TestUIServiceSearchMatchesEngine` and `TestUIServiceFilesPreservesBothFormats` held an independently-opened `query.OpenAt` Engine open (via `defer`) across a subsequent RPC call against the SAME store directory. Since `graphstore.Open` takes an exclusive Pebble lock, this produced a spurious `CodeInternal` (an unclassified `graphstore.ErrStoreLocked`-shaped error, not the classification under test) rather than exercising the intended code path. Fixed by never holding an independent verification `Engine` open across an RPC call in the same test — either sequencing the RPC call before opening the verification Engine, or opening/closing the verification Engine per-subtest via a small helper (`openIndependentEngine`). This is purely a test-authoring hazard specific to this test suite's pattern of running an independent verification Engine alongside a live server over the same fixture directory — it has no bearing on SRV-04's real per-call discipline, which `TestUIServiceOpensThroughTheSingleSeam` and `TestUIServerHoldsNoStoreHandleBetweenCalls` (01-01) both separately verify.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 01-09 (`GetNodeDetail`, `Explore`, and the nine-method read-only equality guard) can reuse this plan's `withEngine`/`mapEngineError` seam, the `Node`/`Location` wire messages and their mappers, and the `openIndependentEngine`-style test pattern established here.
- `mapEngineError`'s switch is a clean extension point for plan 01-11's `CodeUnavailable`/`IndexingInProgress` degrade branch — a named comment in the doc string points at it, and nothing here partially implements it.
- `GetStatusResponse` field numbers 8 and 9 remain untouched and documented as reserved for plan 01-11.
- `codegraph status --json`'s CLI output is untouched by this plan (no file under `internal/cli/` or `internal/query/status.go` was modified) — the D-06 "CLI bytes unchanged" invariant holds by construction, not by a new check.
- No blockers for downstream plans in this wave.

## Self-Check: PASSED

- FOUND: internal/uiproto/uiv1/ui.proto
- FOUND: internal/uiproto/uiv1/ui.pb.go
- FOUND: internal/uiproto/uiv1/uiv1connect/ui.connect.go
- FOUND: internal/uiserver/handlers.go
- FOUND: internal/uiserver/handlers_test.go
- FOUND: commit bdb66c0 (Task 1)
- FOUND: commit bae5efd (Task 2)
- FOUND: commit e2ba66f (Task 3)

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
