---
phase: 03-watcher-on-mcp-default
plan: 04
subsystem: testing
tags: [integration-test, mcp-go, stdio, git-worktree, ci]

# Dependency graph
requires:
  - phase: 03-watcher-on-mcp-default
    provides: "internal/cli/serve.go's serveWatchStart seam + CR-01's two-arg mcp.BuildServer(hasIndex, allowlist, repoPath, start) call (03-03)"
  - phase: 02-status-content-git-worktree-awareness
    provides: "gitmeta.Mismatch.Notice() (the bare U+26A0 glyph), internal/cli/notice_test.go's real-git fixture + bare-glyph helper pattern"
provides:
  - "test/integration/ — a normal Go package (never testdata/) hosting TEST-04's subprocess integration harness"
  - "TestMain: builds the real github.com/seanb4t/codegraph-go/cmd/codegraph binary once per go test run into a package-level binPath var (D-18)"
  - "runGitI/noticeGlyph/containsBareNoticeGlyph/copyFixture/runBinary/buildWorktreeFixture — the substrate helpers, a fourth package-local copy of the established runGitW/runGitM/runGitC pattern"
  - "TestWorktreeNoticeReachesServeMCPExplore — the mandatory D-20 CR-01 anchor: real `serve --mcp` subprocess, real process cwd inside a worktree, real stdio JSON-RPC, mutation-proof against CR-01's root cause"
  - "newServeClient/resultText/exploreAlpha — mcp-go stdio client helpers driving the spawned binary (D-19)"
  - "an explicit named CI step (go test ./test/integration/...) sibling to the golden-suite step (D-17 belt-and-braces)"
