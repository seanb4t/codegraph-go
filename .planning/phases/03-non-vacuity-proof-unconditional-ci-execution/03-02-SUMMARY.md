---
phase: 03-non-vacuity-proof-unconditional-ci-execution
plan: 02
subsystem: testing
tags: [mutation-rehearsal, golden-suite, non-vacuity, corpora, fixture-freeze, go-test]

# Dependency graph
requires:
  - phase: 02-golden-harness-re-authoring-re-freeze
    provides: the re-frozen golden suite (26 goldens) + the CASES.json-driven behavioral assertions — the suite being proven
  - phase: 03-non-vacuity-proof-unconditional-ci-execution
    provides: plan 03-01's unconditional golden CI job + executed-scenario-count self-assertions (the suite in its final CI form)
provides:
  - 03-MUTATION-LOG.md — the per-family mutation → observed RED → byte-clean revert record (FIXT-07), with the D-01 re-mutate call and corpus-precondition header
affects: [phase 05 process/CI sweep, FIXT-07]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 6413   # chars/4 over the realized diff (03-MUTATION-LOG.md 13923 chars + this SUMMARY 11728 chars = 25651/4)
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "apply → run family RED → revert byte-clean → record per family, one targeted mutation per assertion family (D-01)"
    - "EXIT-trap-guarded rename for an out-of-tree corpus mutation, restoring the exact sha-bearing path on any exit"
    - "pre-mutation git diff --quiet gate on every tracked-file mutation and revert (never a destructive blind checkout)"

key-files:
  created:
    - .planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md
  modified: []

key-decisions:
  - "RE-MUTATE all five families this phase (D-01 call): (b) had a count-only prior record (02-04-SUMMARY 25/26), (d)/(e) were specified-but-without-transcript (01-07-SUMMARY:24) — full pasted RED output per family recorded in one artifact, prior records cited as corroboration only"
  - "Family (c) is recorded as a shared-comparison-loop demonstration (TestExploreCLIMatchesMCP diverged; the trio shares the cliOut != mcpOut shape), not a per-sibling claim"
  - "Nothing lands: every mutation reverted byte-clean, final commit carries no mutation byte in testdata/golden/, corpus/, corpora/, or internal/"

patterns-established:
  - "A gate is trusted only after it has been demonstrated RED against a confirmed-applied mutation — recorded per family with the pasted failure as the acceptance condition (T-03-P2-03)"

requirements-completed: [FIXT-07]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "FIXT-07 five-family mutation rehearsal — each of the five assertion families ((a) CASES.json behavioral, (b) golden byte-identity, (c) CLI==MCP trio, (d) hermetic fail-loud resolution, (e) coverage guard) red-demonstrated against the suite in final form with pasted observed failure and byte-clean revert, recorded in 03-MUTATION-LOG.md; suite green afterward (26/26 goldens)."
    requirement: FIXT-07
    verification:
      - kind: manual_procedural
        ref: ".planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md (five family rows with pasted RED output + revert commands)"
        status: pass
      - kind: unit
        ref: "go test -count=1 ./testdata/golden/... (green after rehearsal — 26/26)"
        status: pass
      - kind: unit
        ref: "go test -count=1 ./internal/corpora/... -run TestCorpusCoverageClaim (green after revert)"
        status: pass
      - kind: unit
        ref: "go test -count=1 ./internal/upgrade/... (green)"
        status: pass
    human_judgment: true
    rationale: "The deliverable is the recorded RED evidence in 03-MUTATION-LOG.md. Automation re-runs the suite green (proving reversibility) but cannot re-derive that each family was OBSERVED red — a human must read the pasted failing output per family."

# Metrics
duration: 65min
completed: 2026-08-15
status: complete
---

# Phase 03 Plan 02: Five-Family Mutation Rehearsal Summary

**FIXT-07 proven: each of the five assertion families ((a) CASES.json behavioral, (b) golden byte-identity, (c) CLI==MCP trio, (d) hermetic fail-loud resolution, (e) coverage guard) was red-demonstrated against the suite in its final form, reverted byte-clean, and recorded with pasted failing output in 03-MUTATION-LOG.md — nothing landed.**

## Performance

- **Duration:** ~65 min
- **Started:** 2026-08-15T14:39:53Z
- **Completed:** 2026-08-15T15:05:00Z
- **Tasks:** 3
- **Files modified:** 1 (03-MUTATION-LOG.md) + 1 created (this SUMMARY)

## Accomplishments

- **All five assertion families red-demonstrated (D-01 RE-MUTATE call):** each family received one targeted mutation, was observed RED with the failing test named, and was reverted byte-clean:
  - (a) `TestCorpusBehaviorSynthetic/a-overloaded-same-named-symbols` — defs boundary `!= 2` → `!= 1` → `Node("Validate"): got 2 defs, want 2`.
  - (b) `TestReFrozenGoldensValid` — deleted `corpus/hugo/go-explore.json` → `25/26 goldens verified` naming the missing golden.
  - (c) `TestExploreCLIMatchesMCP/behavioral` — MCP-side one-word query suffix on the behavioral row only → `CLI and MCP output diverge (EXPL-05)` with all four locked rows green.
  - (d) `TestPriorityLanguagesResolveToLockedCorpus/go,+tsjs` — renamed the hugo tree under an EXIT trap → `lockedCorpusDir("go"): ... not found ... run 'task corpora:fetch'` (fail-NOT-skip).
  - (e) `TestCorpusCoverageClaim` — `calls` threshold 29406 → 999999 → `kind calls derived count 58812 below threshold 999999`.
