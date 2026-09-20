---
phase: 01-defect-flake-burn-down
plan: 09
subsystem: ui
tags: [svelte, cytoscape, cytoscape-elk, playwright, graph-console-check]

# Dependency graph
requires:
  - phase: 01-01
    provides: "web/scripts/graph-console-check.mjs — the RED verdict this plan's Task 2 replaces, and the window.__codegraphFileGraphCy debug seam used to diagnose the pair"
  - phase: 01-08
    provides: "The FIX-04 deferred-teardown guard (layoutGeneration/layoutInFlight/onLayoutSettled) this plan's render-flow fix must coexist with, never revert"
  - phase: 01-06
    provides: "web/build/ rebuilt with the favicon fix — this plan rebuilds it again on top"
provides:
  - "The guava invalid-endpoints warnings are root-caused as a first-render-tick race in GraphCanvas.svelte's OWN mount flow (cytoscape's own implicit default 'grid' layout running before this component's effect ever gets control), not a layout-option lever, and closed at that root cause"
  - "A committed, schema-versioned, two-corpus green /graph console verdict with its RED predecessor one commit behind it, closing the phase's clean-console requirement (FIX-05)"
affects: []

# Actuals (#2632)
actuals:
  tokens: 4500
  tasks: 2
  commits: 3
  plan_head_before: 50f6a5acfafc53d7c2ab917832642d09bf6948e2

tech-stack:
  added: []
  patterns:
    - "cytoscape's Core constructor runs an IMPLICIT default layout ('grid' when a container is present) before any caller code gets control if no `layout` option is passed to the constructor — anything that must control the FIRST paint must pass `layout: { name: 'null' }` explicitly, not rely on gating logic that only runs after construction returns."
    - "cytoscape's own `display: 'none'` style bypass (not CSS visibility on the container) is the correct way to keep specific elements out of `checkForInvalidEdgeWarning`'s per-edge loop — `takesUpSpace()` is real-time-evaluated per edge, not cached across a bypass application, and NullLayout's own 'every node at (0,0)' behavior separately suppresses the warning via `nodesOverlap` for the whole graph during any window before real positions exist."

