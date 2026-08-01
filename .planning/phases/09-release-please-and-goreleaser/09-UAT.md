---
status: complete
phase: 09-release-please-and-goreleaser
scope: "full phase — plans 09-01..09-08"
source: [09-01-SUMMARY.md, 09-02-SUMMARY.md, 09-03-SUMMARY.md, 09-04-SUMMARY.md, 09-05-SUMMARY.md, 09-06-SUMMARY.md, 09-07-SUMMARY.md, 09-08-SUMMARY.md]
started: 2026-07-30T00:00:00Z
updated: 2026-08-01T00:00:00Z
---

> **Extended to full-phase scope 2026-08-01.** Tests 1–33 were recorded in the earlier
> session, which was deliberately scoped to plans 09-01..09-06 because 09-07/09-08 had
> not yet executed. Both have since executed (09-07 SKIPPED by maintainer decision,
> 09-08 COMPLETE — v0.2.0 published), so tests 34–42 extend this file to cover them.
> Tests 1–33 are unchanged and were not re-run.
>
> 09-07 and 09-08 carry no structured `coverage:` block (`uat classify-coverage` →
> `mode: legacy`), so tests 34–42 are prose-derived human checkpoints — none were
> auto-passed. Live evidence was re-measured in-session for each and is recorded inline.

## Current Test

[testing complete]

## Tests

### 1. 09-05 D4 — private key material fully removed
expected: No private key material left in the working tree; no secret value appears anywhere in this plan's artifacts
why_human: human_judgment — Executor visibility is scoped to this repository's working tree; whether the maintainer also deleted the downloaded .pem from their local Downloads folder (outside the repo) is attested by the maintainer's report, not independently observable by the executor. Flagging for human confirmation rather than asserting full deletion as machine-verified.
result: pass
evidence_split: |
  MACHINE-VERIFIED (in-repo): `git status --porcelain | rg '\.(pem|key)$'` -> no matches;
  repo-wide `fd -H -e pem -e key` -> no matches; no secret VALUE appears in any phase
  artifact (only the non-sensitive App ID 3982691, Client ID, and key FILENAMES).
  MAINTAINER-ATTESTED (outside repo): deletion of the downloaded .pem from local disk.
  Recorded as attestation, not as machine verification — this is exactly the split the
  checkpoint's human_judgment rationale describes.
note: |
  Flagged at checkpoint time: two PEMs were observed at ~/Downloads earlier in this
  session while diagnosing the App-token 401 (verify-app-key.sh was run against both).
  The maintainer confirmed pass after that was surfaced. The live one
  (fzy-release-please, App 3982691) grants contents/issues/pull_requests write on every
  repository the shared App is installed on, so its local copy is worth removing if it
  has not been already.

### 2. 09-01 D1
expected: TestReleaseWorkflowFileMatchesPattern reads release.yml off disk, reconstructs the SAN a tag push would produce, and proves releaseWorkflowRefPattern accepts it and rejects a renamed-workflow SAN and a branch-ref SAN
result: pass
source: automated
coverage_id: D1

### 3. 09-01 D2
expected: TestReleaseWorkflowTriggerIsTagPushOnly mechanically pins release.yml's on: block to exactly one push/tags trigger (v[0-9]*), enforcing the header comment's claim
result: pass
source: automated
coverage_id: D2

### 4. 09-01 D3
expected: TestWorkflowSourceHelpersFailLoudly proves every parse core returns a non-nil error (never a zero value) on 6 synthetic missing-target cases across all 4 helper pairs
result: pass
source: automated
coverage_id: D3

### 5. 09-01 D4
expected: release-please-config.json / .release-please-manifest.json resolve the repo's real 0.1.0 baseline with no bootstrap PR, no release-as, no extra-files
result: pass
source: automated
coverage_id: D4

