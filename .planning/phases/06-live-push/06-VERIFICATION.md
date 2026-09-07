---
phase: 06-live-push
verified: 2026-09-07T22:52:33Z
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
  - ".planning/phases/06-live-push/06-08-PLAN.md"
  - ".planning/phases/06-live-push/06-08-SUMMARY.md"
  - "corpora/graph-live-update-check.json"
  - "corpora/live-push-concurrency-check.json"
  - "corpora/live-push-multitab-check.json"
  - "internal/uiproto/uiv1/ui.proto"
  - "internal/uiserver/handlers.go"
  - "internal/uiserver/livehandler.go"
  - "internal/uiserver/livepublish.go"
  - "internal/uiserver/server.go"
  - "internal/uiserver/watchtimeout.go"
  - "scripts/live-push-concurrency-check.sh"
  - "scripts/live-push-probe.go"
  - "web/scripts/graph-live-update-check.mjs"
  - "web/scripts/live-push-multitab-check.mjs"
  - "web/src/lib/components/graph/GraphCanvas.svelte"
  - "web/src/lib/components/workbench/AnalysisPanel.svelte"
  - "web/src/lib/live/live-client.ts"
  - "web/src/lib/live/live-store.ts"
  - "web/src/lib/status.ts"
  - "web/src/routes/+layout.svelte"
  - "web/src/routes/+page.svelte"
  - "web/src/routes/browse/+page.svelte"
  - "web/src/routes/graph/+page.svelte"
  - "web/src/routes/health/+page.svelte"
  - "web/src/routes/workbench/+page.svelte"
  - "web/tests/live-route-refetch.test.ts"
covered_digest: "v1:sha256:4a9cd1bef011d7efef127367cd0f4f0fb702f9d6d8d05e245199c940f27fdb9c"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  reopened:
    - "Criterion 1 — reopened by live browser UAT after the initial pass: the root Status route (`web/src/routes/+page.svelte`), the default landing view, subscribed to nothing"
  gaps_closed:
    - "Criterion 1 — the root Status route now subscribes to liveStore and re-issues GetStatus on a newer generation (06-08, commit 110fbb7a)"
  gaps_remaining: []
  regressions: []
---

# Phase 6: Live Push Verification Report

**Phase Goal:** Open views stop going quietly stale — a watcher re-index reaches the browser over the same schema and the same client as every other call, and updates what is on screen in place.
**Verified:** 2026-09-07T22:52:33Z (re-verification) — initial pass 2026-09-07T21:44:29Z
**Status:** passed
**Re-verification:** Yes — criterion 1 was reopened by live UAT and closed by plan 06-08
**Diff range:** phase `033ee9cb..HEAD` (`48213907`); gap-closure range `05d9b64f..HEAD`

## Verification stance

No claim from any `06-0N-SUMMARY.md`, `06-VALIDATION.md`, `06-REVIEW*.md` or `06-SECURITY.md`
was accepted as evidence. Every criterion below resolves to source read directly, a named test
confirmed to *exist* (`rg 'func Test…'` / `rg "it\('…"`) and then run, a corpora record whose
pass formula was **recomputed from the raw fields** rather than read off its own `success` flag,
or — for criterion 5 — the gate **re-executed** against a fresh set of real OS processes.

For this re-verification round the bar was raised further on criterion 1: the fix was
**independently driven RED twice** (once by full revert, once by a surgical revert of only the
ordering guards), and the *shipped bundle* — not just the source — was checked for the fix.

---

## Goal Achievement

