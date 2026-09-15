---
phase: "1"
slug: "defect-flake-burn-down"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| TBD | TBD | TBD | FIX-02 | — | Favicon loads under the unchanged `default-src 'self'` CSP; no CSP-blocked console entry | unit + live-browser | `go test ./internal/uiserver/... -run TestSPA` (negative control — must stay green and unedited) + live `/` + `/graph` load asserting no CSP violation for the icon | ✅ `internal/uiserver/spa_test.go`; ❌ favicon-content check — W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-04 | — | Zero uncaught page errors on `/graph`, both corpora | live-browser | `node web/scripts/graph-console-check.mjs` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-05 | — | Zero `console.warn`/`console.error` from our code, both corpora | live-browser | `node web/scripts/graph-console-check.mjs` (same script) | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-06 | T-10-16 | `index --force` refuses before `RemoveAll` on a held store; warns and rebuilds on a corrupt one | integration | `go test ./internal/cli/... -run TestIndexForce` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-07 | — | Watchdog test passes deterministically under full-suite parallel load | integration | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./...` | ✅ test exists, **currently RED** under this command | ⬜ pending |
| TBD | TBD | TBD | FIX-08 | — | `getppid` seam is per-instance; race structurally impossible | unit + code-shape | `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/daemon/...` **and** `rg -n '^var getppid' internal/daemon/watchdog.go` returning no match | ✅ race test; ❌ code-shape assertion — W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-09 | — | `CheckRegression` refuses a `Repo` mismatch with a rebless-pointing message | unit | `go test ./internal/bench/... -run TestCheckRegression` | ✅ `internal/bench/regression_test.go` | ⬜ pending |
| TBD | TBD | TBD | FIX-10 | GH #15 | Neither workflow's heredoc is terminable by a fork-controlled `PRFILES_EOF` path | shell/script | new script test + Taskfile target | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | FIX-11 | — | Both GH #20 follow-ups end in a recorded decision | process/doc | `git log -S'11279' -- tools/bench/baseline.json`, a `workflow_dispatch` bench run, a `tools/bench/BASELINE.md` entry, `gh issue close 20` | N/A — process gate | ⬜ pending |

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

- [ ] `internal/cli/index_lock_test.go` — FIX-06, modeled on `internal/graphstore/open_lock_test.go`
- [ ] `web/scripts/graph-console-check.mjs` — FIX-04 and FIX-05 together (D-09 names one script)
- [ ] A shell/script test exercising the `$GITHUB_OUTPUT` heredoc block + its Taskfile target — FIX-10
- [ ] A code-shape assertion that `internal/daemon/watchdog.go` has no package-level `var getppid` — FIX-08's "structurally impossible" bar (a passing `-race` run proves only that no race happened *this run*, never that one is impossible)
- [ ] A favicon-content check that fails on the Svelte logo — FIX-02
- [ ] No framework install needed — every gap is a new file inside the existing Go / `.mjs` conventions

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Layout change does not visibly regress the small graph | FIX-05 | D-07 makes "always fix" conditional on a human looking at both graphs; no automated check can judge a layout's readability | Load `/graph` on this repo's index and on pinned guava in Chromium before and after the ELK change; confirm the arrangement is still legible at both scales |
| The shipped mark reads correctly at 16 px and 32 px | FIX-02 | Legibility at favicon size is a visual judgement, not an assertion | Render the shipped SVG at 16 px and 32 px and compare against `.planning/phases/01-defect-flake-burn-down/assets/01-mark-round2-sheet.png` |
| GH #20 follow-up decisions are honest | FIX-11 | A recorded decision is prose; its correctness is editorial | Read the `tools/bench/BASELINE.md` entry and the GH #20 close comment against the measurement actually run |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 250s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
