---
phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
plan: 05
subsystem: ui
tags: [tailwindcss-v4, shadcn-svelte, sveltekit, connect-rpc, app-shell, getstatus]

requires:
  - phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain
    provides: "02-01's web/ pnpm toolchain and embed seam; 02-02's complete D-10 routing/CSP rule (script-src hash re-derives from index.html at handler-construction time); 02-03's web/src/lib/client.ts uiClient"
provides:
  - "Tailwind v4 + shadcn-svelte's Vega token set, installed through pnpm dlx shadcn-svelte@1.5.0 init, not hand-rolled"
  - "web/src/routes/+layout.svelte — the app shell with four named navigation slots (Browse, Workbench, Graph, Health)"
  - "web/src/routes/{browse,workbench,graph,health}/+page.svelte — four real, non-prerendered placeholder routes"
  - "web/src/routes/+page.svelte — the live GetStatus render, all nine response fields, three named degrade states"
  - "The final Phase 2 web/build/ tree — the source state 02-06's drift-guard marker will be computed against"
affects: [02-06, 02-07]

actuals:
  tokens: 10830
  tasks: 3
  commits: 3

tech-stack:
  added: ["tailwindcss 4.3.3", "@tailwindcss/vite 4.3.3", "tw-animate-css 1.4.0", "clsx 2.1.1", "tailwind-merge 3.6.0", "@fontsource-variable/inter ^5.3.0 (shadcn-svelte init side effect)", "@lucide/svelte ^1.33.0 (shadcn-svelte init side effect)", "tailwind-variants ^3.3.1 (shadcn-svelte init side effect)"]
  patterns:
    - "shadcn-svelte's own generated app.css imports `shadcn-svelte/tailwind.css` — a package-relative CSS module that assumes shadcn-svelte stays a devDependency. This plan's acceptance criteria forbid that (T-02-05-06), so the import is replaced with the SAME file's content inlined verbatim (accordion keyframes, bits-ui data-state custom variants, no-scrollbar utility) rather than hand-authoring a substitute — the design TOKEN set itself (:root/.dark/@theme inline) is untouched and still exactly what shadcn-svelte init generated"
    - "GetStatusResponse's three-state classification (healthy / reindexing / not-initialized) reads indexingInProgress directly off the response, never storeExists && !initialized — the client never re-implements a server-side degrade decision"

