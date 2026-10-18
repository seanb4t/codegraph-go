# Tasks

Run Go with `GOTOOLCHAIN=go1.26.9`, the version `go.mod` names. Never run a reproduction against the real home: each one points `HOME` at a directory from `mktemp -d` and also exports `XDG_STATE_HOME="$HOME/.local/state"`. When `HOME` moves, also export the real `GOPATH`, `GOMODCACHE`, `GOCACHE` and `GOENV` from `go env`, or `go test` starts from an empty module cache and build cache.

## 1. Baseline

- [x] 1.1 Reproduce #74 in a scratch home with the released binary: create `$HOME/.codegraph/daemons`, run `git init -q "$HOME/x/repo"`, then from `$HOME/x/repo` pipe the `initialize` request from issue #74 into `codegraph serve --mcp`. Verify that `codegraph status` from the same directory names `$HOME` as the root, or that `serve` fails or creates `$HOME/.codegraph/store`. Keep the scratch home for 8.2.

## 2. The `internal/userdirs` package

- [x] 2.1 Add `internal/userdirs` with `StateDir()` and `ConfigDir()`: each returns `<XDG variable>/codegraph` when the variable is set to an absolute path, and `<os.UserHomeDir()>/.local/state/codegraph` or `<os.UserHomeDir()>/.config/codegraph` otherwise. Neither creates a directory. The package imports only the standard library. Verify `go vet ./internal/userdirs` passes.
- [x] 2.2 Add `internal/userdirs/userdirs_test.go` with `TestStateDir` and `TestConfigDir`, each a table over unset, empty, absolute and relative values of the variable under a `t.Setenv("HOME", t.TempDir())`. Verify `go test ./internal/userdirs -count=1` passes and that the relative case asserts the fallback path, not the variable.

## 3. The home directory is never an index root

- [x] 3.1 In `internal/query/resolve.go`, make `ResolveCodegraphDir` read `os.UserHomeDir()` once before the walk and skip the candidate whose `filepath.Clean` equals the cleaned home directory, continuing to the parent. On a `UserHomeDir` error, skip nothing. Update the doc comment to say what is skipped and why. Verify `go build ./...` passes.
- [x] 3.2 In `internal/query/engine_test.go`, extend `TestResolveCodegraphDir` with subtests under `t.Setenv("HOME", home)` where `home/.codegraph/daemons` exists: `home/x/repo` returns `ErrNotInitialized`; `home` returns `ErrNotInitialized`; `home/x/repo/pkg` with `home/x/repo/.codegraph` resolves to `home/x/repo`. Verify `go test ./internal/query -run TestResolveCodegraphDir -count=1` passes and that the first subtest fails when the skip in 3.1 is commented out.

## 4. Index commands refuse the home directory

- [x] 4.1 In `internal/cli/init.go`, make `targetRoot` compare its result with `os.UserHomeDir()` after `filepath.Clean` on both and return an error naming the directory and saying the home directory cannot be a codegraph project. `init`, `index` and `uninit` need no change of their own. Verify `go build ./...` passes.
- [x] 4.2 Add `TestInitRefusesHomeDirectory` and `TestUninitRefusesHomeDirectory` in `internal/cli`, each under `fakeHome(t)`: run the command with the fake home as its argument, assert a non-zero result whose message names the home directory, and assert that `<home>/.codegraph` was not created (init) or still holds the file the test put there (uninit, with `--force`). Add `TestInitUnderHomeDirectoryProceeds` for `<home>/x/repo`. Verify `go test ./internal/cli -run 'RefusesHomeDirectory|UnderHomeDirectory' -count=1` passes.

## 5. The daemon registry moves to the state directory

