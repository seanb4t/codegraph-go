# Phase 4: Output Hygiene - Pattern Map

**Mapped:** 2026-07-16
**Files analyzed:** 5
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|---------------|
| `internal/graphstore/pebble_store.go` (modify `Open`, line 147) | service (storage seam) | request-response (single config-injection call) | itself (existing file, same function) | exact |
| `internal/graphstore/logger.go` (new: `quietLogger`, D-04 writer seam) | utility (adapter implementing 3rd-party interface) | event-driven (log-callback routing) | `internal/graphstore/pebble_store.go`'s existing `openLockRetrySleep` test-seam var pattern | role-match (seam convention), no direct file analog for the logger type itself |
| `internal/graphstore/logger_test.go` (new: D-08 wiring + behavioral tests) | test | request-response / event-driven | `internal/graphstore/archtest/import_graph_test.go`'s "sanity check the test can fail" discipline + existing `pebble_store.go` open-lock tests | role-match |
| `internal/graphstore/archtest/stdout_confinement_test.go` (new: D-06b guard) | test (archtest) | batch (static analysis) | `internal/graphstore/archtest/import_graph_test.go`, `internal/migrate/archtest/modernc_confinement_test.go` | exact (structural twin, extended with AST/types) |
| `test/integration/mcp_stdout_purity_test.go` (new: D-06a raw-stdio harness) | test (integration) | streaming (raw stdout byte stream) | `test/integration/worktree_notice_test.go` (`newServeClient`) for fixture/spawn conventions; but core mechanism deliberately diverges (see below) | role-match with documented divergence |
| `test/integration/*_test.go` (new/extended: D-09 CLI-noise-absence case) | test (integration) | request-response (subprocess stdout/stderr capture) | `test/integration/main_test.go`'s `runBinary` helper | exact |

## Pattern Assignments

### `internal/graphstore/pebble_store.go` (modify, storage seam)

**Analog:** itself — `Open` at lines 141-157 (this is a one-line-plus-import change, not a new-file pattern)

**Current state** (lines 141-157):
```go
func Open(dir string) (GraphStore, error) {
	var lastErr error
	for attempt := 0; attempt < openLockRetryAttempts; attempt++ {
		if attempt > 0 {
			openLockRetrySleep(openLockRetryBackoff)
		}
		db, err := pebble.Open(dir, &pebble.Options{})
		if err == nil {
			return &pebbleStore{db: db}, nil
		}
		lastErr = classifyOpenError(err)
		if !errors.Is(lastErr, ErrStoreLocked) {
			break
		}
	}
	return nil, lastErr
}
```

**Required change:** line 147 only —
```go
db, err := pebble.Open(dir, &pebble.Options{Logger: quietLogger{}})
```
Do NOT touch the retry loop, `classifyOpenError`, or `ErrStoreLocked` — these were hard-won across three Phase-3 review rounds (CONTEXT.md "Specifics"). `quietLogger` is defined in the new sibling file `logger.go`, same package, so no new import is needed here beyond what already exists.

**Test-only-seam convention already established in this file** (lines 83-90) — mirror this exact shape for the new D-04 writer seam:
```go
// openLockRetrySleep is Open's between-attempts sleep, hoisted behind an
// unexported package var (03-REVIEW.md IN-02) so open_lock_test.go can
// event-synchronize the holder's release on the retry loop's own attempt
// boundaries instead of racing a wall-clock guess under parallel CI load...
// Production behavior is unchanged: the var defaults to time.Sleep and has
// no exported setter.
var openLockRetrySleep = time.Sleep
```
The new `diagWriter` var in `logger.go` should follow this identically: unexported package-level var, defaults to the production value (`os.Stderr`), no exported setter, doc comment explaining the test-capture purpose.

---

### `internal/graphstore/logger.go` (new)

**No direct file analog** (first pebble.Logger implementation in this codebase) — pattern comes from:
1. The pinned `pebble/v2@v2.1.6` `base.Logger` interface (verified in RESEARCH.md, `internal/base/logger.go:18-23`):
```go
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}
```
2. This codebase's existing sentinel-error / doc-comment style from `pebble_store.go` (e.g. `ErrStoreLocked`, `ErrNotFound` — package-level `var`, exhaustive doc comment explaining provenance and rationale, `errors.New("graphstore: ...")`-shaped messages).
3. The `openLockRetrySleep` test-seam convention (above) for `diagWriter`.

