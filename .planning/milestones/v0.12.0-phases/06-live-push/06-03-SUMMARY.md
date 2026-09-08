---
phase: 06-live-push
plan: 03
subsystem: ui
tags: [connect-rpc, svelte, streaming, backoff, jitter, generation-gate]

requires:
  - phase: 06-live-push
    provides: "06-01's frozen WatchGraph wire surface (watchGraph AsyncIterable client method, WatchGraphEvent's six GetStatusResponse-mirrored fields) — this plan's transport contract"
provides:
  - "startLiveClient: a reconnecting for-await stream consumer with exponential backoff+jitter (pinned shape, spread=0.25 < 1/3), generation resume, and a connection-epoch tag on every delivered event"
  - "createLiveStore: the epoch-scoped generation admission gate (new epoch always admits; same epoch requires strictly greater generation) plus appliedAtMs stamping on the observation seam"
  - "classifyStatus widened to StatusLikeFields (Pick over GetStatusResponse's five fields) so a WatchGraphEvent satisfies it directly with no translation layer; StatusGate.applyLiveEvent applies a live event through the SAME classifier with no getStatus round trip, ordered against fetchStatus by one shared monotonic counter"
  - "window.__codegraphLiveObservations: the third __codegraph* seam, populated by the client (events/connections) and the store (appliedAtMs), for 06-06's real-browser multi-tab gate"
  - "Live re-fetch subscriptions in health/browse/workbench, each coalesced with a pending-generation flag rather than suppression"
affects: [06-05, 06-06, 06-07]

actuals:
  tokens: 18649
  tasks: 3
  commits: 5
  plan_head_before: cebd57bf35282ab38495764ac741afe5aeda371c

tech-stack:
  added: []
  patterns:
    - "Pinned exponential-backoff-with-jitter shape as named module constants (LIVE_BACKOFF_BASE_MS/FACTOR/CAP_MS/JITTER_SPREAD), with the jitter spread's < 1/3 growth-theorem bound asserted directly by a test — not left as an emergent property of whatever value was chosen"
    - "Epoch-scoped generation gate: 'new epoch always admits; same epoch requires strictly greater generation' as the general recipe for any resumable stream whose per-process counter can restart mid-session"
    - "Structural Pick-typed classifier contract (StatusLikeFields) so two independently-generated message types (GetStatusResponse, WatchGraphEvent) can share one classification function with the type system, not a runtime convention, keeping them from drifting apart"
    - "Pending-generation coalescer (record the max seen during an in-flight operation, fire exactly one follow-up on settle) as the per-consumer amplification guard, applied identically across three unrelated Svelte components"

key-files:
  created:
    - web/src/lib/live/live-client.ts
    - web/src/lib/live/live-store.ts
    - web/tests/live-client.test.ts
    - web/tests/live-store.test.ts
    - web/tests/live-route-refetch.test.ts
  modified:
    - web/src/app.d.ts
    - web/src/lib/status.ts
    - web/src/routes/+layout.svelte
    - web/src/routes/browse/+page.svelte
    - web/src/routes/health/+page.svelte
    - web/src/lib/components/workbench/AnalysisPanel.svelte
    - web/tests/browse-page.test.ts

