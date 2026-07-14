---
phase: 04-incremental-sync-file-watcher
plan: 04
subsystem: database
tags: [pebble, graphstore, incremental-index, sync, determinism, tdd, go]

# Dependency graph
requires:
  - phase: 04-incremental-sync-file-watcher
    provides: "Plan 04-03's internal/indexer.Sync(repoRoot, storeDir, opts) (Stats, error) — the x/-index prune + dependent-recomputation incremental engine this plan's fixture suite and determinism gate validate"
provides:
  - "internal/indexer/prune_fixtures_test.go — TestPruneFixtures, the INDX-04 acceptance gate: create/modify/delete/rename/move each leave the graph with no orphaned nodes and no dangling edges, verified against a dedicated cross-file-call + cross-file-receiver-method fixture"
  - "internal/indexer/sync_determinism_test.go — TestSyncEqualsReindex, the INDX-03 determinism gate: an incremental Sync that touches every file lands a graph byte-identical to a from-scratch Run over the same final tree (mod Meta.last_sync_unix_ms)"
  - "internal/indexer/testdata/prunefixture — a self-contained, mutable-safe subfixture (Foo/UseFoo cross-file call pair; Widget/Describe/CallDescribe cross-file receiver-method triple) distinct from the shared testdata/gofixture tree several other tests assert an exact file list against"
  - "internal/indexer.copyFixture(t, srcRoot) string — the shared per-subtest fixture-copy helper both this plan's tests reuse"
affects: [04-05, 04-06, 04-07, 04-08, 04-09, watcher, daemon, cli-sync]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dedicated mutable subfixture (testdata/prunefixture) instead of extending the shared testdata/gofixture tree — avoids breaking discover_test.go's TestDiscover_Fixture exact-RelPath-list assertion and determinism_test.go's node-kind/edge expectations, which the plan's own action text explicitly permitted (\"add source files or a dedicated subfixture\")"
    - "copyFixture(t, srcRoot) walks a testdata tree into a fresh t.TempDir() so create/modify/delete/rename/move subtests can freely mutate on-disk files without disturbing the source fixture or sibling subtests"
    - "assertNoOrphansOrDangling(t, r): the shared INDX-04 invariant — every edge's Source/Target resolve via GetNode, every x/ index node entry (for every currently-recorded file) resolves via GetNode"
    - "touchEveryFile(t, root): forces every file through Sync's D-01a \"modified\" classification via a symbol-identity-preserving edit (trailing no-op comment) that changes the content hash without changing any node's (kind, qualifiedName, filePath) identity tuple — a byte-identical rewrite is filtered out entirely by D-01a's content-hash confirm and would never enter the reparse batch"

key-files:
  created:
    - internal/indexer/prune_fixtures_test.go
    - internal/indexer/sync_determinism_test.go
    - internal/indexer/testdata/prunefixture/go.mod
    - internal/indexer/testdata/prunefixture/pkg/a.go
    - internal/indexer/testdata/prunefixture/pkg/b.go
    - internal/indexer/testdata/prunefixture/pkg/types.go
    - internal/indexer/testdata/prunefixture/pkg/methods.go
    - internal/indexer/testdata/prunefixture/pkg/caller.go
  modified: []

