# Proposal: Move per-user state out of `~/.codegraph` and never index the home directory

Issue: #74

## Why

codegraph keeps its per-user daemon registry in `~/.codegraph/daemons/`. That directory has the same name as the per-repository index marker `.codegraph/`. The upward search for a project root (`query.ResolveCodegraphDir`) has no stop condition other than the filesystem root, so from any unindexed directory under the home directory it finds `~/.codegraph` and takes the whole home directory as the project. `codegraph serve --mcp` then runs its startup sync across the home directory. On macOS that fails on iCloud dataless files (`resource deadlock avoided`) or on TCC-protected folders (`operation not permitted`), and it reads private data along the way. `codegraph status` and every other query command resolve the same way, and `codegraph init`, `index` and `uninit` run at `$HOME` act on `~/.codegraph` itself. Issue #74 is `priority:high`.

## What Changes

- Per-user state moves to the XDG state directory: `$XDG_STATE_HOME/codegraph`, by default `~/.local/state/codegraph`. The daemon registry becomes `<state dir>/daemons/`. A relative `XDG_STATE_HOME` is ignored, as the XDG specification says.
- The XDG config directory is defined alongside it: `$XDG_CONFIG_HOME/codegraph`, by default `~/.config/codegraph`. No command writes a file there in this change. It is the one place a future user-level setting goes, so nothing ever lands under `~/.codegraph` again.
- One `internal/userdirs` package resolves both directories. Every other package asks it; none builds a home-relative path itself.
- Existing registry records migrate once: when the new `daemons/` directory does not exist yet and `~/.codegraph/daemons/` does, its `<pid>.json` files move to the new location, the old `daemons/` directory is removed, and `~/.codegraph` is removed only when it is then empty. Files codegraph did not write (for example a `telemetry.json` left by the TypeScript CodeGraph, or a `store/` this bug created) are left in place.
- The home directory is never an index root. `ResolveCodegraphDir` skips the candidate equal to `os.UserHomeDir()` and keeps walking, so a `~/.codegraph` of any origin no longer turns `$HOME` into a project. Indexes in and above the home directory's children keep resolving as before.
- `codegraph init`, `codegraph index` and `codegraph uninit` refuse to run with the home directory as their target and say why. They share `targetRoot`, so the check lives there.
- `codegraph serve --mcp` started in `$HOME`, or in an unindexed directory under it, answers `initialize` with zero tools and never opens `~/.codegraph/store`.
- The `codegraph daemon` help text names the registry location, and `docs/CLI-REFERENCE.md` is regenerated from it.

No CLI output shape or MCP tool result changes. The registry's record format is unchanged; only its directory moves.

## Capabilities

### New Capabilities

- `user-directories`: where codegraph keeps per-user state and per-user config, how `XDG_STATE_HOME` and `XDG_CONFIG_HOME` are honored, that the daemon registry lives in the state directory, and how records in the old location migrate.
- `index-resolution`: how a command finds the index root from its start directory, that the home directory is never an index root, and that the commands that create or remove an index refuse the home directory as a target.

### Modified Capabilities

- `mcp-server`: the requirement "No index means no tools (MCP-03)" gains a scenario for a server started in the home directory, or under it, while `~/.codegraph` exists.

## Out of scope

- A user-level config file. The config directory is defined, and nothing reads or writes it yet.
- A cache directory. `internal/corpora` already uses `XDG_CACHE_HOME` and keeps doing so.
- Deleting foreign files from `~/.codegraph`. A `telemetry.json` written by the TypeScript CodeGraph, or a `store/` this bug built, stays for the user to remove. The changelog line names the cleanup.
- Stopping the upward walk at the home directory. Only the home directory itself is skipped; a project at `/Users/<name>/code/repo` with its index at `/Users/<name>/code/.codegraph` still resolves.
- Platform-native directories (`~/Library/Application Support` on macOS). Both platforms use the XDG layout, as `internal/corpora` and `internal/cli/editordiscovery.go` already do for `XDG_CACHE_HOME` and `XDG_DATA_HOME`.
- The thousands of stale records a long-running machine collects in the registry. `daemon.List` already removes stale records on every call; the migration moves them and the next `List` prunes them.

## Impact

- New package `internal/userdirs` with its tests.
- `internal/daemon/registry.go`: `defaultRegistryDir` resolves through `userdirs`, plus the one-time migration.
- `internal/query/resolve.go`: `ResolveCodegraphDir` skips the home directory. `internal/query/engine_test.go` gains the cases.
- `internal/cli/init.go`: `targetRoot` refuses the home directory. `init`, `index` and `uninit` inherit the check.
- `internal/cli/daemon.go`: help text names the registry location; `task docs:cli` regenerates `docs/CLI-REFERENCE.md`.
- `internal/cli/serve_test.go`: the #74 regression test, using the fake home from #133.
- `internal/cli/install_test.go` and `internal/agents/testhelpers_test.go`: `homeEnv` gains `XDG_STATE_HOME`, so a developer's real `XDG_STATE_HOME` cannot leak into a test run.
- `test/tmux`: the daemon tests point `XDG_STATE_HOME` at the throwaway home and poll the new registry path.
- Changelog fragment of kind Fixes. The registry location was never documented, so this is not marked **BREAKING**; the fragment names the old and new paths.
- Users who hit the bug delete `~/.codegraph/store` by hand, as issue #74 says.
