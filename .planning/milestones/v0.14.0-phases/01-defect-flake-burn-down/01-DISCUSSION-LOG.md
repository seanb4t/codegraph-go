# Phase 1: Defect & Flake Burn-down - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-14
**Phase:** 1-Defect & Flake Burn-down
**Areas discussed:** Favicon mark & delivery (FIX-02), /graph console policy & gate (FIX-04/05), Lock hole & watchdog seam (FIX-06/07/08), Bench gate & CI closures (FIX-09/10/11)

---

## Favicon mark & delivery (FIX-02)

The user opened the discussion by asking to generate the mark with fal.ai rather than hand-draw it: "a mashup of CG, graph, code concepts". `fal-ai/recraft/v4.1/text-to-vector` was chosen because it emits real SVG (the favicon must be a static `'self'`-served file). Round 1 produced three concepts (node-jointed G; hex graph in angle brackets; blue tile with C/G rings + slash); round 2 iterated on the recommended direction.

| Option | Description | Selected |
|--------|-------------|----------|
| Iterate: CG ligature on tile | One more Recraft round: candidate 1's node-jointed letterform as a C+G ligature, bare and on candidate 3's blue tile | ✓ |
| Ship candidate 1 as-is | Node-jointed G, legible at 16 px; accept "G" as the mark | |
| Ship candidate 3 as-is | Blue tile with C/G rings; tile survives 16 px, inner detail does not | |
| Different concept entirely | Fresh prompt round | |

| Option | Description | Selected |
|--------|-------------|----------|
| Lock 5 (tile) as favicon, 4 as bare mark | Same CG ligature in both forms; tile ships as favicon, bare kept for future header use | ✓ |
| Lock 5 only | Favicon only; don't commit an unused bare variant | |
| One more round | Adjust stroke/node/radius/colour and iterate | |

| Option | Description | Selected |
|--------|-------------|----------|
| SVG + 32px PNG + apple-touch-icon | SVG for Chromium/Firefox, PNG fallback Safari uses, 180 px apple-touch-icon; all static under `web/static/` | ✓ |
| SVG only | One file; Safari shows a blank tab icon | |
| SVG + ICO | Classic favicon.ico plus SVG | |

**User's choice:** Iterate → lock round-2 candidates 5 (tile, favicon) and 4 (bare mark) → ship SVG + 32 px PNG + apple-touch-icon.
**Notes:** Six SVGs generated at $0.08 each (~$0.48). Candidates were compared at real 16 px (8× nearest-neighbour) as well as 512 px. Raw SVGs and the round-2 sheet are committed under `assets/`. Colours steered to the app's one chromatic token (Tailwind `blue-700` `#1d4ed8`) plus slate-900.

---

## /graph console policy & gate (FIX-04/05)

| Option | Description | Selected |
|--------|-------------|----------|
| Own code first; no dependency patch | Root-cause in GraphCanvas.svelte; isolate with cause on record if not interceptable; never a pnpm patch | ✓ |
| Own code first; pnpm patch allowed as fallback | Same, but a `pnpm patch` is an acceptable last resort | |
| Try a dependency upgrade first | Bump cytoscape / cytoscape-elk and re-test before touching our code | |

| Option | Description | Selected |
|--------|-------------|----------|
| Fix if it's a layout-option change; else waive on record | Spacing/padding fix ships; structural change is waived with the pair recorded | |
| Always fix | Whatever it takes to reach zero warnings at guava scale, including algorithm/sizing changes | ✓ |
| Root-cause and waive | Identify the pair, record why, close #28 record-only | |

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — clean console is the bar | Fix the `text-valign: right` stylesheet value; gate asserts zero errors and zero warnings | ✓ |
| No — errors only | Leave the warnings; gate asserts on pageerror only | |

| Option | Description | Selected |
|--------|-------------|----------|
| New check script + committed observation | `graph-console-check.mjs` in the style of the five existing live checks; verdict under `corpora/`; not in CI | ✓ |
| New check script, this-repo corpus in CI, guava manual | Same script; this-repo run wired into ci.yml | |
| Extend graph-measure.mjs | Add console assertions to the measurement harness | |

