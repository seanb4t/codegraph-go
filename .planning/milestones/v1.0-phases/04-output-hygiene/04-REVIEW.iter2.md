---
phase: 04-output-hygiene
reviewed: 2026-07-16T22:04:17Z
depth: deep
files_reviewed: 7
files_reviewed_list:
  - internal/graphstore/logger.go
  - internal/graphstore/logger_test.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/archtest/stdout_confinement_test.go
  - internal/graphstore/archtest/stdout_detection_selftest_test.go
  - test/integration/mcp_stdout_purity_test.go
  - test/integration/sync_noise_test.go
findings:
  critical: 1
  warning: 3
  info: 1
  total: 5
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-07-16T22:04:17Z
**Depth:** deep
**Files Reviewed:** 7
**Status:** issues_found

## Summary

HYG-01 (Pebble Logger injection) is implemented correctly and narrowly: the
diff against `e9b8986^` shows the *only* production change to
`pebble_store.go` is `&pebble.Options{}` → `&pebble.Options{Logger:
quietLogger{}}` at the single `pebble.Open` call site inside `Open`'s
retry loop. I traced pebble/v2's actual `Open` source
(`open.go:81-133`) and confirmed the CR-01 lock-held path
(`LockDirectory` → `open.go:129`) returns its error directly, before any
`Logger.Errorf`/`Infof` call — so `classifyOpenError`'s `isLockHeldOS`
matching is untouched by the Logger swap. I also empirically reverted the
injection in a throwaway worktree and confirmed `TestOpenInjectsQuietLogger`
and `TestSyncStderrNoPebbleNoise` both genuinely catch the regression (real
"Found N WALs" / "[JOB ...] WAL ..." noise reappears on stderr) — these two
tests are correctly mutation-proof, not a replica of the wiring.

HYG-02's structural guard (`stdout_confinement_test.go`), however, has a
confirmed, exploitable blind spot: `packages.Load` is called without
`NeedDeps` against exactly the six `guardedPackages`, so only those
packages' *own* source files ever get `pkg.Syntax`/`pkg.TypesInfo`
populated — none of the ~20 packages they transitively import (all of
`internal/indexer/*extract`, `internal/schema`, `internal/gitmeta`,
`internal/parser`, etc.) are inspected at all, despite being genuinely
reachable during a live `serve --mcp` session. I proved this is not
theoretical: adding a real `fmt.Println` call to `internal/schema` in a
throwaway worktree left `TestNoStdoutNoiseInServeReachablePackages`
green. This is the review's headline finding — see CR-01 below.

The two integration tests are directionally sound (`sync_noise_test.go`'s
noise-shape assertion is genuinely mutation-proof, verified empirically),
but `mcp_stdout_purity_test.go` has a real data race in its failure path
and a coverage gap where a broken tool-allowlist could silently narrow what
the test actually exercises while still reporting green.

## Critical Issues

### CR-01: HYG-02 stdout guard does not scan any transitively-imported package, despite claiming to cover "every package that can execute during a serve --mcp session"

**File:** `internal/graphstore/archtest/stdout_confinement_test.go:37-91`
**Issue:**

`guardedPackages` lists exactly six package paths, and
`TestNoStdoutNoiseInServeReachablePackages` loads them with:

```go
cfg := &packages.Config{
    Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
        packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
}
pkgs, err := packages.Load(cfg, guardedPackages...)
...
for _, pkg := range pkgs {          // only the 6 root packages
    for _, f := range pkg.Syntax {  // ast.Inspect only these files
```

`golang.org/x/tools/go/packages` only populates `Syntax`/`Types`/
`TypesInfo` for the packages matching the load *patterns* ("root"
packages) unless `packages.NeedDeps` is also set
(`go/packages/packages.go:805-808`, v0.48.0: `needsrc := (...&&
(rootIndex >= 0 || ld.Mode&NeedDeps != 0))`). `NeedDeps` is not set here,
and even if it were, the `ast.Inspect` loop only ever ranges over the
top-level `pkgs` slice (the 6 roots) — it never descends into
`pkg.Imports`. I verified this empirically with a standalone probe:
loading `internal/graphstore` this way, its own dependency
`internal/schema` comes back with `len(imp.Syntax) == 0` and
`imp.TypesInfo == nil`.

