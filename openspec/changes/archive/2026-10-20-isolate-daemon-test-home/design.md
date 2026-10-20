# Design

## Context

See proposal.md for why. The relevant code today:

- `internal/daemon/registry.go`: `registryDir` is a package-level func var defaulting to `defaultRegistryDir`, which calls `userdirs.StateDir()` (`XDG_STATE_HOME`, else `~/.local/state`) and `os.UserHomeDir()` for the legacy migration. `withRegistryDir(t)` in `registry_test.go` swaps the var for a `t.TempDir()`. Only `registry_test.go`, five tests in `daemon_test.go`, five in `stop_test.go` and one in `lock_test.go` use it; the other `Run`-driving tests in `daemon_test.go`, `soak_test.go`, `lock_test.go` and `watchdog_shape_test.go` reach the real registry.
- `internal/daemon/soak_test.go` holds the package's `TestMain`, which is `goleak.VerifyTestMain(m)`. goleak v1.3.0 runs `m.Run()`, checks for leaked goroutines, then calls `os.Exit` unless a `goleak.Cleanup(func(exitCode int))` option replaces it. The cleanup function receives the final exit code and must exit itself.
- `stop_test.go` and `lock_test.go` re-exec the test binary (`os.Args[0]`) as a helper process. The child inherits the environment.
- `registry_test.go` tests that call `defaultRegistryDir` set `HOME` and `XDG_STATE_HOME` with `t.Setenv`, which restores the value `TestMain` set when each test ends.
- `test/integration/main_test.go`: `TestMain` either takes a prebuilt binary from `CODEGRAPH_TEST_BIN` or runs `go build` into a temporary directory, then runs `m.Run()`. The `go` tool's default `GOCACHE` and `GOMODCACHE` live under `HOME`. Tests spawn the binary with `cmd.Env = os.Environ()` plus a few `CODEGRAPH_*` variables, and `git` through `runGitI`, which passes identity and `commit.gpgsign=false` with `-c` and skips on any failure. No test runs the `go` tool.
- `internal/cli/main_test.go` and `internal/agents/main_test.go` are the shape to copy: one `os.MkdirTemp` home, `os.Setenv` for each `homeEnv` entry, a check that `os.UserHomeDir()` returns that home, `m.Run()`, `os.RemoveAll`, `os.Exit`.

## Goals / Non-Goals

**Goals:**

- No test in `internal/daemon` or `test/integration` can write to the real home or the real `XDG_STATE_HOME`, even when it uses neither `withRegistryDir` nor a fake home of its own.
- The goleak gate in `internal/daemon` keeps working unchanged.

**Non-Goals:**

- `TestMain` does not change the working directory, and it does not replace `withRegistryDir`. A test that asserts on registry contents still needs a registry of its own.
- No shared test helper package. The reasoning from #133 stands: an importable package would be reachable from product code, and the helper is a few lines.

## Decisions

**Backstop only, no per-test retrofit.** Issue #136 asks for the `TestMain` and for a run against empty directories that leaves them empty. Retrofitting the seam-less tests would touch most of `daemon_test.go` for no further protection: with the backstop, those tests share one throwaway registry for the package run, which is the same sharing they have today against the real one, and the tests that care about registry contents already isolate it.

**One `TestMain` that composes goleak and the fake home.** The order inside it: create the home, set the variables, check `os.UserHomeDir()`, then `goleak.VerifyTestMain(m, goleak.Cleanup(func(code int) { _ = os.RemoveAll(home); os.Exit(code) }))`. goleak runs the leak check before it calls the cleanup, so the home still exists while any late goroutine is inspected, and the exit code goleak computed (including a leak turned into `1`) is the one the process exits with. The alternative, two `TestMain`s, is not possible in Go; wrapping goleak's `VerifyNone` by hand would lose its `TestMain` semantics.

**Move `TestMain` to `main_test.go`.** `soak_test.go` keeps `TestSoak` and `TestConvergenceTwoSessions`. A reader who wants the package gate then finds it where `internal/cli` and `internal/agents` keep theirs. The comment that explains the goleak gate (SYNC-06) moves with the function.

**Variables per package.** `internal/daemon` reads only `userdirs.StateDir()` and `os.UserHomeDir()`, so its `homeEnv` sets `HOME`, `XDG_STATE_HOME` and `XDG_CONFIG_HOME` (the two `userdirs` roots, so a future config read is covered too). `test/integration` runs the whole binary, including `install`, so its `homeEnv` sets the union of what the isolated packages set plus state: `HOME`, `XDG_CONFIG_HOME`, `XDG_STATE_HOME` and `HERMES_HOME`.

**Isolate after the build in `test/integration`.** `TestMain` keeps resolving or building `binPath` first, under the real environment, so `go build` sees the real `GOCACHE` and `GOMODCACHE`. Then it creates the fake home, sets the variables, checks `os.UserHomeDir()`, runs `m.Run()`, and removes both temporary directories. Both branches (prebuilt binary from `CODEGRAPH_TEST_BIN` and the local build) go through the same isolation step, so a CI job that passes a prebuilt binary is covered too.

**`TestPipedNeverHang` keeps its own `HOME` override.** It already uses a fresh `t.TempDir()` per subtest and asserts nothing about the home. `cmd.Env` appends to `os.Environ()`, which now carries the fake `XDG_*` values from `TestMain`, so the gap closes without editing the test.

**`TestMain` checks its own work.** After it sets the variables, each `TestMain` stops the run if `os.UserHomeDir()` does not return the temporary home. This is the positive assertion rule `84d1gfpywd` asks of a guard.

## Risks / Trade-offs

- [The goleak cleanup forgets to exit.] → Then every run exits `0`, including failing ones. The tasks prove exit-code propagation once with a throwaway failing test before the change is committed.
- [`git` under a fake home loses `~/.gitconfig`.] → `runGitI` passes everything it needs with `-c` and skips on failure, so a break would show as new skips. The tasks compare the skip list of `test/integration` before and after, as #133 did.
- [A daemon test depends on the real registry on purpose.] → None was found. The full package run shows it if one exists.
- [`TestConvergenceTwoSessions` fails.] → Known flake #114 (memory `jv02sbv356`), not caused by this change. Rerun before investigating.
- [`TestDaemonSharedWriter` times out when `task lint` runs concurrently.] → Memory `wme4kb6a2m`. Run `task test` and `task lint` sequentially.
- [A future command reads a new variable, such as `XDG_CACHE_HOME`.] → The per-package `homeEnv` is the one place to add it. Today the binary does not read `XDG_CACHE_HOME`; `internal/corpora` is not wired into the CLI.

## Migration Plan

None. Test code only. Roll back by reverting the commit.
