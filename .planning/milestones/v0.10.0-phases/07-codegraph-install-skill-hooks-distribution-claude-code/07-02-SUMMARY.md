---
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
plan: 02
subsystem: agents
tags: [uninstall, json-merge, atomic-write, read-error-invariant, hooks, claude-code]

# Dependency graph
requires:
  - phase: 07-codegraph-install-skill-hooks-distribution-claude-code
    plan: 01
    provides: "writeHookEntry, readJSONFileStrict, claudeassets embedded package, and the claudeSkillFilePath/claudeHooksScriptPath/claudeSessionStartBlocks path helpers this plan's removal path reuses for identity"
provides:
  - "removeHookEntry — the array-scoped removal mirror of writeHookEntry, identifying ownership by command string never matcher value, with the same keep-clean cascade removeMcpEntry established"
  - "removeEmbeddedFile / removeSkillDirIfEmpty — D-08-compliant file/directory removal primitives, the latter never recursive"
  - "claudeTarget.Uninstall now removes exactly the three artifacts Phase 7's Install wrote, byte-invariant against unrelated sibling content"
  - "the {install, uninstall} x {read-error, malformed} matrix proven for claudeSettingsPath, plus AutoAllow's two permission functions migrated onto the same fail-loud reader"
affects: [07-03-manifest-version-observability, 07-04-upgrade-auto-refresh]

# Actuals (#2632)
actuals:
  tokens: 7938
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "removeHookEntry: array-scoped JSON removal keyed on command-string ownership inside a block's hooks[] sub-array, with partial-block survival (an unrelated hook entry sharing a codegraph-owned block keeps that block alive) and the same event-key -> hooks-key -> whole-file keep-clean cascade removeMcpEntry already established"
    - "removeSkillDirIfEmpty: os.Remove-then-ReadDir-on-failure rather than matching a platform-specific ENOTEMPTY errno — cross-platform without a build-tag split, and never recursive"
    - "One read-error posture per shared file: addClaudeAllowPermission/removeClaudeAllowPermission migrated from the permissive readJSONFile to the fail-loud readJSONFileStrict because they write to the same claudeSettingsPath the hooks step (writeHookEntry/removeHookEntry) already reads strictly"

key-files:
  created:
    - internal/agents/claude_readerror_test.go
  modified:
    - internal/agents/claude.go
    - internal/agents/shared.go
    - internal/agents/claude_skillpackage_test.go

key-decisions:
  - "removeSkillDirIfEmpty confirms non-empty-directory failure by re-reading the directory (os.ReadDir) rather than matching syscall.ENOTEMPTY, since that errno's exact shape is not portable to Windows — this also gracefully absorbs races (something else populating the directory between os.Remove attempting and failing)."
  - "Task 2's read-error/malformed matrix required zero production code changes — Plan 01's readJSONFileStrict fail-loud posture, already the sole reader for the hooks step, already satisfied all four cells. Task 2 is a pure characterization/regression-lock; only Task 3's AutoAllow migration changed behavior."
  - "Only addClaudeAllowPermission/removeClaudeAllowPermission move to readJSONFileStrict — readJSONFile itself, writeMcpEntry/removeMcpEntry, and every other agent target's config reads keep the permissive fallback, since only claudeSettingsPath is now shared between a fail-loud step (hooks) and a pre-existing step (AutoAllow permission)."

patterns-established:
  - "Ownership-by-command-string, never by matcher/key value, applies symmetrically to both the write direction (writeHookEntry, Plan 01) and the removal direction (removeHookEntry, this plan) — a single identity rule governs the whole merge/unmerge lifecycle for array-scoped hook blocks."

requirements-completed: [AGENT-02]

