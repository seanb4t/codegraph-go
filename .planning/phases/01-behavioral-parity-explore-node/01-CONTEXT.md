# Phase 1: Behavioral Parity — explore & node - Context

**Gathered:** 2026-07-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver TS-CodeGraph-1.3.1-identical **`explore`** and **`node`** behavior in the
shared `internal/query.Engine` so the CLI command and the `codegraph_explore`
MCP tool improve in the same commit:

- **`explore`**: multi-word variadic `<query...>` (tokenized
  CamelCase/snake_case/acronym/dot-notation/plain + stopword filter) →
  graph-relevance ranking via Random-Walk-with-Restart (α=0.25, ~25 iters, TS's
  9 edge kinds + type-hierarchy/BFS/glue-node expansion + named-seeding) → a
  file-level relevance gate that stops weakly-connected `Test*` funcs from
  topping results → a per-root "⚠️ no covering tests" warning.
- **`node`**: enumerate ALL exact-name definitions of an overloaded symbol
  (generated-files-last), the "N definitions named X — returning M in full"
  header, full bodies to TS's budget (≤16 defs / 12,000 chars) + an overflow
  list of the rest; optional file/line narrowing that never empties the set;
  single-definition output stays byte-comparable to TS.
- **TEST-01 harness**: a behavioral fixture harness diffs both commands against
  TS 1.3.1 for ambiguous names, multi-word queries, relevance ordering, and
  coverage warnings — on BOTH the CLI and MCP surfaces — closing v0.1's
  single-symbol golden blind spot. The harness lands with the algorithm, not
  after (template-parity ≠ behavior-parity).

**Requirements:** EXPL-01, EXPL-02, EXPL-03, EXPL-04, EXPL-05, NODE-01, NODE-02,
NODE-03, NODE-04, TEST-01 (10 total).

**Hard constraints for this phase:**
- **Plain text only** — no styling/color/ANSI anywhere in explore/node output.
  The build-enforced archtest lands in Phase 6, but the constraint holds from
  day one (agent/MCP path stays parseable).
- **Shared engine** — the algorithm lives in `internal/query.Engine`; CLI + MCP
  are the same code path (EXPL-05/NODE-04 are structural, not extra plumbing).
- **Highest-risk, load-bearing phase.** EXPL-02's RWR is the single hardest
  item and puts the golden-corpus contract at stake.

**⚠️ SCOPE EXPANSION (post-research, user-decided 2026-07-14 — see D-09/D-10).**
Research (01-RESEARCH.md, commit 6de928a) discovered the roadmap/requirements
ASSUMED 9 RWR edge kinds exist, but the v0.1 extraction pipeline emits only ~5
(`calls`/`imports`/`embeds`/`contains`/`implements`) — the other ~4
(`references`/`overrides`/`instantiates`/`returns`/`type_of`) exist in NO
extractor. It also found TS's `explore` is a ~1,700-line multi-stage pipeline
(~15 auxiliary heuristics), not a drop-in RWR reranker. The user chose the
maximum-fidelity path on both forks, so Phase 1 now ALSO reopens the extraction
pipeline + schema and ports the full heuristic stack. This deliberately enlarges
the phase well past its original "shared query engine only" boundary; the
planner should expect many plans / multiple waves and may surface a
`## PHASE SPLIT RECOMMENDED` — but the user explicitly rejected splitting the
edge-kind work out, so keep it in Phase 1 (prefer more waves over a phase split).

**Not in this phase:** `status` content (Phase 2), worktree awareness (Phase 2),
watcher default (Phase 3), any TUI/styling (Phase 6/7), `query`/`search`
relevance ranking (they keep their current lexical matcher — D-05 still holds).

</domain>

<decisions>
## Implementation Decisions

### TS Ground-Truth Capture (how we pin the exact algorithm & oracle)
- **D-01:** Capture TS 1.3.1 ground truth **white-box + black-box**. Read the
  installed TS dist source
  (`/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/`, confirmed
  present, `codegraph --version` → `1.3.1`) to extract the **exact** RWR
  parameters, the 9 edge-kind set + weights, the 5-channel hybrid match scoring,
  the tokenizer/stopword list, the file-relevance-gate constant, the
  node budget (≤16 / 12,000 chars), the generated-file predicate, and the
  verbatim header/warning strings — AND capture live TS `explore`/`node` outputs
  on the fixture corpus as golden files (both CLI and MCP). **Do this while TS
  1.3.1 is still installed** (the existing `testdata/golden/README.md` +
  `capture.sh` already require the live CLI on PATH). The researcher pins every
  numeric constant from the dist source — do NOT guess or hardcode
  reverse-engineered approximations.

