---
phase: 05-process-ci-in-tree-sweep
plan: 02
subsystem: process
tags: [github-templates, issue-forms, pr-templates, contributor-docs, pr-template-policy]

requires:
  - phase: 05-process-ci-in-tree-sweep (05-01, parallel wave)
    provides: "codegraph migrate removal (CODE-03) — not a hard dependency of this plan; this plan touches no Go source"
provides:
  - "5 issue templates and 4 PR templates with no comparison framing"
  - "bug_report.yml reclassified swept (review H1) instead of the original census's mistaken clean label"
  - "grep-asserted (non-tautological) confirmation that scripts/pr_template_policy.py's TEMPLATE_SIGNALS survive the ## Parity section removal"
affects: [05-03, 05-04, 05-05, 05-06]

actuals:
  tokens: 1490
  tasks: 3
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Grep-assert action=pass from $GITHUB_OUTPUT rather than trusting the policy script's exit code (the script documents exit-always-0 by design)"

key-files:
  created: []
  modified:
    - .github/ISSUE_TEMPLATE/bug_report.yml
    - .github/ISSUE_TEMPLATE/feature_request.yml
    - .github/ISSUE_TEMPLATE/enhancement.yml
    - .github/ISSUE_TEMPLATE/chore.yml
    - .github/pull_request_template.md
    - .github/PULL_REQUEST_TEMPLATE/feature.md
    - .github/PULL_REQUEST_TEMPLATE/enhancement.md

key-decisions:
  - "bug_report.yml install-method option 'Migrated from TypeScript CodeGraph' removed entirely rather than reworded — the existing 'Upgraded via `codegraph upgrade`' option already covers the own-terms upgrade path, so no replacement option was needed."
  - "feature_request.yml id:parity block replaced (not deleted) with id:existing-coverage, asking whether the behavior is already covered by an existing command and whether it changes an agent-consumed output shape — keeps the real guidance, drops the origin-project comparison."
  - "Both PR-template ## Parity sections were deleted outright rather than replaced with a '### Compatibility with existing usage' subsection, because each file already has an adjacent ## Compatibility / ## Behavior change section covering the same agent-output-shape substance — an added subsection would have duplicated it."
  - "One incidental fix outside the plan's described edits: feature_request.yml's unrelated sentence 'The most useful reports here start with...' was reworded to 'submissions' — the plan's own verify regex (bare '-e ports') matched the substring inside 'reports', a false positive against unrelated pre-existing prose, not comparison framing (Rule 3 — blocking-issue fix, see Deviations)."

requirements-completed: [PROC-01, PROC-02]

coverage:
  - id: D1
    description: "5 issue templates carry no comparison framing; bug_report.yml correctly reclassified as swept (review H1) and edited"
    requirement: "PROC-01"
    verification:
      - kind: other
        ref: "rg -c -e parity -e ports -e 'TypeScript CodeGraph' -e 'Behavioral Parity' .github/ISSUE_TEMPLATE/ == 0"
        status: pass
      - kind: other
        ref: "rg -c 'Migrated from TypeScript' .github/ISSUE_TEMPLATE/bug_report.yml == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "PR template + 3 variants carry no comparison framing; fix.md remains byte-identical; format gate (pr_template_policy.py) still keys on surviving headings and returns a captured action=pass verdict"
    requirement: "PROC-02"
    verification:
      - kind: other
        ref: "rg -n 'ports observable|parity decisions|TypeScript CodeGraph' across all 4 PR templates == 0"
        status: pass
      - kind: other
        ref: "rg -c '## Parity' feature.md enhancement.md == 0; git diff --name-only fix.md == empty"
        status: pass
      - kind: other
        ref: "GITHUB_OUTPUT-captured grep -q 'action=pass' for pull_request_template.md, feature.md, enhancement.md (PR_BODY/AUTHOR_ASSOCIATION=OWNER/CHANGED_FILES=docs/RELEASE.md env-driven, no exit-code reliance)"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-08-15
