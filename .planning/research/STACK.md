# Stack Research: macOS Gatekeeper Notarization + Homebrew Distribution

**Domain:** Go release engineering — GoReleaser-native macOS code signing/notarization and Homebrew tap publishing, layered onto an existing signed+attested pipeline
**Researched:** 2026-08-07
**Confidence:** HIGH on the Pro/OSS boundary and config schema (multiple independent official-doc fetches converged); MEDIUM on the single-runner zig-cross-from-macOS recommendation (well-documented pattern, not yet executed in this repo); LOW flagged explicitly where noted.

## Headline Answer to the Central Risk (Q1)

**`goreleaser release --split` / `goreleaser continue --merge` is Pro-only, confirmed from the official docs page itself:** goreleaser.com/customization/partial/ states in its own words *"This capability is exclusively a Pro feature"* — corroborated independently by GoReleaser's own Pro marketing page (goreleaser.com/pro/) and Carlos Becker's (GoReleaser's creator) blog post announcing the feature in v1.12-pro. The **`prebuilt` builder** (importing binaries built outside GoReleaser into its `release` lifecycle) is **also confirmed Pro-only** (goreleaser.com/customization/builds/builders/prebuilt/: *"This feature is exclusively available with GoReleaser Pro"*). Together these close off both routes to "build per-platform on separate runners, then have one `goreleaser release` invocation assemble/publish them" without a paid license. **Confidence: HIGH** — stated in GoReleaser's own docs in nearly identical wording across three independent pages.

**This does NOT force a Pro purchase.** There is a viable, well-precedented OSS path (detailed in Q1 below): run the **entire** `goreleaser release` invocation on a single **macOS** runner, with `zig cc` cross-compiling **both** linux legs (this repo already zig-cross-compiles linux/arm64 today, just from a Linux host — the pattern is host-independent). GoReleaser's own example repository (`goreleaser/example-zig-cgo`) demonstrates exactly this: CGo cross-compilation via zig from any host to any target. **Confidence: MEDIUM** — the pattern is officially documented and demonstrated, but has not been exercised in *this* repo's CGo/tree-sitter build, so Phase 1 execution should still smoke-test it before committing the CI restructure.

## Recommended Stack (Additions Only)

### Core Additions

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| GoReleaser (existing pin) | **v2.17.1** (unchanged — confirmed current: `gh api repos/goreleaser/goreleaser/releases/latest` returns `v2.17.1`, published 2026-07-26, on 2026-08-07) | Same binary, new subcommand (`release` instead of `build --single-target`) and two new config blocks | No version bump needed. `notarize.macos` (cross-platform) predates v2.1; `homebrew_casks` was introduced in v2.10; both are well inside v2.17.1. **Confidence: HIGH** (live GitHub API check + official docs version-since notes). |
| `anchore/quill` (embedded in GoReleaser — **not a separate dependency to add**) | vendored by GoReleaser's `notarize.macos` pipe | Pure-Go, cross-platform Apple code signing + notarization submission, with no `codesign`/`xcrun`/Keychain dependency | This is the mechanism that makes notarization possible **without requiring a macOS runner for the signing step itself** — confirmed directly from GoReleaser's own docs: the cross-platform `notarize.macos` GitHub Actions example runs on `runs-on: ubuntu-latest` with `distribution: goreleaser` (the free/OSS distribution, not `goreleaser-pro`). Quill's own README lists its commands (`sign`, `notarize`, `sign-and-notarize`, `p12 attach-chain`, `describe`, `submission status/logs/list`) with **no `staple` command** — Apple's `stapler` only attaches tickets to `.app`/`.pkg`/`.dmg` containers, not bare Mach-O binaries, and quill does not attempt to work around that. **Confidence: HIGH** (goreleaser.com/customization/sign/notarize/ fetched twice via different tools, converged; quill's own README fetched directly from raw GitHub). |
| `zig` (already a repo dependency for linux/arm64 cross) | pinned `0.15.1` (existing `mlugg/setup-zig@v2.2.1` step) | Extend zig-cross to **both** linux legs from a macOS host, so the entire `goreleaser release` runs on one runner | `CC="zig cc -target x86_64-linux-gnu"` / `aarch64-linux-gnu` works identically regardless of host OS — zig bundles its own libc/sysroots per target, it doesn't borrow the host's. GoReleaser's own `goreleaser/example-zig-cgo` repo demonstrates cross-compiling CGO Go binaries via zig from any host OS to any target. **Confidence: MEDIUM** — officially documented pattern with a working example repo, but this exact CGo/tree-sitter binary + this exact host/target pairing (macOS host → linux/amd64 AND linux/arm64) has not been built in this repo yet; treat as a Phase-1 spike, not a given. |

### GoReleaser Config Additions (`.goreleaser.yaml`)

**1. `notarize:` block (new top-level key) — cross-platform/OSS variant, NOT `macos_native`:**

```yaml
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'
      ids:
        - codegraph-darwin-amd64
        - codegraph-darwin-arm64
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"       # base64 P12, or a file path
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
        # entitlements: omit — see Apple Tooling section; a bare Go CLI needs none
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"              # base64 P8, or a file path
        wait: true
        timeout: 20m
```

Key facts (each independently confirmed from goreleaser.com/customization/sign/notarize/ and goreleaser.com/customization/notarize):
- **Two variants exist**: `notarize.macos` (cross-platform, Quill-backed, **OSS**, any OS) and `notarize.macos_native` (native `codesign`+`xcrun notarytool`+`productsign`, **Pro-only**, macOS-only, needed for App Bundle/DMG/PKG). This repo ships a bare CLI binary, not an app bundle — `macos_native` buys nothing here even setting Pro aside. **Confidence: HIGH.**
- `ids` filters which **build** ids get signed — confirmed to match this repo's existing `codegraph-darwin-amd64`/`codegraph-darwin-arm64` build ids.
- The signed bytes **replace the build artifact in place and flow forward** into whatever consumes it next (archive, checksum, upload) — GoReleaser's own docs: *"Once the binaries are built, the notary step does everything in a single run. The signed binaries are then used from that point forward."* This means the raw darwin binary that `internal/upgrade` will eventually hash **is** the Apple-signed, notarized binary — not a separate unsigned copy. This is consistent with D-02 (format is still "raw binary, not an archive"), but is a real behavioral change worth flagging explicitly to the roadmap: today's binaries are `adhoc, linker-signed`/unsigned; after this change they will carry a real Apple Developer ID signature. **Confidence: HIGH** on the mechanism, this is an architectural implication for the roadmap to weigh, not a stack question to resolve here.
- Does **not** require a macOS runner. Does **not** staple (see Apple Tooling section — this is a hard Apple platform constraint, not a GoReleaser gap).

**2. `homebrew_casks:` block — NOT `brews:`.** `brews:` (formula-based, "hackyish... installed pre-compiled binaries" per GoReleaser's own words) has been soft-deprecated since v2.10 and its docs page is now titled "Homebrew Formulas (deprecated)". `homebrew_casks:` is the correct, current, non-deprecated block, also introduced in v2.10 (confirmed OSS — only `alternative_names`, the `app:` DMG option, `token_type` cross-SCM publishing, and PR `check_boxes` are Pro-gated sub-features; the base cask-publishing flow is free). **Confidence: HIGH** (goreleaser.com/customization/publish/homebrew_casks/, fetched directly).

```yaml
homebrew_casks:
  - name: codegraph
    ids:
      - <archive id producing the zip, NOT the raw-binary archive id>
    binaries:
      - codegraph
    repository:
      owner: seanb4t
      name: homebrew-tap
      branch: main
      # token: use a dedicated PAT env var, see Secrets below — GITHUB_TOKEN
      # cannot write to a different repository.
    commit_author:
      name: fzy-release-please[bot]   # or a dedicated bot identity — match
      email: ...                       # whatever release-please already uses, for consistency
    commit_msg_template: "chore(cask): update codegraph to {{ .Tag }}"
    url:
      template: "https://github.com/seanb4t/codegraph-go/releases/download/{{ .Tag }}/{{ .ArtifactName }}"
```

- **Token scope**: the workflow's default `GITHUB_TOKEN` is scoped to the triggering repo only and **cannot** push to `seanb4t/homebrew-tap`. Every independent source (GoReleaser docs, DNSControl's own GoReleaser writeup, multiple community how-tos) agrees: this requires a **separate PAT** — a fine-grained token scoped to just `homebrew-tap` with "Contents: Read and Write", or a classic token with `repo` scope — stored as a repo secret (commonly named `HOMEBREW_TAP_TOKEN` or similar) and passed via `repository.token` in the cask block, or as the `GITHUB_TOKEN` env var GoReleaser reads by default for the *publish* step specifically (GoReleaser lets you override per-pipe). **Confidence: HIGH** (converged across GoReleaser's own docs and three independent community sources).
- **Binary vs. archive input**: `homebrew_casks` does **not** mandate a zip/tar archive — it can point `url.template` at any downloadable artifact, and Homebrew Cask's own staging mechanism unpacks common archive extensions automatically; a raw binary URL also works if the `binary` artifact stanza is used, which is what GoReleaser generates by default via the `binaries:` key above. **Given this repo's D-02 constraint, the cleanest shape is still to feed `homebrew_casks` from a *second*, zip-formatted `archives:` entry** (see below) rather than the raw-binary one — keeps the raw-binary asset's name/shape contract (`internal/upgrade.releaseAssetName()`) completely untouched by anything brew-related. **Confidence: MEDIUM** (documented behavior, but the "point brew straight at the raw binary vs. a zip" choice is an architecture decision for the roadmap, not something GoReleaser forces either way).
- **Signing note from the docs themselves**: *"casks are supposed to be signed"* — GoReleaser's own homebrew_casks docs page explicitly calls out that unsigned casks either need an `xattr`-based quarantine-removal post-install hook (which the docs themselves warn "Apple may disable... without notice") or proper `sign`/notarization. **This repo doing real notarization via `notarize.macos` is exactly what avoids needing that hack** — a direct synergy between the two new blocks, not a coincidence. **Confidence: HIGH.**

**3. Second `archives:` entry (new — today's single entry is dead `formats: [binary]`).** GoReleaser supports multiple `archives:` blocks distinguished by `id`, each filtered by `ids:` (which *builds* it packages) — confirmed directly from the official archives schema doc, including the exact "split archives by build id" pattern GoReleaser recommends when you need per-format variants of the same builds:

```yaml
archives:
  - id: raw-binary               # existing, unchanged in shape/name_template — D-02
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [binary]
    name_template: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}"
  - id: zip-archive               # new — feeds browser download + homebrew_casks
    ids: [codegraph-linux-amd64, codegraph-linux-arm64, codegraph-darwin-amd64, codegraph-darwin-arm64]
    formats: [zip]
    name_template: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}_archive"
```
**Confidence: HIGH** for the multi-archive mechanism itself (official schema doc, "Splitting Archives by Build" example fetched verbatim); the exact `name_template` values above are illustrative — the *raw-binary* one MUST byte-match today's `internal/upgrade.releaseAssetName()` contract exactly, which is an implementation detail for the plan phase, not something this research can finalize from docs alone.

**4. `checksum:` block wakes up (already present, currently dead per the file's own header comment).** Under `goreleaser build`, it never runs; under `goreleaser release`, it does. It duplicates `release.yml`'s hand-rolled `sha256sum codegraph_* > ..._checksums.txt` step — **that hand-rolled step must be deleted**, not run alongside GoReleaser's, to avoid two divergent checksum files. **Confidence: HIGH** on the duplication risk (both are visible directly in the files this research read); resolving *which one wins* is a Phase-1 implementation decision.

**5. `release:` block — mode against release-please's pre-existing Release object.** Confirmed from the official release customization docs: *"If a release already exists in the target platform before running GoReleaser, the tool will not overwrite existing body text by default"* and GoReleaser *"can automatically replace existing artifacts if an upload fails due to a conflict."* This is exactly the disposition `release.yml`'s current hand-rolled `gh release view` / `gh release upload --clobber` logic already implements by hand (D-04: release-please owns the body, never regenerated). GoReleaser's default behavior needs no `mode: replace` override to preserve that invariant — only `skip_upload: false` (default) so it actually uploads assets. **Confidence: MEDIUM** — docs confirm the default-safe behavior in words, but the exact interaction with a Release object created by a *different* tool (release-please, not a prior GoReleaser run) should be smoke-tested against a real prerelease tag before trusting it in Phase-1 execution.

### What NOT to Add (Q5)

| Avoid | Why | Instead |
|-------|-----|---------|
| `notarize.macos_native` | Pro-only (confirmed: "exclusively available with GoReleaser Pro, since v2.8"), requires a macOS runner + real Keychain, and only adds value for App Bundles/DMG/PKG — this repo ships a bare CLI binary | `notarize.macos` (cross-platform/Quill, OSS, works on any runner) |
| `brews:` (Homebrew Formula block) | Soft-deprecated since v2.10, docs page itself now titled "(deprecated)"; formulas are meant to build-from-source, which is semantically wrong for a pre-built signed binary anyway | `homebrew_casks:` |
| GoReleaser's own `signs:` block (cosign integration) | Would create a **second**, differently-scoped signing flow alongside the existing hand-rolled `cosign sign-blob --bundle` step in `release.yml`'s `assemble` job — two signature sources for the same binary is confusing at best, and `internal/upgrade`'s `defaultVerify` is pinned to the existing per-binary `.sigstore.json` shape/identity (`releaseWorkflowRefPattern`). Do not let GoReleaser sign anything cosign-related. | Keep the existing hand-rolled `cosign sign-blob` step in the `assemble` job, run it on GoReleaser's `release`-produced binaries exactly as it runs on today's `build`-produced ones |
| GoReleaser's `sboms:` block | Duplicates the existing `syft` step already wired into `assemble` | Keep existing `syft` step |
| GoReleaser's built-in SLSA/provenance features (none exist as a first-class block, but don't reach for third-party GoReleaser plugins that claim to) | The existing `slsa-framework/slsa-github-generator` generic-generator job is already correct and independent of the build tool | No change needed here at all |
| `prebuilt` builder / `--split`/`--merge`/`--prepare`/`continue --merge` | All confirmed Pro-only | Single-runner `goreleaser release` (see Headline Answer) |
| homebrew-core | Already decided against by the maintainer (own tap) — external review queue, no schedule control | `seanb4t/homebrew-tap` |
| `gon` / `mitchellh/gon` | Predates GoReleaser's native `notarize.macos` integration (quill absorbed and superseded gon's use case); adding it would be a redundant, unmaintained-adjacent extra dependency doing what `notarize:` now does natively | GoReleaser's built-in `notarize.macos` |
| Windows `.exe`/scoop packaging | Native Windows support was explicitly dropped this project (v0.4.0, #29) | N/A — not in scope |

## Apple Tooling (Q4)

**One-time, human, Apple Developer Portal setup (not CI-automated, done once by the maintainer who already holds the Developer Program membership per the milestone context):**
1. Create/export a **"Developer ID Application"** certificate (the correct type for signing a distributed-outside-App-Store binary; "Developer ID Installer" is for `.pkg` installers, not relevant to a bare binary or zip) as a `.p12` file with its private key, base64-encode it → `MACOS_SIGN_P12` secret.
2. Create an **App Store Connect API key** (Users and Access → Keys → App Store Connect API, "Developer" role is sufficient for notarization) → download the `.p8` file, base64-encode → `MACOS_NOTARY_KEY` secret; note its Key ID → `MACOS_NOTARY_KEY_ID`; note the account's Issuer ID (UUID, shown on the same Keys page) → `MACOS_NOTARY_ISSUER_ID`.
   **Confidence: MEDIUM** — this is standard, widely-documented Apple Developer Portal process (multiple independent how-to sources agree on the steps), not verified live against Apple's actual current UI in this research pass; UI particulars can shift and should be confirmed by the maintainer during Phase-1 execution rather than assumed from docs.

**GitHub Actions secrets required (all net-new to this repo):**

| Secret | Contents | Used by |
|--------|----------|---------|
| `MACOS_SIGN_P12` | base64 of the Developer ID Application `.p12` | `notarize.macos.sign.certificate` |
| `MACOS_SIGN_PASSWORD` | password protecting the `.p12` | `notarize.macos.sign.password` |
| `MACOS_NOTARY_KEY` | base64 of the App Store Connect `.p8` key | `notarize.macos.notarize.key` |
| `MACOS_NOTARY_KEY_ID` | that key's ID | `notarize.macos.notarize.key_id` |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect issuer UUID | `notarize.macos.notarize.issuer_id` |
| (name TBD, e.g. `HOMEBREW_TAP_TOKEN`) | fine-grained PAT scoped to `seanb4t/homebrew-tap`, Contents: Read+Write | `homebrew_casks[].repository.token` |

None of these are macOS-Keychain-dependent — because `notarize.macos` is the Quill cross-platform path, this whole set of secrets works whether the job runs on `ubuntu-latest` or `macos-latest`. **Confidence: HIGH** (directly from GoReleaser's own docs example, which itself runs on `ubuntu-latest`).

**Stapling — the hard constraint, confirmed independently, not just asserted by the milestone context:**
- Apple's `stapler` tool only attaches notarization tickets to `.app` bundles, `.pkg` installers, and `.dmg` disk images — never to a bare executable or a `.zip`. This is an **Apple platform constraint**, not a GoReleaser limitation.
- Quill's command surface (`sign`, `notarize`, `sign-and-notarize`, `submission {list,logs,status}`, `describe`, `extract certificates`, `p12 {attach-chain,describe}` — enumerated from its own README) confirms it never attempts stapling; it only submits to Apple's Notary API and reports status.
- **Practical consequence for this milestone:** even after `notarize.macos` succeeds, the raw binary (and a zip containing it) can only pass a *Gatekeeper online check* (querying Apple's servers for the ticket at first launch) — never an offline staple check. This matches, and independently confirms, the milestone context's own framing: *"stapling requires a container... an offline machine falls back to an online Gatekeeper check that fails."* Getting real offline-capable stapling would require packaging as `.pkg` or `.dmg`, which routes straight back into `notarize.macos_native` — **Pro-only**. **Confidence: HIGH** on the stapling mechanics (Apple platform fact + quill's own documented command surface); this is a hard boundary the roadmap needs to accept, not something more research resolves.

**Hardened runtime / entitlements for a CGo network-I/O CLI:**
- Apple's notary service **requires** hardened runtime + a secure (Developer-ID) timestamp on the signature to accept any submission at all — this is non-negotiable, confirmed by multiple independent sources (Apple's own notarization docs referenced across community how-tos, and GoReleaser's `sign.options`/quill's own signing default). `notarize.macos` handles this automatically as part of `sign-and-notarize` — no manual `--options=runtime` flag needed in this repo's config.
- **Networking is unaffected by hardened runtime** — none of the hardened-runtime restrictions relate to network access; a CLI doing HTTPS calls (this repo's `codegraph upgrade` self-update path) needs no special entitlement for that.
- **JIT/dynamic-code entitlements (`allow-jit`, `allow-unsigned-executable-memory`) are not needed** — those exist for runtimes that generate and execute code at runtime (e.g. V8/Electron). A statically-linked Go binary with a CGo tree-sitter parser does ahead-of-time compilation only; tree-sitter's C scanners are compiled in, not JIT-generated. **No `entitlements:` file is needed for this binary** — leave `sign.entitlements` unset. **Confidence: MEDIUM** — this reasoning is sound and consistent with widely-reported successful notarization of plain Go CLI tools, but no source directly confirms "CGo-with-tree-sitter specifically notarizes clean with zero entitlements" — flagged as a real risk to smoke-test in Phase 1 (build, `quill sign-and-notarize` locally or in CI, then `codesign -dvv` + `spctl -a -vv -t exec` on the result) before assuming it.
- **A separate, older risk class exists and is worth naming explicitly**: historical Go toolchain issues (`golang/go#30488`, `golang/go#34986`, and multiple "signature of the binary is invalid" notarization failures reported for Go binaries) were tied to Go's linker producing Mach-O structures the notary service's signature validator rejected. These reports mostly predate current Go toolchain versions and mostly concern `-buildmode=c-shared` (a different build mode than this repo's plain executable build) or missing the full Apple certificate chain (which quill's `p12 attach-chain` / embedded Apple-cert-chain handling addresses directly). **This repo's darwin binaries already accept `codesign` today** (measured fact from the milestone context: `adhoc, linker-signed` on arm64) — a real Developer-ID re-sign is a strictly smaller ask than getting *any* signature to validate at all, which already works. **Confidence: LOW-MEDIUM** on "this will notarize clean on the first try" — genuinely worth a Phase-1 spike rather than an assumption, given this is exactly the kind of "measure, don't recall" trap this repo's standing rule exists for.

## Integration Points

| File | Change |
|------|--------|
| `.goreleaser.yaml` | Add `notarize:` (macos, cross-platform variant) block; add `homebrew_casks:` block; add second `archives:` entry (zip format) alongside the existing raw-binary one; the existing `checksum:` block needs no change but will now actually execute — remove the header comments documenting it as dead, since under `release` it is live |
| `.github/workflows/release.yml` | Collapse the 2-job matrix (`build` on 2 runner classes) + `assemble` + `provenance` shape so the **build+archive+notarize+homebrew_casks+GH-release-upload** work happens in **one job on one macOS runner**, running `goreleaser release --clean` (not `--split`, not `--single-target`) with `CC`/`CXX` env set for **both** linux legs via zig (in addition to today's linux/arm64-only zig use) and no CC override for the two native darwin legs. The existing `assemble` job's hand-rolled `sha256sum` step must be deleted (GoReleaser's `checksum:` now produces it). The existing `cosign sign-blob`/`syft` steps stay, but move to run **after** `goreleaser release` produces artifacts (either as a later step in the same job over `dist/`, or a downstream job consuming `dist/` — GoReleaser's `release` writes `dist/artifacts.json` exactly like `build` does today, so the existing "locate binary via artifacts.json with a `find` fallback" logic in "Rename to release-asset contract name" likely needs re-pointing at the new dist layout, not rewriting from scratch). The `provenance` job (SLSA generic generator) is unaffected in shape — it still consumes a base64 checksums blob, now sourced from GoReleaser's own `checksum:` output file instead of the hand-rolled one. New secrets (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`, tap PAT) must be added to the job's `env:`. |
| `Taskfile.yml` | If there's a local `task release:check`/equivalent dry-run target that currently calls `goreleaser build --single-target`, it should gain (or get replaced by) a `goreleaser release --skip=publish` (or `--snapshot`) dry-run target so contributors can validate the new blocks locally without secrets — `notarize.macos.enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` already makes notarization a no-op (falls back to quill's ad-hoc-signing-only snapshot behavior) when secrets aren't present, which is exactly the pattern quill's own README recommends for this. |
| `internal/upgrade` | No code change implied by this research alone — but the roadmap should explicitly verify the raw-binary `archives:` entry's `name_template` still produces byte-identical filenames to `internal/upgrade.releaseAssetName()`'s contract, since that logic is what's at risk of drifting when the pipeline moves off the current hand-rolled "Rename to release-asset contract name" shell step. |

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|--------------------------|
| `notarize.macos` (cross-platform/Quill, OSS) | `notarize.macos_native` (Pro) | If this project ever ships a `.app`/`.pkg`/`.dmg` (e.g. a future GUI), native notarization + real stapling becomes necessary and is worth the Pro license at that point — not for a bare CLI binary today |
| Single macOS runner running the whole `goreleaser release` with zig-cross for both linux legs | GoReleaser Pro `--split`/`--merge` (native multi-runner) | If GoReleaser Pro is ever purchased for other reasons (e.g. Windows native signing, nightly builds, faster parallel builds), revisit — it's the more "designed for this" mechanism and would let linux legs stay on cheaper/faster linux runners instead of consolidating everything onto macOS |
| `homebrew_casks:` | `brews:` (formula) | Never, for this project — formula is meant for build-from-source and is the deprecated path; no scenario in this project favors it |
| Own tap (`seanb4t/homebrew-tap`) | homebrew-core | Already decided against (PROJECT.md Key Decisions) — revisit only once adoption independently justifies the review-queue cost |
| GoReleaser's embedded quill (via `notarize.macos`) | Standalone `anchore/quill` CLI invoked as a build hook (the pattern quill's own README shows, and what predates the native GoReleaser integration) | Only if `notarize.macos`'s config surface turns out to be missing something quill's raw CLI exposes — unlikely given `notarize.macos` is a thin wrapper over the same library, but worth knowing this exists as an escape hatch |

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `goreleaser@v2.17.1` | `notarize.macos` (cross-platform) | Feature predates v2.1; fully supported at pinned version — no upgrade needed |
| `goreleaser@v2.17.1` | `homebrew_casks` | Introduced v2.10; fully supported at pinned version — no upgrade needed. `pull_request.token` (separate PR-only token) ships in **v2.18, not yet released** as of this research — not usable yet; not needed for a direct-commit-to-`main` tap workflow anyway |
| `zig 0.15.1` (existing `mlugg/setup-zig@v2.2.1` pin) | cross-compiling CGo to linux/amd64 **and** linux/arm64 from a macOS host | No version-specific incompatibility found; this is the same zig version already validated (per Phase-8/Phase-10 research) for linux/arm64 cross from a Linux host — extending its target set doesn't change its version requirements |
| `goreleaser-action@v7.2.3` (existing pin) | `distribution: goreleaser` (not `goreleaser-pro`) | Unchanged — the OSS distribution string is exactly what the notarize/homebrew_casks docs' own working examples use |

## Sources

- `/websites/goreleaser` (Context7, HIGH confidence — official docs mirror, cross-checked against live goreleaser.com fetches) — archives schema, sign pipe, GitHub Actions notarize example, split/merge/continue/publish --merge commands
- https://goreleaser.com/customization/partial/ (web fetch, HIGH confidence, primary source) — split/merge Pro-only statement, verbatim
- https://goreleaser.com/pro/ , Carlos Becker's "GoReleaser Split and Merge" post (web search, MEDIUM-HIGH, corroborating) — Pro-only status cross-check
- https://goreleaser.com/customization/builds/builders/prebuilt/ (web fetch, HIGH, primary source) — prebuilt builder Pro-only statement, verbatim
- https://goreleaser.com/customization/sign/notarize/ (web fetch x3 with different targeted prompts, HIGH, primary source) — notarize.macos vs macos_native config keys, secrets shape, OS requirements, stapling absence
- https://goreleaser.com/customization/notarize (Context7 mirror, HIGH) — full YAML example, GitHub Actions example showing `runs-on: ubuntu-latest` + `distribution: goreleaser` for the cross-platform path
- https://github.com/anchore/quill README + llms.txt (fetched directly via raw.githubusercontent.com, HIGH, primary source) — command surface confirming no `staple` command; sign/notarize env var names; snapshot ad-hoc-signing pattern
- https://anchore.com/blog/meet-quill-a-cross-platform-code-signing-tool-for-macos/ (web search, MEDIUM, corroborating) — rationale for quill's pure-Go cross-platform design vs. gon's shell-out-to-codesign approach
- https://goreleaser.com/customization/publish/homebrew_casks/ (web fetch, HIGH, primary source) — full key list, OSS-vs-Pro sub-feature split, since-v2.10 note, `token_type`/`pull_request.token` version gating
- https://goreleaser.com/deprecations/ + community migration issue threads (web search, MEDIUM) — brews→homebrew_casks deprecation timeline
- https://goreleaser.com/customization/package/archives (Context7 mirror, HIGH, primary source) — multiple archives blocks by id, "Splitting Archives by Build" pattern, format list including `binary`
- https://goreleaser.com/customization/release (Context7 mirror, HIGH, primary source) — release mode defaults against a pre-existing Release object, disable/skip_upload keys
- `gh api repos/goreleaser/goreleaser/releases/latest` (executed locally, HIGH — direct GitHub API call, 2026-08-07) — confirms v2.17.1 is current
- https://github.com/goreleaser/example-zig-cgo (web search result, MEDIUM — official GoReleaser org example repo, not independently cloned/built in this research pass) — zig cross-compiling CGo from any host to any target
- Multiple community sources on homebrew tap PAT scoping (DNSControl docs, mcginniscommawill.com, dev.to how-tos) (web search, MEDIUM, cross-corroborating but non-primary) — GITHUB_TOKEN repo-scoping limitation, PAT requirement
- Apple hardened runtime / entitlements community sources (developer.apple.com forum threads, multiple Go-notarization blog posts) (web search, LOW-MEDIUM, non-primary, cross-corroborating on the "no special entitlements for plain Go CLI" conclusion but not CGo-tree-sitter-specific) — flagged explicitly as needing a Phase-1 empirical check, not treated as settled fact

---
*Stack research for: macOS Gatekeeper notarization + Homebrew tap distribution (v0.5.0 milestone)*
*Researched: 2026-08-07*
