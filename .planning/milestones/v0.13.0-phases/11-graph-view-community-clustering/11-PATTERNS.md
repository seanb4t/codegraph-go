# Phase 11: Graph View — Community Clustering - Pattern Map

**Mapped:** 2026-09-13
**Files analyzed:** 17 (new + modified)
**Analogs found:** 17 / 17 (every file has an in-repo precedent — RESEARCH.md already did the heavy verification; this document translates it into planner-consumable analog assignments)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/query/community.go` (new) | service (query engine, pure fn) | CRUD (fresh-per-call graph algorithm) | `internal/query/filegraph_cycles.go` (`stronglyConnectedCycles`) | exact |
| `internal/query/traverse.go` (modify — `FileGraph()` call site + `CommunityID`/`CommunityCount` fields) | service | CRUD | itself — the existing `CycleID`/`CycleCount` populate loop (`traverse.go:314-330`) | exact |
| `internal/query/community_test.go` (new) | test | — | `internal/query/filegraph_cycles_test.go` (`TestFileGraphCyclesDeterministicIds`, `TestFileGraphPopulatesCycleFields`) | exact |
| `internal/uiproto/uiv1/ui.proto` (modify — `FileGraphNode.community_id=5`, `FileGraphResponse.community_count=7`) | model/wire schema | CRUD (additive wire schema) | itself — `cycle_id=4` / `cycle_count=6` field history | exact |
| `internal/uiserver/handlers.go` (modify — `fileGraphNodeToProto`/`fileGraphToProto`) | controller | request-response | itself — `CycleId`/`CycleCount` mapping (`handlers.go:1021-1053`) | exact |
| `internal/uiserver/readonly_test.go` (modify — field fixture + length constant) | test | — | itself — `uiProtoFieldFixtureLenAtPlan1001` chain and `{"FileGraphNode","cycle_id",4}` / `{"FileGraphResponse","cycle_count",6}` rows | exact |
| `corpora/graph-cluster-threshold.json` (new, committed alone) | config (committed data) | batch (one-way threshold artifact) | `corpora/graph-render-threshold.json` | exact |
| `corpora/graph-cluster-observations.json` (new, harness-written) | config (committed data) | batch (observation record) | `corpora/observations.json` shape via `tools/corpora/main.go -mode measure` | exact |
| `tools/corpora/main.go` (extend, new `-mode cluster-measure` or sibling) | service (Go in-process harness) | batch (measurement) | itself — existing `-mode measure` (opens cached store, drives indexer+query engine in-process, writes an `Observation`) | exact |
| Ancestry test (new, Go, git-backed) | test | request-response (shell to `git`) | `internal/query/engine_worktree_test.go` (`exec.Command("git", full...)`, lines 31-34) | exact |
| `Taskfile.yml` `check:gonum` (new target) | config/CI (Taskfile) | batch (positive-controlled scan) | `vuln:` target (pinned tool binaries via `go.tool.mod`, ~Taskfile.yml:1500-1546) + `proto:drift` (positive-count guard: "compared N files", fails if N < floor) | exact |
| `go.mod` (modify — `gonum` indirect→direct) | config | CRUD (one-line dependency promotion) | `github.com/bmatcuk/doublestar/v4 v4.10.0` direct require precedent (go.mod:10) | exact |
| `web/src/lib/components/graph/file-graph-transform.ts` (modify — `communityId` on element data) | utility (pure transform) | transform | itself — `cycleId?: number` field + `cycleDiscriminatorClass` (lines 64-65, 108-109, 172, 177-178) | exact |
| `web/src/lib/components/graph/graph-style.ts` (modify — 12 community colour selectors) | component (Cytoscape stylesheet) | transform | itself — `.graph-cycle` selectors (lines 13-18, 141-158) | role-match (no existing `data()`-driven categorical colour mapper; nearest shape is the per-id discriminator class, not a `mapData` numeric one) |
| `web/src/lib/components/graph/GraphCanvas.svelte` (unchanged — ELK layout config documented) | component | request-response | itself | exact (no-change assertion) |
| `web/src/routes/graph/+page.svelte` (modify — "N communities" toolbar line) | component | request-response (poll + render) | itself — `graph-cycle-summary` paragraph (lines 563-573) | exact |
| `web/tests/file-graph-transform.test.ts` / graph-page vitest (extend) | test | — | itself — existing cycle-field assertions in the same test file | exact |
| `web/scripts/check-no-force-layout.mjs` (new) | test (Node source-scan script) | batch (static scan, no browser) | `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess`-style structural scan (Go analog) for the *shape*; `web/scripts/breadcrumb-check.mjs` for Node-script shebang/diagnostic-JSON conventions only (this script is NOT a Playwright script — no browser needed) | role-match |
| `11-MUTATION-LOG.md` (house format, values only) | planning artifact | — | `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md` | exact |
| `11-SECURITY.md` (house format, values only) | planning artifact | — | `.planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md` | exact |

## Pattern Assignments

### `internal/query/community.go` (service, CRUD — new)

**Analog:** `internal/query/filegraph_cycles.go` (whole file, 1-13, 24-45)

**Package doc / "pure function of already-scanned data" contract to copy verbatim in spirit**:
```go
// Source: internal/query/filegraph_cycles.go:1-13
package query

