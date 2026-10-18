# Design

## Context

See proposal.md for why. The relevant code today:

- `query.ResolveCodegraphDir` (`internal/query/resolve.go`) walks `filepath.Dir` upward from `filepath.Abs(start)` and returns the first directory that holds a `.codegraph` directory. The only stop is the filesystem root. Every read path uses it: `query.OpenAt` for the CLI query commands, `serveServerPaths` in `internal/cli/serve.go`, the MCP server's per-request index check (`internal/mcp/server.go`), and the UI server's degrade and live-publish paths (`internal/uiserver`).
- `serveServerPaths` returns `hasIndex=true` for `$HOME`, and `newServeCmd` then calls `indexer.Sync($HOME, ~/.codegraph/store)`. `indexer.DiscoverAll` stops on the first `WalkDir` error, which is the `EDEADLK` or TCC failure in #74.
- `targetRoot` (`internal/cli/init.go`) gives `init`, `index` and `uninit` their root: the first argument made absolute, or the working directory. None of the three walks upward; each acts on `<root>/.codegraph` directly.
- `daemon.registryDir` (`internal/daemon/registry.go`) is a package-level func var, defaulting to `os.UserHomeDir() + /.codegraph/daemons`. `Register`, `Deregister` and `List` call it. Tests swap it for a `t.TempDir()`. `tools/mcpaudit` and `test/tmux` know the path by convention, not by import.
- `~/.codegraph/daemons/` is the only thing this binary writes under `~/.codegraph`. `telemetry.json` there predates the Go port (no commit in this repository ever named it), and `store/` is the bug's own output.
- `internal/corpora.CorpusRoot` already honors `XDG_CACHE_HOME` with a `~/.cache` fallback, and `internal/cli/editordiscovery.go` honors `XDG_DATA_HOME`. Both read the variable directly and fall back to the XDG default under `os.UserHomeDir()`, on macOS too.
- `internal/cli` and `internal/agents` run under a fake home (`TestMain` plus `homeEnv`, #133). `homeEnv` sets `HOME`, `XDG_CONFIG_HOME` and `HERMES_HOME`. The `test/tmux` tests pass `HOME` and `USERPROFILE` to the binary and poll `<home>/.codegraph/daemons`.

## Goals / Non-Goals

**Goals:**

- A `~/.codegraph` of any origin can no longer turn the home directory into a project, for every command that resolves an index.
- The binary writes nothing under `~/.codegraph`, and one package owns the per-user directory layout.
- Daemons started by the previous version stay visible to `codegraph daemon` after the upgrade.

**Non-Goals:**

- No per-user config file, no `--config` flag, no reader for the config directory. The directory is resolved, tested and documented, and that is all.
- No change to `XDG_CACHE_HOME` handling in `internal/corpora`.
- No change to the record format or to `List`'s stale-record pruning.

## Decisions

**Both fixes, not one.** Moving the registry removes the reason `~/.codegraph` exists, but not the directory on machines that already have it: the foreign `telemetry.json` and the bug's `store/` keep it alive, and the migration must not delete files it cannot account for. The resolver guard fixes #74 on those machines today. The move makes sure the guard is the only thing standing between a user and the bug on a fresh machine, instead of the first thing. Alternative considered: a store marker that `init` writes and the resolver requires. Rejected because `indexer.Sync` already built a complete-looking `~/.codegraph/store` on affected machines, and every existing index would need re-marking.

**Skip the home directory, do not stop at it.** The walk treats `os.UserHomeDir()` as one directory that can never match, and continues to the parent. Stopping would change behavior for a project whose index sits between the home directory and the filesystem root, which is not what #74 is about. The comparison is `filepath.Clean` on both sides, with no symlink evaluation: `os.UserHomeDir` returns `$HOME` as set, and `start` is already absolute, so this matches the way the rest of the resolver compares paths. If `os.UserHomeDir` fails, nothing is skipped and the walk is the one that exists today.

**The guard lives in `ResolveCodegraphDir`, and `targetRoot` gets its own.** The resolver is the single read seam, so one change covers `serve`, the MCP server's per-request check, every query command and the UI server. `init`, `index` and `uninit` do not walk, so they need the check at the one function the three share. `targetRoot` returns an error that names the directory; the three commands already return `targetRoot`'s error unchanged.

**One `internal/userdirs` package.** `StateDir()` and `ConfigDir()` each read their XDG variable, accept it only when `filepath.IsAbs` holds, and otherwise fall back to `~/.local/state` or `~/.config` under `os.UserHomeDir()`. Neither creates the directory; the writer does, as `fsatomic.WriteFile` already does for the registry. Alternative considered: `os.UserConfigDir()`. Rejected because it returns `~/Library/Application Support` on macOS, which contradicts how `internal/corpora` and `editordiscovery` already lay out XDG paths on this platform, and because the Go standard library has no state-directory equivalent. Alternative considered: a third-party XDG module. Rejected under the minimal-dependency constraint for two functions.

**`internal/daemon` depends on `internal/userdirs`, not the reverse.** `userdirs` imports only the standard library, so no package below it in the dependency order gains a new edge. `internal/query` does not need `userdirs`: the home directory it skips is `os.UserHomeDir()`, not an XDG path.

**Migration is in `defaultRegistryDir`, idempotent, and cheap when there is nothing to do.** `defaultRegistryDir` resolves the new path, calls `migrateLegacyRegistry(legacyDir, newDir)`, and returns the new path. The migration stats the new directory first and returns at once when it exists, so the steady-state cost is one stat per registry call, and `Register`, `Deregister` and `List` are called a handful of times per process. Records move by read, `fsatomic.WriteFile` to the new path, then remove, so a `XDG_STATE_HOME` on another filesystem works without a cross-device rename path. The old `daemons/` directory is removed with `os.Remove`, and `~/.codegraph` with `os.Remove` too: `os.Remove` on a non-empty directory fails, and that failure is ignored, which is exactly the "only when empty" rule with no listing of the directory's other contents. Tests call `migrateLegacyRegistry` with two `t.TempDir()` paths; the `registryDir` test seam is unchanged. Alternative considered: `List` reads both directories during a transition. Rejected because `daemon stop --all` and `Deregister` would need the same dual path, and there would be no point at which the old directory goes away.

**The regression test for #74 is in `internal/cli`, against `serveServerPaths`.** That function is the one `newServeCmd` calls, and `TestServeServerPathsNoIndex` already pins its no-index shape. The new test creates `<fake home>/.codegraph/daemons/` and asserts `hasIndex=false` for `<fake home>` and `<fake home>/x/repo`. It runs under the package's `TestMain` fake home, and sets `HOME` again with `t.Setenv` for clarity, so the test does not depend on which directory `TestMain` chose. The resolver's own cases go in `internal/query/engine_test.go` beside `TestResolveCodegraphDir`, with `t.Setenv("HOME", ...)`. The acceptance criterion "never opens `~/.codegraph/store`" is pinned by asserting that `<fake home>/.codegraph/store` does not exist after the call, which is a positive check that the directory was looked at.

**`homeEnv` gains `XDG_STATE_HOME`.** Without it, a developer whose shell sets `XDG_STATE_HOME` would have the daemon registry of a test run land in the real state directory even under the fake home. Both `internal/cli` and `internal/agents` get the entry, so the two lists stay the same shape. `test/tmux` sets `XDG_STATE_HOME=<home>/.local/state` next to `HOME` in every daemon test and polls `<home>/.local/state/codegraph/daemons`.

**The `daemon` help text names the location.** One sentence in `newDaemonCmd`'s `Long`, with the variable and the default. `task docs:cli` regenerates `docs/CLI-REFERENCE.md`, and the `docs:cli:drift` check keeps the two equal. No other document in the repository names `~/.codegraph/daemons`.

## Risks / Trade-offs

- [A user has a project rooted at `$HOME`, with `codegraph init` run there] → That setup never worked: the index and the registry shared `~/.codegraph`, and every query from under the home directory resolved to it. After this change `init` says so instead of creating the collision. The changelog line names it.
- [`XDG_STATE_HOME` is set on a filesystem that is unavailable at startup] → `fsatomic.WriteFile` and `os.ReadDir` return the error, and `Register` and `List` already surface errors to their callers. The daemon does not start silently without a record.
- [Two processes migrate at the same time] → Each record is written atomically and removed after. A record both processes move is written twice with the same content and removed once, and the second remove is a no-op `IsNotExist`. The final `os.Remove` of the old directories fails for one of them and is ignored. No record is lost.
- [`os.UserHomeDir` returns a path that differs from the one `start` was built from, for example through a symlinked home] → The guard does not fire, and behavior is today's. The migration to the state directory means a fresh machine has no `~/.codegraph` to hit, so the guard is the backstop, not the fix.
- [The daemon records of a previous version keep working across the migration, but a daemon of the previous version that is still running after the upgrade re-registers nothing] → It registered at start, and the migration moved its record. Its `Deregister` at exit removes a file from the old path that no longer exists, which `Deregister` treats as success. The record then sits in the new directory until `List`'s stale check removes it.
- [The config directory is resolved but unused] → `ConfigDir` is an exported function with a test and no production caller. The linters allow that for exported identifiers. It is the contract for the next user-level setting and costs a few lines.

## Migration Plan

1. Ship in one release. The first `Register`, `Deregister` or `List` after the upgrade moves the records. No step for the user.
2. Users who hit #74 delete `~/.codegraph/store` by hand. The migration does not touch it, because the bug's store is indistinguishable from a file another tool owns.
3. Rollback is installing the previous version. It recreates `~/.codegraph/daemons/` on the next daemon start and does not see the records in the state directory; those are pruned by nothing and stay as small JSON files until deleted by hand.
