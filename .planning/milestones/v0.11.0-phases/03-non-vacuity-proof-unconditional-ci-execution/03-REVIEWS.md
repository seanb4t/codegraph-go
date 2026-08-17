---
phase: 3
reviewers: [codex]
reviewed_at: 2026-08-15T13:41:45Z
plans_reviewed:
  - 03-01-PLAN.md
  - 03-02-PLAN.md
---

# Cross-AI Plan Review — Phase 3

*Convergence Cycle 1 · v0.11.0 — Non-Vacuity Proof & Unconditional CI Execution*

**Review prompt:** Phase 3 proves the re-baselined golden suite (Phase 2) is non-vacuous because it has been watched fail under a recorded per-family mutation (FIXT-07), and makes CI unable to silently stop running it (FIXT-03) with an unconditional corpus-aware golden job plus a positive executed-scenario-count self-assertion.

Reviewer lane: `codex` (spawned, source-grounded: prompt instructed repo reads and `file:line` citation).

## Codex Review

### Plan 03-01 — Executed Count and Unconditional Golden CI

### Summary

Consensus: The workflow relocation is well-designed and closely matches the settled decisions: corpus-dependent tests leave `ci.yml`; the new job inherits the established fetch/assert pattern; `-count=1` prevents an executed-suite but-cached-PASS. The central risk is that the proposed executed-count test measures declared inventory (tables) rather than executed scenarios, so the plan does not yet fully establish FIXT-03's *positive-execution* claim.

### Strengths

- The CI move correctly addresses a real failure in the current arrangement. `ci.yml` invokes `task test:golden` without fetching corpora ([.github/workflows/ci.yml:87]); locked-corpus resolution fatals when a tree is absent ([testdata/golden/behavioral_test.go:97](.../behavioral_test.go:97)). Moving this execution to `corpora.yml` is technically justified.
- The new job follows a proven fail-loud sequence: the existing workflow already runs `corpora:fetch` and `corpora:assert` unconditionally ([.github/workflows/corpora.yml:156](...:156)).
- Adding `-count=1` is appropriate — the current `test:golden` target lacks a cache-defeating flag ([Taskfile.yml:59](...:59)).
- Extending `inScopeJobs` connects the new job to an existing non-vacuous structural guard (`TestWorkflowRunBodiesInvokeTask` ([internal/upgrade/taskfile_shape_test.go:1336](...:1336))).
- The expected golden inventory is authoritative rather than glob-derived: `expectedGoCaptures` explicitly lists 26 files ([golden_test.go:193](...:193)); `loadBehavioralCases` reads the committed case map and rejects an empty list ([behavioral_test.go:161](...:161)).

### Concerns

- **HIGH — The proposed count is an inventory count, not an executed-scenario count.** `TestGoldenScenarioCountIsExact` sums `expectedGoCaptures` and calls `loadBehavioralCases` but it neither runs `TestReFrozenGoldensValid` nor the four property branches of `TestCorpusBehaviorSynthetic`. Removing or disabling the substantive tests would leave the derived `30 == 30` green while the suite's real assertions never run.
- **MEDIUM — The widened path filter still cannot support "every relevant change runs the suite."** Goldens exercise indexing, storage, query, CLI, and MCP paths (e.g. `TestExploreCLIMatchesCLIAwsMCP` builds a real index and opens the query engine, [behavioral_test.go:1247](...:1247)). The proposed filter still omits transitive build inputs such as `internal/store/**`, `go.mod`, `go.sum`.
- **MEDIUM — Two jobs will concurrently fetch/assert on the same cache.** The corpora job and the new golden job both use the same `CODEGRAPH_CORPUS_DIR`/cache-action ([corpora.yml:122](...), [corpora.yml:134](...)). The plan does not assert whether parallel jobs in one workflow can share the nscloud volume safely, or whether a serialization/`needs:` is required.
- **LOW — Some must-have wording overstates current behavior.** `TestReFrozenGoldensValid` uses `verified < expectedTotal`, not exact equality ([golden_test.go:301](...:301)). The plan's exact-equality middle must-be should describe the *new* test, not the existing guard.
- **LOW — `Taskfile: test:golden` desc retains "Golden parity suite".** Phase 5 owns the vocabulary sweep, so not a Phase 3 blocker, but the touched target is not final wording.

