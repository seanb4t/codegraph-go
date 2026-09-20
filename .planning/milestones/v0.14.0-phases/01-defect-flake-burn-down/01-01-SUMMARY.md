---
phase: 01-defect-flake-burn-down
plan: 01
subsystem: testing
tags: [playwright, chromium, cytoscape, cytoscape-elk, svelte, taskfile, console-gate]

# Dependency graph
requires: []
provides:
  - "web/scripts/graph-console-check.mjs — permanent live-Chromium /graph console gate (self + guava corpora)"
  - "corpora/graph-console-check.json — committed pre-fix RED verdict (FIX-04 pageerror, FIX-05 valign warnings + guava overlap diagnosis)"
  - "check:graph-console Taskfile target"
  - "window.__codegraphFileGraphCy debug seam in GraphCanvas.svelte (new, for live edge/node diagnosis)"
affects: [01-04, 01-08, 01-09]

# Actuals (#2632)
actuals:
  tokens: 133740
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Live-Chromium console-capture harness: page.on('console') (warning/error) + page.on('pageerror'), both registered before navigation, feeding a committed schemaVersion:1 verdict JSON under corpora/ — the same shape as the existing graph-cluster-observations.json precedent (GRF-09)."
    - "--self-test as a positive control for a capture harness: plant one console.warn, one console.error and one uncaught pageerror, assert all three recaptured, before trusting any real run."
    - "Debug-only window.__codegraphFileGraphCy telemetry seam (mirrors the existing __codegraphFileGraphMetrics/__codegraphFileGraphGeometry convention) — the only way to page.evaluate() from a warning's edge id back to live source/target position and boundingBox data, since canvas rendering has no per-node DOM element."

