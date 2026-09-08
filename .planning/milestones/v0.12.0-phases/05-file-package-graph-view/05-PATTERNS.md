# Phase 5: File/Package Graph View - Pattern Map

**Mapped:** 2026-08-30
**Files analyzed:** 8
**Analogs found:** 7 / 8 (1 no-analog — new client dependency, GraphCanvas.svelte)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/query/traverse.go` (add `Engine.FileGraph()`) | service (graph rollup) | batch/transform (full scan → aggregation) | `BuildReverseAdjacency` in same file (`traverse.go:31-50`) | exact (same file, same discipline, D-05) |
| `internal/query/filegraph_cycles.go` (new) | utility (algorithm) | transform | none in-repo (no existing Tarjan/SCC) | no analog — build per RESEARCH's Code Examples section |
| `internal/uiproto/uiv1/ui.proto` (add `FileGraph` rpc + messages) | config/schema | request-response | `GetHealthResponse` block (`ui.proto:701-736`) | exact (map<string,int64> precedent, additive-rpc precedent) |
| `internal/uiserver/handlers.go` (add `FileGraph` handler + `fileGraphToProto` mapper) | controller/handler | request-response | `GetHealth` handler (`handlers.go:955-978`) + `healthToProto` (`handlers.go:892-909`) | exact (ordinary `withEngine` shape, named mapper convention) |
| `internal/uiserver/readonly_test.go` (extend fixtures) | test (fixture) | n/a | `wantUIServiceMethods` map + count (`readonly_test.go:39-57`) | exact — value-only edit, MUST NOT touch `mutatingVerbs` |
| `web/src/routes/graph/+page.svelte` | route/component | request-response (mount → RPC) | `web/src/routes/workbench/+page.svelte` | role-match (closest full-route implementation; Browse's URL-state pattern applies too) |
| `web/src/lib/components/graph/GraphCanvas.svelte` (new) | component (imperative-lib wrapper) | event-driven (pan/zoom/tap) | `web/src/lib/components/workbench/DataTable.svelte` (`$effect`-based `createVirtualizer` wiring, lines 95-123) | role-match — same "wrap imperative JS lib in Svelte 5 runes" shape; no cytoscape usage exists yet, so this is a structural analog only |
| `web/src/lib/components/graph/file-graph-transform.ts` (new, pure fn) | utility (transform) | transform | `web/src/lib/browse-url.ts` (pure parse/serialize module, no DOM) | role-match (pure TS module with a documented single-responsibility header comment convention) |
| `web/tests/graph-*.test.ts` (new) | test | n/a | `web/tests/workbench-callers-callees.test.ts`, `web/tests/data-table-virtualization.test.ts` | exact (test-location convention: `web/tests/`, not beside source) |

## Pattern Assignments

### `internal/query/traverse.go` — `Engine.FileGraph()` (service, batch/transform)

**Analog:** `BuildReverseAdjacency`, same file, lines 14-50.

**Discipline to copy verbatim** (doc-comment convention + fresh-per-call, no cache):
```go
// BuildReverseAdjacency builds an in-memory reverse-adjacency map keyed
// by edge.Target, from one full IterateEdges("") scan (D-04). ...
// This is built fresh inside every caller ... no package-level cache,
// no sync.Once ... a long-lived process ... must never serve a stale
// point-in-time reverse view across multiple calls.
func BuildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	rev := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		rev[e.Target] = append(rev[e.Target], e)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return rev, nil
}
```

**Deviation required (D-09):** `FileGraph()` needs a SECOND scan (`IterateNodes()`) before the edge scan, because `schema.Edge` carries no file path — only `schema.Node.FilePath` does, and node IDs are opaque content hashes. `BuildReverseAdjacency`'s single-scan shape does not transfer; copy the discipline (fresh, `defer it.Close()`, `if err := it.Err()`), not the scan count. See RESEARCH.md "Pattern 1" for the full two-scan code shape, including the `n.Kind == "package"` exclusion (D-08) and the `sf == tf` self-edge / `e.Kind == "contains"` exclusion (D-03).

**Test-seam convention** (optional, if a call-count invariant is needed): `traverse.go`'s file also documents an unexported package-var indirection (`buildReverseAdjacency`) used elsewhere in `internal/query` so tests can count invocations without an exported setter — reuse this convention only if a plan requires asserting single-scan-per-call for `FileGraph`, mirroring `internal/graphstore/pebble_store.go`'s `openLockRetrySleep` seam.

---

### `internal/query/status.go` — `edgesByKind` (secondary analog for the per-kind-tally shape)

**Analog:** `status.go` doc comment table + the `EdgesByKind map[string]int64` field (`status.go:45,58` per RESEARCH), same "read-time-derived, sparse map, no Meta persistence" convention `FileGraphEdge.kind_counts` should follow.

**Sparse-map convention to copy:** "a kind with zero observed edges is absent from the map, never present with value 0" — apply the same rule to each aggregated `FileGraphEdge.kind_counts`.

---

### `internal/query/filegraph_cycles.go` (new) — cycle detection (utility, transform)

**No in-repo analog.** Confirmed via search: no `Tarjan`/`SCC`/`StronglyConnected` symbol exists anywhere in `internal/query` or `internal/indexer`. Build from RESEARCH.md's "Tarjan's SCC over the aggregated file-level adjacency" Code Example — a plain `map[string][]string` adjacency built from `FileGraph`'s distinct `(src,tgt)` pairs, kind-agnostic, reporting only SCCs of size ≥ 2 as cycles (self-edges already excluded at aggregation time per D-03, so a size-1 SCC is never a cycle). Keep it in its own file per RESEARCH's Recommended Project Structure — "kept separate for unit-testability independent of the rollup scan itself."

---

### `internal/uiproto/uiv1/ui.proto` — new `FileGraph` rpc + messages (config, request-response)

**Analog:** `GetHealthResponse`, `ui.proto:701-736`.

**Map-field precedent** (copy exactly for `FileGraphEdge.kind_counts`):
```protobuf
// edges_by_kind mirrors StatusResult.EdgesByKind.
map<string, int64> edges_by_kind = 11;
```

**Rpc declaration precedent** (`ui.proto:80`):
```protobuf
rpc GetHealth(GetHealthRequest) returns (GetHealthResponse);
```
Copy this shape for `rpc FileGraph(FileGraphRequest) returns (FileGraphResponse);` — additive-only (D-02a), field numbers pinned at a `blocking-human checkpoint:decision` before codegen, exactly as Phase 4's GetHealth required (see `04-CONTEXT.md` D-01). **Before freezing the rpc name**, check it against all 19 substrings in `readonly_test.go`'s `mutatingVerbs` fixture (RESEARCH already confirmed "FileGraph" is clean, but the planner/implementer must re-verify at freeze time, not trust this document).

Recommended message shape (RESEARCH "Code Examples", not yet locked — planner schedules the checkpoint):
```protobuf
message FileGraphEdge {
  string source_file = 1;
  string target_file = 2;
  map<string, int64> kind_counts = 3; // mirrors ui.proto:736's exact precedent
}
```

---

### `internal/uiserver/handlers.go` — `FileGraph` handler + `fileGraphToProto` mapper (controller, request-response)

**Analog:** `GetHealth` (`handlers.go:955-978`) is the ORDINARY `withEngine` shape to copy — `GetStatus` is documented as "the single, deliberate exception to the withEngine rule" (line 291) and must NOT be copied.

**Handler pattern** (copy this shape, substituting `eng.FileGraph()` for `eng.Status(ctx)`):
```go
func (s *uiService) GetHealth(ctx context.Context, _ *connect.Request[uiv1.GetHealthRequest]) (*connect.Response[uiv1.GetHealthResponse], error) {
	var resp *uiv1.GetHealthResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Status(ctx)
		if err != nil {
			return err
		}
		// ... additional engine calls as needed ...
		resp = healthToProto(result, commitSHA)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
```

**Named-mapper convention** (copy for `fileGraphToProto`): `healthToProto` (`handlers.go:892-909`) — "a named mapper, never an inline literal at the handler call site, so the mapping cannot drift from its source silently." Field set/numbering frozen at a maintainer checkpoint, same as `healthToProto`'s own doc comment records.

**Simpler request-with-params sibling analog:** `Files` (`handlers.go:431-455`) shows the shape when the rpc takes request fields through to an `Engine` options struct (`query.FilesOptions{...}`) — useful if `FileGraphRequest` ends up taking any filter/limit params, though current RESEARCH implies a parameterless `FileGraphRequest{}`.

---

### `internal/uiserver/readonly_test.go` — extend `wantUIServiceMethods` (test fixture, value-only edit)

**Analog:** the same file, lines 39-57 (already extended three times: GetPermalink at 03-05, GetHealth at 04-03).

**Value-only diff to make:**
```go
var wantUIServiceMethods = map[string]struct{}{
	"GetStatus":     {},
	"Search":        {},
	"Files":         {},
	"Callers":       {},
	"Callees":       {},
	"Impact":        {},
	"Affected":      {},
	"GetNodeDetail": {},
	"Explore":       {},
	"GetPermalink":  {},
	"GetHealth":     {},
	"FileGraph":     {},   // <-- add, 12th read-only rpc
}
```
Also bump the doc comment's stated count ("same length (11, as of plan 04-03's GetHealth)" → 12) and add an "UPDATED at plan 05-XX" paragraph following the exact prose convention of the GetPermalink/GetHealth updates above it.

**MUST NOT touch:** `mutatingVerbs` (lines 89-102) — per repo-specific instructions, this fixture must never be weakened. `FileGraph` was confirmed clean against all 19 substrings by RESEARCH; do not add exceptions to this list to accommodate a name collision — pick a different rpc name instead if one ever collides.

---

### `web/src/routes/graph/+page.svelte` — fills the Phase-2 placeholder (route/component)

**Current placeholder content (MUST be replaced, not merely extended):**
```svelte
<!--
  Placeholder slot for Phase 5 (File/Package Graph View): the whole
  repository as one readable picture at file/package granularity. This
  route is a real client-side page — not a restructuring point — so
  Phase 5 fills it rather than building navigation from scratch (D-18).
-->
<h1 class="text-lg font-semibold">Graph</h1>
<p class="mt-1 text-sm text-muted-foreground">
	Phase 5: the whole repository as one readable picture at file/package granularity.
</p>
```
The `<p>` text is internal planning vocabulary (exactly the class of defect Phase 4 shipped and CONTEXT.md's `<specifics>` section calls out by name) and MUST be replaced with real user-facing copy.

**Analog for full-route structure:** `web/src/routes/workbench/+page.svelte` (lines 1-50+) — reads view state exclusively from `page.url.searchParams` via a dedicated `-url.ts` module (`parseWorkbenchParams`/`serializeWorkbenchParams`), writes only through `replaceState`, never direct state assignment. If `/graph` needs URL-addressable state (selected/expanded file, layout mode), follow this exact pattern rather than inventing a third URL module — mirrors `browse-url.ts`'s "ONE URL grammar per view" rule (D-13).

**`$effect`-gated per-tab/per-panel mount discipline** (if the graph view ever gets multiple modes): `workbench/+page.svelte`'s comment on `{#if}`-gating `Tabs.Content` to avoid all-panels-mounted-at-once request fan-out is directly relevant if `/graph` grows a similar multi-mode surface later — not required for a single-canvas MVP.

---

### `web/src/lib/components/graph/GraphCanvas.svelte` (new) — GRF-05's swap seam (component, event-driven)

**No cytoscape usage exists yet in this repo — role-match only, not exact.** Closest structural analog is `web/src/lib/components/workbench/DataTable.svelte`'s `$effect`-based wiring of `@tanstack/svelte-virtual`'s `createVirtualizer` (lines 95-123) — the same "construct an imperative JS library instance inside `$effect`, keep it in sync with reactive props, tear it down on cleanup" shape RESEARCH's own Pattern 2 example already models cytoscape on.

**Pattern to copy (imperative-lib construction inside `$effect`, options re-synced on prop change):**
```svelte
$effect(() => {
	const rowCount = table.getRowModel().rows.length;
	const container = scrollContainer;
	get(rowVirtualizer).setOptions({
		count: rowCount,
		getScrollElement: () => container,
		estimateSize: () => ROW_HEIGHT_PX,
		overscan: OVERSCAN,
		initialRect: { width: 0, height: CONTAINER_HEIGHT_PX }
	});
});
```
RESEARCH's own recommended `GraphCanvas.svelte` shape (construct-in-`$effect`, `return () => cy.destroy()` for cleanup) is the direct cytoscape equivalent — copy DataTable's comment discipline (explain WHY the effect re-runs, note any store-vs-rune subtlety) alongside it, since that repo convention (explaining non-obvious reactive-library interop) is itself the pattern worth reusing, not just the code shape.

**jsdom test-stub precedent** (`web/tests/setup.ts:9-44`, referenced in DataTable's own comment) — the scoped `offsetWidth`/`offsetHeight` stub used for `data-table-scroll` is the precedent for a `GraphCanvas`-scoped stub if one is needed, but per RESEARCH Pitfall 4, actual canvas rendering/pan-zoom is NOT testable this way — budget manual browser UAT instead, per the phase's own `<specifics>` mandate.

---

### `web/src/lib/components/graph/file-graph-transform.ts` (new) — pure transform (utility)

**Analog:** `web/src/lib/browse-url.ts` — pure TS module, zero DOM dependency, documented single-responsibility header comment ("browse-url.ts is the ONE URL grammar for the Browse view... every later plan... extends this module rather than inventing a second one").

**Convention to copy:** the doc-comment-as-contract style — state up front what this module does NOT do (mirrors browse-url.ts's "this module parses SHAPE only... never validates a value's RANGE"). For `file-graph-transform.ts`, the equivalent contract per RESEARCH's anti-patterns is: this module MUST NOT compute cycle membership (server-side only, D-06) — it only maps already-computed `FileGraphResponse` cycle flags onto a CSS class, and it MUST NOT touch the DOM/Cytoscape API directly (that's `GraphCanvas.svelte`'s exclusive job, per GRF-05's seam).

---

### `web/tests/graph-*.test.ts` (new) — test-location convention

**Analog:** all 26 existing files directly under `web/tests/` (e.g., `workbench-callers-callees.test.ts`, `data-table-virtualization.test.ts`, `browse-url.test.ts`). Tests live in `web/tests/`, never beside source (`web/src/lib/components/graph/*.test.ts` would violate this convention). Use `web/tests/fixtures/` and `web/tests/support/` (both already present) for any new fixture data or test helpers, following the existing directory split rather than inventing new locations.

## Shared Patterns

### `withEngine` handler shape
**Source:** `internal/uiserver/handlers.go:51` (definition), `GetHealth` (955-978) as the model call site.
**Apply to:** the new `FileGraph` handler. Do NOT model on `GetStatus` (line 291), which is documented as "the single, deliberate exception."

### Additive-only proto + blocking-human checkpoint before codegen
**Source:** `ui.proto`'s `GetHealthResponse` field-numbering history (04-03's checkpoint, referenced in `handlers.go:895-899`'s doc comment) and CONTEXT.md D-02a.
**Apply to:** the new `FileGraphRequest`/`FileGraphResponse`/`FileGraphEdge` messages and the `FileGraph` rpc declaration — schedule the same `blocking-human checkpoint:decision` gate before field numbers freeze.

### Fresh-per-call, no precomputed projection
**Source:** `BuildReverseAdjacency`'s doc comment (`traverse.go:14-29`) and `edgesByKind`'s doc-comment table row in `status.go`.
**Apply to:** `Engine.FileGraph()` in full — this is ENG-03/D-05/D-09 as written, not a style choice.

### Sparse per-kind map convention
**Source:** `status.go` EdgesByKind doc row: "a kind with zero observed edges is absent from the map, never present with value 0"; `ui.proto:735-736`'s `edges_by_kind`.
**Apply to:** `FileGraphEdge.kind_counts`.

### readonly_test.go fixture extension discipline
**Source:** `readonly_test.go:39-57`'s prose history (three prior extensions, each documented with an "UPDATED at plan X" paragraph explaining why the new rpc belongs in the read set).
**Apply to:** the `FileGraph` addition to `wantUIServiceMethods` — write the same class of justification paragraph; leave `mutatingVerbs` untouched.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/query/filegraph_cycles.go` | utility | transform | No Tarjan/SCC/StronglyConnected implementation exists anywhere in the repo (confirmed via search) — build from RESEARCH.md's Code Examples section, not from an in-repo analog. |
| `web/src/lib/components/graph/GraphCanvas.svelte` | component | event-driven | No cytoscape (or any canvas/graph-rendering imperative library) is wired anywhere in this codebase today. `DataTable.svelte`'s `createVirtualizer` wiring is the closest structural shape (imperative-lib-in-`$effect`) but is a different library entirely — treat as a pattern-of-approach match, not a code-reuse match. |
| `web/package.json` (dependency additions: `cytoscape`, `cytoscape-elk`) | config | n/a | New dependency category (canvas/graph-rendering); no existing entry of this kind to pattern-match against. Follow RESEARCH's Package Legitimacy Audit findings (both `cytoscape` and `cytoscape-dagre` flagged `[SUS]`/`too-new` false-positive, requiring `checkpoint:human-verify`) rather than an in-repo precedent. |

## Metadata

**Analog search scope:** `internal/query/`, `internal/uiserver/`, `internal/uiproto/uiv1/`, `web/src/routes/`, `web/src/lib/`, `web/tests/`
**Files scanned:** `traverse.go`, `status.go`, `handlers.go`, `readonly_test.go`, `ui.proto`, `graph/+page.svelte`, `workbench/+page.svelte`, `browse-url.ts`, `DataTable.svelte`, `web/tests/` directory listing
**Pattern extraction date:** 2026-08-30
