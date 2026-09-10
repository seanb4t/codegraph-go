---
status: testing
phase: 08-tmux-real-pty-harness
source: [08-VERIFICATION.md]
started: 2026-09-10T18:08:06.378Z
updated: 2026-09-10T18:08:06.378Z
---

## Current Test

number: 1
name: The tmux-e2e job actually fired its executed-count assertion on a real CI run
expected: |
  The `tmux-e2e` job log shows `test:tmux: executed=5 skipped=0 expected=5`, the job status is
  success, and the `tmux -V` assertion line shows the observed version matched
  TMUX_EXPECTED_VERSION (or, on the deliberately-unpinned first run, printed the real string for
  committing per D-11's bootstrap).
awaiting: user response

## Tests

### 1. The tmux-e2e job actually fired its executed-count assertion on a real CI run

expected: The `tmux-e2e` job log shows `test:tmux: executed=5 skipped=0 expected=5`, the job status is success, and the `tmux -V` assertion line shows the observed version matched TMUX_EXPECTED_VERSION (or, on the deliberately-unpinned first run, printed the real string for committing per D-11's bootstrap).
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
