# Design

## Context

`InstallOptions.ExecPath` is set in two places: `internal/cli/install.go` from `os.Executable()`, and `refreshInstalledSkills` in `internal/cli/upgrade.go` from the path `upgrade.Run` swapped. Nine targets implement `AgentTarget.Install` and each writes the path into its MCP entry; Claude Code and Codex also render it into a PreToolUse guard, where `renderCodexPreToolGuard` already rejects an empty or relative path. The `internal/cli` tests drive `install` through the real command, so their `os.Executable()` is a `go-build` test binary; #133 pointed their home at a temporary directory, which is why that no longer damages a real config.

Codex loads hooks from `~/.codex/hooks.json` and `<repo>/.codex/hooks.json` (Codex docs, "Where Codex looks for hooks"). This repository's `.codex/` holds only `hooks/`, whose three files are `go:embed` sources listed in `codexassets.go`. `codexPreToolUseBlocks` decodes the fragment's `hooks` object and rewrites the one command literal; any other top-level key is ignored. Codex's hook runner (`codex-rs`, `build_command`) clears the child environment and replays the session's environment snapshot into it, filtering a short list of non-inheritable variables; `PATH` is not on that list. A registered hook whose command cannot be spawned is reported as a failed hook run on each tool call. The repository's guards avoid every PATH-dependent command on the strength of the opposite claim, and `TestHookPreToolUseCodex_LinkedWorktreeWithoutGitIsSilent` already shows the Codex hook firing in a linked worktree as soon as git is on `PATH`.

## Goals / Non-Goals

**Goals:**
- No caller can record an ephemeral path by accident, whichever CLI path reaches `Install`.
- One clear message from `codegraph install`, not nine.
- The Codex hook comments and the spec state what Codex's source does, with the source named.
- Nobody reads `.codex/hooks/hooks.json` as a live hook again.

**Non-Goals:**
- Validating that `ExecPath` exists or is executable. A durable path that is later removed is D-08's silent-exit case, unchanged.
- Rendering git's path into the Codex guard. The local hook command is itself `"$(git rev-parse --show-toplevel)/..."`, so git has to be on the hook's `PATH` before any guard runs.

## Decisions

### D1. The check is a method on `InstallOptions`, called by every target and once by the CLI

`func (o InstallOptions) Validate() error` in `internal/agents/types.go` returns a descriptive error when `ephemeralExecPath(o.ExecPath)` is true and `o.AllowEphemeralExecPath` is false. `ephemeralExecPath` is a pure function over the cleaned path: the base name has the suffix `.test`; any path element is `go-build` followed by one or more digits, the name Go gives its work directory (`go-build2384710239`); or any path element is 64 hexadecimal digits followed by `-d`, the directory in which the Go build cache keeps an executable that `go run` built and cached (`<GOCACHE>/69/6959…dc6-d/codegraph`, removed by `go clean -cache`). `go-builder` and the cache root's own name do not match. The cache rule is a shape test so that `install` never has to run `go env`. Each of the nine `Install` methods begins with the same guard: validate, and on error return a `WriteResult` whose `Errors` holds it, so the CLI's existing CR-01 rule (any error is surfaced and the exit is non-zero) applies. `newInstallCmd` calls `Validate` once right after `os.Executable()` and returns the error directly, so the user sees one line and no target runs. `refreshInstalledSkills` keeps its code path: its per-target error already prints a warning and the "run `codegraph install`" pointer.

Alternative: validate only in the CLI. Leaves `refreshInstalledSkills` and any future caller able to bypass it, which is the gap the issue names. Rejected.

Alternative: validate only inside the targets. Correct but prints the refusal once per selected agent. Rejected in favor of both.

### D2. The override is an environment variable mapped to an option field

`CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1` sets `InstallOptions.AllowEphemeralExecPath` in `newInstallCmd` and in `newUpgradeCmd` before the refresh. The agents layer never reads the environment itself, so `Install` stays a function of its options and the `internal/agents` tests set the field directly. `internal/cli`'s `TestMain` sets the variable beside the home backstop, because every command-level install test runs from a test binary and some (`TestInstall_NotesAreControlCharacterSanitized`, `TestInstall_PlainBranchPathsAreControlCharacterSanitized`) set their own home without `fakeHome`; one new test clears it with `t.Setenv` and asserts the refusal.

Alternative: an `--allow-ephemeral-binary` flag. Does not reach the upgrade refresh and is harder to apply across a test package. Rejected.

### D3. The PATH claim is corrected from Codex's source, not probed

The D-22 comments are rewritten to the source reading: Codex's hook runner (`codex-rs`, `build_command`, read 2026-10-20) clears the child environment and replays the session snapshot with `PATH` kept, and Codex's own hook documentation registers a project hook as `"$(git rev-parse --show-toplevel)/..."`, which cannot run at all without git on the hook's `PATH`. The guards keep avoiding `command -v`, `dirname` and `basename`, now as a robustness choice rather than a necessity, and the rewritten comment names the source. No live probe of Codex: a check whose subject is how Codex behaves is what the repository's third-party rule excludes, and if the reading were wrong the local hook command would fail loudly as a per-call failed hook run in Codex rather than silently. No `CODEGRAPH_GIT` fallback: it could not help the local form, whose command needs git before any guard runs.

### D4. A `description` on the fragment, no rename

Codex documents `description` as optional top-level metadata that does not change which hooks run, and `codexPreToolUseBlocks` reads only `hooks`, so the field is inert on both sides. The text says: this is codegraph's embedded Codex PreToolUse fragment; `codegraph install -t codex --pretool-nudge` rewrites its command into `<repo>/.codex/hooks.json` or `~/.codex/hooks.json` next to a rendered `codegraph-pretooluse.sh`; Codex does not read hooks from `.codex/hooks/`. Renaming the embed source would churn `codexassets.go`, the tests and the planning record for no behavior.

## Risks / Trade-offs

- [A developer runs `go run ./cmd/codegraph install` and is refused] → The message names the path and the override variable, and the fix is `task build` then running `install` from the built binary.
- [The nine guards drift] → `TestEveryTargetRefusesEphemeralExecPath` iterates the registry and asserts each `Install` returns the error and writes nothing, so a target that drops its guard fails the suite.
- [The source reading is wrong and Codex strips `PATH`] → The local hook command fails on every tool call with a visible failed-hook error, the fix is a one-line comment revert plus the old PATH condition in the spec, and the nudge is advisory in the meantime.
- [Changing the embedded templates marks every installed Codex guard as updated on the next install] → Expected and harmless; the guard's behavior is identical.
