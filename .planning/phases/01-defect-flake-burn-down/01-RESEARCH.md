# Phase 1: Defect & Flake Burn-down - Research

**Researched:** 2026-09-14
**Domain:** Go daemon concurrency/test-seam design, cytoscape.js/cytoscape-elk internals, Pebble store lock semantics, GitHub Actions workflow security, Go benchmark-gate hygiene, SvelteKit static-asset/CSP conventions
**Confidence:** HIGH — every claim below is either read from source in this session (own code, `node_modules` vendor source, or the two workflow files) or a live command run against this exact checkout. No web search was used; this phase is entirely repo-internal investigation, so the source hierarchy is `[VERIFIED: <path:lines>]` for code facts and `[VERIFIED: live run]` for reproductions.

## Summary

This phase closes nine ledger items with a single hard constraint: the fix has to remove the *cause*, not narrow the symptom (widen a timeout, patch a dependency, widen a CSP). This research session did not just read the ledger — it re-ran the two load-bearing reproductions the phase asks for and read cytoscape's own vendored source to find the exact missing guard, so the planner is not starting from a hypothesis but from a confirmed mechanism for the two hardest items (FIX-04, FIX-07/08).

**Confirmed live, this session:**
- `TestRunWatchdogCancelsRunOnSimulatedReparent` **fails today** under `go test -race -count=1 ./...` (250.35s wall time before its own bumped 10s×25-scale budget expires), with **zero data races reported** across the whole run. GH #17's functional flake is real and current; GH #13's `-race` symptom is not reproducing (see Investigation 3 below for why both are true at once, and why FIX-08 is not therefore closed).
- Cytoscape 3.34.2's `endBatch()` calls `renderer.notify(...)` with **no guard at all** for a null/destroyed renderer — while the sibling standalone `notify()` method four lines above it in the same file has exactly that guard. This is a real, nameable asymmetry in the vendored dependency, not a guess (`web/node_modules/cytoscape/dist/cytoscape.esm.mjs:15735-15810`).

**Primary recommendation:** For the daemon pair (FIX-07/08), finish the per-instance seam refactor D-13/D-14 already specify — the package-level `getppid`/ticker discipline that already exists (`joinDaemonRun`/`testBudget`, added v0.3.0 Phase 4) reduced the race's *likelihood* but did not remove the *pattern*, and the functional flake this session reproduced proves the wall-clock-ticker cause is still live. For the graph console (FIX-04), the fix is a **timing discipline in `GraphCanvas.svelte`'s own teardown**, not a dependency patch and not a `layoutGeneration`-token check alone — the crash fires *before* the token-checked `layoutstop` callback ever runs.

## User Constraints

<user_constraints>
### Locked Decisions

