# Phase 2: Apple Signing & Notarization - Research

**Researched:** 2026-08-09
**Domain:** GoReleaser native macOS notarization (quill-backed), Apple Gatekeeper verification tooling, CI credential handling
**Confidence:** HIGH for GoReleaser/quill pipeline mechanics (read directly from the pinned v2.17.1 / quill source in this session); MEDIUM for Apple-side tooling semantics (`spctl`, `syspolicy_check`, quarantine xattr) which could not be executed in this session (no macOS host available); LOW/explicitly-flagged for anything depending on the maintainer's actual Apple Developer account state.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Use GoReleaser's native `notarize:` block (quill-backed) rather than hand-rolled `codesign` + `xcrun notarytool` post-hooks. `github.com/goreleaser/quill` is already an indirect dependency at `go.tool.mod:236`.
- **D-02:** Verification is Apple-native regardless of D-01. Quill's own success report is never the oracle. SIGN-02 mandates `spctl -a -vv -t exec` reporting `source=Notarized Developer ID` plus `syspolicy_check distribution` passing; `codesign -dvv` is explicitly recorded as insufficient.
- **D-03 (OPEN RESEARCH QUESTION, not locked):** Hardened runtime is required for notarization. The entitlement set is NOT a locked decision — research must establish what quill supports and what a CGo-linked Go binary actually requires. Working hypothesis to confirm/refute: tree-sitter's C compiles at *build* time, so no JIT/unsigned-executable-memory entitlement should be needed.
- **D-04:** Notarize the raw Mach-O binaries; place the notarize pipe **before** archive/checksum/binary_signs/sboms so `.zip` contents, checksums, cosign signature, and SLSA attestation all describe post-notarization bytes by construction. Rejected: notarizing the `.zip` only (upgrade path consumes the raw binary) or notarizing both shapes (redundant Apple round-trip).
- **D-05:** SIGN-04's byte-identity claim is proven by diffing sha256 at four points: immediately after notarize, on the re-downloaded published asset, on the cosign-signed subject, and on the SLSA-attested subject. Re-downloading is mandatory.
- **D-06:** SIGN-03's RED baseline is the published `v0.5.1` darwin assets (not `v0.5.0`, which has zero assets). RED must be recorded with `com.apple.quarantine` confirmed present via `xattr -p` *before* the `spctl` run.
- **D-07:** SIGN-04's ordering claim is proven by a one-time recorded mutation (deliberately mis-order, record divergent sha256, revert) — no permanent regression test required.
- **D-08:** The notarize pipe is rehearsed locally on the maintainer's Mac via a guarded, committed Taskfile target, before any tag push. CI rehearsal via `workflow_dispatch` was considered and declined.
- **D-09:** The rehearsal target must hard-fail by name when the Developer ID certificate or App Store Connect API key is absent — not after a network round-trip to Apple.
- **D-10:** Criterion 4's full CLI + MCP integration suite runs against the notarized binary itself (re-downloaded published asset), inside the existing post-release-verify workflow.
- **D-11:** Post-release-verify extensions must preserve the event-aware guard and tolerate a zero-asset release entry (`v0.5.0`).
- **D-12 through D-17 (carried forward, do not re-litigate):** patch-forward recovery; release-please is sole tag authority; exactly one job holds `id-token: write`; raw binary stays primary asset; stapling out of scope; `id-token: write`/signing secrets must never reach a `pull_request`-triggerable workflow.

### Claude's Discretion

- The concrete shape of the CI secrets (base64 P12 + password vs. keychain import; App Store Connect `issuer_id`/`key_id`/`key` naming and encoding) — mechanical, follow GoReleaser's `notarize.macos` schema.
- Taskfile target naming, consistent with existing `check:*` / `verify:*` / `release:*` conventions.
- How the recorded SIGN-04 mutation evidence is formatted in the phase artifacts.
- Whether the `docs/RELEASE.md` reproduce-it-yourself commands live in a new section or extend §1.

### Deferred Ideas (OUT OF SCOPE)

- CI rehearsal of the notarize pipe (`workflow_dispatch`, main-branch-only) — declined in favor of local rehearsal (D-08).
- A permanent property-based regression test pinning the notarize pipe's position — declined in favor of one-time recorded mutation (D-07).
- A credential-free contributor dry mode for the rehearsal target — declined, no current consumer.
- DIST-06 — stapled, offline-safe first launch — out of scope per REQUIREMENTS.md.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SIGN-01 | Darwin binaries are Developer ID codesigned and notarized during release, cert + ASC API key as CI secrets | GoReleaser `notarize:` schema fully mapped (Standard Stack, Code Examples); credential shape and env-var wiring confirmed from quill source (no keychain step needed) |
| SIGN-02 | Browser-downloaded, Gatekeeper-blocked-by-default asset passes: `spctl -a -vv -t exec` reports `source=Notarized Developer ID`, `syspolicy_check distribution` passes | Verification tooling researched (Common Pitfalls, Open Questions — execution semantics need live-runner confirmation) |
| SIGN-03 | Gate demonstrated RED first against `v0.5.1`'s un-notarized darwin assets, with quarantine xattr confirmed present | RED-baseline mechanics and xattr format researched; `pipe.Skip` silent-skip pitfall documented as a NEW RED-adjacent risk (notarize pipe can no-op without failing) |
| SIGN-04 | cosign-signed / SLSA-attested / published bytes are byte-identical; ordering must be measured, not assumed | **Headline finding:** current `binary_signs:` runs BEFORE `notarize:` in GoReleaser's hardcoded pipeline — the opposite of what D-04 needs. Concrete fix and verification path documented (Architecture Patterns, Common Pitfalls #1) |
</phase_requirements>

## Summary

GoReleaser 2.17.1 (this repo's pinned tool version, `go.tool.mod:38,234`) exposes native macOS notarization via a `notarize:` top-level block whose `macos` variant is quill-backed, open-source (not GoReleaser Pro — only the `macos_native`/DMG/PKG variant requires Pro), and requires no separate keychain setup: it signs and notarizes entirely in-process via Go library calls into `github.com/goreleaser/quill` (already an indirect dependency, `go.tool.mod:236`), never shelling out to `codesign`/`security`/`xcrun`.

**The single most consequential finding of this research, read directly from the pinned `goreleaser/v2@v2.17.1` source (`internal/pipeline/pipeline.go:63-106`):** GoReleaser's pipe execution order is a **hardcoded Go slice, not driven by `.goreleaser.yaml`'s block order**. `notary.MacOS{}` (the notarize pipe) is registered inside `BuildPipeline`, immediately **after** `sign.BinaryPipe{}` (`binary_signs:`). This means, as this repo's `.goreleaser.yaml` is configured today, **cosign signs the pre-notarization binary bytes, and notarization then mutates those bytes afterward** (quill's `Sign()` patches an `LC_CODE_SIGNATURE` load command directly into the Mach-O) — the exact opposite of what D-04/SIGN-04 requires. `archive.Pipe{}`, `sbom.Pipe{}`, and `checksums.Pipe{}` are all appended *after* `BuildPipeline` in the full `Pipeline` slice, so they automatically see post-notarization bytes with no config change needed — cosign (via `binary_signs:`) is the **only** pipe in the current `.goreleaser.yaml` that sits on the wrong side of notarize, and YAML reordering cannot fix it. The concrete, source-grounded fix — switching cosign from the build-scoped `binary_signs:` pipe to the release-scoped `signs:` pipe with `artifacts: binary` (which is registered well after `BuildPipeline` in the Go slice, i.e., after notarize) — is documented below with the exact filter mechanics and a re-confirmation that the same `${artifact}` Path-vs-Name naming hazard Phase 1 already solved for `binary_signs:` applies identically and requires the identical fix.

