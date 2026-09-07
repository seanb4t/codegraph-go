---
phase: 06-live-push
reviewed: 2026-09-07T20:46:43Z
depth: deep
files_reviewed: 32
files_reviewed_list:
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/degrade_test.go
  - internal/uiserver/handlers.go
  - internal/uiserver/livehandler.go
  - internal/uiserver/livehandler_test.go
  - internal/uiserver/livepublish.go
  - internal/uiserver/livepublish_test.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/rpcname_test.go
  - internal/uiserver/server.go
  - internal/uiserver/server_test.go
  - internal/uiserver/watchtimeout.go
  - internal/uiserver/watchtimeout_test.go
  - scripts/live-push-probe.go
  - web/scripts/graph-live-update-check.mjs
  - web/scripts/live-push-multitab-check.mjs
  - web/scripts/live-push-stable-proxy.mjs
  - web/src/app.d.ts
  - web/src/lib/components/graph/GraphCanvas.svelte
  - web/src/lib/components/workbench/AnalysisPanel.svelte
  - web/src/lib/live/live-client.ts
  - web/src/lib/live/live-store.ts
  - web/src/lib/status.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/browse/+page.svelte
  - web/src/routes/graph/+page.svelte
  - web/src/routes/health/+page.svelte
  - web/tests/browse-page.test.ts
  - web/tests/gen-watchgraph-type.test.ts
  - web/tests/graph-live-update.test.ts
  - web/tests/live-client.test.ts
  - web/tests/live-route-refetch.test.ts
  - web/tests/live-store.test.ts
findings:
  critical: 1
  warning: 4
  info: 2
  total: 7
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-09-07T20:46:43Z
**Depth:** deep
**Files Reviewed:** 32
**Status:** issues_found

## Summary

This phase's mechanics — the store-watcher/change-detector/coalescing-registry engine
(`livepublish.go`), the streaming handler (`livehandler.go`), the write-deadline middleware
(`watchtimeout.go`), and the backoff/epoch/admission logic on the browser side
(`live-client.ts`/`live-store.ts`) — are sound and match their own extensive doc comments. I
traced the specific hazards this review was pointed at and found them correctly closed:

- **Snapshot discipline (criterion 5).** `livepublish.go` imports neither `net/http` nor
  `connectrpc.com/connect` (verified by reading its import block directly) — structurally
  proven to hold no store handle across a stream's lifetime. `computeChange` opens, reads, and
  closes via a single `defer closer.Close()` with no second `openEngine`/`query.OpenAt` call
  site. `livehandler.go`'s `WatchGraph` never calls `withEngine`/`openEngine` at all.
- **The Flusher trap.** `clearWatchDeadline` passes the original `http.ResponseWriter` through
  unwrapped (confirmed by reading the middleware and its four tests, including the paired
  negative test at `watchtimeout_test.go:87` and the direct Flusher-assertion test at `:114`).
  Ordering is `originHostGuard(port, clearWatchDeadline(mux))` exactly as specified
  (`server.go:176`).
- **Coalescing proof.** `TestWatchGraphHandlerCoalescesForANonReadingClient`
  (`livehandler_test.go:221-308`) publishes 5000 generations and asserts all four load-bearing
  clauses (`>=1` floor, strictly-fewer, `received*2<=published` ratio, newest-generation-last) —
  none of the four is redundant with another, matching the documented design.
  `TestRPCNameIsCleanAgainstMutatingVerbs` (`rpcname_test.go`) pairs its zero-length guard with
  a positive control (`GetIndexHealth` must still collide) — not vacuous.
- **The backoff theorem.** `LIVE_BACKOFF_JITTER_SPREAD = 0.25 < 1/3` (`live-client.ts:32`), and
  `live-client.test.ts` unit-tests the strict-growth property directly on the *scheduled* (not
  merely base) delay. `baseDelayMs`/`scheduledDelayMs` are both `null` on the initial connect,
  and the multitab check's `postEntries` filter (`scheduledDelayMs !== null`) and
  `reconnectAttemptsPerTab` count (`postEntries.length`) both correctly exclude the initial
  connect — the cycle-3 review's M-1 finding is resolved at both the unit-test and
  browser-harness layers.
