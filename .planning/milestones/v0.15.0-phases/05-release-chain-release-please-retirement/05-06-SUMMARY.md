---
phase: 05-release-chain-release-please-retirement
plan: 06
subsystem: docs
tags: [changie, release-docs, contributing, runbook, census, github-app]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "release-pr.yml and release:batch (05-01), release-tag.yml and release:tag (05-02), release:goreleaser notes wiring and release:dry-run-signed description (05-03), the .changes/ hygiene exemption (05-04)"
provides:
  - "CONTRIBUTING.md Pull requests teaches fragments, the reasoned changelog-exempt marker, the derived version and the release PR"
  - "docs/RELEASE-PROCEDURES.md describes the changie chain end to end and says a release is cut by merging the release PR"
  - "05-CENSUS.md: one verdict row per base reference line, R01 to R45"
  - "pr-title.yml header says hygiene only; docs/RELEASE.md names release-pr.yml and release-tag.yml"
affects: [05-07, phase-06-live-run]

actuals:
  tokens: 12890
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "verdict table keyed by base commit line numbers, set-equal to the base grep, so no reference is dropped silently"

key-files:
  created:
    - .planning/phases/05-release-chain-release-please-retirement/05-CENSUS.md
  modified:
    - CONTRIBUTING.md
    - docs/RELEASE-PROCEDURES.md
    - docs/RELEASE.md
    - .github/workflows/pr-title.yml

key-decisions:
  - "Verdict counts for the 45 runbook references: 36 rewritten, 8 removed, 1 reworded-history"
  - "rc escape-hatch verdict: removed (D-15). The runbook says changie batch auto -p rc.N can batch a prerelease but is not wired into release:batch, and that release:goreleaser refuses a tag with no .changes/<tag>.md"
  - "Section 7's delete-and-retry procedure now governs a disposable cut (for example the live proof) instead of rc tags only, because no rc route exists any more; the patch-forward posture for stable releases is unchanged"
  - "Section 9 branch-protection note rewritten: the old text said the repository had no rulesets, which is stale (protect-main exists); it now says the chain never pushes to main"
  - "The App is called the release GitHub App throughout; no App setting and no repository setting was changed"

patterns-established:
  - "Doc rewrite verified by the same census pattern that found 45 hits at the base commit, now returning zero in all four docs"

requirements-completed: [DOCS-12, DOCS-13, DOCS-14]

coverage:
  - id: D1
    description: "CONTRIBUTING.md Pull requests teaches a fragment per user-visible change, the changelog-exempt marker, the derived version and the release PR; the check:cross bullet and the Never-do bullet match the chain"
    requirement: DOCS-12
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestContributingReferencesRealTaskTargets"
        status: pass
      - kind: other
        ref: "awk over the Pull requests section, rg -F for the seven required teachings"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/RELEASE-PROCEDURES.md describes fragment, release PR, merge, App-token tag, release.yml; pretag-gate placed in both new workflows; no census hit; no hand git tag instruction"
    requirement: DOCS-13
    verification:
      - kind: other
        ref: "rg census pattern over docs/RELEASE-PROCEDURES.md returns zero; the cut-by-merging sentence, both workflow names and pretag-gate present"
        status: pass
    human_judgment: false
  - id: D3
    description: "05-CENSUS.md verdict table is set-equal to the 45 base reference lines with the closed vocabulary"
    requirement: DOCS-13
    verification:
      - kind: other
        ref: "set-equality command from the plan's Task 2 verify block, run against 5a8603f5"
        status: pass
    human_judgment: false
  - id: D4
    description: "pr-title.yml header says hygiene only with comment lines changed only; docs/RELEASE.md names the changie workflows"
    requirement: DOCS-14
    verification:
      - kind: other
        ref: "git diff -U0 5a8603f5 -- pr-title.yml shows only comment lines; rg 'hygiene only'"
        status: pass
    human_judgment: false
  - id: D5
    description: "Readability and accuracy of the rewritten prose against the chain as built and as it will run live"
    verification: []
    human_judgment: true
    rationale: "No test asserts that prose is clear or that the designed chain behaves as described; the live behavior is proven in Phase 6"

