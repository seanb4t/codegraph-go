---
phase: 06-claude-code-pretooluse-nudge
plan: 04
subsystem: agents
tags: [claude-code, hooks, pretooluse, nudge, manifest, sticky-opt-in, ownership, d-10, d-11, d-13]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01 InstallOptions.PreToolNudge tri-state, rendered guard, claudePreToolUseBlocks; 06-03 cooldown-gated hook
provides:
  - manifest keys manifestKeyPreToolGuard / manifestKeyPreToolFrag (Files keys, no schema bump) and preToolNudgeRecorded
  - recordSkillManifestWithFallback dropKeys parameter (nil at every pre-existing call site)
  - Claude Install Keep/On/Off PreToolUse step (D-10) and Uninstall's always-attempted removal (D-11)
  - Capabilities.HookFiles names the PreToolUse guard for claude-json (D-13)
  - TestPreToolNudge_* lifecycle suite (10 tests); ownership table with the opt-in on and a planted same-matcher PreToolUse block
  - 06-MUTATION-LOG.md Family (d1)-(d4)
affects: [06-05, 06-06, 06-07, CODEX-05]

actuals:
  tokens: 12300
  tasks: 3
  commits: 5
plan_head_before: 743a178ba88d3f550f79e609a7979dcdc5f46080

tech-stack:
  added: []
  patterns:
    - "Sticky opt-in read from manifest Files-key presence, read once at the top of Install before any write"
    - "Explicit dropKeys on the single merged manifest write, so an opt-out forgets its record without a second write"

key-files:
  created:
    - internal/agents/claude_pretooluse_lifecycle_test.go
  modified:
    - internal/agents/claude.go
    - internal/agents/manifest.go
    - internal/agents/skillshared.go
    - internal/agents/types.go
    - internal/agents/capabilities.go
    - internal/agents/capabilities_test.go
    - internal/agents/claude_skillpackage_test.go
    - internal/agents/ownership_test.go
    - internal/cli/testdata/plain/uninstall-local.golden
    - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md

key-decisions:
  - "Phase 6 06-04: the PreToolUse opt-in is recorded as manifest Files keys hooks/pretooluse-nudge.sh and settings.json#hooks.PreToolUse (no schema bump); Keep refreshes only while either key is present and the manifest is readable"
  - "Phase 6 06-04: an Off install reports only artifacts actually removed and drops both keys only when neither removal errored"

patterns-established:
  - "Lifecycle tests compare install result action lists across modes (Keep vs Off) to prove an opt-out adds no report lines on a never-opted location"

requirements-completed: []

coverage:
  - id: D1
    description: "On records both manifest keys (hash of the rendered guard; hashOwnedHookBlocks of the PreToolUse blocks), only when both writes succeeded"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_OnRecordsManifestKeys"
        status: pass
  - id: D2
    description: "Keep refreshes a recorded opt-in for a moved binary; touches nothing when not recorded or when the manifest is unreadable"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_KeepRefreshesWhenRecorded, TestPreToolNudge_KeepNoopWhenNotRecorded, TestPreToolNudge_KeepWithUnreadableManifestTouchesNothing"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (d2)"
        status: pass
  - id: D3
    description: "Off removes the guard and own PreToolUse handlers and forgets the record; a never-opted Off adds no report lines"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_OffRemovesAndForgets, TestPreToolNudge_OffWhenNeverOptedAddsNoFiles"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (d4)"
        status: pass
  - id: D4
    description: "uninstall always attempts the guard and PreToolUse removal (not-found when never opted); Claude drops both keys from a shared manifest"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_UninstallAlwaysAttempts, TestPreToolNudge_ClaudeUninstallDropsKeysFromSharedManifest; internal/cli TestPlainGolden/uninstall-local"
        status: pass
  - id: D5
    description: "idempotency: On twice is byte-identical and all-unchanged; uninstall twice reports not-found with no error"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_ReinstallIsIdempotent"
        status: pass
  - id: D6
    description: "exact-identity ownership: a hand-edited own block duplicates; an unrelated same-matcher PreToolUse block survives install and uninstall"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_HandEditedOwnEntryDuplicates; internal/agents/ownership_test.go#TestOwnershipExactIdentity (32 leaves)"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (d1)"
        status: pass
  - id: D7
    description: "HookFiles for claude-json names the guard; DescribePaths has 7 paths; print-config-style goldens unchanged"
    verification:
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesTableDrivesDerivations, TestCapabilitiesMatchInstallWrites; claude_skillpackage_test.go#TestClaude_DescribePaths_IncludesManifest"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (d3)"
        status: pass