key-files:
  created: []
  modified:
    - web/src/lib/components/graph/GraphCanvas.svelte
    - corpora/graph-console-check.json
    - web/build/** (rebuilt output, committed per web:drift's contract, across both task commits)

key-decisions:
  - "The mechanism is a first-render-tick race in cytoscape's OWN construction, not a layout-option lever: with no `layout` option passed to the Core constructor, cytoscape defaults to `{name: 'grid'}` and runs that layout SYNCHRONOUSLY during construction — entirely before createFileGraphRenderer's start() (this component's own ELK-layout entry point) ever gets control. For the guava android/guava-tests<->android/guava util/concurrent pair specifically, that grid layout's own construction-time getFitViewport()/boundingBox() call computes a genuinely degenerate edge-endpoint geometry, tripping cytoscape's own checkForInvalidEdgeWarning. This was found live by instrumenting checkForInvalidEdgeWarning's actual call site and correlating its timestamp against this component's own code, not by inspecting settled state after the fact — an initial hypothesis (hiding edges only inside start()/runLayout()) was tested, measured to still warn, and was traced to firing BEFORE start() ever runs, which is what led to the actual root cause."
  - "Fix: `layout: { name: 'null' }` on the Core constructor call replaces the implicit default. cytoscape's own NullLayout (cytoscape.esm.mjs, read this task) puts every node at the exact same (0, 0) point rather than computing distinct grid positions, so every edge's `nodesOverlap` is true and `checkForInvalidEdgeWarning`'s own overlap short-circuit suppresses the warning outright — cytoscape's own sanctioned no-op state for 'no layout has run yet,' not a workaround or a waiver."
  - "Kept the originally-planned edge-hide-until-first-layoutstop mechanism (cytoscape's `display` bypass on `opts.cy.edges()`, set before start()'s own ELK `.run()` call, removed exactly once inside the SAME generation-checked `layoutstop` callback FIX-04's deferred-teardown guard already gates on) as defense in depth for the same class of race during ELK's own async completion, even though empirical measurement showed the two OBSERVED warnings were both attributable to the construction-time default layout, not to ELK's own completion. Kept because it is cheap, harmless (a mid-layout teardown's deferred `cy.destroy()` still sees the identical settle callback, so it cannot desync from FIX-04's own guard), and matches the plan's authorized fix surface exactly."
  - "`nodeDimensionsIncludeLabels: true` (already in the working tree from the prior blocked investigation) is kept on its own merits — ELK should see these nodes' true rendered label size — but is explicitly NOT presented as the fix, per the plan's own instruction."

requirements-completed: [FIX-04, FIX-05]

coverage:
  - id: D1
    description: "The guava invalid-endpoints warnings are root-caused (named mechanism, named diagnostic entry) and fixed at that cause via a render-flow change in GraphCanvas.svelte, proven load-bearing by an uncommitted RED (fix removed)/GREEN (fix restored) two-corpus toggle"
    requirement: FIX-05
    verification:
      - kind: e2e
        ref: "node web/scripts/graph-console-check.mjs --out <scratch>, run with `layout: { name: 'null' }` removed (RED: guava consoleWarn=4, invalidEndpointDiagnostics=2, self stays clean) and restored (GREEN: both corpora consoleWarn=0/consoleError=0/pageErrors=0), repeated 3x GREEN with zero flake"
        status: pass
      - kind: other
        ref: "live instrumentation of cytoscape's own checkForInvalidEdgeWarning call site (temporary, removed before commit) correlating warning timestamps against this component's own hide/reveal code, isolating the construction-time default-grid-layout mechanism"
        status: pass
    human_judgment: false
  - id: D2
    description: "The render-flow change coexists with 01-08's FIX-04 deferred-teardown guard: the edge-hide/reveal shares the exact generation token runLayout()'s layoutstop callback already checks, and a navigation away that interrupts an in-flight ELK layout (01-08's own reproduction sequence, re-run against this changed code) produces zero page errors and zero console warn/error entries"
    requirement: FIX-05
    verification:
      - kind: e2e
        ref: "scratch Playwright driver (not committed): 3x CPU-throttled (20x, CDP Emulation.setCPUThrottlingRate) navigate-away-mid-layout cycles against the guava corpus — 0 page errors, 0 console warn/error entries across all 3"
        status: pass
      - kind: other
        ref: "rg -c 'layoutInFlight' GraphCanvas.svelte — 6+ references at HEAD, confirming the FIX-04 guard is intact and unweakened"
        status: pass
    human_judgment: false
  - id: D3
    description: "The clean-console bar is met on both corpora (zero page errors, zero console warn/error), the check script's allowlist stays empty, and no verdict records a non-zero allowlisted count"
    requirement: FIX-05
    verification:
      - kind: e2e
        ref: "node web/scripts/graph-console-check.mjs --out corpora/graph-console-check.json (via task check:graph-console): self and guava both pageErrors=0 consoleWarn=0 consoleError=0; jq assertion on allowlistedCount"
        status: pass
      - kind: unit
        ref: "pnpm check (svelte-check): 0 errors, 0 warnings across 1172 files; task web:drift: source and output halves both MATCH"
        status: pass
    human_judgment: false
  - id: D4
    description: "The committed verdict replaces the RED one plan 01-01 committed, with schemaVersion, the pinned guava sha, a browser version, a non-zero guava node count, and an immediately-preceding commit recording success=false"
    requirement: FIX-05
    verification:
      - kind: e2e
        ref: "jq assertions against corpora/graph-console-check.json (success=true, 2 runs, browserIdentity.version present, guava nodeCount>0, sha 94f39958... present, schemaVersion present) and against the immediately-preceding commit 2c69b3ba (success=false)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live visual verification in Chromium on both this repository's own graph and the pinned guava corpus — legibility, no unlaid-out flash on first paint, expand/collapse, and an interrupted layout — because a construction-time layout change reshapes every graph, not only the pair it targeted"
    requirement: FIX-05
    verification: []
    human_judgment: true
    rationale: "This is the plan's own <human-check> gate (01-VALIDATION.md's Manual-Only Verifications: no automated check can judge a layout's readability). Per this plan's resume_instructions, it is harvested at end-of-phase human review rather than blocking this executor session — HUMAN_VERIFY_MODE is end-of-phase and auto-mode is active, so this task's automated <verify> gates were re-run and passed (see D1-D4) and execution continued without stopping for the human-check, exactly as the tracer/human-verify gate protocol specifies for a non-tracer auto task under this instruction. AWAITING PHASE-LEVEL HUMAN REVIEW: nobody has yet visually confirmed in a real browser that (a) this repository's own /graph and (b) the guava /graph both show a legible arrangement with no visible flash of an unlaid-out graph before the ELK arrangement appears, and that expand/collapse plus an interrupted layout behave correctly on both. If review finds a degradation, the plan's own remedy is a smaller/differently-targeted change (e.g. a minimum size on degenerate nodes) — never an allowlist entry (D-07 forecloses that)."
  - id: D6
    description: "Verdict diff mapping: each entry class that disappeared between the 01-01 RED verdict and this GREEN one is attributed to the specific plan/task that closed it"
    requirement: FIX-05
    verification:
      - kind: other
        ref: "jq diff of corpora/graph-console-check.json at commit 2c69b3ba (RED) vs HEAD (GREEN): text-valign warnings -> 01-08 Task 1; null-renderer TypeError -> 01-08 Task 3; guava invalid-endpoints warnings -> 01-09 Task 1 (this plan); favicon/CSP image warning -> 01-06"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 9: Root-Caused and Fixed the Guava Invalid-Endpoints Race Summary

**Root-caused the guava invalid-endpoints warnings to cytoscape's own implicit default construction-time layout (not an ELK layout-option lever, which had been exhaustively falsified in a prior blocked session), fixed it with `layout: { name: 'null' }` on the Core constructor plus an edge-hide/reveal defense-in-depth around the real ELK layout, and committed the phase's green two-corpus /graph console verdict.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-09-15T18:20:00Z (approx.)
- **Completed:** 2026-09-15T18:58:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 hand-authored (`GraphCanvas.svelte`, `corpora/graph-console-check.json`) + 15 regenerated `web/build/**` output files, across 2 task commits (plus the orchestrator's own Task-1-amendment commit from the earlier blocking-human checkpoint)

## Accomplishments

- **Root-caused the actual mechanism live**, superseding the prior blocked session's falsified LAYOUT_OPTIONS investigation entirely: cytoscape's own `Core` constructor runs an *implicit* default `'grid'` layout synchronously during construction whenever no `layout` option is passed — entirely before `createFileGraphRenderer`'s `start()` (this component's own ELK-layout entry point) ever gets control. For the guava `android/guava-tests/.../util/concurrent` <-> `android/guava/src/.../util/concurrent` pair, that grid layout's own construction-time `getFitViewport()`/`boundingBox()` call computes a genuinely degenerate edge-endpoint geometry, tripping cytoscape's own `checkForInvalidEdgeWarning`. Found by instrumenting the actual `checkForInvalidEdgeWarning` call site live and correlating its firing timestamp against this component's own code — not by inspecting settled state after the fact. An initial hypothesis (hide edges only inside `start()`/`runLayout()`) was implemented, measured to still warn identically, and its own timing data is what pointed at the true, earlier cause.
- Fixed by passing `layout: { name: 'null' }` to the `cytoscapeLib({...})` constructor call: cytoscape's own `NullLayout` puts every node at the exact same `(0, 0)` point instead of computing distinct grid positions, so every edge's `nodesOverlap` is `true` and `checkForInvalidEdgeWarning`'s own overlap short-circuit suppresses the warning — cytoscape's own sanctioned no-op state for "no layout has run yet," not a waiver.
- Kept the plan's originally-authorized edge-hide/reveal mechanism (`display` style bypass on every edge, set before `start()`'s own ELK `.run()`, removed exactly once inside the SAME generation-checked `layoutstop` callback FIX-04's deferred-teardown guard already gates on) as defense in depth for the same race class during ELK's own async completion — verified it shares FIX-04's generation token exactly, so it cannot desync from the teardown guard.
- Proved the fix load-bearing with an uncommitted RED/GREEN toggle: removing the `layout: { name: 'null' }` line reproduces the exact guava warnings (4 console.warn entries, 2 unique `invalidEndpointDiagnostics`); restoring it returns both corpora to zero warnings/errors, confirmed across repeated runs.
- Re-ran 01-08's own interrupting sequence (CPU-throttled navigate-away mid-layout, 3x against the changed layout) — zero page errors, zero console warn/error entries, confirming the render-flow change does not widen or otherwise interact badly with the FIX-04 teardown window.
- Committed the phase's green, schema-versioned, two-corpus `/graph` console verdict via `task check:graph-console`, with its RED predecessor (`2c69b3ba`, plan 01-01) exactly one commit behind it, and mapped every entry class that disappeared to the specific plan/task that closed it.

## Task Commits

1. **Task 1: Close the first-render-tick invalid-endpoints race with a render-flow change** - `676b2540` (fix)
2. **Task 2: Commit the green two-corpus console verdict** - `c74770cf` (docs)

_Preceded in this plan's history by the orchestrator's own amendment commit `31b30012` (docs), which rewrote Task 1's frontmatter/action/verify/acceptance-criteria at the blocking-human checkpoint before this executor session began._

## Files Created/Modified

- `web/src/lib/components/graph/GraphCanvas.svelte` — Core constructor now passes `layout: { name: 'null' }`; `start()` hides every edge via cytoscape's `display` bypass before running the real ELK layout; the shared `layoutstop` callback reveals them exactly once, gated by the same generation token FIX-04 already checks.
- `corpora/graph-console-check.json` — replaced the RED verdict plan 01-01 committed with the green two-corpus verdict (`success: true`).
- `web/build/**` — rebuilt output committed per `task web:drift`'s contract.

## Decisions Made

See `key-decisions` in the frontmatter above — summarized: the mechanism is cytoscape's own implicit default construction-time layout, not an ELK option; the fix is `layout: { name: 'null' }` on the constructor; the edge-hide/reveal mechanism is kept as defense in depth, not because it was the actual fix; `nodeDimensionsIncludeLabels: true` is kept on its own merits, not presented as the fix.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] `task build:release` failed against the ambient Go toolchain**
- **Found during:** Task 1's `<verify>` (first attempt)
- **Issue:** The ambient `go` binary is 1.27.1, newer than `go.mod`'s declared `go 1.26.6`, with no `toolchain` directive to force a download. `github.com/cockroachdb/swiss`'s `//go:linkname` references to Go's internal runtime symbols (`hashFn`, `getRuntimeHasher`, `fastrand64`) are broken under 1.27.1, failing the build with `undefined: ...` link errors — unrelated to any code in this plan (zero Go files touched).
- **Fix:** Prefixed every `task build:release` invocation with `GOTOOLCHAIN=go1.26.6`, exactly matching the phase-level gate's own already-established convention (`GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...`, verbatim from this plan's own `<verification>` section) — not a new pattern, just applying the existing one consistently to the local build step too.
- **Files modified:** None (invocation-only; no source or config change).
- **Verification:** `GOTOOLCHAIN=go1.26.6 task build:release` succeeds cleanly every time it was run this session.
- **Committed in:** N/A (not a source change).

---

**Total deviations:** 1 auto-fixed (1 Rule 3 — build-invocation blocker, pre-existing environment mismatch, not a code change).
**Impact on plan:** None on scope or correctness — a local toolchain-selection workaround, not a code or dependency change. D-06 (no dependency patch, no version bump) is untouched.

## Issues Encountered

- **`task build:release`'s Go toolchain mismatch** — see Deviations above. Resolved by invocation prefix, not a code change.
- **The phase-level gate (`GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...`) surfaced two FAILING packages, both pre-existing, documented flakes entirely unrelated to this plan's diff (which touches zero Go files):**
  - `internal/daemon`: `daemon: sync lost the store-lock race 6 consecutive times; giving up` — a store-lock contention flake under local machine load. Matches the class of flake `.planning/STATE.md`'s Blockers/Concerns already records ("unrelated pre-existing flake `TestDaemonFlushLockRequeueGivesUpPerEpisode` surfaced under this session's high local machine load during full-suite -race verification — logged to deferred-items.md, not fixed (out of scope; D-14 forbids widening any `internal/daemon` timeout constant)").
  - `internal/mcp`: `TestSessionLineReflectsPostAppearanceToolCount` failed on an out-of-order `notifications/tools/list_changed` arriving where a successful `initialize` response was expected — matches the documented "Wire oracle `toolslist-repeat` response ordering flake" (`.planning/STATE.md` Pending Todos: "id-2 response overtaken by id-3 under parallel load on Linux; latent on main, re-run of the identical commit passed").
  - Neither failure is touched by, or attributable to, this plan's Svelte/UI-only diff. Per the deviation rules' scope boundary ("do not auto-fix pre-existing issues unrelated to current task"), these are recorded here for visibility and left exactly as `.planning/STATE.md` already has them — not fixed, not newly filed. Re-running the full suite at a quieter moment (or `-race` in isolation, one package at a time) is the established diagnostic for both, per the existing STATE.md record.
- **The literal `<verify>` command text for Task 1's third gate (`! rg -q 'pattern:' web/scripts/graph-console-check.mjs`) does not evaluate cleanly** — `graph-console-check.mjs` was authored by plan 01-01 and has, since that plan, carried a documentation comment describing the allowlist entry *format* (`// \`{ pattern: '<regex source>', reason: '...' }\``) directly above the `const ALLOWLIST = [];` declaration. That comment's own literal text matches `pattern:`, so the gate's `rg` clause fires regardless of whether an allowlist entry was ever added — a pre-existing false-positive in the gate's own construction, present since 01-01 and untouched by this plan (`git diff` on that file for this plan's commits is empty). The gate's actual intent — "no allowlist entry was added" — is independently and unambiguously confirmed: `const ALLOWLIST = [];` remains a literal empty array, and this plan's `git diff` on `web/scripts/graph-console-check.mjs` is empty (the file was never touched, since it is outside Task 1's authorized `<files>` scope).

## User Setup Required

None.

## Threat Flags

None — no new security-relevant surface introduced. The render-flow change is confined to `GraphCanvas.svelte`'s existing mount/layout lifecycle; no new endpoint, auth path, file access, or schema change.

## Next Phase Readiness

This is the last plan in Phase 01 (Defect & Flake Burn-down). All 9 plans have SUMMARY files. Before closing the phase:

1. **Human review required** (D5 above, awaiting phase-level human review): visually confirm in Chromium, on both this repository's own graph and the pinned guava corpus, that (a) the first paint shows the settled ELK arrangement with no visible flash of an unlaid-out graph, (b) the arrangement stays legible (nodes not flung into unreadable sparseness, edges traceable, initial fit shows the whole graph), (c) expand/collapse of at least one directory node works correctly on both, and (d) navigating away from `/graph` while a layout is still computing on guava, then back, behaves correctly. If a degradation is found, the remedy is a smaller/differently-targeted layout option change — never an allowlist entry (D-07).
2. The two pre-existing load-induced flakes noted under Issues Encountered (`internal/daemon` store-lock contention, `internal/mcp` notification-ordering) are already tracked in `.planning/STATE.md` and are not new to this plan — no new action needed here, but they remain visible risk for a full `-race` CI run under load.
3. Ready for `/gsd-verify-work 01` and the phase-level gate checklist in this plan's own `<verification>` section (GH #13/#17 disposition confirmation, `task check:gonum`/`check:no-force-layout` CI wiring follow-up, etc. — all pre-existing, tracked items, not new to this plan).

## Self-Check: PASSED
