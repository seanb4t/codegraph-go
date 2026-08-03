---
phase: 04-output-hygiene
fixed_at: 2026-07-16T19:40:00Z
review_path: .planning/phases/04-output-hygiene/04-REVIEW.md
iteration: 1
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 04: Code Review Fix Report

**Fixed at:** 2026-07-16T19:40:00Z
**Source review:** .planning/phases/04-output-hygiene/04-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 5 (1 critical, 3 warning, 1 info — `fix_scope: all`)
- Fixed: 5
- Skipped: 0

This run is a continuation of a prior fixer session terminated by a session
quota limit after committing CR-01, WR-01, and WR-02. This report covers
all five findings: the three prior commits (re-verified below) plus WR-03
and IN-01 fixed in this session, plus one additional hardening fix
(scanner.Err() check) identified while assessing the phase task list.

## Fixed Issues

### CR-01: HYG-02 stdout guard does not scan any transitively-imported package

**Files modified:** `internal/graphstore/archtest/stdout_confinement_test.go`
**Commit:** `21e47b9`
**Applied fix:** Replaced the flat `guardedPackages` root-only scan with a
transitive closure walk (`closeOverServeReachableImports`) over module-internal
imports (via `packages.NeedDeps` + recursive `pkg.Imports` walk), excluding
`internal/cli` per D-06b. Added regression guards asserting the closure grew
beyond the six roots and that a known transitive dependency
(`internal/schema`) is present in the reachable set, plus a companion
regression test (`TestStdoutGuardCatchesViolationsInTransitiveDependency`)
planting a violation in a dependency of a guarded package.
**Verification (this session):** Re-ran `go test ./internal/graphstore/archtest/... -v` —
`TestNoStdoutNoiseInServeReachablePackages` and
`TestStdoutGuardCatchesViolationsInTransitiveDependency` both pass.

### WR-01: Data race on `stderrBuf` in the MCP stdout-purity test's failure path

**Files modified:** `test/integration/mcp_stdout_purity_test.go`
**Commit:** `c2fe509`
**Applied fix:** Introduced `syncBuffer`, a mutex-guarded `io.Writer` wrapper
around `bytes.Buffer`, replacing the plain unsynchronized `bytes.Buffer` used
for `cmd.Stderr`, so concurrent writes from `os/exec`'s stderr-copy goroutine
and reads from the test's failure branches are race-safe.
**Verification (this session):** Re-ran
`go test ./test/integration/... -race -run TestServeMCPStdoutIsPureJSONRPC -v` — pass.

### WR-02: `TestServeMCPStdoutIsPureJSONRPC` never checks whether `tools/call` succeeded

**Files modified:** `test/integration/mcp_stdout_purity_test.go`
**Commit:** `cbd134d`
**Applied fix:** Extended the decoded frame struct with an `Error
json.RawMessage` field and added an explicit `t.Fatalf` when `frame.ID == 2`
and `len(frame.Error) > 0`, so an allowlist regression that returns an early
JSON-RPC error can no longer masquerade as a successful exercise of the
second store-open path.
**Verification (this session):** Re-ran
`go test ./test/integration/... -run TestServeMCPStdoutIsPureJSONRPC -v` — pass.

### WR-03: `diagWriter` is a shared, unsynchronized global mutated by tests

**Files modified:** `internal/graphstore/logger.go`, `internal/graphstore/logger_test.go`
**Commit:** `9372385`
**Applied fix:** Replaced the bare `var diagWriter io.Writer` with a
`diagWriterMu sync.RWMutex`-guarded pair of accessor functions
(`getDiagWriter`/`setDiagWriter`); `writeDiagLine` now reads through
`getDiagWriter()` and `captureDiagWriter` (in the test file) swaps/restores
through `setDiagWriter()` instead of direct assignment. Production default
(`os.Stderr`) and behavior are unchanged — no env hatch was added (D-05).
This follows the `sync.RWMutex` convention already used for `pebbleStore.mu`
elsewhere in the package, per the finding's guidance to keep the seam
consistent with existing conventions.
**Verification:** `go build ./internal/graphstore/...` clean;
`go test ./internal/graphstore/... -race -run TestOpenInjectsQuietLogger -v`
and the full `TestQuietLogger*` suite pass under `-race`.

### IN-01: Stdout-confinement predicates cannot see indirect stdout writes

**Files modified:** `internal/graphstore/archtest/stdout_confinement_test.go`
**Commit:** `68e7a91`
**Applied fix:** Added a doc-comment addendum to the package doc comment
(directly below the existing predicate-precision paragraph) explicitly
noting that `os.NewFile(1, "").Write(...)`, `syscall.Write(1, ...)`, or an
`os.Stdout` value threaded through an unrelated variable/interface bypass
all three predicates, and that this residual gap is only covered (if at
all) by `mcp_stdout_purity_test.go`'s runtime check. No code/behavior
change — doc only, as the finding specified.
**Verification:** `go build ./internal/graphstore/...` clean (comment-only
change; confirmed via re-read and `go vet`).

## Additional Fix (identified during continuation-run assessment)

### `bufio.Scanner` loop lacked a `scanner.Err()` check after the purity scan

**Files modified:** `test/integration/mcp_stdout_purity_test.go`
**Commit:** `8a732cc`
**Assessment:** Judged real, not just theoretical. The background scanning
goroutine in `TestServeMCPStdoutIsPureJSONRPC` could exit `scanner.Scan()`'s
loop early on a genuine scan error (e.g. `bufio.ErrTooLong` from a runaway
unterminated line exceeding the 10MB buffer, or an underlying pipe read
error) with zero visibility into *why* it stopped — the existing `!ok`
failure branch only reported "stdout closed", silently swallowing the
actual cause and the unverified bytes that triggered it.
**Applied fix:** Captured `scanner.Err()` into a `scanErr` variable written
only immediately before `close(lines)` in the scanning goroutine, and read
only from the main goroutine's `!ok` branch — safe without additional
synchronization because a channel close happens-before a receive that
returns due to that closure (Go memory model). The failure message now
includes `scanner error: %v`.
**Verification:** `go test ./test/integration/... -run TestServeMCPStdoutIsPureJSONRPC -v`
and `-race` variant both pass; `go vet ./test/integration/...` clean.

## Skipped Issues

None — all five REVIEW.md findings (plus the one additional hardening fix
called out in the task instructions) were fixed and verified.

## Full Suite Verification

- `go build ./...` — clean.
- `go test ./internal/graphstore/... ./internal/graphstore/archtest/... ./test/integration/...` — all pass (including `-race`).
- `go test ./...` — all packages pass except `internal/daemon`'s
  `TestDaemonFlushLockRequeueGivesUpPerEpisode`, which is flaky/timing-sensitive
  (injected lock-contention race) and unrelated to any file touched by this
  phase's findings (`internal/graphstore`, `internal/graphstore/archtest`,
  `test/integration`). Confirmed pre-existing and non-deterministic: passed
  in 3/3 isolated re-runs (`-run TestDaemonFlushLockRequeueGivesUpPerEpisode -count=1`)
  and in most full-suite runs; failed once out of several full-suite runs in
  this session. Not a regression introduced by any commit in this report.

---

_Fixed: 2026-07-16T19:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