duration: 6min
completed: 2026-10-14
status: complete
plan_head_before: 7d2566695f1a47be39087952223aedcfa403a8b1
plan_head_after: 01d32490030161f48ace1e34320071bf74af1c2e
---

# Phase 5 Plan 06: Contributor and Maintainer Docs for the Changie Chain Summary

**CONTRIBUTING.md, the release runbook, docs/RELEASE.md and the pr-title.yml header now describe the fragment, release PR, App-token tag, locked release.yml chain, and each of the runbook's 45 retired-tool references has a recorded verdict.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-10-14T22:04:05Z
- **Completed:** 2026-10-14T22:10:00Z
- **Tasks:** 2
- **Files modified:** 5 (4 edited, 1 created)

## Accomplishments

- CONTRIBUTING.md Pull requests: a fragment per user-visible change with `task changie -- new`, the non-empty-reason `changelog-exempt` marker, the version derived from fragment kinds (Breaking and Features bump the minor below 1.0, the rest bump the patch), the release PR from release-pr.yml, and never hand-writing `.changes/v*.md` or CHANGELOG.md. The PR title stays Conventional-Commits-shaped as hygiene only.
- docs/RELEASE-PROCEDURES.md: section 4 is now "Cutting a release (the changie release PR)" with the exact chain line, six numbered steps, the sentence "A release is cut by merging the release PR.", and the expected no-op note. Section 1 and 3 place pretag-gate in release-pr.yml and release-tag.yml. Section 7 is the changie rollback. Section 9 names the permissions the chain uses and the App-token rationale for both the tag and the release PR.
- The hand-pushed rc tag escape hatch is gone (D-15), with a verdict row, and the runbook points at `changie batch auto -p rc.N` as an unwired option.
- 05-CENSUS.md R01 to R45: set-equal to the 45 base lines, rows ordered by base line, seven cells per row.
- pr-title.yml header says hygiene only (comment lines only, title regex and run body byte-identical to 5a8603f5); docs/RELEASE.md says release-tag.yml pushes the tag the SAN names, after the release PR that release-pr.yml opens is merged.

## Task Commits

1. **Task 1: fragment workflow in CONTRIBUTING.md, pr-title.yml header, docs/RELEASE.md** - `204f5038` (docs)
2. **Task 2a: release runbook rewrite** - `8d2884d9` (docs)
3. **Task 2b: D-10 verdict table** - `01d32490` (docs)

**Plan metadata:** the docs(05-06) completion commit that holds this file with STATE.md, ROADMAP.md and REQUIREMENTS.md.

## Files Created/Modified

- `CONTRIBUTING.md` - Pull requests section, check:cross bullet, Never-do bullet
- `docs/RELEASE-PROCEDURES.md` - sections 1, 3, 4, 6 (one note), 7, 9, 11 (one table cell), 12 (tap App wording)
- `docs/RELEASE.md` - one paragraph naming the changie release workflows
- `.github/workflows/pr-title.yml` - header comment lines only
- `.planning/phases/05-release-chain-release-please-retirement/05-CENSUS.md` - D-10 verdict table (05-07 appends the census transcripts)

## Decisions Made

