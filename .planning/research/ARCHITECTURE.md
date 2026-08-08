# Architecture Research: macOS Notarization + Homebrew Integration into the Existing Release Pipeline

**Domain:** CI/CD release engineering — Go binary supply chain (GoReleaser, cosign, SLSA, syft) extended with Apple notarization and Homebrew tap publishing
**Researched:** 2026-08-07
**Confidence:** MEDIUM overall (repo facts HIGH; GoReleaser OSS/Pro feature boundaries MEDIUM — official docs, fetched via summarizing tool, not independently cross-checked against a second source; no claim here has been machine-verified against a real `goreleaser release` run in this repo)

**Standing rule applied:** every claim below is either (a) read directly from this repo's files (HIGH, file:line cited) or (b) sourced from GoReleaser's own current docs (MEDIUM — single official source per claim). Nothing here is inferred from the roadmap or from memory of older GoReleaser versions. Two claims materially contradict or sharpen what `PROJECT.md`'s Key Decisions table already asserts — both are flagged explicitly rather than silently reconciled, per this repo's own "report the gap, don't paper over it" convention.

---

## 1. Two findings that change the plan before any job graph is drawn

These surfaced during research and are load-bearing enough that the roadmap should see them before phase decomposition, not buried in prose below.

### Finding A — `brews:` is deprecated; `homebrew_casks:` is GoReleaser's current CLI-formula mechanism

PROJECT.md's Key Decisions table says: *"formula published by GoReleaser's `brews:` block"* (`.planning/PROJECT.md:54`, `:178`). GoReleaser's own docs (as of this research date) mark `brews:`/`homebrew_formulas:` **deprecated since v2.10**, explicitly stating *"Homebrew Casks should be used instead."* `homebrew_casks:` is OSS, supports a `binaries:` field for CLI executables (not just `.app` GUI bundles), and is the actively maintained path. `brews:` still functions today (deprecated ≠ removed) but is documented as legacy.

**Confidence:** MEDIUM (GoReleaser docs, single source, fetched via summarizing WebFetch — not raw-read). **Recommendation:** this milestone should decide `brews:` vs `homebrew_casks:` as an explicit Phase-0 research/decision item, not silently follow the PROJECT.md wording. Using a documented-deprecated block for a new integration being built today is the wrong default; using `homebrew_casks:` for a CLI tool is documented but less battle-tested for that use case than the (deprecated) formula path, since casks were originally a GUI-app mechanism. **This needs a short, dedicated spike before Phase 3 (tap integration) is planned in detail** — do not let it be discovered mid-implementation.

### Finding B — true offline-staplable notarization requires either GoReleaser Pro (`dmg:`) or a hand-built `.app` wrapper; the OSS `notarize.macos` (Quill) path notarizes but cannot staple a bare binary or a zip

- GoReleaser OSS ships a cross-platform notarization pipe (`notarize.macos`, Anchore Quill-based) that operates on **raw binaries directly**, before archiving. It works; it is not a stub.
- Apple's stapling mechanism (`stapler staple`) only attaches to a bundle container — `.app`, `.pkg`, `.dmg`. It **cannot** staple a bare Mach-O executable or a `.zip`.
- GoReleaser's `dmg:` block — the one config surface that produces a staplable container with native notarization wired in — is **Pro-only**.
- Consequence: an OSS-only pipeline that notarizes a raw binary and ships it in a `formats:[zip]` archive produces a **notarized-but-unstapled** artifact. Gatekeeper's behavior for that shape is: read `com.apple.quarantine` → attempt an **online** ticket lookup against Apple's servers → cache the result → allow. This satisfies the milestone's literal stated bar (`spctl -a -vv -t exec` → `accepted`, PROJECT.md:52, `:63`) whenever the verifying machine has network — which is the overwhelmingly common case for a first launch. It does **not** satisfy "verifiable fully offline," which the milestone's own key-decision language ("Stapling forces the asset decision," PROJECT.md:61) treats as the real design call.