### 6. 09-01 D5
expected: 6-target pretag-gate job blocks the release-please job via needs:, and its set -euo pipefail failure path is proven to abort on a rejecting GOOS/GOARCH pair
result: pass
source: automated
coverage_id: D5

### 7. 09-02 D1
expected: Release exists -> gh release upload --clobber fires, gh release create never fires, and the upload argv carries neither --generate-notes nor --prerelease
result: pass
source: automated
coverage_id: D1

### 8. 09-02 D2
expected: Release absent + dash-suffixed (prerelease-shaped) tag -> gh release create fires with --prerelease and --generate-notes, upload never fires
result: pass
source: automated
coverage_id: D2

### 9. 09-02 D3
expected: Release absent + stable tag -> gh release create fires without --prerelease
result: pass
source: automated
coverage_id: D3

### 10. 09-02 D4
expected: Zero release assets present -> step exits non-zero with a diagnostic annotation, invoking neither create nor upload
result: pass
source: automated
coverage_id: D4

### 11. 09-02 D5
expected: Release absent + create call fails -> step exits non-zero rather than silently falling back to upload
result: pass
source: automated
coverage_id: D5

### 12. 09-02 D6
expected: The diff is confined to the Publish GitHub release step's body only — release.yml's LOCKED name/trigger/cosign-step identity, internal/upgrade/verify.go, and .goreleaser.yaml are byte-unchanged
result: pass
source: automated
coverage_id: D6

### 13. 09-03 D1
expected: TestPRTitleLintAcceptsAndRejects executes pr-title.yml's own shipped lint step against 4 reject rows (bare title, unaccepted type word, missing colon-space separator, empty subject) — each asserted non-zero exit plus a ::error:: annotation naming the offending title
result: pass
source: automated
coverage_id: D1

### 14. 09-03 D2
expected: 13 accept rows pass exit 0: one per accepted type word (feat/fix/perf/refactor/docs/chore/ci/test/build/revert), a scoped title, a breaking-change-marker title, and an adversarial shell-metacharacter title proven to produce no side effect
result: pass
source: automated
coverage_id: D2

### 15. 09-03 D3
expected: pr-title.yml's pull_request trigger explicitly lists edited alongside opened/synchronize/reopened, closing RESEARCH Pitfall 1 without widening ci.yml's shared trigger
result: pass
source: automated
coverage_id: D3

### 16. 09-03 D4
expected: actionlint job added to ci.yml statically checks all 5 workflow files on every PR/push; diff confined to added lines only
result: pass
source: automated
coverage_id: D4

### 17. 09-04 D1
expected: docs/RELEASE-PROCEDURES.md no longer instructs a maintainer to create/push a version tag by hand as the normal path; §4 documents the release-PR-merge flow with the hand-pushed tag surviving only as the explicitly-labelled rc escape hatch (D-10)
result: pass
source: automated
coverage_id: D1

### 18. 09-04 D2
expected: §3 records the fast-forward merge model with repo evidence (zero merge commits, main is an ancestor today) and explicitly supersedes Phase-8 08-CONTEXT.md's squash-merge wording rather than silently contradicting it (D-09)
result: pass
source: automated
coverage_id: D2

### 19. 09-04 D3
expected: §7 rollback covers all three artifacts a release-please cut leaves behind (tag, GitHub Release, manifest/changelog bump commit + its PR), including the autorelease label consideration
result: pass
source: automated
coverage_id: D3

### 20. 09-04 D4
expected: §1's pre-tag sweep is documented as now-automated in release-please.yml's pretag-gate job, demoted from enforcement point to documentation, without silently dropping the D-09 lesson
result: pass
source: automated
coverage_id: D4

### 21. 09-04 D5
expected: §9 documents the GitHub App prerequisite end to end: creation, three installation permissions, that installation scope differs from a workflow's permissions: block, the two secret names, the deprecated-input note, and the branch-protection bypass-actor requirement
result: pass
source: automated
coverage_id: D5

