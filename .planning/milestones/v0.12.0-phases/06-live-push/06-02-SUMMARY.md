---
phase: 06-live-push
plan: 02
subsystem: api
tags: [fsnotify, pebble, streaming, watcher, coalescing, uiserver]

requires:
  - phase: 06-live-push
    provides: "06-01's frozen WatchGraph wire surface (WatchGraphRequest/WatchGraphEvent, six-field event set) — this plan's event type"
provides:
  - "changeDetector/computeChange: an open-read-close index-metadata change detector that publishes only when last_sync_unix_ms actually moved, never on Pebble's own compaction/WAL/MANIFEST churn (D-01)"
  - "liveRegistry: a bounded, coalescing subscriber registry — capacity-1 per-subscriber channels, seeded with the publisher's current state at Subscribe time, structurally separate send/subscriber counters (T-06-10)"
  - "livePublisher: the store watcher — a three-element armWatches state machine (repoRoot/.codegraph/store) wired to the detector and registry via a watch.Debouncer, goes live on first index and re-arms after a store-directory replacement"
affects: [06-04, 06-06, 06-07]

actuals:
  tokens: 15510
  tasks: 3
  commits: 5
  plan_head_before: 48a45e232e441e69166d689236d81d513e71d1a8

tech-stack:
  added: []
  patterns:
    - "Package-level func-var test seam for a concrete method: engineStatus = (*query.Engine).Status, the same shape as this package's existing openEngine var, letting a test count invocations of a method with no other interception point"
    - "Registry seed-on-subscribe: Subscribe pre-loads the fresh capacity-1 channel with the registry's current event under the same lock that registers the subscriber, so a brand-new subscriber never waits for the next change to see current state"
    - "Three-way subscription teardown convergence: an explicit unsubscribe call, ctx cancellation, and a registry-wide Stop() all remove/close a subscriber's channel through the same guarded path, using a registry-wide done channel so a per-subscription watcher goroutine can never outlive Stop()"
    - "Per-test goleak.VerifyNone(t, goleak.IgnoreCurrent()) scoping instead of a package-wide TestMain, when the package's OTHER tests hold long-lived background goroutines outside the new code's control"

key-files:
  created: []
  modified:
    - internal/uiserver/livepublish.go
    - internal/uiserver/livepublish_test.go

key-decisions:
  - "engineStatus (a new package-level func var wrapping (*query.Engine).Status) was added as its own test seam, distinct from the pre-existing openEngine var, because T-06-40's scoping test needs to count Status derivations specifically — openEngine's own counter cannot distinguish 'opened but returned early' from 'opened and derived Status'"
  - "query.ErrNotInitialized and any OTHER unclassified openEngine failure are folded into the same not-initialized branch (never a third answer, never an escalated error) — mirroring GetStatus's own degradedStatus behavior for its degradeNone case, and satisfying D-01's 'never crash the publisher' mandate for a class of failure the plan did not enumerate by name"
  - "checkAndPublish (livePublisher) adds a SECOND mutex beyond changeDetector's own internal one, serializing 'compute change then publish' as one atomic step — internal/watch.Debouncer's own doc comment documents a real (if rare) possibility of two concurrent fire() invocations, and without this the two could reorder Publish calls relative to their own generation numbers"
  - "Per-test goleak.VerifyNone(t, goleak.IgnoreCurrent()) was substituted for a package-wide TestMain (see Deviations) after a package-wide goleak.VerifyTestMain probe was run against the CURRENT internal/uiserver test suite and failed on ~40 pre-existing, unrelated goroutines (pebble/v2's vfs.diskHealthCheckingFS ticker) from other tests in the package that are outside this plan's scope to fix"
  - "computeChange's meta-signature comparison carries an explicit prevValid bool alongside the liveSignature value, rather than relying on the zero value as a bootstrap sentinel — an opened-but-never-indexed store (hasMeta=false, lastSync=0) is a LEGITIMATE signature that collides with liveSignature{}'s zero value, so 'never checked before' had to be tracked as its own bit"
  - "Task 3's store-rebuild test recreates the store directory via a fresh MkdirAll+graphstore.Open at the SAME path after an os.Rename-away, rather than literally renaming a pre-built replacement directory INTO place — functionally identical to 'the shape an index rebuild or recovery path takes' (the plan's own phrasing), and the simpler shape a real rebuild also takes internally"

patterns-established:
  - "Bounded-coalescing send: select-default-drain-select-default, holding the registry's own mutex for the whole non-blocking sequence — the shape any future single-reader-per-channel coalescing primitive in this codebase should copy"
  - "Fixed three-element watch chain (shallow to deep) plus 'watch the deepest existing path AND its parent' as the general recipe for a non-recursive fsnotify watch that must survive its own watched directory being replaced by rename"

