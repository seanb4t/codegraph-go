---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 07
subsystem: cli
tags: [bubbletea, bubbles, cobra, daemon, tui]

# Dependency graph
requires:
  - phase: 07-interactive-tui-daemon-picker-install-multi-select
    provides: "07-01 InteractiveAllowed dual-TTY gate; 07-02 charm-free daemon registry (List/Register/Deregister); 07-04 StopMatching/StopAll (isStale-corroborated signaling); 07-05 registry+watchdog wired into daemon.Run"
provides:
  - "internal/cli/tui/daemonpicker.go — RunDaemonPicker bubbletea Model (list + stop-one/stop-all/cancel), SortRecordsCurrentFirst ordering helper"
  - "internal/cli/daemon.go restructured into a cobra tree: bare (picker/plain list), daemon start, daemon stop [--all]"
affects: [08-signed-release]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Daemon picker Model (bubbles/v2/list.Model + custom ItemDelegate) mirrors 07-06's agentpicker.go shape, with a resolved daemonAction/target pair dispatched via an unexported dispatchDaemonAction — testable by synthetic tea.Msg without a real tea.Program"
    - "cli-package func var seams (runDaemonPicker, daemonList, daemonStopMatching, daemonStopAll) mirror install.go's interactiveAllowed/runAgentPicker convention for test injection"

key-files:
  created:
    - internal/cli/tui/daemonpicker.go
    - internal/cli/tui/daemonpicker_test.go
  modified:
    - internal/cli/daemon.go
    - internal/cli/daemon_test.go
    - internal/cli/tui/doc.go

key-decisions:
  - "SortRecordsCurrentFirst lives in internal/cli/tui (exported) rather than internal/daemon or duplicated in internal/cli, so the picker's Model and daemon.go's plain non-TTY list share one ordering implementation (TUI-04 same-ordering truth) without a package cycle (internal/cli already imports internal/cli/tui)."
  - "Added a daemonList cli-package func var (default daemon.List) not named in the plan's files_modified list — needed so daemon_test.go can seed deterministic records without depending on any real OS process's liveness, since daemon.List()'s self-heal prunes any record whose pid isn't a genuinely live process in the test binary."
  - "daemon stop's non-real-signal test seam is a NEW cli-package pair (daemonStopMatching/daemonStopAll wrapping daemon.StopMatching/StopAll), not 07-04's internal (package daemon) stopSignal var the plan named — that var is unexported and un-reachable from internal/cli. Same end effect: no real OS signal in tests."
  - "Tasks 2 and 3 were combined into one commit: Task 2's newDaemonCmd() calls cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd()), so Task 3's stop subcommand must exist for Task 2 to compile — no independently-buildable intermediate state, mirroring the 02-status-content plan's own precedent for the same reason."

patterns-established:
  - "Daemon picker bubbletea Model resolves a terminal action+target pair (not dispatching stop calls inline from Update) and exposes an unexported dispatch function callable directly by tests — Model.Update stays synchronous/pure and testable via synthetic tea.Msg."

requirements-completed: [DMON-01, DMON-02, TUI-04]

coverage:
  - id: D1
    description: "Bare `codegraph daemon` on a TTY opens an interactive bubbletea picker (current-project-first) offering stop-one/stop-all/cancel"
    requirement: "DMON-01"
    verification:
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerModel_EnterDispatchesStopOne"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerModel_StopAllDispatches"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerModel_CancelSignalsNothing"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerModel_CurrentRepoFirstOrdering"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonBareCmd_InteractiveAllowed_CallsRunDaemonPicker"
        status: pass
    human_judgment: false
  - id: D2
    description: "Off-TTY, bare `daemon` prints the same current-project-first ordering as a plain list and exits 0 without ever blocking on stdin (including the empty-registry edge)"
    requirement: "TUI-04"
    verification:
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonBareCmd_NonTTY_PrintsSeededRecordsCurrentRepoFirst"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonBareCmd_NonTTY_EmptyRegistry_PrintsNoRunningDaemons"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerModel_EmptyRecordsQuitsWithoutDispatch"
        status: pass
    human_judgment: false
  - id: D3
    description: "`daemon start` preserves the exact old foreground RunE (including the watch.DisabledError friendly-exit branch); `daemon stop [--all]` explicitly and non-interactively signals via daemon.StopMatching/StopAll, with clean exit-0 empty/no-match notices and a non-zero exit on an aggregated stop error; neither the bare command nor stop ever calls daemon.Run"
    requirement: "DMON-02"
    verification:
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStartCmdPolicyDisabledExitsCleanly"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStopCmd_DispatchesToStopMatching"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStopCmd_All_DispatchesToStopAll"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStopCmd_All_EmptyRegistry_CleanNoOp"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStopCmd_NoMatch_Notice"
        status: pass
      - kind: integration
        ref: "internal/cli/daemon_test.go#TestDaemonStopCmd_AggregatedError_ExitsNonZero"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-07-18
