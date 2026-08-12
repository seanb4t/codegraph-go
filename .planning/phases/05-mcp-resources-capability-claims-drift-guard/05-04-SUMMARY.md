---
phase: 05-mcp-resources-capability-claims-drift-guard
plan: 04
subsystem: mcp
tags: [mcp, go-sdk, resources, wire-oracle, mutation-testing]

# Dependency graph
requires:
  - phase: "05-01: resourcesFS embed seam, resourceURIFor, registerResources, resourcesListRequest/resourceReadRequest helpers, assertResourceCacheControl anchor"
  - phase: "05-02: full 10-resource catalog (8 per-tool fact-sheets + 2 behavior docs)"
  - phase: "05-03: resources_schema_drift_test.go (GUARD-01/02), Mutations 7-9"
provides:
  - "9 new wire-oracle scenarios (resources-read-node/search/callers/callees/impact/files/status/tools-filter/index-state) — one Index:false resources/read scenario per resource URI not covered by the 05-01 tracer, each independently proving RSRC-03's never-indexed-repository property"
  - "resources-list-no-index: full 10-entry catalog advertised with no index, paired with the pre-existing toolslist-no-index transcript as criterion 2's proof"
  - "resources-read-unknown: -32602 invalid-params for an unregistered resource URI, anchored, T-05-02's mitigation observed on the wire"
  - "TestEveryAdvertisedResourceURIHasASuccessfulReadScenario: derives the required URI set from a live resources-list capture (never hand-typed) and requires every one to have a successful wire read"
  - "Mutations 10 and 11 in test/wireoracle/MUTATION-PROOF.md: real-tree demonstrated-red proof that the oracle is non-vacuous (capability explicit-zero-value removal) and that RSRC-03's index-independence is structural (registerResources call-site move)"
  - "COVERAGE-BASELINE.md finalized at 42 scenarios with the complete 13-row MCP Resources category table"
affects: []

# Actuals (#2632)
actuals:
  tokens: 14200
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Structural URI-coverage guard (TestEveryAdvertisedResourceURIHasASuccessfulReadScenario) derives its expected set from a LIVE wire capture rather than a hand-typed list — the exact same shape as the pre-existing TestEveryRegisteredToolHasASuccessfulCallScenario, applied to resources/read instead of tools/call"

