---
phase: 09-release-please-and-goreleaser
verified: 2026-08-01T00:00:00Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 9: release-please + GoReleaser Verification Report

**Phase Goal:** Replace the hand-rolled, maintainer-manual tagging step with automated release management — release-please owns version bumps, `CHANGELOG.md`, and tag creation from Conventional Commits; GoReleaser owns building and uploading artifacts.
**Verified:** 2026-08-01
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `release-please-config.json` (`release-type: go`) + `.release-please-manifest.json` exist; merging a release PR cuts the tag without a human running `git tag` | VERIFIED | Both files present with correct content (`release-type: "go"`, manifest `"." : "0.2.0"`). `gh release view v0.2.0` shows `author.login = fzy-release-please[bot]`; `git for-each-ref refs/tags/v0.2.0` shows tagger `fzy-release-please[bot]`. No human ran `git tag`. |
| 2 | The cosign OIDC cert SAN still matches `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern` — `codegraph upgrade` keeps working for every already-shipped binary | VERIFIED | Pattern unchanged: `^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$` (verify.go:44). Empirically confirmed both in-session and via 09-08-SUMMARY: a genuinely shipped v0.1.0 binary (built 2026-07-14, pre-rewiring) ran `codegraph upgrade` and verified the v0.2.0 signature produced by the App-triggered run — cross-checked independently this session on darwin/arm64 and linux/amd64, with `cosign verify-blob` → `Verified OK` and `TestVerifyReleaseE2E` passing against the production issuer/SAN. |
| 3 (AMENDED 2026-07-28) | Artifacts are uploaded into the release-please-created GitHub Release idempotently via `release.yml`'s publish step (`gh release upload --clobber`), with `goreleaser build --single-target` retained and no migration to `goreleaser release` | VERIFIED | `release.yml`'s `assemble` job (lines 277-323) branches on `gh release view` exit status (never `\|\|`-chained) and runs `gh release upload "$TAG" --repo "$REPO" --clobber codegraph_*` on the found-release path. `.goreleaser.yaml` confirms `archives:`/`checksum:` blocks are documented dead config; `release.yml`'s build job invokes only `goreleaser build --single-target --clean` (line 132), never `goreleaser release`. The amendment itself is properly recorded (not silently applied) in ROADMAP.md lines 358-374 and referenced to `docs/RELEASE-PROCEDURES.md` §10(a) and `09-CONTEXT.md` D-05. 09-08-SUMMARY independently proved, via three lines of evidence (timing, exit-status branch structure, and body-content discriminator), that the v0.2.0 publish actually took the UPLOAD (`--clobber`) branch, not the create branch. |
| 4 | Per-binary cosign signing + SLSA provenance + the native 2-OS CGo matrix are all preserved | VERIFIED | `release.yml`'s `assemble` job signs each binary individually (`cosign sign-blob --bundle="${f}.sigstore.json" --yes "$f"`, line 251) — matches the requirement that `defaultVerify` hashes the downloaded binary itself. `provenance` job uses `slsa-framework/slsa-github-generator/.../generator_generic_slsa3.yml@v2.1.0` over the 6-binary subject list (lines 332-357). `build` job matrix builds all 6 platforms natively — ubuntu-latest for linux (native amd64, zig-cross arm64) + both windows targets (zig-cross), macos-latest natively for both darwin arches (lines 61-100). Confirmed live: v0.2.0 release has 20 assets (6 binaries + 6 `.sigstore.json` + 6 `.spdx.json` SBOMs + checksums + provenance bundle) matching a derived 6-platform expectation by set equality (09-UAT test 36). |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `release-please-config.json` | `release-type: go`, no `release-as` sticky key | VERIFIED | Present, `release-type: "go"`, packages `{".":{}}` |
| `.release-please-manifest.json` | seeded and updated by bot | VERIFIED | `{".": "0.2.0"}` — bumped by release-please from `0.1.0` baseline |
| `.github/workflows/release-please.yml` | App-token tag/PR automation, no build/sign | VERIFIED | 43 lines: pre-tag 6-target `go list` gate + `release-please-action` with App-minted token; builds/signs nothing itself |
| `.github/workflows/release.yml` | tag-triggered build/sign/SBOM/publish/provenance, LOCKED SAN contract | VERIFIED | 358 lines: `on: push: tags: v[0-9]*`, 6-leg native matrix, per-binary cosign, syft SBOM, idempotent publish, SLSA generic generator |
| `.goreleaser.yaml` | `build --single-target` config, dead archive/checksum blocks documented | VERIFIED | 6 build IDs, header comments explicitly document `archives:`/`checksum:` as dead config under this pipeline |
| `internal/upgrade/verify.go` | `releaseWorkflowRefPattern` unchanged, anchored full-match | VERIFIED | Line 44, matches `release.yml` filename/trigger exactly |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `release-please.yml` push to `main` | tag `v0.2.0` created | App-token `release-please-action` | WIRED | Confirmed live: tag authored by `fzy-release-please[bot]`, App token (not default `GITHUB_TOKEN`) — the load-bearing distinction documented in-file, since default-token-authored refs don't trigger other workflows |
| tag push `v0.2.0` | `release.yml` fires | `on: push: tags: v[0-9]*` trigger | WIRED | Run `30675077940`, `event=push ref=v0.2.0`, 11/11 jobs success |
| `assemble` job | cosign/SBOM/publish | per-binary loop + `gh release upload --clobber` | WIRED | 20 assets published; upload branch confirmed taken via 3-line-of-evidence proof in 09-08-SUMMARY |
| `assemble` job checksums | `provenance` job | `base64-subjects` output passing 6-binary hash list | WIRED | `TestVerifyReleaseE2E` and independent `slsa-verifier` checks confirm the attested subjects are the 6 binaries |
| shipped v0.1.0 binary | `codegraph upgrade` → v0.2.0 | `internal/upgrade` + unchanged `releaseWorkflowRefPattern` | WIRED | Verified on two platforms (darwin/arm64, linux/amd64) this session: installed sha == checksums entry == SLSA subject, `cosign verify-blob` → `Verified OK` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Release authored by bot, not human | `gh release view v0.2.0 --json author,tagName` | `author.login = fzy-release-please[bot]` | PASS |
| release.yml run succeeded end to end | `gh run view 30675077940 --json jobs` | 11/11 jobs `conclusion: success` | PASS |
| Asset contract matches 6-platform expectation | `gh release view v0.2.0 --json assets` | 20 assets, exact set match | PASS |
| Pattern in verify.go anchored/unchanged | `rg releaseWorkflowRefPattern internal/upgrade/verify.go` | full-match regex present, ties to `release.yml` | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| REL-02 | 09-01..09-08 | Automated release cut (version/changelog/tag) without human `git tag`, signed artifacts still satisfy cosign identity | SATISFIED | v0.2.0 cut end-to-end by release-please + release.yml; 09-08-SUMMARY + independent re-measurement this session confirm both halves |

