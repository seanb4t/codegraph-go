# Phase 6: Rendering Seam & Pretty status/files - Research

**Researched:** 2026-07-17
**Domain:** Go CLI TTY-gated rendering (Charm lipgloss v2) + build-enforced import-graph isolation
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Rendering seam & package placement (TUI-02)**
- **D-01:** A new `internal/cli/present` package is the sole home for `charm.land/lipgloss` styling. It consumes already-computed plain data (`query.StatusResult`, the `files` result struct) and emits colorized, sectioned output. `internal/query` and `internal/mcp` never import charm.
- **D-02:** The plain path stays byte-identical by reuse, not re-derivation. `query.RenderStatusText` and the `files` plain/markdown renderers (`render_results.go` / `render_markdown.go` / `files.go`) are UNCHANGED. The `present` pretty renderer is an additive sibling.
- **D-03:** Branch selection happens once, at the CLI boundary (`internal/cli/status.go`, `internal/cli/files.go` `RunE`): `isTTY && colorEnabled` → `present.Render*(result, w)`; else → the existing plain writer.

**TTY detection & color policy (TUI-02)**
- **D-04:** TTY detection via `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))`, evaluated at the CLI boundary against the real output stream. Do not rely on lipgloss's own color-profile auto-degrade for the piped guarantee — when not a TTY, take the explicit plain branch and bypass lipgloss entirely.
- **D-05:** Honor `NO_COLOR` (de-facto standard): a non-empty `NO_COLOR` forces the plain branch even on a TTY. No new `--color`/`--no-color` flag in v1.0.
- **D-06:** Product output for `status`/`files` keeps going to the same stream it uses today (stdout); the pretty branch changes styling only, never which stream product output lands on.

**Progress feedback (TUI-05)**
- **D-07:** Scope to `init` / `index` / `sync`. Spinner for indeterminate work, progress for known-total phases; same `term.IsTerminal` + `NO_COLOR` gate as D-04/D-05. Non-TTY ⇒ current plain behavior.
- **D-08:** Progress writes to stderr, never stdout.
- **D-09:** v1.0 uses a non-interactive progress renderer — lipgloss-styled spinner/progress frames on a ticker, or `charm.land/bubbles` spinner/progress rendered without a bubbletea event loop. NO stdin reading, NO full bubbletea `Program`.

**Archtest / ANSI-isolation guarantee (TUI-01)**
- **D-10:** A new archtest package mirrors `internal/graphstore/archtest/import_graph_test.go` verbatim in mechanism: `packages.Load` over `github.com/seanb4t/codegraph-go/...` with `Tests: true` and `NeedImports|NeedName|NeedDeps`, asserting no forbidden import path appears in the transitive imports of the guarded packages.
- **D-11:** Forbidden import paths: `charm.land/lipgloss`, `charm.land/bubbletea`, `charm.land/bubbles` (charm v2 vanity paths, not `github.com/charmbracelet/...`). Guarded set: `internal/query` + `internal/mcp` (hard requirement); extend to the full serve-reachable set (`internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query`).
- **D-12:** Keep the same self-defeat sanity guard the precedent carries — fail if NO package imports charm at all. Build order: land the archtest FIRST, watch it fail-closed, THEN add `present` + the Charm deps.

**Charm v2 dependency scope**
- **D-13:** Add `charm.land/lipgloss` (v2) now. Add `charm.land/bubbles` only if the progress/spinner path uses it (D-09); otherwise defer. Do NOT add `charm.land/bubbletea` in Phase 6. Pin exact versions in `go.mod`; prefer pure-Go/no-CGo by construction.

### Claude's Discretion
- Exact lipgloss palette / section styling for status & files (borders, key-bolding, language-list grouping) — bounded by the "plain path byte-identical" invariant; reuse Phase-2 section wording and order.
- Spinner glyphs, progress-bar styling, tick cadence.
- Precise archtest package location/name — must mirror the precedent's mechanics regardless of where it lives.

### Deferred Ideas (OUT OF SCOPE)
- `--color` / `--no-color` explicit override flag — v1.0 relies on TTY-detection + `NO_COLOR`.
- Interactive bubbletea components — daemon picker (DMON-01/TUI-04) and `install`/`uninstall` multi-select (TUI-03) → Phase 7.
- Full Charm dependency-closure audit (CGo / govulncheck / SBOM / reproducible double-build, REL-01) → Phase 8.
- Colorizing read commands beyond `status`/`files`.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUI-01 | Import-graph archtest fails the build if `charm.land/lipgloss/v2`/`bubbletea/v2`/`bubbles/v2` are reachable from `internal/query` or `internal/mcp` | Archtest Structure section — exact mirror pattern of `import_graph_test.go`, corrected forbidden-path list (`/v2` suffix), guarded-set from `stdout_confinement_test.go` |
| TUI-02 | `status`/`files` render colorized, sectioned output on a TTY, byte-identical plain output when piped/non-TTY | Standard Stack (lipgloss v2 API), Code Examples, Architecture Patterns (seam), Validation Architecture (byte-identity test strategy without a real pty) |
| TUI-05 | `init`/`index`/`sync` show TTY-gated progress feedback on stderr | Common Pitfalls #1 (bubbles→bubbletea transitive pull) drives the recommendation: hand-rolled lipgloss ticker spinner, NOT `charm.land/bubbles` |
</phase_requirements>

## Summary

Phase 6 is mechanically straightforward — a new `internal/cli/present` package plus one archtest — but two research findings materially change what CONTEXT.md assumed, and the planner must account for both.

**Finding 1 (corrects D-11's literal import paths):** Charm v2's Go module import paths carry a **`/v2` major-version suffix**, per Go's own module-versioning rules. `go list -m -versions` confirms `charm.land/lipgloss` (no suffix) resolves to the **old v0/v1 API line** now mirrored at the new vanity domain — a *different, wrong* module from the one this phase needs. The actual v2 API (the one with the `Renderer` type removed, `colorprofile`-based downsampling, etc.) lives at `charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, and `charm.land/bubbles/v2`. The archtest's forbidden-import list and every `go.mod` require line MUST use the `/v2`-suffixed paths — the bare paths resolve to a real, installable, but semantically wrong module that would silently defeat both the archtest and the API assumptions in CONTEXT.md.

**Finding 2 (resolves D-13's open conditional and reframes D-09/TUI-05):** `charm.land/bubbles/v2`'s own `go.mod` **requires `charm.land/bubbletea/v2` directly** (`require charm.land/bubbletea/v2 v2.0.7`), and `bubbles/spinner`'s source imports `tea "charm.land/bubbletea/v2"` directly — its `Model.Update` signature is `Update(msg tea.Msg) (Model, tea.Cmd)`. There is no "headless" bubbles spinner/progress that avoids the bubbletea package: importing `charm.land/bubbles/v2/spinner` or `.../progress` at all pulls `charm.land/bubbletea/v2` into `go.mod` as a real, compiled-in dependency — even if `tea.NewProgram` is never called. This directly contradicts D-13's "do NOT add bubbletea in Phase 6." **Recommendation: do not add `charm.land/bubbles` in Phase 6.** Build TUI-05's spinner/progress by hand with `lipgloss` styling driven by a stdlib `time.Ticker`, writing frames to stderr with `\r` redraw — the second option D-09 already names ("hand-rolled lipgloss spinner/progress-bar on a ticker"). This keeps `charm.land/bubbletea/v2` entirely out of `go.mod` until Phase 7, as ROADMAP intends.

Everything else confirms CONTEXT.md's decisions directly: `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))` is the exact signature to call (`func IsTerminal(fd int) bool`, no error return) and is already an indirect module (`v0.45.0`) that will simply move to `require` (not `// indirect`) once Phase 6 imports it directly. Lipgloss v2 builds clean under `CGO_ENABLED=0` (verified) and pulls in only pure-Go transitive dependencies (`colorprofile`, `x/ansi`, `go-colorful`, `go-runewidth`, `uniseg`, `uax29`, etc. — no CGo anywhere in the closure). The plain renderers to preserve (`query.RenderStatusText`, `query.RenderFilesMarkdown`/`printFileTree`) and the exact CLI `RunE` seam points (`internal/cli/status.go:61`, `internal/cli/files.go:58-69`) are read and documented below with their real current signatures.

**Primary recommendation:** Land the archtest first with the corrected `/v2`-suffixed forbidden paths and the six-package serve-reachable guarded set (copy `stdout_confinement_test.go`'s `guardedPackages` verbatim). Then add `charm.land/lipgloss/v2` only (no bubbles) to `go.mod`, build `internal/cli/present` as a pure `(Result) → styled string` sibling package, and gate both `status`/`files` and the `init`/`index`/`sync` progress writer through a single **pure, unit-testable branch-selector function** (`isTTY bool, noColor string → bool`) rather than inlining the fd check at each call site — this is what makes TUI-02's "colorized on a TTY" claim testable without a real pty (lipgloss v2's `Style.Render()` always emits full-fidelity ANSI regardless of terminal capability, so pretty-path *content* is verifiable by direct unit test; the *piped/non-TTY byte-identity* claim is verifiable end-to-end today via the existing `runBinary`/`exec.Command` subprocess harness, whose `bytes.Buffer` stdout is a real non-TTY).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| ANSI/color styling (lipgloss) | CLI (`internal/cli/present`) | — | D-01: sole home for charm imports; must never reach the query/MCP tiers |
| TTY + `NO_COLOR` detection | CLI boundary (`RunE`) | — | D-04/D-05: evaluated once, at the real `os.Stdout` stream, not delegated to a library heuristic |
| Plain-text content/wording | Query engine (`internal/query`) | — | D-02: `RenderStatusText`/`RenderFilesMarkdown` already compute all wording; `present` only adds styling around it, never re-derives content |
| Progress feedback (spinner/bar) | CLI (new progress writer, colocated with or beside `present`) | — | D-08: stderr-only, TTY-gated, non-interactive — a CLI-tier concern, never reachable from `serve --mcp` |
| Import-graph enforcement (archtest) | Build/test tier (`internal/*/archtest`) | — | D-10: a `go test` package, not runtime code; enforces the CLI/query-MCP tier boundary at build time |
| MCP output (unaffected) | MCP (`internal/mcp`) | — | Explicitly out of scope this phase; the archtest's entire purpose is to prove this tier can never see charm |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/lipgloss/v2` | v2.0.5 (2026-06-22) `[VERIFIED: go list -m -json, charmbracelet/lipgloss official repo via Origin metadata]` | Terminal styling (borders, bold keys, sectioned layout) for `status`/`files` pretty output and the hand-rolled progress frames | ROADMAP-locked; the Charm ecosystem's de facto standard for Go TUI styling; the exact library `internal/cli/present` is scoped to own |
| `golang.org/x/term` | v0.45.0 (already an indirect dependency of this module today — will become a direct `require` once imported) `[VERIFIED: go.mod, go doc golang.org/x/term]` | `IsTerminal(fd int) bool` — the TTY-detection primitive named in ROADMAP/D-04 | Standard library extension; matches the exact call the ROADMAP notes specify; zero new transitive surface (already resolved in the module graph) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — `charm.land/bubbles/v2` explicitly NOT recommended) | — | — | See Common Pitfalls #1: bubbles/v2 forces `charm.land/bubbletea/v2` into `go.mod`, violating D-13. Do not add it in Phase 6 regardless of how tempting its spinner/progress components look. |
| `github.com/creack/pty` (test-only, OPTIONAL) | v1.1.24 (latest) `[ASSUMED — not in CONTEXT.md's locked dependency scope; flag for planner/discuss-phase confirmation]` | Spawning a subprocess with a real pty attached to stdout, for an end-to-end "renders ANSI when genuinely on a TTY" test | Only if the planner wants a real-terminal integration test beyond the pure-function + content-assertion strategy in Validation Architecture below; POSIX-only (build-tag `!windows`), well-established (MIT, used by Docker/gopls/vscode-go tooling) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled ticker spinner/progress (recommended) | `charm.land/bubbles/v2` spinner/progress | Bubbles' components are polished and battle-tested, but importing them at all pulls `charm.land/bubbletea/v2` into `go.mod` as a real dependency (confirmed via `go get`/`go.mod` inspection) — directly violates D-13's "do NOT add bubbletea in Phase 6." Rejected for this phase; revisit once Phase 7 already depends on bubbletea anyway. |
| `golang.org/x/term.IsTerminal` (locked) | `github.com/mattn/go-isatty` | Already an indirect dependency (pulled in by lipgloss's own closure and other deps) and functionally equivalent, but ROADMAP explicitly locks `x/term` — no reason to diverge; `x/term` is the stdlib-adjacent, lower-dependency-surface choice. |
| `os.Getenv("NO_COLOR") != ""` (recommended, no new dep) | A `NO_COLOR`-aware helper library | The de-facto `NO_COLOR` spec (no-color.org) is exactly "any non-empty value disables color" — a one-line stdlib check. No library needed; adding one would be unjustified surface for a single `os.Getenv` call. |

**Installation:**
```bash
go get charm.land/lipgloss/v2@v2.0.5
go mod tidy   # promotes golang.org/x/term from // indirect to a direct require
```

**Version verification:** `go list -m -versions charm.land/lipgloss/v2` returned versions through `v2.0.5` (latest non-prerelease as of research date); `go list -m -json charm.land/lipgloss/v2@v2.0.5` confirms `Time: 2026-06-22T09:04:51Z` and `Origin.URL: https://github.com/charmbracelet/lipgloss` — the official repo, not a fork. `golang.org/x/term@v0.45.0` is already the exact version pinned in this module's `go.mod` (indirect block).

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam supports `npm|pypi|crates` only — Go is not a supported ecosystem for that automated check. The verification below was performed manually against the Go module proxy/registry and cross-checked against each module's `Origin` metadata (which records the upstream VCS URL and tag), the closest available substitute for the seam's registry-provenance check.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `charm.land/lipgloss/v2` | Go module proxy (proxy.golang.org) | v2.0.0 stable released; v2.0.5 dated 2026-06-22 | Not exposed by `go list`; the underlying `github.com/charmbracelet/lipgloss` repo is one of the most widely used Go TUI styling libraries (tens of thousands of dependents across the ecosystem) | `github.com/charmbracelet/lipgloss` (Origin.URL, confirmed via `go list -m -json`) | OK | Approved |
| `golang.org/x/term` | Go module proxy | Long-lived `golang.org/x/...` extended-stdlib module (100+ releases back to v0.1.0) | N/A — extended stdlib, universally depended-on | `golang.org/x/term` (official Go team) | OK | Approved — already resolved in this module's dependency graph |
| `charm.land/bubbles/v2` | Go module proxy | v2.1.1 dated 2026-07-04 | Widely used Charm ecosystem component library | `github.com/charmbracelet/bubbles` (Origin.URL confirmed) | OK (legitimate package) | **NOT RECOMMENDED for this phase** — not a legitimacy problem, a dependency-graph problem: its own `go.mod` requires `charm.land/bubbletea/v2` directly, contradicting D-13. Do not add. |
| `github.com/creack/pty` | Go module proxy | Long-lived (v1.0.0 through v1.1.24), stable | Standard test-tooling dependency across the Go ecosystem (Docker, gopls, etc.) | `github.com/creack/pty` | OK | Optional, test-only — flagged `[ASSUMED]` because it is not in CONTEXT.md's locked dependency scope; planner should add a `checkpoint:human-verify` before introducing it if chosen |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none. `charm.land/bubbles/v2` is legitimate but architecturally excluded per the finding above — see Common Pitfalls #1.

*`github.com/creack/pty`, if the planner elects to use it, is tagged `[ASSUMED]` and must be gated behind a `checkpoint:human-verify` task before installing, per the package-legitimacy protocol's fallback rule for ecosystems the automated seam does not cover.*

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │         CLI boundary (RunE)              │
                    │  internal/cli/status.go, files.go,       │
                    │  init.go, index.go, sync.go               │
                    └───────────────┬───────────────────────────┘
                                    │
                    (1) term.IsTerminal(os.Stdout.Fd()) +
                        os.Getenv("NO_COLOR")
                                    │
                    ┌───────────────┴───────────────┐
                    │  choosePresentation(isTTY,     │  ← pure, unit-testable
                    │  noColor) bool                 │
                    └───────────────┬───────────────┘
                          false │           │ true
                                │           │
                  ┌─────────────▼─┐   ┌─────▼──────────────────┐
                  │ existing plain │   │ internal/cli/present    │
                  │ writer (D-02,  │   │ (NEW — sole charm.land/  │
                  │ UNCHANGED)     │   │ lipgloss/v2 importer)    │
                  │                │   │                          │
                  │ query.Render-  │   │ present.RenderStatus(    │
                  │ StatusText()   │   │   result, w)             │
                  │ printFileTree()│   │ present.RenderFiles(     │
                  └───────┬────────┘   │   result, w)             │
                          │            └──────────┬───────────────┘
                          │                        │
                          ▼                        ▼
                    cmd.OutOrStdout()  (D-06: same stream, styling only)
                          │
                          ▼
                    Terminal (TTY) or pipe (byte-identical to plain path
                    when the TTY branch was never taken)

                    ────────────────────────────────────────────
                    Progress (init/index/sync), SEPARATE seam:

                    RunE ──(2) same isTTY+NO_COLOR gate──▶ progress writer
                                                             │
                                             non-TTY: no frames, silent
                                             TTY: lipgloss-styled spinner/
                                             bar on a time.Ticker, written
                                             to os.Stderr with \r redraw
                                             (NEVER charm.land/bubbles —
                                             see Common Pitfalls #1)

                    ────────────────────────────────────────────
                    BUILD-TIME (not runtime): TUI-01 archtest

                    go test ./internal/cli/present/archtest/...
                      packages.Load(Tests:true, .../...)
                      for each of {internal/mcp, internal/graphstore,
                        internal/daemon, internal/watch, internal/indexer,
                        internal/query}:
                        assert charm.land/lipgloss/v2,
                               charm.land/bubbletea/v2,
                               charm.land/bubbles/v2
                        NOT in transitive Imports
                      + self-defeat guard: fail if NOTHING in the whole
                        module imports charm.land/lipgloss/v2
```

### Recommended Project Structure
```
internal/cli/
├── present/                  # NEW — sole home for charm.land/lipgloss/v2
│   ├── status.go             # present.RenderStatus(query.StatusResult, io.Writer)
│   ├── files.go               # present.RenderFiles(query.FilesResult, io.Writer)
│   ├── progress.go            # hand-rolled ticker spinner/progress bar (stderr)
│   ├── tty.go                 # choosePresentation(isTTY bool, noColor string) bool — pure, unit-tested
│   └── archtest/
│       └── import_graph_test.go   # TUI-01: mirrors internal/graphstore/archtest's mechanism
├── status.go                  # RunE: wires isTTY/NO_COLOR → present.RenderStatus vs query.RenderStatusText
├── files.go                    # RunE: same gate → present.RenderFiles vs existing plain branch
├── init.go / index.go / sync.go  # RunE: same gate → present progress writer vs current silent/plain summary
```
(Package name/location is Claude's discretion per CONTEXT — `internal/cli/present` is the CONTEXT-suggested and ROADMAP-named location; the archtest subpackage under it is one reasonable placement mirroring `internal/graphstore/archtest`'s own pattern of living inside the guarded package's tree. An `internal/archtest` top-level location is an equally valid alternative — CONTEXT leaves this open.)

### Pattern 1: Pure branch-selector, real check only at the outermost RunE
**What:** A tiny pure function, `choosePresentation(isTTY bool, noColor string) bool`, holds the D-04/D-05 decision logic (`isTTY && noColor == ""`). `RunE` calls it with the real values (`term.IsTerminal(int(os.Stdout.Fd()))`, `os.Getenv("NO_COLOR")`) and nothing else in the codebase re-implements the check.
**When to use:** Every one of the five call sites this phase touches (`status`, `files`, `init`, `index`, `sync`) — one implementation, tested once in isolation, wired identically everywhere (mirrors this codebase's existing `sortedCounts`/`writeStatusAdvisories` "one shared helper, many callers" convention in `render_status.go`).
**Example:**
```go
// internal/cli/present/tty.go
package present

// ChoosePresentation reports whether the pretty (lipgloss) branch should
// render, per D-04/D-05: isTTY must be true AND NO_COLOR must be unset/empty.
// Pure and side-effect-free so it is testable without a real terminal or fd.
func ChoosePresentation(isTTY bool, noColor string) bool {
	return isTTY && noColor == ""
}
```
```go
// internal/cli/status.go RunE (excerpt) — the only place the real fd is read
isTTY := term.IsTerminal(int(os.Stdout.Fd()))
if present.ChoosePresentation(isTTY, os.Getenv("NO_COLOR")) {
	return present.RenderStatus(result, start, cmd.OutOrStdout())
}
fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
```

### Pattern 2: lipgloss v2 has no global `Renderer` — style, then print through an explicit downsampling writer
**What:** Lipgloss v2 removed `Renderer`/`DefaultRenderer()`/`NewRenderer()` entirely (confirmed via the official v2 upgrade guide). `Style.Render()` now **always emits full-fidelity (truecolor) ANSI**, unconditionally — there is no implicit "auto-detect and degrade" at render time. Downsampling to the terminal's actual capability happens only when *printing*, via `lipgloss.Fprintln`/`lipgloss.Sprint`/`colorprofile`-wrapped writers.
**When to use:** Because D-04 already mandates bypassing lipgloss entirely on non-TTY (the plain branch never touches lipgloss at all), `present`'s pretty path can simply `Style.Render()` and write the resulting string directly to `cmd.OutOrStdout()` with `fmt.Fprint` — no `colorprofile` wrapping is required for correctness (the plain branch already guarantees "no ANSI when piped"; a real TTY session doesn't need downsampling for most modern terminals). If the planner wants finer color-profile fidelity on legacy/16-color terminals, wrap the writer with `colorprofile.NewWriter(w, os.Environ())` before writing — optional polish, not required by any locked decision.
**Example:**
```go
// Source: https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md
import lipgloss "charm.land/lipgloss/v2"

headerStyle := lipgloss.NewStyle().Bold(true)
s := headerStyle.Render("CodeGraph Status")
fmt.Fprintln(w, s) // w is already known-TTY at this call site (D-04's gate) — no downsampling needed
```

### Pattern 3: Hand-rolled ticker progress (TUI-05) — no bubbletea, no bubbles
**What:** A `time.Ticker`-driven loop that renders one lipgloss-styled frame per tick to a `\r`-prefixed line on stderr, started before a long-running phase (`indexer.Run`/`indexer.Sync`) and stopped (with a final clear or completion line) when it returns.
**When to use:** Every TUI-05 call site (`init`, `index`, `sync`) — non-interactive, TTY-gated (same `ChoosePresentation` from Pattern 1, but checking `os.Stderr`'s fd since progress writes to stderr per D-08), reads no stdin, never blocks.
**Example:**
```go
// present/progress.go — illustrative sketch, not verbatim upstream source
// (no CGo, no bubbletea import; pure lipgloss + stdlib time.Ticker)
type Spinner struct {
	frames []string
	style  lipgloss.Style
	w      io.Writer
	stop   chan struct{}
}

func NewSpinner(w io.Writer) *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		style:  lipgloss.NewStyle().Foreground(lipgloss.Color("212")),
		w:      w,
		stop:   make(chan struct{}),
	}
}

func (s *Spinner) Start(label string) {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Fprint(s.w, "\r\033[K") // clear line on stop
				return
			case <-ticker.C:
				frame := s.style.Render(s.frames[i%len(s.frames)])
				fmt.Fprintf(s.w, "\r%s %s", frame, label)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop() { close(s.stop) }
```

### Anti-Patterns to Avoid
- **Adding `charm.land/bubbles/v2` "just for the spinner":** confirmed (Common Pitfalls #1) to force `charm.land/bubbletea/v2` into `go.mod` — directly violates D-13. Use Pattern 3 instead.
- **Referencing bare `charm.land/lipgloss` (no `/v2`) anywhere — go.mod, archtest forbidden-list, or import:** resolves to a real but *different* module (the old v1 API line, now also hosted at the vanity domain). Every reference in this phase — `go.mod` require, `import`, and the archtest's forbidden-path constant — must use the `/v2`-suffixed path.
- **Checking `NO_COLOR` or TTY-ness inside `internal/cli/present` itself:** keep `present` a pure renderer package (`Result → styled string`) with the TTY/NO_COLOR *decision* made once at the `RunE` boundary (Pattern 1) — this is what D-03 means by "branch selection happens once, at the CLI boundary," and it is also what keeps `present`'s renderers trivially unit-testable without faking a terminal.
- **Re-deriving status/files content inside `present`:** D-02 requires `present` to consume `query.StatusResult`/`query.FilesResult` as-is and only add styling — never recompute counts, sort order, or wording that the plain renderer already owns.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ANSI color/style escape sequences | A hand-written `\033[...]m` string builder | `charm.land/lipgloss/v2` `Style.Render()` | Correct SGR sequence composition, width-aware truncation/padding, and border rendering are exactly what lipgloss exists to get right; hand-rolling reintroduces the class of off-by-one/escape-sequence bugs this dependency is meant to eliminate |
| TTY / pty detection | Reading `/dev/tty`, parsing `TERM`, or a custom `isatty` syscall wrapper | `golang.org/x/term.IsTerminal(int(fd))` | Cross-platform (POSIX ioctl + Windows console API) correctness is already solved and locked by ROADMAP; no reason to duplicate it |
| Import-graph boundary enforcement | A regex/grep-based "no `charm.land` string in `internal/query/*.go`" check | `golang.org/x/tools/go/packages`-based archtest (mirroring the two existing precedents) | Regex/string-matching misses aliased imports, build-tag-gated files, and test-file-only imports — exactly the gap the existing `import_graph_test.go` doc comment calls out; this is not a new argument, it is the same one already proven in this codebase twice |

**Key insight:** Every "don't hand-roll" item above already has a proven, in-repo precedent (the two existing archtests) or a locked upstream dependency (lipgloss, x/term) — this phase's job is disciplined reuse of established patterns, not new design.

## Common Pitfalls

### Pitfall 1: `charm.land/bubbles/v2` transitively forces in `charm.land/bubbletea/v2`
**What goes wrong:** A planner reads D-09 ("bubbles spinner/progress rendered without a bubbletea event loop") and assumes bubbles can be added standalone. `go get charm.land/bubbles/v2` (verified in this research session) adds `charm.land/bubbletea/v2` as a **direct require in bubbles' own go.mod**, and `bubbles/spinner`'s Go source directly imports `tea "charm.land/bubbletea/v2"` for its `Model.Update(msg tea.Msg) (Model, tea.Cmd)` signature. The bubbletea package is compiled into the binary the moment `bubbles/spinner` or `bubbles/progress` is imported, regardless of whether `tea.NewProgram(...).Run()` is ever called.
**Why it happens:** Bubbles is designed as a component library FOR bubbletea programs — its `Update`/`Init`/`View` methods implement the `tea.Model` interface contract, which requires bubbletea's `Msg`/`Cmd` types even in isolation.
**How to avoid:** Do not add `charm.land/bubbles` to `go.mod` in this phase. Build the spinner/progress bar by hand (Pattern 3): `lipgloss.Style.Render()` for the visual frame, `time.Ticker` for the animation clock, plain `fmt.Fprintf(stderr, "\r%s", frame)` for the redraw. This is a small amount of code (bubbles' own spinner implementation is ~150 lines) and keeps `charm.land/bubbletea/v2` completely absent from `go.mod` until Phase 7, matching ROADMAP's stated intent ("the daemon picker... is Phase 7's concern").
**Warning signs:** `go.mod` gaining a `charm.land/bubbletea/v2` line (even `// indirect`) during Phase 6 is the signal this pitfall was hit — grep `go.mod` for `bubbletea` before considering the phase done.

### Pitfall 2: bare `charm.land/lipgloss` (no `/v2`) is a real, different, WRONG module
**What goes wrong:** Because Go's module system requires a `/v2` path suffix for major version 2+, and Charm re-hosted its *entire* version history (v0.x through v1.x) at the new `charm.land/lipgloss` vanity path (not just v2), `charm.land/lipgloss` without the suffix resolves successfully — to the OLD API (the one WITH `Renderer`/`NewRenderer()`, not the v2 shape CONTEXT.md's research questions assumed). A typo this small (`go get charm.land/lipgloss` instead of `go get charm.land/lipgloss/v2`) produces a build that compiles, imports successfully, and silently uses an entirely different (and, per ROADMAP, not the intended) API surface.
**Why it happens:** CONTEXT.md's canonical refs describe the import as "`charm.land/lipgloss` (NOT `github.com/charmbracelet/...`)" without the `/v2` suffix — accurate about the vanity-domain part, silent on the version-suffix part, because that distinction only surfaces once you actually resolve both paths against the module proxy (as this research did).
**How to avoid:** Every reference — `go.mod` require line, Go `import` statement, and the archtest's forbidden-import-path constants — must read `charm.land/lipgloss/v2` (and `charm.land/bubbletea/v2`, `charm.land/bubbles/v2` if ever added). Grep the finished `go.mod` for a bare (non-`/v2`) `charm.land/` line as a completion check — there should be none.
**Warning signs:** Any compile error referencing `lipgloss.Renderer`, `lipgloss.NewRenderer`, or `lipgloss.DefaultRenderer` — these symbols do not exist in v2 and their presence in a diff means the wrong module version landed.

### Pitfall 3: in-process CLI tests can never exercise the pretty branch as written
**What goes wrong:** This codebase's existing CLI test harness (`internal/cli/cli_test.go`'s `execCmd`) calls `root.SetOut(&outBuf)` and executes the cobra command tree in-process. `cmd.OutOrStdout()` returns the buffer, but a naive TTY check hardcoded to `os.Stdout.Fd()` inside `RunE` still reads the REAL process's stdout — which under `go test` is virtually never a TTY. A test asserting "the pretty branch renders ANSI" by calling `execCmd("status", ...)` will always take the plain branch and the assertion will never be exercised, silently.
**Why it happens:** `cmd.SetOut()` only redirects where cobra's helpers *write*; it does not redirect what `os.Stdout.Fd()` *is*. The two are decoupled by design (D-06 requires product output and the TTY check to both reference the real stdout stream, which is exactly what makes this decoupling happen in tests).
**How to avoid:** Structure the TTY decision as the pure `ChoosePresentation(isTTY bool, noColor string) bool` function (Pattern 1) and unit-test it directly with boolean inputs — no fd, no pty, no `execCmd` needed. Separately, unit-test `present.RenderStatus`/`present.RenderFiles` by calling them directly (they take no TTY dependency of their own) and asserting the output contains ANSI escape bytes (`\x1b[`) plus the same section keywords the plain renderer uses. Together these two test tiers cover both halves of TUI-02 without ever needing a real terminal. See Validation Architecture below.
**Warning signs:** A "pretty output" test that calls `execCmd(...)` and asserts on ANSI content will pass or fail based on the `go test` runner's stdout, not the code under test — a classic flaky/vacuous test. If such a test exists and is green, verify it is actually exercising the pretty branch (it likely is not).

### Pitfall 4: forgetting the self-defeat guard makes the archtest vacuously green after any refactor
**What goes wrong:** Both existing precedent archtests (`import_graph_test.go`, `modernc_confinement_test.go`) carry an explicit "did this test find ANY importer at all" sanity check — without it, if a future refactor removes the only charm import from `internal/cli/present` (e.g., someone deletes the pretty renderer entirely), the archtest's core assertion ("no forbidden package imports X") becomes vacuously true for the wrong reason, silently losing all coverage.
**Why it happens:** An import-graph absence check is easy to write correctly for the "found a violation" case and easy to forget for the "found nothing to check against" case.
**How to avoid:** D-12 already mandates this — copy the precedent's `if !foundGraphstoreImporter { t.Fatal(...) }` pattern verbatim, renamed for `charm.land/lipgloss/v2` and `internal/cli/present`.
**Warning signs:** An archtest that has been green since Phase 6 landed and has never once gone red during any subsequent phase's development is worth a quick manual check that the self-defeat guard still fires if the charm import is temporarily commented out.

## Code Examples

### The exact current plain renderer being preserved (D-02 — DO NOT MODIFY)
```go
// Source: internal/query/render_status.go:170-192 (this repo, current state)
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
`present.RenderStatus` should mirror this exact section order/wording (D-01 discretion note: "reuse Phase-2 section wording and order") while adding lipgloss styling — a `- header, - stat line, - breakdown, - advisory` structural walk over the *same* `StatusResult`, not a re-derivation.

### The exact current CLI seam point to modify (D-03)
```go
// Source: internal/cli/status.go:56-63 (this repo, current state — the exact
// line the RunE change lands on)
// RenderStatusText already embeds the verbose worktree warning
// (from result.WorktreeMismatch, live since plan 02-04) at D-09's
// structural position — no separate warning print here, which
// would double it.
fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
return nil
```
```go
// Source: internal/cli/files.go:57-70 (this repo, current state — the exact
// lines the RunE change lands on)
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
Both seams are a single `if present.ChoosePresentation(...) { return present.Render*(...) }` inserted immediately before the existing plain-rendering lines — no other line in either `RunE` changes (D-06: same stream, styling only).

### `StatusResult`/`FilesResult` — the exact structs `present` consumes (read-only)
```go
// Source: internal/query/status.go:46-63 (this repo, current state)
type StatusResult struct {
	Initialized      bool
	Version          string
	ProjectPath      string
	IndexPath        string
	FileCount        int64
	NodeCount        int64
	EdgeCount        int64
	DbSizeBytes      int64
	Backend          string
	NodesByKind      map[string]int64
	FilesByLanguage  map[string]int64 // json:"-" — Go-internal only, exists for renderers
	Languages        []string
	PendingChanges   PendingChanges
	WorktreeMismatch *gitmeta.Mismatch
	Stale            bool
	Index            IndexHealth
}
```
```go
// Source: internal/query/files.go:47-71 (this repo, current state)
type FileEntry struct {
	Path      string
	Language  string
	NodeCount int64
	EdgeCount int64
}
type FileTreeNode struct {
	Name     string
	IsDir    bool
	Path     string          // leaf only
	Language string          // leaf only
	Children []*FileTreeNode // dir only
}
type FilesResult struct {
	Format string          // "flat" or "tree"
	Files  []FileEntry     // populated when Format == "flat"/""
	Tree   []*FileTreeNode // populated when Format == "tree"
}
```
`sortedCounts`, `formatNumber`, and `formatMB` (all unexported in `internal/query`) are NOT importable by `internal/cli/present` — either duplicate the small formatting logic package-locally (this codebase's own established precedent, e.g. `status.go`'s `shouldSkipStaleDir` and `render_results.go`'s `renderFileTreeMarkdown` both duplicate rather than import to avoid an unwanted dependency edge) or export them from `internal/query` if the planner prefers a shared helper. Either is consistent with existing codebase convention; CONTEXT does not lock this choice.

### The exact archtest mechanism to mirror
```go
// Source: internal/graphstore/archtest/import_graph_test.go (this repo,
// current state) — the load config and self-defeat guard to copy verbatim,
// substituting the forbidden path(s) and allowed-importer set.
cfg := &packages.Config{
	Mode:  packages.NeedImports | packages.NeedName | packages.NeedDeps,
	Tests: true, // catches a bypass hidden in a _test.go file too
}
pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
// ... for each pkg, check pkg.Imports[forbiddenPath] ...
// ... self-defeat: t.Fatal if NO package was found importing the forbidden path ...
```
TUI-01's archtest is structurally an INVERSE of this precedent (precedent: "only package X may import Y"; TUI-01: "packages in guarded-set G may NOT import Y1/Y2/Y3"), closer in shape to `stdout_confinement_test.go`'s `guardedPackages` + closure-walk pattern than to `import_graph_test.go`'s single-allowed-importer pattern. Recommend basing TUI-01 on `stdout_confinement_test.go`'s `closeOverServeReachableImports` helper (load the six guarded packages with `NeedDeps`, walk the transitive module-internal closure) rather than `import_graph_test.go`'s whole-module scan — the guarded set is fixed and named (D-11), not "everything except one prefix," which is exactly what `stdout_confinement_test.go` was built for. The self-defeat guard (D-12) still applies: assert `charm.land/lipgloss/v2` (or whichever forbidden import) appears SOMEWHERE in the whole-module scan (mirroring `import_graph_test.go`'s pattern for that specific check), proving the archtest isn't vacuously blind.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Lipgloss v1: package-level `Renderer`, `lipgloss.NewRenderer(w)`, implicit color-profile auto-detection at render time | Lipgloss v2: no `Renderer` type; `Style.Render()` always emits full-fidelity ANSI; downsampling is an explicit, separate step at print time (`lipgloss.Println`/`Fprintln`/`colorprofile.NewWriter`) | v2.0.0 (2026, per the module's alpha→beta→stable version history observed via `go list -m -versions`) | Any code (or documentation, including parts of CONTEXT.md's research-question phrasing) written against v1's "construct a renderer against a writer" mental model does not translate directly — there is no renderer object to construct. Style first, decide how/whether to downsample separately. |
| `github.com/charmbracelet/lipgloss` (GitHub-hosted import path) | `charm.land/lipgloss/v2` (vanity import path, /v2 suffixed) | Charm's ecosystem-wide migration to the `charm.land` vanity domain, coincident with the v2 major-version bump for lipgloss/bubbletea/bubbles | ROADMAP already locks this; this research adds the missing `/v2` suffix detail |

**Deprecated/outdated:**
- `lipgloss.NewRenderer()` / `lipgloss.DefaultRenderer()` / the `Renderer` type: removed in v2. Do not port any v1-era code pattern that constructs one.
- `lipgloss.HasDarkBackground()` (no-arg v1 form): v2 requires explicit `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)` — not needed by this phase's scope (status/files styling doesn't require light/dark adaptive colors), noted for completeness only.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `github.com/creack/pty` is a reasonable optional test-only dependency for a real-TTY end-to-end test | Standard Stack / Package Legitimacy Audit | Low — it is explicitly optional; the primary Validation Architecture strategy (pure-function + content-assertion tests) does not depend on it at all. If rejected, simply omit the optional pty-based test tier. |
| A2 | `internal/cli/present`'s exact package/file layout (status.go/files.go/progress.go/tty.go split) | Architecture Patterns / Recommended Project Structure | Low — CONTEXT explicitly leaves package internal layout to Claude's discretion; any equivalent split satisfies D-01/D-03 |
| A3 | The archtest should be structured like `stdout_confinement_test.go` (named guarded-set + closure walk) rather than `import_graph_test.go` (whole-module scan, single allowed-importer) | Code Examples | Low-medium — both approaches can satisfy TUI-01's letter; the recommendation is a structural judgment call from reading both precedents, not a locked decision. If the planner prefers the simpler whole-module-scan shape, D-10/D-12's self-defeat-guard requirement and D-11's forbidden-path list still apply identically. |

**All version numbers, module paths, CGo-safety, and the bubbles→bubbletea transitive-dependency finding are `[VERIFIED]`** via direct `go list -m`/`go get`/`go build -o /dev/null` commands run in this research session against the live Go module proxy — not training-data recall. Given Charm's v2 migration is very recent relative to most training data, this verification was treated as mandatory rather than optional for this phase.

## Open Questions

1. **Should `internal/cli/present` export the small formatting helpers (`formatNumber`, `formatMB`, `sortedCounts`) from `internal/query`, or duplicate them package-locally?**
   - What we know: both patterns already coexist in this codebase (`status.go`'s `shouldSkipStaleDir` duplicates rather than imports across the query/indexer boundary; `render_results.go`'s `renderFileTreeMarkdown` duplicates `internal/cli/files.go`'s `printFileTree` rather than importing across the query/cli boundary in the OTHER direction).
   - What's unclear: whether exporting three small pure functions from `internal/query` (a one-line visibility change, zero behavior change) is preferable to a third package-local duplication, purely on maintenance-burden grounds.
   - Recommendation: duplicate (matches the stronger, more recent precedent — `render_results.go`'s `renderFileTreeMarkdown` doc comment explicitly justifies package-local duplication specifically to avoid a cross-tier dependency edge, which is the same shape of decision here: `internal/cli/present` reaching into `internal/query`'s unexported helpers would be a new, narrow coupling for three ~10-line functions).

2. **Does the progress writer for `init`/`index`/`sync` live inside `internal/cli/present` or as a separate `internal/cli` (non-`present`) helper?**
   - What we know: D-08 requires stderr-only, and the archtest's guarded set (D-11) is scoped to `internal/query`/`internal/mcp`/etc — `internal/cli` in its entirety is already the existing HYG-02 archtest's documented exclusion (legitimately renders product output, never reachable from `serve --mcp`), so there is no build-enforcement reason the progress writer must live inside `present` specifically rather than elsewhere in `internal/cli`.
   - What's unclear: whether keeping ALL lipgloss usage inside literally one package (`present`) for auditability outweighs the minor organizational awkwardness of a progress-writer file sitting alongside status/files renderers it has little else in common with.
   - Recommendation: keep it inside `present` anyway — D-01 already frames `present` as "the sole home for `charm.land/lipgloss` styling," and a single-package charm surface is the simplest thing to point the TUI-01 archtest's self-defeat guard at.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | building/testing this phase's code | ✓ | go.mod declares `go 1.26.5`; lipgloss v2.0.5 and bubbles v2.1.1 both require `go 1.25.0` minimum — satisfied | — |
| Go module proxy network access | resolving `charm.land/lipgloss/v2` and its transitive closure | ✓ (verified live in this research session via `go get`/`go list -m`) | — | If CI/build environments lack proxy access, `GOFLAGS=-mod=mod` + a vendored `go.sum` (already this repo's existing pattern for all other deps) is the standard fallback — no new tooling needed |
| `charm.land` vanity-domain redirect availability | resolving the module path at all (vanity import paths depend on the domain's go-import meta tags staying live) | ✓ (resolved successfully in this session) | — | If `charm.land` ever becomes unreachable, `GOPROXY`'s cached copies (proxy.golang.org retains fetched modules indefinitely per Go's module mirror design) continue to serve existing `go.sum`-pinned versions even if the vanity domain later disappears |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** none currently missing — table above documents fallback paths as a precaution, not an active gap.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (this repo's sole framework — no external test framework in `go.mod`) |
| Config file | none — `go test ./...` per `.github/workflows/ci.yml` |
| Quick run command | `go test ./internal/cli/... ./internal/cli/present/...` |
| Full suite command | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` (mirrors `ci.yml`'s three-tier split) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUI-01 | No `charm.land/{lipgloss,bubbletea,bubbles}/v2` reachable from the six guarded packages | archtest (build-time) | `go test ./internal/cli/present/archtest/... -run TestNoCharmInServeReachablePackages` | ❌ Wave 0 (new file, must land FIRST per D-12's build order) |
| TUI-01 | Self-defeat guard: archtest cannot be vacuously green | archtest (same file) | included in the command above | ❌ Wave 0 (same file) |
| TUI-02 | Branch selection is correct for all four (isTTY × NO_COLOR) combinations | unit | `go test ./internal/cli/present/... -run TestChoosePresentation` | ❌ Wave 0 |
| TUI-02 | Pretty renderer contains ANSI + preserves plain-renderer section wording/order | unit | `go test ./internal/cli/present/... -run TestRenderStatus` / `TestRenderFiles` | ❌ Wave 0 |
| TUI-02 | Piped/non-TTY `status`/`files` output is byte-identical to the pre-Phase-6 plain output | integration (real subprocess, `bytes.Buffer` stdout — already non-TTY) | `go test ./test/integration/... -run TestStatusFilesPlainByteIdentity` | ❌ Wave 0 |
| TUI-02 / golden | The golden/MCP output path is untouched | existing | `go test ./testdata/golden/...` | ✅ (pre-existing — must stay green, unmodified) |
| TUI-05 | Progress writes only to stderr, never stdout, and only when TTY-gated | unit + reachability | `go test ./internal/cli/present/... -run TestProgress` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/cli/... ./internal/cli/present/...`
- **Per wave merge:** `go test ./... && go test ./testdata/golden/...` (skip `./test/integration/...` per-wave if slow; run at phase gate)
- **Phase gate:** Full suite green (all three tiers) before `/gsd-verify-work`, plus a manual `grep -n "charm.land/bubbletea" go.mod` returning nothing, and a manual `go build` + real-terminal smoke test of `codegraph status`/`codegraph files`/`codegraph index` in an actual terminal (the one behavior class the automated suite deliberately does not cover — see Pitfall 3).

### Wave 0 Gaps
- [ ] `internal/cli/present/archtest/import_graph_test.go` — covers TUI-01 (land FIRST, before any charm dependency enters `go.mod`, per D-12's build order — this file should fail to even compile/resolve meaningfully until `present` exists, which is itself the "fails the build" proof point)
- [ ] `internal/cli/present/tty_test.go` — covers the `ChoosePresentation` pure-function matrix (TUI-02)
- [ ] `internal/cli/present/status_test.go` / `files_test.go` — covers pretty-renderer content assertions (TUI-02)
- [ ] `test/integration/status_files_plain_test.go` (or added test functions in an existing integration file) — covers the byte-identity claim end-to-end via `runBinary`'s existing `bytes.Buffer`-stdout pattern (TUI-02 point 4)
- [ ] `internal/cli/present/progress_test.go` — covers TUI-05's stderr-only + TTY-gate behavior
- [ ] No new test framework install needed — stdlib `testing` + this repo's existing `execCmd`/`runBinary` harnesses cover every tier above without a real pty (see Common Pitfalls #3 for why a pty is deliberately NOT required for Wave 0 coverage)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A — local CLI rendering, no auth surface touched |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes (narrow) | `NO_COLOR` env var and TTY-fd values are read via stdlib (`os.Getenv`, `x/term.IsTerminal`) — no parsing of untrusted structured input introduced by this phase |
| V6 Cryptography | no | N/A — no crypto surface touched |
| V7 Error Handling / Logging | yes (narrow) | Progress/spinner writes must go to stderr only (D-08) and must never write to stdout even on error paths — reuses the exact HYG-02 stdout-confinement discipline (`internal/cli` is already excluded from that guard since it's the legitimate product-output tier, but the NEW progress writer must still honor D-08's "stderr, never stdout" as its own local invariant, verified by the Wave-0 `progress_test.go`) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Charm/ANSI styling reaching the agent/MCP surface (the entire reason this phase's archtest exists) | Tampering / Information Disclosure (a model could receive garbled or misleading output if raw ANSI escape bytes leaked into its context) | The TUI-01 archtest itself is the mitigation — build-time, not runtime, enforcement that `internal/mcp`/`internal/query` can never import charm |
| Terminal escape-sequence injection via untrusted repo content (a maliciously crafted file path or symbol name containing raw ANSI/control-character bytes, rendered into a human's real terminal by the NEW pretty path) | Tampering | **Not mitigated by this phase's design as currently scoped** — `lipgloss.Style.Render()` styles the string it's given but does not sanitize/strip pre-existing control characters embedded within it, and `query.StatusResult`/`FilesResult` data (file paths, language names) ultimately derives from indexing an arbitrary (potentially untrusted) repository. This is a pre-existing risk class for ANY terminal tool that echoes repo-derived strings (the plain-text path already has this same exposure today, unchanged by this phase) — Phase 6 does not make it worse, but it also does not close it. Flag for the planner: either explicitly accept this as an out-of-scope, pre-existing risk (consistent with the plain renderers' current behavior, and consistent with virtually every git/file-browsing CLI tool's status quo), or add a control-character-stripping pass before rendering file/symbol names as a defense-in-depth improvement. CONTEXT.md does not mention this threat, so treat any mitigation here as new scope requiring a discuss-phase confirmation rather than an implicit requirement. |
| Progress/spinner writer accidentally corrupting the MCP JSON-RPC stdout stream | Tampering | Reuses D-08 (stderr-only) + the existing HYG-02 archtest's proven pattern; the NEW progress writer's own unit test (`progress_test.go`) must assert zero stdout writes, mirroring `stdout_detection_selftest_test.go`'s self-test-can-fail discipline |

---

## Sources

### Primary (HIGH confidence)
- `charm.land/lipgloss/v2` (Context7 `/charmbracelet/lipgloss`, cross-checked live via `go list -m -json`/`go get`/`go build -o /dev/null` in this session) — v2 API shape (no `Renderer` type, always-full-fidelity `Render()`, `colorprofile`-based downsampling), module path, version, CGo-safety
- `charm.land/bubbles/v2` (Context7 `/charmbracelet/bubbles`, cross-checked live via `go get`/inspecting its `go.mod` and `spinner.go` source in the local module cache in this session) — the bubbletea transitive-dependency finding (Common Pitfalls #1) is a direct, first-hand read of the downloaded module's own `go.mod` and source, not a documentation claim
- `golang.org/x/term` (`go doc golang.org/x/term`, run live in this session) — exact `IsTerminal(fd int) bool` signature
- This repo's own source, read directly in this session: `internal/graphstore/archtest/import_graph_test.go`, `internal/graphstore/archtest/stdout_confinement_test.go`, `internal/migrate/archtest/modernc_confinement_test.go`, `internal/query/render_status.go`, `internal/query/status.go`, `internal/query/files.go`, `internal/query/render_results.go`, `internal/query/render_markdown.go`, `internal/cli/status.go`, `internal/cli/files.go`, `internal/cli/init.go`, `internal/cli/index.go`, `internal/cli/sync.go`, `internal/cli/query.go`, `internal/cli/cli_test.go`, `internal/cli/status_cli_test.go`, `test/integration/main_test.go`, `testdata/golden/golden_test.go`, `go.mod`, `.github/workflows/ci.yml`, `.planning/config.json`

### Secondary (MEDIUM confidence)
- WebSearch cross-check confirming `charm.land/lipgloss/v2` as lipgloss's v2 module path (consistent with the direct `go list -m -versions` verification, corroborating rather than sole source)

### Tertiary (LOW confidence)
- none — every claim in this document that could be verified against a live tool call or this repo's own source was verified this session; no unverified WebSearch-only or training-data-only claims were used for package names or API shapes

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version/module-path/CGo claim verified live against the Go module proxy in this session, not recalled from training data
- Architecture: HIGH — every seam point (RunE bodies, StatusResult/FilesResult shapes, archtest mechanics) is a direct read of this repo's current source, not inference
- Pitfalls: HIGH — the bubbles→bubbletea transitive-dependency finding and the `/v2`-suffix finding are both first-hand, tool-verified discoveries in this session (not carried over from CONTEXT.md's assumptions, which is precisely why they needed correcting)

**Research date:** 2026-07-17
**Valid until:** ~30 days (stable, released Charm v2 APIs; re-verify module versions if planning is delayed past early September 2026, since Charm ships frequent patch releases)