affects: [03-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "transport.WithCommandFunc (mcp-go client/transport) to set the spawned subprocess's real cmd.Dir — proves the harness drives the actual process cwd, not a `-p` flag substitute"
    - "TestMain(m *testing.M) builds a release binary into a temp dir once per package test run, stores the path in a package-level var, os.Exit(m.Run()) after cleanup"

key-files:
  created: [test/integration/main_test.go, test/integration/worktree_notice_test.go]
  modified: [.github/workflows/ci.yml]

key-decisions:
  - "Used mcp-go's transport.WithCommandFunc to set the spawned serve --mcp subprocess's real cmd.Dir directly, rather than the plan's discretionary -p-flag fallback — this drives the actual process cwd (the literal D-19/D-20 requirement) rather than a documented seam-equivalent substitute, since the option was confirmed present in the pinned v0.56.0 client/transport package"
  - "buildWorktreeFixture indexes the main checkout via runBinary (codegraph init <path>, cwd=main) — the real subprocess binary, not an in-process execCmd call — matching D-19's seam distinction even in fixture setup, not just the assertion itself"
  - "copyFixture/runGitI/noticeGlyph/containsBareNoticeGlyph are deliberate FOURTH package-local copies of the established internal/cli/notice_test.go (third copy) pattern — Go test helpers are not importable across packages, an established, documented convention in this codebase"

patterns-established:
  - "Manual mutation-proof verification: temporarily collapsed serve.go's BuildServer(hasIndex, allowlist, repoPath, start) to BuildServer(hasIndex, allowlist, repoPath, repoPath) (the literal CR-01 regression) via sed, confirmed TestWorktreeNoticeReachesServeMCPExplore turned red, reverted via the sed backup, confirmed git diff --exit-code clean before any commit — same discipline as 03-03's manual mutation check"

requirements-completed: [TEST-04]

coverage:
  - id: D1
    description: "test/integration/ is a normal Go package (never testdata/) reached both by go test ./... and by an explicit named CI step"
    requirement: "TEST-04"
    verification:
      - kind: other
        ref: "go list ./... | grep -q '/test/integration$' (LIST-OK)"
        status: pass
      - kind: other
        ref: "grep -q 'go test ./test/integration/' .github/workflows/ci.yml && yq eval '.' .github/workflows/ci.yml (CI-OK)"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestMain builds the real codegraph release binary once per test run into a package-level binPath var; a build failure aborts the run"
    requirement: "TEST-04"
    verification:
      - kind: integration
        ref: "test/integration/main_test.go#TestMain (exercised implicitly by every go test ./test/integration/... run — verified via `go test ./test/integration/... -run TestNothing -count=1 -v`, 3.4s build)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestWorktreeNoticeReachesServeMCPExplore: a real serve --mcp subprocess spawned with cwd inside a real git worktree returns a codegraph_explore payload carrying the bare U+26A0 worktree notice; a main-checkout control shows none; mutation-proof against CR-01's root cause"
    requirement: "TEST-04"
    verification:
      - kind: integration
        ref: "test/integration/worktree_notice_test.go#TestWorktreeNoticeReachesServeMCPExplore"
        status: pass
      - kind: other
        ref: "manual mutation check: BuildServer(hasIndex, allowlist, repoPath, repoPath) turned the test red (worktree text no longer contained the glyph); reverted via sed backup, git diff --exit-code internal/cli/serve.go clean before commit"
        status: pass
    human_judgment: false

# Metrics
duration: 9min
completed: 2026-07-16
status: complete
---

# Phase 3 Plan 4: TEST-04 Subprocess Integration Harness Summary

**A new `test/integration/` package that builds the real codegraph binary once (TestMain), drives a spawned `serve --mcp` subprocess over real stdio JSON-RPC with its process cwd inside a real git worktree, and mutation-proves the CR-01 worktree-notice wiring — plus an explicit CI step that can't be silently dropped.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-16T10:22:24-04:00
- **Completed:** 2026-07-16T10:29:38-04:00
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- `test/integration/` — a normal Go package (never `testdata/`, D-17) reached both by `go list ./...` and by a new explicit CI step, closing the exact GOLDEN-01-shaped silent-skip risk this harness is designed to prevent for itself
- `TestMain` builds `github.com/seanb4t/codegraph-go/cmd/codegraph` once per `go test` run into a package-level `binPath` (D-18) — every test in the package drives the real production binary, not an in-process command tree
- Fourth package-local copies of the established `runGitW`/`runGitM`/`runGitC` and bare-glyph pattern (`runGitI`, `noticeGlyph`, `containsBareNoticeGlyph`) plus `copyFixture`, `runBinary` (drives the spawned binary via `exec.Command`), and `buildWorktreeFixture` (indexes the main checkout through the subprocess binary itself, not an in-process call — the D-19 seam distinction applied even to fixture setup)
- `TestWorktreeNoticeReachesServeMCPExplore` — the mandatory D-20 CR-01 anchor: spawns `serve --mcp` with its real process `cmd.Dir` set inside `.claude/worktrees/probe` (via mcp-go's `transport.WithCommandFunc`, not a `-p` flag substitute), calls `codegraph_explore` over real stdio JSON-RPC, and asserts the payload carries the bare U+26A0 notice; a control run from the main checkout asserts none
- **Manually verified mutation-proof**: collapsing `serve.go`'s `BuildServer(hasIndex, allowlist, repoPath, start)` to `BuildServer(hasIndex, allowlist, repoPath, repoPath)` (the literal CR-01 regression) turned the anchor test red; reverted cleanly before any commit
- `.github/workflows/ci.yml` gains an explicit `go test ./test/integration/...` step, sibling to the existing golden-suite step, with a comment citing the same D-17/GOLDEN-01 belt-and-braces rationale

## Task Commits

Each task was committed atomically:

1. **Task 1: Harness substrate — main_test.go** - `94fb61b` (test)
2. **Task 2: CR-01 anchor — worktree_notice_test.go** - `b630a0d` (test)
3. **Task 3: Wire the explicit integration CI step in ci.yml** - `91f07f9` (ci)

**Plan metadata:** (this commit)

## Files Created/Modified
- `test/integration/main_test.go` - `TestMain` (binary build), `runGitI`, `noticeGlyph`, `containsBareNoticeGlyph`, `copyFixture`, `runBinary`, `buildWorktreeFixture`
- `test/integration/worktree_notice_test.go` - `newServeClient`, `resultText`, `exploreAlpha`, `TestWorktreeNoticeReachesServeMCPExplore`
- `.github/workflows/ci.yml` - new explicit `go test ./test/integration/...` step in the `test` job, sibling to the golden-suite step

## Decisions Made
- Chose `transport.WithCommandFunc` (confirmed present in the pinned `mcp-go@v0.56.0` `client/transport` package) to set the spawned `serve --mcp` subprocess's real `cmd.Dir`, rather than the plan's discretionary `-p worktreeStart` fallback — this drives the literal D-19/D-20 requirement (real process cwd) instead of a documented seam-equivalent substitute.
- `buildWorktreeFixture` indexes the main checkout via `runBinary` (the real subprocess binary, `codegraph init <path>` with `cwd=main`), not an in-process `execCmd` call — applying the D-19 seam distinction to fixture setup as well as the assertion itself, unlike the in-process `internal/mcp/markdown_test.go` fixture it's adapted from.
- `copyFixture`/`runGitI`/`noticeGlyph`/`containsBareNoticeGlyph` are deliberate FOURTH package-local copies of the established pattern (`internal/query/engine_worktree_test.go`'s `runGitW`, `internal/mcp/markdown_test.go`'s `runGitM`, `internal/cli/notice_test.go`'s `runGitC`) — Go test helpers are not importable across packages, an established codebase convention, not an oversight.

## Deviations from Plan

None — plan executed exactly as written. The manual mutation-proof verification step (plan's own acceptance criterion) was performed as specified: `internal/cli/serve.go`'s `BuildServer(hasIndex, allowlist, repoPath, start)` was temporarily edited (via `sed`, backed up) to `BuildServer(hasIndex, allowlist, repoPath, repoPath)`, confirmed `TestWorktreeNoticeReachesServeMCPExplore` turned red (the worktree-cwd payload no longer carried the notice glyph), then reverted from the `sed` backup before any commit — verified via `git diff --exit-code internal/cli/serve.go` after reverting, which was clean.

## Issues Encountered

None. Full verification suite green: `go build ./...`, `go test` across all non-daemon packages (mirroring ci.yml's filtered step), `go test ./testdata/golden/...`, `go test ./internal/daemon/ -count=1`, and `go test ./test/integration/...` all pass. `go vet ./test/integration/...` clean. `gofmt -l` clean on both new test files. YAML validity of `ci.yml` confirmed via both `yq` and Ruby's `YAML.load_file` (Python's `yaml` module was unavailable in this environment, so the plan's literal `python3 -c "import yaml"` verify command was substituted with equivalent available tooling).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TEST-04 is complete: the subprocess integration harness substrate exists, the mandatory CR-01 anchor is mutation-proven, and CI runs it via an explicit named step immune to a future `go list` refactor.
- 03-05 (D-21: WATCH-01/02 default-on + `CODEGRAPH_NO_WATCH=1` off-switch subprocess cases) can now reuse this plan's substrate directly — `binPath`, `runGitI`, `copyFixture`, `runBinary`, `newServeClient`, `resultText` are all in place and package-local to `test/integration/`, no new harness scaffolding needed.
- No blockers.

---
*Phase: 03-watcher-on-mcp-default*
*Completed: 2026-07-16*

## Self-Check: PASSED

All created/modified files (`test/integration/main_test.go`, `test/integration/worktree_notice_test.go`, `.github/workflows/ci.yml`, `03-04-SUMMARY.md`) and all four commit hashes (`94fb61b`, `b630a0d`, `91f07f9`, `91e94b8`) verified present in the working tree and git history.
