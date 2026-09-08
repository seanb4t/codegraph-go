# Phase 5: File/Package Graph View - Research

**Researched:** 2026-08-30
**Domain:** Server-side graph rollup (Go) + Cytoscape.js compound-node rendering (Svelte 5)
**Confidence:** MEDIUM-HIGH — the highest-value numbers (GRF-01's scale inputs) are MEASURED
this session against a real indexed corpus, not estimated. Renderer API surface is
CITED/VERIFIED against official docs and the installed dependency graph. GRF-01's actual
pass/fail verdict is explicitly NOT decided here — that is the blocking spike's own job.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**GRF-01 — the blocking spike**

- **D-01:** The pass condition is INTERACTION LATENCY at the largest corpus, locked before
  the spike runs. Lock a threshold on time-to-first-paint and pan/zoom frame time for the
  file/package rollup of the largest available corpus. The renderer passes if it stays
  interactive at that size. Node/edge counts alone were rejected as a proxy. The concrete
  threshold values are NOT fixed in CONTEXT.md — the spike plan's first task is to choose
  and commit them. Tie-breaks, in order, if more than one candidate passes: bundle size,
  then API fit for hierarchical layout.

- **D-02:** Native compound-node support is a LOCKED PREREQUISITE, and this narrows the
  field before measurement — recorded openly, not disguised. Cytoscape.js has first-class
  compound nodes (`parent` field, auto-inferred parent dimensions, `node.parent()` /
  `eles.move()`). Sigma.js does not — maintainers closed the combo/grouping feature request
  as "too complex and too opinionated" for core. This narrows the field toward Cytoscape.js
  before any measurement, accepted explicitly rather than pretended open. GRF-01 still has a
  real pass/fail: it measures whether the qualifying renderer stays interactive at the
  largest corpus. If it does not, that forces reconsidering the renderer or D-04's
  drill-down decision — the spike must be planned so that outcome is reachable. **This
  research MUST NOT describe the renderer selection as an open two-way comparison** — Sigma
  is disqualified by D-02 before GRF-01 runs; GRF-01 judges only whether Cytoscape.js passes.

**Rollup Semantics (ENG-03, GRF-02)**

- **D-03:** `contains` is EXCLUDED from the rollup. All other edge kinds participate
  (`calls`, `references`, `instantiates`, `imports`, `returns`, `implements`, `type_of`,
  `extends`, `embeds`, `overrides`), with per-kind counts preserved per aggregated edge.
  Self-edges are dropped for the retained kinds (source and target rolling up to the same
  file is not a cross-file dependency). The exact aggregation tuple (distinct
  source-file/target-file/kind) is pinned during planning, informed by GRF-01's measurement.

- **D-05:** `Engine.FileGraph()` follows `BuildReverseAdjacency`'s discipline — computed
  fresh per call from a full edge scan (`internal/query/traverse.go:31-43`), no precomputed
  projection, no new record kind, no re-indexing. This is ENG-03 as written, not open for
  reinterpretation.

**Drill-down and Cycles**

- **D-04:** GRF-03 is IN-PLACE EXPANSION — clicking a file expands it into its symbol nodes
  within the same picture, using the renderer's compound/child-node support. Rejected:
  deep-link into `/browse?file=…`; a side panel.

- **D-06:** GRF-04 cycle detection is computed SERVER-SIDE, inside `Engine.FileGraph`, and
  arrives with the data — testable in Go without a browser, correct regardless of which
  renderer wins GRF-01, and protects GRF-05 (client-side cycle detection would tie a
  correctness property to the renderer).

### Claude's Discretion

- The `Engine.FileGraph` wire shape — another additive, one-way proto field-numbering
  decision of GetHealth's class; expect a `blocking-human checkpoint:decision` before
  codegen. The chosen rpc name must be checked against all 19 `mutatingVerbs` substrings in
  `internal/uiserver/readonly_test.go` before it is frozen.
- What "laid out hierarchically with directory-structural grouping" means concretely, within
  GRF-02's prohibition on a force-directed whole-graph view.
- The GRF-05 seam interface — exactly what a renderer swap must not reach past.
- Which corpus counts as "largest", and whether the spike must index it first.
- Visual treatment of cycles (GRF-04) beyond "distinguished without hunting".

### Deferred Ideas (OUT OF SCOPE)

- Deep-linking a graph node into `/browse` as a *secondary* affordance (context action on an
  expanded symbol) — reasonable later, not GRF-03's mechanism.
- A side panel listing a file's symbols — revisit only if in-place expansion proves
  unreadable at scale.
- Precomputed rollup projection — explicitly out of scope per ENG-03/GRF-05. If the
  fresh-per-call scan proves too slow at corpus scale, that is a finding to record and
  escalate, not to fix by caching inside this phase.
- Sigma.js + graphology with a hand-rolled grouping layer — the path D-02's prerequisite
  narrows away. Revisit only if GRF-01's latency measurement fails for the qualifying
  renderer.
- Overlapping/multi-parent groupings — Cytoscape's compound model is a strict tree, one
  parent maximum. Out of scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GRF-01 | Spike measures real file/package rollup node/edge counts against indexed repos, pass condition locked before dispatch, result selects the renderer | §"GRF-01 scale inputs — measured, not estimated" below supplies the actual numbers the spike's locked threshold needs; §"Renderer API surface" resolves D-02's compound-node prerequisite claims against live docs |
| GRF-02 | Whole-repo file/package graph, aggregated edges, hierarchical directory-structural layout, never force-directed | §"Layout extension comparison" — the three cytoscape layout plugins evaluated against "hierarchical, not force-directed, compound-aware" |
| GRF-03 | Drill into a file to see its symbols | §"Cytoscape.js compound-node API" — `parent` field, `eles.move()`, in-place expansion pattern |
| GRF-04 | Dependency cycles visually distinguished | §"Cycle detection over the rolled-up graph" |
| GRF-05 | Rendering library sits behind a narrow, swap-safe component seam | §"Architecture Patterns" — component seam pattern; §"Don't Hand-Roll" |
| ENG-03 | `Engine.FileGraph()` aggregated adjacency, fresh per call, `BuildReverseAdjacency` discipline | §"`Engine.FileGraph()` concrete shape" — the two-scan requirement and the package-pseudo-node pitfall |
</phase_requirements>

## Summary

GRF-01's pass condition needs a real scale number to threshold against, and this session
produced one by actually indexing the largest available corpus and computing the file-level
rollup with this repo's own graphstore package — not estimating from published guidance.
`google/guava` (already checked out and pre-indexed on this machine) rolls up to **3,233
file-level nodes and 21,554 distinct (source-file, target-file) aggregated edges** (26,355
if counting each retained edge-kind as a separate tuple) after excluding `contains` and
self-edges per D-03. This repository's own rollup is two orders of magnitude smaller: 572
nodes, 1,057 file-pair edges. The spike's locked threshold should target the guava-scale
number, not this repository's own trivial scale.

A structural defect was found and fixed in the process: this repository's index carries 43
synthetic `"package"`-kind pseudo-nodes (`internal/indexer/resolve.go:21-25`) minted for
unresolved *intra-module* imports — they carry `FilePath == ""`. A naive
`Engine.FileGraph()` that builds its node→file lookup by scanning `IterateNodes()` without
excluding `Kind == "package"` will silently roll edges into a phantom empty-string "file"
node, corrupting both the node count and the edge endpoints. Guava (Java) has zero such
nodes, so this defect is invisible on the corpus GRF-01 will actually be measured against —
but it will corrupt this repository's own graph view at launch unless `FileGraph()`
explicitly excludes the pseudo-package kind.

D-02's compound-node prerequisite is confirmed against live Cytoscape.js docs: `parent` is
set at node-creation time and is immutable via `ele.data()`, but mutable via `eles.move()` —
exactly what in-place expansion (D-04) needs. The layout-extension choice is *not* a solved
problem D-02 leaves open, though: of the three plausible Cytoscape layout plugins,
**`cytoscape-fcose` is textually force-directed** ("combines the speed of spectral layout
with the aesthetics of force-directed layout") — in direct tension with GRF-02's explicit
"never force-directed" prohibition, despite being the best-documented compound-aware option.
**`cytoscape-dagre` has no documented compound-node support at all.** **`cytoscape-elk`**
(wrapping `elkjs`'s `layered` algorithm, a genuine Sugiyama-style DAG layout, confirmed
non-force-directed) **is the only option that is both non-force-directed and has documented
hierarchy support** (`elk.hierarchyHandling`), at a real bundle-size cost: elkjs alone is
~433KB gzipped — roughly 3x cytoscape core's own 137KB gzipped.

**Primary recommendation:** Build `Engine.FileGraph()` as a two-scan operation (node→file
map, then edge aggregation), excluding both `contains` and any node with
`Kind == "package"`, following `BuildReverseAdjacency`'s fresh-per-call discipline. Wire it
as a 12th, additive `FileGraph` rpc (name confirmed clean against all 19 `mutatingVerbs`
substrings) with a `map<string, int64>` per-edge kind-count field, mirroring
`GetHealthResponse.edges_by_kind`'s exact precedent. Run GRF-01's spike against
`google/guava`'s already-indexed 3,233/21,554-scale rollup, using this session's numbers as
the sizing input for the threshold the spike locks before dispatch — and budget the
elk-vs-fcose bundle-size/force-directed tension as an explicit tie-break question the spike
plan must resolve, not a foregone conclusion.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| File/package rollup aggregation (ENG-03) | API/Backend (`internal/query.Engine`) | — | Fresh-per-call full scan over the existing Pebble-backed graph store; no new persistence, mirrors `BuildReverseAdjacency`/`edgesByKind` |
| Cycle detection (GRF-04, D-06) | API/Backend | — | Locked by D-06 specifically to keep correctness renderer-independent (protects GRF-05) |
| Wire transport (12th rpc) | API/Backend → Browser/Client | — | ConnectRPC unary rpc, same `withEngine` handler shape as `Callers`/`Callees`/`GetHealth` |
| Graph rendering, layout, pan/zoom | Browser/Client | — | Cytoscape.js is a canvas-rendering client library; SvelteKit route runs with `ssr = false` (`web/src/routes/+layout.ts:5`), so no SSR/prerender concern exists for this browser-only dependency |
| In-place expansion (GRF-03) | Browser/Client | API/Backend (data source) | The expand *interaction* is client-side (`eles.move()`/compound reparent); the symbol data it reveals still comes from the existing `GetNodeDetail`/`Callers`/`Callees` rpcs already used by Browse, not a new endpoint |
| Directory-structural grouping (GRF-02) | Browser/Client (compound-node construction from `FilePath`) | API/Backend (raw file paths) | The server returns flat file paths; the client derives the directory hierarchy (via `path.dirname`-style splitting) and constructs Cytoscape compound `parent` chains — no new server-side "directory" concept is needed since `schema.Node.FilePath` already carries full repo-relative paths |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `cytoscape` | 3.34.2 `[VERIFIED: npm registry — npm view cytoscape version dist-tags time.created time.modified, this session]` | Graph rendering, compound nodes, pan/zoom | D-02's locked prerequisite; only mainstream JS graph library with first-class compound-node support confirmed this session against official docs |
| `cytoscape-elk` | 2.3.0 `[VERIFIED: npm registry]` | Layout extension: hierarchical/layered DAG layout via ELK, with `hierarchyHandling` for compound/nested graphs | Only one of the three plausible layout extensions confirmed BOTH non-force-directed AND compound-aware this session — see "Layout extension comparison" below. **Not yet chosen** — GRF-01's spike must weigh this against `cytoscape-fcose`'s force-directed tension with GRF-02 |
| `elkjs` | pulled in transitively by `cytoscape-elk` at range `^0.9.3` `[VERIFIED: npm view cytoscape-elk dependencies]` — NOT the 0.12.0 bundle-size figure below, which was measured against latest | The actual layout algorithm; `cytoscape-elk` is a thin adapter over it | Confirmed non-force-directed: "a layer-based layout algorithm... based on the ideas originally introduced by Sugiyama et al." `[CITED: github.com/kieler/elkjs README, fetched this session]` |

### Supporting (evaluated, not both recommended — see Common Pitfalls)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `cytoscape-dagre` | 4.0.1 `[VERIFIED: npm registry]` | Alternative hierarchical DAG layout, smaller bundle (~25KB gzip core `dagre` dep, bundled) | Only if GRF-01's spike needs a lighter-weight hierarchical fallback AND the plan is willing to build directory grouping without native compound support (dagre's README documents none) |
| `cytoscape-fcose` | 2.2.0 `[VERIFIED: npm registry]` | Compound-aware "fast compound spring embedder" layout | **Flagged tension, not recommended by default:** its own README self-describes as force-directed (spectral+force hybrid) — in direct textual conflict with GRF-02's "never force-directed" prohibition. Only reconsider if `cytoscape-elk` fails GRF-01's interaction-latency bar at guava scale |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Cytoscape.js | Sigma.js + graphology | Disqualified by D-02 before measurement (no compound-node support in core; maintainers explicitly declined). Not re-litigated here per CONTEXT.md's instruction |
| `cytoscape-elk` | `cytoscape-dagre` | Smaller, simpler, but zero documented compound-node support — would require a hand-rolled workaround to satisfy D-04's compound-based in-place expansion, defeating the reason D-02 narrowed toward Cytoscape.js in the first place |
| `cytoscape-elk` | `cytoscape-fcose` | Compound-aware and well-documented, but is a force-directed algorithm by its own README's description — a real, not merely cosmetic, conflict with GRF-02's explicit prohibition that the plan must address head-on rather than paper over |

**Installation** (not yet run against `web/package.json` — this is what the phase's install
task will run; do not run it during research):
```bash
cd web && pnpm add cytoscape cytoscape-elk
# elkjs is pulled in transitively via cytoscape-elk's own `elkjs: "^0.9.3"` dependency —
# no separate `pnpm add elkjs` needed.
```

**Version verification:** All four package versions above and their publish metadata were
confirmed live this session via `npm view <pkg> version dist-tags time.created` — see
Package Legitimacy Audit for the full signal set including weekly downloads and repo links.

## Package Legitimacy Audit

Ran `gsd_run query package-legitimacy check --ecosystem npm cytoscape cytoscape-elk elkjs
cytoscape-dagre cytoscape-fcose` this session — verbatim output below.

| Package | Registry | Published | Weekly Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----------|-------------------|--------------|---------|-------------|
| `cytoscape` | npm | 2026-08-25 (latest 3.34.2; project itself created 2012-10-18) | 15,505,561 | github.com/cytoscape/cytoscape.js | **[SUS]** — reason: `too-new` | **Flagged, not removed.** Same false-positive class already approved twice in this milestone (03-01 `@testing-library/jest-dom`, 04-01 `@tanstack/svelte-table`/`shadcn-svelte`): a fresh *release* on a 14-year-old, 15.5M-weekly-download, org-maintained project is not a new or hijacked package. Planner must still add `checkpoint:human-verify` per protocol |
| `cytoscape-elk` | npm | 2024-11-26 | 30,173 | github.com/cytoscape/cytoscape.js-elk (official Cytoscape org) | OK | Approved |
| `elkjs` | npm | 2026-07-17 (latest 0.12.0) | 7,021,877 | github.com/kieler/elkjs (Eclipse Kiel project) | OK | Approved |
| `cytoscape-dagre` | npm | 2026-08-28 (2 days before this research session) | 343,162 | github.com/cytoscape/cytoscape.js-dagre (official Cytoscape org) | **[SUS]** — reason: `too-new` | **Flagged, not removed.** Same false-positive class — official-org package, 343K weekly downloads, on a normal maintenance cadence. Only relevant if the plan falls back to dagre |
| `cytoscape-fcose` | npm | 2023-01-17 | 13,922,046 | github.com/iVis-at-Bilkent/cytoscape.js-fcose (academic research group, long-standing) | OK | Approved (but see the force-directed tension flagged above — a legitimacy pass and a design-fit pass are different questions) |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** `cytoscape` (too-new false positive on a 2012-vintage package), `cytoscape-dagre` (too-new false positive on an official-org package, only relevant if dagre is chosen over elk). Both require a `checkpoint:human-verify` task before install per protocol, mirroring the approval precedent already twice established this milestone.

No postinstall scripts on any of the five packages checked (`npm view <pkg>
scripts.postinstall` returned empty for all five, confirmed this session) — no
`BLD-05`-relevant lifecycle-script risk.

None of `cytoscape`, `cytoscape-elk`, `cytoscape-dagre`, `cytoscape-fcose`, or `elkjs` ship a
`types` field pointing to bundled TypeScript definitions **except** `cytoscape` itself
(`index.d.ts`) and `cytoscape-dagre` (`index.d.ts`). `cytoscape-elk` and `cytoscape-fcose`
have no bundled types — a real DX friction point if `cytoscape-elk` is chosen; the plan
should budget a small `declare module 'cytoscape-elk'` ambient-type shim (or check for a
DefinitelyTyped `@types/cytoscape-elk` package at implementation time — not checked this
session).

## Architecture Patterns

### System Architecture Diagram

```
Browser (Svelte 5, /graph route, ssr=false)
  │
  │ 1. mount → ConnectRPC unary call: FileGraph(FileGraphRequest{})
  ▼
uiserver.FileGraph handler (internal/uiserver/handlers.go)
  │  withEngine(ctx, repoPath, func(eng *query.Engine) error { ... })  ← same shape as Callers/Callees/GetHealth
  ▼
query.Engine.FileGraph() (internal/query/traverse.go, new function beside BuildReverseAdjacency)
  │
  ├─ SCAN 1: r.IterateNodes() → build map[nodeID]filePath, SKIPPING Kind == "package" (pseudo-nodes, no real file)
  │
  ├─ SCAN 2: r.IterateEdges("") → for each edge:
  │     skip Kind == "contains"
  │     skip if either endpoint's node was excluded in Scan 1 (package pseudo-node, or otherwise unresolved)
  │     resolve srcFile, tgtFile via the Scan-1 map
  │     skip if srcFile == tgtFile (self-edge at file granularity, D-03)
  │     aggregate into map[(srcFile,tgtFile,kind)]count
  │
  ├─ Build the FileGraph node list from the set of distinct real files
  │
  └─ Cycle detection (D-06): Tarjan's SCC (or DFS 3-color) over the aggregated
     (srcFile → tgtFile) adjacency — a plain directed graph at this point, since
     self-edges are already excluded (a 1-cycle cannot occur)
  ▼
FileGraphResponse{ nodes: [...], edges: [...with per-kind map<string,int64> counts...], cycle_membership: [...] }
  ▼ (wire, ConnectRPC/protobuf, same 16MB WithSendMaxBytes ceiling as every other rpc)
Browser: transform response → Cytoscape elements JSON
  │  nodes: file nodes as leaves, directory-derived compound parents built client-side
  │         from FilePath splitting (server sends flat paths, not a directory tree)
  │  edges: one Cytoscape edge per aggregated (src,tgt) pair, kind-count map on edge data
  │         for the "per-kind counts" GRF-02 requires
  ▼
cy = cytoscape({ container, elements, style, layout: { name: 'elk', ... } })
  │  layout.run() computes hierarchical positions (never force-directed, GRF-02)
  ▼
User clicks a file node → GRF-03: node.move()-based in-place expansion
  reveals symbol children fetched via the EXISTING GetNodeDetail/Callers/Callees rpcs
  (Browse's rpcs — no new endpoint for symbol-level detail)
```

### Recommended Project Structure

```
internal/query/
├── traverse.go          # Engine.FileGraph() lands here, beside BuildReverseAdjacency (D-05)
└── filegraph_cycles.go  # (new) Tarjan's SCC / cycle detection helper, kept separate for
                          #  unit-testability independent of the rollup scan itself

internal/uiproto/uiv1/
└── ui.proto              # 12th rpc: `rpc FileGraph(FileGraphRequest) returns (FileGraphResponse);`

internal/uiserver/
└── handlers.go            # fileGraphToProto mapper, ordinary withEngine shape

web/src/routes/graph/
└── +page.svelte           # fills the Phase-2 placeholder; the "Phase 5:" line MUST be replaced

web/src/lib/components/graph/
├── GraphCanvas.svelte      # GRF-05's swap seam: the ONLY component that imports 'cytoscape'
│                           #  directly. Props in (elements, layout config), events out
│                           #  (nodeClick). Nothing outside this file touches the cytoscape
│                           #  API — this is what "a swap would not reach past" means concretely.
└── file-graph-transform.ts # pure function: FileGraphResponse (wire shape) → Cytoscape
                            #  elements JSON (compound parent assignment from FilePath
                            #  splitting, cycle-membership → CSS class). Zero DOM
                            #  dependency — the part that IS unit-testable in plain vitest
                            #  without jsdom's canvas/layout limitations (see Validation
                            #  Architecture)
```

### Pattern 1: Two-scan rollup with an exclusion set built in the first pass

**What:** `BuildReverseAdjacency` (the ENG-03 precedent, `internal/query/traverse.go:31-50`)
is a *single*-scan pattern because `schema.Edge` carries no file information — filtering by
`Kind` alone is enough. `FileGraph` cannot use a single scan: `schema.Edge{Source, Target}`
holds only node IDs (`internal/schema/graph.pb.go:214-236`, verified this session — the
struct has no file field at all), and `schema.Node.FilePath` (field 5,
`internal/schema/graph.pb.go:41-68`) is the only place a file path lives. Node IDs are a
content hash (`<kind>:<32-hex>`, confirmed via
`internal/indexer/nodeid/nodeid.go:43`/`nodeid_test.go:8`) — **not derivable from the ID
string itself**, so a node→file lookup table must be built from a full `IterateNodes()` pass
before the edge scan can resolve anything.

**When to use:** Any rollup that needs to project edges onto an attribute (file, in this
case) that lives on the node record, not the edge record.

**Example (concrete Go shape, verified against the live schema and store interfaces this
session):**
```go
// internal/query/traverse.go, beside BuildReverseAdjacency
func (e *Engine) FileGraph() (FileGraphResult, error) {
    r, err := e.snapshot() // however Engine currently opens a Reader — mirror existing callers
    if err != nil { return FileGraphResult{}, err }

    // SCAN 1: node id -> file path, EXCLUDING kindPackage pseudo-nodes (FilePath=="")
    nit, err := r.IterateNodes()
    if err != nil { return FileGraphResult{}, err }
    defer nit.Close()
    fileOf := make(map[string]string, /* nodeCount estimate */ 0)
    for nit.Next() {
        n := nit.Node()
        if n.Kind == "package" { // internal/indexer/resolve.go:25's kindPackage constant —
            continue             // synthetic pseudo-node, no real file, must not roll up
        }
        fileOf[n.Id] = n.FilePath
    }
    if err := nit.Err(); err != nil { return FileGraphResult{}, err }

    // SCAN 2: aggregate edges by (srcFile, tgtFile, kind), dropping contains + self-edges
    eit, err := r.IterateEdges("")
    if err != nil { return FileGraphResult{}, err }
    defer eit.Close()
    type key struct{ src, tgt, kind string }
    agg := make(map[key]int64)
    for eit.Next() {
        e := eit.Edge()
        if e.Kind == "contains" { continue }        // D-03
        sf, ok1 := fileOf[e.Source]
        tf, ok2 := fileOf[e.Target]
        if !ok1 || !ok2 || sf == tf { continue }     // unresolved endpoint, or file-level self-edge (D-03)
        agg[key{sf, tf, e.Kind}]++
    }
    if err := eit.Err(); err != nil { return FileGraphResult{}, err }
    // ... build node list from the set of distinct sf/tf values seen, run cycle detection, return
}
```

### Pattern 2: Compound-node construction and reparenting (Cytoscape)

**What:** Compound nodes are declared via a `parent` field in a node's `data` object.
`[CITED: cytoscape.js unstable docs, collection/removeData.md and notation.md, fetched via
Context7 this session]`: *"Compound nodes are specified via the `parent` field in a node's
`data`. Similar to the `source` and `target` fields of edges, the `parent` field is normally
immutable: A node's parent can be specified when the node is added to the graph, and after
that point, this parent-child relationship is immutable via `ele.data()`. However, you can
move child nodes via `eles.move()`."* `id`, `source`, `target`, and `parent` are all listed
together as the immutable-via-`removeData()` fields.

**When to use:** Directory-structural grouping (GRF-02) and in-place expansion (GRF-03) both
need this — directories become compound parent nodes containing file-node children;
expanding a file becomes moving its newly-fetched symbol nodes to have that file as their
`parent`.

**Example:**
```javascript
// Source: cytoscape.js official docs (collection/move.md, notation.md), Context7, this session
// Initial construction: directory compounds + file children
const elements = [
  { data: { id: 'internal/query' } },                          // compound parent (a directory)
  { data: { id: 'internal/query/traverse.go', parent: 'internal/query' } }, // file, child
  // ... one edge per aggregated (src-file,tgt-file) pair, kind-counts on edge data
];

// GRF-03 in-place expansion: reparent newly-fetched symbol nodes under the clicked file
cy.add(symbolNodes.map(s => ({ data: { id: s.id, parent: fileNodeId, label: s.name } })));
// Existing nodes are reparented via move(), never by mutating .data() directly:
someNode.move({ parent: newParentId });
```

**Container mount lifecycle (Svelte 5 runes — modeled on this repo's own precedent for
wiring an imperative library, `web/src/lib/components/workbench/DataTable.svelte:95-123`'s
`$effect`-based `createVirtualizer` wiring):**
```svelte
<script lang="ts">
  import cytoscape from 'cytoscape';
  let container: HTMLDivElement;
  let { elements, layoutName }: { elements: unknown[]; layoutName: string } = $props();

  $effect(() => {
    const cy = cytoscape({ container, elements, layout: { name: layoutName } });
    return () => cy.destroy(); // cy.destroy() cleanup, CITED cytoscape.js core/destroy.md
  });
</script>
<div bind:this={container} class="h-full w-full" data-testid="file-graph-canvas"></div>
```
`cy.resize()` must be called manually whenever the container's CSS dimensions change —
Cytoscape does not observe arbitrary DOM resize, only the `window` resize event `[CITED:
cytoscape.js core/resize.md]`.

### Anti-Patterns to Avoid

- **Client-side cycle detection:** D-06 locks this server-side specifically to keep GRF-04's
  correctness independent of GRF-01's renderer outcome. Do not compute SCCs in
  `file-graph-transform.ts` even though it would be easy to bolt on — that recreates exactly
  the coupling GRF-05's seam exists to prevent.
- **A single edge-scan pass for `FileGraph`:** Unlike `BuildReverseAdjacency`, one scan is
  structurally insufficient — see Pattern 1. Do not "optimize" this into one pass by trying
  to read `schema.Node.FilePath` off the edge record; it is not there.
- **Treating `Kind == "package"` nodes as files:** See Common Pitfalls below — this is the
  concrete defect this session found and root-caused.
- **`cytoscape-fcose` as a default choice without addressing the force-directed tension
  explicitly:** GRF-02's ban is not a style preference this project holds loosely — it cites
  three independent external sources (dependency-cruiser's FAQ, an MSR paper, SourceTrail's
  own v1→v2 retreat) plus this project's own `Out of Scope` table entry. Choosing fcose
  requires either arguing "constrained-to-compound-boundaries force layout is not what that
  prohibition means" as an explicit, recorded decision, or not choosing fcose.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Hierarchical/compound graph layout | A custom directory-tree-to-position layout algorithm | `cytoscape-elk`'s `layered` algorithm with `hierarchyHandling: 'INCLUDE_CHILDREN'` | Sugiyama-style layered layout with genuine nested-graph support is a well-studied, non-trivial algorithm (edge crossing minimization, layer assignment) — reimplementing it is exactly the "huge effort, high risk" class of work this project's own CLAUDE.md flags for the tree-sitter parser decision |
| Cycle detection over a directed graph | A hand-rolled visited-set walk with ad-hoc back-edge tracking | Tarjan's strongly-connected-components algorithm (standard, ~40 lines, well-understood correctness properties) or a 3-color DFS | Both are standard, textbook algorithms with known O(V+E) complexity and no subtle correctness gaps a hand-rolled "walk and remember what I've seen" version tends to have (e.g., correctly handling a node reachable via multiple disjoint back-edges) |
| Pan/zoom/canvas interaction | Raw `<canvas>` + manual hit-testing for click-to-expand | Cytoscape's built-in `tap`/`move` event system | This is precisely the class of problem D-02 already ruled out hand-rolling for (Sigma's own community grouping plugin author "doesn't really implement that nice grouping behavior" — the cited counter-example of what NOT to attempt) |

**Key insight:** This phase's "don't hand-roll" list is short because D-02/D-04 already did
the hard work of ruling out the two obvious hand-roll traps (a custom layout, a custom
grouping mechanism on top of a non-compound renderer) before this research began. The
remaining hand-roll risk is narrower and more Go-specific: cycle detection, which has no
existing helper anywhere in `internal/query` (confirmed via search this session — no
`Tarjan`/`SCC`/`StronglyConnected` symbol exists in the codebase) and must be written new.

## Common Pitfalls

### Pitfall 1: Synthetic `"package"`-kind pseudo-nodes silently corrupt the file rollup

**What goes wrong:** `Engine.FileGraph()`'s node→file lookup (Scan 1 in Pattern 1 above), if
built naively from every `IterateNodes()` record without excluding `Kind == "package"`,
produces a lookup table where 43 entries (in this repository's own index) map to
`FilePath == ""`. Any edge whose source or target is one of these pseudo-nodes then
"resolves" to file `""` rather than being excluded — corrupting the rollup's node count (a
phantom `""` file appears as a distinct rollup node) and creating a nonsensical edge
endpoint (a real file "importing itself" if two such edges collapse, or an edge dangling to
nothing renderable).

**Why it happens:** `internal/indexer/resolve.go:21-25` defines `kindPackage = "package"` —
*"the synthetic node kind for an intra-module import target... not one of goextract's
declared node kinds, since no source declaration produces it; it exists purely so an
`imports` edge has something in-repo to target."* The construction site
(`internal/indexer/resolve.go:203-215`) builds `&schema.Node{Id: pkgID, Kind: kindPackage,
Name: ..., QualifiedName: ref.Name}` with **no `FilePath` field set at all** — confirmed by
reading the literal this session. This is distinct from `goextract.KindFile`
(`internal/indexer/goextract/types.go:9`) — a real, per-source-file node every extractor
emits — and is *only* minted for imports classified as intra-module (an import of one of
this repository's own packages, not an external/stdlib import, which produces no node at
all per `internal/indexer/resolve.go:196-199`'s `isIntraModule` check).

**How to avoid:** Exclude `n.Kind == "package"` when building the Scan-1 lookup table (shown
in Pattern 1's code example). This also means: an `imports` edge from a Go file to one of its
own sibling internal packages will not appear in the file-level rollup at all under this
exclusion, UNLESS the planner explicitly decides intra-module import edges should roll up to
the *importing file* → *some representation of the target package's directory* — a real
design choice CONTEXT.md's `## Claude's Discretion` list does not explicitly cover and should
be flagged to the planner as a genuine gap, not silently resolved either way by this
research.

**Warning signs:** A rollup node count that is off by exactly the number of `"package"`-kind
entries in `nodesByKind` (visible via `codegraph status --json`); an edge in the rendered
graph whose source or target label renders as an empty string.

**Verified this session (MEASURED, not estimated):** reading this repository's own live
index (`codegraph status --json`) shows `nodesByKind.file: 572` alongside a *separate*
`nodesByKind.package: 43`. A naive rollup scan (run once without the exclusion, then again
with it, both via a throwaway program using this repository's own `internal/graphstore` and
`internal/schema` packages against the live Pebble store) produced:

| | Naive (bug present) | Corrected |
|---|---|---|
| Rollup nodes | 573 (includes 1 phantom `""` entry) | 572 (matches `nodesByKind.file` exactly) |
| Aggregated (src,tgt,kind) edges | 1,540 | 1,326 |
| Distinct (src,tgt) file pairs | 1,271 | 1,057 |

461 edges (imports touching a package pseudo-node) were incorrectly retained in the naive
version. `google/guava` (the corpus GRF-01 will likely measure against — see below) has
**zero** `"package"`-kind nodes in its index, so this defect is entirely invisible on the
scale GRF-01's spike will actually exercise — it will only surface once this project's own
`/graph` route is used against this project's own index, making it exactly the kind of gap a
spike measured against an external corpus alone would miss.

### Pitfall 2: `cytoscape-fcose` is textually force-directed — a real tension with GRF-02, not a false alarm

**What goes wrong:** Choosing `cytoscape-fcose` because it is the best-documented,
most-downloaded compound-aware Cytoscape layout extension, without confronting that its own
README self-describes the algorithm as force-directed.

**Why it happens:** fcose's compound-node support is genuinely excellent and
well-documented (`nestingFactor`, `gravityCompound`, `gravityRangeCompound` options exist
specifically for compound layout tuning) — it is easy to see "supports compounds, widely
used, well-maintained" and stop there.

**How to avoid:** `[CITED: github.com/iVis-at-Bilkent/cytoscape.js-fcose README, fetched
this session]`: *"fCoSE (pron. 'f-cosay', **f**ast **Co**mpound **S**pring **E**mbedder)...
combines the speed of spectral layout with the aesthetics of force-directed layout."* GRF-02
prohibits force-directed rendering explicitly and cites three external sources
(dependency-cruiser's FAQ, an MSR "Trimming the Hairball" paper, and SourceTrail's own
v1→v2 retreat) plus this project's own `REQUIREMENTS.md` `Out of Scope` table entry as the
warrant. `cytoscape-elk`'s `layered` algorithm is the only evaluated option confirmed
non-force-directed by its own documentation (`[CITED: github.com/kieler/elkjs README]`:
"based on the ideas originally introduced by Sugiyama et al.") while also supporting
hierarchy via `elk.hierarchyHandling` — a real, documented, actively-discussed option
(`[CITED: eclipse.dev/elk/reference/options/org-eclipse-elk-hierarchyHandling.html`, and
multiple open `kieler/elkjs` GitHub issues discussing its cross-hierarchy-edge rough edges,
confirmed via WebSearch this session]`).

**Warning signs:** A plan that names `cytoscape-fcose` as the chosen layout without an
explicit sentence addressing why a force-directed algorithm satisfies a "never
force-directed" requirement.

### Pitfall 3: elkjs is a genuinely heavy dependency — budget the bundle-size cost honestly

**What goes wrong:** Treating "smallest gzip size wins the D-01 tie-break" as obviously
favoring `cytoscape-dagre`, without weighing that dagre has no compound support at all
(disqualifying it under D-04's requirement, not merely a tie-break loss).

**Verified this session (MEASURED via Bundlephobia's API, not estimated):**

| Package | Minified | Gzipped | Source |
|---------|----------|---------|--------|
| `cytoscape@3.34.2` (core) | 435,583 B | 136,996 B (~134 KB) | `[VERIFIED: bundlephobia.com/api/size, this session]` |
| `elkjs@0.12.0` (the layout algorithm `cytoscape-elk` wraps) | 1,451,053 B | 433,378 B (~423 KB) | `[VERIFIED: bundlephobia.com/api/size, this session]` — **note:** `cytoscape-elk`'s own dependency range is `elkjs: "^0.9.3"`, so the version actually installed transitively will resolve to latest 0.9.x, not 0.12.0; the true installed size was not separately measured this session (Bundlephobia rate-limited the specific 0.9.3 query) — expect a broadly similar order of magnitude, re-verify at implementation time |
| `dagre@0.8.5` (core, bundled inside `cytoscape-dagre`) | 79,169 B | 24,871 B (~24 KB) | `[VERIFIED: bundlephobia.com/api/size, this session]` |
| `cytoscape-fcose@2.2.0` | not measured this session — Bundlephobia rate-limited every retry | not measured | unpacked size (uncompressed, includes non-shipped files) is 8,681,809 B `[VERIFIED: npm view cytoscape-fcose dist.unpackedSize]` — a weak, non-comparable proxy only; do not use this number for the D-01 tie-break, re-measure gzip size at implementation time |

**Context:** this repository's entire committed `web/build/` is currently 892 KB total, with
~584 KB of JS (`du -sh web/build`, `find web/build -name '*.js' | xargs du -ch`, this
session). Adding `cytoscape` + `cytoscape-elk`/`elkjs` would roughly add cytoscape's 134 KB
gzip plus elk's ~423 KB gzip — i.e., **more than doubling** the current JS payload, in a
project whose own `web:drift` and `//go:embed`-into-a-signed-binary architecture makes bundle
size a real, measured cost (per `.claude/CLAUDE.md`'s own supply-chain framing), not a
throwaway concern. `cytoscape-dagre`'s bundled `dagre` core is ~6x smaller — but its README
documents no compound-node support at all, which is a functional disqualifier under D-04
before bundle size even enters the D-01 tie-break ordering. This is the concrete tradeoff the
spike plan should present to the maintainer explicitly, not resolve silently.

### Pitfall 4: jsdom has no layout engine and no canvas by default — know what is and is not testable

**What goes wrong:** Writing a vitest test that mounts `GraphCanvas.svelte`, expects
Cytoscape to actually render to a `<canvas>`, and either gets a cryptic jsdom
"Not implemented: HTMLCanvasElement.prototype.getContext" warning or silently-zero pan/zoom
behavior because the container's `offsetWidth`/`offsetHeight` are hardcoded to 0.

**Why it happens:** This is the identical class of bug 04-07 already hit and fixed for
`@tanstack/svelte-virtual` (`web/tests/setup.ts:9-44`, read this session): jsdom "does not
implement layout" — every element's `offsetWidth`/`offsetHeight` is 0 unless explicitly
stubbed, and jsdom ships no real `<canvas>` 2D rendering context unless the optional `canvas`
npm package is installed (it is not currently a devDependency — confirmed by reading
`web/package.json` this session, no `canvas` entry present).

**How to avoid — the honest split (see Validation Architecture for the full table):**
- `file-graph-transform.ts` (the pure `FileGraphResponse` → Cytoscape-elements function) has
  zero DOM dependency and is fully unit-testable in plain vitest — this is where the bulk of
  correctness testing should concentrate.
- Cytoscape's `headless: true` mode (`[CITED: cytoscape.js core/init.md]`: *"headless... A
  convenience option that initialises the instance to run headlessly... automatic in Node.js
  environments"*) runs the graph *model* (compound structure, even layout position
  computation, since layouts operate on the data model) without any container or canvas —
  this makes compound-parent assignment and cycle-membership-class logic testable via a
  headless `cy` instance in vitest, without needing the `canvas` npm package or an
  `offsetWidth`/`offsetHeight` stub.
- Actual pixel-level rendering, pan/zoom gesture recognition, and click-to-expand's `tap`
  event wiring depend on a real renderer bound to a real canvas — jsdom cannot exercise this
  even with a scoped `offsetWidth`/`offsetHeight` stub (the `data-table-scroll` precedent),
  because that stub only fixes layout dimensions, not canvas 2D context availability. This
  must be manual browser UAT, following the exact precedent Phase 3/4 established
  (`highlight.js`'s missing theme stylesheet was caught by mandated manual UAT, not any
  grep) — and per the phase's own `<specifics>` note, a live browser check against the built
  binary is explicitly expected before this phase transitions, because Phase 4 already
  proved five machine-verification layers can all pass while a user-visible defect (leaked
  internal planning vocabulary) ships.

## Code Examples

### `Engine.FileGraph()` — see Pattern 1 above for the full two-scan shape.

### Tarjan's SCC over the aggregated file-level adjacency
```go
// New file, internal/query/filegraph_cycles.go — no existing helper in internal/query
// does this (confirmed via search this session: no Tarjan/SCC/StronglyConnected symbol
// exists anywhere in internal/query or internal/indexer today).
//
// Standard iterative or recursive Tarjan's algorithm over a plain
// map[string][]string adjacency (built from the FileGraph edge aggregation's distinct
// (src,tgt) pairs, kind-agnostic for cycle purposes — a cycle exists if ANY kind of
// dependency edge closes a loop, not per-kind). Self-edges are already excluded at
// aggregation time (D-03), so a 1-node SCC of size 1 is never itself "a cycle" — only
// SCCs of size >= 2 are reported as cycles.
```

### GetHealthResponse's `map<string,int64>` precedent for FileGraph's edge kind-counts
```protobuf
// Source: internal/uiproto/uiv1/ui.proto:736 (verified this session — exact existing field)
map<string, int64> edges_by_kind = 11; // on GetHealthResponse

// Recommended for the new FileGraph message shape (Claude's Discretion — field numbers to
// be pinned during the blocking-human checkpoint per D-02a discipline):
message FileGraphEdge {
  string source_file = 1;
  string target_file = 2;
  map<string, int64> kind_counts = 3; // mirrors ui.proto:736's exact precedent
}
```

### Cytoscape headless mode for jsdom-safe unit tests
```javascript
// Source: cytoscape.js core/init.md, Context7, this session
// Runs the graph model (compound structure, layout position computation) without any
// container or canvas — usable in plain vitest without jsdom canvas support.
import cytoscape from 'cytoscape';
const cy = cytoscape({ headless: true, elements: testElements });
// assert cy.$('#some-file').isParent(), cy.$('#some-symbol').parent().id() === 'some-file', etc.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| Whole-repo symbol-level force-directed "hairball" graph as the default landing view | File/package-level aggregated, hierarchically-laid-out graph with drill-down | Industry-wide pattern shift documented in `REQUIREMENTS.md`'s own `Out of Scope` table (dependency-cruiser FAQ, MSR "Trimming the Hairball" paper, SourceTrail's v1→v2 retreat) — not a new finding this session, but the phase's own binding rationale | GRF-02's prohibition is not a stylistic preference; it is this project's own explicit response to a well-documented, repeatedly-observed failure mode |
| `smacker/go-tree-sitter`-style single-vendor-fork dependency selection | Verify against official/current sources per-package, not training-data recall | Standing project discipline (this project's own `.claude/CLAUDE.md` "What NOT to Use" table) | Directly informed this research's insistence on live `npm view`/Bundlephobia/Context7 checks over recalling package names and sizes from training data |

**Deprecated/outdated:** None of the three Cytoscape layout extensions evaluated here are
themselves deprecated — all three (`cytoscape-elk`, `cytoscape-dagre`, `cytoscape-fcose`) are
under active maintenance by their respective organizations as of this session's checks.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Directory-structural grouping (GRF-02) should be derived client-side from `FilePath` splitting rather than the server emitting an explicit directory tree | Architectural Responsibility Map, Recommended Project Structure | If wrong, `Engine.FileGraph()`'s wire shape needs an explicit directory/tree field, changing ENG-03's return shape and the proto message design — a real design decision the planner should confirm, not silently inherit from this assumption |
| A2 | Intra-module `imports` edges (the ones that would otherwise target a `"package"`-kind pseudo-node) should be EXCLUDED from the file rollup entirely, rather than represented as a special "imports the internal/foo package" edge to some synthetic directory-level target | Pitfall 1 | If wrong, GRF-02's edge picture is missing a real category of dependency (intra-module imports) that a developer might expect to see; the planner should decide this explicitly rather than have it fall out of an unstated default |
| A3 | The elkjs version actually resolved by `pnpm add cytoscape-elk` (range `^0.9.3`) has bundle size in the same order of magnitude as the 0.12.0 figure measured this session (~423 KB gzip) | Pitfall 3 | If the 0.9.x line is meaningfully smaller/larger, the D-01 bundle-size tie-break input changes; cheap to re-verify at implementation time (`npm view elkjs@^0.9.3 version` then re-run Bundlephobia) |
| A4 | `@types/cytoscape-elk` does not exist on the npm registry (checked only `cytoscape-elk`'s own `types` field, not a separately-scoped `@types/*` package) | Package Legitimacy Audit | Low risk — worst case is a small ambient-module type shim is written that turns out to duplicate an existing `@types` package; easy to discover and discard at implementation time |

## Open Questions

1. **Should GRF-01's locked threshold be measured against the fully-expanded file-level
   scale (3,233 nodes / 21,554 edges for guava) or the initial collapsed directory-level
   scale (134 top-level compound nodes / 819 directory-pair edges for guava, computed this
   session from the same live index)?**
   - What we know: D-01's text says "the file/package rollup of the largest available
     corpus", and GRF-02 mandates directory-structural grouping as the default view — meaning
     a developer's actual first paint is likely the collapsed, ~134-node view, with the
     3,233-node scale only reachable by manually expanding every directory.
   - What's unclear: whether the interaction-latency threshold should be judged at "typical
     first paint" (much smaller, near-certain pass) or "worst case, everything expanded"
     (the real stress case, and the one D-02's compound-node prerequisite genuinely exists to
     serve).
   - Recommendation: the spike plan should measure BOTH and record both numbers against the
     locked threshold — presenting only the easy collapsed-view number as "the" measurement
     would defeat the point of locking a real interaction-latency bar before dispatch.

2. **Does `Engine.FileGraph()`'s response, at guava scale (up to ~26,355 aggregated
   `(src,tgt,kind)` tuples if the wire shape keeps kind un-collapsed, or 21,554 if
   pre-collapsed into per-edge kind-count maps as recommended), fit comfortably inside the
   existing 16 MB `transportSendMaxBytes` ceiling (`internal/uiserver/truncate.go:68`,
   verified this session)?**
   - What we know: a rough per-edge estimate (two file-path strings plus a small kind-count
     map, likely 150–300 bytes serialized) times 21,554 plus 3,233 node records puts a worst
     case around 5–6 MB — comfortably under 16 MB, but not measured directly against the
     actual generated protobuf message this session.
   - What's unclear: whether repeated long repo-relative paths (Java package paths can be
     deep) push the real number meaningfully higher, and whether this repo's own
     `fileCount`/`nodesByKind.file` discrepancy (573 vs 572, see below) signals any other
     latent counting inconsistency worth checking before trusting a byte estimate.
   - Recommendation: measure the actual serialized `FileGraphResponse` size against guava
     once `Engine.FileGraph()` exists, rather than trusting this estimate.

3. **Minor, unresolved discrepancy found this session, unrelated to the package-pseudo-node
   defect: this repository's own live `codegraph status --json` reports `fileCount: 573` but
   `nodesByKind.file: 572` — a 1-record disagreement between the `schema.File` namespace
   (`IterateFiles()`) and the `schema.Node{Kind:"file"}` namespace (`IterateNodes()`)
   `[VERIFIED: codegraph status --json, this session]`.**
   - What we know: both numbers were read directly from the same live index this session; the
     discrepancy is real, not a research artifact.
   - What's unclear: the root cause — plausibly one file that failed extraction
     (`goextract.Extract` records `FileResult.Err` and still returns a result) got a
     `schema.File` record written without a corresponding `KindFile` node, but this was not
     traced to a specific file or line this session.
   - Recommendation: `Engine.FileGraph()`'s rollup node count should be derived from
     `nodesByKind.file` (i.e., actual `KindFile`-kind `Node` records, since those are what
     participate in the edge rollup), not from `fileCount`/`IterateFiles()` — this is already
     what Pattern 1's Scan 1 does by construction (it scans `IterateNodes()`, not
     `IterateFiles()`), so no code change is implied, but the planner should not be surprised
     if `FileGraph()`'s reported node count doesn't match `GetHealth`'s `file_count` field by
     exactly 1.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| `pnpm` | Installing `cytoscape`/`cytoscape-elk` | ✓ | pinned `pnpm@11.23.0` via Corepack (`web/package.json`'s `packageManager` field, unchanged this phase) | — |
| Go toolchain | `Engine.FileGraph()`, cycle detection | ✓ | `GOTOOLCHAIN=go1.26.5` (per project standing constraint; confirmed a plain `go build ./cmd/codegraph` succeeds under this pin this session) | — |
| A large indexed real-world corpus for GRF-01's spike | GRF-01 | ✓ — **already indexed, not merely checked out.** `google/guava` at its pinned SHA (`corpora/manifest.json`) has a live, complete Pebble index at `~/.cache/codegraph/corpora/google-guava-2b0cb53f@94f39958baf7ad51ddf9c70e406ed6b188194daa/.codegraph/store/` on this machine, confirmed this session by opening it directly and by running `codegraph status --json` against it (`fileCount: 3233, nodeCount: 59874, edgeCount: 155080`, matching `corpora/observations.json`'s recorded snapshot to the field). All 8 measured corpora (all but the deliberately-rejected `apache/arrow`) have local checkouts under `~/.cache/codegraph/corpora/`. — | Re-running `codegraph init <path>` against any corpus takes ~1.5s wall clock for guava (measured this session) — reindexing, if ever needed (e.g., after a schema change), is cheap, not a scheduling risk |
| Bundlephobia's public API (for bundle-size verification) | Confirming layout-extension gzip sizes | Partial — aggressively rate-limited this session (`429 Too Many Requests` on repeated calls); `cytoscape`, `elkjs`, and `dagre` core were successfully measured, `cytoscape-fcose` was not | Re-attempt at implementation time with pacing, or measure directly via the project's own `pnpm build` + its existing `WEB_HASH_LIB` size-tracking pipeline (`Taskfile.yml`), which is the more authoritative number for this project's actual `web:drift`/embed-size concern anyway |
| `canvas` npm package (for real canvas rendering in jsdom/vitest) | Full-fidelity Cytoscape rendering tests | ✗ — not a devDependency (`web/package.json`, checked this session) | — | Not needed: the recommended test split (Pitfall 4) keeps canvas-dependent behavior out of jsdom entirely, deferring to manual browser UAT instead of installing `canvas` |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** Bundlephobia rate-limiting (re-measure at
implementation time or via the project's own build pipeline); no `canvas` npm package
(deliberately not needed given the recommended test split).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | vitest 4.1.11 + @testing-library/svelte 5.4.2 (frontend, `web/package.json`, verified this session — unchanged from Phase 4); Go stdlib `testing` (backend) |
| Config file | `web/vite.config.ts` (test block); `web/tests/setup.ts` (global jsdom stubs, extended this phase per Pitfall 4 if a `file-graph-canvas` testid stub is needed) |
| Quick run command | `cd web && pnpm test -- file-graph` (frontend); `GOTOOLCHAIN=go1.26.5 go test ./internal/query/... -run FileGraph` (backend) |
| Full suite command | `cd web && pnpm test`; `GOTOOLCHAIN=go1.26.5 go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ENG-03 | `Engine.FileGraph()` aggregates correctly, excludes `contains` and self-edges, excludes `"package"`-kind pseudo-nodes (Pitfall 1) | unit (Go) | `go test ./internal/query/... -run TestFileGraph` | ❌ Wave 0 — new file `internal/query/filegraph_test.go` |
| GRF-04 | Cycle detection finds real SCCs, does not flag a 1-node non-cycle | unit (Go) | `go test ./internal/query/... -run TestFileGraphCycles` | ❌ Wave 0 — new file `internal/query/filegraph_cycles_test.go` |
| RPC surface (the 12th rpc) | `FileGraph` name is clean against all 19 `mutatingVerbs`, method count updates 11→12 | unit (Go, existing fixture pattern) | `go test ./internal/uiserver/... -run TestUIService` | ✅ — extend `internal/uiserver/readonly_test.go`'s existing `wantUIServiceMethods` map and count literal |
| GRF-02 | `file-graph-transform.ts` correctly builds compound `parent` chains from `FilePath` strings, never emits force-directed layout config | unit (vitest, DOM-free) | `pnpm test -- file-graph-transform` | ❌ Wave 0 — new file `web/tests/file-graph-transform.test.ts` |
| GRF-04 (client) | Cycle-membership from the wire response maps to a distinguishing CSS class on the correct nodes | unit (vitest, Cytoscape `headless: true`, DOM-free per Pitfall 4) | `pnpm test -- graph-canvas` | ❌ Wave 0 — new file `web/tests/graph-canvas.test.ts` |
| GRF-03 (interaction), GRF-02 (rendered layout, pan/zoom) | Click-to-expand actually works; the graph is visually hierarchical, not a hairball | **manual-only** — jsdom cannot exercise real canvas rendering or `tap` gesture recognition (Pitfall 4) | none — browser UAT against the built binary, per the phase's own `<specifics>` mandate | N/A |
| The `/graph` placeholder text | `web/src/routes/graph/+page.svelte:9`'s "Phase 5:" line is replaced | manual UAT (grep alone insufficient — Phase 4's precedent) | `rg -i "phase 5" web/src/routes/graph/` as a cheap pre-check, but not a substitute for viewing the rendered page | ❌ — no existing guard against leaked planning vocabulary in rendered copy anywhere in this codebase (confirmed: Phase 4 shipped the identical defect on `/workbench` past five verification layers) |

### Sampling Rate

- **Per task commit:** `pnpm test -- <touched-file-pattern>` (frontend); `go test
  ./internal/query/... -run <TestName>` (backend) — scoped, fast
- **Per wave merge:** `pnpm test` (full frontend suite); `go test ./...` (full backend
  suite)
- **Phase gate:** Full suite green, PLUS mandatory manual browser UAT against the built
  binary before `/gsd-verify-work` — per this phase's own `<specifics>` note that Phase 4
  proved machine verification alone is categorically insufficient for a rendered-copy defect
  of this exact class.

### Wave 0 Gaps

- [ ] `internal/query/filegraph_test.go` — covers ENG-03 (aggregation correctness, `contains`
      exclusion, self-edge exclusion, `"package"`-pseudo-node exclusion per Pitfall 1)
- [ ] `internal/query/filegraph_cycles_test.go` — covers GRF-04's server-side detection
- [ ] `web/tests/file-graph-transform.test.ts` — covers GRF-02's compound-parent construction,
      DOM-free
- [ ] `web/tests/graph-canvas.test.ts` — covers GRF-04's cycle-class assignment via headless
      Cytoscape, DOM-free
- [ ] Possible `web/tests/setup.ts` extension — a scoped `offsetWidth`/`offsetHeight` stub
      keyed to a `file-graph-canvas` testid, mirroring the existing `data-table-scroll` stub,
      ONLY if any non-headless rendering test proves necessary (Pitfall 4 recommends avoiding
      this need entirely by keeping canvas-dependent assertions out of jsdom)
- [ ] No test framework install needed — vitest/Go stdlib are both already fully configured

## Security Domain

### Applicable ASVS Categories

Project config: `security_enforcement: true`, `security_asvs_level: 1`
(`.planning/config.json`, read this session).

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Local, loopback-only, read-only UI (SRV-01/02/03) — no auth concept exists or is being added by this phase |
| V3 Session Management | No | No session state introduced |
| V4 Access Control | No | No new access-control surface; `FileGraph` reads the same single-repository `Engine` every other rpc reads |
| V5 Input Validation | Yes | `FileGraphRequest` (per the pattern used by `GetHealthRequest`) likely carries no meaningful caller-supplied parameters in v1 (a whole-repo rollup has no natural "which file" input) — if any filter/depth parameter is added, it must go through the same `query.Validate*` pre-check pattern `Affected`'s handler uses (`internal/uiserver/handlers.go:541-560`, read this session) BEFORE `withEngine` opens the store, not after |
| V6 Cryptography | No | Not applicable — no cryptographic operation in this phase |
| V12 (API/response size) | Yes | The existing `connect.WithSendMaxBytes(transportSendMaxBytes)` = 16 MB ceiling (`internal/uiserver/server.go:123`, `internal/uiserver/truncate.go:68`, read this session) already bounds `FileGraph`'s response — Open Question 2 above flags that this has not been measured directly against the actual generated message at guava scale, only estimated |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| DNS rebinding against the loopback-bound UI server | Spoofing | Already mitigated project-wide by SRV-02's exact-match Origin/Host guard (`internal/uiserver/originguard.go`, Phase 1) — this phase adds no new listener, no new mitigation needed |
| Unbounded response growth from an unfiltered whole-repo scan | Denial of Service | `FileGraph` has no caller-controlled fan-out parameter (unlike `Affected`'s `files` list, which needed `query.ValidateAffectedFiles`) — the "unboundedness" is bounded by the repository's own fixed size, not by attacker input, and is further capped by the existing 16 MB send ceiling. No new validation function is needed unless a future filter parameter is added |
| Path traversal via a caller-supplied file path | Tampering | Not applicable in v1 — `FileGraph` returns the whole repository's rollup with no caller-supplied path parameter to confine. If GRF-03's in-place expansion is ever wired to a NEW server call taking a specific file path (rather than reusing the existing `GetNodeDetail`/`Callers`/`Callees` rpcs, as this research recommends), that new call would need the SAME `ValidateRepoRelativePath` confinement `GetNodeDetail`/`GetPermalink` already use (SRV-05) — not a second implementation |

## Sources

### Primary (HIGH confidence — MEASURED this session against live systems)

- This repository's own live index, read via `internal/graphstore`/`internal/schema`
  directly (a throwaway program built and run against `.codegraph/store/`, then deleted —
  not committed) — file/edge rollup counts, the `"package"`-pseudo-node discrepancy, the
  `fileCount`/`nodesByKind.file` discrepancy
- `google/guava`'s pre-existing local index at
  `~/.cache/codegraph/corpora/google-guava-2b0cb53f@…/.codegraph/store/` — same method,
  confirmed against `corpora/observations.json`'s independently-recorded snapshot
- `npm view <pkg> version dist-tags time.created dist.unpackedSize scripts.postinstall
  dependencies peerDependencies` — run live for `cytoscape`, `cytoscape-elk`, `elkjs`,
  `cytoscape-dagre`, `cytoscape-fcose`, `cose-base`, `@types/cytoscape`
- `bundlephobia.com/api/size` — run live for `cytoscape@3.34.2`, `elkjs@0.12.0`,
  `dagre@0.8.5` (rate-limited for `cytoscape-fcose`, not obtained)
- `gsd_run query package-legitimacy check --ecosystem npm cytoscape cytoscape-elk elkjs
  cytoscape-dagre cytoscape-fcose` — verbatim verdicts
- `internal/schema/graph.pb.go:41-68` (Node struct), `:214-236` (Edge struct) — read this
  session, quoted verbatim in Pattern 1
- `internal/indexer/nodeid/nodeid.go:43`, `nodeid_test.go:8-33` — read this session, node ID
  content-hash shape confirmed
- `internal/indexer/resolve.go:21-25, 185-217` — read this session, `kindPackage` pseudo-node
  construction, quoted verbatim in Pitfall 1
- `internal/query/traverse.go:1-60` — `BuildReverseAdjacency`, read this session
- `internal/query/status.go:1-70, 190-240` — `edgesByKind`'s full-scan precedent, read this
  session
- `internal/uiserver/readonly_test.go:1-125` — `wantUIServiceMethods`, `mutatingVerbs` (all
  19 entries), method-count literal, read this session; candidate rpc names checked against
  all 19 substrings programmatically
- `internal/uiproto/uiv1/ui.proto:1-40, 50-80, 628-716, 736` — service block, `GetHealth`
  message shape, `edges_by_kind` map field precedent, read this session
- `web/tests/setup.ts:1-45` — the `data-table-scroll` jsdom stub precedent, read this session
- `web/src/lib/components/workbench/DataTable.svelte:95-123` — `$effect`-based imperative
  library mounting precedent, read this session
- `web/src/routes/+layout.ts:5` — `ssr = false`, read this session
- `web/src/routes/graph/+page.svelte` — the placeholder text to be replaced, read this
  session
- `internal/uiserver/server.go:116-124`, `truncate.go:57-76` — `transportSendMaxBytes`/
  `transportReadMaxBytes`, read this session
- `corpora/manifest.json`, `corpora/observations.json`, `corpora/selection.json` — read this
  session, cross-checked against the live guava index

### Secondary (MEDIUM confidence — official docs, Context7/WebFetch this session)

- Cytoscape.js official docs (`cytoscape/cytoscape.js`), via Context7 — compound-node
  `parent` field semantics, `eles.move()`, `cy.destroy()`, `cy.resize()`, `headless`/
  `styleEnabled` options
- `github.com/iVis-at-Bilkent/cytoscape.js-fcose` README — force-directed self-description,
  compound-layout options, fetched via WebFetch this session
- `github.com/cytoscape/cytoscape.js-dagre` README — hierarchical/DAG algorithm description,
  no documented compound support, fetched via WebFetch this session
- `github.com/kieler/elkjs` README — Sugiyama-style layered algorithm confirmation, fetched
  via WebFetch this session
- `eclipse.dev/elk/reference/options/org-eclipse-elk-hierarchyHandling.html` and multiple
  open `kieler/elkjs` GitHub issues — `hierarchyHandling` option and its documented rough
  edges, via WebSearch this session

### Tertiary (LOW confidence)

- None used as the basis for any load-bearing claim in this document.

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM-HIGH — the renderer's API surface is confirmed live; the specific
  layout-extension CHOICE (elk vs. dagre vs. fcose) is deliberately NOT made here, since
  that is GRF-01's own job, and this research surfaces a genuine three-way tension
  (compound support vs. force-directed prohibition vs. bundle size) rather than resolving it
- Architecture: HIGH — the two-scan requirement, the wire-shape precedent, and the
  component-seam pattern are all grounded in this session's direct reads of the actual Go
  structs, existing proto messages, and existing Svelte components, not inference
- Pitfalls: HIGH — Pitfall 1 (package pseudo-nodes) and the `fileCount`/`nodesByKind.file`
  discrepancy in Open Question 3 are both novel findings this session, verified against live
  data, not carried over from CONTEXT.md or training knowledge

**Research date:** 2026-08-30
**Valid until:** 2026-09-13 (14 days — npm package versions and Bundlephobia sizes move
faster than this project's usual 30-day estimate would assume; re-verify exact `elkjs`
0.9.x size and `cytoscape-fcose` gzip size at implementation time regardless of date, since
both were incomplete this session)