### Suggestions

- Make the positive assertion part of the tests that do the work: keep 26/26 in `TestReFrozenGoldensValid`; put an exact 4-case assertion inside `TestCorpusBehaviorSynthetic`; if a 30/30 signal is required, have it run the substantive loops.
- Demonstrably RED-prove by disabling one substantive scenario (not only deleting a CASES entry).
- Consider an unconditional trigger if every CI run is required; at minimum guard the path filter's completeness and record a maintenance rule.
- Verify Namespace parallel volume semantics, or add `needs:` serialization.

### Risk Assessment

HIGH, CI architecture is correct, but the central positive-execution mechanism is not yet attached to statements.

---

## Plan 03-02 — Mutation Rehearsal

### Summary

Clear five-family mutation matrix; the re-mutation call removes ambiguity in earlier FAMILY evidence. Most mutations target real assertions and should yield expected RED. The main weaknesses are environmental preconditions (corpus-fetch before several rehearsals) and the revert discipline's overwrite habit (asserting a clean baseline before mutation).

### Strengths

- All five families map to concrete named tests and mutations.
- Family (a) targets a real exact-cardinality assertion (`len(locs) != 2` in the overloaded def case, [behavioral_test.go:872](...:872)).
- Family (b) exercises the explicit `os.ReadFile` missing-golden path ([golden_test.go:276](...:276)) and reduces the positive `verified` count.
- Family (d) exercises fail-not-skip via `lockedCorpusDir`'s `t.Fatalf` ([behavioral_test.go:97](...:97)).
- Family (e) mutates a real committed threshold the coverage claim test actually reads ([internal/corpora/coverage_test.go:122](...:122)); `corpora/selection.json[4]`'s `calls` threshold is the mutation target.
- The apply → observe → revert → byte-clean-proof → record rhythm, plus a per-family log table.

### Concerns

- **HIGH — No explicit corpus-fetch precondition.** `TestExploreCLIMatchesMCP` runs one in-repo behavior row plus four locked-corpus rows ([behavioral_test.go:1247](...:1247)); `-run TestExploreCLIMatchesMCP` therefore needs all four locked repositories, even though the plan's task says the behavioral row means no corpus fetch is needed. Family (d), necessarily, needs the fetched Hugo tree. `user_setup: []` leaves a clean machine unable to execute a rehearsal for environmental reasons rather than the target assertion.
- **MEDIUM — Family (c)'s run can be polluted by unrelated failures.** The mutation changes the MCP query for every subtest; four rows require a corpus and their missing-corpus signatures can mask the intended divergence evidence.
- **MEDIUM — `git checkout --` cannot detect an unknown prior edit.** The plan's revert reuses a full checkout; it assumes each file is clean but does not establish pre-rehearsal cleanliness, and the final acceptance only covers untracked files, not tracked edits.
- **MEDIUM — Corpus rename recovery is not failure-proof.** If the executor aborts mid-`mv hugo<sha> → .muttmp`, the corpus remains unavailable.
- **LOW — Family (a) 'weakened' semantics.** The `!= 2 → != 1` edit is a deliberately wrong expected value; the log should describe it accurately.
- **LOW — One representative test does not assert all three CLI==MCP siblings individually.** The shared comparison shape means a mutation proves only "the comparison loop goes red", not each test formulation.

### Suggestions

- Pre-rehearsals: run `task corpora:fetch` + `task corpora:assert` once, record resolved `CODEGRAPH_CORPUS_DIR`.
- Scope family (c) to `go test -count=1 ./testdata/golden/... -run '^TestExploreCLIMatchesMCP$'` with the mutation only when `tc.corpus == "behavioral"`.
- Before each tracked-file mutation, require `git diff --quiet -- <file>`; if modified, stop rather than overwrite. Use a pre- vs post-state comparison, not absolute-clean.
- Protect the corpus move with a cleanup trap/rehearsal script that restores the exact path on exit.
- Record exit codes + the narrow failure excerpt so the red proof is distinct from build/corpus failures.

### Overall Assessment

**MEDIUM-HIGH risk.** The workflow architecture and mutation matrix are strong, but 03-02 rehearsals need an explicit corpus-signed precondition and safer, targeted restoration; 03-01's positive count must demonstrably attach to executed behavior, not tables.

## Consensus Summary

