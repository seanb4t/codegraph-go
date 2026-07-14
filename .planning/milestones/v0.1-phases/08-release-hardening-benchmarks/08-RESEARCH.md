# Phase 8: Release Hardening & Benchmarks - Research

**Researched:** 2026-07-13
**Domain:** Go supply-chain release engineering (CGo cross-compilation, keyless signing, SLSA provenance, SBOM, reproducible builds) + performance benchmarking (Go vs Node.js head-to-head, CI regression gates)
**Confidence:** MEDIUM-HIGH — the release-identity contract, GoReleaser config schema, and cosign v3 bundle format are HIGH confidence (verified against the actual shipped `internal/upgrade` code + official GoReleaser/cosign docs). The CGo darwin cross-compile risk assessment and the GoReleaser-Pro-feature boundary are MEDIUM confidence (consistent multi-source web corroboration, not independently executed in this session). The exact `goreleaser-cross` image tag availability for go1.26.5 and the precise cross-target reproducibility percentage are LOW confidence (unverified, flagged for a Wave-0 spike).

## Summary

This phase has one inversion that dominates everything else: **the verifier shipped in Phase 6, the signer ships now.** `internal/upgrade/verify.go` and `internal/upgrade/upgrade.go` are not a blank slate to design against — they are an *executable contract* that already dictates, byte-for-byte, what the release workflow must produce: a raw (unarchived) binary named `codegraph_<tag>_<goos>_<goarch>[.exe]`, plus a companion `<same-name>.sigstore.json` cosign v3 bundle whose recorded artifact digest is the **sha256 of that individual binary** (not the checksums file), signed by a certificate whose issuer is GitHub's OIDC issuer and whose SAN matches `^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`. This is the single most important finding in this research and it **corrects CONTEXT.md's D-02 decision**: signing only the checksums file (as D-02 literally states) will make `codegraph upgrade` fail signature verification on every real download — cosign must sign each release binary individually. See Finding 1 below.

The second load-bearing finding concerns D-01's single-Linux-runner `zig cc` plan for all 6 targets. Cross-compiling CGo to `darwin` from a Linux container requires either `osxcross` (an unofficial macOS-SDK-sourcing toolchain with a real, documented licensing gray area) or `zig cc` targeting darwin (which multiple independent sources report as unreliable specifically because Apple's SDK libraries, including the DNS resolver `libresolv`, are not distributable and not present in a Linux build environment — this breaks `net/http`'s cgo-based DNS resolution at link time or at runtime, which is a severe risk for this specific binary since `internal/upgrade` and future MCP/network code depend on working DNS). GoReleaser's own Pro-only features (`split`/`continue --merge`, the `prebuilt` builder) are the *documented* way to combine builds from multiple native runners, and this project has no indication of a GoReleaser Pro license. **Recommendation: use a native 2-OS runner matrix (`ubuntu-latest` + `macos-latest`) with `goreleaser build --single-target` (an OSS command) per target, then a manual assembly job (plain shell + cosign + syft + the SLSA generic generator) that does not depend on any GoReleaser Pro feature.** This sidesteps both the osxcross/DNS risk and the Pro paywall. `zig cc` is still the right tool for Linux and Windows cross-targets *from* the Linux runner (those do not carry the darwin SDK/DNS risk).

The third finding: GoReleaser's `sbom:`/`sboms:` block shells out to a real `syft` binary that must be installed in CI (`anchore/sbom-action/download-syft`) — it is not bundled. SLSA provenance should run via the *generic* generator (`generator_generic_slsa3.yml`) over the checksums file, which is architecturally correct per CONTEXT.md D-02 and does not conflict with the per-binary cosign requirement (provenance and code-signing serve different verifiers and can have different scopes).

**Primary recommendation:** Build with a native `ubuntu-latest` + `macos-latest` runner matrix using `goreleaser build --single-target` per target (zig cc for linux/windows cross, native Xcode clang for both darwin arches on the macOS runner), assemble raw binaries named `{{.ProjectName}}_{{.Tag}}_{{.Os}}_{{.Arch}}[.exe]`, cosign-sign **each binary individually** with `--bundle=${artifact}.sigstore.json`, generate SBOM via syft, run SLSA provenance via the generic generator over the checksums file, and gate CI with `govulncheck` plus a `linux/amd64`-blocking double-build hash-diff.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cross-platform binary build (DIST-01) | CI/CD (build tier) | — | Pure build-time concern; no runtime component |
| Artifact signing (DIST-02) | CI/CD (release tier) | Binary (verify-time) | Signing happens in CI; the *already-shipped* verifier (`internal/upgrade`) is the consuming runtime tier — this phase's CI output must satisfy that existing runtime contract, not the other way around |
| SLSA provenance (DIST-02) | CI/CD (release tier) | — | Attestation is generated and published at release time; consumed by external verifiers (`slsa-verifier`), not by the binary itself |
| SBOM (DIST-03) | CI/CD (release tier) | — | Static artifact describing the build; no runtime dependency |
| Vulnerability gate (DIST-03) | CI (test/gate tier) | — | Blocks merge/release, not a runtime concern |
| Reproducible build gate (DIST-04) | CI (test/gate tier) | — | Build-determinism verification, no runtime surface |
| Benchmark harness (PERF-01/02, INDX-06) | CI (bench tier) + standalone Go package (`internal/bench`) | Binary (subject under test) | The harness *drives* both the Go binary and the TS binary as external subprocesses and measures at the OS level — it is not instrumentation inside either subject |
| Synthetic corpus generator (PERF-02) | Standalone Go tool (`tools/bench/gencorpus`) | — | Pure generation logic, no product-runtime coupling |

## Package Legitimacy Audit

This phase introduces **no new `go.mod` production dependencies** — DIST-05 (dependency audit) is already Complete, and every tool this phase adds (GoReleaser, cosign, syft, govulncheck, `slsa-github-generator`) is installed as a CI-time binary or GitHub Action, exactly per the existing `STACK.md` convention ("goreleaser + slsa-github-generator installed as GitHub Actions, not go-installed"). The audit below covers those CI tools instead of `go.mod` packages.

| Tool | Distribution | Verified version/tag | Verdict | Disposition |
|------|--------------|----------------------|---------|-------------|
| `goreleaser` | Homebrew/binary release | v2.17.0 confirmed installed locally `[VERIFIED: local `goreleaser --version`]` | OK | Approved — pin `~> v2` in CI via `goreleaser/goreleaser-action` |
| `goreleaser/goreleaser-action` | GitHub Action | `@v7` referenced in current official examples `[CITED: goreleaser.com/customization/ci/actions]` | OK | Approved — pin to a specific `@vX.Y.Z` tag, not `@v7` floating major, at execution time |
| `sigstore/cosign-installer` | GitHub Action | `@v3` / `@v3.1.2` seen in current official examples `[CITED: goreleaser.com/blog/cosign-v3]` | OK | Approved — pin exact tag at execution time |
| `sigstore/cosign` (CLI, v3) | installed via cosign-installer | v3.x — bundle-format (`--bundle`) is v3's headline change `[CITED: goreleaser.com/blog/cosign-v3]` | OK | Approved |
| `slsa-framework/slsa-github-generator` | reusable workflow | `generator_generic_slsa3.yml@vX.Y.Z` — **must be a full semver tag**, short tags (`@v2`) are rejected by `slsa-verifier` `[CITED: slsa-github-generator README]` | OK | Approved — resolve the exact current tag at execution time (do not hardcode a possibly-stale tag in the plan) |
| `anchore/sbom-action` (`download-syft` sub-action) | GitHub Action | installs `syft` binary for GoReleaser's `sbom:` block to shell out to `[CITED: goreleaser.com/blog/supply-chain-security]` | OK | Approved — required, GoReleaser does not bundle syft |
| `golang.org/x/vuln/cmd/govulncheck` | `go install` or `golang/govulncheck-action@v1` | official Go team tool `[VERIFIED: local `govulncheck` binary present at `/Users/sean/go/bin/govulncheck`]` | OK | Approved |
| `golang.org/x/perf/cmd/benchstat` | `go install` | official `golang.org/x` tool for statistical benchmark comparison `[CITED: pkg.go.dev/golang.org/x/perf/cmd/benchstat]` | OK | Approved |
| `zig` (compiler, used as `CC`/`CXX`) | CI-installed (`goreleaser/setup-zig` or apt/direct download) | deterministic C/C++ cross-compiler `[CITED: goreleaser docs, multiple independent blog corroboration]` | OK | Approved for linux+windows cross targets only — see Finding 2 for the darwin caveat |
| `goreleaser-cross` (ghcr.io Docker image) | Docker image | bundles osxcross+mingw-w64+gnu cross-toolchains `[CITED: github.com/goreleaser/goreleaser-cross README]` — **image tag must be re-verified to have a go1.26.5-matching build at execution time; not confirmed in this session** | SUS (image-tag freshness unverified) | Not recommended as primary path — see Finding 2; if used at all, planner must add a `checkpoint:human-verify` before relying on it for a real release |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** `goreleaser-cross` Docker image tag freshness (go1.26.5 compatibility unverified this session) — not selected as the primary recipe, so this does not block planning, but if the planner falls back to it, gate with `checkpoint:human-verify`.

