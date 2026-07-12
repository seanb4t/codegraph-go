---
phase: 06-agent-integrations-cli-lifecycle
plan: 03
subsystem: infra
tags: [installer, agent-integration, toml, jsonc, yaml, hujson, tdd, go-registry, codex, opencode, hermes]

# Dependency graph
requires:
  - phase: 06-agent-integrations-cli-lifecycle
    provides: "internal/agents foundation (06-01): AgentTarget interface, atomicWriteFile/upsertInstructionsEntry/removeMarkedSection, registerTarget/registry, hujson pinned unimported"
  - phase: 06-agent-integrations-cli-lifecycle
    provides: "claude.go shared helpers (06-02): fileExists, instructionsBody, stdioMcpEntry"
provides:
  - "3 self-registering AgentTarget implementations: codexTarget, opencodeTarget, hermesTarget — completing the full 8-agent roster"
  - "toml.go: hand-rolled spliceTOMLTable/stripTOMLTable/tomlString/tomlStringArray (mirrors TS toml.ts, no TOML dependency)"
  - "First real (imported) use of github.com/tailscale/hujson — Parse/Patch/Format/Pack comment-preserving JSONC edit"
  - "yamlBlockRange/yamlListBlockRange/yamlRemoveRange: hand-rolled indent-aware YAML block/list-item line-range surgery (no YAML dependency)"
