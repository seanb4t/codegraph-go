# Phase 1: Cross-Compile Spike & `goreleaser release` Migration - Context

**Gathered:** 2026-08-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Decide the release-pipeline architecture on measured evidence (REL-05), then collapse today's
three-job / two-runner-class pipeline — `build` (4-leg matrix) → `assemble` (linux amd64) →
`provenance` (SLSA reusable workflow) — into a single `goreleaser release` invocation that
publishes raw per-platform binaries **and** `.zip` archives, with every supply-chain claim
re-proven against assets re-downloaded from the published release (REL-06/07/08/09).

Zero new user-facing capability beyond the `.zip` asset shape. The deliverable is that nothing
regressed, so any later supply-chain regression is unambiguously attributable to this migration
rather than tangled with notarization (Phase 2) or Homebrew (Phase 3) risk.

</domain>

<decisions>
## Implementation Decisions

### Spike Venue & Evidence (REL-05)

- **D-01:** The spike — and therefore the migrated single-runner pipeline — runs on
  `namespace-profile-macos-6x14-tahoe`, the runner `release.yml` and
  `darwin-toolchain-canary.yml` already use. **This corrects ROADMAP.md and PROJECT.md, which
  both say `macos-latest`; that prose is wrong about this repository and should be read as
  "one macOS runner".** Rationale: one runner now serializes four CGo builds plus archive,
  checksum, sign and SBOM (and notarization in Phase 2), and this profile is the
  maintainer-attested native Apple Silicon host that `TestDarwinLegsBuildNatively` and the
  10-03 checkpoint decision were built on.
- **D-02:** The "runs on real Linux" half of the pass condition executes on **Namespace Linux
  profiles** — the existing `namespace-profile-linux-amd64-4x8` for amd64, and a **new Namespace
  linux-arm64 profile** for arm64. No emulation: an emulated exec is the same category of
  weaker proof the criterion already rejects for "build exited 0". **This repository has no
  arm64 execution runner today** — `ci.yml`'s arm64 work is `check:reproducibility:arm64`, a
  double-build determinism diff run on an amd64 host and reported non-blocking. Standing up
  that profile is new infrastructure inside this phase, not a reused job.
- **D-03:** The spike ships as a **permanent, dispatchable canary workflow**, mirroring
  `.github/workflows/darwin-toolchain-canary.yml` (`workflow_dispatch`, same runner family).
  Not a throwaway. Rationale: `release.yml` fires only on `v[0-9]*` tag push, so without a
  canary nothing re-proves zig-cross-from-macOS when zig, the runner image, or a tree-sitter
  grammar moves — and this repository's recurring failure mode is a check that ran once during
  a decision being mistaken for a gate that keeps holding.
- **D-04:** The FAIL bar is **bounded and enumerated before the first run**, not open-ended
  debugging. The plan must write the variation list down in advance; exhausting it declares
  FAIL and triggers the costed GoReleaser Pro fallback. Minimum list: zig version (the pinned
  `0.15.1` plus one newer), glibc target triple with an explicit floor
  (e.g. `x86_64-linux-gnu.2.28` / `aarch64-linux-gnu.2.28`), and static-vs-dynamic
  `CGO_LDFLAGS`. Enumerate-then-exhaust keeps the Pro decision falsifiable.

### Producing the Verification Release (REL-08)

- **D-05:** The migration PR is titled **`feat(release): …`** so release-please computes a minor
  bump (v0.4.0 → v0.5.0, matching the milestone label prediction) and cuts the tag itself.
  D-06R is untouched — release-please still computes and owns the version; no human runs
  `git tag`, and no `Release-As:` footer forces anything. The `feat:` type is honest rather than
  gamed: the release genuinely gains a new user-facing asset shape (`.zip` archives for browser
  download and Homebrew), which is REL-09.
  **This repository is squash-only with `squash_merge_commit_title=PR_TITLE`, so the PR title —
  not the commit subjects — is what release-please parses.** Getting the title wrong yields no
  release at all, and therefore no REL-08 evidence and no Phase-2 RED baseline.
  — **Reversibility:** one-way — once release-please publishes `v0.5.0`, the tag and release
  exist on the public shipped line and cannot be withdrawn without a human deleting a tag,
  which is exactly the authority D-06R reserves.
