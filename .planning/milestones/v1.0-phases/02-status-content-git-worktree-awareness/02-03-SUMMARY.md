---
phase: 02-status-content-git-worktree-awareness
plan: 03
subsystem: api
tags: [markdown, mcp, render, tdd, surf-06]

requires:
  - phase: 01-behavioral-parity-explore-node
    provides: "internal/query render seam (RenderExplore/RenderNode, staleBanner prepend pattern, formatNodeRef/pluralize helpers) this plan's renderers extend"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-02's Engine.Status()/StatusResult (DbSizeBytes, FilesByLanguage) — unrelated to this plan's renderers but same wave"
provides:
  - "5 new markdown renderers in internal/query: RenderCallersMarkdown, RenderCalleesMarkdown, RenderImpactMarkdown, RenderSearchMarkdown, RenderFilesMarkdown"
  - "renderLocationTable([]Location) — the shared table helper backing 4 of the 5 renderers"
  - "renderFileTreeMarkdown([]*FileTreeNode, indent) — the tree-union-branch renderer, ported from internal/cli/files.go's printFileTree"
  - "Unit-test contract proving each renderer's output is not valid JSON, preserves input order, and is ANSI-free"
affects: [02-06]

tech-stack:
  added: []
  patterns:
    - "Marshal*JSON (CLI-only) vs Render*Markdown (MCP-only) caller asymmetry — documented at the file's package comment so a future reader doesn't 'helpfully' reunify them"
    - "Shared renderLocationTable([]Location) backing 4 thin per-tool header renderers, since 4 of 5 SURF-06 payloads share the same Location record type"
    - "Empty-result renders an explicit worded no-results sentence naming the symbol/term, never a headerless table"

key-files:
  created:
    - internal/query/render_results.go
    - internal/query/render_results_test.go
  modified: []

key-decisions:
  - "Collapsed FilePath+StartLine into a single backticked path:line cell in renderLocationTable, matching the existing formatNodeRef convention rather than adding a 5th column"
  - "renderFileTreeMarkdown duplicates internal/cli/files.go's printFileTree algorithm rather than importing/moving it — internal/query must not import internal/cli, and moving printFileTree would put a CLI file in this plan's files_modified, creating a wave conflict with plan 02-07 (same package-local-duplication precedent status.go's shouldSkipStaleDir already set)"
  - "Used a temporary RenderFilesMarkdown panic-stub in Task 2's commit to satisfy Go's whole-package compilation, since Task 1's single RED test file already carried Task 3's file-renderer tests; Task 3 replaced the stub before its own tests ran (documented under Deviations)"

patterns-established:
  - "Render* functions in internal/query return ANSI-free plain strings and never re-sort an already-deterministic input slice — enforced by acceptance-criteria grep checks (sort\\. count == 0, lipgloss/charm.land count == 0) and will apply to any future Render* additions in this package"

requirements-completed: [SURF-06]

coverage:
  - id: D1
    description: "renderLocationTable renders a shared markdown table (header once, rows in input order) backing RenderCallersMarkdown/RenderCalleesMarkdown/RenderImpactMarkdown/RenderSearchMarkdown"
    requirement: "SURF-06"
    verification:
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderLocationTable"
        status: pass
    human_judgment: false
  - id: D2
    description: "RenderCallersMarkdown/RenderCalleesMarkdown/RenderImpactMarkdown/RenderSearchMarkdown each produce non-JSON markdown, preserve input order, render an explicit no-results sentence on empty input, and emit no ANSI"
    requirement: "SURF-06"
    verification:
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderCallersMarkdown"
        status: pass
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderCalleesMarkdown"
        status: pass
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderImpactMarkdown"
        status: pass
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderSearchMarkdown"
        status: pass
    human_judgment: false
  - id: D3
    description: "RenderFilesMarkdown renders both FilesResult union branches: a 4-column flat table, and an indented plain-text tree (not forced into a table)"
    requirement: "SURF-06"
    verification:
      - kind: unit
        ref: "internal/query/render_results_test.go#TestRenderFilesMarkdown"
        status: pass
    human_judgment: false
  - id: D4
    description: "No Marshal*JSON body was modified (D-16's additive rule) — CLI --json path and golden parity oracle unchanged"
    requirement: "SURF-06"
    verification:
      - kind: unit
        ref: "git diff internal/query/traverse.go internal/query/files.go internal/query/search.go internal/cli/ go.mod go.sum (empty)"
        status: pass
      - kind: integration
        ref: "go test ./... (full suite, incl. internal/cli/query_cli_test.go's TestSearchCmd/TestCallersCalleesCmd/TestImpactCmd/TestFilesCmd JSON regression guards)"
        status: pass
    human_judgment: false

