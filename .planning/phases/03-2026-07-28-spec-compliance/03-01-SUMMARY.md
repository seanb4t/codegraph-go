---
phase: 03-2026-07-28-spec-compliance
plan: 01
subsystem: mcp
tags: [mcp, go-sdk, wire-oracle, server-discover, cacheScope, sep-2575]

# Dependency graph
requires:
  - phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
    provides: "go-sdk migration, the AddReceivingMiddleware seam, D-09's tools/list cacheScope correction pattern"
provides:
  - "server/discover's CacheScope corrected from go-sdk's default \"public\" to \"private\" (SPEC-04)"
  - "modern-discover-explore: the first frozen transcript proving SPEC-01/03/04/07(discover half)/08 via a real Modern (2026-07-28) sessionless server/discover + tools/call"
  - "a hand-authored discover cache-control anchor (assertDiscoverCacheControl), independent of the frozen bytes"
  - "ExpectedScenarioCount at 24, the base every other Phase 3 plan's scenario additions build on"
affects: [03-02, 03-03, 03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 3401
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Modern (2026-07-28) sessionless dispatch in wire-oracle scenarios: discoverRequest/modernToolCallRequest carry the SEP-2575 _meta object (io.modelcontextprotocol/protocolVersion, clientInfo, clientCapabilities), never params.protocolVersion"
    - "A per-scenario hand-authored Anchor (assertDiscoverCacheControl) proves a spec-pinned property against a FRESH capture, independent of and alongside the frozen-bytes comparison — a wholesale transcript regeneration cannot silently move both"

key-files:
  created:
    - testdata/wireoracle/transcripts/modern-discover-explore.golden
  modified:
    - internal/mcp/server.go
    - test/wireoracle/scenarios.go
    - test/wireoracle/anchors.go

key-decisions:
  - "One scenario (modern-discover-explore), not two: the discover response and the following sessionless tools/call are captured in a single session, since together they are the minimal proof of SPEC-01/03/04/08 for both a discover result and a tool result"
  - "NoInitialize: true on modern-discover-explore is semantically distinct from edge-call-before-initialize's NoInitialize: true — the former is spec-sanctioned SEP-2575 sessionless dispatch (server accepts it), the latter is a session-ordering violation the server rejects. Documented explicitly in the scenario's doc comment to prevent future readers conflating the two."

patterns-established:
  - "Modern-vs-Legacy protocol signaling lives entirely in _meta, never in params.protocolVersion — codified as modernMetaParams() so no future scenario re-derives this from scratch or falls into the -32601 trap D-01 documented"

requirements-completed: [SPEC-01, SPEC-03, SPEC-04, SPEC-08]

coverage:
  - id: D1
    description: "server/discover's CacheScope corrected to \"private\" (SPEC-04's remaining discover half — ttlMs was already 0 and correct)"
    requirement: "SPEC-04"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-discover-explore"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestSpecAnchorsHold/modern-discover-explore (assertDiscoverCacheControl)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A Modern client's server/discover response (no prior tool call, no prior initialize) carries capabilities, resultType \"complete\", and serverInfo in _meta (SPEC-01/03/08)"
    requirement: "SPEC-01"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-discover-explore"
        status: pass
    human_judgment: false
  - id: D3
    description: "A Modern sessionless tools/call result also carries resultType \"complete\" and serverInfo in _meta, proving SPEC-03/08 hold for a tool result, not only a discover result"
    requirement: "SPEC-03"
    verification:
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/modern-discover-explore"
        status: pass
    human_judgment: false
  - id: D4
    description: "ExpectedScenarioCount moved 23 -> 24 in the same commit as the new scenario and its transcript"
    verification:
      - kind: unit
        ref: "test/wireoracle TestScenarioCountIsExact"
        status: pass
      - kind: unit
        ref: "test/wireoracle TestTranscriptSetMatchesScenarioSet"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-08-06
status: complete
---

# Phase 3 Plan 1: Modern Discover Tracer Summary

**SPEC-04's discover cacheScope corrected to "private" via a one-line middleware branch (mirroring D-09), frozen as `modern-discover-explore` — the first wire-oracle transcript proving a Modern `2026-07-28` sessionless `server/discover` + `tools/call` end-to-end — plus an independent hand-authored anchor demonstrated RED against a confirmed mutation.**

## Performance

- **Duration:** ~6 min (commit-to-commit; investigation/reading time not separately tracked)
- **Started:** 2026-08-06T11:23:28-04:00 (first task commit)
- **Completed:** 2026-08-06T11:28:44-04:00 (final task commit)
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Added `case "server/discover":` to `BuildServer`'s `AddReceivingMiddleware` switch in `internal/mcp/server.go`, correcting `DiscoverResult.CacheScope` from go-sdk's default `"public"` to `"private"` — the exact defect D-09 fixed for `tools/list` in Phase 2, in the one response path Phase 2 never touched. `TTLMs` was left untouched (already 0, already correct).
- Added `modernProtocolVersion`, `modernMetaParams()`, `discoverRequest()`, and `modernToolCallRequest()` to `test/wireoracle/scenarios.go` — hand-authored SEP-2575 `_meta` helpers carrying the protocol version in `_meta`, never `params.protocolVersion` (the exact trap 03-CONTEXT.md D-01 documented, which cost two wrong probe conclusions during this phase's research).
- Added the `modern-discover-explore` scenario: a sessionless `server/discover` (id 1) followed by a sessionless `tools/call` for `codegraph_explore` (id 2), both carrying Modern `_meta`. Froze it via the oracle's own capture CLI (`test/wireoracle/cmd/wireoracle`) against a freshly rebuilt `bin/codegraph` — never hand-written.
- Moved `ExpectedScenarioCount` from 23 to 24 in the same commit as the new scenario and transcript, extending its doc comment's arithmetic.
- Added `assertDiscoverCacheControl` to `test/wireoracle/anchors.go` — a hand-authored anchor (never an SDK constant) decoding only `result.cacheScope`/`result.ttlMs` from a fresh capture, registered against `modern-discover-explore`.
- Demonstrated the anchor RED per the standing repo rule: mutated `CacheScope = "private"` to `"public"` in `internal/mcp/server.go`, rebuilt, confirmed both `TestSpecAnchorsHold/modern-discover-explore` and `TestFrozenTranscriptsMatch/modern-discover-explore` FAIL, reverted, confirmed `git diff -- internal/mcp/server.go` empty, rebuilt, confirmed the full suite green.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end Modern discover — the one-line seam extension** - `252d11b` (feat)
2. **Task 2: The tracer scenario — one Modern session, discover then call, frozen** - `360c240` (test)
3. **Task 3: A hand-authored cache-control anchor, demonstrated RED** - `0c45d6d` (test)

_No plan-metadata commit yet — this SUMMARY plus STATE.md/ROADMAP.md/REQUIREMENTS.md updates land in the final `docs(03-01)` commit below._

## Files Created/Modified

- `internal/mcp/server.go` - added the `server/discover` middleware branch correcting `CacheScope` to `"private"`
- `test/wireoracle/scenarios.go` - added `modernProtocolVersion`, `modernMetaParams`, `discoverRequest`, `modernToolCallRequest`, the `modern-discover-explore` scenario, and bumped `ExpectedScenarioCount` to 24
- `test/wireoracle/anchors.go` - added `assertDiscoverCacheControl` and its `Anchor` registration for `modern-discover-explore`
- `testdata/wireoracle/transcripts/modern-discover-explore.golden` - new frozen transcript (created, never hand-written — captured via `test/wireoracle/cmd/wireoracle`)

## Decisions Made

- Kept the discover-success-plus-tool-call proof as a single scenario rather than two, per the plan's explicit instruction — the two requests together are the minimal proof of SPEC-01/03/04/08 for both a discover result and a tool result.
- Documented in the scenario's doc comment (not just in this summary) that `NoInitialize: true` on `modern-discover-explore` means something structurally different from `edge-call-before-initialize`'s `NoInitialize: true`: the former is spec-sanctioned SEP-2575 sessionless dispatch that the server accepts; the latter is a session-ordering violation the server rejects. Future readers of `scenarios.go` needed this distinction spelled out to avoid conflating the two `NoInitialize` scenarios as "the same kind of edge case."

## Deviations from Plan

None — plan executed exactly as written. No Rule 1/2/3 auto-fixes were needed; the one-line production change and the two wire-oracle additions matched the plan's `<action>` blocks and acceptance criteria without adjustment.

## Issues Encountered

- The `go list -deps ./test/wireoracle | rg -c 'modelcontextprotocol/go-sdk'` acceptance check initially appeared to print `1` when run as part of a multi-command shell block, which would have violated VRFY-01. Isolating the command showed 0 real matches — the `1` was output from the *next* command in the same block (`rg -c 'func assertDiscoverCacheControl' ...`), not from the dependency check, which printed nothing (exit 1, no match, as expected for `rg -c` on zero hits). Re-ran isolated and confirmed 0 matches; no code change was needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `ExpectedScenarioCount` is now 24 and `Scenarios()`/`testdata/wireoracle/transcripts/` agree — the base plans 03-02 through 03-05 build their own scenario additions on top of.
- `modernMetaParams()`, `discoverRequest()`, and `modernToolCallRequest()` are now available in `test/wireoracle/scenarios.go` for reuse by any later Phase 3 plan needing to construct a Modern `_meta`-bearing request (e.g. SPEC-02's `-32602`/`-32022` scenarios).
- The `AddReceivingMiddleware` switch now has three cases (`initialize`, `tools/list`, `server/discover`) — the established, proven seam for SPEC-05's per-request `hasIndex` re-check and SPEC-07's `instructions` field, both explicitly deferred to other plans in this phase.
- No blockers. `internal/mcp/archtest`'s VRFY-02 guard was not triggered — no `mcp.CodeUnsupportedProtocolVersion`/`mcp.MetaKeyProtocolVersion` reference was introduced.

---
*Phase: 03-2026-07-28-spec-compliance*
*Completed: 2026-08-06*

## Self-Check: PASSED

All created/modified files found on disk (`internal/mcp/server.go`, `test/wireoracle/scenarios.go`, `test/wireoracle/anchors.go`, `testdata/wireoracle/transcripts/modern-discover-explore.golden`, this SUMMARY). All three task commits (`252d11b`, `360c240`, `0c45d6d`) found in git log.
