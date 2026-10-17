---
phase: 05-release-chain-release-please-retirement
verified: 2026-10-16T21:10:00Z
status: passed
score: 5/5 must-haves verified
covered_files: [".changie.yaml", ".github/workflows/ci.yml", ".github/workflows/post-release-verify.yml", ".github/workflows/pr-title.yml", ".github/workflows/release-pr.yml", ".github/workflows/release-tag.yml", ".github/workflows/release.yml", ".github/workflows/require-issue-link.yml", ".goreleaser.yaml", ".planning/phases/05-release-chain-release-please-retirement/05-01-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-01-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-02-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-02-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-03-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-03-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-04-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-04-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-05-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-05-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-06-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-06-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-07-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-07-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-08-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-08-SUMMARY.md", "CONTRIBUTING.md", "Taskfile.yml", "docs/RELEASE-PROCEDURES.md", "docs/RELEASE.md", "internal/upgrade/changie_release_shape_test.go", "internal/upgrade/changie_shape_test.go", "internal/upgrade/goreleaser_shape_test.go", "internal/upgrade/release_workflow_shape_test.go", "internal/upgrade/tag_authority_test.go", "internal/upgrade/taskfile_shape_test.go", "scripts/fragment_policy.py", "scripts/pr_template_policy.py", "scripts/test_fragment_policy.py"]
covered_digest: "v3:sha256:2561fe1941c3bf849093108910cc75c524416edf5cb38cb147227043cd250b54"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  gaps_closed:
    - "G-05-1 / prior human item 1 (IN-03, D-15 rc residue in docs/RELEASE-PROCEDURES.md sections 1, 2, 7) closed by 05-08, commit 5653cbf9"
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "The live release PR is opened by the App on main and its required checks report; a release-PR merge fires the CHANGELOG.md path filter and the App-token tag push starts release.yml; the milestone merge is a no-op"
    addressed_in: "Phase 6"
    evidence: "Phase 6 success criteria 1 and 2 (SHIP-01, SHIP-02): after the milestone PR merges the workflow opens the release PR, the maintainer merges it, and the resulting tag is authored by the GitHub App with release.yml run on it. ROADMAP Phase 5 notes: 'the live proof is Phase 6'."
  - truth: "PR #88 linux-cross-canary run exercises task release:dry-run on the macOS runner class"
    addressed_in: "next push to PR #88 (CI observation, not a human check)"
    evidence: "05-UAT.md Deferred Follow-Ups, test 2. The maintainer ruled this a CI observation that runs on its own and shows red on the PR; release:dry-run exited 0 locally here and in the prior verification."
---

# Phase 5: Release Chain & release-please Retirement Verification Report

**Phase Goal:** release-please is replaced end to end: a changie release workflow batches fragments into a reviewable release PR and, on its merge, pushes the tag with the GitHub App token into the unchanged `release.yml`; every guard that pinned release-please now pins the changie chain; the contributor and release docs describe the new chain; and release-please's files are gone.
**Verified:** 2026-10-16T21:10:00Z
**Status:** passed
**Re-verification:** Yes. After gap-closure plan 05-08 (G-05-1). The prior report had no `gaps:` block (status `human_needed`), so every roadmap success criterion was re-checked against the tree at HEAD `b3739379`, not only the changed file.

## Re-verification focus

Between the prior verified commit (`c3ea98af`) and HEAD the only change outside `.planning/` is `docs/RELEASE-PROCEDURES.md` (`git diff --stat`: 1 file, 9 insertions, 6 deletions, commit `5653cbf9`). No Go, workflow, Taskfile, script or config file moved, so the earlier executed rehearsals (batch, tag, dry-run, six re-mutations) still describe the same code. I re-ran the guard suites and the census rather than carrying those results forward, and I read the changed file rather than the SUMMARY.

SUMMARY.md claims were not used as evidence.

## Goal Achievement

### Observable Truths

