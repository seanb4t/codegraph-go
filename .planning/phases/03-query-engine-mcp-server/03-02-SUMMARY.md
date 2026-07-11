---
phase: 03-query-engine-mcp-server
plan: 02
subsystem: database
tags: [query-engine, pebble, graphstore, go]

# Dependency graph
requires:
  - phase: 03-query-engine-mcp-server
    provides: "03-01's Reader.IterateNodes()/IterateFiles() enumeration (the read path this engine will build query verbs on top of)"
provides:
  - "internal/query.Engine — the read-only construction seam wrapping a graphstore.Reader"
  - "internal/query.OpenAt — the single read seam (resolve -> Open -> Snapshot -> Engine + idempotent closer) CLI commands and MCP tool handlers both call (D-08b)"
  - "internal/query.ResolveCodegraphDir — nearest-.codegraph/ upward walk + ErrNotInitialized sentinel (D-01a)"
  - "internal/query.ValidateKind — the goextract-kind-sourced --kind allow-list guard (V5, T-03-02-Kind)"
  - "internal/query's clampDepth/validateLimit/validateMaxFiles — numeric-flag DoS ceilings (V5, T-03-02-DoS)"
  - "internal/query/engine_test.go — the shared copyFixture/indexFixture test harness reused by 03-03/04/05"
affects: [03-03, 03-04, 03-05, 03-06, 03-07, 03-08, query-engine, mcp-server]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "OpenAt: resolve -> graphstore.Open(<dir>/.codegraph/store) -> Snapshot() -> Engine + idempotent io.Closer — the one construction path every future query verb and MCP tool handler reuses"
    - "Upward filepath.Dir walk stopping at filepath.Dir(dir)==dir, returning the containing directory (not the .codegraph/ path itself)"
    - "Authoritative allow-list built from the extractor's own exported Kind* constants (plus one documented unexported-const string literal) rather than a hand-typed literal list, so it cannot silently drift"

key-files:
  created:
    - internal/query/engine.go
    - internal/query/resolve.go
    - internal/query/validate.go
    - internal/query/engine_test.go
  modified: []

key-decisions:
  - "ResolveCodegraphDir returns the directory CONTAINING .codegraph/ (the repo root), not the .codegraph/ path itself — OpenAt then joins codegraphDirName + storeSubdir onto that root, matching the plan's literal task-1 behavior spec over RESEARCH's Pattern-4 example (which returned the .codegraph/ path)"
  - "knownKinds' \"package\" entry is a documented string literal, not an import of internal/indexer's unexported kindPackage const — internal/indexer deliberately keeps it unexported, so the tie-back is a comment, not a compiler-enforced link"
  - "storeSubdir/codegraphDirName constants are re-declared locally in internal/query rather than imported from internal/cli, keeping the dependency direction CLI-depends-on-query (not the reverse)"

patterns-established:
  - "Every future internal/query verb method hangs off Engine{reader graphstore.Reader} and is exercised via engine_test.go's copyFixture+indexFixture harness against a real Pebble-backed store"

requirements-completed: [QRY-01]

coverage:
  - id: D1
    description: "ResolveCodegraphDir resolves the nearest .codegraph/ at or above a start path (including nested subdirectories), or returns a clear ErrNotInitialized at the filesystem root"
    requirement: "QRY-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_test.go#TestResolveCodegraphDir"
        status: pass
    human_judgment: false
  - id: D2
    description: "OpenAt yields a fresh-snapshot Engine + idempotent closer per invocation for an init'd repo, and a clear ErrNotInitialized for an uninitialized one"
    requirement: "QRY-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_test.go#TestOpenAt"
        status: pass
    human_judgment: false
  - id: D3
    description: "clampDepth/validateLimit/validateMaxFiles bound --depth/--limit/--max-files to documented ceilings before any traversal/allocation (V5, T-03-02-DoS)"
    verification:
      - kind: unit
        ref: "internal/query/engine_test.go#TestClampDepth"
        status: pass
      - kind: unit
        ref: "internal/query/engine_test.go#TestValidateLimit"
        status: pass
      - kind: unit
        ref: "internal/query/engine_test.go#TestValidateMaxFiles"
        status: pass
    human_judgment: false
  - id: D4
    description: "ValidateKind accepts the empty string and every goextract-sourced known kind, and rejects an unknown kind with an allowed-set error before any node scan (V5, T-03-02-Kind)"
    verification:
      - kind: unit
        ref: "internal/query/engine_test.go#TestValidateKind"
        status: pass
    human_judgment: false
  - id: D5
    description: "internal/query respects the graphstore/archtest import boundary (no direct pebble/v2 import) even after importing goextract's kind constants"
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest/import_graph_test.go#TestNoPackageBypassesGraphStore"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-11
status: complete
---

# Phase 3 Plan 2: Query Engine Foundation Summary

