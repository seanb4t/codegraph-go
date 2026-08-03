---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 03
subsystem: cli
tags: [cobra, cli-flags, upgrade, release-verification, ts-parity]

requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: "08-01 (impact -d/-j), 08-02 (files -j/--dir) established the same *Var->*VarP mechanical idiom this plan reuses"
provides:
  - "status/query/callers/callees/install/uninstall short-flag TS-parity aliases (-j, -l, -k, -t)"
  - "upgrade --force/-f flag with a verification-safe same-version reinstall guard"
  - "upgrade.Options.Force bool, threaded from internal/cli/upgrade.go into internal/upgrade.Run"
affects: [08-05, 08-08, docs/FLAG-PARITY.md]

tech-stack:
  added: []
  patterns:
    - "*Var -> *VarP mechanical short-flag addition, no RunE body change"
    - "same-version no-op guard placed strictly before checkWritable/download, verify() left byte-for-byte untouched"

key-files:
  created: []
  modified:
    - internal/cli/status.go
    - internal/cli/query.go
    - internal/cli/callers.go
    - internal/cli/callees.go
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - internal/cli/upgrade.go
    - internal/upgrade/upgrade.go
    - internal/upgrade/upgrade_test.go

key-decisions:
  - "Every TS-free short letter adopted per CONTEXT D-04; no existing Go binding (-p/-q/-v/-y) remapped"
  - "upgrade --force only bypasses the same-version no-op guard; verify()'s fatal-error branch and its ordering before swap are unchanged"

patterns-established: []

requirements-completed: [SURF-03]

coverage:
  - id: D1
    description: "status/query/callers/callees/install/uninstall gain TS-parity short flags (-j/-l/-k/-t) with no shorthand collision"
    requirement: "SURF-03"
    verification:
      - kind: unit
        ref: "go build ./... (cobra panics at command construction on any within-command duplicate shorthand)"
        status: pass
      - kind: unit
        ref: "go test ./internal/cli/... -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "upgrade --force/-f forces a same-version reinstall (download->verify->swap) while an unforced same-version run stays a no-op advisory"
    requirement: "SURF-03"
    verification:
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_SameVersionNoOpWithoutForce"
        status: pass
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_ForceReinstallsSameVersion"
        status: pass
    human_judgment: false
  - id: D3
    description: "upgrade --force never weakens verification — a failing fake verify still blocks swap even when Force=true"
    requirement: "SURF-03"
    verification:
      - kind: unit
        ref: "internal/upgrade/upgrade_test.go#TestUpgradeRun_ForceStillVerifiesBeforeSwap"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-19
status: complete
---

# Phase 08 Plan 03: Mechanical short-flag parity + upgrade --force Summary

**Added six commands' missing TS-parity short flags and a new `upgrade --force/-f` flag whose same-version reinstall guard never touches the fail-closed verify() ordering.**

## Performance

- **Duration:** 12 min
- **Tasks:** 2 completed
- **Files modified:** 9

## Accomplishments
- `status` gains `-j`; `query` gains `-l`/`-k`/`-j`; `callers`/`callees` gain `-l`/`-j`; `install`/`uninstall` gain `-t`/`-l` — every TS 1.3.1 short letter that was free in its own command, applied via the mechanical `*Var`→`*VarP` idiom with no RunE body changes.
- `upgrade` gained the whole missing `--force`/`-f` flag (not just a short alias — the flag was entirely absent). Without `--force`, resolving to the same version as `currentVersion` now prints an "already on \<version\>" advisory and returns `nil` before `checkWritable`/download. With `--force`, the same-version case proceeds through the unchanged download → verify → swap sequence.
- Verified via a dedicated TDD RED/GREEN test triple that `--force` bypasses only the no-op guard: `verify()`'s call site and its fatal-error branch (`"signature verification failed"`) are byte-for-byte unchanged and still precede every swap, including forced same-version reinstalls.

## Task Commits

1. **Task 1: mechanical short-flag aliases (status/query/callers/callees/install/uninstall)** - `e1806ed` (feat)
2. **Task 2: upgrade --force/-f flag + verification-safe same-version guard**
   - RED: `40e8542` (test) — 3 new tests fail to compile (`Force` field doesn't exist yet)
   - GREEN: `ca58976` (feat) — `Options.Force` added, same-version guard added, CLI flag wired; all tests pass

**Plan metadata:** (this commit, following SUMMARY.md write)

## Files Created/Modified
- `internal/cli/status.go` - `--json` → `BoolVarP(..., "j", ...)`
- `internal/cli/query.go` - `--kind`/`--limit`/`--json` → `StringVarP`/`IntVarP`/`BoolVarP` with `-k`/`-l`/`-j`
- `internal/cli/callers.go` - `--limit`/`--json` → `-l`/`-j`
- `internal/cli/callees.go` - `--limit`/`--json` → `-l`/`-j`
- `internal/cli/install.go` - `--target`/`--location` → `-t`/`-l`
- `internal/cli/uninstall.go` - `--target`/`--location` → `-t`/`-l`
- `internal/cli/upgrade.go` - new `force` var, `BoolVarP(&force, "force", "f", ...)`, threaded into `upgrade.Options{Force: force}`
- `internal/upgrade/upgrade.go` - new `Options.Force bool` field (doc comment states it never affects verification); `Run` adds a same-version guard (`target == currentVersion && !opts.Force`) placed before `checkWritable`; download→verify→swap sequence and the fatal verify() branch untouched
- `internal/upgrade/upgrade_test.go` - 3 new tests: same-version no-op without Force, Force reinstalls same version through the full sequence, Force still verifies before swap

## Decisions Made
- Adopted every TS-free short letter per CONTEXT D-04's collision policy; no existing Go binding (`-p`/`-q`/`-v`/`-y`) was remapped in any of the six commands.
- Placed the same-version guard immediately after resolving `target` and strictly before `checkWritable`/download, mirroring D-13's existing "refuse before wasting a download" placement style, so the no-op path also never touches the target-writability check.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
SURF-03 is fully closed. Remaining plans in this wave (08-05 SURF-05 docs, 08-08, etc.) can now document the accepted divergences (`-p`/`-q`/`-v`/`-y` retained, `install --auto-allow` behavioral gap, etc.) referencing this plan's short-flag additions as already landed. No blockers for downstream plans.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED
