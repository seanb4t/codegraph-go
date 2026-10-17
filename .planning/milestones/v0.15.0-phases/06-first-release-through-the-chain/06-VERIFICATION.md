---
phase: 06-first-release-through-the-chain
verified: 2026-10-17T18:20:00Z
status: passed
score: 8/8 must-haves verified
covered_files: [".planning/phases/06-first-release-through-the-chain/06-01-PLAN.md", ".planning/phases/06-first-release-through-the-chain/06-01-SUMMARY.md", ".planning/phases/06-first-release-through-the-chain/06-02-PLAN.md", ".planning/phases/06-first-release-through-the-chain/06-02-SUMMARY.md", ".planning/phases/06-first-release-through-the-chain/06-EVIDENCE.md", ".planning/seeds/SEED-005-changie-template-for-other-repos.md"]
covered_digest: "v3:sha256:35991b7885f1ffb717040637eb387985c41d7ac6918aeecd5bb166b041ff4247"
behavior_unverified: 0
overrides_applied: 0
flagged_prohibitions: 6
---

# Phase 6: First Release Through the Chain Verification Report

**Phase Goal:** A real release ships through the new chain end to end: the workflow opens the release PR from this milestone's fragments, the maintainer merges it, the GitHub App pushes the tag, the locked `release.yml` builds, signs and attests with the changie file as the Release body, and an installed v0.14.0 upgrades to it with verification.
**Verified:** 2026-10-17T18:20:00Z
**Status:** passed
**Re-verification:** No, initial verification

The deliverable is a live release, not code. Every claim below was re-observed against live GitHub state and the repository with read-only commands (`gh api`, `gh run view`, `gh release view`, `git`, `task changie -- latest`). `06-EVIDENCE.md` and the SUMMARYs were used as a map, not as evidence. I did not push, tag, edit releases, re-run workflows, run `codegraph upgrade`, or touch the Homebrew install.

## Goal Achievement

### Observable Truths

