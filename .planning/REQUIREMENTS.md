# Requirements: CodeGraph Go — v0.5.0 (macOS Distribution & Homebrew)

**Defined:** 2026-08-08
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

**Milestone framing.** codegraph-go's macOS binaries are measurably un-installable by convention today: `spctl -a -vv -t exec` returns **rejected** on both darwin arches, and there is no package-manager path at all. This milestone closes both — a browser-downloaded asset passes Gatekeeper, and `brew install` works from a tap we control. Backlog **999.5** is promoted and seed **SEED-002** is consumed.

**Constraint that shapes everything.** GoReleaser's `notarize:` and `homebrew_casks:` blocks execute only under `goreleaser release`, and this pipeline has only ever run `goreleaser build --single-target`. Research (2026-08-07) established that `goreleaser release` refuses a `dist/` built elsewhere and that both escape hatches — `release --split`/`continue --merge` and the `prebuilt` builder — are GoReleaser **Pro**. A single `macos-latest` runner with `zig cc` on both linux legs is therefore the only OSS path, and REL-05 exists to prove it is reachable before anything is built on it.

**Three scoping assumptions were falsified by research and corrected before these requirements were written:** `brews:` is deprecated in favour of `homebrew_casks:`; a `.zip` cannot be stapled any more than a bare Mach-O can; and the single-runner question is answered negatively rather than open. See PROJECT.md → Key Decisions.

## v0.5.0 Requirements

### Release Pipeline (REL)

- [x] **REL-05**: A maintainer can decide the pipeline architecture on measured evidence — whether `zig cc` cross-compiles the CGo tree-sitter dependency to linux/amd64 **and** linux/arm64 from a macOS host, proven by the resulting binaries *running on real Linux*, not by the build exiting 0
- [x] **REL-06**: A release is cut by a single `goreleaser release` invocation, with GoReleaser owning archive, checksum, sign, and SBOM generation
- [x] **REL-07**: Exactly one process writes `codegraph_<tag>_checksums.txt` — the hand-rolled `sha256sum` step is deleted in the same change that makes `.goreleaser.yaml`'s `checksum:` block live, so the two can never disagree
- [x] **REL-08**: Every supply-chain claim still verifies against real published assets after the migration — `cosign verify-blob`, `gh attestation verify` (reworded 2026-08-08, plan 01-04: the pre-migration verifier command architecturally cannot verify `actions/attest-build-provenance` output — confirmed by Phase-1 research, not a scope change; D-10 named this replacement command in advance), and a genuinely shipped prior binary self-upgrading through `codegraph upgrade`
- [x] **REL-09**: A release carries both the raw per-platform binaries `codegraph upgrade` consumes and `.zip` archives for browser download and Homebrew, with the raw binary byte-unchanged from today (D-02/Finding 1 preserved, not amended)

### macOS Signing & Notarization (SIGN)

