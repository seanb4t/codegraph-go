---
phase: 01-foundation-storage-schema-parser-strategy
fixed_at: 2026-07-10T22:35:00Z
review_path: .planning/phases/01-foundation-storage-schema-parser-strategy/01-REVIEW.md
iteration: 2
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-07-10T22:35:00Z
**Source review:** .planning/phases/01-foundation-storage-schema-parser-strategy/01-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 2 (CR-05 Critical, WR-05 Warning — the 2 new regressions from the previous fix pass)
- Fixed: 2
- Skipped: 0

This iteration addresses only the two newly-introduced regressions surfaced by the re-review. The 8 findings from the prior review (CR-01..CR-04, WR-01..WR-04) were already fixed in iteration 1 and are not re-touched here. WR-06 (the acknowledged `Commit()`-vs-`Close()` residual race) and the 3 Info findings remain intentionally out of scope, per the review's own deferred/carried-over classification.

## Fixed Issues

### CR-05: `pebbleStore.Close()` (and `pebbleReader.Close()`) not idempotent

**Files modified:** `internal/graphstore/pebble_store.go`, `internal/graphstore/store_test.go`
**Commit:** `1400441`
**Applied fix:** Guarded `pebbleStore.Close()` with the existing `s.closed` `atomic.Bool` via `Swap(true)`, short-circuiting to `nil` on any call after the first instead of re-delegating to `pebble.DB.Close()` (which panics on a second invocation). Added an equivalent `atomic.Bool` guard to `pebbleReader` and applied the same `Swap`-based short-circuit to `pebbleReader.Close()`, since `*pebble.Snapshot.Close()` has the identical double-close-panics shape. Added `TestStoreCloseIsIdempotent` and `TestReaderCloseIsIdempotent`, each calling `Close()` twice and asserting no panic and a nil second-call error.

### WR-05: `parser.Tree.Close()` not hardened against double-close

**Files modified:** `internal/parser/parser.go`, `internal/parser/parser_test.go`, `internal/parser/cgo/parser_cgo_test.go`
**Commit:** `16cf579`
**Applied fix:** Added a `closeOnce sync.Once` field to `Tree` and changed `Close()` to invoke `closeFn` via `t.closeOnce.Do(t.closeFn)` instead of calling it directly, mirroring the guard already used by `CGoParser.Close()` for the same native-double-free risk class. This covers both sequential and concurrent double-close, matching the bar the review explicitly asked to match. Added `TestTreeCloseIsIdempotent` (stub-level, asserts `closeFn` invoked exactly once across two `Close()` calls) and `TestCGoTreeCloseIsSafeToCallTwice` (real CGo tree-sitter tree, asserts no crash/double-free calling `Close()` twice on a live parsed tree).

## Skipped Issues

None — both in-scope findings were fixed cleanly; code matched the review's cited state exactly.

## Verification

- `CGO_ENABLED=1 go build ./...` — clean, after each fix and at the end.
- `CGO_ENABLED=1 go test ./... -race -count=1` — all packages pass:
  - `internal/graphstore` — ok
  - `internal/graphstore/archtest` — ok
  - `internal/parser` — ok
  - `internal/parser/cgo` — ok
  - `internal/schema` — ok
- Both fixes are logic-level idempotency guards (Tier 1 re-read + Tier 2 build/test), not classified as ambiguous logic-correctness findings by the review, so no "requires human verification" flag was applied — the fixes directly implement the review's own suggested code, and the added double-close tests directly exercise the fixed behavior (including a real CGo native double-free proof for WR-05).

---

_Fixed: 2026-07-10T22:35:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
