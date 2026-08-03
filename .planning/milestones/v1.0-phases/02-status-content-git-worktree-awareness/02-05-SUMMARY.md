---
phase: 02-status-content-git-worktree-awareness
plan: 05
subsystem: query
tags: [status, render, cli, mcp, tdd, formatNumber, worktree]

requires:
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-01's internal/gitmeta.Mismatch.Warning()/Notice() and 02-04's Engine.WorktreeMismatch()/query.WorktreeNotice/query.WorktreeWarningBlockquote — this plan's RenderStatusText/RenderStatusMarkdown call these directly"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-02's StatusResult.DbSizeBytes/FilesByLanguage — this plan's sole consumer of both fields (render-only, no new scans)"
provides:
  - "query.RenderStatusText(r StatusResult, projectPath string) string — CLI padded-column status layout (D-09), for plan 02-07 to wire into internal/cli/status.go"
  - "query.RenderStatusMarkdown(r StatusResult) string — MCP bolded-key bullet status layout (D-17), for plan 02-06 to wire into internal/mcp/tools.go's codegraph_status handler"
  - "query.formatNumber(n int64) string (unexported) — en-US comma grouper, no new dependency (D-10)"
  - "query.formatMB(bytes int64) string (unexported) — two-decimal MB rendering (D-07)"
  - "query.sortedCounts(m map[string]int64) []kindCount (unexported) — shared count>0 filter + count-DESC sort + key-ascending tiebreak, used by both renderers' Nodes-by-Kind and Files-by-Language/Languages breakdowns"
affects: [02-06, 02-07]

tech-stack:
  added: []
  patterns:
    - "Two structurally different Render* functions sharing one StatusResult data source but not a renderer — each carries a doc comment naming its single owning surface (CLI-only vs MCP-only) so a future edit collapsing them fails TestRenderStatusMarkdownShape's 'no Index Statistics: header' assertion immediately (T-02-22)"
    - "writeStatusAdvisories(b, r, staleLabel, reindexLabel) — one shared implementation for the staleness/reindex advisory pair, parameterized only by label text so RenderStatusText and RenderStatusMarkdown keep their own voice (plain vs bolded-key) without duplicating the r.Stale / r.Index.ReindexRecommended branching logic"
    - "sortedCounts breaks count ties on key ascending — a documented, deterministic substitute for TS's Object.entries insertion-order tiebreak, which Go's randomized map iteration cannot reproduce"

key-files:
  created:
    - internal/query/render_status.go
    - internal/query/render_status_test.go
  modified: []

key-decisions:
  - "Tasks 2 (GREEN: RenderStatusText) and 3 (GREEN: RenderStatusMarkdown) were combined into a single feat commit: Go requires whole-package compilation, and Task 1's RED test file already references RenderStatusMarkdown, so the package cannot build — let alone run any -run-filtered subset of tests — until both renderers exist. Splitting them into two commits would require git add -p line-splitting of what is effectively one file addition, with no independent verifiable state in between; combining them follows the same precedent 01-08-SUMMARY.md recorded for an identical Go-compilation constraint (\"Task 1/2 combined into one feat commit — shared file, no independent dependency\")"
  - "formatNumber/formatMB/sortedCounts/writeStatLine/writeBreakdownText/writeBreakdownMarkdown/writeStatusAdvisories all live in the single new render_status.go file (not split across CLI/MCP files) — both renderers need the identical filter/sort/format primitives, and CONTEXT.md's Claude's Discretion note explicitly left this file-layout choice open"
  - "Advisory wording (staleness/reindex lines) is this plan's own text, not a TS port — CONTEXT.md's D-09 target structure names the advisory's PRESENCE and its live-signal source (r.Stale, r.Index.ReindexRecommended) but the TS 1.3.1 white-box source captured this session did not preserve the exact advisory string, and CONTEXT.md's Claude's Discretion section covers wording choices generally; the wording keys off 'stale'/'reindex'/'pending'/'up to date' substrings, which is what both this plan's tests and any downstream consumer should match on, not exact prose"
  - "RenderStatusMarkdown's Nodes-by-Kind/Languages advisory labels are bolded ('**Pending Changes:**'/'**Reindex recommended:**') to match the MCP form's bolded-key convention throughout, while RenderStatusText's are plain ('Pending Changes:'/'Reindex recommended:') to match the CLI form's padded-column convention — writeStatusAdvisories takes the label text as a parameter specifically so each renderer keeps its own voice without duplicating the r.Stale/r.Index.ReindexRecommended branch logic"

