---
phase: 05-process-ci-in-tree-sweep
plan: 03
subsystem: ci
tags: [github-actions, actionlint, workflow-sweep, proc-03]

requires:
  - phase: 05-process-ci-in-tree-sweep (05-01)
    provides: codegraph migrate removal (no shared diff surface with this plan)
provides:
  - Two inline-JS action-script strings (require-issue-link.yml, auto-close-unsolicited-prs.yml) reworded off "parity"/"ports observable behavior" framing
  - corpora.yml step name de-paritied, run: task test:golden byte-identical
  - bench.yml free head-to-head job/step display names renamed to own-terms (H2); comparison capability, run:/uses:/needs: bodies untouched
  - Whole-workflow-surface re-verification: actionlint 14/14 exit 0, TestWorkflowRequiredChecks + TestWorkflowRunBodiesInvokeTask green, verify-only workflows byte-untouched
affects: [phase-06-benchmark-decoupling]

actuals:
  tokens: 1350
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Job display `name:` renamed while job ID (`headtohead`) stays — required-check protection covers only the 7 ruleset-bound names, not every job in the repo"
    - "Term-by-term reword, never regex — each string was hand-verified against its file:line before editing"

key-files:
  created: []
  modified:
    - .github/workflows/require-issue-link.yml
    - .github/workflows/auto-close-unsolicited-prs.yml
    - .github/workflows/corpora.yml
    - .github/workflows/bench.yml

key-decisions:
  - "Left bench.yml :31 and :510 'historical reference'/'stale reference' comments untouched — verified these describe an in-repo prior baseline measurement (Phase-9/10-04-PLAN investigation), not TS comparison framing; the plan's line citations for these were stale (file has grown since the census was written)."
  - "Left the run: body echo text 'Head-to-head benchmark results (PERF-01)' (job summary heading, ~line 145) untouched — it lives inside a shell run: body, which the plan's binding constraint explicitly protects (TestWorkflowRunBodiesInvokeTask keys on run: bodies)."

patterns-established:
  - "Renaming a free (non-required-check) job/step display name is safe when the job ID and all if:/inputs references to that ID stay byte-identical."

requirements-completed: [PROC-03]

coverage:
  - id: D1
    description: "require-issue-link.yml and auto-close-unsolicited-prs.yml inline-JS messages reworded off parity framing; corpora.yml step renamed"
    requirement: PROC-03
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: other
        ref: "rg -n -e parity -e 'ports observable' -e 'another implementation' on the two JS files — zero hits"
        status: pass
      - kind: other
        ref: "node --check on both extracted with.script bodies — both parse"
        status: pass
    human_judgment: false
  - id: D2
    description: "bench.yml free head-to-head job/step display names renamed (H2); required-check names and capability steps untouched"
    requirement: PROC-03
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -run 'TestWorkflowRequiredChecks|TestWorkflowRunBodiesInvokeTask'"
        status: pass
      - kind: other
        ref: "actionlint .github/workflows/bench.yml — exit 0"
        status: pass
      - kind: other
        ref: "rg -n -e head-to-head -e 'vs TS' -e 'TS reference' -e 'reference binary' bench.yml — zero hits"
        status: pass
    human_judgment: false
  - id: D3
    description: "Whole-workflow-surface re-verification: actionlint 14/14, workflow guards green, verify-only workflows untouched"
    requirement: PROC-03
    verification:
      - kind: other
        ref: "actionlint .github/workflows/*.yml — exit 0 (14 files)"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... (full package)"
        status: pass
      - kind: other
        ref: "git diff --name-only on ci/release/linux-cross-canary/post-release-verify/release-please.yml — empty"
        status: pass
    human_judgment: true
    rationale: "The live-ruleset gh api check surfaced a pre-existing drift (see Deviations) that the plan instructs to report, not fix — a human should be aware of it even though it's out of this plan's edit scope."

duration: 25min
completed: 2026-08-15
status: complete
---

# Phase 5 Plan 3: Workflow Framing Sweep Summary

**Reworded two contributor-visible inline-JS strings and two workflow display-name families (corpora, bench head-to-head) off retired parity/comparison framing, with zero required-check job-name edits and zero shell run:-body changes.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-15T23:29:00Z
- **Completed:** 2026-08-15T23:54:00Z
- **Tasks:** 3 (2 edited files, 1 verify-only)
- **Files modified:** 4

## Accomplishments
- `require-issue-link.yml` and `auto-close-unsolicited-prs.yml` inline-JS PR-comment strings no longer say "parity decisions" or "ports observable behavior from another implementation" — both keep the issue-first / `.planning/` guidance, both re-verified with `node --check`.
- `corpora.yml`'s golden-suite step is now named "Run the golden suite (testdata/golden)"; `run: task test:golden` is byte-identical.
- `bench.yml`'s free (non-required-check) `headtohead` job display name and its "Run head-to-head benchmark" step name are renamed to own-terms wording (`benchmark publish (PERF-01, non-blocking)` / `Run benchmark`); the job ID, `if:` conditions, artifact names, and every `run:`/`uses:`/`needs:` body are untouched — the comparison capability itself stays for Phase 6 to decouple.
- All 14 workflows pass `actionlint`; `TestWorkflowRequiredChecks` and `TestWorkflowRunBodiesInvokeTask` are green; the five verify-only workflows (`ci`, `release`, `linux-cross-canary`, `post-release-verify`, `release-please`) have zero diff.

