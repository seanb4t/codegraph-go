---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 06
subsystem: ui
tags: [svelte, typescript, gate-hygiene, gap-closure]
gap_closure: true

# Dependency graph
requires:
  - phase: 09-04
    provides: SourcePane.svelte's EditorLinkClient probe effect and the CR-01 corrective re-probe (commit 28d5d795)
  - phase: 09-05
    provides: 09-MUTATION-LOG.md's Closing section and the phase-close gate run this addendum extends
provides:
  - "SourcePane.svelte compiles cleanly under strict TypeScript / svelte-check (0 errors) — the gate 09-VERIFICATION.md found broken"
  - "09-MUTATION-LOG.md ## Closing → ### Gate-hygiene addendum (09-06), a dated record of the silently-red gate window and the forward rule"
affects: []

# Actuals (#2632)
actuals:
  tokens: 1388
  tasks: 2
  commits: 2
plan_head_before: 5d832ba5e28e41b2187384f6a12aa5f929454893

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Capture a guard-narrowed method into a const local (bound to its receiver) before entering an async closure, rather than re-guarding inside the closure — TypeScript's property-access narrowing does not cross a closure boundary, but a const narrowed to a defined function type does."

key-files:
  created: []
  modified:
    - web/src/lib/components/browse/SourcePane.svelte
    - web/build/.build-manifest
    - web/build/_app/immutable/chunks/6PLdMuaE.js
    - web/build/_app/immutable/chunks/BQ7GelJt.js
    - web/build/_app/immutable/chunks/CEosfH05.js
    - web/build/_app/immutable/entry/app.Z9DUbmu6.js
    - web/build/_app/immutable/entry/start.DajafOI3.js
    - web/build/_app/immutable/nodes/0.DzmiGvqV.js
    - web/build/_app/immutable/nodes/1.CvUvSDTD.js
    - web/build/_app/immutable/nodes/3.DAZ4Gy29.js
    - web/build/_app/immutable/nodes/6.CtWH2EtG.js
    - web/build/_app/version.json
    - web/build/index.html
    - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md

key-decisions:
  - "Used the `.bind(activeClient)` capture form named as the plan's primary option (over the alternative hoist-above-the-guard form) — costs nothing, keeps this-binding semantics for any future client implementation, and the two call-site edits are identical either way."

patterns-established:
  - "A closure that calls an optional method captured from a guard-narrowed variable must capture the METHOD itself into a const local before the closure, not re-check the guard inside the closure — the re-check form adds an unreachable branch no test covers, which this plan's own action explicitly rejected as strictly larger than the capture."

requirements-completed: [BRW-11, BRW-12]

coverage:
  - id: D1
    description: "SourcePane.svelte compiles cleanly under svelte-check (0 errors) with the guard-narrowed getEditorLink captured once and called from both the initial probe and the CR-01 re-probe"
    requirement: BRW-11
    verification:
      - kind: other
        ref: "pnpm -C web check (RED: exit 1, 2 errors at 446:31; GREEN: exit 0, 0 errors)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The CR-01 corrective re-probe's behavior and its regression test are unchanged — byte-identical test file, template: resolved appears exactly once, D-11 scope guard holds"
    requirement: "BRW-11, BRW-12"
    verification:
      - kind: unit
        ref: "web/tests/source-pane-editor-link.test.ts#sends a pre-seeded PRESET override as the corrected template via a re-probe (CR-01)"
        status: pass
      - kind: unit
        ref: "four-file vitest gate (source-pane-editor-link, source-pane-breadcrumb, source-pane, browse-page): 34/34 pass"
        status: pass
      - kind: other
        ref: "task web:test: 542/542 passed"
        status: pass
    human_judgment: false
  - id: D3
    description: "The committed SPA is rebuilt and drift-clean in the same commit as the source change"
    requirement: "BRW-11, BRW-12"
    verification:
      - kind: other
        ref: "task web:build && task web:drift"
        status: pass
    human_judgment: false
  - id: D4
    description: "The svelte-check gate gap and its silent window are recorded in a phase artifact"
    requirement: null
    verification:
      - kind: other
        ref: "09-MUTATION-LOG.md ## Closing → ### Gate-hygiene addendum (09-06)"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 6: Gap Closure — svelte-check Gate Hygiene Summary

**Closed the one structured gap from `09-VERIFICATION.md`: captured the guard-narrowed `getEditorLink` once so TypeScript's narrowing survives the CR-01 re-probe's async closure, restoring `pnpm -C web check` to 0 errors with zero behavioral change, and recorded the gate's silent red window in `09-MUTATION-LOG.md`.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-12T21:15:00Z
- **Completed:** 2026-09-12T21:35:00Z
- **Tasks:** 2
- **Files modified:** 14 (1 source file + 12 rebuilt `web/build/**` artifacts + 1 phase doc)

