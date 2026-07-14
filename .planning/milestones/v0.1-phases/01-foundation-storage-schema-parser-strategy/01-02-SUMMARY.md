---
phase: 01-foundation-storage-schema-parser-strategy
plan: 02
subsystem: database
tags: [protobuf, protoc, schema-versioning, go]

# Dependency graph
requires:
  - phase: 01-foundation-storage-schema-parser-strategy (plan 01)
    provides: Go module + internal/schema, internal/graphstore, internal/parser package skeletons
provides:
  - internal/schema package with proto3 Node/Edge/File/Meta message definitions
  - Generated Go types (graph.pb.go) for the four record kinds
  - SchemaVersion constant + additive-only discipline helpers (meta.go)
  - Proven forward-compatibility mechanism for ARCH-01 (round-trip test)
affects: [01-03 (graphstore, consumes these types as stored values), 01-06 (bulk export, marshals these types)]

# Tech tracking
tech-stack:
  added: [google.golang.org/protobuf (now direct), github.com/google/go-cmp (new, direct — required by protocmp)]
  patterns:
    - "Additive-only protobuf schema evolution: reserved field-number ranges (50-59) on Node/Edge for post-v1 annotations"
    - "Versioned Meta record (schema_version) as the single source of truth for on-disk format version"

key-files:
  created:
    - internal/schema/graph.proto
    - internal/schema/graph.pb.go
    - internal/schema/meta.go
    - internal/schema/roundtrip_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Edge record carries optional line/col fields even though the D-03 Pebble edge KEY omits them — key-identity multiplicity is deferred to Phase 2 extractor design, but the record must not lose this data at extraction time"
  - "Added github.com/google/go-cmp as a direct dependency (was previously unused/absent) — required companion to protocmp.Transform() for message comparison in tests"
  - "Did not run `go mod tidy` — it would have stripped 01-01's deliberately pre-pinned, still-unimported deps (pebble/v2, tree-sitter, wazero, x/tools). Manually promoted only google.golang.org/protobuf and go-cmp from indirect to direct in go.mod since those two are now genuinely imported."

patterns-established:
  - "Pattern: never construct Node/Edge/Meta records ad hoc — schema.NewMeta() stamps SchemaVersion in one place so a future version bump only changes there"
  - "Pattern: any retired protobuf field number is added to a `reserved` clause in graph.proto, never deleted or reused (D-02a)"

requirements-completed: [ARCH-01]

coverage:
  - id: D1
    description: "graph.proto defines proto3 Node/Edge/File/Meta messages in package codegraph.v1 with reserved 50-59 annotation ranges on Node and Edge"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "go build ./internal/schema/..."
        status: pass
    human_judgment: false
  - id: D2
    description: "graph.pb.go generated Go types compile and expose Node/Edge/File/Meta structs"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "go vet ./internal/schema/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "meta.go exports SchemaVersion constant (1) and additive-only discipline documentation/helpers"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "internal/schema/roundtrip_test.go#TestMetaRoundTripsSchemaVersion"
        status: pass
    human_judgment: false
  - id: D4
    description: "A record carrying a field in Node's reserved 50-59 range (simulating a future schema writer) round-trips through the current reader without data loss — the ARCH-01 forward-compatibility proof"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "internal/schema/roundtrip_test.go#TestSchemaRoundTripsUnknownFields"
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-07-10
status: complete
---

# Phase 1 Plan 2: Versioned Protobuf Schema Summary

**proto3 Node/Edge/File/Meta messages with reserved annotation ranges, generated Go types, and a round-trip test proving unknown/future fields survive an older reader intact**

## Performance

- **Duration:** 3 min
- **Started:** 2026-07-10T09:51:39-04:00
- **Completed:** 2026-07-10T09:54:05-04:00
- **Tasks:** 2
- **Files modified:** 6 (4 created in internal/schema, go.mod + go.sum updated)

