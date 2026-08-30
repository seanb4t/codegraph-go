---
phase: 05-file-package-graph-view
plan: 02
subsystem: api
tags: [go, protobuf, connect-rpc, wire-shape, tdd, file-graph]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-01's Engine.FileGraph() rollup and stronglyConnectedCycles() cycle detection — the exact Go source this plan projects onto the wire"
provides:
  - "UIService.FileGraph — the twelfth read-only rpc, projecting internal/query.FileGraphResult onto the wire"
  - "FileGraphRequest/FileGraphNode/FileGraphEdge/FileGraphResponse — the frozen, additive-only wire messages"
  - "fileGraphToProto and its per-element mappers — the named mapping layer 05-03's renderer build consumes"
  - "A measured (not estimated) guava-scale response-size figure: 3,713,528 bytes, replacing 05-RESEARCH.md's 5-6 MB estimate"
affects: [05-03, 05-04, 05-05, 05-06, 05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 20798
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Named field-for-field mapper with per-element helpers (fileGraphToProto/fileGraphNodeToProto/fileGraphEdgeToProto), following healthToProto's convention exactly"
    - "Minimal CodeUnimplemented placeholder handler committed alongside a proto regeneration, so a later TDD task's RED phase observes honest 'unimplemented' failures instead of a package build failure — the same Rule 3 blocking-issue fix 04-03's Task 2 established for GetHealth"

key-files:
  created:
    - internal/uiserver/filegraph_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human, reversibility=one-way): maintainer verbatim reply approve-as-proposed against the exact FileGraphRequest(1 field)/FileGraphNode(4)/FileGraphEdge(5)/FileGraphResponse(6) shape presented at the checkpoint. All four sub-decisions answered explicitly — see 'Maintainer Decision' section below for the full verbatim record."
  - "TestFileGraphResponseSizeGuava drives the measurement through a REAL server+client round trip (startedServer + uiv1connect client), not a direct in-process eng.FileGraph()+fileGraphToProto call — this avoids a compile-time dependency from the test file onto fileGraphToProto before Task 3's GREEN phase (which would have broken the RED-before-handler-exists sequencing the same way Task 2's placeholder was needed), and it more faithfully measures what a browser actually receives (the connect-go wire path, not a bypassed short-circuit)."
  - "Task 2's blocking-issue fix (Rule 3): the regenerated uiv1connect.UIServiceHandler interface requires every method for the package to build at all, so a minimal FileGraph placeholder returning connect.CodeUnimplemented was committed in Task 2's GREEN commit — the identical precedent 04-03's Task 2 recorded for GetHealth. Task 3 replaced (not extended) the placeholder with the real handler."

patterns-established:
  - "Wire-shape checkpoint before codegen (D-02a one-way door): this is the third plan (after 03-05's GetPermalink and 04-03's GetHealth) to freeze a new rpc's field numbering at a blocking-human checkpoint before task proto:gen ever runs, and to extend readonly_test.go's chained uiProtoFieldFixtureLenAtPlan* constant rather than restate a bare literal."

requirements-completed: [ENG-03]

coverage:
  - id: D1
    description: "FileGraphResponse's wire shape and field numbering — FileGraphRequest(1 field), FileGraphNode(4), FileGraphEdge(5), FileGraphResponse(6) — frozen at a maintainer-approved blocking-human checkpoint before any codegen ran, with all four sub-decisions (excluded_* counters, in_cycle redundancy, rpc name FileGraph re-verified against all 19 mutatingVerbs substrings, unprefixed message names) answered explicitly."
    requirement: ENG-03
    verification:
      - kind: other
        ref: "Task 1 checkpoint (gate=blocking-human): maintainer reply 'approve-as-proposed' with all four sub-decisions answered individually, recorded verbatim in this SUMMARY's 'Maintainer Decision' section"
        status: pass
    human_judgment: true
    rationale: "The wire shape's field numbering is a one-way door (D-02a) — the maintainer's approve-as-proposed decision IS the coverage, not a machine check alone."
  - id: D2
    description: "UIService exposes exactly twelve read-only methods (the twelfth is FileGraph), no method name contains a mutating-verb substring, and every frozen field number is stable and unique in the generated descriptor."
    requirement: ENG-03
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
  - id: D3
    description: "The FileGraph handler answers with every value internal/query.FileGraphResult carries, opens the engine exactly once through the shared withEngine seam, never degrades and answers on a locked store, and ignores the request's unread path field."
    requirement: ENG-03
    verification:
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphProjectsEngineResult"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphKindCountsAreSparse"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphOpensEngineExactlyOnce"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphDoesNotDegrade"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphRequestPathIsIgnored"
        status: pass
    human_judgment: false
  - id: D4
    description: "The serialized FileGraphResponse for google/guava is a measured byte count — 3,713,528 bytes — below the 16 MiB transportSendMaxBytes ceiling and above the 1,000,000-byte floor, replacing 05-RESEARCH.md's unverified 5-6 MB estimate."
    requirement: ENG-03
    verification:
      - kind: unit
        ref: "internal/uiserver/filegraph_test.go#TestFileGraphResponseSizeGuava (CODEGRAPH_GUAVA_STORE-gated)"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-08-30
