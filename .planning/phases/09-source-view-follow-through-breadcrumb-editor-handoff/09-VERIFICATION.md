---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
verified: 2026-09-12T21:10:00Z
status: gaps_found
score: 4/5 must-haves verified
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
covered_digest: "v1:sha256:ebb7797574be61ba089561a5a88de3a1087a6ab14da2522ae3d800b4d127b3ce"
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "web/src/lib/components/browse/SourcePane.svelte compiles cleanly under strict TypeScript / svelte-check (0 errors), as every 09-03 and 09-04 plan `<verify>` gate required (`pnpm -C web check` / `svelte-check` summary reporting `0 errors`, and 09-04-PLAN.md's own `<verification>` section: '`pnpm check` reports 0 errors')"
    status: failed
    reason: "`pnpm -C web check` (svelte-check --tsconfig ./tsconfig.json) reports 2 errors at HEAD (3c0f4bc7), reproduced twice in this verification session. The error was introduced by commit 28d5d795 ('fix(09): CR-01 re-probe getEditorLink when a pending preset override cannot resolve until presets arrive') and was never caught afterward: 09-05's own phase-close gate list (proto:drift, web:drift, web:components:drift, test:unit, web:test, live gate, preset-count test) does not include `pnpm check`, and the code-review's re-verification passes (09-REVIEW.md iterations 2-3, which fixed WR-01..WR-05 on top of CR-01) re-ran `go build`, `go vet`, `go test`, `pnpm -C web test`, and `task web:drift` but never re-ran svelte-check. The 09-REVIEW.md 'clean, 0 critical, 0 warning' verdict and 09-04-SUMMARY.md's pasted 'COMPLETED 1169 FILES 0 ERRORS' transcript both predate the commit that broke this gate, so neither document is wrong on its own terms — but the gate is broken at HEAD right now and no phase artifact records it. Functionally this is very likely a TypeScript closure-narrowing false positive rather than a live bug (the effect's own top-of-body guard `!activeClient?.getEditorLink` already establishes the property is defined for the whole synchronous body, and `web/tests/source-pane-editor-link.test.ts`'s dedicated CR-01 test — 'sends a pre-seeded PRESET override as the corrected template via a re-probe' — passes), but the phase's own repeatedly-declared acceptance bar (0 svelte-check errors) is not met at HEAD, and the discrepancy was never surfaced in any SUMMARY, REVIEW, WINDOWS.md entry, or SECURITY.md note."
    artifacts:
      - path: "web/src/lib/components/browse/SourcePane.svelte"
        issue: "Line 446, inside the `.then(async (response) => { ... })` closure of the GetEditorLink probe effect: `activeClient.getEditorLink(...)` is called a second time (the CR-01 corrective re-probe) without a fresh guard inside the closure. TypeScript's control-flow narrowing of `activeClient?.getEditorLink` from the effect's top-of-body check does not survive into the nested async closure, so `svelte-check` reports 'Cannot invoke an object which is possibly undefined' / \"'activeClient.getEditorLink' is possibly 'undefined'\" twice at that line."
    missing:
      - "Re-guard the closure before the second call (e.g. `const getEditorLink = activeClient.getEditorLink; if (!getEditorLink) return;` captured once, or an explicit `if (!activeClient.getEditorLink) return;` re-check inside the `.then` callback) so the narrowing survives, then re-run `pnpm -C web check` and confirm `0 errors`."
      - "Rebuild and re-verify `task web:build && task web:drift` after the fix (a source change to SourcePane.svelte requires a `web/build` rebuild to stay drift-clean)."
      - "Record the fix and the fact this gate was silently broken between the CR-01 commit and phase close, so a future audit does not need to rediscover it."
---

# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff Verification Report

