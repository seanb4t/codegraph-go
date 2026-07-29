---
phase: 08-surface-reconciliation-signed-v1-0-0-release
verified: 2026-07-19T21:00:00Z
status: passed
score: 9/9 in-scope must-haves verified (1 of the original 10 moved to Phase 9 with REL-02, 2026-07-28)
behavior_unverified: 0
overrides_applied: 0
requirements_coverage:
  SURF-01: satisfied
  SURF-02: satisfied
  SURF-03: satisfied
  SURF-04: satisfied
  SURF-05: satisfied
  REL-01: satisfied
  REL-02: out_of_scope (moved to Phase 9)
  REL-03: satisfied
  REL-04: satisfied
# Both arrays emptied 2026-07-28: the Affected() ordering item was RESOLVED (see
# "Human Verification Required" #1 in the body for the full evidence + caveat), and the
# REL-02 item moved OUT OF SCOPE to Phase 9. Kept empty rather than deleted so the keys
# stay present for tooling; the body carries the audit trail.
behavior_unverified_items: []
human_verification: []
resolved_2026_07_28:
  - item: "Affected() BFS output-ordering determinism"
    disposition: resolved
    evidence: "TestImpactCallersAffectedDeterministicAcrossRepeatedCalls (internal/query/traverse_test.go:596), green at 387cb4b; sortLocations landed d3f077c/4feb6ff"
  - item: "Signed v1.0.0 release cut (REL-02)"
    disposition: out_of_scope
    evidence: "Rewritten as a release-automation property, reassigned to Phase 9. Never executed; no v1.0.0 tag exists."
---

# Phase 8: Surface Reconciliation & Signed v1.0.0 Release Verification Report

**Phase Goal:** Every TS flag name and default is present or a documented divergence, then the new Charm dependency closure is audited and the "drop-in parity" claim is validated — retiring v0.1's "not yet drop-in" caveat and closing its pending PERF-01.