import "sort"

// filegraph_cycles.go computes strongly-connected-component membership
// over a file-level adjacency (GRF-04, D-06). It does NOT read the
// store, does NOT know about edge kinds, and never runs client-side...
```
`community.go` mirrors this exactly: no store access, pure function `assignCommunities(nodes []FileGraphNode, edges []FileGraphEdge) map[string]int`, called from the identical block in `FileGraph()` where `stronglyConnectedCycles` runs today.

**Call-site pattern to extend** (`internal/query/traverse.go:311-330`, read this session):
```go
cycleIDs := stronglyConnectedCycles(cycleAdj)
distinctCycles := make(map[int]struct{})
for i, n := range result.Nodes {
    if id, ok := cycleIDs[n.Path]; ok {
        result.Nodes[i].CycleID = id
        distinctCycles[id] = struct{}{}
    }
}
result.CycleCount = len(distinctCycles)
// NEW, same shape:
// communityIDs := assignCommunities(result.Nodes, result.Edges)
// distinctCommunities := make(map[int]struct{})
// for i, n := range result.Nodes {
//     id := communityIDs[n.Path]   // every node gets exactly one — no `ok` check (D-03)
//     result.Nodes[i].CommunityID = id
//     distinctCommunities[id] = struct{}{}
// }
// result.CommunityCount = len(distinctCommunities)
```

**Field additions mirror `FileGraphNode.CycleID` / `FileGraphResult.CycleCount`** (`traverse.go:87-89, 125-128`):
```go
// CycleID is 0 for a node in no strongly-connected cycle and is
// ... (existing doc pattern)
CycleID int
// NEW: CommunityID int — 1-based canonical, 0 means "not computed", never
// emitted on the fresh-compute path (D-03).

