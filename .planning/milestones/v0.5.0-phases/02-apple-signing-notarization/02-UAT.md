---
status: complete
phase: 02-apple-signing-notarization
source: [02-01-SUMMARY.md, 02-02-SUMMARY.md, 02-03-SUMMARY.md, 02-04-SUMMARY.md, 02-05-SUMMARY.md, 02-06-SUMMARY.md, 02-07-SUMMARY.md]
started: 2026-08-10T00:08:29Z
updated: 2026-08-10T00:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Gatekeeper Verification Target Runs On Your Mac
expected: `task verify:gatekeeper` names all 12 preconditions when one is missing, uses the D-19 oracle (`spctl -a -vv -t install`, verdict from exit status alone — never `-t exec`, never substring search), writes a SYNTHETIC quarantine xattr with hard-failing readback confirmation, records syspolicy_check non-gating, and emits GATEKEEPER-EVIDENCE schema=1 on both pass and fail paths. Ran live on your Mac against v0.5.1 both darwin arches (RED), a deliberate GATEKEEPER_EXPECT mismatch (gate-can-fail proof), and an invalid GATEKEEPER_EXPECT value (input-validation proof).
result: pass
coverage_id: 02-01/D1
requirement: SIGN-03
reason: human_judgment

### 2. SIGN-03 RED Baseline Reads Honestly
expected: 02-EVIDENCE.md's SIGN-03 section records dated RED observations for both darwin arches (spctl exit 3, digest_match=true), the pre-xattr NON-EVIDENCE control, the D-19 positive controls (docker + codex, exit 0) and the negative control, assumption A1 CLOSED, assumption A2 CONFIRMED with the browser-download comparison, the four-item insufficient-check enumeration, and a docs/RELEASE-PROCEDURES.md §7.1 pointer. Nothing in it claims a measurement that did not happen.
result: pass
coverage_id: 02-01/D2
requirement: SIGN-03
reason: human_judgment

### 3. Notarize Rehearsal Target Behaves As Designed
expected: `task release:rehearse-notarize` with no Apple credentials halts BY NAME on the first missing credential in ~0.036s (exit 201, naming MACOS_SIGN_P12). With real credentials on your Mac it notarized both darwin arches for real against `Developer ID Application: Sean Brandt (8D762W58T4)`, both accepted by `spctl -a -vv -t install`. Under the shipped config `cosign verify-blob` verifies the FINAL binary; under the SIGN04_MUTATE=1 D-07 mutation that relationship INVERTS. A deliberately wrong MACOS_SIGN_PASSWORD leaked zero certificate/key material. `git diff --stat -- .goreleaser.yaml` was empty after every run.
result: pass
verified_by: agent (2026-08-10) — credential-free halt reproduced live (exit 201, 0.059s, names MACOS_SIGN_P12; SIGN04_MUTATE=1 gated identically); TestRehearseNotarizeDeclaresCredentialPreconditions + non-vacuity companion PASS; `git diff --stat -- .goreleaser.yaml` empty. Real-Apple half cross-checked against 02-EVIDENCE.md: NOTARIZE-EVIDENCE lines 382/384 (both arches apple_status=notarized, spctl_exit=0, sha_changed=true), cosign subject table lines 452-453 (presign=false, final=true), inversion recorded line 456, A5 leak verdict RESOLVED.
coverage_id: 02-04 (legacy — no coverage block)
requirement: SIGN-01, SIGN-04
reason: no_coverage_block

### 4. docs/RELEASE.md §1d Gatekeeper Section Does Not Overstate
expected: The §1d section carries the 3-row applicability table with a pending marker on the first-notarized row, the exact guarantee phrase scoped by reference to that table, the operational meaning of each guarantee part, the offline-first-launch limitation named with DIST-06, xattr-write → xattr-readback → spctl reproduction commands (never `.zip`), pass/fail examples showing the `source=` line plus exit status, the exit-status-only verdict rationale, and a six-item "this is NOT verification" list in a subordinated details block. Judgment call: no sentence claims a measurement that has not happened, checked against 02-EVIDENCE.md.
result: pass
verified_by: agent (2026-08-10) — 6 of 8 recorded checks reproduce exactly. Two do not, both traced to plan 02-07 later editing the same file: `rg -c 'spctl -a -vv -t install'` is now 2 (not 1) because 02-07 added an accurate Status-note occurrence at line 16, and that occurrence makes the xattr-readback line no longer the first occurrence of either string. The `pending marker on the first-notarized row` in this test's text is likewise superseded — 02-07 closed it deliberately (`rg -c 'pending' == 0`), row 2 now reads v0.7.0 confirmed GREEN. Substance holds: no sentence claims a measurement that has not happened. Maintainer accepted the supersession.
coverage_id: 02-05/D1
requirement: SIGN-02
reason: validation_failed (coverage block declares kind `automated`, which is not a valid kind)

