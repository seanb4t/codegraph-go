---
phase: 01-foundation-storage-schema-parser-strategy
reviewed: 2026-07-10T18:30:00Z
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
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-10T18:30:00Z
**Depth:** deep
**Files Reviewed:** 18
**Status:** issues_found

## Summary

This is a re-review after the automated fix pass (commits `cddda72`, `af11942`, `1866ffe`, `8e627dc`, `038f9db`, `4b01c71`) that addressed the prior review's 4 Critical + 4 Warning findings.

**All 8 previously-reported Critical/Warning findings are verified fixed, by reading the resulting code (not just trusting commit messages):**

- **CR-01 (tree memory leak):** Fixed. `parser.Tree` now carries a `closeFn`, `Tree.Close()` invokes it, `CGoParser.Parse` wires `parser.NewTree(result, result.Close)`, and both `parser_cgo_test.go` call sites now `defer tree.Close()`.
- **CR-02 (nil native tree):** Fixed. `CGoParser.Parse` returns `ErrParseFailed` when `p.inner.Parse` returns `nil`, before ever wrapping it in a `*parser.Tree`.
- **CR-03 (unbounded Import allocation):** Fixed. `export.go` adds `maxImportRecordBytes` (64 MiB) and rejects any declared record length above it before `make([]byte, length)`.
- **CR-04 (panic on closed DB):** Fixed for the three call sites named in the finding. `Snapshot`, `NewWriter`, and `Export` all check a new `atomic.Bool` guard and return `ErrClosed` instead of reaching Pebble's own closed-DB panic path. The one nuance the fixer explicitly flagged in a code comment — `Commit()` on a `Writer` obtained just before `Close()` is still unguarded — matches the original finding's own fallback acceptance criterion ("at minimum, document the current behavior as a known limitation if not fixed in this phase"). See WR-02 below for why this residual is still worth tracking, not re-raising as Critical.
- **WR-01 (batch never Close()'d):** Fixed. `Commit()` now closes the batch after committing, and `Writer.Close()` is a new interface method for abandoned batches, correctly swallowing Pebble's own redundant `pebble.ErrClosed`.
- **WR-02 (archtest test-file bypass):** Fixed. `packages.Config.Tests` is now `true`, and `stripTestVariant` correctly normalizes the three `PkgPath` shapes `Tests: true` introduces before the prefix check.
- **WR-03 (concurrency test doesn't force overlap):** Fixed. A `readersReady`/`start`-channel barrier now forces every reader to be poised immediately before the writer's `Commit()` call, so the "no torn write" assertion is exercised against a genuine race rather than passing vacuously on lucky scheduling.
- **WR-04 (CGoParser.Close() concurrent double-free):** Fixed. `closeOnce sync.Once` replaces the plain `bool`.

**However, deep tracing into `pebble/v2`'s own source surfaced one new Critical regression and two related Warnings that the fix pass introduced or left adjacent to the areas it touched:**

The CR-04 fix added a "closed" guard to `Snapshot`/`NewWriter`/`Export` but not to `pebbleStore.Close()` itself — and `pebble.DB.Close()` panics if called a second time. I reproduced this empirically (not just from reading the vendored source): calling `store.Close()` twice on a fresh store panics with `pebble: closed`. This directly undermines the stated intent of the CR-04 fix ("stop panicking on a closed store") for the single most common way a `Close()`-based API gets misused — a `defer store.Close()` paired with an explicit early-return `store.Close()`. `pebbleReader.Close()` (wrapping `*pebble.Snapshot.Close()`) has the identical unguarded double-close-panics-in-Pebble shape. Separately, `parser.Tree.Close()` — introduced by this very fix pass — was not given the same idempotency hardening that `CGoParser.Close()` received in the same commit for the same underlying risk (a native/C-level double-free), despite the doc comment explicitly describing the "superseded oldTree" reparse pattern that makes double-close a realistic future trap.

The 3 Info items from the prior review (`IN-01` naming, `IN-02` missing reserved ranges on File/Meta, `IN-03` capture.sh's blunt timestamp regex) were intentionally out of the fix pass's scope and remain open below, carried over unchanged.

## Critical Issues

### CR-05: `pebbleStore.Close()` is not idempotent — a second call panics the process

**File:** `internal/graphstore/pebble_store.go:76-79`
**Issue:** The CR-04 fix added `s.closed.Store(true)` to `Close()` so that *subsequent* `Snapshot`/`NewWriter`/`Export` calls observe the guard — but `Close()` itself never checks `s.closed` before delegating to `s.db.Close()`:
```go
func (s *pebbleStore) Close() error {
	s.closed.Store(true)
	return s.db.Close()
}
```
`pebble.DB.Close()` panics on a second invocation (verified in `pebble/v2@v2.1.6/db.go:1669-1672`: `if err := d.closed.Load(); err != nil { panic(err) }`, and again after acquiring `d.mu`). I confirmed this empirically against this exact codebase:
```go
store, _ := Open(t.TempDir())
_ = store.Close()   // ok
_ = store.Close()   // panics: "pebble: closed"
```
`GraphStore.Close()`'s doc comment ("Close releases the underlying engine handle") gives no idempotency contract, and nothing else in this package's fixed code path stops a caller from making the same double-`Close()` mistake that CR-04 was written to eliminate elsewhere — e.g. a `defer store.Close()` in a long-lived caller (the MCP server this store is built for) combined with an explicit `Close()` on a shutdown error path. This is a crash, not a returned error, exactly the class of defect CR-04 was meant to close off, and it is reachable through the very method the fix touched.
**Fix:** Guard `Close()` with the same atomic, short-circuiting on an already-closed store:
```go
func (s *pebbleStore) Close() error {
	if s.closed.Swap(true) {
		return nil // already closed; avoid pebble's own double-Close panic
	}
	return s.db.Close()
}
```
(`pebbleReader.Close()` has the identical root cause — `*pebble.Snapshot.Close()` also panics via `if db == nil { panic(ErrClosed) }` on a second call, per `pebble/v2@v2.1.6/snapshot.go:134-138` — see WR-03.)

## Warnings

### WR-05: `parser.Tree.Close()` is not hardened against double-close, unlike `CGoParser.Close()` fixed in the same commit

**File:** `internal/parser/parser.go:61-67`, `internal/parser/cgo/parser_cgo.go:70-75`
**Issue:** Commit `cddda72` fixed `CGoParser.Close()`'s double-free risk with `sync.Once`, but the new `Tree.Close()` it introduced in the same commit has no equivalent guard:
```go
func (t *Tree) Close() error {
	if t == nil || t.closeFn == nil {
		return nil
	}
	t.closeFn()
	return nil
}
```
`t.closeFn` is `result.Close` from `go-tree-sitter`'s `tree_sitter.Tree.Close()`, which unconditionally calls `C.ts_tree_delete(t._inner)` with no nil-check on the pointer itself (verified in `go-tree-sitter@v0.25.0/tree.go:113-117`). Calling `parser.Tree.Close()` twice — a plausible mistake given the package doc's own description of the "superseded oldTree" pattern during incremental reparse, where a caller must remember to close exactly the right tree at exactly the right time — double-frees the underlying C tree, which is undefined behavior in C (the crash-isolation contract this package's own doc already calls out as unrecoverable via Go's `recover()`). This is a real gap given the sibling fix in the same commit treated the identical risk class (native double-free) as worth a `sync.Once`-style guard for `Parser.Close()` but not for `Tree.Close()`.
**Fix:** Nil out `closeFn` after invoking it so repeat sequential calls are safe no-ops (matches the pattern already used for `oldTree`/`closeFn == nil` checks elsewhere in this type):
```go
func (t *Tree) Close() error {
	if t == nil || t.closeFn == nil {
		return nil
	}
	fn := t.closeFn
	t.closeFn = nil
	fn()
	return nil
}
```
If concurrent (not just sequential) double-close needs to be safe too — matching the bar `CGoParser.Close()` was held to — embed a `sync.Once` instead, since a plain field write here is not race-free under concurrent callers.

### WR-06: Residual `Commit()`-vs-`Close()` race is real but explicitly accepted/documented — track, don't block

**File:** `internal/graphstore/pebble_store.go:20-27`, `internal/graphstore/batch.go:66-72`
**Issue:** A `Writer` obtained via `NewWriter()` just before `store.Close()` runs can still have its `Commit()` reach `pebble.applyInternal`, which panics if the DB has since closed (verified in `pebble/v2@v2.1.6/db.go:834-837`). The fixer's own comment in `pebble_store.go` acknowledges this: "Commit on a Writer obtained just before Close is a harder race to close entirely without additional coordination and is not guarded by this sentinel." This matches the original CR-04 finding's own stated fallback ("at minimum, document the current behavior as a known limitation if not fixed in this phase") — so this is not a new, silently-reintroduced defect, and I am not re-raising it as Critical. It is, however, a real residual gap worth carrying forward into Phase 2+ planning: fully closing it requires reference-counting outstanding `Writer`/`Reader` handles against the store's `Close()`, which is legitimately out of this foundation phase's scope.
**Fix (deferred, not blocking):** When a graceful-shutdown path is built (the MCP server / watcher daemon in a later phase), coordinate `Close()` with any outstanding `Writer` via a `sync.WaitGroup` or reference count so `Close()` blocks until in-flight `Commit()` calls finish, rather than racing them.

## Info

### IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match

**File:** `internal/graphstore/store.go:51-54`, `internal/graphstore/keys.go:82-93`
**Issue:** Carried over, unchanged, out of this fix pass's scope. The parameter name `srcPrefix` and its doc ("every edge whose source is srcPrefix") read as "starts with," but the length-prefixed key encoding makes this an exact-source match only — `edgeSrcPrefix("a")` never matches a real source `"ab"`. Correct behavior, misleading name.
**Fix:** Rename to `src` (matching `edgeSrcPrefix`'s own parameter name) or add an explicit doc line clarifying exact-match semantics.

### IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`

**File:** `internal/schema/graph.proto:66-91`
**Issue:** Carried over, unchanged, out of this fix pass's scope. `Node`/`Edge` reserve `50 to 59` for future annotation fields; `File`/`Meta` do not, which is inconsistent with the stated additive-only, pre-agreed-slot discipline.
**Fix:** Add a `reserved` range to `File` and `Meta` for consistency, or document why they were deliberately excluded.

### IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns

**File:** `testdata/golden/capture.sh:74`
**Issue:** Carried over, unchanged, out of this fix pass's scope. `sed -E 's/[0-9]{13}(\.[0-9]+)?/<EPOCH_MS>/g'` rewrites any 13-digit substring anywhere in the dump, not just the named timestamp columns.
**Fix:** Scope the substitution to known timestamp columns (column-aware replacement) rather than a blanket numeric-width match.

---

_Reviewed: 2026-07-10T18:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
