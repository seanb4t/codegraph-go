---
phase: 05-file-package-graph-view
plan: 06
subsystem: api
tags: [go, protobuf, connect-rpc, wire-shape, tdd, file-symbols]

# Dependency graph
requires:
  - phase: 05-file-package-graph-view
    provides: "05-08's collapsed-default re-measure (corpora/graph-render-observations-collapsed.json, verdict PASS) and maintainer answer release-collapsed, which unblocked this plan's precondition; the shared uiv1.Node message and node.go's ValidateRepoRelativePath confinement from Phase 1/3"
provides:
  - "internal/query.Engine.FileSymbols(path) — per-file symbol enumeration, ordered by start line then name, capped at MaxFileSymbols (2000), with a true total and truncation flag"
  - "UIService.FileSymbols — the thirteenth read-only rpc, projecting FileSymbolsResult onto the wire via the shared Node message"
  - "FileSymbolsRequest/FileSymbolsResponse — the frozen, additive-only wire messages"
  - "fileSymbolsToProto — the named mapper 05-07's expansion UI consumes"
  - "The research correction: internal/query.FileDetail (GetNodeDetail's file mode) carries only {Path, Source} — it never carried symbol data, so GRF-03's in-place expansion needed this new rpc, not a reuse of an existing one"
affects: [05-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 29942
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Engine method validates the caller-supplied path through the Engine's own confinement (ValidateRepoRelativePath) as a PRECONDITION, before any store scan runs — mirrors GetPermalink's SRV-05 discipline, now established for a second caller-supplied-path rpc"
    - "Server-side cap owned by the engine, referenced (never redeclared) by the wire layer — internal/uiserver imports internal/query, never the reverse, so the cap constant has exactly one home (review H-2)"
    - "Wire layer's own stake in an engine-owned cap: a dedicated marshal-and-measure test (TestFileSymbolsCappedResponseFitsTransportCeiling) asserts the cap is safe for THIS transport, without owning the cap's value"

key-files:
  created:
    - internal/query/filesymbols.go
    - internal/query/filesymbols_test.go
    - internal/uiserver/filesymbols_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - web/src/lib/gen/ui_pb.ts
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go

key-decisions:
  - "Task 1 checkpoint (human, gate=blocking-human, reversibility=one-way): maintainer verbatim reply approve-as-proposed against the exact FileSymbolsRequest(1 field)/FileSymbolsResponse(3) shape presented at the checkpoint. All three sub-decisions answered explicitly — see 'Maintainer Decision' section below for the full verbatim record."
  - "MaxFileSymbols (2000) declared exactly once, in internal/query/filesymbols.go beside the Engine method that applies it — internal/uiserver references query.MaxFileSymbols and never redeclares it (review H-2)."
  - "The cap test at the wire layer (TestFileSymbolsCappedResponseFitsTransportCeiling) marshals a hand-built response directly rather than round-tripping through a real server+store, because building a real 2000+-symbol source file is intractable; the truncation-flag behavior at the wire layer is proven separately (TestFileSymbolsCappedResponseCarriesTruncationFlag) against a real graphstore-backed fixture written directly via graphstore.Writer (not the full indexer pipeline)."
  - "Task 3's Rule 3 blocking-issue fix (identical precedent to 05-02 Task 2's FileGraph placeholder and 04-03 Task 2's GetHealth placeholder): a minimal FileSymbols placeholder returning connect.CodeUnimplemented was committed alongside the proto regeneration, so filesymbols_test.go's RED phase observed honest 'unimplemented' failures rather than a package build failure."

patterns-established:
  - "Wire-shape checkpoint before codegen (D-02a one-way door): this is the fourth plan (after 03-05's GetPermalink, 04-03's GetHealth, 05-02's FileGraph) to freeze a new rpc's field numbering at a blocking-human checkpoint before task proto:gen ever runs, and to extend readonly_test.go's chained uiProtoFieldFixtureLenAtPlan* constant rather than restate a bare literal."

requirements-completed: [GRF-03]

coverage:
  - id: D1
    description: "FileSymbolsResponse's wire shape and field numbering — FileSymbolsRequest(1 field), FileSymbolsResponse(3) — frozen at a maintainer-approved blocking-human checkpoint before any codegen ran, with all three sub-decisions (shared Node reuse, MaxFileSymbols value/owner, index-miss-is-empty-not-error) answered explicitly, and the research correction (FileDetail never carried symbol data) recorded rather than absorbed."
    requirement: GRF-03
    verification:
      - kind: other
        ref: "Task 1 checkpoint (gate=blocking-human): maintainer reply 'approve-as-proposed' with all three sub-decisions answered individually, recorded verbatim in this SUMMARY's 'Maintainer Decision' section"
        status: pass
    human_judgment: true
    rationale: "The wire shape's field numbering is a one-way door (D-02a) — the maintainer's approve-as-proposed decision IS the coverage, not a machine check alone."
  - id: D2
    description: "UIService exposes exactly thirteen read-only methods (the thirteenth is FileSymbols), no method name contains a mutating-verb substring, and every frozen field number is stable and unique in the generated descriptor."
    requirement: GRF-03
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
    description: "Engine.FileSymbols returns every symbol a file declares, ordered by start line then name, excluding the file's own file-kind record and any synthetic package pseudo-node, validates the path through the existing confinement BEFORE scanning, returns an empty result (not an error) for an unindexed path, and caps an over-sized result at MaxFileSymbols while reporting the true total and a truncation flag."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsReturnsAllSymbolsOrderedByLineThenName"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsExcludesFileNode"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsExcludesPackagePseudoNode"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsRejectsPathEscapingRepoRootBeforeScanning"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsRejectsAbsolutePathBeforeScanning"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsUnknownPathReturnsEmptyNotError"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsCapsAtMaxFileSymbols"
        status: pass
      - kind: unit
        ref: "internal/query/filesymbols_test.go#TestFileSymbolsDeterministicAcrossRepeatedCalls"
        status: pass
    human_judgment: false
  - id: D4
    description: "The FileSymbols handler projects the engine result onto the wire faithfully (element-for-element, not length-only), rejects an escaping or absolute path as CodeInvalidArgument, opens the engine exactly once, never degrades on a locked store, and a fully-capped response marshals below transportSendMaxBytes."
    requirement: GRF-03
    verification:
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsReturnsSymbolsInEngineOrder"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsRejectsPathEscapingRepoRoot"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsRejectsAbsolutePath"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsUnknownPathReturnsEmptyNotError"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsCappedResponseCarriesTruncationFlag"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsCappedResponseFitsTransportCeiling"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsOpensEngineExactlyOnce"
        status: pass
      - kind: unit
        ref: "internal/uiserver/filesymbols_test.go#TestFileSymbolsDoesNotDegrade"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-08-31
status: complete
---

# Phase 5 Plan 6: FileSymbols — the Thirteenth Read-Only RPC Summary

**`FileSymbols` is now `UIService`'s thirteenth method — a frozen, additive-only wire projection of a new `internal/query.Engine.FileSymbols` per-file symbol enumeration (ordered, path-confined, capped at 2000 and counted), closing the real gap 05-RESEARCH.md's architectural map got wrong: `GetNodeDetail`'s file mode never carried symbol data at all.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-31T15:00:00Z (approx, continuation resume from Task 1's checkpoint)
- **Completed:** 2026-08-31T15:35:00Z
- **Tasks:** 3
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- Task 1's checkpoint decision recorded verbatim below: maintainer approved the proposed `FileSymbolsRequest`/`FileSymbolsResponse` wire shape exactly as proposed, all three sub-decisions answered, and the research correction (the assumed data source never existed) acknowledged rather than absorbed.
- `internal/query/filesymbols.go` (new): `Engine.FileSymbols(path)` — a fresh-per-call `IterateNodes()` scan, confined by the existing `ValidateRepoRelativePath` gate called FIRST (before any scan), excluding the file's own file-kind record and any synthetic package pseudo-node, sorted by start line then name, capped at the new `MaxFileSymbols` constant (2000) with a true total and truncation flag. 8/8 tests pass (`go test ./internal/query/... -run 'TestFileSymbols' -race`).
- `FileSymbols` added as `UIService`'s thirteenth rpc (`internal/uiproto/uiv1/ui.proto`); `task proto:gen` regenerated the Go and TypeScript clients through the pinned toolchain; `task proto:drift` reports "compared 4 generated files", unchanged floor.
- `fileSymbolsToProto` (`internal/uiserver/handlers.go`) — a named mapper reusing the EXISTING `nodesToProto` helper over the shared `Node` message, per Task 1's sub-decision 1.
- `(*uiService).FileSymbols` — the ordinary `withEngine`-wrapped handler; path validation happens through the engine's own confinement INSIDE the closure exactly as `GetNodeDetail`/`GetPermalink` do it, so `mapEngineError`'s shared translation assigns `CodeInvalidArgument` with no second mapping.
- All three `readonly_test.go` guards green at 13 methods / 162 descriptor fields: `TestUIServiceMethodSetIsExactlyTheReadSet`, `TestUIServiceDeclaresNoMutatingMethod` (19 verbs, unchanged), `TestUIProtoFieldNumbersAreStableAndUnique`.
- `internal/uiserver/filesymbols_test.go` (new): 8/8 tests pass, including a dedicated wire-layer stake in the engine's cap (`TestFileSymbolsCappedResponseFitsTransportCeiling`) that marshals a fully-capped, realistically-populated response and asserts it sits below `transportSendMaxBytes`.
- `cd web && pnpm check` — 1151 files, 0 errors, 0 warnings, against the regenerated TypeScript client (bundle NOT rebuilt in this plan, per plan instruction — 05-07 consumes and rebuilds).

