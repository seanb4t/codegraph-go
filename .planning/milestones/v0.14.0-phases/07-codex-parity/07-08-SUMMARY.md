---
phase: 07-codex-parity
plan: 08
subsystem: codex
tags: [codex, hooks, pretooluse, nudge, cli, tdd, mutation-testing]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "07-07's Codex PreToolUse runtime (hooks.json fragment, guard templates, --harness codex envelope) — this plan adds the sticky Keep/On/Off lifecycle, the explicit hooks-off skip, the /hooks trust Note, and the widened CLI surface on top of that runtime, unchanged"
provides:
  - "internal/agents/toml.go: tomlBoolSetting (plus tomlKeyValue/tomlBoolLiteral/tomlInlineTableBoolValue/splitTOMLInlineTableEntries) — reads a TOML boolean across the plain in-table, root-dotted, and single-line inline-table forms"
  - "internal/agents/codex_pretooluse.go: codexHooksExplicitlyDisabled/codexConfigHooksSetting (D-18 skip), codexHookTrustNote/codexHooksDisabledNote (D-19/D-18 Notes), codexHooksFileWasWritten (Keep's trust-note gate)"
  - "internal/agents/codex.go: codexTarget.Install's PreToolNudge dispatcher now implements On (hooks-disabled-aware), Keep (hasOwnHookBlock-evidenced refresh, D-23), and Off (uninstall)"
  - "internal/cli/install.go, uninstall.go: --pretool-nudge, its note, help text and Example widened to Claude Code AND Codex CLI; docs/CLI-REFERENCE.md regenerated"
  - "internal/agents/ownership_test.go: TestOwnershipExactIdentity's codex leaves now plant and verify a foreign ^Bash$ hooks.json group (D-23/242ec0a)"
  - "Family (g1)-(g4) in 07-MUTATION-LOG.md: four positive-controlled guards proving the lifecycle, hooks-off, note, and ownership guards can each independently fail"
