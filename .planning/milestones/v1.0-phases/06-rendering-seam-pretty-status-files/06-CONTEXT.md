# Phase 6: Rendering Seam & Pretty status/files - Context

**Gathered:** 2026-07-17
**Status:** Ready for planning
**Mode:** --auto --all --chain (all gray areas auto-selected; recommended options chosen and logged in 06-DISCUSSION-LOG.md; auto-advancing to plan → execute)

<domain>
## Phase Boundary

A **build-enforced rendering seam** permanently isolates all Charm/ANSI styling
from the agent/MCP path, and the `status` / `files` commands render colorized,
sectioned output on a TTY while staying **byte-identical plain** when piped or
non-TTY. Concretely:

- **TUI-01** — an import-graph archtest fails the build if
  `charm.land/lipgloss` / `bubbletea` / `bubbles` are reachable from
  `internal/query` or `internal/mcp` (mirrors the existing graphstore/migrate
  archtests).
- **TUI-02** — `status` + `files` render lipgloss-styled sectioned output on a
  TTY, byte-identical plain output otherwise.
- **TUI-05** — `init` / `index` / `sync` show TTY-gated progress feedback
  (spinner/progress), plain otherwise.

**Not in this phase:** interactive bubbletea components — the daemon picker
(DMON-01/TUI-04) and `install`/`uninstall` multi-select (TUI-03) are **Phase 7**;
the full Charm dependency-closure audit (CGo/govulncheck/SBOM/reproducible
build, REL-01) is **Phase 8**; colorizing read commands beyond `status`/`files`;
any change to WHAT commands compute or to the golden/MCP output bytes.

</domain>

<decisions>
## Implementation Decisions

### Rendering seam & package placement (TUI-02)
- **D-01:** A new `internal/cli/present` package is the **sole home** for
  `charm.land/lipgloss` styling. It consumes already-computed plain data
  (`query.StatusResult`, the `files` result struct) and emits colorized,
  sectioned output. `internal/query` and `internal/mcp` never import charm —
  this is exactly the boundary the TUI-01 archtest enforces.
  `[auto] Seam architecture → Selected: dedicated internal/cli/present package consuming plain result structs (recommended)`
- **D-02:** The plain path stays byte-identical by **reuse, not re-derivation**.
  The existing `query.RenderStatusText` (`render_status.go`) and the `files`
  plain/markdown renderers (`render_results.go` / `render_markdown.go` /
  `files.go`) are **UNCHANGED** and remain the verbatim piped/non-TTY output.
  The `present` pretty renderer is an **additive sibling** — it must not alter a
  single byte of the plain path. This is what guarantees TUI-02 byte-identity
  and keeps the golden corpus / MCP surface unchanged.