## Task Commits

Each task was committed atomically, following TDD RED→GREEN discipline for Tasks 2 and 3:

1. **Task 1: Freeze FileSymbolsResponse's wire shape** — no commit (decision-only checkpoint; the maintainer's answer is recorded in this SUMMARY, not a code artifact).
2. **Task 2 RED: failing test for Engine.FileSymbols** — `c5e4dac0` (test)
   **Task 2 GREEN: Engine.FileSymbols + MaxFileSymbols cap** — `0160162d` (feat)
3. **Task 3a RED→GREEN: readonly_test.go fixtures, then proto surface + regenerated clients + placeholder handler**:
   - `614d7a0f` (test) — extend `wantUIServiceMethods`/`uiProtoFieldNumbers` fixtures, observe RED
   - `5c97a04d` (feat) — `FileSymbols` rpc, regenerated clients, placeholder handler; turns the two RED guards GREEN
4. **Task 3d RED→GREEN: the handler and mapper**:
   - `b2dac149` (test) — `internal/uiserver/filesymbols_test.go`, observe RED (7/8 fail with honest "unimplemented", 1 pure-marshal test already passes)
   - `17b85157` (feat) — real `FileSymbols` handler + `fileSymbolsToProto`, all 8 GREEN

_No REFACTOR commits — both GREEN implementations were clean on first pass._

