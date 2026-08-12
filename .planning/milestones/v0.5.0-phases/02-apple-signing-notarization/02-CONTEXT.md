# Phase 2: Apple Signing & Notarization - Context

**Gathered:** 2026-08-09
**Status:** Ready for planning

<domain>
## Phase Boundary

A macOS user who downloads a `codegraph` release asset in a browser can run it without Gatekeeper blocking them — and the project can prove that claim with a check it has already watched fail.

Delivers SIGN-01, SIGN-02, SIGN-03, SIGN-04. No new product capability: the binary already works, and `codegraph upgrade` already installs it successfully. What changes is whether a *browser-downloaded* asset — the one population measured to carry a genuine `com.apple.quarantine` xattr — is trusted by the OS.

**In scope:** Developer ID codesigning + Apple notarization of darwin binaries inside the `goreleaser release` pipeline Phase 1 built; the CI secrets that authenticate it; the RED-then-green Gatekeeper gate; byte-identity of the notarized artifact across notarize → archive → checksum → cosign → SLSA; running the real suite against the notarized binary; the `docs/RELEASE.md` guarantee statement.

**Out of scope:** stapling and offline-safe first launch (deferred DIST-06 — `.zip` and bare Mach-O are both categorically unstaplable and Quill has no staple command); the Homebrew tap and cask (Phase 3, which consumes this phase's notarized bytes); brew-aware `upgrade` (Phase 4); any change to the Linux legs.

</domain>

<decisions>
## Implementation Decisions

### Signing & Notarization Mechanism

- **D-01:** Use GoReleaser's **native `notarize:` block** (quill-backed) rather than hand-rolled `codesign` + `xcrun notarytool` post-hooks. `github.com/goreleaser/quill` is already an indirect dependency at `go.tool.mod:236`, so this adds no new dependency and keeps signing inside GoReleaser's artifact bookkeeping — where `binary_signs:` and `sboms:` already live. Rejected alternative: Apple's own tooling via shell hooks. It would be authoritative, but it puts the signing step in shell glue inside the single unrehearsable path — the exact surface that shipped `v0.5.0` empty (see `h9348wvthq`; `docs/RELEASE-PROCEDURES.md` §7.1) — and outside GoReleaser's artifact records, which `binary_signs:`/`sboms:` depend on. — **Reversibility:** costly — swapping to hand-rolled hooks later means re-deriving the artifact set GoReleaser tracks internally, and re-proving SIGN-04's byte-identity chain against a pipe GoReleaser no longer owns.

- **D-02:** **Verification is Apple-native regardless of D-01.** Quill's own success report is never the oracle. SIGN-02 mandates `spctl -a -vv -t exec` reporting `source=Notarized Developer ID` plus `syspolicy_check distribution` passing; `codesign -dvv` is explicitly recorded as insufficient, because it *already passes today* on the adhoc/linker-signed darwin/arm64 binary that `spctl` rejects.

- **D-03:** Hardened runtime is required for notarization (Apple rejects submissions without it). **The entitlement set is an open research question**, not a locked decision — the researcher must establish what quill supports for entitlements and what a CGo-linked Go binary actually requires under hardened runtime before the planner commits. Working hypothesis to be confirmed or refuted, not assumed: tree-sitter's C compiles at *build* time, not runtime, so no JIT or unsigned-executable-memory entitlement should be needed. Criterion 4 is what converts that from "should" to "measured".

### Notarization Scope & Pipe Ordering

- **D-04:** Notarize the **raw Mach-O binaries**, and place the notarize pipe **before** archive/checksum/binary_signs/sboms. This makes the `.zip` contents, the checksums file, the cosign-signed subject and the SLSA-attested subject all describe post-notarization bytes *by construction* rather than by coincidence. Rejected: notarizing the `.zip` only — `codegraph upgrade` consumes the raw binary (`releaseAssetName`, `internal/upgrade/upgrade.go:209`), which would then ship un-notarized. Also rejected: notarizing both shapes — the `.zip` is built from the raw binary, so it doubles the Apple round-trip for redundant coverage. — **Reversibility:** one-way once published — the ordering determines what cosign signed and what SLSA attested for a given tag; changing it after a release means the published signature and attestation describe bytes that no longer match the shipped asset, and D-07 forbids deleting the release to fix it. Correcting it requires a new tag, patch-forward.

- **D-05:** SIGN-04's byte-identity claim is proven by diffing sha256 at four points — immediately after the notarize pipe, on the **re-downloaded published asset**, on the cosign-signed subject, and on the SLSA-attested subject — for each darwin binary. Re-downloading is not optional: Phase 1's verifiers produced three false negatives precisely because one of them never downloaded the artifact it then read (`h9348wvthq`).

### Proving the Gates Can Fire

- **D-06:** The SIGN-03 RED baseline is the **published `v0.5.1` darwin assets** (`codegraph_v0.5.1_darwin_arm64`, `codegraph_v0.5.1_darwin_amd64`), which are deliberately un-notarized and protected from deletion by `docs/RELEASE-PROCEDURES.md:300`. **Not `v0.5.0`** — that release has zero assets and cannot baseline anything. The RED result must be recorded with the `com.apple.quarantine` xattr confirmed present via `xattr -p` *before* the `spctl` run; `spctl` on a never-quarantined file is explicitly insufficient evidence.

- **D-07:** The SIGN-04 ordering claim is proven by a **one-time recorded mutation**: during execution, deliberately mis-order the pipe, record the divergent sha256 values as evidence in the phase artifacts, then revert. No permanent regression test is required for the ordering. Rationale for declining the standing guard: this repo has already been bitten by a test that *pinned a broken template* and would have resisted correction (`wewn0wp1n1`), so an ordering assertion carries a real risk of freezing the wrong shape. The trade accepted here is explicit — the proof is a recorded observation, not a standing guard, so a future refactor could silently reorder the pipe without a test firing. — **Reversibility:** reversible — a property-based regression test can be added later without undoing anything.

### Rehearsing the Unrehearsable Path

- **D-08:** The notarize pipe is rehearsed **locally on the maintainer's Mac**, via a **guarded, committed Taskfile target**, before any tag push. CI rehearsal via `workflow_dispatch` was considered and declined for this phase. Accepted consequence, stated plainly: the local rehearsal does not exercise the CI environment where an environment-specific bug would live, so first execution in CI remains genuinely first execution.

- **D-09:** The rehearsal target **must hard-fail by name** when the Developer ID certificate or App Store Connect API key is absent — not after a network round-trip to Apple. This mirrors the `command -v cosign` precondition Phase 1 added after `binary_signs:` hung ~5 minutes before failing with `expired_token` (`e2cnrbt6ph`). The target is documented as maintainer-only; contributors are not expected to hold an Apple certificate.

### Post-Release Verification

- **D-10:** Criterion 4's full CLI + MCP integration suite runs against the **notarized binary itself** — the re-downloaded published asset, never a locally rebuilt one — inside the **existing post-release-verify workflow**. That workflow already re-downloads assets for its cosign and attestation checks, so this reuses proven machinery and tests exactly the bytes a user receives. Failure is caught after publish and handled patch-forward per D-07 (project-level). Rejected: a pre-publish gate — it lengthens the single unrehearsable path and adds a new way for the job that must succeed to fail.

- **D-11:** Post-release-verify extensions must preserve the **event-aware guard**: under `workflow_dispatch`, `github.event.workflow_run` is null, and a bare `conclusion` test makes every job skip silently and report green (`h9348wvthq`). Any new job added here inherits that trap. Additionally, any release-walking logic must tolerate a **zero-asset release entry** — `v0.5.0` is permanently present and D-07 makes it so.

### Carried Forward (locked in prior phases — do not re-litigate)

- **D-12:** D-07 (project-level) **patch-forward**: never delete or re-push a published release or tag. Recovery is a new version.
- **D-13:** D-06R: release-please is the sole tag authority. No hand-created `v0.5.x` tag — it would match `release.yml`'s `push: tags: "v[0-9]*"` trigger and falsely fire the release pipeline.
- **D-14:** D-11 (Phase 1): exactly one job in `release.yml` holds `id-token: write`. `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` pins the cosign SAN to `.github/workflows/release.yml@refs/tags/v[0-9]*` — collapsing jobs is fine; **renaming the workflow file or changing the tag trigger is not**.
- **D-15:** The raw binary stays the primary asset (`codegraph upgrade` depends on its name); `.zip` archives ship *alongside*, never instead.
- **D-16:** Stapling is out of scope (DIST-06, revisitable on evidence). The shipped guarantee is **notarized, online-verified, not stapled** — matching GoReleaser's own choice for its own CLI.
- **D-17:** Granting `id-token: write` (or exposing signing secrets) to a `pull_request`-triggerable workflow is a **security regression, not a fix** — established when `binary_signs:` broke the darwin canary in Phase 1. Apple credentials must not become reachable from a PR trigger.

### Claude's Discretion

- The concrete shape of the CI secrets (base64 P12 + password vs. keychain import; App Store Connect `issuer_id`/`key_id`/`key` naming and encoding) — mechanical, follow GoReleaser's `notarize.macos` schema.
- Taskfile target naming, consistent with existing `check:*` / `verify:*` / `release:*` conventions.
- How the recorded SIGN-04 mutation evidence is formatted in the phase artifacts.
- Whether the `docs/RELEASE.md` reproduce-it-yourself commands live in a new section or extend §1.

### Reviewed Todos (not folded)

See `<deferred>`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope and locked requirements

- `.planning/ROADMAP.md` § "Phase 2: Apple Signing & Notarization" — goal, dependency on Phase 1, and the five success criteria in the form they must be proven
- `.planning/REQUIREMENTS.md` lines 24–27 — SIGN-01..SIGN-04 verbatim; line 57 records why stapling is deferred, line 60 why GoReleaser Pro is not adopted
- `.planning/PROJECT.md` — Key Decisions, including the maintainer parity ruling and D-06R/D-07
- `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-CONTEXT.md` — the 17 decisions Phase 1 locked; §"GoReleaser Config Shape" and §"Job Topology & Attestation" constrain where the notarize pipe may sit

### The pipeline this phase modifies

- `.goreleaser.yaml` — `builds:` (4 targets), `archives:` (`raw` + `zip`), `checksum:`, `binary_signs:` (lines ~180–232), `sboms:` (~233–278), `release:`. The long comment blocks above `binary_signs:` and `sboms:` document the `${artifact}` Path-vs-Name collision and are load-bearing context, not decoration
- `.github/workflows/release.yml` — tag-push trigger, the single `id-token: write` job (line ~90), the macOS runner `namespace-profile-macos-6x14-tahoe` (line 87), and `actions/attest-build-provenance` (line ~189)
- `Taskfile.yml` — `release:goreleaser` (:682), `release:dry-run` (:367), `release:dry-run-signed` (:488), `verify:release-assets` (:924), `verify:self-upgrade` (:1121), `check:darwin-toolchain` (:241), `check:darwin-release-build` (:270). Line 957 already anticipates Phase 2 notarization artifacts
- `go.tool.mod:236` — `github.com/goreleaser/quill` present as an indirect dependency; `go.tool.sum:767-768`

### Contracts this phase must not silently break

- `internal/upgrade/verify.go` — `releaseWorkflowRefPattern` pinning the cosign SAN to the workflow path and tag trigger
- `internal/upgrade/upgrade.go:209` — `releaseAssetName`, which binds the raw darwin asset filename that `codegraph upgrade` fetches
- `docs/RELEASE-PROCEDURES.md` §7.1 "Preserved baselines" (line ~286) and line 300 — the standing prohibition on deleting or replacing the `v0.5.1` darwin assets, which are this phase's RED baseline

### Documentation this phase must update

- `docs/RELEASE.md` — criterion 5 requires it state the guarantee exactly (notarized, online-verified, **not stapled**), name offline first launch as a known limitation, and give a reader the literal `xattr` + `spctl` commands to reproduce criterion 2. Existing structure: §1 verification (§a cosign, §b provenance, §c SBOM), §2 dependency tree, §3 reproducibility, § `codegraph upgrade` as consumer

### Background measurements (from promoted backlog 999.5)

- `.planning/ROADMAP.md` § Phase 2 **Notes** — the captured `codesign -dvv` / `spctl` results: darwin/arm64 is `adhoc, linker-signed` with `TeamIdentifier=not set`; darwin/amd64 is `code object is not signed at all`; `spctl -a -vv -t exec` returns **rejected** for both. Also records that a binary fetched by the real `codegraph upgrade` path carries only `com.apple.provenance`, which is why the affected population is browser downloaders specifically

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`github.com/goreleaser/quill`** (`go.tool.mod:236`) — already an indirect dependency; GoReleaser's native notarization backend. D-01 depends on this being present.
- **`verify:release-assets` (`Taskfile.yml:924`)** — already downloads published assets into a temp dir and runs cosign/attestation checks against them. The natural host for SIGN-04's re-downloaded-asset sha256 diff and for D-10's suite run.
- **`verify:self-upgrade` (`Taskfile.yml:1121`)** — already walks releases and was hardened in Phase 1 to require both no-semver-suffix *and* `prerelease: false`, and to skip zero-asset releases. Any new release-walking logic should reuse that discipline rather than re-derive it.
- **`check:darwin-toolchain` (`Taskfile.yml:241`)** — the existing macOS canary; note that `check:darwin-release-build` was deliberately **unwired** from it in Phase 1 (commit `008d51c`) because `binary_signs:` runs during build and demands an OIDC token. Do not re-wire it.
- **The macOS release runner** `namespace-profile-macos-6x14-tahoe` — real Apple tooling (`codesign`, `spctl`, `xcrun notarytool`, `syspolicy_check`) is available there, which is what makes D-02's Apple-native verification possible in CI.

### Established Patterns

- **Gates must be demonstrated RED before being trusted green.** This is the repo's defining discipline and the direct cause of SIGN-03's existence. Phase 1 found gates that could not FIRE and a gate that could never PASS.
- **Assert the property, not a literal string.** A test pinning a literal template froze a broken shape and would have resisted correction (`wewn0wp1n1`).
- **Preconditions halt by name, not by timeout.** Established after the 5-minute `expired_token` hang (`e2cnrbt6ph`).
- **`task` renders every `cmds:` string through Go `text/template` before the shell sees it.** Any `{{ }}` in a Taskfile command is eaten. This shipped `v0.5.0` empty. Notarization commands carrying templated values must avoid braces or be proven through `task`, not through a bare shell.
- **`task` echoes each command before running it**, so a log-line count assertion over-matches by 2x. Assert resolved content.
- **GoReleaser records `type: "Binary"` twice per platform** (raw build output + the archives pipe's renamed asset). Any `dist/artifacts.json` reader must also filter `.extra.Format == "binary"` or it counts 8 where 4 exist.

### Integration Points

- **`.goreleaser.yaml`** gains a `notarize:` block; its position relative to `archives:`/`checksum:`/`binary_signs:`/`sboms:` is what D-04 fixes and D-07 measures.
- **`.github/workflows/release.yml`** gains Apple credentials as secrets on the existing single `id-token: write` job — without adding a second such job (D-14) and without exposing them to any `pull_request`-triggerable workflow (D-17).
- **The post-release-verify workflow** gains the D-10 suite run, subject to D-11's event-aware guard and zero-asset tolerance.
- **`Taskfile.yml`** gains the D-08/D-09 guarded maintainer-only rehearsal target.
- **`docs/RELEASE.md`** gains the criterion-5 guarantee statement and reproduction commands.

</code_context>

<specifics>
## Specific Ideas

- The guarantee sentence in `docs/RELEASE.md` should be stated in the exact three-part form the roadmap uses: **notarized, online-verified, not stapled** — with offline first launch named as a known limitation rather than omitted.
- The reproduction snippet must include the `xattr` step, not just `spctl`. A reader who runs `spctl` on a never-quarantined file gets a misleading pass; that failure mode is explicitly called out in SIGN-03 and should be pre-empted in the docs rather than only in the gate.

</specifics>

<deferred>
## Deferred Ideas

- **CI rehearsal of the notarize pipe** (`workflow_dispatch`, main-branch-only, real credentials, output discarded) — considered and declined in favour of local rehearsal (D-08). Worth revisiting if the first notarized release fails for an environment-specific reason, since that is precisely the gap local rehearsal leaves open.
- **A permanent property-based regression test pinning the notarize pipe's position** — declined in favour of the one-time recorded mutation (D-07). Additive later; nothing in this phase forecloses it.
- **A credential-free contributor dry mode for the rehearsal target** — declined as surface without a current consumer (D-08 makes the target maintainer-only).
- **DIST-06 — stapled, offline-safe first launch** via hand-rolled `pkgbuild`/`productbuild`/`productsign`/`xcrun stapler`. Out of scope per REQUIREMENTS.md:57; revisitable on evidence that real users hit the offline case.

### Reviewed Todos (not folded)

- **Wire oracle toolslist-repeat response ordering flake** (`.planning/todos/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`, area `mcp`, score 0.9) — matched on generic keywords (`ordering`, `test`, `github`), not on domain. MCP wire-protocol concern with no relationship to macOS distribution.
- **Author a codegraph usage skill for agents** (`.planning/todos/2026-08-08-author-a-codegraph-usage-skill-for-agents.md`, area `agents`, score 0.6) — matched on `codegraph`/`cli`/`mcp` keywords. Agent-tooling work, unrelated to signing or notarization.

</deferred>

---

*Phase: 2-Apple Signing & Notarization*
*Context gathered: 2026-08-09*