- [x] 5.1 In `internal/daemon/registry.go`, make `defaultRegistryDir` return `userdirs.StateDir() + /daemons` and call `migrateLegacyRegistry(<home>/.codegraph/daemons, <new dir>)` before returning. Write `migrateLegacyRegistry(legacy, dir string) error`: return nil when `dir` exists or `legacy` does not; otherwise for each `*.json` in `legacy` read it, `fsatomic.WriteFile` it to `dir`, and remove the original; then `os.Remove(legacy)` and `os.Remove(filepath.Dir(legacy))`, ignoring the errors of both removes. Update the doc comments that name `~/.codegraph/daemons`. Verify `go build ./...` passes.
- [x] 5.2 Add to `internal/daemon/registry_test.go`: `TestDefaultRegistryDirUsesStateDir` under `t.Setenv` of `HOME` and `XDG_STATE_HOME`; `TestMigrateLegacyRegistryMovesRecords` (two records move, `legacy` and its parent are gone); `TestMigrateLegacyRegistryKeepsForeignFiles` (parent holds another file, records move, the parent and that file stay); `TestMigrateLegacyRegistryNoopWhenNewDirExists` (nothing moves or is deleted); `TestMigrateLegacyRegistryNoopWithoutLegacy`. Each test asserts the exact set of files afterwards on both sides. Verify `go test ./internal/daemon -run 'DefaultRegistryDir|MigrateLegacyRegistry' -count=1` passes.
- [x] 5.3 In `internal/cli/install_test.go` and `internal/agents/testhelpers_test.go`, add `"XDG_STATE_HOME": filepath.Join(home, ".local", "state")` to `homeEnv`. Verify `go test ./internal/cli ./internal/agents -count=1` passes with `XDG_STATE_HOME` exported to a directory outside the repository, and that the directory stays empty afterwards.

## 6. The #74 regression test for `serve`

- [x] 6.1 In `internal/cli/serve_test.go`, add `TestServeServerPathsIgnoresHomeCodegraph`: under `fakeHome(t)`, create `<home>/.codegraph/daemons`, then call `serveServerPaths(home)` and `serveServerPaths(<home>/x/repo)`; assert `hasIndex` is false, `repoPath` equals the start directory, and `<home>/.codegraph/store` does not exist afterwards. Verify `go test ./internal/cli -run TestServeServerPaths -count=1` passes and that the test fails with the 3.1 skip commented out.

## 7. Terminal tests and help text

- [ ] 7.1 In `test/tmux/daemon_seed.go`, add `XDG_STATE_HOME=<home>/.local/state` to `seedRunningDaemon`'s environment and make `waitForRegistryRecord` poll `<home>/.local/state/codegraph/daemons`. In `test/tmux/daemon_picker_test.go` and `test/tmux/daemon_empty_test.go`, add the same variable to the `env HOME=… USERPROFILE=…` prefix. Verify `task test:tmux` passes (it runs `go test -tags tmux ./test/tmux/...` and needs tmux installed).
- [x] 7.2 In `internal/cli/daemon.go`, add one sentence to `newDaemonCmd`'s `Long` naming the registry location as `$XDG_STATE_HOME/codegraph/daemons`, by default `~/.local/state/codegraph/daemons`. Run `task docs:cli`. Verify `task docs:cli:drift` passes and `git diff --stat docs/CLI-REFERENCE.md` shows only the `daemon` section changed.

## 8. Integration checks

- [x] 8.1 Run `GOTOOLCHAIN=go1.26.9 task test` and `GOTOOLCHAIN=go1.26.9 task lint`. Verify both pass.
- [x] 8.2 Rerun 1.1 against the built binary in the same scratch home. Verify that `serve --mcp` answers `initialize` with `tools=0` on its stderr session line, that `codegraph status` from `$HOME/x/repo` reports not initialized, that `$HOME/.codegraph/store` was not created, and that the `daemons` records from 1.1 now sit in `$HOME/.local/state/codegraph/daemons` with `$HOME/.codegraph` gone.
- [x] 8.3 In a second scratch home that holds `.codegraph/daemons/` and a `.codegraph/telemetry.json`, run `codegraph daemon` from the built binary. Verify the records moved and `$HOME/.codegraph/telemetry.json` is still there.
- [x] 8.4 Add the changelog fragment: `task changie -- new -k Fixes -b "codegraph no longer treats the home directory as a project when ~/.codegraph exists, and the daemon registry moves from ~/.codegraph/daemons to \$XDG_STATE_HOME/codegraph/daemons (default ~/.local/state/codegraph/daemons) with a one-time migration; delete a leftover ~/.codegraph/store by hand."` Verify the fragment file exists under `.changes/unreleased/`.

## Workflow follow-up

- The implementation PR says `Closes #74` and carries the `openspec archive xdg-global-state` result, as `CLAUDE.md` describes.
- Verify the archived change under `openspec/changes/archive/` and the two new main specs under `openspec/specs/user-directories/` and `openspec/specs/index-resolution/` after the archive.
