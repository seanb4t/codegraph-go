---
phase: 06-live-push
verified: 2026-09-07T21:44:29Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - ".planning/REQUIREMENTS.md"
  - ".planning/phases/06-live-push/06-01-PLAN.md"
  - ".planning/phases/06-live-push/06-01-SUMMARY.md"
  - ".planning/phases/06-live-push/06-02-PLAN.md"
  - ".planning/phases/06-live-push/06-02-SUMMARY.md"
  - ".planning/phases/06-live-push/06-03-PLAN.md"
  - ".planning/phases/06-live-push/06-03-SUMMARY.md"
  - ".planning/phases/06-live-push/06-04-PLAN.md"
  - ".planning/phases/06-live-push/06-04-SUMMARY.md"
  - ".planning/phases/06-live-push/06-05-PLAN.md"
  - ".planning/phases/06-live-push/06-05-SUMMARY.md"
  - ".planning/phases/06-live-push/06-06-PLAN.md"
  - ".planning/phases/06-live-push/06-06-SUMMARY.md"
  - ".planning/phases/06-live-push/06-07-PLAN.md"
  - ".planning/phases/06-live-push/06-07-SUMMARY.md"
  - "corpora/graph-live-update-check.json"
  - "corpora/live-push-concurrency-check.json"
  - "corpora/live-push-multitab-check.json"
  - "internal/uiproto/uiv1/ui.proto"
  - "internal/uiserver/livehandler.go"
  - "internal/uiserver/livepublish.go"
  - "internal/uiserver/server.go"
  - "internal/uiserver/watchtimeout.go"
  - "scripts/live-push-concurrency-check.sh"
  - "scripts/live-push-probe.go"
  - "web/scripts/graph-live-update-check.mjs"
  - "web/scripts/live-push-multitab-check.mjs"
  - "web/src/lib/components/graph/GraphCanvas.svelte"
  - "web/src/lib/live/live-client.ts"
  - "web/src/lib/live/live-store.ts"
  - "web/src/lib/status.ts"
  - "web/src/routes/+layout.svelte"
  - "web/src/routes/browse/+page.svelte"
  - "web/src/routes/graph/+page.svelte"
  - "web/src/routes/health/+page.svelte"
covered_digest: "v1:sha256:5b14cca469558517413c72b928aac2d96e9bba384170ac9c0c0c4cf95c033f41"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 6: Live Push Verification Report

**Phase Goal:** Open views stop going quietly stale — a watcher re-index reaches the browser over the same schema and the same client as every other call, and updates what is on screen in place.
**Verified:** 2026-09-07T21:44:29Z
**Status:** passed
**Re-verification:** No — initial verification
**Diff range:** `033ee9cb..b4064168`

## Verification stance

No claim from `06-0N-SUMMARY.md`, `06-VALIDATION.md`, `06-REVIEW.md`, `06-REVIEW-2.md` or
`06-SECURITY.md` was accepted as evidence. Every criterion below resolves to source read
directly, a named test confirmed to *exist* (`rg 'func Test…'` / `it(`) and then run with
`-count=1`, a corpora record whose pass formula was **recomputed from the raw fields** rather
than read off its own `success` flag, or — for criterion 5 — the gate **re-executed by this
verifier** against a fresh set of real OS processes.

## Goal Achievement

