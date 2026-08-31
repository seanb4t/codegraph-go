---
phase: 05-file-package-graph-view
verified: 2026-08-31T17:44:47Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 5: File/Package Graph View Verification Report

**Phase Goal:** A developer can see the whole repository as one readable picture at
file/package granularity, drill into any file for its symbols, and spot dependency cycles —
with the renderer chosen by recorded measurement rather than by assumption.
**Verified:** 2026-08-31T17:44:47Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped 1:1 to ROADMAP Success Criteria)

| # | Truth (GRF-01..05, ENG-03) | Status | Evidence |
|---|---|---|---|
| 1 | GRF-01: a committed measurement, against a pass condition locked before it ran, names the selected renderer | ✓ VERIFIED | See "Criterion 1" section below — verified mechanically (git history + byte-diff), not from prose. |
| 2 | GRF-02: hierarchical, directory-grouped, per-kind-aggregated graph — never force-directed | ✓ VERIFIED | `LAYOUT_OPTIONS` in `GraphCanvas.svelte:111-135` uses `cytoscape-elk`'s `algorithm: 'layered'` (Sugiyama), never `fcose`/`dagre`-force. No `cytoscape-fcose` or force-directed layout dependency exists in `web/package.json`. `file-graph-transform.ts` derives directory-compound `parent` from `FilePath` splitting; `FileGraphEdge.KindCounts` is a real sparse per-kind map (`internal/query/traverse.go:222-249`). Confirmed live against this repo's own 597-file index via headless Playwright: 106 nodes / 152 edges rendered as directory-grouped compound boxes laid out left-to-right hierarchically (screenshot captured; not a force-directed hairball). |
| 3 | GRF-03: clicking a file reveals its symbols in place | ✓ VERIFIED | `internal/query/filesymbols.go` (`Engine.FileSymbols`) + `UIService.FileSymbols` (`internal/uiserver/handlers.go:1096`) + `+page.svelte`'s `toggleFile`/`symbolElementsForFile` incremental-add seam. Directly driven live: clicking a file node's rendered geometry increased the on-canvas node count from 107 to 114 (7 new symbol child nodes appeared, parented to the file) — see "Manual-Only items, machine-verified" below. |
| 4 | GRF-04: cycles visually distinguished without hunting | ✓ VERIFIED | Server-side Tarjan SCC (`internal/query/filegraph_cycles.go`, iterative, non-recursive — `TestFileGraphCyclesDeepChainDoesNotRecurse` passes) populates `CycleID`/`InCycle` on the wire; `graph-style.ts:147-163` marks cycle nodes/edges on THREE non-hunt-required channels (colour + dashed border/line-style + distinct arrow shape). Confirmed live: this repository's own index surfaces 10 real dependency cycles, rendered with a visible red-outlined distinction in the captured screenshot, plus an explicit "This repository has 10 dependency cycles" / "Focus a cycle (10 total)" affordance in the rendered page text. |
| 5 | ENG-03/GRF-05: fresh-per-request rollup (no precomputed projection/new record kind/re-indexing) behind a narrow renderer seam | ✓ VERIFIED | `Engine.FileGraph()` (`traverse.go:172-331`) runs two full scans (`IterateNodes`/`IterateEdges("")`) per call — no `sync.Once`, no package-level cache (grep confirms zero cache/memoization sites in `traverse.go`, `filesymbols.go`, `filegraph_cycles.go`). `GraphCanvas.svelte` is the sole importer of `cytoscape`/`cytoscape-elk` in `web/src` — grep for `cytoscape` outside that file only matches doc-comments in `file-graph-transform.ts`/`graph-style.ts`, never an `import`. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified)

### Criterion 1 — GRF-01's Integrity Property (verified mechanically)

This is the phase's reason for existing, so it was checked with commands, not by reading prose:

1. `git log --oneline -- corpora/graph-render-threshold.json` → **exactly 1 commit** (`2fb27746`, "lock GRF-01's pass condition before any measurement exists").
2. `git merge-base --is-ancestor 2fb27746 <commit-writing-graph-render-observations.json>` → **true** (`1cb8c71e`).
   `git merge-base --is-ancestor 2fb27746 <commit-writing-graph-render-observations-collapsed.json>` → **true** (`92329657`).
3. `git show 1cb8c71e:corpora/graph-render-threshold.json | diff - corpora/graph-render-threshold.json` → **no diff**.
   `git show 92329657:corpora/graph-render-threshold.json | diff - corpora/graph-render-threshold.json` → **no diff**.
   The threshold's four binding bars (`timeToInteractiveMs<=5000`, `panZoomFrameTimeMs<=33.3`, `panZoomFrameTimeP95Ms<=100`, `fileGraphResponseBytes<=16777216`) are byte-identical across both measurement runs.

**The honest history is real, not narrated:**
- `corpora/graph-render-observations.json` (guava, expanded file-level view, 3,233 nodes / 21,554 edges): `verdict: "FAIL"`. All three latency metrics show `status: "measurement-failed"`, `failureReason: "seamReady exceeded its 60000ms deadline"`. Only `fileGraphResponseBytes` (3,713,528 bytes, measured via a real server+client round trip cited from `05-02-SUMMARY.md`) passed.
- `corpora/graph-render-observations-collapsed.json` (same guava corpus, collapsed default): `verdict: "PASS"`. `timeToInteractiveMs=1179.2` (bar 5000), `panZoomFrameTimeMs=8.3` (bar 33.3), `panZoomFrameTimeP95Ms=9` (bar 100), `fileGraphResponseBytes=3713528` (bar 16777216) — all four PASS against the unchanged locked bars. `nodeCount=134`/`edgeCount=819` match the threshold's pre-recorded `recordedNonBinding.collapsedView.approxNodes/approxEdges` exactly.
- The remedy applied (`halt-collapse-default`, selected from the two options written into the locked artifact's own `onFailure` field before any measurement existed) is recorded verbatim in `05-04-SUMMARY.md`'s key-decisions and cross-referenced in `05-08-SUMMARY.md`.
- **The selected renderer is named in the committed artifact, not inferred from preference**: the threshold's own `onFailure` remedy (a) text and `05-04-SUMMARY.md`'s maintainer-decision record both state "Keep Cytoscape + `cytoscape-elk`" — confirmed in the shipped bundle: `web/package.json` carries `cytoscape@3.34.2` + `cytoscape-elk@2.3.0` (exact-pinned) as the only rendering-library dependencies; no Sigma.js/graphology dependency exists anywhere in the tree.

Criterion 1 is met in full: the pass condition was locked before any measurement, the FAIL was not quietly discarded or overwritten (it survives byte-identically at its own commit and in the working tree), the threshold was never widened/lowered in response to the FAIL, and the recorded PASS — on the same unchanged bars — is what actually selects Cytoscape.js + cytoscape-elk as the shipped renderer.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `corpora/graph-render-threshold.json` | GRF-01's locked pass condition | ✓ VERIFIED | 1 commit, ancestor of both measurement commits, byte-identical across both |
| `corpora/graph-render-observations.json` | GRF-01's FAIL measurement | ✓ VERIFIED | `verdict: "FAIL"`, seamReady deadline exceeded at guava expanded scale |
| `corpora/graph-render-observations-collapsed.json` | GRF-01's PASS re-measurement | ✓ VERIFIED | `verdict: "PASS"`, all 4 bars pass on unchanged threshold |
| `internal/query/traverse.go` — `Engine.FileGraph()` | ENG-03/GRF-02 fresh two-scan rollup | ✓ VERIFIED | `traverse.go:172-331`; no cache; excludes `contains`, self-edges, `package` pseudo-nodes (D-03/D-08); sparse per-kind `KindCounts` |
| `internal/query/filegraph_cycles.go` | GRF-04 server-side SCC | ✓ VERIFIED | Iterative Tarjan, `TestFileGraphCyclesDeepChainDoesNotRecurse` proves non-recursive; wired into `FileGraph()`'s `CycleID`/`InCycle`/`CycleCount` |
| `internal/query/filesymbols.go` | GRF-03 `Engine.FileSymbols()` | ✓ VERIFIED | Ordered by line then name, capped at `MaxFileSymbols=2000`, path validated via `ValidateRepoRelativePath` precondition |
| `internal/uiserver/handlers.go` — `FileGraph`/`FileSymbols` rpcs | wire projection | ✓ VERIFIED | `fileGraphToProto`/`fileSymbolsToProto` mappers, both wired through `withEngine(...)`; 12th/13th rpcs, frozen at maintainer-approved blocking-human checkpoints (05-02, 05-06 SUMMARYs) |
| `web/src/lib/components/graph/GraphCanvas.svelte` | GRF-05 renderer seam | ✓ VERIFIED | Sole `cytoscape`/`cytoscape-elk` importer in `web/src`; every prop/event crossing the boundary is plain data (element-data objects, numbers/booleans/strings) |
| `web/src/lib/components/graph/file-graph-transform.ts` | GRF-02 directory-grouping + aggregation transform | ✓ VERIFIED | Pure, DOM-free, cytoscape-free; derives `parent` from `FilePath` splitting; copies (never computes) cycle fields |
| `web/src/lib/components/graph/graph-style.ts` | GRF-04 cycle styling | ✓ VERIFIED | `node.graph-cycle`/`edge.graph-cycle` selectors: dashed border/line-style + distinct arrow shape + colour |
| `web/src/routes/graph/+page.svelte` | filled `/graph` route | ✓ VERIFIED | Placeholder "Phase 5:" subtitle replaced; live-rendered heading is "Graph" with no planning-vocabulary leak (confirmed live, not just by static grep) |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `/graph` route | `UIService.FileGraph` rpc | `uiClient.fileGraph()` on mount | ✓ WIRED | Confirmed live: `window.__codegraphFileGraphMetrics` published (`timeToInteractiveMs=364.2ms`, `nodeCount=106`, `edgeCount=152` against this repo's own 597-file index) |
| `UIService.FileGraph` handler | `Engine.FileGraph()` | `withEngine(...)` closure | ✓ WIRED | `internal/uiserver/handlers.go:1049-1073` |
| Directory node click | `toggleDirectory` → `rollupToElements(response, expandedDirs)` | GraphCanvas `onNodeSelected` → route handler | ✓ WIRED | Confirmed live: clicking an expandable directory's geometry point increased rendered node count 106→107 |
| File node click | `UIService.FileSymbols` rpc → in-place symbol expansion | `toggleFile` → `uiClient.fileSymbols()` → `symbolElementsForFile` → `GraphCanvas.add()` | ✓ WIRED | Confirmed live: clicking a file node increased rendered node count 107→114 (7 symbol children appeared) |
| `FileGraph` cycle fields | rendered cycle styling | `file-graph-transform.ts`'s `classes` copy (never derivation) → `graph-style.ts` selectors | ✓ WIRED | Confirmed live: page renders "This repository has 10 dependency cycles" and visibly distinguishes cycle-member nodes/edges in the captured screenshot |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---|---|---|---|---|
| `GraphCanvas.svelte` elements | `response` (FileGraphResponse) | Real `uiClient.fileGraph()` RPC → `Engine.FileGraph()` → two live `graphstore.Reader` scans | Yes | ✓ FLOWING |
| Node/edge counts on `/graph` | `graphState.response.nodes/edges` | Same live RPC round trip | Yes | ✓ FLOWING |
| Cycle count text | `response.cycleCount` | `stronglyConnectedCycles()` over the same live aggregated adjacency | Yes | ✓ FLOWING |
| Symbol children on file expand | `FileSymbolsResponse.symbols` | Real `uiClient.fileSymbols({path})` RPC → `Engine.FileSymbols()` → live node scan | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Go unit suite for FileGraph/FileSymbols/cycles | `go test ./internal/query/... -run 'FileGraph\|FileSymbols'` | 25/25 tests pass, including `TestFileGraphAgainstThisRepositoryIndex` (597 nodes, 1077 edges, ExcludedPackageNodes=43 against this repo's live index) | ✓ PASS |
| Frontend unit/component suite | `npx vitest run` (web/) | 417/417 tests pass (37 files) | ✓ PASS |
| Type/svelte-check | `pnpm check` (web/) | 0 errors / 0 warnings across 1152 files | ✓ PASS |
| Build-output drift gate | `task web:drift` | PASS — source half (108 files) and output half (32 files) both match their committed digests | ✓ PASS |
| Live end-to-end: `/graph` on this repo's own index | headless Playwright against `./codegraph ui --no-open`, real built binary | Metrics seam published (`timeToInteractiveMs=364.2ms`, `nodeCount=106`), zero page errors, no planning-vocabulary leak, directory expand 106→107 nodes, file→symbols expand 107→114 nodes, "10 dependency cycles" rendered and visually distinguished (screenshot captured) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| ENG-03 | 05-01, 05-02 | `Engine.FileGraph()` fresh-per-call aggregated rollup with per-kind counts | ✓ SATISFIED | `traverse.go:172-331`, no cache, two-scan discipline |
| GRF-01 | 05-01, 05-03, 05-04, 05-08 | Locked-before-measurement renderer selection | ✓ SATISFIED | Verified mechanically above — FAIL then PASS on unchanged bars |
| GRF-02 | 05-03, 05-07, 05-08 | Directory-grouped hierarchical graph, never force-directed | ✓ SATISFIED | `elk`/`layered` algorithm; no force-directed dependency; confirmed live |
| GRF-03 | 05-06, 05-07 | Click a file → see its symbols | ✓ SATISFIED | `FileSymbols` rpc + in-place expansion; confirmed live (107→114 nodes) |
| GRF-04 | 05-01, 05-05 | Cycles visually distinguished | ✓ SATISFIED | Server-side SCC; 3-channel non-colour-dependent styling; confirmed live (10 real cycles rendered distinctly) |
| GRF-05 | 05-03 | Renderer sits behind a narrow swap seam | ✓ SATISFIED | `GraphCanvas.svelte` is the sole cytoscape importer; verified by grep across `web/src` |

No orphaned requirements found — `.planning/REQUIREMENTS.md` lines 148/167-171 map ENG-03 and GRF-01..05 exclusively to Phase 5, and every one appears in a plan's `requirements-completed` list (05-01 through 05-08).

### Anti-Patterns Found

None blocking. Reviewed `05-REVIEW.md` (deep review, 2 Critical / 1 Warning found), `05-REVIEW-FIX.md` (fixes applied), and `05-REVIEW-2.md` (re-review of the fix commits, 0 Critical / 3 Warning found and subsequently resolved per its own "Resolution" section, commits `209a5fc0`/`420b18cb`/`b7fc3897`/`c48a56c5`). Both Critical findings (CR-01 duplicate in-flight `fileSymbols` request; CR-02 symbol-wipe on unrelated directory collapse) were verified present in the shipped code as fixes, not merely claimed:
- `pendingFileSymbols` in-flight guard confirmed present at `+page.svelte:81,333,348,353,376`.
- `dirOf`-based `expandedFiles` resync on directory collapse confirmed present at `+page.svelte:261` / exported from `file-graph-transform.ts:134`.

Review-artifact-id leaks into shipped source comments (a defect class this codebase has hit repeatedly) were rechecked live: `rg -n "phase 5|phase-5|GRF-0|ENG-03|05-0|CR-[0-9]|WR-[0-9]|found in review" web/src/routes/graph/+page.svelte web/src/lib/components/graph/GraphCanvas.svelte` → zero matches.

Security: `05-SECURITY.md` independently re-parsed 51 distinct threat IDs (T-05-01..50 + T-05-SC) against the working tree at HEAD (`c117a898`), `threats_open: 0`, ASVS L1, all 27 high-severity threats mitigated (zero accepted at high). `1b578078` (current HEAD at verification time) is the security-verification commit itself.

Known, deliberately-out-of-scope open items (WINDOWS.md), recorded as such and not treated as defects:
- **#26** (open): non-fatal `cytoscape-elk` adapter `TypeError` at guava scale; layout completes, gestures work. Confirmed as documented — this repo's own live check above produced zero page errors, consistent with the WINDOWS entry's note that this is guava-scale-specific.
- **#28** (open): guava-scale cytoscape layout console.warn for overlapping edge endpoints; pre-existing, non-blocking.
- **#29** (open): `web:drift`'s output half enumerates the filesystem while the source half uses `git ls-files`, a real gate blind spot found and documented with a suggested fix, out of phase scope.
- **#25, #27** (both `status: fixed`): #27's fix verified via `corpora/graph-collapse-affordance-check.json` (`success: true`, directory `106→111→106` and file `111→112→111` both restored exactly, 0 page errors) — checked directly, not taken on the SUMMARY's word.

### Human Verification Required

None. All three items in `05-VALIDATION.md`'s "Manual-Only Verifications" table were attempted and settled by direct machine verification during this pass, per the escalation-gate discipline that a `why_human` claim is testable, not a category:

1. **"Click-to-expand actually expands a file into its symbols in place" (GRF-03)** — the table's stated reason (`jsdom cannot exercise Cytoscape's real tap gesture`) is accurate for the unit-test layer, but the table's own prescribed remedy (build the binary, run `codegraph ui --no-open`, drive `/graph` with a real browser automation tool) is exactly what this verification pass executed: `task build:release`, then a headless Playwright session against the real running binary, clicking the rendered-geometry seam's (`window.__codegraphFileGraphGeometry`) coordinates for a directory node then a file node. Result: 106→107→114 rendered nodes, the last increment being 7 new symbol child nodes. Settled, not deferred.
2. **"The graph reads as hierarchical rather than as a hairball; pan/zoom stays smooth" (GRF-01, GRF-02)** — the binding, decisive measurement (guava scale) is GRF-01's own committed, protocol-locked artifact (`corpora/graph-render-observations-collapsed.json`), already verified mechanically above; re-running it here would not add evidence beyond what the locked protocol already captured with tighter methodology (three cold reloads, a pinned pan/zoom gesture script) than an ad hoc spot check could. This repository's own smaller-scale index was additionally screenshotted live in this pass and visually reads as directory-grouped and hierarchical, not a force-directed hairball.
3. **"`/graph` no longer renders planning vocabulary" (GRF-05 shipped copy)** — settled by reading the live-rendered `document.body.innerText` in the same Playwright session: heading is "Graph", and a `phase 5|phase-5|GRF-0|ENG-03` regex over the rendered body text found no leak.

### Gaps Summary

None. All five ROADMAP success criteria (GRF-01..05, ENG-03) are met, verified against the working tree and a live-driven build of the actual binary — not against SUMMARY.md prose. GRF-01's central integrity property (threshold locked before any measurement, unchanged across a genuine FAIL and a genuine PASS) was checked with git history and byte-level diffs, not narrative. Two Code-review-found Criticals were confirmed fixed in the shipped source. No open WINDOWS.md item for this phase rises to blocking severity; all are either closed with evidence or explicitly recorded as deliberately deferred/non-blocking.

---

_Verified: 2026-08-31T17:44:47Z_
_Verifier: Claude (gsd-verifier)_