// CycleCount is the number of distinct strongly-connected components
CycleCount int
// NEW: CommunityCount int
```

**Deterministic Louvain construction** (gonum precedent, verified this session by RESEARCH.md against `$GOMODCACHE/gonum.org/v1/gonum@v0.17.0/graph/community/`):
```go
// Source: gonum.org/v1/gonum@v0.17.0/graph/community/louvain_common.go:90-91
func Modularize(g graph.Graph, resolution float64, src rand.Source) ReducedGraph
```
Build `simple.NewWeightedUndirectedGraph(0, 0)`, insert nodes in `result.Nodes`' already-sorted order with sequential `int64` IDs (`FileGraph()` already does `sort.Strings(paths)` at `traverse.go:260` — do not re-sort or re-derive order from a map), `g.SetWeightedEdge(...)` with weight = `FileGraphEdge.TotalCount`, call `community.Modularize(g, 1.0, rand.NewPCG(seed1, seed2))` (`math/rand/v2`, not `math/rand`), then canonically relabel `reduced.Communities()` by smallest-member path into 1..N.

---

### `internal/query/community_test.go` (test — new)

**Analog A (determinism shape):** `internal/query/filegraph_cycles_test.go:91-115` (`TestFileGraphCyclesDeterministicIds`) — run the algorithm N≥3 times over identical input, assert canonical-relabel equality element-for-element, log the run count.

**Analog B (fresh-per-call field population, whole-`FileGraph()` integration):** `internal/query/filegraph_cycles_test.go:161-210` (`TestFileGraphPopulatesCycleFields`, quoted in full):
```go
func TestFileGraphPopulatesCycleFields(t *testing.T) {
    nodes := map[string]*schema.Node{ /* ... */ }
    edges := []*schema.Edge{ /* ... */ }
    e := New(&traverseFakeReader{nodes: nodes, edges: edges})
    got, err := e.FileGraph()
    if got.CycleCount != 1 { t.Fatalf(...) }
    // per-node / per-edge assertions on CycleID / InCycle
}
```
`TestFileGraphPopulatesCommunityFields` mirrors this shape exactly, asserting `CommunityID` is 1-based and non-zero on every node (never 0, unlike `CycleID`'s "0 = none").

**RED-control seam (D-04):** no existing precedent in this codebase for a test-only order/seed perturbation hook — this is genuinely new engineering surface; follow `filegraph_cycles.go`'s existing "sorted-input-derived order" doc comment as the contract being defended, and wire the perturbation through an unexported test-only parameter or build-tag seam, recorded RED in `11-MUTATION-LOG.md` per the Phase 9/10 mutation-log convention (see below).

---

### `internal/uiproto/uiv1/ui.proto` (model/wire schema, CRUD)

**Analog:** itself — `cycle_id=4` / `cycle_count=6` field history (verified this session, lines 856-869, 913-933)

```proto
// Source: internal/uiproto/uiv1/ui.proto:856-869
message FileGraphNode {
  string path = 1;
  string language = 2;
  int64 symbol_count = 3;
  int32 cycle_id = 4;
  // NEW: int32 community_id = 5;  <-- confirmed free, next after cycle_id
}
```
```proto
// Source: internal/uiproto/uiv1/ui.proto:913-933
message FileGraphResponse {
  // ... fields 1-5 ...
  int32 cycle_count = 6;
  // NEW: int32 community_count = 7;  <-- confirmed free, next after cycle_count
}
```
Run `task proto:gen` once — regenerates both proto surfaces (Go + TS) in one invocation. No new rpc; `wantUIServiceMethods` stays at 16.

---

### `internal/uiserver/handlers.go` (controller, request-response)

**Analog:** itself — `fileGraphNodeToProto`/`fileGraphToProto` (verified this session, lines 1015-1053)

```go
// Source: internal/uiserver/handlers.go:1021-1053 (structure confirmed this session)
func fileGraphToProto(result query.FileGraphResult) *uiv1.FileGraphResponse {
    // ...
    resp := &uiv1.FileGraphResponse{
        // ...
        CycleCount: int32(result.CycleCount),
        // NEW: CommunityCount: int32(result.CommunityCount),
    }
    return resp
}