status: complete
---

# Phase 5 Plan 2: Issue & PR Template Sweep Summary

**Swept comparison framing from all 5 GitHub issue templates and all 4 PR templates (including the review-flagged `bug_report.yml` migrate-origin option), verified the pr-template-format policy gate still keys on surviving headings via a captured (non-tautological) `action=pass` verdict.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-15
- **Completed:** 2026-08-15
- **Tasks:** 3 (2 edit tasks + 1 verify-only census task)
- **Files modified:** 7

## Accomplishments
- All 5 issue templates (`bug_report`, `feature_request`, `enhancement`, `chore`, `config`) now carry zero comparison-framing terms (`parity`, `ports`, `TypeScript CodeGraph`, `Behavioral Parity`) — verified via `rg -c` summed to 0 across `.github/ISSUE_TEMPLATE/`.
- `bug_report.yml` correctly re-classified from the original research census's mistaken "clean" label to **swept** (review H1): the `Migrated from TypeScript CodeGraph` install-method dropdown option was removed.
- All 4 PR templates (default, `feature.md`, `enhancement.md`, `fix.md`) carry zero `ports observable|parity decisions|TypeScript CodeGraph` hits; the two `## Parity` sections in `feature.md`/`enhancement.md` are gone; `fix.md` remains byte-identical.
- Re-verified that `scripts/pr_template_policy.py`'s `TEMPLATE_SIGNALS` (`## What changed` / `## What this adds` / `## The bug` / `## What was wrong with it` / `## Required checklist` / `## Checklist`) do not key on `## Parity`, and all surviving headings are intact in every PR template after the edit.
- Grep-asserted (not exit-code-asserted, per cycle-3 review's tautology fix) that `pr_template_policy.py` returns `action=pass` via `$GITHUB_OUTPUT` for all three edited templates, both immediately after the edits and again in the task-3 phase-final cross-check.
- Census-clean surfaces `config.yml` and `PULL_REQUEST_TEMPLATE/fix.md` confirmed byte-identical (`git diff --name-only` empty for both).

## Task Commits

Each task was committed atomically:

1. **Task 1: Issue templates — bug_report migrate-option swept (H1), feature_request parity block removed, enhancement compat reworded, chore option updated** — `2089dfd` (feat)
2. **Task 2: PR templates — default-template ports paragraph rewritten, the two ## Parity sections removed, fix.md untouched** — `ff08add` (feat)
3. **Task 3: Verify-only census** — no commit (zero file edits; `config.yml`/`fix.md` confirmed byte-identical, policy verdict re-asserted)

## Files Created/Modified
- `.github/ISSUE_TEMPLATE/bug_report.yml` — SWEPT (review H1): removed the `Migrated from TypeScript CodeGraph` install-method option; the file is no longer census-clean
- `.github/ISSUE_TEMPLATE/feature_request.yml` — removed the `Migration from TypeScript CodeGraph` surface option; replaced `id: parity` with `id: existing-coverage` asking about existing command coverage and agent-consumed output shapes; reworded an unrelated `reports`→`submissions` false-positive substring hit
- `.github/ISSUE_TEMPLATE/enhancement.yml` — reworded the compat field to drop the "behavioral parity with TypeScript CodeGraph v1.3.1" baseline while keeping the "output shapes consumed by agents / documented rather than silent" substance
- `.github/ISSUE_TEMPLATE/chore.yml` — reworded the risk checkbox "The graph schema or migration path" → "The graph schema / meta-key layout" (the migrate capability name is gone)
- `.github/pull_request_template.md` — rewrote the "ports observable behavior from another implementation ... deliberate parity decisions" paragraph on codegraph-go's own terms; kept the issue-approval gate and the `.planning/` reference intact
- `.github/PULL_REQUEST_TEMPLATE/feature.md` — removed the `## Parity` section (its checklist duplicated the adjacent `## Compatibility` section's agent-output-shape substance)
- `.github/PULL_REQUEST_TEMPLATE/enhancement.md` — removed the `## Parity` section (its substance duplicated the earlier `## Behavior change` section)

## Files Verified Unchanged (census)
- `.github/ISSUE_TEMPLATE/config.yml` — **KEEP** (product surface): the "capability matrix" and SECURITY.md references are real product truth (the matrix embeds TypeScript-as-indexed-language rows, D-05), not comparison framing. Zero-byte diff confirmed.
- `.github/PULL_REQUEST_TEMPLATE/fix.md` — **CLEAN**: no parity-era prose exists in this template; it needed no edit. Zero-byte diff confirmed.
- `scripts/pr_template_policy.py` — **UNCHANGED, verify-only**: driven via its real env interface (`PR_BODY`/`AUTHOR_ASSOCIATION`/`CHANGED_FILES`/`GITHUB_OUTPUT`) to prove the format gate's verdict is `action=pass` for all three edited PR templates, both after task 2's edits and again as a task-3 phase-final cross-check. No comparison-framing was found in the script itself.

## Decisions Made
- **bug_report.yml**: removed the `Migrated from TypeScript CodeGraph` option outright rather than rewording it — the file already has `Upgraded via `codegraph upgrade`` as the own-terms upgrade-path option, so a reworded duplicate wasn't needed.
- **feature_request.yml `id: parity` block**: replaced (not deleted) with `id: existing-coverage`, preserving the real guidance (check existing commands first; note agent-consumed output-shape changes) while dropping the "does the original have this" framing.
- **PR template `## Parity` sections**: deleted outright in both `feature.md` and `enhancement.md` rather than replaced with a new `### Compatibility with existing usage` subsection — each file already carries an adjacent section (`## Compatibility` / `## Behavior change`) covering the same "agents consume this output" substance, so adding a new subsection would have been redundant per-file duplication.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reworded an unrelated word in feature_request.yml to unblock the task-1 verify gate**
- **Found during:** Task 1 (issue templates)
- **Issue:** Task 1's `<automated>` verify greps for the bare substring `ports` (no word boundary) across `.github/ISSUE_TEMPLATE/`. An unrelated, pre-existing sentence in `feature_request.yml`'s `problem` field ("The most useful reports here start with...") contains "reports", which matches the substring `ports` — a false positive against ordinary English prose that has nothing to do with comparison framing.
- **Fix:** Reworded "reports" → "submissions" in that one sentence; no meaning change.
- **Files modified:** `.github/ISSUE_TEMPLATE/feature_request.yml` (already in the task's own file set)
- **Verification:** Re-ran `rg -c -e parity -e ports -e "TypeScript CodeGraph" -e "Behavioral Parity" .github/ISSUE_TEMPLATE/` → summed to 0 after the fix.
- **Committed in:** `2089dfd` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking-gate false positive)
**Impact on plan:** No scope creep — the fix only touched a file already in task 1's edit set, and the change is a single-word substitution unrelated to comparison-framing content.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- PROC-01 and PROC-02 requirements are satisfied for this plan's surfaces; no blockers for 05-03 (workflow sweep), 05-04/05-05 (in-tree comment sweep), or 05-06 (corpus re-freeze).
- This plan is template/prose-only — it touched zero Go source files. `go build ./...` and `go vet ./...` were run locally against the working tree and passed with no output (no regressions introduced). The full `go test -count=1 ./...` suite is the orchestrator's authoritative post-merge wave gate and was not run to completion here per orchestrator guidance, since this plan owns no Go source changes.
- `bug_report.yml`'s correction (H1) means downstream census-based verification in other plans that may still reference the old "bug_report is clean" assumption should be re-checked against this SUMMARY, not the original RESEARCH.md census table.

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-15*
