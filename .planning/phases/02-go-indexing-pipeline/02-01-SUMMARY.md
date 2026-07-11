---
phase: 02-go-indexing-pipeline
plan: 01
subsystem: database
tags: [protobuf, sha256, go, schema, node-identity]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: graph.proto/graph.pb.go schema, keys.go appendSegment discipline, SchemaVersion const
provides:
  - Additively extended graph.proto (Node signature/docstring/visibility/is_exported/return_type; Edge provenance/metadata; File errors)
  - Regenerated graph.pb.go with accessor methods for the new fields
  - internal/indexer/nodeid package: NodeID(kind, qualifiedName, filePath) string
  - golang.org/x/sync and golang.org/x/mod promoted to direct go.mod requires
affects: [go-extractor, resolver, symbol-pass, edge-pass]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Additive-only protobuf schema evolution (D-02a): new fields appended after the last existing number, staying below reserved 50-59, never renumbering"
    - "Varint-length-prefixed hash preimage segments (mirrors keys.go's appendSegment) to prevent id-forging segment-boundary injection"

key-files:
  created:
    - internal/indexer/nodeid/nodeid.go
    - internal/indexer/nodeid/nodeid_test.go
    - internal/indexer/nodeid/doc.go
  modified:
    - internal/schema/graph.proto
    - internal/schema/graph.pb.go
    - go.mod

key-decisions:
  - "NodeID uses SHA-256 (never MD5) truncated to 32 hex chars for TS-parity visual id length while retaining collision resistance (D-02a)"
  - "graph.proto extended additively only: new field numbers appended after each message's last field, all below reserved 50-59; SchemaVersion stayed at 1"
  - "x/sync and x/mod promoted from indirect to direct in go.mod by hand-editing (not go mod tidy, per STATE.md caution about stripping pre-pinned deps)"

patterns-established:
  - "Leaf packages (no in-repo imports) for shared primitives like nodeid, so downstream packages (extractor, resolver) can both depend on it without an import cycle"

requirements-completed: [LANG-01, RES-01]

coverage:
  - id: D1
    description: "graph.proto additively extended with Go parity fields (Node signature/docstring/visibility/is_exported/return_type; Edge provenance/metadata; File errors); graph.pb.go regenerated via protoc"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/schema/roundtrip_test.go#TestSchemaRoundTripsUnknownFields"
        status: pass
      - kind: other
        ref: "go build ./internal/schema/... && go test ./internal/schema/... -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "nodeid.NodeID(kind, qualifiedName, filePath) produces deterministic, collision-resistant (SHA-256), injection-safe <kind>:<32-hex> ids"
    requirement: RES-01
    verification:
      - kind: unit
        ref: "internal/indexer/nodeid/nodeid_test.go#TestNodeID_Shape"
        status: pass
      - kind: unit
        ref: "internal/indexer/nodeid/nodeid_test.go#TestNodeID_Deterministic"
        status: pass
      - kind: unit
        ref: "internal/indexer/nodeid/nodeid_test.go#TestNodeID_Distinct"
        status: pass
      - kind: unit
        ref: "internal/indexer/nodeid/nodeid_test.go#TestNodeID_InjectionSafe"
        status: pass
      - kind: unit
        ref: "internal/indexer/nodeid/nodeid_test.go#TestNodeID_EmptyArgs"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-10
status: complete
---

# Phase 2 Plan 01: Node Identity and Schema Field Parity Summary

**Deterministic SHA-256 NodeID content hasher plus additive graph.proto extension carrying every Go extractor parity field, both anchored on the keys.go length-prefixed segment discipline for injection safety**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-11T01:24:39Z
- **Completed:** 2026-07-11
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Extended `internal/schema/graph.proto` additively: Node gained `signature`/`docstring`/`visibility`/`is_exported`/`return_type` (11-15), Edge gained `provenance`/`metadata` (6-7), File gained `errors` (6, repeated) — all below the reserved 50-59 range, `SchemaVersion` unchanged at 1
- Regenerated `graph.pb.go` via `protoc`/`protoc-gen-go` (not hand-edited); Phase-1 round-trip test passes unchanged, proving forward/backward compatibility
- Implemented `internal/indexer/nodeid.NodeID(kind, qualifiedName, filePath) string`: a leaf package producing deterministic, collision-resistant (SHA-256, never MD5), injection-safe `<kind>:<32-hex>` ids via TDD (RED test commit, then GREEN implementation commit)
- Promoted `golang.org/x/sync` and `golang.org/x/mod` from indirect to direct in `go.mod`

## Task Commits

Each task was committed atomically:

1. **Task 1: Additively extend graph.proto and regenerate bindings** - `01d000f` (feat)
2. **Task 2: NodeID content hasher — RED** - `27fa933` (test)
2. **Task 2: NodeID content hasher — GREEN** - `2ee6b0c` (feat)

**Plan metadata:** (pending, this commit)

_Note: Task 2 is TDD — RED (`test`) commit precedes GREEN (`feat`) commit._

## Files Created/Modified
- `internal/schema/graph.proto` - Additive Go-parity fields on Node/Edge/File
- `internal/schema/graph.pb.go` - Regenerated Go bindings (accessor methods for new fields)
- `go.mod` - x/sync, x/mod promoted to direct
- `internal/indexer/nodeid/nodeid.go` - `NodeID` implementation (length-prefixed preimage, SHA-256)
- `internal/indexer/nodeid/nodeid_test.go` - Table-driven determinism/distinctness/injection-safety tests
- `internal/indexer/nodeid/doc.go` - Package doc

## Decisions Made
- SHA-256 truncated to 32 hex chars (not full 64) to match the TS corpus's visual id length while keeping collision resistance — the truncation only shortens the printed digest, it does not weaken the underlying hash family choice mandated by D-02a
- Fixed a `protoc` invocation pitfall mid-task: running `protoc --proto_path=internal/schema internal/schema/graph.proto` writes `paths=source_relative` output relative to the proto_path root, landing the generated file at repo-root `./graph.pb.go` instead of `internal/schema/graph.pb.go`. Corrected by invoking with `--proto_path=.` from the repo root so the relative path resolves correctly (Rule 3 — blocking issue, fixed inline, no separate commit needed since it was caught before any file was staged)

## Deviations from Plan

None - plan executed exactly as written. (The protoc invocation correction above was a build-tooling correction made before any commit, not a deviation from the plan's design.)

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/indexer/nodeid` is ready for the Go extractor (Plan 02-02+) to build node ids as it walks the AST
- `graph.proto`'s new fields are ready for the extractor's symbol pass to populate (signature/docstring/visibility/is_exported/return_type) and the edge pass (provenance/metadata) and file pass (errors)
- No blockers for subsequent Phase 2 plans

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-10*

## Self-Check: PASSED

All created files verified present on disk; all three task commit hashes (`01d000f`, `27fa933`, `2ee6b0c`) verified present in git log.
