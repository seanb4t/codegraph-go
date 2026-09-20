---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 05
subsystem: agents
tags: [skill-package, capability-table, gemini, kiro, antigravity, ownership-guard, tdd]

# Dependency graph
requires:
  - phase: 05-04
    provides: "installDeclaredSkill/uninstallDeclaredSkill (capabilities.go), the D-13 32-leaf ownership oracle (ownershipWantSkillDir, newSkillDirs), TestSkillFrontmatterMatchesEveryWrittenDir"
provides:
  - "geminiSkillDirs(loc): [.gemini/skills/codegraph (written), shared .agents/skills/codegraph alias (documented read path only, D-06 correction (a))]"
  - "kiroSkillDirs(loc): [.kiro/skills/codegraph (written, sole entry — no shared alias documented for Kiro)]"
  - "antigravitySkillDirs(loc), global-only: [~/.gemini/antigravity-cli/skills/codegraph (written), ~/.gemini/config/skills/codegraph ([ASSUMED] 2.0/IDE path, documented only, never written)]"
  - "D-13 ownership oracle (ownershipWantSkillDir) extended to gemini, kiro, antigravity — TestOwnershipExactIdentity stays 32/32 across all six skill-writing targets"
  - "TestSkillFrontmatterMatchesEveryWrittenDir floor raised 6 -> 11 written skill dirs"
  - "--print-config-style goldens re-frozen: gemini/kiro/antigravity skill= fields only"
affects: [05-06, 05-07]

# Actuals (#2632)
actuals:
  tokens: 7692
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Harness-specific SkillDirs (index 0 written, later entries documented read-paths only) reused verbatim for three more targets with no new writer primitives — installDeclaredSkill/uninstallDeclaredSkill already generalized this in 05-04"
    - "Rule 1 fix pattern repeats from 05-04: capabilities_test.go's independent oracle rows for a target pin its PRE-wiring state; wiring SkillDirs legitimately moves that state and the oracle row must move with it"
    - "RED tests call the not-yet-defined harness path through the EXISTING Capabilities().WrittenSkillDir(loc) method (returns \"\" pre-wiring) rather than a not-yet-written helper function, so RED is a genuine assertion failure (file-not-found) rather than a compile error"

key-files:
  created: []
  modified:
    - internal/agents/gemini.go
    - internal/agents/kiro.go
    - internal/agents/antigravity.go
    - internal/agents/gemini_test.go
    - internal/agents/kiro_test.go
    - internal/agents/antigravity_test.go
    - internal/agents/ownership_test.go
    - internal/agents/skillfrontmatter_test.go
    - internal/agents/capabilities_test.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/testdata/plain/print-config-style-local.golden

key-decisions:
  - "capabilities_test.go's TestCapabilitiesDeclared gemini/kiro/antigravity skillDir oracle rows updated from \"\" to their new harness dirs (Rule 1, same pattern 05-04 recorded): these rows pinned each target's PRE-wiring state, and this plan's own Capabilities() change legitimately moved that state."
  - "Kiro's and Antigravity's Uninstall restructured (CR-01, mirroring Cursor's 05-04 precedent) so a config-path resolution error no longer short-circuits the skill-uninstall step — every step now records its own outcome independently."
  - "RED tests for all three harnesses resolve the not-yet-declared skill dir through the target's existing Capabilities().WrittenSkillDir(loc) method rather than calling the plan's not-yet-written geminiSkillDirs/kiroSkillDirs/antigravitySkillDirs helpers directly — this keeps RED a genuine assertion failure (readFile: no such file or directory) instead of a Go compile error, which would not be valid RED evidence under the fail-fast rules (#3770)."

requirements-completed: [AGENT-10, AGENT-11, AGENT-07, AGENT-13]

