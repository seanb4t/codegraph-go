---
phase: 04-output-hygiene
plan: 01
subsystem: database
tags: [pebble, logging, graphstore, hygiene]

requires:
  - phase: 03-watcher-on-mcp-default
    provides: "CR-01 bounded lock-retry loop at graphstore.Open (pebble_store.go), openLockRetrySleep test-seam convention"
provides:
  - "quietLogger: an unexported pebble.Logger implementation in internal/graphstore discarding Infof unconditionally, preserving Errorf/Fatalf via a provenance-prefixed diagWriter seam"
  - "diagWriter: package-level injectable io.Writer seam (defaults os.Stderr, no exported setter) mirroring openLockRetrySleep's convention"
  - "quietLogger wired at graphstore.Open's single pebble.Open call site (pebble_store.go:147)"
affects: [04-output-hygiene (plan 02/03, HYG-02), 06-charm-tui]

tech-stack:
  added: []
  patterns:
    - "pebble.Logger injection at a single Open seam (Options.Logger only, no EventListener/LoggerAndTracer)"
    - "genuinely mutation-proof wiring test: redirect stdlib log's default output (where pebble's DefaultLogger.Infof actually writes when unwired) into the same buffer as the package's own diagWriter seam, so a revert of the injection produces real captured noise instead of a vacuous pass"

key-files:
  created:
    - internal/graphstore/logger.go
    - internal/graphstore/logger_test.go
  modified:
    - internal/graphstore/pebble_store.go

key-decisions:
  - "Provenance prefixes are 'codegraph: pebble: ' (Errorf) and 'codegraph: pebble: fatal: ' (Fatalf) — exact wording was Claude's discretion per CONTEXT.md D-04"
  - "The D-08 mutation-proof wiring test cannot rely on diagWriter capture alone: pebble's base.DefaultLogger (installed when Options.Logger is left nil) writes via stdlib log.Output(2, ...), never touching diagWriter, so a revert of the Open seam would leave diagWriter empty either way — a vacuous pass. Fixed by also redirecting log.SetOutput into the same capture buffer for the duration of the test, so DefaultLogger's real 'Found %d WALs' Infof line becomes observable and the test genuinely fails on revert (manually verified: reverting pebble_store.go:147 turns TestOpenInjectsQuietLogger red, restored after confirming)."
  - "Fatalf's os.Exit(1) is never exercised in tests — only the shared writeDiagLine formatting helper is, per RESEARCH.md Pitfall 3 (calling Fatalf directly would kill the test binary)"

patterns-established:
  - "pebble.Logger single-seam injection: Options.Logger only, EnsureDefaults derives the two logger-relevant EventListener callbacks (BackgroundError/DataCorruption) from it automatically — no separate EventListener wiring needed"

requirements-completed: [HYG-01]

coverage:
  - id: D1
    description: "quietLogger.Infof discards Pebble WAL/compaction/memtable chatter unconditionally; Errorf/Fatalf preserve real diagnostics via a provenance-prefixed diagWriter seam (D-02/D-04)"
    requirement: "HYG-01"
    verification:
      - kind: unit
        ref: "internal/graphstore/logger_test.go#TestQuietLoggerInfofDiscards"
        status: pass
      - kind: unit
        ref: "internal/graphstore/logger_test.go#TestQuietLoggerErrorfWritesProvenance"
        status: pass
      - kind: unit
        ref: "internal/graphstore/logger_test.go#TestQuietLoggerFatalfFormattingHelper"
        status: pass
    human_judgment: false
  - id: D2
    description: "graphstore.Open injects quietLogger at the single pebble.Open seam (pebble_store.go:147); a real Open/write/flush/close cycle emits zero Pebble noise; a direct Errorf still surfaces (control); the wiring is mutation-proof — reverting line 147 turns the test red"
    requirement: "HYG-01"
    verification:
      - kind: unit
        ref: "internal/graphstore/logger_test.go#TestOpenInjectsQuietLogger"
        status: pass
      - kind: unit
        ref: "internal/graphstore/logger_test.go#TestQuietLoggerSilencesStoreActivity"
        status: pass
      - kind: manual_procedural
        ref: "manual revert of pebble_store.go:147 to &pebble.Options{}, re-ran TestOpenInjectsQuietLogger (failed as expected), restored the fix"
        status: pass
    human_judgment: false

duration: 14min
completed: 2026-07-16
status: complete
---

# Phase 4 Plan 1: Pebble Logger Routing (HYG-01) Summary

**quietLogger routes Pebble's WAL/compaction/memtable chatter to a discard while preserving Errorf/Fatalf diagnostics via a provenance-prefixed diagWriter seam, injected at graphstore.Open's single pebble.Open call site**

## Performance

- **Duration:** 14 min
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- `internal/graphstore/logger.go`: unexported `quietLogger` implementing pebble v2's 3-method `base.Logger` interface — `Infof` no-ops (D-02), `Errorf`/`Fatalf` route through a shared `writeDiagLine` helper writing `codegraph: pebble: ...` lines to the `diagWriter` seam (defaults `os.Stderr`, mirrors the existing `openLockRetrySleep` test-seam convention)
- `internal/graphstore/pebble_store.go:147`: `pebble.Open(dir, &pebble.Options{Logger: quietLogger{}})` — the sole one-line change; the CR-01 bounded lock-retry loop, `classifyOpenError`, and `ErrStoreLocked` are byte-unchanged
- Genuinely mutation-proof wiring test (`TestOpenInjectsQuietLogger`): a real Open/write/flush/close cycle against a fresh `t.TempDir()` store is asserted to produce zero bytes of Pebble noise, with the stdlib `log` package's default output redirected into the same capture buffer as `diagWriter` — closing the gap where capturing `diagWriter` alone would vacuously pass regardless of whether the injection is wired (Pebble's `DefaultLogger` never touches `diagWriter`; it writes via `log.Output` directly). Manually verified: reverting line 147 turns this test red.

## Task Commits

Each task was committed atomically (TDD RED→GREEN per task):

1. **Task 1: quietLogger type + diagWriter seam**
   - `f016b9f` test(04-01): add failing test for quietLogger (RED)
   - `172eed5` feat(04-01): implement quietLogger + diagWriter seam (GREEN)
2. **Task 2: Inject quietLogger at the Open seam + mutation-proof wiring test**
   - `05743a9` test(04-01): add failing mutation-proof wiring test for Open's Logger seam (RED)
   - `235e97a` feat(04-01): inject quietLogger at the pebble.Open seam (GREEN)

## TDD Gate Compliance

Both tasks followed RED→GREEN: each task's test commit precedes its implementation commit in git log, and each RED commit was verified to fail (build failure for Task 1's `undefined: quietLogger` etc.; genuine assertion failure for Task 2's `TestOpenInjectsQuietLogger`, which produced real captured Pebble noise before the fix) before the corresponding GREEN commit. No REFACTOR commit was needed — both implementations were minimal on first pass.

## Files Created/Modified
- `internal/graphstore/logger.go` - quietLogger type, provenance-prefix constants, writeDiagLine helper, diagWriter seam
- `internal/graphstore/logger_test.go` - unit tests for Infof/Errorf/Fatalf-formatting behavior + the D-08 mutation-proof wiring test + its control
- `internal/graphstore/pebble_store.go` - one-line change at line 147 injecting `Logger: quietLogger{}`

## Decisions Made
- Provenance prefix wording (`codegraph: pebble: ` / `codegraph: pebble: fatal: `) — Claude's discretion per CONTEXT.md D-04, kept the suggested root
- The D-08 wiring test needed a design beyond literally "assert diagWriter stays empty" (see key-decisions above and inline test doc comments) to be genuinely mutation-proof, since Pebble's `DefaultLogger` bypasses `diagWriter` entirely when unwired — solved by redirecting stdlib `log`'s default output into the same capture buffer for the duration of the test

## Deviations from Plan

None — plan executed exactly as written. The mutation-proof test design required more care than the plan's literal "captured diagWriter" wording implied (see Decisions above), but this is exactly the discretion the plan granted for verification mechanism (D-08) and does not diverge from any locked decision.

## Issues Encountered

None beyond the mutation-proof design consideration documented above, which was resolved during Task 2's RED phase before any GREEN commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- HYG-01 fully satisfied and verified (unit + manual mutation check); `go test ./...` green across the whole module
- HYG-02 (MCP stdout cleanliness) is out of scope for this plan — tracked separately in this phase's remaining plans
- No new dependency, no CI change needed (rides existing `go test ./...` coverage)

## Self-Check: PASSED

All claimed files and commit hashes verified present.

---
*Phase: 04-output-hygiene*
*Completed: 2026-07-16*
