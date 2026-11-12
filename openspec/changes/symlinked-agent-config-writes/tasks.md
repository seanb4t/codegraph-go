# Tasks

## 1. Write through a link in fsatomic

- [ ] 1.1 Add `fsatomic.WriteFileThroughLink(path, content, within string) error` with the step-by-step link resolution of design D2 and the real-path check of design D3. Leave `WriteFile` unchanged. Verify with `go test ./internal/fsatomic/`.
- [ ] 1.2 Add scenario-shaped tests in `internal/fsatomic/fsatomic_test.go`, one per case: `TestWriteFileThroughLink_KeepsLinkAndWritesTarget`, `_FollowsAChainOfLinks`, `_CreatesAMissingTarget`, `_PreservesTargetMode`, `_LoopIsAnErrorAndWritesNothing`, `_TargetOutsideWithinReplacesTheLink`, `_TargetOutsideThroughALinkedDirectoryReplacesTheLink`, `_TargetInsideWithinIsWritten`. Each test that expects no write compares the outside file byte for byte. Watch `_KeepsLinkAndWritesTarget` fail against `WriteFile` before 1.1 lands, and record the output for the pull request.
- [ ] 1.3 Add `TestWriteFile_ReplacesASymlink` to hold the unchanged contract of `WriteFile`, which the spec requirement "Files that codegraph owns replace a symlink" depends on. Verify it passes before and after 1.1.
- [ ] 1.4 Update the package comment of `internal/fsatomic/fsatomic.go` to name both entry points and their callers. Verify by reading `go doc ./internal/fsatomic`.

## 2. Carry the link policy through internal/agents

- [ ] 2.1 Add `linkPolicy`, `linkPolicyFor(loc Location)` and the leaf `writeUserFile(path, content string, p linkPolicy) error` in `internal/agents/shared.go` (design D4). Cover `linkPolicyFor` with `TestLinkPolicyFor_GlobalAcceptsAnyTarget` and `TestLinkPolicyFor_LocalIsConfinedToTheWorkingDirectory`.
- [ ] 2.2 Add the policy parameter to the helpers that edit a user's file (design D5): `writeJSONFile`, `writeMcpEntry`, `removeMcpEntry`, `writeHookEntry`, `removeHookEntry`, `replaceOrAppendMarkedSection`, `removeMarkedSection`, `upsertInstructionsEntry`, and the direct configuration writes in `codex.go`, `hermes.go` and `opencode.go`. Each target passes `linkPolicyFor(loc)`. Keep `atomicWriteFile` and `atomicWriteExecutableFile` for owned files. Verify that `go build ./...` passes and that `go test ./internal/agents/ ./internal/cli/` passes with no test changed except for the new argument.
- [ ] 2.3 Add install-level tests in `internal/agents`, one per spec scenario, through real targets with `fakeHome` and `t.Chdir`: `TestInstall_GlobalConfigSymlinkIsWrittenThrough` (claude, `~/.claude.json`), `TestInstall_GlobalConfigSymlinkChain` (gemini), `TestInstall_GlobalConfigSymlinkToMissingTarget`, `TestUninstall_GlobalConfigSymlinkIsKept`, `TestInstall_GlobalConfigSymlinkKeepsTargetMode`, `TestInstall_LocalInstructionSymlinkInsideProjectIsWrittenThrough`, `TestInstall_LocalInstructionSymlinkOutsideProjectIsReplaced`, `TestInstall_LocalConfigSymlinkOutsideProjectIsReplaced`, `TestInstall_LocalSymlinkOutsideThroughLinkedDirectoryIsReplaced`, `TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing`, `TestInstall_GuardScriptSymlinkIsReplaced`. Each "replaced" test asserts that the outside file is byte for byte as it was and that the project path is a regular file. Watch `TestInstall_GlobalConfigSymlinkIsWrittenThrough` fail before 2.2 and record the output.
- [ ] 2.4 Add one CLI test in `internal/cli` that runs `install --location global` over a symlink loop and asserts a non-zero exit and an error line that names the path. Name it `TestInstall_SymlinkLoopExitsNonZero`.

## 3. Documentation

- [ ] 3.1 Add the local-scope rule to the threat model in `SECURITY.md`: a local install does not write to a file outside the project through a symlink, and it replaces such a link. Verify that the text matches the spec requirement "A local agent file is written through a symlink only inside the project".
- [ ] 3.2 Regenerate `docs/CLI-REFERENCE.md` only if the help text of `install` changes, and verify with `task docs:cli:drift`.

## 4. Integration

- [ ] 4.1 Run `task test`, `task lint` and `openspec validate symlinked-agent-config-writes --strict --no-interactive`. All three pass.
- [ ] 4.2 Add the changelog fragment: `task changie -- new --interactive=false -k Fixes -b "codegraph install keeps a symlinked agent configuration file as a symlink and writes to the file it leads to, except for a link in a project that leads outside the project."`

## Workflow follow-up

- Run `/sb:openspec-review symlinked-agent-config-writes implementation` before the implementation pull request.
- Archive the change with `/opsx:archive` in the implementation pull request.
