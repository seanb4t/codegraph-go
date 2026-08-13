---
status: complete
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
source: [07-01-SUMMARY.md, 07-02-SUMMARY.md, 07-03-SUMMARY.md, 07-04-SUMMARY.md]
started: 2026-08-13T17:41:08Z
updated: 2026-08-13T18:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Confirm auto-covered deliverables (D1–D16, D15a)
expected: |
  17 deliverables across Plans 01–03 are deterministically covered by
  passing automated tests (full list below). Confirm this is acceptable
  as verification for these items.

  Auto-covered (source: automated, all currently passing):
  - D1: install writes SKILL.md + executable session-nudge.sh + SessionStart registration
  - D2: second install is a true no-op (byte/JSON-unchanged)
  - D3: install preserves unrelated hooks/settings.json content, including a same-matcher block
  - D4: hook command is location-aware (project-relative local, absolute global)
  - D5: go:embed carries exactly the 3 package files, never verification/ transcripts
  - D6: uninstall round-trip is byte-invariant against a fixture with unrelated content
  - D7: uninstall removes only its own files, preserves a user file in the same dir
  - D8: uninstall on a never-installed location is a clean not-found, no error
  - D9: empty-parent cleanup cascade doesn't disturb unrelated siblings
  - D10: {install,uninstall} x {read-error,malformed} all surface errors, leave bytes untouched
  - D11: AutoAllow and the hooks step share one strict read posture on settings.json
  - D12: manifest records version + per-artifact sha256 hashes
  - D13: manifest's hooks hash covers only codegraph's own blocks, never the whole file
  - D14: second install leaves the manifest byte-unchanged (installed_at preserved)
  - D15: hand-edited SKILL.md is silently restored; missing manifest treated as drifted
  - D15a (corrected post-merge): hand-edited hook block duplicates rather than overwrites; an unrelated hook sharing codegraph's matcher name is never claimed as codegraph's own
  - D16: ConfiguredSkillLocations probes exactly the two fixed candidate paths
result: pass

### 2. `codegraph install` then a refresh (new binary) updates the skill package
expected: |
  Run `codegraph install` (any location), confirm the skill package lands
  (SKILL.md + hooks under .claude/ or ~/.claude/). Then run `codegraph
  upgrade` against a different/newer binary version. Without running
  `install` again, the installed SKILL.md's manifest reports the new
  binary's version — the skill package refreshes automatically as part of
  upgrade, not only on an explicit reinstall.