### Observable Truths (the five ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Edit → re-index → open view updates in place, **including** the staleness/health chrome | ✓ VERIFIED | Chrome half: `web/src/routes/+layout.svelte:41-46` applies **every admitted live event straight into the status gate** (`statusGate.applyLiveEvent({...live.event, epoch})`) with no round trip; `web/src/lib/status.ts:63-66` defines `StatusLikeFields` as a `Pick<GetStatusResponse, …>` that `WatchGraphEvent` structurally satisfies, so `classifyStatus` (`status.ts:75`) consumes the event directly. Data half: `health/+page.svelte:81,112-114`, `browse/+page.svelte:140`, `graph/+page.svelte:243,300-302`, `AnalysisPanel.svelte:152,182-184` each subscribe to the same `liveStore` context and re-issue **their own** rpc. Behavioral: `web/tests/live-store.test.ts:116` *"StatusGate.applyLiveEvent: no round trip"* and `web/tests/live-route-refetch.test.ts:187,287,319` — all green in the full `pnpm test` run I executed (464/464). |
| 2 | Per-message streaming over plain HTTP/1.1, same Protobuf schema, same Connect client, no separate transport | ✓ VERIFIED | `internal/uiserver/livehandler_test.go:556 TestWatchGraphStreamDeliversMessageByMessage` **run by me** with `-count=1`: PASS in 0.81s, `generations=[2 3 4]`, gaps ≥ 150ms. Its structure is the assertion — it blocks on `stream.Receive()` *before* issuing the next real store write, so a buffered implementation stalls to the 5s deadline (and did, per the recorded RED). Same schema/client: `rpc WatchGraph(WatchGraphRequest) returns (stream WatchGraphEvent);` is the 14th rpc **inside the existing `UIService`** (`internal/uiproto/uiv1/ui.proto:121`); the browser calls it through the one shared client — `live-store.ts:33` `uiClient.watchGraph(...)`, `uiClient` = `createClient(UIService, createConnectTransport(...))` at `web/src/lib/client.ts:23-32`. No new transport: strict `rg '\bWebSocket\b\|\bEventSource\b\|event-stream\|\bSSE\b' web/src/` → 0 hits (positive control: the same sweep finds `createConnectTransport` at `client.ts:24`). HTTP/1.1: no `h2c`/`http2`/`Protocols` configuration anywhere in `internal/uiserver/server.go`; the Go tests exercise it over `httptest` HTTP/1.1 (`livehandler_test.go:89,213,533`). |
| 3 | ≥3 tabs keep receiving; slow client gets backpressure not blocking; reconnect with backoff not a storm; **and** the `pendingWriter` lifecycle verdict recorded | ✓ VERIFIED | See the two sub-rows below — both halves hold. |
| 3a | multi-tab / backpressure / reconnect | ✓ VERIFIED | Backpressure mechanism, **run by me**: `TestWatchGraphHandlerCoalescesForANonReadingClient` PASS — `published=5000 received=11 ratio=0.0022 last_generation=5001` (a non-reading client coalesces to the newest, never blocks the publisher); `TestWatchGraphHandlerMultipleSubscribersIndependentChannels` PASS. Browser half: `corpora/live-push-multitab-check.json` — I **recomputed every clause** of `live-push-multitab-check.mjs:607`'s success formula from the raw fields: 3 tabs, each `receivedGenerationsPerTab` a superset of `triggeredGenerations [2,3,4]` (recomputed → `[true,true,true]`), `blockedTabReceiptsDuringBlock 0` while `healthyTabsIsolationSupersetOK true` (isolation proven with the blocked tab *proven blocked*), `slowTabFinalGeneration 6 == newestGeneration 6` (coalesced, not dropped), `reconnectAttemptsPerTab [3,3,3]` inside the 2..6 no-storm band with per-tab delays growing off base `[1000,2000,4000]` under a 0.25 jitter spread, `resumeCursorSentPerTab [6,6,6]`, `postReconnectAppliedPerTab [1,1,1]`, `pageErrorCount 0`. Backoff shape is also unit-pinned: `web/tests/live-client.test.ts:111,222,268,324` (green in the full suite I ran). |
| 3b | recorded lifecycle verdict vs Phase 1's `pendingWriter` | ✓ VERIFIED | The verdict **exists as a written artifact**: `06-07-SUMMARY.md:125-189`. I re-ran its own commands: the pending/inflight-struct sweep over `$(ls internal/uiserver/*.go \| rg -v _test.go)` returns **0 hits** (exit 1), while the same pattern against `internal/mcp/server.go` returns `466:type pendingWriter struct {` — the control discriminates. Counts reproduce exactly: bare `pendingWriter` = 9, `type pendingWriter struct` = 1. Structural separation in the new code confirmed at `internal/uiserver/livepublish.go:265 sendCount atomic.Int64` vs `:315 r.subs[id] = ch`, `:366 r.sendCount.Add(1)`; `TestLiveRegistryCounterSeparation` (`livepublish_test.go:642`) **run by me** with `-race -count=1`: PASS. |
| 4 | Graph nodes do not jump; layout updated incrementally, not re-run | ✓ VERIFIED | Mechanism in code: `GraphCanvas.svelte` has the **data-only fast path** (`:560-586` — batch data/class updates, `endBatch`, `return` **before** any layout call) and the structural path that captures every surviving node's model position (`:590-597`) and hands it to `runLayout` as `survivorPositions` (`:600`), which authoritatively writes each survivor's prior x/y back (`:337-347`) under a `layoutGeneration` token that aborts superseded runs (`:282,320-336`). Guava-scale measurement: `corpora/graph-live-update-check.json` — real `google/guava@94f39958` checkout, expanded to the **file level** (`fileLevelNodeCountBefore 366 → After 372`), `leafSurvivorCount 349`, `maxLeafDisplacementPx 0`, `meanLeafDisplacementPx 0`, `compoundSurvivorCount 17` also 0px. I recomputed `graph-live-update-check.mjs:573`'s formula from the raw fields: survivors ≥ 100 ✓, added ≥ 5 ✓, after > before ✓, maxDisp == 0 ✓, clean ✓. This is 349 surviving nodes, not the 6-node fixture D-03 explicitly ruled non-generalising. Unit coverage `web/tests/graph-live-update.test.ts` (921 lines) green in the suite I ran. |
| 5 | **Non-negotiable:** real `daemon` + `serve --mcp` + live-push session against one store survives repeated real re-index flushes without starving a sync or holding the store open | ✓ VERIFIED — **re-executed by this verifier** | I ran `bash scripts/live-push-concurrency-check.sh --out <scratch>` myself: **exit 0**, `flushesAttempted 5 / flushesCompleted 5 / flushesStarved 0`, `maxFlushDurationMs 266` against a **freshly measured** `baselineMaxFlushDurationMs 296`, `liveEventsExcludingSeed 5`, all three product processes alive at end with real distinct PIDs (`codegraph daemon start` 42547, `codegraph serve --mcp` 43854, `codegraph ui` 43817). See the "Criterion 5" section below for why the bound and the processes are genuine and not faked. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified, 0 overrides).

