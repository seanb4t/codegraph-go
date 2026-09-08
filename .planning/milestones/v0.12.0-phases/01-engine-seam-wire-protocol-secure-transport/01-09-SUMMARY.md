---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 09
subsystem: api
tags: [connect-rpc, protobuf, buf, go, wire-protocol, node-detail, explore]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-04)
    provides: "internal/query.NodeDetail sum type, MultiDefDetail's lazy per-candidate Definition(n) protocol, and (*Engine).NodeDetail — the exact shapes this plan maps onto the wire"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-05)
    provides: "internal/query.ExploreResult, exported ExploreFileGroup/ExploreBlast component types, (*Engine).ExploreDetail, and the query.ErrNotFound/ErrInvalidArgument typed sentinels this plan's handlers classify through mapEngineError"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-08)
    provides: "The Node/Location shared wire messages and their mappers, the withEngine/mapEngineError seam, and the seven-of-nine UIService method surface this plan completes to nine"
provides:
  - "uiv1.UIService gains GetNodeDetail and Explore, completing the nine-method read-only surface (T-01-04)"
  - "uiv1.NodeDetailMode, GetNodeDetailRequest/Response, NodeDefinition — the discriminated wire projection of internal/query.NodeDetail's three shapes, with per-candidate multi-def calls/called-by preserved rather than flattened"
  - "internal/uiserver.uiMultiDefCap (20) — the UI's own multi-definition gather cap, independent of internal/query/render_markdown.go's nodeMultiDefHardCap"
  - "uiv1.ExploreRequest/Response, ExploreGroup, BlastEntry — the wire projection of internal/query.ExploreResult, with each group carrying its own matched symbols and skeletonized flag keyed by path"
  - "internal/uiserver/readonly_test.go: TestUIServiceMethodSetIsExactlyTheReadSet (bidirectional set equality over the 9-method surface), TestUIServiceDeclaresNoMutatingMethod (non-vacuous verb check), TestUIProtoFieldNumbersAreStableAndUnique (a 97-entry known-number fixture covering every field of every message that exists at this wave, both directions: every entry resolves AND every declared field is covered)"
  - "ui.proto's file-level intent paragraph naming plan 01-10's forthcoming SourceBlob attachment points and their next-free numbers (GetNodeDetailResponse=9, NodeDefinition=5, ExploreGroup=4) — documentation only, not a reserved clause"
affects: ["01-10 (attaches SourceBlob to GetNodeDetailResponse/NodeDefinition/ExploreGroup at the numbers this plan's fixture pins, and extends uiProtoFieldNumbers + declares uiProtoFieldFixtureLenAtPlan0110)", "01-11 (extends the same fixture with GetStatusResponse.store_exists=8/.indexing_in_progress=9 and declares uiProtoFieldFixtureLenAtPlan0111)"]

# Actuals (#2632)
actuals:
  tokens: 26368
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Discriminated wire response (mode + per-mode field groups, all-zero for the other modes) for a Go sum type with more than two shapes — GetNodeDetailResponse mirrors NodeDetail's Mode/File/Definition/Multi exactly, extending 01-08's Node/Location precedent to a three-way discriminant"
    - "Lazy-to-eager boundary crossed exactly at the cap: the mapper calls MultiDefDetail.Definition only for the first uiMultiDefCap matches, preserving the seam's no-I/O-beyond-what's-rendered invariant on the wire, not just in the CLI/MCP render path"
    - "Bidirectional known-number fixture (resolves AND covers) as the corrected, non-contiguity field-stability guard — mirrors internal/schema/meta_commit_test.go's subset-fixture shape but adds the coverage direction cycle-3 found missing there"
    - "Directory-in-place-of-file as the deterministic 'unreadable candidate' test fixture (from 01-04), reused here at the RPC layer for GetNodeDetail's per-candidate lookup-failure path"

key-files:
  created:
    - internal/uiserver/readonly_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - internal/uiserver/handlers.go
    - internal/uiserver/handlers_test.go