key-decisions:
  - "Backoff shape pinned exactly as CONTEXT.md's discretion note anticipated: base = 1000ms, factor = 2, cap = 30000ms, jitter spread = 0.25 (strictly below the 1/3 growth-theorem bound). Only the very first-ever connection attempt skips the wait entirely; every subsequent attempt — including the first reconnect after a long healthy session — waits, because a graceful disconnect can still recur in a storm."
  - "The backoff exponent resets to 0 only when an attempt DELIVERS AT LEAST ONE EVENT, not merely when it establishes a connection. A connection that opens successfully but immediately closes with zero events (the clean-EOF case T-06-42 targets) is treated identically to a thrown failure for backoff purposes, while the connection EPOCH still increments on that same establishment — these are two independent counters answering two different questions (T-06-41 vs T-06-42) that a single shared counter cannot answer for."
  - "Epoch increments on the first non-throwing `.next()` step of a connection (established), never per delivered event and never merely on calling the generated method — this is what lets 3 events on one connection share one epoch while a reconnect (even a zero-event one) always gets a new one."
  - "StatusGate.applyLiveEvent dedups by (epoch, generation) identity before minting an id, rather than minting unconditionally on every call. Without this, a late re-delivery of an already-applied live event would win the one-counter race against a fetch that legitimately started afterward — the SAME failure mode WR-08's fetch-vs-fetch guard already exists to prevent, now recurring at the fetch-vs-live boundary."
  - "live-store.ts stamps appliedAtMs by looking the observation record up by (epoch, generation) IDENTITY, never by generation alone — the same generation number legitimately recurs across a server restart's new epoch, and a generation-only lookup would silently stamp the WRONG record."
  - "The pending-generation coalescer is implemented inline, independently, in each of the three consumers (health/browse/AnalysisPanel) rather than factored into a shared helper — none of Task 3's own files_modified include a shared module, and the ~15-line pattern is small enough that a premature shared abstraction was judged not worth a file the plan didn't already scope."
  - "Deviation (Rule 3): web/tests/browse-page.test.ts's CR-01 test's default 5000ms timeout was widened to 15000ms after isolating the cause to pre-existing environment fragility (see Deviations below) unrelated to this plan's browse/+page.svelte change."

patterns-established:
  - "Injectable clock+random for time-based browser code (LiveClientClock/random), read live at call time so vi.useFakeTimers() patches the default transparently with no bespoke test harness needed"
  - "Observation-only window seam extended a third time (__codegraphLiveObservations, after 05-03's metrics and 05-08's geometry seams), with the SAME 'nothing under web/src/ reads it back' invariant enforced by a paired positive/negative grep in the plan's own verify command"

requirements-completed: [LIV-02, LIV-03]

coverage:
  - id: D1
    description: "The stream consumer reconnects with exponential backoff+jitter (pinned shape, spread < 1/3), resumes from the last-seen generation, and treats a clean stream end identically to a thrown failure"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: backoff growth and jitter bound"
        status: pass
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: generation resume"
        status: pass
    human_judgment: false
  - id: D2
    description: "Events are consumed one at a time as they arrive — the client never collects the stream before acting"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: incremental consumption — never buffers the stream"
        status: pass
    human_judgment: false
  - id: D3
    description: "A connection epoch increments once per successful establishment (not per event) and travels with every delivered event"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: connection epoch"
        status: pass
    human_judgment: false
  - id: D4
    description: "stop() aborts the in-flight request and cancels a pending backoff timer — no reconnect fires after it"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: stop()"
        status: pass
    human_judgment: false
  - id: D5
    description: "The observation seam (__codegraphLiveObservations) is declared, populated by the client, capped, and read by nothing under web/src/ except web/src/lib/live/*"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-client.test.ts#startLiveClient: the observation seam"
        status: pass
      - kind: other
        ref: "rg-based populated/unread grep pair in the plan's own Task 1 <verify>"
        status: pass
    human_judgment: false
  - id: D6
    description: "One classifier serves both a unary GetStatus response and a live WatchGraphEvent, producing the identical verdict across at least 6 field combinations"
    requirement: LIV-02
    verification:
      - kind: unit
        ref: "web/tests/live-store.test.ts#classifyStatus: one classifier for both a unary response and a live event"
        status: pass
    human_judgment: false
  - id: D7
    description: "The chrome updates from a live event with NO getStatus round trip, ordered against in-flight fetches by one shared monotonic counter in both directions"
    requirement: LIV-02
    verification:
      - kind: unit
        ref: "web/tests/live-store.test.ts#StatusGate.applyLiveEvent: no round trip"
        status: pass
      - kind: unit
        ref: "web/tests/live-store.test.ts#StatusGate: the one-counter invariant orders live applications against unary fetches"
        status: pass
    human_judgment: false
  - id: D8
    description: "The generation gate is scoped to the connection epoch — a tab reconnected to a restarted server (generation reset to 1) still applies what it receives"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "web/tests/live-store.test.ts#createLiveStore: the epoch-scoped generation gate"
        status: pass
    human_judgment: false
  - id: D9
    description: "Health, browse, and the workbench analysis panel each re-fetch through the rpc they already own, coalesced with a pending-generation flag rather than suppression"
    requirement: LIV-02
    verification:
      - kind: automated_ui
        ref: "web/tests/live-route-refetch.test.ts#health/+page.svelte: live-triggered re-fetch (LIV-02)"
        status: pass
      - kind: automated_ui
        ref: "web/tests/live-route-refetch.test.ts#workbench AnalysisPanel: live-triggered re-fetch (LIV-02)"
        status: pass
      - kind: automated_ui
        ref: "web/tests/live-route-refetch.test.ts#browse/+page.svelte: live-triggered re-fetch (LIV-02)"
        status: pass
    human_judgment: false
  - id: D10
    description: "The live trigger never goes through notifyNavigated's identity guard — proven over code, not prose"
    requirement: LIV-02
    verification:
      - kind: other
        ref: "rg negative/positive grep pair in the plan's own Task 3 <verify>"
        status: pass
    human_judgment: false