## Task Commits

Each task was committed atomically:

1. **Task 1: Corpora and the two inline-JS comment strings** - `781a55d` (docs)
2. **Task 2: bench.yml re-frame comments and rename free head-to-head names (H2)** - `9d4bb1e` (docs)
3. **Task 3: Re-verify the whole phase** - verify-only, no file changes, no commit (all five named files confirmed byte-untouched)

## Files Created/Modified
- `.github/workflows/require-issue-link.yml` - inline-JS message at :125 reworded (issue-first + prior context, drops "parity decisions")
- `.github/workflows/auto-close-unsolicited-prs.yml` - inline-JS message at :91 reworded (keeps `.planning/` + issue-first, drops "ports observable behavior... parity decisions")
- `.github/workflows/corpora.yml` - step name at :266 renamed ("Run the golden suite"); `run: task test:golden` unchanged
- `.github/workflows/bench.yml` - job display name (:96), step name (:127→126), two step labels (Node setup / install), and two header comments (:4, :45) reworded; job ID `headtohead`, all `if:`/`inputs` references, artifact names, and every `run:`/`uses:` body untouched

## Decisions Made
- Left `bench.yml`'s "historical reference"/"stale reference" comments (near what the plan cited as :31/:510) untouched after verifying in context that they describe an in-repo prior baseline measurement from the Phase-9/10-04-PLAN runner-class investigation, not TS comparison framing. The plan's line citations for these two spots were stale relative to the current file (bench.yml has grown with cpu-diag/scratch-fs-compare/disk-control jobs since the census was written); re-reading each in context confirmed neither is comparison framing under D-01.
- Left the job-summary echo text "## Head-to-head benchmark results (PERF-01)" inside the `Run benchmark` step's `run:` body untouched — it lives inside a shell `run:` body, which both the plan's binding constraint and `TestWorkflowRunBodiesInvokeTask` explicitly protect from edits in this phase.

## Deviations from Plan

### Discovered (not fixed, per plan's own instruction)

**1. Live-ruleset drift: required-check fixture has 7 entries, live ruleset has 6**
- **Found during:** Task 3 (whole-phase re-verification, step 5: `gh api repos/seanb4t/codegraph-go/rulesets/20157557`)
- **Issue:** `internal/upgrade/taskfile_shape_test.go`'s `requiredCheckNames` fixture lists 7 contexts (`test`, `govulncheck (DIST-03, blocking)`, `reproducibility (double-build hash-diff, DIST-04)`, `perf regression gate (PERF-02, INDX-06)`, `actionlint (workflow static analysis)`, `goreleaser check (config validation, DIST-01)`, `pr-title`). The live GitHub ruleset 20157557's `required_status_checks` array currently has only 6 — `goreleaser check (config validation, DIST-01)` is absent from the live-enforced set, even though the job itself exists byte-identical in `ci.yml:412` and matches the fixture string exactly.
- **Disposition:** Per this plan's own acceptance criterion ("if the live-ruleset gh api runs and differs from the fixture names, STOP and report (do not fix the fixture) — branch protection should be the authority"), this is reported, not fixed. It is unrelated to this plan's edits — zero required-check job names were touched by any task in this plan, and the drift predates this phase's execution (it is a live GitHub configuration state, not a repo-file state). Needs a maintainer decision: either add the missing context back to the live ruleset, or the fixture is stale and should be corrected in a separate, reviewed diff — neither action belongs in this framing-sweep plan.
- **Verified via:** `gh api repos/seanb4t/codegraph-go/rulesets/20157557 --jq '.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks'` returned 6 entries, missing the goreleaser-check context.

---

**Total deviations:** 1 discovered-and-reported (live infra drift, out of this plan's edit scope per its own acceptance criterion)
**Impact on plan:** None on this plan's scope. No fixture or ruleset edit was made; PROC-03's own success criteria (framing-free names/comments, byte-identical required-check names in the swept files, actionlint + guards green) are all satisfied.

## Issues Encountered
None beyond the discovered live-ruleset drift above.

## User Setup Required
None - no external service configuration required. (The live-ruleset drift above needs a maintainer/GitHub-admin decision, not a code or local-setup step — see Deviations.)

## Next Phase Readiness
- `bench.yml`'s comparison capability (the runner and installed reference binary) is fully intact for Phase 6 to decouple — only its framing (names/comments) changed here, exactly as scoped.
- The live-ruleset drift (goreleaser-check context missing from ruleset 20157557) should be surfaced to the maintainer before Phase 6 or any future plan that touches required-check names, so it isn't mistaken for something this phase caused.

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-15*
