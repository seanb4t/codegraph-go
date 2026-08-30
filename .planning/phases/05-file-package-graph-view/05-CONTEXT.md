# Phase 5: File/Package Graph View - Context

**Gathered:** 2026-08-30
**Status:** Ready for planning

<domain>
## Phase Boundary

A developer can see the whole repository as one readable picture at file/package
granularity, drill into any file for its symbols, and spot dependency cycles — with the
renderer chosen by recorded measurement rather than by assumption.

**Requirements:** GRF-01, GRF-02, GRF-03, GRF-04, GRF-05, ENG-03

**This phase has a BLOCKING internal gate.** The ROADMAP states it directly: *"no renderer
may be chosen and no `GRF` implementation work may be planned before `GRF-01` resolves
against a pass condition locked before dispatch."* The ordering is the point — a threshold
chosen after seeing the numbers can always be made to endorse a pre-existing preference.

**Scale facts measured on this repository's own live index (2026-08-30):**

| | |
|---|---|
| nodes / edges / files | 7,052 / 16,565 / **573** |
| edge kinds present | 11 |
| `calls` | 6,948 |
| `contains` | **6,406** |
| `references` | 1,695 |
| `instantiates` | 731 |
| `imports` | 461 |
| `returns` / `implements` / `type_of` / `extends` / `embeds` / `overrides` | 215 / 57 / 4 / 2 / 1 / 1 |

Rolled up to file granularity this repository yields on the order of **~573 nodes** —
trivial for any renderer. **This repository cannot answer GRF-01.** The measurement must
run against a large external corpus. Nine are available in `corpora/manifest.json`:
`gohugoio/hugo` (go), `nestjs/nest` (typescript), `google/guava` (java),
`JamesNK/Newtonsoft.Json` (csharp), `serilog/serilog` (csharp), `tiangolo/fastapi`,
`pydantic/pydantic`, `psf/requests` (python), `apache/arrow` (mixed).

`/graph` already exists as a Phase 2 placeholder slot (`web/src/routes/graph/+page.svelte`).
Per **D-18**, Phase 5 *fills* it and does not restructure navigation.

</domain>

<decisions>
## Implementation Decisions

### GRF-01 — the blocking spike

- **D-01:** **The pass condition is INTERACTION LATENCY at the largest corpus, and it is
  written down before the spike runs.**

  Lock a threshold on **time-to-first-paint** and **pan/zoom frame time** for the file/
  package rollup of the largest available corpus. The renderer passes if it stays
  interactive at that size.

  This measures the property criterion 2 actually cares about — *"a developer opens the
  graph view on a real repository and can read it"* — rather than a proxy. Node and edge
  counts alone were rejected: they decide a rendering question without rendering anything,
  and published scale guidance is not this repository's data.

  **The concrete threshold values are not fixed here** — they must be chosen and committed
  *before* dispatch, which is the spike plan's first task. What is fixed here is *what is
  measured* and *that the number is locked first*.

  Tie-breaks, in order, if more than one candidate passes: bundle size, then API fit for
  hierarchical layout.

- **D-02:** **Native compound-node support is a LOCKED PREREQUISITE, and this narrows the
  field before measurement. Recorded openly rather than disguised.**

  D-04 below chooses in-place expansion for GRF-03, which requires nested/parent-child
  nodes. Verified against upstream sources this session:

  - **Cytoscape.js has first-class compound nodes** — a `parent` field in node data, parent
    dimensions auto-inferred from descendants, `node.parent()` / `eles.move()` in the API.
  - **Sigma.js does not, deliberately.** Its maintainers closed the combo/grouping feature
    request stating the feature *"is too complex and too opinionated"* for Sigma's core and
    belongs in an external plugin. The only community approach is a hand-rolled parallel
    tree that hides the parent and un-hides children on interaction — whose own author
    concedes it *"doesn't really implement that nice grouping behavior"*, with edges between
    groups and intra-group placement left unsolved.

  **Consequence, stated plainly:** this prerequisite effectively narrows the field toward
  Cytoscape.js before any measurement. The maintainer accepted that narrowing explicitly
  rather than pretending the choice is open.

  **GRF-01 therefore still has a real pass/fail**, and it is not a formality: it measures
  whether the *qualifying* renderer stays interactive at the largest corpus. **If it does
  not, that is a genuine failure that forces reconsidering either the renderer or D-04's
  drill-down decision** — the spike must be planned so that outcome is reachable and
  actionable, not treated as impossible.

  The ROADMAP's phrasing ("deliberately pre-commits to neither") is honoured in the sense
  that matters: no renderer is selected by preference, and the selection is still gated on
  a recorded measurement against a pre-locked condition.

