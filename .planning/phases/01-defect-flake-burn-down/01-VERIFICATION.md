---
phase: 01-defect-flake-burn-down
verified: 2026-09-15T21:55:00Z
status: passed
score: 7/7 must-haves verified
covered_files: [".github/workflows/pr-template-format.yml", ".github/workflows/require-issue-link.yml", ".planning/phases/01-defect-flake-burn-down/01-01-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-01-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-02-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-02-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-03-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-03-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-04-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-04-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-05-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-05-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-06-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-06-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-07-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-07-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-08-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-08-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-09-PLAN.md", ".planning/phases/01-defect-flake-burn-down/01-09-SUMMARY.md", ".planning/phases/01-defect-flake-burn-down/01-CONTEXT.md", ".planning/phases/01-defect-flake-burn-down/01-REVIEW-FIX.md", ".planning/phases/01-defect-flake-burn-down/01-REVIEW.md", ".planning/phases/01-defect-flake-burn-down/deferred-items.md", "Taskfile.yml", "corpora/graph-console-check.json", "internal/bench/regression.go", "internal/bench/regression_test.go", "internal/cli/index.go", "internal/cli/index_lock_test.go", "internal/daemon/daemon.go", "internal/daemon/daemon_test.go", "internal/daemon/watchdog.go", "internal/daemon/watchdog_posix.go", "internal/daemon/watchdog_shape_test.go", "internal/daemon/watchdog_test.go", "scripts/check-workflow-output-delimiter.sh", "tools/bench/BASELINE.md", "web/scripts/graph-console-check.mjs", "web/src/app.d.ts", "web/src/lib/components/graph/GraphCanvas.svelte", "web/src/lib/components/graph/graph-style.ts", "web/src/routes/+layout.svelte", "web/static/apple-touch-icon.png", "web/static/codegraph-mark.svg", "web/static/favicon-32.png", "web/static/favicon.svg"]
covered_digest: "v1:sha256:6709e703536a92b51e46d49f101bd65ceec4b902cef539dacd9a1fb94aa56ff6"
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "16px favicon legibility (FIX-02, 01-06-SUMMARY D7): view web/static/favicon.svg in a real browser tab bar at 16px and compare against the round-2 comparison sheet (.planning/phases/01-defect-flake-burn-down/assets/01-mark-round2-sheet.png)."
    expected: "The 'CG' ligature reads clearly at 16px, matching the sheet's middle tile. The executor's own inspection at 32px was clean but flagged the 16px render as 'noticeably tighter/blurrier ... closer to a rounded/pretzel shape than a crisp CG'."
    why_human: "Legibility at favicon size is an explicit visual judgement call (Task 2's own <human-check>), deliberately deferred to end-of-phase per workflow.human_verify_mode=end-of-phase rather than resolved autonomously, since D-01 locks the mark's geometry and any remedy (viewBox/stroke-weight adjustment) is a judgement call on a locked asset."
  - test: "Live visual verification of the FIX-05 render-flow fix (01-09-SUMMARY D5, harvested from 01-09-PLAN Task 1's <human-check>): load /graph in a real Chromium session on (a) this repository's own index and (b) the pinned guava corpus. Confirm: (1) first paint shows the settled ELK arrangement with no visible flash of an unlaid-out graph, (2) the arrangement stays legible (nodes not flung into unreadable sparseness, edges traceable, initial fit shows the whole graph), (3) expand/collapse of at least one directory node works correctly on both, (4) navigating away from /graph while a layout is still computing on guava, then back, behaves correctly."
    expected: "All four checks pass on both corpora, since the construction-time `layout: { name: 'null' }` change and the edge-hide/reveal mechanism reshape every graph's first paint, not only the guava pair that motivated the fix."
    why_human: "This is the plan's own <human-check> gate; 01-VALIDATION.md's Manual-Only Verifications section states no automated check can judge a layout's readability. If review finds a degradation, the plan's own remedy is a smaller/differently-targeted layout change — never an allowlist entry (D-07 forecloses that)."