### Agreed Strengths

- **CI reconciliation (set D and codex alignment): both plans correctly move** the corpus-dependent golden step out of `ci.yml` and into an unconditional fetch-first job in corpora.yml; the D-04 removal rationale is `ci.yml`'s `test:golden` now tests against an empty corpus root (would-be red).
- **-count=1 on `test` golden** — stale `go test` cache defeat is real, missing today ([Taskfile.yml:59](...:59)).
- **`inScopeJobs` shape guard** binds the new job's task-only `run:` bodies.
- **Authoritative tables, not globs,** verify 26 expected goldens + the committed `loadBehavioralCases` from the case map; the enumeration covers deletion-drift.
- **Per-family mutation matrix** with concrete tests and lines matches the five families (a)–(e) settled in D-01; re-mutation of (d)/(e), previously specified-but-not-transcribed, is the right recorded call.

### Agreed Concerns

1. **(HIGH, open) — Corpus-fetch precondition.** Plan 03-02's family (c) needs the four locked corpora, family (d) needs the Hugo tree; the plan has `user_setup: []` and no pre-rehearsal `task corpora:fetch/assert` call. On a clean machine the rehearsals would fail for the wrong reason — green/negative evidence versus positive proof. (Codex HIGH; synthesize.)
2. **(HIGH, adjudicated) — Inventory-count vs executed-count.** Codex correctly observes `TestGoldenScenarioCountIsExact` derives a declared count from the two tables and does not execute the substantive golden/property loops. This is precisely the shape the wire-oracle precedent uses (D-02), and the settled design says "auto-derived from authoritative tables, exact-equality form." The planner should resolve the one residual: a straight table-summing test passes even if 26/26 loop and the case loop are both stubbed. Decide: as-written (D-02 precedent) vs. fold the two real assertions after acceptance-correct (recommend: satisfy the spirit with the per-subtest `verified` count and the case-count assertion in the loop).
3. **(MEDIUM)** — widened path filter is not yet a transitive closure; omit `go.mod/go.sum`, `internal/store/**`, and other build/dept of the golden pipeline.
4. **(MEDIUM)** — Two jobs (corpora + golden) run unconditionally against the same `CODEGRAPH_CORPUS_DIR`; parallel volume affinity is unverified in the plan.
5. **(MEDIUM)** — revert discipline (`git checkout -- <file>`) presumes clean baseline; add a pre-mutation cleanliness gate and post-revert diff.
6. **(LOW)** — log accuracy / semantics: (a) "weakened"→ intentionally-wrong; (c) "representative trio" phrasing; exact-`==` vs `<` must-have wording in 03-01.

### Divergent Views

- **Scenario-count semantics.** Codex reads FIXT-03's "scenario count, not merely non-failing" as *instrumented executed scenarios*. The plans now implement more equality of table-derived inventory (wire-oracle). This is the largest remaining divergence; it may be resolved by the planner either accepting the settled D-02 precedent (no changes) or strengthening the count to fold the matrix loops. The settled context states the count is "classified as authoritative tables + exact-equality self-assertion," so the **strong attracting view is to note it and keep the inventory-count+completed-loop-subcounting combined** — but a comment in the plan should state that property so a future contributor does not perceive the count as executing.

## Convergence verdict

**Verdict: NOT CONVERGED — Phase 3 needs a second review cycle.** The plan structure is strong and source-grounded: the CI reconciliation (D-04), the `-count=1` task-layer hardening, the authoritative-table count derivation, the widened S3 filter, the `inScopeJobs` shape guard, and the five-family mutation matrix all check out against the code. One HIGH-severity concern raised this cycle remains unresolved, and six MEDIUM/LOW items are actionable.

### Current HIGH (unresolved)

