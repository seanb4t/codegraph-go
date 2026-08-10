# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14**: the core capabilities (indexing, query, MCP server, sync, migration) work from a signed/attested/SBOM'd release that beats TS 1.3.1 on every measured benchmark — but the CLI/agent surface still diverged *behaviorally* from TS. **v1.0 (Drop-in Parity & Human UX) shipped 2026-08-03**: those gaps are closed. An existing user can now swap binaries with zero change in experience — TS-identical `explore`/`node`/`status` behavior, watcher-on-MCP by default, git/worktree awareness, output hygiene, a human-facing Charm TUI behind a build-enforced rendering seam (the agent/MCP path never sees ANSI), systematic flag reconciliation, fully automated signed releases via release-please + GoReleaser, and contributor-facing local build tooling. **v0.3.0 (MCP Protocol Currency) shipped 2026-08-06**: the stdio MCP server is current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, proven by a wire-level oracle that never imports the SDK it tests, with every Legacy client still working.

**v0.5.0 (macOS Distribution & Homebrew) is in progress.** The binary is correct and verifiable; it is not yet *installable by convention* on macOS. `spctl -a -vv -t exec` returns **rejected** on both darwin arches today, and there is no package-manager path at all. This milestone closes both — a browser-downloaded asset passes Gatekeeper, and `brew install` works from a tap we control — by moving the release pipeline onto `goreleaser release`, whose `notarize:` and `homebrew_casks:` blocks never execute under the `goreleaser build --single-target` matrix the pipeline has always used. That migration is not free: research (2026-08-07) established that `goreleaser release` refuses a `dist/` built elsewhere and that both escape hatches — `release --split`/`continue --merge` and the `prebuilt` builder — are GoReleaser **Pro**. One `macos-latest` runner with `zig cc` cross-compiling both Linux legs is therefore the only OSS path, and whether that works with this repo's CGo tree-sitter dependency is the milestone's single unproven claim. It is spiked first, blocking, with a costed fallback already named.

**Versioning note:** "v1.0" is a *planning-milestone* name, never a release version. The shipped artifact line reached `v0.2.0` at v1.0's close and has since advanced to `v0.3.0` and `v0.4.0`, each computed by release-please from Conventional Commits; there is deliberately no `v1.0.0` tag (maintainer directive D-06R, 2026-07-29). From v0.3.0 onward the milestone label tracks the actual release line, but it is still a planning label carrying **no git tag** — release-please remains the sole tag authority, so `v0.5.0` is a prediction that holds if this milestone lands `feat:` commits. A hand-created `v0.5.0` tag would additionally match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline. Milestones carry no git tag; the milestone record lives in `MILESTONES.md` + `milestones/`. (`milestone-v0.1` exists only because it predates release-please.)

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- ✅ **v1.0 — Drop-in Parity & Human UX** — Phases 1–10 (shipped 2026-08-03) — behavioral + surface parity with TS 1.3.1, human TUI, automated signed releases, local build tooling
- ✅ **v0.3.0 — MCP Protocol Currency** — Phases 1–5 (shipped 2026-08-06) — official Go SDK adoption, `2026-07-28` spec compliance without breaking Legacy clients, a wire-level verification oracle, tool-modfile vulnerability coverage
- 🚧 **v0.5.0 — macOS Distribution & Homebrew** — Phases 1–4 (in progress) — `goreleaser release` migration, Apple notarization, a Homebrew tap, and an `upgrade` that steps aside under brew. Promotes backlog 999.5 and consumes SEED-002
- 📋 **Later** — unscoped. Candidates: the Backlog items below (999.2 tmux TTY harness, 999.4 CheckRegression guard), the v0.5.x deferrals (DIST-06 stapled offline-safe container, BREW-07 homebrew-core), Team Scale (central server, CI-distributed indexes), MRTR/elicitation (MRTR-01), annotations (embeddings/communities/export), local Svelte web UI (SEED-001)

## Phases

<details>
<summary>✅ v0.1 — Initial Release (Phases 1–8) — SHIPPED 2026-07-14</summary>

Full phase details archived in [`milestones/v0.1-ROADMAP.md`](milestones/v0.1-ROADMAP.md); phase artifacts in [`milestones/v0.1-phases/`](milestones/v0.1-phases/).

