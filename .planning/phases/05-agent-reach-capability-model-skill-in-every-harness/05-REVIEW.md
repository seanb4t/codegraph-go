---
phase: 05-agent-reach-capability-model-skill-in-every-harness
reviewed: 2026-09-18T00:00:00Z
depth: deep
files_reviewed: 33
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
  - internal/cli/testdata/plain/print-config-style-local.golden
  - internal/cli/testdata/plain/print-config-style.golden
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/uninstall.go
findings:
  critical: 1
  warning: 1
  info: 2
  total: 4
status: issues_found
---

# Phase 05: Code Review Report

**Reviewed:** 2026-09-18T00:00:00Z
**Depth:** deep
**Files Reviewed:** 33 (git-diff scope; `docs/CLI-REFERENCE.md` counted once)
**Status:** issues_found

## Summary

Reviewed the full capability-table + shared-skill-package implementation
(`internal/agents/capabilities.go`, `skillshared.go`, `manifest.go`, all
eight per-target files, and the `--print-config-style` CLI surface) at
deep depth, tracing call chains from `AgentTarget.Capabilities()` through
`describeDeclaredPaths`/`installDeclaredSkill`/`uninstallDeclaredSkill`
into the shared manifest writer, and cross-checking every finding against
`05-CONTEXT.md`'s locked decisions (D-00…D-17) before treating anything as
a defect.

The capability-table derivation work (D-01…D-04) is careful and
well-tested: `TestCapabilitiesTableDrivesDerivations` and
`TestCapabilitiesMatchInstallWrites` are genuine two-directional guards
built from independent oracles, and `--print-config-style` is correctly
read-only and byte-stable. The ownership/ reversal machinery (D-13…D-17)
has an extensive, well-constructed test suite (`ownership_test.go`,
`skillshared_test.go`, `claude_symlink_test.go`) that reproduces the exact
242ec0a precondition rather than a weaker one.

However, one design decision that was correctly scoped for the *shared*
skill directory (D-07's "an unreadable manifest is read as owned by
Claude, since Claude was the only pre-phase writer") was mechanically
reused, unmodified, for the *harness-exclusive* skill directories this
phase newly introduces (Gemini, Kiro, Antigravity), where Claude never had
any pre-phase manifest and the justification does not hold. This produces
a demonstrable defect: a corrupted manifest at one of those directories
causes the next install to falsely record Claude as a co-owner, which then
prevents `uninstall` from ever fully removing the directory once the real
(sole) requester leaves — a permanent violation of D-08's "deleted only
when `targets` becomes empty" guarantee. Reproduced live against the
current tree (see CR-01). No test in the added suite exercises a corrupted
manifest at a harness-exclusive directory, so this gap shipped unnoticed.

## Critical Issues

### CR-01: Corrupted manifest at a harness-exclusive skill directory falsely attributes Claude as a co-owner, permanently blocking uninstall cleanup

**File:** `internal/agents/manifest.go:87-100` (`manifestRequesters`), consumed by `internal/agents/skillshared.go:178-204` (`recordSkillManifest`) and `internal/agents/skillshared.go:258-339` (`uninstallSkillPackage`)

**Issue:**

`manifestRequesters` folds an unreadable/corrupted manifest into `[]TargetID{Claude}` (manifest.go:88-90, 94-96). The doc comment is explicit about *why*: "Claude's installer was the only writer of any manifest before this phase" — i.e. this fallback was designed for Claude's own directory and the shared `.agents/skills/codegraph` directory (where D-17's symlink makes a pre-phase manifest genuinely ambiguous between "Claude" and "the shared package").

This phase (05-04/05-05) wires the *same* `recordSkillManifest`/`uninstallSkillPackage` functions, via `installDeclaredSkill`/`uninstallDeclaredSkill` (`capabilities.go:201-228`), onto three **harness-exclusive** directories that Claude never wrote to under any schema: Gemini's `.gemini/skills/codegraph`, Kiro's `.kiro/skills/codegraph`, and Antigravity's `~/.gemini/config/skills/codegraph`. For these directories there is no legitimate ambiguity — a pre-phase manifest could never exist there, let alone one written by Claude. The "assume Claude" fallback is simply wrong for this case, not merely imprecise.

