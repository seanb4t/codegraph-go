---
phase: 07-guards-that-cannot-fire
plan: 02
subsystem: testing
tags: [go, go-packages, archtest, dependency-direction, mutation-log, architecture-fitness]

# Dependency graph
requires:
  - phase: 07-guards-that-cannot-fire (plan 01)
    provides: "07-MUTATION-LOG.md header, cleanliness-gate convention, and family (a), ready for this plan to append family (b)"
provides:
  - "internal/query/archtest/import_direction_test.go: TestQueryImportsNoWireLayerOrIndexerRoot, a persisted go/packages archtest forbidding internal/query from resolving a wire-layer dependency (any variant) or the internal/indexer root (production scope), with zero-resolution guards and positive controls"
  - "07-MUTATION-LOG.md family (b): two verbatim RED transcripts, one per forbidden set, each with a cleanliness gate, exact mutation, and byte-clean revert"
  - "T-01-18 pending todo resolved and moved to .planning/todos/completed/"
affects: [07-guards-that-cannot-fire, phase-10-hlt-05-discovery-exclusion-helper]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 4296
  tasks: 3
  commits: 2
  plan_head_before: 815c015f7c99fc219a6446d7231c5f1cd85686cf

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "go/packages-based architecture-fitness test asserting over the resolved TRANSITIVE dependency set (packages.NeedDeps + a recursive walk over pkg.Imports), never a single-hop check or a source-text regex"
    - "Two-scope forbidden-import rule in one archtest: a wide rule checked over every loaded package variant (including Tests:true variants) alongside a narrower rule checked only over the production compilation unit, with the scoping documented in the package doc rather than treated as a silent exception"
    - "pkg.ID (not pkg.PkgPath) is the correct signal for distinguishing an internal-test-augmented go/packages variant from its true production counterpart when both share an identical PkgPath"

key-files:
  created:
    - internal/query/archtest/import_direction_test.go
  modified:
    - .planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md
    - .planning/todos/completed/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md (moved from .planning/todos/pending/)

key-decisions:
  - "D-01's two-scope split implemented as specified: the wire-layer rule runs over every loaded internal/query-rooted package variant; the internal/indexer-root rule runs over the production compilation unit only, since internal/query/engine_test.go legitimately imports the root to build fixtures. Documented in the package doc rather than treated as an unexplained exception."
  - "Fixed a variant-detection bug discovered during Task 1's own acceptance-criteria verification loop (Rule 1 — bug): this repo's golang.org/x/tools v0.48.0 gives the 'internal/query compiled with its own _test.go files' go/packages variant the SAME PkgPath as the true production variant — only pkg.ID carries the distinguishing ' [x.test]' suffix. The plan's literal 'raw PkgPath differs from its normalised path' test-variant heuristic (copied from the graphstore analog, where it never needed to distinguish production from test-augmented) does not by itself separate these two variants, so a naive port produced a false RED on the untouched tree (production internal/query 'resolving' the indexer root, when in fact only the test-augmented variant did, via engine_test.go). Fixed by computing isTestVariant from pkg.ID vs its own stripTestVariant() rather than from pkg.PkgPath, which correctly classifies all four go/packages variants (plain production, test-augmented, external _test package, synthesized .test binary) confirmed against a live debug dump of this module's actual load."
  - "connectrpc.com/connect (external, already an indirect go.mod dependency) was used for the wire-layer RED mutation rather than internal/uiserver or internal/mcp, per the plan's own reasoning: both first-party wire packages already import internal/query, so either would fail as a Go compile-time import cycle rather than through the archtest's own assertion, proving nothing about the guard."
  - "GRD-06's shared-ID gate (multiple sibling plans in this phase declare GRD-06) left that requirement checkbox unmarked by this plan — only GRD-02 was marked complete via requirements.mark-complete, since GRD-06 is not yet ready (07-03/07-04 have not produced their SUMMARY.md)."

requirements-completed: [GRD-02]

