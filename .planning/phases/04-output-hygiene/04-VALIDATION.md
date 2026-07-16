---
phase: 4
slug: output-hygiene
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-16
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard go toolchain |
| **Quick run command** | `go test ./internal/graphstore/... ./internal/mcp/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` |
| **Estimated runtime** | ~120 seconds (full, incl. integration harness binary build) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/graphstore/... ./internal/mcp/...`
- **After every plan wave:** Run `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| T1 quietLogger + diagWriter | 04-01 | 1 | HYG-01 | T-04-01 | Infof discarded; Errorf/Fatalf preserved via diagWriter seam (D-02/D-04) | unit | `go test ./internal/graphstore/ -run 'TestQuietLogger' -v` | ❌ Wave 0 (new logger.go/logger_test.go) | ⬜ pending |
| T2 Open injects quietLogger | 04-01 | 1 | HYG-01 | T-04-02 | Logger wired at single Open seam, mutation-proof; no Infof noise to seam (D-01/D-08) | unit | `go test ./internal/graphstore/ -run 'TestOpenInjectsQuietLogger\|TestQuietLoggerSilencesStoreActivity' -v` | ❌ Wave 0 | ⬜ pending |
| T1 detector predicates + self-test | 04-02 | 1 | HYG-02 | T-04-03 | os.Stdout/bare fmt.Print*/log.SetOutput flagged via go/types, not regex (D-06b) | unit | `go test ./internal/graphstore/archtest/ -run 'TestStdoutGuardDetectsViolations' -v` | ❌ Wave 0 (new archtest files) | ⬜ pending |
| T2 six-package stdout archtest | 04-02 | 1 | HYG-02 | T-04-03 | serve-reachable packages free of stdout writes; rename-proof sanity check (D-06b/D-07) | unit (archtest) | `go test ./internal/graphstore/archtest/ -run 'TestNoStdoutNoiseInServeReachablePackages' -v` | ❌ Wave 0 | ⬜ pending |
| T1 raw-stdio frame purity | 04-03 | 2 | HYG-02 | T-04-04 | every `serve --mcp` stdout line is a JSON-RPC frame; not built on mcp-go client (D-06a) | integration (subprocess) | `go test ./test/integration/ -run 'TestServeMCPStdoutIsPureJSONRPC' -v` | ❌ Wave 0 (new mcp_stdout_purity_test.go) | ⬜ pending |
| T2 sync stderr noise-absence | 04-03 | 2 | HYG-01 | T-04-05 | driven `sync` stderr carries no Pebble noise shapes; not emptiness (D-09) | integration (subprocess) | `go test ./test/integration/ -run 'TestSyncStderrNoPebbleNoise' -v` | ❌ Wave 0 (new sync_noise_test.go) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements (go test + testdata/golden + test/integration harness all present from Phases 1–3).

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