**Recommended shape** (from RESEARCH.md Pattern 1, cross-checked against this repo's conventions):
```go
package graphstore

import (
	"fmt"
	"io"
	"os"

	"github.com/cockroachdb/pebble/v2"
)

// quietLogger implements pebble.Logger (D-01): Infof is the WAL/compaction/
// memtable chatter pebble emits directly, discarded unconditionally (D-02).
// Errorf is preserved real diagnostic signal, written via the diagWriter
// seam with a provenance prefix. Fatalf preserves pebble's own semantics.
type quietLogger struct{}

var _ pebble.Logger = quietLogger{}

func (quietLogger) Infof(format string, args ...any) {}

func (quietLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(diagWriter, "codegraph: pebble: "+format+"\n", args...)
}

func (quietLogger) Fatalf(format string, args ...any) {
	fmt.Fprintf(diagWriter, "codegraph: pebble: fatal: "+format+"\n", args...)
	os.Exit(1)
}

// diagWriter is the D-04 test-only-seam convention (mirrors
// openLockRetrySleep in pebble_store.go): production always writes to
// os.Stderr; tests reassign this var to capture output.
var diagWriter io.Writer = os.Stderr
```

**Import confinement note:** `logger.go` lives in `internal/graphstore` — the sole pebble-aware package (D-04a, enforced by `archtest.TestNoPackageBypassesGraphStore`). Its import of `github.com/cockroachdb/pebble/v2` is already legal per that existing archtest; no changes needed there.

---

### `internal/graphstore/logger_test.go` (new)

**Analog for the "assert wiring, not a replica" discipline:** `internal/graphstore/archtest/import_graph_test.go`'s sanity-check pattern (lines 59-66):
```go
// Sanity check that this test can actually detect a real importer: if
// internal/graphstore itself no longer imports pebble/v2 (e.g. after a
// refactor), the check above is vacuously true for the wrong reason...
if !foundGraphstoreImporter {
	t.Fatal("no package under internal/graphstore was found importing pebble/v2 — ...")
}
```
Apply the same "mutation-proof" discipline (D-08): a wiring test that asserts `Open`'s actual `pebble.Options.Logger` field is a `quietLogger` (reflect or behavioral, Claude's discretion) such that reverting line 147 to `&pebble.Options{}` turns the test red — not merely re-asserting a private constant.

**Behavioral test hook** (from RESEARCH.md, verified against pinned pebble source): `pebble.Open`'s own `"Found %d WALs"` Infof call fires on every `Open`, even a brand-new empty store — cheapest deterministic noise trigger. Capture `diagWriter` (swap package var, restore via `t.Cleanup`) during a real `graphstore.Open`+`Close` cycle and assert zero bytes reached it, then directly invoke `Errorf` and assert the provenance-prefixed line does reach it.

**Fatalf test caution** (RESEARCH.md Pitfall 3): do NOT invoke `quietLogger.Fatalf` directly in a test — it calls `os.Exit(1)` and will kill the test binary. Extract message-formatting into a small internal helper if format-behavior needs asserting, and treat the `os.Exit(1)` line as reviewed-but-untested, mirroring pebble's own `InMemLogger.Fatalf` test-double shape (no exit call).

---

### `internal/graphstore/archtest/stdout_confinement_test.go` (new, D-06b)

**Analog:** `internal/graphstore/archtest/import_graph_test.go` and `internal/migrate/archtest/modernc_confinement_test.go` — both use identical structure: package doc comment stating the boundary being enforced, `packages.Load` with a `packages.Config`, iterate `pkgs`, `t.Errorf` per violation, and a closing "sanity check this test can actually fail" `t.Fatal`.

**Package doc-comment pattern to mirror** (from `import_graph_test.go` lines 1-8):
```go
// Package archtest enforces D-04a: no package outside internal/graphstore
// (and its own subpackages) may import the embedded key-value engine
// directly. Go's internal/ convention alone does NOT stop a sibling
// package... — this test, not the directory convention, is what actually
// enforces the boundary (RESEARCH Pattern 5).
package archtest
```
Adapt to state the HYG-02 stdout-confinement boundary and its six guarded packages instead.

