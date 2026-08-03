---
phase: 01-behavioral-parity-explore-node
plan: 04
subsystem: api
tags: [node, multi-def, markdown-render, go]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: (no direct plan dependency — depends_on is empty; reuses traverse.go's resolveSymbolNode/D-03 full-scan base and node.go's existing single-def RenderNode path)
provides:
  - internal/query/node.go — isGeneratedFile (D-07 verbatim regex list), enumerateSymbolDefs (NODE-01 multi-def scan), narrowNodeMatches (NODE-03 never-empty file/line filter), fetchCalls/fetchCalledBy/renderSingleDefNode/renderMultiDefNode (Node() dispatch)
  - internal/query/render_markdown.go — renderNumberedSourceRange, renderNodeSection, RenderNodeMultiDef (NODE-02 two-line header + HARD_CAP/BODY_BUDGET/LIST_CAP budget + overflow)
affects: [01-17 (golden harness wiring node-multi.json against this render path), any future plan adding a `line`/refined `file` CLI or MCP hint that would call narrowNodeMatches]

tech-stack:
  added: []
  patterns:
    - "Multi-def enumeration reuses D-03's full IterateNodes scan base (like resolveSymbolNode), sorted generated-files-last (isGeneratedFile primary key) then lowest-Id (secondary tie-break — a documented Go-side determinism improvement over TS's non-deterministic row order, RESEARCH Pattern 2)"
    - "narrowNodeMatches is a pure in-memory filter injected as a standalone, independently-tested function — no CLI/MCP `line` flag exists yet to drive it, so it's a forward-looking capability (NODE-03's must-have is satisfied at the function-contract level, ready for a later plan to wire a `line` hint through)"
    - "RenderNodeMultiDef takes a nodeSectionFetch callback (lazy, stops after HARD_CAP) so the pure budget/overflow logic lives in render_markdown.go while the I/O (disk reads, edge iteration) stays in node.go — the same transform/service split the rest of the package already uses"
    - "renderNodeSection's Trail/Calls/CalledBy lines are omitted ENTIRELY when both are empty (confirmed against live TS golden captures) — a real behavioral divergence from the single-def RenderNode, which always renders both lines even when empty"

key-files:
  created: []
  modified:
    - internal/query/node.go
    - internal/query/node_test.go
    - internal/query/render_markdown.go
    - internal/query/render_markdown_test.go

key-decisions:
  - "Kept resolveNodeForDetail's `file != \"\"` behavior completely untouched (exact-match single-winner) rather than routing it through narrowNodeMatches's endsWith/includes semantics — NODE-04 explicitly requires 'keep the existing single-def resolveNodeForDetail path intact', and changing its matching semantics risked silently turning a previously-single-result `-f` disambiguation into a multi-def render for any corpus with 2+ same-name-same-file defs"
  - "narrowNodeMatches (NODE-03) is fully implemented and independently tested (TestNarrowNeverEmpty) but not wired into Engine.Node's public two-arg (symbol, file) signature — there is no `line` hint parameter anywhere in the current CLI/MCP surface (files_modified for this plan excludes internal/cli/node.go and internal/mcp/tools.go), and the golden capture harness (testdata/golden/capture.sh) has no line-hint fixture either. The function is a ready building block for a future plan that adds a `line`/refined `file` flag."
  - "Ad-hoc byte-diffed the real synthetic-parity/node-multi.json golden fixture (not part of the committed test suite — that's plan 01-17's job) during GREEN to sanity-check the render path beyond hand-built unit tests; this surfaced and fixed a real bug (see Deviations) before it could reach a later plan"

patterns-established:
  - "nodeSectionFetch callback pattern: a pure renderer (render_markdown.go) accepts a lazy per-candidate data-fetch closure from its I/O-capable caller (node.go), letting the HARD_CAP/BODY_BUDGET decision loop stay both pure (unit-testable with fabricated data, no Engine needed) and bounded (never fetches source/edges for more than HARD_CAP candidates, even when matches is very large — DoS mitigation T-01-07)"

requirements-completed: [NODE-01, NODE-02, NODE-03, NODE-04]

coverage:
  - id: D1
    description: "isGeneratedFile ports TS's ~24-pattern GENERATED_PATTERNS regex list verbatim (D-07); enumerateSymbolDefs does a full IterateNodes scan collecting every node with a matching Name, sorted generated-files-last then lowest-Id"
    requirement: "NODE-01"
    verification:
      - kind: unit
        ref: "internal/query/node_test.go#TestIsGeneratedFile"
        status: pass
      - kind: unit
        ref: "internal/query/node_test.go#TestNodeMultiDef"
        status: pass
    human_judgment: false
  - id: D2
    description: "RenderNodeMultiDef renders the exact two-line header + HARD_CAP=16/BODY_BUDGET=12000 full-body budget (always rendering the first) + LIST_CAP=20 overflow list with the verbatim closing hint line"
    requirement: "NODE-02"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestRenderNodeMultiDef"
        status: pass
    human_judgment: false
  - id: D3
    description: "narrowNodeMatches filters an already-enumerated node slice by fileHint (endsWith/includes) and/or lineHint (containing-def else nearest), never assigning an empty result"
    requirement: "NODE-03"
    verification:
      - kind: unit
        ref: "internal/query/node_test.go#TestNarrowNeverEmpty"
        status: pass
    human_judgment: false
  - id: D4
    description: "Engine.Node dispatches: a single-match symbol lookup renders via the original, byte-unchanged RenderNode path; multiple matches render via RenderNodeMultiDef — proven both via a direct-RenderNode-comparison regression test and an end-to-end Engine.Node dispatch test"
    requirement: "NODE-04"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestRenderNodeSingleDefUnchanged"
        status: pass
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestNodeMultiDefWiring"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 04: Node Multi-Def Enumeration + Budget/Overflow + Never-Empty Narrowing Summary