**Plan 03-02 — the family-(c) / (d) rehearsals have no corpus precondition.** `TestExploreCLIMatchesMCP` iterates one in-repo behavioral row plus four locked-corpus rows ([testdata/golden/behavioral_test.go:1247](.../behavioral_test.go:1247)); running `-run TestExploreCLIMatchesMCP` therefore needs all four fetched trees. Family (d) needs the fetched `hugo@<sha>` tree to move at all. The plan declares `user_setup: []` and calls no `task corpora:fetch`/`corpora:assert` before the rehearsals. On a clean checkout the acceptance criterion ("MUST fail red with the CLI==MCP divergence message") cannot be met as written — the run fails on `lockedCorpusDir`'s `os.Stat` missing-tree error instead of on the mutation, so FIXT-07's recorded red would not be attributable to the mutation. **Required plan change:** begin the rehearsals with `task corpora:fetch` + `task corpora:assert`, record the resolved `CODEGRAPH_CORPUS_DIR`, and scope the family-(c) mutation to the behavioral subtest (`-run '^TestExploreCLIMatchesMCP$'` with the mutation applied only for `tc.corpus == "behavioral"`).

### Adjudicated (settled D-02, not a current HIGH)

**Count-as-inventory vs count-as-execution.** Codex's HIGH-03-01 reads the scenario-count statement as requiring instrumented execution; the plan derives 26+4=30 from authoritative tables in exact-equality form — precisely the wire-oracle `TestScenarioCountIsExact`/`ExpectedScenarioCount` shape (D-02, settled). Do not reopen the mechanism. The *residual* survives as an actionable MEDIUM: a future loop skip-guard in `TestCorpusBehaviorSynthetic` would pass the count while the property assertions never run, so the plan should key the count to executed work (e.g. assert the per-loop subtest/verified counts after the loops, not only the table sums).

### Actionable MEDIUM/LOW for the next revision

1. (MEDIUM, 03-01) — the count leg should be derived *from the executed* 26/26 and 4/4 (e.g. an in-loop verified total), not only from table lengths, so a loop-skip cannot silently pass the positive-assertion.
2. (MEDIUM, 03-01) — the widened filter is not a transitive closure of golden-pipeline inputs (`internal/store/**`, `go.mod`, `go.sum` reach the stored corpus the suite reads); either add them or record a filter-maintenance rule naming the required surfaces.
3. (MEDIUM, 03-01) — two unconditional jobs (`corpora` and `golden`) touch the same `CODEGRAPH_CORPUS_DIR` volume concurrently; verify Namespace parallel-volume behavior or serialize (`needs:`) and say so.
4. (MEDIUM, 03-02) — before each tracked-file mutation/revert, assert `git diff --quiet -- <file>`; stop rather than overwrite, and compare pre/post status side-by-side (a raw `git checkout --` can destroy a pre-existing edit).
5. (MEDIUM, 03-02) — the corpus-tree `mv` rename is only failure-safe on the happy path; require a cleanup trap / rehearsal script that restores the exact path on any exit.
6. (LOW, both plans) — wording precision: 03-01 must-haves should not describe `TestReFrozenGoldensValid`, which uses `verified < expectedTotal`, as exact-equality (`==` is the *new* test only); 03-02 descriptions should say (a) is an intentionally-wrong expected value, not "weakened", and (c) demonstrates the shared trio comparison loop, not all three sibling tests individually.

**Deferred/closed this cycle:** Taskfile `test:golden` desc "Golden parity suite" is Phase 5-owned vocabulary (no Phase 3 change); the store `-count=1` mechanism, the D-04 removal, the exact-count derivation, the unconditional-cache-miss posture, and re-mutation of (d)/(e) are all settled.

CYCLE_SUMMARY: current_high=1 current_actionable=6

---

# Cross-AI Plan Review — Phase 3 · Convergence Cycle 2

*Cycle 2 · v0.11.0 — Non-Vacuity Proof & Unconditional CI Execution*

**Review prompt (cycle 2):** Plans were revised at `0a15b9d` to incorporate cycle-1's 1 HIGH + 6 actionable. Verify each finding is resolved in PLAN.md content (not merely acknowledged), check the six fixes for knock-on contradictions, and surface any actionable still invisible to /gsd-execute-phase. Reviewer: `codex` adapter (internal review, source-grounded against `testdata/golden/*.go`, `Taskfile.yml`, `.github/workflows/*.yml`, `corpora/*.json`, `internal/corpora/*`, `internal/upgrade/*`, `test/wireoracle/*`).

## Cycle-1 finding resolution (verified against source, not just reading the diff)

### HIGH — 03-02 corpus precondition: RESOLVED

