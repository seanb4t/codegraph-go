---
phase: 05-agent-reach-capability-model-skill-in-every-harness
plan: 01
subsystem: agents
tags: [capability-table, install-cli, print-config-style, agent-config, cobra]

# Dependency graph
requires: []
provides:
  - "AgentTarget.Capabilities() Capabilities on all eight targets (D-01, D-02) — one struct literal per target file, referencing existing path functions by name"
  - "internal/agents/capabilities.go: HookMechanism, ConfigFormat, PathFunc, PathsFunc, Capabilities{Scopes,ConfigFormat,Hooks,MCPConfig,Instructions,SkillDirs} + Supports/InstructionsPath/WrittenSkillDir/ReadOnlySkillDirs/HookFiles, describeDeclaredPaths, globalOnlyPath, skillFileName/skillManifestFileName"
  - "SupportsLocation, DescribePaths and Detect's path inputs are now derivations of the table in all eight targets — no second hand-written copy of any declared path"
  - "codegraph install --print-config-style: read-only, honours -t/--target and -l/--location, styled+plain parity, frozen plain goldens, regenerated docs/CLI-REFERENCE.md"
  - "D-03 guard proven bidirectionally RED: TestCapabilitiesTableDrivesDerivations (16 leaves), TestCapabilitiesMatchInstallWrites (13 leaves), Family (a1)/(a2) in 05-MUTATION-LOG.md"
affects: [05-02, 05-03, 05-04, 05-05, 05-06, 05-07]

# Actuals (#2632)
actuals:
  tokens: 20275
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Capability table as single source of truth: AgentTarget.Capabilities() drives SupportsLocation/DescribePaths/Detect, with a bidirectional guard (declared-vs-written) rather than trusting the derivation by inspection"
    - "PathFunc/PathsFunc as first-class table fields — no target keeps a second hand-written path string once the table declares it"

key-files:
  created:
    - internal/agents/capabilities.go
    - internal/agents/capabilities_test.go
    - internal/cli/printconfigstyle.go
    - internal/cli/printconfigstyle_test.go
    - internal/cli/testdata/plain/print-config-style.golden
    - internal/cli/testdata/plain/print-config-style-local.golden
    - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md
  modified:
    - internal/agents/types.go
    - internal/agents/claude.go
    - internal/agents/cursor.go
    - internal/agents/codex.go
    - internal/agents/opencode.go
    - internal/agents/hermes.go
    - internal/agents/gemini.go
    - internal/agents/antigravity.go
    - internal/agents/kiro.go
    - internal/agents/registry_test.go
    - internal/cli/tui/agentpicker_test.go
    - internal/cli/install.go
    - internal/cli/plain_golden_test.go
    - docs/CLI-REFERENCE.md

key-decisions:
  - "Narrowed antigravityConfigPath() to resolve to the unified path on a fresh machine with no Antigravity config at all (not just once migrated), matching the path Install actually writes there — a real bug the D-03 TestCapabilitiesDeclared_AntigravityMigrationAware test caught before this plan's own GREEN commit landed"
  - "HookFiles hardcodes claudeSettingsPath/claudeHooksScriptPath for HooksClaudeJSON (only Claude declares this mechanism this phase) and returns errHookFilesUndeclared for HooksCodexJSON — a future literal switching to it without declaring its files fails loudly via the D-03 guard rather than silently describing nothing"
  - "Split RED/GREEN across two commits per D-00's TDD discipline: capabilities.go's real logic landed in the RED commit (it does not itself assert anything the RED tests check), while all eight targets' Capabilities() literals were RED zero-value placeholders until the GREEN commit — the four named tests fail on assertion, never on a build error, at the RED commit"

requirements-completed: [AGENT-08]

