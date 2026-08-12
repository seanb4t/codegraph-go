# Phase 1: Cross-Compile Spike & `goreleaser release` Migration - Pattern Map

**Mapped:** 2026-08-08
**Files analyzed:** 11
**Analogs found:** 11 / 11

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.github/workflows/linux-cross-canary.yml` (new, D-03 spike/canary) | workflow (CI) | event-driven (`workflow_dispatch`) | `.github/workflows/darwin-toolchain-canary.yml` | exact |
| `.github/workflows/release.yml` (rewritten to one job) | workflow (CI) | event-driven / batch | itself (in-place rewrite) — topology precedent: `darwin-toolchain-canary.yml`'s single-job shape | role-match |
| `.goreleaser.yaml` (`archives:`, `checksum:`, `binary_signs:`, `sboms:` blocks activated/added) | config | transform (declarative build pipeline) | itself — the existing `builds:`/`archives:`/`checksum:` blocks (already drafted, currently dead) | exact |
| `Taskfile.yml` — new `release:dry-run` / `check:linux-cross` targets (D-06) | config (task runner) | batch (local dry-run invocation) | `check:darwin-release-build` (D-06 explicitly "mirrors" it) | exact |
| `internal/upgrade/release_workflow_shape_test.go` — new/rewritten test funcs (`TestDarwinLegsBuildNatively` rewrite, new `id-token` single-job test, provenance-job-shape replacement) | test (workflow-shape) | transform (parse YAML → assert invariant) | itself — existing `TestDarwinLegsBuildNatively`, `TestProvenanceJobUsesTaggedSLSAGenerator`, `parseReleaseBuildMatrix`/`parseReleaseProvenanceJob` helpers | exact |
| `internal/upgrade/goreleaser_shape_test.go` (or similar new file, D-14 sidecar-name contract test) | test (config-shape) | transform (parse `.goreleaser.yaml` → assert against `releaseAssetName()`) | `release_workflow_shape_test.go`'s helper/test pairing pattern (`parseX`/`mustX`) | role-match |
| `internal/upgrade/release.go` (unchanged — read-only contract) | service (naming contract) | transform (pure string template) | itself — `releaseAssetName()` in `internal/upgrade/upgrade.go:209` (n.b. RESEARCH.md/CONTEXT.md mislabel this as living in `release.go`; it is in `upgrade.go`) | exact |
| `internal/upgrade/verify.go` (unchanged — read-only contract) | service (crypto verification) | transform (cosign-bundle verify, no SLSA code path) | itself | exact |
| `docs/RELEASE.md` §65-69 (D-10 rewording) | doc | transform | itself (in-place edit) | exact |
| `docs/RELEASE-PROCEDURES.md` §224 (D-10 rewording) | doc | transform | itself (in-place edit) | exact |
| `.github/workflows/*.yml` post-release self-upgrade job (D-08) | workflow (CI) | event-driven (post-release) → request-response (download+verify+upgrade) | `assemble` job's `gh release` steps in `release.yml` (env-not-`${{ }}` interpolation, `set -euo pipefail`, GH_TOKEN pattern) | role-match |

## Pattern Assignments

### `.github/workflows/linux-cross-canary.yml` (workflow, event-driven)

**Analog:** `.github/workflows/darwin-toolchain-canary.yml` (73 lines, full file read)

**Header-comment contract pattern** (lines 1-23): every canary workflow opens with a prose block naming (a) what correctness property is at risk, (b) why it's not observable via the tag-only `release.yml` trigger, (c) why the failure mode is "quiet" (builds/signs/attests cleanly but is wrong). The new linux-cross canary must open with the equivalent for zig-cross-from-macOS + CGo tree-sitter parsing correctness (Pitfall 3 in RESEARCH.md is the source material for this prose).

**Trigger pattern** (lines 26-36):
```yaml
on:
  workflow_dispatch:
  pull_request:
    paths:
      - ".github/workflows/release.yml"
      - ".github/workflows/darwin-toolchain-canary.yml"
      - ".goreleaser.yaml"
      - "Taskfile.yml"
      - "go.mod"
      - "go.sum"
```
D-03 requires the new canary to be `workflow_dispatch`-triggerable at minimum (dispatchable, permanent). Extend the `paths:` list with the new canary's own filename and any zig-pin-bearing file.

**Permissions pattern** (lines 37-39):
```yaml
permissions:
  contents: read
```
Least-privilege — this canary does not publish anything, so no `id-token:`/`attestations:`/`contents: write`.

**Job/step shape** (lines 40-73): `runs-on:` the exact same profile string used in `release.yml` (D-01's `namespace-profile-macos-6x14-tahoe` for the macOS half; D-02's Namespace linux profiles for the execution-proof half), checkout pinned to the same SHA, `actions/setup-go` pinned to the same SHA, `./.github/actions/install-task` composite action, then one `task check:*` invocation per narrow-failure-surface step — never inline shell in the workflow file itself. Comment explaining *why* steps are kept separate (lines 64-71) — apply the same "a red run should say WHICH layer broke" rationale when splitting the new canary's zig-setup step from its build step from its real-Linux-execution step.

---

### `.github/workflows/release.yml` (workflow, single-job collapse)

**Analog:** itself, current 375-line 3-job version (full file read)

**Header LOCKED-CONTRACT comment pattern** (lines 1-44): must be preserved and updated in place — the `releaseWorkflowRefPattern` SAN-anchor warning (lines 3-16) is untouched by this migration (D-11 confirms the pattern is keyed to file+ref, not job/runner) but the collapse to one job must be reflected in the prose, and the per-third-party-Action SHA-pinning convention note (lines 39-43) must be updated once the SLSA generic generator's tag-pin exception (line 43, "except the SLSA generic generator...") is removed (D-09 replaces it with `actions/attest-build-provenance`, which should be SHA-pinned like every other Action).

**`env:` version-pin pattern** (lines 55-63): `GORELEASER_VERSION` held equal to `go.tool.mod`'s pin, machine-enforced by `TestGoreleaserPinParity`. Unchanged by this migration — the single job still needs this `env:` block.

**Zig install step** (lines 131-135):
```yaml
- name: Install zig (linux/arm64 cross target only)
  if: matrix.needs_zig
  uses: mlugg/setup-zig@d1434d08867e3ee9daa34448df10607b98908d29 # v2.2.1
  with:
    version: "0.15.1"
```
Under the collapsed single job, drop the `if: matrix.needs_zig` conditional (D-02: both linux legs now cross via zig from the one macOS runner) — always install zig once, unconditionally, before the goreleaser step.

**`env:`-not-`${{ }}` interpolation convention** (lines 153-160, 240-241, 283-284, 298-301): every shell step that touches workflow context passes it via a step `env:` block and references `$VAR` inside `run:` — never inline `${{ }}` into the script body. Any new shell step this phase adds (the D-06 dry-run Taskfile invocation's CI caller, the D-08 post-release job) must follow this exactly.

**Artifact lookup via `dist/artifacts.json` with `find` fallback** (lines 171-183, mirrored in `Taskfile.yml` lines 321-329): the stable-interface + defensive-fallback pattern to reuse anywhere the new single-invocation job needs to locate a GoReleaser-produced binary/artifact by (goos, goarch).

**What gets deleted, not migrated** (Pitfall 4 / REL-07): the `assemble` job's hand-rolled `sha256sum` (line 244), the per-binary `cosign sign-blob` shell loop (lines 249-264), the per-binary `syft` shell loop (lines 269-278), and the "Base64-encode checksums for SLSA provenance" step (lines 280-287) must all be deleted **in the same change** that activates `.goreleaser.yaml`'s `checksum:`/`binary_signs:`/`sboms:` blocks — landing the deletion and the activation separately creates a window where checksums are written twice or not at all.

**`assemble` job's `permissions:` block** (lines 219-224) is the direct precedent for D-11's single-job `id-token: write` shape test — the collapsed job's `permissions:` block should look structurally identical (`contents: write`, `id-token: write`, plus new `attestations: write` per the `actions/attest-build-provenance` code example in RESEARCH.md) with the same load-bearing comment explaining what the OIDC token is used for.

**Provenance job being replaced** (lines 337-376): the entire `provenance:` job (SLSA generic generator, `base64-subjects:`, `private-repository: true`) is deleted and replaced by an `actions/attest-build-provenance` step inside the single collapsed job, per RESEARCH.md's Code Examples section (`subject-checksums: ./dist/codegraph_${{ github.ref_name }}_checksums.txt`, needs `attestations: write` added to `permissions:`).

---

### `.goreleaser.yaml` (config, transform)

**Analog:** itself (full 124-line file read) — the target-state blocks are already drafted and annotated as currently-dead.

**Existing `builds:` blocks** (lines 46-106) are unchanged — 4 entries, each with its own `goos:`/`goarch:`/`env:` (`CC=zig cc -target ...` already present for `codegraph-linux-arm64` at lines 70-73). Per RESEARCH.md Pattern 5/Pitfall 2, this phase must also add the equivalent `CC=zig cc -target x86_64-linux-gnu` / `CXX=zig c++ -target x86_64-linux-gnu` `env:` pair to `codegraph-linux-amd64` (lines 47-63) — today that build id has no CC/CXX override because it's built natively on a linux runner; under D-01/D-02 it becomes a zig-cross target too.

**Dead-config header comment (a)** (lines 3-21) must be rewritten: it currently asserts `archives:`/`checksum:` "are NOT executed by this project's pipeline" because `release.yml` only runs `goreleaser build --single-target`. That sentence becomes false the moment this migration lands `goreleaser release`; the comment must be updated to state the new single-source-of-truth contract instead of the old dead-config warning.

**Existing dead `archives:`/`checksum:` blocks to activate and extend** (lines 108-125):
```yaml
archives:
  - formats: [binary]
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}

checksum:
  name_template: "{{ .ProjectName }}_{{ .Tag }}_checksums.txt"
  algorithm: sha256
```
Per D-15, split this single `archives:` entry into two `id:`-keyed entries (`raw` keeping `formats: [binary]` byte-unchanged, `zip` adding `formats: [zip]`) exactly per RESEARCH.md's Code Examples full block (Pattern 1) — the `raw` entry's `name_template` stays character-identical to what's here today. Add `checksum.ids: [raw, zip]` (D-12) rather than relying on GoReleaser's default inclusion set.

**New blocks with no prior draft in this file** — `binary_signs:` (D-14) and `sboms:` (D-17) — copy verbatim from RESEARCH.md's Code Examples section (already vetted against official GoReleaser docs); there is no existing local analog since signing/SBOM were previously hand-rolled entirely in `release.yml`'s `assemble` job (see release.yml lines 249-278 above for the shell-loop behavior these blocks must reproduce declaratively).

---

### `Taskfile.yml` — new dry-run/cross-check targets (D-06)

**Analog:** `check:darwin-release-build` (`Taskfile.yml:270-347`, full block read)

**Precondition-gating pattern** (lines 286-290):
```yaml
preconditions:
  - sh: '[ "$(go env GOHOSTOS)" = "darwin" ]'
    msg: "check:darwin-release-build must run on a native darwin host — that is the property it exists to verify (D-08). Run it on macOS locally, or dispatch the darwin-toolchain-canary workflow."
  - sh: command -v clang
    msg: "clang not found. Install Xcode or the Command Line Tools: xcode-select --install"
```
The new D-06 target should gate on `GOHOSTOS = darwin` the same way (it's the runner class the single collapsed job runs on) and on `command -v zig` (or the setup-zig-installed binary) rather than `clang`.

**Building GoReleaser natively before invoking it, load-bearing ordering comment** (lines 292-310): the Rosetta/arm64-xcrun trap this comment documents (`GOOS`/`GOARCH` must never leak into the *tool* build, only into the target build) is exactly RESEARCH.md's Pitfall 2 territory — copy this ordering pattern (`go build -modfile=go.tool.mod -o "${GORELEASER_BIN}" github.com/goreleaser/goreleaser/v2` built with no GOOS/GOARCH override, then invoke that binary per-target) into the D-06 target.

**`--snapshot` usage rationale comment** (lines 282-285) — reuse verbatim reasoning: a PR/dispatch HEAD carries no tag, GoReleaser hard-fails without `--snapshot`; D-06 additionally needs `--skip=publish,sign` since this is `goreleaser release`, not `build --single-target`.

**Mach-O arch assertion pattern** (lines 335-344) — if D-06's dry run should also assert produced binaries are the right architecture (not just that the invocation exits 0 — this echoes RESEARCH.md's "don't trust a green exit code" anti-pattern), reuse the `file -b "${BIN}"` + `case` pattern.

**Existing `check:cross` for `TestCheckCrossMatchesGoreleaserTargets` precedent** (`Taskfile.yml:354-368`): if a `check:linux-cross`-named target is added (per the deferred question "does check:cross gain a linux-cross sibling"), model it on this block's `go list -mod=readonly` sweep style, and note this target's own machine-held set-equality test (`TestCheckCrossMatchesGoreleaserTargets`) as the shape-test precedent for asserting the new target's platform list matches `.goreleaser.yaml`.

---

### `internal/upgrade/release_workflow_shape_test.go` — new/rewritten shape tests

**Analog:** itself — `parseReleaseBuildMatrix`/`mustReleaseBuildMatrix` (lines 292-359), `TestDarwinLegsBuildNatively` (lines 452-484), `parseReleaseProvenanceJob`/`mustReleaseProvenanceJob` (lines 382-442), `TestProvenanceJobUsesTaggedSLSAGenerator` (lines 486-506) — full sections read.

**Established helper-pair pattern** (documented at lines 20-26): every new test in this phase follows `parseX(src string) (T, error)` returning a non-nil error (never a silent zero value) on absence, plus a `mustX(t *testing.T, src string) T` wrapper that calls `t.Fatalf`. Apply this exactly for:
- D-13's rewrite of `TestDarwinLegsBuildNatively` — the matrix-based `releaseMatrixEntry`/`parseReleaseBuildMatrix` shape (lines 292-359) goes away entirely once the build matrix collapses to one job; the new invariant needs a different parse target — "the job invoking goreleaser runs on a darwin-class runner" (reuse `isMacOSClassRunner`, lines 361-380, unchanged) AND "linux build ids in `.goreleaser.yaml` carry a `CC=zig cc` override" (a NEW parser reading `.goreleaser.yaml`, not `release.yml` — this is a change of source file for this specific test).
- D-11's new `id-token: write`-single-job shape test: model directly on `parseReleaseProvenanceJob`'s job-boundary-scanning technique (find job start via a `^  <jobname>:\s*$` regex, find job end via the next top-level job regex, scan lines in between) but generalized to scan **every** job in the file and assert exactly one has `id-token: write` in its `permissions:` block, and that job also invokes goreleaser (grep its `run:`/`uses:` steps).
- D-10's `TestProvenanceJobUsesTaggedSLSAGenerator` replacement: same job-boundary-scan technique, retargeted at whatever job now contains the `actions/attest-build-provenance` step, asserting it's SHA-pinned (not tag-pinned, unlike the old SLSA generator exception) with a trailing version comment, matching every other third-party Action in this file.

**Regex-based "tagged reference" assertion pattern** (line 448, `slsaGeneratorTaggedRe`) — same style regex-match-on-a-`uses:`-string technique applies to asserting `actions/attest-build-provenance`'s SHA pin format.

**Demonstrated-RED discipline** (`TestDarwinLegsBuildNatively`'s doc comment, lines 452-456): every new/rewritten test in this phase must be demonstrated RED against a confirmed-applied mutation (standing project decision) — flip the runner to ubuntu, drop the zig `CC=` override, remove `id-token: write`, etc., and confirm the test fails before calling it done.

---

### `internal/upgrade/goreleaser_shape_test.go` (new, D-14 sidecar-name contract)

**Analog:** same file's helper-pair pattern (`release_workflow_shape_test.go` lines 20-26) plus `internal/upgrade/upgrade.go:202-211`'s `releaseAssetName()` as the value under test.

**Core pattern to copy:**
```go
// internal/upgrade/upgrade.go:209-211
func releaseAssetName(version string) string {
	return fmt.Sprintf("codegraph_%s_%s_%s", version, runtime.GOOS, runtime.GOARCH)
}
```
The new test computes `releaseAssetName(<tag>) + ".sigstore.json"` for all four (goos, goarch) platform pairs (mirroring `codegraph-linux-amd64`/`codegraph-linux-arm64`/`codegraph-darwin-amd64`/`codegraph-darwin-arm64` build ids in `.goreleaser.yaml`) and asserts `.goreleaser.yaml`'s `binary_signs[0].signature` template (`"${artifact}.sigstore.json"`, per RESEARCH.md Code Examples) resolves to exactly that string for each. Use the same `parseX`/`mustX` + demonstrated-RED-by-perturbing-the-template discipline as the sibling shape tests. Read `.goreleaser.yaml` off disk the same way `release_workflow_shape_test.go` reads `releaseWorkflowPath` (`os.ReadFile` + a package-relative const path).

---

### `internal/upgrade/release.go` / `verify.go` — read-only contracts (no code changes)

**Analog:** themselves — confirm-only reads.

`internal/upgrade/upgrade.go:209-211`'s `releaseAssetName()` (note: NOT in `release.go` despite CONTEXT.md/RESEARCH.md's file reference — `release.go` (113 lines, fully read) contains only `resolveLatestVersion`/`resolveLatestVersionViaAPI`, no asset-naming code) is the naming contract this phase's `.goreleaser.yaml` `archives:`/`binary_signs:` `name_template`/`signature` fields must agree with byte-for-byte.

`internal/upgrade/verify.go:41-45`'s `releaseWorkflowRefPattern` constant is the SAN-anchor contract D-11 proves against, unchanged by this phase — confirmed to contain zero SLSA/in-toto references (D-10's de-risking fact), so the attestor swap touches nothing here.

Neither file is edited by this phase; they are read-only anchors that new shape tests (above) assert config *elsewhere* still satisfies.

---

## Shared Patterns

### Third-party Action SHA-pinning with trailing version comment
**Source:** `.github/workflows/release.yml` lines 114, 121, 127, 133, 138, 232, 247, 267 (every `uses:` line)
**Apply to:** the new `linux-cross-canary.yml` workflow, and the `actions/attest-build-provenance` step added to the collapsed `release.yml` job.
```yaml
- uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
```
Resolve the exact tag → full commit SHA at implementation time (RESEARCH.md notes `actions/attest-build-provenance`'s pin is not yet confirmed — run `gh api repos/actions/attest-build-provenance/releases/latest` first).

### `env:`-not-`${{ }}` shell interpolation
**Source:** `.github/workflows/release.yml` lines 153-160 (comment + example), reused at 240-241, 283-284, 298-301
**Apply to:** every new shell `run:` step this phase adds (D-06's Taskfile-caller step if any, D-08's post-release self-upgrade job, any new canary step).
```yaml
env:
  TAG: ${{ github.ref_name }}
  GOOS: ${{ matrix.goos }}
  GOARCH: ${{ matrix.goarch }}
run: |
  set -euo pipefail
  DEST="codegraph_${TAG}_${GOOS}_${GOARCH}"
```

### `dist/artifacts.json`-first, `find(1)`-fallback artifact lookup
**Source:** `.github/workflows/release.yml` lines 171-183; mirrored in `Taskfile.yml` lines 321-329
**Apply to:** any new step (D-06 dry-run target, D-08 self-upgrade job, new shape tests reading GoReleaser output) that needs to locate a built binary by (goos, goarch) — never assume a stable `dist/` subdirectory layout.

### Workflow-shape test helper pattern (`parseX`/`mustX`, non-nil-error-on-absence)
**Source:** `internal/upgrade/release_workflow_shape_test.go` lines 20-26 (documented convention), exemplified throughout (`parseWorkflowTopLevelName`/`mustWorkflowTopLevelName` lines 28-51, `parseReleaseBuildMatrix`/`mustReleaseBuildMatrix` lines 306-359, `parseReleaseProvenanceJob`/`mustReleaseProvenanceJob` lines 393-442)
**Apply to:** every new Go test file/function this phase adds (D-11, D-13's rewrite, D-14, REL-07's hand-rolled-step-absence assertion).
```go
func parseX(src string) (T, error) {
    // ... return a real error, never a usable zero value, when target absent
}
func mustX(t *testing.T, src string) T {
    t.Helper()
    v, err := parseX(src)
    if err != nil {
        t.Fatalf("mustX: %v", err)
    }
    return v
}
```

### Demonstrated-RED-before-trusted gate discipline
**Source:** project standing decision (PROJECT.md, echoed in `release_workflow_shape_test.go`'s doc comments at lines 452-456, 486-490)
**Apply to:** every new test in this phase (D-11, D-13 rewrite, D-14, REL-09's `formats: [binary]` mutation-test). Flip the property under test, confirm the test goes red, then restore.

## No Analog Found

None. Every file this phase touches has a direct or role-matching precedent already in the repository — this phase is explicitly scoped as CI/config migration reusing established patterns (canary-workflow shape, workflow-shape-test helper pattern, Taskfile dry-run-target shape), per the phase-specific guidance.

## Metadata

**Analog search scope:** `.github/workflows/`, `.goreleaser.yaml`, `Taskfile.yml`, `internal/upgrade/`
**Files scanned:** `.github/workflows/release.yml` (375 lines, full read), `.github/workflows/darwin-toolchain-canary.yml` (73 lines, full read), `.goreleaser.yaml` (124 lines, full read), `Taskfile.yml` (targeted reads: lines 1-42, 241-348), `internal/upgrade/release.go` (113 lines, full read), `internal/upgrade/verify.go` (114 lines, full read), `internal/upgrade/upgrade.go` (targeted read: lines 180-220), `internal/upgrade/release_workflow_shape_test.go` (targeted reads: lines 1-120, 280-520)
**Pattern extraction date:** 2026-08-08
