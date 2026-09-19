---
phase: 07-codex-parity
plan: 05
subsystem: codex
tags: [codex, capabilities, toml, skills, trust, tdd]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "07-01/07-02's fixed TOML splice (findTOMLTableRange/tomlTableConflict/tomlLineEnding) and the corrected install --yes/--target ordering; 07-04's CODEX-01 live evidence (D-15 both roots read: yes; D-16/A2 not trust-gated: no; trust override grants no trust) consumed directly by this plan's implementation choices"
provides:
  - "Codex's Capabilities() literal declares Scopes {global, local}; Install/Uninstall are fully table-driven (no more loc != LocationGlobal early returns or direct codexConfigPath()/codexInstructionsPath() calls)"
  - "codexSkillDirs (shared .agents/skills/codegraph package plus the live-verified D-15 read-only .codex/skills and $CODEX_HOME/skills roots) and codexTrustNote (the D-10 advisory, wired only at local scope)"
  - "tomlTableConflict is now a visible, named Install/Uninstall error instead of a silent 'unchanged'; an emptied config.toml is removed entirely (D-07/D-08 keep-clean) rather than left empty"
  - "Family (d): four planted-mutation positive controls proving the scope-flip guards (Scopes literal, installDeclaredSkill wiring, the conflict check, the trust Note) can each independently fail"
affects: [07-06, 07-07, 07-09, 07-10, 07-11]

# Actuals (#2632)
actuals:
  tokens: 13892
  tasks: 3
  commits: 4
plan_head_before: a4ba516ffd7ffdf4808a5cc2a7d8b2491748b452

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Codex joins Cursor/opencode's shared-skill-package convention (D-14): codexSkillDirs returns the shared writer dir first, then live-verified read-only entries — the same PathsFunc shape Capabilities.SkillDirs already defines, no new mechanism"
    - "A capability-table target's Install/Uninstall can add a step-local advisory (codexTrustNote) purely from InstallOptions/loc, appended to WriteResult.Notes without touching any other step's outcome — each step still records independently (D-07's discipline generalizes past just file writes)"

key-files:
  created: []
  modified:
    - internal/agents/codex.go
    - internal/agents/codex_test.go
    - internal/agents/capabilities_test.go
    - internal/agents/ownership_test.go
    - internal/agents/registry_test.go
    - internal/cli/tui/agentpicker_test.go
    - internal/cli/printconfigstyle_test.go
    - internal/cli/install_test.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/testdata/plain/print-config-style-local.golden
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "Task 1 and Task 2 were authored as one combined codex_test.go RED commit rather than two separate RED cycles — Task 2's three planned tests (TestCodex_SharedSkillPackage_LastRequester, TestCodex_Install_Local_IsIdempotent, TestCodex_ReadOnlySkillDirsFollowLiveVerdict) went RED alongside Task 1's tests in the same run and turned GREEN the moment Task 1's Capabilities()/installDeclaredSkill/codexSkillDirs implementation landed — no separate code change was needed for Task 2. A second test(07-05) commit was still made (adding an explanatory doc comment, no code change) to satisfy Task 2's own TDD gate honestly, per the 05-03-SUMMARY.md precedent for a test that passes at RED."
  - "Rule 1 deviation: Task 3's own <verify> precondition `rg -c -F 'installDeclaredSkill(&result, t, loc)' internal/agents/codex.go = 1` is a buggy grep — it returns 2 because 'uninstallDeclaredSkill(&result, t, loc)' on a different line also matches the literal substring ('uninstall' contains 'install'). Verified this is a grep-pattern bug (not a real second call site) via a corrected negative-lookbehind pattern (count=1) and by confirming the planned perl mutation, which requires whitespace-only immediately before the call, touches exactly the one Install() call site — Uninstall's call is untouched. Recorded here rather than editing the plan or hand-patching codex.go to force a match it does not have."
  - "Rule 1 deviation: TestUninstall_YesWithExplicitTarget_HonoursTarget's pre-existing assertion (reading Codex's config.toml content after uninstall) was stale against the new D-07/D-08 keep-clean-on-empty behavior: since that config.toml only ever held the codegraph table, uninstalling it now removes the file entirely rather than leaving an empty one. Updated the assertion to check the file's absence instead of its content."
  - "codexTrustNote's wording follows 07-LIVE-SESSIONS.md's live verdicts exactly: it names the TUI trust prompt and the literal `[projects.\"<root>\"] trust_level = \"trusted\"` key (never the `-c projects...trust_level` override, since CODEX-01 confirmed that override grants no trust), and it explicitly says the codegraph skill and AGENTS.md block are read regardless of trust (D-16=no, A2=no)."
  - "codexSkillDirs declares BOTH D-15 read-only roots (.codex/skills locally, $CODEX_HOME/skills globally) since 07-LIVE-SESSIONS.md recorded both verdicts as 'yes' — codegraph never writes to either (D-14); DescribePaths never lists them, asserted directly in tests."

