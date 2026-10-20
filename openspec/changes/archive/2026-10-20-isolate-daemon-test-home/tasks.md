# Tasks

Run Go with `GOTOOLCHAIN=go1.26.9`, the version `go.mod` names. Every reproduction below points `HOME` and `XDG_STATE_HOME` at directories from `mktemp -d` and exports the real `GOPATH`, `GOMODCACHE`, `GOCACHE` and `GOENV` from `go env` first, or `go test` starts from an empty module cache and build cache. Keep the two scratch directories for the whole change so the RED and GREEN runs compare like with like.

## 1. Baseline

- [x] 1.1 Record the tests that skip today: `go test ./internal/daemon ./test/integration -count=1 -v 2>&1 | rg -- '--- SKIP' | sort` saved to a file outside the repository. Verify the log holds `--- PASS` lines, so the pattern matches the log format.
- [x] 1.2 Reproduce #136 (RED): with `HOME` and `XDG_STATE_HOME` set to two fresh `mktemp -d` directories, run `go test ./internal/daemon -count=1`. Verify that `$XDG_STATE_HOME/codegraph/daemons` now exists and holds at least one `.json` record. Verify that `$HOME` holds nothing but what the `go` tool put there, if anything.
- [x] 1.3 Reproduce the integration gap (RED): empty both scratch directories, then run `go test ./test/integration -run '^TestLiveEditAutoSyncReachesExplore$' -count=1` with the same two variables. Verify that `$XDG_STATE_HOME/codegraph/daemons` exists afterwards.

## 2. Backstop in `internal/daemon`

- [x] 2.1 Add `homeEnv(home)` to `internal/daemon/registry_test.go`, returning `HOME`, `XDG_STATE_HOME` and `XDG_CONFIG_HOME` under `home`. Add `internal/daemon/main_test.go` with a `TestMain` that creates a temporary home with `os.MkdirTemp`, sets each `homeEnv` value with `os.Setenv`, stops the run if `os.UserHomeDir()` does not return that home, and calls `goleak.VerifyTestMain(m, goleak.Cleanup(...))` where the cleanup removes the home and exits with the code goleak passes. Delete `TestMain` from `soak_test.go` and move its SYNC-06 comment. Verify `go test ./internal/daemon -count=1` passes.
- [x] 2.2 Prove the exit code still propagates: add a throwaway `TestExitProbe` that calls `t.Fatal`, run `go test ./internal/daemon -run '^TestExitProbe$' -count=1; echo $?`, verify the exit status is `1`, then delete the probe and verify `git status` shows no stray file.
- [x] 2.3 GREEN for 1.2: empty both scratch directories and rerun the 1.2 command. Verify that `$XDG_STATE_HOME` and `$HOME` are both empty afterwards, except for what the `go` tool itself writes under `$HOME`.

## 3. Backstop in `test/integration`

- [x] 3.1 Add `homeEnv(home)` to `test/integration/main_test.go`, returning `HOME`, `XDG_CONFIG_HOME`, `XDG_STATE_HOME` and `HERMES_HOME` under `home`. Extend `TestMain` so that, after `binPath` is resolved or built, it creates a temporary home, sets each `homeEnv` value, stops the run if `os.UserHomeDir()` does not return that home, runs `m.Run()`, then removes the home and the build directory. Both the `CODEGRAPH_TEST_BIN` branch and the build branch go through the isolation step. Verify `go test ./test/integration -count=1` passes.
- [x] 3.2 GREEN for 1.3: empty both scratch directories and rerun the 1.3 command. Verify that `$XDG_STATE_HOME` is empty afterwards and that `$HOME` holds nothing but what the `go` tool writes.

## 4. Integration checks

- [x] 4.1 Rerun the command from 1.1. Verify that its output matches the 1.1 file exactly, so no `git` helper started to skip under the fake home.
- [x] 4.2 Save the list of files under the real `~/.local/state/codegraph/daemons` (or note that the directory is absent), run `GOTOOLCHAIN=go1.26.9 task test` and then `GOTOOLCHAIN=go1.26.9 task lint`, one after the other, and read the list again. Verify both pass and that the list is unchanged. This step only reads the real home.
- [x] 4.3 The change is test-only, so no changelog fragment. Put `<!-- changelog-exempt: test-only change, no user-visible effect -->` in the PR body and verify that the `fragment-required` check passes.

## Workflow follow-up

- The implementation PR says `Closes #136` and carries the `openspec archive isolate-daemon-test-home` result, as `CLAUDE.md` describes.
- Verify the archived change under `openspec/changes/archive/` after the archive.