duration: 46min
completed: 2026-09-19
---

# Phase 6 Plan 04: Sticky PreToolUse Opt-in Lifecycle Summary

**The Claude PreToolUse nudge opt-in is now recorded in the skill manifest as two new Files keys. Plain installs and upgrades (Keep) refresh it only while it is recorded, an explicit Off removes it and forgets the record in one manifest write, and uninstall always attempts removal. The capability table and the 32-leaf exact-identity ownership table both cover the guard and a planted same-matcher PreToolUse block.**

## Performance

- **Duration:** about 46 min
- **Started:** 2026-09-19T09:35:47Z
- **Completed:** 2026-09-19T10:21:42Z
- **Tasks:** 3
- **Files modified:** 11 (1 created)

## Accomplishments

- D-10: `preToolNudgeRecorded(loc)` is read at the top of `Install`, before any write. Here is what each mode does:
  - **On** writes the guard and the blocks and records both keys (CR-01 have-flag rule).
  - **Keep** re-renders the guard only when the opt-in is recorded and the manifest is readable.
  - **Off** removes the guard and blocks and passes both keys as `dropKeys`.
- D-11: `Uninstall` always runs `removeEmbeddedFile(guard)` and `removeHookEntry(settingsPath, "PreToolUse", own)`, reporting `not-found` when the user never opted in. Both keys are added to Claude's exclusive keys, so a shared manifest keeps the other requester's package.
- D-13: `HookFiles(claude-json)` now returns `[settings.json, session-nudge.sh, pretooluse-nudge.sh]`. `--print-config-style` output and both of its goldens are unchanged.
- The ownership table runs every leaf with `PreToolNudge: PreToolNudgeOn`. The Claude leaves plant `{"matcher":"Bash",…}` under PreToolUse and prove that it survives deep-equal, that no own PreToolUse command remains, and that the guard is gone.

## Task Commits

1. **Task 1 RED:** `9ec1fa2a`, test(06-04): add failing sticky opt-in lifecycle tests
2. **Task 1 GREEN:** `9e8ba24f`, feat(06-04): sticky PreToolUse opt-in in the Claude manifest; uninstall always removes it
3. **Task 2 RED:** `ae6992c1`, test(06-04): expect the PreToolUse guard in the capability table and ownership table
4. **Task 2 GREEN:** `13071b7b`, feat(06-04): capability table names the PreToolUse guard (D-13)
5. **Task 3:** `a615ad15`, docs(06-04): record Family (d) ownership, stickiness and capability-table RED demonstrations

## TDD RED evidence

**Task 1** (`9ec1fa2a`, with the placeholders: key consts, `preToolNudgeRecorded` returning `(false, false)`, and `dropKeys` threaded through as nil):

```
--- FAIL: TestPreToolNudge_OnRecordsManifestKeys (0.00s)
--- FAIL: TestPreToolNudge_KeepRefreshesWhenRecorded (0.00s)
--- FAIL: TestPreToolNudge_KeepNoopWhenNotRecorded (0.00s)
--- PASS: TestPreToolNudge_KeepWithUnreadableManifestTouchesNothing (0.00s)
--- FAIL: TestPreToolNudge_OffRemovesAndForgets (0.00s)
--- PASS: TestPreToolNudge_OffWhenNeverOptedAddsNoFiles (0.00s)
--- FAIL: TestPreToolNudge_UninstallAlwaysAttempts (0.01s)
    --- FAIL: TestPreToolNudge_UninstallAlwaysAttempts/never-opted (0.00s)
    --- FAIL: TestPreToolNudge_UninstallAlwaysAttempts/opted (0.00s)
--- FAIL: TestPreToolNudge_ReinstallIsIdempotent (0.00s)
--- PASS: TestPreToolNudge_HandEditedOwnEntryDuplicates (0.00s)
--- FAIL: TestPreToolNudge_ClaudeUninstallDropsKeysFromSharedManifest (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.223s
```