**Confidence:** MEDIUM (GoReleaser docs + Apple's documented stapling constraints, cross-referenced across 3 official-doc fetches this session; internally consistent, not independently re-verified against Apple's current developer docs directly). **This is not framed as a correctness bug** (nothing here signs a different byte stream than ships — see §3) — it is a **scope decision the roadmap must make explicitly**: (1) accept online-notarization-only via zip (OSS, cheapest, matches the milestone's literal `spctl` bar), (2) build a minimal `.app`-bundle wrapper around the CLI binary purely so it can be stapled, then zip the stapled `.app` (OSS-achievable, more engineering), or (3) pay for GoReleaser Pro's native `dmg:`/`notarize.macos_native` path (least engineering, ongoing cost, and reintroduces the Pro-license question already open for split/merge — see §2). Recommend **(1) for v0.5.0**, with (2)/(3) explicitly deferred — it clears the stated bar, costs nothing new, and matches this milestone's existing "Gatekeeper impact is genuinely narrow" framing (PROJECT.md:60).

---

## 2. The `goreleaser build` → `goreleaser release` restructure

### 2.1 Before: today's job graph (as read from `.github/workflows/release.yml`)

```
tag push (v[0-9]*, release-please-only)
        │
        ▼
┌───────────────────────────────────────────────────────────────┐
│ build (matrix: 4 legs, 2 runner classes)                        │
│  linux/amd64  @ namespace-profile-linux-amd64-4x8 (native)      │
│  linux/arm64  @ namespace-profile-linux-amd64-4x8 (zig cc)      │
│  darwin/amd64 @ namespace-profile-macos-6x14-tahoe (native)     │
│  darwin/arm64 @ namespace-profile-macos-6x14-tahoe (native)     │
│  each: `goreleaser build --single-target --clean`               │
│        → rename to codegraph_<tag>_<goos>_<goarch>               │
│        → upload-artifact                                        │
└───────────────────────────────┬───────────────────────────────┘
                                 │ needs: build
                                 ▼
┌───────────────────────────────────────────────────────────────┐
│ assemble (namespace-profile-linux-amd64-2x4)                    │
│  download all 4 binaries                                        │
│  sha256sum → codegraph_<tag>_checksums.txt   (HAND-ROLLED)      │
│  cosign sign-blob --bundle per binary        (HAND-ROLLED)      │
│  syft SBOM per binary                        (HAND-ROLLED)      │
│  base64-encode checksums.txt → hashes output (HAND-ROLLED)      │
│  gh release upload/create --clobber                             │
└───────────────────────────────┬───────────────────────────────┘
                                 │ needs: assemble
                                 ▼
┌───────────────────────────────────────────────────────────────┐
│ provenance (reusable workflow, its own runner)                  │
│  slsa-framework/slsa-github-generator generator_generic_slsa3   │
│  base64-subjects: assemble.outputs.hashes (parses checksums.txt)│
└───────────────────────────────────────────────────────────────┘
```

`archives:` and `checksum:` in `.goreleaser.yaml` are dead — `goreleaser build` never runs them (`.goreleaser.yaml:1-40`, `:108-124`). `release.yml`'s own shell steps are the entire asset-naming/checksum/signing contract today.

### 2.2 After: recommended job graph under `goreleaser release`

```
tag push (v[0-9]*, release-please-only) — UNCHANGED trigger, UNCHANGED file identity
        │
        ▼
┌───────────────────────────────────────────────────────────────┐
│ release (SINGLE job, SINGLE runner: namespace-profile-macos-*)  │
│                                                                   │
│  1. checkout (fetch-depth: 0) — unchanged                       │
│  2. setup-go, cache — unchanged                                 │
│  3. install zig (now used for BOTH linux legs, not just arm64)  │
│  4. install goreleaser (pinned, same version)                   │
│  5. install cosign, syft — still needed on this runner if       │
│     signs:/sboms: shell out to their CLIs (GoReleaser's         │
│     signs:/sboms: pipes call these as external commands)        │
│  6. `goreleaser release --clean`  ← ONE invocation does:         │
│       builds:      all 4 targets, natively-darwin +             │
│                     zig-cross-linux, on this ONE macOS host      │
│       notarize:     darwin binaries only (macos Quill block)    │
│       archives:     raw-binary passthrough (upgrade contract)   │
│                      AND zip archives (brew/browser)             │
│       checksum:     ONE checksums.txt over ALL assets            │
│                      (binaries + archives) — GoReleaser-owned,   │
│                      hand-rolled step DELETED                    │
│       binary_signs: cosign keyless over the raw binaries only    │
│                      (matches internal/upgrade's per-binary      │
│                      .sigstore.json contract exactly)            │
│       signs:        (optional) cosign over the zip archives too  │
│       sboms:        syft SBOM per binary (and optionally per     │
│                      archive)                                    │
│       homebrew_casks (or brews, see Finding A): push formula     │
│                      to seanb4t/homebrew-tap                     │
│       release:      gh-release publish, mode: keep-existing      │
│                      (preserves D-04: release-please owns the    │
│                      Release body/prerelease flag)                │
│  7. emit checksums.txt (or its per-asset hashes) as a job         │
│     output for the provenance job — same mechanism as today's    │
│     `hash` step, just reading GoReleaser's checksums.txt          │
│     instead of the hand-rolled one                                │
└───────────────────────────────┬───────────────────────────────┘
                                 │ needs: release
                                 ▼
┌───────────────────────────────────────────────────────────────┐
│ provenance (unchanged shape — reusable workflow, its own runner)│
│  base64-subjects now covers BOTH binaries AND archives           │
│  (expanded scope — see §3.4)                                     │
└───────────────────────────────────────────────────────────────┘
```

**What moves:**
- `build` + `assemble`'s asset-producing steps collapse into ONE job's ONE `goreleaser release` call.
- The hand-rolled checksum/cosign/syft shell loops in `assemble` (`.github/workflows/release.yml:238-278`) are **deleted**, replaced by native GoReleaser pipes (`checksum:`, `binary_signs:`/`signs:`, `sboms:`).
- The 2-runner-class matrix (`namespace-profile-linux-amd64-4x8` for linux, `namespace-profile-macos-6x14-tahoe` for darwin) collapses to **one runner class** (`namespace-profile-macos-6x14-tahoe`, or equivalent) for the whole release. The Linux Namespace runner is no longer used by `release.yml` at all (still fine elsewhere, e.g. `ci.yml`).
- `zig` installation moves from "linux/arm64-only, conditional" (`release.yml:131-135`, `matrix.needs_zig`) to unconditional, now targeting linux/amd64 too (previously native-only).

**What disappears:**
- The `build` job's matrix entirely (4 parallel legs → 1 sequential/composite build inside one `goreleaser release` run).
- The `assemble` job as a separate `needs: build` stage — folded into `release`.
- The "Rename to release-asset contract name" hand-rolled shell step (`release.yml:150-187`) — GoReleaser's `archives:` `name_template` now IS the naming contract, machine-enforced, not shell-script-enforced.
- The dead-code comments in `.goreleaser.yaml` marking `archives:`/`checksum:` as inert (`.goreleaser.yaml:108-116`) — they go live and those comments must be rewritten, not left stale.

**What MUST be preserved (hard constraints, not suggestions):**
- `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` (`internal/upgrade/verify.go:41-45`) anchors the cosign OIDC cert SAN to `.github/workflows/release.yml@refs/tags/v[0-9]*` — the workflow file's **name and trigger** must not change, and the job producing the cosign signature must still run under this exact workflow+ref. Collapsing jobs is fine; renaming the file or changing the tag trigger pattern is not.
- `internal/upgrade.releaseAssetName()` (`internal/upgrade/upgrade.go:209-211`): `codegraph_<tag>_<goos>_<goarch>`, **no extension**, `<tag>` is the v-prefixed `github.ref_name`, not GoReleaser's stripped `.Version`. The raw-binary `archives:` entry's `name_template` must emit exactly this — `.goreleaser.yaml`'s currently-dead template (`.goreleaser.yaml:119-120`, `{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}`) already matches this shape and can be reused verbatim once live; only a second `archives:` entry (zip, distinct `name_template`) needs to be added.
- `.sigstore.json` sidecar naming (`internal/upgrade/upgrade.go:195`, `assetName+".sigstore.json"`) — GoReleaser's `binary_signs:` default `signature:` template must be set to reproduce `<binary-asset-name>.sigstore.json` exactly, not GoReleaser's own default signature name template.
- Per-binary (not per-checksums-file) cosign signing semantics — `internal/upgrade`'s `defaultVerify` hashes the **downloaded binary itself** (`internal/upgrade/upgrade.go:161-177`, `sha256.Sum256(binary)`). `binary_signs:` (not `signs: artifacts: checksum`) is the correct GoReleaser pipe for this — confirmed via research (§ below) as the pipe purpose-built for "sign the binaries themselves, not the archive."
- `release: mode: keep-existing` (or equivalent existing-release preservation) so release-please's Release body/prerelease flag is never overwritten (D-04, `release.yml:289-296` today hand-rolls this distinction).

### 2.3 Single-runner constraint — candidate architectures

| # | Architecture | Verdict | Rationale |
|---|---|---|---|
| **A** | **All-on-macOS-runner; zig cc cross-compiles both linux legs from the macOS host; darwin builds native.** | **RECOMMENDED** | OSS-compatible — `goreleaser release` is one invocation, one runner, exactly what GoReleaser's own build model requires without Pro. Zig is host-agnostic (bundles its own libc/headers), so `zig cc -target x86_64-linux-gnu` / `aarch64-linux-gnu` from a macOS host is the same class of operation `.goreleaser.yaml` already does today for linux/arm64 FROM a Linux host (`.goreleaser.yaml:65-77`) — just changing the host, not inventing a new mechanism. This is also the resolution PROJECT.md already names as "likely" (PROJECT.md:58). **New risk, front-load it**: linux/amd64 has never been zig-cross-built before (today it's native on a Linux runner, `.goreleaser.yaml:46-63`) — this is a genuinely new, unproven path for THIS project's CGo (tree-sitter) dependency, and must be validated before the rest of the phase is planned, not discovered mid-build. |
| **B** | **Matrix-build (as today) then a separate `goreleaser release --skip=build` job consuming pre-built artifacts.** | REJECTED | Requires hand-assembling GoReleaser's internal `dist/artifacts.json` + per-target directory layout across runners without GoReleaser's own merge logic. This repo's own `release.yml` comment already flags that `dist/` layout is "NOT guaranteed stable across GoReleaser versions" (`release.yml:166-176`) — building a hand-rolled recombination on top of an explicitly-unstable internal format is exactly the kind of undocumented-format dependency this repo's engineering culture rejects elsewhere (see the `dist/artifacts.json`-with-`find`-fallback pattern already defending against this). This is, in effect, reimplementing GoReleaser Pro's split/merge without its guarantees. |
| **C** | **GoReleaser Pro `--split`/`--merge`.** | DEFERRED, not rejected | Purpose-built for exactly this (multi-runner CGo cross-arch, confirmed Pro-exclusive — "introduced in v1.12.0-pro," MEDIUM confidence, official GoReleaser blog/docs). Real, correct, costs a paid license. Only revisit if Option A's zig-cross-linux-from-macOS proves broken or unacceptably slow in the Phase-0 spike. |

