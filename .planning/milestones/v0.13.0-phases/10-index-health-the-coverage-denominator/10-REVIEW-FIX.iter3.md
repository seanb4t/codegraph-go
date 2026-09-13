---
phase: 10-index-health-the-coverage-denominator
fixed_at: 2026-09-13T05:02:44Z
review_path: .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
iteration: 2
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 10: Code Review Fix Report

**Fixed at:** 2026-09-13T05:02:44Z
**Source review:** .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope (critical + warning), this iteration: 1 (WR-01)
- Fixed: 1
- Skipped: 0

IN-01, IN-02, and IN-03 are Info and out of `critical_warning` scope; left untouched per the orchestrator's instructions.

## Carried Forward From Iteration 1

The 10-REVIEW.md re-review (iteration 2) confirmed both of iteration 1's fixes hold under direct tracing and re-run tests — recorded here for a single source of truth on this phase's fix history:

- **CR-01** (CoverageRows silently truncated a page / falsely signaled "no more pages" on a cursor mutated between page fetches) — fixed in commit `725ba1fb`. Re-verified by the iteration-2 reviewer: both segment-transition cases traced correctly, `TestCoverageRowsSurvivesCursorMutationBetweenPages` still green.
- **WR-01 (iteration 1 shape)** (the truncation failure mode was undetectable by clients) — fixed in commit `fbdf17f3` by embedding a generation marker (originally `Meta.LastSyncUnixMs`) in the page token and mapping a mismatch to `connect.CodeAborted`. Re-verified by the iteration-2 reviewer as mechanically sound, but with one residual gap: a wall-clock, millisecond-resolution marker can alias two distinct commits onto the same value, silently defeating the mechanism's own purpose. That residual gap was reopened as **this iteration's WR-01** (see below) and is now fixed.

## Fixed Issues

### WR-01 (iteration 2): The wall-clock, millisecond-resolution generation marker can silently fail to detect a store change that happens within the same millisecond as the previous write

**Files modified:** `internal/schema/graph.proto`, `internal/schema/graph.pb.go`, `internal/schema/meta_commit_test.go`, `internal/indexer/resolve.go`, `internal/indexer/resolve_test.go`, `internal/indexer/sync.go`, `internal/indexer/sync_coverage_test.go`, `internal/indexer/sync_determinism_test.go`, `internal/query/coverage.go`, `internal/query/coverage_test.go`, `internal/uiserver/coverage_test.go`, `web/src/lib/health-view.ts` (doc comment only), `web/build/**`
**Commit:** `10e9c028`

**Applied fix:** Took the real fix the review offered (not the doc-comment-only fallback): replaced the wall-clock marker with a monotonically-incrementing counter.

