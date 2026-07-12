---
phase: 06-agent-integrations-cli-lifecycle
plan: 02
subsystem: infra
tags: [installer, agent-integration, json, marker-fences, tdd, go-registry, claude-code, cursor, gemini, kiro, antigravity]

# Dependency graph
requires:
  - phase: 06-agent-integrations-cli-lifecycle
    provides: "internal/agents foundation (06-01): AgentTarget interface, writeMcpEntry/removeMcpEntry/upsertInstructionsEntry/removeMarkedSection, registerTarget/registry"
provides:
  - "5 self-registering AgentTarget implementations: claudeTarget, cursorTarget, geminiTarget, kiroTarget, antigravityTarget"
  - "Package-level shared helpers (in claude.go, reused across all 5): fileExists, stdioMcpEntry, mcpEntryPresent, instructionsBody"
  - "Claude settings.json permission helpers: addClaudeAllowPermission/removeClaudeAllowPermission"
  - "Antigravity's independent no-type entry builder + unified/legacy ~/.gemini config path resolution (antigravityConfigPath, readMcpEntry)"
affects: [06-04, 06-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-agent target file = struct + init()-self-registration + <agent>ConfigPath/<agent>InstructionsPath path resolvers + Detect/Install/Uninstall/DescribePaths, mirroring internal/indexer/languages_go.go"
    - "Shared package-level helpers (fileExists, stdioMcpEntry, mcpEntryPresent, instructionsBody) defined once in the first-written target file (claude.go) and reused by sibling target files in the same package — no re-derivation per file"
    - "t.Chdir(dir) + fakeHome(t) test isolation for local-vs-global scope tests (avoids polluting the real HOME or cwd)"

key-files:
  created:
    - internal/agents/claude.go
    - internal/agents/cursor.go
    - internal/agents/gemini.go
    - internal/agents/kiro.go
    - internal/agents/antigravity.go
    - internal/agents/claude_test.go
    - internal/agents/cursor_test.go
    - internal/agents/gemini_test.go
    - internal/agents/kiro_test.go
    - internal/agents/antigravity_test.go
  modified: []

key-decisions:
  - "stdioMcpEntry (the shared {type:'stdio', command, args} builder) and fileExists/mcpEntryPresent/instructionsBody live in claude.go rather than shared.go, since 06-01's shared.go was already committed and these are agent-target-shape helpers, not surgical-write primitives — Antigravity deliberately does NOT call stdioMcpEntry, using its own antigravityEntry builder instead (Pitfall 6)"
  - "instructionsBody() derives Claude/Gemini's upsertInstructionsEntry content by stripping instructions.go's codegraphSectionStart/End markers off codegraphInstructionsBlock at runtime, rather than duplicating the block text in a second constant — guarantees the two representations can never drift out of byte-for-byte sync (D-01a)"
  - "Antigravity's unified-vs-legacy detection uses a ~/.gemini/config/.migrated marker file the Go port itself creates on first install (not just reads) — a fresh install writes straight to the unified path and creates the marker in the same call, so a second install never re-runs the legacy-sweep branch"
  - "Cursor's local --path arg is resolved through filepath.EvalSymlinks(os.Getwd()) so the written path matches what a symlinked tmpdir test fixture reports back via the same resolution, keeping the test assertion and the implementation on the same canonicalization"

patterns-established:
  - "Only Claude and Gemini call upsertInstructionsEntry; Cursor and Kiro self-heal-delete a legacy instructions/steering file on install and never write one; Antigravity writes neither, sharing Gemini's GEMINI.md by omission"
  - "DescribePaths for the 3 no-instructions targets (Cursor/Kiro/Antigravity) lists ONLY the MCP config path — the self-healed legacy file is a delete-check target, never an ongoing DescribePaths entry"

requirements-completed: [AGNT-01, AGNT-02, AGNT-03]

coverage:
  - id: D1
    description: "claudeTarget: global ~/.claude.json + local ./.mcp.json (never ./.claude.json), CLAUDE.md marker instructions, AutoAllow settings.json permission, legacy ./.claude.json migration/strip, round-trip byte-invariance"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/agents/claude_test.go (TestClaude_*)"
        status: pass
    human_judgment: false
  - id: D2
    description: "cursorTarget: --path quirk (local=absolute cwd, global=literal ${workspaceFolder}), no instructions file, legacy .cursor/rules/codegraph.mdc self-heal-delete, round-trip byte-invariance"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/agents/cursor_test.go (TestCursor_*)"
        status: pass
    human_judgment: false
  - id: D3
    description: "antigravityTarget: no-type entry builder, global-only, unified/legacy ~/.gemini path detection + stale-legacy sweep, no instructions file of its own"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/agents/antigravity_test.go (TestAntigravity_*)"
        status: pass
    human_judgment: false
  - id: D4
    description: "geminiTarget: type:stdio entry, project-root ./GEMINI.md for local instructions (not ./.gemini/GEMINI.md), ~/.gemini/GEMINI.md for global, round-trip byte-invariance"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/agents/gemini_test.go (TestGemini_*)"
        status: pass
    human_judgment: false
  - id: D5
    description: "kiroTarget: type:stdio entry, no instructions file, legacy steering/codegraph.md self-heal-delete (global+local), MCP-disabled-by-default WriteResult.Notes hint, round-trip byte-invariance"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/agents/kiro_test.go (TestKiro_*)"
        status: pass
    human_judgment: false
  - id: D6
    description: "All 5 targets self-register via init(); AllTargetIDs() includes claude/cursor/gemini/kiro/antigravity; internal/agents remains boundary-clean (no graphstore/indexer/query imports)"
    verification:
      - kind: unit
        ref: "go build ./...; go vet ./internal/agents/...; go test ./internal/agents/... -race"
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 02: JSON-Config Agent Targets (Claude, Cursor, Gemini, Kiro, Antigravity) Summary

**5 self-registering AgentTarget implementations reproducing exact TS-parity paths and quirks — Claude's ./.mcp.json + legacy-migration + settings.json permissions, Cursor's --path arg, Antigravity's no-"type" entry, Gemini's project-root GEMINI.md, Kiro's disabled-by-default note — each proven install→uninstall byte-invariant under -race.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-12T18:57:13Z
- **Completed:** 2026-07-12T19:00:30Z
- **Tasks:** 3
- **Files modified:** 10 (5 new target files, 5 new test files)

## Accomplishments
- `claudeTarget`: global `~/.claude.json` / local `./.mcp.json` (never `./.claude.json`, Pitfall 3), `CLAUDE.md` marker instructions block, `AutoAllow`-gated idempotent `mcp__codegraph__*` settings.json permission, legacy `./.claude.json` codegraph-entry migration into `./.mcp.json` on install and strip-from-both on uninstall
- `cursorTarget`: `--path` quirk (local = absolute cwd via `os.Getwd()`+`EvalSymlinks`, global = literal `${workspaceFolder}` string), no instructions file, self-heal-delete of a legacy `.cursor/rules/codegraph.mdc`
- `antigravityTarget`: its own no-`type` entry builder (`antigravityEntry`, deliberately not routed through the shared `stdioMcpEntry`), global-only (`SupportsLocation(local)` false), unified-vs-legacy `~/.gemini/config` path resolution via a `.migrated` marker with stale-legacy-entry sweep, no instructions file of its own
- `geminiTarget`: `type:stdio` entry at `~/.gemini/settings.json` / `./.gemini/settings.json`, marker-fenced instructions at `~/.gemini/GEMINI.md` (global) but the **project root** `./GEMINI.md` for local (not `./.gemini/GEMINI.md`)
- `kiroTarget`: `type:stdio` entry at `~/.kiro/settings/mcp.json` / `./.kiro/settings/mcp.json`, no instructions file, self-heal-delete of a legacy `steering/codegraph.md` (both scopes), surfaces the "MCP disabled by default — enable in Settings" hint via `WriteResult.Notes` on every install
- Shared package-level helpers introduced once (in `claude.go`) and reused across all 5 targets: `fileExists`, `stdioMcpEntry`, `mcpEntryPresent`, `instructionsBody`

## Task Commits

Each task was committed atomically as RED (test) then GREEN (feat):

1. **Task 1: claude.go — the four-surface target** - `5ea0f68` (test, RED) → `a504d70` (feat, GREEN)
2. **Task 2: cursor.go + antigravity.go — the entry-shape quirks** - `36f45a9` (test, RED) → `2c8853b` (feat, GREEN)
3. **Task 3: gemini.go + kiro.go — project-root instructions + steering self-heal** - `051afdc` (test, RED) → `814d72f` (feat, GREEN)

## Files Created/Modified
- `internal/agents/claude.go` - Claude Code target + shared helpers (`fileExists`, `stdioMcpEntry`, `mcpEntryPresent`, `instructionsBody`) + settings.json permission helpers
- `internal/agents/cursor.go` - Cursor target with the `--path` quirk and legacy `.mdc` self-heal
- `internal/agents/gemini.go` - Gemini CLI target with project-root local `GEMINI.md`
- `internal/agents/kiro.go` - Kiro target with steering self-heal and disabled-by-default note
- `internal/agents/antigravity.go` - Antigravity target with its own no-type entry builder and unified/legacy path resolution
- `internal/agents/claude_test.go` - RED→GREEN coverage: global/local install, AutoAllow idempotency, legacy migration/strip, round-trip, Detect
- `internal/agents/cursor_test.go` - RED→GREEN coverage: `--path` local/global shapes, no-instructions, legacy self-heal, round-trip
- `internal/agents/gemini_test.go` - RED→GREEN coverage: project-root local instructions, `type:stdio`, round-trip
- `internal/agents/kiro_test.go` - RED→GREEN coverage: no-instructions, steering self-heal (both scopes), disabled-by-default note, round-trip
- `internal/agents/antigravity_test.go` - RED→GREEN coverage: no-type entry, global-only, legacy sweep, round-trip

## Decisions Made
- `stdioMcpEntry`/`fileExists`/`mcpEntryPresent`/`instructionsBody` live in `claude.go` (the first-written target file) rather than in 06-01's already-committed `shared.go`, since they're agent-target-shape helpers, not surgical-write primitives — Antigravity deliberately bypasses `stdioMcpEntry` with its own `antigravityEntry` builder (Pitfall 6)
- `instructionsBody()` recovers Claude/Gemini's instructions content by stripping `instructions.go`'s marker constants off `codegraphInstructionsBlock` at runtime rather than duplicating the block text in a second constant, guaranteeing the two can never drift apart (D-01a)
- Antigravity's `.migrated` marker is both read AND written by the Go port itself (not just a compatibility read of a marker some other tool wrote) — a fresh install writes directly to the unified path and creates the marker in the same call
- Cursor's local `--path` value is resolved via `filepath.EvalSymlinks(os.Getwd())` so it matches the canonicalized path a symlinked temp-dir test fixture reports (e.g. macOS `/tmp` → `/private/tmp`)

## Deviations from Plan

None - plan executed exactly as written, including the reconciliation note (only Claude + Gemini write an instructions file; Cursor/Kiro self-heal-delete a legacy one; Antigravity writes none).

## Issues Encountered
- Initial `claude_test.go` draft hit a Go composite-literal-in-if-condition parse ambiguity (`if claudeTarget{}.ID() != Claude {`) — fixed by assigning to a local variable first before the RED commit; not a deviation from plan behavior, just a test-authoring syntax fix made before the RED commit landed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 5 JSON-config agents (`claude`, `cursor`, `gemini`, `kiro`, `antigravity`) are registered and pass `go test ./internal/agents/... -race`
- `internal/cli/install.go`/`uninstall.go` (06-04) can now call `agents.ResolveTargetFlag` and iterate real, tested `AgentTarget` implementations for all 5 of these agents
- Remaining 06-03 targets (Codex, opencode, Hermes — TOML/JSONC/YAML surgery) are independent of this plan's work and unblocked
- No blockers

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 10 created source files (5 target .go files + 5 test files) and the SUMMARY.md itself found on disk; all 6 task/RED/GREEN commit hashes (5ea0f68, a504d70, 36f45a9, 2c8853b, 051afdc, 814d72f) found in git log.
