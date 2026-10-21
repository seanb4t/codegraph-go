# Tasks

## 1. Codex hook facts

- [ ] 1.1 Run the live check from design D3 (scratch `CODEX_HOME`, a trusted PreToolUse hook that dumps `env`, one shell tool call, plus one registered hook whose command does not exist) and write the observation into `design.md` under "Observation"; verify the section names the Codex version, the date, the `PATH` value and the missing-command behavior.
- [ ] 1.2 Rewrite the D-22 paragraph in `.codex/hooks/codegraph-pretooluse-local.sh`, `.codex/hooks/codegraph-pretooluse-global.sh` and the matching comments in `internal/agents/codex_pretooluse.go` to the verified fact; verify `go test ./internal/agents -run Codex` passes and `codegraph install -t codex --pretool-nudge` in a scratch home renders a guard that still contains no `command -v`, `dirname` or `basename`.
- [ ] 1.3 Add the `description` from design D4 to `.codex/hooks/hooks.json` and correct the "dogfoods" comment in `codexassets.go`; verify with `TestCodexFragmentDescriptionIsNotWritten` that a local and a global install write a `hooks.json` without the `description` key and with the same PreToolUse blocks as before.
- [ ] 1.4 Only if 1.1 found an empty `PATH`: render git's absolute path into the Codex guards as `CODEGRAPH_GIT`, honor it in `internal/gitmeta` ahead of the `LookPath` seam, and verify with `TestCodexHookNudgesLinkedWorktreeWithoutPath` that `hook pretooluse --harness codex` run with an empty `PATH` and `CODEGRAPH_GIT` set gives the nudge in a linked worktree of an indexed main checkout. When 1.1 found an inherited `PATH`, mark this done with the note "not needed: PATH inherited".

## 2. Spec coverage for the Codex linked-worktree nudge

- [ ] 2.1 Verify the modified `agent-nudges` scenario: `TestHookPreToolUseCodex_LinkedWorktreeWithoutGitIsSilent` already asserts the Codex hook fires in a linked worktree of an indexed main checkout with git on `PATH` and stays silent without git; rename or split it so one test name states the firing case (`TestHookPreToolUseCodex_LinkedWorktreeFires`) and run `go test ./internal/cli -run 'TestHookPreToolUseCodex_LinkedWorktree'`.

## 3. Ephemeral ExecPath guard

- [ ] 3.1 Add `AllowEphemeralExecPath` to `InstallOptions`, `ephemeralExecPath(path) bool` and `(InstallOptions) Validate() error` in `internal/agents/types.go`; verify with `TestEphemeralExecPath` (table: `/usr/local/bin/codegraph` false, `<tmp>/go-build1/b001/cli.test` true, `<tmp>/go-build1/exe/codegraph` true, `/opt/go-builder/codegraph` false, `/home/x/foo.test` true) and `TestInstallOptionsValidate` (override set → nil).
- [ ] 3.2 Call `opts.Validate()` at the top of every target's `Install` (claude, codex, cursor, gemini, opencode, hermes, antigravity, kiro and any other roster target) returning a `WriteResult` with the error; verify with `TestEveryTargetRefusesEphemeralExecPath` in `internal/agents`, which iterates the registry in a fake home and asserts each target returns the error and leaves the home empty.
- [ ] 3.3 In `internal/cli/install.go`, read `CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH`, set the option, and return `Validate()`'s error before any target runs; in `internal/cli/upgrade.go`, set the option on the refresh. Set the variable in the `internal/cli` helper `fakeHome` with `t.Setenv`. Verify with `TestInstall_RefusesEphemeralExecPath` (helper used, then the variable unset again: exit non-zero, one error line naming the path, no `.claude.json` written) and that `TestInstall_ExecPathAppearsInWrittenConfig` still passes under the override.
- [ ] 3.4 Verify the refresh path with `TestRefreshInstalledSkills_RefusesEphemeralExecPath` in `internal/cli/upgrade_test.go`: a configured Claude location, an ephemeral path, no override → the location's config is unchanged and the returned error names the path.
- [ ] 3.5 Document the refusal and the override in the `install` command's `Long` help, `README.md` and `docs/CLI-REFERENCE.md`; verify `codegraph install --help` shows the variable.

## 4. Suite and changelog

- [ ] 4.1 Run `task test` then `task lint` sequentially and verify both pass.
- [ ] 4.2 Add the fragment: `task changie -- new -k Fixes -b "codegraph install and the upgrade refresh refuse to record a Go test or go run binary as the agent's command; CODEGRAPH_ALLOW_EPHEMERAL_EXEC_PATH=1 overrides" --interactive=false` and verify the file appears under `.changes/unreleased/`.
