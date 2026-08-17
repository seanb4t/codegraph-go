# Deferred Items

Out-of-scope discoveries logged during Phase 4 execution. Per the executor
scope boundary, these are recorded here and NOT fixed by this phase.

## 1. `internal/daemon` load-induced flake under full-suite parallel load

- **Status:** acknowledged
- **Found during:** 04-01 Task 1 (NOTICE trim) verify — `CGO_ENABLED=1 go test -count=1 ./...`
- **Symptom:** `--- FAIL: TestRunWatchdogCancelsRunOnSimulatedReparent (250.34s)` — `Run did not return after a simulated reparent`. Deterministic across two full-suite runs (identical 250.34s duration).
- **Root cause:** documented, PRE-EXISTING flake class. The watchdog uses a fixed 1s wall-clock ticker; under full-suite parallel load the ticker's next fire is delayed past the test budget. The test's own comment (internal/daemon/daemon_test.go:345-351) documents this load-induced flake class and points at a sibling (`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`). CI (`.github/workflows/ci.yml:8-11,78-85,123-130`) and Taskfile (`test:daemon`, Taskfile.yml:104-112) both document it as PRE-EXISTING and isolate `internal/daemon` into its own `-count=1` step rather than running it in the parallel suite.
- **Evidence it is not a regression from 04-01:** the 04-01 change is NOTICE-only (markdown, not compiled into any test). `go test -count=1 ./internal/daemon/` in isolation exits 0; the single test passes in isolation (1.384s); every other package in the full suite is `ok`.
- **Why not fixed here:** out of scope — pre-existing failure in an unrelated file (`internal/daemon`), not caused by the current task's changes. The project's documented handling is the `task test:daemon` isolation, which this phase adopts for verification.
- **Action for later:** none required — the project already routes around it in CI. If the flake worsens, re-open against the daemon watchdog test budget.
