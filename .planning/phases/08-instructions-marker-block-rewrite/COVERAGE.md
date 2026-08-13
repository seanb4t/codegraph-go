# Phase 8 — API Coverage Decision Checkpoint

**Detector:** `api-coverage.cjs --json` over the Phase 8 roadmap scope
**Result:** `{"detected": true, "signals": [{"verb": "wire", "noun": "mcp", "snippet": "…on the MCP wire and in the marker block it writes into agent…"}]}`

The checkpoint fired. It is honoured here rather than dismissed, per the gate's own rule (act on `detected`, do not pattern-match the prose yourself).

## What the detection actually is

The signal is the phrase **"on the MCP wire"** — the verb `wire` adjacent to the noun `mcp`. Phase 8 integrates **no external API, SDK, or service**: `go.mod` and `go.sum` are untouched, no HTTP client is added, no credential is introduced, and no third-party endpoint is called. The only API surface in scope is **codegraph's own MCP server response surface**, which this repository implements and already ships.

That surface is nonetheless a real API contract with real consumers (eight agent clients), so the matrix below is filled in against it rather than skipped.

## Coverage Matrix — codegraph's own MCP server surface

INTEGRATE by default. Every OPT-OUT carries a one-line reason.

| # | Surface | Where it lives | Disposition | Reason (opt-outs only) |
|---|---------|----------------|-------------|------------------------|
| 1 | `initialize` result → `instructions` | `ServerOptions.Instructions`, `internal/mcp/server.go:537` | **INTEGRATE** | — rewritten in plan 08-01, proven on the wire by `TestInstructionsReachesTheWireVerbatim` |
| 2 | `server/discover` result → `instructions` | same SDK field, same const | **INTEGRATE** | — reached through the identical field; covered by the `modern-discover-explore` transcript in plan 08-03 |
| 3 | `resources/list` | `internal/mcp/resources.go` (`resourceURIFor`) | **INTEGRATE** | — named generically by the rewritten const and proven resolvable by a live round-trip in plan 08-01 |
| 4 | `resources/read` | `internal/mcp/resources.go` | **INTEGRATE** | — read on a real advertised URI inside `resourcesClaimResolves` |
| 5 | `tools/list` | `internal/mcp/tools.go` | **OPT-OUT** | Already described by the three pre-existing anchors (`default`, `CODEGRAPH_MCP_TOOLS`, `codegraph init`), all preserved verbatim by D-02; no new integration work. |
| 6 | `tools/call` | `internal/mcp/tools.go` | **OPT-OUT** | Per-tool descriptions are the documented home for call-level guidance; the MCP server-instructions guidance this phase follows explicitly forbids restating them (D-01). |
| 7 | The 10 individual `codegraph://` URIs | `resourceURIFor` map | **OPT-OUT** | D-03 (locked): the const points at `resources/list` generically. Client-side discovery is the standard MCP pattern, and enumerating the URIs inside a 600-byte wire-budgeted literal would create ten more literals to keep from drifting. |
| 8 | Skill package (`.claude/skills/codegraph/SKILL.md`) | root `claudeassets` embed | **INTEGRATE** | — named by the const (scoped to Claude Code) and proven resolvable against `claudeassets.SkillMarkdown()` in plan 08-01. |
| 9 | Hooks package (`SessionStart` nudge) | root `claudeassets` embed | **OPT-OUT** | Not a capability a client discovers over the wire; it fires in the agent runtime. Naming it in `instructions` would spend wire budget on something no MCP client can act on. |
| 10 | `notifications/tools/list_changed` | `internal/mcp/server.go` middleware | **OPT-OUT** | The const already states tools appear "with no client restart required", which is the agent-actionable half. The notification mechanism itself is SDK-level plumbing. |
| 11 | External APIs / SDKs / third-party services | — | **N/A** | None exist in this phase. `go.mod` and `go.sum` are unchanged; the Package Legitimacy Gate is therefore not applicable (08-RESEARCH.md, "Standard Stack"). |

## Disposition

Rows 1-4 and 8 are integrated by plans 08-01 through 08-03. Rows 5-7 and 9-10 are deliberate opt-outs backed by a locked decision (D-01, D-02, D-03) or by a byte-budget constraint. Row 11 records that the external-API branch of this checkpoint is genuinely empty.
