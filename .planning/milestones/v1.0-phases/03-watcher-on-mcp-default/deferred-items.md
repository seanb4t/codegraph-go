# Deferred Items — Phase 3 (Watcher-on-MCP Default)

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's changes).

## 03-03: `internal/daemon.TestConvergenceTwoSessions` flakes under full-suite parallel load

> **CLOSED 2026-08-10** by maintainer decision during `/gsd-audit-uat` — **not fixed, deliberately stopped tracking.** `TestConvergenceTwoSessions` still exists (`internal/daemon/soak_test.go`), so this is a waiver, not a stale entry. It never reproduced outside full-parallel-suite contention and passes reliably in isolation. Current signal: `task test` on 2026-08-10 ran `test:daemon` (64.9s) and `test:race` over `./internal/daemon/...` (73.5s), both green — one clean run, which is not proof of non-flakiness. Related sightings of the sibling `TestSoak` flake were closed the same day in `v0.1-phases/05-language-coverage-resolution-breadth/deferred-items.md`. **Reopen with a fresh entry if it returns.**
>
> The six bullets below are one finding; each carries `status: resolved` because the parser treats every top-level bullet as its own entry.

- **Found during:** 03-03 Task 2 verification (`go test ./... -count=1`).
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."

- **Symptom:** `MANIFEST create failed: open .../TestConvergenceTwoSessions.../.codegraph/store/MANIFEST-000025: no such file or directory`,
  causing `TestConvergenceTwoSessions` (added in 03-02) to fail intermittently.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."

- **Scope:** `internal/daemon/` is untouched by 03-03
  (`git diff --exit-code internal/daemon/` is clean on this plan's commits).
  This is 03-02's territory.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."

- **Reproduction:** Fails intermittently only under `go test ./... -count=1`
  (all packages running in parallel, contending for disk I/O). Passes
  reliably in isolation: `go test ./internal/daemon/... -count=1` and
  `go test ./internal/daemon/... -run TestConvergenceTwoSessions -count=3`
  both green (verified 2026-07-16).
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."

- **Likely cause (not investigated further — out of scope for 03-03):** a
  Pebble store `MANIFEST` creation race in the two-Daemon-instances-sharing-
  one-`.codegraph/`-root soak fixture, under temp-dir/disk contention from
  sibling packages running concurrently in the same `go test ./...` invocation.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."

- **Recommendation:** a future 03-0x plan (or a dedicated flake-fix pass)
  should investigate whether `TestConvergenceTwoSessions` needs additional
  synchronization around store (re)opening after the lock-holder session
  exits, or whether this is purely a CI-runner disk-contention artifact.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat 2026-08-10 — not fixed, deliberately stopped tracking. See the closure note under the heading above; this file's six bullets are one finding, and each carries the field because the parser treats every top-level bullet as its own entry."
