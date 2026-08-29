---
phase: 4
cycle: 2
reviewers: [codex]
reviewed_at: 2026-08-29T21:35:45Z
plans_reviewed:
  - 04-01-PLAN.md
  - 04-02-PLAN.md
  - 04-03-PLAN.md
  - 04-04-PLAN.md
  - 04-05-PLAN.md
  - 04-06-PLAN.md
  - 04-07-PLAN.md
revision_under_review: 08da6afc
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 4 (Cycle 2)

Convergence cycle 2. Cycle 1 raised 9 HIGH + 11 actionable non-HIGH concerns; the plans
were revised in `08da6afc` and each plan now carries a `## Cycle-1 cross-AI review
dispositions` table. This cycle assesses the REVISED plans and counts only what REMAINS
unresolved.

The reviewer was given the closed-issue list (the `RC=$?` shape, the `web:components:drift`
enumeration pathspec, the four Go-side `&&`-list gates, `GetHealth`/`mutatingVerbs`/`10`→`11`,
`proto:drift`'s `nfiles -lt 4` file floor, `@tanstack/svelte-table@9.2.4`, the `doublestar`
match, `Engine.Status`'s pre-existing worktree cost, and `devDependency` placement) and did
not re-raise any of them.

## Codex Review

# Cycle 2 Plan Review

## Summary

The revision substantially improves the phase plans. The nine cycle-1 HIGH findings are mostly closed with concrete mechanisms and discriminating tests: generic table reuse, route-scoped navigation identity, shared analysis metadata, correct verify gates, literal glob escaping, and non-vacuous drift enumeration are all addressed convincingly. One new blocking inconsistency remains in 04-05: the snapshot-agreement design compares a `known|unknown` enum with a commit SHA, which cannot work against the current `IndexStatus` type. Two additional execution risks remain in 04-06 and 04-07. Overall, the plans are close to executable but not yet converged.

## Plan-by-Plan Assessment

### 04-01 — Workbench tracer

Well-designed and materially improved. The generic `DataTable<TRow>` plus caller-provided `getRowId` correctly supports both graph locations and health count rows. The route-scoped identity approach fits the existing architecture: the layout passes every reactive URL through the single `navigationIdentity` function at [web/src/routes/+layout.svelte:47](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/routes/+layout.svelte:47), while the current normalizer deletes view-local parameters before sorting at [web/src/lib/status.ts:81](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:81). Keeping `/workbench` exclusions route-specific avoids changing Browse behavior.

No remaining concern found.

### 04-02 — Recursive file glob

The plan correctly changes both matching sites, preserves the pre-scan validation position, and builds a genuinely discriminating fixture with root-level, nested, and second-language files. The instrumented-reader test closes the earlier weakness where error text alone could not prove validation occurred before scanning.

No remaining concern found.

### 04-03 — GetHealth RPC

The `GetHealth` naming, exact method-set update, additive proto discipline, mapper convention, and local worktree fixture are coherent with the live source. The plan also correctly treats the existing `Engine.Status` worktree-detection cost as an acknowledged pre-existing issue rather than pretending D-02 newly eliminates it.

No remaining concern found.

### 04-04 — Four-tab Workbench

The revised `AnalysisResult<TSummary> = { rows, summary }` contract correctly keeps rows and metadata on the same request-generation lineage. The depth and limit tests now observe the actual status-gate call count, which is stronger than merely checking that `goto` was not invoked.

No remaining concern found.

### 04-05 — Health view

The ordering, warning-presence/absence, generic count table, and single-verdict design are strong. However, the newly added snapshot-agreement mechanism is incompatible with the existing status model.

- **HIGH — NEW:** `snapshotAgreement` cannot compare the two commit SHAs as planned. The plan directs `describeFreshness` to compare `IndexStatus.commit` with `GetHealthResponse.commitSha` at [04-05-PLAN.md:146](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-05-PLAN.md:146), and later requires test gates carrying commits such as `aaa…` at [04-05-PLAN.md:326](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-05-PLAN.md:326). In the live implementation, however, `IndexStatus.commit` is `CommitKnowledge = 'known' | 'unknown'`, not the SHA itself ([web/src/lib/status.ts:32](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:32), [web/src/lib/status.ts:36](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:36)). `classifyStatus` intentionally discards the actual SHA and retains only its presence at [web/src/lib/status.ts:49](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/status.ts:49). As written, the implementation either fails type-checking when tests assign `aaa…`, or compares `"known"` to a SHA and reports false disagreement permanently. This revision introduced the snapshot feature but did not add `web/src/lib/status.ts` or its tests to the plan’s modified files.

### 04-06 — Multi-file Affected flow

Extracting a generic debounce/abort/request-identity mechanism is the right response to the cycle-1 contradiction, and keeping `search.test.ts` byte-unchanged is a useful regression constraint. One existing semantic is not fully represented in the extracted contract or tests.

- **MEDIUM — PARTIALLY RESOLVED cycle-1 extraction concern:** the plan does not explicitly preserve the “backspace below minimum while a request is already in flight” behavior. The current controller aborts the live request, increments `liveRequestId`, and clears results when the query becomes shorter than the minimum ([web/src/lib/search.ts:215](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/search.ts:215)). The proposed generic options expose only `dispatch`, `onResult`, and `onFailure` ([04-06-PLAN.md:184](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-06-PLAN.md:184)), while its test list covers a short initial query but not an in-flight long→short transition ([04-06-PLAN.md:175](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-06-PLAN.md:175)). The unchanged existing tests cover pending disposal and overlapping valid queries, but not this long→short case ([web/tests/search.test.ts:293](/Volumes/Code/github.com/seanb4t/codegraph-go/web/tests/search.test.ts:293)). A faulty extraction could therefore let an old result land after the picker was cleared while every mandated test remains green.

### 04-07 — Drift, measurement, and bundle refresh

The revised disk enumeration, two structural floors, two RED demonstrations, and per-family determinism probe are excellent. Registration in both `inScopeWorkflowFiles` and `inScopeJobs` is consistent with the live guards: the existing lists are at [internal/upgrade/taskfile_shape_test.go:152](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:152) and [internal/upgrade/taskfile_shape_test.go:1526](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:1526).

- **MEDIUM — NEW:** the scheduled workflow is underspecified and may never reach its only `run:` step. The plan mandates checkout-compatible pinned actions in general, but does not explicitly require checkout, the repository’s local Task installer, or Node 24/Corepack setup before `task web:components:drift` ([04-07-PLAN.md:326](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/04-query-workbench-index-health/04-07-PLAN.md:326)). Existing workflows install Task through the checked-out local action, for example [corpora.yml:186](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/corpora.yml:186), and the JS path deliberately sets up Node 24 because Corepack is the pnpm-version mechanism ([ci.yml:121](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:121)). The proposed acceptance criteria run `actionlint` and structural registration tests, but none executes the workflow environment. A syntactically valid workflow lacking these prerequisites would be registered correctly yet fail every scheduled run.

## Strengths

- All live frontend verify commands now combine the test-count floor and original exit status correctly.
- `ROUTE_LOCAL_PARAMS` preserves the single-normalizer architecture and avoids globally suppressing Browse parameters.
- `AnalysisResult<TSummary>` eliminates metadata side channels and keeps late-response protection meaningful.
- The glob plan tests exact path sets, malformed-input scan ordering, root preservation, and matcher escape semantics.
- The drift plan now derives its full subject population from tracked files and proves both the comparison and enumeration floors RED.
- The render-cost threshold is opt-in, avoiding a wall-clock assertion in required PR CI.
- Workflow registration targets the correct fixtures and deliberately leaves `requiredCheckNames` unchanged.

## Suggestions

- Change the status model to retain the actual validated commit SHA alongside `CommitKnowledge`, or remove snapshot comparison and explicitly defer it. If retained, add `web/src/lib/status.ts` and `web/tests/status.test.ts` to 04-05’s files and verify backward behavior.
- Add a generic-controller test for: dispatch a valid term, leave it unresolved, set a below-minimum term, assert the signal is aborted, then settle the old promise and assert no result lands. Specify how callers clear their visible state on that transition.
- Spell out the scheduled workflow bootstrap: checkout, install Task via `./.github/actions/install-task`, set up Node 24/Corepack, then run the exact Task target. Add a workflow-shape assertion for these prerequisites or perform a local runner-equivalent smoke check.

## Risk Assessment

**HIGH.** Most architectural and verification risks are now well controlled, but 04-05 currently specifies an impossible snapshot comparison against the live `IndexStatus` type. That blocks a clean implementation and could otherwise produce a permanently false warning. The debounce transition and workflow bootstrap issues are narrower but should also be corrected before execution.

CYCLE2_REMAINING: high=1 actionable_non_high=2

---

## Consensus Summary

Single grounded reviewer this cycle (Codex, prompt-fed with repo access; the review cites
`file:line` evidence throughout and carries no `[reviewed-without-repo-access]` or
`[reviewed-without-source-citations]` marker). Every finding below was independently
re-verified by the orchestrating agent against live source before being counted; the
verification notes are recorded inline.

**Verdict: 1 HIGH remains (NEW, introduced by the revision), plus 2 actionable MEDIUMs
(1 NEW, 1 a partially-resolved cycle-1 concern).** Seven of the nine cycle-1 HIGHs are
confirmed fully closed with working mechanisms; the eighth and ninth are closed but their
closure introduced the two new findings below.

### Agreed Strengths

- All live `<automated>` verify gates now combine the test-count floor with the original
  exit status as ONE `&&`-chained final command — no vacuous gate remains.
- `ROUTE_LOCAL_PARAMS` (04-01 step d2) preserves the single-normalizer architecture
  (`web/src/lib/status.ts:79-87`) instead of adding a second identity function, and scopes
  the exclusion to `/workbench` so Phase 3's shipped Browse behavior is untouched. The
  assertion iterates `WORKBENCH_PARAM_KEYS` rather than a hand-written array
  (04-01-PLAN.md:514-515, :558), so a missing key fails per key. **Verified:** the layout's
  sole navigation trigger is `navigationIdentity(page.url)` at
  `web/src/routes/+layout.svelte:47`, and `web/tests/status.test.ts` is in 04-01's file set.
- `AnalysisResult<TSummary> = { rows, summary }` (04-04) removes the metadata side channel
  and keeps rows and summary on one request-generation lineage.
- 04-02's instrumented-reader test proves validation ordering rather than inferring it from
  error text.
- 04-07's drift target derives its population from `git ls-files` and proves BOTH the
  comparison and the enumeration RED. Workflow registration in `inScopeWorkflowFiles`
  (`internal/upgrade/taskfile_shape_test.go:1526`) and `inScopeJobs` (`:152`) is the correct
  and complete pair — **verified:** `TestWorkflowFilePopulationMatchesDisk` requires every
  on-disk workflow to appear in exactly one of `inScopeWorkflowFiles` /
  `workflowFileExceptions` (`:1624-1664`), and `TestInScopeJobsPopulationMatchesDisk` requires
  every job in an in-scope file to be registered. `requiredCheckNames` is correctly left alone.
- Extracting `web/src/lib/debounced-rpc.ts` rather than configuring `createSearchController`
  twice is the right resolution of the cycle-1 self-contradiction — `createSearchController`
  hard-codes two symbol-specific RPCs (`web/src/lib/search.ts:129-231`) and is genuinely not
  parameterisable in place.

### Agreed Concerns

**HIGH — NEW (04-05): `snapshotAgreement` compares a `CommitKnowledge` enum against a commit
SHA; as specified it can never work.**

04-05-PLAN.md:153-157 directs `describeFreshness` to compute
`snapshotAgreement: 'agree' | 'differs' | 'unknown'` "by comparing the passed-in
`IndexStatus.commit` with the response's `commitSha`: equal and both non-empty → `agree`".

Independently verified against live source:
- `IndexStatus.commit` is typed `CommitKnowledge = 'known' | 'unknown'`
  (`web/src/lib/status.ts:33`, `:36-39`) — it is never a SHA and never an empty string.
- `classifyStatus` deliberately discards the SHA and keeps only its presence:
  `const commit: CommitKnowledge = response.commitSha ? 'known' : 'unknown';`
  (`web/src/lib/status.ts:49`).
- No plan in this phase changes that type. `rg 'IndexStatus|CommitKnowledge|status\.ts'`
  across all seven plans shows 04-01 touches `navigationIdentity` only (04-01-PLAN.md:622:
  "the exported signature ... unchanged"); 04-05 lists `status.ts` under `read_first`
  (04-05-PLAN.md:120) and NOT under any task's `files`, and its artifact list
  (04-05-PLAN.md:395-405) does not include `status.ts` or `web/tests/status.test.ts`.

Consequences as written: whenever GetHealth returns a real SHA and the gate saw one,
`'known' !== '<sha>'` → `snapshotAgreement === 'differs'` **permanently**, lighting the
`health-snapshot-differs` notice on every healthy load. That is precisely the
"permanently on, which trains the user to ignore it" failure mode 04-05-PLAN.md:131-135
introduces the blank-roots `hasWorktreeMismatch` case to prevent. Alternatively, the Task 3
test that constructs "a stub gate at commit `aaa…`" (04-05-PLAN.md:326-328) fails
`pnpm check` — itself an acceptance criterion (04-05-PLAN.md:192) — because `'aaa…'` is not
assignable to `CommitKnowledge`. Either way the plan is not executable as written.

Fix requires an explicit decision, not an executor judgement call: either (a) widen
`IndexStatus` to carry the validated SHA alongside `CommitKnowledge` — which means adding
`web/src/lib/status.ts` and `web/tests/status.test.ts` to 04-05's `files`, stating that
`classifyStatus`'s existing outputs are unchanged, and confirming D-04 still holds; or
(b) drop `snapshotAgreement` and record the two-snapshot problem as an explicit deferral
with rationale. Threat entry T-04-21 (04-05-PLAN.md:379) leans on `describeFreshness`
rendering the commit, so option (b) must say what T-04-21's mitigation becomes.

**MEDIUM — NEW (04-07): the scheduled `components-drift.yml` job is specified as "one job,
one meaningful `run:` step" with no bootstrap, so it would register cleanly and fail every
scheduled run.**

04-07-PLAN.md:326-341 mandates the triggers, the single `task web:components:drift` run body,
and full-SHA action pinning — but never requires `actions/checkout`, the repo's local
`./.github/actions/install-task`, or `actions/setup-node` (Node 24 / Corepack, which is how
this repo resolves the pinned pnpm version). Verified against live workflows: every existing
Task-invoking job pairs `actions/checkout@df4cb1c0…` with `uses: ./.github/actions/install-task`
(`.github/workflows/ci.yml:51`, `:65`; `.github/workflows/corpora.yml:159`, `:187`), and the JS
path additionally pins `actions/setup-node@8207627…` with `node-version: "24"`
(`.github/workflows/ci.yml:130-133`) because Corepack is absent from Node 25+. The drift target
shells out to `pnpm dlx shadcn-svelte@1.5.1`, so all three are required.

None of 04-07's acceptance criteria (04-07-PLAN.md:371-386) would catch the omission:
`actionlint` validates schema, not runtime prerequisites, and the three registration tests are
structural. Worth stating explicitly in the plan that adding these bootstrap steps does NOT
violate the "one meaningful `run:` step" rule — `checkStepInvokesTask`
(`internal/upgrade/taskfile_shape_test.go:1439-1472`) only inspects steps that have a `run:`
body, and all three bootstrap steps are `uses:` steps, so no `runBodyExceptions` entry is needed.

**MEDIUM — PARTIALLY RESOLVED (04-06): the debounced-rpc extraction does not preserve or test
the "backspace below minimum while a request is in flight" behavior.**

`web/src/lib/search.ts:213-224` does three things when the query drops below
`SEARCH_MIN_CHARS`: aborts the live `AbortController`, increments `liveRequestId` to invalidate
any in-flight response, and clears `live` + `liveFailure`. The extracted contract exposes only
`{ debounceMs, minChars, dispatch, onResult, onFailure }` (04-06-PLAN.md:184-193) — there is no
callback through which a caller learns the term went below minimum, so the state-clearing half
has no specified home. The new `web/tests/debounced-rpc.test.ts` list (04-06-PLAN.md:175-178)
covers the min-length gate, coalescing, the aborted overlap, out-of-order discard, rejection and
`dispose` — but not the long→short in-flight transition.

Independently verified that the byte-unchanged regression proof does not cover it either:
`web/tests/search.test.ts:109-148` tests only a one-character query that issues no RPC, and its
cancellation suite (`:151+`) tests long→longer overlap. There is no test in that file for
backspacing below the minimum while a request is outstanding. A faulty extraction could
therefore let a stale result land into a picker the user already cleared while every mandated
test stays green — and 04-06's dispositions table does not record this case.

Fix: add one behavior bullet and one test to Task 1 — dispatch a valid term, leave it
unresolved, set a below-minimum term, assert the first signal is aborted, then settle the old
promise and assert no result lands — and state explicitly where the visible-state clear lives
(caller-side in `search.ts`'s `setQuery`, or a new `onBelowMinimum`/`onReset` option).

### Divergent Views

None — single reviewer this cycle. The orchestrating agent independently re-verified all three
findings against live source and concurs with each; no additional defect was found in a
targeted pass over the revision's other new mechanisms (`ROUTE_LOCAL_PARAMS` including the
deliberate, documented and per-key-asserted treatment of `mode` as route-local; the
`AnalysisResult` contract; and 04-07's `web:components:drift` verify command, whose
`&&`-chained final command and `${NFILES:-0}` guard both behave correctly).

### Cycle-2 counts

    CYCLE2_REMAINING: high=1 actionable_non_high=2