status: complete
---

# Phase 5 Plan 2: FileGraph — the Twelfth Read-Only RPC Summary

**`FileGraph` is now `UIService`'s twelfth method — a frozen, additive-only wire projection of `internal/query.FileGraphResult` (nodes, aggregated per-kind-counted edges, three exclusion counters, cycle count), backed by a named `fileGraphToProto` mapper, with the guava-scale wire-size question RESEARCH left as a 5-6 MB estimate now measured at 3,713,528 bytes.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-08-30T16:00:00Z (approx, continuation resume)
- **Completed:** 2026-08-30T16:30:00Z
- **Tasks:** 3
- **Files modified:** 7 (1 created, 6 modified)

## Accomplishments

- `FileGraphResponse`'s wire shape and field numbering frozen at a maintainer-approved `blocking-human` checkpoint (Task 1) before any codegen ran — see "Maintainer Decision" below for the verbatim record.
- `FileGraph` added as `UIService`'s twelfth rpc (`internal/uiproto/uiv1/ui.proto`); `task proto:gen` regenerated the Go and TypeScript clients through the pinned toolchain; `task proto:drift` reports "compared 4 generated files", unchanged floor.
- `fileGraphToProto` (`internal/uiserver/handlers.go`) — a named, field-for-field mapper with per-element `fileGraphNodeToProto`/`fileGraphEdgeToProto` helpers, following `healthToProto`'s convention.
- `(*uiService).FileGraph` — the ordinary `withEngine`-wrapped handler, explicitly NOT `GetStatus`'s degrade-and-answer exception.
- The guava-scale response size is now a **measured** number: 3,713,528 bytes (3,233 nodes, 21,554 edges, 0 excluded package nodes, 10,838 excluded self edges, 56,641 excluded contains edges, 162 cycles) — 22.1% of the 16,777,216-byte ceiling, replacing `05-RESEARCH.md`'s unverified 5-6 MB estimate.

## Task Commits

Each task was committed atomically, following TDD RED→GREEN discipline for Tasks 2 and 3:

1. **Task 1: Freeze FileGraphResponse's wire shape** — no commit (decision-only checkpoint; the maintainer's answer is recorded in this SUMMARY, not a code artifact).
2. **Task 2 RED: failing test literals for FileGraph (the 12th rpc)** — `a91f58cd` (test)
   **Task 2 GREEN: FileGraph rpc + regenerated clients + placeholder handler** — `e9d37b9c` (feat)
3. **Task 3 RED: failing tests for the FileGraph handler and mapper** — `a25760cb` (test)
   **Task 3 GREEN: the real handler and fileGraphToProto mapper** — `39bb4d20` (feat)

_No REFACTOR commits — both GREEN implementations were clean on first pass._

## Files Created/Modified

- `internal/uiproto/uiv1/ui.proto` — the `FileGraph` rpc and its four messages, appended additively after `GetHealthResponse`.
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go`, `web/src/lib/gen/ui_pb.ts` — regenerated through `task proto:gen` (pinned toolchain), never hand-edited.
- `internal/uiserver/handlers.go` — `fileGraphToProto`, `fileGraphNodeToProto`, `fileGraphEdgeToProto`, `(*uiService).FileGraph`.
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` gains `"FileGraph"` (count literal 11→12); `uiProtoFieldNumbers` gains 16 entries with new chained constant `uiProtoFieldFixtureLenAtPlan0502 = uiProtoFieldFixtureLenAtPlan0403 + 16`; `mutatingVerbs` left byte-unchanged.
- `internal/uiserver/filegraph_test.go` (new) — `TestFileGraphProjectsEngineResult`, `TestFileGraphKindCountsAreSparse`, `TestFileGraphOpensEngineExactlyOnce`, `TestFileGraphDoesNotDegrade`, `TestFileGraphRequestPathIsIgnored`, `TestFileGraphResponseSizeGuava`.