## Accomplishments

- `SourcePane.svelte`'s probe `$effect` now declares `const getEditorLink = activeClient.getEditorLink.bind(activeClient)` immediately after the existing top-of-body guard, and both the initial probe call and the CR-01 corrective re-probe call this local instead of re-reading `activeClient.getEditorLink` — TypeScript's narrowing of the guard now survives into the async `.then` closure.
- `pnpm -C web check` (svelte-check) restored to 0 errors at HEAD; the committed SPA rebuilt via `task web:build` and re-verified drift-clean via `task web:drift`.
- `09-MUTATION-LOG.md`'s `## Closing` section gained a dated `### Gate-hygiene addendum (09-06)` naming the introducing commit (`28d5d795`), the verified-red HEAD (`3c0f4bc7`), the fix commit (`1e92d0d6`), why the phase-close gate list and code-review re-verification both missed it, and the forward rule for future web-touching phases.

## Task Commits

1. **Task 1: Capture the guard-narrowed getEditorLink once so the CR-01 re-probe type-checks, rebuild, drift-clean** — `1e92d0d6` (fix)
2. **Task 2: Record the silently-red svelte-check gate in 09-MUTATION-LOG.md's Closing section** — `5865d91` (docs)

## RED / GREEN Transcripts

### RED (before the edit, at starting HEAD `5d832ba5`)

```
1789247247766 START ".../codegraph-go/web"
1789247247769 ERROR "src/lib/components/browse/SourcePane.svelte" 446:31 "Cannot invoke an object which is possibly 'undefined'."
1789247247769 ERROR "src/lib/components/browse/SourcePane.svelte" 446:31 "'activeClient.getEditorLink' is possibly 'undefined'."
1789247247769 COMPLETED 1169 FILES 2 ERRORS 0 WARNINGS 1 FILES_WITH_PROBLEMS
[ELIFECYCLE] Command failed with exit code 1.
```

### GREEN (after the edit)

```
$ svelte-kit sync && svelte-check --tsconfig ./tsconfig.json
1789247274048 START ".../codegraph-go/web"
1789247274051 COMPLETED 1169 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS
```

### Four-file vitest gate and `task web:test`

```
total 34 passed 34 failed 0   (source-pane-editor-link, source-pane-breadcrumb, source-pane, browse-page)
web:test: observed numTotalTests=542 numPassedTests=542 (vitest exit 0)
web:test: PASS — 542 of 542 tests passed
```

### `task web:build && task web:drift`

```
web:drift: hashed 114 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (114 files, fa0d79480ce1f3849faabeec63c72fe12227ab0ffaa7a9c6a2b8bfc815d3125e)
web:drift: output half MATCH (32 files, d772f7b81e4f6bb8c316f7707c0eed884f24dcf4841296bcc61796530e08856b)
web:drift: PASS — hashed 114 source files, manifested 32 output files, committed web/build/ matches both digests
```

## Exact Diff Hunk (SourcePane.svelte)

```diff
@@ -418,6 +418,13 @@
 			editorLinkState = { kind: 'idle' };
 			return;
 		}
+		// getEditorLink (09-06): capture the guard-narrowed method once,
+		// bound to activeClient, so TypeScript's narrowing from the guard
+		// above survives into the async .then closure below — property-
+		// access narrowing does not cross a closure boundary, but a const
+		// local narrowed to a defined function type does. Both the initial
+		// probe and the CR-01 corrective re-probe call this same local.
+		const getEditorLink = activeClient.getEditorLink.bind(activeClient);
 		editorLinkState = { kind: 'loading' };
 		const controller = new AbortController();
 		// untrack: lastPresets is $state and this effect's own .then below
@@ -425,8 +432,7 @@
 		// re-trigger this same effect (the fileSymbolsCache pitfall 09-03
 		// already documented and fixed the same way).
 		const template = templateForRequest(override, untrack(() => lastPresets));
-		activeClient
-			.getEditorLink({ path, line, col, template }, { signal: controller.signal })
+		getEditorLink({ path, line, col, template }, { signal: controller.signal })
 			.then(async (response) => {
 				lastPresets = response.presets;
 				// Corrective re-probe (CR-01): a pending PRESET override
@@ -443,7 +449,7 @@
 				if (template === undefined && override?.kind === 'preset') {
 					const resolved = templateForRequest(override, response.presets);
 					if (resolved !== undefined) {
-						const corrected = await activeClient.getEditorLink(
+						const corrected = await getEditorLink(
 							{ path, line, col, template: resolved },
 							{ signal: controller.signal }
 						);
```

