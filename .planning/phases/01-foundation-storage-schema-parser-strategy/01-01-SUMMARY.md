---
phase: 01-foundation-storage-schema-parser-strategy
plan: 01
subsystem: infra
tags: [go-modules, pebble, protobuf, tree-sitter, wazero, module-bootstrap]

# Dependency graph
requires: []
provides:
  - "github.com/seanb4t/codegraph-go Go module with go.mod/go.sum"
  - "All Phase 1 dependencies pinned at researcher-verified versions on correct import paths"
  - "Empty-but-compiling internal/graphstore, internal/schema, internal/parser package boundaries"
affects: [01-02, 01-03, 01-04, 01-05, 01-06, 01-07]

# Tech tracking
tech-stack:
  added:
    - "github.com/cockroachdb/pebble/v2@v2.1.6"
    - "google.golang.org/protobuf@v1.36.11"
    - "github.com/tree-sitter/go-tree-sitter@v0.25.0"
    - "github.com/tree-sitter/tree-sitter-go@v0.25.0"
    - "github.com/tree-sitter/tree-sitter-python@v0.25.0"
    - "github.com/tetratelabs/wazero@v1.12.0"
    - "golang.org/x/tools@v0.48.0"
  patterns:
    - "internal/graphstore, internal/schema, internal/parser as empty package boundaries, populated by later Wave 2/3 plans without a chicken-and-egg import problem"

key-files:
  created:
    - go.mod
    - go.sum
    - internal/graphstore/doc.go
    - internal/schema/doc.go
    - internal/parser/doc.go
  modified:
    - .gitignore

key-decisions:
  - "Used github.com/cockroachdb/pebble/v2 (not the deprecated bare v1 import path) per RESEARCH Pitfall 4"
  - "Deliberately did not run `go mod tidy` — nothing imports the pinned deps yet, and tidy would strip unused requires before Wave 2 code exists"

patterns-established:
  - "Package doc.go stubs carry only a package clause and one-line doc comment describing the boundary role — no interfaces/types until the owning plan lands"

requirements-completed: [INDX-05, ARCH-01]

coverage:
  - id: D1
    description: "Go module github.com/seanb4t/codegraph-go initialized with go.mod/go.sum"
    verification:
      - kind: unit
        ref: "go build ./..."
        status: pass
    human_judgment: false
  - id: D2
    description: "All six Phase 1 dependencies pinned at researcher-verified versions on correct import paths (pebble/v2, not bare pebble)"
    requirement: "ARCH-01"
    verification:
      - kind: unit
        ref: "go list -m github.com/cockroachdb/pebble/v2 (resolves v2.1.6); go list -m github.com/cockroachdb/pebble (exits non-zero)"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/graphstore, internal/schema, internal/parser packages exist and compile empty"
    requirement: "INDX-05"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-10
status: complete
---

# Phase 01 Plan 01: Go Module & Package Skeleton Bootstrap Summary

**Bootstrapped the `github.com/seanb4t/codegraph-go` Go module with all six Phase 1 dependencies pinned at researcher-verified versions (critically `pebble/v2`, not the deprecated bare v1 path) and created the three empty internal package boundaries that Waves 2-3 build on.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-10T13:41:18Z (previous plan-completion commit)
- **Completed:** 2026-07-10T13:46:36Z
- **Tasks:** 2 completed
- **Files modified:** 6 (go.mod, go.sum, .gitignore, 3x doc.go)

## Accomplishments
- Initialized the Go module (`go mod init github.com/seanb4t/codegraph-go`) and pinned `pebble/v2@v2.1.6`, `protobuf@v1.36.11`, `go-tree-sitter@v0.25.0`, `tree-sitter-go@v0.25.0`, `tree-sitter-python@v0.25.0`, `wazero@v1.12.0`, and `golang.org/x/tools@latest` on their correct import paths
- Verified the Pitfall 4 guard directly: `go list -m github.com/cockroachdb/pebble/v2` resolves to v2.1.6, while `go list -m github.com/cockroachdb/pebble` (bare v1 path) exits non-zero — pebble is present only on `/v2`
- Created `internal/graphstore`, `internal/schema`, `internal/parser` as empty, compiling package stubs with role-describing doc comments only (no interfaces/types), unblocking every Wave 2 plan

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize the Go module and pin Phase 1 dependencies** - `e51f198` (feat)
2. **Task 2: Create the three internal package boundaries as empty compiling stubs** - `056dba4` (feat)

_Note: TDD tasks may have multiple commits (test → feat → refactor). This plan has no `tdd="true"` tasks — both are `type="auto"`._

## Files Created/Modified
- `go.mod` - Module declaration + pinned require directives for all six Phase 1 dependencies
- `go.sum` - Checksums for the full dependency graph (direct + transitive)
- `.gitignore` - Extended with Go build/test artifact exclusions (`/dist/`, `*.test`, `*.out`, coverage output), preserving the pre-existing `.planning/research/.cache/` entry
- `internal/graphstore/doc.go` - `package graphstore`, no imports — sole legal caller of pebble/v2 (D-04)
- `internal/schema/doc.go` - `package schema`, no imports — protobuf serialization layer (D-02)
- `internal/parser/doc.go` - `package parser`, no imports — narrow backend-swappable Parser boundary (D-05b)

## Decisions Made
- Used the `/v2` pebble import path exclusively, per RESEARCH Pitfall 4 and the environment notes' explicit warning; verified via `go list -m` that the bare v1 path is not resolvable as a dependency.
- Did not run `go mod tidy`, per the plan's explicit instruction — no code in this plan imports any of the six pinned dependencies, so `tidy` would strip them as unused before Wave 2 plans exist to consume them. All six show as `// indirect` in `go.mod` for the same reason (nothing yet imports them directly); this is expected at this stage and will resolve naturally once Wave 2 code adds imports.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
The module and package skeleton are in place. Wave 2 plans (01-02 schema, 01-03 parser, 01-05 keys, 01-06 store) can now add files to `internal/graphstore`, `internal/schema`, `internal/parser` without a chicken-and-egg import problem, and can begin importing the pinned dependencies (at which point `go mod tidy` becomes safe to run). No blockers identified.

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*