### 22. 09-04 D6
expected: §10 records every divergence this phase accepted (roadmap criterion 3 / D-05, D-08's file location, D-03's Path B with its guard-plus-test requirement), each naming its source decision and reason
result: pass
source: automated
coverage_id: D6

### 23. 09-04 D7
expected: ROADMAP.md's Phase 9 success criterion 3 carries a recorded amendment in the same style as Phase 8's goal amendment, amended in place with rationale rather than deleted or left standing as an unmet claim
result: pass
source: automated
coverage_id: D7

### 24. 09-04 D8
expected: §4 documents the zero-release-worthy-commits silent no-op so a maintainer does not mistake it for a broken pipeline
result: pass
source: automated
coverage_id: D8

### 25. 09-04 D9
expected: docs/RELEASE.md's user-facing verification commands are confirmed still correct against the unchanged cosign identity, with the confirmation recorded rather than assumed
result: pass
source: automated
coverage_id: D9

### 26. 09-04 D10
expected: The folded todo is marked resolved with a pointer to the sections that closed it
result: pass
source: automated
coverage_id: D10

### 27. 09-04 D11
expected: Zero code changes: git diff shows no changes to workflows, internal/upgrade/*.go, or .goreleaser.yaml; go test ./internal/upgrade/... green
result: pass
source: automated
coverage_id: D11

### 28. 09-05 D1
expected: Both required repository secrets (APP_ID, APP_PRIVATE_KEY) exist, confirmed by name and timestamp only
result: pass
source: automated
coverage_id: D1

### 29. 09-05 D2
expected: The GitHub App's declared installation permissions are exactly Contents/Pull requests/Issues at read-and-write, with metadata:read present only as GitHub's mandatory implicit grant (not an over-scope)
result: pass
source: automated
coverage_id: D2

### 30. 09-05 D3
expected: Repository Actions configuration confirmed: Actions enabled; workflow default permissions read; PR create-and-approve enabled (re-verified live, superseding an earlier disabled reading)
result: pass
source: automated
coverage_id: D3

### 31. 09-05 D5
expected: Accepted deviation recorded: reused pre-existing shared App vs. plan's suggested purpose-built App, including the T-09-05-02 blast-radius consequence and the unverified repository_selection scope
result: pass
source: automated
coverage_id: D5

### 32. 09-06 D1
expected: The v1.0 integration branch (gsd/v1.0-drop-in-parity-human-ux) lands on main by fast-forward, preserving all 502 commits and the zero-merge-commit property
result: pass
source: automated
coverage_id: D1

### 33. 09-06 D2
expected: release-please opens a real release PR on main using the GitHub App token, proving the App-token path works end to end
result: pass
source: automated
coverage_id: D2

### 34. 09-08 D1 — v0.2.0 cut by release-please, no human `git tag`
expected: REL-02 first half — tag and Release authored by fzy-release-please[bot], version computed not forced, releases/latest -> v0.2.0
evidence: "author=fzy-release-please[bot]; prerelease=false; releases/latest=v0.2.0; `git log v0.1.0..cce95f3 | rg '^Release-As:'` -> 0 matches"
result: pass

### 35. 09-08 D2 — release.yml fired on the App-authored tag push and went fully green
expected: The App-authored tag push started release.yml (the one link 09-07 was skipped without proving) and every job succeeded
evidence: "run 30675077940, event=push, conclusion=success, 11 jobs, 0 non-success"
result: pass

### 36. 09-08 D3 — asset contract: 20 assets across six platforms
expected: 6 binaries + 6 .sigstore.json + 6 .spdx.json SBOMs + checksums + provenance bundle
evidence: "set equality against a derived 6-platform expectation — actual 20 names == expected 20 names, exact (no extras, no omissions). No zero-byte assets; sizes plausible. Verified by set comparison, not visual inspection of the name list."
note: "The checksums-file-vs-SLSA-subjects comparison run during this test is evidence for test 40, not for this one; the load-bearing evidence here is name-set equality plus the size check."
observation: ".spdx.json SBOMs are byte-size-identical within each OS pair (darwin 254691, linux 252452, windows 263704) — expected for a generic Syft scan, since the Go module graph does not vary by arch. Not a Phase 9 gap; no arch-precise SBOM was claimed."
result: pass

### 37. 09-08 D4 — publish took the UPLOAD branch (D-04)
expected: The Release object predates the release.yml run, so `gh release create` could not have run and `upload --clobber` necessarily did
evidence: "three independent lines, re-measured in-session. (1) TIMING — Release created 2026-08-01T00:14:36Z; run 30675077940 started 00:15:21Z, so the Release predates the run by 45s. (2) STRUCTURE — release.yml:306 selects the branch on `gh release view` exit status (never an ||-chained fallback), so an existing Release forces the upload branch. (3) BODY DISCRIMINATOR, the decisive one — the create branch passes `--generate-notes`, which would overwrite the body; the live body is release-please's changelog (`## [0.2.0](…compare/v0.1.0...v0.2.0)`, `### Features`) and contains NO `What's Changed` / `Full Changelog:` markers. Also: `gh release create` against an existing release errors, and the job succeeded."
log_note: "the job log carries no positive confirmation line — `gh release upload` is silent on success, and Actions echoes the whole run: block, so BOTH branch commands appear in the log as echoed script text. That echo is not execution evidence and was not treated as such."
result: pass

### 38. 09-08 D5 — REL-02 second half: a real v0.1.0 binary self-upgrades to v0.2.0
expected: A genuinely shipped v0.1.0 binary, downloaded fresh, runs its own `codegraph upgrade` and lands on v0.2.0 through the unchanged identity constants
evidence: "re-measured in-session on TWO platforms, both from genuinely shipped v0.1.0 binaries downloaded fresh from v0.1.0's own release. (1) darwin/arm64 — v0.1.0 sha 773223fd… (commit 803b4c9, built 2026-07-14) -> `upgraded to v0.2.0` -> v0.2.0 (commit cce95f3, built 2026-08-01), installed sha a64c1549…. (2) linux/amd64 — run in a debian:bookworm-slim container, `uname -m`=x86_64, v0.1.0 sha 6f05e630… -> `upgraded to v0.2.0` -> installed sha 3cba14af…. For BOTH, installed sha == SLSA-attested subject == checksums entry, asserted programmatically rather than by eye."
scope_note: "Extended beyond the original 09-08 verification, which covered darwin/arm64 only. The linux/amd64 leg was added deliberately because Phase 8's daemon-lock defect (engram azyaqx6b8k) hid for a full phase behind exactly this arm64-verified / amd64-shipped gap. 4 of 6 published platforms (darwin_amd64, linux_arm64, windows_amd64, windows_arm64) remain unexercised by a live upgrade; they share the same internal/upgrade code path and all six verified under cosign."
result: pass

### 39. 09-08 D6 — the supply-chain sha chain closes
expected: The artifact a user receives is byte-identical to the artifact that was attested and signed
evidence: "all four links re-measured in-session on TWO platforms. darwin/arm64 a64c1549… and linux/amd64 3cba14af…, each with: upgrade-installed sha == checksums entry == SLSA-attested subject (asserted programmatically), cosign verify-blob per docs §6(a) verbatim -> `Verified OK`, and TestVerifyReleaseE2E -> PASS. Builder pinned generator_generic_slsa3.yml@refs/tags/v2.1.0."
e2e_note: "TestVerifyReleaseE2E genuinely RAN rather than skipping — its guard (verify_release_e2e_test.go:131) skips unless a real signed artifact is supplied, so it was fed the live downloads via CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE. Both subtests passed (accepts_production_identity, rejects_wrong_identity), and the test log confirms it verified against the PRODUCTION issuer and SAN pattern from verify.go, not the offline fixture trust root used by verify_test.go."
result: pass

### 40. 09-08 D7 — the docs verification defect was real and is fixed
expected: Provenance attests the six binaries, NOT the checksums file; the pre-fix docs told users to verify the checksums file against a bundle that is never published, which would fail on a sound release
evidence: "re-measured in-session while checking test 36 — SLSA subject list = exactly the 6 binaries; codegraph_v0.2.0_checksums.txt is NOT among the attested subjects, though its 6 entries agree with them as sha+name pairs (set equality). The file is the transport for the subject list, not a subject. This is exactly why the pre-fix docs would produce `FAILED: artifact hash does not match provenance subject` on a sound release. docs §6(a) cosign command now runs verbatim to `Verified OK`; PR #6 and PR #7 both MERGED"
demonstrated: "the defect and its fix were both executed in-session against the SAME sound release, using slsa-verifier. FIXED command (binary as subject) -> `Verified build using builder ...generator_generic_slsa3.yml@refs/tags/v2.1.0 at commit cce95f3` / `PASSED: SLSA verification passed`. PRE-FIX command (checksums file as subject) -> `FAILED: expected hash 'c0fb13ad…' not found: artifact hash does not match provenance subject` — the exact failure string 09-08 predicted, on provenance that is entirely valid. This is what a user following the old docs would have seen, and it reads as supply-chain compromise."
docs_state: "docs/RELEASE.md §b now attests over each platform binary directly, carries a dated 'Corrected 2026-08-01' errata note naming the old wrong instruction, and documents the multiple.intoto.jsonl -> codegraph_<tag>.intoto.jsonl rename caveat for v0.2.0-and-earlier."
result: pass

### 41. 09-07 D1 — the skip decision and its recorded rationale are accurate
expected: Maintainer attestation — 09-07 was skipped by your decision after Task 1's blocking-human gate was approved and Task 2 halted on the `Bash(gh pr merge *)` deny; nothing was merged, tagged, released, or pushed; the SUMMARY's reasoning reflects what you actually decided
evidence: "maintainer-attested for the decision itself — not machine-checkable. The falsifiable half WAS checked in-session: no tag exists between v0.1.0 and v0.2.0 (full tag list: milestone-v0.1, v0.0.0-rc.3, v0.1.0, v0.2.0) and no draft releases, so nothing was tagged or released by 09-07; and 09-07-PLAN.md survives on disk changed ONLY in its gate attribute (commit 90c32b8, 1 insertion 1 deletion: gate=\"blocking\" -> gate=\"blocking-human\")."
commit_note: "09-08-SUMMARY.md cites commit ba5a548 for the gate hardening; that commit exists but is NOT on main — it landed as 90c32b8 via PR #3 under the squash-only ruleset. Same change, two hashes. Summaries written pre-merge cite pre-squash hashes; resolve them against main before treating a citation as missing."
result: pass

### 42. 09-07 D2 — the controls added instead of the rehearsal are live
expected: Branch ruleset protect-main is active with 6 required status checks, and both irreversible release checkpoints were hardened to blocking-human
evidence: "ruleset 20157557 protect-main, enforcement=active, target=branch, rules = deletion + non_fast_forward + pull_request + required_status_checks + required_linear_history; allowed_merge_methods=[\"squash\"] only; 6 required checks = test, actionlint, govulncheck (DIST-03), perf regression gate (PERF-02/INDX-06), pr-title, reproducibility (DIST-04). Gate hardening landed on BOTH plans, not just 09-07: 09-08's checkpoint:decision also went blocking -> blocking-human."
bypass_note: "bypass actor is RepositoryRole 5 (admin) with bypass_mode=always — verified, and consistent with what 09-07-SUMMARY.md stated plainly. This ruleset gates the pipeline, not the maintainer."
result: pass

## Summary

total: 42
passed: 42
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
