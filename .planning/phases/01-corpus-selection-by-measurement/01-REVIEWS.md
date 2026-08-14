---
phase: 1
reviewers: [codex]
reviewed_at: 2026-08-14T13:59:13Z
plans_reviewed:
  - 01-01-PLAN.md
  - 01-02-PLAN.md
  - 01-03-PLAN.md
  - 01-04-PLAN.md
  - 01-05-PLAN.md
  - 01-06-PLAN.md
  - 01-07-PLAN.md
---

# Cross-AI Plan Review — Phase 1

## Codex Review

# Cross-AI Plan Review

## Summary

The plans show unusually strong source awareness: they correctly identify the four status render paths, the actual wire-oracle transcript affected, the repository’s Taskfile workflow guard, and the established Namespace cache mechanism. The wave structure is broadly sensible.

However, several execution-level contradictions prevent the phase from reliably reaching its goal. The most serious are:

- the measurement command’s scope and prerequisites are inconsistent;
- the selection metadata has no defined preservation path through regeneration;
- synthetic fallback is allowed to satisfy criteria that explicitly require measured third-party repositories;
- the drift guard appears designed to trust a precomputed coverage summary instead of deriving the claim from locked measurements;
- cached trees are validated by `HEAD` alone, which does not prove their contents;
- the proposed “tree hash” hashes filenames, not file contents;
- the fetch staging design is not concurrency-safe.

Overall risk: **HIGH until these are resolved.**

---

## 01-01 — Status measurement tracer

### Strengths

- The proposed full edge scan uses a real supported primitive. `IterateEdges("")` explicitly means the entire edge namespace in [internal/graphstore/pebble_store.go:285](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:285), and existing query code already relies on that behavior.
- The snapshot-consistency claim is supported: all reads through a `pebbleReader` share one Pebble snapshot, as documented at [internal/graphstore/pebble_store.go:245](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:245).
- Un-suppressing `FilesByLanguage` is mechanically straightforward because it is already populated by the file scan at [internal/query/status.go:220](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:220) and currently suppressed only by its JSON tag at [internal/query/status.go:57](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:57).
- Keeping the raw tally unfiltered is sound. Filtering it through `RankEdges` would hide new or unexpected stored edge kinds.

### Concerns

- **MEDIUM — The plan introduces an unconditional O(edges) cost to every status call.** Current code deliberately obtains the total from `Meta.EdgeCount` to avoid a full edge scan [internal/query/status.go:202](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:202). This affects CLI status and the MCP tool, not only the spike. Documentation alone does not quantify whether this remains acceptable on large indexes.
- **LOW — The end-to-end verification depends on the repository’s existing index containing `calls > 0`.** That is environmental rather than a property of the implementation. The seeded unit tests are the reliable proof; the repository-index probe should be informational.
- **LOW — Iterator closure is deferred until the entire method returns.** Existing code already defers file and node iterator closure at [internal/query/status.go:224](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:224) and [internal/query/status.go:242](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:242). Adding a third simultaneously retained iterator is probably safe, but explicit closure after each completed scan would reduce retained resources.

### Suggestions

- Add a benchmark or at least record status latency on a representative large corpus before treating the full scan as permanent product surface.
- Close each iterator after its `Err()` check instead of retaining all three to method exit.
- Make the real-repository CLI probe assert only shape and internal consistency; leave exact positive counts to deterministic fixtures.

### Risk Assessment

**MEDIUM.** The mechanism is correct, but it permanently changes status complexity.

---

## 01-02 — Dense mode and render surfaces

### Strengths

- The plan correctly finds all four surfaces:

  - piped text at [internal/query/render_status.go:170](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_status.go:170);
  - MCP markdown at [internal/query/render_status.go:225](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_status.go:225);
  - JSON through [internal/cli/status.go:53](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/status.go:53);
  - styled TTY rendering at [internal/cli/status.go:61](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/status.go:61) and [internal/cli/present/status.go:123](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/present/status.go:123).

- Deriving dense keys from `RankEdges` is exactly the right invariant. `RankEdges` is the current canonical set at [internal/query/rwr.go:21](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/rwr.go:21), with an existing exact-membership test at [internal/query/rwr_test.go:13](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/rwr_test.go:13).
- Applying density once before the CLI output branches is clean and keeps MCP sparse by construction. The MCP handler currently passes raw status directly to `RenderStatusMarkdown` at [internal/mcp/tools.go:536](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/tools.go:536).
- The deterministic tie-break follows the existing count-descending/key-ascending behavior at [internal/query/render_status.go:78](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_status.go:78).

### Concerns

- **LOW — The plan duplicates another sorting helper into the TTY package.** That matches current architecture, but it increases the already acknowledged duplication between [internal/query/render_status.go:78](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_status.go:78) and [internal/cli/present/status.go:54](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/present/status.go:54).
- **LOW — `len(d) >= 9` is weaker than the stated dense contract.** Because unranked kinds are preserved, `>= 9` is valid for behavior, but the CLI test must separately prove every `RankEdges` key is present. Length alone cannot do that.

### Suggestions

- In command-level tests, compare every `RankEdges` member against the decoded map rather than relying on map length.
- Consider a shared exported ordered-count representation later; do not widen this phase merely to remove the present duplication.

### Risk Assessment

**LOW.** The plan maps cleanly onto current code and has appropriate cross-surface tests.

---

## 01-03 — Transcript re-freeze and surface assertions

### Strengths

- The plan correctly rejects the original `resources-read-status.golden` assumption. `call-status` performs a `tools/call` against an indexed fixture at [test/wireoracle/scenarios.go:752](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go:752), while `resources-read-status` reads the static documentation resource without an index at [test/wireoracle/scenarios.go:1321](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go:1321).
- The current transcript confirms that live status markdown is frozen in `call-status.golden`, including its nodes and language blocks at [testdata/wireoracle/transcripts/call-status.golden:2](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/wireoracle/transcripts/call-status.golden:2).
- Capturing to a temporary file before replacement is an important safeguard against destroying the oracle on capture failure.
- Testing MCP sparsity through the real server path is stronger than testing `RenderStatusMarkdown` alone.

### Concerns

- **MEDIUM — “Fewer than `len(RankEdges)` rows” is not the actual sparse invariant.** Sparse means no zero-valued row, not necessarily fewer than all ranked kinds. A legitimate fixture could contain all nine kinds with positive counts and still be sparse. The current small fixture likely has fewer kinds, but the test would encode an incidental fixture property rather than D-05.
- **LOW — Comparing changed transcript count to `HEAD~1` assumes the executor’s commit boundaries exactly match the plan.** That is compatible with GSD atomic commits but brittle during recovery or a dirty branch.

### Suggestions

- Drop the “strictly fewer than `RankEdges`” assertion. Parse the live indexed fixture’s actual edge-kind set or assert only that every rendered row is positive and no absent kind is synthesized.
- Compare the transcript diff against the task’s recorded base commit rather than implicitly assuming `HEAD~1`.

### Risk Assessment

**LOW–MEDIUM.** The re-freeze target and mechanism are correct; one assertion should be made semantic rather than fixture-dependent.

---

## 01-04 — Manifest and fetch path

### Strengths

- Strict repository, SHA, and license validation before shell interpolation is appropriate.
- The XDG resolution avoids `os.UserCacheDir`, matching the explicit cross-platform contract rather than native macOS cache conventions.
- Embedding the SHA in the destination path is a strong content-addressing property.
- Keeping a real `.git` directory genuinely exercises git-aware paths; `IterateEdges` and index status already operate against snapshot-backed stores, while worktree metadata is part of the current status contract.
- The plan correctly recognizes the repository’s existing Namespace cache usage, e.g. [ci.yml:60](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:60) and [bench.yml:112](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:112).

### Concerns

- **HIGH — The proposed reproducibility hash does not hash the tree contents.** The verification hashes `git rev-parse HEAD` plus the output of `git ls-files`; that proves the commit ID and tracked filenames, not file bytes, modes, symlink targets, or working-tree cleanliness. A modified cached file produces the same proposed hash.
- **HIGH — An existing destination is trusted solely by checked-out `HEAD`.** A cached tree may have the correct commit checked out but modified tracked files or injected untracked files. The fetch target would report it as complete. This contradicts both reproducibility and the later cache-poisoning claim.
- **MEDIUM — The staging directory design is not concurrency-safe.** A fixed sibling “partial” directory lets concurrent fetches delete or mutate each other’s staging area. Renaming a directory onto an already-created destination is not generally “last-write-wins benign.”
- **MEDIUM — `Slug()` cannot guarantee collision freedom under the proposed grammar.** Replacing `/` with `-` makes repositories such as `a-b/c` and `a/b-c` collide; both are permitted by `[A-Za-z0-9._-]`.
- **MEDIUM — Omitting `apache/arrow` from the manifest conflicts with D-09’s stated role for rejected candidates.** D-09 says rejected candidates remain in the manifest marked unlocked. If the intent is now that pre-measurement disqualifications live only in the record, that is a decision change and should be made explicit rather than smuggled into implementation.

### Suggestions

- Validate a cached checkout with at least:

  - exact `HEAD`;
  - `git diff --quiet`;
  - `git diff --cached --quiet`;
  - no unexpected untracked files;
  - preferably `git write-tree` or `git rev-parse HEAD^{tree}` plus a clean-worktree assertion.

- Verify reproducibility using the Git tree object, not a filename-list hash.
- Use `mktemp -d` under the destination parent and a per-entry lock or atomic create protocol. If the destination appears before commit, validate it and discard the staged copy.
- Use a non-colliding directory encoding such as `org--repo@sha`, while rejecting `--` in components, or include a digest of the canonical repository name.
- Reconcile the Arrow handling with D-09 before execution.

### Risk Assessment

**HIGH.** The basic fetch design is sound, but its current integrity and concurrency checks do not establish the claims the phase depends on.

---

## 01-05 — Measurement-record pipeline

### Strengths

- Driving the indexer and query engine in-process follows existing repository practice and avoids a second CLI parsing implementation.
- Using `MarshalStatusJSON` preserves the actual user-visible contract at [internal/query/status.go:317](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:317).
- Dense edge counts preserve measured zeros.
- Treating generated markdown as a pure projection of committed JSON is a good anti-drift design.
- Failing on a missing measurement file avoids the existing skip-prone corpus behavior.

### Concerns

- **HIGH — The plan’s own smoke verification cannot succeed.** It fetches only the first manifest entry, then runs `measure -scope all`. The plan also requires `-scope all` to measure every manifest entry and fail loudly if any corpus directory is absent. With at least eight candidates, seven are absent, so the verification is guaranteed to fail.
- **HIGH — The tool has no specified way to measure one named candidate.** Plan 01-06 requires incremental candidate search, but Plan 01-05 exposes only `locked` and `all`. Before selection, the locked set is empty; `all` requires every candidate. This defeats the proposed bounded, measure-as-you-go spike.
- **HIGH — Metadata preservation is undefined.** The generated record contains thresholds, rationale, locked set, rejected candidates, and supplier summaries, but `-mode measure` is described as reconstructing the record from indexing. Plan 01-06 later manually adds selection metadata and then reruns `task corpora:measure`; unless the tool explicitly merges and preserves those fields, regeneration will erase them.
- **MEDIUM — The proposed volatile stripping is broader than the source fixture policy but described as “the same policy.”** The existing fixture policy names `dbSizeBytes`, `lastIndexed`, and `score`, plus suffix rules at [testdata/golden/golden_test.go:20](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:20). Additional removals may be justified, but should be identified as measurement-record policy layered on top, not the same policy.
- **LOW — `pendingChanges` is deterministic today.** It is an inert all-zero placeholder according to [internal/query/status.go:34](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:34). Stripping it is unnecessary unless the record explicitly wants a reduced schema.

### Suggestions

- Add `-repo org/name` or `-repos ...` scope so the spike can measure candidates incrementally and honor D-17.
- Separate generated observations from curated selection metadata:

  - either two files (`observations.json` and `selection.json`);
  - or a merge mode that preserves validated policy fields and rejected ledger entries.

- Make `corpora:measure` reject unknown/duplicate preserved metadata and prove round-trip preservation in tests.
- Fix the smoke test to fetch precisely the corpus it measures.
- Document the measurement-specific volatile policy separately from the golden fixture policy.

### Risk Assessment

**HIGH.** The intended architecture is good, but the command interface and regeneration semantics do not support the next wave.