> **Goal amended 2026-07-28**, after this report was written on 2026-07-19. The goal verified against on 2026-07-19 also promised "the first real signed `v1.0.0` is cut" and the closure of v0.1's pending DIST-02; both moved to Phase 9 with REL-02 (rewritten from a project event into a release-automation property). The release clause was **never satisfied and is not claimed to be** — no `v1.0.0` tag exists. It was removed from scope, not achieved. The REL-02 items below are marked OUT OF SCOPE for exactly this reason.
**Verified:** 2026-07-19
**Status:** passed — canonicalized 2026-07-28 from `human_needed` (see Status Amendment at the end of this report)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `impact`'s default BFS depth is 2 (was 5), shared by CLI + MCP via the engine (SURF-01) | ✓ VERIFIED | `internal/query/validate.go:38` `defaultDepth = 2`; `MaxDepth = 50` unchanged; `impact -d/-j` registered (`internal/cli/impact.go:64-66`); `go test ./testdata/golden/... -run TestGoldenParity/impact` passes with `depth:2` fixtures |
| 2 | `files` gains a directory-path prefix filter `--dir`, retaining the existing language `--filter` (SURF-02) | ✓ VERIFIED | `internal/cli/files.go:90` `StringVar(&dir, "dir", ...)`; `internal/query/files.go` `FilesOptions.Dir` + `dirPrefixMatches` (strings.HasPrefix, no glob dep); `files -j` registered; `go test ./internal/query/... -run TestFiles` passes |
| 3 | Missing short-flag aliases added across commands, matching TS letters where free, existing Go bindings preserved (SURF-03) | ✓ VERIFIED | `status -j`, `query -l/-k/-j`, `callers/callees -l/-j`, `install/uninstall -t/-l`, `upgrade -f/--force` all confirmed present via source read; `go build ./...` green (no cobra shorthand collisions) |
| 4 | `affected` gains `--stdin`/`--depth`/`--filter <glob>`/`--quiet` for git-hook/CI scripting, and the engine BFS is depth-bounded with TS test-leaf pruning (SURF-04) | ✓ VERIFIED | `internal/cli/affected.go`: `Args: cobra.ArbitraryArgs`, `--stdin`, `-d/--depth`, `-f/--filter`, `-q/--quiet` all registered; `internal/query/traverse.go` `Affected(files, depth)` BFS with `isTestSymbol` leaf-pruning; `defaultAffectedDepth=5` distinct from `defaultDepth=2`; `go test ./internal/query/... -run TestAffected` and `go test ./internal/cli/... -run Affected` both pass |
| 4b | Affected() BFS output ordering is deterministic across runs (backstop truth, 08-04 must_haves) | ✓ VERIFIED (2026-07-28) | Was PRESENT_BEHAVIOR_UNVERIFIED on 2026-07-19. Closed 2026-07-28: `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` (`traverse_test.go:596`) asserts byte-identical `MarshalAffectedJSON` across 6 calls; green at 387cb4b. Backstop-assertion caveat recorded in the Human Verification section |
| 5 | A systematic per-command flag audit (`docs/FLAG-PARITY.md`) confirms every TS flag + default is present or documented divergence; `search`/`migrate` recorded as Go-only/accepted divergence; self-verifying via a drift test (SURF-05) | ✓ VERIFIED | `docs/FLAG-PARITY.md` (297 lines, one `##` section per `newRootCmd()` command incl. flag-less ones); `search`/`migrate` sections explicitly labeled "Go-only, no TS command"; `go test ./internal/cli/... -run FlagParity` (`TestFlagParityDocCoversRegisteredFlags`) passes |
| 6 | The Charm/TUI dependency closure is audited — no new CGo, `govulncheck` clean, SBOM regenerates, reproducible double-build still passes (REL-01) | ✓ VERIFIED | `internal/cli/present/archtest/charm_cgo_test.go` `TestCharmCgoClosure`: "10 packages, 0 with CgoFiles" (non-vacuous); 08-07-SUMMARY records govulncheck clean, `goreleaser check` + syft SBOM emission, and a hash-matched local double-build reproduction; `go test ./internal/cli/present/archtest/... -run CharmCgo` passes here |
| 7 | Head-to-head benchmarks vs TS 1.3.1 are re-run and published, closing PERF-01 (REL-03) | ✓ VERIFIED | `docs/BENCHMARKS.md` (300 lines) contains a refreshed CI (ubuntu-latest, commit `ca511e7`, 3 real `gh run` IDs) median-of-3 table superseding the old darwin/arm64 provisional numbers; methodology (median-of-3, same corpora, pinned TS 1.3.1) unchanged per prohibition |
| 8 | A real signed `v1.0.0` release is cut (per-binary cosign keyless + SLSA provenance + SBOM), closing DIST-02 (REL-02) | OUT OF SCOPE (moved to Phase 9) | REL-02 was rewritten as a release-automation property and reassigned to Phase 9 (2026-07-28); its testable content (per-binary cosign + SLSA + SBOM) was already proven under v0.1's DIST-02 on release v0.0.0-rc.3 |
| 9 | The drop-in parity claim is validated (behavioral harness + flag audit green) and PROJECT.md's "not yet drop-in" caveat is retired (REL-04) | ✓ VERIFIED | `go test ./testdata/golden/... -count=1` green; `go test ./internal/cli/... -run FlagParity` green; `rg -n "not yet drop-in|drop-in parity" .planning/PROJECT.md` shows both remaining hits are now **positive** ("✓ Complete... drop-in gate green... cut the first signed v1.0.0"), zero "not yet" occurrences |