### Rollup Semantics (ENG-03, GRF-02)

- **D-03:** **`contains` is EXCLUDED from the rollup. All other edge kinds participate,
  with per-kind counts preserved on each aggregated edge.**

  `contains` is symbol-in-file containment. Rolled up to file granularity it becomes a
  self-edge on nearly every node — 6,406 edges of noise on this repository alone, second
  only to `calls`. Excluding it is the difference between a readable picture and a hairball.

  Kept: `calls`, `references`, `instantiates`, `imports`, `returns`, `implements`,
  `type_of`, `extends`, `embeds`, `overrides`.

  Rejected: *only `calls` and `imports`* (drops `references`/`instantiates`/`implements`,
  so a real dependency expressed through a type reference becomes invisible); *all 11 with
  self-edges dropped at rollup* (`contains` then contributes almost nothing after self-edge
  removal, so it is wasted scan work).

  **Self-edges are still dropped** for the retained kinds — an edge whose source and target
  roll up to the same file is not a dependency between files.

  The exact aggregation tuple (distinct source-file / target-file / kind) is pinned during
  planning per the ROADMAP's Notes, informed by GRF-01's measurement.

- **D-05:** **`Engine.FileGraph()` follows `BuildReverseAdjacency`'s discipline** —
  computed fresh per call from a full edge scan (`internal/query/traverse.go:31-43`), no
  precomputed projection, no new record kind, no re-indexing. This is ENG-03 as written and
  is not open for reinterpretation.

### Drill-down and Cycles