## Finding 1 (CRITICAL — corrects CONTEXT.md D-02): cosign must sign each binary individually, not just the checksums file

**Read directly from the shipped code**, not assumed:

`internal/upgrade/upgrade.go` `defaultDownload` (line ~175):
```go
func defaultDownload(version string) (binary []byte, bundleJSON []byte, err error) {
	assetName := releaseAssetName(version)
	binary, err = downloadReleaseAsset(version, assetName)
	...
	bundleJSON, err = downloadReleaseAsset(version, assetName+".sigstore.json")
	...
}

func releaseAssetName(version string) string {
	ext := ""
	if runtime.GOOS == "windows" { ext = ".exe" }
	return fmt.Sprintf("codegraph_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext)
}
```
`internal/upgrade/upgrade.go` `defaultVerify`:
```go
func defaultVerify(binary, bundleJSON []byte) error {
	b, err := loadBundle(bundleJSON)
	...
	digest := sha256.Sum256(binary)
	return verifyRelease(b, trustedMaterial, "sha256", digest[:], releaseWorkflowRefPattern)
}
```
`verifyRelease` builds its policy with `verify.WithArtifactDigest(digestAlgorithm, artifactDigest)` where `artifactDigest` is **the sha256 of the individual downloaded binary**, and rejects the bundle if the subject digest recorded inside the sigstore bundle doesn't match.

**Consequence `[VERIFIED: read from source]`:** a `.sigstore.json` bundle produced by signing `checksums.txt` records checksums.txt's own digest as its subject — it will **never** match `sha256(binary)` for any individual downloaded binary. If the release workflow only signs the checksums file (as CONTEXT.md's D-02 literally states — "cosign v3 keyless signs the checksums file"), **every real `codegraph upgrade` invocation will fail signature verification against a genuine, correctly-published release.** This is not a hypothetical edge case; it is the default path every user hits.

**Fix (what the plan must do instead):**
- `version` here is the **raw git tag** (e.g. `v1.2.3`, with the `v` prefix) — confirmed from `internal/upgrade/release.go`'s `tagFromLocation` regex, which extracts the full `/releases/tag/v1.2.3` path segment verbatim. GoReleaser's `{{ .Version }}` template variable is normally the **tag with the `v` stripped**; `{{ .Tag }}` is the one that keeps it. The archive `name_template` **must use `{{ .Tag }}`, not `{{ .Version }}`**, or every download 404s.
- Configure GoReleaser to publish **raw, unarchived binaries** (not `.tar.gz`/`.zip`) via `archives: - format: binary`, matching `releaseAssetName`'s expected literal shape:
  ```yaml
  archives:
    - format: binary
      name_template: >-
        {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}{{ if eq .Os "windows" }}.exe{{ end }}
  ```
  `[CITED: goreleaser.com/customization/package/archives.md — "format: binary" publishes the raw binary directly with a custom name_template"]`
- Configure the `signs:` block to sign **each binary artifact**, not the checksums file:
  ```yaml
  signs:
    - cmd: cosign
      signature: "${artifact}.sigstore.json"
      args:
        - sign-blob
        - "--bundle=${signature}"
        - "${artifact}"
        - "--yes"
      artifacts: binary   # NOT `checksum` — see Finding 1
  ```
  `[CITED: goreleaser.com/customization/sign/sign.md — the documented sign-blob+--bundle recipe]`. GoReleaser's docs note the `signs` pipe **requires `archives.format: binary`** when `artifacts: binary` is set — consistent with the change above. `[CITED: goreleaser.com/customization/sign/binary_sign.md]`
- **Separately**, SLSA provenance (a different verifier, a different consumer) can still legitimately run over the checksums file per CONTEXT.md D-02 — that part of D-02 is correct and does not need to change. `codegraph upgrade` never checks SLSA provenance; it only checks the per-binary sigstore bundle. Both attestations should ship; they serve different audiences.
- **cosign v3 changed its own CLI surface** relevant here: the old two-file `--output-certificate`/`--output-signature` pattern is superseded by a single `--bundle=${signature}` flag producing one `${artifact}.sigstore.json` file `[CITED: goreleaser.com/blog/cosign-v3]` — this is *exactly* the file shape `internal/upgrade` already expects (`bundle.Bundle.UnmarshalJSON` from `sigstore-go`), which is a good sign the two were designed compatibly, but it means the plan must use the **v3 bundle flag**, not the deprecated `certificate`+`args` two-file pattern still shown in some older GoReleaser examples.
- **Action item for the planner:** add exactly the end-to-end test CONTEXT.md's `<specifics>` section already calls for — a real signed artifact from an actual (or CI-simulated) release run must pass `verifyRelease` against the production identity constants. This is the only way to close the loop on Finding 1 before it's discovered in production by a real user's failed upgrade.

## Finding 2 (corrects CONTEXT.md D-01 default): darwin CGo cross-compile from Linux carries real, cited risk; recommend a native 2-runner matrix instead of a single Linux runner for all 6 targets

**The risk, cited from multiple independent sources, not a single low-confidence blog:**
- Apple does not distribute macOS SDK libraries for redistribution. Cross-compiling CGo-linked binaries targeting `darwin` from a Linux container requires either `osxcross` (an unofficial SDK-extraction toolchain) or `zig cc -target ...-macos` with a manually supplied SDK tarball via `SDKROOT`. `[CITED: johncodes.com/archive/2026/02-11-cross-compiling-cgo]`
- **The specific, concrete failure mode:** native (non-cross) Go builds use Go's own pure-Go DNS resolver; **cross-compiled CGo builds targeting darwin fall back to the system libc resolver** (`libresolv`), which is one of the libraries not distributable/available inside the Linux cross-compile environment. This can surface as either a hard link failure (missing symbol) or — worse — a binary that links successfully against a partial/incorrect `osxcross`-sourced SDK but has broken DNS resolution at runtime. `[CITED: johncodes.com/archive/2026/02-11-cross-compiling-cgo — "native Go builds use Go's internal DNS resolver while cross-compiled builds utilize the system's library resolver"]`. This is directly relevant to this project: `internal/upgrade` makes real HTTPS calls (GitHub Releases, Sigstore TUF root fetch) from every platform's binary, including darwin — a broken resolver on the darwin release artifact would silently break `codegraph upgrade` for every macOS user, the exact opposite of what this phase exists to guarantee.
- **Mitigation that exists but is unverified in this session:** the Go `net` package's `netgo` build tag forces the pure-Go resolver even when `CGO_ENABLED=1`, independent of other CGo code in the binary (tree-sitter). Adding `-tags netgo` to the darwin cross-build *may* fully sidestep the resolver problem without needing a native macOS runner at all. `[ASSUMED — this specific interaction (netgo tag + tree-sitter CGo coexisting in one binary + zig-cross-linked darwin target) was not found verified against a real project in the sources gathered this session; treat as a candidate mitigation to spike, not a proven fix]`.
- **Blog-level confirmation that the "eventual correct" answer is a native runner, not a deeper cross-compile investment:** *"this isn't the most practical and eventually we'll probably align on running this part of the build & release pipeline natively on macOS"* `[CITED: johncodes.com/archive/2026/02-11-cross-compiling-cgo]`. A second independent source (`blog.afoolishmanifesto.com`) reports the author abandoning zig-based macOS cross-compilation entirely for unrelated (codesigning) reasons, corroborating that this specific cross-target is the outlier in an otherwise-solid `zig cc` story for Linux/Windows.
- **The GoReleaser-native path for combining multi-runner builds is Pro-only.** `goreleaser release --split` + `goreleaser continue --merge`, and the `prebuilt` builder (importing binaries built elsewhere), are both explicitly documented as **GoReleaser Pro** features. `[CITED: goreleaser.com/customization/general/partial.md — "With GoReleaser Pro, you can split and merge..."; carlosbecker.com/posts/goreleaser-prebuilt — the prebuilt builder]`. This repo has GoReleaser OSS v2.17.0 installed locally with no evidence of a Pro license anywhere in the codebase or `.planning/` artifacts.