- [x] **SIGN-01**: Darwin binaries are Developer ID codesigned and notarized during the release, with the certificate and App Store Connect API key held as CI secrets
- [x] **SIGN-02**: A user who downloads a release asset in a browser and runs it is not blocked by Gatekeeper — proven by `spctl -a -vv -t install` reporting `accepted` with `source=Notarized Developer ID` (exit 0), against an asset carrying a genuine `com.apple.quarantine` xattr confirmed present via `xattr -p`. **Amended 2026-08-09 (D-19)**: previously specified `-t exec` plus `syspolicy_check distribution`. Both were measured unachievable for this artifact shape on macOS 27.0 — `-t exec` returns `rejected (the code is valid but does not seem to be an app)` for *any* bare Mach-O regardless of notarization (reproduced against Docker's and OpenAI's genuinely Developer-ID-signed, notarized, hardened-runtime CLIs), and `syspolicy_check distribution` returns `Notary Ticket Missing` / Severity **Fatal** / exit 70 for anything not stapled, which DIST-06 puts permanently out of scope. `-t install` is the assessment type that matches a downloaded CLI binary and returns the exact required string; its RED baseline still fires (an adhoc linker-signed binary — this repo's shape today — returns `rejected`, exit 3)
- [x] **SIGN-03**: The Gatekeeper gate is demonstrated RED against a confirmed-applied mutation before it is trusted green — an un-notarized binary must fail it, so a green CI step, a passing `codesign -dvv`, or an Accepted `notarytool` history cannot stand in for verification
- [x] **SIGN-04**: What cosign signs and SLSA attests is byte-identical to what a user downloads — verified by diffing sha256 across pipeline stages, because notarization mutates the Mach-O and the current pipeline never modifies a binary after building it

### Homebrew Distribution (BREW)

- [x] **BREW-01**: A user can run `brew tap seanb4t/tap && brew install codegraph` on a clean machine and get a working binary
- [x] **BREW-02**: The tap is published by GoReleaser's `homebrew_casks:` block on every release, authenticated by a token scoped to the tap repository alone
- [x] **BREW-03**: The cask installs shell completions for bash, zsh, and fish, generated from the binary at cask-build time
- [x] **BREW-04**: The cask installs man pages
- [x] **BREW-05**: The cask carries a mechanism exercising a real command, so a broken cask fails before a user hits it — `hooks.post.install` runs the installed binary's man-page generation and its `version --json` output, and raises (rolling back the install) if either the man pages are absent or the reported version disagrees with the cask's own declared version. **Amended 2026-08-10 (plan 03-04)**: previously specified a cask `test:` block. Measured unachievable, not merely inconvenient: Homebrew Casks carry no `test` stanza (`Cask::DSL.instance_methods(false)` on Homebrew 6.0.16 has no `test` method), `brew test` operates only on installed formulae ("Run the test method provided by an installed formula" — `brew test --help`), and the pinned GoReleaser v2.17.1 `HomebrewCask` struct (`pkg/config/config.go`) exposes no such field. The replacement mechanism — `hooks.post.install`'s two positive assertions (D-11) — was demonstrated RED against two independently confirmed-applied, byte-clean-reverted mutations (a binary that cannot execute; a binary that executes and reports the wrong version), see `03-EVIDENCE.md` §"BREW-05 — the install gate demonstrated RED"
- [x] **BREW-06**: A failed tap push leaves an otherwise-good release intact, and re-running recovers without duplicate or orphaned assets

### Upgrade × Package Manager (UPGR)

- [x] **UPGR-01**: `codegraph upgrade` detects a Homebrew-managed install and refuses, pointing at `brew upgrade codegraph`, and never modifies the Caskroom or the Cellar. **Amended 2026-08-11 (D-01, plan 04-03)**: previously named only "the Cellar" — the formula tree this project has never shipped into. It ships a cask via `homebrew_casks:`, so Caskroom is named as the tree that is actually mutated-against; Cellar is kept as the second covered shape for a future formula path (measured: `03-02-SUMMARY.md:128` records the sentinel resolving to `/opt/homebrew/Caskroom/codegraph/<version>`; `03-EVIDENCE.md:160` records `ls /opt/homebrew/Caskroom/codegraph`)
- [x] **UPGR-02**: Brew detection resolves symlinks to the real install path rather than matching a hardcoded prefix, so it is correct on Apple Silicon `/opt/homebrew`, Intel `/usr/local`, a custom prefix, and linuxbrew, recognizing both the Caskroom and Cellar tree shapes at every prefix. **Amended 2026-08-11 (D-04R, plan 04-03)**: linuxbrew was previously scoped assuming Homebrew on Linux has no cask support; that premise was falsified by research the same day — Homebrew PR #19121 allows `binary`/`zap`-only casks to install on Linux, and Homebrew 6.0.0 shipped four further Linux-cask items — so linuxbrew detection now covers the Caskroom shape too, not Cellar alone
- [x] **UPGR-03**: `codegraph upgrade --check` still works under a brew-managed install — read-only, no mutation — and reports how to upgrade

## v0.5.x Requirements

Deferred to a follow-up release. Tracked but not in this roadmap.

### Distribution polish

- **DIST-06**: Offline-safe first launch on macOS via a stapled container (`.pkg` or `.dmg`), if evidence shows real users hitting the offline case
- **BREW-07**: homebrew-core submission, so users need no `brew tap` step, once adoption justifies the review queue

## Out of Scope

| Feature | Reason |
|---------|--------|
| Stapling / offline-safe first launch | `stapler` attaches tickets only to `.app`/`.pkg`/`.dmg`; a bare Mach-O **and** a `.zip` categorically cannot be stapled, and Quill has no staple command. Shipping notarized-but-unstapled matches GoReleaser's own choice for its own CLI. Gatekeeper still passes with network. Deferred as DIST-06, revisitable on evidence — and reachable without GoReleaser Pro via hand-rolled `pkgbuild`/`productbuild`/`productsign`/`xcrun stapler` |
| homebrew-core submission | External review queue and notability criteria whose outcome we cannot schedule. Own tap ships now; deferred as BREW-07 |
| A `brews:` formula, or a parallel formula kept "just in case" | Deprecated in GoReleaser v2.10; Homebrew's own maintainers recommend casks for GoReleaser-precompiled binaries (Homebrew/brew #20291). A parallel definition directly contradicts the deprecation |
| GoReleaser Pro | Not adopted. Held as a **costed fallback** if REL-05 fails — it would preserve the native darwin matrix, but detaches from `go.tool.mod` and blinds three existing gates (`check:goreleaser`/DIST-01, `TestGoreleaserPinParity`, `tool-vuln`/VULN-01-03, the last turning `GO-2026-5932` from measured into unmeasurable). See PROJECT.md → Key Decisions |
| A `.pkg` installer that writes outside a controlled prefix | Anti-feature — users expect a CLI on `$PATH`, not a system-modifying installer |
| Ad-hoc self-signing (`codesign -s -`) as a substitute for notarization | Anti-feature — still triggers Gatekeeper warnings and does nothing for the "damaged and can't be opened" failure; Homebrew's maintainers call it hacky. Real signing is not meaningfully harder given the certificate is already held |
| Non-macOS package managers (scoop, apt, yum, nix) | Different milestone. macOS is the surface with the measured Gatekeeper failure and the requested brew path |
| Backlog 999.2 (tmux TTY harness) and 999.4 (`CheckRegression` positivity guard) | Deliberately parked to keep this milestone tight. Preserved in `ROADMAP.md` → Backlog |
| The v0.3.0 open threads (`toolslist-repeat` flake, `cmd.Wait` sweep, `stdinLingerReader` bound, `requiredCheckNames` drift, `SECURITY.md` govulncheck wording) | Parked. Recorded in PROJECT.md → Active. `GO-2026-5932` is the one exception that may be revisited on evidence, since this milestone touches `goreleaser` directly |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REL-05 | Phase 1 | Complete |
| REL-06 | Phase 1 | Complete |
| REL-07 | Phase 1 | Complete |
| REL-08 | Phase 1 | Complete |
| REL-09 | Phase 1 | Complete |
| SIGN-01 | Phase 2 | Complete |
| SIGN-02 | Phase 2 | Complete |
| SIGN-03 | Phase 2 | Complete |
| SIGN-04 | Phase 2 | Complete |
| BREW-01 | Phase 3 | Complete |
| BREW-02 | Phase 3 | Complete |
| BREW-03 | Phase 3 | Complete |
| BREW-04 | Phase 3 | Complete |
| BREW-05 | Phase 3 | Complete |
| BREW-06 | Phase 3 | Complete |
| UPGR-01 | Phase 4 | Complete |
| UPGR-02 | Phase 4 | Complete |
| UPGR-03 | Phase 4 | Complete |

**Phase mapping:**

| Phase | Name | Requirements |
|-------|------|--------------|
| 1 | Cross-Compile Spike & `goreleaser release` Migration | REL-05, REL-06, REL-07, REL-08, REL-09 |
| 2 | Apple Signing & Notarization | SIGN-01, SIGN-02, SIGN-03, SIGN-04 |
| 3 | Homebrew Tap & Cask | BREW-01, BREW-02, BREW-03, BREW-04, BREW-05, BREW-06 |
| 4 | `codegraph upgrade` × Homebrew | UPGR-01, UPGR-02, UPGR-03 |

**Coverage:**

- v0.5.0 requirements: 18 total
- Mapped to phases: 18 ✓
- Unmapped: 0
- Duplicated across phases: 0

The v0.5.x deferrals (DIST-06, BREW-07) are deliberately unmapped — they are not in this roadmap.

---
*Requirements defined: 2026-08-08*
*Last updated: 2026-08-08 — traceability populated during roadmap creation (4 phases)*
