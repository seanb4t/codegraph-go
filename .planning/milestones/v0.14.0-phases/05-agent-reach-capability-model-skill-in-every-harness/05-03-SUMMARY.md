---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 03
subsystem: agents
tags: [skill-package, manifest, symlink-safety, claude, upgrade]

# Dependency graph
requires:
  - phase: 05-02
    provides: "installSkillPackage/uninstallSkillPackage (the manifest-owned writer with a requester set), manifestRequesters, resolveSkillDir/sameSkillDir, removeSkillDirIfEmpty's Lstat guard"
provides:
  - "claudeSkillPolicy(loc): resolves refuseUnmanifested vs adoptUnmanifested by comparing claudeSkillDirPath(loc) against sharedSkillDirPath(loc) via sameSkillDir (D-17)"
  - "claudeTarget.Install/Uninstall moved onto writeSkillFile/recordSkillManifest/uninstallSkillPackage — Claude is now an ordinary requester of the shared writer, not a separate code path"
  - "ConfiguredSkillLocations(id) requires id among manifestRequesters(...) — manifest presence alone is no longer proof of a prior install for id"
  - "installSkillPackage (shared-writer side) notes when its directory coincides with Claude's own"
affects: [05-04, 05-05]

# Actuals (#2632)
actuals:
  tokens: 10409
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Both sides of a symlink-shared resource compare via the same primitive (sameSkillDir) before writing — Claude's Install/Uninstall AND the shared writer's installSkillPackage both call it, so neither can drift from the other's notion of 'same directory'"
    - "A shared-ownership discovery function (ConfiguredSkillLocations) must gate on requester-set membership, never on the mere existence of the shared artifact, once more than one party can write that artifact"

key-files:
  created:
    - internal/agents/claude_symlink_test.go
  modified:
    - internal/agents/claude.go
    - internal/agents/claude_skillpackage_test.go
    - internal/agents/manifest.go
    - internal/agents/skillshared.go

key-decisions:
  - "TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage passed even at RED, before any D-17 code existed. This is not a vacuous guard: 05-02's D-07 rule (manifestRequesters reads a nil-Targets/unreadable manifest as owned by [claude]) already produces the correct merged {claude, cursor} requester set on the READ side, purely as a byproduct of read-side self-healing. The three other symlink tests (uninstall, dangling-link reinstall, foreign-content) failed exactly as predicted and are what GREEN actually closes on the WRITE side. Recorded honestly rather than reshaping the test to force a RED it doesn't have."
  - "internal/daemon's TestConvergenceTwoSessions failed once during the full-module run (soak_test.go:238, 'session B did not converge'), then passed cleanly on immediate retry. This is the pre-existing, already-tracked flake in .planning/WINDOWS.md row 37 ('injected contention' chaos test, non-deterministic under cross-package load) — internal/daemon has no dependency on internal/agents, so this plan's changes cannot be its cause. Not fixed (scope boundary); not re-logged (already open in WINDOWS.md)."

requirements-completed: [AGENT-09, AGENT-13]

coverage:
  - id: D1
    description: "Claude's Install/Uninstall route their skill-directory writes through claudeSkillPolicy + writeSkillFile/recordSkillManifest/uninstallSkillPackage; in a symlinked layout (Claude's dir == the shared dir) there is one manifest recording every requester, Claude's uninstall follows the last-requester rule, foreign content is kept untouched through the symlink, and a dangling link is healed on reinstall"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestSymlinkedSkillDir_DanglingLinkReinstall"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestSymlinkedSkillDir_ForeignContentKeptForeign"
        status: pass
    human_judgment: false
  - id: D2
    description: "Claude's non-symlinked (v0.10.0) install/uninstall behaviour is byte-for-byte unchanged: install-local/uninstall-local plain goldens unmodified without regeneration, and every pre-existing Claude skill-package test passes with only one additive assertion (TestClaude_Install_WritesManifest's Targets check)"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/cli TestPlainGolden/install-local, TestPlainGolden/uninstall-local"
        status: pass
      - kind: unit
        ref: "internal/agents/claude_skillpackage_test.go (all pre-existing tests, unmodified except the additive Targets assertion)"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph upgrade's refresh step (ConfiguredSkillLocations) never re-installs Claude at a location where only another agent requested the shared package; legacy and corrupt manifests still count (WR-04/D-07 preserved)"
    requirement: "AGENT-13"
    verification:
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestConfiguredSkillLocations_RequiresClaudeInTargets"
        status: pass
      - kind: unit
        ref: "internal/agents/manifest_test.go#TestConfiguredSkillLocations_ProbesFixedPaths, #TestConfiguredSkillLocations_IncludesLocationWithCorruptedManifest, #TestConfiguredSkillLocations_ExcludesGenuinelyAbsentLocation (unmodified)"
        status: pass
      - kind: unit
        ref: "internal/cli TestUpgradeCommand_RefreshesConfiguredLocations and siblings"
        status: pass
    human_judgment: false
  - id: D4
    description: "The shared writer (installSkillPackage) surfaces a same-directory note when a non-Claude requester's directory coincides with Claude's, so the fact is visible from either side's install output"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/agents/claude_symlink_test.go#TestSharedSkillWriter_NotesSameDirAsClaude"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 3: Claude's Skill Package on the Manifest-Owned Writer (D-17) Summary

