---
phase: 4
cycle: 3
reviewers: [codex]
reviewed_at: 2026-08-29T22:05:00Z
plans_reviewed:
  - 04-01-PLAN.md
  - 04-02-PLAN.md
  - 04-03-PLAN.md
  - 04-04-PLAN.md
  - 04-05-PLAN.md
  - 04-06-PLAN.md
  - 04-07-PLAN.md
revision_under_review: 6ccff5af
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 4 (Cycle 3, FINAL)

Convergence cycle 3, the last automated cycle. Trajectory:

| Cycle | Findings | Revision |
|---|---|---|
| 1 | 9 HIGH + 11 actionable non-HIGH | `08da6afc` |
| 2 | 1 HIGH + 2 actionable non-HIGH — all three INTRODUCED by `08da6afc` | `6ccff5af` |
| 3 (this) | 1 HIGH + 2 actionable non-HIGH — the HIGH was also introduced by `08da6afc` and missed by cycle 2 | — (escalates to maintainer) |

`6ccff5af` touched exactly four files (`04-05`, `04-06`, `04-07` PLAN.md and `04-VALIDATION.md`);
`04-01`–`04-04` were deliberately untouched. Each touched plan carries a
`## Cycle-2 cross-AI review dispositions` table.

The reviewer was given the closed-issue list (the three textual `; test "$RC" -eq 0` negative
examples at `04-01:464`, `04-01:556`, `04-04:315`; the four Go-side `&&`-chained gates; the
`web:components:drift` enumeration pathspec and its 8/2 floors; `ROUTE_LOCAL_PARAMS`;
`components-drift.yml`'s absence from `release.yml` and `requiredCheckNames`; `GetHealth`
naming; `mutatingVerbs`; `proto:drift`'s `nfiles -lt 4` floor; TanStack v9 runes API;
`doublestar` root-level behaviour; and `Engine.Status`'s pre-existing worktree-subprocess
cost) and did not re-raise any of them.

## Verification of the three cycle-2 fixes (independently confirmed in-tree)

**1. `snapshotAgreement` widening — CORRECT.** `web/src/lib/status.ts:36-39` still carries only
`verdict` + `commit`, and `classifyStatus` at `:49-50` reduces the SHA to a presence flag —
so the cycle-2 diagnosis was right. The additive `commitSha: string` fix is the right shape.
The three typed construction sites the plan enumerates at `04-05:132` all exist and are
correctly cited: `web/src/routes/+layout.svelte:30`, `web/src/routes/browse/+page.svelte:47`,
`web/tests/degrade-states.test.ts:17-18`. The new `depends_on: ["04-01","04-03"]` creates no
wave conflict — 04-01 owns `status.ts`/`status.test.ts` in wave 1, 04-05 in wave 2, and no
other wave-2 plan (04-04) touches any file 04-05 touches. The two-suite gate at `04-05:264` is
genuinely discriminating: `web/tests/status.test.ts` has exactly **16** tests today, so the
`>= 18` floor can only be met by adding the two the widening requires, and the separate
health-view `>= 8` floor over a file that does not yet exist cannot be masked by the status
surplus. Both floors and both captured exit statuses are one `&&`-chained final command.

**2. `components-drift.yml` bootstrap — the guard construction is CORRECT.** The four
`test -n "$CO" && test -n "$SN" && test -n "$CL" && test -n "$RL"` guards precede every
`rg -q -F "$VAR"` use, so an empty extraction cannot make `rg -F ""` match everything, and
`test "$CL" -lt "$RL"` cannot run on empty operands. The extraction is robust against real
formatting: `rg -o 'actions/checkout@[0-9a-f]{40}'` hits `.github/workflows/ci.yml:51`,
`actions/setup-node@[0-9a-f]{40}` hits `:132`, and the literal `node-version: "24"` is at
`:134`. `./.github/actions/install-task` is used at `ci.yml:65` and `corpora.yml:187` as the
plan claims. The `checkStepInvokesTask` reasoning is correct —
`internal/upgrade/taskfile_shape_test.go:1399-1401` returns nil for any step with an empty
`Run`, so the three `uses:` bootstrap steps need no `runBodyExceptions` entry.

