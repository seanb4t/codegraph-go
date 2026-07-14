---
phase: 03-query-engine-mcp-server
plan: 06
subsystem: query-engine
tags: [go, markdown-rendering, mcp-parity, golden-corpus, path-traversal-defense]

# Dependency graph
requires:
  - phase: 03-query-engine-mcp-server
    provides: "03-03's lexical matcher (matchNodes/Query/Search) and 03-04's reverse-adjacency builder (buildReverseAdjacency) + resolveSymbolNode + isTestSymbol"
provides:
  - "Engine.Node(symbol, file) — symbol detail markdown (Location/Signature/Trail/Calls →/Called by ←) or a line-numbered verbatim file read"
  - "Engine.Explore(query, maxFiles) — one-round-trip verbatim source + blast radius + call context, byte-shape matched to the golden explore.json"
  - "RenderNode/RenderExplore formatters in render_markdown.go, reused verbatim by the future MCP server (D-08b)"
  - "Engine.repoRoot + NewWithRoot — the confinement root every on-disk source read (Node file-mode, Explore) resolves against"
affects: [03-07-mcp-server, 03-08-golden-parity-test]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Path-traversal defense: resolveSourcePath (node.go) rejects absolute paths and any Clean/Rel-detected '..' escape before os.ReadFile ever runs"
    - "'0 means a small default, not unlimited' clamp convention (clampMaxFiles) extended to a third flag alongside clampDepth"
    - "Markdown formatters (render_markdown.go) take fully-resolved data (nodes, adjacency, raw bytes) and do no I/O themselves — Engine methods own all Reader/filesystem access"

key-files:
  created:
    - internal/query/render_markdown.go
    - internal/query/node.go
    - internal/query/explore.go
    - internal/query/render_markdown_test.go
  modified:
    - internal/query/engine.go
    - internal/query/validate.go

key-decisions:
  - "Added Engine.repoRoot + NewWithRoot (engine.go) — not in the plan's files_modified list, but required: Node (file mode) and Explore need an absolute confinement root for os.ReadFile, and Engine previously carried none. New() is untouched for existing Reader-only callers (repoRoot defaults to empty and disk-read paths return a clear error)."
  - "Added defaultMaxFiles/clampMaxFiles (validate.go) mirroring clampDepth's '0 means default' shape — Explore's per-file verbatim-source read is expensive enough that unlimited-by-default would be a resource-exhaustion footgun, unlike Files' '0 means unlimited' browse-only convention."
  - "Blast-radius bullet's 'N callers in `path`' groups by the matched symbol's OWN file (where it's defined), not the callers' files — confirmed against explore.json's arithmetic: mergeStyle shows '3 callers in `internal/cli/finish.go`' (finish.go is where mergeStyle is DEFINED) with 'tests: `internal/cli/finish_test.go`' as the distinct test-caller-file breakdown, matching node.json's 3 total callers (1 prod + 2 test, both test callers sharing one file)."
  - "Explore's file-grouping skips matches with an empty FilePath (the synthetic 'package' pseudo-node kind from internal/indexer/resolve.go) — there is no on-disk file to read verbatim source from for those."
  - "Node's file-disambiguation (-f flag) reuses traverse.go's resolveSymbolNode when no file is given, and only implements a separate name+file scan when disambiguation is actually requested — avoids duplicating the existing resolver for the common case."

requirements-completed: [QRY-02, QRY-08]

coverage:
  - id: D1
    description: "node <symbol> renders the golden node.json markdown shape (name/kind, Location, Signature, Trail, Calls →, Called by ←)"
    requirement: "QRY-02"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestNode/symbol_mode_renders_the_fixed_node.json_section_order"
        status: pass
    human_judgment: false
  - id: D2
    description: "node <file> renders a line-numbered, tab-indented verbatim read of the file, confined to the repo root"
    requirement: "QRY-02"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestNode/file_mode_renders_a_line-numbered,_tab-indented_verbatim_read"
        status: pass
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestNode/path_escaping_the_repo_root_is_rejected"
        status: pass
    human_judgment: false
  - id: D3
    description: "explore <query> renders the golden explore.json shape: header, blast-radius bullets, the verbatim disclaimer, per-file line-numbered source read fresh from disk, capped at max-files"
    requirement: "QRY-08"
    verification:
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestExplore/renders_the_fixed_explore.json_section_order_for_a_single-file_match"
        status: pass
      - kind: unit
        ref: "internal/query/render_markdown_test.go#TestExplore/max-files_caps_the_number_of_rendered_file_blocks"
        status: pass
    human_judgment: false