key-files:
  created: []
  modified:
    - test/wireoracle/scenarios.go
    - test/wireoracle/oracle_test.go
    - test/wireoracle/anchors.go
    - test/wireoracle/MUTATION-PROOF.md
    - test/wireoracle/COVERAGE-BASELINE.md
    - testdata/wireoracle/transcripts/*.golden (11 new, 0 moved)

key-decisions:
  - "Mutations numbered 10 and 11, not 9 and 10 as the plan's Task 3 text assumed — 05-03-PLAN's own Task 3 already occupied Mutation 9 by the time this task ran. Documented as a numbering note in MUTATION-PROOF.md, matching 05-03-SUMMARY.md's own precedent for the identical situation."
  - "COVERAGE-BASELINE.md's MCP Resources table keeps resources-read-unknown in this category rather than moving it to 'Error shapes', per the plan's explicit instruction — the resources surface reads as one block."

patterns-established: []

requirements-completed: [RSRC-01, RSRC-02, RSRC-03]

coverage:
  - id: D1
    description: "A live client against a real serve --mcp subprocess gets non-empty text/markdown from resources/read on every one of the 10 URIs resources/list advertised, observed on the wire from a real spawned binary — never via the server's own Go API"
    requirement: RSRC-02
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch (all 10 resources-read-* scenarios)"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestEveryAdvertisedResourceURIHasASuccessfulReadScenario"
        status: pass
    human_judgment: false
  - id: D2
    description: "In a directory with no .codegraph/, resources/list returns the full 10-entry catalog and resources/read serves content — frozen on the wire alongside the existing toolslist-no-index transcript"
    requirement: RSRC-03
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/resources-list-no-index, TestFrozenTranscriptsMatch/resources-read-{node,search,callers,callees,impact,files,status,tools-filter,index-state}"
        status: pass
    human_judgment: false
  - id: D3
    description: "resources/read on an unregistered URI returns -32602, matching the codebase's existing error-shape scenarios"
    verification:
      - kind: integration
        ref: "test/wireoracle TestSpecAnchorsHold/resources-read-unknown (assertErrorCode against codeInvalidParams)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The oracle is re-proved non-vacuous against the re-frozen corpus by a confirmed-applied mutation, reverted byte-clean"
    verification:
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutation 10 — Resources: &mcp.ResourceCapabilities{} deleted, 38 scenarios observed red, reverted byte-clean"
        status: pass
    human_judgment: false
  - id: D5
    description: "RSRC-03's index-independence is proved structural (not incidental) by a demonstrated-red mutation"
    verification:
      - kind: other
        ref: "test/wireoracle/MUTATION-PROOF.md Mutation 11 — registerResources(s) moved inside if hasIndex, 10 Index:false scenarios red, 2 Index:true scenarios green, plus internal/mcp TestResourcesRegisterWithoutIndex red at the unit tier"
        status: pass
    human_judgment: false
  - id: D6
    description: "COVERAGE-BASELINE.md and ExpectedScenarioCount agree at 42, and main is green at the phase boundary"
    verification:
      - kind: unit
        ref: "test/wireoracle TestScenarioCountIsExact, TestTranscriptSetMatchesScenarioSet"
        status: pass
      - kind: integration
        ref: "go test ./... -count=1 (repo-wide, on the committed tree after Task 3's mutations reverted)"
        status: pass
    human_judgment: false

duration: ~32min
completed: 2026-08-12
status: complete
---

# Phase 5 Plan 4: MCP Resources Wire Coverage Completion Summary

**Froze wire reads for the remaining 9 resource URIs, the unindexed-catalog and unknown-URI error scenarios, a structural URI-coverage guard, and two demonstrated-red-then-reverted mutations proving the oracle non-vacuous and RSRC-03 structural — closing the phase's full-corpus wire coverage at exactly 42 scenarios.**

## Performance

- **Duration:** ~32 min
- **Started:** 2026-08-12T18:50:00Z (approx.)
- **Completed:** 2026-08-12T19:22:12Z
- **Tasks:** 3
- **Files modified:** 16 (5 code/doc files, 11 new golden transcripts)

## Accomplishments
- 9 new `resources-read-*` scenarios (`node`, `search`, `callers`, `callees`, `impact`, `files`, `status`, `tools-filter`, `index-state`), each `Index: false` — the cheapest scenarios in the corpus, and each independently proving RSRC-03's criterion 2 property that `resources/read` serves content in a never-indexed repository. `ExpectedScenarioCount` 31 → 40.
- `resources-list-no-index` (full 10-entry catalog with no index, paired with `toolslist-no-index`) and `resources-read-unknown` (`-32602` for an unregistered URI, anchored via `assertErrorCode`) — `ExpectedScenarioCount` 40 → 42.
- `resourceURIsFromCapture`, `findResourceReadRequest`, `successfulResourceRead`, and `TestEveryAdvertisedResourceURIHasASuccessfulReadScenario` in `oracle_test.go` — derives the required URI set from a LIVE `resources-list` capture (never hand-typed), so an 11th resource added later turns this test red until it also gets wire coverage.
- Mutations 10 (resources capability's explicit zero value removed — 38 scenarios observed red, the SDK's `ListChanged: true` auto-population confirmed on the wire) and 11 (`registerResources` moved inside `if hasIndex` — 10 scenarios red, 2 green, the exact asymmetry that proves RSRC-03 is structural) in `MUTATION-PROOF.md`, both confirmed-applied, observed red with verbatim failure text, and reverted byte-clean.
- `COVERAGE-BASELINE.md` finalized: header count 42, complete 13-row MCP Resources category table (was a partial 2-row table), Total line arithmetic, closing History bullet for the whole phase.

## Task Commits

1. **Task 1: One wire read scenario per remaining resource URI** - `92a8281` (test)
2. **Task 2: Unindexed catalog, unknown-URI error shape, structural coverage test** - `a2cc1e9` (test)
3. **Task 3: Re-prove the oracle non-vacuous, finalize the coverage baseline** - `44f479c` (test)

## Files Created/Modified
- `test/wireoracle/scenarios.go` - 11 new `Scenario` entries, `ExpectedScenarioCount` 31→42
- `test/wireoracle/oracle_test.go` - `resourceURIsFromCapture`, `findResourceReadRequest`, `successfulResourceRead`, `TestEveryAdvertisedResourceURIHasASuccessfulReadScenario`
- `test/wireoracle/anchors.go` - `resources-read-unknown`'s `-32602` anchor
- `test/wireoracle/MUTATION-PROOF.md` - Mutations 10, 11, and a numbering note explaining the shift from the plan's assumed 9/10
- `test/wireoracle/COVERAGE-BASELINE.md` - header count, complete MCP Resources table, Total line, History bullet
- `testdata/wireoracle/transcripts/*.golden` - 11 new files (9 per-URI reads + `resources-list-no-index` + `resources-read-unknown`); 0 pre-existing transcripts moved

## Decisions Made
- Mutations numbered 10 and 11 (not 9 and 10 as the plan's Task 3 text assumed) — see Deviations.
- `resources-read-unknown` stays in the "MCP Resources" category table rather than "Error shapes", per the plan's explicit instruction, so the resources surface reads as one block.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Mutation numbering continues from 10, not 9**
- **Found during:** Task 3
- **Issue:** The plan's Task 3 text says "these are Mutations 9 and 10" and its acceptance criteria literally check `rg -c '^## Mutation' … returns 10` with the capability-off proof at `## Mutation 9`. By the time this task ran, `test/wireoracle/MUTATION-PROOF.md` already had NINE mutations (05-03-PLAN's own Task 3 appended Mutations 7-9 before this plan executed) — the plan's assumption was stale, identical in shape to 05-03-SUMMARY.md's own documented deviation for the same reason one plan earlier.
- **Fix:** Appended the two new mutations as Mutations 10 and 11 instead, preserving Mutation 9 untouched. Added a dedicated "A note on numbering (05-04-PLAN Task 3)" section to `MUTATION-PROOF.md` immediately before the new mutations, stating the discrepancy plainly and giving the corrected verification commands (`rg -c '^## Mutation' … returns 11`, with the capability-off/non-vacuity proof at `## Mutation 10` and the registration-gating/RSRC-03-structural proof at `## Mutation 11`). Both properties named in the plan's success criteria (non-vacuity re-proof, RSRC-03 index-independence) are demonstrated regardless of the number attached to each.
- **Files modified:** `test/wireoracle/MUTATION-PROOF.md`
- **Commit:** `44f479c`

**2. [Rule 1 - Documentation accuracy] COVERAGE-BASELINE.md's `resources-read-` count check is unreachable as literally specified**
- **Found during:** Task 3
- **Issue:** The plan's acceptance criteria check `rg -c 'resources-read-' test/wireoracle/COVERAGE-BASELINE.md` returns exactly 10. `resources-read-unknown` is REQUIRED to be a row in this category's table per the plan's own instruction ("kept in this category... so the resources surface reads as one block"), and its name inherently matches the same `resources-read-` substring as the 10 valid per-URI reads — making 11 the true structural minimum for the table alone, before counting any prose. The actual count (18) also includes pre-existing History-section prose (from plans 05-01/05-02, not touched by this plan) and this plan's own new History/intro prose, both of which follow the file's established convention of naming scenarios in prose in addition to the table.
- **Fix:** Left the table and prose as the clearest, most accurate representation of the corpus; did not strip informative content chasing an arithmetically-unreachable exact count. Documented the discrepancy here rather than silently reconciling it. The companion criterion (`rg -c 'resources-list' … returns at least 2`) is satisfied (actual: 9).
- **Files modified:** `test/wireoracle/COVERAGE-BASELINE.md`
- **Commit:** `44f479c`

---

**Total deviations:** 2 auto-fixed (1 Rule 3 — blocking issue caused by a prior plan's mutation count not matching this plan's stale assumption; 1 Rule 1 — a documentation acceptance check whose literal target number was arithmetically unreachable given the plan's own content requirement)
**Impact on plan:** No scope change. All three tasks' behavior, acceptance criteria (re-verified against the actual repository state), and the plan's five success criteria are fully satisfied; only mutation numbering and one documentation line-count differ from the plan's literal text.

## Issues Encountered
The pre-existing, documented, load-dependent `internal/daemon` flake (`STATE.md`'s "Daemon extreme-load tail (ACCEPTED, not a gap)") fired twice during this plan's `go test ./... -count=1` runs under concurrent workstation load — once as `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`-family contention, once as `TestRunWatchdogCancelsRunOnSimulatedReparent`. Both times `go test ./internal/daemon/... -count=1` in isolation passed clean. Neither file this plan touches (`internal/mcp`, `test/wireoracle`, `testdata/wireoracle`) was involved.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 5's full success-criteria set is closed: all 10 advertised resource URIs have frozen wire proof of non-empty `text/markdown` reads (criterion 1), the unindexed-repository case is frozen alongside the existing empty-tool-list transcript (criterion 2), the unknown-URI error shape matches the codebase's existing `-32602` convention (criterion 3, implicit in criterion 1's wording), and the oracle is re-proved non-vacuous with RSRC-03's index-independence demonstrated structurally (criterion 5).
- `test/wireoracle/MUTATION-PROOF.md` now carries 11 mutations total; any future contributor extending this file should continue the sequence from 12, not reuse 10/11.
- `ExpectedScenarioCount` is 42, `COVERAGE-BASELINE.md` agrees, and `go test ./...`/`task test:wireoracle` are green on the committed tree with no re-freeze outstanding.

---
*Phase: 05-mcp-resources-capability-claims-drift-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

- FOUND: `testdata/wireoracle/transcripts/resources-read-node.golden`
- FOUND: `testdata/wireoracle/transcripts/resources-list-no-index.golden`
- FOUND: `testdata/wireoracle/transcripts/resources-read-unknown.golden`
- FOUND commit `92a8281` (Task 1, test)
- FOUND commit `a2cc1e9` (Task 2, test)
- FOUND commit `44f479c` (Task 3, test)
- FOUND commit `ddb4370` (docs: SUMMARY.md)
