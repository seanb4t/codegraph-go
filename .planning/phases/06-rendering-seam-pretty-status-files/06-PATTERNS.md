# Phase 6: Rendering Seam & Pretty status/files - Pattern Map

**Mapped:** 2026-07-17
**Files analyzed:** 9 (new) + 5 (modified)
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/cli/present/status.go` | component (renderer) | transform | `internal/query/render_status.go` (`RenderStatusText`) | exact (additive sibling, SURF-06 shape) |
| `internal/cli/present/files.go` | component (renderer) | transform | `internal/query/render_results.go` / `internal/cli/files.go` (`printFileTree`) | exact |
| `internal/cli/present/tty.go` | utility (pure decision fn) | request-response | none direct — new pure-function convention; modeled on `sortedCounts`/helpers in `render_status.go` | role-match |
| `internal/cli/present/progress.go` | component (progress writer) | streaming/event-driven | none direct — new ticker-driven writer; modeled on RESEARCH Pattern 3 sketch | no close analog (new capability) |
| `internal/cli/present/archtest/import_graph_test.go` | test (archtest) | batch (build-time) | `internal/graphstore/archtest/stdout_confinement_test.go` (primary — guarded-set + closure walk) and `internal/graphstore/archtest/import_graph_test.go` (self-defeat guard, `stripTestVariant`) | exact (hybrid of both precedents) |
| `internal/cli/status.go` (modified) | controller (CLI RunE) | request-response | itself — pre-existing file, insertion point only | exact (self) |
| `internal/cli/files.go` (modified) | controller (CLI RunE) | request-response | itself — pre-existing file, insertion point only | exact (self) |
| `internal/cli/init.go` / `index.go` / `sync.go` (modified) | controller (CLI RunE) | event-driven (long-running op + progress) | `internal/cli/sync.go` (`printSyncSummary` call site) as the shared shape across all three | role-match |
| `go.mod` (modified) | config | — | existing `golang.org/x/term v0.45.0 // indirect` line (promote to direct) | exact |

## Pattern Assignments

### `internal/cli/present/status.go` (component, transform)

**Analog:** `internal/query/render_status.go`

**Core pattern to mirror** (verbatim source, `internal/query/render_status.go:170-192` per RESEARCH, confirmed section order):
```go
func RenderStatusText(r StatusResult, projectPath string) string {
	var b strings.Builder
	b.WriteString("CodeGraph Status\n\n")
	fmt.Fprintf(&b, "Project: %s\n", projectPath)
	if warning := r.WorktreeMismatch.Warning(); warning != "" {
		b.WriteString(warning + "\n")
	}
	b.WriteString("\nIndex Statistics:\n")
	writeStatLine(&b, "Files", formatNumber(r.FileCount))
	writeStatLine(&b, "Nodes", formatNumber(r.NodeCount))
	writeStatLine(&b, "Edges", formatNumber(r.EdgeCount))
	writeStatLine(&b, "DB Size", formatMB(r.DbSizeBytes))
	writeStatLine(&b, "Backend", r.Backend)
	writeBreakdownText(&b, "Nodes by Kind:", sortedCounts(r.NodesByKind))
	writeBreakdownText(&b, "Files by Language:", sortedCounts(r.FilesByLanguage))
	writeStatusAdvisories(&b, r, "Pending Changes:", "Reindex recommended:")
	return b.String()
}
```

`present.RenderStatus(r query.StatusResult, projectPath string, w io.Writer)` MUST walk the **same section order** (header → Project → worktree warning → Index Statistics → Nodes by Kind → Files by Language → advisories) applying `lipgloss.Style.Render()` per section/line, never re-deriving counts/sort order. Signature takes an `io.Writer` (not `string`) to match `present.Render*(result, w)` per D-03/CONTEXT.md.

**Data source struct (read-only)** — `internal/query/status.go:46-63` (`StatusResult`) — consume as-is.