**Claude Code's Install/Uninstall now route through 05-02's manifest-owned shared writer with a `filepath.EvalSymlinks`-based same-directory policy, so a user's `~/.claude/skills/codegraph -> ~/.agents/skills/codegraph` symlink is treated as one package with one manifest instead of two writers racing on the same file.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-09-18T20:52:00Z
- **Completed:** 2026-09-18T21:47:00Z
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `claudeSkillPolicy(loc)` compares `claudeSkillDirPath(loc)` against `sharedSkillDirPath(loc)` via `sameSkillDir` (bounded symlink resolution, dangling links followed) BEFORE any write: same directory -> `refuseUnmanifested` (the shared dir's D-14 foreign-content rule governs); distinct directories -> `adoptUnmanifested` (Claude's untouched v0.10.0 behaviour, D-05). A comparison error falls back to the conservative refuse policy and is recorded, never silently swallowed.
- `claudeTarget.Install` writes SKILL.md via `writeSkillFile` and records ownership via `recordSkillManifest(..., Claude, ...)` instead of hand-building a `skillManifest` literal — Claude is now the one-requester case of the same shared-writer code path every other target will use (05-04/05-05), not a second implementation. A kept-foreign write correctly skips the manifest step while every other Claude artifact (MCP config, CLAUDE.md, script, SessionStart hooks) still writes.
- `claudeTarget.Uninstall` replaces its hand-rolled manifest-then-SKILL.md removal with `uninstallSkillPackage(..., Claude, [script, hooksFrag] keys, policy)` — in a symlinked layout this is D-08's last-requester rule: Claude's uninstall now drops only its own two exclusive manifest keys and leaves the shared SKILL.md/manifest intact for any other requester (T-05-11), and never unlinks the user's symlink (a dangling link survives, healed by the next install).
- `ConfiguredSkillLocations(id)` now requires `id` among `manifestRequesters(...)` rather than treating manifest presence alone as proof — closing T-05-13 (`codegraph upgrade`'s refresh step could otherwise silently install Claude at a location only Cursor/opencode had ever requested through a shared symlink). Legacy (no `targets` key) and corrupt manifests still count, per D-07/WR-04.
- The shared writer's own `installSkillPackage`, called for any non-Claude requester, now compares its directory against Claude's via `sameSkillDir` and appends one advisory note naming both paths when they coincide — the "both writers compare" half of D-17 that lives on the shared-writer side rather than Claude's.

## Task Commits

Each task followed RED-GREEN TDD discipline:

1. **Task 1 (RED): failing symlinked-skill-dir tests for Claude** — `759ce812` (test)
2. **Task 1 (GREEN): Claude's skill package on the manifest-owned writer (D-17)** — `3333015f` (fix)
3. **Task 2 (RED): failing upgrade-refresh gate and same-dir note tests** — `136054d8` (test)
4. **Task 2 (GREEN): upgrade refresh requires Claude in requesters; same-dir note** — `6c88da17` (fix)

**Plan metadata:** committed alongside this SUMMARY.

### RED evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestSymlinkedSkillDir_|TestClaude_Install_WritesManifest$' -v` against the `test(05-03):` commit `759ce812`:

```
claude_skillpackage_test.go:828: Targets = [], want [claude]
--- FAIL: TestClaude_Install_WritesManifest (0.00s)
--- PASS: TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage (0.00s)
    claude_symlink_test.go:146: read shared SKILL.md after claude uninstall: open .../TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester.../.agents/skills/codegraph/SKILL.md: no such file or directory
--- FAIL: TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester (0.01s)
    claude_symlink_test.go:275: precondition not met: claude skill dir should be a dangling link before reinstall
--- FAIL: TestSymlinkedSkillDir_DanglingLinkReinstall (0.00s)
    claude_symlink_test.go:354: result.Files does not include a "kept (foreign)" entry for .../TestSymlinkedSkillDir_ForeignContentKeptForeign.../.claude/skills/codegraph: [{...Action:created} {...Action:created} {...SKILL.md Action:updated} {...Action:created} {...Action:created} {...manifest.json Action:created}]
--- FAIL: TestSymlinkedSkillDir_ForeignContentKeptForeign (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.086s
```

`TestClaude_Install_WritesManifest` failed on the new additive assertion exactly as expected (`Targets = []`, since no target yet wrote a requester set). `TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester` failed because the OLD Uninstall unconditionally deleted the shared SKILL.md regardless of other requesters (T-05-11). `TestSymlinkedSkillDir_DanglingLinkReinstall` failed because the OLD writer's plain `os.MkdirAll` errors on a dangling symlink dirent. `TestSymlinkedSkillDir_ForeignContentKeptForeign` failed because the OLD writer had no foreign-content check at all and silently overwrote it (`Action:updated`, not `kept (foreign)`).

`TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage` passed even at RED — see `key-decisions` above for why this is not a vacuous guard.

### RED evidence (Task 2)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestConfiguredSkillLocations_|TestSharedSkillWriter_NotesSameDirAsClaude$' -v` against the `test(05-03):` commit `136054d8`:

```
--- FAIL: TestConfiguredSkillLocations_RequiresClaudeInTargets (0.00s)
    --- FAIL: TestConfiguredSkillLocations_RequiresClaudeInTargets/symlinked_shared_dir_installed_by_another_target_only (0.00s)
        claude_symlink_test.go:452: ConfiguredSkillLocations(Claude) contains global when only cursor requested the shared package: [global]
    --- FAIL: TestConfiguredSkillLocations_RequiresClaudeInTargets/manifest_at_claude's_path_naming_only_cursor_is_excluded (0.00s)
        claude_symlink_test.go:488: ConfiguredSkillLocations(Claude) includes global for a manifest naming only cursor: [global]
    --- PASS: TestConfiguredSkillLocations_RequiresClaudeInTargets/legacy_manifest_with_no_targets_key_is_included (0.00s)
    --- PASS: TestConfiguredSkillLocations_RequiresClaudeInTargets/corrupt_manifest_is_included (0.00s)
--- FAIL: TestSharedSkillWriter_NotesSameDirAsClaude (0.00s)
    --- FAIL: TestSharedSkillWriter_NotesSameDirAsClaude/symlinked_layout_notes_the_shared_directory (0.00s)
        claude_symlink_test.go:564: Notes = [], want exactly one entry
    --- PASS: TestSharedSkillWriter_NotesSameDirAsClaude/plain_(non-symlinked)_layout_has_no_notes (0.00s)
    --- PASS: TestSharedSkillWriter_NotesSameDirAsClaude/requester_claude_never_notes_itself (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.125s
```

The two subtests seeded with legacy/corrupt manifests passed at RED because `manifestRequesters`' pre-existing D-07 rule already reads them as `[claude]` — the two genuinely discriminating subtests (a manifest naming only `cursor`, and the shared-writer's own missing note) failed exactly as predicted.

