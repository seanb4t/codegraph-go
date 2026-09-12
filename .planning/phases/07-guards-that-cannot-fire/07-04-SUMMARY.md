---
phase: 07-guards-that-cannot-fire
plan: 04
subsystem: testing
tags: [go, github-actions, yaml, workflow-shape-test, mutation-log, guard-hardening]

# Dependency graph
requires:
  - phase: 07-guards-that-cannot-fire (plan 01, plan 02, plan 03)
    provides: "07-MUTATION-LOG.md header, cleanliness-gate convention, and families (a)/(b)/(c), ready for this plan to append family (d) and close the log"
provides:
  - "internal/upgrade/release_workflow_shape_test.go: fullWorkflowJob.If field, postReleaseConclusionGuard const, TestPostReleaseJobsDeclareConclusionGuard and its _EmptyDocIsError companion, asserting every post-release-verify.yml job carries the event-aware conclusion guard verbatim with no fixed job-id list and no normaliser"
  - "TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets deleted (D-09): it compared two in-test constant lists and read no workflow, so it could not fail regardless of the real workflows"
  - "07-MUTATION-LOG.md closed: family (d) (two RED transcripts, removed and inverted), the GRD-05 one-line deletion record, and a closing non-vacuity assertion"
  - "All four todos folded into this phase resolved under .planning/todos/completed/"
affects: [07-guards-that-cannot-fire]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 4466
  tasks: 3
  commits: 3
  plan_head_before: 3398c092d2256f4c036c8319918428510e18f4bf

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sibling test function reusing an existing decodeFullWorkflowDoc/postReleaseWorkflowPath parse rather than extending the neighboring test, so a -run filter can exercise one property in isolation during a RED demonstration"
    - "Guard assertion compares a parsed field to one verbatim package-level const with plain string equality — no fixed expected-id list, no normalising helper — so a newly added or altered entry fails on its own rather than because a list went stale"
    - "Deletion over rewrite for a tautological guard: when a test cannot fail and the property it names is judged low-value, remove it outright and record the decision as one line in the mutation log rather than spend the phase on a file-reading rewrite"

key-files:
  created: []
  modified:
    - internal/upgrade/release_workflow_shape_test.go
    - .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md
    - .planning/todos/completed/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md (moved from .planning/todos/pending/)
    - .planning/todos/completed/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md (moved from .planning/todos/pending/)

key-decisions:
  - "Followed D-08 exactly: TestPostReleaseJobsDeclareConclusionGuard is a sibling of TestPostReleaseJobsDeclareCheckoutPolicy, not an extension — placed immediately after it, reusing decodeFullWorkflowDoc and the existing postReleaseWorkflowPath const"
  - "postReleaseConclusionGuard's value was copied byte-for-byte from post-release-verify.yml (confirmed via rg against all five job if: lines before writing the const) rather than retyped, so single quotes and spacing are guaranteed to match the real file rather than a memorized approximation"
  - "Followed D-09 exactly: TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets deleted in full (function, doc comment, and its local releasePleaseAppSecretNames slice) with no stub, skip, or file-reading replacement left behind; homebrewTapCredentialNames kept untouched since six other references in the file depend on it"
  - "Task 1 committed as a single test(07-04) commit rather than a test/feat split: the struct field, const, and both test functions all live in the test file itself with no separate production implementation to add after a RED phase, matching the precedent set by 07-02's archtest task — the real RED demonstrations (guard removed, guard inverted) are Task 3's mutations against the real workflow file, not a compile-time RED against Task 1's own additions"
  - "d2's mutation inverted the conclusion comparison (== to !=) on resolve-tag rather than mutating a different job or a different half of the disjunct, keeping the event-name half and every other job byte-identical — this is what makes the mutation 'still plausible-looking' per the plan's own framing, and is what proves the assertion discriminates on content rather than mere presence of an if: line"

requirements-completed: [GRD-04, GRD-06]

coverage:
  - id: D1
    description: "A test parses post-release-verify.yml's job map, reports how many jobs it inspected (must be > 0), and fails when any single job's if: is not exactly the event-aware conclusion disjunct"
    requirement: "GRD-04"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestPostReleaseJobsDeclareConclusionGuard"
        status: pass
    human_judgment: false
  - id: D2
    description: "The assertion compares each job's parsed if: value against one verbatim expected string with no normaliser and no fixed expected-job-id list, proven by two RED demonstrations that discriminate on content: the guard removed from gatekeeper (fails naming gatekeeper with an empty value) and the guard inverted on resolve-tag (fails naming resolve-tag, quoting both the found and wanted values)"
    requirement: "GRD-04"
    verification:
      - kind: unit
        ref: "07-MUTATION-LOG.md family (d), demonstrations d1 and d2 (RED transcripts against internal/upgrade/release_workflow_shape_test.go#TestPostReleaseJobsDeclareConclusionGuard)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An empty-or-unparseable-document-is-error companion exists, mirroring TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError, so a zero-job parse can never read as a pass"
    requirement: "GRD-04"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError"
        status: pass
    human_judgment: false
  - id: D4
    description: "The tautological tap App secret-distinctness test is deleted (not rewritten, skipped, or stubbed), and homebrewTapCredentialNames survives because other tests in the file consume it"
    requirement: "GRD-04"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go — TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets absent (rg count 0), homebrewTapCredentialNames present 6 times, full package green"
        status: pass
    human_judgment: false
  - id: D5
    description: "07-MUTATION-LOG.md carries exactly four demonstration families, each with pasted verbatim failing output and a byte-clean revert, plus the GRD-05 one-line deletion record and a closing section asserting the non-vacuity property — the log is complete"
    requirement: "GRD-06"
    verification:
      - kind: other
        ref: "07-MUTATION-LOG.md — rg -c '^## Family' == 4; rg -c -- '--- FAIL: TestPostReleaseJobsDeclareConclusionGuard' == 2; git diff --quiet -- .github/workflows/post-release-verify.yml at plan close"
        status: pass
    human_judgment: false
  - id: D6
    description: "All four todos folded into this phase are resolved under .planning/todos/completed/, and .planning/todos/pending/ retains only the brew-trust todo Phase 12 owns"
    requirement: "GRD-06"
    verification:
      - kind: other
        ref: "git ls-files -- .planning/todos/pending/ (1 entry, the brew-trust todo); git ls-files -- .planning/todos/completed/2026-08-09-... and .../2026-08-10-... (both tracked)"
        status: pass
    human_judgment: false