### 5. docs/RELEASE.md Rest-Of-Document Sweep Is Accurate
expected: The stale pre-release status note is gone; the reproducibility section scopes the darwin-signature claim as pending rather than asserted; the codegraph-upgrade-as-consumer section distinguishes the detached Sigstore signature from the embedded Apple signature and states plainly that the former does nothing for Gatekeeper; §1's asset list still matches the real published releases (8 assets × 4 platforms on v0.5.1 and v0.6.0 — notarization mutates the raw binary in place, per D-04, so no new asset type appears).
result: pass
verified_by: agent (2026-08-10) — stale `no v* tag has been pushed` claim confirmed absent (0); Sigstore-vs-Apple distinction present (line 466, "detached"); darwin signature claim at line 410 is scoped STRONGER than recorded (02-07 replaced "pending" with a measured negative — final signed binary not bit-for-bit reproducible by anyone, Apple embeds a per-operation trusted timestamp, with the compare-pre-sign-never-final consequence stated). `rg -c 'reproduc'` is 11, recorded as 9 — same 02-07 supersession as Test 4. D-04 asset-shape cross-check STRENGTHENED: 02-05 could only compare v0.5.1/v0.6.0 (both pre-notarization, so it never actually tested D-04); comparing v0.6.0 (un-notarized) against v0.7.0 (notarized) gives 17 assets each with identical 4-per-platform shape — notarization mutates the raw binary in place, now measured rather than assumed. Maintainer accepted.
coverage_id: 02-05/D2
requirement: SIGN-02
reason: validation_failed (coverage block declares kind `automated`, which is not a valid kind)

### 6. CI Job Wiring Is Structurally Right
expected: The new gatekeeper job and notarized-suite job in post-release-verify.yml are structurally correct — checkout pinning, cosign verify-before-execute ordering, executed-test counting from `go test -json`, and complete precondition sets. This plan wired and shape-tested them but deliberately could not dispatch them; whether they actually PASS against a real published release is Test 7's evidence.
result: pass
verified_by: agent (2026-08-10) — all four structural claims reproduce with no drift. TestVerifyNotarizedSuiteDeclaresNamedPreconditions (+ non-vacuity companion) and TestPostReleaseJobsDeclareCheckoutPolicy PASS; `task lint:actions` exit 0. cosign verify-before-execute ordering confirmed in Taskfile.yml (verify 2452 -> chmod 2464 -> execute 2483), NOT in post-release-verify.yml — searched the Taskfile surface deliberately per yctys69cke (CI jobs are thin wrappers; workflow line 392 is only a comment pointing elsewhere). Executed-test counting at Taskfile 2492 counts `.Test != null` pass/fail events from `go test -json`, and 2495-2496 hard-fails on a zero count regardless of go_test_exit — a POSITIVE work-happened assertion satisfying rule 84d1gfpywd.
coverage_id: 02-06/D4
requirement: SIGN-02
reason: human_judgment

### 7. v0.7.0 Ships Notarized And Launches Without A Gatekeeper Dialog
expected: v0.7.0 was cut and actually notarized (not silently skipped). Gatekeeper GREEN on both darwin arches with quarantine confirmed. The notarized-suite job executed 142 tests against the real published arm64 binary. A manual-dispatch run proved no job silently skips (run 31338409898, all 7 jobs SUCCESS, none skipped; automatic run 31338004416 likewise). And the unproxied part: YOU downloaded the binary in Safari and launched it, and no Gatekeeper dialog appeared.
result: pass
verified_by: agent (2026-08-10) for the checkable half; maintainer for the unproxied half. Run 31338004416 (event=workflow_run) and 31338409898 (event=workflow_dispatch) both 7/7 SUCCESS, none skipped — the two-trigger pairing is what demonstrates the event-aware conclusion guard does not silently skip. GATEKEEPER-EVIDENCE both arches: exit=0, observed=accepted, expect=accepted, xattr_present=true (quarantine really applied), digest_match=true with sha256 == gh_digest (the assessed bytes ARE the published ones), source_assertion=pass. NOTARIZED-SUITE-EVIDENCE: executed_tests=142, go_test_exit=0. syspolicy_exit=70 recorded non-gating per D-19/988bg897x3 (stapling out of scope, DIST-06). The Safari download-and-launch with no Gatekeeper dialog is the maintainer's own unproxied observation — no instrument in this phase stands in for it.
coverage_id: 02-07/D2
requirement: SIGN-02
reason: human_judgment

