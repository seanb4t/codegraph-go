---
phase: 5
reviewers: [codex]
reviewed_at: 2026-08-30T13:53:52.000Z
plans_reviewed: [05-01-PLAN.md, 05-02-PLAN.md, 05-03-PLAN.md, 05-04-PLAN.md, 05-05-PLAN.md, 05-06-PLAN.md, 05-07-PLAN.md]
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 5

## Codex Review

## Summary

The seven-plan sequence is unusually thorough and generally traces the repository’s existing seams correctly: `FileGraph` fits the snapshot-based query model, both RPCs extend the guarded read-only surface additively, and frontend work respects the committed-build and single-error-classifier conventions. However, several plan-internal contradictions would block or invalidate execution as written. The most serious are GRF-01’s inability to durably represent a failed or missing measurement, incompatible expansion tests in 05-07, and an unresolved ownership problem for the `FileSymbols` cap. Overall risk is **HIGH until those issues are corrected**.

## Strengths

- **Plan 05-01 — the two-scan rollup matches the storage model.** `schema.Edge` contains node IDs rather than file paths, while file identity lives on nodes, so a node scan followed by an edge scan is structurally necessary. The graphstore reader supports both operations through one consistent snapshot ([store.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/store.go:36), [store.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/store.go:51)). The plan correctly mirrors `BuildReverseAdjacency`’s iterator-close and post-loop-error discipline ([traverse.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/traverse.go:31)).

- **Plan 05-01 — D-08 is addressed against the right repository.** The source confirms that `"package"` nodes are synthetic import targets and are created without `FilePath` ([resolve.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/indexer/resolve.go:21), [resolve.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/indexer/resolve.go:203)). The dedicated own-index regression test is therefore valuable; guava alone cannot exercise this defect. Its PASS-count floor also prevents a skipped own-index test from satisfying the eight-test verification.

- **Plans 05-02 and 05-06 — read-only RPC governance is grounded in real guards.** The existing positive guard compares both method count and exact membership ([readonly_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:53)), while the negative guard inspects every generated handler method against all 19 substrings and fails if it inspects zero ([readonly_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:89), [readonly_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:110)). Updating the fixture and count alongside each RPC is the right mechanism.

- **Plans 05-02 and 05-06 — ordinary handler shape is correct.** `withEngine` opens one engine, defers closure, and translates errors centrally without retaining a store or reader ([handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:28)). Store-lock errors become typed `Unavailable` responses, while only status degrades successfully ([handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:81)). The plans correctly avoid copying the status exception.

- **Plan 05-06 — the research correction is real.** The existing file-detail engine shape contains only `Path` and `Source` ([detail.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/detail.go:44)); it cannot enumerate symbols. A separate RPC is justified rather than widening an every-navigation response.

- **Plan 05-06 — confinement reuse is well founded.** `ValidateRepoRelativePath` is explicitly a wrapper over the existing source-path confinement gate, intended for callers such as permalink-style validation ([node.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:81)). Reusing it avoids a second path-security implementation.

- **Plans 05-03, 05-05, and 05-07 — build drift checks are non-vacuous.** The repository’s drift target reports both source and output counts, enforces positive structural floors, validates all four marker fields, and compares both hashes ([Taskfile.yml](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:1097)). Rebuilding and committing `web/build` in the same change is consistent with the actual guard.

- **Frontend error handling composes the established vocabulary.** `classifyRpcError` already maps not-found, invalid input, typed indexing-in-progress, and unknown errors, and itself fails safely to `unknown` rather than throwing ([rpc-errors.ts](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/rpc-errors.ts:18), [rpc-errors.ts](/Volumes/Code/github.com/seanb4t/codegraph-go/web/src/lib/rpc-errors.ts:28)).

- **No duplicate cross-plan threat IDs were found.** The numbered IDs `T-05-01` through `T-05-38` are unique. `T-05-SC` is also unique, though its nonnumeric format should be checked against any downstream parser expecting `T-NN-NN`.

## Concerns

- **HIGH — Plan 05-04: a missing or dead browser measurement cannot be durably recorded as the claimed fail-closed result.** The comparator is supposed to produce `FAIL` for a missing or nonnumeric metric, but Task 2’s verification independently requires every `metricResults[].measured` value to be a finite number. A missing value therefore makes the artifact fail verification rather than producing a valid, committed FAIL. More importantly, if the browser crashes before the raw observation file exists, the CLI comparator is never invoked at all. The plan says such a session “is recorded as a FAIL,” but specifies no wrapper that catches that failure and writes a reason-bearing observation. This is the principal wrong-answer/absent-answer risk.

- **HIGH — Plan 05-07: the double-click behavior is internally contradictory.** One test says selecting the same expanded file again leaves its child count unchanged; the next says selecting the expanded file again collapses it and returns its child count to zero. Both describe the same second click and cannot pass against one implementation. The intended cache behavior appears to require three actions: expand, collapse, then re-expand from cache.

- **HIGH — Plan 05-06: the symbol cap has no coherent ownership.** Task 2 requires the query engine to apply the cap. Task 3 places the constant in `internal/uiserver/truncate.go`. `internal/query` cannot import `internal/uiserver` without reversing the established dependency direction, and no cap argument is proposed for the engine method. The current architecture keeps the engine independent of the wire package; `Engine` depends on graphstore/schema, while handlers depend on query ([engine.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:10), [handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:26)). As written, implementation requires a duplicated constant, an undeclared parameter, or an import cycle.

- **HIGH — Plan 05-02 contradicts its own prohibition by adding `FileGraphRequest.path`.** The plan prohibits adding a caller-supplied path or filter parameter in v1, then freezes `string path = 1` and tests that it is ignored. Existing request messages do use this future-path pattern—`GetHealthRequest.path` is serialized but unread ([ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:626))—so either choice can be defensible. The plan must choose one policy rather than simultaneously forbidding and requiring it.