affects: [06-04, 06-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-rolled single-block text-splice editors (toml.go, hermes.go's yaml* helpers) mirror the TOML approach: locate a header line, splice/replace/remove the byte range through the next same-or-shallower boundary or EOF, collapse blank lines symmetrically on removal"
    - "hujson Parse -> build RFC-6902 patch ops conditionally (skip ops for already-correct keys) -> Patch -> Format -> Pack is the one comment-preserving JSONC edit path in this package; existing values are compared via hujson.Standardize + jsonDeepEqual (shared.go) before deciding whether an op is needed, keeping re-runs byte-idempotent"
    - "yamlListBlockRange (indent >= parent, not indent > parent) is a distinct primitive from yamlBlockRange (indent <= parent ends the block) specifically because PyYAML's default block-sequence style puts list items at the SAME indent as their parent key — reusing the map-child boundary rule for a list would truncate the list at its first item (Pitfall 5)"

key-files:
  created:
    - internal/agents/toml.go
    - internal/agents/codex.go
    - internal/agents/opencode.go
    - internal/agents/hermes.go
    - internal/agents/toml_test.go
    - internal/agents/codex_test.go
    - internal/agents/opencode_test.go
    - internal/agents/hermes_test.go
  modified: []

key-decisions:
  - "opencode's mcp.codegraph entry existence/equality check standardizes the current hujson sub-value (hujson.Standardize strips comments -> plain JSON) before comparing against the desired entry via the existing shared.go jsonDeepEqual/normalizeJSON helpers, rather than writing new opencode-specific comparison logic"
  - "opencode's \\$schema value (https://opencode.ai/config.json) and entry shape ({type:\"local\", command:[...], enabled:true}) were confirmed against opencode's own current docs (Context7 /websites/opencode_ai) rather than assumed, since RESEARCH.md didn't pin the literal schema URL"
  - "Hermes' YAML command value is double-quoted via toml.go's existing tomlString() helper (same backslash/quote escaping rules apply to a YAML double-quoted scalar) instead of writing a near-duplicate yamlString() function"
  - "Hermes cli-toolset removal is a simple global first-match line delete (trimmed line == \"- mcp-codegraph\") rather than mirroring the block-range-based append logic — sufficient because the append side only ever produces that one exact line, and it keeps the inverse operation trivially correct regardless of indent"

patterns-established:
  - "Codex and Hermes are both global-only, single-file-format hand-rolled surgical editors (TOML/YAML) with no library dependency, mirroring the TS parity oracle's own toml.ts/hermes.ts reasoning"
  - "opencode is the only target combining hujson comment-preserving JSONC edits with an XDG-only (never %APPDATA%) config-dir resolver and a self-heal sweep for a legacy platform-specific path"
  - "Hermes is the only target with zero instructions-file surface — DescribePaths returns exactly one path (the config file), and no AGENTS.md/CLAUDE.md-equivalent is ever written"

requirements-completed: [AGNT-01, AGNT-02, AGNT-03]

coverage:
  - id: D1
    description: "toml.go: spliceTOMLTable appends when absent (preserving unrelated tables byte-for-byte), replaces only the codegraph block when content differs, is a byte-identical no-op when identical; stripTOMLTable is the exact inverse (round-trip byte-invariant) and a no-op when the table is absent"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/agents/toml_test.go (TestSpliceTOMLTable_*, TestStripTOMLTable_*)"
        status: pass
    human_judgment: false
  - id: D2
    description: "codexTarget: global-only (SupportsLocation(local) false, Install/Uninstall no-op at local), ~/.codex/config.toml [mcp_servers.codegraph] table + ~/.codex/AGENTS.md marker block, round-trip byte-invariant, idempotent re-run"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/agents/codex_test.go (TestCodex_*)"
        status: pass
    human_judgment: false
  - id: D3
    description: "opencodeTarget: hujson comment-preserving Parse->Patch->Format->Pack edit of opencode.jsonc/.json (comments survive install + idempotent re-run), combined [binary,...args] command array, XDG_CONFIG_HOME/~/.config resolution with no runtime.GOOS branch, %APPDATA% stale-entry self-heal sweep guarded so it never touches an XDG-matching real config"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/agents/opencode_test.go (TestOpencode_*)"
        status: pass
    human_judgment: false
  - id: D4
    description: "hermesTarget: global-only, $HERMES_HOME/config.yaml (default ~/.hermes) mcp_servers.codegraph block splice/strip preserving unrelated top-level keys and sibling servers, platform_toolsets.cli append matching the ACTUAL existing list-item indent (both PyYAML-default parent-indent and hand-authored deeper-indent fixtures covered), no duplicate on re-run, round-trip byte-invariant, no instructions file written"
    requirement: "AGNT-03"
    verification:
      - kind: unit
        ref: "internal/agents/hermes_test.go (TestHermes_*)"
        status: pass
    human_judgment: false
  - id: D5
    description: "All 8 roster agents (claude/cursor/codex/opencode/hermes/gemini/antigravity/kiro) now self-registered via init(); internal/agents remains boundary-clean (no graphstore/indexer/query imports); hujson moves from unimported-pin (06-01) to a real, compiled, imported dependency"
    verification:
      - kind: unit
        ref: "go build ./...; go vet ./internal/agents/...; go test ./internal/agents/... -race"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 03: Non-JSON Agent Targets (Codex TOML, opencode JSONC, Hermes YAML) Summary

**Hand-rolled TOML/YAML line-range surgery plus a comment-preserving hujson JSONC edit complete the 8-agent roster — Codex's global-only `[mcp_servers.codegraph]` table, opencode's XDG-resolved `mcp.codegraph` with survives-a-comment-audit round trips, and Hermes' `platform_toolsets.cli` append that detects PyYAML's real (parent-indent, not 4-space) list style instead of assuming one.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-12T19:11:41Z
- **Completed:** 2026-07-12T19:20:42Z
- **Tasks:** 3
- **Files modified:** 8 (4 new target/utility files, 4 new test files)

## Accomplishments
- `toml.go`: `spliceTOMLTable`/`stripTOMLTable` — a ~140-line hand-rolled single-table TOML editor (find `[tableName]` header, splice to the next top-level `[...]` header or EOF) mirroring TS's own `toml.ts` reasoning ("~50KB dependency for ~6 lines of output"); no TOML library added
- `codexTarget`: global-only (`SupportsLocation(local)` false), edits `~/.codex/config.toml`'s `[mcp_servers.codegraph]` table via `toml.go`, upserts `~/.codex/AGENTS.md`'s marker block — one of only 4 of 8 targets with an instructions file
- `opencodeTarget`: the package's first real (imported) use of `github.com/tailscale/hujson` — `Parse` -> conditional RFC-6902 patch ops -> `Format` -> `Pack` preserves every comment in the user's `opencode.jsonc` through install/uninstall/idempotent re-run; `mcp.codegraph.command` is a combined `[binary, ...args]` array (opencode's own convention, confirmed against opencode's current docs); config-dir resolution is `XDG_CONFIG_HOME` or `~/.config` on every OS with **no** `runtime.GOOS` branch (Pitfall 4), plus a guarded `%APPDATA%` stale-entry self-heal sweep
- `hermesTarget`: global-only, `$HERMES_HOME/config.yaml` (default `~/.hermes`); `yamlBlockRange`/`yamlListBlockRange`/`yamlRemoveRange` hand-rolled YAML line-range primitives splice a `mcp_servers.codegraph` block (preserving sibling servers and unrelated top-level keys) and append `mcp-codegraph` to `platform_toolsets.cli` at the list's **actual detected indent** — tested against both a PyYAML-default fixture (list items at the same indent as `cli:`) and a hand-authored deeper-indent fixture (Pitfall 5); writes **no** instructions file (Hermes has none in the TS source)
- All 8 roster agents now registered: `AllTargetIDs()` returns claude/cursor/codex/opencode/hermes/gemini/antigravity/kiro

## Task Commits

Each task was committed atomically as RED (test) then GREEN (feat):

1. **Task 1: toml.go + codex.go** - `21a28c5` (test, RED) -> `4b9f286` (feat, GREEN)
2. **Task 2: opencode.go (hujson)** - `1fdc5ac` (test, RED) -> `12cac0a` (feat, GREEN)
3. **Task 3: hermes.go** - `a2af11d` (test, RED) -> `1938a60` (feat, GREEN)

## Files Created/Modified
- `internal/agents/toml.go` - hand-rolled `[mcp_servers.codegraph]` TOML table splice/strip + `tomlString`/`tomlStringArray` escaping helpers
- `internal/agents/codex.go` - Codex CLI target: global-only, `config.toml` table + `AGENTS.md` marker block
- `internal/agents/opencode.go` - opencode target: hujson comment-preserving JSONC edit, XDG resolver, `%APPDATA%` sweep, `AGENTS.md`
- `internal/agents/hermes.go` - Hermes target: YAML line-range surgery (`mcp_servers.codegraph` + `platform_toolsets.cli`), no instructions file
- `internal/agents/toml_test.go` - RED->GREEN coverage: append/replace/no-op/strip/round-trip
- `internal/agents/codex_test.go` - RED->GREEN coverage: global-only, unrelated-table preservation, round-trip, idempotent re-run
- `internal/agents/opencode_test.go` - RED->GREEN coverage: comment preservation, combined command array, XDG resolution, `.json`-over-`.jsonc` preference, `%APPDATA%` sweep (both triggered and guarded-skip)
- `internal/agents/hermes_test.go` - RED->GREEN coverage: PyYAML-default + deeper-indent cli fixtures, idempotent append, round-trip, no-instructions-file assertion

## Decisions Made
- opencode's entry-equality check reuses `shared.go`'s `jsonDeepEqual`/`normalizeJSON` against a `hujson.Standardize`-stripped current value, rather than writing opencode-specific comparison logic
- opencode's `$schema` value and entry shape were verified against opencode's own current docs (Context7 `/websites/opencode_ai`) since RESEARCH.md didn't pin the literal schema URL string
- Hermes' YAML command scalar reuses `toml.go`'s `tomlString()` escaping helper (same backslash/quote rules apply to a YAML double-quoted scalar) instead of a duplicate YAML-specific string function
- Hermes cli-toolset removal is a simple global first-match line delete rather than mirroring the block-range append logic — correct and simpler because the append side only ever emits that one exact line

## Deviations from Plan

None - plan executed exactly as written, including the reconciliation note (Hermes has no instructions-file surface; only Codex and opencode among this plan's three targets write one).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 8 roster agents (`claude`, `cursor`, `codex`, `opencode`, `hermes`, `gemini`, `antigravity`, `kiro`) are registered and pass `go test ./internal/agents/... -race`
- `internal/cli/install.go`/`uninstall.go` (06-04) can now call `agents.ResolveTargetFlag` and iterate real, tested `AgentTarget` implementations for the full roster
- No blockers

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 8 created source files (4 target/utility .go files + 4 test files) and the SUMMARY.md itself found on disk; all 6 task/RED/GREEN commit hashes (21a28c5, 4b9f286, 1fdc5ac, 12cac0a, a2af11d, 1938a60) found in git log.