- [x] Phase 1: Foundation — Storage, Schema & Parser Strategy (7/7 plans) — completed 2026-07-10
- [x] Phase 2: Go Indexing Pipeline (6/6 plans) — completed 2026-07-11
- [x] Phase 3: Query Engine & MCP Server (9/9 plans) — completed 2026-07-11
- [x] Phase 4: Incremental Sync & File Watcher (9/9 plans) — completed 2026-07-11
- [x] Phase 5: Language Coverage & Resolution Breadth (14/13 plans) — completed 2026-07-12
- [x] Phase 6: Agent Integrations & CLI Lifecycle (6/6 plans) — completed 2026-07-12
- [x] Phase 7: Migration Tool (7/7 plans) — completed 2026-07-13
- [x] Phase 8: Release Hardening & Benchmarks (9/9 plans) — completed 2026-07-14

**Delivered:** the core capabilities of CodeGraph in a single static Go binary — index/query/MCP/sync/migrate — faster and lighter than TS 1.3.1, from a signed/attested/SBOM'd release verified end-to-end on `v0.0.0-rc.3`. **Known gaps carried into v1.0:** the CLI/agent surface diverges *behaviorally* from TS; DIST-02 (real `v*` tag) and PERF-01 (published numbers) remained pending.

</details>

<details>
<summary>✅ v1.0 — Drop-in Parity & Human UX (Phases 1–10) — SHIPPED 2026-08-03</summary>

Full phase details archived in [`milestones/v1.0-ROADMAP.md`](milestones/v1.0-ROADMAP.md); phase artifacts in [`milestones/v1.0-phases/`](milestones/v1.0-phases/); requirements in [`milestones/v1.0-REQUIREMENTS.md`](milestones/v1.0-REQUIREMENTS.md).

- [x] Phase 1: Behavioral Parity — explore & node (17/17 plans) — completed 2026-07-15
- [x] Phase 2: status Content & Git/Worktree Awareness (7/7 plans) — completed 2026-07-16
- [x] Phase 3: Watcher-on-MCP Default (5/5 plans) — completed 2026-07-16
- [x] Phase 4: Output Hygiene (3/3 plans) — completed 2026-07-16
- [x] Phase 5: Git Sync Hooks (5/5 plans) — completed 2026-07-17
- [x] Phase 6: Rendering Seam & Pretty status/files (3/3 plans) — completed 2026-07-17
- [x] Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select (8/8 plans) — completed 2026-07-26
- [x] Phase 8: Surface Reconciliation & Signed v1.0.0 Release (9/9 plans) — completed 2026-07-28
- [x] Phase 9: release-please + GoReleaser (8/8 plans) — completed 2026-08-01 (promoted from backlog 999.3 on 2026-07-27)
- [x] Phase 10: Local Build Tooling & CONTRIBUTING (7/7 plans) — completed 2026-08-03 (promoted from backlog 999.1 on 2026-07-27)

**Delivered:** drop-in behavioral parity with TS CodeGraph v1.3.x, plus the human-facing surface v0.1 lacked. All 10 phases independently verified (`verification_status: passed`), 48/48 requirements satisfied, 72 plans, 594 commits over 20 days (755 files, +98,761/−1,942 lines). The "not yet drop-in" caveat is retired. Release cutting is fully automated — `v0.2.0` was tagged, changelogged, built, signed, SBOM'd and SLSA-attested with no human running `git tag`, and a genuinely shipped prior binary self-upgraded to it byte-identically to the attested subject.

**Carried forward:** four backlog items (999.2–999.5) and the residual darwin release-path check — `release.yml`'s goreleaser/cosign/SLSA steps first execute on the macOS runner class during a real tag push, though a permanent canary already machine-proves that runner's availability and native toolchain.

</details>

<details>
<summary>✅ v0.3.0 — MCP Protocol Currency (Phases 1–5) — SHIPPED 2026-08-06</summary>

Full phase details archived in [`milestones/v0.3.0-ROADMAP.md`](milestones/v0.3.0-ROADMAP.md); phase artifacts in [`milestones/v0.3.0-phases/`](milestones/v0.3.0-phases/); requirements in [`milestones/v0.3.0-REQUIREMENTS.md`](milestones/v0.3.0-REQUIREMENTS.md); audit in [`v0.3.0-MILESTONE-AUDIT.md`](v0.3.0-MILESTONE-AUDIT.md).