## Files Created/Modified

- `internal/agents/claude.go` — `claudeSkillPolicy`; Install's SKILL.md write moved onto `writeSkillFile`/`recordSkillManifest`; Uninstall's manifest+SKILL.md removal moved onto `uninstallSkillPackage`; removed the now-unused `internal/version` import
- `internal/agents/claude_skillpackage_test.go` — one additive assertion (`TestClaude_Install_WritesManifest`'s `Targets == [claude]` check)
- `internal/agents/claude_symlink_test.go` (new) — `symlinkedClaudeLayout` fixture, `TestSymlinkedSkillDir_ClaudeAndSharedAreOnePackage`, `_ClaudeUninstallKeepsOtherRequester`, `_DanglingLinkReinstall`, `_ForeignContentKeptForeign`, `TestConfiguredSkillLocations_RequiresClaudeInTargets`, `TestSharedSkillWriter_NotesSameDirAsClaude`
- `internal/agents/manifest.go` — `ConfiguredSkillLocations` now gates on `containsTarget(manifestRequesters(...), id)`, doc comment rewritten for D-17
- `internal/agents/skillshared.go` — `installSkillPackage` appends a same-directory note for non-Claude requesters via `sameSkillDir`

## Decisions Made

See `key-decisions` in frontmatter — the honest RED-but-already-passing test 1 finding, and the internal/daemon flake.

## Deviations from Plan

None — plan executed exactly as written, including the Task 1/Task 2 boundary (Task 1: Claude's own Install/Uninstall; Task 2: the upgrade-refresh gate and the shared-writer's own note).

## Issues Encountered

- `internal/daemon`'s `TestConvergenceTwoSessions` failed once during the required full-module run, then passed on immediate retry — the pre-existing, already-open flake tracked in `.planning/WINDOWS.md` row 37 (chaos/soak test, non-deterministic under cross-package load). `internal/daemon` has zero dependency on `internal/agents`, confirming this is unrelated to this plan's changes. Not fixed (out of scope); not re-logged (already open).
- This session's shell resolves as zsh rather than bash (`declare -p`/`(eval)` in trace output), which silently disables bash-style unquoted-variable word splitting for multi-line command hashes gathered via `$(...)`. Encountered only in my own ad hoc verification helper (a `for c in $C` loop over multiple commit hashes) — worked around with `git log ... | while IFS= read -r c; do ... done`. Does not affect the plan's own `<verify><automated>` one-liners, which use `&&`-chained single commands with no such loop.

## User Setup Required

None — no external service configuration required.

## Mutation Log

`05-MUTATION-LOG.md` is unchanged by this plan. Its existing families (a1/a2)
verify the D-03 capability-table guard introduced in 05-01; this plan added
no new guard of that shape. This plan's own TDD RED commits (`759ce812`,
`136054d8`) already serve the equivalent "planted violation goes RED, byte-
clean before/after" proof for its four new behavioral tests: the RED commit
IS the pre-mutation (unimplemented) state, its transcript above IS the
observed failure, and the subsequent GREEN commit IS the clean fix — the
same evidentiary shape the mutation log captures, produced by the TDD cycle
itself rather than a separate planted-and-reverted mutation.