- **D-03:** Branch selection happens **once, at the CLI boundary**
  (`internal/cli/status.go`, `internal/cli/files.go` `RunE`):
  `isTTY && colorEnabled` → `present.Render*(result, w)`; else → the existing
  plain writer. `internal/cli/status.go:18-20` already documents this exact
  handoff ("Phase 6 (TUI-02) will colorize this layout with lipgloss/TTY-gating;
  this plan lays down content, sections, and wording only — plain text").

### TTY detection & color policy (TUI-02)
- **D-04:** TTY detection via `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))`
  (ROADMAP-locked), evaluated at the CLI boundary against the real output
  stream. Do **not** rely on lipgloss's own color-profile auto-degrade for the
  piped guarantee — when not a TTY, take the explicit plain branch and bypass
  lipgloss entirely. "Byte-identical when piped" cannot depend on a library
  heuristic.
  `[auto] Plain/pretty switch → Selected: explicit isTTY branch bypassing lipgloss on non-TTY (recommended)`
- **D-05:** Honor `NO_COLOR` (de-facto standard): a non-empty `NO_COLOR` forces
  the plain branch even on a TTY. **No** new `--color`/`--no-color` flag in
  v1.0 — TTY-detection + `NO_COLOR` covers the cases with minimal new surface.
  An explicit override flag is a Deferred idea.
  `[auto] Color override → Selected: NO_COLOR env only, no new flag (recommended)`
- **D-06:** Product output for `status`/`files` keeps going to the same stream
  it uses today (stdout); the pretty branch changes styling only, never which
  stream product output lands on. (Progress is separate — D-08.)

### Progress feedback (TUI-05)
- **D-07:** Scope to `init` / `index` / `sync`. Spinner for indeterminate work,
  progress for known-total phases; same `term.IsTerminal` + `NO_COLOR` gate as
  D-04/D-05. Non-TTY ⇒ current plain behavior (no spinner frames, no ANSI).
- **D-08:** Progress writes to **stderr**, never stdout — consistent with the
  Phase-4 hygiene discipline (stdout stays clean/pipe-safe; for `serve`,
  JSON-RPC-pure). This also structurally keeps progress off the agent/MCP
  surface.
  `[auto] Progress stream → Selected: stderr, TTY-gated (recommended)`
- **D-09:** v1.0 uses a **non-interactive** progress renderer — lipgloss-styled
  spinner/progress frames on a ticker, or `charm.land/bubbles` spinner/progress
  rendered **without** a bubbletea event loop. NO stdin reading, NO full
  bubbletea `Program`. Interactive event-loop components are Phase 7 (TUI-03/04).
  Phase 6 = the seam + passive rendering; the "never hangs when piped"
  interactive fallback is Phase 7's concern.
  `[auto] Progress component → Selected: passive lipgloss/bubbles frames, no bubbletea loop (recommended)`

### Archtest / ANSI-isolation guarantee (TUI-01)
- **D-10:** A new archtest package (e.g. `internal/cli/present/archtest/` or
  `internal/archtest/`) **mirrors `internal/graphstore/archtest/import_graph_test.go`
  verbatim in mechanism**: `packages.Load` over
  `github.com/seanb4t/codegraph-go/...` with `Tests: true` and
  `NeedImports|NeedName|NeedDeps`, asserting no forbidden import path appears in
  the transitive imports of the guarded packages. NOT regex/source scanning
  (misses aliased imports, build-tag files, test variants — the precedent's
  stated reason).
- **D-11:** Forbidden import paths: `charm.land/lipgloss`,
  `charm.land/bubbletea`, `charm.land/bubbles` (charm **v2 vanity paths**, not
  `github.com/charmbracelet/...`). Guarded set: the requirement names
  `internal/query` + `internal/mcp` (hard requirement); **extend to the full
  serve-reachable set** already used by the Phase-4 stdout archtest
  (`internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`,
  `internal/indexer`, `internal/query`) for defense-in-depth — charm must never
  reach the agent path.
- **D-12:** Keep the same **self-defeat sanity guard** the precedent carries
  (`import_graph_test.go:64`): fail if NO package imports charm at all, so a
  refactor that drops the dep can't silently vacuously-pass. Build order
  (ROADMAP note): land the archtest **FIRST**, watch it fail-closed, THEN add
  `present` + the Charm deps.

### Charm v2 dependency scope
- **D-13:** Add `charm.land/lipgloss` (v2) now — required for TUI-02
  status/files styling and TUI-05 progress. Add `charm.land/bubbles` **only if**
  the progress/spinner path uses it (D-09); otherwise defer. Do **not** add
  `charm.land/bubbletea` in Phase 6 (the interactive event loop is Phase 7).
  Pin exact versions in `go.mod`; prefer pure-Go/no-CGo by construction (the
  full REL-01 closure audit is Phase 8).
  `[auto] Dep scope → Selected: lipgloss now, bubbles if needed, bubbletea deferred to P7 (recommended)`

### Claude's Discretion
- Exact lipgloss palette / section styling for status & files (borders,
  key-bolding, language-list grouping) — bounded by the "plain path
  byte-identical" invariant; reuse Phase-2 section wording and order.
- Spinner glyphs, progress-bar styling, tick cadence.
- Precise archtest package location/name — must mirror the precedent's
  mechanics regardless of where it lives.

### Reviewed Todos (not folded)
- **"Document release procedures (maintainer runbook)"** (score 0.6) — reviewed,
  **NOT folded**. It matched only on generic keywords (phase/github/internal);
  it is a release-process runbook belonging to **Phase 8** (Signed v1.0.0
  Release / REL-02), not the rendering seam. Deferred to Phase 8.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope (locked)
- `.planning/ROADMAP.md` §"Phase 6: Rendering Seam & Pretty status/files" —
  goal, 4 success criteria, and the Notes (archtest-first; `internal/cli/present`;
  `charm.land/...` v2 vanity import; `golang.org/x/term.IsTerminal` gating).
- `.planning/REQUIREMENTS.md` — TUI-01, TUI-02, TUI-05 (plus SURF-06 for the
  markdown renderers the pretty path wraps).

### Archtest precedents to mirror (TUI-01)
- `internal/graphstore/archtest/import_graph_test.go` — the exact import-graph
  archtest to mirror (`packages.Load`, `Tests:true`, allowed-prefix check,
  `stripTestVariant`, self-defeat guard at line 64).
- `internal/graphstore/archtest/stdout_confinement_test.go` — Phase-4 stdout
  confinement archtest (the serve-reachable guarded-package-set precedent).
- `internal/migrate/archtest/modernc_confinement_test.go` — second
  import-confinement precedent.

### Plain-path renderers to preserve byte-for-byte (TUI-02)
- `internal/query/render_status.go` — `RenderStatusText` (plain sectioned
  status; the piped path — UNCHANGED).
- `internal/query/render_results.go`, `internal/query/render_markdown.go` —
  files/results markdown + plain renderers (SURF-06).
- `internal/query/files.go`, `internal/query/status.go` — result producers
  (`StatusResult`, files result).
- `internal/cli/status.go` §18-20 — documents the Phase-6 handoff;
  `internal/cli/files.go` — the `files` command.
- `internal/cli/query.go` §27 — output-discipline pattern (os.Stdout handling).

### Cross-phase decisions carried forward
- `.planning/phases/04-output-hygiene/04-CONTEXT.md` — D-04/D-06: stderr-only
  diagnostics discipline + the two-layer (subprocess + archtest) enforcement
  pattern this phase reuses.
- `.planning/phases/02-status-content-git-worktree-awareness/02-CONTEXT.md` —
  D-09/D-17: the status section layout & wording being colorized.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `query.RenderStatusText` + the `files` markdown/plain renderers — the exact
  plain bytes; the pretty path decorates the same `StatusResult`/data and never
  replaces these.
- The archtest harness in `import_graph_test.go` (`packages.Load` +
  `stripTestVariant` helpers) — copy the mechanism; swap the forbidden import
  path(s) and the allowed/guarded sets.
- `query.StatusResult` (documented TS→Go/Pebble key remapping) — the struct the
  `present` package consumes.

### Established Patterns
- Test-only injectable `io.Writer` seam (Phase 3/4) — reuse for the `present`
  renderer and the progress writer so tests capture output.
- "One seam, whole surface" build-enforced archtest, run in CI **alongside**
  `go test ./testdata/golden/...`.
- Additive sibling renderers (SURF-06's `Render*` next to `Marshal*JSON`) — same
  shape here: `present.Render*` beside `query.Render*Text`, originals untouched.

### Integration Points
- `internal/cli/status.go` & `files.go` `RunE`: add the `isTTY` branch calling
  `internal/cli/present`.
- `init` / `index` / `sync` command `RunE`: wire the TTY-gated stderr progress.
- New `go.mod` requires: `charm.land/lipgloss` (+ maybe `charm.land/bubbles`),
  `golang.org/x/term`.

</code_context>

<specifics>
## Specific Ideas

- Charm **v2** vanity import is `charm.land/...` (NOT
  `github.com/charmbracelet/...`) — locked in ROADMAP Notes.
- TTY gate is `golang.org/x/term.IsTerminal` — locked in ROADMAP Notes.
- Archtest lands and fails-closed **before** any Charm dep enters `go.mod`.

</specifics>

<deferred>
## Deferred Ideas

- `--color` / `--no-color` explicit override flag — v1.0 relies on TTY-detection
  + `NO_COLOR`.
- Interactive bubbletea components — daemon picker (DMON-01/TUI-04) and
  `install`/`uninstall` multi-select (TUI-03) → **Phase 7**.
- Full Charm dependency-closure audit (CGo / govulncheck / SBOM / reproducible
  double-build, REL-01) → **Phase 8**.
- Colorizing read commands beyond `status`/`files` (explore/node/callers/…) —
  out of scope; this phase names `status` + `files` only.

### Reviewed Todos (not folded)
- "Document release procedures (maintainer runbook)" — release-process runbook;
  belongs to Phase 8 (REL-02), not the rendering seam.

</deferred>

---

*Phase: 6-Rendering Seam & Pretty status/files*
*Context gathered: 2026-07-17*