coverage:
  - id: D1
    description: "A persisted archtest in internal/query/archtest fails when internal/query acquires a wire-layer dependency (internal/uiserver, internal/mcp, internal/uiproto, or connectrpc.com/connect) anywhere in its resolved transitive dependency set, not merely direct imports"
    requirement: "GRD-02"
    verification:
      - kind: unit
        ref: "internal/query/archtest/import_direction_test.go#TestQueryImportsNoWireLayerOrIndexerRoot (RED demonstration b1, 07-MUTATION-LOG.md family (b))"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same archtest fails when internal/query's production compilation unit acquires a dependency on the internal/indexer ROOT package, while continuing to allow internal/indexer/goextract and internal/indexer/nodeid"
    requirement: "GRD-02"
    verification:
      - kind: unit
        ref: "internal/query/archtest/import_direction_test.go#TestQueryImportsNoWireLayerOrIndexerRoot (RED demonstration b2, 07-MUTATION-LOG.md family (b))"
        status: pass
    human_judgment: false
  - id: D3
    description: "The archtest reports the number of packages loaded and refuses to pass when that number is zero or when the production internal/query package is absent from the load"
    requirement: "GRD-02"
    verification:
      - kind: unit
        ref: "internal/query/archtest/import_direction_test.go (len(pkgs)==0 fatal; foundProductionQuery fatal; t.Logf reports loaded/resolved counts, confirmed via -v run: 'loaded 7 packages; production internal/query resolved 392 transitive dependencies')"
        status: pass
    human_judgment: false
  - id: D4
    description: "Positive controls: internal/graphstore AND internal/indexer/goextract present in the resolved set; internal/parser present but NOT a direct import, proving the walk resolved beyond one hop"
    requirement: "GRD-02"
    verification:
      - kind: unit
        ref: "internal/query/archtest/import_direction_test.go#TestQueryImportsNoWireLayerOrIndexerRoot (green run, GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The archtest package doc names threat T-01-18 and records why the internal/indexer-root rule is scoped to the production compilation unit, naming engine_test.go"
    requirement: "GRD-02"
    verification:
      - kind: other
        ref: "rg -c 'T-01-18|engine_test.go' internal/query/archtest/import_direction_test.go returns 5"
        status: pass
    human_judgment: false
  - id: D6
    description: "07-MUTATION-LOG.md family (b) carries two verbatim RED transcripts, one per forbidden set, each with a pre-mutation cleanliness gate, the exact mutation, the revert, and a byte-clean proof"
    requirement: "GRD-06"
    verification:
      - kind: other
        ref: "07-MUTATION-LOG.md family (b): both fenced transcripts diffed byte-identical against the captured scratch output (red-grd-02-wire.txt, red-grd-02-indexer.txt); rg -c '\\-\\-\\- FAIL: TestQueryImportsNoWireLayerOrIndexerRoot' returns 2"
        status: pass
    human_judgment: false

# Metrics
duration: 22min
completed: 2026-09-09
status: complete
---

# Phase 7 Plan 2: internal/query dependency-direction archtest Summary

**Persisted archtest `TestQueryImportsNoWireLayerOrIndexerRoot` in `internal/query/archtest` forbids `internal/query` from resolving a wire-layer dependency (any loaded variant) or the `internal/indexer` pipeline root (production compilation unit only), proven RED against both forbidden sets and logged verbatim in `07-MUTATION-LOG.md` family (b).**

## Performance

- **Duration:** 22 min
- **Started:** 2026-09-09T00:07:13Z
- **Completed:** 2026-09-09T00:29:00Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- New package `internal/query/archtest` (mirroring `internal/graphstore/archtest`) loads `internal/query`'s import graph via `golang.org/x/tools/go/packages` (`NeedDeps`, `Tests: true`) and walks `pkg.Imports` recursively (not a single-hop check) to build each candidate package's full resolved transitive dependency set.
- Wire-layer rule checks all four forbidden paths (`internal/uiserver`, `internal/mcp`, `internal/uiproto`, `connectrpc.com/connect`) over every loaded `internal/query`-rooted package variant, including `Tests: true` test variants — a wire import smuggled into a `_test.go` file is caught, not invisible.
- Indexer-root rule forbids the exact path `internal/indexer` (never a prefix match, so `internal/indexer/goextract` and `internal/indexer/nodeid` stay allowed) over the production compilation unit only, since `internal/query/engine_test.go` legitimately imports the root today to build fixtures — documented in the package doc, not silently excepted.
- Zero-resolution guards (`len(pkgs) == 0` fatal; production `internal/query` package absent from the load is a separate fatal) and four positive controls (`internal/graphstore` present, `internal/indexer/goextract` present, `internal/parser` present-but-not-direct proving the walk resolves beyond one hop, and a `t.Logf` reporting loaded/resolved counts) make a degenerate load fail loud rather than pass vacuously.
- Both forbidden sets proven RED against a real import in `internal/query/traverse.go` (`connectrpc.com/connect` for the wire layer, `internal/indexer` for the root), each reverted byte-clean, with verbatim transcripts pasted into `07-MUTATION-LOG.md` family (b).
- T-01-18 pending todo closed: moved to `.planning/todos/completed/`, `status: resolved`, `resolved_by_phase: 7`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write the internal/query dependency-direction archtest** - `980d2dd9` (test)
2. **Task 2: RED — two forbidden-import mutations, each watched fail and reverted byte-clean** - no commit (verification-only task; both mutations were reverted byte-clean per the plan's own instruction not to commit either mutation or leave a scratch edit in the tree — the transcripts it produced feed Task 3's commit)
3. **Task 3: Append mutation-log family (b) and close the T-01-18 todo** - `5568e1cf` (docs)

**Plan metadata:** commit created by this SUMMARY's own atomic write+commit step.