requirements-completed: [STAT-01, STAT-02, STAT-03]

coverage:
  - id: D1
    description: "formatNumber groups by thousands with commas, deterministically and locale-independently (0/999/1000/1223/1234567/-1234)"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestFormatNumber"
        status: pass
    human_judgment: false
  - id: D2
    description: "formatMB renders a two-decimal MB value matching ^\\d+\\.\\d{2} MB$"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestFormatMB"
        status: pass
    human_judgment: false
  - id: D3
    description: "sortedCounts filters count>0, sorts DESC, and breaks ties on key ascending — deterministic despite randomized map iteration"
    requirement: STAT-02
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestSortedCounts"
        status: pass
    human_judgment: false
  - id: D4
    description: "RenderStatusText emits Index Statistics:/Nodes by Kind:/Files by Language: section headers with padded stat labels and no Journal: line; Backend renders r.Backend, not a hardcoded TS string"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextSections"
        status: pass
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextBackendFromField"
        status: pass
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextNoJournalLine"
        status: pass
    human_judgment: false
  - id: D5
    description: "Both breakdowns filter zero-count entries, sort by count DESC, and pad the key to 15 columns in RenderStatusText's rendered output"
    requirement: STAT-02
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextBreakdownFilterSortPad"
        status: pass
    human_judgment: false
  - id: D6
    description: "The staleness advisory is driven by r.Stale (never PendingChanges) and the reindex advisory by r.Index.ReindexRecommended, in both renderers"
    requirement: STAT-03
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextStaleAdvisory"
        status: pass
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextReindexAdvisory"
        status: pass
    human_judgment: false
  - id: D7
    description: "RenderStatusText embeds the verbose worktree warning when WorktreeMismatch is non-nil, omits it when nil"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextWorktreeWarning"
        status: pass
    human_judgment: false
  - id: D8
    description: "RenderStatusMarkdown is structurally different from RenderStatusText — bolded-key bullets, no Index Statistics: header, '- key: count' breakdown bullets"
    requirement: STAT-02
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusMarkdownShape"
        status: pass
    human_judgment: false
  - id: D9
    description: "RenderStatusMarkdown embeds the worktree warning as a blockquote via WorktreeWarningBlockquote — every line begins with '> '"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusMarkdownWorktreeBlockquote"
        status: pass
    human_judgment: false
  - id: D10
    description: "Neither renderer emits an ANSI escape byte (0x1b)"
    requirement: STAT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusNoANSI"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 5: Status Renderers (CLI padded columns + MCP bolded-key bullets) Summary

**Built the two structurally different status renderers TS ships — `RenderStatusText` (CLI padded-column layout) and `RenderStatusMarkdown` (MCP bolded-key bullet layout) — sharing one `StatusResult` data source, a hand-rolled locale-independent `formatNumber`, and a shared `sortedCounts` breakdown filter/sort primitive, with no new dependency.**

## Performance

- **Duration:** 25 min
- **Tasks:** 2 commits (TDD RED, then a combined GREEN — see Deviations)
- **Files created:** 2 (`internal/query/render_status.go`, `internal/query/render_status_test.go`)

## Accomplishments

