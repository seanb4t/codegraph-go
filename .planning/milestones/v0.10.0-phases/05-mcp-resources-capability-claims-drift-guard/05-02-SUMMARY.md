---
phase: 05-mcp-resources-capability-claims-drift-guard
plan: 02
subsystem: mcp
tags: [mcp, go-sdk, resources, wire-oracle, embed]

# Dependency graph
requires:
  - "05-01: resourcesFS / resourceURIFor / registerResources registration seam, resourceDescriptionFor drift-prevention pattern"
provides:
  - "The full 10-resource catalog: 8 per-tool fact-sheets (explore already existed; node/search/callers/callees/impact/files/status added here) plus 2 behavior docs (tools-filter, index-state)"
  - "resourceDescriptionFor extended to resolve any companionNames stem via companionTool(stem).Description, plus resourceDescriptionLiteralFor for the 2 behavior-doc stems"
  - "resources-list.golden re-frozen at full 10-URI catalog size, ready for plan 05-04's per-URI resources/read wire coverage"
affects: [05-03-claims-drift-guard, 05-04-wire-coverage-remaining]

# Actuals (#2632)
actuals:
  tokens: 4095
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "resourceDescriptionFor's default branch now checks resourceDescriptionLiteralFor first, then falls back to a companionNames membership scan calling companionTool(stem).Description — extends 05-01's single-case switch without introducing a second lookup mechanism"

key-files:
  created:
    - internal/mcp/resources/node.md
    - internal/mcp/resources/search.md
    - internal/mcp/resources/callers.md
    - internal/mcp/resources/callees.md
    - internal/mcp/resources/impact.md
    - internal/mcp/resources/files.md
    - internal/mcp/resources/status.md
    - internal/mcp/resources/tools-filter.md
    - internal/mcp/resources/index-state.md
  modified:
    - internal/mcp/resources.go
    - test/wireoracle/COVERAGE-BASELINE.md
    - testdata/wireoracle/transcripts/resources-list.golden

key-decisions:
  - "resourceDescriptionFor extended in Task 1 (not deferred to Task 3) to resolve companionNames stems generically via companionTool(stem).Description — the plan's Task 1 action text covered resourceURIFor entries but not this function; extending it was required to avoid registerResources panicking at server construction for node/search/callers/callees (and, by the same mechanism, impact/files/status in Task 2). Documented as a deviation below."
  - "tools-filter.md and index-state.md descriptions live in a new resourceDescriptionLiteralFor map, not inline switch cases, matching the plan's own phrase 'fill in resourceDescriptionFor's literal map' — reshapes the function's default branch into literal-map-then-companion-scan rather than a flat switch."
  - "COVERAGE-BASELINE.md's MCP Resources section and Last-updated line were updated in Task 3's commit (not explicitly named in the plan's action text, but required by the capture protocol's own rule 4: 'update this file's category tables ... in the same change') — the row describing resources-list's advertised set was stale after the re-freeze (still said 'advertises codegraph://tools/explore' when it now advertises 10 URIs)."

patterns-established:
  - "Fact-sheet content for a companion tool is entirely derivable from tools.go's Description/jsonschema tags with no new source of truth — Task 1/2 required zero deviation from the explore.md tracer format."

requirements-completed: [RSRC-01, RSRC-02]

coverage:
  - id: D1
    description: "resources/list advertises exactly 10 entries: 8 per-tool fact-sheets + codegraph://tools-filter + codegraph://index-state"
    requirement: RSRC-01
    verification:
      - kind: unit
        ref: "internal/mcp TestResourcesListAdvertisesRegisteredURIs, TestResourcesReadReturnsMarkdown (grown to 10 URIs with no test edit, driven off the live resources/list result)"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/resources-list — 10 URIs on the wire"
        status: pass
    human_judgment: false
  - id: D2
    description: "resources/read returns non-empty text/markdown for all 10 URIs"
    requirement: RSRC-02
    verification:
      - kind: unit
        ref: "internal/mcp TestResourcesReadReturnsMarkdown"
        status: pass
    human_judgment: false
  - id: D3
    description: "impact.md's two numeric claims (default 2, max 50) match numericClaimRe and equal internal/query/validate.go's live defaultDepth/MaxDepth constants"
    verification:
      - kind: unit
        ref: "internal/mcp TestMCPToolSchemaNumericClaimsMatchEngineConstants (unaffected — tools.go untouched); rg -o against impact.md cross-checked by hand against validate.go:22,49"
        status: pass
    human_judgment: false
  - id: D4
    description: "resources/list still holds all 10 URIs with zero .codegraph/ index present (RSRC-03 at full catalog size)"
    verification:
      - kind: unit
        ref: "internal/mcp TestResourcesRegisterWithoutIndex"
        status: pass
    human_judgment: false
  - id: D5
    description: "Exactly one wire-oracle golden transcript moved (resources-list.golden), one named cause, resources-read-explore.golden byte-identical"
    verification:
      - kind: integration
        ref: "test/wireoracle full suite (31 scenarios); git diff --name-only testdata/wireoracle/transcripts/ lists exactly resources-list.golden"
        status: pass
    human_judgment: false
  - id: D6
    description: "go test ./... -count=1 and go vet ./... clean across the repo (aside from a pre-existing, documented, load-dependent internal/daemon flake unrelated to this plan's files)"
    verification:
      - kind: integration
        ref: "go build ./..., go vet ./..., go test ./... -count=1, go test ./internal/daemon/... -count=1 (isolated re-run, clean)"
        status: pass
    human_judgment: false