## Files Created/Modified

- `internal/query/filesymbols.go` (new) — `FileSymbolsResult`, `MaxFileSymbols`, `Engine.FileSymbols`.
- `internal/query/filesymbols_test.go` (new) — 8 test cases.
- `internal/uiproto/uiv1/ui.proto` — the `FileSymbols` rpc and its two messages, appended additively after `FileGraphResponse`.
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go`, `web/src/lib/gen/ui_pb.ts` — regenerated through `task proto:gen` (pinned toolchain), never hand-edited.
- `internal/uiserver/handlers.go` — `fileSymbolsToProto`, `(*uiService).FileSymbols`.
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` gains `"FileSymbols"` (count literal 12→13); `uiProtoFieldNumbers` gains 4 entries with new chained constant `uiProtoFieldFixtureLenAtPlan0506 = uiProtoFieldFixtureLenAtPlan0502 + 4`; `mutatingVerbs` left byte-unchanged.
- `internal/uiserver/filesymbols_test.go` (new) — `TestFileSymbolsReturnsSymbolsInEngineOrder`, `TestFileSymbolsRejectsPathEscapingRepoRoot`, `TestFileSymbolsRejectsAbsolutePath`, `TestFileSymbolsUnknownPathReturnsEmptyNotError`, `TestFileSymbolsCappedResponseCarriesTruncationFlag`, `TestFileSymbolsCappedResponseFitsTransportCeiling`, `TestFileSymbolsOpensEngineExactlyOnce`, `TestFileSymbolsDoesNotDegrade`.

