---
phase: "1"
slug: "defect-flake-burn-down"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-14"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `01-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (standard library) + `-race` for concurrency packages; hand-rolled Node `.mjs` live-Chromium scripts for the web checks (matching the five existing `web/scripts/*-check.mjs`) |
| **Config file** | none — the `.mjs` live checks are standalone Node scripts invoked directly, governed by no `playwright.config`/`jest.config` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test ./internal/daemon/... ./internal/graphstore/... ./internal/bench/...` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` |
| **Estimated runtime** | ~250 s for the full `-race` suite (measured this session) |

---

## Sampling Rate

- **After every task commit:** the narrowest applicable command — `go test ./internal/bench/...` (FIX-09), `go test ./internal/cli/...` (FIX-06), `go test -race ./internal/daemon/...` (FIX-08), `node web/scripts/graph-console-check.mjs` (FIX-04/05).
- **After every plan wave:** `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` — **this is the only command that exercises FIX-07/08 under the load condition that matters.** An isolated or per-package run passes both before and after the fix, so it is not a discriminating test for FIX-07 (confirmed by live reproduction during research).
- **Before `/gsd-verify-work`:** full suite green under `-race`, plus a recorded `graph-console-check.mjs` run against both corpora committed under `corpora/`.
- **Max feedback latency:** ~250 s (full suite); < 30 s for the narrow per-task commands.

---

## Per-Task Verification Map

> Task IDs are filled in by `/gsd-validate-phase` once PLAN.md files exist. The rows below
> bind each requirement to its proving command and its RED demonstration — the phase-level
> contract the planner must honor when it authors `<verify>` blocks.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-06.1–2 | 01-06 | 2 | FIX-02 | T-01-06-01 | Favicon loads under the unchanged `default-src 'self'` CSP; no CSP-blocked console entry | unit + live-browser | `GOTOOLCHAIN=go1.26.6 go test ./internal/uiserver/ -run TestFavicon -count=1` + `task check:graph-console` | ✅ `internal/uiserver/favicon_test.go` (added by validate-phase, d2aaab2d); ✅ `internal/uiserver/spa_test.go` negative control unedited | ✅ green |
| 01-01.1–2, 01-08.1–3, 01-09.1–2 | 01-01, 01-08, 01-09 | 1, 3, 4 | FIX-04 | — | Zero uncaught page errors on `/graph`, both corpora | live-browser + unit | `task check:graph-console` (self-test, then both corpora) + `cd web && pnpm vitest run tests/graph-live-update.test.ts` | ✅ `web/scripts/graph-console-check.mjs`, ✅ `corpora/graph-console-check.json` (success: true), ✅ `web/tests/graph-live-update.test.ts` (teardown-guard + CR-01 bubble regression) | ✅ green |
| 01-01.1–2, 01-09.1–2 | 01-01, 01-09 | 1, 4 | FIX-05 | — | Zero `console.warn`/`console.error` from our code, both corpora, allowlist empty | live-browser | `task check:graph-console` (same script; RED/GREEN toggle recorded in 01-09-SUMMARY) | ✅ same script + verdict; allowlist array empty | ✅ green |
| 01-04.1–2 | 01-04 | 1 | FIX-06 | T-10-16 | `index --force` refuses before `RemoveAll` on a held store; warns and rebuilds on a corrupt one | integration | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run TestIndexForce -count=1` | ✅ `internal/cli/index_lock_test.go` (4 tests) | ✅ green |
| 01-02.1–3 | 01-02 | 1 | FIX-07 | — | Watchdog test passes deterministically under parallel load (injected ticker, no timeout widening) | integration | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/daemon/...` | ✅ `internal/daemon/watchdog_test.go`, `daemon_test.go` | ✅ green |
| 01-02.1–3 | 01-02 | 1 | FIX-08 | — | `getppid` seam is per-instance; race structurally impossible | unit + code-shape | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/daemon/... -run TestWatchdogSeamShape` **and** `rg -n '^var getppid' internal/daemon/watchdog.go` returning no match | ✅ `internal/daemon/watchdog_shape_test.go` | ✅ green |
| 01-03.1–2 | 01-03 | 1 | FIX-09 | — | `CheckRegression` refuses a `Repo` mismatch with a rebless-pointing message | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/bench/... -run TestCheckRegression -count=1` | ✅ `internal/bench/regression_test.go` (37 subtests) | ✅ green |
| 01-07.1–2 | 01-07 | 2 | FIX-10 | GH #15 | Neither workflow's heredoc is terminable by a fork-controlled `PRFILES_EOF` path (per-run random delimiter) | shell/script | `task check:workflow-output-delimiter` (self-test RED, then real run) | ✅ `scripts/check-workflow-output-delimiter.sh` + Taskfile target | ✅ green |
| 01-05.1–3 | 01-05 | 1 | FIX-11 | — | Both GH #20 follow-ups end in a recorded decision | process/doc | `gh issue view 20 --json state` = CLOSED; `tools/bench/BASELINE.md` entry (drift = fleet +46.90%, code +3.15%) | N/A — process gate (manual-only by design) | ✅ closed |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## RED Demonstration Contract (rule `84d1gfpywd`)

