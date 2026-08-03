---
phase: 07-interactive-tui-daemon-picker-install-multi-select
fixed_at: 2026-07-18T21:30:00Z
review_path: .planning/phases/07-interactive-tui-daemon-picker-install-multi-select/07-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 3
skipped: 2
status: partial
---

# Phase 07: Code Review Fix Report

**Fixed at:** 2026-07-18
**Source review:** .planning/phases/07-interactive-tui-daemon-picker-install-multi-select/07-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (0 Critical, 3 Warning, 2 Info — `fix_scope: all`)
- Fixed: 3
- Skipped: 2

## Fixed Issues

### WR-01: Interactive daemon picker gives zero confirmation of what it actually stopped

**Files modified:** `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/daemonpicker_test.go`
**Commit:** e62f4d8
**Applied fix:** Changed `dispatchDaemonAction`'s signature from `(daemonAction, daemon.Record) error` to `(daemonAction, daemon.Record) ([]daemon.Record, error)`, surfacing the `[]daemon.Record` that `stopMatching`/`stopAll` actually signaled instead of discarding it. Added a new `printDaemonPickerResult` helper (duplicated from `internal/cli/daemon.go`'s `printStoppedDaemons` shape rather than imported, to avoid an `internal/cli/tui` -> `internal/cli` import cycle — `internal/cli` already imports `internal/cli/tui`) that prints one "stopped pid %d (%s)" line per record, a "nothing stopped" notice when the action attempted a stop but nothing matched, and nothing at all for `daemonActionNone` (cancel/quit/empty-registry, which already print nothing by design). `RunDaemonPicker` now calls this after dispatch, using `cmd.OutOrStdout()`. Updated all 5 existing `dispatchDaemonAction` call sites in `daemonpicker_test.go` for the new two-value return, strengthened `TestDaemonPickerModel_EnterDispatchesStopOne` to assert on the returned `stopped` slice, and added `TestPrintDaemonPickerResult` (table-driven, covers stop-one, stop-all/multi-record, no-match, and cancel-is-silent).

### WR-02: `daemon stop --all --path <p>` silently ignores `--path` with no validation

**Files modified:** `internal/cli/daemon.go`, `internal/cli/daemon_test.go`
**Commit:** 3888a46
**Applied fix:** Added `cmd.MarkFlagsMutuallyExclusive("path", "all")` to `newDaemonStopCmd`, so cobra rejects `--all --path <p>` (surfaced in both `--help` and as a RunE error) instead of `--all`'s branch silently winning before `path` is ever resolved. Added `TestDaemonStopCmd_AllAndPath_MutuallyExclusive`, which stubs both `daemonStopMatching`/`daemonStopAll` to `t.Fatal` if called and asserts `daemon stop --all --path /repo` returns a non-nil error without ever dispatching to either.

### WR-03: `SortRecordsCurrentFirst` uses plain string equality while the actual stop path resolves symlinks

**Files modified:** `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/daemonpicker_test.go`
**Commit:** c7509fb
**Applied fix:** Added a `resolveRepoRoot` helper to `daemonpicker.go` — duplicated (not imported; it's unexported in `internal/daemon/stop.go`, and the project already has this exact cross-package-duplication precedent in `internal/gitmeta/worktree.go`'s `realpath` and `internal/agents/cursor.go`) — that normalizes a path via `filepath.EvalSymlinks`, degrading to the raw string on any error (path doesn't exist, permission denied, etc.), never breaking the sort. Rewrote `SortRecordsCurrentFirst` to pre-resolve `currentRepo` and every record's `RepoRoot` once into a paired `(record, resolved)` slice before sorting (avoiding both the parallel-slice desync bug that a naive index-aligned precompute would hit under `sort.SliceStable`'s swaps, and redundant repeated `EvalSymlinks` syscalls per comparator call), then compares the *resolved* paths for the "is this my repo" ordering decision while still using the raw `RepoRoot` strings for the secondary stable sort key. Added `TestSortRecordsCurrentFirst_NormalizesSymlinks` (creates a real temp dir + a symlink to it, asserts a record recorded through the symlinked path still sorts first when `currentRepo` is the real path) and `TestSortRecordsCurrentFirst_SymlinkEvalErrorDegradesToRawString` (nonexistent paths still order correctly via the raw-string fallback).

## Skipped Issues

### IN-01: Daemon picker/list rendering does not dedupe records by pid

**File:** `internal/cli/tui/daemonpicker.go:111-123`, `internal/cli/daemon.go:92-103`
**Reason:** skipped-by-design. The reviewer's own Fix guidance for this finding is "Not urgent — noting for awareness. If ever hardened, dedupe by pid..." — this is not a defect requiring a fix, and per the fixer's operating instructions ("apply only if genuinely trivial + safe; otherwise record as skipped-by-design"), introducing pid-based dedup into both `newDaemonPickerModel` and `printDaemonList` is a deliberate design decision (matching `stopTargets`' `seen[rec.PID]` dedup semantics in a second place) rather than a trivial, unambiguous bug fix — it only matters for a hand-edited or migrated-from-TS registry directory with two on-disk records for the same live pid, an edge case the normal Register/Deregister lifecycle can't produce. Deferred to a future hardening pass if it's ever prioritized.
**Original issue:** `stopTargets` de-duplicates targets by pid before signaling, but `newDaemonPickerModel`/`printDaemonList` render every record `List()` returns verbatim with no pid-based dedup — a duplicate on-disk record for the same pid would render as two rows even though only one process exists.

### IN-02: `daemon.List()` runs its self-heal side effect on every bare `codegraph daemon` invocation, before the interactivity check

**File:** `internal/cli/daemon.go:69-77`
**Reason:** skipped-by-design. The reviewer's Fix guidance is explicit: "None needed; documenting as expected behavior, not a defect." This finding was raised purely for awareness of `registry.go`'s already-documented "no background reaper" design — there is nothing to fix.
**Original issue:** `daemonList()` runs unconditionally before `interactiveAllowed(cmd)` is checked, so every `codegraph daemon` invocation (interactive or not) prunes stale registry records as a side effect of merely listing them — intended behavior per `registry.go:74-83`, not a bug.

---

_Fixed: 2026-07-18_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
