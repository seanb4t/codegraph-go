---
phase: 01-cross-compile-spike-goreleaser-release-migration
verified: 2026-08-08T00:00:00Z
status: passed
score: 4/4 roadmap success criteria verified; 5/5 requirements (REL-05..09) verified; the 1 documentation gap found was closed and re-checked (see 'Gap Closure')
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "docs/RELEASE.md § b names the literal migration-cutover tag (v0.5.1); no 'the first release cut by the migrated pipeline' placeholder remains"
    status: closed
    reason: >
      Plan 01-05's Task 3 <action> explicitly required replacing plan 01-04's
      "the first release cut by the migrated pipeline" marker with the literal
      cutover tag, and Task 3's own acceptance criteria state "no ... placeholder
      remains." 01-05-SUMMARY.md claims "PLAN COMPLETE — all three tasks" and
      documents full REL-08 evidence collection, but the doc close-out for this
      specific item was never applied to the file on disk.
    artifacts:
      - path: "docs/RELEASE.md"
        issue: "Line 76 still reads 'Releases from **`<first-migrated-release-tag>`** onward (this section will name the exact tag once plan 01-05 has cut it)' — v0.5.1 is never named."
    missing:
      - "Replace the `<first-migrated-release-tag>` placeholder in docs/RELEASE.md § b with `v0.5.1`, and drop the parenthetical referring to plan 01-05 as future work."
  - truth: "docs/RELEASE-PROCEDURES.md carries a dated entry recording that v0.5.1 is the first tag cut by the single-goreleaser-release pipeline and that its darwin assets are Phase 2's SIGN-03 RED baseline, not to be deleted/replaced/notarized in place"
    status: closed
    reason: >
      Plan 01-05 Task 3's <action> explicitly required this dated entry. §7
      (Rollback/cleanup) contains only a generic warning about "any un-notarized
      or otherwise deliberately-preserved asset a later phase depends on" with no
      v0.5.1-specific entry anywhere in the file. §10 ("Recorded divergences") is
      still scoped to Phase 9 of a prior milestone and was not extended.
    artifacts:
      - path: "docs/RELEASE-PROCEDURES.md"
        issue: "No occurrence of 'v0.5.1' or 'SIGN-03' anywhere in the file (rg -n 'v0.5.1|SIGN-03' returns nothing)."
    missing:
      - "Add a dated entry (release history or §10-style divergence note) naming v0.5.1, stating its darwin assets are deliberately un-notarized and reserved as Phase 2's SIGN-03 RED baseline."
deferred: []
---

# Phase 1: Cross-Compile Spike & `goreleaser release` Migration Verification Report

**Phase Goal:** A maintainer knows, from measurement rather than inference, whether the OSS
single-runner architecture is reachable — and the release pipeline is then a single
`goreleaser release` invocation whose published assets still satisfy every supply-chain claim
the old pipeline satisfied, while carrying `.zip` archives alongside the raw binaries
`codegraph upgrade` consumes.

**Verified:** 2026-08-08 (re-verification of live external evidence performed directly by this
agent — GitHub Actions run logs, `gh release view`, a live re-download-and-verify of `cosign
verify-blob` and `gh attestation verify` against the published `v0.5.1` release, and direct
file/grep checks of the current working tree)

**Status:** gaps_found (documentation-completeness only — every functional/security claim is
independently re-verified true)