# Metrics
duration: 15min
completed: 2026-09-09
status: complete
---

# Phase 7 Plan 4: post-release-verify conclusion guard + tap-test deletion Summary

**Gave `post-release-verify.yml`'s event-aware conclusion guard a real test — proven RED both when the guard is removed and when it's inverted — deleted the tautological tap App secret-distinctness test outright, and closed the phase's mutation log with its fourth family and the GRD-05 deletion record.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-09T00:36:00Z (approx.)
- **Completed:** 2026-09-09T00:51:25Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- `fullWorkflowJob` gained an `If` field (`yaml:"if"`) and a new package-level const `postReleaseConclusionGuard` holds the exact disjunct copied byte-for-byte from the workflow. `TestPostReleaseJobsDeclareConclusionGuard` compares every job's parsed `if:` against that constant with plain equality — no fixed job-id list, no normaliser — and logs the inspected count (5, must be > 0). `TestPostReleaseJobsDeclareConclusionGuard_EmptyDocIsError` mirrors the Apple-secrets companion exactly.
- `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` deleted in full (function, doc comment, local secret-name slice) per D-09 — it compared two constants declared inside the test file and read no workflow, so it could only fail if the test file itself were edited. `homebrewTapCredentialNames` survives untouched; six other references in the file still consume it.
- Two RED demonstrations proved the new guard discriminates on content, not presence: the `if:` line stripped entirely from `gatekeeper` (fails naming `gatekeeper` with an empty value), and the conclusion comparison inverted (`==` to `!=`) on `resolve-tag` while leaving the event-name half and every other job untouched (fails naming `resolve-tag`, quoting both the found and wanted values). Both mutations reverted byte-clean, confirmed via `git diff --quiet` before and after each.
- `07-MUTATION-LOG.md` closed: family (d) with both pasted verbatim transcripts, the GRD-05 one-line deletion record (not a demonstration), and a closing section asserting the phase-wide non-vacuity property — four families, all pasted verbatim, all byte-clean reverts.
- All four todos folded into this phase now resolved: `.planning/todos/pending/` retains only the brew-trust todo Phase 12 owns.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the parsed if: field, the verbatim guard constant, and the per-job conclusion-guard test** - `e3a99fb8` (test)
2. **Task 2: Delete the tautological tap App secret-distinctness test** - `3dbdcc33` (fix)
3. **Task 3: RED demonstrations, mutation-log family (d), the GRD-05 record, and the phase-wide log check** - `abf6c899` (docs)

**Plan metadata:** commit created by this SUMMARY's own atomic write+commit step.

_Note: Task 1 is `tdd="true"` but produced a single `test(07-04)` commit rather than a test/feat split — the struct field, const, and both test functions all live in the test file with no separate production implementation to write after RED; the actual RED demonstrations against real production content (the workflow file) are Task 3's mutations, matching the precedent 07-02's archtest task set._

## Files Created/Modified
- `internal/upgrade/release_workflow_shape_test.go` - Added `fullWorkflowJob.If`, `postReleaseConclusionGuard`, `TestPostReleaseJobsDeclareConclusionGuard` (+ empty-doc companion); deleted `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` - Appended family (d), the GRD-05 record, and the closing non-vacuity section; log is now complete
- `.planning/todos/completed/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md` - Moved from pending, resolved
- `.planning/todos/completed/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md` - Moved from pending, resolved by deletion

## Decisions Made
See `key-decisions` in frontmatter — all five decisions followed the plan's D-08/D-09 discretion notes exactly; no departures required a judgment call beyond what CONTEXT.md already resolved.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. All acceptance criteria and `<verify>` commands passed on first attempt for every task.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `07-MUTATION-LOG.md` is complete: four families (a-d) plus the GRD-05 record and closing assertion. No further appends expected in this milestone.
- GRD-04 marked complete via `requirements.mark-complete`. GRD-06 was shared across 07-01 and this plan (shared-ID gate, #2388) — both plans now have SUMMARY.md, so GRD-06 becomes ready and is marked complete by this plan's `update_requirements` step.
- `.planning/todos/pending/` retains exactly one tracked file, the brew-trust todo ROADMAP routes to Phase 12 — no other Phase 7 todos remain open.
- Phase 7 (Guards That Cannot Fire) is now fully executed: all four plans (07-01 through 07-04) have produced summaries.

---
*Phase: 07-guards-that-cannot-fire*
*Completed: 2026-09-09*

## Self-Check: PASSED

All modified files found on disk. All three task commits (e3a99fb8, 3dbdcc33, abf6c899) found in git log. Plan produced 3 commits against `plan_head_before` 3398c092 (measured via `git rev-list --count`). `git diff --quiet -- .github/workflows/post-release-verify.yml` confirmed clean at close. Full verification suite (`GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/ ./internal/bench/ ./internal/query/archtest/`) green.
