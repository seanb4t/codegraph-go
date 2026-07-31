---
status: complete
phase: 09-release-please-and-goreleaser
scope: "plans 09-01..09-06 only — 09-07/09-08 not executed; this is NOT full-phase verification"
source: [09-01-SUMMARY.md, 09-02-SUMMARY.md, 09-03-SUMMARY.md, 09-04-SUMMARY.md, 09-05-SUMMARY.md, 09-06-SUMMARY.md]
started: 2026-07-30T00:00:00Z
updated: 2026-07-30T01:20:00Z
---

> **Scoped UAT.** Phase 09 is 6/8 plans complete. Plans 09-07 (disposable live proof) and
> 09-08 (the real release cut) have not run, so REL-02 is not yet met and no release exists.
> This session verifies only what plans 09-01..09-06 delivered.

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

## Summary

total: 33
passed: 33
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