## Maintainer Decision (Task 1, verbatim)

**MAINTAINER DECISION, 2026-08-30: `approve-as-proposed`.**

All four sub-decisions answered explicitly and affirmatively:

1. **The three `excluded_*` counters STAY on the wire — CONFIRMED.** D-03 drops `contains` and self-edges, D-08 drops the synthetic `package` pseudo-nodes; carrying the counts lets the UI state plainly what it is not showing rather than misleading by silent omission. Dropping them was offered and declined.
2. **`in_cycle` STAYS on the edge despite being client-derivable — CONFIRMED.** The redundancy is deliberate: it prevents any future renderer re-implementing a correctness property that D-06 placed server-side on purpose, which is exactly what GRF-05's swappable seam exists to protect. Dropping it was offered and declined.
3. **The rpc name `FileGraph` — CONFIRMED.** The orchestrator independently re-verified this against the live `mutatingVerbs` fixture in `internal/uiserver/readonly_test.go`: 19 substrings extracted (positive control passed), **zero collisions** with `FileGraph`, and a **negative control** confirmed the checker discriminates — it correctly flags `GetIndexHealth` → `Index`, the exact name Phase 4 had to abandon.
4. **Unprefixed message names `FileGraphNode` / `FileGraphEdge` — CONFIRMED**, matching 04-03's `PendingChanges` / `IndexHealth` precedent. The package qualifier disambiguates (`query.FileGraphNode` vs `uiv1.FileGraphNode`).

**Frozen wire shape, approved exactly as proposed:**

```protobuf
rpc FileGraph(FileGraphRequest) returns (FileGraphResponse);   // 12th rpc

message FileGraphRequest  { string path = 1; }                 // declared, deliberately unread

message FileGraphNode {
  string path = 1;  string language = 2;
  int64 symbol_count = 3;  int32 cycle_id = 4;                 // 0 = no cycle, else 1-based component id
}

message FileGraphEdge {
  string source_file = 1;  string target_file = 2;
  map<string, int64> kind_counts = 3;                          // sparse: absent, never 0
  int64 total_count = 4;   bool in_cycle = 5;
}

message FileGraphResponse {
  repeated FileGraphNode nodes = 1;
  repeated FileGraphEdge edges = 2;
  int64 excluded_package_node_count  = 3;
  int64 excluded_self_edge_count     = 4;
  int64 excluded_contains_edge_count = 5;
  int32 cycle_count = 6;
}
```

Field numbering implemented exactly as approved — no amendment.

## RED Output (Task 2, verbatim)

```
readonly_test.go:80: uiv1connect.UIServiceHandler has 11 methods, want exactly 12: map[Affected:{} Callees:{} Callers:{} Explore:{} Files:{} GetHealth:{} GetNodeDetail:{} GetPermalink:{} GetStatus:{} Impact:{} Search:{}]
--- FAIL: TestUIServiceMethodSetIsExactlyTheReadSet (0.00s)
    readonly_test.go:133: inspected 11 UIServiceHandler method names against 19 mutating verbs
--- PASS: TestUIServiceDeclaresNoMutatingMethod (0.00s)
    readonly_test.go:481: inspected 34 messages and 142 fields in the generated uiv1 descriptor
    readonly_test.go:493: known field FileGraphRequest.path (number 1) is missing from the generated descriptor entirely
--- FAIL: TestUIProtoFieldNumbersAreStableAndUnique (0.00s)
FAIL
```

Confirms exactly the plan's `<behavior>` expectation: the set-equality guard and the field-number stability guard both FAIL (11 vs want 12; missing descriptor field), while the negative verb guard stays GREEN with its inspected count unchanged at 11.

## RED Output (Task 3, verbatim)

```
filegraph_test.go:38: FileGraph: unimplemented: FileGraph: not yet implemented (plan 05-02 Task 3)
--- FAIL: TestFileGraphProjectsEngineResult (0.07s)
    filegraph_test.go:134: FileGraph: unimplemented: FileGraph: not yet implemented (plan 05-02 Task 3)
--- FAIL: TestFileGraphKindCountsAreSparse (0.06s)
    filegraph_test.go:177: FileGraph: unimplemented: FileGraph: not yet implemented (plan 05-02 Task 3)
--- FAIL: TestFileGraphOpensEngineExactlyOnce (0.05s)
    filegraph_test.go:209: FileGraph while locked: code = unimplemented, want CodeUnavailable
--- FAIL: TestFileGraphDoesNotDegrade (0.08s)
    filegraph_test.go:228: FileGraph (empty path): unimplemented: FileGraph: not yet implemented (plan 05-02 Task 3)
--- FAIL: TestFileGraphRequestPathIsIgnored (0.05s)
    filegraph_test.go:253: CODEGRAPH_GUAVA_STORE is unset; set it to an indexed google/guava checkout root to run this measurement
--- SKIP: TestFileGraphResponseSizeGuava (0.00s)
FAIL
```