duration: 54min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 3: The Browser-Side Live Client Summary

**A reconnecting WatchGraph stream consumer with pinned exponential-backoff-plus-jitter and a connection-epoch tag, an epoch-scoped generation gate that survives a server restart, and one widened `classifyStatus` that lets the chrome update straight from the event with zero extra round trips.**

## Performance

- **Duration:** 54 min (approx, commit-timestamp-anchored)
- **Started:** 2026-09-07T15:35:00Z (approx)
- **Completed:** 2026-09-07T16:29:04Z
- **Tasks:** 3 (2 `tdd="true"` RED/GREEN pairs + 1 `type="auto"` single-commit task)
- **Files modified:** 12 (5 created, 7 modified)

## Accomplishments

- `startLiveClient` (Task 1, `web/src/lib/live/live-client.ts`): consumes `watchGraph` one event at a time via manual `AsyncIterator.next()` calls — never collecting the stream — reconnects with exponential backoff (base 1000ms, factor 2, cap 30000ms) plus ±25% jitter (`LIVE_BACKOFF_JITTER_SPREAD = 0.25`, verified `< 1/3` directly so the scheduled-delay growth theorem stays sound), applies the SAME backoff to a clean stream end as to a thrown failure, resumes from the last-seen generation on every reconnect, and tags every delivered event with a connection epoch that increments once per successful establishment. 13/13 tests pass.
- `createLiveStore` + `StatusGate.applyLiveEvent` (Task 2, `web/src/lib/live/live-store.ts` + `web/src/lib/status.ts`): `classifyStatus` now takes `StatusLikeFields` (a `Pick` over `GetStatusResponse`'s five status fields), so a `WatchGraphEvent` satisfies it directly with no second representation of the same state. `applyLiveEvent` classifies and emits with NO `getStatus` call, minting from the same monotonic counter `fetchStatus` uses so a live application and an in-flight fetch order correctly in both directions instead of racing — including against a duplicate re-delivery of the identical event. `createLiveStore`'s epoch-scoped gate admits an event when its epoch differs from the last admitted epoch, or its generation is strictly greater within the same epoch, and stamps `appliedAtMs` on the matching observation record by `(epoch, generation)` identity. 11/11 tests pass.
- Route wiring (Task 3): `+layout.svelte` constructs the live store alongside the status gate and applies every admitted event to the gate directly. `health/+page.svelte`, `browse/+page.svelte`, and `AnalysisPanel.svelte` each subscribe to the SAME store and re-issue the rpc they already own — `getHealth`, `getNodeDetail`/`loadBlastRadius` through the existing `NavigationGate`, and the panel's own `run` (only when it currently holds a result) respectively — each coalesced with a pending-generation flag so a live event arriving mid-flight is never dropped, only deferred to exactly one follow-up. 4/4 new tests pass; full suite 447/447 green.

## Task Commits

Each task was committed atomically (Tasks 1 and 2 each carry a RED then GREEN pair per this plan's `tdd="true"` tasks; Task 3 has no `tdd` attribute and is a single commit):

1. **Task 1 RED:** `b3378a8c` — `test(06-03): add failing tests for the live stream consumer` (13/13 real assertion failures against a deliberately wrong stub).
2. **Task 1 GREEN:** `0c393f20` — `feat(06-03): implement the reconnecting live stream consumer` (13/13 pass).
3. **Task 2 RED:** `f0265d91` — `test(06-03): add failing tests for classifyStatus/applyLiveEvent and the live store` (8/11 real logic-gap failures; 3 passed vacuously — disclosed below rather than papered over).
4. **Task 2 GREEN:** `90e2ead6` — `feat(06-03): widen classifyStatus and add the live store's epoch-scoped gate` (11/11 pass).
5. **Task 3 (single commit, `type="auto"`):** `27f492db` — `feat(06-03): wire route subscriptions to re-fetch through their own rpcs` (4/4 new tests pass; full suite 447/447).

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `web/src/lib/live/live-client.ts` — new: `startLiveClient`, the backoff constants, `LiveEvent`/`LiveClientClock`/`LiveClientDeps` types, the observation-seam writer.
- `web/src/lib/live/live-store.ts` — new: `createLiveStore`, the epoch-scoped admission gate, `appliedAtMs` stamping.
- `web/src/app.d.ts` — the third `__codegraph*` seam declaration (`__codegraphLiveObservations`).
- `web/src/lib/status.ts` — `StatusLikeFields`, widened `classifyStatus`, `StatusGate.applyLiveEvent`.
- `web/src/routes/+layout.svelte` — constructs and contexts the live store; applies admitted events to the status gate.
- `web/src/routes/health/+page.svelte`, `web/src/routes/browse/+page.svelte`, `web/src/lib/components/workbench/AnalysisPanel.svelte` — live re-fetch subscriptions with the pending-generation coalescer.
- `web/tests/live-client.test.ts`, `web/tests/live-store.test.ts`, `web/tests/live-route-refetch.test.ts` — new test files (28 new vitest cases total).
- `web/tests/browse-page.test.ts` — one test's timeout widened (deviation, see below).

## Decisions Made

See `key-decisions` in frontmatter. Highlights:
- Two independent counters answer two independent questions: the backoff exponent (reset only on an event, not on mere establishment) prevents a tight reconnect loop against a server that accepts-then-immediately-closes; the connection epoch (incremented on establishment, not on events) is what lets the store's generation gate survive a restart. Conflating them into one counter would have broken one of T-06-41/T-06-42.
- `StatusGate.applyLiveEvent`'s own `(epoch, generation)` dedup exists as defense-in-depth alongside the store's own gate — tested directly against the one-counter invariant so the ordering property holds even if a duplicate somehow reached the gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Widened `web/tests/browse-page.test.ts`'s CR-01 test timeout from the default 5000ms to 15000ms**
- **Found during:** Task 3's full-suite `<verify>` re-run.
- **Issue:** The full suite intermittently reported 446/447 (later, consistently 446/447), with `browse-page.test.ts`'s single test timing out at exactly 5000ms. Isolated to a pre-existing environment fragility, NOT a regression in this plan's code: running that ONE test file alone consistently took ~5.06s of actual test-body time regardless of whether `browse/+page.svelte` carried this plan's live-subscription wiring or was reverted to its pre-06-03 content (`git show`'d and swapped in directly) — both configurations timed out identically in isolation, and both passed when run as part of the FULL suite before this plan's four new test files existed. Adding this plan's test volume tipped the full-suite run over the same 5000ms edge the isolated run was already sitting on.
- **Fix:** Added an explicit `15000` ms timeout as the third argument to that one `it(...)` call — no change to the test's logic, assertions, or the route code it exercises.
- **Files modified:** `web/tests/browse-page.test.ts`
- **Verification:** Full suite (447 tests) green across 3 consecutive runs after the fix; 0 failures. `pnpm run check` clean throughout.
- **Committed in:** `27f492db` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking — a pre-existing test-timeout margin, not a logic defect this plan introduced). **Impact:** No scope creep; the fix touches only the flaking test's own timeout value.

