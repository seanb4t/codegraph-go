# Phase 11: Graph View — Community Clustering - Research

**Researched:** 2026-09-13
**Domain:** Deterministic graph community detection (gonum Louvain) wired additively into an existing Go query engine + ConnectRPC + Svelte/Cytoscape stack
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

Verbatim from `.planning/phases/11-graph-view-community-clustering/11-CONTEXT.md`:

- **D-01:** Community detection is Louvain modularity via `gonum.org/v1/gonum/graph/community.Modularize` (already pinned at v0.17.0 in `go.sum` transitively via sigstore; promoted to a direct require, D-12). Input is an undirected weighted graph built from `FileGraph()`'s aggregated file-pair edges, weight = `FileGraphEdge.TotalCount`, resolution `1.0`. The documented zero-dependency fallback is a hand-rolled label-propagation implementation, used only if gonum's Louvain proves non-deterministic under D-02's sorted input or GRF-10's review rejects gonum.
- **D-02:** Determinism is guaranteed by three mechanisms, all required: (a) nodes inserted in sorted file-path order with sequential `int64` IDs; (b) a fixed seed (`rand.NewSource(<committed constant>)`) driving Louvain's move ordering; (c) canonical relabeling after the run — communities renumbered `1..N` by smallest member path.
- **D-03:** Every file node gets exactly one community (a singleton is its own community). IDs are 1-based canonical; `0` means "not computed" and is never emitted on the fresh-compute path.
- **D-04:** GRF-08's proof: a test runs the clustering ≥3 times on identical input, asserts label-canonicalized equality, and reports the run count; plus a RED control — perturbing node insertion order or the seed must turn the test red, recorded in `11-MUTATION-LOG.md`.
- **D-05:** A new `corpora/graph-cluster-threshold.json` is committed alone as the phase's first commit, mirroring `corpora/graph-render-threshold.json`. The verdict script never writes it; amending/widening/relaxing any value after measurement is forbidden.
- **D-06:** The binding metric is clustering time in isolation, from `FileGraph()`'s rollup complete to community assignment complete, measured in-process by a Go harness against the pinned guava corpus, median of 3 cold runs, max 500 ms.
- **D-07:** `onFailure` is written into the threshold file before measuring: index-time persistence into `Node`'s reserved 50-59 range (`community_id = 50`), computed at commit time and read back by `FileGraph()`.
- **D-08:** The measurement lands as `corpora/graph-cluster-observations.json` plus a verdict artifact whose PASS/FAIL is preserved verbatim whichever way it falls; git ancestry is checked by a test, not asserted in prose. No clustering may be wired into the UI before this verdict resolves.
- **D-09:** Additive wire: `FileGraphNode` gains `int32 community_id = 5`; `FileGraphResponse` gains `int32 community_count = 7`. No other proto change; field-number fixture and `wantUIServiceMethods` (unchanged at 16) updated in the same commit as the proto.
- **D-10:** Rendering = node background colour keyed by `community_id` from a deterministic categorical palette (colour-blind-safe, ~12 hues, index = `(id-1) mod 12`, cycling beyond), applied on the existing ELK layered layout untouched; existing `cycleId` group styling stays as-is and composes with the colour.
- **D-11:** Collapsed directory compound nodes stay neutral (border only) — only file-level nodes are coloured.
- **D-12:** The GRF-06 check has three parts: (a) a toolbar/legend line "N communities" fed by `community_count`; (b) a positive-controlled source scan over `web/src` finding ≥1 `elk` layout reference and zero references to any force-directed layout name (`cose`, `fcose`, `cola`, `euler`, `spread`, `force`); (c) a vitest rendering the transform against a fixture asserting distinct colours == distinct community ids.
- **D-13:** `gonum` is promoted from transitive to a direct `require gonum.org/v1/gonum v0.17.0` in `go.mod` — the version already pinned in `go.sum`; the `go mod tidy` diff is reviewed to be that one line.
- **D-14:** GRF-10 is a Taskfile target (e.g. `check:gonum`) with three positive-controlled halves: `govulncheck ./...` over the main module (must report gonum in the scanned set); an SBOM generated locally the way the release does (syft/cyclonedx-gomod) grepped for `gonum.org/v1/gonum` with a positive control; a cgo-closure scan via `go list -deps -f` over `gonum.org/v1/gonum/graph/community`'s import closure reporting the number of packages inspected (>0) and asserting zero `CgoFiles`/`import "C"`.
- **D-15:** Recompute cadence: fresh on every `FileGraph()` call — no caching in this phase. The 50-59 fallback (D-07) is the only persistence path, and only if GRF-09 fails.
- **D-16:** Phase artifacts in the house format: `11-MUTATION-LOG.md` and `11-SECURITY.md`, every row test-or-verdict, `threats_open` honest.

### Claude's Discretion

Verbatim from CONTEXT.md:

- The committed seed constant, the exact palette values (pick a colour-blind-safe categorical set; the `dataviz` skill's palette is acceptable), and where the "N communities" line sits in the graph toolbar.
- Whether the Go measurement harness is a `go test -run` with a build tag (like `test/tmux`) or a `tools/` command; it must be re-runnable and must read the threshold file rather than carrying defaults.
- The name of the GRF-10 Taskfile target and whether the SBOM half uses syft or cyclonedx-gomod (match what `.goreleaser.yaml` uses).
- Whether `community_count` counts singletons (recommend yes — every node is in exactly one community, so N = number of distinct ids).

### Deferred Ideas (OUT OF SCOPE)

Verbatim from CONTEXT.md:

- Colouring collapsed directory compounds by majority community (D-11 keeps them neutral) — revisit only with a UI decision about mixed-community directories.
- An in-process clustering cache keyed by index generation (D-15 rejects it for this phase; the `coverage_generation` counter from Phase 10 would be the obvious key if it is ever wanted).
- A visible colour → community legend beyond the count line.
- Persisted cluster assignments as the default (out of scope by construction; fallback only).
</user_constraints>

## Project Constraints (from CLAUDE.md)

Extracted from `./.claude/CLAUDE.md` (project instructions), as directives this phase's plan
must honor:

- **Tech stack:** Go (latest stable — but this project pins `go 1.26.6` in `go.mod`; local
  toolchain is 1.27.1, so every `go` command in this phase MUST be prefixed
  `GOTOOLCHAIN=go1.26.6`), single static binary per platform.
- **Supply chain:** Minimal, audited dependencies; CGo only for tree-sitter parsing, with an
  explicit documented exception. `gonum` is this milestone's one sanctioned new direct
  dependency and its import closure MUST be cgo-free — verified this session (see Standard
  Stack / GRF-10 findings).
- **Compatibility:** No external functionality-baseline constraint remains for this project
  (retired 2026-08-13) — this phase's behavior is judged against its own requirements
  (GRF-06/08/09/10) and its own frozen goldens/tests, not another implementation.
- **Architecture:** v1 storage/process design must accommodate future team-scale features
  without a rewrite — not directly implicated by this phase (no new storage format is
  introduced; the reserved-field fallback (D-07) already anticipated cluster assignment at
  schema-design time).
- **Licensing:** MIT; attribution lives only in `NOTICE`, never restated elsewhere — not
  directly implicated by this phase (no new attribution-bearing dependency; gonum is BSD-style
  licensed per its own LICENSE file, compatible with this project's MIT licensing, not
  independently re-verified this session).
- **GSD workflow enforcement:** File-changing tool use for this phase's actual implementation
  must go through `/gsd-execute-phase` (or `/gsd-quick`/`/gsd-debug` for smaller detours), not
  direct ad-hoc edits — this research phase itself performed no production file edits (the one
  incidental `go.mod` mutation from a `GOFLAGS=-mod=mod` probe was reverted with `git checkout
  -- go.mod`, confirmed clean via `git status --porcelain`).

## Summary

Phase 11 adds exactly one new computed field to an existing, already-shipped pipeline
(`Engine.FileGraph()` → `ui.proto` → `file-graph-transform.ts` → `graph-style.ts` →
`GraphCanvas.svelte`), following the `CycleID`/cycle-detector precedent line-for-line.
`gonum.org/v1/gonum v0.17.0` is already resolved in `go.sum` (confirmed: `go list -m all`
shows it present) and its `graph/community` package's full transitive closure — 91 packages
including `mat`, `lapack`, `blas/gonum`, `blas/blas64` — was verified this session to contain
**zero** `.CgoFiles` (`go list -deps -f '{{.ImportPath}} {{len .CgoFiles}}'`, every row `0`) and
never reaches `gonum.org/v1/gonum/blas/cgo`. Promoting it from indirect to direct is a
one-line `go.mod` diff.

`gonum.community.Modularize`'s only source of randomness is a Fisher-Yates shuffle of the
node-processing order inside the local-moving heuristic, driven by a `math/rand/v2.Source`
(NOT `math/rand` or `x/exp/rand` — the package imports `"math/rand/v2"` directly). Critically,
`simple.WeightedUndirectedGraph`'s `Nodes()` iterates a Go `map[int64]graph.Node` — genuinely
map-order-dependent — but gonum's own `reduceUndirected` (called first, unconditionally, inside
`louvainUndirected`) re-sorts that node list via `internal/order.ByID` before establishing the
dense 0..n-1 community indices every subsequent step operates on. This means Louvain's
determinism does **not** depend on defeating `simple`'s map iteration by insertion trickery —
it is already neutralized by gonum's own sort-by-ID step, as long as node IDs are assigned
from a canonical, deterministic scheme (CONTEXT D-02a's sorted-file-path + sequential-int64-ID
insertion). Given a fixed `rand.Source` (e.g. `rand.NewPCG(seed1, seed2)`) and canonical IDs,
`Modularize` is bit-for-bit reproducible across runs in this single-threaded, non-concurrent
algorithm.

`FileGraph()` already produces `result.Nodes` sorted ascending by path (`sort.Strings(paths)`
at `internal/query/traverse.go`) and already runs one fresh-per-call graph algorithm
(`stronglyConnectedCycles`, in the sibling file `internal/query/filegraph_cycles.go`) whose
own doc comments and iterative, no-recursion, sorted-input design are the exact pattern to
mirror for a new `community.go` in the same package. The wire slice is two new field numbers,
both free and verified against the live `.proto`: `FileGraphNode.community_id = 5` (after
`cycle_id = 4`) and `FileGraphResponse.community_count = 7` (after `cycle_count = 6`); no new
rpc, so `wantUIServiceMethods` stays at 16. The UI slice is a new peer field on
`FileGraphNodeData` next to `cycleId`, a new set of 12 static per-community-index Cytoscape
style rules (there is no existing `data()`-driven categorical colour mapper in this codebase —
`graph-style.ts` only uses `mapData` for numeric edge width — so the reusable pattern is the
existing `cycleDiscriminatorClass`-style per-id class generator, extended with real distinct
colours instead of a single shared boundary style), and a toolbar line mirroring the existing
"N dependency cycles" paragraph in `+page.svelte`.

**Primary recommendation:** Implement `community.go` in `internal/query` as a pure function
`assignCommunities(nodes []FileGraphNode, edges []FileGraphEdge, seed1, seed2 uint64) map[string]int`
mirroring `filegraph_cycles.go`'s shape exactly — build a `simple.WeightedUndirectedGraph` with
nodes inserted in sorted-path order and sequential `int64` IDs, weight = `TotalCount`, call
`community.Modularize(g, 1.0, rand.NewPCG(seed1, seed2))`, canonically relabel the returned
`Communities()` by smallest member path, and wire the result into `FileGraph()` exactly where
`CycleID` is populated today. Commit `corpora/graph-cluster-threshold.json` alone, first,
before any of this exists, following `corpora/graph-render-threshold.json`'s exact shape.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Community detection (Louvain) | API / Backend (`internal/query`) | — | Must be correctness-independent of the renderer (mirrors GRF-04's cycle-detection rationale, D-06 in 11-CONTEXT.md); computed fresh per `FileGraph()` call, no cache |
| Wire projection of `community_id`/`community_count` | API / Backend (`internal/uiserver`) | — | Pure field-for-field mapping, same convention as `fileGraphNodeToProto`/`fileGraphToProto` |
| Node colouring by community | Browser / Client (Cytoscape stylesheet) | — | Presentation-only; the client never computes membership, only paints wire-supplied ids (mirrors `cycleId` handling in `file-graph-transform.ts`) |
| "N communities" toolbar line | Browser / Client (`+page.svelte`) | — | Reads `communityCount` off the already-fetched response, no new RPC |
| Clustering-time measurement harness | API / Backend (Go, `tools/` or `test/` build tag) | — | The binding metric is Go in-process time; a browser/Playwright harness (GRF-01's `graph-measure.mjs`) measures the wrong thing here (D-06 in 11-CONTEXT.md is explicit: clustering time in isolation, not page latency) |
| Threshold/verdict artifacts | Repo root (`corpora/`) | — | Same tier as `corpora/graph-render-threshold.json` — committed data, not code |
| Supply-chain checks (govulncheck/SBOM/cgo scan) | CI / Taskfile | — | New `check:gonum`-style target, parallel to the existing `vuln:` target, but scoped to the MAIN module (source mode), matching ci.yml's existing blocking `govulncheck` job rather than the tool-binary `vuln:` target |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GRF-06 | File/package graph colours nodes by community, computed fresh inside `FileGraph()`, on the existing layered layout, never force-directed | Confirmed `FileGraph()`'s exact insertion point for a new `assignCommunities` pass; confirmed `GraphCanvas.svelte` registers only `name: 'elk'` with no other layout extension imported; confirmed the additive proto fields are free (5, 7) |
| GRF-08 | Deterministic community assignment (sorted iteration, fixed seed, canonical relabeling), proven ≥3 runs equal | Traced gonum's exact randomness entry point (Fisher-Yates shuffle via `math/rand/v2.Source`) and confirmed `reduceUndirected`'s `order.ByID` sort neutralizes `simple.WeightedUndirectedGraph`'s map-order `Nodes()`; `filegraph_cycles_test.go`'s `TestFileGraphCyclesDeterministicIds` is the exact test shape to mirror |
| GRF-09 | Clustering-time threshold committed before measurement (GRF-01 protocol), fallback into reserved 50-59 on FAIL | Read `corpora/graph-render-threshold.json` and `05-CONTEXT.md`/`05-04-SUMMARY.md` in full; confirmed the ancestry-proof mechanism (`git merge-base` one-liner) that GRF-01 used and that GRF-09 must instead make a persisted, automated check (D-08's "checked by a test, not asserted in prose") |
| GRF-10 | `gonum.org/v1/gonum` passes govulncheck, appears in SBOM, cgo-free import closure | Ran the exact `go list -deps -f '{{.ImportPath}} {{len .CgoFiles}}' gonum.org/v1/gonum/graph/community` command live: 91 packages, zero cgo, `blas/cgo` never reached; confirmed `syft` and `govulncheck` are both installed locally; read the existing `vuln:`/`govulncheck` CI job split (tool-modfile advisory scan vs. main-module blocking scan) that GRF-10's new target must not duplicate or confuse |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `gonum.org/v1/gonum` | v0.17.0 `[VERIFIED: go.sum:537-538, go list -m all]` | `graph/community.Modularize` — Louvain modularity community detection | Already resolved in this module's build list at exactly the version CONTEXT.md's D-01 states; no version bump, no new module |

### Supporting

None — this phase adds a single dependency promotion (indirect → direct), not a new library.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gonum Louvain | Hand-rolled label-propagation | CONTEXT.md D-01's documented fallback, used ONLY if gonum's determinism cannot be proven under D-02's mechanisms or GRF-10's review rejects gonum outright — not a live alternative under normal conditions |

**Installation:**

```bash
# D-13: promote from indirect to direct — the entire go.mod diff is one line
# (comment removed from the existing `// indirect` require line). Confirmed
# this session with GOFLAGS=-mod=mod (reverted before finishing):
#   gonum.org/v1/gonum v0.17.0 // indirect  ->  gonum.org/v1/gonum v0.17.0
GOTOOLCHAIN=go1.26.6 go mod tidy   # or hand-edit go.mod per Phase 4's doublestar precedent
                                    # if `go mod tidy` is blocked by the pre-existing
                                    # tree-sitter-swift resolution failure noted in STATE.md
```

**Version verification:** `[VERIFIED: go.sum:537-538]`

```
gonum.org/v1/gonum v0.17.0 h1:VbpOemQlsSMrYmn7T2OUvQ4dqxQXU+ouZFQsZOx50z4=
gonum.org/v1/gonum v0.17.0/go.mod h1:El3tOrEuMpv2UdMrbNlKEh9vd86bmQ6vqIcDwxEOc1E=
```

`go list -m gonum.org/v1/gonum` confirms `gonum.org/v1/gonum v0.17.0` resolves cleanly today,
with no `go.mod` edit required to observe it (it is already in the build list transitively).
No newer version was checked against upstream — CONTEXT.md D-13 pins this exact version
deliberately ("no version bump"), so this research did not query gonum's release history.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `gonum.org/v1/gonum` | Go module proxy | Long-established (gonum project, active since ~2013; this session did not re-run the package-legitimacy seam tool, but the package is already resolved in `go.sum` transitively and is the org-maintained numerical/graph library for Go) | Not queried this session (no npm-style download metric for Go modules) | `github.com/gonum/gonum` | Not re-run via `gsd-tools query package-legitimacy check` this session — `[ASSUMED]` pending that seam call at plan time | Approved by maintainer decision already recorded in REQUIREMENTS.md/CONTEXT.md as "the milestone's only new direct require" |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none — `gonum.org/v1/gonum` is already present in this
project's dependency graph at the exact version pinned; this is a promotion, not an introduction.
The planner should still run `gsd_run query package-legitimacy check --ecosystem npm|go gonum.org/v1/gonum`
if that seam supports Go modules, to get a machine-checked verdict on record rather than relying
on this research's `[ASSUMED]` tag alone.

## Architecture Patterns

### System Architecture Diagram

```
                         internal/query.Engine.FileGraph()
                         ─────────────────────────────────
   Store (Pebble)  ──►  scan 1: IterateNodes()         scan 2: IterateEdges("")
   [read-only]           │  build nodeFile map           │  build pairCounts map
                          │  build fileAggs                │  exclude contains/self-edges
                          ▼                                ▼
                   sort paths ascending  ──►  build FileGraphNode[] / FileGraphEdge[]
                                                   │
                                                   ▼
                             stronglyConnectedCycles(adj)   ◄── EXISTING (filegraph_cycles.go)
                                   │  populates CycleID/InCycle/CycleCount
                                   ▼
                       ★ NEW: assignCommunities(nodes, edges, seed)  (community.go)
                             │  build simple.WeightedUndirectedGraph
                             │    (sorted-path insertion, sequential int64 IDs, weight=TotalCount)
                             │  community.Modularize(g, 1.0, rand.NewPCG(seed1, seed2))
                             │  canonical relabel by smallest member path
                             ▼
                       populates FileGraphNode.CommunityID / FileGraphResult.CommunityCount
                                                   │
                                                   ▼
                        FileGraphResult  ──►  internal/uiserver.fileGraphToProto()
                                                   │  (+ fileGraphNodeToProto: CommunityId int32)
                                                   ▼
                        ui.proto FileGraphResponse (community_count=7)
                        ui.proto FileGraphNode (community_id=5)
                                                   │  ConnectRPC / protobuf-JSON
                                                   ▼
                        web/src/lib/gen/ui_pb.ts (generated)
                                                   │
                                                   ▼
                        file-graph-transform.ts: fileNodeElement()
                             adds data.communityId = n.communityId
                             (directory/collapsed nodes stay NEUTRAL — D-11)
                                                   │
                                                   ▼
                        graph-style.ts: 12 new per-community-index style rules
                             node[!isDirectory].graph-community-{(id-1)%12}
                                                   │
                                                   ▼
                        GraphCanvas.svelte (name: 'elk', UNCHANGED layout config)
                                                   │
                                                   ▼
                        +page.svelte: "N communities" toolbar line
                             reads graphState.response.communityCount
```

### Recommended Project Structure

```
internal/query/
├── traverse.go              # FileGraph() gains one new call site + CommunityID/CommunityCount fields
├── filegraph_cycles.go      # UNCHANGED — the sibling precedent, not touched
├── community.go             # NEW: assignCommunities, canonical relabeling, seed constant
├── community_test.go        # NEW: determinism test (≥3 runs), RED-control perturbation test
internal/uiproto/uiv1/
├── ui.proto                 # +2 fields: FileGraphNode.community_id=5, FileGraphResponse.community_count=7
internal/uiserver/
├── handlers.go              # fileGraphNodeToProto/fileGraphToProto gain 2 fields
├── readonly_test.go         # +2 fixture rows, uiProtoFieldFixtureLenAtPlan1101 = ...Plan1001 + 2
web/src/lib/components/graph/
├── file-graph-transform.ts  # FileGraphNodeData gains communityId?: number
├── graph-style.ts           # +12 static per-community selectors (communityDiscriminatorClass)
web/src/routes/graph/
├── +page.svelte             # +1 toolbar <p> mirroring graph-cycle-summary
corpora/
├── graph-cluster-threshold.json    # NEW, committed ALONE, first, before any code above exists
├── graph-cluster-observations.json # NEW, written by the measurement harness
tools/graphcluster/ (or extend tools/corpora -mode)
├── main.go                  # NEW: Go harness — opens guava's cached store, runs FileGraph()+
│                             #      assignCommunities in isolation, times it, writes observations
```

### Pattern 1: Fresh-per-call graph algorithm inside `FileGraph()`

**What:** A pure function taking already-scanned data (nodes/edges) and returning a
membership map, called once per `FileGraph()` invocation, with no package-level cache.

**When to use:** Any graph analysis this phase or a future one adds to the file rollup.

**Example (the exact precedent to mirror):**

```go
// Source: internal/query/filegraph_cycles.go:1-13, 24-45 (read this session)
package query

import "sort"

// filegraph_cycles.go computes strongly-connected-component membership
// over a file-level adjacency (GRF-04, D-06). It does NOT read the
// store, does NOT know about edge kinds, and never runs client-side...
//
// adj maps a file path to the file paths it has an aggregated edge
// toward (FileGraph's rollup, kind-agnostic). ... Component ids are
// 1-based and assigned in strict, sorted-input-derived order: the
// adjacency's keys are iterated in sorted order, and each key's
// successor list is iterated in sorted order, so two calls over the
// same adjacency agree element for element rather than depending on
// Go's randomized map iteration.
func stronglyConnectedCycles(adj map[string][]string) map[string]int {
	// ... (iterative Tarjan's SCC, explicit work stack, no recursion)
}
```

`community.go` should be this file's sibling: same "no store access, pure function of
already-scanned data" shape, called from the exact same block in `FileGraph()` where
`stronglyConnectedCycles` is invoked today (`internal/query/traverse.go:315-330`, read this
session — the `distinctCycles`/`result.Nodes[i].CycleID = id` loop is the direct model for a
`result.Nodes[i].CommunityID = id` loop).

### Pattern 2: Deterministic Louvain construction

**What:** Build `simple.WeightedUndirectedGraph`, insert nodes in a canonical order with
sequential dense IDs, run `community.Modularize` with a fixed `math/rand/v2.Source`, and
canonically relabel the result.

**When to use:** GRF-06/GRF-08's community-detection step, exactly once per `FileGraph()` call.

**Example:**

```go
// Source: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_common.go:90-91
// (read this session, $GOMODCACHE/gonum.org/v1/gonum@v0.17.0/graph/community/louvain_common.go)
func Modularize(g graph.Graph, resolution float64, src rand.Source) ReducedGraph {
	switch g := g.(type) {
	case graph.Undirected:
		return louvainUndirected(g, resolution, src)
	// ...
	}
}

// Source: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_undirected.go:80-96 (read this session)
func louvainUndirected(g graph.Undirected, resolution float64, src rand.Source) *ReducedUndirected {
	c := reduceUndirected(g, nil)   // <-- FIRST call: communities==nil branch sorts by ID
	rnd := rand.IntN
	if src != nil {
		rnd = rand.New(src).IntN
	}
	for {
		l := newUndirectedLocalMover(c, c.communities, resolution)
		if l == nil {
			return c
		}
		if done := l.localMovingHeuristic(rnd); done {
			return c
		}
		c = reduceUndirected(c, l.communities)
	}
}

// Source: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_undirected.go:224-238 (read this session)
// communities==nil branch — THIS is what neutralizes simple.WeightedUndirectedGraph's
// map-order Nodes():
//   nodes := graph.NodesOf(g.Nodes())
//   order.ByID(nodes)   // <-- sorts by n.ID(), regardless of map iteration order
//   communities = make([][]graph.Node, len(nodes))
//   for i := range nodes { communities[i] = []graph.Node{node(i)} }

// Source: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_undirected.go:475-481 (read this session)
// The ONLY randomness entry point — Fisher-Yates shuffle of the node
// processing order before each local-moving pass:
func (l *undirectedLocalMover) shuffle(rnd func(n int) int) {
	l.moved = false
	for i := range l.nodes[:len(l.nodes)-1] {
		j := i + rnd(len(l.nodes)-i)
		l.nodes[i], l.nodes[j] = l.nodes[j], l.nodes[i]
	}
}
```

```go
// PROPOSED (not yet written) — internal/query/community.go's shape:
import (
	"math/rand/v2" // `[VERIFIED: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_common.go:9 imports "math/rand/v2"]`

	"gonum.org/v1/gonum/graph/community"
	"gonum.org/v1/gonum/graph/simple"
)

// communitySeed1/communitySeed2 are the committed constants driving
// rand.NewPCG — chosen once, never changed (Claude's Discretion per
// 11-CONTEXT.md). math/rand/v2.Source requires Uint64() uint64
// `[VERIFIED: go doc math/rand/v2 Source, run this session]`; rand.NewPCG(seed1, seed2 uint64) *PCG
// is the standard deterministic constructor `[VERIFIED: go doc math/rand/v2 NewPCG, run this session]`.
const (
	communitySeed1 uint64 = /* TODO: pick a constant, e.g. 0x636f646567726170 */
	communitySeed2 uint64 = /* TODO: pick a constant */
)

func assignCommunities(nodes []FileGraphNode, edges []FileGraphEdge) map[string]int {
	g := simple.NewWeightedUndirectedGraph(0, 0)
	// D-02a: insert in SORTED path order with SEQUENTIAL int64 IDs —
	// this is what makes order.ByID's re-sort inside reduceUndirected
	// match insertion order deterministically, independent of
	// simple.WeightedUndirectedGraph's own map-backed Nodes() iteration.
	idByPath := make(map[string]int64, len(nodes))
	for i, n := range nodes { // nodes is ALREADY sorted by path (FileGraph's own scan)
		id := int64(i)
		idByPath[n.Path] = id
		g.AddNode(simple.Node(id))
	}
	for _, e := range edges {
		if e.TotalCount == 0 {
			continue
		}
		g.SetWeightedEdge(g.NewWeightedEdge(
			simple.Node(idByPath[e.SourceFile]),
			simple.Node(idByPath[e.TargetFile]),
			float64(e.TotalCount),
		))
	}
	reduced := community.Modularize(g, 1.0, rand.NewPCG(communitySeed1, communitySeed2))
	// canonical relabel: reduced.Communities() is [][]graph.Node keyed by
	// dense internal community index (NOT stable across runs on its own —
	// see D-03/D-02c). Sort communities by their smallest member's PATH,
	// then assign 1..N in that order.
	// ...
}
```

### Pattern 3: Additive proto field + same-commit fixture recipe

**What:** Adding a field to an existing message without a new rpc, with the field-number
fixture and `wantUIServiceMethods` updated in the same commit.

**When to use:** Every wire change in this codebase since Phase 9 (`[VERIFIED:
.planning/phases/10-index-health-the-coverage-denominator/10-01-SUMMARY.md:12]` — "the
additive-rpc + same-commit-fixture-move recipe (proto edit + task proto:gen +
wantUIServiceMethods + field fixture in one commit)").

**Example (verified field numbers, read this session):**

```proto
// Source: internal/uiproto/uiv1/ui.proto:856-870 (read this session)
message FileGraphNode {
  string path = 1;
  string language = 2;
  int64 symbol_count = 3;
  int32 cycle_id = 4;
  // PROPOSED: int32 community_id = 5;  <-- confirmed free, next after cycle_id
}
```

```proto
// Source: internal/uiproto/uiv1/ui.proto:913-933 (read this session)
message FileGraphResponse {
  repeated FileGraphNode nodes = 1;
  repeated FileGraphEdge edges = 2;
  int64 excluded_package_node_count = 3;
  int64 excluded_self_edge_count = 4;
  int64 excluded_contains_edge_count = 5;
  int32 cycle_count = 6;
  // PROPOSED: int32 community_count = 7;  <-- confirmed free, next after cycle_count
}
```

```go
// Source: internal/uiserver/readonly_test.go:507-510 (read this session) — the fixture
// rows to extend, exact style:
{"FileGraphNode", "path", 1},
{"FileGraphNode", "language", 2},
{"FileGraphNode", "symbol_count", 3},
{"FileGraphNode", "cycle_id", 4},
// PROPOSED: {"FileGraphNode", "community_id", 5},
// PROPOSED: {"FileGraphResponse", "community_count", 7},
```

```go
// Source: internal/uiserver/readonly_test.go:282-295 (read this session) — the length-
// fixture chain to extend:
// uiProtoFieldFixtureLenAtPlan1001 EXTENDS uiProtoFieldFixtureLenAtPlan0901
const uiProtoFieldFixtureLenAtPlan1001 = uiProtoFieldFixtureLenAtPlan0901 + 17
// PROPOSED: const uiProtoFieldFixtureLenAtPlan1101 = uiProtoFieldFixtureLenAtPlan1001 + 2
```

`wantUIServiceMethods` (`internal/uiserver/readonly_test.go:79-95`, read this session) stays
at 16 entries unchanged — no new rpc is added, confirmed against `internal/uiserver/handlers.go`
lines 1021-1069 (`fileGraphToProto`/`fileGraphNodeToProto`/`fileGraphEdgeToProto`), which are
the exact functions Task 1 must extend by two fields.

### Anti-Patterns to Avoid

- **Persisting or caching community assignments in this phase:** D-15 in `11-CONTEXT.md` is
  explicit — fresh on every `FileGraph()` call, no cache, ever. The only persistence path is
  the D-07 fallback, and only if GRF-09 measures a FAIL.
- **Colouring collapsed directory compound nodes by majority community:** D-11 forbids this —
  directory nodes stay neutral (border-only, matching today's `expandedDirElement`/
  `collapsedDirElement` shapes, neither of which should gain a `communityId` field).
- **Trusting `reduced.Communities()`'s own numbering as stable:** it is a dense internal index
  assigned during reduction and is NOT guaranteed to label the "same" community the same way
  across two calls, even with identical input and seed, because internal community indices
  are assigned in local-mover processing order. Canonical relabeling by smallest member path
  (D-02c) is mandatory, not cosmetic — skipping it is very likely to fail GRF-08's determinism
  test even when the underlying partition is genuinely identical.
- **Using a browser/Playwright harness for GRF-09:** D-06 in `11-CONTEXT.md` binds the metric
  to "clustering time in isolation... measured in-process by a Go harness" — reusing
  `web/scripts/graph-measure.mjs`'s Playwright machinery would measure page load latency, a
  different (and already-passing, per GRF-01) metric.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Louvain modularity optimization | A custom local-moving/modularity-gain implementation | `gonum.org/v1/gonum/graph/community.Modularize` | Already vetted, already in the dependency graph, already proven deterministic given the mechanisms this research documents; hand-rolling risks subtle non-determinism bugs the fallback (label propagation) exists to cover only if gonum itself fails GRF-10/GRF-08, not as a first choice |
| Categorical colour scale for N communities | A runtime hash-to-HSL function or `d3-scale-chromatic`-style dependency | 12 static, hand-picked colour-blind-safe hex values in `graph-style.ts`, `(id-1) mod 12` indexing (mirrors the existing `cycleDiscriminatorClass` per-id class-generation pattern) | No new JS dependency, matches the file's stated "no `var()`, literal hex" constraint (cytoscape does not resolve CSS custom properties), and is trivially testable (`web/tests`) |
| Ancestry proof for the threshold commit | A hand-run shell one-liner repeated at verify time (GRF-01's actual mechanism) | A persisted Go test or script asserting `git merge-base` ordering, following D-08's "checked by a test, not asserted in prose" | GRF-01 itself only ran this check manually during 05-04's plan execution (`05-04-PLAN.md:808`); GRF-09's own context explicitly upgrades this to a checked-in, automated assertion |

**Key insight:** every piece of this phase already has a direct, working precedent somewhere
in this codebase (cycle detection for the algorithm shape, `GetCoverage`/`GetEditorLink` for
the wire recipe, `graph-render-threshold.json` for the threshold-artifact shape). The actual
new engineering surface is narrow: one algorithm call, two proto fields, twelve colour
constants, and one measurement harness.

## Common Pitfalls

### Pitfall 1: Trusting `g.Nodes()` order on the raw input graph

**What goes wrong:** Assuming `simple.WeightedUndirectedGraph.Nodes()` returns nodes in
insertion order (it does not — `[VERIFIED: gonum.org/v1/gonum@v0.17.0/graph/simple/
weighted_undirected.go:30,136-142]`, `nodes map[int64]graph.Node; func (g
*WeightedUndirectedGraph) Nodes() graph.Nodes { ...; return iterator.NewNodes(g.nodes) }` —
a genuine Go-map iteration).

**Why it happens:** It looks like a natural place to enforce sorted order, so a naive
implementation might try to control it there instead of relying on IDs.

**How to avoid:** Rely on `int64` node IDs being assigned in canonical (sorted-path) order at
construction time — `reduceUndirected`'s `order.ByID(nodes)` sort (its very first call, inside
`louvainUndirected`) sorts by `n.ID()`, so canonical IDs are what actually matter, not
insertion-time iteration order.

**Warning signs:** GRF-08's determinism test passes on some runs and fails on others despite a
fixed seed — almost certainly means node IDs were assigned from map iteration (e.g., iterating
a `map[string]*fileAgg` without first sorting) rather than from the already-sorted
`FileGraph()` node slice.

### Pitfall 2: Degenerate graphs — zero edges, one/two nodes

**What goes wrong:** `newUndirectedLocalMover` returns `nil` when `l.m2 == 0` (total edge
weight is zero) — `[VERIFIED: gonum.org/v1/gonum@v0.17.0/graph/community/
louvain_undirected.go, newUndirectedLocalMover's "if l.m2 == 0 { return nil }" branch, read
this session]`. `louvainUndirected` handles this (`if l == nil { return c }`), returning the
initial per-node singleton partition — every node its own community. This is the CORRECT
behaviour for a repository with no cross-file edges (or a repository whose only file has no
outgoing/incoming edges): every file is its own community, `CommunityCount == len(Nodes)`.
Confirm this is acceptable rendering (every node a different colour) rather than treated as an
error.

**Why it happens:** Not actually a bug in gonum — a planner might assume "zero edges" needs
special-casing in `community.go` when gonum already handles it correctly.

**How to avoid:** Write an explicit unit test asserting the zero-edge and single-node cases
produce one singleton community per node, not a panic or an empty result.

**Warning signs:** A test with a tiny synthetic fixture (1-2 nodes, no edges) crashing or
returning an empty map.

### Pitfall 3: Weight scaling / resolution parameter

**What goes wrong:** CONTEXT.md D-01 pins resolution at `1.0` and weight = raw `TotalCount`
(an integer edge-kind sum per file pair, potentially ranging from 1 to hundreds on a dense
pair). gonum's `Modularize` does not itself require normalized weights — the modularity
formula divides by total edge weight (`m2`) internally — so no explicit normalization step is
needed. The risk is a future contributor "fixing" this by log-scaling or min-max-normalizing
weights without realizing the formula already self-normalizes.

**Why it happens:** Modularity optimization literature sometimes discusses weight scaling for
numerical stability on graphs with extreme weight ranges; this repository's file-pair weights
(edge counts in the tens to low thousands based on GRF-01's guava measurements) are nowhere
near a range where that matters.

**How to avoid:** Do not add a normalization step unless GRF-09's performance measurement or a
correctness bug specifically demands it. Keep raw `TotalCount` as the edge weight, as CONTEXT
D-01 specifies.

**Warning signs:** None currently — this is a preventive note, not an observed defect.

### Pitfall 4: `Q()`/negative-weight panics

**What goes wrong:** `positiveWeightFuncFor` in gonum panics on a negative edge weight
(`[VERIFIED: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_common.go, const
negativeWeight = "community: unexpected negative edge weight"]`). `FileGraphEdge.TotalCount`
is a sum of non-negative per-kind counts (`int64`, always ≥ 0 per `traverse.go`'s
aggregation), so this should never fire in production — but a test fixture that hand-builds a
negative `TotalCount` for convenience would panic the whole `FileGraph()` call.

**How to avoid:** Never construct a test fixture with a negative `TotalCount`; if a test needs
to exercise the zero-weight/no-edge path, use `TotalCount: 0` or omit the edge, not a negative
sentinel.

### Pitfall 5: Performance at 3.2k nodes / 21k edges — untested by this research session

**What we know:** `community.Modularize` is not documented by gonum with Big-O or benchmark
numbers in-tree for this session's reading; Louvain is well known in the broader literature as
near-linear in practice for sparse graphs (this file-pair graph — 21,554 edges over 3,233
nodes — is sparse). CONTEXT.md D-06 sets the pass bar at 500ms median-of-3, "well above
Louvain's expected tens of milliseconds at that scale" — that expectation is `[ASSUMED]`, not
measured in this research session (no gonum benchmark was run against the actual guava
file-pair graph).

**Recommendation:** GRF-09's own measurement is the authoritative answer here — this is
precisely why the threshold-before-measurement protocol exists. Do not treat the "tens of
milliseconds" framing in CONTEXT.md as verified; it is background/motivating context for why
500ms was chosen as the bar, not a substitute for running the harness.

## Code Examples

### GRF-10's cgo-closure check — run and verified this session

```bash
# Source: run live this session against this repository's actual module graph
# (GOFLAGS=-mod=mod used only to let `go list` see the not-yet-promoted
# indirect dependency; go.mod was reverted with `git checkout -- go.mod`
# immediately after, confirmed via `git status --porcelain go.mod` = clean).
GOTOOLCHAIN=go1.26.6 GOFLAGS=-mod=mod go list -deps \
  -f '{{.ImportPath}} {{len .CgoFiles}}' gonum.org/v1/gonum/graph/community
```

`[VERIFIED: command run this session]` — 91 total lines (stdlib + gonum), every `{{len
.CgoFiles}}` value is `0`, and `gonum.org/v1/gonum/blas/cgo` does not appear anywhere in the
closure (only the pure-Go `gonum.org/v1/gonum/blas/gonum` backend is reached, via `mat` →
`lapack/gonum`). This is GRF-10's exact required check shape — the planner's `check:gonum`
Taskfile target should run this literal command (without the `GOFLAGS=-mod=mod` workaround,
once `go.mod` carries the direct require) and assert both "package count > 0" and "zero rows
with a non-zero second column."

### GRF-10's govulncheck main-module scope — confirmed via existing CI split

```bash
# Source: .github/workflows/ci.yml:246-269 (read this session) — the EXISTING
# blocking main-module gate this project already runs, via golang/govulncheck-action.
# GRF-10's new Taskfile target must scope govulncheck to the MAIN module in
# SOURCE mode (like this job), NOT the existing Taskfile.yml `vuln:` target,
# which is scoped to TOOL BINARIES built from go.tool*.mod files and is
# explicitly ADVISORY, not blocking (Taskfile.yml:1461-1546, read this session).
govulncheck ./...
```

`[VERIFIED: /Users/sean/go/bin/govulncheck present via 'command -v govulncheck', run this
session]` — a local binary exists; `[VERIFIED: /opt/homebrew/bin/syft present via 'command -v
syft', run this session]` — syft is also locally installed, matching `.goreleaser.yaml`'s
`sboms:` pipe (`cmd: syft`, `internal_asm... args: [$artifact, --output, spdx-json=$document]`,
read at `.goreleaser.yaml:360-368`).

### The Node reserved-field fallback slot (D-07) — confirmed exact wording

```protobuf
// Source: internal/schema/graph.proto:65 (read this session)
message Node {
  // ... fields 1-49 ...
  reserved 50 to 59; // future: embedding vector, community/cluster assignment
}
```

This comment already names "community/cluster assignment" as the anticipated use of field 50
— direct confirmation that GRF-09's D-07 fallback (index-time persistence into field 50) was
planned for at schema-design time, not improvised this phase.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| N/A — this is the first community-detection feature in this codebase | gonum Louvain, fresh per call | This phase | Establishes the pattern any future graph-analysis feature (e.g. a hypothetical future centrality measure) should follow: pure function, sorted input, `internal/query`-local, no wire-layer import |

**Deprecated/outdated:** None applicable — no prior clustering implementation exists to
deprecate.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Louvain clustering time at guava scale (3,233 nodes / 21,554 edges) will land in "tens of milliseconds," well under GRF-09's 500ms bar | Common Pitfalls #5 | If wrong, GRF-09 measures a genuine FAIL and the phase must execute the D-07 fallback (index-time persistence into `Node` field 50) — this is explicitly an acceptable, planned-for outcome per CONTEXT.md, not a research failure, but the planner should not assume the harness will trivially PASS |
| A2 | `gonum.org/v1/gonum`'s package-legitimacy verdict is clean (age/maintainership/no malicious postinstall) | Package Legitimacy Audit | Low risk — the package is a well-known, long-established numerical computing library already resolved in this project's `go.sum`; the `[ASSUMED]` tag exists only because this session did not invoke `gsd_run query package-legitimacy check` against it, not because of any adverse signal found |
| A3 | The committed seed constants for `rand.NewPCG(seed1, seed2)` can be arbitrary fixed `uint64` values with no further constraint from gonum | Pattern 2 code example | Low risk — `PCG`'s doc states "A zero PCG is equivalent to NewPCG(0, 0)" with no documented restriction on seed values; if this is wrong in some edge case, GRF-08's determinism test will simply fail and surface it immediately, at low cost |
| A4 | The palette proposed for community colours (12 colour-blind-safe hues) does not clash meaningfully with the existing fixed file-node blue (`#2563eb`) or cycle-border red (`#dc2626`) once background-color and border-color compose | Architecture Patterns / Don't Hand-Roll | Low-medium risk — a genuinely poor colour choice is a cosmetic defect caught immediately in manual/live-browser verification (this codebase's established practice per multiple STATE.md entries: "found live, manual UAT, not assumed"), not a functional one |

## Open Questions

1. **Exact committed seed constant and exact 12 hex values for the community palette**
   - What we know: CONTEXT.md explicitly delegates both to "Claude's Discretion."
   - What's unclear: No specific values are fixed by any prior decision.
   - Recommendation: Planner picks concrete values at plan-writing time (this research
     deliberately leaves them as `TODO` in the code example above); a colour-blind-safe
     extended Okabe-Ito/Tableau-style 12-hue set is a reasonable, defensible default, but this
     research did not invoke the `dataviz` skill's `references/palette.md` this session
     (searched for it on this machine — not found locally, so it may be a plugin-scoped
     resource not resolvable via a plain filesystem search) and did not verify it is
     accessible; the planner should try invoking that skill directly at plan time.

2. **Whether the measurement harness extends `tools/corpora`'s existing `-mode measure` or is
   a new sibling `tools/graphcluster`**
   - What we know: `tools/corpora/main.go`'s doc comment (`[VERIFIED: tools/corpora/main.go:1-20,
     read this session]`) already states its `measure` mode "drives this repository's own
     indexer + query engine in-process to produce a measured Observation per corpus" — the
     exact mechanism GRF-09 needs, just for a different metric.
   - What's unclear: Whether reusing that exact command (adding a cluster-timing branch) or
     writing an independent tool better serves "re-runnable, reads the threshold file, never
     carries its own defaults" (CONTEXT.md's discretion note).
   - Recommendation: Extend `tools/corpora` if its existing `Observation` JSON shape can
     accommodate a clustering-time field cleanly; otherwise a new sibling tool following the
     identical structure. Either way, do not write a JS/Playwright harness for this — D-06 is
     explicit that this is a Go in-process measurement.

3. **Whether `gsd_run query package-legitimacy check` supports the Go/module ecosystem**
   - What we know: The Package Legitimacy Gate protocol's example commands are
     `--ecosystem npm|pypi|crates` — Go is not listed.
   - What's unclear: Whether the seam tool has Go-module support at all.
   - Recommendation: Attempt the call at plan time; if unsupported, keep the `[ASSUMED]` tag
     on `gonum.org/v1/gonum`'s legitimacy and route it through a `checkpoint:human-verify` per
     the protocol's own fallback instruction.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `govulncheck` | GRF-10 | ✓ | Local binary at `/Users/sean/go/bin/govulncheck` (version not queried; the Taskfile's own `vuln:` target builds a pinned copy from `go.tool.mod` — the new `check:gonum` target should likely build its own pinned copy the same way rather than relying on this ambient PATH binary) | Build from `go.tool.mod` via `GOWORK=off go build -modfile=go.tool.mod -o ... golang.org/x/vuln/cmd/govulncheck`, exact precedent at `Taskfile.yml:1514` |
| `syft` | GRF-10 (SBOM) | ✓ | `/opt/homebrew/bin/syft` (version not queried) | `.goreleaser.yaml`'s `sboms:` pipe already invokes `syft` by bare command name (not a pinned path), so CI's own toolchain setup is the actual authority — this research only confirms local dev-machine availability |
| `cyclonedx-gomod` | GRF-10 (SBOM alternative) | ✗ | — | Not installed locally; not needed since `.goreleaser.yaml` uses `syft`, and D-14 in CONTEXT.md says "match what `.goreleaser.yaml` uses" — syft is the correct choice, no fallback needed |
| gonum module cache | Reading gonum source for research | ✓ | `$GOMODCACHE/gonum.org/v1/gonum@v0.17.0` present and fully readable | — |
| `go.mod` at project's pinned toolchain | All Go commands this phase | ✓ (with landmine) | Local `go` is 1.27.1; `go.mod` pins `go 1.26.6` | Every `go` invocation in this phase's plans and CI-mirroring commands MUST be prefixed `GOTOOLCHAIN=go1.26.6` — confirmed this session: unprefixed `go list -deps ...` failed with "go: updates to go.mod needed" until `GOFLAGS=-mod=mod` was added, and the CI govulncheck job's own comment (`ci.yml:255-268`) documents a REAL prior incident where an unpinned toolchain silently broke the vulnerability scan on `cockroachdb/swiss`'s go1.27 incompatibility |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** `cyclonedx-gomod` — not needed; `syft` covers GRF-10's
SBOM requirement per the existing `.goreleaser.yaml` precedent.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | Standard `go test`, `GOTOOLCHAIN=go1.26.6` prefix mandatory |
| Framework (web) | vitest, invoked via `pnpm exec vitest run` (`[VERIFIED: Taskfile.yml:682, read this session]`) |
| Config file (Go) | `go.mod` (root); no separate test config |
| Config file (web) | `web/vite.config.ts` (vitest config lives here per prior-phase STATE.md notes) |
| Quick run command (Go) | `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... ./internal/uiserver/...` |
| Quick run command (web) | `cd web && pnpm exec vitest run tests/file-graph-transform.test.ts` |
| Full suite command (Go) | `GOTOOLCHAIN=go1.26.6 go test ./...` |
| Full suite command (web) | `cd web && pnpm exec vitest run` (mirrors `Taskfile.yml`'s `web:test` target, `[VERIFIED: Taskfile.yml:657-682]`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GRF-06 | Nodes coloured by community on the layered layout; no force-directed path reachable | unit (Go: field population) + unit (vitest: transform + positive-controlled source scan) | `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... -run TestFileGraphPopulatesCommunityFields`; `cd web && pnpm exec vitest run tests/file-graph-transform.test.ts` | ❌ Wave 0 (new test files) |
| GRF-06 (force-directed absence) | Positive-controlled source scan: `elk` found ≥1, zero of `cose`/`fcose`/`cola`/`euler`/`spread`/`force` | unit (a small Go or Node script grepping `web/src`) | New script, e.g. `node web/scripts/check-no-force-layout.mjs` or a Go test walking `web/src` | ❌ Wave 0 |
| GRF-08 | ≥3 identical runs, label-canonicalized-equal, reporting run count | unit (Go) | `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... -run TestAssignCommunitiesDeterministic -v` | ❌ Wave 0 (mirrors `TestFileGraphCyclesDeterministicIds`, `[VERIFIED: internal/query/filegraph_cycles_test.go:91-108, read this session]`) |
| GRF-08 (RED control) | Perturbed node order or seed turns the test red | unit (Go, test-only hook) | Same test file, a second sub-test using an injectable perturbation seam | ❌ Wave 0 |
| GRF-09 | Clustering time ≤500ms median-of-3 against pinned guava corpus | manual-only / harness-run (not part of `go test ./...`'s normal fast loop — opens a large external corpus) | New harness command, e.g. `GOTOOLCHAIN=go1.26.6 go run ./tools/graphcluster` (or `tools/corpora -mode cluster-measure`) | ❌ Wave 0 — corpus itself already cached locally, confirmed present at `~/.cache/codegraph/corpora/google-guava-2b0cb53f@94f39958baf7ad51ddf9c70e406ed6b188194daa/.codegraph/store` |
| GRF-09 (ancestry proof) | Threshold commit is an ancestor of every observation commit | unit/script (Go or shell, PERSISTED per D-08, unlike GRF-01's manual one-liner) | New test/script wrapping `git log --format=%H -- corpora/graph-cluster-threshold.json \| tail -1` + `git merge-base` (pattern verified at `05-04-PLAN.md:808`, read this session) | ❌ Wave 0 |
| GRF-10 | govulncheck passes over main module with gonum in scanned set | CI-level (existing `ci.yml` govulncheck job already covers the main module in source mode) + new Taskfile target | `GOTOOLCHAIN=go1.26.6 govulncheck ./...` (verified locally installed) | ✅ existing CI job; ❌ new local Taskfile target |
| GRF-10 (SBOM) | `gonum.org/v1/gonum` appears in a locally-generated SBOM, positive control on a known module | script | `syft . -o spdx-json=/tmp/sbom.json && jq -e '.packages[] \| select(.name==\"gonum.org/v1/gonum\")' /tmp/sbom.json` (positive control: assert `cockroachdb/pebble` also present, per CONTEXT.md's own suggestion) | ❌ Wave 0 |
| GRF-10 (cgo closure) | Zero cgo files across the full import closure, reporting package count | script (verified this session) | `GOTOOLCHAIN=go1.26.6 go list -deps -f '{{.ImportPath}} {{len .CgoFiles}}' gonum.org/v1/gonum/graph/community` | ❌ Wave 0 (wrap the verified-live command in a persisted Taskfile target/test) |

### Sampling Rate

- **Per task commit:** `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... ./internal/uiserver/...` + `cd web && pnpm exec vitest run tests/file-graph-transform.test.ts`
- **Per wave merge:** `GOTOOLCHAIN=go1.26.6 go test ./...` + `cd web && pnpm exec vitest run`
- **Phase gate:** Full suite green, plus GRF-09's harness run recorded (PASS or documented FAIL with fallback), before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/query/community_test.go` — covers GRF-08 (determinism + RED control), degenerate-graph pitfalls
- [ ] `internal/uiserver/readonly_test.go` extensions — covers GRF-06's wire-shape fixture (no new file, extends existing)
- [ ] `web/tests/file-graph-transform.test.ts` extensions — covers GRF-06's transform-layer colouring assertion (no new file, extends existing)
- [ ] A force-directed-absence source scan script — covers GRF-06's D-12b check
- [ ] `corpora/graph-cluster-threshold.json` + its committed-alone-first discipline — covers GRF-09 (must be its own commit, ancestor of everything else, per the phase's own blocking-within-phase rule)
- [ ] A Go measurement harness (`tools/graphcluster` or `tools/corpora` extension) — covers GRF-09's actual measurement
- [ ] A persisted ancestry-proof test/script — covers GRF-09's D-08 upgrade over GRF-01's manual precedent
- [ ] A `check:gonum` Taskfile target — covers GRF-10 end to end (govulncheck + SBOM grep + cgo scan, all three positive-controlled)
- [ ] `11-MUTATION-LOG.md` and `11-SECURITY.md` — house-format artifacts per CONTEXT.md D-16

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | This phase adds no new endpoint requiring auth; `FileGraph` is an existing read-only rpc |
| V3 Session Management | No | Not applicable — no session state introduced |
| V4 Access Control | No | No new access boundary; the community fields ride the same existing `FileGraph` rpc under the same read-only-by-construction discipline as everything else in this project |
| V5 Input Validation | Marginal | The clustering algorithm's only "input" is the store's own already-validated node/edge data (file paths, edge counts) — no new user-controlled input reaches `community.Modularize`; CONTEXT.md D-16 already states this explicitly ("inputs are the store's own file paths and counts") |
| V6 Cryptography | No | No cryptographic operation is introduced. The committed seed constant for `rand.NewPCG` is a determinism mechanism, not a security control — it must NOT be treated as, or documented as, a secret; it is committed in the clear precisely because reproducibility (not unpredictability) is the goal |
| V14 (Dependency/Configuration) | Yes | GRF-10 in full: govulncheck, SBOM inclusion, cgo-free closure verification for the one new supply-chain surface this phase adds |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Malicious/hijacked dependency (supply-chain) | Tampering | GRF-10's three-part check: govulncheck, SBOM presence, cgo-closure scan — all already exercised live this session with clean results (91 packages, 0 cgo) |
| Non-deterministic security-relevant output (a community assignment that silently changes between renders, potentially confusing a reviewer about code-ownership boundaries the colouring implies) | Repudiation / Tampering (weak form — no attacker, but a correctness property with security-adjacent trust implications) | GRF-08's determinism test with a RED-control mutation, per this project's own standing rule ("A gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation", `[VERIFIED: STATE.md standing decisions, line ~174]`) |
| Threshold-file tampering (widening a bar after a failing measurement to force a false PASS) | Tampering | The exact T-05-01/T-05-18 mitigation this project already applies to GRF-01: commit the threshold alone, first; the verdict mechanism never writes it; ancestry is machine-checked (D-08 upgrades this from GRF-01's manual `git merge-base` one-liner to a persisted, automated check) |
| A degenerate/adversarial graph (extremely dense file-pair weights, e.g. from a pathological monorepo) causing pathological Louvain runtime | Denial of Service (very weak — this is a local dev tool, not a network-facing service under attacker control) | GRF-09's measurement against the largest known real corpus (guava) is the primary control; no additional hardening is warranted for a locally-run, read-only developer tool computing over the user's own already-indexed data |

## Sources

### Primary (HIGH confidence)

- `gonum.org/v1/gonum@v0.17.0` source, read directly from `$GOMODCACHE` this session:
  `graph/community/louvain_common.go`, `graph/community/louvain_undirected.go`,
  `graph/simple/weighted_undirected.go`, `internal/order/order.go` — the randomness-entry-point
  and determinism-mechanism findings above are read from these files directly, not inferred.
- `go list -deps -f '{{.ImportPath}} {{len .CgoFiles}}' gonum.org/v1/gonum/graph/community` —
  run live this session, 91-package closure, zero cgo confirmed.
- `go doc math/rand/v2 Source` / `go doc math/rand/v2 NewPCG` — run live this session, confirms
  `Source` interface shape and the `NewPCG(seed1, seed2 uint64) *PCG` constructor.
- `internal/query/traverse.go`, `internal/query/filegraph_cycles.go`,
  `internal/query/filegraph_cycles_test.go` — read in full this session for `FileGraph()`'s
  exact scan/sort/populate structure and the cycle-detection pattern to mirror.
- `internal/uiproto/uiv1/ui.proto`, `internal/uiserver/handlers.go`,
  `internal/uiserver/readonly_test.go` — read directly this session to confirm free field
  numbers (5, 7), the exact handler-mapping functions to extend, and the fixture/length-const
  chain.
- `internal/schema/graph.proto` — read this session, confirms the `reserved 50 to 59` slot's
  own comment already names "community/cluster assignment."
- `web/src/lib/components/graph/file-graph-transform.ts`, `graph-style.ts`,
  `GraphCanvas.svelte` (layout config section), `web/src/routes/graph/+page.svelte` — all read
  directly this session for the exact element-data shape, stylesheet convention, layout
  registration, and toolbar insertion point.
- `Taskfile.yml` (`vuln:` target, lines ~1461-1546), `.github/workflows/ci.yml` (govulncheck
  job, lines 246-269), `.goreleaser.yaml` (`sboms:` block, lines 330-368) — all read this
  session to establish the correct scope split for GRF-10's new target.
- `corpora/graph-render-threshold.json`, `.planning/milestones/v0.12.0-phases/
  05-file-package-graph-view/05-CONTEXT.md`, `05-04-PLAN.md`, `05-04-SUMMARY.md` — read this
  session for the threshold-artifact shape and the exact `git merge-base` ancestry-proof
  mechanism GRF-01 used.
- `internal/corpora/manifest.go`, local filesystem check of
  `~/.cache/codegraph/corpora/google-guava-2b0cb53f@94f39958baf7ad51ddf9c70e406ed6b188194daa/.codegraph/store`
  — confirms the exact corpus-store path formula and that the guava store is a real, already-
  indexed Pebble store ready for a Go harness to open read-only.
- `tools/corpora/main.go` — read this session, confirms an existing in-process
  indexer+query-engine-driven measurement tool this phase's harness can extend or mirror.

### Secondary (MEDIUM confidence)

- `.planning/phases/10-index-health-the-coverage-denominator/10-01-SUMMARY.md` — read this
  session for the additive-wire same-commit-fixture recipe description (prose summary of a
  prior phase's own commits, not the commits themselves).

### Tertiary (LOW confidence)

- None — every claim above traces to a file read or a command run this session, or is
  explicitly tagged `[ASSUMED]` in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — gonum's version, closure, and cgo-freedom were all verified live this session, not assumed from training data.
- Architecture: HIGH — every integration point (proto fields, handler functions, transform/style/layout files, toolbar location) was read directly, not inferred.
- Pitfalls: HIGH for the determinism mechanism (read gonum source directly); MEDIUM for performance expectations (genuinely unmeasured — flagged as A1/Pitfall 5).

**Research date:** 2026-09-13
**Valid until:** ~30 days (stable dependency, stable internal architecture) — but GRF-09's own measurement result is authoritative the moment it runs, superseding this research's performance assumption immediately.
