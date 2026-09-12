---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
plan: 05
subsystem: security
tags: [mutation-testing, threat-model, security-doc, playwright, path-confinement, uri-scheme-allowlist]

# Dependency graph
requires:
  - phase: 09-source-view-follow-through-breadcrumb-editor-handoff
    provides: "plans 09-01..09-04's shipped code and every plan's own <threat_model> block, the union of which this plan's register closes over"
provides:
  - "09-MUTATION-LOG.md families (b)-(d) and a Closing section: the breadcrumb's STALE half watched fail live, the confinement gate's rejected-path test watched fail, and the scheme allowlist's poisoning watched fail — each confirmed applied, byte-cleanly reverted, re-verified green"
  - "09-SECURITY.md: house-format threat register (22 rows, union of all five plans), naming the browser's external-protocol prompt — not CSP — as the security boundary in two verbatim sentences"
  - "A green phase-close transcript (proto:drift, web:drift, test:unit, web:test, live gate, preset-count test) plus an honestly-recorded pre-existing web:components:drift failure (WINDOWS.md #31, out of scope)"
affects: []

# Actuals (#2632)
actuals:
  tokens: 9875
  tasks: 3
  commits: 3
plan_head_before: 4ceedd4bd45a78b335e902688484795942868c9e

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mutation-log families targeted at a DIFFERENT --file than the script's own default when the default's geometry cannot expose the specific bug (the live gate's own early-break condition made internal/query/node.go incapable of demonstrating the nearest-preceding-symbol lie; internal/cli/editorurl.go's short, clustered symbols could)"
    - "A mutation that deletes a confinement call must also remove its now-solely-supporting import in the SAME diff, or the RED demonstration degrades into a compile error instead of the intended assertion failure"

key-files:
  created:
    - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-SECURITY.md
  modified:
    - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md

key-decisions:
  - "Family (b)'s live-gate run was retargeted at --file internal/cli/editorurl.go instead of the script's default internal/query/node.go: the default file's structure makes the scroll loop's own break condition (>=1 empty AND >=1 non-empty ground-truth observation) fire the instant the loop first enters resolveSourcePath's wide 33-79 range, before it can ever reach the post-symbol gap (80-90) where the nearest-preceding lie would surface. editorurl.go's two single-line symbols (lines 22, 23) followed by a real gap let the scroll skip clean over both, landing at line 26 where the oracle says null but the mutated function reports the second single-line symbol — exposing the bug live, in a real Chromium session."
  - "Family (c)'s mutation also removed the now-unused io/fs import (its sole use was fs.ErrNotExist inside the deleted confinement block) so the package still compiles — an unused-import build error would have masked the intended assertion failures behind a compile error."
  - "Cursor and JetBrains preset templates are recorded in 09-SECURITY.md as still [ASSUMED] rather than confirmed: no real Cursor/JetBrains installation was available to visually click through and confirm correct navigation in this autonomous session (a JetBrains Toolbox CLI script being on PATH is not the same as the GUI click-through the human-check requires). WINDOWS.md entry 35 stays open rather than being closed on an unperformed check."

requirements-completed: [BRW-10, BRW-11, BRW-13]

