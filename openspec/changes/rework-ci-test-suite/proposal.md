# Proposal: Replace the bespoke test rigs with open-source tools and cut CI test time

Issue: #221, #75, #235, #185, #232, #106, #212, #105

This change is parts 1, 2 and 4 of #221. Part 3, the property and fuzz tests, is the change `index-invariants`.

## Why

Each new guard went into the single `test` job, and the repository had no baseline to measure that growth. On a PR that changed `internal/indexer`, the `test` job took 314 s, and the `tmux-e2e` job took another 144 s. `internal/daemon` runs twice per push, plain and under `-race`, for 143 s, although the race run covers the plain run. A superseded push runs to completion, because `ci.yml` has no `concurrency` block.

Many gates are bespoke rigs that test third-party behavior, which the project's own rule forbids. Examples are the 14-leg `check:changie`, the Go programs that `web:lockfile`, `web:deps:strict` and `web:audit` write and run, and the self-tests of five `archtest` packages. They cost review time on every change. The tmux rig cannot run on the maintainer's machines, and #75 decided to retire it.

Two flaky tests (#235, #185) fail required checks at random. Four small CI defects (#106, #212, #105, #232) touch the files that this change rewrites.

## What Changes

- CI layout: `dorny/paths-filter` replaces `scripts/planning_only.py`. It gates per-area legs: Go, daemon, web, wire oracle and release checks. A small job named `test` depends on every leg. If any leg failed, that job fails. So the required check `test` keeps its name and reports on every PR. `ci.yml` gets a `concurrency` block that cancels a superseded PR run.
- Test runner: `gotestsum` runs the Go tests, with JUnit output and the slowest-test report. It never reruns a failed test.
- Less repeated work: `internal/daemon` runs once, under `-race`. Only `internal/daemon` keeps `-p 1`. Deterministic packages lose `-count=1`, so Go's test cache skips packages that did not change. The integration and wire oracle steps go, because the unit step already runs them. The timer-bound tests in `internal/watch` and the pure-timer logic in `internal/daemon` move to `testing/synctest`. The daemon tests that drive the real watcher get a seam, so they stop waiting on wall-clock gaps.
- **Open-source tools in place of rigs:**
  - golangci-lint `depguard` and `forbidigo` replace the direct-import and stdout rules of the `archtest` packages, and the self-tests go.
  - `testscript` replaces the build contract in `test/integration`.
  - `teatest/v2` replaces the tmux rig (#75). The `tmux-e2e` job and its required check go.
  - `zizmor` replaces `scripts/check-workflow-output-delimiter.sh`.
  - One `changie` dry-run merge, compared with `CHANGELOG.md`, replaces the 14 legs of `check:changie`.
  - `pnpm install --frozen-lockfile`, `pnpm audit` and pnpm's own `strictDepBuilds` replace the three web rigs.
  - A `paths-filter` rule on `.changes/unreleased/**` replaces `scripts/fragment_policy.py`.
  - `goreleaser build` twice, with a hash comparison, replaces the hand-copied build flags of `check:reproducibility` and its arm64 leg. That removes the GNU `date -d` call, so the check runs on macOS (#212).
- Deletions: `scripts/mutation_check.py`, its tests, `task mutation-check` and the `mutation-check` skill go. #222 trials a tool later.
- Gates keep their positive assertion inside the gate: Each multi-leg self-proof goes. In its place, each gate treats zero inspected items as a failure.
- **Folded fixes:**
  - #235: if an entry of the watched directory disappears, the test helper retries `w.Add`.
  - #185: the browse-page test waits on the condition, not on a fixed number of ticks.
  - #232: two test comments name the watch loop's behavior, not `watchLoop`.
  - #106: the issue-link and PR-template checks read the full file list and escape the exemption reason.
  - #105: if its baseline is older than 90 days or comes from another runner, the nightly perf gate fails.
- OpenSpec rules: `openspec/config.yaml` gets two `rules.tasks` entries. A task that adds a test names the property, fuzz target or scenario-shaped test that covers it. A task that adds a gate states the positive assertion the gate makes.

Out of scope:

- The property, invariant and fuzz tests (`index-invariants`).
- Mutation testing (#222).
- Moving `Taskfile.yml` to mise tasks (#178). New tools get their pins in `mise.toml`, as #178 already set up.
- Any change to product behavior. No file under `cmd/` or `internal/` changes outside test files and test seams.

## Capabilities

### New Capabilities

None. This change alters CI and test tooling only, so it has no spec delta.

### Modified Capabilities

None.

## Impact

- `.github/workflows/`: `ci.yml` (new layout), `fragment-required.yml`, `require-issue-link.yml`, `pr-template-format.yml`, `bench.yml`.
- `Taskfile.yml`: many targets shrink or go. `task --list` changes, and so does each contributor command that names a removed target.
- `scripts/` and `tests/`: about 2,000 lines of Python and shell go.
- `.golangci.yml`: two more linters. `internal/upgrade/golangci_shape_test.go` reads the linter count, so it changes too.
- `mise.toml` and `mise.lock`: pins for `gotestsum` and `zizmor`.
- `go.mod`: `github.com/charmbracelet/x/exp/teatest/v2` (test only) and a direct `github.com/rogpeppe/go-internal` (now indirect), through the minimal-dependency audit.
- The `protect-main` ruleset: the required check `tmux e2e (real-pty harness, TTY-01..TTY-07)` goes, at the time this change merges. The required check `test` keeps its name.
- Engram memory `gq9fpccbka` names the deleted `mutation-check` script. After this change merges, it gets a superseding record.
- No user-visible change, so no changelog fragment.
