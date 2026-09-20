---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 04
subsystem: agents
tags: [skill-package, ownership-guard, capability-table, cursor, opencode, tdd, 242ec0a]

# Dependency graph
requires:
  - phase: 05-03
    provides: "Claude's Install/Uninstall on the manifest-owned writer (installSkillPackage/uninstallSkillPackage), claudeSkillPolicy's D-17 symlink comparison, manifestRequesters"
provides:
  - "TestOwnershipExactIdentity (internal/agents/ownership_test.go): the D-13 32-leaf planted-foreign-entry table over all 8 targets x {global,local} x {clean,foreign-codegraph-dir}, plus an independent WrittenSkillDir cross-check subtest — cites commit 242ec0a418703c6a4dab45188242149960cda77d by SHA and reproduces its exact precondition for the claude/* leaves (D-15)"
  - "Cursor and opencode install the shared .agents/skills/codegraph package through installDeclaredSkill/uninstallDeclaredSkill (capabilities.go) — the ONE table every non-Claude skill-writing target now funnels through (AGENT-08)"
  - "TestSkillFrontmatterMatchesEveryWrittenDir: the embedded SKILL.md's frontmatter (name/description) is valid against the name-must-match-folder rule Cursor, opencode and Kiro's own docs enforce, checked against every target's WrittenSkillDir"
  - "install's per-agent headline treats kept (foreign) as not-a-change; the per-file line still renders it in the Warning role"
  - "05-MUTATION-LOG.md Family (b1)/(b2): both historical shapes of the 242ec0a-class ownership vulnerability (shape-based skill-dir ownership; literal matcher-based hook ownership) demonstrated RED and reverted byte-clean"
affects: [05-05, 05-06, 05-07]

# Actuals (#2632)
actuals:
  tokens: 14715
  tasks: 2
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Planted-foreign-entry table as the D-13 ownership guard shape: plant a foreign sibling at every write site BEFORE Install, assert byte-identity only after the full Install-then-Uninstall round trip (not the necessarily-different mid-install state)"
    - "installDeclaredSkill/uninstallDeclaredSkill as the single table-derived skill step every non-Claude target calls — no target hand-rolls its own skill-directory write once it declares SkillDirs"
    - "CR-01 restructure: a config-path resolution error records itself via result.Errors and falls through to the remaining steps, rather than returning early and silently skipping them"

key-files:
  created:
    - internal/agents/ownership_test.go
    - internal/agents/skillfrontmatter_test.go
  modified:
    - internal/agents/capabilities.go
    - internal/agents/capabilities_test.go
    - internal/agents/cursor.go
    - internal/agents/cursor_test.go
    - internal/agents/opencode.go
    - internal/agents/opencode_test.go
    - internal/cli/install.go
    - internal/cli/install_test.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/testdata/plain/print-config-style-local.golden
    - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md

key-decisions:
  - "capabilities_test.go's TestCapabilitiesDeclared Cursor/opencode skillDir oracle rows updated from \"\" to the shared path (Rule 1): this 05-01 test pinned the pre-plan state as a fixed independent oracle, and this plan's own Capabilities() change legitimately moved that state — the test needed updating, not the implementation."
  - "The plan's own Task 1 verify script counts occurrences of the literal substring installDeclaredSkill(&result, t, loc) across cursor.go/opencode.go and expects 2, but uninstallDeclaredSkill(&result, t, loc) contains that exact substring (uninstall = \"un\" + \"install\"), so a correct implementation wiring BOTH Install and Uninstall through the two symmetric helpers the plan's own interfaces section requires produces 4 matches, not 2. Confirmed with a lookbehind-corrected check (rg -P '(?<!un)installDeclaredSkill\\(&result, t, loc\\)') = 2 (Install-side only) and a separate uninstallDeclaredSkill(&result, t, loc) count = 2 (Uninstall-side, 1 per file) — both sides wired exactly once per file as intended. Every other check in the verify chain passed unmodified; this is a plan-authoring rg-substring-inclusion footgun, not an implementation defect."
  - "reproduce242ec0aPrecondition (D-15) runs a real pre-install (establishing Claude's own manifest) before hand-editing the SessionStart 'startup' matcher slot with an unrelated command, inside the SAME leaf that plants D-13's general foreign fixtures — the two requirements compose without conflict since writeMcpEntry/writeHookEntry only ever touch codegraph's own key/command-identity, never a sibling's."

