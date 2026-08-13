---
phase: 05-mcp-resources-capability-claims-drift-guard
plan: 01
subsystem: mcp
tags: [mcp, go-sdk, resources, wire-oracle, tdd, embed]

# Dependency graph
requires: []
provides:
  - "internal/mcp.resourcesFS / resourceURIFor / registerResources — the one embedded-resource registration seam plans 05-02/05-03/05-04 extend"
  - "capabilities.resources live on the wire, unconditionally, with cacheScope: private on resources/list and resources/read"
  - "test/wireoracle resourcesListRequest/resourceReadRequest helpers + assertResourceCacheControl anchor, ready for plans 05-02/05-04's additional resource scenarios"
affects: [05-02-mcp-resources-remaining-docs, 05-03-claims-drift-guard, 05-04-wire-coverage-remaining]

# Actuals (#2632)
actuals:
  tokens: 33550
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "First go:embed use in this repository (internal/mcp/resources.go), one embed.FS + one filename->URI map + one registration loop"
    - "Fatal-at-construction panic convention (mirrors companionTool/companionHandler) applied to registerResources for any build-time-invariant violation"

key-files:
  created:
    - internal/mcp/resources.go
    - internal/mcp/resources/explore.md
    - internal/mcp/resources_test.go
    - testdata/wireoracle/transcripts/resources-list.golden
    - testdata/wireoracle/transcripts/resources-read-explore.golden
  modified:
    - internal/mcp/server.go
    - test/wireoracle/scenarios.go
    - test/wireoracle/anchors.go
    - test/wireoracle/COVERAGE-BASELINE.md
    - testdata/wireoracle/transcripts/*.golden (25 re-frozen, one named cause)

key-decisions:
  - "cacheScope: private for resources/list and resources/read (05-RESEARCH.md Open Question 1 resolved) — extends the tools/list and server/discover corrections already in the middleware, kept consistent with STATE.md's ttlMs:0/cacheScope:private pairing"
  - "resourceDescriptionFor derives every resource's Description from the tool's own exploreTool()/companionTool() Description, never a hand-typed copy — GUARD-01's drift-prevention shape applied from the start"

patterns-established:
  - "registerResources: one embed.FS, one URI map, one loop, called unconditionally in BuildServer before the `if hasIndex` branch — the structural template plans 05-02/05-04 extend to the remaining 9 resources"

requirements-completed: [RSRC-01, RSRC-02, RSRC-03]

coverage:
  - id: D1
    description: "A live client against a real serve --mcp subprocess sees capabilities.resources in the initialize result"
    requirement: RSRC-01
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/resources-list"
        status: pass
      - kind: unit
        ref: "internal/mcp TestResourcesListAdvertisesRegisteredURIs"
        status: pass
    human_judgment: false
  - id: D2
    description: "resources/list on the wire advertises codegraph://tools/explore with mimeType text/markdown"
    requirement: RSRC-01
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/resources-list (rg '\"uri\":\"codegraph://tools/explore\"' and '\"mimeType\":\"text/markdown\"')"
        status: pass
    human_judgment: false
  - id: D3
    description: "resources/read on codegraph://tools/explore returns non-empty text/markdown on the wire"
    requirement: RSRC-02
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/resources-read-explore"
        status: pass
      - kind: unit
        ref: "internal/mcp TestResourcesReadReturnsMarkdown"
        status: pass
    human_judgment: false
  - id: D4
    description: "BuildServer(false, ...) — no index — still lists and serves codegraph://tools/explore, proven as behavior"
    requirement: RSRC-03
    verification:
      - kind: unit
        ref: "internal/mcp TestResourcesRegisterWithoutIndex"
        status: pass
    human_judgment: false
  - id: D5
    description: "cacheScope: private is anchored on both new resource wire scenarios"
    verification:
      - kind: integration
        ref: "test/wireoracle TestSpecAnchorsHold/resources-list and /resources-read-explore (assertResourceCacheControl)"
        status: pass
    human_judgment: false
  - id: D6
    description: "task test:wireoracle is green with every moved transcript byte attributed to one named cause"
    verification:
      - kind: integration
        ref: "test/wireoracle full suite (31 scenarios/transcripts); git diff -U0 testdata/wireoracle/transcripts/ shows 50 changed lines, all one substitution (capabilities.resources appearing before tools)"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-08-12
status: complete
---

# Phase 5 Plan 1: MCP Resources Tracer Slice Summary

**Registered `codegraph://tools/explore` as a live MCP Resource end-to-end — embedded markdown, `go-sdk` capability, `cacheScope: private`, unit tests, and a re-frozen 31-scenario wire-oracle corpus — proving the whole Resources architecture on one URI before horizontal expansion.**

## Performance

- **Duration:** ~20 min (13:36–13:56 UTC-4)
- **Started:** 2026-08-12T13:36:20-04:00
- **Completed:** 2026-08-12T13:55:53-04:00
- **Tasks:** 3
- **Files modified:** 34

## Accomplishments
- `internal/mcp/resources.go` + `internal/mcp/resources/explore.md`: first `go:embed` use in this repo, `resourceURIFor` map, `resourceDescriptionFor` (derives from `exploreTool()`, never hand-typed), `registerResources` (one loop, fatal-at-construction panics)
- `internal/mcp/server.go`: `Capabilities.Resources = &mcp.ResourceCapabilities{}` set explicitly (D-11 extension); `registerResources(s)` called unconditionally before `if hasIndex` (RSRC-03's structural property); `resources/list`/`resources/read` added to the cacheScope-correction middleware, set to `private`
- Four new `internal/mcp` unit tests (`TestResourcesListAdvertisesRegisteredURIs`, `TestResourcesReadReturnsMarkdown`, `TestResourcesRegisterWithoutIndex`, `TestResourcesReadIsNotVacuous`) via strict RED→GREEN TDD
- 25 of 29 existing wire-oracle transcripts re-frozen under one named cause (`capabilities.resources` now precedes `tools` in every `initialize`/`server/discover` result, per Go's declaration-order JSON marshaling); the 4 scenarios with no `capabilities` object came back byte-identical, as predicted
- Two new wire scenarios (`resources-list`, `resources-read-explore`) with a new `assertResourceCacheControl` anchor pinning `cacheScope: private` on the wire; `ExpectedScenarioCount` 29 → 31; `COVERAGE-BASELINE.md` updated with a new category table

## Task Commits

Each task was committed atomically (Task 1 split into RED/GREEN per its `tdd="true"` frontmatter):

1. **Task 1 (RED):** add failing tests for MCP Resources capability — `34f5036` (test)
2. **Task 1 (GREEN):** register `codegraph://tools/explore` as an MCP Resource — `4a681ae` (feat)
3. **Task 2:** re-freeze wire-oracle transcripts for `capabilities.resources` — `cf1293b` (test)
4. **Task 3:** wire-observe the tracer URI on `resources/list`/`resources/read` — `502eb65` (feat)

## Files Created/Modified
- `internal/mcp/resources.go` - `resourcesFS`, `resourceURIFor`, `resourceDescriptionFor`, `registerResources`
- `internal/mcp/resources/explore.md` - the `codegraph_explore` fact-sheet
- `internal/mcp/resources_test.go` - list/read/no-index/non-vacuity coverage
- `internal/mcp/server.go` - explicit `Capabilities.Resources`, unconditional `registerResources(s)` call, `resources/list`/`resources/read` cacheScope correction
- `test/wireoracle/scenarios.go` - `resourcesListRequest`/`resourceReadRequest` helpers, two new scenarios, `ExpectedScenarioCount` 29→31
- `test/wireoracle/anchors.go` - `assertResourceCacheControl`, two new anchor registrations
- `test/wireoracle/COVERAGE-BASELINE.md` - header count, new category table, History bullet, Total line
- `testdata/wireoracle/transcripts/*.golden` - 25 re-frozen (one named cause), 2 new (`resources-list`, `resources-read-explore`)

## Decisions Made
- `cacheScope: private` for `resources/list`/`resources/read` (05-RESEARCH.md Open Question 1) — chosen over the SDK's `public` default for wire uniformity, consistency with the standing `ttlMs: 0`/`cacheScope: private` pairing, and fail-safe posture for future repository-dependent resource content.
- `resourceDescriptionFor` derives every resource's `Description` from the corresponding tool's own `Description` (`exploreTool()`/`companionTool()`), never a hand-typed copy — applying GUARD-01's drift-prevention shape starting with this tracer rather than retrofitting it in plan 05-03.

## Deviations from Plan

None — plan executed exactly as written. One clarifying note (not a deviation): the plan's Task 1 `<action>` describes `resourceDescriptionFor` as "returns a value from a small explicit literal map for the two behavior-doc stems `tools-filter` and `index-state`" — those two stems don't exist yet in this plan (they're added in 05-02), so `resourceDescriptionFor`'s `default` branch currently panics on any stem other than `"explore"`; the literal-map branch is deferred to 05-02 when those files exist, exactly as the plan's own "Created by LATER plans in this phase" table anticipates.

## Issues Encountered
None. The `internal/daemon` package failed once under this session's concurrent-workstation load during `go test ./... -count=1` — this is the pre-existing, documented, load-dependent "Daemon extreme-load tail (ACCEPTED, not a gap)" flake recorded in STATE.md, unrelated to any file this plan touches (`internal/mcp`, `test/wireoracle`, `testdata/wireoracle`). Confirmed passing in isolation twice (`go test ./internal/daemon/... -count=1`, both runs clean).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The registration template (`resourcesFS` / `resourceURIFor` / `resourceDescriptionFor` / `registerResources`) is proven end-to-end and ready for plan 05-02 to extend with the remaining 9 resource files (8 tool fact-sheets total, 2 behavior docs).
- `resourceDescriptionFor`'s literal-map branch for `tools-filter`/`index-state` is deferred to 05-02, as anticipated by this plan's own "Created by LATER plans" note.
- The wire-oracle corpus is green at 31 scenarios/transcripts with no re-freeze outstanding — plan 05-04 can extend the `resources-*` scenario set without inheriting any debt from this plan.
- GUARD-01/GUARD-02 (the claims drift guard proper) are plan 05-03's scope, not built here; this plan's `resourceDescriptionFor` already derives from source rather than hand-typing, which keeps 05-03's extension additive rather than corrective.

---
*Phase: 05-mcp-resources-capability-claims-drift-guard*
*Completed: 2026-08-12*

## Self-Check: PASSED

All created files confirmed present on disk (`internal/mcp/resources.go`, `internal/mcp/resources/explore.md`, `internal/mcp/resources_test.go`, `testdata/wireoracle/transcripts/resources-list.golden`, `testdata/wireoracle/transcripts/resources-read-explore.golden`). All four task commits confirmed present in `git log` (`34f5036`, `4a681ae`, `cf1293b`, `502eb65`).