**Note (non-blocking):** `.planning/REQUIREMENTS.md` line 89 still shows REL-02 as an unchecked `[ ]` item with stale trailing text ("the live App-token tag cut and signed-artifact proof are still pending"), and the summary table (line 178) still reads "In Progress (09-01/08 plans complete)." Both predate 09-08's completion (2026-08-01) and are now factually stale relative to the codebase and 09-08-SUMMARY/09-UAT evidence. ROADMAP.md's Phase 9 heading checkbox (line 45) is similarly still unchecked. This is a documentation-sync gap, not a functional gap — the underlying requirement is demonstrably satisfied in the live system. Recommend updating REQUIREMENTS.md and the ROADMAP Phase 9 checkbox to reflect completion as part of closing this phase.

### Anti-Patterns Found

None. Scanned `release-please-config.json`, `.release-please-manifest.json`, `.github/workflows/release.yml`, `.github/workflows/release-please.yml`, `.goreleaser.yaml`, `internal/upgrade/verify.go` for `TBD`/`FIXME`/`XXX` debt markers — zero matches. `.goreleaser.yaml`'s dead `archives:`/`checksum:` blocks are explicitly documented in-file as intentional (WR-01, Phase 8 re-review) rather than an unexplained stub.

### Human Verification Required

None. This phase was already subject to extensive UAT (09-UAT.md: 42/42 pass, 0 issues, 0 gaps) covering both the mechanical pipeline and the live end-to-end release/upgrade chain, cross-checked independently in this verification session against two platforms (darwin/arm64, linux/amd64) with matching results.

### Deferred / Contextual Notes (not gaps)

- **09-07 was deliberately SKIPPED** with a full recorded maintainer rationale (`09-07-SUMMARY.md`, status: SKIPPED) — not an incomplete plan. Its disposable-rehearsal purpose was superseded by 09-08 being the pipeline's first live run, and the argument it would have proven was independently retro-confirmed by 09-08.
- **Success criterion 3 was formally amended 2026-07-28** — verified against the amended wording (idempotent `gh release upload --clobber`, `goreleaser build --single-target` retained), not the original engram-modeled wording. The amendment is properly recorded in ROADMAP.md (lines 358-374), not silently applied.
- **REL-02 names no version** (recategorized from event to property, 2026-07-28) — the first automated cut being `0.2.0` rather than `1.0.0` is by design (D-06R), not a gap.
- Pre-existing `internal/daemon` test flake and `CheckRegression`'s missing `Metrics.Repo` comparison are known, previously-recorded issues unrelated to this phase's scope — not scored here.

## Gaps Summary

No gaps found. All four ROADMAP success criteria for Phase 9 are independently verified against live codebase and GitHub state: the config/manifest exist and drove a real bot-authored tag; the cosign SAN pattern is unchanged and empirically still satisfies pre-rewiring binaries; the amended idempotent-publish criterion holds with the amendment properly recorded; and per-binary signing, SLSA provenance, and the native 6-target matrix are all live and correct in the v0.2.0 release. The only finding is a non-blocking documentation-sync note: `.planning/REQUIREMENTS.md` and the ROADMAP Phase 9 heading checkbox have not yet been updated to reflect 09-08's 2026-08-01 completion.

---

_Verified: 2026-08-01_
_Verifier: Claude (gsd-verifier)_