---

## 01-06 — Candidate measurement and corpus locking

### Strengths

- Measuring `overrides` and `type_of` first is a rational search ordering.
- Recording every attempted candidate, including failures, is strong evidence discipline.
- Per-kind thresholds are more meaningful than a single global threshold.
- The plan correctly keeps language coverage at non-zero because only edge thresholds were promoted.
- Rechecking licenses at lock time is prudent.

### Concerns

- **HIGH — Synthetic fallback does not satisfy the phase’s stated success criteria.** The roadmap requires every ranked kind to have a non-zero measured count attributed to a named repository, and explicitly requires named repositories for `overrides` and `type_of`. The plan weakens this to “repository OR synthetic.” D-16 permits a recorded fallback, but that fallback cannot be treated as satisfying FIXT-01 without changing the requirement.
- **HIGH — The synthetic fallback is not implemented or measured in this phase.** The plan says a behavioral case is “to be added,” while its modified files contain only the manifest and measurement artifacts. It can therefore mark synthetic coverage without any indexed corpus or observed count behind it.
- **HIGH — Incremental measurement is blocked by Plan 01-05’s interface.** With only `locked` and `all` scopes, Task 1 cannot fetch and measure three candidates independently while failing on absent candidates.
- **HIGH — Regeneration can erase the manually selected threshold, rationale, rejected Arrow entry, and coverage supplier map.** No preservation mechanism is specified in Plan 01-05.
- **MEDIUM — Threshold selection remains subjective and non-reproducible.** “Wide enough for ordinary churn” and “high enough to be real” do not define an algorithm. Two executors can choose materially different bars from the same measurements.
- **MEDIUM — “Minimum set” is not formally defined.** It could mean minimum cardinality, minimum total files, minimum cache size, or simply a locally minimal set. The selection may vary without an optimization rule.
- **MEDIUM — The verification simultaneously allows synthetic coverage in prose but requires a named repository and positive count in its Python check.** The executable verification and acceptance language disagree.

### Suggestions

- Treat inability to obtain third-party coverage for all nine kinds as a **blocking phase failure** requiring a roadmap/requirements decision. Do not silently satisfy FIXT-01 with synthetic coverage.
- If synthetic fallback is approved as a changed requirement, add and measure the behavioral case in this phase, pin its identity, and make the guard distinguish third-party and synthetic claims.
- Define a deterministic threshold rule, for example a fixed percentage of the best supplier count with explicit rounding and a minimum greater than one.
- Define locked-set optimization explicitly, such as minimum cardinality with deterministic tie-break by total tracked files then repository name.
- Resolve the record regeneration/preservation design before this plan begins.

### Risk Assessment

**HIGH.** This is the phase’s core decision point, and the current plan can record success without meeting the roadmap’s actual criteria.

---

## 01-07 — Drift guard and CI

### Strengths

- The cheap/expensive split matches D-08 well.
- Adding the new workflow job to the Taskfile guard is correct. The guard’s in-scope fixture is explicit at [internal/upgrade/taskfile_shape_test.go:99](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:99), so silently omitting the job would weaken repository policy.
- Reusing the established Namespace action is consistent with current workflows: [ci.yml:60](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:60), [ci.yml:203](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/ci.yml:203), and [bench.yml:112](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/bench.yml:112).
- An unconditional idempotent fetch followed by a positive assertion is superior to branching solely on cache-hit output.
- Exact checked-kind and checked-corpus counts are good defenses against a vacuous pass, following the repository’s existing `ExpectedScenarioCount` pattern at [test/wireoracle/scenarios.go:490](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/scenarios.go:490).

### Concerns

- **HIGH — The edge guard appears to validate the precomputed coverage summary, not derive coverage from locked measurements.** If `coverage[kind]` says a threshold is met while its supplier is unlocked, missing, rejected, or has a lower measurement, the plan does not explicitly require `CheckCoverage` to reject it. This permits the central claim to drift internally while the guard remains green.
- **HIGH — Cache poisoning is not prevented by checking `HEAD`.** The source implementation’s reader snapshot guarantees store consistency, not checkout integrity. A mutable cached checkout can retain the right `HEAD` with modified tracked files or injected untracked files. The plan’s assertion would accept it before indexing.
- **MEDIUM — `ExpectedLockedCorpusCount` as a hand-maintained constant creates a second authority.** The manifest is supposed to be the sole pin and lock authority. A constant duplicates the locked count and must be changed whenever the manifest changes. Positive checking should compare the record’s locked set with manifest-derived locked entries instead.
- **MEDIUM — The path filter omits measurement-producing code.** The expensive workflow triggers on the manifest, workflow, Taskfile, and `internal/corpora/**`, but not `tools/corpora/**`, `internal/query/status.go`, render/marshal code, or indexer/extractor changes. D-08 only mandates manifest-triggered remeasurement, so this is not a direct decision violation, but it allows the measuring algorithm to change without the expensive job running.
- **MEDIUM — The human-check and acceptance criterion contradict the plan.** The action requires the fetch step to be unconditional on warm runs; the final acceptance text says the warm run shows “the fetch skipped.” It should say the step ran and reported already present.
- **LOW — A PR-only path-filtered workflow does not prove the post-merge main-branch environment remains healthy.** Manual dispatch helps, but a `push` path trigger would provide stronger continuing evidence.

### Suggestions

- Make `CheckCoverage` derive the edge claim from first principles:

  1. load and validate the manifest;
  2. derive locked `repo@SHA` identities;
  3. require exact equality with the record’s locked set;
  4. require every locked identity to have a measured outcome;
  5. sum or select counts from those measurements;
  6. require every claimed supplier to be locked and its recorded count to match the measurement;
  7. then apply thresholds.

- Remove `ExpectedLockedCorpusCount`, or generate it from the locked manifest set. Preserve the positive assertion by requiring `checkedCorpora == len(manifest.LockedEntries())` and `checkedCorpora > 0`.
- Harden cached-checkout validation before indexing: clean tracked tree, expected tree object, and controlled untracked-file policy.
- Expand the expensive workflow’s path filter to the measurement tool and status/indexer code that can change measurements, or explicitly document why such changes are covered elsewhere.
- Correct the warm-run acceptance wording.
- Consider a `push` path trigger for the same paths.

### Risk Assessment

**HIGH.** The CI structure is strong, but the current integrity checks can pass while the underlying locked measurements or cached source trees are wrong.

---

# Cross-plan dependency and completeness assessment

## Wave ordering

The high-level ordering is logical:

1. status data layer and manifest can proceed independently;
2. dense rendering follows the data field;
3. transcript and measurement tooling follow render/fetch support;
4. selection follows measurement tooling;
5. CI follows a locked set.

The actual interfaces break that ordering:

- Wave 3 does not produce a measurement command capable of the incremental search Wave 4 requires.
- Wave 3 does not define how Wave 4’s curated metadata survives regeneration.
- Wave 4 may claim synthetic coverage without producing the corresponding corpus data.
- Wave 5 trusts summary fields and cached `HEAD` values without independently reconstructing the evidence.

## Ability to meet Phase 1 success criteria

| Criterion | Assessment |
|---|---|
| Recorded real per-kind and language counts | Achievable after fixing measurement scope and record preservation |
| All 9 kinds and 5 languages covered | Not guaranteed; synthetic fallback currently weakens the criterion |
| `overrides` and `type_of` measured from named repos | Correct search strategy, but failure handling is incompatible with the stated requirement |
| Reproducible pinned fetch | Not yet proven because filenames+HEAD is not a content hash and dirty cached trees pass |
| CI cache miss fetches rather than skips | Structurally strong, but cache integrity assertion must validate contents, not only `HEAD` |

# Priority fixes before execution

1. Add single-/subset-candidate measurement support.
2. Define regeneration semantics that preserve or separately store selection metadata.
3. Resolve whether synthetic coverage is allowed to satisfy FIXT-01; current roadmap language says no.
4. Make coverage derive from locked raw measurements, not the precomputed summary.
5. Validate checkout contents and cleanliness, not merely `HEAD`.
6. Replace the filename-list “tree hash” with actual Git tree integrity.
7. Make staging concurrency-safe and make slug encoding collision-free.
8. Reconcile Arrow’s manifest treatment with D-09.
9. Make threshold and locked-set selection deterministic.

# Overall Risk Assessment

**HIGH.**

The plans are well researched and stronger than average at locating real integration points, but the phase’s central evidence chain is not yet closed:

```text
pinned manifest
  → verified checkout contents
  → measured raw observations
  → deterministic selection policy
  → locked set
  → independently derived coverage guard
```

At present, the checkout-content link, incremental measurement link, selection-regeneration link, and independent guard link each have a material gap. Fixing those does not require redesigning the phase; it requires tightening the data model and command interfaces before Wave 1 execution begins.

---

## Consensus Summary

**Single-reviewer cycle.** Only the Codex lane was requested (`/gsd-review --phase 1 --codex`), so
there is no cross-model agreement to report. In place of consensus weighting, the orchestrator
re-verified every HIGH finding directly against the plan text before recording it below; the
per-finding verification is in the "Verification coverage" block at the end of this file. Codex ran
**with repo access** — its output carries `file:line` citations into real source
(`internal/graphstore/pebble_store.go:285`, `internal/query/status.go:202`,
`test/wireoracle/scenarios.go:752`, `.github/workflows/ci.yml:60`) and no
`REVIEWED-WITHOUT-REPO-ACCESS` marker — so its findings are weighted as a grounded plan review.

The reviewer's central observation is structural rather than local: the phase depends on an
unbroken evidence chain

```text
pinned manifest → verified checkout contents → measured raw observations
  → deterministic selection policy → locked set → independently derived coverage guard
```

and four links in that chain are currently open. Nothing here calls for redesigning the phase or
resequencing the waves; the fixes are to data-model and command-interface details, and all of them
land before Wave 1 executes.

### Agreed Strengths

Single reviewer, so these are Codex's, verified against source rather than corroborated:

- Plans locate real integration points rather than plausible-sounding ones — all four status render
  surfaces, the correct wire-oracle transcript (`call-status`, not `resources-read-status`), the
  Taskfile shape guard, and the repo's existing Namespace cache usage.
- Deriving dense keys from `query.RankEdges` is the right invariant, and an exact-membership test
  for that set already exists at `internal/query/rwr_test.go:13`.
- Keeping the raw edge tally unfiltered (rather than filtering through `RankEdges`) correctly avoids
  hiding unexpected stored edge kinds.
- Refusing to self-skip on a missing corpus — explicitly rejecting the `resolveWeftCorpus` /
  `resolveColbymchenryCorpus` pattern — is exactly the defect class this phase exists to retire.
- Exact checked-kind and checked-corpus counts as the positive half of the drift guard follow the
  repo's established `ExpectedScenarioCount` pattern.
- Staging a capture to a temp file before replacing a golden protects the oracle on capture failure.

### Agreed Concerns

Ranked by the reviewer's severity, deduplicated across plans (several HIGHs were raised twice, once
at the plan that creates the defect and once at the plan that consumes it):

1. **HIGH — the "tree hash" hashes filenames, not bytes** (01-04). `git rev-parse HEAD` plus
   `git ls-files | sort | shasum` proves the commit ID and the tracked-file *list*; a modified
   tracked file, changed mode, or altered symlink target produces an identical hash. This is the
   evidence for success criterion 4.
2. **HIGH — a cached checkout is trusted by `HEAD` alone** (01-04 fetch, 01-07 CI assertion). A tree
   with the correct commit checked out but modified tracked files or injected untracked files is
   accepted as complete and indexed. `T-01-07-01`'s mitigation rests on precisely this check.
3. **HIGH — 01-05's own smoke verification cannot pass.** It fetches the first manifest entry only,
   then runs `-mode measure -scope all`, while the same plan requires an absent corpus directory to
   exit non-zero and name the corpus.
4. **HIGH — no single-/subset-candidate measurement scope** (01-05), which blocks 01-06's bounded,
   measure-as-you-go search: before selection the locked set is empty and `all` demands every
   candidate be present.
5. **HIGH — curated selection metadata has no defined survival path through regeneration** (01-05
   producing, 01-06 consuming). Thresholds, rationale, the rejected-candidate ledger and the
   coverage supplier map are hand-added in 01-06, then `task corpora:measure` is re-run.
