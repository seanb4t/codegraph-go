---
phase: 1
slug: cross-compile-spike-goreleaser-release-migration
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-08
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` — release-pipeline correctness is enforced by **workflow-shape tests** that parse the on-disk YAML of `.github/workflows/*.yml`, `.goreleaser.yaml`, and `Taskfile.yml` and assert structural invariants |
| **Config file** | none — standard `go test`, no framework config |
| **Quick run command** | `go test ./internal/upgrade/... -run <TestName>` |
| **Full suite command** | `go test ./internal/upgrade/...` (shape tests); `go test ./...` (whole repo, 47 packages) |
| **Estimated runtime** | shape suite 0.44s test time / ~2.2s wall including compile; `task release:dry-run` ~15s; `task release:dry-run-signed` ~30s (measured 2026-08-11) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/... -run <TestName>` for the shape test that task adds or rewrites
- **After every plan wave:** Run `go test ./internal/upgrade/...`, plus `task check:goreleaser` (DIST-01) when the wave touched `.goreleaser.yaml`
- **Before `/gsd-verify-work`:** Full `go test ./...` green, plus a real dispatch of the D-03 canary workflow (requires live Namespace macOS + Linux runners — cannot be simulated locally)
- **Max feedback latency:** 3 seconds (local shape tests); canary/post-release legs are CI-latency bound and sampled per wave, not per task

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | REL-05, REL-06 | T-01-01…05 / T-01-SC | Both linux build ids carry a `zig cc`/`zig c++` `CC`/`CXX` override; a mutation removing either turns a Go test red | shape + e2e | `go test ./internal/upgrade/... -run 'TestLinuxBuildIdsCrossCompileViaZig\|TestParseGoreleaserBuildEnv_MissingBuildIDIsError' -v && go test ./internal/upgrade/... && task check:goreleaser && task lint:actions && task release:dry-run` | ✅ | ✅ green |
| 01-01-02 | 01 | 1 | REL-05 | T-01-01…05 / T-01-SC | Canary pins every third-party Action to a full commit SHA; arm64 exec runs on real hardware, never emulation | shape + workflow | `task lint:actions && go test ./internal/upgrade/... && task --list \| rg -q 'check:linux-cross-exec' && rg -c 'namespace-profile-linux-arm64-4x8' .github/workflows/linux-cross-canary.yml` | ✅ | ✅ green |
| 01-01-03 | 01 | 1 | REL-05 | T-01-05 (Repudiation) | REL-05 pass/fail is machine-checked by `check:linux-cross-exec`'s `REL05-EVIDENCE` line, never a green exit code | checkpoint:human-verify | — (no automated command; requires live Namespace Linux runners) | n/a | 📋 manual-only |
| 01-02-01 | 02 | 2 | REL-06, REL-09 | T-01-06…10, T-01-30, T-01-41, T-01-44 / T-01-SC | Raw archive stays `format: binary` byte-unchanged; `checksum.ids` covers exactly `[raw, zip]` | tdd/shape | `go test ./internal/upgrade/... -run 'TestRawArchiveEntryStaysBinaryFormat\|TestZipArchiveSharesRawAssetStem\|TestChecksumCoversRawAndZipIdsOnly\|TestParseGoreleaserArchives_NoArchivesBlockIsError\|TestReleaseAssetNameMatchesGoReleaser' -v && task check:goreleaser && go test ./internal/upgrade/...` | ✅ | ✅ green |
| 01-02-02 | 02 | 2 | REL-06 | T-01-06, T-01-09, T-01-30 (Tampering) | Sign/SBOM sidecar templates resolve to **four distinct** published names — a colliding template silently clobbers 3 of 4 signatures under `replace_existing_artifacts` | tdd/shape | `go test ./internal/upgrade/... -run 'TestSignsSidecarMatchesUpgradeContract\|TestSbomsArePerBinaryWithSpdxNames' -v && task check:goreleaser && go test ./internal/upgrade/... && task release:dry-run` | ✅ | ✅ green |
| 01-02-03 | 02 | 2 | REL-06 | T-01-08 (Repudiation), T-01-44 (DoS) | `release:` pipe is rerun-idempotent and does not rewrite the release body | tdd/shape | `go test ./internal/upgrade/... -run 'TestReleaseBlockIsRerunIdempotent\|TestReleaseBlockDoesNotRewriteReleaseBody' -v && task check:goreleaser && go test ./internal/upgrade/... && task release:dry-run` | ✅ | ✅ green |
| 01-03-01 | 03 | 2 | REL-06 | T-01-11…16, T-01-29 / T-01-SC | `goreleaser release` has exactly one definition site (Taskfile), so CI cannot drift from local | auto/shape | `go test ./internal/upgrade/... && task --list \| rg -q 'release:goreleaser' && task lint:actions` | ✅ | ✅ green |
| 01-03-02 | 03 | 2 | REL-06, REL-07 | T-01-11, T-01-29 (EoP), T-01-13…15 (Tampering) | `id-token: write` scoped to the single GoReleaser job; no hand-rolled `sha256sum` step can disagree with `checksum:` | tdd/shape | `go test ./internal/upgrade/... -run 'TestDarwinLegsBuildNatively\|TestOIDCWriteScopedToSingleGoreleaserJob\|TestNoHandRolledChecksumStepInReleaseWorkflow\|TestParseReleaseJobShapes_NoJobsIsError\|TestWorkflowSourceHelpersFailLoudly' -v && go test ./internal/upgrade/... && task lint:actions && task check:goreleaser` | ✅ | ✅ green |
| 01-04-01 | 04 | 3 | REL-08 | T-01-17…21 / T-01-SC | Provenance attestor is a SHA-pinned native action; `id-token: write` remains single-job scoped | auto/shape | `go test ./internal/upgrade/... -run 'TestProvenanceAttestorIsPinnedNativeAction\|TestParseAttestStep_NoAttestStepIsError\|TestOIDCWriteScopedToSingleGoreleaserJob\|TestWorkflowSourceHelpersFailLoudly' -v && go test ./internal/upgrade/... && task lint:actions` | ✅ | ✅ green |
| 01-04-02 | 04 | 3 | REL-08 | T-01-17 (Repudiation), T-01-20 (Spoofing) | No published instruction names a verifier command that architecturally cannot verify the shipped attestation format | auto/doc | `rg -c 'slsa-verifier verify-artifact' docs/RELEASE.md docs/RELEASE-PROCEDURES.md SECURITY.md README.md .planning/REQUIREMENTS.md; go test ./... && task lint:actions` | ✅ | ✅ green |
| 01-05-01 | 05 | 4 | REL-06, REL-08 | T-01-17…21 | Post-release verification re-downloads **published** assets; integrity verified before the binary is made executable | auto/shape | `task lint:actions && go test ./internal/upgrade/... && task --list \| rg -q 'verify:release-assets' && task --list \| rg -q 'verify:self-upgrade'` | ✅ | ✅ green |
| 01-05-02 | 05 | 4 | REL-06, REL-08 | — | Merge is a one-way door; tag/release must not be re-cut afterward (D-12) | checkpoint:decision | — (no automated command; human merge decision) | n/a | 📋 manual-only |
| 01-05-03 | 05 | 4 | REL-08 | T-01-18 (Tampering), T-01-20 (Spoofing) | REL-08 claims re-verify against the real published release, not a local `dist/` copy | auto | `go test ./... && task lint:actions` | ✅ | ✅ green |
| 01-06-01 | 06 | 3 | REL-06, REL-08 | T-01-34 (Spoofing) | Sign and SBOM pipes emit **four distinct** published names — asserted from `dist/artifacts.json`, never a green exit code | auto/e2e | `task --list \| rg -q 'release:dry-run-signed' && go test ./internal/upgrade/... && task release:dry-run-signed` | ✅ | ✅ green |
| 01-06-02 | 06 | 3 | REL-06 | T-01-34 (Spoofing) | The sign-pipe proof keeps re-firing after the phase closes, via a permanent canary job | auto/workflow | `task lint:actions && go test ./internal/upgrade/... && rg -c 'sign-snapshot' .github/workflows/linux-cross-canary.yml` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky · 📋 manual-only*

*Populated by `/gsd-plan-phase` task breakdown and completed by `/gsd-validate-phase`.*

---

## Wave 0 Requirements

- [x] New canary workflow (D-03) exercising zig-cross-from-macOS for both linux legs and **executing** the resulting binaries on real Linux — `namespace-profile-linux-amd64-4x8` + `namespace-profile-linux-arm64-4x8`, **both already provisioned** — the REL-05 spike itself
- [x] New Taskfile target(s) for the D-06 `goreleaser release --snapshot --skip=publish,sign` dry run
- [x] New Go shape tests in `internal/upgrade/` for: D-11's single-job `id-token: write` scope, D-14's `.sigstore.json` template contract, REL-07's hand-rolled-`sha256sum`-absence assertion
- [x] Rewrite of `TestDarwinLegsBuildNatively` per D-13's new invariant
- [x] Rewrite of `TestProvenanceJobUsesTaggedSLSAGenerator` (or its replacement) for the `actions/attest-build-provenance` job shape, per D-10 — landed as `TestProvenanceAttestorIsPinnedNativeAction`
- [x] New post-release automated self-upgrade job/workflow (D-08) — `.github/workflows/post-release-verify.yml` (`resolve-tag`, `verify-supply-chain`, `self-upgrade`, `gatekeeper`, `notarized-suite`)
- ~~New Namespace linux-arm64 runner profile~~ — **not a gap.** `default-arm64`, `linux-arm64-4×8`, and `linux-arm64-2×4` are already provisioned on the account `[VERIFIED: maintainer dashboard screenshot, 2026-08-08]`; the arm64 leg is a `runs-on:` value, covered by the canary-workflow item above

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| FAIL-bar variation list enumerated before the first spike run | REL-05 | Planning deliverable (D-04), not executable code — the list defines what counts as spike failure and must exist *before* evidence is gathered to avoid post-hoc goalpost-moving | Record the enumerated variations in the plan/PR body prior to the first canary dispatch; a third party must be able to re-read them against the recorded run. **Landed in-tree:** `.github/workflows/linux-cross-canary.yml:17` carries the D-04 FAIL-BAR VARIATION LIST header, with `V1` referenced at the two `mlugg/setup-zig` call sites |
| Architecture decision recorded on re-inspectable evidence (zig-cross success, or GoReleaser Pro adoption with the three named gate repairs) | REL-05 | Human judgment on a one-way-door decision; the *inputs* are automated (canary run output) but the recorded decision is not | Attach the canary run URL + fixture-indexing output to the decision record; if either leg fails, record GoReleaser Pro adoption with `check:goreleaser`/DIST-01, `TestGoreleaserPinParity`, and `tool-vuln`/VULN-01-02-03 entered as scope |
| Canary dispatch + REL-05 architecture decision (task 01-01-03) | REL-05 | The execution proof needs live `namespace-profile-linux-amd64-4x8` and `namespace-profile-linux-arm64-4x8` runners — real Linux hardware of each architecture, never emulation. Not reproducible on a local darwin host | `workflow_dispatch` `.github/workflows/linux-cross-canary.yml`; require a `REL05-EVIDENCE` line from **both** exec jobs carrying non-zero `fileCount`/`nodeCount`. A green exit code, a successful link, or a `--version` print is explicitly not evidence |
| Merge the `feat(release):` PR (task 01-05-02) | REL-06, REL-08 | One-way door — D-12 forbids re-cutting or deleting the tag afterward, so the decision cannot be automated away | Confirm plans 01-01…01-04 and 01-06 are all green and `release:dry-run-signed` reported `count=4 distinct=4` for both pipes before merging |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 13 of 15; the 2 exceptions are declared checkpoints recorded in Manual-Only
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 3s (local shape suite: 0.44s)
- [ ] `nyquist_compliant: true` set in frontmatter — **not set.** Two human checkpoints (real-hardware canary dispatch, one-way merge decision) have no automated command by design

**Approval:** approved 2026-08-11 (PARTIAL — 13 automated, 2 manual-only)

---

## Validation Audit 2026-08-11

| Metric | Count |
|--------|-------|
| Gaps found | 0 MISSING · 2 manual-only |
| Resolved | 0 (no test generation required — every automatable task already carried a green automated command) |
| Escalated | 2 (tasks 01-01-03 and 01-05-02, both declared checkpoints; moved to Manual-Only) |

**Re-execution evidence (all commands run verbatim at HEAD `42427c5`, darwin/arm64, 2026-08-11):**

| Command | Result |
|---------|--------|
| `go test ./internal/upgrade/... -count=1` | exit 0, 0.436s |
| `go test ./...` | exit 0, 47 packages ok, 0 FAIL |
| 6 plan-declared `-run` selections, `-v`, with a positive ran-count assertion | ran 2/9/2/2/13/12 tests, 0 failures |
| `task check:goreleaser` | exit 0 |
| `task lint:actions` | exit 0 |
| `task release:dry-run` | exit 0 — four binaries from one invocation: ELF x86-64, ELF aarch64, Mach-O x86_64, Mach-O arm64 |
| `task release:dry-run-signed` | exit 0 — `SIGN-EVIDENCE count=4 distinct=4`, `SBOM-EVIDENCE count=4 distinct=4` |
| `task --list` target presence (7 targets) | all present |
| `rg -c 'namespace-profile-linux-arm64-4x8' .github/workflows/linux-cross-canary.yml` | 1 |
| `rg -c 'sign-snapshot' .github/workflows/linux-cross-canary.yml` | 4 |
| `rg 'slsa-verifier verify-artifact'` across the 5 published docs | 2 hits, both inside explicitly-labelled pre-migration historical notes; current instructions name `gh attestation verify` |

**Method notes.**

1. **Every `-run` selection was re-run with a positive ran-count assertion,** not just an exit code. `go test -run <pattern>` prints `ok` and exits 0 when the pattern matches **zero** tests — a negative-only guard that passes vacuously the moment a test is renamed. Counting `=== RUN Test` lines is the positive assertion that the named tests actually executed.
2. **One plan-declared test name no longer exists:** plan 01-02's `TestBinarySignsSidecarMatchesUpgradeContract`. This is **not** a Phase 1 gap — Phase 2 plan 02-02 moved cosign from the build-scoped `binary_signs:` pipe to the release-scoped `signs:` pipe (D-18), and the test was renamed `TestSignsSidecarMatchesUpgradeContract` with the assertion intact. The map above cites the current name.
3. **The dry runs validate today's config, not the Phase-1-era config.** `--snapshot` resolves the version from the working tree (`v0.8.0`), so these runs prove the Phase 1 invariants still hold four phases later — which is the point of a retroactive validation. Both runs left `git status` empty (`dist/` is gitignored).
4. **`01-VERIFICATION.md`'s two closed gaps were re-checked on disk,** not taken on trust: `docs/RELEASE.md` names `v0.5.1` (5 occurrences, 0 remaining `<first-migrated-release-tag>` placeholders) and `docs/RELEASE-PROCEDURES.md` carries the `v0.5.1`/`SIGN-03` baseline entry (3 occurrences).
