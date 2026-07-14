---
phase: 06-agent-integrations-cli-lifecycle
plan: 01
subsystem: infra
tags: [cli, installer, agent-integration, json, marker-fences, tdd, go-registry]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    provides: "internal/indexer/languages.go registry-keyed-by-ID pattern mirrored here"
provides:
  - "internal/agents package: AgentTarget interface + value types (types.go)"
  - "Surgical format-preserving write helpers (shared.go): readJSONFile/writeJSONFile/jsonDeepEqual/writeMcpEntry/removeMcpEntry/replaceOrAppendMarkedSection/removeMarkedSection/upsertInstructionsEntry/atomicWriteFile"
  - "ID-keyed target registry (registry.go): registerTarget/GetTarget/AllTargetIDs/AllTargets/DetectAll/ResolveTargetFlag"
  - "Marker-fenced instructions block (instructions.go) with exact D-01a marker text"
  - "github.com/tailscale/hujson pinned in go.mod, unimported until 06-03"
affects: [06-02, 06-03, 06-04]

# Tech tracking
tech-stack:
  added: ["github.com/tailscale/hujson v0.0.0-20260302212456-ecc657c15afd (pinned, unimported)"]
  patterns:
    - "Registry-keyed-by-ID with per-variant self-registering init() (mirrors internal/indexer/languages.go)"
    - "Marker-fenced span upsert/remove as the sole external-file-editing primitive (replaceOrAppendMarkedSection/removeMarkedSection)"
    - "atomicWriteFile (temp-in-same-dir + os.Rename) as the single write path for every helper in the package"

key-files:
  created:
    - internal/agents/types.go
    - internal/agents/shared.go
    - internal/agents/registry.go
    - internal/agents/instructions.go
    - internal/agents/shared_test.go
    - internal/agents/registry_test.go
    - internal/agents/testhelpers_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "upsertInstructionsEntry takes startMarker/endMarker/content as parameters rather than importing instructions.go's consts, per the plan's own note (Task 2 landed before Task 3's consts existed)"
  - "removeMarkedSection strips the blank-line separator on both sides of the marked span so insert->remove is byte-exact against pre-insert content, including deleting the file entirely when it never existed pre-insert"
  - "writeMcpEntry/removeMcpEntry normalize built entries through a JSON marshal/unmarshal round trip before jsonDeepEqual, so callers may pass any concrete Go type (struct, []string, map[string]string) without hand-matching JSON's decoded shapes"
  - "ResolveTargetFlag's auto-detects-nothing fallback returns the Claude target specifically (RESEARCH.md recommendation), not an empty list — avoids a silent no-op install on a clean environment"

patterns-established:
  - "Doc comments on every exported symbol cite the D-NN/T-06-NN identifier they satisfy, mirroring internal/indexer/languages.go's convention"
  - "Boundary discipline: internal/agents/*.go avoid even mentioning the forbidden package paths in comments, since the phase's own acceptance grep matches doc text too"

requirements-completed: [AGNT-01, AGNT-02]

coverage:
  - id: D1
    description: "AgentTarget interface + value types (Location, TargetID, DetectionResult, FileAction/FileResult, WriteResult, InstallOptions) compile and are documented with D-NN citations"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "go build ./internal/agents/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "Surgical-write helpers pass round-trip + idempotency + preserve-siblings + no-panic tests under -race"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/agents/shared_test.go (TestShared*, TestMarker*, TestMcpEntry*, TestReadJSON*, TestRoundTrip*)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Registry lookup/sorted-list/resolve-flag (incl. auto->Claude fallback) pass"
    requirement: "AGNT-02"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go (TestRegistry*, TestResolveTargetFlag*)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Short instructions const carries the exact TS marker fences and a codegraph_explore reference"
    requirement: "AGNT-01"
    verification:
      - kind: unit
        ref: "internal/agents/registry_test.go#TestInstructionsBlock_ExactMarkerText, #TestInstructionsBlock_HasMarkersAndCodegraphExploreReference"
        status: pass
    human_judgment: false
  - id: D5
    description: "hujson pinned unimported per project convention; internal/agents imports none of graphstore/indexer/query"
    verification:
      - kind: unit
        ref: "go.mod direct require + go.sum hash; rg boundary check over internal/agents/"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 01: internal/agents Foundation Summary

**AgentTarget interface, surgical JSON/marker write helpers, and ID-keyed registry for the codegraph install/uninstall subsystem — hujson pinned unimported for 06-03's opencode JSONC editing.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-12T18:23:22Z
- **Completed:** 2026-07-12T18:35:00Z
- **Tasks:** 3
- **Files modified:** 9 (go.mod, go.sum, + 7 new internal/agents files)

