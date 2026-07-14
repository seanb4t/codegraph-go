---
phase: 03-query-engine-mcp-server
plan: 05
subsystem: api
tags: [go, pebble, query-engine, cli-parity, json]

requires:
  - phase: 03-query-engine-mcp-server
    provides: "03-02's Engine/OpenAt construction seam + IterateNodes/IterateFiles/GetMeta Reader extensions"
provides:
  - "Engine.Files (QRY-07) — browses the indexed file structure from the graph via IterateFiles(), never a filesystem scan, with pattern/filter/depth/format options"
  - "Engine.Status (QRY-09) — index health/counts/nodesByKind/languages, with a documented per-key TS-to-Go/Pebble remapping table"
  - "FilesOptions/FilesResult/FileEntry/FileTreeNode + StatusResult/PendingChanges/IndexHealth JSON shapes"
affects: [03-mcp-server, 03-cli-wiring, 03-parity-test]

tech-stack:
  added: []
  patterns:
    - "Colocated FilesOptions/StatusResult structs + JSON shaping directly in files.go/status.go (no shared render_json.go), matching search.go/traverse.go's Marshal* convention"
    - "validateFilesDepth rejects negative/absurd depth outright (0 = unlimited) rather than reusing clampDepth's '0 means small BFS-safe default' semantics — a browse command's default is 'show everything', not 'bound traversal cost'"

key-files:
  created:
    - internal/query/files.go
    - internal/query/status.go
    - internal/query/files_status_test.go
  modified: []

key-decisions:
  - "status.go's StatusResult doc comment is the authoritative per-key TS->Go remapping table resolving RESEARCH Open Question 2: backend=\"pebble\"; journalMode dropped entirely (no Pebble analog); version/builtWithExtractionVersion/currentExtractionVersion all derive from schema.SchemaVersion; reindexRecommended derives from schema.IsCurrentSchemaVersion(meta); pendingChanges/worktreeMismatch render present-but-inert (zero/null) per RESEARCH Assumption A2; projectPath/indexPath render as empty string since Engine carries no path context in its read-only design (engine.go was out of this plan's files_modified)"
  - "Files' Depth=0 means unlimited (not clampDepth's '0 -> defaultDepth=5' convention) — negative or >MaxDepth values are rejected with an error (T-03-05-DoS 'reject absurd depths, reuse validate helpers'), not silently clamped"
  - "Files' format=tree groups already-filtered/sorted FileEntry records into a nested FileTreeNode structure (directories synthesized on demand, sorted directories-first) — this shape is this plan's own design since files has no golden fixture (D-07a)"
  - "Status computes edgeCount from GetMeta().EdgeCount (stamped by the indexer at index time) rather than a second full IterateEdges(\"\") scan, since fileCount/nodeCount are already derived from scans Status needs anyway for nodesByKind/languages"

requirements-completed: [QRY-07, QRY-09]

coverage:
  - id: D1
    description: "Engine.Files browses the indexed file structure from the graph (IterateFiles only, never os.ReadDir) with pattern/filter/depth/format options"
    requirement: QRY-07
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFiles"
        status: pass
    human_judgment: false
  - id: D2
    description: "Engine.Status reports index health/counts/nodesByKind/languages with TS-only status.json keys remapped to Go/Pebble-truthful values per a documented per-key table"
    requirement: QRY-09
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestStatus"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 05: Files & Status Query Commands Summary

**Engine.Files browses the frozen graph's file structure (pattern/filter/depth/format, no filesystem scan) and Engine.Status reports index health with an explicit, documented TS-SQLite-to-Go/Pebble key remapping table resolving RESEARCH Open Question 2.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-11T14:10:00Z (approx)
- **Completed:** 2026-07-11T14:19:43Z
- **Tasks:** 2 completed (RED, GREEN)
- **Files modified:** 3 (all created)

## Accomplishments