coverage:
  - id: D6
    description: "codegraph uninstall after codegraph install returns .claude/settings.json to bytes that jsonDeepEqual its pre-install content, including an unrelated SessionStart block sharing codegraph's own 'startup' matcher, an unrelated PreToolUse event, and an unrelated top-level key"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_SkillPackageRoundTripIsByteInvariant"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Uninstall_RemovesOnlyOwnBlockUnderSharedMatcher"
        status: pass
    human_judgment: false
  - id: D7
    description: "uninstall removes the SKILL.md and session-nudge.sh it installed and nothing else — a user-authored file placed in the same skill directory survives, and the directory is only removed when removing codegraph's own files leaves it empty"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Uninstall_PreservesUserFileInSkillDir"
        status: pass
    human_judgment: false
  - id: D8
    description: "uninstall against a location that was never installed reports ActionNotFound and returns no error for all three new artifacts, leaving every file byte-untouched"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Uninstall_NeverInstalledIsNotFound"
        status: pass
    human_judgment: false
  - id: D9
    description: "the empty-parent keep-clean cascade (SessionStart key -> hooks key -> whole file) removes exactly the empty husks a removal leaves behind, without disturbing an unrelated sibling event"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go#TestClaude_Uninstall_KeepsEmptyParentsClean"
        status: pass
    human_judgment: false
  - id: D10
    description: "all four cells of {install, uninstall} x {read-error, malformed} on .claude/settings.json surface the failure through WriteResult.Errors and leave the file's bytes exactly as found, with a non-vacuity guard proving the matrix discriminates"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_readerror_test.go#TestClaude_SettingsReadFailureNeverDestroysContent"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_readerror_test.go#TestClaude_SettingsReadFailureMatrixIsNotVacuous"
        status: pass
    human_judgment: false
  - id: D11
    description: "within a single Install/Uninstall call, every step that reads .claude/settings.json — the pre-existing AutoAllow permission step and the hooks step alike — shares one read-error posture, so one step cannot fail loud while another silently overwrites the same file"
    requirement: AGENT-02
    verification:
      - kind: unit
        ref: "internal/agents/claude_readerror_test.go#TestClaude_AutoAllowSharesStrictReadPosture"
        status: pass
    human_judgment: false

duration: ~11min
completed: 2026-08-13
status: complete
---

# Phase 7 Plan 2: Uninstall Byte-Invariance and the Read-Error Matrix Summary

**`codegraph uninstall` now removes exactly the three artifacts Phase 7's install wrote — byte-invariant against unrelated sibling content, never recursive on the skill directory — and both settings.json readers (hooks merge and AutoAllow permission) share one fail-loud posture so a single `install --auto-allow` can no longer protect and destroy the same file.**

## Performance

- **Duration:** ~11 min (first RED commit to final GREEN commit)
- **Started:** 2026-08-13T12:21:48-04:00
- **Completed:** 2026-08-13T12:28:15-04:00
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- New `removeHookEntry` in `internal/agents/shared.go` mirrors `writeHookEntry`'s command-string ownership rule for removal: a block mixing codegraph's own hook entry with an unrelated one in the same `hooks[]` sub-array survives with only codegraph's entry stripped, and the same event-key → hooks-key → whole-file keep-clean cascade `removeMcpEntry` established now applies to `hooks.SessionStart` too.
- New `removeEmbeddedFile` and `removeSkillDirIfEmpty` give uninstall D-08-compliant, never-recursive removal of SKILL.md, the hooks script, and (only once genuinely empty) the skill directory itself — proven by a test that plants a user file in the directory and requires it to both survive and, once removed, let the directory go.
- `claudeTarget.Uninstall` gained three new steps in the established `resolve-path-or-append-error, then recordFile` shape; `DescribePaths` now lists all five files codegraph touches with no duplicates.
- A table-driven matrix (`claude_readerror_test.go`) proves all four cells of `{install, uninstall} x {read-error, malformed}` on `.claude/settings.json` surface the failure and leave bytes untouched, plus a non-vacuity guard proving the matrix isn't passing because the operations never ran. This characterized already-correct Plan 01 behavior — no implementation change was needed for Task 2.
- `addClaudeAllowPermission`/`removeClaudeAllowPermission` migrated from the permissive `readJSONFile` to the fail-loud `readJSONFileStrict`, closing the one split-posture hazard that *did* exist: `install --auto-allow` against a malformed settings.json previously overwrote it with only the permission entry before the hooks step ever got a chance to protect it. `readJSONFile`'s doc comment now records which callers deliberately remain on the permissive fallback and why.
- 12 new tests plus the existing `internal/agents`/`internal/cli`/`internal/upgrade` suites all pass; `go vet ./...` is clean; no pre-existing test was deleted or weakened.