## Files Created/Modified
- `internal/query/archtest/import_direction_test.go` - New archtest package: `TestQueryImportsNoWireLayerOrIndexerRoot`, `transitiveDeps`, `stripTestVariant`, forbidden/allowed import-path constants
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` - Appended family (b): two verbatim RED transcripts, cleanliness gates, reverts, green re-run
- `.planning/todos/completed/2026-09-07-internal-query-dependency-direction-has-no-persisted-archtest.md` - Moved from `.planning/todos/pending/`, marked resolved, resolution recorded

## Decisions Made
- D-01's two-scope split implemented exactly as specified (see `key-decisions` in frontmatter for the full reasoning): wire-layer over every variant, indexer-root over production only, with `engine_test.go` named as the reason in both the package doc and this SUMMARY.
- Fixed a real variant-detection bug found during Task 1's own acceptance-criteria loop (Rule 1): `pkg.ID`, not `pkg.PkgPath`, is what distinguishes an internal-test-augmented `go/packages` variant from true production in this repo's `golang.org/x/tools` version — see frontmatter `key-decisions` for the empirical confirmation and the fix.
- Used `connectrpc.com/connect` (external) rather than a first-party wire package for the b1 RED mutation, since `internal/uiserver`/`internal/mcp` both import `internal/query` and would fail as a compile-time cycle instead of through the guard's assertion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed test-variant detection using pkg.ID instead of pkg.PkgPath**
- **Found during:** Task 1 (acceptance-criteria verification loop — `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/` failed on the very first run, on the untouched tree)
- **Issue:** The plan's literal instruction ("treat a package as a test variant when its raw PkgPath differs from its normalised path") is what the `internal/graphstore/archtest` analog does, and works there because that test applies the identical rule to every variant. GRD-02 needs the indexer-root rule to distinguish production from test-augmented variants, and this repo's `golang.org/x/tools v0.48.0` gives the "internal/query compiled with its own `_test.go` files" go/packages variant the exact same `PkgPath` as the true production variant (`"github.com/seanb4t/codegraph-go/internal/query"` for both) — only `pkg.ID` carries the distinguishing `" [internal/query.test]"` suffix. A naive PkgPath-based check therefore could not tell them apart, so the archtest failed on the unmodified tree, reporting the production package as resolving the forbidden `internal/indexer` root — when in reality only the test-augmented variant (via `engine_test.go`) did.
- **Fix:** Computed `isTestVariant` from `pkg.ID != stripTestVariant(pkg.ID)` instead of from `pkg.PkgPath`. Verified against a live debug dump of all seven loaded packages/variants for this module (production, test-augmented, external `_test` package, and synthesized `.test` binary) that this correctly classifies all four shapes.
- **Files modified:** `internal/query/archtest/import_direction_test.go`
- **Verification:** `GOTOOLCHAIN=go1.26.6 go test -count=1 -v ./internal/query/archtest/` passes green on the untouched tree; both RED demonstrations in Task 2 (which specifically target the production-vs-test-scope boundary) produce the correct, discriminating failure.
- **Committed in:** `980d2dd9` (Task 1 commit — the fix landed before the task's own commit, since it was found and fixed within Task 1's own acceptance-criteria gate, not after)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Necessary for correctness — without this fix the archtest would either never compile/pass on the current tree, or would silently fail to distinguish the production-scope boundary D-01 requires. No scope creep; the plan's own artifact list and acceptance criteria are unchanged.

## Issues Encountered

None beyond the deviation above (itself caught and fixed within the plan's own acceptance-criteria gate).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `07-MUTATION-LOG.md` now carries families (a) and (b), three verbatim RED transcripts total, ready for 07-03 (GRD-03) and 07-04 (GRD-04/GRD-05) to append families (c) and (d) plus the GRD-05 deletion record.
- GRD-02 marked complete in REQUIREMENTS.md. GRD-06 remains open (shared across 07-01/02/03/04's mutation-log contributions) — will be marked complete once the last plan declaring it produces its SUMMARY.md, per the shared-ID gate.
- `internal/query/traverse.go` is byte-identical to its pre-plan state; no module dependency was added (`git diff --quiet -- go.mod go.sum` confirmed).
- Phase 10's HLT-05 discovery-exclusion helper now has a machine-enforced boundary: it may not be imported from `internal/query`, since doing so would resolve the forbidden `internal/indexer` root transitively and fail this archtest.
- No blockers for 07-03 (GRD-03, dry-run-signed injection guard).

---
*Phase: 07-guards-that-cannot-fire*
*Completed: 2026-09-09*

## Self-Check: PASSED

All created/modified files found on disk; both task commits (980d2dd9, 5568e1cf) found in git log. Plan produced 2 commits against `plan_head_before` 815c015f (measured via `git rev-list --count`), matching the two atomic-work commits listed above (Task 2 legitimately produced no commit, as designed).
