# Tasks

Run every target as `mise exec -- task <target>` in a shell without `mise activate`. Make one commit for each numbered section, in this order. Put each section's CI timing in the PR body, from `gh run view` and the `gotestsum` slowest-test report.

## 0. Baseline

- [ ] 0.1 Record the `test` and `tmux-e2e` job times of the last green `ci.yml` run on `main` that changed `internal/`, with `gh run view <id> --json jobs`. Put them in the PR body as the "before" column.

## 1. Flaky tests, comments and the test runner

- [ ] 1.1 #235: in `awaitDaemonLock` (`internal/cli/serve_worktree_test.go`), treat an `os.ErrNotExist` from `w.Add` as "try again on the next pass" (design.md D13). Verify `go test ./internal/cli -run '^TestServeWatchStartBuildsThenWatches$' -count=200` passes on macOS.
- [ ] 1.2 #185: change the CR-01 test in `web/tests/browse-page.test.ts` to wait with `vi.waitFor` (design.md D13). Verify that `pnpm exec vitest run` passes 20 times in a row.
- [ ] 1.3 #232: in `internal/daemon/daemon_test.go:488` and `internal/watch/soak_test.go:68`, name the watch loop's behavior in place of `watchLoop`.
- [ ] 1.4 Pin `gotestsum` in `mise.toml`, with its hash in `mise.lock`. Make `test:unit`, `test:daemon` and `test:race` run through `gotestsum --junitfile`, and print `gotestsum tool slowest` at the end. Do not use `--rerun-fails`. Verify `task test` passes locally.

## 2. CI layout (design.md D1 to D4)

- [ ] 2.1 Replace the `changes` job with `dorny/paths-filter`, pinned by SHA, with the outputs of D2. If the filter step fails, set every output to `true`. Delete `scripts/planning_only.py` and `tests/test_planning_only.py`, and remove them from `check:ci-scripts`.
- [ ] 2.2 Split the `test` job into the legs of D3. The `go` leg installs the web dependencies before `proto:drift`. The `daemon` leg runs on the `go` output. Run `internal/daemon` once, under `-race -p 1`. Run `go-race` without `-p 1`. Drop `-count=1` except on the `daemon` leg. Remove the `test:integration` and `test:wireoracle` steps and `task test:daemon`.
- [ ] 2.3 Add the aggregate job named `test` (D1). Gate `govulncheck`, `tool-vuln` and `reproducibility` on the `release` or `ci` output.
- [ ] 2.4 Add the `concurrency` block of D4.
- [ ] 2.5 Verify on the PR with three scratch commits, then drop them. A change to only `web/src/` runs the `web` leg and skips the Go legs, and the required `test` status still reports success. A change to only `.changes/` runs `release-checks`. A change to only `docs/LANGUAGE-CAPABILITY-MATRIX.md` runs the `go` leg, and the `gotestsum` report shows that `TestMatrix_DocMirrorsDescriptor` ran. Link the three runs in the PR body. Verify `task lint:actions` passes.

## 3. Linters in place of `archtest` (design.md D6)

- [ ] 3.1 Add `depguard` rules for D-04a and SDK-02 to `.golangci.yml` (design.md D6). Add `forbidigo` rules for HYG-02 (`os.Stdout`, bare `fmt.Print*`, `log.SetOutput`, production files only) and for VRFY-02. Add the one `ServeStdio` exclusion, scoped to its file and source line. Add the second `golangci-lint run` over `./testdata/golden/...` to `lint:go`. Change `# enabled-linters: 5` to 7 and update `internal/upgrade/golangci_shape_test.go`. Verify that each rule fires: add one forbidden import or call in a scratch commit, run `task lint:go`, see it fail, then drop the scratch commit. Do this also for an `os.Stdout` elsewhere in `internal/mcp`, and for a forbidden protocol-version constant under `testdata/golden`.
- [ ] 3.2 Add `TestDependencyClosure`, which runs `go list -deps` for production and test variants and asserts T-01-18, TUI-01 and REL-01 (design.md D6). It asserts that each closure holds more than zero packages.
- [ ] 3.3 Delete the `archtest` packages and their `*_selftest_test.go` files. Verify `task test` and `task lint` pass.

## 4. Timing in `internal/watch` and `internal/daemon` (design.md D5)

- [ ] 4.1 Move the debouncer-only tests in `internal/watch` into `synctest.Test`. Each moved test keeps its assertions.
- [ ] 4.2 Add `WithWatcher` and the `Watcher` interface, with `Run(context.Context, *watch.Debouncer)` and `Close() error`, to `internal/daemon` (design.md D5). If the option is absent, `Run` uses `watch.Open`. Add `TestRunUsesInjectedWatcher`, which feeds one event through a fake and asserts one sync.
- [ ] 4.3 Move `TestDaemonFlushLockRequeueGivesUpPerEpisode` and the other debounce and quiet-gap tests to the fake watcher inside `synctest.Test`. Keep the tests that prove the real watcher on real files.
- [ ] 4.4 Verify `go test -race -count=20 ./internal/daemon ./internal/watch` passes, and record the `daemon` leg time in the PR body.

