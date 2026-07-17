# Phase 6: Rendering Seam & Pretty status/files - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-17
**Phase:** 6-Rendering Seam & Pretty status/files
**Mode:** --auto --all --chain (all gray areas auto-selected; recommended option chosen per question)
**Areas discussed:** Rendering-seam architecture, TTY/plain-pretty switch, Progress feedback (TUI-05), Archtest structure & guarded set (TUI-01), Charm v2 dependency scope

---

## Rendering-seam architecture (where lipgloss lives)

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated `internal/cli/present` package consuming plain result structs | ROADMAP-named seam; query/mcp stay charm-free; pretty path is additive | ✓ |
| Colorize inside `internal/query` renderers with a color flag | Would make lipgloss reachable from query → archtest can never pass | |
| Style at each command site ad-hoc | No single seam; archtest boundary becomes fuzzy | |

**Auto choice:** dedicated `internal/cli/present` (recommended).
**Notes:** `internal/cli/status.go:18-20` already documents this exact handoff; `query.RenderStatusText` lives in `internal/query`, which the archtest forbids from importing lipgloss — so the pretty renderer MUST live outside query. Plain path preserved by reuse, not re-derivation (D-02).

---

## TTY / plain-pretty switch mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit `isTTY` branch that bypasses lipgloss on non-TTY | `term.IsTerminal(os.Stdout.Fd())`; guarantees byte-identical piped output | ✓ |
| Rely on lipgloss color-profile auto-degrade | Byte-identity would depend on a library heuristic — too risky for the golden guarantee | |

**Auto choice:** explicit branch + `NO_COLOR` env; no new `--color` flag (recommended).
**Notes:** `NO_COLOR` (non-empty) forces plain even on a TTY. `--color`/`--no-color` flag deferred.

---

## Progress feedback (TUI-05)

| Option | Description | Selected |
|--------|-------------|----------|
| Passive lipgloss/bubbles frames on stderr, no bubbletea loop | TTY-gated; stdout stays pipe/JSON-RPC clean; no stdin | ✓ |
| Full bubbletea Program per operation | Interactive event loop = Phase 7 scope; heavier; hang risk when piped | |
| Progress on stdout | Would pollute piped stdout and the hygiene guarantee | |

**Auto choice:** passive frames, stderr, TTY-gated (recommended).
**Notes:** Scope = init/index/sync. Interactive components are Phase 7 (TUI-03/04).

---

## Archtest structure & guarded set (TUI-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror `import_graph_test.go` (`packages.Load`, Tests:true, self-defeat guard); guard full serve-reachable set | Proven precedent; catches aliases/build-tags/test variants | ✓ |
| Regex/source scan for charm imports | Misses aliased imports, build-tag files, test variants | |
| Guard only the two named packages (query, mcp) | Meets the letter of TUI-01 but weaker defense-in-depth | |

**Auto choice:** mirror precedent; forbid `charm.land/lipgloss|bubbletea|bubbles`; guard the Phase-4 serve-reachable set (recommended).
**Notes:** Land archtest FIRST, fail-closed, then add present + deps (ROADMAP note). Keep the self-defeat sanity guard.

---

## Charm v2 dependency scope

| Option | Description | Selected |
|--------|-------------|----------|
| lipgloss now; bubbles only if progress needs it; bubbletea deferred to Phase 7 | Minimal closure for this phase's needs | ✓ |
| Add lipgloss + bubbletea + bubbles all now | Pulls the interactive event loop in a phase early | |

**Auto choice:** lipgloss now, bubbles-if-needed, bubbletea deferred (recommended).
**Notes:** charm v2 vanity import is `charm.land/...`. Pin exact versions. Full REL-01 closure audit is Phase 8.

---

## Claude's Discretion

- lipgloss palette/section styling for status & files (bounded by byte-identical plain invariant).
- spinner glyphs / progress-bar styling / tick cadence.
- exact archtest package location/name (mechanics must mirror the precedent).

## Deferred Ideas

- `--color`/`--no-color` override flag (v1.0 uses TTY + NO_COLOR).
- Interactive bubbletea components: daemon picker (DMON-01/TUI-04), install multi-select (TUI-03) → Phase 7.
- Full Charm dependency-closure audit (CGo/govulncheck/SBOM/reproducible build, REL-01) → Phase 8.
- Colorizing read commands beyond status/files → out of scope.
- Reviewed todo "Document release procedures (maintainer runbook)" (score 0.6) — not folded; belongs to Phase 8 (REL-02).
