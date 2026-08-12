---
phase: 02-apple-signing-notarization
reviewed: 2026-08-09T23:01:40Z
depth: deep
files_reviewed: 13
files_reviewed_list:
  - .github/workflows/post-release-verify.yml
  - .github/workflows/release.yml
  - .goreleaser.yaml
  - docs/RELEASE-PROCEDURES.md
  - docs/RELEASE.md
  - internal/upgrade/goreleaser_shape_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - internal/upgrade/taskfile_shape_test.go
  - Taskfile.yml
  - test/integration/binpath_test.go
  - test/integration/main_test.go
  - test/wireoracle/binpath_test.go
  - test/wireoracle/main_test.go
findings:
  critical: 0
  warning: 3
  info: 1
  total: 4
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-08-09T23:01:40Z
**Depth:** deep
**Files Reviewed:** 13
**Status:** issues_found

## Summary

This phase's deliverable is the Apple signing/notarization pipeline: `.goreleaser.yaml`'s `notarize:`/`signs:` blocks, `release.yml`'s single OIDC-bearing job, `post-release-verify.yml`'s five-job re-verification workflow, the `Taskfile.yml` targets those workflows call (`verify:release-assets`, `verify:self-upgrade`, `verify:gatekeeper`, `verify:notarized-suite`, `release:rehearse-notarize`), the two shape-test files that pin the config's machine-checkable invariants, and the two test harnesses (`test/integration`, `test/wireoracle`) that prove the notarized binary is exercised for real.

