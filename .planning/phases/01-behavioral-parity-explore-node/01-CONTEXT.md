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

**Not in this phase:** `status` content (Phase 2), worktree awareness (Phase 2),
watcher default (Phase 3), any TUI/styling (Phase 6/7), `query`/`search`
relevance ranking (they keep their current lexical matcher).

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
- **D-08:** The file-relevance-gate threshold (memory notes "≥ ~6% mass") is
  taken from the **TS dist source constant**, not the remembered approximation.

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