coverage:
  - id: D1
    description: "09-MUTATION-LOG.md families (b), (c), (d): the stale breadcrumb, the bypassed confinement gate, and the poisoned scheme allowlist each watched fail on the specific assertion that exists for it, byte-cleanly reverted"
    requirement: "BRW-10, BRW-11, BRW-13"
    verification:
      - kind: e2e
        ref: "09-MUTATION-LOG.md#Family (b) — breadcrumb-check.mjs --file internal/cli/editorurl.go, line 26: shown=noEditorURLEnvVar, expected=null, agrees=false, success=false"
        status: pass
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestGetEditorLinkPathConfinementAtRPCBoundary (RED transcript in 09-MUTATION-LOG.md#Family (c))"
        status: pass
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestEditorTemplateSchemeAllowlist + #TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors (RED transcript in 09-MUTATION-LOG.md#Family (d))"
        status: pass
    human_judgment: false
  - id: D2
    description: "09-MUTATION-LOG.md closes with a phase-wide byte-clean proof, a non-vacuity assertion, and a family-to-requirement table covering all four families (a)-(d)"
    requirement: "BRW-13"
    verification:
      - kind: other
        ref: "09-MUTATION-LOG.md#Closing — git diff --quiet across all three mutated paths exits 0, git status --porcelain empty"
        status: pass
    human_judgment: false
  - id: D3
    description: "09-SECURITY.md exists in 08-SECURITY.md's house format, states verbatim that CSP does not govern href navigation and the browser's external-protocol prompt is the consent boundary, with 22 threat rows (>=15 required) each carrying a test or a Verdict, every high-severity row mitigate+test, threats_open:0"
    requirement: "BRW-13"
    verification:
      - kind: other
        ref: "09-SECURITY.md — this plan's own Task 2 <verify> gates (boundary sentences, section presence, row count/shape, high-row gate, cited-test resolution) all passed"
        status: pass
    human_judgment: false
  - id: D4
    description: "Cursor and JetBrains preset templates' real-world correctness (file/line navigation, no project-selection prompt) against an actually-installed IDE"
    verification: []
    human_judgment: true
    rationale: "Requires a human to visually click through a real Cursor or JetBrains installation and confirm correct navigation — no GUI/interactive session was available to this autonomous executor. Recorded honestly as unverified (09-SECURITY.md Notes item 1); WINDOWS.md entry 35 stays open pending that human check."
  - id: D5
    description: "Phase-close gate: proto:drift, web:drift, test:unit, web:test, a fresh live-gate run, and the preset-count test all green on a clean tree at HEAD"
    requirement: "BRW-10, BRW-11, BRW-12, BRW-13"
    verification:
      - kind: integration
        ref: "task proto:drift (4 generated files byte-identical)"
        status: pass
      - kind: integration
        ref: "task web:drift (114 source files / 32 output files, both digests MATCH)"
        status: pass
      - kind: unit
        ref: "task test:unit (all Go packages ok)"
        status: pass
      - kind: unit
        ref: "task web:test (539/539 passed)"
        status: pass
      - kind: e2e
        ref: "node web/scripts/breadcrumb-check.mjs (fresh run, success:true, gutterLinkCount:418, gutterClickIssuedRpc:true)"
        status: pass
      - kind: unit
        ref: "internal/uiserver/editorlink_test.go#TestEditorPresetsAreExactlyThreeAndNameNoZed"
        status: pass
    human_judgment: false
  - id: D6
    description: "web:components:drift's failure is pre-existing, already-tracked (WINDOWS.md #31), and confirmed unrelated to any file this plan touched"
    verification:
      - kind: other
        ref: "git log -1 shows the 14 failing files' last touch predates this plan; git diff 4ceedd4b..HEAD touches no file under web/src/lib/components/ui/"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-12
status: complete
---

# Phase 9 Plan 5: Mutation-Log Close and SECURITY.md Summary

**Three more RED demonstrations watched fail live (a stale breadcrumb caught in a real Chromium session, a bypassed confinement gate, a poisoned scheme allowlist), byte-cleanly reverted, and `09-SECURITY.md` written naming the browser's external-protocol prompt — not CSP — as the boundary, with all 22 threats each carrying a test or a recorded verdict.**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-09-12T19:08:47Z (approx., tree state at plan start)
- **Completed:** 2026-09-12T19:36:20Z
- **Tasks:** 3 (all `type="auto"`)
- **Files modified:** 1 created (`09-SECURITY.md`), 1 modified (`09-MUTATION-LOG.md`); 2 production files (`web/src/lib/breadcrumb.ts`, `internal/uiserver/editorlink.go`) touched-and-byte-cleanly-reverted three times, never committed

## Accomplishments

- **Family (b) — BRW-10 STALE half:** `innermostSymbolAt` mutated to return, among symbols with `startLine <= line`, the one with the greatest `startLine` (the nearest-preceding-symbol lie D-03 forbids). The script's default target (`internal/query/node.go`) cannot expose this mutation — its own scroll-loop break condition fires the instant it first enters the file's one wide symbol range, before ever reaching a post-symbol gap. Retargeted at `internal/cli/editorurl.go` (two single-line symbols followed by a real gap): the live gate, in a real Chromium session, disagreed with its own independent oracle at line 26 (`shown:"noEditorURLEnvVar"`, `expected:null`, `agrees:false`, `success:false`, exit 1). Reverted byte-clean (plus manual cleanup of Vite's content-hashed untracked chunk files `git checkout --` cannot remove), rebuilt, re-verified green against the committed record.
- **Family (c) — BRW-11 SC2:** `eng.ValidateRepoRelativePath(path)` and its error-handling block deleted from `GetEditorLink`; the now-solely-supporting `"io/fs"` import removed in the same mutation so the package still compiled. `TestGetEditorLinkPathConfinementAtRPCBoundary` failed on all four rejection cases (`escape`, `absolute`, `empty`, `symlink-escape`) with "succeeded, want a refusal", while `in-repo_control` kept passing. Reverted, re-verified green.
- **Family (d) — BRW-11/BRW-12 D-13:** `"javascript": {}` added to `editorURLSchemes` (15→16). The count guard (`TestEditorTemplateSchemeAllowlist`) failed first via its own `t.Fatalf`, which short-circuited that test's own membership subtest before it could run — recorded honestly rather than papered over. The membership guard's defeat was independently proven by `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors/template_invalid_via_override`, which uses the exact `javascript:{path}` override and failed with `availability = BUILDABLE, want TEMPLATE_INVALID`. Reverted, re-verified green (both tests, all subtests).
- `09-MUTATION-LOG.md` closed with a phase-wide byte-clean proof (`git diff --quiet` across all three mutated paths, `git status --porcelain` empty), a non-vacuity assertion, and a family-to-requirement table covering all four families (a)-(d).
- `09-SECURITY.md` written in `08-SECURITY.md`'s exact house format: Trust Boundaries, a Threat Register with 22 rows (union of all five plans' `<threat_model>` blocks, deduplicated by id), an Accepted Risks Log (5 rows), Notes (Cursor/JetBrains assumptions, transient-activation, shell-out verdict), a Security Audit Trail, and Sign-Off. Both required boundary sentences appear verbatim. All five `high`-severity rows are `mitigate` with a real, `rg`-resolved Go test. `threats_open: 0`.
- Phase-close gate run: `proto:drift`, `web:drift`, `test:unit`, `web:test` (539/539), a fresh live-gate run, and `TestEditorPresetsAreExactlyThreeAndNameNoZed` all green on a clean tree. `web:components:drift` failed for a pre-existing, already-tracked, out-of-scope reason (see Deviations).

## Task Commits

Each task was committed atomically:

1. **Task 1: Families (b), (c), (d) — a stale breadcrumb, a bypassed confinement gate and a poisoned allowlist, each watched fail and byte-cleanly reverted** — `879f7519` (test)
2. **Task 2: `09-SECURITY.md` — the browser's external-protocol prompt named as the boundary, every threat with a test or a verdict** — `416b6955` (docs)
3. **Task 3: Phase close — every gate green on a clean tree, Zed nowhere, presets counted from the binary** — `3dd44e6a` (docs)

## Files Created/Modified

- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md` — families (b)-(d) appended, `## Closing` with the phase-wide proof, non-vacuity assertion, family-to-requirement table, and the phase-close line
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-SECURITY.md` — the phase's threat register, created new
- `web/src/lib/breadcrumb.ts`, `web/build/**` — touched by family (b)'s mutation, reverted byte-clean (never committed)
- `internal/uiserver/editorlink.go` — touched by families (c) and (d)'s mutations, one at a time, reverted byte-clean between and after (never committed)

## Decisions Made

See `key-decisions` in frontmatter. In short: family (b) needed a different `--file` target than the script's own default to expose the mutation at all, given the live gate's own early-break condition; family (c)'s mutation had to also drop an import to keep the diff compiling; and the Cursor/JetBrains preset templates are recorded as still `[ASSUMED]` rather than fabricated as confirmed, since no real IDE was available to click through in this session.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Family (b)'s default `breadcrumb-check.mjs` target cannot expose the STALE mutation**
- **Found during:** Task 1, first live-gate run against the mutated `innermostSymbolAt` using the script's default `internal/query/node.go`
- **Issue:** The script's scroll loop stops as soon as it has recorded >=1 empty-ground-truth AND >=1 non-empty-ground-truth observation (`observations.length >= 3 && nonEmptyObservations >= 1 && emptyObservations >= 1`). Against `internal/query/node.go`, the first non-empty ground truth appears at line 41, squarely inside `resolveSourcePath`'s wide 33-79 range — both the oracle and the (mutated) product code agree there, so the run completes GREEN without ever reaching the real post-symbol gap (lines 80-90) where the nearest-preceding lie would surface.
- **Fix:** Queried `FileSymbols` directly for several small files to find one whose structure would expose the bug within the loop's break budget, and retargeted the SAME script (unmodified) at `--file internal/cli/editorurl.go`: two single-line symbols (`editorURLEnvVar` line 22, `noEditorURLEnvVar` line 23) followed by a real gap before `discoveredEditor` (29) let a normal ~15-line scroll step skip clean over both, landing at line 26 where the oracle correctly says `null` but the mutated function reports `noEditorURLEnvVar` (the latest-starting symbol with `startLine <= 26`) — `agrees:false`, `success:false`, exit 1.
- **Files modified:** none (diagnostic-only; the mutation and the script itself were unchanged, only the `--file` argument to an already-existing, already-committed script)
- **Verification:** the RED transcript is pasted verbatim in `09-MUTATION-LOG.md`#Family (b); the GREEN re-run afterward reproduced the committed `corpora/breadcrumb-check.json` byte-for-byte using the script's default arguments.
- **Committed in:** `879f7519` (the MUTATION-LOG.md entry documents the reasoning inline; no code commit was needed).

**2. [Rule 3 - Blocking] Family (c)'s confinement-block deletion leaves an unused import**
- **Found during:** Task 1, applying family (c)'s mutation
- **Issue:** Deleting `eng.ValidateRepoRelativePath(path)` and its `if verr != nil { ... }` block also removes the only use of `fs.ErrNotExist`, leaving `"io/fs"` an unused import — which fails `go build` with a compile error, not the intended test-assertion failures. A RED demonstration that is actually a compile error, not an assertion failure, is exactly what this plan's own convention forbids.
- **Fix:** Removed the `"io/fs"` import in the SAME mutation diff. Confirmed `go build ./internal/uiserver/...` succeeded before running the target test.
- **Files modified:** `internal/uiserver/editorlink.go` (mutation only; reverted byte-clean with `git checkout --` before the task's commit — the file carries zero net diff)
- **Verification:** `go build` succeeded; `TestGetEditorLinkPathConfinementAtRPCBoundary` then failed on genuine assertion mismatches, pasted verbatim in `09-MUTATION-LOG.md`#Family (c).
- **Committed in:** n/a (mutation reverted before commit; documented in `879f7519`'s MUTATION-LOG.md entry).

---

**Total deviations:** 2 auto-fixed (1 Rule 1 — a live-gate targeting fix needed to actually observe the intended bug; 1 Rule 3 — a compile-blocking unused import in the mutation diff itself). **Impact on plan:** Zero production-behavior change; both fixes are diagnostic/mutation-authoring corrections needed to produce a genuine, non-vacuous RED demonstration rather than a false GREEN or a compile error. No scope creep.

## Issues Encountered

**`web:components:drift` fails for a pre-existing, already-tracked, out-of-scope reason.** During Task 3's phase-close run, `task web:components:drift` exited 1, naming 14 vendored shadcn-svelte files (`dialog-overlay.svelte`, `input-group*.svelte`, `input.svelte`, `table-*.svelte`, `tabs*.svelte`, `textarea.svelte`) as differing from a fresh `shadcn-svelte@1.5.1` regeneration. This is `.planning/WINDOWS.md` entry 31, opened 2026-09-08 by an earlier phase's validate-phase pass, hypothesized as a local pnpm/Corepack toolchain mismatch (this dev host lacks Corepack, so `pnpm dlx` falls back to a different pnpm major than the workflow's pinned version) rather than local tampering — the workflow itself is schedule/`workflow_dispatch`-only and deliberately excluded from `requiredCheckNames`, so it has never blocked a merge. Confirmed unrelated to this plan: `git diff 4ceedd4b..HEAD` (this plan's own base) touches no file under `web/src/lib/components/ui/`, and each failing file's last commit predates this phase entirely. Per the scope-boundary rule, this was not re-fixed here — it is already logged and tracked, and remains open.

## User Setup Required

None — no external service configuration required.

## Phase close

Full transcripts, run in order, on a clean tree at HEAD `416b69551068e40c9bb44fd550371fdcd0f809f8` (the `09-SECURITY.md` commit):

```
$ GOTOOLCHAIN=go1.26.6 task proto:drift
proto:drift: compared 4 generated files
proto:drift: all 4 generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)

$ task web:drift
web:drift: hashed 114 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (114 files, 8fcb4cc73a68fa889dd120bc70e08bc9429b75ee2b4bc8cb4182e070b3542184)
web:drift: output half MATCH (32 files, 2d10c251722be132e5d783dacd42294ba58fff109f45a95aa2dd7bada994954c)
web:drift: PASS — hashed 114 source files, manifested 32 output files, committed web/build/ matches both digests

$ task web:components:drift
::error::web:components:drift: web/src/lib/components/ui/dialog/dialog-overlay.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/input-group/input-group-addon.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/input-group/input-group-button.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/input-group/input-group-text.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/input-group/input-group.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/input/input.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/table/table-caption.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/table/table-footer.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/table/table-head.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/table/table-row.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/tabs/tabs-list.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/tabs/tabs-trigger.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/tabs/tabs.svelte differs from shadcn-svelte@1.5.1's regeneration
::error::web:components:drift: web/src/lib/components/ui/textarea/textarea.svelte differs from shadcn-svelte@1.5.1's regeneration
task: Failed to run task "web:components:drift": exit status 1
[PRE-EXISTING, TRACKED: .planning/WINDOWS.md entry 31 — see Issues Encountered above. Not caused by this plan; not re-fixed here.]

$ GOTOOLCHAIN=go1.26.6 task test:unit
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	34.373s
[... all other packages ok/cached ...]

$ task web:test
web:test: observed numTotalTests=539 numPassedTests=539 (vitest exit 0)
web:test: PASS — 539 of 539 tests passed

$ GOTOOLCHAIN=go1.26.6 task build:release && node web/scripts/breadcrumb-check.mjs --out "$TMPDIR/breadcrumb-close.json"
breadcrumb-check: oracle has 13 symbols for internal/query/node.go
breadcrumb-check: source-breadcrumb element present
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":1,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":11,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":26,"shown":null,"expected":null,"empty":true,"agrees":true}
breadcrumb-check: observation {"line":41,"shown":"resolveSourcePath","expected":"resolveSourcePath","empty":false,"agrees":true}
breadcrumb-check: editor-link href=vscode://file//Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:1:1
breadcrumb-check: gutterLinkCount=418
breadcrumb-check: gutterClickIssuedRpc=true
breadcrumb-check: wrote .../breadcrumb-close.json — success=true
{"success":true,"obs":5,"href":"vscode://file//Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:1:1","gutter":418,"click":true}
PASS

$ GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run TestEditorPresetsAreExactlyThreeAndNameNoZed ./internal/uiserver/...
--- PASS: TestEditorPresetsAreExactlyThreeAndNameNoZed (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	0.346s

$ test "$(printf 'Zed zed://x' | rg -o -i '\bzed\b|zed://' | wc -l)" = "2"                                    # positive control: PASS
$ test "$(rg -o -i '\bzed\b|zed://' web/src internal/uiserver/editorpresets.go | wc -l)" = "0"                # PASS — Zed nowhere
$ test "$(rg -o 'vscode://|cursor://|idea://' internal/uiserver/editorpresets.go | wc -l)" = "3"              # PASS — presets=3, from the file's own scheme literals
$ test "$(rg -o 'editor-preset-' web/src/lib/components/browse/EditorLinkPicker.svelte | wc -l)" -ge 1        # PASS — picker test-id prefix present

$ git status --porcelain
(empty)
```

`TestEditorPresetsAreExactlyThreeAndNameNoZed` (`internal/uiserver/editorlink_test.go:491-515`) asserts `len(presets)==3`, exact ID order (`vscode`/`cursor`/`jetbrains`), and `inspected==9` (3 presets × 3 fields each scanned for "zed") internally via `t.Fatalf` on any violation — it carries no `t.Logf`, so its PASS line is the full transcript; the `inspected 9 strings, presets=3` figures are the test's own enforced invariants, not printed output.

## Next Phase Readiness

- Phase 9 (Source View Follow-Through — Breadcrumb & Editor Handoff) is functionally and security-documentation complete: all five plans executed, `09-MUTATION-LOG.md` proves all four RED demonstration families this phase committed to, and `09-SECURITY.md` closes BRW-13 with `threats_open: 0`.
- **Two open items carry forward, both already tracked and non-blocking for phase completion:**
  - `.planning/WINDOWS.md` entry 35 (Cursor/JetBrains preset templates still `[ASSUMED]`, pending a real human click-through) — the picker's own UI carries the visible unverified note; no regression risk, purely a documentation/UX-trust item.
  - `.planning/WINDOWS.md` entry 31 (`web:components:drift` pre-existing failure) — unrelated to this phase's work, already excluded from required CI checks, tracked for a future toolchain investigation.
- No blockers. `.planning/STATE.md` should be advanced to reflect Phase 9 fully executed (5/5 plans) once this SUMMARY's state update runs.

## Self-Check: PASSED

- `09-SECURITY.md` present on disk and matches the frontmatter/section contract verified by this plan's own Task 2 `<verify>` gates (re-run above, all passed).
- `09-MUTATION-LOG.md` present on disk with 4 `## Family (` headings and a `## Closing` section (re-verified: `rg -c '^## Family \(' 09-MUTATION-LOG.md` = 4).
- All three commit hashes (`879f7519`, `416b6955`, `3dd44e6a`) found in `git log --oneline`.
- `git status --porcelain` empty; `git diff --quiet -- web/src/lib/breadcrumb.ts web/build internal/uiserver/editorlink.go` exits 0 at HEAD.
- No `codegraph ui` process left running (`pgrep -fl "codegraph ui"` returns nothing).

---
*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Completed: 2026-09-12*
