---
phase: 4
slug: attribution-documentation-sweep
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-15
validated: 2026-08-16
validation_mode: retroactive
automated_rows: 7
manual_only_rows: 2
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) |
| **Config file** | none — `Taskfile.yml` is the single CI definition |
| **Quick run command** | `go test -count=1 ./internal/cli/... ./internal/indexer/capability/...` |
| **Full suite command** | `go test -count=1 ./...` |
| **Estimated runtime** | quick ~15s · full ~1min |

> **`-count=1` is mandatory** — recorded repo gotcha.

---

## Sampling Rate

- **After every task commit:** `go test -count=1 ./...` (full — the sweep touches shared doc/test surfaces)
- **After every plan wave:** `go test -count=1 ./...` + the reference sweeps (`rg` for the deleted framing)
- **Before `/gsd-verify-work`:** full suite green, `rg "FLAG-PARITY"` (harness scope) empty, `gh api repos/seanb4t/codegraph-go/license` returns `MIT` live
- **Max feedback latency:** ~60s (full suite)

---

## Per-Task Verification Map

> Reconciled retroactively by `/gsd-validate-phase 4` on 2026-08-16. Every check **executed**,
> including the live GitHub API call. Raw hit counts are reported, then classified — a "comparison
> sense empty" row whose raw count is non-zero is recorded as such rather than rounded to zero.

| Task ID | Plan | Wave | Requirement | Threat Ref | Test Type | Automated Command | Result | Status |
|---------|------|------|-------------|------------|-----------|-------------------|--------|--------|
| 04-01-T1 | 04-01 | 1 | ATTR-01 | T-04-01, T-04-01-P1 | static | `NOTICE` contains the transcription + one origin sentence; `rg "ported-heuristics\|flag-parity\|drop-in" NOTICE` empty | **0** framing hits; origin sentence present; keep-list blocks intact at `:3-10`, `:16-23`, `:29-46` | ✅ green |
| 04-01-T2 | 04-01 | 1 | ATTR-02 | T-04-02 | static | `rg "Relationship to the original" README.md` empty; `## License` carries one past-tense clause linking to NOTICE | **0**; `README.md:207` — *"This project began as a Go rewrite of CodeGraph; see NOTICE for the original copyright."* | ✅ green |
| 04-01-T3 | 04-01 | 1 | ATTR-03 | T-04-ATTR03 | **live** | `gh api repos/seanb4t/codegraph-go/license --jq .license.spdx_id` | **`MIT`** (re-run live at audit time, not assumed) | ✅ green |
| 04-03-T1 | 04-03 | 3 | DOCS-01 | T-04-09 | static | `rg "parity\|upstream\|drop-in\|the original"` over the 5 named files, comparison sense | **1 raw hit**, all others 0 — `README.md:207`, which *is* the ATTR-02-sanctioned clause. Comparison sense: **0** | ✅ green |
| 04-02-T1 | 04-02 | 2 | DOCS-02 | T-04-03, T-04-05 | static+test | `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` absent; `rg "FLAG-PARITY"` empty; suite green | both **absent**; **0** references outside `.planning/`+CHANGELOG; `go build ./...` exit 0 | ✅ green |
| 04-03-T2 | 04-03 | 3 | DOCS-03 | T-04-08 | static | matrix carries no reference to another implementation's coverage | **0** `reference implementation\|golden-parity\|weft`; `` `typescript` ``/`` `tsx` ``/`` `javascript` `` rows intact at `:27-29`; capability pkg **7 PASS** | ✅ green |
| 04-03-T3 | 04-03 | 3 | DOCS-04 | T-04-10, T-04-11 | static | `rg "parity\|upstream\|drop-in\|the original" docs/`, comparison sense | **4 raw hits, all classified KEEP** — see below. Comparison sense: **0** | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

### DOCS-04 — the four raw hits, classified

`04-VERIFICATION.md` recorded this row as *"only recorded keeps (branch name, pipeline upstream)"*.
Re-derived independently at audit time and the classification holds:

| Location | Text | Verdict |
|---|---|---|
| `docs/RELEASE-PROCEDURES.md:60` | `gsd/v1.0-drop-in-parity-human-ux` | **KEEP** — a git **branch name** from the v1.0 milestone. Rewriting it would falsify a historical procedure record |
| `docs/RELEASE-PROCEDURES.md:581` | *"nothing **upstream** of it was skipped"* | **KEEP** — pipeline-ordering sense, not origin-project sense |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md:181` | *"the **original**ly-approved source"* | **KEEP** — refers to a grammar recommendation. The pattern matched `the original` as a **prefix of `originally`**, an unbounded-match artifact of the same class as the recorded `ports`/`reports` incident |

**Instrument note.** These raw hits are why the row reads *"comparison sense empty"* rather than
*"empty"*. A validation row that demanded a literal zero here would have to be satisfied by editing
a branch name, a pipeline description, and the word "originally" — i.e. by damaging the docs to turn
a gate green. When a gate flags ordinary English, the finding is about the gate.

**Positive control:** the same pattern returns **4,447** hits inside `.planning/`, confirming the
instrument matches where matches should exist.

---

## Wave 0 Requirements

- [x] None required — no new test scaffolding; the sweep is static plus a live licence check. **Confirmed at audit time:** the existing suite survived the FLAG-PARITY guard deletion (`go build ./...` exit 0, `internal/cli` green, `internal/indexer/capability` 7 PASS) and `TestWorkflowRunBodiesInvokeTask` is green.

*`wave_0_complete: true` is therefore vacuously satisfied — there was nothing to build. Recorded explicitly so a reader does not mistake "no Wave 0 items" for "Wave 0 not done".*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Each borderline reference is resolved past-tense-origin (keep) vs comparison (remove), recorded per instance | DOCS-01/03/04 | Term-by-term judgment with recorded reasons, never regex | Read the per-instance decisions in the plan/summary; confirm each borderline reference is classified with a reason |
| The retained attribution reads as one sentence of origin, past tense, no comparison argument | ATTR-01/02 | Prose judgment | Read NOTICE and README `## License` |

---

## Validation Sign-Off

- [x] All tasks have automated verify or a documented manual-only rationale
- [x] `go build ./...` / package suites green after the deletion
- [x] `gh api` licence check verified **live**, not assumed → `MIT`
- [x] No watch-mode flags
- [x] Every static zero positive-controlled; non-zero raw counts classified rather than rounded
- [ ] `nyquist_compliant: true` — **not set**, see below

**`nyquist_compliant: false` — PARTIAL, by decision, not by omission.** All seven ATTR/DOCS
requirements carry an executed check and all are green. The two manual-only rows are irreducibly
judgment-shaped:

1. **Term-by-term classification** (DOCS-01/03/04) — deciding whether each borderline reference is
   past-tense-origin (keep) or comparison (remove). The four `docs/` hits above are the concrete
   case: an automated rule strict enough to flag them would demand edits to a git branch name, a
   pipeline description, and the word "originally". The plans state the rule explicitly — *"resolved
   term-by-term with a recorded reason, never by regex"* — so automating it would contradict the
   requirement it claims to verify.
2. **Prose judgment** (ATTR-01/02) — that the retained attribution reads as one sentence of origin,
   past tense, carrying no comparison argument. Nothing mechanical distinguishes a sentence that
   acknowledges an origin from one that argues about it.

Both were performed and recorded per-instance in `04-VERIFICATION.md` (5/5 must-haves, `gaps: []`,
`deferred: []`). This phase is PARTIAL for these two rows alone.

## Validation Audit 2026-08-16

| Metric | Count |
|--------|-------|
| Rows in map | 7 automated + 2 manual-only |
| Automated & green | 7 |
| Manual-only (by design, performed) | 2 |
| Gaps found | 0 |
| Raw hits requiring classification | 5 (1 README + 4 docs) — all KEEP |
| Escalated | 0 |
| Live API checks re-run | 1 (`gh api … .license.spdx_id` → `MIT`) |

**Approval:** validated 2026-08-16
