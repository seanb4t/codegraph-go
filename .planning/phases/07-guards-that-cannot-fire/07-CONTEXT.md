# Phase 7: Guards That Cannot Fire - Context

**Gathered:** 2026-09-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Four known vacuous guards in this repo are made discriminating, each proven RED against its real failure condition before it is called fixed, with the proofs committed in a mutation log. One fifth guard (the tap App secret-distinctness test) is **deleted, not rewritten** — see D-09. Test and CI-config only. No feature code, no new runtime behaviour, no mutation-testing framework.

In scope: GRD-01, GRD-02, GRD-03, GRD-04, GRD-06 (now four demonstrations), plus deletion of the tautological GRD-05 test. Out of scope: GRD-05 as a rewrite (de-scoped this session), GRD-07, GRD-08 (declined at milestone scoping), the brew-trust docs todo (Phase 12).

</domain>

<decisions>
## Implementation Decisions

### Archtest boundary (GRD-02)

- **D-01:** The archtest forbids `internal/query` from depending on the `internal/indexer` **root** package (the pipeline / discovery path) while explicitly **allowing** the leaf packages `internal/indexer/goextract` and `internal/indexer/nodeid`, which `query` already reaches today for edge-kind constants and node-ID hashing. — **Reversibility:** costly — Phase 10's HLT-05 discovery-exclusion helper must be placed to satisfy this rule (in the indexer or graphstore, never imported from `query`); loosening the rule later would re-open the query-time re-walk that the HLT-05 ruling forbids.
- **D-02:** The forbidden wire-layer set is the **union** of the roadmap's list and the todo's list: `internal/uiserver`, `internal/mcp`, `internal/uiproto`, `connectrpc.com/connect`. The check runs over `query`'s **resolved transitive dependency set** (`packages.NeedDeps`), not over its direct import statements, so a violation smuggled through an intermediary package is caught.
- **D-03:** Positive control: the test asserts that `internal/graphstore` **and** `internal/indexer/goextract` are both present in the resolved set, and reports the number of packages loaded (requiring > 0). graphstore is the Engine's storage seam; goextract is the very leaf D-01 allows, so the control also proves the allow-list path is live.
- **D-04:** The test lives in a new package **`internal/query/archtest/`**, mirroring how `internal/graphstore/archtest` sits beside the package it constrains. Its package doc names threat T-01-18.
- **Consequence for Phase 10 (record for the HLT-05 planner):** the discovery-exclusion helper writes reasons from inside the indexer's discovery path; `query` reads them back through `graphstore` only. `query` may not import the helper.

### GRD-03: dry-run-signed injection guard

