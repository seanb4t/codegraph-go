---
phase: 05-release-chain-release-please-retirement
verified: 2026-10-14T23:30:00Z
status: human_needed
score: 5/5 must-haves verified
covered_files: [".changie.yaml", ".github/workflows/ci.yml", ".github/workflows/post-release-verify.yml", ".github/workflows/pr-title.yml", ".github/workflows/release-pr.yml", ".github/workflows/release-tag.yml", ".github/workflows/release.yml", ".github/workflows/require-issue-link.yml", ".goreleaser.yaml", ".planning/phases/05-release-chain-release-please-retirement/05-01-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-01-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-02-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-02-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-03-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-03-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-04-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-04-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-05-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-05-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-06-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-06-SUMMARY.md", ".planning/phases/05-release-chain-release-please-retirement/05-07-PLAN.md", ".planning/phases/05-release-chain-release-please-retirement/05-07-SUMMARY.md", "CONTRIBUTING.md", "Taskfile.yml", "docs/RELEASE-PROCEDURES.md", "docs/RELEASE.md", "internal/upgrade/changie_release_shape_test.go", "internal/upgrade/changie_shape_test.go", "internal/upgrade/goreleaser_shape_test.go", "internal/upgrade/release_workflow_shape_test.go", "internal/upgrade/tag_authority_test.go", "internal/upgrade/taskfile_shape_test.go", "scripts/fragment_policy.py", "scripts/pr_template_policy.py", "scripts/test_fragment_policy.py"]
covered_digest: "v3:sha256:c3f90bf9eb5b4bd4b1826be9f1dfb5baa877b691e050652357189c5e1d0543e5"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Decide the fate of the stale release-candidate row in docs/RELEASE-PROCEDURES.md section 2 (IN-03): fix it now (one table row, plus the two rc mentions at lines 13 and 339) or accept it as a known residue"
    expected: "Section 2 no longer presents a hand-pushed `v0.0.0-rc.N` tag as a normal, working tag shape, consistent with D-15 and with section 4's 'No hand-pushed prerelease tag' paragraph"
    why_human: "D-15 is a maintainer decision and the residue is a judgment call on scope (fix inside Phase 5 versus defer); a doc row cannot be proven wrong by a test"
  - test: "After the maintainer's next push to PR #88, read the linux-cross-canary run (its cross-build job runs `task release:dry-run` on the macOS runner class) using the read-only commands recorded in 05-03-SUMMARY.md"
    expected: "The cross-build job exits 0 with the RELEASE-NOTES-EVIDENCE line for the newest .changes/v*.md and the four-binary success line, and the linux binaries run on linux/amd64 and linux/arm64"
    why_human: "Needs a push this verification may not make; it is external CI on a runner class that is not available locally. The same target was re-run locally here and exited 0, so this confirms the CI runner class only"
---

# Phase 5: Release Chain & release-please Retirement Verification Report

**Phase Goal:** release-please is replaced end to end: a changie release workflow batches fragments into a reviewable release PR and, on its merge, pushes the tag with the GitHub App token into the unchanged `release.yml`; every guard that pinned release-please now pins the changie chain; the contributor and release docs describe the new chain; and release-please's files are gone.
**Verified:** 2026-10-14T23:30:00Z
**Status:** human_needed
**Re-verification:** No, initial verification

All five roadmap success criteria hold in the codebase. Two items need a human: one documentation residue that contradicts decision D-15, and one CI run that needs a push. Neither falsifies a success criterion. Live behavior of the chain (the merge push firing the `CHANGELOG.md` path filter, the App-token tag starting `release.yml`) is Phase 6 by design and is not counted as a gap here.

SUMMARY.md claims were not used as evidence. Every row below was checked against the tree at HEAD `c3ea98af`, and the load-bearing guards were re-mutated on scratch copies (the real tree was never written, pushed or tagged).

## Goal Achievement

### Observable Truths