coverage:
  - id: D1
    description: "AgentTarget gains Capabilities() Capabilities; one struct literal per target file (8), plus the two interface fakes, referencing existing path functions by name with no restated path strings"
    requirement: "AGENT-08"
    verification:
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesDeclared"
        status: pass
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesDeclared_AntigravityMigrationAware"
        status: pass
    human_judgment: false
  - id: D2
    description: "SupportsLocation, DescribePaths and Detect's path inputs are derivations of Capabilities() in all eight targets, with a bidirectional D-03 guard (declared paths equal derived paths; every created/updated/unchanged Install file is declared)"
    requirement: "AGENT-08"
    verification:
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesTableDrivesDerivations"
        status: pass
      - kind: unit
        ref: "internal/agents/capabilities_test.go#TestCapabilitiesMatchInstallWrites"
        status: pass
      - kind: unit
        ref: "internal/agents/ (full package, -count=1)"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph install --print-config-style is read-only (writes nothing, opens no picker), honours -t/--target and -l/--location filtering, and reports resolution errors and unknown-target errors correctly"
    requirement: "AGENT-08"
    verification:
      - kind: unit
        ref: "internal/cli/printconfigstyle_test.go#TestInstallPrintConfigStyle"
        status: pass
      - kind: unit
        ref: "internal/cli/printconfigstyle_test.go#TestInstallPrintConfigStyle_ReadOnly"
        status: pass
      - kind: unit
        ref: "internal/cli/printconfigstyle_test.go#TestInstallPrintConfigStyle_Filters"
        status: pass
      - kind: integration
        ref: "real binary: install --print-config-style prints 8 lines, writes zero files under a scratch HOME/cwd"
        status: pass
    human_judgment: false
  - id: D4
    description: "--print-config-style renders identically styled and plain (SGR-stripped equality), and the plain bytes are frozen as goldens (print-config-style, print-config-style-local) alongside the pre-existing 29 cases"
    requirement: "AGENT-08"
    verification:
      - kind: unit
        ref: "internal/cli/printconfigstyle_test.go#TestInstallPrintConfigStyle_StyledStripsToPlain"
        status: pass
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestPlainGolden (31 cases)"
        status: pass
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestNoColorNonTTYRegression"
        status: pass
    human_judgment: false
  - id: D5
    description: "docs/CLI-REFERENCE.md is regenerated through task docs:cli in the same commit that adds the flag, with task docs:cli:drift and TestEveryRegisteredFlagIsAccountedFor green and no allowlist entry for the new flag"
    requirement: "AGENT-08"
    verification:
      - kind: unit
        ref: "internal/cli/cli_reference_test.go#TestEveryRegisteredFlagIsAccountedFor"
        status: pass
      - kind: other
        ref: "task docs:cli:drift"
        status: pass
    human_judgment: false

duration: 33min
completed: 2026-09-18
status: complete
---

# Phase 5 Plan 1: Capability Model & Read-Only `--print-config-style` Summary

**One per-target `Capabilities()` table now drives `SupportsLocation`, `DescribePaths`, `Detect`'s path resolution, and a new read-only `codegraph install --print-config-style` flag — proven bidirectionally by a D-03 guard demonstrated RED against two independent mutations.**

## Performance

- **Duration:** 33 min
- **Started:** 2026-09-18T19:20:08Z
- **Completed:** 2026-09-18T19:53:10Z
- **Tasks:** 3
- **Files modified:** 21 (7 created, 14 modified)

## Accomplishments

