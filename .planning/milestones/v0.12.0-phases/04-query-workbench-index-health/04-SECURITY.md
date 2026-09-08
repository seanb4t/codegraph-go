---
phase: 04
slug: query-workbench-index-health
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
created: 2026-08-30
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all 7 plans carry a
`<threat_model>` block. 35 distinct threat IDs, 26 `mitigate` / 11 `accept`.
Verification depth is ASVS L1 (grep-level mitigation presence), which the workflow's
short-circuit declares sufficient at `threats_open: 0`. Every zero-count check below is
paired with a positive control per rule `84d1gfpywd`.

**Blocking threshold:** `security_block_on: high`. All **12** high-severity threats are
`mitigate`; **zero** high-severity threats were accepted. All 11 accepts are `low` or
`medium`.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| npm registry → `web/package.json` → `pnpm-lock.yaml` → committed `web/build/` → signed binary | Two new runtime/dev packages enter an artifact committed to git and embedded in the release | Executable JS |
| shadcn-svelte registry (network) → committed `.svelte` source | The CLI writes component source that is committed and shipped, and never enters `pnpm-lock.yaml` — invisible to every JS supply-chain gate | Component source |
| browser → Connect RPC on loopback | The 11th rpc joins the surface; `depth`/`limit`/`file[]` are client-steerable | Integers, repo-relative paths |
| `StatusResult` → `GetHealthResponse` → browser | Index diagnostics, including host-absolute worktree paths, cross to the browser for the first time | Diagnostics, host paths |
| search term → glob → `internal/query.Files` | A user-typed string is interpolated into a pattern evaluated by `doublestar` over the indexed record set | Glob pattern |
| shared Workbench URL → four analysis RPCs | A link authored by someone else controls `mode`, `symbol`, repeated `file`, `depth`, `limit` | URL parameters |
| indexed symbol names / paths → rendered tables | Repository-derived strings render as text in four tables and a health view | Identifiers, paths |
| Taskfile / workflow gate verdict → merge & release decisions | A guard that cannot fail lets everything through while appearing to guard | CI verdicts |

---

## Threat Register

All 35 threats **CLOSED**. Every mitigation below was verified against the working tree
this session, not accepted from the plan text.

### High severity — the blocking set (12, all `mitigate`, all verified)

| Threat ID | Component | Mitigation | Verified | Status |
|-----------|-----------|------------|----------|--------|
| T-04-01 | vendored shadcn `table` source | Positive-controlled zero-count review for raw-HTML sinks; `web:components:drift` byte-compare | `{@html\|innerHTML\|outerHTML\|insertAdjacentHTML` over `components/ui/` = **0**, positive-controlled by `class` = **172** | closed |
| T-04-06 | `doublestar` swap changing existing pattern behaviour | `non_recursive_unchanged` subtest; no golden/transcript drift | subtest present; `testdata/wireoracle` = 46 tracked files, **0** phase-4 commits touched it (path proven non-empty first) | closed |
| T-04-08 | a mutating rpc entering `UIService` | Positive set-equality fixture +1 entry and `10`→`11`; negative verb guard left untouched | `mutatingVerbs` diff across phase = **0** lines; count literal is `!= 11` | closed |
| T-04-17 | a second health/status verdict | D-04: exactly one `classifyStatus`, consumed by both surfaces | 1 `classifyStatus`, 1 `StatusVerdict`; `health-view.ts`'s 3 references are a type-only import, a comment, and a type annotation — **no verdict computation** | closed |
| T-04-27 | vendored source drifting from its registry origin | `web:components:drift` — pinned-CLI regeneration into scratch + `cmp -s`, disk-derived subject set | live run: `compared 50 vendored component files across 8 components`, PASS | closed |
| T-04-28 | the drift guard becoming a merge dependency on an external registry | `schedule` + `workflow_dispatch` only; absent from `requiredCheckNames` and `release.yml` | triggers are exactly `workflow_dispatch` + `schedule`; **0** in `requiredCheckNames`; **0** in `release.yml` | closed |
| T-04-29 | the drift guard's subject set narrowing silently | Disk-derived enumeration; NO trailing slash; structural floors below current | trailing-slash pathspec = **0** occurrences, positive-controlled by slashless form = **4**; the error message documents the trap inline | closed |
| T-04-30 | committed `web/build/` diverging from source | Rebuild + re-commit; `web:drift` GREEN with recorded digests | `PASS — hashed 103 source files, manifested 31 output files` | closed |
| T-04-31 | installing a dependency on an unverified premise | Over-threshold branch HALTS rather than installing | **Discharged by escalation** — see Note 1 | closed |
| T-04-SC | `[SUS]` npm/CLI legitimacy | `blocking-human` checkpoint; exact version pin; `@1.5.1` dlx pin | maintainer approved 2026-08-29 against an enumerated 9-file set; vendored set verified to match exactly | closed |
| T-04-SC-GO | `doublestar/v4` supply chain | `go mod graph` zero transitive deps; `CGO_ENABLED=0` builds | verified during research by executed tooling; `go.mod` pins `v4.10.0` | closed |
| T-04-SC-TABS | `tabs` vendoring adding an npm entry | Pinned CLI; positive-controlled review; `git diff web/package.json` escalation trigger | `git diff` on `package.json` across the vendoring = **empty**; escalation correctly did not fire | closed |

### Medium and low severity (23, closed)

| Threat ID | Severity | Disposition | Verification |
|---|---|---|---|
| T-04-02, T-04-13 | low | accept | `validateLimit`/`MaxLimit` refuse and `clampDepth` clamps server-side for every caller. Verified: **0 executable** client-side bounds in `web/src` — all 9 textual matches are comments or generated proto doc-strings explaining the server-side enforcement (positive control: `limit` = 69 matches) |
| T-04-03, T-04-05, T-04-11, T-04-14, T-04-15, T-04-20, T-04-21, T-04-22, T-04-23, T-04-24, T-04-32 | medium | mitigate | `escapeGlobLiteral` escapes `\ * ? [ ] { }` with **0** raw-term interpolations bypassing it; D-11's `getAll()`/`get()` asymmetry intact (1 each); proto additive-only — **0** removed lines in `ui.proto` across the phase; route-scoped `ROUTE_LOCAL_PARAMS` with the `/browse` counter-direction asserted |
| T-04-04, T-04-07, T-04-12, T-04-16, T-04-19, T-04-25, T-04-26 | low | accept | Loopback-bound listener with exact-match Origin/Host (SRV-02/03); Svelte escapes interpolated text by construction; maps keyed by registered language/kind sets, not by user input; `Affected` resolves against the indexed record set with no filesystem read |
| T-04-09, T-04-18 | medium | accept | Host-absolute worktree paths on the browser surface — the one scoped exception, maintainer-approved (see Accepted Risks R-04-01) |
| T-04-10 | medium | mitigate | `GetHealth` uses the ordinary `withEngine` + named mapper shape, deliberately not `GetStatus`'s degrade-and-answer exception |

---

## Accepted Risks Log

No accepted risk sits at or above the `high` blocking threshold.

| Risk ID | Threat Ref | Severity | Rationale | Accepted By | Date |
|---------|------------|----------|-----------|-------------|------|
| R-04-01 | T-04-09, T-04-18 | medium | `worktree_mismatch` carries host-absolute `worktree_root`/`index_root` to the browser. A mismatch warning that will not name the two trees is useless, and the field populates only when a mismatch genuinely exists — a clean tree leaks nothing. Extends the exception already granted on the MCP surface (T-02-14). | maintainer, at 04-03's one-way checkpoint | 2026-08-29 |
| R-04-02 | T-04-02, T-04-13 | low | `depth`/`limit` pass through unbounded to a server that already refuses/clamps for every caller. A client-side copy would be a driftable duplicate that could disagree with the authority — and could be bypassed by a direct RPC call anyway. | plan 04-01 / D-07 | 2026-08-29 |
| R-04-03 | T-04-04, T-04-26 | low | Repo-relative paths rendered in the browser; the listener is loopback-bound with exact-match Origin/Host, and Browse already renders the same class of value. | plans 04-01, 04-06 | 2026-08-29 |
| R-04-04 | T-04-07, T-04-25 | low | The glob matcher filters an already-bounded iteration over records in the store — no filesystem traversal, no path-confined open. `doublestar` is a glob matcher, not a backtracking regex engine. | plans 04-02, 04-06 | 2026-08-29 |
| R-04-05 | T-04-12 | low | `files_by_language` / `nodes_by_kind` / `edges_by_kind` are keyed by the indexer's registered language and kind sets — bounded by the registry, not by user input. | plan 04-03 | 2026-08-29 |
| R-04-06 | T-04-16, T-04-19 | low | Symbol names and paths render via Svelte text interpolation, which escapes by construction. The phase's only raw-HTML site remains Phase 3's single highlighting helper. | plans 04-04, 04-05 | 2026-08-29 |

---

## Notes

**Note 1 — T-04-31 was discharged by escalation, not violated.** The threat's control was
*"the over-threshold branch HALTS rather than installing `@tanstack/svelte-virtual`."*
The render-cost measurement did exceed its pre-set thresholds (408–421ms vs 400ms;
363–480ms vs 200ms, median-of-5, reproduced across 4 runs), and the executor **did halt**
and escalate rather than installing. The maintainer then approved the install
(2026-08-30, verbatim: *"Approve @tanstack/svelte-virtual and wire it"*), after the
orchestrator verified the package against live npm: created 2022-07-19 (~4 years old),
`TanStack/virtual`, one transitive dep (`@tanstack/virtual-core@3.17.8`), ~78k weekly
downloads — the same `too-new` false-positive shape adjudicated twice earlier in this
phase. The control performed exactly as designed; the decision it reserved for a human
was made by one. Post-virtualization: 17.4ms / 10.8ms, both well under threshold.

**Note 2 — code review found two Critical defects that no threat had anticipated.** The
deep review (`04-REVIEW.md`) raised CR-01 (a stale `Code.Canceled` rejection overwriting
`idle` state) and CR-02 (`/health` rendering an engine placeholder as a live trust
signal). Neither maps to a threat row — they are correctness and integrity-of-presentation
defects rather than security threats. Both are fixed. CR-02 is worth recording here
because its failure mode is adjacent to security: a page whose stated purpose is
conveying trustworthiness was displaying a fabricated all-zero measurement.

**Note 3 — the re-review found three further defects introduced by the fix pass**, two of
which were vacuous guards (`toBeLessThanOrEqual(5)` satisfied by 0; a failure message
rendered alongside the "No results." string it was raised against). All three are fixed
and verified to discriminate. This is the third consecutive phase in which a fix pass
introduced new defects — recorded as an operational property, not an incident.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-30 | 35 | 35 | 0 | /gsd-execute-phase security gate (orchestrator, ASVS L1 inline; auditor short-circuited per threats_open:0 + register_authored_at_plan_time:true + asvs_level:1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in the Accepted Risks Log
- [x] No accepted risk at or above the `high` blocking threshold
- [x] `threats_open: 0` confirmed
- [x] Every zero-count verification paired with a positive control
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-30
