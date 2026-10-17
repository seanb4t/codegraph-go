---
phase: 06-first-release-through-the-chain
plan: 01
subsystem: release-chain
tags: [changie, github-app, release-please-retirement, goreleaser, cosign, slsa]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: release-pr.yml, release-tag.yml (App-token tag job), locked release.yml, changie config
provides:
  - "Live, recorded proof that the chain runs end to end: App-opened release PR #101, maintainer squash merge, App-pushed tag v0.15.0, release.yml publish, content-equal Release body"
  - "06-EVIDENCE.md transcripts for SHIP-01 and SHIP-02 (input to 06-02's D-12 evidence summary and sign-off)"
affects: [06-02, SHIP-03, SHIP-04]

actuals:
  tokens: 3700
  tasks: 3
  commits: 4
plan_head_before: ab49d80ef3dc933e9a73096268ba12db8c0098b3
plan_head_after: 7942130975d1b35e91c13412c6de676880b15763

tech-stack:
  added: []
  patterns:
    - "Read-only live observation recorded as fenced transcripts in a phase evidence file (after the 05-MUTATION-LOG precedent)"

key-files:
  created:
    - .planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md
    - .planning/phases/06-first-release-through-the-chain/06-01-SUMMARY.md
  modified: []

key-decisions:
  - "The derived version v0.15.0 matched the prediction exactly; no ruling needed on a different version"
  - "A recorded DB Size test flake (engram w2t54bym10) received the single D-10 re-run on ci.yml run 39053062791; nothing else was re-run"

requirements-completed: [SHIP-01, SHIP-02]

duration: 25min
completed: 2026-10-17
status: complete
---

# Phase 6 Plan 01: First Release Through the Chain Summary

**v0.15.0 shipped through the chain Phase 5 built: the App opened release PR #101, the maintainer squash-merged it, the App pushed tag v0.15.0, release.yml published on that push, and the Release body is content-equal to `.changes/v0.15.0.md`.**

## Performance

- **Duration:** about 25 min wall clock from execution start (17:18Z) to the final evidence commit (17:42Z), including the maintainer's merge and a roughly four-minute release.yml wait
- **Started:** 2026-10-17T17:18:17Z
- **Completed:** 2026-10-17T17:43Z
- **Tasks:** 3 (one tracer, one human-action checkpoint, one auto)
- **Files modified:** 1 evidence file created, committed twice

## Names