On the D-03 entitlements question: quill's own signing code (`quill/sign/signing_super_blob.go:19-28`) **always** sets the hardened-runtime code-directory flag (`macho.Runtime`, `0x00010000`) whenever signing with a real (non-adhoc) identity — this is unconditional, not gated by any entitlements config, and satisfies Apple's hardened-runtime-for-notarization requirement automatically. Entitlements are **strictly opt-in**: `generateEntitlements` (`quill/sign/entitlements.go:13-16`) returns `nil` — embeds nothing — when `entitlementsXML == ""`, and GoReleaser's `notarize.macos[].sign.entitlements` field defaults to unset. This directly supports the maintainer's D-03 hypothesis: an empty entitlement set is not merely plausible, it is what happens by construction if the `entitlements:` key is simply omitted. This repo's own codebase has no `plugin` package usage and no `dlopen`/`dlsym` calls (`rg` confirmed zero matches), and all tree-sitter grammar C code compiles into the binary at build time via cgo — there is no runtime dynamic-library-loading or JIT-memory pattern that would trip hardened-runtime restrictions (`disable-library-validation`, `allow-jit`, `allow-unsigned-executable-memory`). This is strong supporting evidence, not proof — proof is exactly what Criterion 4 (running the real test suite against the real notarized binary) exists to produce.

Neither `test/integration` nor `test/wireoracle`'s `TestMain` currently has a seam to point at an already-built/downloaded binary — both hardcode `go build -o binPath ./cmd/codegraph` (`test/integration/main_test.go:39-57`, `test/wireoracle/main_test.go` identical pattern). This must be built new, but a directly-transplantable precedent already exists in this repo: `internal/upgrade/verify_release_e2e_test.go:97-105`'s `CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE` env-var-override pattern.

**Primary recommendation:** Adopt `notarize.macos` as researched, but treat the `binary_signs:`-vs-`notarize:` pipeline-order conflict as a must-fix design decision before writing the plan — not an implementation detail — because it directly determines whether SIGN-04 is even achievable, and the fix (moving cosign to `signs:`) reopens Phase 1's D-14 in a way the planner and maintainer should decide on explicitly rather than discover mid-execution.

---

## D-18 — MAINTAINER RULING (2026-08-09, at plan time): cosign moves to `signs:`

**Status: LOCKED.** This ruling resolves the headline finding above and supersedes the *mechanism* of Phase 1's D-14 (not its asset-name contract). Assumption **A4 is CONFIRMED**; the planner must treat the items below as settled and must not re-litigate them.

**Two corrections to the framing above — apply these when reading the rest of this document.**

