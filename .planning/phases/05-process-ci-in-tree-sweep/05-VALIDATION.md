---
phase: 5
slug: process-ci-in-tree-sweep
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-16
validated: 2026-08-16
validation_mode: retroactive-reconstructed
reconstruction_note: >-
  No VALIDATION.md was seeded at plan time for this phase — the only phase in v0.11.0 missing one.
  This file was reconstructed from the 8 PLAN/SUMMARY pairs and the requirement→task map, per
  validate-phase State B, rather than reconciled from a plan-time draft.
automated_rows: 6
manual_only_rows: 2
tests_executed: 1704
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

**Reconstructed, not reconciled.** Phase 5 is the only phase in this milestone with **no**
plan-time `VALIDATION.md`. Under `validate-phase` State B this file is built from artifacts: the
8 PLAN files' `<automated>` verify blocks, the 8 SUMMARY files' `requirements_completed`
frontmatter, and the requirement→phase map in `REQUIREMENTS.md`. Every command below was
**executed** at reconstruction time against commit `615795b`, not transcribed from a plan.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `Taskfile.yml` is the single suite definition |
| **Quick run command** | `go test -count=1 ./internal/query/... ./internal/cli/...` |
| **Full suite command** | `task test` (with `internal/daemon` isolated at `-count=1`, per its documented load-induced flake) |
| **Estimated runtime** | quick ~20s · full suite several minutes |

> **`-count=1` is mandatory** — a cached `ok` can mask a real regression. Every command in this file
> carries it.

---

## Sampling Rate

- **After every task commit:** the package set that task touched, at `-count=1`
- **After every plan wave:** `go test -count=1 ./...` plus `go test -count=1 ./testdata/golden/...`
  (`./...` excludes `testdata`, which is threat `T-05-04`)
- **Before `/gsd-verify-work`:** full suite green **and** the CODE-01 census clean under a
  **multiline** instrument
- **Max feedback latency:** ~20s on the per-task loop

---

## Per-Task Verification Map

| Task ID | Plan | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Result | Status |
|---------|------|-------------|------------|-----------------|-----------|-------------------|--------|--------|
| 05-01-T1 | 05-01 | CODE-03 | T-05-01, T-05-02, T-05-03 | `migrate` removed with its sole-use dependency and every API remnant; build still links | unit | `go test -count=1 ./internal/graphstore/... ./internal/indexer/... ./internal/query/...` | **812 PASS** | ✅ green |
| 05-02-T1 | 05-02 | PROC-01, PROC-02 | T-05-06, T-05-07, T-05-08 | Templates reworded without dropping the issue-first rule or the output-shape guard | static | framing scan over `.github/ISSUE_TEMPLATE/` (5), `PULL_REQUEST_TEMPLATE/` (3), `pull_request_template.md` | **0 hits**; 5 + 3 templates present | ✅ green |
| 05-03-T1 | 05-03 | PROC-03 | **T-05-09 (critical)**, T-05-10, T-05-11 | The 7 required-check names stay byte-identical; only free display names change | unit (shape) | `go test -count=1 ./internal/upgrade/...` | **164 PASS**, incl. `TestRequiredCheckNamesPreserved` **and** its `_ZeroJobsIsError` non-vacuity guard | ✅ green |
| 05-04/06-T1 | 05-04, 05-06 | CODE-01 | T-05-13, T-05-14, T-05-16, T-05-21, T-05-22 | Query sweep + synthetic rename land with every call site; corpus reword is bound to a re-freeze | unit | `go test -count=1 ./internal/query/... ./internal/indexer/capability/...` + golden suite | **398 PASS**; goldens **26/26**, scenarios **30/30** | ✅ green |
| 05-05-T1 | 05-05 | CODE-01 | T-05-17, T-05-18, T-05-19, T-05-20 | Schema docs paired; archtest `packages.Load` untouched; agents marker bytes identical | unit | `go test -count=1 ./internal/mcp/... ./internal/agents/... ./internal/schema/...` | **435 PASS** | ✅ green |
| 05-07/08-T1 | 05-07, 05-08 | CODE-01 | T-05-07-02, T-05-08-01…05 | Git-hook begin markers unchanged; goldens not re-frozen; census earns its zero | unit + census | `go test -count=1 ./internal/cli/... ./internal/githooks/... ./internal/gitmeta/...` | **293 PASS** | ✅ green |
| 05-04…08-M1 | 05-04…05-08 | CODE-01 | T-05-12, T-05-08-03 | Each retained term resolved term-by-term with a recorded reason, never by regex (D-02) | **manual** | Per-instance adjudication; see Manual-Only | 25 hits classified, **0** comparison-sense | ✅ performed |
| 05-07/08-M2 | 05-07, 05-08 | CODE-01 | T-05-07-04, T-05-08-04 | Every out-of-scope find logged to `WINDOWS.md` rather than swept silently | **manual** | Ledger review under ship-gate pressure | 14 fixed / 2 waived / **3 open** — pressure not yielded to | ✅ performed |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Total executed for this reconstruction: 1,704 passing tests, 0 failures.**

---

## CODE-01 Acceptance Census

