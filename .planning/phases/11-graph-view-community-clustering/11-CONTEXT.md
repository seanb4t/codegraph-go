# Phase 11: Graph View — Community Clustering - Context

**Gathered:** 2026-09-13
**Status:** Ready for planning
**Mode:** Smart discuss (autonomous) — four grey areas proposed with the full question set inside each prompt; every recommended answer accepted by the user

<domain>
## Phase Boundary

The file/package graph reads as groups rather than a flat mesh: nodes coloured by community, computed **fresh and deterministically inside `FileGraph()`** (the `CycleID` precedent, maintainer decision 2026-09-08), rendered as colouring on the **existing ELK layered layout** with the layout itself unchanged.

1. **GRF-09** — the performance premise is *measured, not assumed*: a clustering-time threshold committed alone before any clustering code, following GRF-01's protocol, with the index-time-persistence fallback written down before the measurement runs.
2. **GRF-06 / GRF-08** — Louvain community detection (gonum) on an undirected weighted file graph, deterministic by construction (sorted input, fixed seed, canonical relabeling), surfaced additively on the wire and rendered as colour.
3. **GRF-10** — the milestone's only new direct `go.mod` require passes govulncheck, appears by name in the SBOM, and its import closure is proven cgo-free by a positive-controlled scan.

Out of scope by construction: any force-directed layout (rejected repeatedly across v0.12.0); persisted cluster assignments as the default (retained only as GRF-09's documented fallback); a clustering cache (see D-15); colouring collapsed directory compounds (D-11). `gonum` enters as a direct require at the version already in `go.sum` — no new module, no version bump.

</domain>

<decisions>
## Implementation Decisions

### Algorithm & determinism (GRF-06 / GRF-08)

- **D-01:** Community detection is **Louvain modularity via `gonum.org/v1/gonum/graph/community.Modularize`** (already pinned at v0.17.0 in `go.sum` transitively via sigstore; promoted to a direct require, D-12). Input is an **undirected weighted** graph built from `FileGraph()`'s aggregated file-pair edges, weight = `FileGraphEdge.TotalCount`, resolution `1.0`. The documented zero-dependency fallback is a hand-rolled label-propagation implementation, used **only** if gonum's Louvain proves non-deterministic under D-02's sorted input or GRF-10's review rejects gonum. — **Reversibility:** costly (an algorithm swap changes every rendered assignment).
- **D-02:** Determinism is guaranteed by **three mechanisms, all required**: (a) nodes are inserted into the gonum graph in **sorted file-path order** with sequential `int64` IDs — no Go map-iteration order ever reaches the algorithm; (b) a **fixed seed** (`rand.NewSource(<committed constant>)`) drives Louvain's move ordering; (c) **canonical relabeling** after the run — communities are renumbered `1..N` by their smallest member path, so equal partitions compare equal regardless of gonum's internal numbering. A fixed seed alone is *not* sufficient (gonum's `simple` graphs iterate Go maps internally).
- **D-03:** Every file node gets **exactly one** community (a singleton is its own community). IDs are **1-based canonical**; `0` means "not computed" and is **never emitted** on the fresh-compute path — this mirrors `CycleID`'s "0 = none" convention while keeping "no community" and "not computed" distinguishable on the proto default.
- **D-04:** GRF-08's proof: a test runs the clustering **≥ 3 times** on identical input, asserts label-canonicalized equality, and **reports the run count** (positive assertion); plus a **RED control** — perturbing the node insertion order or the seed (through a test-only hook) must turn the test red, recorded in `11-MUTATION-LOG.md`, not merely claimed.

### Threshold & measurement protocol (GRF-09)

- **D-05:** A new **`corpora/graph-cluster-threshold.json`** is committed **alone as the phase's first commit** (its own plan/task, before any clustering code exists), mirroring `corpora/graph-render-threshold.json` (exactly one commit, ancestor of every measurement commit). The verdict script reads every value from it and **never writes it**; amending, widening or relaxing any value after measurement is forbidden (the GRF-01 `T-05-01` discipline). — **Reversibility:** one-way by design — that is the point of the protocol.
- **D-06:** The **binding metric is clustering time in isolation**: from `FileGraph()`'s rollup complete to community assignment complete, measured in-process by a Go harness against the **pinned guava corpus** (`google/guava@94f39958baf7ad51ddf9c70e406ed6b188194daa`, 3,233 file nodes / 21,554 file-pair edges, already indexed under `~/.cache/codegraph/corpora/`), **median of 3 cold runs, max 500 ms** — well above Louvain's expected tens of milliseconds at that scale, well below GRF-01's 5,000 ms time-to-interactive budget. Total `FileGraph()` wall time and node/edge counts are recorded **non-binding**.
- **D-07:** `onFailure` is written **into the threshold file before measuring**: index-time persistence into `Node`'s reserved 50–59 range (`community_id = 50`), computed at commit time and read back by `FileGraph()` — the maintainer's rejected default becomes the documented fallback, **never a raised bar**.
- **D-08:** The measurement lands as `corpora/graph-cluster-observations.json` plus a verdict artifact whose PASS/FAIL is **preserved verbatim whichever way it falls**; git ancestry (threshold commit is an ancestor of every observation commit) is how the ordering is proven — checked by a test, not asserted in prose. No clustering may be wired into the UI before this verdict resolves (ROADMAP blocking note).

### Wire & rendering (GRF-06)

- **D-09:** Additive wire: `FileGraphNode` gains **`int32 community_id = 5`** (next free after `cycle_id = 4`) and `FileGraphResponse` gains **`int32 community_count`** (next free field after `cycle_count = 6`, i.e. `7`) so the UI reports the count without recounting. No other proto change; the field-number fixture and `wantUIServiceMethods` (unchanged at 16) are updated in the **same commit** as the proto per the 09-01/10-01 recipe. — **Reversibility:** one-way once shipped.
- **D-10:** Rendering = **node background colour keyed by `community_id`** from a **deterministic categorical palette** (colour-blind-safe, ~12 hues, index = `(id − 1) mod 12`, cycling beyond), applied on the **existing ELK layered layout with the layout configuration untouched**; the existing `cycleId` group styling stays as-is and composes with the colour.
- **D-11:** Collapsed **directory compound nodes stay neutral** (border only) — a directory that mixes communities must not lie about its contents; only file-level nodes are coloured.
- **D-12:** The GRF-06 check has three parts: (a) a toolbar/legend line **"N communities"** fed by `community_count` — the check that *reports* the number rendered; (b) a **positive-controlled source scan** over `web/src` that finds ≥ 1 `elk` layout reference and **zero** references to any force-directed layout name (`cose`, `fcose`, `cola`, `euler`, `spread`, `force`) — no force mode reachable from any code path; (c) a vitest rendering the transform against a fixture asserting distinct colours == distinct community ids.

### Supply chain (GRF-10) & recompute cadence

- **D-13:** `gonum` is promoted from transitive to a **direct `require gonum.org/v1/gonum v0.17.0`** in `go.mod` — the version already pinned in `go.sum`; the `go mod tidy` diff is reviewed to be that one line (no new module enters the graph, no version bump).
- **D-14:** GRF-10 is a **Taskfile target** (e.g. `check:gonum`) with three positive-controlled halves: `govulncheck ./...` over the main module (reachability-aware; must report gonum in the scanned set); an SBOM generated locally the way the release does (syft / cyclonedx-gomod) and grepped for `gonum.org/v1/gonum` with a positive control on a known module; and a cgo-closure scan via `go list -deps -f` over `gonum.org/v1/gonum/graph/community`'s import closure that **reports the number of packages inspected (> 0)** and asserts zero with `CgoFiles` or `import "C"` — an empty closure cannot read as clean.
- **D-15:** Recompute cadence, recorded explicitly: **fresh on every `FileGraph()` call** — hence every graph route load and every live-push refresh — with **no caching** in this phase. GRF-09 is exactly the measurement that makes this affordable; a cache would reopen the persistence question the maintainer settled. The 50–59 fallback (D-07) is the only persistence path, and only if GRF-09 fails.
- **D-16:** Phase artifacts in the house format: `11-MUTATION-LOG.md` (perturbed order/seed RED for D-04; threshold-widening RED via the verdict script for D-05; a force-layout name planted in `web/src` RED for D-12b) and `11-SECURITY.md` (the new dependency's trust surface, cgo absence, no user-controlled input reaches the algorithm — inputs are the store's own file paths and counts), every row test-or-verdict, `threats_open` honest.

### Claude's Discretion

- The committed seed constant, the exact palette values (pick a colour-blind-safe categorical set; the `dataviz` skill's palette is acceptable), and where the "N communities" line sits in the graph toolbar.
- Whether the Go measurement harness is a `go test -run` with a build tag (like `test/tmux`) or a `tools/` command; it must be re-runnable and must read the threshold file rather than carrying defaults.
- The name of the GRF-10 Taskfile target and whether the SBOM half uses syft or cyclonedx-gomod (match what `.goreleaser.yaml` uses).
- Whether `community_count` counts singletons (recommend yes — every node is in exactly one community, so N = number of distinct ids).

</decisions>

<canonical_refs>
## Canonical References

### Phase scope and requirements
- `.planning/ROADMAP.md` → `### Phase 11: Graph View — Community Clustering` — goal, four success criteria, the blocking-within-the-phase note, the research flag.
- `.planning/REQUIREMENTS.md` — GRF-06, GRF-08, GRF-09, GRF-10.

### The precedents this phase repeats
- `internal/query/traverse.go:80-110, 172-330` — `FileGraphNode.CycleID`, `FileGraphEdge`, `FileGraph()`, the cycle detector populating `CycleID` fresh per call.
- `internal/query/filegraph_cycles_test.go` — `TestFileGraphCyclesDeterministicIds` (the determinism-test shape to mirror).
- `corpora/graph-render-threshold.json` (exactly one commit) + `web/scripts/graph-verdict.mjs` (fail-closed comparator that never writes the threshold) + `web/scripts/graph-measure.mjs` — GRF-01's protocol.
- `.planning/milestones/v0.12.0-phases/05-file-package-graph-view/05-CONTEXT.md` — the threshold-before-measurement decisions (locked in 05-01, never widened, onFailure written first).
- `corpora/manifest.json`, `corpora/selection.json` — the guava pin (`94f39958…`) and its measured edge counts; local cache `~/.cache/codegraph/corpora/google-guava-2b0cb53f@94f39958…`.

### Wire and UI
- `internal/uiproto/uiv1/ui.proto` — `FileGraphNode` (fields 1–4), `FileGraphResponse` (fields 1–6), `reserved 50 to 59` mirrors.
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` (16), the field-number fixture.
- `web/src/routes/graph/+page.svelte` — the cycleId grouping on element data; `web/src/lib/components/graph/GraphCanvas.svelte`, `graph-style.ts`, `file-graph-transform.ts` — cytoscape + `cytoscape-elk` layered layout and node styling.

### Store and schema
- `internal/schema/graph.proto` — `Node` `reserved 50 to 59` (the D-07 fallback slot).
- `gonum.org/v1/gonum/graph/community` (`louvain_common.go:90` `Modularize(g, resolution, src rand.Source)`), `graph/simple` for the weighted undirected graph.

### Supply chain
- `Taskfile.yml` `vuln:` (govulncheck precedent), `.goreleaser.yaml` `sboms:` block (syft), `go.mod` / `go.sum`.

### Guard and proof precedents
- `.planning/phases/10-index-health-the-coverage-denominator/10-MUTATION-LOG.md`, `10-SECURITY.md` — house formats.
- Repo rule `84d1gfpywd` — every guard carries a positive assertion that it did its work.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `FileGraph()` already produces the exact input Louvain needs (file nodes + aggregated weighted file-pair edges) and already runs a fresh per-call graph algorithm (the cycle detector) — the community pass slots in after it and populates a sibling field.
- `graph-verdict.mjs` is a fail-closed threshold comparator that can be generalised or duplicated for the cluster threshold; `graph-measure.mjs` shows the observation-record shape.
- The `cycleId` grouping in `+page.svelte` shows how per-node numeric data reaches styling without class lists.
- `gonum` v0.17.0 is already in the module graph — GRF-10's work is promotion + proof, not introduction.

### Established Patterns
- Threshold committed alone, observation preserved verbatim, verdict script never writes the threshold (GRF-01).
- Additive-only wire evolution with same-commit fixture updates (09-01, 10-01).
- Positive-controlled guards; mutation logs demonstrate RED before a gate is called fixed.
- Read-only-by-construction: nothing here mutates the store on the fresh path.

### Integration Points
- Engine: `internal/query/traverse.go` `FileGraph()` → new `community.go` (sorted graph build, Louvain, canonical relabel) → `FileGraphNode.CommunityID`, `FileGraphResult.CommunityCount`.
- Wire: `ui.proto` → `internal/uiserver` FileGraph handler mapping → generated TS → `file-graph-transform.ts` → `graph-style.ts` palette → `GraphCanvas.svelte`; toolbar count line in `+page.svelte`.
- Measurement: Go harness over the cached guava index → `corpora/graph-cluster-observations.json` → verdict; test proving threshold-commit ancestry.
- Supply chain: `go.mod` direct require → `check:gonum` Taskfile target (govulncheck + SBOM grep + cgo-closure scan).

</code_context>

<specifics>
## Specific Ideas

- The threshold file's `purpose` string should say, as GRF-01's does, that amending/widening after measurement is forbidden, and name the fallback (`Node` field 50) explicitly.
- Canonical relabeling by *smallest member path* makes assignments stable across runs AND readable in tests (community 1 is always the one containing the lexically-first file).
- The force-layout source scan must carry a positive control (`elk` found ≥ 1) or an empty `web/src` glob would pass it.

</specifics>

<deferred>
## Deferred Ideas

- Colouring collapsed directory compounds by majority community (D-11 keeps them neutral) — revisit only with a UI decision about mixed-community directories.
- An in-process clustering cache keyed by index generation (D-15 rejects it for this phase; the `coverage_generation` counter from Phase 10 would be the obvious key if it is ever wanted).
- A visible colour → community legend beyond the count line.
- Persisted cluster assignments as the default (out of scope by construction; fallback only).

</deferred>
