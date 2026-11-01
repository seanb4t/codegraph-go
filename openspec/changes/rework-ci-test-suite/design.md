# Design

## Context

The `test` job in `ci.yml` (lines 80-295) runs 21 steps in series on one runner. `task test:unit` runs `go test` over every package except `internal/daemon`, so `test/integration` and `test/wireoracle` already run there. `task test:integration` and `task test:wireoracle` then hit the test cache. `task test:daemon` runs `internal/daemon` with `-count=1`. `task test:race` runs `internal/daemon`, `internal/watch`, `internal/cli` and `internal/mcp` with `-race -count=1 -p 1`.

The `changes` job runs `scripts/planning_only.py` and gates `test`, `govulncheck`, `tool-vuln`, `reproducibility` and `tmux-e2e`. The classifier fails open: if it cannot decide, every heavy job runs.

The required checks in `protect-main` are `test`, `actionlint (workflow static analysis)`, `pr-title`, `reproducibility (double-build hash-diff, DIST-04)`, `govulncheck (DIST-03, blocking)`, `goreleaser check (config validation, DIST-01)`, `tmux e2e (real-pty harness, TTY-01..TTY-07)` and `fragment-required`.

`internal/daemon` already has seams: `syncFn`, and `WithProbe`. But `Daemon.Run` calls `watch.Open(d.repoRoot)` directly (`daemon.go:293`), so a test that needs watcher events writes real files and waits. The daemon and watch tests make 38 `time.After` calls and 6 `time.Sleep` calls.

`pnpm-workspace.yaml` already sets `strictDepBuilds: true` and an `audit` section, so pnpm enforces both on its own.

## Goals / Non-Goals

**Goals:**

- The `test` job of a Go change takes about 200 s, down from 314 s. A docs-only or web-only change skips most of it.
- No gate tests third-party behavior, and no gate has a second artifact that tests the gate.
- Each gate keeps a positive assertion inside itself.
- One implementation PR, with one commit for each step in the order below, so a reviewer can read it commit by commit.

**Non-Goals:**

