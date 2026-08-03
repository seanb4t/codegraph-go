---
phase: 3
slug: watcher-on-mcp-default
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-16
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| env → policy | `CODEGRAPH_NO_WATCH` / `CODEGRAPH_FORCE_WATCH` cross from the untrusted process environment into the watch decision | behavior toggle (low sensitivity) |
| filesystem → policy | `/proc/version` contents cross into `DetectWSL` | platform detection string |
| concurrent processes → lockfile | Multiple serve/daemon sessions contend for `.codegraph/daemon.lock` in a shared repo dir | lock ownership (PID + start time) |
| user (argv) → serve RunE | `--no-watch` / `--watch` flags select watch behavior | behavior toggle |
| test harness → spawned binary | argv/env/cwd the harness passes to the real codegraph subprocess | test-authored fixture paths + env |
| subprocess stderr → harness | the disabled-message diagnostic the harness reads back | diagnostic text |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01 | Spoofing/Tampering | env toggle parsing (`WatchDisabledReason`) | low | mitigate | Strict `== "1"` equality (D-10, `internal/watch/policy.go`, 3 sites); truthy env noise cannot flip watch state; strict-env negative cases in `policy_test.go` | closed |
| T-03-02 | Denial of Service | `DetectWSL` `/proc/version` read | low | mitigate | Read failure → `false` (never panics/blocks); result cached (`sync.Once`) so a hostile/slow `/proc` is read at most once per process | closed |
| T-03-03 | Denial of Service | `RunWithRetry` vs contended lockfile | low | mitigate | Jittered backoff + `ctx.Done()` select (no tight spin); reuses the existing atomic Link-based lock path (`internal/daemon/daemon.go`) | closed |
| T-03-04 | Denial of Service | crashed holder's stale lockfile wedging survivors | medium | mitigate | `acquire()`'s `isStale` (PID liveness + start-time corroboration) self-heals on every retry iteration (D-16, `internal/daemon/lock.go`) | closed |
| T-03-05 | Tampering | policy bypass leaving watcher un-gated on the daemon path | low | mitigate | Single enforcement point in `Run` before `acquire` (D-11, daemon.go:214); `cli/daemon.go` inherits it; `TestRunPolicyDisabled` asserts | closed |
| T-03-06 | Tampering | contradictory flags (`--no-watch --watch`) silent pick | low | mitigate | cobra `MarkFlagsMutuallyExclusive` rejects before RunE (D-04, `internal/cli/serve.go`) — deterministic flag error | closed |
| T-03-07 | Denial of Service | watcher startup blocking the MCP handshake (WSL2 recursive walk on-path) | medium | mitigate | D-06: `daemon.New`/policy/`acquire`/`watch.Open` all deferred into `serveWatchStart`'s goroutine; mutation-proof `TestServeWatchStartDeferred` | closed |
| T-03-08 | Information disclosure | disabled-reason message on a model-visible channel | low | mitigate | Message goes to stderr only (D-12); full byte sequence test-pinned incl. banner; MCP payloads unaffected | closed |
| T-03-09 | Tampering | subprocess argv/env injection in the harness | low | mitigate | Hermetic `t.TempDir()` fixtures, `exec.Command` (no shell interpolation — verified none in `test/integration/`) | closed |
| T-03-10 | Denial of Service | hung `serve --mcp` handshake blocking CI indefinitely | medium | mitigate | Bounded `context.WithTimeout` on Initialize/CallTool; git-absence `t.Skip`; TestMain build failure aborts fast | closed |
| T-03-11 | Repudiation/Reachability | harness silently not running (GOLDEN-01 recurrence) | high | mitigate | Normal Go package reached by `go list ./...` PLUS explicit named CI step (`ci.yml`, 3 references incl. IN-08 hardened `go list`) | closed |
| T-03-12 | Tampering | child env injection (`CODEGRAPH_NO_WATCH`) in the harness | low | mitigate | Env test-authored, passed via `exec.Command`'s env param (no shell); the toggle is exactly the strict-`== "1"` surface WATCH-03 gates | closed |
| T-03-13 | Denial of Service | unbounded stderr read / hung handshake in the NO_WATCH case | medium | mitigate | Bounded context + grace-window stderr read (`watch_default_test.go`); Initialize under `context.WithTimeout` | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**Post-plan hardening note:** the three code-review fix rounds (03-REVIEW*.md) strengthened, not weakened, this register after plan authoring: lock-error classification moved behind the `graphstore.ErrStoreLocked` sentinel with build-tagged unix/windows classifiers (removing an EACCES-masquerade misclassification), `ErrWatcherClosed` eliminates a zombie-lock-holder path (reinforcing T-03-04's availability posture), and the requeue/`Debouncer.Add` ctx gates preserve the no-Sync-after-lock-release invariant.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-03-SC | T-03-SC (all plans) | Supply-chain gate not applicable: zero new dependencies this phase — mcp-go client and cobra `MarkFlagsMutuallyExclusive` are existing pinned deps; no install task exists to gate | plan-time disposition (all 5 plans) | 2026-07-16 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-16 | 13 (+1 accepted supply-chain) | 13 | 0 | /gsd-secure-phase L1 grep-verification (short-circuit: plan-time register, ASVS L1, all mitigations evidenced) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-16
