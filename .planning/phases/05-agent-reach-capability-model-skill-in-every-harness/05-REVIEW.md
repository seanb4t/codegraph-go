---
phase: 05-agent-reach-capability-model-skill-in-every-harness
reviewed: 2026-09-18T00:00:00Z
depth: deep
files_reviewed: 34
files_reviewed_list:
  - docs/CLI-REFERENCE.md
  - internal/agents/antigravity.go
  - internal/agents/antigravity_test.go
  - internal/agents/capabilities.go
  - internal/agents/capabilities_test.go
  - internal/agents/claude.go
  - internal/agents/claude_skillpackage_test.go
  - internal/agents/claude_symlink_test.go
  - internal/agents/codex.go
  - internal/agents/cursor.go
  - internal/agents/cursor_test.go
  - internal/agents/gemini.go
  - internal/agents/gemini_test.go
  - internal/agents/hermes.go
  - internal/agents/kiro.go
  - internal/agents/kiro_test.go
  - internal/agents/manifest.go
  - internal/agents/manifest_test.go
  - internal/agents/opencode.go
  - internal/agents/opencode_test.go
  - internal/agents/ownership_test.go
  - internal/agents/registry_test.go
  - internal/agents/shared.go
  - internal/agents/skillfrontmatter_test.go
  - internal/agents/skillshared.go
  - internal/agents/skillshared_test.go
  - internal/agents/types.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/plain_golden_test.go
  - internal/cli/printconfigstyle.go
  - internal/cli/printconfigstyle_test.go
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/uninstall.go
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-09-18T00:00:00Z
**Depth:** deep
**Files Reviewed:** 34
**Status:** issues_found

## Summary

This is iteration 3 (final) of the review. Iteration 2 (`05-REVIEW.iter3.md`)
found CR-01 (still-defective fallback comparison, relocated one level up
from iteration 1's original defect) and WR-01 (missing regression coverage
through the real `cursorTarget`/`opencodeTarget` entry points that let it
ship). `05-REVIEW-FIX.md` reports both fixed in commits `9bf91e7c` (test,
RED-first) and `795a0c19` (fix). Both are re-verified here directly against
the current source — not the fixer's narrative — plus a full
`go build`/`go vet`/`go test` pass and a fresh reading of every consuming
call site.

**CR-01 is genuinely fixed.** `declaredSkillFallback` (`capabilities.go:214-227`)
now resolves `claudeSkillDirPath(loc)` and compares the target's declared
directory against it with `sameSkillDir` — the same D-17-aware comparison
`installSkillPackageWithFallback` already performs for its advisory note —
instead of comparing against `sharedSkillDirPath(loc)`, which was
tautologically true for Cursor/opencode regardless of whether Claude was
even installed. Traced through all three cases the task asked to verify:

- **Harness-exclusive dirs** (Gemini `.gemini/skills/`, Kiro `.kiro/skills/`,
  Antigravity `~/.gemini/config/skills/`) never equal `claudeSkillDirPath`,
  so `sameSkillDir` is `false` and the fallback is `[requester]`.
  `TestGemini_CorruptedManifestAtHarnessExclusiveDir_DoesNotFalselyAttributeClaude`
  (`gemini_test.go:228-267`) pins this through `geminiTarget{}.Install()`/
  `.Uninstall()`; Kiro and Antigravity share the identical
  `installDeclaredSkill`/`declaredSkillFallback` code path with a directory
  shape that can never coincide with Claude's, so the untested-but-identical
  branch is not a new gap.
- **Shared dir, no Claude / unrelated Claude dir** — Cursor/opencode's
  declared directory IS `sharedSkillDirPath`, which no longer equals
  `claudeSkillDirPath` unless a real symlink exists. Pinned by
  `TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude`
  (`cursor_test.go:225-266`), which drives the real `cursorTarget{}.Install()`
  (self-heal over a corrupted manifest) then `.Uninstall()` and asserts both
  that `Targets` never includes Claude and that the shared directory is
  fully swept on uninstall of its sole real requester (the exact D-08
  violation CR-01 closes).
- **Shared dir physically equal to Claude's dir (D-17 symlink, including
  dangling)** — pinned by the positive control
  `TestCursor_CorruptedManifestAtSharedDir_ClaudeSymlinked_StillAttributesClaude`
  (`cursor_test.go:278-324`), using `symlinkedClaudeLayout` (a real relative
  symlink `~/.claude/skills/codegraph -> ../../.agents/skills/codegraph`,
  the exact `npx skills` shape D-17 targets): a legacy (no-`targets`-key)
  manifest at the shared directory still resolves to `[Claude, Cursor]`
  after install, and uninstalling Cursor leaves Claude's package (SKILL.md +
  manifest, `Targets = [claude]`) fully intact. The dangling-symlink
  sub-case is separately covered by `resolveSkillDir`'s own handling
  (`skillshared.go:389-444`, exercised by the pre-existing D-17 symlink
  suite in `claude_symlink_test.go`), unchanged by this fix and still green.