requirements-completed: [AGENT-13, AGENT-04, AGENT-06, AGENT-09]

coverage:
  - id: D1
    description: "TestOwnershipExactIdentity: 32-leaf planted-foreign-entry table over all 8 targets x {global,local} x {clean,foreign-codegraph-dir}, citing 242ec0a by SHA and reproducing its exact precondition for claude/* leaves (D-15)"
    requirement: "AGENT-13"
    verification:
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipExactIdentity (32 leaves)"
        status: pass
      - kind: unit
        ref: "internal/agents/ownership_test.go#TestOwnershipExactIdentity_CrossCheckWrittenSkillDir"
        status: pass
    human_judgment: false
  - id: D2
    description: "Cursor and opencode install/uninstall the shared codegraph skill package through installDeclaredSkill/uninstallDeclaredSkill; opencode's AGENTS.md block is retained; installing both at one location yields one manifest with both targets"
    requirement: "AGENT-04"
    verification:
      - kind: unit
        ref: "internal/agents/cursor_test.go#TestCursor_Install_WritesSharedSkillPackage"
        status: pass
      - kind: unit
        ref: "internal/agents/cursor_test.go#TestCursor_DescribePaths_ListsMcpConfigAndSharedSkill"
        status: pass
      - kind: unit
        ref: "internal/agents/opencode_test.go#TestOpencode_Install_WritesSharedSkillPackage"
        status: pass
      - kind: unit
        ref: "internal/agents/opencode_test.go#TestOpencode_CursorShareOnePackage"
        status: pass
      - kind: unit
        ref: "internal/agents/opencode_test.go#TestOpencode_DescribePaths"
        status: pass
    human_judgment: false
  - id: D3
    description: "SKILL.md frontmatter (name=codegraph, valid pattern/length, non-empty description <=1024 chars) matches the base name of every target's written skill directory (>=6 checked: claude, cursor, opencode x 2 scopes) — the name-must-match-folder rule Cursor/opencode/Kiro enforce"
    requirement: "AGENT-06"
    verification:
      - kind: unit
        ref: "internal/agents/skillfrontmatter_test.go#TestSkillFrontmatterMatchesEveryWrittenDir"
        status: pass
    human_judgment: false
  - id: D4
    description: "install's per-agent headline treats kept (foreign) as not-a-change; the per-file line still renders it in the Warning role"
    requirement: "AGENT-09"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstallStatus_KeptForeignIsNotAChange"
        status: pass
    human_judgment: false
  - id: D5
    description: "The ownership guard is proven to fail against both historical shapes of the 242ec0a-class vulnerability (shape-based skill-dir ownership; literal matcher-based hook ownership), each reverted byte-clean with a green control after"
    requirement: "AGENT-13"
    verification:
      - kind: other
        ref: ".planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md Family (b1)/(b2)"
        status: pass
    human_judgment: false

duration: 38min
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 4: D-13 Ownership Guard, Cursor/opencode Shared Skill, SKILL.md Frontmatter Contract Summary

**A 32-leaf exact-identity ownership table (citing commit 242ec0a by SHA and reproducing its precondition) now guards every harness's skill/config/instructions writes, Cursor and opencode install the shared codegraph skill package through the one capability-table-derived helper, and the guard is proven to fail against both historical shapes of the ownership vulnerability.**

## Performance

- **Duration:** 38 min
- **Started:** 2026-09-18T20:53:00Z
- **Completed:** 2026-09-18T21:31:00Z
- **Tasks:** 2
- **Files modified:** 13 (2 created, 11 modified)

## Accomplishments

