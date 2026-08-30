---
phase: 04-query-workbench-index-health
plan: 07
subsystem: supply-chain
tags: [shadcn-svelte, drift-guard, taskfile, github-actions, render-cost, vitest, web-build]

requires:
  - phase: 04-query-workbench-index-health
    provides: "04-01 (table vendored @1.5.1), 04-04 (tabs vendored @1.5.1), 04-05, 04-06 — all six frontend plans whose source changes invalidated the committed web/build/"
provides:
  - "task web:components:drift — pinned-CLI regeneration into scratch, disk-derived subject set, byte-compare, RED-proven twice"
  - ".github/workflows/components-drift.yml — schedule + workflow_dispatch invocation of the drift target, out of required CI"
  - "task web:render-cost — opt-in 1000-row render/sort measurement, never asserted inside task web:test"
  - "committed web/build/ rebuilt and matching source again — task web:drift GREEN"
  - "three folded todos closed with resolution records"
affects: ["05", "verify-work"]

actuals:
  tokens: 0
  tasks: 3
  commits: 0

tech-stack:
  added: []
  patterns:
    - "web:components:drift mirrors proto:drift's regenerate-into-scratch-and-byte-compare shape (never web:drift's manifest-hash shape)"
    - "opt-in wall-clock assertions behind an env var + dedicated Taskfile target, kept out of any PR-required job"

key-files:
  created:
    - .github/workflows/components-drift.yml
    - web/tests/data-table-render-cost.test.ts
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
    - web/build/

key-decisions:
  - "Task 1 live probe: all 8 vendored component families (including the six vendored via @latest in 03-06) reproduce byte-identically at shadcn-svelte@1.5.1 — no per-family exception or normalization needed."
  - "web:components:drift's comparison is scoped strictly to web/src/lib/components/ui/ — the CLI's own dependency-install step mutates package.json (a devDependency caret range) as a side effect, which the guard must never compare against."

requirements-completed: [WRK-04]

coverage:
  - id: D1
    description: "task web:components:drift regenerates every vendored component family from disk-derived subject set into scratch, byte-compares, reports counts before comparing, and has been watched fail two independent ways"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "task web:components:drift (manual run, recorded in SUMMARY)"
        status: pass
    human_judgment: false
  - id: D2
    description: ".github/workflows/components-drift.yml invokes the guard on schedule + workflow_dispatch only, with a bootstrap chain proven present and correctly ordered"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/... -run 'TestWorkflowRunBodiesInvokeTask|TestWorkflowFilePopulationMatchesDisk|TestInScopeJobsPopulationMatchesDisk'"
        status: pass
    human_judgment: false
  - id: D3
    description: "1000-row render/sort-toggle cost measured with median-of-five and a row-count positive control; wall-clock threshold opt-in only, never in task web:test"
    requirement: "WRK-04"
    verification:
      - kind: unit
        ref: "web/tests/data-table-render-cost.test.ts"
        status: pass
    human_judgment: false
  - id: D4
    description: "committed web/build/ rebuilt; task web:drift GREEN on both source and output halves"
    requirement: "WRK-04"
    verification:
      - kind: other
        ref: "task web:drift"
        status: pass
    human_judgment: false

duration: TBD
completed: 2026-08-30
status: complete
---

# Phase 4 Plan 07: Drift Guard, Render Cost, Bundle Rebuild Summary

**Closed the phase's three remaining obligations: a disk-derived byte-compare guard over vendored shadcn-svelte source (proven RED two independent ways and wired into a schedule-only workflow), a measurement-first render-cost record for the 1000-row workbench table, and a rebuilt committed `web/build/` that closes the RED window opened at 04-01.**

## Performance

- **Duration:** TBD
- **Started:** 2026-08-30T01:35:00Z
- **Tasks:** 3 completed
- **Files modified:** TBD

## Task 1: Determinism probe — recorded findings

**Read first:** `.planning/phases/04-query-workbench-index-health/04-RESEARCH.md` Open Question 2, `web/components.json`, 04-01-SUMMARY.md, 04-04-SUMMARY.md.

**Scratch-tree method used:** `rsync -a --exclude node_modules --exclude .svelte-kit --exclude build web/ "$SCRATCH/web/"` (an identical copy of `web/components.json` travels with the rest of the tree — no hand-typed copy), then `pnpm install --frozen-lockfile` inside the scratch tree (fast — 3.8s, mostly served from the local pnpm content-addressable store, no meaningful network cost) rather than a symlinked `node_modules`: pnpm's `[ERR_PNPM_UNSAFE_MODULES_DIR]` safety check refuses to operate through a `node_modules` symlink that resolves outside the scratch project root, so a real (store-hydrated) install is the only viable scratch shape — this is itself a finding Task 2's target design must carry (a symlink shortcut does not work; a real `pnpm install --frozen-lockfile` inside the scratch tree does, and is cheap because of the content-addressable store).

