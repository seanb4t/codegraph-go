---
phase: 10-index-health-the-coverage-denominator
reviewed: 2026-09-13T05:00:00Z
depth: deep
files_reviewed: 47
files_reviewed_list:
  - internal/graphstore/batch.go
  - internal/graphstore/excludedfile_test.go
  - internal/graphstore/export_test.go
  - internal/graphstore/export.go
  - internal/graphstore/keys.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/store.go
  - internal/indexer/coverage_fixture_test.go
  - internal/indexer/discover_test.go
  - internal/indexer/discover.go
  - internal/indexer/discoverexclusion_test.go
  - internal/indexer/discoverexclusion.go
  - internal/indexer/pipeline_test.go
  - internal/indexer/pipeline.go
  - internal/indexer/resolve_test.go
  - internal/indexer/resolve.go
  - internal/indexer/sync_coverage_test.go
  - internal/indexer/sync.go
  - internal/indexer/synccoverage_test.go
  - internal/indexer/synccoverage.go
  - internal/indexer/testdata/coverage/.hidden/y.go
  - internal/indexer/testdata/coverage/go.mod
  - internal/indexer/testdata/coverage/main.go
  - internal/indexer/testdata/coverage/notes.md
  - internal/indexer/testdata/coverage/tagged.go
  - internal/indexer/testdata/coverage/vendor/x.go
  - internal/query/coverage_test.go
  - internal/query/coverage.go
  - internal/query/errors.go
  - internal/query/expand_test.go
  - internal/query/files_status_test.go
  - internal/query/gather_test.go
  - internal/query/scoring_test.go
  - internal/query/search_test.go
  - internal/query/seeding_test.go
  - internal/query/traverse_test.go
  - internal/schema/exclusion_test.go
  - internal/schema/exclusion.go
  - internal/schema/graph.proto
  - internal/schema/meta_commit_test.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/coverage_test.go
  - internal/uiserver/coverage.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - web/src/lib/components/health/CoverageSection.svelte
  - web/src/lib/health-view.ts
  - web/src/routes/health/+page.svelte
  - web/tests/health-page.test.ts
  - web/tests/health-view.test.ts
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-09-13T05:00:00Z
**Depth:** deep
**Files Reviewed:** 47
**Status:** issues_found

## Summary

Re-review (iteration 2) of the two fix commits applied against the prior review's findings: `725ba1fb` (CR-01: resume `CoverageRows` paging by key-order position via `RawKey()`/`FileKey`/`ExcludedFileKey`) and `fbdf17f3` (WR-01: embed `Meta.LastSyncUnixMs` as a generation marker in the page token; a stale generation aborts with `connect.CodeAborted`; the client retries the whole walk once, then renders a text-only "may be incomplete" notice).

**CR-01 verification — the fix holds.** Traced both segment loops (`f` then `x`) directly:

- Each namespace's iterator (`IterateFiles`/`IterateExcludedFiles`) is bounded to its own single-byte prefix (`f`/`c`) via `pebble.IterOptions{LowerBound, UpperBound}` (`internal/graphstore/pebble_store.go:289-310`), so `bytes.Compare(rawKey, cursorKey)` only ever compares keys within one namespace — cross-segment key-byte ordering (`'c' < 'f'` in ASCII) is a non-issue because the two segments are never compared against each other, only walked as two separate bounded scans.
- `RawKey()` (`it.iter.Key()`) is read and consumed synchronously inside `bytes.Compare` within the same loop iteration, never stored past that statement — safe under Pebble's "valid until next Next/Close" contract; no dangling-slice bug.
- Traced the specific segment-transition case called out for this iteration: cursor is the **last row of the `f` segment** and gets deleted. The `f`-segment loop's `skipping` flag never flips to `false` (no remaining `f` key sorts after the deleted cursor's recomputed key), so the loop drains to `fit.Next() == false` with zero rows emitted from `f` — correct, since everything in `f` after the cursor's position is (by construction) nothing. Control then falls into the `x`-segment block with `skipping := cursorSeg == 'x'` evaluating `false` (cursor was `'f'`), so `x` is walked from its own start, exactly matching what should follow the exhausted `f` segment. No token can point into "the wrong segment": `nextToken`'s `seg` byte is derived from `last.Kind` — the actual segment of the last row emitted, not a guess (`internal/query/coverage.go:313-322`).
- `TestCoverageRowsSurvivesCursorMutationBetweenPages` (`internal/query/coverage_test.go:1024-1106` for the `f`-segment case, `:1108-1189` for the `x`-segment case) genuinely mutates the store via a second `Writer.Commit()` between two page fetches from **fresh** `Engine`/snapshot instances (mirroring `uiserver.GetCoverage`'s per-call snapshot discipline), and asserts both no dropped rows (full-set path-equality across all pages) and a non-empty continuation token in the `x`-segment case where the pre-fix code would have falsely reported `NextPageToken == ""`. Ran it directly (`go test ./internal/query/... ./internal/graphstore/... ./internal/uiserver/...`) — all green.

**WR-01 verification — the mechanism works, with one residual gap (see WR-01 below).** The `CodeAborted` mapping (`internal/uiserver/handlers.go:107-120`) follows the existing `errors.Is`-only discipline exactly (no message-text matching, one translation site), and the aborted message itself (`"coverage: index changed since the previous page was fetched; retry from the first page"`) carries no dynamic data, so it cannot leak an absolute path (T-10-05 discipline preserved). `fetchAllCoverageRows` (`web/src/lib/health-view.ts:335-359`) retries the whole paged walk exactly once via a narrow `isRetryableCoverageAbort` check scoped to `Code.Aborted`, and both the first attempt and the retry are bounded by `COVERAGE_MAX_PAGES` — no infinite loop is possible even if a hostile/buggy server never returns an empty token. `CoverageSection.svelte`'s incomplete notice (`:76-80`) is text-only — no `{@html}`, no interactive control, consistent with D-11.

Independently confirmed (not just per the fix report's own claims):
- `git diff 352a6b8e..HEAD -- internal/uiproto internal/schema/graph.proto` is empty — no proto change.
- `task -s web:drift` — PASS, source and output hashes both MATCH.
- `TestUIProtoFieldNumbersAreStableAndUnique` — 49 messages / 200 fields, unchanged; `wantUIServiceMethods` still 16 entries including `GetCoverage`.
- `go build ./...`, `go vet ./...`, and `go test ./internal/query/... ./internal/graphstore/... ./internal/uiserver/...` all clean.

One residual **Warning** remains on the generation marker's own soundness (see WR-01 below) — not a regression of the two commits, but a limitation of the chosen mechanism worth recording rather than silently accepting. IN-01 and IN-02 from the prior review are unchanged and carried forward as Info.

## Warnings

### WR-01: The wall-clock, millisecond-resolution generation marker can silently fail to detect a store change that happens within the same millisecond as the previous write

**File:** `internal/query/coverage.go:182,205-207`, `internal/indexer/resolve.go:822`, `internal/indexer/sync.go:204,445`
**Issue:** The chosen generation marker is `time.Now().UnixMilli()`, stamped at all three coverage-bearing write sites. `CoverageRows` treats two page tokens as "same generation, safe to resume" purely by integer equality of this value (`cursorGeneration != generation`, `coverage.go:205`). If two coverage-affecting commits (e.g. two back-to-back `Sync` runs, or a `Sync` that lands between a client's two page fetches on a fast loopback connection) happen to stamp the identical millisecond — a real possibility given Go's `time.Now()` resolution is platform-dependent, and unavoidable if the host's wall clock is ever stepped backward by NTP — the generation check passes even though the store changed underneath the walk. This does not reintroduce CR-01's row-dropping bug (that fix is now purely position-based and correct per-page regardless of generation), but it does silently defeat WR-01's own stated purpose: the client is told "no inconsistency detected" for a walk that may, in fact, straddle two different index states, with no way to notice. The residual window is narrow (sub-millisecond commit timing) but the whole point of adding this marker was to make that class of race detectable rather than silent — a marker that can alias defeats that goal exactly in the case that matters (rapid successive Syncs, e.g. a debounced batch of file-watcher events firing close together).
**Fix:** Replace or augment the wall-clock marker with a monotonically-incrementing counter stored in `Meta` (e.g. `Meta.CoverageGeneration int64`, incremented by exactly 1 at each of the same three write sites the review already enumerated) rather than derived from `time.Now()`. This guarantees two-distinct-writes-imply-two-distinct-generations regardless of clock resolution or backward clock steps, with the same "zero new write-path plumbing beyond one field" cost the current design already accepted for `LastSyncUnixMs`. If reusing `LastSyncUnixMs` is preferred to avoid a schema change, at minimum document the residual gap in the field's own doc comment so a future reader does not assume the generation check is airtight.

## Info

### IN-01: `coverageExtractionDetail`'s absolute-path scrub is a plain substring replace, not anchored to a path boundary

**File:** `internal/query/coverage.go:331-337`
**Issue:** Unchanged from the prior review. `strings.ReplaceAll(detail, repoRoot, ".")` replaces every literal occurrence of `repoRoot`, not just a leading-path occurrence, and depends on `e.repoRoot`'s capitalization/symlink-resolution matching whatever was embedded in the underlying parser error verbatim (e.g. a `/tmp` vs `/private/tmp` class of mismatch on macOS). Carried forward as low-severity: pre-existing pattern, not introduced by this phase, and the current test suite confirms the common case redacts correctly.
**Fix:** No action required unless a broader repoRoot-redaction audit is already planned; if one is, include this call site.

### IN-02: `unsupportedExtensionDetail`/exclusion `detail` strings are unbounded on the write path

**File:** `internal/indexer/discoverexclusion.go:78-100`
**Issue:** Unchanged from the prior review. No explicit byte bound is applied at write time the way `coverageDetailMaxBytes` bounds the read-time `Detail` for extraction failures. Low-severity: filesystem path-component limits already bound this in practice.
**Fix:** No action required now; noted for completeness since coverage rows are now exposed over the wire to a browser.

### IN-03: `fetchAllCoverageRows`'s second (retry) attempt discards any rows it collected before aborting a second time

**File:** `web/src/lib/health-view.ts:335-359`
**Issue:** When the retried walk (`attempt()` called a second time) itself throws `Code.Aborted` partway through — e.g. after successfully collecting several pages, then hitting a third concurrent Sync — the final `.catch` returns a hardcoded `{ known: true, rows: [], incomplete: true }` rather than whatever partial `rows` the second attempt had already accumulated before it threw. This is not a correctness bug (the `incomplete: true` flag already tells the UI not to trust the list, and rendering zero rows under that flag is a defensible, conservative choice), but it is a minor, easily-avoidable loss of otherwise-good partial data that a reader might not expect from the variable name `rows` no longer meaning "what we have" once an error path is taken.
**Fix:** Optional: thread the partially-collected rows out of the second `attempt()` call (e.g. via a mutable accumulator passed in, or by having `attempt()` return partial rows alongside the thrown error) so the incomplete state can still show whatever was legitimately fetched. Not required before shipping.

---

_Reviewed: 2026-09-13T05:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