- **MEDIUM — Plan 05-03: the pending-request unmount assertion cannot destroy a Cytoscape instance “exactly once” if the canvas has never mounted.** The route renders `GraphCanvas` only after the RPC succeeds. If it is unmounted while the request remains pending, no graph instance should exist, so the correct destroy count is zero. To test destruction exactly once, first resolve and mount the canvas, then unmount it. The pending-request test should instead assert that late resolution creates no instance and applies no elements.

- **MEDIUM — Plan 05-01: `SymbolCount` is specified in a way that counts the file node itself.** Scan one says to increment the file’s `SymbolCount` for every retained node, including the `KindFile` node. That makes a file with three symbols report four. The schema and UI distinguish full nodes from lightweight locations, and file nodes are real graph nodes ([ui.proto](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/ui.proto:83)). If the field is intended to mean declared symbols, count only non-file nodes; otherwise rename it to make the semantics honest.

- **MEDIUM — Plan 05-01: the claimed proof that the SCC implementation is iterative is not reliable.** A 10,000-deep recursive traversal does not necessarily fail in Go because goroutine stacks grow. Passing that test does not prove the implementation is nonrecursive. The explicit prohibition is useful, but the acceptance mechanism needs structural enforcement or a much stronger adversarial depth test with carefully bounded test resources.

- **MEDIUM — Plan 05-04: the observation schema and comparator responsibility are underspecified for two corpora.** Threshold metrics bind to guava, but Task 2 also records this repository’s latency and frame metrics. The proposed comparator accepts a single “observation object,” while the output needs binding metrics, a second corpus entry, collapsed counts, raw TTI samples, and method provenance. Without a concrete schema, the CLI can easily compare the wrong corpus or confuse raw samples with binding medians.

- **MEDIUM — Plan 05-04: the frame-time measurement is insufficiently reproducible.** “Install an rAF sampler and script continuous pan and zoom” does not define the pan distance, zoom range, input cadence, viewport size, browser version, warm-up, or whether ELK layout work overlaps sampling. A recorded threshold is only meaningful if repeat executions run comparable gestures.

- **MEDIUM — Plan 05-03: the measurement seam’s “time to interactive” excludes RPC and transform time.** It starts at `layout.run()` and ends at `layoutstop`. That is accurately a layout duration, but it is not wall-clock route time-to-first-paint. This can be acceptable if the locked threshold explicitly defines it that way; otherwise a slow two-scan RPC or large protobuf decode can be invisible while the artifact reports a fast “time to interactive.” `withEngine` also does not observe cancellation during scans ([handlers.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/handlers.go:43)), increasing the importance of measuring the complete user-visible path.

- **MEDIUM — Plan 05-05: deriving cycle groups from renderer class strings weakens the seam.** The route is instructed to parse the per-cycle discriminator class. Classes are renderer-facing presentation vocabulary. The transform already carries `cycleId`; grouping directly from typed element data would avoid parsing CSS-like strings and keep the route independent of Cytoscape conventions.

- **MEDIUM — Plan 05-07: expansion re-layout threatens the stable-layout requirement that follows in Phase 6.** Re-running the entire ELK layout after each file expansion may move every other node. The plan tests node and edge counts outside the subtree, but not positions. This phase does not own LIV-04, yet the chosen expansion mechanism can make stable live updates materially harder. At minimum, record this architectural consequence.

- **LOW — Plan 05-06 omits `internal/uiserver/truncate.go` from `files_modified`.** Task 1 and Task 3 explicitly place the cap there, but the front matter does not list it. This matters for execution tooling that uses the declared file set for scope validation.

- **LOW — Plan 05-03’s dependency accounting is inconsistent.** It describes three installed runtime dependencies, but only directly adds two and lets `elkjs` remain transitive. That is fine operationally, but package-approval and exact-pin language should distinguish direct exact pins from a transitive version locked only through `pnpm-lock.yaml`.

- **LOW — Existing proto-number guards deserve explicit extension.** `readonly_test.go` already contains a transcribed proto-field-number fixture below the shown method guards ([readonly_test.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/readonly_test.go:130)). The plans protect old responses mainly by “additions-only” diff checks. Adding the new message fields to the established field-number fixture would provide stronger long-term protection for both one-way wire decisions.

## Suggestions

- **05-04:** Define an explicit raw-observation schema with per-metric `{value, method, status, failureReason}`. Permit `value: null` only when `status: "measurement-failed"`, and make the comparator turn that into a valid FAIL result. Add a measurement wrapper that always writes a raw artifact in `finally`, even when browser launch, navigation, seam polling, or sampling fails.

- **05-04:** Separate `bindingObservation` for guava from `additionalCorpora`. The comparator should read only the former, assert its repo and SHA match the threshold, and copy—not judge—the latter.

- **05-04:** Measure both end-to-end route readiness and layout duration, or rename the current metric to `layoutDurationMs`. If only one can bind, make the locked artifact state exactly where the timer begins and ends.

- **05-04:** Commit a deterministic browser-measurement script rather than describing agent-browser actions only. Pin viewport, gesture path, duration, reload semantics, sample calculation, and browser identity.

- **05-07:** Rewrite the interaction tests as: first click fetches and expands; second click collapses with no fetch; third click re-expands from cache with no fetch and no duplicate children.

- **05-06:** Put `MaxFileSymbols` in `internal/query`, or pass a limit into the engine method after validating it. Prefer the former if the cap is an engine response invariant. The wire layer may assert that this cap sits safely below `transportSendMaxBytes`, whose current 16 MiB backstop is real and handler-wide ([truncate.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/truncate.go:36), [server.go](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/uiserver/server.go:121)).

- **05-02:** Resolve the request-path contradiction. Either make `FileGraphRequest` empty, or explicitly exempt its ignored future field from the prohibition and retain the ignored-field regression test.

- **05-01:** Define `SymbolCount` precisely and test a file with a known number of symbol nodes plus its file node.

- **05-03:** Split lifecycle tests into two cases: pending request unmount creates/destroys zero graph instances; mounted canvas unmount destroys exactly one.

- **05-05:** Group cycles by the numeric `cycleId` already carried in application element data, not by parsing classes.