key-decisions:
  - "Used a dedicated internal/indexer/testdata/prunefixture subfixture rather than extending the shared testdata/gofixture tree named in the plan's files_modified list — discover_test.go's TestDiscover_Fixture asserts an EXACT sorted RelPath list against testdata/gofixture; adding files there would have broken that test (and risked perturbing determinism_test.go's node-kind/edge assertions). The plan's own action text explicitly authorized this alternative (\"add source files or a dedicated subfixture; keep it a valid module with go.mod\")."
  - "Re-scoped the MODIFY subtest from the plan's literal \"change a called func's body (its id changes)\" to \"rename a symbol within its file\" — nodeid.NodeID (Phase-2 D-02a, internal/indexer/nodeid/nodeid.go) hashes (kind, qualifiedName, filePath) only, never source content, so a body-only edit leaving the name unchanged cannot change a node's id. Renaming Foo->Foo2 within pkg/a.go is the modify that genuinely changes identity, making \"the old id has no node and nothing targets it\" a real, provable assertion rather than a vacuous one."
  - "Re-scoped MOVE to relocate a caller/callee PAIR together into a new subdirectory (pkg/sub/), not just the callee alone — internal/indexer/resolve.go's unqualified-call resolution is keyed by directory-derived import path (symbolindex.go's byImportAndName), so moving only Foo across a directory boundary crosses a real Go package boundary and correctly leaves UseFoo's unqualified reference unresolved (matching real Go semantics, not a prune bug). Moving both files together keeps them in the same (relocated) package, so the edge regenerates cleanly and the no-orphans/no-dangling invariant is exercised on a genuine cross-directory move rather than degenerating into a same-directory rename."
  - "TestSyncEqualsReindex's \"touch every file\" step appends a no-op trailing comment rather than rewriting files with byte-identical content — D-01a's content-hash confirm (Plan 04-03) explicitly SKIPS a file whose hash matches the stored value even after an mtime bump, so a literal byte-identical rewrite would never enter Sync's modified/reparse batch at all, defeating the property test's purpose. A content change that doesn't move any symbol's identity tuple forces the classification while keeping the resulting graph shape identical to a from-scratch build."

patterns-established:
  - "Prune-invariant fixture suites live in their own mutable subfixture directory (testdata/<purpose>fixture) rather than extending shared, assertion-pinned fixtures (testdata/gofixture) — copyFixture(t, srcRoot) is the reusable per-subtest isolation mechanism."

requirements-completed: [INDX-04, INDX-03]

coverage:
  - id: D1
    description: "CREATE: a new file calling an existing symbol lands its node + calls edge with no dangling edge"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/indexer/prune_fixtures_test.go#TestPruneFixtures/create"
        status: pass
    human_judgment: false
  - id: D2
    description: "MODIFY: a symbol-renaming edit prunes the old node id completely — nothing targets it, no dangling edge"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/indexer/prune_fixtures_test.go#TestPruneFixtures/modify"
        status: pass
    human_judgment: false
  - id: D3
    description: "DELETE: removing a file prunes its nodes, edges, File record, and x/ index entries entirely"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/indexer/prune_fixtures_test.go#TestPruneFixtures/delete"
        status: pass
    human_judgment: false
  - id: D4
    description: "RENAME: a same-directory rename prunes the old path's File/x-index/node and regenerates the caller's edge against the new path's id"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/indexer/prune_fixtures_test.go#TestPruneFixtures/rename"
        status: pass
    human_judgment: false
  - id: D5
    description: "MOVE: a cross-directory move of a caller/callee pair prunes both old records and regenerates the edge cleanly at the new path's ids, with no orphans/dangling edges"
    requirement: "INDX-04"
    verification:
      - kind: unit
        ref: "internal/indexer/prune_fixtures_test.go#TestPruneFixtures/move"
        status: pass
    human_judgment: false
  - id: D6
    description: "An incremental Sync touching every file yields a graph byte-identical to a from-scratch Run over the same final tree, after normalizing Meta.last_sync_unix_ms, under -race and GOMAXPROCS=8"
    requirement: "INDX-03"
    verification:
      - kind: unit
        ref: "internal/indexer/sync_determinism_test.go#TestSyncEqualsReindex"
        status: pass
    human_judgment: false
  - id: D7
    description: "Whole-repo build/vet/race-test suite stays green after the new fixture suite and determinism gate land"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... && go test ./... -count=1"
        status: pass
      - kind: unit
        ref: "go test ./internal/indexer/... -race -count=1"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-07-11
status: complete
---

# Phase 4 Plan 4: Rename/Delete/Move Prune Fixture Suite + Sync-Equals-Reindex Determinism Gate Summary

**TestPruneFixtures (create/modify/delete/rename/move, no-orphans/no-dangling invariant) and TestSyncEqualsReindex (touch-every-file Sync byte-identical to from-scratch Run) both pass green against Plan 04-03's existing Sync() implementation — no prune or determinism bug found.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-11T19:25:00Z
- **Completed:** 2026-07-11T19:37:05Z
- **Tasks:** 2
- **Files modified:** 8 (7 created, 0 modified)

