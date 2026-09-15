---
phase: 01-defect-flake-burn-down
reviewed: 2026-09-15T21:10:00Z
depth: deep
files_reviewed: 29
files_reviewed_list:
  - .github/workflows/pr-template-format.yml
  - .github/workflows/require-issue-link.yml
  - Taskfile.yml
  - corpora/graph-console-check.json
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/cli/index.go
  - internal/cli/index_lock_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/watchdog.go
  - internal/daemon/watchdog_posix.go
  - internal/daemon/watchdog_shape_test.go
  - internal/daemon/watchdog_test.go
  - scripts/check-workflow-output-delimiter.sh
  - tools/bench/BASELINE.md
  - web/scripts/graph-console-check.mjs
  - web/src/app.d.ts
  - web/src/lib/components/graph/GraphCanvas.svelte
  - web/src/lib/components/graph/graph-style.ts
  - web/src/routes/+layout.svelte
  - web/static/codegraph-mark.svg
  - web/static/favicon.svg
  - web/tests/graph-communities.test.ts
  - web/tests/graph-cycles.test.ts
  - web/tests/graph-edge-detail.test.ts
  - web/tests/graph-expand.test.ts
  - web/tests/graph-expansion.test.ts
  - web/tests/graph-live-update.test.ts
  - web/tests/graph-tracer.test.ts
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-09-15T21:10:00Z
**Depth:** deep
**Files Reviewed:** 29
**Status:** issues_found

## Summary

This is iteration 3 of `--auto` re-review, the final one. Iteration 1
(`01-REVIEW.iter2.md`) found one Critical (CR-01: the overlapping-layout
`layoutstop` listener was registered on `cy` itself, so a bubbled completion
from a stale, superseded layout could satisfy the current generation's
`.one()` early) and one Info (IN-01: `tools/bench/BASELINE.md` cites a
CI-run link subject to GitHub's retention window). CR-01 was fixed in
`fbffefc1` and confirmed closed in iteration 2 (`01-REVIEW.iter3.md`).
Iteration 2 also surfaced a new Warning (WR-01: five `web/tests/graph-*.test.ts`
`edges()` fakes missing `.style()`/`.removeStyle()`, throwing an unhandled
`TypeError` on the FIX-05 reveal path), fixed in two commits: `37ec9c39` (the
five named files) and `119dc37c` (an orchestrator-caught extension to
`graph-edge-detail.test.ts` and the three still-broken fakes in
`graph-live-update.test.ts`, needed to get the full suite's process exit code
to 0).

**This pass re-verifies both prior fixes and re-scans the full scope for
anything new.**

**CR-01 — remains closed, no regression.** Re-inspected
`GraphCanvas.svelte:396-463`: `thisLayoutRun = opts.cy.layout(...)` is still
constructed and `thisLayoutRun.one('layoutstop', ...)` still registered
*before* `activeLayoutRun = thisLayoutRun.run()` two lines later, with
nothing async in between. The generation-token check
(`myGeneration !== layoutGeneration`) is still the unconditional first line
of the callback. FIX-04's deferred-teardown path
(`isLayoutInFlight()`/`onLayoutSettled()`/the 10s fallback at
`GraphCanvas.svelte:1149-1200`) and FIX-05's hide-until-first-`layoutstop`
reveal (`start()` at line 544, the reveal at line 415) are unchanged and
still gated by the same per-instance-scoped callback. Neither of the two
WR-01 commits touched `GraphCanvas.svelte` — both are test-file-only diffs
— so there is no new surface here to regress.

**WR-01 — verified closed tree-wide, not just in the two commits' named
files.** Independently ran `rg -n 'edges\(\)' web/tests/` and inspected
every one of the 9 `edges()` fake definitions across all 7 files
(`graph-live-update.test.ts` alone defines 3): all 9 now return both
`style: () => {}` and `removeStyle: () => {}`. Ran the full suite
(`cd web && pnpm vitest run`) twice from a cold shell: **exit code 0 both
times**, `Test Files 49 passed (49)` / `Tests 585 passed (585)` both times,
zero lines matching `unhandled` (case-insensitive) in either captured log —
independently confirming the orchestrator's own run at this HEAD. Also
re-ran each of the six previously-flaky files standalone
(`graph-tracer`, `graph-expand`, `graph-expansion`, `graph-communities`,
`graph-cycles`, `graph-edge-detail`), and `graph-tracer` +
`graph-live-update` together three consecutive times — all green,
deterministically. Reviewed both fix commits' diffs directly (`git show
37ec9c39`, `git show 119dc37c`): both are minimal, mechanical additions of
the two no-op methods to existing fake-object literals, applied verbatim to
the pattern already used elsewhere in the same files — no logic changes, no
new dead code, no new gaps introduced. `pnpm check` (svelte-check) still
reports `0 ERRORS 0 WARNINGS`.

**Go-side files unchanged since iteration 1, re-verified with a fresh
build/test pass rather than assumed stable.** `go build ./...` succeeds;
`go test ./internal/bench/... ./internal/cli/... ./internal/daemon/...`
passes with no failures (`internal/daemon` and `internal/cli` exercised
live, not from cache). No new commits touched `internal/`, `Taskfile.yml`,
`.github/workflows/`, `corpora/`, or `scripts/` since iteration 1's review,
and nothing in this pass's re-scan of those files (including a fresh
grep sweep for debug artifacts, empty catches, and hardcoded secrets across
the full file scope) surfaced anything new.

No new Critical or Warning was found on this pass. WR-01 and CR-01 are both
closed with independent, direct verification (not by re-trusting the fix
reports' own claims).

IN-01 (`tools/bench/BASELINE.md`'s CI-run-retention documentation note)
remains present and unchanged — it was explicitly out of the
`critical_warning` fix scope in both prior iterations and is re-reported
honestly here rather than silently dropped, per the orchestrator's
instruction for this final pass.

## Info

### IN-01: `tools/bench/BASELINE.md`'s in-flight discriminator link points to a specific CI run that will eventually be pruned by GitHub's retention policy

**File:** `tools/bench/BASELINE.md:439,477`
**Issue:** Unchanged since iteration 1. The "GH #20 follow-up decisions"
section cites `run 34980422924` as the source of the
`16569.160272289788` files/s figure, with `[gh20-run]` linking directly to
that run's Actions page. GitHub Actions run artifacts and logs are not
retained indefinitely (typically 90 days by default), so the underlying
evidence behind this specific number may become unreachable while the
markdown file — explicitly written as a permanent investigation record
("kept complete on purpose") — persists indefinitely. Confirmed still
present at the same lines on this final pass; correctly left untouched by
both fix iterations, which were scoped to `critical_warning` findings only.
**Fix:** No action required for this review (documentation durability note,
not a functional defect); if the artifact/log is still needed as evidence
after the retention window, consider mirroring the `baseline-candidate`
artifact's raw numbers into the repo (e.g. alongside
`tools/bench/baseline.json`'s own history) rather than relying solely on the
external run link.

---

_Reviewed: 2026-09-15T21:10:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
