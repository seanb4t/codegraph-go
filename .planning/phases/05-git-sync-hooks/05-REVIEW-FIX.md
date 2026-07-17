---
phase: 05-git-sync-hooks
fixed_at: 2026-07-17T02:53:12Z
review_path: .planning/phases/05-git-sync-hooks/05-REVIEW.md
iteration: 7
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 05: Code Review Fix Report

**Fixed at:** 2026-07-17T02:53:12Z
**Source review:** .planning/phases/05-git-sync-hooks/05-REVIEW.md
**Iteration:** 7

**Summary:**
- Findings in scope: 2 (WR-01, IN-01)
- Fixed: 2
- Skipped: 0

## Fixed Issues

### WR-01: `Remove` still surfaces zero signal for a malformed marker block, unlike `Install`'s identical-condition handling

**Files modified:** `internal/githooks/githooks.go`, `internal/githooks/githooks_test.go`, `internal/cli/githooks_test.go`, `internal/cli/uninit_test.go`
**Commit:** `a3118d3`
**Applied fix:** Mirrored `Install`'s CR-01 handling in `Remove`'s `stripMarkerBlock` `ok == false` branch — it now appends `fmt.Errorf("%s: hook file has a malformed codegraph marker block — please fix or remove it manually", hook)` to `RemoveResult.Errors` before `continue`, in addition to the existing correct behavior of leaving the file untouched and not reporting it as removed. This closes the last "same detection semantics, same skip-untouched convention, but not the same error-surfacing" gap in the `{Install,Remove,Status}` × `{read-error,malformed-marker}` matrix this round's task asked to verify was closed.

Added three regression tests:
- `TestRemove_MalformedMarkerBlock_LeavesFileUntouchedAndAccumulatesError` (`internal/githooks/githooks_test.go`, package-level, driving `Remove` against the existing `malformedHookFixture()`) — asserts the hook is absent from `Removed`, present in `Errors` (naming the hook), and the file is byte-for-byte unchanged.
- `TestGithooksRemove_MalformedMarkerBlock_SurfacesWarning` (`internal/cli/githooks_test.go`) — CLI-level: the standalone `githooks remove` command now prints the malformed-marker warning to stderr for a hand-damaged hook fixture (previously only tested for an unwritable directory in this file).
- `TestUninit_MalformedHook_SurfacesWarning` (`internal/cli/uninit_test.go`) — CLI-level: `uninit`'s D-06 best-effort cleanup surfaces the same warning on stderr via `printHookErrors`, rather than silently removing `.codegraph/` and reporting success while a broken hook file is left behind with no signal.

### IN-01: `WR-02` label reused for two unrelated findings within the same file

**Files modified:** `internal/githooks/githooks.go`
**Commit:** `dc5fa27`
**Applied fix:** Renamed the newer inline comment's tag in `Remove`'s read-error branch — previously `// WR-02: mirror Install's CR-02 distinction...` — to `// Read-error accumulation (round-5 fix, distinct from the Concurrency (WR-02) doc-comment finding above): ...`. This de-duplicates the label from the pre-existing "Concurrency (WR-02)" doc-comment finding on `Install`'s/`Remove`'s doc comments (first recorded several review iterations ago, `fe3f4e1`) so a reader searching this file for "WR-02" no longer finds two unrelated defects sharing one ID from two different review cycles' numbering namespaces. Documentation-only change; no functional impact.

## Skipped Issues

None — all findings were fixed.

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./internal/githooks/... ./internal/cli/...` — all pass, including the 3 new regression tests
- `go test ./...` (full suite) — one unrelated failure, `internal/daemon`'s `TestDaemonFlushLockRequeueGivesUpPerEpisode` (a timing-sensitive lock-contention test); re-running that test in isolation passed cleanly, and `internal/daemon` was not touched by this fix — confirmed pre-existing flake, not a regression.

---

_Fixed: 2026-07-17T02:53:12Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 7_