6. **HIGH — synthetic fallback is allowed to satisfy FIXT-01** (01-06). Roadmap success criteria 2
   and 3 require a non-zero measured count *attributed to a named repository*, and name `overrides`
   and `type_of` specifically. D-16 permits a *recorded* fallback; it does not amend the criterion.
7. **HIGH — the synthetic fallback is neither implemented nor measured in this phase** (01-06). The
   behavioral case is "to be added" and the task's file list contains only manifest and measurement
   artifacts, so a kind can be marked synthetically covered with no indexed corpus behind it.
8. **HIGH — the coverage guard validates the precomputed summary rather than deriving the claim**
   (01-07). `CheckCoverage(rec)` reads `coverage[kind]`; nothing requires the claimed supplier to be
   locked, present, or to match its own raw measurement, so the central claim can drift internally
   while the guard stays green.

Recurring MEDIUM themes worth reading together: two hand-maintained second authorities
(`ExpectedLockedCorpusCount` beside the manifest; the coverage summary beside the raw measurements),
two under-specified selection rules (the per-kind threshold, and what "minimum set" optimizes), and
three places where a plan's executable verification and its own prose acceptance text disagree
(01-06 synthetic, 01-07 warm-run wording, 01-03 sparsity).

### Divergent Views

No divergence to report — one reviewer. Two points where the reviewer's own reading is worth
recording as contested rather than settled:

- **Codex disputes `T-01-04-03`'s claim that "two concurrent fetches of the same entry both rename
  identical trees, so last-write-wins is benign."** Renaming a directory onto an already-existing
  directory is not a benign atomic replace on POSIX, and a fixed sibling `partial` path lets one
  fetch delete another's staging area. The plan does address concurrency — the mitigation is simply
  asserted to hold rather than shown to.
- **Codex reads the omission of `apache/arrow` from the manifest as conflicting with D-09**, which
  keeps rejected candidates in the manifest marked unlocked. It may be that pre-measurement
  disqualification is intended to live only in the record; if so that is a decision change and needs
  stating, not implying.

### Explicitly not raised (pre-settled, supplied to the reviewer)

The reviewer was told, and correctly did not flag: the absent `UPDATE_GOLDEN`/`-update` transcript
flag (LOCKED v0.3.0 Phase 1 decision; `test/wireoracle/cmd/wireoracle/main.go` is the human-run
redirect entrypoint); the use of `namespacelabs/nscloud-cache-action` over `actions/cache` (LOCKED
v1.0 Phase 10 D-06, 9 existing uses); and `intel/API-SURFACE.md` reporting 0 symbols (regex/JS-only
extractor in a Go repo — absence there means UNKNOWN, never NONEXISTENT).

---

## Verification coverage

Every HIGH below was re-checked by the orchestrator against the plan text before being recorded, so
the count is not taken on the reviewer's word:

| # | Finding | Verified at |
|---|---------|-------------|
| 1 | Filename-list tree hash | `01-04-PLAN.md:348` — `git rev-parse HEAD && git ls-files \| sort \| shasum`; restated as the criterion at `:356` |
| 2 | Cached tree trusted by HEAD | `01-04-PLAN.md:315` (already-present → exit zero), `01-07-PLAN.md:464` (`T-01-07-01` mitigation rests on HEAD equality) |
| 3 | Smoke verification cannot pass | `01-05-PLAN.md:240` (fetch entry[0], then `-scope all`) against `01-05-PLAN.md:191-192,249` (absent dir → non-zero exit) |
| 4 | No subset scope | `01-05-PLAN.md:193,222` — `-scope` accepts only `locked` and `all` |
| 5 | Metadata preservation undefined | `01-05-PLAN.md:200-236` has no merge/preserve step; `01-06-PLAN.md:181` re-runs the generator over `corpora/measurements.json` |
| 6 | Synthetic satisfies FIXT-01 | `01-06-PLAN.md:164,280,423` (`… OR the record marks that kind as synthetically covered`) vs ROADMAP success criteria 2–3 |
| 7 | Synthetic not implemented in-phase | `01-06-PLAN.md:102,181` — task file lists carry only `corpora/*.json` and `docs/CORPUS-MEASUREMENT.md`; no behavioral-corpus source |
| 8 | Guard trusts the summary | `01-07-PLAN.md:155-182` — `CheckCoverage(rec MeasurementRecord)` checks `coverage[kind]` against thresholds; no locked-set / raw-measurement reconciliation |

Source-grounding of the review itself: Codex cited 20+ distinct `file:line` locations across
`internal/graphstore/`, `internal/query/`, `internal/cli/`, `internal/mcp/`, `test/wireoracle/`,
`testdata/`, and `.github/workflows/`; spot-checks of `internal/query/rwr.go:21`,
`internal/query/status.go:202` and `test/wireoracle/scenarios.go:752` matched the claims. No
`REVIEWED-WITHOUT-REPO-ACCESS` marker was emitted.


---

## Verification coverage (source-grounding pass, cycle 1)

Run by the plan-review-convergence orchestrator, 2026-08-14, against plans at `f85f9c7`.

### Adapter substitution — disclosed, not silent

`gsd-tools drift-guard authority --raw` resolved the effective authority to **`intel`**.
The `intel` adapter was **NOT** used. It is regex/JS-only and `gsd-tools intel api-surface`
regenerated `.planning/intel/API-SURFACE.md` with `symbolCount: 0` against this Go tree — the
fifth recorded occurrence of that result in this repo. Under `intel`, every Go symbol would
resolve `UNCHECKABLE`, producing a pass in which nothing was actually checked: the vacuous-gate
shape rule `84d1gfpywd` forbids.

**Adapter actually used: `grep`** (ripgrep over the real tree, plus the repo's own `.codegraph`
index). Under `grep`, signature mismatches cannot be asserted, so no signature claim below is
reported as VERIFIED or MISSING — signatures are `UNCHECKABLE` by construction of the adapter.

### Excluded from verification (declared new artifacts)

All symbols listed under each plan's "Artifacts this phase produces" section were excluded, per
the source-grounding contract — these are created by this phase, not references to existing code.
All 7 plans carry that section. Excluded set includes `internal/corpora/*`, `tools/corpora/*`,
`corpora/manifest.json`, `corpora/measurements.json`, `docs/CORPUS-MEASUREMENT.md`,
`.github/workflows/corpora.yml`, `query.DenseEdgesByKind`, `query.StatusResult.EdgesByKind`,
and all `Test*` names introduced by this phase.

### Existing file paths — 20 cited, 20 VERIFIED, 0 MISSING

`internal/query/status.go` · `internal/query/render_status.go` · `internal/query/rwr.go` ·
`internal/query/rwr_test.go` · `internal/query/expand.go` · `internal/cli/status.go` ·
`internal/cli/present/status.go` · `internal/agents/opencode.go` · `internal/graphstore/keys.go` ·
`internal/indexer/resolve.go` · `internal/mcp/tools.go` · `internal/mcp/markdown_test.go` ·
`testdata/golden/gocapture/main.go` · `testdata/golden/golden_parity_test.go` ·
`test/wireoracle/cmd/wireoracle/main.go` · `testdata/wireoracle/transcripts/call-status.golden` ·
`internal/upgrade/taskfile_shape_test.go` · `.github/workflows/linux-cross-canary.yml` ·
`.github/workflows/ci.yml` · `Taskfile.yml`

### Existing Go symbols — 21 cited, 21 VERIFIED, 0 MISSING, 0 AMBIGUOUS

`RankEdges` · `StatusResult` · `RenderStatusText` · `RenderStatusMarkdown` · `sortedCounts` ·
`IterateEdges` · `FilesByLanguage` · `NodesByKind` · `TestRankEdges` · `ExpectedScenarioCount` ·
`TestWorkflowRunBodiesInvokeTask` · `inScopeJobs` · `synthesizeOverrides` · `emitTypeOfRef` ·
`pinnedWeftCommit` · `resolveWeftCorpus` · `resolveColbymchenryCorpus` · `edgeKey` ·
`GetEdgeCount` · `newStatusCmd` · `TestFrozenTranscriptsMatch`

### INFO — reported, not blocking

| Item | Verdict | Note |
|---|---|---|
| `gocapture/main.go` (01-05:203) | INFO | Prose shorthand. The full path `testdata/golden/gocapture/main.go` is VERIFIED and is cited in full elsewhere in the same plan. No action required. |
| All Go signatures | UNCHECKABLE | The `grep` adapter cannot assert signatures. Never treated as verified. |

### Finding raised BY this pass (not a drift verdict)

Following `resolveWeftCorpus` to its first definition site surfaced
**`tools/bench/realcorpus/manifest.go`** — an existing pinned-SHA corpus manifest
(`Entry{Name, SourceURL, Tag, CommitSHA, License, SelectionRule, QueryTerms, EnvVar, SiblingDir}`,
`Corpora()`, `Entry.Resolve()`, `ErrNeedsClone`), consumed by `tools/bench/runner/main.go`, with
three pinned entries. Both `01-RESEARCH.md` and `01-PATTERNS.md` assert no such manifest exists;
`01-PATTERNS.md` files it under "No Analog Found."

This is a **planning finding, not a symbol-drift verdict**, and is carried into the replan rather
than recorded here as a coverage result. Full detail in the replan payload.

**Methodological note:** this pass's own first sweep also verified that "No Analog Found" row as
standing. It was caught only by following a symbol to where it actually lives, not by searching for
the concept. Two agents plus one orchestrator narrowed identically on the same expected name.


---

# Cross-AI Plan Review — Phase 1, CONVERGENCE CYCLE 2

<!-- cycle: 2
     reviewers: [codex]
     reviewed_at: 2026-08-14T02:47:00Z
     plans_reviewed_at_commit: f108c68 ("docs(01): incorporate cycle-1 review feedback across all 7 plans")
     prior_cycle: 2166174 + 26f894e (cycle-1 review and its source-grounding coverage block)
     plans_reviewed: 01-01-PLAN.md … 01-07-PLAN.md -->

Reviewed at `f108c68`, against the cycle-1 review recorded above. The reviewer was given the
cycle-1 findings, the settled-context list (the locked transcript-flag decision,
`nscloud-cache-action`, the `intel` 0-symbol extractor, the deliberate `internal/corpora` /
`realcorpus` separation, and 01-06's intentional HALT-and-escalate), and was instructed to count
only what remains unresolved.

## Codex Review (cycle 2)

### Summary

The revision substantially converged: **all eight cycle-1 HIGH concerns now have concrete
plan-level responses**, visible in tasks, behavior blocks, actions, verify commands, acceptance
criteria, must_haves and threat models — not merely acknowledged in prose. However, new HIGH
inconsistencies make the revised execution path internally impossible or self-failing: incremental
measurements overwrite prior observations, real edge maps are incorrectly required to contain
exactly nine keys despite preserving unranked kinds, threshold calculation is circular with corpus
selection, and one plan consumes a Taskfile target created a wave later. One concurrency race also
remains actionable.

### Cycle-1 HIGH disposition

| ID | Cycle-1 concern | Status | Evidence |
|---|---|---|---|
| H1 | "Tree hash" covered filenames, not contents | RESOLVED | `01-04-PLAN.md:367-374` replaces it with `HEAD^{tree}` plus empty `--porcelain --ignored`; the old design is named and forbidden at `:386-390`; mutation test at `:446` |
| H2 | Cached checkout trusted by `HEAD` alone | RESOLVED | Four-part check defined once at `01-04-PLAN.md:367-374`, invoked on the existing-destination path `:403-409` and the staged tree `:417-418`, and by CI at `01-07-PLAN.md:32,275-281,387-389,485` |
| H3 | Smoke test fetched one corpus but measured all | RESOLVED | `01-05-PLAN.md:283` now fetches and measures the SAME entry via `-repos`; pinned as an acceptance criterion at `:298` |
| H4 | No single/subset candidate scope | RESOLVED | `-repos org/name[,org/name...]` specified in the behavior block at `01-05-PLAN.md:230-234`; used by `01-06-PLAN.md:159-162` and pinned at `01-06-PLAN.md:227` |
| H5 | Regeneration erased curated selection metadata | RESOLVED | Two-file split — `corpora/observations.json` (generated) and `corpora/selection.json` (curated, never tool-written) — at `01-05-PLAN.md:88-93`; enforced by acceptance criteria `:303` and `:305` |
| H6 | Synthetic coverage treated as satisfying FIXT-01 | RESOLVED | HALT-and-escalate at `01-06-PLAN.md:184-193`; the guard rejects non-empty `syntheticKinds` at `01-07-PLAN.md:175` |
| H7 | Synthetic fallback neither implemented nor measured | RESOLVED | Removed from the Phase-1 success path; behavioral-corpus authoring explicitly deferred to Phase 2 |
| H8 | Coverage guard trusted a stored summary | RESOLVED | `CheckCoverage(m, obs, sel)` derives the claim in seven steps at `01-07-PLAN.md:86-97,188-192`; no supplier field is read, so the claim cannot disagree with the evidence |

**All eight are resolved as plan content. None is merely acknowledged.**

### New concerns introduced by the revision

#### HIGH — Incremental measurement overwrites earlier observations

`01-05-PLAN.md:25,88` state that `observations.json` is "fully reconstructed" by the generator, and
acceptance criterion `01-05-PLAN.md:297` requires `-mode measure -repos <one entry>` to write "an
observations file with **one entry**" (pinned executably at `01-05-PLAN.md:286`,
`assert len(m)==1`). `01-06-PLAN.md:155-162` then measures `google/guava`,
`JamesNK/Newtonsoft.Json` and `nestjs/nest` **one at a time** via `-repos` and asserts
`len(o)>=3` at `01-06-PLAN.md:202`. Under the specified write semantics each invocation replaces
the previous file, so that assertion cannot pass.

