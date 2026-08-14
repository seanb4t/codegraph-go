---
phase: 1
reviewers: [codex]
reviewed_at: 2026-08-14T02:04:59Z
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