- **Epoch-scoped generation gate.** `live-store.ts`'s `admit()` unconditionally admits the
  first event of a new epoch and only requires strict generation growth *within* an epoch —
  correctly avoids the "restarted server's counter resets to 1" deafness trap.
- **Publisher shutdown ordering.** `Server.Serve`'s `ctx.Done()` branch stops the publisher
  *before* calling `Shutdown` (`server.go:243`), exactly as required — but see BL-01... no,
  see WR-01 below for the one branch of `Serve` where this ordering is silently skipped.

**Vacuity-sweep counts, as requested:** I examined every zero-count/zero-length assertion I
could find across the reviewed Go tests and `.mjs` harnesses (`rpcname_test.go`'s
`len(mutatingVerbs)==0`, the coalescing test's four clauses, `graph-live-update-check.mjs`'s
`maxLeafDisplacementPx===0`, `live-push-multitab-check.mjs`'s `blockedTabReceiptsDuringBlock===0`
and `pageErrorCount===0`) — **8 zero/upper-bound assertions inspected, 0 found without a
paired positive control or non-zero floor.** I found **1** vacuous-in-effect conditional that
is not a test guard but production logic masquerading as a race guard — see IN-01.

One genuine correctness defect was found in the browser live-refresh wiring (CR-01, below),
and a narrower Go-side lifecycle gap (WR-01). The rest are quality/maintainability items,
including confirmation of the orchestrator's already-recorded plan-id-leak finding with an
updated, much larger count than originally scoped.

## Critical Issues

### CR-01: `health/+page.svelte`'s live-triggered refetch races the initial mount fetch with no ordering guard — can silently revert to stale data

**File:** `web/src/routes/health/+page.svelte:47-122`
**Issue:**

The mount-time `GetHealth` fetch (lines 47-60) and the live-triggered `GetHealth` fetch
(`issueLiveHealthFetch`, lines 75-97) each construct their **own independent**
`AbortController` and assign `pageState` directly from whichever promise settles, with **no
shared ordering token** between the two call sites. Each one only checks its own
`controller.signal.aborted`, which is never set true by the other.

Compare this to the other two routes this same plan (`06-03 Task 3`) wired identically in
intent:

- `browse/+page.svelte`'s `issueBrowseLiveRefetch` calls `gate.advance()`/`gate.isCurrent(...)`
  — the **same** `NavigationGate` the URL-driven effect already uses (`browse/+page.svelte:60-128`
  vs `:130-199`) — so whichever fetch advances the gate last wins and the other's response is
  discarded.
- `AnalysisPanel.svelte`'s `issueLiveRerun` explicitly "Reuses the SAME `requestId` counter as
  the effect above" (`:144-150`) — confirmed: `const id = ++requestId;` (`:126` and its
  live-triggered counterpart) is checked against the shared `requestId` in both places.

`health/+page.svelte` does neither. It has no `NavigationGate`, no shared `requestId`, and no
mechanism at all connecting the two `.then()` callbacks.

**Reachable failure sequence:**
1. Route mounts; the mount effect issues `GetHealth` (F1), landing on the server exactly when
   `withEngine`/`openEngine` may be contending with a real concurrent re-index (this is
   precisely the scenario Criterion 5 exists to stress — `06-CONTEXT.md`'s own concurrency
   note).
2. Before F1 resolves, a real re-index publishes a new live event with generation > the
   baseline the `first`-flag recorded at mount. The live effect (lines 99-122) does **not**
   gate on `pageState.kind === 'loaded'` (unlike `graph/+page.svelte`'s
   `if (graphState.kind !== 'loaded') return;`, which explicitly closes this exact hole for the
   graph route) — it fires `issueLiveHealthFetch` (F2) immediately.