**`packages.Config` divergence (deliberate, per RESEARCH.md):** the two existing precedents use:
```go
cfg := &packages.Config{
	Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps,
	Tests: true,
}
```
The new guard must ADD `NeedSyntax | NeedTypes | NeedTypesInfo | NeedFiles | NeedCompiledGoFiles` to walk AST, and RESEARCH.md's own reasoning recommends `Tests: false` (a documented divergence — a `_test.go` file printing to stdout during `go test` never touches the real `serve --mcp` binary). Document this divergence inline in the new file's doc comment, mirroring the existing files' habit of explaining every non-obvious choice.

**Full worked example** (from RESEARCH.md Pattern 2 — copy this shape, filling in `isOSStdoutRef`/`isBareFmtPrint`/`isLogSetOutput` helpers that resolve via `pkg.TypesInfo.Uses` rather than string-matching identifier names):
```go
var guardedPackages = []string{
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/graphstore",
	"github.com/seanb4t/codegraph-go/internal/daemon",
	"github.com/seanb4t/codegraph-go/internal/watch",
	"github.com/seanb4t/codegraph-go/internal/indexer",
	"github.com/seanb4t/codegraph-go/internal/query",
	// internal/cli is DELIBERATELY excluded — it legitimately renders
	// product output via cmd.OutOrStdout() (03-PATTERNS.md).
}

func TestNoStdoutNoiseInServeReachablePackages(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, guardedPackages...)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				switch expr := n.(type) {
				case *ast.SelectorExpr:
					if isOSStdoutRef(expr, pkg.TypesInfo) {
						t.Errorf("%s: references os.Stdout — diagnostics must use the injected stderr seam, never stdout directly", pkg.PkgPath)
					}
				case *ast.CallExpr:
					if isBareFmtPrint(expr, pkg.TypesInfo) {
						t.Errorf("%s: calls a bare fmt.Print*(no writer) — stdout is reserved for the MCP JSON-RPC transport", pkg.PkgPath)
					}
					if isLogSetOutput(expr, pkg.TypesInfo) {
						t.Errorf("%s: calls log.SetOutput — must never redirect stdlib log's default stderr output", pkg.PkgPath)
					}
				}
				return true
			})
		}
	}
}
```

**Sanity-check requirement (Pitfall 4, mandatory):** mirror the existing precedents' closing check — assert e.g. `len(pkgs) == len(guardedPackages)` or that each named package's `Syntax` is non-empty, so a future rename of one of the six guarded packages can't silently stop being checked.

**`isAllowedImporter`/`stripTestVariant` helper style** to reuse verbatim if any test-variant path normalization is needed (both existing files share this helper unmodified):
```go
func stripTestVariant(pkgPath string) string {
	if i := strings.IndexByte(pkgPath, ' '); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	pkgPath = strings.TrimSuffix(pkgPath, ".test")
	return pkgPath
}
```

---

### `test/integration/mcp_stdout_purity_test.go` (new, D-06a)

**Analog for fixture/spawn conventions:** `test/integration/worktree_notice_test.go`'s `newServeClient` (lines 15-45) and `test/integration/main_test.go`'s `copyFixture`/`runBinary` helpers — reuse `binPath` (package var from `TestMain`), `copyFixture(t)` for the fixture dir, and the `t.Cleanup`-registered process-kill pattern.

**Critical divergence (do NOT reuse `newServeClient`/`mcpclient.Client` for the actual assertion):** `mcp-go@v0.56.0`'s stdio transport silently skips any stdout line that fails `json.Unmarshal` (`client/transport/stdio.go:readResponses`) — a test built on `Client.Initialize`/`Client.CallTool` succeeding can never detect stray non-frame bytes on stdout. Build a narrow, purpose-built raw reader instead (RESEARCH.md Pattern 3):
```go
func TestServeMCPStdoutIsPureJSONRPC(t *testing.T) {
	dir := copyFixture(t)
	// ... init the fixture via runBinary(t, dir, nil, "init", dir) ...

	cmd := exec.Command(binPath, "serve", "--mcp")
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = &bytes.Buffer{} // diagnostics go here, never asserted for purity
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	scanner := bufio.NewScanner(stdout)
	// write a raw initialize request line, then a tools/call request line
	// (e.g. codegraph_status, which exercises indexer.Sync's startup
	// reconcile store-open path — the deliberate noise-provoking case)...
	for scanner.Scan() {
		line := scanner.Bytes()
		var frame struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(line, &frame); err != nil || frame.JSONRPC == "" {
			t.Fatalf("non-JSON-RPC byte on stdout: %q", line)
		}
		// ... stop once the expected number of responses is seen ...
	}
}
```
Every other test in `test/integration/` should keep using `newServeClient`/`mcpclient.Client` unmodified — this is an additive sibling helper, not a replacement.