| #   | Truth | Status | Evidence (re-observed this pass) |
| --- | ----- | ------ | -------------------------------- |
| 1 | SC1 / SHIP-01: the workflow opened one `chore(main): release vX.Y.Z` PR from the milestone's fragment, the maintainer merged it, and `changie latest` on main answers the derived version | ✓ VERIFIED | `gh pr list --head changie/release --state all` returns exactly one PR, #101, author `app/fzy-release-please`, `isDraft` false, title `chore(main): release v0.15.0`, state MERGED, `mergedBy.login` `seanb4t`, merge commit `0246de0e391b7de98052389800f8347fb9657d3b`. PR body is the rendered `.changes/v0.15.0.md`. `task changie -- latest` prints `v0.15.0` (with `GOTOOLCHAIN=go1.26.9`). `origin/main` HEAD is `0246de0e (tag: v0.15.0)`, parent `ab49d80e` (#88). The edge case "no second release PR after the release merge" holds: one PR on the head, MERGED. |
| 2 | SC2 / SHIP-02: the tag was authored by the GitHub App (not a human, not `GITHUB_TOKEN`), `release.yml` ran on that tag push, and the Release body is content-equal to `.changes/v0.15.0.md` under the D-11 normalization | ✓ VERIFIED | `git/ref/tags/v0.15.0` is a lightweight tag (object type `commit`) at `0246de0e`. release-tag run 39054420866 (success) logs `release:tag: pushed v0.15.0 at 0246de0e...` and its job mints the token with `actions/create-github-app-token`. release.yml run 39054491964: event `push`, head_branch `v0.15.0`, head_sha `0246de0e...`, conclusion `success`, attempt 1; `actor` and attempt-1 `triggering_actor` are both `fzy-release-please[bot]`. Body check re-run by me: `norm` on the live Release body and on `git show 0246de0e:.changes/v0.15.0.md`, both non-empty, `cmp` prints `CONTENT-EQUAL 5`. Release is published, not draft, not prerelease, 17 assets including `.sigstore.json` and `.spdx.json` per platform. |
| 3 | SC3 / SHIP-03: `post-release-verify` is green against the published assets, and `codegraph upgrade` from an installed v0.14.0 verifies the new artifact against the unchanged cosign identity on darwin/arm64 and linux/amd64 | ✓ VERIFIED | post-release-verify run 39054786827 (event `workflow_run`, headSha `0246de0e...`): conclusion `success`, 7 of 7 jobs `success`. Its own `self-upgrade proof (darwin/arm64)` and `(linux/amd64)` jobs log `upgraded to v0.15.0` (re-read from the run log), which independently corroborates the two local proofs. `gh api releases/latest` names `v0.15.0`, so an unpinned `upgrade` resolves it. `git diff v0.14.0 0246de0e -- internal/upgrade/verify.go` is empty, so the cosign identity is unchanged. The local darwin/arm64 and Namespace-devbox linux/amd64 transcripts in `06-EVIDENCE.md` show `v0.14.0` → `upgraded to v0.15.0` → `--version` reporting the tag commit. I did not repeat them (the phase's own prohibition on running `upgrade` applies to this pass too). |
| 4 | SC4 / SHIP-04: SEED-005 is annotated with the verified trigger, and the capability tag CAP-04 installs is recorded as the one this release ran with | ✓ VERIFIED | `git diff origin/main -- SEED-005...` is exactly +8 lines, 0 deleted, one prose paragraph appended inside `## When to Surface`; frontmatter unchanged (`status: dormant`, `trigger_when` intact); no heading added. It names v0.15.0, the Release URL, the date 2026-10-17, and `gsd-capability-changie v0.1.2 (commit 2631063f05fffc7152d70a9c98eb0b7479a0029a)`. `.gsd-capabilities.json` records `0.1.2` from `...gsd-capability-changie.git#v0.1.2`; the installed checkout HEAD is `2631063f...`; `gh api .../gsd-capability-changie/commits/v0.1.2` returns the same sha; `CONTRIBUTING.md` lines 233 and 239 name `#v0.1.2`. All four match. |
| 5 | D-03/D-04: follow-up branch hygiene. #88's squash commit is an ancestor of HEAD, and the branch carries only `.planning/` changes | ✓ VERIFIED | `git merge-base --is-ancestor ab49d80e HEAD` and `... 0246de0e HEAD` both true. `git diff --name-only origin/main...HEAD -- ':(exclude).planning/'` prints nothing. Branch `gsd/v0.15.0-changie-release-management` is not on the remote (`git branch -r` finds nothing), so nothing was pushed. |
| 6 | D-06: the branch adds nothing under `.changes/`, so merging it will not open a new release PR | ✓ VERIFIED | `git diff --name-only origin/main...HEAD -- .changes/` prints nothing; `ls -A .changes/unreleased/` shows only `.gitkeep`. |
| 7 | D-10: at most one re-run, and only for a recorded infrastructure/flake cause | ✓ VERIFIED | Every run on the tag commit (`ci`, `release-pr`, `release-tag`, `release`, `post-release-verify`) is `attempt 1`, `success`. The one re-run is ci.yml run 39053062791 on the release PR (the recorded `TestColorFlagInvalidValue` `DB Size:` flake); that is the single permitted re-run, with the cause and log excerpt recorded in `06-EVIDENCE.md`. |
| 8 | D-11/D-12: SHIP-02 and ROADMAP SC2 carry the content-equal wording, and the maintainer signed off the evidence summary once | ✓ VERIFIED | `REQUIREMENTS.md` SHIP-02 and ROADMAP Phase 6 SC2 both read "content-equal ... amended by Phase 6 D-11, 2026-10-17". `06-EVIDENCE.md` ends with `## Evidence summary` (SC1 to SC4 rows, 05-UAT test 3 and 4 links, `D-10 re-runs:`, A1 to A4) and `## Sign-off` quoting the reply `and approved`. The release-PR merge was by `seanb4t`, not the executor. No `06-VERIFICATION.md` existed before this pass. |

**Score:** 8/8 truths verified (0 present, behavior-unverified)

Behavior-dependence: truths 2 and 3 assert runtime behavior (tag authorship, signed-artifact verification on upgrade). They are backed by recorded live workflow runs and an independent CI-side upgrade proof on both platforms, not by symbol presence.

### Must-have prohibitions (judgment tier)

The six plan prohibitions are `verification: judgment`. Per the soft-gate rule this is a non-authoritative judge verdict, flagged `unverified-prohibition: human review recommended`. Autonomous completion therefore reads "complete with 6 flagged prohibitions". The observations behind each verdict are listed so a human can confirm quickly.

| Plan | Prohibition | Verdict (non-authoritative) | Observation |
| ---- | ----------- | --------------------------- | ----------- |
| 06-01 | No manual tag, Release edit, `CHANGELOG.md`/`.changes/` edit, or push to `changie/release` | Not violated | Only the App-token job pushed the tag (actor evidence above). Release `created_at` = PR merge time, `published_at` 17:40:18Z, no later edits visible. Branch diff has no `.changes/` or `CHANGELOG.md` change. |
| 06-01 | No merge/approve/close of any PR, no issue opened, no branch pushed | Not violated | #101 `mergedBy` = `seanb4t`; follow-up branch absent from the remote. |
| 06-01 | No re-run of a deterministic failure, no job re-run more than once | Not violated | See truth 7. |
| 06-02 | No `upgrade` with the real HOME, no Homebrew touch | Not violated, with a note | Recorded manifest hash before/after the proofs was identical (`eb80e383...`). The manifest hash on disk now differs (see Notes). |
| 06-02 | No heading/frontmatter key added to SEED-005, no status/`trigger_when` change | Not violated | Truth 4. |
| 06-02 | No sign-off recorded that the maintainer did not give; no `06-VERIFICATION.md` written by the executor | Not violated | Sign-off quotes the maintainer's reply; no VERIFICATION file preceded this one. The reply itself is a conversational event I cannot re-observe. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `06-EVIDENCE.md` | contains `## SHIP-02 Release body (D-11)` (06-01) and `## Evidence summary` (06-02) | ✓ VERIFIED | Both headings plus `## Sign-off` present; claims spot-checked against live state above. |
| `.planning/seeds/SEED-005-...md` | contains `https://github.com/seanb4t/codegraph-go/releases/tag/v` | ✓ VERIFIED | Present in the appended paragraph. |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| maintainer squash merge of #101 | release-tag.yml tag job | `CHANGELOG.md` path trigger plus App token | WIRED | release-tag run 39054420866 on `0246de0e`, log `release:tag: pushed v0.15.0 at 0246de0e...`. |
| App-pushed tag | release.yml run | push event on `v[0-9]*`, actor is the App bot | WIRED | Run 39054491964, actor/triggering_actor `fzy-release-please[bot]`. |
| `.changes/v0.15.0.md` | GitHub Release body | `--release-notes` | WIRED | `CONTENT-EQUAL 5` re-run by me. |
| v0.14.0 binary | v0.15.0 assets | `codegraph upgrade`, in-process sigstore check | WIRED | CI `self-upgrade proof` jobs log `upgraded to v0.15.0` on both platforms. |
| Evidence summary | maintainer sign-off | blocking-human checkpoint | WIRED | `## Sign-off` quotes the reply. |

### Data-Flow Trace (Level 4)

Not applicable: no code was added by this phase.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Release body equals changie file | `norm` + `cmp` on `gh api releases/tags/v0.15.0 --jq .body` vs `git show 0246de0e:.changes/v0.15.0.md` | `CONTENT-EQUAL 5` | ✓ PASS |
| changie latest on main | `GOTOOLCHAIN=go1.26.9 task changie -- latest` | `v0.15.0` | ✓ PASS |
| Tag points at release commit | `gh api commits/v0.15.0 --jq .sha` | `0246de0e391b7de98052389800f8347fb9657d3b` | ✓ PASS |
| GitHub latest release | `gh api releases/latest --jq .tag_name` | `v0.15.0` | ✓ PASS |
| Cosign identity file unchanged | `git diff --stat v0.14.0 0246de0e -- internal/upgrade/verify.go` | empty | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED. The plans declare no `probe-*.sh`, and the phase is a live-run phase with no migration or tooling code.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| SHIP-01 | 06-01 | Workflow opens the release PR from the fragments; merged; `changie latest` answers the derived version | ✓ SATISFIED | Truth 1 |
| SHIP-02 | 06-01 | App-authored tag, `release.yml` ran on it, Release body content-equal to `.changes/v0.15.0.md` | ✓ SATISFIED | Truth 2 |
| SHIP-03 | 06-02 | `post-release-verify` green; upgrade from v0.14.0 verifies against the unchanged cosign identity on both platforms | ✓ SATISFIED | Truth 3 |
| SHIP-04 | 06-02 | SEED-005 annotated; capability tag recorded | ✓ SATISFIED | Truth 4 |

All four IDs in the plan frontmatter (`06-01`: SHIP-01, SHIP-02; `06-02`: SHIP-03, SHIP-04) appear in `REQUIREMENTS.md`, and the traceability table maps exactly these four to Phase 6. No orphaned requirements.

### Anti-Patterns Found

None. `rg 'TBD|FIXME|XXX'` over `06-EVIDENCE.md` and SEED-005 finds nothing. The phase adds no code.

### Advisory (not blocking)

| # | Finding | Category | Why advisory |
| - | ------- | -------- | ------------ |
| 1 | `06-REVIEW.md` WR-01, WR-02, IN-01, IN-02 (all `open` in `06-REVIEW-DISPOSITION.md`): the v0.15.0 notes' heading/link format deviates from earlier entries, the only link points to the milestone PR #88, the Release body repeats its own title, and "Breaking" may overstate the change | other | They concern the published notes' wording. The goal and SC2 require only that body equals the changie file, which holds. D-09 makes any fix patch-forward through the chain. Triage is a maintainer decision. |

### Deferred Items

None. The 05-UAT tests 3 and 4 deferred to this phase are discharged here (App-opened PR with 9 of 9 required checks; path filter fired, App tag push started `release.yml`, milestone merge fired no `release-tag` run).

### Notes for the maintainer

- **Local manifest changed after the proofs.** `~/.claude/skills/codegraph/.codegraph-manifest.json` now hashes `967dd0a0...` (mtime 14:05 local), not the `eb80e383...` recorded before and after the upgrade proofs. The file now says `codegraph_version: "dev"` and `installed_at: 2026-10-17T18:05:04Z`, about 20 minutes after the proofs (about 17:45Z) and after the sign-off. A v0.14.0-to-v0.15.0 upgrade would not write `dev`. This looks like a separate dev-build `install` run from the session's later gates, not the upgrade proof, so I do not count it against truth 3. It is worth a glance if you did not run an install yourself.
- Housekeeping outside this verification: the ROADMAP phase-list checkbox for Phase 6 (line 154) is still unchecked, and the follow-up branch is unpushed. Both belong to the post-verify and ship steps.
- Gates reported clean by the orchestrator, not re-run by me: post-merge build with `task test:unit` and `task test:daemon`; changie-fragments verify:post (`skip: no-user-visible-change`); Nyquist (`06-VALIDATION.md` validated, compliant); secure-phase (`06-SECURITY.md`, 0 open); TDD gate (no TDD plans).

## Gaps Summary

No gaps. All four roadmap success criteria and the four plan-level contracts that sit behind them hold against live GitHub state. v0.15.0 went through the intended chain: App-opened release PR #101, maintainer merge, App-pushed tag, `release.yml` publish with a content-equal body, a green `post-release-verify`, and a verified v0.14.0-to-v0.15.0 upgrade on both platforms. SEED-005 and the capability tag are recorded. The only open items are advisory (the published notes' format and the local manifest note above).

---

_Verified: 2026-10-17T18:20:00Z_
_Verifier: Claude (gsd-verifier)_
