---
phase: 02-go-indexing-pipeline
plan: 02
subsystem: indexer
tags: [go, go-build, modfile, file-discovery, determinism]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline (plan 01)
    provides: internal/indexer/nodeid.NodeID content hasher, additively-extended graph.proto/graph.pb.go
provides:
  - internal/indexer.Discover(root) ([]DiscoveredFile, string, error) — deterministic, build-filtered, import-path-tagged file discovery (Pass-0 of the two-pass pipeline)
  - internal/indexer.DiscoveredFile struct (AbsPath/RelPath/ImportPath) — the shape Pass 1 (extract) and Pass 2 (resolve) consume
  - Committed multi-package Go fixture (example.com/gofixture) under internal/indexer/testdata/gofixture/ — shared ground truth for all downstream Phase-2 indexer tests
affects: [go-extractor, resolver, symbol-pass, edge-pass, determinism-tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "File discovery via go/build.Context.MatchFile (never a hand-rolled //go:build parser or go/packages+go list)"
    - "MatchFile called with each file's own parent directory, never a hoisted/cached dir (Pitfall 5)"
    - "Result slices sorted by a stable key (RelPath) before returning — determinism's first line of defense"
    - "golang.org/x/mod/modfile.Parse for go.mod module-path resolution (never regex over go.mod text)"

key-files:
  created:
    - internal/indexer/discover.go
    - internal/indexer/discover_test.go
    - internal/indexer/doc.go
    - internal/indexer/testdata/gofixture/go.mod
    - internal/indexer/testdata/gofixture/pkga/pkga.go
    - internal/indexer/testdata/gofixture/pkga/embed.go
    - internal/indexer/testdata/gofixture/pkgb/pkgb.go
    - internal/indexer/testdata/gofixture/main.go
    - internal/indexer/testdata/gofixture/skip_linux.go
  modified: []

key-decisions:
  - "Discover returns (files, modulePath, err) rather than bundling modulePath onto every DiscoveredFile — computed once, kept out of the per-file struct the interfaces block specifies"
  - "Directory skip check compares d.Name() (not full path) against vendor/dot-prefix, guarded by p != root so a root path that itself starts with a dot is never self-skipped"
  - "Fixture's skip_linux.go coexists with main.go in package main at the fixture root — no declaration collision since skip_linux.go declares only the package clause, so the discovery test's GOOS-conditional assertion works identically on darwin (dev) and linux (CI)"

patterns-established:
  - "Shared multi-package Go fixture (example.com/gofixture) under testdata/, isolated by its own go.mod, as the single ground-truth source every extraction/resolution/determinism test in Phase 2 binds against"

requirements-completed: [LANG-01]

coverage:
  - id: D1
    description: "Discover walks a repo and returns only build-context-included .go files, in a stable RelPath-sorted order, skipping vendor/ and dot-prefixed directories"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_Fixture"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_Deterministic"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_SkipsVendorAndDotDirs"
        status: pass
    human_judgment: false
  - id: D2
    description: "Each discovered file carries its computed Go import path (module path + relative dir)"
    requirement: LANG-01
    verification:
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_ImportPaths"
        status: pass
      - kind: unit
        ref: "internal/indexer/discover_test.go#TestDiscover_MissingGoMod"
        status: pass
    human_judgment: false
  - id: D3
    description: "A committed multi-package Go fixture exercises functions, value+pointer methods, struct/interface embedding, intra- and cross-package calls, imports, and constants"
    requirement: LANG-01
    verification:
      - kind: other
        ref: "cd internal/indexer/testdata/gofixture && go vet ./..."
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-11
status: complete
---

# Phase 2 Plan 02: File Discovery Seam and Shared Test Fixture Summary

**Deterministic Pass-0 file discovery (`internal/indexer.Discover`) built on `go/build.Context.MatchFile` and `golang.org/x/mod/modfile`, plus the committed multi-package Go fixture every downstream Phase-2 indexer test binds against**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-11T01:31:00Z
- **Completed:** 2026-07-11T01:35:18Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Committed an isolated `example.com/gofixture` module (own `go.mod`, under `testdata/`) exercising exported/unexported functions, an intra-package unqualified call, value- and pointer-receiver methods, struct and interface embedding, a package-level constant and variable, a cross-package qualified call, and a GOOS-suffixed file for build-tag filtering — the ground truth for all Phase-2 extraction/resolution/determinism tests
- Implemented `internal/indexer.Discover(root) ([]DiscoveredFile, string, error)`: walks the repo via `filepath.WalkDir`, skips `vendor/` and dot-prefixed directories, includes a `*.go` file iff `go/build.Context.MatchFile(dir, name)` — called with each file's own parent directory, never a hoisted value (Pitfall 5) — reports true, tags each file with its computed Go import path via `golang.org/x/mod/modfile`-parsed module path, and returns results sorted by `RelPath` for determinism
- Verified via TDD: a RED `discover_test.go` (build failure — `Discover`/`DiscoveredFile` undefined) committed before the GREEN `discover.go` implementation
- `go build ./...`, `go vet ./internal/indexer/...`, and `go test ./... -count=1` all pass across the whole module

## Task Commits

Each task was committed atomically:

1. **Task 1: Multi-package Go test fixture** - `5ff6421` (feat)
2. **Task 2: File discovery seam — RED** - `9976b77` (test)
2. **Task 2: File discovery seam — GREEN** - `29a7ab2` (feat)

**Plan metadata:** (pending, this commit)

_Note: Task 2 is TDD — RED (`test`) commit precedes GREEN (`feat`) commit._

## Files Created/Modified
- `internal/indexer/testdata/gofixture/go.mod` - Isolated fixture module `example.com/gofixture`
- `internal/indexer/testdata/gofixture/pkga/pkga.go` - Exported/unexported functions, intra-package call, value/pointer receiver methods, const, var
- `internal/indexer/testdata/gofixture/pkga/embed.go` - Struct embedding (`Derived` embeds `Base`) and interface embedding (`ReadWriter` embeds `Reader`)
- `internal/indexer/testdata/gofixture/pkgb/pkgb.go` - Cross-package qualified call (`pkga.Alpha`) and method call on a freshly-constructed literal
- `internal/indexer/testdata/gofixture/main.go` - Entry point importing `pkgb`
- `internal/indexer/testdata/gofixture/skip_linux.go` - GOOS-suffixed file proving `MatchFile` filtering
- `internal/indexer/discover.go` - `DiscoveredFile` type + `Discover(root)` implementation
- `internal/indexer/discover_test.go` - Fixture/determinism/import-path/vendor-dot-dir/missing-go.mod test cases
- `internal/indexer/doc.go` - Package doc for `internal/indexer`

## Decisions Made
- `Discover` returns the module path as a second return value rather than embedding it per-file, keeping `DiscoveredFile` exactly the three fields (`AbsPath`, `RelPath`, `ImportPath`) the interfaces block specifies
- Directory-skip logic compares only `d.Name()` (never the full path) against `vendor`/dot-prefix, explicitly guarded so the walk root itself is never skipped even if its own basename happens to start with a dot
- Kept `skip_linux.go` and `main.go` both as `package main` at the fixture root — valid Go since `skip_linux.go` has no declarations to collide with, letting the discovery test assert GOOS-conditional inclusion/exclusion without a second fixture directory

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/indexer.Discover` and `DiscoveredFile` are ready for Pass 1 (extract) to consume as its file-iteration input
- The `example.com/gofixture` fixture is ready for the extractor/resolver/determinism tests in subsequent Phase-2 plans to bind against without redefining test data
- No blockers for subsequent Phase 2 plans

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-11*

## Self-Check: PASSED
