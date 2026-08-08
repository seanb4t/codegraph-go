# Requirements: CodeGraph Go — v0.5.0 (macOS Distribution & Homebrew)

**Defined:** 2026-08-08
**Core Value:** An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

**Milestone framing.** codegraph-go's macOS binaries are measurably un-installable by convention today: `spctl -a -vv -t exec` returns **rejected** on both darwin arches, and there is no package-manager path at all. This milestone closes both — a browser-downloaded asset passes Gatekeeper, and `brew install` works from a tap we control. Backlog **999.5** is promoted and seed **SEED-002** is consumed.

**Constraint that shapes everything.** GoReleaser's `notarize:` and `homebrew_casks:` blocks execute only under `goreleaser release`, and this pipeline has only ever run `goreleaser build --single-target`. Research (2026-08-07) established that `goreleaser release` refuses a `dist/` built elsewhere and that both escape hatches — `release --split`/`continue --merge` and the `prebuilt` builder — are GoReleaser **Pro**. A single `macos-latest` runner with `zig cc` on both linux legs is therefore the only OSS path, and REL-05 exists to prove it is reachable before anything is built on it.

**Three scoping assumptions were falsified by research and corrected before these requirements were written:** `brews:` is deprecated in favour of `homebrew_casks:`; a `.zip` cannot be stapled any more than a bare Mach-O can; and the single-runner question is answered negatively rather than open. See PROJECT.md → Key Decisions.

## v0.5.0 Requirements

### Release Pipeline (REL)

- [ ] **REL-05**: A maintainer can decide the pipeline architecture on measured evidence — whether `zig cc` cross-compiles the CGo tree-sitter dependency to linux/amd64 **and** linux/arm64 from a macOS host, proven by the resulting binaries *running on real Linux*, not by the build exiting 0
- [ ] **REL-06**: A release is cut by a single `goreleaser release` invocation, with GoReleaser owning archive, checksum, sign, and SBOM generation
- [ ] **REL-07**: Exactly one process writes `codegraph_<tag>_checksums.txt` — the hand-rolled `sha256sum` step is deleted in the same change that makes `.goreleaser.yaml`'s `checksum:` block live, so the two can never disagree
- [ ] **REL-08**: Every supply-chain claim still verifies against real published assets after the migration — `cosign verify-blob`, `slsa-verifier verify-artifact`, and a genuinely shipped prior binary self-upgrading through `codegraph upgrade`
- [ ] **REL-09**: A release carries both the raw per-platform binaries `codegraph upgrade` consumes and `.zip` archives for browser download and Homebrew, with the raw binary byte-unchanged from today (D-02/Finding 1 preserved, not amended)

### macOS Signing & Notarization (SIGN)

- [ ] **SIGN-01**: Darwin binaries are Developer ID codesigned and notarized during the release, with the certificate and App Store Connect API key held as CI secrets
- [ ] **SIGN-02**: A user who downloads a release asset in a browser and runs it is not blocked by Gatekeeper — proven by `spctl -a -vv -t exec` reporting `source=Notarized Developer ID` and `syspolicy_check distribution` passing, against an asset carrying a genuine `com.apple.quarantine` xattr
- [ ] **SIGN-03**: The Gatekeeper gate is demonstrated RED against a confirmed-applied mutation before it is trusted green — an un-notarized binary must fail it, so a green CI step, a passing `codesign -dvv`, or an Accepted `notarytool` history cannot stand in for verification
- [ ] **SIGN-04**: What cosign signs and SLSA attests is byte-identical to what a user downloads — verified by diffing sha256 across pipeline stages, because notarization mutates the Mach-O and the current pipeline never modifies a binary after building it

### Homebrew Distribution (BREW)

- [ ] **BREW-01**: A user can run `brew tap seanb4t/tap && brew install codegraph` on a clean machine and get a working binary
- [ ] **BREW-02**: The tap is published by GoReleaser's `homebrew_casks:` block on every release, authenticated by a token scoped to the tap repository alone
- [ ] **BREW-03**: The cask installs shell completions for bash, zsh, and fish, generated from the binary at cask-build time
- [ ] **BREW-04**: The cask installs man pages
- [ ] **BREW-05**: The cask carries a `test:` block exercising a real command, so a broken cask fails before a user hits it
- [ ] **BREW-06**: A failed tap push leaves an otherwise-good release intact, and re-running recovers without duplicate or orphaned assets

### Upgrade × Package Manager (UPGR)

- [ ] **UPGR-01**: `codegraph upgrade` detects a Homebrew-managed install and refuses, pointing at `brew upgrade codegraph`, and never modifies the Cellar
- [ ] **UPGR-02**: Brew detection resolves symlinks to the real install path rather than matching a hardcoded prefix, so it is correct on Apple Silicon `/opt/homebrew`, Intel `/usr/local`, a custom prefix, and linuxbrew
- [ ] **UPGR-03**: `codegraph upgrade --check` still works under a brew-managed install — read-only, no mutation — and reports how to upgrade

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

Populated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| REL-05 | TBD | Pending |
| REL-06 | TBD | Pending |
| REL-07 | TBD | Pending |
| REL-08 | TBD | Pending |
| REL-09 | TBD | Pending |
| SIGN-01 | TBD | Pending |
| SIGN-02 | TBD | Pending |
| SIGN-03 | TBD | Pending |
| SIGN-04 | TBD | Pending |
| BREW-01 | TBD | Pending |
| BREW-02 | TBD | Pending |
| BREW-03 | TBD | Pending |
| BREW-04 | TBD | Pending |
| BREW-05 | TBD | Pending |
| BREW-06 | TBD | Pending |
| UPGR-01 | TBD | Pending |
| UPGR-02 | TBD | Pending |
| UPGR-03 | TBD | Pending |

**Coverage:**
- v0.5.0 requirements: 18 total
- Mapped to phases: 0
- Unmapped: 18 ⚠️ (roadmap not yet created)

---
*Requirements defined: 2026-08-08*
*Last updated: 2026-08-08 after initial definition*
