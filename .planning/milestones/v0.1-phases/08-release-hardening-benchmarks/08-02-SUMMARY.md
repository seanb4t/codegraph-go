---
phase: 08-release-hardening-benchmarks
plan: 02
subsystem: infra
tags: [benchmarking, rusage, tdd, os-exec, json]

# Dependency graph
requires: []
provides:
  - "internal/bench.PeakRSSBytes(*os.ProcessState) (int64, error) — OS-level, unit-normalized peak RSS"
  - "internal/bench.Metrics — json-tagged data holder for throughput/latency/RSS/cold-start"
affects: [08-06-regression-gate, 08-07-head-to-head-runner]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "OS-level peak-RSS capture via exec.Cmd.ProcessState.SysUsage() asserted to *syscall.Rusage, delegating unit normalization to an unexported pure helper (normalizeMaxrss) so the OS-branching logic is testable without a real child process"
    - "Plain json-tagged data-holder struct with no side-effecting methods, mirroring internal/version.VersionInfo's discipline"

key-files:
  created: [internal/bench/rss.go, internal/bench/rss_test.go, internal/bench/metrics.go]
  modified: []

key-decisions:
  - "Reworded the rss.go package doc to describe the excluded measurement path as \"in-process Go runtime memory statistics\" instead of the literal string \"runtime.MemStats\" — the plan's own acceptance criteria required a grep for that literal string to find zero matches in the package, which conflicted with the plan's action text asking for that exact phrase in the doc comment. Preserved the documented intent (never use in-process memory stats) while satisfying the grep guard."

patterns-established:
  - "Pure OS-branching helpers (normalizeMaxrss-shaped) take explicit inputs (goos, raw value) instead of reading runtime.GOOS internally, so both platform branches are deterministically testable on any host"

requirements-completed: [PERF-01, INDX-06]

coverage:
  - id: D1
    description: "PeakRSSBytes normalizes ru_maxrss to bytes correctly on both Linux (KB->bytes) and Darwin (bytes identity), and fails loud on unsupported OSes, via the exported normalizeMaxrss test seam"
    requirement: "PERF-01"
    verification:
      - kind: unit
        ref: "internal/bench/rss_test.go#TestRSSNormalize"
        status: pass
    human_judgment: false
  - id: D2
    description: "Metrics is a json-tagged data holder that round-trips through JSON without field loss"
    requirement: "INDX-06"
    verification:
      - kind: unit
        ref: "internal/bench/rss_test.go#TestSmoke"
        status: pass
    human_judgment: false

# Metrics
duration: 6min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 02: RSS Normalization + Metrics Core Summary

**OS-level peak-RSS capture (`PeakRSSBytes`) with Linux-KB-vs-Darwin-bytes normalization, plus a json-tagged `Metrics` data holder, both proven test-first (RED→GREEN).**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-13T17:04:00Z (approx, following 08-01)
- **Completed:** 2026-07-13T17:10:23Z
- **Tasks:** 2 completed
- **Files modified:** 3 (`internal/bench/rss.go`, `internal/bench/rss_test.go`, `internal/bench/metrics.go`, all created)

## Accomplishments
- `internal/bench.PeakRSSBytes(*os.ProcessState) (int64, error)` reads `syscall.Rusage.Maxrss` off a completed child process and normalizes it to bytes, delegating the OS-branching arithmetic to an unexported, deterministically-testable `normalizeMaxrss(goos string, rawMaxrss int64) (int64, error)` helper
- `normalizeMaxrss` implements the exact Pitfall-4 contract: Linux `ru_maxrss` (KB) × 1024, Darwin `ru_maxrss` (bytes) unchanged, any other OS returns a loud error and zero value — never a silent 1024x corruption
- `internal/bench.Metrics` is a plain json-tagged struct (Subject, Repo, GOOS, GOARCH, FilesPerSec, BytesPerSec, QueryLatencyMedianMS, PeakRSSBytes, ColdStartMS) mirroring `internal/version.VersionInfo`'s no-side-effects discipline
- TDD gate proven in git history: `test(08-02)` commit fails to compile (RED), `feat(08-02)` commit passes both tests and `go vet` (GREEN)

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Write failing tests for RSS normalization and the Metrics smoke path** - `1618c9f` (test)
2. **Task 2 (GREEN): Implement rss.go + metrics.go to pass the tests** - `b1099da` (feat)

**Plan metadata:** (this commit)

_Note: no REFACTOR commit — the GREEN implementation was already minimal and needed no cleanup pass._

## Files Created/Modified
- `internal/bench/rss_test.go` - `TestRSSNormalize` (table test driving linux/darwin/unsupported-OS branches of `normalizeMaxrss`) + `TestSmoke` (JSON round-trip of `Metrics`)
- `internal/bench/rss.go` - `PeakRSSBytes` (OS-level peak-RSS reader) + `normalizeMaxrss` (unit-normalization helper) + package doc stating the external-measurement, no-network/no-crypto/no-panic contract
- `internal/bench/metrics.go` - `Metrics` struct (json-tagged data holder)

## Decisions Made
- Reworded the package doc comment in `rss.go` to avoid the literal substring `runtime.MemStats` (see key-decisions above) — the plan's acceptance criteria's grep guard and its own action-text instruction were mutually contradictory; resolved by preserving the documented intent through a paraphrase instead of the exact banned string.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Reworded package doc comment to avoid literal `runtime.MemStats` string**
- **Found during:** Task 2 (writing the `rss.go` package doc per the plan's action text)
- **Issue:** The plan's action text instructed the package doc to state "never reads runtime.MemStats" verbatim, but the plan's own acceptance criteria required `grep -q 'runtime.MemStats' internal/bench/*.go` to find NO match — a literal self-contradiction, since grep matches comment text same as code.
- **Fix:** Paraphrased the doc comment to say "never via in-process Go runtime memory statistics" instead of the literal API name, preserving the documented discipline while satisfying the grep guard.
- **Files modified:** `internal/bench/rss.go`
- **Verification:** `grep -n 'runtime.MemStats' internal/bench/*.go` returns no match (exit 1); tests and `go vet` still pass.
- **Committed in:** `b1099da` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix — plan wording self-contradiction)
**Impact on plan:** Cosmetic wording fix only; no behavioral change. No scope creep.

## Issues Encountered
None beyond the plan-wording contradiction documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/bench.PeakRSSBytes` and `internal/bench.Metrics` are ready for Plan 08-06's committed-baseline regression gate and Plan 08-07's head-to-head Go-vs-TS runner to consume directly
- No blockers

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: internal/bench/rss.go
- FOUND: internal/bench/rss_test.go
- FOUND: internal/bench/metrics.go
- FOUND: 1618c9f (Task 1 commit)
- FOUND: b1099da (Task 2 commit)
