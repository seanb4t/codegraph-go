---
phase: 06-claude-code-pretooluse-nudge
plan: 05
subsystem: cli
tags: [claude-code, hooks, pretooluse, nudge, dogfood, cobra-changed, tri-state, upgrade, d-09, d-10, d-12, d-13]
status: complete

requires:
  - phase: 06-claude-code-pretooluse-nudge
    provides: 06-01 PreToolUse fragment + InstallOptions.PreToolNudge; 06-04 sticky manifest record, Keep/On/Off lifecycle, ownership table
provides:
  - dogfooded hooks.PreToolUse in .claude/settings.json, deep-equal to the fragment (D-13)
  - fragment reshaped to six single-handler PreToolUse blocks (three Bash blocks: grep, rg, find)
  - hookEventBlock helper; TestHookRegistrationMatchesFragmentAndScript/{SessionStart,PreToolUse}; TestPreToolUseRegistrationShape
  - install --pretool-nudge tri-state via cobra Changed, and the D-09 stderr note
  - upgrade refresh passing PreToolNudgeKeep explicitly
  - TestInstall_PreToolNudge_* (5) and TestRefreshInstalledSkills_CarriesPreToolNudge
  - 06-MUTATION-LOG.md Family (e1)-(e4)
affects: [06-06, 06-07]

actuals:
  tokens: 11964
  tasks: 3
  commits: 5
plan_head_before: 6935aa3efa273b3c64495b38e997e86f8bf0b625

tech-stack:
  added: []
  patterns:
    - "Tri-state bool flag through cobra Changed: not given = Keep, given true = On, given false = Off"
    - "One handler per own hook block, so the ownership unit (the block) equals the handler"

key-files:
  created: []
  modified:
    - .claude/hooks/hooks.json
    - .claude/settings.json
    - internal/agents/hookpackage_test.go
    - internal/agents/claude_pretooluse_lifecycle_test.go
    - internal/agents/claude_pretooluse_test.go
    - internal/cli/install.go
    - internal/cli/upgrade.go
    - internal/cli/install_test.go
    - internal/cli/upgrade_test.go
    - docs/CLI-REFERENCE.md
    - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md

key-decisions:
  - "Phase 6 06-05: the PreToolUse fragment registers one handler per block (three Bash blocks for grep/rg/find), so a hand-edit of any single own handler duplicates rather than being overwritten via its siblings; ownership code unchanged"
  - "Phase 6 06-05: --pretool-nudge is read through cobra Changed (not given = Keep); the D-09 note goes to stderr, plain, once, before the per-agent report"

patterns-established:
  - "Dogfooded hook registrations are pinned twice: deep-equality with the fragment and an explicit byte-shape test over both files"

requirements-completed: []

coverage:
  - id: E1
    description: "settings.json registers hooks.PreToolUse deep-equal to the fragment; every command resolves to an executable guard"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestHookRegistrationMatchesFragmentAndScript/PreToolUse"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (e3)"
        status: pass
  - id: E2
    description: "registration shape: six single-handler blocks, Bash if rules grep/rg/find, timeout 5, no statusMessage, no other keys, in both files"
    requirement: NUDGE-03
    verification:
      - kind: unit
        ref: "internal/agents/hookpackage_test.go#TestPreToolUseRegistrationShape/{settings.json,hooks.json}"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (e4)"
        status: pass
  - id: E3
    description: "a hand-edit of ONE own handler (Bash(rg *)) survives reinstall byte-identical; 7 blocks (edited + 6 own)"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/agents/claude_pretooluse_lifecycle_test.go#TestPreToolNudge_HandEditedOwnEntryDuplicates"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (e4)"
        status: pass
  - id: E4
    description: "--pretool-nudge tri-state via Changed; plain install keeps and refreshes; =false removes and stays removed"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_PreToolNudge_{OptInRegisters,StickyAcrossPlainInstall,ExplicitFalseRemoves}"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (e2)"
        status: pass
  - id: E5
    description: "D-09 note on stderr once when the flag is given (either value) and Claude is not selected; none otherwise; exit 0"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_PreToolNudge_{NoteWhenClaudeNotSelected,NoNoteWhenClaudeSelected}"
        status: pass
  - id: E6
    description: "upgrade refresh carries the opt-in (guard re-rendered for the new binary) and adds it nowhere else"
    requirement: NUDGE-06
    verification:
      - kind: unit
        ref: "internal/cli/upgrade_test.go#TestRefreshInstalledSkills_CarriesPreToolNudge/{opted_in_refreshed,never_opted_not_added}"
        status: pass
      - kind: mutation
        ref: "06-MUTATION-LOG.md#Family (e1)"
        status: pass

