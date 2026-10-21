# Proposal: Refuse an ephemeral binary path at install, and settle the Codex hook facts

Issue: #134, #140, #142

## Why

`codegraph install` and the skill refresh after `codegraph upgrade` write whatever `os.Executable()` returns into every agent's MCP entry and PreToolUse guard. From a Go test binary or a `go run` build that is a path under a `go-build` temporary directory that disappears minutes later, after which the agent cannot start the server. #73 broke Claude Code's MCP entry this way; #133 closed the leak on the test side, but the product still accepts the path (#134).

Two Codex hook questions have been open since Phase 7 and were raised again by the worktree work in #139. The guard templates say Codex runs hooks with an empty `PATH`, which would keep the Codex nudge silent in every linked worktree; nobody has verified it, and Codex's hook runner source replays the session environment into the hook, `PATH` included (#140). And the tracked `.codex/hooks/hooks.json` names a rendered script that is not in the repository, which looked like a broken dogfood hook (#142). It is not: Codex loads project hooks from `<repo>/.codex/hooks.json`, and the tracked file at `.codex/hooks/hooks.json` is the `go:embed` fragment that `install` rewrites. Nothing in this repository registers it with Codex.

## What Changes

- `InstallOptions` gains validation in the agents layer: an `ExecPath` whose base name ends in `.test`, or that has a path element named `go-build` plus digits (a Go work directory), is ephemeral. Every target's `Install` refuses it with an error naming the path and writes nothing, and `codegraph install` exits non-zero with one message before any target runs. The upgrade refresh surfaces the same error through its existing warning. `InstallOptions.AllowEphemeralExecPath`, set from the environment variable `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1`, is the explicit override.
- The D-22 comments in both guard templates and in `internal/agents/codex_pretooluse.go` are rewritten to what Codex's hook runner source does: the hook inherits the session `PATH`. The comment names the source; no live probe of Codex is run, because a check of third-party behavior is outside what this repository verifies, and a wrong reading would surface loudly as a failed hook run on every tool call. The `agent-nudges` spec drops the "when git is on its PATH" condition for the Codex hook, so a linked worktree of an indexed main checkout gets the nudge from both harnesses; silence when the inherited `PATH` holds no git, or git cannot run, stays the exception.
- The embedded Codex fragment gets a top-level `description` saying it is codegraph's embedded template, that `install` rewrites it into `<repo>/.codex/hooks.json` or `~/.codex/hooks.json`, and that Codex does not read it from this path. Codex documents `description` as metadata that changes nothing. The `codexassets.go` comment that calls the directory "dogfooded" is corrected.

## Capabilities

### New Capabilities

- `agent-install`: what `codegraph install` and the upgrade refresh record for an agent, starting with the binary path they are allowed to record.

### Modified Capabilities

- `agent-nudges`: the requirement "A nudge fires only in an indexed project directory" loses the Codex-specific `PATH` condition in its linked-worktree scenario, and the "Codex hook with no git on its PATH" scenario is reworded to the inherited session `PATH`.

## Out of scope

- A `--exec-path` flag to install for a different binary than the one running. Build the binary and run `install` from it.
- Making the guards tolerate a moved binary. D-08 keeps that behavior: a missing binary exits silently.
- Registering a real dogfood Codex hook for this repository. Anyone who wants one runs `codegraph install -t codex --pretool-nudge`.
- Changing what any target renders beyond the shared path check.

## Impact

- `internal/agents/types.go` (the option and its validation), each target's `Install` (one guard at the top), `internal/cli/install.go` (the up-front check and the environment variable), `internal/cli/upgrade.go` (the option on the refresh; its error reporting already covers the refusal).
- `.codex/hooks/hooks.json`, `.codex/hooks/codegraph-pretooluse-local.sh`, `.codex/hooks/codegraph-pretooluse-global.sh`: embedded sources, so the rendered guard bytes change and the next `install` reports them updated.
- `openspec/specs/agent-nudges/spec.md` through the delta.
- Tests in `internal/agents` and `internal/cli`; `internal/cli`'s `TestMain` sets the override, because the test binary is ephemeral by definition.
- `README.md` and `docs/CLI-REFERENCE.md` where `install` is described.
- Changelog fragment, kind Fixes.