- [x] Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle (7/7 plans) — completed 2026-08-05
- [x] Phase 2: SDK Migration — official go-sdk on the existing surface (5/5 plans) — completed 2026-08-06
- [x] Phase 3: `2026-07-28` Spec Compliance (5/5 plans) — completed 2026-08-06
- [x] Phase 4: Supply-Chain Coverage & Daemon Substrate Fixes (3/3 plans) — completed 2026-08-06
- [x] Phase 5: Live Tool-Catalog Change Notification (1/1 plans) — completed 2026-08-06

**Delivered:** codegraph-go's stdio MCP server is current with spec revision `2026-07-28` on `modelcontextprotocol/go-sdk@v1.7.0`, with `mark3labs/mcp-go` gone. All 25 requirements satisfied, all 5 phases independently verified, 21 plans over 136 commits in 4 days (174 files, +32,372/−1,901). Legacy clients are unbroken — a `2024-11-05` client completes a session and calls a tool, asserted on the wire. The verification oracle grew 23 → 28 frozen transcripts and never imports the SDK it tests.

**Accepted limitations, carried not closed:** the daemon extreme-load timeout tail and its feedback-latency tradeoff (CI load ruled the governing standard for MAINT-02), and `GO-2026-5932` — a real, unmitigated vulnerability reachable in `goreleaser`'s binary through cosign/rekor's unmaintained openpgp, surfaced by the new advisory scan rather than resolved.

**No git tag.** release-please is the sole tag authority (D-06R); a `v0.3.0` tag would match `release.yml`'s `v[0-9]*` trigger and falsely fire the release pipeline.

</details>

### 🚧 v0.5.0 — macOS Distribution & Homebrew (In Progress)

**Milestone Goal:** Make installing codegraph on macOS trustworthy and conventional — a browser-downloaded release asset passes Gatekeeper, and `brew install` works from a tap we control — by moving the release pipeline onto `goreleaser release` so its native notarization and Homebrew support become available, without breaking the raw-binary contract `codegraph upgrade` depends on. Phase numbering restarts at 1 for this milestone (v0.1, v1.0 and v0.3.0 archived), matching this repo's established convention.

**Ordering is load-bearing, not stylistic:**

- **Phase 1 opens with a blocking spike, and the milestone's architecture is contingent on its outcome.** `goreleaser release` refuses a `dist/` built elsewhere, and both escape hatches are Pro-only, so a single `macos-latest` runner with `zig cc` on both Linux legs is the *only* OSS shape. Whether zig-cross links this repo's CGo tree-sitter dependency for linux from a macOS host is unproven — today only linux/arm64 is zig-crossed, and from a Linux host. **Pass condition (locked in PROJECT.md):** `zig cc` cross-compiles CGo tree-sitter to linux/amd64 **and** linux/arm64 from a macOS host **and** the resulting binaries run on real Linux. Anything less — including a build that exits 0 but produces a binary that will not run — is a fail.
- **The fallback is costed and named, not discovered mid-phase.** On spike failure: buy GoReleaser Pro (tier per EULA; Personal $165/yr, Startup $247/yr as verified 2026-08-07) and reshape around `release --split`/`continue --merge`, which preserves the native darwin matrix. That branch carries **three explicit gate repairs as added scope**, because Pro is not `go install`-able from a public module and detaches from `go.tool.mod`: (1) `check:goreleaser`/DIST-01 validates with the OSS binary and would reject Pro-only keys while fork PRs cannot read `GORELEASER_KEY`; (2) `TestGoreleaserPinParity` loses one side of its equality; (3) `tool-vuln`/VULN-01-02-03 stops covering the releaser that actually ships releases, turning `GO-2026-5932` from **accepted and measured** into **unmeasurable** — this repo's own recurring failure mode, a gate going blind. `release.yml` also carries a deliberate comment that no Pro directive is used anywhere; reversing it is a recorded decision, not a silent config change.
- **Phase 1 changes the mechanism without changing the product.** Zero new user-facing capability; the deliverable is that nothing regressed. Any supply-chain regression is then unambiguously attributable to the migration rather than tangled with notarization or brew risk.
- **Notarization (Phase 2) precedes the Homebrew cask (Phase 3), because the cask ships notarized bytes.** GoReleaser runs `notarize` before `archive`/`checksum`/`sign`/`sbom`, so the `.zip` the cask points at contains post-notarization bytes by construction — but that ordering is measured here (sha256 diffed per stage), never trusted from documentation.
- **Phase 1's release is Phase 2's RED baseline.** Phase 1 publishes `.zip` archives before notarization exists, which produces a real, published, un-notarized darwin asset. SIGN-03 requires the Gatekeeper gate be demonstrated RED against exactly that before it is trusted green.
- **Brew detection (Phase 4) can be developed in parallel with Phases 2–3 against a constructed Cellar-shaped symlink tree, but its acceptance needs the real tap Phase 3 publishes.** A path-prefix guess is the exact shape of gate this repo keeps finding cannot fire, so the honest acceptance test is a genuine `brew tap` + `brew install` followed by a refused `codegraph upgrade`.
- **`v0.5.0` carries no git tag.** release-please is the sole tag authority (D-06R), and a `v0.5.0` tag would match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline.

