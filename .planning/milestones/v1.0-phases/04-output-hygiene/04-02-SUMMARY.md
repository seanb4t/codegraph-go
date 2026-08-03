---
phase: 04-output-hygiene
plan: 02
subsystem: testing
tags: [archtest, go-packages, go-types, mcp, stdout, hygiene]

requires:
  - phase: 03-watcher-on-mcp-default
    provides: "the two existing archtest precedents (import_graph_test.go, modernc_confinement_test.go) this plan mirrors"
provides:
  - "isOSStdoutRef/isBareFmtPrint/isLogSetOutput: type-checked (go/types) detection predicates for stdout-writing constructs, proven against synthetic fixtures"
  - "TestNoStdoutNoiseInServeReachablePackages: a build-time archtest scanning the six serve-reachable packages for os.Stdout/bare fmt.Print*/log.SetOutput"
  - "TestStdoutGuardDetectsViolations: mutation-proof self-test of the detection predicates"
affects: [04-output-hygiene (plan 03, HYG-02 end-to-end harness half), 06-charm-tui]

tech-stack:
  added: []
  patterns:
    - "go/packages archtest extended with NeedSyntax|NeedTypes|NeedTypesInfo + go/ast.Inspect for structural stdout-write confinement, mirroring the existing import-graph confinement pattern"
    - "Detection predicates resolve identifiers through info.Uses + types.Object.Pkg().Path(), never string-matching source text"
    - "Tests: false in packages.Config (deliberate divergence from the two Tests:true precedents) — a _test.go file's stdout write never touches the real serve --mcp binary's stdout, and this choice also avoids false positives on this package's own logger_test.go (log.SetOutput) and indexer's _test.go files (fmt.Println)"

key-files:
  created:
    - internal/graphstore/archtest/stdout_confinement_test.go
    - internal/graphstore/archtest/stdout_detection_selftest_test.go
  modified: []

key-decisions:
  - "Tests: false confirmed correct by direct verification, not just RESEARCH's stated reasoning: grepping the six guarded packages found log.SetOutput calls only in internal/graphstore/logger_test.go (the Phase-1 mutation-proof wiring test's capture-buffer redirect) and fmt.Println only in internal/indexer's _test.go files — both legitimate test-only constructs that Tests: true would have falsely flagged"
  - "Positive/negative self-test fixtures designed with exact expected counts (1 stdout ref, 1 bare print, 1 SetOutput in positive; 0/0/0 in negative) rather than open-ended assertions, so a predicate regression in either direction is caught precisely"
  - "Manually verified the mutation-intuition claim from the plan's <verification> section: injected fmt.Println into internal/query/engine.go, confirmed TestNoStdoutNoiseInServeReachablePackages fails with the expected error message, then reverted (git diff confirms zero residual change)"

patterns-established:
  - "Structural stdout-confinement archtest as a sibling to the existing import-graph confinement archtest — same package, same file-naming convention, same 'assert the wiring, not a replica' + Pitfall-4 sanity-check discipline"

requirements-completed: [HYG-02]

coverage:
  - id: D1
    description: "Three detection predicates (isOSStdoutRef, isBareFmtPrint, isLogSetOutput) resolve identifiers via go/types (info.Uses + Pkg().Path()), correctly flagging os.Stdout/bare fmt.Print*/log.SetOutput and correctly ignoring os.Stderr/fmt.Fprintf/log.Printf/a locally shadowed os identifier"
    requirement: "HYG-02"
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest/stdout_detection_selftest_test.go#TestStdoutGuardDetectsViolations"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestNoStdoutNoiseInServeReachablePackages scans the six serve-reachable packages (mcp, graphstore, daemon, watch, indexer, query) via go/packages and fails on any stdout-write violation; internal/cli remains excluded; green on today's zero-violation baseline (D-07); Pitfall-4 sanity check guards against a silently-vacuous pass"
    requirement: "HYG-02"
    verification:
      - kind: unit
        ref: "internal/graphstore/archtest/stdout_confinement_test.go#TestNoStdoutNoiseInServeReachablePackages"
        status: pass
      - kind: manual_procedural
        ref: "manually injected fmt.Println into internal/query/engine.go, re-ran TestNoStdoutNoiseInServeReachablePackages (failed as expected with the correct error message), reverted (git diff confirms no residual change)"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-07-16
status: complete
---

# Phase 4 Plan 2: Stdout Confinement Archtest (HYG-02 Structural Guard) Summary

