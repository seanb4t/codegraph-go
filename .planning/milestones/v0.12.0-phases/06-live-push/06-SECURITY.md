---
phase: 06
slug: live-push
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
created: 2026-09-07
---

# Phase 6 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all seven plans carry a
`<threat_model>` block. Independently re-parsed during the audit (not taken from the
orchestrator's summary): **47 distinct threat IDs, zero duplicates** (`T-06-01`…`T-06-46`
plus `T-06-SC`). Severity counted directly from the seven tables: **2 critical, 25 high,
15 medium, 5 low** — matching the orchestrator's inventory exactly. Disposition:
**44 mitigate, 3 accept**.

Verification depth is ASVS L1 (grep-level mitigation presence, with several high/critical
threats additionally confirmed by running their cited tests fresh with `-count=1`, never
cached). Every zero-count check is paired with a positive control per rule `84d1gfpywd`,
and every threat's evidence was checked against the working tree at HEAD (`ca7583b7`),
not accepted from plan or SUMMARY text.

**Blocking threshold:** `security_block_on: high`. All 25 high-severity threats and both
critical threats are `mitigate`; **zero high-or-critical threats were accepted**.
HIGH∩ACCEPT is empty — independently confirmed by cross-referencing severity and
disposition across all seven registers. All 3 accepts are low or medium.

**Unregistered surface:** zero occurrences of "threat" (case-insensitive) across all seven
SUMMARY files, paired with a positive control — `Deviations` matched in all seven, proving
the search ran. Genuine absence, not a silent-zero miss.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → loopback Connect listener (`WatchGraph`, 14th rpc, first streaming method) | Untrusted request path joining an existing guarded surface | `since_generation` (deliberately unread server-side) |
| filesystem (`.codegraph/store`) → UI process | Untrusted, high-volume event source; the daemon or a manual index is the writer | fsnotify events, `Meta` comparisons |
| UI process → shared Pebble store | Cross-process exclusive lock; the UI must never deny it to a writer nor hold it across a stream | open-read-close per check, never a retained handle |
| publisher → N subscribers | Per-subscriber state held for a client that may vanish or stall | the six `WatchGraphEvent` fields |
| `http.Server` timeouts → handler | A per-request deadline change is a deliberate, path-scoped relaxation of a documented safety property | none — timing behaviour only |
| server graph payload → browser renderer | Re-fetched data drives an element swap on a live canvas | node/edge counts, positions, cycle ids |
| browser tabs → test-only loopback proxy → `codegraph ui` | A harness process sits on the path the origin/host guard protects | Host/Origin headers, rewritten to upstream's authority |
| three concurrent processes (daemon, `serve --mcp`, ui) → one Pebble store | Cross-process exclusive lock; the failure mode is denial, not slowdown | store lock, `Meta` reads |
| committed bundle → shipped binary | The embedded app is served from committed build output | compiled JS/CSS/HTML |
| dependency manifests → build | Supply-chain surface for the phase | `go.mod` / `go.sum` / `package.json` / `pnpm-lock.yaml`, all unchanged |

---

## Threat Register

All **47 threats CLOSED**. Every `mitigate` disposition was verified against the working
tree — code read directly, or the cited test re-run fresh — never accepted from prose.

### Critical severity — 2, both `mitigate`, both verified

| Threat ID | Component | Mitigation | Verified | Status |
|-----------|-----------|------------|----------|--------|
| T-06-05 | store handle held across a stream lifetime | `computeChange` opens via `openEngine`, `defer closer.Close()`, one open per check | `livepublish.go` imports neither `net/http` nor `connectrpc.com/connect` — structurally cannot hold a stream-scoped handle. `WatchGraph` calls only `Subscribe` / `unsubscribe`. `TestLiveChangeOpenCloseBalance` PASS fresh | closed |
| T-06-32 | UI live push starving a concurrent daemon sync | real three-process gate, zero-starvation against a measured baseline, demonstrated RED | `corpora/live-push-concurrency-check.json` — `flushesCompleted:5, flushesStarved:0, maxFlushDurationMs:309` against a bound of 1783. RED demonstrated and reverted, with the daemon's own `store lock held` log lines captured | closed |

### High severity — the blocking set, 25, all `mitigate`, all verified