| #  | Truth (roadmap contract) | Status | Evidence |
|----|--------------------------|--------|----------|
| SC1 | Release workflow is actionlint-clean, SHA-pinned, job bodies in `Taskfile.yml`. On push to `main` with fragments it runs `pretag-gate` then batch+merge and opens or updates one `chore(main): release vX.Y.Z` PR; with none it does nothing. On a push that changes `CHANGELOG.md` it mints an App token and pushes `vX.Y.Z`, skipping with success if the tag exists, never with `GITHUB_TOKEN` (REL-10, REL-11) | VERIFIED | `.github/workflows/release-pr.yml` and `release-tag.yml` read in full: both `pretag-gate` -> job via `needs:`, `pretag-gate` runs `task check:cross`; `release-pr` uses `peter-evans/create-pull-request@5f6978fa… # v8.1.1` with `token: steps.app-token.outputs.token`, branch `changie/release`, title `chore(main): release <version>`, `body-path: .changes/<version>.md`, `draft: false`; `release-tag.yml` triggers on `push: branches:[main] paths:[CHANGELOG.md]`, checkout `token` is the App-token output, tag step is `task release:tag`. `actionlint` exit 0 over all workflows. I ran `task release:batch` on a scratch copy of HEAD: prints `rendered v0.15.0 from 1 fragment(s)`, writes `version=v0.15.0` to `GITHUB_OUTPUT`, renders `.changes/v0.15.0.md`, merges `CHANGELOG.md` with the next `## [` heading at line start, and `go test ./internal/upgrade/` is green on that post-batch tree (the Pitfall 1 trap). With zero fragments it printed the no-op line, exit 0, empty `GITHUB_OUTPUT`. I ran `task release:tag` against a scratch bare origin: run 1 pushed `v0.15.0` (`[new tag]`), run 2 printed "already exists on origin — nothing to do" exit 0, run 3 without `GITHUB_ACTIONS=true` was refused by the precondition. Shape tests `TestReleasePRWorkflowShape`, `TestReleaseBatchTargetShape`, `TestReleaseTagWorkflowShape`, `TestReleaseTagTargetShape` pass. Re-mutated: tag-job checkout switched to `github.token` -> `TestReleaseTagWorkflowShape` and `TestTagCreationHasSingleAppTokenSite` FAIL; `paths: [CHANGELOG.md]` deleted -> `TestReleaseTagWorkflowShape` FAILS. |
| SC2 | `release.yml` differs from `main` by one comment hunk only; `release:goreleaser` receives `--release-notes .changes/${TAG}.md`; shape test asserts `changelog.disable: true` absent; `release:dry-run` passes the same flag for the tag being released; both refuse a missing or empty notes file; a `task release:dry-run` run exits 0 before any real tag exists (REL-12, REL-13, amended by D-12 and D-14) | VERIFIED | `git diff 7b12cb92 HEAD -- .github/workflows/release.yml` is exactly one hunk of comment lines near line 188 (names `release-pr.yml and release-tag.yml`); trigger and steps untouched. `Taskfile.yml` `release:goreleaser` derives `TAG` from `git describe --exact-match --tags HEAD`, `NOTES=".changes/${TAG}.md"`, `[ ! -s "${NOTES}" ]` -> `::error::` exit 1, then `release --clean --release-notes "${NOTES}"`. `release:dry-run` derives `TAG` from `changie latest`, same refusal, prints `RELEASE-NOTES-EVIDENCE`, runs `GORELEASER_CURRENT_TAG=… release --skip=publish,sign,validate --clean --release-notes`, no `--snapshot`. I ran `task release:dry-run` on a scratch copy of the post-batch tree on this darwin host (zig, syft, cosign present): exit 0 in 28 s, `RELEASE-NOTES-EVIDENCE tag=v0.15.0 file=.changes/v0.15.0.md bytes=328 sha256=3dca5a9b…`, four correctly typed binaries, `rc=0`. Re-mutated: `--release-notes` dropped from `release:goreleaser` -> `TestReleaseGoreleaserPassesReleaseNotes` FAILS; `changelog: {disable: true}` appended to `.goreleaser.yaml` -> `TestGoreleaserChangelogNotDisabled` FAILS. No body-equality relation is asserted anywhere (D-14 honored). |
| SC3 | Every test that pinned release-please went RED against the replacement and was re-pointed: pre-major guard reads `.changie.yaml`; Taskfile job registry names the new `pretag-gate`; forbidden-`release:`-keys rationale restated; `pr_template_policy.py` swaps exemptions for the release PR's own; `TestGsdTagCreationIsDisabled` passes unchanged; a new assertion that the App-token step is the only tag site goes RED against a planted second tag push (GRD-15, GRD-16) | VERIFIED | `TestChangieBreakingBumpsMinorPreOne` (renamed from the release-please name) reads `.changie.yaml`; re-mutated Breaking `auto: minor` -> `major` -> `TestChangieConfigShape` and `TestChangieBreakingBumpsMinorPreOne` FAIL. `taskfile_shape_test.go:153-156` registers `release-pr.yml` and `release-tag.yml` `pretag-gate`, `inScopeWorkflowFiles` lists both. `goreleaser_shape_test.go` `TestReleaseBlockDoesNotRewriteReleaseBody` restated for `.changes/${TAG}.md`. `pr_template_policy.py` `EXEMPT_PREFIXES` has `.changes/`, `require-issue-link.yml` has the mirrored `.changes/*)` arm, and `test_hygiene_allowlists_mirror_require_issue_link` holds the pair; `CHANGELOG.md` stays in the exact set, the release-please filenames are gone. `TestGsdTagCreationIsDisabled` passes; its code tokens are unchanged, only prose reworded under D-09. GRD-16: `TestTagCreationHasSingleAppTokenSite` passes (17 workflow files of 17 on disk, 84 run bodies, 72 Taskfile targets, 2 tag-creating lines in 1 unit, `release:tag` invoked from `release-tag.yml/tag/Tag the release`). I planted `run: git push origin --tags` in a scratch `ci.yml`: the test FAILED naming `workflow:ci.yml/test/Build`, and passed again after reverting. `TestTagCreationScanFlagsPlantedSites` also passes. No guard is red: `go test ./internal/upgrade/` ok, `task check:fragment-policy` ran 46 of 46 (floor 46), `task check:changie` 14 of 14. |
| SC4 | `CONTRIBUTING.md` teaches fragment-per-change, never hand-write `.changes/v*.md`, how to exempt, how the version is derived; `docs/RELEASE-PROCEDURES.md` describes fragment -> release PR -> merge -> App-token tag -> locked `release.yml`, places `pretag-gate` in the new workflows, states a release is cut by merging the release PR, with 45 release-please references resolved under a recorded verdict; `pr-title.yml` header is hygiene only; `release:dry-run-signed` comment and `docs/RELEASE.md` pipeline paragraph name the changie workflow (DOCS-12, DOCS-13, DOCS-14) | VERIFIED (one residue, see Warnings) | `CONTRIBUTING.md` Pull requests section (lines 133-196) covers all four points, including the `<!-- changelog-exempt: <reason> -->` marker and "Never hand-write or edit `.changes/v*.md`". `docs/RELEASE-PROCEDURES.md` section 4 line 79 "A release is cut by merging the release PR" and the chain line; section 1 note names `pretag-gate` in both workflows. `05-CENSUS.md` holds 45 verdict rows R01-R45, each `rewritten`, `removed` or `reworded-history`, each with a resolution; the doc itself has zero release-please text (census empty). `pr-title.yml` lines 1-6 say "Hygiene only … carries no versioning weight". `Taskfile.yml` `release:dry-run-signed` desc names `release-pr.yml`, `release-tag.yml`, `release.yml`. `docs/RELEASE.md` lines 88-91 name the changie release workflow. |
| SC5 | As the phase's last change, `release-please.yml`, `release-please-config.json` and `.release-please-manifest.json` are deleted, `changie latest` is the version source of truth, and a positive-controlled census reports zero references outside the three exclusions (REL-14) | VERIFIED | All three files absent (`ls` fails on each); commit `ffd84c4a` deletes them and follows all docs commits. `task changie -- latest` = `v0.14.1`. I re-ran the census on a fresh `git archive HEAD` copy: base prints nothing. Positive control in that copy: a planted line in `.changes/header.tpl.md` and a file named `release_please.txt` were both found (content half and filename half); a planted line in `.changes/unreleased/` was correctly not found. Boundary shown in both directions (D-11). The only remaining mentions are `CHANGELOG.md` (2) and `.changes/v0.2.0.md` (2), the excluded history. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Decisions D-01..D-15 spot-check