- [x] **Phase 1: Cross-Compile Spike & `goreleaser release` Migration** - Decide the pipeline architecture on measured evidence, then move to one `goreleaser release` invocation that publishes both raw binaries and `.zip` archives with every supply-chain claim re-proven against real published assets (completed 2026-08-08)
- [x] **Phase 2: Apple Signing & Notarization** - A browser-downloaded darwin asset stops being blocked by Gatekeeper, proven by a gate shown RED against the un-notarized binary first (completed 2026-08-09)
- [ ] **Phase 3: Homebrew Tap & Cask** - `brew tap seanb4t/tap && brew install codegraph` works on a clean machine, with completions, man pages, a real `test:` block, and a proven-recoverable tap-push failure
- [ ] **Phase 4: `codegraph upgrade` × Homebrew** - `codegraph upgrade` detects a brew-managed install, refuses, and points at `brew upgrade codegraph` — never touching the Cellar

## Phase Details

### Phase 1: Cross-Compile Spike & `goreleaser release` Migration

**Goal**: A maintainer knows, from measurement rather than inference, whether the OSS single-runner architecture is reachable — and the release pipeline is then a single `goreleaser release` invocation whose published assets still satisfy every supply-chain claim the old pipeline satisfied, while carrying `.zip` archives alongside the raw binaries `codegraph upgrade` consumes.
**Depends on**: Nothing (first phase of v0.5.0)
**Requirements**: REL-05, REL-06, REL-07, REL-08, REL-09
**Success Criteria** (what must be TRUE):

  1. The pipeline architecture is decided on evidence a third party can re-inspect: either a recorded run of a `zig cc`-cross-compiled linux/amd64 binary **and** a linux/arm64 binary — both built on a macOS host, both **executing on real Linux** and indexing a fixture, not merely reporting `--version` — or, on failure of either leg, a recorded decision to adopt GoReleaser Pro with the three named gate repairs (`check:goreleaser`/DIST-01, `TestGoreleaserPinParity`, `tool-vuln`/VULN-01-02-03) entered as scope before any restructuring lands. A build exiting 0 is explicitly not evidence for this criterion (REL-05)
  2. A real release is cut by one `goreleaser release` invocation, and `gh release view <tag> --json assets` lists exactly one `codegraph_<tag>_checksums.txt` whose contents cover every published asset exactly once — with the hand-rolled `sha256sum` step gone from `release.yml`, so a search of that file for it returns nothing rather than a second writer racing the first (REL-06, REL-07)
  3. Against assets **re-downloaded from the published release** — never a local `dist/` copy — `cosign verify-blob` returns Verified OK under the unchanged `release.yml@refs/tags/v*` SAN, `gh attestation verify` passes against the `actions/attest-build-provenance` attestation, and a genuinely shipped prior binary self-upgrades via `codegraph upgrade` on darwin/arm64 and linux/amd64 (REL-08)
  4. For each platform the release carries both a raw `codegraph_<tag>_<goos>_<goarch>` asset — extension-free, directly executable, and byte-shape-identical to what `internal/upgrade.releaseAssetName()` builds — and a distinctly-named `.zip`; a mutation changing the raw entry away from `formats: [binary]` turns a test red rather than shipping an archive that verifies its signature and then bricks the install (REL-09)

**Plans**: 6/6 plans executed

Plans:
**Wave 1**