duration: 48min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 3: SURF-06 Markdown Renderers Summary

**Five new sibling markdown renderers in `internal/query` (callers/callees/impact/search/files) added strictly additively next to the untouched `Marshal*JSON` CLI helpers, measured at -38.7% bytes on `files` against this repo's own 308-file index.**

## Performance

- **Duration:** 48 min
- **Started:** 2026-07-15T22:49:00Z (approx.)
- **Completed:** 2026-07-15T23:37:39Z
- **Tasks:** 3
- **Files modified:** 2 (both new)

## Accomplishments
- `renderLocationTable([]Location) string`: one shared markdown table helper backing the 4 renderers whose payloads share `Location{Name, Kind, FilePath, StartLine}` — header once, one row per record, `path:line` collapsed into a single backticked cell
- `RenderCallersMarkdown`, `RenderCalleesMarkdown`, `RenderImpactMarkdown`, `RenderSearchMarkdown`: thin per-tool renderers, each with a bolded header carrying its own scalars (symbol/term, count; impact additionally carries depth/nodeCount/edgeCount) and an explicit worded no-results sentence on empty input
- `RenderFilesMarkdown`: both `FilesResult` union branches — a 4-column flat table (Path/Language/Nodes/Edges) and, via new `renderFileTreeMarkdown`, an indented plain-text tree ported from `internal/cli/files.go`'s `printFileTree` algorithm
- Zero `Marshal*JSON` bodies touched — verified by an empty `git diff` on `traverse.go`/`files.go`/`search.go`/`internal/cli/`/`go.mod`/`go.sum`, and by the full `go test ./...` suite staying green including the CLI `--json` regression guard tests (`TestSearchCmd`, `TestCallersCalleesCmd`, `TestImpactCmd`, `TestFilesCmd`)
- Measured on this repo's own index (308 files): `files` JSON = 28,505 bytes, markdown = 17,471 bytes (**-38.7%**) — directionally confirms D-16's cited -41% figure (the small delta from D-16's number is expected: that figure was a separate, earlier measurement, not re-run in this exact form)

## Task Commits

Each task was committed atomically (TDD RED/GREEN):

1. **Task 1: RED — renderer unit tests for all 5 shapes** - `cbd997a` (test)
2. **Task 2: GREEN — shared location table + 4 Location-backed renderers** - `b178757` (feat)
3. **Task 3: GREEN — RenderFilesMarkdown, both union branches** - `a2d3492` (feat)

_No REFACTOR commit was needed — both GREEN implementations matched the test file's target shape from Task 1 with no follow-up cleanup._

## Files Created/Modified
- `internal/query/render_results.go` - the 5 `Render*Markdown` functions + `renderLocationTable` + `renderFileTreeMarkdown`, plus a package comment recording the `Marshal*JSON`-CLI-only vs `Render*Markdown`-MCP-only caller asymmetry
- `internal/query/render_results_test.go` - table-driven tests for all 5 renderers: not-valid-JSON, positive markdown marker, empty-set sentence, non-alphabetical order preservation, single/multi-row, and ANSI-freedom; plus the `files` tree-branch's own indentation-depth assertion

## Decisions Made
- Collapsed `FilePath`+`StartLine` into one `path:line` cell in `renderLocationTable`, matching the existing `formatNodeRef` `"%s (%s:%d)"` convention rather than adding a 5th table column
- Ported (duplicated, not moved) `internal/cli/files.go`'s `printFileTree` into `internal/query/render_results.go` as `renderFileTreeMarkdown` — `internal/query` must not import `internal/cli`, and moving the CLI file would put it in this plan's `files_modified`, conflicting with plan 02-07's wave ownership. Same precedent as `status.go`'s `shouldSkipStaleDir`.
- `RenderFilesMarkdown` treats both `""` and the literal `"flat"` as the flat branch, matching `FilesOptions.Format`'s documented default and the MCP tool's own `req.GetString("format", "")` default
- Recorded, per plan instruction, that TS's MCP `files` defaults to `tree` while ours defaults to `flat` — a pre-existing divergence explicitly out of scope for this plan (Phase 8 SURF territory)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Temporary `RenderFilesMarkdown` stub in Task 2's commit**
- **Found during:** Task 2 (GREEN — shared location table + 4 Location-backed renderers)
- **Issue:** Task 1's RED commit put all 5 renderers' tests in one file (`render_results_test.go`, as the plan specified). Go compiles an entire package before applying a `-run` filter, so Task 2's verify command (`go test ./internal/query/... -run 'TestRenderCallers|TestRenderCallees|TestRenderImpact|TestRenderSearch|TestRenderLocationTable' -v`) could not build — `TestRenderFilesMarkdown` (Task 3's test) referenced the not-yet-implemented `RenderFilesMarkdown`, regardless of the `-run` filter excluding it.
- **Fix:** Added a minimal `RenderFilesMarkdown(r FilesResult) string { panic(...) }` stub at the end of Task 2's commit, purely to satisfy compilation. The stub was never exercised (excluded by Task 2's `-run` filter) and was fully replaced with the real flat/tree implementation in Task 3, in the same commit that added `renderFileTreeMarkdown`.
- **Files modified:** `internal/query/render_results.go` (stub added in Task 2's commit `b178757`, replaced in Task 3's commit `a2d3492`)
- **Verification:** Task 2's filtered test run passed GREEN with the stub present but unreached; Task 3's full-package `go test ./internal/query/...` and `go test ./...` passed GREEN with the stub fully replaced.
- **Committed in:** `b178757` (stub), superseded by `a2d3492` (real implementation)

