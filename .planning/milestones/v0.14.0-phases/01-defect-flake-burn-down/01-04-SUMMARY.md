---
phase: 01-defect-flake-burn-down
plan: 04
subsystem: cli
tags: [go, cobra, pebble, graphstore, cli, tdd]

# Dependency graph
requires: []
provides:
  - "`codegraph index --force` refuses (non-zero exit) against a store another live process holds open, before any destructive `os.RemoveAll`"
  - "three-way classification of a prior store's readability (never-indexed / locked / corrupt), surfaced through `graphstore`'s exported sentinels only"
affects: [phase-03-verb-rename (the refusal message names `codegraph unlock`, which Phase 3 renames to `daemon unlock`)]

# Actuals (#2632)
actuals:
  tokens: 4100
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Directory-identity assertion (os.SameFile on a directory's os.FileInfo) to prove a wipe never ran, instead of exact directory-listing equality, when the directory under test is a live pebble store performing its own background housekeeping (WAL rotation/flush/manifest rotation) independent of the code under test."

key-files:
  created:
    - internal/cli/index_lock_test.go
  modified:
    - internal/cli/index.go

key-decisions:
  - "Corrupted pebble/v2's live MANIFEST file (found via its marker.manifest.* marker) to produce a genuine, non-sentinel Open failure for the D-11 corrupt-store test — pebble/v2 has no CURRENT file (verified by inspecting a freshly-init'd store's directory listing on this tree), so the plan's suggested corruption target does not exist in this pebble version."
  - "The ErrStoreLocked refusal message omits a pid, per the plan's own fallback guidance: unlike internal/daemon/lock.go's own PID-tracking lockfile, graphstore.ErrStoreLocked carries no holder pid, and priorCoverageGeneration only ever sees pebble's own in-process lock-collision error, not a pid-bearing record."

requirements-completed: [FIX-06]

coverage:
  - id: D1
    description: "`codegraph index --force` against a store another live process holds open exits non-zero with errors.Is(err, graphstore.ErrStoreLocked), before os.RemoveAll runs — proven by reading the store's own directory identity and coverage generation back through the held handle, not merely by exit code (D-10 / WINDOWS #36 / T-10-16)."
    requirement: FIX-06
    verification:
      - kind: unit
        ref: "internal/cli/index_lock_test.go#TestIndexForceRefusesWhileStoreIsHeld"
        status: pass
    human_judgment: false
  - id: D2
    description: "A never-indexed store still rebuilds silently on `--force`, unchanged (D-11 control)."
    requirement: FIX-06
    verification:
      - kind: unit
        ref: "internal/cli/index_lock_test.go#TestIndexForceProceedsSilentlyWhenNeverIndexed"
        status: pass
    human_judgment: false
  - id: D3
    description: "A corrupt/unreadable store warns on stderr naming the floor-0 consequence, then rebuilds successfully — `--force` is the recovery path (D-11)."
    requirement: FIX-06
    verification:
      - kind: unit
        ref: "internal/cli/index_lock_test.go#TestIndexForceWarnsAndRebuildsOnUnreadableStore"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-09-15
status: complete
---

# Phase 01 Plan 04: Refuse `index --force` Against a Held Store Summary

**`priorCoverageGeneration` now returns `(int64, error)` and `newIndexCmd`'s RunE classifies that error three ways — locked (refuse before `RemoveAll`), never-indexed (silent floor 0), corrupt (warn and rebuild) — closing WINDOWS #36's store-wipe-under-a-live-holder hole.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-15T04:41:00Z
- **Completed:** 2026-09-15T04:58:50Z
- **Tasks:** 2 (RED, GREEN)
- **Files modified:** 2

## Accomplishments
- `codegraph index --force` against a store held open by another process now exits non-zero, wrapping `graphstore.ErrStoreLocked` with `%w`, and names both live remedies (`codegraph daemon stop`, `codegraph unlock`) — proven by a regression test that holds the lock across the call and reads the store's own directory identity and coverage generation back through the held handle afterward, not merely by exit code.
- A never-indexed store (`.codegraph/` exists, store never written) still rebuilds silently on `--force`, exactly as before — the control that proves the fix didn't turn "refuse when unreadable" into "refuse whenever the store cannot be read".
- A corrupt/unreadable store now prints a warning on stderr naming the floor-0 consequence, then proceeds to rebuild successfully — `--force` remains the sanctioned recovery path for this case (D-11).
- `internal/cli` still imports no pebble package; the classification is entirely on `graphstore`'s exported `ErrNotFound`/`ErrStoreLocked` sentinels (`TestNoPackageBypassesGraphStore` stays green, preserving D-04a).
- `codegraph index --help` lists the same flag count as before the plan (baseline recorded below) — no new flag, per D-10.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): hold the store lock across `index --force` and watch the wipe happen** - `6c7b226d` (test)
2. **Task 2 (GREEN): classify the prior store three ways and refuse before the wipe** - `2bd660b3` (fix)