3. F2 resolves first (plausible: F1 may still be blocked behind a store lock retry; F2 starts
   later against an already-unlocked store).
4. F1 finally resolves and unconditionally overwrites `pageState` with its now-stale response.
   Nothing re-triggers: `liveIssuedGeneration` was already advanced past the live generation in
   step 2, so that generation can never fire another refetch.

The health page is then stuck showing pre-reindex data with no further UI signal, silently
defeating LIV-02 for this one route. This is the exact race class `web/src/lib/status.ts`'s own
doc comment documents as **WR-08**, already found and fixed once in this codebase
("fetchStatus had no cancellation or response-identity guard... an ordinary loopback
interleaving... could revert the banner to a stale verdict that then never updates"). This
phase's health-page wiring reintroduces the identical unguarded race.

**Not covered by the existing test:** `live-route-refetch.test.ts`'s health-page tests
(`:180-232`) always `await waitFor(() => expect(calls).toBe(1))` — i.e., wait for the mount
fetch to fully settle — before ever calling `live.deliver(...)`. The race window (a live event
arriving *before* the mount fetch resolves) is never exercised.

**Fix:** Share one monotonic request-ordering token between the mount effect and
`issueLiveHealthFetch`, mirroring `AnalysisPanel.svelte`'s pattern:

```ts
let requestId = 0;

$effect(() => {
	const controller = new AbortController();
	const id = ++requestId;
	uiClient
		.getHealth({}, { signal: controller.signal })
		.then((response) => {
			if (id !== requestId || controller.signal.aborted) return;
			pageState = { kind: 'loaded', response };
		})
		.catch((err: unknown) => {
			if (id !== requestId || controller.signal.aborted) return;
			pageState = { kind: 'failed', failure: describeWorkbenchFailure(err, indexStatus) };
		});
	return () => controller.abort();
});

function issueLiveHealthFetch(generation: bigint): void {
	liveIssuedGeneration = generation;
	liveInFlight = true;
	const controller = new AbortController();
	const id = ++requestId;
	uiClient
		.getHealth({}, { signal: controller.signal })
		.then((response) => {
			if (id !== requestId || controller.signal.aborted) return;
			pageState = { kind: 'loaded', response };
		})
		// ...same id guard in .catch()/.finally()
}
```

**Resolved (2026-09-07, `5fc33c57`):** Applied exactly as prescribed above — one
shared `requestId` between the mount effect and `issueLiveHealthFetch`. Added
a regression test (`live-route-refetch.test.ts`, "CR-01 regression") that
delivers a live event while the mount fetch is still unresolved and then
resolves the live-triggered fetch before the stale mount fetch; confirmed
RED against the pre-fix code (mount's stale response won) before the fix
landed, GREEN after. Also resolves WR-02 below (health now shares the same
ordering primitive as browse/AnalysisPanel).

## Warnings

### WR-01: `Server.Serve`'s abnormal-exit branch never stops the live publisher — goroutine/fsnotify-watcher leak

**File:** `internal/uiserver/server.go:224-259`
**Issue:** `Serve` only calls `s.stopPublisher()` on the `ctx.Done()` branch (line 243). The
sibling branch — `case err := <-errCh: return normalizeServeErr(err)` (lines 231-232) — returns
directly with **no call to `stopPublisher()`**. This branch is taken whenever
`s.srv.Serve(s.ln)` returns on its own for a reason other than `ctx` being cancelled by the
caller (e.g. the listener's underlying socket fails for a reason not initiated by this
package's own `Close()`, such as hitting a file-descriptor limit, or any other `Accept` error
`http.Server.Serve` propagates that isn't `http.ErrServerClosed`). In that branch, the
publisher's fsnotify watcher, debouncer goroutine, and watch-loop goroutine are never joined or
released.

This is distinct from the `Close()` path, which is fine (`Close()` itself calls
`stopPublisher()` before closing the listener, so a caller doing `cancel(); srv.Close()` or
`srv.Close()` alone is covered). The gap is specifically: **`Serve()` running as the sole
owner of the lifecycle, with the underlying `net.Listener`/`http.Server` failing on its own**
— untested (`TestServeReturnsNilOnContextCancellation` only exercises the `ctx.Done()` branch)
and unhandled.

**Fix:**
```go
select {
case err := <-errCh:
	s.stopPublisher()
	return normalizeServeErr(err)
case <-ctx.Done():
	...
```

**Resolved (2026-09-07, `b3b0d06e`):** Applied exactly as prescribed. Added
`TestServeStopsPublisherOnAbnormalExit`, which closes the raw listener
directly (never through Serve/Shutdown/Close) so Serve takes the errCh
branch with `ctx` never cancelled, then positively asserts the publisher's
registry was stopped via `liveRegistry.Subscribe`'s own documented
stopped-registry behavior (an already-closed channel). Confirmed RED
against the pre-fix code (channel yielded a value instead of being closed)
before the fix landed, GREEN after. The `ctx.Done()` branch's
publisher-before-`Shutdown` ordering is unchanged.

### WR-02: `AnalysisPanel`/`browse`/`health` live-refresh wiring correctly guards two of three routes; `health` is the outlier (cross-reference to CR-01)

**File:** `web/src/routes/health/+page.svelte`
**Issue:** Recorded separately from CR-01 for visibility: this is a design-consistency gap,
not just a single-file bug. Three routes were given near-identical doc comments this phase
("Coalesced with a PENDING-GENERATION flag... see health/+page.svelte's identical comment") but
only two of the three (`browse`, `AnalysisPanel`) actually share an ordering primitive with
their pre-existing fetch effect. `health` is the one that does not, despite its own doc comment
being the one the other two files point to as the canonical explanation. A future reader
copying `health/+page.svelte`'s pattern for a fifth view would propagate the same gap.
**Fix:** Apply CR-01's fix, then consider extracting the "one live-generation coalescer" shape
(now duplicated three times with near-identical code across `health`, `browse`,
`AnalysisPanel`, and a fourth variant in `graph/+page.svelte`) into a shared helper so the
ordering-guard requirement cannot be dropped by a future call site the way it was here.

**Resolved (2026-09-07, `5fc33c57`):** CR-01's fix applied; all three routes now
share an ordering primitive. The shared-helper extraction is deferred (a
refactor, not a defect fix) — recorded here for a future cleanup pass rather
than done in this fix pass.

### WR-03: `GraphCanvas.svelte`'s `addedElements` effect (file-to-symbol expansion) lacks the identity guard the same double-invocation hazard forced onto `elements` and `liveElements`

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:914-924` (pre-existing code, not
modified by this phase — confirmed via `git diff 033ee9cb..HEAD`)
**Issue:** This phase discovered and fixed **two** real, guava-scale-only bugs
(`06-05-SUMMARY.md`'s Deviations #1 and #2) both caused by the same root shape: a Svelte 5
`$effect` tracking an array-shaped `$state` prop firing *twice* for one logical value change.
Both fixes added a reference-identity guard (`lastAppliedElements`, `lastAppliedLiveElements`)
because the pre-existing boolean/no-guard pattern silently ate or double-applied a real update.

The fourth effect (`addedElements`, the file-to-symbol expansion's incremental-add path) has
**no such guard** — only "batch is non-empty" (line 921-923). Its own comment states the
"no-guard-needed" reasoning addresses only the *first-run* case (default empty array, `add()`
no-ops on empty), which is a different concern from the *double-fire-on-a-real-change* hazard
the other two effects were found to have. If this effect is subject to the same
double-invocation pattern (not proven here, but neither ruled out — it was never
guava-scale-tested in this phase, since file-to-symbol expansion is a Phase-4/5 feature outside
this phase's own measurement scope), a repeated invocation would call
`renderer.add(batch)` twice with the identical element array, which either throws (cytoscape
rejects a duplicate node id) or double-appends to `liveAddedElements`, corrupting the
symbol-survivor bookkeeping `swapElements()`/`replace()` depend on.
**Fix:** Add the same `lastAppliedAddedElements` reference-identity guard used for the other
two props, or explicitly justify in a comment why `addedElements` is provably immune to the
hazard the sibling effects were not.

**Resolved (2026-09-07, `d2780a2f`):** Added `lastAppliedAddedElements`, mirroring
`lastAppliedElements`/`lastAppliedLiveElements` exactly (same guard order:
length check, then identity check). **No regression test accompanies this
fix** — per this session's own anti-vacuity discipline, a jsdom test that
merely exercises the guard's own reference comparison, without ever
triggering a genuine Svelte double-`$effect`-invocation, would itself be a
vacuous guard on a vacuity finding. Per `06-05-SUMMARY.md`, the underlying
double-invocation behavior was only ever reproduced via real-browser
testing at guava scale, and neither existing sibling guard has an isolated
unit test proving double-invocation would break without it. 463/463
frontend tests continue to pass; `pnpm check`: 0 errors.

### WR-04: Plan-id leaks in shipped source comments — confirmed, count is far larger than previously scoped

**Files:** all `06-0[0-9]` occurrences across the 32 reviewed files (see list below)
**Issue:** Confirming the orchestrator's already-recorded finding (not rediscovering it): the
six previously-cited sites are still present verbatim —
`internal/uiserver/livehandler.go:22` ("replacing 06-01's CodeUnimplemented placeholder"),
`internal/uiserver/livepublish.go:2,7,10,293,526`. A broader sweep of the exact same
`06-0[0-9]` pattern across **all 32 files in this review's scope** (counted with `rg -o | wc
-l`, not `-c`, per this project's own durable-gate discipline) finds **53 total occurrences**,
not 6 — spread across test files (`readonly_test.go` ×5 lines, `livepublish_test.go`,
`livehandler_test.go`, `degrade_test.go`), the `.mjs` harnesses (`live-push-stable-proxy.mjs`,
`graph-live-update-check.mjs` ×4, `live-push-multitab-check.mjs` ×4), the `.svelte`/`.ts`
library and route files (`status.ts` ×3, `+layout.svelte`, `browse/+page.svelte`,
`graph/+page.svelte` ×4, `health/+page.svelte`, `AnalysisPanel.svelte`), and — most
significantly — **`internal/uiproto/uiv1/ui.proto` itself** (3 occurrences, lines 109, 942,
952: "WatchGraphRequest is plan 06-01's request...", "WatchGraphEvent is plan 06-01's push
payload..."), whose message-level doc comments are compiled verbatim into the generated
`ui.pb.go` and `web/src/lib/gen/ui_pb.ts` docstrings — confirmed present at `ui_pb.ts:1881,
1905, 2234` (excluded from this review as generated output, but their *source* is the in-scope
proto file). This is the concrete mechanism by which a plan-id leak becomes "rendered" output
in the sense the orchestrator's citation of the Phase-4 UI-copy precedent warns about: once
`06-01` compiles into a generated docstring, a future contributor reading the generated client
in isolation (which is the common case for downstream consumers) has no way to know what "plan
06-01" refers to once phase numbering resets in the next milestone.
**Fix:** Not applied here per the orchestrator's own instruction (avoid unplanned mid-wave
edits). Recommend a dedicated cleanup pass, prioritizing `ui.proto` first since it is the one
site whose leak propagates into build artifacts, then the non-test production files
(`livehandler.go`, `livepublish.go`, `status.ts`, the four route files), with test-file
citations lowest priority (a citation in a test's own doc comment is closer to this
repository's already-established, widespread convention of citing the introducing plan number
for historical traceability — e.g. `status.ts`'s own many "04-05 Task 1"/"03-09 Task 1"
citations predate this phase and were not flagged).

**Resolved (2026-09-07, `a6bc08c1`):** Applied in the prescribed priority order.
`ui.proto`'s 3 occurrences fixed first and regenerated via `task proto:gen`
(`task proto:drift` confirmed byte-identical afterward — docstring-only
diff, no field-number changes). Then `livehandler.go` (1), `livepublish.go`
(8), `status.ts` (3), and the four route/component files (`+layout.svelte`,
`browse/+page.svelte` ×1, `graph/+page.svelte` ×4, `health/+page.svelte` ×1,
`AnalysisPanel.svelte` ×1) — 20 occurrences total, replaced with
descriptions of the invariant each comment was actually pointing at. **30
occurrences remain** (down from 53), entirely in test files and the
`.mjs` real-browser measurement harnesses — both explicitly deprioritized
above and left for a future cleanup pass. 463/463 frontend tests pass;
`pnpm check`: 0 errors; `task test:unit`: PASS.

## Info

### IN-01: `status.ts`'s `applyLiveEvent` contains a vacuous, always-false guard

**File:** `web/src/lib/status.ts:275-279`
**Issue:**
```ts
const id = ++requestId;
// Synchronous end-to-end: nothing can supersede this id between
// minting it and emitting, so it always wins UNLESS it was
// itself recognized as a duplicate above.
if (id !== requestId) return;
emit(classifyStatus(fields));
```
The comment itself states the precondition for this branch to fire can never occur ("nothing
can supersede this id between minting it and emitting"). Since `++requestId` and the
comparison execute synchronously with no `await` between them, `id !== requestId` is
unreachable dead code — not a defensive guard, since it can never actually defend against
anything on this path (unlike the identical-looking guard inside `fetchStatus`'s `.then`/
`.catch`, which genuinely can be superseded across the `await` boundary). This is exactly the
"assertion that cannot fail proves nothing" shape this codebase's own `rpcname_test.go` names
by rule `84d1gfpywd` — here appearing as production logic rather than a test assertion, so its
cost is reader confusion (a future maintainer could reasonably assume a live event's emission
is genuinely racy against something) rather than a false-passing gate.
**Fix:** Remove the dead check, or replace the comment with a note that it is intentionally a
no-op invariant-assertion rather than functional logic (if left in for documentation purposes).

**Resolved (2026-09-07, `9d68ae8f`):** Removed the dead `if (id !== requestId) return;`
check. The `requestId` bump itself is kept (it is the real mechanism that
supersedes any in-flight `fetchStatus` call), with a comment explaining why
the bump alone is sufficient.

### IN-02: `WatchGraphRequest.since_generation` is threaded end-to-end but never read server-side

**Files:** `internal/uiserver/livehandler.go:43`, `web/src/lib/live/live-client.ts:104,185,195`
**Issue:** The client faithfully populates and sends `sinceGeneration: lastGeneration` on every
(re)connection (`live-client.ts:195`), and the wire message carries it
(`WatchGraphRequest.since_generation`, `ui.proto:944-948`). Server-side, `WatchGraph`
(`livehandler.go:43`) discards the entire request value: `_ *connect.Request[uiv1.WatchGraphRequest]`.
`Subscribe` always seeds a new subscriber with the registry's single `current` event
regardless of what the client claims to have already seen. This matches the recorded maintainer
decision (`06-01-SUMMARY.md`: "since_generation alone is the resume handle... nothing per-client
to filter on" — the server has no per-generation history to serve from, only a single current
value), so this is **not** a functional bug, but it is worth flagging: a future reader
encountering a populated, documented request field that is silently ignored on the only server
implementing it may reasonably assume resumption is partially server-driven when it is, in
fact, entirely a client-side concern (the epoch-scoped admission gate in `live-store.ts`).
**Fix:** None required functionally; consider a one-line comment on the Go handler noting the
field is deliberately unread, for symmetry with the equally deliberate doc comments already
present on the TypeScript side.

**Resolved (2026-09-07, `d8c7b4b4`):** Added the one-line comment on
`WatchGraph`. No functional change, matching the "None required
functionally" guidance above.

---

_Reviewed: 2026-09-07T20:46:43Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