## Accomplishments
- `internal/indexer/prune_fixtures_test.go`'s `TestPruneFixtures` — the INDX-04 acceptance gate (ROADMAP success criterion 2): five subtests (create/modify/delete/rename/move) each assert, via the shared `assertNoOrphansOrDangling` helper, that every edge's endpoints resolve to a live node and every x/ index node entry resolves via `GetNode`
- `internal/indexer/testdata/prunefixture` — a dedicated, mutation-safe subfixture with a cross-file call pair (`pkg/a.go`'s `Foo` <- `pkg/b.go`'s `UseFoo`) and a cross-file receiver-method triple (`pkg/types.go`'s `Widget` <- `pkg/methods.go`'s `Describe` <- `pkg/caller.go`'s `CallDescribe`)
- `internal/indexer/sync_determinism_test.go`'s `TestSyncEqualsReindex` — the INDX-03 determinism bar surviving incremental update: a `Sync` that reparses every file (via `touchEveryFile`'s symbol-identity-preserving edit) lands a graph byte-identical to a from-scratch `Run` over the same final tree, verified under `-race` and `GOMAXPROCS(8)`
- `copyFixture(t, srcRoot) string` — the shared per-subtest fixture-isolation helper both new test files use
- All 6 new subtests (5 prune + 1 determinism) pass on first run against the unmodified Plan 04-03 `sync.go` — the fixture suite is genuine validation-first evidence that the existing prune/determinism implementation is correct, not a rubber stamp

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED, landed GREEN): prune fixture suite** - `1e86ed8` (test)
2. **Task 2 (GREEN): sync-equals-reindex determinism gate** - `efc47b1` (test)

