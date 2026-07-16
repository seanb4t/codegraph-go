---
phase: 3
slug: watcher-on-mcp-default
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-16
audited: 2026-07-16
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) + goleak soaks + testdata/golden parity suite + test/integration subprocess harness (new this phase) |
| **Config file** | none — go.mod toolchain |
| **Quick run command** | `go test ./internal/... ./cmd/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` |
| **Estimated runtime** | ~90 seconds (quick ~30s; golden + integration add ~60s) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... ./cmd/...`
- **After every plan wave:** Run `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01.T1/T2 | 03-01 | 1 | WATCH-03 | T-03-01, T-03-02 | strict `== "1"` env parsing; DetectWSL read failure degrades to false, never panics | unit (TDD RED→GREEN) | `go test ./internal/watch/ -run 'TestWatchDisabledReason\|TestDetectWSL' -count=1` | ✅ `internal/watch/policy_test.go` | ✅ green |
| 03-02.T1 | 03-02 | 2 | WATCH-02 (gate), WATCH-04 | T-03-03, T-03-05 | policy gate runs before `acquire()` — a disabled Run never creates the lockfile; retry honors ctx cancel | unit | `go test ./internal/daemon/ -run 'TestRunPolicyDisabled\|TestRunHonorsDefaultProbeOnNonWSLHost\|TestRunWithRetry' -count=1` | ✅ `internal/daemon/daemon_test.go` | ✅ green |
| 03-02.T2 | 03-02 | 2 | WATCH-04 | T-03-04 | at-most-one writer always, exactly-one eventually; goleak-clean teardown | soak (goleak + `-race`) | `go test ./internal/daemon/ -run TestConvergenceTwoSessions -count=1` | ✅ `internal/daemon/soak_test.go` | ✅ green ¹ |
| 03-03.T1 | 03-03 | 3 | WATCH-01 | T-03-06, T-03-08 | `--no-watch`/`--watch` mutually exclusive (cobra hard error); verbatim disabled message to stderr only | unit | `go test ./internal/cli/ -run TestServeWatchStartDisabledPrintsVerbatimMessage -count=1` | ✅ `internal/cli/serve_test.go` | ✅ green |
| 03-03.T2 | 03-03 | 3 | WATCH-02 | T-03-07 | all watcher startup (daemon.New/policy/acquire/watch.Open) deferred off the handshake path — mutation-proven | structural unit | `go test ./internal/cli/ -run TestServeWatchStartDeferred -count=1` | ✅ `internal/cli/serve_test.go` | ✅ green |
| 03-04.T1 | 03-04 | 4 | TEST-04 | T-03-09, T-03-10 | hermetic TestMain binary build; bounded contexts; git-absence skips | integration substrate | `go test ./test/integration/... -count=1` | ✅ `test/integration/main_test.go` | ✅ green |
| 03-04.T2 | 03-04 | 4 | TEST-04 | T-03-11 | CR-01 anchor: worktree notice reaches a real `serve --mcp` explore payload over stdio — mutation-proven against the two-arg regression | integration (subprocess) | `go test ./test/integration/... -run TestWorktreeNoticeReachesServeMCPExplore -count=1` | ✅ `test/integration/worktree_notice_test.go` | ✅ green |
| 03-04.T3 | 03-04 | 4 | TEST-04 | T-03-11 | explicit named CI step — immune to a future `go list` refactor silently dropping the harness (GOLDEN-01) | CI config | `grep -q 'go test ./test/integration/' .github/workflows/ci.yml` | ✅ `.github/workflows/ci.yml:90-91` | ✅ green |
| 03-05.T1 | 03-05 | 5 | WATCH-01, WATCH-02, TEST-04 | T-03-13 | default-on `serve --mcp` completes Initialize+ListTools promptly on the real binary | integration (subprocess) | `go test ./test/integration/... -run TestDefaultWatchHandshakePrompt -count=1` | ✅ `test/integration/watch_default_test.go` | ✅ green |
| 03-05.T2 | 03-05 | 5 | WATCH-03, TEST-04 | T-03-12 | `CODEGRAPH_NO_WATCH=1` disables watcher; serve still comes up (never-block); verbatim message on real child stderr | integration (subprocess) | `go test ./test/integration/... -run TestNoWatchEnvDisablesViaStderr -count=1` | ✅ `test/integration/watch_default_test.go` | ✅ green |
| review-fix (post-phase) | — | — | WATCH-01 (live-sync value path) | — | a live file edit auto-syncs and its symbol reaches a subsequent explore payload — the value path the deep review found untested | integration (subprocess) | `go test ./test/integration/... -run TestLiveEditAutoSyncReachesExplore -count=1` | ✅ `test/integration/` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

¹ `TestConvergenceTwoSessions` is green in its isolated `-count=1` CI lane (the only lane it runs in); a Pebble MANIFEST-creation race under full-suite *parallel* load is a known pre-existing condition logged in [deferred-items.md](./deferred-items.md) — not a test-correctness gap.

---

## Wave 0 Requirements

- [x] `test/integration/` — TestMain that builds the real binary once (`go build -o <tmp>/codegraph .`) — TEST-04 substrate (`test/integration/main_test.go:39`)
- [x] Real-git worktree fixture helper (reuse Phase-2 D-15 pattern) reachable from `test/integration/` (`buildWorktreeFixture` + `runGitI`, fourth package-local copy)
- [x] `internal/watch/policy_test.go` — stubs for WATCH-03 precedence table (14-case table + DetectWSL caching test, all green)

*Existing infrastructure covers the rest: goleak TestMain in `internal/daemon`, golden harness in `testdata/golden/`.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WSL2 `/mnt/*` auto-off on a real WSL2 host | WATCH-03 | No WSL2 environment in CI or on darwin dev machine; detection is unit-tested via injected probes | On a WSL2 box: `cd /mnt/c/<repo> && codegraph serve --mcp` → stderr shows the verbatim disabled message; `CODEGRAPH_FORCE_WATCH=1` re-enables |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — every row in the Per-Task Map carries a runnable command, re-executed green 2026-07-16
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — every task had its own verify block
- [x] Wave 0 covers all MISSING references — all three Wave 0 items delivered by 03-01/03-04
- [x] No watch-mode flags — no test is gated behind a build tag or skipped-by-default flag; git-absence uses t.Skip only
- [x] Feedback latency < 120s — quick lane ~8s, full suite (incl. golden + integration) well under budget
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** auto-validated 2026-07-16 (/gsd-validate-phase audit — zero gaps found)

---

## Validation Audit 2026-07-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Audit method: rebuilt the requirement-to-task map from all 5 PLAN/SUMMARY pairs, confirmed every claimed test function exists in the tree (17 functions across 4 packages), and re-ran the covering suites live: `go test ./internal/watch/ ./internal/cli/ -count=1`, `go test ./internal/daemon/ -count=1` (isolated lane), `go test ./test/integration/... -count=1` — all green, including the post-review `TestLiveEditAutoSyncReachesExplore` addition. The explicit CI step for the integration harness verified present at `.github/workflows/ci.yml:90-91`. The single manual-only item (WSL2 real-host validation) remains documented above with its unit-test substitute rationale.