**Plan metadata:** (this commit, following)

_TDD plan: RED commit adds the three new test functions against unmodified `index.go`; GREEN commit implements the fix and also corrects a flaw discovered in the RED test's own assertion methodology (see Deviations)._

## Files Created/Modified
- `internal/cli/index_lock_test.go` - Three new tests: `TestIndexForceRefusesWhileStoreIsHeld`, `TestIndexForceProceedsSilentlyWhenNeverIndexed`, `TestIndexForceWarnsAndRebuildsOnUnreadableStore`.
- `internal/cli/index.go` - `priorCoverageGeneration` returns `(int64, error)`; `newIndexCmd`'s RunE gained a three-way `switch`/`errors.Is` classification strictly before `os.RemoveAll(storeDir)`.

## RED Transcript (against unmodified `index.go`, commit `6a540ccb`)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestIndexForceRefusesWhileStoreIsHeld|TestIndexForceProceedsSilentlyWhenNeverIndexed|TestIndexForceWarnsAndRebuildsOnUnreadableStore' -count=1 -v
index_lock_test.go:99: index --force error = "graphstore: store lock held: lock held by current process"; want it to name `codegraph daemon stop`
--- FAIL: TestIndexForceRefusesWhileStoreIsHeld (0.95s)
--- PASS: TestIndexForceProceedsSilentlyWhenNeverIndexed (0.10s)
    index_lock_test.go:194: corrupt-store failure mode on this tree: pebble: malformed manifest file "MANIFEST-000001" for DB ".../.codegraph/store"
    index_lock_test.go:201: index --force on a corrupt store printed no warning naming the floor-0 consequence; stderr=""