### Criterion 5 — the specific things the brief asked me to confirm

| Question | Answer | Evidence |
|---|---|---|
| Is the bound against a **measured** baseline or a hardcoded constant? | **Measured.** | `scripts/live-push-concurrency-check.sh:253-262` runs a full `BASELINE phase — daemon alone, $FLUSH_COUNT real re-index flushes` before the UI/probe topology starts; `:376 BASELINE_MAX_MS=$(jq '…max…' "$WORKDIR/baseline-all.json")`; `:379 MAX_ALLOWED=$(( 3 * BASELINE_MAX_MS + 1000 ))`. Nothing is hardcoded except the 3× multiplier and the 1000ms additive slack. My own run's baseline (296ms) differs from the committed record's (261ms) — confirming it is re-derived per run, not a stored constant. |
| Are the three processes **genuinely spawned**? | **Yes.** | `daemon`: `:250 CODEGRAPH_DEBOUNCE_MS=… "$BIN" daemon start --path "$SCRATCH_REPO" … & DAEMON_PID=$!` with a `kill -0` liveness check. `ui`: `:275 "$BIN" ui --no-open --path … &`, and the script *waits for the process to print its real URL* before proceeding. `serve --mcp`: spawned as a real child by the probe — `scripts/live-push-probe.go:187 cmd := exec.Command(*bin, "serve", "--mcp", "--path", *path)`, whose **own** PID is reported (`:211 verdict["serveMcpPid"] = cmd.Process.Pid`). MCP liveness is proven by a **real `codegraph_status` tool-call response** over a real initialize handshake, never by a bare process check (`corpora…json.mcpToolCallResponse`). All four PIDs in my run were distinct and non-zero. |
| Is the flush completion oracle real? | **Yes.** | Each flush writes a uniquely-named Go function into the scratch repo and polls the product's own `codegraph search "$token" --json` until *that exact symbol* is queryable (`:184-198`) — not a bare `last_sync_unix_ms` advance, which would only prove *an* index ran. |
| Is "starved" operationally defined? | **Yes.** | `:216-221` — starved iff the daemon's own log for that flush window contains a verbatim store-lock contention line, **or** the flush never became queryable within `flushTimeoutMs`. Not a judgement call. |
| Does the publisher hold the store open? | **No.** | `internal/uiserver/livepublish.go:98-127` — `computeChange` opens the engine per wake and `defer closer.Close()` at `:127`; the snapshot is never held across the stream's lifetime, which is exactly the property `06-CONTEXT.md`'s criterion-5 block names as binding. |
| Was the gate demonstrated **RED**? | **Claimed, and not independently reproducible from the repo — by design.** | `06-07-SUMMARY.md:198` records `flushesAttempted 3 / flushesCompleted 0 / flushesStarved 3` with the daemon's own store-lock log lines captured as `starvationEvidence`, produced by a **temporary, reverted** edit gated on `CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN=1` that skipped `defer closer.Close()`. I confirmed the revert is complete: `rg 'CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN'` over the whole tree excluding `.planning/` → **0 hits**. So the RED artifact is honest about being a one-off and the flag did not leak into shipped source — but the RED itself is **claimed, not independently confirmable by me**. I mitigated this by confirming the gate's discriminating *machinery* is real and reachable (the starvation predicate, the log-scrape, the timeout branch, the `SUCCESS=false` clauses at `:381-389`), and by re-running the gate GREEN myself. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/uiproto/uiv1/ui.proto` | `WatchGraph` server-streaming rpc on the existing `UIService` | ✓ VERIFIED | `:121`, 14th rpc, same service; `task proto:drift` PASS (4 generated files byte-identical). |
| `internal/uiserver/livepublish.go` | Store watcher, `Meta`-based change detector with open/read/close, bounded coalescing registry | ✓ VERIFIED | 665 lines; `computeChange` `:98` with `defer closer.Close()` `:127`; `liveRegistry` `:250-426`. |
| `internal/uiserver/livehandler.go` | Streaming handler + subscriber lifecycle | ✓ VERIFIED | 96 lines, 16 tests in `livehandler_test.go`, all green. |
| `internal/uiserver/watchtimeout.go` | Path-scoped write-deadline clearing (D-02) | ✓ VERIFIED | 67 lines; passes the **original** `w` through so connect's `http.Flusher` assertion still sees it (documented `:1-33`); `TestStreamDeadline*` ×4 green, incl. `TestStreamDeadlineNonStreamPathStillCutOff` proving the 60s bound survives for unary paths. |
| `web/src/lib/live/live-client.ts` | Reconnecting stream consumer, jittered backoff, generation resume | ✓ VERIFIED | 238 lines; jitter spread pinned `< 1/3` with the growth theorem stated and unit-tested. |
| `web/src/lib/live/live-store.ts` | Epoch-scoped admission gate | ✓ VERIFIED | 98 lines; epoch-or-greater-generation admission handles server restart (counter restarting at 1). |
| `scripts/live-push-concurrency-check.sh` | Criterion 5 real-process gate | ✓ VERIFIED | 458 lines; **re-executed by me, exit 0**. |
| `scripts/live-push-probe.go` | Real Connect stream client + persistent MCP harness | ✓ VERIFIED | 270 lines, `//go:build ignore`; spawns real `serve --mcp`. |
| `web/scripts/live-push-multitab-check.mjs` | 3-tab browser gate | ✓ VERIFIED | 649 lines; success formula audited clause-by-clause and recomputed against the record. |
| `web/scripts/graph-live-update-check.mjs` | Guava-scale displacement gate | ✓ VERIFIED | 606 lines; formula recomputed against the record. |
| `web/build/**` | Rebuilt bundle matching sources | ✓ VERIFIED | `task web:drift` PASS — 110 source files, 32 output files, **both** digests match; `git status --porcelain web/build` clean. |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `+layout.svelte` | `live-store.ts` | `createLiveStore()` + `setContext('liveStore', …)` (`:40-41`) | ✓ WIRED |
| `+layout.svelte` | `status.ts` status gate | `statusGate.applyLiveEvent(...)` on every admitted event (`:44`) — **criterion 1's chrome link** | ✓ WIRED |
| `live-store.ts` | `client.ts` `uiClient` | `uiClient.watchGraph(request, options)` (`:33`) | ✓ WIRED |
| `health/+page.svelte` | `uiClient.getHealth` | `getContext('liveStore')` → live-triggered re-fetch with shared ordering token (`:81,112`) | ✓ WIRED |
| `browse/+page.svelte` | `uiClient.getNodeDetail` | `getContext('liveStore')` (`:140`) | ✓ WIRED |
| `graph/+page.svelte` | `uiClient.fileGraph` / `fileSymbols` | `getContext('liveStore')` (`:243,300`) | ✓ WIRED |
| `graph/+page.svelte` | `GraphCanvas.svelte` | `applyLiveUpdate` seam → fast path / write-back path (`:560,590,600`) | ✓ WIRED |
| `server.go` | `watchtimeout.go` | `clearWatchDeadline` middleware scoped to the WatchGraph procedure only | ✓ WIRED |

