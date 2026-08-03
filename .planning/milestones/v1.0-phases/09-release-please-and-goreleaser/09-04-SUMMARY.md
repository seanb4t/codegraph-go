---
phase: 09-release-please-and-goreleaser
plan: 04
subsystem: infra
tags: [docs, release-please, github-app, roadmap, runbook]

# Dependency graph
requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-01's release-please spine, 09-02's D-04 create-if-absent-else-upload-clobber publish step, and 09-03's D-08 PR-title lint (recorded as its own divergence) — all three needed to be true before the runbook describing them could be rewritten accurately"
provides:
  - "docs/RELEASE-PROCEDURES.md §3/§4/§7 rewritten for the release-please flow: fast-forward merge model with repo evidence, release-PR-merge trigger (not hand-run git tag), and rollback covering all three artifacts a cut leaves behind"
  - "docs/RELEASE-PROCEDURES.md §9 — the GitHub App prerequisite (D-02) end to end: creation, three installation permissions, the two secret names, the installation-scope-vs-workflow-permissions distinction, the deprecated app-id/client-id note, pre-flight commands, and the branch-protection bypass-actor requirement"
  - "docs/RELEASE-PROCEDURES.md §10 — three recorded divergences (roadmap criterion 3/D-05, D-08's file location, D-03's deliberate non-implementation with its full recipe)"
  - ".planning/ROADMAP.md Phase 9 success criterion 3 amended in place with a dated blockquote, Phase-8-amendment-style, recording D-05"
  - "the folded runbook todo marked resolved in place, pointing at the sections that closed it"
affects: [09-05, 09-06, 09-07, 09-08]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-task same-file split committed via targeted forward/reverse Edit application (add Task-2 content, verify Task-1-only acceptance criteria, commit; re-add Task-2 content, verify, commit) rather than a single combined commit — preserves per-task atomicity when both tasks touch the same file"

key-files:
  created: []
  modified:
    - docs/RELEASE-PROCEDURES.md
    - .planning/ROADMAP.md
    - .planning/todos/pending/2026-07-14-document-release-cut-procedures-runbook.md

key-decisions:
  - "§3's repo-evidence numbers (roughly 477 commits ahead / ~160 feat-fix-perf / ~217 docs-only) are cited as the figures D-09 recorded at decision time, not re-measured live at execution time (live count is now 493/165/228 as the branch continued moving) — the runbook cites the evidence that settled the fast-forward-vs-squash decision, not a live-updating counter"
  - "§9 documents the disposable live proof concept (a throwaway prerelease cycle proving the App-token wiring before the real v1.0.0 cut, per RESEARCH's Validation Architecture) in prose, since §7's rollback pointer and §6's mandatory-verification note both reference it by name — the actual live-proof execution remains 09-06/09-07's deliverable, not this plan's"
  - "Folded todo marked resolved in place (front-matter status/resolved_at/resolved_by_phase fields plus a Resolution section) rather than moved — no resolved/ subdirectory convention exists under .planning/todos/"

patterns-established: []

requirements-completed: []

