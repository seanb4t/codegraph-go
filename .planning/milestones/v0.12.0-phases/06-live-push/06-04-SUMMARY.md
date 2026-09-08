---
phase: 06-live-push
plan: 04
subsystem: api
tags: [connect-rpc, streaming, http-server, goroutine-lifecycle, uiserver]

requires:
  - phase: 06-live-push
    provides: "06-01's frozen WatchGraph wire surface (WatchGraphRequest/WatchGraphEvent, the UIServiceWatchGraphProcedure constant) and 06-02's livePublisher engine (Subscribe/Stop, the coalescing registry) — this plan's transport half"
provides:
  - "clearWatchDeadline (watchtimeout.go): a path-scoped middleware clearing http.Server's absolute 60s WriteTimeout for exactly the WatchGraph procedure, passing the ResponseWriter through UNWRAPPED so connect-go's flush-capability admission gate still admits the handler"
  - "(*uiService).WatchGraph real body: subscribe, send-loop, deregister — replacing 06-01's CodeUnimplemented placeholder, holding no store handle"
  - "runWatchGraphLoop: the extracted, independently-testable send loop, branching on ctx.Err() BEFORE classifying a send failure so a genuine transport fault is never mislabelled a client cancellation"
  - "Server.publisher/publisherCancel: the publisher's lifetime owned end to end by Listen (constructs), Serve's ctx.Done() branch and Close (both stop it, idempotently, BEFORE Shutdown)"
  - "TestWatchGraphStreamDeliversMessageByMessage: the phase's tracer, proving per-message delivery TIMING over a real store write, real fsnotify, real HTTP/1.1, and the real generated client"
affects: [06-05, 06-06, 06-07]

actuals:
  tokens: 12283
  tasks: 3
  commits: 3
  plan_head_before: 0cc0d7041fc6bca0edeb974e2aff9677d788f187

tech-stack:
  added: []
  patterns:
    - "Path-scoped deadline-clearing middleware, modelled directly on originHostGuard's pass-through-unwrapped shape: call http.NewResponseController(w) for its side effect only, never substitute or wrap w, so a library's own flusher-capability type assertion still sees the real writer"
    - "Extracted send-loop interface (watchGraphSender) so a streaming rpc's error-classification branch is unit-testable without a real HTTP/Connect round trip, when the library's own stream type has no exported constructor"
    - "Server-owned background-engine lifetime: construct in the bind step (Listen) against a locally-derived context, stop it from every shutdown path (Serve's ctx.Done(), Close) through one idempotent sync.Once-guarded method, and stop it BEFORE the graceful-shutdown call that would otherwise wait on it"

key-files:
  created:
    - internal/uiserver/watchtimeout.go
    - internal/uiserver/watchtimeout_test.go
    - internal/uiserver/livehandler_test.go
  modified:
    - internal/uiserver/livehandler.go
    - internal/uiserver/server.go
    - internal/uiserver/server_test.go
    - internal/uiserver/handlers.go
    - internal/uiserver/degrade_test.go

key-decisions:
  - "clearWatchDeadline matches on the generated UIServiceWatchGraphProcedure constant, never a hand-typed path literal, and is composed as the pinned expression originHostGuard(port, clearWatchDeadline(mux)) so the origin/host guard stays outermost — a forbidden host or origin is rejected before any deadline on that connection is touched"
  - "The publisher's construction context is derived from context.Background() inside Listen, not from any context Listen receives — Listen itself takes none (Options carries RepoPath and Addr only). Server owns both publisher and publisherCancel so every exit path (Serve's ctx.Done, Close) can stop it"
  - "runWatchGraphLoop was extracted as a plain function over a minimal watchGraphSender interface specifically so the ctx.Err()-first classification branch could be tested deterministically with a fake sender, since connect.ServerStream has no exported constructor and building one requires a full real HTTP/Connect round trip"
  - "A send failure is classified by checking ctx.Err() at return time, not by trusting connect-go's own internal ctx-aware wrapping inside Send — this keeps the classification decision visible in this package's own code, at the one site (runWatchGraphLoop) rather than split between this package and the library's internals"
  - "Deviation (Rule 1): TestDegradedRPCOpensTheStoreExactlyOnce's openEngine counting wrapper is installed AFTER startedServer rather than before, because Listen's new publisher construction now performs its own legitimate, synchronous openEngine call (the bootstrap check) that is unrelated to what that pre-existing test measures"

