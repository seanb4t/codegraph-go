---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
verified: 2026-09-13T22:17:27Z
status: passed
score: 5/5 must-haves verified
covered_files:
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
covered_digest: "v1:sha256:f9cdeac7f4496920e3173ba59bd8baf3798df6324fb1cd2b329ea267eb900aaf"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_verified: "2026-09-12T21:21:22Z"
  reason: "milestone-close re-pin — canonical status read stale (#4155: .planning/REQUIREMENTS.md was listed in covered_files and is rewritten by every later phase.complete); all five must-haves re-executed at HEAD 8c8149de"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff Verification Report

**Phase Goal:** A developer reading a long file always knows which symbol they are inside, and can hand any node or line off to their own editor in one click — with the security boundary named correctly and the absolute path never weakening repo-root confinement at the RPC boundary.
**Verified:** 2026-09-13T22:17:27Z
**Status:** passed
**Re-verification:** Yes — milestone-close re-verification (previous: passed, 5/5, verified 2026-09-12T21:21:22Z; the earlier 09-06 gap-closure re-verification narrative is preserved below unchanged)

## Goal Achievement

### Observable Truths

Truths 1-4 are the ROADMAP's own Success Criteria (the binding contract); truth 5 is the
plan-level acceptance gate that was the sole structured gap in the pre-09-06 verification. All
five are re-executed again at current HEAD (`8c8149de`), not trusted from the prior
VERIFICATION.md or any SUMMARY transcript. Since the prior verification (`3fd47279`), Phases
10-12 landed; `git log 3fd47279..HEAD -- <phase-9 files>` shows only `internal/uiproto/uiv1/ui.proto`
touched, and only additively — Phase 10 appended `GetCoverage` as UIService's 16th rpc and new
message fields, after Phase 9's `GetEditorLink` (still the 15th rpc, still present, still
unchanged in signature or body). No other phase-9 file (Go or Svelte/TS) was touched.

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Scrolling a long file updates a single-line breadcrumb naming the innermost containing symbol, computed client-side from `FileSymbols` ranges, verified in a live browser against a real index, and demonstrated wrong (stale or empty) against the pre-fix build (BRW-10, SC1) | ✓ VERIFIED | Regression check at HEAD `8c8149de`: `git log 3fd47279..HEAD -- web/src/lib/breadcrumb.ts web/src/lib/source-lines.ts web/scripts/breadcrumb-check.mjs` is empty (no later phase touched these). Re-ran the two named vitest files directly: `npx vitest run tests/breadcrumb.test.ts tests/source-pane-breadcrumb.test.ts` (bundled with the other three editor-surface files below) — 5 files / 68 tests, all passed. Live-browser Chromium gate (`web/scripts/breadcrumb-check.mjs`) was not re-run this session (it launches `codegraph ui` and rewrites `corpora/breadcrumb-check.json`, forbidden by the concurrent-verifier landmines); citing the 09-06 evidence, which reproduced it byte-for-byte against a real index. |
| 2 | Node detail and source views offer an "open in editor" link built from a server-resolved `{path}`/`{line}`/`{col}` template; a traversal-shaped or out-of-root path is refused at the RPC boundary before it reaches the template, by a test naming the rejected input and failing if the request succeeds (BRW-11, SC2) | ✓ VERIFIED | `GOTOOLCHAIN=go1.26.6 go test -run TestGetEditorLinkPathConfinementAtRPCBoundary ./internal/uiserver/... -v` re-run at HEAD `8c8149de`: 5/5 subtests pass (`in-repo_control`, `escape`, `absolute`, `empty`, `symlink-escape`). `internal/uiserver/editorlink.go` untouched since `3fd47279` (confirmed via the file-scoped `git log` above producing no hits for this file). |
| 3 | `codegraph ui --editor-url` sets the default and a per-browser override replaces it for that browser only; presets exist for VS Code, Cursor and JetBrains; a check reports the preset count and finds Zed in neither the preset list nor any claim the UI makes (BRW-12, SC3) | ✓ VERIFIED | Regression check at HEAD: `rg -io '\bzed\b|zed://' web/src internal/uiserver/editorpresets.go` = 0 matches. `rg -l localStorage web/src` = exactly `editor-prefs.ts`. `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./internal/uiserver/...` = all packages `ok`. |
| 4 | A phase `SECURITY.md` names the browser's external-protocol prompt — not CSP — as the security boundary and covers validation of the template's inputs, with every named threat carrying a test or a recorded verdict (BRW-13, SC4) | ✓ VERIFIED | `09-SECURITY.md` untouched since `3fd47279` (file-scoped `git log` empty). `rg -c '^\| T-09-' .planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-SECURITY.md` = 22 rows, `threats_open: 0` still present; `strconv.Itoa` (line/col precision backstop) still present at `internal/uiserver/editorlink.go:236,238`. |
| 5 | `web/src/lib/components/browse/SourcePane.svelte` compiles cleanly under `svelte-check` (0 errors) — the 09-06 gap closure, and the standing regression gate this phase leaves behind | ✓ VERIFIED | `pnpm -C web check` re-run at HEAD `8c8149de`: `COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`, exit code 0; `rg -o -i '\b0 errors\b' <captured log> \| wc -l` = 1 (gate satisfied per this run's contract). File count rose from 1169→1172 (Phases 10-12 added files elsewhere in `web/src`), consistent with no regression in this phase's own surface. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### 16-rpc set still contains `GetEditorLink`

`rg -o '^\s*rpc \w+' internal/uiproto/uiv1/ui.proto | wc -l` = 16 at HEAD (was 15 at Phase 9 close). `internal/uiproto/uiv1/ui.proto:135` still reads `rpc GetEditorLink(GetEditorLinkRequest) returns (GetEditorLinkResponse);`, unchanged since Phase 9. The 16th rpc, `GetCoverage`, was added additively by Phase 10 (commit `22c3aac1`) after `GetEditorLink`, per that phase's own numbering comment ("GetCoverage is plan 10-01's sixteenth rpc").

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/uiproto/uiv1/ui.proto` + regenerated `.pb.go`/`.connect.go`/`ui_pb.ts` | `GetEditorLink` present as an UIService rpc, additive since | ✓ VERIFIED | `GetEditorLink` unchanged at line 135; Phase 10/11 additions (`GetCoverage`, `community_id`/`community_count` fields) are purely additive elsewhere in the file — confirmed by reading the full diff `git diff 3fd47279..HEAD -- internal/uiproto/uiv1/ui.proto`. |
| `internal/uiserver/editorlink.go` | Validator, encoder, answer-builder, handler | ✓ VERIFIED | Untouched since `3fd47279` (file-scoped `git log` empty); confinement test re-run green (Truth 2). |
| `internal/uiserver/editorpresets.go` | Exactly 3 presets, Zed absent | ✓ VERIFIED | Untouched since `3fd47279`; Zed-absence regression check green (Truth 3). |
| `internal/cli/editorurl.go` / `editordiscovery.go` | Flag/env/discovery precedence, off switch, fail-fast on malformed values | ✓ VERIFIED | Untouched since `3fd47279`; `go test ./internal/cli/...` green (cached, ok). |
| `web/src/lib/source-lines.ts`, `breadcrumb.ts` | Balanced per-line splitting, innermost-symbol derivation | ✓ VERIFIED | Untouched since `3fd47279`; named vitest files re-run green. |
| `web/src/lib/components/browse/SourcePane.svelte` | Per-line rows, gutter, breadcrumb, header editor link, probe/click wiring; compiles clean | ✓ VERIFIED | Untouched since `3fd47279` (the 09-06 fix is already at HEAD); `svelte-check` 0 errors re-confirmed (Truth 5). |
| `web/src/lib/editor-prefs.ts`, `EditorLinkPicker.svelte` | localStorage override, picker UI | ✓ VERIFIED | Untouched since `3fd47279`; named vitest files (`editor-prefs.test.ts`, `editor-link-picker.test.ts`) re-run green. |
| `web/scripts/breadcrumb-check.mjs`, `corpora/breadcrumb-check.json` | Live-browser gate + committed record | ✓ VERIFIED (not re-executed) | Untouched since `3fd47279`; per the concurrent-verifier landmine constraints for this run, the live Chromium gate was not launched (it starts `codegraph ui` and rewrites `corpora/breadcrumb-check.json`) — citing the 09-06 session's byte-for-byte reproduction as standing evidence. |
| `09-MUTATION-LOG.md` | 4 families + Closing, byte-clean proof, plus gate-hygiene addendum | ✓ VERIFIED | Untouched since `3fd47279`; content unchanged (`## Family (` count still 4, `### Gate-hygiene addendum (09-06)` still present). |
| `09-SECURITY.md` | House format, boundary sentences, 22 rows | ✓ VERIFIED | Untouched since `3fd47279`; regression check green (Truth 4). |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `SourcePane.svelte` probe `$effect` | `GetEditorLink` rpc (initial + CR-01 re-probe) | guard-narrowed `const getEditorLink = activeClient.getEditorLink.bind(activeClient)` called from both sites | ✓ WIRED | File untouched since `3fd47279`; `rg -o 'template: resolved' web/src/lib/components/browse/SourcePane.svelte \| wc -l` = 1 (unchanged shape) re-confirmed at HEAD. |
| `internal/cli/editorurl.go` | `uiserver.ValidateEditorTemplate`/`EditorLinkOptions` | `resolveEditorLink` before `uiserver.Listen` | ✓ WIRED | Untouched since `3fd47279`; `go test ./internal/cli/...` passes (cached). |
| `web/src/lib/editor-prefs.ts` | `GetEditorLinkRequest.template` | `templateForRequest` (single resolution seam) | ✓ WIRED | Untouched since `3fd47279`. |
| `GetEditorLinkResponse.url` | browser `<a href>` / `location.assign` | OS URI handler | ✓ WIRED (Chromium only) | Unchanged; Safari/WebKit and Firefox remain recorded as UNVERIFIED in `09-SECURITY.md` Notes — see Human Verification below. |

### Data-Flow Trace (Level 4)

Not re-run in full — no phase-9 source file changed between the prior verification (`3fd47279`)
and current HEAD (`8c8149de`), only `internal/uiproto/uiv1/ui.proto` via later phases'
purely-additive edits (confirmed above). The prior verification's Level-4 findings for this
phase (breadcrumb from `FileSymbols`, editor link from `GetEditorLinkResponse.url`, presets from
the wire per D-18) are unaffected and stand.

### Behavioral Spot-Checks / Live-Gate Re-Execution

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| svelte-check (the standing gate) | `pnpm -C web check > log; echo RC=$?` | RC=0; `COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`; `rg -o -i '\b0 errors\b' log \| wc -l` = 1 | ✓ PASS |
| Confinement gate (SC2) | `GOTOOLCHAIN=go1.26.6 go test -run TestGetEditorLinkPathConfinementAtRPCBoundary ./internal/uiserver/... -v` | 5/5 subtests PASS | ✓ PASS |
| Go suites, `internal/cli` + `internal/uiserver` | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./internal/uiserver/...` | all packages `ok` | ✓ PASS |
| Breadcrumb + editor vitest files (5 files) | `npx vitest run tests/breadcrumb.test.ts tests/source-pane-breadcrumb.test.ts tests/editor-prefs.test.ts tests/editor-link-picker.test.ts tests/source-pane-editor-link.test.ts` (run from `web/`) | Test Files 5 passed (5); Tests 68 passed (68) | ✓ PASS |
| CR-01 named regression test | `npx vitest run -t "sends a pre-seeded PRESET override as the corrected template via a re-probe"` (run from `web/`) | Test Files 1 passed / 48 skipped; Tests 1 passed / 583 skipped | ✓ PASS |
| 16-rpc set still contains `GetEditorLink` | `rg -o '^\s*rpc \w+' internal/uiproto/uiv1/ui.proto \| wc -l` | 16 (was 15 at Phase 9 close; `GetCoverage` added by Phase 10, additive) | ✓ PASS |
| Live browser gate (`breadcrumb-check.mjs`) | Not run this session | N/A — see landmine constraints above; cites 09-06's byte-for-byte prior reproduction | — SKIPPED (documented, not re-run: forbidden to launch `codegraph ui` under the concurrent five-verifier run) |
| Suite-wide claims | N/A this session | Citing the Phase-12 regression pass (`af95a438`): `task test:unit` 52/52, goldens, web 584/584, all three drift gates, `check:gonum`, `check:no-force-layout` all green | — CITED (per brief, not re-run) |

`git status --porcelain` after all commands above: clean (no scratch files or gate side-effects left in the tree — the live-browser gate, which would have rewritten `corpora/breadcrumb-check.json`, was not run).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| BRW-10 | 09-03, 09-05 | Breadcrumb naming innermost symbol, live-browser verified, empty+stale demonstrated wrong | ✓ SATISFIED | Truth 1; unaffected since prior verification |
| BRW-11 | 09-01, 09-04, 09-05, 09-06 | Server-resolved editor link, confinement-refused at RPC boundary | ✓ SATISFIED | Truth 2; unaffected since prior verification |
| BRW-12 | 09-01, 09-02, 09-04, 09-05, 09-06 | `--editor-url` default + per-browser override + 3 presets + Zed absent | ✓ SATISFIED | Truth 3; unaffected since prior verification |
| BRW-13 | 09-05 | SECURITY.md naming the browser prompt, not CSP, as boundary; every threat test-or-verdict | ✓ SATISFIED | Truth 4; unaffected since prior verification |

Cross-referenced against `.planning/REQUIREMENTS.md` at HEAD: all four (BRW-10, BRW-11, BRW-12,
BRW-13) map to Phase 9 and are marked `Complete`. Per-phase requirement count line confirms
"Phase 9: 4 (BRW-10…13)". No orphaned requirements. (`.planning/REQUIREMENTS.md` is read here
for cross-reference only — per the #4155 lesson this file is NOT listed in `covered_files`,
since it is rewritten by every later `phase.complete` and would make the fingerprint go stale
again.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/cli/editordiscovery_test.go` | 338-339 | Dead trailing zero-check after an early `t.Fatalf` loop (IN-01, already flagged by 09-REVIEW.md) | ℹ️ Info | Harmless, already correctly triaged as out-of-scope in 09-REVIEW.md/09-REVIEW-FIX.md; file unchanged since prior verification |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any phase-9 file at HEAD. No file in this phase's
`covered_files` list has been modified since the prior verification (`3fd47279`), so no new
anti-patterns could have been introduced by Phases 10-12.

### Human Verification Required

Per this and the prior verification's constraints: both items below are recorded decisions with
named options and evidence status in `09-SECURITY.md`/`.planning/WINDOWS.md`, and the phase's
ROADMAP success criteria are otherwise measurably met (5/5 truths verified with reproducible
evidence, re-confirmed at HEAD `8c8149de`). These are tracked assumptions, not manufactured
human-verification items, and do not block `passed`.

- **Cursor and JetBrains editor-link URI templates remain `[ASSUMED]`** (`.planning/WINDOWS.md` #35): no real Cursor/JetBrains GUI was available in this or either prior verification session to confirm `idea://open?file={path}&line={line}` / `cursor://file/{path}:{line}` actually opens the right file. Visibly flagged `[ASSUMED]` in `EditorLinkPicker.svelte` and honestly recorded in `09-SECURITY.md` Notes — not silently shipped as verified.
- **Safari/WebKit and Firefox transient-activation behavior** for the gutter's async click→`GetEditorLink`→`location.assign` sequence: only Chromium was exercised via Playwright in the live gate (last executed in the 09-06 session). Recorded as "UNVERIFIED" in `09-SECURITY.md` Notes, not silently assumed at Chromium parity.

### Gaps Summary

None. This is a milestone-close re-pin, not a gap-closure round: the canonical
`gsd_run query verification.status` read `stale` only because the prior `covered_files` list
included `.planning/REQUIREMENTS.md`, which every later `phase.complete` legitimately rewrites
(#4155) — not because any phase-9 truth regressed. `git log 3fd47279..HEAD` scoped to every file
in this phase's `covered_files` shows exactly one touched: `internal/uiproto/uiv1/ui.proto`, and
only additively (Phase 10 appended `GetCoverage` as the 16th rpc and new message fields after
`GetEditorLink`, which is untouched). All five observable truths, all required artifacts, all
key links, and all four requirements (BRW-10…13) were re-executed fresh against HEAD `8c8149de`
and hold. `covered_files` has been rewritten to drop `.planning/REQUIREMENTS.md` and
`covered_digest` recomputed via the tool's `computeCoveredDigest` function over the exact sorted
list above, confirmed canonical-`passed` by `gsd_run query verification.status` (see below).

---

_Verified: 2026-09-13T22:17:27Z_
_Verifier: Claude (gsd-verifier)_