| #  | Truth (roadmap contract) | Status | Evidence |
|----|--------------------------|--------|----------|
| SC1 | Release workflow is actionlint-clean, SHA-pinned, job bodies in `Taskfile.yml`. Push to `main` with fragments runs `pretag-gate`, then batch+merge, and opens or updates one `chore(main): release vX.Y.Z` PR; with none it does nothing. A push to `main` that changes `CHANGELOG.md` mints an App token and pushes `vX.Y.Z`, skipping with success if the tag exists, never with `GITHUB_TOKEN` (REL-10, REL-11) | VERIFIED | `.github/workflows/release-pr.yml` and `release-tag.yml` exist; `task lint:actions` (actionlint) exits clean. `TestReleasePRWorkflowShape`, `TestReleaseBatchTargetShape`, `TestReleaseTagWorkflowShape`, `TestReleaseTagTargetShape` all `--- PASS` in this run. `go test -count=1 ./internal/upgrade/` ok. `task check:changie` 14 of 14 passed, including leg 14 (batch auto then merge renders v0.15.0 with every heading at line start, history byte-identical). The earlier scratch rehearsals of `release:batch` (v0.15.0 rendered, empty-set no-op) and `release:tag` (push, skip-if-exists, refuse outside Actions) apply unchanged since no code moved. |
| SC2 | `release.yml` differs from `main` by one comment hunk only; `release:goreleaser` receives `--release-notes .changes/${TAG}.md`; shape test asserts `changelog.disable: true` absent; `release:dry-run` passes the same flag; both refuse a missing or empty notes file; `task release:dry-run` exits 0 before any real tag exists (REL-12, REL-13, amended by D-12 and D-14) | VERIFIED | `git diff 7b12cb92 HEAD -- .github/workflows/release.yml` has exactly 1 hunk (`rg -c '^@@'` = 1). `TestReleaseGoreleaserPassesReleaseNotes` and `TestGoreleaserChangelogNotDisabled` `--- PASS`. Dry-run exit 0 with the evidence line was observed in the prior verification on a scratch tree; `Taskfile.yml` is unchanged since, so the result stands. No body-equality relation asserted (D-14). |
| SC3 | Every test that pinned release-please went RED against the replacement and was re-pointed: pre-major guard reads `.changie.yaml`; job registry names the new `pretag-gate`; forbidden-`release:`-keys rationale restated; `pr_template_policy.py` swaps exemptions for the release PR's own; `TestGsdTagCreationIsDisabled` passes unchanged; a new single-App-token-tag-site assertion goes RED against a planted second tag push; no guard left red (GRD-15, GRD-16) | VERIFIED | `TestChangieBreakingBumpsMinorPreOne`, `TestGsdTagCreationIsDisabled`, `TestTagCreationHasSingleAppTokenSite` `--- PASS` here. `task check:fragment-policy` ran 46 of 46 (floor 46). `05-MUTATION-LOG.md` holds the RED transcripts; the planted `git push origin --tags` RED was reproduced in the prior verification and the guard code is unchanged. Full `./internal/upgrade/` package green, so no guard is red. |
| SC4 | `CONTRIBUTING.md` teaches fragment-per-change, never hand-write `.changes/v*.md`, how to exempt, how the version is derived; `docs/RELEASE-PROCEDURES.md` describes fragment, release PR, merge, App-token tag, locked `release.yml`, places `pretag-gate` in the new workflows, states a release is cut by merging the release PR, 45 release-please references resolved under a recorded verdict; `pr-title.yml` header is hygiene only; `release:dry-run-signed` comment and `docs/RELEASE.md` paragraph name the changie workflow (DOCS-12, DOCS-13, DOCS-14) | VERIFIED | Read in this run: runbook section 1 note names `pretag-gate` in both `release-pr.yml` and `release-tag.yml`; section 4 holds "A release is cut by merging the release PR" and the chain. `05-CENSUS.md` holds the 45 verdict rows. The doc has zero release-please text (census below). **The prior D-15 residue is closed in the file** (see "G-05-1 closure check"). The `CONTRIBUTING.md`, `pr-title.yml`, `Taskfile.yml` and `docs/RELEASE.md` content is unchanged since the prior verification; `TestContributingReferencesRealTaskTargets` is inside the passing package. |
| SC5 | As the phase's last change, `release-please.yml`, `release-please-config.json` and `.release-please-manifest.json` are deleted, `changie latest` is the version source of truth, and a positive-controlled census reports zero references outside the three exclusions (REL-14) | VERIFIED (see ordering note) | All three paths absent (`ls` errors on each). Commit `ffd84c4a` deletes exactly those three files plus the `taskfile_shape_test.go` registry entries. `task changie -- latest` = `v0.14.1`. Census re-run on a fresh `git archive HEAD` copy with the recorded instrument: base prints nothing; after planting `Release Please` in `docs/RELEASE-PROCEDURES.md` and a file `scripts/__release_please_control__.txt`, it prints exactly those two (content half and filename half). Ordering note below. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### G-05-1 closure check (the 05-08 delta, read from the file)

