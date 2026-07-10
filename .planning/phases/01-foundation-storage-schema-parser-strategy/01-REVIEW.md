---
phase: 01-foundation-storage-schema-parser-strategy
reviewed: 2026-07-10T19:15:00Z
depth: deep
files_reviewed: 18
files_reviewed_list:
  - internal/graphstore/store.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/batch.go
  - internal/graphstore/export.go
  - internal/graphstore/keys.go
  - internal/graphstore/store_test.go
  - internal/graphstore/export_test.go
  - internal/graphstore/keyenc_test.go
  - internal/graphstore/archtest/import_graph_test.go
  - internal/parser/parser.go
  - internal/parser/parser_test.go
  - internal/parser/cgo/parser_cgo.go
  - internal/parser/cgo/parser_cgo_test.go
  - internal/schema/graph.proto
  - internal/schema/meta.go
  - internal/schema/roundtrip_test.go
  - testdata/golden/golden_test.go
  - testdata/golden/capture.sh
findings:
  critical: 0
  warning: 1
  info: 3
  total: 4
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-10T19:15:00Z
**Depth:** deep
**Files Reviewed:** 18
**Status:** issues_found (only pre-accepted residuals remain)

## Summary

This is the final re-review after the second fix pass (commits `1400441`, `16cf579`), which addressed the two regressions (CR-05, WR-05) surfaced by the previous re-review.

**CR-05 is genuinely resolved.** `pebbleStore.Close()` (`internal/graphstore/pebble_store.go:77-82`) now guards on `s.closed.Swap(true)`, short-circuiting to `nil` on any call after the first, before ever reaching `pebble.DB.Close()`'s own double-close panic. `pebbleReader.Close()` (`pebble_store.go:152-157`) received the identical treatment via its own new `closed atomic.Bool` field, guarding `*pebble.Snapshot.Close()`'s equivalent panic. I traced both guards against their call sites: `Snapshot()`/`NewWriter()`/`Export()` still check `s.closed.Load()` (unchanged from the CR-04 fix) and correctly observe `true` once `Close()` has run, so there is no window where the "closed" guard and the "idempotent Close" guard disagree — a single `atomic.Bool` per struct serves both purposes consistently. `TestStoreCloseIsIdempotent` and `TestReaderCloseIsIdempotent` (`store_test.go:198-230`) exercise exactly the double-`Close()` sequence that previously panicked, and assert `nil` (not just "no panic") on the second call — a real behavioral assertion, not a placebo test. Ran the full suite under `-race`; both pass.

**WR-05 is genuinely resolved.** `parser.Tree` gained a `closeOnce sync.Once` field (`internal/parser/parser.go:44`), and `Close()` now runs `t.closeOnce.Do(t.closeFn)` instead of invoking `closeFn` directly (`parser.go:73-79`). This is the stronger of the two options the prior review offered (a plain `closeFn = nil` reassignment would have been sequential-safe only; `sync.Once` is also race-free under concurrent callers, matching the bar `CGoParser.Close()` was already held to in the same commit). `TestTreeCloseIsIdempotent` (`parser_test.go:77-90`) asserts the close callback fires exactly once across two `Close()` calls (a counting assertion, stronger than a no-panic check), and `TestCGoTreeCloseIsSafeToCallTwice` (`parser_cgo_test.go:77-99`) exercises the same guard through the real CGo backend against an actual native tree-sitter `*Tree`, not just a stub. Both pass under `-race`.

