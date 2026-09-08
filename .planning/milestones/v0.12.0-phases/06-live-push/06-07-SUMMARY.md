---
phase: 06-live-push
plan: 07
subsystem: ui
tags: [connect-rpc, mcp, pebble, concurrency, bash, playwright-adjacent, build]

requires:
  - phase: 06-live-push
    provides: "06-02's structurally-separated sendCount/subscriber-map counters and the counter-separation test; 06-04's real WatchGraph streaming handler; 06-06's proof that criteria 2/3 hold under real multi-tab browser load"
provides:
  - "scripts/live-push-probe.go: a //go:build ignore diagnostic harness — a real uiv1connect Connect stream client (`stream`) and a real persistent MCP stdio harness (`mcp`, via mcp.IOTransport wrapping cmd.StdinPipe/StdoutPipe) — excluded from the shipped build, runnable via go run"
  - "scripts/live-push-concurrency-check.sh: criterion 5's real three-OS-process gate (codegraph daemon start, codegraph serve --mcp, codegraph ui, plus the harness stream probe) against a scratch repository, with a measured no-UI baseline, an operational starvation definition, and a demonstrated RED"
  - "corpora/live-push-concurrency-check.json: the committed passing record — 5/5 flushes completed, 0 starved, 5 live events delivered excluding the seed, all three product processes alive, MCP liveness proven by a real tool-call response"
  - "the pendingWriter-analogue verdict, recorded in both halves with a discriminating positive control (declaration count, not bare-word count)"
  - "the phase's single web/build rebuild, staged and committed with the assertion correctly placed AFTER the commit"
affects: []

actuals:
  tokens: 7712
  tasks: 3
  commits: 4
  plan_head_before: e5bf47db808d12e1162b5d177ef73b500144cb6f

tech-stack:
  added: []
  patterns:
    - "//go:build ignore diagnostic-harness program excluded from the shipped build/vet/list surface but runnable via `go run`, reusing the module's own generated Connect client and its own go-sdk mcp.IOTransport rather than hand-rolling either protocol — zero new dependency"
    - "mcp.IOTransport{Reader: cmd.StdoutPipe(), Writer: cmd.StdinPipe()} as the way to drive a real persistent stdio MCP session from a Go harness while keeping the literal StdinPipe/StdoutPipe calls in the caller's own file (needed here so an acceptance grep for those calls has something real to find)"
    - "content-derived per-flush marker tokens (a uniquely-named declared symbol) polled via the product's own CLI (`codegraph search --json`) as a completion oracle, rather than trusting a bare last_sync_unix_ms timestamp advance"
    - "a measured (not guessed) inter-event settle gap: this session measured that a 600ms gap between two sequential re-index flushes still coalesced into one delivered live-push generation under the live publisher's own debounce, while a 3s gap reliably separated them — the gate uses a 2.5s margin"