**Recommended recipe (primary, not fallback) — native 2-OS runner matrix, no GoReleaser Pro feature required:**

| Runner | Targets built | Toolchain |
|--------|--------------|-----------|
| `ubuntu-latest` | `linux/amd64` (native), `linux/arm64` (cross), `windows/amd64` + `windows/arm64` (cross) | native gcc for linux/amd64; `zig cc -target aarch64-linux-gnu` for linux/arm64; `zig cc -target x86_64-windows-gnu` / `aarch64-windows-gnu` for windows — Windows does **not** carry the darwin DNS-resolver risk: Go's `net` package uses its own resolver on Windows regardless of CGO, per Go's own resolver-selection documentation `[ASSUMED — carried from general Go `net` package resolver-selection knowledge, not independently re-verified this session against go1.26.5's current docs; low-risk assumption since this behavior has been stable across many Go releases]` |
| `macos-latest` (Apple Silicon host) | `darwin/arm64` (native) + `darwin/amd64` (cross-arch, same-OS) | Xcode's own `clang`, `-arch x86_64`/`-arch arm64` — this is a **same-OS** cross (arch-only, not OS-boundary), which Xcode supports natively with the real, licensed SDK already on the runner. No `osxcross`, no manually-sourced SDK tarball, no DNS-resolver risk. |

Each target is built independently via **`goreleaser build --single-target`** (an OSS command — confirmed no Pro gate on `build`, only on `release --split`/`continue --merge`/the `prebuilt` builder), invoked once per (`GOOS`,`GOARCH`) matrix cell on its appropriate runner. Each job uploads its one binary as a GitHub Actions artifact (`actions/upload-artifact`). A final `ubuntu-latest` job downloads all 6 artifacts and performs the **assembly step manually** (plain shell, not a GoReleaser Pro merge):
1. Rename/verify each binary matches `codegraph_<tag>_<os>_<arch>[.exe]`.
2. `sha256sum codegraph_* > codegraph_<tag>_checksums.txt`.
3. `cosign sign-blob --bundle=<binary>.sigstore.json --yes <binary>` — once **per binary** (Finding 1).
4. `syft <binary> -o spdx-json=<binary>.spdx.json` (or catalog the whole `dist/` directory in one pass) — installed via `anchore/sbom-action/download-syft`.
5. Call `slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@<pinned-tag>` with `base64-subjects` built from the checksums file, per the standard recipe (see Code Examples).
6. `gh release create <tag> codegraph_* codegraph_*.sigstore.json codegraph_*.spdx.json codegraph_*_checksums.txt` (or `softprops/action-gh-release`) to publish everything as one release.

**Documented fallback (higher risk, explicitly not primary) — the `goreleaser-cross` Docker image on a single Linux runner:** bundles `osxcross` + `mingw-w64` + gnu cross-toolchains in one container, is OSS (not Pro-gated), and lets `goreleaser release` run its full pipeline (build → archive → checksum → sbom → sign) in one job. `[CITED: github.com/goreleaser/goreleaser-cross]` This still carries the exact darwin SDK/DNS risk described above (it uses `osxcross` internally) and its image-tag-to-Go-version alignment for `go1.26.5` was **not verified in this session** — flag `[SUS]`, gate behind `checkpoint:human-verify` if the planner chooses this path over the native-matrix recommendation.

**Either way, `-trimpath` + cleared `-buildid=` + `mod_timestamp: '{{ .CommitTimestamp }}'` (reproducibility, Finding/D-03) apply uniformly regardless of which cross-compile strategy is chosen — these are Go-toolchain-level flags, not GoReleaser-pipeline-level.**

## Standard Stack

### Core

| Library/Tool | Version | Purpose | Why Standard |
|---|---|---|---|
| GoReleaser | v2.17.0 confirmed installed `[VERIFIED: local goreleaser --version]`; pin via `goreleaser/goreleaser-action@<pinned tag>` in CI | Build orchestration, archiving, checksums, SBOM/signs config blocks | Already the project's decided stack (`CLAUDE.md`); OSS-tier commands (`build --single-target`, `release`) are sufficient for this phase's design — Pro features are explicitly avoided (Finding 2) |
| cosign (v3 CLI) | latest v3.x via `sigstore/cosign-installer@<pinned tag>` `[CITED: goreleaser.com/blog/cosign-v3]` | Keyless artifact signing | `--bundle` flag produces exactly the `.sigstore.json` shape `internal/upgrade`'s `sigstore-go` bundle parser already expects |
| `slsa-framework/slsa-github-generator` | `generator_generic_slsa3.yml@<exact semver tag — resolve at execution time>` `[CITED: slsa-github-generator README — full-semver-tag requirement]` | SLSA3 provenance over checksums file | Generic generator avoids the Go-builder's rebuild-under-fixed-config conflict with zig-cc CGo (CONTEXT.md D-02, confirmed correct) |
| syft (via `anchore/sbom-action/download-syft`) | latest `[CITED: goreleaser.com/blog/supply-chain-security]` | SBOM generation | Standard, GoReleaser's own documented default cataloger |
| `golang.org/x/vuln/cmd/govulncheck` | latest, `[VERIFIED: local binary present]` | Call-graph-aware vuln gate | Already decided stack; blocking CI gate per D-02 |
| `golang.org/x/perf/cmd/benchstat` | latest `[CITED: pkg.go.dev/golang.org/x/perf/cmd/benchstat]` | Statistical benchmark comparison, A/B regression detection | Standard companion to `go test -bench`; computes median + confidence interval, exactly matching D-05's median-of-5 methodology |
| `zig` | pinned version (deterministic compiler — good for DIST-04) | `CC`/`CXX` for linux/arm64 + windows/amd64+arm64 cross builds from `ubuntu-latest` | `[CITED: goreleaser/example-zig-cgo]`, restricted per Finding 2 to non-darwin targets |

### Supporting

| Library | Purpose | When to Use |
|---|---|---|
| `actions/upload-artifact` / `actions/download-artifact` | Pass built binaries between matrix jobs | Native-runner-matrix assembly step (Finding 2) |
| `golang/govulncheck-action@v1` | Wraps govulncheck as a first-class Action with SARIF support | Simpler than manual `go install` + invoke in CI; `[CITED: golang/govulncheck-action]` |
| `gh release create` / `softprops/action-gh-release` | Publish assembled release assets | Final step of the manual-assembly recipe (Finding 2) |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|---|---|---|
| Native 2-OS runner matrix + manual assembly | `goreleaser-cross` Docker image, single Linux runner | Simpler single-job pipeline, but inherits osxcross's SDK/DNS risk and an unverified go1.26.5 image-tag alignment — Finding 2 |
| Native 2-OS runner matrix + manual assembly | GoReleaser Pro (`--split`/`continue --merge`) | Would let GoReleaser own the whole pipeline including cross-runner merge — requires purchasing a Pro license, out of scope unless the user explicitly opts in |
| `-tags netgo` on darwin zig cross-build | Native `macos-latest` runner (chosen) | `netgo` might fully solve the resolver problem for less CI cost, but is unverified this session against tree-sitter's own CGo coexisting in the same binary — worth a Wave-0 spike, but do not bet the release pipeline on an unverified mitigation when a proven-safe native runner is one YAML block away |