- `TestOwnershipExactIdentity` (`internal/agents/ownership_test.go`): 32 leaves (8 targets x {global,local} x {clean,foreign-codegraph-dir}), each planting a foreign MCP entry (in the target's own canonical on-disk format), a foreign instructions section, a foreign `.agents/skills/other/` sibling, and — in the foreign variant — a manifest-less SKILL.md at every `newSkillDirs` root (shared, `.gemini/skills`, `.kiro/skills`, plus Antigravity's two global-only documented paths). Every foreign artifact is asserted byte-identical after the full Install-then-Uninstall round trip; every codegraph-owned entry is asserted gone. The doc comment cites commit `242ec0a418703c6a4dab45188242149960cda77d` by SHA and names the differential it closed. The `claude/*` leaves additionally reproduce that exact precondition (install once to establish a manifest, hand-edit the SessionStart `"startup"` matcher slot with an unrelated command, re-install, uninstall) per D-15.
- An independent `WrittenSkillDir` oracle (`ownershipWantSkillDir`) cross-checks `Capabilities().WrittenSkillDir` for every target x supported location in a separate subtest, so the table and the oracle cannot drift together.
- Cursor and opencode now declare `SkillDirs: sharedSkillDirs` and call the new `installDeclaredSkill`/`uninstallDeclaredSkill` (`capabilities.go`) — the ONE table-derived helper install/uninstall funnel through for every skill-writing target except Claude (its own symlink-aware `claudeSkillPolicy`). Cursor's `Install` was restructured (CR-01) so a config-path resolution error no longer skips the skill step. opencode's existing AGENTS.md instructions write is unchanged.
- `TestSkillFrontmatterMatchesEveryWrittenDir` (`internal/agents/skillfrontmatter_test.go`) parses the embedded SKILL.md's frontmatter with the standard library only and asserts it satisfies the name-must-match-folder rule Cursor, opencode, and Kiro's own docs (fetched 2026-09-18) independently enforce, checked against every target's written skill directory (6 dirs today: claude, cursor, opencode x 2 scopes).
- `install`'s per-agent headline now treats `kept (foreign)` like `unchanged` (a foreign directory codegraph left untouched is not a change codegraph made, D-14); the per-file line still renders `kept (foreign): <path>` in the Warning role.
- Two `05-MUTATION-LOG.md` Family (b) entries prove the guard fails against both historical shapes of the `242ec0a`-class vulnerability: (b1) widening `skillDirIsForeign` to also accept a bare SKILL.md as ownership proof; (b2) the literal `242ec0a` reintroduction — `writeHookEntry`'s `isOwned` claiming any `"startup"`-matcher block regardless of command identity. Both confirmed applied, demonstrated RED with verbatim transcripts, reverted byte-clean, and re-verified green.

## Task Commits

Each task followed RED-GREEN TDD discipline (Task 1) or a single feat+docs pair (Task 2, `type="auto"` — no `tdd="true"` attribute):

1. **Task 1 (RED): failing exact-identity ownership table and shared-skill tests** — `c26c4fed` (test)
2. **Task 1 (GREEN): Cursor and opencode install the shared skill package** — `49a0e5de` (feat)
3. **Task 2: SKILL.md frontmatter contract and kept-foreign install headline** — `a2888e2c` (feat)
4. **Task 2: Family (b) mutation-log demonstrations** — `ee8ae1a9` (docs)

**Plan metadata:** committed alongside this SUMMARY.

### RED evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestOwnershipExactIdentity$|TestCursor_|TestOpencode_|TestCapabilities' -v` against the `test(05-04):` commit — every failure is an assertion mismatch against the not-yet-wired Cursor/opencode Capabilities(), never a build error:

```
cursor_test.go:142: want exactly 3 paths (mcp config + shared SKILL.md + manifest), got [.../.cursor/mcp.json]
--- FAIL: TestCursor_DescribePaths_ListsMcpConfigAndSharedSkill (0.00s)
cursor_test.go:184: readFile(.../.agents/skills/codegraph/SKILL.md): no such file or directory
--- FAIL: TestCursor_Install_WritesSharedSkillPackage (0.00s)
opencode_test.go:265: want exactly 4 paths (config + instructions + shared SKILL.md + manifest), got [.../opencode.json .../AGENTS.md]
--- FAIL: TestOpencode_DescribePaths (0.00s)
opencode_test.go:287: readFile(.../.agents/skills/codegraph/SKILL.md): no such file or directory
--- FAIL: TestOpencode_Install_WritesSharedSkillPackage (0.00s)
opencode_test.go:323: after cursor install: targets=[] present=false err=<nil>, want [cursor]
--- FAIL: TestOpencode_CursorShareOnePackage (0.00s)
ownership_test.go:527: expected {.../.agents/skills/codegraph, "kept (foreign)"} in Install result, got [{Path:.../.cursor/mcp.json Action:created}]
--- FAIL: TestOwnershipExactIdentity (0.08s)
    --- FAIL: TestOwnershipExactIdentity/cursor/global/clean
    --- FAIL: TestOwnershipExactIdentity/cursor/global/foreign-codegraph-dir
    --- FAIL: TestOwnershipExactIdentity/cursor/local/clean
    --- FAIL: TestOwnershipExactIdentity/cursor/local/foreign-codegraph-dir
    --- FAIL: TestOwnershipExactIdentity/opencode/global/clean
    --- FAIL: TestOwnershipExactIdentity/opencode/global/foreign-codegraph-dir
    --- FAIL: TestOwnershipExactIdentity/opencode/local/clean
    --- FAIL: TestOwnershipExactIdentity/opencode/local/foreign-codegraph-dir
FAIL
```

