---
phase: "5"
slug: "release-chain-release-please-retirement"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-10-14"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (`internal/upgrade`) + Python `unittest` (`scripts/test_fragment_policy.py`) + Taskfile gate targets (`check:changie`, `check:fragment-policy`) + actionlint (`task lint:actions`) |
| **Config file** | none beyond `Taskfile.yml`, `.golangci.yml`, `.github/workflows/ci.yml` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<the plan's tests>'` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 && task check:fragment-policy && task check:changie && task lint:actions`, then `task test` at the phase gate |
| **Estimated runtime** | ~60 seconds (package 0.7 s; `check:changie` and `check:fragment-policy` dominate) |

---

## Sampling Rate

- **After every task commit:** Run `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<the plan's tests>'`, plus `task lint:actions` when a workflow file changed
- **After every plan wave:** Run the full `internal/upgrade` package, `task check:fragment-policy`, `task check:changie`, `task lint:actions`
- **Before `/gsd-verify-work`:** Full suite plus `task test` green; scratch-tree rehearsal (copy the tree, run `release:batch`, then the `Changie` Go tests and `task check:changie` on it); `task release:dry-run` transcript captured; census clean
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-T1 | 05-01 | 1 | REL-10, GRD-15 | T-05-01, T-05-02, T-05-03, T-05-SC | release PR opened with the App token, ready for review; skip-ci scan; no expression inside a run body | Go shape + Taskfile gate + worktree e2e | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestChangieConfigShape\|TestReleasePRWorkflowShape\|TestReleaseBatchTargetShape\|TestWorkflowRunBodiesInvokeTask\|TestWorkflowFilePopulationMatchesDisk\|TestInScopeJobsPopulationMatchesDisk)$' -count=1 -v`; `task check:changie` (14 of 14); worktree `task release:batch` (tracer and empty set) | ✅ (changie_release_shape_test.go, leg 14) | ✅ green |
| 05-01-T2 | 05-01 | 1 | REL-10 | T-05-01, T-05-02, T-05-04 | each 05-01 guard bites | mutation families 05-01/a-h | `rg` over 05-MUTATION-LOG.md for the eight family headings (a-h), the Byte-clean proofs and the FAIL lines 05-01 Task 2's verify requires | ✅ (log) | ✅ green |
| 05-02-T1 | 05-02 | 2 | REL-11, GRD-16, GRD-15 | T-05-06, T-05-07, T-05-08 | tag pushed only with the App token; exactly one tag-creating site; release:tag refuses outside Actions | Go shape + scanner over literal inputs | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestReleaseTagWorkflowShape\|TestReleaseTagTargetShape\|TestTagCreationHasSingleAppTokenSite\|TestTagCreationScanFlagsPlantedSites\|TestGsdTagCreationIsDisabled\|…)$' -count=1 -v`; `task lint:actions` | ✅ | ✅ green |
| 05-02-T2 | 05-02 | 2 | REL-11 | T-05-08, T-05-11 | absent, present, transport-error and outside-Actions branches | rehearsal against a scratch bare origin (evidence, not a committed test) + D-06 value edit | `rg` for the four rehearsal lines in the log; REL-11/SC1 text and heading-count equality vs 5a8603f5 | n/a | ✅ green |
| 05-02-T3 | 05-02 | 2 | GRD-16 | T-05-07 | RED against a planted second tag push (workflow run body and Taskfile target) | mutation families 05-02/a-i | `rg` for the nine family headings (a-i), 4+ GRD-16 FAIL lines, the scanner-subtest FAIL lines of families g-i, `task:check:cross`, `ci.yml/test/planted` | n/a | ✅ green |
| 05-03-T1 | 05-03 | 3 | REL-12, REL-13, GRD-15, DOCS-14 | T-05-12, T-05-13, T-05-14 | goreleaser receives `--release-notes .changes/${TAG}.md`; a missing or empty notes file is refused; release.yml locked | Go shape | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestReleaseGoreleaserPassesReleaseNotes\|TestReleaseDryRunRehearsesReleaseNotes\|TestGoreleaserChangelogNotDisabled\|TestReleaseBlockDoesNotRewriteReleaseBody\|…)$' -count=1 -v`; `git diff -U0 7b12cb92 -- .github/workflows/release.yml` = one comment-only hunk | ✅ (changelog guard) | ✅ green |
| 05-03-T2 | 05-03 | 3 | REL-12, REL-13 | T-05-12, T-05-13 | each 05-03 guard bites | REL-12 and REL-13 value edits (D-12, D-14) + mutation families 05-03/a-e | `rg` over REQUIREMENTS/ROADMAP and the log | n/a | ✅ green |
| 05-03-T3 | 05-03 | 3 | REL-13 | T-05-15 | no publish, sign or tag in the rehearsal | captured `task release:dry-run` (darwin) on the newest seed and on a worktree-batched version | `rg` for two RELEASE-NOTES-EVIDENCE lines, two four-binary lines and two `release:dry-run exit=0` lines in the log | n/a | ✅ green |
| 05-04-T1 | 05-04 | 4 | GRD-15 | T-05-17, T-05-19 | release PR housekeeping by path; allowlists held equal | Python unittest | `python3 -B -m unittest discover -s scripts -p 'test_fragment_policy.py' -v` | ✅ (file exists; 6 new methods) | ✅ green |
| 05-04-T2 | 05-04 | 4 | GRD-15 | T-05-16 | linear-time exemption markers | Python unittest (timing) + executed-count gate | `task check:fragment-policy` (46 of 46, floor 46) | ✅ (4 new methods) | ✅ green |
| 05-04-T3 | 05-04 | 4 | GRD-15 | T-05-16, T-05-19 | each hygiene guard bites | mutation families 05-04/a-f | `rg` over the log for the six family headings (a-f) and the FAIL lines 05-04 Task 3's verify requires | n/a | ✅ green |
| 05-05-T1 | 05-05 | 5 | GRD-15, GRD-16 | T-05-20, T-05-21, T-05-22 | pre-1.0 Breaking stays minor; GSD tagging stays off | Go + RED-against-replacement simulation | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^(TestChangieBreakingBumpsMinorPreOne\|TestGsdTagCreationIsDisabled)$' -count=1 -v`; code-token equality of TestGsdTagCreationIsDisabled vs 5a8603f5 | ✅ (re-pointed) | ✅ green |
| 05-05-T2 | 05-05 | 5 | GRD-15 | T-05-20 | wrong-workflow SAN still rejected | Go fixture + comment-only workflow diff | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1`; `task lint:actions` | ✅ | ✅ green |
| 05-06-T1 | 05-06 | 4 | DOCS-12, DOCS-14 | — | — | Go doc-target test + `rg` | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -run '^TestContributingReferencesRealTaskTargets$' -count=1 -v`; `rg` teachings and census checks | ✅ | ✅ green |
| 05-06-T2 | 05-06 | 4 | DOCS-13 | T-05-23, T-05-24, T-05-25 | no hand-tag instruction survives | `rg` + verdict-table set equality | base-line set of R01-R45 equals the 45 lines at 5a8603f5; verdict vocabulary closed | ✅ (05-CENSUS.md) | ✅ green |
| 05-07-T1 | 05-07 | 6 | REL-14, GRD-15 | T-05-26 | deletion never leaves the suite red | Go + deletion-commit shape | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1`; deletion commit touches exactly four paths | ✅ | ✅ green |
| 05-07-T2 | 05-07 | 6 | REL-14 | T-05-27, T-05-28 | census is not vacuous; no outward action | positive-controlled census + phase gate | census exclusions equal REL-14's three clauses; control prints exactly the four in-scope plant lines (incl. `.changes/header.tpl.md`) and neither excluded plant; real census empty; `task test`, `task check:changie`, `task check:fragment-policy`, `task lint:actions` | n/a | ✅ green |
| 05-08-T1 | 05-08 | 7 | DOCS-13 | T-05-29 | no hand-pushed prerelease tag shape survives in runbook sections 1, 2 and 7 (gap G-05-1) | `rg`/`awk` section assertions + Go test reading the runbook | the plan's `<automated>` verify: section 4 D-15 paragraph byte-identical to 1c700149, section 2 two-shape table (4 table lines) with D-15 and §4 pointer, stale passages absent, Taskfile.yml unchanged, and `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade/ -count=1 -v` with the `--- PASS: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (` line asserted | ✅ (docs/RELEASE-PROCEDURES.md) | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Every guard is shown RED against a confirmed-applied, byte-cleanly-reverted mutation, with a positive assertion that it counted a non-zero population (rule `84d1gfpywd`). The requirement-to-test map is `05-RESEARCH.md` §Validation Architecture.

---

## Wave 0 Requirements

- [x] `internal/upgrade/changie_release_shape_test.go` — workflow and Taskfile shape for REL-10/REL-11, GRD-16, `release:goreleaser` and `release:dry-run` (created in 05-01 Task 1, extended in 05-02 and 05-03)
- [x] `changelog.disable` guard in `internal/upgrade/goreleaser_shape_test.go` — REL-12 assumes one exists; none does (05-03 Task 1, `TestGoreleaserChangelogNotDisabled`)
- [x] `check:changie` leg 14 for the batched release-file shape, with the leg-count constants raised to 14 consistently (05-01 Task 1); the empty-set no-op and the `release:batch` output are held by `TestReleaseBatchTargetShape` plus disposable-worktree rehearsals, because a Taskfile target runs in the Taskfile's own directory and cannot be pointed at check:changie's scratch copy
- [x] `scripts/test_fragment_policy.py` additions (exemption swap, parity, WR-01 timing/grammar) and the floor raised from 36 to 46 in `check:fragment-policy` (05-04)
- [x] REL-14 census: a recorded, positive-controlled instrument in `05-CENSUS.md` (05-07), not a committed Taskfile target (D-09 "plain grep census", D-01)

No new framework install is needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `task release:dry-run` passes `--release-notes .changes/${TAG}.md` for the tag being released, refuses a missing or empty notes file, and exits 0 before any real tag exists (D-14: this repository's wiring only) | REL-13 | Needs the darwin host GoReleaser toolchain; the transcript is captured evidence, not a CI check | Run `task release:dry-run` (with `env -u` for the empty `MACOS_*` variables) against an existing seed and a scratch-batched file; paste the RELEASE-NOTES-EVIDENCE lines and the `release:dry-run exit=0` lines into the plan SUMMARY (05-03 Task 3). The sha256 in each evidence line records which file was passed. How GoReleaser renders the Release body is GoReleaser's behavior and is not tested in this phase (D-14) |
| `release.yml` differs from post-#99 `main` by one change | REL-12 | One-time diff evidence | `git diff 7b12cb92 -- .github/workflows/release.yml` pasted into the plan SUMMARY |
| Merge-commit push fires the `CHANGELOG.md` path filter; App-token tag starts `release.yml` | REL-10 / REL-11 | Live GitHub behavior | Phase 6 (live proof), not this phase |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-10-16 (validate-phase re-audit after gap closure 05-08: 18/18 rows green, 0 gaps)

## Validation Audit 2026-10-14

| Metric | Count |
|---|---|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
