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