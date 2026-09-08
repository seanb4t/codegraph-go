---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 02
subsystem: testing
tags: [golden-tests, mcp, tdd, byte-identity, mutation-testing, go]

# Dependency graph
requires: []
provides:
  - "internal/goldenspec — the single authoritative capture-spec package (LockedCorpusArgs, LanguageToLockedSlug, SlugToRepo, GoldenCapture, CallExploreViaMCP/CallNodeViaMCP/CallNodeViaMCPWithArgs/MCPResultText) imported by both gocapture and the golden test package"
  - "TestGoldensMatchLiveEngineOutput — a byte-identity oracle over all 26 frozen golden pairs, proven able to fail via a hand-run mutation of internal/query/render_markdown.go"
  - "TestGoldensMatchLiveEngineOutputIsNonVacuous — a permanent companion guard proving compareGoldenOutput itself detects an appended byte and a flipped interior byte"
affects: ["01-04 (Engine seam extraction) — this oracle is the safety net that must stay green across that extraction"]

# Actuals (#2632)
actuals:
  tokens: 11835
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared capture-spec package (internal/goldenspec) so a generator (package main) and a test package can both import one authoritative table/call-shape instead of maintaining independent copies"
    - "Parent-owned results slice for attempted/completed/matched counts across t.Run subtests, so a t.Fatal inside a subtest cannot silently under-count"
    - "Mutation-proof performed by hand against a reverse-patch-protected working tree, with the observed failing set measured and locked rather than assumed universal"

key-files:
  created:
    - internal/goldenspec/spec.go
    - internal/goldenspec/mcp.go
    - internal/goldenspec/spec_test.go
    - testdata/golden/byte_identity_test.go
  modified:
    - testdata/golden/gocapture/main.go
    - testdata/golden/behavioral_test.go
    - testdata/golden/golden_test.go

key-decisions:
  - "Extended the plan's single-function mutation target (RenderNode) to also mutate renderNodeSection's called-by emission, after discovering serilog's 'LogEvent' multi-def golden survives a RenderNode-only mutation while genuinely containing called-by content — see Deviations."
  - "Closed the CLI-Engine before driving MCP cases against the same on-disk directory (mirroring TestExploreCLIMatchesMCP/TestNodeCLIMatchesMCP's existing pattern), after an initial run showed query.OpenAt's underlying store open is exclusive and a concurrently-open Engine causes MCP calls to return a spurious error result."

patterns-established:
  - "Pattern: byte-identity oracle enumerates its case list BEFORE any subtest runs and asserts the count immediately, deriving attempted/completed/matched from a parent-owned slice rather than in-subtest counters."

requirements-completed: [ENG-01, ENG-02]

coverage:
  - id: D1
    description: "Shared capture-spec package (internal/goldenspec) eliminates the duplicate locked-args table and MCP call shape previously declared in both gocapture/main.go and behavioral_test.go"
    requirement: "ENG-01"
    verification:
      - kind: unit
        ref: "internal/goldenspec/spec_test.go#TestLockedCorpusArgsAreTheFrozenValues"
        status: pass
      - kind: unit
        ref: "testdata/golden/behavioral_test.go#TestExploreCLIMatchesMCP"
        status: pass
      - kind: unit
        ref: "testdata/golden/behavioral_test.go#TestNodeCLIMatchesMCP"
        status: pass
    human_judgment: false
  - id: D2
    description: "Byte-identity oracle over all 26 frozen golden pairs, proven able to fail via a hand-run mutation with the failing set measured and every survivor confirmed called-by-free"
    requirement: "ENG-02"
    verification:
      - kind: unit
        ref: "testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput"
        status: pass
      - kind: unit
        ref: "testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutputIsNonVacuous"
        status: pass
    human_judgment: false

# Metrics
duration: 35min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 2: Byte-Identity Oracle Summary