affects: [07-09, 07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 17461
  tasks: 3
  commits: 6
plan_head_before: 0f3d803f30c1dea87797e50fb1676558587b0eb8

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Codex's PreToolUse stickiness evidence is its own exact-identity hooks.json group, read directly via hasOwnHookBlock — never a shared skill manifest (D-23). Unlike Claude's preToolNudgeEvidenced (which widens a manifest-only check with a settings.json fallback), Codex has no manifest concept for this opt-in at all, so the dispatcher probes hooks.json unconditionally."
    - "tomlBoolSetting reuses the 07-01 TOML line-scanner (splitTOMLLines, tomlLineState, isTOMLHeaderLine, tomlNormalizedHeaderPath) rather than a second parser, extending it with a value-returning key/value split (tomlKeyValue) instead of touching the existing table-range scanner."

key-files:
  created: []
  modified:
    - internal/agents/toml.go
    - internal/agents/toml_test.go
    - internal/agents/codex_pretooluse.go
    - internal/agents/codex_pretooluse_test.go
    - internal/agents/codex.go
    - internal/agents/ownership_test.go
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - internal/cli/install_test.go
    - docs/CLI-REFERENCE.md
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "codexHookTrustNote's wording and codexHooksDisabledNote's wording were left to Claude's Discretion per 07-CONTEXT.md — settled on \"Codex skips a new or changed hook until you trust it — open /hooks in Codex and trust the codegraph PreToolUse hook\" (never --dangerously-bypass-hook-trust) and \"Codex hooks are explicitly disabled by <file> (<key>) — the codegraph PreToolUse nudge was not installed\" respectively."
  - "codexConfigHooksSetting's source string format (\"<path> ([features] <key>)\") was Claude's Discretion; chosen to name both the governing file and the exact table.key checked, matching D-18's \"Note names the file and key\" truth."
  - "TestCodexPreToolNudge_SkippedWhenHooksDisabled's per-case Notes assertion counts only Notes containing \"hooks are explicitly disabled\", not len(Notes) — a local install also carries codexTrustNote's unrelated D-10 trust-prompt Note, discovered while chasing the GREEN run (documented as a deviation below)."
  - "Family (g) mutations reuse the plan's own pinned code sites (the codexHooksExplicitlyDisabled signature line for g2) so the mutation-log transcript and the plan's own <verify> gate — which re-applies g2 live — describe the identical mutation."

requirements-completed: []

coverage:
  - id: D1
    description: "--pretool-nudge On installs the Codex guard and hooks.json group and prints exactly one Note naming /hooks, at both local and global scope"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_OnWritesAndNotesTrust"
        status: pass
    human_judgment: false
  - id: D2
    description: "Keep refreshes the guard for a moved binary only when codegraph's own hooks.json group is already present (D-23), leaving the definition and hooks.json bytes unchanged when nothing moved, and printing no trust Note in that case"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_KeepRefreshesWhenOwnGroupPresent"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_KeepNoopWhenNotOptedIn"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing"
        status: pass
    human_judgment: false
  - id: D3
    description: "Off removes the guard and codegraph's own hooks.json group, and a later Keep never brings them back — the opt-in stays explicit (D-18)"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_OffRemovesAndForgets"
        status: pass
    human_judgment: false
  - id: D4
    description: "An On install is skipped — writing neither the guard nor hooks.json, and printing a Note naming the file and key — whenever the governing config.toml explicitly disables Codex hooks via any of tomlBoolSetting's three recognized forms, at either scope, under either key name; a local true overrides a global false, and an unqualified true still writes"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_SkippedWhenHooksDisabled (7 subtests)"
        status: pass
      - kind: unit
        ref: "internal/agents/toml_test.go#TestTOMLBoolSetting (10 subtests)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Reinstalling On is idempotent (every file reports unchanged), and hand-editing the installed group's command makes On append a fresh owned group beside the untouched hand-edited one rather than overwriting it (242ec0a)"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_ReinstallIsIdempotent"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexPreToolNudge_HandEditedOwnGroupDuplicates"
        status: pass
    human_judgment: false
  - id: D6
    description: "No Note produced by a Codex install — across On/Keep/Off, both scopes, and the hooks-disabled skip — ever advises bypassing Codex's hook trust review; install --help and uninstall --help likewise carry no such advice, and install --help names both Codex and /hooks"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/codex_pretooluse_test.go#TestCodexNotesNeverAdviseTrustBypass"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstallHelpNeverAdvisesTrustBypass"
        status: pass
    human_judgment: false
  - id: D7
    description: "--pretool-nudge's D-09 note fires exactly once, on stderr, only when neither Claude nor Codex is among the selected targets; selecting Codex alone suppresses it and still writes Codex's guard and hooks.json"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_PreToolNudge_NoNoteWhenCodexSelected"
        status: pass
    human_judgment: false
  - id: D8
    description: "A CLI-level --pretool-nudge opt-in registers Codex's guard and hooks.json group, a plain install keeps it, and --pretool-nudge=false removes it — the same lifecycle proven at the agents-package level now proven through the real cobra command"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_PreToolNudge_CodexOptInRegisters"
        status: pass
    human_judgment: false
  - id: D9
    description: "docs/CLI-REFERENCE.md is regenerated only through task docs:cli and passes task docs:cli:drift, naming Codex, /hooks and \"never blocks a tool call\", with only the install and uninstall sections changed"
    requirement: "CODEX-05"
    verification:
      - kind: other
        ref: "task docs:cli:drift (exit 0, byte-identical to a fresh regeneration)"
        status: pass
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor"
        status: pass
    human_judgment: false
  - id: D10
    description: "TestOwnershipExactIdentity's codex leaves (still 32 total) plant a foreign ^Bash$ hooks.json group before Install; it survives install and uninstall byte-identical, and after uninstall no group carries codegraph's own command and the guard is gone"
    requirement: "CODEX-05"
    verification:
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipExactIdentity (32 leaves)"
        status: pass
    human_judgment: false
  - id: D11
    description: "Family (g1)-(g4): four planted-mutation positive controls (Keep collapsing into Off, the hooks-disabled early-return, the widened note narrowed back to Claude-only, and 242ec0a's matcher-shape ownership recovery reintroduced for Codex) each independently demonstrated RED and reverted byte-clean"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (g1)-(g4)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 8: Codex Nudge Lifecycle and CLI Opt-In Summary

**The Codex PreToolUse opt-in is now sticky on its own hooks.json evidence (Keep/On/Off), skipped with a Note when Codex hooks are explicitly disabled via any of three TOML forms, and `--pretool-nudge` widens end-to-end through the CLI, its help text, and the regenerated reference — proven against a real planted `^Bash$` ownership vulnerability and three lifecycle mutations.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-09-19T20:05:00Z (approx, first Read tool call)
- **Completed:** 2026-09-19T20:56:21Z (final Family (g) commit)
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- `tomlBoolSetting` (toml.go) reads a TOML boolean across three forms Codex's `config.toml` may use — a plain in-table `key = true|false`, a root dotted `table.key = …`, and a single-line inline `table = { … key = … }` — reusing the 07-01 line scanner rather than a second parser.
- `codexHooksExplicitlyDisabled`/`codexConfigHooksSetting` (codex_pretooluse.go) read the governing `config.toml` — local falling back to global — checking `[features] hooks` then the deprecated `codex_hooks` alias, naming the governing file and key as `source`.
- `codexHookTrustNote`/`codexHooksDisabledNote` render the D-19 "trust it in /hooks" Note on every successful opt-in write and the D-18 "hooks are explicitly disabled" Note on every skip.
- `codexTarget.Install`'s `PreToolNudge` dispatcher now implements the full Keep/On/Off lifecycle: On checks the hooks-disabled setting first; Keep refreshes only when `hasOwnHookBlock` finds codegraph's own group already in hooks.json (D-23 — no manifest involved) and is silent on a malformed or absent file; Off always uninstalls.
- `install.go`'s `--pretool-nudge` flag help, Long help, Example, and D-09 stderr note all widen from Claude-only to "Claude Code and Codex CLI"; `uninstall.go`'s Long help does the same. `docs/CLI-REFERENCE.md` regenerated via `task docs:cli` in the same commit as the help change, verified clean under `task docs:cli:drift`.
- `TestOwnershipExactIdentity`'s Codex leaves (still 32 total across all targets) now plant a foreign `{"matcher":"^Bash$","hooks":[{"command":"/opt/some-other-tool/on-startup.sh"}]}` group before every Install, asserting it survives install and uninstall byte-identical (242ec0a's exact-identity discipline, this time for Codex's own matcher).
- Family (g1)-(g4) in `07-MUTATION-LOG.md`: four planted mutations (Keep collapsing into Off, the hooks-disabled check silently answering "not disabled", the widened note narrowed back to Claude-only, and 242ec0a's matcher-shape recovery reintroduced for `^Bash$`) each demonstrated RED for real and reverted byte-clean.

## Task Commits

Each task was committed atomically (TDD RED/GREEN, per plan):

1. **Task 1 (RED): add failing Codex nudge lifecycle and hooks-off tests** — `86f6f80c` (test)
2. **Task 1 (GREEN): sticky Codex PreToolUse opt-in with the hooks-off skip and trust note** — `83d50206` (feat)
3. **Task 2 (RED): expect --pretool-nudge to configure Codex too** — `017e40b2` (test)
4. **Task 2 (GREEN): --pretool-nudge configures Codex CLI too** — `1adc1011` (feat)
5. **Task 2 fix (found while getting the full suite green): scope Codex's foreign-group assertion to leaves that planted it** — `363dfafd` (fix)
6. **Task 3: Family (g) mutation log** — `32ba6aa4` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note: Task 1 and Task 2 both followed TDD (`tdd="true"`) with genuine test → feat RED/GREEN pairs — see the RED Evidence section below for the pasted `--- FAIL:` transcripts. One intervening `fix(07-08)` commit corrects a bug this plan's own test additions introduced in a shared test helper (not the plan's target production code) — see Deviations._

## RED Evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestTOMLBoolSetting$|TestCodexPreToolNudge_|TestCodexNotesNeverAdviseTrustBypass$' -v` against the unmodified (pre-Task-1) code:

```
--- FAIL: TestCodexPreToolNudge_OnWritesAndNotesTrust (0.01s)
    --- FAIL: TestCodexPreToolNudge_OnWritesAndNotesTrust/local (0.00s)
    --- FAIL: TestCodexPreToolNudge_OnWritesAndNotesTrust/global (0.00s)
--- FAIL: TestCodexPreToolNudge_KeepRefreshesWhenOwnGroupPresent (0.00s)
--- PASS: TestCodexPreToolNudge_KeepNoopWhenNotOptedIn (0.00s)
--- PASS: TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing (0.00s)
    codex_pretooluse_test.go:709: Off left the guard (Lstat err <nil>)
--- FAIL: TestCodexPreToolNudge_OffRemovesAndForgets (0.00s)
--- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled (0.03s)
    --- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled/features_hooks_false_local (0.00s)
    --- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled/global_false_applies_to_local (0.00s)
    --- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled/codex_hooks_false_deprecated (0.00s)
    --- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled/dotted_root_key (0.00s)
    --- FAIL: TestCodexPreToolNudge_SkippedWhenHooksDisabled/inline_table (0.00s)
    --- PASS: TestCodexPreToolNudge_SkippedWhenHooksDisabled/local_true_overrides_global_false (0.00s)
    --- PASS: TestCodexPreToolNudge_SkippedWhenHooksDisabled/hooks_true_writes (0.00s)
--- PASS: TestCodexPreToolNudge_ReinstallIsIdempotent (0.00s)
--- PASS: TestCodexPreToolNudge_HandEditedOwnGroupDuplicates (0.00s)
--- PASS: TestCodexNotesNeverAdviseTrustBypass (0.03s)
--- PASS: TestTOMLBoolSetting (0.00s)
    (all 10 subtests PASS — tomlBoolSetting was written alongside its own test in this task,
    honestly recorded rather than reshaped to force an artificial RED; the genuine RED needed
    for this task lives entirely in the codex.go dispatcher tests above)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.224s
FAIL
```

`TestCodexPreToolNudge_KeepNoopWhenNotOptedIn`, `TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing`, `TestCodexPreToolNudge_ReinstallIsIdempotent`, `TestCodexPreToolNudge_HandEditedOwnGroupDuplicates`, and `TestCodexNotesNeverAdviseTrustBypass` already passed against the pre-Task-1 On-only implementation, since Keep was already a full no-op (matching "writes nothing") and On's existing idempotency/duplicate-handling needed no lifecycle change — honestly recorded per the 07-05/06/07 precedent rather than reshaped to force an artificial RED.

## RED Evidence (Task 2)

`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestInstall_PreToolNudge_|TestInstallHelpNeverAdvisesTrustBypass$' -v` against the unmodified (pre-Task-2) `install.go`/`uninstall.go`:

```
--- PASS: TestInstall_PreToolNudge_OptInRegisters (0.01s)
--- PASS: TestInstall_PreToolNudge_StickyAcrossPlainInstall (0.00s)
--- PASS: TestInstall_PreToolNudge_ExplicitFalseRemoves (0.03s)
--- FAIL: TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected (0.01s)
    --- FAIL: TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected/given_true (0.00s)
    --- FAIL: TestInstall_PreToolNudge_NoteWhenNeitherClaudeNorCodexSelected/given_false (0.00s)
    install_test.go:831: stderr carries the note 0 times, want 1; stderr:
        note: --pretool-nudge only configures Claude Code, which is not among the selected agents; nothing was changed for it
--- FAIL: TestInstall_PreToolNudge_NoNoteWhenCodexSelected (0.01s)
    install_test.go:858: note printed although Codex was selected; stderr:
        note: --pretool-nudge only configures Claude Code, which is not among the selected agents; nothing was changed for it
--- PASS: TestInstall_PreToolNudge_CodexOptInRegisters (0.01s)
--- FAIL: TestInstallHelpNeverAdvisesTrustBypass (0.00s)
    install_test.go:921: install --help does not mention /hooks: [old help text pasted]
--- PASS: TestInstall_PreToolNudge_NoNoteWhenClaudeSelected (0.01s)
FAIL
```

`TestInstall_PreToolNudge_CodexOptInRegisters` already passed — Task 1's `codex.go` dispatcher already implements the full lifecycle the CLI merely routes into, so this test needed no CLI-level behavior change, only the note/help wording did.

## Corpus/Table Coverage

- `TestTOMLBoolSetting`: 10/10 subtests pass, covering the three recognized forms and four "must report unset" shapes (non-bool value, absent key, key in another table, inside a multi-line string).
- `TestCodexPreToolNudge_SkippedWhenHooksDisabled`: 7/7 subtests pass, covering both key names, both scopes (with local-overrides-global), and all three `tomlBoolSetting` forms.
- `TestOwnershipExactIdentity`: 32/32 leaves pass (all 8 targets × {global, local} × {clean, foreign-codegraph-dir}), with the Codex leaves' new foreign-`^Bash$`-group plant/assert included.

## Files Created/Modified

- `internal/agents/toml.go` — `tomlBoolSetting`, `tomlKeyValue`, `tomlBoolLiteral`, `tomlInlineTableBoolValue`, `splitTOMLInlineTableEntries`
- `internal/agents/toml_test.go` — `TestTOMLBoolSetting` (10 subtests)
- `internal/agents/codex_pretooluse.go` — `codexHooksExplicitlyDisabled`, `codexConfigHooksSetting`, `codexHookTrustNote`, `codexHooksDisabledNote`, `codexHooksFileWasWritten`
- `internal/agents/codex_pretooluse_test.go` — the sticky-lifecycle test suite (On/Keep/Off, hooks-disabled skip, idempotency, hand-edit duplication, trust-bypass-never)
- `internal/agents/codex.go` — `codexTarget.Install`'s `PreToolNudge` dispatcher (On/Keep/Off)
- `internal/agents/ownership_test.go` — `ownershipCodexForeignBashGroup`, `plantForeignCodexHooksGroup`, `assertCodexOwnPreToolUseEntriesGoneAfterUninstall`, wired into `runOwnershipLeaf`
- `internal/cli/install.go` — widened `--pretool-nudge` note, flag help, Long help, Example
- `internal/cli/uninstall.go` — widened Long help
- `internal/cli/install_test.go` — renamed/widened note test, new Codex-selected/opt-in/help tests
- `docs/CLI-REFERENCE.md` — regenerated via `task docs:cli`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (g1)-(g4)

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug in own test code] `assertOwnEntriesGoneAfterUninstall`'s new Codex branch broke `TestOwnershipSharedInstructions`**
- **Found during:** running the full `internal/agents` suite after Task 2's GREEN commit, before Task 3
- **Issue:** The new Codex-specific PreToolUse assertion was added directly inside the shared helper `assertOwnEntriesGoneAfterUninstall`, which `TestOwnershipSharedInstructions` also calls for Codex — but that test never plants `ownershipCodexForeignBashGroup` (it installs with no `PreToolNudge` opt-in at all). The new branch unconditionally asserted hooks.json still exists post-uninstall, so `TestOwnershipSharedInstructions`'s six leaves failed with "`.codex/hooks.json` missing after uninstall — the unrelated `^Bash$` group should have survived", even though nothing was ever written there.
- **Fix:** Moved the Codex-specific assertion out of the shared helper into `runOwnershipLeaf` itself — the one call site that actually plants the foreign group before Install — leaving `assertOwnEntriesGoneAfterUninstall`'s generic behavior (used by both `TestOwnershipExactIdentity` and `TestOwnershipSharedInstructions`) unchanged.
- **Files modified:** `internal/agents/ownership_test.go`
- **Verification:** re-ran `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ -count=1` — both `ok` — before and after the fix, confirming the fix resolved the regression without breaking anything else.
- **Committed in:** `363dfafd` (fix)