A second half of the same defect: the default scope is `locked` (`01-05-PLAN.md:231`), so the final
`task corpora:measure` after locking reconstructs observations from the locked set only, deleting
every measured-but-rejected candidate's observation. That contradicts `01-06-PLAN.md:230` ("every
candidate fetched and measured appears in the record regardless of score") and
`01-06-PLAN.md:359`. Conversely, retaining unlocked candidates' observations would break
`corpora:drift` (`01-07-PLAN.md:282-285`), which re-measures the **locked** corpora and then
demands `corpora/observations.json` be diff-clean. As written the two requirements are mutually
exclusive.

**Plan change required.** Pick and specify one model: `-repos` replaces only the named entries in
place; or each incremental run writes a fragment followed by an explicit combine step; or fetch
incrementally but measure all fetched candidates together once the search completes. Add a test
that measures A, then B, and proves both survive, plus a test that final locked-set regeneration
retains measured rejected candidates — and reconcile whichever model is chosen with
`corpora:drift`'s clean-diff requirement.

#### HIGH — "Exactly nine keys" contradicts preservation of unranked edge kinds

`01-02-PLAN.md:107` requires `DenseEdgesByKind` to return **the union**, so "an unranked kind is
never silently dropped". The repository emits real unranked edge kinds: `contains` is written by
`internal/indexer/resolve.go:191` and by every extractor (`csharpextract.go:238,433`,
`javaextract.go:249,462`, `rubyextract.go:234,332,363`, `cextract.go:214,283,329,380,394`,
`rustextract.go:142`), while `query.RankEdges` (`internal/query/rwr.go:21-31`) holds exactly nine
kinds and `contains` is not among them.

Nevertheless `01-05-PLAN.md:288`, `01-06-PLAN.md:206` and acceptance criterion `01-06-PLAN.md:228`
assert `len(edgesByKind) == 9`. Any real corpus will carry `contains`, so these checks are expected
to fail on first execution.

**Plan change required.** Replace the exact-length checks with: every `query.RankEdges` key is
present; those nine values may be zero; any additional key is permitted only with a positive
observed count. That is what `01-02-PLAN.md`'s own CLI contract already says.

#### HIGH — Threshold computation and locked-set selection are mutually circular

`01-06-PLAN.md:259` defines each kind's threshold as `max(2, floor(best/2))` where `best` is "the
highest measured count for that kind **across the locked candidates**", while `01-06-PLAN.md:287`
selects the locked set as the minimum-cardinality set that supplies every kind **at its threshold**.
The locked set is required to compute the thresholds that are required to choose the locked set.
Multiple fixed points can exist, and the verification only recomputes thresholds from the final
authored set — it never proves that set is globally minimum or that the declared
file-count/name tie-break was applied.

Second, a boundary defect: when the best non-zero count for a kind is `1`,
`max(2, floor(1/2)) = 2`, a threshold no measured repository supplies, so `CheckCoverage` fails
permanently. The existing escalation path (`01-06-PLAN.md:184-193`) is written for **zero**
third-party coverage, not for this threshold-induced failure. `overrides` and `type_of` — the two
at-risk kinds the whole bounded search exists for — are exactly where a count of 1 is plausible.

**Plan change required.** Derive thresholds from a non-circular universe (all measured eligible
candidates); then enumerate candidate subsets and select the deterministic optimum; implement that
selection in code or add an executable verifier that independently enumerates subsets and proves
cardinality and tie-break optimality; and explicitly halt when a derived threshold is
unsatisfiable, including the `best == 1` case.

#### HIGH — `task corpora:assert` is consumed a wave before it is defined

`01-06-PLAN.md:356` runs `task corpora:fetch && task corpora:assert` in a verify block,
`01-06-PLAN.md:369` pins it as an acceptance criterion, and `01-06-PLAN.md:513` restates it as a
phase verification. But `corpora:assert` is created only by `01-07-PLAN.md:269-281` — 01-06 is wave
4, 01-07 is wave 5. The only earlier Taskfile plan, 01-04, adds exactly three targets and says so:
`01-04-PLAN.md:45` (`provides: "corpora:fetch, corpora:fetch-one, corpora:assert-one"`) and
`01-04-PLAN.md:360`. 01-05 adds only `corpora:measure` (`01-05-PLAN.md:276`).

01-06 therefore cannot pass its own verification — the same defect class as cycle-1 H3,
reintroduced at a different seam by the revision.

**Plan change required.** Either move `corpora:assert` into 01-04 Task 3 (it is a pure loop over
the already-defined `corpora:assert-one`), or replace 01-06's use with an explicit loop over
`corpora:assert-one` and drop the acceptance criterion naming `corpora:assert`.

### Remaining concerns

- **MEDIUM — concurrent promotion is still check-then-act.** `01-04-PLAN.md:419-425` correctly
  retracts the earlier "benign last-write-wins" claim and forbids renaming onto an existing
  directory, but the replacement protocol re-checks the destination and then moves. Two processes
  can both observe absence before either moves, and `mv` of a directory onto an existing directory
  nests rather than failing cleanly. No concurrent-fetch verification exists in any 01-04 verify
  block. **Change needed:** a per-destination lock or an atomic destination-claim whose loser is
  guaranteed to fail without nesting, plus a verify that launches two fetches simultaneously and
  asserts one canonical destination, no nested staging tree, and no leftover staging path.