The security-relevant invariants called out in the review brief hold up under inspection: `signs:` (not `binary_signs:`) is correctly used and tested (`TestSignsSidecarMatchesUpgradeContract`); the `notarize.macos[0].enabled` five-term conjunction is a real `AND`, tested under all 7 controlled environments; `spctl -a -vv -t install` (never `-t exec`) derives its verdict from exit status alone, never substring search (`Taskfile.yml`'s `verify:gatekeeper`, cross-checked against `docs/RELEASE.md` §1d); stapling is documented as out of scope everywhere it's mentioned; the five Apple credential secrets are step-level-only, scoped to the single `release` job, and `TestAppleSecretsScopedToSingleReleaseJob` mechanically forbids their appearance on any `pull_request`/`pull_request_target`-triggerable workflow; the `[ cond ] && VAR=x`-as-last-loop-statement footgun is explicitly called out and avoided (converted to `if`/`fi`) in `release:rehearse-notarize`, with a comment recording the prior incident. `post-release-verify.yml`'s `if:` guards are consistently event-aware (never the bare `conclusion == 'success'` form that would silently skip under `workflow_dispatch`), and every `gh`-shelling step carries its own `GH_TOKEN`.

Three issues surfaced from deeper cross-file tracing, none rising to BLOCKER: a documentation/comment drift in two reviewed files that still describes the pre-D-18 (and specifically wrong) signing pipe as current; a latent pagination-correctness risk in `resolve-tag`'s rarely-exercised fallback branch; and a coverage gap where two of the four release platforms' cosign signatures are never independently re-verified by any automated post-release job.

## Warnings

### WR-01: Stale `binary_signs:` references contradict the current (same-phase) D-18 signing pipe

**File:** `.github/workflows/release.yml:34-39, 131-134`
**File:** `docs/RELEASE-PROCEDURES.md:130-133`

**Issue:** `.goreleaser.yaml`'s own header comment records D-18 (LOCKED, maintainer, 2026-08-09) as moving cosign signing from the build-scoped `binary_signs:` pipe to the release-scoped `signs:` pipe — specifically *because* `binary_signs:` ran before `notarize:` and therefore signed pre-notarization bytes, a real ordering bug this phase fixed. `internal/upgrade/goreleaser_shape_test.go`'s `TestSignsSidecarMatchesUpgradeContract` and `release_workflow_shape_test.go` correctly test against `signs:`.

However, two of the files in this review's own scope still describe `binary_signs:` as the pipe currently in effect, not as history:

- `release.yml:38`: "`.goreleaser.yaml`'s `binary_signs:` pipe (plan 01-02), not a shell loop in this file."
- `release.yml:133`: "Moved up from the deleted assemble job: `.goreleaser.yaml`'s `binary_signs:` pipe shells out to cosign, so it must be on PATH before goreleaser runs."
- `docs/RELEASE-PROCEDURES.md:130-133`: "**Sign** — `binary_signs:` shells out to cosign keyless (`sign-blob --bundle=...`) per binary — internal/upgrade hashes the downloaded binary itself, not a checksums file, so per-binary signing is required, same trust boundary as before."

None of these three passages is guarded by a shape test — `TestNoGoreleaserHooksInReleaseConfig` and friends check `.goreleaser.yaml`'s actual structure, not prose in `release.yml`'s comments or `RELEASE-PROCEDURES.md`. A maintainer debugging a future signing-order regression, or "fixing" `release.yml`'s comment to match `RELEASE-PROCEDURES.md`, would be pointed at the wrong (already-rejected) mechanism by two independent, in-scope files.

**Fix:** Update all three passages to name `signs:` (release-scoped), matching `.goreleaser.yaml`'s own comments and `docs/RELEASE.md` (which already correctly describes the current pipe). If the historical `binary_signs:` shape is worth preserving as context, phrase it explicitly as "formerly `binary_signs:`, moved to `signs:` per D-18" rather than as present-tense fact.

### WR-02: `resolve-tag`'s tag-by-commit fallback wraps a `gh api --paginate --jq` result in a JSON array, which does not merge cleanly across pages

**File:** `.github/workflows/post-release-verify.yml:158-165`

**Issue:** When `head_branch` is empty or not `v[0-9]*`-shaped, `resolve-tag` falls back to:

```sh
MATCHES_JSON="$(gh api "repos/${REPO}/tags" --paginate --jq --arg sha "${HEAD_SHA}" \
  '[.[] | select(.commit.sha==$sha) | .name] | map(select(test("^v[0-9]")))')"
MATCH_COUNT="$(printf '%s' "${MATCHES_JSON}" | jq 'length')"
```

`gh api --paginate` combined with `--jq` applies the jq filter to *each page independently* and concatenates the per-page outputs — it does not first merge all pages into one JSON array and then apply the filter once. Because this filter wraps its result in `[...]`, a repository whose `/tags` endpoint spans more than one page (default page size is 30) would produce `MATCHES_JSON` as multiple concatenated JSON *documents* (e.g. `[]\n["v1.2.3"]\n[]`) rather than one JSON array. `jq 'length'` over that multi-document stream then emits multiple lines (e.g. `0\n1\n0`), and the subsequent `[ "${MATCH_COUNT}" != "1" ]` string comparison would treat the whole multi-line value as unequal to `"1"` even when exactly one real match exists on one page — producing a confusing hard failure ("expected exactly one v[0-9]* tag ... found 0\n1\n0") instead of correctly resolving the tag.

This only affects the fallback branch (exercised when `head_branch` is not already tag-shaped), and it fails loud rather than silently picking the wrong tag — but it is a real correctness bug in the one place this file most needs to be right: automated, unattended post-release verification. It is currently latent because this repository's tag count is well under the pagination threshold, but will resurface without warning once the repository accumulates enough `v[0-9]*` tags (release + rc tags together).

**Fix:** Either drop `--paginate` here (tags endpoints are rarely large enough to need it, and a hard cap is acceptable for a fallback path) and rely on a single page, or restructure to merge pages before filtering, e.g. collect raw pages first (`gh api ... --paginate` with no `--jq`, letting `gh`'s built-in array-merge for un-filtered list responses combine pages), then pipe the merged array through `jq` once as a separate step — mirroring how `verify:self-upgrade`'s `RELEASES_JSON="$(gh api "repos/${REPO}/releases" --paginate)"` (no `--jq` on the `gh api` call itself) already avoids this exact hazard two call sites away.

### WR-03: Two of the four release platforms' cosign signatures are never independently re-verified by any automated post-release job

**File:** `Taskfile.yml:1886-1912` (`verify:release-assets`), `Taskfile.yml:2422-2458` (`verify:notarized-suite`), `.github/workflows/post-release-verify.yml:245-259` (`self-upgrade` matrix)

**Issue:** Tracing which job verifies which platform's cosign bundle:

- `verify:release-assets` hardcodes exactly one asset: `codegraph_${TAG}_linux_amd64` (line ~1896-1911) — cosign `verify-blob` and `gh attestation verify` both run only against this one platform.
- `self-upgrade`'s matrix covers `darwin/arm64` and `linux/amd64` only (`post-release-verify.yml:251-258`) — each leg's `codegraph upgrade` call does perform an in-process cosign check for its own platform, but only for those two.
- `verify:notarized-suite` downloads and cosign-verifies `darwin/arm64` only (`GOOS: darwin`/`GOARCH: arm64` pinned in the job).
- `gatekeeper`'s matrix covers both darwin arches, but its oracle is `spctl` (Apple's embedded Developer ID signature), a completely different trust mechanism from the detached cosign bundle `codegraph upgrade` actually checks.

Net effect: `codegraph_<tag>_darwin_amd64`'s cosign `.sigstore.json` bundle and `codegraph_<tag>_linux_arm64`'s cosign `.sigstore.json` bundle are never checked by *any* automated job. A corrupted, mis-signed, or tampered cosign sidecar affecting only one of those two assets would ship and remain undetected indefinitely by CI — a user on `linux/arm64` or `darwin/amd64` running `codegraph upgrade` would be the first to discover it (in the desired fail-closed way, per `internal/upgrade/verify.go`, so this is not a silent-acceptance risk to the end user) — but the phase's own stated goal (automated re-proof of REL-08's claims "on every future release," `post-release-verify.yml`'s header comment) is not fully met for these two platforms.

`docs/RELEASE.md` §1a explicitly scopes the *manual* verification bar to "at least one platform," so this may be an accepted trade-off rather than an oversight — but that manual-verification carve-out does not obviously justify the *automated* pipeline's narrower coverage, and no comment in `post-release-verify.yml` or `Taskfile.yml` states this four-platform gap as a deliberate decision the way, e.g., the darwin/amd64-execution-deferral in `notarized-suite`'s header comment does.

**Fix:** Either extend `verify:release-assets` to loop cosign `verify-blob` + `gh attestation verify` over all four raw assets (cheap — no Apple round-trip, no execution, just `gh release download` + `cosign verify-blob` four times instead of once), or add an explicit, reasoned comment recording why `linux_amd64`-only is sufficient evidence for the other three platforms' cosign bundles, matching this codebase's own convention of recording scope decisions rather than leaving them implicit.

## Info

### IN-01: `docs/RELEASE.md` and `docs/RELEASE-PROCEDURES.md` diverge on whether `binary_signs:`/`signs:` is even mentioned

**File:** `docs/RELEASE.md` (no `binary_signs:`/`signs:` reference at all — describes cosign generically)
**File:** `docs/RELEASE-PROCEDURES.md:130-133` (stale `binary_signs:` reference — see WR-01)

**Issue:** `docs/RELEASE.md` was kept current through the D-18 rename (it never names the internal pipe, so it never went stale), while `docs/RELEASE-PROCEDURES.md`'s parallel "what release.yml's one job does" narrative did drift (WR-01). The two maintainer-facing docs are inconsistent with each other on a config-internal detail that only one of them chose to expose.

**Fix:** Covered by WR-01's fix; noted separately here only to flag that the asymmetry between the two docs (one detailed, one narrative) is itself worth watching for future drift — the more detailed doc is the one that goes stale first.

---

_Reviewed: 2026-08-09T23:01:40Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
