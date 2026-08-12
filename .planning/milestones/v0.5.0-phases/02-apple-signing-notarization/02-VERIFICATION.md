---
phase: 02-apple-signing-notarization
verified: 2026-08-09T23:30:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 2: Apple Signing & Notarization Verification Report

**Phase Goal:** A macOS user who downloads a `codegraph` release asset in a browser can run it without Gatekeeper blocking them — and the project can prove that claim with a check it has already watched fail.

**Verified:** 2026-08-09T23:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Success Criteria 1–5)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Gate shown RED first, with quarantine confirmed, 4 insufficient-check types explicitly recorded | ✓ VERIFIED | `02-EVIDENCE.md` "SIGN-03 — RED baseline" + "Pre-xattr control (NON-EVIDENCE)" section: v0.5.1 darwin/arm64 and darwin/amd64 both `spctl -a -vv -t install` → exit 3, `xattr_present=true` confirmed BEFORE assessment (readback matches written value). The four insufficient checks (green CI, `codesign -dvv`, `notarytool history` Accepted, spctl-on-unquarantined-file) are each explicitly itemized under "D-19 — the oracle" §1–4, with the pre-xattr control recorded as its own NON-EVIDENCE-labelled subsection so it can't be conflated with the real verdict. `Taskfile.yml:2116` `verify:gatekeeper` implements this — read and confirmed at source (below). |
| 2 | GREEN on published notarized asset, verdict from exit status not substring, `source=Notarized Developer ID`, `-t install` not `-t exec` | ✓ VERIFIED | `02-EVIDENCE.md` "SIGN-02 — GREEN Gatekeeper verdict on the published release": both `v0.7.0` darwin arches report `GATEKEEPER-EVIDENCE ... expect=accepted observed=accepted exit=0 xattr_present=true source_assertion=pass`. Independent unproxied corroboration on the maintainer's own Mac via real Safari download: `spctl -a -vv -t install` → `accepted`, `source=Notarized Developer ID`, `origin=Developer ID Application: Sean Brandt (8D762W58T4)`. Verdict derivation confirmed at source: `Taskfile.yml:2299-2308` derives `OBSERVED` from `SPCTL_EXIT` alone via a `case` statement (0→accepted, 3→rejected, else→hard-fail unclassifiable) — the `source=` string check (`grep -q`, line 2315) is a *separate*, independently-labelled `source_assertion` field, never substituted for the exit-status verdict. `-t install` (never `-t exec`) confirmed in the command itself (line 2283) and in the task's own `desc:` explaining why (lines 2124-2126). |
| 3 | sha256 identical across notarize→re-download→GitHub-digest→cosign→SLSA, mis-order makes it diverge | ✓ VERIFIED (with an honestly-recorded, pre-flagged scope note) | `02-EVIDENCE.md` "SIGN-04 — five-point byte identity on the published release": for both darwin arches, `final_local_sha256` = re-downloaded asset sha256 = GitHub's own recorded digest, all three identical 64-hex-char values. `cosign verify-blob` succeeds against the final subject (point 4) — captured verbatim for the linux/amd64 subject, and separately corroborated for darwin/arm64 via the `self-upgrade proof (darwin/arm64)` job's own in-process cosign check (green). `gh attestation verify` succeeds (point 5). The deliberate mis-ordering mutation (`02-EVIDENCE.md` "SIGN-04 — ordering, measured (D-07)") shows the relationship literally inverts: shipped config `cosign_verifies_final=true`/`verifies_presign=false`; mutated config `verifies_presign=true`/`verifies_final=false`. **Honest scope limit** (not a gap — explicitly called out in both `02-EVIDENCE.md` and this phase's task instructions): `verify:release-assets`'s cosign/attestation bindings (points 4/5) are matrixed against `linux/amd64` only, not darwin; darwin/amd64's points 4/5 rest on the same-workflow same-signing-identity binding proven against linux/amd64's bytes, not its own. This is recorded plainly in the evidence rather than papered over, consistent with the phase's stated discipline. |
| 4 | Full CLI+MCP suite runs green against the notarized binary itself | ✓ VERIFIED | `02-EVIDENCE.md` "Criterion 4 — the suite against the notarized binary": `notarized suite proof (darwin/arm64)` job, `executed_tests=142`, `go_test_exit=0`, subject binary sha256 identical to the byte-identity chain's darwin/arm64 value (same bytes, not a separate build). `test/wireoracle` (MCP wire-protocol suite) confirmed in scope via 02-03's empirical finding that `normalize.go`'s `serverVersion` rule erases the differing version field, so all 27 frozen scenarios pass unmodified against a release-shaped binary. **Scope limit, explicitly recorded, not hidden:** darwin/arm64 only — no darwin/amd64 executed-suite leg exists in this phase. Wiring confirmed at source: `post-release-verify.yml`'s `notarized-suite` job checks out `ref: needs.resolve-tag.outputs.tag` (binds tests to the release's own commit, not the workflow's default ref) and declares `needs: [resolve-tag, verify-supply-chain]` (cosign-verifies before executing, closing the "tampered asset could run before its own verification job fails" ordering risk named in the plan). `Taskfile.yml:2440-2443` hard-fails if the cosign bundle download itself is missing, before any execution. |
| 5 | `docs/RELEASE.md` states the exact guarantee, names offline-launch limitation, gives reproduction commands | ✓ VERIFIED | `docs/RELEASE.md:181` states the guarantee in the exact three-part form: **notarized, online-verified, not stapled**, scoped via a per-release applicability table (line 175-177) that correctly marks `v0.7.0` onward as covered and pre-v0.7.0 releases as `§d does not apply`. Offline limitation named explicitly at line 193-196 ("Known limitation: offline first launch fails..."). Reproduction commands given verbatim, in the correct xattr-then-spctl order (lines 248, 256, 260), with `spctl -a -vv -t install` and exit-status framing at lines 258-260, 284 ("Read the exit status, never a text search"). The four insufficient-check enumeration is mirrored in the doc at lines 296-313, including `-t exec`'s bare-Mach-O rejection at line 313. No `pending` markers remain anywhere in the file (grep returned zero matches) — all were resolved against the real `v0.7.0` release per 02-07. A documentation defect this phase's own step 8 (real Safari download) uncovered — missing chmod/executable-bit guidance for a browser-downloaded raw Mach-O — was fixed (`docs/RELEASE.md:215`, `chmod +x ...`). |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.goreleaser.yaml` `notarize:` block | quill-backed sign+notarize, ids scoped to 2 darwin build ids, `enabled:` 5-way conjunction, no entitlements key | ✓ VERIFIED | Read at source, lines 250-265. Matches D-01/D-03/D-04 exactly. |
| `.goreleaser.yaml` `signs:` block | release-scoped (not `binary_signs:`), `ids: [raw]`, name-template-derived `signature:` | ✓ VERIFIED | Lines 325-333. D-18 (moved to release-scoped pipe) confirmed at source, not just in comments. |
| `Taskfile.yml` `verify:gatekeeper` | RED/GREEN dual-purpose target, exit-status-only verdict, named preconditions | ✓ VERIFIED | Lines 2116-2374, read in full. All must-have behaviors (input validation halt before network, xattr-then-spctl ordering, digest cross-check, exit-status-only verdict, non-gating syspolicy_check) present and matching evidence exactly. |
| `Taskfile.yml` `verify:notarized-suite` | cosign-verify-before-execute, downloads real published binary | ✓ VERIFIED | Lines 2376-2450+, read in full. Verify-before-execute ordering confirmed at the code level (comment + hard-fail if bundle missing). |
| `Taskfile.yml` `release:rehearse-notarize` | maintainer-only, hard-fails by name on missing credentials | ✓ VERIFIED | Target exists at line 950; D-09 precondition-halt proof recorded in evidence (0.036s, before network). |
| `Taskfile.yml` `release:record-final-hashes` | invoked from release job, records `final_local_sha256` post-everything | ✓ VERIFIED | Target exists at line 843. |
| `.github/workflows/post-release-verify.yml` gatekeeper + notarized-suite jobs | event-aware guard, tag-pinned checkout, ordering dependency | ✓ VERIFIED | Read at source: `if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'` on every job (D-11 guard); `ref: needs.resolve-tag.outputs.tag` on both jobs; `notarized-suite` declares `needs: [resolve-tag, verify-supply-chain]`. |
| `docs/RELEASE.md` Gatekeeper section | guarantee statement, limitation, reproduction commands | ✓ VERIFIED | See truth 5 above. |
| Shape tests (8 named tests across 3 files) | pin the machine-checkable invariants | ✓ VERIFIED | All 8 named tests (`TestVerifyGatekeeperDeclaresNamedPreconditions`, `TestSignsSidecarMatchesUpgradeContract`, `TestNotarizeMacosIdsCoverDarwinBuildIDs`, `TestNotarizeMacosEnabledIsEnvGated`, `TestNotarizeMacosHasExactlyOneEntry`, `TestNotarizeMacosOmitsEntitlements`, `TestAppleSecretsScopedToSingleReleaseJob`, `TestRehearseNotarizeDeclaresCredentialPreconditions`) run individually by this verifier — all PASS. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SIGN-01 | 02-02, 02-04, 02-06, 02-07 | Darwin binaries Developer ID codesigned and notarized during release, credentials as CI secrets | ✓ SATISFIED | `.goreleaser.yaml` `notarize:` block; real `v0.7.0` release notarize pipe ran (`02-EVIDENCE.md` "SIGN-02"); 5 Apple secrets scoped to single release job (`TestAppleSecretsScopedToSingleReleaseJob` PASS) |
| SIGN-02 | 02-01, 02-03, 02-05, 02-06, 02-07 | User downloading in browser is not blocked by Gatekeeper; `spctl -a -vv -t install` accepted, exit 0, genuine quarantine confirmed | ✓ SATISFIED | Truth 2 above; independent maintainer machine + Safari download confirmation |
| SIGN-03 | 02-01, 02-05, 02-07 | Gate demonstrated RED before trusted GREEN; insufficient checks named | ✓ SATISFIED | Truth 1 above |
| SIGN-04 | 02-02, 02-04, 02-07 | Byte-identity across pipeline stages, ordering measured not assumed | ✓ SATISFIED (scope-limited, honestly recorded — see Truth 3) | Truth 3 above |

REQUIREMENTS.md itself already marks all four `[x]` complete and the traceability table `Complete` (lines 24-27, 76-79) — cross-checked against actual evidence and source above rather than trusted at face value; the codebase evidence supports that marking.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`), placeholder returns, or hardcoded-empty stub patterns were found in the phase's modified files (`Taskfile.yml`, `.goreleaser.yaml`, `.github/workflows/{release,post-release-verify}.yml`, `docs/RELEASE.md`, shape test files). One category of finding did surface, carried over verbatim from `02-REVIEW.md` (produced by this phase's own code-review step, 0 critical / 3 warning):

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.github/workflows/release.yml` | 38, 133 | Stale prose still names `binary_signs:` as the current pipe, though `.goreleaser.yaml` itself moved to `signs:` per D-18 | ⚠️ Warning | Documentation drift only — no shape test guards prose comments; a future maintainer "fixing" `release.yml`'s comment to match the equally-stale `RELEASE-PROCEDURES.md` line would compound the confusion. Does not affect any of the 5 success criteria (verified: `docs/RELEASE.md`, the user-facing doc criterion 5 requires, is NOT affected — it never named the internal pipe). |
| `docs/RELEASE-PROCEDURES.md` | 130-133 | Same stale `binary_signs:` reference | ⚠️ Warning | Same as above — internal maintainer doc, not `docs/RELEASE.md` |
| `.github/workflows/post-release-verify.yml` | 158-165 | `gh api --paginate --jq` in `resolve-tag`'s rarely-exercised fallback branch concatenates per-page JSON documents rather than merging before filtering — a real latent correctness bug that would misfire only once the repo's tag count crosses one API page | ⚠️ Warning | Currently unexercised (repo's tag count is well under the pagination threshold, and the primary `head_branch` path — the one that fired for the actual v0.7.0 verification runs recorded in evidence — never reaches this fallback). Does not affect any of the 5 success criteria as measured against v0.7.0. Left open by the phase's own review; not remediated as part of this phase's SUMMARYs. |
| Coverage (not a code pattern) | — | `verify:release-assets`'s cosign/attestation bindings are `linux/amd64`-scoped only, not matrixed over all 4 platforms | ℹ️ Info (already reflected in Truth 3 as an honestly-recorded scope limit, per this phase's own explicit self-disclosure in `02-EVIDENCE.md`) | Not a gap — the phase task instructions explicitly pre-authorize treating this as a recorded scope limit, and `02-EVIDENCE.md`'s own "Points 4 and 5" section states it plainly rather than implying broader coverage. |

None of these four findings are BLOCKERs: they were surfaced by the phase's own code-review step (0 critical), none contradicts a success criterion, and two of them (the coverage scope limit, the doc-comment drift) are pre-disclosed rather than hidden. The pagination bug (WR-02) is a genuine latent defect but is untriggered by anything this phase's success criteria measure.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Shape tests exist and pass individually (not full-suite) | `go test ./internal/upgrade/... -run 'TestVerifyGatekeeperDeclaresNamedPreconditions\|TestSignsSidecarMatchesUpgradeContract\|TestNotarizeMacosIdsCoverDarwinBuildIDs\|TestNotarizeMacosEnabledIsEnvGated\|TestNotarizeMacosHasExactlyOneEntry\|TestNotarizeMacosOmitsEntitlements\|TestAppleSecretsScopedToSingleReleaseJob\|TestRehearseNotarizeDeclaresCredentialPreconditions' -v` | All 12 subtests (8 named + 4 `_...IsError` companions) PASS | ✓ PASS |
| `notarize:`/`signs:` blocks present in shipped config, matching plan intent | `rg -n "^notarize:\|^signs:" .goreleaser.yaml` + full read | Found, content matches D-01/D-04/D-18/D-15 verbatim | ✓ PASS |
| No stray `pending` markers left in `docs/RELEASE.md` | `rg -n "pending\|PENDING" docs/RELEASE.md` | Zero matches | ✓ PASS |

Note: the full CLI/MCP suite against the real notarized binary (criterion 4) is CI/Apple-hardware-dependent and cannot be re-run by this verifier locally against the published asset; that criterion's evidence rests on the recorded CI job conclusion (`SUCCESS`, `executed_tests=142`, `go_test_exit=0`) cross-checked against the workflow wiring (tag-pinned checkout, verify-before-execute ordering) read directly from `post-release-verify.yml` and `Taskfile.yml` source — not merely the SUMMARY narrative. Per the task's own note, the regression gate (`task test` full suite, including `-race`) already ran and passed; it was not re-run here.

### Human Verification Required

None. All 5 success criteria are backed by recorded, source-cross-checked evidence (CI job conclusions, verbatim command transcripts, and independently-reproduced maintainer-machine observations), not solely by SUMMARY narrative.

### Gaps Summary

No gaps. All 5 ROADMAP success criteria and all 4 requirement IDs (SIGN-01..SIGN-04) are verified against actual source (`.goreleaser.yaml`, `Taskfile.yml`, `.github/workflows/{release,post-release-verify}.yml`, `docs/RELEASE.md`) and against `02-EVIDENCE.md`'s recorded real measurements against the published `v0.7.0` release — not merely against SUMMARY claims. The phase's recurring failure family ("a check whose failure is indistinguishable from its success") was specifically checked for and not found: `verify:gatekeeper`'s verdict is read from `spctl`'s exit status via an explicit `case` statement (never substring search), the RED baseline is recorded with an explicit NON-EVIDENCE control (spctl on a never-quarantined file), the D-19 pivot from `-t exec` to `-t install` is justified with a measured before/after comparison rather than asserted, and the notarized-suite job's executed-test count is asserted non-zero (142) rather than trusted from a bare green exit code. Three warning-level findings from the phase's own code review (stale doc-comment prose in two maintainer-only files, a latent pagination bug in a rarely-exercised fallback branch, and an already-self-disclosed darwin cosign/attestation coverage scope limit) are recorded above for completeness; none blocks the phase goal or contradicts a success criterion.

---

_Verified: 2026-08-09T23:30:00Z_
_Verifier: Claude (gsd-verifier)_