- **Uninstall uses the same fallback as install** — confirmed by direct
  reading: `installDeclaredSkill` and `uninstallDeclaredSkill`
  (`capabilities.go:238-275`) both call `declaredSkillFallback(dir, loc,
  t.ID())` identically before delegating to
  `installSkillPackageWithFallback`/`uninstallSkillPackageWithFallback`, and
  both regression tests above exercise the fallback through both halves of
  the same target's Install/Uninstall pair.

**WR-01 is genuinely fixed.** The two new tests in `cursor_test.go` drive
the production `Install()`/`Uninstall()` entry points rather than calling
`installSkillPackage`/`uninstallSkillPackage` directly, closing exactly the
coverage gap that let the relocated CR-01 defect ship undetected.

Full re-verification environment: `GOTOOLCHAIN=go1.26.6 go build ./...`,
`go vet` (every `internal/...` package except `internal/daemon`, run
separately per this task's constraint), and
`go test ./internal/agents/... ./internal/cli/...` all pass cleanly
(`internal/agents` 3.9s, `internal/cli` and its subpackages all green, no
skips, no `-short` gating).

No new Critical or Warning findings. Three Info items: IN-01 and IN-02
carried forward unchanged from the prior iteration (still valid, still
low-severity, still out of scope for a critical/warning-only fix pass), and
one new item (IN-03) on the fix's own doc comments, per this task's request
to judge whether new/changed comments describe current behavior versus
narrating the review's own history.

## Info

### IN-01: Dead defensive check — `caps.MCPConfig == nil` in `configStyleFields` can never execute

**File:** `internal/cli/printconfigstyle.go:41-43`

**Issue:** Unchanged since the prior iteration: `configStyleFields` guards against `caps.MCPConfig == nil` before calling it, but every one of the 8 registered targets' `Capabilities()` literals sets `MCPConfig` unconditionally (documented as "Required for every Location in Scopes," `capabilities.go:83-85`). Given the current registry, this branch is unreachable and untestable. Harmless.

**Fix:** No action required; if kept as defense-in-depth, a one-line comment noting it is currently unreachable given D-02's contract would save a future reader the trouble of trying to hit it via the CLI.

### IN-02: `ActionKeptForeign` has no dedicated styled role and no test coverage of its rendering

**File:** `internal/cli/install.go:203-218` (`printAgentResults`)

**Issue:** Unchanged since the prior iteration: the styled-output branch maps `FileAction` to a `present.Role` via an explicit switch (`ActionUnchanged, ActionKept, ActionNotFound` → `RoleLabel`); `ActionKeptForeign` falls through to `default: actionRole = present.RoleWarning` with no comment stating this is intentional, and no test asserts on the styled rendering of a `kept (foreign):` line specifically.

**Fix:** If `RoleWarning` is the intended treatment, add a one-line comment next to the switch stating so and a coverage case asserting the `kept (foreign):` line renders with the warning role.

### IN-03: The CR-01 fix's doc comments narrate the review/iteration history rather than only describing current behavior

**File:** `internal/agents/capabilities.go:192-213` (`declaredSkillFallback`), `internal/agents/cursor_test.go:209-224, 268-277`

**Issue:** The doc comment this iteration's fix rewrote is accurate (independently verified above) but spends most of its length narrating the defect's history rather than describing what the function does and why: "(05-REVIEW.md, re-verified in iteration 2)", "Comparing against the shared path itself (an earlier draft of this fix) is tautologically true for Cursor and opencode... it fires unconditionally regardless of whether Claude is even installed" (`capabilities.go:192-213`). A reader six months from now with no access to `05-REVIEW.md` gains little from "an earlier draft of this fix" — the comment would be equally complete, and more durable, stated purely as "why `claudeSkillDirPath` and not `sharedSkillDirPath`" without the iteration-numbered edit history. The two new test doc comments in `cursor_test.go` (`TestCursor_CorruptedManifestAtSharedDir_NoClaudePresent_DoesNotFalselyAttributeClaude` and its positive-control sibling) do the same: "is CR-01's iteration-2 regression test", "unlike ...Gemini..., which exercises a harness-exclusive directory, this drives the REAL production shared-directory path -- ... the iteration-1 defect". This is consistent with the project's established convention elsewhere in this phase (D-15 explicitly requires citing a commit hash in one specific test's doc comment for a differential this codebase cares about long-term), so it is not a one-off lapse — but it means the convention has spread from "cite one commit for one specific closed differential" to "narrate every iteration of every review pass," which is a different, more maintenance-costly thing: every future edit to this function invites another paragraph of superseded review-iteration commentary rather than a comment that stays accurate by describing only the current, shipped behavior.

**Fix:** Trim both comments to state the current behavior and its rationale (why the comparison target is `claudeSkillDirPath`, why that's the only case "assume Claude" is justified) without dating it to a specific review iteration or contrasting it against an "earlier draft." Where traceability to the finding is valuable, a single terse citation (as D-15 already establishes for `242ec0a`) is enough — the iteration-by-iteration narrative belongs in `05-REVIEW-FIX.md`/`05-REVIEW.iter*.md`, which already exist for exactly this purpose and are not source comments future maintainers must keep re-reading.

---

_Reviewed: 2026-09-18T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
