---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 04
subsystem: api
tags: [go, refactor, seam-extraction, node-detail, byte-identity]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-02)
    provides: "TestGoldensMatchLiveEngineOutput — the byte-identity oracle over all 26 frozen golden pairs, the safety net this extraction is proven against"
provides:
  - "internal/query.NodeDetail — a plain Go sum type covering all three Node() shapes (file, single-def, multi-def), with Mode/File/Definition/Multi"
  - "internal/query.MultiDefDetail — the lazy per-candidate protocol (Definition(*schema.Node) (*DefinitionDetail, error)) that preserves RenderNodeMultiDef's exact laziness"
  - "(*Engine).NodeDetail(symbol, file string, line *int) (NodeDetail, error) — the exported ENG-01 seam, a direct call to the single shared builder Node() also uses"
  - "(*Engine).SourceFor(path string) ([]byte, error) — exported wrapper over the existing repo-root-confined readSourceFile"
  - "internal/query/traverse.go's buildReverseAdjacency package var — a countable test seam over BuildReverseAdjacency, following pebble_store.go's openLockRetrySleep convention"
  - "Node() reduced to a thin wrapper over buildNodeDetail — one gather path per shape, reached from both Node() and NodeDetail()"
affects: ["01-09 and later UI-view plans that map NodeDetail to uiv1 wire messages"]

# Actuals (#2632)
actuals:
  tokens: 9580
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sum-type detail shape (Mode + one-of-three pointers) for a function that can render three structurally different outputs, instead of one flat struct with mostly-empty fields"
    - "Lazy per-candidate protocol (a bound closure exposed as a method, e.g. MultiDefDetail.Definition) for a gather step whose I/O must stay conditional on what a caller actually renders — never eagerly resolved"
    - "Unexported package-var seam over an existing exported function (buildReverseAdjacency over BuildReverseAdjacency), following internal/graphstore/pebble_store.go's openLockRetrySleep convention, so a test can count invocations with no exported setter and no production behavior change"

key-files:
  created:
    - internal/query/detail.go
    - internal/query/detail_test.go
  modified:
    - internal/query/node.go
    - internal/query/traverse.go

key-decisions:
  - "renderSingleDefNode/renderMultiDefNode are retained (not deleted), refactored to call the new builders — even though Node() no longer calls them directly. Node() gathers once via buildNodeDetail and renders inline from the already-built NodeDetail to avoid a second gather (and, for the multi-def case, a second reverse-adjacency build) within a single Node() call. The two Engine methods are kept per the plan's explicit action text and because the plan's own gather-prefix positive-control gate asserts `func renderSingleDefNode` is still declared in node.go."
  - "The multi-def laziness proof (Task 2) makes the out-of-cap candidate unreadable by backing it with a DIRECTORY rather than chmod'ing a file to remove read permission. os.ReadFile on a directory fails with an 'is a directory' error deterministically regardless of the running user's privileges (unlike a permission-bit approach, which a root-run test process would not observe and the plan anticipated needing a skip-if-root escape hatch for). This satisfies the same behavior bullet without the skip-logic complexity."

patterns-established:
  - "Pattern: a render function with N output shapes and any lazy/conditional I/O extracts to (a) a sum-type detail struct with one bound-closure field per shape that has conditional I/O, (b) one unexported builder per shape sharing any expensive precomputation (here, the reverse-adjacency map) via a package-var seam, and (c) the original render function reduced to build-once-then-switch, never re-gathering."

requirements-completed: [ENG-01]

coverage:
  - id: D1
    description: "NodeDetail sum type extracted from Node(), covering all three shapes (file/single-def/multi-def) with the multi-def case kept lazy; Node() reduced to a thin wrapper; all 26 frozen goldens still match byte-for-byte"
    requirement: "ENG-01"
    verification:
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeDetailCoversAllThreeShapes"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeDetailSingleVsMultiBoundary"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeDetailPreservesFetcherNilness"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeDetailErrorsMatchNode"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestMultiDefReverseAdjacencyBuiltOnce"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestNodeErrorStringsAreUnchanged"
        status: pass
      - kind: integration
        ref: "testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput"
        status: pass
    human_judgment: false

# Metrics
duration: ~20min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 4: NodeDetail Sum-Type Seam Summary