## Accomplishments
- Authored `graph.proto` (proto3, package `codegraph.v1`) defining Node, Edge, File, and Meta messages, with `reserved 50 to 59;` on both Node and Edge as the physical ARCH-01 annotation slot
- Generated `graph.pb.go` via `protoc` (libprotoc 35.1) + `protoc-gen-go` v1.36.11
- Added `meta.go` exporting `SchemaVersion = 1`, `NewMeta()`, and `IsCurrentSchemaVersion()`, documenting the additive-only (D-02a) discipline
- Proved the ARCH-01 forward-compatibility mechanism with `TestSchemaRoundTripsUnknownFields`: a field injected at number 55 (inside Node's reserved range, simulating a future schema writer) unmarshals through the current reader without error and survives a remarshal
- Two supporting round-trip tests: plain Node equality under `protocmp.Transform()`, and `Meta.schema_version` round-trip

## Task Commits

Each task was committed atomically:

1. **Task 1: Define graph.proto and generate the versioned record types** - `607a156` (feat)
2. **Task 2: Prove forward-compatible round-trip of unknown/future fields** - `1cd00ea` (test)

**Plan metadata:** (this commit, docs: complete plan)

_Note: Task 2 is tagged `tdd="true"` in the plan; see "TDD Gate Compliance" below for how RED/GREEN played out here._

## Files Created/Modified
- `internal/schema/graph.proto` - proto3 message definitions (Node, Edge, File, Meta) with reserved annotation ranges
- `internal/schema/graph.pb.go` - generated Go types
- `internal/schema/meta.go` - SchemaVersion constant + additive-only helpers
- `internal/schema/roundtrip_test.go` - ARCH-01 forward-compat proof + supporting round-trip tests
- `go.mod` / `go.sum` - added `github.com/google/go-cmp` as a direct dependency; promoted `google.golang.org/protobuf` from indirect to direct

## Decisions Made
- Edge carries `line`/`col` fields even though the D-03 Pebble edge key omits them, per RESEARCH Pitfall 2 — preserves call-site data at extraction time even though key-identity multiplicity is a Phase 2 decision
- File record fields: `path`, `content_hash` (documented as requiring SHA-256, Security Domain V6), `language`, `node_count`, `edge_count`
- Meta record fields: `schema_version`, `node_count`, `edge_count`, `last_sync_unix_ms`, `healthy`, `health_message`
- 01-04 (TS DDL capture) has not landed yet, so field names/types were designed against the RESEARCH Pattern 3 sketch; a code comment in `graph.proto` flags this for reconciliation once 01-04's captured DDL is available
- Manually promoted only `google.golang.org/protobuf` and `github.com/google/go-cmp` from `// indirect` to direct in `go.mod`, rather than running `go mod tidy` — a full tidy would have stripped 01-01's deliberately pre-pinned, still-unimported deps (`pebble/v2`, tree-sitter grammars, `wazero`, `x/tools`), which STATE.md records as an intentional decision pending later waves

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added github.com/google/go-cmp as a direct dependency**
- **Found during:** Task 2 (round-trip test)
- **Issue:** `protocmp.Transform()` returns a `cmp.Option` that must be passed to `google/go-cmp`'s `cmp.Diff`/`cmp.Equal`; that package was not yet a dependency of the module
- **Fix:** `go get github.com/google/go-cmp@latest` (resolved v0.7.0), then manually removed the `// indirect` marker since it's now directly imported by `roundtrip_test.go`. This is Google's own canonical comparison library and the plan explicitly directs use of `protocmp` for message equality, so it is a required, unambiguously legitimate companion package, not a discretionary addition.
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./... && go vet ./...` clean; `go mod verify` reports all modules verified
- **Committed in:** 1cd00ea (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — missing test dependency)
**Impact on plan:** Necessary for the plan's own explicit direction to use protocmp; no scope creep, no production code affected.

## TDD Gate Compliance

Task 2 is marked `tdd="true"`. The RED phase surfaced a genuine failure: the first draft of `TestSchemaRoundTripsUnknownFields` compared the full decoded `Node` against the original using `protocmp.Transform()` alone, which counts the injected unknown field as a diff (since the original has no such field) — the test failed with `decoded Node differs from original on known fields`, correctly proving the naive comparison approach was wrong, not the production code. The fix was `protocmp.IgnoreUnknown()` on the known-fields comparison, with unknown-field preservation asserted separately by scanning the re-marshaled wire bytes — this is the actual ARCH-01 proof. No production code (`graph.pb.go`, `meta.go`) required changes to pass; the underlying unknown-field-preservation behavior is protobuf's own built-in mechanism (as documented in 01-RESEARCH.md), and Task 1's generated types already exercise it correctly. The RED→GREEN cycle here was entirely within the test file itself (bad assertion → correct assertion), which is why both the RED investigation and the GREEN fix landed in the same `test(01-02):` commit rather than a separate `test` + `feat` pair — there was no new production behavior to add.

## Issues Encountered
None beyond the TDD gate note above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `internal/schema` provides the record types Plan 01-03 (`GraphStore`/pebble store) will marshal into Pebble values and Plan 01-06 (bulk export) will stream
- Field names/types are provisional pending 01-04's TS DDL capture — flagged in-code; any reconciliation must remain additive-only (D-02a)
- No blockers for 01-03

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*

## Self-Check: PASSED

All created files verified present on disk; both task commits (607a156, 1cd00ea) verified present in git log.
