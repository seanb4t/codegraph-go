---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 08
subsystem: testing
tags: [go, githooks, integration-testing, bubbletea, tty, cli]

# Dependency graph
requires:
  - phase: 07-06
    provides: install/uninstall's off-TTY auto-fallback (interactiveAllowed/runAgentPicker seam)
  - phase: 07-07
    provides: the restructured daemon command tree (bare picker/list, start, stop) and its off-TTY plain-list fallback
provides:
  - "internal/githooks/githooks_test.go::TestInstall_EditThenRemove_ByteInvariant — proves install -> user edit outside the marker block -> remove is byte-identical to (pre-install original + edit)"
  - "test/integration/piped_never_hang_test.go::TestPipedNeverHang — proves the real binary never hangs on `daemon` (bare) or `install` with piped/closed stdio, under a bounded timeout"
affects: [phase-08-release-hardening]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Bounded goroutine+select-against-time.After wrapper around a subprocess call, so a hang fails the test instead of blocking go test (mirrors watch_default_test.go's context.WithTimeout convention)"
    - "Throwaway HOME/USERPROFILE env vars passed to runBinary so subprocess tests never mutate the developer's real ~/.codegraph daemon registry or real agent configs"

key-files:
  created:
    - test/integration/piped_never_hang_test.go
  modified:
    - internal/githooks/githooks_test.go

key-decisions:
  - "Simulated the user's out-of-marker-block edit as an inserted line in the surviving base content (before the block, which always sits at end-of-file per Install's strip-then-append-at-end semantics) rather than appended after the block — appending after the block round-trips through an extra blank line from the block's own separator/trailing-newline bytes, which is not what 'byte-identical to original + edit' means; editing the base content is the faithful simulation of a real hand-edit and is a clean byte-invariant round trip against the existing implementation (no bug found, GREEN as expected)."
  - "install's off-TTY never-hang case exercises the real auto-fallback path (no --target flag) rather than --target none, since D-17/TUI-04 specifically wants the auto-fallback branch proven, not the explicit-target branch; both daemon and install subtests run under a throwaway HOME so neither mutates real developer state (T-07-08-02)."

requirements-completed: [TEST-03, TUI-04]

coverage:
  - id: D1
    description: "githooks install -> user edit outside marker block -> remove returns the file byte-identical to (pre-install original + edit), with install->install fixed point and remove->remove no-op backstopped in the same test"
    requirement: "TEST-03"
    verification:
      - kind: unit
        ref: "internal/githooks/githooks_test.go#TestInstall_EditThenRemove_ByteInvariant"
        status: pass
    human_judgment: false
  - id: D2
    description: "daemon (bare) and install, spawned as the real binary with piped/closed stdin+stdout under a bounded timeout, never hang and exit with non-interactive output (no ANSI escape leakage)"
    requirement: "TUI-04"
    verification:
      - kind: integration
        ref: "test/integration/piped_never_hang_test.go#TestPipedNeverHang/daemon_bare"
        status: pass
      - kind: integration
        ref: "test/integration/piped_never_hang_test.go#TestPipedNeverHang/install_auto_fallback"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-18
status: complete
---

# Phase 7 Plan 08: TEST-03 Byte-Invariance & Piped Never-Hang Summary

**Two new tests close Phase 7's last verification gap: a githooks round-trip byte-invariance test and a real-binary piped-stdio never-hang harness for `daemon`/`install`.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-07-18
- **Tasks:** 2
- **Files modified:** 2 (1 new, 1 extended)

## Accomplishments
- `TestInstall_EditThenRemove_ByteInvariant` proves install -> user edit outside the marker block -> remove returns the hook file byte-identical to the pre-install original plus the user's edit, with the marker block fully stripped — closing the exact gap RESEARCH identified (the existing `TestRemove_WithUserContent_PreservesRemainderBytes` only covered remove-preserving-remainder, not the full round trip).
- The same test backstops install->install (fixed point) and remove->remove (clean no-op) against this fixture.
- New `test/integration/piped_never_hang_test.go` spawns the real, built binary for `daemon` (bare) and `install` with piped stdin+stdout under a 10s bounded timeout, in a goroutine racing `time.After` — a hang fails the test instead of blocking `go test` itself.
- Both never-hang subtests run under a throwaway `HOME`/`USERPROFILE`, proving neither command's off-TTY fallback (`daemon` -> plain list, `install` -> auto) prompts or mutates real developer state, and asserting stdout/stderr carry no ANSI escape sequences (interactive-TUI leak detection).

## Task Commits

Each task was committed atomically:

1. **Task 1: githooks install->edit->remove byte-invariance test (D-16)** - `220de37` (test)
2. **Task 2: piped never-hang integration cases for daemon + install (D-17)** - `84ebfbf` (test)

**Plan metadata:** (this commit) - `docs(07-08): complete TEST-03 byte-invariance & piped never-hang plan`

## Files Created/Modified
- `internal/githooks/githooks_test.go` - Added `TestInstall_EditThenRemove_ByteInvariant`
- `test/integration/piped_never_hang_test.go` - New: `TestPipedNeverHang` (daemon bare + install auto-fallback, piped stdio, bounded timeout)

## Decisions Made
- The out-of-marker-block user edit is simulated as a line inserted into the surviving base content (before the block), not appended after it — appending after the block introduces an extra blank line from the block's own trailing-newline bytes during strip, which would make the byte-identity assertion fail for a reason unrelated to the actual invariant under test. Editing the base content is the faithful "user hand-edits the hook file" simulation and produces a genuinely byte-identical round trip against the existing Phase-5 implementation (verified GREEN, no bug found).
- `install`'s never-hang subtest exercises the real off-TTY auto-fallback (no `--target` flag), not `--target none`, since D-17/TUI-04's contract is specifically about the auto-fallback branch never hanging; a throwaway `HOME` keeps this safe to run without touching real agent configs.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Mid-session, an accidental `git stash` / `git stash pop` pair (used briefly to isolate a pre-existing `GOOS=windows go vet` failure) popped an unrelated, pre-existing stash entry from an old `phase-06` branch session, producing a merge conflict in `internal/cli/root.go`. Recovered immediately via a targeted `git checkout HEAD -- internal/cli/root.go` (restoring the file to the committed HEAD state, verified via `git diff HEAD` showing no delta) without touching the stash entry itself (left in the stash list, not mine to drop). No commits, no other files, and no test results were affected by this detour; confirmed via `git status --short` and a full rebuild/retest immediately after.
- `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/ ./internal/graphstore/` fails with `undefined: tree_sitter.Node` (CGo tree-sitter grammar bindings excluded under windows/amd64 build constraints) — confirmed **pre-existing and unrelated to this plan**: reproduced identically with none of this plan's changes applied. Not fixed here (out of this plan's scope — test-file-only changes cannot affect a cross-compilation constraint in `internal/indexer/goextract`); flagged for Phase 8 (release hardening) per STATE.md's existing Charm v2/CGo audit blocker note.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- TEST-03 and TUI-04 are both proven against the real implementation; Phase 7 (Interactive TUI — Daemon Picker & Install Multi-Select) is now fully executed (8/8 plans).
- The pre-existing `GOOS=windows go vet` cross-compilation failure in `internal/indexer/goextract`/`routes` (CGo tree-sitter bindings excluded under windows/amd64) remains open and should be addressed in Phase 8 alongside the already-tracked Charm v2 CGo/govulncheck/SBOM audit.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

- FOUND: internal/githooks/githooks_test.go
- FOUND: test/integration/piped_never_hang_test.go
- FOUND: .planning/phases/07-interactive-tui-daemon-picker-install-multi-select/07-08-SUMMARY.md
- FOUND commit 220de37 (test: githooks byte-invariance)
- FOUND commit 84ebfbf (test: piped never-hang)