### Observable Truths (the five ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Edit → re-index → **the open view** updates in place, **including** the staleness/health chrome | ✓ VERIFIED — **reopened after the initial pass, closed by 06-08** | See "Criterion 1" below. Every one of the app's six index-data surfaces now applies live events; the previously-missed root Status route is fixed at `web/src/routes/+page.svelte:24,84-104`, covered by three discriminating tests, and present in the committed bundle. |
| 2 | Per-message streaming over plain HTTP/1.1, same Protobuf schema, same Connect client, no separate transport | ✓ VERIFIED | `internal/uiserver/livehandler_test.go:556 TestWatchGraphStreamDeliversMessageByMessage` **run by me** with `-count=1`: PASS in 0.81s, `generations=[2 3 4]`, gaps ≥ 150ms. Its structure is the assertion — it blocks on `stream.Receive()` *before* issuing the next real store write, so a buffered implementation stalls to the 5s deadline (and did, per the recorded RED). Same schema/client: `rpc WatchGraph(WatchGraphRequest) returns (stream WatchGraphEvent);` is the 14th rpc **inside the existing `UIService`** (`internal/uiproto/uiv1/ui.proto:121`); the browser calls it through the one shared client — `live-store.ts:33` `uiClient.watchGraph(...)`, `uiClient` = `createClient(UIService, createConnectTransport(...))` at `web/src/lib/client.ts:23-32`. No new transport: strict `rg '\bWebSocket\b\|\bEventSource\b\|event-stream\|\bSSE\b' web/src/` → 0 hits (positive control: the same sweep finds `createConnectTransport` at `client.ts:24`). HTTP/1.1: no `h2c`/`http2`/`Protocols` configuration anywhere in `internal/uiserver/server.go`; the Go tests exercise it over `httptest` HTTP/1.1 (`livehandler_test.go:89,213,533`). |
| 3 | ≥3 tabs keep receiving; slow client gets backpressure not blocking; reconnect with backoff not a storm; **and** the `pendingWriter` lifecycle verdict recorded | ✓ VERIFIED | See the two sub-rows below — both halves hold. |
| 3a | multi-tab / backpressure / reconnect | ✓ VERIFIED | Backpressure mechanism, **run by me**: `TestWatchGraphHandlerCoalescesForANonReadingClient` PASS — `published=5000 received=11 ratio=0.0022 last_generation=5001` (a non-reading client coalesces to the newest, never blocks the publisher); `TestWatchGraphHandlerMultipleSubscribersIndependentChannels` PASS. Browser half: `corpora/live-push-multitab-check.json` — I **recomputed every clause** of `live-push-multitab-check.mjs:607`'s success formula from the raw fields: 3 tabs, each `receivedGenerationsPerTab` a superset of `triggeredGenerations [2,3,4]` (recomputed → `[true,true,true]`), `blockedTabReceiptsDuringBlock 0` while `healthyTabsIsolationSupersetOK true` (isolation proven with the blocked tab *proven blocked*), `slowTabFinalGeneration 6 == newestGeneration 6` (coalesced, not dropped), `reconnectAttemptsPerTab [3,3,3]` inside the 2..6 no-storm band with per-tab delays growing off base `[1000,2000,4000]` under a 0.25 jitter spread, `resumeCursorSentPerTab [6,6,6]`, `postReconnectAppliedPerTab [1,1,1]`, `pageErrorCount 0`. Backoff shape is also unit-pinned: `web/tests/live-client.test.ts:111,222,268,324` (green in the full suite I ran). |
| 3b | recorded lifecycle verdict vs Phase 1's `pendingWriter` | ✓ VERIFIED | The verdict **exists as a written artifact**: `06-07-SUMMARY.md:125-189`. I re-ran its own commands: the pending/inflight-struct sweep over `$(ls internal/uiserver/*.go \| rg -v _test.go)` returns **0 hits** (exit 1), while the same pattern against `internal/mcp/server.go` returns `466:type pendingWriter struct {` — the control discriminates. Counts reproduce exactly: bare `pendingWriter` = 9, `type pendingWriter struct` = 1. Structural separation in the new code confirmed at `internal/uiserver/livepublish.go:265 sendCount atomic.Int64` vs `:315 r.subs[id] = ch`, `:366 r.sendCount.Add(1)`; `TestLiveRegistryCounterSeparation` (`livepublish_test.go:642`) **run by me** with `-race -count=1`: PASS. |
| 4 | Graph nodes do not jump; layout updated incrementally, not re-run | ✓ VERIFIED | Mechanism in code: `GraphCanvas.svelte` has the **data-only fast path** (`:560-586` — batch data/class updates, `endBatch`, `return` **before** any layout call) and the structural path that captures every surviving node's model position (`:590-597`) and hands it to `runLayout` as `survivorPositions` (`:600`), which authoritatively writes each survivor's prior x/y back (`:337-347`) under a `layoutGeneration` token that aborts superseded runs (`:282,320-336`). Guava-scale measurement: `corpora/graph-live-update-check.json` — real `google/guava@94f39958` checkout, expanded to the **file level** (`fileLevelNodeCountBefore 366 → After 372`), `leafSurvivorCount 349`, `maxLeafDisplacementPx 0`, `meanLeafDisplacementPx 0`, `compoundSurvivorCount 17` also 0px. I recomputed `graph-live-update-check.mjs:573`'s formula from the raw fields: survivors ≥ 100 ✓, added ≥ 5 ✓, after > before ✓, maxDisp == 0 ✓, clean ✓. This is 349 surviving nodes, not the 6-node fixture D-03 explicitly ruled non-generalising. Unit coverage `web/tests/graph-live-update.test.ts` (921 lines) green in the suite I ran. |
| 5 | **Non-negotiable:** real `daemon` + `serve --mcp` + live-push session against one store survives repeated real re-index flushes without starving a sync or holding the store open | ✓ VERIFIED — **re-executed by this verifier** (initial round) | `bash scripts/live-push-concurrency-check.sh --out <scratch>`: **exit 0**, `flushesAttempted 5 / flushesCompleted 5 / flushesStarved 0`, `maxFlushDurationMs 266` against a **freshly measured** `baselineMaxFlushDurationMs 296`, `liveEventsExcludingSeed 5`, all three product processes alive at end with real distinct PIDs (`codegraph daemon start` 42547, `codegraph serve --mcp` 43854, `codegraph ui` 43817). See the "Criterion 5" section for why the bound and the processes are genuine. Not re-run this round — justified below. |

**Score:** 5/5 truths verified (0 present-but-behavior-unverified, 0 overrides).

---

## Criterion 1 — reopened by live UAT, closed by plan 06-08

### What happened

Criterion 1 passed the initial verification round (2026-09-07T21:44:29Z, HEAD `b4064168`) at
5/5. A **live browser UAT run afterwards found a genuine criterion-1 defect the verification had
missed**: `web/src/routes/+page.svelte` — the root Status route, the default landing view
`codegraph ui` opens — subscribed to nothing. It fetched `getStatus` once in `onMount` and then
rendered node/edge/file counts a generation behind while displaying `Stale: no`. Every other
index-data surface updated around it; this one, the first thing a developer sees, did not.