- **05-07:** Capture positions of unaffected file nodes before and after expansion. If full relayout intentionally moves them, record that as an explicit Phase-6 constraint rather than only asserting unchanged counts.

- **05-02/05-06:** Extend the existing proto-field-number fixture for every newly frozen field, in addition to running `proto:drift`.

## Risk Assessment

**Overall risk: HIGH.**

The architecture is sound and the repository evidence supports most of the proposed seams. The risk comes from executable contradictions rather than broad design uncertainty: GRF-01 cannot currently preserve a valid fail-closed result when measurement itself fails; 05-07 specifies mutually exclusive behavior for the same click; and 05-06 places a required cap in a package the engine cannot depend on. Fixing those three issues, plus clarifying the ignored `FileGraphRequest.path`, would reduce the phase to **MEDIUM** risk dominated by browser-performance reproducibility and Cytoscape/ELK interaction behavior.

---

## Consensus Summary

One external reviewer ran (Codex, `gpt-5.6-sol`), with repo access and citing
`file:line` evidence throughout — its findings are source-grounded, not plan-text
restatements. With a single reviewer there is no cross-reviewer consensus to compute,
so the orchestrator independently re-verified every HIGH and the load-bearing MEDIUMs
against the plans and the repository before recording them. Verification results are
noted inline below; one Codex HIGH is downgraded on evidence, and three findings the
orchestrator raised independently are added.

Codex's overall verdict: **HIGH risk**, driven by three executable contradictions
rather than by design uncertainty. The orchestrator concurs on two of the three and
downgrades the fourth-listed HIGH.

The settled ground held: Codex did **not** re-open the Cytoscape-vs-Sigma renderer
selection and did **not** re-raise 05-03's wave ordering. D-02 and D-10 were respected.

### Agreed Concerns (verified)

- **HIGH — 05-04: a failed or absent measurement cannot be durably recorded as a
  fail-closed FAIL.** *Verified.* Task 1's comparator does fail closed on an absent,
  non-numeric or non-finite metric (`05-04-PLAN.md:110-113`). But Task 2's automated
  verify (`05-04-PLAN.md:246`) independently requires **every** `metricResults[].measured`
  to be a finite number and exits 1 otherwise. So a legitimately-FAIL observation whose
  measurement never produced a number fails the verify command rather than being
  committed as a valid FAIL. Worse, if the browser session dies before the raw
  observation file is written, the comparator is never invoked at all and no artifact
  exists — `test -f corpora/graph-render-observations.json` simply fails and the task
  stalls with no recorded verdict. T-05-21 asserts "that outcome IS a measurement …
  recorded as a FAIL with the reason", but no plan text specifies the wrapper that
  makes that true. This is exactly the phase's worst failure mode and the mitigation
  is asserted rather than mechanised.

- **HIGH — 05-06: the `FileSymbols` symbol cap has no coherent owner.** *Verified.*
  Task 2 (`05-06-PLAN.md:267`) places cap application in the query engine
  (`internal/query/filesymbols.go`); Task 3 (`05-06-PLAN.md:298`, `:352`) places the cap
  constant in `internal/uiserver/truncate.go`. The dependency runs one way —
  `internal/uiserver/handlers.go:11` imports `internal/query`, never the reverse — so as
  written this needs a duplicated constant, an undeclared parameter, or an import cycle.
  No cap parameter is proposed on the engine method.