Exactly a capture line plus two call-site substitutions, as the plan required. No new imports, no type assertions, no non-null operators, no suppression comments.

## Files Created/Modified

- `web/src/lib/components/browse/SourcePane.svelte` — the capture + two call-site substitutions above
- `web/build/.build-manifest` and the 12 changed/renamed/added content-hashed chunk files under `web/build/_app/**` plus `web/build/_app/version.json` and `web/build/index.html` — rebuilt via `task web:build`, verified via `task web:drift`
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md` — one new `### Gate-hygiene addendum (09-06)` subsection under the existing `## Closing` heading

## Decisions Made

See `key-decisions` in frontmatter. Used the plan's primary `.bind(activeClient)` capture form (over the hoist-above-the-guard alternative it also offered) — both are equivalent in effect and call-site edits; the bind form was chosen as written.

## Deviations from Plan

None — plan executed exactly as written. Task's own Step 0 RED gate reproduced the gap exactly as `09-VERIFICATION.md` recorded it (exit 1, exactly two `ERROR` lines at `SourcePane.svelte" 446:31`) before any edit was made, and the fix was the smallest control-flow-preserving change the plan specified.

## Issues Encountered

**Task's own `task web:build`/`task web:drift` gate literal expects exactly 2 `half MATCH` occurrences and 1 `PASS` line when the command is run non-silently** (`task web:build && task web:drift > file`). In this execution environment, `task` (v3.52.0) echoes each Taskfile command's full multi-line shell script to stdout before running it (no `silent: true` is set on `web:build`/`web:drift` in `Taskfile.yml`), so the literal strings `half MATCH` and `web:drift: PASS` inside the echoed `echo "..."` script lines match the same `rg` pattern as the real output lines, doubling the count (4 and 2 respectively) and failing the gate's exact-count assertion even though the underlying build and drift check both genuinely passed. Re-running the identical commands with `task -s` (task's own documented silent flag, which suppresses the command echo but not its stdout/stderr) produced output matching the gate's exact expected counts (2 `half MATCH`, 1 `PASS`) with byte-identical semantic content. Used `task -s web:build && task -s web:drift` for the final commit-preceding run; the committed `web/build/.build-manifest` and `web/build/**` reflect that final silent run. Not logged as a code deviation (no source file changed to work around this) — it is an environment/tooling observation about this Taskfile's default verbosity versus the plan's assumed quiet output, left here for visibility rather than filed to WINDOWS.md since it affects gate *transcript formatting* only, never the pass/fail verdict itself.

**Separately observed (not a regression, not fixed):** two consecutive `task web:build` runs with no source changes between them produced different content-hash filenames for several chunks (e.g. `entry/start.*.js`, `entry/app.*.js`, several `nodes/*.js` and `chunks/*.js`) and therefore different `output-sha256` digests — Vite/Rollup's content-hashing for this project's chunk graph is not perfectly reproducible byte-for-byte across separate build invocations. This does not affect `web:drift`'s correctness within a single build+drift pass (the manifest is always written by the same build that produced the files it describes), and is out of this gap-closure plan's scope; noted here only because it was directly observed while diagnosing the `half MATCH` count above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The phase's own repeatedly-declared "0 svelte-check errors" acceptance bar (09-03-PLAN.md Task 2, 09-04-PLAN.md Tasks 2-3) is restored and green at HEAD.
- `09-VERIFICATION.md`'s single structured gap is closed; re-running its Truth 5 check (`pnpm -C web check`) now reports 0 errors, exit 0.
- BRW-11 and BRW-12 keep their previously-verified status: no behavioral change, D-11 scope guard, `location.assign` count, and no-`window.open` invariants all still hold.
- No blockers.

## Self-Check: PASSED

- `web/src/lib/components/browse/SourcePane.svelte` present and containing the capture line: confirmed via `rg`.
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md` contains the new `### Gate-hygiene addendum (09-06)` heading exactly once, `## Family (` count still 4, `## Closing` count still 1: confirmed via `rg`.
- Both commit hashes (`1e92d0d6`, `5865d91`) verified present via `git log --oneline`.
- `pnpm -C web check` (0 errors), `task web:test` (542/542), and `task web:drift` (both halves MATCH, PASS) re-confirmed green at HEAD immediately before writing this SUMMARY.

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*