--- FAIL: TestIndexForceWarnsAndRebuildsOnUnreadableStore (0.11s)
FAIL
```

**Note on the held-lock RED failure's actual mechanism:** against unmodified code, `index --force` does NOT return `nil` — it still returns a non-nil error, because `os.RemoveAll`/`os.MkdirAll` run unconditionally (the pre-fix defect), and `indexer.Run`'s own subsequent `pebble.Open` on the recreated (but not lock-cleared — pebble/v2 tracks locks in-process by absolute path) directory then collides with the still-held lock and fails on its own. So the RED failure here is on the **message assertion** (the current text is a bare pebble internal string, not the intended user-facing refusal) rather than on `errors.Is` itself — the wipe genuinely proceeds either way, which is the defect this plan closes by moving the refusal to before `RemoveAll`, not after `indexer.Run`'s own incidental collision.

`git show --stat --format= HEAD -- internal/cli/index.go` was empty at the RED commit — `index.go` was byte-unchanged by Task 1.

## GREEN Transcript (after fix, commit `2bd660b3`)

```
$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run 'TestIndexForce' -count=1 -v
--- PASS: TestIndexForceRefusesWhileStoreIsHeld (0.51s)
--- PASS: TestIndexForceProceedsSilentlyWhenNeverIndexed (0.08s)
--- PASS: TestIndexForceWarnsAndRebuildsOnUnreadableStore (0.14s)
--- PASS: TestIndexForceRebuildBumpsCoverageGeneration (0.24s)
PASS
```

Stable across 3 repeated runs (no flakiness observed). `go test ./internal/cli/... -count=1` — all subpackages pass. `rg -v '^\s*//' internal/cli/index.go | rg -n 'ErrStoreLocked|ErrNotFound|os.RemoveAll'` shows the `ErrStoreLocked` branch at a lower line number than `os.RemoveAll` on non-comment lines. `TestNoPackageBypassesGraphStore` passes with no `cockroachdb/pebble` import in `internal/cli`. `codegraph index --help | rg -c -- '^\s+--'` = `1` both before and after the plan (the plan's own verify command; only `--workers` has no short flag and matches this pattern).

## Decisions Made
- Corrupted pebble/v2's live MANIFEST file (discovered via its `marker.manifest.*` marker file) rather than a `CURRENT` file, because pebble/v2 has no `CURRENT` file at all on this tree — confirmed by inspecting a freshly-`init`'d store's directory listing before writing the corrupt-store test. `graphstore.Open` against a corrupted MANIFEST fails with `pebble: malformed manifest file "MANIFEST-000001" for DB "..."`, a genuine non-sentinel error, pinned by name in the test's own setup assertion (not weakened to "any error").
- The `ErrStoreLocked` refusal message omits a pid: `graphstore.ErrStoreLocked` carries no holder pid (unlike `internal/daemon/lock.go`'s own pid-tracking lockfile record), and `priorCoverageGeneration` only ever observes pebble's own in-process lock-collision error. Per the plan's own fallback guidance, the pid is omitted rather than guessed; the message instead names both remedies (`codegraph daemon stop`, `codegraph unlock`) without a pid.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a flaw in the RED test's own assertion, discovered while verifying GREEN**
- **Found during:** Task 2 (running the GREEN gate for the first time)
- **Issue:** `TestIndexForceRefusesWhileStoreIsHeld`'s original assertion compared the store directory's file *listing* before and after the refused call for exact equality. A live pebble handle performs its own background housekeeping (WAL rotation, memtable-to-SST flush, manifest rotation) independently of anything the CLI does — confirmed live: after the ~500ms the CLI's own (correctly refused) attempt spends internally, the held store's directory legitimately gained `000005.sst`/`000007.log`/`MANIFEST-000006` and lost `000002.log`/`OPTIONS-000003`, purely from the held handle's own janitor activity, with the fix already correctly in place and RemoveAll never running. The exact-listing assertion therefore failed even against the CORRECT implementation — a test design flaw, not a production bug.
- **Fix:** Replaced the listing-equality assertion with an `os.SameFile` identity check on `storeDir` itself (its `os.FileInfo`/inode, captured before and after). `os.RemoveAll(storeDir)` + `os.MkdirAll(storeDir, ...)` necessarily replaces the directory's own inode; pebble's internal file churn inside an unchanged directory does not. The file listing is still captured and logged (informational only, not asserted on) so a future reader can see what changed.
- **Files modified:** `internal/cli/index_lock_test.go` (folded into the Task 2 GREEN commit, since discovering and fixing it was part of verifying GREEN — the RED commit's failure mode was unaffected by this change, confirmed by re-running RED-equivalent conditions were not re-tested against the corrected assertion, since RED's failure was on the message check, which fires before the directory check either way).
- **Verification:** Re-ran the full `TestIndexForceRefusesWhileStoreIsHeld` three times after the fix — stable PASS each time, with the informational listing log confirming the expected churn pattern.
- **Committed in:** `2bd660b3` (Task 2 commit, alongside the production fix).

---

**Total deviations:** 1 auto-fixed (test-methodology bug, Rule 1).
**Impact on plan:** No scope creep — the fix is confined to the test file this plan already owns, and does not weaken any assertion (it replaces a noisy assertion with a more precise one that still fails if `RemoveAll` runs).

## Issues Encountered
- `task build:release` fails without `GOTOOLCHAIN=go1.26.6` pinned explicitly (a locally-installed go1.27.1 silently shadows `go.mod`'s declared `go 1.26.6`) — this matches the environment note already recorded in `01-RESEARCH.md`'s Standard Stack table; no code change needed, just re-ran with the pin.
- A full-suite `go test ./... -count=1` run (unrelated to this plan's own verify gates, run as extra diligence) showed `internal/daemon` FAIL once under full parallel load, then PASS cleanly in isolation (65s). This plan touches only `internal/cli`; the failure is the same load-sensitive "Daemon extreme-load tail" class already recorded as an open, accepted item in `deferred-items.md` and `.planning/STATE.md`. Logged a new entry to `.planning/phases/01-defect-flake-burn-down/deferred-items.md` rather than investigating further (out of scope per the Scope Boundary rule).

## TDD Gate Compliance

| Gate | Required | Found | Status |
|------|----------|-------|--------|
| RED | Yes | `test(01-04): add failing hold-the-lock-across-index-force regression test` (`6c7b226d`) | Present |
| GREEN | Yes | `fix(01-04): refuse index --force against a held store before RemoveAll` (`2bd660b3`) | Present, non-standard prefix (see note) |
| REFACTOR | No | — | Not needed; no cleanup pass required after GREEN |

**Note on the GREEN commit's prefix:** the generic TDD template names `feat({phase}-{plan}): ...` as the GREEN commit pattern; this plan's own Task 2 `<action>` explicitly instructed `fix(01-04): refuse index --force against a held store before RemoveAll`, because the change is a bug fix (closing WINDOWS #36's store-wipe-under-a-live-holder defect) rather than new-feature work — `fix` is the more accurate conventional-commit type per this repo's own commit-type table. The RED→GREEN discipline itself was followed exactly (test written and watched fail first, per the transcripts above, then the minimal implementation made it pass); only the literal commit-type token differs from the generic template's default. Confirmed bounded to this milestone (`git log ... v0.13.0..HEAD`) per this repo's own numbering-collision warning (a bare `feat(01-04)` grep matches an unrelated `v0.12.0`-era commit).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- FIX-06 is closed: WINDOWS #36 / T-10-16 can be marked resolved.
- The refusal message names `codegraph unlock` (the verb live today) rather than `daemon unlock` — Phase 3's verb-rename work (VERB-04) should update this message's wording when it renames the verb, per the phase's own noted intent (D-10).
- `internal/daemon`'s load-sensitive full-suite flake remains open (tracked in `deferred-items.md`), unrelated to this plan.

---
*Phase: 01-defect-flake-burn-down*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: `internal/cli/index_lock_test.go`
- FOUND: `internal/cli/index.go`
- FOUND: `.planning/phases/01-defect-flake-burn-down/01-04-SUMMARY.md`
- FOUND commit `6c7b226d` (RED)
- FOUND commit `2bd660b3` (GREEN)