All five always-run tests fail with real, honest "unimplemented" errors against Task 2's `CodeUnimplemented` placeholder (never a compile error), and the guava-gated test skips by default. GREEN: all 6 PASS (5 always-run + guava, with `CODEGRAPH_GUAVA_STORE` set).

## Measured Guava Response Size (Task 3, replacing the RESEARCH estimate)

```
guava FileGraphResponse: 3713528 bytes serialized, 3233 nodes, 21554 edges,
0 excluded package nodes, 10838 excluded self edges, 56641 excluded contains
edges, 162 cycles
```

- **Corpus:** `google/guava` @ `94f39958baf7ad51ddf9c70e406ed6b188194daa` (the exact SHA pinned in `corpora/graph-render-threshold.json`, indexed at `~/.cache/codegraph/corpora/google-guava-2b0cb53f@94f39958...`).
- **Measured, not estimated:** the byte count comes from a real Connect RPC round trip (`startedServer` + `uiv1connect` client) against the live server, `proto.Marshal`-ed on the returned `*uiv1.FileGraphResponse`.
- **Node/edge counts match `corpora/graph-render-threshold.json`'s own corpus note exactly** (3,233 file-level nodes, 21,554 distinct file-pair edges after D-03's contains/self-edge exclusion and D-08's package-pseudo-node exclusion) — an independent cross-check that the rollup and this measurement agree.
- **`0` excluded package nodes** — expected: `05-CONTEXT.md` records that `google/guava` has zero synthetic `"package"`-kind pseudo-nodes, which is exactly why D-08's regression test had to run against this repository's own index rather than the measurement corpus.
- **Delta against `05-RESEARCH.md`'s estimate:** RESEARCH estimated 5-6 MB (5,242,880-6,291,456 bytes); the measured figure, 3,713,528 bytes, is **1,529,352 to 2,577,928 bytes (29-41%) below** the estimated range — well within the 16,777,216-byte ceiling at 22.1% of it, with the 1,000,000-byte floor cleared by more than 3.7×.

## Decisions Made