- **Corpus precondition honored (review HIGH):** `task corpora:fetch` (4/4 confirmed) + `task corpora:assert` (4/4 verified) ran before any rehearsal; the resolved `CODEGRAPH_CORPUS_DIR` (`/Users/sean/.cache/codegraph/corpora`) is recorded in the log header so each observed RED is attributable to the mutation, never to a missing locked tree.
- **Pre-mutation cleanliness gate never fired (review finding):** `git diff --quiet -- <file>` asserted before every tracked-file mutation and revert across all five families; exit 0 every time, so no pre-existing tracked edit was overwritten and no revert was a blind destructive checkout.
- **Reversibility proven:** after all rehearsals, `git status --porcelain` is clean except the log artifact; `git diff 09c4279..HEAD -- testdata/ corpus/ corpora/ internal/` is empty (no mutation byte landed); the full suite passes green afterward.
- **03-MUTATION-LOG.md created as the single visible artifact:** header records the D-01 re-mutate call and the corroborating prior records (01-04-SUMMARY:99, 01-07-SUMMARY:24, 02-04-SUMMARY:34), the corpus-precondition evidence, the cleanliness-gate result, five family rows each with pasted observed failure + revert command + byte-clean proof, and a conclusion row (suite ran 26/26 goldens, red in all five families observed).

## Task Commits

Each task was committed atomically:

1. **Task 1: Rehearse families (a) and (c)** - `c7f9424` (docs: record families (a) and (c) mutation rehearsal)
2. **Task 2: Rehearse families (b) and (d)** - `a13d7c3` (docs: record families (b) and (d) mutation rehearsal)
3. **Task 3: Rehearse family (e) and complete the log** - `ac8d7fb` (docs: complete 03-MUTATION-LOG with family (e) and FIXT-07 conclusion)

**Plan metadata:** this SUMMARY is committed in the final docs commit (solo worktree wave; orchestrator owns shared state writes).

## Files Created/Modified

- `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` - The complete per-family record: header (D-01 call, corpus precondition, resolved `CODEGRAPH_CORPUS_DIR`, cleanliness-gate result), five family rows (mutation applied, pasted observed failure, revert command, byte-clean proof), and the FIXT-07 conclusion row.

## Decisions Made

- **RE-MUTATE all five families this phase** (the D-01 call): family (b) had a count-only prior record (02-04-SUMMARY:34 "25/26") and families (d)/(e) had "specified" RED demonstrations with no observed-failure transcript (01-07-SUMMARY:24). Re-mutating everything against the suite in final form with full pasted output is the cheapest complete reading of "the observed failure … recorded per family"; prior records are cited as corroboration in the log header.
- **Family (c) recorded as a shared-comparison-loop demonstration:** only `TestExploreCLIMatchesMCP` was mutated, because the trio's three tests share the `cliOut != mcpOut` comparison shape. The log explicitly does not claim each sibling failed individually.
- **Family (d) ran under an EXIT trap in one shell invocation:** the rename → run → restore was a single `bash` invocation with `trap 'mv "$orig.muttmp" "$orig"' EXIT`, restoring the exact sha-bearing hugo path on any exit (T-03-P2-07), and a pure `mv` rename never a copy/delete (T-03-P2-02).

## Deviations from Plan

None of substance — the plan executed as written; every mutation was applied, observed red, and reverted byte-clean, and nothing landed. Two documentation-shape notes:

- The plan's task-3 prose cites the committed `calls` threshold as 29405, but the actual committed value in `corpora/selection.json` is 29406. The mutation target was "a value above the observed best" (999999), which is satisfied regardless of the exact committed base; the mutation used the real value 29406 → 999999. No behavioral impact.
- The observed family-(a) failure message retains the literal `want 2` text because only the boundary conditional was mutated (`!= 2` → `!= 1`), not the message string — the plan's intent (a deliberately wrong expected boundary red-demonstrating the defs-count assertion) is exactly what occurred.

## Issues Encountered

- None. The corpus precondition passed on the first try (all four locked corpora already cached and verified); every family went red exactly as designed on the first mutation, and every revert restored byte-clean.

## Known Stubs

None. The plan lands no code — only the mutation log and this summary. No placeholder values, no empty data paths, no TODOs.

## Threat Flags

None. The rehearsal touched no new network endpoint, auth path, file-access pattern, or schema change beyond the plan's modeled register (T-03-P2-01..07). The transient family-(d) rename and family-(b) golden delete were restored byte-clean before any commit.

## Self-Check: PASSED

- Created files exist: `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` (verified `test -f`).
- Commits exist: `c7f9424`, `a13d7c3`, `ac8d7fb` (verified via `git log --oneline`).
- Working tree clean: `git status --porcelain` returns nothing except the log before the final docs commit.
- No mutation bytes landed: `git diff 09c4279..HEAD --stat -- testdata/ corpus/ corpora/ internal/` is empty.
- Phase gate green after rehearsal: `go test -count=1 ./testdata/golden/...` (ok, 26/26), `go test -count=1 ./internal/corpora/...` (ok), `go test -count=1 ./internal/upgrade/...` (ok).

## Next Phase Readiness

- FIXT-07 is closed: the suite is proven non-vacuous with a recorded per-family RED demonstration, and the final-form suite is green. Phase 03 is complete (03-01 CI wiring + 03-02 non-vacuity proof).
- Phase 5 (Process, CI & In-Tree Sweep) can proceed: it waits on this phase so no in-tree comment change shares a diff with a golden change — the golden suite and its CI wiring are frozen, and this rehearsal changed none of them.
- The 03-MUTATION-LOG.md is the standing evidence artifact referenced by `/gsd-verify-work` and any future audit of the suite's non-vacuity.

---
*Phase: 03-non-vacuity-proof-unconditional-ci-execution*
*Plan: 02*
*Completed: 2026-08-15*