duration: 11min
completed: 2026-09-19
---

# Phase 6 Plan 05: Dogfooded Registration and Sticky CLI Opt-in Summary

**This repository now registers the same PreToolUse nudge it ships, as six single-handler blocks pinned in both `.claude/settings.json` and the fragment. `--pretool-nudge` is a cobra-`Changed` tri-state with a D-09 stderr note, and `upgrade` passes `PreToolNudgeKeep` explicitly, so the opt-in survives plain installs and upgrades and is removed only by `--pretool-nudge=false` or uninstall.**

## Performance

- **Duration:** about 11 min
- **Started:** 2026-09-19T10:26:25Z
- **Completed:** 2026-09-19T10:37Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- The fragment's single three-handler Bash block became three single-handler Bash blocks (grep, rg, find). Grep, Glob, Read and SessionStart are byte-identical. `claudePreToolUseBlocks` does not depend on the block count, so it needed no change. `isOwned`, `writeHookEntry` and `removeHookEntry` are untouched.
- `.claude/settings.json` now carries `hooks.PreToolUse` deep-equal to the fragment. The two files are now byte-identical (same blob `f32cbb2a`). SessionStart is unchanged: the jq `-c` output at `da2b0c2d^` and at HEAD compares equal with `cmp`.
- `install`: not given = Keep, `--pretool-nudge` = On, `--pretool-nudge=false` = Off. When the flag is given and no resolved target is Claude, install writes the note once to `cmd.ErrOrStderr()` before `printAgentResults`.
- `upgrade`: `refreshInstalledSkills` passes `PreToolNudge: agents.PreToolNudgeKeep`. A doc comment contrasts this with AutoAllow's non-sticky `false`.
- `docs/CLI-REFERENCE.md` was regenerated through `task docs:cli`. Only the flag's usage line changed, and `task docs:cli:drift` exits 0.

## Task Commits

1. **Task 1 RED:** `da2b0c2d`, test(06-05): expect the dogfooded PreToolUse registration
2. **Task 1 GREEN:** `98f014d5`, feat(06-05): dogfood the PreToolUse nudge registration in this repository (D-13)
3. **Task 2 RED:** `895cebd2`, test(06-05): add failing --pretool-nudge stickiness, note and upgrade tests
4. **Task 2 GREEN:** `88b1563b`, feat(06-05): sticky --pretool-nudge tri-state, scope note, and upgrade carries the opt-in
5. **Task 3:** `4de63b58`, docs(06-05): record Family (e) upgrade, stickiness and dogfood-registration RED demonstrations

## TDD RED evidence

**Task 1** (`da2b0c2d`, `go test ./internal/agents/ -count=1 -v`):

```
--- FAIL: TestPreToolNudge_HandEditedOwnEntryDuplicates (0.00s)
--- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks (0.00s)
    --- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks/local (0.00s)
    --- FAIL: TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks/global (0.00s)
--- FAIL: TestHookRegistrationMatchesFragmentAndScript (0.00s)
    --- PASS: TestHookRegistrationMatchesFragmentAndScript/SessionStart (0.00s)
    --- FAIL: TestHookRegistrationMatchesFragmentAndScript/PreToolUse (0.00s)
--- FAIL: TestPreToolUseRegistrationShape (0.00s)
    --- FAIL: TestPreToolUseRegistrationShape/settings.json (0.00s)
    --- FAIL: TestPreToolUseRegistrationShape/hooks.json (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/agents	2.642s
```

Messages:
- `claude_pretooluse_lifecycle_test.go:385: after the first On install PreToolUse has 4 blocks, want 6 owned`
- `claude_pretooluse_test.go:140: hooks.PreToolUse has 4 blocks, want 6`
- `hookpackage_test.go:381: hookEventBlock: ../../.claude/settings.json has no hooks.PreToolUse`
- `hookpackage_test.go:457: ../../.claude/hooks/hooks.json: hooks.PreToolUse has 4 blocks, want 6`