**Duplication convention (Open Question #1, recommended by RESEARCH):** `sortedCounts`, `formatNumber`, `formatMB` are unexported in `internal/query` — duplicate small pure-function equivalents locally in `internal/cli/present` rather than exporting, matching this codebase's existing precedent of package-local duplication across the query/cli boundary (see `render_results.go`'s `renderFileTreeMarkdown` vs `internal/cli/files.go`'s `printFileTree`).

---

### `internal/cli/present/files.go` (component, transform)

**Analog:** `internal/cli/files.go` (`printFileTree`) + `internal/query/render_results.go`

**Data source structs (read-only)** — `internal/query/files.go:47-71` (`FileEntry`, `FileTreeNode`, `FilesResult`).

**Core pattern to mirror** — the existing plain tree/flat-list walk in `internal/cli/files.go`'s `RunE` (lines ~57-70, see excerpt below): `present.RenderFiles(r query.FilesResult, w io.Writer)` styles the same `Format == "tree"` branch vs. flat `Path (Language)` branch, adding lipgloss key-bolding / indentation styling only — never recomputing the tree structure.

---

### `internal/cli/present/tty.go` (utility, pure decision fn)

**No direct in-repo analog** — this is RESEARCH's own recommended new pattern (Pattern 1), justified against Pitfall 3 (in-process CLI tests can't exercise a hardcoded `os.Stdout.Fd()` check). Implement verbatim as sketched:
```go
// ChoosePresentation reports whether the pretty (lipgloss) branch should
// render, per D-04/D-05: isTTY must be true AND NO_COLOR must be unset/empty.
func ChoosePresentation(isTTY bool, noColor string) bool {
	return isTTY && noColor == ""
}
```
Real fd/env values are read ONLY at the `RunE` call sites (`status.go`, `files.go`, `init.go`, `index.go`, `sync.go`) — never inside `present` itself (Anti-Pattern warning in RESEARCH).

---

### `internal/cli/present/progress.go` (component, streaming)

**No direct in-repo analog** — new capability. Modeled on RESEARCH Pattern 3's ticker sketch (verbatim, illustrative — not copied from any upstream source):
```go
type Spinner struct {
	frames []string
	style  lipgloss.Style
	w      io.Writer
	stop   chan struct{}
}

func NewSpinner(w io.Writer) *Spinner { /* ... */ }
func (s *Spinner) Start(label string) { /* time.Ticker loop, \r redraw */ }
func (s *Spinner) Stop()              { close(s.stop) }
```
MUST write only to the writer passed in (call sites pass `os.Stderr` per D-08 — never `os.Stdout`). MUST NOT import `charm.land/bubbles` or `charm.land/bubbletea` (Pitfall 1) — hand-roll with `lipgloss.Style.Render()` + stdlib `time.Ticker` only.

---

### `internal/cli/present/archtest/import_graph_test.go` (test, archtest — TUI-01)

**Primary analog:** `internal/graphstore/archtest/stdout_confinement_test.go` — copy its **guarded-set + closure-walk** shape (`guardedPackages`, `isModuleInternalPackage`, `closeOverServeReachableImports`) since TUI-01's requirement ("named packages may NOT import X/Y/Z") is structurally the inverse of `import_graph_test.go`'s "only package X may import Y" and matches `stdout_confinement_test.go`'s own inverse shape exactly.

**Guarded set to reuse verbatim** (`internal/graphstore/archtest/stdout_confinement_test.go:36-43`):
```go
var guardedPackages = []string{
	"github.com/seanb4t/codegraph-go/internal/mcp",
	"github.com/seanb4t/codegraph-go/internal/graphstore",
	"github.com/seanb4t/codegraph-go/internal/daemon",
	"github.com/seanb4t/codegraph-go/internal/watch",
	"github.com/seanb4t/codegraph-go/internal/indexer",
	"github.com/seanb4t/codegraph-go/internal/query",
}
```
(D-11 names this exact six-package set for defense-in-depth; `internal/query`/`internal/mcp` are the hard requirement.)

