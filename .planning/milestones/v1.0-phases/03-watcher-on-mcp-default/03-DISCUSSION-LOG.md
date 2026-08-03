# Phase 3: Watcher-on-MCP Default - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-16
**Phase:** 3-watcher-on-mcp-default
**Mode:** --auto (all areas auto-selected; recommended option chosen per question, no interactive prompts)
**Areas discussed:** Flag surface & precedence (WATCH-01), Handshake-path budget (WATCH-02), Watch-policy port shape (WATCH-03), Concurrent-session convergence (WATCH-04), Subprocess harness architecture (TEST-04)

---

## Flag surface & precedence (WATCH-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Remove `--watch`, add `--no-watch` only | Exact TS surface; breaks v0.1 `--watch` invocations | |
| Keep `--watch` as deprecated no-op | Backward compatible but semantically dead | |
| Repurpose `--watch` as explicit force-on | Flag analogue of `CODEGRAPH_FORCE_WATCH=1`; overrides WSL2 auto-off; `--no-watch` still wins; both together = flag error | ✓ |

**Auto-selected:** Repurpose `--watch` as force-on (recommended)
**Notes:** Precedence mirrors TS watch-policy.js: opt-out (flag/env) → force-on (flag/env) → WSL2 auto-off → default on. TS's `--no-watch`→env-mutation routing is a Commander artifact, deliberately not ported — policy takes explicit inputs, env never mutated.

---

## Handshake-path budget (WATCH-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Move everything async incl. reconcile Sync | TS-like lazy catchUpSync; loses the shipped "first tool reads a current graph" guarantee (D-06/SYNC-03) | |
| Watcher startup fully off-path; reconcile stays synchronous | Zero watcher code before ServeStdio by construction; reconcile is stat-prefiltered cheap and not watcher startup | ✓ |
| Status quo (daemon.New on-path, Run in goroutine) | Mostly off-path already, but not provable by construction | |

**Auto-selected:** Watcher startup fully off-path, reconcile unchanged (recommended)
**Notes:** Verified via extracted seam function + mutation test (WR-01 precedent) and the TEST-04 handshake assertion.

---

## Watch-policy port shape (WATCH-03)

| Option | Description | Selected |
|--------|-------------|----------|
| `internal/watch/policy.go`, exact TS semantics, enforced in daemon.Run | One seam covers serve + standalone daemon, matching TS's watcher-level enforcement | ✓ |
| New `internal/watchpolicy` package | More separation, no consumer benefit | |
| Inline checks in serve.go only | Standalone daemon would double-watch WSL2; diverges from TS | |

**Auto-selected:** `internal/watch/policy.go` + daemon.Run enforcement (recommended)
**Notes:** Verbatim TS strings for the disabled message; one documented divergence — `fs.watch` (Node API name) becomes `file watching` in the WSL2 reason string. `'1'`-strict env checks ported exactly.

---

## Concurrent-session convergence (WATCH-04)

| Option | Description | Selected |
|--------|-------------|----------|
| Defer-once (status quo) | Never two writers, but zero writers after holder exits | |
| Defer-and-retry with jittered cadence | Surviving session takes over the lock when writer exits; goleak-clean teardown | ✓ |
| TS-style pool/proxy | Out of v1.0 scope (DMON-FUT-01) | |

**Auto-selected:** Defer-and-retry (recommended)
**Notes:** Researcher must confirm `acquire()` stale-lock/liveness semantics before finalizing (D-16) — retry must self-heal against a crashed holder's stale lock.

---

## Subprocess harness architecture (TEST-04)

| Option | Description | Selected |
|--------|-------------|----------|
| `test/integration/` normal package + explicit CI step | Included in `go test ./...` AND named in CI; TestMain-built binary; mcp-go client over real stdio | ✓ |
| Build-tagged package | Silently excluded by default — the exact GOLDEN-01 failure mode | |
| Under `testdata/` | go tool ignores it; requires remembering the explicit invocation | |

**Auto-selected:** `test/integration/` + explicit CI step (recommended)
**Notes:** Anchor = CR-01 worktree-notice case with real git worktree fixture + main-checkout control, mutation-proven; plus default-on handshake and `CODEGRAPH_NO_WATCH=1` cases. Zero new dependencies (mcp-go client already in the module).

---

## Claude's Discretion

Goroutine/channel structure in serve.go's watcher block; retry interval constant and jitter; test naming/layout inside `test/integration/`; ordering of policy log vs lock acquisition inside the goroutine.

## Deferred Ideas

- `CODEGRAPH_WATCH_DEBOUNCE_MS` parity (TS #403) — Phase 8 surface reconciliation
- TS liveness watchdog (#850) / PPID watchdog — Phase 7 (DMON-03)
- TS pool/proxy multi-session sharing — DMON-FUT-01 (out of v1.0)
- Release runbook todo — reviewed, not folded (3rd time); belongs in Phase 8
