---
phase: 06
slug: benchmark-de-coupling-memory-sweep
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 26
threats_closed: 26
threats_mitigated: 24
threats_accepted: 2
threats_high: 15
register_rows: 31
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 06 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. All 6 PLAN files carry a `<threat_model>` block, so
`register_authored_at_plan_time: true`. With `asvs_level: 1` and `threats_open: 0`, the L1
short-circuit applies; verified against the tree at commit `368ea01`.

**Register-shape note.** 26 distinct threat IDs across **31** rows: three IDs (`T-06-02`,
`T-06-08`, `T-06-22`) are reused in two plans each. Unlike Phase 2 — where a reused ID meant *the
same threat re-asserted at a plan-specific severity* — here the reuse covers **different
components**. `T-06-02` is `internal/bench/regression_test.go` (the oracle) in 06-03 but the
*census exclusion list* in 06-04; `T-06-22` is the publish job's D-06 properties in 06-02 but the
"fires against the committed baseline" claim in 06-03. Both readings are recorded below rather than
merged, because merging them would lose a component.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| runner stdout → committed measurement artifact | The artifact becomes the source every published figure is generated from; a divergence between what ran and what was recorded is unfalsifiable later | Metrics JSON |
| `06-PUBLISH-RESULTS.json` → `docs/BENCHMARKS.md` | Published figures are a public performance claim | Throughput / latency / RSS |
| `bench.yml` → third-party actions & package managers | The workflow is a supply-chain surface; a `rebless` job that gains `contents: write` becomes a write path to the baseline | Action refs, permissions |
| mutation rehearsal → `internal/bench/` working tree | A rehearsal deliberately breaks the gate; a missed revert lands mutation bytes | Source bytes |
| rehearsal → `internal/bench/regression_test.go` | The oracle must not be edited by the rehearsal that tests it | Oracle bytes |
| memory sweep → engram spine records of real history | Records document what actually happened; overwriting one destroys history rather than correcting it | Durable records |
| memory sweep → `PROJECT.md` / `STATE.md` structure | Both are parsed by scope-sensitive readers; a heading change silently alters what tooling sees | Headings, front-matter keys |
| memory sweep → `NOTICE` / `LICENSE` / `CHANGELOG.md` | Legally- or tool-owned files that the sweep must not touch | Attribution, generated log |

---

## Threat Register

### High

| Threat ID | Plan | Component | Mitigation & verification | Status |
|-----------|------|-----------|---------------------------|--------|
| T-06-01 | 06-03 | `internal/bench/regression.go` during the D-08 rehearsal | Pre-mutation `git diff --quiet` gate, shell `trap … EXIT` restoring the file on any exit path, empty post-revert diff. **Verified:** `internal/bench` + `tools/bench` **0 dirty paths**; 9 trap/revert markers in the log | closed |
| T-06-02 | 06-03 | `internal/bench/regression_test.go` — **the oracle** | Asserted unchanged against the recorded **PRE-PLAN** SHA, not `HEAD` (which an earlier commit in the same plan would already have moved) | closed |
| T-06-02 | 06-04 | the phase acceptance census **exclusion list** | Every exclusion cited to a recorded prior adjudication in `06-CENSUS.md`; the instrument positive-controlled before any zero is trusted | closed |
| T-06-03 | 06-02 | `bench.yml` `rebless` job — Elevation of Privilege | Task 1 asserts the diff added or removed **no** `permissions:` / `contents: write` line against the recorded pre-plan SHA | closed |
| T-06-04 | 06-02 | third-party action pins in `bench.yml` | An arithmetic assertion over three printed counts that **cannot pass at zero**. **Verified live:** `USES_TOTAL=21`, `USES_PINNED=19`, `USES_LOCAL=2`, and 19 + 2 = 21 — matching the plan's recorded expectation exactly | closed |
| T-06-06 | 06-01 | `tools/bench/baseline.json` via the runner | Publish mode never reads `-rebless` and never writes a baseline; asserted by `git diff --quiet "$BASE" -- internal/bench/ tools/bench/baseline.json` | closed |
| T-06-09 | 06-02 | supply chain reachable from the workflow | The global `npm install -g` of an external indexer is **deleted**. **Verified:** `bench.yml` now contains **zero** package-manager installs of any kind (`npm`/`pip`/`cargo`/`go install`) | closed |
| T-06-11 | 06-03 | `tools/bench/baseline.json` | Every task asserts `git diff --quiet "$BASE" -- tools/bench/baseline.json` plus `test -f`, so neither an intervening commit nor a deletion passes. **Verified:** file present | closed |
| T-06-12 | 06-04 | `docs/BENCHMARKS.md` figures — Spoofing | Every published figure is **generated** from the committed raw `06-PUBLISH-RESULTS.json` by `publishcheck -emit-rows` and re-matched verbatim, so all ten cells are traceable. No figure is typed by hand. **Verified:** raw JSON committed, `-emit-rows` present, `publishcheck` **8 PASS** | closed |
| T-06-14 | 06-05 | `PROJECT.md` / `STATE.md` structure | Value-only edits; a prohibition forbids adding, renaming or removing any heading, front-matter key or marker. **Verified:** `## Core Value` heading intact, 11 STATE.md front-matter keys intact | closed |
| T-06-15 | 06-05 | project history in `PROJECT.md` | Enumerate-then-edit ordering; every changed line must map to a row whose recorded verdict was `sweep`; the keep-historical population is enumerated first | closed |
| T-06-18 | 06-06 | engram records of **real project history** | Enumerate → classify → approve ordering, a blocking `checkpoint:decision` before the first write, and **correction by supersede only** — a prohibition forbids deletion. **Verified:** 4 supersede records live in the spine (`mw5z9s9bft`, `b9wjge7375`, `gxwkk3necn`, `xj1stbrsw6`), each preserving the original | closed |
| T-06-19 | 06-06 | the completeness claim | Per-scope page counts recorded; verdict counts must **sum to** the enumerated total (169 records); the rule population named after live discovery | closed |
| T-06-21 | 06-06 | an unavailable or partially readable store — DoS | The precondition **halts** the task rather than degrading to a partial sweep; a prohibition forbids recording completion against a scope that was not fully read | closed |
| T-06-22 | 06-02 | the publish job's own D-06 properties | All four properties asserted against the **parsed `jobs.publish` subtree**, never by a file-wide grep another job could satisfy. **Verified:** `TestBenchPublishJobShape` **PASS** | closed |
| T-06-22 | 06-03 | the claim that the gate fires "against the committed baseline" | The test loads `tools/bench/baseline.json` **from disk** and passes it as the baseline argument. **Verified:** `TestCheckRegressionAgainstCommittedBaseline` **PASS** with three subtests — one in-frame pass and **two proving it fails** (throughput 11% slower; peak RSS 16% larger) | closed |
| T-06-23 | 06-04 | BENCH-03's completion claim — Repudiation | A blocking `checkpoint:decision` resolves BENCH-03 into exactly one of two verbatim tokens, asserted present exactly once. **Verified:** `BENCH03_STATUS=closed-by-ci-run`, **1** occurrence | closed |

### Medium

| Threat ID | Plan | Component | Status |
|-----------|------|-----------|--------|
| T-06-07 | 06-01 | `resolveOrClone` corpus fetch pinned to `Entry.CommitSHA`, never `HEAD` | closed |
| T-06-08 | 06-01 | the framing census — positive control planted before any zero is trusted | closed |
| T-06-08 | 06-02 | the bounded census asserts legitimate `comparison` uses **survive** (count ≥ 5), so a green census cannot have been bought with a find-and-replace | closed |
| T-06-10 | 06-03 | the rehearsal record — output pasted verbatim, a category-error RED explicitly disqualified. **Verified:** 46 pasted FAIL/regression blocks | closed |
| T-06-16 | 06-05 | `NOTICE` / `LICENSE` / `CHANGELOG.md` — two-sided guard. **Verified:** all three present; NOTICE's verbatim upstream copyright line intact (1 match) | closed |
| T-06-17 | 06-05 | the swept wording's durability — source-before-mirror rule so the next regeneration cannot restore retired text | closed |
| T-06-20 | 06-06 | tool-surface assumptions verified live in-session rather than hard-coded from LOW-confidence research | closed |
| T-06-24 | 06-04 | committed measurement vs the runner's actual stdout — teed to a temp file in the same pipeline and compared with `cmp` | closed |
| T-06-25 | 06-05 | `STATE.md`'s core-value statement asserted as normalised byte equality against `PROJECT.md`'s `## Core Value`, with non-empty and minimum-length floors | closed |
| T-06-26 | 06-05 | MEM-02's file-half completion claim — exactly one status token, present exactly once. **Verified:** `MEM02_FILES_STATUS=verified-fresh-session`, 1 occurrence | closed |

### Accepted

| Threat ID | Plan(s) | Component | Status |
|-----------|---------|-----------|--------|
| T-06-13 | 06-04 | measurement artifacts committed under `.planning/` (information disclosure) | closed |
| T-06-SC | 06-01, 06-04, 06-06 | npm/pip/cargo installs | closed |

**Totals:** 26 distinct threats over 31 rows — 24 mitigated, 2 accepted, **0 open**. High-severity: 15, all verified. No critical threats in this phase.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-06-01 | T-06-13 | The runner emits only counts, durations, RSS figures and platform labels — no path outside the scratch root, no credential, no environment capture. Committing them under `.planning/` publishes performance data that `docs/BENCHMARKS.md` publishes anyway. | plan author (06-04) | 2026-08-16 |
| R-06-02 | T-06-SC | This phase adds **zero** package-manager installs. Its only package-manager change is a **removal** — the `npm install -g` of the comparison binary, deleted in 06-02. The Package Legitimacy Gate governs additions; a removal reduces supply-chain surface and is recorded here rather than omitted. | plan authors (06-01, 06-04, 06-06) | 2026-08-16 |

---

## Acknowledged Gaps

### MEM-02 store half — accepted, not demonstrated

This is a **verification** gap, not an open threat, and is recorded here for completeness because it
touches the same status-token mechanism `T-06-23` and `T-06-26` establish.

`MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` appears exactly once in
`06-LIVE-VERIFICATIONS.md` and was deliberately **not** upgraded to `verified-fresh-session`.
Maintainer ruling 2026-08-16, verbatim: *"MEM-02 - accept it and move on."* No session that was both
genuinely fresh **and** engram-tooled ever observed the spine recall.

The threat model anticipated exactly this outcome: `T-06-26`'s mitigation reads *"the fresh-session
read may go unperformed but cannot go unrecorded."* That is the property that held. The distinction
between accepted and demonstrated is preserved in the token itself rather than smoothed away, which
is what makes the gap auditable later. Full detail in `06-VERIFICATION.md ## Acknowledged Gaps` and
`.planning/v0.11.0-MILESTONE-AUDIT.md`.

---

## Threat Flags from Execution

No Phase 6 summary carries a `## Threat Flags` section. `06-VERIFICATION.md` records
`status: passed` on re-verification (first pass `human_needed` 3/5) with 5/5 requirements closed,
and the acknowledged gap above stated explicitly rather than folded into the score.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 26 (31 rows) | 26 | 0 | `/gsd-secure-phase 6` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep depth against the tree at commit `368ea01`, plus executed tests for
every mitigation that names one. No phantom commands in this phase — every named test
(`TestBenchPublishJobShape`, `TestCheckRegressionAgainstCommittedBaseline`, `publishcheck`) exists
and passes.

**The strongest control in this register is `T-06-04`'s arithmetic.** Rather than asserting "no
unpinned actions" — a negative that goes vacuous the moment the pattern stops matching — it prints
three counts and asserts `PINNED + LOCAL == TOTAL`. Re-derived independently at audit time:
19 + 2 = 21. A guard shaped this way cannot pass by finding nothing, which is precisely what rule
`84d1gfpywd` requires. `T-06-08`'s census does the same thing from the other direction, asserting
that legitimate `comparison` uses **survive** (count ≥ 5) so a green census cannot have been bought
with a find-and-replace.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
