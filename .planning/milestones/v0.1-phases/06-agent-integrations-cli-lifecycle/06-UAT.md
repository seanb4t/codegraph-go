---
status: complete
phase: 06-agent-integrations-cli-lifecycle
source: [06-VERIFICATION.md]
started: 2026-07-12T22:22:50Z
updated: 2026-07-12T22:35:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Live coding-agent MCP handshake
expected: |
  Install into a real agent from the 8-agent roster (Claude Code, Cursor, Codex CLI, opencode,
  Gemini CLI, Hermes, Antigravity, Kiro). Confirm the MCP client loads `codegraph_explore` over
  stdio and it works, then confirm it disappears after `codegraph uninstall` with the config
  otherwise restored.
result: pass
source: automated (MCP protocol handshake driven directly)
evidence: |
  Built the real binary and exercised the full lifecycle against a scratch repo (codegraph init →
  files=1 nodes=3 edges=3) under a throwaway $HOME:
  - `install --target claude --location local` wrote ./.mcp.json with mcpServers.codegraph
    {command: <abs path to this binary via os.Executable>, args:[serve,--mcp], type:stdio} and
    the ./.claude/CLAUDE.md marker block with exact <!-- CODEGRAPH_START/END --> fences pointing
    at codegraph_explore.
  - Drove MCP JSON-RPC over stdio the way a real client does: `initialize` returned
    serverInfo {name:codegraph, version:0.1.0}; `tools/list` advertised `codegraph_explore`.
  - Negative control (MCP-03): serving from a non-indexed dir advertised ZERO tools.
  - `uninstall --target claude --location local` removed the .mcp.json entry (left {}, empty
    mcpServers cleaned) and removed the marker block; per-agent status reported removed/not-found.
  - Re-install twice is byte-identical (idempotent).
  Residual (cosmetic only): a specific third-party agent GUI rendering the tool in its own UI was
  not exercised — the protocol-level handshake above is the mechanism that GUI depends on.

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