**SLSA generic generator subject list under Option A:** unaffected in mechanism — the `provenance` job still consumes a base64-encoded checksums file and lets the generic generator parse `<sha256>  <name>` lines as subjects (`release.yml:196-214`, `:280-287`). What changes is **scope**: today's checksums.txt covers 4 binaries only; under the new `archives:` config it will cover binaries **and** zip archives in one file (unless deliberately filtered via `checksum:`'s own `ids`/artifact-type options). **Recommend widening SLSA coverage to include the archives** — browser/brew users download the zip, not the raw binary, and "what gets signed and attested is what users actually download" (this milestone's own stated correctness bar) argues for covering both artifact classes, not just the one `internal/upgrade` cares about.

**`checksum:` duplication:** resolve by **deletion, not reconciliation** — once `goreleaser release` runs `checksum:` live with the exact `name_template` this repo's `.goreleaser.yaml` already documents (`.goreleaser.yaml:122-124`, matching `release.yml`'s hand-rolled `codegraph_${TAG}_checksums.txt`), the hand-rolled `sha256sum` step (`release.yml:238-244`) becomes a strictly redundant second writer of the same filename and must be removed, not kept as a cross-check. Two writers of one filename in one job is a race/last-writer-wins hazard, not a safety net.

**cosign — GoReleaser's `signs:`/`binary_signs:` pipes vs explicit workflow steps:**
Move into GoReleaser's native pipes. Researched shape (MEDIUM confidence, GoReleaser docs):
```yaml
binary_signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]
    output: true
```
`binary_signs:` (not `signs: artifacts: binary`) is the correct pipe specifically because this project now has **non-binary archives coexisting** with raw binaries — GoReleaser's docs state `artifacts: binary` under `signs:` only applies "when `archives.format` is 'binary'" for *all* archives; `binary_signs:` exists precisely for "sign the built binaries themselves, when your archives are NOT `format: binary`" (v2.2+). This is exactly this project's post-milestone shape (raw binary passthrough archives + real zip archives, side by side).