- [x] 01-01-PLAN.md — REL-05 spike (tracer): one `goreleaser release --snapshot` on macOS cross-compiles both linux legs via zig, and a permanent canary executes them on real Linux

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — `.goreleaser.yaml` owns archive/checksum/sign/SBOM/release: raw + `.zip` entries by id, 8-payload checksum scope, `binary_signs:`, per-binary `sboms:`, and an explicit `release:` block pinning rerun idempotency
- [x] 01-03-PLAN.md — collapse `release.yml` to one `goreleaser release` job; delete the hand-rolled checksum/sign/SBOM shell; scope `id-token: write` to that one job

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-04-PLAN.md — swap the SLSA generic generator for `actions/attest-build-provenance`; rewrite every published verification instruction and REL-08's wording
- [x] 01-06-PLAN.md — exercise the `binary_signs:` sign pipe before the one-way release, using a throwaway local cosign key, proving four DISTINCT `.sigstore.json` sidecars rather than four colliding ones

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-05-PLAN.md — cut a real release and re-prove every supply-chain claim against re-downloaded published assets, automated to re-fire on every future release

### Phase 2: Apple Signing & Notarization

**Goal**: A macOS user who downloads a release asset in a browser can run it without Gatekeeper blocking them — and the project can prove that claim with a check it has already watched fail.
**Depends on**: Phase 1 (notarization slots into a working `goreleaser release` pipeline, not a half-migrated one; Phase 1's published un-notarized asset is this phase's RED baseline)
**Requirements**: SIGN-01, SIGN-02, SIGN-03, SIGN-04
**Success Criteria** (what must be TRUE):

  1. The gate is shown RED first: `spctl -a -vv -t install` on the darwin asset Phase 1 published, with a `com.apple.quarantine` xattr confirmed present via `xattr -p`, reports `rejected` (exit 3) — recorded before any green run, so the check is known to be able to fire. A green CI step, a passing `codesign -dvv` (which already passes today on the adhoc-signed darwin/arm64 binary `spctl` rejects), an Accepted `notarytool` history entry, and `spctl` on a file that was never quarantined are each explicitly recorded as insufficient (SIGN-03)
  2. The same command on the notarized, published darwin asset — quarantine xattr again confirmed present before the run — reports `accepted` with `source=Notarized Developer ID` (exit 0), and the verdict is taken from the exit status rather than from a substring search (SIGN-01, SIGN-02). **Amended 2026-08-09 (D-19)**: criteria 1 and 2 previously specified `-t exec`, and criterion 2 additionally required `syspolicy_check distribution` to pass. Measured on macOS 27.0: `-t exec` rejects *any* bare Mach-O with `rejected (the code is valid but does not seem to be an app)` regardless of notarization — reproduced against two vendors' genuinely notarized Developer ID CLIs — so it would have produced a false RED in criterion 1 and been permanently unreachable in criterion 2; and `syspolicy_check distribution` is Fatal (exit 70, `Notary Ticket Missing`) for anything unstapled, contradicting DIST-06. `-t install` is the assessment type matching a downloaded CLI binary, returns the required string, and still rejects the adhoc linker-signed shape this repo ships today
  3. The sha256 recorded immediately after the notarize pipe, the sha256 of the re-downloaded published asset, the cosign-signed subject, and the SLSA-attested subject are all the same value for each darwin binary — and a deliberately mis-ordered pipe makes them diverge, so the ordering is measured rather than trusted from GoReleaser's documentation (SIGN-04)
  4. The full CLI and MCP integration suite runs green against the notarized, hardened-runtime binary **itself** rather than a locally rebuilt one, so a library-validation load failure surfaces as a test failure instead of as a user's first-run crash
  5. `docs/RELEASE.md` states the shipped guarantee exactly — notarized, online-verified, **not stapled** — names offline first launch as a known limitation, and gives a reader the literal `xattr` + `spctl` commands to reproduce criterion 2 themselves

**Notes**: Promoted from backlog **999.5**, whose captured measurements stand: `codesign -dvv` reports darwin/arm64 as `adhoc, linker-signed` with `TeamIdentifier=not set` (the Go linker emits this so the Apple Silicon kernel will exec the binary — it satisfies the kernel, not Gatekeeper), darwin/amd64 as `code object is not signed at all`, and `spctl -a -vv -t exec` returns **rejected** for both. cosign is a different mechanism entirely — a detached Sigstore sidecar verified in-process by `internal/upgrade`, not an embedded `LC_CODE_SIGNATURE` — and does nothing for Gatekeeper. The affected population is browser downloaders from the GitHub Releases page; a binary fetched by the real `codegraph upgrade` path was measured to carry only `com.apple.provenance`. 999.5's open asset-shape question (a bare Mach-O can be notarized but not stapled) is resolved across Phases 1 and 3: archives ship *alongside* raw binaries, and stapling is out of scope because `.zip` and bare Mach-O are both categorically unstaplable and Quill has no staple command.