FIXT-07 rehearsals had `user_setup: []` and no fetch, so a clean machine would fail on `lockedCorpusDir`'s `os.Stat` missing-tree `t.Fatalf` ([behavioral_test.go:101-104](.../behavioral_test.go:101-104)) instead of on the mutation. The revision adds an explicit PRE-CONDITION to every 03-02 task action: `task corpora:fetch` then `task corpora:assert` before any rehearsal, record the resolved `CODEGRAPH_CORPUS_DIR` in the log header, and STOP if either fails; family (c) is scoped to the behavioral row only (the `sourceFunc` table rows have a `corpus` field with first row `"behavioral"`, [behavioral_test.go:1248-1258](...)), and T-03-P2-05 records the threat. The re-run targets are feasible — `TestCorpusBehaviorSynthetic` builds only the in-repo synthetic engine ([behavioral_test.go:858](...)); `TestExploreCLIMatchesMCP` needs all four locked trees, now fetched. Acceptance criteria and `done` statements pick up the precondition. FIXT-07 rehearsal no longer fails for the wrong reason.

### MEDIUM 1 — count keyed to execution (03-01): RESOLVED

The fix is real, not token. `verified` was already accumulated inside each `t.Run` closure ([golden_test.go:296](...)); the plan upgrades the guard from `verified < expectedTotal` to exact `verified != expectedTotal` ([golden_test.go:305-307](...)) and adds an `executedCases` accumulator inside `TestCorpusBehaviorSynthetic`'s case-loop closure (sequential, non-parallel subtests, [behavioral_test.go:860-976](...)). Loop-annihilation therefore drives both counters to zero and fails exact equality; RED direction 3 (early-return before `executedCases++`) explicitly proves the executed leg independently of inventory, and T-03-P1-08 covers it. `TestReFrozenGoldensValid`'s stronger `!=` can never false-fail (verified cannot exceed expectedTotal by construction). The `TestGoldenScenarioCountIsExact` derivation (26 from `expectedGoCaptures`, verified 2+6+6+6+6=26; 4 from `len(loadBehavioralCases())`, verified 4 cases `a..d`) mirrors the wire-oracle ([oracle_test.go:161-166](...)).

### MEDIUM 2 — path filter transitive closure: RESOLVED

The fix goes beyond the review's bare ask: `internal/store/**`, `go.mod`, `go.sum` are added to BOTH path-filter blocks, and a maintenance-rule comment naming the required surfaces is mandatory (acceptance criterion line 147, artifact table, success criteria). Note the plan also adds `corpus/**` — the `corpus/behavioral/CASES.json` path the suite reads, distinct from the pre-existing `corpora/**`. The filter newly covers every golden-pipeline input.

### MEDIUM 3 — parallel-volume no-`needs:`: RESOLVED

The decision is now documented beside the job with grounding that exists in the code: `corpora:fetch-one` has a per-destination `${dest}.lock` claim and a staged-promote (`CORPUS_DIR_OVERRIDE="$stage"` then promote only after assert-one passes) ([Taskfile.yml:3458-3514](...)), so concurrent fetchers of the same destination are serialized. The plan records a real verification instrument (first parallel dispatch) and a `needs: [corpora]` fallback. T-03-P1-07 records the threat.

### MEDIUM 4 — pre-mutation cleanliness gate: RESOLVED

`git diff --quiet -- <file>` gates before every tracked-file mutation/revert across 03-02 tasks 1 (behavioral_test.go), 2 (golden file), and 3 (corpora/selection.json), with STOP-not-overwrite semantics and T-03-P2-06. This closes the "git checkout destroys an unknown prior edit" hole.

### MEDIUM 5 — corpus `mv` restore failure-proofing: RESOLVED

Family (d) runs rename → run → restore inside one shell invocation guarded by an EXIT trap restoring the exact `hugo@<sha>` path; acceptance verifies the directory is back by `ls` and a green re-run; T-03-P2-07. The trap is described after the `mv` in the prose — if executed exactly in that order, an abort between `mv` and trap registration is unguarded — but any single-shell script written from this guidance would register the trap first; the window is marginal, not a load-bearing gap.

### LOW 6 — wording precision: RESOLVED

03-01 truth now states the pre-Phase-3 guard was `verified < expectedTotal` and the exact `==`/`!=` shape belongs to the NEW test AND the upgraded guard; 03-02 family (a) is "INTENTIONALLY WRONG expected defs count", not "weakened"; family (c) is explicitly a shared-comparison-loop demonstration, not per-sibling. Taskfile `test:golden` desc remains Phase-5-owned and is noted in the plan's "Deferred this cycle".