The other 24 of 32 ownership leaves (claude, codex, gemini, hermes, kiro, antigravity) PASS unchanged at RED, confirming the guard is scoped to the targets this plan wires, not a whole-suite false positive.

### 32-leaf count (GREEN)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestOwnershipExactIdentity$' -v | rg -c -- '--- PASS: TestOwnershipExactIdentity/[a-z]+/(global|local)/(foreign-codegraph-dir|clean) '
32
```

### Family (b) outcomes

- **(b1)** — `skillDirIsForeign` widened to also accept a bare SKILL.md as ownership proof (`internal/agents/skillshared.go`) → `TestOwnershipExactIdentity/cursor/{global,local}/foreign-codegraph-dir` and `.../opencode/{global,local}/foreign-codegraph-dir` go RED (foreign SKILL.md silently overwritten, manifest created where none should exist). Reverted byte-clean; green control confirmed.
- **(b2)** — the literal `242ec0a` reintroduction: `writeHookEntry`'s `isOwned` claims any `"startup"`-matcher block regardless of command identity (`internal/agents/shared.go`) → all four `TestOwnershipExactIdentity/claude/*` leaves go RED (`settings.json missing after uninstall — the unrelated startup hook should have survived`). Reverted byte-clean; green control confirmed.

Full transcripts, diffs, and revert proofs are recorded in `05-MUTATION-LOG.md` under "Family (b1)" / "Family (b2)".

## Files Created/Modified

- `internal/agents/ownership_test.go` — `TestOwnershipExactIdentity` (32 leaves), `TestOwnershipExactIdentity_CrossCheckWrittenSkillDir`, `ownershipWantSkillDir`, `newSkillDirs`, and every planting/assertion helper
- `internal/agents/skillfrontmatter_test.go` — `TestSkillFrontmatterMatchesEveryWrittenDir`, `parseSkillFrontmatter`, `skillNamePattern`
- `internal/agents/capabilities.go` — `installDeclaredSkill`, `uninstallDeclaredSkill`
- `internal/agents/cursor.go` — `Capabilities()` gains `SkillDirs: sharedSkillDirs`; `Install`/`Uninstall` call the two helpers; CR-01 restructure
- `internal/agents/opencode.go` — `Capabilities()` gains `SkillDirs: sharedSkillDirs`; `Install`/`Uninstall` call the two helpers
- `internal/agents/cursor_test.go`, `internal/agents/opencode_test.go` — updated `DescribePaths` pinned counts (1→3, 2→4) and new shared-skill-package tests
- `internal/agents/capabilities_test.go` — `TestCapabilitiesDeclared`'s Cursor/opencode skillDir oracle rows updated (Rule 1)
- `internal/cli/install.go` — `installStatus` treats `ActionKeptForeign` like `ActionUnchanged`
- `internal/cli/install_test.go` — `TestInstallStatus_KeptForeignIsNotAChange`
- `internal/cli/testdata/plain/print-config-style(-local).golden` — regenerated via `-update-plain-goldens`; diff confined to the cursor/opencode lines
- `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md` — Family (b1)/(b2)

## Decisions Made

See `key-decisions` in frontmatter — the `capabilities_test.go` oracle update (Rule 1), the plan's own verify-script rg-substring-inclusion footgun (documented, corrected check run instead, every other check green unmodified), and the D-15 precondition/general-fixture composition order.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `TestCapabilitiesDeclared`'s stale Cursor/opencode skillDir oracle**
- **Found during:** Task 1 GREEN verification (full `internal/agents` package run)
- **Issue:** `TestCapabilitiesDeclared` (from 05-01) pinned Cursor's and opencode's `WrittenSkillDir` as `""` — the correct pre-plan state, but stale the moment this plan's own `Capabilities()` change gave both targets `SkillDirs: sharedSkillDirs`.
- **Fix:** Updated the two rows' `skillDir` expectations to the shared path (`filepath.Join(home, ".agents", "skills", "codegraph")` / `filepath.Join(".agents", "skills", "codegraph")`), matching every other target's row shape in the same table.
- **Files modified:** `internal/agents/capabilities_test.go`
- **Verification:** `TestCapabilitiesDeclared` green afterward; full `internal/agents` suite green.
- **Committed in:** `49a0e5de` (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug). **Impact:** Necessary correctness fix for a test whose own fixed state the plan's change legitimately moved; no scope creep.

## Issues Encountered

- **Plan verify-script rg-substring footgun (not a code defect):** Task 1's `<verify><automated>` block counts occurrences of the literal substring `installDeclaredSkill(&result, t, loc)` across `cursor.go`/`opencode.go` and asserts it equals `2`. Because `uninstallDeclaredSkill(&result, t, loc)` contains that exact substring (`uninstall` = `"un"` + `"install"`), a correct implementation that wires BOTH `Install` and `Uninstall` through the two symmetric helpers the plan's own interfaces section requires produces `4` matches, not `2`. Confirmed via a lookbehind-corrected check: `rg -P -o '(?<!un)installDeclaredSkill\(&result, t, loc\)' internal/agents/cursor.go internal/agents/opencode.go | wc -l` = `2` (the Install-side calls only), and a separate `uninstallDeclaredSkill(&result, t, loc)` count = `2` (the Uninstall-side calls, one per file) — confirming both sides are correctly wired exactly once per file. Every other check in the plan's `<verify><automated>` chain (build, gofmt, vet, the 32-leaf count, `TestOpencode_CursorShareOnePackage`, the `242ec0a` citation, the packages-ok check, and both golden-line checks) passed unmodified. Recorded here rather than silently reshaping the plan's own script.

## User Setup Required

None — no external service configuration required.

## Advisory: pre-existing legacy self-heal deletion (T-05-19, out of scope)

Per this plan's own "Planner notes" and threat register (T-05-19, disposition `accept`): `.cursor/rules/codegraph.mdc` and `.kiro/steering/codegraph.md` are deleted BY NAME on install today — a same-name deletion that predates this phase (pinned by `TestCursor_Install_SelfHealsLegacyRulesFile` / `TestKiro_Install_SelfHealsLegacySteeringFile`), not a write this plan adds. `TestOwnershipExactIdentity` deliberately does not plant foreign content at those two exact paths (per the plan's own instruction), so this pre-existing gap is neither exercised nor fixed here. Recorded as a follow-up advisory, not changed in this plan.

## Next Phase Readiness

- `05-05` can wire Gemini/Kiro/Antigravity onto `installDeclaredSkill`/`uninstallDeclaredSkill` (Gemini/Kiro via their own harness-specific `SkillDirs` entries, D-06) with `TestOwnershipExactIdentity`'s existing `newSkillDirs`/`ownershipWantSkillDir` oracle ready to extend, and `TestSkillFrontmatterMatchesEveryWrittenDir`'s floor (currently `>= 6`) ready to rise.
- `05-06`/`05-07` own the live-session evidence (D-09…D-12) this plan's tests deliberately do not attempt (D-00): whether Cursor and opencode actually READ `.agents/skills/codegraph/` is established there, never by a unit test.
- No blockers carried forward beyond the recorded T-05-19 advisory above.

## Self-Check: PASSED

- `internal/agents/ownership_test.go` exists: FOUND
- `internal/agents/skillfrontmatter_test.go` exists: FOUND
- Commit `c26c4fed` (test): FOUND in `git log --oneline --all`
- Commit `49a0e5de` (feat): FOUND in `git log --oneline --all`
- Commit `a2888e2c` (feat): FOUND in `git log --oneline --all`
- Commit `ee8ae1a9` (docs): FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... -count=1`: PASS (all packages ok)
- `TestOwnershipExactIdentity` 32/32 PASS leaves confirmed via `rg -c`
- Full module test (excluding `internal/daemon`, run alone per WINDOWS #37): all 60 packages `ok`
- `internal/daemon` alone: `ok`
- Zero diff outside `internal/agents`/`internal/cli`/the mutation log since `c373e091` (05-03 complete): confirmed via `git diff --stat`
- Zero diff under `internal/query`, `internal/mcp`, `testdata/golden`, `testdata/wireoracle`, `go.mod`: confirmed empty
- `git status --short`: clean

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*