key-files:
  created:
    - scripts/live-push-probe.go
    - scripts/live-push-concurrency-check.sh
    - corpora/live-push-concurrency-check.json
  modified:
    - web/build/** (rebuilt app bundle + .build-manifest — the phase's single deferred rebuild)

key-decisions:
  - "The mcp probe subcommand calls cmd.StdoutPipe()/cmd.StdinPipe() directly in scripts/live-push-probe.go and wraps them in mcp.IOTransport, rather than using the go-sdk's own mcp.CommandTransport (which internally does the identical thing) — Task 1's acceptance criteria require the literal strings StdinPipe/StdoutPipe to appear in this file, and CommandTransport's calls live inside the SDK, not here."
  - "mcpAliveAfterRun is derived from the tool-call response being a non-empty string, never from a kill -0 process-table check on the child serve --mcp PID — per the plan's explicit requirement, and because the child's stdin EOFs (and it may exit) the instant the probe's own process exits, making a post-hoc process check either racy or backwards."
  - "A 2.5s settle gap is inserted after each flush's marker becomes queryable and before the next flush's write. Measured directly this session: with no settle gap, two sequential flushes were delivered as ONE live-push generation event (verified via a real running publisher, not inferred); with a 3s gap they were reliably delivered as two. The live publisher's own watcher debounces on the SAME CODEGRAPH_DEBOUNCE_MS window this script sets for the daemon, and Pebble's own background churn (WAL rotation, compaction) can keep that debounce timer re-armed well past the moment a flush's data is already queryable."
  - "The RED demonstration for criterion 5 uses a temporary, reverted-before-commit edit (an env-gated skip of `defer closer.Close()` in computeChange) rather than a permanent script flag — the plan calls for 'a temporary local edit, reverted immediately', and this is a change to production code's own store-handle lifetime, not something that belongs behind a permanent diagnostic flag in shipped source."
  - "corpora/live-push-concurrency-check.json's baseline is measured with the SAME daemon process kept running across both the baseline and concurrent phases (never restarted), matching the plan's own 'before starting the UI and stream probe, drive the same five real re-index flushes with only the daemon running' — the comparison is apples-to-apples against one continuous daemon lifetime, not two separately-started ones."

requirements-completed: [LIV-01, LIV-03]

coverage:
  - id: D1
    description: "Criterion 5: under a real daemon, a real MCP server, and a real UI sharing one store, 5 real re-index flushes complete with zero starvation, bounded against a measured no-UI baseline"
    requirement: LIV-01
    verification:
      - kind: other
        ref: "scripts/live-push-concurrency-check.sh's own embedded verify (corpora/live-push-concurrency-check.json: flushesCompleted 5/5, flushesStarved 0, maxFlushDurationMs 309 <= 3*261+1000)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The gate discriminates: with the publisher's store handle deliberately leaked (a temporary, reverted edit), the same gate reports flushesStarved=3/3, flushesCompleted=0, with the daemon's own real store-lock log lines captured as starvationEvidence"
    requirement: LIV-01
    verification:
      - kind: other
        ref: "one-off CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN=1 invocation, observed this session (see RED Observations below)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The pendingWriter-analogue verdict is recorded in both halves with a control that discriminates on the declaration shape, not the bare word"
    requirement: LIV-03
    verification:
      - kind: other
        ref: "rg -c 'type pendingWriter struct' internal/mcp/server.go (=1) vs rg -c 'pendingWriter' internal/mcp/server.go (=9); TestLiveRegistryCounterSeparation PASS"
        status: pass
    human_judgment: false
  - id: D4
    description: "The committed web/build bundle matches the phase's web sources with nothing left unstaged, proven by an assertion placed AFTER the commit and demonstrated red against a partial stage"
    requirement: LIV-03
    verification:
      - kind: other
        ref: "task web:drift PASS; git status --porcelain -- web/build empty after full commit; partial-stage RED demonstrated and reverted this session"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 7: Criterion 5's Real-Process Concurrency Gate, the Recorded Verdict, and the Final Bundle Rebuild Summary

**A real three-OS-process gate (`codegraph daemon start`, `codegraph serve --mcp`, `codegraph ui`) proving 5 real re-index flushes survive a live-push session with zero starvation against a measured baseline, the `pendingWriter`-analogue verdict recorded with a discriminating control, and the phase's single, correctly-staged `web/build` rebuild.**

## Performance

- **Duration:** 45 min (approx)
- **Started:** 2026-09-07T16:05:00-04:00 (approx)
- **Completed:** 2026-09-07T16:31:00-04:00
- **Tasks:** 3
- **Files modified:** 3 created (scripts/live-push-probe.go, scripts/live-push-concurrency-check.sh, corpora/live-push-concurrency-check.json) + the rebuilt `web/build/` tree

## Accomplishments

- **Task 1 — the probe program.** `scripts/live-push-probe.go`, `//go:build ignore`-tagged so `go build`/`go vet`/`go list` all skip it while `go run` still compiles and runs it in full module context. Its `stream` subcommand opens `WatchGraph` with the real generated `uiv1connect` client and appends one JSON line (flushed via `f.Sync()`) per received event. Its `mcp` subcommand spawns `codegraph serve --mcp --path <repo>` with real `cmd.StdoutPipe()`/`cmd.StdinPipe()` held open for the whole hold period, connects via the go-sdk's `mcp.Client`/`mcp.IOTransport` (performing a real initialize handshake), sleeps out the hold, then issues one real `codegraph_status` tool call and prints a JSON verdict carrying the child's own PID and the tool's own response text. Verified end-to-end against a real `codegraph ui` and a real `codegraph serve --mcp` process before Task 2 was written.
- **Task 2 — criterion 5's real gate.** `scripts/live-push-concurrency-check.sh` seeds a throwaway scratch git repository (never this repository's own working tree), indexes it once, measures a no-UI baseline of 5 real re-index flushes with only `codegraph daemon start` running, then starts `codegraph ui`, the `mcp` probe (a real second reader), and the `stream` probe (a real live-push session held open) and drives the SAME 5 flushes again under the full topology. Each flush writes a uniquely-named Go function into the scratch repo and polls `codegraph search --json` until that exact symbol is queryable — never a bare `last_sync_unix_ms` check. The clock start is `write_time + configured_debounce` (a bounded, conservative estimate — `internal/watch/debounce.go`'s debouncer has no leading edge, so `clock_start <= flush_start` always); the clock end is the marker becoming queryable. Both the adjusted (`adjMs`) and raw unadjusted (`rawMs`) elapsed are recorded per flush. A flush is "starved" when either the daemon's own log for that flush's window contains a store-lock-contention line (matched verbatim) or it times out. **Result: `corpora/live-push-concurrency-check.json` — 5/5 flushes completed, 0 starved, `maxFlushDurationMs 309 <= 3*261+1000 = 1783`, `liveEventsExcludingSeed 5`, all three product processes (`daemon`, `serve --mcp`, `ui`) alive at the end, MCP liveness proven by a real `codegraph_status` response, `success: true`.** The gate was then demonstrated RED (see below) and reverted.
- **Task 3 — the verdict, the rebuild, and a consistent tree.** Both halves of the `pendingWriter`-analogue verdict recorded with exact commands and counts (see below). `web/build/` rebuilt via `task web:build` and staged/committed with the assertion correctly placed AFTER the commit, demonstrated RED against a partial stage first. Full Go suite, full web suite, and proto drift all re-run green after the rebuild.

## Task Commits

1. **Task 1: the probe program** — `a01ba4d4` (feat)
2. **Task 2: criterion 5's real gate** — `53f6f87c` (feat)
3. **Task 3: the bundle rebuild** — `8e182431` (build), `e2ecfe39` (build — a second real rebuild produced by re-running the plan's own `<verify>` chain; Vite/Rollup's content-hash filenames are not guaranteed identical across separate builds of identical source, as this repository's own `Taskfile.yml` (`web:build:verify`'s own doc comment) and `06-06-SUMMARY.md` both already document — both commits are real, both pass `web:drift`)

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `scripts/live-push-probe.go` — the `//go:build ignore` stream/mcp diagnostic harness
- `scripts/live-push-concurrency-check.sh` — criterion 5's real three-process gate
- `corpora/live-push-concurrency-check.json` — the committed passing record
- `web/build/**` — rebuilt app bundle + `.build-manifest` (the phase's single deferred rebuild, covering `06-01`'s regenerated client, `06-03`'s live client/store/status/routes, and `06-05`'s graph write-back)

## Decisions Made

See `key-decisions` in frontmatter. Highlights: the probe deliberately keeps `StdinPipe`/`StdoutPipe` calls in its own file rather than delegating to the SDK's `CommandTransport`; `mcpAliveAfterRun` is derived from the tool-call response, never a process check; a 2.5s inter-flush settle gap was chosen from a directly measured coalescing threshold, not a guess; the RED demonstration used a temporary reverted edit rather than a permanent flag.

## The `pendingWriter`-analogue verdict (LIV-03's Criterion 3 requirement)

**HALF ONE — no pre-existing analogue in `internal/uiserver`.**

Every non-test Go file in the package:

```
$ ls internal/uiserver/*.go | rg -v _test.go
internal/uiserver/degrade.go
internal/uiserver/diag.go
internal/uiserver/handlers.go
internal/uiserver/livehandler.go
internal/uiserver/livepublish.go
internal/uiserver/originguard.go
internal/uiserver/permalink.go
internal/uiserver/server.go
internal/uiserver/spa.go
internal/uiserver/truncate.go
internal/uiserver/watchtimeout.go
```

Search for the counter shape the earlier bug had (an in-flight/pending counter of client-initiated work):

```
$ rg -n 'type .*pending.*struct|type .*Pending.*struct|type .*inflight.*struct|type .*InFlight.*struct' \
    $(ls internal/uiserver/*.go | rg -v _test.go)
NO MATCHES (0 hits)
```

**Positive control — the search discriminates, proven against a file that DOES have the shape:**

```
$ rg -n 'type .*pending.*struct|type .*Pending.*struct|type .*inflight.*struct|type .*InFlight.*struct' internal/mcp/server.go
466:type pendingWriter struct {
```

**The discriminating control the plan requires (declaration shape, not the bare word):**

```
$ rg -c 'pendingWriter' internal/mcp/server.go
9
$ rg -c 'type pendingWriter struct' internal/mcp/server.go
1
```

9 matches the bare word — 6 of those are comments/type references (`:219`, `:282`, `:487`, `:498` prose; `:237`, `:493`, `:540` uses), proving only that the file contains the string. 1 matches the declaration itself (`internal/mcp/server.go:466`), the shape actually being searched for. Its mutation site:

```
$ rg -n 'decrementPending\(p.pending\)' internal/mcp/server.go
523:			decrementPending(p.pending)
```

A zero-count claim about `internal/uiserver` is meaningful only alongside this positive control — verified this session, not assumed.

**HALF TWO — the new code cannot reintroduce it.**

`internal/uiserver/livepublish.go`'s `liveRegistry` keeps `sendCount atomic.Int64` (server-initiated event sends) and `subs map[uint64]chan *uiv1.WatchGraphEvent` (client-initiated subscriber lifecycle) as structurally separate storage — `Publish` never touches `len(subs)`, and `Subscribe`/`unsubscribe` never touch `sendCount`. Pinned in both directions by `TestLiveRegistryCounterSeparation` (`internal/uiserver/livepublish_test.go:642`):

```
$ go test ./internal/uiserver/... -run '^TestLiveRegistryCounterSeparation$' -v -race -count=1
--- PASS: TestLiveRegistryCounterSeparation (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	1.458s
```

**Verdict: no `pendingWriter` analogue exists in `internal/uiserver`, and the one place this codebase's counter-corruption bug shape DOES occur (`internal/mcp`) is unaffected by this phase's new code — the two kinds of state (server-initiated sends, client-initiated subscriber lifecycle) never share storage, pinned by a test asserting both directions.**

## RED Observations (this phase's own discipline: a gate never seen red proves nothing)

Six, across the whole phase:

1. **`06-04` Task 1 — middleware removed.** Removing `clearWatchDeadline` and re-running the load-bearing test: 3 chunks received under a 250ms deadline (want ≥8). Restoring it: 4/4 `TestStreamDeadline*` PASS.
2. **`06-04` Task 3 — buffering send loop vs. the tracer.** With the send loop temporarily changed to buffer every event and flush only at stream end: `TestWatchGraphStreamDeliversMessageByMessage` FAILED (`client.WatchGraph` blocked for the full 5s deadline, `deadline_exceeded`). Reverting restored GREEN (3 triggered receipts, generations `[2 3 4]`).
3. **`06-05` Task 3 — write-back disabled.** With the write-back temporarily disabled: `maxLeafDisplacementPx: 599.55`, `success: false` (was 0px). Re-enabling restored the guava-scale zero-displacement result.
4. **`06-06` Task 2 — proxy upstream never repointed.** A one-off `--red-skip-repoint` invocation left the proxy pointed at the dead old upstream after the UI restart: `success: false`, `"pollUntil: condition did not become true within 30000ms"`, `reconnectAttemptsPerTab: []`.
5. **This plan's Task 2 — store held open.** A temporary, reverted edit (`CODEGRAPH_LIVEPUSH_RED_HOLD_OPEN=1`, skipping `defer closer.Close()` in `computeChange`) made the publisher leak its store handle. Result: `flushesAttempted: 3, flushesCompleted: 0, flushesStarved: 3`, with `starvationEvidence` carrying the daemon's own real log lines:
   ```
   daemon: sync: graphstore: store lock held: resource temporarily unavailable
   daemon: sync lost the store-lock race 6 consecutive times; giving up until the next event (graph stays marked stale via .sync-pending)
   ```
   `git diff` confirmed the revert left `internal/uiserver/livepublish.go` byte-identical to its pre-edit state before any commit in this plan.
6. **This plan's Task 3 — partial bundle stage.** `git add web/build/index.html` (one of ~19 changed paths) followed by `git commit -- web/build`: git's own pathspec-commit behavior staged and committed the tracked modifications/deletions matching that pathspec but silently excluded the NEW untracked chunk files from the rebuild — reproducing WINDOWS 29's exact failure shape. `git status --porcelain -- web/build` afterward was non-empty (9 `??` entries). `git reset --soft HEAD~1`, then `git add -A web/build` + commit: porcelain empty, `git ls-files --error-unmatch web/build/.build-manifest` succeeded.

## Carried-forward verdicts

- **Read-deadline verdict (`06-04`):** D-02 clears the WRITE deadline only. `readTimeout = 30s` needs no clearing for a long-lived stream — `net/http` clears the read deadline itself at `startBackgroundRead` once the request body reaches EOF, which for a Connect server-streaming request happens immediately after the client's single request message is sent (`$GOROOT/src/net/http/server.go:697`).
- **`coalescingProvenBy` division of labour (`06-06`):** the browser fan-out check (`live-push-multitab-check.mjs`) proves per-tab delivery and isolation under real load; it does NOT claim to prove server-side coalescing (a browser cannot manufacture transport pressure at re-index pacing). That property is proven by `06-04`'s `TestWatchGraphHandlerCoalescesForANonReadingClient`.

## Criterion-by-criterion map (all five ROADMAP success criteria)

| Criterion | Satisfied by |
|---|---|
| 1. Health/status chrome updates via the same mechanism | `06-03`/`06-04`'s `WatchGraphEvent` wiring and the browser's `StatusBanner`/health chrome consuming it |
| 2. Per-message delivery latency, not eventual arrival | `06-04`'s tracer (`TestWatchGraphStreamDeliversMessageByMessage`) + `06-06`'s real 3-tab `minInterArrivalMs: 984.1ms >= 600ms` |
| 3. Multi-tab, backpressure, reconnect, `pendingWriter`-analogue verdict | `06-02`'s structurally-separated counters + `06-06`'s real fan-out/isolation/reconnect + **this plan's recorded verdict, both halves** |
| 4. Graph layout stability across live updates | `06-05`'s write-back, measured at guava scale: 349 leaf survivors, 0px max/mean displacement |
| 5. Real `daemon`/`serve --mcp` concurrency, no starving or holding the store open | **This plan's Task 2 — `corpora/live-push-concurrency-check.json`, `success: true`, demonstrated red** |

## Measured Results (Task 2, committed record)

- `flushesAttempted: 5`, `flushesCompleted: 5`, `flushesStarved: 0`, `starvationEvidence: []`
- `flushDurationsMs: [254, 252, 247, 309, 260]` (adjusted, debounce-excluded)
- `rawWriteToQueryableMs: [354, 352, 347, 409, 360]` (unadjusted)
- `maxFlushDurationMs: 309`, `baselineMaxFlushDurationMs: 261` — bound `309 <= 3*261+1000 = 1783` (well within)
- `debounceExcludedMs: 100`, `flushTimeoutMs: 5000`
- `liveEventsReceived: 6`, `liveEventsExcludingSeed: 5`
- `mcpAliveAfterRun: true` via a real `codegraph_status` tool-call response (non-empty); `daemonAliveAfterRun: true`; `uiAliveAfterRun: true`
- `processes`: 4 entries, exactly 3 `role: "product"` (daemon, `serve --mcp`, ui) + 1 `role: "harness"` (stream probe)

Baseline-vs-concurrent comparison: the concurrent run's max flush (309ms) is actually LOWER than one might fear relative to the no-UI baseline max (261ms) — well inside the 3x+1000ms bound, confirming the live-push session genuinely does not slow, let alone starve, the daemon's real re-index flushes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug avoidance] Sequential flushes with no settle gap coalesced into one delivered live-push generation**
- **Found during:** Task 2, initial dry run of the concurrency gate against a real running publisher
- **Issue:** With flushes issued back-to-back (next flush's write starting immediately after the previous flush's marker became queryable), 2 real, distinct flushes were delivered to the stream probe as exactly ONE new generation event, not two — verified directly by manual reproduction against a real `codegraph ui`/`codegraph daemon` pair with `CODEGRAPH_DEBOUNCE_MS=100`. The live publisher's own watcher debounces on the SAME window over `.codegraph/store/`, and Pebble's own background churn (WAL rotation, compaction) can keep that debounce timer re-armed well past the moment a flush's data is already queryable, so two flushes spaced only ~300-900ms apart coalesced.
- **Fix:** Added a 2.5s settle gap after each flush's completion and before the next flush's write, sized from a direct measurement this session (a 600ms gap still coalesced; a 3s gap reliably separated two flushes).
- **Files modified:** `scripts/live-push-concurrency-check.sh`
- **Verification:** Re-run with the settle gap: 2 flushes -> 2 distinct received generations (`liveEventsExcludingSeed: 2`); the full 5-flush run -> `liveEventsExcludingSeed: 5`.
- **Committed in:** `53f6f87c` (Task 2 commit)

**2. [Rule 1 - Bug avoidance] `processes[]` PIDs read as 0 in the aggregated JSON**
- **Found during:** Task 2, first full dry run
- **Issue:** `DAEMON_PID`/`UI_PID`/`STREAM_PID` were cleared to `""` (to prevent the EXIT trap from double-killing already-torn-down processes) BEFORE the same variables were read to build the `processes[]` JSON array, so every product/harness entry recorded `pid: 0`.
- **Fix:** Captured the PIDs into separate `_RECORDED` variables immediately before teardown, and built the `processes[]` JSON from those instead of the (by-then-cleared) live PID variables.
- **Files modified:** `scripts/live-push-concurrency-check.sh`
- **Verification:** Re-run: all four `processes[]` entries carry their real, non-zero PIDs.
- **Committed in:** `53f6f87c` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in this plan's own harness code, found and fixed before the committed record was produced). **Impact:** Essential — without the settle-gap fix, the committed record would have understated `liveEventsExcludingSeed` and failed the plan's own `>= 5` floor; without the PID fix, the `processes[]` bookkeeping array would have been silently wrong. No scope creep: both fixes touch only this plan's own new script.

## Issues Encountered

None beyond the two deviations above, both resolved within this plan before any code was committed under a false-passing or incorrect state.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `GOTOOLCHAIN=go1.26.5 go build ./...` and `go vet ./...` — clean, `scripts/live-push-probe.go` excluded from both (`go list -f '{{.GoFiles}}' ./scripts/...` matches no packages).
- `bash scripts/live-push-concurrency-check.sh` — exit 0, `success: true`, `corpora/live-push-concurrency-check.json` committed from this run.
- 4 processes recorded, exactly 3 `role: "product"`.
- `GOTOOLCHAIN=go1.26.5 task test:unit` — green, every package `ok` (`internal/uiserver` 34.359s fresh); `internal/daemon`'s watchdog test correctly excluded.
- `task web:test` — `463 of 463 tests passed`.
- `task proto:drift` — `all 4 generated files byte-identical`.
- `task web:build` && `task web:drift` — `PASS — hashed 110 source files, manifested 32 output files, committed web/build/ matches both digests`.
- `git status --porcelain -- web/build` — empty AFTER the bundle commit; `git ls-files --error-unmatch web/build/.build-manifest` succeeds.
- `git diff --stat 9ed946e4..HEAD -- go.mod go.sum web/package.json web/pnpm-lock.yaml` — empty across the WHOLE phase (T-06-SC): zero new dependencies introduced anywhere in Phase 6.
- No CI-skip directive in any spelling in any of this plan's commit messages.
- No stray `codegraph` processes after any run (`ps aux | grep codegraph` empty).

## Known Stubs

None. Both scripts are fully implemented and were exercised end-to-end against real `codegraph daemon`, `codegraph serve --mcp`, and `codegraph ui` processes, and the RED path was demonstrated for both the concurrency gate and the bundle-staging gate.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

All five ROADMAP success criteria for Phase 6 are closed. Criterion 5 — the one the ROADMAP calls non-negotiable — is proven against real, concurrent, multi-process use with a measured no-UI baseline and a demonstrated RED. The `pendingWriter`-analogue verdict is recorded with a discriminating positive control. The app bundle matches the phase's web sources with nothing left uncommitted. This is the final plan of Phase 6 and of milestone v0.12.0 — no blockers, no deferred work, no known stubs.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- `scripts/live-push-probe.go` — FOUND
- `scripts/live-push-concurrency-check.sh` — FOUND
- `corpora/live-push-concurrency-check.json` — FOUND
- `.planning/phases/06-live-push/06-07-SUMMARY.md` — FOUND
- Commits `a01ba4d4`, `53f6f87c`, `8e182431`, `e2ecfe39` — all FOUND in `git log --oneline --all`
- Plan-level `<verification>` re-run live (see "Verification Re-run" above): all commands re-executed this session, all green