**Plans**: 7/7 plans executed

Plans:
**Wave 1**

- [x] 02-01-PLAN.md — tracer: build `verify:gatekeeper` end-to-end against a published asset (D-19 oracle: `spctl -a -vv -t install`, verdict by exit status) and record the SIGN-03 RED baseline on v0.5.1, settling the synthetic-quarantine question

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — apply the D-18 ruling: cosign moves to the release-scoped `signs:` pipe, `notarize:` goes live with explicit darwin ids and an env-gated `enabled:`, the false rationale comment is retracted, and every goreleaser caller gets a notarize-reachability verdict
- [x] 02-03-PLAN.md — `CODEGRAPH_TEST_BIN` seam in both real-binary harnesses, with a resolver that aborts by name rather than silently rebuilding

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-04-PLAN.md — guarded maintainer-only notarize rehearsal (D-08/D-09) and the D-07 one-time mis-order mutation, with the cosign subject determined by `cosign verify-blob` against a separately-built pre-sign baseline
- [x] 02-05-PLAN.md — `docs/RELEASE.md` states the guarantee exactly (notarized, online-verified, not stapled), names the offline limitation, and gives the reproduction commands

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 02-06-PLAN.md — Apple secrets on the single OIDC-bearing release job with a runtime-enumerating scoping test, plus the post-release Gatekeeper and notarized-suite jobs

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 02-07-PLAN.md — cut the real release and record criteria 2, 3 and 4 against the re-downloaded published assets

### Phase 3: Homebrew Tap & Cask

**Goal**: A macOS user installs codegraph the way they install everything else — `brew install` — and it keeps working across releases rather than only on the day it was hand-checked.
**Depends on**: Phase 2 (the cask ships notarized bytes) and Phase 1 (the `.zip` the cask points at)
**Requirements**: BREW-01, BREW-02, BREW-03, BREW-04, BREW-05, BREW-06
**Success Criteria** (what must be TRUE):

  1. On a machine with no prior codegraph, `brew tap seanb4t/tap && brew install codegraph` completes and the installed binary runs a real command — verified cold against the cask GoReleaser both rendered and pushed inside the same automated release run (BREW-01, BREW-02). **Amended 2026-08-10 (plan 03-05):** previously required "at least one release later, against a cask GoReleaser regenerated rather than the one hand-checked at first publish." That wording guarded against a curated-tap workflow this project does not have — GoReleaser renders the cask and pushes it inside one automated CI run, with no hand-check step for a first-release cask to be dependent on, so a second release proves nothing about that specific worry. The maintainer reduced this phase's scope to one release: a second release would have exercised GoReleaser's tap-push UPDATE path and `brew upgrade`, both of which are code this project does not own, cannot patch, and which surface on the next natural release regardless — an accepted, named gap (`03-EVIDENCE.md` "Scope reduction, recorded plainly"), not a silently dropped requirement. The cold install verified here (`03-EVIDENCE.md` "BREW-01 — the cold install") is against the real `v0.8.0` release's tap-published cask, cross-checked by sha256 against the real downloaded assets
  2. After that install, `codegraph` completes its subcommands in bash, zsh **and** fish, and `man codegraph` renders — all generated from the binary at cask-build time, so a new subcommand appears without anyone editing a committed completion file (BREW-03, BREW-04)
  3. The cask's `hooks.post.install` block runs the installed binary and asserts two things about the result — that man-page generation produced more than one page, and that the reported version equals the cask's own declared version — and is demonstrated to **fail** on each assertion independently when pointed at a deliberately broken binary, so a broken cask is caught before a user hits it rather than passing vacuously. **Amended 2026-08-10 (plan 03-04):** previously named a cask `test:` block, a stanza Homebrew Casks do not have; see `.planning/REQUIREMENTS.md` BREW-05's amendment note and `03-EVIDENCE.md` for the falsification evidence (BREW-05)
  4. **Amended 2026-08-10 (D-19, plan 03-05):** this criterion is split into two halves with distinct evidentiary status, and neither is claimed as the other. **Half one — failure-and-recovery — is a STRUCTURAL ARGUMENT, not executed evidence:** the pinned GoReleaser module's own source shows the cask publisher runs strictly after the release publisher in the publish pipeline (`internal/pipe/publish/publish.go`'s own comment, "brew et al use the release URL, so, they should be last"), so a failed tap push cannot corrupt an already-complete release — but no failed push has ever been observed, none is planned, and a rehearsal was considered and rejected by decision (D-18R). See `03-EVIDENCE.md` "BREW-06, half one" for the argument, its citations, and the named remedy that would close the gap. **Half two — release integrity — IS executed evidence:** the existing permanent post-release verification re-verifies checksums, cosign bundles, SBOMs and build provenance against re-downloaded published assets on every release, and this phase's one cut release (`v0.8.0`) shows an asset-list shape identical to the prior release (`v0.7.0`) — 17 entries each, differing only by the version string — with no duplicated and no orphaned entries. See `03-EVIDENCE.md` "BREW-06, half two" (BREW-06)
  5. The token that writes the tap can write `seanb4t/homebrew-tap` and nothing else — demonstrated by that same token being refused a write to `seanb4t/codegraph-go` (BREW-02)

