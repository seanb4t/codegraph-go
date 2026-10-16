---
status: diagnosed
phase: 05-release-chain-release-please-retirement
source: [05-VERIFICATION.md]
started: 2026-10-14T23:45:00Z
updated: 2026-10-16T20:08:46.888Z
---

## Current Test

[testing complete]

## Tests

### 1. Decide the fate of the stale release-candidate row in docs/RELEASE-PROCEDURES.md section 2 (IN-03): fix it now (one table row, plus the two rc mentions at lines 13 and 339) or accept it as a known residue

expected: Section 2 no longer presents a hand-pushed `v0.0.0-rc.N` tag as a normal, working tag shape, consistent with D-15 and with section 4's 'No hand-pushed prerelease tag' paragraph
result: issue
reported: "fix it"
severity: major

### 2. After the maintainer's next push to PR #88, read the linux-cross-canary run (its cross-build job runs `task release:dry-run` on the macOS runner class) using the read-only commands recorded in 05-03-SUMMARY.md

expected: The cross-build job exits 0 with the RELEASE-NOTES-EVIDENCE line for the newest .changes/v*.md and the four-binary success line, and the linux binaries run on linux/amd64 and linux/arm64
result: skipped
reason: "Covered by PR #88's linux-cross-canary run on the next push (it triggers on Taskfile.yml); a deferred CI observation, not a human check. release:dry-run already exited 0 locally twice (05-03-SUMMARY)."

### 3. 05-01 D6: The live release PR is opened by the App on main and its required checks report

expected: The live release PR is opened by the App on main and its required checks report
result: skipped
reason: "Deferred to Phase 6 by 05-01 itself (needs the App secrets and a real push to main); the first natural release run on main shows it. Same disposition as test 2."

### 4. 05-02 D7: A real release-PR merge fires the CHANGELOG.md path filter, the App-token tag push starts release.yml, and the milestone merge is a no-op

expected: A real release-PR merge fires the CHANGELOG.md path filter, the App-token tag push starts release.yml, and the milestone merge is a no-op
result: skipped
reason: "Deferred to Phase 6 by 05-02 itself (live GitHub behavior needs the App secrets and a real push to main); the first natural release run on main shows it. Same disposition as test 2."

### 5. 05-06 D5: Readability and accuracy of the rewritten docs/RELEASE-PROCEDURES.md runbook prose

expected: Readability and accuracy of the rewritten prose against the chain as built and as it will run live
result: pass

### 6. One commit deletes the three retired files and removes their inScopeJobs entry, usesOnlyJobExceptions entry and inScopeWorkflowFiles element; it touches exactly those four paths and the package passes at that commit

expected: One commit deletes the three retired files and removes their inScopeJobs entry, usesOnlyJobExceptions entry and inScopeWorkflowFiles element; it touches exactly those four paths and the package passes at that commit
result: pass
source: automated
coverage_id: D2

### 7. The census instrument finds planted content, filename and in-scope .changes/header.tpl.md references and skips the .planning/ and .changes/unreleased/ plants; on the real tree it prints nothing, and the history files still hold the hits the exclusion skips

expected: The census instrument finds planted content, filename and in-scope .changes/header.tpl.md references and skips the .planning/ and .changes/unreleased/ plants (each confirmed to match under an unexcluded grep); on the real tree it prints nothing, and the history files still hold the hits the exclusion skips
result: pass
source: automated
coverage_id: D3

### 8. The deletion is the last change outside .planning/; no CI-skip phrase in any commit since 5a8603f5; no scratch worktree remains; changie latest is the version source

expected: The deletion is the last change outside .planning/; no CI-skip phrase in any commit since 5a8603f5; no scratch worktree remains; changie latest is the version source
result: pass
source: automated
coverage_id: D4

### 9. Phase gate: task test, check:changie 14 of 14, check:fragment-policy 46 of 46, lint:actions, the upgrade package, release.yml one comment-only hunk against 7b12cb92, verify.go / CHANGELOG.md / .changes/v*.md equal to 5a8603f5

expected: Phase gate: task test, check:changie 14 of 14, check:fragment-policy 46 of 46, lint:actions, the upgrade package, release.yml one comment-only hunk against 7b12cb92, verify.go / CHANGELOG.md / .changes/v*.md equal to 5a8603f5
result: pass
source: automated
coverage_id: D5

## Summary

total: 9
passed: 5
issues: 1
pending: 0
skipped: 3
blocked: 0

## Gaps

- gap_id: G-05-1
  truth: "Section 2 no longer presents a hand-pushed `v0.0.0-rc.N` tag as a normal, working tag shape, consistent with D-15 and with section 4's 'No hand-pushed prerelease tag' paragraph"
  status: failed
  reason: "User reported: fix it"
  severity: major
  test: 1
  root_cause: "Plan 05-06 (8d2884d9) added the D-15 'No hand-pushed prerelease tag' paragraph to section 4 but left three older rc passages from 08-09 (be456901) untouched; its census pattern targeted release-please references, not rc tag wording (review finding IN-03, disposition open)"
  artifacts:
    - path: "docs/RELEASE-PROCEDURES.md"
      issue: "line 48: section 2 tag-conventions table row 'Release candidate | v0.0.0-rc.3 | Prerelease ... | Yes' presents a hand-pushed rc tag as a working shape; under D-15 it starts release.yml but release:goreleaser refuses it (no .changes/<tag>.md)"
    - path: "docs/RELEASE-PROCEDURES.md"
      issue: "line 13: section 1 'Before tagging anything (rc or stable)' implies rc is a live tag kind"
    - path: "docs/RELEASE-PROCEDURES.md"
      issue: "line 339: section 7 'Prereleases (-rc.N) are safer to delete/retag' describes rc tags as a normal transient convention"
  missing:
    - "Rewrite or remove the section 2 rc table row so it agrees with section 4's D-15 paragraph (point to section 4)"
    - "Drop 'rc or' from the section 1 pre-tag gate sentence"
    - "Reword or remove the section 7 rc retag sentence"
    - "Leave historical rc.1 references (line 15, Taskfile.yml:5437) and the correct D-15 text (lines 172-177, Taskfile.yml:2958) unchanged"
  debug_session: ""