**3. below-minimum-while-in-flight — CORRECT.** `web/src/lib/search.ts` `setQuery`'s
below-minimum branch does exactly the three things the plan claims: abort (`:219`), request-ID
invalidation (`:223`), visible-state clear (`:224`). `onBelowMinimum()` gives the third a home
while leaving `SEARCH_MIN_CHARS` in one place. Settling the original promise *after* the
below-minimum term is the right way to prove the identity bump, since abort alone does not
stop a non-cooperative promise. The byte-unchanged `search.test.ts` claim is now correctly
qualified as necessary-not-sufficient in three places (`key_links`, Task 1 step (b), behavior
block).

`04-VALIDATION.md` was updated consistently with all three (rows `04-05-T1` and `04-05-T3`
gained `T-04-22` and the separate 8/18 floors).

## Codex Review

# Cycle 3 Plan Review

## 04-01 — Tracer, URL Grammar, and Callers Table

### Summary

The tracer is well chosen and validates the riskiest frontend integration before the other analyses build on it. The plan also correctly addresses the shared status-gate amplification problem and makes `DataTable` generic for later health tables.

### Strengths

- The server already exposes the required Callers contract as `symbol` plus `limit`, returning `Location` rows, so this is a genuine vertical slice rather than speculative API work ([internal/uiproto/uiv1/ui.proto:258](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:258)).
- Passing raw limits through is consistent with the authoritative server validation at [internal/query/validate.go:107](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/validate.go:107).
- The route-scoped status-identity change addresses a real mechanism: the layout calls `notifyNavigated(navigationIdentity(page.url))` reactively at [web/src/routes/+layout.svelte:47](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:47), while the existing normalizer currently strips only `q` at [web/src/lib/status.ts:70](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:70).
- Sorting, failure taxonomy, URL round trips, repeated `file=` values, and client-bound absence all receive executable assertions.

### Concerns

- None remaining.

### Suggestions

- No blocking changes. Preserve the plan’s tracer-first commit boundary during execution.

### Risk Assessment

**MEDIUM.** The plan is large and introduces a newly released table adapter, but its staged RED/type-check/DOM verification is proportionate.

---

## 04-02 — Recursive File Globs

### Summary

The backend fix is correctly placed in `Engine.Files`, which serves CLI, MCP, and UI callers. However, the RED task as currently ordered cannot reach its intended seven subtests.

### Strengths

- The root cause is accurately located: both pattern validation and per-file matching currently use `filepath.Match` at [internal/query/files.go:147](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:147) and [internal/query/files.go:170](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:170).
- The pre-scan rejection test is meaningful because iteration begins only after validation at [internal/query/files.go:153](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:153).
- Testing root, nested, non-recursive, brace, malformed, and literal-metacharacter behavior gives strong regression coverage.
- The plan correctly preserves `filepath.ToSlash`, visible at [internal/query/files.go:162](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:162), and chooses the slash-oriented matcher.

### Concerns

- **HIGH — Task 1 cannot produce the specified RED observation.** The RED test directly imports and calls `doublestar.Match` ([04-02-PLAN.md:161](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-02-PLAN.md:161)), but `doublestar` is absent from the current `go.mod`; the dependency is added only in Task 2. Consequently, `go test` will fail during package compilation before any subtest executes. The Task 1 gate requires at least seven observed subtests ([04-02-PLAN.md:190](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-02-PLAN.md:190)), making the plan internally unsatisfiable as ordered.

### Suggestions

- Add `doublestar/v4` before the RED run while leaving production code on `filepath.Match`; the engine-level nested and brace cases will still be meaningfully RED.
- Alternatively, move the direct matcher-contract subtest into Task 2 after dependency installation. Keep the engine regression suite as Task 1’s RED proof.

### Risk Assessment

**HIGH.** Execution stops in Task 1 unless the dependency/test ordering is corrected.

---

## 04-03 — GetHealth RPC

