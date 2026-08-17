---
phase: 04
slug: attribution-documentation-sweep
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 14
threats_closed: 14
threats_mitigated: 13
threats_accepted: 1
threats_high: 6
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. All **3** PLAN files carry a threat model, so
`register_authored_at_plan_time: true`. With `asvs_level: 1` and `threats_open: 0`, the L1
short-circuit applies; mitigations were verified against the tree at commit `3f88ae3`.

**Register-form note.** `04-03-PLAN.md` expresses its threat model as `## Threat Model` /
`## Trust Boundaries` / `## STRIDE Threat Register` **markdown headings** rather than the
`<threat_model>` XML block the other two plans use. A survey keyed only on `<threat_model>` reports
this phase as 2/3 and Phase 5 as 2/8, both wrong. Both forms are in use across this milestone and
either satisfies "authored at plan time"; a detector for this repo must match the **union**.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| `NOTICE` (in-git) → `LICENSE` (in-git) | The NOTICE trim must never spill an attribution line into `LICENSE` — the boundary this project's own history says must not be crossed (observed `NOASSERTION` regression) | Attribution text |
| README sweep → product surface | The sweep removes comparison words but must not remove indexed-language rows or command rows that are product truth | Documented capability |
| capability matrix doc → `matrix.go` / `matrix_test.go` | The doc is the human-readable half of a coverage contract; an edit crossing into a coverage value or a machine-readable Gaps bullet breaks the test that parses them | Coverage values |
| deleted drift guard → CI shape guard | Deleting `flag_parity_test.go` must not break `TestWorkflowRunBodiesInvokeTask` | Workflow/Taskfile shape |
| `docs/FLAG-PARITY.md` → contributor-facing templates | Every `.github` template and the auto-close comment string that linked to the deleted doc is a dangling-reference trap if not swept | Doc links |
| CONTRIBUTING doc → contributor expectations | The Issue-first paragraph is the contract contributors read; the rewrite must drop framing without weakening the issue-before-PR rule | Process contract |

---

## Threat Register

| Threat ID | Plan | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | 04-01 | Tampering | NOTICE trim (ATTR-01) | **high** | mitigate | Explicit keep-list with positive assertion per block. **Verified in full:** the LICENSE-warning block survives at `NOTICE:3-10`, the upstream copyright transcription at `:16-23`, and Third-party dependencies at `:29-46` | closed |
| T-04-02 | 04-01 | DoS (capability loss) | README sweep (ATTR-02) | **high** | mitigate | The sweep forbids touching the Commands/Language tables. **Verified:** 4 TS/TSX/JS language references survive in README. *(The `migrate` row this threat also protected was subsequently and deliberately removed — see Cross-Phase Supersession below.)* | closed |
| T-04-ATTR03 | 04-01 | Tampering | LICENSE untouched (ATTR-03) | **high** | mitigate | `LICENSE` is never opened for edit and the live detection check is re-run after the NOTICE edit. **Verified live at audit time:** `gh api repos/seanb4t/codegraph-go/license --jq .license.spdx_id` → **`MIT`**; `LICENSE` first line is `MIT License`; last touched by `6486ac4`, which predates this phase | closed |
| T-04-01-P1 | 04-01 | Tampering | NOTICE transcription fidelity | medium | mitigate | The transcription is byte-identical to the upstream `LICENSE`, retrieved 2026-08-01 via `gh api`. The fidelity guard survives at `NOTICE:25-27`: *"Do not 'correct' the capitalization of that name… The plausible-looking correction is wrong."* | closed |
| T-04-03 | 04-02 | Tampering | dangling reference to deleted FLAG-PARITY / `flag_parity_test.go` | **high** | mitigate | Full blast radius swept — 9 edit-lines across 7 owned files, plus NOTICE (04-01) and CONTRIBUTING (04-03). **Verified:** **0** `FLAG-PARITY` references outside `.planning/` and `CHANGELOG.md`; `internal/cli/flag_parity_test.go` absent | closed |
| T-04-04 | 04-01 | Tampering | doc↔product-truth boundary | medium | mitigate | Census-driven edits only; **no regex over `TypeScript`/`TS`** — the discipline that keeps `tsextract`-the-capability distinct from the origin project | closed |
| T-04-05 | 04-02 | Tampering | deleted guard breaking CI / capability matrix | **high** | mitigate | Zero Taskfile targets and zero CI steps referenced either artifact. **Verified:** `internal/cli` builds and tests green; `TestWorkflowRunBodiesInvokeTask` green | closed |
| T-04-06 | 04-02 | DoS (coverage loss) | removing the live drift guard silently | medium | mitigate | The DOCS-05 deferral is recorded as a **knowing** reduction (D-03) rather than an unremarked loss — flag-documentation coverage is intentionally reduced this milestone and the replacement reference is scheduled separately | closed |
| T-04-07 | 04-02 | Tampering | over-broad sweep in the same lines | medium | mitigate | Each edit is a one-line dangling-link/name removal; acceptance asserts `git diff` shows no framing vocabulary beyond the named removals | closed |
| T-04-08 | 04-03 | DoS (capability loss) | capability matrix reword (DOCS-03) | **high** | mitigate | Row/value/gap-bullet changes forbidden; TS/TSX/JS rows are product truth (D-05). **Verified:** `` `typescript` ``, `` `tsx` ``, `` `javascript` `` rows intact at `docs/LANGUAGE-CAPABILITY-MATRIX.md:27-29` with coverage values, plus per-language detail sections at `:81`, `:87`, `:93`; `go test ./internal/indexer/capability/...` → **7 PASS** | closed |
| T-04-09 | 04-03 | Tampering | CONTRIBUTING Issue-first rewrite (DOCS-01) | medium | mitigate | The rewrite drops the dangling FLAG-PARITY link inside the same paragraph while keeping the `.planning/` reference and the issue-before-PR rule intact | closed |
| T-04-10 | 04-03 | Tampering | verify-only surfaces edited despite a CLEAN census | medium | mitigate | `git diff --quiet` over all 8 verify-only surfaces — any edit fails the task. Confirmed-not-edited is a distinct verdict from swept | closed |
| T-04-11 | 04-03 | Tampering | stale rewrites inventing a nonexistent harness | medium | mitigate | The three stale rows must describe the **current** fail-loud behavioral suite, not an imagined one — a rewrite that invents a harness is worse than the framing it replaced | closed |
| T-04-SC | all | Tampering | npm/pip/cargo installs | low | accept | No new external packages; in-git markdown edits only — see Accepted Risks | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*

