---
phase: 01-behavioral-parity-explore-node
plan: 10
subsystem: api
tags: [explore, hybrid-search, scoring, rerank, go]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: "internal/query/gather.go (01-07) — gatherChannel1/2/3 + gatherMerge's merged candidate set, and the shared isTestFile path predicate this plan's H7 reuses"
provides:
  - "internal/query/gather.go — applyTestFileDampening (H7), applyCoreDirectoryBoost (H8), applyMultiTermReRank (H9), distinctiveExactMatchExemptIDs, and the applyPostMergeRerankers composition that wires all three together in RESEARCH's pipeline order"
affects: ["01-11 (subgraph expansion consumes the fully re-ranked candidate set)", "01-14 (relevance gate reuses isTestFile alongside these rerankers' scoring)"]

tech-stack:
  added: []
  patterns:
    - "Post-merge rerankers as pure, in-place-mutating functions over []gatherCandidate — no reader/graph access, consistent with plan 07's channel/merge shape"
    - "Order-sensitive composition (applyPostMergeRerankers): the H9 exemption set is computed BEFORE H7 runs so a later heuristic's exemption can gate an earlier heuristic's dampening — documented explicitly since it inverts the naive 'H7 then H8 then H9' reading of the pipeline table"

key-files:
  created: []
  modified:
    - internal/query/gather.go
    - internal/query/gather_test.go

key-decisions:
  - "H8's 'per-file edge count' is implemented as a per-file CANDIDATE-COUNT proxy, not a true graph-edge count — applyCoreDirectoryBoost is a pure function over the already-merged candidate set (RESEARCH's own framing: 'Pure functions over the candidate set, TDD against synthetic candidates'), with no graphstore.Reader access to compute real edge counts at this stage. Documented in-code; if a future plan needs true edge-weighted dominance, only applyCoreDirectoryBoost's counting loop needs to change — no call-site impact."
  - "H9's 'stem-grouped term groups' uses a new, documented lightweight suffix-stripping stemTerm() (plural -s/-es/-ies, verb -ing/-ed), not a full Porter-stemmer port and not TS's getStemVariants() — 01-03 already deliberately deferred getStemVariants() (D-02, see tokenize.go), and RESEARCH §C.2/H9 pins only the grouping OUTCOME and score multiplier, not a specific stemming algorithm."
  - "isDistinctiveIdentifier is a documented heuristic substitute (>=6 runes AND (contains '_' OR an internal case transition)) for TS's uncited isDistinctiveIdentifier (query-utils.js) — RESEARCH's frozen citation set names the function but the TS dist JS (which would show its real implementation) is no longer present on this machine (see 01-07-SUMMARY.md's same finding). The substitute captures the one behavior RESEARCH DOES pin: distinctive identifiers are structured multi-word symbol names, not short/common English words, and are exempt from H7's test-file dampening when they exact-match a query term."
  - "H7/H9 composition order is deliberately NOT the pipeline table's literal H7-then-H8-then-H9 reading: applyPostMergeRerankers computes the H9 distinctive-identifier exemption set FIRST, then runs H7 (passing the exemption in), then H8, then H9's multiplier — this is the only order that lets the exemption actually gate the dampening, per the plan's explicit order-sensitivity note."

patterns-established:
  - "applyTestFileDampening's exemptIDs parameter was added in Task 1 (H7) even though nothing populated it yet, specifically so Task 2 (H9) could wire in the exemption without redefining or duplicating H7 — a forward-compatible signature added ahead of its first real caller."

requirements-completed: [EXPL-02, EXPL-03]

coverage:
  - id: D1
    description: "H7 test-file dampening: score *= 0.3 for test-file candidates, short-circuited entirely when the query mentions test/spec"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherTestDampen"
        status: pass
    human_judgment: false
  - id: D2
    description: "H8 core-directory boost: +25 to every candidate sharing a >=3x-dominant file's directory prefix, including nested subdirectories; no-op below the 3x threshold"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherCoreDirBoost"
        status: pass
    human_judgment: false
  - id: D3
    description: "H9 multi-term co-occurrence rerank: score *= 1+matchCount*0.5 when >=2 stem-grouped term groups match a candidate's name/directory"
    requirement: "EXPL-03"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestGatherMultiTermReRank"
        status: pass
    human_judgment: false
  - id: D4
    description: "Distinctive-identifier exact matches are exempt from H7's test-file dampening, verified both via the exemption-set helper directly and via the full applyPostMergeRerankers composition (order-sensitive: exemption computed before H7 runs)"
    requirement: "EXPL-03"
    verification:
      - kind: unit
        ref: "internal/query/gather_test.go#TestDistinctiveIdentifierExemption"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 10: Post-Merge Gather Rerankers (H7-H9) Summary