### Summary

The plan cleanly adds an additive diagnostic RPC without expanding the frequently fetched `GetStatusResponse`. Naming, field freezing, handler structure, and method-set guards are well covered.

### Strengths

- The existing service has exactly ten methods, with `GetPermalink` last at [internal/uiproto/uiv1/ui.proto:50](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:50), so the eleventh-method assertion is grounded.
- The plan correctly updates both the positive method-set fixture and its count while leaving the negative mutating-verb guard unchanged.
- `withEngine` usage, explicit mapping, commit validation, nil/non-nil mismatch tests, and non-degraded behavior cover the important backend seams.
- It correctly records rather than conceals the pre-existing fact that `Engine.Status` already computes worktree mismatch.

### Concerns

- None remaining.

### Suggestions

- No required changes. The human wire-shape checkpoint is justified because field numbers are irreversible.

### Risk Assessment

**MEDIUM.** Proto generation and worktree fixtures are substantial, but the plan has strong drift, reflection, and handler-level verification.

---

## 04-04 — Workbench Tabs, Impact, and Callees

### Summary

The plan builds coherently on the tracer and fixes the earlier metadata-flow problem by carrying rows and summary through one guarded result lineage.

### Strengths

- The existing wire confirms Impact returns `node_count`, `edge_count`, and affected locations together ([internal/uiproto/uiv1/ui.proto:290](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:290)).
- The different server-side depth behavior is preserved: Impact clamps at [internal/query/validate.go:71](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/validate.go:71), while no client duplicate is introduced.
- The proposed `AnalysisResult<TSummary>` prevents a late superseded response from updating summary data independently of rows.
- Tests distinguish route remount, `goto`, RPC rerun, and status-gate refetch; these are separate mechanisms and appropriately tested separately.

### Concerns

- None remaining.

### Suggestions

- No required changes.

### Risk Assessment

**MEDIUM.** Frontend state orchestration is complex, but dependencies and failure paths are explicit and test-backed.

---

## 04-05 — Health View

### Summary

The cycle-2 `commitSha` widening now makes snapshot comparison implementable without changing the five-member verdict model. One construction site remains omitted from the formal modification set.

### Strengths

- The current type indeed discards the raw SHA, retaining only `CommitKnowledge` at [web/src/lib/status.ts:36](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:36) and [web/src/lib/status.ts:49](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:49). Adding `commitSha` is therefore the correct fix.
- The three typed production/helper constructors named in the plan exist at [web/src/routes/+layout.svelte:30](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:30), [web/src/routes/browse/+page.svelte:47](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/browse/+page.svelte:47), and [web/tests/degrade-states.test.ts:17](/Volumes/Code/github.com/seanb4t/codegraph-go/web/tests/degrade-states.test.ts:17).
- Depending on both 04-01 and 04-03 is sound: 04-01 edits `status.ts` in wave 1, while 04-05 runs in wave 2.
- The two-suite verification command is discriminating: each JSON result has its own minimum and equality check, followed by both captured exit statuses.
- The health page’s ordering, mismatch presence/absence, snapshot agreement/disagreement, and single-call behavior are tested rather than left to visual judgment.

### Concerns

- **MEDIUM — the modification manifest omits a known required construction site.** The structural status-gate stub at [web/tests/browse-page.test.ts:178](/Volumes/Code/github.com/seanb4t/codegraph-go/web/tests/browse-page.test.ts:178) declares `{ verdict, commit }` and emits an object without `commitSha` at line 180. The plan notices this in `read_first` and says to modify it if `pnpm check` fails, but `web/tests/browse-page.test.ts` is absent from `files_modified`. Thus the implementation instructions and plan metadata disagree.

### Suggestions

- Add `web/tests/browse-page.test.ts` to `files_modified` and explicitly require its stub callback/emitted object to carry `commitSha`.
- Keep `pnpm check` as the final authority in case additional construction sites emerge.

### Risk Assessment

**MEDIUM.** The substantive cycle-2 fix is sound; the remaining issue is plan-manifest completeness rather than architectural correctness.