**Secondary analog:** `internal/graphstore/archtest/import_graph_test.go` — copy the **self-defeat guard** (D-12) and `stripTestVariant` helper verbatim:
```go
const (
	pebbleImportPath      = "github.com/cockroachdb/pebble/v2" // → swap for lipgloss/v2, bubbletea/v2, bubbles/v2
	allowedImporterPrefix = "github.com/seanb4t/codegraph-go/internal/graphstore" // → n/a for this archtest's inverse shape
)

cfg := &packages.Config{
	Mode:  packages.NeedImports | packages.NeedName | packages.NeedDeps,
	Tests: true, // catches a bypass hidden in a _test.go file too
}
pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
// ...
if !foundGraphstoreImporter { // → rename: foundCharmImporter
	t.Fatal("no package was found importing <forbidden path> — this test cannot verify enforcement")
}

func stripTestVariant(pkgPath string) string {
	if i := strings.IndexByte(pkgPath, ' '); i >= 0 {
		pkgPath = pkgPath[:i]
	}
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	pkgPath = strings.TrimSuffix(pkgPath, ".test")
	return pkgPath
}
```

**Forbidden import paths (D-11, /v2-corrected per RESEARCH Finding 1 — CRITICAL, do not use the bare paths):**
```go
var forbiddenImportPaths = []string{
	"charm.land/lipgloss/v2",
	"charm.land/bubbletea/v2",
	"charm.land/bubbles/v2",
}
```

**Build order (D-12, ROADMAP-locked):** this archtest file lands FIRST, before `internal/cli/present` exists or any Charm dep enters `go.mod` — verify it fails-closed (self-defeat guard fires) before adding `present` + `lipgloss/v2`.

---

### `internal/cli/status.go` (modified — CLI seam, TUI-02)

