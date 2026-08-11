---
phase: 4
cycle: 5
reviewers: [codex, opencode]
reviewers_unavailable: []
reviewed_at: 2026-08-11T16:12:44Z
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md, 04-04-PLAN.md, 04-05-PLAN.md, 04-06-PLAN.md]
plans_revision: 28ee153
source_grounding: true
drift_guard_authority: intel
prior_cycle: cycle-3 REVIEWS.md text retained in git history (cycle 4's review died on a session limit and produced no document)
---

# Cross-AI Plan Review — Phase 4 (Cycle 5, final maintainer-authorized cycle)

Reviewers were given the cycle-5 mandate — verify each of the four claimed cycle-5 fixes **by
measurement**, adjudicate the `04-05` pending-todo count drift, confirm the threshold inventory
moved only by addition, confirm the do-not-regress list, and hunt the standing list for newly
introduced defects — plus the `04-CONTEXT.md` supersession ground rules (D-01 Caskroom-not-Cellar,
D-08 seam assertion, D-09 `--check` step-aside, D-04R linuxbrew both shapes) and repo rule
`84d1gfpywd`. No lane raised a stale-ROADMAP finding, so the ground rules held.

Only `04-06-PLAN.md` and `04-VALIDATION.md` changed this cycle (`git diff 983d624 28ee153` names
exactly those two files); `04-01`…`04-05` are byte-unchanged, so their floors cannot have regressed
by construction.

## Codex Review

### 1. Summary

Cycle-5’s four targeted fixes are correctly present in both the plan and validation matrix. Direct measurement confirms the PASS regex remains precise, hash cardinality closes the missing-AFTER hole, the criterion loop matches an explicitly mandated document shape, and `TRAP_ARMED` is scoped honestly.

One execution-blocking pre-existing defect remains: plan 04-05 asserts a global pending-todo count of 7, but the live count is 10. This should be fixed in the plan before execution, not reconciled ad hoc.

Overall verdict: **NEEDS ONE PLAN FIX**, then ready to execute.

### 2. Strengths

- Fix 1 is correct in both locations: [04-06-PLAN.md:430](04-06-PLAN.md:430) and [04-VALIDATION.md:56](04-VALIDATION.md:56). Against three sample lines—exact name, `Extra`, and a subtest—the corrected regex matched exactly one. The duration suffix is required while the full anchoring rejects lookalikes.

- Fix 2 is additive as required. The `-eq 2` cardinality check precedes the retained `sort -u … -eq 1` equality check at [04-06-PLAN.md:430](04-06-PLAN.md:430), mirrored at [04-VALIDATION.md:56](04-VALIDATION.md:56). Measurement with only a valid BEFORE line yielded `cardinality=1` but `distinct=1`: the old leg passes, while the new combined gate correctly fails.

- Fix 3’s producer and consumer agree exactly. Task 2 mandates literal labels `Criterion 1` through `Criterion 4` at [04-06-PLAN.md:605](04-06-PLAN.md:605); the per-token fixed-string loop enforces them at [04-06-PLAN.md:628](04-06-PLAN.md:628) and [04-VALIDATION.md:57](04-VALIDATION.md:57). This avoids creating another unsatisfiable documentation gate.

- Fix 4 is an honest split of responsibility. The action makes placement a hard constraint—append after EXIT-trap registration and before `brew tap`—and explicitly rejects moving it at [04-06-PLAN.md:330](04-06-PLAN.md:330). The automated assertion proves that the arming step completed; the plan explicitly says it does not prove textual ordering, retaining transcript inspection for that at [04-06-PLAN.md:438](04-06-PLAN.md:438). The marker is not being asked to prove more than it can.

- Structural checks hold: six plans, fifteen tasks, fifteen validation rows, six threat models, six artifact sections, and 15/15 `read_first` and `acceptance_criteria` blocks. Anchored `<automated>` open/close counts match independently in every plan: 3/3, 3/3, 2/2, 2/2, 3/3, 2/2.

- The cycle-5 commit diff confirms the operational changes are additive: the old PASS pattern was replaced, while new cardinality, criterion-loop, and trap-marker assertions were added. No `-eq 0` gate changed. The plan records this explicitly at [04-06-PLAN.md:164](04-06-PLAN.md:164).

### 3. Concerns

- **HIGH — Plan 04-05 has a guaranteed unrelated failure.** Its automated gate requires the total pending-todo count to be exactly 7 at [04-05-PLAN.md:507](04-05-PLAN.md:507), mirrored at [04-VALIDATION.md:55](04-VALIDATION.md:55). The live directory contains 10 files. Plans 04-02 and 04-05 move three named todos, so unrelated additions or removals should not determine whether their work passes. This is not acceptable as execution-time reconciliation: the failure is already known, deterministic, and unrelated to the task’s deliverable.

  The same brittle global-count assumption also appears earlier as expected totals 8 and 7 at [04-02-PLAN.md:456](04-02-PLAN.md:456) and [04-05-PLAN.md:537](04-05-PLAN.md:537). It couples independent todo intake to phase execution.

- **LOW — Threshold inventory wording is narrower than raw textual counts.** Comparing cycle 4 to cycle 5 confirms no existing comparison operator was changed in the executable gates, but raw whole-file counts rise more than the reported 6→7/0→1/2→3 because the disposition and explanation prose repeats the expressions. This is not a functional defect; future inventory claims should explicitly say “executable gate instances” to avoid ambiguity.

No new standing-hunt violations were found in cycle-5 changes.

### 4. Suggestions

1. Replace global todo counts with named-set assertions:

   - Assert each folded todo is absent from `pending/`.
   - Assert each is present exactly once in `completed/`.
   - Preserve the existing rename/status check where useful.
   - Do not assert the total size of either directory.

2. Apply that correction consistently to both [04-05-PLAN.md](04-05-PLAN.md) and its mirrored [04-VALIDATION.md](04-VALIDATION.md). Consider removing the analogous total-count coupling from 04-02 as well.

3. Keep the cycle-5 fixes unchanged; all four are mechanically sound.

### 5. Risk Assessment

**Current risk: HIGH for execution reliability, LOW for implementation design.**

The Homebrew detection, refusal semantics, restoration safety, and cycle-5 acceptance gates are unusually thorough and internally consistent. However, plan 04-05 will currently stop for an unrelated global todo count. Once that single brittle gate is converted to named artifact assertions, residual risk drops to **LOW–MEDIUM**, dominated by the intentionally manual real-tap acceptance run and its documented natural-release residual.

---

## OpenCode Review

### 1. Summary

All four cycle-5 fixes are **present and correctly implemented in both files**, verified by execution rather than by reading. I re-ran the critical patterns against fixture data. I found **one new LOW** (a residual hole in FIX 2's cardinality leg, demonstrated below), and I adjudicate the `todos/pending` drift as **a finding requiring a plan change** (MEDIUM). The TRAP_ARMED honesty split is sound. Threshold inventory moved by addition only, as reported. This plan set is ready to execute; neither finding blocks.

### 2. Cycle-5 fix verification (by measurement)

**FIX 1 — leg-1 PASS pattern: VERIFIED.** Present in `04-06-PLAN.md:430` and `04-VALIDATION.md:56`. I executed the corrected pattern against a three-line fixture (`_RealInstall (0.00s)`, `_RealInstallExtra (0.00s)`, indented `_RealInstall/sub (0.00s)`): it matches **exactly 1**. The `Extra` suffix is excluded because the pattern demands `<space>(duration)` immediately after the name; the subtest line is excluded by the `^` anchor (go test indents subtests). The accompanying acceptance criterion at `04-06-PLAN.md:450-456` records the measurement and explicitly forbids re-simplifying the suffix away. Precision retained.

**FIX 2 — hash cardinality: PRESENT, ADDITIVE, but with a residual hole (new LOW, below).** Both legs are in `04-06-PLAN.md:430` and `04-VALIDATION.md:56`, cardinality first, distinctness second, and a receipt carrying only a BEFORE hash now fails the `-eq 2` leg as intended.

**FIX 3 — criterion-mapping loop: VERIFIED, including the beyond-brief literal mandate.** The `for c in 1 2 3 4 … -F … "Criterion $c"` loop is in `04-06-PLAN.md:628` (executable, with terminating `</automated>`) and `04-VALIDATION.md:57`. Crucially, `04-06-PLAN.md:605` mandates the literal token with matching case: "**Label each of the four mappings with the literal token `Criterion N` — capital C, `Criterion 1` through `Criterion 4`**". The gate greps `"Criterion $c"` (capital C) with `-F`; the action prescribes exactly that literal. The FIX-1 failure mode (gate greps a literal nothing prescribes) is not recreated. The cycle-1 false-claim sites are annotated rather than silently edited.

**FIX 4 — TRAP_ARMED: VERIFIED, and the honesty split is honest.** The marker append is at `04-06-PLAN.md:329-331`, stated as a hard constraint: "**This append MUST stay in exactly this position: after the registration, before the first mutation. Moving it later… is a defect, not a refactor, and a future edit that relocates it must be rejected**." The gate (`-eq 1` on `^TRAP_ARMED=1$`) is in both files. **On the adjudication question:** the split is honest, and the plan's own framing is the correct one. The marker is a runtime artifact proving the arming *step executed* in a run that later produced a passing gate; it cannot prove textual ordering (a script could append the marker and never register a trap), and the plan says exactly that at `:161` and `:440-443`, leaving ordering to the acceptance criterion and `<human-check>`. The marker is not asked to carry more than it proves. One true limitation worth one sentence: `TRAP_ARMED=1` is emitted by the script under test, so it attests the script *claims* the trap was armed — but since the baseline's RUN_ID binding and the `test -s` artifact-existence loop tie it to this run, and the receipt independently proves cleanup actually ran (`RESTORE_INVOCATIONS=1`, hash equality), the evidentiary chain is closed from two directions. This is as strong as a self-reported marker can be, and the plan does not overclaim it.

### 3. Standing hunt list — new instances

**Automated-tag balance: clean.** Anchored counts per plan file: 04-01 3/3, 04-02 3/3, 04-03 2/2, 04-04 2/2, 04-05 3/3, 04-06 2/2. No unclosed `<automated>`.

No new instances of: exit-status-only `go test -run`, pre-satisfied floors (the new floors — TRAP_ARMED, hash cardinality, criterion loop — are all RED today: the artifacts they grep do not exist until the run produces them), multi-`-e` set-membership misuse, or unanchored-suffix patterns.

### 4. Concerns

**LOW — FIX 2's cardinality leg counts key-agnostic lines, so duplicate BEFORE lines satisfy it.** `04-06-PLAN.md:430`: `rg -c -e '^PAYLOAD_SHA256_(BEFORE|AFTER)=[0-9a-f]{64}$'` counts lines matching *either* key. I measured: a receipt containing two identical `PAYLOAD_SHA256_BEFORE=<hash>` lines and no AFTER line yields cardinality 2 **and** distinctness 1 — both legs pass with the after-hash never recorded. This is a narrower residue than the one C5-2 closed (it requires the receipt writer to duplicate a line, which step 3(c)'s fixed seven-line format makes unlikely), but it is the same species the cycle keeps finding: a combined pattern where per-token assertions are the established fix. Two `-eq 1` assertions — one on `^PAYLOAD_SHA256_BEFORE=`, one on `^PAYLOAD_SHA256_AFTER=` — close it for the price of one extra `test`. Given this is the final cycle, acceptable to ship as-is with the receipt's fixed seven-line shape as the practical guard; flagging so the hole is on record rather than discovered in cycle 6.

**MEDIUM — adjudication: the `todos/pending == 7` gate in 04-05 Task 3 requires a plan change.** `04-05-PLAN.md:507` and `04-VALIDATION.md:55` assert `ls .planning/todos/pending/ | wc -l` **-eq 7**; the live count is **10** (confirmed). The gate will be RED at execution for reasons unrelated to the phase's work — and worse, the plan's own arithmetic (`04-05-PLAN.md:537`: "down from 8 after plan 04-02's two closures") assumed 04-02 starts at 10, while `04-02-PLAN.md` Task 3 asserts pending **-eq 8** after closing 2, i.e. it assumed a start of 10 *and describes* "down from 10". Today pending is 10 *before* 04-02 runs, so 04-02's gate lands at 8 and 04-05's at 7 — internally consistent *if nothing else touches the directory*, but three new todos have already arrived since the arithmetic was written, and any future capture invalidates both gates silently. This is precisely the cycle-1 lesson: a total count where set membership is meant. **Recommendation:** change both gates to name-based assertions — 04-02: the two folded filenames absent from `pending/` and present in `completed/` (already asserted at `04-02-PLAN.md` acceptance via `rg -c -e 'stale-pages' -e 'strand-the-phase-4-sentinel'` on completed; make pending symmetric); 04-05: the one folded filename moved. Keep the count as informational only. This is a two-line edit per file; doing it now is cheaper than an execution-time reconciliation that re-opens the review question.

### 5. Suggestions

1. Apply the name-based todo assertion (above) to 04-02 and 04-05 in the same amendment pass; mirror into VALIDATION rows 04-02-03 / 04-05-03.
2. Optionally split the hash cardinality leg per key (above).
3. Non-blocking: the leg-1 pattern's duration class `[0-9.]+s` would also accept a malformed `(..s)`; irrelevant in practice since the line is tool-emitted, no change needed.

### 6. Risk Assessment

**LOW overall — approve for execution.** The cycle-5 changes are purely additive gate strengthening on an acceptance plan whose workstation-safety architecture (merged single trapped script, four blocking preconditions, bidirectional baseline restoration, RUN_ID binding) survived re-verification: trap armed before first mutation at `04-06-PLAN.md:299/:329` vs. first mutation at step 4, disarm-then-exit signal handlers at `:320-327`, preconditions at `:171-175`. Structure intact: 6 plans / 15 tasks, tag balance clean, 15 VALIDATION rows, no floor lowered (verified: new legs are `-eq 1`/`-eq 2`/`-ge 1` additions on previously unasserted properties). The two open items are one MEDIUM gate-robustness fix (todo counts) that is cheap to land now and will otherwise fail noisily at execution, and one LOW theoretical residue in the new hash gate. Neither touches the detection logic, the threat model, or the safety mechanisms.

---

## Consensus Summary

**Two independent lanes completed with repo access.** Codex and OpenCode were both prompt-fed the
same source-grounding instructions and both returned reviews citing `file:line` evidence that
resolves correctly against the live files. The OpenCode lane — UNAVAILABLE in cycle 3 after two
`[spawn error: ETIMEDOUT]` deaths — completed here in ~3.5 minutes with a zero-byte
`gsd-review-opencode.err` and 8 KB of substantive content. Per the cycle-5 mandate the lane's
`ok: true` field was **not** trusted on its own: the `.err` file was checked and the produced
sections were confirmed to contain real, resolvable citations before the lane was counted. This is
therefore a genuine two-lane consensus, not a single-lane result presented as one.

Every finding below was additionally **re-measured independently by the orchestrating agent**
against the live repo; those measurements are recorded in "Orchestrator verification" and, in one
case, they **overturn a lane's HIGH**.

### Agreed Strengths

- **All four cycle-5 fixes are present in both `04-06-PLAN.md` and `04-VALIDATION.md` and are
  mechanically sound.** Both lanes verified each by execution against fixture data rather than by
  reading. Three parties (Codex, OpenCode, orchestrator) independently reproduced FIX 1's
  measurement: corrected pattern 1 match, cycle-4 pattern 0, pre-merge unanchored form 3.
- **FIX 3's producer and consumer agree exactly.** Both lanes confirmed `04-06-PLAN.md:605`
  mandates the literal `Criterion N` (capital C) that the `-F` loop at `:628` greps, and both
  named this as specifically preventing a re-run of FIX 1's failure mode in a new place.
- **FIX 4's honesty split is honest.** Both lanes independently reached the same verdict on the
  adjudication question: the marker is a runtime artifact proving the arming step *completed*, it
  cannot prove textual ordering, and the plan says exactly that at `04-06-PLAN.md:339-344` and
  `:161`. Neither lane found it overclaimed. OpenCode added that the RUN_ID binding and the
  receipt's independent restoration proof close the evidentiary chain from two directions.
- **The threshold inventory moved only by addition; no floor was raised or lowered.** Confirmed by
  all three parties against the `983d624..28ee153` diff.
- **Structure and the do-not-regress list hold.** 6 plans / 4 waves / 15 tasks / 15 VALIDATION
  rows; anchored `<automated>` balance 3/3, 3/3, 2/2, 2/2, 3/3, 2/2; `read_first` 15/15 and
  anchored `acceptance_criteria` 15/15; `<threat_model>` ×6; "Artifacts this phase produces" ×6;
  `T-04-01`…`T-04-22` contiguous (22 distinct).
- **No lane raised a stale-ROADMAP finding**, so the `04-CONTEXT.md` supersession ground rules held
  for a third consecutive cycle.

### Agreed Concerns

- **The `todos/pending` total-count gates couple phase execution to unrelated todo intake**
  (`04-02-PLAN.md:456` `-eq 8`; `04-05-PLAN.md:507` `-eq 7`; mirrored at `04-VALIDATION.md` rows
  `04-02-03` and `04-05-03`). Both lanes independently reached the same remediation — assert the
  folded todos **by name** (absent from `pending/`, present in `completed/`) and demote the total
  to informational. This is the cycle-1 `rg -c` set-membership lesson applied to a directory
  listing. **Severity is contested** — see Divergent Views.
- **FIX 2's cardinality leg counts key-agnostic lines**, so a receipt with two
  `PAYLOAD_SHA256_BEFORE` lines and no `AFTER` line satisfies both the new `-eq 2` cardinality leg
  and the pre-existing `sort -u … -eq 1` distinctness leg. Raised by OpenCode and independently
  found and measured by the orchestrator before OpenCode returned; **missed by Codex**, which
  reported the fix fully sound. It is the same species FIX 2 exists to close — a combined pattern
  standing in for per-token assertions — one layer down.

### Divergent Views

- **Severity of the `todos/pending` gate — and its factual premise.** Codex rated it **HIGH**, on
  the stated ground that "the live directory contains 10 files" against a gate demanding 7, making
  the failure "already known, deterministic". OpenCode rated it **MEDIUM** and, in the same
  paragraph, correctly noted the chain is "internally consistent *if nothing else touches the
  directory*". **The orchestrator's measurement resolves this in OpenCode's favour and refutes the
  HIGH's premise** — see "Adjudication: the `todos/pending` drift" below. Both lanes' remediation
  is nonetheless correct and is retained as a MEDIUM.
- **Severity of the FIX 2 residue.** OpenCode rated it **LOW** (the fixed seven-line receipt shape
  makes a duplicated line unlikely). The orchestrator rates it **MEDIUM**: the duplication path is
  reachable through precisely the double-invocation defect the plan itself documents at
  `04-06-PLAN.md:321-324`, and the whole point of FIX 2 was that "unlikely by construction" is not
  what a gate is for. Codex did not find it at all.
- **Overall risk.** Codex: "HIGH for execution reliability, LOW for implementation design",
  entirely driven by the todo-count premise the orchestrator refutes. OpenCode: "LOW overall —
  approve for execution". With the premise corrected, the orchestrator concurs with OpenCode.

## Adjudication: the `todos/pending` drift

**The maintainer's brief and Codex both state this gate "will be RED at execution". Measurement
shows it will not be.** The reasoning error in both is comparing a gate's expected *post*-state
against the directory's current *pre*-state, without accounting for wave ordering.

Measured:

| Fact | Evidence |
|---|---|
| Live `pending/` count | **10** (`ls .planning/todos/pending/ \| wc -l`) |
| `04-02-PLAN.md` is **wave 1** | `04-02-PLAN.md:5` `wave: 1`; `04-VALIDATION.md:46-48` rows `04-02-0*` column 3 = `1` |
| `04-02` closes **2** todos, gate asserts `-eq 8` | `04-02-PLAN.md:421` `<files>` (two todo paths), `:456` `<automated>` |
| `04-05-PLAN.md` is **wave 3** | `04-05-PLAN.md:5` `wave: 3`; `04-VALIDATION.md:53-55` column 3 = `3` |
| `04-05` closes **1** todo, gate asserts `-eq 7` | `04-05-PLAN.md:442` `<files>`, `:507` `<automated>` |
| All three named todos are still in `pending/` | confirmed by listing |
| `pending/` has been **10 at every commit** from `ffad9ee` (which took it 9→10) through `28ee153` | `git ls-tree -r --name-only <c> -- .planning/todos/pending/ \| wc -l` at `ffad9ee`, `dbdc68f`, `63ac13e`, `1d221f9`, `3661fab`, `983d624`, `28ee153` |
| `ffad9ee` is an **ancestor of** `dbdc68f` (the cycle-1 replan that wrote `-eq 8`) | `git merge-base --is-ancestor ffad9ee dbdc68f` → true |

Therefore the chain is **10 → (wave 1, 04-02 closes 2) → 8 ✓ → (wave 3, 04-05 closes 1) → 7 ✓**.
Both gates are arithmetically correct and both will be **GREEN**, not RED. The plans' own prose is
also correct: `04-02-PLAN.md:467` "down from 10" and `04-05-PLAN.md:537` "down from 8 after plan
04-02's two closures". OpenCode's claim that "three new todos have already arrived since the
arithmetic was written" is **factually wrong** — the count has not moved since before the cycle-1
replan authored these numbers.

**Verdict: not a HIGH, and not a guaranteed execution failure. It IS a real MEDIUM requiring a
PLAN.md change**, for the reason both lanes gave independently rather than the reason either
stated: the gate asserts a **directory total** where **set membership** is meant, so any todo
captured between now and execution — an event this very branch demonstrates in `ffad9ee`, an
unrelated `/gsd-capture` landing mid-review-cycle — silently flips both gates RED without any
defect in the phase's work. That is the cycle-1 `rg -c` lesson, and it is cheap to close now.

## Orchestrator verification (independently measured)

Every claim below was executed, not reasoned about.

| Claim | Method | Result |
|---|---|---|
| FIX 1 present in both files | `rg -n` on `04-06-PLAN.md`, `04-VALIDATION.md` | Live gate at `04-06-PLAN.md:430` and `04-VALIDATION.md:56` both carry `\([0-9.]+s\)$` ✓ |
| FIX 1 old shape gone from live gates | `rg -c -F 'PASS: TestDetectBrewManaged_RealInstall$'` | 1 hit per file — **both are retrospective prose** in disposition tables (`04-06-PLAN.md:158`, `04-VALIDATION.md:179`), not gates ✓ |
| FIX 1 precision retained | three-line fixture (`_RealInstall`, `_RealInstallExtra`, `_RealInstall/sub`, each with a duration suffix) | corrected = **1**, cycle-4 = **0**, pre-merge unanchored = **3** ✓ exactly as the planner claimed |
| FIX 2 additive, not a replacement | read `04-06-PLAN.md:430` | `-eq 2` cardinality leg **and** `sort -u … -eq 1` distinctness leg both present, cardinality first ✓ |
| FIX 2 rejects a BEFORE-only receipt | fixture `rB.txt` | cardinality **1** → gate FAILS ✓ |
| FIX 2 rejects a duplicate-BEFORE receipt | fixture `rC.txt` (two identical `BEFORE`, no `AFTER`) | cardinality **2**, distinct **1** → gate **PASSES** ✗ **new finding** |
| FIX 3 gate present and executable | `rg -n 'for c in 1 2 3 4'` | `04-06-PLAN.md:628` (inside a closed `<automated>`) and `04-VALIDATION.md:57` ✓ |
| FIX 3 action mandates the grepped literal | read `04-06-PLAN.md:605-610` | mandates "the literal token `Criterion N` — capital C", and states the rationale is FIX 1's failure mode ✓ |
| FIX 4 ordering is a hard constraint | read `04-06-PLAN.md:329-344` | "**This append MUST stay in exactly this position: after the registration, before the first mutation**… a future edit that relocates it must be rejected" ✓ |
| FIX 4 non-overclaim | read `04-06-PLAN.md:339-344` | states the marker establishes the arming step COMPLETED and "does NOT on its own establish the script's textual ordering" ✓ **honest** |
| No floor raised or lowered | `git diff 983d624 28ee153` removed-vs-added threshold multisets | removed `-eq 0`×4 `-eq 1`×12 `-ge 1`×4 `-ge 3`×2; added `-eq 0`×4 `-eq 1`×20 `-eq 2`×5 `-ge 1`×8 `-ge 3`×3. `-eq 0` unchanged; every delta is an addition. The single `-ge 3` delta is **prose** in a new disposition row, not a gate ✓ |
| Unclosed `<automated>` hunt | anchored `^\s*<automated>` vs `</automated>\s*$` per file | 3/3, 3/3, 2/2, 2/2, 3/3, 2/2 = 15/15 ✓. Unanchored counts (3/3, 6/3, 2/2, 2/2, 7/3, 16/2) diverge wildly — **prose mentions**, exactly the trap the hunt list names |
| `acceptance_criteria` 15/15 | anchored `^\s*<acceptance_criteria>$` vs `^\s*</acceptance_criteria>$` | 15/15 ✓ (unanchored reports 19 — 4 prose mentions in disposition tables) |
| Structure | counts across all six plans | 15 tasks, 15 `read_first`, 15 VALIDATION rows, 6 `<threat_model>`, 6 artifact sections, `T-04-01`…`T-04-22` = 22 distinct contiguous ✓ |
| `04-01`…`04-05` byte-unchanged | `git diff --name-only 983d624 28ee153` | only `04-06-PLAN.md` and `04-VALIDATION.md` ✓ — their floors cannot have regressed by construction |
| `04-05`'s named-PASS count is satisfiable | `04-05` artifacts table lists exactly two `TestCosignIdentityPolicy*` funcs; gate greps the **prefix** `'^--- PASS: TestCosignIdentityPolicy'` with `-eq 2` | consistent — deliberately prefix-matching with an exact cardinality, not an unanchored floor ✓ |

## Verification coverage

**Source-grounding authority.** `drift-guard authority` = `intel`, but `.planning/intel/API-SURFACE.md`
carries `symbolCount: 0`, `stale: true`, and its own banner — "*api-map.json has no entries (intel
extraction is regex/JS-only or not yet populated). Treat absence here as 'unknown', not 'does not
exist'*". This is a **Go** repository. Non-JS symbols under an `intel` authority are **UNCHECKABLE
by the workflow's own definition** — never MISSING. All Go symbols below are therefore classified
UNCHECKABLE → INFO, with ripgrep/Read confirmation recorded instead.

