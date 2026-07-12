---
phase: 7
slug: migration-tool
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-12
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — Go modules, standard `go test` |
| **Quick run command** | `go test ./internal/migrate/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~60–120 seconds (full); `internal/daemon` TestSoak is a known pre-existing flake under parallel load — re-run `go test ./internal/daemon/ -count=1` isolated before treating a daemon failure as a regression |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/migrate/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

> Populated by the planner (per-task `<verify>` blocks) and execution. Anchors: every task tied to MIGR-01 / MIGR-02 must carry an automated `go test` command or a Wave 0 fixture dependency.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 7-01-01 | 01 | 1 | MIGR-01 | — | reader opens source SQLite read-only; source bytes never mutated | unit | `go test ./internal/migrate/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Aged-`.codegraph/` SQLite fixture reconstruction harness — build a real `*.db` in-Go from `testdata/golden/ts-schema.dump.sql` (no external `sqlite3` dependency) so the migrator has a real source DB to run against (MIGR-02 SC-3; see RESEARCH F1–F4)
- [ ] `internal/migrate` test scaffolding + shared fixtures for round-trip (read TS → write new format → assert invariants)
- [ ] `modernc.org/sqlite` added to `go.mod` (read-only reader dep)

*Framework exists (`go test`); Wave 0 is fixture + dependency setup, not framework install.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Migrate a genuinely aged real-world TS `.codegraph/` (predating later schema columns) end-to-end | MIGR-02 | A truly aged production index cannot be fully synthesized; the reconstructed fixture approximates but a live aged dir is the ground truth | If a real aged `.codegraph/` is available, run `codegraph migrate --from <aged-dir>` and confirm validation passes / fails loudly on corruption |

*Automated coverage handles reconstructed-fixture migration + all invariant checks; only the live-aged-dir confirmation is manual.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references (aged-DB fixture harness)
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
