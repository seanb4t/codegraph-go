---
phase: 7
slug: interactive-tui-daemon-picker-install-multi-select
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-19
---

# Phase 7 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> State B (register authored at plan time, all 8 PLAN.md carry `<threat_model>`).
> ASVS L1, block_on: high. threats_open: 0 → L1 short-circuit (grep-depth
> verification of every high-severity mitigation against the implementation;
> the auditor spawn is reserved for ASVS ≥ 2 or an unresolved register).

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| terminal → process | stdin/stdout fds decide whether a blocking interactive read (`tea.NewProgram`) is issued | tty-ness (fd char-device state) |
| filesystem → signal delivery | On-disk `~/.codegraph/daemons/<pid>.json` records are parsed and their pids may be signaled | pid + StartedAt (untrusted on-disk) |
| concurrent daemons → registry dir | Multiple daemons write/prune one directory with no global lock | atomic per-daemon record files |
| OS/kernel → process | Parent-liveness is a pure OS query (`getppid`/`OpenProcess`), no untrusted input | ppid / process handle |
| supply chain → build | Two new third-party modules (`bubbletea/v2`, `bubbles/v2`) enter the dependency closure | pinned pure-Go modules |
| user intent → cross-process stop | Picker/`stop` command triggers a real OS signal to a registry pid | SIGTERM (POSIX) / TerminateProcess (Windows) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01-01 | Tampering | new charm deps (bubbletea/bubbles) | medium | mitigate | Same charmbracelet org as Phase-6 lipgloss/v2; pinned exact versions (v2.0.8/v2.1.1); pure-Go, no new CGo; full REL-01 closure audit → Phase 8 | closed |
| T-07-01-02 | Denial of Service | interactive gate | high | mitigate | `InteractiveAllowed` (tty.go:49) requires BOTH fds be terminals before any `tea.NewProgram`; piped/CI can never reach a blocking stdin read (TUI-04) | closed |
| T-07-01-03 | Elevation of Privilege | archtest guard | low | mitigate | Guarded engine set stays charm-free by construction; `TestNoCharmInServeReachablePackages` fails the build on any leak — verified green | closed |
| T-07-02-01 | Tampering | registry record file | high | mitigate | A record is never trusted for signaling alone — `List()` (registry.go:107) prunes any pid failing lock.go `isStale` (live AND /proc start-time corroborates `rec.StartedAt`); a forged/stale/reused pid is pruned, not acted on | closed |
| T-07-02-02 | Input Validation | malformed record JSON | medium | mitigate | `json.Unmarshal` into a fixed `Record`; a decode failure is skipped as "unreadable", never evaluated as code | closed |
| T-07-02-03 | Denial of Service | unbounded registry dir | low | accept | Written only by `daemon start` under the same user account — not externally reachable or multi-tenant (AR-07-01) | closed |
| T-07-02-04 | Access Control | `~/.codegraph/daemons/` perms | low | mitigate | Per-user home dir, standard `fsatomic` modes — same trust boundary as the existing `.codegraph/daemon.lock` | closed |
| T-07-03-01 | Denial of Service | watchdog goroutine | high | mitigate | Joinable `stop()` blocks until the goroutine returns on every teardown path; goleak-gated `TestMain` proves no leak (daemon.go:262, `-race` green) | closed |
| T-07-03-02 | Tampering | reparent false-positive | low | accept | A spurious cancel only tears the daemon down cleanly (lock released, registry pruned); captured-baseline predicate minimizes false positives vs bare `ppid==1` (AR-07-02) | closed |
| T-07-03-03 | Repudiation | Windows liveness gap | medium | mitigate | Windows half documented + CI compile-checked (mingw-w64 cross-vet); runtime behavior is a manual-verify item (07-UAT test 3), never a silent no-op | closed |
| T-07-04-01 | Tampering / EoP | sendStop target pid from a forged/stale/reused-pid record | high | mitigate | Re-corroborate `isStale` immediately before signaling (stop.go:69); a mismatch is pruned/skipped, never signaled. OS signal-permission model additionally bounds targets to the same user's processes | closed |
| T-07-04-02 | Repudiation | Windows hard-kill vs POSIX graceful | medium | accept | Documented divergence (A4, Open Q#1); registry self-heals the ungraceful exit on the next scan (AR-07-03) | closed |
| T-07-04-03 | Denial of Service | StopAll signaling many pids | low | accept | Bounded by the registry's own record count (one per local `daemon start`), same-user surface only (AR-07-04) | closed |
| T-07-05-01 | Denial of Service | watchdog goroutine in Run | high | mitigate | `stop()` joined on every teardown path (daemon.go:262/335); goleak-gated `TestMain` proves no leak | closed |
| T-07-05-02 | Tampering | orphaned record after crash | low | mitigate | Best-effort `Deregister` on shutdown; a crash-orphaned record is pruned by the next `List()` self-heal (D-05) and never trusted for signaling (07-04 re-corroborates) | closed |
| T-07-05-03 | Spoofing | implicit daemon spawn | low | mitigate | No implicit-start path added; `Run` stays caller-invoked-only (D-03 no auto-spawn) | closed |
| T-07-06-01 | Denial of Service | install/uninstall picker on non-TTY / CI | high | mitigate | `InteractiveAllowed` gates before `tea.NewProgram` (install.go:87, uninstall.go:52); `-y` short-circuits first; off-TTY resolves to auto (D-13); piped never-hang enforced by 07-08 | closed |
| T-07-06-02 | Tampering (terminal state) | agent DisplayName render | low | accept | DisplayNames come from the hardcoded `internal/agents` registry, not attacker input; bubbles escapes its own rendering (AR-07-05) | closed |
| T-07-06-03 | Elevation of Privilege | charm leak into internal/agents | low | mitigate | Picker confined to `internal/cli/tui`; archtest keeps `internal/agents` charm-free — verified green | closed |
| T-07-07-01 | Denial of Service | bare daemon on non-TTY / piped | high | mitigate | `InteractiveAllowed` gates before `tea.NewProgram` (daemon.go:82, `&& len(records) > 0`); non-TTY takes the plain-list-exit-0 path (D-12); enforced by 07-08 | closed |
| T-07-07-02 | Tampering / EoP | stop targeting a forged/stale pid | high | mitigate | stop-one/stop-all route through `daemon.StopMatching`/`StopAll` (07-04), which re-corroborate each pid via `isStale` before signaling — a poisoned record is never signaled | closed |
| T-07-07-03 | Spoofing | implicit daemon spawn | low | mitigate | Bare `daemon` and `stop` never call `daemon.Run`; only `daemon start` does (daemon.go:146, D-03) | closed |
| T-07-08-01 | Denial of Service | interactive hang on piped stdio | high | mitigate | The bounded-timeout goroutine+select turns any hang into a test FAILURE, catching a mis-gated `tea.NewProgram` before release (TUI-04, `TestPipedNeverHang`) | closed |
| T-07-08-02 | Tampering | install mutating real HOME during test | low | mitigate | Test runs install under a temp HOME / `--target none` so the developer's real agent configs are untouched | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**9 high-severity threats, all `mitigate`, all grep-verified present in the implementation. 0 open at or above the `high` block threshold.**

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-07-01 | T-07-02-03 | Registry dir is same-user, single-writer per `daemon start` — not externally reachable or multi-tenant; unbounded growth self-heals via `List()` pruning | secure-phase (Sean) | 2026-07-19 |
| AR-07-02 | T-07-03-02 | A false-positive reparent only cleanly tears down a daemon (lock released, registry pruned); self-healing tolerates it | secure-phase (Sean) | 2026-07-19 |
| AR-07-03 | T-07-04-02 | Windows `daemon stop` hard-kills (`TerminateProcess`) — no POSIX-SIGTERM equivalent for an arbitrary external process; registry self-heals the ungraceful exit. Softer Windows control-channel deferred (Open Q#1) | secure-phase (Sean) | 2026-07-19 |
| AR-07-04 | T-07-04-03 | `StopAll` is bounded by the local registry's record count (one per `daemon start`), same-user surface only | secure-phase (Sean) | 2026-07-19 |
| AR-07-05 | T-07-06-02 | Agent DisplayNames are hardcoded in the `internal/agents` registry, not attacker-controlled; bubbles escapes its own render | secure-phase (Sean) | 2026-07-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-19 | 24 | 24 | 0 | secure-phase (L1 short-circuit, grep-verified; auditor not spawned) |

**Cross-checks reinforcing this audit:** the 9 high-severity mitigations were independently confirmed by (1) gsd-verifier's goal-backward pass (7/7 must-haves against real code), (2) the deep `/gsd-code-review` (0 Critical — it specifically traced the `daemon.Run` teardown, the `isStale` corroboration before signaling, and the charm-isolation archtest), and (3) human UAT, whose two rendering-bug fixes (G-07-1, G-07-2) further hardened the InteractiveAllowed/never-hang boundary (T-07-01-02, T-07-06-01, T-07-07-01, T-07-08-01).

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-19