**Task 2** (`895cebd2`, `go test ./internal/cli/ -count=1 -run 'TestInstall_PreToolNudge_|TestRefreshInstalledSkills_' -v`):

```
--- PASS: TestInstall_PreToolNudge_OptInRegisters (0.01s)
--- PASS: TestInstall_PreToolNudge_StickyAcrossPlainInstall (0.00s)
--- FAIL: TestInstall_PreToolNudge_ExplicitFalseRemoves (0.00s)
--- FAIL: TestInstall_PreToolNudge_NoteWhenClaudeNotSelected (0.00s)
    --- FAIL: TestInstall_PreToolNudge_NoteWhenClaudeNotSelected/given_true (0.00s)
    --- FAIL: TestInstall_PreToolNudge_NoteWhenClaudeNotSelected/given_false (0.00s)
--- PASS: TestInstall_PreToolNudge_NoNoteWhenClaudeSelected (0.01s)
--- PASS: TestRefreshInstalledSkills_CarriesPreToolNudge (0.01s)
    --- PASS: TestRefreshInstalledSkills_CarriesPreToolNudge/opted_in_refreshed (0.00s)
    --- PASS: TestRefreshInstalledSkills_CarriesPreToolNudge/never_opted_not_added (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.503s
```

Messages:
- `install_test.go:799: after --pretool-nudge=false: guard .claude/hooks/pretooluse-nudge.sh still present (stat err <nil>)`
- `install_test.go:824: stderr carries the note 0 times, want 1`

Four of the new Task 2 tests pass against the pre-plan code. `InstallOptions.PreToolNudge`'s zero value is already `PreToolNudgeKeep`, so the old `if pretoolNudge { On }` mapping and upgrade's field-less options already behaved as Keep for a not-given flag. The plan expected these tests to fail for that reason, but they do not. Their guards are positive-controlled by mutation instead:
- Family (e1) turns `opted_in_refreshed` RED.
- Family (e2) turns `StickyAcrossPlainInstall` RED.

`OptInRegisters` and `NoNoteWhenClaudeSelected` are positive controls for On and for the note's absence.

## PASS counts (GREEN)

- Task 1 verify: both registration subtests PASS; 2/2 `TestPreToolUseRegistrationShape` subtests; 4/4 pre-existing hook tests plus the rewritten hand-edit test; **32/32 `TestOwnershipExactIdentity` leaves**. `jq` confirms six single-handler PreToolUse blocks, and that PreToolUse and SessionStart are equal across the two files.
- Task 2 verify: 5/5 `TestInstall_PreToolNudge_*`; 2/2 `CarriesPreToolNudge` subtests; `TestRefreshInstalledSkills_OnlyPreviouslyConfiguredLocations`, `TestEveryRegisteredFlagIsAccountedFor`, `TestPlainGolden` and `TestInstall_Idempotent_RerunReportsUnchanged` PASS; `task docs:cli:drift` exit=0.
- Task 3 verify: the (e1) re-plant goes RED and reverts clean; 4 Family (e) sections; `internal/agents` and `internal/cli` both `ok`.

## Reference diff

```diff
-      --pretool-nudge        Claude Code only: also register a PreToolUse hook that points Claude at codegraph_explore when it searches
+      --pretool-nudge        Claude Code only: register a PreToolUse hook that points Claude at codegraph_explore when it searches; remembered across install and upgrade until --pretool-nudge=false or uninstall
```

No plain golden changed.

## Family (e) outcome (06-MUTATION-LOG.md)

| Family | Mutation | RED line | Revert |
|---|---|---|---|
| (e1) D-10 | upgrade.go `PreToolNudgeKeep` → `PreToolNudgeOff` | `--- FAIL: TestRefreshInstalledSkills_CarriesPreToolNudge/opted_in_refreshed`, exit=1 | byte-clean; green control ok |
| (e2) D-10 | install.go `nudge := agents.PreToolNudgeKeep` → `PreToolNudgeOff` (not-given read as Off) | `--- FAIL: TestInstall_PreToolNudge_StickyAcrossPlainInstall`, exit=1 | byte-clean; green control ok |
| (e3) D-13 | settings.json only: `Bash(rg *)` → `Bash(rg*)` | `--- FAIL: TestHookRegistrationMatchesFragmentAndScript/PreToolUse` (SessionStart still PASS), exit=1 | byte-clean; green control ok |
| (e4) NUDGE-06 | hooks.json only: pre-plan merged three-handler Bash block restored | `--- FAIL: TestPreToolNudge_HandEditedOwnEntryDuplicates` and `--- FAIL: TestPreToolUseRegistrationShape/hooks.json`, exit=1 | byte-clean; green control ok |