- **HIGH — 05-07: the two "select the same file again" tests contradict each other.**
  *Verified.* `05-07-PLAN.md:177-179` requires that selecting an already-expanded file
  again issues no request **and leaves the child count unchanged**;
  `05-07-PLAN.md:181-183` requires that selecting an expanded file again **collapses it,
  returning its child count to zero**. Both describe the same second click and cannot
  both pass against the single implementation at `05-07-PLAN.md:221` ("if it is expanded,
  collapse it"). The plan's own MUST NOT at `:42` names the intended three-step sequence
  — expand, collapse without a request, re-expand from what was already fetched — so the
  first test is describing the *third* action and is mis-specified.

- **MEDIUM — 05-05: cycle groups are derived by parsing renderer class strings.**
  *Verified.* `05-05-PLAN.md:222-224` makes "the cycle discriminator class Task 1 added"
  the grouping key for the route's focus control, while the transform already carries the
  typed wire `cycleId` (`:39`, `:165`). Grouping from typed element data instead keeps the
  route free of Cytoscape presentation vocabulary and strengthens the GRF-05 seam the
  plan is otherwise careful about.

- **MEDIUM — 05-01: `SymbolCount` counts the file node itself.** *Verified.*
  `05-01-PLAN.md:358-364` increments `SymbolCount` for every retained node, and the
  `KindFile` record is itself a retained node with a non-empty `FilePath`. A file with
  three symbols therefore reports four. No acceptance criterion pins the semantics, so
  the off-by-one would ship silently into the rendered node labels.

- **MEDIUM — 05-04: the observation schema is underspecified for two corpora.**
  *Verified.* The threshold binds on guava (`05-01-PLAN.md:272`), but Task 2 step (e)
  also records this repository's own latency and frame metrics as "a second corpus entry"
  (`05-04-PLAN.md:223-226`), while the comparator takes "an observation object"
  (`:126-128`) with no stated corpus discrimination. Nothing prevents the comparator from
  judging the wrong corpus's numbers against guava's bars.

- **MEDIUM — 05-04: the frame-time gesture is not reproducible.** *Verified.*
  `05-04-PLAN.md:196-200` specifies "a continuous pan and zoom for three seconds" without
  pinning viewport size, pan distance, zoom range, input cadence, warm-up, or browser
  identity. A locked numeric bar compared against an unpinned gesture is only as
  meaningful as the gesture's repeatability.

- **MEDIUM — 05-03: the measurement seam times layout only, not the user-visible path.**
  *Verified.* The seam runs from `layout.run()` to `layoutstop` (`05-03-PLAN.md:361-365`),
  while the threshold key is named `timeToInteractiveMs` (`05-01-PLAN.md:224`). A slow
  two-scan RPC or a large protobuf decode is invisible to the metric under that name.
  Either the locked artifact must state precisely where the timer starts and stops, or
  the metric should be renamed `layoutDurationMs`.

- **MEDIUM — 05-07: full ELK re-layout on expansion may move unrelated nodes.**
  *Verified as unaddressed.* The plan asserts unchanged node and edge **counts** outside
  the expanded subtree (`05-07-PLAN.md:184`) but never positions. Phase 6's LIV-04 wants
  stable in-place updates; this phase does not own it, but the expansion mechanism chosen
  here constrains it. Worth recording as a deliberate consequence.

- **LOW — 05-06 omits `internal/uiserver/truncate.go` from `files_modified`.** *Verified.*
  Tasks 1 and 3 both write the cap constant there; the frontmatter file list
  (`05-06-PLAN.md:7-15`) does not include it, which matters for scope-validating tooling.

- **LOW — 05-03's dependency accounting is inconsistent.** Three runtime dependencies are
  described, but only `cytoscape` and `cytoscape-elk` are added directly; `elkjs` arrives
  transitively and is pinned only through `pnpm-lock.yaml`. The exact-pin and
  package-approval language should distinguish the two cases.

- **LOW — the existing proto-field-number fixture in `internal/uiserver/readonly_test.go`
  is not extended.** New messages are protected mainly by additions-only diff counts;
  transcribing the newly-frozen field numbers into the established fixture would give the
  one-way numbering discipline a durable guard rather than a per-commit one.

### Downgraded on Verification

- **05-02's `FileGraphRequest.path` — Codex rated HIGH; downgraded to MEDIUM.** The
  conflict is real but is a wording conflict, not an executable one. `05-02-PLAN.md:48`
  states flatly "MUST NOT add a caller-supplied filter, path, depth or limit parameter to
  FileGraphRequest in v1", while Task 1 (`:172`) freezes `string path = 1`. T-05-09
  (`:425`) already reconciles the intent — the field is declared and **never read**, with
  `TestFileGraphRequestPathIsIgnored` (`:346`) asserting it — and the same
  declared-but-unread pattern is precedent in this repo at
  `internal/uiproto/uiv1/ui.proto:626` (`GetHealthRequest.path`). The fix is to narrow the
  prohibition's wording to "no parameter that **reaches the store**", not to change the
  design. Left as-is, an executor reading the MUST NOT literally would refuse Task 1.

### Raised Independently by the Orchestrator

- **MEDIUM — 05-04: the frame-sample floor of 60 is asserted in prose but not machine-checked.**
  `05-04-PLAN.md:253` requires the recorded frame-sample count to be at least 60, and
  T-05-19 (`:360`) cites that floor as a named mitigation for "a PASS recorded for a
  measurement that silently did not run". But `frameSampleCount` is not one of the four
  locked threshold metrics (`05-01-PLAN.md:268`: `timeToInteractiveMs`,
  `panZoomFrameTimeMs`, `panZoomFrameTimeP95Ms`, `fileGraphResponseBytes`), so the
  comparator never judges it, and Task 2's automated verify (`:246`) never reads it. The
  mitigation `gsd-secure-phase` will look up by ID has no machine-checkable form. Add the
  count to the verify command with a `>= 60` floor.

- **MEDIUM — `git status --porcelain web/build | wc -l` reports 0 is a vacuous guard.**
  It appears un-positive-controlled at `05-03-PLAN.md:434`, `05-05-PLAN.md:319`, and
  `05-07-PLAN.md:315`. A zero is equally produced by a clean committed bundle and by a
  `web/build` that was never generated. Per rule `84d1gfpywd` every other zero-assertion
  in this plan set is positive-controlled — these three are the exceptions. Chain a
  positive control such as `git log -1 --format= --name-only -- web/build | wc -l`
  reporting at least 1 in the same commit.

- **LOW — only 05-05 carries the explicit PASS precondition; 05-06 and 05-07 do not.**
  D-10 states that "05-05, 05-06 and 05-07 remain gated on" GRF-01's verdict, and 05-04's
  prohibition (`:41`) names only 05-05 and 05-06. `05-05-PLAN.md:100` carries
  `<precondition>corpora/graph-render-observations.json records an overall verdict of
  PASS …</precondition>`; neither 05-06 nor 05-07 carries any `<precondition>`. The gate
  does hold transitively through the wave DAG (`depends_on: [05-04, 05-05]` and
  `[05-05, 05-06]`), so this is defence-in-depth rather than an open hole — but the
  safeguard D-10 names is only actually written down in one of the three plans.

### Verified Clean (adversarial sweeps that found nothing)

These were checked explicitly because the caller flagged them as this repo's known
failure modes. All passed.

- **No duplicate threat IDs.** `T-05-01` … `T-05-38` are defined exactly once each across
  all seven registers, contiguous and non-overlapping. Codex independently reached the
  same conclusion. (Codex notes `T-05-SC` uses a non-`T-NN-NN` shape — worth confirming
  against any downstream parser, but it is unique.)
- **No vacuous `go test -run` gate.** Every one of the five `go test -run` invocations
  (`05-01:396`, `05-01:478`, `05-02:303`, `05-02:394`, `05-06:277`, `05-06:360`) chains a
  `--- PASS` line count with a non-zero floor via a single `&&`, and several plans carry
  an explicit MUST NOT against the `RC=$?` shape that discards the floor. The three prior
  incidents are genuinely closed.
- **Filtered vitest runs are floored the same way.** All seven `pnpm test -- <filter>`
  gates count `✓` lines against a floor, and the successor floors strictly increase
  (Task 1's 6 → Task 2's 11 in 05-05; 6 → 13 in 05-07), which is what proves new cases
  exist rather than the same suite re-running.
- **Every other zero-assertion is positive-controlled.** Roughly twenty `reports 0`
  criteria across the seven plans each pair with a non-zero control over a path the
  search demonstrably reaches. The three `web/build` cases above are the only exceptions.
- **D-08's package-node exclusion is tested against this repository's own index, not
  guava.** `05-01-PLAN.md:328-329` states the corrected 572 / 1,057 figures explicitly,
  names the fact that guava carries zero package pseudo-nodes and therefore structurally
  cannot catch this, and T-05-02 (`:509`) records the own-index regression test as the
  only guard. `TestFileGraphExcludesPackagePseudoNodes` (`:306`) is the fixture test and
  the own-index assertion is separate. `internal/indexer/resolve.go` is explicitly
  off-limits (`:42`, `:47`) and the exclusion lives in the new scan.
- **`mutatingVerbs` is protected and `wantUIServiceMethods` grows correctly.**
  `05-06-PLAN.md:46` forbids modifying, weakening, allowlisting around, reordering, or
  adding an exception to the fixture, and requires renaming the rpc if it collides with
  any of the nineteen substrings. Both 05-02 (`:308`) and 05-06 (`:365`) assert the
  fixture diff is empty, positive-controlled by the file having changed at all.
- **Proto field numbering is additive-only and human-gated.** Both new rpcs sit behind a
  `blocking-human` `checkpoint:decision` before codegen (`05-02` Task 1, `05-06` Task 1),
  both record the approved field list verbatim in their SUMMARY, and both assert the
  proto diff contains zero removed lines, positive-controlled by the added-line count.
- **D-10's stated safeguards are implemented, not merely asserted.** The threshold is
  locked in 05-01 (wave 1) before 05-03 builds anything (wave 3); the lock commit must
  carry exactly one path and the observations file must not yet exist (`05-01:274-275`);
  the comparator is committed code that fails closed; and 05-04 explicitly removes
  "widen the threshold" from the checkpoint's answer set (`05-04:359`). The one gap is
  the missing precondition in 05-06/05-07, noted above.

### Divergent Views

Not applicable — a single reviewer ran. Where the orchestrator's independent verification
disagreed with Codex, the disagreement is recorded under **Downgraded on Verification**
above rather than left as an open divergence.

---

# Cross-AI Plan Review — Phase 5 — Convergence Cycle 2

Reviewed at 2026-08-30T13:53:52Z against the plan text at commit `dc95b06a`
("docs(05): revise phase plans for cross-AI review cycle 1"). Reviewer: Codex
(`gpt-5.6-sol`, reasoning=low), source-grounded with repo access. Cycle 1's findings and
the orchestrator's verification of them are preserved above, unmodified.

## Codex Review (cycle 2)

## Summary

The revision resolves two of the three cycle-1 HIGH findings cleanly. **H-2 holds:**
`MaxFileSymbols` has one proposed owner in `internal/query`, and the existing dependency
direction supports `internal/uiserver` referencing it without duplication or a cycle.
**H-3 holds:** expansion is now an unambiguous three-click sequence — expand, collapse,
re-expand from cache — with cumulative request count fixed at one. **H-1 is substantially
improved but only partially resolved:** measurement failures are now valid, committable
FAIL outcomes, distinct from malformed verification; however, the wrapper still has no
required timeout around browser launch, navigation, seam polling, or sampling. A promise
that hangs never reaches `finally`, despite the plan claiming that hanging sessions always
leave an artifact. The remaining cycle-1 findings are otherwise resolved. Overall risk
remains **HIGH** because the phase's blocking measurement gate still has one path to
producing no verdict.

## Cycle-1 Fix Verification

| Cycle-1 finding | Status | Verification |
|---|---|---|
| H-1: failed/absent measurement could not be durably recorded | **PARTIALLY RESOLVED** | The comparator must accept `{value:null,status:"measurement-failed",failureReason}` as a normal per-metric FAIL (`05-04-PLAN.md:147`, `:201`). The wrapper must write from `finally` and exit zero after recording (`05-04-PLAN.md:223`, `:231`). Task 2's verifier accepts well-formed FAIL artifacts and rejects inconsistent ones (`05-04-PLAN.md:387`). But the plan specifies no timeout or `Promise.race`; a hung await never enters `finally`. |
| H-2: incoherent symbol-cap ownership | **RESOLVED** | The cap is explicitly owned by `internal/query` and referenced as `query.MaxFileSymbols` from the wire layer (`05-06-PLAN.md:44`, `:389`). This matches the actual dependency direction: `internal/uiserver` imports `internal/query` (`internal/uiserver/handlers.go:9`); production files under `internal/query` do not import `internal/uiserver`. The acceptance checks prohibit a second declaration (`05-06-PLAN.md:301`). |
| H-3: contradictory second-click behavior | **RESOLVED** | The cases now describe different actions: click one fetches and expands, click two collapses to zero without fetching, and click three re-expands from cache without fetching (`05-07-PLAN.md:183`, `:187`, `:192`). Exact child counts and a cumulative request count of one are required. |
| `FileGraphRequest.path` contradicted the "no path" prohibition | **RESOLVED** | The prohibition now distinguishes a declared field from one that reaches the store and explicitly requires the handler to ignore it (`05-02-PLAN.md:50`). |
| Pending-request unmount expected a destruction that could not occur | **RESOLVED** | The plan now separates pending unmount — zero constructions and zero applied elements — from mounted-canvas unmount — one destruction (`05-03-PLAN.md:34`, `:430`). |
| `SymbolCount` included the file node | **RESOLVED** | The semantic contract now says declared symbols only and excludes the `KindFile` record (`05-01-PLAN.md:26`); the fixture requires exactly three symbols rather than four. |
| Deep-chain test did not prove an iterative SCC implementation | **RESOLVED** | The deep-chain case is now corroborating evidence, accompanied by a structural check that permits only the function declaration and no self-call plus an explicit-stack check (`05-01-PLAN.md:563`). |
| Observation schema could compare the wrong corpus | **RESOLVED** | The comparator judges only `bindingObservation`, checks its repo and SHA against the threshold, and copies `additionalCorpora` without scoring it (`05-04-PLAN.md:209`, `:387`). |
| Browser gesture was not reproducible | **RESOLVED** | Viewport, reload count, warm-up, sampling duration, pan path, and zoom sweep are locked in the threshold artifact (`05-01-PLAN.md:258`, `:329`). The wrapper is forbidden from carrying defaults (`05-04-PLAN.md:275`). |
| `timeToInteractiveMs` measured only layout time | **RESOLVED** | It is now defined from request issuance through `layoutstop`, including RPC, decode, transform, and layout; `layoutDurationMs` remains a separate non-binding metric (`05-01-PLAN.md:235`, `05-03-PLAN.md:34`). |
| Cycle grouping parsed renderer classes | **RESOLVED** | The route must group from typed `cycleId`, with class parsing prohibited and a class-stripped behavioral fixture required (`05-05-PLAN.md:42`, `:203`). |
| Expansion relayout risk to LIV-04 was unrecorded | **RESOLVED** | The plan explicitly accepts whole-layout movement, measures unaffected-node displacement, and records it as a Phase-6 constraint (`05-07-PLAN.md:31`, `:252`). |
| `truncate.go` missing from 05-06 file scope | **RESOLVED BY DESIGN CHANGE** | The cap no longer belongs there. The plan requires `truncate.go` to remain unchanged and positively controls that absence against changes elsewhere in `internal/uiserver` (`05-06-PLAN.md:301`). |
| Direct versus transitive dependency accounting was inconsistent | **RESOLVED** | `cytoscape` and `cytoscape-elk` are exact direct pins; `elkjs` is explicitly transitive and lockfile-pinned (`05-03-PLAN.md:253`). |
| New proto fields lacked durable numbering guards | **RESOLVED** | Both RPC plans append field fixtures and extend chained fixture-length constants rather than replacing them with literals (`05-02-PLAN.md:290`, `05-06-PLAN.md:357`). |
| M-1: frame sample floor existed only in prose | **RESOLVED** | The observation verifier requires `frameSampleCount >= 60` whenever frame metrics are numeric and requires overall FAIL otherwise (`05-04-PLAN.md:387`, `:395`). |
| M-2: clean `web/build` status was vacuous | **RESOLVED** | All three bundle-producing plans pair clean status with a positive tracked-file count and separately require the hash-based drift guard (`05-03-PLAN.md:472`, `05-05-PLAN.md:332`, `05-07-PLAN.md:353`). |
| 05-06 and 05-07 lacked explicit GRF-01 PASS preconditions | **RESOLVED** | Both now require a PASS artifact and recorded `release` decision (`05-06-PLAN.md:109`, `05-07-PLAN.md:98`). |

## Strengths

- The revised measurement artifact distinguishes three states correctly: numeric
  measurement, declared measurement failure, and broken invocation. That is a meaningful
  improvement over merely allowing `null`.
- The independent observation verifier rechecks corpus identity, metric coverage,
  finite-number/failure shape, per-metric versus overall verdict consistency, and the
  frame-sample floor (`05-04-PLAN.md:387`). A comparator bug is therefore less likely to
  silently bless its own output.
- D-08 remains correctly anchored to this repository rather than guava. The plan requires
  the 572-node/1,057-pair own-index regression (`05-01-PLAN.md:390`). The source confirms
  why: synthetic package nodes are created without `FilePath`
  (`internal/indexer/resolve.go:21`, `:203`).
- All scoped Go test commands retain both `GOTOOLCHAIN=go1.26.5` and a non-zero
  `--- PASS` floor (`05-01-PLAN.md:472`, `05-02-PLAN.md:330`, `05-06-PLAN.md:405`).
- The read-only service guards are being extended rather than weakened. The current source
  reflects exact method-set equality and a separately positive-controlled mutating-verb
  scan (`internal/uiserver/readonly_test.go:53`, `:89`, `:110`).
- The CSP reasoning is consistent with the repository: the server supplies
  `default-src 'self'` without `worker-src` (`internal/uiserver/spa.go:99`), and the plans
  prohibit remote/blob workers.

## Concerns

- **HIGH — H-1 still does not cover a hanging browser operation.** The plan repeatedly
  claims that a hanging or crashing session always records a FAIL (`05-04-PLAN.md:514`),
  but no timeout, abort controller, deadline, or `Promise.race` is specified anywhere in
  05-04. A `finally` block executes after settlement; it does not make a never-settling
  launch/navigation/seam-poll promise settle. This leaves the original "no artifact and no
  verdict" failure mode open for a hang, even though thrown failures are now handled
  correctly.

- **MEDIUM — the live measurement path for the second corpus is underspecified.** Task 2
  first invokes the wrapper with the binding corpus, then says to repeat the run for this
  repository and place it in `additionalCorpora` (`05-04-PLAN.md:304`, `:353`). The
  wrapper's arguments and merge behavior are not concretely defined: it is unclear whether
  the second invocation appends to the first raw artifact, writes a second raw artifact
  that the comparator merges, or overwrites the binding observation. The comparator's
  separation is strong, but the producer-side assembly remains ambiguous.

- **LOW — the wrapper's "no defaults" grep does not cover all locked protocol fields.**
  The check covers viewport, reload, warm-up, sampling, pan, and `zoomSteps`, but omits
  literal assignments for `deviceScaleFactor`, `zoomMin`, and `zoomMax`
  (`05-04-PLAN.md:275`). `readProtocol` tests are intended to catch missing fields, but
  this particular structural guard could still pass if those three values were hard-coded
  in the live browser path.

## Suggestions

- Add a locked timeout policy to `measurementProtocol`, or derive explicit deadlines from
  existing locked durations. Wrap browser launch, navigation, seam polling, each reload,
  and frame sampling with a rejecting deadline. Test a never-settling promise and require
  it to produce a written `measurement-failed` artifact.
- Define the raw observation assembly explicitly. A clean shape would be one wrapper
  invocation per corpus producing separate raw files, followed by a deterministic merge
  step that names exactly one binding file and an array of additional files before invoking
  the comparator.
- Expand the no-default structural check to include `deviceScaleFactor`, `zoomMin`, and
  `zoomMax`, or better, test the live-session configuration builder as a pure function and
  assert it equals `readProtocol(threshold)` field-for-field.

## Risk Assessment

**Overall risk: HIGH.**

Most cycle-1 defects are convincingly resolved, including H-2 and H-3, and the plans now
have strong positive controls around scoped tests, generated bundles, corpus
discrimination, field numbering, and package-node exclusion. The remaining risk is
concentrated rather than broad: the phase's blocking GRF-01 gate still claims to record
browser hangs without specifying a mechanism that can turn a hang into a thrown failure.
Until explicit deadlines make `finally` reachable for never-settling operations, the
phase's worst failure mode remains possible.

---

## Consensus Summary (cycle 2)

One external reviewer ran (Codex, `gpt-5.6-sol`), with repo access and citing `file:line`
evidence throughout. With a single reviewer there is no cross-reviewer consensus to
compute, so the orchestrator independently re-verified Codex's HIGH and both non-HIGH
findings against the plans and the repository, re-ran every standing check, and swept for
the two failure shapes this project has recorded three phases running: (a) a fix that
satisfies its own literal check while the symptom persists, and (b) a vacuous guard
created *inside* a fix for a vacuity finding. One additional finding of shape (b) was
found by the orchestrator and is recorded below.

The settled ground held. Codex did **not** re-open the Cytoscape/ELK renderer selection
(D-02/D-07), did **not** re-raise 05-03's wave ordering (D-10), and did **not** treat the
`uiProtoFieldNumbers` fixture extension as scope creep.

### Cycle-1 Convergence

**15 of 16 cycle-1 findings are RESOLVED.** Independently confirmed by the orchestrator
against the current plan text: H-2 (`05-06-PLAN.md:44`, `:51`, `:389`, `:483` — the cap is
declared once in `internal/query`, the wire layer references `query.MaxFileSymbols`, and
`internal/uiserver/truncate.go` is removed from scope entirely rather than added to
`files_modified`); H-3 (`05-07-PLAN.md:174-192` — three explicitly ordered clicks with a
cumulative request count of exactly 1 held across all three, and the plan names the
mis-specification it corrects); the `FileGraphRequest.path` wording (`05-02-PLAN.md:50`
now scopes the prohibition to READING); M-1 (`05-03-PLAN.md:34`, `:424-430`); M-3
(`05-01-PLAN.md:26`, `:434`, `:481`); M-4 (`05-01-PLAN.md:563` — structural, not just
behavioural); M-5 (`05-05-PLAN.md:42`, `:47`, `:202-205`); M-6 (`05-01-PLAN.md:258-270`,
`:329` — 11 protocol values locked in the same commit as the bars); M-7 (`05-04-PLAN.md:387`
— `bindingObservation` corpus/sha asserted before any comparison); M-8 (`05-01-PLAN.md:230`,
`05-03-PLAN.md:379-385` — `timeToInteractiveMs` widened to request-issued→layoutstop,
`layoutDurationMs` split off as recorded-non-binding); M-9 (`05-07-PLAN.md:250-264` — the
whole-relayout consequence is recorded as an explicit Phase-6 constraint with displacement
numbers captured to size it); the orchestrator's own M-2 (`05-03:472`, `05-05:332`,
`05-07:353` — all three `web/build` zeros now chained to a `git ls-files … -ge 1` positive
control); L-1, L-2 (`05-03-PLAN.md:317`), L-3 (preconditions now in all three of 05-05,
05-06 and 05-07), and L-4 (`uiProtoFieldNumbers` extended in both RPC plans with chained
length constants).