Identity/OIDC properties are preserved either way — GoReleaser's `signs:`/`binary_signs:` still shells out to the real `cosign` CLI running inside the same GitHub Actions job, under the same `id-token: write` permission and the same GitHub OIDC issuer, producing an identical certificate SAN shape (still anchored to `release.yml@refs/tags/v*`). The verified identity does not change; only which process (a shell step vs. a GoReleaser-invoked subprocess) issues the `cosign sign-blob` call.

---

## 3. Where notarization sits in the artifact-flow graph

### 3.1 Full trace: build → notarize → archive → SBOM → cosign → SLSA → release assets → consumers

```
[builds:]  4 raw Mach-O/ELF binaries (darwin×2 native, linux×2 zig-cross)
     │
     ├─ linux/amd64, linux/arm64 ─────────────────────────────┐
     │                                                          │
     └─ darwin/amd64, darwin/arm64                              │
           │                                                    │
           ▼                                                    │
     [notarize.macos:]  darwin binaries ONLY (ids: filter)      │
       codesign (Developer ID Application cert)                 │
       submit to notarytool (App Store Connect API key)         │
       wait for Apple's response                                │
       — MODIFIES THE BYTES on darwin binaries (embeds a         │
         code signature into the Mach-O) —                       │
           │                                                    │
           ▼                                                    │
     darwin binaries are now the FINAL, signed-by-Apple bytes    │
     (still un-stapled — see Finding B, §1)                      │
           │                                                    │
           └────────────────┬───────────────────────────────────┘
                             ▼
                   [archives:]  TWO entries per build id:
                     (a) formats:[binary]  → raw passthrough,
                         name: codegraph_<tag>_<os>_<arch>
                         (upgrade contract, D-02)
                     (b) formats:[zip]     → compressed container,
                         name: codegraph_<tag>_<os>_<arch>.zip
                         (browser + brew contract)
                             │
                             ▼
                   [checksum:]  ONE checksums.txt over ALL of
                     the above (binaries + zips)
                             │
                             ▼
              ┌──────────────┴──────────────┐
              ▼                              ▼
     [binary_signs:]                   [sboms:]
       cosign sign-blob over             syft SBOM per binary
       the RAW BINARIES only             (and optionally per zip)
       → codegraph_<tag>_<os>_<arch>
         .sigstore.json
              │                              │
              └──────────────┬───────────────┘
                              ▼
                   [homebrew_casks:] (or brews:, Finding A)
                     formula references the ZIP archive's
                     GitHub release download URL + its sha256
                     (from checksum:'s output)
                              │
                              ▼
                   [release:]  gh release publish,
                     mode: keep-existing (preserves D-04)
                     uploads: binaries, zips, .sigstore.json,
                              .spdx.json, checksums.txt
                              │
                              ▼
                   (separate job) [provenance:]
                     SLSA3 over checksums.txt subjects
                     (binaries + zips, widened scope — §2.3)
```

### 3.2 Ordering correctness — no bug found in the GoReleaser-native pipeline, IF ordering is preserved

Notarization happens on **raw binaries**, and GoReleaser's own pipe order runs `notarize` before `archive`/`checksum`/`sign`/`sbom` (confirmed via GoReleaser's notarize docs: "operates on raw Go binaries directly, before archiving" — MEDIUM confidence, official docs). This is the **correct** order: the bytes that get archived, checksummed, cosign-signed, SBOM'd, and SLSA-attested are the **post-notarization** bytes — i.e., exactly what a user downloads and what `spctl` will assess. There is no window in the native-pipe design where a pre-notarization binary is signed/attested while a post-notarization one ships.