- **D-06:** Before the first real release, a **new Taskfile target** runs
  `goreleaser release --snapshot --skip=publish,sign` on a native macOS host. This directly
  mirrors `check:darwin-release-build`, which already uses `--snapshot` because a PR HEAD carries
  no tag and GoReleaser hard-fails without one. It exercises the real 4-target composition plus
  archive and checksum without publishing anything.
- **D-07:** Recovery posture is **patch forward**. Never delete a published release or tag —
  deleting and re-pushing a tag re-fires `release.yml` and is a human touching tag authority,
  and it would destroy the un-notarized darwin asset Phase 2 needs as its SIGN-03 RED baseline.
  A wrong release is fixed by the next release-please patch.
  — **Reversibility:** costly — the policy itself is a choice, but violating it once destroys
  Phase 2's RED baseline, which cannot be recreated after notarization lands.
- **D-08:** REL-08's third claim — a genuinely shipped prior binary self-upgrading — is an
  **automated post-release job** (or an extension of the new canary), not a manual runbook step.
  It downloads the real prior release binary for darwin/arm64 and linux/amd64, runs
  `codegraph upgrade` against the new release, and asserts the resulting binary's sha256 equals
  the attested subject. It re-fires on every release instead of being true only on the day it
  was hand-checked.

### Job Topology & Attestation

- **D-09:** **One job.** `actions/attest-build-provenance` replaces the
  `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0`
  caller entirely. Maintainer rationale: the GitHub-native attestor is less custom than a
  third-party reusable workflow, and changing the verification surface is acceptable now.
  — **Reversibility:** one-way — releases published under the native attestor and releases
  published under the generic generator carry different provenance formats and different
  builder identities. Reverting fixes future releases but leaves the shipped line permanently
  mixed, and the published verification instructions will already have changed.
- **D-10:** The switch is **unconditional**, not gated on research. The following all enter
  Phase-1 scope explicitly, named up front rather than discovered mid-phase:
  `internal/upgrade/release_workflow_shape_test.go`'s
  `TestProvenanceJobUsesTaggedSLSAGenerator` (asserts the provenance job `uses:` a tagged
  generic generator and declares no `runs-on:`), `docs/RELEASE.md` §65-69,
  `docs/RELEASE-PROCEDURES.md` §224, `SECURITY.md`, `README.md`, and **the wording of REL-08
  itself**, which currently names `slsa-verifier verify-artifact` as a claim that must pass.
  If research finds `slsa-verifier` does not accept native attestations, the verification
  command becomes `gh attestation verify` and REL-08 is reworded — that is accepted, not a
  blocker.
  De-risking fact already established: `internal/upgrade/verify.go` contains **no** SLSA or
  in-toto references. It verifies the cosign bundle over the binary. `codegraph upgrade` is
  therefore completely indifferent to which attestor is used; the blast radius is
  documentation, one shape test, and one requirement sentence.
- **D-11:** The cosign SAN is **proven, not assumed**, by two mechanisms: (a) live
  `cosign verify-blob` against a re-downloaded published asset using the unchanged
  `--certificate-identity-regexp` (already REL-08 criterion 3), and (b) a **new shape test**
  asserting exactly one job in `release.yml` carries `id-token: write` and that it is the job
  invoking goreleaser. Today nothing machine-holds where the OIDC token lives, and that token
  is what mints the cert whose SAN `internal/upgrade`'s `releaseWorkflowRefPattern` anchors on.
  The pattern itself (`.github/workflows/release.yml@refs/tags/v[0-9]*`) is keyed to the
  workflow file and ref, not to a job or runner, so it should survive the move — "should" is
  what (a) and (b) exist to convert into "does".
- **D-12:** Checksums and attestation subjects cover **8 payloads** — the 4 raw binaries and
  the 4 `.zip` archives. Excluded: `.sigstore.json` and `.spdx.json` sidecars (signing and
  attesting over a signature is circular) and the checksums file itself. This satisfies REL-06
  criterion 2's "every published asset exactly once" for the things a user actually downloads,
  and extends provenance to the zips the Homebrew cask and browser downloaders will consume in
  Phases 2–3.
  — **Reversibility:** costly — the published checksums file shape is a contract external
  consumers may script against; narrowing it later silently drops coverage.
- **D-13:** `TestDarwinLegsBuildNatively` is **rewritten to the new invariant**, not deleted and
  not taught to accept two shapes. New assertion: the job invoking goreleaser runs on a darwin
  runner (darwin stays native) **and** the linux build ids carry a `zig cc` `CC`/`CXX` override.
  Demonstrated RED by flipping the runner to ubuntu. The D-08 property it was written to
  protect — a Linux-hosted CGo cross-link silently breaking libresolv DNS resolution in a binary
  that makes real HTTPS calls — is exactly as load-bearing after the migration as before.

