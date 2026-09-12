---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
verified: 2026-09-12T21:21:22Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - .planning/REQUIREMENTS.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-01-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-01-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-02-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-02-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-03-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-03-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-04-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-04-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-05-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-05-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-06-PLAN.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-06-SUMMARY.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-CONTEXT.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-MUTATION-LOG.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-RESEARCH.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-REVIEW-FIX.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-REVIEW.md
  - .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-SECURITY.md
  - internal/cli/editordiscovery.go
  - internal/cli/editordiscovery_test.go
  - internal/cli/editorurl.go
  - internal/cli/editorurl_test.go
  - internal/cli/ui.go
  - internal/cli/ui_test.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/editorlink.go
  - internal/uiserver/editorlink_test.go
  - internal/uiserver/editorpresets.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/server.go
  - web/scripts/breadcrumb-check.mjs
  - web/src/lib/breadcrumb.ts
  - web/src/lib/components/browse/EditorLinkPicker.svelte
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/lib/editor-prefs.ts
  - web/src/lib/source-lines.ts
covered_digest: "v1:sha256:938f3ec91c55cd42550ce0b7e43c13f7a7b92f265eb7cafc9a5574be162ad56d"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "SourcePane.svelte's svelte-check gate (2 errors at 446:31, from CR-01 commit 28d5d795 through HEAD 3c0f4bc7) — fixed by commit 1e92d0d6, which captures the guard-narrowed `getEditorLink` once into a const bound to `activeClient` immediately after the effect's top-of-body guard, so TypeScript's narrowing survives into the async `.then` closure for both the initial probe and the CR-01 corrective re-probe. `pnpm -C web check` now reports 0 ERRORS, 0 WARNINGS, exit 0, re-confirmed twice in this session."
  gaps_remaining: []
  regressions: []
---

# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff Verification Report

**Phase Goal:** A developer reading a long file always knows which symbol they are inside, and can hand any node or line off to their own editor in one click — with the security boundary named correctly and the absolute path never weakening repo-root confinement at the RPC boundary.
**Verified:** 2026-09-12T21:21:22Z
**Status:** passed
**Re-verification:** Yes — after gap closure (09-06)

## Goal Achievement

### Observable Truths