requirements-completed: [CODEX-01, CODEX-02, CODEX-03]

coverage:
  - id: D1
    description: "Codex's Capabilities() literal declares Scopes {global, local}; SupportsLocation, Detect, DescribePaths and --print-config-style all derive correctly from the table for the new local scope"
    requirement: "CODEX-02"
    verification:
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesDeclared/codex"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_SupportsLocation_GlobalAndLocal"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_DescribePaths_Local"
        status: pass
    human_judgment: false
  - id: D2
    description: "Install/Uninstall are fully table-driven: a local install writes .codex/config.toml, the repo-root AGENTS.md block, and the shared skill package; a local install carries a D-10 trust Note naming the absolute repo root; nothing is written under the fake HOME"
    requirement: "CODEX-02"
    verification:
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Install_Local_WritesConfigInstructionsAndSkill"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Install_Local_TrustNote"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Detect_Local_AfterInstallReportsConfigured"
        status: pass
      - kind: other
        ref: "real-binary tracer: codegraph install --target codex --location local --yes / uninstall (07-05-PLAN.md Task 1 <verify>)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An existing conflicting codegraph TOML definition is refused with a named error (never silently absorbed as unchanged), the config.toml stays byte-identical, and the AGENTS.md step still runs; an uninstall that empties config.toml removes the file entirely"
    requirement: "CODEX-02"
    verification:
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Install_RefusesConflictingCodegraphTable"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Uninstall_EmptiedConfigIsRemoved"
        status: pass
    human_judgment: false
  - id: D4
    description: "Codex is one more requester of the shared skill package (idempotent, sibling-safe) and declares only the D-15-verified read-only skill roots — never writes a second copy"
    requirement: "CODEX-03"
    verification:
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_SharedSkillPackage_LastRequester"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_Install_Local_IsIdempotent"
        status: pass
      - kind: unit
        ref: "internal/agents/codex_test.go#TestCodex_ReadOnlySkillDirsFollowLiveVerdict"
        status: pass
    human_judgment: false
  - id: D5
    description: "--target auto and the agent picker's pre-check detect Codex through a local .codex/ directory; the picker still lists all 8 targets"
    requirement: "CODEX-02"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestResolveTargetFlag_AutoDetectsCodexAtLocal"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerModel_PreChecksCodexAtLocal"
        status: pass
    human_judgment: false
  - id: D6
    description: "The first codex.go commit of this phase lands after the CODEX-01 evidence commit and rewrites the doc comment with a dated citation of 07-LIVE-SESSIONS.md, dropping the stale '4 of 8 targets' count"
    requirement: "CODEX-01"
    verification:
      - kind: other
        ref: "git ancestry + subject/diff assertions from 07-05-PLAN.md Task 1 <verify> (re-run at SUMMARY time, see Self-Check)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Family (d): four planted-mutation positive controls (Scopes literal, installDeclaredSkill wiring, the conflict check, the trust Note append) each independently demonstrated RED and reverted byte-clean"
    verification:
      - kind: other
        ref: ".planning/phases/07-codex-parity/07-MUTATION-LOG.md Family (d1)-(d4)"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 5: Codex Project-Local Scope Summary