### Fixture Equivalence Oracle (how strict "identical" is)
- **D-02:** The parity oracle is **normalized/structural equivalence with an
  explicit, documented allowed-divergence list**, not blanket byte-identity.
  Byte-identical is the target where TS output is deterministic (single-def
  `node`, the multi-def header wording, the "⚠️ no covering tests" warning,
  budget counts). For ranked lists, assert **ordering + set membership + warning
  presence + header text + budget counts** after canonicalizing volatile bits
  (absolute→repo-relative paths, whitespace, float-jitter tie-breaks). Every
  allowed divergence is recorded and justified in the harness. Rationale: a JS
  float RWR and a Go float RWR are not guaranteed bit-identical; pretending
  otherwise would make the harness flaky or force score-fudging. Memory's
  standing lesson applies: template-parity ≠ behavior-parity.

### Fixture Corpus (what we diff against)
- **D-03:** **Reuse the existing pinned golden corpus** (`testdata/golden/corpus/`
  — `colbymchenry-codegraph`, `weft-go`, each with `explore.json`/`node.json`)
  **AND add a small purpose-built synthetic fixture** exercising exactly the
  v0.1 blind spot: (a) deliberately overloaded / same-named symbols → `node`
  multi-def; (b) multi-word queries → explore tokenization; (c) a
  `Test*`-heavy / weakly-connected case → file-relevance gate + "no covering
  tests"; (d) a case where a structurally-connected symbol must outrank a
  lexical name-match → RWR ordering. Every behavioral fixture runs on **BOTH**
  the CLI command and the `codegraph_explore` MCP tool (EXPL-05/NODE-04).

### RWR Determinism & Tie-Breaking
- **D-04:** Ranking MUST be deterministic and reproducible (the golden-corpus
  contract and the fixture harness both depend on it): **fixed 25 iterations**
  (no convergence-threshold early-exit that could vary run-to-run),
  **deterministic node iteration order** for seeding, **stable tie-break =
  score-descending then lexicographic `Id`-ascending** (reuse the existing D-03
  lowest-`Id` convention from `resolveSymbolNode`), and **round scores to a
  fixed precision before comparison** so float jitter never reorders results.

### Scope Boundary (prevent creep into the whole matcher)
- **D-05:** Only **`explore`** switches to the new RWR relevance pipeline.
  `query` and `search` keep their current lexical matcher — out of Phase 1
  scope. `node` gains multi-def enumeration but **reuses** `resolveSymbolNode`'s
  full-scan + lowest-`Id` determinism as the resolution base rather than
  introducing a parallel resolver.
- **D-06:** Fixtures (TEST-01) land **before/with** the algorithm change in the
  same commit series — not a "write tests after" phase. This is the explicit
  guard against re-earning v0.1's blind spot.
- **D-07:** `node`'s generated-files-last sort uses **TS's exact generated-file
  predicate** (to be extracted verbatim from the dist source — likely path/name
  patterns; researcher confirms the precise rule, do not approximate).
- **D-08:** The file-relevance-gate threshold is taken from the **TS dist source
  constant** (research pinned it to `0.06`, but the real gate is a **5-way OR**,
  not a bare threshold — port the full gate, not just the constant).

### Edge-Kind Coverage (user-decided post-research — SCOPE EXPANSION)
- **D-09:** **Expand extraction to all 9 RWR edge kinds.** Research found the
  graph emits only ~5 of TS's `RANK_EDGES` set; the missing ~4
  (`references`/`overrides`/`instantiates`/`returns`/`type_of` — confirm the
  exact TS set + names from 01-RESEARCH.md) are added to the schema AND emitted
  by the language extractors, then the graph is re-indexed, so RWR ranks over
  the full 9-kind set (not a reduced subset). This reopens the v0.1 Phase-5
  extraction pipeline + schema and is the single largest scope item in the
  phase. Planner MUST account for: whether `Edge.kind` is a proto enum (needs a
  bump) or a string; a `SchemaVersion` bump if required; re-indexing this repo's
  own `.codegraph`; **regenerating the golden corpus**; and the **migrate-tool
  impact** (v0.1 Phase-7 `internal/migrate` reads TS SQLite → Pebble — a schema
  change may ripple there). RANK_EDGES is **undirected and UNWEIGHTED** in TS
  (no per-kind weights despite the phase description's phrasing — research
  confirmed). Prioritize the validated-full priority-4 languages
  (Go/Java/C#/Python/TS-JS) for the new edge kinds; mainstream-6 follow the
  existing D-11 full-or-documented-partial capability matrix.
- **D-10:** **Port TS's full auxiliary heuristic stack faithfully** (~15 items:
  the 5-channel hybrid match, type-hierarchy expansion, glue-node injection,
  per-overload seeding/disambiguation tiers, the 5-way relevance gate, the
  5-tier file sort, both tokenizers). This is a faithful port of the ~1,700-line
  TS pipeline, not a minimal RWR. Each heuristic's constants/rules come verbatim
  from the TS dist source (01-RESEARCH.md citations). Where a heuristic depends
  on data Go genuinely cannot produce, document it as an explicit D-02 allowed
  divergence rather than silently dropping it.