**Notes**: Consumes seed **SEED-002**. `homebrew_casks:` is the block, not the deprecated `brews:` (GoReleaser v2.10; Homebrew's own maintainers recommend casks for GoReleaser-precompiled binaries per Homebrew/brew #20291). The default `GITHUB_TOKEN` cannot write cross-repo, so a dedicated least-privilege PAT is required. `gh`/`lazygit`'s build-time updater-disable ldflag does not transfer — casks ship precompiled binaries — so the cask-compatible analogue is a `homebrew_casks.hooks.post.install` sentinel, which is also the most robust signal Phase 4 can key on.

**Plans**: 4/5 plans executed

Plans:
**Wave 1**

- [x] 03-01-PLAN.md — confirm the one-way tap/cask naming, then a tracer: `codegraph man` plus a minimal `homebrew_casks:` block, rendered by GoReleaser and installed by real Homebrew, with the hook executing the installed binary

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-02-PLAN.md — complete the cask: D-11's two positive assertions, the Phase-4 sentinel, symmetric uninstall, three-shell completions, and shape tests asserting properties rather than literals
- [x] 03-03-PLAN.md — the tap repository, a second GitHub App installed on it alone, the mint placed by a measured job-output verdict, and a release that halts on a missing or non-distinct tap credential

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 03-04-PLAN.md — watch the install gate fail twice and the tap credential be refused once, then amend BREW-05 and criterion 3 to name the mechanism that exists (D-09)

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 03-05-PLAN.md — cut two releases, install cold from the second's regenerated cask, and record criterion 4's split: an argument for the failure half (D-18R), executed evidence for the integrity half

### Phase 4: `codegraph upgrade` × Homebrew

**Goal**: Neither install path lies about what is installed — `codegraph upgrade` recognizes a Homebrew-managed install, steps aside with an actionable pointer, and never mutates the Cellar behind brew's back.
**Depends on**: Phase 3 for **acceptance**. Implementation can proceed in parallel with Phases 2–3 against a constructed Cellar-shaped symlink tree in a temp dir, and the `Run`-branch logic is unit-testable through the existing `internal/upgrade` func-var seam — but the phase is not done until it has been run against the real tap Phase 3 publishes, because a synthetic layout can only prove the mechanism, not that it matches what actually ships.
**Requirements**: UPGR-01, UPGR-02, UPGR-03
**Success Criteria** (what must be TRUE):

  1. After a genuine `brew tap` + `brew install codegraph` from the Phase-3 tap, `codegraph upgrade` refuses, names `brew upgrade codegraph` in its message, and the Cellar binary's sha256 and mtime are unchanged afterward — measured, not assumed from the refusal message (UPGR-01)
  2. `codegraph upgrade --check` under that same install reports the available version and how to get it, and mutates nothing — same sha256/mtime evidence, and `brew upgrade codegraph` still succeeds afterward (UPGR-03)
  3. Detection fires on a resolved-symlink Cellar layout under Apple Silicon `/opt/homebrew`, Intel `/usr/local`, a custom prefix, and linuxbrew — and does **not** fire on a non-brew binary sitting at a path that merely contains the string `Cellar`. The false-positive case is an executing test, not a comment, because a path-prefix guess is the exact shape of gate this repo keeps finding cannot fire (UPGR-02)
  4. A non-brew install on a machine where `brew` is absent from `PATH` upgrades normally, so the detection can never turn `codegraph upgrade` into a hard dependency on Homebrew being present (UPGR-02)

**Plans**: TBD

Plans:

- [ ] TBD (populate with /gsd-plan-phase 4)

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
| ----- | --------- | -------------- | ------ | --------- |
| 1. Cross-Compile Spike & `goreleaser release` Migration | v0.5.0 | 6/6 | Complete    | 2026-08-08 |
| 2. Apple Signing & Notarization | v0.5.0 | 7/7 | Complete    | 2026-08-09 |
| 3. Homebrew Tap & Cask | v0.5.0 | 4/5 | In Progress|  |
| 4. `codegraph upgrade` × Homebrew | v0.5.0 | 0/TBD | Not started | - |
| 999.2. tmux e2e/UAT test harness | Backlog | 0/0 | Not started | - |
| 999.4. CheckRegression positivity guard | Backlog | 0/0 | Not started | - |

## Backlog

### Phase 999.2: tmux e2e/UAT test harness and suite (BACKLOG)

**Goal:** [Captured for future planning] A real-PTY end-to-end test harness that drives the interactive TUI through **tmux** (send-keys + capture-pane) so the terminal actually replies to escape queries and actually scrolls — the exact conditions the current piped/non-TTY suite can never reproduce. Motivation: v1.0 Phase 7's human UAT caught two user-visible TUI bugs that BOTH the full piped automated suite AND a deep multi-agent code review missed, because they only manifest on a live TTY — G-07-1 (bare `daemon` on a TTY with an empty registry leaked the terminal's DECRQM capability-probe responses `^[[?2026;2$y^[[?2027;0$y`) and G-07-2 (both bubbletea pickers rendered inline without alt-screen → heavy flicker + blank list). bubbletea Models are unit-testable via synthetic `tea.Msg` (state transitions) but that path never renders. Scope a suite that spawns the release binary inside a tmux pane and asserts on `capture-pane` output: (a) bare `daemon` empty-registry prints ONLY `no running daemons` with no leaked escape sequences; (b) the daemon picker enters the alternate screen, renders `Running daemons` + a seeded record, and restores the main buffer on quit (no residual escapes in scrollback); (c) the install/uninstall checkbox picker renders `[x]`/`[ ]` glyphs, `space` toggles, `q`/`esc` cancels with zero config writes; (d) no flicker proxy (stable capture across N frames). Reuse the `tmux` skill's send-keys/capture-pane idioms; gate the suite behind a build tag / CI job that has tmux available (skip cleanly where it isn't). This is the missing rung between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual human UAT (thorough, unautomated).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.4: CheckRegression current-metrics positivity guard (BACKLOG)

**Goal:** [Captured for future planning] Close the degenerate-input bypass in `internal/bench.CheckRegression`, surfaced and **reproduced** during the Phase 10 security audit (recorded in `10-SECURITY.md` → "Advisory — Unregistered Surface"; also code-review finding WR-06). Calling `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and an otherwise-matching frame returns `nil` — **both** the relative RSS regression check and the absolute INDX-06 memory ceiling silently pass. The function already validates that the *baseline* metrics are positive; it never validates the *current* ones, so a zero or negative current reading reads as "no regression" instead of "unusable measurement". This is unreachable through today's only caller because `internal/bench.PeakRSSBytes` returns an error rather than a zero on failure, but `CheckRegression` is exported, its doc comment claims it "never misleads", and the phase-10 audit already showed how easily a frame-descriptor blind spot becomes a live gate failure. Scope: add a positivity/sanity check on `current` mirroring the existing baseline check, refusing rather than passing on a non-positive throughput or RSS reading, with an error naming which field was degenerate. This belongs to the repo's documented class of **gates that cannot fire** (the retracted 10.6% perf claim, the inverted `rg -qv` gate, the 51.5%-stale baseline) — so the fix must be demonstrated RED with a degenerate-input test, not merely added.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