**Built the missing byte-diff oracle over all 26 frozen golden pairs — proven able to fail via a two-function hand-run mutation, closing the ENG-01/ENG-02 safety-net gap D-04 assumed already existed.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-23
- **Tasks:** 2
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- Extracted the previously-duplicated locked capture-args table, language/slug maps, MCP call shape, and golden envelope struct into a single new `internal/goldenspec` package, imported by both `gocapture` (the generator) and the golden test package (the oracle) — pinned against a literal fixture by `TestLockedCorpusArgsAreTheFrozenValues`.
- Reduced `behavioral_test.go`'s four MCP helper functions to thin `t.Helper()` wrappers over `goldenspec`'s error-returning functions, eliminating the second full declaration set the plan's own research flagged.
- Built `TestGoldensMatchLiveEngineOutput`: a byte-identity oracle that enumerates all 26 (corpus, golden) pairs, drives one Engine+directory build per corpus, and byte-diffs live `Node()`/`Explore()` output (CLI and MCP surfaces) against each golden's `Output` field — reporting `attempted`/`completed`/`matched` from a parent-owned results slice.
- Added `TestGoldensMatchLiveEngineOutputIsNonVacuous` as a permanent companion, proving `compareGoldenOutput` itself detects an appended byte and a flipped interior byte.
- Performed the D-04 mutation-proof by hand against `internal/query/render_markdown.go`, watched the oracle go RED with a measured failing set, confirmed every surviving node-bearing pair genuinely has no called-by content, then fully restored the file via a captured reverse patch and re-confirmed 26/26/26.

## Task Commits

Each task was committed atomically:

1. **Task 1: One authoritative capture spec, importable by both the generator and the oracle** - `8a911a5` (feat)
2. **Task 2: Byte-identity comparison over all 26 pairs, counted three ways and proven able to fail** - `9507e7e` (test)

**Plan metadata:** (this commit)

## TDD Gate Compliance

This plan's frontmatter declares `type: tdd`, but its two tasks are fundamentally extraction/characterization work rather than new-behavior implementation, so the classic RED (failing test) → GREEN (implementation) sequence does not map cleanly onto either task:

- **Task 1** moves already-working values into a new package and adds one new pinning test (`TestLockedCorpusArgsAreTheFrozenValues`) against the moved value — there is no meaningful "failing" state to write first, since the package the test imports did not exist until the move that makes it pass.
- **Task 2** builds a brand-new oracle test against **unmodified, already-correct** production code (`Node()`/`Explore()`) — by design (the plan's stated purpose: prove the oracle's own correctness while the code under test is known-good), so the very first run is expected to pass, not fail.