- `Engine.Files` (QRY-07) reads exclusively from `reader.IterateFiles()` — proven filesystem-independent by a test that deletes a file from disk mid-test and confirms it still appears in the result — with `Pattern` (glob), `Filter` (language), `Depth` (directory nesting), and `Format` (`flat`/`tree`) options.
- `Engine.Status` (QRY-09) reports `initialized`/`fileCount`/`nodeCount`/`edgeCount`/`nodesByKind`/`languages` from `IterateFiles`/`IterateNodes` scans plus `GetMeta()`, and renders the golden `status.json`'s TS-SQLite-only fields (`backend`, `journalMode`, `builtWithExtractionVersion`, `currentExtractionVersion`, `pendingChanges`, `worktreeMismatch`) via an explicit, doc-commented per-key mapping table on `StatusResult`.
- Resolved RESEARCH Open Question 2 (the concrete TS-key-to-Go-analog table) as a concrete, tested decision rather than leaving it to future improvisation.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): TestFiles/TestStatus pin browse options + the status key-mapping table** - `d55eead` (test)
2. **Task 2 (GREEN): Implement Engine.Files + Engine.Status with the D-05 status remapping** - `2c8866a` (feat)

_No REFACTOR commit — GREEN passed cleanly on first implementation, no cleanup needed beyond `gofmt -w`._

## Files Created/Modified

- `internal/query/files.go` - `Engine.Files`, `FilesOptions`/`FilesResult`/`FileEntry`/`FileTreeNode`, `validateFilesDepth`, `buildFileTree`/`sortFileTree`, `MarshalFilesJSON`
- `internal/query/status.go` - `Engine.Status`, `StatusResult`/`PendingChanges`/`IndexHealth` with the authoritative TS-to-Go remapping doc comment, `MarshalStatusJSON`
- `internal/query/files_status_test.go` - `TestFiles`/`TestStatus`, reusing `copyFixture`/`indexFixture` from `engine_test.go` at runtime (not modified)

## Decisions Made

- **Depth=0 means unlimited for `files`**, deliberately diverging from `clampDepth`'s "0 → small BFS-safe default" convention used by `impact`/`callers`/`callees`: a browse command's useful default is "show everything," and the DoS mitigation is instead a reject-on-absurd-input check (negative or `>MaxDepth`) plus a `MaxLimit` cap on the returned set — not a traversal-cost bound, since `files` never traverses, it filters an already-enumerated set.
- **`journalMode` is dropped entirely** (key omitted) rather than rendered as an empty placeholder — Pebble has no user-facing WAL/journal-mode concept, so there is nothing meaningful to remap it to, per D-05's "drop... keys that have no Go analog" language.
- **`projectPath`/`indexPath` render as empty strings.** `Engine` carries no path context (its construction seam in `engine.go` was intentionally out of this plan's `files_modified` — Wave-3 isolation), so there is no host path to leak in the first place; the keys stay present for shape parity with an inert value, trivially satisfying T-03-05-Leak.
- **`edgeCount` sourced from `GetMeta().EdgeCount`**, not a second full edge scan — the indexer already stamps this count at index time (`internal/indexer/resolve.go`), and `Status` already performs full `IterateFiles`/`IterateNodes` scans for the counts `nodesByKind`/`languages` require, so reusing `Meta` avoids a redundant `IterateEdges("")` pass.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `gofmt -w` auto-aligned `status.go`'s struct tags after the initial write (cosmetic only, no logic change) before the GREEN commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- QRY-07 and QRY-09 are both complete; `Engine.Files`/`Engine.Status` are ready for CLI wiring (`internal/cli`'s `files`/`status` commands) and MCP tool registration in a later Wave-3/Wave-4 plan, per D-08b's "one engine, two front-ends" model.
- The `status.go` remapping table is the concrete artifact a later parity-test plan (MCP-04, diffing against `testdata/golden/corpus/weft-go/status.json`) should consume directly — no further TS-key-mapping research needed.
- No blockers for downstream Phase-3 plans.

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

- FOUND: internal/query/files.go
- FOUND: internal/query/status.go
- FOUND: internal/query/files_status_test.go
- FOUND: commit d55eead (test(03-05): RED)
- FOUND: commit 2c8866a (feat(03-05): GREEN)