key-files:
  created:
    - web/scripts/graph-console-check.mjs
    - corpora/graph-console-check.json
  modified:
    - Taskfile.yml
    - web/src/lib/components/graph/GraphCanvas.svelte
    - web/src/app.d.ts
    - web/build/** (rebuilt output, committed to match the source change per web:drift's contract)

key-decisions:
  - "The guava run's invalid-endpoints warnings trace to one directory pair (android/guava-tests/test/com/google/common/util/concurrent <-> android/guava/src/com/google/common/util/concurrent) via 2 edges (opposite direction). Both endpoints are non-parent (isParent:false, childCount:0 — i.e. still-collapsed leaf directory nodes, not a degenerate zero-size box), and their SETTLED bounding boxes do NOT overlap (~640px apart in y). This suggests the warning fires during an earlier render pass — before ELK's async layoutPositions() write-back lands (the same mechanism 01-RESEARCH.md Investigation 1 documents for FIX-04) — rather than reflecting the final settled layout. Recorded as an observation for plan 01-09 to investigate; not interpreted or fixed here."
  - "Added window.__codegraphFileGraphCy (Rule 2 deviation) since no existing seam exposes live cytoscape edge/node state for diagnosis — see Deviations section."

requirements-completed: [FIX-04, FIX-05]

coverage:
  - id: D1
    description: "graph-console-check.mjs exists with a --self-test positive control that plants and recaptures one console.warn, one console.error, and one uncaught pageerror"
    requirement: FIX-04
    verification:
      - kind: other
        ref: "node web/scripts/graph-console-check.mjs --self-test"
        status: pass
    human_judgment: false
  - id: D2
    description: "Two-corpus (self + guava) console gate records a committed RED verdict against the pre-fix build: pageerror (notify TypeError, FIX-04), 2 text-valign warnings (FIX-05 stylesheet half)"
    requirement: FIX-04
    verification:
      - kind: other
        ref: "task check:graph-console (self-test + node web/scripts/graph-console-check.mjs --out corpora/graph-console-check.json), verified via jq against the committed verdict"
        status: pass
    human_judgment: false
  - id: D3
    description: "check:graph-console Taskfile target exists (self-test then real check, guava-fetched precondition) and is not referenced from any .github/workflows/ file (D-09)"
    requirement: FIX-05
    verification:
      - kind: other
        ref: "rg -q 'check:graph-console' Taskfile.yml && ! rg -q 'graph-console-check' .github/workflows/"
        status: pass
    human_judgment: false
  - id: D4
    description: "Guava-scale invalid-endpoints overlap diagnosis: named edge ids traced to source/target ids, positions and bounding boxes via a new page.evaluate() call into the live cytoscape instance"
    requirement: FIX-05
    verification:
      - kind: other
        ref: "jq check on corpora/graph-console-check.json (guava run has >=1 invalidEndpointDiagnostics entry with sourceId/targetId); rg check for the pinned guava sha"
        status: pass
    human_judgment: true
    rationale: "The diagnosis is structurally proven (real edge/node ids, positions, bounding boxes recorded), but the INTERPRETATION — that the warning fires during an earlier render pass rather than reflecting the final settled overlap — is an observation, not a fix, and the eventual FIX-05 layout change (plan 01-09) should have a human confirm this reading before acting on it."

duration: ~45min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 1: Live-Chromium /graph Console Gate Summary

**New `web/scripts/graph-console-check.mjs` boots the real `codegraph` binary against this repo's own index and the pinned guava corpus, captures every `page.on('console')` warning/error and `page.on('pageerror')` before navigation, and commits a pre-fix RED verdict at `corpora/graph-console-check.json` naming the exact FIX-04 TypeError, the two FIX-05 stylesheet warnings, and a diagnosed guava-scale overlapping directory pair with coordinates.**

## Performance

- **Duration:** ~45 min
- **Tasks:** 2
- **Files modified:** 20 (4 hand-authored source/config files + 16 rebuilt `web/build/` output files)

## Accomplishments

- Built `web/scripts/graph-console-check.mjs`, the first script in `web/scripts/` to capture `console.warn`/`console.error` (every sibling script only captures `pageerror`), following the established `repoRoot()`/`pollUntil()`/diagnostic-JSON-on-every-exit-path convention.
- `--self-test` positive control: plants one `console.warn`, one `console.error`, and one uncaught `pageerror` on `about:blank`, and asserts the harness recaptures all three before any real corpus run is trusted.
- Extended the gate to the pinned `google/guava` corpus, reading its repo/sha pair from `corpora/selection.json`'s `lockedSet` (never a second hard-coded sha), and added the invalid-endpoints overlap diagnostic that traces a captured warning's edge id back to its live source/target positions and bounding boxes.
- Added `check:graph-console` to `Taskfile.yml` (self-test → real two-corpus check, guava-fetched precondition), confirmed NOT referenced from any `.github/workflows/` file (D-09).
- Ran the gate against the current, unmodified embedded build and committed the failing verdict at `corpora/graph-console-check.json` — the RED evidence plans 01-04/01-08/01-09 close against.

## Task Commits

1. **Task 1: End-to-end console capture on this repo's graph — one corpus, one path** - `c01eff7d` (feat)
2. **Task 2: Extend to the pinned guava corpus and diagnose the overlapping pair** - `2c69b3ba` (feat)

**Plan metadata:** _pending — added in the final docs commit_

## Files Created/Modified

- `web/scripts/graph-console-check.mjs` - the permanent `/graph` console gate (both corpora + overlap diagnostic)
- `corpora/graph-console-check.json` - committed pre-fix RED verdict
- `Taskfile.yml` - new `check:graph-console` target
- `web/src/lib/components/graph/GraphCanvas.svelte` - added `window.__codegraphFileGraphCy` debug-only telemetry seam (deviation, see below)
- `web/src/app.d.ts` - typed the new `__codegraphFileGraphCy` global
- `web/build/**` - rebuilt SPA output, committed to keep `web:drift`'s source/output digests consistent with the `GraphCanvas.svelte` change

## Decisions Made

- Guava's invalid-endpoints warnings trace to one directory pair via 2 opposite-direction edges; both endpoints are real (non-degenerate) bounding boxes that do NOT overlap post-settle — see `key-decisions` in frontmatter for the full observation and its implication for plan 01-09.
- `self`'s `repo`/`sha` identity in the verdict is best-effort (git remote/HEAD, falling back to a directory-name/`"unknown"` default) since — unlike guava's pin — it is informational only, not something a verify gate checks byte-for-byte.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Added `window.__codegraphFileGraphCy` debug seam to GraphCanvas.svelte**

- **Found during:** Task 2 (guava overlap diagnostic)
- **Issue:** Task 2's action text requires `page.evaluate()` to read a live edge's `source()/target()` positions and bounding boxes from "the live instance" — but no existing seam exposes the cytoscape core instance (or any edge-level data) to the page's `window`. `window.__codegraphFileGraphMetrics`/`Geometry` publish per-node summary data only, and canvas rendering leaves no per-node DOM element to query another way. Without this, the diagnostic capability Task 2 is explicitly chartered to build could not exist at all.
- **Fix:** Added `window.__codegraphFileGraphCy = cy;` immediately after `renderer.start()` in `GraphCanvas.svelte`'s mount effect, mirroring the existing `__codegraphFileGraphMetrics`/`__codegraphFileGraphGeometry` telemetry convention already in the same file (additive, read-only, "a real renderer always provides"). Cleared on teardown (`window.__codegraphFileGraphCy = undefined`) before `cy.destroy()`, matching cytoscape's own "extra safe" reference-clearing discipline documented in 01-RESEARCH.md Investigation 1. Typed as `unknown` in `web/src/app.d.ts`'s existing `declare global { interface Window }` block, cast at both the assignment and read sites — the same `as any`/cast discipline this file already applies to every cytoscape-shaped value it cannot express without importing the library.
- **Files modified:** `web/src/lib/components/graph/GraphCanvas.svelte`, `web/src/app.d.ts`, and `web/build/**` (the SPA had to be rebuilt via `task web:build` and the Go binary rebuilt via `task build:release` to embed the change; `web/build/**` was committed to keep `task web:drift`'s source/output digest match intact — an uncommitted rebuild would have made the committed build stale relative to source).
- **Verification:** `pnpm run check` (svelte-check): 0 errors. `pnpm test` (vitest): 584/584 passing, no regressions. `go test ./internal/uiserver/... -run TestSPA`: all 14 subtests pass, including the byte-identical `default-src 'self'` CSP assertion (D-05's negative control) — confirming this change touches no CSP directive. `task web:drift`: PASS (source and output digests both match the marker after the rebuild).
- **Threat assessment:** Debug-only, read-only, additive telemetry. Assigning to `window` emits no console output (does not itself trip D-08's bar) and does not widen any CSP directive. The exposed data (node/edge ids, positions, bounding boxes) is already visible or inferable from the rendered graph itself — no credential, token, or new information disclosure. Scoped narrowly to graph structure already public to anyone viewing the page.
- **Committed in:** `2c69b3ba` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical functionality)
**Impact on plan:** Necessary for Task 2's explicit deliverable (the overlap diagnosis) to exist at all. Widens the plan's declared `files_modified` beyond `graph-console-check.mjs`/`Taskfile.yml`/`corpora/graph-console-check.json` to include `GraphCanvas.svelte`, `app.d.ts`, and the rebuilt `web/build/` tree — flagged here explicitly since none of those three were in the plan's frontmatter list. No scope creep beyond what Task 2 itself required; no user-facing behavior change.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new-debug-surface | web/src/lib/components/graph/GraphCanvas.svelte | New `window.__codegraphFileGraphCy` global exposes the live cytoscape instance for diagnostic scripts. Read-only, additive, no CSP/console impact (see Deviations §1 threat assessment above); not in this plan's original `<threat_model>` STRIDE register since the seam did not exist when that register was authored. |

## Known Stubs

None — both tasks' deliverables are fully wired (self-test passes, both corpora run, guava diagnostic populated).

## Issues Encountered

- **Go toolchain mismatch (environment, not a defect):** the locally-installed `go1.27.1` silently shadows `go.mod`'s declared `go 1.26.6` (exactly the environment note 01-RESEARCH.md flagged). Every `task build:release` / `task web:build` invocation in this session used `GOTOOLCHAIN=go1.26.6` explicitly. No code change; documented here so the next executor in this phase does the same.
- **Duplicate console entries:** both the self corpus's `text-valign` warnings and the guava corpus's `invalid-endpoints` warnings were observed to fire more than once (e.g. 2x, 4x raw occurrences before dedup) across a single page load — consistent with a layout/render pass running twice (the same double-render family FIX-04's own root cause belongs to). `extractInvalidEndpointEdgeIds` deduplicates by edge id before diagnosing, so this did not affect the recorded evidence's correctness, but it is additional circumstantial support for the double-mount/double-effect hypothesis 01-RESEARCH.md Investigation 1 already documents.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `web/scripts/graph-console-check.mjs` and `task check:graph-console` are ready for plans 01-04 (FIX-04) and 01-08 (FIX-05 stylesheet half) to re-run as their own `<verify>` gate once each fix lands — the committed pre-fix verdict at `corpora/graph-console-check.json` is the baseline they close against.
- Plan 01-09 (FIX-05 layout fix) has the named overlapping pair, edge ids, positions and bounding boxes it needs to start from — but should first investigate why the SETTLED bounding boxes recorded here do not actually overlap (see Decisions above) before choosing a spacing/sizing fix, since the diagnosis may need re-capturing at the moment the warning actually fires (mid-layout) rather than after settle, if the current after-settle read is misleading.
- No blockers for the next wave.

## Self-Check: PASSED

- FOUND: `web/scripts/graph-console-check.mjs`
- FOUND: `corpora/graph-console-check.json`
- FOUND: commit `c01eff7d` (Task 1)
- FOUND: commit `2c69b3ba` (Task 2)
- FOUND: `check:graph-console` target in `Taskfile.yml`
- Re-ran plan-level `<verification>`: `node web/scripts/graph-console-check.mjs --self-test` exits 0 with warn=1/error=1/pageerror=1 recaptured; `task check:graph-console` exits non-zero with `success: false` in the committed verdict; guava run's `invalidEndpointDiagnostics` has 2 entries with sourceId/targetId/positions/boundingBoxes; `rg -q 'check:graph-console' Taskfile.yml && ! rg -q 'graph-console-check' .github/workflows/` exits 0.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