## Task Commits

Each task followed its own RED/GREEN split, except Task 2 (tests characterized already-correct behavior with no implementation change needed):

1. **Task 1: uninstall removes exactly what install wrote — RED** - `6a13a93` (test)
2. **Task 1: uninstall removes exactly what install wrote — GREEN** - `b42f791` (feat)
3. **Task 2: prove the {install, uninstall} x {read-error, malformed} matrix** - `2bbd9a7` (test)
4. **Task 3: AutoAllow split-posture hazard — RED** - `001c6ee` (test)
5. **Task 3: unify AutoAllow onto the strict reader — GREEN** - `8d501eb` (fix)

**Plan metadata:** commit pending (this SUMMARY.md, applied by the orchestrator per wave-merge protocol — worktree mode excludes STATE.md/ROADMAP.md writes from this plan's own commits)

## Files Created/Modified
- `internal/agents/claude_readerror_test.go` (new) - the read-error/malformed matrix (Task 2) plus the AutoAllow split-posture regression tests (Task 3)
- `internal/agents/shared.go` - new `removeHookEntry`, `removeEmbeddedFile`, `removeSkillDirIfEmpty`; `readJSONFile`'s doc comment extended to record the divergence Task 3 introduces
- `internal/agents/claude.go` - `Uninstall` gains three removal steps plus the empty-directory sweep; `DescribePaths` gains the skill file and script paths; `addClaudeAllowPermission`/`removeClaudeAllowPermission` migrated to `readJSONFileStrict`
- `internal/agents/claude_skillpackage_test.go` - 6 new tests: round-trip byte-invariance, shared-matcher ownership discrimination, the empty-parent cascade, user-file survival, never-installed `ActionNotFound`, and the five-distinct-paths `DescribePaths` invariant

## Decisions Made
- `removeSkillDirIfEmpty` confirms a non-empty-directory failure by re-reading the directory (`os.ReadDir`) after `os.Remove` fails, rather than matching a platform-specific `ENOTEMPTY` errno — portable across Windows without a build-tag split, and it also absorbs a race where something else populates the directory between the failed remove and the check.
- Task 2's matrix needed no production code change: Plan 01's `readJSONFileStrict` was already the sole reader for the hooks step, and that was already sufficient for all four cells. This plan's only *behavior* change is Task 3's AutoAllow migration, which closes a real, previously-reachable hazard (confirmed RED before the fix): `install --auto-allow` against malformed settings.json used to overwrite it.
- Only the two Claude-specific AutoAllow functions move to the strict reader — `readJSONFile` itself, the MCP-entry path, and every other agent target keep the permissive fallback their own doc comments deliberately chose, since only `claudeSettingsPath` is now shared between a fail-loud step and a pre-existing permissive one.

## Deviations from Plan
None — plan executed exactly as written. Task 2's "no implementation change needed" outcome was anticipated by the plan's own framing (Plan 01's `07-01-SUMMARY.md` explicitly noted `readJSONFileStrict` already existed for this purpose) and is not a deviation.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 03 (manifest/version observability) and Plan 04 (upgrade auto-refresh) both depend on `claudeassets`, the location-aware path helpers, and now the symmetric write/remove hook-merge primitives (`writeHookEntry`/`removeHookEntry`) established across Plans 01–02 — no new file-safety primitive should be needed for either.
- The full AGENT-02 surface (byte-invariant uninstall, never-recursive directory cleanup, the read-error/malformed matrix, and the unified settings.json read posture) is now proven; Plan 03's manifest work can build its own comparison logic without re-deriving any of this plan's removal or read-error handling.

## Self-Check: PASSED

- FOUND: internal/agents/claude_readerror_test.go
- FOUND: internal/agents/claude.go (removeHookEntry/removeEmbeddedFile/removeSkillDirIfEmpty call sites present)
- FOUND: internal/agents/shared.go (removeHookEntry/removeEmbeddedFile/removeSkillDirIfEmpty defined)
- FOUND commits: 6a13a93, b42f791, 2bbd9a7, 001c6ee, 8d501eb

---
*Phase: 07-codegraph-install-skill-hooks-distribution-claude-code*
*Completed: 2026-08-13*