func fileGraphNodeToProto(n query.FileGraphNode) *uiv1.FileGraphNode {
    return &uiv1.FileGraphNode{
        // ...
        CycleId: int32(n.CycleID),
        // NEW: CommunityId: int32(n.CommunityID),
    }
}
```
Field-for-field mapping, same convention — no new mapper function needed since this is a straight int passthrough on an existing message (unlike Phase 10's `coverageToProto`, which mapped a compound sub-message).

---

### `internal/uiserver/readonly_test.go` (test)

**Analog:** itself — the `uiProtoFieldFixtureLenAtPlanNNNN` chain and fixture rows (lines 186-295, 504-553, 598-599)

```go
// Source: internal/uiserver/readonly_test.go:510, 521 (existing rows to extend)
{"FileGraphNode", "cycle_id", 4},
{"FileGraphResponse", "cycle_count", 6},
// NEW:
{"FileGraphNode", "community_id", 5},
{"FileGraphResponse", "community_count", 7},
```
```go
// Source: internal/uiserver/readonly_test.go:282-295 (the length-fixture chain to extend)
const uiProtoFieldFixtureLenAtPlan1001 = uiProtoFieldFixtureLenAtPlan0901 + 17
// NEW: const uiProtoFieldFixtureLenAtPlan1101 = uiProtoFieldFixtureLenAtPlan1001 + 2
```
`wantUIServiceMethods` is unchanged at 16 (no new rpc). All of this — proto edit, `task proto:gen`, both fixture updates — lands in the SAME commit (the 09-01/10-01 recipe, already documented in RESEARCH.md).

---

### `corpora/graph-cluster-threshold.json` (config, one-way artifact — new, committed alone first)

**Analog:** `corpora/graph-render-threshold.json` (full file read this session)

Copy its exact shape: `schemaVersion`, a `purpose` string stating amend/widen-after-measurement is forbidden and naming the fallback, `corpus` (pin `google/guava@94f39958baf7ad51ddf9c70e406ed6b188194daa`), `metrics.<name>.max` with a `definition` string, and (per D-07) an `onFailure` block written before measurement, naming the `Node` field-50 persistence fallback explicitly — mirror the render threshold's `purpose` wording:
```json
"purpose": "GRF-09's pass condition, locked before any measurement exists (D-05). ... Amending, widening or relaxing any value in this file after measurement is forbidden."
```
Binding metric per D-06: `clusteringTimeMs.max: 500`, `statistic: median`, `runs: 3`, cold runs against the guava corpus's already-scanned `FileGraphResult`.

---

### `tools/corpora/main.go` (service, Go in-process harness — extend)

**Analog:** itself — the existing `-mode measure` doc comment and implementation (verified this session, lines 1-40)

```go
// Source: tools/corpora/main.go:1-9 (package doc, confirms the exact
// mechanism GRF-09 needs, just for a different metric)
// ... or (in a later plan) CI read for the corpora manifest, plus
// the measurement instrument Plan 01-05 builds: given -mode, it prints
// ... drives this repository's own
// indexer + query engine in-process to produce a measured Observation
// per corpus and upsert it into corpora/observations.json (measure).
```
Add a `cluster-measure` mode (or extend `measure`'s `Observation` shape with a clustering-time field) that: opens the cached guava store (no re-fetch), calls `Engine.FileGraph()` to get the rollup, times `assignCommunities` (or the internal call inside `FileGraph()`) in isolation across 3 cold runs, computes the median, reads `corpora/graph-cluster-threshold.json` for the bar (never carries its own default — Claude's Discretion note in CONTEXT.md), and writes `corpora/graph-cluster-observations.json` with a `verdict` field, exiting non-zero on FAIL. Do NOT reuse `web/scripts/graph-measure.mjs`/`graph-verdict.mjs` (Playwright) — D-06 requires a Go in-process harness.

---

### Ancestry test (new, Go, git-backed — D-08's "checked by a test, not asserted in prose")

**Analog:** `internal/query/engine_worktree_test.go:31-34` (verified this session):
```go
// Source: internal/query/engine_worktree_test.go:31-34
cmd := exec.Command("git", full...)
cmd.Dir = dir
out, err := cmd.CombinedOutput()
if err != nil { /* ... */ }
```
`TestClusterThresholdCommitIsAncestorOfEveryMeasurement` (named in `11-VALIDATION.md`) shells to `git merge-base --is-ancestor <threshold-commit> <observation-commit>` (or `git log --follow` to find the threshold file's first commit, then `git merge-base --is-ancestor`) using this exact `exec.Command("git", ...)` + `cmd.Dir` shape, and reports the resolved commit hash plus the count of measurement commits compared (positive assertion, not just a boolean).

---

### `Taskfile.yml` `check:gonum` (new target)

**Analog A (pinned tool-binary build via `go.tool.mod`):** `vuln:` target (Taskfile.yml ~1500-1546, read this session):
```yaml
# Source: Taskfile.yml (vuln: target body, read this session)
GOWORK=off go build -modfile=go.tool.mod -o "${scratch}/govulncheck" golang.org/x/vuln/cmd/govulncheck
```
`check:gonum`'s govulncheck half builds its own pinned copy the same way, but scopes it to the MAIN module in SOURCE mode (`govulncheck ./...`, matching `.github/workflows/ci.yml`'s existing blocking main-module job per RESEARCH.md), NOT `vuln:`'s advisory `-mode=binary` scan over tool binaries — different scope, same build mechanism.

**Analog B (positive-count guard style, never a vacuous pass):** `proto:drift` (Taskfile.yml:369-480):
```yaml
# Source: Taskfile.yml:436-440 (the positive-count-floor pattern to copy)
echo "proto:drift: compared ${nfiles} generated files"
if [ "${nfiles}" -lt 4 ]; then
  echo "::error::proto:drift: enumerated only ${nfiles} ... — a broken enumeration must fail loud, never read as a clean pass"
  exit 1