Plan `06-08` was written (`4e3d1dca`) and executed (`110fbb7a`) to close it. This section
records that fix, verified independently of the orchestrator's account of it.

### Verified: every index-data view now subscribes

**Enumeration method (stated, because the original miss was an enumeration failure).** I did
**not** start from the subscriber list. I enumerated the app's index-data surfaces *first*, from
the wire contract inward, and only then intersected with the subscriber list:

1. Took all 14 rpc names straight from `internal/uiproto/uiv1/ui.proto`
   (`rg -o 'rpc ([A-Za-z]+)\(' -r '$1'`).
2. Searched for each rpc's client call site **multiline-tolerantly** (`rg -U`). This mattered:
   a naive single-line `rg 'uiClient\.[a-zA-Z]+\('` finds only 3 files, because
   `+page.svelte` — the very file at issue — writes `uiClient` and `.getStatus({})` on
   *separate lines*. The single-line form would have hidden the defect a second time.
3. Followed indirection: rpcs with no direct `uiClient.` site are reached through
   `src/lib/browse-state.ts`, `src/lib/search.ts`, `src/lib/file-search.ts`,
   `src/lib/status.ts` and `SourcePane.svelte`; each was traced to its consuming view.
4. Enumerated all route files (`find src/routes -type f` → 7, incl. `+layout.ts`) and all
   `.svelte` components, and confirmed the remaining components are presentational (they
   appear in **no** `rg -l '\buiClient\b' src/` hit).
5. Only then compared against `rg -n "getContext.*liveStore"` / `liveStore.subscribe`.

**Result — the full surface set, each row checked, none omitted:**

| Surface | Index-derived data it renders | Data source | Applies live events? |
|---|---|---|---|
| `routes/+layout.svelte` (StatusBanner chrome) | staleness / health verdict, commit | `status.ts` gate | ✓ `statusGate.applyLiveEvent(...)` `:41-46` — direct application, no refetch |
| `routes/+page.svelte` (**root Status route**) | all 9 `GetStatusResponse` fields | `getStatus` | ✓ **fixed by 06-08** — `:24` `getContext`, `:84-104` subscribe |
| `routes/health/+page.svelte` | health counts / trust verdict | `getHealth` | ✓ `:81,114` |
| `routes/browse/+page.svelte` | node detail + blast radius | `browse-state.ts` → `getNodeDetail` | ✓ `:140,178` → `issueBrowseLiveRefetch` |
| `routes/graph/+page.svelte` | file graph + symbols | `fileGraph`, `fileSymbols` | ✓ `:243,302` |
| `routes/workbench/+page.svelte` | impact / callers / callees / affected | those 4 rpcs, dispatched through `AnalysisPanel` | ✓ via `AnalysisPanel.svelte:152,184` → `issueLiveRerun` |

**Surfaces deliberately *not* subscribed — checked, and each with a reason, rather than
silently omitted:**

| Surface | Why not a criterion-1 gap |
|---|---|
| `SearchPanel.svelte` / command palette (`search.ts`: `search`, `explore`, `files`) | Typeahead. Results exist only while the user is typing and are re-queried per keystroke; a selection navigates away. There is no persistent surface left on screen to go stale. |
| `FilePicker.svelte` (`file-search.ts`: `files`) | Same typeahead shape. |
| `SourcePane.svelte` (`getPermalink`) | A permalink is **commit-pinned by construction**. Continuing to point at the commit it was minted for is correct behaviour, not staleness. A commit *move* is separately surfaced: `WatchGraphEvent` carries `commit_sha` (field 6) and the layout applies it into the status chrome. |

### Verified: the root route's fix is real (source read at HEAD `48213907`)

| Required property | Verdict | Evidence in `web/src/routes/+page.svelte` |
|---|---|---|
| Obtains `liveStore` via `getContext` | ✓ | `:24 const liveStore = getContext<LiveStore \| undefined>('liveStore');` — the same `'liveStore'` key `+layout.svelte:41` sets. |
| A newer generation re-issues the **same** `getStatus` call | ✓ | `:84-104` subscribe → `:102 issueStatusFetch(generation)` → `:46 uiClient.getStatus({})`. The generation is a **trigger only**. |
| **D-05:** no `WatchGraphEvent` field enters the render path | ✓ | `rg -n 'live\.event' src/routes/+page.svelte` → exactly two hits, `:90` and `:94`, **both** `live.event.generation` and both used only for ordering comparisons. The render path is `{@const status = state.status}` (`:126`), and every one of the 10 markup bindings (`:145,148,151,154,157,160,165,177,180`) reads `status.*` — a `GetStatusResponse`. Nothing from the event is rendered. |
| Mount fetch and live fetches share ONE monotonic `requestId` | ✓ | `:29 let requestId = 0;` is the single counter; `:44 const id = ++requestId;` sits inside `issueStatusFetch`, which is the **only** fetch path — `onMount` calls `issueStatusFetch(null)` (`:80`) and the live path calls `issueStatusFetch(generation)`. |
| **Every** settle callback is guarded | ✓ | `:48` in `.then` and `:56` in `.catch` — both `if (id !== requestId) return;`. There is no third settle path that writes `state` (the `.finally` at `:59-67` touches only the coalescer bookkeeping, never `state`). |
| The coalescer **records a pending generation** rather than suppressing | ✓ | `:95-100` — an event arriving while `liveInFlight` sets `livePendingGeneration` to the newest generation and returns; `:60-66 .finally` then issues exactly one follow-up with that recorded generation. Suppression alone would lose it permanently; this does not. |
| Rendered markup and `classify()` unchanged | ✓ | The diff has exactly **three hunks** (`@@ -3,8 +3,15 @@`, `@@ -14,10 +21,31 @@`, `@@ -25,8 +53,54 @@`), the last ending at new line 106. `classify()` is at `:112` and the markup begins at `:119` — both outside every hunk. The whole change removes exactly **two** lines: `import { onMount } from 'svelte';` and `onMount(() => {`. |