coverage:
  - id: D1
    description: "docs/RELEASE-PROCEDURES.md no longer instructs a maintainer to create/push a version tag by hand as the normal path; §4 documents the release-PR-merge flow with the hand-pushed tag surviving only as the explicitly-labelled rc escape hatch (D-10)"
    verification:
      - kind: other
        ref: "grep -c 'git tag' inside §4's escape-hatch subsection only; §4's opening prose names the release-PR merge as the trigger"
        status: pass
    human_judgment: false
  - id: D2
    description: "§3 records the fast-forward merge model with repo evidence (zero merge commits, main is an ancestor today) and explicitly supersedes Phase-8 08-CONTEXT.md's squash-merge wording rather than silently contradicting it (D-09)"
    verification:
      - kind: other
        ref: "grep -q 'fast-forward' docs/RELEASE-PROCEDURES.md; §3 names 08-CONTEXT.md D-08 by identifier"
        status: pass
    human_judgment: false
  - id: D3
    description: "§7 rollback covers all three artifacts a release-please cut leaves behind (tag, GitHub Release, manifest/changelog bump commit + its PR), including the autorelease label consideration"
    verification:
      - kind: other
        ref: "manual read-through of §7; all three artifacts named, autorelease label check present"
        status: pass
    human_judgment: false
  - id: D4
    description: "§1's pre-tag sweep is documented as now-automated in release-please.yml's pretag-gate job, demoted from enforcement point to documentation, without silently dropping the D-09 lesson"
    verification:
      - kind: other
        ref: "grep -q 'pretag-gate' docs/RELEASE-PROCEDURES.md; §1's original command block byte-unchanged (diff confirmed)"
        status: pass
    human_judgment: false
  - id: D5
    description: "§9 documents the GitHub App prerequisite end to end: creation, three installation permissions, that installation scope differs from a workflow's permissions: block, the two secret names, the deprecated-input note, and the branch-protection bypass-actor requirement"
    verification:
      - kind: other
        ref: "grep -q 'APP_PRIVATE_KEY'/'bypass' docs/RELEASE-PROCEDURES.md; §9 manually reviewed against all six required elements"
        status: pass
    human_judgment: false
  - id: D6
    description: "§10 records every divergence this phase accepted (roadmap criterion 3 / D-05, D-08's file location, D-03's Path B with its guard-plus-test requirement), each naming its source decision and reason"
    verification:
      - kind: other
        ref: "grep -qi 'divergence' docs/RELEASE-PROCEDURES.md; §10 contains exactly 3 lettered entries (a)/(b)/(c), each naming a decision ID"
        status: pass
    human_judgment: false
  - id: D7
    description: "ROADMAP.md's Phase 9 success criterion 3 carries a recorded amendment in the same style as Phase 8's goal amendment, amended in place with rationale rather than deleted or left standing as an unmet claim"
    verification:
      - kind: other
        ref: "grep -c 'criterion 3' .planning/ROADMAP.md == 1; blockquote amendment present below the criteria list, dated and naming D-05"
        status: pass
    human_judgment: false
  - id: D8
    description: "§4 documents the zero-release-worthy-commits silent no-op so a maintainer does not mistake it for a broken pipeline"
    verification:
      - kind: other
        ref: "manual read-through of §4's 'Expected silent no-op' paragraph"
        status: pass
    human_judgment: false
  - id: D9
    description: "docs/RELEASE.md's user-facing verification commands are confirmed still correct against the unchanged cosign identity, with the confirmation recorded rather than assumed"
    verification:
      - kind: other
        ref: "grep -c the anchored SAN pattern in docs/RELEASE.md against internal/upgrade/verify.go's releaseWorkflowRefPattern constant — both byte-identical (see this SUMMARY's docs/RELEASE.md Comparison section)"
        status: pass
    human_judgment: false
  - id: D10
    description: "The folded todo is marked resolved with a pointer to the sections that closed it"
    verification:
      - kind: other
        ref: ".planning/todos/pending/2026-07-14-document-release-cut-procedures-runbook.md front matter + Resolution section"
        status: pass
    human_judgment: false
  - id: D11
    description: "Zero code changes: git diff shows no changes to workflows, internal/upgrade/*.go, or .goreleaser.yaml; go test ./internal/upgrade/... green"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -count=1"
        status: pass
      - kind: other
        ref: "git diff --quiet -- internal/upgrade .github .goreleaser.yaml (exit 0 after every task commit)"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-07-29
status: complete
---

# Phase 9 Plan 4: RELEASE-PROCEDURES.md rewrite + recorded divergences Summary

**Rewrote docs/RELEASE-PROCEDURES.md's §3/§4/§7 for the release-please-automated flow, added a §9 GitHub App setup runbook and a §10 recorded-divergences ledger, amended ROADMAP.md's GoReleaser criterion in place with a dated D-05 rationale, and resolved the folded maintainer-runbook todo — a pure documentation plan, zero source/workflow diff.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 3
- **Files modified:** 3 (all modified, none created)