### Behavioral Spot-Checks (all run by this verifier)

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full Go unit suite | `GOTOOLCHAIN=go1.26.5 task test:unit` | exit 0, every package `ok` | ✓ PASS |
| Full web suite | `cd web && pnpm test` | exit 0 — **42 files, 464/464 tests** | ✓ PASS |
| Criterion 2 per-message timing | `go test ./internal/uiserver/ -run '^TestWatchGraphStreamDeliversMessageByMessage$' -count=1 -v` | PASS 0.81s, `generations=[2 3 4]`, gaps ≥ 150ms | ✓ PASS |
| Backpressure / coalescing | `go test … -run '^TestWatchGraphHandlerCoalescesForANonReadingClient$' -count=1 -v` | PASS — `published=5000 received=11 ratio=0.0022` | ✓ PASS |
| Fan-out isolation | `go test … -run '^TestWatchGraphHandlerMultipleSubscribersIndependentChannels$' -count=1` | PASS | ✓ PASS |
| Lifecycle counter separation | `go test … -run '^TestLiveRegistryCounterSeparation$' -race -count=1 -v` | PASS | ✓ PASS |
| Criterion 5 real-process gate | `bash scripts/live-push-concurrency-check.sh --out <scratch>` | **exit 0** — 5/5 completed, 0 starved, 266ms ≤ 3×296+1000 | ✓ PASS |
| Proto drift | `task proto:drift` | exit 0 — 4 files byte-identical | ✓ PASS |
| Web drift | `task web:drift` | exit 0 — source **and** output digests match | ✓ PASS |
| Zero new dependencies | `git diff --stat 033ee9cb..HEAD -- go.mod go.sum web/package.json web/pnpm-lock.yaml` | **empty**; positive control on `web/src/lib/status.ts` prints `67 insertions(+), 7 deletions(-)`, so the command is not silently broken | ✓ PASS |

Every named test was confirmed to **exist** (`rg 'func Test…'` / `rg 'it\('`) before being run, so a
zero-match `-run` pattern could not masquerade as a pass.

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|---|---|---|---|
| RPC-04 | Connect server-streaming method carries re-index events over plain HTTP/1.1 | ✓ SATISFIED | `ui.proto:121`; no h2c in `server.go`; `TestWatchGraphStreamDeliversMessageByMessage` PASS; no SSE/WebSocket anywhere in `web/src/`. |
| LIV-01 | Watcher re-index events feed the streaming rpc | ✓ SATISFIED | `livepublish.go` store watcher → `Meta` change detector → registry → handler; end-to-end against **real** `daemon`/`serve --mcp`/`ui` in the gate I re-ran (5 real flushes → 5 live events excluding seed). |
| LIV-02 | Open views update in place when the index changes | ✓ SATISFIED | 4 subscribing surfaces (`health`, `browse`, `graph`, `AnalysisPanel`) + the chrome path via `applyLiveEvent`; `live-route-refetch.test.ts` ×5 cases green. |
| LIV-03 | Multiple tabs, backpressure, clean shutdown/reconnect | ✓ SATISFIED | Go coalescing + independent-channel tests run by me; `corpora/live-push-multitab-check.json` recomputed; backoff-growth unit tests green; **plus** the recorded `pendingWriter` verdict at `06-07-SUMMARY.md:125-189`, whose commands I re-ran. |
| LIV-04 | Graph layout stays stable across live updates | ✓ SATISFIED | Fast path + authoritative write-back + generation token in `GraphCanvas.svelte`; 349 survivors at 0px in `corpora/graph-live-update-check.json`. |

