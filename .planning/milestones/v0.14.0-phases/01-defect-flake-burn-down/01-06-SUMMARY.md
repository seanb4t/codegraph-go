---
phase: 01-defect-flake-burn-down
plan: 06
subsystem: ui
tags: [svelte, sveltekit, svg, favicon, csp, static-assets, uiserver]

# Dependency graph
requires:
  - phase: 01-01
    provides: "web/scripts/graph-console-check.mjs — permanent live-Chromium /graph console gate reused here to prove zero favicon/CSP entries"
provides:
  - "web/static/favicon.svg, favicon-32.png, apple-touch-icon.png, codegraph-mark.svg — cleaned, locked-colour codegraph mark shipped as static self-served files"
  - "web/src/routes/+layout.svelte — three ordered static icon <link> tags replacing the Vite-inlined data: URI import"
  - "web/build/** rebuilt and drift-clean, embedding the new static assets and the layout change"
affects: [01-09]

# Actuals (#2632)
actuals:
  tokens: 68935
  tasks: 2
  commits: 2
  plan_head_before: 81e95b6563b2e2cc5e48b36cee2b2a5ac68ba763

tech-stack:
  added: []
  patterns:
    - "Static self-served binary/vector assets under web/static/ need zero server code — SvelteKit's adapter-static copies web/static/<file> to web/build/<file> verbatim, exactly like the pre-existing web/static/robots.txt precedent."
    - "Verify a static asset's live behaviour against the actually-rendered DOM, not the pre-hydration static HTML shell, when the app runs with `export const ssr = false` (SPA mode) — svelte:head content never appears in the committed web/build/index.html for either the pre-fix or post-fix build, confirmed identical for both."

