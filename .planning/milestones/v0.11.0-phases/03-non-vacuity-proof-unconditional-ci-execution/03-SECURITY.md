---
phase: 03
slug: non-vacuity-proof-unconditional-ci-execution
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 16
threats_closed: 16
threats_mitigated: 14
threats_accepted: 2
threats_critical: 1
threats_high: 4
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 03 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. Both PLAN files carried a `<threat_model>` block, so
`register_authored_at_plan_time: true`. With `asvs_level: 1` and `threats_open: 0`, the L1
short-circuit applies; mitigations were verified against the tree at commit `97fd855`.

This phase carries the milestone's **only `critical` threat**. That is appropriate to its subject: a
mutation rehearsal deliberately breaks the golden suite to prove it can go red, so a rehearsal whose
revert is missed would land mutation bytes inside the very commit that certifies the guard. The
phase's threat model treats its own method as the primary hazard, which is the right posture.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| corpus cache volume → `golden` job | The nscloud volume is a **mutable** cache any PR can populate; the job reads it to serve the golden suite | Cached corpus trees |
| `corpora.yml` path filter → workflow firing | A PR changing golden inputs crosses this boundary; if the filter omits them the workflow never fires and the suite is un-proven | PR file paths |
| `go test` cache → test result | The test cache can return a stale PASS without executing against the current corpus tree | Cached test verdicts |
| mutation rehearsal → git tree | A rehearsal applies a real mutation; a missed revert lands a mutation in a golden-guard commit | Source/fixture bytes |
| rehearsal → local corpus cache | Family (d) moves a corpus tree outside the repo; an inexact restore leaves later runs resolving a missing tree | Corpus tree path |

---

## Threat Register

### Plan 03-01 — CI execution

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-P1-01 | Spoofing | `corpora.yml` `golden` job | medium | mitigate | Job declares the nscloud runner class; workflow-level `permissions: contents: read` (line 123-124) applies to it. **Verified:** zero `id-token: write` anywhere in `corpora.yml` — the only two `id-token` strings are in a comment *documenting* the withheld permission | closed |
| T-03-P1-02 | Tampering | nscloud cache volume → `golden` job | **high** | mitigate | The job runs `task corpora:assert` (four-part: `.git` present, `HEAD == pin`, `git status --porcelain --ignored` empty, tree resolves) **after** cache restore, so a poisoned volume fails the job. **Verified:** *"Assert corpora present at pinned SHAs"* step present, running `task corpora:assert` | closed |
| T-03-P1-03 | Denial of Service | fetch/cache failure silently skipping the job | **high** | mitigate | `task corpora:fetch` and `task corpora:assert` are **unconditional** — not gated on `steps.cache-corpora.hit`; a miss falls through to a real fetch. **Verified:** no `if:` on either step; the workflow comment states *"a cache MISS falls through to a REAL fetch"* | closed |
| T-03-P1-04 | Tampering | `ci.yml` running corpus-dependent tests | **high** | mitigate | D-04 removal — the `test:golden` step left `ci.yml`, so the assertion now runs only where pinned corpora are fetched. **Verified:** `test:golden` = **0** hits in `ci.yml`, **2** in `corpora.yml` (positive control) | closed |
| T-03-P1-05 | (process gap) | golden-only PR bypassing the corpus-aware workflow | medium | mitigate | Widened transitive-closure path filter covers `testdata/golden/**`, `corpus/**`, `corpora/**` and the golden-pipeline packages | closed |
| T-03-P1-06 | (process gap) | `go test` cache returning a stale PASS | medium | mitigate | `test:golden` carries `-count=1`, so the golden job always executes against the current corpus tree. **Verified:** 1 occurrence in the `test:golden` target | closed |
| T-03-P1-07 | Concurrency | `corpora` + `golden` jobs sharing the volume | medium | mitigate | Documented no-`needs:` decision; the fetch driver's claim-lock / staged-promote (`Taskfile.yml:3472`) already serializes concurrent fetchers of the same destination | closed |
| T-03-P1-08 | Spoofing | executed count detached from execution | medium | mitigate | Counters live **inside** the loops and exact equality binds them to the totals — `expectedTotal++` (line 249) and `verified++` (line 284) are both loop-body increments, checked by `if verified != expectedTotal` (line 293). A loop that early-returns cannot report a full count | closed |
| T-03-P1-SC | Tampering | npm/pip/cargo installs | low | accept | No new external packages; all actions are already-pinned reuse — see Accepted Risks | closed |