| Symbol / identifier | Present in repo today | Classification |
|---|---|---|
| `internal/upgrade/` package (15 files incl. `taskfile_shape_test.go`, `verify.go`, `upgrade.go`) | yes | confirmed by `ls` |
| `detectBrewManaged`, `brewPointerMessage`, `brewInstall` and fields | no | **EXCLUDED** — `04-01-PLAN.md` § "Artifacts this phase produces" declares them new; the section states explicitly "none of these exist in the codebase today — do not treat them as drift" |
| `TestDetectBrewManaged_RealInstall`, `CODEGRAPH_BREW_ACCEPTANCE_PATH` | no | **EXCLUDED** — 04-01 artifacts; consumed by `04-06` per its own artifacts note |
| `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern`, `…_ZeroLiteralsIsError`, `extractCosignIdentityLiterals`, `cosignIdentityLiteral`, `cosignIdentitySANCorpus`, `SIG_DIR`, `PRIOR_SIG`, `SELF-UPGRADE-VERIFY-EVIDENCE` | no | **EXCLUDED** — `04-05-PLAN.md` artifacts |
| `FRESH_MAN_PAGE_COUNT`, `MAN_BASELINE_MARKER`, `MAN_REINSTALL_MARKER`, `man_pages_before`, `fresh_man_pages`, `CASK-REHEARSE-EVIDENCE schema=3` | no | **EXCLUDED** — `04-02-PLAN.md` artifacts |
| `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes` | no | **EXCLUDED** — `04-04-PLAN.md` artifacts |
| `04-EVIDENCE.md`, `UPGR-ACCEPTANCE-EVIDENCE` | no | **EXCLUDED** — `04-06-PLAN.md` artifacts |
| `Taskfile.yml` targets `verify:release-assets`, `release:rehearse-cask`, `release:dry-run-signed` | yes | confirmed by `rg` on `Taskfile.yml` |

**No symbol referenced by any plan was found MISSING.** Every absent identifier is declared under
its plan's "Artifacts this phase produces" section, all six of which are present.

## Convergence status

| Cycle | HIGH | Actionable non-HIGH | Total |
|---|---|---|---|
| 1 | 6 | 13 | 19 |
| 2 | 4 | 7 | 11 |
| 3 | 1 | 2 | 3 |
| 4 | — (review died on a session limit; fixes verified out-of-band, 4 new items found) | — | 4 |
| **5** | **0** | **2** | **2** |

Codex raised one HIGH; the orchestrator's wave-ordering and git-history measurements refute its
factual premise (the gate is GREEN, not RED) and it is carried forward at MEDIUM on the
brittleness ground both lanes independently identified. The record is left explicit so the
maintainer can overrule the downgrade — the remediation is identical either way.

Both remaining items are **cheap, local, and additive**: two per-key `test` legs in one
`<automated>` block plus its VALIDATION mirror, and name-based todo assertions replacing two
directory totals plus their two VALIDATION mirrors. Neither touches the detection logic, the
threat model, the trapped-script architecture, or any safety mechanism. Both lanes and the
orchestrator agree the phase is otherwise ready to execute.