| Threat ID | Mitigation | Verified |
|---|---|---|
| T-06-01 | the 14th method stays read-only | `wantUIServiceMethods` carries `"WatchGraph"`; the guard logs "14 method names against 19 mutating verbs"; PASS fresh |
| T-06-02 | proto field numbering pinned per D-02a | 7 entries pinned with a compiler-checked length; `TestUIProtoFieldNumbersAreStableAndUnique` PASS, "42 messages, 169 fields" |
| T-06-07, T-06-08 | bounded, coalescing, never-blocking send | `make(chan …, 1)`; `trySend` uses `select` / `default` exclusively — never a bare blocking send |
| T-06-09, T-06-20 | subscriber released on disconnect | `unsubscribe()` deletes and closes; `defer unsubscribe()` in the handler; goleak tests PASS |
| T-06-10, T-06-35 | server-initiated sends cannot corrupt client-initiated state | `sendCount` and `subs` structurally separate; `TestLiveRegistryCounterSeparation` PASS. Zero `type .*pending.*struct` hits in `internal/uiserver`, positive-controlled against `internal/mcp/server.go:466` |
| T-06-12 | a store-lock error never escalates to a crash | classification by exported sentinel `graphstore.ErrStoreLocked`, never message text |
| T-06-18 | origin guard unchanged | `originguard.go` has **zero** diff lines across the entire phase |
| T-06-19, T-06-21, T-06-22 | deadline cleared for one path; writer unwrapped; guard outermost | matches only `UIServiceWatchGraphProcedure`; `next.ServeHTTP(w, r)` with the original `w`; `originHostGuard(port, clearWatchDeadline(mux))` verbatim at `server.go:176` |
| T-06-23 | the streaming method is not a write vector | handler opens no store — no `openEngine` / `withEngine` / `query.OpenAt` anywhere in `livehandler.go` |
| T-06-27 | no full layout run per live event at corpus scale | `nodeSetUnchanged` fast path merges data with zero layout calls |
| T-06-28 | no graph payload on the event | `WatchGraphEvent` has exactly 6 scalar fields |
| T-06-33, T-06-34 | no reconnect storm; a stalled tab cannot degrade others | `reconnectAttemptsPerTab [3,3,3]`, delays strictly increasing and pairwise distinct; `blockedTabReceiptsDuringBlock:0` with `healthyTabsIsolationSupersetOK:true` |
| T-06-36 | no partially staged bundle | porcelain empty **after** commit; `git ls-files --error-unmatch` succeeds; `web:drift` PASS at 110 source / 32 output |
| T-06-39, T-06-41 | publisher arms and re-arms; a restart cannot deafen tabs | `armWatches` state machine over `[repoRoot, .codegraph, store]`; epoch-scoped admission always admits a new epoch |
| T-06-46 | a live session cannot slow a concurrent sync | baseline measured before the UI starts, bounded at 3× + 1000 ms |
| T-06-SC | zero new packages | `git diff --stat` on all four manifests genuinely empty, positive-controlled against a file that did change |

### Medium and low severity — 20, closed

Verified individually. Highlights: `proto:drift` byte-identical across 4 generated files
(T-06-03); `schema.IsCommitSHA` applied at the live-push exit (T-06-11); generation
increments serialized under lock (T-06-15); the `Meta` comparison runs strictly **before**
`engineStatus`, so an unchanged store performs no filesystem scan (T-06-40); a clean EOF
is treated identically to a thrown failure for backoff purposes (T-06-42); the publisher
is stopped **before** `Shutdown` on the normal path (T-06-43); zero CI-skip directives in
any commit message across the phase (T-06-38); the test-only proxy rewrites both `host`
and `origin` to the upstream's own authority and never enters the shipped binary (T-06-45).

---

## Accepted Risks Log

No accepted risk sits at or above the `high` blocking threshold.

| Risk ID | Threat Ref | Severity | Rationale | Accepted By | Date |
|---------|------------|----------|-----------|-------------|------|
| R-06-01 | T-06-04, T-06-17 | low | `commit_sha` and the other `WatchGraphEvent` fields are **already** exposed by `GetStatusResponse` / `GetHealthResponse` over the identical loopback-only, origin-guarded surface. The streaming rpc discloses nothing the browser cannot already read. | plans 06-01, 06-03 | 2026-09-07 |
| R-06-02 | T-06-25 | medium | No global subscriber cap. The listener is loopback-only, same-origin-guarded and single-user by design; per-subscriber memory is O(1) and every subscriber is released on disconnect. A cap would add a failure mode without closing a reachable vector on this surface. | plan 06-04 | 2026-09-07 |

---

## Notes

**Note 1 — criterion 5's gate proved itself, not just the code.** The real three-process
gate was demonstrated RED by temporarily removing one `defer closer.Close()`, producing
`flushesStarved 3/3` alongside the daemon's own `graphstore: store lock held` and
`sync lost the store-lock race 6 consecutive times; giving up` log lines — then restored to
a genuine PASS against unmodified bars. A gate that can be made to fail on command, for the
stated reason, is worth more than one that has only ever passed.

**Note 2 — the code-review findings are correctness defects, not attack surface.** CR-01
(a stale mount response overwriting a newer live-triggered one on `/health`) and WR-01 (a
watcher/goroutine leak on `Serve`'s abnormal-exit branch) neither disclose data across a
trust boundary, bypass a confinement check, nor weaken the origin/host guard. Both are fixed
and were independently re-verified by **reverting each fix hunk, re-running the exact named
test, confirming a symptom-matching RED, then restoring** and confirming a clean `git diff`.
WR-03 / WR-04 / IN-01 / IN-02, and the re-review's WR-2-01 / IN-2-01, are documentation and
dead-code items with zero functional impact.

**Note 3 — WINDOWS 26 / 28 / 29 carried forward per `06-CONTEXT.md`'s maintainer ruling.**
26 and 28 are rendering-robustness observations with no security implication. **29 is a real
gate blind spot** — `web:drift`'s output half enumerates the filesystem via `find`
(`Taskfile.yml:60`) while its source half uses `git ls-files` (`:42`), so it cannot by itself
detect an incompletely-staged bundle. This phase's own 06-07 plan independently reproduced
that exact failure shape and closed it **operationally** via the assert-after-commit pattern
(T-06-36), rather than touching the gate, which remains out of phase scope. 30 plan-id leaks
remain in test and harness files as an accepted deprioritization; production source and
generated code are clean.

**Note 4 — no count discrepancy.** Severity and disposition counts matched the orchestrator's
inventory exactly on independent re-extraction, unlike Phase 5's audit which found and
corrected a one-count misplacement.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-07 | 47 | 47 | 0 | gsd-secure-phase (ASVS L1; every `mitigate` threat verified against working-tree code and tests at HEAD `ca7583b7`, diff base `033ee9cb`) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in the Accepted Risks Log
- [x] No accepted risk at or above the `high` blocking threshold
- [x] `threats_open: 0` confirmed
- [x] Every zero-count verification paired with a positive control
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-07