Here is each failure message:
- `:114 manifest "hooks/pretooluse-nudge.sh" = "", want the hash of the rendered guard on disk …`
- `:142 Keep with a recorded opt-in did not re-render the guard for the moved binary (D-01b)`
- `:183 positive control: an On install did not record "hooks/pretooluse-nudge.sh"`
- `:228 Off left the guard …/.claude/hooks/pretooluse-nudge.sh`
- `:296 Uninstall guard FileResult = "", want "not-found" (D-11: always attempted)`
- `:309 Uninstall guard FileResult = "", want "removed"`
- `:355 first Uninstall guard = "", want "removed"`
- `:478 precondition: the shared manifest does not record "hooks/pretooluse-nudge.sh"`

Three tests pass against the placeholders:
- **KeepWithUnreadable:** the placeholder already returns the correct `(false, false)` for a corrupt manifest.
- **OffWhenNeverOpted:** it pins a property the pre-06-04 code already had.
- **HandEditedOwnEntryDuplicates:** ownership did not change in this plan.

Their guards are positive-controlled elsewhere: the Keep branch by (d2), the Off branch by (d4), and ownership by (d1).

**Task 2** (`ae6992c1`):

```
--- FAIL: TestCapabilitiesTableDrivesDerivations (0.01s)
    --- FAIL: TestCapabilitiesTableDrivesDerivations/claude/global (0.00s)
    --- FAIL: TestCapabilitiesTableDrivesDerivations/claude/local (0.00s)
--- FAIL: TestCapabilitiesMatchInstallWrites (0.02s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/global (0.00s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/local (0.00s)
--- FAIL: TestClaude_DescribePaths_IncludesManifest (0.00s)
    capabilities_test.go:423: Install wrote/kept ".claude/hooks/pretooluse-nudge.sh", which the table does not declare: …
    claude_skillpackage_test.go:953: DescribePaths(global) expected 7 paths, got 6: …
```

In the same RED run, `TestOwnershipExactIdentity` already passed all 32 leaves, because Task 1's implementation was in place. Family (d1) proves that the extended ownership guard can fail.

## PASS counts (GREEN)

- 10/10 named `TestPreToolNudge_*` lifecycle PASS lines
- 16 `TestCapabilitiesTableDrivesDerivations`, 13 `TestCapabilitiesMatchInstallWrites` and **32 `TestOwnershipExactIdentity`** leaf PASS lines; `TestClaude_DescribePaths_IncludesManifest` PASS
- `TestPlainGolden` ok, with the `uninstall-local` and `install-local` subtests PASS

## Golden diff

The golden was regenerated with `-update-plain-goldens`. Only `uninstall-local.golden` changed:

```diff
@@ -7,3 +7,5 @@ Claude Code: removed
   removed: .claude/skills/codegraph/SKILL.md
   removed: .claude/hooks/session-nudge.sh
   removed: .claude/settings.json
+  not-found: .claude/hooks/pretooluse-nudge.sh
+  not-found: .claude/settings.json
```

Both print-config-style goldens are byte-identical from the phase base (`603efc95`) to HEAD.

## Family (d) outcome (06-MUTATION-LOG.md)

| Family | Mutation | RED line | Revert |
|---|---|---|---|
| (d1) D-11/242ec0a | `isOwned` claims any `matcher == "Bash"` block (shared.go) | `--- FAIL: TestOwnershipExactIdentity/claude/{global,local}/{clean,foreign-codegraph-dir}` (4 leaves; the other 28 PASS), exit=1 | byte-clean; green control ok |
| (d2) D-10 | Keep-and-recorded routed to the remove branch (claude.go) | `--- FAIL: TestPreToolNudge_KeepRefreshesWhenRecorded`, exit=1 | byte-clean; green control ok |
| (d3) D-13 | `HookFiles` returns `[settingsPath, scriptPath]` (capabilities.go) | `--- FAIL: TestCapabilitiesMatchInstallWrites/claude/{global,local}`, exit=1 | byte-clean; green control ok |
| (d4) D-10 | Off path sets `dropKeys = nil` (claude.go) | `--- FAIL: TestPreToolNudge_OffRemovesAndForgets`, exit=1 | byte-clean; green control ok |

