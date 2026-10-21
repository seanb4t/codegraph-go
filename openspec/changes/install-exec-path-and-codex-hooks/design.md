# Design

## Context

`InstallOptions.ExecPath` is set in two places: `internal/cli/install.go` from `os.Executable()`, and `refreshInstalledSkills` in `internal/cli/upgrade.go` from the path `upgrade.Run` swapped. Nine targets implement `AgentTarget.Install` and each writes the path into its MCP entry; Claude Code and Codex also render it into a PreToolUse guard, where `renderCodexPreToolGuard` already rejects an empty or relative path. The `internal/cli` tests drive `install` through the real command, so their `os.Executable()` is a `go-build` test binary; #133 pointed their home at a temporary directory, which is why that no longer damages a real config.

Codex loads hooks from `~/.codex/hooks.json` and `<repo>/.codex/hooks.json` (Codex docs, "Where Codex looks for hooks"). This repository's `.codex/` holds only `hooks/`, whose three files are `go:embed` sources listed in `codexassets.go`. `codexPreToolUseBlocks` decodes the fragment's `hooks` object and rewrites the one command literal; any other top-level key is ignored. Codex's hook runner (`codex-rs`, `build_command`) clears the child environment and replays the session's environment snapshot into it, filtering a short list of non-inheritable variables; `PATH` is not on that list. A registered hook whose command cannot be spawned is reported as a failed hook run on each tool call. The repository's guards avoid every PATH-dependent command on the strength of the opposite claim, and `TestHookPreToolUseCodex_LinkedWorktreeWithoutGitIsSilent` already shows the Codex hook firing in a linked worktree as soon as git is on `PATH`.

## Goals / Non-Goals

**Goals:**
- No caller can record an ephemeral path by accident, whichever CLI path reaches `Install`.
- One clear message from `codegraph install`, not nine.
- The Codex hook comments and the spec state what Codex actually does, with a recorded observation behind them.
- Nobody reads `.codex/hooks/hooks.json` as a live hook again.

**Non-Goals:**
- Validating that `ExecPath` exists or is executable. A durable path that is later removed is D-08's silent-exit case, unchanged.
- Rendering git's path into the Codex guard unless the live check demands it.

## Decisions

### D1. The check is a method on `InstallOptions`, called by every target and once by the CLI

`func (o InstallOptions) Validate() error` in `internal/agents/types.go` returns a descriptive error when `ephemeralExecPath(o.ExecPath)` is true and `o.AllowEphemeralExecPath` is false. `ephemeralExecPath` is a pure function over the cleaned path: the base name has the suffix `.test`, or any path element has the prefix `go-build`. Each of the nine `Install` methods begins with the same guard: validate, and on error return a `WriteResult` whose `Errors` holds it, so the CLI's existing CR-01 rule (any error is surfaced and the exit is non-zero) applies. `newInstallCmd` calls `Validate` once right after `os.Executable()` and returns the error directly, so the user sees one line and no target runs. `refreshInstalledSkills` keeps its code path: its per-target error already prints a warning and the "run `codegraph install`" pointer.

Alternative: validate only in the CLI. Leaves `refreshInstalledSkills` and any future caller able to bypass it, which is the gap the issue names. Rejected.

Alternative: validate only inside the targets. Correct but prints the refusal once per selected agent. Rejected in favor of both.

### D2. The override is an environment variable mapped to an option field

`CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1` sets `InstallOptions.AllowEphemeralExecPath` in `newInstallCmd` and in `newUpgradeCmd` before the refresh. The agents layer never reads the environment itself, so `Install` stays a function of its options and the `internal/agents` tests set the field directly. The `internal/cli` helper that fakes the home (`fakeHome`) sets the variable with `t.Setenv`, because every command-level install test runs from a test binary; one new test leaves it unset and asserts the refusal.

Alternative: an `--allow-ephemeral-binary` flag. Does not reach the upgrade refresh and is harder to apply across a test package. Rejected.

### D3. The live Codex check comes first and decides one task

Task 1.1 runs Codex 0.163.0 with `CODEX_HOME` pointing at a scratch directory holding a `hooks.json` whose PreToolUse command writes `env` to a file, trusts the hook through `/hooks`, runs one shell tool call and records the resulting `PATH` under "Observation" below and in the memory store. The source reading above predicts an inherited `PATH`. On that result the change rewrites the D-22 comments (the guards still avoid PATH-dependent commands, now as a robustness choice rather than a necessity) and the spec delta applies as written. If `PATH` is empty, task 1.4 applies instead: `install` renders git's absolute path (`exec.LookPath("git")` at install time) into a `CODEGRAPH_GIT` environment variable the guard exports before running the binary, `gitmeta` honors `CODEGRAPH_GIT` ahead of its `LookPath` seam, and the delta still holds because the Codex hook then finds git without a `PATH`.

### D4. A `description` on the fragment, no rename

Codex documents `description` as optional top-level metadata that does not change which hooks run, and `codexPreToolUseBlocks` reads only `hooks`, so the field is inert on both sides. The text says: this is codegraph's embedded Codex PreToolUse fragment; `codegraph install -t codex --pretool-nudge` rewrites its command into `<repo>/.codex/hooks.json` or `~/.codex/hooks.json` next to a rendered `codegraph-pretooluse.sh`; Codex does not read hooks from `.codex/hooks/`. Renaming the embed source would churn `codexassets.go`, the tests and the planning record for no behavior.

### Observation

Filled in by task 1.1: Codex version, date, the hook's `PATH` value (or "empty"), and whether a hook whose command does not exist produced a per-call error.

## Risks / Trade-offs

- [A developer runs `go run ./cmd/codegraph install` and is refused] → The message names the path and the override variable, and the fix is `task build` then running `install` from the built binary.
- [The nine guards drift] → `TestEveryTargetRefusesEphemeralExecPath` iterates the registry and asserts each `Install` returns the error and writes nothing, so a target that drops its guard fails the suite.
- [The live check cannot be run (no Codex auth on the machine)] → Record that in the observation and keep the comments on the source reading, naming the Codex commit read. The spec delta does not depend on the answer.
- [Changing the embedded templates marks every installed Codex guard as updated on the next install] → Expected and harmless; the guard's behavior is identical.