**User's choice:** Own-code fix, no dependency patch; always fix the guava overlap; warnings in scope; new check script with a committed observation.
**Notes:** "Always fix" was chosen over the recommended conditional rule; the visual-verification-on-both-corpora constraint (D-07) was recorded as the guard against a layout change regressing the small graph.

---

## Lock hole & watchdog seam (FIX-06/07/08)

| Option | Description | Selected |
|--------|-------------|----------|
| Refuse: hard error before RemoveAll | `ErrStoreLocked` → exit non-zero naming the holder, pointing at `daemon stop` / `unlock` | ✓ |
| Warn and proceed | Warn that the floor could not be read, then RemoveAll anyway | |
| Refuse unless a second flag is given | Hard error by default; `--ignore-lock` proceeds | |

| Option | Description | Selected |
|--------|-------------|----------|
| Warn and rebuild | Corrupt is what `--force` recovers from; warn, floor 0, proceed | ✓ |
| Refuse, same as locked | Any non-ErrNotFound error refuses | |
| Silent floor 0, unchanged | Keep today's tolerate-to-0 for corrupt; only locked changes | |

| Option | Description | Selected |
|--------|-------------|----------|
| Per-instance: field on Daemon, set at construction | `startWatchdog` takes the func; test-only `Option`; no package var remains | ✓ |
| Keep the package var, add explicit synchronization | Mutex / atomic.Pointer + warning comment | |
| Keep the var, serialize the tests | Package-level test lock, no t.Parallel | |

| Option | Description | Selected |
|--------|-------------|----------|
| Inject the ticker/clock alongside the seam | Per-instance option carries the interval / tick channel; assert on a signal not a deadline; confirm GH #13's leak | ✓ |
| Isolate the test in its own go test invocation | Pitfall 14's fallback; mechanism stays unexplained | |
| Investigate first via /gsd-debug, then choose | Confirm whether #13 and #17 share a root cause before choosing | |

**User's choice:** Refuse on live lock; warn-and-rebuild on corrupt; per-instance seam; injected ticker/clock.
**Notes:** All four recommended options accepted.

---

## Bench gate & CI closures (FIX-09/10/11)

| Option | Description | Selected |
|--------|-------------|----------|
| Strict equality, existing template | GOOS/Runner/ScratchFS shape; empty = never recorded; different seed/count is a different corpus | ✓ |
| Equality after normalization | Tolerate cosmetic renames | |

| Option | Description | Selected |
|--------|-------------|----------|
| Per-run random delimiter + shell test harness | `PRFILES_$(openssl rand -hex 16)` in both workflows; repo-local shell test feeds a `PRFILES_EOF` path through the block | ✓ |
| Drop the heredoc: write paths to a file, pass the path | Avoid multi-line GITHUB_OUTPUT entirely | |
| Per-run random delimiter, exercised by a real fork PR | Same fix; verified by a throwaway PR | |

| Option | Description | Selected |
|--------|-------------|----------|
| Close #1 on the record; run the #2 discriminator | Won't-do the cache-volume experiment; one run of the old-baseline commit on today's runner | ✓ |
| Run both experiments | Spend Namespace 8x16 sessions and the discriminator | |
| Close both on the record | No new measurements | |

**User's choice:** Strict `Repo` equality; per-run delimiter with a shell test harness; close #20-1 on the record and run the #20-2 discriminator.
**Notes:** Scouting established that `Metrics.Repo` is already `synthetic-seed{N}-count{M}` at the regression-mode write site, which removed Pitfall 16's normalisation question from the table.

---

## Claude's Discretion

- FIX-06 refuse-message and corrupt-warning wording.
- Committed vs build-rendered PNG/apple-touch-icon, as long as `web:build:verify` / `web:drift` cannot pass with the Svelte logo present.
- "Ignore stale run" vs "cancel on destroy" for FIX-04 once root-caused live.
- The specific ELK spacing/sizing expression for FIX-05, under D-07's visual-verification constraint.
- Injected-ticker shape (interval override vs tick channel).
- Name/location of the D-17 shell test harness.

## Deferred Ideas

- Perf-gate baseline staleness check (GH #20 follow-up 2's closing observation) — new gate capability, backlog.
- In-app header placement for the bare mark — a UI change outside a burn-down phase.
