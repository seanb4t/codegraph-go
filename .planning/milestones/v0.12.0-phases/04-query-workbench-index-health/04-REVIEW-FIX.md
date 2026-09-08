---
phase: 04-query-workbench-index-health
fixed_at: 2026-08-30T03:00:00Z
review_path: .planning/phases/04-query-workbench-index-health/04-REVIEW.md
iteration: 1
findings_in_scope: 10
fixed: 10
skipped: 0
status: all_fixed
---

# Phase 4: Code Review Fix Report

**Fixed at:** 2026-08-30T03:00:00Z
**Source review:** .planning/phases/04-query-workbench-index-health/04-REVIEW.md
**Iteration:** 1

**Scope:** Critical + Warning findings only (2 Critical, 8 Warning). Info findings (IN-01
through IN-09) are out of scope per dispatch instructions and were not touched.

**Summary:**
- Findings in scope: 10
- Fixed: 10
- Skipped: 0

**Isolation note:** `workflow.use_worktrees` is `false` in `.planning/config.json`, and the
dispatching orchestrator explicitly instructed "Do NOT create/switch/delete branches or
worktrees." Per the fixer contract's documented opt-out, all edits and commits were made
directly in the main checkout on `gsd/v0.12.0-local-graph-ui` — no worktree was created.

## Fixed Issues

### CR-01: Clearing an analysis input while a request is in flight renders a false "Something went wrong" error

**Files modified:** `web/src/lib/components/workbench/AnalysisPanel.svelte`,
`web/tests/workbench-callers-callees.test.ts`
**Commit:** `aca810f0`
**Applied fix:** Replaced the hand-rolled module-level `abortController` variable + manual
abort call with the dispatch effect's own returned cleanup closure. Svelte calls this
cleanup before every re-run (covering the `run → undefined` idle transition CR-01
describes) and on unmount — aborting the in-flight request and bumping `requestId` in the
same step, on every path, not only the dispatch path. This closes CR-01 and WR-02 (no
teardown) with one mechanism instead of two competing ones, per the reviewer's explicit
note to fix them coherently. Added a regression test for the `run → undefined` transition;
verified it fails against the unfixed effect (false "Something went wrong" banner
reproduced) before passing against the fix.

### CR-02: `/health` presents a hardcoded, always-zero "Pending changes" tally as a live trust signal