---

**Total deviations:** 1 auto-fixed (1 blocking — a Go-compilation-model consequence of the plan's single-test-file structure, not a plan defect)
**Impact on plan:** No scope creep; the stub was never part of any assertion path and is gone by the plan's final commit.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 5 `Render*Markdown` functions plus `renderLocationTable`/`renderFileTreeMarkdown` are exported/available in `internal/query` for plan 02-06 to wire into `internal/mcp/tools.go`'s 5 call sites (`json.Marshal(locs)`/`query.Marshal*JSON` → `query.Render*Markdown`) — this plan deliberately does not touch `internal/mcp/` at all.
- Exact markdown shape per renderer, for 02-06's MCP assertions to pin against:
  - `renderLocationTable`: `| Name | Kind | Location |` header, `|---|---|---|` separator, one `| \`name\` | kind | \`path:line\` |` row per record.
  - `RenderCallersMarkdown` / `RenderCalleesMarkdown`: `**Callers of \`Symbol\`** — N caller(s)` / `**Callees of \`Symbol\`** — N callee(s)` header, blank line, table; empty ⇒ `**Callers of \`Symbol\`** — no callers found.` (callees analogous).
  - `RenderImpactMarkdown`: `**Impact of \`Symbol\`** — depth D, N nodes, E edges` header, blank line, table; empty ⇒ same header + `— no affected symbols found.`
  - `RenderSearchMarkdown`: `**Search: \`term\`** — N result(s)` header, blank line, table; empty ⇒ `**Search: \`term\`** — no results found.`
  - `RenderFilesMarkdown` flat: `**Files** — N indexed` header, blank line, `| Path | Language | Nodes | Edges |` table; empty ⇒ `**Files** — no files found.` Tree: `**Files (tree)**` header, blank line, indented `name/` / `name (language)` list via `renderFileTreeMarkdown`.
- `internal/query/render_results.go`'s package comment documents the `Marshal*JSON`-CLI-only vs `Render*Markdown`-MCP-only asymmetry explicitly, so 02-06's executor (and any future reader) has the rationale in the code itself, not just this SUMMARY.
- No blockers. `go test ./...` is fully green; `go.mod`/`go.sum` are byte-identical to before this plan.

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*