**Installation (CI, not `go.mod`):**
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
go install golang.org/x/perf/cmd/benchstat@latest
# cosign, goreleaser, slsa-github-generator, syft: installed as GitHub Actions in CI, not go-installed
```

**Version verification performed this session:**
- `goreleaser --version` → v2.17.0, confirmed locally installed `[VERIFIED]`.
- `govulncheck` binary present at `/Users/sean/go/bin/govulncheck` `[VERIFIED]`.
- `cosign` binary present at `/opt/homebrew/bin/cosign` `[VERIFIED]` — version not queried this session, assume v3.x per Homebrew's current formula; **planner should re-run `cosign version` at execution time** to confirm v3, not v2.
- `zig`, `syft` **not installed locally** — both are CI-only per the recommended design; no local verification possible or needed.
- Go toolchain: `go1.26.5 darwin/arm64` `[VERIFIED: go version]`, matching `go.mod`'s `go 1.26.5` directive.
- `/opt/homebrew/bin/codegraph` → TS CodeGraph `1.3.1` confirmed installed and runnable `[VERIFIED: codegraph --version]` — the head-to-head benchmark target is real and available, not hypothetical.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────────┐
                         │   git tag v* pushed                          │
                         └──────────────────┬────────────────────────────┘
                                            │ triggers .github/workflows/release.yml
                    ┌───────────────────────┼────────────────────────────┐
                    │                       │                            │
             ┌──────▼──────┐        ┌───────▼───────┐            ┌───────▼───────┐
             │ ubuntu-latest│        │ ubuntu-latest │            │  macos-latest │
             │ linux/amd64  │        │ linux/arm64   │            │ darwin/arm64  │
             │ (native gcc) │        │ (zig cc)      │            │ (native clang)│
             └──────┬──────┘        └───────┬───────┘            └───────┬───────┘
                    │  goreleaser build --single-target (per target, x6) │
                    │  (windows/amd64+arm64 via zig on ubuntu; darwin/amd64 via -arch x86_64 on macos)
                    └──────────────────┬──────────────────────────────────┘
                                       │ actions/upload-artifact (raw binaries)
                                ┌──────▼──────┐
                                │ assemble job │  ubuntu-latest
                                │ (final)      │
                                └──────┬──────┘
                    ┌──────────────────┼──────────────────────┐
                    │                  │                      │
            sha256sum → checksums.txt  │              cosign sign-blob --bundle
                                       │              (PER BINARY — Finding 1)
                    ┌──────────────────┼──────────────────────┐
                    │                  │                      │
             syft SBOM per binary      │        slsa generic generator
                                       │        (over checksums.txt)
                    └──────────────────┼──────────────────────┘
                                       │
                                gh release create
                             (uploads: binaries + .sigstore.json
                              + .spdx.json + checksums.txt + provenance)
                                       │
                                       ▼
                    ┌──────────────────────────────────────┐
                    │  internal/upgrade (already shipped)   │
                    │  downloads binary + <name>.sigstore.json│
                    │  verifies via sigstore-go bundle parser │
                    │  + Sigstore public-good TUF trust root  │
                    └──────────────────────────────────────┘

  Separate, parallel CI (.github/workflows/ci.yml, on PR/push):
   go test ./...  →  govulncheck ./...  →  double-build hash-diff (linux/amd64, blocking)
                                        →  perf-regression gate (synthetic 100k corpus, committed baseline)

  Separate, on-demand (.github/workflows/bench.yml):
   build fresh codegraph binary  →  shell out to /opt/homebrew-equivalent installed TS codegraph@1.3.1
                                  →  median-of-5 runs, OS-level RSS via exec.Cmd.ProcessState.SysUsage()
                                  →  docs/BENCHMARKS.md raw numbers published
```

### Recommended Project Structure
```
.github/workflows/
├── release.yml         # LOCKED name/trigger (Phase-6 verify.go contract) — tag v[0-9]* only
├── ci.yml               # test suite + govulncheck + double-build gate + perf-regression gate
└── bench.yml            # on-demand/scheduled head-to-head publish (not blocking)
.goreleaser.yaml         # repo root — build/archive/checksum config (signs/sbom done manually per Finding 2)
internal/bench/          # measurement primitives: OS-level RSS capture, median-of-N runner, metric types
tools/bench/
├── gencorpus/           # deterministic synthetic 100k+ file corpus generator (network-free)
├── realcorpus/          # pinned real-repo manifest (reuses tools/spike/testdata pattern — now internal/bench-scoped)
└── runner/              # head-to-head CLI: shells out to both Go + TS binaries, captures metrics, writes baseline/current JSON
docs/
├── RELEASE.md           # verify signature + provenance + SBOM commands (user-facing)
└── BENCHMARKS.md         # methodology + raw per-repo numbers
```

### Pattern 1: OS-level peak-RSS capture via `exec.Cmd.ProcessState.SysUsage()`

**What:** After a child process exits, `exec.Cmd.Wait()` populates `ProcessState`, whose `SysUsage()` method returns `*syscall.Rusage` on Unix (Linux and Darwin both implement `wait4`-backed rusage). This is the correct **external, OS-level** measurement Peak RSS methodology (D-05) requires — it works identically for the Go binary and the TS/Node binary since it measures the *child process*, not anything internal to either runtime.

**When to use:** Every benchmark run of both the Go and TS `codegraph` binaries — this is the only fair, apples-to-apples RSS comparison, per D-05's own reasoning.

**Critical, easy-to-miss pitfall:** `Rusage.Maxrss` units differ by OS. **On Linux, `ru_maxrss` is in kilobytes. On macOS (BSD-derived), it is in bytes.** `[CITED: multiple corroborating sources — nodejs/node issue #44332 documents Node itself hitting exactly this cross-platform inconsistency in `process.resourceUsage().maxRSS`; general BSD/Linux getrusage documentation confirms the KB-vs-bytes split]`. The harness **must** branch on `runtime.GOOS` and normalize (divide macOS's raw byte value by 1024, or keep everything in bytes and multiply Linux's KB value by 1024) before writing a metric to the committed baseline JSON — failing to normalize this will silently produce a "1024x regression" or "1024x improvement" false signal the first time CI runs on a different OS than the value was recorded on.

**Example:**
```go
// Source: pattern derived from Go's documented exec.Cmd.ProcessState.SysUsage()
// contract (pkg.go.dev/os/exec) + the Linux-vs-macOS ru_maxrss unit split
// documented in nodejs/node#44332. [CITED — see Sources]
func peakRSSBytes(state *os.ProcessState) (int64, error) {
	ru, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0, fmt.Errorf("bench: platform does not expose syscall.Rusage")
	}
	switch runtime.GOOS {
	case "linux":
		return ru.Maxrss * 1024, nil // Linux: ru_maxrss is in KB
	case "darwin":
		return ru.Maxrss, nil // Darwin: ru_maxrss is already in bytes
	default:
		return 0, fmt.Errorf("bench: unsupported OS for RSS measurement: %s", runtime.GOOS)
	}
}
```
**Windows note:** `syscall.Rusage` with a populated `Maxrss` is a Unix-only contract; Windows does not support this path the same way. Since DIST-01's platform matrix includes Windows but INDX-06/PERF-01's headline RSS number is explicitly about comparing against the TS **Node** process (D-05's own stated rationale), and the benchmark corpus/harness is not required to run on all three OSes identically — **recommend running the benchmark harness itself only on Linux and/or macOS CI runners**, documenting Windows as out of scope for the RSS metric specifically (the build/signing/binary-availability guarantees still cover Windows via DIST-01; only the *benchmark measurement* is scoped narrower). Flag this as an explicit scoping note for the planner, not a silent gap.

### Pattern 2: Deterministic synthetic corpus generator for the CI regression gate (PERF-02, INDX-06)

**What:** A Go program (`tools/bench/gencorpus`) that, given a fixed seed, deterministically materializes a 100k+-file directory tree with syntactically valid, cross-referencing source files across the project's supported languages (weighted toward Go, per this project's own priority-language ordering) — entirely offline, no network, no external repo clone.

