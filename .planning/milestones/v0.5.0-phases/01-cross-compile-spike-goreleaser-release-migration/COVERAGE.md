# API Coverage — Phase 1 (Cross-Compile Spike & `goreleaser release` Migration)

Phase 1 is where this repository's external-API verification machinery originates.
Phases 2 and 3 both opt out by deferring to it — Phase 2's declaration names
`gh release download`, `cosign verify-blob`, and `gh attestation verify` as
"pre-existing verification machinery from Phase 1, not new capability surface."
That deferral needs a declaration to defer *to*. This is it.

Every capability is either INTEGRATE (consumed) or OPT-OUT (considered and
declined, with the reason). Full coverage is the default; nothing is left
unclassified. Call sites are enumerated below the matrix.

## Detector outcome (recorded, not assumed)

The `api-coverage` detector was run at seal time against this phase directory and
returned `{"block": true, "detected": true, "coverage_present": false, "signals":
[{"verb": "(surface)", "noun": "api"}]}`.

Unlike Phases 2 and 3 — where the detector returned `detected: false` and the
declaration was written pre-emptively — this phase's detection is a **true
positive**. The surface below is real, is consumed by shipped code, and had no
declaration until now.

## Coverage matrix

| capability | decision | reason |
|---|---|---|
| GitHub Releases API — publish | INTEGRATE | GoReleaser's `release:` pipe, explicitly pinned: `replace_existing_artifacts: true`, `prerelease: auto`, no name_template/header/footer/draft/disable key (D-06R). Tag-push only. |
| GitHub Releases API — read published assets | INTEGRATE | Re-downloads PUBLISHED assets, never a local `dist/` copy — a local copy never passes through the real upload path, the transparency log, or the attestation subject binding. |
| GitHub REST — refs, tags, commits | INTEGRATE | Tag resolution and validation in `post-release-verify.yml`'s `resolve-tag` job. `$TAG` is resolved and validated there and passed downstream, never re-read from `head_branch`. |
| GitHub Attestations API — write provenance | INTEGRATE | `actions/attest-build-provenance` v4.2.2, SHA-pinned, as the last step of the single GoReleaser job, over the 8-payload subject set the checksums file covers. Replaced SLSA (D-09). |
| GitHub Attestations API — verify provenance | INTEGRATE | `gh attestation verify` is the ONLY published verification command for releases from the migrated pipeline onward (D-10). |
| GitHub Actions OIDC token endpoint | INTEGRATE | `id-token: write` is held by EXACTLY ONE job, no allowance elsewhere (D-11), enforced structurally by `TestOIDCWriteScopedToSingleGoreleaserJob` rather than by convention. |
| Sigstore — Fulcio CA and Rekor log | INTEGRATE | Reached transitively through `cosign`, never directly. Keyless signing via the OIDC token above; verification pins BOTH issuer and certificate identity — a bare `verify-blob` accepts anyone's. |
| `slsa-github-generator` + `slsa-verifier` | OPT-OUT | Declined and replaced (D-09/D-10). `slsa-verifier verify-artifact` ARCHITECTURALLY CANNOT verify `attest-build-provenance` output — different format, different location. Not a preference. |
| GoReleaser Pro | OPT-OUT | Declined, not purchased. Held as a costed fallback if REL-05 failed; REL-05 PASSED on V1, first dispatch (canary run 31273571889), so the OSS single-runner path stands. |
| Apple notarization / App Store Connect API | OPT-OUT | Out of scope here; delivered by Phase 2. Phase 1 deliberately shipped `v0.5.1` with UN-NOTARIZED darwin assets, reserved as Phase 2's SIGN-03 RED baseline. |
| Homebrew tap / cask publishing | OPT-OUT | Out of scope for this phase; delivered by Phase 3. The `homebrew_casks:` block now present in `.goreleaser.yaml` belongs to Phase 3, not here. |
| GitHub Packages / OCI container registries | OPT-OUT | Declined. This project ships platform binaries and archives; no OCI artifact is built or published, so there is no registry surface to integrate against. |
| GitHub Releases API — delete / re-cut | OPT-OUT | Declined by policy, not merely unused (D-12): never delete or re-push a published release or tag. Recovery is patch-forward, per `docs/RELEASE-PROCEDURES.md`. |

## Call sites

- **Releases — publish:** `.goreleaser.yaml` `release:` block; `Taskfile.yml` `release:goreleaser`. Fires only from the tag-push `release` job.
- **Releases — read:** `Taskfile.yml` `verify:release-assets` and `verify:self-upgrade` — `gh release download` ×13, `gh release view --json assets` ×4, `gh api repos/{REPO}/releases --paginate`.
- **GitHub REST — refs/tags/commits:** `.github/workflows/post-release-verify.yml` — `repos/{REPO}/tags --paginate`, `repos/{REPO}/git/ref/tags/{TAG}`, `repos/{REPO}/commits/{TAG}`.
- **Attestations — write:** `.github/workflows/release.yml:289`, `actions/attest-build-provenance@4d101475d8b20a2381f78447822ac1eab6504dd8` (v4.2.2, SHA-pinned with a trailing version comment); `attestations: write` declared at `:98`.
- **Attestations — verify:** `Taskfile.yml` `verify:release-assets`; `.github/workflows/post-release-verify.yml`; published instructions in `docs/RELEASE.md` §1(b) and `docs/RELEASE-PROCEDURES.md`.
- **OIDC:** `.github/workflows/release.yml`, one `id-token: write` holder. Granting it to any `pull_request`-triggerable workflow would let any PR mint a token in the repo identity — the reason `check:darwin-release-build` was unwired from the darwin canary rather than given the permission.
- **Sigstore:** `.goreleaser.yaml` `signs:` (release-scoped, keyless); `cosign verify-blob` ×12 across 3 `Taskfile.yml` call sites, each carrying `--certificate-oidc-issuer` and `--certificate-identity-regexp`; `cosign sign-blob` in `release:dry-run-signed` against a throwaway local key, no OIDC.

## Boundary note

Every read-path integration above verifies integrity BEFORE the downloaded
artifact is made executable, never after. Parallel independent CI jobs give no
ordering guarantee, so a tampered asset would otherwise be executed even if a
sibling supply-chain job's own verification later failed.

## Provenance of this declaration

Written retroactively on 2026-08-11 during `/gsd-verify-work 1`, which halted on
the `api-coverage` `verify:pre` gate. Phases 2 and 3 wrote theirs at plan time;
this phase predates the gate firing against it. Every claim above was checked
against the tree at HEAD `42427c5` — call sites were counted, not recalled, and
the `notarize:`/`homebrew_casks:` blocks now present in `.goreleaser.yaml` were
confirmed to belong to Phases 2 and 3 rather than being claimed here.
