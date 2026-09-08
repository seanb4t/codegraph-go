---
phase: 06-live-push
plan: 06
subsystem: ui
tags: [playwright, real-browser, reverse-proxy, reconnect, backpressure, fan-out]

requires:
  - phase: 06-live-push
    provides: "06-03's browser-side live client/store and the __codegraphLiveObservations seam (events[]/connections[]), and 06-04's real WatchGraph streaming handler with the publisher's lifetime owned by Server — this plan's transport and observation contracts"
provides:
  - "web/scripts/live-push-stable-proxy.mjs: startProxy({port}) -> {url, setUpstream, close}, a fixed-origin loopback reverse proxy that rewrites Host/Origin to the upstream's own authority and streams (never buffers) — the only way a browser tab can survive `codegraph ui` restarting on a new ephemeral port"
  - "web/scripts/live-push-multitab-check.mjs: a real 3-tab Playwright gate proving criterion 2's per-message timing paired with a per-tab received-generation superset (never a vacuous single-receipt bound), criterion 3's fan-out isolation with a PROVEN-blocked tab, and a jittered reconnect against the stable proxy origin"
  - "corpora/live-push-multitab-check.json: the committed diagnostic record — 3 tabs, 3 triggered generations all received by all tabs, a genuinely blocked tab with 0 receipts during a proven 1800ms block, and a real reconnect with growing client-scheduled delays"
affects: [06-07]

actuals:
  tokens: 9462
  tasks: 2
  commits: 2
  plan_head_before: 81bbc146b8e30d1d415f18d314dc09f0cb413a3c

tech-stack:
  added: []
  patterns:
    - "A test-only fixed-port loopback proxy in front of a production process that deliberately has no bind/port flag, rewriting Host/Origin to the SAME spelling as the real upstream so a security guard (originHostGuard's membership+pairing check) is exercised exactly as a real same-origin browser request would, never bypassed"
    - "Real main-thread blocking via a synchronous busy loop inside page.evaluate(), never awaited until after triggering concurrent work on sibling pages, to PROVE isolation rather than assume it — the blocked tab's own performance.now() start/end pair is the single clock used to judge which of its own receipts fall inside the block window, avoiding any Node-vs-browser clock skew"
    - "Extracting a TS module's pinned numeric constants via source-file regex from a plain Node script (no TS loader) rather than hard-coding a value that could silently drift from the real implementation — same discipline as 06-05's corpusDir() mirroring Go's manifest logic in JS"

key-files:
  created:
    - web/scripts/live-push-stable-proxy.mjs
    - web/scripts/live-push-multitab-check.mjs
    - corpora/live-push-multitab-check.json

key-decisions:
  - "Real re-indexes against THIS repository's own already-indexed .codegraph/ store are produced by creating small, disposable, syntactically valid Go source files under a throwaway internal/livepushprobe/ package and running a real `codegraph sync` — the same disposable-probe-file technique 06-05 used against the guava corpus, adapted to this repo's own language (Go) since the plan's own precondition names this repository's index as sufficient. Every probe file (and the now-empty directory) is removed, and one final `codegraph sync` is run, in the script's own finally block; `corpusCleanAtExit` is asserted via `git status --porcelain -- internal/livepushprobe` exactly as graph-live-update-check.mjs asserts it for its own corpus checkout."
  - "`codegraph ui` is started with `CODEGRAPH_DEBOUNCE_MS=100` (an already-established test-only env knob used throughout this repo's own Go test suite — internal/daemon, internal/watch, internal/uiserver all set it) so three real, deliberately-spaced re-indexes (600ms apart) never coalesce into fewer than three generation events, while still exercising the real fsnotify-watch-then-Meta-compare pipeline end to end."
  - "The client's own backoff constants (LIVE_BACKOFF_BASE_MS, LIVE_BACKOFF_JITTER_SPREAD) are read out of web/src/lib/live/live-client.ts by regex rather than hard-coded, so minDowntimeMs (4x the base delay) and the record's own jitterSpread stay correct if the pinned shape is ever retuned."
  - "The currently-built `codegraph` binary on disk already embeds the full live-push frontend (06-01 through 06-05's changes) even though the COMMITTED web/build/ predates them — verified directly by fetching the served bundle and finding `watchGraph`/`codegraphLiveObservations` in it before writing a single line of this plan's code. No rebuild of any kind was needed or performed for this plan; `git status --porcelain -- web/build` stays empty at every commit, and 06-07's single rebuild is entirely untouched."
  - "Deviation (Rule 1 - bug, found via a standalone repro before it could silently break this plan's own RECONNECT scenario): the Task 1 proxy did not detect a mid-stream upstream failure. Node's http client surfaces a killed process's socket failure on the RESPONSE object (aborted/error/close-without-end) once headers have already been forwarded — not on the outbound ClientRequest, which only sees connection-time failures. A fetch through the proxy hung forever across a real SIGKILL until `upstreamRes` itself was given error/aborted/close-without-`complete` handlers that call the same `failUpstream()` path the outbound-request error handler already used. Fixed in the SAME commit as Task 2, since Task 2's own real-browser run is what would have surfaced this as an unexplained 30s timeout with no diagnosis."

