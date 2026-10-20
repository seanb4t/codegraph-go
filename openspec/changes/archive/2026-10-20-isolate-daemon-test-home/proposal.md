# Proposal: Keep the daemon and integration tests out of the real home directory

Issue: #136

## Why

`go test ./internal/daemon` writes daemon records into the developer's real per-user registry. Every test that drives `Run`, `Register` or `Deregister` without the `withRegistryDir` seam resolves the registry through `defaultRegistryDir`, which reads `XDG_STATE_HOME` and the real home. On the maintainer's machine that left 934 stale records under `~/.codegraph/daemons`, which the move to `$XDG_STATE_HOME/codegraph/daemons` (#137) then migrated and refilled. The package has a `TestMain`, but it only runs the goleak check; nothing points the home elsewhere.

A survey for the same gap found one more test package that reaches the real home. `test/integration` spawns the real binary with `os.Environ()`. `TestLiveEditAutoSyncReachesExplore` runs `serve --mcp` in an indexed main checkout with the watcher enabled, so the in-process daemon registers in the real registry and creates `$XDG_STATE_HOME/codegraph/daemons` if it is missing. `TestPipedNeverHang` fakes `HOME` but not `XDG_STATE_HOME` or `XDG_CONFIG_HOME`, so on a machine that exports either one, its `daemon` run prunes the real registry and its `install` run can write an agent config under the real config directory.

## What Changes

- `internal/daemon` gets the same backstop `internal/cli` and `internal/agents` got in #133: before any test runs, `TestMain` points `HOME`, `XDG_STATE_HOME` and `XDG_CONFIG_HOME` at a temporary directory, stops the run if `os.UserHomeDir()` does not resolve there, and removes the directory after the run. The existing goleak gate stays; the two compose in one `TestMain`, moved from `soak_test.go` to a `main_test.go` like the other packages.
- `test/integration` gets the same backstop. Its `TestMain` already builds the binary before `m.Run()`; the home moves only after that build, so the `go` tool keeps its real build and module caches. Every spawned `codegraph` and `git` inherits the fake home.
- In each package, `TestMain` takes the list of variables from one `homeEnv` helper, as the two isolated packages do today.
- No product code changes. No CLI output, MCP result or installed file changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change alters only test code, so it sets `skip_specs: true`.

## Out of scope

- Pointing every seam-less daemon test at `withRegistryDir` one by one. The backstop makes the whole package safe; the per-test seam stays for tests that assert on registry contents. Tests that share the throwaway registry share it the way they share the real one today.
- `test/wireoracle`. Every scenario sets `CODEGRAPH_NO_WATCH=1` and runs only `init`, `sync` and `serve`, so the daemon returns before it registers and nothing reads the home. Its `TestMain` has the same shape as the integration one and can take the same backstop later if a scenario ever needs the home.
- `test/tmux`, `internal/query`, `internal/userdirs` and `internal/corpora`. They already fake `HOME` and the `XDG_*` variables in every test that reads them, and the reads are read-only.
- Cleaning a machine that already holds stale records. `codegraph daemon` prunes records whose process is gone on its next run, as the issue says.

## Impact

- Test code only: a new `internal/daemon/main_test.go`, `internal/daemon/soak_test.go` (the goleak `TestMain` moves out), a `homeEnv` helper in `internal/daemon`, and `test/integration/main_test.go`.
- No changelog fragment. The PR body carries a `changelog-exempt` marker.
