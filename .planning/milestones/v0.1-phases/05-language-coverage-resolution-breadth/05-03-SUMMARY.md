---
phase: 05-language-coverage-resolution-breadth
plan: 03
subsystem: indexing
tags: [resolution, symbol-index, determinism, go, tdd]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    plan: 01
    provides: LanguageSpec registry, ProjectDescriptor interface, per-language ModuleKey hook seam
  - phase: 05-language-coverage-resolution-breadth
    plan: 02
    provides: DiscoveredFile.ImportPath now carrying a per-language ModuleKey (Go's importPathFor is the first instance), generalized Discover/Extract dispatch
provides:
  - Per-language module-key resolution seam (symbolIndex.byModuleKeyAndName, doc comments generalized off Go-specific "importPath" language)
  - WR-01 fix — deterministic, order-independent same-(moduleKey, name) collision resolution (lowest node Id wins) with a Collisions counter, applied consistently in both overlay() and newSymbolIndexFromStore()
  - WR-02 fix — a selector call on a non-identifier operand (foo().Bar()) can no longer mis-resolve as a same-package unqualified reference
  - A documented, evidence-based finding that the "call-as-argument extraction gap" (252e2sav94) does not reproduce against the current goextract.go — locked in via regression tests, no code change needed
affects: [05-04, 05-05, every Wave-4 per-language extractor/resolver inheriting symbolindex.go/resolve.go's shape]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "addSymbol(names, name, id) helper: the single collision-resolution chokepoint both overlay() (fresh/reparse path) and newSymbolIndexFromStore() (store-seeded path) call, so Sync and from-scratch index-building can never disagree on which node wins a same-(moduleKey, name) collision"
    - "Synthetic non-matching PkgAlias (\"<\" + operand.Kind() + \">\") to force a selector ref through resolveSelector's existing alias-membership boundary instead of leaking an empty PkgAlias that resolveNameRef would misroute to resolveUnqualified"

key-files:
  created:
    - internal/indexer/symbolindex_test.go
  modified:
    - internal/indexer/symbolindex.go
    - internal/indexer/resolve_test.go
    - internal/indexer/goextract/goextract.go
    - internal/indexer/goextract/goextract_test.go

key-decisions:
  - "WR-01 tie-break mirrors query.Engine.resolveSymbolNode's own lowest-Id-wins pattern (internal/query/traverse.go) rather than inventing a new determinism rule — one canonical tie-break convention project-wide"
  - "Collisions is tracked as a symbolIndex.Collisions int field (incremented in addSymbol) but NOT wired through to Sync's Stats/CLI --verbose output — that plumbing touches sync.go and cmd/, both outside this plan's files_modified scope. The count exists and is tested/inspectable now; wiring it to user-facing output is a follow-up, not a silent drop (D-06a intent satisfied within this plan's scope)."
  - "WR-02's fix lives entirely in goextract.go's recordCall, not resolve.go — resolveSelector's alias-membership boundary is the correct existing mechanism to route a doomed-to-be-unresolved selector call through; the fix only needed to stop the extractor from emitting an empty PkgAlias that skipped that boundary"
  - "Investigated (not assumed) the call-as-argument gap before touching code: 7 combinations of outer/inner call shapes (identifier/selector, single/multi-arg, 2-deep nesting, binary-expr argument) all extract and resolve correctly against current goextract.go/resolve.go. Reproduced the EXACT golden-corpus shape from testdata/golden/golden_parity_test.go's documented Impact-test discrepancy (finish.AddCommand(a.newFinishOpenCmd(), a.newFinishReconcileCmd())) — it also extracts correctly at the goextract level. The residual golden gap traces to Go's receiver-based method dispatch (\"a\" is a method receiver, not a real import alias — same ratified 'no local-variable type tracking' limitation as TestResolve_UnresolvedMethodCall), not to call-as-argument extraction. Per the TDD fail-fast rule ('the feature may already exist... do not skip RED by proceeding with a passing test'), this was investigated thoroughly rather than skipped — the conclusion is that D-05's third item was already satisfied by goextract.go's existing walkDescendants design (it unconditionally descends into every node's children, so nested call_expressions in argument position were never actually dropped). No code change was made for this item; only regression tests were added to lock the correct behavior in."

patterns-established: []

# LANG-02..05 remain unchecked in REQUIREMENTS.md: this plan lands only the
# per-language resolution seam (ModuleKey-keyed symbol index) and the three
# deferred Go fixes (D-05) — no Java/C#/Python/TS extraction or resolution
# exists yet. Full per-language requirement completion happens in the
# Wave-4 plans that consume this seam.
requirements-completed: []

coverage:
  - id: D1
    description: "symbolIndex is keyed by a per-language module key (ModuleKey) rather than Go's importPath concept — the two-level map structure (byModuleKeyAndName) is unchanged, only the outer-key computation generalizes; doc comments across symbolindex.go updated off Go-specific language"
    verification:
      - kind: unit
        ref: "internal/indexer/resolve_test.go (all TestResolve_* cases pass unchanged against the renamed field)"
        status: pass
    human_judgment: false
  - id: D2
    description: "WR-01: a same-(moduleKey, name) collision between two files' declared symbols resolves to a deterministic, order-independent target (lowest node Id wins) instead of silently last-write-wins overwriting; the collision is counted, never silently dropped"
    verification:
      - kind: unit
        ref: "internal/indexer/symbolindex_test.go#TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent"
        status: pass
      - kind: unit
        ref: "internal/indexer/symbolindex_test.go#TestSymbolIndex_NoCollisionForDistinctNames"
        status: pass
      - kind: unit
        ref: "internal/indexer/symbolindex_test.go#TestSymbolIndex_ReOverlaySameIDNotACollision"
        status: pass
    human_judgment: false
  - id: D3
    description: "WR-02: a selector call whose operand is a non-identifier expression (foo().Bar()) never mis-resolves as a same-package unqualified reference to an unrelated same-named symbol"
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_SelectorNonIdentifierOperandNeverAliasQualified"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_SelectorNonIdentifierOperandNeverMisresolvesSamePackage"
        status: pass
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_LocalVariableReceiverCallUnchanged"
        status: pass
    human_judgment: false
  - id: D4
    description: "Call-as-argument (outer(inner())) extracts and resolves into calls edges for BOTH outer and inner — investigated and confirmed already correct, locked in via regression tests"
    verification:
      - kind: unit
        ref: "internal/indexer/goextract/goextract_test.go#TestExtract_CallAsArgument"
        status: pass
      - kind: unit
        ref: "internal/indexer/resolve_test.go#TestResolve_CallAsArgument"
        status: pass
    human_judgment: false
  - id: D5
    description: "Determinism preserved: full repo test suite green under -race, golden-parity harness green, two-from-scratch-rebuilds-byte-identical gate (TestDeterministicRebuild) green"
    verification:
      - kind: unit
        ref: "go test ./... -race -count=1 (all 13 packages pass)"
        status: pass
      - kind: integration
        ref: "testdata/golden/golden_parity_test.go#TestGoldenParity"
        status: pass
      - kind: unit
        ref: "internal/indexer/determinism_test.go#TestDeterministicRebuild"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 03: Resolution Breadth — Module-Key Seam + WR-01/WR-02 Fixes Summary

**Generalized symbolIndex to a per-language module-key seam and fixed WR-01 (deterministic same-module name collisions) and WR-02 (selector-on-non-identifier-operand mis-resolution); investigated and confirmed the call-as-argument extraction gap was already resolved by goextract.go's existing recursive walk.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-07-12
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments
- Renamed `symbolIndex.byImportAndName` → `byModuleKeyAndName` and updated every doc comment referencing Go-specific "importPath" language to "moduleKey" — the two-level `map[string]map[string]string` structure is byte-identical, only its semantic framing generalized (Pitfall 2)
- Added a shared `addSymbol(names, name, id)` helper that both `overlay()` (fresh-build/reparse path) and `newSymbolIndexFromStore()` (store-seeded path) now call — the single WR-01 collision-resolution chokepoint, so Sync's store-seeded index and a from-scratch `newSymbolIndex` can never disagree on which node wins a collision
- WR-01 fixed: a same-(moduleKey, name) collision now keeps the lexicographically lowest node Id (mirroring `query.Engine.resolveSymbolNode`'s own tie-break), proven order-independent by shuffling file-processing order in `TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent`; every genuine collision increments `symbolIndex.Collisions`
- WR-02 fixed: `recordCall`'s `selector_expression` branch now only sets `PkgAlias` to a real identifier when the operand actually is one; a non-identifier operand (`foo().Bar()`, `arr[i].Bar()`) gets a synthetic `"<" + operand.Kind() + ">"` alias that can never match a real import alias, forcing the ref through `resolveSelector`'s existing narrowest-safe-set boundary to a deterministic "unresolved" instead of a same-package false match
- Investigated the call-as-argument extraction gap (D-05's third item, 252e2sav94) across 7 combinations of outer/inner call shapes, including the exact golden-corpus shape (`finish.AddCommand(a.newFinishOpenCmd(), a.newFinishReconcileCmd())`) — all extract and resolve correctly today. No code change was needed; regression tests lock this in. The residual golden-parity `Impact` discrepancy documented in `testdata/golden/golden_parity_test.go` traces to Go's (already-ratified, out-of-scope) receiver-based method dispatch limitation, not to argument-position extraction.

## Task Commits

Each task was committed with a RED (test) then GREEN (fix) TDD pair:

1. **Task 1: Per-language module-key seam + WR-01 collision handling**
   - `816a4c7` (test) — failing WR-01 collision regression test, confirmed RED by reverting the implementation and observing a genuine compile failure
   - `809193a` (fix) — module-key rename + `addSymbol` collision handling, confirmed GREEN
2. **Task 2: WR-02 selector-on-non-identifier + call-as-argument investigation**
   - `a16635a` (test) — WR-02 regression tests (confirmed RED against pre-fix code) + call-as-argument lock-in tests (confirmed already-GREEN, documented per the TDD fail-fast investigation rule)
   - `b738387` (fix) — WR-02 fix in `recordCall`, confirmed GREEN

**Plan metadata:** this SUMMARY's own commit closes out the plan.

## Files Created/Modified
- `internal/indexer/symbolindex.go` - `byModuleKeyAndName` rename, `addSymbol` collision-resolution helper, `Collisions` counter field, doc comments generalized
- `internal/indexer/symbolindex_test.go` - new file: WR-01 collision determinism/order-independence, no-false-collision, and re-overlay-not-a-collision tests
- `internal/indexer/resolve_test.go` - `TestResolve_SelectorNonIdentifierOperandNeverMisresolvesSamePackage` (WR-02, end-to-end) and `TestResolve_CallAsArgument` (regression lock-in)
- `internal/indexer/goextract/goextract.go` - `recordCall`'s selector branch: synthetic non-matching `PkgAlias` for non-identifier operands (WR-02)
- `internal/indexer/goextract/goextract_test.go` - `TestExtract_CallAsArgument`, `TestExtract_SelectorNonIdentifierOperandNeverAliasQualified`, `TestExtract_LocalVariableReceiverCallUnchanged`

## Decisions Made
- WR-01's tie-break reuses `query.Engine.resolveSymbolNode`'s existing lowest-Id-wins convention rather than inventing a new determinism rule
- `Collisions` is tracked and tested but not yet plumbed to `Stats`/CLI `--verbose` (that touches `sync.go`/`cmd/`, outside this plan's `files_modified`) — a deliberate, documented scope boundary, not a silent drop
- WR-02's fix lives in the extractor (`goextract.go`), not the resolver — `resolveSelector`'s alias-membership boundary was already correct and is preserved verbatim; the bug was the extractor leaking an empty `PkgAlias` that skipped that boundary entirely
- The call-as-argument item required investigation, not implementation: 7 test combinations (including the exact golden-corpus shape) all already pass against unmodified code. Per the TDD fail-fast rule, this was confirmed deliberately rather than assumed away — the tests are committed as regression guards, and the finding is documented here for future readers of D-05/252e2sav94

## Deviations from Plan

### Auto-fixed Issues

None — both fixes (WR-01, WR-02) were implemented exactly as the plan's `<action>` blocks specified.

### Investigation Finding (not a deviation, but worth flagging)

**Call-as-argument extraction gap does not reproduce against current code**
- **Found during:** Task 2, before writing any fix
- **Investigation:** Per the TDD fail-fast rule ("if a test passes unexpectedly during RED, investigate, do not skip"), 7 combinations of nested call shapes were tested against unmodified `goextract.go`/`resolve.go`: plain `outer(inner())`, selector-outer/identifier-inner, identifier-outer/selector-inner, 2-deep nesting, binary-expression argument, multi-argument, and the exact shape from `testdata/golden/golden_parity_test.go`'s documented `Impact` discrepancy (`finish.AddCommand(a.newFinishOpenCmd(), a.newFinishReconcileCmd())`). All 7 extract every nested call correctly and all resolvable ones (i.e., not routed through an unresolvable receiver-variable selector) produce correct `calls` edges.
- **Root cause of the observed conflation:** `walkDescendants` (goextract.go) already unconditionally descends into every node's children regardless of kind — it never stops at the first `call_expression` it finds — so a call nested in an argument list was never actually dropped by extraction. The golden-corpus symptom that originally motivated D-05's third item traces instead to Go's receiver-based method dispatch resolution (`a.newFinishOpenCmd()` — "a" is a method receiver, not a real import alias, so it correctly falls through to "unresolved" via the same ratified "no local-variable type tracking" boundary as `TestResolve_UnresolvedMethodCall`) — an entirely separate, already-scoped-out limitation (RES-02 dispatch synthesis, a later Phase 5 plan), not an extraction bug.
- **Action taken:** No code change. Added `TestExtract_CallAsArgument` and `TestResolve_CallAsArgument` as regression guards so this behavior is locked in going forward.
- **Files modified:** internal/indexer/goextract/goextract_test.go, internal/indexer/resolve_test.go
- **Committed in:** a16635a (test commit)

---

**Total deviations:** 0 auto-fixed. One documented investigation finding (no code change required).
**Impact on plan:** Full scope delivered; the third D-05 item is satisfied by evidence that it was already correct, not by a fix.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `symbolIndex` is per-language-module-keyed and collision-deterministic; every Wave-4 language extractor/resolver inherits this exact shape (two-level map, `addSymbol` chokepoint, `resolveSelector`'s alias-membership boundary preserved verbatim)
- WR-01 and WR-02 — the two genuine resolution bugs D-05 identified — are fixed with regression tests; the third item (call-as-argument) is proven already-correct and locked in
- `symbolIndex.Collisions` exists and is tested but not yet surfaced via `Stats`/CLI `--verbose` — a natural, low-risk follow-up for whichever future plan next touches `sync.go`
- `go build ./...`, `go vet ./...`, and `go test ./... -race -count=1` pass across all 13 packages; `testdata/golden/... -run TestGoldenParity` remains green (with its pre-existing, now-explained `Impact` tolerance note); `TestDeterministicRebuild`/`TestRealRepoStructure` (the byte-identical-rebuild gate) remain green

---
*Phase: 05-language-coverage-resolution-breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED

All created/modified files confirmed present on disk; all four task commits (816a4c7, 809193a, a16635a, b738387) confirmed in git log.