coverage:
  - id: D1
    description: "Gemini CLI installs the codegraph skill at .gemini/skills/codegraph (harness-specific write), documents the shared .agents/skills/codegraph alias as read-only, never writes to it, and its GEMINI.md instructions are unaffected"
    requirement: "AGENT-10"
    verification:
      - kind: unit
        ref: "internal/agents/gemini_test.go#TestGemini_Install_WritesHarnessSkillDir"
        status: pass
      - kind: unit
        ref: "internal/agents/gemini_test.go#TestGemini_DescribePaths"
        status: pass
    human_judgment: false
  - id: D2
    description: "Kiro installs the codegraph skill at .kiro/skills/codegraph (its sole skill dir), writes no AGENTS.md at either scope, and its legacy steering self-heal is unchanged"
    requirement: "AGENT-11"
    verification:
      - kind: unit
        ref: "internal/agents/kiro_test.go#TestKiro_Install_WritesHarnessSkillDir"
        status: pass
      - kind: unit
        ref: "internal/agents/kiro_test.go#TestKiro_Install_WritesNoAgentsMd"
        status: pass
      - kind: unit
        ref: "internal/agents/kiro_test.go#TestKiro_Install_SelfHealsLegacySteeringFile"
        status: pass
    human_judgment: false
  - id: D3
    description: "Antigravity installs the codegraph skill at the agy CLI's global skill dir, documents the [ASSUMED] 2.0/IDE path as read-only without writing it, writes no AGENTS.md, and is a complete no-op at local scope"
    requirement: "AGENT-07"
    verification:
      - kind: unit
        ref: "internal/agents/antigravity_test.go#TestAntigravity_Install_WritesCliSkillDir"
        status: pass
      - kind: unit
        ref: "internal/agents/antigravity_test.go#TestAntigravity_ReadOnlySkillDirDocumented"
        status: pass
      - kind: unit
        ref: "internal/agents/antigravity_test.go#TestAntigravity_Local_WritesNothing"
        status: pass
    human_judgment: false
  - id: D4
    description: "The D-13 ownership oracle now covers all six skill-writing targets (claude, cursor, opencode, gemini, kiro, antigravity) — 32/32 leaves pass, and TestSkillFrontmatterMatchesEveryWrittenDir's floor is raised from 6 to 11 written dirs"
    requirement: "AGENT-13"
    verification:
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipExactIdentity (32 leaves)"
        status: pass
      - kind: unit
        ref: "internal/agents/skillfrontmatter_test.go#TestSkillFrontmatterMatchesEveryWrittenDir"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 5: Gemini CLI, Kiro, and Antigravity Skill Writes Summary

**Gemini CLI, Kiro, and Antigravity now each install the codegraph skill package at the harness-specific directory their own docs name, wired through the same table-derived writer 05-04 introduced, with the D-13 ownership guard extended to cover all six skill-writing targets.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-09-18T21:42:50Z
- **Completed:** 2026-09-18T22:23:11Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- `geminiSkillDirs(loc)` (`internal/agents/gemini.go`): returns `[.gemini/skills/codegraph (written), sharedSkillDirPath(loc) (documented read-only)]`. Per D-06 correction (a) — confirmed against the current `google-gemini/gemini-cli` docs (`docs/cli/skills.md`, fetched 2026-09-18) — Gemini CLI reads both at the same tier and the `.agents/skills/` alias actually outranks `.gemini/skills/` on a name collision, so a coexisting shared package (written by Cursor/opencode) is harmless. Only index 0 is written; nothing is ever written under `.agents/skills/` by Gemini. Wired through `installDeclaredSkill`/`uninstallDeclaredSkill` after the instructions step in `Install`/`Uninstall`. GEMINI.md instructions handling is unchanged.
- `kiroSkillDirs(loc)` (`internal/agents/kiro.go`): returns `[.kiro/skills/codegraph]` — the sole entry; no shared `.agents/skills/` alias is documented for Kiro's skill discovery. Per D-06 correction (d) and `kiro.dev/docs/steering` (fetched 2026-09-18), Kiro separately reads a literal `AGENTS.md` at `./AGENTS.md` / `~/.kiro/steering/AGENTS.md` on its own — recorded as an advisory (possible doubled instructions when another target writes those paths), not changed. Kiro's `Uninstall` was restructured (CR-01) so a config-path resolution error no longer skips the skill-uninstall step. The legacy steering self-heal and `kiroDisabledByDefaultNote` are untouched.
- `antigravitySkillDirs(loc)` (`internal/agents/antigravity.go`), **global-only**: returns `[~/.gemini/antigravity-cli/skills/codegraph (written — the surface 05-06's live `agy` session exercises), ~/.gemini/config/skills/codegraph ([ASSUMED] 2.0/IDE path, documented only, never written)]`. Per D-06 corrections (b)(c) and `antigravity.google/docs/skills.md`/`rules-workflows.md` (fetched 2026-09-18): the shared `.agents/skills/` alias is documented only at *workspace* scope for Antigravity, which this global-only target never reaches, and Antigravity's instructions arrive solely through Gemini's own `~/.gemini/GEMINI.md` write — no new `AGENTS.md`. Wired through `installDeclaredSkill`/`uninstallDeclaredSkill` at the end of the global `Install` path (after the CR-02 migration/marker logic, which is untouched); `Uninstall` restructured (CR-01, mirroring Kiro) so a config-path error no longer short-circuits the skill step.
- The D-13 ownership oracle (`ownershipWantSkillDir` in `internal/agents/ownership_test.go`) is extended with gemini, kiro, and antigravity (global) cases. `TestOwnershipExactIdentity` stays 32/32, now genuinely exercising all six skill-writing targets' foreign-dir `kept (foreign)` and clean-dir own-package assertions (`newSkillDirs`'s planting sites already covered these paths from 05-04).
- `TestSkillFrontmatterMatchesEveryWrittenDir`'s floor raised from `>= 6` to `>= 11` written skill dirs (claude, cursor, opencode, gemini, kiro × 2 scopes + antigravity × 1 = 11).
- `--print-config-style` goldens regenerated (`-update-plain-goldens`): the global golden's `gemini`, `kiro`, and `antigravity` lines' `skill=` fields changed; the local golden's `gemini`/`kiro` lines changed (Antigravity is global-only, so the local golden's antigravity line — `scopes=global (local not supported)` — is unaffected).

