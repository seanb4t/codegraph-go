# Agent Capabilities

This is the AGENT-14 per-harness capability table
(`.planning/phases/07-codex-parity/07-CONTEXT.md` D-27/D-28) — the human-readable half of
what each of the 8 roster agents receives at each scope: its MCP config file, instructions
file, skill package, and lifecycle-hook mechanism. The machine-readable half is the
`Capabilities()` literal inside each target's own file in `internal/agents/*.go`, and
`internal/agents/capability_doc_test.go` proves the two stay identical: every code-derived
column below (scope, MCP config, format, instructions, skill, hooks, nudge) is computed
straight from `Capabilities()` for all 8 targets × 2 scopes and compared against this table.
If this document and the code ever drift, `go test ./internal/agents/ -run
'TestCapabilityDoc'` fails — this table cannot silently overclaim what an agent actually gets.

## Legend

- `~` stands for your home directory (global scope). A relative path (no `~`) is relative to
  the repository root — a project-local install.
- `none` — this target declares nothing for that column; there is nothing to write or read.
- `not supported` — this target does not support that scope at all (e.g. Codex-era global-only
  agents at local scope); every code-derived column reads `not supported` and the verification
  column reads `n/a`.
- **Nudge** is derived from the **Hooks** column, not a separate capability: `claude-json` →
  `SessionStart + opt-in PreToolUse`; `codex-json` → `opt-in PreToolUse`; `none` → `none`.
- **Verification** is hand-kept, never code-derived, and always one of exactly three forms:
  `verified <date> (<evidence file>)` when a live session in this repository's `.planning/`
  history proved the row; `[ASSUMED] (<source>, fetched <date>)` when no live session ran and
  the row rests on the vendor's own published documentation; `n/a` for an unsupported scope.

## Capability Table

| Target | Scope | MCP config | Format | Instructions | Skill | Hooks | Nudge | Verification |
|---|---|---|---|---|---|---|---|---|
| `antigravity` | global | `~/.gemini/config/mcp_config.json` | json | none | `~/.gemini/config/skills/codegraph` | none | none | verified 2026-09-18 (05-LIVE-SESSIONS.md) |
| `antigravity` | local | not supported | not supported | not supported | not supported | not supported | not supported | n/a |
| `claude` | global | `~/.claude.json` | json | `~/.claude/CLAUDE.md` | `~/.claude/skills/codegraph` | claude-json | SessionStart + opt-in PreToolUse | [ASSUMED] (code.claude.com/docs/en/mcp, fetched 2026-09-19) |
| `claude` | local | `.mcp.json` | json | `.claude/CLAUDE.md` | `.claude/skills/codegraph` | claude-json | SessionStart + opt-in PreToolUse | verified 2026-09-19 (06-LIVE-SESSIONS.md) |
| `codex` | global | `~/.codex/config.toml` | toml | `~/.codex/AGENTS.md` | `~/.agents/skills/codegraph` | codex-json | opt-in PreToolUse | verified 2026-09-19 (07-LIVE-SESSIONS.md) |
| `codex` | local | `.codex/config.toml` | toml | `AGENTS.md` | `.agents/skills/codegraph` | codex-json | opt-in PreToolUse | verified 2026-09-19 (07-LIVE-SESSIONS.md) |
| `cursor` | global | `~/.cursor/mcp.json` | json | none | `~/.agents/skills/codegraph` | none | none | [ASSUMED] (cursor.com/docs/skills.md, fetched 2026-09-18) |
| `cursor` | local | `.cursor/mcp.json` | json | none | `.agents/skills/codegraph` | none | none | [ASSUMED] (cursor.com/docs/skills.md, fetched 2026-09-18) |
| `gemini` | global | `~/.gemini/settings.json` | json | `~/.gemini/GEMINI.md` | `~/.gemini/skills/codegraph` | none | none | [ASSUMED] (raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md, fetched 2026-09-18) |
| `gemini` | local | `.gemini/settings.json` | json | `GEMINI.md` | `.gemini/skills/codegraph` | none | none | [ASSUMED] (raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md, fetched 2026-09-18) |
| `hermes` | global | `~/.hermes/config.yaml` | yaml | none | none | none | none | [ASSUMED] (hermes-agent.ai/blog/hermes-mcp-integration-guide, fetched 2026-09-19) |
| `hermes` | local | not supported | not supported | not supported | not supported | not supported | not supported | n/a |
| `kiro` | global | `~/.kiro/settings/mcp.json` | json | none | `~/.kiro/skills/codegraph` | none | none | [ASSUMED] (kiro.dev/docs/skills/, fetched 2026-09-18) |
| `kiro` | local | `.kiro/settings/mcp.json` | json | none | `.kiro/skills/codegraph` | none | none | [ASSUMED] (kiro.dev/docs/skills/, fetched 2026-09-18) |
| `opencode` | global | `~/.config/opencode/opencode.jsonc` | jsonc | `~/.config/opencode/AGENTS.md` | `~/.agents/skills/codegraph` | none | none | [ASSUMED] (opencode.ai/docs/skills.md, fetched 2026-09-18) |
| `opencode` | local | `opencode.jsonc` | jsonc | `AGENTS.md` | `.agents/skills/codegraph` | none | none | verified 2026-09-18 (05-LIVE-SESSIONS.md) |

