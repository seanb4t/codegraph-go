# Phase 1: Behavioral Parity — explore & node - Research

**Researched:** 2026-07-14
**Domain:** Reverse-engineering TS CodeGraph 1.3.1's `explore`/`node` ranking algorithm for a faithful Go port
**Confidence:** MEDIUM-HIGH for the extracted constants/predicates (all VERIFIED against the live TS 1.3.1 dist source with file:line citations); LOW-MEDIUM for full-fidelity behavioral parity of the entire heuristic stack (see Summary — this is the phase's central risk)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Capture TS 1.3.1 ground truth **white-box + black-box**. Read the
  installed TS dist source to extract the exact RWR parameters, the 9 edge-kind
  set + weights, the 5-channel hybrid match scoring, the tokenizer/stopword
  list, the file-relevance-gate constant, the node budget (≤16 / 12,000 chars),
  the generated-file predicate, and the verbatim header/warning strings — AND
  capture live TS `explore`/`node` outputs on the fixture corpus as golden
  files (both CLI and MCP). Pin every numeric constant from the dist source —
  do NOT guess or hardcode reverse-engineered approximations.
- **D-02:** The parity oracle is **normalized/structural equivalence with an
  explicit, documented allowed-divergence list**, not blanket byte-identity.
  Byte-identical is the target where TS output is deterministic (single-def
  `node`, the multi-def header wording, the "⚠️ no covering tests" warning,
  budget counts). For ranked lists, assert ordering + set membership + warning
  presence + header text + budget counts after canonicalizing volatile bits.
  Every allowed divergence is recorded and justified in the harness.
- **D-03:** Reuse the existing pinned golden corpus (`testdata/golden/corpus/`
  — `colbymchenry-codegraph`, `weft-go`) AND add a small purpose-built
  synthetic fixture exercising: (a) overloaded/same-named symbols → `node`
  multi-def; (b) multi-word queries → explore tokenization; (c) a
  `Test*`-heavy/weakly-connected case → file-relevance gate + "no covering
  tests"; (d) structurally-connected-beats-lexical ordering. Every behavioral
  fixture runs on BOTH the CLI and MCP surfaces.
- **D-04:** Ranking MUST be deterministic and reproducible: fixed 25
  iterations (no convergence early-exit), deterministic node iteration order
  for seeding, stable tie-break = score-descending then lexicographic
  `Id`-ascending (reuse the existing lowest-`Id` convention from
  `resolveSymbolNode`), and round scores to a fixed precision before
  comparison.
- **D-05:** Only `explore` switches to the new RWR relevance pipeline. `query`
  and `search` keep their current lexical matcher. `node` gains multi-def
  enumeration but reuses `resolveSymbolNode`'s full-scan + lowest-`Id`
  determinism as the resolution base rather than introducing a parallel
  resolver.
- **D-06:** Fixtures (TEST-01) land before/with the algorithm change in the
  same commit series.
- **D-07:** `node`'s generated-files-last sort uses TS's exact generated-file
  predicate (extracted verbatim — see Code Examples).
- **D-08:** The file-relevance-gate threshold ("≥ ~6% mass") is taken from the
  TS dist source constant, not the remembered approximation — **confirmed
  exact: `0.06` (6% of the per-subgraph max aggregated per-file RWR mass)**,
  see Code Examples.

### Claude's Discretion

- Package layout within `internal/query` for the RWR pipeline (new file(s) vs
  extending `explore.go`/`traverse.go`), the internal score/rank data types,
  and the harness's normalization helper structure — planner/executor choose,
  so long as the shared-engine + plain-text constraints hold.

### Deferred Ideas (OUT OF SCOPE)

- `query`/`search` relevance ranking — stays lexical; not in Phase 1 (only
  `explore` gets RWR).
- Any styling/colorized output for explore/node — Phase 6 (rendering seam).
- `status` content, worktree awareness — Phase 2.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXPL-01 | Multi-word `<query...>` tokenized (CamelCase/snake_case/acronym/dot-notation/plain), stopword-filtered | Two distinct TS tokenizers extracted verbatim (`extractSymbolsFromQuery`, `extractSearchTerms` + `STOP_WORDS`) — see Code Examples §1–2. Go CLI currently `cobra.ExactArgs(1)` — confirmed single-arg-only bug. |
| EXPL-02 | Rank by graph relevance (RWR α=0.25, ~25 iters, 9 edge kinds, type-hierarchy+BFS+glue-node expansion) | RWR implementation extracted verbatim (`computeGraphRelevance`, tools.js:2321-2386) — see Code Examples §3. Edge-kind set confirmed exact. **Major gap found: Go schema only has edge-kind analogs for ~4 of TS's 9 RANK_EDGES — see Open Questions §1.** |
| EXPL-03 | File-level relevance gate stops weakly-connected `Test*` funcs from topping results | Gate logic extracted verbatim (tools.js:2763-2783) — see Code Examples §4. Gate is a 5-way OR, not a single threshold — see Open Questions §2. |
| EXPL-04 | Per-root "⚠️ no covering tests" warning | Exact trigger condition + exact string extracted (tools.js:2276-2294) — see Code Examples §5. |
| EXPL-05 | Identical CLI/MCP output (shared engine), verified on fixtures | Existing Go `Engine`/`exploreHandler` already share one code path (internal/query/engine.go, internal/mcp/tools.go) — structural work is in `capture.sh` extension for MCP-surface capture. |
| NODE-01 | Enumerate ALL exact-name defs, generated-files-last | `findSymbolMatches` extracted verbatim (tools.js:4193-4234) + `isGeneratedFile` predicate extracted verbatim (generated-detection.js) — see Code Examples §6–7. |
| NODE-02 | "N definitions named X — returning M in full" header, ≤16 defs/12,000 chars, overflow list | Exact header/budget logic extracted verbatim (tools.js:3633-3676) — see Code Examples §8. **Header is TWO lines, not one combined string — see Open Questions §3.** |
| NODE-03 | Optional file/line narrowing never empties the set | Exact narrowing logic extracted verbatim (tools.js:3603-3620) — see Code Examples §9. Go's existing `resolveNodeForDetail` (`-f` flag) narrows but currently only supports single-def resolution; needs extension to the multi-def path. |
| NODE-04 | Single-def `node` byte-comparable to TS (CLI+MCP) | **Already substantially satisfied**: existing Go `RenderNode` was built byte-for-byte against the golden `node.json` fixture (Trail section, Calls→/Called by← format all match). Confirmed via `testdata/golden/corpus/weft-go/node.json`. |
| TEST-01 | Behavioral fixture harness, CLI+MCP, closes v0.1 blind spot | Existing `capture.sh` always passes `--max-files 1` to explore and `-f <file>` to node — **the existing golden corpus has NEVER exercised multi-file ranking or multi-def enumeration.** Harness extension plan in Architecture Patterns §Fixture Harness. |
</phase_requirements>

## Summary

TS CodeGraph 1.3.1's `explore` is **not** a simple "lexical match → RWR rerank"
pipeline as the phase's shorthand summary suggests. It is a ~1,700-line,
multi-stage heuristic pipeline spanning `context/index.js`'s `ContextBuilder`
(hybrid exact-name + definition-prefix + FTS text search, each independently
scored and boosted, plus type-hierarchy expansion) and `mcp/tools.js`'s
`handleExplore` (glue-node injection, named-symbol seeding with per-name
disambiguation tiers, RWR file-mass scoring, a five-way relevance gate, a
change-surface "buried rescue" pass, and a five-tier file sort). `node`'s
multi-def enumeration (the smaller of the two surfaces) is comparatively
compact and fully extracted below.

**All constants the phase's CONTEXT.md flagged "confirm from source" are now
pinned exactly**, with file:line citations against the live TS 1.3.1 install:
RWR α=0.25, fixed 25 iterations, the 9-member `RANK_EDGES` set (verbatim
names, undirected, **unweighted** — every member contributes equally, there
is no per-kind weight despite the phase description's "+ weights" phrasing),
the file-relevance-gate constant (`0.06`, i.e. 6% of the max per-file RWR
mass — but the real gate is a 5-way OR, not a bare threshold), the node
budget (`HARD_CAP=16`, `BODY_BUDGET=12000`), the generated-file regex list,
and the exact header/warning strings.

**The central planning risk**: byte-identical (or even close structural)
parity with TS's *complete* explore ranking would require porting the whole
~1,700-line heuristic stack — dozens of magic-number boosts (+50/+10/+3/+1
file-score tiers, ×0.3 test dampening, +25 core-directory boost, ×(1+0.5·n)
multi-term boost, +15+brevityBonus prefix-match boost, GLUE_NODE_CAP=60,
MIN_SIBLINGS=3 polymorphic-sibling skeletonization, adaptive output budgets
keyed on project file count) that exist to fix specific reported bugs on
specific real-world repos (comments cite issue numbers: #185, #383, #720,
#1034, #1046, #1064, #1074, #1145, #1185). D-02's "normalized/structural
equivalence, not blanket byte-identity" oracle is what makes this phase
tractable: the plan should port the **core structural signal** (RWR + the
gate's primary graph-mass clause + generated-files-last + the exact budget/
header/warning strings) faithfully, and treat the wider auxiliary-heuristic
stack as an explicitly documented, allowed divergence — not attempt a
line-for-line port. This must be a **discuss-phase or plan-time decision**,
not something the executor discovers mid-implementation.

**Second major risk, structural not algorithmic**: the current Go schema
(shared across all six language extractors — go/java/csharp/python/ts/
mainstream) emits only 4 cross-file edge kinds — `calls`, `imports`,
`embeds`, `contains` — plus one synthesized kind, `implements`. TS's RWR
`RANK_EDGES` set is `calls, references, extends, implements, overrides,
instantiates, returns, type_of, imports` (9 kinds). Direct overlap is only 3
kinds (`calls`, `imports`, `implements`); `embeds` is Go's closest analog to
TS's `extends` but is a **different literal string** and is never promoted to
`extends` (only to `implements`, per the existing embeds→implements promotion
pattern in `resolve.go`). The other 5 TS kinds (`references`, `overrides`,
`instantiates`, `returns`, `type_of`) have **zero data source** anywhere in
the current extraction pipeline. This means the Go RWR graph will be
structurally sparser than TS's — a real behavioral divergence, not a code
bug — and the plan needs an explicit decision on how to represent this (see
Open Questions §1).

**Primary recommendation:** Port the RWR core (α=0.25, 25 fixed iterations,
undirected, unweighted, using whatever edge kinds the Go schema currently
supports — document the 9-vs-N gap as an explicit allowed divergence per
D-02), the file-relevance gate's graph-mass clause (`score ≥ 0.06 × max`) plus
an entry-point/named-symbol keep-clause, EXPL-04's exact trigger + string,
NODE-01/02/03's exact enumeration/budget/header/narrowing logic (these are
fully mechanical, low-risk, and should be ported byte-for-byte), and defer
the remaining ~15 auxiliary lexical-search heuristics (co-location boost,
core-directory boost, multi-term corroboration, buried-rescue, polymorphic-
sibling skeletonization, adaptive output budgets) to a documented "known
divergence" list that the fixture harness explicitly does NOT assert on.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Query tokenization (EXPL-01) | API/Backend (`internal/query`) | — | Pure text processing over the query string before any store access; no client/server split exists in this CLI+MCP-only product. |
| Candidate-symbol gathering (lexical match) | API/Backend (`internal/query`) | Database/Storage (`internal/graphstore`) | `matchNodes`/future hybrid gather is a full-scan read over the Pebble snapshot — logic lives in `query`, storage access via `graphstore.Reader`. |
| RWR graph-relevance ranking (EXPL-02) | API/Backend (`internal/query`) | — | Pure in-memory power-iteration over an already-gathered subgraph; no storage or rendering concerns — belongs in a new `internal/query` file per D-05's discretion. |
| File-relevance gate (EXPL-03) | API/Backend (`internal/query`) | — | Consumes RWR output directly; same tier as the ranking it gates. |
| "No covering tests" warning (EXPL-04) | API/Backend (`internal/query`) | — | Reuses `BuildReverseAdjacency` + a test-file predicate; computed alongside the existing blast-radius entries. |
| Node multi-def enumeration/budget (NODE-01/02) | API/Backend (`internal/query`) | — | Extends existing `resolveNodeForDetail`/`Node` in `internal/query/node.go`. |
| Markdown rendering (both) | API/Backend (`internal/query/render_markdown.go`) | — | Plain-text renderer shared by CLI and MCP — no client tier exists to own this. |
| CLI/MCP surface parity (EXPL-05/NODE-04) | API/Backend (`internal/cli`, `internal/mcp`) | — | Both are thin adapters calling the same `Engine` methods — already structurally correct; only the CLI's `cobra.ExactArgs(1)` → variadic-args flag needs fixing. |
| Fixture harness (TEST-01) | Database/Storage (test fixtures) | API/Backend (`testdata/golden`) | Golden-file capture/diff is a test-infrastructure concern, not runtime code. |

**No browser/frontend/CDN tier exists in this project** — CodeGraph Go is a
CLI + MCP server only; all capabilities above collapse to a single backend
process tier by design.

## Standard Stack

No new external dependencies are required for this phase. RWR/personalized-
PageRank via fixed-iteration power iteration is ~60 lines of plain Go over
`[]float64` slices and adjacency lists (the TS implementation itself is
hand-rolled, ~65 lines, no library) — see **Don't Hand-Roll** below for why
this is the correct call, not a contradiction of that section's spirit.

### Core

No additions to `go.mod`.

### Supporting

No additions to `go.mod`.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled power-iteration RWR | `gonum.org/v1/gonum/graph` + a generic PageRank routine | Gonum's graph package models directed weighted graphs generically and would add a dependency + an adapter layer to bridge `schema.Node`/`schema.Edge` into gonum's graph interfaces, for a computation that's ~40 lines of dense/sparse linear algebra either way at this graph size (subgraph capped at ~200-260 nodes per TS's own `maxNodes: 200` / `GLUE_NODE_CAP: 60`). TS itself hand-rolls it for the same reason. Reject — adds a dependency for no capability gain at this scale. |

**Installation:** N/A — no new packages.

**Version verification:** N/A — no new packages; existing `go.mod` dependencies
(`mark3labs/mcp-go`, `spf13/cobra`, `cockroachdb/pebble`) are unaffected by
this phase's work.

## Package Legitimacy Audit

**Not applicable this phase** — no new external packages are introduced. The
RWR pipeline, tokenizers, and node-budget logic are pure standard-library Go
(`sort`, `strings`, `regexp`, `unicode`) implementing algorithms extracted
directly from the TS 1.3.1 dist source (see Code Examples).

## Architecture Patterns

### System Architecture Diagram — TS 1.3.1's actual explore pipeline

```
                    query string (raw, possibly multi-word)
                              │
                              ▼
              ┌───────────────────────────────┐
              │  extractSymbolsFromQuery()     │  context/index.js:64
              │  (CamelCase/snake/SCREAMING/   │
              │   acronym/dot-notation names)  │
              └───────────────┬───────────────┘
                              │ symbolsFromQuery[]
                              ▼
   ┌─────────────────────────────────────────────────────────┐
   │           ContextBuilder.findRelevantContext()          │  context/index.js:433
   │  ┌─────────────┐ ┌──────────────┐ ┌────────────────────┐│
   │  │Ch.1: exact- │ │Ch.2: titlecase│ │Ch.3: FTS text search││
   │  │name lookup  │ │definition-   │ │ per extractSearch-   ││
   │  │+ co-location│ │prefix search │ │Terms() token,        ││
   │  │boost (+20/  │ │(class/iface/ │ │ multi-term merge      ││
   │  │dup file)    │ │struct/…)     │ │ (+5/extra term match) ││
   │  └──────┬──────┘ └──────┬───────┘ └──────────┬───────────┘│
   │         └───────────────┴────────────────────┘            │
   │                         │ merge, max-score-wins            │
   │                         ▼                                  │
   │   test-file ×0.3 dampen → core-dir +25 boost →              │
   │   multi-term co-occurrence ×(1+0.5n) boost →                │
   │   type-hierarchy expansion (extends/implements,             │
   │   maxNodes/4 budget, 2 passes) → BFS traversal to            │
   │   maxNodes(200)/traversalDepth(3)/minScore(0.2)              │
   └─────────────────────────┬───────────────────────────────┘
                              │ subgraph {nodes, edges, roots}
                              ▼
        ┌───────────────────────────────────────────────┐
        │        handleExplore() — mcp/tools.js:2398     │
        │ 1. glue-node injection (callers/callees of      │
        │    roots, same-file-only, cap 60)                │
        │ 2. named-symbol seeding: tokenize again,          │
        │    resolve EVERY exact-name def, disambiguate      │
        │    overloads (≤3 defs: all; >3: type-token-corrob-  │
        │    orated ≤4, else top-1 by centrality)              │
        │ 3. per-file score: named-seed=+50, entry=+10,        │
        │    connected=+3, other=+1; filter score≥3             │
        │ 4. hard-exclude test/spec files (unless query          │
        │    mentions "test")                                    │
        │ 5. computeGraphRelevance() — RWR, α=0.25, 25 iters,     │
        │    9-kind undirected unweighted edge set → per-node     │
        │    mass → aggregated per-file mass                      │
        │ 6. relevance GATE: keep file if (mass≥0.06×max) OR       │
        │    central OR entry OR change-surface-rescued OR         │
        │    (≥2 distinct query-term hits)                          │
        │ 7. 5-tier sort: named-seed-file > corroborated            │
        │    (entry/central + ≥2 terms) > graph-mass (1% epsilon)   │
        │    > term-hits > !low-value > !generated > score > count  │
        └─────────────────────────┬───────────────────────────────┘
                                  ▼
                     render: header, blast-radius,
                  verbatim-source-per-file (adaptive budget)
```

### Recommended Project Structure

```
internal/query/
├── explore.go        # Explore() orchestrator — extend, don't replace: keeps
│                      # groupMatchesByFile/buildBlastEntry/readSourceFile
├── rwr.go             # NEW (D-05 discretion): computeGraphRelevance port —
│                      # pure func(nodeIDs []string, edges []*schema.Edge,
│                      # seedIDs map[string]bool) map[string]float64
├── explore_gate.go    # NEW (D-05 discretion): file-relevance gate + sort
│                      # tiers, isolated for unit-testability against fixed
│                      # synthetic subgraphs (no store dependency)
├── tokenize.go        # NEW: port of extractSymbolsFromQuery + extractSearch-
│                      # Terms + STOP_WORDS (EXPL-01) — two distinct funcs,
│                      # do not conflate them (see Code Examples §1-2)
├── node.go            # extend: multi-def enumeration (findSymbolMatches-
│                      # equivalent), HARD_CAP/BODY_BUDGET, header lines
├── traverse.go         # unchanged base (resolveSymbolNode reused per D-05)
└── render_markdown.go  # extend: multi-def header/overflow, "no covering
                         # tests" warning line
```

### Pattern 1: RWR power iteration (EXPL-02 core)
**What:** Fixed-iteration personalized PageRank over an undirected, unweighted
adjacency built from a whitelisted edge-kind set, restarting to a uniform
distribution over seed nodes present in the candidate set.
**When to use:** Only inside `explore`'s ranking step — not `query`/`search`
(D-05).
**Example (verbatim TS source, port target):** see Code Examples §3.

### Pattern 2: Generated-files-last stable sort (NODE-01/D-07)
**What:** A single `sort.SliceStable` keyed only on a path-regex predicate —
NOT combined with any other tie-break in TS. The original relative order
(from SQLite's un-ordered `SELECT * FROM nodes WHERE name = ?`, effectively
insertion/rowid order) is preserved for non-generated vs. non-generated ties.
**When to use:** `node`'s multi-def enumeration only.
**Divergence note:** Go's Pebble store has no equivalent implicit "row order"
— the existing codebase convention (D-04, `resolveSymbolNode`) is lowest-`Id`
tie-break. The plan should apply `isGeneratedFile` as the PRIMARY sort key
(exact port) and lowest-`Id` as the secondary tie-break (documented, sourced
divergence — TS's own primary-key order is non-deterministic across
re-indexes anyway, so this is a reasonable and honest substitution, not a
loss of fidelity).

### Anti-Patterns to Avoid
- **Treating RWR edge weights as configurable per-kind:** TS's `RANK_EDGES` is
  a plain `Set` — presence/absence only, no weight map. Do not invent
  per-kind weights; every edge in the 9-kind set contributes identically.
- **Conflating the two tokenizers:** `extractSymbolsFromQuery` (exact-name
  identifier extraction: CamelCase/snake/SCREAMING/acronym/dot-notation,
  minimum-length + `commonWords` filtering) and `extractSearchTerms` (FTS
  term extraction: splits compounds into sub-words, adds stem variants,
  filters via the SEPARATE `STOP_WORDS` set) serve different purposes and
  have different filter lists. EXPL-01's phrasing ("tokenized ...,
  stopword-filtered") maps most directly to `extractSearchTerms` +
  `STOP_WORDS`, but `explore`'s named-symbol seeding (Pattern giving files
  their `+50` score) depends on `extractSymbolsFromQuery`'s identifier
  extraction. Both matter; do not port only one and assume it covers EXPL-01.