fi
```
`check:gonum`'s SBOM-grep half and cgo-closure-scan half both need this same "report N, assert N > 0" floor before asserting the actual condition (D-14: "reports the number of packages inspected (> 0)").

**cgo-closure command, verified live this session (RESEARCH.md), copy literally:**
```bash
GOTOOLCHAIN=go1.26.6 go list -deps -f '{{.ImportPath}} {{len .CgoFiles}}' gonum.org/v1/gonum/graph/community
```
91 rows, every second column `0`. Positive control per D-14: a known-cgo package (e.g. `github.com/tree-sitter/go-tree-sitter`) must report a count > 0 in the same scan mechanism, proving the scan can actually detect cgo.

---

### `go.mod` (config, CRUD — one-line direct-require promotion)

**Analog:** `github.com/bmatcuk/doublestar/v4 v4.10.0` direct require (go.mod:10, no `// indirect` suffix) — the precedent for "a transitive dependency promoted to direct is a one-line diff, no version bump." `gonum.org/v1/gonum v0.17.0 // indirect` → `gonum.org/v1/gonum v0.17.0` is the same one-line edit; verify via `git diff go.mod` shows exactly one line changed (D-13).

---

### `web/src/lib/components/graph/file-graph-transform.ts` (utility, transform)

**Analog:** itself — `cycleId` field + `cycleDiscriminatorClass` (verified this session, lines 64-65, 108-109, 172, 177-178)