Every new guard in this phase MUST be watched fail before it passes, with the transcript
pasted into the plan's `<verify>` evidence. A guard carrying only a negative assertion
("no error appeared") passes vacuously the moment its anchor stops matching.

| Requirement | RED demonstration |
|-------------|-------------------|
| FIX-02 | Run the favicon check against the pre-fix build (Svelte logo + `data:` URI) — must report the CSP-blocked entry. |
| FIX-04/05 | Run `graph-console-check.mjs` against the pre-fix build — must report the `notify` TypeError, the `invalid endpoints` warnings at guava scale, and the two `text-valign` warnings, with counts. |
| FIX-06 | Hold the Pebble LOCK across `index --force` against the **pre-fix** binary — the test must fail (the wipe proceeds at floor 0). |
| FIX-07 | Already RED: `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` fails `TestRunWatchdogCancelsRunOnSimulatedReparent` today (250.35 s, zero data races). Record the pre-fix transcript. |
| FIX-08 | The code-shape assertion must fail against the current tree (`var getppid` is present today at `internal/daemon/watchdog.go:21`). |
| FIX-09 | The new `TestCheckRegression` mismatch subtest must fail against the pre-fix `regression.go` (no `Repo` guard → returns nil). |
| FIX-10 | The shell harness must fail against the **current** workflow block when fed a file list containing a literal `PRFILES_EOF` line. |

---

## Wave 0 Requirements

- [x] `internal/cli/index_lock_test.go` — FIX-06, modeled on `internal/graphstore/open_lock_test.go`
- [x] `web/scripts/graph-console-check.mjs` — FIX-04 and FIX-05 together (D-09 names one script)
- [x] `scripts/check-workflow-output-delimiter.sh` + `task check:workflow-output-delimiter` — a shell/script test exercising the `$GITHUB_OUTPUT` heredoc block + its Taskfile target — FIX-10
- [x] `internal/daemon/watchdog_shape_test.go` — a code-shape assertion that `internal/daemon/watchdog.go` has no package-level `var getppid` — FIX-08's "structurally impossible" bar (a passing `-race` run proves only that no race happened *this run*, never that one is impossible)
- [x] `internal/uiserver/favicon_test.go` — a favicon-content check that fails on the Svelte logo — FIX-02 (added by validate-phase 2026-09-15, RED-proven)
- [x] No framework install needed — every gap is a new file inside the existing Go / `.mjs` conventions

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Layout change does not visibly regress the small graph | FIX-05 | D-07 makes "always fix" conditional on a human looking at both graphs; no automated check can judge a layout's readability | Load `/graph` on this repo's index and on pinned guava in Chromium before and after the ELK change; confirm the arrangement is still legible at both scales |
| The shipped mark reads correctly at 16 px and 32 px | FIX-02 | Legibility at favicon size is a visual judgement, not an assertion | Render the shipped SVG at 16 px and 32 px and compare against `.planning/phases/01-defect-flake-burn-down/assets/01-mark-round2-sheet.png` |
| GH #20 follow-up decisions are honest | FIX-11 | A recorded decision is prose; its correctness is editorial | Read the `tools/bench/BASELINE.md` entry and the GH #20 close comment against the measurement actually run |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags (`pnpm vitest run` one-shot; `go test -count=1`)
- [x] Feedback latency < 250s (`task check:graph-console` ~90 s; unit gates < 80 s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-09-15 by /gsd-validate-phase (autonomous run, maintainer chose "Fix the gap")

## Validation Audit 2026-09-15
| Metric | Count |
|--------|-------|
| Gaps found | 1 (FIX-02: favicon checks existed only as plan 01-06's one-shot `<automated>` commands) |
| Resolved | 1 (`internal/uiserver/favicon_test.go`, commit d2aaab2d — 4 tests, RED-proven by inverting the `svelte` assertion) |
| Escalated | 0 |

Post-execution note: FIX-04/05's web coverage was tightened after execution by the code-review fix loop (CR-01 bubble-scoping regression in `graph-live-update.test.ts`; WR-01 fake-completeness across 7 test files) — `cd web && pnpm vitest run` must be judged by exit code 0 plus zero "unhandled errors" lines, not by the pass count.
