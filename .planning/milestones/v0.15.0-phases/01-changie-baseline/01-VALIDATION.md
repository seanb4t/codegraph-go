---
phase: "1"
slug: "changie-baseline"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-25"
validated: "2026-09-25"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go 1.26.6 pinned via `go.mod`; local Go 1.27.1 needs `GOTOOLCHAIN=go1.26.6` for whole-module runs) plus Taskfile drift-guard targets |
| **Config file** | `Taskfile.yml` (`check:changie`, `changie`, `vuln`, `lint:actions`), `go.tool-changie.mod` |
| **Quick run command** | `GOWORK=off go test ./internal/upgrade/ -run '^TestChangie' -count=1` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go test -p 1 -count=1 ./...` (serial form is the trustworthy gate; see spine `whad9x6gxq`) |
| **Estimated runtime** | ~2 s quick · ~90 s `task check:changie` · ~4 min full suite |

---

## Sampling Rate

- **After every task commit:** Run `GOWORK=off go test ./internal/upgrade/ -run '^TestChangie' -count=1`
- **After every plan wave:** Run `GOTOOLCHAIN=go1.26.6 go test -p 1 -count=1 ./...` and `task check:changie`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | CHG-01, CHG-02 | T-01-01 (supply chain: pinned changie via go.tool-changie.mod + go.sum) | Tool built only from the checksum-verified module proxy at v1.26.0 | unit | `GOWORK=off go test ./internal/upgrade/ -run '^(TestChangie\|TestToolModfiles)' -count=1 -v` | ✅ | ✅ green |
| 1-01-01 | 01 | 1 | CHG-02 | — | N/A | integration | `task changie -- merge --dry-run \| cmp - CHANGELOG.md` and `git diff --quiet main -- CHANGELOG.md` | ✅ | ✅ green |
| 1-01-02 | 01 | 1 | CHG-01 | — | N/A | unit | `GOWORK=off go test ./internal/upgrade/ -run '^TestContributingReferencesRealTaskTargets$' -count=1 -v` | ✅ | ✅ green |
| 1-02-01 | 02 | 2 | CHG-03, CHG-04 | T-02-01 (scratch-copy isolation: guard never mutates the source tree) | Before/after checksum snapshot proves tree unchanged | integration | `task check:changie` (asserts `11 of 11 checks passed`) | ✅ | ✅ green |
| 1-02-02 | 02 | 2 | CHG-01, CHG-03 | T-02-02 (CI step adds no permissions/secrets/uses) | actionlint clean; step invokes only `task check:changie` | unit | `GOWORK=off go test ./internal/upgrade/ -run '^(TestChangieCheckWiredIntoCI\|TestChangieBinaryInToolVulnScan\|TestWorkflowRunBodiesInvokeTask\|TestGateStancesStated)$' -count=1 -v` and `task lint:actions` | ✅ | ✅ green |
| 1-02-03 | 02 | 2 | CHG-03, CHG-04 | — | N/A | other | Mutation log assertions (`rg` over `01-MUTATION-LOG.md`, 4 families) then `task check:changie` green on the main tree | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Requirement coverage (both directions):

| Requirement | Named tests | Status |
|-------------|-------------|--------|
| CHG-01 | `TestChangieConfigShape`, `TestChangieToolPinnedInIsolatedModfile`, `TestChangieWrapperTaskRecordsInstallPath`, `TestChangieBinaryInToolVulnScan`, `TestToolModfilesRemainIsolated`, `TestToolModfilesPopulationMatchesDisk` | COVERED |
| CHG-02 | `TestChangieBaselineLayout`, `TestChangieVersionSeedsMatchChangelog`, `task changie -- merge --dry-run \| cmp` | COVERED |
| CHG-03 | `task check:changie` legs 1–3, `TestChangieCheckWiredIntoCI`, `TestTaskfileGatesFailLoud` | COVERED |
| CHG-04 | `task check:changie` legs 4–9 (non-interactive write, missing PR, undeclared kind, PR below minInt, non-integer PR, sub-second collision) | COVERED |

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. The RED-first guards in `internal/upgrade/changie_shape_test.go` (commits `01057ac1`, `1d023e70`) were the Wave 0 stubs and are now green.

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-09-25 (auto mode; verifier 4/4, code review 0 critical)

## Validation Audit 2026-09-25
| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |
