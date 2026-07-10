---
phase: 01-foundation-storage-schema-parser-strategy
reviewed: 2026-07-10T00:00:00Z
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
  critical: 4
  warning: 4
  info: 3
  total: 11
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-10
**Depth:** deep
**Files Reviewed:** 18
**Status:** issues_found

## Summary

Reviewed the storage (Pebble-backed `GraphStore`), schema (protobuf + `meta.go`), parser (`parser.Parser` seam + CGo tree-sitter backend), and golden-fixture capture layers at deep, cross-file depth, tracing call chains into `github.com/cockroachdb/pebble/v2` and `github.com/tree-sitter/go-tree-sitter` source to verify claims made in this codebase's own doc comments.

The key-encoding layer (`keys.go`) and its tests are genuinely strong — the length-prefix scheme is correct and the adversarial test suite in `keyenc_test.go` actually proves the injection-safety and range-scan-contiguity properties it claims, including on adjacent/prefix-colliding paths. The schema round-trip test (`roundtrip_test.go`) is a real forward-compat proof, not a smoke test.

However, four confirmed defects undermine the concurrency and resource-safety guarantees this phase exists to establish: (1) every parsed tree-sitter `Tree` leaks its underlying C memory forever, because the shared `parser.Tree` wrapper exposes no `Close()` and the CGo backend never calls the tree's own `Close()`; (2) the CGo backend can silently return a "successful" `*parser.Tree` that wraps a `nil` native tree, which later crashes on first use instead of surfacing an error; (3) the bulk `Import` path performs an unbounded `make([]byte, length)` allocation driven directly by an untrusted length prefix, an open DoS vector on the one code path in this phase explicitly meant to read externally-produced data; and (4) `Close()` racing with any in-flight `Snapshot()`/`NewWriter()`/`Commit()` call panics the whole process instead of returning an error, verified against `pebble/v2`'s own source. Additionally, the architecture-boundary test can be bypassed via a `_test.go`-only import, and the concurrency proof test doesn't force genuine reader/writer overlap.

## Critical Issues

### CR-01: Every parsed tree-sitter Tree leaks its underlying C memory