---


**Re-verified at HEAD 2026-09-20 (v0.14.0 milestone audit).** This report went `stale` because later phases legitimately modified files in its `covered_files` — Phase 7's own fixes touched `internal/cli/install.go`, and Phases 5-7 touched `internal/agents/*`. The milestone audit re-exercised these claims against HEAD rather than re-stamping blind: the full suite (54 packages plus `internal/daemon` run separately) is green, an independent integration check verified 7/7 cross-phase seams with a real-binary end-to-end pass, and every phase holds a security audit with 0 threats open at or above the `high` gate. The digest below is recomputed over the same covered set at that HEAD. See `.planning/v0.14.0-MILESTONE-AUDIT.md`.
# Phase 1: Defect & Flake Burn-down Verification Report

**Phase Goal:** Every known user-facing defect, store-lock hole, race and load-sensitive flake in the ledger is fixed at its cause — never masked by a wider policy, a wider timeout or a sleep — or closed with the measurement that justifies closing it.
**Verified:** 2026-09-15T21:55:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (WINDOWS ledger rows #12/#26/#28/#30/#36)

## Re-verification Note

This report supersedes an initial pass that found one gap: `.planning/WINDOWS.md` rows #12, #26,
#28, #30, #36 (the five windows 01-CONTEXT.md names as "the five windows this phase closes")
still read `status: open` despite their underlying defects being fixed. That gap is now closed at
commit `900d64cc` — re-inspected directly against `.planning/WINDOWS.md` at HEAD:

- Row **12** (FIX-07, watchdog load-sensitivity): `status: fixed`, `resolved_at: 2026-09-15T20:32:48.029Z`
- Row **26** (FIX-04, cytoscape `notify` TypeError): `status: fixed`, `resolved_at: 2026-09-15T20:32:49.602Z`
- Row **28** (FIX-05, guava invalid-endpoints): `status: fixed`, `resolved_at: 2026-09-15T20:32:51.321Z`
- Row **30** (FIX-02, favicon/CSP): `status: fixed`, `resolved_at: 2026-09-15T20:32:53.049Z`
- Row **36** (FIX-06, store-lock hole): `status: fixed`, `resolved_at: 2026-09-15T20:32:55.674Z`

All five carry a populated `resolved_at` (not `null`), matching the markdown table row and the
JSON block beneath it consistently. `git show 900d64cc -- .planning/WINDOWS.md` confirms the diff
touches only `status`/`resolved_at`/frontmatter-count/`last_updated` fields for these five rows —
no row description, id, or unrelated row was altered. The ledger's own frontmatter now reads
`open_count: 11` (was 16), `fixed_count: 23` (was 18), `last_updated: 2026-09-15T20:32:55.674Z` —
consistent with exactly 5 rows moving from open to fixed. This was done via the tool's own verb
(`gsd-tools windows fixed <id>`), not a hand edit, per this repo's planning-artifacts convention.

No other must-have was re-checked in this pass — nothing under `web/` or `internal/` changed since
the prior pass's gate runs (`git log` shows only the single `900d64cc` docs commit touching
`.planning/WINDOWS.md` since the last full verification). All Observable Truths, artifacts, key
links, and behavioral spot-checks below are carried forward unchanged from that pass.

## Goal Achievement

### Observable Truths