**A build-time go/packages+go/ast+go/types archtest makes any stdout write from the six serve-reachable packages a compile-test failure, mirroring the existing import-graph confinement precedent and proven mutation-capable both at the detection-predicate layer (synthetic fixtures) and the package-scan layer (a real injected violation)**

## Performance

- **Duration:** 9 min
- **Tasks:** 2
- **Files modified:** 2 (both created)

## Accomplishments
- `isOSStdoutRef`/`isBareFmtPrint`/`isLogSetOutput`: three unexported predicates in `internal/graphstore/archtest/stdout_confinement_test.go` that resolve identifiers through `go/types` (`info.Uses` + `types.Object.Pkg().Path()`), never string-matching — correctly flag `os.Stdout`, bare `fmt.Print`/`Printf`/`Println`, and `log.SetOutput`, and correctly ignore `os.Stderr`, `fmt.Fprintf`, `log.Printf`, and a locally shadowed identifier named `os`
- `TestStdoutGuardDetectsViolations` (self-test): synthetic positive fixture (all three constructs, once each) fully flagged; synthetic negative fixture (every documented non-match case) flagged zero times — proves the predicates can actually fail
- `TestNoStdoutNoiseInServeReachablePackages`: loads `internal/mcp`, `internal/graphstore`, `internal/daemon`, `internal/watch`, `internal/indexer`, `internal/query` via `packages.Load` (`NeedSyntax|NeedTypes|NeedTypesInfo` + base flags, `Tests: false`) and fails on any violation; `internal/cli` deliberately excluded; a Pitfall-4 sanity check (`len(pkgs) != 6` or empty `Syntax`) prevents a silent no-op; green on today's D-07 zero-violation baseline

## Task Commits

1. **Task 1: Detector predicates + mutation-proof self-test (RED→GREEN)**
   - `679e88d` test(04-02): add failing self-test for stdout guard detection predicates
   - `2d1559f` feat(04-02): implement stdout guard detection predicates (GREEN)
2. **Task 2: Package-scanning archtest over the six serve-reachable packages**
   - `9978655` feat(04-02): scan six serve-reachable packages for stdout writes (HYG-02)

**Plan metadata:** (this commit)

## TDD Gate Compliance

Task 1 followed RED→GREEN: `679e88d` (test) precedes `2d1559f` (feat) in git log. RED was verified as a real build failure (`undefined: isOSStdoutRef` etc.), not a vacuous pass — confirmed via `go test ./internal/graphstore/archtest/ -run TestStdoutGuardDetectsViolations -v` before the GREEN commit. No REFACTOR commit was needed. Task 2 was `type="auto"` (not TDD) per the plan.

## Files Created/Modified
- `internal/graphstore/archtest/stdout_confinement_test.go` - the three detection predicates + `guardedPackages` + `TestNoStdoutNoiseInServeReachablePackages`
- `internal/graphstore/archtest/stdout_detection_selftest_test.go` - synthetic positive/negative fixtures + `TestStdoutGuardDetectsViolations`

## Decisions Made
- Confirmed `Tests: false` empirically (not just per RESEARCH's stated reasoning) by grepping the six guarded packages for `log.SetOutput`/`fmt.Println`: both appear only in `_test.go` files (Phase-1's `logger_test.go` mutation-proof wiring test, and `internal/indexer`'s test files) — `Tests: true` would have produced false-positive violations on legitimate test-only code
- Designed self-test fixtures with exact expected counts per construct (not just "at least one"/"zero"), so the self-test itself is a precise, mutation-sensitive assertion
- Manually verified the plan's own mutation-intuition claim by injecting a real `fmt.Println` into `internal/query/engine.go`, confirming the guard fails with the expected message, then reverting (zero residual diff)

## Deviations from Plan

None — plan executed exactly as written. Both tasks' acceptance criteria were met without needing any Rule 1-4 deviation.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- HYG-02's structural half (build-time guard) is complete and verified two ways: synthetic-fixture self-test + a real injected-and-reverted violation
- HYG-02's end-to-end half (the D-06a raw-stdio JSON-RPC frame-purity harness assertion, and D-09's CLI-noise-absence check) remains for Plan 04-03, per this plan's own `<flagged_assumptions>` note
- `go test ./...` green across the whole module; no new dependency, no new CI step (rides existing `go test ./...` coverage per D-10)

## Self-Check: PASSED

All claimed files and commit hashes verified present.

---
*Phase: 04-output-hygiene*
*Completed: 2026-07-16*