**Codex CLI flips from global-only to {global, local} through the capability table — local install writes `.codex/config.toml`, the shared repo-root `AGENTS.md`, and the shared skill package, with a live-evidence-grounded trust Note and a visible TOML-conflict refusal.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-09-19T13:20:00-04:00 (approx, first Read tool call)
- **Completed:** 2026-09-19T14:15:00-04:00 (approx, final commit)
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- Codex's `Capabilities()` literal now declares `Scopes: {global, local}`, per-location `codexConfigPath`/`codexInstructionsPath` (`PathFunc`), and `codexSkillDirs` (the shared `.agents/skills/codegraph` package plus the live-verified D-15 read-only `.codex/skills` and `$CODEX_HOME/skills` roots). `Install`/`Uninstall` are now fully table-driven — the `loc != LocationGlobal` early returns and direct path-function calls are gone.
- Every local install writes `.codex/config.toml` (through the fixed 07-01 splice), the shared repo-root `AGENTS.md` marker block, and the shared skill package (`installDeclaredSkill`), and appends a D-10 trust Note naming the absolute repo root and the real TUI trust prompt / `[projects."<root>"] trust_level = "trusted"` key — codegraph never writes that entry itself, and the Note never suggests the `-c projects...trust_level` override (07-LIVE-SESSIONS.md confirmed it grants no trust).
- An existing conflicting `codegraph` TOML definition is now a visible, named `Install`/`Uninstall` error (via `tomlTableConflict`) rather than a silent "unchanged," and an uninstall that empties `config.toml` removes the file entirely (the `removeMarkedSection`/`removeHookEntry` keep-clean precedent, now applying to Codex's TOML config at both scopes).
- `--target auto` and the agent picker's pre-check detect Codex through a local `.codex/` directory (`Detect`'s existing `fileExists(filepath.Dir(configPath))` fallback needed no change — only the capability table's `Scopes` did).
- Regenerated the two `print-config-style` plain goldens via `-update-plain-goldens`: exactly one line changed in each, matching the interfaces block.
- Four Family (d) mutation-log entries prove the scope-flip guards can each independently fail: the `Scopes` literal, the `installDeclaredSkill` wiring, the TOML conflict check, and the trust Note append — all planted, observed RED for real, and reverted byte-clean.

## Task Commits

Each task was committed atomically (TDD RED/GREEN, per plan):

1. **Task 1 (RED): expect Codex at local scope through the capability table** — `b36e48d` (test)
2. **Task 1 (GREEN): Codex project-local scope through the capability table** — `3d39080` (feat)
3. **Task 2 (RED, already-GREEN honest record): pin Codex's shared-skill requester and read-only skill dirs** — `1fec97f` (test; no code change required, see Deviations)
4. **Task 3: Family (d) mutation log** — `4249752` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note: this plan followed TDD (`tdd="true"` on Tasks 1-2); Task 1 produced its own test→feat pair, Task 2's three planned tests were already satisfied by Task 1's implementation._

## Files Created/Modified

- `internal/agents/codex.go` — dual-scope `Capabilities()`, table-driven `Install`/`Uninstall`, `codexSkillDirs`, `codexTrustNote`, corrected doc comment citing 07-LIVE-SESSIONS.md
- `internal/agents/codex_test.go` — replaced global-only tests with dual-scope equivalents; added local-install, trust-note, conflict-refusal, empty-removal, describe-paths, detect, shared-skill-requester, and read-only-skill-dirs tests
- `internal/agents/capabilities_test.go` — Codex's row in `TestCapabilitiesDeclared` now expects both scopes and the shared skill dir; `TestCapabilitiesMatchInstallWrites` leaf floor raised 13 → 14
- `internal/agents/ownership_test.go` — `ownershipWantSkillDir` groups Codex with Cursor/opencode (shared skill package)
- `internal/agents/registry_test.go` — new `TestResolveTargetFlag_AutoDetectsCodexAtLocal` (real registry, scratch `.codex/` dir)
- `internal/cli/tui/agentpicker_test.go` — new `TestAgentPickerModel_PreChecksCodexAtLocal` (real registry, scratch `.codex/` dir)
- `internal/cli/printconfigstyle_test.go` — `TestInstallPrintConfigStyle_Filters` switched its global-only example from codex to hermes
- `internal/cli/install_test.go` — `TestUninstall_ReportsUnsupportedForWrongLocation` switched to hermes; `TestUninstall_YesWithExplicitTarget_HonoursTarget`'s Codex-config assertion updated for the new keep-clean-on-empty behavior
- `internal/cli/testdata/plain/print-config-style.golden` / `print-config-style-local.golden` — regenerated, one codex line each
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (d1)-(d4)

## RED Evidence (Task 1, representative excerpt)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ ./internal/cli/tui/ -count=1 -run '...' -v` against the unmodified (global-only) `codex.go`, 16 tests failing as expected:

```
--- FAIL: TestCapabilitiesDeclared (0.00s)
    --- FAIL: TestCapabilitiesDeclared/codex (0.00s)
    capabilities_test.go:434: executed 13 leaf subtests, want at least 14 (6 global+local targets x 2, 2 global-only targets x 1)
--- FAIL: TestCapabilitiesMatchInstallWrites (0.03s)
    codex_test.go:25: codex should support local (D-09 scope flip)
--- FAIL: TestCodex_SupportsLocation_GlobalAndLocal (0.00s)
    codex_test.go:45: readFile(.../.codex/config.toml): open .../.codex/config.toml: no such file or directory
--- FAIL: TestCodex_Install_Local_WritesConfigInstructionsAndSkill (0.00s)
    codex_test.go:97: expected exactly one trust note, got 0: []
--- FAIL: TestCodex_Install_Local_TrustNote (0.00s)
    codex_test.go:132: expected a conflict error, got none: {Files:[] Notes:[] Errors:[]}
--- FAIL: TestCodex_Install_RefusesConflictingCodegraphTable (0.00s)
    codex_test.go:179: precondition: config.toml should exist after install
--- FAIL: TestCodex_Uninstall_EmptiedConfigIsRemoved (0.00s)
    codex_test.go:223: DescribePaths(local) missing ".../.codex/config.toml", got []
--- FAIL: TestCodex_DescribePaths_Local (0.00s)
--- FAIL: TestCodex_Install_Local_IsIdempotent (0.00s)
    codex_test.go:383: expected AlreadyConfigured after local install, got {Installed:false AlreadyConfigured:false ConfigPath:}
--- FAIL: TestCodex_Detect_Local_AfterInstallReportsConfigured (0.00s)
    codex_test.go:431: targets = [opencode], want exactly {codex, opencode}
--- FAIL: TestCodex_SharedSkillPackage_LastRequester (0.01s)
    codex_test.go:488: ReadOnlySkillDirs(local) = [], want to contain ".../.codex/skills/codegraph" (D-15 .codex/skills read: yes)
--- FAIL: TestCodex_ReadOnlySkillDirsFollowLiveVerdict (0.00s)
--- FAIL: TestOwnershipExactIdentity (0.26s)
    --- FAIL: TestOwnershipExactIdentity/codex/global/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/codex/global/foreign-codegraph-dir (0.00s)
--- FAIL: TestOwnershipExactIdentity_CrossCheckWrittenSkillDir (0.01s)
    --- FAIL: TestOwnershipExactIdentity_CrossCheckWrittenSkillDir/codex/global (0.00s)
    registry_test.go:189: expected auto-detected targets to include codex given a scratch .codex/ dir, got [{}]
--- FAIL: TestResolveTargetFlag_AutoDetectsCodexAtLocal (0.00s)
    agentpicker_test.go:109: expected the Codex row to start checked given a scratch .codex/ dir, checked=map[7:true]
--- FAIL: TestAgentPickerModel_PreChecksCodexAtLocal (0.00s)
```

All 16 turned GREEN after Task 1's `feat(07-05)` commit; the two `print-config-style` goldens were then regenerated and the full targeted suite re-ran clean.

## D-15/D-16 Branches Taken (07-LIVE-SESSIONS.md CODEX-01 verdicts)

- **D-15 `.codex/skills` read: yes** and **D-15 `CODEX_HOME/skills` read: yes** (both verdicts recorded PASS in 07-04's live evidence) → `codexSkillDirs` declares BOTH read-only roots (local `.codex/skills/codegraph`, global `~/.codex/skills/codegraph`), never written (D-14). `TestCodex_ReadOnlySkillDirsFollowLiveVerdict` pins this and `DescribePaths` is asserted to never list either.
- **D-16 project `.agents/skills` trust-gated: no** and **A2 AGENTS.md trust-gated in a real session: no** → `codexTrustNote`'s wording says the codegraph skill and AGENTS.md block are read regardless of trust — it does NOT claim they need trust, matching the live evidence exactly.
- **Trust override (`-c projects...trust_level`) grants trust: no** → `codexTrustNote` names only the real TUI trust prompt and the `[projects."<root>"] trust_level = "trusted"` key, never the CLI override, as the plan required.

## Golden Diff

```
--- print-config-style.golden
-codex: scopes=global mcp=<HOME>/.codex/config.toml format=toml instructions=<HOME>/.codex/AGENTS.md skill=none hooks=none
+codex: scopes=global,local mcp=<HOME>/.codex/config.toml format=toml instructions=<HOME>/.codex/AGENTS.md skill=<HOME>/.agents/skills/codegraph hooks=none

--- print-config-style-local.golden
-codex: scopes=global (local not supported)
+codex: scopes=global,local mcp=.codex/config.toml format=toml instructions=AGENTS.md skill=.agents/skills/codegraph hooks=none
```

Exactly one line changed in each golden, as the interfaces block specified.

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan's own Task 3 precondition grep has a substring-collision bug**
- **Found during:** Task 3 (Family (d2) mutation)
- **Issue:** The plan's `<verify>` precondition `rg -c -F 'installDeclaredSkill(&result, t, loc)' internal/agents/codex.go = "1"` returns `2`, not `1` — `uninstallDeclaredSkill(&result, t, loc)` on a separate line also matches the literal substring, since `"uninstall"` contains `"install"` as a substring.
- **Fix:** Verified the substance directly: a corrected pattern `rg -c -P '(?<!un)installDeclaredSkill\(&result, t, loc\)'` returns `1`, and the planned perl mutation itself (which requires whitespace-only immediately before the call) is confirmed to touch exactly the one `Install()` call site — `Uninstall`'s call is untouched, verified via `git diff`. Proceeded with the real mutation, RED capture, and revert; documented the grep bug in `07-MUTATION-LOG.md`'s Family (d2) entry rather than editing the plan text or codex.go to force a match that isn't real.
- **Files modified:** none (documentation only, in `07-MUTATION-LOG.md`)
- **Verification:** ran both the naive and corrected patterns side by side; confirmed identical mutation diff either way
- **Committed in:** `4249752` (Family (d) mutation log)

**2. [Rule 1 - Bug] Stale test assertion after the D-07/D-08 keep-clean behavior change**
- **Found during:** Task 1's full-package regression check (`go test ./internal/agents/... ./internal/cli/...`)
- **Issue:** `TestUninstall_YesWithExplicitTarget_HonoursTarget` (pre-existing, `internal/cli/install_test.go`) asserted Codex's global `config.toml` still existed (with the codegraph table stripped) after an explicit `--target codex` uninstall. Since that config.toml only ever held the codegraph table, Task 1's new keep-clean-on-empty behavior (a strip that empties the file removes it entirely, matching `removeMarkedSection`'s precedent) now deletes the file, and `readFileString` failed with "no such file or directory."
- **Fix:** Updated the assertion to check the file's absence (`os.Stat` + `os.IsNotExist`) instead of reading and inspecting its content, with a comment explaining why.
- **Files modified:** `internal/cli/install_test.go`
- **Verification:** full `internal/agents`/`internal/cli` package suites green after the fix
- **Committed in:** `3d39080` (Task 1 GREEN commit)

---

**Total deviations:** 2 auto-fixed (1 wrong-query bug in a plan-authored verify script, documented rather than silently patched around; 1 stale test assertion made stale by an intentional, plan-mandated behavior change)
**Impact on plan:** Both were necessary corrections with no scope creep — neither touched product behavior beyond what the plan already specified.

## Issues Encountered

- Full-repo `go test ./...` (run once as an out-of-scope regression sanity check, not part of this plan's own verify gate) showed one failure: `internal/daemon`'s `TestDaemonSharedWriter` ("store lock held: injected contention" / "session B did not converge"). Re-ran `internal/daemon` alone — passed cleanly in 64s. This is the pre-existing, already-tracked cross-package-load flake documented in `.planning/WINDOWS.md` row 37 and `.planning/debug/resolved/daemon-stale-linux-flake.md` (rotating failure set including `TestDaemonSharedWriter`, confirmed unrelated to `internal/agents`/`internal/cli` changes — `internal/daemon` has zero dependency on either package). Not fixed (out of scope); not re-logged (already open).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CODEX-01, CODEX-02, and CODEX-03 are all marked complete in `REQUIREMENTS.md` (all declaring plans — 07-04/07-05 for CODEX-01; 07-01/07-02/07-05 for CODEX-02; 07-05 alone for CODEX-03 — now have SUMMARYs).
- `internal/agents/codex.go` is now fully table-driven for scope, config, instructions, and skill dirs; `Hooks` stays `HooksNone` — 07-07 is unblocked to add `codex-json` hooks onto the same literal without touching anything this plan built.
- 07-06 (D-11's shared-`AGENTS.md` uninstall rule, D-12's `AGENTS.override.md` Note) can build directly on this plan's local `AGENTS.md` wiring — Codex and opencode already share the identical repo-root path and marker contract.
- The real-binary local install/uninstall tracer (Task 1's `<verify>`) confirms the shared skill package, the trust Note, and the TOML splice all work end-to-end through the actual compiled binary, not just unit tests.
- No blockers.

## Self-Check: PASSED

- `internal/agents/codex.go` — FOUND, contains `Scopes:       []Location{LocationGlobal, LocationLocal}` and `07-LIVE-SESSIONS.md`
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — FOUND, contains `## Family (d1)` through `## Family (d4)`
- Commit `b36e48d` (test, Task 1 RED) — FOUND in `git log --oneline --all`
- Commit `3d39080` (feat, Task 1 GREEN) — FOUND in `git log --oneline --all`
- Commit `1fec97f` (test, Task 2) — FOUND in `git log --oneline --all`
- Commit `4249752` (docs, Task 3) — FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... -count=1` re-run at SUMMARY time — exit 0, all packages `ok`
- First `internal/agents/codex.go` commit since the 07-01 plans commit is `3d39080` (`feat(07-05): ...`), an ancestor-confirmed descendant of `97c82219` (`docs(07-04): record CODEX-01 live evidence`), and its diff contains a line citing `07-LIVE-SESSIONS.md` — confirmed
- `rg -q 'codexConfigPath\(\)|codexInstructionsPath\(\)|globalOnlyPath\(codex' internal/agents/codex.go` exits 1 (no match) — confirmed
- Both `internal/cli/testdata/plain/print-config-style{,-local}.golden` changed exactly one line each — confirmed via `git show --stat`

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