### Plan 03-02 — mutation rehearsals

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-P2-01 | Tampering | mutation rehearsal workflow | **critical** | mitigate | Every rehearsal is single-shot **apply → run → revert** before the next apply; each task enforces order in the action body and ends by re-running the family green. **Verified live:** working tree clean (0 dirty paths); `TestReFrozenGoldensValid` → 26/26 and `TestGoldenScenarioCountIsExact` → 30/30 **executed** at audit time; 5 families recorded with 11 revert commands | closed |
| T-03-P2-02 | (integrity) | local corpus cache, family (d) | medium | mitigate | The tree rename backs off to its exact sha-bearing name in the same parent, verified by a green re-run of the resolution test | closed |
| T-03-P2-03 | Repudiation | `03-MUTATION-LOG.md` evidence | medium | mitigate | Each entry carries pasted failing output plus the exact revert command; an entry without pasted output does not satisfy FIXT-07. **Verified:** 19 pasted FAIL/Fatal blocks across 5 family sections, 11 revert commands | closed |
| T-03-P2-04 | Denial of Service | rehearsal run in CI | low | accept | Rehearsals are local-only apply-revert acts, never run in CI — see Accepted Risks | closed |
| T-03-P2-05 | (integrity) | missing corpus masks the intended RED | **high** | mitigate | `task corpora:fetch` + `task corpora:assert` run first and `CODEGRAPH_CORPUS_DIR` is recorded (review HIGH); the rehearsal stops if either fails, so a RED caused by a missing corpus cannot be mistaken for a RED caused by the mutation. **Verified:** 8 `corpora:fetch`/`corpora:assert` references and 2 `CODEGRAPH_CORPUS_DIR` records in the log, under a dedicated *"Header — corpus precondition (review HIGH)"* section | closed |
| T-03-P2-06 | (integrity) | blind `git checkout --` destroys a pre-existing tracked edit | medium | mitigate | Mandatory `git diff --quiet -- <file>` gate before every tracked-file mutation and revert; non-zero exit **stops** the rehearsal. **Verified:** 5 occurrences, under a *"Pre-mutation cleanliness gate (review finding)"* section | closed |
| T-03-P2-07 | (integrity) | interrupted family-(d) rename leaves the corpus missing | medium | mitigate | The whole rename → run → restore runs in **one** shell invocation with an `EXIT` trap restoring the exact `hugo@<sha>` path | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (high) count toward `threats_open`*

**Totals:** 16 threats — 14 mitigated, 2 accepted, **0 open**. At/above the block threshold: 1 critical + 4 high, all mitigated and verified.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-03-01 | T-03-P1-SC | No new external packages are introduced. Every action is already-pinned reuse (`actions/checkout`, `actions/setup-go`, `namespacelabs/nscloud-cache-action`), so the npm/pip/cargo Package Legitimacy Gate does not apply. | plan author (03-01) | 2026-08-15 |
| R-03-02 | T-03-P2-04 | Mutation rehearsals are **local-only** apply-revert acts and never run in CI. The permanent artifacts are the green suite plus `03-MUTATION-LOG.md`; no rehearsal step exists in any workflow, so a rehearsal cannot consume CI capacity or leave a mutated tree on a runner. | plan author (03-02) | 2026-08-15 |

---

## Threat Flags from Execution

**Both** Phase 3 summaries carry a `## Threat Flags` section — the only phase in this milestone where
every plan does — and both record **"None."**

- `03-01-SUMMARY.md`: *"The `golden` job's surface (new CI job reading the nscloud cache volume and
  the widened path filter) is exactly the surface the plan's `<threat_model>` specifies."*
- `03-02-SUMMARY.md`: *"The rehearsal touched no new network endpoint, auth path, file-access
  pattern, or schema change beyond the plan's modeled register (T-03-P2-01..07)."*

Each is a **scoped** negative — "no surface beyond the modeled register" — rather than a bare "none",
which is what makes it checkable. A bare negative flag would be the vacuous-assertion shape rule
`84d1gfpywd` warns about.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 16 | 16 | 0 | `/gsd-secure-phase 3` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep depth against the tree at commit `97fd855`, plus two **executed**
tests for the critical threat. One count-based check over-flagged and was corrected by reading:
`id-token` returns 2 hits in `corpora.yml`, but both are inside a comment *documenting* that the job
withholds the permission — the opposite of a finding. The actual declaration is workflow-level
`permissions: contents: read` at line 123. A grep count is a pointer to a place to read, never a
verdict on its own.

The critical threat (T-03-P2-01) was **not** verified from the summary's own historical
`git diff` claim: that range now spans Phases 4–6 and legitimately shows later deletions. Live
evidence was used instead — a clean working tree plus both non-vacuity guards re-run green at audit
time.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