duration: 25min
completed: 2026-08-12
status: complete
---

# Phase 5 Plan 2: MCP Resources — Remaining Docs Summary

**Fanned out from the proven tracer slice to the full 10-resource catalog — seven more per-tool fact-sheets (node/search/callers/callees/impact/files/status) plus two behavior docs (tools-filter, index-state) — and re-froze the `resources-list` wire transcript under one named cause.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3
- **Files modified:** 12 (9 created, 3 modified)

## Accomplishments
- `internal/mcp/resources/{node,search,callers,callees}.md`: fact-sheets for the 4 remaining zero-numeric-claim companion tools, each restating only `tools.go`'s own `Description`/jsonschema tags (D-01)
- `internal/mcp/resources/{impact,files,status}.md`: fact-sheets for the 3 remaining companion tools, including `impact.md`'s two numeric claims ("default 2", "max 50") pinned to `internal/query/validate.go`'s live `defaultDepth`/`MaxDepth` constants for plan 05-03's guard to match against `tools.go`'s own claims
- `internal/mcp/resources/{tools-filter,index-state}.md`: the two behavior docs — `CODEGRAPH_MCP_TOOLS`'s current narrowing trichotomy (unset/named-subset/empty, D-06's no-hedge stance) naming all 7 companion tools explicitly and both count-claim shapes ("7 companion tools", "8 tools"); and `.codegraph/`-presence gating, the `codegraph init` remedy, and live per-request re-check
- `internal/mcp/resources.go`: 9 new `resourceURIFor` entries (D-08/D-09/D-10), `resourceDescriptionFor` extended to resolve any `companionNames` stem generically via `companionTool(stem).Description`, plus a new `resourceDescriptionLiteralFor` map for the 2 behavior-doc stems
- Re-froze `testdata/wireoracle/transcripts/resources-list.golden`: exactly one named cause (the advertised resource set grew from the tracer's 1 entry to all 10), `resources-read-explore.golden` confirmed byte-identical, no other golden moved
- Updated `test/wireoracle/COVERAGE-BASELINE.md`'s MCP Resources section and Last-updated line to describe the full-catalog `resources-list` content

## Task Commits

1. **Task 1:** fact-sheets for node, search, callers, callees — `4926784` (feat)
2. **Task 2:** fact-sheets for impact, files, status — `a6a96ad` (feat)
3. **Task 3:** behavior docs and re-freeze resources-list transcript — `20093d3` (feat)

## Files Created/Modified
- `internal/mcp/resources/node.md` - `codegraph_node` fact-sheet
- `internal/mcp/resources/search.md` - `codegraph_search` fact-sheet
- `internal/mcp/resources/callers.md` - `codegraph_callers` fact-sheet
- `internal/mcp/resources/callees.md` - `codegraph_callees` fact-sheet
- `internal/mcp/resources/impact.md` - `codegraph_impact` fact-sheet, numeric claims pinned to `validate.go`
- `internal/mcp/resources/files.md` - `codegraph_files` fact-sheet
- `internal/mcp/resources/status.md` - `codegraph_status` fact-sheet
- `internal/mcp/resources/tools-filter.md` - `CODEGRAPH_MCP_TOOLS` behavior doc
- `internal/mcp/resources/index-state.md` - index-state preconditions behavior doc
- `internal/mcp/resources.go` - 9 new `resourceURIFor` entries, extended `resourceDescriptionFor`, new `resourceDescriptionLiteralFor` map
- `test/wireoracle/COVERAGE-BASELINE.md` - MCP Resources section + Last-updated line refreshed for full catalog
- `testdata/wireoracle/transcripts/resources-list.golden` - re-frozen, 10 URIs on the wire

## Decisions Made
- `resourceDescriptionFor` extended to a generic `companionNames` membership scan in Task 1, ahead of the plan's Task 3 note about the literal map — required so the 4 new files in Task 1 (and the 3 in Task 2) don't panic `registerResources` at server construction. See Deviations below.
- `tools-filter.md`/`index-state.md` Descriptions live in a new `resourceDescriptionLiteralFor` map rather than additional switch cases, directly matching the plan's own phrasing ("fill in `resourceDescriptionFor`'s literal map").
- `COVERAGE-BASELINE.md`'s MCP Resources section rewritten (not just the golden re-frozen) — the existing row's "advertises `codegraph://tools/explore`" text was stale the moment `resources-list.golden` moved to 10 entries; left uncorrected it would misdescribe the scenario for any future reader, including plan 05-04.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Extended `resourceDescriptionFor` beyond the plan's stated `resourceURIFor`-only edit**
- **Found during:** Task 1
- **Issue:** The plan's Task 1 `<action>` instructs adding four `resourceURIFor` entries but says nothing about `resourceDescriptionFor`. That function's `default` case (as 05-01 left it) panics on any stem other than `"explore"`. Since `registerResources` calls `resourceDescriptionFor(stem)` for every embedded file, adding `node.md` etc. without a matching description source would panic at server construction — failing the plan's own acceptance criterion ("`go run ./cmd/codegraph --help` ... does not panic").
- **Fix:** Extended `resourceDescriptionFor`'s default branch to scan `companionNames` and, on a match, return `companionTool(stem).Description` — the same drift-prevention shape (`GUARD-01`) 05-01 already established for `explore`. Applied once in Task 1, reused unchanged by Task 2 and Task 3's literal-map addition.
- **Files modified:** `internal/mcp/resources.go`
- **Commit:** `4926784`

**2. [Rule 2 - Missing critical functionality] Updated `COVERAGE-BASELINE.md`'s MCP Resources section**
- **Found during:** Task 3
- **Issue:** The plan's Task 3 `<action>` describes the re-freeze mechanics but does not explicitly mention editing `COVERAGE-BASELINE.md`. The capture protocol this repo's own `test/wireoracle/COVERAGE-BASELINE.md` documents (`<read_first>` for this task) states rule 4: any transcript-byte-moving change must "update this file's category tables and Total line in the same change." The existing `resources-list` row description was written for the 1-URI tracer state and would misdescribe the plan's actual 10-URI result if left unedited.
- **Fix:** Rewrote the MCP Resources section (row description, category intro paragraph, History bullet, Last-updated line) to describe the post-re-freeze state, without changing `ExpectedScenarioCount` (still 31 — no scenario was added).
- **Files modified:** `test/wireoracle/COVERAGE-BASELINE.md`
- **Commit:** `20093d3`

## Issues Encountered
`internal/daemon`'s `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` failed once during a full `go test ./... -count=1` run under this session's concurrent-workstation load — this is the pre-existing, documented, load-dependent "Daemon extreme-load tail (ACCEPTED, not a gap)" flake recorded in STATE.md, unrelated to any file this plan touches (`internal/mcp`, `test/wireoracle`, `testdata/wireoracle`). Confirmed passing in isolation (`go test ./internal/daemon/... -count=1`, clean, 64s).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 10 resources are registered, listed, and readable; `resources-list.golden` is frozen at full catalog size, ready for plan 05-04 to add per-URI `resources/read` wire coverage for the 9 URIs beyond `codegraph://tools/explore`.
- `resourceDescriptionFor` now derives every companion-tool description generically (not just `explore`'s hand-wired case) — plan 05-03's claims-drift guard extends onto a function that already fully implements GUARD-01's derive-don't-hand-type shape for all 8 tool stems and 2 behavior-doc stems.
- Every numeric claim this plan wrote (`impact.md`'s "default 2"/"max 50") is a live read of `internal/query/validate.go`, not a copied literal — 05-03's `engineConstantFor`-style guard has real drift to catch if either side ever moves.
- `tools-filter.md` deliberately carries no hedge about `CODEGRAPH_MCP_TOOLS`'s planned future removal (D-06) — plan 05-03's guard is the mechanism that will catch this doc going stale if/when that removal actually happens, per T-05-04's accepted disposition.

---
*Phase: 05-mcp-resources-capability-claims-drift-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk (`internal/mcp/resources/{node,search,callers,callees,impact,files,status,tools-filter,index-state}.md`, `internal/mcp/resources.go`, `test/wireoracle/COVERAGE-BASELINE.md`, `testdata/wireoracle/transcripts/resources-list.golden`). All three task commits confirmed present in `git log` (`4926784`, `a6a96ad`, `20093d3`).