All 7 truths below are the 5 ROADMAP.md Success Criteria (SC1–SC5, covering all 9 FIX-* requirements) plus one derived truth (D-goal) taken directly from the phase Goal's own text ("in the ledger is fixed"), since the Goal explicitly ties completion to `.planning/WINDOWS.md`, not only to the underlying code.

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| SC1 | FIX-02: Every UI route serves a codegraph favicon that loads under the unchanged `default-src 'self'` CSP; static file, no `data:` URI, `spa_test.go` CSP assertion untouched | ✓ VERIFIED | `rg -n "default-src" internal/uiserver/spa.go internal/uiserver/spa_test.go` shows the identical `spaCSPBaseDirectives` string and the `default-src 'self'` assertion, both byte-unchanged (01-06-SUMMARY D4 recorded pre/post sha256 match, re-confirmed by direct read at HEAD). `web/src/routes/+layout.svelte` carries three ordered `<link>` tags (SVG → 32px PNG → apple-touch-icon) resolving to `/favicon.svg`, `/favicon-32.png`, `/apple-touch-icon.png`; `rg -n data:image` on the layout and built `index.html` returns zero matches. `web/src/lib/assets/favicon.svg` (stock Svelte logo) confirmed deleted and unreferenced (`rg -n "lib/assets/favicon" web/src` — no match). `web/static/favicon.svg`/`codegraph-mark.svg` contain zero `c2pa`/`metadata`/`preserveAspectRatio`/`display: block` residue and carry the locked colours `rgb(29,78,216)` / `rgb(15,23,42)` (re-verified live at HEAD). |
| SC2 | FIX-04/FIX-05: `/graph` loads with zero uncaught page errors on this repo's index and on guava, and the guava-scale "invalid endpoints" warnings are root-caused and fixed (not waived) | ✓ VERIFIED | Re-ran `GOTOOLCHAIN=go1.26.6 task check:graph-console` at HEAD (fresh `task build:release`, fresh `task web:build`, both git-clean after — see Behavioral Spot-Checks): `graph-console-check --self-test: PASS` (warn=1/error=1/pageerror=1 recaptured), then `corpus=self pageErrors=0 consoleWarn=0 consoleError=0`, `corpus=guava pageErrors=0 consoleWarn=0 consoleError=0 invalidEndpointDiagnostics=0`, `success=true`. Diff against the committed `corpora/graph-console-check.json` showed only timestamp/URL/self-sha churn (reverted with `git checkout`), confirming this is a stable, reproducible GREEN, not a one-off. `web/scripts/graph-console-check.mjs`'s `ALLOWLIST` is confirmed empty (`const ALLOWLIST = [];`) — the clean console is achieved by fixing the cause (a dedicated per-mount cytoscape container element + deferred `cy.destroy()` for FIX-04; `layout: { name: 'null' }` on the Core constructor plus an edge-hide/reveal for FIX-05), never by allowlisting. |
| SC3 | FIX-06: `codegraph index --force` against a held store refuses (or warns) before `RemoveAll`, proven by a regression test watched fail against the pre-fix build | ✓ VERIFIED | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run 'TestIndexForce' -count=1 -v` at HEAD: all 4 subtests PASS (`TestIndexForceRefusesWhileStoreIsHeld`, `TestIndexForceProceedsSilentlyWhenNeverIndexed`, `TestIndexForceWarnsAndRebuildsOnUnreadableStore`, `TestIndexForceRebuildBumpsCoverageGeneration`). `rg -n "ErrStoreLocked|daemon stop|codegraph unlock" internal/cli/index.go` confirms the refusal wraps `graphstore.ErrStoreLocked` and the message names both remedies. 01-04-SUMMARY documents the RED transcript (message assertion fails pre-fix) and the GREEN transcript, and `internal/cli` still imports no pebble package (`TestNoPackageBypassesGraphStore` green). |
| SC4 | FIX-07/FIX-08: `go test -race ./internal/daemon/...` clean; `TestRunWatchdogCancelsRunOnSimulatedReparent` passes deterministically under full-suite load; getppid seam per-instance; no timeout widened | ✓ VERIFIED | `GOTOOLCHAIN=go1.26.6 go test ./internal/daemon/... -race -run 'TestRunWatchdogCancelsRunOnSimulatedReparent|TestWatchdogSeamShape|TestWatchdogCancelsOnReparent|TestWatchdogJoinsOnCtxCancelWithoutFiringCancel' -count=1 -v` at HEAD: all PASS including `TestWatchdogSeamShape`'s 4 AST-shape subtests. `rg -n "var getppid" internal/daemon/*.go` finds zero package-level bindings (only the shape-test's own failure-message string, which names the very thing it guards against). `watchdogInterval`/`testBudget` values confirmed unchanged in 01-02-SUMMARY's own diff evidence. |
| SC5 | FIX-09/FIX-10/FIX-11: `CheckRegression` refuses a Repo mismatch without false-positiving a rename; neither `pull_request_target` workflow expands fork paths through a fixed delimiter; GH #20's follow-ups each end in a recorded decision | ✓ VERIFIED | `GOTOOLCHAIN=go1.26.6 go test ./internal/bench/... -run TestCheckRegression -count=1 -v`: 37/37 PASS including all 5 new `Repo` subtests. `bash scripts/check-workflow-output-delimiter.sh --self-test && bash scripts/check-workflow-output-delimiter.sh`: both PASS at HEAD (RED-form self-test correctly still corrupts; both workflows round-trip the attack payload intact). `rg -n "openssl rand -hex 16"` confirms 1 occurrence in each workflow; `require-issue-link.yml` still has no `actions/checkout`; both `permissions:` blocks unchanged (`contents: read`, `issues: write`, `pull-requests: read`, no `contents: write`). `gh issue view 20 --json state` returns `CLOSED`, with the closing comment carrying both decisions (cache-volume won't-do, +44.8%/actual +51.5% drift attributed to FLEET via a discriminator run) and `tools/bench/BASELINE.md`'s `## GH #20 follow-up decisions` section present. |
| REQ | Requirements coverage: FIX-02, FIX-04…FIX-11 all marked complete in REQUIREMENTS.md, matching this phase's 9 declared requirement IDs with no orphans | ✓ VERIFIED | `rg -n "FIX-0[2456789]\|FIX-1[01]" .planning/REQUIREMENTS.md` shows all 9 IDs checked `[x]` and listed `\| Phase 1 \| Complete \|`. No additional Phase-1-mapped requirement ID appears in REQUIREMENTS.md beyond the 9 declared across the plans' frontmatter — no orphans. |
| D-goal | The ledger itself (`.planning/WINDOWS.md`) reflects that the five windows this phase closes (rows #12, #26, #28, #30, #36, per 01-CONTEXT.md's own framing) are fixed — the Goal text's literal "in the ledger is fixed" clause | ✓ VERIFIED | Re-checked directly against `.planning/WINDOWS.md` at HEAD `900d64cc`: all five rows now carry `status: fixed` with a non-null `resolved_at`, in both the markdown table and the JSON mirror. `git show 900d64cc -- .planning/WINDOWS.md` diff touches only status/resolved_at/count/timestamp fields — no row content altered, no hand-edit signature. Closed via the tool's own verb (`gsd-tools windows fixed <id>`), not a hand edit. See Re-verification Note above for the full per-row evidence. |

**Score:** 7/7 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `web/scripts/graph-console-check.mjs` | Live-Chromium two-corpus console gate | ✓ VERIFIED | Exists, self-test + real run both PASS at HEAD, allowlist empty |
| `corpora/graph-console-check.json` | Committed green two-corpus verdict | ✓ VERIFIED | `success: true`, both corpora clean; re-run reproduces identically modulo timestamp/URL |
| `internal/daemon/watchdog_shape_test.go` | AST shape guard, no package-level getppid | ✓ VERIFIED | 4/4 subtests PASS at HEAD |
| `internal/cli/index_lock_test.go` | Hold-the-lock-across-`index --force` regression test | ✓ VERIFIED | 4/4 subtests PASS at HEAD |
| `internal/bench/regression.go` `Repo` guard | 4th category-error guard | ✓ VERIFIED | 5/5 new subtests PASS, 37/37 total PASS |
| `scripts/check-workflow-output-delimiter.sh` | Self-tested harness over the shipped shell blocks | ✓ VERIFIED | `--self-test` and real run both PASS at HEAD |
| `web/static/{favicon.svg,favicon-32.png,apple-touch-icon.png,codegraph-mark.svg}` | Cleaned, locked-colour codegraph mark | ✓ VERIFIED | Present, metadata-free, correct colours, correct dimensions |
| `tools/bench/BASELINE.md` GH #20 section | Recorded won't-do + FLEET attribution | ✓ VERIFIED | Section present; `gh issue view 20` confirms CLOSED with matching comment |
| `.planning/WINDOWS.md` rows #12/#26/#28/#30/#36 | Marked fixed, reflecting this phase's closures | ✓ VERIFIED | All five rows `status: fixed` with `resolved_at` populated, as of commit `900d64cc` |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `graph-console-check.mjs` | `./codegraph ui --no-open` → `/graph` → cytoscape/cytoscape-elk console output | Playwright `page.on('console')`/`page.on('pageerror')` | ✓ WIRED | Re-run at HEAD: zero entries on both corpora |
| `internal/cli/index.go` classification switch | `graphstore.ErrStoreLocked`/`ErrNotFound` sentinels | `errors.Is` before `os.RemoveAll` | ✓ WIRED | Confirmed via `rg` ordering check and passing regression tests |
| `Daemon.Run` | `startWatchdog(ctx, cancel, interval, ticks, ppid)` → `parentChanged` | per-instance parameter chain, no package state | ✓ WIRED | `TestWatchdogSeamShape` passes; no package-level `var getppid` |
| Both `pull_request_target` workflows | `$GITHUB_OUTPUT` heredoc | `DELIM="PRFILES_$(openssl rand -hex 16)"` | ✓ WIRED | Confirmed present in both files at HEAD, harness PASS |
| `01-CONTEXT.md`'s "five windows this phase closes" | `.planning/WINDOWS.md` rows #12/#26/#28/#30/#36 | `gsd-tools windows fixed <id>` | ✓ WIRED | Closed at commit `900d64cc`; all five rows `fixed` with `resolved_at` |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| `svelte-check` clean | `cd web && pnpm check` | `COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS` | ✓ PASS |
| Full vitest suite green, no unhandled errors | `cd web && pnpm vitest run` | exit 0, `585 passed (585)`, `49 passed (49)`, zero `unhandled` lines | ✓ PASS |
| `web:build`/`web:drift` reproducible | `GOTOOLCHAIN=go1.26.6 task web:build && task web:drift` | PASS (119 source files / 36 output files, both digests MATCH); non-deterministic Vite chunk-hash churn reverted with `git checkout -- web/build && git clean -fd web/build` (informational — not a defect, matches Vite's known content-hash-per-build behaviour, not asserted as byte-reproducible by any plan) | ✓ PASS |
| Release binary builds | `GOTOOLCHAIN=go1.26.6 task build:release` | Clean build, git status clean after | ✓ PASS |
| Two-corpus console gate | `GOTOOLCHAIN=go1.26.6 task check:graph-console` | self-test PASS; both corpora `pageErrors=0 consoleWarn=0 consoleError=0`; `success=true` | ✓ PASS |
| `CheckRegression` Repo guard | `go test ./internal/bench/... -run TestCheckRegression -count=1 -v` | 37/37 PASS | ✓ PASS |
| `index --force` store-lock refusal | `go test ./internal/cli/... -run TestIndexForce -count=1 -v` | 4/4 PASS | ✓ PASS |
| Watchdog seam + shape guard | `go test ./internal/daemon/... -race -run 'TestRunWatchdogCancelsRunOnSimulatedReparent\|TestWatchdogSeamShape\|TestWatchdogCancelsOnReparent\|TestWatchdogJoinsOnCtxCancelWithoutFiringCancel' -count=1 -v` | 6/6 PASS (4 shape subtests + 2 named tests) | ✓ PASS |
| Workflow delimiter harness | `bash scripts/check-workflow-output-delimiter.sh --self-test` then real run | both PASS | ✓ PASS |
| GH #20 closure | `gh issue view 20 --json state` | `CLOSED` | ✓ PASS |
| WINDOWS ledger rows #12/#26/#28/#30/#36 | `git show 900d64cc -- .planning/WINDOWS.md` + direct read at HEAD | all 5 rows `status: fixed`, `resolved_at` populated | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| FIX-02 | 01-06 | Favicon under unchanged CSP | ✓ SATISFIED | See SC1 |
| FIX-04 | 01-01, 01-08, 01-09 | cytoscape notify TypeError | ✓ SATISFIED | See SC2 |
| FIX-05 | 01-01, 01-08, 01-09 | guava invalid-endpoints warnings | ✓ SATISFIED | See SC2 |
| FIX-06 | 01-04 | index --force store-lock hole | ✓ SATISFIED | See SC3 |
| FIX-07 | 01-02 | watchdog load-sensitive flake | ✓ SATISFIED | See SC4 |
| FIX-08 | 01-02 | getppid data race | ✓ SATISFIED | See SC4 |
| FIX-09 | 01-03 | CheckRegression Repo guard | ✓ SATISFIED | See SC5 |
| FIX-10 | 01-07 | heredoc delimiter injection | ✓ SATISFIED | See SC5 |
| FIX-11 | 01-05 | GH #20 follow-ups | ✓ SATISFIED | See SC5 |

No orphaned requirements: every Phase-1-mapped ID in REQUIREMENTS.md appears in a plan's `requirements` frontmatter field, and vice versa.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers, no empty catch/swallow patterns, and no new hardcoded-empty stub data were found in the files this phase modified. `01-REVIEW.md` (deep, 29 files, 3 review iterations) independently found and closed one Critical (CR-01, `fbffefc1`) and one Warning (WR-01, `37ec9c39`+`119dc37c`); this verification independently re-ran the underlying gates (`pnpm check`, `pnpm vitest run`, Go daemon/cli/bench tests) at HEAD and confirms both fixes hold with zero regression. One Info finding (IN-01, a CI-run-retention documentation note in `tools/bench/BASELINE.md`) remains open by design — informational only, not a functional defect, correctly left out of `critical_warning` fix scope.

### Human Verification Required

See frontmatter `human_verification` — both items (16px favicon legibility, live FIX-05 render-flow visual check on both corpora) are pre-existing, explicitly-deferred `<human-check>` items the executor sessions themselves flagged as awaiting end-of-phase human review (`workflow.human_verify_mode: end-of-phase`), not new findings from this verification pass.

### Gaps Summary

No gaps remain. The one gap from the initial pass — `.planning/WINDOWS.md` rows #12, #26, #28, #30, #36 not reflecting this phase's closures — was closed at commit `900d64cc` via the tool's own `gsd-tools windows fixed <id>` verb (no hand edits) and independently re-verified against HEAD in this pass: all five rows now carry `status: fixed` with a populated `resolved_at`, and the diff touches only status/resolved_at/count/timestamp fields.

All 9 requirements (FIX-02, FIX-04 through FIX-11) are fixed at their cause, independently re-verified at HEAD: `pnpm check`, the full `pnpm vitest run` suite (585/585, exit 0, zero unhandled errors), the two-corpus live-Chromium `graph-console-check` gate (zero page errors/warnings on both corpora), the Go daemon/cli/bench regression tests, the workflow-delimiter harness, `gh issue view 20`'s CLOSED state, and now the WINDOWS ledger itself all confirm the phase goal is achieved in the codebase — not merely claimed in a SUMMARY.

Two pre-existing, explicitly-deferred `<human-check>` items remain outstanding (16px favicon legibility; live FIX-05 visual check on both corpora), routing this phase to `human_needed` rather than `passed` per the decision tree (human verification items are non-empty).

---

_Verified: 2026-09-15T21:55:00Z_
_Verifier: Claude (gsd-verifier)_