status: complete
---

# Phase 7 Plan 07: Daemon Picker & Command Tree Summary

**Restructured `codegraph daemon` into a cobra tree (bare picker/plain-list + `start` + `stop [--all]`) and added a bubbletea daemon picker Model in `internal/cli/tui`, resolving the TS name collision.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-07-18T20:11:00Z
- **Completed:** 2026-07-18T20:36:24Z
- **Tasks:** 3 (Task 2+3 combined into one commit — see Deviations)
- **Files modified:** 5

## Accomplishments
- New bubbletea `daemonPickerModel` in `internal/cli/tui/daemonpicker.go`: a `bubbles/v2/list.Model` over `daemon.Record`, sorted current-repo-first via the new exported `SortRecordsCurrentFirst`, offering stop-one (enter), stop-all ("a"), and cancel (q/esc/ctrl+c) — an empty record set quits from `Init()` without ever dispatching.
- `RunDaemonPicker(cmd, currentRepo, records) error` wires the Program to `cmd`'s own stdin/stdout and dispatches the resolved action via the injectable `stopMatching`/`stopAll` seams (`daemon.StopMatching`/`daemon.StopAll`).
- `codegraph daemon` restructured into a tree: the bare command TTY-gates (via the existing `interactiveAllowed`/`tui.InteractiveAllowed` seam) into `runDaemonPicker` or a plain non-TTY list (`printDaemonList`) using the SAME ordering; `daemon start` is the old foreground `RunE` moved verbatim (including the `watch.DisabledError` friendly-exit branch); `daemon stop [--all]` explicitly signals via new `daemonStopMatching`/`daemonStopAll` seams, printing clean exit-0 notices on empty/no-match and surfacing an aggregated error as non-zero.
- No auto-spawn preserved structurally: only `daemon start`'s `RunE` calls `daemon.New`/`Run` — the bare command and `stop` never do.

## Task Commits

