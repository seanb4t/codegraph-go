---
phase: 03-watcher-on-mcp-default
verified: 2026-07-16T16:03:25Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 3: Watcher-on-MCP Default Verification Report

**Phase Goal:** `serve --mcp` runs live in-process auto-sync by default (matching TS's auto-sync), with a `--no-watch` opt-out and a WSL2/slow-filesystem auto-off policy — restoring the live-sync experience with zero config change and without ever delaying the MCP handshake.
**Verified:** 2026-07-16T16:03:25Z
**Status:** passed
**Re-verification:** No — initial verification

## Method

This was not a documentation review. I read all 5 PLAN/SUMMARY pairs, all three code-review rounds (03-REVIEW.md, 03-REVIEW-2.md, 03-REVIEW-3.md) and both fix reports, then independently:

- Ran `go build ./...`, `go test ./...` (cached and `-count=1`), `go test ./testdata/golden/... -count=1`, `go test ./test/integration/... -count=1 -v`, `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...`, and `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` myself — not trusting SUMMARY-reported command output.
- Read the current (post-review-fix) `internal/cli/serve.go`, `internal/cli/daemon.go`, `internal/watch/policy.go`, and the policy-gate section of `internal/daemon/daemon.go` directly, since the three review rounds materially changed `serve.go`/`daemon.go`/`graphstore` after the plans were written.
- **Actually performed the CR-01 mutation test** (not just read the SUMMARY's claim of having done so): edited `serve.go`'s `BuildServer(hasIndex, allowlist, repoPath, start)` to `BuildServer(hasIndex, allowlist, repoPath, repoPath)`, reran `TestWorktreeNoticeReachesServeMCPExplore`, confirmed it went red (the worktree-cwd payload lost the notice glyph), then reverted and confirmed `git diff --exit-code internal/cli/serve.go` was clean.
- Confirmed `internal/agents/*.go` is byte-identical to `main` across the whole phase (`git diff --exit-code $(git merge-base main HEAD) HEAD -- internal/agents/`) — the literal "zero config change" claim underlying WATCH-01.
- Read `TestServeWatchStartDeferred`'s and `TestConvergenceTwoSessions`'s actual assertions to confirm they exercise the real production seams (not hand-built replicas) with genuine happens-before synchronization, not sleep/timeout races.
- Reproduced the pre-existing, documented `internal/daemon` full-suite-parallel flake (`go test ./... -count=1` failed `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`) and confirmed it is (a) a known pre-existing condition documented in the archived Phase 8 (v0.1) CONTEXT.md, predating this phase, and (b) not present when run via CI's actual isolated command (`go test ./internal/daemon/ -count=1`, 4/4 clean, including under `-race`).

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `serve --mcp` runs the file watcher by default with `--no-watch` to opt out (flipping the opt-in `--watch`); `install` already writes the byte-identical invocation, so live sync returns with no config change (WATCH-01) | ✓ VERIFIED | `internal/cli/serve.go:216-219`: `serveWatchStart` is called unconditionally (no `if watchMode` gate); flags `--no-watch`/`--watch` (repurposed force-on) exist (`serve.go:243-245`) with `cmd.MarkFlagsMutuallyExclusive("no-watch","watch")`. `internal/agents/*.go` confirmed byte-identical to `main` (`git diff --exit-code` clean) — the invocation `install` writes is unchanged. |
| 2 | Watcher startup never delays the MCP handshake or first-tool availability — started off the handshake path (WATCH-02) | ✓ VERIFIED | `serveWatchStart` (`serve.go:82-140`) spawns one goroutine and returns before `daemon.New`/policy-check/`acquire`/`watch.Open` run inside it. `TestServeWatchStartDeferred` (`internal/cli/serve_test.go:88-135`) uses a genuine block-until-released channel (no sleep race) to prove this; ran it green (`-race -count=1`). `TestDefaultWatchHandshakePrompt` (`test/integration/watch_default_test.go`) proves it end-to-end on the real spawned binary — ran green (0.69s). |
| 3 | A WSL2/slow-filesystem watch-policy auto-disables the watcher, honoring env precedence (`CODEGRAPH_NO_WATCH`/force-on), matching TS's escape hatch (WATCH-03) | ✓ VERIFIED | `internal/watch/policy.go`'s `WatchDisabledReason` implements the exact D-04 precedence (NoWatch→ForceWatch→WSL2+`/mnt/[a-z]`→default) with verbatim TS reason strings and strict `=="1"` env checks. `go test ./internal/watch/ -count=1` green; `TestNoWatchEnvDisablesViaStderr` (subprocess-level) confirms the verbatim stderr message reaches a real child process — ran green. |
| 4 | Concurrent `serve --mcp` sessions on one repo converge to a single writer (no double-watching), goleak-clean (WATCH-04) | ✓ VERIFIED | `daemon.RunWithRetry` (`daemon.go:266-279`) replaces defer-once with jittered defer-and-retry on `ErrLockLive`. `TestConvergenceTwoSessions` (`internal/daemon/soak_test.go`) proves at-most-one-writer-always and exactly-one-writer-eventually inside the package's existing goleak `TestMain`. Ran isolated (`go test ./internal/daemon/ -count=1` and `-race -count=1 -p 1`) — green 4/4 runs, including under `-race`. |
| 5 | A subprocess integration harness drives the real binary end-to-end (CLI argv + `serve --mcp` stdio JSON-RPC), CR-01 anchor case, CI wired alongside `go test ./testdata/golden/...` (TEST-04) | ✓ VERIFIED | `test/integration/` is a normal Go package (not `testdata/`), `TestMain` builds the real `cmd/codegraph` binary once. `TestWorktreeNoticeReachesServeMCPExplore` spawns the real binary with its real process `cmd.Dir` set inside a git worktree (via `transport.WithCommandFunc`, not a `-p` substitute) and asserts the bare U+26A0 notice reaches a real `codegraph_explore` payload, with a clean main-checkout control. **Personally reproduced the mutation-proof claim**: reverting `BuildServer`'s two-arg distinction turned this test red; reverted cleanly. `.github/workflows/ci.yml` has an explicit `go test ./test/integration/...` step (line 83-84) sibling to the golden step. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/watch/policy.go` | WATCH-03 decision function | ✓ VERIFIED | `WatchDisabledReason`, `DetectWSL` (cached), `ErrWatchDisabled`, `Probe` all present, exported, verbatim reason strings confirmed via read + grep |
| `internal/daemon/daemon.go` | Policy-gate-first `Run` + `RunWithRetry` + jitter | ✓ VERIFIED | Policy check is `Run`'s literal first statement (before `acquire()`); `RunWithRetry`/`jitter` present and unit/soak-tested |
| `internal/cli/serve.go` | Default-on flip + flags + off-handshake seam | ✓ VERIFIED | Read in full; matches plan's must_haves exactly; CR-01 two-arg `BuildServer` call intact; D-07 reconcile Sync unchanged in position (still synchronous pre-handshake) |
| `internal/cli/daemon.go` | Friendly disabled-message parity (review-added) | ✓ VERIFIED | `errors.Is(err, watch.ErrWatchDisabled)` branch prints the same D-12 message and exits cleanly |
| `test/integration/*.go` | Subprocess harness + CR-01 anchor + WATCH cases | ✓ VERIFIED | 4 integration tests present, all pass in a fresh `-count=1` run: `TestWorktreeNoticeReachesServeMCPExplore`, `TestDefaultWatchHandshakePrompt`, `TestNoWatchEnvDisablesViaStderr`, `TestLiveEditAutoSyncReachesExplore` (the latter added by the CR-01 review-fix pass, WR-04) |
| `.github/workflows/ci.yml` | Explicit integration step + race step + windows vet gate | ✓ VERIFIED | All three steps present and confirmed functionally (ran the exact commands locally); YAML validated |
| `internal/graphstore/locked_unix.go` / `locked_windows.go` | Platform-correct lock classification (review-added, CR-01 round 2/3) | ✓ VERIFIED | Build-tagged files present; `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` ran clean locally |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `watch.WatchDisabledReason` | `daemon.Run` | policy-gate-first enforcement | ✓ WIRED | `daemon.go:177` calls it as literal first statement |
| `watch.WatchDisabledReason` | `serve.go` disabled message | re-derived reason for the D-12 stderr string | ✓ WIRED | `serve.go:130` |
| `daemon.RunWithRetry` | `serve.go` goroutine | `serveWatchStart`'s background goroutine | ✓ WIRED | `serve.go:119` |
| `mcp-go` stdio client | spawned `serve --mcp` binary | `transport.WithCommandFunc` sets real `cmd.Dir` | ✓ WIRED | Confirmed by reading `worktree_notice_test.go` and by the mutation test passing/failing correctly |
| `graphstore.ErrStoreLocked` | `daemon.Run` requeue / `serve.go` reconcile downgrade | `errors.Is` sentinel (review-added, CR-01 fix) | ✓ WIRED | Read `daemon.go:214` and `serve.go:201` directly; both use the sentinel, not chain-sniffing |

### Behavioral Spot-Checks / Test Execution (run by me, not sourced from SUMMARY)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full build | `go build ./...` | clean | ✓ PASS |
| Full suite (cached) | `go test ./...` | all green including `test/integration` | ✓ PASS |
| Golden parity suite | `go test ./testdata/golden/... -count=1` | green, 12.7s | ✓ PASS |
| Subprocess integration harness | `go test ./test/integration/... -count=1 -v` | 4/4 tests pass (`TestDefaultWatchHandshakePrompt`, `TestNoWatchEnvDisablesViaStderr`, `TestLiveEditAutoSyncReachesExplore`, `TestWorktreeNoticeReachesServeMCPExplore`) | ✓ PASS |
| Concurrency race detector (CI's exact command) | `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` | green | ✓ PASS |
| Windows lock-classifier typecheck (CI's exact command) | `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` | clean | ✓ PASS |
| **CR-01 mutation test (performed live, not read from SUMMARY)** | reverted `BuildServer`'s 4th arg to `repoPath`, reran anchor test | test failed as expected (glyph missing) | ✓ MUTATION-PROOF CONFIRMED |
| Isolated daemon suite (CI's exact command) | `go test ./internal/daemon/ -count=1` ×4 | green every run | ✓ PASS |
| Full-suite parallel run (`go test ./... -count=1`, not CI's actual command) | — | `internal/daemon.TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` failed once under full-package-parallel disk contention | ⚠️ see note below — pre-existing, documented, not phase-3-introduced |

**Note on the reproduced daemon flake:** `go test ./... -count=1` (all packages in parallel) triggered a failure in `internal/daemon`, a package explicitly documented as flaky under full-suite parallel disk contention since the archived v0.1 Phase 8 (`08-CONTEXT.md` line 143: "`internal/daemon` `TestSoak` + flush-lock tests are known pre-existing flakes under full-suite parallel load"). CI does **not** run this command — it isolates `internal/daemon` into its own `-count=1` step precisely because of this documented, pre-existing condition (`.github/workflows/ci.yml:86-93`). Running CI's actual isolated command 4 times (including once under `-race`) was clean every time. This is not a phase-3 regression and does not block the phase goal.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| WATCH-01 | 03-03, 03-05 | Default-on watcher, `--no-watch` opt-out | ✓ SATISFIED | serve.go read directly; subprocess test passes |
| WATCH-02 | 03-02, 03-03, 03-05 | Off-handshake-path watcher startup | ✓ SATISFIED | Structural seam + mutation-proof unit test + subprocess handshake-prompt test, all run green |
| WATCH-03 | 03-01, 03-02 | WSL2/env watch-policy | ✓ SATISFIED | policy.go read directly; unit tests + subprocess NO_WATCH test green |
| WATCH-04 | 03-02 | Concurrent-session convergence, goleak-clean | ✓ SATISFIED | `TestConvergenceTwoSessions` read + run green under `-race` |
| TEST-04 | 03-04, 03-05 | Subprocess integration harness + CR-01 anchor + CI wiring | ✓ SATISFIED | 4 integration tests run green; CI steps present and functionally verified; mutation-proof personally reproduced |

No orphaned requirements: all 5 IDs (WATCH-01..04, TEST-04) declared across the 5 plans' `requirements:` frontmatter match REQUIREMENTS.md's Phase 3 mapping exactly.

### Anti-Patterns Found

None. Scanned every phase-touched file (`internal/watch/policy.go`, `internal/watch/policy_test.go`, `internal/daemon/daemon.go`, `internal/daemon/daemon_test.go`, `internal/daemon/soak_test.go`, `internal/cli/serve.go`, `internal/cli/serve_test.go`, `internal/cli/daemon.go`, `internal/graphstore/pebble_store.go`, `internal/graphstore/locked_unix.go`, `internal/graphstore/locked_windows.go`, `internal/graphstore/open_lock_test.go`, `internal/graphstore/locked_unix_test.go`, `internal/graphstore/locked_windows_test.go`, `test/integration/*.go`, `.github/workflows/ci.yml`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` — zero matches.

### Review Findings Disposition

Three review rounds ran against this phase (03-REVIEW.md: 1 critical + 4 warnings; 03-REVIEW-2.md: 1 critical + 2 warnings; 03-REVIEW-3.md: 0 critical + 1 warning, targeted delta review). All Criticals and Warnings across all three rounds were fixed, each with a commit and a fix report (03-REVIEW-FIX.md iteration 1: 5/5 fixed; 03-REVIEW-FIX-2.md iteration 2: 3/3 fixed; round 3's single Warning fixed at commit `3a4c2f6`, confirmed present in `.github/workflows/ci.yml` and independently re-run clean by me). Remaining Info-level items across all rounds (recompute-vs-carry reason duplication, requeue-log off-by-one, TOCTOU shutdown-latency, `hasIndex` startup snapshot, windows AV-sharing-violation imprecision, a test-comment provenance error, a timing-margin note) are explicitly documented as accepted, non-blocking residuals in the review reports themselves — none affect the phase's observable truths.

### Human Verification Required

None. Every observable truth had either a direct read of production code, a freshly-run automated test, or a personally-performed mutation test as evidence.

### Gaps Summary

No gaps. All 5 roadmap Success Criteria / requirement IDs are verified against the current (post-three-review-round) codebase, not against the plan text or SUMMARY claims. The one anomaly encountered during verification (a daemon-package flake under an atypical full-parallel test invocation) is a documented pre-existing condition unrelated to this phase's changes, correctly isolated by CI's actual test steps, and confirmed clean when run via CI's real commands.

---

_Verified: 2026-07-16T16:03:25Z_
_Verifier: Claude (gsd-verifier)_
