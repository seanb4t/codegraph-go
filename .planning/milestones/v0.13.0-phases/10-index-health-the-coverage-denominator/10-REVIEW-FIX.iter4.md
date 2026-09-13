---
phase: 10-index-health-the-coverage-denominator
fixed_at: 2026-09-13T13:57:11Z
review_path: .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
iteration: 3
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 10: Code Review Fix Report

**Fixed at:** 2026-09-13T13:57:11Z
**Source review:** .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
**Iteration:** 3

**Summary:**
- Findings in scope (critical + warning), this iteration: 1 (CR-01)
- Fixed: 1
- Skipped: 0

IN-01, IN-02, and IN-03 are Info and out of `critical_warning` scope; left untouched. This was an explicitly authorized single extra targeted pass beyond the normal review-fix loop cap, scoped to CR-01 only.

## Carried Forward From Iterations 1–2

The 10-REVIEW.md re-review (iteration 3) confirmed iterations 1 and 2's mechanics are sound in isolation — recorded here for a single source of truth on this phase's fix history:

- **CR-01 (iteration 1 shape)** (CoverageRows silently truncated a page / falsely signaled "no more pages" on a cursor mutated between page fetches) — fixed in commit `725ba1fb`. Re-verified across iterations 2 and 3: both segment-transition cases traced correctly, `TestCoverageRowsSurvivesCursorMutationBetweenPages` still green.
- **WR-01 (iteration 1 shape)** (the truncation failure mode was undetectable by clients) — fixed in commit `fbdf17f3` by embedding a generation marker (originally `Meta.LastSyncUnixMs`) in the page token and mapping a mismatch to `connect.CodeAborted`.
- **WR-01 (iteration 2)** (the wall-clock, millisecond-resolution generation marker could alias two distinct commits landing in the same host millisecond) — fixed in commit `10e9c028` by replacing the marker with `Meta.coverage_generation`, a monotonically-incrementing counter read-modify-written by exactly 1 at each of the three coverage-bearing meta-write sites. Re-verified by the iteration-3 reviewer as mechanically sound in isolation — but see this iteration's CR-01 below for the residual gap the iteration-2 reviewer's own re-review found in that fix's own justification.

## Fixed Issues

### CR-01 (iteration 3): `Meta.coverage_generation` reset to 1 on every full re-index, defeating the exact aliasing guarantee WR-01 (iteration 2) was built to provide

**Files modified:** `internal/cli/index.go`, `internal/cli/index_test.go` (new), `internal/indexer/pipeline.go`, `internal/indexer/pipeline_test.go`, `internal/indexer/resolve.go`, `internal/indexer/resolve_test.go`
**Commit:** `83666cc`

**Applied fix:** Took the reviewer's preferred option (option 2 in 10-REVIEW.md's Fix section — "stop wiping `Meta.coverage_generation`'s effective history at all" by threading the prior store's generation through as a floor), not the random-salt alternative (option 1). Option 2 was simpler and kept `Meta` self-contained, matching the discipline `writeGraph` already applies for the Sync-backfill-over-an-existing-store case.

