# Pitfalls Research — macOS Notarization + Homebrew Distribution

**Domain:** Adding Apple notarization and a Homebrew tap to an existing signed/attested Go release pipeline (GoReleaser + cosign + SLSA3 + release-please)
**Researched:** 2026-08-07
**Confidence:** MEDIUM-HIGH overall (Apple/Homebrew/GoReleaser official docs and issue trackers are HIGH confidence; some integration-specific predictions are MEDIUM — this repo's own `.goreleaser.yaml`/`release.yml` are the ground truth, read directly, so those claims are HIGH by construction)

This file assumes the reader has `.planning/PROJECT.md`, `.goreleaser.yaml`, and `.github/workflows/release.yml` open. Every pitfall below is framed as: **what goes wrong → how to detect it → how to prevent it → which phase owns it**, per this repo's standing rule that a gate is not trusted until demonstrated RED against a confirmed-applied mutation.

---

## Critical Pitfalls

### Pitfall 1: `goreleaser release` cannot consume dist/ artifacts built on other runners — the single-runner conflict is real, not hypothetical

**What goes wrong:**
`goreleaser release` "expects to wholly manage the `dist` directory, returning an error if it exists" (confirmed via GoReleaser's own GitHub issue tracker). There is no `--use-existing-dist` / `--skip-build` flag in OSS. The only mechanism GoReleaser ships for building on separate runners and combining results into one release is **Split & Merge** (`goreleaser release --split` / `goreleaser continue --merge`), and that is explicitly, exclusively a **GoReleaser Pro** feature per the official docs ("This feature is exclusively available with GoReleaser Pro"). This directly falsifies the implicit assumption in today's `release.yml` design (native-darwin-on-macOS + zig-cross-linux-on-Linux, merged in a downstream `assemble` job) that a plain `goreleaser release` invocation could simply replace the current `build --single-target` + hand-rolled assemble step. It cannot, as OSS, across two runner classes.

**Why it happens:** The current pipeline's per-runner `goreleaser build --single-target` pattern was deliberately chosen (per `.goreleaser.yaml`'s own header comment) to avoid GoReleaser Pro dependencies. Migrating to `goreleaser release` while assuming the same two-runner-class topology "just works" is exactly the kind of unverified assumption PROJECT.md itself flagged ("must be confirmed, not assumed") — and this research confirms the assumption was right to be suspicious of.

**How to avoid:** Run the entire `goreleaser release` invocation on a single runner. Given darwin must build natively (the recorded libresolv/DNS finding), that runner must be `macos-latest`/Namespace macOS, with `zig cc` cross-compiling both linux/amd64 and linux/arm64 from the same host — exactly the "likely resolution" PROJECT.md already named. This eliminates the merge problem entirely: one `goreleaser release` process builds all 4 targets, archives, checksums, signs (if quill or a macOS-native path is used for cosign too — cosign itself doesn't care about host OS), notarizes, and publishes, without ever needing Pro's split/merge.

**Warning signs:** A plan that still shows `build` as a per-runner matrix job feeding an `assemble` job, but relabels the assemble step's `goreleaser build --single-target` as `goreleaser release`. That shape cannot work — `release` refuses a pre-populated `dist/`.

**Detection (RED-demonstrable):** Write a throwaway workflow that runs `goreleaser build --single-target` on `namespace-profile-linux-amd64-4x8` to populate `dist/`, uploads it as an artifact, downloads it into a second job, and then runs `goreleaser release --clean` in that second job. This MUST fail with GoReleaser's dirty/existing-dist error. Confirming this failure, live, is the RED proof that the two-runner-class + `release` combination cannot work in OSS before any phase plan is written around it.

**Phase to address:** The `goreleaser release` migration phase — this is the enabling/blocking decision every other feature in the milestone (notarize, archives, brews) depends on. Must be resolved before notarization or brews work begins.

---

### Pitfall 2: Notarization "succeeds" (Accepted) while `spctl` still rejects the binary — a real, documented failure family, not a hypothetical

**What goes wrong:** `notarytool submit` returning `Accepted` is not proof the shipped artifact will pass Gatekeeper. Apple Developer Forums document multiple real cases of `notarytool history` showing successful notarization, `stapler` reporting success, and `spctl -a -vv -t exec` still rejecting the artifact with "Unnotarized Developer ID" or similar. One documented root cause: the notary ticket did not cover all the Mach-O images actually present in the shipped artifact (e.g. a nested binary, a nested dylib, or — relevant here — a binary added to an archive *after* the notarized artifact was built), so Gatekeeper's post-hoc ticket lookup finds no matching record for what's actually on disk.

**Why it happens:** Notarization is a request to notarize *exactly the bytes submitted*. If the pipeline notarizes binary A, then repackages/re-signs/rebuilds anything downstream (re-archiving, adding a README to the zip, GoReleaser regenerating the archive after signing instead of before), the bytes Gatekeeper checks are no longer the bytes Apple has a ticket for. This is an ordering bug (see Pitfall 6), and it produces exactly this symptom.

**How to avoid:** Notarize the *final* artifact shape — the archive/container that will actually be distributed, built in its final form, with nothing added or modified afterward. Verify by running `spctl -a -vv -t exec` against the literal file that will be uploaded to the GitHub Release, not against an intermediate build artifact.

**Warning signs:** `notarytool history` shows Accepted, but this check was run against a local build artifact in `dist/`, not against the file downloaded from the published GitHub Release.

**Detection (RED-demonstrable):** On a genuinely quarantined copy of the actual published release asset (see Pitfall 3 for how to force real quarantine), run `spctl -a -vv -t exec <path>` and require `accepted` with `source=Notarized Developer ID` in the output. Anything else — including a `notarytool` history entry of Accepted — is not sufficient evidence.

**Phase to address:** Notarization phase, as the phase's own verification gate (not a side check — this should be the phase's primary UAT criterion, matching PROJECT.md's own framing: "The bar is `spctl -a -vv -t exec` returning `accepted` on a file actually carrying `com.apple.quarantine` — not on the pipeline being wired").

**Confidence:** HIGH (multiple corroborating Apple Developer Forum threads on this exact failure shape, cross-referenced).

---

### Pitfall 3: The "wired up but cannot fire" trap, applied to notarization — four specific checks that feel like verification and are not

This is the single highest-priority section given this repo's documented history (retracted fictitious perf regression, inverted `rg -qv` gate, stale perf baseline, degenerate-input `CheckRegression` pass, drifted `requiredCheckNames` fixture, two passthrough tests). Notarization has its own version of every one of these failure shapes.

**Checks that FEEL like verification but are NOT:**

1. **A green CI step.** GoReleaser's `notarize:` block can be configured (or accidentally end up configured, e.g. via a missing/empty secret) to skip notarization silently and still exit 0 — the step shows green whether or not notarization actually ran. A green `notarize` job step proves the *command* didn't error; it proves nothing about whether Apple's notary service was ever contacted.
2. **`codesign -dvv` passing.** This validates that *a* signature is present and its certificate chain is valid — it passes identically for an ad-hoc signature (no Developer ID, no notarization possible) and for a properly Developer-ID-signed, notarized binary. Multiple sources confirm this explicitly: passing `codesign` verification "does not indicate notarization status." Never treat this as evidence of notarization.
3. **`notarytool` history/submission status showing `Accepted`.** As Pitfall 2 demonstrates, this is necessary but not sufficient — it proves Apple accepted *some* submission, not that the ticket covers the bytes actually shipped, and not that stapling (where applicable) succeeded.
4. **`spctl` on a file that was never quarantined.** This is the most insidious one, because it is the *exact same command* as the correct check, run on the wrong file. Gatekeeper's full assessment path (the one that actually rejects unnotarized software for end users) is triggered by the `com.apple.quarantine` extended attribute, which is set by browsers, `curl -O` with certain flags is used with, Safari/Chrome downloads, AirDrop, and Mail — but NOT by a local build artifact freshly produced by `goreleaser build`, NOT by `git clone`, and NOT by the real `codegraph upgrade` path (per PROJECT.md's own measurement: "a binary fetched by the real `codegraph upgrade` path was measured to carry only `com.apple.provenance`"). Running `spctl -a -vv -t exec` on such a file exercises a much weaker code path and can return `accepted` even for software that would be rejected for a real end user. One source states this plainly: "Gatekeeper signature checks are performed only to files with the Quarantine attribute, not to every file."

**The ONE trustworthy check:**

```bash
# Force a real quarantine attribute, matching what a browser download sets
# (com.apple.quarantine's value format: <flags>;<timestamp-hex>;<agent>;<uuid>)
xattr -w com.apple.quarantine "0081;$(printf '%x' "$(date +%s)");Safari;$(uuidgen)" <path-to-downloaded-asset>

# Now run the assessment Gatekeeper actually performs on a downloaded file
spctl -a -vv -t exec <path-to-downloaded-asset>
```

Expect: `<path>: accepted` and `source=Notarized Developer ID` on the second line. Any other `source=` value (e.g. `source=Unnotarized Developer ID`, or a rejection) means the release is not actually trustworthy for real users, regardless of what CI reported.

Do this against the literal file downloaded from the real, published GitHub Release URL (or at minimum an artifact with a genuinely synthesized quarantine attribute, as above) — never against a local `dist/` build artifact.

**Phase to address:** Notarization phase. This exact sequence (force quarantine → `spctl`) should be the phase's acceptance gate, run by a human or a scripted macOS runner step against the real published asset — and per this repo's standing rule, the phase is not done until this has been shown to move `rejected` → `accepted` on a genuinely quarantined download, not merely asserted.

**Confidence:** HIGH for the mechanism (official Apple documentation on quarantine + spctl behavior, cross-referenced across multiple independent sources); the exact `xattr` flag/timestamp encoding is MEDIUM confidence (community-documented format, not from a single canonical Apple source) — verify the synthesized attribute actually reproduces browser-download behavior before relying on it as the sole gate, or use a real browser download as the primary check and the `xattr` trick only as a fast local iteration aid.

---

### Pitfall 4: Stapling a bare Mach-O binary is impossible — this forces the archive-vs-raw-binary asset split, it isn't optional polish

**What goes wrong:** Apple's notary service can notarize a bare executable (by wrapping it in a zip for submission), but **stapling requires a container** — a `.app` bundle, `.pkg` installer, or `.dmg` disk image. Per Apple's own guidance (multiple corroborating sources): "Tickets can't be stapled to single-file Mach-O executables, but they can be stapled to Installer packages containing them." GoReleaser's own docs are explicit about a related trap: "Do not use this method if you create App Bundles. App Bundles in which only the binary is signed/notarized are deemed damaged by macOS" — i.e. signing/notarizing the inner binary is not a substitute for signing/notarizing the container itself.

**Why it matters here specifically:** This is exactly the constraint PROJECT.md already identified as "the real design call" — a bare Mach-O notarized-but-not-stapled binary falls back to Apple's online ticket lookup at Gatekeeper-check time, which **fails on an offline machine or when Apple's OCSP/notary lookup service is unreachable**. This is why the milestone's own Key Decisions committed to "archives alongside raw binaries" rather than trying to notarize+staple the raw binary that `internal/upgrade` needs to remain byte-unchanged.

**How to avoid:** Notarize and staple the **archive** (zip is the simplest container GoReleaser natively supports for a CLI binary; `.dmg`/`.pkg` are also viable but add build complexity with little benefit for a CLI tool) as the browser-download / Homebrew-facing asset. Never attempt to staple the raw binary that `internal/upgrade` consumes — that asset stays exactly as it is today (unnotarized-at-the-file-level is fine, because `internal/upgrade` never triggers a Gatekeeper quarantine check in the first place, per PROJECT.md's own `com.apple.provenance`-only measurement).

**Warning signs:** A plan step that says "staple the binary" (singular, bare) instead of "staple the archive." Also watch for `xcrun stapler staple` being invoked directly on a `codegraph_<tag>_darwin_arm64` raw-binary asset — this will fail (stapler returns an error for non-container inputs) or, worse, silently produce a container that still fails offline Gatekeeper checks if some tooling wraps it in a technically-valid-but-wrong container shape.

**Detection (RED-demonstrable):** Run `xcrun stapler staple <raw-binary-path>` on the actual raw binary GoReleaser/`release.yml` produces and confirm it errors (this proves the constraint is real, not assumed) before designing the archive path. Then run `xcrun stapler validate <archive-path>` on the actual zip that will ship and confirm it succeeds.

**Phase to address:** Notarization phase, in coordination with the archives phase (they are two views of the same asset-shape decision, per PROJECT.md's own Key Decision rationale).

**Confidence:** HIGH (GoReleaser official docs + multiple independent Apple-developer-community sources agree on this exact constraint).

---

### Pitfall 5: The checksums-file collision is not hypothetical — it is already latent in this repo's own config

**What goes wrong:** `.goreleaser.yaml`'s `checksum:` block (currently documented as dead config) is:

```yaml
checksum:
  name_template: "{{ .ProjectName }}_{{ .Tag }}_checksums.txt"
```

which resolves to `codegraph_v0.5.0_checksums.txt`. `release.yml`'s `assemble` job independently, by hand, produces:

```bash
sha256sum codegraph_* > "codegraph_${TAG}_checksums.txt"
```

which resolves to the **identical filename**. The moment `goreleaser release` actually runs its `checksum:` pipe (which it will, once the migration from `build` to `release` lands — this is precisely the "two blocks that have never executed wake up" risk PROJECT.md names), there will be two independent processes writing a file with the same name into the same release: GoReleaser's own checksums covering whatever `archives:`/binaries GoReleaser itself produced, and (if the hand-rolled step is not removed) the old shell step's checksums covering the renamed assets it finds on disk. Depending on execution order and whether `gh release upload ... --clobber` runs before or after GoReleaser's own publish step, one silently overwrites the other, or GoReleaser's own publish step fails outright on a duplicate asset name.

**Why it happens:** This is a direct consequence of migrating one component (the checksum mechanism) of a pipeline that has two independent implementations of the same responsibility, without deleting the now-redundant one. It's the textbook "gate that cannot fire" shape this repo keeps rediscovering — except here the risk isn't a test silently never running, it's an artifact silently being overwritten with a **different, undocumented set of covered files** than the one that was actually attested/signed.

**How to avoid:** When the `goreleaser release` migration lands, the hand-rolled `sha256sum codegraph_* > ...` step in `release.yml`'s `assemble` job (or its successor) must be deleted, not left to run alongside GoReleaser's own `checksum:` pipe. Decide, explicitly, whether GoReleaser's cosign/SLSA integration signs GoReleaser's own checksums file or continues the current per-binary-signing scheme (see Pitfall 6) — do not let both checksum generators exist simultaneously even transiently, because a partial migration where GoReleaser produces `checksum:` output *and* the old shell step also runs is the exact shape where "last writer wins" silently determines what the SLSA provenance and cosign identities actually cover.

**Detection (RED-demonstrable):** After the migration, download the actual `codegraph_<tag>_checksums.txt` from the published release and diff its line count/content against the full asset list (`gh release view <tag> --json assets`). Every published binary and archive must have exactly one entry; a missing or duplicated entry proves the collision occurred.

**Phase to address:** The `goreleaser release` migration phase, as an explicit removal task tracked alongside the archives/checksum-block-goes-live risk PROJECT.md already flags.

**Confidence:** HIGH — this is derived directly from reading this repo's own `.goreleaser.yaml` and `release.yml`, not from external sources.

---

### Pitfall 6: `internal/upgrade`'s cosign verification contract can silently stop matching what's actually shipped

**What goes wrong:** `internal/upgrade`'s `defaultVerify` hashes the **downloaded raw binary itself** (`sha256.Sum256(binary)`) and verifies a per-binary `.sigstore.json` bundle produced by `cosign sign-blob` in the `assemble` job — this is explicit in `release.yml`'s own comments ("internal/upgrade's defaultVerify hashes the DOWNLOADED BINARY ITSELF ... cosign MUST sign each binary individually"). Once `goreleaser release` is live with `archives:` and `notarize:` blocks, there is a real risk of introducing a step, anywhere in the new pipeline, that re-derives, re-copies, or re-names the raw binary asset *after* cosign has already signed it — for example, if a future refactor accidentally has GoReleaser's own `archives:` pipe repackage the binary from a different intermediate `dist/` path than the one cosign actually hashed, or if notarization is (incorrectly, per Pitfall 4) attempted against the raw binary and that process rewrites the file's signature bytes (codesign always rewrites the Mach-O when it signs), producing a binary whose bytes no longer match what cosign attested.

**Symptom:** `codegraph upgrade` starts failing with a hash-mismatch or signature-verification error for every user on a specific platform, but the release itself "looks" successful — the GitHub Release page has assets, cosign's own log shows `Verified OK` (because cosign verified the version of the file *it* was given at signing time, which may not be the version that ended up published).

**Why it's easy to miss:** The `cosign verify-blob` and SLSA verification steps this repo already runs check that a signature/attestation is internally consistent — they don't independently confirm that the *specific bytes at the public download URL* are the ones that were hashed, unless the verification step re-downloads the released asset (not a local `dist/` copy) before verifying, which is what the existing `TestVerifyReleaseE2E` does today (per PROJECT.md: "executing against a real artifact"). Any new pipeline step inserted between "binary built" and "binary published" that touches the binary's bytes — including codesign for notarization, if mistakenly applied to the raw asset — breaks this invariant.

**How to avoid:** Keep an explicit, auditable ordering invariant: raw binary is built → cosign signs the raw binary → raw binary is uploaded, byte-for-byte, unchanged. Notarization/codesigning/stapling happen only on the **separate archive artifact**, never on the raw binary that `internal/upgrade` consumes. Do not let GoReleaser's `archives:` block be configured to "archive" by re-copying/re-touching the same file path cosign already signed — verify GoReleaser's archive step reads from the already-built, already-signed binary without modification (it should, since `archives:` operates on already-built artifacts, but this must be confirmed for this repo's specific build-id/archive-id wiring once configured, not assumed).

**Detection (RED-demonstrable):** Extend the existing `TestVerifyReleaseE2E` (or add a sibling test) that, after a real release, downloads the raw binary asset AND independently downloads the archive asset for the same platform, and asserts the raw binary's bytes inside the archive (if extracted) are byte-identical to the standalone raw-binary asset. A divergence here is exactly the symptom this pitfall predicts, and should be demonstrated impossible (or fixed) before the phase is considered done, not assumed safe because cosign's own step reported success.

**Phase to address:** Split across the `goreleaser release` migration phase (ordering) and the notarization phase (must notarize the archive only, never re-touch the raw binary) — call out explicitly as an acceptance criterion in whichever phase finalizes the archive-building step.

**Confidence:** MEDIUM — the specific failure mechanism (codesign rewrites Mach-O bytes) is HIGH confidence (universally documented codesign behavior), but whether this repo's eventual GoReleaser config actually risks it depends on implementation details not yet written; flagged as a design constraint to verify against, not a confirmed bug.

---

### Pitfall 7: Hardened runtime entitlements — likely a non-issue for this specific CGo binary, but must be verified, not assumed

**What goes wrong (if it applies):** macOS's Hardened Runtime enables library validation by default, which "only allows processes to load code signed by Apple or with the same Team ID as the executable." A binary that `dlopen()`s an unsigned or differently-signed shared library at runtime will fail with a code-signing-related mmap/load error under Hardened Runtime unless `com.apple.security.cs.disable-library-validation` is set — and setting that entitlement itself has a documented downside ("Disabling library validation makes it harder to pass Gatekeeper" per Apple Developer Forum reports of it interfering with notarization in some configurations).

**Why this is likely NOT a live issue for codegraph-go specifically:** Per this repo's own architecture (STACK.md / CLAUDE.md), `tree-sitter/go-tree-sitter` and its grammar modules are linked via **CGo at compile time** — the C code is compiled and statically linked into the single Go binary, not loaded via `dlopen()` at runtime. A statically-linked CGo binary has no runtime dynamic-library-loading behavior for tree-sitter itself, so Hardened Runtime's library-validation restriction should not be triggered by it. This should hold for Pebble (pure Go), fsnotify (pure Go), and the MCP/CLI stack (pure Go) as well — none of them dlopen anything at runtime.

**Where it could still bite:** If the Go runtime itself dynamically loads any system library at startup (e.g. certain cgo-linked networking/DNS paths, or `net` package's use of the system resolver via `libresolv`/`libSystem` on darwin — notably, this repo's own release.yml comments already flag libresolv/DNS as a live darwin-specific concern in a different context, i.e. cross-compilation, not hardened runtime, but it's the same subsystem). Hardened Runtime's library validation applies specifically to *bundle-signed dependent libraries*, not to the OS's own system frameworks (Apple-signed libraries always pass library validation under any Team ID) — so this risk is low but not exactly zero without an actual test.

**How to avoid:** Do not add `com.apple.security.cs.disable-library-validation` speculatively. Sign with default Hardened Runtime options (`--options=runtime`) and run the binary end-to-end (all CLI commands, `serve --mcp`, indexing a real repo) on the actual notarized+signed artifact before assuming entitlements are unnecessary.

**Detection (RED-demonstrable):** After signing with Hardened Runtime enabled and no special entitlements, run the full existing CLI/MCP integration test suite against the signed macOS binary (not just `spctl`/notarization checks — an actual functional smoke test). A load failure (typically manifesting as a crash on startup with a codesign/dyld error in `Console.app` or stderr, not a Gatekeeper rejection) is the signal that an entitlement is actually needed; absence of a crash across the full command surface is the evidence entitlements are not needed, not an assumption.

**Phase to address:** Notarization phase, as a functional-smoke-test acceptance criterion alongside the `spctl` gate.

**Confidence:** MEDIUM — the general Hardened Runtime mechanism is HIGH confidence (well-documented Apple behavior), but whether it applies to this specific binary is an inference from this repo's known architecture, not something directly tested by this research pass.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Leaving the hand-rolled `sha256sum` step in `release.yml` "just in case" while also enabling GoReleaser's `checksum:` block | Feels like a safety net during migration | Two checksum files/generators racing (Pitfall 5) — silently changes what's actually attested | Never past the migration PR that flips `build` → `release`; delete in the same change |
| Notarizing all 4 build targets (including Linux) because it's easier than conditionalizing the `notarize:` block per-OS | Simpler GoReleaser config, one code path | Wasted CI time and unpredictable notarization-API latency on binaries that will never be Gatekeeper-checked (a documented real-world GoReleaser gotcha) | Never — gate `notarize:` to darwin build IDs only from the start |
| Skipping the "force real quarantine + spctl" check in favor of trusting `notarytool` history during early iteration | Faster local dev loop | Exactly the false-positive shape of Pitfall 3 — a maintainer can convince themselves it works when it doesn't | Acceptable ONLY as a fast local iteration signal, never as the phase's actual acceptance gate |
| Using a personal Apple ID / ad-hoc signing during development instead of the real Developer ID cert | No secrets management needed locally | `codesign -dvv` passes identically for ad-hoc and Developer-ID-signed binaries (Pitfall 3, item 2) — easy to forget which one CI actually uses | Fine for pure local dev signing sanity checks; never for anything claiming to validate the release pipeline |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| GoReleaser `release` ↔ release-please | Letting GoReleaser's own `release:` publisher create/manage the GitHub Release (title, body, changelog) when release-please already created it with its own changelog body | Configure GoReleaser to skip GitHub Release creation/changelog entirely (`release.disable: true` or `--skip=publish` for the release-object step while still using GoReleaser for build/archive/sign/notarize/checksum), reusing this repo's existing pattern of `gh release upload ... --clobber` against the release-please-created Release, exactly as `release.yml` does today for assets |
| GoReleaser `release` ↔ multi-runner build topology | Assuming `goreleaser release` can consume `dist/` artifacts assembled from separate `build` jobs on different runner classes, the way `build --single-target` currently is assembled | Confirmed impossible in OSS (Pitfall 1) — restructure to a single macOS runner using `zig cc` for both linux legs, or explicitly budget for GoReleaser Pro's Split & Merge |
| GoReleaser `brews:` ↔ tap repo | Publishing the formula (tap push) before the corresponding GitHub Release assets finish uploading, so `brew install` immediately after a release momentarily 404s on the download URL | GoReleaser's own publish ordering runs `release` (asset upload) before `brews:` (tap push) within a single `goreleaser release` invocation by design — but if brews publishing is ever split into a separate job/step (e.g. for token-scoping reasons), it MUST be sequenced strictly after asset upload completes and be verified, not assumed sequential |
| GoReleaser `brews:` ↔ parallel formula builds | Multiple brew formula uploads to the same tap repo racing each other (documented GoReleaser issue: "the 2nd upload always fails ... only one of the formulas being uploaded and committed") | Not directly applicable here (one formula, `codegraph`) unless a `-bin`/cask-style second formula is ever added later — if it is, this race is a known, documented GoReleaser bug to watch for |
| `internal/upgrade` ↔ Homebrew Cellar | Letting `codegraph upgrade` self-replace the binary at the symlinked `/opt/homebrew/bin/codegraph` (or `/usr/local/bin`) path, silently diverging from what `brew`'s Cellar manifest records as installed | Per Homebrew's own Acceptable Formulae policy, self-update MUST be disabled when the tool is a formula (this repo's committed decision — `codegraph upgrade` refuses under a brew-managed install) — verify detection resolves symlinks to a real `Cellar/codegraph/<version>` path, not a path-prefix string match (see Pitfall 8) |
| cosign/SLSA verification ↔ notarization | Assuming notarization "does something" for the cosign/SLSA-verified raw-binary path | It does nothing — notarization and Gatekeeper are Apple-specific, cosign/SLSA are supply-chain provenance for a different threat model and a different consumer (`internal/upgrade`, not Gatekeeper). PROJECT.md itself already states this ("cosign is a *different* mechanism and does nothing for Gatekeeper") — do not let a notarization PR description imply it strengthens the existing attestation chain, it's orthogonal |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing the Developer ID Application `.p12` and App Store Connect API key as long-lived GitHub Actions secrets without rotation tracking | A leaked/stale secret silently breaks notarization (submission rejected) or, worse, is usable by anyone with repo-secret access to notarize arbitrary binaries under this identity | Track cert/key expiry explicitly (Apple Developer certs typically expire annually); add an explicit CI failure mode check (notarization step failing with an identity/auth error, not a build error) to the release runbook so cert rotation is caught fast, not discovered on a broken release |
| Assuming a Team ID mismatch "can't happen" because there's only one Apple Developer Program membership | If the App Store Connect API key or the Developer ID cert is ever regenerated under a different team context (e.g. after an Apple account restructuring), notarization fails with an opaque "Team is not yet configured for notarization" error that looks like an infra issue, not an identity issue | When notarization first fails post-setup, check team configuration/API-key-team association before assuming a pipeline bug — per Apple's own forum guidance, this specific error class routes to Developer Programs Support, not a technical fix |
| Treating a brew-detection bypass (a user manually placing the binary at a Homebrew-looking path without actually being brew-managed) as out of scope | Low severity, but a false-positive "refuse to upgrade" for a non-brew user who happens to have `/opt/homebrew/bin` in PATH is a real usability regression, and a false-negative (fails to detect a real brew install) lets `codegraph upgrade` corrupt the Cellar | Detect via resolving the running binary's real path (`os.Executable()` + `filepath.EvalSymlinks`) and checking whether it resolves into an actual `Cellar/<formula>/<version>/bin/` structure — not a bare prefix string match (see Pitfall 8) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| `codegraph upgrade` under Homebrew prints a generic "refused" message | User doesn't know what to do next | Match PROJECT.md's own committed UX: explicitly point at `brew upgrade codegraph` |
| A user downloads the raw binary from GitHub Releases via browser (triggering quarantine) instead of the archive | Binary fails Gatekeeper on first run even though the pipeline is fully correct, because the raw binary was never notarized/stapled (only the archive was, by design — Pitfall 4) | Release notes / README must clearly steer browser-downloaders to the archive (`.zip`) asset, not the raw binary, for interactive/GUI use; the raw binary remains correctly intended only for `codegraph upgrade`'s non-browser fetch path |
| `brew install codegraph` succeeds but the binary silently fails at first run due to an untested Hardened Runtime interaction (Pitfall 7) | Worse than a `curl`-downloaded failure, because Homebrew users have a strong trust prior that `brew install` "just works" | Full functional smoke test (not just `spctl`) against the exact bottle/binary Homebrew would install, before considering the notarization phase done |

## "Looks Done But Isn't" Checklist

- [ ] **Notarization pipeline green:** Often missing the forced-quarantine `spctl` check against the actual published archive — verify with `xattr -w com.apple.quarantine ...` + `spctl -a -vv -t exec` on the real downloaded asset, not a local `dist/` file (Pitfall 3)
- [ ] **`goreleaser release` migration "complete":** Often missing removal of the now-redundant hand-rolled checksum step, leaving two checksum generators racing (Pitfall 5) — verify by diffing the published checksums file's line count against the real asset list
- [ ] **Homebrew tap "working":** Often verified only by a manual `brew install` run once, right after a release, when GitHub's CDN/API is warm — verify by testing `brew install` cold, some time after a release, and after a formula update from a *second* subsequent release (catches livecheck/version-bump edge cases)
- [ ] **`codegraph upgrade` brew-refusal "tested":** Often tested only against a hand-constructed fake path (e.g. an env var or a literal `/opt/homebrew/` string check) rather than a real `brew install` followed by running the actual binary — verify against a genuine `brew tap` + `brew install` on a real machine, per PROJECT.md's own stated bar ("Detection must be tested against a real brew-managed layout, not a path-prefix guess")
- [ ] **Archive asset "notarized":** Often means "the binary inside the archive was signed" — verify the *archive itself* was submitted to and accepted by `notarytool`, and that `xcrun stapler validate` succeeds on the archive, not just that `codesign --verify` succeeds on the inner binary (Pitfall 4)
- [ ] **cosign/SLSA verification "still passes" post-migration:** Often re-run only against local build output, not against the actual published release — extend `TestVerifyReleaseE2E`-style checks to run after every future `goreleaser release`-based release, not just the first one during development (Pitfall 6)

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|----------------|------------------|
| Checksums collision published in a real release (Pitfall 5) | MEDIUM | Delete the wrong checksums asset via `gh release delete-asset`, regenerate the correct one from the actual published binaries, re-upload, and re-run SLSA provenance if the base64-subjects hash changed (may require a follow-up patch release if `internal/upgrade`'s verify path depends on it) |
| Notarized-but-not-stapled release shipped, offline users failing Gatekeeper (Pitfall 4) | LOW–MEDIUM | Notarization tickets remain valid; staple after the fact with `xcrun stapler staple` against the already-notarized archive and re-upload the asset — no need to re-notarize |
| Brew formula pointing at a since-deleted or renamed asset (tap/release race, or a force-pushed formula) | LOW | Regenerate and re-push the formula from the current release's real checksums/URLs; `brew update` picks up the corrected tap on the next run — but any user who already ran `brew install` during the broken window needs to `brew reinstall` |
| `codegraph upgrade` false-positive brew-refusal shipped (blocks a legitimate non-brew user) | LOW | Patch release with corrected detection logic; document a manual override/workaround in the interim (e.g. a documented flag or direct binary replacement instructions) |
| Developer ID cert expired mid-release-cycle, notarization pipeline broken until renewed | MEDIUM (external dependency on Apple's cert issuance turnaround) | Fall back to shipping unnotarized archives temporarily (raw binaries + cosign/SLSA path is entirely unaffected and continues to work for `codegraph upgrade` users) while the cert is renewed; communicate the temporary Gatekeeper-friction to browser-downloading users |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification (RED-demonstrable) |
|---------|-------------------|-----------------------------------|
| 1. Single-runner conflict / Pro-only split-merge | `goreleaser release` migration phase | Reproduce the dist-exists error live with a two-job pattern before committing to final runner topology; confirms single-macOS-runner + zig-cc-for-linux is required |
| 2. Notarization Accepted but `spctl` rejects | Notarization phase | `spctl -a -vv -t exec` on the real published archive after forcing quarantine, expect `accepted` + `source=Notarized Developer ID` |
| 3. "Wired up but cannot fire" false-positive checks | Notarization phase (acceptance gate, not a side check) | The forced-quarantine `spctl` sequence is the phase's UAT criterion; every other check (codesign -dvv, notarytool history, green CI) explicitly documented as insufficient in the phase's own verification notes |
| 4. Stapling requires a container | Notarization phase + Archives phase (shared decision) | `xcrun stapler staple` on the raw binary must fail (proves constraint); `xcrun stapler validate` on the shipped archive must succeed |
| 5. Checksums-file collision | `goreleaser release` migration phase | Diff published checksums file's covered-file list against `gh release view --json assets` after first real release under the new pipeline |
| 6. cosign/SLSA-attested bytes diverge from published bytes | `goreleaser release` migration phase + Notarization phase | Extend `TestVerifyReleaseE2E` to assert raw-binary bytes are untouched between cosign-signing time and publish time, run against every future release, not just once |
| 7. Hardened Runtime entitlements | Notarization phase | Full CLI/MCP functional smoke test against the actual signed+notarized binary with default (no extra entitlement) Hardened Runtime options; a dyld/codesign crash is the signal an entitlement is needed |
| 8. Brew-managed-install detection fragility (see Integration Gotchas) | `codegraph upgrade` brew-detection phase | Test against a real `brew tap` + `brew install` on a real machine (Apple Silicon `/opt/homebrew` at minimum; Intel `/usr/local` and linuxbrew as available), resolving symlinks to a real Cellar path rather than string-matching a prefix |
| Self-update-vs-Cellar conflict (Homebrew policy) | `codegraph upgrade` brew-detection phase | Confirm `brew audit --new --formula codegraph` does not flag self-update behavior — Homebrew's own Acceptable-Formulae audit explicitly checks for this policy area |
| Tap push racing release asset publish | Homebrew tap phase | Verify GoReleaser's own within-run publish ordering (`release` before `brews`) is preserved if brews publishing is ever separated into its own job; add a real cold `brew install` test run some time after a release, not immediately after, to catch propagation-timing issues |

## Sources

- [Notarization successful but spctl … | Apple Developer Forums (thread 128497)](https://developer.apple.com/forums/thread/128497) — MEDIUM-HIGH; documents Accepted-but-rejected family, notary ticket missing Mach-O images
- [Gatekeeper rejects notarized app | Apple Developer Forums (thread 794080)](https://developer.apple.com/forums/thread/794080) — MEDIUM-HIGH
- [spctl --type install rejects notarized .pkg on macOS 26 Tahoe | Apple Developer Forums (thread 817887)](https://developer.apple.com/forums/thread/817887) — MEDIUM; recent (Tahoe-era) corroboration the failure family is still live
- [App Fails spctl After signing and notarization | Apple Developer Forums (thread 767998)](https://developer.apple.com/forums/thread/767998) — MEDIUM
- [Notarize macOS Applications – GoReleaser official docs](https://goreleaser.com/customization/sign/notarize/) — HIGH; App-Bundle-inner-binary-only trap, native vs quill methods, requires macOS runner for native path
- [Notarized MacOS application blocked by Gatekeeper when downloaded | Apple Developer Forums (thread 706638)](https://developer.apple.com/forums/thread/706638) — MEDIUM
- [Apple Codesigning In Depth: Part I — Kayla McArthur](https://kayla.is/posts/codesigning-part-i/) — MEDIUM; codesign vs spctl distinction, ad-hoc signature behavior
- [macOS distribution gist — rsms](https://gist.github.com/rsms/929c9c2fec231f0cf843a1a746a416f5) — MEDIUM; community-compiled but cross-corroborated with official sources
- [Split & Merge – GoReleaser official docs](https://goreleaser.com/customization/general/partial/) — HIGH; confirms Pro-only, explains split/merge mechanics
- [GoReleaser Split and Merge — Carlos Becker (GoReleaser maintainer's own blog)](https://carlosbecker.com/posts/goreleaser-split-merge/) — HIGH; maintainer-authored, directly authoritative
- [Release Merged Builds / Using Existing Builds During Release · Issue #2320 · goreleaser/goreleaser](https://github.com/goreleaser/goreleaser/issues/2320) — HIGH; official GitHub issue confirming `release` cannot consume pre-existing `dist/`, no `--skip-build` flag exists
- [Multiple brew formulas fail to upload to the same repository · Issue #1120 · goreleaser/goreleaser](https://github.com/goreleaser/goreleaser/issues/1120) — MEDIUM-HIGH; documented parallel-tap-push race
- [Git is in a dirty state – GoReleaser official error docs](https://goreleaser.com/resources/errors/dirty/) — HIGH
- [goreleaser release CLI reference – GoReleaser official docs](https://goreleaser.com/cmd/goreleaser_release/) — HIGH; `--skip` valid values including `notarize`, `homebrew`
- [Homebrew Documentation: Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae) — HIGH; official self-update policy, directly relevant to `codegraph upgrade` brew-refusal decision
- [Homebrew Documentation: Adding Software to Homebrew](https://docs.brew.sh/Adding-Software-to-Homebrew) — HIGH; `brew audit --new` requirement
- [M1 Mac has reverted HOMEBREW_PREFIX to /usr/local · Discussion #664 · Homebrew/discussions](https://github.com/Homebrew/discussions/discussions/664) — MEDIUM; real-world prefix-detection edge cases (migrated systems, symlink confusion)
- [HOMEBREW_PREFIX error when use `brew` symlink · Issue #16044 · Homebrew/brew](https://github.com/Homebrew/brew/issues/16044) — MEDIUM
- [How notarization works – The Eclectic Light Company](https://eclecticlight.co/2020/08/28/how-notarization-works/) — MEDIUM-HIGH; independent technical writer, cross-corroborated with Apple docs, widely cited in the macOS dev community
- [Notarization: the hardened runtime – The Eclectic Light Company](https://eclecticlight.co/2021/01/07/notarization-the-hardened-runtime/) — MEDIUM-HIGH
- [Disable library validation entitlements makes app fail GateKeeper | Apple Developer Forums (thread 673889)](https://developer.apple.com/forums/thread/673889) — MEDIUM
- [Notarization says I'm not member of my team | Apple Developer Forums (thread 119445)](https://developer.apple.com/forums/thread/119445) — MEDIUM; Team ID mismatch symptom class
- [Error 7000 "Team is not yet configured for notarization" | Apple Developer Forums (thread 814080)](https://developer.apple.com/forums/thread/814080) — MEDIUM
- [Building and notarizing command tools as Universal binaries – The Eclectic Light Company](https://eclecticlight.co/2020/08/27/building-and-notarizing-command-tools-as-universal-binaries/) — MEDIUM-HIGH; directly relevant CLI-tool (not .app) notarization guidance
- [Possible to notarize only a single … | Apple Developer Forums (thread 131610)](https://developer.apple.com/forums/thread/131610) — MEDIUM; confirms notarize-the-container-not-the-binary requirement, stapling-to-single-Mach-O impossibility
- `.goreleaser.yaml` and `.github/workflows/release.yml` (this repo, read directly 2026-08-07) — HIGH; ground truth for the checksums-collision and asset-naming findings

---
*Pitfalls research for: macOS Gatekeeper notarization + Homebrew tap distribution, added to an existing signed/attested Go release pipeline*
*Researched: 2026-08-07*