### Verified: the tests are real, and they discriminate

**Existence, by exact-text grep** (a `-t` pattern matching nothing exits 0 and looks like a pass —
so the declarations were confirmed present *before* anything was run):

```
211:describe('+page.svelte (root Status route): live-triggered re-fetch (LIV-02)', () => {
212:  it('delivering one live event issues exactly ONE additional status call; a replayed lower generation issues none', …
234:  it('the pending-generation coalescer: an event during an in-flight re-fetch issues no second concurrent call, and exactly one follow-up carries the newer generation once it settles', …
263:  it('CR-01 regression: a live event arriving while the mount fetch is still unresolved must not let the stale mount response overwrite the newer live-triggered one', …
```

`web/tests/live-route-refetch.test.ts` now declares **8** `it(` cases (5 pre-existing health-route
cases + these 3).

**GREEN:** `pnpm vitest run tests/live-route-refetch.test.ts` → `Test Files 1 passed (1) · Tests 8 passed (8)`.

**RED #1 — full revert (the defect itself).** `git checkout 05d9b64f -- web/src/routes/+page.svelte`
(confirmed the fix was out: `rg -c 'getContext|liveStore' src/routes/+page.svelte` → 0), then re-ran:

```
⎯⎯⎯ Failed Tests 3 ⎯⎯⎯
 FAIL  +page.svelte (root Status route) > delivering one live event issues exactly ONE additional status call…
 FAIL  +page.svelte (root Status route) > the pending-generation coalescer…
 FAIL  +page.svelte (root Status route) > CR-01 regression…
AssertionError: expected 1 to be 2 // Object.is equality
      Tests  3 failed | 5 passed (8)
```

`expected 1 to be 2` is the UAT symptom exactly: a live event produced **no** additional
`GetStatus` call, so the page kept rendering generation-old counts. The 5 health-route cases
stayed green, so the failure is scoped to the reverted file rather than a global break.

**RED #2 — surgical revert (proves the CR-01 case is not riding on the subscription).** With the
fix restored, I deleted **only** the two `if (id !== requestId) return;` guards (`:48`, `:56`) and
left everything else intact:

```
⎯⎯⎯ Failed Tests 1 ⎯⎯⎯
 FAIL  +page.svelte (root Status route) > CR-01 regression: a live event arriving while the mount fetch is still unresolved…
      Tests  1 failed | 7 passed (8)

 ❯ tests/live-route-refetch.test.ts:288:17
    286|   resolveMount(statusResponse({ commitSha: 'a'.repeat(40) }));
    287|   await new Promise((r) => setTimeout(r, 0));
    288|   expect(screen.getByText('b'.repeat(40))).toBeInTheDocument();
       |                 ^
```

The newer live-triggered response (`bbbb…`) was overwritten on screen by the late, stale mount
response (`aaaa…`) — precisely the CR-01 failure mode, isolated to the one guard that prevents it.
Each of the three tests therefore discriminates against a distinct real defect.

**Restored and clean:** `git checkout HEAD -- web/src/routes/+page.svelte`; `git status --porcelain`
empty and `git diff HEAD --stat` empty, both re-confirmed after all gates finished.

### Verified: the fix is in the artifact that actually ships

`web:drift` proving source↔output digests match is the strong claim, but I also checked the
bundle directly, because the browser runs `web/build/`, not `web/src/`:

- `web/build/_app/immutable/nodes/2.ChamdFov.js` is uniquely the root Status route — it contains
  `Could not reach the server` and `Index is healthy` (control: node `3.BXcqitIo.js` contains
  neither).
- That chunk contains `liveStore` (1 hit). The **pre-fix** chunk at `05d9b64f`
  (`2.C970eePy.js`) contains **0**. The control discriminates, so this is real evidence the
  shipped bundle carries the subscription and not merely the source tree.

### Why the initial pass missed it — as a checkable lesson

The initial report stated that "four surfaces subscribe to the same liveStore and re-fetch through
their own rpcs (`health:81`, `browse:140`, `graph:243`, `AnalysisPanel:152`)". That enumeration was
**correct**. It was also the wrong enumeration to run: it started from the artifacts that exist and
confirmed each is wired, which can never surface a surface that was never wired at all. Absence has
no line number to cite.

Two concrete, checkable rules follow, both applied above:

1. **For an "every X does Y" criterion, enumerate X from a source that is independent of Y.**
   Here: derive the surface set from the proto's rpc list and the route/component tree, *then*
   intersect with the subscriber list — never read the surface set off the subscriber list.
   Any surface in the first set and not the second is the finding, and it is only visible if
   the first set was built independently.
