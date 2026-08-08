# Phase 1: Cross-Compile Spike & `goreleaser release` Migration - Research

**Researched:** 2026-08-08
**Domain:** CI/CD release pipeline engineering — GitHub Actions, GoReleaser v2, CGo cross-compilation, Sigstore/cosign, GitHub artifact attestations
**Confidence:** MEDIUM (config-shape claims verified against official GoReleaser docs; runtime behavior of zig-cross-from-macOS and the arm64 execution leg is exactly what REL-05's spike measures — no amount of research substitutes for that measurement)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Spike Venue & Evidence (REL-05)**

- **D-01:** The spike — and therefore the migrated single-runner pipeline — runs on
  `namespace-profile-macos-6x14-tahoe`, the runner `release.yml` and
  `darwin-toolchain-canary.yml` already use. This corrects ROADMAP.md and PROJECT.md, which
  both say `macos-latest`; that prose is wrong about this repository and should be read as
  "one macOS runner."
- **D-02:** The "runs on real Linux" half of the pass condition executes on **Namespace Linux
  profiles** — the existing `namespace-profile-linux-amd64-4x8` for amd64, and a **new Namespace
  linux-arm64 profile** for arm64. No emulation. This repository has no arm64 execution runner
  today. Standing up that profile is new infrastructure inside this phase, not a reused job.
- **D-03:** The spike ships as a **permanent, dispatchable canary workflow**, mirroring
  `.github/workflows/darwin-toolchain-canary.yml` (`workflow_dispatch`, same runner family). Not
  a throwaway.
- **D-04:** The FAIL bar is **bounded and enumerated before the first run**, not open-ended
  debugging. Minimum variation list: zig version (pinned `0.15.1` plus one newer), glibc target
  triple with an explicit floor (e.g. `x86_64-linux-gnu.2.28` / `aarch64-linux-gnu.2.28`), and
  static-vs-dynamic `CGO_LDFLAGS`. Exhausting the list declares FAIL and triggers the costed
  GoReleaser Pro fallback.

**Producing the Verification Release (REL-08)**

- **D-05:** The migration PR is titled **`feat(release): …`** (squash-only repo,
  `squash_merge_commit_title=PR_TITLE` — release-please parses the PR title, not commit
  subjects). D-06R untouched — release-please still owns tagging.
- **D-06:** Before the first real release, a **new Taskfile target** runs
  `goreleaser release --snapshot --skip=publish,sign` on a native macOS host, mirroring
  `check:darwin-release-build`.
- **D-07:** Recovery posture is **patch forward**. Never delete a published release or tag.
- **D-08:** REL-08's third claim (a genuinely shipped prior binary self-upgrading) is an
  **automated post-release job**, not a manual runbook step.

**Job Topology & Attestation**

- **D-09:** **One job.** `actions/attest-build-provenance` replaces
  `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0`
  entirely. One-way: releases under the two attestors carry different provenance formats and
  builder identities.
- **D-10:** The switch is **unconditional**, not gated on research. In scope: rewriting
  `TestProvenanceJobUsesTaggedSLSAGenerator`, `docs/RELEASE.md` §65-69,
  `docs/RELEASE-PROCEDURES.md` §224, `SECURITY.md`, `README.md`, and the wording of REL-08 itself
  (currently names `slsa-verifier verify-artifact`). If research finds `slsa-verifier` does not
  accept native attestations, the verification command becomes `gh attestation verify` and REL-08
  is reworded — accepted, not a blocker. De-risking fact: `internal/upgrade/verify.go` has zero
  SLSA/in-toto references; `codegraph upgrade` is indifferent to which attestor is used.
- **D-11:** The cosign SAN is proven, not assumed, by (a) live `cosign verify-blob` against a
  re-downloaded published asset, and (b) a new shape test asserting exactly one job in
  `release.yml` carries `id-token: write` and that it is the job invoking goreleaser.
- **D-12:** Checksums and attestation subjects cover **8 payloads** — 4 raw binaries + 4 `.zip`
  archives. Excluded: `.sigstore.json`/`.spdx.json` sidecars and the checksums file itself.
- **D-13:** `TestDarwinLegsBuildNatively` is **rewritten to the new invariant**: the job invoking
  goreleaser runs on a darwin runner AND the linux build ids carry a `zig cc` `CC`/`CXX`
  override. Demonstrated RED by flipping the runner to ubuntu.

**GoReleaser Config Shape**

- **D-14:** The `<asset>.sigstore.json` sidecar contract is held by a static PR-time unit test
  computing `internal/upgrade.releaseAssetName() + ".sigstore.json"` for all four platforms and
  asserting `.goreleaser.yaml`'s `binary_signs.signature` template resolves to exactly that,
  demonstrated RED by perturbing the template.
- **D-15:** The archive is `codegraph_<tag>_<goos>_<goarch>.zip` — same stem as the raw asset plus
  `.zip`. Two `archives:` entries keyed by `id`; the raw entry stays `formats: [binary]`.
- **D-16:** The `.zip` contains binary + LICENSE + README (GoReleaser's conventional default).
  Completions and man pages stay out.
- **D-17:** SBOMs stay per-binary, preserving today's `<asset>.spdx.json` names. Zips get no
  separate SBOM.

### Claude's Discretion

None claimed in the discussion — every question received an explicit answer. Four items were
routed to research (this document) rather than to the maintainer:

1. Whether a Namespace **linux-arm64** profile is available on this account, and its exact
   profile label (D-02 depends on it). **See Environment Availability — this is an
   account/dashboard action, not something this research can resolve from the repo.**
2. Whether `mlugg/setup-zig` runs on the Namespace macOS image, and whether zig's macOS host
   support covers `x86_64-linux-gnu` as well as `aarch64-linux-gnu` for CGo. **See Common
   Pitfalls / Sources — zig ships official macOS aarch64 binaries and treats Linux glibc targets
   as first-class regardless of host; the specific Namespace image was not independently
   confirmed.**
3. Whether GoReleaser's `checksum:` block can be scoped to exclude sidecar artifacts, and
   whether `binary_signs:` supports an exact `${artifact}.sigstore.json` signature template
   (D-12, D-14). **Resolved — see Code Examples: both are natively supported.**
4. Whether `actions/attest-build-provenance` can attest a multi-subject list the way the generic
   generator parses each `<sha256>  <name>` line of `checksums.txt`, and whether
   `slsa-verifier verify-artifact` accepts its output. **Resolved — see Architecture Patterns /
   Pitfall 1: yes to the first half, no to the second half; a third option
   (`slsa-verifier verify-github-attestation`) exists beyond the two D-10 named.**

### Deferred Ideas (OUT OF SCOPE)

- DIST-04 double-build determinism under a changed host (whether zig-crossing from macOS instead
  of building natively on Linux still produces a byte-identical double build, and what becomes
  of `check:reproducibility:arm64`).
- Where the zig version pin lives once zig is needed on the macOS host for both Linux legs.
- Whether `check:cross`, `check:darwin-toolchain`, `check:darwin-release-build` survive as-is,
  merge, or gain a linux-cross sibling.
- `GO-2026-5932` revisit (accepted-unmitigated openpgp exposure in goreleaser's own binary) — may
  be revisited on evidence, not scheduled.
- Backlog bookkeeping question (999.3/999.6) — unrelated maintainer call.
- Wire oracle `toolslist-repeat` flake — explicitly not folded into this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-05 | Decide pipeline architecture on measured evidence: `zig cc` cross-compiles CGo tree-sitter to linux/amd64 **and** linux/arm64 from a macOS host, proven by the resulting binaries running on real Linux, not by build exit 0 | Architecture Patterns (single-runner topology), Common Pitfalls 1-3 (zig-from-macOS specifics, glibc floor, GOOS/GOARCH env trap), Environment Availability (arm64 runner gap) |
| REL-06 | A release is cut by a single `goreleaser release` invocation, GoReleaser owns archive/checksum/sign/SBOM | Architecture Patterns (topology diagram), Code Examples 1 (archives+checksum+binary_signs+sboms full config) |
| REL-07 | Exactly one process writes `codegraph_<tag>_checksums.txt` | Code Examples 1 (checksum block), Common Pitfalls 4 (hand-rolled step deletion ordering) |
| REL-08 | Every supply-chain claim still verifies against real published assets: `cosign verify-blob`, SLSA/attestation verification, `codegraph upgrade` self-upgrade | Architecture Patterns (attestation format incompatibility finding), Code Examples 2-3, Common Pitfalls 1 |
| REL-09 | Release carries raw binaries (byte-unchanged) and `.zip` archives | Code Examples 1 (dual `archives:` entries by id), Don't Hand-Roll (archive naming/format_overrides) |
</phase_requirements>

## Summary

This phase collapses a proven, working three-job pipeline (`build` matrix → `assemble` →
`provenance`) into a single `goreleaser release` invocation on one macOS runner, and the entire
risk surface is concentrated in one unverified fact: **whether `zig cc`, invoked from a macOS
host, can cross-compile this project's CGo tree-sitter dependency to linux/amd64 and linux/arm64
and produce binaries that actually run on real Linux** (not just link). Everything else in this
phase — activating GoReleaser's already-written `archives:`/`checksum:` blocks, adding
`binary_signs:`/`sboms:` blocks, swapping the SLSA generic generator for
`actions/attest-build-provenance` — is config-shape work with direct, verified documentation
support and low technical risk. The project has already done the harder research (Phase-8/10-era)
establishing that `goreleaser release` refuses a foreign `dist/` and that GoReleaser Pro's split/
merge and prebuilt-builder escape hatches are the only alternative; this phase's job is to prove
or disprove that the OSS single-runner path is *reachable*, on measured evidence, before any
config work is trusted.

Two research findings materially affect the plan. First, `slsa-verifier verify-artifact` **does
not** accept attestations produced by `actions/attest-build-provenance` — they are architecturally
different formats, and `slsa-verifier` ships a separate, dedicated subcommand
(`verify-github-attestation`) for the native format that D-10 did not name. `gh attestation
verify` (D-10's stated fallback) remains the simpler, more idiomatic choice and is what
GoReleaser's own official docs recommend pairing with `actions/attest`, but the plan should decide
explicitly between the two rather than silently picking one at implementation time. Second,
`goreleaser release` builds each `builds:` entry using *that entry's own* `goos:`/`goarch:` list —
it does not read a caller-supplied `GOOS`/`GOARCH` environment variable the way
`goreleaser build --single-target` does. The current `build` job's "Build single target" step sets
`GOOS`/`GOARCH` as step `env:`; that must be removed (or become a no-op) under the migrated
single-invocation shape, or CGo cross-compilation for the wrong host-target pairing could
silently short-circuit.

**Primary recommendation:** Treat REL-05 as a real spike with an enumerated FAIL bar (D-04) before
touching any GoReleaser config. Stand up the Namespace linux-arm64 profile via the dashboard
first — it is an external, no-code dependency with lead time, not implementation work, and it
blocks the "real Linux execution" half of the pass condition. Once the spike passes, the
`.goreleaser.yaml` config changes are additive and well-documented (activate dead `archives:`/
`checksum:` blocks already in the correct shape, add `binary_signs:` and `sboms:` blocks) — the
job is porting proven config vocabulary, not inventing new mechanism.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cross-compile CGo binaries (linux/amd64, linux/arm64) | CI / Build tooling (macOS runner, zig cc) | — | GoReleaser's `builds:` pipe invokes the Go toolchain with per-build `CC`/`CXX` env; the runner is the compute tier, GoReleaser is the orchestration layer on top of it |
| Native darwin binary compilation | CI / Build tooling (macOS runner, Xcode clang) | — | Unchanged from today — darwin never cross-links (D-13/libresolv risk) |
| Archive/checksum/sign/SBOM generation | CI / Release tooling (GoReleaser pipes) | — | Moves from hand-rolled shell (today) into GoReleaser's own pipes — this is the core REL-06/07 migration |
| Build provenance attestation | CI / GitHub-native (`actions/attest-build-provenance`) | — | Replaces a third-party reusable workflow with a GitHub-first-party Action; still a CI-tier concern, not application code |
| Release-asset naming contract | CI (GoReleaser template) ↔ Application (`internal/upgrade`) | Application (`internal/upgrade.releaseAssetName()`) | Two independent implementations of the same string must agree — this is the load-bearing cross-tier contract the phase must not silently break |
| Signature/attestation verification | Application (`internal/upgrade/verify.go`, in-process cosign-bundle verify) | External CLI (`cosign verify-blob`, `gh attestation verify`, human/CI verification) | `codegraph upgrade` verifies in-process at install time; the external CLI commands in `docs/RELEASE.md` are for a human or CI to independently re-prove the same claim post-publish |
| Real-Linux execution proof (REL-05) | CI / Execution runner (Namespace linux-amd64 + new linux-arm64 profiles) | — | A build exiting 0 proves nothing about runtime correctness on the target OS/arch; only an execution runner proves it |

## Standard Stack

### Core

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|---------------|
| GoReleaser | v2.17.1 `[VERIFIED: go.tool.mod:38,234]` | Build/archive/checksum/sign/SBOM/release orchestration | Already the project's chosen release tool (DIST-01); this phase activates config it already partially writes |
| `mlugg/setup-zig` | pinned `d1434d08867e3ee9daa34448df10607b98908d29` (`v2.2.1` tag), zig `0.15.1` `[VERIFIED: .github/workflows/release.yml:131-135]` | Installs the zig toolchain used as `CC`/`CXX` for the linux/arm64 CGo cross-build | Already in the build graph for one leg (linux/arm64); the spike extends it to run from a macOS host and adds the linux/amd64 leg |
| `sigstore/cosign-installer` + `cosign` CLI | pinned `6f9f17788090df1f26f669e9d70d6ae9567deba6` (`v4.1.2` tag) `[VERIFIED: .github/workflows/release.yml:247]` | Keyless per-binary signing (`sign-blob --bundle`) | Already in use; migrates from a hand-rolled shell loop into GoReleaser's `binary_signs:` pipe calling the same `cosign` binary |
| `actions/attest-build-provenance` | latest `v3.x` line as of research (verify exact tag at implementation time — GitHub Marketplace lists it as a thin wrapper over `actions/attest` as of "v4") `[ASSUMED — WebSearch only, not cross-checked against an authoritative release page]` | Native GitHub build-provenance attestation, replacing the SLSA generic generator | D-09/D-10 maintainer decision — GitHub-native attestor, less custom infra than a third-party reusable workflow |

### Supporting

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `anchore/sbom-action/download-syft` | pinned `e22c389904149dbc22b58101806040fa8d37a610` (`v0.24.0`) `[VERIFIED: .github/workflows/release.yml:267]` | SBOM generation binary, invoked by GoReleaser's `sboms:` pipe (`cmd: syft`) instead of a hand-rolled loop | Already in use — this phase changes the *caller* (GoReleaser pipe vs. shell `for` loop), not the tool |
| `gh` CLI (`GH_TOKEN`) | GitHub-hosted, ambient | Release asset upload/verification, `gh attestation verify` | Already the publish mechanism; may become the sole post-migration verification command for REL-08 depending on the D-10 wording decision (see Pitfall 1) |
| GitHub CLI `attestation` subcommand | ships with `gh` | Verify `actions/attest-build-provenance` output | Documented replacement candidate for `slsa-verifier verify-artifact` if that tool proves incompatible (it does — see Pitfall 1) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `actions/attest-build-provenance` | `slsa-framework/slsa-github-generator` (status quo) | Already rejected by the maintainer (D-09) — more custom infrastructure, third-party reusable workflow, less first-party support |
| `gh attestation verify` for REL-08 | `slsa-verifier verify-github-attestation` | A real, documented, released subcommand exists specifically for GitHub-native attestations (distinct from `verify-artifact`). Keeps `slsa-verifier` as the named tool in REL-08's wording with a smaller diff, but adds a second CLI dependency (`slsa-verifier` binary) beyond `gh`, which ships everywhere already. `gh attestation verify` is simpler and matches D-10's stated fallback — recommend it unless the plan wants to preserve `slsa-verifier` as the verification tool name for continuity with existing docs |
| GoReleaser OSS single-runner (this phase's subject) | GoReleaser Pro (`release --split`/`continue --merge`, `prebuilt` builder) | Already the named, costed fallback (PROJECT.md) if REL-05's spike fails — not evaluated further here per D-04's enumerate-then-exhaust design |

**Installation:** No new Go module dependencies. This phase is entirely CI/config surface
(`.goreleaser.yaml`, `.github/workflows/*.yml`, `Taskfile.yml`, docs) plus new Go test files in
`internal/upgrade/`. No `go get`/`npm install` step applies.

**Version verification:** GoReleaser v2.17.1 confirmed live in `go.tool.mod:38,234`
`[VERIFIED: go.tool.mod]`. `actions/attest-build-provenance`'s exact current tag was **not**
independently confirmed against an authoritative release page in this session — the plan must run
`gh api repos/actions/attest-build-provenance/releases/latest` (or check the GitHub Marketplace
listing) before pinning a SHA, per this repo's own convention of pinning every third-party Action
to a full commit SHA with a trailing version-tag comment.

## Package Legitimacy Audit

Not applicable. This phase adds zero new Go module dependencies and zero new package-manager
installs — it is CI workflow YAML, `.goreleaser.yaml` configuration, `Taskfile.yml` targets, and
new Go test files exercising already-vendored tooling (`goreleaser`, `cosign`, `syft`, `gh`, all
already present in the pipeline). The one new third-party surface is a GitHub Action
(`actions/attest-build-provenance`), which is first-party GitHub tooling, not a registry package,
and is out of scope for the npm/PyPI/crates legitimacy-check protocol.

## Architecture Patterns

### System Architecture Diagram

**Today (3 jobs, 2 runner classes):**

```
tag push (v[0-9]*)
  │
  ▼
┌─────────────────────────────────────────────────────────────┐
│ build (matrix: 4 legs, 2 runner classes)                     │
│  linux/amd64  ─┐ namespace-profile-linux-amd64-4x8            │
│  linux/arm64  ─┤ (zig cc cross for arm64 leg only)             │
│  darwin/amd64 ─┐ namespace-profile-macos-6x14-tahoe            │
│  darwin/arm64 ─┘ (native Xcode clang, no cross)                │
│  each: goreleaser build --single-target → upload-artifact      │
└─────────────────┬──────────────────────────────────────────┘
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ assemble  (namespace-profile-linux-amd64-2x4)                 │
│  download 4 artifacts → sha256sum (hand-rolled) →              │
│  cosign sign-blob per binary → syft per binary →                │
│  base64-encode checksums → gh release upload/create             │
└─────────────────┬──────────────────────────────────────────┘
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ provenance (slsa-github-generator reusable workflow, no        │
│  runs-on override possible) — attests over checksums.txt       │
│  subjects, uploads .intoto.jsonl to the release                │
└─────────────────────────────────────────────────────────────┘
```

**After migration (1 job, 1 runner class — pending REL-05 spike passing):**

```
tag push (v[0-9]*)
  │
  ▼
┌─────────────────────────────────────────────────────────────┐
│ release (namespace-profile-macos-6x14-tahoe, single job)      │
│                                                                 │
│  1. checkout + setup-go + setup-zig (macOS host)                │
│  2. task release:goreleaser  →  goreleaser release --clean      │
│     ├─ builds: 4 entries, each with its OWN goos/goarch/CC/CXX  │
│     │    linux/amd64  (CC=zig cc -target x86_64-linux-gnu)      │
│     │    linux/arm64  (CC=zig cc -target aarch64-linux-gnu)     │
│     │    darwin/amd64 (native clang)                            │
│     │    darwin/arm64 (native clang)                            │
│     ├─ archives:  2 entries per build id (raw binary, .zip)     │
│     ├─ checksum:  codegraph_<tag>_checksums.txt (8 payloads)    │
│     ├─ binary_signs: cosign sign-blob --bundle per artifact      │
│     ├─ sboms: syft per binary → <asset>.spdx.json                │
│     └─ release: gh-equivalent publish (GoReleaser's own)         │
│  3. actions/attest-build-provenance (subject-checksums:          │
│     dist/checksums.txt) — one native attestation, 8 subjects     │
└─────────────────┬──────────────────────────────────────────┘
                   ▼
        published release: 4 raw binaries + 4 .zips +
        1 checksums.txt + 8 .sigstore.json + 8 .spdx.json +
        1 native attestation (GH Attestations API, not a
        .intoto.jsonl release asset)
```

**Downstream, unchanged:**
```
codegraph upgrade  ──▶ downloads codegraph_<tag>_<goos>_<goarch>
                        + .sigstore.json ──▶ verifyRelease()
                        (in-process, cosign-bundle only,
                         no SLSA/attestation code path)
```

### Pattern 1: Multiple `archives:` entries keyed by `id`, same build ids, different formats

**What:** GoReleaser natively supports declaring two (or more) `archives:` blocks that both
reference the same `builds`/`ids` set but produce different output shapes — one `formats:
[binary]` (raw, uncompressed, directly executable — what `internal/upgrade` downloads) and one
`formats: [zip]` (a real archive containing the binary plus `LICENSE`/`README` by default).
**When to use:** REL-09 — shipping both a raw binary and a `.zip` for the same platform without
disturbing the raw asset's name or byte shape.
**Example:**
```yaml
# Source: https://goreleaser.com/customization/package/archives (Context7, official docs)
archives:
  - id: raw
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [binary]
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}

  - id: zip
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [zip]
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}
    # default files: LICENSE*, README*, CHANGELOG — matches D-16 exactly, no
    # explicit `files:` override needed
```
Note: the doc field is `ids:` (plural, replacing the deprecated `builds:` field as of GoReleaser
v2.8) — this repo is on v2.17.1, well past that deprecation, so `ids:` is the correct field name,
not `builds:`.

### Pattern 2: `binary_signs:` with an exact sidecar-name template (D-14)

**What:** `binary_signs:` (distinct from `signs:`) signs binaries regardless of archive format,
and its `signature` field accepts an arbitrary template — it is not locked to the default
`${artifact}_{{ .Os }}_{{ .Arch }}...` shape.
**When to use:** Producing the exact `<binary-name>.sigstore.json` sidecar name
`internal/upgrade`'s `defaultDownload` already expects (`assetName + ".sigstore.json"`).
**Example:**
```yaml
# Source: https://goreleaser.com/customization/sign/binary_sign,
#         https://goreleaser.com/customization/sign (Context7, official docs)
binary_signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: binary
```

### Pattern 3: Per-binary `sboms:` preserving `<asset>.spdx.json` (D-17)

**What:** The `sboms:` pipe defaults `artifacts:` to `archive`, not `binary` — this must be set
explicitly or the SBOM pipe will catalog the archive outputs (including the new `.zip`s) instead
of matching today's per-binary `.spdx.json` contract.
**Example:**
```yaml
# Source: https://goreleaser.com/customization/sbom (Context7, official docs)
sboms:
  - id: binary-sbom
    artifacts: binary
    documents:
      - "${artifact}.spdx.json"
    cmd: syft
    args: ["$artifact", "--output", "spdx-json=$document"]
```

### Pattern 4: Scoping `checksum:` to exactly 8 payloads (D-12)

**What:** GoReleaser's default checksum inclusion is "all published binaries, archives, linux
packages, and source archives" — `.sigstore.json` and `.spdx.json` sidecars are pipe *outputs*,
not archive/binary/package artifacts, so they are excluded by default without needing an
`ids:` filter. No source archive is published in this project's config (no `source:` block), so
the default set should already resolve to exactly the 4 raw binaries + 4 zips. Recommend making
this explicit anyway via `checksum.ids` referencing the two archive ids from Pattern 1, so the
"exactly 8" invariant is asserted by config, not by absence-of-configuration.
**Example:**
```yaml
# Source: https://goreleaser.com/customization/package/checksum (Context7, official docs)
checksum:
  name_template: "{{ .ProjectName }}_{{ .Tag }}_checksums.txt"
  algorithm: sha256
  ids: [raw, zip]   # explicit — matches the two archives: entries in Pattern 1
```

### Pattern 5: `goreleaser release` builds per-entry `goos`/`goarch`, not caller `GOOS`/`GOARCH`

**What:** Unlike `goreleaser build --single-target` (today's mechanism, which requires the caller
to pin `GOOS`/`GOARCH` as step `env:` so GoReleaser knows which single target to build), a full
`goreleaser release` invocation iterates every `builds:` entry using **that entry's own**
`goos:`/`goarch:` lists (confirmed by the general `builds:` reference example, which declares
`goos: [linux, darwin, windows]` per build id with no external `GOOS` dependency).
**When to use:** The single-invocation migration (REL-06). The current `build` job's "Build
single target" step sets `env: GOOS: ${{ matrix.goos }} / GOARCH: ${{ matrix.goarch }}` — under
the migrated single-job shape, that pattern disappears entirely (no per-leg matrix), and no
`GOOS`/`GOARCH` env should be set on the one `goreleaser release` invocation; each of the 4
existing `builds:` entries in `.goreleaser.yaml` already declares its own `goos:`/`goarch:`
(confirmed: `.goreleaser.yaml:50-51,68-69,86-87,99-100`) and needs no change on that axis.

### Anti-Patterns to Avoid

- **Setting `GOOS`/`GOARCH` job env on the migrated `goreleaser release` step:** carried over by
  habit from the old `--single-target` shape, this is at best a no-op and at worst confusing
  during debugging — GoReleaser resolves targets from `builds:` config, not the environment, once
  running `release`/`build` in full-matrix mode.
- **Trusting a green `goreleaser build`/`release` exit code as REL-05 evidence:** the phase's own
  success criterion says this explicitly — a build exiting 0 proves the linker accepted the
  cross-compiled object, not that the resulting binary executes correctly on the target OS/glibc,
  which is exactly the class of failure zig-cross + external-C-scanner tree-sitter grammars can
  produce silently.
- **Verifying attestations against a local `dist/` copy:** REL-08 requires re-downloading
  published assets. Local `dist/` files never pass through the real GitHub Releases upload path,
  Sigstore transparency log, or attestation subject binding — verifying against them can pass
  even when the actually-published asset is broken.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Checksums file generation | A `sha256sum ... > file.txt` shell step (today's `assemble` job does this) | GoReleaser's `checksum:` pipe | This is REL-07's entire point — two writers of the same filename is the bug being fixed, and GoReleaser's own pipe is already correctly shaped in the "dead" config |
| Per-binary signing loop | A hand-rolled `for f in codegraph_*; do cosign sign-blob ...; done` (today's `assemble` job) | `binary_signs:` pipe | GoReleaser iterates the right artifact set automatically and keeps the signature template as declarative config instead of shell string-matching (`case "$f" in *_checksums.txt ...`) |
| Per-binary SBOM loop | A hand-rolled `for f in codegraph_*; do syft ...; done` | `sboms:` pipe with `artifacts: binary` | Same shell-vs-declarative tradeoff; also removes the risk of the shell loop's exclusion `case` drifting out of sync with the checksum/sign loops' exclusion cases (three independent copies of the same glob-exclusion logic exist today) |
| Multi-subject attestation subject list construction | Manually base64-encoding `<sha256> <name>` lines (today's `assemble` job's "Base64-encode checksums for SLSA provenance" step) | `actions/attest-build-provenance`'s `subject-checksums:` input, pointed at GoReleaser's own `dist/checksums.txt` | GoReleaser's own docs recommend exactly this pairing (`customization/attestations`); the action parses the shasum-format file itself, no custom encoding step needed |
| Release asset publish + idempotent re-run | A `gh release create`/`gh release upload --clobber` branch (today's `assemble` job) | GoReleaser's own `release:` pipe (built into `goreleaser release`) | GoReleaser's release pipe already handles the release-please-created-release-exists-vs-doesn't-exist cases and asset replacement; verify its `--clobber`-equivalent behavior during D-06's `--snapshot` dry run before removing the hand-rolled branch entirely |

**Key insight:** almost everything this phase's job-topology work does is *deleting* hand-rolled
shell that duplicates functionality GoReleaser's own pipes already implement, correctly, in
config the project wrote a year ago and annotated as currently inert ("dead configuration").
That inversion — the target state is less code, not more — is the main reason this phase's
config-shape risk is low. The spike (REL-05) is the one piece with no such precedent to lean on.

## Common Pitfalls

### Pitfall 1: `slsa-verifier verify-artifact` cannot verify `actions/attest-build-provenance` output

**What goes wrong:** REL-08 as currently worded names `slsa-verifier verify-artifact` as a
required passing check. After the D-09/D-10 attestor swap, that command will fail against every
real release — not because anything is broken, but because it is architecturally the wrong
verifier for the new attestation format.
**Why it happens:** `slsa-verifier verify-artifact` is built to verify provenance from "the SLSA
generator" or Google Cloud Build — GitHub-native attestations from `actions/attest-build-
provenance` are a structurally different format, produced and consumed through GitHub's own
Attestations API/transparency log rather than a standalone `.intoto.jsonl` file. `slsa-verifier`
recognizes this and ships a **separate, dedicated subcommand**, `verify-github-attestation`, with
its own flag set (`--attestation-path`, `--builder-id`, `--source-uri`, and a positional
artifact/module-file argument) specifically for this case `[CITED: github.com/slsa-framework/slsa-verifier README]`.
**How to avoid:** D-10 already anticipated and accepted this outcome — REL-08's wording changes.
The plan should decide explicitly between two real options rather than defaulting silently:
(a) `gh attestation verify <asset> -R seanb4t/codegraph-go` (D-10's stated fallback, simplest,
matches GoReleaser's own documented pairing recommendation), or (b)
`slsa-verifier verify-github-attestation --attestation-path <bundle> --source-uri
github.com/seanb4t/codegraph-go <asset>` (keeps `slsa-verifier` as the named tool, requires
first downloading the attestation bundle via `gh attestation download`, one extra step).
Recommend (a) for simplicity and because it needs no extra CLI dependency beyond `gh`, which is
already a hard runtime dependency of the `assemble`→`release` job today.
**Warning signs:** A shape test or doc example that still says `slsa-verifier verify-artifact`
after the attestor swap lands is silently testing/documenting a command that will 100% fail
against the first real post-migration release.

### Pitfall 2: `goreleaser release`'s per-build `goos`/`goarch` vs. caller-set `GOOS`/`GOARCH`

**What goes wrong:** Carrying over the current job's `env: GOOS: ${{ matrix.goos }}` pattern into
the single migrated job could mask a misconfiguration (e.g. if the migrated step still sets a
single `GOOS`/`GOARCH` pair, only that one target would build under some invocation modes, or the
setting becomes silently irrelevant while still present and confusing during review).
**Why it happens:** `goreleaser build --single-target` genuinely needs an external `GOOS`/`GOARCH`
signal because it's building exactly one target per invocation across 4 separate CI job runs.
`goreleaser release` does not have that problem — it already knows all 4 targets from
`.goreleaser.yaml`'s 4 `builds:` entries, each with its own `goos:`/`goarch:` list `[VERIFIED:
.goreleaser.yaml:50-51,68-69,86-87,99-100]`.
**How to avoid:** Remove the `GOOS`/`GOARCH` step-env entirely from the migrated single-invocation
job; verify via D-06's `--snapshot --skip=publish,sign` dry run that all 4 platforms actually
appear in `dist/artifacts.json` after the run.
**Warning signs:** `dist/artifacts.json` containing fewer than 4 binary entries after a
`goreleaser release`/`--snapshot` run.

### Pitfall 3: zig-cross-from-macOS is untested in this repo's own CI to date

**What goes wrong:** Today's `release.yml` zig-crosses linux/arm64 **from a linux/amd64 host**
(`namespace-profile-linux-amd64-4x8`). The spike's whole premise (D-01/D-02) is running that same
zig cross-compilation **from a macOS host** instead, plus adding the linux/amd64 leg as a second
zig cross-target (today linux/amd64 is native gcc on a linux runner, not zig at all). Neither of
those exact configurations has ever executed on this project's own infrastructure.
**Why it happens:** zig officially ships Linux glibc targets as first-class outputs from any
host including macOS (`zig cc -target x86_64-linux-gnu` / `aarch64-linux-gnu` are supported cross
targets regardless of host OS `[CITED: multiple community sources, cross-checked against zig's
own target-support model — this project's specific grammars/scanners were not tested by this
research]`), and `mlugg/setup-zig` fetches zig's own official prebuilt tarballs (zig itself
publishes `zig-macos-aarch64-<version>.tar.xz` officially `[CITED: ziglang.org/download]`), so
there is no structural reason this should fail — but this project's specific dependency set
includes tree-sitter grammars with hand-written external C/C++ scanners (per PARSER-DECISION.md
and the project's own CLAUDE.md "Parser Decision" section), which is exactly the category of code
most likely to behave differently under a different host/cross-toolchain pairing than under
native compilation. This is precisely why REL-05 is a measured spike and not an inference.
**How to avoid:** Follow D-04's enumerate-then-exhaust protocol literally — write the FAIL-bar
variation list (zig version, glibc floor, static/dynamic `CGO_LDFLAGS`) into the plan text before
the first run, and treat "the binary linked" and "the binary parsed a real fixture on real Linux
hardware for its actual target arch" as two separate, both-required checks.
**Warning signs:** A binary that builds and even runs `--version` successfully but segfaults or
returns wrong results specifically when the tree-sitter parse path is exercised — the failure
mode the "index a fixture, not `--version`" pass-condition language exists to catch.

### Pitfall 4: Deleting the hand-rolled checksum step in a different change than activating GoReleaser's

**What goes wrong:** REL-07's whole point is that `.goreleaser.yaml`'s `checksum:` block and
`release.yml`'s hand-rolled `sha256sum` step must never both be live at once — if the deletion and
the activation land in separate commits/PRs, there is a window where either both run (name
collision / non-deterministic overwrite) or neither runs (no checksums file at all, breaking
`docs/RELEASE.md`'s documented verification steps).
**Why it happens:** the natural instinct when de-risking a migration is to land it in small,
separately-reviewable steps — but this specific pair is a single atomic contract, not two
independent changes.
**How to avoid:** Land the checksum-block activation and the hand-rolled-step deletion in the same
change, exactly as REL-07's requirement text already specifies ("the hand-rolled `sha256sum` step
is deleted in the same change that makes `.goreleaser.yaml`'s `checksum:` block live").
**Warning signs:** A PR diff touching only `.goreleaser.yaml` or only `release.yml` for this
specific pair, not both.

## Code Examples

### Full activated `.goreleaser.yaml` additions (archives, checksum, binary_signs, sboms)

```yaml
# Source: https://goreleaser.com/customization/package/archives,
#         https://goreleaser.com/customization/package/checksum,
#         https://goreleaser.com/customization/sign/binary_sign,
#         https://goreleaser.com/customization/sbom  (Context7, official docs, MEDIUM confidence)
# Combines Patterns 1-4 above into one config block. `builds:` section unchanged
# from the existing 4 entries in this repo's .goreleaser.yaml.

archives:
  - id: raw
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [binary]
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}

  - id: zip
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [zip]
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}

checksum:
  name_template: "{{ .ProjectName }}_{{ .Tag }}_checksums.txt"
  algorithm: sha256
  ids: [raw, zip]

binary_signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: binary

sboms:
  - id: binary-sbom
    artifacts: binary
    documents:
      - "${artifact}.spdx.json"
    cmd: syft
    args: ["$artifact", "--output", "spdx-json=$document"]
```

### `actions/attest-build-provenance` step (replaces the `provenance:` job)

```yaml
# Source: https://goreleaser.com/customization/attestations (Context7, official docs)
# and https://github.com/actions/attest-build-provenance (WebSearch, cross-checked
# against GoReleaser's own recommended pairing) — MEDIUM confidence; verify the
# exact release tag/SHA to pin before landing (see Standard Stack note).
permissions:
  contents: write
  id-token: write
  attestations: write   # new permission this job needs — absent from today's `assemble` job

steps:
  # ... goreleaser release step ...
  - uses: actions/attest-build-provenance@<PIN-AT-IMPLEMENTATION-TIME>
    with:
      subject-checksums: ./dist/codegraph_${{ github.ref_name }}_checksums.txt
```

### Post-release self-upgrade proof job (D-08)

```yaml
# Illustrative shape only — not sourced from official docs, this is new
# project-specific automation composing existing pieces (internal/upgrade's
# CLI, a prior real release's asset, sha256 comparison). [ASSUMED shape,
# confirm exact `codegraph upgrade` flags against internal/upgrade/upgrade.go
# at implementation time — this file was read for its download/verify
# contract, not its full CLI flag surface, in this research session.]
- name: Prove self-upgrade against the just-published release
  run: |
    set -euo pipefail
    PRIOR_ASSET="codegraph_<prior-tag>_${GOOS}_${GOARCH}"
    curl -fsSL -o codegraph "https://github.com/${REPO}/releases/download/<prior-tag>/${PRIOR_ASSET}"
    chmod +x codegraph
    ./codegraph upgrade
    NEW_SHA=$(sha256sum codegraph | awk '{print $1}')
    EXPECTED_SHA=$(sha256sum "codegraph_${TAG}_${GOOS}_${GOARCH}" | awk '{print $1}')
    [ "$NEW_SHA" = "$EXPECTED_SHA" ] || { echo "::error::self-upgrade produced unexpected sha256"; exit 1; }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Third-party `slsa-framework/slsa-github-generator` reusable workflow for build provenance | GitHub-native `actions/attest-build-provenance` | Ongoing industry shift as GitHub's Artifact Attestations feature matured; this repo's own D-09 decision reflects that shift | Different provenance format, different verification command (`gh attestation verify` / `slsa-verifier verify-github-attestation`, not `slsa-verifier verify-artifact`); one-way for the shipped release line |
| Hand-rolled shell loops for checksum/sign/SBOM in a CI `assemble` job | GoReleaser's own `checksum:`/`binary_signs:`/`sboms:` pipes, invoked via `goreleaser release` | This repo wrote the target-state config over a year ago (annotated "dead" pending this exact migration) | Fewer independent glob-exclusion implementations to keep in sync; REL-07's checksum-collision bug class becomes structurally impossible rather than manually avoided |
| `archives.builds:` field (deprecated) | `archives.ids:` field | GoReleaser v2.8 `[CITED: goreleaser.com/deprecations]` | This repo (v2.17.1) must use `ids:`, not `builds:`, in any new/edited `archives:` entries |
| `signs:` with `artifacts: binary` (requires `archives.formats: binary`) | `binary_signs:` (format-independent) | Documented as the currently-recommended mechanism for signing binaries regardless of archive format `[CITED: goreleaser.com/customization/sign/binary_sign]` | Directly relevant here: once `.zip` archives coexist with raw binaries (REL-09), `signs:` with the binary-only constraint would no longer cleanly apply; `binary_signs:` is the correct primitive for this phase's dual-archive shape |

**Deprecated/outdated:**
- `archives.format:` (singular) → `archives.formats:` (plural list) — this repo's current
  `.goreleaser.yaml` still uses the singular `formats: [binary]` list-of-one form, which is the
  *plural* field already (not the deprecated singular string field) — no change needed there,
  confirmed by reading the file directly `[VERIFIED: .goreleaser.yaml:118]`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `actions/attest-build-provenance`'s current recommended version/tag (referenced as "v3.x" / "wrapper as of v4" in search results) | Standard Stack | Low — this is a pin-selection detail resolved trivially at implementation time via `gh api repos/actions/attest-build-provenance/releases/latest`, following the repo's own existing SHA-pinning convention. Does not affect architecture or config shape. |
| A2 | `mlugg/setup-zig` runs correctly on the specific `namespace-profile-macos-6x14-tahoe` image (as opposed to zig-the-toolchain supporting macOS in general, which is confirmed) | Common Pitfalls 3, Claude's Discretion item 2 | Medium — if the action itself has an OS-detection bug or the Namespace image lacks a dependency the action assumes, the spike's first run surfaces this immediately and cheaply (it's the very first step of D-03's canary), not a hidden late failure |
| A3 | zig's cross-compiled linux/amd64 and linux/arm64 outputs for *this project's specific* tree-sitter grammar set (including grammars with external C/C++ scanners, e.g. Swift, Kotlin) will link and run correctly | Pitfall 3, entire REL-05 premise | This is not a research gap to close — it is the exact fact REL-05's spike exists to measure. Flagging it here so the plan does not treat any research claim in this document as a substitute for running the spike. |
| A4 | GoReleaser's default `checksum:` inclusion set (binaries/archives/linux-packages/source-archives) will, in this project's specific config with no `source:` block, exclude `.sigstore.json`/`.spdx.json` sidecars without an explicit `ids:` filter | Pattern 4 / D-12 | Low — recommended mitigation (explicit `checksum.ids: [raw, zip]`) removes the dependency on this default behavior entirely; treat the explicit form as the plan's baseline, not the implicit default |
| A5 | `gh attestation verify` is the simpler/preferred REL-08 wording vs. `slsa-verifier verify-github-attestation` | Pitfall 1 | Low — both are real, both work; this is a stylistic/dependency-surface recommendation, not a correctness claim. The plan should make the choice explicit rather than defaulting silently. |

**A3 is the load-bearing one.** Every other assumption in this table is a minor implementation
detail resolvable in minutes at execution time. A3 is the entire reason this phase exists as a
"spike" rather than a straightforward config migration — treat REL-05's enumerated-FAIL-bar
protocol (D-04) as non-negotiable, not as due diligence to be abbreviated because the research
above sounds encouraging about zig's general Linux cross-compilation support.

## Open Questions

1. **Which REL-08 verification command replaces `slsa-verifier verify-artifact`: `gh attestation
   verify` or `slsa-verifier verify-github-attestation`?**
   - What we know: both are real, documented, working commands for verifying
     `actions/attest-build-provenance` output (Pitfall 1). D-10 named `gh attestation verify` as
     its accepted fallback if `slsa-verifier verify-artifact` proves incompatible — which this
     research confirms it does.
   - What's unclear: whether the maintainer has a preference for keeping `slsa-verifier` as the
     named tool in public-facing docs (continuity) vs. simplifying to `gh` alone (fewer
     dependencies for an external verifier to install).
   - Recommendation: default to `gh attestation verify` per D-10's own stated fallback unless the
     plan-checker or maintainer flags a reason to prefer `slsa-verifier verify-github-attestation`.

2. **Exact Namespace linux-arm64 profile label and current account availability.**
   - What we know: Namespace's custom-profile mechanism (`namespace-profile-*`, dashboard-created
     at `cloud.namespace.so/workspace/actions/profiles`) is how this repo's existing
     `namespace-profile-macos-6x14-tahoe` and `namespace-profile-linux-amd64-4x8` profiles were
     provisioned `[CITED: namespace.so/docs/reference/github-actions/runner-configuration]`.
   - What's unclear: whether this specific account already has arm64 capacity/entitlement, and
     what the maintainer will name the new profile.
   - Recommendation: this is a pre-implementation, human, dashboard-side action item — surface it
     at the very start of the plan (see Environment Availability) so it isn't discovered mid-spike
     as a blocking gap with unknown lead time.

3. **Does the migrated single job's runtime budget (4 CGo builds + zig setup + archive + checksum
   + sign + SBOM + attest, serialized on one macOS runner) fit comfortably inside GitHub Actions'
   job timeout, and does it meaningfully lengthen the release critical path?**
   - What we know: today's 3-job pipeline parallelizes the 4 build legs across 2 runner classes;
     the migration serializes everything onto one runner (explicitly accepted tradeoff per
     CONTEXT.md D-01's rationale: "one runner now serializes four CGo builds plus archive,
     checksum, sign and SBOM").
   - What's unclear: the actual wall-clock number — not measured in this research session.
   - Recommendation: capture wall-clock time from the first `--snapshot` dry run (D-06) and the
     first real spike run; this is a fact to observe during execution, not to estimate here.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `namespace-profile-macos-6x14-tahoe` | D-01, entire migrated pipeline | ✓ | — (already in production use by `release.yml`, `darwin-toolchain-canary.yml`) | — |
| `namespace-profile-linux-amd64-4x8` | D-02 (amd64 execution leg) | ✓ | — (already in production use by `release.yml`'s `build` job) | — |
| **New Namespace linux-arm64 profile** | D-02 (arm64 execution leg — no prior instance of this exists in the repo) | ✗ **unconfirmed — requires dashboard provisioning at `cloud.namespace.so/workspace/actions/profiles`** | — | None viable within REL-05's own pass condition — emulation is explicitly rejected as "the same category of weaker proof the criterion already rejects for 'build exited 0'" (D-02). This is a hard blocker for the arm64 half of REL-05 until provisioned. |
| `mlugg/setup-zig` on a macOS runner | D-01/D-02, REL-05 spike | Not yet exercised on `namespace-profile-macos-6x14-tahoe` specifically (only exercised today on the linux amd64 profile, for the arm64 leg) | pin `d1434d0...` / zig `0.15.1` `[VERIFIED: release.yml:131-135]` | If the action itself fails on this image (distinct from zig-the-toolchain's general macOS support, which is not in question), fall back to manually downloading zig's official macOS tarball and adding it to `PATH` in a raw step |
| `actions/attest-build-provenance` | D-09/D-10 | ✓ (first-party GitHub Action, no account provisioning needed) | exact tag TBD — verify at implementation time | — |
| `gh` CLI / `GH_TOKEN` | Release publish, REL-08 verification, D-08's self-upgrade job | ✓ (already ambient in every GitHub Actions job) | — | — |
| `cosign` CLI | `binary_signs:`, REL-08 verification | ✓ (already installed via `sigstore/cosign-installer` today) | pin `6f9f177...` / `v4.1.2` `[VERIFIED: release.yml:247]` | — |
| `slsa-verifier` CLI | Only if Open Question 1 resolves toward `verify-github-attestation` instead of `gh attestation verify` | Not currently installed anywhere in this repo's CI | — | Install via its own GitHub Action/binary download if this path is chosen; otherwise not needed at all |

**Missing dependencies with no fallback:**
- The **new Namespace linux-arm64 profile** (D-02). This must be provisioned before REL-05's
  arm64 execution leg can be attempted. Recommend this be the first task in the plan, sequenced
  ahead of any GoReleaser config work, since it has external lead time this research cannot
  bound.

**Missing dependencies with fallback:**
- `mlugg/setup-zig` on the macOS Namespace image — low risk (zig's own macOS support is solid),
  but unverified specifically on this image; a manual-tarball fallback exists if the action itself
  misbehaves.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's standard `testing` package — this project's release-pipeline correctness is enforced by **workflow-shape tests** (Go tests that parse the on-disk YAML of `.github/workflows/*.yml` / `.goreleaser.yaml` / `Taskfile.yml` and assert structural invariants), not conventional unit tests of application logic |
| Config file | none (standard `go test`, no framework config file) |
| Quick run command | `go test ./internal/upgrade/... -run TestDarwinLegsBuildNatively` (or any specific `Test*` name below) |
| Full suite command | `go test ./internal/upgrade/...` (all shape tests in this package); `task test:unit` for the broader unit suite |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|-------------|
| REL-05 | Cross-compiled linux/amd64 + linux/arm64 binaries execute on real Linux hardware, indexing a real fixture | integration / canary workflow run (not a `go test`) | `task check:darwin-toolchain` / `task check:darwin-release-build` pattern extended to a new `check:linux-cross-*` target dispatched via the new canary workflow | ❌ Wave 0 — new canary workflow + Taskfile targets do not exist yet |
| REL-05 | FAIL-bar variation list enumerated before first run | plan-text artifact, not a test | N/A — this is a planning deliverable (D-04), not code | N/A |
| REL-06 | Single `goreleaser release` invocation produces all expected `dist/artifacts.json` entries | shape/dry-run test | `task <new D-06 target>` (`goreleaser release --snapshot --skip=publish,sign`) then assert `dist/artifacts.json` contains 4 binary + archive entries | ❌ Wave 0 — new Taskfile target per D-06 |
| REL-07 | Exactly one process writes `codegraph_<tag>_checksums.txt`; hand-rolled step is gone | static grep/shape test | `rg -n "sha256sum" .github/workflows/release.yml` returning nothing (per the phase's own success-criterion wording) — recommend encoding this as a Go shape test alongside the existing `release_workflow_shape_test.go` suite rather than a manual grep | ❌ Wave 0 — new test |
| REL-08 | Cosign SAN proven live; `id-token: write` scoped to exactly one job | shape test | Extends `internal/upgrade/release_workflow_shape_test.go` per D-11 | ❌ Wave 0 — new test function |
| REL-08 | `.sigstore.json` sidecar name contract | static PR-time unit test | New test computing `releaseAssetName() + ".sigstore.json"` against `.goreleaser.yaml`'s `binary_signs.signature` template, per D-14 | ❌ Wave 0 — new test |
| REL-08 | Self-upgrade against a real prior release | post-release automated job (not `go test`) | New GitHub Actions job/step per D-08 | ❌ Wave 0 |
| REL-09 | Raw binary byte-unchanged; `.zip` mutation goes RED | shape test with a demonstrated-RED mutation | Rewrite of `TestDarwinLegsBuildNatively` (D-13) plus a new mutation-tested assertion that flipping the raw archive entry away from `formats: [binary]` fails a test | ❌ Wave 0 — existing test rewritten, new assertion added |

### Sampling Rate

- **Per task commit:** targeted `go test ./internal/upgrade/... -run <NewTestName>` for whichever
  shape test the task under way is adding/rewriting.
- **Per wave merge:** `go test ./internal/upgrade/...` (full shape-test suite) plus, where the
  wave touched `.goreleaser.yaml`, `task check:goreleaser` (DIST-01 validation).
- **Phase gate:** full `go test ./...` plus a real dispatch of the new D-03 canary workflow (this
  cannot be simulated locally — it requires the actual Namespace macOS + Linux runners) before
  `/gsd-verify-work`.

### Wave 0 Gaps

- [ ] New canary workflow file (D-03) exercising zig-cross-from-macOS for both linux legs,
      executing the resulting binaries on real Linux (amd64 profile + new arm64 profile) —
      the REL-05 spike itself.
- [ ] New Taskfile target(s) for the D-06 `--snapshot --skip=publish,sign` dry run.
- [ ] New Go test(s) in `internal/upgrade/` for: D-11's `id-token: write` single-job shape,
      D-14's `.sigstore.json` template contract, REL-07's hand-rolled-step-absence assertion.
- [ ] Rewrite of `TestDarwinLegsBuildNatively` per D-13's new invariant.
- [ ] Rewrite of `TestProvenanceJobUsesTaggedSLSAGenerator` (or its replacement) for the
      `actions/attest-build-provenance` job shape, per D-10.
- [ ] New post-release automated self-upgrade job/workflow (D-08).
- [ ] New Namespace linux-arm64 profile (infrastructure, not code — see Environment Availability).

## Security Domain

### Applicable ASVS Categories

This phase is CI/CD release-pipeline infrastructure, not an application feature — most ASVS
(application-security) categories do not apply directly. The relevant control surface is
supply-chain/build integrity, which ASVS treats lightly; the more precise frame is the existing
threat-register pattern this repo already uses for its release pipeline (see `SECURITY.md` /
prior phase threat registers referenced in STATE.md).

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No user-facing auth surface in this phase |
| V3 Session Management | No | N/A |
| V4 Access Control | Partially | GitHub Actions `permissions:` blocks (least-privilege `id-token: write`/`attestations: write`/`contents: write` scoping) are this phase's closest analog — D-11's new shape test is exactly an access-control-scope assertion |
| V5 Input Validation | Partially | Existing pattern already enforced repo-wide: shell steps never interpolate `${{ }}` directly, workflow context passed via `env:` (`[VERIFIED: release.yml:153-160]`) — any new step in this phase must follow the same rule |
| V6 Cryptography | Yes | Cosign keyless signing (Sigstore/Fulcio/Rekor) — never hand-roll; already the established mechanism, this phase only relocates *how* it's invoked (`binary_signs:` pipe vs. hand-rolled loop), not the cryptographic mechanism itself |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| OIDC token over-scoping — `id-token: write` present on more than one job/step, widening which workflow runs can mint a cosign-trusted certificate | Elevation of Privilege / Spoofing | D-11's new shape test asserts exactly one job carries `id-token: write` after the job-topology collapse — this is more, not less, important post-migration since collapsing 3 jobs into 1 removes the natural boundary that made "which job has the OIDC token" easy to eyeball |
| Workflow-file rename/trigger drift silently invalidating the cosign SAN anchor (`internal/upgrade/verify.go`'s `releaseWorkflowRefPattern`) | Tampering / Repudiation | `TestReleaseWorkflowFileMatchesPattern`/`TestReleaseWorkflowTriggerIsTagPushOnly` already exist and are unaffected by this phase's job-topology change — the phase must not rename `release.yml` or its `on:` trigger (already a carried-in constraint per STATE.md) |
| Attestation-format substitution silently breaking a documented verification command without updating the corresponding doc/test | Repudiation | D-10's explicit named scope (the five docs + one shape test) exists precisely to prevent shipping a supply-chain claim (REL-08) that a real user's copy-pasted command can no longer satisfy |
| Shell command injection via unescaped `${{ }}` workflow-context interpolation in a new step | Tampering | Follow the repo's existing convention: pass context via `env:`, reference via `$VAR`, never interpolate `${{ }}` directly inside a `run:` script body `[VERIFIED: release.yml:153-160]` |
| Untrusted third-party Action supply-chain risk from a newly-added Action (`actions/attest-build-provenance`) | Tampering | Pin to a full commit SHA with a trailing version-tag comment, matching every other third-party Action in `release.yml` today (the file's own header comment states this convention and its one documented exception) |

## Sources

### Primary (HIGH confidence)
- `.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/darwin-toolchain-canary.yml`, `Taskfile.yml`, `internal/upgrade/{release.go,verify.go,upgrade.go,release_workflow_shape_test.go,taskfile_shape_test.go,verify_release_e2e_test.go}`, `go.tool.mod`, `docs/RELEASE.md`, `PARSER-DECISION.md`, `.planning/{REQUIREMENTS.md,STATE.md}` — all read directly this session

### Secondary (MEDIUM confidence)
- GoReleaser official docs via Context7 (`/websites/goreleaser`): `customization/package/archives`, `customization/package/checksum`, `customization/sign/binary_sign`, `customization/sign`, `customization/sbom`, `customization/attestations`, `customization/builds/builders/go`, `getting-started/quick-start`, `deprecations`, `resources/deprecations`, `blog/reproducible-builds`
- `github.com/slsa-framework/slsa-verifier` README (WebFetch) — `verify-artifact` vs. `verify-github-attestation` command distinction and exact flag set
- `github.com/actions/attest-build-provenance` README (WebFetch) — action is a wrapper over `actions/attest` as of major version 4; verification via `gh attestation verify`
- `namespace.so/docs/reference/github-actions/runner-configuration` (WebFetch) — `namespace-profile-*` custom-profile provisioning mechanism (dashboard-created, not a built-in label)

### Tertiary (LOW confidence, marked for validation)
- WebSearch results on zig cross-compilation of CGo/glibc targets from a macOS host, glibc-version target-triple suffix syntax, `actions/attest-build-provenance` current version tag, and `gh attestation verify` flag syntax — general/community sources, not independently cross-checked against this project's specific tree-sitter grammar set (which is exactly what REL-05's spike measures directly instead)

## Metadata

**Confidence breakdown:**
- Standard stack (GoReleaser config vocabulary): HIGH — every claimed field/behavior verified against official Context7-sourced docs
- Architecture (job topology, attestation format incompatibility): MEDIUM-HIGH — the topology is derived directly from locked CONTEXT.md decisions and verified repo files; the attestation-format finding (Pitfall 1) is independently cross-checked across two sources
- Pitfalls (zig-from-macOS specifics): LOW-MEDIUM — general zig/CGo cross-compilation behavior is well-documented, but this project's specific grammar set's behavior under that exact toolchain pairing is unmeasured by design; this is the phase's own stated purpose (REL-05), not a research gap

**Research date:** 2026-08-08
**Valid until:** 14 days (CI/CD tooling — GoReleaser, GitHub Actions, Sigstore/attestation ecosystem — moves fast enough that config-shape claims should be re-verified against `goreleaser check`/live docs at implementation time rather than trusted indefinitely)