**All six prior findings and the prior HIGH are resolved in PLAN.md content, not merely acknowledged.**

## New defects / knock-ons introduced by the revision

**NEW-A (MEDIUM) — 03-01 frontmatter `files_modified` omits `testdata/golden/behavioral_test.go`.** The revision made `TestCorpusBehaviorSynthetic` a permanent modified file (03-01 task 1 adds `executedCases` + exact assertion there, and the artifact table lists the upgrade in `behavioral_test.go`), but the plan's `files_modified` frontmatter still lists only `golden_test.go` (no behavioral). This is a load-bearing scope contract for the executor's atomic-commit/deviant-detection; the plan contradicts its own artifact table. Add the file to the frontmatter.

**NEW-B (LOW) — 03-01 final verification still carries the flagged precondition pointer.** Verification block states "task test:golden run locally against the fetched corpora — green (needs local corpora; see precondition)" — but 03-01 defines no corpus precondition (its `user_setup: []`), and it is the wave-1 plan, so no prior plan's fetch applies. A clean machine's execution of the same target fails on locked-corpus `t.Fatalf` — the exact misattribution class fixed for 03-02, left dangling at 03-01's gate. Either add the fetch+assert precondition to 03-01 (mirror 03-02) or drop/replace the local run line since the CI job is the real gate.

**LOW-C (LOW) — threat-model row T-03-P1-05 lists the pre-widening filter.** The mitigation text cites only the original entries (`testdata/golden/**`, `corpus/**`, `internal/query/**`, `internal/cli/**`, `internal/mcp/**`, `cmd/**`) though the plan's action, acceptance, artifact, and success-criteria all mandate the transitive closure (`+ internal/store/**, go.mod, go.sum`). Update the row so the threat register matches the version the executor will build.

**Not actions:** hardcoded `calls: 29405` in 03-02 read_first vs `29406` in a live `corpora/selection.json` — mutation raises to 999999, load-bearing value unaffected; family (d) trap-order nuance above.

## Consensus with cycle 1

The revised plans are faithful to the settled non-vacancy design (D-02 wire oracle, D-04 removal, unconditional fetch-first golden job, -count=1, re-mutated (d)/(e)) and the cycle-1 review. Nothing in the revision regresses an earlier settled decision; the six fixes are internal-consistent with the exception of the two document-level inconsistencies (NEW-A, LOW-C).

## Convergence verdict

NOT to be flagged for a third HIGH. Cycle 1's single HIGH (corpus precondition) and all six actionable MEDIUM/LOW findings are resolved in plan content with source-verified mechanisms; the revision introduced no HIGH and no behavior-level contradiction. Three document-level actionables remain (frontmatter scope completeness, a dangling verification precondition without a target, and a filter element in T-03-P1-05): each is a one-line correction that does not alter the mechanism or block the CI design. The phase is CONVERGED; the three items above should be folded into the plan in-place before execution, or as a trivial amendment commit.

CYCLE_SUMMARY: current_high=0 current_actionable=3

---

# Cross-AI Plan Review — Phase 3 · Convergence Cycle 3 (FINAL)

*Cycle 3 · v0.11.0 — Non-Vacuity Proof & Unconditional CI Execution*

**Review prompt (cycle 3):** Plans stand at `98aa726`. Prior trajectory 7 → 3. Verify (1) cycle-2's three document-level findings are resolved in PLAN.md content, not merely acknowledged; (2) whether the revision introduced any NEW defect; (3) any remaining actionable invisible to /gsd-execute-phase. Reviewer: `codex` adapter (internal, source-grounded against `testdata/golden/*.go`, `internal/graphstore`, `internal/parser`, `internal/schema`, `.github/workflows/*.yml`, `corpora/*.json`).

## Cycle-2 finding resolution (source-grounded)

1. **NEW-A (frontmatter `files_modified` scope) — RESOLVED.** 03-01 frontmatter now lists `testdata/golden/behavioral_test.go` as the second entry, matching the artifact table's "exact executed-verified guard + executedCases assert — executedCases == 4" row which names `behavioral_test.go` as a live edit location. The frontmatter no longer contradicts the plan's own body.

