---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 03
subsystem: infra
tags: [go, cgo, x-sys-windows, ppid, watchdog, goleak, ci, zig-vs-mingw]

# Dependency graph
requires:
  - phase: 07-01
    provides: internal/daemon package skeleton and daemon.Run's wg.Wait()-joined goroutine-lifecycle discipline this plan mirrors
provides:
  - "startWatchdog(ctx, cancel, interval) (stop func()) — POSIX reparent detection via a captured-baseline os.Getppid() comparison (subreaper-robust), Windows parent-liveness via x/sys/windows OpenProcess+WaitForSingleObject"
  - "Joinable, goleak-clean watchdog goroutine lifecycle ready for 07-05 to wire into daemon.Run"
  - "CI windows go vet gate extended to ./internal/daemon/, backed by a real mingw-w64 cross-CGO toolchain"
affects: [07-05 (wires startWatchdog into daemon.Run and serve --mcp), phase 8 REL-01 (CGo/SBOM audit for the whole daemon closure, including this watchdog's x/sys/windows import)]

# Tech tracking
tech-stack:
  added: [golang.org/x/sys (promoted indirect→direct require; windows-only package already vetted in phase 6/3)]
  patterns:
    - "Injectable package-level func var (getppid = os.Getppid) as a test seam, mirroring daemon.go's onSyncStart/onWatchOpen convention — lets watchdog_test.go drive a synthetic reparent deterministically without forking"
    - "Captured-baseline reparent detection (compare current ppid to the value captured at daemon start) instead of a bare ppid==1 check — robust to subreaper reparenting per RESEARCH Pattern 5"
    - "stop func() that blocks on a done channel closing, mirroring daemon.Run's wg.Wait() join discipline, so a background poll goroutine never leaks past teardown (goleak-clean)"
    - "CGo cross-target CI vet gate needs a REAL cross C toolchain when the vetted package transitively pulls CGo deps (unlike the pure-Go graphstore precedent) — apt-installed gcc-mingw-w64-x86-64 + CC=x86_64-w64-mingw32-gcc, not just GOOS=windows"

key-files:
  created:
    - internal/daemon/watchdog.go
    - internal/daemon/watchdog_posix.go
    - internal/daemon/watchdog_windows.go
    - internal/daemon/watchdog_test.go
  modified:
    - go.mod
    - .github/workflows/ci.yml

key-decisions:
  - "watchdogInterval = 1s (Claude's discretion within D-07's 1-2s range)"
  - "Windows CI vet gate uses gcc-mingw-w64-x86-64 (apt) instead of the project's existing zig-cc convention (release.yml/.goreleaser.yaml) — zig cc failed locally with an AccessDenied error trying to create .zig-cache inside the read-only Go module cache directory of the tree-sitter-c CGo dependency (confirmed reproducible with both the pinned zig 0.15.1 and 0.16.0); mingw-w64's gcc has no such cache-directory behavior and vets cleanly. This is scoped only to the new internal/daemon vet step — the existing zig-based release cross-build pipeline is untouched."

patterns-established:
  - "Pattern: any future windows-tagged file inside a package that transitively imports CGo tree-sitter bindings (internal/indexer) needs the mingw-w64 toolchain step in CI, not just the graphstore precedent's bare GOOS=windows go vet."

requirements-completed: [DMON-03]

coverage:
  - id: D1
    description: "PPID watchdog detects supervising-process death via a captured-baseline getppid() comparison (POSIX) and cancels the daemon/watcher ctx, with a joinable stop() proving no goroutine leak"
    requirement: "DMON-03"
    verification:
      - kind: unit
        ref: "internal/daemon/watchdog_test.go#TestWatchdogCancelsOnReparent"
        status: pass
      - kind: unit
        ref: "internal/daemon/watchdog_test.go#TestWatchdogJoinsOnCtxCancelWithoutFiringCancel"
        status: pass
      - kind: unit
        ref: "go test ./internal/daemon/... -race -count=1 (goleak.VerifyTestMain gate, soak_test.go)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Windows parent-liveness half (watchdog_windows.go) compiles and typechecks under a real cross-CGO toolchain, and the CI gate proves it on every push (no Windows runner exists to execute it)"
    requirement: "DMON-03"
    verification:
      - kind: other
        ref: "GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go vet ./internal/daemon/"
        status: pass
    human_judgment: true
    rationale: "No Windows runtime runner exists in CI or locally; actual OpenProcess/WaitForSingleObject runtime behavior on a real Windows host is a documented divergence (RESEARCH Environment Availability), not automatable here — compile-only verification is the established locked_windows.go precedent, but a human should confirm on a real Windows box before the watchdog ships as load-bearing (07-05 wiring)."

duration: 30min
completed: 2026-07-18
status: complete
---

# Phase 07 Plan 03: PPID Reparent Watchdog Summary

**A joinable, goleak-clean PPID watchdog (POSIX captured-baseline reparent detection + Windows OpenProcess/WaitForSingleObject liveness) that cancels the daemon ctx when its supervising process dies, with the Windows half typechecked in CI via a real mingw-w64 cross toolchain.**

## Performance

- **Duration:** ~30 min
- **Completed:** 2026-07-18
- **Tasks:** 2
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments
- `startWatchdog(ctx, cancel, interval) (stop func())` — platform-independent orchestration; captures the original ppid, polls on a ticker, cancels ctx and returns on reparent, joins cleanly on ctx cancellation
- POSIX `parentChanged` via a captured-baseline `os.Getppid()` comparison — robust to subreaper reparenting, not just `ppid == 1` (RESEARCH Pattern 5)
- Windows `parentChanged`/`parentAlive` via `x/sys/windows` `OpenProcess(SYNCHRONIZE)`+`WaitForSingleObject(0)`, explicitly avoiding the unreliable `STILL_ACTIVE` sentinel
- `golang.org/x/sys` promoted from an indirect to a direct `go.mod` require
- CI's windows `go vet` gate extended to `./internal/daemon/`, backed by an apt-installed `gcc-mingw-w64-x86-64` cross toolchain (a genuine requirement discovered during this plan — see Deviations)

## Task Commits

Each task was committed atomically (Task 1 is TDD: RED → GREEN):

1. **Task 1 RED: failing watchdog tests** - `e1a931a` (test)
2. **Task 1 GREEN: startWatchdog + watchdog_posix.go** - `a6c683c` (feat)
3. **Task 2: watchdog_windows.go + x/sys direct + CI vet gate** - `fddc7af` (feat)

_No REFACTOR commit needed — GREEN implementation matched the RESEARCH code examples closely enough that no cleanup pass was warranted._

**Plan metadata:** committed via final `docs(07-03): ...` commit (see below).

## Files Created/Modified
- `internal/daemon/watchdog.go` - platform-independent `startWatchdog` orchestration, `getppid` test seam, `watchdogInterval` const
- `internal/daemon/watchdog_posix.go` - `//go:build !windows` captured-baseline `parentChanged`
- `internal/daemon/watchdog_windows.go` - `//go:build windows` `parentChanged`/`parentAlive` via x/sys/windows
- `internal/daemon/watchdog_test.go` - RED-then-GREEN reparent-cancel + ctx-cancel-join tests
- `go.mod` - `golang.org/x/sys` promoted to a direct require (`go mod tidy -e`)
- `.github/workflows/ci.yml` - new "Install mingw-w64" + "Typecheck windows PPID watchdog" steps

## Decisions Made
- `watchdogInterval = 1 * time.Second` (Claude's discretion within D-07's stated 1-2s range).
- CI's new windows vet step for `internal/daemon` uses **mingw-w64** (`gcc-mingw-w64-x86-64` apt package + `CC=x86_64-w64-mingw32-gcc`), diverging from the project's existing `zig cc` convention used by `release.yml`/`.goreleaser.yaml` for actual release cross-builds. See Deviations below for why.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `internal/daemon` transitively pulls CGo tree-sitter bindings, so a bare `GOOS=windows go vet` fails — needed a real cross-CGO toolchain, not just the graphstore precedent**
- **Found during:** Task 2 (windows vet gate verification)
- **Issue:** The plan's read_first material pointed at `internal/graphstore`'s existing `GOOS=windows go vet` CI step as the precedent to extend. Unlike `graphstore`, `internal/daemon` imports `internal/indexer` (for flush/symbol extraction), which transitively imports 13 tree-sitter grammar CGo bindings (`internal/indexer/routes`, `goextract`, etc. — the project's one justified CGo exception). Without an actual C toolchain able to target `windows/amd64`, Go auto-disables cgo for the cross-target and those bindings' build constraints exclude all files, producing `undefined: tree_sitter.Node` across `internal/indexer/routes` and `internal/indexer/goextract` — this is the exact "pre-existing, unrelated" limitation `03-REVIEW-FIX-2.md` already documented for `GOOS=windows|linux go build ./...` on this host.
- **Fix:** Verified locally that a real cross C toolchain resolves it. Tried the project's existing `zig cc -target x86_64-windows-gnu` convention (matching `.goreleaser.yaml`'s pinned windows/amd64 build entry, using both the pinned zig 0.15.1 and a newer 0.16.0) first — both reproducibly failed with `unable to open local cache directory '.../tree-sitter-c@v0.24.2/.zig-cache': AccessDenied`, because Go's module cache directories are mode `dr-xr-xr-x` (read-only by design, cross-platform) and `zig cc` tries to create a `.zig-cache` next to the C source file it's compiling — which lives inside that read-only module cache for a dependency. Switched to `gcc-mingw-w64-x86-64` (plain gcc, no cache-directory side effect) via `CC=x86_64-w64-mingw32-gcc` + `CGO_ENABLED=1`, which vets cleanly with exit 0 and zero errors.
- **Files modified:** `.github/workflows/ci.yml` (new "Install mingw-w64" step before the new vet step)
- **Verification:** `env -u CGO_ENABLED CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` → exit 0, no output, locally reproduced with the apt-equivalent Homebrew `mingw-w64` package on this dev host.
- **Committed in:** `fddc7af` (Task 2 commit)
- **Note for future work:** this raises an open question about whether `release.yml`'s existing `zig cc` windows/amd64 cross-build (which also compiles `internal/indexer`'s tree-sitter deps for the real release binary) hits the same `.zig-cache`/read-only-module-cache issue on GitHub's actual `ubuntu-latest` runners — not verified either way in this session, and explicitly out of scope for 07-03 (that's Phase 8 REL-01's CGo/reproducible-build audit territory). Flagging here so REL-01 doesn't rediscover this from scratch.

---

**Total deviations:** 1 auto-fixed (1 blocking — CI toolchain choice, no source-code architecture change)
**Impact on plan:** The watchdog implementation itself (Task 1, both files' predicates) matched the plan/RESEARCH code examples with zero deviation. The only deviation was in the CI verification mechanism for Task 2's stated acceptance criterion, and it was resolved with a locally-verified, standard, minimal fix scoped to a single new CI step — no scope creep into release.yml.

## Issues Encountered
- See Deviations #1 above — the zig-vs-mingw CI toolchain investigation was the only non-trivial issue; resolved without touching any production code.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `startWatchdog`/`stop func()` is ready for 07-05 to wire into `daemon.Run` and `serve --mcp`'s teardown paths (D-08: cancel-ctx-only, no new shutdown path).
- Windows liveness is compile-checked in CI but not runtime-verified on a real Windows host — flagged as `human_judgment: true` in this SUMMARY's coverage block; a human should confirm actual behavior on Windows before or shortly after 07-05 ships.
- The zig-vs-mingw CGo cross-compile question for `release.yml`'s actual windows release build is an open flag for Phase 8 (REL-01), not a blocker for this plan or 07-05.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

All created files verified present on disk; all task commits (`e1a931a`, `a6c683c`, `fddc7af`) verified present in `git log --oneline --all`.
