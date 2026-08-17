---
phase: 05-process-ci-in-tree-sweep
plan: 06
subsystem: testing
tags: [golden-fixtures, corpus, behavioral-testing, re-freeze]

requires:
  - phase: 05-process-ci-in-tree-sweep
    provides: "05-01's testdata/golden/README.md ts-* rewrite and golden-suite scaffold, which this plan's re-freeze runs against unchanged"
provides:
  - "corpus/behavioral/src/{accounts/manager.go, accounts/validate.go, orders/validate.go, recovery/recovery_test.go} comment rows reworded from 'synthetic-parity corpus' to 'behavioral corpus' (D-03/D-07)"
  - "corpus/behavioral/go-explore-multi.json re-frozen via task golden:regen, embedding the reworded comment bytes; go-node-multi.json and CASES.json proven byte-unchanged"
affects: []

actuals:
  tokens: 2341
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Small attributed golden re-freeze: a comment reword and its forced re-freeze land as two atomic commits in the same plan, with the diff asserted (not eyeballed) to be exactly the source-comment files plus the one transcript that embeds them"

key-files:
  created: []
  modified:
    - corpus/behavioral/src/accounts/manager.go
    - corpus/behavioral/src/accounts/validate.go
    - corpus/behavioral/src/orders/validate.go
    - corpus/behavioral/src/recovery/recovery_test.go
    - corpus/behavioral/go-explore-multi.json

key-decisions:
  - "go-node-multi.json did NOT change on regen — its capture does not embed the reworded comment bytes (review A6's conditional target correctly did not fire). Confirmed by git diff --name-only showing only go-explore-multi.json changed."
  - "CASES.json and the golden scenario count (30/30: 26 goldens + 4 CASES) are unchanged — verified via TestGoldenScenarioCountIsExact, not merely a green go test."

patterns-established: []

requirements-completed: [CODE-01]

coverage:
  - id: D1
    description: "Four corpus/behavioral src comments reworded from 'synthetic-parity corpus' to 'behavioral corpus' — comment-only edits, case-caption labels (a)/(b)/(c) and all Go code untouched"
    requirement: CODE-01
    verification:
      - kind: unit
        ref: "rg -c 'synthetic-parity corpus' corpus/behavioral/src/ -> 0 matches (no output)"
        status: pass
      - kind: unit
        ref: "rg -c 'behavioral corpus' across the 4 target files -> 4 matches total"
        status: pass
      - kind: unit
        ref: "git diff --stat corpus/behavioral/src/ -> 4 files, 1 line changed each"
        status: pass
    human_judgment: false
  - id: D2
    description: "go-explore-multi.json re-frozen via task golden:regen; diff is exactly the two reworded comment byte-runs; go-node-multi.json and CASES.json unchanged; golden suite green with scenario count intact"
    requirement: CODE-01
    verification:
      - kind: unit
        ref: "git diff --name-only after task golden:regen -> only corpus/behavioral/go-explore-multi.json (go-node-multi.json, CASES.json absent)"
        status: pass
      - kind: unit
        ref: "rg -l 'synthetic-parity corpus' corpus/behavioral/*.json -> 0 matches (no output)"
        status: pass
      - kind: unit
        ref: "go test -count=1 ./testdata/golden/... (exit 0, 29.2s)"
        status: pass
      - kind: unit
        ref: "task test:golden (exit 0)"
        status: pass
      - kind: unit
        ref: "go test -count=1 -run TestGoldenScenarioCountIsExact -v -> 'goldens: 26/26, CASES cases: 4/4, total: 30/30'"
        status: pass
    human_judgment: false

duration: ~10min
completed: 2026-08-15
status: complete
---

# Phase 5 Plan 6: Corpus Comment Reword & Attributed Golden Re-freeze Summary