2. **Grep shape is part of the evidence.** The single-line `rg 'uiClient\.[a-zA-Z]+\('` used to
   enumerate call sites silently skips the multiline `uiClient` / `.getStatus({})` chain that
   `+page.svelte` uses — it returns 3 files where the true answer is 6. When a sweep is
   load-bearing for a completeness claim, run it in multiline mode (`rg -U`) and pair it with a
   positive control that the sweep can find something it is known to contain.

A third, cheaper guard, also now applied: the phase's own test file covered five of six index-data
surfaces. **The surface missing from the suite was the surface missing the fix.** A coverage
enumeration against the same independently-derived surface list would have caught this before UAT
did.

---

## Criterion 5 — the specific things the brief asked me to confirm (initial round)

| Question | Answer | Evidence |
|---|---|---|
| Is the bound against a **measured** baseline or a hardcoded constant? | **Measured.** | `scripts/live-push-concurrency-check.sh:253-262` runs a full `BASELINE phase — daemon alone, $FLUSH_COUNT real re-index flushes` before the UI/probe topology starts; `:376 BASELINE_MAX_MS=$(jq '…max…' "$WORKDIR/baseline-all.json")`; `:379 MAX_ALLOWED=$(( 3 * BASELINE_MAX_MS + 1000 ))`. Nothing is hardcoded except the 3× multiplier and the 1000ms additive slack. My own run's baseline (296ms) differs from the committed record's (261ms) — confirming it is re-derived per run, not a stored constant. |
| Are the three processes **genuinely spawned**? | **Yes.** | `daemon`: `:250 CODEGRAPH_DEBOUNCE_MS=… "$BIN" daemon start --path "$SCRATCH_REPO" … & DAEMON_PID=$!` with a `kill -0` liveness check. `ui`: `:275 "$BIN" ui --no-open --path … &`, and the script *waits for the process to print its real URL* before proceeding. `serve --mcp`: spawned as a real child by the probe — `scripts/live-push-probe.go:187 cmd := exec.Command(*bin, "serve", "--mcp", "--path", *path)`, whose **own** PID is reported (`:211 verdict["serveMcpPid"] = cmd.Process.Pid`). MCP liveness is proven by a **real `codegraph_status` tool-call response** over a real initialize handshake, never by a bare process check (`corpora…json.mcpToolCallResponse`). All four PIDs in my run were distinct and non-zero. |
| Is the flush completion oracle real? | **Yes.** | Each flush writes a uniquely-named Go function into the scratch repo and polls the product's own `codegraph search "$token" --json` until *that exact symbol* is queryable (`:184-198`) — not a bare `last_sync_unix_ms` advance, which would only prove *an* index ran. |
| Is "starved" operationally defined? | **Yes.** | `:216-221` — starved iff the daemon's own log for that flush window contains a verbatim store-lock contention line, **or** the flush never became queryable within `flushTimeoutMs`. Not a judgement call. |
| Does the publisher hold the store open? | **No.** | `internal/uiserver/livepublish.go:98-127` — `computeChange` opens the engine per wake and `defer closer.Close()` at `:127`; the snapshot is never held across the stream's lifetime, which is exactly the property `06-CONTEXT.md`'s criterion-5 block names as binding. |
| Was the gate demonstrated **RED**? | **Claimed, and not independently reproducible from the repo — by design.** | `06-07-SUMMARY.md:198` records `flushesAttempted 3 / flushesCompleted 0 / flushesStarved 3` with the daemon's own store-lock log lines captured as `starvationEvidence`, produced by a **temporary, reverted** edit gated on `CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN=1` that skipped `defer closer.Close()`. I confirmed the revert is complete: `rg 'CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN'` over the whole tree excluding `.planning/` → **0 hits**. So the RED artifact is honest about being a one-off and the flag did not leak into shipped source — but the RED itself is **claimed, not independently confirmable by me**. I mitigated this by confirming the gate's discriminating *machinery* is real and reachable (the starvation predicate, the log-scrape, the timeout branch, the `SUCCESS=false` clauses at `:381-389`), and by re-running the gate GREEN myself. |

---

## Criteria 2–5 under the gap-closure change: still hold

The gap-closure range `05d9b64f..HEAD` is provably narrow, so criteria 2–5 could not have been
invalidated by it:

| Check | Command | Result |
|---|---|---|
| Every non-planning, non-bundle file changed in the range | `git diff --name-only 05d9b64f..HEAD \| rg -v '^\.planning/\|^web/build/'` | **exactly two**: `web/src/routes/+page.svelte`, `web/tests/live-route-refetch.test.ts` |
| Any Go / proto / server-side / script file changed | `git diff --name-only 05d9b64f..HEAD \| rg '\.go$\|\.proto$\|^internal/\|^cmd/\|^scripts/'` | **NONE** |
| `mutatingVerbs` (the read-only enforcement seam) touched | diff of the range restricted to the 5 files containing `mutatingVerbs` | **NONE — untouched** |
| Zero dependency change | `git diff --stat 05d9b64f..HEAD -- go.mod go.sum web/package.json web/pnpm-lock.yaml` | **empty**; positive control on `web/src/routes/+page.svelte` prints `76 insertions(+), 2 deletions(-)`, so the command is not silently broken |