The six guarded packages transitively import (via `go list -f
'{{.Imports}}'`) at least: `internal/schema`, `internal/gitmeta`,
`internal/parser`, `internal/parser/cgo`, `internal/indexer/dispatch`,
`internal/indexer/nodeid`, `internal/indexer/routes`, and all 12
`internal/indexer/*extract` / `internal/indexer/mainstream/*extract`
language-extractor packages — every one of which runs during a live
`serve --mcp` session's startup reconcile or a debounced `indexer.Sync`.
None of these are inspected by this test.

**Proof (reproduced in a throwaway git worktree, reverted before
finishing review):**

```go
// added to internal/schema/stdout_violation_probe.go
package schema
import "fmt"
func printProbeViolation() {
    fmt.Println("this should be flagged by HYG-02 but is NOT in guardedPackages")
}
```

```
$ go test ./internal/graphstore/archtest/... -run TestNoStdoutNoiseInServeReachablePackages -v
=== RUN   TestNoStdoutNoiseInServeReachablePackages
--- PASS: TestNoStdoutNoiseInServeReachablePackages (0.24s)
PASS
```

A real, unambiguous stdout write in a package that `internal/graphstore`,
`internal/indexer`, and `internal/query` all import and call directly is
completely invisible to this guard. The doc comment's claim — "every
package that can execute during a `serve --mcp` session ... must never
write to stdout" — is false as implemented; only the 6 named packages'
own files are covered, not their dependency graph. This is the same
"vacuously green" failure mode the test's own Pitfall-4 comment warns
about (checking `len(pkgs)` and per-package `len(pkg.Syntax)==0`), but the
authors checked for the wrong failure mode — they hardened against
`packages.Load` silently *not resolving* a requested root package, but
missed that resolved dependencies are never checked at all.

**Fix:** Either (a) pass `packages.NeedDeps` and walk `pkg.Imports`
recursively (filtered to `seanb4t/codegraph-go/...` paths, to avoid
flagging third-party/stdlib deps), or (b) replace `guardedPackages` with a
single `"github.com/seanb4t/codegraph-go/..."` load plus an explicit
serve-reachability filter (mirroring `import_graph_test.go`'s existing
`Tests: false`, whole-module pattern), scoped by excluding `internal/cli`
and any other package proven unreachable from `serve --mcp`. Whichever
approach is chosen, add a regression test analogous to
`stdout_detection_selftest_test.go` that plants a violation in a
*dependency* of a guarded package (not the guarded package itself) and
asserts the suite catches it — closing exactly the gap this finding
demonstrates.

## Warnings

### WR-01: Data race on `stderrBuf` in the MCP stdout-purity test's failure path

**File:** `test/integration/mcp_stdout_purity_test.go:63-64, 137-138, 157`
**Issue:** `cmd.Stderr = &stderrBuf` (a plain `bytes.Buffer`, not
synchronized). When `cmd.Stderr` is set to a non-`*os.File` `io.Writer`,
`os/exec` starts a background goroutine that copies the subprocess's
stderr pipe into it (`os/exec/exec.go`, `c.goroutine = append(...)`,
joined only inside `cmd.Wait()`/`awaitGoroutines`). Both failure branches
read `stderrBuf.String()` — `t.Fatalf(..., stderrBuf.String())` at line
137 and the deadline branch at line 157 — while that copying goroutine is
still running (`cmd.Wait()` has not been called; it only runs later, from
the `t.Cleanup` registered at line 69-72, after `t.Fatalf` has already
triggered `runtime.Goexit`). This is a genuine unsynchronized concurrent
read/write on a `bytes.Buffer` and will be flagged by `go test -race` if
either failure branch is ever exercised while the subprocess is still
actively writing to stderr.
**Fix:** Guard `stderrBuf` with a `sync.Mutex` (or use a
concurrency-safe buffer, e.g. wrap writes/reads under a mutex, or read via
`cmd.Process.Kill(); cmd.Wait()` *before* formatting the failure message
so the copy goroutine has already joined):
```go
t.Cleanup(func() {
    _ = cmd.Process.Kill()
    _ = cmd.Wait()
})
...
case <-deadline:
    _ = cmd.Process.Kill()
    _ = cmd.Wait() // join the stderr-copy goroutine before reading stderrBuf
    t.Fatalf("timed out ...: %s", stderrBuf.String())
```