After Task 3, `git diff --quiet HEAD -- internal/agents/shared.go internal/agents/claude.go internal/agents/capabilities.go` passes.

## Verification

- `go build ./...`, `go vet ./...` and `gofmt -l internal cmd` are clean.
- Every package except `internal/daemon` passes: 53 `ok`, 0 FAIL. `internal/daemon` also passes when run alone (64.3 s).
- No `internal/cli` flake occurred in any run.
- `golangci-lint run ./internal/agents/` reports 2 `unused` findings (`claudeSkillFilePath` and `ownership242ec0aSHA`). The same 2 appear at the pre-plan commit `743a178b`, so they are pre-existing and out of scope.

## Decisions Made

- An Off install reports a FileResult only when the action is `removed`, and errors still go to `result.Errors`. The keys are dropped only if neither removal errored.
- The have-flag rule also covers the hash: if `hashOwnedHookBlocks` fails for the PreToolUse blocks, the error is recorded and the keys are not written.

## Deviations from Plan

**1. [Rule 1 - Test-spec bug] Hand-edit test edits every handler of codegraph's Bash block, not just the first**
- **Found during:** Task 1 (writing `TestPreToolNudge_HandEditedOwnEntryDuplicates`)
- **Issue:** `writeHookEntry`'s `isOwned` treats a block as owned when ANY of its handlers carries an own command. codegraph's Bash block has three handlers (`grep`/`rg`/`find` `if` rules). If only the first handler is edited, the block is still owned and gets rewritten, which gives 5 blocks, not the 6 the plan expects.
- **Fix:** The test re-points all three handlers of that block (` --edited`), which is the "hand-edited own entry" shape that duplicates. Ownership code is unchanged (the 242ec0a rule is untouched).
- **Note for the maintainer:** a user who edits only ONE handler of a multi-handler own block loses that edit on the next install, because the whole block is rewritten. This is existing block-level exact-identity behaviour, and before Phase 6 it could not arise because SessionStart blocks have one handler. It is reported here, not changed.

**2. [Rule 2 - Doc accuracy] `internal/agents/types.go` doc comment updated**
- `InstallOptions.PreToolNudge`'s comment said Keep and Off "touch nothing yet". It now describes the D-10 behaviour. This file is not in `files_modified`, and the change is doc-only (commit `9e8ba24f`).

**3. [Plan acceptance-criterion defect] Task 2's "print-config-style goldens byte-identical across the phase" check**
- `git log --grep='^test\(06-01\): ' | tail -1` picks up a `test(06-01)` commit from an earlier milestone (2026-07-12, `7095f81a`). The goldens legitimately changed after that commit, so the literal command exits 1.
- The property the criterion intends does hold. The goldens are unchanged from this phase's base `603efc95` (06-01's `plan_head_before`) and from both of this phase's `test(06-01)` commits (`5905039e`, `e8a91d61`) to HEAD. No code change was needed.

## Accepted limitations (flagged, not fixed)

- When Claude's skill directory is foreign (kept, D-14), the manifest step is skipped, so an opt-in there is written but not recorded. Keep then neither refreshes nor removes it, and uninstall still removes it. This is the plan's own accepted limitation.
- With Keep and an unreadable manifest, nothing is refreshed or removed, as specified. However, the same Install's manifest step self-heals the corrupt manifest (existing `writeManifest` behaviour) without the PreToolUse keys, because they could not be read. From then on the opt-in is unrecorded: later Keep installs leave the hook in place but stop refreshing it, and uninstall still removes it. Re-running with `--pretool-nudge` re-records it.
- Concurrent installs are last-writer-wins per file (NUDGE-06 backstop truth). Nothing new coordinates them.

## Known Stubs

None.

## Threat Flags

None. No new surface beyond the plan's threat model (T-06-19..22).

## Self-Check: PASSED