- `AgentTarget` gained `Capabilities() Capabilities`; all eight target files (Claude, Cursor, Codex, opencode, Hermes, Gemini, Antigravity, Kiro) each declare one struct literal describing their scopes, config format, hook mechanism, and per-location MCP-config/instructions/skill-dir resolvers — referencing their existing path functions by name, never restating a path string.
- `SupportsLocation`, `DescribePaths`, and `Detect`'s path resolution are now pure derivations of that table in every target, with the installed-evidence fallback in `Detect` computed as an ancestor of a table path (e.g. `filepath.Dir` of the resolved instructions/config path) rather than a hand-typed literal.
- Found and fixed a real bug while building the table's own test oracle: `antigravityConfigPath()` resolved to the *legacy* path on a machine with no Antigravity config at all, while `Install` always writes the *unified* path there — narrowed so the table (and `Detect`/`DescribePaths`) now print exactly what `Install` writes on a fresh machine.
- Shipped `codegraph install --print-config-style`: read-only, short-circuits before any write-capable branch, honours `-t/--target` and `-l/--location`, renders styled and plain identically, and is golden-frozen (`print-config-style`, `print-config-style-local`) alongside the pre-existing 29 plain-golden cases.
- Proved the D-03 capability-table guard in both directions with a real RED/GREEN cycle: `TestCapabilitiesTableDrivesDerivations` (16 leaves) catches a path `DescribePaths` reports that the table doesn't declare; `TestCapabilitiesMatchInstallWrites` (13 leaves) catches a path `Install` writes that the table stops declaring. Both are demonstrated RED in `05-MUTATION-LOG.md` (Family (a1)/(a2)) against planted, byte-cleanly-reverted mutations.
- Regenerated `docs/CLI-REFERENCE.md` through `task docs:cli` in the same commit that added the flag; `task docs:cli:drift` and `TestEveryRegisteredFlagIsAccountedFor` are green with no allowlist entry needed.

## Task Commits

Each task was committed atomically, split across RED/GREEN per this plan's TDD discipline:

1. **Task 1 (RED): capability-table and --print-config-style tests** — `16708421` (test)
2. **Task 1 (GREEN): capability table + read-only install --print-config-style** — `c569c1a5` (feat)
3. **Task 2: derive SupportsLocation/DescribePaths/Detect from the table** — `dd076f52` (refactor)
4. **Task 3: styled branch, frozen goldens, regenerated CLI reference** — `f1116725` (feat)
5. **Task 3: Family (a) mutation-log demonstrations** — `3a8d97fb` (docs)

_TDD tasks produce multiple commits (test → feat → refactor); this plan's tracer task (Task 1) is RED→GREEN, Task 2 is a behaviour-preserving refactor with no RED-first commit (its positive control is Family (a)), and Task 3 bundles the styled/goldens/reference work into one feat commit plus a separate docs commit for the mutation log._

### RED evidence (Task 1)

`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/ ./internal/cli/tui/ -count=1 -run 'TestCapabilitiesDeclared$|TestInstallPrintConfigStyle$|TestInstallPrintConfigStyle_ReadOnly$|TestInstallPrintConfigStyle_Filters$' -v` against the `test(05-01):` commit:

```
capabilities_test.go:149: Scopes = [], want [global]
    capabilities_test.go:149: Scopes = [], want [global local]
    ... (one per target)
--- FAIL: TestCapabilitiesDeclared (0.01s)
    --- FAIL: TestCapabilitiesDeclared/antigravity (0.00s)
    --- FAIL: TestCapabilitiesDeclared/claude (0.00s)
    --- FAIL: TestCapabilitiesDeclared/codex (0.00s)
    --- FAIL: TestCapabilitiesDeclared/cursor (0.00s)
    --- FAIL: TestCapabilitiesDeclared/gemini (0.00s)
    --- FAIL: TestCapabilitiesDeclared/hermes (0.00s)
    --- FAIL: TestCapabilitiesDeclared/kiro (0.00s)
    --- FAIL: TestCapabilitiesDeclared/opencode (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.189s
    printconfigstyle_test.go:87: stdout has 7 lines, want 8 (one per registered target):
        Claude Code: configured
          created: .../.claude.json
          ... (a real Install run, not the read-only report)
--- FAIL: TestInstallPrintConfigStyle (0.00s)
    printconfigstyle_test.go:149: runAgentPicker must never be called for --print-config-style
--- FAIL: TestInstallPrintConfigStyle_ReadOnly (0.00s)
    printconfigstyle_test.go:189: expected exactly 2 lines, got 8:
        ...
--- FAIL: TestInstallPrintConfigStyle_Filters (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.459s
```