**Disk-derived family enumeration** (the exact command Task 2's target will also run):

```
git ls-files -- 'web/src/lib/components/ui/' | awk -F/ '{print $6}' | sort -u
```

Output: `button command dialog input input-group table tabs textarea` — **8 families**, matching the plan's own "not just this phase's two" framing.

**Trailing-slash counter-check** (T-04-29 evidence):

```
git ls-files -- 'web/src/lib/components/ui/*/' | wc -l   →  0
git ls-files -- 'web/src/lib/components/ui/' | wc -l      →  50
```

Confirmed live in this repository: the trailing-slash pathspec returns **zero** files; the no-slash form returns **50**. Per-family counts: button 2, command 12, dialog 11, input 2, input-group 7, table 9, tabs 5, textarea 2 (sums to 50).

### Question 1 — Is the output byte-identical to what is committed?

**Yes, for all 8 families, with zero differences.** Ran `pnpm dlx shadcn-svelte@1.5.1 add button command dialog input input-group table tabs textarea -y -o` inside the scratch tree, then `diff -rq` each committed family directory against its scratch-regenerated counterpart:

```
=== button ===
=== command ===
=== dialog ===
=== input ===
=== input-group ===
=== table ===
=== tabs ===
=== textarea ===
```

Every block is empty — `diff -rq` reports nothing when trees are identical. No import-path rewriting difference, no formatting difference, no trailing-newline difference, no `index.ts` ordering difference.

### Question 2 — Is it deterministic across two consecutive runs into two fresh scratch trees?

**Yes.** Built a second, fully independent scratch tree (`probe2`, its own `rsync`, its own `pnpm install --frozen-lockfile`, its own `pnpm dlx shadcn-svelte@1.5.1 add ... -y -o`) and diffed `probe1`'s regenerated output against `probe2`'s, family by family — all 8 `diff -rq` blocks empty. Two independently-installed, independently-regenerated scratch trees produce byte-identical component source.

### Question 3 — Does `-o` behave the same against a target directory that already contains DIFFERENT content as it does against an empty one?

**Yes.** Appended a deliberate corruption line to `probe1/web/src/lib/components/ui/button/button.svelte`, confirmed the diff showed the injected line, then re-ran `pnpm dlx shadcn-svelte@1.5.1 add button -y -o` over the now-dirty target. Result: the file was overwritten back to byte-identical with the committed canonical file. `-o` unconditionally overwrites regardless of the target's prior content — no divergence between the empty-scratch case the guard actually uses and a dirty-target case.

### Question 4 — Does the CLI mutate anything outside the component directory in the scratch tree?

**Yes — `package.json`.** `diff` of the scratch tree's `package.json` against the repo's showed exactly one line changed: `"@lucide/svelte": "^1.34.0"` (committed) became `"@lucide/svelte": "^1.35.0"` (scratch, post-`add`). The CLI's own "Installing dependencies with pnpm..." step re-resolves at least one existing devDependency's semver range as a side effect of vendoring `input-group` (which references a lucide icon). `components.json` and `web/src/app.css` were unchanged; a full `diff -rq` of `web/src` (excluding `node_modules`, `.svelte-kit`, and the 8 known component directories) showed no other file touched.

**Consequence for Task 2's design:** the guard's comparison MUST be scoped strictly to `web/src/lib/components/ui/` — never to `package.json` or `pnpm-lock.yaml` — both because that is the guard's actual mandate (T-04-27 is about component *source*, which no lockfile-based gate can see) and because the CLI's own side effect would otherwise make the guard fail every run for a reason that has nothing to do with drift.

### Question 5 — Per-family reproduction table

| Family | Vendored by | Original vendoring invocation | Reproduces at 1.5.1? |
|---|---|---|---|
| button | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| command | 03-06 | `pnpm dlx shadcn-svelte@latest add command popover` | ✅ byte-identical |
| dialog | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| input | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| input-group | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |
| table | 04-01 | `pnpm dlx shadcn-svelte@1.5.1 add table -y -o` | ✅ byte-identical |
| tabs | 04-04 | `pnpm dlx shadcn-svelte@1.5.1 add tabs -y -o` | ✅ byte-identical |
| textarea | 03-06 | `pnpm dlx shadcn-svelte@latest add ...` | ✅ byte-identical |

**Every one of the 8 disk-derived families reproduces byte-identically at `shadcn-svelte@1.5.1`** — including the 6 families 03-06 vendored via `@latest` at an earlier date, meaning `@latest` at that time already resolved to `1.5.1` (or the registry has not changed those component templates since). No family diverges.

### Branch taken

**Byte-identical and deterministic → Task 2 wires the byte-compare exactly as designed, with NO per-family exception and NO normalization.** The "likely HALT" outcome the plan flagged as probable did not occur; all evidence points at the straightforward design. `web/components.json` is copied verbatim as part of the whole-tree scratch copy (not hand-retyped), satisfying the read_first requirement that a scratch tree carry an identical copy of it.

No repository file outside `.planning/` was modified by this task; all scratch trees live under the session scratchpad directory (`/private/tmp/claude-...`) and were never enumerated with `git ls-files`.

<!-- gsd:write-continue -->