**Current RunE tail** (verbatim, `internal/cli/status.go:56-63`, this repo's current state):
```go
// RenderStatusText already embeds the verbose worktree warning
// (from result.WorktreeMismatch, live since plan 02-04) at D-09's
// structural position — no separate warning print here, which
// would double it.
fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
return nil
```
**Change:** insert a single branch immediately before this line — no other line changes (D-06):
```go
if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
	return present.RenderStatus(result, start, cmd.OutOrStdout())
}
fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
return nil
```
The `status.go:18-20` doc comment ("Phase 6 (TUI-02) will colorize...") should be updated/removed once this lands — it's the exact forward-reference this phase resolves.

---

### `internal/cli/files.go` (modified — CLI seam, TUI-02)

**Current RunE tail** (verbatim, `internal/cli/files.go:57-70`, this repo's current state):
```go
out := cmd.OutOrStdout()
fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
if result.Format == "tree" {
	printFileTree(out, result.Tree, "")
} else {
	for _, f := range result.Files {
		fmt.Fprintf(out, "%s (%s)\n", f.Path, f.Language)
	}
}
return nil
```
**Change:** same isTTY branch inserted before the tree/flat `if`, gating the whole block vs. `present.RenderFiles(result, out)` (worktree notice printing stays in the plain path per D-02 unless `present` also renders it — Claude's discretion on exact placement, bounded by "plain path byte-identical").

---

### `internal/cli/init.go` / `index.go` / `sync.go` (modified — progress seam, TUI-05)

**Analog:** `internal/cli/sync.go`'s existing `RunE` → `indexer.Sync(...)` → `printSyncSummary(cmd, stats, quiet, verbose)` call shape (verbatim excerpt, current state):
```go
storeDir := filepath.Join(codegraphDir, storeDirName)
stats, err := indexer.Sync(root, storeDir, indexer.Options{
	Workers: workers,
	Verbose: verbose,
	Quiet:   quiet,
})
if err != nil {
	return err
}
printSyncSummary(cmd, stats, quiet, verbose)
return nil
```
**Change:** wrap the long-running `indexer.Sync`/`indexer.Run` call with `present.NewSpinner(os.Stderr)`'s `Start`/`Stop`, gated by the same `ChoosePresentation` check evaluated against `os.Stderr`'s fd (D-08 — progress checks stderr's TTY-ness, not stdout's, since progress writes to stderr). `init.go` and `index.go` share the identical shape around their own `indexer.Run`/rebuild calls. `--quiet`/`--verbose` flags already exist on all three commands — the spinner MUST also respect `quiet` (no frames when `--quiet`).

---

### `go.mod` (modified — config)

**Current line to promote** (`go.mod:134`):
```
golang.org/x/term v0.45.0 // indirect
```
**Change:** `go get charm.land/lipgloss/v2@v2.0.5 && go mod tidy` — this both adds the new `charm.land/lipgloss/v2` direct require and promotes `golang.org/x/term` from `// indirect` to a direct require (since `internal/cli/present`/`internal/cli/status.go` will import it directly). **Do NOT** add `charm.land/bubbles` or `charm.land/bubbletea` (any version, with or without `/v2`) — grep `go.mod` for `bubbletea` as a completion check; it must return nothing in this phase.

## Shared Patterns

### TTY + NO_COLOR gate (D-04/D-05)
**Source:** New `internal/cli/present/tty.go` (`ChoosePresentation`), RESEARCH Pattern 1
**Apply to:** All five `RunE` call sites (`status.go`, `files.go`, `init.go`, `index.go`, `sync.go`) — one implementation, real fd/env read only at each call site:
```go
isTTY := term.IsTerminal(int(os.Stdout.Fd())) // or os.Stderr.Fd() for progress call sites
if present.ChoosePresentation(isTTY, os.Getenv("NO_COLOR")) {
	// pretty branch
}
```

### Additive-sibling renderer (SURF-06 shape, D-02)
**Source:** `internal/query/render_status.go` / `render_results.go` / `render_markdown.go` — `Render*` functions living beside `Marshal*JSON`, originals untouched.
**Apply to:** `present.RenderStatus`/`present.RenderFiles` — new siblings beside `query.RenderStatusText`/the files plain writer; neither existing function's body changes by one byte.

### Archtest mechanism (go/packages, not regex) — D-10
**Source:** `internal/graphstore/archtest/import_graph_test.go` (self-defeat guard, `stripTestVariant`) + `internal/graphstore/archtest/stdout_confinement_test.go` (guarded-set closure walk)
**Apply to:** `internal/cli/present/archtest/import_graph_test.go` — `packages.Load` with `Tests: true`, `NeedImports|NeedName|NeedDeps`, checking each of the six `guardedPackages` for any of the three `/v2`-suffixed forbidden import paths, plus the self-defeat guard.

### stderr-only progress discipline (D-08, mirrors Phase-4 HYG-02)
**Source:** `internal/graphstore/archtest/stdout_confinement_test.go`'s `isOSStdoutRef`/`isBareFmtPrint` predicates (the enforcement mechanism, not directly reusable code but the discipline this phase's progress writer must itself honor)
**Apply to:** `internal/cli/present/progress.go` — every write goes to the `io.Writer` passed by the caller (which MUST be `os.Stderr` at call sites), never `os.Stdout`; verify with a local `progress_test.go` asserting zero stdout writes (mirrors `stdout_detection_selftest_test.go`'s self-test-can-fail discipline).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/cli/present/progress.go` | component | streaming/event-driven | No existing ticker-driven/animated-output code in this codebase; net-new capability per RESEARCH Pattern 3 (hand-rolled, not copied from any upstream library since bubbles is excluded) |
| `internal/cli/present/tty.go` | utility | request-response | No existing pure-decision-function convention isolated this way; net-new pattern justified by Pitfall 3 (testability without a real pty) |

## Metadata

**Analog search scope:** `internal/cli/`, `internal/query/`, `internal/graphstore/archtest/`, `internal/migrate/archtest/`, `go.mod`
**Files scanned:** `internal/cli/status.go`, `internal/cli/files.go`, `internal/cli/init.go`, `internal/cli/index.go`, `internal/cli/sync.go`, `internal/query/render_status.go`, `internal/query/status.go`, `internal/query/files.go`, `internal/graphstore/archtest/import_graph_test.go`, `internal/graphstore/archtest/stdout_confinement_test.go`, `go.mod`
**Pattern extraction date:** 2026-07-17