1. **GoReleaser is not wrong, and the ordering is not a bug.** The commit that introduced `binary_signs:` ([`a8916c0`](https://github.com/goreleaser/goreleaser/commit/a8916c080ea52afbb8bdd31404ae8de637fc247a), PR #5018) *explicitly* inserted `sign.BinaryPipe{}` above `notary.MacOS{}`, and `main` still has that order. GoReleaser's own docs state the model plainly (`Create Binaries → Sign Binaries → Notarize Binaries`; "the signed binaries are then used from that point forward") and document `binary_signs:` as a build-phase pipe. Nothing upstream is broken. **No upstream issue exists in either direction** — the `binary_signs:` + `notarize:` combination is simply never shown together in GoReleaser's docs and appears unexercised. Wherever this document says "WRONG ORDER" or implies a GoReleaser defect, read it instead as: *this repo's configuration would be asking GoReleaser for the wrong thing.*

2. **The mutation comes from the signing half, not the notarizing half.** `notary.MacOS` is a **sign-and-notarize** pipe — its own `String()` is `"sign & notarize macOS binaries"`. `macos.go` calls `quill.Sign(*signCfg)` against `bin.Path`, embedding `LC_CODE_SIGNATURE` in place. Apple's notarization *submission* mutates nothing (and stapling is out of scope, DIST-06). So the precise claim is: a `binary_signs:` sidecar describes bytes that quill's **signing** step then rewrites.

**The ruling — rename one YAML key.** `.goreleaser.yaml:223-231`:

```diff
-binary_signs:
-  - cmd: cosign
+signs:
+  - cmd: cosign
+    ids: [raw]
+    artifacts: binary
     signature: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"
     args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]
-    artifacts: binary
```

**Why this is byte-safe [VERIFIED at source this session]:** both pipes share `signone()` in `internal/pipe/sign/sign.go`. Line 182 binds `env["artifact"] = art.Path` for the signing pass. For `formats: [binary]`, `archive.Pipe`'s `skip()` branch (`internal/pipe/archive/archive.go:296-310`) creates the `UploadableBinary` with **`Path: binary.Path`** — the exact same file quill mutated in place — while carrying `Goos`/`Goarch` through (so `.Os`/`.Arch` templating still resolves per-platform) and tagging `ExtraID: archive.ID` (= `raw`). Therefore: same file signed, sidecar name unchanged character-for-character, and `internal/upgrade/upgrade.go:195`'s `assetName + ".sigstore.json"` download contract is untouched. **The `signature:` template must stay exactly as written** — line 275 rebinds `env["artifact"] = art.Name` for the publish-naming pass, so the Path-vs-Name hazard is still live and the explicit Go-template is still the mitigation.

**A4 CONFIRMED — Phase 1's recorded rationale for rejecting `signs:` is false at source level.** `.goreleaser.yaml:183-186` claims `signs:`'s `artifacts: binary` mode "requires `archives.formats: binary` project-wide". It does not: `sign.go:114` is `artifact.ByType(artifact.UploadableBinary)`, a per-artifact-type filter, and `archive.Pipe` emits one `UploadableBinary` **per binary per archive entry**. The coexisting `zip` entry emits `UploadableArchive` and suppresses nothing. `signs:` also accepts `ids:` (`sign.go:125`), so `ids: [raw]` scopes it explicitly. The false belief traces to GoReleaser's docs phrasing — *"binary: binaries (only when `archives.format` is 'binary', use binary_signs otherwise)"* — which reads as a project-wide precondition but is per-entry in the code. **This is the fifth instance of this repo's recurring "following upstream docs faithfully was the trap" failure class.** The plan MUST rewrite that `.goreleaser.yaml` comment block; leaving a now-known-false rationale in a load-bearing config comment is how the next maintainer re-derives the wrong conclusion.

**Consequences the planner must carry:**

- **Side benefit, in scope to at least evaluate:** `sign.Pipe` is **not** in `BuildCmdPipeline`, so `goreleaser build` stops requiring a cosign keyless OIDC token. That is precisely the breakage that forced unwiring `check:darwin-release-build` from the darwin-toolchain-canary in Phase 1 (commit `008d51c`, memory `e2cnrbt6ph`). Note that `notary.MacOS` **is** in `BuildPipeline`, so the `enabled:` guard (Pitfall 3) is what keeps unprivileged `goreleaser build` callers from reaching Apple — this is exactly how GoReleaser's own repo guards it. Re-wiring the canary is a *candidate*, not a mandate; if the plan defers it, say so explicitly rather than silently.
- **D-04's ordering claim narrows and gets easier.** `archives:` (step 26), `sboms:` (33) and `checksum:` (34) already run after `notary.MacOS` (22) with no config change. After this rename, `signs:` lands at step 35. So SIGN-04 criterion 3 is satisfiable **by construction** once the rename is made — but D-05/D-07 still require it to be *measured*, not trusted.
- **D-07's one-time recorded mutation now has an obvious, cheap subject.** The mis-ordering proof SIGN-04 criterion 3 demands is simply: configure cosign under `binary_signs:` (the pre-ruling state), record the divergent sha256s, then revert to `signs:`. The RED evidence and the fix are the same experiment.
- **`TestBinarySignsSidecarMatchesUpgradeContract` must move deliberately, not be deleted.** Per memory `wewn0wp1n1`: a test pinning a broken invariant resists correction worse than no test. Retarget it at the `signs:` block, keep it asserting the *property* (four distinct resolved names, each matching `internal/upgrade`'s download contract), never a literal string.
- **Still unverified, and the plan must not assume it:** that `quill.Sign` changes the sha256 is inferred from its purpose and from criterion 1's existing measurement (`adhoc, linker-signed` today), **not** from a before/after Mach-O diff — that needs real macOS hardware and a Developer ID cert. Assumption **A3** (no later release-scoped pipe re-mutates the binary after `sign.Pipe`) likewise stands unproven past `checksums.Pipe{}`/`sign.Pipe{}` and must be settled empirically by the rehearsal, not by further source-reading.

## Architectural Responsibility Map

This project is a CLI/build-pipeline, not a web app; the standard Browser/SSR/API/CDN/DB tiers do not apply cleanly. Adapted to this phase's actual layers:

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Developer ID code signing | GoReleaser build pipeline (`notarize.macos[].sign`, quill in-process) | — | Runs inside `goreleaser release`'s build phase, mutates the Mach-O directly; no shell-out |
| Apple notarization submission | GoReleaser build pipeline (`notarize.macos[].notarize`, quill in-process, App Store Connect API) | External: Apple's notary service | Network round-trip to Apple; `wait: true` blocks the pipe until Apple responds |
| Cosign detached signature | GoReleaser release pipeline (`signs:`/`binary_signs:`, cosign subprocess) | CI: `id-token: write` OIDC | **Must run AFTER notarize** — currently does not (headline finding) |
| SLSA build provenance | GitHub Actions (`actions/attest-build-provenance`, separate step after `task release:goreleaser` completes) | GitHub Attestations API | Runs after ALL GoReleaser pipes finish; unaffected by the ordering bug |
| Gatekeeper verification gate | Client OS (`spctl`, `syspolicyd`) on the macOS runner | CI: `post-release-verify.yml` | Apple-native, never GoReleaser/quill's own success report (D-02) |
| CI secrets | GitHub Actions secrets → single `id-token: write` job's `env:` | Local: maintainer's shell env for rehearsal | Must never reach a `pull_request`-triggerable workflow (D-17) |
| Test-suite-against-real-binary | `test/integration` (+ optionally `test/wireoracle`) `TestMain` | `post-release-verify.yml` self-upgrade-style download step | New seam required — does not exist today |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/goreleaser/goreleaser/v2` | v2.17.1 (pinned, `go.tool.mod:234`) [VERIFIED: go.tool.mod:234 — `github.com/goreleaser/goreleaser/v2 v2.17.1 // indirect`] | Release pipeline, includes native `notarize:` block | Already this project's release tool (Phase 1); `notarize.macos` ships in OSS GoReleaser (not Pro) |
| `github.com/goreleaser/quill` | `v0.0.0-20260630015114-8310f3e9a321` (indirect, pulled in by goreleaser/v2) [VERIFIED: go.tool.mod:236 — `github.com/goreleaser/quill v0.0.0-20260630015114-8310f3e9a321 // indirect`] | In-process macOS code signing + notarization (no `codesign`/`xcrun` shell-out) | Already an indirect dependency (D-01); no new package added by this phase |

No new external packages are introduced by this phase — see **Package Legitimacy Audit** below.

**Version verification:** confirmed by direct `go.tool.mod` inspection this session (not `npm view`/`pip`-style registry lookup, since these are Go modules already vendored in the local module cache at `$(go env GOMODCACHE)/github.com/goreleaser/{goreleaser/v2@v2.17.1,quill@v0.0.0-20260630015114-8310f3e9a321}`, confirmed present via `go mod download -modfile=go.tool.mod`).

### Supporting

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `spctl` (macOS built-in, `/usr/sbin/spctl`) | Gatekeeper assessment CLI | SIGN-02/SIGN-03's primary oracle — `spctl -a -vv -t exec <path>` |
| `syspolicy_check` (macOS built-in) | Newer syspolicyd-backed distribution-readiness check | SIGN-02's secondary oracle — `syspolicy_check distribution <path>` (exact CLI syntax/output NOT verified this session — see Open Questions) |
| `xattr` (macOS built-in, `/usr/bin/xattr`) | Read/write extended attributes, incl. `com.apple.quarantine` | D-06/D-09's RED-baseline setup and confirmation |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `notarize.macos` (quill-backed, OSS) | `notarize.macos_native` (Pro, `codesign`+`security`+`xcrun notarytool`, DMG/PKG only) | Pro-only; also only supports `.dmg`/`.pkg` artifact shapes, not the bare-binary shape this project ships (D-15/D-16). Not viable — already excluded by the project's GoReleaser-Pro-as-costed-fallback stance (PROJECT.md) |
| Hand-rolled `codesign`/`xcrun notarytool` shell steps | `notarize.macos` | Rejected by D-01: puts signing in unrehearsable shell glue outside GoReleaser's artifact bookkeeping — same class of risk that shipped v0.5.0 empty |

**Installation:** no new `go install`/`npm install` step — `notarize:` is config-only; `quill` builds into the already-pinned `goreleaser` tool binary.

## Package Legitimacy Audit

**No new external packages are installed by this phase.** `github.com/goreleaser/quill` is already an indirect dependency of `github.com/goreleaser/goreleaser/v2` at `go.tool.mod:236` [VERIFIED: go.tool.mod:236], confirmed present in the local Go module cache this session via `go mod download -modfile=go.tool.mod`. The Package Legitimacy Gate protocol is not applicable — no `npm view`/`pip index`/`cargo search` invocation is needed because no new dependency crosses into `go.mod`/`go.tool.mod`.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|--------------|---------|-------------|
| `github.com/goreleaser/quill` | Go modules (proxy.golang.org) | Pre-existing indirect dep (already pinned before this phase) | N/A (Go modules have no download-count metric) | github.com/goreleaser/quill | OK (pre-approved by Phase 1's D-01 acceptance) | No action — already vendored |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
                 goreleaser release --clean  (task release:goreleaser, release.yml)
                              │
                              ▼
                 ┌─── BuildPipeline (per platform) ───┐
                 │  build.Pipe{}                       │
                 │  universalbinary / upx (n/a here)   │
                 │  sign.BinaryPipe{}  ← binary_signs:  │  ★ cosign signs HERE today (PRE-notarize)
                 │  notary.MacOS{}     ← notarize:      │  ★ quill codesigns + submits to Apple HERE,
                 │                                      │    MUTATES the Mach-O in place
                 └──────────────────┬───────────────────┘
                                    │  (same on-disk Path, refreshed artifact metadata)
                                    ▼
                 archive.Pipe{}  →  sbom.Pipe{}  →  checksums.Pipe{}  →  sign.Pipe{}  ← signs:
                 (raw + zip)        (per-binary)     (8-payload file)     (release-scoped,
                                                                           runs AFTER notarize
                                                                           automatically)
                                    │
                                    ▼
                              publish.New()  →  GitHub Release assets
                                    │
                              (separate GH Actions step, AFTER task release:goreleaser returns)
                                    ▼
                    actions/attest-build-provenance  →  GitHub Attestations API
                                                          (SLSA — reads the FINAL checksums file,
                                                           always post-notarize, unaffected by the
                                                           binary_signs:/notarize: ordering bug)

                 ── separately, post-publish ──
                 post-release-verify.yml (workflow_run, D-11 event-aware guard)
                   → resolve-tag
                   → verify-supply-chain (cosign verify-blob + gh attestation verify, re-downloaded)
                   → self-upgrade (darwin/arm64 + linux/amd64 matrix)
                   → [NEW, this phase] gatekeeper verification (spctl + syspolicy_check, re-downloaded,
                       quarantine xattr applied before assessment)
                   → [NEW, this phase] CLI+MCP integration suite against the re-downloaded notarized
                       binary (requires a new TestMain env-var seam)
```

### Recommended Project Structure

No new packages/directories — this phase modifies existing config and adds test/CI seams:

```
.goreleaser.yaml         # gains notarize: block; MUST also resolve the binary_signs:-vs-notarize:
                          # ordering conflict (see Pitfall #1) — likely means moving cosign from
                          # binary_signs: to signs: {artifacts: binary}
.github/workflows/
  release.yml             # gains 5 Apple secrets in the Release step's env:
  post-release-verify.yml # gains a gatekeeper-verification job + (D-10) an integration-suite job
Taskfile.yml              # gains a guarded notarize-rehearsal target (D-08/D-09) and a
                          # verify:gatekeeper-style target callable from post-release-verify.yml
test/integration/
  main_test.go             # TestMain gains a CODEGRAPH_TEST_BIN-style override, mirroring
                          # internal/upgrade/verify_release_e2e_test.go's CODEGRAPH_E2E_BINARY pattern
docs/RELEASE.md            # gains criterion-5 guarantee statement + xattr/spctl reproduction commands
```

### Pattern 1: `notarize.macos` config shape

**What:** GoReleaser's native, quill-backed macOS sign+notarize block.
**When to use:** Every darwin build target that must pass Gatekeeper.
**Example (fields confirmed against `pkg/config/config.go:1013-1038` in the pinned v2.17.1 module):**
```yaml
# Source: pkg/config/config.go:1013-1038 (pinned goreleaser/v2@v2.17.1), cross-checked
# against https://goreleaser.com/customization/notarize (Context7, HIGH confidence — official docs)
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'   # explicit — do not omit (ambiguous empty-string default)
      ids:                                          # MUST list this project's actual darwin build ids —
        - codegraph-darwin-amd64                    # default is [ProjectName] ("codegraph"), which does
        - codegraph-darwin-arm64                    # NOT match either build id and causes a SILENT SKIP
                                                      # (pipe.Skipf, non-fatal) — see Pitfall #2
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"       # base64 .p12 content OR a file path (auto-detected)
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
        # entitlements: intentionally OMITTED — D-03's working hypothesis. quill embeds
        # NOTHING when this key is unset (quill/sign/entitlements.go:13-16); hardened
        # runtime is applied unconditionally regardless (see Pattern 2 below).
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"              # base64 .p8 content OR a file path
        wait: true
        timeout: 20m                                 # default 10m; TWO darwin binaries are notarized
                                                      # SEQUENTIALLY within this one config entry
                                                      # (internal/pipe/notary/macos.go:90, a plain for
                                                      # loop, not parallel) — budget accordingly
```

### Pattern 2: the pipeline-ordering fix for SIGN-04

**What:** `binary_signs:` (build-scoped cosign) runs BEFORE `notary.MacOS{}` in GoReleaser's hardcoded pipeline; `signs:` (release-scoped) runs well AFTER it.
**Evidence [VERIFIED: internal/pipeline/pipeline.go:63-106, pinned goreleaser/v2@v2.17.1]:**
```go
// BuildPipeline (lines 63-106) — runs during BOTH `goreleaser build` and
// `goreleaser release` (release = BuildPipeline + more, appended):
//   ...
//   // sign binaries
//   sign.BinaryPipe{},      // <- binary_signs: HERE
//   // notarize macos apps
//   notary.MacOS{},         // <- notarize: HERE, AFTER binary_signs — WRONG ORDER for D-04
// }

// Pipeline (lines 120-175) = BuildPipeline + the following, IN ORDER:
//   changelog.Pipe{}, archive.Pipe{}, sourcearchive.Pipe{}, nfpm.Pipe{}, srpm.Pipe{},
//   makeself.Pipe{}, snapcraft.Pipe{}, flatpak.Pipe{},
//   sbom.Pipe{},        // <- sboms: — automatically AFTER notarize, no fix needed
//   checksums.Pipe{},   // <- checksum: — automatically AFTER notarize, no fix needed
//   sign.Pipe{},        // <- signs: — automatically AFTER notarize — THIS is where cosign belongs
//   ... aur, nix, winget, brew, cask, ... publish.New(), metadata.ArtifactsPipe{}, announce.Pipe{}
```
**Recommended fix — move cosign from `binary_signs:` to `signs:` with `artifacts: binary`:**
```yaml
# `signs:` filters ByType(artifact.UploadableBinary) — internal/pipe/sign/sign.go:113-114
# [VERIFIED]. For THIS project's archives: shape (a `raw` id with formats:[binary] and a
# SEPARATE `zip` id with formats:[zip]), the `raw` entry alone produces the 4 UploadableBinary
# artifacts (internal/pipe/archive/archive.go:296-330, the `skip()` fast path taken only for
# formats:[binary] entries) [VERIFIED]. The `zip` entries produce UploadableArchive-typed
# artifacts instead, so they do NOT collide with this filter. This directly contradicts Phase
# 1's D-14 comment in .goreleaser.yaml ("signs: pipe's artifacts: binary mode... requires
# archives.formats: binary project-wide") — that claim does not hold up against the pinned
# v2.17.1 source read this session; flag for maintainer re-confirmation before committing to
# the switch (see Open Questions).
signs:
  - cmd: cosign
    # SAME name-derived-not-path-derived fix Phase 1 already applied to binary_signs: (D-14) —
    # the ${artifact} Path-vs-Name hazard is identical in signone() for both sign.Pipe{} and
    # sign.BinaryPipe{} (they share the same signone() function, internal/pipe/sign/sign.go:179).
    signature: "{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}.sigstore.json"
    args:
      - "sign-blob"
      - "--bundle=${signature}"
      - "${artifact}"
      - "--yes"
    artifacts: binary
```
**Verification path (mirrors 01-06's `release:dry-run-signed` pattern):** rehearse with a throwaway local cosign key, then assert from `dist/artifacts.json` — never a green exit code — that the Signature-typed record's sha256 (recompute by hashing the artifact at `.path`) equals the sha256 of the notarized, archived, checksummed binary. This is the empirical proof the plan needs; source-reading alone establishes *why* the current config is wrong, not that the proposed fix is complete.

### Anti-Patterns to Avoid

- **Trusting `goreleaser check` (`task check:goreleaser`) to validate `notarize:` semantically.** This project's own `.goreleaser.yaml` comments already document that `goreleaser check` does not validate the `artifacts:` enum for `signs:`/`sboms:`; the same class of blind spot likely applies to `notarize:` (not independently confirmed this session — no dedicated `checks` package reference to `Notarize` was found in the pinned source). Do not treat a clean `check:goreleaser` run as evidence the notarize config is correct.
- **Relying on `--skip=sign` to also skip `notarize:`.** They are separate skip keys (`skips.Sign` vs `skips.Notarize`, `internal/skips/skips.go:16-24`) [VERIFIED]. `task release:dry-run`'s `--skip=publish,sign` will still attempt to run `notary.MacOS{}`; safety in unprivileged CI canaries (`linux-cross-canary.yml`, `darwin-toolchain-canary.yml`) depends entirely on `enabled:` correctly evaluating false when Apple secrets are absent — see Pitfall #3.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| macOS code signing + notarization | `codesign`/`xcrun notarytool` shell steps | `notarize.macos` (quill in-process) | D-01: keeps signing inside GoReleaser's artifact bookkeeping; avoids re-adding shell glue to the single unrehearsable release path |
| Detecting whether a binary is Gatekeeper-trusted | Parsing `codesign -dvv` output | `spctl -a -vv -t exec` + `syspolicy_check distribution` | D-02: `codesign -dvv` already passes on the adhoc-signed, Gatekeeper-rejected binary today — it answers "is this signed" not "does Gatekeeper trust this" |
| Quarantine-attribute simulation for a RED-baseline test | A custom quarantine-attribute writer/format guesser | `xattr -w com.apple.quarantine "0081;<hex-epoch>;<agent>;<uuid>"` | Apple's own documented four-field format (flags;timestamp;agent;event-UUID); a wrong format could produce a false RED or false GREEN — see Open Questions for the un-verified event-UUID requirement |

**Key insight:** every piece of this phase's *signing* mechanism already has a maintained, in-process Go implementation (quill) reachable through the tool this project already depends on. The only genuinely custom code this phase should write is (a) the pipeline-order fix, (b) the CI/local credential wiring, (c) the new test-binary-override seam, and (d) the Gatekeeper verification script — none of which quill or GoReleaser provide directly.

## Common Pitfalls

### Pitfall 1: `binary_signs:` cosign-signs the WRONG bytes (headline finding)
**What goes wrong:** As `.goreleaser.yaml` is configured today (`binary_signs:`, not `signs:`), cosign's detached signature is computed over the binary BEFORE quill's notarize pipe mutates it (embeds `LC_CODE_SIGNATURE`). The published, checksummed, archived asset is the POST-notarize binary. `cosign verify-blob` against the real published asset will fail sha256 comparison implicitly — the bundle was signed over different bytes than what ships.
**Why it happens:** `sign.BinaryPipe{}` and `notary.MacOS{}` are both registered inside the hardcoded `BuildPipeline` Go slice, with `sign.BinaryPipe{}` first (`internal/pipeline/pipeline.go:102-106`). `.goreleaser.yaml`'s written block order has no effect on this.
**How to avoid:** Move cosign to the release-scoped `signs:` pipe (`artifacts: binary`), which is registered after `BuildPipeline` entirely — see Architecture Patterns, Pattern 2.
**Warning signs:** `cosign verify-blob` succeeding against a LOCAL `dist/` copy (pre-notarize state may coincidentally match) but failing against the re-downloaded PUBLISHED asset — exactly the class of bug D-05 exists to catch, and exactly what this repo's own `h9348wvthq` incident (a verifier that never downloaded what it verified) already taught it to distrust.

### Pitfall 2: silent skip on `ids:` mismatch
**What goes wrong:** `notarize.macos[].ids` defaults to `[ProjectName]` = `["codegraph"]` (`internal/pipe/notary/macos.go:35-37`) [VERIFIED]. This project's darwin build ids are `codegraph-darwin-amd64`/`codegraph-darwin-arm64`, not `codegraph`. If `ids:` is left unset, the filter `artifact.ByIDs("codegraph")` matches zero darwin binaries, and `signAndNotarize` returns `pipe.Skipf("no darwin binaries found with ids: %s", ...)` — a **non-fatal** `ErrSkip` that GoReleaser logs as informational and continues past, exit code 0.
**Why it happens:** GoReleaser's `Skip`/`Skipf` pipe-error convention treats "nothing matched the filter" as an acceptable no-op, not a hard failure — the same class of blind spot this project's own comments already documented for `signs:`/`sboms:` ("a filter matching nothing is a HARD FAILURE, not a pass" per the `release:dry-run-signed` Taskfile comment).
**How to avoid:** Explicitly set `ids: [codegraph-darwin-amd64, codegraph-darwin-arm64]`. Verify via `dist/artifacts.json` (or a live `spctl` run) that notarization actually happened — never trust a green `goreleaser release` alone.
**Warning signs:** A green release with un-notarized darwin binaries and no error anywhere in the logs.

### Pitfall 3: `enabled:` must correctly gate BOTH directions
**What goes wrong:** `notarize.macos[].enabled` is a templated boolean string (default: unset/false per docs). If it evaluates true with no valid credentials, the pipe hard-fails (bad — but loud). If it evaluates false in the REAL release job (secrets present but the template is wrong), the pipe silently skips and ships an un-notarized binary with a green CI run — a much worse, silent failure.
**Why it happens:** The recommended idiom (`enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'`) is a template that must be verified against BOTH environments: unprivileged canaries (`linux-cross-canary.yml`, `darwin-toolchain-canary.yml` — no Apple secrets, must stay false) and the real `release.yml` job (secrets present, must be true).
**How to avoid:** After adding the secrets to `release.yml`, explicitly assert (via `dist/artifacts.json` or a `spctl` check) that notarization fired on a real dry-run/rehearsal — not just that the job was green.
**Warning signs:** Same as Pitfall 2 — a green release, un-notarized binaries, silent.

### Pitfall 4: `--skip=sign` does not skip `notarize:`
**What goes wrong:** `skips.Sign` and `skips.Notarize` are distinct keys (`internal/skips/skips.go`). `task release:dry-run`'s `--skip=publish,sign` will still attempt `notary.MacOS{}`.
**Why it happens:** GoReleaser added `notarize` as its own skip key, separately from `sign`.
**How to avoid:** Rely on `enabled:` templating (Pitfall 3) to keep unprivileged CI canaries from reaching Apple; do not assume `--skip=sign` provides that safety. `--skip=notarize` IS a valid value for `goreleaser release` (in `skips.Release`, confirmed `internal/skips/skips.go`) but is **not** valid for `goreleaser build` (not in `skips.Build`) — same restriction `binary_signs:` already has, and already mitigated by this project's prior decision to unwire `check:darwin-release-build` from CI (Phase 1, commit `008d51c`).
**Warning signs:** A CI canary unexpectedly attempting a real Apple API call and hanging/timing out (mirroring the `expired_token` 5-minute hang `binary_signs:` caused before its own precondition was added).

### Pitfall 5: no existing binary-override seam in the integration/wireoracle test harnesses
**What goes wrong:** `test/integration/main_test.go:39-57` and `test/wireoracle/main_test.go`'s `TestMain` both hardcode `go build -o binPath github.com/seanb4t/codegraph-go/cmd/codegraph` [VERIFIED: read this session]. There is no env var or flag to point either harness at an already-downloaded, notarized binary.
**Why it happens:** These harnesses were built to test "the real binary," but "real" has always meant "freshly compiled from the checked-out source," never "the actual published release asset."
**How to avoid:** Add an env-var override mirroring the existing, proven precedent at `internal/upgrade/verify_release_e2e_test.go:97-105` (`CODEGRAPH_E2E_BINARY`/`CODEGRAPH_E2E_BUNDLE`, checked before falling back to the default behavior).
**Warning signs:** none yet — this is a gap to fill, not a live bug.

### Pitfall 6: quarantine xattr's fourth field (event UUID) may or may not need to reference a real LaunchServices database entry
**What goes wrong:** `xattr -w com.apple.quarantine "0081;<hex>;<agent>;<uuid>"` sets the flag bits and metadata Gatekeeper/`spctl` read, but the fourth field is documented as referencing a `com.apple.LaunchServices.QuarantineEventsV2` database row in some sources. If `spctl`/`syspolicyd` cross-checks that database and a synthetic UUID is absent from it, the RED-baseline (or the notarized GREEN check) result could be an artifact of the *test rig*, not of Gatekeeper's real policy.
**Why it happens:** Not independently confirmed this session — no macOS host was available to execute `xattr`/`spctl` directly.
**How to avoid:** Verify empirically, on the real macOS runner (`namespace-profile-macos-6x14-tahoe`), that a synthetic quarantine xattr (no real LaunchServices DB entry) produces the SAME `spctl` verdict as a genuine browser download, before trusting either the RED or the GREEN result. See Open Questions.
**Warning signs:** `spctl` behaving differently on a synthetic-xattr file vs. a genuinely browser-downloaded file of the identical binary.

## Code Examples

### GitHub Actions secrets wiring (release.yml)
```yaml
# Source: pattern confirmed by cross-referencing internal/pipe/notary/macos.go's
# tmpl.ApplyAll target fields (cfg.Sign.Certificate, cfg.Sign.Password, cfg.Notarize.Key,
# cfg.Notarize.KeyID, cfg.Notarize.IssuerID) against release.yml's existing single
# id-token: write job pattern [VERIFIED against both files this session]
- name: Release
  id: release
  run: task release:goreleaser
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
    MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}
    MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
    MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
    MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
```
No `security create-keychain`/`xcrun notarytool store-credentials` step is needed — unlike GoReleaser's own published GitHub Actions example (which is for a different/older workflow shape) [CITED: https://goreleaser.com/customization/notarize], quill's `BytesFromFileOrEnv` (`quill/pki/load/bytes.go:14-58`) [VERIFIED] auto-detects whether `certificate`/`key` is a filesystem path or raw base64 content and decodes accordingly — a base64 secret passed straight through as an env var is sufficient.

### Test-binary-override seam (new, mirroring an existing precedent)
```go
// Source: internal/upgrade/verify_release_e2e_test.go:92-105 (existing precedent, this
// session VERIFIED) — mirror this shape in test/integration/main_test.go's TestMain and,
// if wireoracle is in scope for Criterion 4, test/wireoracle/main_test.go's TestMain.
func TestMain(m *testing.M) {
	if envBin := os.Getenv("CODEGRAPH_TEST_BIN"); envBin != "" {
		binPath = envBin
		os.Exit(m.Run())
	}
	// ...existing go build fallback unchanged...
}
```

### Gatekeeper RED-baseline reproduction (docs/RELEASE.md criterion 5)
```sh
# Source: D-06/D-09's specifics + this project's own captured measurements
# (ROADMAP.md Phase 2 Notes) — the xattr step is REQUIRED, not optional (D-02's
# "spctl on a never-quarantined file is insufficient evidence").
curl -LO https://github.com/seanb4t/codegraph-go/releases/download/<tag>/codegraph_<tag>_darwin_arm64
xattr -w com.apple.quarantine "0081;$(printf '%x' "$(date +%s)");Safari;$(uuidgen)" codegraph_<tag>_darwin_arm64
xattr -p com.apple.quarantine codegraph_<tag>_darwin_arm64   # confirm BEFORE assessing
spctl -a -vv -t exec codegraph_<tag>_darwin_arm64
syspolicy_check distribution codegraph_<tag>_darwin_arm64
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `brews:` (Homebrew formula) | `homebrew_casks:` | GoReleaser v2.10 | Already recorded in this project's REQUIREMENTS.md; not directly relevant to Phase 2 but confirms the pinned v2.17.1 postdates this deprecation |
| `notarize.macos_native` requiring keychain/`security`/`xcrun` shell steps (per GoReleaser's own published doc example) | `notarize.macos` (quill-backed, in-process, no shell-out) | quill integration predates this repo's pin (present since early GoReleaser v2.1+, entitlements support added v2.6) | The GitHub Actions doc example with `security create-keychain` appears to be either stale or scoped to the Pro/native variant — this project should NOT copy it verbatim |

**Deprecated/outdated:** none directly relevant beyond the above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `syspolicy_check distribution <path>` accepts a bare Mach-O binary (not just an app bundle) and its pass/fail output format matches what's assumed in Code Examples | Standard Stack, Code Examples | If it requires an app bundle or has different output, SIGN-02's second verification leg needs a different invocation shape — must be confirmed on the real macOS runner |
| A2 | A synthetic `xattr -w com.apple.quarantine "0081;...;...;$(uuidgen)"` (no real LaunchServices QuarantineEventsV2 database entry) produces the same `spctl`/`syspolicyd` verdict as a genuine browser download | Common Pitfalls #6, Code Examples | If `syspolicyd` cross-checks the events database, the RED/GREEN result could be a test-rig artifact rather than a real Gatekeeper verdict — must verify on real hardware before trusting either result |
| A3 | Switching cosign from `binary_signs:` to `signs: {artifacts: binary}` is a complete fix for SIGN-04, with no other GoReleaser-internal ordering constraint between `sign.Pipe{}` and other release-scoped pipes (`aur`, `nix`, `brew`, `cask`, `publish`) that would reintroduce a mismatch | Architecture Patterns Pattern 2 | If some later pipe also mutates the binary (none identified in this session's read of `pipeline.go`, but not exhaustively traced past `checksums.Pipe{}`/`sign.Pipe{}`), the fix would be incomplete — verify empirically via the recommended rehearsal, not by source-reading alone |
| A4 | This project's own Phase 1 D-14 rationale ("`signs:` pipe's `artifacts: binary` mode... requires `archives.formats: binary` project-wide") does not hold against the pinned v2.17.1 source | Architecture Patterns Pattern 2 | If D-14's original concern was correct for a reason not visible in the code paths read this session (e.g., an `ids:` interaction, or a check-time validation, not a runtime filter behavior), reverting to `binary_signs:` and finding an alternative fix for the ordering problem would be needed — flagged explicitly for maintainer re-confirmation |
| A5 | Neither `sign.BinaryPipe{}` nor `notary.MacOS{}` write GoReleaser secret material (P12/API key bytes) into logs on error paths | Security Domain | If `quill.Sign`/`quill.Notarize`'s error wrapping ever includes raw certificate/key bytes in a returned `error`, and that error is logged verbatim by `caarlos0/log`, secrets could leak into CI logs — not traced exhaustively this session |

**If this table is empty:** N/A — see entries above.

## Open Questions

1. **Does `syspolicy_check distribution` accept a bare Mach-O binary, or only an app bundle/installer?**
   - What we know: the tool exists, is part of `syspolicyd`, and is documented informally as validating pre-distribution readiness (web search only, no official Apple documentation page located this session).
   - What's unclear: exact accepted input types and exact output format/exit codes for a bare, non-bundled executable — this project ships bare Mach-O binaries (D-16), not `.app` bundles.
   - Recommendation: run it against a real notarized darwin asset on `namespace-profile-macos-6x14-tahoe` early in plan execution (ideally as a Wave 0 spike), before writing the plan's verification-step wording.

2. **Does the quarantine xattr's fourth field need to reference a real LaunchServices QuarantineEventsV2 database entry for `spctl`/`syspolicyd` to treat the file as authentically quarantined?**
   - What we know: the four-field format (`flags;timestamp;agent;event-uuid`) is well-documented; the flag semantics (`0081` = downloaded-and-first-launch) are well-documented.
   - What's unclear: whether `syspolicyd` cross-validates the UUID against the actual events database, or merely reads the xattr's flag bits.
   - Recommendation: same as #1 — verify on real hardware before the plan locks in a specific `xattr -w` invocation as the canonical RED/GREEN test rig.

3. **Does `goreleaser check` validate the `notarize:` block's schema at all (e.g., catch a typo'd field name)?**
   - What we know: this project's own `.goreleaser.yaml` comments document that `goreleaser check` does NOT validate the `artifacts:` enum for `signs:`/`sboms:` — a documented existing blind spot for a structurally similar pipe.
   - What's unclear: whether `notarize:`'s validation is any stricter — not traced in the `checks` package this session (no `Notarize` reference found there, but that alone does not prove no validation exists).
   - Recommendation: treat `goreleaser check` as non-authoritative for this block; verify correctness via `dist/artifacts.json` inspection during rehearsal, same discipline already applied to `signs:`/`sboms:`.

4. **Is there any GoReleaser-internal pipe registered between `sign.Pipe{}` (`internal/pipeline/pipeline.go` line ~143) and `publish.New()` that could further mutate a darwin binary artifact (invalidating the cosign signature a second time)?**
   - What we know: the pipes between them in this project's actual config are inert (no `aur:`, `nix:`, `winget:`, `brew:`/`cask:` — Phase 3's concern, not yet configured — `chocolatey:`, `krew:`, `scoop:` blocks exist in this repo's `.goreleaser.yaml`).
   - What's unclear: whether any FUTURE addition (e.g., Phase 3's `homebrew_casks:`) could reintroduce a similar ordering hazard for darwin binaries specifically — worth a forward note for Phase 3's own research rather than solved here.
   - Recommendation: not blocking for this phase; flag for Phase 3 research to re-check pipeline ordering once `homebrew_casks:` is added.

5. **The maintainer's actual Apple Developer Program enrollment status, certificate type availability, and App Store Connect API key provisioning cannot be verified from this repository.**
   - This mirrors the phase's own explicit caution: prior overconfident claims about repo state ("the repo never references an arm64 runner" reported as "the account has no arm64 runner") were wrong because the evidence couldn't reach the claim. This research can confirm what GoReleaser/quill NEED (a Developer ID Application certificate's `.p12`, an App Store Connect API key's issuer ID/key ID/`.p8`) but cannot confirm whether the maintainer already holds these — that is an out-of-repo, account-state fact PROJECT.md's Operator Next Steps already flags as an external lead-time item.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| macOS runner (`namespace-profile-macos-6x14-tahoe`) | notarize pipe, `spctl`/`syspolicy_check`/`xattr`, local rehearsal | ✓ (already used by `release.yml`, both canaries) | macOS "Tahoe" (per runner profile name) | none needed — already the project's darwin runner class |
| Apple Developer ID Application certificate (`.p12`) | `notarize.macos[].sign.certificate` | **unverified from this repo** | — | none — blocking, external, human-provisioned (PROJECT.md Operator Next Steps already flags this as a lead-time item) |
| App Store Connect API key (issuer ID, key ID, `.p8`) | `notarize.macos[].notarize.*` | **unverified from this repo** | — | none — blocking, external, human-provisioned |
| `spctl`, `xattr`, `codesign` (macOS built-ins) | SIGN-02/SIGN-03 verification | ✓ (standard on any macOS install, incl. CI runner images) | OS-bundled | — |
| `syspolicy_check` (macOS built-in) | SIGN-02 secondary verification | assumed ✓ on a recent macOS (Tahoe), not independently confirmed this session | — | if genuinely absent on the runner image, D-02's SIGN-02 wording would need a documented fallback — flag during Wave 0 |
| `quill` / `goreleaser/v2` (Go modules) | notarize pipe itself | ✓ (already in module cache, confirmed this session) | goreleaser v2.17.1, quill `v0.0.0-20260630015114-8310f3e9a321` | — |

**Missing dependencies with no fallback:**
- Apple Developer ID Application certificate and App Store Connect API key — external, human-provisioned, must exist before the plan's rehearsal/execution steps can run for real (D-09's precondition-check target will surface this loudly and early if absent, which is the intended behavior, not a gap to fix).

**Missing dependencies with fallback:**
- none identified — `syspolicy_check`'s presence is assumed, not confirmed; if absent, the fallback is documenting SIGN-02 with `spctl` alone plus a note, pending maintainer confirmation.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (this repo's only test framework; no BATS/pytest-equivalent) |
| Config file | none — `Taskfile.yml` targets wrap `go test` invocations |
| Quick run command | `task test:unit` (excludes daemon), plus the specific new targets this phase adds |
| Full suite command | `task test:unit && task test:integration && task test:golden && task test:daemon` (existing composition; no single "run everything" target exists today) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|-------------|
| SIGN-01 | notarize pipe fires with real credentials, produces a Developer-ID-signed, notarized darwin binary | manual-once (local rehearsal, D-08) + CI (real release) | `task <new-notarize-rehearsal-target>` (maintainer-only, D-09 preconditions) | ❌ Wave 0 — new target |
| SIGN-02 | `spctl`/`syspolicy_check` GREEN against the notarized, quarantined, re-downloaded asset | manual/CI | new `task verify:gatekeeper`-style target, callable from `post-release-verify.yml` | ❌ Wave 0 — new target |
| SIGN-03 | `spctl` RED against `v0.5.1`'s un-notarized, quarantined asset | manual-once, recorded in phase artifacts (D-06) | ad hoc shell (see Code Examples) — NOT a standing automated gate per D-06/D-07's philosophy | ❌ one-time, not a persisted test |
| SIGN-04 | sha256 identity across notarize→archive→checksum→cosign→SLSA, plus one deliberate mis-order mutation | manual-once, recorded (D-07) | ad hoc shell during rehearsal, following the `release:dry-run-signed` pattern | ❌ one-time, not a persisted test |
| Criterion 4 | full CLI+MCP integration suite green against the notarized, re-downloaded binary itself | integration | `CODEGRAPH_TEST_BIN=<path> task test:integration` (new env var) | ❌ Wave 0 — TestMain seam does not exist yet |

### Sampling Rate

- **Per task commit:** N/A for the Apple-credential-gated pieces (cannot run without real secrets); `task test:unit`/`task vet` for any Go code changes (e.g., the TestMain seam).
- **Per wave merge:** local rehearsal (D-08) before any tag push — this IS the "full suite" for this phase's core mechanism, since CI cannot safely rehearse with real Apple credentials on a PR trigger (D-17).
- **Phase gate:** the RED-then-GREEN sequence (D-06 → real release → D-10's post-release-verify run) is itself the phase gate; there is no separate "full suite green" checkpoint distinct from this sequence, since a real release is inherently irreversible and one-shot per D-07's patch-forward posture.

### Wave 0 Gaps

- [ ] `test/integration/main_test.go`'s `TestMain` — add `CODEGRAPH_TEST_BIN` env-var override, mirroring `internal/upgrade/verify_release_e2e_test.go:97-105`
- [ ] A new `Taskfile.yml` target for the D-08/D-09 guarded local notarize rehearsal, with named preconditions for the certificate and API key (mirroring `command -v cosign`-style preconditions already used elsewhere in this Taskfile)
- [ ] A new `Taskfile.yml` target (e.g. `verify:gatekeeper`) wrapping the `xattr`+`spctl`+`syspolicy_check` sequence, parameterized by TAG/asset path, for reuse by both the RED-baseline step and the real post-release GREEN check
- [ ] `.github/workflows/post-release-verify.yml` — new job(s) for the Gatekeeper check and the integration-suite-against-real-binary check, preserving the D-11 event-aware guard verbatim

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | no | not applicable — no end-user auth in this phase |
| V3 Session Management | no | not applicable |
| V4 Access Control | partial | GitHub Actions secrets scoped to the single `id-token: write` job (D-14 carried forward); Apple secrets must never reach a `pull_request`-triggerable workflow (D-17) |
| V5 Input Validation | partial | `notarize.macos[].ids`/`enabled` are Go-template strings resolved against `.Env` — no user-controlled input reaches these templates in this phase's scope |
| V6 Cryptography | yes | Developer ID signing + App Store Connect API key handling — never hand-roll; use quill's in-process P12/API-key handling (`quill/pki/load`) as-is, never re-implement PKCS12 parsing or JWT-signing for the ASC API |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Apple signing cert/API key exposed to a `pull_request`-triggerable workflow | Information Disclosure / Elevation of Privilege | D-17 (carried forward): secrets live ONLY in `release.yml`'s single `id-token: write` job's `env:`; `darwin-toolchain-canary.yml`/`linux-cross-canary.yml` must never reference them |
| Secret material leaking into CI logs via an error message from `quill.Sign`/`quill.Notarize` | Information Disclosure | Not independently confirmed safe this session (Assumption A5) — worth a targeted check during plan execution: trigger a deliberate bad-password failure locally and inspect whether any raw key material appears in the error text |
| A stale/compromised `.p12`/`.p8` used indefinitely | Tampering / Repudiation | Out of scope for this phase (credential rotation policy is an operational, not implementation, concern) — note but do not build tooling for it |
| Quarantine-xattr test rig producing a false GREEN/RED (Pitfall #6 / Open Question #2) | Spoofing (of Gatekeeper's real verdict) | Verify empirically on real hardware before trusting either result as the phase's RED/GREEN evidence |

## Sources

### Primary (HIGH confidence)
- `internal/pipeline/pipeline.go` (pinned `goreleaser/v2@v2.17.1`, read directly from local Go module cache this session) — pipe registration order, `BuildPipeline`/`Pipeline`/`BuildCmdPipeline` composition
- `internal/pipe/notary/macos.go` (same pinned module) — `notarize.macos` runtime behavior, filter logic, default IDs, skip semantics
- `internal/pipe/sign/sign.go`, `internal/pipe/sign/sign_binary.go` (same pinned module) — `signs:`/`binary_signs:` filter behavior, shared `signone()` naming-template hazard
- `internal/pipe/archive/archive.go` (same pinned module) — `UploadableBinary` vs `UploadableArchive` type assignment for `formats: binary` vs `formats: zip` archive entries
- `internal/skips/skips.go` (same pinned module) — `skips.Sign` vs `skips.Notarize` as distinct keys; `skips.Release`/`skips.Build` valid-value lists
- `pkg/config/config.go` (same pinned module) — `Notarize`/`MacOSSignNotarize`/`MacOSNotarize`/`MacOSSign` struct field names and YAML tags
- `github.com/goreleaser/quill@v0.0.0-20260630015114-8310f3e9a321` (read directly from local Go module cache) — `quill/sign.go`, `quill/sign/signing_super_blob.go`, `quill/sign/entitlements.go`, `quill/pki/load/p12.go`, `quill/pki/load/bytes.go`
- This repository's own `.goreleaser.yaml`, `Taskfile.yml`, `.github/workflows/{release,post-release-verify,darwin-toolchain-canary,linux-cross-canary}.yml`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `test/integration/main_test.go`, `test/wireoracle/main_test.go`, `internal/upgrade/verify_release_e2e_test.go` — all read directly this session

### Secondary (MEDIUM confidence)
- Context7 `/websites/goreleaser` and `/goreleaser/goreleaser` — `notarize.macos` documented field list, GitHub Actions example (flagged as likely stale/scoped-differently against the actual source behavior)

### Tertiary (LOW confidence)
- WebSearch results on `syspolicy_check` exact CLI syntax/output and quarantine-xattr event-UUID semantics — no official Apple documentation page located; flagged explicitly as Open Questions requiring live-hardware verification

## Metadata

**Confidence breakdown:**
- Standard stack / GoReleaser+quill mechanics: HIGH — read directly from the pinned source this session, not inferred from docs or training data
- Architecture / pipeline ordering: HIGH for the ordering bug itself (direct source read); MEDIUM for the proposed fix's completeness (Assumption A3/A4, needs empirical rehearsal confirmation)
- Apple-tooling verification semantics (`spctl`/`syspolicy_check`/quarantine xattr): MEDIUM — documented format/behavior found via web search, but nothing executed on real macOS hardware this session
- Credentials/CI wiring: HIGH — read directly from quill's `BytesFromFileOrEnv` source
- Pitfalls: HIGH for GoReleaser-mechanics-derived pitfalls; MEDIUM for Apple-tooling-derived pitfalls

**Research date:** 2026-08-09
**Valid until:** 30 days for the GoReleaser/quill mechanics (stable, pinned version, won't drift until this project bumps the pin); 7 days for anything Apple-Gatekeeper-behavior-dependent if the maintainer's Apple Developer account state or macOS runner image changes in the interim
</content>