- **Assuming the gate is a bare threshold:** EXPL-03's file-relevance gate is
  a 5-way boolean OR (graph-mass ≥ 6%, OR central, OR entry/named file, OR
  change-surface-rescued, OR ≥2 distinct term hits), guarded so it never
  prunes below 2 files. A single-clause port will under-select files TS
  would keep (e.g., a named-by-agent file with near-zero RWR mass).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|--------------|
| Power-iteration RWR at this graph scale (≤~260 nodes) | A generic sparse-matrix library (gonum) | Plain `[]float64` + adjacency-list power iteration — this is what TS itself does; a library adds an adapter-layer dependency for no capability the hand-rolled version lacks at this scale. |
| Camel/snake/acronym/dot-notation tokenization | A general-purpose NLP tokenizer package | Port the two TS regex-based extractors verbatim (Code Examples §1-2) — they are 3rd-party-independent regex logic already proven against TS's own test corpus; a general NLP library would diverge from TS's specific splitting rules (e.g. the acronym-then-titlecase boundary rule) and defeat the parity goal. |

**Key insight:** This phase's goal is *reproducing a specific existing
algorithm's exact behavior*, not solving the general problem well — so the
"don't hand-roll" instinct (reach for a library that solves the general
problem better) is actively counterproductive here. A library's own opinions
about tokenization/ranking will diverge from TS's idiosyncratic rules
(several of which exist to fix specific reported bugs, not to be "correct"
in the abstract), producing worse parity, not better code.