key-decisions:
  - "uiMultiDefCap set to 20, deliberately distinct from render_markdown.go's nodeMultiDefHardCap (16) — not because the value must differ, but because the two constants must never be the same DECLARATION, and a distinct value makes that independence visible in tests rather than merely asserted in a comment."
  - "Plan 01-10's SourceBlob field numbers are NOT pre-allocated as a reserved clause anywhere — recorded as documentation only in ui.proto's file-level comment, per the plan's own explicit withdrawal of the comment+reserved approach (cycle-2 C2-1). The only enforceable cross-wave protection is this plan's own known-number fixture, which pins every number IT allocates against renumbering; 01-10 extends the fixture rather than pre-empting it."
  - "The known-number fixture's scope is 'every field of every message in the descriptor at this wave', not 'only fields this plan declares' — per the plan's corrected scope rule (cycle-3 H1). Concretely this means the fixture (97 entries) includes GetStatusRequest/GetStatusResponse (fields 1-7, allocated by 01-01/01-08) and IndexingInProgress (allocated by 01-01), neither of which this plan touches, alongside this plan's own 26 new entries across 7 new messages (GetNodeDetailRequest, NodeDefinition, GetNodeDetailResponse, ExploreRequest, ExploreGroup, BlastEntry, ExploreResponse)."
  - "TestUIServiceExploreGroupsCarryTheirOwnSources' 'source-matches-group-path' subtest checks that each WIRE group's path is a valid, non-empty lookup key into the independently-computed ExploreResult.Sources map — not that the wire itself carries source bytes (it doesn't yet; that is plan 01-10's job). This is the forward-compatibility proof that the keying survives the mapping, ahead of the byte payload landing."
  - "GetNodeDetail's per-candidate multi-def gather fixtures use synthetic cross-package overloaded functions (same function name declared in N separate Go packages, each with its own uniquely-named unexported helper) rather than the CLI's real corpora — Go tolerates the same name across packages (unlike within one package), and this gives deterministic, arbitrarily-sized overload counts for the cap test without depending on a real corpus happening to contain one."
  - "The 'unreadable within-cap candidate' error-class test replaces EVERY candidate's file with a directory of the same name post-indexing (not just one), because match order across the two candidates is not this test's claim to make — this guarantees a within-cap failure regardless of internal enumeration order, mirroring the deterministic-by-construction discipline of 01-04's own version of this technique."

patterns-established:
  - "Pattern: for an RPC's <verify> gate that includes 'task proto:gen && ... && git status --porcelain <path> is empty', the check is a POST-COMMIT confirmation, not a pre-commit gate — the working tree is legitimately dirty in that path while the task's own uncommitted edits exist. Run the test-suite portion of the gate first to decide whether to commit, commit the task, then re-run the full gate verbatim as a post-commit idempotency/completeness confirmation. This matches the plan's own stated Gate Conventions vacuity shape 5 ('under atomic-commit-per-task the working tree is clean at verification time')."

requirements-completed: [RPC-01, RPC-02, SRV-03]