requirements-completed: [LIV-01, LIV-03]

coverage:
  - id: D1
    description: "An event is published only when last_sync_unix_ms actually changed; Pebble's own compaction/WAL/MANIFEST churn produces zero events"
    requirement: LIV-01
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeNoEventWhenUnchanged"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeEventOnChange"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeStatusScoped"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every read of the index's metadata opens the store, reads, and closes it before returning — no store handle outlives a single check"
    requirement: LIV-01
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeOpenCloseBalance"
        status: pass
    human_judgment: false
  - id: D3
    description: "A store whose exclusive lock is held is a retry-on-next-wake, never a publisher crash and never a terminated stream"
    requirement: LIV-01
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeLockHeldSoftSkip"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveChangeUnclassifiedOpenErrorNeverPublishesForever"
        status: pass
    human_judgment: false
  - id: D4
    description: "A UI started against a repository with no .codegraph at all goes live when the FIRST index appears"
    requirement: LIV-01
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveWatcherGoesLiveOnFirstIndex"
        status: pass
    human_judgment: false
  - id: D5
    description: "A store directory removed and recreated (a rebuild by rename) re-arms the watch; the publisher never goes permanently deaf"
    requirement: LIV-01
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveWatcherReArmsAfterStoreDirectoryReplacement"
        status: pass
    human_judgment: false
  - id: D6
    description: "A slow subscriber receives the newest state and never blocks a fast one; the publisher's send is non-blocking under all subscriber states"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryPublishCoalescesFullBuffer"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryPublishNeverBlocksNonReadingSubscriber"
        status: pass
    human_judgment: false
  - id: D7
    description: "A new subscriber receives the publisher's current state immediately, without waiting for the next change"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistrySubscribeSeedsCurrentState"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryTwoSubscribersSeeSameCurrentGeneration"
        status: pass
    human_judgment: false
  - id: D8
    description: "Server-initiated event sends are counted separately from client-initiated subscriber state (criterion 3's recorded verdict)"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryCounterSeparation"
        status: pass
    human_judgment: false
  - id: D9
    description: "Cancelling a subscriber's context removes it from the registry and leaks no goroutine"
    requirement: LIV-03
    verification:
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryContextCancelRemovesSubscriber"
        status: pass
      - kind: unit
        ref: "internal/uiserver/livepublish_test.go#TestLiveRegistryNoGoroutineLeak"
        status: pass
    human_judgment: false
  - id: D10
    description: "The new internal/uiserver -> internal/watch import disturbs neither guarded archtest closure"
    verification:
      - kind: other
        ref: "go test ./internal/graphstore/archtest/ ./internal/cli/present/archtest/ -count=1"
        status: pass
    human_judgment: false

duration: 210min
completed: 2026-09-07
status: complete
---

# Phase 6 Plan 2: Live Change Detector, Coalescing Registry, and Store Watcher Summary

**Server-side live-push engine: an open-read-close index-metadata change detector, a bounded coalescing fan-out registry seeded on Subscribe, and a three-state fsnotify arming state machine that goes live on the first index and survives a store-directory rename — all with zero HTTP/Connect dependency.**

## Performance

- **Duration:** 210 min
- **Started:** 2026-09-07T15:10:00Z (approx)
- **Completed:** 2026-09-07T18:40:00Z (approx, self-check time)
- **Tasks:** 3 (2 `tdd="true"` RED/GREEN pairs + 1 `type="auto"` single-commit task)
- **Files modified:** 2 (both new: `internal/uiserver/livepublish.go`, `internal/uiserver/livepublish_test.go`)

## Accomplishments