- M (#88 squash commit): `ab49d80ef3dc933e9a73096268ba12db8c0098b3`
- N (release PR): 101, https://github.com/seanb4t/codegraph-go/pull/101
- TAG: `v0.15.0`, equal to the pre-merge `task changie -- next auto` prediction as an exact string
- SHA (release PR squash commit, the tag commit): `0246de0e391b7de98052389800f8347fb9657d3b`
- release-pr.yml on M: https://github.com/seanb4t/codegraph-go/actions/runs/39052978056 (`release:batch: rendered v0.15.0 from 1 fragment(s)`)
- release-pr.yml on SHA: https://github.com/seanb4t/codegraph-go/actions/runs/39054420946 (`nothing to release`)
- release-tag.yml on SHA: https://github.com/seanb4t/codegraph-go/actions/runs/39054420866 (`release:tag: pushed v0.15.0 at 0246de0e391b7de98052389800f8347fb9657d3b`)
- release.yml on the tag push: https://github.com/seanb4t/codegraph-go/actions/runs/39054491964 (event push, attempt 1, success)
- Release: https://github.com/seanb4t/codegraph-go/releases/tag/v0.15.0 (published 2026-10-17T17:40:18Z, not draft, not prerelease)

## Accomplishments

- SHIP-01: release-pr.yml on #88's merge rendered v0.15.0 from the one `Breaking` fragment; exactly one `changie/release` PR exists, authored by `app/fzy-release-please`, not a draft, titled `chore(main): release v0.15.0`; all 9 required checks (equal to the live protect-main count) report `pass`. After the maintainer's merge, origin/main merged into the follow-up branch and `task changie -- latest` prints `v0.15.0`.
- SHIP-02: the tag push came from the App (release.yml run `actor` and attempt 1 `triggering_actor` are both `fzy-release-please[bot]`), the run is `push` of `v0.15.0` at SHA and concluded success, and the Release body equals `.changes/v0.15.0.md` under the D-11 normalization (`CONTENT-EQUAL lines=5`, both sides non-empty).
- Edge cases observed: the #88 merge did not fire release-tag.yml (empty `--commit` query, with the release-pr.yml run on the same commit as the positive control); on the release squash commit release-pr.yml logged `nothing to release` and no second `changie/release` PR opened (the one PR is MERGED).

## Task Commits

1. **Task 1: Pre-flight, follow-up branch, and the App-opened release PR recorded** - `c4d56838` (docs)
2. **Task 2: Maintainer squash-merges the release PR** (checkpoint:human-action, no commit) - verified `state MERGED`, `mergedBy.login` `seanb4t`, merge commit `0246de0e391b7de98052389800f8347fb9657d3b`, merged 2026-10-17T17:35:19Z
3. **Task 3: Record the App-authored tag, release run, `changie latest` and the content-equal body** - `bdea3c83` (merge of origin/main into the follow-up branch, subject `chore(06-01): merge main into the follow-up branch after the release PR merged`) and `79421309` (docs, evidence sections)

**Plan metadata:** committed separately after this file (docs: complete plan).

**Task 2 reply, verbatim:** `PR merged`

## Files Created/Modified

- `.planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md` - live-run transcripts for SHIP-01 and SHIP-02 (sections: Pre-flight, SHIP-01 release PR, Release-PR merge, SHIP-01 changie latest on main, SHIP-02 App-authored tag and release.yml, SHIP-02 Release body (D-11))

The follow-up branch still differs from origin/main only under `.planning/` (D-04). Nothing was pushed.

## Verify Runs (Task 1 tracer)

- **RED:** the RED run required by Task 1 step 1 was taken by the previous executor agent before it created E. Its output was not carried in the handoff, so it is not reproduced here and this agent did not re-observe it. By construction the command could not pass at that point: E did not exist, so the `rg` checks on E and the `docs(06-01):` commit check would have failed.
- **GREEN:** re-run on the follow-up branch after the release (the `next auto: v0.15.0` line in E is what keeps it valid once no fragment remains): exit 0, printed `TASK1_VERIFY_GREEN`.
- Task 3's first automated verify: exit 0 (`VERIFY1_GREEN`). Its second (the `norm` + `cmp` comparison): `CONTENT-EQUAL lines=5`.
- Plan-level verification: no GitHub CI-skip phrase in `git log --format=%B origin/main..HEAD`; `git diff --name-only origin/main...HEAD -- ':(exclude).planning/'` empty.

## Decisions Made

None beyond the plan. The derived version equalled the prediction, so the "derived version is authoritative" branch was not exercised.

## Deviations from Plan

None - plan executed exactly as written, with these recorded notes (none changed scope or outcome):

- **D-10 re-runs:** one. Task 1, ci.yml run 39053062791 on the release PR: `TestColorFlagInvalidValue` failed on a one-line `DB Size:` difference (0.02 MB vs 0.01 MB), the known flake in engram w2t54bym10, and the orchestrator pre-authorized the single re-run for that signature (`gh run rerun 39053062791 --failed`; attempt 2 succeeded). Task 3: none. release-pr.yml 39054420946, release-tag.yml 39054420866, release.yml 39054491964 and the ci.yml run on SHA (39054420975) each ran once and succeeded. The flake itself is a pre-existing nondeterministic test and out of scope here; it is not fixed by this plan.
- **Unsigned commits:** the local `.git/config` has `commit.gpgsign=false`, so commits `c4d56838`, `bdea3c83` and `79421309` carry no signature (`%G?` is `N`). No `git -c commit.gpgsign=false` override was needed; signing was not attempted because the repository setting already disables it (rule xmz3xknbj0 context). The release squash commit `0246de0e` is GitHub's own.
- **`commits: 4` in the frontmatter** is the measured `git rev-list --count ab49d80e..HEAD`. One of the four is main's release squash commit `0246de0e` brought in by the merge; three were authored by this plan (`c4d56838`, `bdea3c83`, `79421309`).
- **Release record author:** the Release's `author.login` is `github-actions[bot]` (GoReleaser publishes with the workflow `GITHUB_TOKEN`). That is distinct from the tag's author; the gated authorship property (the tag push, read from attempt 1) is `fzy-release-please[bot]`.

## Issues Encountered

None blocking. The one flaky test above was handled under D-10.

## Known Stubs

None. This plan adds no code.

## Threat Flags

None. No new network endpoint, auth path, file access pattern or schema was introduced; the plan only read from GitHub and wrote one planning file.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The release is real and stable (D-09): tag `v0.15.0` and its Release are published. Recovery from any defect is patch-forward only.
- 06-02 can proceed (SHIP-03 post-release-verify and `codegraph upgrade` from v0.14.0, SHIP-04 SEED-005 annotation, and the D-12 evidence summary). The maintainer's single sign-off on SHIP-01 and SHIP-02 (D-12) is 06-02 Task 3; these two requirements are marked complete on the strength of the recorded evidence in E, pending that sign-off.
- The follow-up branch `gsd/v0.15.0-changie-release-management` is local only and unpushed; publishing it is `/gsd-ship`'s job after verification (D-04).

---
*Phase: 06-first-release-through-the-chain*
*Completed: 2026-10-17*

## Self-Check: PASSED

- `06-EVIDENCE.md` exists, with all six required sections.
- Commits `c4d56838`, `bdea3c83` and `79421309` are ancestors of HEAD; `0246de0e` is the second parent of `bdea3c83`, with two parents confirmed by `git rev-list --parents -n 1`.
- Tag `v0.15.0` resolves to `0246de0e391b7de98052389800f8347fb9657d3b` through the GitHub API.