### GoReleaser Config Shape

- **D-14:** The `<asset>.sigstore.json` sidecar contract is held by a **static PR-time unit
  test** that computes `internal/upgrade.releaseAssetName() + ".sigstore.json"` for all four
  platforms and asserts `.goreleaser.yaml`'s `binary_signs.signature` template resolves to
  exactly that, demonstrated RED by perturbing the template. The dynamic side is already
  covered — D-08's post-release upgrade job fails outright if the sidecar name is wrong. This is
  the one contract whose breakage bricks every user's `codegraph upgrade`.
- **D-15:** The archive is `codegraph_<tag>_<goos>_<goarch>.zip` — the **same stem as the raw
  asset plus `.zip`**. The two `archives:` entries are keyed by `id`, which GoReleaser supports
  natively, so the raw entry stays `formats: [binary]` and D-02/Finding 1 is preserved
  byte-unchanged rather than amended. The prefix-glob ambiguity hazard (`for f in codegraph_*`)
  leaves with the shell steps being deleted.
  — **Reversibility:** costly — published asset names appear in cask URLs, docs and user
  scripts; renaming later breaks anything pinned to the old shape.
- **D-16:** The `.zip` contains **binary + LICENSE + README** (GoReleaser's conventional
  default). Completions and man pages stay **out**: BREW-03/BREW-04 require those generated from
  the binary at cask-build time, so shipping them in the zip would create a second, staler
  source and defeat the "a new subcommand appears without anyone editing a committed file"
  property.