**Phase Goal:** A developer reading a long file always knows which symbol they are inside, and can hand any node or line off to their own editor in one click — with the security boundary named correctly and the absolute path never weakening repo-root confinement at the RPC boundary.
**Verified:** 2026-09-12T21:10:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Truths 1-4 are the ROADMAP's own Success Criteria (the binding contract); truth 5 is a
repeatedly-declared plan-level acceptance gate (`<verify>`/`<verification>` sections in
09-03-PLAN.md Task 2 and 09-04-PLAN.md Tasks 2-3) re-executed at HEAD per this verification's
instructions, not trusted from any SUMMARY transcript.

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Scrolling a long file updates a single-line breadcrumb naming the innermost containing symbol, computed client-side from `FileSymbols` ranges, verified in a live browser against a real index, and demonstrated wrong (stale or empty) against the pre-fix build (BRW-10, SC1) | ✓ VERIFIED | Re-ran `node web/scripts/breadcrumb-check.mjs` live against a real Chromium + this repo's own `.codegraph/store` index at HEAD: `source-breadcrumb` present, 5 observations, `success:true`, byte-identical to the committed `corpora/breadcrumb-check.json`. `09-MUTATION-LOG.md` family (a) pastes the pre-fix binary (`aafee950`) producing `breadcrumbPresent:false`, exit 1 (empty half); family (b) pastes a live-Chromium run against a mutated `innermostSymbolAt` disagreeing with its own independent oracle at a real gap line (`shown:"noEditorURLEnvVar"`, `expected:null`, `agrees:false`, exit 1 — stale half), then a byte-clean revert and green re-run. Both RED demonstrations are pasted verbatim, not narrated. |
| 2 | Node detail and source views offer an "open in editor" link built from a server-resolved `{path}`/`{line}`/`{col}` template; a traversal-shaped or out-of-root path is refused at the RPC boundary before it reaches the template, by a test naming the rejected input and failing if the request succeeds (BRW-11, SC2) | ✓ VERIFIED | `TestGetEditorLinkPathConfinementAtRPCBoundary` re-run at HEAD: 5/5 subtests pass (`in-repo_control`, `escape`, `absolute`, `empty`, `symlink-escape`). `09-MUTATION-LOG.md` family (c) pastes the confinement call deleted from `GetEditorLink`, all four rejection subtests failing with "succeeded, want a refusal" while the in-repo control kept passing, then a byte-clean revert and green re-run. Header link (`data-testid="editor-link"`) and gutter buttons (`gutterLinkCount:418`, real click→rpc observed) confirmed live in the re-run gate. |
| 3 | `codegraph ui --editor-url` sets the default and a per-browser override replaces it for that browser only; presets exist for VS Code, Cursor and JetBrains; a check reports the preset count and finds Zed in neither the preset list nor any claim the UI makes (BRW-12, SC3) | ✓ VERIFIED | `TestEditorPresetsAreExactlyThreeAndNameNoZed` passes. `rg -o -i '\bzed\b|zed://' web/src internal/uiserver/editorpresets.go` = 0 (positive control `Zed zed://x` = 2, confirming the pattern itself works). `rg -o 'vscode://|cursor://|idea://' internal/uiserver/editorpresets.go` = 3. `rg -l getEditorLink web/src` = exactly `SourcePane.svelte` + generated client (D-11 scope guard). `rg -l localStorage web/src` = exactly `editor-prefs.ts`. `TestResolveEditorLinkPrecedence`/`TestResolveEditorLinkOffSwitch`/`TestUICommandDiscoveredDefaultHasProvenance` (flag→env→discovered→unconfigured, off switch, WR-04's flag-short-circuits-malformed-env fix) all pass. `EditorLinkPicker.svelte` renders exactly the wire's 3 presets with a visible `[ASSUMED]` note under Cursor/JetBrains (WINDOWS.md #35, honestly still open). |
| 4 | A phase `SECURITY.md` names the browser's external-protocol prompt — not CSP — as the security boundary and covers validation of the template's inputs, with every named threat carrying a test or a recorded verdict (BRW-13, SC4) | ✓ VERIFIED | `09-SECURITY.md` contains both required verbatim sentences ("CSP does not govern href navigation..." / "The browser's external-protocol prompt is the consent boundary..."). 22 `T-09-*` rows, every row carries a `Test...`/`.test.ts` name or a `Verdict:` marker (0 rows lack both, 0 TBD cells — checked programmatically). All 5 `high`-severity rows (`T-09-01,02,05,07,21`) are `mitigate` with a resolvable test. Every cited `Test...` name resolves to a real `func Test...(` at HEAD (checked via `rg` across `internal/` and `web/`, 0 missing). `threats_open: 0`. |
| 5 | `web/src/lib/components/browse/SourcePane.svelte` compiles cleanly under `svelte-check` (0 errors) — an explicit, repeated `<verify>`/`<verification>` gate in 09-03-PLAN.md Task 2 and 09-04-PLAN.md Tasks 2-3 | ✗ FAILED | Re-ran `pnpm -C web check` twice at HEAD (3c0f4bc7): `COMPLETED 1169 FILES 2 ERRORS 0 WARNINGS 1 FILES_WITH_PROBLEMS`, exit 1. Both errors are at `SourcePane.svelte:446:31` ("Cannot invoke an object which is possibly 'undefined'" / "'activeClient.getEditorLink' is possibly 'undefined'"), inside the CR-01 corrective re-probe added by commit `28d5d795` — a code-review fix that landed after every plan's own GREEN evidence for this gate was captured. See Gaps Summary. |

**Score:** 4/5 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/uiproto/uiv1/ui.proto` + regenerated `.pb.go`/`.connect.go`/`ui_pb.ts` | `GetEditorLink` as UIService's 15th rpc, additive | ✓ VERIFIED | `task proto:drift` green (4 files byte-identical); `wantUIServiceMethods` contains `"GetEditorLink"`; `TestUIServiceMethodSetIsExactlyTheReadSet`/`TestUIProtoFieldNumbersAreStableAndUnique` pass. |
| `internal/uiserver/editorlink.go` | Validator, encoder, answer-builder, handler | ✓ VERIFIED | `strconv.Itoa` used for line/col (backstop truth confirmed by direct read); WR-01's unterminated-placeholder fix present and tested. |
| `internal/uiserver/editorpresets.go` | Exactly 3 presets, Zed absent | ✓ VERIFIED | `TestEditorPresetsAreExactlyThreeAndNameNoZed` passes; scheme-literal count = 3. |
| `internal/cli/editorurl.go` / `editordiscovery.go` | Flag/env/discovery precedence, off switch, fail-fast on malformed values | ✓ VERIFIED | All named tests pass (286 PASS, 0 FAIL across `./internal/cli/...`); WR-02's source-level no-spawn guard (`TestEditorDiscoverySourceNeverSpawnsAProcess`) present. |
| `web/src/lib/source-lines.ts`, `breadcrumb.ts` | Balanced per-line splitting, innermost-symbol derivation | ✓ VERIFIED | Unit tests pass; never-un-escapes property confirmed. |
| `web/src/lib/components/browse/SourcePane.svelte` | Per-line rows, gutter, breadcrumb, header editor link, probe/click wiring | ⚠️ VERIFIED (with a build-hygiene defect) | Functionally wired and behaviorally proven (vitest 542/542, live gate), but fails `svelte-check` — see Truth 5 / Gaps. |
| `web/src/lib/editor-prefs.ts`, `EditorLinkPicker.svelte` | localStorage override, picker UI | ✓ VERIFIED | 15 + 14 unit tests pass; localStorage confined to this one file. |
| `web/scripts/breadcrumb-check.mjs`, `corpora/breadcrumb-check.json` | Live-browser gate + committed record | ✓ VERIFIED | Re-run at HEAD reproduces the committed record (`success:true`, `gutterLinkCount:418`, `gutterClickIssuedRpc:true`). |
| `09-MUTATION-LOG.md` | 4 families + Closing, byte-clean proof | ✓ VERIFIED | All 4 `## Family (` headings + `## Closing` present; phase-wide byte-clean proof re-confirmed (`git status --porcelain` empty at HEAD). |
| `09-SECURITY.md` | House format, boundary sentences, 22 rows | ✓ VERIFIED | See Truth 4. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `SourcePane.svelte` | `GetEditorLink` rpc | load-time probe `$effect` + gutter click handler | ✓ WIRED | `location.assign` count = 1, `window.open` count = 0; live gate observed the click→rpc sequence. |
| `internal/cli/editorurl.go` | `uiserver.ValidateEditorTemplate`/`EditorLinkOptions` | `resolveEditorLink` before `uiserver.Listen` | ✓ WIRED | `TestUICommandRefusesMalformedEditorURLBeforeBinding` passes; one validator reused by both CLI and per-request handler. |
| `internal/cli/editordiscovery.go` | `uiserver.EditorPresets()` | `templateForLauncher` | ✓ WIRED | No duplicated scheme string in `editordiscovery.go`; discovered templates all pass `ValidateEditorTemplate`. |
| `web/src/lib/editor-prefs.ts` | `GetEditorLinkRequest.template` | `templateForRequest` (single resolution seam) | ✓ WIRED | Same-value-twice invariant test passes; probe and gutter click both call it. |
| `GetEditorLinkResponse.url` | browser `<a href>` / `location.assign` | OS URI handler | ✓ WIRED (Chromium only) | Confirmed live; Safari/WebKit and Firefox recorded as unverified in `09-SECURITY.md` Notes (honest, not silently assumed). |

### Behavioral Spot-Checks / Live-Gate Re-Execution

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Go unit/integration suite, `internal/uiserver` | `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/uiserver/...` | 295 PASS, 0 FAIL | ✓ PASS |
| Go unit/integration suite, `internal/cli` | `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/cli/...` | 286 PASS, 0 FAIL | ✓ PASS |
| Confinement + allowlist + preset tests | `go test -run 'TestGetEditorLinkPathConfinementAtRPCBoundary\|TestEditorTemplateSchemeAllowlist\|TestEditorPresetsAreExactlyThreeAndNameNoZed\|TestUIServiceMethodSetIsExactlyTheReadSet\|TestUIProtoFieldNumbersAreStableAndUnique' ./internal/uiserver/...` | all PASS | ✓ PASS |
| Full web test suite | `pnpm -C web test` | 542/542 passed (48 files) | ✓ PASS |
| Targeted phase test files | `vitest run` on 6 phase-specific test files | 75/75 passed | ✓ PASS |
| `task web:drift` | rebuild-independent digest comparison | source 114 files MATCH, output 32 files MATCH | ✓ PASS |
| `task proto:drift` | 4 generated files vs. fresh regen | byte-identical | ✓ PASS |
| `pnpm -C web check` (svelte-check) | strict TS check | **2 ERRORS**, exit 1 | ✗ FAIL (see Truth 5) |
| Live browser gate | `node web/scripts/breadcrumb-check.mjs` (real Chromium, real index) | `success:true`, breadcrumb + editor-link + gutter all confirmed, no leftover process | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| BRW-10 | 09-03, 09-05 | Breadcrumb naming innermost symbol, live-browser verified, empty+stale demonstrated wrong | ✓ SATISFIED | Truth 1, Mutation Log families (a)+(b) |
| BRW-11 | 09-01, 09-04, 09-05 | Server-resolved editor link, confinement-refused at RPC boundary | ✓ SATISFIED | Truth 2, Mutation Log family (c) |
| BRW-12 | 09-01, 09-02, 09-04, 09-05 | `--editor-url` default + per-browser override + 3 presets + Zed absent | ✓ SATISFIED | Truth 3, Mutation Log family (d) |
| BRW-13 | 09-05 | SECURITY.md naming the browser prompt, not CSP, as boundary; every threat test-or-verdict | ✓ SATISFIED | Truth 4 |

No orphaned requirements: `.planning/REQUIREMENTS.md` maps exactly BRW-10…13 to Phase 9, and all four are marked `Complete` there and are individually traced above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `web/src/lib/components/browse/SourcePane.svelte` | 446 | Possibly-undefined function invocation inside an async closure (`svelte-check` strict-mode error) | ⚠️ Warning (functionally likely inert; formally a failed, previously-declared acceptance gate) | Breaks the phase's own repeatedly-declared "0 svelte-check errors" gate; undocumented anywhere (no WINDOWS.md entry, no SECURITY.md note, no SUMMARY correction) |
| `internal/cli/editordiscovery_test.go` | 338-339 | Dead trailing zero-check after an early `t.Fatalf` loop (IN-01, already flagged by 09-REVIEW.md) | ℹ️ Info | Harmless; correctly identified and correctly left unfixed (Info-severity, out of fix scope) as recorded in `09-REVIEW.md`/`09-REVIEW-FIX.md` |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase touched. No stub/placeholder patterns found in `SourcePane.svelte`, `EditorLinkPicker.svelte`, `editorlink.go`, or `editordiscovery.go` beyond the documented `[ASSUMED]` UI note (which is an honest, visible flag, not a hidden stub).

### Human Verification Required

Both items below are **decisions requiring human input**, not "type pass" checkpoints — per this
verification's own constraints, since neither can be resolved by any automated check available in
this session.

#### 1. Cursor and JetBrains editor-link URI templates remain `[ASSUMED]`

**Decision needed:** Is `idea://open?file={path}&line={line}` (and the analogous `cursor://file/{path}:{line}` form) actually correct against a real, installed Cursor or JetBrains IDE — does it open the right file at the right line without an intervening project-selection prompt?

**Options:**
- **Confirm** against a real install and close `.planning/WINDOWS.md` entry 35, updating `EditorLinkPicker.svelte`'s `[ASSUMED]` note and `09-SECURITY.md` Notes item 1 to "confirmed against `<version>`".
- **Leave open** — the current state (visibly flagged `[ASSUMED]` in the picker UI, honestly recorded as `not tested — remains [ASSUMED]` in both `09-SECURITY.md` and `WINDOWS.md`) is not a silent gap; it is a correctly-labeled unknown.

**Evidence status:** No JetBrains/Cursor GUI was available in the autonomous execution session (a JetBrains Toolbox CLI script being on `PATH` was explicitly and correctly *not* treated as equivalent to the visual click-through this requires — 09-05-SUMMARY.md's own reasoning). This verification session likewise had no GUI IDE available to test.

#### 2. Safari/WebKit and Firefox transient-activation behavior for the gutter's click→rpc→navigate sequence

**Decision needed:** Does the gutter's async click→`GetEditorLink`→`location.assign` sequence still fall inside the browser's transient-activation window on Safari/WebKit (research Pitfall 1 flags Safari as measurably stricter than Chromium here)? If not, the external-protocol prompt could silently fail to fire on that browser.

**Options:**
- **Test on real Safari** and update `09-SECURITY.md` Notes item 2 from "UNVERIFIED" to a confirmed/refuted result.
- **Accept Chromium-only coverage** as sufficient for this phase (the header link, which carries zero activation risk, works identically everywhere; only the gutter's async path is browser-sensitive) and track the Safari gap explicitly if it matters for this project's supported-browser matrix.

**Evidence status:** The live gate (`breadcrumb-check.mjs`) only drives Chromium (Playwright's chromium channel); no Safari/WebKit automation was run in this session either. Recorded honestly in `09-SECURITY.md` as "UNVERIFIED," not silently assumed at Chromium parity.

### Gaps Summary

One concrete, reproducible gap: **`SourcePane.svelte` fails `svelte-check` (2 errors) at HEAD**,
despite three separate plan `<verify>`/`<verification>` blocks (09-03-PLAN.md Task 2, 09-04-PLAN.md
Task 2, 09-04-PLAN.md Task 3) each explicitly requiring `pnpm -C web check` / `svelte-check` to report
"0 errors" as a precondition for the task to be considered done. The break was introduced by the
code-review fix commit `28d5d795` ("CR-01 re-probe getEditorLink..."), which added a second
`activeClient.getEditorLink(...)` call inside an async `.then()` closure — TypeScript's control-flow
narrowing of the effect's top-of-body `!activeClient?.getEditorLink` guard does not survive across
that closure boundary, so `svelte-check` (strict mode) cannot prove the call is safe even though the
guard almost certainly makes it safe at runtime (the dedicated CR-01 vitest case passes; the object
being invoked is a `const` captured once and never reassigned).

This was never caught because:
- `09-05-PLAN.md`'s phase-close gate list (Task 3) does not include `pnpm check`/svelte-check at all — it lists `proto:drift`, `web:drift`, `web:components:drift`, `test:unit`, `web:test`, the live gate, and the preset-count test, and 09-05-SUMMARY.md's pasted phase-close transcript confirms none of those commands is svelte-check.
- The code-review's re-verification (09-REVIEW.md iterations 2-3, which fixed WR-01 through WR-05 — all landing *after* CR-01) re-ran `go build`, `go vet`, `go test`, `pnpm -C web test`, and `task web:drift`, but never re-ran svelte-check either.
- 09-04-SUMMARY.md's own pasted "0 ERRORS" transcript is accurate for the state it was captured in — CR-01 did not exist yet — so no SUMMARY document is lying; the gap opened entirely inside the code-review/phase-close window, in a script this project runs by hand rather than wires into `Taskfile.yml` or CI.

**Recommended closure:** a one-line defensive re-guard inside the `.then` closure (or hoisting
`activeClient.getEditorLink` into a locally-narrowed `const` before the closure), followed by
`pnpm -C web check` reporting 0 errors, `task web:build && task web:drift` to keep the committed SPA
byte-identical, and a note added to the phase's own record (SUMMARY addendum or a fresh WINDOWS.md
entry, orchestrator's call) so this silent gate regression doesn't recur invisibly next time a
similar fix lands late in a phase.

**Everything else in the phase is solid.** All four ROADMAP success criteria are met with strong,
independently-reproduced evidence: a live-browser breadcrumb gate demonstrated wrong against both a
pre-fix binary and a live mutation; a real Connect-RPC confinement test demonstrated wrong with the
gate physically removed; a positive scheme allowlist demonstrated wrong with `javascript` admitted;
and a SECURITY.md whose every one of 22 threat rows resolves to a real, re-runnable test or an
explicit verdict. The code-review loop (5 findings, 2 iterations) converged clean and its fixes are
substantively real (verified by reading the diffs directly, not trusting the review's own narrative).
The one gap found here is narrow, almost certainly benign at runtime, and entirely a documentation/
gate-hygiene miss rather than a functional regression in the shipped breadcrumb or editor-handoff
behavior.

---

_Verified: 2026-09-12T21:10:00Z_
_Verifier: Claude (gsd-verifier)_