**2. [Rule 1 - Bug in own test assertion] `TestCodexPreToolNudge_SkippedWhenHooksDisabled`'s Notes count was too strict**
- **Found during:** Task 1's GREEN run
- **Issue:** The test originally asserted `len(res.Notes) == 1` for the disabled cases, but a local Codex install also carries `codexTrustNote`'s unrelated D-10 "loads this project's MCP server only once trusted" Note — so a correct disabled skip legitimately produces 2 Notes at local scope, not 1.
- **Fix:** Changed the assertion to count only Notes containing "hooks are explicitly disabled", ignoring the unrelated trust-prompt Note.
- **Files modified:** `internal/agents/codex_pretooluse_test.go` (folded into the Task 1 GREEN commit, before it landed)
- **Verification:** re-ran the full `TestCodexPreToolNudge_SkippedWhenHooksDisabled` suite — all 7 subtests pass.
- **Committed in:** `83d50206` (part of the Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 bugs in this plan's own test code, discovered and corrected before they could ship as false failures or a hollow guard)
**Impact on plan:** No production-code behavior was affected by either fix — both are test-code corrections. Neither weakens any guard; the first fix actually restores test power that a naive fix (loosening the Codex assertion instead of relocating it) would have lost.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CODEX-05 is declared by six plans in this phase (07-01, 07-07, 07-08, 07-09, 07-10, 07-11); `gsd_run query requirements.ready-ids` confirms it is NOT yet ready to mark complete (0/1, siblings still pending) — correctly deferred, not marked here.
- The Codex nudge's full lifecycle (On/Keep/Off), its hooks-disabled skip, its trust Note, and its widened CLI surface are now complete end-to-end — 07-09/07-10/07-11 build on this without touching any file this plan modified except through their own declared scope.
- `docs/CLI-REFERENCE.md` is byte-identical to a fresh `task docs:cli` regeneration; `TestEveryRegisteredFlagIsAccountedFor` still passes with no new allowlist entries needed.
- No blockers.

## Self-Check: PASSED

- `internal/agents/toml.go` — FOUND, contains `func tomlBoolSetting(`
- `internal/agents/codex_pretooluse.go` — FOUND, contains `func codexHooksExplicitlyDisabled(loc Location) (bool, string, error) {`
- `internal/agents/codex.go` — FOUND, `PreToolNudgeKeep`/`PreToolNudgeOff` cases present in `Install`
- `docs/CLI-REFERENCE.md` — FOUND, mentions `Codex` and `/hooks`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — FOUND, contains `## Family (g1)` through `## Family (g4)`
- Commit `86f6f80c` (test, Task 1 RED) — FOUND in `git log --oneline --all`
- Commit `83d50206` (feat, Task 1 GREEN) — FOUND in `git log --oneline --all`
- Commit `017e40b2` (test, Task 2 RED) — FOUND in `git log --oneline --all`
- Commit `1adc1011` (feat, Task 2 GREEN) — FOUND in `git log --oneline --all`
- Commit `363dfafd` (fix, deviation 1) — FOUND in `git log --oneline --all`
- Commit `32ba6aa4` (docs, Task 3) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ -count=1` re-run at SUMMARY time — exit 0, both `ok`
- `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` re-run at SUMMARY time — exit 0, byte-identical
- `rg -c 'PreToolNudgeKeep' internal/agents/codex.go` — 1, confirmed
- `git show --stat 1adc1011 -- docs/CLI-REFERENCE.md` — lists the file, confirmed regenerated in the same commit as the help change

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