```ts
// Source: web/src/lib/components/graph/file-graph-transform.ts:64-65, 108-109
cycleId?: number;
// ...
function cycleDiscriminatorClass(cycleId: number): string {
    return `graph-cycle-${cycleId}`;
}
```
```ts
// Source: file-graph-transform.ts:172, 177-178 (fileNodeElement's shape)
cycleId: n.cycleId
// ...
if (n.cycleId !== 0) {
    return { data, classes: `${CYCLE_CLASS} ${cycleDiscriminatorClass(n.cycleId)}` };
}
```
Add `communityId?: number` as a peer field on `FileGraphNodeData`; a `communityDiscriminatorClass(communityId)` returning `graph-community-${(communityId - 1) % 12}` (D-10's palette indexing). Per D-11, directory/collapsed node elements (`expandedDirElement`/`collapsedDirElement`) must NOT gain a `communityId` field — only file-level nodes are coloured. The existing `cycleId`/`CYCLE_CLASS` composition (lines 177-178) shows exactly how a numeric wire field becomes a class list without disturbing existing classes — the community class composes alongside it, not replacing it.

---

### `web/src/lib/components/graph/graph-style.ts` (component, transform — no existing categorical `data()` colour mapper)

**Analog:** itself — the `.graph-cycle` selector family (lines 13-18, 141-158)

```ts
// Source: graph-style.ts:141-147, 155-158 (the shape to extend, NOT copy verbatim —
// this is a single shared style, community needs 12 DISTINCT per-index styles)
selector: 'node.graph-cycle',
// ...
selector: 'edge.graph-cycle',
```
No existing rule uses `data()`-driven categorical colour (the file's only `mapData` usage is for numeric edge width) — the closest reusable shape is the discriminator-class-per-id generation pattern from `file-graph-transform.ts`'s `cycleDiscriminatorClass`, extended here into 12 literal-hex Cytoscape style rules keyed by `node.graph-community-{0..11}` (never CSS custom properties — Cytoscape does not resolve `var()`). Per D-11, no directory-compound selector variant is added.

---

### `web/src/routes/graph/+page.svelte` (component, request-response — toolbar line)

**Analog:** itself — the `graph-cycle-summary` paragraph (verified this session, lines 563-573)

```svelte
<!-- Source: web/src/routes/graph/+page.svelte:563-573 -->
<p class="mt-1 text-xs text-muted-foreground" data-testid="graph-cycle-summary">
    {#if graphState.response.cycleCount > 0}
        This repository has {graphState.response.cycleCount} dependency cycle{graphState.response
            .cycleCount === 1
            ? ''
            : 's'}.
    {:else}
        This repository has no dependency cycles.
    {/if}
</p>
```
A sibling `data-testid="graph-community-summary"` paragraph reads `graphState.response.communityCount` the same way, no new RPC (D-12a). Placement is Claude's Discretion per CONTEXT.md — placing it adjacent to the cycle-summary paragraph mirrors this file's existing toolbar composition.

---

### `web/scripts/check-no-force-layout.mjs` (test, static source scan — new)

**Analog (script conventions, shebang/diagnostic style only — NOT a Playwright script):** `web/scripts/breadcrumb-check.mjs:1` shebang convention (`#!/usr/bin/env node`) and doc-header style (own-purpose comment naming what it proves and what it does NOT do, e.g. `graph-verdict.mjs`'s header: "This file does NOT measure anything... every ambiguity resolving to FAIL").

**Analog (positive-controlled structural scan shape, Go precedent to mirror in JS):** `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess` — same shape as Phase 10's `TestCoverageSourceNeverWalksDisk` (10-PATTERNS.md, reproduced here for the JS translation):
```js
// Analogous structure (Go original at internal/cli/editordiscovery_test.go,
// JS translation for this Node script):
// 1. glob/read every file under web/src
// 2. forbidden = ['cose', 'fcose', 'cola', 'euler', 'spread', 'force']
//    — fail if any match found (D-12b)
// 3. positive control: required = ['elk'] — fail if NOT found (an empty
//    glob must not read as a clean pass)
// 4. report both counts (elk-reference count, force-name-match count)
```
This is a pure static grep-style scan — no browser needed, unlike every other `web/scripts/*-check.mjs` file in this repo, which are Playwright-based. Do not copy `breadcrumb-check.mjs`'s `chromium`/`startCodegraphUi` machinery; only its header-comment and exit-code discipline.

---

### `11-MUTATION-LOG.md` (planning artifact, house format — values only)

**Analog:** `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md` (full structure read this session)

Copy the header shape (Phase/Date/Scope), the pre-mutation cleanliness gate convention:
```
Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` is
asserted to exit 0. ...
$ git diff --quiet -- internal/query/coverage.go internal/uiserver/readonly_test.go; echo $?
0
```
and one `## Family (x)` section per D-16 family: (a) seed/node-order perturbation (GRF-08), (b) threshold widened after measurement (GRF-09 ancestry), (c) force-layout name injected into `web/src` (GRF-06 scan), (d) cgo package injected into the scan's positive control (GRF-10) — each with an `**Instrument:**` command block and a `**Pre-mutation cleanliness gate:**` block, exactly Phase 10's shape.

---

### `11-SECURITY.md` (planning artifact, house format — values only)

**Analog:** `.planning/phases/10-index-health-the-coverage-denominator/10-SECURITY.md` (frontmatter + Trust Boundaries table read this session)

Copy frontmatter shape (`phase`, `slug`, `status`, `threats_open`, `asvs_level`, `created`) and the `## Trust Boundaries` table format:
```
| Boundary | Description | Data Crossing |
```
Phase 11's boundaries: (1) `internal/query.FileGraph()`'s own scanned data → `assignCommunities` (no external input — inputs are the store's own file paths and counts, per D-16); (2) `gonum.org/v1/gonum`'s import closure as a new trust surface (cgo absence, GRF-10); (3) wire crossing `community_id`/`community_count` from `internal/uiserver` to the browser (read-only, no new user-controlled input path, mirrors Phase 10's boundary 2 shape). `threats_open` must be honest per D-16 — do not default to 0 without the corresponding mitigation rows.

## Shared Patterns

### Fresh-per-call graph algorithm, no cache
**Source:** `internal/query/filegraph_cycles.go` (whole file) + `internal/query/traverse.go:311-330` call site
**Apply to:** `internal/query/community.go`, `internal/query/traverse.go` — no package-level cache, no persistence except the D-07 fallback slot.

### Additive-only proto/wire evolution + same-commit fixture recipe
**Source:** `internal/uiproto/uiv1/ui.proto`'s `cycle_id=4`/`cycle_count=6` history; `internal/uiserver/readonly_test.go`'s `uiProtoFieldFixtureLenAtPlanNNNN` chain; `09-01-SUMMARY.md`'s "additive-rpc + same-commit-fixture-move recipe"
**Apply to:** `ui.proto`, `internal/uiserver/handlers.go`, `internal/uiserver/readonly_test.go` — proto edit + `task proto:gen` + fixture + length-constant bump, ONE commit.

### Threshold-committed-alone, verdict-never-writes-threshold (GRF-01 protocol)
**Source:** `corpora/graph-render-threshold.json`, `web/scripts/graph-verdict.mjs`'s header doc ("This file does NOT measure anything... never writes to corpora/graph-render-threshold.json")
**Apply to:** `corpora/graph-cluster-threshold.json` + its Go-harness verdict logic — never amended/widened after measurement (D-05).

### Positive-controlled guard, never a vacuous pass
**Source:** `Taskfile.yml`'s `proto:drift` (`nfiles < 4` floor check); `internal/cli/editordiscovery_test.go`'s scan-with-required-substring shape
**Apply to:** `check:gonum`'s SBOM grep and cgo-closure scan; `web/scripts/check-no-force-layout.mjs`'s `elk`-found requirement; `community_test.go`'s ≥3-run-count assertion.

### git-shell-out test pattern
**Source:** `internal/query/engine_worktree_test.go:31-34` (`exec.Command("git", full...)`, `cmd.Dir`, `cmd.CombinedOutput()`)
**Apply to:** the new ancestry test proving the threshold commit is an ancestor of every measurement commit (D-08).

### Per-id discriminator class → static style rule set
**Source:** `web/src/lib/components/graph/file-graph-transform.ts`'s `cycleDiscriminatorClass`; `graph-style.ts`'s `.graph-cycle` selectors
**Apply to:** the new `communityDiscriminatorClass`/12-hue static selector set — literal hex values, no CSS custom properties, no runtime hash-to-colour function.

### House-format planning artifacts (values only, never invented structure)
**Source:** `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md`, `10-SECURITY.md`
**Apply to:** `11-MUTATION-LOG.md`, `11-SECURITY.md` — same frontmatter keys, same section headings, only the row/family content differs.

## No Analog Found

None. Every file this phase touches has at least a role-match analog in the codebase, consistent with RESEARCH.md's own claim ("every piece of this phase already has a direct, working precedent somewhere in this codebase"). The two genuinely novel engineering surfaces with no literal precedent to copy — (1) the RED-control seed/order-perturbation test seam for GRF-08, and (2) the GRF-09 Go-in-process clustering-time harness's exact CLI shape (`tools/corpora -mode cluster-measure` vs. a sibling `tools/graphcluster`) — are Claude's Discretion per `11-CONTEXT.md` and are called out inline above rather than forced onto a mismatched analog.

## Metadata

**Analog search scope:** `internal/query/`, `internal/uiproto/uiv1/`, `internal/uiserver/`, `tools/corpora/`, `corpora/`, `Taskfile.yml`, `go.mod`, `web/src/lib/components/graph/`, `web/src/routes/graph/`, `web/scripts/`, `.planning/phases/10-index-health-the-coverage-denominator/`
**Files scanned:** ~20 (via RESEARCH.md's prior verification plus this session's targeted confirmatory reads of `traverse.go`, `filegraph_cycles.go`, `filegraph_cycles_test.go`, `handlers.go`, `readonly_test.go`, `ui.proto`, `file-graph-transform.ts`, `graph-style.ts`, `+page.svelte`, `Taskfile.yml` (`vuln:`, `proto:drift`), `corpora/graph-render-threshold.json`, `graph-verdict.mjs`, `tools/corpora/main.go`, `engine_worktree_test.go`, `go.mod`, `10-MUTATION-LOG.md`, `10-SECURITY.md`)
**Pattern extraction date:** 2026-09-13

## PATTERNS COMPLETE