### Claude's Discretion
- Package layout within `internal/query` for the RWR pipeline (new file(s) vs
  extending `explore.go`/`traverse.go`), the internal score/rank data types, and
  the harness's normalization helper structure — planner/executor choose, so
  long as the shared-engine + plain-text constraints hold.

### Folded Todos
None folded. (See Reviewed Todos below.)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` § "Phase 1: Behavioral Parity — explore & node" — goal,
  5 success criteria, and the "highest-risk / plain-text-only / fixtures-first" notes.
- `.planning/REQUIREMENTS.md` § EXPL-01..05, NODE-01..04, TEST-01 — the 10
  requirements this phase closes (+ TEST-01 mapping note: harness lands here,
  Phase 8 REL-04 re-runs it as the drop-in gate, does not own it).
- `.planning/PROJECT.md` § Current Milestone + Key Decisions — the drop-in
  parity bar and the "agent/MCP output stays plain" decision.

### TS 1.3.1 reference implementation (source of truth for exact params)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/` — **ABSOLUTE
  external path**; installed TS 1.3.1 (confirmed present, on PATH). Read the
  explore/node/relevance modules to pin RWR α/iters/edge-weights, the 5-channel
  match scoring, tokenizer/stopwords, file-relevance gate constant, node budget,
  generated-file predicate, and verbatim header/warning strings. **Capture now —
  before uninstalling TS.**

### Current implementation (the swap/extension points)
- `internal/query/explore.go` — current lexical `Explore` + `groupMatchesByFile`
  (the maxFiles DoS cap) + `buildBlastEntry`; the lexical `matchNodes` call is
  the swap point for the RWR pipeline.
- `internal/query/node.go` — current single-def `Node` + `resolveNodeForDetail`
  (`-f` disambiguation); gains multi-def enumeration + budget + overflow.
- `internal/query/traverse.go` — `resolveSymbolNode` (D-03 full-scan, lowest-`Id`
  tie-break) reused as node's resolution base; `matchNodes`/lexical matcher.
- `internal/query/render_markdown.go` — `RenderExplore` / `RenderNode` (plain
  string renderers to extend for the warning + multi-def header/overflow).
- `internal/query/engine.go` — the shared `Engine` (single CLI+MCP code path).
- `internal/mcp/tools.go` § `exploreTool`/`exploreHandler` + the `node` case —
  MCP surface; already delegates to `Engine.Explore`/`Engine.Node` (EXPL-05/NODE-04).
- `internal/cli/explore.go`, `internal/cli/node.go` — CLI surface (arg parsing;
  `explore` must accept variadic `<query...>` per EXPL-01).

### Extraction pipeline & schema (D-09 edge-kind expansion — reopened)
- `internal/schema/graph.pb.go` + the `.proto` source — `Edge.kind` definition;
  determines whether new edge kinds need a proto/enum change + `SchemaVersion`
  bump. (Memory: `Edge` provenance fields were pre-reserved, but new KINDS are a
  different change — verify.)
- `internal/indexer/resolve.go` — cross-file resolution (two-pass
  parallel-extract → sequential-resolve); where new edge kinds get resolved.