- **Verdicts:** 36 rewritten, 8 removed (R lines 76, 82, 169, 189, 323, 352, 355, 356 by base line), 1 reworded-history (base line 408, the Issues-scope history).
- **rc hatch:** removed, per D-15. A prerelease can be batched with `changie batch auto -p rc.N` but is not wired into `release:batch`; `release:goreleaser` refuses a tag without `.changes/<tag>.md`.
- **Section 7 scope:** broadened from "rc cuts only" to "a disposable cut" because the rc route no longer exists. A stable release is still patched forward, never rewritten.
- **Section 9 branch protection:** rewrote the stale "no rulesets today" note. It now describes protect-main and says the chain never pushes to `main` (the App pushes only the `changie/release` branch and the tag).
- **Section 9 live proof:** no longer describes a disposable rc tag on a scratch branch; says the chain's first real run is its live proof (Phase 6 does that run).
- **Issues write permission:** the doc says the chain does not use it and an App holding it from before may keep it. No App setting was changed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The plan's census-style verify pattern false-positives on existing text**
- **Found during:** Task 1 verify
- **Issue:** `rg -i 'release[-_ ]?please|release-as'` also matches `release-assets` in CONTRIBUTING.md's `task verify:release-assets` bullet, which is correct existing text naming a real target (the plan also requires `TestContributingReferencesRealTaskTargets` to pass).
- **Fix:** Used `\brelease-as\b` for the Release-As half of the check. The CONTRIBUTING text is unchanged. The first half (`release[-_ ]?please`) is unchanged.
- **Files modified:** none
- **Verification:** the corrected pattern returns zero hits across CONTRIBUTING.md, pr-title.yml, docs/RELEASE.md and RELEASE-PROCEDURES.md
- **Committed in:** n/a (verification command only)

### Pre-existing items left alone

- The Task 1 acceptance criterion `rg -P '[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]'` over whole files matches one character in docs/RELEASE.md line 132, the check mark in the pasted `cosign verify-blob` sample output ("Verification succeeded!"). It is verbatim tool output from before this plan, and this plan adds no emoji (checked on the added lines of every diff). Left unchanged.

---

**Total deviations:** 1 auto-fixed (1 blocking verify pattern), 1 pre-existing item noted
**Impact on plan:** None on the delivered docs.

## Issues Encountered

- The plan's `git-merge-guard` hook rejected a long heredoc-fed Python command on quoting; the edit script was written to the scratchpad with the Write tool and run from there. No effect on the output.
- `.git/gsd-plan-head-before-05-06` held a commit that is not an ancestor of HEAD, so it was reset to HEAD (7d256669) before the first commit, as the orchestrator directed.
- TDD resolver marked tdd.md applicable, but this plan defines no RED/GREEN check (docs only). The tdd-red-evidence check was not run (repo rule x1cjy9vyhq).

## Census State After This Plan

The four docs this plan touches (CONTRIBUTING.md, docs/RELEASE-PROCEDURES.md, docs/RELEASE.md, pr-title.yml) have zero hits for `release[-_ ]?please` in tracked files. The same pattern finds all 45 lines in the base copy of the runbook, which is the positive control for this pattern on this file. Tracked hits still outside the excluded paths, owned by other plans: release-please.yml, release-please-config.json, ci.yml (1), post-release-verify.yml (2), release_workflow_shape_test.go (1), tag_authority_test.go (15) and taskfile_shape_test.go (7). 05-05 and 05-07 own those; this plan does not touch them.

## User Setup Required

None - no external service configuration required. No App or repository setting was changed.

## Next Phase Readiness

- 05-07 can append the REL-14 census instrument, positive control and clean-tree census to 05-CENSUS.md.
- Live behavior the docs describe (the path filter on a squash merge, the App-token tag starting release.yml) is proven in Phase 6; the docs state the designed chain.

## Self-Check: PASSED

- Files found: CONTRIBUTING.md, docs/RELEASE-PROCEDURES.md, docs/RELEASE.md, .github/workflows/pr-title.yml, 05-CENSUS.md
- Commits found on HEAD: 204f5038, 8d2884d9, 01d32490
- No file deletions in the three commits
- Plan verification: TestContributingReferencesRealTaskTargets PASS, `go test ./internal/upgrade/` PASS, census set-equality check PASS (45 rows, closed vocabulary)

---
*Phase: 05-release-chain-release-please-retirement*
*Completed: 2026-10-14*
