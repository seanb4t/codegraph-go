---
phase: 07-interactive-tui-daemon-picker-install-multi-select
reviewed: 2026-07-18T21:23:35Z
depth: deep
files_reviewed: 20
files_reviewed_list:
  - internal/cli/daemon.go
  - internal/cli/daemon_test.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/uninstall.go
  - internal/cli/tui/tty.go
  - internal/cli/tui/agentpicker.go
  - internal/cli/tui/daemonpicker.go
  - internal/cli/tui/daemonpicker_test.go
  - internal/cli/tui/doc.go
  - internal/daemon/daemon.go
  - internal/daemon/registry.go
  - internal/daemon/watchdog.go
  - internal/daemon/watchdog_posix.go
  - internal/daemon/watchdog_windows.go
  - internal/daemon/stop.go
  - internal/daemon/stop_posix.go
  - internal/daemon/stop_windows.go
  - internal/githooks/githooks_test.go
  - test/integration/piped_never_hang_test.go
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: clean
---

# Phase 07: Code Review Report (iteration 2 — post-fix re-review)

**Reviewed:** 2026-07-18
**Depth:** deep
**Files Reviewed:** 20 (scope per config; `.github/workflows/ci.yml` also in scope, unchanged since iter 1, reviewed below)
**Status:** clean

## Summary

This is iteration 2 of the `--auto` fix loop, re-reviewing after commits
`e62f4d8` (WR-01), `3888a46` (WR-02), and `c7509fb` (WR-03) from
`07-REVIEW.iter2.md`. `git diff --stat 9dd7d71 c7509fb` confirms the fix
commits touched exactly four files: `internal/cli/tui/daemonpicker.go`,
`internal/cli/tui/daemonpicker_test.go`, `internal/cli/daemon.go`,
`internal/cli/daemon_test.go`. Everything else in scope is byte-identical to
what iteration 1 already deep-reviewed and cleared.

**Job 1 — verifying the 3 prior Warnings are genuinely resolved (not papered
over):**

1. **WR-01 (stopped-daemon confirmation)** — `dispatchDaemonAction` now
   returns `([]daemon.Record, error)` (`daemonpicker.go:238-247`), and
   `RunDaemonPicker` (`daemonpicker.go:283-299`) calls the new
   `printDaemonPickerResult` unconditionally after dispatch, on
   `cmd.OutOrStdout()`, before returning the error. I traced both action
   paths: stop-one (`daemonActionStopOne` → `stopMatching(target.RepoRoot)`)
   and stop-all (`daemonActionStopAll` → `stopAll()`) both flow through the
   same `dispatchDaemonAction` → `printDaemonPickerResult` sequence, so a
   partial stop-all failure now prints the successes it did signal (not just
   swallow them behind the error), matching the non-interactive path's
   behavior. The per-record line format (`"stopped pid %d (%s)\n"`,
   `daemonpicker.go:269` vs `daemon.go:248` `printStoppedDaemons`) is
   byte-for-byte identical between the two duplicated helpers — no drift.
   `daemonActionNone` (cancel/quit/empty-registry) is explicitly excluded
   (`daemonpicker.go:261-263`), so cancelling still prints nothing, as
   before. Verified via `TestPrintDaemonPickerResult`'s four table cases and
   `TestDaemonPickerModel_EnterDispatchesStopOne`'s assertion on the returned
   `stopped` slice — both pass (`go test ./internal/cli/tui/...`).
2. **WR-02 (`--all`/`--path` mutual exclusion)** — `cmd.MarkFlagsMutuallyExclusive("path", "all")`
   is registered on `newDaemonStopCmd`'s own `FlagSet` (`daemon.go:239`),
   which cobra validates in `ValidateFlagGroups()` before `RunE` ever runs —
   confirmed by `TestDaemonStopCmd_AllAndPath_MutuallyExclusive`
   (`daemon_test.go:290-304`), which stubs both `daemonStopMatching`/
   `daemonStopAll` to `t.Fatal` if called and asserts `daemon stop --all
   --path /repo` returns a non-nil error without either ever firing. Plain
   `daemon stop --all` and `daemon stop --path <p>` are untouched code paths
   (mutual exclusion only triggers when cobra's `Changed()` is true for both
   flags) and their pre-existing tests
   (`TestDaemonStopCmd_All_DispatchesToStopAll`,
   `TestDaemonStopCmd_DispatchesToStopMatching`) still pass.
3. **WR-03 (symlink-aware sort)** — `SortRecordsCurrentFirst`
   (`daemonpicker.go:90-123`) now pre-resolves every record's `RepoRoot`
   (plus `currentRepo`) into a single `entry{record, resolved}` slice
   *before* calling `sort.SliceStable`, and swaps whole `entry` structs — so
   there is no parallel-slice desync risk (verified by inspection: one
   slice, one comparator, no separate index-aligned lookup table).
   `resolveRepoRoot` (`daemonpicker.go:67-72`) degrades to the raw string on
   any `EvalSymlinks` error, never panicking. Ordering is still
   deterministic: primary key is `resolved == resolvedCurrent`, secondary key
   is the *raw* `RepoRoot` string, tertiary is `PID` — a total order with no
   ties left unresolved. `TestSortRecordsCurrentFirst_NormalizesSymlinks`
   (real symlink via `os.Symlink` + `t.TempDir()`) and
   `TestSortRecordsCurrentFirst_SymlinkEvalErrorDegradesToRawString`
   (nonexistent paths) both pass, and I confirmed
   `internal/daemon/stop.go`'s own `resolveRepoRoot` (`stop.go:37-42`) is
   textually identical in behavior (same `EvalSymlinks`-or-fallback shape),
   so the picker's "is this my repo" ordering answer and `StopMatching`'s
   targeting answer can no longer diverge on a symlinked path.