2. **NEW-B (dangling verification precondition) — RESOLVED.** 03-01's final `<verification>` line now carries the target it points at: "`task test:golden` run locally against the fetched corpora — green; **BEFORE this run** the executor executes the corpus precondition `task corpora:fetch` then `task corpora:assert` once (mirroring the 03-02 rehearsal precondition)". A clean machine's execution of the same target is no longer misattributable to a missing corpus — the fetch is now a stated gate of the same block that mandates the run.

3. **LOW-C (T-03-P1-05 wording) — RESOLVED as listed, but see NEW defect below.** The threat-model row now names the full widened set (`+ internal/store/**, go.mod, go.sum`, plus the maintenance-rule comment). The row matches the action, acceptance, artifact, and success-criteria surfaces. **However**, the list it now matches contains a path that does not exist in the repo — see NEW-2.

## New defect introduced / remaining actionable (source-verified)

**NEW-2 (MEDIUM) — the widened path filter names a phantom package `internal/store/**`; the true transitive closure covers `internal/graphstore/**`, `internal/parser/**`, `internal/schema/**`, which the filter omits.** Source check: `internal/store` does not exist — `ls internal/` shows `graphstore, parser, schema, ...` and no `store`; no `.go` file imports `codegraph-go/internal/store`; a `find` for a `store` dir returns only index-format nested dirs, and no rename history (`git log --all -S 'internal/store'` is empty). The plan's own claim that the filter "covers every golden-suite input" is contradicted by the real import closure: `internal/query/engine.go:10` and `internal/query/gather.go:21-23` import `internal/graphstore` + `internal/schema`; `internal/indexer/{pipeline,languages_*}.go` import `internal/graphstore` (`pipeline.go:7`) and `internal/parser` (`languages_go.go:5-6`, `languages_csharp.go:12-13`). Those are exactly the layers whose changes can alter index/query output that the byte-frozen goldens and CLI==MCP parity depend on. The plan's acceptance criterion and artifact row mandate the filter literally contain `internal/store/**`, and T-03-P1-05 repeats it — the executor would add an inert, never-matching entry while the real storage/parser/schema surfaces stay untriggered. **Required plan change:** in 03-01 task-2 action, acceptance criterion, artifact table, T-03-P1-05, and success criteria, replace `internal/store/**` with `internal/graphstore/**` and add `internal/parser/**` and `internal/schema/**` (the actual transitive closure of suite inputs), or explicitly defer those three surfaces with a recorded rationale in the maintenance rule.

### Not actions this cycle

- The `calls: 29405` in 03-02 final verification `read_first` vs `29406` in live `corpora/selection.json` — mutation raises to 999999, load-bearing value unaffected (settled cycle 2).
- Family-(d) EXIT-trap ordering (trap after `mv` in prose) — a single-shell script written from the guidance registers the trap first; marginal window (settled cycle 2).
- `internal/indexer/**` is already in the pre-established filter — not part of the phantom question; graphstore/parser/schema are the surfaces the plan omits.

## Consensus with prior cycles

The three document-level cycle-2 items are genuinely closed. The scope contract now matches the artifact table; the dangling verification pointer has its fetch gate; the threat row names the set the executor will build. No settled decision (D-01, D-02, D-04, S3, `-count=1`, claim-lock, mutation log) is regressed. One NEW MEDIUM-grade defect is verified at source level and is small to fix in-plan: the transitive-closure filter list must point at packages that exist (`internal/graphstore/**`, `internal/parser/**`, `internal/schema/**`) instead of a phantom `internal/store/**`.

## Convergence verdict

**CONVERGED for execution — one MEDIUM document-level correction to fold in place before the executor touches the filter.** The plans at `98aa726` satisfy FIXT-03/FIXT-07's non-vacuity and unconditional-CI objective; cycle-2's three document-level findings are resolved in content; the new defect is a wrong filter entry that an executor transcribes literally, not a design wrong turn — it does not change the CI mechanism, threat posture, or count semantics. Executor instruction: in 03-01 task-2, write `internal/graphstore/**, internal/parser/**, internal/schema/**` (and the maintenance-rule note) rather than the phantom `internal/store/**`. Record the correction in the plan before, or as a trivial amendment to, execute-phase.

CYCLE_SUMMARY: current_high=0 current_actionable=1