duration: 18min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 06: Node/Explore Markdown Rendering Summary

**`Engine.Node`/`Engine.Explore` reproduce the golden node.json/explore.json markdown templates byte-for-byte, reading source fresh from disk with path-traversal-safe confinement to the repo root**

## Performance

- **Duration:** 18 min
- **Started:** 2026-07-11T10:24:31-04:00
- **Completed:** 2026-07-11T10:42:17-04:00
- **Tasks:** 2 (RED, GREEN)
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments
- `Engine.Node(symbol, file)` renders symbol detail (`**name** (kind)` / Location / Signature / Trail / `Calls →` / `Called by ←`) by reusing 03-04's `resolveSymbolNode` and `buildReverseAdjacency` — no duplicated traversal logic — or, when `symbol` is empty and `file` is given, a line-numbered verbatim file read.
- `Engine.Explore(query, maxFiles)` selects symbols via 03-03's lexical matcher, groups them by file (capped at `maxFiles` distinct files, `0` defaulting to 5 via the new `clampMaxFiles`), computes each symbol's blast radius from the shared reverse-adjacency map, and renders each selected file's verbatim source read fresh from disk on every call.
- The verbatim-source disclaimer paragraph is copied byte-for-byte from `explore.json` into `render_markdown.go`'s `sourceDisclaimer` const; `TestExplore` extracts the disclaimer from both the golden fixture and the live render and asserts byte-equality, so any future paraphrase or reordering fails the test (D-05a, T-03-06-Drift).
- All on-disk reads (`Node` file mode, `Explore`) go through `resolveSourcePath`, which rejects absolute paths and any `..`-escape via `filepath.Clean` + a `filepath.Rel`-based confinement check before `os.ReadFile` ever runs (T-03-06-Path).

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): TestNode/TestExplore pin the golden markdown templates** - `6d1efd3` (test)
2. **Task 2 (GREEN): Implement RenderNode/RenderExplore + Engine.Node/Explore** - `47ba35f` (feat)

**Plan metadata:** (pending — this commit)

## Files Created/Modified
- `internal/query/render_markdown.go` - `RenderNode`/`RenderExplore` formatters, `renderNumberedSource` line-numbering helper, the verbatim `sourceDisclaimer` const, `pluralize`/`formatNodeRef`/`joinNodeRefs`/`renderBlastBullet`/`joinSymbolKindList` helpers
- `internal/query/node.go` - `Engine.Node`, `resolveNodeForDetail` (name+file disambiguation), `resolveSourcePath`/`readSourceFile` (shared path-safety gate)
- `internal/query/explore.go` - `Engine.Explore`, `exploreFileGroup`/`exploreBlast` result types, `groupMatchesByFile`, `buildBlastEntry`
- `internal/query/render_markdown_test.go` - `TestNode`/`TestExplore` byte-shape assertions against the golden fixtures, plus max-files-cap and path-escape-rejection cases
- `internal/query/engine.go` (modified) - added `Engine.repoRoot` field and `NewWithRoot` constructor; `OpenAt` now threads the resolved repo root through
- `internal/query/validate.go` (modified) - added `defaultMaxFiles`/`clampMaxFiles`

