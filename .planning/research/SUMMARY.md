# Project Research Summary

**Project:** CodeGraph Go — milestone v0.5.0 "macOS Distribution & Homebrew"
**Domain:** Release engineering — Apple Gatekeeper notarization + Homebrew distribution, added to an existing signed/attested Go release pipeline
**Researched:** 2026-08-07
**Confidence:** MEDIUM-HIGH

> **Orchestrator note (#222 self-heal).** The synthesizer agent fabricated a write restriction and returned this document inline instead of writing it. The orchestrator persisted it, reconciled the three cross-document tensions itself, and folded in two maintainer decisions taken after the researchers returned. Content is the synthesizer's; the reconciliations and the Decisions Taken section are the orchestrator's.

## Executive Summary

This milestone adds two macOS distribution capabilities — Gatekeeper-acceptable notarized binaries and a Homebrew tap — to a release pipeline that already ships cosign keyless signatures, syft SPDX SBOMs, and SLSA3 provenance, all verified end-to-end on a real release. The work is release engineering, not product engineering: almost every risk lives in the interaction between the new capabilities and the proven pipeline, not in the new capabilities themselves.

Four researchers converged independently on the same recommended shape: migrate `release.yml` from a per-platform `goreleaser build --single-target` matrix to a **single `goreleaser release` invocation on one `macos-latest` runner**, with `zig cc` cross-compiling both Linux legs from the macOS host; wire GoReleaser's OSS `notarize.macos` (Quill-backed) to the darwin binaries; publish a `.zip` archive **alongside** the existing raw-binary archive via a second `archives:` entry keyed by `id`; publish the tap through `homebrew_casks:`; and teach `codegraph upgrade` to detect a Homebrew-managed install and refuse.

The dominant risk is narrow and now well-identified. The single-runner question that PROJECT.md flagged as unconfirmed is **answered negatively**: both `release --split`/`continue --merge` and the `prebuilt` builder are GoReleaser Pro, and `goreleaser release` refuses to consume a `dist/` produced elsewhere. That makes the single-macOS-runner path a hard constraint rather than a preference — and reduces the open question to one empirical unknown: whether `zig cc` cross-compiles this repo's CGo tree-sitter dependency to **linux/amd64 and linux/arm64 from a macOS host**. Today only linux/arm64 is zig-crossed, and from a Linux host. That spike is the milestone's highest-leverage risk and belongs in the first phase.

## Key Findings

### Recommended Stack

No new Go dependencies. Every addition is release-pipeline configuration plus CI secrets. GoReleaser v2.17.1 (already pinned in `go.tool.mod` and both workflows) ships everything needed in its OSS distribution.

**Core additions:**
- **`notarize.macos`** (GoReleaser OSS, Quill-backed): notarizes darwin binaries. Cross-platform — GoReleaser's own example runs it on `ubuntu-latest`; no macOS runner and no Keychain required. `notarize.macos_native` is the Pro/macOS-only variant and buys nothing here, since codegraph ships a bare CLI rather than an app bundle.
- **`homebrew_casks:`** (GoReleaser OSS since v2.10): publishes the tap. `brews:` is deprecated.
- **A second `archives:` entry keyed by `id`**: adds a `.zip` for the browser-download and cask paths. Multiple archives blocks are natively supported, so the existing raw-binary archive is untouched.
- **`binary_signs:`** rather than `signs:` with `artifacts: binary` — the v2.2+ pipe built specifically for the raw-binary-plus-archive shape.
- **`zig cc`** extended to both Linux legs (already used for linux/arm64). `goreleaser/example-zig-cgo` is the official demonstration of the CGo-cross pattern.

**Six new CI secrets:** Developer ID certificate (base64 P12) + its password; App Store Connect API key, issuer ID, and key ID; and a Homebrew tap PAT. The default `GITHUB_TOKEN` **cannot** write cross-repo, so the tap push needs a dedicated PAT scoped to `seanb4t/homebrew-tap` only.

**What NOT to add:** nothing that duplicates cosign, syft, or SLSA — those keep their current wiring and identities. No `gon`. No GoReleaser Pro. No `.pkg`/`.dmg` builder (see the stapling finding).

### Expected Features

**Table stakes:**
- Darwin binaries that pass `spctl -a -vv -t exec` on a genuinely quarantined download, reporting `source=Notarized Developer ID`.
- `brew tap seanb4t/tap && brew install codegraph` working on a clean machine.
- `codegraph upgrade` refusing under a Homebrew-managed install with a pointer to `brew upgrade codegraph`. **This is Homebrew policy, not hygiene** — the Acceptable Formulae rules require self-update behaviour to be disabled.

**Deferrable:** shell completions shipped in the cask; full offline-staplable distribution.

**Anti-features:** auto-submitting to homebrew-core; a GUI installer; a `.pkg` writing outside the prefix; an updater that silently mutates the Cellar.

**Precedent.** GoReleaser's own project dogfoods exactly the target shape — `notarize.macos` + `homebrew_casks`, **no staple step**. `gh` and `lazygit` both solve the self-updater-vs-package-manager conflict at compile time via a build-time ldflag; that trick does not transfer directly to a cask (casks ship precompiled binaries rather than building per-install), so the cask-compatible analogue is a `homebrew_casks.hooks.post.install` sentinel file. `atuin` explicitly rejects self-replacing updaters under a package manager.

**Brew detection must resolve symlinks to the real Cellar path, not guess the prefix.** `google-gemini/gemini-cli` PR #14727 is a real, documented false-positive regression from prefix-guessing; the fix that shipped there queries `brew --prefix` live and resolves symlinks. Apple Silicon `/opt/homebrew` vs Intel `/usr/local` vs custom prefixes vs linuxbrew vs migrated systems are all real variants.

### Architecture Approach

**Before:** four parallel `goreleaser build --single-target --clean` matrix jobs, then an assemble job layering cosign, syft, a hand-rolled `sha256sum`, and SLSA subject collection.

**After:** one `macos-latest` job running the whole `goreleaser release`, with GoReleaser owning archive, checksum, sign, and SBOM natively, and `zig cc` cross-compiling both Linux legs.

**Artifact flow and its correctness invariant.** GoReleaser's pipe ordering runs `notarize` **before** `archive`/`checksum`/`sign`/`sbom`, so nothing signs pre-notarization bytes while shipping post-notarization ones. That ordering is correctness-safe *by design* — but per this repo's standing rule it must become a **measured acceptance gate** (sha256 diffed at each stage), not a property trusted from documentation.

**Dual-asset topology.** The raw binary (`codegraph_<tag>_<goos>_<goarch>`, no extension) is consumed by `internal/upgrade.releaseAssetName()` and must remain byte-unchanged — decision D-02 survives intact. The `.zip` serves browser downloads and the cask. Distinct name templates keep the two from colliding.

**The checksums collision is already latent, not hypothetical.** `.goreleaser.yaml`'s currently-dead `checksum:` block and `release.yml`'s hand-rolled `sha256sum` step both emit `codegraph_<tag>_checksums.txt`. The moment `goreleaser release` runs, both wake up and race. **Resolution is deletion of the hand-rolled step in the same change** — not reconciliation of two writers.

**New component:** `seanb4t/homebrew-tap`, a separate repository GoReleaser pushes to. Failure mid-release must not corrupt an otherwise-good release; a deliberate tap-push failure should prove the rest is unaffected and a re-run should recover without duplication.

### Critical Pitfalls

1. **`goreleaser release` cannot consume a pre-built `dist/`** from other runners; `--split`/`--merge` and `prebuilt` are Pro-only. Falsifies the matrix-then-release architecture outright.
2. **`notarytool` Accepted ≠ `spctl` pass.** The notary ticket may not cover the final bytes as shipped. Documented repeatedly in Apple's own forums, including recent Tahoe-era threads.
3. **Four checks that feel like verification but are not:** a green CI step; `codesign -dvv` passing (**it passes on adhoc-signed binaries too — this repo's darwin/arm64 already passes it today while `spctl` says `rejected`**); `notarytool history` showing Accepted; `spctl` run on a file that was never quarantined. The single trustworthy check is forcing a real `com.apple.quarantine` xattr on the **actually-published asset** and running `spctl -a -vv -t exec`, expecting `source=Notarized Developer ID`.
4. **Stapling a bare Mach-O or a `.zip` is impossible.** Apple's `stapler` attaches tickets only to `.app`/`.pkg`/`.dmg`; Quill has no staple command at all. Platform constraint, not a tooling gap.
5. **Checksums-file collision** (above) — latent in the repo today.
6. **cosign/SLSA byte divergence** if the pipeline re-touches a raw binary after signing. Invariant: build → notarize → cosign-sign → upload, byte-unchanged thereafter.
7. **Hardened Runtime entitlements** are likely a non-issue for a statically-linked CGo binary (C compiled in, not `dlopen()`-loaded), but this is reasoning, not measurement — verify by running the full CLI/MCP suite against a notarized binary.
8. **Tap-push race:** a formula published before the GitHub Release asset exists points at a 404.

## Implications for Roadmap

### Phase 1: De-risking Spike & `goreleaser release` Restructure

Prove `zig cc` cross-compiles this repo's CGo tree-sitter dependency to linux/amd64 **and** linux/arm64 from a macOS host, and that the resulting binaries run on real Linux. Then perform the mechanical `build` → `release` migration: consolidate to one macOS runner, delete the hand-rolled `sha256sum` step, adopt `binary_signs:`, and **re-prove every existing supply-chain claim under the new mechanism** — `cosign verify-blob`, `slsa-verifier verify-artifact`, and a real `codegraph upgrade` self-upgrade against published assets. Zero new user-facing capability; the deliverable is that nothing regressed.

Owns pitfalls 1, 5, 6.

### Phase 2: Apple Notarization

Wire `notarize.macos` with the Developer ID certificate and App Store Connect API key. Acceptance is the forced-quarantine `spctl` check against the real published asset — explicitly **not** any of the four false-positive checks in pitfall 3. Demonstrate the gate RED first: it must fail against the current un-notarized binary before it is trusted to pass against the notarized one. Verify the notarize-before-sign ordering by diffing sha256 at each stage rather than trusting the documented pipe order.

Owns pitfalls 2, 3, 7.

### Phase 3: Archives & Homebrew Tap

Add the second `archives:` entry producing `.zip`, create `seanb4t/homebrew-tap`, and wire `homebrew_casks:` with a least-privilege PAT. Acceptance is a real `brew tap && brew install` on a clean machine — plus a deliberately failed tap push proving the rest of the release is unaffected and a re-run recovers without duplication.

Owns pitfalls 4 (accepted, documented) and 8.

### Phase 4: `codegraph upgrade` Brew Detection

Detect a Homebrew-managed install by resolving symlinks to the real Cellar path (or a cask post-install sentinel), never by prefix-guessing. Refuse with a pointer to `brew upgrade codegraph`. Development can start in parallel with Phase 3, but final acceptance must run against the real tap from Phase 3.

### Phase Ordering Rationale

Risk is front-loaded deliberately. The zig-cross spike gates everything — if it fails, the entire chosen architecture is unreachable in OSS GoReleaser and the milestone must be rescoped, so it must be answered before any restructuring work is committed. Phase 1 then changes the *mechanism* without changing the *product*, which makes any regression attributable to the migration alone rather than tangled with new capability. Notarization precedes archives because the cask must ship notarized bytes. Brew detection comes last because its only honest acceptance test needs a real tap to exist.

### Research Flags

- **Zig-cross-linux-from-macOS with CGo tree-sitter is unproven in this repo.** Single highest-leverage risk. Spike before committing to the restructure.
- **Hardened Runtime applicability to this specific static-CGo build is inference, not measurement.** Verify empirically.
- **GoReleaser's `release:` pipe behaviour against a Release object release-please already created** is documented in prose but was not smoke-tested against a real tag in this pass. D-06R means release-please creates the tag and Release first; GoReleaser must upload into it rather than create a competing one.
- **The `xattr` quarantine-flag encoding** used in the RED-check script is community-documented rather than from a canonical Apple source. A real browser download should remain the primary check, with the scripted xattr as the automatable proxy.

## Decisions Taken (post-research, maintainer)

Two questions the research raised were put to the maintainer and answered before roadmapping:

1. **Notarization bar: online-only, gap documented.** Ship `notarize.macos` on a `.zip`; Gatekeeper passes via online ticket lookup. Offline first launch remains a **documented known limitation**, matching what GoReleaser itself ships for its own CLI. No GoReleaser Pro licence, no hand-built `.app` wrapper. Stapling is deliberately deferred, not forgotten.
2. **`homebrew_casks:` confirmed, correcting `brews:`.** PROJECT.md's recorded decision ("formula published by GoReleaser's `brews:` block") was falsified by three independent researchers and has been corrected.

## Confidence Assessment

| Area | Level | Basis | Gaps |
|------|-------|-------|------|
| GoReleaser Pro/OSS boundary (split/merge, prebuilt, notarize variants, dmg) | HIGH | STACK fetched the boundary three separate ways and corroborated against the maintainer's own blog and GitHub issue #2320; PITFALLS reached the same conclusion independently. ARCHITECTURE self-rated this MEDIUM as a single-pass fetch — the independent corroboration raises the combined confidence to HIGH rather than averaging down | None material |
| Stack config schema and secrets shape | HIGH | Official GoReleaser docs, multiple fetches, versions confirmed live via `gh api` on 2026-08-07 | Apple Developer Portal UI not verified live |
| Stapling constraint (`.zip` and bare Mach-O cannot be stapled) | HIGH | Apple's own documentation, Quill's command surface, and GoReleaser's docs — three independent researchers reached it separately | None material |
| `brews:` deprecated → `homebrew_casks:` | HIGH | GoReleaser PR #5780 and deprecation docs, Homebrew/brew PR #20291 with maintainers' direct statements, plus GoReleaser's own dogfooded config | None material |
| Repo-internal facts (release.yml, .goreleaser.yaml, Taskfile.yml, internal/upgrade) | HIGH | Direct file reads with line citations | None |
| Notarization false-positive check family | HIGH | Multiple independent Apple Developer Forum threads on the exact Accepted-but-rejected shape, including recent ones | Predictions not yet observed here; phase-specific RED tests required |
| Single-runner zig-cross recommendation | MEDIUM | Logically sound, consistent with this repo's existing zig-cross precedent, and officially demonstrated in `goreleaser/example-zig-cgo` — but that example was not cloned or built, and the macOS-host variant is unexercised here | **The milestone's primary spike** |
| Hardened Runtime applicability | MEDIUM | Mechanism well-documented; applicability to this static-CGo build is inference | Requires empirical check |
| Homebrew tap PAT scope, brew detection | MEDIUM-HIGH | Official Homebrew docs HIGH; PAT scoping corroborated across several non-primary sources | Detection untested against a real Homebrew layout |

### Gaps to Address

1. Zig-cross-linux-from-macOS with CGo tree-sitter — **Phase 1, blocking.**
2. Real Gatekeeper validation via forced quarantine on the published asset — **Phase 2, acceptance gate.**
3. Notarize-before-sign ordering verified by sha256 diff at each stage — **Phase 2.**
4. GoReleaser `release:` behaviour against a release-please-created Release — **Phase 1.**
5. Brew-managed Cellar layout detection against a real install — **Phase 4.**

## Sources

### Primary (HIGH confidence)

- GoReleaser official docs: `/customization/partial/` (split/merge Pro-only, verbatim), `/customization/builds/builders/prebuilt/` (prebuilt Pro-only, verbatim), `/customization/sign/notarize/`, `/customization/publish/homebrew_casks/`, `/customization/package/archives`, `/customization/release`, `/customization/sign/binary_sign/`, `/cmd/goreleaser_release/`, `/resources/errors/dirty/`
- goreleaser/goreleaser issue #2320 — `release` cannot consume a pre-existing `dist/`; no `--skip-build` exists
- goreleaser/goreleaser PR #5780 — introduced `homebrew_casks`, deprecated `brews`
- `github.com/goreleaser/goreleaser/.goreleaser.yaml` (main) — the project's own dogfooded notarize + cask config, no staple step
- Carlos Becker (GoReleaser maintainer), "GoReleaser Split and Merge" — maintainer-authored, directly authoritative on Pro status
- `github.com/anchore/quill` README + llms.txt (raw.githubusercontent.com) — command surface confirming no `staple` command
- Apple: `developer.apple.com/documentation/security/customizing-the-notarization-workflow` and `.../notarizing-macos-software-before-distribution` — staple-ability constraints
- Homebrew: `docs.brew.sh/Acceptable-Formulae` (self-update policy), `docs.brew.sh/Adding-Software-to-Homebrew`, `docs.brew.sh/Formula-Cookbook`, `docs.brew.sh/FAQ`
- Homebrew/brew PR #20291 — maintainers' direct statements on formula-vs-cask for precompiled binaries
- `cli/cli` (issues #6949, #10242, #2141; PRs #70784, #4247; its macOS install documentation) and `jesseduffield/lazygit` (its updater package and user-config gate, PR #189) — build-time updater-disable precedent. **External repositories, cited for precedent only** — no file in this repo is implied
- `google-gemini/gemini-cli` PR #14727 — real false-positive brew-detection bug and its `brew --prefix` + symlink-resolution fix
- `gh api repos/goreleaser/goreleaser/releases/latest` executed locally 2026-08-07 — confirms v2.17.1 current
- This repo, read directly 2026-08-07: `.goreleaser.yaml`, `.github/workflows/release.yml`, `Taskfile.yml`, `internal/upgrade/{upgrade,verify,release,swap}.go`, `.planning/PROJECT.md`

### Secondary (MEDIUM confidence)

- Apple Developer Forums threads 128497, 794080, 817887, 767998, 706638, 706379, 689337, 651808, 723397, 673889, 119445, 814080 — Accepted-but-rejected family, Team ID mismatch, `com.apple.provenance` behaviour, entitlement traps. Forum posts rather than official docs, but several are from Apple DTS engineers and are internally consistent across threads
- The Eclectic Light Company: "How notarization works", "Notarization: the hardened runtime", "Building and notarizing command tools as Universal binaries" — independent, widely cited, cross-corroborated with Apple docs
- `goreleaser/example-zig-cgo` — official GoReleaser org example; **not cloned or built in this pass**
- goreleaser/goreleaser issue #1120 — documented parallel-tap-push race
- Homebrew/discussions #664, Homebrew/brew issue #16044 — real prefix-detection edge cases on migrated systems
- Kayla McArthur, "Apple Codesigning In Depth: Part I" — codesign vs spctl distinction, ad-hoc signature behaviour
- `devenjarvis/lathe`, `dash0-cli` brew-tap-migration (2026-06), `derailed/k9s` packaging, atuin docs/forum — real shipping configs and a dated formula→cask migration
- Community tap/PAT setup writeups (DNSControl docs, mcginniscommawill.com, dev.to, bindplane.com, engineered.at) — used only to confirm mechanical config shape, never for judgment calls

### Tertiary (LOW confidence)

- Community writeups on unsigned-CLI user-facing symptoms (donatstudios.com, ctxloom.dev, NJannasch/vibecockpit PR #24) — cross-checked against each other and Apple Forums, but non-primary
- Hand-rolled `.pkg` staple pipelines (octet-stream.net, scriptingosx.com) — relevant only if the deferred stapling decision is ever revisited
- General "no special entitlements for a plain Go CLI" blog consensus — **not CGo-tree-sitter-specific**; explicitly flagged as needing an empirical Phase-2 check rather than treated as settled
- The `xattr` quarantine-flag encoding used in the RED-check script — community-documented, not canonical Apple; a real browser download stays the primary check

---
*Synthesized from STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md — researched 2026-08-07*