## Task Commits

Each task followed RED-GREEN TDD discipline:

1. **Task 1 (RED): failing Gemini and Kiro skill-dir tests** — `f8c202bb` (test)
2. **Task 1 (GREEN): Gemini CLI and Kiro install the skill at their own dirs** — `aecc78af` (feat)
3. **Task 2 (RED): failing Antigravity CLI skill-dir tests** — `4fc2e06c` (test)
4. **Task 2 (GREEN): Antigravity installs the skill at the agy CLI global dir** — `a9c64a17` (feat)

**Plan metadata:** committed alongside this SUMMARY.

### RED evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestGemini_|TestKiro_|TestOwnershipExactIdentity$|TestCapabilities|TestSkillFrontmatterMatchesEveryWrittenDir$' -v` against the `test(05-05): add failing Gemini and Kiro skill-dir tests` commit — every failure is an assertion mismatch (the harness path resolves to `""` pre-wiring, or a stale path-count), never a build error:

```
gemini_test.go:122: want exactly 4 paths (config + instructions + harness SKILL.md + manifest), got [/Users/sean/.gemini/settings.json /Users/sean/.gemini/GEMINI.md]
--- FAIL: TestGemini_DescribePaths (0.00s)
gemini_test.go:168: readFile(SKILL.md): open SKILL.md: no such file or directory
--- FAIL: TestGemini_Install_WritesHarnessSkillDir/global (0.00s)
--- FAIL: TestGemini_Install_WritesHarnessSkillDir/local (0.00s)
kiro_test.go:139: want exactly 3 paths (mcp config + Kiro SKILL.md + manifest), got [/Users/sean/.kiro/settings/mcp.json]
--- FAIL: TestKiro_DescribePaths_ListsMcpConfigAndSkill (0.00s)
kiro_test.go:181: readFile(SKILL.md): open SKILL.md: no such file or directory
--- FAIL: TestKiro_Install_WritesHarnessSkillDir/global (0.00s)
--- FAIL: TestKiro_Install_WritesHarnessSkillDir/local (0.00s)
    --- FAIL: TestOwnershipExactIdentity/gemini/global/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/gemini/global/foreign-codegraph-dir (0.00s)
    --- FAIL: TestOwnershipExactIdentity/gemini/local/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/gemini/local/foreign-codegraph-dir (0.00s)
    --- FAIL: TestOwnershipExactIdentity/kiro/global/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/kiro/global/foreign-codegraph-dir (0.00s)
    --- FAIL: TestOwnershipExactIdentity/kiro/local/clean (0.00s)
    --- FAIL: TestOwnershipExactIdentity/kiro/local/foreign-codegraph-dir (0.00s)
FAIL
```

The other 24 of 32 ownership leaves (claude, codex, cursor, hermes, opencode, antigravity) PASS unchanged at RED, confirming the guard is scoped to this task's targets, not a whole-suite false positive.

### RED evidence (Task 2)

Against the `test(05-05): add failing Antigravity CLI skill-dir tests` commit:

