---
phase: 04-codegraph-upgrade-homebrew
plan: 03
subsystem: docs
tags: [roadmap, requirements, project-md, homebrew, cask, amendment]

# Dependency graph
requires:
  - phase: 03-homebrew-tap-cask
    provides: measured Caskroom paths (03-02-SUMMARY.md, 03-EVIDENCE.md) that this plan cites
provides:
  - ROADMAP.md Phase-4 criteria 1-3, Goal, Depends-on, checkbox item, and Phase-3 Notes
    naming the mechanism actually being built (Caskroom, seam-based proof, --check step-aside,
    linuxbrew dual-shape coverage)
  - REQUIREMENTS.md UPGR-01/UPGR-02 amended with dated notes and falsification citations
  - PROJECT.md scope bullet and Key Decisions row matching the amended requirements
affects: [04-01, 04-02, 04-04, 04-05, 04-06]

actuals:
  tokens: 6188
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: [scoped Edit-only amendments to tool-owned planning artifacts, dated amendment-note convention]

key-files:
  created: []
  modified:
    - .planning/ROADMAP.md
    - .planning/REQUIREMENTS.md
    - .planning/PROJECT.md

key-decisions:
  - "Followed D-01/D-04R/D-08/D-09 exactly as scoped in 04-CONTEXT.md — no new architectural decisions made in this plan"
  - "STATE.md's stale mirrored decision line at :191 deliberately left unedited — tool-owned generated artifact, corrected via PROJECT.md as the authoritative source"

patterns-established:
  - "Amendment landed at every place a claim appears (criteria, goal, depends-on, checkbox item, milestone-ordering bullet, Notes) per the Phase-3 UF-1 lesson, not just the normative criteria block"

requirements-completed: []

coverage: []

duration: ~20min
completed: 2026-08-11
status: complete
---

# Phase 4 Plan 3: ROADMAP/REQUIREMENTS/PROJECT.md Amendment Summary

**Amended every normative Phase-4 scope statement across ROADMAP.md, REQUIREMENTS.md, and PROJECT.md from `Cellar` to `Caskroom`-aware wording, replaced the sha256/mtime proof with a seam-based assertion, corrected `--check`'s promised behavior, and replaced the falsified "linuxbrew has no casks" premise with PR #19121 / Homebrew 6.0.0 citations.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-11T17:05:34Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- ROADMAP.md: Success Criteria 1-3 amended with dated notes (D-01, D-04R, D-08, D-09) — criterion 1 now proves the refusal via the `Options.download`/`Options.swap` seam instead of a sha256/mtime comparison; criterion 2 no longer promises an available-version report and matches UPGR-03's actual wording; criterion 3's linuxbrew leg now covers both the Caskroom and Cellar shapes, with the falsified "unreachable by construction" claim replaced by PR #19121 and Homebrew 6.0.0 citations
- ROADMAP.md: Goal, Depends-on, the Phase-4 checkbox list item, the milestone-ordering bullet, and Phase 3's Notes closing clause all restated to name Caskroom — closing the exact "amendment landed only on criteria" gap Phase 3's UF-1 flagged
- REQUIREMENTS.md: UPGR-01 and UPGR-02 each carry a dated 2026-08-11 amendment note with falsification citations; UPGR-03 left byte-unchanged (it was already correct and is what ROADMAP criterion 2 was amended toward)
- PROJECT.md: scope bullet and Key Decisions row amended to match, with the D-03 install-receipt requirement added to the mechanism description
- Milestone scope re-proved intact after every edit: `getMilestonePhaseFilter` still lists all four Phase-4-and-earlier directories, including `04-codegraph-upgrade-homebrew`

## Task Commits

Each task was committed atomically:

1. **Task 1: Amend the ROADMAP — criteria 1-3, the goal, the depends-on, the checkbox item, and the two prose claims outside the criteria block** - `acfa7ab` (docs)
2. **Task 2: Amend REQUIREMENTS UPGR-01/UPGR-02 and PROJECT.md's Key Decisions, and record the tool-owned STATE.md mirror** - `fafc6b9` (docs)

