---
phase: 01-behavioral-parity-explore-node
plan: 12
subsystem: query
tags: [rwr, named-seeding, overload-disambiguation, explore, d-04, tdd]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 03)
    provides: "extractSymbolsFromQuery (H1 tokenizer) — seeding's own >=3/<=16 token filter runs on top of it"
  - phase: 01-behavioral-parity-explore-node (plan 06)
    provides: "computeGraphRelevance's seedIDs restart-vector input shape this plan's SeedIDs feeds"
provides:
  - "seedNamedSymbols: H13's end-to-end named-symbol seeding — full-scan exact-name resolution + per-overload disambiguation tiers"
  - "resolveDefsByName: getNodesByName-equivalent full-scan (NOT FTS), D-04 lowest-Id-sorted"
  - "smallOverloadSeed: <=3-def inject-all + def0/caller-ratio>=0.25*maxCallers seed tier"
  - "largeOverloadSeed / corroboratedDefs / topBySubstance: >3-def type-token-corroborated(<=4)-else-top-1-by-substance disambiguation"
  - "pascalCaseTypeTokens: the PascalCase type-token bias set, project-name excluded"
affects: [01-13, 01-16 (explore's wiring/rendering plans consume seedNamedSymbols' seed set + Primary tier for the +50 named-seed file score)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure, graphstore.Reader-driven heuristic function (no Engine dependency) — mirrors rwr.go/expand.go/gather.go's fully-unit-testable-on-synthetic-index discipline"
    - "Two-task RED/GREEN split within one shared file: Task 1 lands the <=3-def branch with the >3-def branch provisionally stubbed (skipped, not seeded); Task 2 replaces the stub — each task's own test commit precedes its own feat commit, so both individually satisfy the plan-level TDD gate"

key-files:
  created:
    - internal/query/seeding.go
    - internal/query/seeding_test.go
  modified: []

key-decisions:
  - "Body substance = a def's own line span (EndLine-StartLine+1) — no verbatim TS source for H13's exact 'body substance' measure survives in the frozen 01-RESEARCH.md capture (the live TS dist is no longer readable on this machine, per gather.go's/expand.go's established package-doc precedent for the same constraint), so this plan documents its own, cheap Reader-only proxy rather than a second disk read"
  - "Type-token corroboration correlates a def's OWNING type's Name (via traverse.go's buildContainsIndex) against the query's PascalCase tokens — deliberately does NOT also self-match a def's own Name against the same token set, since the resolved query token itself is frequently PascalCase-shaped (e.g. 'Process') and would otherwise trivially self-corroborate every def sharing that name, defeating the disambiguation entirely. Documented as this plan's own design (no verbatim TS mechanism survives) alongside the body-substance divergence"
  - "Project name is a caller-supplied string parameter (not derived internally from Engine.repoRoot) — keeps seedNamedSymbols a pure function over (Reader, query, projectName), TDD-testable against a synthetic index without an Engine fixture; the wiring plan (13/16) is expected to pass filepath.Base(repoRoot) at the Explore() call site"
  - "Task 1's provisional stub for >3-defs (silently skip the name rather than seed an undisambiguated set) was a deliberate intermediate-state choice so Task 1's own tests never depend on Task 2's not-yet-written large-overload code, and so Task 1's GREEN commit is independently correct and safe (no incorrect seeding of >3-def names) even if Task 2 were never applied"

patterns-established:
  - "seedResult/seedName: the (SeedIDs, Names[{Name, Injected, Primary}]) shape plan 13's +50 named-seed file score and plan 6's computeGraphRelevance restart vector should both consume — Injected feeds the RWR seed set, Primary is the narrower 'seed tier' subset for file-score purposes"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "H13 name resolution uses a full-scan exact-name lookup (getNodesByName-equivalent), never FTS/substring matching, and applies H13's own >=3-char/<=16-token filter on top of extractSymbolsFromQuery"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/seeding_test.go#TestSeedingResolve_ExactNameOnlyNotFTS|TestSeedingResolve_TokenMinLength|TestSeedingResolve_TokenMaxCountSixteen"
        status: pass
    human_judgment: false
  - id: D2
    description: "<=3-def small-overload branch injects ALL defs into the seed set; the seed tier is def0 (lowest-Id) plus any co-named def whose caller count is >=0.25*maxCallers, correctly excluding a below-threshold def from the tier while still injecting it"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/seeding_test.go#TestSeedingSmallOverload_BothInjectedTierByCallerRatio|TestSeedingSmallOverload_BelowThresholdExcludedFromTier|TestSeedingSmallOverload_ThreeDefsAllInjected"
        status: pass
    human_judgment: false
  - id: D3
    description: ">3-def large-overload branch prefers PascalCase-type-token-corroborated defs (matched via a def's owning type, capped at 4), falls back to the single greatest-body-substance def when nothing corroborates, and is correctly wired end-to-end through seedNamedSymbols"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/seeding_test.go#TestSeedingLargeOverload_TypeTokenCorroboration|TestSeedingLargeOverload_CorroboratedCapAtFour|TestSeedingLargeOverload_TopOneBySubstance|TestSeedingLargeOverload_NoTypeTokensFallsBackToSubstance|TestSeedingLargeOverload_WiredThroughSeedNamedSymbols"
        status: pass
    human_judgment: false
  - id: D4
    description: "The project name never biases overload selection — excluded case-insensitively from the PascalCase type-token set, and a def whose owning type happens to share the project's name is correctly denied corroboration, falling back to top-1-by-substance"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/seeding_test.go#TestSeedingProjectNameExcluded_TokenFiltered|TestSeedingProjectNameExcluded_NoFalseCorroboration"
        status: pass
    human_judgment: false

# Metrics
duration: 35min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 12: Named-Symbol Seeding + Per-Overload Disambiguation Tiers (H13) Summary

**seedNamedSymbols: H13's full-scan named-symbol resolution and per-overload disambiguation tiers (small-overload inject-all + caller-ratio tier; large-overload type-token-corroborated<=4 else top-1-by-substance), ported from RESEARCH §C.2 with two explicitly documented divergences where no verbatim TS source survives.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-07-15T13:32:00Z
- **Completed:** 2026-07-15T14:07:56Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `seedQueryTokens`/`resolveDefsByName`: H1's `extractSymbolsFromQuery` re-tokenized with H13's own >=3-char/<=16-token cap, then resolved via a full-scan exact-name lookup (NOT the FTS/gather channels H3-H6 use) — D-04 lowest-Id-sorted, matching TS's unordered `SELECT` per Assumption A3
- `smallOverloadSeed`: <=3-def "inject all" — every def seeded into the RWR restart-vector set, with a narrower "seed tier" (def0 + any co-named def whose caller count is >=0.25*maxCallers) exposed separately for plan 13's +50 named-seed file score
- `largeOverloadSeed`/`corroboratedDefs`/`topBySubstance`/`pascalCaseTypeTokens`: >3-def disambiguation — PascalCase query type tokens (project name excluded) corroborate up to 4 defs by matching a def's OWNING type's name (via `traverse.go`'s `buildContainsIndex`); when nothing corroborates, the single def with the greatest body substance (line span) wins
- `seedNamedSymbols`: the end-to-end orchestrator producing `seedResult{SeedIDs, Names[]}` — `SeedIDs` is the deduplicated, sorted seed-node-id set future plans feed into `computeGraphRelevance`'s restart vector; `Names` carries per-token `Injected`/`Primary` for plan 13's file-scoring
- All 13 tests green: `go test ./internal/query/ -run 'TestSeeding' -count=1` and the full repo suite (`go test ./...`)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED+GREEN name resolution + the <=3-defs seeding tier** - `2c4d12a` (test), `bf2e4ad` (feat)
2. **Task 2: RED+GREEN the >3-defs disambiguation** - `9dd7f7d` (test), `9c0e4df` (feat)

**Plan metadata:** (this commit)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

## Files Created/Modified
- `internal/query/seeding.go` - `seedNamedSymbols`, `seedQueryTokens`, `resolveDefsByName`, `smallOverloadSeed`, `largeOverloadSeed`, `corroboratedDefs`, `topBySubstance`, `bodySubstance`, `pascalCaseTypeTokens`, `seedResult`/`seedName` types
- `internal/query/seeding_test.go` - `seedingFakeReader` test double + 13 tests covering both disambiguation tiers, the token filters, and project-name exclusion

## Decisions Made
1. **Body substance = line span (EndLine-StartLine+1)** — the RESEARCH §C.2/H13 row names "top-1 by body substance" as the >3-def no-corroboration fallback but no verbatim TS code for the exact measure survives in the frozen capture (the live TS dist is no longer readable on this machine — the same constraint gather.go's and expand.go's package doc comments already document for their own heuristics). A def's line span is a cheap, `graphstore.Reader`-only proxy for "how much implementation this def contains," keeping the function free of a second disk read and consistent with rwr.go/expand.go/gather.go's pure-algorithm discipline. Documented explicitly in seeding.go's package doc comment as a D-02 divergence, not silently substituted.
2. **Type-token corroboration is owning-type-only, not self-name-matching** — a def corroborates only via its OWNING type's Name (the type that "contains" it as a method, reusing `traverse.go`'s existing `buildContainsIndex` rather than writing a new index). Deliberately does NOT also check a def's own Name against the type-token set: since the resolved query token itself (e.g. "Process") is frequently PascalCase-shaped, that path would trivially self-corroborate every def sharing the queried name and defeat H13's actual purpose (using a *type* token to distinguish *between* same-named overloads). This is this plan's own resolved design given RESEARCH doesn't pin the exact TS corroboration mechanism.
3. **projectName is a plain parameter, not derived from Engine.repoRoot inside this function** — `seedNamedSymbols(reader, query, projectName string)` stays a pure function testable against a synthetic index with no `Engine`/`repoRoot` fixture required, matching the plan's "Pure function over a Reader + query" instruction (extended with one extra string parameter, not two Reader-plus-query-only). The caller (a following wiring plan, 13/16) is expected to pass `filepath.Base(repoRoot)`.
4. **Task 1's >3-def branch was a deliberate provisional stub (silently skip)** — rather than seed an undisambiguated superset while Task 2's tier logic didn't yet exist, Task 1's `seedNamedSymbols` simply skips any name resolving to >3 defs. This kept Task 1's own GREEN commit independently correct (never wrong-seeds a large overload) and let Task 2's RED tests genuinely fail against not-yet-existing `largeOverloadSeed`/`pascalCaseTypeTokens`/`largeOverloadCorroboratedCap` symbols (a real compile-failure RED, not just an assertion failure), confirmed via `go test ./internal/query/ -run 'TestSeedingLargeOverload|TestSeedingProjectNameExcluded' -count=1` failing to build before the Task 2 GREEN commit.

## Deviations from Plan

### Auto-fixed Issues
None - no bugs, missing functionality, or blocking issues were encountered; both tasks landed as scoped.

### Documented Divergences (not auto-fixes — pre-declared by the plan's own D-02/D-10 framework)
1. **[D-02] Body-substance measure** — see Decision 1 above. No user sign-off needed per D-10's own text ("where a heuristic depends on data Go genuinely cannot produce, document it as an explicit D-02 allowed divergence") — this is the same class of divergence 01-10/01-11's plans already exercised (H10-H12's traversal admission order) for the identical "TS dist no longer readable" constraint.
2. **[D-02] Type-token corroboration mechanism** — see Decision 2 above, same divergence class.

## Issues Encountered
None. `gofmt` flagged a trivial struct-literal column-alignment diff in `seeding_test.go` after Task 2's additions widened a map's key column (an artifact of `gofmt`'s alignment algorithm, not a logic issue) — fixed via `gofmt -w` and folded into Task 2's feat commit (noted in that commit message).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `seedNamedSymbols`'s `seedResult{SeedIDs, Names[]}` is ready for the explore-wiring plan (13/16) to consume: `SeedIDs` feeds `computeGraphRelevance`'s (rwr.go, plan 6) restart-vector `seedIDs` parameter; each `seedName.Primary` is the subset plan 13's H14 +50 named-seed file score should key off (not `Injected`, when a small overload injects a caller-ratio-excluded def).
- The wiring plan is expected to supply `projectName` as `filepath.Base(e.repoRoot)` (or equivalent) at the `Explore()` call site — this plan intentionally left that derivation out of scope, keeping `seedNamedSymbols` a pure, Engine-independent function.
- Golden-corpus validation against the real re-indexed corpus remains deferred to plan 17, consistent with plan 6's/plan 11's own scoping notes.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

All created files (`internal/query/seeding.go`, `internal/query/seeding_test.go`, this SUMMARY.md) and all commit hashes (`2c4d12a`, `bf2e4ad`, `9dd7f7d`, `9c0e4df`, `7d86fa7`) verified present via `git log --oneline --all`.
