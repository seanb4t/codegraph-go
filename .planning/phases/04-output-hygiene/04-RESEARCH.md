# Phase 4: Output Hygiene - Research

**Researched:** 2026-07-16
**Domain:** Embedded-store (Pebble/v2) logger injection + MCP stdio transport-purity guarantees, in Go
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Pebble logger routing (HYG-01)**
- **D-01:** A small unexported logger type in `internal/graphstore` — the sole pebble-aware package (v0.1 D-04a; no other package may see pebble types) — implements pebble v2's three-method `base.Logger` interface (`Infof` / `Errorf` / `Fatalf`, verified against the pinned `pebble/v2@v2.1.6` sources). It is injected via `pebble.Options.Logger` at the module's SINGLE `pebble.Open` seam (`internal/graphstore/pebble_store.go:147`, today `&pebble.Options{}`). The requirement text names `pebble.Options.Logger` — that is the locked mechanism; no global stdlib-log hijacking.
- **D-02:** Level routing: `Infof` → discard unconditionally (this is the WAL/compaction/memtable chatter); `Errorf` → preserved, written to stderr; `Fatalf` → preserve Pebble's default semantics (message to stderr, then exit — `DefaultLogger.Fatalf` is stdlib `log.Fatalf`-equivalent). Do NOT soften Fatalf: pebble only Fatalfs on invariant violations where continuing is unsafe, and the phase's own guardrail is "real errors are never hidden".
- **D-03:** Set ONLY `Options.Logger` — no `LoggerAndTracer`, no custom `EventListener`. Pebble's `EnsureDefaults` derives the default `EventListener` from `o.Logger` (options.go ~1469), so the quiet logger silences the derived event noise too. One field, one seam, whole surface.
- **D-04:** The preserved error path writes through a package-level injectable `io.Writer` seam defaulting to `os.Stderr` (the established test-only-seam convention from Phase 3) so tests can capture output; production is always stderr per the repo-wide diagnostics rule (T-03-07-Leak, `internal/mcp/server.go:63-66`). Pebble-originated error lines carry a provenance prefix (exact wording Claude's discretion, suggestion: `codegraph: pebble: `).
- **D-05:** NO new env escape hatch (`CODEGRAPH_PEBBLE_LOG`-style) to re-enable INFO logs in v1.0 — the discard is unconditional. TS has no analogue (better-sqlite3 doesn't chatter, which is exactly why this is a parity gap); a new env var is new documented/audited surface for a debugging convenience nobody has asked for. Real errors still surface via D-02. A general verbose/debug knob is a Deferred idea.

**MCP stdout cleanliness (HYG-02)**
- **D-06:** Two-layer enforcement, belt and braces (Phase-3 convention):
  - **(a) Subprocess-harness assertion:** ride the existing `test/integration/` harness (Phase-3 D-17..D-21 — real binary, real stdio JSON-RPC; do NOT invent a second harness). A real `serve --mcp` session that exercises real store activity (the startup reconcile `indexer.Sync` + `initialize` → `tools/call`) must produce stdout where EVERY line parses as a JSON-RPC frame (`json.Unmarshal` succeeds AND the `jsonrpc` field is present); any non-frame byte on stdout fails the test.
  - **(b) Structural stdout guard:** a new archtest mirroring the two existing precedents (`internal/graphstore/archtest/import_graph_test.go`, `internal/migrate/archtest/modernc_confinement_test.go`) fails if any serve-reachable non-CLI package (`internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query`) references `os.Stdout`, calls bare stdout-writing `fmt.Print*` (no explicit writer), or calls `log.SetOutput`. `internal/cli` is EXCLUDED — it legitimately renders command output to `cmd.OutOrStdout()` (03-PATTERNS.md output-discipline pattern). Mechanism (go/ast walk vs token scan) is Claude's discretion; it must be a normal Go test so `go test ./...` runs it.
  - Note: `internal/daemon`'s existing `log.Printf` calls are COMPLIANT — stdlib log defaults to stderr; the guard targets stdout references, not stdlib-log usage.
- **D-07:** HYG-02 is expected to be a guarantee-and-regression-lock, not a behavior change: scouting found zero stdout writers in the guarded packages today (the only `os.Stdout` match is a comment). If the archtest or harness DOES surface a real violation, fixing it is in scope for this phase — that's the point of running the guard before claiming the requirement.

**Verification (both requirements)**
- **D-08:** Mutation-proof the HYG-01 wiring (the 8×-recurred green-suite lesson: assert the wiring, not a replica). A test must assert `graphstore.Open` actually passes the quiet logger — reverting pebble_store.go:147 to `&pebble.Options{}` must turn it red. Plus a behavioral test: capture the D-04 seam writer during store activity that provokes pebble INFO output (open/write/flush/close cycles emit WAL/job lines under the default logger) and assert zero pebble noise, while a directly-invoked `Errorf` still reaches the writer.
- **D-09:** One cheap CLI-side behavioral check: a normal command driven end-to-end (e.g. `sync` or `status` via the subprocess harness) asserts its stderr carries no pebble-shaped noise (no `[JOB `-style / WAL / compaction lines) — absence-of-substring on noise shapes only, NOT emptiness (legit codegraph warnings may appear). Placement and command choice are Claude's discretion.
- **D-10:** No new CI steps needed: the archtest lives under a normal package (covered by `go test ./...`) and the harness additions live in `test/integration/` (covered by Phase 3's explicit named CI step). Verify both remain green rather than adding steps.

### Claude's Discretion
- Logger type name and file placement inside `internal/graphstore`; exact provenance-prefix wording; archtest implementation mechanism (go/ast vs scanner); which command the D-09 CLI-noise check drives and where it lives; whether the quiet logger buffers/rate-limits repeated Errorf lines (only if trivially cheap — otherwise pass-through).

### Deferred Ideas (OUT OF SCOPE)
- **Verbose/debug knob to re-enable pebble INFO logs** (env or flag) — rejected for v1.0 (D-05); revisit only if real-world store debugging demands it (Phase 8 surface-reconciliation candidate at the earliest).
- **TUI-01 lipgloss/bubbletea import-graph archtest** (Phase 6) — the D-06(b) stdout guard is a sibling precedent, not a substitute; Phase 6 should mirror it for the Charm packages.
- **Shared in-process store handle** to close the residual >400ms lock-contention window (Phase-3 residual) — future design work, untouched by this phase.
- **Not in this phase:** the TTY-gated lipgloss rendering seam and its import-graph archtest (TUI-01/02/05, Phase 6), git sync hooks (HOOK, Phase 5), any change to WHAT commands print as their own product output (only *library* noise is in scope), the >400ms shared-store-handle lock-contention window (Phase-3 residual design work), any new user-facing verbosity/debug surface.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| HYG-01 | Pebble's internal WAL/INFO log noise no longer prints on any command (explicit `pebble.Options.Logger` routing INFO→discard) while real errors are preserved | Confirmed exact `Logger`/`EventListener`/`EnsureDefaults` mechanics against the pinned `pebble/v2@v2.1.6` source (module cache), the single `pebble.Open` call site, and the concrete Infof call sites (`open.go:383/385/1062`, `compaction_picker.go:1374`, `obsolete_files.go:307/312`) that will need to go silent — see Architecture Patterns and Code Examples. |
| HYG-02 | No library log output ever reaches MCP stdout — JSON-RPC framing stays clean; diagnostics go to stderr only | Confirmed today's zero-violation baseline, the two archtest precedents' mechanism (`go/packages` import-graph, not regex), and a **critical harness pitfall**: `mcp-go`'s stdio client (`client/transport/stdio.go`) silently `continue`s past any stdout line that fails `json.Unmarshal` — reusing it as-is for the frame-purity assertion would produce a test that can never fail. See Common Pitfalls #1 and Code Examples for the required raw-capture harness variant. |
</phase_requirements>

## Summary

This phase is small, mechanical, and almost entirely verifiable by reading the pinned dependency's actual source rather than external docs — which is what this research did. Two independent findings sharpen the plan beyond what CONTEXT.md's decisions already lock in:

**HYG-01 mechanics are simpler than the "WAL/compaction chatter routes through EventListener" framing might suggest.** Reading `pebble/v2@v2.1.6/event.go`'s `EventListener.EnsureDefaults` shows almost every event callback (`CompactionBegin/End`, `FlushBegin/End`, `TableCreated`, etc.) already defaults to a no-op function regardless of the Logger — only `BackgroundError` (routes to `Logger.Errorf`) and `DataCorruption` (routes to `Logger.Fatalf`) are logger-derived. The actual "WAL/INFO chatter" everyone means comes from **direct `opts.Logger.Infof(...)` calls scattered through pebble's own code** — `open.go:383` ("Found %d WALs"), `open.go:1062` ("[JOB %d] WAL %s stopped reading..."), `compaction_picker.go:1374` ("pickAuto: L%d->L%d..."), `obsolete_files.go:307/312` (cleanup queue backlog). Setting `Options.Logger` to a quiet-Infof logger silences all of this at the source; D-03's "EventListener derives from Logger" claim is still correct for the two logger-derived callbacks, it's just not where most of the visible noise originates. Critically, `open.go:383` ("Found %d WALs") fires on **every** `Open` call, even against a brand-new empty store (`len(wals)==0`) — this is the cheapest, most deterministic hook for the D-08 behavioral test.

**HYG-02's frame-purity test cannot be built on top of the existing `newServeClient`/`mcpclient.Client` helper.** `mcp-go@v0.56.0`'s stdio transport (`client/transport/stdio.go:readResponses`) reads stdout line-by-line and does `if err := json.Unmarshal(...); err != nil { continue }` — any non-JSON-RPC line is **silently skipped**, not surfaced as an error, and the client only exposes `Stderr()` (wired through `client.GetStderr`), never the raw stdout stream. A test built on `c.Initialize`/`c.CallTool` succeeding would pass even if garbage lines were interleaved on stdout, because each line is parsed independently and the well-formed response line still gets through. D-06(a)'s assertion needs a **new, small, hand-rolled raw-stdio driver** in `test/integration/` (spawn via plain `exec.Command`, own the whole `cmd.Stdout` buffer, read every line yourself) — not a layer on top of `newServeClient`. This is the single most important thing the planner needs to know before writing this phase's tasks.

**Primary recommendation:** Implement HYG-01 as a ~20-30 line unexported `quietLogger` type in `internal/graphstore` (3 methods: `Infof` no-ops, `Errorf` writes to the D-04 stderr seam with a provenance prefix, `Fatalf` calls the real fatal path) injected as `&pebble.Options{Logger: quietLogger{}}` at `pebble_store.go:147`, verified by (1) a wiring test that fails when the Logger field is reverted, and (2) a behavioral test using `pebble.Open`'s own "Found %d WALs" Infof call as the deterministic noise trigger. Implement HYG-02's archtest by extending the existing `golang.org/x/tools/go/packages`-based pattern with `NeedSyntax|NeedTypes|NeedTypesInfo` to walk the AST of the six named packages' production files (not test files) looking for `os.Stdout` selector references, bare `fmt.Print/Println/Printf` calls, and `log.SetOutput` calls. Implement HYG-02's harness assertion as a new hand-rolled raw-stdio JSON-RPC driver alongside (not built on) the existing `newServeClient` helpers.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Pebble Logger injection (HYG-01) | Database/Storage (`internal/graphstore`) | — | The sole pebble-aware package (D-04a) already owns the single `pebble.Open` call site; the logger is a storage-tier concern with no reason to be visible above `graphstore`. |
| Stderr diagnostics writer seam (D-04) | Database/Storage (`internal/graphstore`) | — | Package-local injectable seam, mirroring the existing `openLockRetrySleep` test-seam convention already in the same file. |
| MCP stdout transport purity (HYG-02 structural guard) | API/Backend (`internal/mcp`) + cross-cutting (`internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query`) | — | `internal/mcp` owns the actual JSON-RPC stdio transport (via `server.ServeStdio`); the guard's scope is every package reachable from that server, not just `internal/mcp` itself — a stray print in `internal/indexer` during a watcher-triggered sync is just as fatal to the transport as one in `internal/mcp`. |
| MCP stdout transport purity (HYG-02 harness assertion) | API/Backend (`test/integration`, driving the real binary's CLI/API tier) | — | Black-box, subprocess-level verification; belongs beside the existing Phase-3 harness, not inside any production package. |
| CLI stdout rendering (excluded from HYG-02 guard) | Browser/Client-analog (`internal/cli`, terminal-rendering tier) | — | `internal/cli` legitimately writes product output to `cmd.OutOrStdout()` — a different tier's responsibility (human-facing terminal rendering) that this phase must not touch or break. |

## Standard Stack

No new third-party dependency is introduced by this phase. All work uses the module's existing pinned dependencies and the Go standard library.

### Core (existing, reused)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/cockroachdb/pebble/v2` | v2.1.6 (pinned, `go.mod`) [VERIFIED: `go.mod` + module cache at `$(go env GOMODCACHE)/github.com/cockroachdb/pebble/v2@v2.1.6`] | Embedded KV store; exposes the `base.Logger` interface this phase implements | Already the project's chosen storage engine (v0.1 D-04); HYG-01 is a configuration change at its existing `Open` seam, not a new dependency |
| `github.com/mark3labs/mcp-go` | v0.56.0 (pinned, `go.mod`) [VERIFIED: `go.mod` + module cache; confirmed a stale v0.45.0 module also sits in the local cache from an earlier state — always resolve the version from `go.mod`, not just cache directory listing] | MCP stdio server + the existing test-integration client machinery | Already the project's MCP server library (Phase 3); this phase's harness work rides its existing `test/integration` usage |
| `golang.org/x/tools` (`go/packages`) | v0.48.0 (pinned, `go.mod`) [VERIFIED: `go.mod`] | Import-graph and (for the new guard) AST/type-checked syntax loading | Already used by both existing archtest precedents; extending its `packages.Config.Mode` flags (adding `NeedSyntax`/`NeedTypes`/`NeedTypesInfo`) is the natural, zero-new-dependency path to a structural stdout-reference guard |
| stdlib `log`, `os`, `go/ast`, `go/types` | Go stdlib (toolchain per `go.mod`) [VERIFIED: Go stdlib source, `$(go env GOROOT)/src/log/log.go:87` — `var std = New(os.Stderr, "", LstdFlags)`] | Confirms `log.Printf`/`log.Println` default to stderr; AST walking for the new archtest | No new dependency; `internal/daemon`'s existing `log.Printf` calls are already compliant by construction |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| A hand-written `go/ast` walk over `packages.Load(... NeedSyntax ...)` output | A plain text/regex scan for `"os.Stdout"` / `"fmt.Print"` strings across the six packages' `.go` files | Regex is simpler but exactly the failure mode the two existing archtests' own doc comments reject ("NOT regex/string-matching over source — regex misses aliased imports, build-tag-gated files, and test variants"); an AST+types walk correctly ignores string literals/comments containing the text "os.Stdout" and correctly resolves that the identifier really refers to the `os` package, not a locally shadowed variable named `os`. Reuse the precedent's own reasoning here. |
| A new hand-rolled raw-stdio JSON-RPC driver for D-06(a) | Reusing `newServeClient`/`mcpclient.Client` and asserting only that `Initialize`/`CallTool` succeed | Documented above (Summary) and in Common Pitfalls #1 — the existing client silently discards any non-JSON-RPC line, so it structurally cannot detect the violation this test exists to catch. This is not a stylistic preference; the existing helper is provably inadequate for this specific assertion. |
| `Options.Logger` only (D-03, locked) | Also setting a custom `EventListener` | Rejected by the locked decision itself; research confirms this is safe — `EnsureDefaults` (options.go:1466-1469) only installs a default `EventListener{}` if `o.EventListener == nil`, and that default's `BackgroundError`/`DataCorruption` callbacks route through whatever `o.Logger` was already set to (event.go:960-990) — no independent event-listener wiring is needed to keep those two callbacks routed through the quiet logger's `Errorf`/`Fatalf`. |

**Installation:** None — no `go get`/`go.mod` changes required for this phase.

## Package Legitimacy Audit

**Not applicable.** This phase introduces zero new external packages; every dependency touched (`github.com/cockroachdb/pebble/v2`, `github.com/mark3labs/mcp-go`, `golang.org/x/tools`) is already pinned in `go.mod` and was vetted in prior phases. No `package-legitimacy check` run was needed.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌───────────────────────────────────────────┐
                         │        `codegraph serve --mcp` process     │
                         │                                             │
   stdin (JSON-RPC) ────▶│  internal/mcp (stdio transport, tools.go)  │──▶ stdout (JSON-RPC ONLY)
                         │        │            ▲                       │      ▲
                         │        │ tool call  │ startup reconcile     │      │ MUST STAY CLEAN
                         │        ▼            │ (indexer.Sync)        │      │ (HYG-02 guard)
                         │  internal/query ◀────┤                       │      │
                         │   (Engine, read-only)│                       │      │
                         │        │             │                       │      │
                         │        ▼             ▼                       │      │
                         │  internal/graphstore (pebble.Open, ONLY      │      │
                         │  pebble-aware package — D-04a)                │      │
                         │        │                                     │      │
                         │        ├─ Infof  ──▶ quietLogger ──▶ discard  │      │
                         │        ├─ Errorf ──▶ quietLogger ──▶ stderr ──┼──────┘  (never stdout)
                         │        └─ Fatalf ──▶ quietLogger ──▶ stderr, os.Exit(1)
                         │                                             │
                         │  internal/daemon / internal/watch          │
                         │   (background watcher goroutine, off        │
                         │    the handshake path — WATCH-02)           │
                         │        └─ log.Printf ──▶ stdlib log ──▶ stderr (already compliant)
                         └───────────────────────────────────────────┘

  test/integration (subprocess harness, black-box):
    spawn real binary ──▶ own the WHOLE cmd.Stdout buffer (NOT via mcp-go's client) ──▶
    assert every line == valid JSON-RPC frame (D-06a)

  archtest (build-time, go/packages + go/ast):
    load {mcp, graphstore, daemon, watch, indexer, query} production syntax ──▶
    fail build if any package references os.Stdout / bare fmt.Print* / log.SetOutput (D-06b)
```

A reader can trace HYG-01 top-to-bottom on the left column (Infof/Errorf/Fatalf routing) and HYG-02 on the right column (stdout stays JSON-RPC-only, enforced both at runtime by the harness and at build time by the archtest).

### Recommended Project Structure
No new directories. Two new files are the expected minimal footprint:
```
internal/graphstore/
├── pebble_store.go       # MODIFIED: Options{Logger: quietLogger{}} at line 147
├── logger.go             # NEW: quietLogger type (D-01/D-02/D-03) + stderr writer seam (D-04)
├── logger_test.go        # NEW: D-08 wiring + behavioral tests
internal/graphstore/archtest/
├── import_graph_test.go  # EXISTING — pattern to extend/mirror
├── stdout_confinement_test.go  # NEW: D-06(b) structural guard (or a similarly named sibling file)
test/integration/
├── mcp_stdout_purity_test.go   # NEW: D-06(a) raw-stdio frame-purity case
├── (existing files gain D-09's CLI-noise-absence case, or it lands in a new small file)
```

### Pattern 1: Minimal pebble.Logger implementation, injected at the single Open seam
**What:** A 3-method unexported type satisfying `pebble.Logger` (`= base.Logger`, verified alias at `pebble/v2@v2.1.6/logger.go:10`), set as `Options.Logger` only.
**When to use:** Exactly once, at `graphstore.Open`'s `pebble.Open(dir, &pebble.Options{...})` call.
**Example:**
```go
// Source: verified against github.com/cockroachdb/pebble/v2@v2.1.6
// internal/base/logger.go (Logger interface) and options.go (Options.Logger,
// EnsureDefaults, LoggerAndTracer precedence) — module cache read directly.

// quietLogger implements pebble.Logger (D-01): Infof is the WAL/compaction/
// memtable chatter pebble emits directly (open.go's "Found %d WALs",
// compaction_picker.go's "pickAuto: ..."), discarded unconditionally (D-02).
// Errorf is real diagnostic signal (background-error/metrics-error paths)
// and is preserved via the D-04 stderr seam with a provenance prefix.
// Fatalf preserves pebble's own semantics: message out, then exit — pebble
// only calls Fatalf on invariant violations where continuing is unsafe
// (e.g. version_set.go's "MANIFEST not locked for writing"), so softening
// it would hide exactly the corruption signal this phase must never hide.
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
```go
// pebble_store.go:147 — the one-line change at the single Open seam.
db, err := pebble.Open(dir, &pebble.Options{Logger: quietLogger{}})
```

### Pattern 2: go/packages AST-based structural guard (extends the existing import-graph precedent)
**What:** Both existing archtests (`TestNoPackageBypassesGraphStore`, `TestModerncSQLiteConfinedToMigrate`) use `packages.Config{Mode: packages.NeedImports | packages.NeedName | packages.NeedDeps, Tests: true}` and inspect `pkg.Imports`. The new HYG-02 guard needs actual syntax, not just the import graph, so it must add `NeedSyntax | NeedTypes | NeedTypesInfo | NeedFiles | NeedCompiledGoFiles` to the mode and walk each target package's `ast.File`s with `go/ast.Inspect`.
**When to use:** For the D-06(b) structural guard, scoped to exactly the six named packages (`internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query`).
**Example:**
```go
// Source: pattern extends golang.org/x/tools/go/packages usage already
// established in internal/graphstore/archtest/import_graph_test.go and
// internal/migrate/archtest/modernc_confinement_test.go (both read directly
// from this codebase). NOT regex/string scanning — same rationale those
// two files already document.
package archtest

import (
	"go/ast"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

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
		// Tests: false (deliberate divergence from the two import-confinement
		// precedents above): a _test.go file printing to stdout during
		// `go test` never touches the real `serve --mcp` binary's stdout —
		// only production compilation units matter for this guard.
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

// isOSStdoutRef/isBareFmtPrint/isLogSetOutput resolve identifiers via
// pkg.TypesInfo.Uses (types.Object.Pkg().Path()) rather than string-matching
// the identifier name, so a locally shadowed variable named "os" or "fmt"
// cannot produce a false positive — the same package-graph precision the
// existing precedents already apply at the import level.
```

### Pattern 3: Raw-stdio harness for the D-06(a) frame-purity assertion (does NOT reuse the mcp-go client)
**What:** A dedicated helper that spawns the real binary via plain `exec.Command`, owns `cmd.Stdout` directly (a pipe or buffer the test reads itself, line-by-line, asserting purity on every line), and speaks minimal hand-framed JSON-RPC over `cmd.Stdin`/`cmd.Stdout` to drive `initialize` → `tools/call`.
**When to use:** Exactly once, for the new D-06(a) test case. `newServeClient` and friends remain correct and unmodified for every other existing MCP-behavior test in `test/integration/` — this is an additive sibling helper, not a replacement.
**Example:**
```go
// Source: this codebase's test/integration/worktree_notice_test.go (existing
// newServeClient pattern) + github.com/mark3labs/mcp-go@v0.56.0's
// client/transport/stdio.go (read directly from the module cache) — that
// file's readResponses does `if err := json.Unmarshal(...); err != nil {
// continue }` on every stdout line, silently discarding anything that
// doesn't parse. That makes the existing client UNSUITABLE for a
// frame-purity assertion: a garbage line preceding a well-formed response
// would never fail the existing helper. This raw variant reads every byte
// itself and fails loudly on the first non-frame line.
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
	// Every line this scanner reads MUST satisfy:
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

### Anti-Patterns to Avoid
- **Building the frame-purity assertion on top of `newServeClient`/`mcpclient.Client`:** provably inadequate — see Pattern 3 and Common Pitfalls #1.
- **Regex/string-scanning source files for `os.Stdout`:** both existing archtests explicitly reject this approach in their own doc comments for the import-graph case; the same reasoning (aliased imports, shadowed identifiers, comments/strings containing the literal text) applies even more strongly to a token scan for stdout references.
- **Wrapping pebble's `EventListener` instead of just `Options.Logger`:** unnecessary — `EnsureDefaults` already derives the two logger-relevant event callbacks (`BackgroundError`, `DataCorruption`) from `o.Logger`; adding a custom `EventListener` would duplicate that wiring for no benefit and risks silently overriding pebble's own defaults for the ~20 other callbacks that are supposed to stay no-ops.
- **Softening `Fatalf` to a non-fatal log line:** explicitly rejected by D-02 — pebble's own `Fatalf` call sites (e.g. `version_set.go`'s MANIFEST-lock invariant checks, `obsolete_files.go`'s sort-invariant checks) are places where continuing execution is unsafe; converting them to non-fatal would violate the phase's own "never hide store-corruption errors" guardrail.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting whether a package imports pebble/modernc-sqlite/etc. | A custom `go/build`-based import walker | `golang.org/x/tools/go/packages` (already the project's chosen tool for this, used twice) | Already vetted, already in `go.mod`, and its `Tests: true` mode correctly surfaces bypasses hidden in `_test.go` files that a naive walker would miss. |
| Verifying "does this package reference os.Stdout" | A regex/string scan | `go/packages` with `NeedSyntax`/`NeedTypes`/`NeedTypesInfo` + `go/ast.Inspect` | Regex cannot distinguish a real `os.Stdout` selector reference from the same text appearing in a comment, a string literal, or a locally shadowed identifier named `os`. Type-checked AST inspection resolves the actual package path per identifier use. |
| A minimal JSON-RPC stdio client for the purity test | A second general-purpose MCP client library or a hand-rolled full client | The narrowest possible hand-rolled reader: `cmd.StdoutPipe()` + `bufio.Scanner` + `json.Unmarshal` per line, purpose-built ONLY to assert purity, not to be a reusable client | The existing `mcp-go` client is the correct, full-featured tool for every OTHER test in `test/integration/` — this is the one narrow case where its designed-in tolerance for malformed lines (silently skipping them) is disqualifying, so a tiny bespoke reader is justified rather than "not hand-rolling" as a blanket rule. |

**Key insight:** every "don't hand-roll" item above is really "reuse the tool this codebase already chose for the adjacent problem" — `go/packages` for structural checks, `mcp-go`'s client for protocol-level test driving — except the one narrow spot (raw byte-level purity) where the chosen tool's own designed behavior (tolerant parsing) is precisely what must be bypassed to test what this phase promises.

## Common Pitfalls

### Pitfall 1: The obvious frame-purity test (built on `newServeClient`) can never fail
**What goes wrong:** A test that spawns `serve --mcp` via the existing `newServeClient` helper, calls `Initialize`/`CallTool`, and asserts success looks like it validates HYG-02 — but it doesn't, and it never will, even if a real stdout-pollution regression is introduced.
**Why it happens:** `mcp-go@v0.56.0`'s `client/transport/stdio.go:readResponses` reads stdout line-by-line and does `if err := json.Unmarshal([]byte(line), &baseMessage); err != nil { continue }` — a malformed line is silently skipped, not surfaced. As long as the actual JSON-RPC response line for a given request ID still arrives (on some later line), the client-level call succeeds regardless of what garbage preceded it.
**How to avoid:** Build the purity assertion as a separate, small, raw-stdio reader (Pattern 3) that reads and validates EVERY line itself, independent of `mcp-go`'s client. Do not layer this assertion on `Client.Initialize`/`Client.CallTool` success.
**Warning signs:** A "frame purity" test whose only assertions are that `Initialize` and `CallTool` return without error — that's testing client-level protocol success, not stdout purity.

### Pitfall 2: Assuming pebble's EventListener is the source of the visible noise
**What goes wrong:** Time is spent building/wiring a custom `pebble.EventListener` to suppress "compaction/flush noise", when in fact ~20 of the ~22 EventListener callbacks already default to no-ops (`event.go:960-1024`+) regardless of the Logger — the visible noise is from **direct `Logger.Infof` calls** pebble makes throughout its own code (open, compaction-picker, obsolete-files cleanup), not from event callbacks.
**Why it happens:** The requirement text and common Pebble-tuning folklore ("route WAL/compaction noise") suggests an EventListener-shaped fix; the locked decision (D-03) correctly identifies that only `Options.Logger` needs setting, but the reasoning ("EnsureDefaults derives EventListener from Logger") undersells that most of the visible noise never touches EventListener at all.
**How to avoid:** Set `Options.Logger` only (as locked); do not add EventListener wiring. Verify via the concrete Infof call sites this research already found (`open.go:383/385/1062`, `compaction_picker.go:1374`, `obsolete_files.go:307/312`).

### Pitfall 3: Trying to unit-test `Fatalf` directly will kill the test binary
**What goes wrong:** `pebble`'s own `DefaultLogger.Fatalf` calls `os.Exit(1)` after logging (base/logger.go:41-45); a naively-designed `quietLogger.Fatalf` that preserves this semantic (as D-02 requires) will also call `os.Exit(1)`. A test that directly invokes the real `quietLogger.Fatalf` (rather than a substitutable/observable path) terminates the whole `go test` process, silently "passing" by aborting rather than by assertion.
**How to avoid:** Test `Fatalf`'s formatting/write behavior by capturing the D-04 writer seam's output BEFORE the `os.Exit(1)` call, or by extracting the message-formatting logic to a helper that `Fatalf` calls before exiting — verify the message content and provenance prefix through that helper, and treat `os.Exit(1)` itself as an untested (but code-reviewed) one-liner, exactly the same tradeoff pebble's own `DefaultLogger.Fatalf` and `base.InMemLogger.Fatalf` make (the latter just prepends "FATAL: " and logs — it does NOT call os.Exit, precisely because it's the test-double shape). Consider mirroring `InMemLogger`'s approach: the D-04 seam capturing everything through `Errorf`-shaped output, with `Fatalf` sharing that formatting path and *only* additionally calling `os.Exit(1)` as a final, untested statement.
**Warning signs:** A test that "passes" suspiciously fast or that other tests in the same file mysteriously stop running (the whole test binary process exited).

### Pitfall 4: `packages.Load` silently returns nothing if given the wrong pattern shape
**What goes wrong:** Passing package import paths that don't resolve (typo, wrong `go.mod` module prefix) makes `packages.Load` return an empty or unhelpful result rather than an obvious error, exactly the failure the existing archtest precedents already guard against with their own "sanity check" (`if !foundGraphstoreImporter { t.Fatal(...) }`).
**How to avoid:** Mirror that same sanity-check pattern for the new HYG-02 guard: after loading, assert that syntax was actually found for a package known to exist (e.g. assert `len(pkgs) == 6` or that each named package's `Syntax` is non-empty) so a future refactor that renames one of the six guarded packages can't silently stop checking it.
**Warning signs:** A guard test that always passes even against a deliberately-reintroduced violation during manual verification.

### Pitfall 5: Confusing "goes to stderr" with "is quiet"
**What goes wrong:** Assuming HYG-01 is done once pebble's noise is confirmed to go to stderr (not stdout) — it already does, by default, since `DefaultLogger.Infof/Errorf/Fatalf` all call `log.Output(2, ...)` and stdlib `log`'s default output is `os.Stderr` (`$(go env GOROOT)/src/log/log.go:87`). HYG-01 is about ELIMINATING the noise from stderr (a UX/clutter fix), not about redirecting it away from stdout (that's already true today and is HYG-02's separate concern).
**How to avoid:** Keep the two requirements' scopes distinct in the plan: HYG-01 = quiet Infof + preserved Errorf/Fatalf on stderr (a behavior change users will notice — less clutter); HYG-02 = a regression-lock proving nothing (now or in the future) writes non-JSON-RPC bytes to stdout (structurally a no-op today, per D-07).

## Code Examples

Verified patterns from the pinned dependency's actual source (module cache, not web docs):

### Pebble's Logger interface (the exact 3 methods to implement)
```go
// Source: github.com/cockroachdb/pebble/v2@v2.1.6/internal/base/logger.go:18-23
// (read directly from $(go env GOMODCACHE))
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}
```

### Confirmation that `Options.Logger` alone is sufficient (no `LoggerAndTracer` needed)
```go
// Source: github.com/cockroachdb/pebble/v2@v2.1.6/open.go:84,91-95
opts.EnsureDefaults() // sets opts.Logger = DefaultLogger if nil (options.go:1463-1464)
// ...
if opts.LoggerAndTracer == nil {
	// Since D-03 sets neither tracer, this branch always runs: our
	// quietLogger gets wrapped in a no-op tracer automatically.
	opts.LoggerAndTracer = &base.LoggerWithNoopTracer{Logger: opts.Logger}
} else {
	opts.Logger = opts.LoggerAndTracer
}
```

### Where pebble's Infof chatter actually originates (deterministic test hook)
```go
// Source: github.com/cockroachdb/pebble/v2@v2.1.6/open.go:379-386
// Fires on EVERY Open call, even a brand-new empty store (len(wals)==0) —
// the cheapest, most deterministic trigger for the D-08 behavioral test.
wals, err := wal.Scan(walDirs...)
// ...
d.opts.Logger.Infof("Found %d WALs", redact.Safe(len(wals)))
for i := range wals {
	d.opts.Logger.Infof("  - %s", wals[i])
}
```
Other confirmed Infof/Fatalf emission sites worth knowing about (all pinned to `pebble/v2@v2.1.6`, read directly): `open.go:1062` (`"[JOB %d] WAL %s stopped reading at offset..."` — the exact `[JOB ` shape D-09's absence check should match against), `compaction_picker.go:1374` (`"pickAuto: L%d->L%d..."`), `obsolete_files.go:307,312` (cleanup-queue backlog messages), and roughly a dozen `Fatalf` call sites in `compaction.go`, `version_set.go`, `format_major_version.go`, `db.go`, `ingest.go`, `table_stats.go`, `read_state.go`, `obsolete_files.go` — all invariant-violation/corruption paths, confirming D-02's "Fatalf only fires when continuing is unsafe" framing.

### The module's one and only `pebble.Open` call site
```go
// Source: internal/graphstore/pebble_store.go:147 (this codebase, current state)
db, err := pebble.Open(dir, &pebble.Options{})
// grep confirms no other pebble.Open call exists anywhere in the module.
```

## State of the Art

Not applicable in the usual sense — this is a first-time implementation of logger routing and a stdout-purity guard for this codebase, not a migration from an older approach. The one relevant "old → current" framing:

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `&pebble.Options{}` (pebble's `DefaultLogger`, which writes every Infof/Errorf/Fatalf to stdlib `log` → stderr, unfiltered) | `&pebble.Options{Logger: quietLogger{}}` (Infof discarded, Errorf/Fatalf preserved with provenance prefix) | This phase | Every command that opens the store (query commands, `sync`, `serve --mcp`'s startup reconcile, the watcher's debounced flushes) stops printing WAL/compaction/job-queue chatter to stderr, while real background errors and fatal invariant violations remain visible. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The exact provenance-prefix wording (`codegraph: pebble: `) is a suggestion, not a verified requirement — CONTEXT.md itself marks this "Claude's discretion" | Code Examples, Pattern 1 | None — explicitly discretionary, not a fact needing confirmation. |
| A2 | Recommending `Tests: false` for the new HYG-02 archtest's `packages.Config` (a deliberate divergence from the two existing precedents, which use `Tests: true`) is this research's own reasoning, not something verified against a prior review round | Pattern 2 | Low — if a reviewer disagrees, flipping to `Tests: true` and filtering out `_test.go`-suffixed `pkg.PkgPath` variants (using the existing `stripTestVariant` helper's approach) is a small, mechanical change; worth flagging for the planner to confirm during plan review rather than treating as locked. |

**All other claims in this research were verified by direct source inspection** (the pinned `pebble/v2@v2.1.6` and `mcp-go@v0.56.0` module-cache sources, the Go stdlib `log` package source, and this codebase's own files) rather than external documentation or training-data recall — no `[ASSUMED]`-tagged package names or unverified library facts appear above.

## Open Questions

1. **Should the D-09 CLI-noise-absence check run against `sync` or `status`?**
   - What we know: Both commands open the store (triggering the "Found %d WALs" Infof and possibly compaction/cleanup Infofs on a non-trivial fixture); `status` is read-only and cheaper to fixture, `sync` additionally exercises the write path (flush/compaction) and so is a strictly stronger noise-provocation case.
   - What's unclear: Whether the phase wants the cheapest possible check (status) or the most noise-provoking one (sync) — CONTEXT.md leaves this as explicit Claude's-discretion (D-09).
   - Recommendation: Use `sync` against the existing `copyFixture(t)` gofixture — it's already available in `test/integration/main_test.go`, requires no new fixture, and exercises strictly more of pebble's Infof surface (flush + possible compaction) than a bare `status` call would on a small fixture.

2. **Does the D-06(b) archtest need `Tests: true` after all?**
   - What we know: The two existing precedents use `Tests: true` specifically to catch a bypass hidden in a `_test.go` file of some OTHER, unexpected package. The new stdout guard is scoped to six NAMED packages' own reachability from `serve --mcp`, where test files never run in production.
   - What's unclear: Whether a future reviewer will want symmetry with the existing precedents' mode flags regardless of this asymmetry in what's being checked.
   - Recommendation: Ship with `Tests: false` (Pattern 2) and document the rationale inline in the new archtest file's doc comment (mirroring the existing files' habit of explaining every non-obvious choice) so a reviewer sees the reasoning rather than an unexplained divergence.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the whole module | ✓ | matches `go.mod` toolchain directive | — |
| `github.com/cockroachdb/pebble/v2@v2.1.6` (module cache) | Direct source verification of Logger/Options/EventListener mechanics | ✓ | v2.1.6 (exact pinned version present in `$(go env GOMODCACHE)`) | — |
| `github.com/mark3labs/mcp-go@v0.56.0` (module cache) | Direct source verification of the stdio client's line-skipping behavior | ✓ | v0.56.0 (a stale v0.45.0 copy also present in cache — always resolve from `go.mod`, not directory listing) | — |
| `git` | Fixture setup in `test/integration/` (unrelated to this phase but shared harness dependency) | ✓ (already required/used by existing Phase-3 tests) | — | Existing tests already `t.Skip` gracefully if git is unavailable |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None — all required tooling is already present and already used by prior phases.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no third-party test framework anywhere in this module) |
| Config file | none — `go test` with package paths and explicit CI steps (`.github/workflows/ci.yml`) |
| Quick run command | `go test ./internal/graphstore/... ./internal/graphstore/archtest/...` |
| Full suite command | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` (mirrors the three explicit CI steps — `go test ./...` alone silently skips `testdata/` and `test/integration/` is already separately named per GOLDEN-01/D-17) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HYG-01 | `graphstore.Open` passes a quiet logger (wiring, D-08) | unit | `go test ./internal/graphstore/ -run TestOpen.*Logger -v` | ❌ Wave 0 (new `logger_test.go`) |
| HYG-01 | Infof noise (e.g. "Found %d WALs") never reaches the D-04 writer seam during real Open/flush/compaction/close activity; a directly-invoked Errorf still reaches it (behavioral, D-08) | unit | `go test ./internal/graphstore/ -run TestQuietLogger -v` | ❌ Wave 0 (new `logger_test.go`) |
| HYG-01 | A driven CLI command's real stderr carries no pebble-shaped noise substrings (D-09) | integration (subprocess) | `go test ./test/integration/ -run TestSyncStderrNoPebbleNoise -v` | ❌ Wave 0 (new test case) |
| HYG-02 | No serve-reachable package references `os.Stdout` / bare `fmt.Print*` / `log.SetOutput` (structural, D-06b) | unit (archtest, `go/packages`+AST) | `go test ./internal/graphstore/archtest/... -run TestNoStdoutNoise -v` | ❌ Wave 0 (new `stdout_confinement_test.go` or similarly named sibling) |
| HYG-02 | Every stdout line from a real `serve --mcp` session (exercising startup reconcile + a tool call) parses as a JSON-RPC frame (behavioral, D-06a) | integration (subprocess, raw stdio) | `go test ./test/integration/ -run TestServeMCPStdoutIsPureJSONRPC -v` | ❌ Wave 0 (new raw-stdio harness helper + test case) |

### Sampling Rate
- **Per task commit:** `go test ./internal/graphstore/... ./internal/graphstore/archtest/...` (fast — no subprocess spawn)
- **Per wave merge:** `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` (full suite, including the subprocess-spawning harness cases)
- **Phase gate:** Full suite green (all three commands above) before `/gsd-verify-work`, per D-10 — no new CI steps needed, both new test locations are already covered by existing explicit CI steps.

### Wave 0 Gaps
- [ ] `internal/graphstore/logger.go` + `internal/graphstore/logger_test.go` — quietLogger type, D-04 writer seam, D-08 wiring + behavioral tests
- [ ] `internal/graphstore/archtest/stdout_confinement_test.go` (or similarly named sibling file) — D-06(b) structural guard
- [ ] A new raw-stdio harness helper in `test/integration/` (e.g. `mcp_stdout_purity_test.go`) — D-06(a), deliberately NOT built on `newServeClient`
- [ ] A new or extended test case in `test/integration/` for D-09's CLI-noise-absence check (can share a file with an existing test or be new — Claude's discretion per CONTEXT.md)
- No framework install needed — stdlib `testing` + already-pinned `golang.org/x/tools/go/packages` cover everything.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This phase touches no authentication surface. |
| V3 Session Management | No | No session concept involved. |
| V4 Access Control | No | No access-control surface changes. |
| V5 Input Validation | No (marginal) | No new external input is parsed by this phase; the quiet logger only formats pebble's own internally-generated diagnostic strings, never user- or attacker-controlled data reaching a new parser. |
| V6 Cryptography | No | Not applicable. |
| V7 Error Handling & Logging (ASVS 4.x numbering; the directly relevant category for this phase) | Yes | Standard control: strict channel separation between the transport (stdout, JSON-RPC only) and diagnostics (stderr, human/log-shaped text) — exactly what HYG-02 mechanizes. Never log secrets or unredacted sensitive data; this phase does not introduce any new logged field, so no new redaction need arises, but the provenance-prefixed Errorf/Fatalf messages should be reviewed at implementation time to confirm they never echo raw store-path contents beyond what's already an accepted exception (T-02-14, existing host-path-in-diagnostics precedent). |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Log/output-stream confusion — diagnostic text reaching the same channel a downstream parser trusts as protocol framing (here: stdout as JSON-RPC transport) | Tampering / Information Disclosure | Strict channel separation (HYG-02's whole purpose): all diagnostics forced to stderr via the D-04 writer seam and the D-06(b) structural guard; the D-06(a) harness closes the loop by proving no other code path can leak a stray byte onto stdout. This is a textbook mitigation for the general "log injection into a machine-parsed channel" pattern, applied here at the process-transport level rather than the more commonly discussed HTTP-log-injection level. |
| Fatal-path denial-of-service via an attacker-triggerable invariant violation (`Logger.Fatalf` calling `os.Exit(1)`) | Denial of Service | Out of scope for this phase to newly mitigate — pebble's own `Fatalf` call sites are internal invariant checks (corruption, MANIFEST-lock violations), not attacker-reachable from typical codegraph usage (no untrusted network input reaches pebble directly); D-02 explicitly preserves this behavior rather than softening it, since hiding a real corruption signal is a worse outcome than the process exiting. Worth a one-line note in the plan's threat-model disposition rather than a mitigation task. |

## Sources

### Primary (HIGH confidence — direct pinned-source verification, module cache)
- `github.com/cockroachdb/pebble/v2@v2.1.6` — `internal/base/logger.go`, `options.go`, `open.go`, `event.go`, `compaction.go`, `compaction_picker.go`, `obsolete_files.go`, `version_set.go` (read directly from `$(go env GOMODCACHE)/github.com/cockroachdb/pebble/v2@v2.1.6/`) — Logger interface, Options.Logger/LoggerAndTracer/EnsureDefaults mechanics, EventListener defaults, every concrete Infof/Fatalf call site cited above.
- `github.com/mark3labs/mcp-go@v0.56.0` — `client/transport/stdio.go`, `client/stdio.go` (read directly from `$(go env GOMODCACHE)/github.com/mark3labs/mcp-go@v0.56.0/`) — the line-skipping `readResponses` behavior that motivates the raw-stdio harness recommendation; confirmed `GetStderr` is exposed but no stdout equivalent exists.
- Go stdlib `log` package (`$(go env GOROOT)/src/log/log.go:87`) — confirms `log`'s default output is `os.Stderr`.
- This codebase directly: `internal/graphstore/pebble_store.go`, `internal/graphstore/archtest/import_graph_test.go`, `internal/migrate/archtest/modernc_confinement_test.go`, `internal/mcp/server.go`, `internal/cli/serve.go`, `internal/cli/query.go`, `internal/daemon/daemon.go`, `test/integration/main_test.go`, `test/integration/worktree_notice_test.go`, `test/integration/watch_default_test.go`, `test/integration/watch_live_sync_test.go`, `.github/workflows/ci.yml`, `go.mod`, `.planning/config.json`.

### Secondary (MEDIUM confidence)
- None used — every non-trivial claim in this research traces to a direct pinned-source or in-repo read (see Primary above), which is a stronger provenance tier than an external documentation citation for this specific, highly version-pinned, code-grounded phase.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every version claim verified against `go.mod` and the actual module cache.
- Architecture: HIGH — the single `pebble.Open` seam, the two archtest precedents' exact mechanism, and the mcp-go client's line-skipping behavior were all confirmed by reading the actual source, not inferred.
- Pitfalls: HIGH — Pitfall 1 (the mcp-go silent-skip issue) is a first-principles finding from reading `stdio.go` directly, not a training-data recollection; Pitfall 2/3/5 are similarly grounded in the pinned pebble source.

**Research date:** 2026-07-16
**Valid until:** Pinned to `pebble/v2@v2.1.6` and `mcp-go@v0.56.0` exactly — re-verify the Logger/EventListener mechanics and the stdio client's line-handling behavior if either dependency is bumped before this phase executes (30-day estimate for the surrounding Go-tooling facts, which are stable; the two pinned-library behavioral claims are valid only as long as those exact versions remain pinned in `go.mod`).