---

### `test/integration/*_test.go` (new/extended, D-09 CLI-noise-absence case)

**Analog:** `test/integration/main_test.go`'s `runBinary` (lines 165-183) — already returns separate `stdout, stderr string` from a subprocess run; reuse directly.

**Pattern:**
```go
func TestSyncStderrNoPebbleNoise(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init: %v: %s", err, stderr)
	}
	_, stderr, err := runBinary(t, dir, nil, "sync")
	if err != nil {
		t.Fatalf("sync: %v: %s", err, stderr)
	}
	for _, noise := range []string{"[JOB ", "WAL ", "compaction", "pickAuto"} {
		if strings.Contains(stderr, noise) {
			t.Errorf("sync stderr contains pebble-shaped noise %q:\n%s", noise, stderr)
		}
	}
}
```
Per RESEARCH.md's Open Question #1 recommendation: drive `sync` (not `status`) against the existing `copyFixture(t)` fixture — exercises strictly more of pebble's Infof surface (flush + possible compaction) than a bare read-only `status` call. This is an absence-of-substring check on noise SHAPES only, not full-output emptiness (legitimate codegraph warnings may still appear in stderr).

## Shared Patterns

### Package-level test-only injectable seam (used by both D-04 and prior phases)
**Source:** `internal/graphstore/pebble_store.go:83-90` (`openLockRetrySleep`)
**Apply to:** `internal/graphstore/logger.go`'s `diagWriter` var — unexported package-level var, production default (`time.Sleep` / `os.Stderr`), no exported setter, doc comment explaining the test-capture purpose and that production behavior is unchanged.

### Archtest "assert wiring, not a replica" + mutation-proof sanity check
**Source:** `internal/graphstore/archtest/import_graph_test.go:59-66`, `internal/migrate/archtest/modernc_confinement_test.go:61-68`
**Apply to:** Both `logger_test.go`'s D-08 wiring test (must go red if line 147's `Logger:` field is reverted) and the new `stdout_confinement_test.go` (must have a "did this test actually check anything" sanity assertion, per Pitfall 4).

### Diagnostics-to-stderr-never-stdout (already-stated repo-wide rule)
**Source:** `internal/mcp/server.go:63-66` (`WarnUnknownToolsTo` doc comment) — "Diagnostics never go to stdout — stdout is reserved for the MCP JSON-RPC transport (T-03-07-Leak) — so callers must pass os.Stderr, never os.Stdout, in production."
**Apply to:** `quietLogger.Errorf`/`Fatalf` (write to `diagWriter`, defaulting to stderr) and the entire rationale for the D-06(b) archtest guard.

### `go/packages`-based structural confinement (NOT regex)
**Source:** both existing archtest files' shared doc-comment rationale ("NOT regex/string-matching over source — regex misses aliased imports, build-tag-gated files, and test variants")
**Apply to:** `stdout_confinement_test.go` — extend with `NeedSyntax`/`NeedTypes`/`NeedTypesInfo` rather than falling back to a text scan.

## No Analog Found

None — every file in this phase has at least a role-match analog; the two genuinely novel pieces (`quietLogger` itself, and the raw-stdio purity reader) are covered by RESEARCH.md's verified, pinned-source-derived Code Examples (Pattern 1 and Pattern 3) rather than an in-repo analog, since this is the first pebble.Logger implementation and the first raw-stdio test harness in the codebase.

## Metadata

**Analog search scope:** `internal/graphstore/`, `internal/graphstore/archtest/`, `internal/migrate/archtest/`, `internal/mcp/server.go`, `test/integration/`
**Files scanned:** `pebble_store.go`, `archtest/import_graph_test.go`, `internal/migrate/archtest/modernc_confinement_test.go`, `test/integration/main_test.go`, `test/integration/worktree_notice_test.go`, `internal/mcp/server.go`
**Pattern extraction date:** 2026-07-16
