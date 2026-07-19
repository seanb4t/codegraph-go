---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 09
subsystem: release-engineering
tags: [release, cosign, slsa, sbom, docs, drop-in-parity, verify.go]

# Dependency graph
requires:
  - phase: 08-06
    provides: docs/FLAG-PARITY.md drift test (SURF-05 flag audit, green)
  - phase: 08-07
    provides: Charm dependency-closure CGo/govulncheck/SBOM audit (REL-01, green)
  - phase: 08-08
    provides: refreshed docs/BENCHMARKS.md head-to-head numbers (REL-03)
provides:
  - docs/RELEASE-PROCEDURES.md maintainer release-cut runbook
  - PROJECT.md "not yet drop-in" caveat retired at every site (REL-04 gate closed)
  - a structured, described maintainer-manual handoff for the v1.0.0 tag push
affects: [milestone-completion, v1.0-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Release runbook (maintainer-facing) as a docs/RELEASE.md sibling, not a merge into it"
    - "Two-half drop-in gate (behavioral harness AND flag-parity drift test) both green as the sole precondition for a caveat-retirement doc edit"

key-files:
  created:
    - docs/RELEASE-PROCEDURES.md
  modified:
    - .planning/PROJECT.md

key-decisions:
  - "REL-02 (the actual signed v1.0.0 tag cut) is NOT marked complete — only the release-readiness work (the runbook) is delivered by this plan; the tag push itself is the maintainer's own manual action per D-08/D-11 and this plan's explicit prohibition"
  - "Fixed one additional stale cross-reference beyond the 4 enumerated caveat sites (the 'Parity v1 -> team features v2' Key Decisions row, which literally said 'parity itself is not yet complete') for internal consistency with the retirement edit — a Rule 1 auto-fix, not scope creep"

requirements-completed: [REL-04]

coverage:
  - id: D1
    description: "docs/RELEASE-PROCEDURES.md maintainer runbook: pre-tag 6-target go list -mod=readonly gate, tag conventions, branch/tag model, verify.go LOCKED contract cited verbatim, post-release cosign/slsa verification, rc-tag rollback, gpgsign pipeline-only caveat"
    requirement: REL-02
    verification:
      - kind: other
        ref: "test -f docs/RELEASE-PROCEDURES.md && grep -q 'go list -mod=readonly' docs/RELEASE-PROCEDURES.md && grep -q 'releaseWorkflowRefPattern' docs/RELEASE-PROCEDURES.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "REL-04 drop-in gate run green (TEST-01 behavioral harness + FLAG-PARITY drift test) BEFORE editing PROJECT.md"
    requirement: REL-04
    verification:
      - kind: integration
        ref: "go test ./testdata/golden/... -count=1"
        status: pass
      - kind: unit
        ref: "go test ./internal/cli/... -run FlagParity -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "PROJECT.md 'not yet drop-in' caveat retired at every site (4 confirmed + 1 adjacent stale cross-reference), rg zero-hit confirmed; verify.go and release.yml byte-unchanged"
    requirement: REL-04
    verification:
      - kind: other
        ref: "rg -q 'not yet drop-in' .planning/PROJECT.md (expected exit 1 / no match)"
        status: pass
      - kind: other
        ref: "git diff --stat internal/upgrade/verify.go .github/workflows/release.yml (expected empty)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Maintainer-manual pre-tag gate + signed v1.0.0 tag push — the final action of the phase and of v1.0"
    requirement: REL-02
    verification: []
    human_judgment: true
    rationale: "Per D-08/D-11 and this plan's explicit prohibition, the executor MUST NOT create or push the v1.0.0 git tag. This is a maintainer-manual action with no automatable substitute — checkpoint returned to the orchestrator for human execution."

# Metrics
duration: 20min
completed: 2026-07-19
status: complete
---

# Phase 8 Plan 09: Release Runbook, Drop-In Gate & Caveat Retirement Summary

**Wrote the maintainer release-cut runbook, ran both drop-in-gate halves green, retired PROJECT.md's "not yet drop-in" caveat at every site, and handed off the maintainer-manual v1.0.0 tag push as a checkpoint — the final action of Phase 8 and of the v1.0 milestone.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2 of 3 completed (Task 3 is a maintainer-manual checkpoint, correctly not executed by this agent)
- **Files modified:** 2 (1 created, 1 edited)

## Accomplishments

- **docs/RELEASE-PROCEDURES.md** — the REL-02 maintainer runbook: the mandatory pre-tag 6-target `go list -mod=readonly ./...` sweep (the exact check that would have caught v0.1's `rc.1` linux-only `go.sum` failure), tag conventions (`v0.0.0-rc.N` prerelease / `vX.Y.Z` stable / `milestone-v*` never-fires), the branch/tag model (integration branch → squash-merge `main` → tag `main`), what the tag push triggers (`release.yml`'s `build`/`assemble`/`provenance` jobs), the `verify.go` LOCKED contract cited **verbatim**, post-release `cosign verify-blob`/`slsa-verifier verify-artifact` commands, rc-tag rollback/cleanup, and the `-c commit.gpgsign=false` pipeline-only caveat.
- **Drop-in gate run green before any caveat edit**: `go test ./testdata/golden/... -count=1` (Phase-1 TEST-01 behavioral harness vs real TS 1.3.1) and `go test ./internal/cli/... -run FlagParity -count=1` (docs/FLAG-PARITY.md drift test) both passed.
- **PROJECT.md caveat retirement**: all 4 confirmed sites (milestone goal line, repo-state paragraph, the "Full parity in v1" Key Decisions row, the "Milestone v1.0" decision-log row) plus one adjacent stale cross-reference reworded to reflect "parity closed, v1.0.0 shipped." `rg -n "not yet drop-in" .planning/PROJECT.md` now returns zero hits.
- **verify.go and release.yml confirmed byte-unchanged** (`git diff --stat` empty for both) throughout this plan.
- **Maintainer checkpoint returned** for the pre-tag gate + `git tag v1.0.0 && git push` — the executor did not create or push any tag.

## Task Commits

Each task was committed atomically:

1. **Task 1: docs/RELEASE-PROCEDURES.md maintainer runbook (REL-02 folded todo)** - `be45690` (docs)
2. **Task 2: REL-04 drop-in gate + PROJECT.md caveat retirement** - `57da8c8` (docs)
3. **Task 3: Maintainer-manual pre-tag gate + signed v1.0.0 tag (LAST action)** - NOT executed by this agent; returned as a checkpoint (see below). No commit — this task produces no repository change until the maintainer acts.

**Plan metadata:** (this SUMMARY's own commit, see below)

## Files Created/Modified

- `docs/RELEASE-PROCEDURES.md` - New maintainer release-cut runbook (pre-tag gate, tag conventions, LOCKED verify.go contract, post-release verification, rollback, gpgsign caveat)
- `.planning/PROJECT.md` - "not yet drop-in" caveat retired at all 4 confirmed sites + 1 adjacent stale cross-reference; `internal/upgrade/verify.go`/`.github/workflows/release.yml` untouched

## Decisions Made

- REL-02 (the actual signed `v1.0.0` cut) is intentionally left `[ ]` (pending) in REQUIREMENTS.md — only the release-*readiness* work (the runbook) is this plan's deliverable for REL-02; the tag push itself remains the maintainer's manual action.
- Fixed one additional stale, self-contradicting cross-reference beyond the plan's 4 enumerated caveat sites (the "Parity v1 → team features v2" row, which literally read "parity itself is not yet complete") for internal consistency with the retirement edit — documented below as a Rule 1 auto-fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stale self-contradicting cross-reference in PROJECT.md's Key Decisions table**
- **Found during:** Task 2 (PROJECT.md caveat retirement)
- **Issue:** The "Parity v1 → team features v2" Key Decisions row contained the sentence "Note: parity itself is not yet complete (see above), so v0.1 precedes the parity/1.0 bar." — accurate before this plan's edits, but directly contradicted by the caveat-retirement edits made to the 4 confirmed sites in the same document, within the same table.
- **Fix:** Reworded the row's Outcome column to reflect parity closed and `v1.0.0` shipped, matching the other updated rows.
- **Files modified:** `.planning/PROJECT.md`
- **Verification:** `rg -n "not yet drop-in|not yet complete" .planning/PROJECT.md` returns zero hits for both phrasings after the fix; full re-run of `go test ./testdata/golden/...` and `go test ./internal/cli/... -run FlagParity` still green.
- **Committed in:** `57da8c8` (part of the Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 Rule 1)
**Impact on plan:** Necessary for internal document consistency; no scope creep — the fix is confined to the same document and table the plan already required editing.

## Issues Encountered

None. Both drop-in gate halves were confirmed green on the first run, `internal/upgrade/verify.go` and `.github/workflows/release.yml` diffs were confirmed empty throughout, and the caveat sweep matched exactly the 4 sites the plan and research had already located (plus the one adjacent stale row noted above).

## User Setup Required

None for Tasks 1-2. **Task 3 requires maintainer action** — see the checkpoint returned by this executor (and `docs/RELEASE-PROCEDURES.md` §§1-4) for the exact pre-tag gate and `git tag v1.0.0 && git push origin v1.0.0` sequence.

## Next Phase Readiness

This is the final plan of Phase 8 and of the v1.0 milestone. Everything GSD-trackable and automatable is complete: all SURF-01..05 surface reconciliation, the REL-01 Charm/CGo/govulncheck/SBOM audit, the REL-03 benchmark refresh, the REL-02 maintainer runbook, and the REL-04 drop-in gate + caveat retirement. The one remaining action — the pre-tag 6-target gate followed by `git tag v1.0.0 && git push` — is an intentional, permanent hand-off to the maintainer per D-08/D-11 and this plan's explicit prohibition; no further agent execution will perform it. Once pushed, `release.yml` builds/signs/attests the release and REL-02 can be marked complete in REQUIREMENTS.md.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED

- FOUND: docs/RELEASE-PROCEDURES.md
- FOUND: be45690 (Task 1 commit)
- FOUND: 57da8c8 (Task 2 commit)
- CONFIRMED: `rg -n "not yet drop-in" .planning/PROJECT.md` → zero hits
- CONFIRMED: `git diff --stat internal/upgrade/verify.go .github/workflows/release.yml` → empty (both files byte-unchanged)