key-files:
  created:
    - web/components.json
    - web/src/app.css
    - web/src/lib/utils.ts
    - web/src/routes/browse/+page.svelte
    - web/src/routes/workbench/+page.svelte
    - web/src/routes/graph/+page.svelte
    - web/src/routes/health/+page.svelte
  modified:
    - web/package.json
    - web/pnpm-lock.yaml
    - web/vite.config.ts
    - web/src/routes/+layout.svelte
    - web/src/routes/+page.svelte
    - web/build/** (rebuilt, recommitted — final Phase 2 source tree)

key-decisions:
  - "shadcn-svelte init's interactive prompts have no fully non-interactive CLI path (piped stdin is treated as EOF and the process exits with nothing written) — drove the CLI through a real pty (Python's stdlib pty module + pyte for VT100 rendering) rather than skipping the CLI or hand-authoring its output. Every prompt answer is recorded below."
  - "Removed `shadcn-svelte` from package.json/pnpm-lock.yaml after `init` auto-added itself as a devDependency (^1.5.0) — the task's own acceptance criteria and threat T-02-05-06 require it stay dlx-only. `pnpm remove shadcn-svelte -D` cleanly excised it from both files with zero residual references."
  - "shadcn-svelte init required web/src/app.css to ALREADY exist (it validates the path and refuses to proceed against a nonexistent file, contrary to a literal reading of 'init generates app.css'); created a one-line `@import \"tailwindcss\";` placeholder before the CLI's CSS-path prompt so init could overwrite it with the real token set — this is not hand-authoring the token set, since the CLI's own confirmed 'Continue?' step then replaced every line with its generated output."
  - "The three states' active-entry / degrade-classification code paths are unit-untestable without a JS test framework (this phase intentionally has none — see Task 3's own <behavior> block). Verified end-to-end instead: real `codegraph ui` process, curl against both an indexed repo (real non-zero counts) and an empty directory (the exact all-defaults-omitted not-initialized JSON shape Connect's JSON encoding produces), matching 02-01's precedent for the same class of test-framework gap."

requirements-completed: []  # BLD-02 is declared by 02-01/02-02/02-03/02-05 (all now have SUMMARYs); RPC-03 is declared by 02-02/02-05 (both now have SUMMARYs). Left empty here per the shared-ID gate (#2388) contract — state.update_requirements computes readiness via requirements.ready-ids and marks both complete in REQUIREMENTS.md as part of this plan's close-out, not by hand-editing this field.

coverage:
  - id: D1
    description: "Tailwind v4 and shadcn-svelte's Vega token set are installed through pnpm dlx shadcn-svelte@1.5.0 init (exact pin, registry dist.integrity confirmed before execution) rather than hand-rolled, and the build emits real, non-empty CSS"
    requirement: BLD-02
    verification:
      - kind: other
        ref: "cd web && pnpm install --frozen-lockfile && pnpm build && find build/_app/immutable -name '*.css' | head -1 — non-empty, 11868 bytes"
        status: pass
      - kind: other
        ref: "pnpm exec svelte-check --threshold error --output human — 'found 0 errors and 0 warnings'"
        status: pass
      - kind: other
        ref: "components.json + src/app.css + src/lib/utils.ts present, none hand-authored; shadcn-svelte absent from package.json/pnpm-lock.yaml (rg -c 'shadcn-svelte' both files — no matches)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The app shell has four named navigation slots (Browse/Workbench/Graph/Health), each a real client-side route, none prerendered — ROADMAP criterion 2 stays honestly testable"
    requirement: RPC-03
    verification:
      - kind: other
        ref: "pnpm build && test -s build/index.html && test ! -e build/{browse,workbench,graph,health}/index.html — all four absent, index.html present and non-empty"
        status: pass
      - kind: other
        ref: "pnpm exec svelte-check --threshold error — 'found 0 errors and 0 warnings'"
        status: pass
    human_judgment: false
  - id: D3
    description: "The live GetStatus render walks the full seam (embedded asset -> generated TS client -> Connect JSON -> Phase 1's handler -> real index data) and renders honestly in the indexed and degraded states"
    requirement: RPC-03
    verification:
      - kind: integration
        ref: "go test ./internal/uiserver/... -run 'TestEmbeddedFSMatchesOnDiskBuildTree|TestEmbeddedBuildTreeIsNonTrivial|TestSPA' -v — 15/15 named PASS, 0 FAIL, against the rebuilt tree"
        status: pass
      - kind: other
        ref: "curl against a live `codegraph ui --no-open` process on this indexed repo: GetStatus returned {initialized:true, nodeCount:5763, edgeCount:13585, fileCount:495, stale:true, storeExists:true}; against an empty temp dir: {} (all defaults, matching the not-initialized classification path)"
        status: pass
      - kind: other
        ref: "curl -D on GET / against the same live process: Content-Security-Policy script-src hash re-derived to sha256-4Q4njbW/jqf4iiAzFO/5g3Zkm91EV0s8vobwOnApFtA= against the new index.html's bootstrap script, no manual CSP edit"
        status: pass
    human_judgment: true
    rationale: "The plan's own <verify> names a literal browser open (console error check, click-through navigation, direct-URL reload) as a <human-check>. Per workflow.human_verify_mode=end-of-phase, this is deferred to end-of-phase UAT; this executor performed the mechanical equivalent (curl against both the indexed and not-indexed states, CSP header inspection) and it passed, matching 02-01's precedent for the same deferral shape."

duration: ~55min
completed: 2026-08-24
status: complete
---

# Phase 2 Plan 5: SPA Toolchain, Embedded App Shell & JS Supply Chain — App Shell & Live GetStatus Summary

**Tailwind v4 + shadcn-svelte's Vega token set installed via `pnpm dlx shadcn-svelte@1.5.0 init` (driven through a real pty since the CLI has no non-interactive path), a four-slot app shell with real Browse/Workbench/Graph/Health routes, and a live `GetStatus` render proving the full embed-to-RPC seam works in a real `codegraph ui` process against both an indexed repo and an empty directory.**

## Performance

- **Duration:** ~55 min (includes significant time discovering that `shadcn-svelte init` cannot be driven by piped stdin and building a pty-based driver to answer its prompts)
- **Completed:** 2026-08-25T00:10:18Z
- **Tasks:** 3
- **Files:** 7 created, 6 modified (including the rebuilt `web/build/` tree)

## Accomplishments

- Installed `tailwindcss@4.3.3` and `@tailwindcss/vite@4.3.3` as devDependencies, wired `@tailwindcss/vite` into `vite.config.ts`'s plugin array before `sveltekit()`; confirmed no `postcss.config.*` exists anywhere under `web/`
- Added `clsx@2.1.1`, `tailwind-merge@3.6.0` (runtime) and `tw-animate-css@1.4.0` (dev) at the plan's pinned versions
- Confirmed `shadcn-svelte@1.5.0`'s registry `dist.integrity` matched the plan's pinned string exactly, twice (once well before execution, once immediately before invocation), then ran `pnpm dlx shadcn-svelte@1.5.0 init` against the existing project
- **`shadcn-svelte init` has no working non-interactive path.** Piped stdin (`printf '\n...' | pnpm dlx shadcn-svelte@1.5.0 init`) is silently treated as EOF and the process exits 0 having created nothing — confirmed empirically before building a workaround. Drove the CLI through a real pseudo-terminal instead: Python's stdlib `pty` module to fork the child attached to a pty, and `pyte` (installed into a scratch venv) to render the actual VT100 screen so each prompt could be read and answered correctly, rather than guessing blind against ANSI-noise text
- Recorded every prompt answer: preset method = "Choose from a list of pre-configured presets" (default); preset = **Vega - Lucide / Inter** ("The classic shadcn/ui look", default); global CSS file = `src/app.css` (had to be pre-created as a one-line `@import "tailwindcss";` placeholder — the CLI validates the path exists before proceeding, then overwrites it with the full generated token set); lib alias = `$lib`; components alias = `$lib/components`; ui alias = `$lib/components/ui`; utils alias = `$lib/utils`; hooks alias = `$lib/hooks`; "Updates to your src/app.css are required... Continue?" = Yes (default)
- `init` generated `web/components.json`, `web/src/app.css` (full Vega token set: `--background`, `--foreground`, `--primary`, etc. plus `@theme inline` mappings), and `web/src/lib/utils.ts` (`cn()` helper) — none hand-authored
- **Deviation, auto-fixed:** `init` added `shadcn-svelte@^1.5.0` itself to `package.json`'s devDependencies (its default assumption) — removed via `pnpm remove shadcn-svelte -D`, confirmed zero remaining references in `package.json` or `pnpm-lock.yaml`
- **Deviation, auto-fixed:** the generated `app.css` imports `shadcn-svelte/tailwind.css`, unresolvable once `shadcn-svelte` is removed as a dependency — inlined that file's own verbatim content (accordion keyframes, `bits-ui` `data-state` custom variants, a `no-scrollbar` utility) in its place; this is boilerplate CSS infrastructure, not the generated design-token set, so it does not violate "do not hand-author the token set"
- `pnpm approve-builds`: **zero** `allowBuilds` entries before and after this task's installs — no lifecycle/build scripts in the Tailwind/shadcn dependency set
- Imported `app.css` from `+layout.svelte` so the tokens and Tailwind output actually reach the rendered page
- Did **not** run `shadcn-svelte add` for any component — `src/lib/components/ui/` and `src/lib/hooks/` are empty directories (confirmed via `git ls-files`); BLD-06's audit-blind-spot boundary (registry-vendored component source) stays not-yet-live
- Extended `+layout.svelte` into the app shell: header nav with four entries (Browse, Workbench, Graph, Health) as real `<a>` `href`s, active entry highlighted via `$app/state`'s `page` store
- Added four placeholder routes, each naming the phase and one-line ROADMAP goal that will fill it: `browse` → Phase 3 (find any symbol/file, verbatim source with callers/callees/blast radius); `workbench` → Phase 4 (run the four graph analyses interactively); `graph` → Phase 5 (whole-repo file/package-granularity picture); `health` → Phase 4's index-health half (index trustworthiness at a glance)
- Confirmed none of the four routes prerender: `pnpm build` produces `build/index.html` with no per-route `index.html`
- Replaced the tracer's placeholder `+page.svelte` with the live status render: imports `uiClient` from `$lib/client` (02-03), calls `getStatus({})` on mount, constructs no transport of its own
- Rendered all **nine** `GetStatusResponse` fields with human labels (Initialized, Schema version, Nodes, Edges, Files, Stale, Indexed commit, Index directory found, Re-index in progress) — verified against `internal/uiproto/uiv1/ui.proto:116-177`, confirming the message declares `indexing_in_progress = 9`
- Three states (healthy / reindexing / not-initialized) are each named in the rendered output; `indexingInProgress` selects the third state directly — never `storeExists && !initialized`
- Empty `commitSha` renders an explicit "unknown (graph predates the commit-aware schema field, or was built outside a git checkout)" rather than an error or blank cell
- A transport rejection is caught once and renders a named error state ("Could not reach the server: ...") rather than an indefinite loading spinner
- Rebuilt `web/build/` from the final Phase 2 source tree (Tailwind CSS output, four nav routes, live status page) — this is the tree 02-06's drift-guard marker will be computed against
- Re-ran 02-01's embedded-vs-on-disk file-list diff and 02-02's full SPA/CSP suite against the rebuilt tree: **15/15** named tests pass (`TestEmbeddedFSMatchesOnDiskBuildTree`, `TestEmbeddedBuildTreeIsNonTrivial`, plus 13 `TestSPA*` tests), 0 failures; CSP `script-src` hash re-derived automatically to `sha256-4Q4njbW/jqf4iiAzFO/5g3Zkm91EV0s8vobwOnApFtA=` against the new `index.html`'s bootstrap script bytes, no code change needed
- Mechanical human-check equivalent (browser deferred to end-of-phase UAT, see Coverage D3): ran `go build -o /tmp/cg ./cmd/codegraph && /tmp/cg ui --no-open` against this repo (which HAS an index) and curled it — `GetStatus` returned real counts (`nodeCount: 5763, edgeCount: 13585, fileCount: 495, initialized: true, stale: true, storeExists: true`); ran the same binary from an empty temp directory — `GetStatus` returned `{}` (every field at its Connect-JSON default, matching the not-initialized classification path exactly, since protobuf JSON omits default/false/zero/empty values)
- `git status --porcelain web/build` is empty after the final commit — the rebuilt tree is committed, not left dirty

## Task Commits

1. **Task 1: Tailwind v4 and shadcn-svelte's design system, installed not hand-rolled** — `4e17cef` `feat(02-05): install Tailwind v4 and shadcn-svelte's token set`
2. **Task 2: Root layout with four named navigation slots and four real routes** — `a1e75da` `feat(02-05): app shell with four named navigation slots and real routes`
3. **Task 3: The live GetStatus render — the seam proven in the browser** — `73cf405` `feat(02-05): live GetStatus render and final Phase 2 build`

**Plan metadata:** committed separately after this SUMMARY (see below).

_Note on Task 3's `tdd="true"` attribute: this task's own `<behavior>` block explicitly states "there is no JS test framework in this phase and none is required" and that verification is an end-to-end exercise via the existing Go test suite plus a human-check. There was therefore no synthetic JS unit test to write RED-then-GREEN against; a single `feat` commit carries the implementation, verified against the plan's own Go-side `<verify>` block (15/15 named PASS) and the mechanical human-check equivalent above. See "TDD Gate Compliance" below._

## Files Created/Modified

- `web/components.json`, `web/src/app.css`, `web/src/lib/utils.ts` — shadcn-svelte `init`'s generated output (Vega preset), none hand-authored beyond the pre-existence placeholder and the `shadcn-svelte/tailwind.css` inline substitution
- `web/package.json`, `web/pnpm-lock.yaml` — Tailwind/shadcn dependency set added; `shadcn-svelte` itself removed after `init` auto-added it
- `web/vite.config.ts` — `@tailwindcss/vite` plugin added to the plugin array
- `web/src/routes/+layout.svelte` — app shell: `app.css` import, four-entry nav, `$app/state`-driven active highlighting
- `web/src/routes/{browse,workbench,graph,health}/+page.svelte` — four real placeholder routes
- `web/src/routes/+page.svelte` — the live `GetStatus` render, all nine fields, three named states
- `web/build/**` — rebuilt and recommitted (final Phase 2 source tree)

## Decisions Made

See `key-decisions` in frontmatter.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `shadcn-svelte init` has no non-interactive CLI path**
- **Found during:** Task 1, step (c) — running the pinned `init` invocation
- **Issue:** `pnpm dlx shadcn-svelte@1.5.0 init` renders an interactive, full-screen TUI (select lists, text prompts) with no `--yes`/non-interactive flag. Piped stdin is treated as immediate EOF: the process exits 0 having written no files at all — verified directly before attempting any workaround.
- **Fix:** Built a small pty-based driver (Python's stdlib `pty` module + `pyte` for VT100 screen rendering, installed into a scratch venv since neither is a project dependency) to spawn the CLI attached to a real pseudo-terminal, read its rendered screen state, and send `\r` to accept each prompt's default option — recording every answer as it appeared.
- **Files modified:** none (tooling-only workaround; the driver script lives in the session scratchpad, not the repo)
- **Verification:** `web/components.json`, `web/src/app.css`, `web/src/lib/utils.ts` all present and match the CLI's own generated output for the Vega preset
- **Committed in:** `4e17cef`

**2. [Rule 3 - Blocking] `shadcn-svelte init` requires its target CSS file to already exist**
- **Found during:** Task 1, mid-`init` — the "Where is your global CSS file?" prompt
- **Issue:** `init` validates `src/app.css` and refuses to proceed ("does not exist. Please enter a valid path.") if the file is absent — despite generating that file's contents as part of its own output.
- **Fix:** Created a minimal one-line placeholder (`@import "tailwindcss";`) before the CLI reached that prompt, so the path validation passed; the CLI's own subsequent "Continue?" confirmation then overwrote every line with its generated token set. No part of the final token set was hand-typed.
- **Files modified:** `web/src/app.css` (placeholder only — fully overwritten by the CLI before this task ended)
- **Verification:** `web/src/app.css`'s final content matches `init`'s own generated output byte-for-byte apart from the one intentional substitution in deviation 4 below
- **Committed in:** `4e17cef`

**3. [Rule 3 - Blocking] `shadcn-svelte init` added itself as a devDependency, violating this task's own acceptance criteria**
- **Found during:** Task 1, post-`init` — inspecting `package.json`
- **Issue:** `init` added `"shadcn-svelte": "^1.5.0"` to `devDependencies` as its default behavior. The task's own acceptance criteria and threat T-02-05-06 explicitly require `shadcn-svelte` to remain dlx-only, never a committed dependency.
- **Fix:** `pnpm remove shadcn-svelte -D`. Confirmed zero remaining references via `rg -c "shadcn-svelte" package.json pnpm-lock.yaml`.
- **Files modified:** `web/package.json`, `web/pnpm-lock.yaml`
- **Verification:** `rg` confirms zero matches; `pnpm build` still succeeds after the removal (once deviation 4 below is also applied)
- **Committed in:** `4e17cef`

**4. [Rule 1 - Bug] Removing `shadcn-svelte` broke a package-relative CSS import `init` had written**
- **Found during:** Task 1, first `pnpm build` after deviation 3
- **Issue:** `web/src/app.css` (generated by `init`) contains `@import "shadcn-svelte/tailwind.css";`, which resolves to a real file inside the now-removed `shadcn-svelte` package. Build failed: `Can't resolve 'shadcn-svelte/tailwind.css'`.
- **Fix:** Inlined that file's own content verbatim (read directly from the pnpm dlx cache before it could be garbage-collected) in place of the import — accordion keyframes, `bits-ui` `data-state` custom variants (`data-open`, `data-closed`, `data-checked`, etc.), and a `no-scrollbar` utility. This is boilerplate CSS infrastructure the CLI itself ships, not part of the generated design-token palette (`:root`/`.dark`/`@theme inline`), so inlining it does not violate the "do not hand-author the token set" instruction.
- **Files modified:** `web/src/app.css`
- **Verification:** `pnpm build` succeeds, emits real CSS (11868 bytes); `pnpm exec svelte-check --threshold error` reports 0 errors
- **Committed in:** `4e17cef`

---

**Total deviations:** 4 auto-fixed (all Rule 3/blocking except the last, which is Rule 1/bug — a direct consequence of applying deviation 3).
**Impact on plan:** All four are mechanical corrections needed to make the plan's own explicit prohibition (no `shadcn-svelte` in `package.json`) and its own instruction (do not hand-author the token set) simultaneously true against what the pinned CLI actually does today. No architecture change, no scope creep — the design token set itself is exactly what `shadcn-svelte init` generated for the Vega preset.

## Issues Encountered

None beyond the deviations above.

## TDD Gate Compliance

Task 3 carries `tdd="true"` at the task level (not a plan-level `type: tdd`). Its own `<behavior>` block states explicitly: "there is no JS test framework in this phase and none is required" — the verification is an end-to-end exercise via the plan's Go-side `<verify>` block (embedded-tree tests + full SPA/CSP suite) plus a `<human-check>`. There is consequently no synthetic RED/GREEN JS-test pair to author; a single `feat` commit (`73cf405`) carries the implementation, gated by the plan's own re-run of 15 named Go tests (0 failures) before commit. This mirrors 02-01's precedent for the identical class of test-framework gap in this phase.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `web/build/` now reflects the complete Phase 2 source tree (Tailwind v4 + shadcn tokens, four nav routes, live status page) — 02-06's drift-guard marker (`web/build/.build-manifest`) should be computed against exactly this commit's tree, not an earlier one.
- **Requirement readiness:** `BLD-02` is declared by 02-01/02-02/02-03/02-05 — all four now have SUMMARYs, so `requirements.ready-ids` is expected to report it ready at this plan's state-update step. `RPC-03` is declared by 02-02/02-05 — both now have SUMMARYs, same expectation. Neither was hand-marked in this SUMMARY's `requirements-completed` field; the shared-ID gate computes and writes REQUIREMENTS.md.
- **BLD-06's recorded limitation stays not-yet-live:** Phase 2 ships zero shadcn-svelte components (`src/lib/components/ui/` is empty). The gap (registry-vendored component source bypassing `pnpm audit`) becomes live at Phase 3's first `shadcn-svelte add <component>` — inherited named, not rediscovered.
- **The pty-driver workaround for `shadcn-svelte init`'s interactive-only CLI is a one-time cost for this phase.** No future Phase 2 plan re-runs `init` (Phase 3's `shadcn-svelte add <component>` is a different, and per its own `--help`, likely also-interactive command Phase 3 will need to solve on its own terms — flagged here so it isn't rediscovered from scratch).
- 02-05's own human-check (literal browser open, console error check, click-through navigation) is deferred to end-of-phase UAT per `workflow.human_verify_mode=end-of-phase`; this plan's mechanical curl-based equivalent passed against both the indexed and not-indexed states, and the CSP header showed zero indication of a violation-worthy mismatch (the hash re-derivation covers exactly the one inline bootstrap script the new build ships).

## Self-Check: PASSED

Confirmed on disk: `web/components.json`, `web/src/app.css`, `web/src/lib/utils.ts`, `web/src/routes/+layout.svelte`, `web/src/routes/browse/+page.svelte`, `web/src/routes/workbench/+page.svelte`, `web/src/routes/graph/+page.svelte`, `web/src/routes/health/+page.svelte`, `web/src/routes/+page.svelte`, `web/vite.config.ts`, `web/package.json`, `web/pnpm-lock.yaml`, this SUMMARY.md.
Confirmed in `git log --oneline --all`: `4e17cef`, `a1e75da`, `73cf405`.
Re-ran the plan's exact `<verification>` block post-commit at HEAD: `pnpm install --frozen-lockfile` clean; `pnpm exec svelte-check --threshold error` — "0 errors, 0 warnings"; `pnpm build` succeeds, real non-empty CSS emitted; `go build ./...` and `go test ./internal/uiserver/...` — full package green (`ok`, 11.063s); `git status --porcelain web/build` empty; no `postcss.config.*` anywhere under `web/`; `shadcn-svelte` absent from `package.json`/`pnpm-lock.yaml`; `src/lib/components/ui/` and `src/lib/hooks/` empty per `git ls-files`.

---
*Phase: 02-spa-toolchain-embedded-app-shell-js-supply-chain*
*Plan: 05*
*Completed: 2026-08-24*