**Plan metadata:** pending (this SUMMARY's own commit)

## Files Created/Modified

- `.planning/ROADMAP.md` - Phase-4 Success Criteria 1-3, Goal, Depends-on, checkbox item, milestone-ordering bullet, Phase-3 Notes closing clause
- `.planning/REQUIREMENTS.md` - UPGR-01, UPGR-02 (UPGR-03 untouched)
- `.planning/PROJECT.md` - scope bullet, Key Decisions row

## Decisions Made

None beyond what 04-CONTEXT.md already locked (D-01, D-02 [amendment-note wording only], D-03 [cited in PROJECT.md], D-04R, D-08, D-09). This plan is pure documentation-amendment work executing decisions made during discuss-phase/research; no new architectural or scoping decisions were made here.

**One scope call worth recording:** this plan's own frontmatter lists `requirements: [UPGR-01, UPGR-02, UPGR-03]` (inherited from the phase-level requirement list), but this plan only corrects the *prose* describing those requirements — it does not implement detection logic (that is 04-01) and does not run the real-tap acceptance (that is 04-06). `requirements-completed` is therefore left empty in this SUMMARY's frontmatter and no REQUIREMENTS.md checkboxes were checked; UPGR-01/02/03 remain `[ ]` pending, to be marked complete by the plan(s) that actually close them.

## Deviations from Plan

None - plan executed exactly as written. All seven scoped Task-1 edits and both Task-2 edits landed as specified; every acceptance criterion in both tasks was verified via the exact `rg`/`git diff`/`node` commands the plan specified before committing.

## Verification Evidence

Ran every acceptance-criteria command from both tasks plus the plan-level `<verification>` block after each commit:

- `rg --no-heading -n -e 'Cellar' .planning/ROADMAP.md | rg -v -e 'Caskroom' | wc -l` → `0`; unfiltered count → `7` (≥4 required)
- `rg -n -e '^- \[.\] \*\*Phase 4:' .planning/ROADMAP.md | rg -c -e 'Caskroom'` → `1`
- `rg -c -e 'Options\.download' .planning/ROADMAP.md` → `2`; `rg -c -e 'Options\.swap'` → `2`
- `rg -c -e 'sha256 and mtime'` within the Phase-4 block (lines 209-242) → `0`
- `rg -n -e 'reports the available version'` within the Phase-4 block → no match; `rg -c -e 'brew outdated'` within block → `1`
- Criterion-3 line contains `linuxbrew` and `#19121`; `rg -c -e 'unreachable'` within the Phase-4 block → `0`
- `rg -n -e 'most robust signal Phase 4 can key on' .planning/ROADMAP.md` → no match (after a mid-task fix — see below); `rg -c -e 'homebrew_casks:. is the block' -e 'Homebrew/brew #20291'` → `1`
- `node -e '...getMilestonePhaseFilter...'` → lists all 4 phase directories including `04-codegraph-upgrade-homebrew`, both after Task 1 and again after Task 2
- `rg --no-heading -n -e 'Cellar' .planning/REQUIREMENTS.md .planning/PROJECT.md | rg -v -e 'Caskroom' | wc -l` → `0`; unfiltered count → `4` (≥3 required)
- `rg -n -e '^- \[ \] \*\*UPGR-0[12]\*\*' -A 0 .planning/REQUIREMENTS.md` → 2 lines, each containing `Amended 2026-08-1`
- `git diff .planning/REQUIREMENTS.md` shows no hunk touching the UPGR-03 line
- `rg -c -e '#19121' .planning/REQUIREMENTS.md` → `1`; `rg -c -e 'INSTALL_RECEIPT' .planning/PROJECT.md` → `1`
- `rg -c -e 'path-prefix guess' .planning/PROJECT.md` → `2`
- `git diff --exit-code .planning/STATE.md` → exits 0 (untouched)
- `rg -c -e 'Traceability' .planning/REQUIREMENTS.md` → `1`; UPGR traceability rows unchanged (confirmed via diff — no `+`/`-` on those lines)
- `git diff --exit-code go.mod go.sum` → exits 0
- Post-commit deletion check on both commits: `git diff --diff-filter=D --name-only HEAD~1 HEAD` → empty (no files deleted)

**Note on the Phase-3 Notes edit:** the first draft of the amendment note quoted the literal phrase "most robust signal Phase 4 can key on" inside a paraphrase, which still matched the acceptance-criteria grep even though it was inside a quoted retelling. Caught by re-running the exact `rg` check before committing and rewritten to paraphrase without reproducing the literal substring — no commit was made with the failing state; this was corrected before Task 1's commit, not a Rule 1 post-commit fix.

## Issues Encountered

None beyond the self-caught wording issue above (corrected pre-commit).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP/REQUIREMENTS/PROJECT.md now describe the mechanism 04-01 already implemented (tracer landed) and that 04-02, 04-04, 04-05, 04-06 will build on — no plan downstream in this phase needs to re-derive or guess at Caskroom-vs-Cellar wording
- `.planning/STATE.md:191`'s stale mirror is an explicitly recorded, deliberate exclusion (tool-owned generated artifact); the next tool-driven state write will carry PROJECT.md's corrected text forward
- No blockers for 04-04/04-05/04-06

---
*Phase: 04-codegraph-upgrade-homebrew*
*Completed: 2026-08-11*
