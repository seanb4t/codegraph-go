---
phase: 04-output-hygiene
verified: 2026-07-16T23:45:00Z
status: passed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 4: Output Hygiene Verification Report

**Phase Goal:** No library log noise ever pollutes command output or the MCP transport — Pebble's WAL/INFO chatter is routed away while real errors survive, and MCP stdout stays clean JSON-RPC.
**Verified:** 2026-07-16T23:45:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Pebble's internal WAL/INFO log noise no longer prints on any command, while real errors are preserved (HYG-01, roadmap SC1) | ✓ VERIFIED | `internal/graphstore/logger.go`: `quietLogger.Infof` is an empty body (unconditional discard); `Errorf`/`Fatalf` route through `writeDiagLine` to the `diagWriter` seam. `pebble_store.go:147` injects `Logger: quietLogger{}` at the sole `pebble.Open` seam. `TestOpenInjectsQuietLogger` (a real Open/write/flush/close cycle) passes green. Independently re-verified: manually reverted line 147 to `&pebble.Options{}` — test genuinely failed with real captured noise (`"Found 0 WALs"`); restored — test passed again; `git status` clean after. |
| 2 | No library log output ever reaches MCP stdout — JSON-RPC framing stays clean; diagnostics go to stderr only (HYG-02, roadmap SC2) | ✓ VERIFIED | Structural: `TestNoStdoutNoiseInServeReachablePackages` scans the transitive module-internal import closure (not just the six named roots — CR-01 fix) of `internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query` via go/types-resolved predicates; green today. Runtime: `TestServeMCPStdoutIsPureJSONRPC` spawns the real `serve --mcp` binary, owns `cmd.StdoutPipe()` directly (not the tolerant mcp-go client — RESEARCH Pitfall 1 avoided), and asserts every line is a valid JSON-RPC frame through a startup reconcile + a store-opening `tools/call`. Both pass, including under `-race`. |
| 3 | The HYG-01 wiring test is mutation-proof — reverting the injection turns it red (D-08) | ✓ VERIFIED | Independently reproduced (not just trusting SUMMARY): reverted `pebble_store.go:147`, re-ran `TestOpenInjectsQuietLogger`, got a genuine failure with real Pebble noise bytes (`"Found 0 WALs"`) in the failure message; restored and confirmed green again; `git status --porcelain` empty after. |
| 4 | The stdout-confinement archtest is provably able to fail, including on violations in transitive dependencies, not just the six named packages (D-07/Pitfall 4, CR-01) | ✓ VERIFIED | `TestStdoutGuardCatchesViolationsInTransitiveDependency` plants a synthetic `fmt.Println` in `internal/schema` (a transitive dependency of every guarded package, not one of the six roots itself) via an in-memory `packages.Config.Overlay` and asserts the closure scan flags it. Ran directly: passes. `TestStdoutGuardDetectsViolations` proves the three detection predicates flag real constructs and ignore compliant ones via synthetic fixtures. Ran directly: passes. |
| 5 | The frame-purity harness is provably able to fail on a real stdout violation, not layered on a client that silently tolerates malformed lines (D-06a) | ✓ VERIFIED | Source inspection confirms `mcp_stdout_purity_test.go` uses plain `exec.Command` + `cmd.StdoutPipe()` + `bufio.Scanner`, never `newServeClient`/`mcpclient.Client`. `04-REVIEW.md`'s clean re-review independently confirmed via `-race -count=5` runs that this is genuinely race-free and fail-capable; `04-03-SUMMARY.md` documents an injected-and-reverted `fmt.Println` mutation that the test caught with the exact expected message. I independently re-ran the test (including `-race`) and it passes on current tree. |
| 6 | `internal/cli` remains excluded from the HYG-02 guard (D-06b) | ✓ VERIFIED | `excludedInternalPackagePrefixes` in `stdout_confinement_test.go` explicitly excludes `internal/cli` with an inline rationale comment; `isModuleInternalPackage` filters it out of the closure walk. |
| 7 | Pebble Errorf/Fatalf diagnostics are never discarded or softened — only Infof is (D-02) | ✓ VERIFIED (prohibition, judgment) | `logger.go`: `Infof` has an empty body; `Errorf` and `Fatalf` both call `writeDiagLine` (never no-op'd); `Fatalf` still calls `os.Exit(1)` after formatting — pebble's fatal semantics are unchanged, not softened. |
| 8 | No new env/flag escape hatch re-enables Pebble INFO logging (D-05) | ✓ VERIFIED (prohibition, judgment) | `rg` over `internal/graphstore/logger.go` and `pebble_store.go` finds no `os.Getenv`/flag reference tied to logging; `Infof` discard is unconditional with no branch. |
| 9 | The D-09 noise check asserts absence of noise shapes only, never stderr emptiness | ✓ VERIFIED (prohibition, judgment) | `sync_noise_test.go`'s `TestSyncStderrNoPebbleNoise` iterates a fixed needle list (`[JOB `, `WAL `, `compaction`, `pickAuto`) with `strings.Contains` — no assertion that stderr length is zero. |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/graphstore/logger.go` | quietLogger + diagWriter seam | ✓ VERIFIED | Exists, substantive, compiles, wired at `pebble_store.go:147`, race-safe (`diagWriterMu` RWMutex — WR-03 fix) |
| `internal/graphstore/logger_test.go` | Behavioral + mutation-proof tests | ✓ VERIFIED | 5 tests, all pass, mutation-proof independently re-verified |
| `internal/graphstore/pebble_store.go` | One-field Logger injection at Open | ✓ VERIFIED | Line 147 only change; CR-01 lock-retry loop untouched (confirmed via diff and manual mutation test) |
| `internal/graphstore/archtest/stdout_confinement_test.go` | Build-time stdout-confinement archtest | ✓ VERIFIED | Transitive-closure walk (post-CR-01 fix), passes, `go vet` clean |
| `internal/graphstore/archtest/stdout_detection_selftest_test.go` | Detector self-test | ✓ VERIFIED | Positive/negative fixtures, passes |
| `internal/graphstore/archtest/stdout_closure_selftest_test.go` | CR-01 closure regression lock | ✓ VERIFIED | Plants violation in `internal/schema` (a dependency, not a root), passes |
| `test/integration/mcp_stdout_purity_test.go` | Raw-stdio JSON-RPC frame-purity harness | ✓ VERIFIED | Passes, including under `-race`; WR-01/WR-02/scanner.Err() fixes present and correct |
| `test/integration/sync_noise_test.go` | CLI-side Pebble-noise-absence check | ✓ VERIFIED | Passes; absence-of-substring only, not emptiness |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `pebble_store.go` `Open` | `quietLogger{}` | `&pebble.Options{Logger: quietLogger{}}` at line 147 | ✓ WIRED | Confirmed by source read + mutation test (revert → red, restore → green) |
| `quietLogger.Errorf`/`Fatalf` | `diagWriter` seam | `writeDiagLine` → `getDiagWriter()` | ✓ WIRED | `TestQuietLoggerErrorfWritesProvenance`/`TestQuietLoggerSilencesStoreActivity` pass |
| `stdout_confinement_test.go` closure walk | six guarded packages' transitive deps | `packages.NeedDeps` + recursive `pkg.Imports` walk | ✓ WIRED | `TestStdoutGuardCatchesViolationsInTransitiveDependency` proves a violation in `internal/schema` (a dependency) is caught |
| `mcp_stdout_purity_test.go` | real `serve --mcp` subprocess stdout | raw `cmd.StdoutPipe()` + `bufio.Scanner`, not mcp-go client | ✓ WIRED | Source confirms no `mcpclient`/`newServeClient` used for the purity assertion |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| HYG-01 | 04-01, 04-03 | Pebble WAL/INFO log noise routed away; real errors preserved | ✓ SATISFIED | `quietLogger` + wiring + mutation-proof test + `TestSyncStderrNoPebbleNoise` end-to-end |
| HYG-02 | 04-02, 04-03 | No library log output reaches MCP stdout | ✓ SATISFIED | Structural archtest (closure-fixed) + `TestServeMCPStdoutIsPureJSONRPC` runtime proof |

No orphaned requirements — REQUIREMENTS.md maps exactly HYG-01/HYG-02 to Phase 4, and both are declared and satisfied by the phase's plans.

### Anti-Patterns Found

None. Scanned all 8 phase-touched files for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|not yet implemented|coming soon` — zero matches. `go vet ./internal/graphstore/... ./test/integration/...` clean.

### Behavioral Spot-Checks / Mutation-Proof Checks (independently re-run, not trusting SUMMARY claims)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| HYG-01 wiring mutation-proof | Manually reverted `pebble_store.go:147` to `&pebble.Options{}`, ran `TestOpenInjectsQuietLogger`, restored | Genuine failure with real "Found 0 WALs" noise bytes on revert; pass on restore; `git status` clean after | ✓ PASS |
| HYG-02 unit suite | `go test ./internal/graphstore/... -run 'TestQuietLogger\|TestOpenInjectsQuietLogger' -v` | All 5 tests pass | ✓ PASS |
| HYG-02 archtest suite | `go test ./internal/graphstore/archtest/... -v` | All 4 tests pass (including CR-01's transitive-dependency regression lock) | ✓ PASS |
| HYG-02 end-to-end frame purity | `go test ./test/integration/... -run TestServeMCPStdoutIsPureJSONRPC -v` and `-race` variant | Pass, both plain and under `-race` | ✓ PASS |
| HYG-01 end-to-end CLI noise | `go test ./test/integration/... -run TestSyncStderrNoPebbleNoise -v` | Pass | ✓ PASS |
| Full module suite | `go test ./...` | All packages pass | ✓ PASS |
| Golden suite | `go test ./testdata/golden/...` | Pass | ✓ PASS |
| `go build ./...` | clean | ✓ PASS |

Known pre-existing flake `internal/daemon` `TestDaemonFlushLockRequeueGivesUpPerEpisode` (timing-sensitive, predates this phase) was not encountered during this verification's full-suite run — not a regression concern.

### 6 Post-SUMMARY Review-Fix Commits Verified Present and Correct

The plans' SUMMARY.md files were written before a deep code review (iteration 1, `04-REVIEW.iter2.md`) found 1 critical + 3 warnings + 1 info issue, all fixed in 6 subsequent commits, then confirmed clean by a second deep review (`04-REVIEW.md`, status: clean). All 6 commits are present in git history and their fixes independently verified in the current tree:

| Commit | Finding | Verified |
|--------|---------|----------|
| `21e47b9` | CR-01: stdout guard didn't scan transitive dependencies | ✓ `closeOverServeReachableImports` uses `NeedDeps` + recursive walk; `TestStdoutGuardCatchesViolationsInTransitiveDependency` passes |
| `c2fe509` | WR-01: data race on `stderrBuf` | ✓ `syncBuffer` mutex-guarded wrapper present; `-race` clean |
| `cbd134d` | WR-02: tools/call error-blindness | ✓ Frame struct decodes `Error json.RawMessage`; `t.Fatalf` on non-empty error at id==2 |
| `9372385` | WR-03: unsynchronized `diagWriter` global | ✓ `diagWriterMu sync.RWMutex` + `getDiagWriter`/`setDiagWriter` accessors present |
| `68e7a91` | IN-01: doc addendum for indirect-write residual risk | ✓ Package doc comment in `stdout_confinement_test.go` documents the residual |
| `8a732cc` | scanner.Err() check missing | ✓ `scanErr` captured and surfaced in the `!ok` failure branch |

### Human Verification Required

None. All must-haves are objectively verifiable through code inspection, unit tests, mutation testing, and real-binary integration tests — no visual, UX, or external-service-dependent behavior in this phase's scope.

### Gaps Summary

No gaps. All roadmap Success Criteria, all plan-level must-haves (truths, artifacts, key links, prohibitions), and all requirement IDs (HYG-01, HYG-02) are verified against the actual codebase — not just the SUMMARY.md narrative. The phase's own deep-review cycle caught a genuine coverage gap (CR-01: the original stdout guard scanned only 6 root packages, missing ~20 transitively-imported packages that were provably reachable and provably invisible to the guard) and 4 other real issues; all 5 were fixed and re-verified clean in a second review round. This verification independently re-ran the mutation-proof checks (reverting the HYG-01 Open-seam injection; the review's own transitive-dependency violation) rather than trusting the SUMMARY/REVIEW claims, and confirmed the same results.

---

_Verified: 2026-07-16T23:45:00Z_
_Verifier: Claude (gsd-verifier)_