```
antigravity_test.go:220: readFile(.../.gemini/antigravity-cli/skills/codegraph/SKILL.md): open .../.gemini/antigravity-cli/skills/codegraph/SKILL.md: no such file or directory
--- FAIL: TestAntigravity_Install_WritesCliSkillDir (0.00s)
antigravity_test.go:268: ReadOnlySkillDirs(global) = [], want [.../.gemini/config/skills/codegraph]
--- FAIL: TestAntigravity_ReadOnlySkillDirDocumented (0.00s)
    --- FAIL: TestOwnershipExactIdentity/antigravity/global/clean (0.02s)
    --- FAIL: TestOwnershipExactIdentity/antigravity/global/foreign-codegraph-dir (0.01s)
    --- PASS: TestOwnershipExactIdentity/antigravity/local/clean (0.00s)
    --- PASS: TestOwnershipExactIdentity/antigravity/local/foreign-codegraph-dir (0.06s)
skillfrontmatter_test.go:106: checked 10 written skill dirs, want at least 11 (claude, cursor, opencode, gemini, kiro x 2 scopes + antigravity x 1)
--- FAIL: TestSkillFrontmatterMatchesEveryWrittenDir (0.00s)
FAIL
```

Antigravity's local leaves PASS unchanged at RED (global-only target, unaffected by the not-yet-declared global skill dir); only the 2 global leaves and the two Antigravity-specific tests and the frontmatter floor go RED — confirming the guard is scoped correctly.

