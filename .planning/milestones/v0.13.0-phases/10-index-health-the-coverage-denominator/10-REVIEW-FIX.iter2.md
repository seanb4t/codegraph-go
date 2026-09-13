---
phase: 10-index-health-the-coverage-denominator
fixed_at: 2026-09-13T04:39:04Z
review_path: .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 10: Code Review Fix Report

**Fixed at:** 2026-09-13T04:39:04Z
**Source review:** .planning/phases/10-index-health-the-coverage-denominator/10-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (critical + warning): 2
- Fixed: 2
- Skipped: 0

Info findings IN-01 and IN-02 were out of `critical_warning` scope and were left untouched, per the orchestrator's instructions.

## Fixed Issues

### CR-01: CoverageRows silently truncates a page (and can falsely signal "no more pages") when the cursor row is deleted/changed between two page fetches

**Files modified:** `internal/graphstore/store.go`, `internal/graphstore/pebble_store.go`, `internal/graphstore/keys.go`, `internal/query/coverage.go`, `internal/query/coverage_test.go`
**Commit:** `725ba1fb`

**Applied fix:** Resumed the paged walk by POSITION in the store's own key order instead of value-equality against the cursor's decoded path.

- Added `RawKey() []byte` to `graphstore.FileIterator` and `graphstore.ExcludedFileIterator` (implemented on the only production implementers, `pebbleFileIterator`/`pebbleExcludedFileIterator` — no test-only fake implements these interfaces directly, so no other call site needed updating).
- Exported `graphstore.FileKey(path)` / `graphstore.ExcludedFileKey(path)`, thin wrappers around the existing unexported `fileKey`/`excludedFileKey` builders — these are pure functions of `path` alone, so a caller can recompute the cursor's own key bytes independent of whether that record still exists.
- Rewrote `CoverageRows`' two segment loops (`internal/query/coverage.go`) to compare each row's `RawKey()` against the cursor's recomputed key via `bytes.Compare`, flipping `skipping` to `false` the instant a row's key sorts strictly after the cursor's — this correctly resumes at the next row after a deleted/mutated cursor's position in both the extraction-failed (`f`) and excluded-file (`x`) segments, and can no longer produce the "worse" false-`NextPageToken==""` case, since the transition is now driven by total key order rather than requiring the exact prior value to reappear.
- Added `TestCoverageRowsSurvivesCursorMutationBetweenPages` (two subtests: `f`-segment and `x`-segment cursor deletion), which seeds a store directly via a `graphstore.Writer`, fetches a page, mutates the cursor's own record via a second `Writer.Commit()` (simulating a concurrent Sync), then fetches from a **fresh** `Engine`/snapshot — mirroring `uiserver.GetCoverage`'s per-call snapshot discipline (SRV-04). Confirmed RED against the prior implementation (both bug modes reproduced exactly as CR-01 described: rows silently dropped, and — for the `x`-segment case — an empty `NextPageToken` while a row remained) and GREEN after the fix.
- Existing test `TestCoverageRowsOrderingAndPagingAreStable` stayed green unmodified.

### WR-01: CoverageRows' silent-truncation failure mode is undetectable by clients even though the page contract implies completeness

**Files modified:** `internal/query/coverage.go`, `internal/query/errors.go`, `internal/query/coverage_test.go`, `internal/uiserver/handlers.go`, `internal/uiserver/coverage_test.go`, `web/src/lib/health-view.ts`, `web/src/lib/components/health/CoverageSection.svelte`, `web/src/routes/health/+page.svelte`, `web/tests/health-view.test.ts`, `web/tests/health-page.test.ts`, `web/build/**`
**Commit:** `fbdf17f3`

