---
phase: 05-agent-reach-capability-model-skill-in-every-harness
fixed_at: 2026-09-18T00:00:00Z
review_path: .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-REVIEW.md
iteration: 2
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-09-18T00:00:00Z
**Source review:** .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 2 (CR-01, WR-01; IN-01/IN-02 out of scope for `critical_warning`)
- Fixed: 2
- Skipped: 0

**Verification environment:** main checkout at `/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.14.0-milestone` (no worktree — this task's config explicitly overrode the standard isolated-worktree flow: "Work only in ... on branch gsd/v0.14.0-milestone (no worktree)"). `go build ./...`, `go vet ./internal/...`, and `go test ./internal/agents/ ./internal/cli/... -count=1` all ran green after both commits (`GOTOOLCHAIN=go1.26.6`).

## Fixed Issues

### WR-01: No test exercises Cursor's/opencode's real `Install`/`Uninstall` path against a corrupted shared manifest

**Files modified:** `internal/agents/cursor_test.go`
**Commit:** `9bf91e7c` (`test(05): CR-01 regression -- shared skill dir corrupted-manifest fallback must not falsely attribute claude via cursor's real Install/Uninstall path`)
**Applied fix:** Added two regression tests driving the real production path (`cursorTarget{}.Install()`/`Uninstall()`, not the shared internals directly):

1. `TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude` — fresh `fakeHome`, no Claude directory anywhere, no D-17 symlink. Installs Cursor, corrupts the shared manifest, reinstalls (self-heal), asserts `Targets` never includes Claude and equals exactly `[cursor]`, then uninstalls and asserts the shared directory is fully removed.
2. `TestCursor_CorruptedManifestAtSharedDir_ClaudeSymlinked_StillAttributesClaude` — positive control using `symlinkedClaudeLayout` (Claude's directory really is a symlink onto the shared directory, D-17) with a legacy (no-`targets`-key) manifest seeded at the shared directory. Asserts the fallback still resolves to `[Claude, Cursor]` after install, and that uninstalling Cursor leaves Claude's package (SKILL.md + manifest, `Targets = [claude]`) fully intact.

**RED confirmation (test 1, run against the pre-fix code before the CR-01 commit below):**
```
cursor_test.go:253: CONFIRMED BUG: Claude falsely attributed as requester of .../.agents/skills/codegraph with no claude install/symlink ever present: targets=[claude cursor]
--- FAIL: TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude (0.00s)
--- PASS: TestCursor_CorruptedManifestAtSharedDir_ClaudeSymlinked_StillAttributesClaude (0.00s)
FAIL
```
(Test 2, the positive control, passes both before and after the CR-01 fix — it pins the genuine case that must never regress.)

**GREEN confirmation:** after the CR-01 fix commit below, both tests pass; `go test ./internal/agents/ ./internal/cli/... -count=1` is fully green (including `TestOwnershipExactIdentity` and every existing iteration-1 regression test).

### CR-01: `declaredSkillFallback` compares against the wrong reference path, so the "assume Claude" fallback still fires unconditionally for Cursor/opencode's shared skill directory

**Files modified:** `internal/agents/capabilities.go`
**Commit:** `795a0c19` (`fix(05): CR-01 compare declaredSkillFallback against claude's own dir, not the shared path`)
**Applied fix:** `declaredSkillFallback` (capabilities.go) now resolves `claudeSkillDirPath(loc)` and compares `dir` against it with `sameSkillDir` — reusing the exact D-17-aware comparison `installSkillPackageWithFallback` already performs for its advisory note — instead of comparing `dir` against `sharedSkillDirPath(loc)`, which was tautologically true for Cursor/opencode (their declared skill directory IS the shared path by definition) regardless of whether Claude was installed at all. "Assume Claude" now only fires when Claude's own directory and the target's declared directory are the same physical directory on this machine (the genuine `npx skills` symlink case, D-17); every other case — including the common Cursor/opencode-only, no-Claude machine, and every harness-exclusive directory (Gemini, Kiro, Antigravity) — falls back to `[requester]`.

Verified against the current tree: the two regression tests above (committed first, RED against the pre-fix code) now pass; `TestGemini_CorruptedManifestAtHarnessExclusiveDir_DoesNotFalselyAttributeClaude` and every D-17 symlink test in `claude_symlink_test.go` (`TestSymlinkedSkillDir_*`, `TestConfiguredSkillLocations_RequiresClaudeInTargets`, `TestSharedSkillWriter_NotesSameDirAsClaude`) remain green — no regression to the genuine Claude-symlink case this fallback exists to preserve.

## Skipped Issues

None — both in-scope findings were fixed.

---

_Fixed: 2026-09-18T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_

## --auto loop summary (orchestrator)

| Iteration | Review result | Fix commits |
|---|---|---|
| 1 | CR-01 (corrupted harness-exclusive manifest read as owned by claude), WR-01 (`describeDeclaredPaths` dropped `HookFiles` errors), IN-01, IN-02 | `6eedc2e9` test, `a5d4e23a` fix (CR-01: explicit per-call-site fallback); `c99e1085` test, `4b5482e9` fix (WR-01: panic on `errHookFilesUndeclared` — reachable only by a misdeclared target; `DescribePaths` has no production CLI caller) |
| 2 | CR-01 residual (fallback compared against the shared path, always true for Cursor/opencode), WR-01 (no test through the real Cursor/opencode path), IN-01, IN-02 | `9bf91e7c` test, `795a0c19` fix (this report, above) |
| 3 | critical 0, warning 0; Info only: IN-01 (dead `caps.MCPConfig == nil` check, `printconfigstyle.go`), IN-02 (`ActionKeptForeign` renders with the default role, no test), IN-03 (CR-01 comments narrated the review history) | `2ac49ad6` — IN-03 comment-only tidy by the orchestrator (the loop's three iterations were spent); IN-01 and IN-02 left open as Info |

In-scope findings (critical + warning) converged to zero at iteration 3. Every fix landed after a RED regression test (pasted `--- FAIL:` lines above and in the iteration-1 report, kept in git history of this file's inputs via the commits listed).