## Maintainer Decision (Task 1, verbatim)

**MAINTAINER DECISION, 2026-08-31: `approve-as-proposed`.**

All three sub-decisions answered explicitly and affirmatively:

1. **Reuse the shared 15-field `Node` message — CONFIRMED.** A repository has one symbol shape; a narrower fourth shape would start a second vocabulary for the same domain object that future work must keep in sync. The unused `docstring`/`signature`/`return_type` bytes are the cheaper cost. A narrower message was offered and declined.
2. **`MaxFileSymbols = 2000`, declared in `internal/query` beside the Engine method that applies it — CONFIRMED.** This was settled as review finding H-2: `internal/uiserver` imports `internal/query` and never the reverse, so any other owner requires a duplicated constant, an undeclared parameter, or an import cycle. The wire layer owns `TestFileSymbolsCappedResponseFitsTransportCeiling` instead — asserting a fully-capped response marshals below `transportSendMaxBytes`, not owning the number.
3. **A path miss returns an empty list, zero total, no error — CONFIRMED.** The graph asks about paths `FileGraph` just handed it, so a miss means the index moved under the view: an ordinary race, not a failure state.

**Orchestrator's independent verification of the freeze claims** (re-derived from source, not accepted from the plan's report):
- `FileSymbols` checked against the live `mutatingVerbs` fixture: **19 verbs extracted (positive control passed), zero collisions.** A **negative control** confirmed the checker discriminates — it correctly flags `GetIndexHealth` → `Index`, the exact name Phase 4 had to abandon.
- The method-set count literal moved 12 → 13 (confirmed: `readonly_test.go:91` after the edit).
- **The research correction is confirmed:** `internal/query/detail.go`'s `FileDetail` really is exactly `{Path string; Source []byte}` — confirmed by reading `internal/query/detail.go:47-50`. `05-RESEARCH.md`'s claim that GRF-03's symbol data comes from existing rpcs is factually wrong, and this rpc closes a real gap.
- The shared `Node` message has 15 fields, confirmed by reading `internal/uiserver/readonly_test.go`'s existing `uiProtoFieldNumbers` fixture entries for `Node`.

**Frozen wire shape, approved exactly as proposed:**

```protobuf
rpc FileSymbols(FileSymbolsRequest) returns (FileSymbolsResponse);   // 13th rpc

message FileSymbolsRequest  { string path = 1; }

message FileSymbolsResponse {
  repeated Node symbols = 1;
  int32 total_count     = 2;
  bool  truncated       = 3;
}
```

Field numbering implemented exactly as approved — no amendment.

## RED Output (Task 2, engine method, verbatim)