- `query.RenderStatusText(r StatusResult, projectPath string) string` — TS's `bin/codegraph.js` ~900-985 padded-column layout: `CodeGraph Status` header, `Project: <path>`, the verbose worktree warning (when present), `Index Statistics:` with `  Files:     `/`  Nodes:     `/`  Edges:     `/`  DB Size:   `/`  Backend:   ` rows (11-column label padding), `Nodes by Kind:`/`Files by Language:` breakdowns (15-column key padding, count>0 filtered, count-DESC sorted), and the staleness/reindex advisories. `Journal:` is dropped (no Pebble analog); `Backend:` renders `r.Backend` (`pebble`), never TS's `node:sqlite` string.
- `query.RenderStatusMarkdown(r StatusResult) string` — TS's `mcp/tools.js` ~3890-3945 bolded-key bullet layout: `**CodeGraph Status**`, the blockquoted worktree warning (via `WorktreeWarningBlockquote`, not a second inline transform), `**Files indexed:**`/`**Total nodes:**`/`**Total edges:**`/`**Database size:**`/`**Backend:**`, `**Nodes by Kind:**`/`**Languages:**` bullet lists (`- key: count`), and the same staleness/reindex advisories with bolded labels. Structurally different from the CLI form by design and by test (no `Index Statistics:` header).
- `formatNumber` — a hand-rolled, deterministic en-US comma grouper (`1223` → `"1,223"`, `1234567` → `"1,234,567"`, `-1234` → `"-1,234"`), replacing TS's locale-dependent `n.toLocaleString()`. No `golang.org/x/text/message` dependency added (D-10).
- `formatMB` — `fmt.Sprintf("%.2f MB", bytes/1024/1024)`, matching TS's `.toFixed(2)` (D-07).
- `sortedCounts` — the shared filter (`count > 0`) + sort (count DESC, key-ascending tiebreak) primitive both renderers' Nodes-by-Kind and Files-by-Language/Languages breakdowns use.

## Task Commits

1. **Task 1: RED — formatNumber, MB rendering, and both status renderers** — `da93274` (test)
2. **Tasks 2+3: GREEN — formatNumber/formatMB/sortedCounts/RenderStatusText/RenderStatusMarkdown** — `26b74ae` (feat)

No REFACTOR commit was needed — the GREEN implementation matched Task 1's target contract with no follow-up cleanup.

## Files Created

- `internal/query/render_status_test.go` — 13 tests covering formatNumber, formatMB, sortedCounts, both renderers' sections/padding/backend/no-Journal/breakdown-order/advisories/worktree-warning/blockquote/ANSI-freedom
- `internal/query/render_status.go` — both renderers, their shared `kindCount`/`formatNumber`/`formatMB`/`sortedCounts`/`writeStatLine`/`writeBreakdownText`/`writeBreakdownMarkdown`/`writeStatusAdvisories` primitives, each with a doc comment naming its single owning surface

## Sample Renderings (against a fixture StatusResult)

**`RenderStatusText`** (no worktree mismatch, files=100, nodes=1,234,567, edges=1,223, dbSize=1234567 bytes, NodesByKind={function:10}, FilesByLanguage={go:42,python:7,yaml:0,javascript:19}, Stale=false, ReindexRecommended=false):

```
CodeGraph Status

Project: /Users/sean/code/codegraph-go

Index Statistics:
  Files:     100
  Nodes:     1,234,567
  Edges:     1,223
  DB Size:   1.18 MB
  Backend:   pebble
Nodes by Kind:
  function        10
Files by Language:
  go              42
  javascript      19
  python          7

Index is up to date.
```

**`RenderStatusMarkdown`** (same fixture, with a worktree mismatch + Stale=true + ReindexRecommended=true):

```
**CodeGraph Status**

> ⚠ This CodeGraph index belongs to a different git working tree.
>   Running in: /Users/sean/.claude/worktrees/phase-02
>   Index from: /Users/sean/code/codegraph-go
> Results reflect that tree's code (often a different branch), not this worktree — symbols changed only here are missing. Run "codegraph init -i" in this worktree for a worktree-local index.

**Files indexed:** 100
**Total nodes:** 1,234,567
**Total edges:** 1,223
**Database size:** 1.18 MB
**Backend:** pebble

**Nodes by Kind:**
- function: 10

**Languages:**
- go: 42
- javascript: 19
- python: 7

**Pending Changes:** a sync is recommended — this index may be stale. Run "codegraph sync" to update.

**Reindex recommended:** this index predates the current schema version. Run "codegraph index --force" to rebuild.
```