The phase's defining gate. Run at reconstruction time with the **multiline** instrument
(`rg -U`), because a line-based search **cannot** match a phrase wrapped across a comment line
break — the exact blind spot that produced two false-green sweeps in this repo (05-04, 05-05) and
forced the 05-07/05-08 gap-closure plans.

| Pattern | Raw hits in `internal/ tools/ test/` | Comparison-sense |
|---------|--------------------------------------|------------------|
| `parity` | 14 | **0** |
| `upstream` | 11 | **0** |
| `drop-in` | 0 | 0 |
| `head-?to-?head` | 0 | 0 |
| wrapped `no TS\n// precedent` class | 0 | 0 |

**All 25 raw hits classified as legitimate:**

| Sense | N | Representative sites |
|---|---|---|
| `parity` = two things **in this repo** agree | 13 | `TestGoreleaserPinParity` (MAINT-03: `go.tool.mod` pin ↔ workflow pin), `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern` (policy literal ↔ compiled regex) plus its `_ZeroLiteralsIsError` vacuity guard, the fragment/registration parity guard |
| `parity` = a **record of the retirement** | 1 | `internal/query/status.go:44` — *"TS parity is no longer owed on `--json` shape… formally retired 2026-08-13"* |
| `upstream` = a genuine **dependency** | 3 | `upstream go-sdk#976`, *"an upstream go-sdk defect"*, *"not an upstream bug"* (goreleaser) |
| `upstream` = **dataflow / call-stack** position | 8 | *"decided upstream of these renderers"*, *"upstream iteration-order contract"*, *"applied upstream in internal/cli/status.go"*, *"recovered upstream"* |

**Positive control:** the same multiline instrument returns **3,596** `parity` hits inside
`.planning/` and **1** `upstream` hit in `NOTICE`, confirming it matches where matches exist. An
unchecked zero cannot distinguish absence from a misaimed search.

**Why this census is the argument for D-02.** A term blocklist would have flagged all 25 of these
and demanded edits to a pin-agreement test name, a dataflow comment, a dependency-defect reference,
and the sentence that *records the retirement itself*. That is the vocabulary-drift-guard proposal
(`VOCAB-01`) the milestone explicitly declined — it *"either goes vacuous or fights legitimate uses"*.
The one-time sweep plus term-by-term review is the chosen posture, and this census is what it looks
like when it works.

---

## Wave 0 Requirements

None required — Phase 5 built no new test scaffolding. It swept prose and identifiers across
existing, already-tested packages and deleted one command with its dependency. The existing suite
is the instrument, and it survived: 1,704 tests green across the six package sets the eight plans
name.

*`wave_0_complete: true` is therefore vacuously satisfied. Recorded explicitly so a reader does not
mistake "no Wave 0 items" for "Wave 0 not done".*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Each retained comparison-vocabulary term is resolved term-by-term with a recorded reason (D-02) | CODE-01 | The 25 census hits above are the proof: an automated rule strict enough to flag comparison framing also flags `TestGoreleaserPinParity`, `upstream go-sdk#976`, and the sentence recording the retirement. `VOCAB-01` was declined for exactly this reason | Run the multiline census; classify every hit by sense; record the reason per instance |
| Every out-of-scope find is logged to `WINDOWS.md` rather than swept silently | CODE-01 | `/gsd-ship` blocks on `open_count > 0`, creating direct pressure to close genuine open entries to reach zero. Whether an entry was *closed* or *made to look closed* is a judgment | Read the ledger; confirm open entries remain open on their merits |

> The second row is the more important one. Threat `T-05-08-04` names the pressure explicitly, and
> the recorded orchestrator error in this phase — hardcoding `open_count == 3` into an executor
> brief, which was right when written and wrong once new findings appeared — is the same failure in
> the opposite direction. The durable lesson recorded from it: **pin the property, not the literal**
> ("no CODE-01 entry remains open" survives new findings; `open_count == 3` does not).

---

## Validation Sign-Off

- [x] All tasks have automated verify or a documented manual-only rationale
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (6 automated rows precede the 2 manual rows)
- [x] Wave 0 covers all MISSING references — none required
- [x] No watch-mode flags
- [x] Every test command forces `-count=1`
- [x] Every declared command **executed**, not read; counts recorded
- [x] CODE-01 census run with a **multiline** instrument and positive-controlled
- [ ] `nyquist_compliant: true` — **not set**, see below

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** All five requirements
(PROC-01, PROC-02, PROC-03, CODE-01, CODE-03) carry executed automated verification and all are
green. The two manual-only rows are irreducibly judgment-shaped, and this phase is the milestone's
clearest demonstration of why: the census's own 25 hits are the counter-example to any automated
version of the classification rule. Automating it would contradict D-02, the ruling that governs it.

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 6 automated + 2 manual-only |
| Automated & green | 6 |
| Manual-only (by design, performed) | 2 |
| Gaps found | 0 |
| Census hits requiring classification | 25 — all KEEP, 0 comparison-sense |
| Escalated | 0 |
| Tests executed | 1,704 PASS / 0 FAIL |

**Approval:** validated 2026-08-16