### 8. signs: block matches the upgrade contract (D-18 rename)
expected: signs: block renamed from binary_signs: with ids: [raw], artifacts: binary, byte-identical signature/cmd/args; property asserted for all 4 platforms, not a literal string
result: pass
source: automated
coverage_id: 02-02/D1

### 9. notarize: block gates on the five-term credential conjunction
expected: notarize.macos[0].ids names exactly the two darwin build ids (exact-set, both directions mutation-proved); enabled: is a five-term credential conjunction; sign: declares no entitlements key; exactly one macos: entry enforced by the parser
result: pass
source: automated
coverage_id: 02-02/D2

### 10. Every goreleaser-invoking Taskfile target covered
expected: check:goreleaser, check:darwin-release-build, release:dry-run, release:dry-run-signed, release:goreleaser all conform to the asserted shape
result: pass
source: automated
coverage_id: 02-02/D3

### 11. deferred-items.md created for phase 02
expected: Records the deferred darwin-toolchain-canary re-wiring with its revisit condition
result: pass
source: automated
coverage_id: 02-02/D4

### 12. CODEGRAPH_TEST_BIN override in test/integration
expected: Unset variable behaves exactly as before; a valid override skips the build and spawns against the named binary; an invalid override aborts by name before temp dir, build, or any test run
result: pass
source: automated
coverage_id: 02-03/D1

### 13. CODEGRAPH_TEST_BIN override in test/wireoracle
expected: Same resolver/TestMain/table test duplicated deliberately; suite proven IN SCOPE for criterion 4 by running against a release-shaped binary
result: pass
source: automated
coverage_id: 02-03/D2

### 14. Apple credentials readable by exactly one job
expected: The five Apple credentials are readable only by release.yml's release job, never by a pull_request-triggerable workflow, always at step-level env: — asserted by a runtime-enumerating test with both non-vacuity proofs
result: pass
source: automated
coverage_id: 02-06/D1

### 15. OIDC signing surface not widened
expected: release.yml's file name, tag trigger, and single id-token: write job unchanged by the credentials-scoping change
result: pass
source: automated
coverage_id: 02-06/D2

### 16. Event-aware conclusion guard on every post-release job
expected: All 5 jobs in post-release-verify.yml carry the verbatim event-aware conclusion guard, and a dry evaluation confirms it runs (not skips) under workflow_dispatch
result: pass
source: automated
coverage_id: 02-06/D3

### 17. final_local_sha256 observable and wired
expected: release:record-final-hashes emits final_local_sha256, honestly labelled as a post-everything measurement, wired into release.yml between the release and attestation steps, hard-failing on absent metadata
result: pass
source: automated
coverage_id: 02-07/D1

### 18. Criteria 2/3/4 recorded against the published release
expected: 02-EVIDENCE.md carries the SIGN-02 GREEN section + RED-vs-GREEN table, the SIGN-04 five-point byte identity with narrow A3 verdict, and the criterion-4 notarized-suite section with its arm64-only scope limit; docs/RELEASE.md's two pending markers closed with citations plus chmod guidance
result: pass
source: automated
coverage_id: 02-07/D3

## Summary

total: 18
passed: 18
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none]

## Notes

Two coverage-block defects were observed during this session. Neither is a defect in a
deliverable, and neither produced a gap:

1. **02-05-SUMMARY.md declares `kind: automated`**, which is not a valid verification
   kind (valid: unit, integration, e2e, automated_ui, manual_procedural, other). All 6
   of its verification refs errored on this, so both D1 and D2 fell through to human
   checkpoints as `validation_failed`. The refs themselves were runnable and were run.

2. **Stale count assertions superseded by a later plan in the same phase.** 02-05's
   `rg -c 'spctl -a -vv -t install' == 1` and `rg -c 'reproduc' == 9` no longer
   reproduce (now 2 and 11) because plan 02-07 legitimately edited docs/RELEASE.md
   afterwards. Its "xattr readback is the first occurrence of either string in the file"
   claim is false for the same reason. Per the swr2e3akn4 convention, completed plan
   artifacts are a record of observation at their own time and were NOT retro-edited.

One check was strengthened beyond what its plan could achieve: 02-05's D-04 asset-shape
cross-check compared v0.5.1 against v0.6.0, both pre-notarization, so it never actually
tested D-04. This session compared v0.6.0 (un-notarized) against v0.7.0 (notarized) —
17 assets each, identical 4-per-platform shape — making D-04 measured, not assumed.