**When to use:** The PERF-02/INDX-06 CI gate, which per CONTEXT.md D-04 must never depend on network access or flake on a remote clone.

**Design guidance** `[ASSUMED — no single authoritative external source for "the" synthetic-corpus pattern; this is standard practice reasoning, consistent with the project's own established pinned-fixture discipline in `tools/spike/testdata/` and `testdata/golden/`]`:
- Seed the RNG once (`rand.New(rand.NewSource(fixedSeed))`) so the same seed always produces byte-identical output — this is what makes the committed baseline metrics meaningful across CI runs.
- Generate files with **real cross-file references** (imports, calls between generated symbols) — a corpus of 100k files with zero edges would understate indexing cost, since cross-file resolution (RES-01) is a real cost center this project's own architecture explicitly measures.
- Size the corpus to comfortably exceed "100k+ files" (e.g., 120k) so INDX-06's literal requirement is unambiguously met even accounting for any files the generator itself skips (build-tag-excluded files, etc., mirroring `internal/indexer`'s real `Discover` skip logic).
- Commit the *generator*, not the 120k generated files, to git — regenerate at CI time into a scratch directory. This mirrors the project's own established "pinned, reproducible, no giant binary blobs in git" discipline.

### Pattern 3: Committed-baseline + tolerance-band regression gate

**What:** A JSON file (e.g. `tools/bench/baseline.json`) checked into git, containing the last-blessed metric values (throughput, RSS, cold-start latency) for the synthetic corpus. CI re-runs the same benchmark, computes the delta, and fails if it exceeds the tolerance band.

**When to use:** PERF-02's CI regression gate.