### WR-02: `TestServeMCPStdoutIsPureJSONRPC` never checks whether the `tools/call` actually succeeded

**File:** `test/integration/mcp_stdout_purity_test.go:107-159`
**Issue:** The decoded `frame` struct only carries `jsonrpc` and `id`:
```go
var frame struct {
    JSONRPC string  `json:"jsonrpc"`
    ID      float64 `json:"id"`
}
```
`sawToolResponse` is set as soon as any frame with `id == 2` arrives,
whether it is a successful `result` or a JSON-RPC `error` object (e.g.
"unknown tool", or a parameter-validation failure). The test's own doc
comment states the point of calling `codegraph_status` is to
"deliberately exercise a SECOND store-open ... on top of the startup
reconcile" — but if `CODEGRAPH_MCP_TOOLS` allowlisting regresses (env var
typo, allowlist logic bug), the server would return an early
tool-not-found error *before* ever reaching `graphstore.Open`, and this
test would still report success, silently no longer exercising the
noise-provoking path it exists to cover — while its comments continue to
claim it does.
**Fix:** Decode the full frame (`result`/`error` presence) and assert the
tool response has no `error` field:
```go
var frame struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      float64         `json:"id"`
    Error   json.RawMessage `json:"error"`
}
...
if frame.ID == 2 && len(frame.Error) > 0 {
    t.Fatalf("codegraph_status tools/call returned a JSON-RPC error, the store-open path was not exercised: %s", frame.Error)
}
```

### WR-03: `diagWriter` is a shared, unsynchronized global mutated by tests

**File:** `internal/graphstore/logger.go:65`, `internal/graphstore/logger_test.go:17-24`
**Issue:** `diagWriter` is a bare package-level `io.Writer` var, and
`captureDiagWriter` reassigns it directly with no mutex/atomic. Today this
is safe only because no test in package `graphstore` calls `t.Parallel()`
and no background goroutine calls `quietLogger.Errorf`/`Fatalf`
concurrently with a test's capture window. It is a latent footgun:
adding `t.Parallel()` to any test in this package, or a future scenario
where a long-lived shared `pebbleStore` (explicitly anticipated in this
same file's own `mu` doc comment) triggers a background `Errorf` while
another goroutine's test has redirected `diagWriter` to its own buffer,
produces a data race on both the `io.Writer` variable itself and the
`*bytes.Buffer` it points to.
**Fix:** Either document the "never add `t.Parallel()` in this package"
invariant explicitly next to `diagWriter`'s declaration, or guard reads/
writes of `diagWriter` behind an `atomic.Value`/mutex so the seam is
race-safe if that assumption is ever violated.

## Info

### IN-01: Stdout-confinement predicates cannot see indirect stdout writes

**File:** `internal/graphstore/archtest/stdout_confinement_test.go:123-200`
**Issue:** `isOSStdoutRef`/`isBareFmtPrint`/`isLogSetOutput` only match
direct, statically-resolvable identifier references. A write via
`os.NewFile(1, "").Write(...)`, `syscall.Write(1, ...)`, or a value
captured from `os.Stdout` earlier and threaded through an unrelated
variable/interface would not be detected by any of the three predicates
(and is not claimed to be, but is worth calling out given the test's
confident "Expected GREEN today — D-07's zero-violation baseline"
framing). This is an inherent limitation of AST/identifier-based static
analysis, not a fixable defect, but should be listed as a residual risk
anywhere this guard's guarantee is documented for future readers.
**Fix:** No code change required; consider a one-line addendum to the
package doc comment noting this class of bypass is out of scope for the
static guard and would only be caught by `mcp_stdout_purity_test.go`'s
runtime check (once CR-01's coverage gap for indirectly-reachable
packages is also closed).

---

_Reviewed: 2026-07-16T22:04:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