---

## 04-06 — Multi-file Affected Workbench

### Summary

The cycle-2 below-minimum correction is now complete: the shared mechanism owns cancellation and invalidation while each consumer clears its own visible state.

### Strengths

- The existing controller proves the required three-part behavior: abort at [web/src/lib/search.ts:219](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/search.ts:219), request-ID invalidation at line 223, and visible-state clearing at line 224.
- The new `onBelowMinimum` contract preserves that behavior without duplicating the length rule in both clients.
- Settling the original promise after abort is the correct way to prove the identity bump, since abort alone cannot prevent a non-cooperative promise from resolving.
- Search tests remain byte-unchanged but are no longer overstated as sufficient coverage.
- Affected’s existing wire shape supports the proposed summary flow: repeated input files and `affected_tests` are defined at [internal/uiproto/uiv1/ui.proto:307](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:307).

### Concerns

- None remaining.

### Suggestions

- No required changes.

### Risk Assessment

**MEDIUM.** The refactor touches subtle async behavior, but the plan now explicitly tests coalescing, overlap, stale resolution, below-minimum invalidation, and disposal.

---

## 04-07 — Drift Guard, Render Measurement, and Bundle Refresh

### Summary

The plan correctly makes vendored-source drift non-vacuous and scheduled without placing a live-network dependency in required CI. The cycle-2 bootstrap fix is mostly sound, but its acceptance gate does not verify the complete required step order.

### Strengths

- The action-pin extraction matches the real formatting in CI: checkout appears as `actions/checkout@<40 hex>` at [ci.yml:51](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:51), setup-node at [ci.yml:132](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:132), and Node 24 at line 134.
- The four `test -n` checks close the empty-pattern hole for `CO`, `SN`, `CL`, and `RL`.
- The workflow-shape reasoning is correct: `uses:` steps have empty `Run` and are accepted immediately by [internal/upgrade/taskfile_shape_test.go:1399](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1399).
- Disk-derived subject enumeration, file/component floors, two RED demonstrations, and per-family determinism probing form a strong non-vacuity story.
- Keeping wall-clock assertions behind an opt-in target avoids making required Vitest runs load-sensitive.

### Concerns

- **MEDIUM — bootstrap ordering is only partially verified.** The plan requires checkout → install-task → setup-node → drift run, but the acceptance command records line numbers only for checkout and the run ([04-07-PLAN.md:418](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-07-PLAN.md:418)). It merely checks that install-task, setup-node, and `node-version: "24"` occur somewhere in the workflow. A workflow with setup-node or install-task after `task web:components:drift` would pass this gate but fail its scheduled execution. The repository’s workflow-shape tests validate run-body form and population, not step dependency order ([internal/upgrade/taskfile_shape_test.go:1453](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1453)).

### Suggestions

- Extract line numbers for install-task and setup-node as well, positively assert all are non-empty, then require:
  `checkout < install-task < setup-node < run`.
- Optionally verify `node-version: "24"` belongs to the extracted setup-node step rather than merely appearing elsewhere in the file.

### Risk Assessment

**MEDIUM.** The design is strong, but the bootstrap acceptance gate currently permits a workflow that is syntactically green yet operationally broken.

---

# Overall Assessment

The plans are substantially converged and, aside from 04-02’s dependency-ordering defect, are implementation-ready. The cycle-2 `commitSha`, workflow-bootstrap, and below-minimum fixes are conceptually correct. Remaining work is narrow:

- **Unresolved HIGH concerns: 1**
- **Unresolved actionable MEDIUM/LOW concerns: 2**

---

## Consensus Summary

Single grounded reviewer this cycle (Codex, source-grounded with `file:line` citations
throughout). Its findings were independently re-verified in-tree by the orchestrator before
being recorded here; all three stand.

### Agreed Strengths

- The three cycle-2 fixes are **correct**, not merely present. The `commitSha` widening is the
  right resolution and its enumeration of construction sites is accurate; the drift-workflow
  guard construction genuinely closes the empty-pattern hole; the `onBelowMinimum` contract
  preserves all three parts of the below-minimum semantics without duplicating the length rule.