**Read-only `internal/query` engine scaffold — `OpenAt`'s resolve→open→snapshot→closer seam, the nearest-`.codegraph/` upward walk, and DoS/kind input-validation guards, all TDD RED→GREEN with a shared `copyFixture`/`indexFixture` test harness.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-11T09:39:34-04:00 (approx, first commit-adjacent read)
- **Completed:** 2026-07-11T09:44:34-04:00
- **Tasks:** 2 (RED + GREEN)
- **Files modified:** 4 (all created)

## Accomplishments
- `internal/query.Engine` + `New`/`OpenAt` established as the single read seam (D-08b) — `OpenAt` resolves the nearest `.codegraph/`, opens the store, takes a fresh `Snapshot()` (D-02), and returns an idempotent `io.Closer`
- `ResolveCodegraphDir` implements the upward `filepath.Dir` walk (D-01a, RESEARCH Pattern 4), returning `ErrNotInitialized` at the filesystem root
- `clampDepth`/`validateLimit`/`validateMaxFiles` enforce documented `MaxDepth=50`/`MaxLimit=1000`/`MaxFiles=1000` ceilings (V5, RESEARCH Pitfall 4, T-03-02-DoS)
- `ValidateKind` closes the `--kind` validation gap (T-03-02-Kind): the allow-list is built from `goextract`'s exported `Kind*` constants plus the documented `"package"` synthetic kind, so it cannot silently drift from what the extractor actually emits
- `engine_test.go`'s `copyFixture`/`indexFixture` harness (mirroring `internal/cli/cli_test.go`'s helper, extended with a real `indexer.Run` build) is now the shared scaffold 03-03/04/05 will extend

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Engine + ResolveCodegraphDir + validation clamps with failing harness tests** - `8562e72` (test)
2. **Task 2 (GREEN): Implement resolve/open/validate; harness green; boundary verified** - `27e67f2` (feat)

_TDD gate sequence confirmed: `test(03-02)` commit precedes `feat(03-02)` commit; no REFACTOR commit needed (GREEN bodies were the already-designed shape, no cleanup pass required)._

## Files Created/Modified
- `internal/query/engine.go` - `Engine` type, `New`, `OpenAt` (resolve → `graphstore.Open` → `Snapshot()` → `Engine` + idempotent `engineCloser`)
- `internal/query/resolve.go` - `ResolveCodegraphDir` upward walk, `ErrNotInitialized` sentinel, `codegraphDirName` constant
- `internal/query/validate.go` - `MaxDepth`/`MaxLimit`/`MaxFiles` ceilings, `clampDepth`/`validateLimit`/`validateMaxFiles`, `ValidateKind` + `knownKinds`
- `internal/query/engine_test.go` - `copyFixture`/`indexFixture` shared harness + RED/GREEN behavior tests for all of the above

## Decisions Made
- `ResolveCodegraphDir` returns the directory *containing* `.codegraph/` (matching the plan's task-1 behavior spec literally: "returns the nearest directory containing a `.codegraph` subdir"), not the `.codegraph/` path itself as RESEARCH's Pattern-4 illustrative snippet did — `OpenAt` then does the `filepath.Join(dir, codegraphDirName, storeSubdir)` composition itself
- `knownKinds`'s `"package"` entry stays a documented string literal rather than importing `internal/indexer`'s unexported `kindPackage` — the const is deliberately unexported in that package, so this plan ties the two together via a comment (flagged for anyone changing `kindPackage`'s value in the future) instead of forcing an export change out-of-scope for this plan
- `storeSubdir`/`codegraphDirName` are re-declared as local constants in `internal/query` rather than imported from `internal/cli`, keeping the dependency direction CLI→query (query commands will import `internal/query`, not the reverse)

## Deviations from Plan

None - plan executed exactly as written. `ValidateKind`'s allow-list construction used `goextract`'s exported constants for 8 of 9 kinds as directed; the 9th (`"package"`) is a documented literal per the constraint that `internal/indexer`'s `kindPackage` is unexported (see Decisions Made).

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/query.Engine`/`OpenAt`/`ValidateKind`/`clampDepth`/`validateLimit`/`validateMaxFiles` are ready for 03-03 (`query`/`node`/`search`), 03-04 (`callers`/`callees`/`impact`), and 03-05 (`affected`/`files`/`status`) to build their query-verb methods directly on top of
- `engine_test.go`'s `copyFixture`/`indexFixture` helpers are exported-within-package and ready for those plans to extend with additional `t.Run` cases in the same file or new `_test.go` files in the same package
- No blockers or concerns carried forward

---
*Phase: 03-query-engine-mcp-server*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created files (`internal/query/engine.go`, `resolve.go`, `validate.go`, `engine_test.go`) and both task commit hashes (`8562e72`, `27e67f2`) verified present on disk / in `git log`.