- Added `int64 coverage_generation = 10` to `schema.Meta` in `graph.proto` — additive, next free field number after `has_coverage = 9`; `SchemaVersion` untouched at `1`. Doc comment mirrors `has_coverage`'s own additive/compat wording, explicitly stating why a counter replaces a wall-clock marker.
- Regenerated with `task proto:gen` (both proto surfaces); `task proto:drift` clean (4/4 generated files byte-identical, header included). `ui.proto` was **not** touched — the field lives entirely in `graph.proto`'s `Meta`, and `internal/uiproto/uiv1/ui.proto` / `internal/uiserver/readonly_test.go` have an empty `git diff` from before this fix, confirmed directly (`TestUIProtoFieldNumbersAreStableAndUnique` unaffected).
- Extended `TestKnownMetaFieldNumbersAreStable` (`internal/schema/meta_commit_test.go`) with `{"coverage_generation", 10}` in the same commit as the proto regen, per D-02a's discipline.
- `CoverageGeneration` is incremented by exactly 1, read-modify-write, at each of the same three meta-write sites `has_coverage` is stamped at — never a second writer:
  - `internal/indexer/sync.go`'s small-commit write site (mtime/coverage-only refresh): `newMeta.CoverageGeneration = meta.GetCoverageGeneration() + 1`, reading `meta` (the prior `Meta` already loaded via `r0.GetMeta()` at the top of `Sync`).
  - `internal/indexer/sync.go`'s full incremental-commit write site: same pattern, same prior `meta`.
  - `internal/indexer/resolve.go`'s `writeGraph` (the from-scratch commit path, used both by `indexer.Run` and by `Sync`'s D-02b backfill delegation): `writeGraph` did not previously read any prior `Meta`, so it now opens a `store.Snapshot()` **Reader** (never a second `Writer`) before staging its own `Writer` batch, reads the prior `CoverageGeneration` (treating `graphstore.ErrNotFound` — a genuinely fresh store — as `0`, mirroring `needsFileIndexBackfill`'s own established pattern), closes the Reader, and stamps `priorGeneration + 1` into the meta record staged on its single `Writer.Commit()`. This makes a from-scratch rewrite over an **existing** store (the backfill case) continue the same monotonic sequence a live page token may already be pinned to, rather than resetting to 1 and potentially colliding with a still-valid outstanding token.
- `internal/query/coverage.go`: `CoverageRows` now reads `meta.GetCoverageGeneration()` instead of `meta.GetLastSyncUnixMs()` as the token's embedded/compared generation. `encodeCoverageToken`/`decodeCoverageToken`'s doc comments updated to describe the counter, not the clock. The token's on-wire *shape* (`[kind byte][generation, 8 bytes big-endian][path bytes]`) is unchanged — only which `Meta` field feeds the 8-byte generation value changed, so no `ui.proto`/wire-contract change was needed here either (D-10 discretion held).
- Compat: the `has_coverage` short-circuit in both `CoverageSummary` and `CoverageRows` runs **before** `CoverageGeneration` is ever read — an old graph (field unset, decodes as `0`) still reports `Known: false` and never reaches the token-minting code path, so there is no compatibility gap for a pre-this-fix graph.
- Fixed a stale doc comment in `web/src/lib/health-view.ts` naming the retired mechanism (`LastSyncUnixMs`) — comment-only, the token stays opaque to this file's logic, so no behavioral change. Ran the full web gate anyway since the file was touched: `pnpm -C web check` (0 errors), `pnpm -C web test` (570 passed), `task web:build && task -s web:drift` (PASS) — `web/build/**` committed in the same commit.

**New tests (per the orchestrator's required regression coverage):**

1. `TestCoverageGenerationIncrementsByExactlyOnePerCommit` (`internal/indexer/sync_coverage_test.go`) — drives three real, back-to-back coverage-affecting commits (`Run`'s from-scratch write, then two of `Sync`'s incremental small-commit writes) and asserts each step's `CoverageGeneration` is exactly the previous one plus 1. No clock injection was needed or attempted: the counter's correctness does not depend on timing, which is precisely the property being proven — the commits are driven back-to-back specifically so they may land within the same host millisecond on a fast machine, and the assertion holds regardless.
2. `TestCoverageRowsAbortsOnGenerationChangeEvenWhenLastSyncUnixMsIsUnchanged` (`internal/query/coverage_test.go`) — mints a page token at generation 1 with a fixed `LastSyncUnixMs`, then proves two things with the same fixture: (a) bumping `LastSyncUnixMs` alone while holding `CoverageGeneration` fixed does **not** trip the abort (inverse control — the check is not clock-keyed), and (b) bumping `CoverageGeneration` to 2 while holding `LastSyncUnixMs` at the exact same value the token was minted with **does** trip `ErrAborted` — the literal scenario a wall-clock marker would have missed.
3. Updated `TestCoverageRowsAbortsWhenGenerationChangedBetweenPages` and `TestGetCoverageAbortsWhenGenerationChangedBetweenPages` (Engine and real-listener levels) to bump `CoverageGeneration` instead of `LastSyncUnixMs`, since the mechanism itself moved.
4. Updated `TestCoverageRowsRejectsMalformedTokens`'s positive-control token to embed `meta.GetCoverageGeneration()`.
5. Added `TestWriteGraphStampsMonotonicCoverageGeneration` (`internal/indexer/resolve_test.go`, three subtests: fresh store starts at 1; an existing store's prior generation is read and incremented; a non-`ErrNotFound` Snapshot/GetMeta failure propagates and aborts before any `Writer` is opened) — this required adding a minimal `stubReader` implementing `graphstore.Reader` to the existing `stubStore` test double, since `writeGraph` now calls `store.Snapshot()`.
6. `exportNormalized` in `internal/indexer/sync_determinism_test.go` now also zeroes `Meta.CoverageGeneration` before comparing two independently-built stores (`TestSyncEqualsReindex`), for the same reason it already zeroes `LastSyncUnixMs`: the counter tracks each store's own write-event history (2 meta-writes via seed-then-touch-Sync vs. 1 meta-write via a single from-scratch `Run`), not graph content — comparing it raw would fail a determinism gate for a reason unrelated to determinism.

## Skipped Issues

None — the one in-scope finding (WR-01, iteration 2) was fixed.

## Verification

All verification below ran in the **main working tree** (this repo has `workflow.use_worktrees: false` in `.planning/config.json`, so no isolated worktree was created for this fix run per the documented opt-out) — the numbers are reproducible directly from this checkout.

- `GOTOOLCHAIN=go1.26.6 go build ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go vet ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... ./internal/graphstore/... ./internal/uiserver/... ./internal/indexer/... ./internal/schema/...` — all green, including every pre-existing coverage/determinism test and all six new/updated tests listed above.
- `task proto:gen` then `task proto:drift` — PASS, 4/4 generated files byte-identical (header included) to the pinned toolchain's regeneration.
- `git diff 352a6b8e..HEAD -- internal/uiproto internal/schema/graph.proto` scoped to `internal/uiproto` alone is empty — `ui.proto` was not touched by this fix; only `graph.proto` changed.
- `pnpm -C web check` — `COMPLETED 1170 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` (gate: RC=0 and exactly one `0 errors` match).
- `pnpm -C web test` — 570 passed (unchanged from iteration 1's post-fix count; `health-view.ts`'s only change this iteration was a doc comment).
- `task web:build && task -s web:drift` — build succeeded, drift check `PASS` (source and output hashes both matched); `web/build/**` committed in the same commit as the `health-view.ts` comment fix.
- No `codegraph ui` process was left running; the working tree is clean except this report and the pre-existing, untouched `10-REVIEW.md` / `10-REVIEW.iter2.md` / `10-REVIEW-FIX.iter2.md` artifacts, which this fixer was instructed not to modify.

No source files were left in a broken state; no uncommitted source changes remain outside this report.

---

_Fixed: 2026-09-13T05:02:44Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