- **D-05:** The awk `--key=` injection, the additions-only diff guard, and the new positive assertion are **extracted into one script** under `scripts/` (name at Claude's discretion) taking the committed config path, the output path, and the key path. Both Task targets that share the anchor (`release:dry-run-signed` and `release:rehearse-notarize`) call the script; the goreleaser invocation stays in Task. — **Reversibility:** reversible.
- **D-06:** The positive assertion counts the `--key=` lines **added** in the diff and requires the count to equal **exactly 1**, printing the count. Zero means the anchor no longer matches (the hang case); two or more means a duplicated `sign-blob` block, which is also a config error worth refusing.
- **D-07:** **No** shape test in `taskfile_shape_test.go` asserting that both Task targets call the script. The script's own assertion plus its RED demonstration is the guard. (User: "why are you proposing tests for tests?")
- RED demonstration: run the script against a copy of `.goreleaser.yaml` with the `sign-blob` args block re-indented; runs on any host, no darwin/zig/syft/cosign preconditions.

### GRD-04: post-release-verify conclusion guard

- **D-08:** The test parses `post-release-verify.yml`'s job map and requires **every** job's `if:` to equal the exact disjunct string `github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'` **verbatim**. It reports the number of jobs inspected and requires it to be > 0, with an empty-or-unparseable-document-is-error companion (per the `TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError` precedent). No fixed expected-id list, no normaliser. A new unguarded job fails on its own, not because a list went stale.

### GRD-05: tap App secret-distinctness test — DE-SCOPED

- **D-09:** GRD-05 is **removed from Phase 7**. `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets` (`internal/upgrade/release_workflow_shape_test.go` ~1544) is **deleted** in this phase rather than rewritten. D-16's two-distinct-Apps property rests on documentation and the v0.5.0 one-time proof. — **Reversibility:** reversible — a file-reading version can be added in a later milestone if the property ever matters enough. Rationale (user): the property is low-value, a lying test is worse than none, and rewriting it is not worth the phase's time. REQUIREMENTS.md, ROADMAP.md and PROJECT.md were updated in this session; the 2026-08-10 todo is resolved by the deletion and keeps `resolves_phase: 7`.

### GRD-01: CheckRegression current-metrics positivity

- **D-10:** The current-metrics check sits **immediately after the two existing baseline positivity checks** and before the delta math, in `internal/bench/regression.go`. Two lines mirror the two baseline lines exactly; each returns its own error naming the field and value (shape: `bench: invalid current: PeakRSSBytes must be positive, got 0`). The frame-attribution checks (platform, runner, scratch FS) stay first. The runner in `tools/bench/runner/main.go` needs no change; it already prints metrics on any gate error.
- The RED test replays the **historical** degenerate frame from the Phase 10 audit, not a synthetic one: `CheckRegression(baseline, current, ceiling=1)` with `current.PeakRSSBytes = 0` and an otherwise-matching frame. With `tdd_mode` on, the test lands first and is watched fail against the pre-fix build. A companion case covers `current.FilesPerSec = 0`.

### GRD-06: mutation log

- **D-11:** `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md`, following `03-MUTATION-LOG.md`'s shape: pre-mutation cleanliness gate (`git diff --quiet -- <file>`), the mutation applied, pasted failing output, the revert command, byte-clean proof, green re-run. **Four** entries (GRD-01, GRD-02, GRD-03, GRD-04). GRD-05 gets a one-line record of the deletion decision, not a demonstration.

### Claude's Discretion

- Script name and argument order under `scripts/` for D-05.
- Whether the GRD-02 loader uses `Tests: true` (the precedent does); if `query`'s test variants legitimately pull something forbidden, document why rather than weaken the set.
- Whether GRD-04 extends `TestPostReleaseJobsDeclareCheckoutPolicy` or is a sibling test reusing its parse.
- Exact error wording, test names, and commit granularity.

### Folded Todos

- **release:dry-run-signed's additions-only diff guard passes vacuously** (`.planning/todos/pending/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md`) → GRD-03, D-05..D-07.
- **post-release-verify.yml's event-aware conclusion guard has no test** (`.planning/todos/pending/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md`) → GRD-04, D-08.
- **tap App secret-distinctness test is tautological** (`.planning/todos/pending/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md`) → resolved by deletion, D-09.
- **internal/query dependency-direction has no persisted archtest** (`.planning/todos/pending/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md`) → GRD-02, D-01..D-04.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and scope
- `.planning/REQUIREMENTS.md` §Guard Hardening — GRD-01..04, GRD-06 as scoped; GRD-05 moved to v2 (declined 2026-09-08)
- `.planning/ROADMAP.md` §Phase 7 — goal, success criteria, notes (skip the research pass; historical degenerate frame; query→indexer question resolved by D-01)
- `.planning/ROADMAP.md` §Backlog 999.4 — the original CheckRegression defect record and its reproduction
- `.planning/research/PITFALLS.md` (the 999.4 section, ~line 350) — the fix-pass anti-patterns to avoid: `t.Skip` on zero, a floor satisfiable by another measurement bug

### Todos folded into this phase
- `.planning/todos/pending/2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md` — T-02-08, both anchor users, RED procedure
- `.planning/todos/pending/2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md` — T-02-18, why the disjunct is load-bearing, evidence run IDs
- `.planning/todos/pending/2026-08-10-tap-app-secret-distinctness-test-is-tautological-and-reads-no-workflow.md` — the test being deleted, and why it cannot fail
- `.planning/todos/pending/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md` — T-01-18, "count uses, never the declaration"

### Structural precedents (reuse, do not reinvent)
- `internal/graphstore/archtest/import_graph_test.go` — the `go/packages` archtest shape with `Tests: true`, zero-package guard, positive-control importer, `stripTestVariant`
- `internal/upgrade/bench_workflow_shape_test.go` — parsed-subtree workflow assertions with bespoke YAML structs
- `internal/upgrade/release_workflow_shape_test.go` — `TestPostReleaseJobsDeclareCheckoutPolicy` (~1369, already enumerates the post-release-verify job map); `..._EmptyDocIsError` companions (~1289); the tap test to delete (~1544)
- `.planning/milestones/v0.11.0-phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` — the mutation-log shape GRD-06 follows

### Code under change
- `internal/bench/regression.go` — `CheckRegression`; the two baseline positivity lines D-10 mirrors
- `internal/bench/regression_test.go` — existing table test to extend with the degenerate-current cases
- `Taskfile.yml` — `release:dry-run-signed` (from ~1771; anchor at 1841) and `release:rehearse-notarize` (from ~2367; anchor at 2381)
- `.github/workflows/post-release-verify.yml` — five jobs, each carrying the disjunct on its `if:` line
- `.github/workflows/release.yml:194-195`, `.github/workflows/release-please.yml:83-84` — the two secret pairs (context for the deletion only)

### Rules
- engram rule `84d1gfpywd` — a guard MUST carry a positive assertion that it did its work

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/graphstore/archtest/import_graph_test.go`: copy its loader config, zero-package fatal, positive-control pattern and `stripTestVariant` helper for the new `internal/query/archtest`.
- `internal/upgrade/release_workflow_shape_test.go`: `decodeFullWorkflowDoc` and the job-map enumeration in `TestPostReleaseJobsDeclareCheckoutPolicy` give GRD-04 its parse for free.
- `internal/bench/regression_test.go`: a table test with `errHint` substring matching; the two new degenerate-current cases slot into the same table.
- `03-MUTATION-LOG.md`: the header, per-family sections, and "byte-clean proof" wording to copy for `07-MUTATION-LOG.md`.

### Established Patterns
- Archtests assert over the resolved package graph via `go/packages`, never regex over source (aliased imports, build tags, test variants all evade regex).
- Workflow shape tests assert against the parsed subtree, never a file-wide grep, and every negative assertion has an "empty doc is error" twin.
- Every RED demonstration records the pre-mutation cleanliness gate, the exact mutation, pasted output, and the byte-clean revert.
- `CheckRegression` orders frame attribution (platform, runner, scratch FS) before numeric validity before tolerance math; D-10 preserves that order.

### Integration Points
- `internal/query` currently resolves to `internal/indexer/goextract` (via `expand.go`) and `internal/indexer/nodeid`; the allow-list in D-01 must name both or the archtest goes RED on day one for the wrong reason.
- Both Task targets share one anchor line and one guard block; D-05 replaces both blocks with a single script call.
- `tools/bench/runner/main.go:627` is the only caller of `CheckRegression`; it already wraps and prints on error.

</code_context>

<specifics>
## Specific Ideas

- User rejected "tests for tests": a shape test that asserts a guard is wired is not wanted. The guard's own assertion plus its recorded RED demonstration is the standard.
- User rejected fixing a low-value lying test: when a test cannot fail and the property it names does not matter much, delete it rather than spend the phase rewriting it.
- The GRD-01 test must be the historical frame (`ceiling=1`, `current.PeakRSSBytes = 0`), reproduced from the Phase 10 audit, not a synthetic frame.

</specifics>

<deferred>
## Deferred Ideas

- **GRD-05 file-reading rewrite** — de-scoped; if D-16's two-App distinctness ever needs a machine check, add a test that reads `release.yml` and `release-please.yml` and asserts disjoint, non-empty secret sets. Recorded in REQUIREMENTS.md → v2.

### Reviewed Todos (not folded)
- **brew trust instructions recommend the broader --tap grant** (`.planning/todos/pending/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md`) — matched at 0.6; ROADMAP already routes it to Phase 12 (DOCS-07) as a docs-tail item with no guard of its own.

</deferred>

---

*Phase: 07-guards-that-cannot-fire*
*Context gathered: 2026-09-08*