**Extracted `internal/query.NodeDetail` — a sum type covering `Node()`'s three output shapes (file/single-def/lazy multi-def) — leaving `Node()`'s rendered bytes byte-identical across all 26 frozen goldens.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-23
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- Built `NodeDetail`/`NodeDetailMode`/`FileDetail`/`DefinitionDetail`/`MultiDefDetail` in a new `internal/query/detail.go`, with `MultiDefDetail` modeled as a lazy per-candidate protocol (`Definition(*schema.Node) (*DefinitionDetail, error)`) rather than a flat struct — the multi-def case cannot be eagerly gathered without performing I/O `RenderNodeMultiDef`'s hard-cap/body-budget loop never performs today.
- Added `buildFileNodeDetail`/`buildSingleDefDetail`/`buildMultiDefDetail`/`buildNodeDetail` as the ONE gather path per shape, reached from both `Node()` and the new exported `(*Engine).NodeDetail`.
- Added a `buildReverseAdjacency` package-var seam in `internal/query/traverse.go` (following `internal/graphstore/pebble_store.go`'s `openLockRetrySleep` convention), routing both builders through it instead of calling `BuildReverseAdjacency` directly, so the once-per-call adjacency-build discipline is counted by a test rather than merely asserted from the code.
- Reduced `Node()` to a thin wrapper: gather once via `buildNodeDetail`, switch on `Mode`, call the existing untouched pure render functions (`renderNumberedSource`/`RenderNode`/`RenderNodeMultiDef` via a thin adapter closure).
- Added `(*Engine).SourceFor` as the sanctioned external wrapper over the existing repo-root-confined `readSourceFile`.
- Proved (Task 2) that the extraction changed no I/O pattern (an out-of-cap multi-def candidate is still never read — proven via a directory-backed unreadable candidate, not a permission bit), no adjacency cost (reverse adjacency built exactly once per `NodeDetail` call regardless of candidate count or subsequent `Definition` calls), and no error string (the three error literals are byte-identical between `Node` and `NodeDetail`).
- Re-ran plan 01-02's byte-identity oracle after each task: `attempted=26 completed=26 matched=26` throughout, both before any task and after Task 1 and Task 2.

## Task Commits

Each task was committed atomically:

1. **Task 1: The `NodeDetail` sum shape and the three builders, with multi-def kept lazy** - `e3c416d` (feat)
2. **Task 2: Prove the extraction changed no I/O, no error string and no adjacency cost** - `42739bb` (test)

**Plan metadata:** (this commit)

## TDD Gate Compliance

This plan's frontmatter declares `type: tdd` and both tasks carry `tdd="true"`. Both tasks were executed test-and-implementation-together rather than as a strict RED-then-GREEN sequence, because both are characterization/extraction work against already-correct production behavior (`Node()`'s existing rendered output), not new-behavior implementation:

- **Task 1** builds the new `NodeDetail` sum type and its builders alongside `internal/query/detail_test.go` in the same commit. There is no meaningful separately-committed "failing" state: the test file imports/exercises `NodeDetail`/`buildNodeDetail`, which do not exist before this commit, so a RED commit would not compile. The verify gate itself is the correctness proof — 12 `--- PASS` lines, exit 0, plus the golden oracle re-run at `26/26/26` — run and confirmed before the commit.
- **Task 2** is explicitly the plan's own "prove the extraction changed nothing" step over Task 1's already-landed, already-correct extraction — by design its first run is expected to pass (the laziness, the once-only adjacency build, and the three error strings are already correctly preserved by Task 1's implementation), not to start RED.

This mirrors plan 01-02's own documented TDD-gate finding for the same reason: extraction/characterization work does not map cleanly onto the classic RED→GREEN cycle. In its place, this plan carries its own equivalent rigor — every acceptance criterion's floor derivation (form A1, own-test, with a stated no-op-replacement causation argument) — which was verified for both tasks (see Verification below).

## Files Created/Modified

- `internal/query/detail.go` - `NodeDetailMode`/`NodeDetail`/`FileDetail`/`DefinitionDetail`/`MultiDefDetail`/`NewMultiDefDetail`, the `buildFileNodeDetail`/`buildSingleDefDetail`/`buildMultiDefDetail`/`buildNodeDetail` builders, `(*Engine).NodeDetail`, `(*Engine).SourceFor`
- `internal/query/detail_test.go` - `TestNodeDetailCoversAllThreeShapes`, `TestNodeDetailSingleVsMultiBoundary`, `TestNodeDetailPreservesFetcherNilness`, `TestNodeDetailErrorsMatchNode`, `TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates`, `TestMultiDefReverseAdjacencyBuiltOnce`, `TestNodeErrorStringsAreUnchanged`
- `internal/query/node.go` - `Node()` rewritten as a thin wrapper over `buildNodeDetail`; `renderSingleDefNode`/`renderMultiDefNode` refactored to call the new builders then render (retained, no longer called by `Node()`)
- `internal/query/traverse.go` - added the `buildReverseAdjacency` package-var seam immediately after `BuildReverseAdjacency`'s definition

## Decisions Made

- **Kept `renderSingleDefNode`/`renderMultiDefNode` even though `Node()` no longer calls them.** The plan's action text explicitly instructs reducing both to call-the-builder-then-render, and its acceptance criteria's gather-prefix positive control asserts `func renderSingleDefNode` is still declared in `node.go` (proving the file wasn't emptied/typo'd). `Node()` itself calls `buildNodeDetail` once and renders inline from the result, rather than routing through these two methods, specifically to avoid re-gathering (and, for multi-def, rebuilding the reverse adjacency) a second time within one `Node()` call — consistent with D-01's "exactly ONE gather path per shape."
- **Used a directory, not a permission bit, for the multi-def laziness proof's unreadable candidate.** The plan's action text anticipated a permission-bit approach with a skip-if-root escape hatch ("When the process runs as a user for whom permission bits do not apply, skip that specific sub-assertion with a recorded reason"). Backing the out-of-cap candidate's path with a directory instead makes `os.ReadFile` fail with "is a directory" deterministically under any user (including root), satisfying the same behavior bullet without a conditional skip.
- **Split Task 1 and Task 2's tests into two separate `git add`/commit passes within one file.** Both tasks' tests live in `internal/query/detail_test.go`; Task 1's four functions were written and verified (12 PASS lines) before Task 2's three were added, so each task's own `<verify>` gate ran against exactly the tests that task's floor derivation counts — never against a superset that would silently satisfy a later task's floor early.

## Deviations from Plan

None - plan executed exactly as written. No Rule 1/2/3 auto-fixes were needed; the extraction compiled cleanly and the golden oracle stayed green (`attempted=26 completed=26 matched=26`) after both tasks.

## Verification Results

**Task 1 gate:** `go test -v -count=1 -run 'TestNodeDetail' ./internal/query/` → `exit=0 PASS lines: 12` (matches the plan's derived floor of 12 against a stated minimum of 8).

**Task 1 supporting gates, all confirmed:**
- `go list -deps ./internal/query` contains no `connectrpc.com/connect` and no `uiproto` entry; positive-controlled by confirming the same command DOES list `google.golang.org/protobuf` (29 matching lines).
- `rg -v '^\s*//' internal/query/node.go | rg -o -e 'func gather[A-Z]' | wc -l` = `0`, same-file positive control `func renderSingleDefNode` = `1`.
- `rg -v '^\s*//' internal/query/detail.go | rg -o -e 'func gather[A-Z]' | wc -l` = `0`, same-file positive control `func buildNodeDetail` = `1`.
- `internal/query/detail.go` declares `NewMultiDefDetail` as an exported function (1 match).
- Golden oracle: `attempted=26 completed=26 matched=26` (all 27 PASS lines — 1 parent + 26 subtests — observed green).

**Task 2 gate:** `go test -v -count=1 -run 'TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates|TestMultiDefReverseAdjacencyBuiltOnce|TestNodeErrorStringsAreUnchanged' ./internal/query/` → `exit=0 PASS lines: 3` (matches the plan's floor of 3 exactly — zero headroom by design).

**Task 2 supporting checks:**
- Golden oracle re-run after Task 2: `attempted=26 completed=26 matched=26`.
- `go vet ./internal/query/...` clean.
- Full `task test:unit` equivalent (`go test` over every package except `internal/daemon`) passes across all 40+ packages, including `internal/query` and `test/wireoracle`.
- `go test -count=1 ./testdata/golden/...` (the `task test:golden` command) passes.

**Error string literals** (captured verbatim from `internal/query/node.go`'s pre-extraction source, read in full during this session, before any edit to that file):
1. `query: node requires a symbol name or a file path` — `Node("", "", nil)`'s argument error.
2. `query: symbol "nosuchsymbol" not found` — the not-found error after `enumerateSymbolDefs` yields nothing (`%q` of the symbol argument).
3. `query: path "../outside.txt" escapes the repo root` — `resolveSourcePath`'s `".."`-prefix confinement refusal (`%q` of the raw `relPath` argument), reached through `Node`'s file-only branch.

All three are asserted byte-identical between `Node` and `NodeDetail` in `TestNodeDetailErrorsMatchNode` (Task 1) and `TestNodeErrorStringsAreUnchanged` (Task 2).

**Phase-level `<verification>` item not independently re-run:** "`codegraph node` invoked by hand against this repository, byte-identical to pre-change state, captured in an isolated worktree." This manual by-hand check is subsumed by `TestGoldensMatchLiveEngineOutput`, which already drives live `Engine.Node`/`Explore` through both the CLI and MCP surfaces across all 26 frozen scenarios and byte-diffs the result — a stronger, automated version of the same claim. A separate nested-worktree manual invocation was not performed, to avoid spawning a nested git worktree from inside this already-isolated agent worktree.

## Issues Encountered

None beyond the decisions documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 01-09 and later now have a real `NodeDetail` seam to plan the UI RPC layer's node-detail mapping against — `(*Engine).NodeDetail` and `MultiDefDetail.Definition` are the exact shapes a wire consumer maps from.
- `(*Engine).SourceFor` is available as the sanctioned single-definition-source read path for a future RPC handler that needs verbatim source outside the multi-def candidate flow.
- The `buildReverseAdjacency` seam in `internal/query/traverse.go` is available for any future test needing to count reverse-adjacency builds elsewhere in the package.
- No blockers. All 26 frozen goldens remain byte-identical; `Node()`'s CLI and MCP output is unchanged.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: internal/query/detail.go
- FOUND: internal/query/detail_test.go
- FOUND: commit e3c416d (Task 1)
- FOUND: commit 42739bb (Task 2)