patterns-established:
  - "Standalone Node repro scripts (fetch through the proxy against a throwaway http.createServer, killed mid-stream) as the fast, real-browser-free way to validate a reverse-proxy's failure-propagation behavior before trusting a 90-second Playwright run to surface (or hide) the same bug."

requirements-completed: [LIV-03]

coverage:
  - id: D1
    description: "The fixed-port loopback proxy rewrites both Host and Origin to the upstream's own authority (proven by observation against a throwaway upstream, not by reading the proxy's own source) and streams the response rather than buffering it"
    requirement: LIV-03
    verification:
      - kind: other
        ref: "web/scripts/live-push-stable-proxy.mjs's own <verify> (Task 1's embedded node -e check)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Three or more real browser tabs each receive every triggered generation, proven by a per-tab received-generation superset check, with inter-arrival timing between triggered receipts bounded by the real trigger spacing — never a bound over a single receipt"
    requirement: LIV-03
    verification:
      - kind: e2e
        ref: "corpora/live-push-multitab-check.json (triggeredGenerations [2,3,4], receivedGenerationsPerTab all supersets, minInterArrivalMs 984.1ms >= triggerSpacingMs 600ms)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A deliberately slow tab is PROVEN blocked (zero receipts recorded during its own measured block window) while the other two tabs keep receiving every triggered generation on schedule, and the recovered tab catches up to the newest generation"
    requirement: LIV-03
    verification:
      - kind: e2e
        ref: "corpora/live-push-multitab-check.json (blockedTabBlockedMs 1800ms, blockedTabReceiptsDuringBlock 0, healthyTabsIsolationSupersetOK true, slowTabFinalGeneration === newestGeneration === 6)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A dropped connection reconnects with a growing, jittered, client-scheduled delay against a stable origin that survives the upstream process being killed and replaced, and every tab applies at least one event after the restart despite the restarted publisher's generation counter beginning again at 1"
    requirement: LIV-03
    verification:
      - kind: e2e
        ref: "corpora/live-push-multitab-check.json (reconnectAttemptsPerTab [3,3,3], reconnectDelaysPerTab strictly increasing per tab and pairwise distinct at attempt 1, postReconnectAppliedPerTab [1,1,1], resumeCursorSentPerTab [6,6,6])"
        status: pass
      - kind: other
        ref: "plan <verify>'s embedded node -e check run directly against the committed record"
        status: pass
    human_judgment: false
  - id: D5
    description: "The record is written on failure too: with the proxy's upstream deliberately never repointed after the restart, the reconnect wait times out and the script writes success:false with a stated reason"
    requirement: LIV-03
    verification:
      - kind: other
        ref: "one-off --red-skip-repoint invocation, observed this session (see Deviations/Verification below)"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 6: The Fixed-Port Proxy And The Three-Tab Browser Gate Summary