### Open — HIGH

- **HIGH — 05-04: the hang half of T-05-21 is asserted, not mechanised.** *Verified
  independently.* T-05-21 (`05-04-PLAN.md:514`) names the threat as "the measurement
  session itself **hanging** or crashing the browser at corpus scale, leaving no record",
  and the mitigation names exactly one mechanism: "runs the whole session inside one `try`
  whose `finally` ALWAYS writes the raw observation". `finally` runs on settlement. A
  browser launch, `page.goto`, seam poll or rAF-sampling promise that never settles never
  reaches it, and the process holds until something outside the plan kills it — at which
  point there is no artifact, the comparator never runs, `test -f
  corpora/graph-render-observations.json` (`:387`) fails, and Task 2 stalls with no
  recorded verdict. `rg -ni "timeout|deadline|promise\.race|abortcontroller|watchdog"`
  over `05-04-PLAN.md` and `05-01-PLAN.md` returns **zero** matches, so no deadline exists
  anywhere in the locked protocol or the wrapper's specification. This is precisely failure
  shape (a): the fix satisfies its own literal check — `rg -c 'finally'
  web/scripts/graph-measure.mjs` reports at least 1 (`:277`) — while the symptom the
  finding was about persists for the hang case. The crash and throw cases ARE genuinely
  fixed; the hang case is not.

  **What the plan needs:** a `timeoutMs`-class value (or a small set of them) added to the
  locked `measurementProtocol` block in `05-01-PLAN.md:258-270` and to its 11-value
  verify at `:329`; a requirement in `05-04-PLAN.md` Task 1 that every awaited browser
  operation in `graph-measure.mjs` is raced against a rejecting deadline derived from that
  block; and a `graph-measure.test.ts` case feeding a never-settling promise through the
  deadline helper and asserting it yields `failedObservation`'s shape. Without that, the
  `finally` acceptance criterion at `:277` is satisfiable by a script that still hangs.

