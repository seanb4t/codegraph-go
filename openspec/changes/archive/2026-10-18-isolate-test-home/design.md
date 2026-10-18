# Design

## Context

See proposal.md for why. The relevant code today:

- `newUpgradeCmd` (`internal/cli/upgrade.go`) calls `refreshInstalledSkillsFunc(target, …)` after a successful swap and without `--check`. `target` is `os.Executable()`. `refreshInstalledSkills` refreshes every Claude location that already has a codegraph skill manifest, so the leak fires only on a machine where codegraph is installed for Claude Code.
- Every agent target resolves its global paths through `os.UserHomeDir()`, except opencode (`XDG_CONFIG_HOME` first) and Hermes (`HERMES_HOME` first).
- Each package has its own `fakeHome(t)`. Both copies set the same three variables with `t.Setenv`. The `internal/cli` copy also `t.Chdir`s into a fresh project directory.
- Neither package has a `TestMain`. Neither package calls `t.Parallel()`.
- Subprocesses that the tests in these packages start: `git` (`runGit`, `runGitC`), the generated guard scripts, `/bin/sh -n`, and a re-exec of the test binary (`daemon_test.go`). None of them runs the `go` tool.

## Goals / Non-Goals

**Goals:**

- No test in `internal/cli` or `internal/agents` can write to the real home directory, even when it forgets `fakeHome`.
- The leaking test stops running a real install at all.

**Non-Goals:**

- `TestMain` does not change the working directory. Some tests read files by relative path, and local-scope isolation stays the job of `fakeHome`'s `t.Chdir`.
- `TestMain` does not replace `fakeHome`. Tests that inspect what was written to the home still need a fresh home of their own.

## Decisions

**Fix the cause and add a backstop.** The test fix alone leaves the next forgetful test free to leak. A `TestMain` alone would still let the test run a real install with a `cli.test` path, only into a throwaway home. That is harmless, but it is not what the test is for: it checks argument parsing. The other `upgrade` tests that reach a successful swap already stub the refresh. Doing both matches the two acceptance criteria in #73.

**One temporary home for the whole package run.** `TestMain` creates one directory with `os.MkdirTemp`, sets the variables with `os.Setenv`, runs `m.Run()`, removes the directory, then exits with the run's code. This is the shape the `TestMain` in `test/integration` already has. A fresh home per test stays `fakeHome`'s job. The two layers compose: `t.Setenv` records the value `TestMain` set and restores it when the test ends.

**One list of variables per package.** A small package-level helper, `homeEnv(home)`, returns `HOME`, `XDG_CONFIG_HOME` and `HERMES_HOME` for a home root. `fakeHome` sets each one with `t.Setenv`, and `TestMain` sets each one with `os.Setenv` and checks the error. A variable added for a new agent target then reaches both layers. There are two copies of the helper, one per package, as there are two copies of `fakeHome` today. The alternative, an exported `internal/testutil` package, would be importable from product code. It would also be the first package of its kind here, and a three-line list does not justify that.

**`TestMain` checks its own work.** After it sets the variables, `TestMain` stops the run if `os.UserHomeDir()` does not return the temporary home. This is the positive assertion that rule `84d1gfpywd` asks of a guard, and it lives inside the guard, so no test tests a test.

**The leaking test stubs the refresh as a no-op and calls `fakeHome(t)`.** It does not assert anything about the refresh. `TestUpgradeCommand_RefreshesConfiguredLocations` already pins that the refresh runs once with the resolved path.

## Risks / Trade-offs

- [The `git` helpers lose `~/.gitconfig`.] → `runGit` and `runGitC` already pass the identity, the default branch and `commit.gpgsign=false` with `-c`, so they need no global config. But both helpers call `t.Skipf` when `git` fails. A break would show up as new skips, not as failures. The tasks compare the list of skipped tests before and after the change.
- [A test depends on real home contents on purpose.] → None was found. If one exists, the full run of both packages fails, which shows it.
- [A future agent target reads a new variable, such as a `CODEX_HOME`.] → `TestMain` cannot detect that. The per-package helper is the one place to add it. Today `codex.go` reads only `os.UserHomeDir()`.
- [A subprocess inherits the fake home.] → This is intended. No subprocess in these packages runs the `go` tool, whose default caches live under `HOME`.

## Migration Plan

None. Test code only. Roll back by reverting the commit.