- `computeChange`/`changeDetector` (Task 1): opens through the package's existing `openEngine` seam, defers `Close`, reads `IndexMeta()` and compares `last_sync_unix_ms` against retained state BEFORE ever deriving `(*query.Engine).Status` — Pebble's own compaction/WAL/MANIFEST churn never reaches the expensive scan. A held store lock (`graphstore.ErrStoreLocked`) is a soft "retry next wake" outcome classified by `errors.Is` against the exported sentinel, never message text. 9/9 `TestLiveChange*` tests pass (8 required + one extra for an unclassified open error).
- `liveRegistry` (Task 2): a bounded, coalescing subscriber registry. `Subscribe` pre-loads the fresh channel with the registry's current event under the same lock that registers the subscriber — the fix for a real defect where a reconnecting tab would otherwise see nothing until the next re-index. `Publish` never blocks (drain-then-replace on a full buffer). `sendCount` (server-initiated sends) and the subscriber map (client-initiated lifecycle) are structurally separate storage, recording this repository's Criterion-3-required verdict against the `internal/mcp` `pendingWriter` bug shape. 11/11 `TestLiveRegistry*` tests pass under `-race`, stable across 3 runs.
- `livePublisher` (Task 3): a non-recursive fsnotify watch over the fixed `[repoRoot, .codegraph, store]` chain, wired to the detector and registry via a `watch.Debouncer`. `armWatches` watches the deepest existing path plus its parent (so a Remove/Rename of the watched directory is itself observed) and re-runs on every raw Create/Remove/Rename and every debounced flush. 6/6 `TestLiveWatcher*` tests pass under `-race`, including the two named T-06-39 acceptance tests. Both `archtest` packages remain green.

## Task Commits