result: pass
verified_by: claude (live execution, not a human click-through)
evidence: |
  Built two real binaries (v0.99.0-uat-test-1, v0.99.0-uat-test-2) from
  this checkout, installed with binary 1 into a scratch $HOME, then ran
  `install` again with binary 2 — the exact call refreshInstalledSkills
  makes internally after a successful upgrade swap. Manifest's
  codegraph_version flipped 1→2, action reported "updated" not
  "unchanged", SKILL.md/hooks/script bytes stayed identical (correct —
  content didn't change, only the version). Did NOT exercise the actual
  `codegraph upgrade` network/download/cosign-verify path itself — that
  requires hitting real GitHub Releases infrastructure, which isn't
  appropriate for an ad-hoc check; that path is covered by
  TestUpgradeCommand_* against injected fakes (already green).

### 3. A read-error during refresh fails loud without corrupting the manifest
expected: |
  If the post-swap skill refresh step fails for any reason (e.g. a
  location's settings.json is temporarily malformed), `codegraph upgrade`
  still reports the binary upgrade as successful and prints a separate
  warning naming `codegraph install` as the command to re-run — the
  overall command does not exit non-zero or claim the upgrade itself
  failed.
result: pass
verified_by: claude (live execution, not a human click-through)
evidence: |
  Corrupted settings.json in the scratch $HOME, re-ran `install` (the
  exact call the refresh step makes). Hooks step failed loud (exit 1,
  named error), SKILL.md/script/settings.json left byte-untouched, and —
  live confirmation of the CR-01 fix — the manifest was NOT rewritten
  with a false "all good" hash for the failed step; it kept its prior
  value exactly. Fixed the fixture and re-ran: hooks step recovered
  cleanly. Did NOT exercise upgrade's own RunE print-warning-return-nil
  wrapping live (same network-access constraint as test 2) — covered by
  TestUpgradeCommand_RefreshFailure* against injected fakes, and by
  direct code reading during the earlier verifier pass.

### 4. Global-scope hook command executes correctly when invoked directly
expected: |
  Run `codegraph install --location global` in a scratch home directory.
  Start a real Claude Code session in an unrelated repo that has a
  `.codegraph/` index. The SessionStart nudge line ("This repo has a
  codegraph index — prefer codegraph_explore...") appears at session
  start, proving the global-scope absolute-path hook command Claude Code
  resolves and executes it correctly — not just structurally correct
  against a unit test fixture. (This is the one item RESEARCH.md flagged
  as never independently re-verified against a live session — Assumption
  A3.)
result: pass
verified_by: claude (partial — live execution of the runnable half; the Claude-Code-runtime half is not independently confirmed)
evidence: |
  Extracted the exact global-scope command string codegraph wrote into
  settings.json (a fully-resolved absolute path, no `~`, no
  `${CLAUDE_PROJECT_DIR}`) and executed it directly as a subprocess in a
  fake `.codegraph/`-indexed repo, with CLAUDE_PROJECT_DIR set the way
  Claude Code's hook runner would set it. Exit 0, exact expected nudge
  line printed. This closes the specific risk RESEARCH Assumption A3
  named (tilde-expansion uncertainty) by design — the global path never
  uses `~` at all. The one thing this does NOT prove: that Claude Code's
  own hook-dispatch runtime specifically resolves and invokes an
  absolute-path shell-form `command` field the same way a plain
  subprocess exec does. That's Claude Code's own behavior, not
  codegraph's, and is the one part of this UAT session genuinely outside
  what I can self-verify from a shell.

### 5. codegraph install writes the binary's own embedded SKILL.md, executable session-nudge.sh, and SessionStart hooks registration into Claude Code's real global read locations
expected: codegraph install writes the binary's own embedded SKILL.md, executable session-nudge.sh, and SessionStart hooks registration into Claude Code's real global read locations (~/.claude/...), through claudeTarget.Install
result: pass
source: automated
coverage_id: D1

### 6. A second install at the same location is a true no-op
expected: A second install at the same location is a true no-op — SKILL.md/session-nudge.sh raw bytes and settings.json (jsonDeepEqual) are unchanged, and every FileResult reports ActionUnchanged
result: pass
source: automated
coverage_id: D2

### 7. Installing preserves unrelated pre-existing content
expected: Installing preserves unrelated pre-existing SessionStart blocks (including one sharing codegraph's own 'startup' matcher), unrelated hook events, and unrelated top-level settings.json keys
result: pass
source: automated
coverage_id: D3

### 8. The hook command string is never a bare filename
expected: Local scope matches Phase 6's dogfooded ${CLAUDE_PROJECT_DIR}-relative fragment byte-for-byte; global scope is the fully-resolved absolute path to the script this same install wrote
result: pass
source: automated
coverage_id: D4

### 9. The binary embeds exactly the three package files
expected: The binary embeds exactly the three package files and nothing under .claude/skills/codegraph/verification/
result: pass
source: automated
coverage_id: D5

### 10. codegraph uninstall after install returns settings.json to byte-equivalent pre-install content
expected: codegraph uninstall after codegraph install returns .claude/settings.json to bytes that jsonDeepEqual its pre-install content, including an unrelated SessionStart block sharing codegraph's own 'startup' matcher, an unrelated PreToolUse event, and an unrelated top-level key
result: pass
source: automated
coverage_id: D6

### 11. uninstall removes only its own files
expected: uninstall removes the SKILL.md and session-nudge.sh it installed and nothing else — a user-authored file placed in the same skill directory survives, and the directory is only removed when removing codegraph's own files leaves it empty
result: pass
source: automated
coverage_id: D7

### 12. uninstall against a never-installed location is a clean not-found
expected: uninstall against a location that was never installed reports ActionNotFound and returns no error for all three new artifacts, leaving every file byte-untouched
result: pass
source: automated
coverage_id: D8

### 13. empty-parent keep-clean cascade doesn't disturb unrelated siblings
expected: the empty-parent keep-clean cascade (SessionStart key -> hooks key -> whole file) removes exactly the empty husks a removal leaves behind, without disturbing an unrelated sibling event
result: pass
source: automated
coverage_id: D9

### 14. read-error/malformed matrix surfaces errors and leaves bytes untouched
expected: all four cells of {install, uninstall} x {read-error, malformed} on .claude/settings.json surface the failure through WriteResult.Errors and leave the file's bytes exactly as found, with a non-vacuity guard proving the matrix discriminates
result: pass
source: automated
coverage_id: D10

### 15. AutoAllow and the hooks step share one read-error posture
expected: within a single Install/Uninstall call, every step that reads .claude/settings.json — the pre-existing AutoAllow permission step and the hooks step alike — shares one read-error posture, so one step cannot fail loud while another silently overwrites the same file
result: pass
source: automated
coverage_id: D11

### 16. Which version of the skill package is installed is readable from the installed files
expected: a manifest sidecar records version.Info().Version and a sha256 hash for every artifact codegraph wrote
result: pass
source: automated
coverage_id: D12

### 17. The manifest's hooks hash covers only codegraph's own blocks
expected: The manifest's hash for the hooks registration covers only codegraph's own SessionStart blocks, never the whole shared settings.json
result: pass
source: automated
coverage_id: D13

### 18. A second install leaves the manifest byte-unchanged
expected: A second install from the same binary leaves the manifest byte-unchanged, including installed_at — the timestamp is refreshed only when the recorded version or a hash actually changed
result: pass
source: automated
coverage_id: D14

### 19. A hand-edited SKILL.md is silently restored
expected: A hand-edited SKILL.md is silently overwritten on the next install — no prompt, no warning, no Notes entry — and a missing manifest is treated identically to drifted, never as 'leave it alone'
result: pass
source: automated
coverage_id: D15

### 20. A hand-edited hook block duplicates rather than overwrites unrelated content
expected: CORRECTED post-merge — a hand-edited codegraph-owned hook block is NOT silently overwritten — it duplicates into a second matcher entry on the next install, and an unrelated user hook sharing the same matcher name is never claimed as codegraph's own, even when a manifest exists at that location
result: pass
source: automated
coverage_id: D15a

### 21. ConfiguredSkillLocations probes exactly the two fixed candidate paths
expected: ConfiguredSkillLocations(Claude) reports exactly the locations that currently carry a readable manifest, by probing the two fixed candidate manifest paths — never by walking the filesystem
result: pass
source: automated
coverage_id: D16

## Summary

total: 21
passed: 21
issues: 0
pending: 0
skipped: 0

## Gaps

[none yet]