Truths 1-4 are the ROADMAP's own Success Criteria (the binding contract); truth 5 is the
plan-level acceptance gate that was the sole structured gap in the prior verification
(3c0f4bc7). All five are re-executed at HEAD (`3fd47279`), not trusted from any SUMMARY
transcript.

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Scrolling a long file updates a single-line breadcrumb naming the innermost containing symbol, computed client-side from `FileSymbols` ranges, verified in a live browser against a real index, and demonstrated wrong (stale or empty) against the pre-fix build (BRW-10, SC1) | ✓ VERIFIED | Regression check: `09-MUTATION-LOG.md` families (a)/(b) unchanged since prior verification (no source touched by 09-06 affects breadcrumb.ts, source-lines.ts, or breadcrumb-check.mjs). `task web:test` re-run at HEAD: 542/542 pass, including the breadcrumb-specific test files. |
| 2 | Node detail and source views offer an "open in editor" link built from a server-resolved `{path}`/`{line}`/`{col}` template; a traversal-shaped or out-of-root path is refused at the RPC boundary before it reaches the template, by a test naming the rejected input and failing if the request succeeds (BRW-11, SC2) | ✓ VERIFIED | `GOTOOLCHAIN=go1.26.6 go test -run TestGetEditorLinkPathConfinementAtRPCBoundary ./internal/uiserver/... -v` re-run at HEAD: 5/5 subtests pass (`in-repo_control`, `escape`, `absolute`, `empty`, `symlink-escape`). `internal/uiserver/editorlink.go` untouched by 09-06 (only `web/src/lib/components/browse/SourcePane.svelte` and docs changed). |
| 3 | `codegraph ui --editor-url` sets the default and a per-browser override replaces it for that browser only; presets exist for VS Code, Cursor and JetBrains; a check reports the preset count and finds Zed in neither the preset list nor any claim the UI makes (BRW-12, SC3) | ✓ VERIFIED | Regression check at HEAD: `rg -io '\bzed\b\|zed://' web/src internal/uiserver/editorpresets.go` = 0 matches. `rg -l localStorage web/src` = exactly `editor-prefs.ts`. `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/...` = all packages ok (cached, unaffected by 09-06's SourcePane-only diff). |
| 4 | A phase `SECURITY.md` names the browser's external-protocol prompt — not CSP — as the security boundary and covers validation of the template's inputs, with every named threat carrying a test or a recorded verdict (BRW-13, SC4) | ✓ VERIFIED | Regression check: `09-SECURITY.md` untouched by 09-06. `rg -c '^\| T-09-'` = 22 rows; `threats_open: 0` confirmed; `strconv.Itoa` (the line/col precision backstop) still present at `internal/uiserver/editorlink.go:236,238`. |
| 5 | `web/src/lib/components/browse/SourcePane.svelte` compiles cleanly under `svelte-check` (0 errors) — an explicit, repeated `<verify>`/`<verification>` gate in 09-03-PLAN.md Task 2 and 09-04-PLAN.md Tasks 2-3, and the prior verification's sole structured gap | ✓ VERIFIED | `pnpm -C web check` re-run twice at HEAD (`3fd47279`): `COMPLETED 1169 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`, exit 0. Fix commit `1e92d0d6` captures `const getEditorLink = activeClient.getEditorLink.bind(activeClient)` immediately after the effect's top-of-body guard; both the initial probe and the CR-01 corrective re-probe call this local. Diff confirmed by reading `web/src/lib/components/browse/SourcePane.svelte:418-455` directly, not from the SUMMARY's pasted transcript alone. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/uiproto/uiv1/ui.proto` + regenerated `.pb.go`/`.connect.go`/`ui_pb.ts` | `GetEditorLink` as UIService's 15th rpc, additive | ✓ VERIFIED | Unchanged by 09-06; `task proto:drift` not re-run this session (no proto/generated-file diff in 09-06's commits — confirmed via `git show 1e92d0d6 5865d918 3fd47279 --stat`). |
| `internal/uiserver/editorlink.go` | Validator, encoder, answer-builder, handler | ✓ VERIFIED | Untouched by 09-06 (`git diff 3c0f4bc7..3fd47279 -- internal/uiserver/editorlink.go` is empty); confinement test re-run green (Truth 2). |
| `internal/uiserver/editorpresets.go` | Exactly 3 presets, Zed absent | ✓ VERIFIED | Untouched by 09-06; Zed-absence regression check green (Truth 3). |
| `internal/cli/editorurl.go` / `editordiscovery.go` | Flag/env/discovery precedence, off switch, fail-fast on malformed values | ✓ VERIFIED | Untouched by 09-06; `go test ./internal/cli/...` green. |
| `web/src/lib/source-lines.ts`, `breadcrumb.ts` | Balanced per-line splitting, innermost-symbol derivation | ✓ VERIFIED | Untouched by 09-06; `task web:test` 542/542. |
| `web/src/lib/components/browse/SourcePane.svelte` | Per-line rows, gutter, breadcrumb, header editor link, probe/click wiring; compiles clean | ✓ VERIFIED | Gap closed — see Truth 5. `svelte-check` 0 errors; CR-01 regression test still passes; `location.assign` semantics and D-11 scope guard unchanged (only the capture/call-site edit landed). |
| `web/src/lib/editor-prefs.ts`, `EditorLinkPicker.svelte` | localStorage override, picker UI | ✓ VERIFIED | Untouched by 09-06. |
| `web/scripts/breadcrumb-check.mjs`, `corpora/breadcrumb-check.json` | Live-browser gate + committed record | ✓ VERIFIED | Untouched by 09-06; not re-run this session per verification constraints (optional — prior verification reproduced it byte-for-byte and 09-06 touched only a type capture with no behavioral surface). |
| `09-MUTATION-LOG.md` | 4 families + Closing, byte-clean proof, plus new gate-hygiene addendum | ✓ VERIFIED | `## Closing` → `### Gate-hygiene addendum (09-06)` present, dated, names introducing commit (`28d5d795`), verified-red HEAD (`3c0f4bc7`), fix commit (`1e92d0d6`), root cause, and a forward rule. `## Family (` count still 4. |
| `09-SECURITY.md` | House format, boundary sentences, 22 rows | ✓ VERIFIED | Untouched by 09-06; regression check green (Truth 4). |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `SourcePane.svelte` probe `$effect` | `GetEditorLink` rpc (initial + CR-01 re-probe) | guard-narrowed `const getEditorLink = activeClient.getEditorLink.bind(activeClient)` called from both sites | ✓ WIRED | Read directly from `SourcePane.svelte` (lines ~418-455): the capture line sits immediately after the existing `!activeClient?.getEditorLink` guard; the original `activeClient.getEditorLink(...)` call and the CR-01 `activeClient.getEditorLink(...)` call inside `.then` are both replaced with `getEditorLink(...)`. `rg -o 'template: resolved' web/src/lib/components/browse/SourcePane.svelte \| wc -l` = 1 (unchanged shape). |
| `internal/cli/editorurl.go` | `uiserver.ValidateEditorTemplate`/`EditorLinkOptions` | `resolveEditorLink` before `uiserver.Listen` | ✓ WIRED | Untouched by 09-06; `TestUICommandRefusesMalformedEditorURLBeforeBinding` passes (cached, unaffected). |
| `web/src/lib/editor-prefs.ts` | `GetEditorLinkRequest.template` | `templateForRequest` (single resolution seam) | ✓ WIRED | Untouched by 09-06. |
| `GetEditorLinkResponse.url` | browser `<a href>` / `location.assign` | OS URI handler | ✓ WIRED (Chromium only) | Unchanged; Safari/WebKit and Firefox remain recorded as UNVERIFIED in `09-SECURITY.md` Notes — see Human Verification below. |

### Data-Flow Trace (Level 4)

Not re-run in full — the 09-06 diff is confined to a closure-capture refactor with no new data source or rendering path (confirmed by reading the diff hunk in `09-06-SUMMARY.md` against the live file). The prior verification's Level-4 findings for this phase (breadcrumb from `FileSymbols`, editor link from `GetEditorLinkResponse.url`, presets from the wire per D-18) are unaffected and stand.

### Behavioral Spot-Checks / Live-Gate Re-Execution

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| svelte-check (the gap) | `pnpm -C web check` | `COMPLETED 1169 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`, exit 0 (re-run twice) | ✓ PASS |
| Web drift | `task -s web:drift` | source half MATCH (114 files), output half MATCH (32 files), PASS | ✓ PASS |
| Full web test suite | `task web:test` | 542/542 passed | ✓ PASS |
| CR-01 named regression test | `npx vitest run -t "sends a pre-seeded PRESET override as the corrected template via a re-probe"` | 1 passed, 541 skipped (targeted run) | ✓ PASS |
| Confinement gate (SC2) | `GOTOOLCHAIN=go1.26.6 go test -run TestGetEditorLinkPathConfinementAtRPCBoundary ./internal/uiserver/... -v` | 5/5 subtests PASS | ✓ PASS |
| Go suites, `internal/cli` + `internal/uiserver` | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./internal/uiserver/...` | all packages `ok` | ✓ PASS |
| CR-01 re-probe shape unchanged | `rg -o 'template: resolved' web/src/lib/components/browse/SourcePane.svelte \| wc -l` | 1 | ✓ PASS |
| Live browser gate | `node web/scripts/breadcrumb-check.mjs` | SKIPPED (optional per verification constraints; prior verification reproduced it byte-for-byte and 09-06 touched only a type capture) | — SKIPPED (documented) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| BRW-10 | 09-03, 09-05 | Breadcrumb naming innermost symbol, live-browser verified, empty+stale demonstrated wrong | ✓ SATISFIED | Truth 1; unaffected by 09-06 |
| BRW-11 | 09-01, 09-04, 09-05, 09-06 | Server-resolved editor link, confinement-refused at RPC boundary | ✓ SATISFIED | Truth 2; 09-06 closed the svelte-check gate on this requirement's UI surface with no behavioral change |
| BRW-12 | 09-01, 09-02, 09-04, 09-05, 09-06 | `--editor-url` default + per-browser override + 3 presets + Zed absent | ✓ SATISFIED | Truth 3; 09-06 closed the svelte-check gate on this requirement's UI surface with no behavioral change |
| BRW-13 | 09-05 | SECURITY.md naming the browser prompt, not CSP, as boundary; every threat test-or-verdict | ✓ SATISFIED | Truth 4; unaffected by 09-06 |

Cross-referenced against `.planning/REQUIREMENTS.md`: all four (BRW-10, BRW-11, BRW-12, BRW-13) map to Phase 9 and are marked `Complete`. No orphaned requirements — REQUIREMENTS.md's Phase 9 total (4) matches the plans' declared `requirements:` fields exactly.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/cli/editordiscovery_test.go` | 338-339 | Dead trailing zero-check after an early `t.Fatalf` loop (IN-01, already flagged by 09-REVIEW.md) | ℹ️ Info | Harmless, already correctly triaged as out-of-scope in 09-REVIEW.md/09-REVIEW-FIX.md |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file touched by 09-06 or the phase overall. The previously-open anti-pattern (the svelte-check gate break at `SourcePane.svelte:446`) is resolved — the diff is exactly a capture line plus two call-site substitutions, no suppression comments, no `@ts-ignore`, no non-null assertions.

### Human Verification Required

Per this verification's constraints: both items below are recorded decisions with named options
and evidence status in `09-SECURITY.md`/`WINDOWS.md`, and the phase's ROADMAP success criteria
are otherwise measurably met (5/5 truths verified with reproducible evidence). These are tracked
assumptions, not manufactured human-verification items, and do not block `passed`.

- **Cursor and JetBrains editor-link URI templates remain `[ASSUMED]`** (`.planning/WINDOWS.md` #35): no real Cursor/JetBrains GUI was available in this or the prior verification session to confirm `idea://open?file={path}&line={line}` / `cursor://file/{path}:{line}` actually opens the right file. Visibly flagged `[ASSUMED]` in `EditorLinkPicker.svelte` and honestly recorded in `09-SECURITY.md` Notes — not silently shipped as verified.
- **Safari/WebKit and Firefox transient-activation behavior** for the gutter's async click→`GetEditorLink`→`location.assign` sequence: only Chromium was exercised via Playwright in the live gate. Recorded as "UNVERIFIED" in `09-SECURITY.md` Notes, not silently assumed at Chromium parity.

### Gaps Summary

None. The single structured gap from the prior verification (`SourcePane.svelte` failing
`svelte-check` with 2 errors at line 446:31) is closed by 09-06's commit `1e92d0d6`: the
guard-narrowed `getEditorLink` is captured once into a `const` bound to `activeClient`
immediately after the effect's existing top-of-body guard, and both the initial probe and the
CR-01 corrective re-probe call that local instead of re-reading the possibly-undefined property
inside the async closure. `pnpm -C web check` now reports 0 errors at HEAD (re-confirmed twice
in this session), `task -s web:drift` reports both halves MATCH, `task web:test` reports
542/542, the CR-01 regression test passes unchanged, and the gap's silent window
(commit `28d5d795` through `3c0f4bc7`) plus a forward rule are now recorded in
`09-MUTATION-LOG.md`'s `## Closing → ### Gate-hygiene addendum (09-06)`. All four ROADMAP
success criteria remain independently verified with reproducible RED/GREEN mutation evidence,
unaffected by this narrowly-scoped fix.

---

_Verified: 2026-09-12T21:21:22Z_
_Verifier: Claude (gsd-verifier)_