Each task was committed atomically (Tasks 1 and 2 each carry a RED then GREEN pair per this plan's `tdd="true"` tasks; Task 3 has no `tdd` attribute and is a single commit):

1. **Task 1 RED:** `6e8b2218` — `test(06-02): add failing tests for the live change detector` (8 genuine assertion failures against a stub, 0 build errors, 0 panics).
2. **Task 1 GREEN:** `2c826267` — `feat(06-02): implement the live change detector (open, read, close, decide)` (9/9 pass).
3. **Task 2 RED:** `244cd108` — `test(06-02): add failing tests for the bounded coalescing subscriber registry` (10/11 genuine assertion failures against a stub; the 11th, a goroutine-leak test, passes vacuously against a no-op stub — documented honestly rather than papered over).
4. **Task 2 GREEN:** `305438a0` — `feat(06-02): implement the bounded coalescing subscriber registry` (11/11 pass under `-race`).
5. **Task 3 (single commit, `type="auto"`):** `e6079a8f` — `feat(06-02): the store watcher — non-recursive fsnotify plus debounce, arming state machine` (6/6 `TestLiveWatcher*` pass under `-race`, both archtest packages green).

**Plan metadata:** commit follows this SUMMARY.

## Files Created/Modified

- `internal/uiserver/livepublish.go` — new: `engineStatus` test seam, `liveSignature`/`computeChange`/`changeDetector` (Task 1), `liveRegistry`/`trySend` (Task 2), `armWatches`/`livePublisherPathChain`/`livePublisher` (Task 3). No HTTP or Connect import of any kind.
- `internal/uiserver/livepublish_test.go` — new: 26 tests total across the three tasks (9 `TestLiveChange*`, 11 `TestLiveRegistry*`, 6 `TestLiveWatcher*`), plus shared fixtures (`writeLiveMeta`, `recvWithTimeout`, `assertNothingReceived`, `countingCloser`, `withCountingOpenEngine`, `withCountingEngineStatus`, `newTestLivePublisher`).

## Decisions Made

See `key-decisions` in frontmatter. Highlights:
- `engineStatus` added as a second, purpose-specific test seam alongside the pre-existing `openEngine` var — needed to prove T-06-40's status-derivation scoping independently of the open/close counting `openEngine` already proves.
- `query.ErrNotInitialized` and any other unclassified `openEngine` failure both degrade to the same not-initialized event shape, mirroring `GetStatus`'s own `degradedStatus` for its `degradeNone` case — a defensive choice the plan implied ("never crash") but did not spell out for the "other" bucket by name.
- A second mutex (`livePublisher.checkMu`) serializes "compute change, then publish" end to end, in addition to `changeDetector`'s own internal lock, closing a theoretical publish-reordering gap `internal/watch.Debouncer`'s own doc comment names as a real (if rare) possibility.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug avoidance] Package-wide `goleak.VerifyTestMain` breaks ~40 pre-existing, unrelated tests — substituted scoped per-test `goleak.VerifyNone(t, goleak.IgnoreCurrent())`**
- **Found during:** Task 2 (the goroutine-leak acceptance criterion names `goleak` coverage for the package, and the task's own action text says "Add `goleak.VerifyTestMain` coverage for this package if none exists yet")
- **Issue:** No `TestMain` exists in `internal/uiserver` today (verified via `rg -n "func TestMain"`). Before writing the registry, I probed a package-wide `goleak.VerifyTestMain(m)` against the CURRENT test suite (temporary probe file, deleted before any production code changed) and it failed with ~40 leaked goroutines, all `pebble/v2/vfs.(*diskHealthCheckingFS).startTickerLocked` tickers from OTHER tests in this package (e.g. `startedServer`'s `go func(){ _ = srv.Serve(ctx) }()` pattern used across ~10 existing test files, none of which join that goroutine before the test returns). These are pre-existing, out of this plan's scope, and unrelated to live push.
- **Fix:** `TestLiveRegistryNoGoroutineLeak` uses `defer goleak.VerifyNone(t, goleak.IgnoreCurrent())` instead — this snapshots whatever is already running (including any lingering ticker from an earlier test) and asserts only that nothing NEW survives past this test's own `Stop()`, which is the actual property the acceptance criterion asks for ("a publisher started and stopped with N subscribers attached leaks no goroutine").
- **Files modified:** `internal/uiserver/livepublish_test.go` (no `TestMain` added)
- **Verification:** `go test -run '^TestLiveRegistry' -race -count=1` passes; the temporary package-wide probe (not committed) demonstrably failed for reasons unrelated to this plan's code, confirming the substitution is correct rather than papering over a real leak.
- **Committed in:** `305438a0` (Task 2 GREEN commit)

**2. [Rule 1 - Bug avoidance] `writeLiveMeta` test fixture retries on a transient `ErrStoreLocked` collision with the live publisher's own background check**
- **Found during:** Task 3 (`TestLiveWatcherUnrelatedChurnProducesNoEvent` failed intermittently with `graphstore: store lock held: lock held by current process` under `-race`)
- **Issue:** Task 3's tests run a REAL `livePublisher` concurrently checking the same on-disk store in the background while the test fixture also opens it directly to write Meta. This is an expected, benign race — exactly the kind of concurrent access criterion 5 exists to prove is safe — and `graphstore.Open`'s own bounded retry (5 attempts × 100ms) occasionally wasn't enough headroom under `-race`'s instrumentation overhead.
- **Fix:** `writeLiveMeta` now retries on `graphstore.ErrStoreLocked` for up to 5s (20ms between attempts) before failing the test, rather than treating the fixture's own first attempt as authoritative.
- **Files modified:** `internal/uiserver/livepublish_test.go`
- **Verification:** 3 consecutive `-race` runs of `TestLiveWatcher*`, all green (`/tmp/06-02-t3-run{1,2,3}.log`).
- **Committed in:** `e6079a8f` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — avoiding a bug/breakage the plan's literal instruction would otherwise have caused or a genuine test-timing flake). **Impact:** No scope creep; both fixes make the test suite more correct, not less strict.

## TDD Gate Compliance

Both `tdd="true"` tasks (1 and 2) carry a `test(06-02):` commit strictly before their `feat(06-02):` commit:
- Task 1: `6e8b2218` (test) → `2c826267` (feat)
- Task 2: `244cd108` (test) → `305438a0` (feat)

Neither needed a REFACTOR commit — GREEN's first implementation held without further cleanup.

**`gsd_run check tdd-red-evidence` was NOT invoked, and this is a recorded finding, not a silently skipped step.** That tool's classifier (`parseNodeTestSummary`/`tapFailedTestNames`, `gsd-core/bin/lib/prohibition-enforcement.cjs`) parses Node's `node --test` TAP reporter output specifically (`# tests N` / `# pass N` / `# fail N` summary lines, `ok N - <name>` / `not ok N - <name>` per-test lines) — a format Go's `go test -v` does not produce (`--- PASS: <Name>` / `--- FAIL: <Name>`, `ok <pkg> <time>` / `FAIL <pkg>`). Feeding Go output through it would misclassify EVERY genuine Go RED phase as `zero_tests_discovered` (since `summary.tests` would always read `0`), which is worse than not running it — a tool that always reports failure regardless of the actual test outcome teaches nothing. This is consistent with every other Go `type: tdd` plan already merged in this repository (`01-04`, `01-11`, `03-02`, `03-06`, `03-07`, `03-08`, `03-09`, `04-03`): none of their SUMMARYs reference `tdd-red-evidence` either.

**RED evidence recorded manually instead**, per the gate's actual intent (a nonzero exit with the target test failing on a real assertion, never a compile error or fixture crash):
- Task 1: `go test -run '^TestLiveChange' -v -count=1` against the stub — 8/8 `--- FAIL` lines, each showing a real assertion mismatch (e.g. `first check: changed = false, want true`), 0 build errors, 0 panics. Full log captured at commit time.
- Task 2: `go test -run '^TestLiveRegistry' -v -race -count=1` against the stub — 10/11 `--- FAIL` lines with real assertion mismatches; `TestLiveRegistryNoGoroutineLeak` passed vacuously (nothing runs in the no-op stub, so nothing can leak) — disclosed above rather than hidden, mirroring `05-08-SUMMARY.md`'s own precedent for a test that happened to already hold against a stub ("largely already-GREEN by construction, documented honestly").

## Issues Encountered

None beyond the two deviations above, both resolved within this plan.

## Verification Re-run (plan-level `<verification>` block, end-to-end)

- `GOTOOLCHAIN=go1.26.5 go test ./internal/uiserver/ -count=1 -race` — green, `ok  github.com/seanb4t/codegraph-go/internal/uiserver  27.992s` (fresh, `go clean -testcache` run immediately before, per this repo's "a `(cached)` run is not evidence" rule).
- `GOTOOLCHAIN=go1.26.5 task test:unit` — green, every package `ok` (including `internal/uiserver` at 27.144s and `internal/query` at 10.554s); `internal/daemon`'s watchdog test correctly excluded per the target's own deliberate design.
- Both archtest packages (`internal/graphstore/archtest`, `internal/cli/present/archtest`) green, confirming the new `internal/uiserver -> internal/watch` import disturbs neither guarded closure.
- `GOTOOLCHAIN=go1.26.5 go build ./...` and `go vet ./...` — clean, no output.
- `GOTOOLCHAIN=go1.26.5 task lint:go` (`golangci-lint run ./...`) — `0 issues.`

## Known Stubs

None. `livePublisher`'s constructor/`Subscribe`/`Stop` are fully implemented, tested end-to-end against a real fsnotify watcher and a real Pebble store — no placeholder bodies remain in this plan's files. (`internal/uiserver/livehandler.go`'s `CodeUnimplemented` placeholder from `06-01` is unchanged by this plan; `06-04` still owns replacing it — this plan deliberately built the engine that placeholder will call into, per its own objective: "no HTTP or Connect dependency of any kind — the handler in `06-04` consumes this, not the other way round.")

## User Setup Required

None — no external service configuration required.

## Recorded observations for `06-06`/`06-07` (per this plan's `<output>` instructions)

- **Debounce window:** `CODEGRAPH_DEBOUNCE_MS` (default 2000ms in production; tests override to 20ms). Production behavior unchanged from `internal/watch`'s existing `DebounceDuration()`.
- **Channel capacity:** 1 per subscriber (D-04) — the smallest buffer that lets "seed, then coalesce on top of it" work with a single send/drain/send sequence.
- **Measured open/close balance (Task 1, `TestLiveChangeOpenCloseBalance`):** 3 checks against a real store → opens == closes, opens ≥ 2 (both asserted; exact count depends on how many checks report `changed`, which is deterministic per that test's fixture but not pinned to a specific literal by design, per the acceptance criterion's own "opens >= 2" floor rather than an exact-count assertion).
- **Three reachable arming states, and which watches each holds** (T-06-39):
  1. `{repoRoot}` — no `.codegraph` ancestor exists yet (armWatches finds `deepestIdx == 0`, no parent to add).
  2. `{codegraphDir, repoRoot}` — `.codegraph` exists but `store` does not (or was just removed).
  3. `{storeDir, codegraphDir}` — the steady state once an index exists; `repoRoot` is dropped as shallower than `storeDir`'s parent.
- **Observed generation sequence from `TestLiveWatcherGoesLiveOnFirstIndex`:** subscribe against a repo with no `.codegraph` at all → seeded event at generation 1 (`initialized=false`, `store_exists=false`) → a real first index is created → exactly one further event at generation 2 (`initialized=true`) received within the bounded wait. `pub.Stop()` returns cleanly as the test's own secondary check.

## Next Phase Readiness

`06-04` can now build `(*uiService).WatchGraph`'s real body directly against `livePublisher`'s `Subscribe`/`Stop` — no HTTP/Connect coupling was introduced into this file, so the handler is a thin adapter over what already exists here. `06-06`/`06-07` can cite the debounce window, channel capacity, arming states, and open/close balance figures recorded above directly, rather than re-measuring them. No blockers.

---
*Phase: 06-live-push*
*Completed: 2026-09-07*

## Self-Check: PASSED

- `internal/uiserver/livepublish.go` — FOUND
- `internal/uiserver/livepublish_test.go` — FOUND
- `.planning/phases/06-live-push/06-02-SUMMARY.md` — FOUND
- Commits `6e8b2218`, `2c826267`, `244cd108`, `305438a0`, `e6079a8f` — all FOUND in `git log --oneline --all`
- All acceptance criteria for Tasks 1, 2, and 3 re-verified live (see "Verification Re-run" above)