- 04-01, 04-03, 04-04 and 04-06 now have **no remaining concerns at any severity**.
- Verification gates across the phase are discriminating rather than ceremonial — separate
  floors per suite, both exit statuses `&&`-chained, negative greps carrying explicit positive
  controls, and present/absent assertions in both directions for every warning-style UI element.

### Agreed Concerns

- **HIGH (04-02, dependency ordering).** Task 1's RED test calls `doublestar.Match` directly
  (`04-02-PLAN.md:161-168`), but `github.com/bmatcuk/doublestar/v4` is **not in `go.mod`**
  (confirmed: `rg doublestar go.mod` → no match) and is added only by Task 2 step (a)
  (`04-02-PLAN.md:217`). The package therefore fails to compile, `go test` reports zero
  subtests, and Task 1's gate — `test "$(rg -c '^ *--- (PASS|FAIL)' /tmp/q-red.log)" -ge 7`
  (`04-02-PLAN.md:190`) — cannot be satisfied. The plan additionally instructs the executor to
  record that `escaped_metacharacter_is_literal` *already passes*, which is impossible.
  **Provenance:** this subtest was introduced by the cycle-1 revision `08da6afc` as the partial
  mitigation for 04-06's unescaped-metacharacter HIGH (see 04-02's cycle-1 dispositions table,
  row 3). Cycle 2 did not touch 04-02 and did not catch it. It is dispositioned nowhere.
- **MEDIUM (04-05, manifest completeness).** `web/tests/browse-page.test.ts:178-180` carries a
  fourth status-gate construction site. The plan acknowledges it — but only in **Task 3's**
  `read_first` (`04-05-PLAN.md:377`), as a conditional ("if Task 1's widening makes that stub
  fail `pnpm check`, add the new field there too"), while the widening happens in **Task 1**,
  and the file is absent from `files_modified`. Orchestrator note that qualifies the severity:
  the stub declares its own structural param type and is passed through an untyped Svelte
  context `Map`, so TypeScript method-parameter bivariance means `pnpm check` will most likely
  **not** fail on it — making this a manifest/instruction disagreement and a stale mock rather
  than a compile break. Real but closer to LOW than MEDIUM.
- **MEDIUM (04-07, partial order assertion).** The bootstrap gate (`04-07-PLAN.md:419`)
  extracts line numbers only for checkout (`CL`) and the run step (`RL`) and asserts
  `CL -lt RL`. `install-task`, `setup-node` and `node-version: "24"` are only asserted to occur
  *somewhere* in the file. A workflow placing `install-task` or `setup-node` **after**
  `task web:components:drift` passes this gate and fails every scheduled run — the exact
  failure mode the cycle-2 fix exists to prevent. The repo's own workflow-shape tests validate
  run-body form and job population, not step order.

### Divergent Views

None — a single grounded reviewer ran this cycle.

### Orchestrator observations (NOT counted as actionable)

- `04-06-PLAN.md:312` runs `file-search.test.ts`, `debounced-rpc.test.ts` and `search.test.ts`
  in one vitest invocation behind a single `numTotalTests < 8` floor. `search.test.ts` alone
  has **13** tests today, so that numeric floor is non-discriminating for the two new suites —
  the very "surplus masks shortfall" property 04-05 was given separate floors to avoid. It is
  **not** actionable because the adjacent acceptance criteria at `04-06:321`, `:323` and `:324`
  independently require named tests and wired callbacks in both new files by content grep, and
  `numPassedTests !== numTotalTests` still enforces green. Worth tightening if the maintainer
  is editing 04-06 anyway; not a blocker on its own.
- `04-07-PLAN.md:92` cites `ci.yml:130-133` for the Node 24 pin; the `node-version: "24"` line
  is actually `ci.yml:134`. Prose citation only, no gate depends on it.
- No `/tmp/*.json|log` filename collides across the seven plans, so parallel wave execution
  cannot cross-contaminate verify artifacts.