## Accomplishments
- §3 now documents the fast-forward merge model with the repo evidence that settled it (zero merge commits on `main`, `main` an ancestor of the integration branch, ~477/160/217 commit split at decision time), explicitly superseding Phase 8's `08-CONTEXT.md` D-08 squash-merge wording on evidence while honoring its substance in full.
- §4 replaces the hand-run `git tag` with the release-PR-merge trigger flow (pretag-gate → release-please PR → merge → tag+Release in one API call → `release.yml` fires → `assemble` uploads with `--clobber`), documents the `Release-As:` one-shot footer and the zero-release-worthy-commits silent no-op, and retains the hand-pushed tag as an explicitly-labelled `rc`-only escape hatch (D-10).
- §7 now covers all three artifacts a release-please cut leaves behind (tag, GitHub Release, version-bump commit + its PR) instead of just the tag, including the `autorelease` label consideration for a retried cut.
- New §9 documents the one-time GitHub App setup end to end (three installation permissions, two secret names, the installation-scope-vs-workflow-permissions distinction, the deprecated `app-id`/`client-id` input note, pre-flight `gh` commands, and the branch-protection bypass-actor requirement).
- New §10 records the three divergences this phase accepted (roadmap criterion 3/D-05, D-08's `pr-title.yml` file location, D-03's deliberately-unimplemented Path B with its full 5-step recipe), each naming its source decision.
- ROADMAP.md's Phase 9 success criterion 3 rewritten to state what shipped (idempotent upload via `--clobber`, `goreleaser build --single-target` retained) with a dated blockquote amendment below the criteria list, matching Phase 8's amendment style.
- The folded todo (`2026-07-14-document-release-cut-procedures-runbook.md`) marked resolved in place with a pointer to the closing sections.
- §1, §2, §5, §6, §8 confirmed substantively unchanged (§5's verbatim `verify.go` constants block and §6's two command blocks are byte-identical; §1's original command block is byte-identical, with an additive automation note appended after it).

## Task Commits

1. **Task 1: Rewrite §3/§4/§7 for the release-please flow** - `62916bc` (docs)
2. **Task 2: New §9 GitHub App prerequisite + §10 recorded divergences + §1/§6 notes** - `a9ee863` (docs)
3. **Task 3: Amend ROADMAP Phase 9 criterion 3 and resolve the folded todo** - `b5d6745` (docs)

## Files Created/Modified
- `docs/RELEASE-PROCEDURES.md` - §3/§4/§7 rewritten; §9/§10 added; additive notes to §1/§6; §2/§5/§8 untouched
- `.planning/ROADMAP.md` - Phase 9 success criterion 3 rewritten + dated amendment blockquote (scoped edit, Phase 9 section only)
- `.planning/todos/pending/2026-07-14-document-release-cut-procedures-runbook.md` - marked resolved in place

## Decisions Made
- Both tasks touching `docs/RELEASE-PROCEDURES.md` (Task 1: §3/§4/§7; Task 2: §1/§6 notes + §9/§10) were committed as two atomic, separately-verifiable commits rather than one combined commit, by writing the full final content first, verifying it, then temporarily reverting Task 2's additions, re-running Task 1's acceptance criteria against that intermediate state, committing, and re-applying Task 2's additions for its own commit. This preserves per-task git history granularity even though both tasks share one file.
- §3's cited commit-split numbers (roughly 477/160/217) are D-09's decision-time evidence, reproduced as-written rather than re-measured against the live branch (which has since grown to 493/165/228 commits ahead as work continued) — the runbook is documenting the evidence that justified the fast-forward decision, not maintaining a live counter.
- §9 introduces the "disposable live proof" concept in prose (a throwaway prerelease cycle proving the App-token wiring end-to-end before the real `v1.0.0` cut, per `09-RESEARCH.md`'s Validation Architecture section) because §6's and §7's cross-references to it needed a real definition to point at — the live proof's actual execution remains plan 09-06/09-07's deliverable.
- The folded todo was marked resolved via front-matter fields (`status`, `resolved_at`, `resolved_by_phase`) plus an inline Resolution section, and left in `.planning/todos/pending/` since no `resolved/` subdirectory convention exists in this repo's todo tree.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' acceptance criteria were satisfied without requiring an architectural change or an out-of-scope fix.

## `docs/RELEASE.md` Comparison (must_haves item)

Confirmed `docs/RELEASE.md`'s user-facing cosign verification command (§1a) still matches `internal/upgrade/verify.go`'s live `releaseWorkflowRefPattern` constant:

```
verify.go:   ^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$
RELEASE.md:  ^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$
```

Both are full-match anchored patterns for the same identity; the only textual difference is `[^\s]` (Go regexp shorthand) vs. `[^[:space:]]` (POSIX ERE class for `grep -E`/`bash`), which are equivalent character classes in their respective regex dialects — not a semantic divergence. `RELEASE-PROCEDURES.md` §6 carries the identical POSIX-class form. No mismatch found; `docs/RELEASE.md` left unchanged.

## Issues Encountered
None.

## User Setup Required
None for this plan. The GitHub App creation/installation this plan's new §9 documents remains plan 09-05's blocking human checkpoint.

## Next Phase Readiness
- The runbook now accurately describes the pipeline built by 09-01/09-02/09-03: fast-forward merge, release-PR-merge trigger, three-artifact rollback, GitHub App prerequisite, and every accepted divergence.
- 09-05 (the GitHub App creation checkpoint) can proceed directly against §9's procedure.
- 09-06's fast-forward merge can proceed directly against §3's documented model; 09-06/09-07's disposable live proof and eventual real `v1.0.0` cut can follow §4/§6/§7/§9 as written.
- No blockers for 09-05 onward.

## Self-Check: PASSED

`docs/RELEASE-PROCEDURES.md`, `.planning/ROADMAP.md`, and `.planning/todos/pending/2026-07-14-document-release-cut-procedures-runbook.md` all confirmed modified on disk with the expected content (§1-§10 headers present in order, ROADMAP criterion 3 amendment present, todo front matter carries `status: resolved`). All 3 task commit hashes (`62916bc`, `a9ee863`, `b5d6745`) confirmed present in `git log --oneline`.

---
*Phase: 09-release-please-and-goreleaser*
*Completed: 2026-07-29*
