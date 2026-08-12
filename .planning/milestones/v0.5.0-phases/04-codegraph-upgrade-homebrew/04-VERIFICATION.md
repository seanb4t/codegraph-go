---
phase: 04-codegraph-upgrade-homebrew
verified: 2026-08-11T18:26:34Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 4: `codegraph upgrade` × Homebrew Verification Report

**Phase Goal:** Neither install path lies about what is installed — `codegraph upgrade` recognizes a Homebrew-managed install, steps aside with an actionable pointer, and never mutates the Caskroom or Cellar behind brew's back.
**Verified:** 2026-08-11T18:26:34Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (four amended ROADMAP success criteria, verified against the amended text)

| # | Truth (amended criterion) | Status | Evidence |
|---|---|---|---|
| 1 | (UPGR-01) Genuine `brew tap`+`brew install` → `codegraph upgrade` refuses, names `brew upgrade codegraph`, neither `Options.download` nor `Options.swap` invoked; real-tap run only has to observe the refusal | ✓ VERIFIED | `internal/upgrade/upgrade.go:98-105` — `detectBrewManaged` runs before `resolveLatest`; refusal is `fmt.Errorf`, returns before any seam call. `TestUpgradeRun_RefusesBrewManagedCask` (`upgrade_test.go:304-391`) independently re-run here, PASS — asserts `resolveLatestCalled/downloadCalled/verifyCalled/swapCalled` all false and error contains `brew upgrade codegraph` + resolved install dir. `04-EVIDENCE.md` Leg 1 (genuine, unmutated `TestDetectBrewManaged_RealInstall` PASS against a real `brew install` from `seanb4t/tap`) + Leg 2 (`codegraph upgrade` against that real Caskroom path, `exit=1`, message matches exactly) jointly satisfy the amended "only has to observe the refusal" text |
| 2 | (UPGR-03) `codegraph upgrade --check` under the same install steps aside with the same pointer (not a version), mutates nothing (same seam), exits 0 | ✓ VERIFIED | `upgrade.go:100-103` returns `nil` before `resolveLatest`. `TestUpgradeRun_CheckBrewManagedStepsAside` (`upgrade_test.go:443-481`) independently re-run here, PASS — all four seams unfired, output contains pointer + resolved dir, no version number. `04-EVIDENCE.md` Leg 2: real binary `upgrade --check` exit `0`, identical pointer sentence, no second version manufactured |
| 3 | (UPGR-02) Detection fires on resolved-symlink Caskroom/Cellar at Apple Silicon, Intel, custom prefix, and linuxbrew (BOTH shapes), and does NOT fire on a non-brew binary at a path merely containing `Cellar`, as an EXECUTING test | ✓ VERIFIED | `TestDetectBrewManaged` (`brew_test.go:159-329`), independently re-run here, PASS across all 16 sub-tests: 8 detected rows (4 prefixes × {Caskroom, Cellar}), plus `cellar-no-receipt-not-detected` and `caskroom-no-receipt-not-detected` — both are executing sub-tests (not comments, not skipped) that put a real `Cellar`/`Caskroom`-shaped path with no `INSTALL_RECEIPT.json` through the real detector and assert `ok == false`. Row-count guards (`len(rows) < 16`, `notDetectedCount < 7`) fail loudly if a row is silently dropped |
| 4 | (UPGR-02) Non-brew install on a machine where `brew` is absent from `PATH` upgrades normally — detection never depends on Homebrew existing | ✓ VERIFIED | `internal/upgrade/brew.go` — `detectBrewManaged` uses only `filepath.EvalSymlinks`/`filepath.Dir`/`os.Stat`; `rg exec\.Command\|exec\.LookPath\|"brew"` across `internal/upgrade/*.go` and `internal/cli/upgrade*.go` returns zero production hits (only a `bash` exec in an unrelated lint test, and a `verify_test.go` assertion that `verify.go` itself contains no `os/exec`). No code path in the detector or `Run()` can depend on `brew` being on `PATH` |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/upgrade/brew.go` | Structural detector (path-shape + `INSTALL_RECEIPT.json`), shared pointer message builder | ✓ VERIFIED | `detectBrewManaged` (D-03/D-12) walks the resolved binary's ancestry, matches `Caskroom`/`Cellar` at index ≥2, requires a receipt at the tree-specific location; `brewPointerMessage` is the single shared string builder consumed by both the refusal and `--check` |
| `internal/upgrade/brew_test.go` | 16-row constructed-tree table incl. executing false-positive rows | ✓ VERIFIED | Present, substantive, all 16 rows pass on independent re-run; row-count floor guards present |
| `internal/upgrade/upgrade.go` (`Run()`) | Early brew-branch before `resolveLatest`, D-05/D-06/D-09/D-10/D-11 semantics | ✓ VERIFIED | Lines 98-105; `opts.Force` is never read by this branch (enforces D-06 by omission, not a runtime check) |
| `internal/cli/upgrade.go` | Wires `os.Executable()` target + `upgrade.Options` through to `Run`, propagates refusal error for non-zero exit | ✓ VERIFIED | `RunE` returns `upgradeRunFunc(...)` unchanged; cobra surfaces a non-nil error as a non-zero exit by default |
| `.goreleaser.yaml` / `Taskfile.yml` sentinel removal (D-02) | Phase-3 sentinel write/remove/assertion fully deleted | ✓ VERIFIED | `rg codegraph-brew-install` across both files returns zero hits; `hooks.post.install` now carries the UF-5 mtime/size-baseline freshness assertion instead |
| `README.md` / `docs/RELEASE.md` / `--help` (D-10, Wave 2) | Document both exit behaviors and the pointer command | ✓ VERIFIED | All three carry `brew upgrade codegraph` / brew-managed-install language; `04-EVIDENCE.md` Leg 2 captures the real `--help` text verbatim, matching |
| `.planning/phases/04-codegraph-upgrade-homebrew/04-EVIDENCE.md` | Real-tap acceptance run, three legs with distinct evidentiary status | ✓ VERIFIED | Present, substantive; explicitly separates genuine-tree (Leg 1), payload-substituted (Leg 2), and unexecuted (Leg 3) evidence rather than conflating them |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `internal/upgrade/upgrade.go: Run()` | `internal/upgrade/brew.go: detectBrewManaged` | Direct call at head of `Run()`, before `resolveLatest` | ✓ WIRED | Confirmed by reading `Run()` and by both seam-based tests passing |
| `internal/cli/upgrade.go: newUpgradeCmd().RunE` | `internal/upgrade.Run` (via `upgradeRunFunc`) | `os.Executable()` resolves `targetPath`; `version.Info().Version` resolves `currentVersion` | ✓ WIRED | Matches D-13's "caller resolves targetPath" design; confirmed by reading the CLI wiring |
| Refusal message ↔ `--check` step-aside message | `brewPointerMessage(inst)` | Single shared builder, called from both `Run()` branches | ✓ WIRED | One function, two call sites in `Run()` (lines 99, 101/104) — cannot drift apart at this layer (the cobra `Long` text is a separately-pinned third literal, correctly disclosed as such in 04-01's must_haves rather than hidden) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full detection table fires/refuses correctly | `go test ./internal/upgrade/... -run 'TestDetectBrewManaged$' -v` | 16/16 sub-tests PASS | ✓ PASS |
| `Run()` refuses + seams unfired (bare upgrade) | `go test ./internal/upgrade/... -run TestUpgradeRun_RefusesBrewManagedCask -v` | PASS | ✓ PASS |
| `Run(--check)` steps aside + seams unfired | `go test ./internal/upgrade/... -run TestUpgradeRun_CheckBrewManagedStepsAside -v` | PASS | ✓ PASS |
| `--force` powerless against refusal | `go test ./internal/upgrade/... -run TestUpgradeRun_ForceDoesNotOverrideBrewRefusal -v` | PASS | ✓ PASS |
| No `brew`/PATH dependency in detector | `rg 'exec\.Command|exec\.LookPath|"brew"' internal/upgrade/*.go internal/cli/upgrade*.go` | Zero production hits | ✓ PASS |
| Sentinel fully removed | `rg codegraph-brew-install .goreleaser.yaml Taskfile.yml` | No output | ✓ PASS |

All five were re-run independently by this verifier (not taken from SUMMARY claims), against the current merged tree.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| UPGR-01 | 04-01, 04-06 | Detect brew-managed install, refuse, point at `brew upgrade codegraph`, never mutate Caskroom/Cellar | ✓ SATISFIED (codebase) | Criteria 1 above; `04-EVIDENCE.md` criterion-mapping table marks it "Fully evidenced" |
| UPGR-02 | 04-01, 04-06 | Symlink-resolved detection correct across prefixes/linuxbrew; never depends on `brew` on PATH | ✓ SATISFIED (codebase) | Criteria 3 and 4 above |
| UPGR-03 | 04-01, 04-06 | `--check` stays read-only under a brew install, reports how to upgrade | ✓ SATISFIED (codebase) | Criterion 2 above |

**Note — REQUIREMENTS.md checkboxes:** `.planning/REQUIREMENTS.md:40-42` still show UPGR-01/02/03 as `- [ ]` and the traceability table (`:86-88`) still reads "Pending", unlike Phases 1-3's requirements which flip to `[x]`/"Complete" once verified. Plan `04-03` deliberately left them unchecked on the stated grounds that amending prose does not itself deliver a requirement (`04-03-SUMMARY.md`). Having now independently re-run the seam tests, the detection table, and cross-checked the real-tap evidence against the amended criteria, this verifier's determination is that all three ARE discharged by 04-01 (implementation + unit proof) and 04-06 (real-tap observation) — the codebase evidence is sufficient. The stale checkboxes are a documentation-sync gap for the ship/completion step to close, not a phase-goal blocker; flagged here so it isn't silently missed.

### Anti-Patterns Found

None in the files this phase modified. `git diff` of `.goreleaser.yaml`, `Taskfile.yml`, `internal/upgrade/*`, `internal/cli/upgrade*.go` against the phase base contains no added `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers. No stub returns, no hardcoded-empty props, no console-log-only handlers in the reviewed detection/refusal code path.

### Deep Code Review Cross-Check (04-REVIEW.md)

0 Critical, 1 Warning (WR-01: `Taskfile.yml`'s `verify:self-upgrade` guards the cosign-bundle download against "matched nothing" but not the binary download identically — fail-closed already via `cosign verify-blob` erroring on a missing file, non-blocking, cosmetic diagnostic-clarity fix), 2 Info (TOCTOU between detection and swap is structural/accepted, not a defect; detection predicate independently cross-checked against a real live `1password-cli` cask and several Cellar formulae on the reviewer's own machine — both receipt-location assumptions confirmed correct). None of these affect the phase goal.

### Known Residual Gap — judged, not silently accepted

`04-EVIDENCE.md` Leg 3 ("the fully natural path") is explicitly **not executed and not claimed**: no released binary yet carries this phase's detection code, so Leg 2 observed the refusal through a genuine Caskroom tree with a substituted payload (a binary built from this worktree, swapped into the real install) rather than through a naturally `brew install`-ed release. The evidence file states this distinction in its own leg heading rather than blurring it.

**Judgment:** this does not block the phase goal. The amended criterion 1 text only requires the real-tap acceptance run to "observe the refusal" — it does not require the observed binary to itself be a CI-released artifact, and it explicitly says the seam-unfired half is instead carried by 04-01's unit-level proof (which this verifier independently re-ran and confirms passes). Leg 1 already proves detection fires unmutated against Homebrew's own genuinely-produced tree (`TestDetectBrewManaged_RealInstall`, independently reasoned about here via the harness log's field-by-field cross-check against the recorded `ls`/`realpath` output). The chicken-and-egg constraint (you cannot observe phase-4 code via a natural release cut before phase 4 has merged) is inherent to verifying pre-release code, is disclosed with a precise, actionable closing condition (next release cut publishes a cask carrying this code; a subsequent `brew upgrade codegraph` or fresh install exercises it end to end), and does not represent an untested code path — the code path IS tested, just not via the specific "genuinely-released, brew-installed, zero-substitution" instance. Treating this as a phase-goal blocker would be penalizing honest disclosure of a sequencing constraint that structurally cannot be closed before this phase ships.

### Human Verification Required

None. No item in this phase required judgment calls beyond what the codebase, tests, and disclosed evidence already settle.

### Gaps Summary

No gaps found. All four amended ROADMAP success criteria are independently verified against the current codebase (not SUMMARY claims): detection is structural, prefix-agnostic, receipt-gated, PATH-independent; `Run()` branches before any network call; the seam-based proof plus the real-tap observation jointly satisfy the amended UPGR-01/UPGR-03 criteria; the Phase-3 sentinel is fully removed; documentation is in sync; the deep code review found no Critical/blocking issues. The one disclosed residual (Leg 3, a natural-release observation) is judged non-blocking per the reasoning above. The only follow-up worth tracking outside this phase is syncing REQUIREMENTS.md's UPGR-01/02/03 checkboxes/traceability status to "Complete" at ship time, and optionally WR-01's Taskfile diagnostic-clarity fix.

---

_Verified: 2026-08-11T18:26:34Z_
_Verifier: Claude (gsd-verifier)_