| Decision | Honored? | Evidence |
|----------|----------|----------|
| D-01 documented pattern, no space shuttle | Yes | `peter-evans/create-pull-request` + `actions/create-github-app-token`; changie run through `task changie`, no `changie-action` |
| D-02/D-03/D-04 | Yes | PR via App token, ready not draft, body is the rendered file |
| D-05 hygiene by `.changes/` path in both gates | Yes | Python set and shell `case` mirrored, mirror test passes |
| D-06 `CHANGELOG.md` path trigger, skip if tag exists | Yes | `release-tag.yml` `paths`, `release:tag` ls-remote branch, REL-11/SC1 amended |
| D-07 `pretag-gate` gates both jobs | Yes | both workflows |
| D-08/D-12 flag lives in `release:goreleaser` | Yes | `release.yml` one comment hunk |
| D-09/D-10/D-11 census | Yes | see SC5; 45 verdicts recorded |
| D-13 `.changie.yaml` newlines amendment | Yes | `newlines` has 4 keys; leg 14 and post-batch Go tests pass |
| D-14 wiring only | Yes | no body-equality assertion |
| D-15 rc escape hatch removed | Partly | procedure removed (section 4 line 172), but section 2 table row and two other rc mentions remain; see Warnings |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.github/workflows/release-pr.yml` | REL-10 release-PR workflow | VERIFIED | 98 lines, wired to `task release:batch`, registered in shape tests |
| `.github/workflows/release-tag.yml` | REL-11 tag workflow | VERIFIED | 90 lines, wired to `task release:tag`, path-filtered |
| `Taskfile.yml` `release:batch`, `release:tag`, `release:goreleaser`, `release:dry-run` | Job bodies | VERIFIED | executed in scratch (batch, tag, dry-run); goreleaser body read and mutation-tested |
| `.changie.yaml` | Kinds, `newlines`, Breaking -> minor | VERIFIED | read in full |
| `internal/upgrade/changie_release_shape_test.go` | Shape tests and GRD-16 scan | VERIFIED | 1687 lines, passes, RED reproduced |
| `scripts/pr_template_policy.py`, `fragment_policy.py` | Release PR exemption, linear markers (WR-01) | VERIFIED | `task check:fragment-policy` 46/46 |
| `docs/RELEASE-PROCEDURES.md`, `CONTRIBUTING.md`, `docs/RELEASE.md`, `pr-title.yml` | Docs | VERIFIED | see SC4 |
| `release-please.yml`, `release-please-config.json`, `.release-please-manifest.json` | Deleted | VERIFIED (absent) | |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `release-pr.yml` batch step | `Taskfile.yml` `release:batch` | `run: task release:batch`, output `version` | WIRED | `steps.batch.outputs.version` gates the PR step; `GITHUB_OUTPUT` write exercised |
| `release-tag.yml` tag step | `Taskfile.yml` `release:tag` | `run: task release:tag` with App-token checkout | WIRED | exercised against a scratch bare origin |
| `release.yml` | `release:goreleaser` | single `run: task release:goreleaser` | WIRED | unchanged step, flag lives in the target |
| `release:goreleaser` / `release:dry-run` | `.changes/${TAG}.md` | `--release-notes` | WIRED | dry-run exited 0 on a scratch tree |
| `pr_template_policy.py` allowlist | `require-issue-link.yml` `case` | mirror test | WIRED | `test_hygiene_allowlists_mirror_require_issue_link` |
| release PR | required checks | App-token authorship | WIRED by construction | live check is Phase 6 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| PR title/body | `steps.batch.outputs.version`, `.changes/<version>.md` | `changie latest` + `changie batch/merge` over real fragments | Yes (executed: v0.15.0 from the real Breaking fragment) | FLOWING |
| tag | `changie latest` | `.changes/*.md` | Yes (executed) | FLOWING |
| release body | `.changes/${TAG}.md` | passed to GoReleaser | Yes (dry-run evidence line: 328 bytes) | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Release-chain and guard tests | `go test -count=1 ./internal/upgrade/` | ok | PASS |
| Build and CLI | `go build ./...`; `go test -count=1 ./internal/cli/` | ok; ok | PASS |
| Fragment policy | `task check:fragment-policy` | 46 of 46 executed (floor 46) | PASS |
| Changie guard | `task check:changie` | 14 of 14 | PASS |
| Workflow lint | `actionlint`; `task lint:actions` | exit 0 | PASS |
| CLI reference drift | `task docs:cli:drift` | byte-identical | PASS |
| Batch end to end (scratch) | `task release:batch` | v0.15.0 rendered; post-batch Go tests ok | PASS |
| Empty-set no-op (scratch) | `task release:batch` with no fragments | notice, rc 0, no output | PASS |
| Tag rehearsal (scratch) | `GITHUB_ACTIONS=true task release:tag` x2, then without | push, skip, refuse | PASS |
| Dry run (scratch) | `task release:dry-run` | rc 0, 4 binaries, evidence line | PASS |
| Six re-mutations (scratch) | planted tag push; `--release-notes` dropped; `changelog.disable`; App token -> `github.token`; path filter dropped; Breaking -> major | each FAILS a named test; clean tree passes | PASS (all RED) |

I did not re-run the full workspace suite. The orchestrator's six post-wave gates and the regression gate cover it, and `internal/daemon` is unchanged since `f109a9d1` (its documented flake is handled by `task test:daemon`).

### Probe Execution

SKIPPED. The phase declares no `probe-*.sh` probes.

### Requirements Coverage

Every ID in the PLAN frontmatter was cross-referenced against `.planning/REQUIREMENTS.md`. All ten IDs appear in plans and in the Traceability table mapped to Phase 5. No requirement mapped to Phase 5 is missing from the plans, so there are no orphans.

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| REL-10 | 05-01 | release-PR workflow with `pretag-gate` and batch/merge | SATISFIED | SC1 |
| REL-11 | 05-02 | App-token tag on `CHANGELOG.md` push, skip if exists | SATISFIED | SC1; wording amended per D-06 |
| REL-12 | 05-03 | `release.yml` locked; `--release-notes`; no `changelog.disable` | SATISFIED | SC2; amended per D-12 |
| REL-13 | 05-03 | notes flag in both targets, empty-file refusal, captured dry run | SATISFIED | SC2; amended per D-14; dry run re-run here, rc 0 |
| REL-14 | 05-07 | three files deleted, positive-controlled census zero | SATISFIED | SC5 |
| GRD-15 | 05-01, 05-03, 05-04, 05-05, 05-07 | guards re-pointed, RED first | SATISFIED | SC3 |
| GRD-16 | 05-02, 05-05 | `TestGsdTagCreationIsDisabled` unchanged; single App-token tag site, RED against planted push | **SATISFIED in the codebase, but REQUIREMENTS.md still shows `[ ]` and "Pending"** | SC3. Both halves exist and pass; the planted-second-tag RED was reproduced here. No plan claimed mark-complete because the assertion (05-02) and the unchanged-test prose (05-05) landed in two plans. The orchestrator should flip the checkbox on the GRD-16 line and the Traceability row to Complete. |
| DOCS-12 | 05-06 | CONTRIBUTING rewrite | SATISFIED | SC4 |
| DOCS-13 | 05-06 | runbook rewrite, 45 verdicts | SATISFIED | SC4; see Warnings for the rc row |
| DOCS-14 | 05-06, 05-03 | pr-title header, dry-run-signed comment, RELEASE.md paragraph | SATISFIED | SC4 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (phase-modified files) | - | `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` scan of the added lines | none found | No debt markers |

### Warnings (non-blocking)

1. **D-15 residue (IN-03, confirmed).** `docs/RELEASE-PROCEDURES.md` section 2 line 48 still lists `Release candidate | v0.0.0-rc.3 | … | Triggers release.yml? Yes` as a normal tag shape, line 13 says "Before tagging anything (rc or stable)", and line 339 says "Prereleases (-rc.N) are safer to delete/retag". Section 4 line 172 says no hand-pushed prerelease tag exists. The doc therefore contradicts itself, and a reader following section 2 would push a tag that starts the macOS signing job and then fails in `release:goreleaser` for want of `.changes/<tag>.md`. This does not falsify DOCS-13 or SC4, which are met, but D-15 asked for the escape hatch to be removed from the runbook and the table row is its remaining trace. Routed to the human decision above.
2. **Untriaged review findings.** `05-REVIEW-DISPOSITION.md` shows all 8 findings `open` (0 critical). WR-01 (a stale open release PR when the fragment count drops to zero) and WR-02/WR-03 (pre-existing hygiene-gate weaknesses: 100-file truncation; unescaped reason on stderr) have no recorded `fixed`, `skipped` or `deferred` decision. None breaks a success criterion: REL-10's "does nothing when there are no fragments" holds literally, and the race in WR-01 is narrow. They should be dispositioned before the milestone closes.
3. **Roadmap and state bookkeeping not yet reconciled** (orchestrator action, not a code gap): Phase 5 shows `[ ]` and "In Progress" in `ROADMAP.md` and GRD-16 is Pending in `REQUIREMENTS.md`.

### Human Verification Required

#### 1. Release-candidate row in the runbook (D-15 residue)

**Test:** Read `docs/RELEASE-PROCEDURES.md` section 2 (line 48), line 13 and line 339 against section 4's D-15 paragraph, then fix or accept.
**Expected:** The runbook no longer presents a hand-pushed rc tag as a working shape.
**Why human:** A maintainer decision (D-15) and a scope call; a table row cannot be proven wrong by a test.

#### 2. linux-cross-canary run on PR #88

**Test:** After the maintainer's next push to PR #88, read the `linux-cross-canary` run with the read-only commands in `05-03-SUMMARY.md`.
**Expected:** The cross-build job runs `task release:dry-run` to exit 0, prints the `RELEASE-NOTES-EVIDENCE` line and the four-binary success line, and the linux binaries run on linux/amd64 and linux/arm64.
**Why human:** Needs a push and an external macOS runner class. The same target already passed locally here, so this confirms the CI runner only.

### Gaps Summary

No success criterion failed and there are no blockers. The goal is achieved in the codebase: release-please is gone, the changie release-PR and tag workflows exist and their repository-owned bodies run correctly in scratch rehearsals, the guards that pinned release-please now pin the changie chain and each fires RED against a fresh mutation, and the docs describe the new chain.

GRD-16 is satisfied. The single-tag-site assertion `TestTagCreationHasSingleAppTokenSite` exists, passes, and fails when a second `git push --tags` is planted; `TestGsdTagCreationIsDisabled` passes with unchanged logic. Only the `REQUIREMENTS.md` bookkeeping row is stale.

---

_Verified: 2026-10-14T23:30:00Z_
_Verifier: Claude (gsd-verifier)_