**Test-file dampening (×0.3), dominant-directory boost (+25), and multi-term co-occurrence re-rank (×1+0.5×matchCount) ported into `internal/query/gather.go`, composed with a distinctive-identifier exemption that deliberately runs before the dampening it gates.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-15T14:35:00Z
- **Completed:** 2026-07-15T14:55:50Z
- **Tasks:** 2 (each RED → GREEN)
- **Files modified:** 2 (both extended, not created)

## Accomplishments
- `applyTestFileDampening` (H7): multiplies a test-file candidate's score by `testFileDampeningFactor` (0.3), skipped entirely (for every candidate) when the raw query text mentions "test" or "spec"; accepts an `exemptIDs` set from day one so H9's exemption could be wired in without redefining H7
- `applyCoreDirectoryBoost` (H8): finds the file with the most candidates, checks it against the next-most-frequent distinct file at the exact `coreDirectoryDominanceRatio` (3.0) threshold, and adds a flat `coreDirectoryBoost` (+25) to every candidate whose file shares that dominant file's directory prefix — including nested subdirectories, via `sharesDirectoryPrefix`
- `applyMultiTermReRank` (H9): groups query terms by a lightweight `stemTerm` (plural/verb suffix stripping), counts how many distinct groups match a candidate's Name or file directory, and multiplies score by `1 + matchCount*multiTermRerankStep` (0.5) when `matchCount >= multiTermRerankMinGroups` (2)
- `isDistinctiveIdentifier` + `distinctiveExactMatchExemptIDs`: a documented heuristic (>=6 runes, underscore or internal case transition) flags exact query-term matches against a candidate's Name as exempt from H7's dampening
- `applyPostMergeRerankers`: the order-sensitive composition — computes the exemption set BEFORE running H7, then H8, then H9's multiplier, then re-sorts (score-desc/Id-asc, D-04)

## Task Commits

Each task's RED and GREEN steps were committed atomically:

1. **Task 1: H7 test-file dampening + H8 core-directory boost** — RED `6c563a5` (test), GREEN `be67068` (feat)
2. **Task 2: H9 multi-term rerank + distinctive-identifier exemption** — RED `4b4059a` (test), GREEN `ea5f96d` (feat)

_TDD gate sequence verified: `test(...)` commits precede their `feat(...)` commits in git log for both tasks._

## Files Created/Modified
- `internal/query/gather.go` - added `testFileDampeningFactor`/`coreDirectoryBoost`/`coreDirectoryDominanceRatio`/`multiTermRerankStep`/`multiTermRerankMinGroups`/`distinctiveIdentifierMinLength` consts; `queryMentionsTestOrSpec`, `applyTestFileDampening`, `fileDir`, `sharesDirectoryPrefix`, `applyCoreDirectoryBoost`, `stemTerm`, `groupTermsByStem`, `applyMultiTermReRank`, `isDistinctiveIdentifier`, `distinctiveExactMatchExemptIDs`, `applyPostMergeRerankers`
- `internal/query/gather_test.go` - `TestGatherTestDampen`, `TestGatherCoreDirBoost`, `TestGatherMultiTermReRank`, `TestDistinctiveIdentifierExemption`

## Decisions Made
- See key-decisions above for the four documented substitutions/design choices (H8's candidate-count proxy for edge count, H9's lightweight stemmer, the isDistinctiveIdentifier heuristic, and the deliberate H9-before-H7 composition order).

## Deviations from Plan

None — plan executed exactly as written. The plan itself anticipated (and explicitly called out) that RESEARCH doesn't pin verbatim source for H8's "edge count" definition, H9's stemming algorithm, or isDistinctiveIdentifier's exact rule (only the score-arithmetic constants and grouping/exemption OUTCOMES are cited verbatim in RESEARCH §C.2), so the documented substitutions above are within the plan's own stated scope, not unplanned deviations.

## Issues Encountered
`go test ./...` showed one unrelated flaky failure in `internal/daemon` (a debounce-timing goroutine panic in `TestDaemon...flush`/`fire`) that this plan's files (`internal/query/gather.go`, `gather_test.go`) cannot have caused — neither `internal/daemon` nor `internal/watch` was touched. Re-running `go test ./internal/daemon/... -count=1` in isolation passed cleanly, confirming a pre-existing flaky/racy test unrelated to this plan's scope (out of scope per the executor's scope-boundary rule — not fixed here).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `applyPostMergeRerankers` is the single entry point plan 11 (subgraph expansion) should call on `gatherMerge`'s output before building the RWR seed set — it takes the merged candidates, the raw query string, and the FTS terms list (`extractSearchTerms`'s output), and returns the fully H7-H9-reranked, re-sorted candidate slice.
- None of H7-H9 are wired into `Explore()` yet, by design (mirrors 01-07-SUMMARY.md's same note) — that wiring, plus H10+'s type-hierarchy expansion and beyond, is later plans' work.
- `isTestFile` (01-07), `applyTestFileDampening`'s exemption plumbing, and `distinctiveExactMatchExemptIDs` are all ready for plan 14's relevance gate (H17) to reuse directly if it needs the same test-file/distinctive-identifier signals.

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED
