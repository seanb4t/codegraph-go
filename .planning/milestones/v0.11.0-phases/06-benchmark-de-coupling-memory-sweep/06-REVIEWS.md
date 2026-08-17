---
phase: 6
reviewers: [codex]
reviewed_at: 2026-08-16T18:22:00Z
review_cycle: 6
plans_reviewed: [06-01-PLAN.md, 06-02-PLAN.md, 06-03-PLAN.md, 06-04-PLAN.md, 06-05-PLAN.md, 06-06-PLAN.md]
---

# Cross-AI Plan Review — Phase 6

> This file holds **six** convergence cycles. Cycle 6 (immediately below) is the **confirmation
> pass** after the cycle-5 exhaustive gate-reachability audit: it assesses the CURRENT plan text at
> commit `0c78d3b`. Cycles 5, 4, 3, 2 and 1 are retained further down as audit history — their
> findings are **not** current, and their HIGHs are relabelled RESOLVED where a later cycle
> re-verified the closure. Consult them only to trace how a cycle-6 statement came to be.
>
> **Finding trend across the loop: 24 → 12 → 6 → 2 → 4 → 0.**

---

# Cycle 6 — confirmation pass, assessment of the current plans (`0c78d3b`)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and independently extended by
the orchestrating reviewer with its own measurement against the working tree at `0c78d3b`.

**Verdict: the plans are sound. 0 HIGH, 0 actionable MEDIUM/LOW.**

This is stated plainly because a clean result is the correct outcome here, not a failure to look
hard enough. Every claim the cycle-5 replan makes was re-derived from source rather than accepted on
assertion, using the plans' own verbatim gate commands; the one discrepancy either reviewer raised
was itself run down and **refuted** (see *Refuted findings* below). No eighth instance of the phase's
recurring defect class was found — not in the new exclusions, not in the repointed
`.github/workflows/bench.yml:295`, and not in the audit's own companion assertions.

## Independent verification log — per claim, per gate

Every row below was executed by the orchestrating reviewer at `0c78d3b`, using the pattern verbatim
from the plan site cited, not a paraphrase. Where an earlier sloppy paraphrase disagreed with the
plan, that is recorded too — twice the paraphrase was wrong and the plan was right.

### 1. HIGH 1 closure — glob anchoring and the companion assertions

| Check | Method | Observed | Verdict |
|---|---|---|---|
| Anchored glob keeps `publishcheck/main_test.go` in the surface | built the post-execution tree in a scratch dir (both `main_test.go` files present), ran `rg --files -g '!tools/bench/runner/main_test.go' tools/bench` | `tools/bench/publishcheck/main_test.go` **retained** | claim CONFIRMED |
| Unanchored glob drops it too | same tree, `rg --files -g '!main_test.go' tools/bench` | **no files retained** — both dropped | claim CONFIRMED; forbidding the short form at `06-01:609` and `06-04:592` is correct and load-bearing |
| Why it is not visible on today's tree | `rg --files tools/bench \| rg main_test` | only `tools/bench/runner/main_test.go` exists | `publishcheck/main_test.go` is created by 06-01 (`06-01:13`, `:413`); the hazard is real only post-execution, which is exactly the tree the audit is required to evaluate against |
| Companion baseline on a clean file | `TB`/`RT`/`OT` per `06-01:607` | `TB=1 RT=1 OT=0` → **exit 0** | passes on correct work |
| Companion vs HOLE A — second `ts-binary` smuggled into the excluded file | appended `// ts-binary`, re-ran | `TB=2 RT=1 OT=0` → **exit 1** | companion genuinely REJECTS the hole |
| Companion vs HOLE B — stale `runHeadToHead` hidden in the excluded file | appended `func x(){ runHeadToHead() }`, re-ran | `TB=1 RT=1 OT=1` → **exit 1** | companion genuinely REJECTS the hole |

Both exclusions are bounded, not holes: each of the three sites (`06-01:607`, `06-01:659`,
`06-04:638`) carries companions pinning the excluded scope's content, and the companions were
verified to fail — not merely to pass — which is the only test that distinguishes a bound from a
hole.

### 2. The exhaustive gate-reachability audit — enumeration and re-verdicting

**Per-plan enumeration, counted independently:**

| Plan | Claimed | Distinct `GNN.x` rows measured | Raw row matches | Verdict |
|---|---|---|---|---|
| 06-01 | 16 | 16 (`G01.1`–`G01.16`) | 18 | matches — the 2 extra are `G01.1`/`G01.13` repeated in the four-UNSATISFIABLE summary table |
| 06-02 | 21 | 21 (`G02.1`–`G02.21`) | 22 | matches — 1 repeat (`G02.1`) |
| 06-03 | 24 | 24 | 24 | matches |
| 06-04 | 25 | 25 (`G04.1`–`G04.25`) | 26 | matches — 1 repeat (`G04.19`) |
| 06-05 | 21 | 21 | 21 | matches |
| 06-06 | 15 | 15 | 15 | matches |
| **Total** | **122** | **122** | 126 | **claim CONFIRMED** |

A first pass counted 126 and looked like an over-count in the plans' favour; running the ids down
showed all four extras are the same gates re-listed in the *four UNSATISFIABLE gates* summary table
at `06-01:234-241`. The enumeration is exact.

**Completeness of the enumeration — per zero-expectation gate.** The defect class lives in `= 0`
gates over surfaces the phase itself writes to, so every such gate in all six plans was extracted by
variable name and mapped back to an audit row:

| Plan | `= 0` gate variables | All present in audit table? |
|---|---|---|
| 06-01 | `M`, `N`, `OT`, `PINDIFF`, `X` | yes |
| 06-02 | `DIAGDIFF`, `FIGDIFF`, `FIXDIFF`, `PERMDIFF` | 3 of 4 as rows; `DIAGDIFF` recorded in `G02.1`'s fix column (see note) |
| 06-03 | `CATERR`, `N`, `TOLDIFF`, `X` | yes |
| 06-04 | `N`, `TOTAL`, `XOT` | yes |
| 06-05 | `CM`, `M`, `N` | yes |
| 06-06 | `DEL` | yes |

*Note on `DIAGDIFF`, and why it is not a finding.* The audit's convention is that assertions the
audit's **own fixes** introduce are recorded in the fixed gate's row rather than promoted to new
numbered rows — the same treatment `TB`/`RT`/`OT` receive under `G01.1`, `G01.13` and `G04.19`. So
the 122 count excludes the seven fix-companions by construction. This is consistent, disclosed at the
fix sites, and — decisively — **every one of those seven companions was independently measured here
and holds** (rows 1 and 3 of this log). No gate is left unverified, so there is no PLAN.md change to
make; reporting this as actionable would be manufacturing a finding.

**Independent re-verdicting of SATISFIABLE gates whose scope the phase writes to.** Two were chosen
specifically because the audit records them at **zero headroom** — the least forgiving verdicts in
the table, and the ones an under-measured basis would break first:

| Gate | Audit's basis | Sloppy paraphrase | Gate's **verbatim** pattern | Verdict |
|---|---|---|---|---|
| `G02.16` `SHAPETEST_FIXTURE_ENTRIES >= 10` | "measured 10 — exactly at the floor" | `\{Workflow:` → **11** | `^\s*\{Workflow:` (per `06-02` `FIXNOW=`) → **10** | **audit CORRECT**; my 11 wrongly included the inline `bad := []runBodyException{{Workflow: …}}` at `taskfile_shape_test.go:1392` |
| `G02.19` `comparison` in `BASELINE.md` `>= 5` | "measured 5, zero headroom, at `:39 :87 :97 :120 :297`" | `-i 'comparison'` → **6** | `\bcomparison\b` (per `06-02:457`) → **5**, at exactly `:39 :87 :97 :120 :297` | **audit CORRECT**; my 6th was `comparisons` at `BASELINE.md:260`, which the word boundary properly excludes |
| `G02.13` `BASELINE_AND_SHAPETEST_RETIRED_TERMS_TOTAL = 0` | "3 hits, all in declared scope" | — | census family over both files → **3**, at `BASELINE.md:64`, `BASELINE.md:241`, `taskfile_shape_test.go:102` | **audit CORRECT**; all three are named in 06-02 Task 2's `<action>` |
| `G04.20` `BARE_TS_RESIDUAL_LINES >= 1` | "`corpora/manifest.json:10`, plus the excluded `main_test.go` line" | — | `\bTS\b` over the census surface → **62** lines across 14 files; `corpora/manifest.json:10` carries `TS/JS` inside a `LOCKED (01-06)` note, and 06-03's only permitted edit to that file is line 2 (`06-03:52`) | **audit CORRECT**, and the floor is well protected — the KEEP-class line is explicitly un-editable |

The audit survived adversarial re-verdicting on the two gates most likely to break. In both cases the
audit's number was right and the looser pattern was wrong, which is the outcome that actually
evidences a mechanically-derived basis rather than a reasoned one.

### 3. The seventh instance — `bench.yml:295` measurements

| Claim | Command | Observed | Verdict |
|---|---|---|---|
| 26 census hits over `bench.yml` | census family verbatim from `06-04:638` (`rg -U -o -i …`) | **26** | CONFIRMED |
| 25 in-scope | hits at `:4 :5 :14` (header 1-14), `:42 :44 :46` (42-52), `:69 :71` (69-73), `:96 :99 :117 :122×2 :123×2 :130 :131 :134 :138 :140 :141 :146 :149 :156 :157` (96-158) | **25** | CONFIRMED |
| exactly one out of scope, at `:295` | same run | **`295:headtohead`**, the only hit above 157 | CONFIRMED |
| `:295` is a dangling reference | `sed -n '288,300p'`; job ids via `rg -n '^  [a-z][a-z0-9-]*:$'` | `:295` sits in the preamble comment of `cpu-diag-github` (job at `:313`) and names the `headtohead` job (`:96`) that 06-02 deletes | CONFIRMED — repointing, not excluding, is the right remedy, and matches the `BASELINE.md:241` / `taskfile_shape_test.go:102` precedent |
| companion `DM=34` | `rg -o -e 'cpu-diag' -e 'scratch-fs-compare' -e 'disk-control' bench.yml \| wc -l` | **34** | CONFIRMED |
| companion `CD=1` | `rg -o -F 'NOT a gate and NOT part of the'` | **1** | CONFIRMED |
| companion `NB=1` | `rg -o -F 'ever touches baseline.json'` | **1** | CONFIRMED |
| none of the 34 falls inside a region 06-02 edits | line numbers of all 34 filtered against 1-14, 42-52, 69-73, 96-158 | **in-region count = 0** | CONFIRMED — so `DIAG_JOB_DIFF_LINES=0` is reachable |

One point worth recording for the executor, though it is not a defect: three of the 34 mentions sit
at `bench.yml:74-76`, immediately below the `69-73` dispatch-inputs region and **inside the same
`options:` list**. The authorised edit removes `- headtohead` at `:71` and repoints `default:` at
`:69`; with `git diff -U0` that produces no `+`/`-` line at `:74-76`, so `DIAGDIFF=0` holds. It would
only break if the executor reflowed the list — which the `<action>` does not ask for and the
companion would catch by name. The bound is tight but correct, and the companion is precisely the
instrument that makes the tightness safe.

### 4. LOW 1 — the per-heading gate, reproduced

Counterexample document (`## 1. Methodology` duplicated, regression-gate heading absent):

| Form | Result |
|---|---|
| OLD summed gate (`H=3`) | `H=3` → **exit 0** — accepts the malformed document |
| NEW per-heading gate (`06-04:384`) | `HEADING[## 1. Methodology]=2` → **exit 1**, and it **names the duplicated heading** |

CONFIRMED exactly as claimed, including the naming behaviour. This is the phase's own *per item,
never aggregate* rule correctly applied to headings.

### 5. The three near-misses — genuine, not defects wearing a rationale

`06-03`'s `TOLERANCE_DIFF_LINES = 0` (`G03.6`) was checked most closely, since it is the one whose
regex admits comment text:

- Lines in `internal/bench/regression.go` matching the gate's regex
  (`DefaultThroughputTolerance|DefaultRSSTolerance|= 0\.`): **`5, 9, 11, 15, 22, 23, 113, 116, 121,
  124`** — exactly the **ten** lines the audit names.
- The `<action>`'s sweep target is `:37-48`. **None of the ten is inside it.** Verified by inspection
  of `:33-52`, which is the platform-mismatch block.
- `TOLERANCE_CONSTANTS_PRESENT` measures **2** today, matching the gate's `= 2` companion.

So the gate is satisfiable on the correct path, the bound is by instruction (`06-03:241`, `:248`),
and the gate itself is unchanged. **Bounding a pattern is not weakening a gate** — nothing was
deleted here, and the companion `TOLERANCE_CONSTANTS_PRESENT = 2` keeps the constants pinned. This
is a genuine near-miss correctly recorded rather than an unfixed defect.

`G03.23` (`CRITICAL_ORACLE_CASES_PRESENT` scoped to `regression_test.go` while `baseline_gate_test.go`
mirrors both names) and `G02.19`/`G02.16` (the two zero-headroom floors) are likewise genuine: each
is satisfiable as written, each records *why* a plausible future edit would break it, and none
requires a plan change now.

### 6. Eighth-instance sweep — negative result

The three surfaces most likely to carry a new instance were checked directly:

- **The new exclusions** — bounded and hole-rejecting (section 1). The bare-`TS` residual control
  (`G04.20`) still walks the excluded file, so the narrowing cannot hide a regression there.
- **The repointed `bench.yml:295`** — the edit changes one job-id word; all four companion numbers
  measured true, and zero of the 34 protected identifiers falls in an edited region (section 3).
- **The audit's own companions** — all seven measured and holding (sections 1 and 3).

Additionally, every file under the census surface carrying retired terms was mapped to the plan
licensed to remove it. `internal/bench/regression_test.go:120`, `internal/bench/metrics.go:6` and
`internal/bench/rss.go` are all inside `06-03`'s declared `files_modified` (`06-03:8-11`, `:192`),
so no census hit is orphaned in a file no plan may touch — which is the exact shape all seven prior
instances took.

**No eighth instance found.**

## Refuted findings

Codex raised exactly one discrepancy, and flagged it non-actionable itself. It is nonetheless
**refuted** rather than merely deprioritised:

> Codex: "The repeated historical claim '26 total / 25 in scope / 1 outside' is off by one. The
> actual current source yields 25/24/1."

Run down: the plans' census pattern at `06-04:638` is `rg -U -o -i` — **case-insensitive**.
`.github/workflows/bench.yml:146` carries `Head-to-head` with a capital H.

| Pattern form | Hits over `bench.yml` |
|---|---|
| `rg -U -o -i` (the plans' verbatim form) | **26** |
| `rg -U -o` (case-sensitive) | **25** |

The 26th hit is `146:Head-to-head`, matched only with `-i`. Codex re-derived the pattern family
without `-i` and undercounted by one. **The plans' 26/25/1 is correct as written**; no change is
required, and the audit history should not be amended.

## Strengths

- **The exclusions are bounds, not holes, and this was proven by making them fail.** `06-01:607` and
  `06-04:638` pair the path-anchored glob with `TB=1 / RT=1 / OT=0`; both constructed holes went red.
  An exclusion whose companion has only ever been seen to pass is not evidence — these have been seen
  to fail.
- **The anchoring distinction is real and correctly directed.** Verified on a simulated
  post-execution tree: anchored retains `tools/bench/publishcheck/main_test.go`, unanchored drops it.
  Forbidding the short form **by name** at `06-01:609` and `06-04:592` is the right level of
  specificity — a prose warning would not have survived an executor reaching for the obvious glob.
- **The 122-gate audit is exact and its bases are genuinely measured.** Enumeration confirmed
  per plan; and on the two zero-headroom gates the audit's numbers beat a looser independent
  re-measurement, which is the signature of a mechanically-derived basis rather than a reasoned one.
- **The seventh instance was fixed at its source, not excluded.** `bench.yml:295` names a job id
  06-02 deletes; repointing it is correct, and the four-number companion (`DIAGDIFF=0 CD=1 NB=1
  DM=34`) converts the authorisation into a bound rather than a licence.
- **`G04.20` is the right kind of control.** A `>= 1` residual that still walks the excluded file
  makes the exclusion self-policing, and its surviving source (`corpora/manifest.json:10`) is a
  LOCKED KEEP-class line that no plan may edit — so the floor cannot be swept away by correct work.
- **The standing rule at `06-01:405-410`** ("before adding or changing an expected-value gate,
  evaluate it against the **post**-execution tree in its declared scope, per file, and record the
  basis") is the correct generalisation of all seven instances, and is the durable fix.

## Concerns

**None.**

Zero HIGH. Zero actionable MEDIUM/LOW. The plans are sound at `0c78d3b` and this loop has converged.

## Suggestions

**None.** No plan change is recommended. On a confirmation pass at this depth, a suggestion that is
not backed by a reproduced failure is a cost, not a contribution.

## Risk Assessment

**LOW** for plan correctness.

The residual risks are environmental and are already named in the plans, not defects to fix here:

- Go and network-dependent checks need a writable build/temp environment (both reviewers hit sandbox
  denials running `go test`; no test failure was observed, the command could not start).
- `06-06` must halt unless the engram tools are actually registered and enumerable, exactly as D-16
  requires — which the plan already gates as a blocking checkpoint.

No LOCKED decision (D-01..D-16) needs re-opening.

---


## Verification coverage — Cycle 6 (source-grounding, per item)

Stated per file / per gate, never as an aggregate — cycle 3 refuted an aggregate claim that
concealed exactly one counterexample, and that counterexample was a HIGH.

| # | Claim under test | Source touched | Executed? | Result |
|---|---|---|---|---|
| 1 | anchored vs unanchored glob | simulated post-execution tree (`runner/` + `publishcheck/` both present) | yes | anchored retains publishcheck; unanchored drops both — claim CONFIRMED |
| 2 | companion rejects HOLE A (2nd `ts-binary`) | `06-01:607` predicate | yes | `TB=2` → exit 1 — REJECTS |
| 3 | companion rejects HOLE B (stale `runHeadToHead`) | `06-01:607` predicate | yes | `OT=1` → exit 1 — REJECTS |
| 4 | 122-gate enumeration, per plan | `06-01` `<gate_reachability_audit>` | yes | 16/21/24/25/21/15 = 122 CONFIRMED (126 raw − 4 summary repeats) |
| 5 | every `= 0` gate mapped to an audit row | all six PLAN.md | yes | 20 variables; 19 as rows, `DIAGDIFF` in `G02.1`'s fix column per the audit's stated convention, and independently measured |
| 6 | `G02.16` re-verdict (zero headroom) | `internal/upgrade/taskfile_shape_test.go` | yes | verbatim `^\s*\{Workflow:` → **10** — audit CORRECT |
| 7 | `G02.19` re-verdict (zero headroom) | `tools/bench/BASELINE.md` | yes | verbatim `\bcomparison\b` → **5** at `:39 :87 :97 :120 :297` — audit CORRECT |
| 8 | `G02.13` re-verdict | `BASELINE.md`, `taskfile_shape_test.go` | yes | 3 hits, all in declared scope — audit CORRECT |
| 9 | `G04.20` re-verdict (`>= 1` floor) | census surface, `corpora/manifest.json:10` | yes | 62 residual lines / 14 files; KEEP-class line un-editable per `06-03:52` — audit CORRECT |
| 10 | `bench.yml` 26 / 25 / 1 split | `.github/workflows/bench.yml` | yes | 26 total, 25 in-region, 1 at `:295` — CONFIRMED |
| 11 | `:295` is a dangling reference | `bench.yml:288-300`, job ids | yes | preamble of `cpu-diag-github` (`:313`) naming deleted job `headtohead` (`:96`) — CONFIRMED |
| 12 | companions `DM=34 / CD=1 / NB=1` | `bench.yml` | yes | 34 / 1 / 1 — CONFIRMED |
| 13 | none of the 34 in an edited region | `bench.yml`, regions 1-14/42-52/69-73/96-158 | yes | in-region count **0** — CONFIRMED |
| 14 | LOW 1 old gate accepts counterexample | `06-04:384` predecessor | yes | `H=3` → exit 0 — CONFIRMED |
| 15 | LOW 1 new gate rejects and names it | `06-04:384` | yes | `HEADING[## 1. Methodology]=2` → exit 1 — CONFIRMED |
| 16 | `TOLDIFF` near-miss genuine | `internal/bench/regression.go` | yes | 10 matching lines `5,9,11,15,22,23,113,116,121,124`; none in `:37-48` — CONFIRMED |
| 17 | census hits all owned by some plan | `internal/bench/*`, census surface | yes | `regression_test.go:120`, `metrics.go:6`, `rss.go` all in `06-03` `files_modified` — no orphan |
| 18 | Codex's 25/24/1 discrepancy | `bench.yml`, `-i` flag | yes | `-i` → 26, no `-i` → 25; 26th is `146:Head-to-head` — Codex REFUTED |

Not executed: `go build` / `go vet` / `go test` (sandbox denied Go's temp build dir for both
reviewers — no test failure was observed; the command could not start). This affects no finding
above, all of which are lexical/structural gates evaluated directly.

## Codex Review — Cycle 6 (verbatim, source-grounded)

## Summary

The six plans are sound at commit `0c78d3b`. I found no current HIGH, MEDIUM, or actionable LOW concerns and no eighth instance of the “correct execution cannot satisfy the assertion” defect class. The cycle-5 exclusions are narrowly scoped and companion-pinned, the 122-gate audit enumerates the claimed 16/21/24/25/21/15 gates exactly, the workflow dangling reference is correctly assigned to 06-02, and the tolerance-comment issue is genuinely a bounded near-miss. One audit-history measurement is off by one—25/24/1 workflow hits rather than 26/25/1—but no executable gate depends on that historical count.

## Verification log

### Repository and source access

- `git rev-parse HEAD`
  - Output: `0c78d3b4625cda09026ab3190f61adff01226ffa`
  - Exit: 0
- `git status --short`
  - Output: empty
  - Exit: 0
- `codegraph explore ...`
  - Failed because the read-only sandbox denied access to `.codegraph/store/LOCK`.
  - I used read-only `rg`, `sed`, and git-object inspection instead.

### 1. Exclusion and companion assertions

Glob behavior:

```text
rg --files . -g '!tools/bench/runner/main_test.go' | ...main_test.go
→ 5 other main_test.go files retained

rg --files . -g '!main_test.go' | ...main_test.go
→ 0 files retained
```

Thus the short glob is indeed unanchored and over-excludes, while the mandated full path only removes the runner test. This supports the warning in `06-01-PLAN.md:607` and `06-04-PLAN.md:638`.

Companion predicates, exercised with synthesized stdin:

```text
CLEAN_COMPANION_EXIT=0 TB=1 RT=1 OT=0
SECOND_FLAG_HOLE_EXIT=1 TB=2
STALE_SYMBOL_HOLE_EXIT=1 OT=1
```

The companions genuinely reject both requested holes:

- A second `ts-binary` fails `TB=1`.
- A hidden `runHeadToHead` fails `OT=0`.

The three exclusion sites all carry companions rather than bare exclusions:

- `06-01-PLAN.md:607`
- `06-01-PLAN.md:721`
- `06-04-PLAN.md:638`

The current tree does not yet contain `tools/bench/publishcheck/main_test.go`; it is created by 06-01. The repository-wide glob test nevertheless proves the exact glob semantics the plan relies on.

### 2. Gate-reachability audit

I independently bounded each per-plan audit subsection and counted its `GNN.*` rows:

```text
06-01 = 16
06-02 = 21
06-03 = 24
06-04 = 25
06-05 = 21
06-06 = 15
SUM   = 122
```

This exactly matches the declared audit at `06-01-PLAN.md:218-405`.

A broader lexical scan found many more gate-bearing tokens—such as repeated `test`, `go test`, `diff`, and `rg -q` references—because the plans repeat gates in action, verification, acceptance, review-history, and threat-model prose. Inspection of the actual audit table confirmed it enumerates the distinct expected-value gates rather than every textual occurrence. I found no omitted executable gate form that creates a new unsatisfiable post-execution assertion.

Sampled write-surface gates included:

- Runner exclusion and companions: `06-01-PLAN.md:605-615`
- Workflow census and diagnostic companion: `06-02-PLAN.md:406-408`
- Tolerance-diff gate: `06-03-PLAN.md:296`
- Phase census and exclusion companions: `06-04-PLAN.md:638-659`
- Generated-file equality and bounds: `06-05-PLAN.md:332-353`
- Approved/executed set equality: `06-06-PLAN.md:345-373`

All sampled gates are satisfiable under their prescribed post-execution content.

### 3. Workflow hit split and diagnostic population

Using the exact retired-term pattern family over `.github/workflows/bench.yml`:

```text
TOTAL=25
IN_DECLARED_EDIT_REGIONS=24
OUTSIDE_DECLARED_EDIT_REGIONS=1
```

The sole outside hit is:

```text
295: # NOT a gate and NOT part of the rebless/headtohead flow — neither job
```

The diagnostic identifier population is:

```text
DIAG_MENTIONS=34
```

This confirms the substantive claim: exactly one dangling reference exists outside the ordinary rewrite regions, and the companion’s expected `DM=34` is correct. The source occurrence is at `.github/workflows/bench.yml:295`; its authorized repoint and companion are specified at `06-02-PLAN.md:350-376` and `06-02-PLAN.md:406-408`.

The repeated historical claim “26 total / 25 in scope / 1 outside” is off by one. The actual current source yields 25/24/1. This does not affect execution because neither 25 nor 26 is an asserted post-edit value; the executable requirement is zero retired terms after the rewrite.

### 4. Heading counterexample

Counterexample:

```markdown
## 1. Methodology
## 1. Methodology
## 2. Absolute numbers
```

Observed:

```text
OLD_SUM_EXIT=0 OLD_SUM=3
NEW_PER_HEADING_EXIT=1
```

The old summed gate accepts the malformed document. The new independent per-heading equality rejects it, as required by `06-04-PLAN.md:388-397`.

### 5. Tolerance near-miss

Actual matching locations in `internal/bench/regression.go`:

```text
5, 9, 11, 15, 22, 23, 113, 116, 121, 124
```

The authorized sweep range at lines 37–48 produced:

```text
TARGET_MATCH_EXIT=1
```

That means there are zero matches within the prescribed edit range. The source confirms the tolerance comments and constants lie outside it at `internal/bench/regression.go:5`, `:9`, `:11`, `:15`, `:22-23`, while the planned rewrite is constrained to `:37-48` by `06-03-PLAN.md:236-249`.

This is genuinely a near-miss bounded by a precise instruction, not an unfixed contradiction.

### 6. Targeted Go tests

Attempted:

```text
go test -count=1 ./internal/bench/... ./internal/upgrade/... ./tools/bench/...
```

The sandbox blocked creation of Go’s temporary build directory:

```text
go: creating work dir: mkdir .../go-build...: operation not permitted
```

No test failure was observed; the test command could not start under the read-only permission profile.

## Strengths

- The exclusions are bounded and mechanically pinned. The exact-path glob plus `TB=1`, `RT=1`, and `OT=0` prevents the runner-test exclusion from becoming a hiding place (`06-01-PLAN.md:605-615`, `06-04-PLAN.md:638`).

- The gate audit is genuinely per-gate. Its six sections contain exactly the claimed 122 rows, including post-execution reasoning for gates over files the phase itself edits (`06-01-PLAN.md:248-404`).

- The workflow correction fixes a real dangling identifier at its source rather than excluding it. The diagnostic jobs retain a separately measured 34-identifier population (`.github/workflows/bench.yml:295`, `06-02-PLAN.md:350-376`, `:406-408`).

- The heading assertion now enforces each required section independently. The reproduced counterexample distinguishes the new mechanism from the old aggregate check (`06-04-PLAN.md:388-397`).

- The tolerance near-miss is explicitly scoped away from sensitive declarations and comparison branches. Actual source locations agree with the plan’s stated range (`internal/bench/regression.go:5-23`, `:113-124`; `06-03-PLAN.md:236-249`).

- External one-way memory writes remain guarded by live tool discovery, exhaustive enumeration, and a blocking approval checkpoint. The plan does not pretend the unavailable/unregistered engram surface can be inferred (`06-06-PLAN.md:169-261`, `:295-324`).

## Concerns

None.

Non-actionable verification note: the audit-history prose reports 26/25/1 workflow hits, while the current source measures 25/24/1. The sole outside-region hit and all executable expectations remain correct.

## Suggestions

None.

## Risk Assessment

**LOW** for plan correctness. The in-repo work is unusually well constrained, with positive controls, exact-set checks, post-execution reachability analysis, and explicit handling of irreversible operations.

Execution still has two expected environmental risks rather than plan defects:

- Go and network-dependent checks require a writable build/temp environment.
- Plan 06-06 must halt unless the engram tools are actually registered and enumerable, exactly as D-16 requires.

---
# Cycle 5 — audit history (superseded by Cycle 6 above)

> **Status: RESOLVED.** Cycle 5's single HIGH, its MEDIUM and its two LOWs were all closed at `0c78d3b`
> and independently re-verified in Cycle 6 above. Retained to trace how Cycle 6's statements came to be.
>
> _Original heading: Cycle 5 — confirmation pass, assessment of the current plans (`0e2aeba` / `7eff0fa`)_

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer with independent measurement against the working tree at `7eff0fa`, plus two
supplementary audits (gate reachability; cross-plan structure).

**Why a fifth cycle exists:** cycle 4 cleared all HIGHs and left two actionable findings, both closed
at `0e2aeba`. This pass exists to confirm those closures against source rather than on assertion, and
to check specifically whether the closure itself introduced a sixth instance of the phase's recurring
defect class. The user has now spent two escalation decisions on this phase, so severity is
calibrated tightly: **HIGH is reserved for a finding that would prevent the phase goal, create unsafe
execution, or invalidate verification.** A clean result is a valid outcome, not a failure to look.

## Verdict — Cycle 5

**1 HIGH. 3 actionable non-HIGH (1 MEDIUM, 2 LOW).** All four cycle-4 claims reproduce exactly
against source, and **the cycle-4 replan (`0e2aeba`) introduced nothing** — every line it changed is
sound. But this pass found a **sixth instance of the recurring defect class** that pre-dates cycle 4
and survived all four prior cycles: `06-01`'s own mandated removal-proof test must contain the
literal `ts-binary`, which three separate `= 0` census gates forbid. It is demonstrated empirically
below, not argued. None of the four findings requires re-opening any LOCKED decision D-01..D-16.

The recurring defect class is *"an assertion the CORRECT path cannot satisfy"*. HIGH 1 is squarely
that, and structurally identical to cycle 3's HIGH 2 (a KEEP-class occurrence in a file the gating
plan cannot modify, matched by a pattern hard-coded inside an automated verify block, with no
authorised escape). Of the three non-HIGH findings, two fail in the **opposite**, safer direction (a
gate that is too permissive; a declaration that is merely incomplete); only MEDIUM 1 could redden
correct work, and it has an explicit in-plan disambiguator a careful executor would follow.

## Verification of the four cycle-4 claims — Cycle 5

Every number below was re-measured by the orchestrator from the repo root, and independently by
Codex. Both agreed on every figure.

### Claim 1 — 06-05 MEDIUM 1: `reimplementation` adjudicated, gate NOT tightened — **VERIFIED**

| Check | Result |
|---|---|
| `rg -n -i -e '\breimplementation\b' -e '\bport\b' .claude/CLAUDE.md` | exactly **2 lines** — `:17` (`- **Licensing**: Original is MIT — port with attribution`, the file's sole bare `port`, which the re-sync removes) and `:83` (`### Option C — Native pure-Go tree-sitter reimplementation…`, the parser-runtime sense) |
| `:83` lies inside the `keep-divergent` stack block | block is `.claude/CLAUDE.md:21-141`; 83 ∈ [21,141]; the task may touch that block only at line 93 |
| `\breimplementation\b` absent from `CLAUDE_MD_FRAMING_TOTAL` | gate set at `06-05-PLAN.md:357` is exactly `parity, drop-in, ported, rewrite of, TS CodeGraph, colbymchenry` — the term is **not** in it |
| `PROJECT.md` Licensing bullet untouched | `git show 0e2aeba --stat` touched only `06-04-PLAN.md` and `06-05-PLAN.md`; `PROJECT.md:246` still carries the D-13 form |
| Re-synced source text scores 0 against the framing gate | gate set over `sed -n '242p;246p' .planning/PROJECT.md` → **0** |
| D-13 fidelity of the KEEP adjudication | `06-CONTEXT.md:152-158` — *"Records that describe what was decided or done at a point in time are true and **must survive untouched**"*. "the project this one began as a reimplementation of" is squarely that class. The adjudication is faithful, and re-authoring the bullet would indeed re-open D-13 (LOCKED) |

**The two-row bookkeeping reconciles.** `06-05-PLAN.md:362` keeps Task 1's `sweep` row for line 17
(accounting for what the edit REMOVED) and adds a `keep-historical` row (what it PUT THERE).
`06-05-PLAN.md:374`'s downstream assertion is **directional** — *"every changed line maps to a row
whose verdict was `sweep`"* — so an additional non-`sweep` row cannot break it. Task 1's row gate
(`06-05-PLAN.md:211`) is `ROWS >= 10 && SWEEP >= 1 && KEEP >= 1 && DIV = 1`; the new row is appended
by **Task 2**, after that gate has already run, and would in any case only raise `ROWS` and `KEEP`,
both floors. `06-06`'s gates slice `## Engram enumeration` / `## Supersedes executed` with `awk` and
use the disjoint verdict vocabulary `supersede` / `leave-historical`, so no downstream count can
collide. **Downstream count collisions: 0.**

### Claim 2 — 06-04 LOW 1: the last bare `\bTS\b` bounded — **VERIFIED**

| Measurement | Claimed | Measured | |
|---|---|---|---|
| Bounded set, whole `docs/BENCHMARKS.md` | 52 | **52** | ✅ |
| Old bare set (as at `a643713`), whole file | 48 | **48** | ✅ |
| Bounded set over carry-forward ranges `:7-48` + `:213-300` | 7 hits / 6 lines at `:39 :42 :46 :215 :237(×2) :240` | **exactly that** | ✅ |
| `TS/JS` indexed-language probe | bare 2 → bounded 0 | bare **2**, bounded **0** | ✅ |
| Retired-sense probe "6/6" | 6/6 | reads as *6 of the 6 flagged lines caught*; all six are caught | ✅ |

The seven hits, verbatim: `39:TS/Node`, `42:TS binary`, `46:TS-CodeGraph`, `215:head-to-head`,
`237:head-to-head`, `237:TS-subject`, `240:head-to-head`.

**The only surviving bare `\bTS\b` in the phase is the deliberate completeness FLOOR.** The sole
executable occurrence is `06-04-PLAN.md:578` (`BARE_TS_RESIDUAL_LINES >= 1`). Every other occurrence
across the six plans is prose inside a `<review_incorporation>` table — `06-01:207`, `06-03:160`,
`06-04:151`, `06-04:587`, `06-05:121`. There is **no** bare `\bTS\b` zero gate anywhere in the phase.

**Bounding does not silently lose coverage.** Dropping bare `\bTS\b` from the document gate does
narrow absolute detection, but `06-04-PLAN.md:578`'s residual control re-runs the bare instrument
over a surface whose **first** entry is `docs/BENCHMARKS.md`, and `:587` requires every residual line
to be quoted and adjudicated in `06-CENSUS.md`. Detection is preserved through adjudication rather
than through a gate that forbids the KEEP sense.

**The expected residual survives the phase.** `06-04-PLAN.md:587` expects `corpora/manifest.json:10`
as the residual line. Checked against source: that line is the **`gohugoio/hugo`** entry, marked
`LOCKED (01-06)`, whose note reads *"supplies TS/JS via its 25 javascript tracked files"*. It is a
Phase-1 corpus pin, **not** the `colbymchenry-codegraph` benchmark entry that 06-01 removes (that one
lives in `tools/bench/realcorpus/manifest.go`). So `BARE_TS_RESIDUAL_LINES >= 1` stays reachable
after 06-01 lands. This was the most plausible route to a sixth instance and it is closed.

### Claim 3 — the self-surfaced fifth instance (§2.x restatement) — **VERIFIED, and sound**

`docs/BENCHMARKS.md`'s heading structure confirms the split: `### Peak RSS is measured at the OS
level, not in-process` spans `:30-49` (contains `:39 :42 :46`), and `### The one real, committed
number: the synthetic regression baseline` spans `:213-241` (contains `:215 :237 :240`). All three of
the latter are cross-references to the head-to-head table being deleted:

```
215: Unlike the head-to-head table above, `tools/bench/baseline.json` — the
237: It is not a head-to-head number (there is no TS-subject equivalent captured
240: live head-to-head run.
```

**The claim that §1 "already directs you to re-author" `:39 :42 :46` is substantiated, not merely
asserted.** `06-04-PLAN.md:290-292` carries the literal instruction *"Do not justify it by contrast
with another runtime."* — which is exactly what those three lines do (they justify the OS-level RSS
measurement by contrast with Node / the TS binary). The restatement at `06-04-PLAN.md:325-345` names
all six lines, states the measurement, and says precisely what "near-verbatim" preserves (the number,
its rationale, its re-blessing procedure) and what it does not (pointers to a deleted table).

**No criterion was weakened.** Every line removed by `0e2aeba` has a strictly stronger or equal
successor: the old bare gate → the bounded gate; *"already reads on its own terms"* → the measured
six-line restatement; Task 2's `<files>` list → the same list **plus** `06-MEMORY-SWEEP.md`; two
acceptance bullets → expanded forms at `06-04:623` and `06-05:412`. Nothing was deleted without a
replacement. Total diff: 60 insertions / 7 deletions across two files.

### Claim 4 — expected-value literals and the sole-declarer claim — **VERIFIED**

| Literal | Asserted | Measured today | |
|---|---|---|---|
| `GSD_MARKERS` | `= 14` | **14** | ✅ |
| `CLAUDE_MD_HEADINGS` | `= 25` | **25** | ✅ |
| `CLAUDE_MD_STACK_BLOCK_ANCHORS` | `>= 12` (16 today) | **16** (per-token 2 / 6 / 8) | ✅ |
| `CLAUDE_MD_FRAMING_TOTAL` | `= 0`; 3 lines to clear | **5 hits on 3 lines** — `7:rewrite of`, `7:colbymchenry`, `7:ported`, `15:TS CodeGraph`, `93:parity` | ✅ |
| `CLAUDE_MD_CHANGED_LINES` | `6..24` | 4 changed lines → 8 diff lines, in range; `06-05:366` adds a record-and-report escape for a legitimate overrun | reachable |

**Sole-declarer claim confirmed, per file:** `06-01`, `06-02`, `06-03`, `06-04` and `06-06` contain
**zero** references to `.claude/CLAUDE.md`; `06-05` has 50. Only `06-05-PLAN.md:7-13` declares the
file. Nothing upstream can move any of these literals. This is the load-bearing half of the claim and
it holds.

**Source-before-mirror is reachable.** `PROJECT.md:5` (the source of `.claude/CLAUDE.md:7`) currently
scores 3 hits — `rewrite of`, `colbymchenry`, `ported` — and Task 1 marks it `sweep`; Task 2 fixes the
source first, then re-syncs. `PROJECT.md:242` and `:246` (the sources of `:15` and `:17`) already
score **0**. Line 93's prescribed replacement (*"a bounded later refactor, not a blocker for
shipping"*) also scores 0. So `CLAUDE_MD_FRAMING_TOTAL=0` is reachable **through** the authorised
edits, not despite them.

## Sixth-instance check — Cycle 5

**The cycle-4 replan introduced no new instance — but a sixth instance EXISTS in the current plans
and is reported as HIGH 1 below.** It lives in `06-01`, which `0e2aeba` did not touch, and it
survived cycles 1-4 undetected. The distinction matters for attribution only; the finding is current
and unresolved either way.

The remainder of the census surface is clean. Evidence, stated per file rather than in aggregate:

The phase census surface, measured today under the bounded (class-fix-D) set, with each nonzero file
mapped to the plan that owns write authority for it:

| File | Bounded census hits | Owning plan (wave) | Zero reachable? |
|---|---:|---|---|
| `docs/BENCHMARKS.md` | 47 | 06-04 (W3) — full rewrite | ✅ |
| `tools/bench/runner/main.go` | 35 | 06-01 (W1) | ✅ |
| `tools/bench/runner/main_test.go` | 5 | 06-01 (W1) | ✅ |
| `tools/bench/realcorpus/manifest.go` | 12 | 06-01 (W1) | ✅ |
| `tools/bench/realcorpus/manifest_test.go` | 5 | 06-01 (W1) | ✅ |
| `.github/workflows/bench.yml` | 26 | 06-02 (W2) | ✅ |
| `tools/bench/BASELINE.md` | 2 | 06-02 (W2) | ✅ |
| `internal/upgrade/taskfile_shape_test.go` | 1 | 06-02 (W2) | ✅ |
| `internal/bench/metrics.go` | 3 | 06-03 (W1) | ✅ |
| `internal/bench/regression.go` | 2 | 06-03 (W1) | ✅ |
| `internal/bench/rss.go` | 1 | 06-03 (W1) | ✅ |
| `internal/corpora/manifest.go` | **0** | — | already clean |
| **`corpora/manifest.json`** | **0** | — (declared KEEP, in no plan's `files_modified`) | ✅ **this is the cycle-3 HIGH 2 counterexample, and it is now 0** |
| `tools/bench/headtohead-*.json` (6) | — | 06-01 (W1) — `git rm` | removed |

Every file with a nonzero count has an owning plan with write authority. The one file that has **no**
owner — `corpora/manifest.json`, the exact counterexample that made cycle 3's aggregate claim a HIGH —
now scores **0** under the bounded set. `PHASE6_CENSUS_TOTAL = 0` is reachable by the correct path.

Further reachability checks run this cycle:

- **`CENSUS_CRITICAL_PRESENT=13/13`** — all thirteen named files exist today; 06-01 deletes only the
  six `headtohead-*.json` captures, none of which is in the critical thirteen. `rg --files` was
  confirmed to accept explicit file arguments (surface = 26 files, floor is `>= 12`).
- **`U_PINNED = 19`** (`06-02-PLAN.md:351`) — measured today: `U_TOTAL=22`, `U_LOCAL=2`,
  `U_PINNED=20`. The rewrite drops exactly one pinned action, `actions/setup-node` at
  `bench.yml:118` (required, since `setup-node` is in the same gate's retired-terms set), giving
  20 − 1 = **19**. The literal is derived, not guessed, and the companion assertion
  `U_PINNED + U_LOCAL == U_TOTAL` makes it a set-consistency check rather than a bare count.
- **`PRESENT[CommitSHA] >= 1`** — 0 in the current document, but the rewrite is directed to emit it
  twice (`06-04-PLAN.md:293`, and the provenance block), and the CamelCase spelling is grounded in
  the Go struct field at `tools/bench/realcorpus/manifest.go:57`. Reachable.
- **`DIVERGENT_VERDICTS = 1`** — reachable, but see MEDIUM 1 below for the one residual ambiguity.

## Findings — Cycle 5

### HIGH 1 — the mandated removal-proof test must contain `ts-binary`, and three `= 0` census gates forbid it

**This is the sixth instance of the phase's recurring defect class.** It reddens three gates across
two plans on a fully correct execution, and neither plan's executor has an authorised escape.

**The mandate.** `06-01-PLAN.md:34` (`must_have`) and `06-01-PLAN.md:243-246` (`<behavior>`) require,
in `tools/bench/runner/main_test.go` (the file header at `:239`), a new test
`TestParseFlags_RejectsRetiredComparisonBinaryFlag`:

> *"`parseFlags` called with the retired comparison-binary flag returns a **non-nil** error, because
> the flag is no longer registered. This is what proves the flag is gone rather than merely
> undocumented."* — required by D-07 / rule `84d1gfpywd`.

**The flag is `-ts-binary`** — named as such in `06-01-PLAN.md:113` (the deletion inventory:
"`config.tsBinary` field and its `-ts-binary` flag") and `:337`, and live today at
`tools/bench/runner/main.go:178`. Go's `flag` package identifies flags **by name**, so every faithful
implementation — `parseFlags([]string{"-ts-binary", …})`, or `fs.Lookup("ts-binary") == nil` — leaves
the literal string in `tools/bench/runner/main_test.go`. There is no substitute literal, and the plan
itself forbids the alternative: `06-04-PLAN.md:539` — *"Never reword the innocent text."*

**The three gates it reddens.** All three are `= 0` assertions whose surface includes
`tools/bench/runner/` and whose pattern set matches `ts-binary`:

| Gate | Location | Surface | Matching pattern |
|---|---|---|---|
| `RETIRED_RUNNER_SYMBOLS_TOTAL = 0` | `06-01-PLAN.md:398` | `tools/bench/runner/` | `-e 'ts-binary'` |
| `BENCH_TRACER_SURFACE_TOTAL = 0` | `06-01-PLAN.md:489` (set at `:446-458`) | `tools/bench/realcorpus/` + `tools/bench/runner/` | `-e '\bts-binary\b'` |
| `PHASE6_CENSUS_TOTAL = 0` | `06-04-PLAN.md:578` | `docs/BENCHMARKS.md tools/bench …` | `-e '\bts-binary\b'` |

**Demonstrated, not argued.** A minimal faithful implementation of the mandated test was written to
a scratch tree and both pattern sets run against it:

```go
func TestParseFlags_RejectsRetiredComparisonBinaryFlag(t *testing.T) {
	if _, err := parseFlags([]string{"-ts-binary", "/custom/codegraph"}); err == nil {
		t.Fatal("expected parseFlags to reject the retired flag")
	}
}
```

```
06-01:398 RETIRED_RUNNER_SYMBOLS_TOTAL = 1     (asserted 0)
06-04:578 / 06-01:489 bounded set     = 1     (asserted 0)
```

**Neither executor has an authorised escape.**

- The 06-01 patterns are hard-coded inside `<acceptance_criteria>` shell snippets; an executor cannot
  rewrite its own acceptance criterion. (`06-01-PLAN.md:490` anticipates *pattern narrowing* during
  the task's step 2, but that provision governs the census the executor runs by hand, not the
  literal gate text.)
- The 06-04 pattern is inside an `<automated>` block, and `06-04-PLAN.md:526-527` states this
  explicitly: *"the pattern lives inside an automated verify block the executor cannot legitimately
  edit."* `06-04-PLAN.md:534-542` closes the exclusion list — *"Adding an exclusion that is not on
  this list, or is not backed by a citation, is **forbidden**"* — and
  `tools/bench/runner/main_test.go` is **not** in 06-04's `files_modified`, so 06-04 cannot fix the
  source either.

**This is exactly cycle 3's HIGH 2, one file over.** There, `corpora/manifest.json:10` held a
KEEP-class `TS/JS` in a file no plan declared, matched by a hard-coded pattern in an automated block.
Here, `tools/bench/runner/main_test.go` holds a KEEP-class `ts-binary` — *removal evidence*, the
opposite of retired framing — matched by a hard-coded pattern in an automated block, in a file 06-04
does not declare.

**The planner caught the identical hazard one line over and fixed it.** `06-01-PLAN.md:260-265`
forbids using `"ts"` as the publishcheck wrong-subject fixture value, in terms that describe this
finding precisely:

> *"`tools/bench` is inside 06-04's Phase-6 census surface, whose bounded pattern set counts a quoted
> `"ts"` as retired subject framing; a fixture written the obvious way would make 06-04's
> `PHASE6_CENSUS_TOTAL=0` unreachable through no fault of 06-04's executor."*

The `-ts-binary` literal in the removal-proof test is the same hazard — and worse, because
`"ts"` → `"other"` had a substitute literal and a flag name does not.

**PLAN.md change needed.** Pre-authorise the narrowing in-plan at all three sites, exactly as class
fix D was pre-authorised — this is D-04's own stated remedy (*"If a pattern flags text that is
genuinely innocent, the finding is about the pattern. Bound it tighter and record the change with the
flagged text quoted"*), so no LOCKED decision is re-opened:

1. `06-01-PLAN.md:398` — exclude the single removal-proof assertion from
   `RETIRED_RUNNER_SYMBOLS_TOTAL`'s surface (e.g. `-g '!main_test.go'` paired with a companion
   assertion that `main_test.go` contains the rejection test), or bound `ts-binary` to its
   *registration* sense.
2. `06-01-PLAN.md:446-458` — same narrowing for `BENCH_TRACER_SURFACE_TOTAL`.
3. `06-04-PLAN.md:578` — add `tools/bench/runner/main_test.go`'s rejection case to the
   **pre-authorised** narrowing at `:526-527` and to the exclusion list at `:534-542`, with the
   flagged text quoted and adjudicated in `06-CENSUS.md`, citing `06-01-PLAN.md:34` / D-07 /
   rule `84d1gfpywd` as the source. Add a companion positive assertion so the exclusion cannot hide
   a real regression: the excluded file must contain **exactly one** `ts-binary` occurrence, inside
   `TestParseFlags_RejectsRetiredComparisonBinaryFlag`.

Whichever form is chosen, apply it to **all three** sites — fixing one leaves the phase red at the
other two.

### MEDIUM 1 — 06-05 Task 1 does not say which row absorbs `.claude/CLAUDE.md:83`, and the natural reading reddens `DIVERGENT_VERDICTS = 1`

**Evidence.** Task 1's enumeration (`06-05-PLAN.md:171-176`) runs a census over `.claude/CLAUDE.md`,
`.planning/PROJECT.md` and `.planning/STATE.md` with a pattern set that explicitly **includes
`reimplementation`**. Measured against the current tree, that census returns hits in
`.claude/CLAUDE.md` on lines **7, 15, 17, 83, 93**. Line 83 is the parser-runtime
`reimplementation`.

`06-05-PLAN.md:30` requires *"Every occurrence found in the three files carries a recorded verdict…
none was dropped silently"*, and `:202-203` specifies *"one table row per occurrence: file, line,
quoted text, verdict, reason."* So `:83` takes a row. Its verdict cannot be `sweep` (the line is
outside the task's sanctioned edit scope, and the plan's own prohibitions forbid re-authoring that
block). It is not a past-tense record, so `keep-historical` is semantically wrong. The natural
verdict is **`keep-divergent`** — and that drives `DIV` to 2, failing
`test "$DIV" = "1"` at `06-05-PLAN.md:211` **on a correct execution**.

**Why this is MEDIUM and not HIGH.** The correct path *is* available and *is* named:
`06-05-PLAN.md:216` states *"exactly one row carries the `keep-divergent` verdict — the
`.claude/CLAUDE.md` stack-block row… two fails (the class is being used as a catch-all)"*, and the
pre-identified table at `:197` already demonstrates a **range** row (`lines 21-141`) absorbing
in-block occurrences. An executor who reads `:216` will fold `:83` into that row. So the gate is
reachable — this is an ambiguity that *could* redden, not an assertion that *must*.

**Why it is still worth closing.** Line 93 is also inside 21-141 and gets its **own** row
(`sweep IN PLACE`), so the block row demonstrably does not absorb every in-block occurrence — the
executor must judge. And `0e2aeba` just made `reimplementation` highly salient in three places in
06-05 **and** established a precedent of "a notable single occurrence gets its own appended row"
(the new line-17 `keep-historical` row). Both changes raise, not lower, the chance an executor gives
`:83` a dedicated row. `:83` is never mentioned in Task 1's context anywhere in the plan.

**Also imprecise:** the new text at `06-05-PLAN.md:262-268` says Task 1 *"saw the term in
`PROJECT.md` and never in the file it lands in"*. Task 1 **does** see `reimplementation` in
`.claude/CLAUDE.md` — at `:83`. The claim is true only of the line-17 occurrence.

**PLAN.md change needed.** In `06-05-PLAN.md`, add one clause to Task 1 — either in the verdict table
at `:197` or in the acceptance criterion at `:216`:

> `.claude/CLAUDE.md:83`'s `reimplementation` occurrence lies inside the `GSD:stack-*` block and is
> accounted for by that block's single `keep-divergent` range row. It does **not** take its own row —
> a second `keep-divergent` row would drive `DIVERGENT_VERDICTS` to 2 and redden a correct execution.
> (Contrast line 93, which does take its own row because it is edited in place.)

Optionally correct `:262-268` to say "never at the line it lands in".

### LOW 1 — 06-04's required-heading assertion is aggregate, and passes a malformed document

**Evidence.** `06-04-PLAN.md:364` states *"each named heading present exactly once"* but implements it
as a single summed count:

```sh
H=$(rg -o -F -e '## 1. Methodology' -e '## 2. Absolute numbers' \
     -e '## 3. Regression gate (PERF-02, INDX-06)' docs/BENCHMARKS.md | wc -l)
TOP=$(rg '^## ' docs/BENCHMARKS.md | wc -l)
test "$H" = "3" && test "$TOP" -ge 3
```

Reproduced counterexample (a three-line file: `## 1. Methodology` twice, `## 2. Absolute numbers`
once, `## 3. Regression gate…` absent):

```
BENCHMARKS_REQUIRED_HEADINGS=3 BENCHMARKS_TOP_SECTIONS=3
GATE_EXIT=0
```

A duplicated Methodology heading plus a **missing regression-gate section** passes. `BENCH-01`
explicitly requires the rewritten document to carry the regression-gate description
(`06-04-PLAN.md:340`).

**Why LOW.** This fails in the *permissive* direction — malformed work passes — which is the opposite
of the recurring defect class, and it is independently backstopped: `PRESENT[CheckRegression] >= 1`
and `PRESENT[baseline.json] >= 1` in the same gate both require regression-gate content. The
heading-identity claim is simply weaker than its own prose says. Note the irony: this is precisely
the *"per file, never aggregate"* lesson, applied to headings.

**PLAN.md change needed.** In `06-04-PLAN.md:364`, replace the summed `H` with three independent
counts and require each to equal `1`:

```sh
for h in '## 1. Methodology' '## 2. Absolute numbers' '## 3. Regression gate (PERF-02, INDX-06)'; do
  C=$(rg -o -F "$h" docs/BENCHMARKS.md | wc -l | tr -d ' ')
  echo "HEADING[$h]=$C"; test "$C" = "1" || exit 1
done
```

Retain `BENCHMARKS_TOP_SECTIONS >= 3` as the printed non-empty companion.

### LOW 2 — `corpora/manifest.json` is in 06-03's `files_modified` but not in Task 1's `<files>`

**Evidence.** `06-03-PLAN.md:14` lists `corpora/manifest.json` in `files_modified`, and
`06-03-PLAN.md:237-243` explicitly authorises editing its top-level `note` at line 2 (*"If that value
needs a change, make it and say so in the SUMMARY: it is committed data"*). But Task 1's `<files>` at
`06-03-PLAN.md:174` omits it.

This is the residue of the cycle-3 HIGH 2 fix, which reconciled the **frontmatter** but not the
**task** element — and it is the same class of omission that `0e2aeba` closed for 06-05 Task 2
(`06-MEMORY-SWEEP.md` added to `<files>`). Practical risk is low: the plan itself measures that
line 2 is already clean for every pattern in the census, so the expected outcome is no edit at all.

**PLAN.md change needed.** Append `, corpora/manifest.json` to `06-03-PLAN.md:174`'s `<files>`.

### Noted, not actionable

- **06-03 Task 3 transiently writes `internal/bench/regression.go`.** `06-03-PLAN.md:380` declares
  only `06-MUTATION-LOG.md`, yet the action at `:389` applies two mutation families to
  `internal/bench/regression.go` under a `trap … EXIT` restore. Net change is zero, the file is in
  the plan's `files_modified` via Task 1, and the precondition at `:379` already names it. No
  lost-write hazard; no change required.
- **Wave-1 content coupling between 06-01 and 06-03.** `06-03-PLAN.md:32` states a truth in the past
  tense of 06-01's completion, while both sit in wave 1 with `depends_on: []`. Not a write conflict
  and not a blocked gate: the acceptance criterion is a SUMMARY statement (`:274`), no automated gate
  reads `tools/bench/realcorpus`, `read_first` at `:182` already handles both orderings, and D-10
  keeps the package name and path stable. Recorded so a later reviewer does not rediscover it as new.

## Cross-plan structure — Cycle 5

Audited independently this cycle; **all clean**.

- **File ownership — zero same-wave write conflicts.** Waves: **W1** = 06-01, 06-03 · **W2** = 06-02
  · **W3** = 06-04 · **W4** = 06-05 · **W5** = 06-06. The only wave-1 pair (06-01, 06-03) has
  strictly disjoint file sets, so `ROADMAP.md:280`'s "parallel — zero file overlap" claim holds.
  The two shared planning artifacts are strictly sequential across waves:
  `06-MEMORY-SWEEP.md` (06-05 W4 creates → 06-06 W5 extends) and `06-LIVE-VERIFICATIONS.md`
  (06-04 W3 creates → 06-05 W4 appends → 06-06 W5 appends).
- **Wave / dependency ordering — correct.** Every consumer runs strictly later than its producer:
  `06-CENSUS.md` (06-04 W3 → read by 06-05 W4), `06-PUBLISH-RESULTS.json` (intra-plan, gated by a
  `<precondition>` at `06-04-PLAN.md:250`), `.github/workflows/bench.yml` (06-02 W2 → gated by
  06-04 W3 at `:167`), `tools/bench/publishcheck` (built W1, consumed W3).
  `06-06` correctly carries no `PREPLAN-SHA.txt` — it has no git tamper guard, and
  `06-01-PLAN.md:95` registers the artifact as "one per plan, NN = 01..05" deliberately.
- **Requirement coverage — complete.** Phase 6 scope per `REQUIREMENTS.md:122` / `ROADMAP.md:266` is
  **BENCH-01, BENCH-02, BENCH-03, MEM-01, MEM-02**. BENCH-01 → 06-04; BENCH-02 → 06-01/06-03/06-04;
  BENCH-03 → 06-02/06-04; MEM-01 → 06-06; MEM-02 → 06-05 (file half) + 06-06 (store half). No
  uncovered requirement; no plan claims a requirement outside Phase 6 scope. All 16 LOCKED decisions
  D-01..D-16 are referenced by at least one plan.
- **Blocking checkpoints — exactly two, both intact.** `06-04-PLAN.md:371` (resolve BENCH-03 status,
  guarding Task 4 via the precondition at `:423`) and `06-06-PLAN.md:247` (confirm supersede
  classification, guarding Task 3 via the precondition at `:287`). The `a643713 → 0e2aeba` diff
  contains no line matching `checkpoint`, `gate="blocking"`, `wave:`, `depends_on` or `autonomous`.
- **Autonomous flags — unchanged.** `06-01 true · 06-02 true · 06-03 true · 06-04 false ·
  06-05 true · 06-06 false`, byte-identical at `a643713` and `HEAD`. Both `false` values sit on
  exactly the two plans carrying blocking checkpoints — the correct pairing.

## Risk Assessment — Cycle 5

**MEDIUM**, driven entirely by HIGH 1. Everything else is in good shape: the benchmark de-coupling,
workflow rewrite, mutation-proof, one-way memory safeguards and cross-plan structure are sound and
grounded in source; all four cycle-4 fixes reproduce exactly; and the census now reaches zero per
file including `corpora/manifest.json`, the counterexample that made cycle 3's aggregate claim a
HIGH.

HIGH 1 is bounded and mechanical — a pattern narrowing at three named sites, using a remedy the plans
already pre-authorise elsewhere — but it is genuinely blocking: on a correct execution the phase goes
red at `06-01`'s own acceptance criteria and again at `06-04`'s automated census, with no authorised
escape at either. Left unfixed it produces precisely the outcome this convergence loop exists to
prevent: an executor who is doing the right thing sees red and is trained to discount it.

Once HIGH 1 is closed at all three sites, residual risk is **LOW**. The three non-HIGH findings are
small, local and independently backstopped; none blocks execution, and two of the three fail in the
permissive rather than the correct-work-reddens direction.

## Divergent views — Cycle 5

Codex assessed the phase at **0 HIGH / 0 MEDIUM / 1 LOW** and explicitly concluded that no sixth
instance was introduced. The orchestrator's supplementary gate-reachability audit found HIGH 1, which
Codex missed. The disagreement is not interpretive — HIGH 1 is settled by measurement (the scratch
probe above), and Codex's own reasoning would reach the same conclusion given the same probe. Recorded
here rather than smoothed over, because it is the same failure mode cycle 3 documented: a
reviewer's "all clear" that was an aggregate impression rather than a per-file check. Codex's LOW
(the aggregate heading count) was independently reproduced by the orchestrator and is carried as
LOW 1 above.

## Codex Review — Cycle 5 (verbatim, source-grounded)

## 1. Summary

At HEAD `7eff0fa`, the six plans are coherent and executable against the current source. All four cycle-4 claims reproduce, the fixes in `0e2aeba` do not introduce a sixth correct-path-fails-red defect, and no HIGH or MEDIUM concern remains. I found one actionable LOW in an older heading-identity assertion: its aggregate count can pass with one required heading duplicated and another absent. Overall: **0 HIGH / 0 MEDIUM / 1 LOW**.

## 2. Verification of the four cycle-4 claims

### Claim 1 — `reimplementation` adjudication

Command:

```sh
rg -n -i -e '\breimplementation\b' -e '\bport\b' .claude/CLAUDE.md
```

Result: exactly **2 lines**:

- [`.claude/CLAUDE.md:17`](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:17) — sole bare `port`, removed by the authorized re-sync.
- [`.claude/CLAUDE.md:83`](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:83) — parser-runtime `reimplementation`, outside the permitted stack-block edit at line 93.

Bookkeeping is sound:

- Task 1’s row gate runs before Task 2 and uses floors for `ROWS`, `SWEEP`, and `KEEP`; only `DIV=1` is exact, and the new row is `keep-historical`, not `keep-divergent` ([06-05-PLAN.md:211](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:211)).
- Task 2 explicitly retains the old line-17 `sweep` row and adds a separate post-edit `keep-historical` row ([06-05-PLAN.md:362](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:362)).
- The changed-line mapping remains against `sweep` rows ([06-05-PLAN.md:374](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:374)).
- Plan 06 bounds its counts to `## Engram enumeration`, so both D-12 rows are structurally excluded ([06-06-PLAN.md:222](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:222), [06-06-PLAN.md:229](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:229)).

Downstream count collisions: **0**.

Wrong-fix checks:

```sh
sed -n '357p' 06-05-PLAN.md | rg -o -F '\breimplementation\b' | wc -l
git diff --quiet 0e2aeba^ HEAD -- .planning/PROJECT.md
```

Results: **0** gate occurrences; project diff exit **0**. The Licensing text remains intact at [PROJECT.md:246](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:246), as D-13 requires.

**VERIFIED.**

### Claim 2 — bounded `TS` gate

I ran the exact bounded pattern set from [06-04-PLAN.md:355](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:355) over the specified ranges:

```sh
{ sed -n '7,48p' docs/BENCHMARKS.md
  sed -n '213,300p' docs/BENCHMARKS.md
} | rg -U -o -i <bounded-pattern-set> | wc -l
```

Result: **7 hits on 6 lines**:

- `39`
- `42`
- `46`
- `215`
- `237` twice
- `240`

For the whole file:

```sh
rg -U -o -i <bounded-pattern-set> docs/BENCHMARKS.md | wc -l
```

Result: **52**.

The exact pre-fix bare command was:

```sh
rg -U -o \
  -e '\bTS\b' \
  -e '\bhead-to-head\b' \
  -e '\bheadtohead\b' \
  -e '\bcolbymchenry\b' \
  -e '\bcodegraph-ts\b' \
  -e '\bcomparator\b' \
  -e '\b1\.3\.1\b' \
  -e '[0-9]+(\.[0-9]+)?[x×] (faster|slower|more|less)' \
  docs/BENCHMARKS.md | wc -l
```

Result: **48**.

Probe results:

- Two-line `TS/JS` fixture: bare **2**, bounded **0**.
- Five-line retired-sense fixture: bounded **6 hits on 5 lines**.

Searching every plan for the exact executable literal `'\bTS\b'` finds only [06-04-PLAN.md:578](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:578), the deliberate `BARE_TS_RESIDUAL_LINES >= 1` completeness floor. Other occurrences are explanatory history, not gates.

**VERIFIED.**

### Claim 3 — §2.x restatement

The same bounded range command reports the three §2.x lines exactly as claimed:

- [docs/BENCHMARKS.md:215](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:215)
- [docs/BENCHMARKS.md:237](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:237), two hits
- [docs/BENCHMARKS.md:240](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:240)

All three point to the table or live head-to-head run being removed. The revised action explicitly says to drop those clauses while preserving the number, rationale, and re-blessing procedure ([06-04-PLAN.md:324](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:324)-[338](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:338)).

No criterion was weakened: the replacement gate detects **52** current matches versus the former gate’s **48**, while required-content and emitted-row assertions remain intact ([06-04-PLAN.md:355](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:355)-[363](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:363)).

**VERIFIED.**

### Claim 4 — expected literals and sole declarer

Commands and results:

```sh
rg -o '<!-- GSD:[^>]*-->' .claude/CLAUDE.md | wc -l
# 14

rg '^#{1,6} ' .claude/CLAUDE.md | wc -l
# 25

rg -o -F \
  -e 'The Parser Decision' \
  -e 'cockroachdb/pebble' \
  -e 'tree-sitter/go-tree-sitter' \
  .claude/CLAUDE.md | wc -l
# 16
```

Per-token anchor counts were **2**, **6**, and **8**, respectively.

Frontmatter declaration search, per plan:

| Plan | `.claude/CLAUDE.md` declarations |
|---|---:|
| 06-01 | 0 |
| 06-02 | 0 |
| 06-03 | 0 |
| 06-04 | 0 |
| 06-05 | 1 |
| 06-06 | 0 |

Thus only [06-05-PLAN.md:7](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:7)-[13](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:13) declares the file, and no later plan can invalidate the literals.

**VERIFIED.**

## 3. Strengths

- The fresh workflow is guarded through a parsed `jobs.publish` subtree, including exact runner/env values, non-empty run bodies, publish-not-gate prohibitions, job-summary use, and upload settings ([06-02-PLAN.md:168](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:168)-[206](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:206)). These correspond to real load-bearing source properties at [bench.yml:98](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:98), [bench.yml:101](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:101), [bench.yml:151](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:151), and [bench.yml:158](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:158).

- BENCH-02 drives the committed baseline through both numeric bands while copying the category-frame fields, correctly respecting the guard order in [regression.go:49](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:49)-[128](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:128). The test design and mutation non-vacuity proof are explicit at [06-03-PLAN.md:289](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PLAN.md:289)-[336](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PLAN.md:336).

- Published rows are generated mechanically from the measurement JSON, covering every metric cell rather than spot-checking throughput ([06-04-PLAN.md:304](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:304)-[315](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:315)).

- The memory operation is correctly split into read-only enumeration, a blocking classification checkpoint, and approved one-way supersedes, with an explicit halt if true supersede semantics are unavailable ([06-06-PLAN.md:247](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:247)-[307](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:307)). This respects D-13 and D-15 rather than quietly reopening them.

## 4. Concerns

No HIGH concerns. No MEDIUM concerns.

- **LOW — The required-heading assertion is still aggregate.** [06-04-PLAN.md:364](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:364) says each of the three headings is present exactly once, but sums all matches into `H=3`. I ran this counterexample:

```text
## 1. Methodology
## 1. Methodology
## 2. Absolute numbers
```

The full stated gate returned:

```text
BENCHMARKS_REQUIRED_HEADINGS=3 BENCHMARKS_TOP_SECTIONS=3
FULL_GATE_EXIT=0
```

Thus a duplicated Methodology heading plus a missing Regression gate heading passes. This does not threaten the phase goal because the substantive gate content is independently asserted, but the heading-identity claim is weaker than stated.

Exact PLAN change needed: in `06-04-PLAN.md:364`, replace aggregate `H=3` with three independent counts—one per required heading—and require each to equal `1`. Retain `TOP >= 3` as the non-empty companion.

## 5. Sixth-instance check

**No sixth instance was introduced.**

Evidence:

- The current `.claude/CLAUDE.md` framing gate has **5 hits on 3 lines**—7, 15, and 93—and Task 2 explicitly edits those lines. The authorized source text for lines 15 and 17 and the prescribed line-93 replacement each score **0** against that gate.
- `reimplementation` is deliberately excluded because [CLAUDE.md:83](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:83) must survive and the re-sync adds another KEEP-class use; adding it would recreate the defect.
- The rewritten BENCHMARKS gate’s **7 hits on 6 carried-forward lines** are all explicitly handled by the action; none is a legitimate required survivor.
- The only executable bare `\bTS\b` is a positive completeness floor, not a zero gate.
- The LOW heading issue above is the opposite failure direction—a malformed document can pass—not an assertion that fails correct work.

## 6. Risk Assessment

**LOW.** The core benchmark, workflow, mutation, census, and one-way memory safeguards are sound, and all cycle-4 fixes reproduce against source. The sole actionable issue is a small heading-cardinality assertion that does not undermine BENCH-01’s substantive verification.


## Verification coverage — Cycle 5

Every claim in this cycle was checked against the working tree at `7eff0fa`. Commands were run twice
— once by Codex with full repo access, once by the orchestrator — and agreed on every figure except
where noted under *Divergent views*.

**Files read in full or in the relevant region:** all six `06-0N-PLAN.md`; `06-CONTEXT.md`
(D-01..D-16); `06-REVIEWS.md` cycles 1-4; `.planning/PROJECT.md`; `.planning/STATE.md`;
`.planning/REQUIREMENTS.md`; `.planning/ROADMAP.md`; `.claude/CLAUDE.md`; `docs/BENCHMARKS.md`;
`.github/workflows/bench.yml`; `corpora/manifest.json`; `tools/bench/realcorpus/manifest.go`;
`tools/bench/runner/main.go`; `tools/bench/runner/main_test.go`; `internal/bench/{metrics,regression,rss}.go`;
`internal/corpora/manifest.go`.

**Measurements executed (not inferred):**

- `rg -n -i -e '\breimplementation\b' -e '\bport\b' .claude/CLAUDE.md` → 2 lines (`:17`, `:83`)
- bounded census over `docs/BENCHMARKS.md` → 52; old bare set → 48; carry-forward ranges → 7 hits / 6 lines
- `TS/JS` probe → bare 2, bounded 0
- `GSD_MARKERS` → 14; `CLAUDE_MD_HEADINGS` → 25; `CLAUDE_MD_STACK_BLOCK_ANCHORS` → 16;
  `CLAUDE_MD_FRAMING_TOTAL` → 5 hits on lines 7/15/93
- re-sync source text (`PROJECT.md:242`, `:246`) scored against the framing gate → 0
- per-file bounded census over the whole Phase-6 surface (13 files, table above), including
  `corpora/manifest.json` → 0
- `CENSUS_CRITICAL_PRESENT` → 13/13; census surface → 26 files
- `U_TOTAL` / `U_LOCAL` / `U_PINNED` over `bench.yml` → 22 / 2 / 20
- heading-gate counterexample reproduced → `H=3`, `GATE_EXIT=0` with a required heading absent
- **HIGH 1 probe**: minimal faithful `TestParseFlags_RejectsRetiredComparisonBinaryFlag` written to a
  scratch tree; `06-01:398` set → **1**, `06-04:578` / `06-01:489` bounded set → **1**, both asserted `= 0`
- `git show 0e2aeba --stat`; `git diff a643713 0e2aeba` (60 insertions / 7 deletions, two files;
  no `checkpoint`, `gate="blocking"`, `wave:`, `depends_on` or `autonomous` line touched)

**Coverage claims in this cycle are stated PER FILE.** The one place a prior cycle made an aggregate
claim, it concealed a counterexample that was a HIGH (cycle 3, `corpora/manifest.json`). Two findings
this cycle — HIGH 1 and LOW 1 — are themselves aggregate-vs-per-item defects, which is why the
distinction is repeated here rather than assumed understood.

**Not verified (out of scope for a plan review):** no plan was executed; no CI run was triggered;
`06-MEMORY-SWEEP.md`, `06-CENSUS.md`, `06-LIVE-VERIFICATIONS.md` and `06-PUBLISH-RESULTS.json` do not
exist yet, so gates over them were assessed for reachability by construction rather than measured.

---


# Cycle 4 — audit history (superseded by Cycle 5 above)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer against the working tree at `0b996e9`.

**Why a fourth cycle exists:** the normal three-cycle escalation gate was overridden with explicit
user authorisation because the finding trend was strongly converging (24 → 12 → 6). Severity is
calibrated accordingly: HIGH is reserved for a finding that would prevent the phase goal, create
unsafe execution, or invalidate verification.

## Consensus Summary — Cycle 4

**All five claims the cycle-3 replan makes were independently re-measured against the working tree
and all five hold.** Nothing was accepted on assertion; every number below was re-derived by running
the plan's own command, or the equivalent, at `0b996e9`.

| Claim | Independent re-measurement | Verdict |
|---|---|---|
| 06-01 test surface: before-set is **54**, `sort -u` also 54, `cpudiag` `[no test files]`, three removals present, four additions, `54 − 3 + 4 = 55` | `go test -list '.*' ./tools/bench/... \| rg '^Test' \| sort \| wc -l` → **54**; `sort -u` → **54**; `go test -list ./tools/bench/cpudiag/...` → `[no test files]`; removals at sorted lines **32, 52, 53**; `go list` → four packages | **HOLDS** — the planner's correction of the cycle-3 review's own 55 was right |
| Census zero reachable: bounded `TS`, 06-03 scope 7 → 7 with `corpora/manifest.json` 1 → 0; 06-04 surface 168 → **170** with that file 1 → 0 | 06-03 bounded set → **7**, all six lines inside `internal/bench/*.go` that Task 1 sweeps, `corpora/manifest.json` → **0**. 06-04 bounded set → **170** total, `corpora/manifest.json` → **0**; every per-file number in the plan's table reproduced exactly (see coverage block) | **HOLDS** |
| 06-03 class fix E: comment-only gate restated as code-portion equality; old gate red (`2`), new green, planted `Subject` → `Subj` still red | Reproduced on a scratch copy of `internal/bench/metrics.go`: old gate → **2** (red); new gate → `CODE_PORTION_IDENTICAL=true` (green); planted `Subject` → `Subj` → **RED** with `MINUS` / `PLUS` differing on the code prefix | **HOLDS** |
| Cross-plan fixture trap: publishcheck wrong-subject literal pinned to `"other"`, not `"ts"` | `06-01-PLAN.md:261` states the prohibition and the reason; `"other"` is matched by no pattern in 06-04's bounded set; `'"ts"'` is | **HOLDS** — and no other fixture literal in any plan collides (see below) |
| Four actionable findings incorporated: 06-06 approved-set extractor bounded with both raw counts asserted `> 0`; 06-03 band counters scoped to `--- FAIL`; 06-05 ceiling basis stated with record-don't-trim; 06-06 literals stated in the action | All four verified in place at `06-06-PLAN.md:349`, `06-03-PLAN.md:484`, `06-05-PLAN.md:331`, `06-06-PLAN.md:218/204` | **HOLDS** |

**No fifth instance of the "positive assertion the tested path cannot produce" defect was
introduced this round.** Every new equality carries a measured basis, and the two strictest —
`TESTS_AFTER = 55` and the zero-headroom `THRU/RSSN >= 2` — were traced to the output the correct
path actually emits. `TESTS_AFTER` additionally sits in a manual criterion carrying an explicit
escape ("If the executor's `TESTS_AFTER` is not 55, that is a real finding to report — not a number
to adjust"), which is the right shape for an exact-count assertion.

**No HIGH concerns remain.** Two non-HIGH findings are new and not yet in any PLAN.md.

### Agreed Strengths

- **The test-surface delta is now arithmetically closed and independently reproducible.** Only three
  test functions in `tools/bench/runner/main_test.go` are tied to the comparison path
  (`TestParseFlags_OverridesApply:481`, `TestResolveTSBinary_FindsOnPath:496`,
  `TestResolveTSBinary_EmptyWhenNotFound:512`); no fourth test would be forced out by Task 1's
  deletions, so the removed set of exactly three is correct against source, not just against the
  plan's own narrative.
- **The bounded census is a net strengthening, not a relaxation** — verified rather than accepted.
  The bounded form returns **170** where the bare form returned 168, because it additionally catches
  camelCase `tsBinary` (`tools/bench/runner/main.go`) and `internal/bench/metrics.go:9`'s quoted
  `"ts"`. Every one of the thirteen per-file counts in `06-04-PLAN.md:478-493` reproduced to the
  digit.
- **Every file with a non-zero census count is a declared sweep target of exactly one plan.** Checked
  against each plan's `files_modified`: `docs/BENCHMARKS.md` → 06-04; `runner/main{,_test}.go`,
  `realcorpus/manifest{,_test}.go`, the six `headtohead-*.json` → 06-01; `bench.yml`, `BASELINE.md`,
  `internal/upgrade/taskfile_shape_test.go` → 06-02; `internal/bench/{metrics,regression,rss,regression_test}.go`
  → 06-03. `corpora/manifest.json` is at 0 and is now declared by 06-03 scoped to line 2 only
  (`06-03-PLAN.md:52`), reconciling the frontmatter with the paragraph that contemplates editing it.
- **The completeness control on the bounding is the right shape.** `BARE_TS_RESIDUAL_LINES >= 1`
  with every residual quoted and adjudicated in `06-CENSUS.md` makes an under-covering enumeration
  visible instead of shipping a false zero, and the `>= 1` floor (rather than `= 1`) is deliberately
  slack so that legitimate new indexed-language text cannot redden correct work.
- **The code-portion-equality restatement of the comment-only gate is sound in both directions.**
  Pure comment lines strip to empty and drop out on both sides, so a re-authored block may change
  its line count; a code line carrying a trailing comment contributes its identical prefix to both
  lists; a real semantic change breaks the equality. The plan states the known limit (a `//` inside a
  string literal) rather than hiding it, and names the four independent gates that cover it.
- **The zero-headroom band counters are tied to deterministic Go output, not to timing.**
  `--- FAIL` scoping is what makes them discriminate: on a green `-v` paste the same counters read
  `0/0`. The exact-2 expectation follows from the two oracles (the pre-existing table subtest and
  `TestCheckRegressionAgainstCommittedBaseline`) each emitting one band-named `--- FAIL` line, and
  the action explicitly requires both to be pasted. Inflation from quoted prose in the log moves the
  count in the safe direction, since the assertion is `>= 2`.
- **The cross-plan fixture trap was caught by the planner before any reviewer raised it**, and the
  sweep generalises: no other fixture or test literal in any of the six plans lands a census-matched
  token inside a census-surface file. 06-03's new `internal/bench/baseline_gate_test.go` sits inside
  the `internal/bench` surface and its pinned literals (`committed baseline throughput 11% slower:
  exceeds band fails`, `CheckRegression(committed baseline) = nil, want error`) match no pattern in
  either bounded set. 06-02's new `internal/upgrade/bench_workflow_shape_test.go` is outside the
  census surface by construction — the surface names `internal/upgrade/taskfile_shape_test.go`
  specifically, not the directory.
- **06-06's extractor pair is now symmetric.** Both halves use
  `awk '/^## <heading>/{f=1;next} /^## /{f=0} f'`, and `^## ` genuinely does not match a `### `
  subheading (third character is `#`, not a space), so per-scope subsections stay inside the window.
  Both raw line counts are printed and asserted `> 0`, closing the empty-versus-empty read.
- **06-02's counts re-verified independently:** `.github/workflows/bench.yml` carries **22** `uses:`
  lines, **20** pinned and **2** local (`./.github/actions/install-task`); deleting the one
  `actions/setup-node` line gives exactly the asserted `U_PINNED=19` / `U_LOCAL=2`. `Taskfile.yml`
  has exactly one `bench*` target (`bench:regression:`), matching the asserted
  `TASKFILE_BENCH_TARGETS`.
- **The 06-04 census surface survives execution with margin.** `rg --files` over the declared surface
  returns **26** files today; after 06-01 deletes six captures and 06-01/06-03 add three files it is
  **23**, against a floor of `>= 12`, and all thirteen `CENSUS_CRITICAL_PRESENT` files survive.

### Agreed Concerns

Both reviewers agree there is no HIGH-severity concern in the current plan text. The two findings
below are MEDIUM and LOW, both new, both absent from every PLAN.md.

### Divergent Views

- Codex framed the 06-05 Licensing re-sync as "can **preserve** reimplementation framing". The
  orchestrating reviewer's re-reading of `.claude/CLAUDE.md:17` (`Original is MIT — port with
  attribution`) against `.planning/PROJECT.md:246` shows the re-sync is a strict **improvement** —
  present-tense `port` is replaced by past-tense origin wording. The finding survives, but as
  "an unadjudicated token enters the file and the gate is silent about it", not as a regression.
  Severity therefore MEDIUM, not HIGH.
- Codex suggested rewriting `.planning/PROJECT.md`'s Licensing constraint to be origin-neutral and
  adding `-e '\breimplementation\b'` to `CLAUDE_MD_FRAMING_TOTAL`. The orchestrating reviewer
  **rejects the second half taken alone**: with the current authoritative PROJECT.md wording, adding
  that pattern makes the gate unreachable on correct work — precisely the fifth instance of the
  defect class this cycle exists to prevent. And the first half pushes against **D-13** (LOCKED),
  which requires point-in-time statements to survive untouched. See the finding below for the form
  that does not re-open a locked decision.

---

## Non-HIGH findings — Cycle 4

### MEDIUM 1 (CYCLE 4 — RESOLVED in `0e2aeba`, re-verified by measurement in cycle 5) — 06-05 Task 2's authorised Licensing re-sync introduces `reimplementation` into `.claude/CLAUDE.md`, and neither Task 1's enumeration nor the final gate sees it

**Evidence.** `06-05-PLAN.md:246-249` authorises the re-sync explicitly: "Two CLAUDE.md occurrences
(the Compatibility and Licensing constraints) are already stale mirrors of PROJECT.md text corrected
at v0.11.0 scoping — those are re-syncs, not re-authorings, and the current PROJECT.md wording is
authoritative." The authoritative text at `.planning/PROJECT.md:246` reads "Attribution for the
project this one began as a **reimplementation** of lives in `NOTICE`…". The acceptance gate at
`06-05-PLAN.md:331` counts only `\bparity\b`, `\bdrop-in\b`, `\bported\b`, `\brewrite of\b`,
`\bTS CodeGraph\b`, `\bcolbymchenry\b` — measured against the current tree it returns **5**
occurrences on lines **7, 15, 93**, exactly as the plan states, and it will still return `0` after
the re-sync. Task 1's enumeration pattern set (`06-05-PLAN.md:160-161`) *does* include
`reimplementation`, but Task 1 runs **before** Task 2's edit, so the token that Task 2 introduces
into `.claude/CLAUDE.md` is enumerated in `PROJECT.md` and never in the file it lands in.

**Why this is MEDIUM and not HIGH.** MEM-02's success criterion is scoped to the **present tense**
("recalls no memory describing codegraph-go as a port or parity project in the present tense"), and
"began as a reimplementation of" is a point-in-time origin statement — D-13's protected KEEP class.
The change is a net improvement over the present-tense `port with attribution` it replaces, and the
`<human-check>` on a fresh session covers the outcome. Nothing is broken; a decision is being made
silently that ought to be recorded.

**PLAN.md change needed (06-05).** Add one acceptance-criterion sub-bullet under Task 2 stating:
(a) the Licensing re-sync introduces `reimplementation` into `.claude/CLAUDE.md`, (b) that
occurrence is adjudicated as the D-13 KEEP class — past-tense origin, not present-tense framing —
and must be recorded as such in `06-MEMORY-SWEEP.md`'s agent-facing-files table alongside the
enumerated hits, and (c) `\breimplementation\b` and bare `\bport\b` are deliberately **outside**
`CLAUDE_MD_FRAMING_TOTAL` because the authoritative PROJECT.md wording carries the first, so
including it would make the gate unreachable on correct work. **Do not** add the pattern to the gate
and **do not** rewrite PROJECT.md's Licensing bullet to be origin-neutral — the first creates the
fifth instance of this phase's own defect class, and the second requires re-opening **D-13**
(LOCKED), which protects point-in-time statements.

### LOW 1 (CYCLE 4 — RESOLVED in `0e2aeba`, re-verified by measurement in cycle 5) — 06-04's stated rationale for the `>= 1` residual floor contradicts its own `docs/BENCHMARKS.md` gate

**Evidence.** `06-04-PLAN.md:564` justifies the `BARE_TS_RESIDUAL_LINES >= 1` floor (rather than
`= 1`) on the ground that "a rewritten `docs/BENCHMARKS.md` may legitimately name `TS/JS` as an
indexed-language pair and an exact count would then redden correct work". But the document-scoped
gate at `06-04-PLAN.md:334` runs `rg -U -o -e '\bTS\b' … docs/BENCHMARKS.md` and requires
`BENCHMARKS_RETIRED_TERMS_TOTAL=0` — that alternation is **bare**, it is the one place in the phase
where the bare form survived class fix D, and it forbids exactly the `TS/JS` the other criterion
says is legitimate.

**Impact.** Not an execution blocker: 06-04 authors `docs/BENCHMARKS.md` itself, so the executor
simply will not write `TS/JS` there, and the `>= 1` floor remains satisfied by
`corpora/manifest.json:10` (measured: the sole bare-`TS` line in that file, and the file is swept by
no plan at line 10). The cost is a contradictory instruction for a future reader and a residual
bare-`\bTS\b` gate that the phase otherwise eliminated.

**PLAN.md change needed (06-04).** Either (a) replace the bare `-e '\bTS\b'` in the
`BENCHMARKS_RETIRED_TERMS_TOTAL` alternation with the same bounded `TS` senses class fix D applies
everywhere else in the plan, and keep the line-564 rationale as written; or (b) keep the bare gate
and restate line 564's rationale as "the residual population may grow through adjudicated
indexed-language uses **outside `docs/BENCHMARKS.md`**, which this plan's own document gate holds at
zero". Option (a) is preferred — it removes the last bare instance of the pattern the phase spent a
HIGH closing.

---

## Codex Review — Cycle 4 (verbatim, source-grounded)

<!-- Codex ran with full repo access against the same prompt. Its `go` invocations failed
     in its sandbox (build-cache creation), which it states; it reproduced the 54-test figure
     from `func Test*` declarations and the census figures with the exact patterns. The
     orchestrating reviewer re-ran every measurement with a working toolchain — see the
     Verification coverage table. -->

#### Summary

The six plans are strongly converged and executable. Independent checks support the five cycle-4 claims: the test surface is 54 unique top-level tests; the bounded census produces the claimed per-file counts and total of 170 while sparing `corpora/manifest.json:10`; the comment-only gate detects ordinary semantic field changes; the mutation log counters are now tied to actual `--- FAIL` lines; and both memory-set extractors are section-bounded.

I found one current issue: Plan 06-05 can preserve “reimplementation” framing in the live agent instructions because it treats PROJECT.md’s Licensing constraint as authoritative while its final CLAUDE.md gate does not search for `reimplementation`. This is a MEDIUM concern, not a blocker to starting the phase.

#### Strengths

##### 06-01 — Benchmark tracer

- The test-surface arithmetic is correct. The current tree contains 54 top-level `Test*` functions and 54 unique names. The three named removals exist at [main_test.go:481](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:481), [main_test.go:496](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:496), and [main_test.go:512](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:512). The plan adds exactly four top-level symbols, yielding `54 - 3 + 4 = 55`, as asserted at [06-01-PLAN.md:405](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:405).

- `cpudiag` genuinely contributes no tests; the four packages are `cpudiag`, `gencorpus`, `realcorpus`, and `runner`, while the 54 existing names come from the latter three.

- The wrong-subject fixture is correctly pinned to `"other"` at [06-01-PLAN.md:261](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:261). This prevents the later `'"ts"'` census pattern from rejecting correct new test code.

- The tracer removes the complete two-subject architecture, not merely the mode name. That matches the current surface: `tsBinary`, `resolveTSBinary`, and the two-subject loop remain live at [main.go:178](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:178), [main.go:202](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:202), and [main.go:321](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:321).

##### 06-02 — Workflow publishing

- The workflow verifier is properly job-scoped. This matters because the current workflow’s publish predecessor and `rebless` both contain upload behavior; a file-wide assertion could be satisfied by the wrong job. The plan’s parsed `jobs.publish` design at [06-02-PLAN.md:143](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:143) closes that loophole.

- The post-GREEN perturbation is well chosen: removing only publish’s `if-no-files-found` while leaving rebless intact directly distinguishes subtree-scoped verification from a file-wide grep.

- The `both` dispatch path, ordered action list, runner identity, summary output, and non-gating contract are all explicit. These are materially stronger than comment-only assertions.

##### 06-03 — Regression proof and comment sweep

- The committed baseline becomes an actual test input, not merely a protected file. The plan’s new test derives current values from the committed frame, which avoids the platform/runner/scratch-fs category errors that precede numeric comparison in [regression.go:105](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:105).

- The mutation targets are sound. The live branches are at [regression.go:112](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:112) and [regression.go:120](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:120); widening their tolerances makes the existing `want error` assertions fail at [regression_test.go:509](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:509).

- Scoping band counters to `--- FAIL` lines fixes the prior vacuity. The current test names at [regression_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:43) and [regression_test.go:62](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:62) would appear under `RUN` and `PASS` on green, but cannot satisfy the revised failure-line filter.

- The zero-headroom requirement is acceptable because the action explicitly requires both failing subtest lines to be pasted. It is strict, but tied to deterministic Go test names rather than timing or incidental output.

- The revised comment-only mechanism is materially sound for this edit. It strips trailing comments and compares the remaining code portions, so changing only the comment at [metrics.go:9](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/metrics.go:9) passes, while changing `Subject` to `Subj` changes the retained code prefix and fails. The plan also records the known `//`-inside-string limitation.

##### 06-04 — Publication and census

- Independent remeasurement reproduced the claimed bounded census exactly:

  - `docs/BENCHMARKS.md`: 47
  - `tools/bench/runner/main.go`: 35
  - `.github/workflows/bench.yml`: 26
  - `tools/bench/realcorpus/manifest.go`: 12
  - Six captures: 5 each
  - Remaining declared targets: the stated 1–5 counts
  - Total: 170

- The bounded set produces zero hits in [corpora/manifest.json:10](/Volumes/Code/github.com/seanb4t/codegraph-go/corpora/manifest.json:10), correctly preserving the `TS/JS` indexed-language statement. The bare completeness control still finds that line.

- Every non-zero per-file census count maps to one declared owning plan: runner/captures to 06-01, workflow and baseline prose to 06-02, `internal/bench` to 06-03, and benchmark documentation to 06-04.

- Generating documentation rows through `publishcheck -emit-rows` makes all metric cells traceable. This is substantially safer than spot-checking throughput alone.

- BENCH-03 cannot be closed silently. The blocking decision and explicit pending token preserve honest phase status if no real workflow run exists.

##### 06-05 — Agent-facing files

- Enumerate-before-edit and source-before-mirror are the right ordering. The current stale mirror is visible at [.claude/CLAUDE.md:15](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:15), while PROJECT.md already carries the newer constraint at [.planning/PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).

- The decision not to regenerate the stack block from its now-unrelated named source is well guarded by ordered headings, marker identity, and stack-specific anchor counts.

- The Core Value equality check is a strong positive assertion. It avoids merely deleting two known phrases while allowing different stale text to survive.

##### 06-06 — Durable memory

- The read-only enumeration, one-way approval checkpoint, and supersede-only execution correctly reflect the external store’s risk.

- Both approved and executed ID extractors are now bounded to their respective `##` sections. The `awk` boundaries at [06-06-PLAN.md:341](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:341) prevent rows in adjacent sections from contaminating either set.

- Requiring non-empty raw section counts, duplicate checks, and exact set equality closes both silent-empty and duplicate-collapse cases.

- The literal heading words and `pages enumerated:` wording are now stated directly in the action rather than being hidden acceptance-test requirements.

#### Concerns

- **MEDIUM — 06-05 can preserve live `reimplementation` framing in `.claude/CLAUDE.md`.**  
  PROJECT.md’s current Licensing constraint says the project “began as a reimplementation” at [.planning/PROJECT.md:246](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:246). Plan 06-05 calls that current PROJECT wording authoritative for re-syncing the Licensing mirror at [06-05-PLAN.md:256](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:256), even though Task 1’s enumeration includes `reimplementation` at [06-05-PLAN.md:161](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:161). The final whole-file CLAUDE gate omits that term at [06-05-PLAN.md:331](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:331). Therefore correct execution can re-sync “reimplementation” into live startup instructions and still report `CLAUDE_MD_FRAMING_TOTAL=0`. This weakens MEM-02 and the milestone’s attribution-consolidation goal.

- **LOW — 06-04’s “expected one bare TS residual” statement is narrower than the actual acceptance logic.**  
  The logic correctly allows `>=1` and requires adjudicating every line, so execution remains safe. The prose expectation at [06-04-PLAN.md:564](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:564) should be treated as informational; a rewritten benchmark document could legitimately add another TypeScript-language reference.

- **LOW — Full Go command execution could not be repeated in this read-only review environment.**  
  Go failed while creating its build cache. I independently reproduced the 54-test result from all `func Test*` declarations and reproduced the census results with the exact patterns. This is an environment limitation, not evidence against the plans.

No fifth verification assertion was found that necessarily fails on correct work. The strictest candidate—the two-per-band `--- FAIL` count—has zero slack but is satisfiable under the explicitly required complete paste.

#### Suggestions

- In [06-05-PLAN.md](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md), explicitly add `.planning/PROJECT.md`’s Licensing constraint to the sweep table and require it to become origin-neutral, for example: “MIT. Legal attribution is contained in `NOTICE`; `LICENSE` remains verbatim MIT text.”

- Add `-e '\breimplementation\b'` to `CLAUDE_MD_FRAMING_TOTAL` in 06-05 Task 2. Also assert the Licensing bullet remains present so deletion cannot buy the zero.

- Clarify in 06-04 that the expected bare-TS residual population may grow only through adjudicated indexed-language uses; the executable `>=1` rule is already correct.

#### Risk Assessment

**Overall risk: MEDIUM-LOW.**

The benchmark and CI half is unusually well verified: the cycle-4 measurements hold, cross-plan census collisions are controlled, and mutation proofs are tied to the real failure path. The memory work has appropriate fail-loud and one-way approval controls. The remaining material risk is narrow and local to Plan 06-05: a live origin-derived “reimplementation” clause can survive both re-sync and the final gate. Fixing that explicit term and source line should bring the plans to LOW risk.

---

## Risk Assessment — Cycle 4

**Overall risk: LOW.**

The benchmark and CI half of the phase is verified to an unusual standard: every expected value in
the plans was re-derived from the working tree during this review and every one matched, including
thirteen per-file census counts, two independent test-surface measurements, the workflow `uses:`
pinning arithmetic, and a red/green/red reproduction of the restated comment-only gate. The two
mechanisms that carry the most verification weight — the mutation rehearsal's band counters and the
census zero — now both discriminate, having been shown to read `0/0` and to be reachable
respectively. The memory half retains its one-way approval checkpoint and its fail-loud extractors,
and is bounded by locked decisions rather than by executor judgement.

The residual risk is documentation-shaped, not execution-shaped: one authorised edit introduces a
term into agent instructions without recording the adjudication, and one gate keeps a bare pattern
the rest of the phase retired. Neither prevents the phase goal, creates unsafe execution, or
invalidates verification. **These plans are ready to execute.**

## Verification coverage — Cycle 4

Every figure below was produced by running the command at `0b996e9`, not by reading the plan.

| Claim under test | Command run | Result | Plan's stated value |
|---|---|---|---|
| 06-01 before-set size | `go test -list '.*' ./tools/bench/... \| rg '^Test' \| sort \| wc -l` | **54** | 54 ✓ |
| 06-01 no duplicate names | `sort -u … \| wc -l` | **54** | 54 ✓ |
| 06-01 `cpudiag` contributes none | `go test -list '.*' ./tools/bench/cpudiag/...` | `[no test files]` | ✓ |
| 06-01 removal names present | `rg -n` on the sorted capture | lines **32, 52, 53** | 32, 52, 53 ✓ |
| 06-01 no fourth forced removal | `rg -n 'tsBinary\|resolveTSBinary\|headtohead' tools/bench/runner/main_test.go` | only the three named tests | ✓ |
| 06-04 census total | bounded alternation over the declared surface | **170** | 170 ✓ |
| 06-04 `corpora/manifest.json` | same, that file alone | **0** | 0 ✓ |
| 06-04 per-file table (13 rows) | same, per file | 47 / 35 / 26 / 12 / 5 / 5 / 5×6 / 3 / 2 / 2 / 1 / 1 / 1 — sums to **170** | all ✓ |
| 06-04 bare-TS residual reachable | `rg -U -n -i '\bTS\b'` over the surface | **62** lines today; `corpora/manifest.json:10` is swept by no plan, so `>= 1` survives | ✓ |
| 06-04 census surface size | `rg -U --files` over the surface | **26** today → **23** after execution | `>= 12` ✓ |
| 06-04 critical-file survival | membership check of the 13 names | all 13 present | `= 13` ✓ |
| 06-03 bounded scope | bounded alternation over `internal/bench/`, `internal/corpora/manifest.go`, `corpora/manifest.json` | **7** hits over 6 lines, all in files 06-03 sweeps; `corpora/manifest.json` → **0** | 7 / 0 ✓ |
| 06-03 class fix E — old gate | `rg -v '^[+-]\s*//'` on a comment-only scratch diff | **2** (red) | 2 ✓ |
| 06-03 class fix E — new gate | code-portion equality on the same diff | `CODE_PORTION_IDENTICAL=true` | green ✓ |
| 06-03 class fix E — planted code change | `Subject` → `Subj` | **RED** | red ✓ |
| 06-03 `--- FAIL` scoping discriminates | pattern trace over Go's `-v` rendering | band names appear on `--- FAIL` lines exactly twice per family | `>= 2` reachable ✓ |
| 06-02 workflow pinning | `rg -o 'uses: *[^ ]+' .github/workflows/bench.yml` | **22** total, **20** pinned, **2** local | 19 + 2 after the one deletion ✓ |
| 06-02 Taskfile bench targets | `rg -n '^  bench' Taskfile.yml` | `bench:regression:` only | ✓ |
| 06-05 framing gate today | the gate's own alternation over `.claude/CLAUDE.md` | **5** occurrences on lines **7, 15, 93** | "three hits to clear … lines 7, 15 and 93" ✓ |
| 06-05 structural companions | markers / headings / stack anchors | **14** / **25** / **16** | 14 / 25 / `>= 12` ✓ |
| 06-06 `awk` boundary vs `### ` | regex trace of `/^## /` against `### ` | no match — subsections stay in window | ✓ |
| Cross-plan fixture collision | census alternation vs every new-file literal in all six plans | `"other"` unmatched; `"ts"` matched; 06-03's pinned strings unmatched | ✓ |

**Not verifiable here, stated as such:** the four *added* test names cannot be measured before
execution — only the arithmetic and the removal set can be, and both were. `06-04`'s
`BENCH03_STATUS` path depends on a real CI run that does not exist yet, which is exactly why the
plan makes it a blocking decision gate with an explicit `pending-no-ci-run` token. The engram store
was not queried; D-16 already records it as unreachable at context-gathering time.

---

# Cycle 3 — audit history (superseded by Cycle 4 above)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer against the working tree at `fd02272`.

## Consensus Summary — Cycle 3

The cycle-2 replan did what it claimed. **All five claims the replan makes were independently
verified against source and plan text, and all five hold.** The `TOLNAMED` gate that cycle 2 flagged
as unsatisfiable is now asserted on `want error` plus the Go-mangled band-naming subtest names, and
both are text the mutated path genuinely emits; `CATERR=0` survives a full verbatim `-v` paste
because Go renders the category subtests' names with underscores while the negative grep searches
space-separated phrases. The self-contradictory pre-plan-SHA guard is replaced by two properties
that are mechanically true given the task ordering. The `.claude/CLAUDE.md` sweep is bounded by
three criteria whose numbers all check out against the file as it stands today. The planner's own
class fix C is real and completely applied — all nine sites now count with `wc -l`, and no new
instance of the broken form was introduced. The three rejected findings are each legitimate and
each recorded in a PLAN.md rather than only in a commit message.

**Two new HIGHs survive, and both are the same defect class in new places — an assertion the correct
path cannot satisfy.**

- **HIGH 1 — 06-01.** Task 1 asserts an exact before/after `go test -list` delta over
  `./tools/bench/...` whose allowed *added* set contains exactly three names, while the same task
  creates the `tools/bench/publishcheck` package with its own test file, necessarily contributing at
  least one more top-level `Test*` name to that package pattern.
- **HIGH 2 — 06-03 and 06-04.** Both censuses assert a total of `0` over a scope that includes
  `corpora/manifest.json`, whose line 10 carries `TS/JS` inside a `LOCKED (01-06)` corpus-language
  note. That file is in no plan's `files_modified`, the text is the plans' own declared KEEP class,
  and the two escapes an executor has — reword it, or add an exclusion — are both explicitly
  prohibited. The sanctioned escape, "bound the pattern", is unavailable because the pattern is
  hard-coded inside two `<automated>` blocks.

Neither is a style defect. 06-01 and 06-03 are both `wave: 1`, and `06-02 → 06-04 → 06-05 → 06-06`
chain off them, so between the two findings every plan in the phase is affected.

The lesson this phase earned applies to both verbatim: **a positive assertion the correct path
cannot satisfy is worse than no assertion at all**, because it fails on correct work and trains the
executor to treat red as expected. Cycle 1's class fix B produced class fix C by exactly this
mechanism; these two are the same shape one layer up — an exact *set*, and an over-broad *pattern*,
rather than an exact *count*.

Both fixes are small and local, and neither requires a locked decision to be re-opened.

### Agreed Strengths

- **The `TOLNAMED` rewrite is correct, not merely reworded.** `internal/bench/regression.go:113-126`
  is unreachable once either tolerance is widened to `1.0`, so the plan's new prohibition
  (`06-03-PLAN.md:49`) forbidding a criterion that demands `throughput regressed` / `peak RSS grew`
  is exactly right, and the replacement assertion at `06-03-PLAN.md:448` names only
  `want error` (emitted at `internal/bench/regression_test.go:511`) and the two band subtest names.
- **The band names are reachable twice per family, by construction.** Task 2 pins the new test's
  subtests as `committed baseline throughput 11% slower: exceeds band fails` and
  `committed baseline peak RSS 16% larger: exceeds band fails` (`06-03-PLAN.md:286-290`), whose
  Go-mangled forms *contain* the `throughput_11%_slower` / `peak_RSS_16%_larger` substrings the gate
  greps for. The planner's dry-run figures (`WANT_ERROR_ASSERTIONS=4 THROUGHPUT_BAND_REDS=2
  RSS_BAND_REDS=2`) fall out of this arithmetic exactly.
- **The CATERR prose trap is closed explicitly.** `06-03-PLAN.md:429-432` instructs the executor not
  to quote the three category phrases in the log's own prose, because the gate counts them over the
  whole file — the one way a correct rehearsal could still have tripped it.
- **Class fix C was applied beyond the nine named sites, as a reading discipline.** 06-04, 06-05 and
  06-06 each carry an explicit "checked, not assumed" row (`06-04-PLAN.md:143`, `06-05-PLAN.md:113`,
  `06-06-PLAN.md:123`) recording that their only surviving `rg -c` uses are `= "1"` comparisons
  backed by a real match. That is the right way to record a negative.
- **The three rejections are recorded where the executor will read them**, not only in git history:
  `06-03-PLAN.md:144`, `06-05-PLAN.md:112`, `06-02-PLAN.md:283`.

### Agreed Concerns

Two concerns rise to HIGH — 06-01's exact test-surface delta, and the 06-03/06-04 census zero — both
stated in full below. Four non-HIGH findings follow them (two MEDIUM, two LOW), each with the
concrete PLAN.md change it needs.

### Divergent Views

Codex and the orchestrating reviewer agreed on HIGH 1 and on all five claim verdicts. HIGH 2 and the
four non-HIGH findings came from the orchestrating reviewer's own measurements and were not in
Codex's output; none contradicts anything Codex reported.

**One cycle-2 statement is refuted, not merely superseded.** The cycle-2 coverage table below records
*"Phase-6 census is reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a
declared sweep target."* Re-measured file-by-file this cycle, that is wrong for exactly one file:
`corpora/manifest.json` holds one hit and is a declared sweep target of no plan. HIGH 2 is the
consequence. The cycle-2 aggregate count was right; the "every one" claim was not checked
per-file, and the aggregate hid it.

Codex's line citations for `internal/bench/regression_test.go` and `regression.go` were each off by
roughly two lines (`:509`/`:112`/`:120` against measured `:511`/`:113`/`:121`); the substance was
correct and the measured numbers are used throughout this cycle.

---

## HIGH 1 (CYCLE 3 — RESOLVED in `0b996e9`, re-verified in cycle 4) — 06-01's exact test-surface delta is unsatisfiable by its own Task 1

**Where.** `06-01-PLAN.md:272` (before-snapshot), `06-01-PLAN.md:380-381` (the delta assertion),
`06-01-PLAN.md:470` (the same statement in `<verification>`), `06-01-PLAN.md:86-93` (artifact
register).

**The contradiction, measured.**

1. Task 1's action opens with *"Before touching a single file"* and captures
   `go test -list '.*' ./tools/bench/... | rg '^Test' | sort` (`06-01-PLAN.md:263,272`). Measured
   against the tree at `fd02272`, that before-set is **55 names** across four packages
   (`go list ./tools/bench/...` → `cpudiag`, `gencorpus`, `realcorpus`, `runner`). It contains all
   three names the plan expects to remove — `TestParseFlags_OverridesApply`,
   `TestResolveTSBinary_EmptyWhenNotFound`, `TestResolveTSBinary_FindsOnPath` — so the *removed*
   half of the assertion is sound.
2. The **same task** creates `tools/bench/publishcheck/main.go` and
   `tools/bench/publishcheck/main_test.go` (`06-01-PLAN.md:12-13`, `files_modified`;
   `06-01-PLAN.md:241-254` specifies seven behaviours for that test file).
   `tools/bench/publishcheck` is inside the `./tools/bench/...` package pattern, so its top-level
   test names appear in the **after**-list at `06-01-PLAN.md:380`.
3. `06-01-PLAN.md:371` independently requires `go test -run TestPublishCheck -v
   ./tools/bench/publishcheck/...` to name the rejecting cases as RUN/PASS — which is only possible
   if at least one top-level test whose name matches `TestPublishCheck` exists.
4. But `06-01-PLAN.md:381` states: *"the added set MUST equal exactly `TestCorporaHasExactlyTwoEntries`,
   `TestParseFlags_PublishOverridesApply`, `TestParseFlags_RejectsRetiredComparisonBinaryFlag`"* and
   requires the executor to *"verify with `diff` against those two literal sorted lists."*

A correct execution therefore produces an added set of **at least four** names and the `diff`
against the three-name literal list is non-empty. The criterion fails on correct work.

**Why HIGH rather than MEDIUM.** 06-01 is `wave: 1` with `depends_on: []`; 06-02 (`wave: 2`), 06-04
(`wave: 3`), 06-05 (`wave: 4`) and 06-06 (`wave: 5`) chain off it. A gate that reddens on correct
work at the head of the chain halts five of six plans, and the failure looks like a real
regression — the executor's most likely response is to hunt for a test that "vanished for another
reason" that does not exist, or to weaken the assertion ad hoc. That is precisely the outcome the
phase's own lesson forbids.

**The fix (small, entirely inside 06-01).**

1. `06-01-PLAN.md:241` — name the publishcheck test symbol(s) explicitly in the `<behavior>` block.
   Prefer **one** top-level `TestPublishCheck` carrying the seven cases as subtests: that adds
   exactly one stable name to the delta while `-run TestPublishCheck -v` still names each case.
2. `06-01-PLAN.md:381` — extend the allowed added set to the four-name literal list
   (`TestCorporaHasExactlyTwoEntries`, `TestParseFlags_PublishOverridesApply`,
   `TestParseFlags_RejectsRetiredComparisonBinaryFlag`, `TestPublishCheck`).
3. `06-01-PLAN.md:470` — restate "three removals and three additions" as "three removals and four
   additions" so `<verification>` agrees with the criterion.
4. `06-01-PLAN.md:86-93` — add the concrete test symbol to the artifact register, which today lists
   the publishcheck *files* but not the test symbol the delta assertion turns on.

An acceptable alternative is to scope both snapshots to the packages that existed before the task
(`./tools/bench/realcorpus/... ./tools/bench/runner/... ./tools/bench/cpudiag/...
./tools/bench/gencorpus/...`), leaving the new package's surface to `06-01-PLAN.md:371`. The
four-name list is preferable: it keeps the assertion over the whole `tools/bench` surface, which is
what the criterion exists to police.

---

## HIGH 2 (CYCLE 3 — RESOLVED in `0b996e9`, re-verified in cycle 4) — the retired-term census cannot reach zero in either 06-03 or 06-04: `corpora/manifest.json:10` is matched by `\bTS\b` and is in no plan's `files_modified`

**Where.** `06-03-PLAN.md:224` (`BENCH_PKG_RETIRED_TERMS_TOTAL = 0`), `06-04-PLAN.md:494` and the
matching criterion at `06-04-PLAN.md:501` (`PHASE6_CENSUS_TOTAL = 0`).

**Measured.** Both censuses scope `corpora/manifest.json` and both include a bare `\bTS\b`
alternation (06-04 additionally with `-i`). Running each pattern set verbatim against the tree at
`fd02272`:

```
06-03 census over its full scope  (internal/bench/ internal/corpora/manifest.go corpora/manifest.json) → 7
06-03 census over corpora/manifest.json alone                                                          → 1   (line 10)
06-04 census over corpora/manifest.json alone, case-insensitive                                        → 1   (line 10)
```

The single hit is the substring `TS` inside `TS/JS` at `corpora/manifest.json:10`:

> `"note": "LOCKED (01-06): closes the Go priority-4 coverage gap and supplies TS/JS via its 25
> javascript tracked files, which is how this minimum-cardinality set covers all five priority
> languages. Roadmap shortlist."`

**Why a correct execution cannot clear it.**

1. `corpora/manifest.json` appears in **no** plan's `files_modified` — checked across all six.
   06-03's list carries `internal/corpora/manifest.go` (the Go file) but not the JSON.
2. 06-03's declared reading and editing scope for that file is line **2**, the *top-level* `note`
   (`06-03-PLAN.md:166`, `:220-221`). Line 10 is a *per-entry* note and is outside it. Line 2 is
   already clean for every 06-03 pattern — measured.
3. The flagged text is the plan's own **KEEP class**. `06-04-PLAN.md:448-449` already excludes
   `internal/indexer/**` and `testdata/golden/**` as "TypeScript-the-indexed-language product
   surface … Phase 5's recorded KEEP class". `TS/JS` in a corpus-language-coverage note is that
   class exactly — it is not retired comparison framing.
4. Both escapes are closed to the executor. `06-04-PLAN.md:45` prohibits rewording innocent
   third-party prose *or* widening an exclusion to make the total read zero; `06-04-PLAN.md:451-452`
   forbids adding an exclusion not on the list. The sanctioned remedy — `06-04-PLAN.md:454-455`,
   *"If a pattern flags text that is genuinely innocent, the finding is about the pattern. Bound it
   tighter"* — is unavailable at execution time, because the pattern is **hard-coded inside the
   `<automated>` blocks** of two different plans, one of which (06-03) is `wave: 1` and
   `autonomous: true`.

So the executor of 06-03 hits a terminal gate that reads `1`, cannot legitimately reach `0`, and has
no in-plan instruction covering the case. 06-04 then hits the same wall with a wider pattern set.

**This also refutes a cycle-2 verification.** The cycle-2 coverage table recorded *"Phase-6 census is
reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a declared sweep
target."* That is wrong for exactly one file: `corpora/manifest.json` is not a declared sweep target
of any plan, and it holds one of those hits.

**The fix (in PLAN.md, before execution — it cannot be done by the executor).**

1. `06-03-PLAN.md:224` and `06-04-PLAN.md:494` — bound `\bTS\b` to the retired-framing senses the
   sweep is actually about, e.g. `\bTS CodeGraph\b`, `\bGo-vs-TS\b`, `\bts-binary\b`, `\bTS binary\b`,
   dropping the bare alternation. 06-03 already carries `\bGo-vs-TS\b` as a separate pattern, so its
   bare `\bTS\b` is largely redundant; 06-04's `-i` additionally makes it match any lowercase `ts`
   identifier that ever appears on the census surface, which is a standing false-positive generator.
2. `06-04-PLAN.md:445-452` — record the bounding in `06-CENSUS.md` with the flagged text quoted, as
   `06-04-PLAN.md:454-455` requires. That is the plan's own prescribed handling; it just has to be
   pre-authorised rather than improvised.
3. `06-03-PLAN.md:7-15` — while here, add `corpora/manifest.json` to `files_modified`, or state
   explicitly that it is read-only to this plan. `06-03-PLAN.md:220-221` already contemplates editing
   it (*"If `corpora/manifest.json`'s `note` needs a change, say so in the SUMMARY: it is committed
   data, so the edit ships in the repo"*) while the frontmatter does not declare it — an executor
   following the action would write a file the plan says it does not modify.

---

## Non-HIGH findings

### [CYCLE 3 — RESOLVED in `0b996e9`] MEDIUM 1 — 06-06's approved-set extraction is still file-wide while the executed-set extraction is section-bounded

Cycle 2's MEDIUM 4 replaced the executed-set extractor with a section boundary
(`awk '/^## Supersedes executed/{f=1;next} /^## /{f=0} f'`, `06-06-PLAN.md:321`), which is correct.
The **approved**-set extractor on the same line was left file-wide:
`rg -N '^\|' "$F" | rg -F '| supersede |'`.

`06-06-PLAN.md:196` fixes the enumeration table's columns as
`short_id | scope | durable content | verdict | reason`, so `| supersede |` is a verdict literal —
but the extractor scans the whole document, including `## Supersedes executed (MEM-01)` (columns at
`06-06-PLAN.md:292-295`) and `## Completeness and accepted gaps`. If any executed row or completeness
row renders a bare `| supersede |` cell, that row's first-column `short_id` joins the approved set as
a duplicate, `APPROVED_SUPERSEDE_IDS != APPROVED_UNIQUE`, and the task fails on correct work. The
risk is modest — the executed table's third column is specified as "the operation name called", which
in practice is `supersede_memory`, not `supersede` — but this is the identical failure mode the
section-boundary fix was introduced to remove, left applied to only one of the two sets.

**PLAN.md change needed:** at `06-06-PLAN.md:321`, bound the approved-set extraction to its own
section with the same technique — `awk '/^## Engram enumeration/{f=1;next} /^## /{f=0} f' "$F"` piped
into the existing `rg -F '| supersede |'`. (`^## ` does not match the `### ` per-scope subheadings, so
the scope sections stay inside the window.) Record the extractor's raw line count in the SUMMARY
alongside the executed set's, as `06-06-PLAN.md`'s own criterion already requires for the executed
half.

### [CYCLE 3 — RESOLVED in `0b996e9`] MEDIUM 2 — two of the four counters in 06-03's replacement gate pass vacuously on a GREEN run (measured)

`06-03-PLAN.md:448` asserts four numbers over `06-MUTATION-LOG.md`: `CATERR = 0`,
`WANTERR >= 2`, `THROUGHPUT_BAND_REDS >= 2`, `RSS_BAND_REDS >= 2`. The rehearsal command at
`06-03-PLAN.md:376` uses `-v`, and `-v` prints `=== RUN` and `--- PASS` lines for **every** subtest,
not only failing ones.

Ran the gate's own four counters against a verbatim `-v` paste of the **unmutated** tree at `fd02272`
(`go test -count=1 -v -run 'TestCheckRegression' ./internal/bench/...`):

```
CATERR=0   THRU=2   RSSN=2   WANTERR=0
```

So `THROUGHPUT_BAND_REDS` and `RSS_BAND_REDS` are **already satisfied by a green run** — they measure
that the subtests exist, not that either one went red. Only `WANTERR` discriminates: it is `0` on
green and becomes `>= 2` only on the mutated path. The reachability note at `06-03-PLAN.md:449`
("Family A's pasted `-v` output carries `throughput_11%_slower` on the table subtest's `=== RUN` and
`--- FAIL` lines … so `>= 2` has headroom") is therefore true but describes the counters as evidence
of a RED when they are not.

The gate as a whole is **not** vacuous — `WANTERR` carries it, and `RED_BLOCKS_RECORDED >= 2`
(`06-03-PLAN.md:439`) plus `COMMITTED_BASELINE_REDS_RECORDED` (`06-03-PLAN.md:446`) enforce
two-family coverage — which is why this is MEDIUM and not HIGH. But rule `84d1gfpywd` is precisely
about a positive assertion that can pass without the thing it names having happened, and two of these
four can.

**PLAN.md change needed:** at `06-03-PLAN.md:448`, tie the band counters to the RED rather than to the
paste — count the band name only on `--- FAIL` lines, e.g.
`THRU=$(rg -N -- '--- FAIL' "$LOG" | rg -o -F 'throughput_11%_slower' | wc -l | tr -d ' ')` and the
same for `peak_RSS_16%_larger`, keeping `>= 2` (table + committed-baseline subtest per family). Then
amend the reachability note at `06-03-PLAN.md:449` to record the measured green-run values above as
the reason the counters were narrowed.

### [CYCLE 3 — RESOLVED in `0b996e9`] LOW 1 — 06-05's `CLAUDE_MD_CHANGED_LINES <= 24` ceiling is tighter than the task's own scope statement

`06-05-PLAN.md:290-292` expects "roughly four changed lines" (7, 15, 17, 93) — eight diff lines,
comfortably inside the 6..24 window, and the floor of 6 is exactly reachable since at least three
distinct lines must change (measured: framing matches live at `.claude/CLAUDE.md:7`, `:15`, `:93`).
But the same action also says "Re-sync the generated PROJECT block from the corrected PROJECT.md"
(`06-05-PLAN.md:287`), and 06-05 edits `.planning/PROJECT.md` in the same task. The generated block
spans `.claude/CLAUDE.md:1-19`; if the PROJECT.md corrections touch the Core Value paragraph (line 9)
and more than one further constraint bullet, the re-sync alone can push past twelve changed lines and
trip the ceiling on correct work.

**PLAN.md change needed:** at `06-05-PLAN.md:290`, state the ceiling's basis — that 24 diff lines
allows up to twelve changed lines, i.e. the four expected plus eight of slack for the PROJECT-block
re-sync — and add a one-line instruction that if the re-sync legitimately exceeds it, the executor
records the count and the reason rather than trimming the re-sync to fit the number.

### [CYCLE 3 — RESOLVED in `0b996e9`] LOW 2 — two of 06-06 Task 1's gates require literal phrases its `<action>` never asks for

`06-06-PLAN.md:203` asserts `PAGES=$(rg -o -i 'pages enumerated' …); test "$PAGES" -ge 3` and
`SCOPES=$(rg -o -i '^### .*(spine|rule:repo|overlay)' …)`. Both are counted over
`06-MEMORY-SWEEP.md`, a file this same task authors — so a correct execution *can* satisfy them, but
only by guessing the wording. The `<action>` asks the executor to "record the page count" and to
write "one subsection per scope"; it never states that the literal phrase **"pages enumerated"** must
appear three times, nor that the `###` headings must contain `spine` / `rule:repo` / `overlay`.

This is the mildest form of the family this cycle is about: not unsatisfiable, but satisfiable only
by accident. It is LOW because the whole file is executor-authored and a re-read of the criterion
resolves it.

**PLAN.md change needed:** in `06-06-PLAN.md`'s Task 1 `<action>`, state the two required literals
directly — the per-scope subsection heading form (`### <scope> — spine` / `rule:repo` / `overlay`)
and the phrase "pages enumerated: <n>" once per scope — so the executor writes them deliberately
rather than reverse-engineering them from the gate.

---

## Risk Assessment — Cycle 3

**MEDIUM until HIGH 1 and HIGH 2 are corrected; LOW afterward.**

The verification architecture is now sound in design: every load-bearing gate in the six plans was
traced to the file and line it asserts over, and with the two exceptions above each one both fails on
a mutated tree and passes on a correct one. What remains is not architectural — it is two gates whose
constants were not re-measured against the tree they will run on.

Both HIGHs are narrow and fixable entirely inside PLAN.md: four edits in `06-01-PLAN.md` for HIGH 1,
and one pattern-bounding edit in each of `06-03-PLAN.md:224` and `06-04-PLAN.md:494` (plus a
`files_modified` correction) for HIGH 2. Neither requires new work, new tasks, or re-planning. But
both sit on `wave: 1` plans, so both must be fixed *before* execution rather than caught during it —
and neither is something the executor is permitted to fix for itself.

The 16 locked decisions D-01..D-16 remain respected. **No finding in this cycle requires re-opening a
locked decision**, and no finding is an escalation about one. In particular HIGH 2 is a pattern
defect, not a challenge to D-08 or to the Phase-5 KEEP-class adjudication the plans correctly cite —
it is the plans' own doctrine (`06-04-PLAN.md:454-455`, "the finding is about the pattern") applied
one file further than the plans applied it themselves.

---

## Verification coverage — Cycle 3

Codex ran with full repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present. The orchestrating reviewer independently
re-executed or re-read every load-bearing claim below against the working tree at `fd02272` before
accepting it. Where Codex's line numbers differed from measurement (it cited
`internal/bench/regression_test.go:509` and `regression.go:112/120`), the measured values are used.

| Claim | Source / command actually run | Verdict |
|---|---|---|
| Cycle-2 claim 1 — mutated path emits `want error`, not the tolerance strings | `internal/bench/regression.go:9,15,113-126`; `rg -n 'want error' internal/bench/regression_test.go` → **511** | **VERIFIED** — widening either constant to `1.0` makes 113-126 unreachable; the only emitted assertion is at `:511` |
| Cycle-2 claim 1 — band subtest names are Go-mangled substrings the gate can grep | `internal/bench/regression_test.go:43,62` → `throughput 11% slower: exceeds band fails`, `peak RSS 16% larger: exceeds band fails`; `06-03-PLAN.md:286-290` pins the committed-baseline analogues | **VERIFIED** — `throughput_11%_slower` / `peak_RSS_16%_larger` are substrings of both pairs |
| Cycle-2 claim 1 — the planner's dry-run figures are arithmetically consistent | 2 failing subtests per family × 2 families = `WANT_ERROR_ASSERTIONS=4`; table + committed-baseline per band = `THROUGHPUT_BAND_REDS=2`, `RSS_BAND_REDS=2` | **VERIFIED** — figures fall out of the specified test surface |
| Cycle-2 claim 1 — `CATERR=0` holds against a full `-v` paste | Ran the gate's four counters over a verbatim `-v` paste of the unmutated tree: `CATERR=0 THRU=2 RSSN=2 WANTERR=0`; underscore rendering confirmed (`=== RUN TestCheckRegression/runner_mismatch_between_baseline_and_current_fails…`); error strings at `regression.go:74,94` print only on failure | **VERIFIED** — and the same measurement surfaced MEDIUM 2: `THRU`/`RSSN` are already 2 on green |
| Tree is green at `fd02272` before any plan work | `go build ./...`, `go vet ./internal/bench/... ./tools/bench/...`, `go test -count=1 ./internal/bench/... ./tools/bench/...` | **CONFIRMED** — all green; the plans start from a clean baseline |
| Cycle-2 claim 2 — HEAD-relative property is true by construction | `06-03-PLAN.md:334-336`; Task 2's `files_modified` adds only `internal/bench/baseline_gate_test.go` | **VERIFIED** |
| Cycle-2 claim 2 — pre-plan-SHA property is comment-only-scoped | `06-03-PLAN.md:337-339` — `rg '^[+-][^+-]' \| rg -v '^[+-]\s*//' \| wc -l` over the four files | **VERIFIED** — no `git diff --quiet` "apart from" semantics remain |
| Cycle-2 claim 3 — 25 headings | `rg '^#{1,6} ' .claude/CLAUDE.md \| wc -l` → **25** | **VERIFIED** |
| Cycle-2 claim 3 — `>= 12` stack-block anchors | `rg -o -F -e 'The Parser Decision' -e 'cockroachdb/pebble' -e 'tree-sitter/go-tree-sitter' .claude/CLAUDE.md \| wc -l` → **16** | **VERIFIED** — floor holds with headroom |
| Cycle-2 claim 3 — GSD marker count | `rg -o '<!-- GSD:[^>]*-->' .claude/CLAUDE.md \| wc -l` → **14** | **VERIFIED** |
| Cycle-2 claim 3 — the 6..24 window is satisfiable | Framing matches live at `.claude/CLAUDE.md:7` (three on one line), `:15`, `:93`; plan also sweeps `:17`. 3 lines minimum → 6 diff lines; 4 expected → 8 | **VERIFIED at the floor** — ceiling caveat recorded as LOW 2 |
| Cycle-2 claim 4 — all nine class-C sites converted to `wc -l` | `PINDIFF` `06-01:376`; `PERMDIFF` `06-02:341`; `FIXDIFF` `06-02:399,407`; `FIGDIFF` `06-02:410`; `NC` `06-03:224`; `CONTENT`/`NONCOMMENT` `06-03:231,338`; `TOLDIFF` `06-03:234`; `CATERR` `06-03:448` | **VERIFIED** — 9/9 |
| Cycle-2 claim 4 — no NEW broken instance | `rg -n '\|\| true' 06-0*-PLAN.md` → 7 hits; every one is either prose describing the defect or a `= "1"` comparison against a real match | **VERIFIED** |
| Cycle-2 claim 4 — 06-04/05/06 genuinely needed none | `06-04-PLAN.md:143`, `06-05-PLAN.md:113`, `06-06-PLAN.md:123`; their only `rg -c` uses are the `NOTICE` copyright `= "1"` checks | **VERIFIED** |
| Class-C shape surviving at `06-03-PLAN.md:176` (baseline capture, not a gate) | Ran it verbatim → `metrics.go=51 regression.go=70 regression_test.go=70 rss.go=18` | **CONFIRMED SAFE** — all four non-zero, so the `±25%` comparison at `06-03:232` is well-defined; not raised as a finding |
| Cycle-2 claim 5a — mutation-alternative rejection recorded in a PLAN.md | `06-03-PLAN.md:144`, and the prohibition it produced at `06-03-PLAN.md:47` (forbids editing the test table) | **VERIFIED** — legitimate and recorded where the executor reads it |
| Cycle-2 claim 5b — `STACK.md` rejection is factually right | `rg -c -i 'parity' .planning/research/STACK.md` → **no match**; rejection recorded at `06-05-PLAN.md:112` | **VERIFIED** — Codex's original finding was false |
| Cycle-2 claim 5c — Node step line numbers | `.github/workflows/bench.yml` → `- name:` at **117**, `uses: actions/setup-node@…` at **118**, `node-version:` at **120**; recorded at `06-02-PLAN.md:283` as `117-120` | **VERIFIED** — the LOW claim of `118-121` was wrong |
| 06-02's `U_PINNED = 19` arithmetic still correct | `rg -o '^\s*uses: '` → 22, `'^\s*uses: \./'` → 2, `'uses: [^@[:space:]]+@[0-9a-f]{40}'` → **20**; `bench.yml:118` is the one pinned action removed, `bench.yml:123` is a `run:` step | **CONFIRMED** — 20 − 1 = 19, + 2 local = 21 total |
| 06-02's Taskfile exact-set + floor | `BENCHTASKS=bench:regression:,`, `ALLTASKS=47` (floor 40) | **CONFIRMED** |
| `\bcomparison\b` floors survive the sweep | `rg -c` → `tools/bench/BASELINE.md` **5** (floor 5, zero headroom — carried forward from cycle 2), `internal/bench/regression.go` **7** (floor 5) | **CONFIRMED** |
| 06-04's three `docs/BENCHMARKS.md` headings are reachable | `rg -n '^## ' docs/BENCHMARKS.md` → `## 1. Methodology` (7), `## 2. Raw numbers` (89), `## 3. Regression gate (PERF-02, INDX-06)` (242); Task 2 renames §2 to `## 2. Absolute numbers` | **CONFIRMED** — `H=3` attainable |
| **NEW HIGH — 06-01 added-set is exactly three** | `go test -list '.*' ./tools/bench/... \| rg '^Test' \| sort \| wc -l` → **55** across `cpudiag`/`gencorpus`/`realcorpus`/`runner`; Task 1 adds package `tools/bench/publishcheck` with `main_test.go` (`06-01-PLAN.md:12-13,241-254`) and requires `-run TestPublishCheck` to name cases (`06-01-PLAN.md:371`) | **REFUTED** — the added set must contain ≥ 4 names; `06-01-PLAN.md:381` and `:470` demand exactly 3 |
| 06-01's *removed* set is sound | All three of `TestParseFlags_OverridesApply`, `TestResolveTSBinary_EmptyWhenNotFound`, `TestResolveTSBinary_FindsOnPath` present in the measured before-set | **CONFIRMED** |
| No other exact-set assertion in 06-01/02/03 is unsatisfiable | `rg -n 'MUST equal exactly\|equal exactly' 06-01/02/03-PLAN.md` — only `06-01:381` and the Taskfile set, which measures true today | **CONFIRMED** |
| **NEW HIGH — census zero over `corpora/manifest.json`** | Ran both pattern sets verbatim: 06-03's over its full scope → **7**, over `corpora/manifest.json` alone → **1** at line 10; 06-04's case-insensitive set over the same file → **1** at line 10. `rg -n 'corpora/manifest.json'` across all six plans' `files_modified` → **absent from every one** | **REFUTED** — `06-03:224` and `06-04:494` assert `= "0"`; the residual hit is `TS/JS` in a `LOCKED (01-06)` note the plans' own KEEP class protects |
| Cycle-2's "every census hit is inside a declared sweep target" | Re-measured per-file rather than in aggregate | **NOT UPHELD** — true for every file except `corpora/manifest.json` |
| 06-03 declares the file it may edit | `06-03-PLAN.md:166,220-221` contemplate editing `corpora/manifest.json`'s top-level note; `06-03-PLAN.md:7-15` `files_modified` lists `internal/corpora/manifest.go` but not the JSON | **REFUTED** — declared-artifact mismatch, folded into HIGH 2's fix |
| 06-04's other exact-equality gates | `CENSUS_FILES_SCANNED` 26 (floor 12); `CENSUS_CRITICAL_PRESENT` **13/13**; positive control `POSCTRL_MULTILINE=5 POSCTRL_LINEBASED=3`; `## 3. Regression gate (PERF-02, INDX-06)` byte-exact at `docs/BENCHMARKS.md:242`; `EMITTED=2` matches the 3→2 corpus drop (`weft-go`, `cockroachdb-pebble` survive `tools/bench/realcorpus/manifest.go`) | **CONFIRMED** — all reachable |
| 06-05's exact-equality gates | `MK=14`, `HD=25`, `STACKANCH=16`, `PV_LEN=208`, `STATE_MD_RETIRED_CORE_VALUE` phrases confined to `STATE.md:26`, `CM=0` with the bullet retained at `PROJECT.md:244`, `state.load` parses | **CONFIRMED** — plan is clean |
| 06-05's re-sync does not break its own gate | The Licensing line re-synced from `PROJECT.md:246` imports "reimplementation", which is **not** in the `CLAUDE_MD_FRAMING_TOTAL` pattern set | **CONFIRMED** |
| 06-06 status-token namespaces do not collide | `MEM02_STORE_STATUS=` is not a substring of `MEM02_FILES_STATUS=` (06-05), and neither matches `BENCH03_STATUS=` (06-04); all three count over the shared `06-LIVE-VERIFICATIONS.md` | **CONFIRMED** — the three "exactly one token" counts stay independent |
| `NOTICE` copyright literal is byte-correct | `rg -c -F 'Copyright (c) 2026 Colby Mchenry' NOTICE` → **1**; the unusual "Mchenry" capitalisation is what the file contains and carries its own do-not-correct warning | **CONFIRMED** |

---

# Cycle 2 — audit history (superseded by Cycles 3 and 4 above)

**Reviewer:** Codex (source-grounded, full repo access), cross-checked and extended by the
orchestrating reviewer against the working tree at `6a2816b`.

## Consensus Summary — Cycle 2

The replan is a substantial, mostly successful convergence. All five cycle-1 HIGHs received real
structural answers rather than prose patches: the D-06 checklist became one publish-job-scoped Go
test, the committed `baseline.json` became a live test input whose own non-vacuity is demonstrated
by the same mutation that reddens the pre-existing table, and BENCH-03 was separated from BENCH-01
behind a blocking decision gate with a two-token ledger. The SHA-pin arithmetic is correct. The
three HEAD-relative guard exemptions are each justified. No sound `rg -c` floor was over-corrected,
and every floor the plans rely on is satisfiable against the current tree.

What remains is concentrated in **verification correctness rather than intent**: three criteria are
mechanically impossible or self-defeating as written, and several instruments are weaker than the
claims resting on them.

### Agreed Strengths

- **06-02's publish-job scoping is real.** Every D-06 assertion in `<behavior>` (06-02-PLAN.md:140-184)
  is anchored to the parsed `jobs.publish` subtree, including the upload step's OWN `with:` map. The
  untouched `rebless` upload at `.github/workflows/bench.yml:274` — verified present, carrying
  `if-no-files-found: error` — can no longer stand in for the publish job's guard. `key_links`
  (06-02-PLAN.md:41) states the property explicitly, so an executor cannot read it as optional.
- **06-03's HIGH 4 fix is genuine, not nominal.** `internal/bench/baseline_gate_test.go` reads
  `../../tools/bench/baseline.json` from disk (06-03-PLAN.md:236-240) — not a temp copy — and Task 3
  requires each mutation family to redden BOTH oracles (06-03-PLAN.md:329-343, 370). The committed
  file at `tools/bench/baseline.json` carries populated `goos`/`goarch`/`runner`/`scratch_fs`, so the
  four category guards in `internal/bench/regression.go:36-105` stay quiet for a frame-derived
  current record and the numeric bands really are the thing under test.
- **SHA-pin arithmetic is correct.** Measured on the working tree: `USES_TOTAL=22`, `USES_LOCAL=2`
  (`./.github/actions/install-task` at `bench.yml:330` and `:380`), `USES_PINNED=20`. The edit deletes
  exactly one pinned action (`actions/setup-node`, `bench.yml:118`), leaving 19 pinned + 2 local = 21.
  The latent false failure cycle 1 flagged is genuinely fixed.
- **The three HEAD-relative exemptions are each correct.** 06-05 Task 1 is the plan's first task, so
  HEAD and the pre-plan SHA are the same commit; 06-03's FIXT-07 gates assert working-tree cleanliness
  at an instant, which is what makes "the mutation applied" and "the revert was byte-clean" provable;
  06-02's fixture check constrains Task 2's own uncommitted diff. None is the blind-window bug wearing
  a rationale (one residual caveat at LOW-3 below).
- **Class fix B was applied without over-correction.** Floors and line-oriented counts were correctly
  left alone, and each is satisfiable today: `rg -c '\bcomparison\b' tools/bench/BASELINE.md` = 5 at
  lines 39/87/97/120/297 — **disjoint from the two lines 06-02 Task 2 sweeps (64 and 241)**, so the
  zero-headroom `>= 5` floor survives the sweep; `internal/bench/regression.go` = 7;
  `TASKFILE_BENCH_TARGETS` = `bench:regression:,` exactly; `ALLTASKS` = 47 (floor 40); `GSD_MARKERS`
  = 14; `SHAPETEST_FIXTURE_ENTRIES` = 10 (floor 10); both surviving corpus pins present.
- **06-05's core-value equality gate executes correctly.** Run against the current tree the extraction
  yields `PROJECT_CORE_VALUE_LEN=208` (floor 60) and a non-empty STATE side, and the Compatibility
  bullet at `.planning/PROJECT.md:244` matches `^\- \*\*Compatibility\*\*` and contains
  `codegraph migrate` exactly once — so the region-scoped criterion is both satisfiable and non-vacuous.
- **The D-15 rejection is legitimate and adequately recorded.** Requiring post-sweep re-query evidence
  would reopen a locked decision (`06-CONTEXT.md:166-172`). It is recorded three times over — in
  `<edge_probe_coverage>`, in `<review_incorporation>`'s "Explicitly NOT applied" block, and now
  mechanically as the `MEM02_STORE_STATUS=accepted-by-d15-evidence-standard` token. This is the right
  shape: the risk is named, not denied.

### Agreed Concerns

Three HIGHs are **new in cycle 2** — two of them introduced by the class-fix restatements themselves.

---

## HIGH 1 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-03 Task 3's `TOLNAMED >= 2` gate cannot pass with the specified mutation

`06-03-PLAN.md:372` requires:

```
TOLNAMED=$(rg -o -e 'throughput regressed' -e 'peak RSS grew' 06-MUTATION-LOG.md | wc -l); test "$TOLNAMED" -ge 2
```

But the mutation widens `DefaultThroughputTolerance` / `DefaultRSSTolerance` to `1.0`
(`06-03-PLAN.md:325`, `:341`), which makes `CheckRegression` return **`nil`**. Those two strings live
only inside the error-construction branches at `internal/bench/regression.go:114` and `:124`, which
the mutation suppresses by design. The observed RED is
`CheckRegression() = nil, want error` — the assertion at `internal/bench/regression_test.go:512`.
No tolerance-naming line can appear in verbatim output.

The same defect is in the plan's prohibition at `06-03-PLAN.md:47` ("MUST NOT record a mutation family
as demonstrated RED unless the observed failure names the tolerance being violated"). Read literally,
the chosen mutation direction obliges the executor to **report a failed rehearsal**, or to hand-write
the phrases as prose — which is exactly the certifying-your-own-result failure the FIXT-07 protocol
exists to prevent.

This was introduced *by* class fix B: the pre-revision form was the `CATERR` half alone; the positive
companion is new and is mis-specified.

**Fix:** assert what the specified mutation actually emits — the two failing subtest names plus the
literal `CheckRegression() = nil, want error` — and keep `CATERR=0` as the wrong-RED disqualifier.
If the intent is genuinely to see the tolerance named, the mutation must move the *current* values
past the band instead of widening the constant, which is a different (and more invasive) rehearsal.

## HIGH 2 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-03 Task 2 carries a self-contradictory pre-plan-SHA guard

`06-03-PLAN.md:296`:

> `git diff --quiet "$BASE" -- internal/bench/regression.go internal/bench/regression_test.go internal/bench/metrics.go internal/bench/rss.go` exits 0 **apart from Task 1's committed comment sweep**

Task 1 commits comment edits to exactly those four files (`06-03-PLAN.md:140`, `:166-192`). A plain
`git diff --quiet "$BASE"` has no "apart from" semantics; it will exit 1 every time. The criterion is
mechanically impossible as written, and an executor that runs it verbatim halts on a correct tree.

**Fix:** either scope it HEAD-relative (Task 2 adds a file and edits none, which HEAD compares exactly),
or keep the pre-plan anchor and filter the diff to comment-only hunks — the same shape Task 1 already
uses at `06-03-PLAN.md:212`.

## HIGH 3 (CYCLE 2 — RESOLVED in `fd02272`, re-verified in cycle 3) — 06-05's stack-block "re-sync" would replace ~119 lines of agent instructions and break the plan's own gate

`06-05-PLAN.md:161` gives the CLAUDE.md:93 occurrence this verdict rule:

> check whether this block still matches its named source (`.planning/research/STACK.md`). **If it is a
> stale mirror, re-sync**; if the phrase is live in the source, sweep the source and re-sync

Measured against the tree: **the phrase is NOT live in the source.** `rg -i parity
.planning/research/STACK.md` returns nothing. But the block is not merely stale — the source has been
**wholly replaced**. `.claude/CLAUDE.md` lines 21-141 mirror the original v1 tech-stack research
(tree-sitter, Pebble, mcp-go, the Parser Decision), while the current
`.planning/research/STACK.md` (198 lines) is the v0.11.0 *agent-onboarding skill/plugin + MCP Resources*
research — an entirely different document. A `diff` of the two shares essentially nothing beyond the
`### Core Technologies` heading.

Executing the stated rule therefore:

1. replaces ~119 lines of live agent instructions with unrelated content — far beyond a framing sweep,
   and in tension with this plan's own prohibition against removing headings in `.claude/CLAUDE.md`
   (`06-05-PLAN.md:44`);
2. **breaks the plan's own acceptance gate**: `.planning/research/STACK.md:151` contains `drop-in`,
   which is in the `CLAUDE_MD_FRAMING_TOTAL=0` pattern set (`06-05-PLAN.md:264`). The re-sync imports
   the very framing the census forbids.

Codex reported an adjacent MEDIUM here — that `.planning/research/STACK.md` belongs in
`files_modified` — which is a **false finding**: nothing in that file needs editing. The real defect is
that the re-sync branch is unexecutable.

**Fix:** give CLAUDE.md:93 an explicit verdict in the plan rather than a conditional. The minimal
correct action is to sweep the `parity` aside in `.claude/CLAUDE.md` in place, and record in
`06-MEMORY-SWEEP.md` that the stack block has diverged wholesale from its named source and that a full
regeneration is out of Phase 6's scope (a separate concern to raise, since the next regeneration will
overwrite the block regardless). Add a `keep-divergent` verdict class or a stated exception; do not
leave "re-sync" as the standing instruction.

---

### Actionable non-HIGH concerns

- **MEDIUM 1 — BENCH-03's `closed-by-ci-run` token is accepted with no URL or run validation.**
  06-04 Task 4's check (`06-04-PLAN.md:454`) counts the token and requires exactly one. The
  acceptance text names `<url>` (`:459`) but nothing asserts a non-empty URL, a matching workflow, a
  matching ref, or `conclusion == success`. A mis-selected closed option passes the machine check.
  *Plan change:* on the closed branch, require a non-empty `https://github.com/.../actions/runs/<id>`
  match and record `gh run view <id> --json conclusion,event,headBranch,url` output beneath the token.
- **MEDIUM 2 — `TestBenchReblessJobShape` does not cover rebless's run bodies.** The fixture
  (`06-02-PLAN.md:173-184`) covers `runs-on`, `env`, `if`, ordered step names, the upload `with:` map
  and the absence of `permissions:` — but not the three load-bearing `run:` bodies in that job
  (`bench.yml` ~194, ~220, ~276), which include the `-rebless` invocation the D-13/D-01 exception exists
  to contain. A body could change while the fixture stays green, so the `must_haves` claim "The `rebless`
  job's block ... unchanged by this edit" (`06-02-PLAN.md:29`) exceeds the instrument. The literal-fixture
  choice over a before/after hash is defensible; its *coverage* is not yet sufficient.
  *Plan change:* extend the fixture to normalised run bodies, or hash the canonical serialisation of the
  whole parsed `jobs.rebless` subtree as the fixture value.
- **MEDIUM 3 — 06-02's TDD RED does not discriminate publish-job scoping.** Task 1 is `tdd="true"` and
  requires the verbatim RED to be pasted (`06-02-PLAN.md:201-207`), but the RED is produced by
  `jobs.publish` being absent — a single missing-anchor failure that reddens every assertion at once.
  It is therefore no evidence at all about the property cycle-1 HIGH 1/2/3 turned on: whether each
  individual assertion is scoped to the publish job rather than reading the file globally. 06-03 holds
  itself to a stricter standard (a mutation rehearsal) for exactly this reason.
  *Plan change:* after GREEN, perturb one publish-job property in a scratch copy (e.g. delete the
  publish job's `if-no-files-found`, leaving rebless's intact) and record that the named assertion
  reddens — one discriminating observation, not a second protocol.
- **MEDIUM 4 — 06-06 Task 3's executed-set extraction uses a fixed `rg -A400` window**
  (`06-06-PLAN.md:312`). It fails in both directions: a long execution table is truncated (false set
  mismatch), and a short one spills into `## Completeness and accepted gaps`, whose rows can also match
  `^\| \`?[a-z0-9]{6,}\`? \|` (false extra ids). The set-equality check is the mechanism that closes
  cycle-1's `06-06:248`, so its extractor should not be the weak link.
  *Plan change:* bound the section at the next `^## ` heading (e.g. `awk '/^## Supersedes executed/{f=1;next}/^## /{f=0}f'`).
- **LOW 1 — the publish job's action check is a SET, so duplicates survive** (`06-02-PLAN.md:164`). An
  extra checkout or setup-go step passes. Not central to D-06; assert an ordered list or per-action counts.
- **LOW 2 — the `both` dispatch option is not in the identifier family.** `bench.yml:73` keeps `both`
  as a choice and the current job's `if:` at `:99` tests `inputs.job == 'both'`. 06-02's identifier list
  (`:244-247`) names only the map key, the `default:`/`options:` entry and the `if:` conditional, and the
  shape test asserts only "the scheduled-event term and the dispatch term naming this job id"
  (`:146-147`). Drop `|| inputs.job == 'both'` and the `both` option silently becomes rebless-only —
  the exact dead-dispatch-option failure `key_links` (`:38`) warns about.
  *Plan change:* add `both` to the identifier family and to the `if:` assertion.
- **LOW 3 — 06-02's HEAD-relative fixture guard has a residual blind window.** `FIXDIFF` +
  `FIXPRESENT >= 10` (`06-02-PLAN.md:321`) catches a *gutted* fixture but not a *modified* one if the
  executor commits Task 2's edit before running the gate. The exemption's rationale is sound; the
  companion is one-sided.
  *Plan change:* make the companion an exact entry-count equality against the pre-plan SHA extraction,
  or state that the gate must run pre-commit.
- **LOW 4 — `rg -c '^## ' docs/BENCHMARKS.md` cannot show which sections exist** (`06-04-PLAN.md:309`).
  A heading count is consistent with any three headings. Assert the three heading texts instead.
- **LOW 5 — line-reference drift in 06-02.** `:103` cites `internal/bench/regression.go:72` for the
  cross-runner refusal (correct), while `:224` cites `:53` for the same claim — `:53` is the *platform*
  mismatch message. `:216` cites the Node setup step at 117-120; it is at 118-121. Cosmetic, but these
  are the `read_first` coordinates an executor navigates by.

### Divergent Views

Single-reviewer cycle; divergence is between Codex and the orchestrating reviewer's own verification:

- Codex raised MEDIUM "`.planning/research/STACK.md` must be added to 06-05's `files_modified` because
  the parity phrase is live at STACK.md:93". **Not upheld** — `rg -i parity .planning/research/STACK.md`
  returns no match; STACK.md:93 is a different document's line. The underlying block *is* divergent, but
  in a far more serious way (HIGH 3 above).
- Codex rated the class-fix-A exemptions "conceptually valid" but flagged 06-03 Task 2 as mixing the
  anchor with an inexpressible exception. Confirmed independently and promoted to HIGH 2.

### Cycle-1 status roll-up

| Cycle-1 HIGH | Cycle-2 status |
|---|---|
| HIGH 1 — D-06.1 never asserted | **RESOLVED** — exact `runs-on` + `CODEGRAPH_BENCH_RUNNER` on the publish subtree |
| HIGH 2 — D-06.4 pre-satisfied workflow-wide | **RESOLVED** — upload `with:` map and step-summary assertion both publish-scoped |
| HIGH 3 — D-06.3 rested on a comment | **RESOLVED** — all publish-job run bodies inspected structurally |
| HIGH 4 — committed baseline never loaded | **RESOLVED** — loads the committed path; non-vacuity demonstrated by the shared mutation |
| HIGH 5 — BENCH-03 closable in silence | **RESOLVED in design**; residual mechanical gap tracked as MEDIUM 1 |

## Risk Assessment — Cycle 2

**MEDIUM.** Architecture, wave ordering and instrument design are sound and materially better than
cycle 1. Two acceptance criteria will halt a correct execution (HIGH 1, HIGH 2) and one instruction
will damage `.claude/CLAUDE.md` if followed literally (HIGH 3). All three are localised text fixes in
two plans; none requires re-opening a locked decision. With them corrected the set drops to
**LOW-MEDIUM**.

---

# Cycle 1 — audit history (superseded by Cycles 2, 3 and 4 above)

## Codex Review

# Cross-AI Plan Review

## Summary

The plans are unusually thorough and generally trace the real implementation accurately. Their wave ordering is sound, the comparison-runner subtraction is well scoped, and the mutation/census protocols reflect known repository failure modes. The main weaknesses are verification strength rather than implementation intent: 06-02 does not mechanically enforce all four locked D-06 properties; 06-03 proves the pure `CheckRegression` function but does not prove the committed `baseline.json` participates; and 06-04 permits BENCH-03 to close after a local fallback without a successful workflow run. Several negative-only `git diff` and `rg` criteria also remain vacuous-pass-shaped under rule `84d1gfpywd`.

Overall risk: **MEDIUM**. The plans are likely to produce the intended code, but their acceptance gates can certify incomplete BENCH-02/BENCH-03 outcomes.

---

## Plan 06-01 — Benchmark runner de-coupling

### Strengths

- The subtraction points match the current code precisely: the old mode dispatch, flag, resolver, and two-subject loop are all real surfaces at [tools/bench/runner/main.go:156](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:156), [tools/bench/runner/main.go:161](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:161), [tools/bench/runner/main.go:178](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:178), and [tools/bench/runner/main.go:306](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main.go:306).

- The paired test edits are correctly identified. Existing tests directly exercise `-ts-binary` and `resolveTSBinary` at [tools/bench/runner/main_test.go:482](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:482) and [tools/bench/runner/main_test.go:494](/Volumes/Code/github.com/seanb4t/codegraph-go/tools/bench/runner/main_test.go:494).

- The end-to-end assertion is strongly positive: exactly two records, exact subject value, positive throughput, and `median_of_trials == 1`. This avoids an empty-output pass.

- The surviving-pin check has independent support from the proposed exact-entry test, so the negative diff check is not the only oracle.

### Concerns

- **MEDIUM — The verification introduces an undeclared Node dependency.** The plan removes Node from the benchmark architecture but parses smoke-test JSON with `node -e` at [06-01-PLAN.md:246](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:246). A machine with Go and git but no Node fails despite the implemented Go path being correct.

- **LOW — “No `t.Skip`/`//nolint` added” is negative-only and underspecified.** At [06-01-PLAN.md:257](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:257), there is no companion test-count or file-set assertion specifically proving the test files still exist. The named positive test assertions mitigate this for the three critical tests, but not for the entire package.

- **LOW — `git diff --quiet -- NOTICE` is vacuous-pass-shaped.** At [06-01-PLAN.md:311](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PLAN.md:311), it passes if `NOTICE` is absent from both compared states or if the check is run after an unintended commit. It does not prove the file exists or retains its expected legal content.

### Suggestions

- Replace Node JSON checks with a small Go verifier or an existing Go test.
- Add `test -f NOTICE` plus a stable positive assertion for its MIT copyright line.
- Record before/after `go test -list` output or exact expected test names rather than relying on the `t.Skip` absence check alone.

### Risk Assessment

**LOW–MEDIUM.** The implementation path is solid; most risk lies in portable verification.

---

## Plan 06-02 — Workflow publisher and baseline prose

### Strengths

- The plan correctly identifies the current comparison setup at [bench.yml:96](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:96), Node setup at [bench.yml:117](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:117), npm installation at [bench.yml:122](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:122), and dual-binary invocation at [bench.yml:138](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:138).

- The job-id, dispatch option, and conditional are treated as one identifier family. The current three linked sites are visible at [bench.yml:69](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:69), [bench.yml:71](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:71), and [bench.yml:99](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:99).

- Preserving action SHA pins is appropriate. The existing actions are fully pinned, for example [bench.yml:104](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:104) and [bench.yml:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:154).

- The plan correctly recognizes that `TestWorkflowRunBodiesInvokeTask` does not protect `bench.yml`; the exclusion is explicit at [internal/upgrade/taskfile_shape_test.go:99](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:99).

### Concerns

- **HIGH — D-06.1 is stated but not positively asserted.** The plan requires exact `runs-on` and `CODEGRAPH_BENCH_RUNNER` values, but its automated verification at [06-02-PLAN.md:154](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:154) checks neither. A rewrite could move the publisher to `ubuntu-latest`, omit the environment variable, and still pass every automated criterion. The current load-bearing values are at [bench.yml:98](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:98) and [bench.yml:101](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:101).

- **HIGH — D-06.4 is checked workflow-wide rather than publish-job-wide.** `rg -c 'if-no-files-found: error' >= 1` at [06-02-PLAN.md:162](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:162) passes if the publisher drops its upload assertion because another job already contains the same setting at [bench.yml:274](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:274). There is also no automated positive assertion for `$GITHUB_STEP_SUMMARY`.

- **HIGH — D-06.3 is asserted mainly through prose.** The new job could call `CheckRegression`, invoke `-mode regression`, or add a shell threshold failure while retaining a “non-blocking” comment. The acceptance criteria only require that a line contain `-mode publish`; they do not structurally inspect all run bodies in the publish job.

- **MEDIUM — D-06.2 is incompletely enforced.** `git diff --quiet -- Taskfile.yml` proves this plan did not edit the Taskfile in the current diff, but not that a `bench:publish` target does not already exist or arrive through another concurrent change. The workflow test deliberately excludes all of `bench.yml` at [internal/upgrade/taskfile_shape_test.go:100](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:100).

- **MEDIUM — The SHA equality check is vacuous when all `uses:` lines disappear.** At [06-02-PLAN.md:163](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:163), `0 == 0` passes. Actionlint does not require a checkout, Go setup, cache, or artifact-upload action.

- **LOW — The `rebless` diff criterion is ambiguous.** “No matches beyond comment-prose lines explicitly listed in the SUMMARY” at [06-02-PLAN.md:165](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-02-PLAN.md:165) is not mechanically checkable and does not prove the job block is byte-identical.

### Suggestions

Add a structural YAML test that parses the `publish` job and asserts:

- exact `runs-on`;
- exact matching `CODEGRAPH_BENCH_RUNNER`;
- one inline `go run ./tools/bench/runner -mode publish`;
- no `task bench:*`, `CheckRegression`, `-mode regression`, `-rebless`, Node, npm, or alternate binary;
- a summary step containing `$GITHUB_STEP_SUMMARY`;
- an upload step for `publish-results.json` with `if-no-files-found: error`;
- expected action set/count, not merely pin-format equality.

Compare the `rebless` job’s parsed subtree before and after rather than grepping diff text.

### Risk Assessment

**HIGH.** This is the largest verification gap: all four locked D-06 properties are described, but two are weakly checked and two are not mechanically checked at all.

---

## Plan 06-03 — Regression-gate proof

### Strengths

- The selected mutations target real, isolated tolerance constants at [internal/bench/regression.go:9](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:9) and [internal/bench/regression.go:15](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression.go:15).

- Existing tests provide exact non-vacuous oracles: the 11% throughput case at [internal/bench/regression_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:43) and the 16% RSS case at [internal/bench/regression_test.go:62](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:62).

- The plan correctly rejects category-error failures. `CheckRegression` checks platform/runner identity before numeric tolerances, so requiring the RED to name throughput or RSS is important.

- The clean-tree precondition, `EXIT` trap, confirmed mutation diff, RED, revert, and green rerun are a strong application of the repository’s mutation protocol.

### Concerns

- **HIGH — This does not prove “against the committed `baseline.json`.”** The tests construct in-memory `Metrics` values and call `CheckRegression` directly at [internal/bench/regression_test.go:502](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/bench/regression_test.go:502). The committed baseline is merely asserted untouched at [06-03-PLAN.md:229](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-03-PLAN.md:229); it is never loaded or passed to the function. This proves the tolerance branches remain live, but not the complete BENCH-02 mechanism named in the requirement.

- **MEDIUM — The oracle-integrity check is diff-relative and can pass after an unintended prior commit.** `git diff --quiet -- internal/bench/regression_test.go` only proves the file is currently clean relative to HEAD. It does not prove the test table is byte-identical to the pre-plan version. The before/after subtest count catches deletion/addition counts but not changed values or expected outcomes.

- **LOW — “Comment-only” does not guarantee semantic comments were preserved.** The diff-shape check is useful, but a complete deletion/replacement of all comments would pass if every changed line still starts with `//`. The positive retained-`comparison` count partly mitigates this.

### Suggestions

- Add a small test or command that reads `tools/bench/baseline.json`, unmarshals it into `bench.Metrics`, derives same-category current records with 11% lower throughput and 16% higher RSS, then calls `CheckRegression` and positively asserts the expected error categories.
- Hash `regression_test.go` before Task 1 and compare the hash after Task 2, or explicitly assert the values/names of the two critical cases.
- Keep the mutation rehearsal; it remains valuable even after adding the committed-baseline integration check.

### Risk Assessment

**MEDIUM–HIGH.** The pure function is convincingly proven, but the plan overstates that proof as covering the committed baseline.

---

## Plan 06-04 — Measurement, documentation, and census

### Strengths

- The task order is explicit and genuinely prevents document-first execution. Task 2 has a hard precondition that the capture exists and contains two positive records at [06-04-PLAN.md:151](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:151). Within normal GSD sequential task execution, numbers cannot be published before Task 1 produces the artifact.

- The capture validation is strongly positive: exact record count, exact repo set, exact subject, and positive values at [06-04-PLAN.md:135](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:135).

- The proposed document content maps well to the existing methodology. The source currently contains median-of-5 at [docs/BENCHMARKS.md:20](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:20), a comparison target at [docs/BENCHMARKS.md:50](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:50), and the regression tolerances at [docs/BENCHMARKS.md:261](/Volumes/Code/github.com/seanb4t/codegraph-go/docs/BENCHMARKS.md:261).

- The census includes a positive control, a multiline-vs-line-based comparison, a scanned-file floor, and positive required-content checks. This is substantially better than a negative vocabulary scan alone.

### Concerns

- **HIGH — BENCH-03 can remain unverified while the plan still completes.** The plan explicitly allows a local fallback at [06-04-PLAN.md:118](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:118), while the live workflow verification applies only “if the preferred CI path was used” at [06-04-PLAN.md:136](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:136). BENCH-03 says `bench.yml` runs and publishes; a local runner invocation does not establish that.

- **MEDIUM — The byte-identical-to-stdout assertion does not establish byte identity.** At [06-04-PLAN.md:142](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PLAN.md:142), checking that the indexed file’s first line starts with `[` plus a SUMMARY statement does not compare it to an independently retained stdout capture. It also depends on the file having been staged.

- **MEDIUM — Numeric traceability is only a spot check of `files_per_sec`.** The plan publishes five metrics per corpus but only requires spot-checking `files_per_sec`. Query latency, peak RSS, bytes/sec, and cold start could be transcribed incorrectly without failing.

- **MEDIUM — Local measurements may be unsuitable as published project figures.** The fallback records an empty runner label but does not impose stable runner conditions. This does not contradict D-02, but it weakens reproducibility and makes the published figures potentially machine-specific.

- **LOW — The file-floor assertion can still admit a shifted surface.** `>= 12` proves something was scanned, but not the exact expected file set. A deleted critical file and unrelated added files could preserve the count.

### Suggestions

- Make successful `bench.yml` dispatch a blocking BENCH-03 acceptance criterion. If unavailable, allow BENCH-01 documentation work to proceed but leave BENCH-03 explicitly open.
- Tee stdout simultaneously to a temporary file and the committed artifact, then compare SHA-256 values.
- Generate the Markdown data rows mechanically from JSON or verify all ten metric cells against parsed JSON.
- Record an exact sorted census file list and compare it to the expected set, rather than using only a floor.

### Risk Assessment

**MEDIUM–HIGH.** BENCH-01 ordering is sound, but BENCH-03 can be marked complete without its required workflow execution.

---

## Plan 06-05 — Agent-facing files

### Strengths

- Enumerate-before-edit is excellent for preventing post-hoc classification.
- Source-before-mirror addresses a real current inconsistency: `.claude/CLAUDE.md` still carries the old compatibility framing at [CLAUDE.md:15](/Volumes/Code/github.com/seanb4t/codegraph-go/.claude/CLAUDE.md:15), while PROJECT declares it retired at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- The stale capability defect is real: STATE still promises migration at [STATE.md:26](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/STATE.md:26), and PROJECT’s constraint still names `codegraph migrate` at [PROJECT.md:244](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md:244).
- Region-scoping the PROJECT check is correct because historical mentions legitimately survive elsewhere.

### Concerns

- **MEDIUM — The STATE assertion checks only two exact phrases, not the resulting core-value equality.** At [06-05-PLAN.md:214](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:214), STATE can retain different stale framing and pass. The task says it must match PROJECT’s Core Value, but no automated equality check enforces that.

- **MEDIUM — Marker count is not byte-integrity.** `GSD_MARKERS=14` at [06-05-PLAN.md:221](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:221) allows marker names, order, or source attributes to change while retaining the count.

- **LOW — Fresh-session verification is optional/manual in practice.** The human check at [06-05-PLAN.md:215](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:215) is the closest direct test of MEM-02’s file half, but the completion criteria do not clearly block on it.

- **LOW — Negative `git diff --quiet` checks remain context-dependent.** The protection for NOTICE/LICENSE/CHANGELOG at [06-05-PLAN.md:224](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PLAN.md:224) proves only no current uncommitted delta, not immutable content across task commits.

### Suggestions

- Parse both Core Value sections and assert exact normalized equality.
- Capture the exact ordered GSD marker lines before editing and compare them afterward.
- Make the fresh-session review an explicit blocking checkpoint or clearly retain MEM-02 as pending until performed.
- Compare protected-file hashes against the pre-plan commit rather than the current working-tree diff.

### Risk Assessment

**MEDIUM.** The editorial reasoning is strong; generated-file integrity and direct MEM-02 proof need firmer gates.

---

## Plan 06-06 — Engram memory sweep

### Strengths

- The fail-loud tool-availability precondition directly respects D-16.
- The plan correctly separates read-only enumeration, one-way classification approval, and durable writes.
- Pagination exhaustion, per-scope page counts, deterministic sorting, zero-record scope reporting, and verdict arithmetic are strong positive assertions.
- The blocking decision checkpoint is appropriate because superseding is one-way.
- The plan explicitly preserves historical clauses in mixed records and forbids substituting overwrite/delete for supersede.

### Concerns

- **MEDIUM — A returned identifier proves write acceptance, not supersede semantics.** The plan acknowledges this at [06-06-PLAN.md:86](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:86) and [06-06-PLAN.md:228](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:228). This is the locked D-15 trade-off, so it is not a defect to reopen, but residual risk remains material.

- **MEDIUM — The known-minimum-three-rules assertion may conflate memory records with rule-list entries.** The plan requires a rule-scope record count of at least three at [06-06-PLAN.md:160](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:160), but research explicitly says the live tool surface is unverified. If rules are exposed through a distinct endpoint or pagination shape, this condition could falsely report a truncated enumeration.

- **MEDIUM — The automated closure check does not reconcile executed entries with approved IDs.** It checks section presence and absence of “delete/overwrite” verdicts, but not set equality between approved `supersede` short IDs and executed rows. The prose acceptance criterion requires equality; the automated command at [06-06-PLAN.md:248](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:248) does not enforce it.

- **LOW — MEM-02 remains assertion-based by locked decision.** The plan properly calls this out at [06-06-PLAN.md:82](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/06-benchmark-de-coupling-memory-sweep/06-06-PLAN.md:82). This is an accepted evidence limitation, not a planning contradiction.

### Suggestions

- After live tool discovery, distinguish rule objects from ordinary memory records and apply the “at least three” assertion to the correct population.
- Add an exact set comparison: approved supersede `short_id`s must equal executed-original `short_id`s, with no duplicates.
- Record the supersede operation name and returned response shape for each write, not only an identifier.
- Clearly mark MEM-02 as “accepted by D-15 evidence standard” rather than “demonstrated,” unless the optional fresh-session check is actually performed.

### Risk Assessment

**MEDIUM.** The one-way operation is handled carefully, but external-tool uncertainty and the deliberately accepted lack of re-query evidence prevent a low-risk rating.

---

## Cross-plan dependency assessment

The wave graph is coherent:

- 06-01 and 06-03 can run independently with no declared file overlap.
- 06-02 correctly depends on the new `publish` runner mode.
- 06-04 correctly depends on all benchmark-side plans.
- 06-05 follows the completed in-tree state.
- 06-06 follows the file sweep and places durable external writes behind a checkpoint.

The principal dependency issue is semantic rather than scheduling: **06-04’s local fallback allows BENCH-03 to remain unproven even though later plans can proceed and the phase may appear complete.**

## Final recommendation

Approve after revisions to three gates:

1. Add a structural, publish-job-scoped verifier for all four D-06 properties in 06-02.
2. Add a committed-`baseline.json` integration assertion to 06-03.
3. Make a successful `bench.yml` publish run mandatory for BENCH-03, while allowing local measurement only as a BENCH-01 fallback.

Those changes would reduce overall execution risk from **MEDIUM** to **LOW–MEDIUM** without reopening any locked decision.

---

## Consensus Summary

Only one external reviewer lane was requested (`--codex`), so there is no multi-reviewer
consensus to synthesize. What follows is the orchestrator's own verification of Codex's
findings against the repository — every HIGH below was independently re-checked against the
cited file before being carried forward.

Codex's verdict: **approve after revisions**, overall risk MEDIUM. The plans are judged
unusually thorough and their wave ordering sound; the weaknesses are concentrated in
*verification strength*, not implementation intent. Three gates would have to change to close
the review: a publish-job-scoped structural verifier for all four D-06 properties (06-02), a
committed-`baseline.json` integration assertion (06-03), and a mandatory successful `bench.yml`
run for BENCH-03 (06-04).

### Strengths (single reviewer, source-verified)

- Subtraction points in 06-01 match the live code exactly (`tools/bench/runner/main.go:156,161,178,306`), and the paired test edits at `main_test.go:482,494` are correctly identified.
- 06-03's mutation targets are real isolated constants (`internal/bench/regression.go:9,15`) with exact pre-existing oracles (`internal/bench/regression_test.go:43,62`), and the plan correctly rejects category-error REDs.
- 06-04 Task 2 carries a hard `<precondition>` that `06-PUBLISH-RESULTS.json` exists with two positive records (`06-04-PLAN.md:151`) — **the ordering question raised for this review is answered affirmatively**: under normal GSD sequential task execution the measurement provably lands before the document that publishes it.
- 06-06 correctly separates read-only enumeration, a blocking one-way classification checkpoint, and durable writes, and respects D-16's fail-loud tool-availability precondition.

### Concerns — HIGH (all five independently re-verified)

1. **06-02 / D-06.1 is never asserted.** The `<automated>` block at `06-02-PLAN.md:154` and every acceptance criterion at `:157-166` check the job key, the conditional, the retired-term census, the SHA pins, and three `git diff` negatives — but **nothing** checks `runs-on: namespace-profile-linux-amd64-4x8` or the `CODEGRAPH_BENCH_RUNNER` env var. Confirmed: those are the load-bearing values at `.github/workflows/bench.yml:98,101`, and D-06.1 calls runner identity "part of the measurement's validity, not cosmetics" (`06-CONTEXT.md:78-82`). A from-scratch rewrite onto `ubuntu-latest` with no env var passes every automated criterion.
2. **06-02 / D-06.4 is checked workflow-wide, and is already satisfied elsewhere.** `test "$(rg -c 'if-no-files-found: error' .github/workflows/bench.yml)" -ge 1` (`06-02-PLAN.md:154,162`) is satisfied by the pre-existing `rebless` job upload at `.github/workflows/bench.yml:274`. Confirmed by reading that line. The publish job can therefore omit its own upload assertion and still pass. Separately, D-06.4 also requires a `GITHUB_STEP_SUMMARY` block (`06-02-PLAN.md:26,132`) and **no criterion asserts `$GITHUB_STEP_SUMMARY` at all**.
3. **06-02 / D-06.3 (non-blocking publish-not-gate) rests on prose.** The contract is restated only in a run-step comment (`06-02-PLAN.md:128`). No criterion structurally inspects the publish job's run bodies, so a rewrite that calls `CheckRegression`, invokes `-mode regression`, or adds a shell threshold failure — while keeping the "non-blocking" comment — passes.
4. **06-03 does not prove the gate fires "against the committed `baseline.json`."** Roadmap success criterion 2 names the committed baseline explicitly. The rehearsal drives `CheckRegression` through in-memory `Metrics` values; `06-03-PLAN.md:229` only asserts `git diff --quiet -- tools/bench/baseline.json`, i.e. that the file was *not touched*. Verified by search: **no Go test loads `tools/bench/baseline.json`** — the only test references (`tools/bench/runner/main_test.go:318,731`) write their own baseline into a temp dir. The tolerance branches are proven live; the committed-data path is not.
5. **06-04 lets BENCH-03 close without ever running the workflow.** The plan sanctions a local fallback at `06-04-PLAN.md:118-124`, and BENCH-03's only live verification is a `<human-check>` explicitly scoped "If the preferred CI path was used" (`06-04-PLAN.md:136`). Verified verbatim. Taking the fallback leaves BENCH-03 — "a `bench.yml` run publishes the absolute numbers without invoking another implementation anywhere in the job" — with **zero** verification, while the plan still completes.

### Concerns — MEDIUM / LOW

Per-plan detail is in the Codex section above. Two cross-cutting shapes the orchestrator adds
after censusing all six plans (28 occurrences of `git diff --quiet` / "returns no matches"):

- **`git diff --quiet -- <path>` is HEAD-relative, not baseline-relative.** Every use of it as a
  tamper guard (NOTICE, LICENSE, CHANGELOG.md, `tools/bench/baseline.json`, `internal/bench/`,
  the 06-05 target files) proves only that the *working tree* currently matches HEAD. Because
  GSD commits per task, any change made and committed by an earlier task in the same plan is
  invisible to a later task's guard. To hold across tasks these must compare against the
  pre-plan commit (`git diff --quiet <pre-plan-sha> -- <path>`) or a recorded hash.
- **`rg -c '<pat>'` used as a "returns no matches" assertion is an inverted gate.** `rg -c`
  exits 1 and prints nothing on the clean path, so any executor chaining it with `&&` sees the
  *good* outcome as a failure — and any executor reading exit status leniently sees a broken
  command as a pass. This repo already has the inverted-`rg -qv` gate in its documented
  failure history. Occurrences: `06-01-PLAN.md:255`, `06-02-PLAN.md:164,165,215`,
  `06-03-PLAN.md:158,228`. Restate each as `test "$(… | rg -c … || true)" = "0"` with a printed
  positive token, paired with a count that must be > 0.

### Divergent Views

None — single reviewer.

### Locked-decision contacts (reported, not raised as defects)

- 06-06's "a returned identifier proves write acceptance, not supersede semantics" is the
  **locked D-15 trade-off**, acknowledged in-plan at `06-06-PLAN.md:86,228`. Reported as
  residual risk, not a defect.
- MEM-02 remaining assertion-based rather than demonstrated is likewise accepted by D-15 and
  called out at `06-06-PLAN.md:82`.
- The local-measurement fallback in 06-04 does not contradict D-02; the HIGH above is about
  BENCH-03's *verification*, not about re-opening the fallback decision.

---

## Verification coverage

The orchestrator independently re-read the following before accepting Codex's findings. Every
HIGH was confirmed at its cited line; none were taken on the reviewer's word.

| Claim | Source checked | Verdict |
|---|---|---|
| D-06.1 unasserted | `06-02-PLAN.md:140-166`, `.github/workflows/bench.yml:96-101` | CONFIRMED — no `runs-on` / `CODEGRAPH_BENCH_RUNNER` assertion exists |
| D-06.4 satisfied by another job | `.github/workflows/bench.yml:274` | CONFIRMED — `if-no-files-found: error` present in the `rebless` upload |
| D-06.4 step-summary unasserted | `06-02-PLAN.md:26,132,154-166` | CONFIRMED — requirement stated, never asserted |
| D-06.3 prose-only | `06-02-PLAN.md:128,157-166` | CONFIRMED |
| D-06 is a four-property locked list | `06-CONTEXT.md:77-90` | CONFIRMED |
| Committed baseline never loaded | `rg 'baseline\.json' --type go internal/ tools/`; `06-03-PLAN.md:221-231` | CONFIRMED — only temp-dir baselines in tests |
| BENCH-03 human-check is conditional | `06-04-PLAN.md:118-124,136` | CONFIRMED verbatim |
| 06-04 ordering is guaranteed | `06-04-PLAN.md:151` `<precondition>` | CONFIRMED — Codex's *strength* finding is correct; the measurement provably precedes the document |
| Negative-guard census | 28 hits across all six `*-PLAN.md` | CONFIRMED — enumerated above |

Codex ran with repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present.

---

## Verification coverage — Cycle 2

Codex ran with full repo access and produced `file:line`-grounded findings throughout; no
`[reviewed-without-repo-access]` marker was present. The orchestrating reviewer independently
re-executed or re-read every load-bearing claim below against the working tree at `6a2816b`
before accepting it. One Codex finding was **not upheld**.

| Claim | Source / command actually run | Verdict |
|---|---|---|
| SHA-pin arithmetic (expected 19) | `rg -o '^\s*uses: ' / '^\s*uses: \./' / 'uses: [^@[:space:]]+@[0-9a-f]{40}' .github/workflows/bench.yml` → 22 / 2 / 20; `bench.yml:118` is the deleted pinned action | CONFIRMED — 19 pinned + 2 local = 21 total after the edit |
| Publish-job scoping is real | `06-02-PLAN.md:41,140-184,266` | CONFIRMED — every D-06 assertion anchored to the `jobs.publish` subtree |
| `rebless` upload would pre-satisfy a file-wide check | `.github/workflows/bench.yml:270-274` | CONFIRMED present — and now provably out of scope |
| Committed baseline genuinely loaded | `06-03-PLAN.md:236-240` (`../../tools/bench/baseline.json`), `tools/bench/baseline.json` (populated `goos`/`goarch`/`runner`/`scratch_fs`) | CONFIRMED — real file, category guards stay quiet for a frame-derived record |
| Mutation reddens both oracles | `internal/bench/regression.go:9,15,114,124`; `internal/bench/regression_test.go:42-69,512` | CONFIRMED for the RED itself — **but** the emitted text is `CheckRegression() = nil, want error` (→ HIGH 1) |
| Task 2 pre-plan diff is impossible | `06-03-PLAN.md:140,166-192,296` | CONFIRMED — Task 1 commits all four named files |
| `\bcomparison\b >= 5` floor survives the sweep | `rg -n '\bcomparison\b' tools/bench/BASELINE.md` → lines 39/87/97/120/297; swept lines are 64 and 241 | CONFIRMED — disjoint; floor holds with zero headroom |
| `internal/bench/regression.go` comparison floor | `rg -c` → 7 (floor 5) | CONFIRMED |
| Taskfile exact-set + floor | `BENCHTASKS=bench:regression:,`, `ALLTASKS=47` (floor 40) | CONFIRMED |
| Fixture-entry floor | `rg -o '^\s*\{Workflow:' internal/upgrade/taskfile_shape_test.go` → 10 (floor 10) | CONFIRMED — exactly at the floor |
| GSD marker count | `rg -o '<!-- GSD:[^>]*-->' .claude/CLAUDE.md` → 14 | CONFIRMED |
| Core-value extraction executes | 06-05's `PV`/`SV` commands run verbatim → `PV_LEN=208` (floor 60), `SV` non-empty | CONFIRMED — gate is satisfiable and non-vacuous |
| Compatibility bullet region-scope | `rg '^\- \*\*Compatibility\*\*' .planning/PROJECT.md` matches `:244`; `codegraph migrate` in it = 1, in the file = 3 | CONFIRMED — file-wide ban would indeed be unsatisfiable |
| Census surface + critical files | `rg -U --files <surface>` → 26 files today, 20 after 06-01 deletes the six captures; all 13 critical files present | CONFIRMED — floor 12 and `13/13` both hold |
| Phase-6 census is reachable | Ran the full cycle-2 pattern set → 168 hits today, every one inside a declared sweep target | CONFIRMED — a zero is attainable |
| Surviving corpus pins | `rg -o 'f89ae3ea…|dbdc1acb…' tools/bench/realcorpus/manifest.go` → 2; manifest has 3 entries today | CONFIRMED |
| Codex: `.planning/research/STACK.md` needs editing | `rg -i parity .planning/research/STACK.md` → **no match** | **NOT UPHELD** — false finding; the real defect is the wholesale block divergence (HIGH 3) |
| Stack block divergence | `diff` of `.claude/CLAUDE.md` lines 22-140 vs `.planning/research/STACK.md` | CONFIRMED — different documents; `STACK.md:151` contains `drop-in`, which would break `CLAUDE_MD_FRAMING_TOTAL=0` |
| Three HEAD-relative exemptions | `06-05-PLAN.md:101`, `06-02-PLAN.md:321`, `06-03-PLAN.md:402-407` | CONFIRMED correct (one residual caveat → LOW 3) |
| No sound `rg -c` floor over-corrected | Enumerated all 24 `rg -c` occurrences across the six plans | CONFIRMED — every remaining use is a floor, a line-oriented count, or a `|| true`-guarded restatement |