- `internal/cli/index.go`'s `newIndexCmd` now calls a new `priorCoverageGeneration(storeDir)` helper **before** `os.RemoveAll(storeDir)`: it opens the store read-only via `graphstore.Open` → `Snapshot()` → `GetMeta()` (no `Writer` ever opened), reads `CoverageGeneration`, and fully `Close()`s the store before returning — releasing Pebble's lock file so the immediately-following `RemoveAll` never contends with a lock this call is still holding (mirrors `needsFileIndexBackfill`'s own isolated open/close discipline in `internal/indexer/sync.go`). Every failure mode (missing store, no prior Meta record, corrupted/unreadable store) tolerates to `0` — this call can only ever *raise* the floor `writeGraph` would otherwise stamp on its own, never lower it, so a failure here cannot regress behavior below iteration 2's fix.
- The resulting value is threaded into `indexer.Run(root, storeDir, indexer.Options{..., CoverageGenerationFloor: coverageGenerationFloor})` — a new `Options.CoverageGenerationFloor int64` field (`internal/indexer/pipeline.go`), documented as a no-op at its zero value (every other caller — `codegraph init`'s fresh store, and `Sync`'s D-02b backfill/incremental paths, which never wipe `storeDir`) is unaffected.
- The floor is threaded from `Options` through `run()`'s `resolve(...)` call, through the `resolveFunc` type and `Resolve`'s own signature, down to `writeGraph` (`internal/indexer/resolve.go`), which now stamps `max(priorGeneration, coverageGenerationFloor) + 1` instead of unconditionally `priorGeneration + 1`. `priorGeneration` here remains `writeGraph`'s own pre-existing Snapshot-based read of the (post-wipe, now-empty) TARGET store — the fix's contribution is purely the `coverageGenerationFloor` lower bound, sourced from a store that has since been deleted.
- All eight existing `writeGraph`/`Resolve` call sites in `internal/indexer/resolve_test.go` and the one `resolveFunc`-shaped test stub in `internal/indexer/pipeline_test.go` were updated to the new signature, passing `0` (a no-op, preserving each test's original pre-fix semantics exactly) since none of them exercise the wipe-then-rebuild scenario this fix addresses.

**New test (per the orchestrator's required regression coverage):**

`TestIndexForceRebuildBumpsCoverageGeneration` (`internal/cli/index_test.go`) drives the exact real-world sequence CR-01 describes, through the actual CLI command tree (not two `writeGraph` calls against a store that was never wiped):

1. `codegraph init` on a fixture augmented with a `vendor/` directory (an OS-independent second coverage-gap row, on top of `.codegraph/` itself always being one `EXCLUSION_REASON_DIR_DOTPREFIX` row) — guaranteeing >= 2 coverage rows so a `PageSize: 1` request always yields a real `NextPageToken`.
2. Opens the resulting store via `query.OpenAt`, calls `CoverageRows(CoverageRowsOptions{PageSize: 1})`, and keeps the returned `NextPageToken` as the "pre-wipe" token a live client would be holding.
3. Runs `codegraph index --force` — the real `RemoveAll` + `MkdirAll` + `indexer.Run` sequence this finding is about.
4. Asserts the rebuilt store's `Meta.CoverageGeneration` (read directly) is **strictly greater** than the first run's.
5. Re-opens the rebuilt store and replays the pre-wipe token through `CoverageRows`, asserting `errors.Is(err, query.ErrAborted)` — the token is correctly rejected rather than silently accepted against an unrelated graph generation.

Traced against the pre-fix code: with the fix reverted, both `init` and the first `index --force` compute a floor of `0` (the field would not exist) and `writeGraph`'s own Snapshot read of the freshly-wiped store returns `ErrNotFound` → `priorGeneration = 0` → stamps `1` both times, so step 4's `gen2 <= gen1` assertion (`1 <= 1`) would fail — confirming the test is load-bearing, not vacuous.

## Skipped Issues

None — the one in-scope finding (CR-01, iteration 3) was fixed.

## Verification

All verification below ran in the **main working tree** (`.planning/config.json` has `workflow.use_worktrees: false`, the documented opt-out — no isolated worktree was created for this fix run, matching iteration 2's own verification note) — the numbers are reproducible directly from this checkout.

- `GOTOOLCHAIN=go1.26.6 go build ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go vet ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./internal/indexer/... ./internal/query/... ./internal/uiserver/...` — all green, including the new `TestIndexForceRebuildBumpsCoverageGeneration` and every required carried-forward test: `TestCoverageGenerationIncrementsByExactlyOnePerCommit`, `TestWriteGraphStampsMonotonicCoverageGeneration`, `TestCoverageRowsSurvivesCursorMutationBetweenPages`, `TestCoverageSourceNeverWalksDisk`.
- `task proto:drift` — PASS, 4/4 generated files byte-identical to the pinned toolchain's regeneration. No `.proto` file was touched by this fix (schema field 10, `coverage_generation`, already existed from iteration 2 — this fix is pure Go plumbing, no new field).
- `task -s web:drift` — PASS (source and output hashes both matched). No `web/` file was touched by this fix.
- No `codegraph ui` process was left running; the working tree is clean except this report and the pre-existing, untouched `10-REVIEW.md` / `10-REVIEW.iter2.md` / `10-REVIEW.iter3.md` / `10-REVIEW-FIX.iter2.md` / `10-REVIEW-FIX.iter3.md` artifacts, which this fixer was instructed not to modify.

No source files were left in a broken state; no uncommitted source changes remain outside this report.

---

_Fixed: 2026-09-13T13:57:11Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 3_