coverage:
  - id: D1
    description: "GetNodeDetail answers all three internal/query.NodeDetail shapes over the wire (file/single-def/multi-def), with per-candidate multi-def calls/called-by preserved rather than flattened, going through (*Engine).NodeDetail exclusively"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceNodeDetailCoversAllThreeModes"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceNodeDetailGoesThroughEngineBuilder"
        status: pass
    human_judgment: false
  - id: D2
    description: "The UI's own uiMultiDefCap bounds per-candidate gathering (never the reported total), reporting the TRUE total candidate count regardless of the cap, and out-of-cap candidates are listed without detail rather than read"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceNodeDetailCapsCandidatesAndReportsTheTotal"
        status: pass
    human_judgment: false
  - id: D3
    description: "GetNodeDetail classifies every caller-reachable rejection with the mapped Connect code: not-found symbol -> CodeNotFound, neither symbol nor file -> CodeInvalidArgument, an unreadable within-cap candidate -> CodeInternal (failing the whole RPC, never a partial response)"
    requirement: "RPC-02"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceNodeDetailErrorClasses"
        status: pass
    human_judgment: false
  - id: D4
    description: "Explore answers internal/query.ExploreResult over the wire through (*Engine).ExploreDetail exclusively: each group carries its own matched symbols and skeletonized flag keyed by path, blast entries map one-to-one onto ExploreResult.Blasts, and a zero-match query is a successful response (never CodeNotFound/CodeInternal)"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceExploreGroupsCarryTheirOwnSources"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceExploreZeroMatchIsNotAnError"
        status: pass
      - kind: unit
        ref: "internal/uiserver/handlers_test.go#TestUIServiceExploreRejectsEmptyQuery"
        status: pass
    human_judgment: false
  - id: D5
    description: "No RPC can mutate the index: the nine-method surface is asserted by set equality in both directions (an added or removed method fails), and the field numbers are pinned by stability and coverage over the generated descriptor rather than by contiguity — both guards demonstrated failing under a deliberate mutation, then restored"
    requirement: "SRV-03"
    verification:
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceMethodSetIsExactlyTheReadSet"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIServiceDeclaresNoMutatingMethod"
        status: pass
      - kind: unit
        ref: "internal/uiserver/readonly_test.go#TestUIProtoFieldNumbersAreStableAndUnique"
        status: pass
    human_judgment: false

# Metrics
duration: ~75min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 9: GetNodeDetail, Explore & Read-Only-By-Construction Summary

**`GetNodeDetail` and `Explore` land on the wire through the exact seams plans 01-04/01-05 extracted, completing the nine-method `UIService` surface, whose read-only property and field-number stability are now asserted by bidirectional set/descriptor equality rather than stated intention.**

## Performance

- **Duration:** ~75 min
- **Completed:** 2026-08-23
- **Tasks:** 3
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `GetNodeDetail` answers all three `internal/query.NodeDetail` shapes (file/single-def/multi-def) over the wire through `(*Engine).NodeDetail` exclusively — no markdown parsing, no second gather path (D-01).
- Per-candidate multi-definition data survives the mapping intact: `NodeDefinition` carries each candidate's own node, calls, and called-by, never a shared or collapsed value — proved by an ID-keyed comparison against `MultiDefDetail.Definition`, not a positional one.
- `uiMultiDefCap` (20) is a named constant bounding per-candidate gathering; the response always reports the TRUE total match count regardless of the cap, so a client can render "showing N of M". Proved with a 22-package synthetic overload fixture: exactly 20 candidates carry gathered detail, exactly 2 are listed without it.
- A per-candidate lookup failure (an unreadable within-cap candidate) fails the WHOLE `GetNodeDetail` RPC through `mapEngineError`, never a partially-populated response — matching `Node()`'s own behavior exactly.
- `Explore` answers `internal/query.ExploreResult` over the wire through `(*Engine).ExploreDetail` exclusively: each `ExploreGroup` carries its own matched symbols and skeletonized flag, looked up by the group's own path — never by index — and `BlastEntry` mirrors `ExploreBlast` field-for-field. A zero-match query is a SUCCESSFUL response with the empty marker, never `CodeNotFound`/`CodeInternal`.
- `ui.proto`'s file-level comment documents plan 01-10's forthcoming `SourceBlob` attachment points (`GetNodeDetailResponse`=9, `NodeDefinition`=5, `ExploreGroup`=4) as intent, explicitly NOT as a `reserved` clause — a descriptor carries no comments, so pinning an undeclared number could only fail this plan, pass vacuously, or block 01-10's use of it.
- `TestUIServiceMethodSetIsExactlyTheReadSet` asserts the nine-method `UIService` surface by set equality in BOTH directions (an added OR removed method fails), replacing a negative-only verb-blacklist guard that would pass vacuously the moment its anchor stopped matching.
- `TestUIProtoFieldNumbersAreStableAndUnique` pins a 97-entry known-number fixture — every field of every message that exists in the generated descriptor at this wave, including `GetStatusRequest`/`GetStatusResponse` (allocated by earlier plans) — checked in BOTH directions: every fixture entry resolves, AND every declared field is covered. Both guards were watched fail under a deliberate mutation (a 10th method name; a nonexistent `message.field`), the failure output recorded, and the file restored — not reverted by discarding it.