## Decisions Made
- Extended `Engine` with a `repoRoot` field via a new `NewWithRoot` constructor rather than changing `New`'s signature, so every existing `New(fakeReader)` call site in `search_test.go`/`traverse_test.go` continues to compile unchanged.
- Interpreted the golden's "N callers in `path`" blast-radius phrasing as counting callers of a symbol grouped under the symbol's OWN defining file (not the callers' files) — verified against `node.json`'s exact caller list for `mergeStyle` (3 total: 1 production + 2 test, both tests sharing one file), which matches `explore.json`'s "3 callers ... ; tests: `internal/cli/finish_test.go`" bullet exactly.
- Node/Explore's fenced source blocks are hardcoded to the ``` ```go ``` `` language tag, matching this phase's Go-only extraction scope (D-05, "languages reflect Go-only extraction until Phase 5").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2/3 - Missing Critical / Blocking] Added Engine.repoRoot + NewWithRoot**
- **Found during:** Task 2 (implementing Node file-mode and Explore)
- **Issue:** The plan's method signatures (`Node(symbol, file string)`, `Explore(query string, maxFiles int)`) take no repo-root parameter, and `Engine` (engine.go, excluded from this plan's `files_modified`) carried only a `graphstore.Reader` — with no way to safely resolve relative `FilePath`s to absolute disk paths for `os.ReadFile`, the feature could not be implemented at all.
- **Fix:** Added an unexported `repoRoot string` field to `Engine`, a new `NewWithRoot(r, repoRoot)` constructor, and updated `OpenAt` to call `NewWithRoot(reader, dir)` using the directory `ResolveCodegraphDir` already resolves (the repo root, per resolve.go's existing walk-up logic). `New(r)` is unchanged — an `Engine` built via `New` has an empty `repoRoot` and its disk-read paths return a clear error rather than reading from an unintended location.
- **Files modified:** internal/query/engine.go
- **Verification:** `go build ./...` and `go test ./internal/query/... -count=1` both green; `search_test.go`/`traverse_test.go`'s existing `New(&searchFakeReader{...})` call sites compile unchanged.
- **Committed in:** 47ba35f (Task 2 commit)

**2. [Rule 2 - Missing Critical] Added defaultMaxFiles/clampMaxFiles**
- **Found during:** Task 2 (implementing Explore)
- **Issue:** `Explore`'s `maxFiles` parameter needed a "0 means a sane default" clamp (matching `clampDepth`'s established convention) so an unset flag doesn't default to "unlimited" — unlike `Files`' browse-only `Depth=0` convention, Explore's per-file work includes an actual disk read of verbatim source, so unlimited-by-default is a real resource-exhaustion risk, not just a UX nicety.
- **Fix:** Added `defaultMaxFiles = 5` and `clampMaxFiles(n int) int` to validate.go, mirroring `clampDepth`'s shape exactly; `validateMaxFiles` still runs first so an explicit out-of-range request is rejected rather than silently clamped.
- **Files modified:** internal/query/validate.go
- **Verification:** `TestExplore/max-files_caps_the_number_of_rendered_file_blocks` passes.
- **Committed in:** 47ba35f (Task 2 commit)

**3. [Rule 1 - Bug] Excluded FilePath-less matches from Explore's file grouping**
- **Found during:** Task 2, while running the newly-written max-files test (`Explore("e", 1)`)
- **Issue:** `Explore("e", 1)` failed with "query: empty file path" — the lexical matcher (03-03) also matches the synthetic `package` pseudo-node kind (internal/indexer/resolve.go), which carries no `FilePath`, so `readSourceFile("")` was rejected by the new path-safety gate.
- **Fix:** `groupMatchesByFile` now skips any ranked match whose `FilePath` is empty before file-grouping — there is no on-disk file to read verbatim source from for a synthetic package node.
- **Files modified:** internal/query/explore.go
- **Verification:** `TestExplore` passes; full `go test ./internal/query/... -count=1` green.
- **Committed in:** 47ba35f (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (2 missing-critical/blocking, 1 bug)
**Impact on plan:** All three were required for the feature to compile/work correctly at all, or were caught by the plan's own test-writing discipline. No scope creep — engine.go/validate.go changes are minimal, additive, and non-architectural.

## Issues Encountered
None beyond the deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/query` now implements every read-only query verb this phase's engine needs (`Query`/`Search`/`Callers`/`Callees`/`Impact`/`Affected`/`Files`/`Status`/`Node`/`Explore`) plus their formatters — the MCP server plan (03-07) can wire tool handlers directly onto these Engine methods per D-08b (one engine, two front-ends), with no new engine logic required.
- `RenderNode`/`RenderExplore` are ready to be exercised by the future golden-parity test against `testdata/golden/corpus/weft-go/{node,explore}.json` for full end-to-end shape verification (this plan's tests assert fixed-region byte-shape against our own fixture plus disclaimer byte-equality against the golden, but do not yet run the CLI/MCP surface against the weft-go corpus itself — that's the parity-test plan's job).
- No blockers.

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created/modified files confirmed present; both task commits (6d1efd3, 47ba35f) confirmed in git log.