```
# github.com/seanb4t/codegraph-go/internal/query [github.com/seanb4t/codegraph-go/internal/query.test]
internal/query/filesymbols_test.go:64:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:92:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:116:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:144:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:164:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:180:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:194:20: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:215:18: undefined: MaxFileSymbols
internal/query/filesymbols_test.go:230:16: e.FileSymbols undefined (type *Engine has no field or method FileSymbols)
internal/query/filesymbols_test.go:234:25: undefined: MaxFileSymbols
internal/query/filesymbols_test.go:234:25: too many errors
FAIL	github.com/seanb4t/codegraph-go/internal/query [build failed]
FAIL
```

Honest build failure (never an import-only stub) — the engine method and cap constant did not exist yet. GREEN: 8/8 PASS after `internal/query/filesymbols.go` was written.

## RED Output (Task 3a, readonly_test.go fixtures, verbatim)

```
=== RUN   TestUIServiceMethodSetIsExactlyTheReadSet
    readonly_test.go:91: uiv1connect.UIServiceHandler has 12 methods, want exactly 13: map[Affected:{} Callees:{} Callers:{} Explore:{} FileGraph:{} Files:{} GetHealth:{} GetNodeDetail:{} GetPermalink:{} GetStatus:{} Impact:{} Search:{}]
--- FAIL: TestUIServiceMethodSetIsExactlyTheReadSet (0.00s)
=== RUN   TestUIServiceDeclaresNoMutatingMethod
    readonly_test.go:144: inspected 12 UIServiceHandler method names against 19 mutating verbs
--- PASS: TestUIServiceDeclaresNoMutatingMethod (0.00s)
=== RUN   TestUIProtoFieldNumbersAreStableAndUnique
    readonly_test.go:515: inspected 38 messages and 158 fields in the generated uiv1 descriptor
    readonly_test.go:527: known field FileSymbolsRequest.path (number 1) is missing from the generated descriptor entirely
--- FAIL: TestUIProtoFieldNumbersAreStableAndUnique (0.00s)
```

Confirms exactly the plan's `<behavior>` expectation: the set-equality guard and the field-number stability guard both FAIL (12 vs want 13; missing descriptor field), while the negative verb guard stays GREEN with its inspected count unchanged at 12. After the proto edit + `task proto:gen`, all three: `TestUIServiceMethodSetIsExactlyTheReadSet` PASS (13 methods), `TestUIServiceDeclaresNoMutatingMethod` PASS (13 methods inspected), `TestUIProtoFieldNumbersAreStableAndUnique` PASS (40 messages, 162 fields, up from 158).

## RED Output (Task 3d, handler and mapper, verbatim)

```
filesymbols_test.go:39: FileSymbols: unimplemented: FileSymbols: not yet implemented (plan 05-06 Task 3)
--- FAIL: TestFileSymbolsReturnsSymbolsInEngineOrder (0.09s)
    filesymbols_test.go:98: FileSymbols(../../../../etc/passwd): code = unimplemented, want CodeInvalidArgument
--- FAIL: TestFileSymbolsRejectsPathEscapingRepoRoot (0.06s)
    filesymbols_test.go:116: FileSymbols(/etc/passwd): code = unimplemented, want CodeInvalidArgument
--- FAIL: TestFileSymbolsRejectsAbsolutePath (0.05s)
    filesymbols_test.go:134: FileSymbols(go.mod): unimplemented: FileSymbols: not yet implemented (plan 05-06 Task 3)
--- FAIL: TestFileSymbolsUnknownPathReturnsEmptyNotError (0.05s)
    filesymbols_test.go:218: FileSymbols(big.go): unimplemented: FileSymbols: not yet implemented (plan 05-06 Task 3)
--- FAIL: TestFileSymbolsCappedResponseCarriesTruncationFlag (0.05s)
--- PASS: TestFileSymbolsCappedResponseFitsTransportCeiling (0.00s)
    filesymbols_test.go:301: FileSymbols: unimplemented: FileSymbols: not yet implemented (plan 05-06 Task 3)
--- FAIL: TestFileSymbolsOpensEngineExactlyOnce (0.05s)
    filesymbols_test.go:333: FileSymbols while locked: code = unimplemented, want CodeUnavailable
--- FAIL: TestFileSymbolsDoesNotDegrade (0.08s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/uiserver	0.816s
FAIL
```