Reproduced against the current tree:
1. Fresh `Install(Global)` on the `geminiTarget` writes `.gemini/skills/codegraph/.codegraph-manifest.json` with `Targets: [gemini]`.
2. Hand-corrupt that manifest file (simulating a partial write, disk error, or external edit — the exact class of event `writeManifest`'s own self-heal comment anticipates).
3. Re-run `Install(Global)`. `recordSkillManifest` reads the corrupted manifest, `readManifest` returns an error, `manifestRequesters` returns `[Claude]`, Gemini is appended → the manifest is rewritten with `Targets: [claude, gemini]`.
4. `Uninstall(Global)` on `geminiTarget` removes `gemini` from the set. `remaining = [claude]` is non-empty, so `uninstallSkillPackage` takes the "keep the package, just drop my exclusive keys" branch — SKILL.md and the manifest are **never removed**, and the directory survives forever with a manifest asserting Claude jointly owns it. Claude's own `Uninstall` never touches this path (it operates on `~/.claude/skills/codegraph`, a different directory), so there is no subsequent operation that can ever bring `Targets` back to empty. The only recovery is a manual `rm -rf`.

This directly contradicts the invariant this phase is built to guarantee: "the package is deleted only when `targets` becomes empty" (D-08) and "codegraph never leaves stray files behind" (AGENT-13's reversal contract). Verified with a throwaway test against the current tree:

```
--- PASS: TestREPRO_CorruptedHarnessManifestFalselyAttributesClaude
    Targets after self-heal reinstall: [claude gemini]
    CONFIRMED BUG: Claude falsely attributed as requester of .../.gemini/skills/codegraph
    CONFIRMED BUG: skill dir survived uninstall of its only real requester (gemini) due to phantom claude attribution
```

No test in `ownership_test.go`, `skillshared_test.go`, `gemini_test.go`, `kiro_test.go`, or `antigravity_test.go` plants a *corrupted* (present-but-undecodable) manifest at a harness-exclusive directory — every corrupted-manifest test (`TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude`, `TestConfiguredSkillLocations_IncludesLocationWithCorruptedManifest`, `TestConfiguredSkillLocations_RequiresClaudeInTargets/corrupt manifest is included`) exercises either the shared directory or Claude's own directory, where the "assume Claude" fallback is defensible. The ownership guard's "foreign-codegraph-dir" variant (`ownership_test.go`) plants a directory with **no manifest at all**, which is a different (and correctly-handled) code path from a manifest that exists but fails to parse.

**Fix:** Scope the "assume Claude" fallback to only the two directories where it is actually justified (Claude's own directory, and the shared directory reached through `installSkillPackage`'s D-17 symlink-aware path), rather than applying it unconditionally inside `manifestRequesters`. Concretely, thread the correct fallback through the call site instead of hard-coding it in the shared helper, e.g.:

```go
// manifestRequesters now takes the fallback owner explicit at each call
// site, instead of hard-coding Claude everywhere.
func manifestRequesters(m skillManifest, present bool, readErr error, unreadableFallback []TargetID) []TargetID {
	if readErr != nil {
		return unreadableFallback
	}
	if !present {
		return nil
	}
	if m.Targets == nil {
		return unreadableFallback
	}
	out := make([]TargetID, len(m.Targets))
	copy(out, m.Targets)
	return out
}
```

Callers for Claude's own directory and the shared directory pass `[]TargetID{Claude}` (preserving today's behavior exactly); `recordSkillManifest`/`uninstallSkillPackage`, when invoked for a harness-exclusive directory (Gemini/Kiro/Antigravity), should pass `nil` (or `[]TargetID{requester}`) instead — treating a corrupted manifest there the same way `writeManifest` already treats it everywhere else: self-healing to a clean, single-owner state rather than inventing a co-owner that can never legitimately relinquish ownership.

## Warnings

### WR-01: `describeDeclaredPaths` silently swallows `HookFiles` resolution errors, contradicting the documented "loud failure" contract for a future `HooksCodexJSON` target

**File:** `internal/agents/capabilities.go:262-266`

**Issue:** `errHookFilesUndeclared`'s doc comment (capabilities.go:50-56) and `HookFiles`'s doc comment (capabilities.go:151-156) both state the intent plainly: a future target that sets `Hooks: HooksCodexJSON` without also updating `HookFiles` "must fail loudly here (surfacing as a D-03 guard failure) rather than silently describing no hook files at all."

That promise is not actually kept. The only consumer of `HookFiles` is `describeDeclaredPaths`, and it discards the error unconditionally:

```go
if hookFiles, err := caps.HookFiles(loc); err == nil {
    for _, p := range hookFiles {
        add(p)
    }
}
```

If `caps.HookFiles(loc)` returns `errHookFilesUndeclared`, this branch is simply skipped — `DescribePaths()` returns an incomplete-but-successful path list, `result.Errors` is never touched, and nothing "fails loudly." The D-03 guard test (`TestCapabilitiesTableDrivesDerivations`'s `expectedDeclaredPaths` oracle, `capabilities_test.go:265-302`) wouldn't catch this either, since it special-cases only `HooksClaudeJSON` and would independently omit the same hook files the oracle never learned to derive for `HooksCodexJSON` — so oracle and implementation would agree by omission, and the test would stay green.

This is inert today (no registered target sets `HooksCodexJSON`), but it means the specific safety net this phase's own documentation describes for Phase 7's Codex-hooks work does not exist yet, and nothing in the current test suite would notice its absence.

**Fix:** Either propagate the error into `result.Errors` in `describeDeclaredPaths` (mirroring how `installDeclaredSkill`/`uninstallDeclaredSkill` already record resolution errors via `result.Errors` rather than dropping them), or — since `DescribePaths(loc) []string` has no error return in the `AgentTarget` interface — have `describeDeclaredPaths` `panic` on `errHookFilesUndeclared` specifically (a genuine programmer error: a target literal that declares a hooks mechanism this package cannot describe), so the failure is visible at test time rather than silently absent from a path list.

## Info

### IN-01: Dead defensive check — `caps.MCPConfig == nil` in `configStyleFields` can never execute

**File:** `internal/cli/printconfigstyle.go:41-43`

**Issue:** `configStyleFields` guards against `caps.MCPConfig == nil` before calling it, but every one of the 8 registered targets' `Capabilities()` literals sets `MCPConfig` unconditionally (it is documented as "Required for every Location in Scopes" in `capabilities.go:83-85`). Given the current registry, this branch is unreachable and untestable (no test exercises it, and none could without hand-constructing a non-conformant `AgentTarget`). Harmless, but worth noting as it can mask a real future regression (a target literal that forgets to set `MCPConfig`) behind a clean error message that no test will ever trip.

**Fix:** No action required; if kept intentionally as defense-in-depth against a future malformed literal, consider a one-line comment noting it is currently unreachable given D-02's "required for every Location in Scopes" contract, so a future reader doesn't waste time trying to hit it via the CLI.

### IN-02: `ActionKeptForeign` has no dedicated styled role and no test coverage of its rendering

**File:** `internal/cli/install.go:203-218` (`printAgentResults`)

**Issue:** The styled-output branch of `printAgentResults` maps `FileAction` to a `present.Role` via an explicit switch (`ActionUnchanged, ActionKept, ActionNotFound` → `RoleLabel`); `ActionKeptForeign` — the new action this phase introduces for D-14's foreign-directory protection — falls through to the `default: actionRole = present.RoleWarning` case. This may be intentional (a kept-foreign directory is arguably worth flagging to the user), but it is not stated anywhere, and no test (`install_test.go`, `printconfigstyle_test.go`) asserts on the styled rendering of a `kept (foreign)` line specifically — only `installStatus`'s plain-text roll-up behavior is tested (`TestInstallStatus_KeptForeignIsNotAChange`).

**Fix:** If `RoleWarning` is the intended treatment, add a one-line comment next to the switch stating so and add a coverage case (styled-output test asserting the `kept (foreign):` line renders with the warning role) so a future refactor of this switch doesn't silently change it.

---

_Reviewed: 2026-09-18T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