**This ordering constraint is worth stating as an explicit acceptance check, not just trusted from docs** (per this repo's "measure the binary, don't infer" rule): after Phase 2 lands, a verification step should diff the sha256 of the notarized darwin binary as it exists in `dist/` immediately after the `notarize` pipe runs against the sha256 actually recorded in the released checksums.txt / cosign bundle / SBOM subject. If GoReleaser's pipe ordering is ever changed by a config mistake (e.g., a custom `before:`/`after:` hook, or accidentally scoping `notarize.ids` to run post-archive against an archive artifact type it doesn't support), this is exactly the class of "signed a pre-X, shipped a post-X" correctness bug the quality gate calls out. **No evidence such a bug exists today** — this is a forward-looking acceptance gate to add, not a defect found in research.

### 3.3 What must happen before/after, concretely

| Stage | Must run before | Must run after | Why |
|---|---|---|---|
| `builds:` | everything | — | source of all bytes |
| `notarize.macos:` (darwin only) | `archives:`, `checksum:`, `binary_signs:`, `sboms:` | `builds:` | modifies darwin binary bytes; everything downstream must attest to the modified bytes |
| `archives:` (both entries) | `checksum:`, `binary_signs:`, `sboms:`, `homebrew_casks:`, `release:` | `notarize.macos:` (for darwin ids), `builds:` (for linux ids — no notarize step applies) | archives must contain final bytes |
| `checksum:` | `release:`, `provenance` job | `archives:` | must hash final archived+raw assets, not pre-archive intermediates |
| `binary_signs:` | `release:` | `archives:` (raw-binary entry) | must sign the exact bytes `internal/upgrade` will later hash |
| `sboms:` | `release:` | `archives:`/`builds:` | SBOM should describe the shipped artifact, not an intermediate |
| `homebrew_casks:` | — (terminal within `release`) | `archives:` (zip entry), `checksum:` | formula's sha256 must match the actual uploaded zip |
| `release:` (publish) | — | all of the above | uploads everything in one shot |
| `provenance` (separate job) | — | `release` job (needs:) | subjects must be the exact bytes now sitting in the GitHub Release |

---

## 4. Dual-asset topology

### 4.1 Asset inventory and consumers

| Asset | Producer | Naming | Consumer | Breaks if renamed |
|---|---|---|---|---|
| `codegraph_<tag>_<os>_<arch>` (raw binary, no extension) | `archives:` entry (a), `formats:[binary]` | Must equal `internal/upgrade.releaseAssetName()` exactly (`internal/upgrade/upgrade.go:209-211`) | `codegraph upgrade` (`downloadReleaseAsset`, `internal/upgrade/upgrade.go:219-220`) | **Yes — silently.** `codegraph upgrade` builds this URL by string template; a name-shape drift produces a 404, not a build failure. `TestReleaseAssetNameMatchesGoReleaser` (`internal/upgrade/verify_release_e2e_test.go`, referenced at `.goreleaser.yaml:1-21`) is the only machine guard — it must be re-run/re-verified once `archives:` goes live, not just trusted from the comment. |
| `codegraph_<tag>_<os>_<arch>.sigstore.json` | `binary_signs:` | `<raw-binary-name>+".sigstore.json"` | `codegraph upgrade`'s `defaultVerify` (`internal/upgrade/upgrade.go:195`) | Yes, same failure mode — a missing/misnamed sidecar makes every upgrade fail signature verification. |
| `codegraph_<tag>_<os>_<arch>.zip` (or GoReleaser's default archive template) | `archives:` entry (b), `formats:[zip]` | Distinct from the raw-binary name — must not collide | Browser downloaders (GitHub Releases page), `homebrew_casks:` formula's `url` | Homebrew formula pins a URL+sha256 at publish time; renaming the template breaks the **next** `brew install`/`brew upgrade` for anyone on an older formula version pointing at the old name, though GoReleaser regenerates the formula each release, so this is a forward-compat risk (new releases self-heal), not a permanent break. |
| `codegraph_<tag>_checksums.txt` | `checksum:` | `.goreleaser.yaml`'s existing (currently dead) `name_template` (`.goreleaser.yaml:122-124`) | `provenance` job (base64-subjects source) | Yes — the provenance job's `hash` step reads this exact filename. |
| `*.spdx.json` | `sboms:` | Per-artifact, syft default or explicit template | Downstream SBOM consumers (not currently machine-verified by any test in this repo) | No known automated consumer today — lowest risk of the set. |
| `<formula>.rb` in `seanb4t/homebrew-tap` | `homebrew_casks:`/`brews:` | Formula/cask class name, conventionally derived from `name:` | `brew install`/`brew upgrade` end users | Breaks discoverability (`brew search`) if renamed; does not break already-installed users' `brew upgrade` since brew tracks by formula file path in the tap, not by release asset name. |

### 4.2 What `internal/upgrade`'s asset-name resolution assumes today, and the collision risk

`releaseAssetName()` (`internal/upgrade/upgrade.go:202-211`) assumes:
1. Exactly one asset per `(tag, goos, goarch)` triple matches the pattern `codegraph_<tag>_<goos>_<goarch>` with **no extension** and **no other qualifier**.
2. That asset is a raw, directly-executable binary — `downloadReleaseAsset` (`internal/upgrade/upgrade.go:219-236`) does no extraction, decompression, or archive handling; the downloaded bytes are hashed and atomically swapped in as-is (`internal/upgrade/swap.go:40-84`).

**Collision risk from adding archives:** low, *if* the zip archive's `name_template` includes an extension (GoReleaser's default archive template already does — `.zip`/`.tar.gz` — and the existing dead `archives:` block already documents the extension-free raw-binary shape as deliberate, `.goreleaser.yaml:117-121`). The real risk is **not** name collision but **config drift**: if a future edit changes the raw-binary `archives:` entry's `formats:` away from `[binary]` (e.g., someone "simplifies" to one shared archive block for both use cases), GoReleaser would silently start producing a `.tar.gz`/`.zip` under the name `codegraph_<tag>_<os>_<arch>` with a `.tar.gz` suffix change or, worse, the SAME literal filename now containing compressed bytes — `codegraph upgrade` would download it, `sha256.Sum256` a byte stream that IS what was signed (cosign would still verify, since it signs whatever bytes are named), but `atomicSwap` would then install a **non-executable archive** as the binary, since it never unpacks. This would pass signature verification and still brick the install. **Recommend**: a machine test analogous to `TestReleaseAssetNameMatchesGoReleaser` should also assert `archives:`'s raw-binary entry has `formats: [binary]` specifically (not just that the name matches), so a format regression fails CI rather than shipping.

---

## 5. The tap repository (`seanb4t/homebrew-tap`) as a new component

**What it structurally is:** a second, independent public GitHub repository (not a directory in `codegraph-go`), containing formula/cask `.rb` files under a `Formula/` (or cask-equivalent) directory — the Homebrew convention for a third-party "tap." It has its own git history, its own default branch, and no CI of its own required (GoReleaser writes directly via git operations).

**How GoReleaser pushes to it:** `homebrew_casks:`/`brews:` clones the tap repo, writes/updates the generated formula file, and commits+pushes directly to its default branch (or opens a PR, if `pull_request:` config is set — an OSS-available option per the docs surfaced this session, though `check_boxes`-style PR automation is Pro-only). No GitHub App or webhook is required — it's a git push authenticated via a token.

**Credential and least-privilege shape:** GoReleaser's docs explicitly warn the default `GITHUB_TOKEN` from the `release.yml` job **cannot** be used — it's scoped to the triggering repo only (`seanb4t/codegraph-go`), not `seanb4t/homebrew-tap`. A **separate PAT** (classic, or a fine-grained token scoped to just `seanb4t/homebrew-tap` with Contents: Read and Write) is required, stored as a repo secret on `codegraph-go` (e.g. `HOMEBREW_TAP_TOKEN`) and referenced in `.goreleaser.yaml`'s `repository.token` via `{{ .Env.HOMEBREW_TAP_TOKEN }}`. **Least-privilege shape:** a fine-grained PAT scoped to exactly `seanb4t/homebrew-tap` with Contents read/write only (no Actions, no other repos, no org-wide scope) is strictly narrower than a classic PAT with `repo` scope across the whole account — recommend fine-grained if GitHub's fine-grained PAT support is mature enough for this use case by implementation time (worth a quick live check, not assumed).

**Failure mode if the tap push fails mid-release:** under `goreleaser release`, publish steps run as part of the single pipeline invocation; a `homebrew_casks:`/`brews:` push failure would fail that pipe but — per GoReleaser's general pipeline design — does not automatically roll back already-published GitHub Release assets from earlier pipes in the same run (release assets, cosign signatures, and SBOMs would already be live). This is a **partial-success** state: users can `curl`/browser-download and `codegraph upgrade`, but `brew install` breaks (formula missing or stale). Recommend the release job **not** treat a tap-push failure as fully fatal to the workflow's exit status in a way that blocks re-running — GoReleaser's own retry/re-run semantics (idempotent re-publish via `--clean` + existing-release `mode: keep-existing`) should let a second workflow run (or a manual `goreleaser continue`-equivalent) safely retry just the tap push without re-uploading already-published assets. This needs to be verified against GoReleaser's actual re-run behavior in practice, not assumed — flag as a Phase-3 acceptance test (kill the tap-push step deliberately, confirm the rest of the release is unaffected and a re-run recovers).

---

## 6. `codegraph upgrade` brew detection — seam placement

**Recommended location:** a new file within the existing `internal/upgrade` package (not a new top-level package) — e.g. `internal/upgrade/brewdetect.go` — because the detection result gates `upgrade.Run` (`internal/upgrade/upgrade.go:82-153`) at the same decision point `checkWritable` already gates it (before any download), and `Run`'s existing `Options` struct + function-var seam pattern (`resolveLatest`, `download`, `verify`, `swap` — all package-level func-typed fields defaulting to production implementations, `internal/upgrade/upgrade.go:69-72`, `:88-91`, `:123-126`, `:132-135`, `:143-146`) is exactly the established convention this repo already uses for testability (matches the `beforeChildCopy`/`getppid`/`registryDir`-style seam pattern named in the project's own conventions).