_TDD plan: Task 1's RED framing (expecting the suite to possibly surface a partial-prune bug) landed GREEN on first run — Plan 04-03's prune implementation already handles all five file operations correctly. No `fix(04-04)` commit was needed for either task; both tests passed against the unmodified `sync.go`._

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/indexer/prune_fixtures_test.go` - New: `TestPruneFixtures` (5 subtests), `copyFixture`, `assertNoOrphansOrDangling`, `assertFileIndexEmpty`
- `internal/indexer/sync_determinism_test.go` - New: `TestSyncEqualsReindex`, `exportNormalized`, `touchEveryFile`
- `internal/indexer/testdata/prunefixture/go.mod` - New: dedicated subfixture module
- `internal/indexer/testdata/prunefixture/pkg/a.go` - New: `Foo` — the create/modify/delete/rename/move subject
- `internal/indexer/testdata/prunefixture/pkg/b.go` - New: `UseFoo` — cross-file caller of `Foo`
- `internal/indexer/testdata/prunefixture/pkg/types.go` - New: `Widget` struct declaration
- `internal/indexer/testdata/prunefixture/pkg/methods.go` - New: `Widget.Describe` (receiver type declared in a different file)
- `internal/indexer/testdata/prunefixture/pkg/caller.go` - New: `CallDescribe` (cross-file receiver-method call)

## Decisions Made
- **Dedicated `testdata/prunefixture` subfixture** instead of extending the shared `testdata/gofixture` tree the plan's `files_modified` listed — `discover_test.go`'s `TestDiscover_Fixture` asserts an exact sorted file list against `testdata/gofixture`; extending it would have broken that test. The plan's action text explicitly authorized "a dedicated subfixture" as an alternative.
- **MODIFY subtest re-scoped to a within-file rename** (`Foo` -> `Foo2`) rather than a body-only content edit — `nodeid.NodeID` (Phase-2 D-02a) hashes `(kind, qualifiedName, filePath)` only, never source content, so a body-only edit cannot change a node's id. This is the modify that genuinely exercises "the old id has no node and nothing targets it."
- **MOVE subtest re-scoped to relocate a caller/callee pair together** into a new subdirectory, not the callee alone — unqualified-call resolution is import-path-scoped (directory-derived), so moving only the callee crosses a real Go package boundary and correctly leaves the caller's reference unresolved (matching real Go semantics). Moving the pair together keeps them in the same relocated package so the edge regenerates and the invariant is exercised meaningfully.
- **`TestSyncEqualsReindex`'s "touch every file" step uses a symbol-identity-preserving edit** (trailing no-op comment), not a byte-identical rewrite — Plan 04-03's D-01a content-hash confirm explicitly skips a file whose hash still matches after an mtime bump, so a literal identical rewrite would never enter the reparse batch, defeating the test's purpose of exercising Sync's real write path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `move` subtest write helper missing parent directory creation**
- **Found during:** Task 1 (`TestPruneFixtures/move` first run)
- **Issue:** `writeFile` (an existing `sync_test.go` helper) does not `MkdirAll` its target's parent directory (unlike `writeFixture`), so writing into a not-yet-created `pkg/sub/` subdirectory failed with `open ... pkg/sub/a.go: no such file or directory`.
- **Fix:** Added an explicit `os.MkdirAll(filepath.Join(repoRoot, "pkg", "sub"), 0o755)` before the per-file `writeFile` calls in the `move` subtest.
- **Files modified:** `internal/indexer/prune_fixtures_test.go`
- **Verification:** `go test ./internal/indexer/... -run TestPruneFixtures -race -count=1` green (all 5 subtests)
- **Committed in:** `1e86ed8` (Task 1 commit — caught and fixed before commit, so no separate fix commit needed)

---

**Total deviations:** 1 auto-fixed (1 bug, test-scaffolding only — no `sync.go` production code changed)
**Impact on plan:** Neither task required a correctness fix to `sync.go`; Plan 04-03's prune and determinism implementation is already correct against this plan's acceptance gate. No scope creep.

### Notable Findings (not deviations — documentation clarifications)

- **04-CONTEXT.md's D-03 justification ("a symbol whose content is unchanged keeps a stable n/id, so a pure move does not churn inbound edges") does not hold literally against `nodeid.NodeID`'s actual implementation** (`internal/indexer/nodeid/nodeid.go`): the id hash preimage is `(kind, qualifiedName, filePath)` — `filePath` is part of the identity tuple, so ANY rename or move necessarily changes every affected symbol's node id, confirmed directly by this plan's own tests (`oldFooID`/`newFooID` differ in both the `rename` and `move` subtests) and consistent with Plan 04-03's own `TestSyncReExtractsDependents`. What DOES hold, and is what this plan's tests actually prove, is the practically important property: **the resulting graph is coherent** — no orphaned nodes, no dangling edges, and (when caller/callee move/stay together) inbound edges regenerate cleanly at the new ids with no unresolved churn beyond the identity change itself. This is a documentation-accuracy note for a future CONTEXT.md pass, not a code defect — `sync.go`'s delete(old)+add(new) model (D-03's actual mechanism) is exactly correct.

## Issues Encountered
None beyond the test-scaffolding fix documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- INDX-04's acceptance gate (no orphans/no dangling edges across create/modify/delete/rename/move) and INDX-03's determinism bar (sync-equals-reindex byte-identical Export) are both proven against the current `internal/indexer.Sync` implementation — Plans 04-05 (watcher), 04-06 (MCP reconnect), 04-07 (daemon), 04-08 (`codegraph sync` CLI), and 04-09 can build on `Sync` with these invariants already validated, no further prune-correctness work pending.
- The `testdata/prunefixture` subfixture and `copyFixture`/`assertNoOrphansOrDangling`/`assertFileIndexEmpty` helpers are available for reuse by any later plan that needs to exercise file-operation prune correctness (e.g., watcher-driven Sync calls in 04-05).
- The CONTEXT.md D-03 "stable ids on move" documentation note above is worth a future correction pass but does not block any downstream plan — no plan in this phase's remaining scope depends on literal node-id stability across a rename/move.

---
*Phase: 04-incremental-sync-file-watcher*
*Completed: 2026-07-11*

## Self-Check: PASSED

All 8 key files and both task commits (1e86ed8 test, efc47b1 test) verified present.