**No new regression from the pass-2 guards.** I checked the specific risk vectors called out for this review:
- `Close()`'s `closed.Swap(true)` vs. `closed.Load()` in `Snapshot`/`NewWriter`/`Export`: same underlying `atomic.Bool`, monotonic one-way transition (false→true, never reset) — no ordering or staleness issue is possible with this shape.
- `sync.Once` on `Tree`: no reentrancy — `closeFn` (`result.Close`, the native tree-sitter release) does not call back into `Tree.Close()`, so no self-deadlock potential.
- No path where a resource is now silently *not* freed: in both `pebbleStore.Close()`/`pebbleReader.Close()`, the guard only short-circuits on the *second and later* call — the first call still unconditionally reaches `s.db.Close()`/`r.snap.Close()`. In `Tree.Close()`, `sync.Once.Do` guarantees exactly one execution of `closeFn` across any number of calls, sequential or concurrent — never zero.
- Both fix-pass-2 commits are narrowly scoped (`pebble_store.go` + its test; `parser.go`/`parser_cgo.go` + their tests) — `batch.go`, `export.go`, `keys.go`, and the `GraphStore`/`Reader`/`Writer` interfaces in `store.go` are untouched, so there is no ripple risk into the areas the first fix pass already touched.

`go build ./...` and `go test ./... -race` both pass cleanly across all five packages.

**Remaining items are exactly the pre-accepted residuals, nothing new:**
- **WR-06** (documented, non-blocking): the `Commit()`-vs-`Close()` race on a `Writer` obtained just before `store.Close()` remains explicitly out of scope for this phase, per the fixer's own comment in `pebble_store.go:24-26` and the original finding's stated fallback criterion. Not re-raised as Critical.
- **IN-01, IN-02, IN-03**: unchanged, carried over from the original review, explicitly out of scope for both fix passes.

No new Critical or Warning findings were identified in this pass.

## Warnings

### WR-06: Residual `Commit()`-vs-`Close()` race is real but explicitly accepted/documented — track, don't block

**File:** `internal/graphstore/pebble_store.go:20-27`, `internal/graphstore/batch.go:66-72`
**Issue:** A `Writer` obtained via `NewWriter()` just before `store.Close()` runs can still have its `Commit()` reach `pebble.applyInternal`, which panics if the DB has since closed. The fixer's own comment in `pebble_store.go` acknowledges this: "Commit on a Writer obtained just before Close is a harder race to close entirely without additional coordination and is not guarded by this sentinel." This matches the original CR-04 finding's own stated fallback ("at minimum, document the current behavior as a known limitation if not fixed in this phase"), so this is not a newly-introduced or silently-reintroduced defect. It is a real residual gap worth carrying into Phase 2+ planning: fully closing it requires reference-counting outstanding `Writer`/`Reader` handles against the store's `Close()`.
**Fix (deferred, not blocking):** When a graceful-shutdown path is built (the MCP server / watcher daemon in a later phase), coordinate `Close()` with any outstanding `Writer` via a `sync.WaitGroup` or reference count so `Close()` blocks until in-flight `Commit()` calls finish, rather than racing them.

## Info

### IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match

**File:** `internal/graphstore/store.go:51-54`, `internal/graphstore/keys.go:82-93`
**Issue:** Carried over, unchanged, out of scope for both fix passes. The parameter name `srcPrefix` and its doc ("every edge whose source is srcPrefix") read as "starts with," but the length-prefixed key encoding makes this an exact-source match only — `edgeSrcPrefix("a")` never matches a real source `"ab"`. Correct behavior, misleading name.
**Fix:** Rename to `src` (matching `edgeSrcPrefix`'s own parameter name) or add an explicit doc line clarifying exact-match semantics.

### IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`

**File:** `internal/schema/graph.proto:66-91`
**Issue:** Carried over, unchanged, out of scope for both fix passes. `Node`/`Edge` reserve `50 to 59` for future annotation fields; `File`/`Meta` do not, which is inconsistent with the stated additive-only, pre-agreed-slot discipline.
**Fix:** Add a `reserved` range to `File` and `Meta` for consistency, or document why they were deliberately excluded.

### IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns

**File:** `testdata/golden/capture.sh:74`
**Issue:** Carried over, unchanged, out of scope for both fix passes. `sed -E 's/[0-9]{13}(\.[0-9]+)?/<EPOCH_MS>/g'` rewrites any 13-digit substring anywhere in the dump, not just the named timestamp columns.
**Fix:** Scope the substitution to known timestamp columns (column-aware replacement) rather than a blanket numeric-width match.

---

_Reviewed: 2026-07-10T19:15:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
