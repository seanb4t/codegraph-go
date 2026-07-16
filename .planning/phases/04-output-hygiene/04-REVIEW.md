---
phase: 04-output-hygiene
reviewed: 2026-07-16T23:15:00Z
depth: deep
files_reviewed: 7
files_reviewed_list:
  - internal/graphstore/logger.go
  - internal/graphstore/logger_test.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/archtest/stdout_confinement_test.go
  - internal/graphstore/archtest/stdout_detection_selftest_test.go
  - test/integration/mcp_stdout_purity_test.go
  - test/integration/sync_noise_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 04: Code Review Report

**Reviewed:** 2026-07-16T23:15:00Z
**Depth:** deep
**Files Reviewed:** 7
**Status:** clean

## Summary

This is the post-fix re-review of iteration 1's 1 Critical + 3 Warnings + 1
Info. All five findings were re-verified against the current tree, and all
five are genuinely closed at the root cause — not just cosmetically
addressed. No new Critical or Warning defects were introduced by the six fix
commits. One pre-existing Info-level residual risk (structurally identical
in spirit to the already-accepted IN-01) is newly worth recording.

**CR-01 (stdout guard closure gap) — verified closed by mutation test.**
`closeOverServeReachableImports` (`stdout_confinement_test.go`) now loads
with `packages.NeedDeps` and recursively walks `pkg.Imports`, filtered to
module-internal paths and excluding `internal/cli`. I did not just read the
diff — I reverted `packages.NeedDeps` out of the `Mode` bitmask by hand,
re-ran `TestStdoutGuardCatchesViolationsInTransitiveDependency`, and
confirmed it fails exactly as iteration 1's finding predicted
(`"...but the closure scan did not flag it — got violations: []"`), then
restored the file and confirmed it passes green again. The new
`stdout_closure_selftest_test.go` (added by this fix, plants a real
`fmt.Println` in `internal/schema` via `packages.Config.Overlay`, no file on
disk touched) is a genuine, working regression lock on this exact gap —
not a file in the reviewed `files:` scope for this iteration, but directly
load-bearing for CR-01's closure and inspected for that reason.

**WR-01 (stderrBuf data race) — verified closed under `-race`.** The plain
`bytes.Buffer` was replaced with a mutex-guarded `syncBuffer`. Ran
`go test -race -count=5 ./test/integration/... -run TestServeMCPStdoutIsPureJSONRPC`
— clean on every iteration, no race reported on the buffer.

**WR-02 (tools/call error-blindness) — verified by logic trace.** The frame
struct now decodes `Error json.RawMessage` and the `case 2:` branch fails
loudly (`t.Fatalf`) before setting `sawToolResponse` if `len(frame.Error) >
0`. This closes the exact silent-success-on-allowlist-regression path
iteration 1 flagged.

**WR-03 (diagWriter unsynchronized global) — verified closed for the flagged
race class.** All reads now go through `getDiagWriter()` (RLock) and all
test-side writes through `setDiagWriter()` (Lock), replacing the bare var
reassignment. I wrote a throwaway probe test hammering `setDiagWriter`/
`getDiagWriter` from 100 concurrent goroutines under `-race` — clean. This
closes the specific race iteration 1 described (bare var reassignment
racing a concurrent read), which is the race that actually existed. See the
Info item below for a narrower, pre-existing residual this fix does not
(and was never claimed to) close.

**IN-01 (residual risk of indirect stdout writes) — verified: doc addendum
landed** in `stdout_confinement_test.go`'s package comment, matching what
iteration 1 asked for.

**The `setDiagWriter` "unused" signal is a false lead, not a live defect.**
`setDiagWriter` (`logger.go:90`) is called from `logger_test.go:22,23`
(`captureDiagWriter`), in the same package, and is the only production
write-path for `diagWriter` besides the `os.Stderr` default — i.e. the
accessor pair is wired exactly as WR-03's fix intended: production code
reads via `getDiagWriter()`, tests install via `setDiagWriter()`. Confirmed
via `staticcheck ./internal/graphstore/...` (zero findings) and
`go vet ./...` (zero findings) — neither flags it. The "unused" report
traces to a whole-program-reachability-style analysis (gopls's
unusedfunc-class check, or an equivalent tool run without test-file
consideration) that doesn't count a symbol reachable only from `_test.go`
files in its own package as "used." This is a known false-positive class
for that style of analysis, not a bypassed seam or dead helper — the
accessor pair is genuinely load-bearing.

## Info

### IN-02: `diagWriterMu` only guards the `diagWriter` variable, not concurrent `Write` calls on the writer it points to

**File:** `internal/graphstore/logger.go:56-58, 80-96`
**Issue:** `writeDiagLine` calls `getDiagWriter()` under `RLock`, but releases the lock *before* calling `fmt.Fprintf` on the returned writer:
```go
func writeDiagLine(prefix, format string, args ...any) {
	fmt.Fprintf(getDiagWriter(), prefix+format+"\n", args...)
}
```
This correctly fixes the race WR-03 described (a bare var reassignment racing a concurrent read of that var — confirmed via a 100-goroutine concurrent `setDiagWriter`/`getDiagWriter` probe under `-race`, clean). It does not, and was never claimed to, protect two concurrent `quietLogger.Errorf`/`Fatalf` calls from racing on the *same* returned writer's internal state. Today this is inert: the production default (`os.Stderr`) is safe for concurrent `Write` calls, and the doc comment correctly states no test in this package currently uses `t.Parallel()` or drives concurrent `Errorf` calls against a captured `*bytes.Buffer` (which is *not* safe for concurrent `Write`). This mirrors IN-01's framing exactly — a documented, currently-inert residual, not a live bug — but is worth recording explicitly now that the mutex exists, so a future reader doesn't mistake `diagWriterMu` for a guarantee it doesn't provide if a long-lived shared `pebbleStore` (already anticipated in `pebble_store.go`'s own `mu` doc comment) ever drives concurrent `Errorf` calls against a test capture buffer.
**Fix:** No code change required for the current risk level. If/when a test ever needs concurrent `Errorf` calls against a captured buffer (e.g. testing the anticipated long-lived shared store), either capture into a `syncBuffer`-style mutex-guarded writer (the same pattern `mcp_stdout_purity_test.go`'s `WR-01` fix already established) or extend the doc comment on `diagWriterMu` to state explicitly that it does not extend to the writer's own internal state.

---

_Reviewed: 2026-07-16T23:15:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