### Open — Actionable MEDIUM / LOW

- **MEDIUM — 05-04: the producer-side assembly of the two-corpus observation is
  undefined.** *Verified.* Task 2 step (b) (`05-04-PLAN.md:304-309`) runs the wrapper once
  against guava "giving it the corpus repo and sha for the binding entry and a path to
  write the raw observation to", and step (e) (`:353-360`) says to "Repeat the
  time-to-interactive and frame-time measurement against THIS repository's own index and
  record it as an entry in `additionalCorpora`". Nothing states whether that is a second
  wrapper invocation to a second path plus a merge step, an append to the first artifact,
  or a wrapper flag — and a second invocation writing to the same path would overwrite
  `bindingObservation` outright. `failedObservation(reason, metricKeys)` (`:171-178`) is
  specified as producing a `bindingObservation` only, so the failure path for the second
  corpus is undefined too. The comparator's read side is airtight (`:387` reads
  `bindingObservation` and copies `additionalCorpora`); the write side is not.
  **PLAN.md change needed:** name the wrapper's corpus-role argument explicitly in Task 1's
  behavior block, state one raw file per invocation plus a deterministic merge that names
  exactly one binding file and an array of additional files, and add an acceptance criterion
  that the merged artifact carries exactly one `bindingObservation` and at least one
  `additionalCorpora` entry.