## 5. `teatest` in place of tmux (design.md D8, #75)

- [ ] 5.1 Read the current `teatest/v2` docs with Context7. Make sure that the module path is correct and that it accepts a `charm.land/bubbletea/v2` model. If it does not, stop and report back.
- [ ] 5.2 Add the module with `go get` and a hand edit of `go.mod`. Write `TestInstallPickerFrameStableWhileIdle`, `TestInstallPickerFooterAt100x30`, `TestInstallPickerCancelWritesNoConfig`, `TestDaemonPickerLeavesAltScreenLast` and `TestDaemonEmptyRegistryWritesNoModeQuery` next to the models they test. Each asserts that the program wrote more than zero bytes.
- [ ] 5.3 Delete `test/tmux/`, `task test:tmux`, `task test:tmux:install` and the `tmux-e2e` job.
- [ ] 5.4 Right before the PR merges, and with the maintainer's approval, remove `tmux e2e (real-pty harness, TTY-01..TTY-07)` from the required status list of the `protect-main` ruleset.

## 6. `testscript` in place of the integration build contract (design.md D7)

- [ ] 6.1 Make `github.com/rogpeppe/go-internal` a direct requirement. Add `testscript.Main` to `test/integration` with `codegraph` as the command. If `CODEGRAPH_TEST_BIN` is set, the command runs that file (design.md D7).
- [ ] 6.2 Move the 12 integration tests to `.txtar` scripts under `test/integration/testdata/`. Keep a small Go helper command for the tests that need a timeout or a pipe that stays open. A guard asserts that the script directory holds 12 or more scripts.
- [ ] 6.3 Delete `binpath_test.go` and the build step in `main_test.go`. Verify `go test ./test/integration/...` passes. Then build `codegraph`, run the suite with `CODEGRAPH_TEST_BIN` set to it, and verify it passes. Set `CODEGRAPH_TEST_BIN=/usr/bin/false` and verify that the suite fails, which shows that the scripts run the named file.

## 7. Smaller replacements and deletions (design.md D9 to D12)

- [ ] 7.1 Shrink `check:changie` to the dry-run comparison with `CHANGELOG.md`. It asserts that it compared more than zero versions.
- [ ] 7.2 Delete `web:lockfile`, `web:deps:strict` and the Go classifier of `web:audit`. Keep `web:deps` and `pnpm audit`.
- [ ] 7.3 Pin `zizmor` in `mise.toml` and run it in the `actionlint` job. Fix or record each finding it reports. Delete `scripts/check-workflow-output-delimiter.sh` and `task check:workflow-output-delimiter`.
- [ ] 7.4 Replace `scripts/fragment_policy.py` and `tests/test_fragment_policy.py` with the `paths-filter` rules and shell step of D9 in `fragment-required.yml`. Keep the job name `fragment-required`. Move the `pr_template_policy` tests that lived in `test_fragment_policy.py` to `tests/test_pr_template_policy.py`. Verify on the PR that `fragment-required` passes with this change's `changelog-exempt` marker.
- [ ] 7.5 Remove the positive-control legs of `check:gonum` and `check:no-force-layout`. Each asserts that it scanned more than zero items.
- [ ] 7.6 Delete `scripts/mutation_check.py`, `tests/test_mutation_check.py`, `task mutation-check` and `.claude/skills/mutation-check/`.
- [ ] 7.7 #212: rewrite `check:reproducibility` and `check:reproducibility:arm64` on `goreleaser build --snapshot` (D10), and remove the "Compute determinism inputs" step. Verify `task check:reproducibility` passes on macOS and in CI.
- [ ] 7.8 #106: list PR files with `gh api --paginate` in `require-issue-link.yml` and `pr-template-format.yml`, and escape the reason in `pr_template_policy.emit` (D11). Add `test_emit_escapes_workflow_command` and `test_truncated_file_list_is_not_exempt` to `tests/test_pr_template_policy.py`.
- [ ] 7.9 #105: add the baseline age and runner test to `bench.yml` (D12). Run it once with `gh workflow run bench.yml --ref <branch> -f job=regression`, and link the regression job's output in the PR body.

## 8. OpenSpec rules and close-out

- [ ] 8.1 Add two `rules.tasks` entries to `openspec/config.yaml`. A task that adds a test names the property, fuzz target or scenario-shaped test that covers it. A task that adds a gate states the positive assertion the gate makes.
- [ ] 8.2 Search for each removed task and script name with `rg`, and update each doc and comment that still names one.
- [ ] 8.3 Verify `task test` and `task lint` pass, and put the "after" timings in the PR body.
- [ ] 8.4 Supersede engram memory `gq9fpccbka`, which names the deleted `mutation-check` script.
- [ ] 8.5 Put `<!-- changelog-exempt: CI and test tooling only, no user-visible change -->` in the PR body.