7 of 8 tests fail with real, honest "unimplemented" errors against Task 3a's `CodeUnimplemented` placeholder (never a compile error); `TestFileSymbolsCappedResponseFitsTransportCeiling` already passes since it marshals a hand-built response and never calls the handler. GREEN: 8/8 PASS after the real handler and `fileSymbolsToProto` replaced the placeholder.

## Decisions Made

- **Task 1:** Maintainer approved the proposed `FileSymbolsResponse` wire shape verbatim (`approve-as-proposed`) — see "Maintainer Decision" above.
- **Task 3 test-design choice:** `TestFileSymbolsCappedResponseFitsTransportCeiling` marshals a hand-built `*uiv1.FileSymbolsResponse` directly rather than round-tripping through a real server+store, because a real source file with 2000+ actual symbol declarations is not tractable to construct or parse for a unit test. The truncation-flag *behavior* at the wire layer is instead proven by a separate test (`TestFileSymbolsCappedResponseCarriesTruncationFlag`) against a real graphstore-backed fixture written directly through `graphstore.Writer` (`PutNode`/`PutMeta`/`Commit`) rather than the full indexer pipeline — this keeps the test fast while still exercising the real `Engine.FileSymbols` → `fileSymbolsToProto` → wire path.
- **`newFileSymbolsBigStore` helper (new pattern for this file):** writes directly to a `graphstore.GraphStore` via `NewWriter()`/`PutNode`/`PutMeta`/`Commit` rather than through `indexer.Run`, because generating 2000+ real Go symbol declarations for the real tree-sitter-based indexer to parse would be both slow and pointless — `FileSymbols` reads index records only (T-05-32), so a hand-built store record set exercises the exact same code path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Minimal `FileSymbols` placeholder required to keep `internal/uiserver` building after the proto regeneration**
- **Found during:** Task 3a, immediately after `task proto:gen` — re-running `go build ./internal/uiserver/...`.
- **Issue:** The regenerated `uiv1connect.UIServiceHandler` interface gained a `FileSymbols` method the moment the proto was regenerated. `*uiService` has no forward-compatibility embed, so Go requires it to implement every interface method for the package to build at all — this broke compilation of the WHOLE `internal/uiserver` package, not just the newly-added guard tests, before Task 3d's real handler existed.
- **Fix:** Added a minimal `(*uiService).FileSymbols` placeholder to `handlers.go` in the same commit as the proto regeneration (`5c97a04d`), returning `connect.CodeUnimplemented` rather than a fabricated response — deliberately, so `filesymbols_test.go`'s RED phase would observe real, honest "unimplemented" failures (confirmed: 7 of 8 always-run-against-the-handler tests failed with `unimplemented` errors before Task 3d's real implementation existed). Task 3d replaced (not extended) the placeholder with the real handler and mapper.
- **Files modified:** `internal/uiserver/handlers.go`
- **Verification:** all three readonly guards GREEN after the placeholder; `TestFileSymbols*` RED (7/8 FAIL, 1 PASS for the pure-marshal test) before the real implementation; GREEN (8/8 PASS) after.
- **Committed in:** `5c97a04d` (placeholder), `17b85157` (real implementation replaces it)

---

**Total deviations:** 1 auto-fixed (1 blocking).
**Impact on plan:** The fix is the identical, previously-recorded precedent from `05-02`'s Task 2 (FileGraph) and `04-03`'s Task 2 (GetHealth) — a mechanical consequence of Go's whole-package build model whenever a proto regeneration adds an rpc, not a new class of problem. No weakening of any threshold, bound, or acceptance criterion.