key-files:
  created:
    - web/static/favicon.svg
    - web/static/codegraph-mark.svg
    - web/static/favicon-32.png
    - web/static/apple-touch-icon.png
  modified:
    - web/src/routes/+layout.svelte
    - web/build/** (rebuilt output, committed per web:drift's contract)
  deleted:
    - web/src/lib/assets/favicon.svg

key-decisions:
  - "Rendered the two rasters with rsvg-convert 2.63.0 (already on this machine) rather than adding an image-processing devDependency, per D-04's discretion and RESEARCH.md's Package Legitimacy Audit (zero new packages this phase)."
  - "Cleaned both SVGs with a small one-off python3 regex script (not committed) rather than hand-editing ~7-17KB of C2PA-metadata-laden markup, to guarantee every remaining <path> byte stayed untouched — verified the cleanup touched nothing but the four named removals (metadata, xmlns:c2pa, preserveAspectRatio, style, width/height, plus the bare mark's background rect)."
  - "16px legibility (Task 2's <human-check>) is recorded with rendered evidence but left unresolved pending end-of-phase human review, per workflow.human_verify_mode=end-of-phase — D-01 locks the mark's geometry, so no autonomous stroke/viewBox adjustment was made even though the 16px render looked tight in my own inspection. See Issues Encountered."

requirements-completed: [FIX-02]

coverage:
  - id: D1
    description: "Both shipped SVGs (favicon.svg, codegraph-mark.svg) are metadata-free, keep their locked fills (rgb(29,78,216) tile / rgb(15,23,42) bare edges), retain every path, and carry a square viewBox with no width/height"
    requirement: FIX-02
    verification:
      - kind: other
        ref: "rg-based svg-clean/colour/full-canvas checks from 01-06-PLAN.md Task 1 <verify>"
        status: pass
    human_judgment: false
  - id: D2
    description: "favicon-32.png (32x32) and apple-touch-icon.png (180x180) are real, non-blank renders of the mark, not blank canvases"
    requirement: FIX-02
    verification:
      - kind: other
        ref: "python3 PNG chunk-parse + zlib IDAT decode, corrected for the plan's off-by-4 chunk-offset bug (see Issues Encountered) — both files pass dimension, size and >4-distinct-byte-value checks"
        status: pass
    human_judgment: false
  - id: D3
    description: "The root layout carries exactly three icon links, SVG before 32px PNG before apple-touch-icon, with no Svelte expression in any href, and the stock Svelte logo is gone and unreferenced"
    requirement: FIX-02
    verification:
      - kind: other
        ref: "rg link-count/link-order checks + test ! -e web/src/lib/assets/favicon.svg from 01-06-PLAN.md Task 2 <verify>"
        status: pass
    human_judgment: false
  - id: D4
    description: "internal/uiserver/spa.go and spa_test.go are byte-unchanged across this plan (D-05); go test ./internal/uiserver/... is green"
    requirement: FIX-02
    verification:
      - kind: unit
        ref: "shasum -a 256 -c against the plan's pre-recorded digests (both OK) + go test ./internal/uiserver/... -count=1 (ok, 77.9s)"
        status: pass
    human_judgment: false
  - id: D5
    description: "task web:drift passes after the rebuild; web/build/index.html references no data:image URI"
    requirement: FIX-02
    verification:
      - kind: other
        ref: "task web:drift (source+output digest MATCH); rg check on web/build/index.html"
        status: pass
    human_judgment: false
  - id: D6
    description: "A live console run against this repo's index reports zero favicon/CSP/refused-load entries, and the three icon links actually render in the live DOM, each resolving 200 with the correct content-type"
    requirement: FIX-02
    verification:
      - kind: e2e
        ref: "node web/scripts/graph-console-check.mjs --corpus self --out (jq assertion: zero favicon/CSP/refused-load entries); ad hoc Playwright DOM check (not committed) confirming <link> order and 200 responses"
        status: pass
    human_judgment: false
  - id: D7
    description: "The shipped mark reads correctly at 16px and 32px, compared against the round-2 comparison sheet, in a real tab bar"
    requirement: FIX-02
    verification: []
    human_judgment: true
    rationale: "Legibility at favicon size is a stated visual judgement (Task 2's own <human-check>), deferred to end-of-phase per workflow.human_verify_mode=end-of-phase. Rendered evidence gathered and recorded in Issues Encountered; a human must confirm before this is closed."

duration: ~30min
completed: 2026-09-15
status: complete
---

# Phase 1 Plan 6: Codegraph Favicon Under the Unchanged CSP Summary

**The stock SvelteKit favicon is replaced by the locked codegraph "CG" ligature mark, shipped as four cleaned static files under `web/static/` and linked via three ordered `<link>` tags — closing WINDOWS #30/FIX-02 by fixing the asset, never the CSP.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 2
- **Files modified:** 21 (4 new static assets, 1 layout edit, 1 deleted stock logo, 15 rebuilt `web/build/` entries)

## Accomplishments

- Cleaned both locked Recraft SVG sources (`01-mark-favicon-tile.raw.svg`, `01-mark-bare.raw.svg`) per D-03: stripped the ~17-20KB C2PA `<metadata>` block, the `xmlns:c2pa` namespace, `preserveAspectRatio="none"`, and `style="display: block;"`, dropped `width`/`height` so both icons scale to their container, and (bare mark only) removed the leading full-canvas white background rect path — every remaining `<path>` byte-identical to the source, including `fill="rgb(29,78,216)"` (tile) and `fill="rgb(15,23,42)"` (bare edges).
- Rendered `favicon-32.png` (32x32) and `apple-touch-icon.png` (180x180) from the cleaned tile SVG with `rsvg-convert 2.63.0` — `rsvg-convert -w N -h N --keep-aspect-ratio -o <out> web/static/favicon.svg` — committed as static files with zero new `web/package.json` dependency.
- Replaced `+layout.svelte`'s single Vite-inlined `data:` URI favicon link with three static, root-relative `<link>` tags in SVG -> 32px PNG -> apple-touch-icon order (documented inline as load-bearing), and deleted the stock Svelte logo (`web/src/lib/assets/favicon.svg`) with no remaining reference anywhere under `web/src`.
- Proved `internal/uiserver/spa.go` and `spa_test.go` are byte-unchanged across the whole plan (sha256 match against the plan's pre-recorded digests) — the CSP was never touched, only the asset.
- Rebuilt `web/build/` (`task web:build`), confirmed `task web:drift` PASS on both the source and output digests, and ran the live two-corpus console gate (`web/scripts/graph-console-check.mjs --corpus self`) plus an ad hoc live-DOM check confirming all three icon links render in the correct order and each resolves 200 under the unchanged CSP.

## Task Commits

1. **Task 1: Clean the locked mark SVGs and render the two raster icons** - `033c8adb` (feat)
2. **Task 2: Link the static icons from the root layout and delete the Svelte logo** - `6a73f170` (fix)

**Plan metadata:** _pending — added in the final docs commit_

## Files Created/Modified

- `web/static/favicon.svg` - cleaned tile mark, the shipped favicon (SVG)
- `web/static/codegraph-mark.svg` - cleaned bare mark, no consumer this phase (D-02)
- `web/static/favicon-32.png` - 32x32 raster, rendered from the cleaned tile SVG
- `web/static/apple-touch-icon.png` - 180x180 raster, rendered from the cleaned tile SVG
- `web/src/routes/+layout.svelte` - removed the `$lib/assets/favicon.svg` import; replaced the single icon `<link>` with three ordered static links
- `web/src/lib/assets/favicon.svg` - DELETED (stock Svelte logo)
- `web/build/**` - rebuilt SPA output (15 files: `.build-manifest`, `index.html`, `_app/version.json`, four content-hashed chunk renames/adds/deletes from the source change, plus the four new static assets copied verbatim)

## Decisions Made

See `key-decisions` in frontmatter: rsvg-convert chosen over adding an image-processing devDependency; a throwaway python3 script used for SVG cleanup to guarantee byte-identical path preservation; the 16px legibility human-check deliberately left open for end-of-phase review rather than unilaterally adjusting D-01-locked geometry.

## Deviations from Plan

None — plan executed exactly as written. Two of the plan's own literal `<verify>` commands could not be run as-written due to pre-existing issues in the verify commands themselves (not in the shipped assets); both are documented in full below with the corrected verification evidence that proves the underlying acceptance criteria.

## Issues Encountered

- **Task 1's literal PNG verify command has an off-by-4 IDAT-extraction bug.** The command `b[i+8:i+8+struct.unpack('>I',b[i-4:i])[0]]` (where `i` is the offset of the `IDAT` type-field match) skips 4 bytes too many — PNG chunk layout is `length(4) + type(4) + data(length) + crc(4)`, so data starts at `i+4`, not `i+8`. Running the command verbatim against both correctly-rendered PNGs raises `zlib.error: Error -3 while decompressing data: incorrect header check` — confirmed this is unconditional (any valid single-IDAT-chunk PNG fails it, not just ours) by manually parsing the chunk table (`IHDR`/`bKGD`/`IDAT`/`IEND`, one `IDAT` occurrence in each file, correct declared lengths) and re-running the zlib decompress with the correct offset (`i+4`), which succeeds: `favicon-32.png` decodes to 181 distinct byte values, `apple-touch-icon.png` to 255 — both far past the `>4` blank-canvas floor. Both rasters are proven real, correctly-sized, non-blank renders; the plan's diagnostic script itself needs its offset fixed, independent of this deliverable.
- **Task 2's literal `web/build/index.html` data:image/favicon `rg` check cannot pass for this app, in either its pre-fix or post-fix state.** `web/src/routes/+layout.ts` sets `export const ssr = false; export const prerender = false` (pure SPA mode) — confirmed via `git show HEAD~1:web/build/index.html | rg favicon` returning zero matches even against the OLD build carrying the `data:` URI. `<svelte:head>` content is only injected client-side after JS hydration; it was never present in the static HTML shell for either the broken or the fixed icon markup, so the check's absence-of-`data:image` half trivially passes but its presence-of-`/favicon.svg` half was never going to find anything in the static file regardless of correctness. Verified the real behaviour instead: booted `task build:release`'s binary via `codegraph ui --no-open`, loaded the live page in headless Chromium, and confirmed via `page.$$eval` that all three `<link>` elements are present in the live DOM in the documented SVG -> PNG -> apple-touch-icon order, and each resolves `200` with the correct `content-type` (`image/svg+xml`, `image/png`, `image/png`) against the running server. This is the positive proof the plan's live console-gate step (also run, see below) only proves negatively (absence of errors).
- **16px legibility (Task 2's `<human-check>`) is not yet human-confirmed.** Rendered `web/static/favicon.svg` at 16px and 32px via `rsvg-convert`, magnified 8x both with nearest-neighbour (`magick -filter point`) and Lanczos (`magick -filter Lanczos`) for a fair comparison against a real browser's anti-aliased scaling, and visually compared against `01-mark-round2-sheet.png`'s middle tile. At 32px the C and G are clearly separable and match the sheet. At 16px, in my own inspection, the shared vertical edge and the C/G separation are noticeably tighter/blurrier than at 32px — it reads closer to a rounded/pretzel shape than a crisp "CG" at that exact pixel size, though the corner-node styling is preserved. Per D-01, the mark's geometry is locked and costly to change, and this decision explicitly requires human judgement (Task 2's own text: "If it does not read at 16px, the remedy is a viewBox or stroke-weight adjustment... NOT a different mark"). Per `workflow.human_verify_mode: end-of-phase`, this is deliberately left open rather than resolved autonomously — no SVG geometry was touched. **This item needs end-of-phase human sign-off**; if the 16px render is judged illegible in a real tab bar, the fix is a viewBox/stroke-weight adjustment to `web/static/favicon.svg` (never `codegraph-mark.svg`'s underlying artwork), tracked as a follow-up to this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- FIX-02 is functionally closed: the favicon loads under the unchanged CSP, verified live end-to-end (DOM, network, console).
- The 16px legibility human-check (D7 above) is the one open item — surface it at the phase's end-of-phase human-verify checkpoint alongside the phase's other deferred `<human-check>` items.
- No blockers for plan 01-09 (FIX-05 layout fix), which depends on 01-01's console gate, not on this plan.

## Self-Check: PASSED

- FOUND: `web/static/favicon.svg`
- FOUND: `web/static/codegraph-mark.svg`
- FOUND: `web/static/favicon-32.png`
- FOUND: `web/static/apple-touch-icon.png`
- FOUND: `web/build/favicon.svg`, `web/build/codegraph-mark.svg`, `web/build/favicon-32.png`, `web/build/apple-touch-icon.png` (rebuilt output)
- FOUND: commit `033c8adb` (Task 1)
- FOUND: commit `6a73f170` (Task 2)
- Re-ran plan-level `<verification>`: SVG clean/colour checks pass; PNGs pass corrected chunk-parse decode; layout link-count=3 and link-order sorted; `spa.go`/`spa_test.go` sha256 both `OK`; `go test ./internal/uiserver/... -count=1` → `ok` (77.9s); `task web:drift` → PASS (source 119 files, output 36 files, both digests MATCH); `web/build/index.html` has no `data:image` (though see Issues Encountered on this check's limited value for an SPA build); live console gate reports zero favicon/CSP/refused-load entries (only pre-existing FIX-04/FIX-05 entries, as the plan predicts); live DOM check confirms all three icon links render in order and resolve 200.
- `git rev-list --count 81e95b6..HEAD` = 2, matching `actuals.commits: 2`.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*