- `internal/indexer/goextract/` + per-language extractor packages
  (`internal/indexer/pyextract/`, `csharpextract/`, etc.) + the `LanguageSpec`
  registry — the v0.1 Phase-5 pattern the new edge kinds are added into
  (Go/Java/C#/Python/TS-JS first per D-09).
- `internal/indexer/routes/` — the `route`/`implements` synthesized-edge
  precedent (heuristic provenance) to mirror for any synthesized new kinds.
- `internal/migrate/translate.go` — migrate-tool read path; check whether a
  schema change ripples into TS-SQLite→Pebble translation (D-09 migrate impact).

### Behavioral fixture harness (extend, don't rebuild)
- `testdata/golden/README.md` — golden-fixture provenance + capture protocol (D-06).
- `testdata/golden/capture.sh` — the live-TS capture script to extend for the
  new behavioral fixtures (CLI + MCP).
- `testdata/golden/corpus/colbymchenry-codegraph/{explore,node}.json`,
  `testdata/golden/corpus/weft-go/{explore,node}.json` — existing single-symbol
  golden outputs (template-shape baseline to extend with behavioral cases).
- `testdata/golden/golden_test.go`, `testdata/golden/golden_parity_test.go` —
  existing diff harness the new behavioral assertions extend.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BuildReverseAdjacency(reader)` (`internal/query`): reverse-adjacency map used
  by both explore's blast-radius and node's called-by — the caller/coverage data
  source for EXPL-04's "no covering tests" and node's calledBy.
- `isTestSymbol(node)`: existing test-symbol heuristic — reused by the
  file-relevance gate (EXPL-03) and the covering-tests check (EXPL-04).
- `resolveSymbolNode` / `resolveNodeForDetail`: deterministic (lowest-`Id`)
  symbol resolution — node's multi-def enumeration builds on this.
- `readSourceFile` / `resolveSourcePath`: repo-root-confined, symlink-safe
  (WR-03/EvalSymlinks) fresh-from-disk reads — reused for verbatim def bodies.
- `groupMatchesByFile(ranked, maxFiles)`: the maxFiles distinct-file DoS cap
  (T-03-06) — kept; the RWR ranking feeds into it instead of lexical order.
- `staleBanner` / `computeStale`: staleness banner already wired into explore.

### Established Patterns
- **Shared `Engine`, one commit** — CLI and MCP both call `Engine.Explore`/
  `Engine.Node`; behavior parity is structural (mirrors how v0.1 shipped query/MCP).
- **WR-04 dangling-edge tolerance** — skip `ErrNotFound` on edge endpoints rather
  than aborting the whole render; keep this in the RWR traversal.
- **Deterministic lowest-`Id` tie-break** — the codebase-wide convention D-04 extends.
- **Golden-corpus diff harness** (`testdata/golden/` + `capture.sh`) — the
  TEST-01 pattern to extend, not reinvent.
- **Plain-string renderers** — `RenderExplore`/`RenderNode` return plain text;
  keep them ANSI-free (Phase 6 archtest will enforce it later).

### Integration Points
- `matchNodes` (lexical) → replaced by the RWR relevance pipeline **for explore
  only**; `query`/`search` keep calling the lexical matcher.
- `RenderExplore` extended for the per-root "⚠️ no covering tests" warning;
  `RenderNode` extended for the multi-def "N definitions named X — returning M in
  full" header + full bodies + overflow list.
- `exploreHandler` + the MCP `node` case must exercise the exact same `Engine`
  methods (they already do) — the fixture harness asserts this on both surfaces.

</code_context>

<specifics>
## Specific Ideas

- Header/warning strings must match TS **verbatim** — the "⚠️ no covering tests"
  warning (exact emoji + wording) and the "N definitions named X — returning M in
  full" header text are copied from the TS dist source, not paraphrased.
- TS reference is the installed dist at
  `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph`, v1.3.1 (per memory
  `76t84ynav5`); the 6-stage explore algorithm summary from memory `p82eny7gf5`
  (tokenize+stopwords → 5-channel hybrid → type-hierarchy+BFS+glue-node →
  named-seeding → RWR(α=0.25, 25 iter, 9 edge kinds) → file-gate ≥~6% mass) is
  the map — but the researcher confirms every constant against the dist source.

</specifics>

<deferred>
## Deferred Ideas

- `query` / `search` relevance ranking — stays lexical; not in Phase 1 (only
  `explore` gets RWR).
- Any styling / colorized output for explore/node — Phase 6 (rendering seam).
- `status` content, worktree awareness — Phase 2.

### Reviewed Todos (not folded)
- **"Document release procedures (maintainer runbook)"**
  (`2026-07-14-document-release-cut-procedures-runbook.md`, matcher score 0.40) —
  **deferred to Phase 8 (REL).** It is a release-engineering docs task tagged
  `resolves_phase 8`; it has no bearing on explore/node behavioral parity. The
  `--auto` ≥0.4 auto-fold default was overridden by the scope guardrail for this
  borderline docs-vs-algorithm mismatch.

</deferred>

---

*Phase: 1-behavioral-parity-explore-node*
*Context gathered: 2026-07-14*