- **D-17:** SBOMs stay **per-binary, preserving today's `<asset>.spdx.json` names**, so DIST-03
  carries forward unchanged and the per-platform build-constraint precision (only modules
  actually compiled into that platform's binary) is retained. Zips get no separate SBOM — the
  content would duplicate their binary's.
  — **Reversibility:** costly — published SBOM artifact names are part of the release asset
  contract.

### Claude's Discretion

None claimed — every question in this discussion received an explicit answer. Four items were
deliberately routed to research rather than to the maintainer, because they are facts to be
established, not preferences:

1. Whether a Namespace **linux-arm64** profile is available on this account, and its exact
   profile label (D-02 depends on it).
2. Whether `mlugg/setup-zig` runs on the Namespace macOS image, and whether zig's macOS host
   support covers `x86_64-linux-gnu` as well as `aarch64-linux-gnu` for CGo.
3. Whether GoReleaser's `checksum:` block can be scoped to exclude sidecar artifacts, and
   whether `binary_signs:` supports an exact `${artifact}.sigstore.json` signature template
   (D-12, D-14 depend on both).
4. Whether `actions/attest-build-provenance` can attest a multi-subject list the way the generic
   generator parses each `<sha256>  <name>` line of `checksums.txt`, and whether
   `slsa-verifier verify-artifact` accepts its output or verification becomes
   `gh attestation verify` only (D-10's doc-rewrite scope depends on the answer).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope and locked decisions
- `.planning/ROADMAP.md` § "Phase 1: Cross-Compile Spike & `goreleaser release` Migration" — goal, the four success criteria, and the ordering rationale. Note D-01: its `macos-latest` phrasing is wrong for this repo.
- `.planning/REQUIREMENTS.md` § "Release Pipeline (REL)" — REL-05 through REL-09 verbatim, plus the Out of Scope table (GoReleaser Pro, stapling, `brews:`).
- `.planning/PROJECT.md` § Key Decisions (2026-08-07) — the locked REL-05 pass condition, the costed Pro fallback with its three named gate repairs, and the `homebrew_casks:`-not-`brews:` correction.
- `.planning/STATE.md` § Session Continuity → "PHASE 1 CARRY-INS" — the three carry-in constraints (don't destroy Phase 2's RED baseline; don't rename `release.yml` or change its trigger; the no-Pro comment is a recorded decision).

### Release pipeline as it stands today
- `.goreleaser.yaml` — 4 build ids; `codegraph-linux-arm64` **already** sets `CC=zig cc -target aarch64-linux-gnu`. The `archives:` and `checksum:` blocks are annotated as **dead configuration** that never executes under `goreleaser build --single-target`; this phase wakes them up. Header comments (a)–(d) are the contracts.
- `.github/workflows/release.yml` — the three jobs being collapsed. §55-63 the `GORELEASER_VERSION` pin; §73-194 the `build` matrix and the "Rename to release-asset contract name" step; §215-274 `assemble`'s hand-rolled `sha256sum`, per-binary `cosign sign-blob` and syft loop; §350-358 the `provenance` job.
- `.github/workflows/darwin-toolchain-canary.yml` — the shape precedent for D-03's permanent canary.
- `.github/workflows/ci.yml` §239-273 — the existing zig setup step and `check:reproducibility:arm64`, the only arm64 work in CI today (build-only, amd64 host, non-blocking).
- `Taskfile.yml` — §241 `check:darwin-toolchain`, §270 `check:darwin-release-build` (the `--snapshot` precedent D-06 mirrors), §349 `check:goreleaser` (DIST-01), §354 `check:cross`.

### Contracts and gates the migration must not silently break
- `internal/upgrade/release.go` — `releaseAssetName()`, the authoritative raw-asset name shape (`codegraph_<tag>_<goos>_<goarch>`, v-prefixed tag).
- `internal/upgrade/verify.go` — `releaseWorkflowRefPattern` (the cosign SAN anchor) and `defaultVerify`. Confirmed to contain no SLSA/in-toto references.
- `internal/upgrade/release_workflow_shape_test.go` — `TestDarwinLegsBuildNatively` (D-13 rewrites), `TestProvenanceJobUsesTaggedSLSAGenerator` (D-10 replaces), `TestReleaseWorkflowFileMatchesPattern`, `TestReleaseWorkflowTriggerIsTagPushOnly`.
- `internal/upgrade/taskfile_shape_test.go` — `TestWorkflowRunBodiesInvokeTask` (**forces** the `goreleaser release` invocation into a Taskfile target), `TestGoreleaserPinParity`, `TestCheckCrossMatchesGoreleaserTargets`, `TestGateStancesStated`, `TestRequiredCheckNamesPreserved`.

### Documentation that D-10 puts in scope
- `docs/RELEASE.md` §65-69 — the `slsa-verifier verify-artifact` instructions users are told to run.
- `docs/RELEASE-PROCEDURES.md` §224 — the maintainer-side equivalent.
- `SECURITY.md`, `README.md` — both reference the SLSA verification path.

### Background
- `PARSER-DECISION.md` — the CGo tree-sitter exception (DIST-05); why CGo is present at all, which is the whole reason REL-05 is uncertain.
- `.planning/research/PITFALLS.md`, `.planning/research/STACK.md`, `.planning/research/SUMMARY.md` — the 2026-08-07 research that falsified the three scoping assumptions.
- `CONTRIBUTING.md` — local build tooling contract; `TestContributingReferencesRealTaskTargets` holds it to real Taskfile targets, so a new target must be reflected there.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`.goreleaser.yaml`'s dead `archives:`/`checksum:` blocks** — already written in the correct shape (`formats: [binary]`, `{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}`, sha256). The migration largely activates existing config rather than authoring it.
- **`zig cc` is already in the build graph** — `codegraph-linux-arm64` sets `CC=zig cc -target aarch64-linux-gnu` / `CXX=zig c++ …`. The spike changes exactly one variable (host = macOS instead of linux-amd64) and adds one target (`x86_64-linux-gnu`).
- **`darwin-toolchain-canary.yml`** — a working `workflow_dispatch` canary on the target macOS runner; D-03's new canary is a sibling, not a novel pattern.
- **`check:darwin-release-build`** — already proves GoReleaser can drive a darwin build on a native mac using `--snapshot`, including the load-bearing "build the tool natively first, don't let GOOS/GOARCH leak into the tool build" ordering. D-06's dry-run target extends this.
- **`mlugg/setup-zig@v2.2.1` pinned to zig `0.15.1`** in `release.yml` §131-135 — the existing pin D-04's variation list starts from.

### Established Patterns
- **`Taskfile.yml` is the single definition of every CI job body**, machine-enforced by `TestWorkflowRunBodiesInvokeTask`. The `goreleaser release` invocation must live in a Taskfile target; a bare `run:` in the workflow fails an existing gate.
- **A gate is not trusted until demonstrated RED against a confirmed-applied mutation** (standing decision, PROJECT.md). Applies to every new test in this phase: D-11's `id-token` shape test, D-13's rewritten darwin test, D-14's sidecar-name test, and REL-09's `formats: [binary]` mutation.
- **Shell steps never interpolate `${{ }}` directly** — workflow context is passed via `env:` and referenced as `$TAG`/`$GOOS` so a crafted tag name is data, not script text (`release.yml` §153-160). Any new step follows this.
- **Artifact lookup goes through `dist/artifacts.json`**, GoReleaser's documented stable interface, with a `find(1)` fallback — dist/ subdirectory names are not stable across GoReleaser versions (`release.yml` §164-179).
- **`GORELEASER_VERSION` in `release.yml` is held equal to `go.tool.mod`'s pin** by `TestGoreleaserPinParity`. Moving one without the other is a test failure, and this is one of the three gates the Pro fallback would blind.

### Integration Points
- **`release.yml` → `internal/upgrade`**: the raw asset name, the `.sigstore.json` sidecar name, and the cosign cert SAN are a three-part contract consumed at runtime by `codegraph upgrade`. All three cross the migration boundary.
- **`release.yml` → Phase 2**: this phase's published, deliberately un-notarized darwin asset is Phase 2's SIGN-03 RED baseline. Preserve it.
- **`.goreleaser.yaml` `archives:` → Phase 3**: the `.zip` this phase adds is what the Homebrew cask points at.
- **New Namespace linux-arm64 profile → account/billing**: an external provisioning dependency, like the Phase-2 Apple credentials and the Phase-3 tap PAT. Surface it early; it is not implementation work.

</code_context>

<specifics>
## Specific Ideas

- **The runner-name discrepancy is a scoping correction, not a typo.** ROADMAP.md and PROJECT.md both say `macos-latest`; the pipeline uses `namespace-profile-macos-6x14-tahoe`. D-01 resolves it in favour of the Namespace profile. Planning should treat "one `macos-latest` runner" in those documents as meaning "one macOS runner" and may propose correcting the prose.
- **"Runs on real Linux" means indexing a fixture, not `--version`.** ROADMAP criterion 1 is explicit; the canary must exercise the CGo tree-sitter path, which is the thing zig-cross could plausibly break.
- **Enumerate the FAIL variations in the plan text, before any run.** D-04's value is that the Pro decision is falsifiable — a list written after a failure is rationalization.
- **Name the doc/test churn as tasks, not as cleanup.** D-10's five documents plus one shape test are scope, decided up front.

</specifics>

<deferred>
## Deferred Ideas

- **DIST-04 double-build determinism under a changed host** — whether the byte-identical
  double-build gate still holds when the Linux binaries are zig-crossed from a macOS host rather
  than built natively on Linux, and what becomes of `check:reproducibility:arm64` (today an
  amd64-hosted, non-blocking build-only leg). Raised and explicitly not discussed; likely
  surfaces during Phase-1 planning as a consequence of D-01/D-02 rather than as new scope.
- **Where the zig version pin lives** once zig is needed on the macOS host for both Linux legs —
  `release.yml`'s `setup-zig` step, `.goreleaser.yaml` env, or `Taskfile.yml` — and whether one
  pin governs CI, the new canary, and local builds. Not discussed.
- **Whether `check:cross`, `check:darwin-toolchain` and `check:darwin-release-build` survive
  as-is, merge, or gain a linux-cross sibling.** Not discussed.
- **`GO-2026-5932` revisit.** STATE.md notes this milestone touches `goreleaser` directly, so the
  accepted-unmitigated openpgp exposure may be revisited *on evidence*. Not in Phase-1 scope
  unless the migration surfaces something new.
- **Backlog bookkeeping question (999.3 / 999.6 strike-vs-annotate)** — an outstanding maintainer
  call recorded in STATE.md Blockers/Concerns, unrelated to this phase.

### Reviewed Todos (not folded)

- **Wire oracle `toolslist-repeat` response-ordering flake** (`.planning/todos/`, created
  2026-08-07, area `mcp`, severity major) — matched Phase 1 at score 0.6 on the generic keywords
  "test" and "yml" only. Deliberately **not folded**: it is an MCP JSON-RPC arrival-order
  assertion with no connection to goreleaser, notarization or Homebrew, and folding it would
  widen a phase whose entire ordering rationale is that a supply-chain regression must be
  unambiguously attributable to the migration.

</deferred>

---

*Phase: 1-Cross-Compile Spike & `goreleaser release` Migration*
*Context gathered: 2026-08-08*