**Concrete shape, following the existing pattern exactly:**
```go
// brewdetect.go
type brewCheckFunc func(targetPath string) (bool, error)

var checkBrewManaged brewCheckFunc = defaultCheckBrewManaged // production default, package var, test-overridable

func defaultCheckBrewManaged(targetPath string) (bool, error) {
    // resolve targetPath (already os.Executable()-resolved by the CLI caller,
    // same as targetPath's existing contract, upgrade.go:79-81)
    // real signal, NOT a path-prefix guess (explicit non-goal per PROJECT.md:179):
    //   - brew --prefix codegraph (if `brew` is on PATH) resolving to a Cellar
    //     path that targetPath is a symlink into, OR
    //   - targetPath, once symlinks are resolved (filepath.EvalSymlinks),
    //     containing "/Cellar/codegraph/" in its resolved path
    // ...
}
```

Then `Run` (or the CLI-level `internal/cli` upgrade command, wherever `targetPath` is first resolved) calls `checkBrewManaged(targetPath)` and, if true, returns a fixed, actionable error (*"codegraph was installed via Homebrew; run `brew upgrade codegraph` instead"*) **before** `checkWritable`/download — same "fail before touching anything" ordering the D-13 writable-check already establishes (`internal/upgrade/upgrade.go:117-121`).

