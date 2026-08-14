# Phase 1: Corpus Selection by Measurement - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

A **blocking measurement spike**. It indexes candidate third-party MIT / Apache-2.0 repositories, records real per-edge-kind and per-language counts from actual indexing output, locks a final corpus set on that evidence, and makes that set reproducibly fetchable at pinned commit SHAs without vendoring any corpus source.

**No phase that re-freezes a golden may be planned until this one resolves.**

Delivers FIXT-01 (measurement + locked set) and FIXT-02 (reproducible pinned fetch + CI cache).

In scope, as decided below: extending `codegraph status` with per-edge-kind counting (the measuring instrument does not exist today), a corpora manifest, a measurement record, a Taskfile fetch target, the repo's first `actions/cache` usage, and a coverage drift guard.

Not in scope: renaming or re-freezing any golden (Phase 2), the non-vacuity mutation proof (Phase 3), any attribution/documentation/process sweep (Phases 4–6).

</domain>

<decisions>
## Implementation Decisions

### Measurement instrument

Scouting established the load-bearing fact: **no per-edge-kind counting surface exists in the tree.** `internal/cli/status.go` renders "Nodes by Kind"; there is no edge aggregation anywhere in the CLI. FIXT-01 cannot be satisfied without building the instrument first.

- **D-01:** The instrument is an extension of `codegraph status`, not a throwaway binary under `tools/`. `status` gains an **Edges by Kind** section alongside the existing Nodes by Kind. The trade-off was raised and the user chose product surface over spike-scoped tooling: the capability is discoverable and reusable rather than deleted with the phase. — **Reversibility:** costly — removing it later means retracting a shipped CLI section, a `--json` field, and an MCP resource field, and re-freezing the wire-oracle transcript a second time.

- **D-02:** The new section lands on **all three surfaces** — human text, `--json` (`query.StatusResult` / `query.MarshalStatusJSON`), and the MCP status resource. Consequence accepted: **Phase 1 owns a re-freeze of `testdata/wireoracle/transcripts/resources-read-status.golden`**, which the roadmap's sequencing had assumed Phase 2/3 would own. The payoff is that the measurement record is produced by `codegraph status --json` directly, with no bespoke parsing path. — **Reversibility:** costly — the `--json` shape and the MCP resource shape are consumed contracts.

- **D-03:** `StatusResult.FilesByLanguage` (`internal/query/status.go:57`) **already exists but carries `json:"-"`**, suppressing it. That suppression exists to match an output shape the project no longer owes anyone — the Compatibility constraint was formally retired 2026-08-13 by this milestone. Phase 1 **un-suppresses it in the same diff** as `edgesByKind`. Both halves of FIXT-01's data then come from one `status --json` call. `languages []string` stays (redundant with the map's key set, but harmless — removing it is a breaking JSON change and was explicitly not chosen). — **Reversibility:** costly — same consumed-contract argument as D-02.

- **D-04:** `edgesByKind` is **sparse by default, dense behind an opt-in flag.** Rationale from the user: everyday `status` should not be buried in zeros, so density is a deliberate act rather than the norm. In dense mode the map carries **all 9 `RANK_EDGES` kinds including explicit zeros**, and its **key set is DERIVED from the existing `RANK_EDGES` constant, never hand-listed** — with a test asserting key-set equality. This closes the absent-vs-measured-zero ambiguity (rule `84d1gfpywd`) that FIXT-01 criterion 3 depends on, and means a future 10th edge kind cannot go silently unmeasured. **The measurement run uses the dense flag.**

- **D-05:** `codegraph://status` (the MCP resource) is argument-less and cannot opt in. It emits **sparse**, matching the flagless CLI default. CLI-default and MCP therefore agree; density stays CLI-only.

### Measurement record shape

- **D-06:** The record is **machine-readable JSON as source of truth, with generated prose.** Raw `status --json` output per candidate, keyed by `repo@SHA`, is committed; the human-readable markdown is generated from it. The prose can never drift from the numbers, and a downstream guard has something to count against. A hand-written-markdown-only record was rejected precisely because numbers become prose the moment they are transcribed.

- **D-07:** The record is **re-runnable, not frozen once.** A Taskfile target re-indexes the locked corpora and regenerates the JSON, and a **drift guard asserts the coverage claim still holds** (every kind clears its bar, every priority-4 language non-zero). This is the positive-assertion posture rule `84d1gfpywd` requires — bumping a pinned SHA that kills `type_of` coverage fails loudly instead of passing silently.

