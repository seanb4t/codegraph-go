---
phase: 3
slug: non-vacuity-proof-unconditional-ci-execution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-15
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), plus `go-sdk` MCP in-process client for the CLI==MCP trio |
| **Config file** | none — `Taskfile.yml` is the single CI definition (`TestWorkflowRunBodiesInvokeTask`) |
| **Quick run command** | `go test -count=1 ./testdata/golden/...` |
| **Full suite command** | `task test:golden` (needs `-count=1` added — S5 gap) |
| **Estimated runtime** | quick ~25s (indexes locked corpora) |

> **`-count=1` is mandatory** — the current `test:golden` task lacks it (S5); a cached `ok` could mask a regression.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./testdata/golden/...`
- **After every plan wave:** `go test -count=1 ./testdata/golden/... ./internal/upgrade/...` (golden + taskfile-shape)
- **Before `/gsd-verify-work`:** `go test -count=1 ./...` green, and each FIXT-07 family has its recorded RED demonstration
- **Max feedback latency:** ~30s

---

## Per-Task Verification Map

> Seeded from the requirement→test map; task IDs assigned by the planner.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | FIXT-03 | — | N/A | unit | `go test -count=1 ./testdata/golden/...` (zero executed scenarios → RED) | ✅ (`golden_test.go` extends) | ⬜ pending |
| TBD | TBD | TBD | FIXT-03 | — | N/A | unit | `go test -count=1 ./testdata/golden/...` executed count == expected (self-asserted) | ✅ (extend `TestReFrozenGoldensValid`) | ⬜ pending |
| TBD | TBD | TBD | FIXT-07 | — | N/A | per-family mutation/revert | each family's recorded RED demonstration | ❌ W0 (mutation log) | ⬜ pending |
| TBD | TBD | TBD | FIXT-03 | — | N/A | CI | corpora.yml golden job runs unconditional; fetch/cache failure fails loudly | ❌ W0 (new job) | ⬜ pending |
| TBD | TBD | TBD | FIXT-03 | — | N/A | static | `task --list` shows the golden job; `TestWorkflowRunBodiesInvokeTask` green with the new `inScopeJobs` entry | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] The executed-count self-assertion folded into the golden test (extend `TestReFrozenGoldensValid` to cover the 4 CASES.json cases: 26 → 30, with the exact-equality form)
- [ ] The unconditional golden job in `corpora.yml` + the `inScopeJobs` entry
- [ ] The widened path filter in `corpora.yml` (S3 gap: add `testdata/golden/**`, `corpus/**`, and the query/CLI/MCP surfaces)
- [ ] `-count=1` added to the `test:golden`/`golden` Taskfile target (S5 gap)
- [ ] The FIXT-07 mutation log / per-family recorded RED demonstrations
- [ ] `test:golden` removed from `ci.yml`'s test job (D-04)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Each FIXT-07 family's RED demonstration is genuine (mutation applied, observed failure, byte-clean revert) | FIXT-07 | Requires applying a real mutation to the suite and watching the family fail, then reverting — a rehearsal, not a test | Per family: apply the mutation, run the family, observe RED, `git checkout --` / revert byte-clean, record mutation + failure + revert |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Every test command forces `-count=1`
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