**Totals:** 14 distinct threats — 13 mitigated, 1 accepted, **0 open**. High-severity: 6, all mitigated and verified.

---

## Cross-Phase Supersession (recorded, not a violation)

`T-04-02` protected two things in README: the **indexed-language rows** and the **`migrate` command
row**. The language rows survive and are verified above. The `migrate` row does **not** — and its
absence is correct.

Phase 4 protected it because, at the time, `codegraph migrate` was product surface and removing its
documentation would have been capability loss. Maintainer ruling **D-04 (2026-08-15)** then dropped
the command itself — `internal/migrate`, the `modernc.org/sqlite` dependency, and all its
documentation — as `CODE-03`'s amended scope in Phase 5. The doc row went away *with* the capability,
which is exactly the condition `T-04-02` was guarding against the *absence* of.

This is recorded here so a later reader comparing Phase 4's threat model against the tree does not
mistake a deliberate supersession for a regression. A threat's mitigation can be legitimately undone
by a later ruling; what must not happen is that undoing going unrecorded.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-04-01 | T-04-SC | No new external packages are introduced by any of the three plans — every edit is to in-git markdown — so the npm/pip/cargo Package Legitimacy Gate does not apply. Recorded rather than omitted, in all three plans independently. | plan author (04-01, 04-02, 04-03) | 2026-08-15 |

---

## Threat Flags from Execution

No Phase 4 summary carries a `## Threat Flags` section. `04-VERIFICATION.md` records `gaps: []`,
`deferred: []`, no anti-patterns, and 5/5 must-haves verified. The absence of a flags section is a
template difference from Phases 2 and 3, not a suppressed finding — the verification report covers
the same ground with an explicit empty-gap result.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 14 | 14 | 0 | `/gsd-secure-phase 4` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep depth against the tree at commit `3f88ae3`, one **live** API call,
and two executed test runs. Three of this audit's own searches returned misleading results and were
corrected by reading the file rather than trusting the count:

1. `04-03-PLAN.md` reported "no threat model" because the detector matched only `<threat_model>`; it
   uses markdown headings. This also revealed Phase 5 has **8/8** modelled plans, not 2/8.
2. `NOTICE` reported no MIT transcription because the pattern searched for the permission-grant body.
   The transcription is deliberately **partial** — header plus copyright line only (`:19-23`).
3. The capability matrix reported no TS rows because the pattern was title-case and un-backticked;
   the rows are `` | `typescript` | `` and present.

None of the three was a real gap. All three were the same failure: an unchecked zero reported the
instrument, not the population.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