## Next Phase Readiness

- `05-04`/`05-05` can now wire Cursor/opencode/Antigravity/Gemini/Kiro onto `installSkillPackage(sharedSkillDirPath(loc), loc, <target>, refuseUnmanifested)` with confidence that Claude's own directory is already correctly reconciled against theirs through `sameSkillDir` on both sides (D-17's "both writers compare").
- No blockers or concerns carried forward.

## Self-Check: PASSED

- `internal/agents/claude_symlink_test.go` exists: FOUND
- Commit `759ce812` (test): FOUND in `git log --oneline --all`
- Commit `3333015f` (fix): FOUND in `git log --oneline --all`
- Commit `136054d8` (test): FOUND in `git log --oneline --all`
- Commit `6c88da17` (fix): FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ -count=1`: PASS (both packages ok)
- Plain goldens (`install-local`, `uninstall-local`) unchanged: confirmed via `git diff --quiet HEAD -- internal/cli/testdata/plain/`
- No `05-03` commit touched `internal/cli/testdata/plain/`: confirmed per-commit via `git show --name-only`
- Full module (excluding `internal/daemon`, 60 packages per WINDOWS #37): all `ok`
- `internal/daemon` alone: `ok` on retry (see Issues Encountered)
- `claude_skillpackage_test.go`'s only 05-03 edit is additive: confirmed via `git show -p 759ce812 -- internal/agents/claude_skillpackage_test.go`

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*