**Files modified:** `internal/uiproto/uiv1/ui.proto`, `internal/uiproto/uiv1/ui.pb.go`,
`web/src/lib/gen/ui_pb.ts`, `internal/uiserver/health_test.go`,
`web/src/routes/health/+page.svelte`, `web/tests/health-page.test.ts`
**Commit:** `30080a95`
**Applied fix:** Chose "stop rendering it" (the review's preferred option) — removed the
`{#if pageState.response.pendingChanges}` block from `/health` entirely rather than
labeling a fabricated zero as "not yet tracked." The proto field stays on the wire
unchanged (frozen, D-02a) — field 12 unrenumbered — with a `// NOTE: inert placeholder`
comment added and the three generated artifacts regenerated (`task proto:gen`); `proto:drift`
stays green at 4 files. Replaced the tautological Go assertion in
`TestGetHealthProjectsStatusResult` (which compared the wire value against
`eng.Status()`'s own always-zero value — a comparison that can never distinguish "real
data" from "placeholder") with an explicit assertion that both sides are the documented
all-zero shape; verified this new assertion fails when the wire mapping is deliberately
broken (Added off-by-one), confirming it is a real check. Added a web regression test that
mounts `/health` with a POPULATED `pendingChanges` value and asserts nothing renders;
verified it fails against the unfixed template (where the block was truthy for any defined
value) before passing.

### WR-01: The Workbench titles the no-index state "Index is being rebuilt", contradicting TrustVerdict's own copy

**Files modified:** `web/src/lib/workbench-failure.ts`, `web/tests/workbench-failure.test.ts`
**Commit:** `6e2583bc`
**Applied fix:** Changed the `not-found` + `verdict === 'no-index'` branch's title from
"Index is being rebuilt" to "No index for this repository", matching
`TrustVerdict.svelte`'s vocabulary for the identical verdict. The genuinely distinct
`indexing` branch (an actual rebuild in progress) keeps its own unchanged title; both still
map to the `index-stale` kind for testid grouping. Added a regression test asserting the
new title and that it stays distinct from the real `indexing` title; verified it fails
against the unfixed copy.

### WR-02: `AnalysisPanel`'s dispatch effect has no teardown — in-flight analyses are never cancelled on unmount

**Files modified:** `web/src/lib/components/workbench/AnalysisPanel.svelte`
**Commit:** `aca810f0` (fixed together with CR-01 — same root cause, same mechanism)
**Applied fix:** See CR-01 above. The effect's own returned cleanup now aborts the in-flight
request on unmount (tab switch, navigation away from `/workbench`), not only on
requestKey change.

### WR-03: The Symbol field fires one full graph-analysis RPC per keystroke, with no debounce

**Files modified:** `web/src/routes/workbench/+page.svelte`,
`web/tests/workbench-callers-callees.test.ts`
**Commit:** `5865846b`
**Applied fix:** Debounced `handleSymbolInput`'s URL write using the existing
`SEARCH_DEBOUNCE_MS` (150ms) from `search.ts`, rather than inventing a second constant.
depth/limit stay immediate. Added a regression test typing "HandleRequest" one character at
a time with no delay between keystrokes, asserting exactly one Callers RPC fires with the
final value; verified it fails against the unfixed handler (14 calls, one per keystroke)
before passing (1 call).

### WR-04: `DataTable` indexes the row model with an unguarded virtual index

**Files modified:** `web/src/lib/components/workbench/DataTable.svelte`,
`web/tests/data-table-virtualization.test.ts` (new file), `web/build/*` (rebuilt)
**Commit:** `40391ecb`
**Applied fix:** Added an `{#if row}` guard around the rendered row template, so an
out-of-range virtual index (reachable when `rows` shrinks in place past the virtualizer's
stale scroll-derived count) renders nothing for that slot instead of throwing
`TypeError: Cannot read properties of undefined (reading 'id')`. Added a regression test
that scrolls a 60-row table near its end, then rerenders the SAME component instance with
5 rows (no remount) — verified this reproduces the exact crash live against the unfixed
template, and resolves cleanly after the fix.

### WR-05: Virtualization sets `aria-rowcount` without any `aria-rowindex`

**Files modified:** `web/src/lib/components/workbench/DataTable.svelte`,
`web/tests/data-table-virtualization.test.ts` (new file), `web/build/*` (rebuilt)
**Commit:** `40391ecb` (fixed together with WR-04 — same block of markup)
**Applied fix:** Added `aria-rowindex={1}` to the header row and
`aria-rowindex={virtualRow.index + 2}` to each rendered data row (1-based, offset by the
header), so assistive tech can place a windowed row within the full declared
`aria-rowcount`. Regression test asserts both values; verified `null` (unset) against the
unfixed template.

### WR-06: `web:components:drift` never enables Corepack despite its precondition accepting it

**Files modified:** `Taskfile.yml`
**Commit:** `e9a3a5a5`
**Applied fix:** Added the same `corepack enable` resolution block `web:deps` already uses,
at the top of `web:components:drift`'s recipe (rather than a `deps: [web:deps]`
dependency — this target already does its own real, isolated `pnpm install
--frozen-lockfile` in a scratch tree, so pulling in a second main-tree install would be
heavier and redundant just for this one line). Verified end-to-end: `task
web:components:drift` still passes at 50 vendored component files across 8 components,
matching the pre-fix baseline; the guard's structural floors (8 files / 2 components) were
not touched. The corepack branch itself could not be live-reproduced on this dev host
(pnpm is on PATH directly here; corepack is not installed at all) — it is a byte-for-byte
structural mirror of `web:deps`'s already-proven resolution block.

### WR-07: `FilePicker` never renders `searchState.failure`

**Files modified:** `web/src/lib/components/workbench/FilePicker.svelte`,
`web/tests/workbench-affected.test.ts`, `web/build/*` (rebuilt)
**Commit:** `7859e74a`
**Applied fix:** Adapted the review's suggested fix to the actual code: `searchState.failure`
is already a *classified* `RpcFailure` (produced by `file-search.ts`'s own
`classifyRpcError`), not a raw error — passing it into `describeWorkbenchFailure` (which
expects `unknown` and re-runs `classifyRpcError` internally) would hit the
`!(err instanceof ConnectError)` fallback and collapse every kind to `'unknown'` with a
garbage `"[object Object]"` message. Rendered `searchState.failure.message` directly
instead — the review's own minimal alternative, which is correct here rather than merely a
fallback. Added a regression test: a Files client that rejects renders
`file-picker-failure` with the error message; verified it fails (no such element exists)
against the unfixed component.

### WR-08: `debounced-rpc.ts`'s `dispose()` aborts without invalidating the request identity

**Files modified:** `web/src/lib/debounced-rpc.ts`, `web/tests/debounced-rpc.test.ts`,
`web/build/*` (rebuilt)
**Commit:** `601069d4`
**Applied fix:** Added `requestId += 1` to `dispose()`, mirroring the below-minimum path's
existing identical line four lines above it. Added a regression test where an in-flight
request's promise settles (with an abort-shaped rejection) *after* `dispose()` is called,
asserting `onFailure` is never invoked; verified it fails (onFailure called once) against
the unfixed `dispose()`.

## Skipped Issues

None — all ten in-scope findings were fixed.

## Verification

Every fix's `git stash`-based RED verification is recorded individually in its commit
message. Final gate run after all ten fixes (main checkout, `gsd/v0.12.0-local-graph-ui`,
`workflow.use_worktrees: false`):

| Gate | Result |
|---|---|
| `task web:test` | PASS — 268 of 268 tests passed (baseline 260; +8 new regression tests, one per finding requiring one) |
| `cd web && pnpm check` | 1020 files, 0 errors, 0 warnings (baseline 1019; +1 new test file) |
| `GOTOOLCHAIN=go1.26.5 task test:unit` | all packages ok |
| `GOTOOLCHAIN=go1.26.5 task web:drift` | PASS — 103 source files / 31 output files (rebuilt twice, after the WR-04/05 DataTable change and again after the WR-07/08 FilePicker/debounced-rpc changes) |
| `task web:components:drift` | PASS — 50 vendored component files across 8 components (network-dependent scratch-tree regeneration; ran successfully in this session) |
| `GOTOOLCHAIN=go1.26.5 task proto:drift` | PASS — 4 generated files byte-identical (after CR-02's proto comment regeneration) |
| `task web:render-cost` | PASS — 18.29ms initial render / 10.73ms sort-toggle median (thresholds 400ms/200ms) |
| `GOTOOLCHAIN=go1.26.5 task web:lockfile` | PASS — 233 packages, lockfileVersion '9.0', 233/233 integrity-bearing |
| `GOTOOLCHAIN=go1.26.5 task web:deps:strict` | PASS — strictDepBuilds true, allowBuilds committed (0 entries, 0 denials) |
| `GOTOOLCHAIN=go1.26.5 task web:audit` | PASS — 0 advisories |

No hard constraint was violated: the `GetHealthResponse` wire (16 fields) was not
renumbered, removed, or repurposed — only a doc comment was added to field 12 and the
three generated artifacts regenerated to match. `readonly_test.go`'s `mutatingVerbs` guard
was not touched. `proto:drift`'s `nfiles -lt 4` floor and `web:components:drift`'s 8-file/
2-component floors were not raised. No test was deleted; the one previously-tautological
Go assertion (CR-02) was replaced with a real one, not removed. No frozen wire-oracle
transcript or CLI golden was re-baselined. No new dependency was added.

---

_Fixed: 2026-08-30T03:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
