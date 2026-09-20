# Phase 1: Defect & Flake Burn-down - Context

**Gathered:** 2026-09-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Nine ledger items closed at their cause, never masked: the stock-Svelte favicon that also violates CSP (FIX-02, WINDOWS #30); the `/graph` console entries — the cytoscape `notify` null TypeError, the guava-scale "invalid endpoints" warnings and the two `text-valign: right` style warnings (FIX-04/05, WINDOWS #26/#28); the `codegraph index --force` store-lock hole (FIX-06, WINDOWS #36 / T-10-16); the load-sensitive watchdog test and the `getppid` seam data race (FIX-07/08, WINDOWS #12, GH #17, GH #13); and three bench/CI hygiene defects — `CheckRegression` never comparing `Metrics.Repo` (FIX-09, GH #16), the fixed heredoc delimiter over fork-controlled paths in two `pull_request_target` workflows (FIX-10, GH #15), and GH #20's two perf-gate follow-ups (FIX-11).

Touches `web/`, `internal/cli/index.go`, `internal/daemon`, `internal/bench`, `tools/bench/`, and the two `pull_request_target` workflows only. `FIX-03` (picker footer) is deliberately Phase 7. Nothing here widens a CSP directive, a timeout, or a retry policy.

</domain>

<decisions>
## Implementation Decisions

### Favicon mark & delivery (FIX-02)
- **D-01:** The codegraph mark is the fal.ai-generated **octagonal node-jointed "CG" ligature**: every stroke a straight segment, a solid circular node at every corner and terminal, C and G sharing one vertical edge. Generated with `fal-ai/recraft/v4.1/text-to-vector` over two rounds (six candidates); round-2 candidates 4 and 5 were locked. Raw SVGs are committed at `.planning/phases/01-defect-flake-burn-down/assets/01-mark-favicon-tile.raw.svg` (white ligature on a `blue-700` `#1d4ed8` rounded-square tile) and `01-mark-bare.raw.svg` (blue nodes on slate `#0f172a` edges, no tile). — **Reversibility:** costly — once a release ships it, the tile is the icon users recognise in their tab bar; replacing it later is identity churn, not a migration.
- **D-02:** The **tile variant is the favicon**; the **bare variant is kept alongside** as the mark for any future in-app header use (no header placement this phase). Both are the same letterform — one identity.
- **D-03:** The raw Recraft output is **not** the shipped file. Clean-up before shipping: strip the C2PA `<metadata>` block (~20 KB), drop `preserveAspectRatio="none"`, set a square `viewBox`, remove `style="display: block;"` and any white background rect, keep only the vector paths. Tile corner radius and colours stay as generated.
- **D-04:** Formats shipped: **SVG + 32×32 PNG + 180×180 apple-touch-icon**, all as **static files under `web/static/`**, all served as `'self'`. `+layout.svelte` links all three (`<link rel="icon" type="image/svg+xml">`, `<link rel="icon" sizes="32x32" type="image/png">`, `<link rel="apple-touch-icon">`). The `import favicon from '$lib/assets/favicon.svg'` path — the thing Vite inlines as a `data:` URI — goes away; `web/src/lib/assets/favicon.svg` (the Svelte logo) is deleted.
- **D-05:** `internal/uiserver/spa.go`'s `spaCSPBaseDirectives` and `spa_test.go`'s `default-src` assertion are **untouched** (WINDOWS #30: "the asset choice is the defect"). No `img-src`, no `data:`.

### /graph console policy & gate (FIX-04/05)
- **D-06:** FIX-04 (`Cannot read properties of null (reading 'notify')`) is fixed **in our own code, never via a dependency patch**. Leading hypothesis for the planner to confirm: cytoscape core's `endBatch()` calls `renderer.notify(...)` without the `destroyed() || !renderer` guard that `notify()` itself has, and cytoscape-elk's async ELK promise resolves `layoutPositions` against an instance that has been destroyed and re-created (mount/effect double-run) — `GraphCanvas.svelte` opens batches at `:387, :459, :492, :555` and `cy.destroy()` at `:898`. Fix the lifecycle/batch discipline on our side (ignore or cancel a stale layout run once the instance is torn down). If it genuinely cannot be intercepted from our side, **isolate with the root cause recorded** — no `pnpm patch`, no `patches/` directory. Dependency upgrade is not the first move.
- **D-07:** FIX-05 (guava "invalid endpoints"): root-cause which collapsed pair overlaps under ELK at that density, then **always fix** — including a layout-option, sizing-model or algorithm change if that is what closes it. **Constraint:** any layout change is visually verified live in Chromium on **both** this repo's graph and the pinned guava corpus before it lands, because it reshapes every graph.
- **D-08:** The two `text-valign: right` warnings are **in scope** — `right` is a `text-halign` value; fix the stylesheet. The bar is a **clean console**: zero uncaught page errors **and** zero `console.warn`/`console.error` originating from our code, on both corpora.
- **D-09:** The gate is a **new `web/scripts/graph-console-check.mjs`** in the style of the five existing live checks (`graph-expand-check.mjs`, `breadcrumb-check.mjs`, …): boots the real binary, loads `/graph` on this repo's index and on the pinned guava corpus, captures `page.on('pageerror')` and console messages, fails on any. Verdict **committed under `corpora/`** per the v0.13.0 GRF-09 `graph-cluster-observations.json` precedent, with a Taskfile target. **Not wired into CI** — guava is too heavy for every run.

### Store-lock hole (FIX-06)
- **D-10:** `codegraph index --force` against a store held by a live process **refuses**: `priorCoverageGeneration` (or its replacement) surfaces `errors.Is(err, graphstore.ErrStoreLocked)` and the command exits non-zero **before `os.RemoveAll`**, naming the holder and pointing at `codegraph daemon stop` / `codegraph unlock` (`daemon unlock` after Phase 3). A user who wants to proceed stops the holder first. No new flag.
- **D-11:** A **corrupt/unreadable** store (Open or `GetMeta` fails with something other than `ErrStoreLocked` / `ErrNotFound`) **warns and rebuilds**: print that the prior coverage generation could not be read so the floor is 0, then proceed — `--force` is the recovery path for exactly this case. `ErrNotFound` (never indexed / no Meta yet) stays silent at floor 0.
- **D-12:** Regression test is the **hold-the-lock-across-`index --force`** shape named in WINDOWS #36, modelled on `internal/graphstore/open_lock_test.go`, and is **watched fail against the pre-fix build** (RED transcript in the plan's verify gates, per this repo's guard discipline and rule `84d1gfpywd`).

### Watchdog seam & flake (FIX-07/08)
- **D-13:** The `getppid` seam becomes **per-instance**: `startWatchdog` takes the `func() int` as a parameter; `Daemon` carries it as an unexported field set by a **test-only `Option`** (mirrors `onSync` / `onSyncStart` / `syncFn`). `parentChanged` takes the func (or becomes a method). **No package-level `var getppid` remains** — the FIX-08 race is structurally impossible, not serialised by test discipline. — **Reversibility:** costly — the signature change touches `watchdog.go`, `watchdog_posix.go`, `watchdog_windows.go`, `daemon.go` and both tests; going back means re-introducing the global.
- **D-14:** FIX-07 is fixed by **injecting the ticker/clock alongside the seam**: the same per-instance option carries the poll interval and/or a tick channel so the test drives "next poll" deterministically instead of racing a real 1 s ticker under load. `TestRunWatchdogCancelsRunOnSimulatedReparent` asserts on a signal, not a deadline. **No timeout constant changes anywhere** (Pitfall 14; the fix comment cites WINDOWS #12's load-sensitivity finding). The `testBudget` scaling helper stays as-is.
- **D-15:** GH #13's second half — the watchdog goroutine leaked from `TestConvergenceTwoSessions` (`soak_test.go:181` → `RunWithRetry` → `Run` → `startWatchdog`) — is **confirmed or refuted in this phase**, not assumed. If real, the leak is closed (the test joins its daemon before returning) and `go test -race ./internal/daemon/...` is the gate for both halves.

### Bench gate & CI closures (FIX-09/10/11)
- **D-16:** `CheckRegression` gains a **strict-equality `Repo` guard on the existing GOOS/Runner/ScratchFS template**: a non-empty-vs-non-empty mismatch refuses with a message pointing at the rebless mechanism (`tools/bench/BASELINE.md`); empty on either side means "never recorded", never wildcard. No normalisation — `Repo` is already the synthesized corpus id `synthetic-seed{N}-count{M}` at the regression-mode write site (`tools/bench/runner/main.go:691`), so a different seed/count **is** a different corpus and the only sanctioned "rename" is a rebless. `TestCheckRegression` gets the mismatch subtest.
- **D-17:** FIX-10 uses a **per-run random delimiter** — `DELIM="PRFILES_$(openssl rand -hex 16)"` — in **both** `.github/workflows/require-issue-link.yml` and `.github/workflows/pr-template-format.yml` (GitHub's own guidance for multi-line outputs from untrusted content; Pitfall 17). It is **exercised by a repo-local shell test** that feeds a file list containing a literal `PRFILES_EOF` path through the same shell block and asserts the output parses intact — a repeatable gate runnable in CI, not a throwaway fork PR.
- **D-18:** GH #20 follow-up 1 (Namespace cache volume on 8×16) is **closed on the record as won't-do**: `ubuntu-latest` is free and 28× more stable, the adoption bar was "match it", and storage class cannot reach past the host-placement variance the issue leaves unrefuted. Follow-up 2 (+44.8% baseline drift) gets **one manually-dispatched discriminator run**: the old-baseline commit on today's runner, same seed/count, recorded in `tools/bench/BASELINE.md`, attributing the drift to fleet vs code. Both decisions are recorded on GH #20 and it is closed.

### Claude's Discretion
- Exact wording of the FIX-06 refuse message and the corrupt-store warning.
- Whether the 32 px PNG and apple-touch-icon are committed files or rendered from the SVG by a build step — either is acceptable as long as `web:build:verify` / `web:drift` cannot pass with the Svelte logo still present.
- Which of "ignore stale run" vs "cancel on destroy" closes FIX-04, once the root cause is confirmed live.
- How the ELK spacing/sizing change for FIX-05 is expressed, subject to D-07's visual-verification constraint.
- Shape of the injected ticker (interval override vs explicit tick channel) — whichever lets the test assert on a signal without a real timer.
- Name and location of the shell test harness for D-17.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Ledger & research
- `.planning/WINDOWS.md` — rows #12, #26, #28, #30, #36 (the five windows this phase closes; each carries the recorded fix path)
- `.planning/research/PITFALLS.md` §Pitfall 14–18 — watchdog flake, `getppid` seam, `Metrics.Repo`, heredoc delimiter, favicon/CSP
- `.planning/REQUIREMENTS.md` — FIX-02, FIX-04…FIX-11 definitions

### GitHub issues (read via `gh issue view N`)
- GH #13 — `getppid` data race + leaked watchdog goroutine mechanism (FIX-08, D-15)
- GH #15 — heredoc delimiter over fork-controlled paths; both workflows named (FIX-10)
- GH #16 — `CheckRegression` never compares `Metrics.Repo` (FIX-09)
- GH #17 — `TestRunWatchdogCancelsRunOnSimulatedReparent` 3/3 fail under load, 3/3 pass isolated (FIX-07)
- GH #20 — perf-gate follow-ups with raw data and the discriminator described (FIX-11)

### Mark assets (D-01)
- `.planning/phases/01-defect-flake-burn-down/assets/01-mark-favicon-tile.raw.svg` — locked favicon source (tile)
- `.planning/phases/01-defect-flake-burn-down/assets/01-mark-bare.raw.svg` — locked bare mark source
- `.planning/phases/01-defect-flake-burn-down/assets/01-mark-round2-sheet.png` — the comparison sheet the choice was made from

### Bench & precedent
- `tools/bench/BASELINE.md` — rebless mechanism and the incident history the existing three guards cite
- `corpora/graph-cluster-observations.json` — the committed-observation precedent D-09 follows (v0.13.0 GRF-09)
- `internal/graphstore/open_lock_test.go` — the lock-collision test shape D-12 models

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/scripts/{graph-expand-check,breadcrumb-check,graph-live-update-check,graph-collapse-affordance-check,live-push-multitab-check}.mjs` — five live-Chromium checks that already boot the real binary and capture `page.on('pageerror')`; `graph-console-check.mjs` copies their harness shape.
- `internal/daemon` test-seam convention (`onSync`, `onSyncStart`, `syncFn`, `onWatchOpen` — unexported fields, no exported setter, set via the variadic `Option`) — D-13/D-14 add `getppid` and the ticker the same way.
- `internal/daemon/testbudget_test.go` — `testBudget()` and `joinDaemonRun()` (MAINT-01/02) already give the tests a single scaling knob and a Goexit-safe join; the FIX-07 fix uses them rather than adding timeouts.
- `internal/bench/regression.go` — the GOOS/GOARCH, Runner, ScratchFS guards (`:56`, `:79`, `:99`) and their `runnerString`/`scratchFSString` message helpers are the template for the `Repo` guard.
- `graphstore.ErrStoreLocked` (`pebble_store.go:110`) and `ErrNotFound` (`pebble_store.go:21`) — the sentinels `priorCoverageGeneration` must distinguish; `graphstore.Open` already retries ~400 ms before returning `ErrStoreLocked`.

### Established Patterns
- **Guards must fail RED before they pass** (rule `84d1gfpywd`; every v0.13.0 phase committed mutation transcripts) — D-12's lock test and D-17's shell harness are both watched fail first.
- **Live Chromium is the gate for `/graph`** (v0.12.0 lesson: jsdom + `headless:true` never exercised the renderer path that produces WINDOWS #26).
- **Committed observations under `corpora/`** for measurements too heavy for CI (GRF-01, GRF-09).
- **`spa_test.go:439`'s exact `default-src 'self'` assertion is deliberate** (T-02-02-06) — the favicon fix must leave it byte-identical.
- CI already isolates `internal/daemon` (`ci.yml:221` `-count=1`; `:241` `-race -p 1`), which is why GH #17's full-suite failure is invisible there — the fix must make the test pass *inside* `go test ./...` locally, not rely on the CI split.

### Integration Points
- `web/src/routes/+layout.svelte:7` / `:91` — the `$lib/assets/favicon.svg` import and `<link rel="icon">` that change to static `/favicon.svg` + PNG + apple-touch-icon.
- `web/static/` — currently `robots.txt` only; the three icon files land here.
- `internal/cli/index.go:101–107` — `priorCoverageGeneration(storeDir)` followed by `os.RemoveAll(storeDir)`; the refuse/warn branch sits between them.
- `internal/daemon/watchdog.go:21` (`var getppid`), `:36` (`startWatchdog`), `watchdog_posix.go:14` (`parentChanged`), `daemon.go:262` (the call site inside `Run`) — the seam refactor.
- `web/src/lib/components/graph/GraphCanvas.svelte` — batch sites `:387/:459/:492/:555`, `LAYOUT_OPTIONS :139`, `layout().run() :374`, `destroy() :898`.
- `.github/workflows/require-issue-link.yml:44–50` and `pr-template-format.yml:41–46` — the two heredoc blocks.

</code_context>

<specifics>
## Specific Ideas

- The mark was chosen by looking at candidates rendered at **real 16 px** (8× nearest-neighbour) — the planner should carry that check forward: the shipped SVG is verified at 16 px and 32 px, not only at full size.
- Colours were chosen to match the app: Tailwind `blue-700` `#1d4ed8` is the one chromatic value in `web/src/app.css` (`oklch(0.488 0.243 264.376)`); the bare mark uses slate-900 `#0f172a` for edges.
- For FIX-06 the message should read like `acquire()`'s existing `ErrLockLive` text (`lock.go:208`) — "pid=N — stop it first, or run …" — so the two lock-refusal messages the user can hit feel like one tool.

</specifics>

<deferred>
## Deferred Ideas

- **Perf-gate baseline staleness check** — GH #20 follow-up 2 observes that "if GitHub's fleet can shift ~45% in two days, any baseline goes stale fast and the gate needs a staleness check, not just a one-time refresh." That is a new gate capability, not a defect; backlog candidate for a later milestone.
- **In-app header placement for the bare mark** — D-02 keeps the bare variant but no UI uses it yet; a header/brand row is a UI change outside a burn-down phase.

</deferred>

---

*Phase: 01-defect-flake-burn-down*
*Context gathered: 2026-09-14*