Criterion-by-criterion consequence:

- **Criterion 2** rests on `ui.proto`, `internal/uiserver/*`, and `web/src/lib/client.ts` /
  `live-store.ts` — none changed. `task proto:drift` re-run PASS (4 generated files
  byte-identical). `internal/uiserver` re-run with `-count=1`: `ok … 33.100s`.
- **Criterion 3** rests on the Go handler/registry and `live-client.ts` — none changed; the
  corpora records are untouched by the range.
- **Criterion 4** rests on `GraphCanvas.svelte` and `routes/graph/+page.svelte` — neither changed.
- **Criterion 5** rests entirely on Go binaries and `scripts/` — none changed. The gate was **not**
  re-run this round, and this is the stated justification: the range contains zero Go, zero proto
  and zero script bytes, so a re-run could produce no new information about it. `web/build/` was
  rebuilt, and that rebuild is separately gated by `task web:drift` (below).

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/uiproto/uiv1/ui.proto` | `WatchGraph` server-streaming rpc on the existing `UIService` | ✓ VERIFIED | `:121`, 14th rpc, same service; `task proto:drift` PASS (4 generated files byte-identical) — re-run this round. |
| `internal/uiserver/livepublish.go` | Store watcher, `Meta`-based change detector with open/read/close, bounded coalescing registry | ✓ VERIFIED | 665 lines; `computeChange` `:98` with `defer closer.Close()` `:127`; `liveRegistry` `:250-426`. |
| `internal/uiserver/livehandler.go` | Streaming handler + subscriber lifecycle | ✓ VERIFIED | 96 lines, 16 tests in `livehandler_test.go`, all green. |
| `internal/uiserver/watchtimeout.go` | Path-scoped write-deadline clearing (D-02) | ✓ VERIFIED | 67 lines; passes the **original** `w` through so connect's `http.Flusher` assertion still sees it (documented `:1-33`); `TestStreamDeadline*` ×4 green, incl. `TestStreamDeadlineNonStreamPathStillCutOff` proving the 60s bound survives for unary paths. |
| `web/src/lib/live/live-client.ts` | Reconnecting stream consumer, jittered backoff, generation resume | ✓ VERIFIED | 238 lines; jitter spread pinned `< 1/3` with the growth theorem stated and unit-tested. |
| `web/src/lib/live/live-store.ts` | Epoch-scoped admission gate | ✓ VERIFIED | 98 lines; epoch-or-greater-generation admission handles server restart (counter restarting at 1). |
| `web/src/routes/+page.svelte` | **06-08:** root Status route applies live events | ✓ VERIFIED | 182 lines; `getContext` `:24`, subscribe `:84-104`, single `requestId` `:29,44` with both settle guards `:48,56`, pending-generation coalescer `:95-100,60-66`; markup and `classify()` provably outside every diff hunk. |
| `web/tests/live-route-refetch.test.ts` | Route-level live re-fetch coverage incl. the root route | ✓ VERIFIED | 8 `it(` cases; the 3 new root-route cases confirmed present by exact grep, run GREEN, and independently driven RED twice. |
| `scripts/live-push-concurrency-check.sh` | Criterion 5 real-process gate | ✓ VERIFIED | 458 lines; **re-executed, exit 0** (initial round). |
| `scripts/live-push-probe.go` | Real Connect stream client + persistent MCP harness | ✓ VERIFIED | 270 lines, `//go:build ignore`; spawns real `serve --mcp`. |
| `web/scripts/live-push-multitab-check.mjs` | 3-tab browser gate | ✓ VERIFIED | 649 lines; success formula audited clause-by-clause and recomputed against the record. |
| `web/scripts/graph-live-update-check.mjs` | Guava-scale displacement gate | ✓ VERIFIED | 606 lines; formula recomputed against the record. |
| `web/build/**` | Rebuilt bundle matching sources | ✓ VERIFIED | `task web:drift` re-run PASS — 110 source files (`1bafb2fa…`), 32 output files (`6809d302…`), **both** digests match; `git status --porcelain` clean; and the root-route chunk independently confirmed to carry the fix (see Criterion 1). |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `+layout.svelte` | `live-store.ts` | `createLiveStore()` + `setContext('liveStore', …)` (`:40-41`) | ✓ WIRED |
| `+layout.svelte` | `status.ts` status gate | `statusGate.applyLiveEvent(...)` on every admitted event (`:44`) — **criterion 1's chrome link** | ✓ WIRED |
| **`routes/+page.svelte`** | **`uiClient.getStatus`** | **`getContext('liveStore')` `:24` → subscribe `:84` → `issueStatusFetch` `:36` → `getStatus` `:46`** | **✓ WIRED (06-08 — previously ABSENT)** |
| `live-store.ts` | `client.ts` `uiClient` | `uiClient.watchGraph(request, options)` (`:33`) | ✓ WIRED |
| `health/+page.svelte` | `uiClient.getHealth` | `getContext('liveStore')` → live-triggered re-fetch with shared ordering token (`:81,114`) | ✓ WIRED |
| `browse/+page.svelte` | `browse-state.ts` → `getNodeDetail` | `getContext('liveStore')` (`:140,178`) → `issueBrowseLiveRefetch` | ✓ WIRED |
| `graph/+page.svelte` | `uiClient.fileGraph` / `fileSymbols` | `getContext('liveStore')` (`:243,302`) | ✓ WIRED |
| `workbench/+page.svelte` | `impact`/`callers`/`callees`/`affected` | `AnalysisPanel.svelte:152,184` → `issueLiveRerun` re-invokes the route's `run` | ✓ WIRED |
| `graph/+page.svelte` | `GraphCanvas.svelte` | `applyLiveUpdate` seam → fast path / write-back path (`:560,590,600`) | ✓ WIRED |
| `server.go` | `watchtimeout.go` | `clearWatchDeadline` middleware scoped to the WatchGraph procedure only | ✓ WIRED |

### Behavioral Spot-Checks

Rows marked **(re-run)** were executed in this re-verification round; the rest in the initial round.

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full web suite **(re-run)** | `cd web && pnpm test` | exit 0 — **42 files, 467/467 tests** (464 + the 3 new root-route cases) | ✓ PASS |
| Type/template check **(re-run)** | `cd web && pnpm check` | `1159 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS` | ✓ PASS |
| Root-route live re-fetch, GREEN **(re-run)** | `pnpm vitest run tests/live-route-refetch.test.ts` | 8/8 passed | ✓ PASS |
| Root-route live re-fetch, **RED #1** (full revert) **(re-run)** | same, with `+page.svelte` at `05d9b64f` | 3 failed \| 5 passed — `expected 1 to be 2` on all three | ✓ DISCRIMINATES |
| Root-route live re-fetch, **RED #2** (ordering guards only) **(re-run)** | same, with `:48,:56` guards deleted | 1 failed \| 7 passed — CR-01 case only, at `tests/…:288` | ✓ DISCRIMINATES |
| Phase Go package **(re-run)** | `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -count=1` | `ok … 33.100s` | ✓ PASS |
| Proto drift **(re-run)** | `GOTOOLCHAIN=go1.26.5 task proto:drift` | exit 0 — 4 files byte-identical | ✓ PASS |
| Web drift **(re-run)** | `GOTOOLCHAIN=go1.26.5 task web:drift` | exit 0 — source **and** output digests match | ✓ PASS |
| Shipped bundle carries the fix **(re-run)** | `rg -c liveStore web/build/…/nodes/2.ChamdFov.js` vs the same on `05d9b64f`'s `2.C970eePy.js` | `1` vs `0` — control discriminates | ✓ PASS |
| Working tree clean after all gates **(re-run)** | `git status --porcelain` / `git diff HEAD --stat` | both empty | ✓ PASS |
| Zero new dependencies in the gap-closure range **(re-run)** | `git diff --stat 05d9b64f..HEAD -- go.mod go.sum web/package.json web/pnpm-lock.yaml` | **empty**; positive control on `+page.svelte` prints `76 insertions(+), 2 deletions(-)` | ✓ PASS |
| Full Go unit suite | `GOTOOLCHAIN=go1.26.5 task test:unit` | exit 0, every package `ok` | ✓ PASS |
| Criterion 2 per-message timing | `go test ./internal/uiserver/ -run '^TestWatchGraphStreamDeliversMessageByMessage$' -count=1 -v` | PASS 0.81s, `generations=[2 3 4]`, gaps ≥ 150ms | ✓ PASS |
| Backpressure / coalescing | `go test … -run '^TestWatchGraphHandlerCoalescesForANonReadingClient$' -count=1 -v` | PASS — `published=5000 received=11 ratio=0.0022` | ✓ PASS |
| Fan-out isolation | `go test … -run '^TestWatchGraphHandlerMultipleSubscribersIndependentChannels$' -count=1` | PASS | ✓ PASS |
| Lifecycle counter separation | `go test … -run '^TestLiveRegistryCounterSeparation$' -race -count=1 -v` | PASS | ✓ PASS |
| Criterion 5 real-process gate | `bash scripts/live-push-concurrency-check.sh --out <scratch>` | **exit 0** — 5/5 completed, 0 starved, 266ms ≤ 3×296+1000 | ✓ PASS |

Every named test was confirmed to **exist** (`rg 'func Test…'` / `rg "it\('…"`) before being run, so
a zero-match `-run`/`-t` pattern could not masquerade as a pass.

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|---|---|---|---|
| RPC-04 | Connect server-streaming method carries re-index events over plain HTTP/1.1 | ✓ SATISFIED | `ui.proto:121`; no h2c in `server.go`; `TestWatchGraphStreamDeliversMessageByMessage` PASS; no SSE/WebSocket anywhere in `web/src/`. Untouched by the gap-closure range. |
| LIV-01 | Watcher re-index events feed the streaming rpc | ✓ SATISFIED | `livepublish.go` store watcher → `Meta` change detector → registry → handler; end-to-end against **real** `daemon`/`serve --mcp`/`ui` in the re-executed gate (5 real flushes → 5 live events excluding seed). Untouched by the gap-closure range. |
| LIV-02 | Open views update in place when the index changes | ✓ SATISFIED — **after 06-08** | All **six** index-data surfaces now apply live events (root Status, health, browse, graph, workbench-via-AnalysisPanel, plus the chrome via `applyLiveEvent`), enumerated independently from the proto rpc list and the route tree rather than from the subscriber list. `live-route-refetch.test.ts` 8/8, with the 3 root-route cases independently driven RED. Before 06-08 this requirement was **not** satisfied: the default landing view never re-fetched. |
| LIV-03 | Multiple tabs, backpressure, clean shutdown/reconnect | ✓ SATISFIED | Go coalescing + independent-channel tests run by me; `corpora/live-push-multitab-check.json` recomputed; backoff-growth unit tests green; **plus** the recorded `pendingWriter` verdict at `06-07-SUMMARY.md:125-189`, whose commands I re-ran. |
| LIV-04 | Graph layout stays stable across live updates | ✓ SATISFIED | Fast path + authoritative write-back + generation token in `GraphCanvas.svelte`; 349 survivors at 0px in `corpora/graph-live-update-check.json`. |

No orphaned requirements: `.planning/REQUIREMENTS.md:144,175-178` maps exactly RPC-04 and
LIV-01..04 to Phase 6, and all five are claimed by the phase's plans.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `scripts/live-push-concurrency-check.sh` | 98 | `XXXXXX` | ℹ️ Info | `mktemp -d` template, not a debt marker. |

Debt-marker gate: **clean.** Re-run over the two files changed in `05d9b64f..HEAD`:
`rg 'TBD\|FIXME\|XXX'` → none, `rg 'TODO\|HACK\|PLACEHOLDER'` → none, with a positive control
(`rg -c 'LIV-02'` → 2 and 4 hits respectively) proving the sweep can find what those files contain.
The phase-wide sweep from the initial round is unchanged.

### Known-and-accepted items — checked, none materially worse

| Item | Verdict |
|---|---|
| **WINDOWS 26** (non-fatal `cytoscape-elk` adapter error) | Present, and identified: `corpora/graph-live-update-check.json` records `pageErrorCount: 1` — `"Cannot read properties of null (reading 'notify')"`. `05-07-SUMMARY.md:252` names WINDOWS 26 as exactly *"the deterministic cytoscape-elk adapter `notify` error"*, and `05-VERIFICATION.md:121` documents it as guava-scale-specific. Non-fatal (layout completed; 349 survivors measured at 0px). Not materially worse. |
| **WINDOWS 28** (guava-scale layout warning) | Not re-encountered; unchanged. |
| **WINDOWS 29** (`web:drift` staging blind spot) | Confirmed present as described (`Taskfile.yml:42` `git ls-files` vs `:60` `find`). The compensating control held again this round: `git status --porcelain` empty and `task web:drift` matches **both** digests. |
| ~30 plan-id leaks in test/harness files | Per `06-CONTEXT.md`'s maintainer ruling. The 06-08 test file adds `06-08` references in comments; production sources changed by this phase still contain **0**. Matches the accepted scope. |
| **Carry-forward 1** — stock Svelte favicon that also violates CSP (`06-08-SUMMARY.md`) | Confirmed exactly as described, **not worse**: `web/src/lib/assets/favicon.svg` still contains `svelte-logo` and `#ff3e00`; `internal/uiserver/spa.go:99` still sets `default-src 'self'` with no `img-src`. Pre-existing (Phase 2 app-shell era), deliberately out of Phase 6 scope. |
| **Carry-forward 2** — two unconfirmed count observations (`sync` 24498 vs status 14047; `GetHealth` internal off-by-one) | Recorded as observations, not asserted as bugs; not re-investigated and no evidence found that either is worse. Out of Phase 6 scope. |
| `.playwright-mcp/` not gitignored | Absent from the tree at HEAD; `git status --porcelain` empty. Noted, unchanged. |

## Gaps Summary

None outstanding. Criterion 1 was **genuinely not met** at the initial pass — the default landing
view rendered stale counts under a `Stale: no` label — and is now met: all six index-data surfaces
apply live events, the root route's fix satisfies every required property including D-05 and the
shared ordering token, its three tests were independently driven RED twice, and the fix is present
in the committed bundle the server actually serves. Criteria 2–5 are untouched by the gap-closure
range (zero Go, proto, script or dependency bytes changed) and their gates re-run green.

### Confidence ledger (what I proved vs. what I could only audit)

- **Verified by primary evidence generated in this re-verification round:** criterion 1 end to end
  — independent surface enumeration, source read at HEAD, hunk-boundary proof that markup and
  `classify()` are unchanged, GREEN plus two independent REDs, shipped-bundle check with a
  discriminating control, both drift gates, `pnpm check`, the 467-test suite, the
  zero-dependency and zero-server-change claims, and a clean tree confirmed after every mutation.
- **Verified by primary evidence generated in the initial round:** criteria 2, 3 (mechanism half +
  recorded-verdict half), 5; the phase-wide debt-marker scan.
- **Verified by machine-written record whose verdict I recomputed from raw fields, plus code and
  unit tests I ran:** the browser halves of criteria 3 and 4 (`live-push-multitab-check.json`,
  `graph-live-update-check.json`). I did not re-drive Chromium: those scripts write probe files
  into this repository and run a real `codegraph sync`, which would mutate the user's live index.
- **Claimed but not independently confirmable:** criterion 5's RED demonstration (a deliberately
  temporary, reverted source edit). The revert is verified complete; the RED observation itself
  rests on the summary's record.

---

_Verified: 2026-09-07T22:52:33Z (re-verification; initial pass 2026-09-07T21:44:29Z)_
_Verifier: Claude (gsd-verifier)_
