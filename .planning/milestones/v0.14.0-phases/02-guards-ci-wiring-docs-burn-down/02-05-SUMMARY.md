---
phase: 02-guards-ci-wiring-docs-burn-down
plan: 05
subsystem: web-ui
tags: [shadcn-svelte, vendored-components, drift, web-build, tailwind-v4]

# Dependency graph
requires: []
provides:
  - "All eight vendored shadcn-svelte component families re-vendored to one registry snapshot (shadcn-svelte@1.5.1), reviewed diff, `task web:components:drift` PASS transcript"
  - "web/build/** rebuilt in the same commit as the web/src change"
affects: [02-07 (closes WINDOWS #31 citing this plan's PASS transcript)]

# Actuals (#2632)
actuals:
  tokens: 260572
  tasks: 2
  commits: 1
  plan_head_before: 67c9f4934db68f93b6368169692c98a9275f00f3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Re-vendor via the identical pinned CLI invocation the drift monitor itself uses (`pnpm dlx shadcn-svelte@1.5.1 add <families> -y -o`), run against the real web/ tree rather than a scratch copy, then revert the CLI's incidental package.json/lockfile side effect before committing"

key-files:
  created: []
  modified:
    - web/src/lib/components/ui/button/button.svelte
    - web/src/lib/components/ui/command/command.svelte
    - web/src/lib/components/ui/command/command-group.svelte
    - web/src/lib/components/ui/command/command-input.svelte
    - web/src/lib/components/ui/command/command-item.svelte
    - web/src/lib/components/ui/command/command-link-item.svelte
    - web/src/lib/components/ui/command/command-separator.svelte
    - web/src/lib/components/ui/command/command-shortcut.svelte
    - web/src/lib/components/ui/dialog/dialog-content.svelte
    - web/src/lib/components/ui/dialog/dialog-description.svelte
    - web/src/lib/components/ui/dialog/dialog-overlay.svelte
    - web/src/lib/components/ui/input/input.svelte
    - web/src/lib/components/ui/input-group/input-group.svelte
    - web/src/lib/components/ui/input-group/input-group-addon.svelte
    - web/src/lib/components/ui/input-group/input-group-button.svelte
    - web/src/lib/components/ui/input-group/input-group-text.svelte
    - web/src/lib/components/ui/table/table-caption.svelte
    - web/src/lib/components/ui/table/table-footer.svelte
    - web/src/lib/components/ui/table/table-head.svelte
    - web/src/lib/components/ui/table/table-row.svelte
    - web/src/lib/components/ui/tabs/tabs.svelte
    - web/src/lib/components/ui/tabs/tabs-list.svelte
    - web/src/lib/components/ui/tabs/tabs-trigger.svelte
    - web/src/lib/components/ui/textarea/textarea.svelte
    - web/build/** (rebuilt bundle + .build-manifest)

key-decisions:
  - "Regenerated directly in the real web/ tree (not a scratch copy) since D-09 already established the current dev machine resolves the pinned pnpm 11.23.0 inside web/ via pnpm's own packageManager self-management, with no Corepack required — matching the plan's stated fallback rule (local run counts as fresh evidence when `cd web && pnpm --version` prints 11.23.0)"
  - "The `pnpm dlx shadcn-svelte@1.5.1 add ...` step itself ran under bare pnpm v12.4.1 (dlx does not inherit the project's packageManager pin the way a direct `pnpm install` does — confirmed both in Task 1's drift measurement and Task 2's regeneration, both printing 'Done in ...s using pnpm v12.4.1' for the dlx step specifically, vs 'pnpm v11.23.0' for the plain install steps). This is a pre-existing property of the unmodified `web:components:drift` Taskfile target, not something this plan changed. It does not undermine D-09: the file-differing set produced locally (24 files) is byte-identical in scope to CI run 34820878640's Corepack-pinned 2026-09-14 diff, empirically confirming the drift is registry-side, not a function of which pnpm version drives the `dlx` template-copy step"
  - "Reverted the CLI's incidental `@lucide/svelte` devDependency caret bump (^1.34.0 -> ^1.46.0) in web/package.json and web/pnpm-lock.yaml; `pnpm install --frozen-lockfile` confirmed lockfile consistency after the revert — no dependency change is part of this commit"
  - "Reverted `corpora/graph-console-check.json`'s regenerated timestamp/port/sha fields after running the check:graph-console gate locally — that file is gate-run output, not part of this plan's declared file scope, and its per-run values (generatedAt, ephemeral localhost port, current HEAD sha) would be a meaningless diff in the commit"

patterns-established:
  - "Empirical drift-scope cross-check: comparing a fresh local re-run's differing-file set against a prior CI run's differing-file set (both under the pinned CLI version) as independent confirmation that a monitored drift is registry-side rather than toolchain-side, without needing to control every environment variable identically"

requirements-completed: [GRD-10]

coverage:
  - id: D1
    description: "Fresh drift measurement under pinned pnpm 11.23.0 (`task web:components:drift`) run before any regeneration, naming every differing file (24 of 50 across 7 of 8 families) and confirming zero delta against RESEARCH's 2026-09-14 CI evidence"
    requirement: GRD-10
    verification:
      - kind: other
        ref: "task web:components:drift (RED) — plan Task 1 <verify> automated gates, both passed (pnpm --version == 11.23.0; transcript contains the count line, pnpm v11.23.0, >=1 differ line, non-zero exit; git status --porcelain web/ empty before and after)"
        status: pass
    human_judgment: false
  - id: D2
    description: "All eight vendored families regenerated to one registry snapshot via a single `pnpm dlx shadcn-svelte@1.5.1 add ... -y -o` invocation; incidental package.json/lockfile side effect reverted; committed as one human-reviewed commit with web/build/** rebuilt in the same commit"
    requirement: GRD-10
    verification:
      - kind: other
        ref: "task web:components:drift (PASS, all 50 files across 8 components byte-identical) — plan Task 2 <verify> gate 1"
        status: pass
      - kind: other
        ref: "cd web && pnpm check — 0 errors, 0 warnings, exit 0 — plan Task 2 <verify> gate 2"
        status: pass
      - kind: unit
        ref: "cd web && pnpm vitest run — 585/585 tests passed, zero 'unhandled errors' lines, exit 0 — plan Task 2 <verify> gate 3"
        status: pass
      - kind: other
        ref: "task web:build && task web:drift && GOTOOLCHAIN=go1.26.6 task build:release — all exit 0 — plan Task 2 <verify> gates 4/5"
        status: pass
      - kind: e2e
        ref: "task check:graph-console — self-test PASS, self+guava corpora both zero page/console errors — plan Task 2 <verify> gate 6"
        status: pass
      - kind: e2e
        ref: "node web/scripts/breadcrumb-check.mjs — success=true, gutterLinkCount=418 — plan Task 2 <verify> gate 7"
        status: pass
      - kind: other
        ref: "git show --format= --name-only HEAD — both web/src/lib/components/ui/** and web/build/** present, web/pnpm-lock.yaml absent, git status --porcelain web/ empty — plan Task 2 <verify> gate 8"
        status: pass
      - kind: other
        ref: "git log -1 commit message — subject matches ^fix\\(02-05\\): re-vendor, no ci-skip marker, no @latest — plan Task 2 <verify> gate 9"
        status: pass
    human_judgment: true
    rationale: "The per-file visual/API classification (which of the 24 changes are cosmetic reordering vs. genuine rendering changes) is a judgment call the maintainer reviews at end-of-phase per D-12 — automated gates prove the regenerated UI still builds/type-checks/tests/renders correctly, not that every visual delta is acceptable."

# Metrics
duration: 15min
completed: 2026-09-16
status: complete
---

# Phase 2 Plan 5: Re-vendor All Eight shadcn-svelte Component Families Summary

**Re-vendored all eight shadcn-svelte families to the `@1.5.1` registry's current snapshot in one reviewed commit, rebuilt `web/build/**` in the same commit, and drove the rebuilt binary through every D-12 gate (check, vitest, build+drift, release build, live graph-console, breadcrumb-check, components-drift) — all green, closing the drift `web:components:drift` has reported since 2026-09-07 for 02-07 to formally close WINDOWS #31.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-16T01:51:00Z (approx, immediately following 02-03's completion)
- **Completed:** 2026-09-16T02:02:06Z
- **Tasks:** 2 completed
- **Files modified:** 45 (24 vendored `.svelte` source files + 21 `web/build/**` output files)

## Accomplishments

- Measured the drift fresh under the pinned pnpm (11.23.0, resolved inside `web/` via pnpm's own `packageManager` self-management, no Corepack needed) before touching anything: `task web:components:drift` reported `compared 50 vendored component files across 8 components`, 24 differing files across 7 of 8 families, exit 201 — an exact match, file-for-file, to RESEARCH's 2026-09-14 CI evidence (run 34820878640). Zero delta between the two runs, taken roughly 36 hours apart, is itself corroborating evidence that this is registry-side drift.
- Regenerated all eight families (`button command dialog input input-group table tabs textarea`) in one `pnpm dlx shadcn-svelte@1.5.1 add ... -y -o` invocation directly against the real `web/` tree.
- Reviewed the diff file-by-file (see table below); classified two genuine visual changes, one additive utility class, and 21 purely cosmetic/internal changes (Tailwind class-string reordering and Tailwind v4 arbitrary-value spacing-syntax normalization with no rendered effect).
- Reverted the CLI's incidental `@lucide/svelte` devDependency caret bump; confirmed `pnpm install --frozen-lockfile` still passes (lockfile untouched, unchanged from before the run).
- Rebuilt `web/build/**` in the same commit (`task web:build && task web:drift`, both green) so `web:drift`'s SOURCE half never went RED.
- Ran every D-12 gate against the rebuilt `./codegraph` binary: `pnpm check` (0 errors), `pnpm vitest run` (585/585 passed, zero "unhandled errors" lines), `task build:release` (GOTOOLCHAIN=go1.26.6), `task check:graph-console` (self-test PASS, zero console/page errors on both the self and guava corpora), `node web/scripts/breadcrumb-check.mjs` (success), and the closing `task web:components:drift` — PASS, all 50 files across 8 components byte-identical.
- Committed once: both `web/src/lib/components/ui/**` and `web/build/**` in the same commit, no lockfile change, no CI-skip marker, no `@latest`.

## Task Commits

Both tasks executed against the same commit boundary — Task 1 was a read-only measurement (no source file touched, transcript recorded here), Task 2 produced the single re-vendor commit:

1. **Task 1: Fresh drift measurement under the pinned pnpm** — no commit (read-only measurement into a scratch tree; source tree confirmed untouched before and after)
2. **Task 2: Regenerate all eight families, review, gate, rebuild, commit once** - `75f5633c` (fix)

**Plan metadata:** committed separately per `git_commit_metadata` step below.

## Files Created/Modified

### Per-file diff table (24 vendored source files)

| File | +/- | Kind | Note |
|---|---|---|---|
| `button/button.svelte` | 5/5 | **Visual** + internal | `secondary` variant hover changed from `hover:bg-secondary/80` (opacity blend) to `hover:bg-[color-mix(in_oklch,var(--secondary),var(--foreground)_5%)]`; base/outline/ghost/destructive class strings reordered only, no set change |
| `command/command.svelte` | 1/1 | Internal | Class-string reordering only |
| `command/command-group.svelte` | 1/1 | Internal | Class-string reordering only |
| `command/command-input.svelte` | 1/1 | Internal | Class-string reordering only |
| `command/command-item.svelte` | 2/2 | Internal | Class-string reordering; `group-has-[[data-slot=command-shortcut]]` normalized to Tailwind v4 shorthand `group-has-data-[slot=command-shortcut]` (same selector); dropped unused `cn-command-item-indicator` marker class (confirmed unreferenced elsewhere in the repo) |
| `command/command-link-item.svelte` | 1/1 | **Visual/behavioral** | Selected-state styling switched from `aria-selected:bg-accent aria-selected:text-accent-foreground` (accent-color scheme, distinct icon color rule) to the identical `data-selected:bg-muted data-selected:text-foreground ... group/command-item` class string `command-item.svelte` uses — the two components are now visually consistent; also drops the `[&_svg:not([class*='text-'])]:text-muted-foreground` icon rule |
| `command/command-separator.svelte` | 1/1 | Internal | Class-string reordering only |
| `command/command-shortcut.svelte` | 1/1 | Internal | Class-string reordering only |
| `dialog/dialog-content.svelte` | 1/1 | Internal | Class-string reordering; `max-w-[calc(100%-2rem)]` normalized to v4 syntax `max-w-[calc(100%_-_2rem)]` (identical computed value) |
| `dialog/dialog-description.svelte` | 1/1 | Internal | Class-string reordering only |
| `dialog/dialog-overlay.svelte` | 1/1 | Internal | Class-string reordering only |
| `input/input.svelte` | 2/2 | Internal | Class-string reordering only (both `<input>` elements) |
| `input-group/input-group.svelte` | 1/1 | Internal | Class-string reordering only |
| `input-group/input-group-addon.svelte` | 1/1 | Internal | Class-string reordering; `[calc(var(--radius)-5px)]` normalized to v4 syntax `[calc(var(--radius)_-_5px)]` |
| `input-group/input-group-button.svelte` | 3/3 | Internal | Class-string reordering; `[calc(var(--radius)-5px)]` v4 syntax normalization (x2); dropped unused `cn-input-group-button-size-sm` marker class (confirmed unreferenced elsewhere) |
| `input-group/input-group-text.svelte` | 1/1 | Internal | Class-string reordering only |
| `table/table-caption.svelte` | 1/1 | Internal | Class-string reordering only |
| `table/table-footer.svelte` | 1/1 | Internal | Class-string reordering only |
| `table/table-head.svelte` | 1/1 | Internal | Class-string reordering only |
| `table/table-row.svelte` | 6/1 | **Visual (additive)** | New `has-aria-expanded:bg-muted/50` utility added (muted background when the row contains an `aria-expanded` element); markup reformatted to multiple attribute lines, no other class change; inert unless a row contains an aria-expanded child |
| `tabs/tabs.svelte` | 1/1 | Internal | `data-[orientation=horizontal]:flex-col` normalized to v4 shorthand `data-horizontal:flex-col` (same selector) |
| `tabs/tabs-list.svelte` | 3/3 | Internal | `group-data-[orientation=vertical]` normalized to v4 shorthand `group-data-vertical` (x2, same selector); dropped unused `cn-tabs-list-variant-default`/`cn-tabs-list-variant-line` marker classes (confirmed unreferenced elsewhere) |
| `tabs/tabs-trigger.svelte` | 2/2 | Internal | Class-string reordering; `calc(100%-1px)` -> `calc(100%_-_1px)` and `group-data-[orientation=...]` -> `group-data-horizontal`/`group-data-vertical` v4 syntax normalization (x4, same selectors) |
| `textarea/textarea.svelte` | 1/1 | Internal | Class-string reordering only |

**Visual/API changes named:** `button.svelte` (secondary hover color-mix), `command-link-item.svelte` (selected-state color scheme now matches `command-item.svelte`), `table-row.svelte` (new additive `has-aria-expanded:bg-muted/50` utility). All three verified functionally safe: `pnpm check`, `pnpm vitest run` (585 passed), and the live Chromium `check:graph-console`/`breadcrumb-check.mjs` gates against the rebuilt binary all pass with zero errors.

**Byte-identical families:** none — all 8 families had at least one differing file (`command` family fully touched, 7/7 files).

### `web/build/**` (21 files, rebuilt bundle)

Rebuilt via `task web:build`; `task web:drift` confirms both the SOURCE half (119 files, digest match) and OUTPUT half (36 files, digest match) against the freshly-regenerated `.build-manifest`. Chunk file names changed (content-hashed by Vite) due to the upstream component source changes; no manual edits.

## Decisions Made

- Regenerated directly in the real `web/` tree rather than a scratch copy, since the plan's own D-09 fallback rule permits a local run whenever `cd web && pnpm --version` prints `11.23.0` — confirmed on this machine via pnpm's own `packageManager` self-management (no Corepack installed or required).
- Observed and documented (not fixed — out of scope, pre-existing Taskfile target behavior) that the `pnpm dlx shadcn-svelte@1.5.1 add ...` step itself runs under bare pnpm `v12.4.1` rather than the pinned `11.23.0`, because `pnpm dlx` resolves its own version independently of the project's `packageManager` field the way a direct `pnpm install`/`pnpm check` invocation does. This did not undermine the exercise: the differing-file set produced was byte-for-byte identical (as a set) to the Corepack-pinned CI run from 2026-09-14, which is strong empirical evidence the drift is registry-side, not an artifact of which pnpm version drives the CLI's own internal install step.
- Reverted the CLI's incidental `@lucide/svelte` caret-range bump (`^1.34.0` -> `^1.46.0`) in `web/package.json`/`web/pnpm-lock.yaml` — not a reviewed, intentional dependency change.
- Reverted `corpora/graph-console-check.json`'s regenerated timestamp/port/sha diff after running the `check:graph-console` gate — that file is gate-run output outside this plan's declared file scope; its per-run values would be a meaningless commit diff.

## Deviations from Plan

None — plan executed exactly as written. The `pnpm dlx` version observation above is a factual note about pre-existing Taskfile target behavior (not a bug introduced by this plan, and not something this plan's scope calls for changing — `web:components:drift`'s own precondition already accepts "corepack OR pnpm on PATH" and this plan did not touch that target), not a deviation requiring a fix.

## Issues Encountered

None.

## TDD Gate Compliance

Not applicable in the RED/GREEN/REFACTOR sense: this plan's `type` is `execute`, not `tdd`, and its work is a regeneration of vendored third-party component source plus a mechanical rebuild — not behavior-adding application code. The pre-existing RED is the drift monitor itself: Task 1 proved `task web:components:drift` fails (exit 201, 24 differing files) before any change; Task 2's re-vendor turns the same target GREEN (PASS, 50/50 byte-identical). No separate `test(...)` commit was required or produced.

## Known Stubs

None.

## Threat Flags

None — this plan's changes stay within the `<threat_model>` boundaries already declared in the plan (registry-sourced component code, CLI dependency-install side effect, regenerated SPA in the embedded binary); no new network endpoint, auth path, file access pattern, or schema change was introduced.

## User Setup Required

None.

## Self-Check: PASSED

- FOUND: `web/src/lib/components/ui/button/button.svelte`
- FOUND: `web/build/.build-manifest`
- FOUND: `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-05-SUMMARY.md`
- FOUND: commit `75f5633c` in `git log --oneline --all`