**File:** `internal/parser/cgo/parser_cgo.go:51-65`, `internal/parser/parser.go:19-47`
**Issue:** `tree_sitter.Tree` has its own `Close()` method, separate from `Parser.Close()` (confirmed in `go-tree-sitter@v0.25.0/tree.go:113` and `parser.go:63-67`, which shows `Parser.Close()` only calls `C.ts_parser_delete` on the parser — it never touches any previously-returned tree's memory). `CGoParser.Parse` does:
```go
result := p.inner.Parse(source, native)
return parser.NewTree(result), nil
```
and the shared `parser.Tree` type (`internal/parser/parser.go`) exposes no `Close()` method at all — only `Inner()`. There is therefore no way for any caller, anywhere in this codebase or a future one, to ever free the C-allocated tree-sitter tree for a parsed file. Every single `Parse()` call leaks native memory for the lifetime of the process. This directly contradicts the package doc's own "Resource contract" ("implementations allocate C or WASM-backed memory for parsers/trees... Callers MUST call Close()") — the contract describes tree cleanup but no such cleanup path exists.
**Fix:** Add `Close() error` to `parser.Tree` (delegating to the backend's stored tree via a backend-registered close func, or require backends to wrap trees in a type that implements it), and call it from every place a `*parser.Tree` is discarded (including on incremental-reparse, where the old tree is superseded by the new one and must be closed once the new tree is built). Minimal shape:
```go
type Tree struct {
    inner any
    closeFn func()
}
func NewTree(inner any, closeFn func()) *Tree { return &Tree{inner: inner, closeFn: closeFn} }
func (t *Tree) Close() error {
    if t == nil || t.closeFn == nil { return nil }
    t.closeFn()
    return nil
}
// in CGoParser.Parse:
return parser.NewTree(result, func() { result.Close() }), nil
```

### CR-02: CGo backend can return a "successful" Tree wrapping a nil native tree

**File:** `internal/parser/cgo/parser_cgo.go:63-64`
**Issue:** `tree_sitter.Parser.Parse`/`ParseWithOptions` returns `nil *tree_sitter.Tree` when the underlying `ts_parser_parse_with_options` C call returns `NULL` (confirmed in `go-tree-sitter@v0.25.0/parser.go:354-360`: `if cNewTree != nil { return newTree(cNewTree) }; return nil`). `CGoParser.Parse` does not check for this:
```go
result := p.inner.Parse(source, native)
return parser.NewTree(result), nil
```
`parser.NewTree(result)` always allocates a non-nil `*parser.Tree`, even when `result` is a nil `*tree_sitter.Tree`. Every existing caller pattern in this codebase (`parser_test.go`, `parser_cgo_test.go`) checks `tree == nil` to detect failure — that check will never fire here. A caller that later does `oldTree.Inner().(*tree_sitter.Tree)` (as `CGoParser.Parse` itself does for incremental reparse) and calls a method on the result (e.g. `.RootNode()`) will nil-deref inside CGo — an in-process crash with no Go-level `recover()` possible, which is exactly the tail-risk this package's own doc calls out as "accepted" for genuine grammar-scanner bugs, except this instance is triggered by the wrapper's own missing error propagation, not an adversarial grammar. No existing test exercises this path (there is no test that forces `ts_parser_parse_with_options` to return NULL, e.g. via a parser with no language set).
**Fix:**
```go
func (p *CGoParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
    if len(source) > parser.MaxSourceBytes {
        return nil, parser.ErrSourceTooLarge
    }
    var native *tree_sitter.Tree
    if oldTree != nil {
        if t, ok := oldTree.Inner().(*tree_sitter.Tree); ok {
            native = t
        }
    }
    result := p.inner.Parse(source, native)
    if result == nil {
        return nil, errors.New("cgo: tree-sitter parse returned no tree")
    }
    return parser.NewTree(result), nil
}
```

### CR-03: Import performs an unbounded allocation driven by an untrusted length prefix

**File:** `internal/graphstore/export.go:117-141`
**Issue:** `Import` reads each record's frame as `[kind][uvarint length][bytes]` with no ceiling on `length`:
```go
length, err := binary.ReadUvarint(br)
...
data := make([]byte, length)
if _, err := io.ReadFull(br, data); err != nil { ... }
```
`length` comes directly off an untrusted `io.Reader` (a migration source file, a corrupted store dump, or any future network-exposed import path) with no sanity check. A corrupted or crafted stream declaring a multi-gigabyte (or larger) length causes `make([]byte, length)` to either panic (`runtime: makeslice: len out of range`) or attempt a huge allocation that can OOM-kill the process — an unauthenticated denial-of-service on the one code path in this phase explicitly designed to read externally-produced data (D-06/ARCH-01's migration + backup framing). This is the same class of risk `parser.MaxSourceBytes` was introduced specifically to prevent on the parse path, but no analogous ceiling exists here.
**Fix:** Reject frames whose declared length exceeds a sane ceiling before allocating:
```go
const maxImportRecordBytes = 64 * 1024 * 1024 // or similar, documented ceiling
...
if length > maxImportRecordBytes {
    return fmt.Errorf("import: record length %d exceeds ceiling %d", length, maxImportRecordBytes)
}
data := make([]byte, length)
```

### CR-04: Close() races with Snapshot()/NewWriter()/Commit() panic instead of returning errors

**File:** `internal/graphstore/pebble_store.go:44-53`, `internal/graphstore/batch.go:61-63`
**Issue:** `pebbleStore.Snapshot()` calls `s.db.NewSnapshot()` and `pebbleWriter.Commit()` (via `NewWriter`'s `s.db.NewBatch()` and `batch.Commit(pebble.Sync)`) route into pebble internals that `panic()` — not return an error — once the underlying `*pebble.DB` is closed. Verified directly in `pebble/v2@v2.1.6`:
- `db.go:1635-1637` inside `NewSnapshot`: `if err := d.closed.Load(); err != nil { panic(err) }`
- `db.go:835-837` inside `applyInternal` (reached from `Batch.Commit`): `if err := d.closed.Load(); err != nil { panic(err) }`

None of the call sites in `pebble_store.go`/`batch.go` guard against this. The `GraphStore`/`Reader`/`Writer` interfaces in `store.go` all declare `error` return values for exactly these operations, implying callers can handle failure gracefully — but in practice, any caller that races a `Close()` (e.g. a normal graceful-shutdown path, or a background re-index goroutine finishing after the store was told to close) against an in-flight `Snapshot()`, `NewWriter()`, or `Commit()` will crash the entire process. For a long-lived MCP server process — the explicit consumer of this store — shutdown-vs-in-flight-request races are a realistic, not contrived, scenario.
**Fix:** Track closed state explicitly in `pebbleStore` (e.g. an `atomic.Bool`) and check it before delegating to Pebble in `Snapshot()`, `NewWriter()`, and `Export()`, returning a typed error (e.g. `ErrClosed`) instead of letting the call reach Pebble's panic path. `Commit()` on an in-flight `Writer` obtained just before `Close()` is a harder race to close entirely without additional coordination (e.g. a reference-counted close-barrier) — at minimum, document the current behavior as a known limitation if not fixed in this phase, since it currently silently contradicts the interface's error-based contract.

## Warnings

### WR-01: Writer batches are never Close()'d — defeats Pebble's batch pool, and there is no cleanup path for an abandoned Writer

**File:** `internal/graphstore/batch.go:61-63`, `internal/graphstore/store.go:82-106`
**Issue:** Pebble's own test suite consistently pairs `db.NewBatch()` with `defer b.Close()` (e.g. `pebble/v2@v2.1.6/batch_test.go:1291-1292`), because `NewBatch`/`NewIndexedBatch` pull from a `sync.Pool` (`batch.go:451: b := batchPool.Get().(*Batch)`) and `Batch.Close()` is what returns the batch to that pool for reuse. `pebbleWriter.Commit()` here only calls `w.batch.Commit(pebble.Sync)` and never `w.batch.Close()`. Worse, the `Writer` interface (`store.go`) has no `Close()`/cleanup method at all, so a caller that starts staging Puts, hits a marshal error, and abandons the `Writer` without calling `Commit()` has no way to release the batch either. This isn't a hard memory leak (the `Batch` struct is ordinary Go-GC'd memory once unreferenced), but it is a confirmed deviation from the library's documented usage idiom and forfeits the batch-recycling optimization on every single write in a write path meant to be exercised once per file-change/debounce window at scale.
**Fix:** Call `w.batch.Close()` after a successful (and, ideally, failed) `Commit()`, and add a `Close() error` method to the `Writer` interface so callers can release an abandoned batch:
```go
func (w *pebbleWriter) Commit() error {
    err := w.batch.Commit(pebble.Sync)
    if closeErr := w.batch.Close(); err == nil {
        err = closeErr
    }
    return err
}
```

### WR-02: archtest's bypass check can be defeated by a `pebble/v2` import placed only in a `_test.go` file

**File:** `internal/graphstore/archtest/import_graph_test.go:29-31`
**Issue:** `packages.Config{Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps}` never sets `Tests: true`. Per `golang.org/x/tools/go/packages`'s own doc comment on the `Tests` field, only when `Tests: true` does `packages.Load` include "the package as compiled for the test" variant — i.e. the compilation unit that in-package `_test.go` files (and `_test`-suffixed external test packages) contribute imports to. Without it, an import statement that appears *only* inside a `_test.go` file anywhere in the module — including a hypothetical bypass package entirely outside `internal/graphstore` that imports `pebble/v2` solely from its test file — is invisible to this check and will not be flagged. The phase context (and this test's own doc comment) specifically asks whether the boundary "can't be trivially bypassed"; as written, it can be, via test files.
**Fix:** Set `Tests: true` in the `packages.Config` and iterate `pkgs` accounting for the additional `[pkg.test]`/`pkg_test` package variants `Tests: true` introduces (their `PkgPath` typically still resolves cleanly under `isAllowedImporter`, but verify against the loaded output before landing).

### WR-03: TestConcurrentReadersSingleWriter does not force genuine overlap between a reader's Snapshot() and the writer's in-flight Commit()

**File:** `internal/graphstore/store_test.go:24-123`
**Issue:** The test spawns one writer goroutine and 16 reader goroutines with `sync.WaitGroup` but no barrier or synchronization forcing any reader's `Snapshot()` call to actually land while the writer's `Commit()` is mid-flight. Go's scheduler could serialize the whole run (all 16 readers' `Snapshot()`+`GetNode` loops complete before the writer's goroutine is even scheduled, or vice versa) and the test would still pass — the `seen != 0 && seen != numNodes` assertion is correct in isolation, but it is not guaranteed to ever observe the interleaved window it claims to prove is safe, especially under `-race`'s different (typically slower, more serialized) scheduling behavior. This weakens the INDX-05 concurrency proof's practical value: a regression that reintroduced a genuinely torn write could still pass this test by chance.
**Fix:** Add an explicit synchronization point that forces overlap — e.g. have the writer signal (via a channel) after staging all `PutNode` calls but *before* calling `Commit()`, have readers wait on that signal before calling `Snapshot()`, and only then let the writer proceed to `Commit()` (with a short delay or an additional handshake to increase the odds the reader's snapshot call and the commit truly race). This makes the "no torn write, even under a forced race" property an actual property of the test run rather than a matter of scheduling luck.

### WR-04: CGoParser.Close() is idempotent only for sequential calls, not concurrent ones

**File:** `internal/parser/cgo/parser_cgo.go:69-76`
**Issue:**
```go
func (p *CGoParser) Close() error {
    if p.closed {
        return nil
    }
    p.inner.Close()
    p.closed = true
    return nil
}
```
`p.closed` is a plain `bool` with no synchronization. Two goroutines calling `Close()` concurrently on the same `*CGoParser` can both observe `p.closed == false` before either sets it, and both call `p.inner.Close()` — a double-free of the underlying C parser (`ts_parser_delete` called twice on the same pointer), which is undefined behavior in C and would also be flagged by Go's race detector as a data race on `p.closed`. The package doc's "Resource contract" only promises repeat calls are safe, without specifying single-goroutine-only — if that's the intended contract, it should be stated explicitly; if concurrent `Close()` is expected to be safe, this needs a `sync.Once` or atomic guard.
**Fix:**
```go
type CGoParser struct {
    inner  *tree_sitter.Parser
    closeOnce sync.Once
}
func (p *CGoParser) Close() error {
    p.closeOnce.Do(func() { p.inner.Close() })
    return nil
}
```

## Info

### IN-01: `IterateEdges(srcPrefix string)` naming implies wildcard prefix matching but is actually an exact-source match

**File:** `internal/graphstore/store.go:51-54`, `internal/graphstore/keys.go:82-93`
**Issue:** The parameter name `srcPrefix` and the doc ("every edge whose source is srcPrefix") read naturally as "every edge whose source *starts with* this prefix," but the length-prefixed key encoding (deliberately, for injection safety) makes this an exact-source match only — `edgeSrcPrefix("a")` will never match edges whose real source is `"ab"`. This is correct and intentional, but the naming could mislead a future caller into assuming multi-source wildcard queries are supported.
**Fix:** Rename the parameter to `src` (matching `edgeSrcPrefix`'s own parameter name) or add an explicit doc line: "matches the exact source id only, never a textual prefix of it."

### IN-02: `File` and `Meta` messages have no reserved field range for future annotations, unlike `Node`/`Edge`

**File:** `internal/schema/graph.proto:66-91`
**Issue:** `Node` and `Edge` both reserve `50 to 59` for post-v1 annotation fields (embeddings, community assignments) per the ARCH-01 forward-compat discipline described in the file's own header comment. `File` and `Meta` have no equivalent reserved range, which is inconsistent with the stated "additive-only, pre-agreed slot" discipline — even if File/Meta are less likely to need embedding-style fields, a per-file aggregate annotation (e.g. a file-level health/community summary) is plausible and would have to compete for "whatever's next" instead of a pre-agreed slot.
**Fix:** Add a `reserved` range to `File` and `Meta` for consistency, or note explicitly in the proto file why they were deliberately excluded.

### IN-03: `strip_sql_timestamps` in capture.sh strips any coincidental 13-digit number, not just the intended timestamp columns

**File:** `testdata/golden/capture.sh:74`
**Issue:** `sed -E 's/[0-9]{13}(\.[0-9]+)?/<EPOCH_MS>/g'` runs over the entire SQL dump text and will rewrite *any* 13-digit numeric substring it finds — including a coincidental ID, hash fragment, or line-number value that happens to be exactly 13 digits — not just the `updated_at`/`indexed_at`/`applied_at` columns the comment says it targets. Given the current fixed schema (nodes/edges/files/schema_versions with a small, known column set) this is low risk in practice, but it is a blunt instrument that would silently corrupt an unrelated 13-digit value in a future column without any signal.
**Fix:** Scope the substitution to the specific known timestamp columns (e.g. via `jq`/awk column-aware replacement, or a `sed` pattern anchored to the known `INSERT INTO ... VALUES(...)` column positions) rather than a blanket numeric-width match across the whole dump.

---

_Reviewed: 2026-07-10_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