## Accomplishments
- `AgentTarget` interface + value types (`Location`, `TargetID`, `DetectionResult`, `FileAction`/`FileResult`, `WriteResult`, `InstallOptions`) — every exported symbol doc-commented against the D-NN decision it serves
- Surgical, format-preserving write primitives (`readJSONFile`, `writeJSONFile`, `jsonDeepEqual`, `writeMcpEntry`, `removeMcpEntry`, `replaceOrAppendMarkedSection`, `removeMarkedSection`, `upsertInstructionsEntry`, `atomicWriteFile`) — all funneled through one atomic-write chokepoint (V12), all defensive against malformed input (V5), proven byte-invariant on insert→remove round trip (T-06-02)
- ID-keyed target registry (`registerTarget`/`GetTarget`/`AllTargetIDs`/`AllTargets`/`DetectAll`/`ResolveTargetFlag`) mirroring `internal/indexer/languages.go`'s registry-keyed-by-ID shape, including `--target auto`'s zero-detected → Claude-fallback behavior
- Marker-fenced short instructions block (`codegraphInstructionsBlock`) with the exact `<!-- CODEGRAPH_START/END -->` D-01a parity-contract text and a `codegraph_explore` reference
- `github.com/tailscale/hujson` pinned in `go.mod`'s direct require block (manual edit, no `go mod tidy`, per project convention), unimported until 06-03

## Task Commits

Each task was committed atomically; Tasks 2 and 3 (both `tdd="true"`) each landed as a RED test commit followed by a GREEN implementation commit:

1. **Task 1: Pin hujson + scaffold package types.go + test harness** - `8131e2f` (feat)
2. **Task 2: shared.go surgical write helpers** - `7095f81` (test, RED) → `1549fad` (feat, GREEN)
3. **Task 3: registry.go + instructions.go** - `bfb599f` (test, RED) → `e624dbe` (feat, GREEN)

## Files Created/Modified
- `internal/agents/types.go` - `AgentTarget` interface + value types (D-02/D-03/D-04/D-05/D-07/D-08 citations)
- `internal/agents/shared.go` - readJSONFile/writeJSONFile/jsonDeepEqual/writeMcpEntry/removeMcpEntry/replaceOrAppendMarkedSection/removeMarkedSection/upsertInstructionsEntry/atomicWriteFile
- `internal/agents/registry.go` - registerTarget/GetTarget/AllTargetIDs/AllTargets/DetectAll/ResolveTargetFlag
- `internal/agents/instructions.go` - codegraphSectionStart/End consts + codegraphInstructionsBlock
- `internal/agents/testhelpers_test.go` - fakeHome/readFile/writeFile fixture-isolation trio
- `internal/agents/shared_test.go` - RED→GREEN behavior coverage for every shared.go helper
- `internal/agents/registry_test.go` - RED→GREEN behavior coverage for registry.go + instructions.go, with fakeTarget stubs + resetRegistryForTest isolation
- `go.mod` / `go.sum` - hujson pinned direct require + hash

## Decisions Made
- Kept `upsertInstructionsEntry`'s markers as function parameters rather than importing `instructions.go`'s consts, exactly as the plan's Task 2 action text anticipated (Task 2 landed before Task 3's consts existed)
- `removeMarkedSection` strips the blank-line separator on both sides of the marked span so insert→remove round-trips to byte-exact pre-insert content, including deleting the file entirely when it never existed before insert
- `writeMcpEntry`/`removeMcpEntry` normalize built entries through a JSON marshal/unmarshal round trip before `jsonDeepEqual`, so per-agent callers (06-02/06-03) can pass any concrete Go type without hand-matching `readJSONFile`'s decoded `any` shapes
- `ResolveTargetFlag("auto", ...)` falls back to just the `Claude` target when zero agents are detected (RESEARCH.md's least-surprise recommendation), avoiding a silent no-op install on a clean environment

## Deviations from Plan

None - plan executed exactly as written. Two doc-comment wordings (in `types.go` and `registry.go`) were adjusted after the fact because their prose happened to contain the literal package-path strings the plan's own boundary-discipline grep checks for (`internal/graphstore`/`internal/indexer`/`internal/query`) — reworded to describe the same constraint without the literal path text, so the acceptance grep and the doc comment's intent both hold. Not a deviation from plan behavior, just wording that would have false-failed the plan's own acceptance check.

## Issues Encountered
None.

## Next Phase Readiness
- `internal/agents` compiles, is fully boundary-clean (no graphstore/indexer/query imports), and every helper the per-agent target files (06-02, 06-03) need is in place and tested
- `AgentTarget` interface is stable and ready for `claude.go`/`cursor.go`/`gemini.go`/`kiro.go`/`antigravity.go` (06-02) and `codex.go`/`opencode.go`/`hermes.go` (06-03) to implement
- `hujson` is pinned and ready to import in 06-03 for opencode's comment-preserving JSONC editing
- No blockers

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*

## Self-Check: PASSED

All 7 created files found on disk; all 5 task/RED/GREEN commit hashes (8131e2f, 7095f81, 1549fad, bfb599f, e624dbe) found in git log.