- Property and fuzz tests (`index-invariants`).
- Mutation testing (#222).
- Mise tasks in place of `Taskfile.yml` (#178).

## Decisions

### D1. An aggregate `test` job keeps the required check

The `test` job becomes a small job with `needs:` on every leg and `if: always()`. If any leg reports `failure` or `cancelled`, it fails. If each leg passed or was skipped, it passes. The legs get new names, which the ruleset does not list. So `protect-main` changes only once: the tmux check goes (D8).

Alternative: make each leg a required check. Then every new or renamed leg needs a ruleset edit, and a forgotten edit blocks every PR.

### D2. `paths-filter` decides which legs run, and it errs toward running

A filter that lists the inputs of each leg misses inputs. Examples are `.changes/**` for `check:changie`, the `buf` configuration for `proto:drift`, `docs/CLI-REFERENCE.md` for `docs:cli:drift`, and `tools/bench/baseline.json`, which `internal/bench/baseline_gate_test.go` reads. So the filters name what a leg can skip, not what it needs. The `changes` job runs `dorny/paths-filter`, pinned by SHA, with `predicate-quantifier: every`, and these outputs:

| output | true when |
|---|---|
| `go` | any changed file is outside `openspec/**`, `web/src/**`, `web/tests/**`, and Markdown files other than `docs/CLI-REFERENCE.md` |
| `web` | any changed file is under `web/**`, is a `.proto` file or a `buf*.yaml` file, or is `Taskfile.yml`, `mise.toml` or `mise.lock` |
| `release` | any changed file is outside `openspec/**`, `web/src/**`, `web/tests/**` and Markdown files |
| `ci` | any changed file is under `.github/**`, `scripts/**` or `tests/**`, or is `Taskfile.yml`, `mise.toml`, `mise.lock` or `.golangci.yml` |

A change that matches `ci` runs every leg. On a push to `main`, every leg runs. If the filter step fails, the job sets every output to `true`, as `planning_only.py` did. `planning_only.py` and its tests go.

### D3. The legs

| leg | runs when | content |
|---|---|---|
| `go` | `go` or `ci` | build, vet, `lint:go`, `web:deps` (for `protoc-gen-es`), `gotestsum` over every package except `internal/daemon`, `corpora:verify`, `proto:drift`, `docs:cli:drift` |
| `go-race` | `go` or `ci` | `gotestsum -- -race` over `internal/cli`, `internal/mcp` and `internal/watch`, in parallel |
| `daemon` | `go` or `ci` | `gotestsum -- -race -p 1 ./internal/daemon` |
| `web` | `web` or `ci` | `web:deps`, vitest, `pnpm audit`, the SPA rebuild and drift guards, `check:no-force-layout` |
| `release-checks` | `release` or `ci` | `check:gonum`, `check:changie` (D9) |

The daemon leg runs on any Go change, not only on a change under `internal/daemon`. The daemon runs `indexer.Sync` and opens the store (`internal/daemon/daemon.go:15-17`, `:469-474`), so an indexer or graphstore change can break it. On an unrelated change, Go's test cache makes the leg short.

`proto:drift` needs `web/node_modules/.bin/protoc-gen-es` (`Taskfile.yml:424`), so the `go` leg installs the frozen web dependencies first, from the pnpm cache.

The plain `internal/daemon` run goes, because the race run covers it. `-count=1` stays only on the `daemon` leg. The other legs restore the Go build and test cache, so an unchanged package is a cache hit. `govulncheck`, `tool-vuln` and `reproducibility` keep their own jobs and their required-check names, and use the `release` or `ci` output in place of `planning_only`.

Each leg runs `gotestsum --junitfile` and prints the 20 slowest tests with `gotestsum tool slowest`. It never uses `--rerun-fails`, because a rerun hides a flake.

### D4. `concurrency` cancels a superseded PR run

`ci.yml` gets `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: ${{ github.event_name == 'pull_request' }} }`. A push to `main` always runs to completion, so each merged commit gets its full set of checks.

### D5. `synctest` for timers, and a watcher seam for the daemon

The tests in `internal/watch` that wait only on the debouncer move into `synctest.Test`, so their timers advance at once.

`Daemon.Run` uses two methods of the watcher: `Run(ctx, deb)` (`daemon.go:349`) and `Close()` (`:297`). Both backends already have them. The Linux backend reads fsnotify channels inside its own `Run`, and the macOS backend reads FSEvents batches inside its own `Run`. So the seam is an interface with exactly those two methods, `Run(context.Context, *watch.Debouncer)` and `Close() error`, and an option `WithWatcher(open func(root string) (Watcher, error))`. A fake's `Run` feeds the debouncer by hand, inside `synctest.Test`. Then the quiet gap in `TestDaemonFlushLockRequeueGivesUpPerEpisode` and the debounce waits pass in fake time. The tests that prove the real watcher works keep real files and real time, and stay few.

`synctest` fakes time only for goroutines that wait on channels, timers or a `WaitGroup`. A goroutine that waits on fsnotify or Pebble I/O keeps real time. That is why the daemon needs the seam.

### D6. Linters for direct rules, one closure test for transitive rules

| today | kind of rule | replacement |
|---|---|---|
| `graphstore/archtest/import_graph_test.go`: only `graphstore` imports Pebble (D-04a) | direct import | `depguard` |
| `cli/archtest/mcp_sdk_confinement_test.go` (SDK-02) | direct import | `depguard` |
| `graphstore/archtest/stdout_confinement_test.go` (HYG-02) | call site | `forbidigo` (below) |
| `mcp/archtest/protocol_version_test.go` (VRFY-02) | reference | `forbidigo` (below) |
| `query/archtest/import_direction_test.go` (T-01-18) | transitive | `TestDependencyClosure` |
| `cli/present/archtest/import_graph_test.go` (TUI-01) | transitive | `TestDependencyClosure` |
| `cli/present/archtest/charm_cgo_test.go` (REL-01) | transitive | `TestDependencyClosure` |

HYG-02 keeps all three of its bans: `os.Stdout`, a bare `fmt.Print*`, and `log.SetOutput`. The `forbidigo` rules apply to production files of each `internal/` package except `internal/cli`, so test files stay out, as today. The one legitimate reference, the `ServeStdio` transport writer in `internal/mcp/server.go`, gets an exclusion in `.golangci.yml` that names that file and the matched source line. A second `os.Stdout` anywhere else in `internal/mcp` still fails.

VRFY-02 today loads the module and also `testdata/golden`, which `./...` skips (`protocol_version_test.go:232`). So `lint:go` adds a second `golangci-lint run` over `./testdata/golden/...`, with only the VRFY-02 rule enabled.

`TestDependencyClosure` runs `go list -deps` once for the production packages and once with `-test`. It asserts the three transitive rules, and keeps T-01-18's split: the wire-layer rule covers test variants, and the indexer-root rule covers production only. It asserts that each closure it read holds more than zero packages. The four `*_selftest_test.go` files go. `internal/upgrade/golangci_shape_test.go` reads `# enabled-linters: 5`, so the comment and the test change to 7.

### D7. `testscript` for `test/integration`

`test/integration` has 12 tests in 10 files, and its `TestMain` builds the binary or reads `CODEGRAPH_TEST_BIN`. `verify:notarized-suite` (`Taskfile.yml:5272-5279`), which `post-release-verify.yml` runs, sets `CODEGRAPH_TEST_BIN` to the downloaded, notarized release binary. So the integration suite tests the published artifact after each release, and that must not change.

With `testscript.Main`, the scripts run a `codegraph` command. If `CODEGRAPH_TEST_BIN` is unset, that command is the test binary itself, so a normal run needs no build step. If `CODEGRAPH_TEST_BIN` is set, the `codegraph` command runs that file. Each test becomes a `.txtar` script under `test/integration/testdata/`. A test that needs a timeout or a pipe that never closes keeps a small Go helper command in the script. `binpath_test.go` and the build step in `TestMain` go. `github.com/rogpeppe/go-internal` moves from an indirect to a direct requirement.

### D8. `teatest/v2` for the tmux rig (#75)

The decision on #75 stands. Five of the six tmux tests become in-process tests with `NewTestModel(..., WithInitialTermSize(100, 30))`. The mode-query leak test and the main-buffer test become model-level assertions on the output bytes: no query sequence appears outside the alt screen, and the alt-screen leave sequence is the last control sequence written. `test/tmux/`, `task test:tmux`, `task test:tmux:install` and the `tmux-e2e` job go. The `protect-main` ruleset drops its required check right before the PR merges, with the maintainer's approval, because that is an edit outside the repository.

### D9. Smaller replacements

- `check:changie` becomes one dry-run merge whose output must equal `CHANGELOG.md`. It asserts that it compared more than zero versions.
- `web:lockfile`, `web:deps:strict` and the Go classifier in `web:audit` go. `web:deps` (frozen lockfile, with `strictDepBuilds` enforced by pnpm) and `pnpm audit` with the existing `audit.ignore` stay.
- `scripts/check-workflow-output-delimiter.sh` goes. `zizmor`, pinned in `mise.toml`, runs beside `actionlint` in the `actionlint` job, so no required-check name changes.
- `scripts/fragment_policy.py` goes. `fragment-required.yml` runs `paths-filter` with an `added` rule on `.changes/unreleased/**` and a code rule on `cmd/**` and `internal/**`. A short shell step reads three facts: code changed, no fragment was added, and the PR body has no `changelog-exempt` marker. If all three hold, the step fails. At release time, `changie` itself rejects a fragment with an unknown kind.
- `check:gonum` and `check:no-force-layout` lose their positive-control legs. Each asserts that it scanned more than zero items.
- `scripts/mutation_check.py`, `tests/test_mutation_check.py`, `task mutation-check` and `.claude/skills/mutation-check/` go. `check:ci-scripts` keeps only `test_vuln_report`.

### D10. `goreleaser build` for the reproducibility check (#212)

`check:reproducibility` runs `goreleaser build --snapshot --clean --id codegraph-linux-amd64` twice, copies the binary out of `dist/` after each run, and compares the two hashes. The arm64 task does the same with `--id codegraph-linux-arm64` and stays advisory. Both use the flags in `.goreleaser.yaml`, so no flags are copied by hand and no shell formats a date. GoReleaser and zig are pinned in `mise.toml`, so the check runs on macOS and Linux. The CI step "Compute determinism inputs" goes, because GoReleaser reads the commit itself.

### D11. The hygiene checks read every file (#106)

`require-issue-link.yml` and `pr-template-format.yml` list the PR's files with `gh api --paginate repos/{owner}/{repo}/pulls/{n}/files`, which returns up to 3,000 files, not 100. If the API reports 3,000 files, the check treats the PR as not exempt. `pr_template_policy.emit` replaces each newline in the reason with a space and strips a leading `::`, so PR-body text cannot become a workflow command.

### D12. The perf gate checks its baseline (#105)

Before the comparison, `bench.yml` reads the date of the last commit that changed the baseline, with `git log -1 --format=%ct -- tools/bench/baseline.json`. The checkout sets `fetch-depth: 0`, because a shallow checkout reports the checkout boundary, not the last change. If the command prints nothing, the job fails. The job also reads the `runner` field of the file. If the baseline is older than 90 days, or `runner` differs from `CODEGRAPH_BENCH_RUNNER`, the job fails with a message that names the `rebless` job. The file keeps its fields.

### D13. The two flaky tests

- #235: in `awaitDaemonLock` (`internal/cli/serve_worktree_test.go`), an `os.ErrNotExist` from `w.Add` on the index directory goes to the next loop pass, like a missing watch does today. Any other error still fails the test.
- #185: the CR-01 test in `web/tests/browse-page.test.ts` waits with `vi.waitFor` on the RPC count and the URL, in place of a fixed number of microtask flushes. The fix is accepted after 20 full-suite runs with no failure.

### D14. Order of the commits

1. The two flaky tests, #232, and `gotestsum`.
2. `paths-filter`, the legs, the aggregate `test` job and `concurrency` (D1 to D4).
3. `depguard`, `forbidigo` and `TestDependencyClosure` (D6).
4. The daemon and watch timing work (D5).
5. `teatest` (D8).
6. `testscript` (D7).
7. The smaller replacements and deletions (D9), `goreleaser build` (D10), #106 (D11) and #105 (D12).
8. The `openspec/config.yaml` rules.

## Risks / Trade-offs

- One large PR is harder to review than seven small ones. The commit order above, with the timing of each commit in the PR body, keeps it readable. The user chose fewer PRs.
- A path filter that is too narrow skips a leg that a change needs. So the Go filters list only what a leg can skip (D2). The `ci` filter runs every leg, and every push to `main` runs every leg.
- `forbidigo` on each package except `internal/cli` is stricter than the closure walk of HYG-02, because it also covers packages that `serve` does not reach. Each exclusion in `.golangci.yml` names one file and one source line, as the `ServeStdio` exclusion does, never a whole package.
- The model-level byte assertions in D8 do not prove what a real terminal shows. The maintainer accepted this on #75.
- The `teatest/v2` module path and its API for `charm.land/bubbletea/v2` are not yet verified. The implementation reads the current docs first. If the module does not fit, the implementation stops and reports back before it writes the tests.
- `zizmor` and `gotestsum` need a prebuilt release in mise with a `mise.lock` checksum. If one of them has none, the implementation asks before it picks another install path.