## TDD Gate Compliance

Both `tdd="true"` tasks (1 and 2) carry a `test(06-03):` commit strictly before their `feat(06-03):` commit:
- Task 1: `b3378a8c` (test) → `0c393f20` (feat)
- Task 2: `f0265d91` (test) → `90e2ead6` (feat)

Neither needed a REFACTOR commit — GREEN's first implementation held without further cleanup.

**`gsd_run check tdd-red-evidence` was NOT invoked** — its classifier parses Node's `node --test` TAP reporter output; this is a Vitest-based frontend plan, a format that tool does not support either (consistent with `06-01`/`06-02`'s own precedent of not invoking it for their respective toolchains).

**RED evidence recorded manually, per the gate's actual intent** (a nonzero exit with the target tests failing on real assertions, never a compile error or fixture crash):
- Task 1: temporarily replaced `live-client.ts` with a stub (`LIVE_BACKOFF_JITTER_SPREAD = 0.9`, a no-op `startLiveClient`), ran `pnpm exec vitest run tests/live-client.test.ts --reporter=json` — 13/13 `failed`, all genuine assertion mismatches, 0 import/build errors. Restored the real implementation, re-ran — 13/13 passed.
- Task 2: temporarily reverted `status.ts` to its pre-Task-2 committed content (`git show HEAD:web/src/lib/status.ts`) and stubbed `live-store.ts` with a no-op `createLiveStore`, ran `pnpm exec vitest run tests/live-store.test.ts --reporter=json` — 8/11 `failed` on real logic gaps (`applyLiveEvent is not a function`, no generation gate, etc.); 3 passed VACUOUSLY (the `classifyStatus` equivalence tests pass regardless of the type widening since JS is structurally untyped at runtime, and the bare-null-delivery subscribe test is trivially satisfied by any no-op stub) — disclosed here rather than hidden, mirroring `06-02-SUMMARY.md`'s own precedent for a test that already held against a stub. Restored the real implementations, re-ran — 11/11 passed.