Each task was committed atomically (Task 1's TDD RED/GREEN pair, Tasks 2+3 combined — see Deviations):

1. **Task 1 RED: failing daemon picker Model test** - `bacff95` (test)
2. **Task 1 GREEN: daemon picker Model implementation** - `4154877` (feat)
3. **Tasks 2+3: daemon.go tree restructure + stop subcommand** - `979e285` (feat)

**Plan metadata:** (this commit) `docs(07-07): complete daemon picker & command tree plan`

## Files Created/Modified
- `internal/cli/tui/daemonpicker.go` - `RunDaemonPicker`, `daemonPickerModel`, `daemonDelegate`, `dispatchDaemonAction`, exported `SortRecordsCurrentFirst`
- `internal/cli/tui/daemonpicker_test.go` - synthetic-`tea.Msg` unit tests for ordering/stop-one/stop-all/cancel/navigation/empty
- `internal/cli/tui/doc.go` - dropped the now-redundant 07-01 anchor blank imports (both `agentpicker.go` and `daemonpicker.go` import bubbles/bubbletea directly now)
- `internal/cli/daemon.go` - `newDaemonCmd()` tree (bare RunE + `AddCommand(newDaemonStartCmd(), newDaemonStopCmd())`), `printDaemonList`, `printStoppedDaemons`, new `runDaemonPicker`/`daemonList`/`daemonStopMatching`/`daemonStopAll` func-var seams
- `internal/cli/daemon_test.go` - renamed/retargeted the policy-disabled test to `daemon start`; added bare-command ordering/picker-wiring tests and stop dispatch/no-op/error tests

## Decisions Made
- `SortRecordsCurrentFirst` lives in `internal/cli/tui` (exported), reused by both the picker's Model and `daemon.go`'s plain-list fallback, so the two presentations can never diverge in ordering (TUI-04) — no duplicated comparator logic.
- Added a `daemonList` func-var seam (not in the plan's stated `files_modified`, but required for correctness of testing): `daemon.List()`'s own self-heal prunes any record whose pid isn't a genuinely live OS process, so seeding deterministic multi-record ordering/dispatch tests from `internal/cli` needed an injection point rather than real `Register`/`Deregister` calls with fabricated pids.
- `daemon stop`'s "no real signal in tests" requirement is satisfied by a new `internal/cli`-local `daemonStopMatching`/`daemonStopAll` seam pair (wrapping `daemon.StopMatching`/`StopAll`), not 07-04's package-private `stopSignal` var the plan's `<read_first>` named — that var lives in `package daemon` and is unreachable from `internal/cli`. Same net effect (no OS signal delivered during tests).
- Tasks 2 and 3 were implemented and committed together: Task 2's `newDaemonCmd()` directly calls `newDaemonStopCmd()` (Task 3's function) in its `AddCommand` wiring, so no intermediate state between the two tasks compiles independently.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added a `daemonList` func-var seam to make ordering/dispatch tests deterministic**
- **Found during:** Task 2/3 (writing `daemon_test.go`'s bare-command tests)
- **Issue:** `daemon.List()` self-heals on every call — pruning any record whose pid isn't a genuinely live process. Tests seeding fabricated multi-record registries via `daemon.Register` with arbitrary pids (111/222/333) were silently pruned to zero records, failing the ordering/picker-wiring assertions.
- **Fix:** Added `var daemonList = daemon.List` in `daemon.go`, used by the bare `RunE`; tests stub it directly to return a fixed record set, bypassing the liveness self-heal entirely.
- **Files modified:** internal/cli/daemon.go, internal/cli/daemon_test.go
- **Verification:** `go test ./internal/cli/ -run TestDaemon` green
- **Committed in:** 979e285 (Task 2+3 commit)

**2. [Rule 3 - Blocking] Tasks 2 and 3 combined into one commit**
- **Found during:** Task 2 implementation
- **Issue:** Task 2's action explicitly wires `cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd())`, referencing Task 3's not-yet-written function — the package cannot compile with only Task 2's changes applied.
- **Fix:** Implemented both tasks' code, then committed once (following the same combination the 02-status-content-git-worktree-awareness plan already used for an identical whole-package-compilation reason).
- **Files modified:** internal/cli/daemon.go, internal/cli/daemon_test.go
- **Verification:** `go build ./...` and `go test ./internal/cli/ -run 'TestDaemon'` both green after the combined commit
- **Committed in:** 979e285

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking issues, both necessary to complete the plan as specified)
**Impact on plan:** No scope creep; both deviations are implementation-mechanics adjustments to satisfy the plan's own stated behavior (TUI-04 same-ordering truth, and Task 2's literal `AddCommand` wiring), not new functionality.

## Issues Encountered
None beyond the two documented deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- DMON-01, DMON-02, TUI-04 requirements complete for this plan's scope.
- `codegraph daemon` (bare/start/stop), the daemon picker Model, and the shared current-repo-first ordering are all in place; 07-08 (the plan referenced by this plan's `TUI-04` backstop verification) can now exercise the full piped-stdio never-hang path end-to-end against the real binary.
- `go build ./...`, `go vet ./...`, and `go test ./...` are all green across the module (not just this plan's packages) as of this plan's completion.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

All created/modified files verified present on disk; all task commit hashes (bacff95, 4154877, 979e285) verified present in git log.
