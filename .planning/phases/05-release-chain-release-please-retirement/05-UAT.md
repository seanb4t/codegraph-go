---
status: complete
phase: 05-release-chain-release-please-retirement
source: [05-VERIFICATION.md]
started: 2026-10-14T23:45:00Z
updated: 2026-10-16T19:48:17.065Z
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

## Summary

total: 2
passed: 0
issues: 1
pending: 0
skipped: 1
blocked: 0

## Gaps

- gap_id: G-05-1
  truth: "Section 2 no longer presents a hand-pushed `v0.0.0-rc.N` tag as a normal, working tag shape, consistent with D-15 and with section 4's 'No hand-pushed prerelease tag' paragraph"
  status: failed
  reason: "User reported: fix it"
  severity: major
  test: 1
  artifacts: []
  missing: []