- **MEDIUM — 05-06 Task 2's new positive control is unsatisfiable at the point it runs.**
  *Raised independently by the orchestrator — failure shape (b).* The cycle-1 L-1 fix
  replaced "add `truncate.go` to `files_modified`" with a guard that the file is never
  written. At `05-06-PLAN.md:302` that guard reads: "`git diff --name-only
  internal/uiserver/truncate.go | wc -l` reports 0 across this plan's commits,
  positive-controlled by `git diff --name-only internal/uiserver/ | wc -l` reporting at
  least 1". But Task 2's `<files>` (`:243`) is `internal/query/filesymbols.go,
  internal/query/filesymbols_test.go` — Task 2 touches **nothing** under
  `internal/uiserver`, so the positive control reports 0 and the criterion cannot pass at
  Task 2 under either reading of `git diff` (working-tree scope or plan-commit-range
  scope). Task 3's copy of the same guard (`:410`) is sound, because Task 3's `<files>`
  (`:314`) does include `internal/uiserver/handlers.go`. **PLAN.md change needed:** drop
  the criterion from Task 2 (it belongs to the task that writes the wire layer), or
  re-point Task 2's positive control at a tree Task 2 demonstrably modifies — e.g.
  `git diff --name-only internal/query/ | wc -l` reporting at least 1 — while keeping the
  `internal/uiserver/truncate.go` zero.

- **LOW — 05-04's no-defaults structural check covers 8 of the 11 locked protocol
  values.** *Verified.* `05-01-PLAN.md:329` locks and machine-checks exactly eleven:
  `viewportWidth`, `viewportHeight`, `deviceScaleFactor`, `coldReloads`, `warmupMs`,
  `sampleDurationMs`, `panStepPx`, `panSteps`, `zoomMin`, `zoomMax`, `zoomSteps`. The
  acceptance criterion at `05-04-PLAN.md:275` greps for literal assignments to only eight
  of them — `deviceScaleFactor`, `zoomMin` and `zoomMax` are absent from the alternation,
  so a wrapper hard-coding those three would pass the guard whose whole purpose is to prove
  the wrapper carries no gesture defaults. **PLAN.md change needed:** add the three missing
  keys to the alternation at `:275`, or replace the grep with the stronger form Codex
  suggests — a pure `sessionConfig(protocol)` builder asserted field-for-field equal to
  `readProtocol(threshold)`.

- **LOW — two uncontrolled zero-assertions remain in 05-04 (pre-existing, missed in cycle
  1).** `05-04-PLAN.md:281` and `:490` both read "`git diff --name-only
  corpora/graph-render-threshold.json | wc -l` reports 0" with no positive control on
  either. Rule `84d1gfpywd` requires every guard to carry a positive assertion that it did
  its work; these two zeros are equally produced by "the plan did not touch the threshold"
  and by "the diff was scoped to a path that does not exist". Cycle 1 asserted the three
  `web/build` cases were "the only exceptions" — these two were missed, and `git show
  01475f19` confirms they predate the revision rather than being introduced by it.
  **PLAN.md change needed:** chain each to a positive control over a tree the same diff
  demonstrably reaches, e.g. `git diff --name-only corpora/ | wc -l` reporting at least 1
  at `:281` (where the observations file is being written in the same task).

### Verified Clean in Cycle 2 (standing checks re-run)

- **`go test -run` gates.** All six scoped invocations (`05-01:472`, `05-01:557`,
  `05-02:330`, `05-02:422`, `05-06:296`, `05-06:405`) still chain a `--- PASS` line count
  with a non-zero floor via a single `&&`. The `RC=$?` anti-pattern is still explicitly
  prohibited in all seven plans.
- **`GOTOOLCHAIN=go1.26.5`.** Present on every Go command in all seven plans; zero bare
  `go test`/`go build`/`go vet` invocations.
- **`go mod tidy`.** Appears only as a prohibition (`05-06:56`, `05-03:263`).
- **Vitest floors.** All eight `pnpm test --` gates count `✓|√` lines against a non-zero
  floor, and successor floors strictly increase within each plan (05-05: 6→12; 05-07:
  6→15), which is what proves new cases exist rather than the same suite re-running.
- **No upper-bound-only gates.** `rg -n '\-le [0-9]|at most [0-9]|no more than [0-9]'`
  over all seven plans returns nothing — the `<= N` corollary of rule `84d1gfpywd` has no
  violations.
- **Threat IDs.** 39 definition rows across the seven registers, 39 distinct, zero
  duplicates. Unchanged in count from cycle 1; no renumbering occurred.
- **`mutatingVerbs`.** Still protected by an explicit MUST NOT in both 05-02 (`:44`) and
  05-06 (`:47`), with the collision remedy being to rename the rpc. `wantUIServiceMethods`
  grows by exactly one entry per new rpc.
- **D-08.** The package-pseudo-node exclusion is still tested against this repository's own
  index, with the 572/1,057 figures stated and the guava-cannot-catch-this reasoning intact.
- **Wave DAG.** Coherent and unchanged: 05-01 → 05-02 → 05-03 → 05-04 → 05-05 → 05-06 →
  05-07, one plan per wave, `depends_on` consistent with the wave numbers.

### Divergent Views

Not applicable — a single reviewer ran. Where the orchestrator's independent verification
went beyond Codex, the additional finding is recorded above under **Open — Actionable
MEDIUM / LOW** rather than left as an open divergence. The orchestrator concurs with
Codex's HIGH on evidence and with both of its non-HIGH findings.
