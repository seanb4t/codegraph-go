---
phase: 06-first-release-through-the-chain
plan: 02
subsystem: release-chain
tags: [changie, cosign, sigstore, codegraph-upgrade, post-release-verify, namespace-devbox, seed]

requires:
  - phase: 06-first-release-through-the-chain
    provides: "plan 06-01: release v0.15.0 published through the chain, SHIP-01 and SHIP-02 evidence in 06-EVIDENCE.md"
provides:
  - "post-release-verify run 39054786827 green (success, 7 of 7 jobs) on the tag commit, and GitHub latest release is v0.15.0"
  - "An installed v0.14.0 upgraded to v0.15.0 through the unchanged cosign identity on darwin/arm64 and linux/amd64 (SHIP-03)"
  - "SEED-005 annotated with the verified trigger; capability gsd-capability-changie v0.1.2 at 2631063f recorded (SHIP-04)"
  - "06-EVIDENCE.md Evidence summary (SC1 to SC4 all PASS) and the maintainer's D-12 Sign-off"
affects: [06-VERIFICATION, gsd-audit-milestone, gsd-complete-milestone, gsd-ship]

actuals:
  tokens: 3248
  tasks: 3
  commits: 2
plan_head_before: da1585b4fce2d716507b382fa2937b630341e2f3
plan_head_after: c6667e7e14b4f8e5aa136eae0dbc29b474cb5ae6

tech-stack:
  added: []
  patterns:
    - "Upgrade proofs run from the published prior asset in an empty scratch dir that is also HOME, with the maintainer's skill manifest sha256 asserted unchanged"
    - "linux/amd64 proofs run on an ephemeral Namespace devbox when OrbStack amd64 emulation fails"

key-files:
  created:
    - .planning/phases/06-first-release-through-the-chain/06-02-SUMMARY.md
  modified:
    - .planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md
    - .planning/seeds/SEED-005-changie-template-for-other-repos.md

key-decisions:
  - "The D-12 sign-off was the maintainer's reply 'and approved' (2026-10-17), recorded verbatim in E under ## Sign-off; it was not written before the reply arrived"
  - "SEED-005 was annotated by filling its existing ## When to Surface section with one prose paragraph; status: dormant and trigger_when were left alone"

requirements-completed: [SHIP-03, SHIP-04]

duration: 19min
completed: 2026-10-17
status: complete
---

# Phase 6 Plan 02: Upgrade Proofs, SEED-005 and Sign-off Summary

**An installed v0.14.0 verified and swapped to v0.15.0 in-process against the unchanged cosign identity on darwin/arm64 and an x86_64 Namespace devbox, post-release-verify is green on the tag commit, SEED-005 carries the verified trigger, and the maintainer signed off the evidence summary.**

## Performance

- **Duration:** about 19 min wall clock, from 06-01's close (17:43Z) to the maintainer's sign-off (about 18:02Z). It includes a devbox login and the wait at the blocking-human checkpoint.
- **Completed:** 2026-10-17
- **Tasks:** 3 (two auto, one `checkpoint:human-verify` with `gate="blocking-human"`)
- **Files modified:** 3 (E and SEED-005 modified, this SUMMARY created)

## Names

- TAG: `v0.15.0`
- SHA (tag commit): `0246de0e391b7de98052389800f8347fb9657d3b`
- Release: https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0 (published 2026-10-17T17:40:18Z)
- post-release-verify run: https://github.com/seanb4t/codegraph-go/actions/runs/39054786827 (`success`, 7 jobs, all `success`, selected by headSha equal to SHA)
- Capability: `gsd-capability-changie` `v0.1.2`, commit `2631063f05fffc7152d70a9c98eb0b7479a0029a` (installed checkout equals the tag commit in the capability repo; CONTRIBUTING.md lines 233 and 239 name the same tag)

## Accomplishments