After Task 3, `git diff --quiet HEAD -- internal/cli/upgrade.go internal/cli/install.go .claude/settings.json` passes, and so does the same check for `.claude/hooks/hooks.json`.

## Verification

- `go build ./...`, `go vet ./...` and `gofmt -l internal cmd` are clean.
- All packages except `internal/daemon` pass: 53 `ok`, 0 FAIL. `internal/daemon` also passes when run alone (64.4 s).
- No `internal/cli` flake occurred in any run.
- `golangci-lint run ./internal/agents/ ./internal/cli/` reports 3 findings, none from this plan:
  - 2 `unused` (`claudeSkillFilePath`, `ownership242ec0aSHA`), already noted as pre-existing in 06-04.
  - 1 staticcheck `QF1011` in `internal/cli/serve.go:267`, a file this plan did not touch.

## Decisions Made

- The D-09 note uses `slices.ContainsFunc(targets, t.ID() == agents.Claude)` over the resolved targets, so `--target auto` with no Claude detected also gets the note.
- The note subtests are named `given_true` and `given_false` rather than after the flag text, so `--- PASS:` greps over `--pretool-nudge` stay unambiguous.

## Deviations from Plan

**1. [Rule 3 - Blocking] `assertPreToolUseBlocks` in `internal/agents/claude_pretooluse_test.go` updated to six single-handler blocks**
- **Found during:** Task 1
- **Issue:** 06-01's helper hard-coded the four-block shape (one Bash block with three handlers). The fragment split would have turned `TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks` RED at GREEN. That file is not in `files_modified`.
- **Fix:** The helper was changed in the RED commit to expect the same six-block shape as the new shape test, so it also shows up as RED evidence.
- **Commit:** `da2b0c2d`

**2. [Plan text] The rewritten hand-edit test no longer plants an unrelated same-matcher Bash block**
- The plan requires 7 blocks after reinstall (6 own + the edited one). The 06-04 version also planted an unrelated `Bash` block, which would make the count 8. The plan's count was followed.
- Same-matcher survival stays covered by `TestOwnershipExactIdentity`, whose Claude leaves plant `{"matcher":"Bash",…}` under PreToolUse (32/32 PASS).
- The rewritten test also checks that each of the 6 own blocks is present and deep-equal to `claudePreToolUseBlocks`, and that the edited block appears exactly once.

**3. [Plan acceptance-criterion defect] Task 1's SessionStart-unchanged check resolves the wrong base**
- `git log --grep='^test\(06-05\): ' | tail -1` finds `0328f83f`, a `test(06-05)` commit from an earlier milestone. 06-04 hit the same problem with `test(06-01)`.
- The check was run against this plan's own RED commit (`da2b0c2d^`), and SessionStart compares equal with `cmp`.

**4. [Honest RED scope] (e4)'s RED fires on the block-count precondition**
- The hand-edit test fails at `:385` (4 own blocks, want 6), which is before it edits the `rg` handler. The overwrite through unedited siblings is not re-exhibited in this run; 06-04 recorded it (Deviation 1). The log entry says this plainly.

## Unexpected

- Four of the six Task 2 tests pass at RED because of the Keep zero value (see TDD RED evidence). (e1) and (e2) supply their positive control.
- `.claude/settings.json` and `.claude/hooks/hooks.json` are now byte-identical files. If the dogfood settings ever gain a non-hook key (for example `permissions`), they will diverge, which is expected; the tests compare only `hooks.*`.

## Known Stubs

None.

## Threat Flags

None. The only new surface is the one T-06-25 accepts: the dogfood hook now runs in Claude Code sessions opened in this repository.

## Self-Check: PASSED
