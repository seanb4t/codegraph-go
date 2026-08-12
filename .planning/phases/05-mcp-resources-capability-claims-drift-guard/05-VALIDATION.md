---
phase: 5
slug: mcp-resources-capability-claims-drift-guard
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-12
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none — `go test ./...` via `Taskfile.yml`'s `test`/`test:wireoracle` tasks |
| **Quick run command** | `go test ./internal/mcp/...` |
| **Full suite command** | `go test ./...` and `task test:wireoracle` (`go test ./test/wireoracle/...`) |
| **Estimated runtime** | ~30 seconds (unit) / ~60 seconds (full incl. wire-oracle) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/mcp/...`
- **After every plan wave:** Run `go test ./... && task test:wireoracle`
- **Before `/gsd-verify-work`:** Full suite must be green, including a demonstrated-red-then-reverted mutation proof appended to `test/wireoracle/MUTATION-PROOF.md` for GUARD-02 (success criterion 3)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 0 | RSRC-01 | — | N/A | unit | `go test ./internal/mcp/... -run TestResource` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 0 | RSRC-02 | — | N/A | unit | `go test ./internal/mcp/... -run TestResource` | ❌ W0 | ⬜ pending |
| 05-01-03 | 01 | 0 | RSRC-03 | — | N/A | unit + wire | `go test ./internal/mcp/... -run TestResourcesRegisterWithoutIndex` / `task test:wireoracle` | ❌ W0 | ⬜ pending |
| 05-01-04 | 01 | 0 | GUARD-01 | — | N/A | unit | `go test ./internal/mcp/... -run TestMCPResourceClaimsMatchEngineConstants` | ❌ W0 | ⬜ pending |
| 05-01-05 | 01 | 0 | GUARD-02 | — | N/A | unit | `go test ./internal/mcp/... -run TestResourceSetMatchesCompanionNames` | ❌ W0 | ⬜ pending |
| 05-01-06 | 01 | 1 | RSRC-01/02/03 | — | N/A | wire | `task test:wireoracle` (re-frozen transcripts) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Exact task IDs are provisional — the planner assigns final task IDs; this table's requirement→command mapping is the binding contract.*

---

## Wave 0 Requirements

- [ ] `internal/mcp/resources/*.md` — the 10 fact-sheet/behavior-doc source files (content authoring, D-01 through D-10)
- [ ] `internal/mcp/resources.go` — `go:embed` directive, `resourceURIFor` map, `registerResources(s)` function
- [ ] `internal/mcp/resources_test.go` — unit-level list/read coverage (mirrors `server_test.go`'s `newTestSession`/`listToolNames` pattern)
- [ ] `internal/mcp/resources_schema_drift_test.go` (or extend `tools_schema_drift_test.go` directly) — GUARD-01/02 extension
- [ ] `test/wireoracle/scenarios.go` additions — new request helpers + new `Scenario` entries + `ExpectedScenarioCount` bump

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