- SHIP-03: `gh api repos/seanb4t/codegraph-go/releases/latest` names `v0.15.0` (A4 holds), `internal/upgrade/verify.go` is byte-identical between v0.14.0 and the tag commit, and post-release-verify concluded `success` with 7 of 7 jobs `success`.
- SHIP-03 darwin/arm64: the published v0.14.0 raw asset, run with `HOME` set to an empty scratch dir that was also the cwd, printed `codegraph version v0.14.0 (commit 80377d28...`, then `codegraph v0.15.0 is available (current: v0.14.0)`, `upgraded to v0.15.0`, and `codegraph version v0.15.0 (commit 0246de0e..., built 2026-10-17T17:35:19Z)`. The sha256 of the maintainer's `~/.claude/skills/codegraph/.codegraph-manifest.json` was `eb80e383...bd7b45` before and after. The Homebrew install was not touched.
- SHIP-03 linux/amd64: the same four outputs on an ephemeral Namespace devbox (`uname -m` printed `x86_64`, Ubuntu 24.04.3 LTS). The devbox was expired afterwards and `devbox list` no longer shows `p06-upgrade-amd64`.
- SHIP-04: SEED-005 gained one prose paragraph (8 lines added, 0 deleted) inside `## When to Surface` naming the tag, its Release URL, the date, and `gsd-capability-changie v0.1.2 (commit 2631063f...)`. Frontmatter, heading count and every line outside that section match origin/main.
- D-12: E ends with `## Evidence summary` (four rows, SC1 to SC4, all PASS, the `05-UAT test 3` and `05-UAT test 4` links, `D-10 re-runs:`, A1 to A4 outcomes, D-06 carry-forward) and `## Sign-off`, which quotes the maintainer's reply verbatim.

## Task Commits

1. **Task 1: post-release-verify green and the v0.14.0 upgrade proofs (SHIP-03)** - `a9cb1372` (docs)
2. **Task 2: SEED-005 annotation, capability tag record, evidence summary (SHIP-04, D-12)** - `c6667e7e` (docs)
3. **Task 3: the maintainer's sign-off (D-12)** - `checkpoint:human-verify`, `gate="blocking-human"`; the `## Sign-off` edit to E is committed with this SUMMARY, as the task's acceptance criteria direct

**Plan metadata:** committed separately (docs: complete plan), together with E's sign-off, STATE.md, ROADMAP.md and REQUIREMENTS.md.

**Sign-off reply, verbatim:** `and approved` (maintainer, 2026-10-17)

## Files Created/Modified

- `.planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md` - appended `SHIP-03 post-release-verify`, `SHIP-03 upgrade darwin/arm64`, `SHIP-03 upgrade linux/amd64`, `SHIP-04 SEED-005 and capability tag`, `Evidence summary` and `Sign-off`
- `.planning/seeds/SEED-005-changie-template-for-other-repos.md` - one paragraph in `## When to Surface`
- `.planning/phases/06-first-release-through-the-chain/06-02-SUMMARY.md` - this file

The branch still differs from origin/main only under `.planning/` (D-04), adds nothing under `.changes/` (D-06), and carries no GitHub CI-skip phrase. Nothing was pushed; the branch has no upstream on purpose. `06-VERIFICATION.md` was not written; gsd-verifier owns it.

## Verify Runs

- Task 1 and Task 2 automated verify commands were run by the previous executor agent and passed before commits `a9cb1372` and `c6667e7e`.
- This agent re-ran the plan-level checks after the sign-off edit: `rg '^## Sign-off'` finds the section and the verbatim reply; no CI-skip phrase in `git log --format=%B origin/main..HEAD`; `git diff --name-only origin/main...HEAD -- ':(exclude).planning/'` is empty; `git diff --name-only origin/main...HEAD -- .changes/` is empty and `.changes/unreleased/` holds only `.gitkeep`.

## Decisions Made

None beyond the plan.

## Deviations from Plan

None - plan executed exactly as written, with these recorded notes (none changed scope or outcome):

