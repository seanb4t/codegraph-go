---
phase: 02-go-indexing-pipeline
reviewed: 2026-07-11T02:52:42Z
depth: deep
files_reviewed: 24
files_reviewed_list:
  - cmd/codegraph/main.go
  - internal/cli/root.go
  - internal/cli/init.go
  - internal/cli/index.go
  - internal/cli/uninit.go
  - internal/cli/cli_test.go
  - internal/indexer/discover.go
  - internal/indexer/doc.go
  - internal/indexer/discover_test.go
  - internal/indexer/extract.go
  - internal/indexer/extract_test.go
  - internal/indexer/goextract/goextract.go
  - internal/indexer/goextract/types.go
  - internal/indexer/goextract/doc.go
  - internal/indexer/goextract/goextract_test.go
  - internal/indexer/nodeid/nodeid.go
  - internal/indexer/nodeid/doc.go
  - internal/indexer/nodeid/nodeid_test.go
  - internal/indexer/resolve.go
  - internal/indexer/symbolindex.go
  - internal/indexer/resolve_test.go
  - internal/indexer/pipeline.go
  - internal/indexer/pipeline_test.go
  - internal/indexer/determinism_test.go
findings:
  critical: 0
  warning: 2
  info: 5
  total: 7
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-07-11T02:52:42Z
**Depth:** deep
**Files Reviewed:** 24
**Status:** issues_found

## Summary

Deep cross-file review of the Phase-2 Go indexing pipeline (discover → extract → resolve → write → CLI). I traced the extractor → resolver → pipeline → CLI call chains and audited the four focus areas explicitly.

The load-bearing correctness properties the phase targets are genuinely well-built and I could not break them:

- **Determinism (INDX-02):** Pass 1 writes results into a pre-allocated slice indexed by file position (`extract.go:70,90,124`), never in completion order; `Discover` sorts by `RelPath` (`discover.go:101`); `writeGraph` sorts nodes by id, files by path, and edges by (src,kind,target) before staging (`resolve.go:251-257`); `collapseEdges` picks its representative by a total-order sort, not processing order (`resolve.go:207-231`). `symbolIndex` never ranges a map to decide output. Meta stamps no timestamp (`NewMeta` sets only `SchemaVersion`). No residual map/goroutine nondeterminism found.
- **Concurrency (D-04):** exactly `min(limit,len(files))` persistent workers, each constructing one `parser.Parser` and pulling indices from a single `atomic.Int64` (`extract.go:78-94`). No shared Parser/Tree; disjoint slice writes; `defer p.Close()` registered only after successful construction.
- **Resource safety:** `store.Close` deferred on every `run` return path; `writeGraph` calls `w.Close()` on every staging-error path and `Commit()` (which closes the batch) on success; `tree.Close()` deferred in `goextract.Extract`; `CGoParser.Parse` enforces `MaxSourceBytes` before touching the C parser (`parser_cgo.go:59-61`).
- **Security (ASVS L1):** node ids and content hashes use SHA-256 (`nodeid.go:49`, `goextract.go:31`); keyspace segments and id preimages are varint-length-prefixed, blocking key/id injection; `uninit`/`index` scope `RemoveAll` to the resolved `.codegraph/` subtree; the AST walk is iterative (`walkDescendants`) so a deep AST cannot exhaust the Go stack. No synthesized interface→impl dispatch edges leak in — only ground-truth calls/imports/embeds/contains are emitted (correct Phase-5 scope discipline).

The defects below are in **reference-resolution accuracy** — the resolver produces wrong or false-positive edges under name collisions and expression-receiver method calls. These are deterministic (so the determinism gate stays green) and untested, which is exactly the class of bug an adversarial review must surface.

## Warnings

### WR-01: Methods are indexed under their bare name and can shadow a same-package function, producing wrong `calls` edges

**File:** `internal/indexer/symbolindex.go:37-46`
**Issue:** `newSymbolIndex` inserts every non-file node under its bare `Name` into `byImportAndName[importPath][name]`, including `KindMethod` nodes (a method's `Name` is the bare method name, not `Recv.Method`). A same-package unqualified call can only ever bind to a package-level func/type/const/var — never a method — yet methods pollute that same namespace. When a package legitimately contains both `func Foo()` and `func (t T) Foo()` (common, e.g. a `Close()`/`String()` helper alongside a method of the same name), the later-inserted entry wins and overwrites the earlier:

```
names[en.Node.Name] = en.Node.Id   // method "Foo" clobbers func "Foo"
```

An unqualified call `Foo()` in that package (recorded with `PkgAlias:""`, resolved via `resolveUnqualified`, `resolve.go:143`) then resolves to the **method** node id instead of the function — a wrong `calls` edge in the committed graph. It is deterministic (insertion follows sorted file order), so `TestDeterministicRebuild` cannot catch it, and no test exercises a func/method name collision.
**Fix:** Exclude `KindMethod` from the unqualified symbol index (methods are unreachable by bare-name reference), or key methods separately so they never occupy the unqualified namespace:

```go
for _, en := range r.Nodes {
    if en.Node.Kind == goextract.KindFile || en.Node.Kind == goextract.KindMethod {
        continue // methods are never reachable via a same-package bare-name call
    }
    names[en.Node.Name] = en.Node.Id
}
```

### WR-02: Selector calls on a non-identifier operand collapse to `PkgAlias:""` and are resolved as same-package calls, creating false-positive `calls` edges

**File:** `internal/indexer/goextract/goextract.go:597-616` (interacts with `resolve.go:141-146`, `symbolindex.go:76-83`)
**Issue:** In `recordCall`, `pkgAlias` is only set when the selector's operand is a bare `identifier`:

```go
var pkgAlias string
if operand != nil && operand.Kind() == "identifier" {
    pkgAlias = operand.Utf8Text(ex.src)
}
```

For any selector whose operand is *not* a plain identifier — `foo().Bar()`, `arr[0].Bar()`, `x.y.Bar()`, `(&T{}).Bar()` — `pkgAlias` stays `""`. The emitted `UnresolvedRef` is then indistinguishable from a genuine bare call `Bar()`, so Pass 2 routes it through `resolveUnqualified(callerImportPath, "Bar")`, which has **no import-alias guard**. If the caller's own package declares any symbol named `Bar` (function, type, const, var — or, per WR-01, a method), the method call on an arbitrary expression falsely resolves to it, producing a wrong `calls` edge. The design's safety argument (`symbolindex.go:51-58`) only holds for identifier operands (the tested `w.Describe()` case at `resolve_test.go:184`); the expression-operand path defeats it. Untested.
**Fix:** Preserve the "this was a selector, not a bare identifier" distinction so Pass 2 never treats an expression-receiver method call as an unqualified same-package reference. E.g. when the operand is present but not an identifier, either skip recording the ref or tag it with a sentinel `PkgAlias` that `resolveNameRef` refuses to resolve unqualified:

```go
if operand != nil && operand.Kind() != "identifier" {
    return // receiver is an expression, not a package/identifier — not resolvable in the narrowest-safe set
}
```

## Info

### IN-01: Multi-name const/var specs stamp identical positions on every declared name

**File:** `internal/indexer/goextract/goextract.go:301-331`
**Issue:** `emitConstVarSpec` computes `pos`/`end` once from the whole spec and applies them to every name in a grouped declaration, so `const a, b = 1, 2` yields nodes for `a` and `b` with identical `StartCol`/`EndCol`. Node ids stay distinct (id keys on name), so this is positional-metadata inaccuracy, not an identity collision.
**Fix:** Use each identifier child's own `StartPosition()`/`EndPosition()` for its node.

### IN-02: Embedded struct field carrying a struct tag is missed

**File:** `internal/indexer/goextract/goextract.go:203-206`
**Issue:** The embedded-field test requires `fd.NamedChildCount() != 1`. A valid embedded field with a tag (e.g. ``Base `json:"base"` ``) has two named children (the type + the tag literal), so it is skipped and no `embeds` ref is emitted — a false negative for tagged embeds.
**Fix:** Detect the embed by the absence of a `field` name child (single type child + optional `tag` field) rather than an exact child count of 1.

### IN-03: `collapseEdges` representative tiebreak includes a `filePath` term that can never break a tie

**File:** `internal/indexer/resolve.go:220-230`
**Issue:** All candidates in a group share the same `Source` (it is part of the collapse triple), so `nodeFilePath[ci.Source] == nodeFilePath[cj.Source]` for every pair; the `fi != fj` branch is dead and the effective order is `(line, col)` only. The doc comment (`resolve.go:190-195`) claims the total order is `(filePath, line, col)`, which is misleading. Behavior is still deterministic and correct.
**Fix:** Drop the redundant `filePath` comparison (and the `nodeFilePath` argument if unused elsewhere) or correct the comment to state the order is `(line, col)`.

### IN-04: `readGraphCounts` failure drops the partial Stats other error paths preserve

**File:** `internal/indexer/pipeline.go:100-103`
**Issue:** Every other error path in `run` returns a partially-populated `Stats` (Files/Duration/Unresolved), but the `readGraphCounts` failure returns a bare `Stats{}`, discarding Files/Duration/Unresolved that were already known. Minor observability inconsistency.
**Fix:** Return `Stats{Files: len(files), Unresolved: unresolved, Duration: time.Since(start)}, err`.

### IN-05: Dot-import (`import . "pkg"`) calls are not resolved to the dot-imported package

**File:** `internal/indexer/goextract/goextract.go:593-596`, `internal/indexer/resolve.go:141-146`
**Issue:** A call to a dot-imported symbol appears as a bare `identifier` with no selector, so it is recorded with `PkgAlias:""` and resolved only against the caller's own package via `resolveUnqualified` — it can never bind to the dot-imported package's symbol (and could false-match a same-package symbol of the same name). This is arguably within the RQ-2 narrowest-safe-set boundary, but it is undocumented at the call site and untested. Dot imports are rare in production Go.
**Fix:** Either document that dot-imported references are intentionally left unresolved, or leave them unresolved explicitly rather than routing them through same-package lookup.

---

_Reviewed: 2026-07-11T02:52:42Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