## Common Pitfalls

### Pitfall 1: Assuming EXPL-02 is achievable as "lexical match, then RWR rerank"
**What goes wrong:** A plan that treats RWR as a drop-in reranking step over
the EXISTING `matchNodes` lexical output will miss that TS's `subgraph` going
into RWR is itself built from a completely different, much larger gather
(hybrid exact/definition-prefix/FTS + type-hierarchy expansion + glue nodes +
named seeding) — not the plain substring/prefix match `matchNodes` performs
today.
**Why it happens:** The phase's shorthand ("RWR reranks lexical matches") is
an oversimplification convenient for the roadmap-level description.
**How to avoid:** Scope EXPL-01/02's tasks around the ACTUAL TS pipeline
stages documented in Architecture Patterns above, explicitly deciding (with
user sign-off, since D-02 already grants latitude) which auxiliary stages are
in-scope-for-fidelity vs. documented-divergence.
**Warning signs:** A plan whose EXPL-02 task list has 1-2 items ("add RWR
scoring function", "wire it into Explore") instead of ~6-8 (tokenizer,
hybrid gather, glue nodes, named seeding, RWR, gate, sort tiers).

### Pitfall 2: Edge-kind gap silently degrading RWR quality
**What goes wrong:** Porting `computeGraphRelevance` verbatim but feeding it
Go's actual edge kinds (`calls`, `imports`, `implements`, `embeds`,
`contains`) without adjustment will silently produce a sparser, less
accurate ranking than TS's — especially on OOP-heavy codebases where
`extends`/`overrides`/`type_of`/`returns` carry real structural signal TS's
5 unavailable kinds provide and Go's schema doesn't.
**Why it happens:** The RWR *algorithm* is fully portable; the *data* it
consumes is a fixed function of what each language extractor already emits,
which was designed for other purposes (Phase 5's route detection, dispatch
resolution) before this phase's RWR need existed.
**How to avoid:** Document this gap explicitly in the plan and the fixture
harness's allowed-divergence list (D-02). Consider whether `embeds` should be
added to the Go-side RANK_EDGES analog (structurally it's the closest match
to `extends`) even though it's a different literal name than TS's set.
**Warning signs:** A plan that lists `RANK_EDGES` as "the same 9 kinds as
TS" without cross-checking against `internal/indexer/goextract/types.go`.