## Task Commits

Each task was committed atomically:

1. **Task 1: `GetNodeDetail` — all three modes on the wire, with per-candidate multi-def** - `f36bf82` (feat)
2. **Task 2: `Explore` — groups carrying their own sources and blast entries** - `e9c9ae6` (feat)
3. **Task 3: Assert read-only by construction and pin the field numbers (SRV-03)** - `1d53f68` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/uiproto/uiv1/ui.proto` - `NodeDetailMode`, `GetNodeDetailRequest`/`Response`, `NodeDefinition`, `ExploreRequest`/`Response`, `ExploreGroup`, `BlastEntry`; the `GetNodeDetail`/`Explore` rpcs added to `UIService`; the file-level intent paragraph naming plan 01-10's forthcoming `SourceBlob` attachment points
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go` - regenerated via `task proto:gen`, verified byte-identical via `task proto:drift` after every task
- `internal/uiserver/handlers.go` - `uiMultiDefCap`, `nodesToProto`, `nodeDetailModeToProto`, `nodeDefinitionToProto`, `nodeDetailToProto`, the `GetNodeDetail` handler; `exploreGroupToProto`/`exploreGroupsToProto`, `blastEntryToProto`/`blastEntriesToProto`, `exploreResultToProto`, the `Explore` handler
- `internal/uiserver/handlers_test.go` - `buildOverloadedFixture` (synthetic cross-package overload generator), `TestUIServiceNodeDetailCoversAllThreeModes`, `TestUIServiceNodeDetailGoesThroughEngineBuilder`, `TestUIServiceNodeDetailCapsCandidatesAndReportsTheTotal`, `TestUIServiceNodeDetailErrorClasses`; `copyBehavioralFixture` (reproduced from `internal/query/explore_test.go`'s unexported helper), `TestUIServiceExploreGroupsCarryTheirOwnSources`, `TestUIServiceExploreZeroMatchIsNotAnError`, `TestUIServiceExploreRejectsEmptyQuery`
- `internal/uiserver/readonly_test.go` - new file: `wantUIServiceMethods`, `TestUIServiceMethodSetIsExactlyTheReadSet`, `mutatingVerbs`, `TestUIServiceDeclaresNoMutatingMethod`, `uiProtoFieldNumber`, `uiProtoFieldFixtureLenAtPlan0109`, `uiProtoFieldNumbers` (97 entries), `TestUIProtoFieldNumbersAreStableAndUnique`

## Decisions Made

See `key-decisions` in frontmatter for the six load-bearing ones. Summarized:

1. **`uiMultiDefCap = 20`**, deliberately its own declaration independent of `nodeMultiDefHardCap` (16) — the point is the independent declaration, not a specific numeric distance.
2. **No pre-allocation of 01-10's `SourceBlob` field numbers** via any mechanism other than documentation in `ui.proto`'s file-level comment. The only enforceable cross-wave protection is this plan's own known-number fixture pinning the numbers it DOES allocate.
3. **The known-number fixture's scope is the whole descriptor at this wave**, not just this plan's own declarations — 97 entries across 19 pre-existing + 7 new messages, closing the gap cycle-3 found where `GetStatusResponse` was pinned by nothing.
4. **The "source-matches-group-path" subtest checks keying, not payload** — `ExploreGroup` carries no source bytes yet; the subtest proves the wire's `path` field is a valid, resolvable key into `ExploreResult.Sources`, which is the property 01-10's attachment depends on.
5. **Synthetic cross-package overload fixtures**, not real corpora, for GetNodeDetail's multi-def tests — deterministic, arbitrarily-sized, and does not depend on a real corpus happening to contain an overloaded symbol of the right cardinality.
6. **The "unreadable within-cap candidate" test corrupts every candidate's file**, not just one, so the test's claim does not depend on internal enumeration order.

## Deviations from Plan

### Auto-fixed Issues

None — Rules 1/2/3 were not triggered. The one significant EXECUTION-ORDER finding (not a deviation from the plan's substance) is recorded below because it affects how a future reader should interpret this plan's own `<verify>` gates.

**1. [Process finding, not a Rule 1/2/3 fix] The `git status --porcelain internal/uiproto/` check inside each task's `<verify>` gate is a POST-COMMIT confirmation, not a pre-commit gate**
- **Found during:** Task 1, first attempt to run the literal `<verify>` command before committing
- **Issue:** `git status --porcelain internal/uiproto/`, run BEFORE a task's own edits are committed, is non-empty by construction (the task's own uncommitted `ui.proto`/`ui.pb.go`/`ui.connect.go` changes show as modified relative to HEAD) — it can never read empty pre-commit for a task that touches these files, regardless of whether `task proto:gen` is idempotent.
- **Resolution:** Interpreted per the plan's own "Gate conventions" canon, vacuity shape 5: "under atomic-commit-per-task the working tree is clean AT VERIFICATION TIME" — i.e., verification-time is understood to be POST-commit for this specific check. Ran the test-suite portion of each task's gate first (to decide whether to commit), committed the task, then re-ran the FULL literal `<verify>` command (including the `git status --porcelain` clause) as a post-commit confirmation — which passed cleanly for all three tasks. No plan text was changed; no gate was weakened.
- **Files modified:** None (an execution-sequencing finding, not a code change)
- **Verification:** All three tasks' `<verify>` gates, run verbatim post-commit, passed: Task 1 `exit=0 PASS lines: 10` (floor 7), Task 2 `exit=0 PASS lines: 6` (floor 5), Task 3 `exit=0 PASS lines: 3` (floor 3, zero headroom).
- **Committed in:** N/A (process-only; no separate commit)

---

**Total deviations:** 0 auto-fixed. One process/interpretation finding recorded above, with zero code impact.
**Impact on plan:** None — every gate specified in the plan was run verbatim and passed at the correct point in the task sequence.

## Issues Encountered

None beyond the verify-gate sequencing finding documented above, resolved during execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-10's one-way `SourceBlob` checkpoint can now lock an attachment point a human can already read in the tree: `GetNodeDetailResponse`, `NodeDefinition` and `ExploreGroup` all exist, with their next-free numbers (9, 5, 4 respectively) documented in `ui.proto`'s file-level comment.
- Plan 01-10 must EXTEND `internal/uiserver/readonly_test.go`'s `uiProtoFieldNumbers` fixture (append its nine new source-field entries) and declare `uiProtoFieldFixtureLenAtPlan0110 = uiProtoFieldFixtureLenAtPlan0109 + 9`, never rewriting the existing 97 entries or restating a bare literal.
- Plan 01-11 will do the same for `GetStatusResponse.store_exists = 8` / `.indexing_in_progress = 9`, declaring `uiProtoFieldFixtureLenAtPlan0111 = uiProtoFieldFixtureLenAtPlan0110 + 2`.
- The nine-method `UIService` surface is now complete and pinned by `TestUIServiceMethodSetIsExactlyTheReadSet`'s bidirectional set equality — no later plan in this phase should need to touch the RPC method set itself, only response field additions.
- All plan-level `<verification>` items independently re-confirmed at the end of this plan: `go build ./...` clean; `go test -race -count=1 ./internal/uiserver/...` passes; `task proto:drift` reports `compared 3 generated files` / all byte-identical; `task test:unit` passes across the whole repo; `task test:golden` reports `attempted=26 completed=26 matched=26`; `go vet ./internal/uiserver/... ./internal/uiproto/...` clean.
- No blockers for plan 01-10.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: internal/uiproto/uiv1/ui.proto
- FOUND: internal/uiproto/uiv1/ui.pb.go
- FOUND: internal/uiproto/uiv1/uiv1connect/ui.connect.go
- FOUND: internal/uiserver/handlers.go
- FOUND: internal/uiserver/handlers_test.go
- FOUND: internal/uiserver/readonly_test.go
- FOUND: commit f36bf82 (Task 1)
- FOUND: commit e9c9ae6 (Task 2)
- FOUND: commit 1d53f68 (Task 3)
