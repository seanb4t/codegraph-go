# Tasks

## 1. Codex hook facts

- [x] 1.1 Rewrite the D-22 paragraph in `.codex/hooks/codegraph-pretooluse-local.sh`, `.codex/hooks/codegraph-pretooluse-global.sh` and the matching comments in `internal/agents/codex_pretooluse.go` to the source reading in design D3 (inherited `PATH`; avoiding PATH-dependent commands is a robustness choice), naming the source; verify `go test ./internal/agents -run Codex` passes and `codegraph install -t codex --pretool-nudge` in a scratch home renders a guard that still contains no `command -v`, `dirname` or `basename`.
- [x] 1.2 Add the `description` from design D4 to `.codex/hooks/hooks.json` and correct the "dogfoods" comment in `codexassets.go`; verify with `TestCodexFragmentDescriptionIsNotWritten` that a local and a global install write a `hooks.json` without the `description` key and with the same PreToolUse blocks as before.

## 2. Spec coverage for the Codex linked-worktree nudge

- [x] 2.1 Verify the modified `agent-nudges` scenario: `TestHookPreToolUseCodex_LinkedWorktreeWithoutGitIsSilent` already asserts the Codex hook fires in a linked worktree of an indexed main checkout with git on `PATH` and stays silent without git; rename or split it so one test name states the firing case (`TestHookPreToolUseCodex_LinkedWorktreeFires`) and run `go test ./internal/cli -run 'TestHookPreToolUseCodex_LinkedWorktree'`.

## 3. Ephemeral ExecPath guard

- [x] 3.1 Add `AllowEphemeralExecPath` to `InstallOptions`, `ephemeralExecPath(path) bool` and `(InstallOptions) Validate() error` in `internal/agents/types.go`; verify with `TestEphemeralExecPath` (table: `/usr/local/bin/codegraph` false, `<tmp>/go-build1/b001/cli.test` true, `<tmp>/go-build1/exe/codegraph` true, `/opt/go-builder/codegraph` false, `/home/x/foo.test` true, `<cache>/69/<64 hex digits>-d/codegraph` true, `<cache>/69/<64 hex digits>/codegraph` false) and `TestInstallOptionsValidate` (override set → nil).
- [x] 3.2 Call `opts.Validate()` at the top of every target's `Install` (claude, codex, cursor, gemini, opencode, hermes, antigravity, kiro and any other roster target) returning a `WriteResult` with the error; verify with `TestEveryTargetRefusesEphemeralExecPath` in `internal/agents`, which iterates the registry in a fake home and asserts each target returns the error and leaves the home empty.
- [x] 3.3 In `internal/cli/install.go`, read `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH`, set the option, and return `Validate()`'s error before any target runs; in `internal/cli/upgrade.go`, set the option on the refresh. Set the variable in `internal/cli`'s `TestMain` next to `isolateHome`. Verify with `TestInstall_RefusesEphemeralExecPath` (`t.Setenv("CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH", "")`: exit non-zero, one error line naming the path, no `.claude.json` written), that `TestInstall_NotesAreControlCharacterSanitized` and `TestInstall_PlainBranchPathsAreControlCharacterSanitized` still pass unchanged, and that `TestInstall_ExecPathAppearsInWrittenConfig` still passes under the override.
- [x] 3.4 Verify the refresh path with `TestRefreshInstalledSkills_RefusesEphemeralExecPath` in `internal/cli/upgrade_test.go`: a configured Claude location, an ephemeral path, no override → the location's config is unchanged and the returned error names the path.
- [x] 3.5 Document the refusal and the override in the `install` command's `Long` help, `README.md` and `docs/CLI-REFERENCE.md`; verify `codegraph install --help` shows the variable.

## 4. Suite and changelog

- [x] 4.1 Run `task test` then `task lint` sequentially and verify both pass.
- [x] 4.2 Add the fragment: `task changie -- new -k Fixes -b "codegraph install and the upgrade refresh refuse to record a Go test or go run binary as the agent's command; CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1 overrides" --interactive=false` and verify the file appears under `.changes/unreleased/`.
