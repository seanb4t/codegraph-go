---
status: testing
phase: 06-agent-integrations-cli-lifecycle
source: [06-VERIFICATION.md]
started: 2026-07-12T22:22:50Z
updated: 2026-07-12T22:22:50Z
---

## Current Test

number: 1
name: Live coding-agent MCP handshake (install → tool visible → uninstall → tool gone)
expected: |
  After `codegraph install --target <agent>`, opening the agent shows `codegraph_explore`
  as an available MCP tool and it returns results. After `codegraph uninstall`, the tool is
  gone and the agent's config file is byte-identical to its pre-install state (modulo the
  CodeGraph section). Verify on at least one real agent (Claude Code or Cursor recommended).
awaiting: user response

## Tests

### 1. Live coding-agent MCP handshake
expected: |
  Install into a real agent from the 8-agent roster (Claude Code, Cursor, Codex CLI, opencode,
  Gemini CLI, Hermes, Antigravity, Kiro). Confirm the MCP client loads `codegraph_explore` over
  stdio and it works, then confirm it disappears after `codegraph uninstall` with the config
  otherwise restored. Automated substitutes (scratch-$HOME install→uninstall round-trip +
  config-shape assertions) already pass; only a real MCP client process starting the server
  over stdio cannot be proven automatically.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