**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (merged from ROADMAP Success Criteria 1-4, REL-05..09)

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | REL-05: pipeline architecture decided on re-inspectable evidence — both linux legs, built on one macOS host via `zig cc`, EXECUTE on real (non-emulated) Linux and index a real tree to a non-zero graph | ✓ VERIFIED | Independently re-queried canary run [31273571889](https://github.com/seanb4t/codegraph-go/actions/runs/31273571889): all 3 jobs `success` (`cross-build`, `exec (real linux/amd64)`, `exec (real linux/arm64)`). Re-extracted resolved `REL05-EVIDENCE` lines from the raw log: `uname=x86_64 elf=x86-64 files=430 symbols=4281` and `uname=aarch64 elf=aarch64 files=430 symbols=4281` — non-zero, uname/elf agree, arm64 leg ran on `namespace-profile-linux-arm64-4x8` (no emulation). |
| 2 | REL-06/07: a real release is cut by ONE `goreleaser release` invocation; `gh release view <tag> --json assets` lists exactly one `codegraph_<tag>_checksums.txt` covering every published asset exactly once; the hand-rolled `sha256sum` step and `--clobber` publisher are gone from `release.yml` | ✓ VERIFIED | `release.yml` collapsed to a single `release` job (read in full — one `run:` body, `task release:goreleaser`). `rg -c 'sha256sum\|shasum -a 256'` and `rg -c '\-\-clobber'` against `release.yml` both return 0. Downloaded the real `codegraph_v0.5.1_checksums.txt` from the published release: exactly 8 lines, one per raw binary + one per `.zip`, no sidecars, no self-reference. `gh release view v0.5.1` confirms 17 assets total (8 payloads + 8 sidecars + 1 checksums file). |
| 3 | REL-08: against assets RE-DOWNLOADED from the published release, `cosign verify-blob` returns Verified OK under the unchanged SAN, `gh attestation verify` passes, and a genuinely shipped prior binary self-upgrades via `codegraph upgrade` on darwin/arm64 and linux/amd64 | ✓ VERIFIED | Independently re-downloaded `codegraph_v0.5.1_linux_amd64` + its `.sigstore.json` and ran `cosign verify-blob` with `docs/RELEASE.md` § a's exact regex myself: `Verified OK`. Independently ran `gh attestation verify codegraph_v0.5.1_linux_amd64 --repo seanb4t/codegraph-go`: exit 0. Re-queried `post-release-verify` run [31285981504](https://github.com/seanb4t/codegraph-go/actions/runs/31285981504): all 4 jobs `success`, log contains resolved `SELF-UPGRADE-EVIDENCE` lines for both darwin/arm64 and linux/amd64 with `upgraded_sha256 == published_sha256` in both. |
| 4 | REL-09: each platform's release carries both a raw `codegraph_<tag>_<goos>_<goarch>` asset (byte-shape-identical to `releaseAssetName()`) and a distinctly-named `.zip`; a mutation off `formats: [binary]` turns a test red | ✓ VERIFIED | `.goreleaser.yaml` `archives:` has `id: raw` (`formats: [binary]`, unchanged `name_template`) and `id: zip` (`formats: [zip]`, same stem). `TestRawArchiveEntryStaysBinaryFormat`/`TestZipArchiveSharesRawAssetStem` present and passing (`go test ./internal/upgrade/...` green). Downloaded the real v0.5.1 zip and inspected it: contains `codegraph` + `LICENSE` + `README.md` + `CHANGELOG.md`, no completions, no man pages (D-16). |
| 5 | D-11: exactly one job in `release.yml` may hold `id-token: write`; D-09: `actions/attest-build-provenance` replaces the SLSA generic generator entirely | ✓ VERIFIED | `rg -n 'id-token: write' release.yml` shows one real permission declaration (line 90) and two comment-only mentions (lines 95, 117). `rg -c 'slsa-framework'` returns 0. `actions/attest-build-provenance@4d101475d8b20a2381f78447822ac1eab6504dd8 # v4.2.2` present, SHA-pinned. |
| 6 | Sign/SBOM name-collision defect class (cycle-3 review HIGH-A/HIGH-B) is fixed and dynamically proven, not just statically asserted | ✓ VERIFIED | `.goreleaser.yaml`'s `binary_signs.signature` and `sboms.documents` both use Go-template-FIELD-based (`.ProjectName`/`.Tag`/`.Os`/`.Arch` or `.ArtifactName`) templates, not `${artifact}`. Independently re-queried canary run [31282287965](https://github.com/seanb4t/codegraph-go/actions/runs/31282287965): all 4 jobs `success`; re-extracted `SIGN-EVIDENCE count=4 distinct=4` and `SBOM-EVIDENCE count=4 distinct=4` from the raw log. Real v0.5.1 release carries 4 distinct `.sigstore.json` and 4 distinct `.spdx.json` names (confirmed in the asset list). |
| 7 | Documentation close-out: `docs/RELEASE.md` § b names the real cutover tag (no placeholder remains); `docs/RELEASE-PROCEDURES.md` records v0.5.1 as the SIGN-03 RED baseline | ✗ FAILED | See Gaps below. |
| 8 | Every published verification instruction names a command that works against the new format (`gh attestation verify`, not `slsa-verifier verify-artifact`) | ✓ VERIFIED | `rg -n 'slsa-verifier verify-artifact' SECURITY.md README.md .planning/REQUIREMENTS.md` returns nothing. `docs/RELEASE.md`/`docs/RELEASE-PROCEDURES.md` retain the pre-migration command only inside explicitly-labelled historical notes. `.planning/ROADMAP.md`'s Phase 1 Success Criterion 3 (checked directly) already reads `gh attestation verify`, not the old command — the reconciliation flagged as outstanding in 01-04-SUMMARY.md has since been done. |

**Score:** 4/4 ROADMAP success criteria independently re-verified true against live external
evidence (GitHub Actions runs, the published `v0.5.1` release, and a fresh local
`cosign verify-blob`/`gh attestation verify` re-run performed by this agent, not copied from any
SUMMARY). All 5 REL-05..09 requirements are functionally and security-wise satisfied. One
narrow documentation-completeness gap found (truth #7) — a plan-scoped acceptance criterion for
plan 01-05's Task 3 that its own SUMMARY claims was completed but was not applied to the files.

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `.goreleaser.yaml` | raw+zip archives, scoped checksum, `binary_signs:`, `sboms:`, `release:` blocks, all live | ✓ VERIFIED | Read in full; matches every plan claim; `task check:goreleaser` exits 0 |
| `.github/workflows/release.yml` | single job, single runner class, native attestor, no hand-rolled checksum/sign/sbom/publish shell | ✓ VERIFIED | Read in full; one `run:` body; `slsa-framework` absent; `sha256sum`/`--clobber` absent |
| `.github/workflows/linux-cross-canary.yml` | permanent canary: cross-build + 2 real-Linux exec jobs + sign-snapshot | ✓ VERIFIED | 4 jobs present per canary run 31282287965's job list |
| `.github/workflows/post-release-verify.yml` | `workflow_run`-triggered, resolve-tag/verify-supply-chain/self-upgrade jobs, event-aware guard | ✓ VERIFIED | File exists (13.8KB), header + guard logic present; run 31285981504 shows all 4 jobs green |
| `internal/upgrade/goreleaser_shape_test.go`, `internal/upgrade/release_workflow_shape_test.go` | shape tests holding every contract, mutation-RED demonstrated | ✓ VERIFIED | `go test ./internal/upgrade/...` exits 0 |
| `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `SECURITY.md`, `README.md` | rewritten for `gh attestation verify`; RELEASE.md names the real cutover tag; RELEASE-PROCEDURES.md records the SIGN-03 baseline | ⚠️ PARTIAL | Attestor-command rewrite fully done and verified; the two remaining doc-completion items (cutover tag name, dated SIGN-03 baseline entry) are NOT done — see Gaps |
| Published `v0.5.1` GitHub release | 17 assets: 4 raw + 4 zip + 4 sigstore + 4 spdx + 1 checksums | ✓ VERIFIED | Directly queried via `gh release view v0.5.1 --json assets` |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `.goreleaser.yaml` `archives[id=raw].name_template` | `internal/upgrade.releaseAssetName()` | `TestReleaseAssetNameMatchesGoReleaser` | ✓ WIRED | Test passes; raw asset names in the real v0.5.1 release match the expected shape byte-for-byte |
| `.goreleaser.yaml` `binary_signs.signature` | `internal/upgrade`'s `.sigstore.json` download contract | `cosign verify-blob` against a re-downloaded published asset | ✓ WIRED | Independently re-run by this agent: `Verified OK` |
| `.github/workflows/release.yml`'s attest step | `.goreleaser.yaml`'s `checksum.name_template` | `TestProvenanceAttestorIsPinnedNativeAction`'s cross-file template resolution | ✓ WIRED | Test passes; `gh attestation verify` independently re-run: exit 0 |
| `post-release-verify.yml` | published GitHub release | `workflow_run` trigger (never `release: [published]`) | ✓ WIRED | Run 31285981504 confirms the trigger fired correctly and produced real evidence |
| `docs/RELEASE.md` § b | the real cutover tag | doc close-out (plan 01-05 Task 3) | ✗ NOT_WIRED | Placeholder `<first-migrated-release-tag>` never replaced — see Gaps |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full internal/upgrade shape-test suite | `go test ./internal/upgrade/...` | `ok` (0.5s) | ✓ PASS |
| GoReleaser config validity | `task check:goreleaser` | `1 configuration file(s) validated` | ✓ PASS |
| Actionlint over all workflows | `task lint:actions` | exit 0, no output | ✓ PASS |
| Live cosign verification against a re-downloaded v0.5.1 asset | `cosign verify-blob --bundle codegraph_v0.5.1_linux_amd64.sigstore.json ...` | `Verified OK` | ✓ PASS |
| Live attestation verification against a re-downloaded v0.5.1 asset | `gh attestation verify codegraph_v0.5.1_linux_amd64 --repo seanb4t/codegraph-go` | exit 0 | ✓ PASS |
| Real published checksums file coverage | downloaded `codegraph_v0.5.1_checksums.txt` | 8 lines, matches 4 raw + 4 zip | ✓ PASS |
| Real published zip contents (D-16) | downloaded and `unzip -l` on `codegraph_v0.5.1_linux_amd64.zip` | `codegraph`, `LICENSE`, `README.md`, `CHANGELOG.md` only | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| REL-05 | 01-01 | Pipeline architecture decided on measured evidence | ✓ SATISFIED | Canary run 31273571889, independently re-queried |
| REL-06 | 01-02, 01-03, 01-05 | Single `goreleaser release` invocation owns archive/checksum/sign/SBOM | ✓ SATISFIED | `.goreleaser.yaml` + `release.yml` read directly; v0.5.1 published via this exact pipeline |
| REL-07 | 01-02, 01-03, 01-05 | Exactly one checksums writer | ✓ SATISFIED | `sha256sum`/`--clobber` absent from `release.yml`; real checksums file downloaded and inspected |
| REL-08 | 01-04, 01-05 | Every supply-chain claim re-verified against published assets | ✓ SATISFIED | cosign + attestation independently re-run by this agent; self-upgrade evidence from run 31285981504 |
| REL-09 | 01-02 | Raw + `.zip` per platform, raw byte-unchanged | ✓ SATISFIED | Real v0.5.1 zip downloaded and inspected; raw asset name shape test passes |

No orphaned requirements — `.planning/REQUIREMENTS.md`'s Phase 1 mapping names exactly REL-05..09,
and all five appear in at least one plan's `requirements:` frontmatter field.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `docs/RELEASE.md` | 76 | Stale placeholder text (`<first-migrated-release-tag>`, "this section will name the exact tag once plan 01-05 has cut it") in a shipped user-facing doc, after the plan that was supposed to resolve it already ran | ⚠️ Warning | A user reading the current doc sees an obviously unfinished sentence referencing internal planning machinery ("plan 01-05") instead of the real tag. Does not affect functional correctness — the `gh attestation verify` command below it is correct and independently verified to work. |
| `docs/RELEASE-PROCEDURES.md` | — | Missing the dated SIGN-03-baseline entry Task 3 required; the information exists only in `01-05-SUMMARY.md`, not in the maintainer-facing runbook | ⚠️ Warning | Lower risk since 01-05-SUMMARY.md and this VERIFICATION.md both preserve the fact that v0.5.1's darwin assets are Phase 2's un-notarized RED baseline — but the intended single source of truth (the runbook) does not yet say so. |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase modified.

## Deviations Confirmed as Honestly Recorded

Per the `known_deviations` supplied for this verification, all four were independently
cross-checked against the codebase/history and found honestly and accurately recorded:

1. **v0.5.0 shipped empty; recovery was patch-forward to v0.5.1.** Confirmed: `gh release view
   v0.5.0` shows `isPrerelease: true, assetCount: 0`; `gh release list` shows the tag was never
   deleted (still present, chronologically between v0.4.0 and v0.5.1). No tag deletion anywhere
   in `git log`/`gh release list`.
2. **Plan 01-05's verifiers had three false negatives, fixed in PR #41 and re-dispatched green.**
   Confirmed: `git log --oneline` shows `65c4c41 ci(release): fix three false negatives in the
   post-release verifiers (#41)`; run 31285981504 (post-#41) shows all 4 jobs green.
3. **`check:darwin-release-build` unwired from darwin-toolchain-canary; coverage moved to
   `release:dry-run` in linux-cross-canary's cross-build job.** Consistent with `release:dry-run`
   being invoked inside `linux-cross-canary.yml`'s `cross-build` job (confirmed by reading
   01-01-SUMMARY.md and the canary's job list from run 31273571889/31282287965).
4. **Phase 2's SIGN-03 baseline must come from v0.5.1, not v0.5.0.** Confirmed in
   `01-05-SUMMARY.md`'s "Carry into Phase 2" section — though, per the Gaps above, this fact was
   not also propagated into `docs/RELEASE-PROCEDURES.md` as Task 3 required.

## Human Verification Required

None. Every truth was either independently re-derived by this agent against live external
evidence (GitHub Actions API, the published GitHub release, a fresh local `cosign`/`gh` run) or
directly confirmed by reading the current working-tree files and running the project's own test
suite. The two gaps found are objectively checkable (`rg` against the doc files) and require no
subjective judgment call.

## Gaps Summary

The phase's actual engineering goal — proving the OSS single-runner architecture is reachable and
migrating to a `goreleaser release`-owned pipeline whose published assets still satisfy every
supply-chain claim — is **fully achieved and independently re-verified against live evidence**,
including a real published release (`v0.5.1`, 17 assets) this agent re-downloaded and
re-verified itself (`cosign verify-blob` → `Verified OK`; `gh attestation verify` → exit 0).

The one gap found is narrow and purely editorial: plan 01-05's Task 3 explicitly required two
documentation close-out edits (naming the real cutover tag in `docs/RELEASE.md` § b, and adding a
dated SIGN-03-baseline entry to `docs/RELEASE-PROCEDURES.md`), and its own acceptance criteria
required both. `01-05-SUMMARY.md` reports "PLAN COMPLETE — all three tasks," but neither edit is
present in the current working tree. This is exactly the kind of SUMMARY-vs-codebase discrepancy
this verification process exists to catch — recorded here rather than trusted.

Recommendation: this is a trivial two-line documentation fix, not a re-open of any engineering
work. It does not block Phase 2 from starting (Phase 2's own plans should read
`01-05-SUMMARY.md`'s "Carry into Phase 2" section directly, which does correctly state the
v0.5.1/SIGN-03 fact), but should be closed — either as a fast-follow doc commit or via an
explicit override recorded in this file — before the milestone ships.

---

_Verified: 2026-08-08_
_Verifier: Claude (gsd-verifier)_

---

## Gap Closure (orchestrator, 2026-08-09)

Both documentation gaps this report found were closed after it was written. They were the only
outstanding items — every functional and supply-chain claim (REL-05…REL-09) was already proven
against live external evidence, including a real published release.

**Gap 1 — `docs/RELEASE.md` § b cutover tag.** The `<first-migrated-release-tag>` placeholder is
replaced with `v0.5.1`, and the parenthetical treating plan 01-05 as future work is dropped. Two
clarifications were added while there, both of which this phase learned the hard way:

- Why the first migrated release is `v0.5.1` and not `v0.5.0` — `v0.5.0` published zero assets,
  is kept per D-07, and has nothing to verify.
- That the attestation's **subjects are the binaries, not the checksums file**, so verifying
  `codegraph_<tag>_checksums.txt` returns HTTP 404 by design. This was a live trap during Task 3's
  verification and is the same drift issue #14 records.

**Gap 2 — `docs/RELEASE-PROCEDURES.md` dated baseline entry.** Added as new **§7.1 "Preserved
baselines — do not delete or replace (dated)"**, placed inside §7 (Rollback/cleanup) because §7's
existing prose already warns abstractly about destroying "any un-notarized or otherwise
deliberately-preserved asset a later phase depends on as its baseline" — the concrete instance now
sits directly under the general rule rather than in a separate list. It records both `v0.5.1` (the
SIGN-03 RED baseline; its darwin assets must not be deleted, replaced, or notarized in place) and
`v0.5.0` (permanent zero-asset prerelease), and states explicitly that `v0.5.0` is NOT the
baseline despite being the first v0.5.x tag.

**Re-checked after the edits:**

```
rg -n 'first-migrated-release-tag' docs/     -> no matches (placeholder gone)
rg -c 'v0\.5\.1'  docs/RELEASE-PROCEDURES.md -> 2
rg -c 'SIGN-03'   docs/RELEASE-PROCEDURES.md -> 2
go test -count=1 ./internal/upgrade/          -> ok
task lint:actions                             -> clean
```

Status updated `gaps_found` → `passed`; both gap entries updated `failed` → `closed`. The gap
bodies are left intact rather than deleted, so the record still shows what was missed and why —
a verifier that catches a real omission is evidence the gate works, and erasing it would hide that.