### Pitfall 3: Under-capturing new fixtures with the existing `capture.sh` defaults
**What goes wrong:** `capture.sh`'s existing `explore` call hardcodes
`--max-files 1` and its `node` call hardcodes `-f <file>` (single-def mode)
— re-running it unchanged captures NOTHING new about multi-file ranking or
multi-def enumeration, the exact two behaviors this phase adds.
**Why it happens:** `capture.sh` was written in an earlier phase for
template-shape parity only (single symbol, single file) — its existing
invocations are deliberately narrow.
**How to avoid:** Add NEW `capture_repo`-style invocations (or a new function)
for: multi-word explore queries without `--max-files 1`; `node <name>`
WITHOUT `-f` on a name known to have 2+ defs; an MCP-surface capture path
(the existing script only exercises the TS CLI, never TS's MCP server).
**Warning signs:** A TEST-01 plan that only re-runs `capture.sh` unchanged.

### Pitfall 4: Two-line node header rendered as one string
**What goes wrong:** TS's multi-def output is `header` (`**N definitions
named "X"**`) followed by a SEPARATE line (`Returning M in full[; K more
listed below] — pick the one you need (no Read required).`), joined by a
blank line — not one combined sentence as the phase description's shorthand
("N definitions named X — returning M in full") might suggest.
**Why it happens:** The phase Notes paraphrase/compress the actual two-line
format for brevity.
**How to avoid:** Port the exact two-line structure — see Code Examples §8.
**Warning signs:** A `RenderNode` extension producing a single line with an
em-dash joining "N definitions" and "returning M".

## Code Examples

Verified patterns extracted directly from the live TS CodeGraph 1.3.1
install at
`/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/`
(the actual platform bundle the npm shim execs — NOT the `dist/*.d.ts`-only
top-level package, which ships no `.js`). All citations `[VERIFIED: TS 1.3.1
dist]`.

### §1. Exact-symbol tokenizer (feeds named-symbol seeding)
```javascript
// Source: context/index.js:64-145 [VERIFIED: TS 1.3.1 dist]
function extractSymbolsFromQuery(query) {
    const symbols = new Set();
    // CamelCase: 2+ chars, starts with letter
    const camelCasePattern = /\b([A-Z][a-z]+(?:[A-Z][a-z]*)*|[a-z]+(?:[A-Z][a-z]*)+)\b/g;
    // snake_case: 3+ chars
    const snakeCasePattern = /\b([a-z][a-z0-9]*(?:_[a-z0-9]+)+)\b/gi;
    // SCREAMING_SNAKE_CASE
    const screamingPattern = /\b([A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+)\b/g;
    // ALL_CAPS acronyms, 2+ chars (REST, HTTP, LRU, API)
    const acronymPattern = /\b([A-Z]{2,})\b/g;
    // dot.notation — extracts both the full path AND each part
    const dotPattern = /\b([a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)+)\b/g;
    // plain lowercase identifiers, 3+ chars
    const lowercasePattern = /\b([a-z][a-z0-9]{2,})\b/g;
    // ... (all patterns run in sequence, results unioned into `symbols`)
    // Then filtered against a ~90-word commonWords Set (see context/index.js:118-143)
    return Array.from(symbols).filter(s => !commonWords.has(s.toLowerCase()));
}
```

### §2. FTS-term tokenizer + stopwords (EXPL-01's literal target)
```javascript
// Source: search/query-utils.js:102-120, 189-242 [VERIFIED: TS 1.3.1 dist]
exports.STOP_WORDS = new Set([
    'the', 'a', 'an', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for',
    'of', 'with', 'by', 'from', 'is', 'it', 'that', 'this', 'are', 'was',
    'be', 'has', 'had', 'have', 'do', 'does', 'did', 'will', 'would', 'could',
    'should', 'may', 'might', 'can', 'shall', 'not', 'no', 'all', 'each',
    'every', 'how', 'what', 'where', 'when', 'who', 'which', 'why',
    'i', 'me', 'my', 'we', 'our', 'you', 'your', 'he', 'she', 'they',
    'show', 'give', 'tell', 'been', 'done', 'made', 'used', 'using', 'work',
    'works', 'found', 'also', 'into', 'then', 'than', 'just', 'more', 'some',
    'such', 'over', 'only', 'out', 'its', 'so', 'up', 'as', 'if', 'look',
    'need', 'needs', 'want', 'happen', 'happens', 'affect', 'affected',
    'break', 'breaks', 'failing', 'implemented', 'implement',
    'code', 'file', 'files', 'function', 'method', 'class', 'type',
    'fix', 'bug', 'called',
]);

function extractSearchTerms(query, options) {
    const includeStems = options?.stems !== false;
    const tokens = new Set();
    // 1. Preserve compound identifiers: camelCase/PascalCase (>=3 chars, lowercased)
    // 2. Preserve snake_case compounds (>=3 chars, lowercased)
    // 3. Split camelCase/PascalCase into words: "getUserName" -> "get User Name"
    const camelSplit = query
        .replace(/([a-z])([A-Z])/g, '$1 $2')
        .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2');
    // 4. underscores/dots -> spaces
    const normalised = camelSplit.replace(/[_.]+/g, ' ');
    // 5. split on non-alphanumeric, lowercase, filter len<3 and STOP_WORDS
    const words = normalised.split(/[^a-zA-Z0-9]+/).filter(Boolean);
    for (const word of words) {
        const lower = word.toLowerCase();
        if (lower.length < 3 || exports.STOP_WORDS.has(lower)) continue;
        tokens.add(lower);
    }
    // 6. (optional) add stem variants via getStemVariants() for FTS prefix matching
    return [...tokens];
}
```

### §3. RWR core (EXPL-02's load-bearing algorithm)
```javascript
// Source: mcp/tools.js:2321-2386 [VERIFIED: TS 1.3.1 dist]
// The 9-member edge-kind set — presence/absence only, NO per-kind weights.
const RANK_EDGES = new Set([
    'calls', 'references', 'extends', 'implements', 'overrides',
    'instantiates', 'returns', 'type_of', 'imports',
]);

computeGraphRelevance(nodeIds, edges, seedIds) {
    const out = new Map();
    const n = nodeIds.length;
    if (n === 0) return out;
    const idx = new Map();
    for (let i = 0; i < n; i++) idx.set(nodeIds[i], i);

    const adj = Array.from({ length: n }, () => []);
    for (const e of edges) {
        if (!RANK_EDGES.has(e.kind)) continue;
        const i = idx.get(e.source), j = idx.get(e.target);
        if (i === undefined || j === undefined || i === j) continue;
        adj[i].push(j);
        adj[j].push(i); // undirected — reachable either direction
    }

    // Restart vector: uniform over seeds present in the candidate set.
    // Falls back to uniform-over-all if no seed landed in the set.
    const r = new Array(n).fill(0);
    let rsum = 0;
    for (const id of seedIds) {
        const i = idx.get(id);
        if (i !== undefined) { r[i] = 1; rsum += 1; }
    }
    if (rsum === 0) { for (let i = 0; i < n; i++) r[i] = 1; rsum = n; }
    for (let i = 0; i < n; i++) r[i] /= rsum;

    const alpha = 0.25;
    let s = r.slice();
    for (let iter = 0; iter < 25; iter++) {          // FIXED 25 iterations, no early-exit
        const next = new Array(n).fill(0);
        for (let i = 0; i < n; i++) {
            const si = s[i];
            if (si === 0) continue;
            const d = adj[i].length;
            if (d === 0) { next[i] += si; continue; }  // dangling: keep its mass
            const share = si / d;
            for (const j of adj[i]) next[j] += share;
        }
        for (let i = 0; i < n; i++) s[i] = (1 - alpha) * next[i] + alpha * r[i];
    }
    for (let i = 0; i < n; i++) out.set(nodeIds[i], s[i]);
    return out;
}
```

### §4. File-relevance gate (EXPL-03) — a 5-way OR, not a threshold
```javascript
// Source: mcp/tools.js:2763-2783 [VERIFIED: TS 1.3.1 dist]
// Aggregated per-file: fileGraphScore.get(fp) = sum of node-level RWR mass
// for every node in that file. maxGraph = max fileGraphScore across all files.
if (maxGraph > 0) {
    const gated = relevantFiles.filter(([fp]) =>
        (fileGraphScore.get(fp) ?? 0) >= maxGraph * 0.06     // (1) graph mass ≥ 6% of max
        || centralFiles.has(fp)                               // (2) graph-central + textually matched
        || entryFiles.has(fp)                                 // (3) defines an agent-named symbol
        || changeSurfaceFiles.has(fp)                         // (4) buried-rescue: named method's
                                                                //     signature type, near-zero mass
        || (fileTermHits.get(fp) ?? 0) >= 2                   // (5) ≥2 distinct query terms matched
    );
    if (gated.length >= 2) relevantFiles = gated;             // never prunes below 2 files
}
```

### §5. "No covering tests" warning (EXPL-04) — exact trigger + exact string
```javascript
// Source: mcp/tools.js:2276-2294 [VERIFIED: TS 1.3.1 dist]
// (inside buildBlastRadiusSection, per subgraph root)
const callerFiles = [...new Set(uniq.map((n) => rel(n.filePath)))];
const testFiles = callerFiles.filter((f) => isTestFile(f));
const nonTest = callerFiles.filter((f) => !isTestFile(f));
// ... "shown"/"more" formatting for nonTest ...
const tests = testFiles.length > 0
    ? `; tests: ${testFiles.slice(0, FILE_CAP).map(f => `\`${f}\``).join(', ')}${testFiles.length > FILE_CAP ? ` +${testFiles.length - FILE_CAP}` : ''}`
    : '; ⚠️ no covering tests found';           // <-- EXACT string, note "found", trailing
entries.push(`- \`${root.name}\` (${rel(root.filePath)}:${root.startLine}) — ${uniq.length} caller${uniq.length === 1 ? '' : 's'}${where}${tests}`);
// Trigger: uniq.length > 0 (has ≥1 direct caller) is checked BEFORE this block
// (an early `continue` skips roots with zero callers entirely — no warning for
// a root with no blast radius at all, only for one WITH callers but no tests).
```
`isTestFile` (the file-PATH predicate used here — NOT a symbol-name
heuristic like Go's existing `isTestSymbol`) is defined in
`search/query-utils.js:300-332` [VERIFIED: TS 1.3.1 dist]: matches
`test_*`/`*_test.*`/`*.test.*`/`*-spec.*`/`*Test.ext`/`*Tests.ext`/
`*TestCase.ext`/`*Spec.ext` filenames, plus `/tests?/`, `/__tests__/`,
`/specs?/`, `/testlib/`, `/testing/`, and CamelCase `*Test*/`
Gradle/Kotlin-style directories, plus non-production dirs (`integration`,
`sample(s)`, `example(s)`, `fixture(s)`, `benchmark(s)`, `demo(s)`).

### §6. Multi-def enumeration + generated-files-last (NODE-01)
```javascript
// Source: mcp/tools.js:4193-4234 [VERIFIED: TS 1.3.1 dist]
findSymbolMatches(cg, symbol) {
    const isQualified = /[.\/]|::/.test(symbol);
    if (!isQualified) {
        const exact = cg.getNodesByName(symbol);   // SELECT * FROM nodes WHERE name = ? (no ORDER BY)
        if (exact.length > 0) {
            return [...exact].sort((a, b) =>
                (isGeneratedFile(a.filePath) ? 1 : 0) - (isGeneratedFile(b.filePath) ? 1 : 0));
            // ^ STABLE sort, generated-file-only key — no secondary tie-break in TS.
        }
        const fuzzy = cg.searchNodes(symbol, { limit: 10 });
        return fuzzy[0] ? [fuzzy[0].node] : [];
    }
    // ... qualified-name path via FTS + matchesSymbol filter, same generated-last sort ...
}
```

### §7. Generated-file predicate (D-07) — verbatim, complete
```javascript
// Source: extraction/generated-detection.js:27-82 [VERIFIED: TS 1.3.1 dist]
const GENERATED_PATTERNS = [
    /\.pb\.go$/, /\.pulsar\.go$/, /_grpc\.pb\.go$/,
    /_mock\.go$/, /_mocks\.go$/, /^mock_[^/]+\.go$/,
    /\.generated\.[jt]sx?$/, /\.gen\.[jt]sx?$/, /\.pb\.[jt]s$/, /_pb\.[jt]s$/, /_grpc_pb\.[jt]s$/,
    /\.min\.m?js$/,
    /_pb2(_grpc)?\.py$/, /_pb2\.pyi$/,
    /\.pb\.(cc|h)$/,
    /\.g\.cs$/, /Grpc\.cs$/,
    /OuterClass\.java$/, /Grpc\.java$/,
    /\.pb\.swift$/,
    /\.g\.dart$/, /\.freezed\.dart$/, /\.pb\.dart$/, /\.pbgrpc\.dart$/, /\.chopper\.dart$/,
    /\.generated\.rs$/,
];
function isGeneratedFile(filePath) {
    return GENERATED_PATTERNS.some((p) => p.test(filePath));
}
```

### §8. Multi-def header/budget/overflow (NODE-02) — two-line header, exact strings
```javascript
// Source: mcp/tools.js:3633-3676 [VERIFIED: TS 1.3.1 dist]
const header = `**${matches.length} definitions named "${symbol}"**`;
if (!includeCode) {
    const list = matches.map((n) => `- \`${n.name}\` (${n.kind}) — ${n.filePath}:${n.startLine}`);
    return [header, '', 'Re-query with `includeCode: true` to get every body in one call — no need to pick one first.', '', ...list].join('\n');
}
const BODY_BUDGET = 12000;   // char budget, leaves room under 15000 MAX_OUTPUT_LENGTH
const HARD_CAP = 16;         // max FULL bodies rendered
const rendered = [], listed = [];
let used = 0;
for (const n of matches) {
    if (rendered.length >= HARD_CAP) { listed.push(n); continue; }
    const section = renderNodeSection(cg, n, true);
    if (rendered.length === 0 || used + section.length <= BODY_BUDGET) {
        rendered.push(section); used += section.length;
    } else { listed.push(n); }
}
const out = [
    header,
    // SEPARATE second line, joined to header by a blank line — NOT one sentence:
    `Returning ${rendered.length} in full${listed.length ? `; ${listed.length} more listed below` : ''} — pick the one you need (no Read required).`,
    '',
    rendered.join('\n\n---\n\n'),
];
if (listed.length) {
    const LIST_CAP = 20;
    out.push('', '**Other definitions**', ...listed.slice(0, LIST_CAP).map(n => `- \`${n.name}\` (${n.kind}) — ${n.filePath}:${n.startLine}`));
    if (listed.length > LIST_CAP) out.push(`- … +${listed.length - LIST_CAP} more`);
    out.push('', `> Need one of these in full? Call codegraph_node again with \`file\` (e.g. \`"${listed[0].filePath.split('/').pop()}"\`) or \`line\` — do NOT Read it.`);
}
```

### §9. File/line narrowing that never empties the set (NODE-03)
```javascript
// Source: mcp/tools.js:3603-3620 [VERIFIED: TS 1.3.1 dist]
if (matches.length > 1 && (fileHint || lineHint !== undefined)) {
    const norm = (p) => p.replace(/\\/g, '/').toLowerCase();
    let narrowed = matches;
    if (fileHint) {
        const fh = norm(fileHint);
        const byFile = narrowed.filter((n) => norm(n.filePath).endsWith(fh) || norm(n.filePath).includes(fh));
        if (byFile.length > 0) narrowed = byFile;      // <-- only replaces if non-empty
    }
    if (lineHint !== undefined && narrowed.length > 1) {
        const containing = narrowed.filter((n) => n.startLine <= lineHint && (n.endLine ?? n.startLine) >= lineHint);
        narrowed = containing.length > 0
            ? containing
            : [...narrowed].sort((a, b) => Math.abs(a.startLine - lineHint) - Math.abs(b.startLine - lineHint)).slice(0, 1);
        // ^ no containing def: falls back to NEAREST by startLine distance, never empty
    }
    if (narrowed.length > 0) matches = narrowed;        // <-- final guard: never assign empty
}
```

### §10. Current Go `Engine`/CLI extension points (confirmed exact bug + shape)
```go
// Source: internal/cli/explore.go:24 [VERIFIED: repo]
Args:  cobra.ExactArgs(1),   // BUG for EXPL-01: rejects >1 arg; multi-word queries need quoting today.
                              // Fix: cobra.MinimumNArgs(1) + strings.Join(args, " ") before Engine.Explore.

// Source: internal/query/search.go:75 [VERIFIED: repo] — the swap point for EXPL-02
func (e *Engine) matchNodes(term, kind string) ([]rankedNode, error) { /* plain lexical scan */ }
// Explore() (explore.go:114) calls e.matchNodes(query, "") directly — this call site is
// where the new hybrid-gather + RWR pipeline plugs in, NOT a rerank of matchNodes' output.
```

### §11. Existing Go/TS edge-kind inventory (the parity gap, exact)
```go
// Source: internal/indexer/goextract/types.go:32-50 [VERIFIED: repo]
// Shared by ALL extractors (go/java/csharp/python/ts/mainstream) — see
// languages_*.go doc comments, each explicitly "mirrors goextract's shape":
const (
    RefKindCalls    = "calls"
    RefKindImports  = "imports"
    RefKindEmbeds   = "embeds"     // struct/class inheritance AND interface impl,
    RefKindContains = "contains"    // undistinguished until resolve.go's promotion pass
)
const EdgeKindImplements = "implements"  // synthesized: embeds→implements promotion
                                          // (target is an interface) OR structural
                                          // method-set match (Go only)
// embeds NEVER promotes to a distinct "extends" edge kind — it stays "embeds"
// when the target is a class/struct, by design (RESEARCH Pattern 2, cited in
// every extractor's doc comment).
```
```
TS RANK_EDGES (9):  calls, references, extends, implements, overrides, instantiates, returns, type_of, imports
Go equivalents (5):  calls, imports,    embeds*,  implements, —,        —,           —,      —,       imports
                                        (*different literal string, not auto-promoted to "extends")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| N/A — this is a reverse-engineering research task against a fixed, pinned upstream version (TS 1.3.1), not a moving external ecosystem. No "current best practice" drift applies. | | | |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Full-fidelity, byte-identical parity with TS's entire ~1,700-line explore heuristic stack is out of scope for this phase; only the core structural signal (RWR + primary gate clause + budgets/strings) should be ported, with the rest treated as a documented divergence per D-02. | Summary, Common Pitfalls §1 | If wrong (user actually wants the FULL stack ported), the phase's effort/task-count estimate is off by roughly 5-8x — this is a scope decision the plan (or a discuss-phase follow-up) must make explicit and get user sign-off on, not something the executor should discover mid-task. |
| A2 | The Go schema's `embeds` edge kind is the closest structural analog to TS's `extends`, and it is reasonable (though a documented divergence) to include it in Go's RANK_EDGES-equivalent set even though TS never emits an `extends` literal from `embeds`. | Open Questions §1, Code Examples §11 | If the planner instead excludes `embeds` entirely (treating only the 3 exact-name-match kinds `calls`/`imports`/`implements` as portable), Go's RWR graph will be even sparser than assumed here, and class/struct-inheritance-based relevance (a common "structurally related" case on Go's own struct-embedding idiom) will be invisible to the ranker. |
| A3 | SQLite's un-ordered `SELECT * FROM nodes WHERE name = ?` (TS's `getNodesByName`) returns rows in effectively-stable-but-implementation-defined (rowid/insertion) order, so TS's own multi-def tie-break (before the generated-files-last stable sort) has no meaningful semantic the Go port needs to replicate — substituting the codebase's existing lowest-`Id` convention (D-04) as the secondary tie-break is a safe, honest divergence. | Architecture Patterns, Pattern 2 | If TS's row order is actually deterministic and semantically meaningful (e.g., correlates with declaration order in a way agents rely on), a lowest-Id substitution could produce a different "first-listed" def than TS on ties — low risk since D-02 already treats ordering ties as an allowed-divergence category for non-primary sort keys. |

## Open Questions

1. **Edge-kind gap: how should EXPL-02's RANK_EDGES-equivalent set be defined
   given Go's schema only supports ~4-5 of TS's 9 kinds?**
   - What we know: TS's set is `calls, references, extends, implements,
     overrides, instantiates, returns, type_of, imports` (9, exact, verified).
     Go's schema currently emits `calls, imports, embeds, contains,
     implements` (5, exact, verified) — no extractor across all 6 languages
     produces `references`/`overrides`/`instantiates`/`returns`/`type_of`
     edges at all.
   - What's unclear: Whether adding these 5 missing edge kinds to the
     extraction pipeline is in scope for this phase (it would touch every
     language extractor — a much larger surface than "port an algorithm"),
     or whether the phase should ship with a documented, narrower edge set
     and treat the resulting ranking-quality gap as an accepted v1.0
     divergence.
   - Recommendation: Treat as out-of-scope for Phase 1 (adding extraction for
     5 new edge kinds across 6 language extractors is itself phase-sized
     work). Use `calls, imports, implements, embeds, contains` as the Go
     RANK_EDGES-equivalent set (5 kinds — `contains` added because, unlike
     TS, Go's schema uses it for file→symbol AND type→method structural
     containment, which is real connectivity signal a 4-kind set would
     discard). Document this explicitly as an allowed divergence in the
     fixture harness (D-02) and flag it to the user for confirmation before
     planning locks it in — this changes the RWR's practical ranking
     accuracy versus TS.

2. **How much of the ~15-item auxiliary heuristic stack (co-location boost,
   core-directory boost, multi-term corroboration, buried-rescue,
   polymorphic-sibling skeletonization, adaptive per-project-size output
   budgets, etc.) should the plan attempt to port for EXPL-02/EXPL-03?**
   - What we know: Each heuristic exists to fix a specific reported bug on a
     specific real-world repo (issue numbers cited in TS source comments);
     none of them are load-bearing for the phase's stated success criteria
     ("graph relevance beats lexical", "weakly-connected symbols don't top
     results", "no covering tests warning") in isolation — the core RWR +
     primary gate clause + generated-files-last + exact strings ARE
     load-bearing.
   - What's unclear: Whether the synthetic fixtures (D-03's "structural
     beats lexical" case) can be constructed to pass with ONLY the core
     pipeline, or whether some auxiliary heuristic (e.g. the test-file ×0.3
     dampening, or the multi-term corroboration tier) is actually necessary
     to produce TS-matching *ordering* on the specific fixtures chosen.
   - Recommendation: Build the synthetic fixtures FIRST (per D-06, fixtures
     land with/before the algorithm) using only the core pipeline's expected
     behavior, and add auxiliary heuristics only if a specific fixture case
     genuinely requires one to pass. This keeps the port bounded by the
     phase's actual test surface rather than TS's full feature set.

3. **Does EXPL-01's "tokenized ..., stopword-filtered" requirement map to
   `extractSearchTerms`+`STOP_WORDS`, `extractSymbolsFromQuery`, or both?**
   - What we know: Both functions exist and both feed into `explore`'s
     pipeline for different purposes (Code Examples §1-2); EXPL-01's
     requirement text lists exactly the four identifier-shape categories
     `extractSymbolsFromQuery` recognizes (CamelCase/snake_case/acronym/
     dot-notation) but uses the word "stopword-filtered", which is
     `extractSearchTerms`'s vocabulary (STOP_WORDS), not
     `extractSymbolsFromQuery`'s (`commonWords`, a different, larger list).
   - What's unclear: Whether the phase's success-criteria author intended
     one unified tokenizer or was summarizing both TS functions loosely.
   - Recommendation: Port both (they serve genuinely different roles in the
     pipeline — named-symbol seeding needs `extractSymbolsFromQuery`'s exact
     identifier extraction; the FTS-equivalent gather needs
     `extractSearchTerms`'s stopword-filtered term list) rather than picking
     one, since EXPL-02's named-seeding (`+50` file score) and EXPL-01's
     literal "stopword-filtered" wording each require a different one of the
     two.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) + `testdata/golden/` fixture-diff harness |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/query/... ./testdata/golden/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXPL-01 | Multi-word query tokenizes, doesn't 0-match | unit | `go test ./internal/query/... -run TestTokenize -v` | ❌ Wave 0 (new tokenize.go + test) |
| EXPL-02 | RWR ranks structurally-connected symbol above lexical-only match | unit + golden | `go test ./internal/query/... -run TestComputeGraphRelevance -v` + golden fixture diff | ❌ Wave 0 (new rwr.go + test; extend `capture.sh` for the fixture) |
| EXPL-03 | Weakly-connected `Test*` func doesn't top results | golden | fixture diff against synthetic D-03(c) corpus | ❌ Wave 0 (new synthetic fixture) |
| EXPL-04 | "no covering tests" warning fires/doesn't fire correctly | unit | `go test ./internal/query/... -run TestNoCoveringTestsWarning -v` | ❌ Wave 0 |
| EXPL-05 | CLI/MCP byte-identical explore output | golden parity | `go test ./testdata/golden/... -run TestGoldenParity -v` (extend for MCP surface) | ✅ `golden_parity_test.go` exists — extend, don't replace |
| NODE-01/02 | Multi-def enumeration, header, budget, overflow | unit + golden | `go test ./internal/query/... -run TestNodeMultiDef -v` | ❌ Wave 0 (new synthetic overloaded-symbol fixture) |
| NODE-03 | File/line narrowing never empties set | unit | `go test ./internal/query/... -run TestNarrowNeverEmpty -v` | ❌ Wave 0 |
| NODE-04 | Single-def byte-comparable | golden | `go test ./testdata/golden/... -run TestGoldenFixturesExist -v` | ✅ already covered by existing `node.json` fixture + `RenderNode` |
| TEST-01 | Harness itself: CLI+MCP, closes v0.1 blind spot | integration | `go test ./testdata/golden/... -v` | ✅ scaffolding exists (`golden_test.go`, `golden_parity_test.go`) — extend `capture.sh` + add synthetic corpus per D-03 |

### Sampling Rate
- **Per task commit:** `go test ./internal/query/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/query/tokenize.go` + `tokenize_test.go` — EXPL-01
- [ ] `internal/query/rwr.go` + `rwr_test.go` — EXPL-02 core
- [ ] `internal/query/explore_gate.go` + test — EXPL-03
- [ ] Extend `internal/query/node.go` + `node_test.go` — NODE-01/02/03
- [ ] Extend `testdata/golden/capture.sh` — new behavioral invocations (multi-word explore without `--max-files 1`, overloaded `node` without `-f`) on BOTH corpora, both CLI and MCP surfaces
- [ ] New synthetic fixture corpus per D-03 (overloaded symbols, multi-word query, `Test*`-heavy weakly-connected case, structural-beats-lexical case) — likely `testdata/golden/corpus/synthetic-parity/`
- [ ] MCP-surface capture path in `capture.sh` (currently CLI-only) — needs a way to invoke the TS MCP server's `codegraph_explore`/`codegraph_node` tools programmatically (likely via `mcp-go`'s test client, or a small Node harness script driving TS's stdio MCP server directly)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A — local CLI/MCP tool, no auth surface |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes | Query strings already validated non-empty (`WR-05` pattern, `strings.TrimSpace(query) == ""`) before any scan; the new tokenizers must preserve this — an empty/whitespace-only tokenized query must not silently become a "match everything" query, matching Explore's existing `WR-05` guard. |
| V6 Cryptography | no | N/A — no crypto surface in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| DoS via unbounded RWR graph size on a pathological query (e.g. a single-character token matching thousands of nodes) | Denial of Service | Port TS's existing bounds verbatim: `maxNodes: 200` (subgraph cap before RWR), `GLUE_NODE_CAP: 60`, `searchLimit`-derived caps on each channel — these already exist in TS specifically to bound this cost; do not port RWR without also porting its input-size caps. |
| Path traversal via a crafted `fileHint` in NODE-03's narrowing | Tampering | Already mitigated by existing `resolveSourcePath`/`readSourceFile` (T-03-06-Path, `internal/query/node.go:32-89`) for the file-READ path; the NEW narrowing logic (§9) only FILTERS an already-resolved in-memory node list by substring match on `FilePath` — it does not open a new file handle, so no new path-traversal surface is introduced by NODE-03 itself. Confirm this remains true in the executor's implementation (narrowing must stay a pure in-memory filter, never a fresh disk read keyed on the raw hint). |

## Sources

### Primary (HIGH confidence — direct extraction from live TS 1.3.1 install)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/mcp/tools.js` — `computeGraphRelevance` (RWR core), `handleExplore` (full pipeline), `handleNode`/`findSymbolMatches`/`renderNodeSection` (multi-def), `buildBlastRadiusSection` (no-covering-tests warning). Confirmed via `codegraph --version` → `1.3.1`.
- `.../dist/context/index.js` — `ContextBuilder.findRelevantContext`, `extractSymbolsFromQuery`, type-hierarchy expansion.
- `.../dist/search/query-utils.js` — `STOP_WORDS`, `extractSearchTerms`, `isTestFile`, `getStemVariants`, `isDistinctiveIdentifier`.
- `.../dist/extraction/generated-detection.js` — `isGeneratedFile` (verbatim regex list).
- `.../dist/graph/traversal.js` — `getTypeHierarchy`/`getTypeAncestors`/`getTypeDescendants` (extends/implements walk).
- `.../dist/db/queries.js` — `getNodesByName` (confirms no `ORDER BY`, informing Assumption A3).

### Secondary (repo-verified, current codebase state)
- `internal/query/{explore,node,traverse,search,engine,render_markdown}.go`, `internal/cli/{explore,node}.go`, `internal/mcp/tools.go` — current Go extension points, confirmed `cobra.ExactArgs(1)` bug, confirmed shared-`Engine` structural correctness for EXPL-05/NODE-04.
- `internal/indexer/goextract/types.go` + `internal/indexer/{javaextract,csharpextract,pyextract,tsextract,mainstream}` doc comments — confirmed the 5-vs-9 edge-kind gap (every extractor mirrors goextract's kind vocabulary).
- `testdata/golden/{README.md,capture.sh,golden_test.go,golden_parity_test.go}` + `corpus/weft-go/{node,explore}.json` — confirmed existing harness's `--max-files 1`/`-f` narrowness (Pitfall 3), confirmed existing `RenderNode` already matches TS's single-def golden shape byte-for-byte (informing NODE-04's low-risk assessment).

### Tertiary (LOW confidence)
None — all claims in this research trace to a directly-read source file with a file:line citation, either the live TS install or the current repo.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; the "don't hand-roll" call is a direct mirror of TS's own implementation choice.
- Extracted TS constants/predicates (RWR params, edge-kind set, gate constant, node budget, generated-file regex, header/warning strings): HIGH — every one has a direct file:line citation against the live TS 1.3.1 dist.
- Full-pipeline architecture (hybrid gather → glue nodes → named seeding → RWR → gate → sort): HIGH for what TS actually does; MEDIUM for how much of it the plan should attempt to port (this is a scope decision flagged in Open Questions §2, not a factual uncertainty).
- Edge-kind parity gap: HIGH confidence the gap exists and its exact shape (verified against every extractor's source); MEDIUM confidence on the recommended mitigation (Open Questions §1 — genuinely a decision point, not a fact).
- Pitfalls: HIGH — each is grounded in a specific, cited code path, not speculation.

**Research date:** 2026-07-14
**Valid until:** Tied to the TS 1.3.1 install remaining available at
`/opt/homebrew/lib/node_modules/@colbymchenry/codegraph` — if that install is
upgraded or removed, this document (and its file:line citations) becomes the
frozen ground truth per `testdata/golden/README.md`'s own stated policy
("if capture.sh can no longer run ... the already-committed fixtures ...
remain the frozen ground truth; do not attempt to hand-edit them"). No
external-ecosystem staleness clock applies (this is not a moving-target
library dependency).