## Issues Encountered

None beyond the one deviation above, resolved within this plan.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `task web:test` — PASS: `web:test: observed numTotalTests=447 numPassedTests=447 (vitest exit 0)`.
- `pnpm -C web run check` — exits 0: `COMPLETED 1158 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`.
- `git status --porcelain -- web/build` — empty; no `web/build` rebuild was committed by this plan (06-07's single, staged rebuild remains untouched).

## Known Stubs

None. Every piece built in this plan — the stream consumer, the epoch-scoped store, `applyLiveEvent`, and all three route subscriptions — is fully implemented and tested end-to-end against synthetic streams and mounted components; no placeholder bodies remain in this plan's files.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

`06-05` can subscribe the graph route to the SAME `liveStore` context key this plan established, layering its own layout-stability work on top without re-deriving the generation/epoch contract. `06-06`'s real-browser multi-tab gate can read `window.__codegraphLiveObservations` directly — both `events` and `connections` are populated exactly as declared, with `appliedAtMs` distinguishing "received" from "applied". `06-07`'s real-process gate and the phase's single `web/build` rebuild are unaffected by anything in this plan. No blockers.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- All 13 files (5 created, 7 modified, this SUMMARY) — FOUND on disk.
- Commits `b3378a8c`, `0c393f20`, `f0265d91`, `90e2ead6`, `27f492db` — all FOUND in `git log --oneline --all`.
- Plan-level `<verification>` re-run live: `task web:test` PASS (447/447), `pnpm -C web run check` 0 errors/0 warnings, `git status --porcelain -- web/build` empty.
