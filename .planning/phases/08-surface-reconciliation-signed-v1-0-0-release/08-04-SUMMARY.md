---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 04
subsystem: api
tags: [go, bfs, traversal, query-engine, affected, tdd]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: "08-01's defaultDepth=2 impact clamp (the sibling constant this plan deliberately does NOT reuse)"
provides:
  - "Engine.Affected(files, depth) depth-bounded BFS with TS 1.3.1 test-files-as-leaves pruning"
  - "defaultAffectedDepth=5 / clampAffectedDepth engine constant, distinct from impact's defaultDepth=2"
affects: ["08-05 (affected CLI --depth/--stdin/--filter/--quiet flag wiring)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Affected's BFS frontier/next-frontier loop shape mirrors Impact's (traverse.go) exactly, but replaces 'expand everything' with 'test dependents are leaves'"

key-files:
  created: []
  modified:
    - internal/query/validate.go
    - internal/query/traverse.go
    - internal/query/traverse_test.go
    - internal/cli/affected.go

key-decisions:
  - "clampAffectedDepth is a standalone mirror of clampDepth (not a shared clampDepthWithDefault helper) — simplest, lowest-risk option from the plan's either/or menu"
  - "affected.go's call site updated to eng.Affected(args, 0) as a minimal compile fix only; full --depth/--stdin/--filter/--quiet flag wiring is explicitly deferred to 08-05"
  - "visited-set now marks BFS seed node IDs up front (mirroring Impact's own seed-visited convention) to prevent cyclic re-expansion across multiple hops — the old single-hop version had no need for this since it never expanded past hop 1"

requirements-completed: [SURF-04]

coverage:
  - id: D1
    description: "defaultAffectedDepth=5 constant + clampAffectedDepth(n) clamp, distinct from impact's defaultDepth=2/clampDepth, sharing MaxDepth=50 ceiling"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestClampAffectedDepth"
        status: pass
    human_judgment: false
  - id: D2
    description: "Engine.Affected(files, depth) depth-bounded BFS with TS test-files-as-leaves pruning (test dependents recorded but not expanded; non-test dependents keep expanding)"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffectedDepthBFSWithTestLeafPruning"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffected"
        status: pass
    human_judgment: false
  - id: D3
    description: "Negative depth rejected, empty-files no-op, dangling reverse edge mid-BFS skipped not aborted"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffectedNegativeDepthRejected"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffectedEmptyFilesReturnsEmptyResultNoError"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestAffectedSkipsDanglingEdgeInsteadOfFailing"
        status: pass
  - id: D4
    description: "Affected output ordering determinism across runs (backstop truth — not asserted as covered by this plan's own tests)"
    requirement: "SURF-04"
    verification: []
    human_judgment: true
    rationale: "Plan's own <verification> section marks this a backstop truth surfaced for human confirmation in validate-phase, not something this plan's automated tests assert as covered."

duration: 6min
completed: 2026-07-19
status: complete
---

# Phase 8 Plan 4: Affected depth-bounded BFS with test-leaf pruning Summary

**`Engine.Affected` rewritten from a one-hop reverse-adjacency lookup into a real depth-bounded BFS with TS 1.3.1's test-files-as-leaves pruning, plus a new `defaultAffectedDepth=5` constant deliberately distinct from impact's `defaultDepth=2`.**

## Performance

- **Duration:** 6 min
- **Completed:** 2026-07-19
- **Tasks:** 2 completed
- **Files modified:** 4

## Accomplishments
- `Engine.Affected(files []string, depth int)` is now a genuine BFS over the reverse-adjacency map, bounded by `clampAffectedDepth(depth)`, mirroring `Impact`'s frontier/next-frontier loop shape
- Test-files-as-leaves semantics ported from TS 1.3.1: a dependent passing `isTestSymbol` is recorded as an affected test AND is a leaf — never queued for further expansion; non-test dependents keep expanding and are never themselves recorded
- New `defaultAffectedDepth = 5` constant + `clampAffectedDepth` clamp added beside `defaultDepth`/`clampDepth`, sharing the `MaxDepth=50` ceiling but never silently reusing impact's now-2 default
- `internal/cli/affected.go`'s sole call site updated to `eng.Affected(args, 0)` — package compiles; full `--depth` flag wiring deferred to 08-05 as scoped

## Task Commits

Each task was committed atomically (TDD RED/GREEN):

1. **Task 1: defaultAffectedDepth + clampAffectedDepth** — `f602e6a` (test, RED, shared with Task 2's tests), `872d168` (feat, GREEN)
2. **Task 2: Affected → depth-bounded BFS with test-leaf pruning** — `da5adc4` (feat, GREEN)

**Plan metadata:** commit pending (this SUMMARY + STATE/ROADMAP update)

_Note: Task 1 and Task 2's RED tests were both added in the single `f602e6a` test commit since they live in the same `traverse_test.go` file with no independently-buildable intermediate state between them (Go's whole-package compilation model — same precedent as prior phases' combined-commit decisions). Task 1's GREEN (`872d168`, the constants alone) does not make the package build on its own, since the RED test commit already references `Affected(files, depth)`'s new signature; the package only builds again once Task 2's GREEN (`da5adc4`) lands the BFS rewrite. This mirrors the documented 02-status-content-git-worktree-awareness precedent for combined Task GREEN commits under Go's whole-package model._

## Files Created/Modified
- `internal/query/validate.go` — added `defaultAffectedDepth=5` constant and `clampAffectedDepth` clamp function
- `internal/query/traverse.go` — rewrote `Affected` as a depth-bounded BFS with test-leaf pruning
- `internal/query/traverse_test.go` — updated `TestAffected`'s call site to the new 2-arg signature; added `TestClampAffectedDepth`, `TestAffectedDepthBFSWithTestLeafPruning` (4 subtests), `TestAffectedNegativeDepthRejected`, `TestAffectedEmptyFilesReturnsEmptyResultNoError`, `TestAffectedSkipsDanglingEdgeInsteadOfFailing`; fixed one additional pre-existing call site (`Affected zero-match` subtest) discovered mid-implementation
- `internal/cli/affected.go` — updated the sole `eng.Affected(...)` call site to pass `0` for depth (defers to `defaultAffectedDepth`)

## Decisions Made
- `clampAffectedDepth` implemented as a standalone function mirroring `clampDepth`'s body (not a shared `clampDepthWithDefault(n, def)` helper) — the plan offered either as acceptable; the standalone mirror was the lower-risk, more legible option and keeps `clampDepth`/`clampAffectedDepth` each trivially auditable in isolation
- BFS seed nodes are now marked `visited` up front (mirroring `Impact`'s own convention of marking its single seed node visited) — a deliberate addition beyond the old single-hop code's behavior, needed once `Affected` genuinely expands across multiple hops, to prevent a cyclic call graph from re-visiting a changed file's own seed symbol
- `affected.go`'s call site passes a literal `0` rather than threading any new flag — the plan explicitly scopes full `--depth`/`--stdin`/`--filter`/`--quiet` flag wiring to 08-05; this plan's CLI touch is the minimal compile fix only

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking compile error] A second, previously-missed `Affected` call site in the `Affected zero-match` subtest**
- **Found during:** Task 2 GREEN verification (`go test ./internal/query/... -run TestAffected`)
- **Issue:** `traverse_test.go` had a second pre-existing call to `engine.Affected([]string{"nonexistent-file-with-no-callers.go"})` (single-arg) inside an unrelated `Callers`/`Affected` zero-match test block that the plan's `<read_first>` line-range citation (485-538) did not cover — the signature-change compile error surfaced it.
- **Fix:** Updated the call site to pass an explicit depth (`2`).
- **Files modified:** `internal/query/traverse_test.go`
- **Verification:** `go build ./...` and `go test ./internal/query/... -count=1` both green afterward.
- **Committed in:** `da5adc4` (part of Task 2's GREEN commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking compile error)
**Impact on plan:** Purely a missed call-site discovered by the compiler; no scope creep, no design change.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `Engine.Affected(files []string, depth int)` and `clampAffectedDepth`/`defaultAffectedDepth` are ready for 08-05 to wire `--depth` (and `--stdin`/`--filter`/`--quiet`) into `internal/cli/affected.go`'s cobra flags.
- No blockers. `go build ./...` and `go test ./internal/query/... -count=1` both green; no regressions in `Impact`/`Callers`/`Callees`.
- The plan's backstop truth (Affected output-ordering determinism) is intentionally left for human confirmation during a later `validate-phase` pass, per the plan's own `<verification>` section — not a gap introduced by this execution.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*