- **D-01/D-02/D-03:** The codegraph mark (octagonal "CG" ligature) is locked; tile variant is the favicon, bare variant is unplaced this phase. Raw SVGs need cleanup (strip C2PA metadata, `preserveAspectRatio`, square viewBox) before shipping.
- **D-04:** Ship SVG + 32×32 PNG + 180×180 apple-touch-icon as **static files under `web/static/`**, all `'self'`-served. `+layout.svelte` links all three; `$lib/assets/favicon.svg` (Svelte logo, Vite-inlined as `data:`) is deleted.
- **D-05:** `internal/uiserver/spa.go`'s `spaCSPBaseDirectives` and `spa_test.go`'s `default-src` assertion are untouched. No `img-src`, no `data:`.
- **D-06:** FIX-04 is fixed in **our own code**, never a dependency patch (no `pnpm patch`, no `patches/`). If genuinely uninterceptable, isolate with the cause recorded.
- **D-07:** FIX-05 is root-caused (which collapsed pair overlaps under ELK) then **always fixed** (layout-option, sizing-model, or algorithm change acceptable). Any layout change verified live in Chromium on both this repo's graph and the pinned guava corpus.
- **D-08:** The two `text-valign: right` warnings are in scope (should be `text-halign`). Bar is a clean console: zero uncaught errors, zero `console.warn`/`console.error` from our code, on both corpora.
- **D-09:** New `web/scripts/graph-console-check.mjs` (styled like the five existing live checks), verdict committed under `corpora/`, has a Taskfile target, **not** wired into CI.
- **D-10:** `codegraph index --force` against a held store **refuses**: surface `errors.Is(err, graphstore.ErrStoreLocked)`, exit non-zero before `os.RemoveAll`, name the holder, point at `codegraph daemon stop` / `codegraph unlock`. No new flag.
- **D-11:** Corrupt/unreadable store (Open/GetMeta fails with something other than `ErrStoreLocked`/`ErrNotFound`) **warns and rebuilds** (floor 0). `ErrNotFound` stays silent at floor 0.
- **D-12:** Regression test is the hold-the-lock-across-`index --force` shape (WINDOWS #36), modeled on `open_lock_test.go`, watched fail against the pre-fix build (RED transcript required).
- **D-13:** `getppid` seam becomes **per-instance**: `startWatchdog` takes `func() int` as a parameter; `Daemon` carries it as an unexported field set by a test-only `Option`. No package-level `var getppid` remains.
- **D-14:** FIX-07 fixed by injecting the ticker/clock alongside the seam; `TestRunWatchdogCancelsRunOnSimulatedReparent` asserts on a signal, not a deadline. **No timeout constant changes anywhere.**
- **D-15:** GH #13's leaked-goroutine half (from `TestConvergenceTwoSessions`) is confirmed or refuted in this phase, not assumed. If real, closed; `go test -race ./internal/daemon/...` is the gate for both halves.
- **D-16:** `CheckRegression` gains a strict-equality `Repo` guard on the GOOS/Runner/ScratchFS template. No normalisation — `Repo` is already the synthesized `synthetic-seed{N}-count{M}` id at the regression-mode write site.
- **D-17:** FIX-10 uses a per-run random delimiter (`DELIM="PRFILES_$(openssl rand -hex 16)"`) in **both** workflows, exercised by a repo-local shell test that feeds a `PRFILES_EOF`-named path through the same block.
- **D-18:** GH #20 follow-up 1 closed won't-do (recorded). Follow-up 2 gets one manually-dispatched discriminator run (old-baseline commit, today's runner, same seed/count), recorded in `tools/bench/BASELINE.md`. Both close GH #20.

### Claude's Discretion

- Exact wording of the FIX-06 refuse/warn messages.
- Whether the 32px PNG / apple-touch-icon are committed files or build-step-rendered.
- Which of "ignore stale run" vs "cancel on destroy" closes FIX-04.
- How the ELK spacing/sizing change for FIX-05 is expressed.
- Shape of the injected ticker (interval override vs explicit tick channel).
- Name/location of the FIX-10 shell test harness.

### Deferred Ideas (OUT OF SCOPE)

- Perf-gate baseline staleness check (GH #20's "the gate needs a staleness check, not just a one-time refresh") — backlog candidate, not this phase.
- In-app header placement for the bare mark — no UI consumes it this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-02 | Favicon serves under unchanged CSP | SvelteKit static-asset section below; confirmed current `+layout.svelte:7,91` import/link shape and absence of image tooling in `web/package.json` |
| FIX-04 | `/graph` zero uncaught errors — cytoscape-elk `notify` null TypeError | Investigation 1 below: exact missing-guard line in vendored cytoscape 3.34.2, full call chain traced from `cytoscape-elk`'s async `.then()` through to the crash site |
| FIX-05 | Guava-scale "invalid endpoints" warnings root-caused | Investigation 2 below: exact warning-emission code, mechanism, and the gate/tooling path to name the pair |
| FIX-06 | `priorCoverageGeneration` distinguishes lock/corrupt from not-found | Investigation 4 below: current code read, sentinel definitions, message-parity precedent, test-shape precedent all confirmed |
| FIX-07 | Watchdog test passes deterministically under load | Investigation 3 below: live-reproduced failure this session; exact root cause (real 1s wall-clock ticker racing CPU contention) |
| FIX-08 | `getppid` seam race-free | Investigation 3 below: confirmed the package-level global still exists in current code; confirmed prior partial fix (v0.3.0 Phase 4) reduces likelihood but does not remove the pattern |
| FIX-09 | `CheckRegression` compares `Metrics.Repo` | Investigation 5 below: all three `Repo` write sites read; confirmed the only site feeding `CheckRegression` already writes a stable synthesized id |
| FIX-10 | Fixed heredoc delimiter over fork-controlled paths | Both workflow files read verbatim; exact block spans and existing script/test conventions confirmed |
| FIX-11 | GH #20 follow-ups each get a recorded decision | GH #20 read in full including its follow-up comment; decision mechanics confirmed against `tools/bench/BASELINE.md`'s existing rebless discipline |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Tech stack: Go (latest stable), single static binary; pnpm (not npm) for the SPA; the SPA build stays outside the reproducible release path.
- `Taskfile.yml` is the single definition of every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it) — any new check must land as a Task target, not inline-only workflow YAML.
- A gate is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation (rule `84d1gfpywd`).
- Never invent structure in a tool-owned generated file (`.planning/ROADMAP.md`, `.planning/STATE.md`).
- No `[ci skip]`/`[skip ci]` ever. Agent commits may use `-c commit.gpgsign=false`.
- The agent/MCP output path and `--json` stay byte-identical — not directly relevant to this phase's file set (`web/`, `internal/cli/index.go`, `internal/daemon`, `internal/bench`, two workflows) but worth flagging: none of this phase's changes should touch `internal/query`/`internal/mcp`.
- Prefer `rg`/`sg` over grep for any census-style search this phase's tasks perform (e.g. verifying no other `pull_request_target` workflow shares the heredoc shape).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Favicon delivery + CSP compliance (FIX-02) | Browser/Client (static asset) | Frontend Server (SvelteKit static routing, `web/static/`) | A static asset served under `'self'` needs no server logic change; SvelteKit's adapter-static build is the only "server" tier involved |
| Graph render console cleanliness (FIX-04/05) | Browser/Client (cytoscape/cytoscape-elk lifecycle inside `GraphCanvas.svelte`) | — | Entirely a client-side rendering-library lifecycle bug; no backend/API involvement |
| Store-lock refusal on `index --force` (FIX-06) | CLI / Backend (`internal/cli/index.go`, `internal/graphstore`) | — | Filesystem-lock-aware logic belongs in the CLI command and the storage package that owns the sentinel errors |
| Daemon watchdog seam & flake (FIX-07/08) | Backend (`internal/daemon`) | — | Pure Go concurrency/test-infrastructure defect, no client or API surface |
| Bench regression gate (FIX-09) | CI / Build tooling (`internal/bench`, `tools/bench/runner`) | — | A comparison-gate library and its CLI driver; not part of the shipped product |
| CI workflow hardening (FIX-10) | CI / Build tooling (`.github/workflows/*.yml`) | — | GitHub Actions YAML plus the scripts they shell out to |
| Perf-gate follow-up decisions (FIX-11) | CI / Build tooling (`tools/bench/BASELINE.md`, GH issue record) | — | A documentation/decision-recording exercise, not code |

## Standard Stack

No new external packages are introduced by this phase (see Package Legitimacy Audit below). This phase is entirely fixes within the existing stack:

| Area | Existing Library/Tool | Version (confirmed) | Role in this phase |
|------|------------------------|----------------------|---------------------|
| Graph rendering | `cytoscape` | 3.34.2 (`web/node_modules/cytoscape/package.json:3`) [VERIFIED: web/node_modules/cytoscape/package.json:3] | Root-cause site for FIX-04/05 |
| ELK layout adapter | `cytoscape-elk` | 2.3.0 (`web/node_modules/cytoscape-elk/package.json:3`) [VERIFIED: web/node_modules/cytoscape-elk/package.json:3] | Async promise chain that triggers FIX-04's crash |
| ELK engine | `elkjs` | 0.9.3 (per STATE.md Phase 5 decision — resolved transitively, not the 0.12.0 research once measured) [CITED: .planning/STATE.md Phase 5 entry] | Layout algorithm whose spacing options are the FIX-05 lever |
| Live-Chromium harness | `@playwright/test` (already a devDependency, used by the five existing `web/scripts/*-check.mjs`) | not re-verified this session (no version change needed) [ASSUMED — unchanged from prior phases] | Model for the new `graph-console-check.mjs` |
| Delimiter generation | `openssl` (already present on the runner and on this dev machine: LibreSSL 3.3.6 locally) [VERIFIED: live `openssl version` run] | n/a | FIX-10's per-run delimiter; `openssl rand -hex 16` |
| Go toolchain | go1.26.6 (declared `go 1.26.6` in `go.mod`; a locally-installed go1.27.1 silently shadows it unless `GOTOOLCHAIN=go1.26.6` is pinned) [VERIFIED: go.mod line 1; live `go env GOVERSION` shows 1.27.1 by default] | — | **Environment note for whoever executes this phase**: run Go commands with `GOTOOLCHAIN=go1.26.6` explicitly, or a locally newer `go` binary on `PATH` silently wins over `go.mod`'s declared version |

**Installation:** None required. Every fix in this phase edits existing files; no `go get`, no `pnpm add`.

**Version verification:** Not applicable — no new package versions to verify.

## Package Legitimacy Audit

**No external packages are added by this phase.** FIX-02's PNG/apple-touch-icon rendering does not require adding `sharp` or `rsvg-convert` to `web/package.json` — neither exists there today [VERIFIED: `rg` over `web/package.json` for `sharp|rsvg|svgo|imagemin` returned no matches this session] — and D-04 explicitly permits shipping them as **committed static files**, which avoids introducing a new devDependency entirely. Given the project's own stated preference for keeping the SPA build's JS toolchain minimal and out of the reproducible release path (STATE.md: "No JS toolchain inside the reproducible build"), the recommendation is to render the PNG/apple-touch-icon **once, outside the repo's build pipeline** (any local tool — ImageMagick, `sips`, an online converter, or a design tool export) and commit the resulting binary files directly, rather than adding an image-processing library as a devDependency for a one-time asset generation task.

| Package | Registry | Verdict | Disposition |
|---------|----------|---------|-------------|
| (none) | — | — | No new packages this phase |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram — FIX-04/05 (the graph console defects)

```
User navigates to /graph
        │
        ▼
GraphCanvas.svelte mount $effect (line 819)
        │
        ├─► cytoscape.core() constructs `cy`
        ├─► createFileGraphRenderer(cy, ...) → renderer.start()
        │         │
        │         ▼
        │   runLayout() called (line 313)
        │         │
        │         ├─► opts.cy.one('layoutstop', <our guarded callback>)   ◄── layoutGeneration token
        │         └─► opts.cy.layout({...LAYOUT_OPTIONS, fit}).run()
        │                   │
        │                   ▼
        │            cytoscape-elk Layout.run() (src/layout.js:161)
        │                   │
        │                   ├─► new ELK().layout(graph)   ◄── ASYNC, returns a Promise
        │                   │         (elkjs worker computes layout — takes real wall-clock time,
        │                   │          scales with node/edge count — 135 nodes/163 cycles @ guava)
        │                   │
        │                   └─► .then(() => nodes.layoutPositions(layout, options, getPos))
        │                             │  ▲
        │                             │  └── THIS CALLBACK FIRES WHENEVER THE PROMISE RESOLVES,
        │                             │      REGARDLESS OF WHETHER `cy` HAS BEEN DESTROYED
        │                             │      IN THE MEANTIME. cytoscape-elk's Layout.stop()/
        │                             │      destroy() (src/layout.js:183-189) are BOTH no-ops
        │                             │      ("return this;") — they cannot cancel the elkjs
        │                             │      promise or suppress this .then() from firing.
        │                             ▼
        │                    cytoscape core layoutPositions() (dist/cytoscape.esm.mjs:13216)
        │                             │  (LAYOUT_OPTIONS has no `animate` key → falsy → else branch)
        │                             ▼
        │                    nodes.positions(getFinalPos)  (line 13324)
        │                             │
        │                             ▼
        │                    elesfn.positions() (dist/cytoscape.esm.mjs:10942)
        │                             │
        │                             ├─► cy.startBatch()
        │                             ├─► ele.position(pos) for each node
        │                             └─► cy.endBatch()          ◄── CRASH SITE, see below
        │
        └─► [If the component/effect unmounts before the promise above resolves:]
                  return () => { ...; cy.destroy(); }   (line 895-897)
                            │
                            ▼
                  destroyRenderer() (dist/cytoscape.esm.mjs:15886)
                            │
                            └─► cy._private.renderer = null;  // "to be extra safe, remove the ref"
                                        (line 15897 — VERBATIM comment in the vendored source)

  ── LATER, when the still-pending ELK promise above finally resolves ──

                  cy.endBatch() (dist/cytoscape.esm.mjs:15786)
                            │
                            ├─ batchCount reaches 0
                            ├─ var renderer = this.renderer();     // renderer IS NULL
                            └─ Object.keys(_p.batchNotifications).forEach(eventName => {
                                   renderer.notify(eventName, eles)   ◄── TypeError: Cannot read
                               })                                        properties of null
                                                                          (reading 'notify')
```

### Investigation 1 — FIX-04 root cause (D-06), CONFIRMED via source read

**The exact missing guard**, quoted verbatim [VERIFIED: web/node_modules/cytoscape/dist/cytoscape.esm.mjs:15735-15810]:

The standalone `notify()` method that every *other* cytoscape mutation path uses has this guard:

```js
notify: function notify(eventName, eventEles) {
    var _p = this._private;
    if (this.batching()) { ... return; }
    if (!_p.notificationsEnabled) { return; }
    var renderer = this.renderer();

    // exit if destroy() called on core or renderer in between frames #1499 #1528
    if (this.destroyed() || !renderer) {
      return;
    }
    renderer.notify(eventName, eventEles);
},
```

`endBatch()` — which is what actually fires at the end of a batched position-write, i.e. exactly what `elesfn.positions()` (called from cytoscape-elk's async resolution path) triggers — reimplements the renderer-notify dispatch **inline, with no equivalent guard**:

```js
endBatch: function endBatch() {
    var _p = this._private;
    if (_p.batchCount === 0) { return this; }
    _p.batchCount--;
    if (_p.batchCount === 0) {
      _p.batchStyleEles.updateStyle();
      var renderer = this.renderer();

      // notify the renderer of queued eles and event types
      Object.keys(_p.batchNotifications).forEach(function (eventName) {
        var eles = _p.batchNotifications[eventName];
        if (eles.empty()) {
          renderer.notify(eventName);        // ← renderer can be null here
        } else {
          renderer.notify(eventName, eles);  // ← or here
        }
        ...
```

`renderer` becomes `null` via `destroyRenderer()`, whose own comment states the intent explicitly [VERIFIED: web/node_modules/cytoscape/dist/cytoscape.esm.mjs:15886-15897]: `cy._private.renderer = null; // to be extra safe, remove the ref`. The library's authors anticipated a late-firing async callback reaching a torn-down renderer (see the `#1499 #1528` GitHub-issue-number comment on `notify()`'s own guard) and fixed it in one call path but not the other — this is a genuine, nameable gap in cytoscape 3.34.2, not a misuse on this repo's part.

**Full call chain, traced from source, not assumed:**

1. `cytoscape-elk/src/layout.js:172-178` [VERIFIED]: `elk.layout(graph).then(() => nodes.filter(...).layoutPositions(layout, options, getPos))`. `elk.layout()` returns a Promise from elkjs (a real, node/edge-count-scaling async computation — plausibly hundreds of ms at guava's 135-collapsed-node/163-cycle scale).
2. `Layout.stop()` and `Layout.destroy()` in the same file (lines 183-189) are **both literal no-ops** (`return this; // chaining`) [VERIFIED: web/node_modules/cytoscape-elk/src/layout.js:183-189] — so `GraphCanvas.svelte`'s existing `activeLayoutRun.stop()` call (line 329-331, `runLayout`) has **zero effect** against an ELK-backed layout. The code's own comment at line 322-328 already flags this as "best-effort... not the correctness mechanism" for the `layoutstop` token — but the token check only protects the callback GraphCanvas itself registers via `cy.one('layoutstop', ...)`; it cannot protect cytoscape's own internal `endBatch()` call, which fires *before* the `layoutstop` event is even emitted (`layoutPositions`'s non-animate branch calls `nodes.positions()` — which triggers the crash — *before* it reaches `layout.emit({type: 'layoutstop', ...})` a few lines later).
3. `cytoscape.esm.mjs:13216` `layoutPositions()`: since `LAYOUT_OPTIONS` sets no `animate` key (falsy), the `else` branch runs `nodes.positions(getFinalPos)` [VERIFIED: web/src/lib/components/graph/GraphCanvas.svelte:139-152 — the `LAYOUT_OPTIONS` object has no `animate` key].
4. `elesfn.positions()` (line 10942-10967) wraps the per-node `.position()` writes in `cy.startBatch()`/`cy.endBatch()`.
5. `endBatch()` (line 15786-15810) is the crash site described above.

**Where GraphCanvas's own teardown sits relative to this:** the component's mount `$effect` (starts `web/src/lib/components/graph/GraphCanvas.svelte:819`) returns a cleanup at lines 895-897 that calls `cy.destroy()` unconditionally, with no check for whether an ELK-backed layout promise is still outstanding [VERIFIED: web/src/lib/components/graph/GraphCanvas.svelte:893-897].

**Why "mount/effect double-run" is a plausible trigger, not just a hypothesis:** this exact class of bug — a Svelte 5 effect firing twice and confusing element-identity state — has already happened for real in this codebase: STATE.md Phase 6 records "Two double-invocation Svelte 5 effect bugs fixed via identity-comparison guards ... found only by real-browser testing at guava scale" [CITED: .planning/STATE.md, Phase 06 decisions]. A mount→cleanup→remount sequence (route re-navigation, or a Svelte-5-strict-mode double effect run in dev) would call `cy.destroy()` on the *first* `cy` while the first mount's ELK promise is still in flight, then construct a *second* `cy` — the crash fires against the first (now-destroyed) instance whenever elkjs's worker finishes.

**Fix direction (D-06 compliant — our code only):** Since `cytoscape-elk`'s `stop()`/`destroy()` cannot cancel the pending elkjs promise, the two Claude's Discretion options reduce to one realistic path: **defer or gate the actual `cy.destroy()` call** rather than "cancel on destroy" (which cannot work against a no-op `stop()`). Concretely: track whether an ELK-backed `runLayout()` call is currently in flight (a boolean alongside the existing `layoutGeneration` counter, set true right before `.run()` and cleared only when that generation's `layoutstop` fires or a bounded timeout elapses), and in the mount effect's cleanup, either (a) skip `cy.destroy()` and instead null out event handlers / detach the DOM node if a layout is still pending (accepting a short-lived, self-resolving no-op `cy` instance that never gets torn down until the pending promise settles), or (b) wrap the whole `nodes.positions(...)` write in a `try { } catch { /* cy torn down mid-flight, benign */ }` at the ONE call site GraphCanvas controls — but GraphCanvas does not control that call site (it's inside cytoscape-elk/cytoscape core). **Recommendation: option (a).** This requires a live Chromium reproduction (via a temporary debug instrumentation or the new `graph-console-check.mjs` itself run against a pre-fix build) to confirm the double-mount trigger exact sequence before landing the fix — this research pass traced the mechanism from source but did not drive a live browser this session.

### Investigation 2 — FIX-05 root cause (D-07), mechanism confirmed, specific pair NOT yet identified

**The exact warning-emission code** [VERIFIED: web/node_modules/cytoscape/dist/cytoscape.esm.mjs:24263-24273]:

```js
BRp$c.checkForInvalidEdgeWarning = function (edge) {
  var rs = edge[0]._private.rscratch;
  if (rs.nodesOverlap || number$1(rs.startX) && number$1(rs.startY) && number$1(rs.endX) && number$1(rs.endY)) {
    rs.loggedErr = false;
  } else {
    if (!rs.loggedErr) {
      rs.loggedErr = true;
      warn('Edge `' + edge.id() + '` has invalid endpoints and so it is impossible to draw. ...');
    }
  }
};
```

The warning fires **exactly once per edge** (guarded by `rs.loggedErr`) whenever that edge's rendered endpoint coordinates are non-finite (`NaN`/`undefined`) — cytoscape's own message text states this is what happens "when the source node and the target node overlap" (i.e., its bounding box degenerates and edge-endpoint math divides by zero or similar). This is deterministic per-edge, not a timing flake.

**What the gate needs to do to name the pair** (this is D-09's `graph-console-check.mjs` job, not something this research session could complete without a live Chromium session against the pinned guava checkout):
1. Capture the `console.warn` text via Playwright's `page.on('console', ...)` — it already contains the edge id (`Edge \`<id>\` has invalid endpoints...`).
2. `page.evaluate()` back into the live `cy` instance: `cy.getElementById(edgeId).source().position()` / `.target().position()` and their `boundingBox()`s, to confirm the overlap and get exact coordinates.
3. Cross-reference against `LAYOUT_OPTIONS`'s current ELK config (`web/src/lib/components/graph/GraphCanvas.svelte:139-152`) — today it sets `elk.hierarchyHandling`, the three interactive-strategy keys, and a per-parent `elk.padding` via `nodeLayoutOptions`, but sets **no explicit `elk.spacing.nodeNode` or `elk.layered.spacing.nodeNodeBetweenLayers`** [VERIFIED: web/src/lib/components/graph/GraphCanvas.svelte:139-165 — full option block read, no spacing keys present]. At 135 collapsed nodes / 163 cycles, ELK's *default* spacing may be insufficient to separate two collapsed-directory nodes whose computed sizes (via `nodeDimensionsIncludeLabels`-style logic, also unset here) leave them at coincident coordinates.

**Recommendation for the plan:** treat this as a two-step task — (1) build the diagnostic capability above into `graph-console-check.mjs` so a live run against the pinned guava corpus names the exact pair and its coordinates, (2) apply an ELK spacing/sizing fix informed by that diagnosis (most likely `elk.spacing.nodeNode` and/or `elk.layered.spacing.nodeNodeBetweenLayers`, possibly combined with `nodeDimensionsIncludeLabels: true` if the warning correlates with label-heavy collapsed nodes), verified via live Chromium on both corpora per D-07's constraint.

### Investigation 3 — FIX-07/08 (D-13..D-15): CONFIRMED live reproduction + confirmed current code shape

**Current code, read this session** [VERIFIED: internal/daemon/watchdog.go:1-56]:

```go
// getppid is an unexported package-level test seam: ...
var getppid = os.Getppid

func startWatchdog(ctx context.Context, cancel context.CancelFunc, interval time.Duration) (stop func()) {
	original := getppid()
	...
	go func() {
		...
		ticker := time.NewTicker(interval)
		...
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if parentChanged(original) {
					cancel()
					return
				}
			}
		}
	}()
	return func() { <-done }
}
```

`watchdogInterval = 1 * time.Second` (a real `time.Ticker`, not injectable) [VERIFIED: internal/daemon/watchdog.go:13]. There is **no `watchdog_windows.go`** in this codebase — `find internal/daemon -iname '*watchdog*'` returns only `watchdog.go`, `watchdog_posix.go` (`//go:build !windows`), and `watchdog_test.go` [VERIFIED: live `find` this session]. This corrects the phase brief's own assumption ("check `watchdog_windows.go`'s shape under the `GOOS=windows` build tag") — **there is no windows-specific watchdog file to touch**, consistent with README's documented decision that native Windows support was dropped in v0.4.0 (WSL2 only) [CITED: .planning/STATE.md Blockers/Concerns section].

**A prior, partial fix already exists and is misleadingly labeled "closed" in its own doc comment.** `internal/daemon/testbudget_test.go` (added 2026-08-07, v0.3.0 Phase 4, commit `13f2875`) [VERIFIED: `git log --diff-filter=A` this session] states in its header comment: "the two shared test-only helpers that close MAINT-01 (issue #13...) and MAINT-02 (issue #17...)". **Both GH #13 and GH #17 are still OPEN today** [VERIFIED: live `gh issue view 13/17 --json state` this session, both `"state":"OPEN"`, created 2026-08-01 — six days *before* this "closing" fix landed]. Reading `04-02-SUMMARY.md` [VERIFIED: .planning/milestones/v0.3.0-phases/04-supply-chain-coverage-daemon-substrate-fixes/04-02-SUMMARY.md] resolves the apparent contradiction:

- The fix (`testBudget()` + `joinDaemonRun()`, applied at every `Daemon.Run`/`RunWithRetry` spawn site) makes every test **join its spawned goroutine via `t.Cleanup` before restoring any package-level seam** — this eliminates the specific `-race` interleaving GH #13 first reported (an orphaned goroutine from one test reading `getppid` while a different, later test's cleanup writes it back).
- Its own "Known Limitation" section is explicit: the **race component is proven eliminated** (0/3 post-fix full-suite `-race` runs vs 3/3 pre-fix), but the **plain-timeout tail persists under load** and was *not* reduced to zero across 8 unfiltered runs even at the maximum allowed scale (25x-40x) — "A human reviewer should treat MAINT-01 as fully closed and MAINT-02 as closed **at its structural cause**... with a residual, honestly-measured, environment-confounded tail."
- Crucially: **the package-level `var getppid = os.Getppid` was never removed.** `TestRunWatchdogCancelsRunOnSimulatedReparent` (current code, read this session) [VERIFIED: internal/daemon/daemon_test.go:306-316] still does `origGetppid := getppid; ...; getppid = func() int {...}` — a direct write to the same package global. The fix serializes access via join discipline; it does not make the race *structurally impossible* the way D-13 requires. This is exactly Pitfall 15's own warning: "fixing the symptom... does not fix the underlying design defect... a new test added later... can reintroduce the race independently."

**Live reproduction this session** [VERIFIED: live command run, `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...`, 2026-09-14]:

```
--- FAIL: TestRunWatchdogCancelsRunOnSimulatedReparent (250.35s)
    daemon_test.go:352: Run did not return after a simulated reparent — watchdog is not wired into Run (D-07/D-08)
FAIL	github.com/seanb4t/codegraph-go/internal/daemon	324.076s
```

Full run: 5m45s wall clock, **zero `WARNING: DATA RACE` lines anywhere in the output**, every other of the ~55 packages passed. This is a clean, direct reproduction of GH #17's exact symptom (functional timeout under full-suite load, isolated-run passes) **on the current codebase, today** — confirming FIX-07 is not accidentally already fixed, and that the existing `testBudget` scaling (currently 25x default, clamped at 40x) is not sufficient headroom under this session's load, exactly as `04-02-SUMMARY.md`'s own "Known Limitation" predicted would keep happening. The isolated companion run (`go test -race -run 'TestConvergenceTwoSessions|TestWatchdogCancelsOnReparent' -count=5 ./internal/daemon/`) passed cleanly in 28.3s with zero races [VERIFIED: live command run].

**GH #13's leaked-goroutine half (D-15):** `soak_test.go`'s `TestConvergenceTwoSessions` already calls `joinDaemonRun(t, cancelA, runErrA)` and `joinDaemonRun(t, cancelB, runErrB)` for both of its spawned `RunWithRetry` goroutines [VERIFIED: internal/daemon/soak_test.go, code read this session shows both join calls present]. Combined with the full-suite run above showing **zero** "goroutine leak" messages (the exact string `joinDaemonRun`'s own failure path would emit) and zero goleak `TestMain` failures, **the specific leaked-goroutine mechanism GH #13 originally reported appears to already be closed** by the v0.3.0 Phase 4 join-discipline fix — but this is an inference from one run's absence of the symptom, not a positive falsification attempt against the pre-join-discipline code (that code no longer exists to test against). Recommend the plan record this as: "GH #13's specific leaked-goroutine reproduction was not observed in a fresh 2026-09-14 full-suite run; the join-discipline fix from v0.3.0 Phase 4 appears to have closed it, but the underlying `getppid` package-global pattern this issue was filed against still exists in production code and is what D-13 removes."

**Fix direction for the plan (matches D-13/D-14 exactly, now evidence-backed):**
1. Thread `getppid func() int` and a ticker/interval source through `startWatchdog`'s parameters, stored as unexported `Daemon` fields set via a new `Option` (mirrors `WithProbe`, `internal/daemon/daemon.go:155-161`) — this makes the race structurally impossible rather than merely rare.
2. `parentChanged` (`internal/daemon/watchdog_posix.go:14`, `func parentChanged(original int) bool { return getppid() != original }`) needs to take the injected func rather than reading the package var, or become a method on a struct that owns it.
3. For the ticker: since D-14 says "inject the ticker/clock alongside the seam" and "asserts on a signal, not a deadline," the cleanest shape given the existing test helpers (`testBudget`, `joinDaemonRun`) is an injectable `<-chan time.Time` (a test can send on it directly, no real ticker involved) with production code defaulting to `time.NewTicker(watchdogInterval).C` — this removes the *real 1-second wall-clock wait* from the test's critical path entirely, which is what actually caused this session's live 250s failure (the ticker only fires once per real second, and under CPU contention the goroutine scheduling to observe that fire is what got starved).
4. **No test-only setter mirrors the "no exported setter" convention** the codebase already uses for `onSync`/`onSyncStart`/`syncFn`/`onWatchOpen` (all four are unexported fields written directly by same-package `_test.go` files, not through an `Option`) [VERIFIED: internal/daemon/daemon.go:97-137, internal/daemon/daemon_test.go direct assignment at lines 375/509/590/679]. Only `WithProbe` uses the exported `Option` pattern, because it is also used by *production* code (`serve --mcp`). Since `getppid`/ticker injection is test-only, the **precedent favors an unexported field set directly by `daemon_test.go`/`watchdog_test.go`**, not a new exported `Option` — CONTEXT.md's own phrasing ("a test-only `Option`") should be read as "test-only unexported field," consistent with the codebase's own dominant convention, unless the planner has a reason to diverge.

### Investigation 4 — FIX-06 (D-10..D-12): current code and sentinels confirmed

**Current `priorCoverageGeneration`** [VERIFIED: internal/cli/index.go:30-48, quoted in full]:

```go
func priorCoverageGeneration(storeDir string) int64 {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return 0
	}
	defer store.Close()

	r, err := store.Snapshot()
	if err != nil {
		return 0
	}
	defer r.Close()

	meta, err := r.GetMeta()
	if err != nil {
		return 0
	}
	return meta.GetCoverageGeneration()
}
```

Called at `internal/cli/index.go:101`, immediately followed by `os.RemoveAll(storeDir)` at line 107 — confirming the exact defect: **every** `Open`/`Snapshot`/`GetMeta` error (lock-held, corrupt, or genuinely never-indexed) collapses to the same `return 0`, and the caller proceeds straight to `RemoveAll` regardless.

**Sentinels to distinguish, both read this session:**
- `graphstore.ErrNotFound = errors.New("graphstore: not found")` [VERIFIED: internal/graphstore/pebble_store.go:21]
- `graphstore.ErrStoreLocked = errors.New("graphstore: store lock held")` [VERIFIED: internal/graphstore/pebble_store.go:110] — set by `classifyOpenError` inside `Open`'s own bounded retry (~400ms per the doc comment at line ~120), so by the time `priorCoverageGeneration` observes it, `Open`'s retry budget is already exhausted.

**Message precedent to mirror** (per the phase brief's own suggestion, confirmed present) [VERIFIED: internal/daemon/lock.go:208]:

```go
return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
```

D-10 wants the FIX-06 refuse message to read like this — same "pid=N — stop it first, or run `X`" shape, substituting `codegraph daemon stop` / `codegraph unlock` per the phase's own pointer (note: `codegraph unlock` becomes `daemon unlock` in Phase 3 per VERB-04 — this phase should point at whichever verb is live *today*, i.e. `codegraph unlock`, since Phase 3 has not run yet).

**Test shape to model, confirmed present** [VERIFIED: internal/graphstore/open_lock_test.go, full file read]: `TestOpenSecondOpenInProcessReturnsErrStoreLocked` holds a first `Open` handle, opens a second `Open` on the same dir, and asserts `errors.Is(err, ErrStoreLocked)`. D-12's regression test is the same shape at the CLI layer: hold a `graphstore.Open` handle in the test, invoke `codegraph index --force`'s command path (in-process, via cobra's `RunE`, or a subprocess), and assert the command errors (non-zero) **and** `os.RemoveAll` never ran (e.g. assert the held store's files are untouched) — this must be watched fail against the current code first (rule `84d1gfpywd`), since today it silently proceeds to floor-0 + `RemoveAll`.

**Implementation shape for the fix:** `priorCoverageGeneration` needs to return an error (or a sentinel-carrying result) instead of collapsing everything to `int64`, so the caller in `newIndexCmd`'s `RunE` (`internal/cli/index.go:101`) can branch: `errors.Is(err, graphstore.ErrNotFound)` → floor 0, silent (D-11); `errors.Is(err, graphstore.ErrStoreLocked)` → refuse, non-zero exit, before `RemoveAll` (D-10); anything else → warn to stderr, floor 0, proceed (D-11's corrupt-store path).

### Investigation 5 — FIX-09 (D-16): write-site investigation confirms no normalization needed

**All three `Repo` write sites, read this session** [VERIFIED: tools/bench/runner/main.go:370, :691, :759]:

1. `main.go:370` (publish mode): `Repo: entry.Name` — a named real-corpus label (e.g. "guava"). **Never reaches `CheckRegression`** — publish mode "publishes raw numbers and never gates" (comment at the same call site).
2. `main.go:691` (regression mode, per-trial measurement): `Repo: fmt.Sprintf("synthetic-seed%d-count%d", cfg.seed, cfg.count)` — a fully deterministic synthesized string, never a filesystem path.
3. `main.go:759` (regression mode, median-of-trials aggregation): `Repo: trials[0].Repo` — carries the same synthesized string forward unchanged (all trials in one run share the same `cfg.seed`/`cfg.count` by construction, so this can never diverge across trials, per the existing comment on the adjacent `ScratchFS` field at the same site).

`CheckRegression` is invoked from exactly one call site in the whole runner (`main.go:627`), and only `-mode regression` reaches it (`-mode` accepts only `"publish"` or `"regression"`, confirmed at line 149) [VERIFIED: `rg -n "unknown -mode|mode.*regression|CheckRegression\(" tools/bench/runner/main.go`]. This confirms **D-16's claim exactly**: the only `Repo` values `CheckRegression` will ever compare are the synthesized `synthetic-seed{N}-count{M}` strings, which are stable and representation-free (no path-vs-URL ambiguity Pitfall 16 warned about is reachable through this call path) — a strict-equality guard following the existing GOOS/Runner/ScratchFS template (`internal/bench/regression.go:56-110`, each following the identical "non-empty-vs-non-empty mismatch refuses; empty-either-side means never-recorded, not wildcard" shape) is correct and sufficient, with **zero normalization logic needed**.

**Template to copy** [VERIFIED: internal/bench/regression.go:91-110, the `ScratchFS` guard, structurally identical to what a `Repo` guard needs]:

```go
if baseline.ScratchFS != current.ScratchFS {
    return fmt.Errorf(
        "bench: scratch filesystem mismatch: baseline was measured on scratch_fs %s but "+
            "this run is %s; ...",
        scratchFSString(baseline.ScratchFS), scratchFSString(current.ScratchFS),
    )
}
```

A `Repo` guard follows this exact shape (with a `repoString` helper mirroring `runnerString`/`scratchFSString`'s "(not recorded)" degrade-empty behavior), placed alongside the other three category-error checks, before the `FilesPerSec <= 0`/`PeakRSSBytes <= 0` validity checks. `TestCheckRegression` (`internal/bench/regression_test.go:8`, one `t.Run(tt.name, ...)` table loop at line 541) [VERIFIED: existing test structure confirmed via `rg`] gets one new subtest in the same table.

### Investigation 6 — FIX-10 (D-17): both workflows read verbatim, fix scope confirmed

Both files' vulnerable blocks are byte-identical in shape [VERIFIED: .github/workflows/require-issue-link.yml:44-50, .github/workflows/pr-template-format.yml:41-46, both files read in full]:

```sh
{
  echo "list<<PRFILES_EOF"        # or "files<<PRFILES_EOF" in pr-template-format.yml
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "PRFILES_EOF"
} >> "$GITHUB_OUTPUT"
```

Neither workflow checks out the PR head (`require-issue-link.yml` does no checkout at all; `pr-template-format.yml` explicitly checks out the base branch, per its own comment) — the injection surface is narrow (the changed-file-path list only), matching GH #15's own stated bounded-impact assessment.

**No existing bats/shellcheck test harness exists in this repo** [VERIFIED: `find . -iname '*.bats'` returned nothing; `scripts/` contains only `inject-cosign-key.sh`, `live-push-concurrency-check.sh`, `live-push-probe.go`, `pr_template_policy.py` — no shell-test convention]. The closest precedent is `scripts/inject-cosign-key.sh`, which was **extracted from inline workflow YAML into a standalone script specifically so it could be exercised via a `task` target** (`Taskfile.yml:2156,2672` invoke it directly) — this is the established pattern for "test a piece of workflow shell logic outside of an actual GitHub Actions run." **Recommendation:** extract the delimiter-generation + heredoc-write block from both workflows into a single shared `scripts/write-multiline-output.sh` (or similar), parameterized by the output key name and a value source, invoke it from both workflows, and add a Taskfile target that runs it with a stub `gh` on `PATH` (a tiny shell function or script named `gh` earlier in `PATH` that echoes a fixed file list including a `PRFILES_EOF`-named path) and asserts the resulting `$GITHUB_OUTPUT` file parses back to the full, uncorrupted list. This gives a real, repeatable, CI-runnable gate (matching `84d1gfpywd`'s RED-before-GREEN requirement: run the test against the *old* fixed-delimiter script first to see it actually fail/corrupt, then against the fixed per-run-delimiter version to see it pass) without introducing a new test framework (bats) this repo has never used.

`openssl` is confirmed present locally (LibreSSL 3.3.6) [VERIFIED: live `which openssl && openssl version`]; its presence on `ubuntu-latest` (the runner both workflows use) is extremely well-established (GitHub's hosted runner images document OpenSSL as a default-installed tool) but was **not independently verified against a live `ubuntu-latest` runner this session** — tag this specific claim `[ASSUMED]` for the Assumptions Log, even though it is very low-risk (the phase's own `-scratch-fs`/CI infrastructure already assumes a standard Ubuntu toolchain throughout).

### Investigation 7 — FIX-11 (D-18): decision mechanics, no code change

GH #20's two follow-ups (read in full, including the PR #57 discriminator comment) require **no code change**, only two recorded decisions:
1. Follow-up 1 (Namespace cache volume on 8×16): close **won't-do** — the issue's own text already states the bar ("matching `ubuntu-latest`'s stability") and the reasoning (free + already 28× more stable). This is a documentation/close action on GH #20 plus a note in `tools/bench/BASELINE.md`.
2. Follow-up 2 (+44.8% baseline drift): run the one discriminator GH #20 itself specifies — check out the commit the old baseline (11,279.59 files/s, reblessed 2026-07-31) was recorded at, run the regression benchmark on today's runner with the same `-seed 42 -count 120000`, compare against today's 16,330 files/s figure. `git log -S'11279' -- tools/bench/baseline.json` is the way to find that commit [not run this session — recommend the executing plan run this directly, since it is a one-command lookup with no ambiguity]. This is a `workflow_dispatch`-triggered manual run (the regression bench's own runner accepts `-mode regression -seed -count` flags, confirmed present at `tools/bench/runner/main.go:159` and surrounding flag definitions), recorded in `tools/bench/BASELINE.md` per D-18's own text, then GH #20 is closed with both decisions on the record.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting a torn-down cytoscape instance | A custom "is this cy still alive" wrapper duplicating cytoscape's own `destroyed()` | `cy.destroyed()` (already exists, used by `notify()` itself) | Cytoscape already exposes exactly this predicate; GraphCanvas's fix should check it (or an equivalent generation/in-flight flag) rather than reinvent liveness tracking |
| A new shell-test framework (bats) for FIX-10 | Introducing `bats-core` as a new devDependency for one test | A plain shell script + a Taskfile target with a stubbed `gh` (mirrors `inject-cosign-key.sh`'s existing precedent) | This repo has zero bats/shellcheck infrastructure today; adding a framework for a single test is disproportionate and outside this phase's stated file scope |
| Store-lock detection in `internal/cli` | Re-implementing platform-specific lock-held error sniffing in `index.go` | `graphstore.ErrStoreLocked` / `errors.Is` (already exists, already classifies platform-specific pebble errors) | `graphstore` is the sole pebble-aware package by design (D-04a); bypassing it from `internal/cli` would violate the existing architecture boundary the archtest enforces |
| Watchdog time control | A hand-rolled fake clock library | A simple injectable `<-chan time.Time` (or interval override) alongside the existing per-instance seam pattern | The codebase already has a working, minimal test-seam convention (`onSync`/`syncFn`/etc.) — introducing a general-purpose clock abstraction is over-engineering for one ticker |

**Key insight:** every "don't hand-roll" here is really "don't hand-roll — this repo already has the primitive; use it consistently with sibling code," not "go get a library." This phase's own architecture already contains every tool needed to fix all nine items.

## Common Pitfalls

### Pitfall: widening the watchdog timeout to "fix" FIX-07
**What goes wrong:** `TestRunWatchdogCancelsRunOnSimulatedReparent`'s own comment (line 344-352, read this session) already admits its timeout was bumped once ("10s (not the file's usual 5s)... under heavy full-suite parallel load... the ticker's next fire after the simulated reparent can be delayed well past 1s") — a second bump is the obvious next move and is explicitly forbidden by D-14.
**Why it happens:** it's a one-line change with immediate local payoff.
**How to avoid:** the injectable ticker (D-14) removes the real-wall-clock dependency entirely rather than buying more of it.
**Warning signs:** a diff touching only a numeric constant in `watchdog.go`/`daemon_test.go`/`testbudget_test.go` with no signal/channel-injection change.

### Pitfall: treating the FIX-08 fix as done because the race isn't reproducing right now
**What goes wrong:** this session's own live full-suite `-race` run showed zero races — a plausible reading is "GH #13 is already fixed, skip it." D-13 explicitly requires removing the package-level `var getppid` regardless, and this research confirms the global **still exists** in production code today.
**How to avoid:** implement the per-instance seam per D-13 regardless of whether the race currently reproduces; absence of a currently-observed race is not evidence the pattern is safe (Pitfall 15's own argument, now doubly confirmed by the v0.3.0 Phase 4 precedent of a "closed" issue that isn't).

### Pitfall: patching cytoscape/cytoscape-elk directly for FIX-04
**What goes wrong:** the missing `endBatch()` guard is a genuinely tempting one-line patch (`pnpm patch cytoscape` to add the same `if (!renderer) return;` check `notify()` has). D-06 explicitly forbids this (no `pnpm patch`, no `patches/`).
**How to avoid:** the fix must live in `GraphCanvas.svelte`'s own teardown timing, per Investigation 1's recommended direction.

### Pitfall: fixing only one of the two `pull_request_target` workflows for FIX-10
**What goes wrong:** `require-issue-link.yml` is the one GH #15 names in its title, but `pr-template-format.yml` has the byte-identical vulnerable shape. D-17 and Pitfall 17 both call this out explicitly.
**How to avoid:** the shared-script extraction recommended in Investigation 6 makes "fix only one" structurally hard — both workflows call the same script.

### Pitfall: a naive `if baseline.Repo != current.Repo` without checking write sites first (FIX-09)
**What goes wrong:** Pitfall 16 warned this could false-positive on a legitimate corpus rename. Investigation 5 confirms this specific repo's write sites make that risk moot for the actual `CheckRegression` call path — but a future refactor that starts calling `CheckRegression` from publish-mode `Metrics` (with `Repo: entry.Name`, a real corpus name) would reintroduce exactly this risk.
**How to avoid:** the guard's doc comment should state explicitly (as the other three guards' comments do) that `Repo` is expected to be the regression-mode synthesized id, so a future reader who considers wiring publish-mode Metrics through `CheckRegression` sees the assumption spelled out.

## Runtime State Inventory

Not applicable — this phase is a defect/flake burn-down, not a rename, refactor, or migration. No renamed identifiers, no data migration. (FIX-06's fix changes CLI *behavior* on an existing error path; FIX-07/08's fix changes an *internal test seam*; neither renames anything a user-facing string, database key, or external config references.)

## Code Examples

### FIX-06: the branching shape `newIndexCmd`'s `RunE` needs

```go
// Source: pattern derived from graphstore.ErrStoreLocked (internal/graphstore/pebble_store.go:110),
// graphstore.ErrNotFound (internal/graphstore/pebble_store.go:21), and the ErrLockLive message
// shape at internal/daemon/lock.go:208 — all read this session.
coverageGenerationFloor, err := priorCoverageGeneration(storeDir) // signature changes to return error
switch {
case errors.Is(err, graphstore.ErrNotFound):
    // never indexed — floor stays 0, no message (D-11)
case errors.Is(err, graphstore.ErrStoreLocked):
    return fmt.Errorf("%w: another process holds the store — stop it first (`codegraph daemon stop`) "+
        "or run `codegraph unlock` once it has exited", graphstore.ErrStoreLocked) // D-10, before RemoveAll
case err != nil:
    fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not read the prior coverage generation (%v) — "+
        "rebuilding from floor 0\n", err) // D-11, corrupt/unreadable store
}
```

### FIX-09: the `Repo` guard to add to `CheckRegression`

```go
// Source: internal/bench/regression.go:91-110 (the ScratchFS guard), verbatim template.
if baseline.Repo != current.Repo {
    return fmt.Errorf(
        "bench: corpus mismatch: baseline was measured against %s but this run is %s; "+
            "a corpus/seed/count change is a measurement-frame change of the same kind as "+
            "GOOS/GOARCH, Runner, or ScratchFS, so this comparison would be meaningless. An "+
            "empty repo value means it predates repo recording, which is not a wildcard match "+
            "against a recorded one — re-bless the baseline (tools/bench/BASELINE.md) instead "+
            "of comparing across them",
        repoString(baseline.Repo), repoString(current.Repo),
    )
}
```

### FIX-10: per-run delimiter (both workflows)

```sh
# Source: GH #15's own suggested fix, cross-checked against GitHub's documented
# multi-line $GITHUB_OUTPUT guidance (per Pitfall 17's citation).
DELIM="PRFILES_$(openssl rand -hex 16)"
{
  echo "list<<${DELIM}"
  gh pr view "$PR_NUMBER" --repo "$GITHUB_REPOSITORY" --json files --jq '.files[].path'
  echo "${DELIM}"
} >> "$GITHUB_OUTPUT"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Package-level `var getppid = os.Getppid` test seam, serialized only by test join discipline | Per-instance injected `getppid`/ticker field on `Daemon` | This phase (D-13/D-14) | Removes the race structurally rather than relying on every future test author remembering the join-before-restore discipline |
| `priorCoverageGeneration` returning a bare `int64` that collapses every error to floor-0 | Returns an error the caller classifies via `errors.Is` | This phase (D-10/D-11) | Distinguishes "never indexed" from "actively locked" from "corrupt", enabling a refuse vs. warn-and-proceed split |
| Fixed heredoc delimiter for `$GITHUB_OUTPUT` multi-line values | Per-run random delimiter | This phase (D-17) | Closes a real (if narrow-impact) GitHub Actions injection class over `pull_request_target`-sourced untrusted content |

**Deprecated/outdated:** the stock SvelteKit favicon (`web/src/lib/assets/favicon.svg`) and its `import`-based, Vite-inlined delivery — replaced by static files under `web/static/` per D-04.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `openssl` is present on the `ubuntu-latest` GitHub-hosted runner both `pull_request_target` workflows use | Investigation 6 (FIX-10) | Low — extremely well-established GitHub runner image content; if wrong, `openssl rand` fails loudly at workflow run time (fail-closed, not a silent security gap) and `date +%s%N` + `$RANDOM` is a trivial fallback |
| A2 | `@playwright/test`'s currently-installed version is unchanged from prior phases and needs no re-verification for the new `graph-console-check.mjs` | Standard Stack | Low — this phase does not add Playwright usage patterns beyond what the five existing `*-check.mjs` scripts already exercise |
| A3 | Rendering the 32px PNG / 180px apple-touch-icon outside the repo's build pipeline (any local tool) and committing the result is preferable to adding an image-processing devDependency | Package Legitimacy Audit / FIX-02 | Low-medium — this is Claude's Discretion per CONTEXT.md, not a locked decision; if the executing agent has no local image tool available, a devDependency (e.g. `sharp`) becomes the fallback and would need its own legitimacy check at that time |
| A4 | The FIX-04 fix direction (defer/gate `cy.destroy()` rather than "cancel on destroy") is correct without having driven a live Chromium reproduction this session | Investigation 1 (FIX-04) | Medium — the call chain and missing guard are verified from source with high confidence, but the *exact* trigger sequence (which effect double-fires, under what navigation pattern) was traced by code-reading, not live-observed; the implementing plan should still budget for a live-Chromium diagnostic pass before committing to the fix shape |
| A5 | "GH #13's leaked-goroutine mechanism is already closed" (Investigation 3, D-15) | Investigation 3 (FIX-07/08) | Medium — this is an inference from the absence of the goroutine-leak failure signature in one fresh full-suite run, not a positive falsification against the original pre-fix code (which no longer exists to test). The plan should still run the specific reproduction GH #13 describes (`go test -race -run 'TestConvergenceTwoSessions|TestWatchdogCancelsOnReparent' -count=5`, done) and consider it strong-but-not-conclusive evidence |

**If this table is empty:** N/A — see above.

## Open Questions

1. **What is the EXACT overlapping node pair at guava scale (FIX-05)?**
   - What we know: the mechanism (non-finite endpoint coordinates from an ELK-computed overlap), the warning-emission code, and the diagnostic technique to find it (capture the edge id from `console.warn`, then `page.evaluate` the two endpoints' positions/bounding boxes).
   - What's unclear: which specific collapsed-directory pair, and whether the cause is missing spacing options, missing `nodeDimensionsIncludeLabels`, or something else entirely (e.g. a zero-child collapsed node whose bounding box degenerates to a point).
   - Recommendation: build the diagnostic into `graph-console-check.mjs` itself (D-09 already requires building this script) as the first task, then let its live output against the pinned guava corpus drive the actual ELK-option fix — this is unavoidably an implementation-time discovery, not something research-without-a-browser could complete.

2. **Exact trigger sequence for FIX-04's double-mount / stale-teardown race.**
   - What we know: the full call chain and the exact missing guard in cytoscape; that Svelte-5 double-effect-invocation bugs have happened for real in this codebase before (STATE.md Phase 6).
   - What's unclear: whether the trigger in THIS component is a genuine Svelte strict-mode double-mount, a route re-navigation while a layout is still computing, or simply "any navigation away from `/graph` before ELK finishes at guava scale" (which WINDOWS #26 itself already noted does NOT reproduce — "Does NOT fire when navigating away mid-layout" — meaning the trigger might be something more specific than plain unmount timing).
   - Recommendation: instrument (temporarily) `GraphCanvas.svelte`'s mount effect with a console log on entry/cleanup and reproduce live against this repo's own (smaller, faster-to-lay-out) index first, where the timing window is easier to hit deliberately (e.g. by throttling CPU in devtools) before attempting it at guava scale.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain matching `go.mod`'s `go 1.26.6` | All Go-side fixes (FIX-06/07/08/09) | ✓ (via `GOTOOLCHAIN=go1.26.6`) | go1.26.6 confirmed present as a cached toolchain | A locally-newer `go` (1.27.1 seen this session) on `PATH` will silently be used instead unless `GOTOOLCHAIN` is pinned — **not a blocker, but a real footgun**: this session's first race-repro attempt failed to even build (`cockroachdb/swiss` incompatible with go1.27.1) until `GOTOOLCHAIN=go1.26.6` was set explicitly |
| `openssl` | FIX-10's delimiter generation | ✓ locally (LibreSSL 3.3.6); [ASSUMED] on `ubuntu-latest` | — | `$RANDOM`+`date +%s%N` string if genuinely absent |
| `@playwright/test` + a real Chromium | FIX-04/05's live verification, D-09's new check | Not independently re-verified this session (unchanged from prior phases' working state) | — | None needed — this is pre-existing, working infrastructure |
| `git`, `gh` CLI | FIX-11's discriminator run, GH issue closing | ✓ (both used live this session) | — | — |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** `openssl` (trivial shell fallback if ever absent).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (standard library), `-race` for concurrency packages |
| Framework (web live checks) | `@playwright/test`-style `.mjs` scripts (not the vitest unit suite) — no formal framework, hand-rolled harness matching the five existing `web/scripts/*-check.mjs` |
| Config file | none — no `jest.config`/`playwright.config` governs these `.mjs` scripts; they are standalone Node scripts invoked directly |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/daemon/... ./internal/graphstore/... ./internal/bench/...` |
| Full suite command | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` (the exact command that reproduced FIX-07 live this session) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| FIX-02 | Favicon loads under unchanged CSP, no CSP-blocked console entry | manual/live-browser (existing `spa_test.go` CSP assertion stays green as a negative control) | `go test ./internal/uiserver/... -run TestSPA` (existing) + a live Chromium load of `/graph` or `/` checking `page.on('console')`/network for the icon | ✅ `internal/uiserver/spa_test.go` exists; ❌ no automated favicon-content check exists yet — Wave 0 gap |
| FIX-04 | Zero uncaught page errors on `/graph`, both corpora | live-browser | new `web/scripts/graph-console-check.mjs` (D-09) | ❌ Wave 0 gap — script does not exist yet |
| FIX-05 | Zero `console.warn`/`console.error` from our code, both corpora | live-browser | same `graph-console-check.mjs` | ❌ same Wave 0 gap as FIX-04 |
| FIX-06 | `index --force` refuses/warns correctly on a held/corrupt store | unit/integration | new test modeled on `internal/graphstore/open_lock_test.go`, in `internal/cli` | ❌ Wave 0 gap — test does not exist yet |
| FIX-07 | Watchdog test passes deterministically under full-suite load | integration | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` (must show `TestRunWatchdogCancelsRunOnSimulatedReparent` passing — this session confirms it currently fails) | ✅ test exists (`internal/daemon/daemon_test.go:302`), currently RED under the full-suite command |
| FIX-08 | `go test -race ./internal/daemon/...` clean, seam per-instance | unit + code-shape | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/daemon/...` **plus** a code-shape check (`rg -n '^var getppid' internal/daemon/watchdog.go` returns nothing after the fix) | ✅ race test exists and currently passes in isolation; the code-shape assertion is new — Wave 0 gap for an automated check, or verified by code review |
| FIX-09 | `CheckRegression` refuses a `Repo` mismatch | unit | `go test ./internal/bench/... -run TestCheckRegression` | ✅ `internal/bench/regression_test.go` exists with a table-driven structure ready for a new subtest |
| FIX-10 | Neither workflow's heredoc is exploitable via a `PRFILES_EOF`-named path | new shell/script test | new script + Taskfile target (Investigation 6) | ❌ Wave 0 gap — no such test exists |
| FIX-11 | GH #20 follow-ups each end in a recorded decision | manual/process | `git log -S'11279' -- tools/bench/baseline.json` + a `workflow_dispatch` bench run + `tools/bench/BASELINE.md` edit + `gh issue close 20` | N/A — process/documentation gate, not a code test |

### Sampling Rate

- **Per task commit:** the narrowest applicable command above (e.g. `go test ./internal/bench/...` for FIX-09; the new `graph-console-check.mjs` for FIX-04/05; `go test ./internal/cli/...` for FIX-06).
- **Per wave merge:** `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` (the full-suite command) — this is the ONLY command that actually exercises FIX-07/08's fix under the load condition that matters; a per-package or isolated run cannot validate this requirement (confirmed this session: isolated run passes both before and after nothing changes, so it is not a discriminating test for this requirement).
- **Phase gate:** Full suite green (including `-race`) before `/gsd-verify-work`, plus the live `graph-console-check.mjs` run against both corpora recorded under `corpora/`.

### Wave 0 Gaps

- [ ] `internal/cli/index_lock_test.go` (or similar) — covers FIX-06, modeled on `internal/graphstore/open_lock_test.go`
- [ ] `web/scripts/graph-console-check.mjs` — covers FIX-04 and FIX-05 together (D-09 names this as one script)
- [ ] `scripts/write-multiline-output.sh` (or chosen name) + its Taskfile-driven test — covers FIX-10
- [ ] A code-shape assertion (test or code-review checklist item) that `internal/daemon/watchdog.go` has no package-level `var getppid` after the fix — covers FIX-08's "structurally impossible" bar, since a passing `-race` run alone cannot prove a race is *impossible*, only that it didn't happen in this run
- [ ] No framework install needed — every gap above is a new file within the existing Go/`.mjs` testing conventions, not a new toolchain

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V1 Architecture | yes | The `internal/cli` → `internal/graphstore` boundary (D-04a, enforced by `TestNoPackageBypassesGraphStore`) must remain intact when FIX-06 is implemented — no ad hoc pebble-error sniffing added to `internal/cli` |
| V5 Input Validation | yes | FIX-10: `pull_request_target`-sourced content (PR changed-file paths) must never be trusted as shell-safe; the delimiter fix is the control, but the recommended shared-script extraction should also `printf '%s\n'` paths rather than relying solely on delimiter uniqueness (per Pitfall 17's own secondary caution about embedded newlines) |
| V7 Error Handling & Logging | yes | FIX-06: the corrupt-store warning path must not leak sensitive path information beyond what's already user-visible (`storeDir` is already a user-known local path, so no new leakage) |
| V14 Configuration | yes | FIX-10: workflow `permissions:` blocks on both files are already minimal (`contents: read`, `issues: write`, `pull-requests: read` — no `contents: write`) and must remain unchanged by this fix — confirmed read this session on both files |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| GitHub Actions multi-line `$GITHUB_OUTPUT` heredoc injection via untrusted `pull_request_target` content | Tampering | Per-run unpredictable delimiter (D-17); GitHub's own documented guidance |
| A "fix" that widens a deliberately-tight security/reliability posture to make a symptom disappear (CSP `img-src data:`, watchdog timeout, dependency patch bypassing a library's own safety intent) | Tampering / Denial of Service (of the guard itself) | This entire phase's own governing constraint: fix causes, never widen policies — already enforced by D-05, D-06, D-14 |
| Store-lock TOCTOU between a read-check and a destructive `RemoveAll` | Tampering (of index data) | FIX-06's refuse-before-`RemoveAll` ordering, matching the existing `graphstore.Open`'s own bounded-retry-then-classify discipline |

## Sources

### Primary (HIGH confidence — read from source or live-run this session)

- `web/node_modules/cytoscape/dist/cytoscape.esm.mjs` (3.34.2) — `notify()`, `endBatch()`, `destroyRenderer()`, `layoutPositions()`, `positions()`, `checkForInvalidEdgeWarning()` — full mechanism for FIX-04/05
- `web/node_modules/cytoscape-elk/src/layout.js` (2.3.0) — async `elk.layout().then()` chain, no-op `stop()`/`destroy()`
- `internal/daemon/watchdog.go`, `watchdog_posix.go`, `watchdog_test.go`, `daemon.go`, `daemon_test.go`, `soak_test.go`, `testbudget_test.go` — full current-state read for FIX-07/08
- `internal/cli/index.go`, `internal/graphstore/pebble_store.go`, `internal/graphstore/open_lock_test.go`, `internal/daemon/lock.go` — FIX-06
- `internal/bench/regression.go`, `internal/bench/metrics.go`, `tools/bench/runner/main.go` — FIX-09
- `.github/workflows/require-issue-link.yml`, `.github/workflows/pr-template-format.yml` — FIX-10
- Live command: `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` (2026-09-14, this session) — FIX-07/08 reproduction
- Live command: `GOTOOLCHAIN=go1.26.6 go test -race -run 'TestConvergenceTwoSessions|TestWatchdogCancelsOnReparent' -count=5 ./internal/daemon/` — FIX-08 isolated-pass confirmation
- `gh issue view 13/15/16/17/20 --json ...` — all five GH issues read in full including comments
- `.planning/WINDOWS.md` rows #12, #26, #28, #30, #36 — read in full
- `.planning/research/PITFALLS.md` Pitfalls 14–19 — read in full
- `.planning/milestones/v0.3.0-phases/04-supply-chain-coverage-daemon-substrate-fixes/04-02-SUMMARY.md` — the prior partial fix's own honest limitation record

### Secondary (MEDIUM confidence)

- `.planning/STATE.md` — decisions log, Phase 5/6 elkjs version and Svelte-double-effect precedents (cited, not independently re-verified this session)

### Tertiary (LOW confidence)

- `openssl` presence on GitHub's `ubuntu-latest` runner image (verified locally, not on the actual runner) — see Assumptions Log A1

## Metadata

**Confidence breakdown:**
- FIX-04/05 (graph console): HIGH for mechanism (read from vendored source, verbatim quotes with line numbers), MEDIUM for the exact fix shape (no live-Chromium reproduction was driven this session) and LOW-MEDIUM for FIX-05's specific overlapping pair (genuinely requires a live run against the pinned guava corpus, which is D-09's own job)
- FIX-06 (store lock): HIGH — full code, sentinels, and test-shape precedent read and cross-checked
- FIX-07/08 (watchdog/getppid): HIGH — live-reproduced this session, current code read, prior-fix history reconciled from its own SUMMARY
- FIX-09 (bench Repo): HIGH — all write sites read, call graph confirmed, no normalization risk found
- FIX-10 (heredoc delimiter): HIGH for the vulnerability and fix; MEDIUM for the test-harness convention recommendation (no existing precedent to copy exactly, extrapolated from `inject-cosign-key.sh`)
- FIX-11 (perf-gate follow-ups): HIGH — process-only, both GH issue text and existing rebless mechanism confirmed

**Research date:** 2026-09-14
**Valid until:** 14 days — this research is pinned to exact line numbers in vendored `node_modules` content and the current state of Go source files; any `pnpm install` that bumps `cytoscape`/`cytoscape-elk`, or any unrelated commit to `internal/daemon`/`internal/cli`/`internal/bench` before this phase executes, invalidates specific line citations (though the underlying mechanisms/decisions would likely still hold).
