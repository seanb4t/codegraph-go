---
phase: 08-release-hardening-benchmarks
plan: 04
subsystem: infra
tags: [ci-cd, release, cosign, slsa, sbom, github-actions, supply-chain]

# Dependency graph
requires:
  - phase: 08-release-hardening-benchmarks (plan 01)
    provides: ".goreleaser.yaml build config (goreleaser build --single-target per matrix cell)"
provides:
  - ".github/workflows/release.yml — tag-triggered native 2-OS matrix build + per-binary cosign sign + syft SBOM + SLSA3 provenance + gh release publish"
affects: [08-05, 08-06, 08-07, 08-08, 08-09, internal/upgrade (consumer of the published asset+sigstore.json contract)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Native 2-OS runner matrix (ubuntu-latest for linux+windows via zig cc, macos-latest natively for both darwin arches) instead of a single-runner zig-cc-everywhere approach — avoids the darwin libresolv/DNS cross-link risk documented in 08-RESEARCH.md Finding 2"
    - "Per-binary cosign sign-blob --bundle=<binary>.sigstore.json (NOT signing only checksums.txt) — the shipped internal/upgrade verifier binds the signature to each binary's own sha256, so per-binary bundles are the mandatory contract, not an enhancement"
    - "SLSA provenance via the *generic* generator (generator_generic_slsa3.yml) over already-built artifacts rather than the Go-specific builder, because the Go SLSA builder can't accommodate the CGo cross-build"
    - "Every third-party Action pinned to a full commit SHA with a trailing semver comment (e.g. actions/checkout@df4cb1c...  # v6.0.3) — no floating major tags anywhere in the workflow"

key-files:
  created:
    - .github/workflows/release.yml
  modified: []

key-decisions:
  - "Task 3 (checkpoint:human-verify, gate=blocking) was auto-approved under --auto pipeline mode rather than executed as a real CI validation. This is recorded honestly below and in REQUIREMENTS.md — the real 6-target signed release, darwin DNS resolution, and live OIDC cert-SAN check have NOT been performed and remain a pending action item for the maintainer before the workflow can be trusted end-to-end."

requirements-completed: [DIST-01]

coverage:
  - id: T1
    description: "Native 2-OS runner matrix builds all 6 raw binaries with contract-matching names (codegraph_<tag>_<os>_<arch>[.exe]); YAML parses; no floating Action major tags; darwin cells use no zig"
    requirement: "DIST-01"
    verification:
      - kind: unit
        ref: "python3 -c \"import yaml; yaml.safe_load(open('.github/workflows/release.yml'))\" && ! grep -nE 'uses:.*@v[0-9]+$' .github/workflows/release.yml"
        status: pass
    human_judgment: false
  - id: T2
    description: "Assembly job signs each binary individually via cosign sign-blob producing per-binary .sigstore.json; SLSA generic generator pinned to a full vX.Y.Z semver; publish step uploads binaries + sigstore.json + spdx.json + checksums; no GoReleaser Pro directive is invoked"
    requirement: "DIST-02, DIST-03"
    verification:
      - kind: unit
        ref: "grep -q 'sign-blob' .github/workflows/release.yml && grep -qE 'generator_generic_slsa3\\.yml@v[0-9]+\\.[0-9]+\\.[0-9]+' .github/workflows/release.yml && grep -q 'build --single-target' .github/workflows/release.yml"
        status: pass
    human_judgment: false
  - id: T3
    description: "A real signed 6-target release publishes, the darwin binary resolves DNS, and the cosign cert SAN matches releaseWorkflowRefPattern — validated by cutting a pre-release tag and observing the release workflow run to completion"
    requirement: "DIST-02, DIST-03"
    verification:
      - kind: human
        ref: "checkpoint:human-verify, gate=blocking — AUTO-APPROVED under --auto pipeline mode, NOT independently verified against a real CI run"
        status: pending
    human_judgment: true

# Metrics
duration: 2min (Tasks 1+2 authoring); Task 3 real-CI validation still outstanding
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 04: Release Workflow (Native Matrix + Cosign + SLSA + SBOM) Summary

**Tag-triggered `.github/workflows/release.yml` authored and locally validated (YAML parse, no floating Action pins, per-binary `sign-blob`, SLSA generator pinned to full semver, no GoReleaser Pro features) — the first real signed 6-target release with darwin DNS + live OIDC verification is DEFERRED to an actual pre-release tag push and has NOT been proven in this session.**

## Performance

- **Duration:** Task 1 committed `ee258d9` at 2026-07-13T14:00:46-04:00; Task 2 committed `bc685a3` at 2026-07-13T14:01:31-04:00 (~45s apart — authoring, not the full CI cycle)
- **Tasks:** 3 total (2 auto tasks executed + committed; 1 checkpoint task auto-approved this session)
- **Files modified:** 1 created (`.github/workflows/release.yml`, 295 lines total across both commits)

## Accomplishments

- **Task 1** (`ee258d9`): `name: release`, trigger scoped to `on: push: tags: ['v[0-9]*']` only — matches `releaseWorkflowRefPattern` in `internal/upgrade/verify.go`. A native 2-OS matrix (`ubuntu-latest` → linux/amd64 native gcc, linux/arm64 + windows/amd64 + windows/arm64 via pinned `mlugg/setup-zig`; `macos-latest` → darwin/arm64 + darwin/amd64 both natively, no zig) runs `goreleaser build --single-target --clean` per cell, renames the output to the exact `codegraph_<tag>_<os>_<arch>[.exe]` contract shape, and uploads via pinned `actions/upload-artifact`. Every Action pinned to a full commit SHA with a semver comment.
- **Task 2** (`bc685a3`): An assembly job (`ubuntu-latest`, `id-token: write` + `contents: write`) downloads all 6 artifacts, computes `codegraph_<tag>_checksums.txt`, installs cosign via pinned `sigstore/cosign-installer` and runs `cosign sign-blob --bundle="${f}.sigstore.json" --yes "$f"` **once per binary** (never only over the checksums file — Finding 1), installs syft via pinned `anchore/sbom-action/download-syft` for per-binary `.spdx.json` SBOMs, and publishes everything via `gh release create`. A separate job invokes `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0` (full semver, not a short tag) over the checksums.
- **Task 3** (checkpoint): Auto-approved under `--auto` pipeline mode per continuation instructions. See "Deferred Real-CI Validation" below — this is explicitly NOT a substitute for the real validation the checkpoint asks for.

## Task Commits

Each auto task was committed atomically:

1. **Task 1: Native 2-OS build matrix producing all 6 raw binaries** - `ee258d9` (feat)
2. **Task 2: Assembly job — cosign signing, SBOM, SLSA provenance, gh release publish** - `bc685a3` (feat)

**Task 3 (checkpoint):** No code commit — bookkeeping only (this SUMMARY + STATE/ROADMAP/REQUIREMENTS updates), see below.

## Files Created/Modified

- `.github/workflows/release.yml` — full tag-triggered release workflow: build matrix job (Task 1) + assembly job + SLSA provenance job (Task 2)

## Decisions Made

- Followed the plan's native 2-OS matrix over the single-Linux-runner `zig cc`-everywhere fallback, per 08-RESEARCH.md Finding 2 (darwin CGo cross-link DNS-resolver risk).
- Task 3's checkpoint was auto-approved to unblock pipeline progression, but the real validation it asks for (an actual tagged CI run) is recorded as a pending action item, not as completed work. No fabricated CI evidence is claimed anywhere in this summary.

## Deviations from Plan

None - Tasks 1 and 2 executed exactly as written (verified: `ee258d9` and `bc685a3` exist in git history, `.github/workflows/release.yml` exists with the expected content). Task 3's disposition (auto-approve vs. real validation) is not a deviation — it is the documented behavior of `--auto` pipeline mode for `checkpoint:human-verify` tasks, explicitly called out as a pending item rather than silently treated as done.

## Issues Encountered

None during authoring. The unresolved item is not an "issue" in the bug sense — it is unproven-by-design: real CGo cross-compilation, cosign OIDC signing, and SLSA provenance generation can only happen inside a real GitHub Actions run against a pushed tag, which this session cannot execute.

## Deferred Real-CI Validation (READ BEFORE TRUSTING THIS RELEASE PATH)

**What IS verified (local, static analysis only):**
- `.github/workflows/release.yml` exists at the correct path with `name: release` and trigger `on: push: tags: ['v[0-9]*']` only.
- `grep -c 'sign-blob'` shows per-binary signing (not checksums-only).
- `generator_generic_slsa3.yml@v2.1.0` — full semver, not a short tag.
- `goreleaser build --single-target` is the only GoReleaser invocation — no Pro-only subcommand (`release --split`, `continue --merge`, `prebuilt`).
- No Action is pinned to a floating major tag — every `uses:` line carries a full commit SHA with a semver comment.
- The darwin build cells configure no zig/cross-linker step (native `macos-latest` runner only).

**What is NOT verified — deferred to an actual tagged release, exactly as a `human_needed` UAT item:**
1. Push a pre-release tag (`git tag v0.0.0-rc.1 && git push origin v0.0.0-rc.1`) and watch the `release` workflow run to completion in GitHub Actions.
2. Confirm the release contains 6 raw binaries named `codegraph_v0.0.0-rc.1_<os>_<arch>[.exe]`, each with a sibling `<binary>.sigstore.json`, plus `.spdx.json` SBOMs, a checksums file, and SLSA provenance (`*.intoto.jsonl`).
3. On a real macOS machine, download the darwin binary and run `codegraph upgrade --check` (or equivalent network call) to confirm DNS resolution works — the specific failure mode Finding 2 warns about.
4. Run `slsa-verifier verify-artifact` against a binary using the generated provenance, and exercise `cosign verify-blob`/bundle verification against a `.sigstore.json`.
5. Confirm the cosign cert SAN matches `release.yml@refs/tags/v0.0.0-rc.1` — i.e. that `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` actually accepts a real signature produced by this workflow.

**No claim is made anywhere in this summary that steps 1-5 have passed.** They have not been run. DIST-02 and DIST-03 remain `Pending` in `.planning/REQUIREMENTS.md` for this reason — the workflow authoring is complete and locally sound, but "every release artifact is cosign-signed... and users can verify" (DIST-02) and "every release publishes an SBOM" (DIST-03) are claims about real release behavior that only a real tagged CI run can substantiate.

## User Setup Required

**Action needed before this release path can be trusted:** push a pre-release tag per step 1 above and work through steps 2-5. This is a genuine manual/CI action, not something automatable from this session (no `id-token: write` OIDC context, no real GitHub Actions runner, no darwin hardware available here).

## Next Phase Readiness

- `.github/workflows/release.yml` is authored, locally validated, and ready to fire on the next `v[0-9]*` tag push.
- DIST-01 was already marked Complete (from Plan 08-01's build-matrix config). DIST-02 and DIST-03 remain Pending pending the real-CI validation checklist above — do not mark them Complete until a maintainer confirms steps 1-5.
- No blockers for continuing to the next plan in this phase; the pending real-CI validation is independent of subsequent plans' work.

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: .github/workflows/release.yml
- FOUND: ee258d9 (Task 1 commit)
- FOUND: bc685a3 (Task 2 commit)
