# Deferred Items — Phase 3 (Watcher-on-MCP Default)

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's changes).

## 03-03: `internal/daemon.TestConvergenceTwoSessions` flakes under full-suite parallel load

- **Found during:** 03-03 Task 2 verification (`go test ./... -count=1`).
- **Symptom:** `MANIFEST create failed: open .../TestConvergenceTwoSessions.../.codegraph/store/MANIFEST-000025: no such file or directory`,
  causing `TestConvergenceTwoSessions` (added in 03-02) to fail intermittently.
- **Scope:** `internal/daemon/` is untouched by 03-03
  (`git diff --exit-code internal/daemon/` is clean on this plan's commits).
  This is 03-02's territory.
- **Reproduction:** Fails intermittently only under `go test ./... -count=1`
  (all packages running in parallel, contending for disk I/O). Passes
  reliably in isolation: `go test ./internal/daemon/... -count=1` and
  `go test ./internal/daemon/... -run TestConvergenceTwoSessions -count=3`
  both green (verified 2026-07-16).
- **Likely cause (not investigated further — out of scope for 03-03):** a
  Pebble store `MANIFEST` creation race in the two-Daemon-instances-sharing-
  one-`.codegraph/`-root soak fixture, under temp-dir/disk contention from
  sibling packages running concurrently in the same `go test ./...` invocation.
- **Recommendation:** a future 03-0x plan (or a dedicated flake-fix pass)
  should investigate whether `TestConvergenceTwoSessions` needs additional
  synchronization around store (re)opening after the lock-holder session
  exits, or whether this is purely a CI-runner disk-contention artifact.