**Reworded the four `corpus/behavioral/src` "synthetic-parity corpus" doc comments to "behavioral corpus" and re-froze the one golden transcript that embeds those bytes, with the diff asserted (not eyeballed) to be exactly the expected narrow set.**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-08-15 (session start, worktree branch `worktree-agent-ab78d0b790eaea184`)
- **Completed:** 2026-08-15
- **Tasks:** 2/2 completed
- **Files modified:** 5 (4 corpus src comments, 1 golden transcript)

## Accomplishments

- Reworded the retired "synthetic-parity corpus" name to "behavioral corpus" in the doc comments of all four target files (`accounts/manager.go` case (b), `accounts/validate.go` case (a), `orders/validate.go` case (a), `recovery/recovery_test.go` case (c)) — exactly one comment-text edit per file, verified via `rg -c` counts (0 old-name hits, 4 new-name hits) and `git diff --stat` confirming 1 line changed per file with no code touched.
- Ran `task golden:regen` against the locked corpora (output kept visible, not redirected) and confirmed via `git diff --name-only` that the ONLY file that changed is `corpus/behavioral/go-explore-multi.json` — `go-node-multi.json` did not change (its capture does not embed the reworded comment bytes; review A6's conditional target correctly did not fire) and `corpus/behavioral/CASES.json` was untouched.
- Inspected the full `git diff` of `go-explore-multi.json`: the only content change is the two "synthetic-parity corpus" → "behavioral corpus" byte-runs (accounts/manager.go's embedded case (b) comment and accounts/validate.go's embedded case (a) comment) — no other JSON content changed.
- `go test -count=1 ./testdata/golden/...` and `task test:golden` both green; `TestGoldenScenarioCountIsExact` explicitly confirmed the scenario count is unchanged (26/26 goldens, 4/4 CASES cases, 30/30 total).

## Task Commits

Each task was committed atomically:

1. **Task 1: Reword the four corpus/behavioral src comments from synthetic-parity to behavioral** - `53dba85` (docs)
2. **Task 2: Re-freeze the behavioral transcripts via task golden:regen and assert the narrow diff (review A6)** - `dab1b7a` (test)

**Plan metadata:** this commit (docs: complete plan)

## Files Created/Modified

- `corpus/behavioral/src/accounts/manager.go` - comment reword only (case (b) doc comment)
- `corpus/behavioral/src/accounts/validate.go` - comment reword only (case (a) doc comment)
- `corpus/behavioral/src/orders/validate.go` - comment reword only (case (a) doc comment)
- `corpus/behavioral/src/recovery/recovery_test.go` - comment reword only (case (c) doc comment)
- `corpus/behavioral/go-explore-multi.json` - re-frozen; embeds the manager.go and accounts/validate.go reworded comment bytes

## Decisions Made

- **`go-node-multi.json` was NOT a required re-freeze target and did not change.** The plan's review-A6-derived scope named it a conditional target only if its capture embeds the reworded bytes. `git diff --name-only` after `task golden:regen` confirmed it is byte-identical to before — the node capture does not embed the `corpus/behavioral/src` comment text. No action taken beyond confirming this via the diff.
- **The narrow-diff assertion was run as an explicit `rg -v` exclusion check** (`git diff --name-only | rg -v -e "<the 4 src files>" -e "<the 2 json files>"` returning empty) rather than an eyeballed count, per review A6's "assert, don't print" directive.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' `<automated>` verify blocks (as specified in the plan text) ran successfully; no shell-quoting defects were encountered in this plan's verify blocks (unlike the never-green gates found and fixed in 05-03/05-04 during convergence review).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The corpus/behavioral src comments and their goldens are now consistent on the "behavioral corpus" vocabulary — no stale frozen oracle remains that would go red on the next capture.
- No re-baseline was taken: only the two attributable byte-runs changed in `go-explore-multi.json`, and every other golden transcript (including `go-node-multi.json` and `CASES.json`) is byte-identical to before this plan.
- This plan did not touch `internal/query` or the capability matrix (05-04's scope) and did not touch `testdata/golden/README.md` (05-01's scope, already merged).

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-15*