| Required by 05-UAT.md "missing" list / 05-08 must_haves | Evidence at HEAD | Status |
|---|---|---|
| Section 2 intro says two shapes; prerelease row removed; table has exactly Stable and Internal milestone marker | `docs/RELEASE-PROCEDURES.md` line 44 "Two distinct shapes:"; table rows are `Stable` and `Internal milestone marker` only | MET |
| Section 2 pointer to section 4 naming D-15, no example prerelease tag | Lines 51-54: "There is no hand-pushed prerelease tag shape (D-15). Every `v[0-9]*` tag is pushed by `release-tag.yml` ... See §4, "No hand-pushed prerelease tag (D-15)"." Section 2 contains no `rc.N` and no "release candidate" | MET |
| Section 1 sentence drops the prerelease kind | Line 13 "Before tagging **anything**, run the following for **all 4** release targets." | MET |
| Section 7 prerelease-is-transient sentence removed, stable sentence kept | Lines 340-343 end after "...what the tag now points to."; stable-release sentence byte-intact | MET |
| Section 4 D-15 paragraph untouched | Paragraph extracted from HEAD and from `1c700149` with the plan's `awk`: identical (`D15-PARA-IDENTICAL`) | MET |
| Line-15 historical `rc.1` failure sentence and `Taskfile.yml` unchanged | Line 15 still reads "`rc.1` failure"; `git diff --quiet 1c700149 -- Taskfile.yml` clean | MET |
| One docs-only commit, no skip-CI phrase | `5653cbf9` touches only `docs/RELEASE-PROCEDURES.md`; subject `docs(05-08): ...`; zero skip-CI matches | MET |
| Test reading this file still passes, PASS line asserted | `--- PASS: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (0.00s)` seen in this run | MET |

