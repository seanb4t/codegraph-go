---
phase: 01-foundation-storage-schema-parser-strategy
plan: 03
subsystem: parser
tags: [tree-sitter, cgo, go-tree-sitter, parser-interface]

# Dependency graph
requires:
  - phase: 01-01
    provides: module skeleton with pre-pinned (unimported) tree-sitter deps in go.mod
provides:
  - Narrow, backend-swappable parser.Parser interface (D-05b) with opaque Tree type
  - MaxSourceBytes file-size ceiling + ErrSourceTooLarge sentinel (Security Domain V5 / T-01-03 DoS mitigation)
  - Production CGo tree-sitter backend (internal/parser/cgo) parsing Go and Python
affects: [01-07 (wazero spike A/B's against this interface), Phase 2+ extractors that consume Parser]

# Tech tracking
tech-stack:
  added: [github.com/tree-sitter/go-tree-sitter v0.25.0, github.com/tree-sitter/tree-sitter-go v0.25.0, github.com/tree-sitter/tree-sitter-python v0.25.0]
  patterns: [narrow backend-swappable interface seam (Parser), opaque wrapper type (Tree.inner any) to keep call sites backend-agnostic]

key-files:
  created:
    - internal/parser/parser.go
    - internal/parser/parser_test.go
    - internal/parser/cgo/parser_cgo.go
    - internal/parser/cgo/parser_cgo_test.go
  modified:
    - go.mod

key-decisions:
  - "Tree wraps its backend-specific value in an unexported `any` field (NewTree/Inner accessors) rather than a generic type parameter, keeping the interface simple for the two backends (CGo now, wazero in 01-07) without forcing call sites to specialize on a type parameter"
  - "Manually promoted go-tree-sitter + both grammar modules from indirect to direct requires in go.mod (no go mod tidy run) to avoid stripping 01-01's pre-pinned pebble/v2, wazero, and x/tools deps that are still unimported"
  - "Removed a duplicate package doc comment from parser.go since doc.go already carries the package-level Go doc comment"

patterns-established:
  - "Parser interface (Parse(source, oldTree) (*Tree, error) + Close() error) is the only shape both the CGo and future wazero backends implement — callers never import backend-specific tree-sitter types"
  - "Size-ceiling enforcement happens inside Parse before any backend-specific parse call, so the guard test can run without CGO_ENABLED"

requirements-completed: [DIST-05]

coverage:
  - id: D1
    description: "Narrow Parser interface (Parse/Close) with opaque Tree type, independent of any backend"
    requirement: DIST-05
    verification:
      - kind: unit
        ref: "internal/parser/parser_test.go#TestParserAcceptsInputUnderCeiling"
        status: pass
    human_judgment: false
  - id: D2
    description: "File-size ceiling (MaxSourceBytes) rejects oversize input before any parse call, with ErrSourceTooLarge sentinel"
    requirement: DIST-05
    verification:
      - kind: unit
        ref: "internal/parser/parser_test.go#TestParserRejectsOversizeInput"
        status: pass
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoParseRejectsOversizeInput"
        status: pass
    human_judgment: false
  - id: D3
    description: "CGo tree-sitter backend parses Go and Python source into a Tree"
    requirement: DIST-05
    verification:
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoParsesGoSource"
        status: pass
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoParsesPythonSource"
        status: pass
    human_judgment: false
  - id: D4
    description: "Parser resources are explicitly freed via Close(), safe to call once"
    requirement: DIST-05
    verification:
      - kind: unit
        ref: "internal/parser/cgo/parser_cgo_test.go#TestCGoCloseIsSafeToCallOnce"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-07-10
status: complete
---

# Phase 01 Plan 03: Parser Interface + CGo Tree-sitter Backend Summary

**Narrow backend-swappable Parser interface (D-05b) with a production CGo tree-sitter backend parsing Go and Python, gated by a Security-V5 file-size ceiling and mandatory Close() cleanup**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-10T14:03:04Z
- **Tasks:** 2
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments
- `internal/parser` package: `Parser` interface (`Parse`/`Close`) + opaque `Tree` type, `MaxSourceBytes` ceiling, `ErrSourceTooLarge` sentinel, and a doc comment stating C segfaults are unrecoverable via `recover()`
- `internal/parser/cgo` package: `NewGoParser`/`NewPythonParser` implementing the interface via `github.com/tree-sitter/go-tree-sitter` v0.25.0 + the Go/Python grammar bindings, with the size ceiling enforced before any C call and `Close()` freeing the underlying tree-sitter parser
- Full TDD gate sequence for both tasks: failing test committed first, then the minimal implementation to pass

## Task Commits

Each task was committed atomically (TDD RED/GREEN per task):

1. **Task 1: Define the narrow Parser interface** —
   - `2b95bb3` test(01-03): add failing test for Parser size-ceiling guard (RED)
   - `db2137d` feat(01-03): define narrow backend-swappable Parser interface (GREEN)
2. **Task 2: Implement the CGo tree-sitter backend for Go and Python** —
   - `399d455` test(01-03): add failing test for CGo tree-sitter Go/Python backend (RED)
   - `fd76e9e` feat(01-03): implement CGo tree-sitter backend for Go and Python (GREEN)
   - `38d9aa3` fix(01-03): remove duplicate package doc comment in parser.go (Rule 1 auto-fix)

_Note: TDD tasks have RED (test) then GREEN (feat) commits, as required._

## Files Created/Modified
- `internal/parser/parser.go` - `Parser` interface, opaque `Tree` type, `MaxSourceBytes`/`ErrSourceTooLarge`, security/crash-isolation contract documented in the interface doc comment
- `internal/parser/parser_test.go` - `TestParserRejectsOversizeInput`, `TestParserAcceptsInputUnderCeiling`, `TestParserCloseIsCallable` against a CGo-free stub Parser
- `internal/parser/cgo/parser_cgo.go` - `CGoParser`, `NewGoParser`, `NewPythonParser` implementing `parser.Parser` via tree-sitter CGo bindings
- `internal/parser/cgo/parser_cgo_test.go` - `TestCGoParsesGoSource`, `TestCGoParsesPythonSource`, `TestCGoParseRejectsOversizeInput`, `TestCGoCloseIsSafeToCallOnce`
- `go.mod` - promoted `go-tree-sitter`, `tree-sitter-go`, `tree-sitter-python` from indirect to direct requires (manual edit, no `go mod tidy`)

## Decisions Made
- Wrapped backend trees in `Tree{ inner any }` with `NewTree`/`Inner()` accessors instead of Go generics, keeping the interface trivial for the two backends this project needs (CGo now, wazero in 01-07)
- Manually edited `go.mod` to promote the three tree-sitter requires to direct rather than running `go mod tidy`, per the environment note that a full tidy would strip 01-01's pre-pinned-but-still-unimported `pebble/v2`, `wazero`, and `golang.org/x/tools` deps
- Removed the redundant package doc comment I initially added to `parser.go` since `doc.go` (from 01-01) already carries the canonical package comment — kept as a same-scope auto-fix (Rule 1) rather than a separate deviation entry below since it was self-introduced and corrected within the same task before commit review

## Deviations from Plan

None outside the self-corrected duplicate-doc-comment fixup noted above (Rule 1 — code-quality issue introduced and fixed within the same task, not a deviation from the plan's design).

## Issues Encountered
None. The plan's `01-RESEARCH.md` API sketch for `NewGoParser`/`SetLanguage` matched the installed `go-tree-sitter@v0.25.0` module exactly (verified directly against the module cache source before writing the backend), so no debugging was needed on the CGo API shape.

## User Setup Required
None - no external service configuration required. CGo toolchain (Apple clang) was already verified available in this environment before starting.

## Next Phase Readiness
- The `parser.Parser` interface is the seam Plan 01-07 will implement a second (wazero) time to A/B against this CGo backend and settle the parser-strategy decision (D-05)
- `go.mod`'s tree-sitter requires are live and direct; `pebble/v2`, `wazero`, and `x/tools` remain correctly pinned-but-unimported for later plans
- No blockers for 01-07 or subsequent extractor work that will consume `parser.Parser`

---
*Phase: 01-foundation-storage-schema-parser-strategy*
*Completed: 2026-07-10*

## Self-Check: PASSED

All created files found on disk; all 5 task commit hashes (2b95bb3, db2137d, 399d455, fd76e9e, 38d9aa3) found in git log.