Git log order is `feat(01-02)` (Task 1) then `test(01-02)` (Task 2) — GREEN before a would-be RED, not RED-then-GREEN. **The plan-level RED/GREEN gate as literally defined is not satisfied by this commit sequence.** In its place, this plan carries its own more specific and more rigorous validation mechanism (D-04's mutation-proof): the oracle was demonstrated able to fail against a real, confirmed-applied production mutation, with the failing set measured and every non-failing node-bearing pair individually confirmed to genuinely lack the content the mutation removed (see "Mutation-Proof Results" below). This is the property rule `84d1gfpywd` and D-04 actually require for this task, and it was proven; the generic RED/GREEN commit-ordering heuristic simply does not apply to a package-extraction-plus-characterization-oracle plan.

## Files Created/Modified

- `internal/goldenspec/spec.go` - `PerCorpusArgs`, `LockedCorpusArgs`, `LanguageToLockedSlug`, `SlugToRepo`, `GoldenCapture` (moved, exported, values unchanged)
- `internal/goldenspec/mcp.go` - `CallExploreViaMCP`, `CallNodeViaMCP`, `CallNodeViaMCPWithArgs`, `MCPResultText` (moved, error-returning shape)
- `internal/goldenspec/spec_test.go` - `TestLockedCorpusArgsAreTheFrozenValues`, pinning the moved table against a literal fixture
- `testdata/golden/byte_identity_test.go` - `TestGoldensMatchLiveEngineOutput`, `TestGoldensMatchLiveEngineOutputIsNonVacuous`, `goldenIdentityCase`/`goldenIdentityCases`/`compareGoldenOutput`
- `testdata/golden/gocapture/main.go` - now imports `internal/goldenspec` instead of declaring its own copies; MCP helper functions removed
- `testdata/golden/behavioral_test.go` - `languageToLockedSlug`/`slugToRepo`/`goldenCapture` deleted in favor of `goldenspec.` references; four MCP helpers reduced to thin wrappers
- `testdata/golden/golden_test.go` - three bare `goldenCapture` references updated to `goldenspec.GoldenCapture` (Rule 3 auto-fix — see Deviations)

## Decisions Made

- **Extended the mutation-proof to a second function.** The plan's action text named exactly one mutation target (`RenderNode`). Running that mutation alone produced a failing set of 3 pairs (all in the `requests` corpus) and 10 non-failing node-bearing pairs. Per the plan's own acceptance criteria, every non-failing pair must be individually confirmed to have no called-by content in its golden, or the survivor is an "oracle hole." Nine of the ten survivors confirmed cleanly empty, but `serilog/go-node-multi.json` genuinely contains a `Called by ←` line — it renders through `RenderNodeMultiDef`/`renderNodeSection` (the multi-def path), a function `RenderNode`'s mutation never touches. Rather than accept this as an unresolved oracle hole, I additionally mutated `renderNodeSection`'s own called-by emission (a second, equally minimal, fully-reversed test-only mutation) and re-ran the proof. The combined failing set became 4 pairs (`serilog/go-node-multi.json`, `requests/go-node.json`, `requests/go-node-multi.json`, `requests/go-node-mcp.json`), and all 9 remaining survivors were independently confirmed called-by-free. This closes the gap without weakening any check — it strengthens the proof to cover both code paths that can render a "Called by" line.
- **Closed the Engine before driving MCP cases against the same directory.** An initial implementation kept the per-corpus `*query.Engine` open (via `t.Cleanup`) while also driving MCP cases against the same on-disk directory, and all 8 MCP-kind subtests failed with "returned an error result." `query.OpenAt`'s underlying store open is exclusive; `TestExploreCLIMatchesMCP`/`TestNodeCLIMatchesMCP` (behavioral_test.go) already establish the correct ordering — close the CLI-side Engine before calling the MCP helpers against the same index. `buildEngineAndDirAt` was changed to return an explicit close function instead of registering it via `t.Cleanup`, and the main test closes the Engine after running that corpus's Engine-kind subtests and before running its MCP-kind subtests.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed golden_test.go's compile break from removing the `goldenCapture` type**
- **Found during:** Task 1
- **Issue:** `goldenCapture` was declared in `behavioral_test.go` but referenced bare (same package) in `golden_test.go` (`TestGoSideFixturesRegenerated`, `TestReFrozenGoldensValid` — 3 use sites). Deleting the declaration from `behavioral_test.go` per Task 1's explicit instruction broke `golden_test.go`'s compile.
- **Fix:** Added the `internal/goldenspec` import to `golden_test.go` and updated its 3 bare `goldenCapture` references to `goldenspec.GoldenCapture`.
- **Files modified:** testdata/golden/golden_test.go (not originally in this plan's `files_modified`, but required for the package to compile — no other wave-1 plan touches this file)
- **Verification:** `go vet ./testdata/golden/` clean; `task test:golden` passes.
- **Committed in:** `8a911a5` (Task 1 commit)

**2. [Rule 1 - Bug] Extended the mutation-proof to `renderNodeSection` after discovering `RenderNode`-only mutation left an oracle hole**
- **Found during:** Task 2's hand-run mutation-proof
- **Issue:** See "Decisions Made" above — `serilog/go-node-multi.json` survives a `RenderNode`-only mutation while genuinely containing called-by content, because it renders through the sibling multi-def function `renderNodeSection`.
- **Fix:** Mutated `renderNodeSection`'s called-by branch alongside `RenderNode`'s, in the same hand-run session; fully restored both via one captured reverse patch.
- **Files modified:** internal/query/render_markdown.go (mutated and restored — NOT part of the committed diff; `git status --porcelain internal/query/` confirmed empty before continuing)
- **Verification:** Combined mutation produces a 4-pair failing set; re-run after restore returns to `attempted=26 completed=26 matched=26 of 26`.
- **Committed in:** N/A (mutation was never committed — restored before Task 2's commit)

---

**Total deviations:** 2 auto-fixed (1 blocking compile fix, 1 bug/gap-closing in the verification's own rigor)
**Impact on plan:** Both were necessary for correctness — the first for the package to compile at all, the second to actually satisfy the plan's own "no oracle hole" acceptance criterion rather than merely appearing to. No scope creep into production behavior; `internal/query/render_markdown.go`'s committed content is byte-for-byte unchanged.

## Mutation-Proof Results

**Mutation applied:** dropped `RenderNode`'s unconditional `**Called by ←**` line AND `renderNodeSection`'s conditional `if len(calledBy) > 0 { ... }` block (both in `internal/query/render_markdown.go`).

**Applied-confirmation:** `git diff --stat internal/query/render_markdown.go` showed `4 deletions(-)` before running the test.

**Failing set (4 of 26):**
- `serilog/go-node-multi.json` (multi-def "LogEvent", first candidate has real callers)
- `requests/go-node.json` (single-def "Session")
- `requests/go-node-multi.json` (single-def "Request" — despite its "-multi" filename, "Request" resolves to exactly one definition in this corpus, so it renders via the single-def `RenderNode` path)
- `requests/go-node-mcp.json` (single-def "Session" via MCP)

**Survivor confirmations (9 node-bearing pairs that did not fail, each individually checked for a `Called by` string in its raw golden JSON):**
- `hugo/go-node.json`, `hugo/go-node-multi.json`, `hugo/go-node-mcp.json` — all multi-def renders with zero calls/calledBy on every candidate; no "Called by" text present.
- `guava/go-node.json`, `guava/go-node-multi.json`, `guava/go-node-mcp.json` — same.
- `serilog/go-node.json`, `serilog/go-node-mcp.json` — same.
- `behavioral/go-node-multi.json` — same (both "Validate" definitions have zero callers).

No survivor with called-by content remains — the reported "oracle hole" stop condition does not apply once both rendering paths are covered.

**Restore method:** captured `git diff internal/query/render_markdown.go > mutation-reverse2.patch` immediately after applying the combined mutation, then restored with `git apply -R mutation-reverse2.patch`.

**Post-restore confirmation:** `test -d internal/query && git status --porcelain internal/query/` printed nothing (clean); re-running `TestGoldensMatchLiveEngineOutput` returned `attempted=26 completed=26 matched=26 of 26`, PASS.

**Pre-move baseline (recorded per acceptance criteria):** `TestExploreCLIMatchesMCP` (1 + 5 = 6 PASS lines) and `TestNodeCLIMatchesMCP` (1 + 6 = 7 PASS lines) both pass unchanged after the `internal/goldenspec` move, for a combined 13 PASS lines — the same case structure as before the move (no cases added or removed, only their internal call path changed).

**`LockedCorpusArgs` values (before/after transcription, proving the move changed no value):** identical to the `lockedCorpusArgs` map previously in `testdata/golden/gocapture/main.go` lines 73-102 — hugo (Page/Site), guava (Preconditions/ImmutableList), serilog (LoggerConfiguration/LogEvent), requests (Session/Request) — see `internal/goldenspec/spec_test.go`'s literal fixture.

## Issues Encountered

None beyond the two deviations documented above, both resolved during execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-04's `Node()`/`Explore()` extraction now has a real byte-identity oracle to prove itself against — `TestGoldensMatchLiveEngineOutput` must stay green (26/26/26) across that extraction, and `TestGoldensMatchLiveEngineOutputIsNonVacuous` guards the oracle's own comparison mechanism permanently.
- `internal/goldenspec` is now the single place to update locked capture arguments or the MCP call shape; both the generator and every future oracle consume it.
- No blockers.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: internal/goldenspec/spec.go
- FOUND: internal/goldenspec/mcp.go
- FOUND: internal/goldenspec/spec_test.go
- FOUND: testdata/golden/byte_identity_test.go
- FOUND: commit 8a911a5 (Task 1)
- FOUND: commit 9507e7e (Task 2)