**`node` now enumerates ALL exact-name definitions of an overloaded symbol (generated-files-last), renders TS's exact two-line multi-def header with a HARD_CAP=16/BODY_BUDGET=12000 full-body budget and a LIST_CAP=20 overflow list, supports a never-empty file/line narrowing filter, and keeps single-definition output byte-identical to before this plan.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-07-15
- **Tasks:** 3 (each RED → GREEN)
- **Files modified:** 4 (node.go, node_test.go, render_markdown.go, render_markdown_test.go)

## Accomplishments
- `isGeneratedFile` (D-07): verbatim port of TS's ~24-pattern `GENERATED_PATTERNS` regex list from `extraction/generated-detection.js:27-82`
- `enumerateSymbolDefs` (NODE-01): full `IterateNodes` scan collecting every node with a matching `Name` (reusing D-03's `resolveSymbolNode` scan base), sorted generated-files-last (primary) then lowest-`Id` (secondary, documented divergence per RESEARCH Pattern 2)
- `narrowNodeMatches` (NODE-03): a pure in-memory filter over the enumerated slice — `fileHint` endsWith/includes-narrows only if the result is non-empty, `lineHint` prefers a containing def else falls back to the single nearest by `|StartLine-lineHint|`, and the final guard never assigns an empty set
- `renderNodeSection` + `RenderNodeMultiDef` (NODE-02): the exact two-line header (`**N definitions named "X"**` immediately followed by `Returning M in full[; K more listed below] — pick the one you need (no Read required).`), full bodies rendered up to `HARD_CAP`/within `BODY_BUDGET` (always rendering at least the first), and a `**Other definitions**` overflow list capped at `LIST_CAP` with the verbatim closing hint
- `Engine.Node` now dispatches on `enumerateSymbolDefs`'s match count: 1 match → the original, byte-unchanged `RenderNode` path (NODE-04); >1 match → `RenderNodeMultiDef`
- Ad-hoc verified the render path against the real `testdata/golden/corpus/synthetic-parity/node-multi.json` golden capture (indexed the corpus, called `Engine.Node("Validate", "")`, byte-diffed) — this caught and fixed a real bug before it reached plan 01-17's golden harness (see Deviations)

## Task Commits

Each task's RED and GREEN steps were committed atomically:

1. **Task 1: isGeneratedFile + multi-def enumeration (NODE-01)** — RED `9d26883` (test), GREEN `0f0d1a6` (feat)
2. **Task 2: never-empty file/line narrowing (NODE-03)** — RED `ad7f205` (test), GREEN `ebbe25a` (feat)
3. **Task 3: two-line header + budget + overflow render (NODE-02/04)** — RED `10b1cdb` (test), GREEN `3d76d5f` (feat)

_TDD gate sequence verified: each `test(...)` commit precedes its `feat(...)` commit in git log._

## Files Created/Modified
- `internal/query/node.go` — `generatedFilePatterns`, `isGeneratedFile`, `enumerateSymbolDefs`, `normalizeNarrowPath`, `absInt32`, `narrowNodeMatches`, `fetchCalls`, `fetchCalledBy`, `renderSingleDefNode`, `renderMultiDefNode`; `Node()` refactored to dispatch between them
- `internal/query/node_test.go` — `TestIsGeneratedFile`, `TestNodeMultiDef`, `TestNarrowNeverEmpty`
- `internal/query/render_markdown.go` — `renderNumberedSourceRange`, `nodeMultiDefHardCap`/`nodeMultiDefBodyBudget`/`nodeMultiDefListCap`, `nodeSectionFetch`, `renderNodeSection`, `RenderNodeMultiDef`
- `internal/query/render_markdown_test.go` — `TestRenderNodeMultiDef`, `TestRenderNodeSingleDefUnchanged`, `TestNodeMultiDefWiring`

## Decisions Made
- Left `resolveNodeForDetail`'s `file != ""` exact-match single-winner behavior completely untouched — NODE-04 explicitly requires the existing path stay intact, and TS's endsWith/includes narrowing semantics could silently change a previously-single-result `-f` disambiguation into a multi-def render if a corpus ever had 2+ same-name-same-file defs. `narrowNodeMatches` exists as a fully-tested, ready-to-wire building block instead.
- `narrowNodeMatches` (NODE-03) is implemented and independently tested but not reachable through the current public `Engine.Node(symbol, file string)` signature — no `line` hint exists anywhere in the CLI/MCP surface today, and this plan's `files_modified` scope excludes `internal/cli/node.go`/`internal/mcp/tools.go`. A future plan adding a `line`/refined `file` flag can call it directly.
- Ad-hoc byte-diffed the live `synthetic-parity/node-multi.json` golden capture during GREEN (not committed as a test — that's plan 01-17's job) to sanity-check beyond hand-built fixtures; this is what caught the source-slicing bug below.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Multi-def source blocks rendered the whole file instead of just the definition's own line range**
- **Found during:** Task 3, ad-hoc verification against the real `synthetic-parity/node-multi.json` golden capture
- **Issue:** `renderNodeSection` initially called the existing `renderNumberedSource(source)` on the full on-disk file content, numbering every line from 1 — producing a code block containing the whole file (package decl, imports, doc comments, unrelated functions) instead of just the matched definition's own body. The golden capture shows only the def's `[StartLine,EndLine]` span, numbered with its TRUE on-disk line numbers (e.g. a function starting at line 10 is shown as `10\tfunc ...`, not renumbered from 1).
- **Fix:** Added `renderNumberedSourceRange(content, startLine, endLine)` — slices to the def's own line span (clamping safely on an out-of-range/zero `EndLine`) and numbers with true on-disk line numbers. `renderNodeSection` now calls this instead of `renderNumberedSource`.
- **Files modified:** internal/query/render_markdown.go
- **Verification:** ad-hoc byte-diff against `testdata/golden/corpus/synthetic-parity/node-multi.json` — code blocks now match exactly (line numbers and content); re-ran `go test ./internal/query/ -count=1` — still green
- **Committed in:** 3d76d5f (folded into Task 3's GREEN commit, discovered and fixed before that commit)

### Documented, out-of-scope divergences (not bugs)

The same ad-hoc golden diff surfaced two further differences from the live TS capture that are **explicitly out of this plan's scope**, per `01-CONTEXT.md`'s phase boundary ("Independent of the edge-kind work... Wave 1") and D-02's allowed-divergence policy:

1. **Section order** — my output orders `accounts/validate.go` and `orders/validate.go` differently than the captured TS run. TS's sort has no secondary tie-break (RESEARCH Pattern 2: its own row order is non-deterministic across re-indexes); this plan intentionally adds a lowest-`Id` secondary tie-break for Go-side determinism, a documented divergence, not a bug.
2. **Missing type-reference targets in `Calls →`** (e.g. `UserAccountManager`, `errEmptyOrder`) — these require the `references`/`type_of` edge kinds that plans 01-05 through 01-12 (the D-09 edge-kind expansion, later in this same phase) add to the extraction pipeline. `fetchCalls` is unchanged from the pre-existing single-def logic (still filters `RefKindCalls` only), so this is a pre-existing gap this plan does not touch, not a regression.

Both will resolve automatically once the edge-kind expansion plans land — no follow-up action needed in this plan.

---

**Total deviations:** 1 auto-fixed (Rule 1 — source-range slicing bug, caught by ad-hoc golden verification before it could reach plan 01-17's harness); 2 documented out-of-scope divergences (ordering, missing edge kinds — both explicitly deferred elsewhere in the phase)
**Impact on plan:** The bug fix increases fidelity to the actual TS algorithm and was caught before any downstream plan depended on the broken behavior. No scope creep, no architectural change.

## Issues Encountered
None beyond the one auto-fixed bug above (source-range slicing).

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Plan 01-17 (golden harness wiring) can now assert `node-multi.json` byte-parity against `Engine.Node` directly — the render path is real, tested, and already ad-hoc-verified against `synthetic-parity/node-multi.json`'s code-block content and structure (modulo the two documented divergences above, both scoped to later plans in this phase).
- Plans 01-05 through 01-12 (D-09 edge-kind expansion: `references`/`overrides`/`instantiates`/`returns`/`type_of`) will, once landed, close the "missing type-reference in Calls →" gap noted above — no changes needed in `node.go`/`render_markdown.go` for that; `fetchCalls`'s existing `RefKindCalls` filter and `fetchCalledBy`'s reverse-adjacency map (built from the same edge kind) will pick up new edges automatically as soon as they're extracted, since both iterate the graph's edges rather than hardcoding a kind list.
- `narrowNodeMatches` (NODE-03) is ready for a future plan that adds a `line` hint (and/or relaxes `file` to endsWith/includes semantics) to the CLI (`internal/cli/node.go`) and MCP (`internal/mcp/tools.go`) surfaces — no changes to `node.go` itself would be needed, just a new call site.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

All created/modified files and all 6 task commit hashes (9d26883, 0f0d1a6, ad7f205, ebbe25a, 10b1cdb, 3d76d5f) verified present in git log.
