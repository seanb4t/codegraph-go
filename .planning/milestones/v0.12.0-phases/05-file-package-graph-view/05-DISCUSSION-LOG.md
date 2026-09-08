# Phase 5: File/Package Graph View - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-30
**Phase:** 05-file-package-graph-view
**Areas discussed:** GRF-01 pass condition, edge-kind rollup, GRF-03 drill-down, GRF-04 cycle placement, and a follow-up on the compound-node/GRF-01 tension

---

## Grounding performed before questions

Measured live against this repository's own index rather than assumed:
`nodeCount 7052 / edgeCount 16565 / fileCount 573`, and the full `edgesByKind` tally
(`calls` 6948, `contains` 6406, `references` 1695, `instantiates` 731, `imports` 461,
`returns` 215, `implements` 57, `type_of` 4, `extends` 2, `embeds` 1, `overrides` 1).

Read `BuildReverseAdjacency` (`traverse.go:31-43`) as the discipline ENG-03 names, the
edge-kind constants in `goextract/types.go`, `corpora/manifest.json`'s nine repositories,
and the current `/graph` placeholder route.

Todo cross-reference: 4 matches, highest score 0.6 (`release:dry-run-signed`), all in
release/ci/testing areas unrelated to this phase. **None folded.**

---

## GRF-01 pass condition

| Option | Description | Selected |
|--------|-------------|----------|
| Interaction latency at the largest corpus | Threshold on time-to-first-paint and pan/zoom frame time for the rollup of the largest corpus; tie-breaks bundle size then layout API fit | ✓ |
| Node/edge count thresholds only | Record counts, choose by published scale guidance | |
| Hierarchical-layout capability first, scale second | Primary condition is native hierarchical/compound layout; latency as tie-break | |

**User's choice:** Interaction latency at the largest corpus (recommended).
**Notes:** Chosen because it measures the property criterion 2 actually states — a developer opens it and can read it — rather than a proxy. The count-only option was noted as deciding a rendering question without rendering anything.

---

## Edge-kind rollup

| Option | Description | Selected |
|--------|-------------|----------|
| Exclude `contains`, keep the rest | 6406 `contains` edges become self-edges at file granularity | ✓ |
| Only `calls` and `imports` | Cleanest picture, 7409 of 16565 edges | |
| All 11 kinds, self-edges dropped at rollup | Complete, but `contains` then contributes almost nothing | |
| Decide during planning from GRF-01's data | Defer per the ROADMAP Notes | |

**User's choice:** Exclude `contains`, keep the rest (recommended).
**Notes:** Self-edges are still dropped for the retained kinds. The exact aggregation tuple stays pinned during planning per the ROADMAP.

---

## GRF-03 drill-down

| Option | Description | Selected |
|--------|-------------|----------|
| Deep-link into Browse | `/browse?file=…` — zero new UI, reuses Phase 3's file view and NAV-01 | |
| In-place expansion in the graph | File expands into symbol nodes within the same picture | ✓ |
| Side panel, graph stays put | Panel beside the graph listing symbols | |

**User's choice:** In-place expansion — **diverging from the orchestrator's recommendation.**
**Notes:** This choice triggered the follow-up below. The orchestrator had flagged at option-time that in-place expansion is "materially more renderer-specific work, and makes the renderer choice harder to swap behind GRF-05's seam"; the follow-up established that the consequence was larger than that phrasing conveyed.

---

## Follow-up: the compound-node / GRF-01 tension

**Raised by the orchestrator after the drill-down choice, not asked as part of the original set.**

Research performed before raising it, so the concern was grounded rather than speculative:

- **Cytoscape.js** has first-class compound nodes — `parent` field in node data, parent dimensions auto-inferred from descendants, `node.parent()` / `eles.move()`. Compounds form a strict tree, one parent maximum.
- **Sigma.js** does not, and its maintainers declined to add them. From `jacomyal/sigma.js#1201`: the feature *"is too complex and too opinionated"* for Sigma's core and belongs in an external plugin. The only community approach shown is a hand-rolled parallel tree hiding the parent and un-hiding children, whose author concedes it *"doesn't really implement that nice grouping behavior"* and leaves inter-group edges and intra-group placement unsolved.

**The problem:** in-place expansion hard-requires compound nodes, which effectively pre-selects Cytoscape — hollowing out GRF-01, a spike that exists specifically so the renderer is *"chosen by recorded measurement rather than by assumption."*

| Option | Description | Selected |
|--------|-------------|----------|
| Make compound support a locked prerequisite, stated openly | Record the narrowing; GRF-01 measures whether the qualifying renderer stays interactive, with a real failure branch | ✓ |
| Keep GRF-01 fully open; drop in-place expansion | Revert GRF-03 to deep-link or side panel so neither renderer is excluded on capability | |
| Let the spike measure BOTH, compound included | Cytoscape's native compounds vs a hand-rolled Sigma equivalent, on latency and implementation cost | |
| Defer the drill-down decision until after GRF-01 | Lock only pass condition and rollup now; decide GRF-03 once the winning renderer is known | |

**User's choice:** Make compound support a locked prerequisite, stated openly.
**Notes:** The narrowing is recorded in CONTEXT.md D-02 in plain terms, including the instruction that downstream agents must not describe the renderer selection as an open two-way comparison. GRF-01 retains a genuine pass/fail — if the qualifying renderer cannot stay interactive at the largest corpus, that failure must be reachable and actionable, forcing reconsideration of the renderer or of the drill-down decision.

---

## GRF-04 cycle placement

| Option | Description | Selected |
|--------|-------------|----------|
| Server-side in `Engine.FileGraph` | Answer arrives with the data; testable in Go; renderer-independent | ✓ |
| Client-side in the renderer | Both candidates ship graph algorithms; minimal wire shape | |
| Server-side, separate opt-in call | Keeps FileGraph lean; second round trip | |

**User's choice:** Server-side in `Engine.FileGraph` (recommended).
**Notes:** Also the choice that protects GRF-05 — computing cycles client-side would tie a correctness property to the renderer, so a swap would have to re-implement it.

---

## Claude's Discretion

- The `Engine.FileGraph` wire shape (a 12th rpc; expect another one-way field-numbering gate, and check the name against `mutatingVerbs`' 19 substrings).
- What "laid out hierarchically with directory-structural grouping" means concretely.
- The GRF-05 seam interface.
- Which corpus counts as "largest", and whether the spike must index it first.
- Visual treatment of cycles beyond "distinguished without hunting".

## Deferred Ideas

- Deep-linking a graph node into `/browse` as a *secondary* affordance.
- A side panel listing a file's symbols — revisit only if in-place expansion proves unreadable at scale.
- Precomputed rollup projection — explicitly out of scope per ENG-03/GRF-05.
- Sigma.js + graphology with a hand-rolled grouping layer — revisit only if GRF-01's latency measurement fails.
- Overlapping / multi-parent groupings — Cytoscape's compound model is a strict tree.

## Final Gate

Offered four further gray areas (the FileGraph wire shape, what hierarchical grouping means concretely, the GRF-05 seam interface, and which corpus is largest). User selected **"I'm ready for context"** — all four recorded under Claude's Discretion.