### Verification-criterion false positive investigated and confirmed benign (not fixed) — same finding as 05-02, re-confirmed for this plan

- **The `uiProtoFieldNumber\{` diff-pattern acceptance criterion never matches, for the same reason 05-02-SUMMARY.md already recorded.** `git diff internal/uiserver/readonly_test.go | rg -c '^-.*uiProtoFieldNumber\{'` and the `^\+` counterpart both return 0 once the change is committed, because Go composite-literal slice elements (`{"FileSymbolsRequest", "path", 1},`) do NOT repeat the slice's element type name per entry — only the `var uiProtoFieldNumbers = []uiProtoFieldNumber{` declaration line contains that substring, and this plan's diff never touches that line.
- **Confirmed the actual invariant holds, with the corrected pattern 05-02 established:** `git show 614d7a0 -- internal/uiserver/readonly_test.go | rg -c '^-\t\{"'` (removed tuple lines) → **0**; `git show 614d7a0 -- internal/uiserver/readonly_test.go | rg -c '^\+\t\{"'` (added tuple lines) → **4**. Zero existing entries were removed or rewritten; exactly four were appended — the append-only property this task's own action text and `readonly_test.go`'s EXTENDS-by-exactly-N convention require, independently confirmed by `TestUIProtoFieldNumbersAreStableAndUnique` passing (every prior plan's entries still resolve).
- **Not fixed:** per verification_discipline, a plan gate that cannot pass as literally written is a finding to report, not to silently relax by rewording the acceptance criterion in-place. This is the SAME plan-authoring defect 05-02-SUMMARY.md already flagged for a future plan-check pass — recorded again here because it recurs verbatim in this plan's criteria, not because it is a new finding.

## Issues Encountered

None beyond the deviation and the re-confirmed criterion note above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `FileSymbols` is on the wire, frozen, tested, and closes the real data-source gap `05-RESEARCH.md` got wrong — 05-07's in-place expansion has everything it needs to fetch and render a file's symbols.
- `internal/schema/` was never touched by this plan (only `internal/uiproto/uiv1/` and its generated clients) — the drift guard's 4-file floor is unchanged.
- `web/build` was NOT rebuilt in this plan (only `web/src/lib/gen/ui_pb.ts`, the generated client source) — per the plan's own instruction, 05-07 is the plan that consumes the new client and rebuilds the bundle.
- GRF-03 is now satisfied — ready to mark complete.
- No blockers.

## Self-Check: PASSED

- `test -f internal/query/filesymbols.go` → FOUND
- `test -f internal/query/filesymbols_test.go` → FOUND
- `test -f internal/uiserver/filesymbols_test.go` → FOUND
- `test -f internal/uiproto/uiv1/ui.proto` → FOUND (modified)
- `git log --oneline --all | grep -q c5e4dac0` → FOUND
- `git log --oneline --all | grep -q 0160162d` → FOUND
- `git log --oneline --all | grep -q 614d7a0f` → FOUND
- `git log --oneline --all | grep -q 5c97a04d` → FOUND
- `git log --oneline --all | grep -q b2dac149` → FOUND
- `git log --oneline --all | grep -q 17b85157` → FOUND
- `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run 'TestFileSymbols' -race -count=1` → PASS (8/8)
- `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/... -run 'TestUIService|TestFileSymbols|TestUIProtoFieldNumbersAreStableAndUnique' -v -count=1` → PASS (33/33 lines, including all readonly guards and 8/8 FileSymbols tests)
- `GOTOOLCHAIN=go1.26.5 task test:unit` → PASS (all packages)
- `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -race -count=1` → PASS
- `task proto:drift` → PASS ("compared 4 generated files", byte-identical)
- `cd web && pnpm check` → PASS (0 errors, 0 warnings, 1151 files)
- `git diff Taskfile.yml` → empty

---
*Phase: 05-file-package-graph-view*
*Completed: 2026-08-31*
