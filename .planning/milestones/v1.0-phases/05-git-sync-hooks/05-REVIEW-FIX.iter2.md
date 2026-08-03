---
phase: 05-git-sync-hooks
fixed_at: 2026-07-17T02:05:39Z
review_path: .planning/phases/05-git-sync-hooks/05-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-07-17T02:05:39Z
**Source review:** .planning/phases/05-git-sync-hooks/05-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (fix_scope: all — critical, warning, and info)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: Unterminated/malformed marker block silently destroys trailing file content

**Files modified:** `internal/githooks/githooks.go`, `internal/githooks/githooks_test.go`
**Commit:** `3be729f`
**Applied fix:** `stripMarkerBlock` now returns `(string, bool)` — the bool
is `false` when a begin marker has no matching end marker, in which case
the original content is returned untouched rather than truncated at the
begin marker. `Install` falls back to the raw existing content as its base
(no data loss, dangling marker preserved) and `Remove` skips the hook
entirely (not reported as removed) when the strip can't be trusted. This
is a deliberate, narrowly-scoped Go-only divergence from TS's inherited
data-loss bug — well-formed marker-block behavior stays byte-for-byte TS
semantics per 05-CONTEXT D-02/D-03; only the malformed/unterminated-input
path changed. Added a unit regression test on `stripMarkerBlock` directly
and an end-to-end `Remove` regression test asserting a malformed hook file
with content both before and after a dangling begin marker is left
completely unchanged.

### WR-01: Install/Remove silently swallow per-hook write/delete errors with no diagnostic

**Files modified:** `internal/githooks/githooks.go`, `internal/cli/githooks.go`
**Commit:** `c8ce880`
**Applied fix:** Added an `Errors []error` field to both `InstallResult`
and `RemoveResult`; each per-hook write/delete failure is now wrapped with
`fmt.Errorf("%s: %w", hook, err)` and appended rather than discarded (the
loop still continues past a failure). `newGithooksInstallCmd`/
`newGithooksRemoveCmd` print one `warning: ...` line per accumulated error
to stderr via a new `printHookErrors` helper.

### WR-02: No synchronization across the read-modify-write sequence in Install/Remove

**Files modified:** `internal/githooks/githooks.go`
**Commit:** `fe3f4e1`
**Applied fix:** Documentation-only, per the review's own "low-priority,
worth at least documenting" guidance — added an explicit "Concurrency"
doc-comment section to both `Install` and `Remove` stating they are not
safe to call concurrently against the same `projectRoot` (individual
writes are atomic via `fsatomic.WriteFile`, but the surrounding
read-modify-write sequence is not). No locking was added; this is a
rarely-concurrent CLI operation as the review itself notes.

### WR-03: Missing test coverage for install/remove failure and partial-failure paths

**Files modified:** `internal/githooks/githooks_test.go`, `internal/cli/githooks_test.go`
**Commit:** `cded5ab`
**Applied fix:** Added tests for all four previously-untested reachable
paths named in the finding: `Install`'s "could not access the git hooks
directory" skip branch (hooks-dir path pre-occupied by a regular file), a
partial-success `Install` (one hook's target path is a directory so its
write fails while the other two succeed) asserting the WR-01 `Errors`
field, an unwritable-hooks-dir `Remove` failure (chmod-based, skipped
under root/Windows per the existing `internal/upgrade` convention), the
CLI's "Could not install git hooks..." sync-fallback message (all three
hook writes fail), and the CLI's "No git sync hooks were installed —
nothing to remove." message.

### IN-01: Duplicate pluralization logic between uninit.go and githooks.go

**Files modified:** `internal/cli/githooks.go`
**Commit:** `80b8973`
**Applied fix:** Replaced the inline `suffix := "s"; if len(...) == 1 {
suffix = "" }` block in `newGithooksRemoveCmd` with a call to the existing
`plural(len(result.Removed))` helper already defined in `uninit.go` (same
package `cli`, no new import needed).

### IN-02: Watcher-enabled advisory test depends on ambient environment rather than asserting it

**Files modified:** `internal/cli/init_advisory_test.go`
**Commit:** `8681757`
**Applied fix:** Added `t.Setenv("CODEGRAPH_NO_WATCH", "")` at the top of
`TestInitAdvisory_WatcherEnabled` to make the "watcher enabled"
precondition explicit rather than relying on the ambient
test/CI/developer-shell environment happening not to have the env var set.

### IN-03: `githooks status`/TS's `isSyncHookInstalled` report "installed" based on marker text only, not executability

**Files modified:** `internal/githooks/githooks.go`, `internal/githooks/githooks_test.go`, `internal/cli/githooks.go`, `internal/cli/githooks_test.go`
**Commit:** `5bc6348`
**Applied fix:** Added a `HookStatus.Executable bool` field (Go-only,
beyond TS parity) computed via `info.Mode().Perm()&0o111 != 0` alongside
the existing marker-text `Installed` check. `githooks status` now prints a
distinct `"installed but not executable"` state instead of folding
exec-bit health into the flat `installed`/`not installed` boolean. Added a
unit test on `Status` and an end-to-end CLI test asserting the new status
line.

## Skipped Issues

None — all findings were fixed.

---

_Fixed: 2026-07-17T02:05:39Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