**A stable-origin loopback reverse proxy in front of a restartable `codegraph ui`, and a real 3-tab Playwright gate proving criterion 2's per-message timing (superset-paired, never vacuous) and criterion 3's fan-out isolation and jittered reconnect — all against this repository's own live index, with zero frontend rebuild needed.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-09-07T19:30:00Z (approx, first commit timestamp minus setup)
- **Completed:** 2026-09-07T19:52:42Z
- **Tasks:** 2 (both `type="auto"`)
- **Files modified:** 3 (3 created: the proxy, the check script, the committed record)

## Accomplishments

- **Task 1 — `web/scripts/live-push-stable-proxy.mjs`:** `startProxy({port})` returns `{url, setUpstream, close}` — a loopback reverse proxy bound once on an OS-assigned port, forwarding to a mutable upstream. Every forwarded request has its `Host` AND `Origin` headers rewritten to the upstream's own authority (the SAME spelling for both, satisfying `originHostGuard`'s membership-plus-pairing check at `internal/uiserver/originguard.go:42-67`), and the response is piped, never buffered. Verified by its own embedded gate against a throwaway upstream that records what it actually received: `upstreamSawHost: "127.0.0.1:<upstream-port>"` (never the proxy's own authority), `upstreamSawOrigin: "http://127.0.0.1:<upstream-port>"` (correctly paired), `chunkCount: 2`, `firstToLastMs: 148` (the two deliberately-spaced chunks arrived ~150ms apart, proving the proxy streams rather than accumulates). Introduces zero dependency — `node:http`/`node:url` only — and `git diff --stat -- web/package.json web/pnpm-lock.yaml` stays empty.
- **Task 2 — `web/scripts/live-push-multitab-check.mjs`:** Opens 3 real Chromium tabs on the proxy's own origin, each navigating to `/health` — no special "subscribe" action needed, since `+layout.svelte` constructs the live client and store unconditionally at mount. Three scenarios, each against this repository's OWN already-indexed `.codegraph/` store:
  - **FAN-OUT AND TIMING:** 3 real re-indexes (via disposable `internal/livepushprobe/*.go` probe files + a real `codegraph sync`), each triggered only after every tab confirmed receipt of the previous one, spaced 600ms apart. Result: `triggeredGenerations: [2, 3, 4]`, every tab's `receivedGenerationsPerTab` a superset of that set, `minInterArrivalMs: 984.1ms >= triggerSpacingMs (600ms)` — the superset check is what makes the inter-arrival bound non-vacuous (a single receipt per tab would trivially satisfy an inter-arrival minimum with no consecutive pair to bound).
  - **FAN-OUT ISOLATION:** tab 2's main thread genuinely blocked for 1800ms via a real synchronous busy loop inside `page.evaluate()` (never awaited until after the isolation-phase triggers are issued), while 2 more real re-indexes fire and are confirmed received by the two healthy tabs. Result: `blockedTabBlockedMs: 1800`, `blockedTabReceiptsDuringBlock: 0` (the tab was PROVEN blocked, not merely believed to be — computed entirely from the blocked page's own `performance.now()` clock, never cross-referenced against Node's), `healthyTabsIsolationSupersetOK: true`, and the recovered blocked tab reaches `slowTabFinalGeneration: 6 === newestGeneration: 6`. Per this plan's own explicit scope, this scenario does NOT claim to prove server-side coalescing (a browser cannot manufacture transport pressure at re-index pacing) — `coalescingProvenBy` names `06-04`'s `TestWatchGraphHandlerCoalescesForANonReadingClient` as the actual proof of that property.
  - **RECONNECT:** the real `codegraph ui` process is SIGKILLed, the upstream stays down for `minDowntimeMs` (4x the client's own `LIVE_BACKOFF_BASE_MS`, extracted from `live-client.ts` by regex rather than hard-coded) plus a real margin, then a NEW `codegraph ui` process is started and the proxy is repointed — the tabs' own origin never moves. Result: `reconnectAttemptsPerTab: [3, 3, 3]` (bounded above at 6, below at 2), `reconnectDelaysPerTab` strictly increasing per tab and pairwise distinct at the first attempt, `reconnectBaseDelaysPerTab` consistent with the scheduled delays within the pinned jitter spread, `postReconnectAppliedPerTab: [1, 1, 1]` (every tab applied at least one event after the restart — the live proof of 06-03's connection-epoch rule against a publisher whose generation counter restarted at 1), `resumeCursorSentPerTab: [6, 6, 6]` (every tab resumed from what it had actually seen).

  The plan's own embedded `<verify>` command (the fixed `node ... ; node -e '...'` two-step) was run directly against the committed record and printed `PASS` with **zero spurious failures**, exactly as specified.

## Task Commits

1. **Task 1: the fixed-port loopback proxy** — `bbdd471e` (feat)
2. **Task 2: three real tabs — timing, isolation, reconnect** (includes the Rule 1 proxy fix found while validating Task 2) — `9ad903e` (feat)

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `web/scripts/live-push-stable-proxy.mjs` — new: `startProxy`, the fixed-origin reverse proxy, Host/Origin rewrite, streaming pass-through, and (added during Task 2's own validation) response-object failure propagation for a mid-stream upstream kill.
- `web/scripts/live-push-multitab-check.mjs` — new: the 3-tab real-browser gate covering all three scenarios above, with a `--red-skip-repoint` diagnostic flag for the RED-path reproduction (never used for the committed record).
- `corpora/live-push-multitab-check.json` — new: the committed diagnostic record from a real, passing run.

## Decisions Made

See `key-decisions` in frontmatter. Highlights:
- No frontend rebuild was needed: the currently-built `codegraph` binary already embeds all of 06-01 through 06-05's changes (verified directly by fetching the served bundle and grepping for `watchGraph`/`codegraphLiveObservations` before writing any code), even though the committed `web/build/` on disk predates them. This plan's own scope forbids rebuilding the bundle regardless (06-07's job); this verification confirmed that restriction cost nothing.
- Disposable Go probe files (this repo's own language) replace 06-05's disposable Java files as the mechanism for producing real, `Discover()`-visible re-indexes against a live `.codegraph/` store — same technique, different target corpus (this repository itself, per the plan's own precondition).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The Task 1 proxy did not propagate a mid-stream upstream failure to the client**
- **Found during:** Task 2's own real-browser RECONNECT scenario — the first full run of the multitab check hung at the reconnect wait step and timed out at 30s with no other symptom.
- **Issue:** Isolated via a standalone Node repro (fetch a long-lived stream through the proxy, then `up.close()` the upstream mid-stream): the client-side fetch's `ReadableStream` never signalled `done` or threw — it simply stalled forever. Node's `http.request`'s outbound `ClientRequest` only emits `'error'` for CONNECTION-time failures; once headers have already been received and forwarded, a killed process's socket failure surfaces on the RESPONSE object instead (`'aborted'`, `'error'`, or a `'close'` that never saw `'end'`). The proxy had error handling only on the outbound request, so a real `SIGKILL` mid-stream (exactly the RECONNECT scenario's own shape) left the client's fetch hanging with no way to notice the upstream was gone — silently defeating the entire scenario, since the client's own reconnect/backoff logic never engaged.
- **Fix:** Added `upstreamRes.on('error', ...)`, `upstreamRes.on('aborted', ...)`, and `upstreamRes.on('close', ...)` (guarded by `!upstreamRes.complete`) that all route through the SAME `failUpstream()` path the outbound-request error handler already used — deduplicated with a `failed` flag so a genuine race between multiple signals never double-writes the response.
- **Files modified:** `web/scripts/live-push-stable-proxy.mjs`
- **Verification:** The standalone repro (`fetch` through the proxy, kill the upstream mid-stream) went from `ended? false` (hang) to `STREAM ERROR terminated` / `ended? true`. Task 1's own embedded `<verify>` gate re-run and still PASSES (host/origin rewrite and streaming-not-buffering both hold). The full multitab check subsequently ran to `success: true` twice in direct succession.
- **Committed in:** `9ad903e` (Task 2 commit — the bug was found validating Task 2 and would have silently broken this plan's own RECONNECT scenario, so it is committed alongside the code that surfaced it, not as a separate retroactive fix to Task 1's already-landed commit).

---

**Total deviations:** 1 auto-fixed (Rule 1 — a bug in this plan's own harness code, found and fixed before it could produce a false negative in this plan's own gate). **Impact:** Essential — without this fix, the RECONNECT scenario cannot ever pass, and the failure mode (a silent 30s hang with no diagnostic) is exactly the kind of unexplained timeout this plan's own header comment on the Task 1 proxy warns against. No scope creep: the fix touches only the proxy's failure-propagation path.

## Issues Encountered

None beyond the one deviation above, resolved within this plan, before any code was committed under a false-passing state.

## RED-path reproduction (per this task's own instruction, acceptance criteria)

Run once, deliberately, with `--red-skip-repoint` (leaving the proxy pointed at the now-dead OLD upstream after the restart — never used for the committed record):

```
live-push-multitab-check: wrote /tmp/live-push-multitab-check-RED.json — success=false
error: "live-push-multitab-check: pollUntil: condition did not become true within 30000ms"
triggeredGenerations: [2, 3, 4]   (the FAN-OUT/TIMING and ISOLATION scenarios still completed normally — they run before the restart)
reconnectAttemptsPerTab: []       (never populated — the reconnect wait never resolved)
corpusCleanAtExit: true           (cleanup still ran to completion via the finally block)
```

This confirms the record is written on failure too, with a stated reason, and confirms the RECONNECT scenario genuinely exercises the proxy's stable-origin property rather than passing vacuously — a harness that could never fail this way would not be proving anything.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `node web/scripts/live-push-multitab-check.mjs` — PASS, `success: true`; run twice in direct succession with consistent results (`corpora/live-push-multitab-check.json` committed from the second run).
- Task 1's proxy gate (embedded `<verify>`) — PASS, re-run after the Rule 1 fix: `upstreamSawHost`/`upstreamSawOrigin` correctly rewritten and paired, `chunkCount: 2`, `firstToLastMs: 148ms`.
- `git status --porcelain -- web/build` — empty at every commit in this plan.
- `git diff --stat -- web/package.json web/pnpm-lock.yaml` — empty.
- No stray `codegraph` processes and no leftover probe files after any run (`ps aux | grep codegraph` empty; `git status --porcelain` clean beyond this plan's own three intended files, both before and after the RED-path reproduction run).

## Known Stubs

None. Both scripts are fully implemented and were exercised end-to-end against a real `codegraph ui` process (started, killed, and restarted for real), a real Chromium browser, and this repository's own real `.codegraph/` index.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Criterion 2's browser-side timing proof and criterion 3's fan-out/isolation/reconnect proof are both closed, against a harness that has been shown capable of failing (the RED-path reproduction) as well as passing. `06-07`'s real-process concurrency gate (criterion 5), the `pendingWriter`-analogue verdict recording, and the phase's single `web/build` rebuild are unaffected by anything in this plan and remain that plan's own subject. No blockers.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- All 3 files (web/scripts/live-push-stable-proxy.mjs, web/scripts/live-push-multitab-check.mjs, corpora/live-push-multitab-check.json) — FOUND on disk.
- Commits `bbdd471e`, `9ad903e` — both FOUND in `git log --oneline --all`.
- Plan-level `<verification>` re-run live: `node web/scripts/live-push-multitab-check.mjs` PASS (`success: true`, run twice), Task 1's embedded proxy gate PASS, `git status --porcelain -- web/build` empty, `git diff --stat -- web/package.json web/pnpm-lock.yaml` empty.