- **D-04:** **GRF-03 is IN-PLACE EXPANSION in the graph** — clicking a file expands it into
  its symbol nodes within the same picture, using the renderer's compound/child-node
  support.

  This keeps the developer in one view and exercises the hierarchical layout GRF-02
  requires. It is materially more renderer-specific work than the alternatives, and it is
  the decision that drives D-02's prerequisite — see there for the consequence.

  Rejected: *deep-link into `/browse?file=…`* (zero new UI, reuses Phase 3's file view and
  NAV-01's URL model, but makes "drill in" a navigation away from the graph); *side panel*
  (in-place feel without compound nodes, but a new component duplicating part of Browse).

- **D-06:** **GRF-04 cycle detection is computed SERVER-SIDE, inside `Engine.FileGraph`,
  and arrives with the data.**

  The rollup already performs a full edge scan; detecting cycles over the aggregated file
  graph there means the answer is testable in Go without a browser and stays correct
  regardless of which renderer wins GRF-01.

  **This is also what protects GRF-05.** Computing cycles client-side would tie a
  correctness property to the renderer — a swap would have to re-implement it, which is
  precisely what the swappable seam exists to prevent.

  Consequence the planner must handle: cycle output joins the wire shape, which is a
  one-way proto field-numbering decision (see Claude's Discretion).

### Post-Research Amendments (2026-08-30)

Three findings from `05-RESEARCH.md` required maintainer rulings. Recorded here as
decisions so the planner reads settled ground, not open questions.

- **D-07:** **The layout extension is `cytoscape-elk` (with `elkjs`). This is forced by the
  two locked constraints, not chosen by preference.**

  Research established a genuine three-way tension, and only one option survives it:
  - `cytoscape-fcose` — best compound support, but **self-describes as force-directed**,
    which GRF-02 bans outright.
  - `cytoscape-dagre` — hierarchical and small, but **zero compound support**, which D-04's
    in-place expansion requires.
  - `cytoscape-elk` — the only candidate confirmed both non-force-directed and
    compound-aware (`hierarchyHandling`). `elkjs`'s own description: *"Automatic graph
    layout based on Sugiyama's algorithm."*

  **Bundle cost, measured not assumed:** cytoscape core ~137KB gzip + elkjs ~423KB gzip
  ≈ 560KB, against a current embedded JS bundle of **160KB gzip** (`web/build/`, 892K on
  disk). That is a 3.5× increase in the JS — but it embeds into a **75MB binary**, so it is
  **~0.7% of what ships**, in an artifact the tree-sitter grammars already dominate by
  three orders of magnitude. The maintainer accepted the cost on that basis, consistent
  with the standing ruling to prefer established dependencies over half-baked alternatives.

  Versions confirmed live: `cytoscape@3.34.2`, `cytoscape-elk@2.3.0`, `elkjs@0.12.0`.

  This does NOT pre-empt GRF-01. The spike still measures whether this stack stays
  interactive at the largest corpus, and a failure remains actionable — see D-02.

- **D-08:** **`FileGraph()` MUST exclude `Kind == "package"` nodes from the rollup.**

  Research found a real defect that hides on the measurement corpus and appears in
  production. `internal/indexer/resolve.go` mints synthetic `"package"`-kind pseudo-nodes
  for intra-module import targets — its own comment says they are *"not one of goextract's
  declared node kinds, since no source declaration produces it"* — and they carry
  `FilePath == ""`. There are **43** in this repository (`nodesByKind.package: 43`,
  measured live). A naive node→file lookup rolls their edges into a phantom `""` file.

  Verified impact of the exclusion: **573→572 nodes, 1,540→1,326 edges**.

  **`google/guava` has ZERO such nodes.** So `GRF-01`'s measurement corpus cannot surface
  this, and the defect would appear only when a developer opened the graph on *this*
  repository — at launch.

  The fix stays inside this phase's new code; `resolve.go` is NOT changed (that would alter
  indexer behaviour every CLI, MCP and golden consumer depends on). **A regression test
  against this repository's own index is mandatory** — guava cannot catch it, which is
  precisely why an explicit test is required rather than relying on corpus coverage.

- **D-09 (CORRECTION to D-05):** **`FileGraph()` requires TWO scans, not one.**

  D-05 states FileGraph follows `BuildReverseAdjacency`'s discipline. The *discipline*
  (fresh per call, no precomputed projection, no new record kind) holds and is unchanged.
  But the *single-scan* shape does not transfer: `schema.Edge` carries only node IDs, and
  node IDs are opaque content hashes — so resolving a node to its containing file needs a
  node scan before the edge scan can be aggregated. Confirmed by reading `graph.pb.go` and
  `nodeid.go`. Do not attempt to force a single pass.

  **Open, for the planner:** research estimated (did not measure) the wire response at
  guava scale at ~5-6MB against a 16MB `transportSendMaxBytes` ceiling. Treat that as
  unverified and measure it — an estimate presented as a measurement is the failure mode
  this phase's own GRF-01 exists to avoid.

### Claude's Discretion

- **The `Engine.FileGraph` wire shape.** A new rpc means another additive, one-way proto
  field-numbering decision of the same class as Phase 4's `GetHealth` — which earned a
  `blocking-human` `checkpoint:decision` before codegen. The planner should expect to
  schedule the same gate. Note the naming landmine: `internal/uiserver/readonly_test.go`'s
  `mutatingVerbs` fixture is matched by bare `strings.Contains` over method names, so the
  chosen rpc name must be checked against all 19 substrings before it is frozen.
- **What "laid out hierarchically with directory-structural grouping" means concretely**,
  within GRF-02's prohibition on a force-directed whole-graph view.
- **The GRF-05 seam interface** — exactly what a renderer swap must not reach past.
- **Which corpus counts as "largest"**, and whether the spike must index it first. `guava`
  and `arrow` are the plausible candidates; the spike must record the actual rollup counts
  rather than assuming from repository size.
- Visual treatment of cycles (GRF-04) beyond "distinguished without hunting".

</decisions>

<specifics>
## Specific Ideas

- **The maintainer accepted a narrowing rather than disguising it.** Offered four ways to
  resolve the compound-node/GRF-01 tension — including dropping in-place expansion to keep
  the measurement fully open — the choice was to keep in-place expansion and **record the
  narrowing explicitly**. Downstream agents should read D-02 as the authority on what
  GRF-01 does and does not decide, and must not describe the renderer selection as an open
  two-way comparison.

- **Phase 4 established that machine verification is categorically insufficient for
  user-facing properties.** Five verification layers passed a `/workbench` page that
  rendered *"Phase 4: run the four graph analyses…"* — internal planning vocabulary — as
  its subtitle, because no test asserts the absence of planning vocabulary in rendered
  copy. **`web/src/routes/graph/+page.svelte:9` currently renders
  `"Phase 5: the whole repository as one readable picture at file/package granularity."`
  This phase fills that route and MUST replace that line.** A live browser check against
  the built binary is expected before this phase transitions.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` §"Phase 5: File/Package Graph View" — goal, the five success
  criteria, the blocking-within-phase constraint, and the Notes on aggregation semantics
  and the "largest real corpus" requirement.
- `.planning/REQUIREMENTS.md` — ENG-03 (line 33), GRF-01..05 (lines 64-68).

### The engine seam ENG-03 follows
- `internal/query/traverse.go:31-43` — `BuildReverseAdjacency`: full edge scan, fresh per
  call, `IterateEdges("")` with a kind filter. The shape D-05 mirrors.
- `internal/query/status.go:45,58,207-228` — `edgesByKind`, a read-time-derived per-kind
  tally from one full edge scan; documents the same fresh-per-call discipline and the
  sparse-map convention (a kind with zero edges is absent, never present with value 0).
- `internal/indexer/goextract/types.go:33-36,59-62` — the edge-kind constants
  (`calls`, `imports`, `embeds`, `contains`, `references`, `instantiates`, `returns`,
  `type_of`).

### Measurement inputs
- `corpora/manifest.json` — the nine available corpora. `corpora/observations.json` and
  `corpora/selection.json` are the existing recorded-measurement artifacts; GRF-01's
  measurement should follow their conventions rather than inventing a new format.

### Prior-phase decisions this phase builds on
- `.planning/phases/04-query-workbench-index-health/04-CONTEXT.md` — D-01 (additive rpc
  rather than extending a hot message), D-06 (one shared table), D-07 (bounds stay
  server-side), and the ⚠ CORRECTED block under D-01 on the `mutatingVerbs` naming landmine.
- `.planning/phases/03-browse-inspect-navigation/03-CONTEXT.md` — D-13 (one URL grammar per
  view), D-04 (one Connect-error classification), D-18 (placeholder routes are filled, not
  restructured), D-22 (the vendored-component supply-chain gap).
- `.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md` — D-02a
  additive-only proto discipline; D-04's corrected exit-status-AND-count verification rule.

### Surfaces being extended
- `web/src/routes/graph/+page.svelte` — the placeholder this phase fills (and whose
  `Phase 5:` subtitle it must replace).
- `internal/uiproto/uiv1/ui.proto` — 11 rpcs today; `FileGraph` would be the 12th.
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods`, the method-count literal
  (currently `11`), and the `mutatingVerbs` fixture that must never be weakened.
- `web/src/lib/components/workbench/DataTable.svelte` — the one shared table, if any
  tabular surface is needed.

### External library documentation (verified this session)
- Cytoscape.js compound nodes — `parent` field, auto-inferred parent dimensions,
  `node.parent()` / `eles.move()`; compounds form a strict tree, one parent maximum.
- `jacomyal/sigma.js#1201` — maintainers declining combo/grouping in core, and the
  hand-rolled parallel-tree workaround with its stated limitations.

### Standing repository rules
- Rule `84d1gfpywd` — every guard carries a positive assertion that it did its work.
  Corollary learned in Phase 4: **pair every upper bound with a non-zero lower bound** —
  `toBeLessThanOrEqual(N)` is satisfied by 0.
- Rule `f18zrdsgx5` — never `[ci skip]`; `protect-main` requires 6 status checks.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`BuildReverseAdjacency`** (`traverse.go:31`) — the exact full-scan, fresh-per-call
  shape ENG-03 is specified against. Note it filters to a single kind; `FileGraph` keeps
  more, so the filter predicate differs but the discipline does not.
- **`edgesByKind`'s scan** (`status.go:207-228`) — an existing precedent for deriving a
  per-kind tally from one full edge iteration, including the sparse-map convention.
- **`corpora/`** — nine real repositories with existing manifest/observation/selection
  artifacts. GRF-01 does not need to invent a measurement-recording format.
- **`DataTable.svelte`** — the one shared sortable table from Phase 4, if a tabular surface
  is wanted alongside the graph.
- **`rpc-errors.ts` / `workbench-failure.ts`** — existing Connect-error classification;
  a graph view's failures should compose these rather than adding a third vocabulary.
- **`/graph` route** — already mounted in the layout nav; no navigation work required.

### Established Patterns
- **Additive-only proto** (D-02a), with a `blocking-human` gate before codegen freezes
  field numbers.
- **Ordinary handler shape** — `withEngine(...)` plus a named `xToProto` mapper, as
  `Callers`/`Callees`/`Files`/`GetHealth` do. `GetStatus`'s degrade-and-answer form is a
  documented single exception and must not be copied.
- **Fresh-per-request derivation** over precomputed projections — `edgesByKind`,
  `BuildReverseAdjacency`, and now `FileGraph`.
- **One X per concern** — one URL grammar per view, one error classifier, one table, one
  status verdict.

### Integration Points
- `internal/query/` — where `Engine.FileGraph()` lands, beside `traverse.go`.
- `internal/uiserver/handlers.go` + `ui.proto` — the 12th rpc and its mapper.
- `web/src/routes/graph/+page.svelte` — the view.
- `web/package.json` — the renderer dependency, which will need a package-legitimacy
  checkpoint of the same class Phase 4 ran twice.

</code_context>

<deferred>
## Deferred Ideas

- **Deep-linking a graph node into `/browse`** — rejected as GRF-03's mechanism in favour
  of in-place expansion, but a *secondary* affordance (e.g. a context action on an expanded
  symbol) remains reasonable and costs almost nothing, since NAV-01's URL model already
  supports it.
- **A side panel listing a file's symbols** — the third GRF-03 option; revisit only if
  in-place expansion proves unreadable at scale.
- **Precomputed rollup projection** — explicitly out of scope per ENG-03/GRF-05
  ("no precomputed projection, no new record kind, no re-indexing"). If `FileGraph`'s
  fresh-per-call scan proves too slow at corpus scale, that is a finding to record and
  escalate, not to fix by caching inside this phase.
- **Sigma.js + graphology with a hand-rolled grouping layer** — the path D-02's
  prerequisite narrows away. Revisit only if GRF-01's latency measurement fails for the
  qualifying renderer.
- **Overlapping / multi-parent groupings** — Cytoscape's compound model is a strict tree,
  one parent maximum. Anything needing overlapping sets would require an extension
  (`cytoscape.js-bubblesets` or similar) and is out of scope.

</deferred>

---

*Phase: 05-file-package-graph-view*
*Context gathered: 2026-08-30*