**Design decision (documented per the orchestrator's instructions):** the review offered two alternative shapes — (A) an additive `GetCoverageResponse` field (e.g. `bool truncated`/`int64 total_rows`), or (B) a snapshot/generation marker the page token itself carries, detected server-side and reported as `CodeFailedPrecondition`/`CodeAborted`. **I chose (B)** and specifically reused the store's existing `Meta.LastSyncUnixMs` field as the generation marker, because:

- `LastSyncUnixMs` is already stamped at all three coverage-bearing write sites (`indexer.Run`'s from-scratch write via `resolve.go`, and `Sync`'s two commit paths in `sync.go`) — using it required **zero new write-path plumbing**, no new Meta field, and no risk of missing a fourth write site later.
- It required **no `ui.proto` change** at all: the page token is already opaque to the client (`T-10-04`), so embedding an 8-byte generation inside it is invisible on the wire contract. `TestUIProtoFieldNumbersAreStableAndUnique` (`internal/uiserver/readonly_test.go`) stayed green **unmodified** (still 49 messages / 200 fields), confirming no proto regen, no `task proto:gen`/`task proto:drift`, and no field-number fixture update were needed — the review's own phrasing ("ui.proto MAY add a response field") was conditional on choosing shape (A), which I did not.
- The "signal" for the client is simply the `connect.CodeAborted` Connect error code on the RPC itself — a code every Connect response already carries — rather than a new success-response field the client must remember to check.

**Applied fix:**

- `internal/query/coverage.go`: `encodeCoverageToken`/`decodeCoverageToken` now frame a token as `[kind byte][generation, 8 bytes big-endian][path bytes]` (previously `[kind byte][path bytes]`). `CoverageRows` reads the store's current `meta.GetLastSyncUnixMs()` once per call and, for any non-empty incoming token, rejects a disagreeing embedded generation as the new `query.ErrAborted` classification (`"coverage: index changed since the previous page was fetched; retry from the first page"`) — before any segment walk runs.
- `internal/query/errors.go`: added `ErrAborted`/`abortedf`, following the existing `ErrInvalidArgument`/`invalidArgumentf` pattern exactly (never a `%w` wrap, per `classifiedError`'s documented contract).
- `internal/uiserver/handlers.go`: `mapEngineError` gained one new case, `errors.Is(err, query.ErrAborted) -> connect.CodeAborted`, alongside the existing `ErrNotFound`/`ErrInvalidArgument`/`ErrStoreLocked` arms — no second translation site.
- `web/src/lib/health-view.ts`: `fetchAllCoverageRows` now retries the **whole** paged walk from the first page exactly once when a page fetch throws a `ConnectError` with `code === Code.Aborted` (checked via a narrow, single-purpose `isRetryableCoverageAbort` helper — not a second general-purpose Connect-error classifier; `rpc-errors.ts` remains the one page-level failure classifier, D-04). If the retry also aborts, it resolves to `{ known: true, rows: [], incomplete: true }` instead of throwing or silently asserting a complete list. Any other error still propagates unchanged.
- `web/src/lib/components/health/CoverageSection.svelte`: `RowsState`'s `loaded` variant gained an `incomplete: boolean` field; the section renders a text-only "Coverage rows may be incomplete — the index changed while loading. Reload to confirm." notice when true — no interactive control, consistent with D-11's text-only-remedy rule.
- `web/src/routes/health/+page.svelte`: threads `result.incomplete` from `fetchAllCoverageRows`'s resolved value into `rowsState`.
- New tests: `TestCoverageRowsAbortsWhenGenerationChangedBetweenPages` (Engine level, with a same-generation replay positive control), `TestGetCoverageAbortsWhenGenerationChangedBetweenPages` (real listener, asserting `connect.CodeOf(err) == connect.CodeAborted`), three new cases in `health-view.test.ts` (retry-then-succeed, retry-then-incomplete, non-Aborted errors propagate without retrying), and one new case in `health-page.test.ts` asserting the `health-coverage-rows-incomplete` test id renders end-to-end. All generation bumps in tests are **hand-set literals**, never `time.Now()`, so none of the new tests can flake on two commits landing in the same host millisecond.
- Updated the pre-existing `TestCoverageRowsRejectsMalformedTokens` malformed-token table for the new 9-byte-minimum token shape (added a `too short to hold the generation field` case) and updated its positive-control token to embed the real current generation via `eng.IndexMeta()`.
- Regenerated `web/build/**` via `task web:build` (source-files=115, output-files=32) and confirmed `task web:drift` clean (source and output hashes both MATCH) since `web/src` files changed.

## Skipped Issues

None — both in-scope findings were fixed.

## Verification

All verification below ran in the **main working tree** (this repo has `workflow.use_worktrees: false` in `.planning/config.json`, so no isolated worktree was created for this fix run per the orchestrator's documented opt-out) — the numbers are reproducible directly from this checkout.

- `GOTOOLCHAIN=go1.26.6 go build ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go vet ./...` — clean.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/query/... ./internal/graphstore/... ./internal/uiserver/... ./internal/indexer/...` — all green, including every pre-existing coverage test and both new regression tests.
- `GOTOOLCHAIN=go1.26.6 go test ./...` (full suite) — all green **except** `internal/daemon`'s `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` (first full run) / `TestConvergenceTwoSessions` (isolated re-run of the package), both timing-sensitive tests unrelated to this fix — `internal/daemon` has zero diff in either commit, and re-running `internal/daemon` standalone reproduced a **different** failing test each time under host load, confirming pre-existing flakiness rather than a regression. Documented here per the reviewer instructions to report pre-existing failures separately rather than mask them.
- `pnpm -C web test` — 570 passed (566 baseline + 4 new: 3 in `health-view.test.ts`, 1 in `health-page.test.ts`).
- `pnpm -C web check` — `COMPLETED 1170 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` (gate: RC=0 and exactly one `0 errors` match).
- `task web:build && task -s web:drift` — build succeeded, drift check `PASS` (source and output hashes both matched), `web/build/**` committed in the WR-01 commit.
- `ui.proto` was **not** modified (see WR-01's design-decision note above), so `task proto:gen`/`task proto:drift` and the `readonly_test.go` field-number fixture were correctly left untouched — `TestUIProtoFieldNumbersAreStableAndUnique` confirmed this by staying green with the same 49 messages / 200 fields count before and after.

No source files were left in a broken state; no uncommitted changes remain outside this report.

---

_Fixed: 2026-09-13T04:39:04Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
