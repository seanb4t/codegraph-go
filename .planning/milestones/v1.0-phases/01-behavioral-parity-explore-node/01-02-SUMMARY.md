---
phase: 01-behavioral-parity-explore-node
plan: 02
subsystem: indexer
tags: [goextract, edge-kinds, rank-edges, d-09]

# Dependency graph
requires:
  - phase: 01-behavioral-parity-explore-node (plan 01)
    provides: Phase 1 context/decisions (D-09 edge-kind expansion scope)
provides:
  - "Six new shared string constants (RefKindReferences, RefKindInstantiates, RefKindReturns, RefKindTypeOf, EdgeKindExtends, EdgeKindOverrides) in goextract's shared vocabulary"
  - "Updated UnresolvedRef.Kind doc comment enumerating the new Pass-1-captured kinds"
affects: [01-05 (resolve.go + per-language extractor emission), 01-06 (query/rwr.go RankEdges set), 01-08, 01-09]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One shared edge-kind definition, not three copies — new RefKind*/EdgeKind* constants declared once in goextract/types.go, referenced everywhere else"

key-files:
  created: []
  modified:
    - internal/indexer/goextract/types.go

key-decisions:
  - "No SchemaVersion bump, no proto regeneration — Edge.kind is a free-form proto3 string (graph.proto:73), so the 6 new kinds are additive DATA values, not a schema change"
  - "extends/overrides get EdgeKind* constants (Pass-2 synthesis only, no UnresolvedRef.Kind case); references/instantiates/returns/type_of get RefKind* constants (Pass-1 captured, do get an UnresolvedRef.Kind case) — matches RESEARCH §B's Pass-1 vs Pass-2 distinction"

patterns-established:
  - "EdgeKindExtends/EdgeKindOverrides doc comments mirror EdgeKindImplements's discipline verbatim: named here so resolve.go's synthesis and query/rwr.go's RankEdges share ONE definition rather than copies"

requirements-completed: [EXPL-02]

coverage:
  - id: D1
    description: "The 6 D-09 edge-kind names exist as single shared string constants in goextract/types.go, ready for resolve.go emission and rwr.go's RankEdges"
    requirement: "EXPL-02"
    verification:
      - kind: unit
        ref: "go build ./internal/indexer/goextract/ && go vet ./internal/indexer/goextract/"
        status: pass
      - kind: other
        ref: "go doc -all ./internal/indexer/goextract — confirms all 6 constant names present"
        status: pass
    human_judgment: false
  - id: D2
    description: "No SchemaVersion bump and no proto regeneration occurred"
    requirement: "EXPL-02"
    verification:
      - kind: other
        ref: "git diff --stat internal/schema/ (empty output)"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-07-15
status: complete
---

# Phase 1 Plan 2: D-09 Edge-Kind Vocabulary Foundation Summary

**Added 6 shared edge-kind string constants (references/instantiates/returns/type_of/extends/overrides) to goextract's vocabulary — the single load-bearing foundation task (F1) that unblocks the extraction and RWR-ranking work in later plans.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-07-15T12:15:06Z
- **Completed:** 2026-07-15T12:20:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Declared `RefKindReferences`, `RefKindInstantiates`, `RefKindReturns`, `RefKindTypeOf` as Pass-1-captured constants
- Declared `EdgeKindExtends`, `EdgeKindOverrides` as Pass-2-synthesis-only constants, mirroring `EdgeKindImplements`'s doc-comment discipline verbatim
- Updated `UnresolvedRef.Kind` doc comment to enumerate the new Pass-1 kinds alongside the existing ones
- Confirmed zero change to `internal/schema/` — no SchemaVersion bump, no proto regeneration (Edge.kind is a free-form proto3 string)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the 6 D-09 edge-kind constants to goextract vocabulary** - `7b31c53` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/indexer/goextract/types.go` - Added 6 new exported constants (4 `RefKind*`, 2 `EdgeKind*`) and updated the `UnresolvedRef.Kind` doc comment

## Decisions Made
None beyond what the plan specified — followed the exact pattern from 01-PATTERNS.md's "goextract/types.go (extend)" section (mirroring `EdgeKindImplements`'s doc-comment discipline).

## Deviations from Plan

### Notes (not auto-fixes — verification method adjustment only)

**1. Plan's literal `<verify><automated>` command undercounts due to `go doc` const-group collapsing**
- **Found during:** Task 1 verification
- **Issue:** The plan's automated verify command (`go doc ./internal/indexer/goextract | grep -c ... | awk '$1>=6...'`) uses `go doc` without `-all`. Go's `doc` tool collapses grouped `const` blocks into a single summary line (e.g. `const RefKindCalls = "calls" ...`), so only 3 of the 6 new names surface as literal substring matches, undercounting even though all 6 constants are correctly declared and exported.
- **Fix:** No code fix needed. Verified the actual goal (6 constants exist, package compiles) via `go doc -all ./internal/indexer/goextract`, which shows all 6 names with their doc comments, plus `go build`/`go vet` (both pass cleanly). This is a plan-authoring artifact, not an implementation gap — no source change warranted (Rule 3 would apply if code were broken; here the verify *script* under-specifies a flag, so I substituted an equivalent check rather than editing the plan).
- **Files modified:** None (verification-only)
- **Verification:** `go build ./internal/indexer/goextract/ && go vet ./internal/indexer/goextract/` passes; `go doc -all ./internal/indexer/goextract` shows all 6 new constant names; `git diff --stat internal/schema/` is empty.
- **Committed in:** N/A (no code change required)

---

**Total deviations:** 0 code deviations; 1 verification-method note (plan's `go doc` flag omission, worked around with `-all`)
**Impact on plan:** None on scope or correctness — implementation matches the plan exactly.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The 6 shared edge-kind constants are ready for plan 05 (resolve.go's Pass-2 emission + per-language extractors' Pass-1 emission) and plan 06 (query/rwr.go's RankEdges set) to reference — no re-declaration needed anywhere downstream.
- `internal/schema/` is untouched, confirming F1's "no schema/proto/SchemaVersion change" contract holds for the rest of the D-09 foundation wave (F2-F5).

---
*Phase: 01-behavioral-parity-explore-node*
*Completed: 2026-07-15*

## Self-Check: PASSED

- FOUND: internal/indexer/goextract/types.go
- FOUND: commit 7b31c53