patterns-established:
  - "A streaming rpc's error-classification logic is extracted into a bare function over a minimal sender interface whenever the underlying stream type has no exported constructor, so the classification branch itself — not just the wire path — has a fast, deterministic unit test"

requirements-completed: [RPC-04, LIV-03]

coverage:
  - id: D1
    description: "A stream survives past the server's absolute 60s write deadline; all 13 unary rpcs and the SPA handler keep the unchanged bound"
    requirement: RPC-04
    verification:
      - kind: unit
        ref: "internal/uiserver/watchtimeout_test.go#TestStreamDeadlineSurvivesPastWriteTimeout"
        status: pass
      - kind: unit
        ref: "internal/uiserver/watchtimeout_test.go#TestStreamDeadlineNonStreamPathStillCutOff"
        status: pass
    human_judgment: false
  - id: D2
    description: "connect-go's flush-capability admission gate still admits the handler: the response writer clearWatchDeadline hands to next satisfies http.Flusher, and the response-controller deadline call itself returns no error"
    requirement: RPC-04
    verification:
      - kind: unit
        ref: "internal/uiserver/watchtimeout_test.go#TestStreamDeadlineHandlerReceivesFlushableWriter"
        status: pass
      - kind: unit
        ref: "internal/uiserver/watchtimeout_test.go#TestStreamDeadlineResponseControllerSetsDeadlineWithoutError"
        status: pass
    human_judgment: false
  - id: D3
    description: "Opening a stream registers exactly one subscriber (rise proven, not just the fall) and delivers the publisher's current state as the first message before any change occurs"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerRegistersAndDeregistersSubscriber"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerFirstMessageIsCurrentState"
        status: pass
    human_judgment: false
  - id: D4
    description: "A client that stops reading is coalesced, not queued: over a real HTTP/1.1 socket, publishing 5000 generations with no Receive call yields at most half received, at least one received, and the newest published generation as the last received"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerCoalescesForANonReadingClient"
        status: pass
    human_judgment: false
  - id: D5
    description: "A client disconnect is classified as a cancellation; a send failure raised while the context is still live is not — both branches proven, so a handler that hard-codes cancellation for every failure fails the second"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerDisconnectClassifiedAsCancellation"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerSendFailureWithLiveContextIsNotClassifiedAsCancellation"
        status: pass
    human_judgment: false
  - id: D6
    description: "The publisher's lifetime is owned by the Server: Listen starts it, Serve and Close both stop it (idempotently), and Listen-without-Serve leaks no goroutine"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestServerPublisherLifetimeReleasedOnCloseWithoutServe"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestServerPublisherLifetimeStopIsIdempotent"
        status: pass
    human_judgment: false
  - id: D7
    description: "Cancelling the server context with an open stream returns from Serve well under the 5-second Shutdown budget — the publisher stops BEFORE Shutdown is called"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestServerPublisherLifetimeShutdownIsPromptWithAnOpenStream"
        status: pass
    human_judgment: false
  - id: D8
    description: "Streams open and close without leaking a goroutine across repeated cycles"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphHandlerNoGoroutineLeak"
        status: pass
    human_judgment: false
  - id: D9
    description: "A real index write on disk reaches a real client one message at a time — per-message TIMING proven, not eventual arrival — with the deliberately-buffered handler observed FAILING this exact test"
    requirement: RPC-04
    verification:
      - kind: integration
        ref: "internal/uiserver/livehandler_test.go#TestWatchGraphStreamDeliversMessageByMessage"
        status: pass
    human_judgment: false

duration: 95min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 4: WatchGraph On The Wire Summary