**How to make it testable without a real Homebrew install** (the milestone's own stated bar — "tested against a real brew-managed layout, not a path-prefix guess," PROJECT.md:55, `:179`):
1. **Unit-level, via the func-var seam:** `upgrade_test.go`-style tests inject a fake `checkBrewManaged` returning true/false, proving `Run` refuses/proceeds correctly — this covers `Run`'s branching logic without touching the filesystem or `brew` at all (mirrors how `upgrade_test.go` already injects `resolveLatest`/`download`/`verify`/`swap`).
2. **Integration-level, real layout without real Homebrew:** `defaultCheckBrewManaged`'s own test should construct a **real symlink tree on disk** shaped exactly like Homebrew's actual install layout (`$(brew --prefix)/Cellar/codegraph/<version>/bin/codegraph` symlinked from `$(brew --prefix)/bin/codegraph`) inside a `t.TempDir()`, and assert detection fires on the resolved-symlink path and does NOT fire on an unrelated symlink or a plain non-symlinked binary at a path that merely contains the substring "Cellar" (the exact "path-prefix guess" this repo's own decision log calls out as an insufficient gate, PROJECT.md:179). This proves the mechanism against the real layout shape without requiring the `brew` binary or a real Homebrew installation in CI — matching how `check:darwin-toolchain`/`check:darwin-release-build` already separate "prove the mechanism" from "requires exotic host state" (`Taskfile.yml:241-345`).
3. If `brew --prefix` is used as a signal at all, it must be **optional/best-effort** (if `brew` isn't on PATH, fall back to the symlink-resolution check alone) — never a hard dependency that makes `codegraph upgrade` fail entirely on a non-brew machine that also happens to lack `brew` on PATH.

---

## 7. New vs modified components — explicit list

| Component | New / Modified | Notes |
|---|---|---|
| `.github/workflows/release.yml` | **Modified** (structural — job graph collapses) | File identity, name, and tag trigger MUST NOT change (cosign SAN contract, `internal/upgrade/verify.go:41-45`) |
| `.goreleaser.yaml` | **Modified** (multiple new blocks) | `archives:` (2nd entry added, both go live), `checksum:` (goes live), `notarize:` (new), `binary_signs:`/`signs:` (new, replaces workflow-step cosign), `sboms:` (new, replaces workflow-step syft), `homebrew_casks:` or `brews:` (new — decision pending, Finding A), `release:` (new, `mode: keep-existing`) |
| `Taskfile.yml` | **Modified** | New targets almost certainly needed: a local notarization dry-run/validation target (paralleling `check:darwin-toolchain`), a `check:goreleaser`-equivalent assertion that the new blocks are internally consistent (raw-binary format assertion from §4.2), and per this repo's own D-01/D-02 convention, any new CI step's command body belongs in a task target, not inline in the workflow YAML |
| `.github/workflows/darwin-toolchain-canary.yml` | **Modified or superseded by a new canary** | Must additionally prove zig-cross-TO-linux FROM a macOS host now that this is a real release-path dependency, not just darwin-native build proof |
| `internal/upgrade/brewdetect.go` (or similarly named) | **New** | Brew-managed-install detection, seam-testable per §6 |
| `internal/upgrade/upgrade.go` | **Modified** | Wire the new detection check into `Run` (or the CLI caller) before `checkWritable` |
| `internal/cli` (upgrade command surface) | **Possibly modified** | Actionable error message surfacing ("run `brew upgrade codegraph`") |
| `seanb4t/homebrew-tap` (external repo) | **New** | Not part of this repo's tree; created once, then written to by every release |
| Apple Developer Program enrollment, Developer ID Application cert, App Store Connect API key | **New** (external prerequisites) | CI secrets: cert (`.p12`+password) and API key (`.p8`+issuer/key IDs), least-privilege = scoped only to the release job's environment, never exposed to PR-triggered workflows |
| `HOMEBREW_TAP_TOKEN` (or equivalent PAT) | **New** (CI secret) | Least-privilege = fine-grained PAT scoped to `seanb4t/homebrew-tap` only |
| `docs/RELEASE.md` / `docs/RELEASE-PROCEDURES.md` | **Modified** | Already shown to drift from reality once before (the checksums-file-vs-binaries provenance-scope correction noted at `release.yml:205-210`) — must be updated in the same change that restructures the pipeline, not after |

---

## 8. Suggested build order

Front-loading risk means: prove the single-runner zig-cross claim and the notarization ordering/stapling reality **before** committing to a phase plan that assumes either works as hoped. Both are exactly the kind of "confident wrong conclusion from a malformed probe" this repo has a recorded history of (per milestone context) — measure, don't infer.

### Phase 0 — De-risking spikes (no shipped user-facing change)
1. **Zig-cross-linux-from-macOS spike.** On a real macOS runner (reuse `darwin-toolchain-canary.yml`'s runner class or extend it), CGO-cross-build linux/amd64 AND linux/arm64 via `zig cc` and confirm: (a) it links successfully against this project's CGo dependency (tree-sitter grammars), (b) the resulting binary runs correctly (at minimum, boots and indexes a small fixture — not just "compiles"), (c) ideally, byte-for-byte or functional parity against today's native-linux build. **This is the single highest-leverage risk in the whole milestone** — if it fails, Option A (§2.3) is dead and the milestone must fall back to Option C (GoReleaser Pro) or renegotiate scope.
2. **`homebrew_casks:` vs `brews:` decision spike** (Finding A). Small, cheap, but must land before Phase 3's tap integration is planned in detail — the two blocks have different config shapes and different `binaries:`/`ids:` semantics.
3. **Notarization container-shape decision spike** (Finding B). Confirm the online-only-notarization-via-zip acceptance is genuinely sufficient for the milestone's stated `spctl` bar (it should be, per research), and get explicit maintainer sign-off that full offline stapling is out of scope for v0.5.0 — this is a scope decision, not an engineering task, and should not be discovered mid-Phase-2.

### Phase 1 — `goreleaser build` → `goreleaser release` restructure (mechanical, no new capability)
Depends on: Phase 0 spike 1 (must know the single-runner approach works before restructuring around it).
- Rewrite `.goreleaser.yaml`: dual `archives:` entries, live `checksum:`, `binary_signs:`, `sboms:`, `release: mode: keep-existing`.
- Collapse `release.yml`'s `build`+`assemble` jobs into one macOS-runner job.
- Update the `provenance` job to consume GoReleaser-native checksums output.
- **Acceptance is measurement, not inference**: cut a real tag (or rc-style tag per this repo's existing `check:darwin-release-build`/D-10 pattern), then independently verify `cosign verify-blob`, `slsa-verifier verify-artifact`, and a live `codegraph upgrade` self-upgrade all still pass — the same three checks this repo's own history already performs at every release-pipeline change (PROJECT.md:13, `:94`).
- This phase alone delivers zero new user-facing capability but re-proves the entire existing supply-chain claim under the new mechanism, isolated from notarize/brew risk — any regression here is unambiguously attributable to the restructure.

### Phase 2 — Apple codesign + notarization
Depends on: Phase 1 shipped and verified (notarization slots into an already-working `goreleaser release` pipeline, not a half-migrated one); Phase 0 spike 3's scope decision.
- Acquire and wire Apple Developer ID Application cert + App Store Connect API key as least-privilege CI secrets.
- Add `notarize.macos`, scoped via `ids:` to darwin build ids only.
- **Correctness acceptance gate (§3.2)**: verify the notarized bytes are what's actually archived/checksummed/signed/SBOM'd/attested — diff sha256 at each stage, don't trust pipe-ordering docs alone.
- **User-facing acceptance gate**: `spctl -a -vv -t exec` moves `rejected` → `accepted` on a genuinely `com.apple.quarantine`-tagged download (real browser download, not a synthetic xattr set) — the milestone's own stated bar (PROJECT.md:52, `:63`).

### Phase 3 — Dual-asset archives + Homebrew tap
Depends on: Phase 1 (archives infrastructure) and Phase 2 (notarized darwin bytes should be what the zip contains — sequencing archives after notarization is already enforced by GoReleaser's pipe order, §3, but the darwin archive's *content* isn't meaningfully "done" until Phase 2 ships, even though the archive *mechanism* itself is Phase-1 work covering all 4 platforms).
- Finalize `homebrew_casks:`/`brews:` config (per Phase 0 spike 2's decision).
- Create `seanb4t/homebrew-tap`, wire the least-privilege PAT.
- **Acceptance gate**: `brew tap seanb4t/tap && brew install codegraph` on a clean machine (the milestone's own stated bar, PROJECT.md:54).
- **Failure-mode acceptance gate (§5)**: deliberately fail the tap-push step and confirm the rest of the release (binaries, cosign, SLSA, GitHub Release) is unaffected, then confirm a re-run recovers the tap push without duplicating or corrupting already-published assets.

### Phase 4 — `codegraph upgrade` brew detection
Depends on: nothing from Phases 1-3 structurally (it's pure Go logic in `internal/upgrade`), but should land **after** Phase 3 so the real Homebrew-managed layout it's tested against (§6.2) reflects the actual shipped tap/formula shape rather than a guessed one. Can be developed in parallel with Phases 1-3 if a synthetic Homebrew-shaped fixture is built early, but final acceptance against a **real** `brew install` (per the milestone's explicit non-goal of a path-prefix guess) should follow Phase 3.
- Implement `checkBrewManaged` seam per §6.
- Unit tests via the func-var seam (parallel-safe with Phases 1-3).
- Integration test against a constructed real-shaped Cellar symlink tree (§6.2, item 2).
- Final acceptance: install via the real published tap (Phase 3's output), attempt `codegraph upgrade`, confirm refusal + correct pointer message, confirm `brew upgrade codegraph` still works.

### Ordering summary

```
Phase 0 (spikes, parallel)
   ├─ 0.1 zig-cross-linux-from-macOS  ─────────┐
   ├─ 0.2 homebrew_casks vs brews decision      │
   └─ 0.3 notarization container-shape decision │
                                                 ▼
                                          Phase 1 (restructure)
                                                 │
                        ┌────────────────────────┴───────────────┐
                        ▼                                        ▼
                 Phase 2 (notarize)                   Phase 4 (brew detection,
                        │                               dev can start early;
                        ▼                               final accept waits)
                 Phase 3 (archives + tap) ──────────────────────►│
                                                                  ▼
                                                          Final acceptance
```

---

## 9. Anti-patterns to avoid

### Anti-pattern: trusting `dist/artifacts.json` layout stability across a hand-rolled multi-runner recombination
**What people do:** try to run `goreleaser build` per-runner (as today) then stitch results into a single `goreleaser release --skip=build` invocation by manually reconstructing GoReleaser's internal dist layout.
**Why it's wrong:** this repo's own code already documents that layout as explicitly unstable across versions (`release.yml:166-176`). It's reimplementing GoReleaser Pro's split/merge without its guarantees — see §2.3, Option B.
**Instead:** run the whole `goreleaser release` on one runner (Option A), or pay for Pro's actual split/merge (Option C).

### Anti-pattern: silently keeping the deprecated `brews:` block because PROJECT.md happened to name it
**What people do:** implement exactly what a planning doc says, even after research surfaces that the named mechanism is documented-deprecated by its own maintainers.
**Why it's wrong:** PROJECT.md is a planning artifact, not ground truth about GoReleaser's current API surface — and per this repo's own "measure, don't infer" standing rule, a stale assumption baked into a decision log is exactly the failure mode to catch, not propagate.
**Instead:** flag it (done, §1 Finding A), let the roadmap make an explicit choice with the current facts.

### Anti-pattern: treating "notarized" and "stapled" as the same property
**What people do:** ship a notarized-but-unstapled zip and describe it as "fully offline-Gatekeeper-verifiable."
**Why it's wrong:** it isn't — see Finding B. It IS `spctl`-acceptable when network is available, which is a real, valuable, and probably sufficient property for this milestone, but claiming more than that invites a false "verified" status later.
**Instead:** state the actual guarantee precisely in `docs/RELEASE.md` and PROJECT.md's decision log once this ships: "notarized, online-verified; not stapled."

---

## Sources

- `.planning/PROJECT.md` (HIGH — direct read, this repo)
- `.github/workflows/release.yml` (HIGH — direct read, this repo)
- `.goreleaser.yaml` (HIGH — direct read, this repo)
- `Taskfile.yml` (HIGH — direct read, this repo)
- `internal/upgrade/upgrade.go`, `verify.go`, `release.go`, `swap.go` (HIGH — direct read, this repo)
- GoReleaser official docs: notarization (`/customization/sign/notarize/`), Homebrew casks (`/customization/publish/homebrew_casks/`), Homebrew formulas/deprecation (`/customization/publish/homebrew_formulas/`), archives (`/customization/archive/`), signing (`/customization/sign/`), binary signing (`/customization/sign/binary_sign/`), releases/existing-release mode (`/customization/release/`), split/merge (`/customization/general/partial/`), DMG (`/customization/package/dmg/`) — MEDIUM confidence each (official source, single-pass fetch via a summarizing tool this session, not independently cross-checked against a second source or against a real `goreleaser release` run)
- GoReleaser Pro feature-boundary claims (split/merge, native `dmg:`/`notarize.macos_native`) — MEDIUM confidence, consistent across multiple official-doc fetches this session, not verified against a live Pro trial

---
*Architecture research for: macOS Gatekeeper notarization + Homebrew distribution integration into codegraph-go's existing signed/attested GoReleaser release pipeline*
*Researched: 2026-08-07*