- **D-08:** Guard cost is split by trigger. **Every CI run** performs the cheap check: assert the *committed* JSON satisfies the coverage claim — no indexing, no corpora needed. A **path-filtered job re-indexes and diffs only when the pinned-SHA manifest changes.** Re-indexing hugo/guava on every push was rejected on wall-clock; manual-only re-measurement was rejected because a pin bump with stale numbers would pass CI, which is the exact silent-narrowing failure this phase exists to prevent.

- **D-09:** **One corpora manifest is the sole pin authority.** It carries, per entry: repository, commit SHA, license, and a `locked` flag. The Taskfile fetch target, the CI cache key, and the drift guard all read it; the measurements JSON references entries by `repo@SHA`. **Rejected candidates remain in the manifest marked unlocked** — measured, recorded, not fetched. Restating SHAs in Taskfile vars was rejected because two copies of a SHA create a third invariant to police.

### Corpus scope & fetch

- **D-10:** A corpus entry means a **whole repository, always** — no subtree pinning. `repo@SHA` is the entire input, so there is nothing about scope to get wrong or to drift. **Known consequence, flagged during discussion and accepted:** this likely rules `apache/arrow` out on size alone, which makes **finding a dedicated C# corpus part of Phase 1's job** rather than a monorepo slice. — **Reversibility:** reversible — adding an optional `subtree` field to the manifest later is additive.

- **D-11:** Fetch is a **shallow git fetch at the pinned SHA** (`git init` + `git fetch --depth 1 origin <sha>` + checkout). Chosen over a codeload tarball specifically because **the fetched tree keeps a real `.git` directory**, so the corpora exercise codegraph's git/worktree-aware indexing path (`internal/gitmeta`, `StatusResult.WorktreeMismatch`) — matching how users actually index repositories. A tarball corpus would have left that path unexercised, and Phase 2's goldens would have inherited the gap.

- **D-12:** Fetched corpora land **outside the working tree**, at an **XDG-relative default** — `${XDG_CACHE_HOME:-$HOME/.cache}/…` — **overridable via `CODEGRAPH_CORPUS_DIR`**. This keeps the repo small and shares one fetch across worktrees. It echoes the existing `CODEGRAPH_{JAVA,PYTHON,TSJS}_CORPUS` env-var convention already present in the parity tests. An in-tree gitignored path was explicitly not chosen.

- **D-13:** CI caching is **one `actions/cache` entry per corpus, keyed `corpus-{repo}-{sha}`.** This is the **first `actions/cache` usage in the repository** — no workflow uses it today. Content-addressed keying means a hit is provably the right tree, so **no `restore-keys` prefix fallback** is used: a prefix match on a SHA key would be wrong by definition and would silently hand back another SHA's tree. Per-corpus granularity matters because GitHub gives a repo **10 GB of total cache with LRU eviction** — a single blob keyed on the manifest hash would refetch everything on any one pin bump and is likelier to hit the ceiling alone. **FIXT-02's requirement stands: a cache miss falls through to a fetch, never to a skip.**

### Coverage bar & fallback

- **D-14:** The bar is a **threshold per kind, not the roadmap's stated "non-zero" floor.** This is a deliberate tightening — a suite that clears a threshold also clears non-zero, so FIXT-01's criteria remain satisfied.

- **D-15:** **N is set from measured data, then frozen.** The spike measures first; the threshold is chosen against the real observed distributions and frozen into the drift guard. A number invented up front risks being either vacuous or one no real corpus clears. **The rationale for the chosen N must be written into the measurement record explicitly** — N is selected by the same pass it judges, so the reasoning is the only thing making it reviewable.

- **D-16:** Fallback is **real corpus first, behavioral corpus as a recorded fallback.** If a kind misses its bar across the shortlist, search continues into additional candidates; only if the gap survives that does the purpose-built corpus (the one surviving as `corpus/behavioral/` under FIXT-05) get a deliberately-added targeted case. **When the fallback is used, the measurement record must state that the kind is covered synthetically rather than by a third-party repository** — synthetic coverage is weaker evidence and must not read as equivalent.

- **D-17:** The search is bounded by **candidate count** — a fixed number of additional candidates beyond the shortlist. Once exhausted, the behavioral-corpus fallback triggers. **Every candidate tried is named in the record with what it scored**, so failed candidates become evidence rather than lost effort. (The specific count was left to planning; "one extra round" and "fall back immediately" were both offered and not chosen.)

### Claude's Discretion

The user did not defer any area wholesale. Items left open by explicit choice to move on, and therefore open to the planner's judgment:

- The dense-mode flag's name (`--all-kinds` / `--verbose` / other), and whether `nodesByKind` should receive the same dense treatment for consistency.
- Manifest file format (YAML vs JSON) and its path; where the generated prose document lands (`docs/` vs `testdata/golden/`); whether the generated prose is regenerated-and-diffed in CI or written once.
- What "running it twice from clean produces the same tree" is verified *by* (tree hash vs file count vs other), and whether the fetch target is idempotent by default or needs `--force`.
- The exact extra-candidate count in D-17, and the exact N in D-15 (which is data-driven by construction).
- Whether the 5 priority-4 languages carry a file-count threshold or stay at non-zero — only the *edge kinds* got a threshold decision.
- Whether a behavioral-corpus fallback is permissible for a *language* miss as well as an edge-kind miss.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone scope and sequencing
- `.planning/ROADMAP.md` §"Phase 1: Corpus Selection by Measurement" — the 5 success criteria, and the "Notes" paragraph establishing the shortlist as a starting point rather than a decision
- `.planning/ROADMAP.md` §"🚧 v0.11.0 — Standalone Project Identity" — the ordering rationale, in particular the first bullet explaining *why* Phase 1 blocks (v1.0 Phase 1's finding that this repo's own idiomatic Go produces zero `overrides` and `type_of` edges)
- `.planning/REQUIREMENTS.md` — FIXT-01 and FIXT-02 verbatim; the Out of Scope table's vendoring row (fetch-at-pinned-SHA was chosen to avoid adding redistribution obligations to the `NOTICE` file this milestone trims)

### The measuring instrument (D-01 … D-05)
- `internal/query/status.go` — `StatusResult` (line 46), specifically `NodesByKind` (57), `FilesByLanguage` with its `json:"-"` tag (57), and `Languages` (58); `MarshalStatusJSON` is the `--json` contract
- `internal/cli/status.go` — the human renderer; its doc comment at line 19 enumerates the current section order (`Index Statistics:` / `Nodes by Kind:`), and line 75 registers `--json` / `-j`
- `internal/cli/status_cli_test.go` — the human-output assertions that D-02 will move
- `testdata/wireoracle/transcripts/resources-read-status.golden` — the MCP status resource transcript that D-02 commits Phase 1 to re-freezing
- Wherever `RANK_EDGES` is defined and consumed — `internal/query/scoring.go`, `internal/query/explore.go`, `internal/query/expand.go`, `internal/query/rwr.go`, `internal/indexer/resolve.go`, `internal/indexer/goextract/types.go`. **D-04 requires the dense key set be derived from this constant, not restated.**

### Fetch, cache, and the existing corpus situation
- `Taskfile.yml` §`test:golden` (line 59) — the existing golden target and its GOLDEN-01 rationale (`go list ./...` never discovers `testdata/`); the new fetch target sits beside it
- `.github/workflows/ci.yml` lines 89–95 — the golden-suite step and the comment explaining why it is separate. **No workflow in this repo uses `actions/cache`; D-13 introduces the first.**
- `testdata/golden/README.md` — the Corpus table
- `testdata/golden/corpus/` — note that `weft-go/` and `colbymchenry-codegraph/` contain **captured JSON output, not source**; only `synthetic-parity/` ships an actual `src/`. A source-fetch path is therefore net-new, not a replacement.
- `testdata/golden/golden_parity_test.go` lines 113–268 — the current self-skip and runtime-`git clone` behavior these decisions replace
- `testdata/golden/parity_{java,python,csharp,tsjs}_test.go` — the `CODEGRAPH_{LANG}_CORPUS` env-var convention D-12 echoes
- `.gitignore` — carries `testdata/golden/corpus/*/src/.codegraph/`; D-12 lands corpora outside the tree, so this may need no change

### Standing repo rules that shaped these decisions
- engram rule `84d1gfpywd` — a guard MUST carry a positive assertion; negative-only guards pass vacuously. Directly shaped D-04, D-07 and D-08.
- engram rule `f18zrdsgx5` — never `[skip ci]` in a commit message in this repo (`protect-main` requires 6 status checks).
- engram record `gw79qy2a9z` — the v0.11.0 scoping decision, including that PROJECT.md's Compatibility constraint was **retired** 2026-08-13. This is the basis for D-03.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `query.StatusResult` / `query.MarshalStatusJSON`: the whole measurement payload rides on this one struct once `edgesByKind` is added and `filesByLanguage` un-suppressed. No second data path is needed.
- `StatusResult.FilesByLanguage`: **already computed today**, merely suppressed from JSON. Half of FIXT-01's data exists and is being discarded.
- The `RANK_EDGES` constant: already the single vocabulary of the 9 kinds, consumed across `internal/query/` and `internal/indexer/`. D-04 makes it the source of the dense key set.
- `testdata/golden/gocapture/`: existing precedent for a Go program that drives indexing and emits structured output; its `main.go` already references the `RANK_EDGES` set.
- `tools/bench/` and `tools/mcpaudit/`: established `tools/` convention, if any helper binary turns out to be needed alongside the `status` extension.
- `CODEGRAPH_{JAVA,PYTHON,TSJS}_CORPUS` env vars in the parity tests: the naming convention `CODEGRAPH_CORPUS_DIR` follows.