**The write-deadline middleware, the real streaming handler, and the phase's tracer proving per-message delivery timing over a real store write, real fsnotify, and a real HTTP/1.1 client.**

## Performance

- **Duration:** 95 min
- **Started:** 2026-09-07T16:38:00Z (approx)
- **Completed:** 2026-09-07T18:13:00Z (approx, self-check time)
- **Tasks:** 3 (2 `type="auto"` + 1 `type="tracer"`)
- **Files modified:** 8 (3 created, 5 modified)

## Accomplishments

- `clearWatchDeadline` (Task 1, `internal/uiserver/watchtimeout.go`): clears `http.Server`'s absolute 60s `WriteTimeout` for exactly the generated `UIServiceWatchGraphProcedure` path, passing `w` through UNWRAPPED so connect-go's `checkServerStreamsCanFlush` (`protocol.go:347-350`) — a bare, non-traversing `http.Flusher` type assertion — still admits the handler. Wired as the pinned expression `originHostGuard(port, clearWatchDeadline(mux))`. Removing the deadline-clearing call and re-running the load-bearing test was observed FAILING this session (3 chunks received under a 250ms deadline, want ≥8); restoring it passed again (4/4 `TestStreamDeadline*` PASS).
- `(*uiService).WatchGraph` (Task 2, `internal/uiserver/livehandler.go`): the real subscribe/send/deregister body replacing 06-01's `CodeUnimplemented` placeholder. Holds no store handle — `withEngine`/`openEngine`/`query.OpenAt` appear nowhere in the file. The send loop is extracted into `runWatchGraphLoop` over a minimal `watchGraphSender` interface (since `connect.ServerStream` has no exported constructor), letting the ctx.Err()-first classification branch be tested directly against a fake sender. `Server` gained `publisher`/`publisherCancel`/`publisherStopOnce`; `Listen` constructs the publisher against its own `context.Background()`-derived context (Listen itself takes none) and passes it into `uiService`; `Serve`'s `ctx.Done()` branch stops the publisher BEFORE calling `Shutdown` (T-06-43 — an open stream would otherwise hold the 5s Shutdown budget); `Close` does the same for the bind-then-decline case. 15/15 tests in this task's verify slice pass under `-race` (11 `TestWatchGraphHandler*`, 3 `TestServerPublisherLifetime*`, `TestUIServiceHoldsNoStoreTypedField`), including `TestWatchGraphHandlerCoalescesForANonReadingClient` observed stable across 3 consecutive runs (received 18-26 of 5000 published, ratio 0.0036-0.0052, newest generation always last).
- `TestWatchGraphStreamDeliversMessageByMessage` (Task 3, the phase's tracer, `internal/uiserver/livehandler_test.go`): wires a real index write on disk through fsnotify, the debouncer, the change detector, the publisher's registry, the deadline-cleared middleware, real HTTP/1.1, and the real generated Connect client. Asserts TIMING — it blocks on receiving message *k* over the real socket before triggering message *k+1* — never a post-hoc arrival count. RED demonstrated this session with the send loop temporarily changed to buffer every event and flush only at stream end: the test FAILED (`client.WatchGraph` itself blocked for the full 5-second stream deadline and returned `deadline_exceeded`, because a buffering handler withholds even the seeded first message until the stream ends). Reverting restored GREEN with 3 triggered receipts, generations `[2 3 4]`, every inter-arrival gap ≥ the 150ms trigger spacing.

## Task Commits

Each task was committed atomically:

1. **Task 1: The deadline-clearing middleware** — `7544f244` (feat)
2. **Task 2: The streaming handler — subscribe, send, and deregister** — `b4f797b6` (feat)
3. **Task 3: End to end — the phase's tracer** — `c050696c` (test)

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `internal/uiserver/watchtimeout.go` — new: `clearWatchDeadline`, D-02's path-scoped write-deadline middleware
- `internal/uiserver/watchtimeout_test.go` — new: 4 `TestStreamDeadline*` tests
- `internal/uiserver/livehandler.go` — real `WatchGraph` body + extracted `runWatchGraphLoop`/`watchGraphSender`, replacing 06-01's placeholder
- `internal/uiserver/livehandler_test.go` — new: 3 `runWatchGraphLoop` classification tests, 11 `TestWatchGraphHandler*` (real HTTP), 3 `TestServerPublisherLifetime*`, and the Task 3 tracer
- `internal/uiserver/server.go` — `Server.publisher`/`publisherCancel`/`publisherStopOnce`; publisher construction in `Listen`; stop-before-`Shutdown` in `Serve`; stop in `Close`; `clearWatchDeadline` wired into the guard chain
- `internal/uiserver/server_test.go` — `TestUIServiceHoldsNoStoreTypedField`'s field-type-set expectation extended to `{"string", "*uiserver.livePublisher"}` (value-only edit; the `Options` half is untouched — confirmed by `git diff` carrying zero `wantOptsFields` lines)
- `internal/uiserver/handlers.go` — `uiService` gains a `publisher *livePublisher` field
- `internal/uiserver/degrade_test.go` — deviation fix, see below

## Decisions Made

See `key-decisions` in frontmatter. Highlights:
- The publisher's construction context is deliberately its OWN `context.Background()`-derived value, not something threaded through `Listen`'s parameters — `Listen` takes no context today (`Options` carries only `RepoPath`/`Addr`), so "tied to the server's lifetime" means `Server` owns that lifetime explicitly via `publisherCancel`.
- `runWatchGraphLoop`'s extraction into a bare function over `watchGraphSender` exists purely for testability: `connect.ServerStream` has no exported constructor, so the ctx.Err()-first classification branch would otherwise only be reachable through a full real HTTP/Connect round trip, which cannot deterministically force a "send failure while context is still live" condition.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestDegradedRPCOpensTheStoreExactlyOnce`'s `openEngine` counting wrapper reordered to install AFTER `startedServer`**
- **Found during:** Task 2's plan-level `<verification>` re-run (`go test ./internal/uiserver/ -count=1 -race`)
- **Issue:** This pre-existing test (from an earlier phase's D-15 work) swapped the package-level `openEngine` seam for a counting wrapper, then called `startedServer(t, dir)`, then made one degraded RPC call, asserting `openEngine` was invoked exactly once. `Listen`'s new publisher construction (Task 2) also calls `openEngine` once, synchronously, as its own bootstrap check (`newLivePublisher` → `checkAndPublish` → `computeChange`) — against the SAME package-level seam. With the store already locked by the test's own holder, that bootstrap call took the soft `ErrStoreLocked` skip branch (no crash) but still incremented the counter, making the total 2 instead of 1.
- **Fix:** Moved the `openEngine` swap to occur AFTER `startedServer(t, dir)` returns, so the publisher's own construction-time open (which happens during `Listen`, using the un-swapped `openEngine`) is not counted — isolating the counter to exactly what the test's own name and doc comment describe: one degraded RPC call.
- **Files modified:** `internal/uiserver/degrade_test.go`
- **Verification:** `TestDegradedRPCOpensTheStoreExactlyOnce` passes in isolation and as part of the full package suite under `-race`.
- **Committed in:** `b4f797b6` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — a test-counting-seam interaction directly caused by this task's own change to `Listen`). **Impact:** No scope creep; the fix restores the pre-existing test's actual intent without weakening its assertion.

## Issues Encountered

**Pre-existing, out-of-scope flake observed and NOT fixed:** `test/integration.TestLiveEditAutoSyncReachesExplore` (from Phase 3, commit `1a1b07bd`, unrelated to `internal/uiserver`) failed once during a full `task test:unit` run under concurrent-package load, with the test's own error message self-diagnosing "CR-01 store-lock collision?". Re-run in isolation: 3/3 PASS. Re-ran the full `task test:unit`: green. This matches the SAME class of documented, out-of-scope timing flake this repo already carries for `internal/daemon`'s watchdog test under parallel load (repo rule: "known, not a regression") — a different package, same root cause (timing sensitivity under concurrent test-binary contention), and not something this plan's `internal/uiserver` changes could cause or should fix.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -count=1 -race` — green, fresh (`go clean -testcache` run immediately before): `ok github.com/seanb4t/codegraph-go/internal/uiserver 37.276s`.
- `GOTOOLCHAIN=go1.26.5 task test:unit` — green (re-run after the flake above self-resolved on retry).
- `TestListenSetsEveryServerTimeout` — still passes unchanged (all 4 timeout subtests PASS).
- `GOTOOLCHAIN=go1.26.5 go vet ./...` and `GOTOOLCHAIN=go1.26.5 task lint:go` — clean, `0 issues.`

### Task-level automated verify commands (each re-run and captured)

- Task 1: `--- PASS: TestStreamDeadline*` × 4, `--- PASS: TestListenSetsEveryServerTimeout` × 1, 0 FAIL/DATA RACE.
- Task 2: `--- PASS: TestWatchGraphHandler*` × 11 (floor 8), `--- PASS: TestWatchGraphHandlerCoalescesForANonReadingClient` × 1 (exact), `--- PASS: TestServerPublisherLifetime*` × 3 (floor 2), `--- PASS: TestUIServiceHoldsNoStoreTypedField` × 1, 0 FAIL/DATA RACE.
- Task 3: `--- PASS: TestWatchGraphStreamDeliversMessageByMessage` × 1 (exact), 0 FAIL/DATA RACE, `task test:unit` green.

## Read-deadline verdict (recorded, per this plan's `<output>` instructions)

D-02 clears the WRITE deadline only. `readTimeout = 30 * time.Second` (`server.go:48`) does NOT need clearing for a long-lived stream: `net/http` clears the read deadline itself at `startBackgroundRead` (`$GOROOT/src/net/http/server.go:697`, `cr.rwc.SetReadDeadline(time.Time{})`) once the request body reaches EOF, which for a Connect server-streaming request happens immediately after the client's single request message is sent. No code change follows from this verdict — it is recorded here, with its source, because it is the first thing the next reader of this file would ask.

## Coalescing gate observations (recorded, per this plan's `<output>` instructions)

`TestWatchGraphHandlerCoalescesForANonReadingClient`, 3 consecutive `-race` runs, 5000 published (325 KB) with zero `Receive` calls until publishing finished:

| Run | received | ratio | last generation |
|-----|----------|-------|------------------|
| 1 | 18 | 0.0036 | 5001 (newest) |
| 2 | 23 | 0.0046 | 5001 (newest) |
| 3 | 26 | 0.0052 | 5001 (newest) |

All three orders of magnitude under the 0.5 threshold, and all three land within the previously-measured 3-28 range for a correct capacity-1 coalescing implementation. The gate deliberately does not rely on the socket ever blocking (measured threshold ~7,825 messages / ~496 KB on this project's own development platform) — do not raise `coalescePublishCount` to try to force a socket stall; that repeats a measured, disproven approach.

## Known Stubs

None. `(*uiService).WatchGraph`'s placeholder from 06-01 is fully replaced; no placeholder body remains reachable on this path — `TestWatchGraphStreamDeliversMessageByMessage` is the structural gate that could not pass while it survived, and it now does.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

`06-05` can subscribe the graph route's live-layout work to a server that now actually streams events end to end — the transport half of live push is complete and proven under timing, coalescing, and lifecycle assertions. `06-06`'s multi-tab and proxy gates, and `06-07`'s real-process gate, build directly on `WatchGraph`'s real body and the `Server.publisher` lifetime established here. No blockers.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- All 9 files (3 created, 5 modified, this SUMMARY) — FOUND on disk.
- Commits `7544f244`, `b4f797b6`, `c050696c` — all FOUND in `git log --oneline --all`.
- Plan-level `<verification>` re-run live: `go test ./internal/uiserver/ -count=1 -race` PASS (fresh, non-cached), `task test:unit` PASS, `TestListenSetsEveryServerTimeout` unchanged, `go vet`/`task lint:go` clean.