No orphaned requirements: `.planning/REQUIREMENTS.md:144,175-178` maps exactly RPC-04 and
LIV-01..04 to Phase 6, and all five are claimed by the phase's plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `scripts/live-push-concurrency-check.sh` | 98 | `XXXXXX` | ℹ️ Info | `mktemp -d` template, not a debt marker. |

Debt-marker gate: **clean.** `rg 'TBD|FIXME|XXX'` across the 37 non-planning, non-bundle files
changed in `033ee9cb..HEAD` yields only the `mktemp` template above; `rg 'TODO|HACK|PLACEHOLDER'`
yields **zero**. Both sweeps were validated with a positive control (`rg -l -i live` over the same
file list → 33 files), because my first attempt at this scan used a broken shell loop that silently
returned nothing — the control caught it and the scan was redone.

### Known-and-accepted items — checked, none materially worse

| Item | Verdict |
|---|---|
| **WINDOWS 26** (non-fatal `cytoscape-elk` adapter error) | Present, and now **identified**: `corpora/graph-live-update-check.json` records `pageErrorCount: 1` — `"Cannot read properties of null (reading 'notify')"`. `05-07-SUMMARY.md:252` names WINDOWS 26 as exactly *"the deterministic cytoscape-elk adapter `notify` error"*, and `05-VERIFICATION.md:121` documents it as guava-scale-specific — which is precisely the run that produced it here. Non-fatal (layout completed; 349 survivors measured at 0px). Note for the record: `graph-live-update-check.mjs:573`'s success formula does **not** gate on `pageErrorCount`, whereas its sibling `live-push-multitab-check.mjs:609` does. Consistent with the accepted carry-forward; not materially worse. |
| **WINDOWS 28** (guava-scale layout warning) | Not re-encountered; unchanged. |
| **WINDOWS 29** (`web:drift` staging blind spot) | Confirmed present as described (`Taskfile.yml:42` `git ls-files` vs `:60` `find`). The phase's compensating control held: `git status --porcelain web/build` is empty and `task web:drift` matches **both** digests. |
| ~30 plan-id leaks in test/harness files | **Counted: 33** `06-0[0-9]` occurrences, and all 33 are in test/harness files. Production (non-test, non-script) sources changed by this phase contain **0**. Matches the accepted scope exactly. |

## Gaps Summary

None. All five ROADMAP Success Criteria — including the non-negotiable criterion 5 and the
binding Notes qualifications on criteria 2, 4 and 5 — are met, and the evidence for each is
primary rather than narrative.

### Confidence ledger (what I proved vs. what I could only audit)

- **Verified by primary evidence I generated myself:** criteria 1, 2, 3 (mechanism half + the
  recorded-verdict half), 5; both drift gates; the zero-dependency claim; the debt-marker scan.
- **Verified by machine-written record whose verdict I recomputed from raw fields, plus code
  and unit tests I ran:** the browser halves of criteria 3 and 4 (`live-push-multitab-check.json`,
  `graph-live-update-check.json`). I did **not** re-drive Chromium: `live-push-multitab-check.mjs`
  writes probe files into *this* repository and runs a real `codegraph sync`, which would mutate
  the user's live index, and `graph-live-update-check.mjs` requires the cached guava corpus
  checkout. Both records are internally consistent, carry `browserIdentity` (chromium
  151.0.7922.34), and satisfy every clause of their own scripts' success formulas as recomputed
  by me — but the act of running the browser is second-hand.
- **Claimed but not independently confirmable:** criterion 5's RED demonstration (a deliberately
  temporary, reverted source edit). The revert is verified complete; the RED observation itself
  rests on the summary's record.

---

_Verified: 2026-09-07T21:44:29Z_
_Verifier: Claude (gsd-verifier)_
