# Tasks

Run every target as `mise exec -- task <target>` in a shell without `mise activate`. Run `task test` and `task lint` one after the other, not at the same time.

## 1. The path function in `internal/agents`

- [x] 1.1 Add `StableExecPath(path string) (recorded string, versioned bool)` next to `ephemeralExecPath` in `internal/agents/types.go`, as design.md describes. It finds the last `installs/<tool>/<version>` or `Caskroom/<token>/<version>` run of elements. A `<version>` of `latest` is not versioned. It accepts a candidate only if `os.SameFile` holds for `os.Stat` of both paths. Add the table test `TestStableExecPath` to `internal/agents/types_test.go`. It builds each layout under `t.TempDir()` with a regular file and real relative links. The cases and their results:
  - mise `latest` to the same version: the link, not versioned.
  - `latest` to another version: unchanged, versioned.
  - The asdf layout with no `latest`: unchanged, versioned.
  - A broken `latest` link: unchanged, versioned.
  - A cask with its `bin` link: the link, not versioned.
  - A cask without the link: unchanged, versioned.
  - A path that is already `latest`: unchanged, not versioned.
  - `/usr/local/bin/codegraph`: unchanged, not versioned.
  - An `installs` element with only one element after it: unchanged, not versioned.

  Verify `go test ./internal/agents -run TestStableExecPath -count=1` passes. If you replace the `os.SameFile` call with `true`, the test must fail.

## 2. `codegraph install`

- [x] 2.1 Add the package var `executablePath = os.Executable` to `internal/cli/install.go` and use it in place of the direct call. Apply `agents.StableExecPath` to its result before `InstallOptions` is built, so `Validate` checks the recorded path. If `versioned` is true, write the warning line from design.md once to `cmd.ErrOrStderr()` before the target loop. Add `TestInstall_RecordsMiseLatestLink` (the seam returns `<tmp>/installs/codegraph/0.16.3/codegraph` and `latest` links to `./0.16.3`. The opencode entry from `--target opencode --location global` names the `latest` path, and stderr is empty) and `TestInstall_WarnsOnceForVersionedPath` (an asdf layout with no `latest` and `--target all --location global`. Every written entry names the real path, stderr holds exactly one line with the warning and the path, and the command exits zero). Both use `fakeHome`. Verify `go test ./internal/cli -run 'TestInstall_(RecordsMiseLatestLink|WarnsOnceForVersionedPath|ExecPathAppearsInWrittenConfig|RefusesEphemeralExecPath)' -count=1` passes.
- [x] 2.2 Update the paragraph in the install command's `Long` help that says install records the absolute path of the binary. The new text says that a mise or Homebrew cask install records the stable link. It also says that a versioned path with no stable link gets one warning. Run `task docs:cli` to regenerate `docs/CLI-REFERENCE.md`. Verify `task docs:cli:drift` passes.

## 3. The refresh after `codegraph upgrade`

- [x] 3.1 In `refreshInstalledSkills` (`internal/cli/upgrade.go`), apply `agents.StableExecPath` to `execPath` before the install loop and ignore `versioned`. The self-replace target in `newUpgradeCmd` keeps the real path. Add `TestRefreshInstalledSkills_RecordsMiseLatestLink`: install for Claude with `fakeHome`, then call `refreshInstalledSkills` with a versioned path whose `latest` links to the same version. `.claude.json` names the `latest` path. Verify `go test ./internal/cli -run TestRefreshInstalledSkills -count=1` passes.

## 4. Whole-suite checks and the changelog

- [x] 4.1 Run `task test` and then `task lint`. Verify both pass. Report any failure that is already on `main` apart from this change.
- [x] 4.2 Add the changelog fragment: `task changie -- new --interactive=false -k Fixes -b "codegraph install now records the mise latest link or the Homebrew bin link in place of a versioned path, so agent configurations keep working after a package-manager upgrade."` Verify the new file is under `.changes/unreleased/`.

## Workflow follow-up

- The implementation PR says `Closes #231` and carries the code, the fragment and the `openspec archive stable-package-manager-exec-path` result.
- After the release, run `codegraph install --target opencode --location global` once on Sean's machine, because `opencode.json` is not managed by chezmoi and still names the 0.15.0 path.