**Job 2 — adversarial check of the fix edits for new defects:** None found.
Specifically checked and cleared:
- The paired-slice sort: empty input (`entries := make([]entry, 0)`, no
  panic, returns an empty-but-non-nil slice — a harmless nil-vs-empty
  cosmetic difference from the pre-fix behavior, not a bug); single-element
  input (comparator never invoked, trivially correct); all-equal `RepoRoot`
  (falls through to the `PID` tertiary key, still a total, deterministic
  order).
- `dispatchDaemonAction`'s actual signal-targeting logic still reads
  `target.RepoRoot` (the raw, unresolved string) off the *sorted* record —
  the resolved-path comparison introduced by WR-03 is used only for display
  ordering, never substituted into the value passed to
  `stopMatching`/`daemon.StopMatching`, so no behavior change to which
  daemon actually gets signaled.
- The duplicated `resolveRepoRoot` (tui) vs `internal/daemon/stop.go`'s
  `resolveRepoRoot`: identical implementations, confirmed side-by-side.
- The duplicated `printDaemonPickerResult` (tui) vs `internal/cli/daemon.go`'s
  `printStoppedDaemons`: identical per-record format string; the only
  difference is the empty-case wording (`"nothing stopped"` vs the
  non-interactive path's `"no running daemon(s)"` variants) — see IN-03
  below, an awareness note, not a defect.
- Charm confinement: `internal/cli/present/archtest/import_graph_test.go`'s
  `guardedPackages` list does not include `internal/cli` or any subpackage
  (explicitly excluded via `excludedInternalPackagePrefixes`), and
  `internal/daemon` (which IS guarded) imports neither `internal/cli` nor
  `internal/cli/tui` — so `daemonpicker.go`'s `charm.land/bubbles/v2` and
  `charm.land/bubbletea/v2` imports never enter the serve-reachable closure
  the archtest polices. `go test ./internal/cli/present/archtest/...`
  passes.
- `go build ./...`, `go vet ./...`, and `gofmt -l` on all four touched files
  are all clean.

**Job 3 — full high-risk surface re-scan (unchanged files):** Re-read
`daemon.Run`'s full defer/goroutine-join ordering, both `watchdog_*.go`
variants, both `stop_*.go` signal variants, `registry.go`'s
self-heal/atomic-write path, and `tty.go`'s `InteractiveAllowed` gate. All
byte-identical to what iteration 1 traced and cleared (confirmed via
`git diff --stat` against the pre-fix commit showing zero changes to these
files). No new goroutine leak, deadlock, race, or never-hang violation
surfaced. `go test ./internal/daemon/... ./internal/cli/... ./test/integration/...`
all pass, including the goleak-gated `TestMain`s.

No Critical or Warning findings. The two Info items iteration 1 explicitly
marked "no fix required" still hold and are carried forward for the record;
one new Info-level wording observation (IN-03) is added, also not a defect.

## Info

### IN-01: Daemon picker/list rendering does not dedupe records by pid (carried forward, still holds)

**File:** `internal/cli/tui/daemonpicker.go:155-167`, `internal/cli/daemon.go:92-103`
**Issue:** `stopTargets` (`internal/daemon/stop.go:60-67`) de-duplicates
targets by pid before signaling, but `newDaemonPickerModel`/`printDaemonList`
still render every record `List()` returns verbatim, with no pid-based
dedup. Only reachable via a hand-edited or migrated-from-TS registry
directory with two on-disk records for the same live pid.
**Fix:** Not urgent. If ever hardened, dedupe by pid the same way
`stopTargets` does. (Skipped-by-design in iteration 1's fix pass; still
correct to skip.)

### IN-02: `daemon.List()` runs its self-heal side effect before the interactivity check (carried forward, still holds)

**File:** `internal/cli/daemon.go:69-77`
**Issue:** `daemonList()` is called unconditionally before
`interactiveAllowed(cmd)` is checked, so every `codegraph daemon` invocation
(interactive or not) prunes stale registry records as a side effect of
merely listing them.
**Fix:** None needed — explicitly documented/intended behavior per
`registry.go:74-83`'s "no background reaper" design.

### IN-03: `printDaemonPickerResult`'s "nothing matched" wording diverges from `printStoppedDaemons`'s callers

**File:** `internal/cli/tui/daemonpicker.go:264-266`, `internal/cli/daemon.go:208-210,225-227`
**Issue:** WR-01's fix explicitly aimed for "the same
`printStoppedDaemons`-shaped confirmation lines (or a 'nothing to stop'
notice)" between the interactive and non-interactive paths. The per-record
"stopped pid %d (%s)" line is identical between both, but the empty-result
notice differs in wording: the interactive picker prints `"nothing
stopped"` while `daemon stop`/`daemon stop --all` print `"no running daemon
for %s"`/`"no running daemons"` respectively. A user who runs both entry
points for the equivalent "nothing to stop" outcome sees different English,
which is a minor UX inconsistency, not a correctness defect — dispatch,
signaling, and exit codes are unaffected.
**Fix:** Optional. If pursued, have `RunDaemonPicker` reuse
`daemon.go`'s exact wording (would require exporting or threading the
target/action context through, since the non-interactive messages are
target-specific ("no running daemon for %s") vs stop-all's generic message)
— not worth the churn unless a future pass unifies these two entry points
more broadly.

---

_Reviewed: 2026-07-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
