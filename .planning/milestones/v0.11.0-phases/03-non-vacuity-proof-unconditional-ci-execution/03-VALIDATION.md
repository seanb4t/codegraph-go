---
phase: 3
slug: non-vacuity-proof-unconditional-ci-execution
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-15
validated: 2026-08-16
validation_mode: retroactive
automated_rows: 4
manual_only_rows: 1
tests_executed: 113
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
| **Full suite command** | `task test:golden` (carries `-count=1` — **S5 gap closed**) |
| **Estimated runtime** | quick ~25s (indexes locked corpora) |

> **`-count=1` is mandatory** — a cached `ok` could mask a regression. **Status corrected
> 2026-08-16:** this row previously read *"the current `test:golden` task lacks it (S5)"*. That was
> true when seeded and is now false — the target carries `-count=1` (verified: 1 occurrence). A
> plan-time gap note left standing after the gap closes becomes a stale fact frozen in prose, which
> is a documented failure mode in this repo; it is corrected here rather than preserved.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./testdata/golden/...`
- **After every plan wave:** `go test -count=1 ./testdata/golden/... ./internal/upgrade/...` (golden + taskfile-shape)
- **Before `/gsd-verify-work`:** `go test -count=1 ./...` green, and each FIXT-07 family has its recorded RED demonstration
- **Max feedback latency:** ~30s

---

## Per-Task Verification Map

> Reconciled retroactively by `/gsd-validate-phase 3` on 2026-08-16. Every command **executed**;
> counts recorded rather than inferred from exit status.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Result | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|--------|--------|
| 03-01-T1 | 03-01 | 1 | FIXT-03 | T-03-P1-08 | Zero executed scenarios → RED; both legs fatal at 0 before the sum | unit | `go test -count=1 ./testdata/golden/... -run TestGoldenScenarioCountIsExact` | **30/30** (goldens 26/26, cases 4/4) | ✅ green |
| 03-01-T2 | 03-01 | 1 | FIXT-03 | T-03-P1-08 | Executed count == expected, self-asserted from the enumeration **and** the filesystem | unit | `go test -count=1 ./testdata/golden/... -run TestReFrozenGoldensValid` | **26/26 goldens verified** | ✅ green |
| 03-01-T3 | 03-01 | 1 | FIXT-03 | T-03-P1-02, T-03-P1-03 | `corpora.yml` golden job runs unconditionally; fetch/cache failure fails loudly | CI (static shape) | `go test -count=1 ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask` | **1 PASS**; no `if:` gate on fetch/assert | ✅ green |
| 03-01-T4 | 03-01 | 1 | FIXT-03 | T-03-P1-04, T-03-P1-06 | `test:golden` runs only where corpora are fetched; `-count=1` defeats the test cache | static | `rg "test:golden" .github/workflows/ci.yml` → empty; Taskfile target carries `-count=1` | **0** in `ci.yml`, **2** in `corpora.yml`, `-count=1` present | ✅ green |
| 03-02-T1 | 03-02 | 2 | FIXT-07 | T-03-P2-01, T-03-P2-03 | Each family demonstrated RED: mutation applied → observed failure → byte-clean revert | **manual** (per-family mutation/revert) | Recorded in `03-MUTATION-LOG.md` | 5 families, 19 pasted FAIL blocks, 11 reverts | ✅ performed |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Full-package cross-check:** `go test -count=1 ./testdata/golden/...` → **79 PASS / 0 FAIL**;
`go test -count=1 ./internal/upgrade/... -run 'Taskfile|Workflow'` → **33 PASS / 0 FAIL**.
**Total executed for this reconciliation:** 113 passing tests, 0 failures.

---

## Wave 0 Requirements

All Wave 0 items complete — each verified in the tree at commit `97fd855`.

- [x] Executed-count self-assertion folded into the golden test — delivered as `TestGoldenScenarioCountIsExact` (26 goldens + 4 CASES cases = 30, **exact equality**, with both legs fatal at zero before the sum). Paired with `TestReFrozenGoldensValid`, which pins the same enumeration to the filesystem via `os.ReadFile` — neither test alone would be sufficient
- [x] The unconditional golden job in `corpora.yml` + the `inScopeJobs` entry — job present as *"golden (fetch, assert, unconditional golden suite)"*; `inScopeJobs` carries it (2 references); `TestWorkflowRunBodiesInvokeTask` green
- [x] Widened path filter (**S3 gap closed**) — `testdata/golden/**`, `corpus/**`, `corpora/**`, `internal/query/**`, `internal/cli/**`, `internal/mcp/**` all present
- [x] `-count=1` on the `test:golden` target (**S5 gap closed**) — 1 occurrence verified
- [x] The FIXT-07 mutation log — `03-MUTATION-LOG.md`, 5 family sections (a–e), 19 pasted FAIL/Fatal blocks, 11 revert commands, plus a *"Pre-mutation cleanliness gate"* and a *"corpus precondition (review HIGH)"* header
- [x] `test:golden` removed from `ci.yml`'s test job (**D-04**) — 0 hits in `ci.yml`, 2 in `corpora.yml` (positive control)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Each FIXT-07 family's RED demonstration is genuine (mutation applied, observed failure, byte-clean revert) | FIXT-07 | Requires applying a real mutation to the suite and watching the family fail, then reverting — a rehearsal, not a test | Per family: apply the mutation, run the family, observe RED, `git checkout --` / revert byte-clean, record mutation + failure + revert |

---

## Validation Sign-Off

- [x] All tasks have automated verify or a documented manual-only rationale
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (4 automated rows precede the single manual row)
- [x] Wave 0 covers all MISSING references — all 6 items complete
- [x] No watch-mode flags
- [x] Every test command forces `-count=1`
- [x] Every declared command **executed**, not read; counts recorded
- [x] Stale plan-time gap notes (S5) corrected rather than left frozen
- [ ] `nyquist_compliant: true` — **not set**, see below

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** FIXT-03 is fully automated
and green. FIXT-07 is not automatable **by construction**: its requirement is that each assertion
family has been *watched to fail* under a real mutation that was then reverted byte-clean. A
standing automated version would have to either leave a mutation in the tree or assert nothing —
the second is precisely the vacuous-guard shape (rule `84d1gfpywd`) this phase exists to disprove.
So the demonstration is one-time and recorded, not re-fired, and the phase is PARTIAL for that
reason alone. This matches the v0.5.0 Phase 3 precedent, where four proofs were deliberately made
one-time and `nyquist_compliant: false` meant "proven once on purpose", not "untested".

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 5 |
| Automated & green | 4 |
| Manual-only (by design, performed) | 1 |
| Gaps found | 0 (all 6 Wave 0 gaps had already been closed by execution) |
| Stale facts corrected | 1 (S5 `-count=1` note) |
| Escalated | 0 |
| Tests executed | 113 PASS / 0 FAIL |

**Approval:** validated 2026-08-16
