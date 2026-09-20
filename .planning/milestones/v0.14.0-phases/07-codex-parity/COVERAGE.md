# API Coverage — Codex CLI integration surface

> Full coverage by default. Opt-outs are explicit, reasoned decisions.

Phase 7 integrates codegraph with Codex CLI's configuration and hook surfaces (files Codex reads and the
PreToolUse hook protocol it invokes). Decisions follow 07-CONTEXT D-09…D-23 and its Deferred Ideas.

| capability | decision | reason |
|---|---|---|
| mcp_servers in the global ~/.codex/config.toml | INTEGRATE | |
| mcp_servers in the project .codex/config.toml | INTEGRATE | |
| AGENTS.md instructions (global ~/.codex/AGENTS.md) | INTEGRATE | |
| AGENTS.md instructions (repo root, shared with opencode) | INTEGRATE | |
| AGENTS.override.md | OPT-OUT | explicitly out of scope — a user-owned file codegraph never writes; install adds a Note when it shadows AGENTS.md (D-12) |
| skills in ~/.agents/skills and .agents/skills | INTEGRATE | |
| skills in .codex/skills and $CODEX_HOME/skills | OPT-OUT | not needed — Codex does not merge same-name skills, so a second copy would list twice (D-14); declared read-only only if the live check shows Codex reading them (D-15) |
| skill metadata agents/openai.yaml | OPT-OUT | not needed yet — deferred to v2 (07-CONTEXT Deferred Ideas) |
| hooks.json PreToolUse | INTEGRATE | |
| hooks.json SessionStart | OPT-OUT | explicitly out of scope — a Codex SessionStart nudge is deferred (07-CONTEXT Deferred Ideas) |
| other hook events (PostToolUse, Stop, Subagent*, PermissionRequest) | OPT-OUT | not needed — the milestone's nudge contract is PreToolUse only (NUDGE-03, CODEX-05) |
| hooks registered in a config.toml [hooks] table | OPT-OUT | not needed — hooks.json carries the same registration and keeps the hook out of the user's config.toml |
| project trust entry projects.<path>.trust_level | OPT-OUT | explicitly out of scope — trust is the user's decision; codegraph only prints how to grant it (D-10) |
| [features] hooks / codex_hooks flags | OPT-OUT | not needed — read-only: codegraph reads them to skip the hook write and never writes a feature flag (D-18) |
| plugin packaging and marketplace | OPT-OUT | not needed yet — deferred to v2 (07-CONTEXT Deferred Ideas) |
| codex mcp add CLI | OPT-OUT | not needed — codegraph writes config.toml directly through the fixed TOML splice |
| rules / exec policy | OPT-OUT | not needed — the nudge adds context and never gates a command |