**Score:** 9/9 in-scope truths verified (8 primary + the 4b backstop sub-item, all green as of 2026-07-28); 0 behavior-unverified; 1 out of scope, moved to Phase 9 (truth 8, REL-02) and therefore excluded from the denominator rather than counted as a miss.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/query/validate.go` | `defaultDepth=2`, `defaultAffectedDepth=5`, `MaxDepth=50` | ✓ VERIFIED | Confirmed via source read |
| `internal/query/traverse.go` | `Affected(files, depth)` depth-bounded BFS w/ test-leaf pruning | ✓ VERIFIED | Confirmed via source read + `TestAffected`/`TestAffectedDepthBFSWithTestLeafPruning` green |
| `internal/query/files.go` | `FilesOptions.Dir` + prefix predicate | ✓ VERIFIED | Confirmed; `strings.HasPrefix`-only, no glob dep |
| `internal/cli/impact.go`, `files.go`, `affected.go`, `status.go`, `query.go`, `callers.go`, `callees.go`, `install.go`, `uninstall.go`, `upgrade.go` | SURF-03 short flags + SURF-04 scripting flags | ✓ VERIFIED | All grep-confirmed present, `go build ./...` green (no shorthand collisions) |
| `internal/cli/present/archtest/charm_cgo_test.go` | Non-vacuous, fail-closed CGo-closure guard | ✓ VERIFIED | Exists, passes, "10 packages, 0 with CgoFiles" |
| `internal/cli/flag_parity_test.go` | Drift guard over `docs/FLAG-PARITY.md` | ✓ VERIFIED | Exists, passes |
| `docs/FLAG-PARITY.md` | Per-command TS↔Go matrix, self-verifying | ✓ VERIFIED | 297 lines, all `newRootCmd()` commands covered incl. `search`/`migrate` Go-only |
| `docs/BENCHMARKS.md` | Refreshed CI head-to-head numbers | ✓ VERIFIED | 300 lines, real CI run IDs, methodology unchanged |
| `docs/RELEASE-PROCEDURES.md` | Maintainer runbook | ✓ VERIFIED | 183 lines; contains `go list -mod=readonly` sweep and `releaseWorkflowRefPattern` verbatim citation |
| `.planning/PROJECT.md` | Caveat retired at every site | ✓ VERIFIED | `rg "not yet drop-in"` → 0 hits |
| `internal/upgrade/verify.go`, `.github/workflows/release.yml` | Byte-unchanged (LOCKED contract) | ✓ VERIFIED | `git diff <phase-8-start>..HEAD -- these files` empty; constants (`releaseWorkflowRefPattern`/`releaseOIDCIssuer`/`releaseRepoSlug`) intact |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `internal/query/validate.go` `defaultDepth` | `internal/query/traverse.go Impact()` | `clampDepth` | ✓ WIRED | Both CLI `impact` and MCP `codegraph_impact` inherit via shared engine |
| `internal/cli/files.go --dir` | `internal/query.FilesOptions.Dir` | struct literal wiring | ✓ WIRED | Confirmed in source |
| `internal/cli/affected.go` stdin/args/flags | `eng.Affected(files, depth)` (08-04 BFS) | `collectAffectedFiles` → `eng.Affected` call | ✓ WIRED | Confirmed; `--quiet` bypasses `present.RenderFiles`/`WorktreeNotice` per grep |
| `internal/cli/flag_parity_test.go` | `docs/FLAG-PARITY.md` | `os.ReadFile` + `pflag.Flag` substring assertion | ✓ WIRED | Test passes; self-defeat guard exercised per SUMMARY |
| `docs/FLAG-PARITY.md` + `testdata/golden` harness | REL-04 drop-in gate → `.planning/PROJECT.md` caveat retirement | both gates run green before the PROJECT.md edit | ✓ WIRED | Both gates independently re-confirmed green in this verification pass |
| `docs/RELEASE-PROCEDURES.md` | `internal/upgrade/verify.go` LOCKED constants | cites verbatim, does not edit | ✓ WIRED | `releaseWorkflowRefPattern` string present in runbook; verify.go diff empty |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Package builds cleanly (no cobra shorthand collisions across all SURF-03/04 edits) | `go build ./...` | exit 0 | ✓ PASS |
| Golden/behavioral parity harness (TEST-01, incl. `impact` depth-2 fixtures) | `go test ./testdata/golden/... -count=1` | all subtests PASS | ✓ PASS |
| Flag-parity drift guard (SURF-05) | `go test ./internal/cli/... -run FlagParity -count=1` | PASS | ✓ PASS |
| Charm CGo-closure guard (REL-01) | `go test ./internal/cli/present/archtest/... -run CharmCgo -count=1` | PASS — "10 packages, 0 with CgoFiles" | ✓ PASS |
| Affected engine BFS + CLI scripting surface | `go test ./internal/query/... -run TestAffected` and `go test ./internal/cli/... -run Affected` | both PASS | ✓ PASS |
| Full workspace suite (regression check) | `go test ./... -count=1` | All packages PASS except `internal/daemon` (2 flaky timing tests, both PASS on immediate re-run in isolation; package untouched by any Phase-8 plan) | ✓ PASS (flake noted, not a Phase-8 regression) |
| Caveat retirement / drop-in gate wiring | `rg "not yet drop-in" .planning/PROJECT.md` | 0 hits | ✓ PASS |
| v1.0.0 tag existence (REL-02) | `git tag -l`, `git describe --tags` | no `v1.0.0` tag; `v0.1.0-452-gf6278ef` | OUT OF SCOPE (moved to Phase 9) — REL-02 reassigned as a release-automation property; its testable content already proven under v0.1's DIST-02 |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's PLAN/SUMMARY files; none found under `scripts/`. Skipped (no applicable probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| SURF-01 | 08-01 | `impact` default depth 5→2 | ✓ SATISFIED | `defaultDepth=2`, golden fixtures match |
| SURF-02 | 08-02 | `files --dir` new filter, `--filter` retained | ✓ SATISFIED | `FilesOptions.Dir`, `dirPrefixMatches` |
| SURF-03 | 08-01/02/03/05 | Missing short-flag aliases added | ✓ SATISFIED | Confirmed across 9 commands |
| SURF-04 | 08-04/05 | `affected` scripting flags + depth-bounded BFS | ✓ SATISFIED (BFS ordering backstop unverified — see truth 4b) | `Affected(files,depth)`, CLI flags, tests green |
| SURF-05 | 08-06 | Flag audit doc + drift test | ✓ SATISFIED | `docs/FLAG-PARITY.md` + `flag_parity_test.go` |
| REL-01 | 08-07 | Charm closure audit | ✓ SATISFIED | `charm_cgo_test.go` + recorded govulncheck/SBOM/double-build evidence |
| REL-02 | 08-09 (Task 3) | Signed v1.0.0 cut | OUT OF SCOPE (moved to Phase 9) | Rewritten as a release-automation property and reassigned to Phase 9; testable content already proven under v0.1's DIST-02 on release v0.0.0-rc.3 |
| REL-03 | 08-08 | Benchmarks re-run + published | ✓ SATISFIED | `docs/BENCHMARKS.md` refreshed with real CI numbers |
| REL-04 | 08-09 (Tasks 1-2) | Drop-in validation + caveat retirement | ✓ SATISFIED | Both gate halves green; caveat retired at every site |

No orphaned requirements: SURF-06 and TEST-04 (the two other SURF/TEST IDs near this phase in REQUIREMENTS.md) are explicitly mapped to Phase 2 and Phase 3 respectively per the traceability table and ROADMAP mapping notes — not orphaned to Phase 8.

### Anti-Patterns Found

None. Scanned all files touched by this phase's 9 plans (`internal/query/{validate,traverse,files}.go`, `internal/cli/{impact,files,affected,status,query,callers,callees,install,uninstall,upgrade}.go`, `internal/upgrade/upgrade.go`, `internal/cli/present/archtest/charm_cgo_test.go`, `internal/cli/flag_parity_test.go`, `docs/FLAG-PARITY.md`, `docs/RELEASE-PROCEDURES.md`, `docs/BENCHMARKS.md`, `.planning/PROJECT.md`) for `TBD|FIXME|XXX|HACK|PLACEHOLDER|not yet implemented|coming soon` — zero matches.

### Human Verification Required

#### 1. Affected() BFS output-ordering determinism (backstop truth, 08-04)

**Test:** Run `internal/query.Engine.Affected(files, depth)` repeatedly against the same fixture/index (e.g. `go test ./internal/query/... -run TestAffected -count=20 -race`) and diff the ordered `AffectedTests` result across runs.
**Expected:** Byte-identical path ordering on every run — no Go map-iteration-order leakage into the returned slice.
**Why human:** 08-04-PLAN.md's own frontmatter marks this must-have `verification: backstop` and its SUMMARY explicitly defers it "for human confirmation during a later `validate-phase` pass." No test added in this phase asserts cross-run stability; the code is present and BFS-correct (verified above) but this specific invariant is unexercised.

**RESOLVED 2026-07-28 — `/gsd-verify-work 8`, UAT Test 2 (`08-UAT.md`).** The "no test asserts cross-run stability" clause above was accurate when written on 2026-07-19 but was overtaken by the code-review fix loop, which landed after this report. `internal/query/traverse.go` now applies `sortLocations` to all four traversals — Impact/Callers/Affected (WR-05, d3f077c) and Callees (WR-02, 4feb6ff), the latter sorting *before* the limit/MaxLimit truncation so the cap selects a stable prefix rather than an arbitrary one. `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` (`internal/query/traverse_test.go:596`) calls `engine.Affected([]string{"pkga/target.go"}, 2)` six times and requires byte-identical `MarshalAffectedJSON` output across every call; verified green 2026-07-26 at commit 387cb4b.

**Recorded caveat (not a blocker):** this is a backstop assertion, not proof the sort is load-bearing — it would also pass if the underlying Pebble scan order were already stable. What the must-have asked for (cross-run ordering *asserted* rather than *assumed*) is satisfied. The companion `TestCalleesSortedDeterministically` **is** proven non-vacuous: removing `sortLocations` fails it with got "Zeta", want "Alpha".

#### 2. ~~Signed v1.0.0 release cut (REL-02)~~ — OUT OF SCOPE (moved to Phase 9)

REL-02 was rewritten as a release-automation property (release-please owns version bump/`CHANGELOG.md`/tag creation, with the resulting signed artifacts still satisfying `internal/upgrade/verify.go`'s cosign identity) and reassigned from Phase 8 to Phase 9 on 2026-07-28. This is no longer a Phase 8 human-verification obligation — it is neither passed nor failed here; it is out of scope for this phase. Its previously-testable content (per-binary cosign + SLSA provenance + SBOM) was already proven under v0.1's DIST-02 on release `v0.0.0-rc.3`. See `.planning/REQUIREMENTS.md`'s rewritten REL-02 and `.planning/ROADMAP.md` Phase 9.

### Gaps Summary

No blocking gaps. All 8 requirement IDs Phase 8 now owns (SURF-01..05, REL-01, REL-03, REL-04) are satisfied in the codebase with passing automated evidence (build green, `go test ./...` green apart from a pre-existing/unrelated `internal/daemon` flake that passes on isolated re-run, golden/behavioral harness green, flag-parity drift test green, CGo-closure guard green). The one remaining open item is **expected, not a defect**:

1. **REL-02 is out of scope for Phase 8** (moved to Phase 9, 2026-07-28) — it was rewritten as a release-automation property that Phase 9's release-please + GoReleaser work now owns; its previously-testable content (per-binary cosign + SLSA + SBOM) was already proven under v0.1's DIST-02 on `v0.0.0-rc.3`. This is not a code gap.
2. ~~**Affected() BFS ordering determinism**~~ — **RESOLVED 2026-07-28.** Closed by `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` (added by the code-review fix loop after this report was written) and confirmed via UAT Test 2. See the Human Verification section above for the evidence and its recorded caveat.

Both items were routed to human verification per the escalation-gate pattern rather than being silently marked passed or falsely marked failed. Item 1 was removed from scope; item 2 was resolved with evidence.

---

_Verified: 2026-07-19_
_Verifier: Claude (gsd-verifier)_

## Status Amendment — 2026-07-28

`status` canonicalized `human_needed` → `passed` during `/gsd-verify-work 8`.

**Basis.** Phase 8 owns 8 requirements (SURF-01..05, REL-01, REL-03, REL-04); all are `satisfied`. Both items that held this report at `human_needed` are now closed, by different and independently-recorded means:

| Item | Disposition | How |
|------|-------------|-----|
| Signed `v1.0.0` release (REL-02) | **OUT OF SCOPE** | Rewritten as a release-automation property and reassigned to Phase 9 on 2026-07-28. Never executed; no `v1.0.0` tag exists. Removed from scope — **not** achieved, and not claimed to be. |
| Affected() BFS ordering determinism | **RESOLVED** | Automated test landed post-report (d3f077c / 4feb6ff), verified green at 387cb4b, confirmed by UAT Test 2. |

**What this status does NOT assert.** It does not assert that a v1.0.0 release was cut, that a tag exists, or that REL-02 was met. REL-02 remains `Pending` under Phase 9 in `.planning/REQUIREMENTS.md`. The phase goal was amended the same day (see the amendment note near the top of this report) precisely so this status is measured against what Phase 8 actually owns.

_Amended by: `/gsd-verify-work 8`, 2026-07-28_