- **MEDIUM — the `-out` flag is used but never declared.** `01-05-PLAN.md:283,292,293,294` all pass
  `-out /tmp/obsN.json`, but `-out` appears in no behavior block, action, artifact
  (`01-05-PLAN.md:36` lists only `-repos/-scope`) or acceptance criterion, and it sits in tension
  with behavior `01-05-PLAN.md:238` ("`-mode measure` writes `corpora/observations.json` and
  NOTHING ELSE"). An executor implementing only the action would leave every 01-05 Task 2 verify
  command failing. **Change needed:** declare `-out` in the behavior block with its default, and
  reword `:238` so it constrains which *record* files are written rather than the output path.
- **MEDIUM — `git diff --quiet <path>` exits 128 when the path does not exist.** Used as an
  executable acceptance criterion at `01-05-PLAN.md:303` and `01-06-PLAN.md:233`, and cited in
  threat T-01-05-02 at `01-05-PLAN.md:406`. `corpora/selection.json` does not exist during 01-05
  and early 01-06 — `01-05-PLAN.md:361` says so explicitly. **Change needed:** use
  `git diff --quiet -- corpora/selection.json` guarded by an existence test, so the criterion means
  "unchanged" rather than hard-failing on absence.
- **MEDIUM — a verify command that asserts nothing.** `01-06-PLAN.md:317` runs
  `task corpora:measure && git diff --stat corpora/observations.json docs/CORPUS-MEASUREMENT.md`.
  `git diff --stat` exits 0 unconditionally; it prints, it does not check. Every other verify in
  this phase is assertive. **Change needed:** replace with `git diff --quiet --` plus an explicit
  expectation, or a Python assertion over the regenerated file.
- **MEDIUM — the shared artifact list is stale in both directions.** The
  `artifacts_this_phase_produces` block replicated in all seven plans (`01-01:297`, `01-02:367`,
  `01-03:322`, `01-04:547`, `01-05:447-448`, `01-06:470-471`, `01-07:546-547`) names two tests no
  task instructs anyone to write — `TestMeasureRegenerationPreservesSelection` (obsolete: the
  two-file split removed the merge step it tested) and `TestMeasureReposScopeMeasuresExactlyNamed`
  — and omits six that tasks do create (`TestSelectionRejectsUnreasonedRejection`,
  `TestProseDerivesSupplierFromObservations`, `TestProseLabelsSyntheticCoverage`,
  `TestProseRendersWithoutSelection`, `TestCoverageRejectsLockedSetMismatch`,
  `TestCoverageRejectsMissingObservation`). Related: 01-05 Task 2 carries `tdd="true"` but names
  zero tests in its action. **Change needed:** reconcile the list across all seven copies and give
  01-05 Task 2 its test names.
- **LOW — 01-06 Task 1's `<files>` list is incomplete.** `01-06-PLAN.md:146` names only
  `corpora/observations.json`, but the task's action writes `corpora/selection.json` on the HALT
  path (`:186`) and every `-mode measure` run also regenerates `docs/CORPUS-MEASUREMENT.md`
  (`01-05-PLAN.md:358`). **Change needed:** add both paths.
- **LOW — `CheckCoverage`'s stated responsibilities exceed its signature.** must_haves truth
  `01-07-PLAN.md:25` says it "loads the manifest" and behavior `:181` says loading fails with the
  path named when a document is missing or malformed, but the declared signature `:188` takes three
  already-decoded documents. Loading belongs to the caller. **Change needed:** either move the
  load-failure behavior to the Taskfile/CLI layer that calls it, or change the signature.
- **LOW — the "rejected candidate" sub-case of H8 is structural but untested.** A rejected
  candidate is unlocked, so the derived-from-locked-observations design excludes it by construction,
  and `TestCoverageRejectsUnlockedSupplier` (`01-07-PLAN.md:219-221`) covers the mechanism. But the
  word "rejected" appears nowhere in `01-07-PLAN.md`. **Change needed:** one behavior line or a
  named test asserting that an observation belonging to a rejected (unlocked) candidate cannot
  supply a kind's coverage.

### Overturned rejections

**None.** The plans' argued dispositions — the deliberate separation from `tools/bench/realcorpus`,
the `nscloud-cache-action` mechanism, the absent transcript update flag, the removal of the
synthetic fallback, and the rejection of a merge step in favour of single-owner files
(`01-05-PLAN.md:95-96`) — are each well reasoned and consistent with repository evidence. The
ownership-over-merge argument in particular is stronger than what cycle 1 asked for: "the generator
has no write path to that file" is a control that cannot be wrong, whereas a merge routine is code
that can be.

### Risk assessment

**HIGH.** The cycle-1 evidence-chain defects were addressed thoughtfully and thoroughly, but the
revised observation lifecycle cannot accumulate the measurements 01-06 expects, the dense-map
verification contradicts the repository's actual edge vocabulary, the selection algorithm is
circular and not independently executable, and one plan consumes a Taskfile target created a wave
later. These are execution-blocking rather than cosmetic — but all four are localized specification
defects, not architectural ones.

Source-level integration assumptions were separately confirmed sound: `indexer.Run` supports
arbitrary temporary store directories (`internal/indexer/pipeline.go:76`); the capture path
demonstrates reopening that store and constructing `query.NewWithRoot`
(`testdata/golden/gocapture/main.go:131`); the MCP handler renders raw sparse status without
densifying (`internal/mcp/tools.go:532`); and the workflow run-body guard genuinely enforces bare
Task invocations (`internal/upgrade/taskfile_shape_test.go:1295`).

---

## Consensus Summary (cycle 2)

One prompt-fed source-grounded reviewer (Codex) ran this cycle, cross-checked by an independent
orchestrator verification pass over the same plan text and repository. Where the two agreed, the
finding is recorded as confirmed; the orchestrator pass contributed the wave-ordering HIGH and four
of the MEDIUMs, and Codex contributed the three internal-contradiction HIGHs.

### Agreed strengths

- **Every cycle-1 HIGH landed as machine-visible plan content**, not prose. Each fix appears in at
  least one of: behavior block, action, verify command, acceptance criterion, must_haves truth,
  prohibition, or threat-model row. Several are additionally defended against regression by greps
  that would fail if the old design returned (`01-04-PLAN.md:462`).
- **The fixes are argued, not just applied.** Retracted designs are named and forbidden in place —
  the filename-list hash (`01-04:386-390`), the merge step (`01-05:95-96`), the
  `ExpectedLockedCorpusCount` constant (`01-07:199-207`), the "benign last-write-wins" rename claim
  (`01-04:419-425`), and the synthetic fallback (`01-06:95-97`). A future reader cannot reintroduce
  them by accident.
- **The two-file split is a stronger control than the merge cycle 1 implied.** Ownership — "the
  generator has no write path to `selection.json`" — is not falsifiable by a coding error the way a
  merge routine is.
- **`CheckCoverage` now derives rather than validates**, which structurally eliminates three of the
  four drift sub-cases cycle 1 named, rather than testing for each individually.
- **Source grounding on both sides was accurate.** The plan's own CI claim — nine
  `nscloud-cache-action` uses at `ci.yml:60,203,232,356,386,415` and `bench.yml:112,374,473` —
  verified exactly, as did the `resolveWeftCorpus`/`resolveColbymchenryCorpus` skip precedent it
  cites as the pattern the guard must not follow.

### Agreed concerns

The four HIGHs above, in priority order: the observation-accumulation contradiction (it blocks
01-06 Task 1 outright and has a second, opposite-direction conflict with `corpora:drift`); the
nine-key assertion (it will fail on the first real corpus and contradicts 01-02's own union
semantics); the threshold/locked-set circularity (plus the unsatisfiable `best == 1` boundary, which
lands precisely on the two at-risk kinds); and the `corpora:assert` wave inversion.

The recurring theme across all four is the one cycle 1 named, displaced one level: **a plan's
executable verification disagreeing with its own specification.** Cycle 1 found it in 01-05's smoke
test; the revision fixed that instance and introduced four more at new seams. The
observation-lifecycle and edge-vocabulary contradictions in particular would each have been caught
by executing a single verify command against a real corpus.

### Divergent views

None to report — one prompt-fed reviewer. Two points worth recording as adjudicated rather than
settled:

- The orchestrator **downgraded** the reviewer's implicit framing of H8 as fully closed, to note one
  untested sub-case (rejected candidates), and separately **declined** to treat the missing `-out`
  declaration as HIGH despite its having the same "plan cannot pass its own verification"
  consequence as the wave-inversion HIGH — the difference being that `-out`'s resolution is a
  one-line declaration an executor could reasonably infer from the verify block, whereas the others
  are genuine contradictions between two plan statements.
- The reviewer's cycle-1 concurrency finding was **partially** addressed: 01-04's step 7 now
  retracts the wrong claim and states a protocol, so that half is resolved; the residual TOCTOU
  window and the absence of any concurrent-fetch verification remain open, and are carried as the
  MEDIUM above rather than re-counted as the original HIGH.

### Verification coverage (cycle 2)

Every HIGH recorded above was re-checked by the orchestrator against plan text and repository source
before being counted:

| # | Finding | Verified at |
|---|---------|-------------|
| 1 | Observations cannot accumulate | `01-05-PLAN.md:286,297` (one `-repos` run writes exactly one entry) vs `01-06-PLAN.md:202` (`assert len(o)>=3`); second half at `01-05-PLAN.md:231` (default `locked`) vs `01-06-PLAN.md:230,359` vs `01-07-PLAN.md:282-285` |
| 2 | Nine-key assertion vs real vocabulary | `01-05-PLAN.md:288`, `01-06-PLAN.md:206,228` (`len(edgesByKind) == 9`) vs `01-02-PLAN.md:107` (union) vs `internal/indexer/resolve.go:191` and nine extractor sites emitting `contains`; `internal/query/rwr.go:21-31` confirms `RankEdges` has exactly nine members, `contains` not among them |
| 3 | Threshold/selection circularity | `01-06-PLAN.md:259` (`best` across the LOCKED candidates) vs `01-06-PLAN.md:287` (locked set chosen to meet those thresholds); `max(2, floor(1/2)) = 2` computed directly |
| 4 | `corpora:assert` wave inversion | Used at `01-06-PLAN.md:356,369,513`; defined at `01-07-PLAN.md:269-281`; 01-04 provides only three targets per `01-04-PLAN.md:45,360`; 01-05 adds only `corpora:measure` per `01-05-PLAN.md:276` |

Artifact-naming consistency was swept across all seven plans: `corpora/manifest.json`,
`corpora/observations.json` and `corpora/selection.json` are used uniformly, and the single
surviving `corpora/measurements.json` reference (`01-05-PLAN.md:75`) is a deliberate,
named-and-rejected historical citation rather than drift.

No `REVIEWED-WITHOUT-REPO-ACCESS` marker was emitted; the reviewer cited repository `file:line`
evidence throughout, and its citations were spot-checked against the tree.

### Convergence verdict

```
CYCLE 2 — current_high=4  current_actionable=8
```

Cycle-1 HIGHs still open: **0 of 8.** All four HIGHs above were introduced by the cycle-1 fixes
themselves. The phase is converging on substance and regressing on internal consistency; the remedy
is a targeted consistency pass over the four contradictions, not a redesign.

---

# Cross-AI Plan Review — Phase 1, CONVERGENCE CYCLE 3 (FINAL)

<!-- cycle: 3
     reviewers: [codex]
     reviewed_at: 2026-08-14T13:29:25Z
     plans_reviewed_at_commit: 3c118c4 ("docs(01): resolve cycle-2 review findings — observation upsert, edge vocabulary, non-circular selection, wave ordering")
     prior_cycles: 2166174 + 26f894e (cycle 1), bce629b (cycle 2)
     plans_reviewed: 01-01-PLAN.md … 01-07-PLAN.md -->

Reviewed at `3c118c4`, against the cycle-2 review recorded above. The reviewer was given both
prior cycles, the settled-context list (the locked transcript-flag decision, `nscloud-cache-action`,
the `intel` 0-symbol extractor, the deliberate `internal/corpora` / `realcorpus` separation, 01-06's
intentional HALT-and-escalate, and the deliberate observations-upsert / selection-single-owner
asymmetry), the five specific fixes applied at `3c118c4`, and was instructed to hunt for knock-on
contradictions those fixes introduced elsewhere — the failure mode cycle 2 exhibited.

## Codex Review (cycle 3)

### Summary

The revision resolves **three of cycle 2's four HIGH findings and all eight of its actionable
lower-severity findings** in executable plan content — task actions, behavior blocks, verify
commands, acceptance criteria, must_haves and threat-model rows, not prose acknowledgement. The
fourth HIGH (threshold/selection circularity) is resolved *as library design* but only PARTIALLY as
executable plan content: the cure introduced a new consumption-before-definition gap of exactly the
class cycle 2's own H4 named. Three further consistency defects — two of them stale pre-revision
text the edit left behind — remain actionable.

### Cycle-2 finding disposition

All twelve cycle-2 findings (4 HIGH + 8 actionable) are listed so none is silently dropped.

| Cycle-2 finding | Status | Current plan evidence |
|---|---|---|
| HIGH — incremental measurement overwrites observations | **RESOLVED** | Upsert keyed by `repo@sha` as a must_haves truth at `01-05-PLAN.md:25`; read-modify-write specified in the behavior block at `:288-291` and required in the action at `:318-323`; `TestMeasureUpsertsWithoutDroppingPriorEntries` and `TestMeasureLockedScopeRetainsUnlockedObservations` named at `:338-342`; accumulation pinned as an acceptance criterion at `:371`. The opposite-direction half is reconciled explicitly at `01-07-PLAN.md:296-306` — locked-scope re-measure recomputes the same keys while an out-of-scope rejected candidate is neither re-measured nor removed, so `corpora:drift`'s clean-diff and D-17's keep-every-candidate hold together, and `-prune` is forbidden there by name. |
| HIGH — exact-nine-key checks contradict unranked kinds | **RESOLVED** | Every `len(...) == 9` is gone. Membership-plus-positive-extra semantics replace it at `01-05-PLAN.md:356-360` and `01-06-PLAN.md:204-208`; the acceptance criterion at `01-06-PLAN.md:231` names the rejected `== 9` design and why. Agrees with the real vocabulary: `contains` at `internal/indexer/resolve.go:191` and the nine-member `query.RankEdges` at `internal/query/rwr.go:21-31`. |
| HIGH — threshold computation circular with locked-set selection | **PARTIAL** | The circularity and the `best == 1` boundary are both genuinely fixed *in library design*: `ComputeThresholds(obs, eligible)` sources `best` from all measured eligible candidates with `min(max(2, best/2), best)` at `01-05-PLAN.md:184-200`; `SelectLockedSet` enumerates by increasing cardinality under a total tie-break at `:205-213`; five named tests at `:231-236`. **But the executable half is unimplemented — see New defects (c).** |
| HIGH — `corpora:assert` consumed a wave before defined | **RESOLVED** | Created in wave 1 by `01-04-PLAN.md:448-455`, declared in that plan's `provides` at `:45`, and `01-07-PLAN.md:284-287` now explicitly states it is NOT defined there and names why the earlier placement broke 01-06. |
| MEDIUM — concurrent promotion is check-then-act | **PARTIAL** | An atomic `mkdir "<dir>.lock"` claim with three enumerated branches at `01-04-PLAN.md:419-439`, plus a real two-process concurrency verify at `:472` and a behavior criterion at `:483`. **But the surrounding text was not reconciled — see New defects (e).** |
| MEDIUM — `-out` used but never declared | **RESOLVED** | Declared with its default in the behavior block at `01-05-PLAN.md:282-284`, required in the action at `:324`, pinned as an acceptance criterion at `:372`. |
| MEDIUM — `git diff --quiet <path>` exits 128 on an absent path | **RESOLVED** | Both sites now carry an existence guard and say why it is not cosmetic: `01-05-PLAN.md:378` and `01-06-PLAN.md:236`; the threat row at `01-05-PLAN.md:481` records it too. |
| MEDIUM — a verify command that asserts nothing | **RESOLVED** | `git diff --stat` replaced by two regenerations followed by `git diff --quiet --` at `01-06-PLAN.md:335`. |
| MEDIUM — shared artifact list stale in both directions | **RESOLVED** | The obsolete `TestMeasureRegenerationPreservesSelection` is gone; all six previously-omitted tests are present; `TestMeasureReposScopeMeasuresExactlyNamed` is now actually instructed at `01-05-PLAN.md:342`. The block is byte-identical across all seven plans (verified by hash). 01-05 Task 2 now names its six tests in its action at `:338-343`. |
| LOW — 01-06 Task 1 `<files>` incomplete | **RESOLVED** | All three paths named at `01-06-PLAN.md:146`. |
| LOW — `CheckCoverage` responsibilities exceed its signature | **RESOLVED** | Loading reassigned to the caller in the must_haves truth at `01-07-PLAN.md:25` and restated in the behavior block at `:185-188`, which names the earlier draft's misattribution; the acceptance criterion at `:256` pins "`CheckCoverage` itself performs no I/O". |
| LOW — rejected-candidate supplier case untested | **RESOLVED** | Behavior line at `01-07-PLAN.md:178`, `TestCoverageRejectsRejectedCandidateAsSupplier` at `:229-232`, and it is in the shared test inventory. |

**11 of 12 fully resolved as plan content; 1 partial. None merely acknowledged.**

### New defects introduced by 3c118c4

Checked per the five fixes applied at that commit.

#### (a) Upsert-keyed-by-`repo@sha` observation lifecycle — NO REGRESSION

Flag and field vocabulary agrees across all seven plans: `-repos`, `-scope`, `-out`, `-prune`,
`observations`, `lockedSet`, and repository-at-SHA identities are used identically wherever they
appear. 01-05 writes only in-scope keys and preserves the rest; 01-06 depends on that accumulation;
01-07 re-measures locked keys without pruning. The three-way composition is argued in place at
`01-07-PLAN.md:296-306`, not merely asserted.

#### (b) RankEdges-membership replacing `len(...) == 9` — NO CONTRADICTION, one robustness gap

The exact-length regression is fully gone and no plan disagrees with another about the semantics.
Carried below as an actionable LOW: the *Python* verify blocks still hand-restate the nine members
(`01-05-PLAN.md:358`, `01-06-PLAN.md:206`, `01-06-PLAN.md:340`) while `query.RankEdges`
(`internal/query/rwr.go:21-31`) is named throughout as the sole authority — and 01-05's own Go-side
acceptance criterion (`01-05-PLAN.md:249`) greps to prove the nine strings are NOT restated in
`internal/corpora/record.go`. The prohibition is scoped to Go source, so this is a drift vector
rather than a contradiction, but it is the one place a tenth ranked kind could be added while every
check stayed green.

#### (c) HIGH — non-circular `ComputeThresholds` plus `tools/corpora -mode select`: the mode is consumed but never created

The library half landed. The **executable** half did not.

- `01-05-PLAN.md:181-213` (Task 1) creates `ComputeThresholds` and `SelectLockedSet` in
  `internal/corpora/record.go`.
- `01-05-PLAN.md:259` (Task 2) extends `tools/corpora/main.go` with `measure` **only** — its
  must_haves artifact entry says exactly that: `provides: "-mode measure with -repos/-scope"`
  (`01-05-PLAN.md:38`). Task 3 adds only prose rendering.
- `01-04-PLAN.md:312` creates the dispatch and says it supports **two** modes, `root` and `entries`.
- No task in any of the seven plans instructs an executor to add a `select` mode.

Yet `01-06-PLAN.md:260` mandates computing the threshold and locked set with
`go run ./tools/corpora -mode select`, `:354-358` pipes its stdout to a file and parses
`minEdgesPerKind` and `lockedSet` out of it, `:390` makes byte-equality with that output an
acceptance criterion — *the single criterion the plan says proves the set is the global optimum
rather than merely a satisfying one* — and `:391` makes its non-zero exit the HALT trigger.

This is cycle-2 H4's defect class reintroduced at a new seam by the fix for cycle-2 H3. Two further
gaps ride on it, both invisible to `/gsd-execute-phase`:

1. **The stdout contract is unspecified.** `01-06-PLAN.md:355-357` assumes a JSON object with
   `minEdgesPerKind` and `lockedSet` keys. No behavior block anywhere declares it.
2. **The `eligible` universe is undefined at the CLI layer.** Both library functions take
   `eligible []string`; `01-05-PLAN.md:185-186` glosses it as "every candidate with an
   observation, regardless of lock state", but nothing says how `-mode select` derives it — and
   `01-06-PLAN.md:344-346`'s independent Python recomputation uses `obs.values()`, i.e. *all*
   observations. If the Go mode's eligibility ever differs from "all observations", that verify
   and the mode disagree by construction.

Listing `select` in the shared artifact inventory (`01-05-PLAN.md:549` and its six identical
copies) does not instruct anyone to build it: that block is a bill of materials, not a task.
`/gsd-execute-phase` can complete every 01-05 task, mark the plan done, and reach 01-06 wave 4 with
the command it is required to run absent.

**Plan change required.** Give `01-05` Task 2 (or a new Task 4) a `select` mode: action, behavior
block declaring the stdout JSON shape and the eligible-universe derivation, the non-zero
unsatisfiable-kinds exit, at least one test, and an acceptance criterion. Add it to the
`tools/corpora/main.go` artifact `provides` string at `01-05-PLAN.md:38`.

#### (d) Moving `corpora:assert` from wave 5 to wave 1 — NO REGRESSION

Consistent everywhere: defined at `01-04-PLAN.md:448-455`, in that plan's `provides` at `:45` and
its behavior criteria at `:476,484`; consumed at `01-06-PLAN.md:383,397,554`; explicitly
not-redefined at `01-07-PLAN.md:284-287` and consumed by the CI workflow step at `:407`. The shared
Taskfile-target inventory lists it identically in all seven plans. The empty-locked-set refusal
(`01-04-PLAN.md:476`) correctly covers the wave-1 state in which nothing is locked yet.

#### (e) MEDIUM — the atomic `mkdir` claim is not internally singular, and its downstream text is stale

Two incompatible sequences are given for the same operation:

- The numbered action reads stage (`01-04-PLAN.md:411-415`) → integrity-check the staged tree
  (`:417-418`) → **then** step 7's claim (`:419`).
- Step 7's own claim-acquired branch reads claim → **"stage, integrity-check the staged tree,
  `mv`"** (`:429-431`).

The difference is not cosmetic: claim-before-network serializes the whole shallow fetch across
processes, while claim-after-staging parallelizes transfer and serializes only promotion. Step 7's
claim-refused branch ("discard the staged copy", `:433`) presupposes the *second* ordering, which
step 5 contradicts.

Compounding it, two downstream passages still describe the superseded design:

- Threat row **T-01-04-04** (`01-04-PLAN.md:514`): "A racer that finds the destination already
  present validates it and discards its own copy rather than renaming over it." The claim
  mechanism is absent, and so is the new third branch — a loser finding the destination **absent
  or failing** now exits NON-ZERO naming the claim path (`:435-439`).
- Success criterion (`01-04-PLAN.md:618`): "Staging is unique per invocation and the concurrent
  loser validates-and-discards" — same stale framing, and it is the plan's own summary of what
  the task achieved.

This is precisely the pre-revision-text-left-behind class the regression hunt was asked to find.

**Plan change required.** Pick one exact sequence and write it once; update T-01-04-04 and the
success criterion to describe the claim protocol including the non-zero third branch.

### Remaining concerns

- **HIGH — `-mode select` has no implementation task** (defect (c) above). 01-06 cannot pass its
  own acceptance criterion at `01-06-PLAN.md:390`, and the "independently re-derived optimum"
  guarantee — the entire point of the cycle-2 H3 fix — is unverifiable as planned.
- **MEDIUM — the atomic fetch protocol gives two incompatible orderings and leaves two stale
  descriptions** (defect (e) above): `01-04-PLAN.md:411-418` vs `:429-431`, plus `:514` and `:618`.
- **MEDIUM — dead pre-revision text contradicts the upsert lifecycle.** `01-05-PLAN.md:90` still
  reads "**`corpora/observations.json`** — generated. Fully reconstructed by `task corpora:measure`."
  Full reconstruction at the default `locked` scope is *exactly* the design cycle-2 H1 proved
  broken, and it contradicts this plan's own must_haves truth at `01-05-PLAN.md:25` ("never deletes
  an entry outside its scope") and its Task 2 behavior at `:288-291`. The authoritative statements
  are correct; this one is leftover narrative in the objective — the first thing an executor reads.
  **Change needed:** reword `:90` to "upserted by `task corpora:measure`, keyed by `repo@sha`".
- **LOW — the Python verify blocks hand-restate the nine `RankEdges` members** at
  `01-05-PLAN.md:358`, `01-06-PLAN.md:206` and `01-06-PLAN.md:340`, while `query.RankEdges`
  (`internal/query/rwr.go:21-31`) is named plan-wide as the authority and `01-05-PLAN.md:249`
  greps to forbid restating them in Go. A tenth ranked kind would be omitted from observations and
  thresholds with every one of these checks still green. **Change needed:** derive the set in the
  verify (e.g. a `tools/corpora` mode that emits the ranked kinds), or add an acceptance criterion
  asserting the Python literal equals `query.RankEdges`.
- **LOW — `-out` does not scope the second artifact it claims to.** `01-05-PLAN.md:283-284` says
  `-out` "exists so a smoke run can write to a scratch path without touching the committed record",
  but the immediately preceding bullet (`:285-287`) says `-mode measure` always writes
  `docs/CORPUS-MEASUREMENT.md` — which is itself a committed record. Once Task 3 wires the renderer
  in (`:433-435`), the Task 2 verify commands that pass `-out /tmp/obs1.json` (`:353,365,366,367`)
  regenerate the committed document from a one-entry scratch file. Benign in first-pass task order,
  not benign on any re-run or phase-level re-verification. **Change needed:** either state that the
  document write is skipped when `-out` is non-default, or add a companion doc-path flag, and
  reword `:283-284` to match.
- **LOW — the concurrency verify does not check the leftover it advertises.** `01-04-PLAN.md:472`
  echoes "no leftover claim or staging" but tests only `test ! -d "$D.lock"` and
  `test -z "$(ls -d "$D".* 2>/dev/null)"` — and `$D.lock` is itself matched by `$D.*`, so the two
  are redundant with each other. Staging is a `mktemp -d` under the destination's **parent**
  (`01-04-PLAN.md:411`), i.e. `<parent>/tmp.XXXXXX`, which neither test can see. The step-8
  "remove the staging directory on every exit path" guarantee is therefore unverified.
  **Change needed:** snapshot the parent directory's entries before and after and assert equality
  apart from the destination.

### Overturned rejections

**None.** Every argued disposition re-checked this cycle holds: the two-file ownership split over a
merge (`01-05-PLAN.md:95-96`), the deliberate `internal/corpora` / `tools/bench/realcorpus`
separation with its Phase 6 handoff, the removal of the synthetic fallback, 01-06's HALT-and-escalate
with its two now-distinct conditions, the non-blocking fetch loser (`01-04-PLAN.md:437-439` — a
build tool that waits can deadlock a CI job), and the deliberate observations-upsert /
selection-single-owner asymmetry, which `01-05-PLAN.md:26` now defends on the correct grounds:
both sides of an observation merge are machine-produced and keyed `repo@sha`, so re-measuring a key
recomputes it wholly and a pin bump makes a new key rather than staling an old one.

### Risk assessment

**MEDIUM.** Substantially converged from cycle 2: eleven of twelve findings closed as executable
plan content, the four HIGH-severity internal contradictions reduced to one, and — importantly —
the revision introduced **no new cross-plan contradiction** in four of the five areas it touched.
The single remaining HIGH is a missing implementation task, not a design flaw: the algorithm, its
tests and its verification are all specified; only the CLI seam that exposes them was never
assigned to a task. That is a bounded, additive fix to one plan. The three text-consistency
findings are localized, and two of them are stale sentences rather than live specifications.

### Verification coverage (cycle 3)

Every finding recorded above was re-checked against plan text and repository source before being
counted:

| # | Finding | Verified at |
|---|---------|-------------|
| 1 | `-mode select` consumed, never created | Consumed: `01-06-PLAN.md:260,354,390,391`. Creation searched for and absent: `01-04-PLAN.md:312` ("two modes"), `01-05-PLAN.md:38,259,298` (measure only), `01-05-PLAN.md:392` (prose only). The only mention outside 01-06 is the shared inventory line naming `root`, `entries`, `measure`, `select` — identical in all seven plans. Independently found by both the reviewer and the orchestrator pass. |
| 2 | Fetch-protocol ordering ambiguity | `01-04-PLAN.md:411-418` (stage then check) vs `:429-431` (claim then stage then check); loser branch `:433` presupposes the latter |
| 3 | Stale racer text | `01-04-PLAN.md:514` (T-01-04-04) and `:618` (success criterion) vs the three-branch protocol at `:427-439` |
| 4 | "Fully reconstructed" contradicts upsert | `01-05-PLAN.md:90` vs `:25` and `:288-291` |
| 5 | Hand-restated nine kinds in Python | `01-05-PLAN.md:358`, `01-06-PLAN.md:206,340` vs `internal/query/rwr.go:21-31` and the Go-side prohibition at `01-05-PLAN.md:216-217,249` |
| 6 | `-out` does not scope the document | `01-05-PLAN.md:283-284` vs `:285-287`; verify sites `:353,365,366,367`; renderer wired at `:433-435` |
| 7 | Unverified staging leftover | `01-04-PLAN.md:472` (the two tests) vs `:411` (staging is `mktemp -d` under the parent) |

Cross-cutting sweeps that came back CLEAN: the `artifacts_this_phase_produces` block is byte-identical
across all seven plans (SHA-1 `3abd502f…`); `corpora/manifest.json`, `corpora/observations.json` and
`corpora/selection.json` are named uniformly; no `len(...) == 9` survives anywhere; every
`git diff --quiet` against a possibly-absent path carries its existence guard; and the wave graph
(`01-01`/`01-04` → `01-02` → `01-03`/`01-05` → `01-06` → `01-07`) has no target, flag, function or
mode consumed earlier than its defining wave — with the single exception recorded as finding 1.

No `REVIEWED-WITHOUT-REPO-ACCESS` marker was emitted; the reviewer cited repository `file:line`
evidence throughout and its citations were spot-checked against the tree.

---

## Consensus Summary (cycle 3)

One prompt-fed source-grounded reviewer (Codex) ran this cycle, cross-checked by an independent
orchestrator verification pass over the same plan text and repository. **Both independently and
first identified the `-mode select` gap as the cycle's only HIGH** — the strongest available signal
that it is real rather than an artifact of one reviewer's reading. The orchestrator pass
additionally contributed the "Fully reconstructed" stale-text finding, the `-out`/document-scope
finding, and the unverified-staging-leftover finding; the reviewer contributed the fetch-ordering
ambiguity and its two stale downstream passages, and the hand-restated-`RankEdges` robustness gap.

### Agreed strengths

- **The regression rate dropped sharply.** Cycle 2's revision introduced four HIGHs at four
  separate seams. This one introduced one, in the single area where the fix required a new
  executable surface — and left the other four areas it touched contradiction-free.
- **The fixes are argued in place, not merely applied.** Each superseded design is named and
  forbidden where it used to live: the circular threshold universe (`01-06-PLAN.md:263-268`), the
  `== 9` check (`01-06-PLAN.md:231`), the truncate-and-rebuild write (`01-05-PLAN.md:318-323`), the
  wave-5 placement of `corpora:assert` (`01-07-PLAN.md:284-287`), and both earlier promotion
  protocols (`01-04-PLAN.md:419-425`). A future reader cannot reintroduce them by accident.
- **The `min(max(2, best/2), best)` clamp is the right shape of fix.** It makes every derived
  threshold satisfiable *by construction* rather than testing for the bad case, and the plan says
  honestly that `best == 1` yields a threshold of `1` — D-14's tightening being arithmetically
  unavailable there — instead of inventing a bar nothing can clear.
- **The upsert/clean-diff composition is reasoned, not asserted.** `01-07-PLAN.md:296-306`
  demonstrates why D-17's keep-every-candidate and `corpora:drift`'s clean-diff hold together, and
  identifies `repo@sha` keying as the property the whole argument rests on.

### Agreed concerns

One: **`tools/corpora -mode select` is consumed by 01-06 and created by nobody.** It carries the
acceptance criterion that the entire cycle-2 H3 fix exists to make checkable, so its absence
silently removes the optimality proof rather than failing loudly. The remedy is additive and bounded
— one mode, one behavior block, one test, one acceptance criterion, in 01-05.

The recurring theme, now in its third cycle, is unchanged and worth naming plainly: **each revision
fixes the named defects and leaves one new seam where a plan consumes something no plan produces.**
Cycle 1 → 01-05's smoke test; cycle 2 → `corpora:assert`; cycle 3 → `-mode select`. The structural
remedy is not another review cycle but a mechanical cross-plan check before execution: for every
executable token a verify block or acceptance criterion names — Taskfile target, CLI mode, flag,
exported function — assert some earlier-or-equal-wave task's action creates it.

### Divergent views

None material — one prompt-fed reviewer. Two adjudications worth recording:

- The reviewer rated the hand-restated `RANK={...}` Python literals MEDIUM. The orchestrator
  **downgraded** them to LOW: the plans' no-restatement prohibition (`01-05-PLAN.md:216-217`) is
  explicitly scoped to `internal/corpora/record.go`, so this is a future-proofing gap rather than a
  contradiction with a stated contract, and nothing in Phase 1 adds a tenth ranked kind.
- The reviewer folded the fetch-ordering ambiguity and its stale downstream text into one MEDIUM;
  the orchestrator agrees and counts them as one finding, since a single edit to 01-04 Task 3
  resolves both.

### Convergence verdict

```
CYCLE_SUMMARY: current_high=1 current_actionable=5
```

Cycle-1 HIGHs still open: **0 of 8.** Cycle-2 HIGHs still open: **1 of 4** (the threshold/selection
HIGH, partially — its library design is fixed, its executable surface is unassigned). The one
remaining HIGH is a missing implementation task in a single plan, not a design defect: it is the
smallest and most localized state this phase has been in across three cycles, and the fix is
additive to `01-05-PLAN.md` alone.

---

# Cross-AI Plan Review — Phase 1, CONVERGENCE CYCLE 4 (USER-AUTHORISED EXTRA CYCLE)

Reviewed at `d05eb54`. Reviewer: Codex (prompt-fed, source-grounded, repo access confirmed).
Orchestrator ran an independent sweep of the same surfaces before reading the reviewer's output,
plus a scoped second sweep over 01-01/01-02/01-03/01-07.

Prior cycles: cycle 1 (`2166174`) 8 HIGH / 17 actionable; cycle 2 (`bce629b`) 4 HIGH / 8 actionable;
cycle 3 (`17e8a24`) 1 HIGH / 5 actionable. Plans revised at `d05eb54`.

## Codex Review

## Summary

The plans are substantially converged and generally trace the repository correctly: the status path, four render surfaces, wire-oracle target, existing Namespace cache, and workflow task-body guard all match the source. However, cycle 4 still contains two execution-blocking verification defects. Both are cross-plan sequencing errors introduced around the new `select` mode. The phase is not ready for execution until they are corrected.

## Consumed-but-not-created sweep

| Surface checked | Producers | Consumers | Result |
|---|---|---|---|
| `tools/corpora -mode root` | 01-04 Task 2 | 01-04 fetch/root checks | Clean |
| `-mode entries` and `-locked` | 01-04 Task 2 | 01-04 through 01-07 | Clean |
| `-mode measure` | 01-05 Task 2 | 01-05 through 01-07 | Clean |
| `-mode select` | 01-05 Task 2 | 01-05 and 01-06 | **Mismatch in 01-05 verification state; see HIGH-1** |
| `-mode kinds` | 01-05 Task 2 | 01-05 and 01-06 | Clean |
| Flags `-repos`, `-scope`, `-out`, `-prune` | 01-05 Task 2 | 01-05 through 01-07 | Clean |
| `--all-kinds` | 01-02 Task 2 | 01-02 and measurement path | Clean |
| `corpora:fetch`, `fetch-one`, `assert-one`, `assert` | 01-04 Task 3 | 01-04, 01-06, 01-07 | Clean; created before consumption |
| `corpora:measure` | 01-05 Task 2 | 01-06 and 01-07 | Clean |
| `corpora:verify`, `corpora:drift` | 01-07 Task 2 | CI/corpora workflow in 01-07 | Clean |
| Selection JSON fields `minEdgesPerKind`, `lockedSet`, `eligible` | 01-05 `select` contract | 01-06 verification | Structurally clean |
| `observations.json` / `selection.json` ownership | 01-05 / 01-06 | 01-06 and 01-07 | Clean |
| `/tmp/proposed.json` | 01-06 verification command at line 360 | Earlier command at line 342 | **Consumed before creation; HIGH-2** |
| Wire transcript capture flags `-bin`, `-fixture`, `-scenario` | Existing capture command | 01-03 | Clean; source declares all three at `test/wireoracle/cmd/wireoracle/main.go:20-27` |
| Workflow bare-task bodies | Existing guard plus 01-07 fixture update | 01-07 workflow | Clean; guard mechanism confirmed at `internal/upgrade/taskfile_shape_test.go:1335-1375` |

No later-wave artifact consumption was found beyond the verification-order defects below. The plan dependency graph itself is correctly ordered.

## Concerns

- **HIGH — 01-05’s new `select` verification cannot succeed in the state that plan creates.**

  Task 2 creates only `/tmp/obs1.json` and `/tmp/obs2.json` using non-default `-out` paths at `01-05-PLAN.md:395-407`. The plan explicitly says non-default `-out` does not write the committed observations document at `01-05-PLAN.md:294-302`. Nevertheless, the next commands invoke `go run ./tools/corpora -mode select` without an observations-path argument at `01-05-PLAN.md:410-411`, so it reads the default `corpora/observations.json`, which this task has not created.

  Even if it were changed to read `/tmp/obs1.json`, the success expectation remains contradictory: `SelectLockedSet` requires every priority language to have a non-zero count at `01-05-PLAN.md:209-214`, while the smoke record contains exactly one repository measured at `01-05-PLAN.md:395-407`. A single first manifest entry cannot cover Go, Java, C#, Python, and TS/JS. Under the mode’s declared contract, `select` must exit non-zero for that incomplete observation universe, but the verification expects valid JSON.

  This is the fourth-cycle instance of the priority defect class: the command now exists, but its required input state is not created before consumption.

- **HIGH — 01-06 reads `/tmp/proposed.json` before generating it.**

  The threshold verification opens `/tmp/proposed.json` at `01-06-PLAN.md:338-344`. The command that creates the file does not run until the following verification block at `01-06-PLAN.md:360-366`.

  On a clean executor this fails with file-not-found. On a reused environment it is worse: a stale `/tmp/proposed.json` could let the first threshold check validate the committed selection against an eligibility universe from an earlier run. That undermines the plan’s central claim that the eligibility universe is independently recomputed from current observations.

  The later block does correctly compare the recomputed thresholds and locked set, but it does not rescue the preceding failed or stale-input check.

## Suggestions

- For **HIGH-1**, edit 01-05 Task 2 so `select` is tested against a purpose-built observations fixture containing:

  - all five priority languages;
  - every `RankEdges` kind;
  - at least one valid satisfying subset.

  Add a flag such as `-observations <path>` if `-out` is intentionally measure-output-only, or explicitly define `-out` as the input path for `select` too. Then:

  - use the complete fixture for the successful stdout-contract checks;
  - keep the one-real-corpus smoke test for `measure`;
  - add a separate verification that the one-corpus record makes `select` exit non-zero and name the uncovered languages/kinds.

  Do not generate the committed `corpora/observations.json` early merely to make this verification pass; that would blur the ownership and wave boundary established for 01-06.

- For **HIGH-2**, move:

  ```sh
  go run ./tools/corpora -mode select > /tmp/proposed.json
  ```

  before every command that reads `/tmp/proposed.json`. Prefer one verification block that:

  1. removes or atomically replaces the temporary file;
  2. runs `select`;
  3. validates its exact three-key schema;
  4. recomputes threshold expectations using that same freshly generated file;
  5. compares thresholds and locked set to `selection.json`.

  This eliminates both missing-file and stale-file behavior.

## Risk Assessment

**HIGH.** The implementation design is mostly coherent, but two mandatory verification sequences are impossible or order-dependent as written. One directly repeats the consumed-input defect class emphasized for convergence cycle 4; the other can accidentally consume stale temporary evidence.

**Verdict: not ready for `/gsd-execute-phase`.** Make the two plan-only corrections above, then execution should be reconsidered.

---

## Orchestrator adjudication — cycle 4

The reviewer ran with repo access (no `[reviewed-without-repo-access]` marker) and its two HIGHs
were both reproduced independently by the orchestrator before its output was read. A second scoped
sweep covered 01-01, 01-02, 01-03 and 01-07, which the reviewer treated more lightly.

### Independent consumed-but-not-created sweep (orchestrator)

Mechanical, over all seven plans at `d05eb54`:

| Surface | Referenced | Created | Result |
|---|---|---|---|
| `-mode root` | 01-04 | 01-04 Task 2 | clean |
| `-mode entries` | 01-04, 01-05 | 01-04 Task 2 | clean |
| `-mode measure` | 01-05, 01-06, 01-07 | 01-05 Task 2 | clean |
| `-mode select` | 01-05, 01-06 | 01-05 Task 2 | mode created — **input state not created**, see H-4.1 |
| `-mode kinds` | 01-05, 01-06 | 01-05 Task 2 | clean; `query.RankEdges` confirmed at `internal/query/rwr.go:21` with exactly 9 members, and `internal/query/rwr_test.go:25` already pins the count |
| `-locked` | 01-04:316 | 01-04 Task 2 | clean |
| `-repos` `-scope` `-out` `-prune` | 01-05, 01-07 | 01-05 Task 2 (`:278-308`, `:350`, `:366`) | clean |
| `--all-kinds` | 01-02 | 01-02 Task 2 | clean |
| `task corpora:fetch` / `fetch-one` / `assert-one` / `assert` | 01-04, 01-05, 01-06, 01-07 | 01-04 Task 3 (wave 1) | clean — created strictly before every consumer |
| `task corpora:measure` | 01-05, 01-06, 01-07 | 01-05 Task 2 (wave 3) | clean |
| `task corpora:verify` / `corpora:drift` | 01-07 only | 01-07 Task 2 (wave 5) | clean |
| `task lint:actions` | 01-07 | pre-existing, `Taskfile.yml:3225` | clean |
| stdout keys `minEdgesPerKind` / `lockedSet` / `eligible` | 01-06:342-344, :362-365 | 01-05:286-291, :333-337 | shape clean; **ordering broken**, see H-4.2 |
| `corpora/observations.json` | 01-06, 01-07 | first written at 01-05:499 (wave 3, default `-out`) | see H-4.1 |
| `corpora/selection.json` | 01-06, 01-07 | 01-06 Task 2 (wave 4), hand-authored | clean |
| wireoracle `-bin` / `-fixture` / `-scenario` | 01-03 | pre-existing, `test/wireoracle/cmd/wireoracle/main.go` | clean |

The mode/flag/target registry blocks reproduced verbatim in all seven plans
(`01-01:305-310` … `01-07:573-578`) agree with each other and with the producers above. **The
enumerated-token half of the class is clean for the first cycle in four.** What is not clean is the
class's next form: a command that exists, invoked in a state its input does not yet exist in.

### Current HIGH concerns

**H-4.1 (NEW, introduced at `d05eb54`) — 01-05 Task 2's `-mode select` verifies cannot pass, on two
independent grounds.**

`01-05-PLAN.md:410-411` run `go run ./tools/corpora -mode select` and pipe stdout into `python3
json.load`. At that point in task order:

1. *The input file does not exist.* Every measure invocation in that verify block redirects to a
   scratch path — `-out /tmp/obs1.json` (`:395`), `/tmp/obs2.json` (`:407`), `/tmp/obs3.json`
   (`:408`), `/tmp/obs4.json` (`:412`) — and `:294-299` states that a non-default `-out` also
   suppresses the document write. Nothing writes the default `corpora/observations.json` until
   Task 3's verify at `:499`. `corpora/` does not exist in the tree today. Task 1's own behavior
   block settles what happens then: "`LoadObservations` and `LoadSelection` on a missing file
   return an error naming the path — never an empty document" (`:156-157`). So `select` exits
   non-zero with empty stdout and the `json.load` fails.
2. *Even given an observations file, the assertion is unsatisfiable at this wave.* `SelectLockedSet`
   requires every `PriorityLanguages` member to have a non-zero file count (`:209-213`), and
   `PriorityLanguages` is five languages (`:183`). `01-05:418` insists — correctly — that the smoke
   path measure exactly the one corpus it fetched. One repository cannot supply Go, Java, C#, Python
   and TS/JS, so `select` must take its declared no-qualifying-subset branch (`:288`, `:338-339`)
   and exit non-zero. That branch is 01-06's HALT trigger; here it is fed to a contract assertion.

Compounding both: the behavior block never says **which** observations file `select` reads, or
whether it honours `-out`. `-out` is defined as the file `measure` "read-modify-write"s (`:294`),
and `select` "writes no file" (`:340`), so an executor has no declared way to point `select` at a
scratch fixture even if they diagnose the problem.

*Fix (01-05 Task 2 only):* declare `select`'s input path explicitly — either that it honours `-out`
as a read path, or a separate `-observations <path>` flag — then split the verify into (a) a
synthetic multi-language fixture that exercises the success contract, and (b) an assertion that the
one-real-corpus record makes `select` exit non-zero naming the uncovered languages. Do **not** write
the committed `corpora/observations.json` early to make this pass: that would move wave-4 evidence
into wave 3 and blur the ownership boundary 01-05 and 01-06 were split to protect.

**H-4.2 (NEW, introduced at `d05eb54`) — 01-06 Task 2 reads `/tmp/proposed.json` one verify block
before the command that writes it.**

`01-06-PLAN.md:342` does `proposed=json.load(open('/tmp/proposed.json'))` and `:343-344` derives the
whole `ELIGIBLE` universe from it. The only producer is `:360`, `go run ./tools/corpora -mode select
> /tmp/proposed.json`, in the *next* `<automated>` block. `git diff 17e8a24 d05eb54` confirms line
342 is new in this revision; this is not inherited.

On a clean executor the first block dies with `FileNotFoundError`. On a reused machine the failure
mode is worse and silent: a stale `/tmp/proposed.json` from an earlier attempt supplies the
eligibility universe, and `:344`'s cross-check (`set(ELIGIBLE)==set(proposed['eligible'])`) compares
the stale file against itself, so it cannot detect the substitution. The threshold recomputation at
`:350-358` — the criterion at `:396` that is the whole point of making selection deterministic —
would then validate the committed selection against a universe from a previous run.

*Fix (01-06 Task 2 only):* merge `:338-359` and `:360-366` into one block that `rm -f`s the path,
runs `select` into it, validates the three-key schema, and only then recomputes thresholds and
compares against `corpora/selection.json`.

### Current actionable non-HIGH concerns

- **MEDIUM — `01-04:323` still asserts a count where `01-04:336` demands set equality.** The verify
  is `assert len(e)==9, ('… an exact count, not a lower bound', len(e))`; the criterion it backs
  says "EXACTLY the nine seeded repositories — **set equality against the enumerated list, not a
  count** and not a lower bound", with six lines of rationale (`:338-347`) explaining why a count is
  insufficient. Cycle 3 asked for `>= 9` → set equality; the revision delivered `>= 9` → `== 9`.
  Swapping one seeded repository for another still passes. *Fix:* assert
  `{x['repo'] for x in e} == {…the nine names…}` in `01-04:323`.

- **MEDIUM — `01-07:476` is unsatisfiable under this repository's action-pinning convention, and
  satisfying it literally would un-pin two actions.** The criterion reads "the workflow contains
  exactly one 40-character hex string, the cache action pin, and no corpus SHA". The same plan's
  step list mandates Checkout (`:389`) and Set up Go (`:390`), and every `uses:` in
  `.github/workflows/` is SHA-pinned — `actions/checkout@df4cb1c0…` and
  `actions/setup-go@924ae3a1…` at `ci.yml:51,54`, zero unpinned references repo-wide. The new
  workflow will carry three 40-hex strings. In a project whose stated core value is a verifiable
  supply chain, an executor taking this criterion literally un-pins two actions. *Fix:* reword to
  "no 40-hex string other than the pinned action refs — in particular no corpus SHA", or make it a
  `grep -cE '[0-9a-f]{40}' … returns 3`.

- **MEDIUM — `01-07:435`'s path-filter verify cannot fail, which is the vacuous-guard shape this
  phase exists to eliminate.** The command is `for p in …; do grep -q "$p" … || echo "MISSING PATH
  FILTER: $p"; done; echo "path filter checked"` — a miss prints a line and the loop exits 0
  regardless, then an unconditional `echo` seals it. Criterion `:471` ("the measuring algorithm
  cannot change without this job running") has no failing command behind it. *Fix:* collect misses
  and `exit 1`, or drop the `|| echo` and let `grep -q` fail the block.

- **MEDIUM — `01-06:337`'s idempotence check tests working-tree cleanliness, not idempotence, and
  fails for an unrelated reason.** `task corpora:measure && task corpora:measure && git diff --quiet
  -- corpora/observations.json docs/CORPUS-MEASUREMENT.md`: with no commit argument `git diff`
  compares the worktree to the index, and Task 2's own action (`:328`) has already regenerated both
  files relative to what Task 1 committed. The block therefore exits non-zero whether or not
  regeneration is idempotent. (Were the files instead untracked at that moment, it would pass
  vacuously — `git diff` ignores untracked paths.) *Fix:* capture `cksum` / `sha256sum` of both
  files after the first `task corpora:measure` and compare after the second.

- **LOW — `| tail -N` on `go test` masks the exit status the paired criteria assert.** A pipeline's
  status is `tail`'s. `01-03:234` is `go test -count=1 ./... 2>&1 | tail -20` while `01-03:243`
  claims "`go test ./...` exits 0"; the same pairing appears at `01-07:444` and, less critically, at
  `01-01:160,162`, `01-02:155`, `01-03:150,151,232,233`, `01-07:246,247,324,325,326,430,431`.
  `01-02:156,287` and `01-07:321-323` are the correctly-shaped counterexamples. *Fix:* at minimum
  un-pipe the two full-suite gates, or prefix with `set -o pipefail`.

- **LOW — `01-07:331` and `:332`'s Taskfile greps are fragile against text the plans instruct the
  executor to write.** `Taskfile.yml` contains zero occurrences of `prune` and zero of
  `selection.json` today, so both start satisfiable, and no other plan adds either to a Taskfile
  target body — `-prune` is a tool flag (`01-05:366`), not a task argument. The risk is
  self-inflicted: `01-07:307` ("which is exactly why this target must never pass it") and
  `01-05:390` ("record in its description … that it never writes the curated selection file") both
  invite the executor to write the reasoning into a `desc:`, and a `desc:` is not a comment.
  `:332` carries a `planner-discipline-allow` escape; `:331` does not. *Fix:* scope both greps to
  the target body, or extend `:331`'s allowance to match `:332`'s.

### Dispositions — raised and NOT counted

- **`$BASE_SHA` in `01-03:152` and criteria `:158-160`.** Not an undefined variable: `01-03:136-141`
  instructs the executor to capture `git rev-parse HEAD` into the SUMMARY before any edit and use
  that recorded SHA as the diff base, and each criterion restates the provenance inline. Incorporated
  into action and acceptance_criteria — RESOLVED.
- **`01-01:170`'s pre/post timing halt rule with no baseline command.** The criterion is explicitly
  "(cost, **recorded not asserted**)" and the action carries the before/after instruction. RESOLVED.
- **`01-02:227`'s `assert 0 in d.values()` as an environmental assertion.** Grounded in the phase's
  own load-bearing premise — this repository's idiomatic Go produces zero `overrides` and zero
  `type_of` (FIXT-01 criterion 3, `01-06:25`). Not a latent flake.
- **`nscloud-cache-action`'s `path:` input having no precedent in this tree.** `01-07:424-427`
  already names it as "one first-use unknown to confirm on the real run rather than assert" with a
  `<human-check>` behind it. RESOLVED by explicit deferral.
- **01-04's fetch protocol and concurrency verify.** Traced end to end. The claim → stage → check →
  promote sequence at `01-04:421-461` is internally consistent: step 4 rules out an existing
  destination before the claim, so step 7's `mv` cannot nest; both claim-refused branches terminate;
  step 8's cleanup obligation is what `01-04:492`'s parent-directory snapshot actually tests, and a
  `mktemp -d` staging directory under that parent is visible to it where a `$D.lock` / `$D.*` probe
  would not be. `wait` with no operands returns 0, so the losing racer's non-zero exit correctly does
  not fail the block. No finding.
- **Wave ordering.** No plan consumes an artifact produced by a later wave. `corpora:assert`'s
  placement in 01-04 (wave 1) rather than 01-07 (wave 5), for 01-06 (wave 4), is correct and is
  argued in place at `01-07:285-288`.

### Trajectory

25 → 12 → 6 → **8** (2 HIGH + 6 actionable).

The rise is not a regression in the plans' substance. The enumerated consumed-but-not-created
surface — modes, flags, task targets, JSON keys — is **clean for the first time in four cycles**,
and nothing any plan argues has been overturned. What this cycle bought is a different, narrower
class: the token exists, but the *state* it needs does not yet exist at the point of invocation
(H-4.1, H-4.2), plus five verification-formulation defects that the previous three cycles' focus on
existence never looked for. Both HIGHs are confined to a single task in a single plan each, and
both fixes are local edits to a verify block with no cross-plan coordination.

### Convergence verdict

```
CYCLE_SUMMARY: current_high=2 current_actionable=6
```

Cycle-1 HIGHs still open: **0 of 8.** Cycle-2 HIGHs still open: **0 of 4.** Cycle-3's HIGH: **closed**
— `-mode select` now has a real creating task at `01-05:286-291`, `:323-341`, `:427-429`. Both HIGHs
above are new at `d05eb54`, making this the fourth consecutive revision to introduce a defect while
closing the prior cycle's. Consistent with the previous three cycles, the phase is **not ready for
`/gsd-execute-phase`** as it stands; the two HIGH fixes touch `01-05-PLAN.md` Task 2 and
`01-06-PLAN.md` Task 2 only.