## Notes

- **Shared paths.** `.agents/skills/codegraph` (and its global counterpart `~/.agents/skills/codegraph`)
  is written by Cursor, opencode and Codex — one skill package, one manifest listing every agent
  that installed it, not three separate copies. The repo-root `AGENTS.md` is shared by Codex and
  opencode at local scope: it is kept until the LAST of the two registered targets still
  declaring it uninstalls, never removed just because one of them does.
- **Codex project config loads only for a trusted project.** Codex's `.codex/config.toml` (and,
  per the live D-16/A2 verdicts recorded in `07-LIVE-SESSIONS.md`, the codegraph skill and the
  repo-root `AGENTS.md` block) load only once the project is trusted, via Codex's own TUI trust
  prompt or a `trust_level = "trusted"` entry it writes to `~/.codex/config.toml`. codegraph
  never writes that trust entry itself.
- **Codex's PreToolUse nudge is opt-in and hook-trust-gated.** It is written only when `--pretool-nudge`
  is passed, is skipped (with a Note) when the user has explicitly set `[features] hooks = false`
  (or the deprecated `codex_hooks = false`), and — like any new or changed Codex hook — will not
  fire until it is reviewed and trusted in `/hooks`. If the nudge appears not to be firing after
  install, check `/hooks` directly: Codex gives no exec-side error or warning for a
  not-yet-trusted hook, only the interactive trust prompt's own option text.
- **The nudge's cooldown is per-session AND per-subagent for Codex, exactly parallel to Claude
  Code.** Codex's subagent `PreToolUse` stdin carries `agent_id` (a UUIDv7) and `agent_type`
  alongside the parent's `session_id`; the main thread carries neither
  (`07-LIVE-SESSIONS.md`, A4). This is the same granularity Claude Code already has — not a
  narrower, Codex-specific limitation.
- **codegraph always appends its hook group LAST** in `hooks.json`. Codex's hook trust is
  position-keyed (a group's trust hash is recorded against its array index), so removing a
  hook group that precedes a foreign one re-flags that foreign hook as "modified" even though
  its bytes never changed (`07-LIVE-SESSIONS.md`, D-23). Appending last means codegraph's own
  install/uninstall never shifts — and therefore never re-flags — anyone else's hook.
- **codegraph resolves Codex's global files under `~/.codex`.** A non-default `CODEX_HOME` is
  not honoured.
- **Codex's interactive skill listing may truncate the codegraph skill's description** (an
  advisory, not a `SKILL.md` change — the front-loaded trigger words survive either way);
  `codex debug prompt-input`'s raw dump showed the full, untruncated description
  (`07-LIVE-SESSIONS.md`, B7).
- **Hermes receives only the MCP entry** — no instructions file, no skill package.
- **Cursor, Gemini CLI and Kiro are `[ASSUMED]`** because no live session could run against them
  on this machine (no Cursor account; Gemini CLI and Kiro are not installed) — their rows rest
  on the vendors' own published documentation, cited above with a fetch date.

## See also

- `docs/LANGUAGE-CAPABILITY-MATRIX.md` — the equivalent drift-tested table for per-language
  extraction/resolution/dispatch/routing coverage.