Every failure is an assertion mismatch (wrong scopes, wrong stdout shape, a real write happening) — never a build error. `TestAgentPickerModel_*` (the fake-stub compile check) passed unchanged throughout.

## Files Created/Modified

- `internal/agents/capabilities.go` — `Capabilities` struct, `HookMechanism`/`ConfigFormat` enums, `PathFunc`/`PathsFunc`, `describeDeclaredPaths`, `globalOnlyPath`
- `internal/agents/capabilities_test.go` — `TestCapabilitiesDeclared(_AntigravityMigrationAware)`, `TestCapabilitiesTableDrivesDerivations`, `TestCapabilitiesMatchInstallWrites`
- `internal/agents/{claude,cursor,codex,opencode,hermes,gemini,antigravity,kiro}.go` — one `Capabilities()` literal each; `SupportsLocation`/`DescribePaths`/`Detect` rewritten as table derivations
- `internal/agents/types.go` — `AgentTarget` interface gains `Capabilities() Capabilities`
- `internal/agents/registry_test.go`, `internal/cli/tui/agentpicker_test.go` — the two interface fakes gain `Capabilities()`
- `internal/cli/printconfigstyle.go` — `configStyleFields`, `printConfigStyle` (plain + styled)
- `internal/cli/printconfigstyle_test.go` — `TestInstallPrintConfigStyle(_ReadOnly|_Filters|_StyledStripsToPlain)`
- `internal/cli/install.go` — `--print-config-style` flag + read-only RunE branch, new Example line
- `internal/cli/plain_golden_test.go` — two new golden cases, doc-comment count 29→31
- `internal/cli/testdata/plain/print-config-style(-local).golden` — frozen bytes
- `docs/CLI-REFERENCE.md` — regenerated (install section only)
- `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md` — Family (a1)/(a2)

## Decisions Made

See `key-decisions` in frontmatter — the Antigravity fresh-machine bug fix, the `HookFiles`/`errHookFilesUndeclared` loud-failure design for an undeclared hook mechanism, and the RED/GREEN commit split.

## Deviations from Plan

None — plan executed exactly as written, including the planner's noted Antigravity narrowing and the RED/GREEN split called out in the plan's own "Planner notes."

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The capability table is the single source of truth every remaining phase-5 plan (05-02 through 05-07) edits by adding fields/literals rather than hand-writing a second path — the phase's stated dependency chain (table + guard first → ownership test → shared writers → per-harness wiring → live sessions → docs) is unblocked.
- `05-02` can now extend `skillManifest` and add the shared skill writer against a stable `Capabilities.SkillDirs`/`WrittenSkillDir`/`ReadOnlySkillDirs` contract.
- No blockers or concerns carried forward.

## Self-Check: PASSED

- `internal/agents/capabilities.go` exists: FOUND
- `internal/agents/capabilities_test.go` exists: FOUND
- `internal/cli/printconfigstyle.go` exists: FOUND
- `internal/cli/printconfigstyle_test.go` exists: FOUND
- `internal/cli/testdata/plain/print-config-style.golden` exists: FOUND
- `internal/cli/testdata/plain/print-config-style-local.golden` exists: FOUND
- `.planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md` exists: FOUND
- Commit `16708421` (test): FOUND in `git log --oneline --all`
- Commit `c569c1a5` (feat): FOUND in `git log --oneline --all`
- Commit `dd076f52` (refactor): FOUND in `git log --oneline --all`
- Commit `f1116725` (feat): FOUND in `git log --oneline --all`
- Commit `3a8d97fb` (docs): FOUND in `git log --oneline --all`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... -count=1`: PASS (all packages ok)
- `task docs:cli:drift`: exit 0, byte-identical
- Real binary `install --print-config-style`: 8 lines, zero files written under a scratch HOME/cwd

---
*Phase: 05-agent-reach-capability-model-skill-in-every-harness*
*Completed: 2026-09-18*
