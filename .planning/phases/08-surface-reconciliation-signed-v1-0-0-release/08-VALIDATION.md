---
phase: 8
slug: surface-reconciliation-signed-v1-0-0-release
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-19
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./internal/cli/... ./internal/query/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/...` |
| **Estimated runtime** | ~60–120 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/cli/... ./internal/query/...`
- **After every plan wave:** Run `go test ./... && go test ./testdata/golden/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 8-01-01 | 01 | 1 | SURF-01 | — | N/A | unit | `go test ./internal/query/...` | ❌ W0 | ⬜ pending |

*Seeded draft — validate-phase (§6) populates the full per-task map from the finalized PLAN.md files. Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Populated by validate-phase from PLAN.md `<automated>` verify blocks. If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Actual `v1.0.0` tag push → signed release build | REL-02 | Tag push is the maintainer's manual runbook action; triggers CI signing pipeline | Follow `docs/RELEASE-PROCEDURES.md`; verify with `cosign verify-blob` + `slsa-verifier` |
| Published head-to-head benchmark numbers | REL-03 | Requires running the bench harness against a live TS 1.3.1 install and publishing results | Run `bench.yml` methodology; refresh `docs/BENCHMARKS.md` |

*Draft — validate-phase reconciles against the finalized plans.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