- **D-10 re-runs:** none in this plan. post-release-verify run 39054786827 was not re-run and neither upgrade leg was repeated to reach a pass. (The one re-run in the phase was 06-01's ci.yml run 39053062791 for the known `DB Size:` flake, engram w2t54bym10.)
- **Unsigned commits:** commits `a9cb1372`, `c6667e7e` and this plan's metadata commit carry no signature (`%G?` is `N`). The local `.git/config` already has `commit.gpgsign=false`, a pre-existing setting, so no `git -c commit.gpgsign=false` override was used and signing was not attempted (rule xmz3xknbj0 context).
- **Stale plan ledger reset:** `.git/gsd-plan-head-before-06-02` held `a97234d1`, left over from an earlier attempt, which would have counted another plan's commits. It was reset to `da1585b4fce2d716507b382fa2937b630341e2f3` (06-01's closing commit), so `commits: 2` is the measured `git rev-list --count da1585b4..c6667e7e` at SUMMARY write time. The metadata commit that carries this SUMMARY comes after the measurement.
- **Upstream gap (no gsd-tools verb annotates a planted seed):** `list-seeds` is read-only and `plant-seed --enrich` rewrites only trigger, scope and why. The annotation is a value written into the existing `## When to Surface` section; no heading or frontmatter key was invented (generated-files rule). Worth reporting upstream as a missing verb.

## Authentication Gates

**Task 1, linux/amd64 leg:** the first `devbox create` failed with `Not logged in to Namespace`. `devbox auth check-login` printed nothing and exited 0 in that logged-out state, so it is not a reliable login test. `devbox login` completed the browser flow (`You are now logged into workspace "Personal"`) and the create was repeated and succeeded. This is normal flow for an auth gate, not a deviation.

Exact devbox create flags: `devbox create --ephemeral --platform linux/amd64 --no_checkout --name p06-upgrade-amd64 --purpose 'v0.14.0 upgrade proof (Phase 6 SHIP-03)' --image builtin:default --size s`. No `--setup_github`, so no GitHub credential reached the devbox (T-06-09). `devbox create` also wrote a `p06-upgrade-amd64.devbox.namespace` entry into `~/.ssh/config`; none remains after teardown. `ca-certificates` needed no install (A2 holds).

## Issues Encountered

None blocking.

## Known Stubs

None. This plan adds no code.

## Threat Flags

None. No new network endpoint, auth path, file access pattern or schema was introduced. The plan downloaded public release assets, ran them with an isolated HOME, and wrote planning files. T-06-05 (manifest unchanged), T-06-07 (headSha-only selection, `success 7 0`), T-06-08 (sign-off from the maintainer's reply only) and T-06-09 (no credentials on the devbox) were each applied as planned.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

All four Phase 6 success criteria are PASS in E and signed off. Next, as post-phase GSD steps:

1. `/gsd-verify-work 6`: gsd-verifier writes `06-VERIFICATION.md` from E.
2. `/gsd-audit-milestone`: the final pass over all 29 requirements (D-05).
3. `/gsd-complete-milestone` (D-04).
4. `/gsd-ship`: pushes the follow-up branch and opens the one planning-only PR (D-04). Before that PR merges, re-run `git diff --name-only origin/main...HEAD -- .changes/unreleased/` and expect no output (D-06). The capability's verify:post at phase close must not write a fragment.

---
*Phase: 06-first-release-through-the-chain*
*Completed: 2026-10-17*

## Self-Check: PASSED

- `06-EVIDENCE.md` exists with `## SHIP-03 post-release-verify`, `## SHIP-03 upgrade darwin/arm64`, `## SHIP-03 upgrade linux/amd64`, `## SHIP-04 SEED-005 and capability tag`, `## Evidence summary` and `## Sign-off`.
- Commits `a9cb1372` and `c6667e7e` are ancestors of HEAD on `gsd/v0.15.0-changie-release-management`.
- Tag `v0.15.0` resolves to `0246de0e391b7de98052389800f8347fb9657d3b` (recorded in E from the GitHub API).