**Example (tolerance check, values per D-05's starting points — throughput regression > 10%, RSS growth > 15%):**
```go
// Source: pattern only — no direct external citation; implements D-05's
// explicitly stated starting tolerance values.
func checkRegression(baseline, current Metrics) error {
	throughputDelta := (baseline.FilesPerSec - current.FilesPerSec) / baseline.FilesPerSec
	if throughputDelta > 0.10 {
		return fmt.Errorf("throughput regressed %.1f%% (budget: 10%%)", throughputDelta*100)
	}
	rssDelta := (current.PeakRSSBytes - baseline.PeakRSSBytes) / baseline.PeakRSSBytes
	if rssDelta > 0.15 {
		return fmt.Errorf("peak RSS grew %.1f%% (budget: 15%%)", rssDelta*100)
	}
	return nil
}
```
**Re-bless path:** a documented, explicit command (e.g. `go run ./tools/bench/runner -rebless`) that overwrites `baseline.json` — must be a deliberate human action (a PR that touches only `baseline.json`, reviewable in isolation), never an automatic CI side effect, per D-05.

### Anti-Patterns to Avoid
- **In-process `runtime.MemStats` for peak RSS:** explicitly called out as unfair/non-comparable by CONTEXT.md's own `<specifics>` — it cannot be measured against the TS Node process at all, since the harness doesn't run inside Node's process. Always measure externally via the OS (Pattern 1).
- **Conflating SBOM generation with vulnerability scanning as one CI step:** PITFALLS.md's own research flags this as a documented anti-pattern in real Go supply-chain pipelines — run `govulncheck` as a distinct, separately-reported step from `syft` SBOM generation.
- **Trusting "the workflow ran green" as proof of SLSA level or reproducibility:** PITFALLS.md documents real projects that shipped SLSA/reproducibility *claims* without ever running `slsa-verifier` against a real published release or ever diffing two real builds' hashes. This phase's CI must include the actual verification step (double-build hash diff; an end-to-end `verifyRelease` test against a real signed artifact), not just the generation step.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Keyless artifact signing | A custom Fulcio/Rekor client, or long-lived key management | `cosign` v3 CLI via `sigstore-go`'s own bundle format (already the runtime dependency in `internal/upgrade`) | The verifier is already written against `sigstore-go`'s `bundle.Bundle` type; signing must produce exactly that shape — cosign v3's `--bundle` flag is the maintained, canonical producer of it |
| SBOM generation | A hand-rolled `go list -m all` dump | `syft` via GoReleaser's `sbom:` block | Syft resolves actual license/purl/CPE metadata and is the tool the project's own `CLAUDE.md` already designates |
| SLSA provenance | A custom attestation-signing script | `slsa-framework/slsa-github-generator`'s generic reusable workflow | Achieving actual SLSA Build L3 (isolated build identity, not influenced by the calling workflow) requires the specific reusable-workflow isolation pattern this project provides — a hand-rolled equivalent would not actually achieve L3, per PITFALLS.md's own documented gap ("build environment isn't isolated from the calling workflow") |
| Statistical benchmark comparison (median, confidence interval, regression detection) | Custom stats code over raw `go test -bench` output | `golang.org/x/perf/cmd/benchstat` | Computes median + confidence interval per-benchmark and A/B comparison out of the box; exactly matches D-05's median-of-N requirement |
| Cross-compiling CGo | Hand-written Docker cross-toolchain images from scratch | `zig cc` (linux/windows) + native Xcode clang (darwin, same-OS arch cross) | Zig's C/C++ compiler is deterministic and well-trodden for exactly this; native Xcode clang on a real macOS runner needs zero extra tooling for cross-*arch*-same-OS builds |

**Key insight:** almost everything in this phase has an existing, load-bearing consumer already shipped in this repo (`internal/upgrade`) — the temptation to "design the release pipeline fresh" must be resisted in favor of "read the contract the verifier already enforces and produce exactly that." Finding 1 exists precisely because that discipline wasn't yet applied to CONTEXT.md's D-02.

## Common Pitfalls

### Pitfall 1: Signing the checksums file only (see Finding 1)
**What goes wrong:** `codegraph upgrade` fails signature verification for every real release.
**Why it happens:** CONTEXT.md's D-02 default reads naturally as "sign the one aggregate file," which is the common pattern in *generic* GoReleaser tutorials — but this project's verifier was written against a per-binary bundle contract before this phase existed.
**How to avoid:** `signs: artifacts: binary` (not `checksum`), one `.sigstore.json` per released binary — see Finding 1's exact config.
**Warning signs:** any test that downloads a real (or CI-simulated) release asset and runs it through `verifyRelease` fails with a digest mismatch, not an identity/SAN mismatch — that specific failure mode is the signature of this exact bug.

### Pitfall 2: darwin CGo cross-compile silently breaks DNS resolution
**What goes wrong:** a `zig cc`/`osxcross`-cross-linked darwin binary either fails to link (missing `libresolv` symbols) or links against a partial/incorrect SDK and gets DNS resolution wrong at runtime — surfacing as `codegraph upgrade` (or any future networked feature) mysteriously failing only on macOS, only in CI-built releases, never in a locally-built binary (because local darwin builds are native, not cross-compiled).
**Why it happens:** Apple's SDK/DNS libraries are not distributable; a Linux cross-compile environment fundamentally lacks them, and workarounds (`osxcross`) are best-effort community reconstructions, not an official SDK.
**How to avoid:** build darwin targets on a real `macos-latest` runner (Finding 2's primary recommendation) — this isn't a cross-compile at all for `darwin/arm64` (native) and is only an arch-level (not OS-level) cross for `darwin/amd64`, both fully supported by Apple's own toolchain with zero extra tooling.
**Warning signs:** any CGo-cross-darwin build succeeds at `go build` time but hasn't been *run* (not just linked) on a real Mac before being published as a release asset.

### Pitfall 3: GoReleaser Pro-only features silently assumed available
**What goes wrong:** a plan or workflow file references `goreleaser release --split` / `goreleaser continue --merge` / the `prebuilt` builder, which fail (or require a license key) under OSS GoReleaser.
**Why it happens:** many current GoReleaser blog posts and examples (including the official `example-zig-cgo` and `example-split-merge-real` repos) demonstrate patterns that are Pro-gated without prominently flagging it in every example.
**How to avoid:** use `goreleaser build --single-target` (confirmed OSS) per runner and assemble manually (Finding 2's recipe) unless the project explicitly decides to purchase GoReleaser Pro.
**Warning signs:** `goreleaser release --split` exits with a license-related error in CI the first time it's actually run (not caught by a config-only dry validation).

### Pitfall 4: `ru_maxrss` unit mismatch across OSes silently corrupts the regression baseline
**What goes wrong:** a baseline recorded on Linux (KB units) compared against a current run on macOS (byte units) without normalization looks like either a ~1024x regression or a ~1024x improvement.
**Why it happens:** this is a genuinely obscure BSD-vs-Linux getrusage divergence that even Node.js itself has an open, acknowledged bug report about (`nodejs/node#44332`).
**How to avoid:** always normalize to one unit (Pattern 1's example normalizes to bytes) at capture time, before writing any value to the JSON baseline or comparing to it.
**Warning signs:** a "regression" that is suspiciously close to a power-of-two multiple (1024x, 1024²x) of the baseline value.

### Pitfall 5: SLSA/reproducibility claims asserted without ever being independently re-verified
**What goes wrong:** the release pipeline "works" (green checkmarks) but no one has ever actually run `slsa-verifier` against a real published release, or diffed the hash of two independent builds of the same commit — so the claimed properties (SLSA L3, reproducibility) may not actually hold.
**Why it happens:** each individual tool (cosign, the SLSA generator, the reproducibility flags) is easy to "turn on" and demo in isolation; the properties that matter require end-to-end verification that isn't automatically enforced by any single tool's own success. `[CITED: .planning/research/PITFALLS.md Pitfall 9, this project's own prior research]`
**How to avoid:** the double-build hash-diff gate (linux/amd64, blocking per D-03) and an end-to-end `verifyRelease`-against-a-real-artifact test (per CONTEXT.md's own `<specifics>` and Finding 1's action item) must both actually run in CI, not just exist as documented intent.
**Warning signs:** a `docs/RELEASE.md` that documents verify commands but no CI job that has ever actually executed them against a real release asset.

## Code Examples

### Cross-platform `CC`/`CXX` selection (linux+windows via zig; darwin via native clang)
```yaml
# Source: pattern combining goreleaser/example-zig-cgo (linux/windows legs)
# [CITED: github.com/goreleaser/goreleaser-example-zig-cgo/.goreleaser.yaml]
# with the darwin leg run on a native macos-latest runner instead (Finding 2),
# so no CC/CXX override is needed there at all — the runner's own Xcode
# clang is already correct for both darwin arches.
builds:
  - id: codegraph-linux-amd64
    goos: [linux]
    goarch: [amd64]
    env:
      - CGO_ENABLED=1
    ldflags: &version_ldflags
      - -s -w
      - -X github.com/seanb4t/codegraph-go/internal/version.Version={{.Tag}}
      - -X github.com/seanb4t/codegraph-go/internal/version.Commit={{.FullCommit}}
      - -X github.com/seanb4t/codegraph-go/internal/version.Date={{.CommitDate}}
    flags:
      - -trimpath
    mod_timestamp: "{{ .CommitTimestamp }}"

  - id: codegraph-linux-arm64
    goos: [linux]
    goarch: [arm64]
    env:
      - CGO_ENABLED=1
      - CC=zig cc -target aarch64-linux-gnu
      - CXX=zig c++ -target aarch64-linux-gnu
    ldflags: *version_ldflags
    flags: [-trimpath]
    mod_timestamp: "{{ .CommitTimestamp }}"

  - id: codegraph-windows-amd64
    goos: [windows]
    goarch: [amd64]
    env:
      - CGO_ENABLED=1
      - CC=zig cc -target x86_64-windows-gnu
      - CXX=zig c++ -target x86_64-windows-gnu
    ldflags: *version_ldflags
    flags: [-trimpath]
    mod_timestamp: "{{ .CommitTimestamp }}"

  # windows/arm64 mirrors windows/amd64 with -target aarch64-windows-gnu

  # darwin/arm64 + darwin/amd64: run this SAME builds file (filtered by
  # --single-target) on macos-latest — no CC/CXX override needed, native
  # Xcode clang already handles both -arch arm64 and -arch x86_64.
```
**Version-ldflags note (verified against `internal/version/version.go`):** the three symbols above (`Version`, `Commit`, `Date`) are the *exact* fully-qualified paths the shipped `internal/version` package expects — confirmed from the package's own doc comment `[VERIFIED: read from source]`. `{{.Tag}}` (not `{{.Version}}`) should populate `Version` too, for consistency with the `v`-prefixed asset-naming requirement (Finding 1), though the package itself doesn't care about the `v` prefix specifically — `codegraph version` will simply display whatever string is injected.

### Raw-binary archive + per-binary signing (Finding 1's fix, full config)
```yaml
# Source: goreleaser.com/customization/package/archives.md (format: binary),
# goreleaser.com/customization/sign/sign.md (sign-blob --bundle pattern),
# goreleaser.com/blog/cosign-v3 (v3's --bundle flag). [CITED]
archives:
  - format: binary
    name_template: >-
      {{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}{{ if eq .Os "windows" }}.exe{{ end }}

checksum:
  name_template: "{{ .ProjectName }}_{{ .Tag }}_checksums.txt"
  algorithm: sha256

signs:
  - cmd: cosign
    signature: "${artifact}.sigstore.json"
    args:
      - sign-blob
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: binary   # per-binary, NOT checksum — Finding 1
```

### SLSA generic generator wiring over the checksums file
```yaml
# Source: goreleaser.com/blog/slsa-generation-for-your-artifacts (adapted).
# [CITED]
jobs:
  goreleaser:
    outputs:
      hashes: ${{ steps.hash.outputs.hashes }}
    steps:
      - id: goreleaser
        uses: goreleaser/goreleaser-action@<pinned-tag>
        with:
          args: release --clean
      - id: hash
        env:
          ARTIFACTS: "${{ steps.goreleaser.outputs.artifacts }}"
        run: |
          set -euo pipefail
          checksum_file=$(echo "$ARTIFACTS" | jq -r '.[] | select(.type=="Checksum") | .path')
          echo "hashes=$(cat "$checksum_file" | base64 -w0)" >> "$GITHUB_OUTPUT"

  provenance:
    needs: [goreleaser]
    permissions:
      actions: read
      id-token: write
      contents: write
    uses: slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@<exact-semver-tag>
    with:
      base64-subjects: "${{ needs.goreleaser.outputs.hashes }}"
      upload-assets: true
```
**Pin requirement:** the SLSA reusable workflow **must** be referenced by a full `@vX.Y.Z` semver tag — `slsa-verifier` explicitly rejects shorter tag references (`@v2`, `@v2.1`) when verifying provenance. `[CITED: slsa-framework/slsa-github-generator README]`

### Double-build reproducibility gate (linux/amd64, blocking)
```bash
# Source: pattern synthesis from golang-nuts discussion + goreleaser reproducible-builds
# blog + this project's own PITFALLS.md Pitfall 9 recommendation. [CITED/ASSUMED mix]
export SOURCE_DATE_EPOCH=$(git log -1 --format=%ct)
build() {
  CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -trimpath -ldflags "-s -w -buildid= -X .../version.Version=${TAG} -X .../version.Commit=${SHA} -X .../version.Date=${DATE}" \
    -o "$1" ./cmd/codegraph
}
build codegraph_build1
build codegraph_build2
sha256sum codegraph_build1 codegraph_build2
diff <(sha256sum codegraph_build1 | cut -d' ' -f1) <(sha256sum codegraph_build2 | cut -d' ' -f1) \
  || { echo "REPRODUCIBILITY GATE FAILED: linux/amd64 build is non-deterministic"; exit 1; }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| cosign v2 two-file signature output (`--output-certificate` + `--output-signature`) | cosign v3 single `--bundle=${signature}` unified `.sigstore.json` | cosign v3 release `[CITED: goreleaser.com/blog/cosign-v3]` | This project's `internal/upgrade` was written against the v3 bundle shape already — the release pipeline must match, not the older v2 two-file pattern still shown in some stale tutorials |
| `builder_go_slsa3.yml` (Go-specific SLSA builder, rebuilds the binary itself) | `generator_generic_slsa3.yml` (generic generator, attests over already-built artifacts) | Documented in SLSA project's own guidance for non-standard build setups | Required for this project specifically because the Go builder cannot accommodate `zig cc` CGo cross-compilation — CONTEXT.md D-02 already made this call correctly |

**Deprecated/outdated:**
- cosign v2's `COSIGN_EXPERIMENTAL=1` env var pattern for keyless signing — keyless is on by default in current cosign without needing this flag; `[CITED: multiple current cosign/GitHub Actions guides no longer reference this env var]`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `-tags netgo` fully resolves the darwin cross-compile DNS-resolver risk if the planner chooses the zig-cc-single-Linux-runner fallback instead of the native-matrix recommendation | Finding 2 | If wrong, a darwin release binary built this way could ship with broken DNS resolution, silently breaking `codegraph upgrade` and any future networked feature on macOS — high impact, low likelihood if the primary (native-matrix) recommendation is followed instead |
| A2 | Windows cross-compiled binaries do not carry the same cgo-DNS-resolver risk as darwin, because Go's `net` package uses its own resolver on Windows regardless of CGO_ENABLED | Finding 2 | If wrong, windows/amd64+arm64 release binaries could have the same class of DNS bug as darwin — moderate impact; verify with a real Windows-hosted network round trip test before first release, not just a build/link check |
| A3 | The `goreleaser-cross` Docker image has a tag with Go version compatible with this project's `go 1.26.5` directive | Finding 2 (fallback path) | If wrong, the fallback recipe simply doesn't work as described — low risk since it's explicitly not the primary recommendation, but the planner must verify tag availability before committing to this path |
| A4 | `syft <binary> -o spdx-json` can catalog a statically-linked CGo Go binary meaningfully (vs. scanning source/`go.mod` directly) | Package Legitimacy Audit / Code Examples | If SBOM quality from binary-scanning is poor, fall back to `sboms: - artifacts: archive` (source-based cataloging, GoReleaser's documented default) — low risk, easy fallback exists |
| A5 | The synthetic 100k+-file corpus generator design (seeded RNG, cross-referencing symbols, ~120k files) is sufficient to make INDX-06's "100k+ file monorepo" requirement and PERF-02's regression gate both meaningfully exercised | Architecture Patterns Pattern 2 | If the synthetic corpus under-represents real-world file/symbol distribution, the regression gate could pass/fail on patterns that don't predict real-repo behavior — mitigated by D-04's own explicit design (a real large monorepo MAY additionally validate the headline INDX-06 number outside the blocking gate) |

## Open Questions

1. **Does the project have (or intend to purchase) a GoReleaser Pro license?**
   - What we know: OSS GoReleaser v2.17.0 is installed locally; no license artifact found anywhere in the repo or `.planning/`.
   - What's unclear: whether a future budget decision could change this, which would reopen the simpler Pro `--split`/`continue --merge` path as viable.
   - Recommendation: plan and build against OSS-only (Finding 2's native-matrix recipe) now; if a Pro license is acquired later, the merge step can be swapped for GoReleaser's native one without touching the build-matrix or signing logic.

2. **Exact current pinned tags** for `goreleaser-action`, `cosign-installer`, and `generator_generic_slsa3.yml` were seen in web sources as `@v7`, `@v3`/`@v3.1.2`, and a version around `@v1.9.0`–`@v2.1.0` respectively, but **not independently pinned/verified against the live GitHub Actions Marketplace in this session**.
   - What we know: these are the tags current official examples reference as of this research.
   - What's unclear: the single, current, most-recent stable tag for each, at the moment the plan is executed (these move independently of this research date).
   - Recommendation: the planner/execution phase must re-resolve the exact current tag for each action immediately before writing the workflow YAML — do not hardcode the tags found in this document without a final freshness check.

3. **Whether `-tags netgo` is a viable mitigation for the darwin zig-cross DNS risk (Assumption A1), should the native-matrix recommendation ever need to be revisited (e.g. GitHub-hosted `macos-latest` runner cost/availability becomes a constraint).**
   - What we know: `netgo` forces the pure-Go resolver independent of other CGo code in the binary, in general.
   - What's unclear: whether it fully compiles/links cleanly in a `zig cc`-cross-linked darwin target that also links tree-sitter's CGo grammars, and whether it's sufficient (vs. also needing supplementary SDK pieces for non-network CGo linking to succeed at all).
   - Recommendation: not needed for the primary recommendation; only worth a Wave-0 spike if `macos-latest` runner minutes become a real cost/availability constraint later.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| `goreleaser` | Build orchestration | ✓ | v2.17.0 | — |
| `cosign` | Keyless signing | ✓ | unqueried this session — verify v3.x at execution | — |
| `govulncheck` | Vuln gate | ✓ | present at `/Users/sean/go/bin/govulncheck` | — |
| `zig` | linux/arm64 + windows cross-compile | ✗ | — | Install in CI via `goreleaser/setup-zig` or direct download — not needed on local dev machine |
| `syft` | SBOM generation | ✗ | — | Install in CI via `anchore/sbom-action/download-syft` |
| Installed TS `codegraph@1.3.1` | PERF-01 head-to-head reference | ✓ | 1.3.1, at `/opt/homebrew/bin/codegraph` | Benchmark harness must shell out to this exact installed binary for the published head-to-head numbers; CI runners will need TS CodeGraph installed separately (`npm install -g @colbymchenry/codegraph@1.3.1`) for the on-demand `bench.yml` workflow |
| `macos-latest` GitHub-hosted runner (Apple Silicon) | Native darwin build (Finding 2) | assumed ✓ (standard GitHub Actions offering) `[ASSUMED — not independently confirmed against the live GitHub Actions runner catalog this session]` | — | If unavailable/cost-prohibitive, fall back to the `goreleaser-cross` Docker-image recipe (documented fallback, higher risk — Finding 2) |

**Missing dependencies with no fallback:** none — every missing local tool (`zig`, `syft`) is CI-only by design and has a documented install path.

**Missing dependencies with fallback:** `zig` and `syft` (documented CI install steps above).

## Validation Architecture

### Test Framework
| Property | Value |
|---|---|
| Framework | Go's built-in `testing` package (`go test`) — no third-party test framework in this repo `[VERIFIED: repo-wide grep, 104 `func Test` matches, no pytest/jest-equivalent config found]` |
| Config file | none — plain `go test ./...`; a documented pre-existing flake exists in `internal/daemon` (`TestSoak`) under full-suite parallel load per `.planning/STATE.md`'s own carried-forward note |
| Quick run command | `go test ./internal/bench/... -run TestSmoke` (new, this phase) |
| Full suite command | `go test ./... ` (isolate `internal/daemon` with `-count=1` if flaky, per STATE.md's documented workaround) plus `go test ./internal/bench/... -bench=. -benchmem` for the perf gate |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| DIST-01 | Single static binary per platform, no bundled runtime, no install-time compilation | smoke (CI workflow) | CI job per matrix target runs `file <binary>` + `./codegraph --version` on its own native/cross-produced artifact | ❌ Wave 0 — new CI workflow |
| DIST-02 | Every artifact cosign-signed + SLSA provenance, user-verifiable | integration (end-to-end, real artifact) | a real (or CI-simulated) signed binary + `.sigstore.json` run through `verifyRelease` — the exact test CONTEXT.md's `<specifics>` calls for | ❌ Wave 0 — new test in `internal/upgrade` package, e.g. `verify_release_e2e_test.go` |
| DIST-03 | SBOM published; govulncheck + dep scan gate CI | smoke (CI workflow) | `govulncheck ./...` (blocking) + confirm `*.spdx.json` uploaded as a release asset | ❌ Wave 0 — new CI step |
| DIST-04 | Reproducible builds, double-build gate | integration (CI script) | the double-build hash-diff script (Code Examples), blocking on linux/amd64 | ❌ Wave 0 — new CI script |
| PERF-01 | Published head-to-head benchmarks, median-of-N, raw numbers | manual-only (published artifact, not a pass/fail gate) | `go run ./tools/bench/runner -mode headtohead` producing `docs/BENCHMARKS.md` numbers | ❌ Wave 0 — new tool |
| PERF-02 | CI regression gate against a corpus including 100k+ files | integration (CI gate) | `go run ./tools/bench/runner -mode regression` against committed `baseline.json` | ❌ Wave 0 — new tool + baseline |
| INDX-06 | Index 100k+ file monorepo within bounded memory, peak RSS tracked in CI | integration (CI gate, shared with PERF-02) | same regression-gate run also asserts peak RSS against a bounded-memory ceiling, not just a relative regression | ❌ Wave 0 — same tool, additional assertion |

### Sampling Rate
- **Per task commit:** `go test ./internal/bench/... -run TestSmoke` (fast unit-level sanity of the RSS-capture/normalization logic, Pattern 1's pitfall — this is exactly the kind of pure logic bug that should never reach CI)
- **Per wave merge:** full `go test ./...` + `govulncheck ./...`
- **Phase gate:** full suite green + a real double-build hash-diff run + (at least once, manually or via a tag-triggered dry run) an actual `verifyRelease` pass against a real signed release artifact, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/bench/rss_test.go` — unit tests for the Linux-KB-vs-Darwin-bytes normalization logic (Pattern 1/Pitfall 4) — this is cheap, pure-function-testable, and directly guards against the single most likely silent-corruption bug in this whole phase
- [ ] `internal/upgrade/verify_release_e2e_test.go` (or equivalent) — the real-signed-artifact-against-`verifyRelease` test CONTEXT.md's `<specifics>` explicitly calls for
- [ ] `tools/bench/gencorpus/` — the synthetic corpus generator itself, plus a test asserting deterministic output (same seed → same file count/hash)
- [ ] `tools/bench/baseline.json` — initial committed baseline, established by a first real run before the gate can be blocking
- [ ] `.github/workflows/{release,ci,bench}.yml` — none exist yet; this phase creates all three from scratch (`[VERIFIED: no .github/workflows/ directory exists in the repo yet]`)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | no | n/a — this phase has no user-facing auth surface |
| V3 Session Management | no | n/a |
| V4 Access Control | no | n/a |
| V5 Input Validation | yes (narrow) | benchmark harness shells out to two external binaries with controlled, non-attacker-supplied arguments (fixed pinned repos, generated corpus paths) — no untrusted external input crosses this phase's own code |
| V6 Cryptography | yes | **never hand-roll** — cosign v3 (Sigstore ecosystem) owns all signing/verification cryptography; `internal/upgrade`'s existing `sigstore-go`-based verifier is the sole consumer and must not be reimplemented or bypassed by this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Release artifact tampering (a downloaded binary is not what was actually built/signed) | Tampering | cosign v3 keyless signature + SLSA provenance, verified end-to-end (Finding 1, DIST-02) — this is precisely the property the whole phase exists to guarantee |
| Supply-chain compromise via a malicious CI Action version | Tampering / Elevation of Privilege | Pin every third-party GitHub Action to a full commit SHA or exact semver tag (not a floating major like `@v3`) — especially load-bearing for `slsa-github-generator`, which explicitly requires this for its own verifier to trust the provenance |
| Overly-broad OIDC trust scope (a signature from any workflow in the repo being accepted, not just the release workflow) | Elevation of Privilege | `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` is already a full-match, anchored regex scoped to exactly `release.yml` on `refs/tags/v*` — this phase must not weaken that pattern; if the workflow filename ever changes, the pattern must change in lockstep (already documented in the source's own comments) |
| Release-time secret/credential leakage (e.g. a long-lived signing key checked into CI config) | Information Disclosure | keyless signing via GitHub OIDC (`id-token: write`) — no long-lived key exists to leak, by design |

## Sources

### Primary (HIGH confidence)
- `internal/upgrade/verify.go`, `internal/upgrade/upgrade.go`, `internal/upgrade/release.go`, `internal/version/version.go` (this repo, read directly this session) — the release-identity and asset-naming contract; Finding 1 and Finding 2's version-ldflags note are both sourced directly from this code, not inferred
- `/goreleaser/goreleaser` via Context7 — builds/env/CC-CXX config, sbom block, signs block, archives `format: binary`, checksum config, partial/split-merge docs
- `goreleaser.com/blog/cosign-v3` (web, official GoReleaser blog) — cosign v3 `--bundle` flag change
- `goreleaser.com/blog/slsa-generation-for-your-artifacts` (web, official) — SLSA generic generator wiring recipe
- `goreleaser.com/customization/general/partial.md` (web, official, via Context7) — confirms split/merge is GoReleaser Pro
- `github.com/goreleaser/goreleaser-example-zig-cgo/.goreleaser.yaml` (web, official example repo) — the CC/CXX-per-platform zig-cc recipe, and the darwin-uses-native-clang detail
- `slsa-framework/slsa-github-generator` README (web, official) — full-semver-tag pinning requirement

### Secondary (MEDIUM confidence)
- `johncodes.com/archive/2026/02-11-cross-compiling-cgo` (web, independent blog, cross-checked against the official example repo's own darwin-uses-clang-not-zig choice — the two sources are consistent with each other, which raises confidence) — the darwin osxcross/DNS-resolver risk (Finding 2)
- `blog.afoolishmanifesto.com/posts/golang-zig-cross-compilation` (web, independent blog) — corroborates darwin cross-compile as the outlier weak point in an otherwise-solid zig-cc story
- `nodejs/node` GitHub issue #44332 (web, primary-source bug tracker, not a blog) — the Linux-KB-vs-macOS-bytes `ru_maxrss` unit split (Pitfall 4/Pattern 1)
- `.planning/research/PITFALLS.md`, `.planning/research/STACK.md` (this project's own prior-phase research, produced 2026-07-10) — SLSA L2-vs-L3 distinction, reproducibility gaps, CGo cross-compile warning signs

### Tertiary (LOW confidence)
- `goreleaser-cross` image go-version-tag freshness for `go1.26.5` — not independently verified this session, flagged `[SUS]` in the Package Legitimacy Audit and treated as a documented fallback only, not the primary recommendation

## Metadata

**Confidence breakdown:**
- Standard stack / release-identity contract: HIGH — verified directly against shipped source code, not inferred
- Cosign/GoReleaser config schema: HIGH — Context7-sourced official docs, cross-checked against multiple official blog posts
- Darwin cross-compile risk assessment: MEDIUM — multiple independent web sources corroborate each other, but nothing was independently executed/reproduced in this session (no `zig`/`osxcross` installed locally to test)
- Benchmark harness design (RSS normalization, synthetic corpus): MEDIUM-HIGH for the RSS unit pitfall (primary-source GitHub issue), MEDIUM for the synthetic-corpus generator design (reasoned from project convention, not externally sourced)
- Pitfalls: HIGH — largely inherited from this project's own prior, dedicated PITFALLS.md research plus this session's own source-code reading

**Research date:** 2026-07-13
**Valid until:** 14 days for the exact Action version tags (Open Question 2 — these move fast and must be re-resolved at execution time); 60 days for the architectural findings (release-identity contract, darwin cross-compile risk, GoReleaser Pro boundary) which are unlikely to change quickly
