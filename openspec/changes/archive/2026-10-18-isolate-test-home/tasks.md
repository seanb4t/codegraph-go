# Tasks

Run Go with `GOTOOLCHAIN=go1.26.9`, the version `go.mod` names. Never run a reproduction against the real home: each one points `HOME` at a directory from `mktemp -d`. When `HOME` moves, also export the real `GOPATH`, `GOMODCACHE`, `GOCACHE` and `GOENV` from `go env`, or `go test` starts from an empty module cache and build cache.

## 1. Baseline

- [x] 1.1 Record the tests that skip today: `go test ./internal/cli ./internal/agents -count=1 -v -skip '^TestUpgradeCommand_NoArgsDefaultsToLatest$' 2>&1 | rg -- '--- SKIP' | sort` saved to a file outside the repository. The `-skip` keeps the unfixed leaking test away from the real home. Verify the file lists every skip (it may be empty) and that the log holds `--- PASS` lines, so the pattern matches the log format.
- [x] 1.2 Reproduce the leak in a scratch home: with `HOME` set to a fresh `mktemp -d` directory, run `codegraph install --target claude --location global` from the installed binary, then `go test ./internal/cli -run '^TestUpgradeCommand_NoArgsDefaultsToLatest$' -count=1`. Verify that `jq .mcpServers.codegraph "$HOME/.claude.json"` in the scratch home now names a `go-build…/cli.test` command. Keep the scratch home for 2.3.

## 2. Backstop in `internal/cli`

- [x] 2.1 In `internal/cli/install_test.go`, move the three `HOME`, `XDG_CONFIG_HOME` and `HERMES_HOME` values into one helper, `homeEnv(home)`, that returns them as a map, and make `fakeHome` set each one with `t.Setenv`. Verify `go test ./internal/cli -count=1` passes (until 2.2 lands, with the 1.1 `-skip`).
- [x] 2.2 Add `internal/cli/main_test.go` with a `TestMain` that creates a temporary home with `os.MkdirTemp`, sets each `homeEnv` value with `os.Setenv`, stops the run if `os.UserHomeDir()` does not return that home, runs `m.Run()`, removes the home, and exits with the run's code. Verify `go test ./internal/cli -count=1` passes.
- [x] 2.3 Prove the backstop against the real leak, before the test is fixed: reset the scratch home from 1.2 with `codegraph install --target claude --location global`, then rerun the 1.2 test with that `HOME`. Verify that `jq .mcpServers.codegraph` in the scratch home still names the installed binary.

## 3. Fix the leaking test

- [x] 3.1 In `TestUpgradeCommand_NoArgsDefaultsToLatest` (`internal/cli/upgrade_test.go`), call `fakeHome(t)` and stub `refreshInstalledSkillsFunc` with a no-op restored by `t.Cleanup`, as `TestUpgradeCommand_RefreshesConfiguredLocations` does. Verify `go test ./internal/cli -run '^TestUpgradeCommand_' -count=1` passes.

## 4. Backstop in `internal/agents`

- [x] 4.1 In `internal/agents/testhelpers_test.go`, apply the same helper split as 2.1 to `fakeHome`. Verify `go test ./internal/agents -count=1` passes.
- [x] 4.2 Add `internal/agents/main_test.go` with the same `TestMain` as 2.2. Verify `go test ./internal/agents -count=1` passes.

## 5. Integration checks

- [x] 5.1 Rerun the command from 1.1. Verify that its output matches the 1.1 file exactly, so no `git` helper started to skip under the fake home.
- [x] 5.2 Save `jq .mcpServers.codegraph ~/.claude.json`, run `go test ./internal/cli ./internal/agents -count=1`, and read the entry again. Verify both reads match. This step only reads the real home.
- [x] 5.3 Run `GOTOOLCHAIN=go1.26.9 task test` and `GOTOOLCHAIN=go1.26.9 task lint`. Verify both pass.
- [x] 5.4 The change is test-only, so no changelog fragment. Put `<!-- changelog-exempt: test-only change, no user-visible effect -->` in the PR body and verify that the `fragment-required` check passes.

## Workflow follow-up

- The implementation PR says `Closes #73` and carries the `openspec archive isolate-test-home` result, as `CLAUDE.md` describes.
- Verify the archived change under `openspec/changes/archive/` after the archive.