Remaining `rc` and `prerelease` mentions in the file were checked line by line: line 15 (historical v0.1 failure), lines 147-148 (the GoReleaser `prerelease: auto` flag), lines 176-181 (the correct D-15 paragraph), line 288 and 296 (v0.5.0's GitHub `prerelease` flag history). None presents a hand-pushed prerelease tag as a working shape. The section 2 table no longer lists one, so a reader following section 2 can no longer be led to push a tag that `release:goreleaser` would then refuse.

### Ordering note on SC5 ("as the phase's last change")

`ffd84c4a` (the deletion) is no longer the last commit outside `.planning/`: `5653cbf9` (05-08, docs only) follows it. This is a literal reading of 05-07's must-have, not the roadmap's intent. The reason the deletion must come last is that nothing pinning the retired tool may still be live when it goes, and no later change may re-introduce a reference. 05-08 edits one runbook, adds no release-please text (census empty at HEAD), changes no guard, and the whole `internal/upgrade` package passes. I treat this as satisfied and record it so the maintainer can see it; if the maintainer wants the literal ordering enforced, an `overrides:` entry would be the place, but none is needed for the goal.

### Decisions D-01..D-15 spot-check

| Decision | Honored? | Evidence |
|----------|----------|----------|
| D-01 documented pattern | Yes | `peter-evans/create-pull-request` + `actions/create-github-app-token`; changie via `task changie` |
| D-05 hygiene by `.changes/` path in both gates | Yes | `check:fragment-policy` 46/46 includes the mirror test |
| D-06/D-07 `CHANGELOG.md` path trigger, `pretag-gate` gates both jobs | Yes | shape tests pass |
| D-08/D-12 flag in `release:goreleaser`, `release.yml` one comment hunk | Yes | 1 hunk vs `7b12cb92` |
| D-13 `.changie.yaml` newlines amendment | Yes | `check:changie` leg 14 |
| D-14 wiring only | Yes | no body-equality assertion |
| D-15 hand-pushed rc escape hatch removed | **Yes (now fully)** | section 4 paragraph plus sections 1, 2, 7 aligned by 05-08; prior "Partly" resolved |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/release-pr.yml` | REL-10 workflow | VERIFIED | present, shape test passes, actionlint clean |
| `.github/workflows/release-tag.yml` | REL-11 workflow | VERIFIED | present, shape test passes |
| `Taskfile.yml` `release:batch`, `release:tag`, `release:goreleaser`, `release:dry-run` | job bodies | VERIFIED | target-shape tests pass; file unchanged since prior executed rehearsals |
| `.changie.yaml`, `.changes/` seeds | config, history | VERIFIED | `check:changie` 14/14 |
| `internal/upgrade/changie_release_shape_test.go` and sibling guards | re-pointed guards | VERIFIED | package green |
| `scripts/pr_template_policy.py`, `fragment_policy.py` | release PR exemption | VERIFIED | 46/46 |
| `docs/RELEASE-PROCEDURES.md`, `CONTRIBUTING.md`, `docs/RELEASE.md`, `pr-title.yml` | docs | VERIFIED | read; G-05-1 closed |
| `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json` | deleted | VERIFIED (absent) | |
| `05-CENSUS.md`, `05-MUTATION-LOG.md` | verdicts and RED transcripts | VERIFIED | present; 45 verdict rows |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `release-pr.yml` batch step | `Taskfile.yml` `release:batch` | `run: task release:batch`, output `version` | WIRED | shape test |
| `release-tag.yml` tag step | `Taskfile.yml` `release:tag` | `run: task release:tag`, App-token checkout | WIRED | shape test; GRD-16 scan names this single site |
| `release.yml` | `release:goreleaser` | single `run: task release:goreleaser` | WIRED | unchanged |
| `release:goreleaser` / `release:dry-run` | `.changes/${TAG}.md` | `--release-notes` | WIRED | target tests |
| `pr_template_policy.py` allowlist | `require-issue-link.yml` `case` | mirror test | WIRED | in 46/46 |
| runbook section 2 | runbook section 4 D-15 paragraph | pointer sentence | WIRED | lines 51-54 name D-15 and §4; target paragraph exists at line 176 |

### Data-Flow Trace (Level 4)

Unchanged from the prior report: PR title/body flow from `changie latest` and `changie batch/merge` over real fragments; the tag flows from `changie latest`; the release body path flows from `.changes/${TAG}.md` to GoReleaser. No code moved, and `check:changie` leg 14 re-ran the batch+merge path this run. FLOWING.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Release-chain and guard tests | `GOTOOLCHAIN=go1.26.9 go test -count=1 ./internal/upgrade/` | ok | PASS |
| Ten named guard tests | `go test -v -run '<names>' ./internal/upgrade/` | 10 `--- PASS`, 0 FAIL | PASS |
| Fragment policy | `task check:fragment-policy` | 46 of 46 (floor 46) | PASS |
| Changie guard | `task check:changie` | 14 of 14 | PASS |
| Workflow lint | `task lint:actions` | exit 0 | PASS |
| `release.yml` lock | `git diff 7b12cb92 HEAD -- .github/workflows/release.yml` | 1 hunk | PASS |
| Version source | `task changie -- latest` | `v0.14.1` | PASS |
| Census, base and positive control | recorded instrument on fresh `git archive HEAD` | base empty; planted content and filename both found | PASS |
| Plan 05-08 invariants | D-15 paragraph diff vs `1c700149`; Taskfile diff; commit file list; skip-CI scan | all clean | PASS |

I did not re-run the full workspace suite or `task release:dry-run`. The orchestrator's regression gate (54/54 packages; `internal/daemon` isolated pass of a known load flake) covers the suite, and the dry-run inputs are unchanged since the prior run.

### Probe Execution

SKIPPED. The phase declares no `probe-*.sh` probes.

### Requirements Coverage

Every ID in the PLAN frontmatter (05-01 REL-10, GRD-15; 05-02 REL-11, GRD-16, GRD-15; 05-03 REL-12, REL-13, GRD-15, DOCS-14; 05-04 GRD-15; 05-05 GRD-15, GRD-16; 05-06 DOCS-12, DOCS-13, DOCS-14; 05-07 REL-14, GRD-15; 05-08 DOCS-13) was cross-referenced against `.planning/REQUIREMENTS.md`. All ten phase IDs are present in plans and mapped to Phase 5 in the Traceability table. No requirement mapped to Phase 5 is missing from the plans (no orphans). REQUIREMENTS.md now shows all ten `[x]` and "Complete", including GRD-16 (the prior bookkeeping gap is closed).

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| REL-10 | 05-01 | release-PR workflow behind `pretag-gate` | SATISFIED | SC1 |
| REL-11 | 05-02 | App-token tag on `CHANGELOG.md` push, skip if exists | SATISFIED | SC1 |
| REL-12 | 05-03 | `release.yml` locked; `--release-notes`; no `changelog.disable` | SATISFIED | SC2 |
| REL-13 | 05-03 | notes flag in both targets, empty-file refusal, dry run exits 0 | SATISFIED | SC2 |
| REL-14 | 05-07 | three files deleted, positive-controlled census zero | SATISFIED | SC5 |
| GRD-15 | 05-01, 05-03, 05-04, 05-05, 05-07 | guards re-pointed, RED first | SATISFIED | SC3 |
| GRD-16 | 05-02, 05-05 | tag authority unchanged; single App-token tag site, RED against planted push | SATISFIED | SC3 |
| DOCS-12 | 05-06 | CONTRIBUTING rewrite | SATISFIED | SC4 |
| DOCS-13 | 05-06, 05-08 | runbook rewrite, 45 verdicts, D-15 consistency | SATISFIED | SC4 and G-05-1 closure |
| DOCS-14 | 05-06, 05-03 | pr-title header, dry-run-signed comment, RELEASE.md paragraph | SATISFIED | SC4 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `docs/RELEASE-PROCEDURES.md` (05-08 delta) | - | `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` scan of added lines | none found | No debt markers |

### Advisory (open review findings, not blocking)

`05-REVIEW-DISPOSITION.md` lists all 10 findings (0 critical, 5 warning, 5 info) as `open`; none has a recorded `fixed`, `skipped` or `deferred` decision. That file is tool-owned and I did not edit it. They are documentation-quality and hardening follow-ups, not success-criterion failures.

**WR-02 judged against DOCS-13 and the phase goal: advisory doc follow-up, not a gap.** The review argues that 05-08's removal of the `-rc.N` retag sentence leaves the section 9 live-proof teardown (delete release and tag, then revert) colliding with the "never reuse a stable tag name" rule, because the revert makes `release-pr.yml` reopen a release PR for the same derived version. I agree the collision is real, but:

- DOCS-13 requires that the runbook describe the chain end to end, place `pretag-gate` in the new workflows, state that a release is cut by merging the release PR, and resolve the 45 references under recorded verdicts. All four hold. It does not require that the rollback procedure be free of a tie between two rules.
- The sentence 05-08 removed was the contradiction G-05-1 and D-15 required removing. The collision exists because D-15 removed the only transient tag shape, not because of how 05-08 edited it.
- The runbook already scopes the rule: section 7 (lines 261-278) says the delete-and-retry procedure "applies only to a disposable cut (§9's live proof) — never to a real `vX.Y.Z`", and the "never reuse" paragraph is explicitly about **stable** releases. The unresolved question is which rule wins for the live-proof cut, and that question only becomes operational in Phase 6, which runs the live proof.
- It involves a maintainer choice (carve-out versus requiring a different next version), so I did not invent an answer, and it is not a thing a test can prove wrong.

Recommended handling: disposition WR-02 as `deferred` to Phase 6 entry (or `fixed` with the review's suggested carve-out) before the Phase 6 live proof is run. It is recorded here and in the review disposition; it does not change this phase's verdict.

### Human Verification Required

None. The two items in the prior report are resolved or reclassified:

1. D-15 rc residue: closed by 05-08 and checked against the file above.
2. linux-cross-canary run on PR #88: the maintainer ruled this a deferred CI observation, not a human check. It is recorded under `deferred:` and in `05-UAT.md` Deferred Follow-Ups, and is deliberately not re-created as a human verification item.

The live behavior of the chain (App-authored release PR with required checks reporting, the `CHANGELOG.md` path filter firing on a real merge, the App-token tag starting `release.yml`, the milestone merge being a no-op) is Phase 6 by design and is listed under `deferred:`; it is not counted as a gap here.

### Bookkeeping for the orchestrator (not a code gap)

`.planning/ROADMAP.md` still shows the Phase 5 progress line as `- [ ] **Phase 5: ...**` (line 153) although the plan list reads 8/8 executed. Flip it through the normal phase-complete path.

### Gaps Summary

No success criterion failed and there are no blockers. The goal is achieved in the codebase: release-please's three files are gone, the changie release-PR and tag workflows exist and are pinned by passing shape tests, `release.yml` is held to one comment hunk, the guards that pinned release-please now pin the changie chain, and the contributor and release docs describe the new chain. The one defect the prior report held open, the runbook still presenting a hand-pushed prerelease tag in sections 1, 2 and 7, is fixed in the file and consistent with D-15.

---

_Verified: 2026-10-16T21:10:00Z_
_Verifier: Claude (gsd-verifier)_
