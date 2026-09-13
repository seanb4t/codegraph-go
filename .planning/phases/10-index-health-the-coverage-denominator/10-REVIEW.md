---
phase: 10-index-health-the-coverage-denominator
reviewed: 2026-09-13T00:00:00Z
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
  critical: 1
  warning: 1
  info: 2
  total: 4
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-09-13T00:00:00Z
**Depth:** deep
**Files Reviewed:** 47
**Status:** issues_found

## Summary

Reviewed the Phase 10 coverage-denominator write path (discover → resolve/sync → graphstore → query → uiserver → Svelte) end to end, including all nine deep-depth cross-file concerns called out in the review scope. The write-path invariants the phase is built around hold up under direct tracing and are well covered by the accompanying test suite:

- Exactly one `graphstore.Writer` per commit point in every one of Run's/Sync's three write sites (from-scratch `writeGraph`, Sync's small mtime/coverage-only commit, Sync's full incremental commit) — confirmed by direct read and by the `git diff` against the pre-Phase-10 base.
- `Meta.has_coverage` is stamped independently and unconditionally at all three meta-write sites, never inferred from `c/` key presence; `CoverageSummary`/`CoverageRows` short-circuit to `Known:false` before touching any iterator when the flag is unset.
- The discovered-count denominator and every per-file exclusion reason come from one `DiscoverAll` walk; `internal/query/coverage.go` contains zero filesystem calls (verified by direct read; the file also carries its own `TestCoverageSourceNeverWalksDisk` structural guard).
- `internal/query` does not import `internal/indexer` from production code (only two `_test.go` files do, which is the codebase's own accepted, documented pattern for seeding fixtures — see `sync.go`'s own comment on why a production import would create a cycle).
- Export/Import carries the `c/` namespace as record kind 5 explicitly; an old export (no kind-5 frames, no `has_coverage` on the Meta frame) imports cleanly into a fresh store that reports `Known:false`, consistent with D-06.
- Fixture hygiene holds: `testdata/coverage/` has no committed symlink, the oversized file and dangling symlink are generated at test time, and every coverage test that seeds a fixture lets `indexer.Run`/`graphstore.Open` create the store directory rather than pre-creating it (avoiding the phantom `.codegraph` dot-prefix exclusion 10-02 fixed).
- The read-only guard (`wantUIServiceMethods`, 16 entries, set-equality both directions) and the `mutatingVerbs` negative check both correctly cover the new `GetCoverage` rpc, including a decoy (`GetIndexCoverage`) proving the discriminator is live.
- The Svelte `CoverageSection` component uses text interpolation exclusively — no `{@html}`, no interactive controls, no event handlers — and `fetchAllCoverageRows` is bounded by `COVERAGE_PAGE_SIZE`/`COVERAGE_MAX_PAGES` with a hard error on runaway pagination.

One BLOCKER was found in `CoverageRows`' cross-segment cursor-resume logic (`internal/query/coverage.go`): under concurrent disk mutation between two page fetches, a resume token whose exact row has since disappeared or changed can cause the remainder of that token's segment to be silently dropped with no error and no indication to the caller that pagination is incomplete — the opposite of this phase's own stated invariant (I explicitly did not find this covered by any existing test, including the dedicated `TestCoverageRowsOrderingAndPagingAreStable`, which never mutates the store between page fetches).

## Critical Issues

### CR-01: CoverageRows silently truncates a page (and can falsely signal "no more pages") when the cursor row is deleted/changed between two page fetches

**File:** `internal/query/coverage.go:215-276`
**Issue:**

`CoverageRows` resumes a paged walk by re-walking each segment (`f` = extraction-failed File records, `x` = ExcludedFile records) from its own start and skipping forward until it observes the **exact previously-returned row** (`path == cursorPath`) once more, then resumes emitting from the next row:

```go
skipping := cursorSeg == 'f'
for len(rows) <= pageSize && fit.Next() {
    ...
    if skipping {
        if path == cursorPath {
            skipping = false
        }
        continue
    }
    rows = append(rows, CoverageRow{...})
}
```

This design assumes the cursor's row is still present, unchanged, in the store when the next page is fetched. Because `internal/uiserver`'s `GetCoverage` opens a fresh `Engine`/snapshot per call (documented in this same file's own header comment: *"two consecutive CoverageRows page calls against a live server may be answered from different snapshots"*), and the frontend's `fetchAllCoverageRows` walks pages back-to-back with real network round trips in between, a background `Sync` (daemon debounce, or a concurrent `codegraph sync`) that prunes or upserts the cursor's own record between two page fetches is squarely in-scope, not a hypothetical race.

When this happens:

1. **`f`-segment case:** if the cursor's File record no longer has errors (fixed) or was deleted, `path == cursorPath` never matches. `skipping` stays `true` for the rest of the `fit` iteration, so **every remaining extraction-failed row that has not yet been returned is silently dropped from this page** — not deferred, dropped: the walk then falls through into the `x` segment and never revisits `f` again (the `!filterByReason && cursorSeg != 'x'` guard at the top of the function permanently excludes the `f` segment once the cursor has moved past it).
2. **`x`-segment case (worse):** if the cursor's ExcludedFile record was pruned or changed reason, the same non-match leaves `skipping == true` for the rest of the `xit` iteration. Since `x` is the last segment, `rows` ends the call empty, `len(rows) > pageSize` is false, so **`NextPageToken` is left empty** — the caller (and therefore `fetchAllCoverageRows`/the Coverage UI) is told pagination is *complete* when in fact an unbounded number of trailing rows were silently discarded.

Both cases violate the invariant this phase's own comments repeatedly assert for the coverage feature (never guess, never silently drop, D-01/D-07/D-14) and are exactly what the code-review scope for this file asked to be verified ("a page token pointing at a path deleted between pages ... cannot loop or skip an entire segment"). No existing test exercises a store mutation between two page fetches — `TestCoverageRowsOrderingAndPagingAreStable` pages a static, never-mutated fixture, so this defect is not caught anywhere in the current suite.

**Fix:** Resume by position, not by value-equality on the mutable payload. The cleanest fix is to make the token carry the store's own key bytes (or a byte-comparable encoding of `(path)` compared with `bytes.Compare` against the iterator's raw key, not the decoded field) so "have I passed the cursor" is answered by ordering, not by requiring the exact same record to still exist:

```go
// Sketch: compare against the iterator's raw key using the store's own
// key order, so a row's disappearance or mutation cannot break the
// comparison — only its position in the key space matters.
for len(rows) <= pageSize && fit.Next() {
    if skipping {
        if bytesGreaterThan(fit.RawKey(), cursorKey) {
            skipping = false
        } else {
            continue
        }
    }
    ...
}
```

If exposing raw keys through the `graphstore.Reader` iterator interfaces is too large a change for this fix, an acceptable interim mitigation is to detect the "walked the whole segment without ever finding cursorPath" condition explicitly and treat it as "resume from the top of this segment" (re-emit everything, deduplicating client-side by path) rather than silently emitting zero rows and moving on — that at minimum turns silent data loss into a bounded, detectable duplicate rather than an invisible gap, and must never allow the `x`-segment case to end with `NextPageToken == ""` when the walk stopped early because the cursor was never found.

## Warnings

### WR-01: `CoverageRows`' silent-truncation failure mode is undetectable by clients even though the page contract implies completeness

**File:** `internal/query/coverage.go:278-289`, `web/src/lib/health-view.ts:316-341`
**Issue:** Related to CR-01 but distinct: even setting aside the specific skip-logic bug, the wire contract gives the client no way to tell "this page is short because we're at the true end" apart from "this page is short because the server's resume walk silently gave up." `GetCoverageResponse` has no flag for "an inconsistency was detected mid-walk." `fetchAllCoverageRows` therefore has no way to surface a partial result as anything other than success. This means even a smaller-scope fix for CR-01 (e.g. detecting the not-found-cursor case and refusing further pagination with an error) still needs *some* signal reaching the client so the Coverage section can render "list may be incomplete" instead of quietly asserting completeness.
**Fix:** Once CR-01 is fixed at the server, consider adding a boolean such as `possibly_incomplete` to `GetCoverageResponse` (additive, D-02a-safe) for the case where the resume cursor could not be exactly re-located, so the UI can render an honest "coverage rows may be stale, reload to confirm" notice instead of asserting a false negative ("no more rows").

## Info

### IN-01: `coverageExtractionDetail`'s absolute-path scrub is a plain substring replace, not anchored to a path boundary

**File:** `internal/query/coverage.go:296-302`
**Issue:** `strings.ReplaceAll(detail, repoRoot, ".")` replaces every literal occurrence of `repoRoot` in the joined error string, not just a leading-path occurrence. In the extremely unlikely case that `repoRoot` is a very short string that also occurs as a substring elsewhere in an unrelated error message (e.g. a one-or-two-character checkout directory name that happens to also appear in a parser diagnostic's quoted snippet), the scrub could either over-redact unrelated text or, if the path capitalization/symlink-resolution differs between what was embedded in the error and what `e.repoRoot` holds (a known macOS `/tmp` vs `/private/tmp` class of mismatch), fail to redact at all, which is exactly the information-disclosure scenario T-10-05 exists to prevent. This is pre-existing risk surface inherited from the general "redact repoRoot" pattern elsewhere in the codebase, not new to this phase, and the existing test (`TestCoverageRowsDetailIsScrubbedAndBounded`) does confirm the common case redacts correctly on the current dev platform — flagging only because Phase 10 is the first caller to put this string on an unauthenticated loopback wire surface (`GetCoverage`) rather than a local CLI.
**Fix:** No action required unless a broader audit of `repoRoot`-based redaction across the codebase is already planned; if one is, this call site should be included.

### IN-02: `unsupportedExtensionDetail`/exclusion `detail` strings are unbounded on the write path

**File:** `internal/indexer/discoverexclusion.go:78-100`
**Issue:** `ExcludedFile.Detail` for `EXCLUSION_REASON_UNSUPPORTED_EXTENSION` is the file's own extension string, and for `EXCLUSION_REASON_SIZE_LIMIT` it's a rendered byte-count string — both effectively bounded by filesystem path-component length limits already, so this is not exploitable in practice, but there's no explicit bound applied at write time (`newExcludedFile`) the way `coverageDetailMaxBytes` bounds the read-time `Detail` for extraction failures in `internal/query/coverage.go`. A single degenerate filename (e.g. a very long dotfile crafted to have an enormous "extension" via repeated dots) could produce a `Detail` string well past what the UI comfortably renders, though the underlying filesystem's own path-length ceiling makes this a low-severity theoretical gap rather than a practical one.
**Fix:** No action required now; note only for completeness, since coverage rows are now exposed over the wire to a browser.

---

_Reviewed: 2026-09-13T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