### 32-leaf count (GREEN, both tasks)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestOwnershipExactIdentity$' -v | rg -c -- '--- PASS: TestOwnershipExactIdentity/[a-z]+/(global|local)/(foreign-codegraph-dir|clean) '
32
```

### Golden diffs (byte-confined to the affected fields)

`print-config-style-local.golden` (Task 1, gemini + kiro only):
```
-gemini: scopes=global,local mcp=.gemini/settings.json format=json instructions=GEMINI.md skill=none hooks=none
+gemini: scopes=global,local mcp=.gemini/settings.json format=json instructions=GEMINI.md skill=.gemini/skills/codegraph hooks=none
-kiro: scopes=global,local mcp=.kiro/settings/mcp.json format=json instructions=none skill=none hooks=none
+kiro: scopes=global,local mcp=.kiro/settings/mcp.json format=json instructions=none skill=.kiro/skills/codegraph hooks=none
```

`print-config-style.golden` (Task 1, gemini + kiro; Task 2, antigravity):
```
-antigravity: scopes=global mcp=<HOME>/.gemini/config/mcp_config.json format=json instructions=none skill=none hooks=none
+antigravity: scopes=global mcp=<HOME>/.gemini/config/mcp_config.json format=json instructions=none skill=<HOME>/.gemini/antigravity-cli/skills/codegraph hooks=none
-gemini: scopes=global,local mcp=<HOME>/.gemini/settings.json format=json instructions=<HOME>/.gemini/GEMINI.md skill=none hooks=none
+gemini: scopes=global,local mcp=<HOME>/.gemini/settings.json format=json instructions=<HOME>/.gemini/GEMINI.md skill=<HOME>/.gemini/skills/codegraph hooks=none
-kiro: scopes=global,local mcp=<HOME>/.kiro/settings/mcp.json format=json instructions=none skill=none hooks=none
+kiro: scopes=global,local mcp=<HOME>/.kiro/settings/mcp.json format=json instructions=none skill=<HOME>/.kiro/skills/codegraph hooks=none
```

## Files Created/Modified

- `internal/agents/gemini.go` — `geminiSkillDirs`; `Capabilities()` gains `SkillDirs: geminiSkillDirs`; `Install`/`Uninstall` call the two helpers
- `internal/agents/kiro.go` — `kiroSkillDirs`; `Capabilities()` gains `SkillDirs: kiroSkillDirs`; `Install`/`Uninstall` call the two helpers; CR-01 restructure
- `internal/agents/antigravity.go` — `antigravitySkillDirs`; `Capabilities()` gains `SkillDirs: antigravitySkillDirs`; `Install`/`Uninstall` call the two helpers; CR-01 restructure
- `internal/agents/gemini_test.go`, `internal/agents/kiro_test.go`, `internal/agents/antigravity_test.go` — new harness-skill-dir tests, updated `DescribePaths` pinned counts
- `internal/agents/ownership_test.go` — `ownershipWantSkillDir` extended with gemini/kiro/antigravity cases
- `internal/agents/skillfrontmatter_test.go` — floor raised 6 → 11
- `internal/agents/capabilities_test.go` — `TestCapabilitiesDeclared`'s gemini/kiro/antigravity skillDir oracle rows updated (Rule 1)
- `internal/cli/testdata/plain/print-config-style(-local).golden` — regenerated via `-update-plain-goldens`

## Decisions Made

See `key-decisions` in frontmatter — the three capabilities_test.go oracle updates (Rule 1, same pattern as 05-04), the Kiro/Antigravity CR-01 Uninstall restructure, and the RED-test-via-existing-method technique used to keep RED a genuine assertion failure rather than a compile error.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `TestCapabilitiesDeclared`'s stale gemini/kiro/antigravity skillDir oracle rows**
- **Found during:** Task 1 and Task 2 GREEN verification (full `internal/agents` package run)
- **Issue:** `TestCapabilitiesDeclared` pinned gemini's, kiro's, and antigravity's `WrittenSkillDir` as `""` — correct pre-plan state, but stale the moment this plan's own `Capabilities()` changes gave each target a `SkillDirs` resolver.
- **Fix:** Updated the three rows' `skillDir` expectations to each target's new harness dir, matching every other target's row shape in the same table (same pattern 05-04 recorded for Cursor/opencode).
- **Files modified:** `internal/agents/capabilities_test.go`
- **Verification:** `TestCapabilitiesDeclared` green afterward; full `internal/agents` suite green.
- **Committed in:** `aecc78af` (Task 1 GREEN commit) and `a9c64a17` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed pattern applied twice (2 bug fixes across the two tasks — same root cause each time). **Impact:** Both auto-fixes necessary corrections for test state the plan's own change legitimately moved; no scope creep.

## Issues Encountered

- **Pre-existing flaky integration test (out of scope):** `test/integration#TestLiveEditAutoSyncReachesExplore` FAILed once during the full-suite run ("codegraph_explore returned a tool error during a live watch session (CR-01 store-lock collision?)"). This plan makes zero changes under `test/integration`, `internal/watch`, or `internal/mcp` (confirmed via `git diff --stat`), and the test passes cleanly both alone and in a fresh `./test/integration/...` re-run — a timing-sensitive flake unrelated to this plan's work, not investigated further per the scope boundary (only auto-fix issues directly caused by the current task's changes).
- **RED-test technique note (not a defect):** the plan's interfaces section names `geminiSkillDirs`/`kiroSkillDirs`/`antigravitySkillDirs` as functions this plan creates; calling them directly from the new RED tests before they exist would be a Go compile error, not a valid RED assertion failure under the #3770 fail-fast rules. Each RED test instead resolves the not-yet-declared path through the target's existing `Capabilities().WrittenSkillDir(loc)` method (returns `""` pre-wiring), producing a genuine `readFile: no such file or directory` assertion failure — confirmed and pasted above for both tasks.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `05-06` can proceed to the live-session evidence (D-09…D-12): Cursor/opencode/Antigravity sessions on the maintainer's installed harnesses, plus `[ASSUMED]` documentation for Gemini CLI and Kiro (not installed locally, per D-10).
- `05-07` inherits a fully six-target D-13 ownership guard and an 11-dir frontmatter floor with no further capability-table work outstanding for this milestone's declared skill-writing targets.
- No blockers carried forward. The one flaky `test/integration` failure noted above is unrelated to this plan and was not further investigated (out of scope).

## Self-Check: PASSED

- `internal/agents/gemini.go` modified: FOUND
- `internal/agents/kiro.go` modified: FOUND
- `internal/agents/antigravity.go` modified: FOUND
- Commit `f8c202bb` (test): FOUND in `git log --oneline --all`
- Commit `aecc78af` (feat): FOUND in `git log --oneline --all`
- Commit `4fc2e06c` (test): FOUND in `git log --oneline --all`
- Commit `a9c64a17` (feat): FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... -count=1`: PASS (all packages ok)
- `TestOwnershipExactIdentity` 32/32 PASS leaves confirmed via `rg -c`
- Full module test (excluding `internal/daemon`, run alone per WINDOWS #37): 59/60 packages `ok`; `test/integration` flaked once on an unrelated live-watch test, confirmed passing on re-run and in isolation, zero diff under its own directory
- `internal/daemon` alone: `ok`
- Zero diff outside `internal/agents`/`internal/cli` since `4b3e109f` (05-04 complete): confirmed via `git diff --stat`
- Zero diff under `internal/query`, `internal/mcp`, `testdata/golden`, `testdata/wireoracle`, `go.mod`: confirmed empty
- `git status --short`: clean

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*