Note the `yaml:0` entry (zero count) is filtered from both renderings, and the survivors appear in count-DESC order (go=42 > javascript=19 > python=7) in both — confirming `sortedCounts` is genuinely shared.

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:
- **Tasks 2+3 combined into one commit** — Go's whole-package compilation model means Task 1's RED test file (referencing `RenderStatusMarkdown`) blocks the package from building at all until both renderers exist; there is no way to get an independently-buildable intermediate state between them. Documented as this plan's one deviation from the literal task-per-commit instruction.
- **Advisory wording is this plan's own text**, not a byte-for-byte TS port — the captured TS 1.3.1 source for this session's D-01 read did not preserve the exact `Pending Changes:`/reindex advisory strings, and CONTEXT.md's D-09 target structure specifies the advisory's *presence* and its live-signal source, not its exact prose. Both renderers' tests match on substrings (`stale`/`pending`/`up to date`/`reindex`), which is the durable contract for downstream callers.
- **`sortedCounts`'s tiebreak is key-ascending**, not TS's `Object.entries` insertion order — Go's map iteration is deliberately randomized, so a key tiebreak is the only deterministic substitute available; documented as a minor, intentional divergence (both in the code doc comment and here).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Task 2/Task 3 split is not independently buildable in Go**
- **Found during:** Task 2 (GREEN — attempting to commit `RenderStatusText` alone)
- **Issue:** The plan structures Task 2 (GREEN: `RenderStatusText`) and Task 3 (GREEN: `RenderStatusMarkdown`) as two separate commits to the same file, each with its own `go test ./internal/query/... -run '...'` verify command. But Task 1's RED commit (`da93274`) already added `render_status_test.go`, which references `RenderStatusMarkdown` directly (Tests 10/11/12). Go compiles a package as a whole before running any tests, so `-run` filtering cannot make the package buildable with only `RenderStatusText` present — every test in the file, including Task 3's, must compile even if not executed.
- **Fix:** Implemented both `RenderStatusText` and `RenderStatusMarkdown` (plus their shared primitives) in one pass, committed as a single `feat` commit covering both Task 2 and Task 3's GREEN scope. All acceptance-criteria grep checks (`Journal:`, `node:sqlite`, `PendingChanges`, `lipgloss`/`charm.land`/`x/text`, `ReplaceAll`) were run against the final file and all return 0, satisfying both tasks' literal acceptance criteria despite the commit-count change.
- **Files modified:** `internal/query/render_status.go`
- **Verification:** `go test ./internal/query/... -run 'TestRenderStatusText|TestFormatNumber|TestFormatMB|TestSortedCounts' -v` and `go test ./internal/query/... -run TestRenderStatusMarkdown -v` both pass; full `go test ./...` green; `go vet ./internal/query/...` clean.
- **Committed in:** `26b74ae`

---

**Total deviations:** 1 auto-fixed (Rule 3 — a Go-compilation-model blocking issue in the plan's task/commit split, not a code defect). No scope creep; no product behavior differs from what either task specified.

## Issues Encountered

None beyond the Rule 3 deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `query.RenderStatusText` is ready for plan 02-07 to wire into `internal/cli/status.go`, replacing the terse `backend=… files=… nodes=…` line. The CLI supplies `projectPath` (its own resolved start path) as the second argument.
- `query.RenderStatusMarkdown` is ready for plan 02-06 to wire into `internal/mcp/tools.go`'s `codegraph_status` handler, replacing `MarshalStatusJSON`'s direct call there (the CLI `--json` path keeps using `MarshalStatusJSON` unchanged — confirmed via `git diff internal/query/status.go` returning empty).
- Both renderers already correctly consume `Engine.WorktreeMismatch()`/`StatusResult.WorktreeMismatch` (live since plan 02-04) and `StatusResult.DbSizeBytes`/`FilesByLanguage` (live since plan 02-02) — no further plumbing is needed in `internal/query` for STAT-01/02/03.
- Zero new dependencies; `go.mod`/`go.sum` are byte-identical to before this plan (`git diff --stat go.mod go.sum` empty, confirmed).
- `MarshalStatusJSON` and `StatusResult` were never touched by this plan (`git diff internal/query/status.go` empty) — the CLI `--json` contract and golden-parity oracle remain intact.

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*

## Self-Check: PASSED

Both referenced files (`internal/query/render_status.go`, `internal/query/render_status_test.go`) verified present on disk; both task commits (`da93274`, `26b74ae`) verified present in git log.