- **Task 1:** Maintainer approved the proposed `FileGraphResponse` wire shape verbatim (`approve-as-proposed`) — see "Maintainer Decision" above.
- **Task 2 blocking-issue fix (Rule 3):** a minimal `FileGraph` placeholder returning `connect.CodeUnimplemented` was added in Task 2's GREEN commit — required because the regenerated `uiv1connect.UIServiceHandler` interface unconditionally demands every method for the package to build, and Task 3's RED phase needed to observe honest "unimplemented" failures rather than a compile error. Identical to the precedent `04-03-SUMMARY.md` records for `GetHealth`'s Task 2.
- **Task 3 test-design choice:** `TestFileGraphResponseSizeGuava` measures through a real server+client round trip rather than calling `eng.FileGraph()` + `fileGraphToProto` directly in-process. This was deliberate, not incidental: a direct call would have created a compile-time dependency from `filegraph_test.go` onto `fileGraphToProto` at RED time (before Task 3's mapper existed), breaking the same RED-before-handler-exists sequencing Task 2's placeholder was needed to preserve — the whole package would fail to build rather than fail honestly. The round-trip approach also more faithfully measures what a browser actually receives.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Minimal `FileGraph` placeholder required to keep `internal/uiserver` building after Task 2's codegen**
- **Found during:** Task 2, step (e) — re-running the readonly guards immediately after `task proto:gen`.
- **Issue:** The regenerated `uiv1connect.UIServiceHandler` interface gained a `FileGraph` method the moment the proto was regenerated. `*uiService` has no forward-compatibility embed, so Go requires it to implement every interface method for the package to build at all — this broke compilation of the WHOLE `internal/uiserver` package, not just the newly-added guard tests, before Task 3's handler existed.
- **Fix:** Added a minimal `(*uiService).FileGraph` placeholder to `handlers.go` in Task 2's own GREEN commit, returning `connect.CodeUnimplemented` rather than a fabricated response — deliberately, so Task 3's `filegraph_test.go` RED phase would observe real, honest "unimplemented" failures against it (confirmed: all 5 always-run tests failed with `unimplemented` errors before Task 3's real implementation existed). Task 3 replaced (not extended) the placeholder with the real handler and mapper.
- **Files modified:** `internal/uiserver/handlers.go`
- **Verification:** both readonly guards GREEN after the placeholder (Task 2's `<verify>`); `TestFileGraph*` RED (5 FAIL against the placeholder, 1 SKIP for the env-gated test) before Task 3's real implementation; GREEN (6 PASS) after.
- **Committed in:** `e9d37b9c` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking).
**Impact on plan:** The fix is the identical, previously-recorded precedent from `04-03`'s Task 2 (GetHealth) — a mechanical consequence of Go's whole-package build model whenever a proto regeneration adds an rpc, not a new class of problem. No weakening of any threshold, bound, or acceptance criterion.

### Verification-criterion false positive investigated and confirmed benign (not fixed)

- **Task 2's `uiProtoFieldNumber\{` diff-pattern acceptance criterion never matches, for any correctly-written extension.** The criterion reads: `git diff ... | rg -c '^-.*uiProtoFieldNumber\{'` reports 0, "positive-controlled by" `git diff ... | rg -c '^\+.*uiProtoFieldNumber\{'` reporting at least 16. Go composite-literal slice elements (`{"FileGraphNode", "path", 1},`) do NOT repeat the slice's element type name (`uiProtoFieldNumber`) per entry — only the `var uiProtoFieldNumbers = []uiProtoFieldNumber{` declaration line itself contains that substring, and this plan's diff does not touch that line. Ran the literal command: both directions returned rg exit code 1 (zero matches), not "0 / ≥16" as the criterion's prose implies. This is not specific to this plan — the identical pattern would have returned 0/0 against every prior extension (01-10, 01-11, 03-05, 04-03) too, since none of them repeat the type name per entry either; the criterion's wording has never actually distinguished "removed" from "added" for this fixture's literal syntax.
- **Confirmed the actual invariant holds, with a corrected pattern:** `git diff internal/uiserver/readonly_test.go | rg -c '^-\t\{"'` (removed tuple lines) → **0**; `git diff internal/uiserver/readonly_test.go | rg -c '^\+\t\{"'` (added tuple lines) → **16**. Zero existing entries were removed or rewritten; exactly sixteen were appended — the append-only property Task 2's action text and `readonly_test.go`'s own EXTENDS-by-exactly-N convention require, independently confirmed by `TestUIProtoFieldNumbersAreStableAndUnique` passing (every prior plan's entries still resolve).
- **Not fixed:** per verification_discipline, a plan gate that cannot pass as literally written is a finding to report, not to silently relax by rewording the acceptance criterion in-place. Recorded here as a plan-authoring finding for a future plan-check pass.

## Issues Encountered

None beyond the deviation and the investigated criterion above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `FileGraph` is on the wire, frozen, tested, and its response size at guava scale is measured (3,713,528 bytes, well under the 16 MiB ceiling) — 05-03's renderer build has everything it needs to fetch and render the file graph.
- `internal/schema/` was never touched by this plan (only `internal/uiproto/uiv1/` and its generated clients) — the drift guard's 4-file floor is unchanged.
- ENG-03 is now satisfied by both declaring plans (05-01, 05-02) — ready to mark complete.
- No blockers.

## Self-Check: PASSED

- `test -f internal/uiserver/filegraph_test.go` → FOUND
- `test -f internal/uiproto/uiv1/ui.proto` → FOUND (modified)
- `git log --oneline --all | grep -q a91f58cd` → FOUND
- `git log --oneline --all | grep -q e9d37b9c` → FOUND
- `git log --oneline --all | grep -q a25760cb` → FOUND
- `git log --oneline --all | grep -q 39bb4d20` → FOUND
- `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestUIServiceMethodSetIsExactlyTheReadSet|TestUIServiceDeclaresNoMutatingMethod|TestUIProtoFieldNumbersAreStableAndUnique' -v -count=1` → PASS (3/3)
- `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestFileGraph' -v -count=1` (CODEGRAPH_GUAVA_STORE set) → PASS (6/6)
- `GOTOOLCHAIN=go1.26.5 task test:unit` → PASS (all packages, `internal/uiserver` 29.207s)
- `task proto:drift` → PASS ("compared 4 generated files", byte-identical)
- `cd web && pnpm check` → PASS (0 errors, 0 warnings)
- `git diff Taskfile.yml` → empty

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-30*