### Established Patterns
- **Goldens live outside `go list ./...`.** `testdata/` is invisible to `./...` expansion (GOLDEN-01), which is why `task test:golden` exists as its own target and why CI runs it as a separate step. Any new corpus-dependent test inherits this.
- **Positive assertions over negative guards** (rule `84d1gfpywd`). The drift guard in D-07/D-08 must assert coverage holds, not merely that nothing failed.
- **One reviewed diff, one named cause.** The milestone's ordering rationale is explicit that a change must be attributable. D-02's transcript re-freeze should be separable from the `edgesByKind` feature diff.
- **Self-skipping is the failure mode being retired.** Today's golden tests `t.Skip` when a corpus is absent (`golden_parity_test.go:149`, `:250`, `:268`) and one performs a runtime `git clone`. FIXT-03 kills this in Phase 3, but Phase 1's fetch target is what makes that possible.

### Integration Points
- `internal/query/status.go` → `internal/cli/status.go` → the MCP status resource: one struct change propagates to all three of D-02's surfaces.
- Corpora manifest → Taskfile fetch target, `actions/cache` key, and drift guard: three consumers of one file (D-09).
- Corpora landing path (D-12) → consumed by Phase 2's re-frozen goldens and Phase 3's unconditional-execution guard. **This path is a cross-phase contract, not a Phase 1 local detail.**
- `.github/workflows/ci.yml` gains its first cache step; `bench.yml` may later want the same corpora (Phase 6) — worth not painting the cache design into a CI-job-local corner.

</code_context>

<specifics>
## Specific Ideas

- **"Only at a verbose, or 'include all edges' option — so general use which doesn't need the absolute clarity isn't killed with a bunch of zeros."** (D-04) The user's framing: rigor should be available on demand, not imposed on the default reading experience.
- **XDG-relative, not in-tree** (D-12). The offered option defaulted to in-tree; the user overrode it to `XDG_CACHE_HOME`-relative.
- The shortlist to measure, from the roadmap and explicitly **not locked**: `gohugoio/hugo` (Apache-2.0, Go), `nestjs/nest` (MIT, TypeScript), `google/guava` (Apache-2.0, Java), `apache/arrow` (Apache-2.0, mixed monorepo incl. C#). D-10's whole-repo rule puts `arrow` at real risk on size; a dedicated C# candidate should be sought.
- Every candidate must be **MIT or Apache-2.0**. Non-negotiable, from REQUIREMENTS.md.

</specifics>

<deferred>
## Deferred Ideas

- **Dense `nodesByKind`** — D-04 makes `edgesByKind` dense-on-demand; whether `nodesByKind` should gain the same opt-in density for symmetry was raised and deliberately left. A later phase or milestone, not this one.
- **Dropping `StatusResult.Languages`** — offered alongside D-03 and not chosen, because it is a breaking JSON removal. If `filesByLanguage` becomes the canonical form, retiring `languages` is a clean follow-up with its own scope.
- **A file-count threshold for the 5 priority-4 languages** — D-14 set a threshold for edge *kinds* only. Languages remain at the roadmap's non-zero bar unless planning revisits it.

### Reviewed Todos (not folded)
All four Phase-1 todo matches were reviewed and deliberately **not folded** — each belongs to a different subsystem, and Phase 1 is a blocking spike that should not widen.

- **Add golangci-lint with gofmt and idiomatic Go linters** (score 0.9, area: ci) — matched on keyword overlap (add / idiomatic / taskfile / repository), not on subject. Phase 1 does add Go code and a Taskfile target, but introducing a linter is its own CI change.
- **`release:dry-run-signed`'s additions-only diff guard passes vacuously when the awk anchor stops matching** (0.6, area: release) — the *same defect class* as rule `84d1gfpywd`, which shaped D-04/D-07/D-08. Thematically adjacent and worth fixing, but it would put a release-workflow diff inside a corpus-measurement phase.
- **brew trust instructions recommend the broader `--tap` grant and carry no security framing** (0.4, area: docs) — unrelated; matched on "third"/"party".
- **`post-release-verify.yml`'s event-aware conclusion guard has no test asserting it** (0.3, area: ci) — again the vacuous-guard class, again the wrong subsystem.

</deferred>

---

*Phase: 1-Corpus Selection by Measurement*
*Context gathered: 2026-08-13*
