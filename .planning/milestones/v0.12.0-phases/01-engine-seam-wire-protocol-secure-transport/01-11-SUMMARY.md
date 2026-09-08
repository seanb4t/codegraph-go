---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 11
subsystem: api
tags: [connect-go, protobuf, uiserver, graphstore, degrade, srv-04, pebble]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport
    provides: "Plan 01-10's nine-RPC UIService surface with SourceBlob attached, and plan 01-08's allocation of GetStatusResponse fields 8/9"
provides:
  - "internal/uiserver.classifyDegrade — an explicit ordered classifier (not-initialized outranks locked) over query.ErrNotInitialized and graphstore.ErrStoreLocked"
  - "internal/uiserver.errIndexingInProgress / indexingInProgressMessage — the one CodeUnavailable + typed IndexingInProgress detail shape every non-GetStatus handler's degraded response uses"
  - "internal/uiserver.degradedStatus — GetStatus's D-16 answer from filesystem facts alone (query.ResolveCodegraphDir) when openEngine fails, with graph-derived counts zeroed"
  - "GetStatusResponse.store_exists=8 / .indexing_in_progress=9 — the fields plan 01-08 allocated, now declared and populated"
  - "internal/graphstore.TestOpenSucceedsOnTheFinalAttempt — the final-attempt boundary test of Open's bounded retry loop, event-synchronized via an acknowledgement channel on the openLockRetrySleep seam"
affects: [04-index-health, 05-ui-shell, 06-ui-polish]

# Actuals (#2632)
actuals:
  tokens: 13876
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Degrade classification is an explicit ordered errors.Is check over exported sentinels only (query.ErrNotInitialized, graphstore.ErrStoreLocked) — never a string match, never a broader check"
    - "A handler with a genuine partial-availability answer (GetStatus, D-16) bypasses the shared withEngine/mapEngineError seam entirely and owns its own degrade path, documented as the one deliberate exception"
    - "D-15's no-second-retry-layer property is proven by counting openEngine invocations (a package-level func var swap), never by a wall-clock assertion"
    - "A causal-edge test release (a channel fired from a wrapped seam's first invocation) replaces timer-based test coordination for proving a transient collision rides out a bounded retry budget"
    - "The graphstore package's own retry-loop seam (openLockRetrySleep) is extended with an acknowledgement channel, not just a signal channel, when a test needs to make the FINAL retry attempt (not just any attempt) deterministic"

key-files:
  created:
    - internal/uiserver/degrade.go
    - internal/uiserver/degrade_test.go
  modified:
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiserver/handlers.go
    - internal/uiserver/readonly_test.go
    - internal/graphstore/open_lock_test.go

key-decisions:
  - "D-14: a locked store past graphstore.Open's retry budget renders as CodeUnavailable carrying a typed IndexingInProgress detail — never CodeFailedPrecondition, never surfaced to the human as a failure"
  - "D-15: no retry layer added above graphstore.Open's existing bounded budget; the no-second-layer property is proven by counting openEngine invocations, not by timing"
  - "D-16: GetStatus is the one deliberate exception to withEngine — it calls openEngine directly and degrades to a successful, filesystem-facts-only response when the open fails, because Engine.Status() never runs in that case today and there is nothing to fall back to"
  - "GetStatusResponse.initialized stays false in BOTH degrade cases; the new store_exists field is what distinguishes not-initialized (false) from locked (true) — this is a doc-comment clarification of an existing field's semantics, not a new field's invention"
  - "statusToProto now sets store_exists=true unconditionally on the successful path, since Status() only ever runs against an already-opened store — an honest value the plan's action text did not explicitly call out but which the new field's own meaning requires"
  - "degradedStatus's actual signature is degradedStatus(repoPath string, openErr error) (*uiv1.GetStatusResponse, error) — the plan's stated degradedStatus(repoPath string) (*uiv1.GetStatusResponse, error) cannot discriminate not-initialized from indexing-in-progress without the original open error, so the classification input was added as a second parameter (see Deviations)"

patterns-established:
  - "Deferred: build-tagged integration coverage — a task that declares no test of its own (Form A3) still leaves a named heading in the SUMMARY for the executor to point at, decided at plan time on every branch, not contingent on what the exercise produces"

requirements-completed: [SRV-04]

coverage:
  - id: D1
    description: "A locked store renders as CodeUnavailable with a typed IndexingInProgress detail on every non-GetStatus RPC (D-14), never as a raw error or CodeFailedPrecondition"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestNonStatusRPCsDegradeWhenAHolderNeverReleases"
        status: pass
    human_judgment: false
  - id: D2
    description: "A holder that releases within graphstore.Open's bounded budget produces a normal successful result from every RPC, driven by a causal edge rather than a timer"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget"
        status: pass
    human_judgment: false
  - id: D3
    description: "No second retry layer exists above graphstore.Open's own bounded budget (D-15), proven by counting store opens, not by measuring time"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestDegradedRPCOpensTheStoreExactlyOnce"
        status: pass
    human_judgment: false
  - id: D4
    description: "Status still answers when the store cannot be opened at all, reporting store-exists and indexing-in-progress from filesystem facts with graph-derived counts zeroed (D-16)"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestStatusDegradesOnLock"
        status: pass
    human_judgment: false
  - id: D5
    description: "A repository with no .codegraph/ at all reports a distinct not-initialized state, never conflated with the locked state"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestStatusOnUninitializedRepo"
        status: pass
    human_judgment: false
  - id: D6
    description: "The degrade precedence (not-initialized outranks locked) is an explicit ordered check, tested with a synthetic error satisfying both sentinels since the two conditions cannot co-occur on a real filesystem"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestClassifyDegradePrecedence"
        status: pass
    human_judgment: false
  - id: D7
    description: "Calling the same RPC twice against a locked store yields identical CodeUnavailable and detail both times, mutating nothing"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestDegradeIsIdempotent"
        status: pass
    human_judgment: false
  - id: D8
    description: "The final-attempt boundary of graphstore.Open's bounded retry loop is deterministic and event-synchronized, distinct from the release-between-attempts case an existing test already covers"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/graphstore/open_lock_test.go#TestOpenSucceedsOnTheFinalAttempt"
        status: pass
    human_judgment: false
  - id: D9
    description: "Concurrent RPCs each open, snapshot and close independently; a holder attempted only after every concurrent RPC completes acquires the store without needing a fairness guarantee"
    requirement: "SRV-04"
    verification:
      - kind: unit
        ref: "internal/uiserver/degrade_test.go#TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete"
        status: pass
    human_judgment: false
  - id: D10
    description: "The coexistence property (a real codegraph daemon / serve --mcp running concurrently with codegraph ui) holds against real processes, and a fresh Open succeeds immediately once no handle is retained"
    verification:
      - kind: manual_procedural
        ref: "Task 3 Exercises A/B/C transcripts, below"
        status: pass
    human_judgment: true
    rationale: "Exercises A/B/C drive real codegraph binaries as separate OS processes (daemon start, serve --mcp, a throwaway holder) — outside what an automated go test invocation can assert; the transcripts below are the evidence, but confirming real-process behavior against a human's own subsequent judgment is the intended verification path for this deliverable."

# Metrics
duration: ~105min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 11: Degrade Path Legibility (SRV-04) Summary

**A locked store answers as `CodeUnavailable`+`IndexingInProgress` on every read RPC except `GetStatus`, which degrades to a successful, filesystem-facts-only response — proven with a causally-released holder for the transient case, a permanently-held one for the sustained case, and a counted (never timed) proof that no second retry layer exists above `graphstore.Open`'s own bounded budget.**

## Performance

- **Duration:** ~105 min
- **Started:** 2026-08-23T~16:00Z (est.)
- **Completed:** 2026-08-23T17:45Z
- **Tasks:** 3
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `internal/uiserver/degrade.go`: `classifyDegrade` (an explicit ordered check — not-initialized outranks locked), `errIndexingInProgress` (the one `CodeUnavailable`+`IndexingInProgress` shape every non-`GetStatus` handler uses), `degradedStatus` (D-16's filesystem-facts-only `GetStatus` answer)
- `GetStatusResponse` gains `store_exists=8` and `indexing_in_progress=9` — the numbers plan 01-08 allocated, now declared in the descriptor and populated; plan 01-09's known-number fixture extended by exactly two entries (`uiProtoFieldFixtureLenAtPlan0111 = uiProtoFieldFixtureLenAtPlan0110 + 2`)
- `GetStatus` becomes the one handler that bypasses `withEngine`: it calls `openEngine` directly and degrades to a successful response from `query.ResolveCodegraphDir` when the open fails, instead of erroring
- Nine behavioral degrade tests in `internal/uiserver/degrade_test.go`, each naming the single outcome it requires (no "succeeds or degrades" assertions anywhere)
- `internal/graphstore/open_lock_test.go` gains `TestOpenSucceedsOnTheFinalAttempt`, the final-attempt boundary test that did not previously exist — built on the same `openLockRetrySleep` seam plus an acknowledgement channel
- Three multi-process exercises run against real `codegraph` binaries (Task 3), all confirming the expected coexistence and degrade behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: The degrade classifier, the typed error and a `Status` that answers without the store** - `dcfd964` (feat)
2. **Task 2: Determinate degrade behavior — one named outcome per test, opens counted not timed** - `d21f05c` (test)
3. **Task 3: Multi-process evidence, honestly scoped, and the phase verification close-out** - no code commit (Form A3: this task's only file edit, one package-doc comment line, was already made as part of Task 2's edit to `degrade_test.go`'s header; its deliverable is the recorded transcripts below)

**Plan metadata:** (this commit, docs: complete plan)

_TDD gate sequence: see "TDD Gate Compliance" below — this plan's tasks combine schema/production-code/test authoring within one task's action block, verified together by one gate, rather than splitting into separate RED-then-GREEN commits._

## Files Created/Modified

- `internal/uiserver/degrade.go` - `degradeKind`, `classifyDegrade`, `indexingInProgressMessage`, `errIndexingInProgress`, `degradedStatus`
- `internal/uiserver/degrade_test.go` - all nine behavioral degrade tests plus test-only helpers (`namedRPCCall`, `allNineRPCCalls`, `nonStatusRPCCalls`, `indexingInProgressDetail`, `snapshotRepoFiles`, `containsPathSeparator`)
- `internal/uiproto/uiv1/ui.proto` / `ui.pb.go` - `GetStatusResponse.store_exists=8`, `.indexing_in_progress=9`; doc-comment updates to `initialized` and `IndexingInProgress`
- `internal/uiserver/handlers.go` - `mapEngineError` gains the `graphstore.ErrStoreLocked` branch; `GetStatus` rewritten to bypass `withEngine`; `statusToProto` sets `store_exists=true`
- `internal/uiserver/readonly_test.go` - `uiProtoFieldFixtureLenAtPlan0111` and its two-entry fixture extension
- `internal/graphstore/open_lock_test.go` - `TestOpenSucceedsOnTheFinalAttempt`

## Decisions Made

- **`degradedStatus`'s real signature took a second parameter.** See "Deviations from Plan" below — this is the one place the plan's literal function signature could not be implemented as written without losing the ability to distinguish the two degrade states.
- **`statusToProto` now sets `store_exists=true` on the successful path.** The plan's action text did not explicitly instruct this, but leaving the new field always `false` on the ordinary success path would make the field lie about a store that manifestly exists (`Status()` only ever runs against an already-opened store). Applied as Rule 2 (missing critical functionality — an unset boolean the client is documented to trust).
- **`initialized`'s doc comment was updated** to state plainly that it is `false` in BOTH degrade cases, and that `store_exists` is what distinguishes them — a clarification of an existing field's semantics that this plan is the first to need, not a new invention.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug/design gap] `degradedStatus`'s stated one-argument signature cannot classify which degrade state occurred**
- **Found during:** Task 1 (writing `degrade.go`)
- **Issue:** The plan specifies `degradedStatus(repoPath string) (*uiv1.GetStatusResponse, error)`. `query.ResolveCodegraphDir(repoPath)` alone can only ever report `degradeNotInitialized` or `degradeNone` — it never returns `graphstore.ErrStoreLocked`, since it only stats the `.codegraph/` directory and never attempts to open the store. A one-argument `degradedStatus` therefore cannot distinguish the not-initialized case from the indexing-in-progress case; it would always report `store_exists=false, indexing_in_progress=false` even when the real failure was a lock held past the retry budget.
- **Fix:** Added `openErr error` as `degradedStatus`'s second parameter. `GetStatus` passes the error `openEngine` actually returned; `degradedStatus` classifies it via `classifyDegrade` first (short-circuiting to the caller's `mapEngineError` path on `degradeNone`), then independently confirms `store_exists` via `query.ResolveCodegraphDir(repoPath)` — so the filesystem-facts-only property the plan describes is preserved (store_exists is never merely trusted from openErr's own classification), while indexing_in_progress is only answerable from the classified openErr.
- **Files modified:** `internal/uiserver/degrade.go`, `internal/uiserver/handlers.go`
- **Verification:** `TestStatusDegradesOnLock` and `TestStatusOnUninitializedRepo` both pass and assert the two states are distinct (one asserts `indexing_in_progress=true, store_exists=true`; the other asserts both `false`).
- **Committed in:** `dcfd964` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug/design-gap correction to a plan-specified function signature)
**Impact on plan:** Necessary for correctness — the plan's literal signature could not deliver the D-16 property it describes. No scope creep; the fix stayed inside `degrade.go`/`handlers.go`, the exact files the plan already named for this task.

## TDD Gate Compliance

This plan's frontmatter is `type: tdd` and Tasks 1 and 2 carry `tdd="true"`, but the task commits do not follow a strict `test(...)` → `feat(...)` RED-then-GREEN sequence at the plan level: Task 1 committed as `feat(01-11)` (dcfd964, combining the proto schema change, `degrade.go`'s production code, `handlers.go`'s wiring, the fixture extension, AND its own four behavior-block tests in one commit, since the task's own `<verify>` gate requires all of them passing together as one unit), and Task 2 committed as `test(01-11)` (d21f05c, adding five more behavioral tests plus the `graphstore` boundary test, with no new production code — the seams it exercises, `openEngine` and `openLockRetrySleep`, already existed from plan 01-01 and `graphstore`'s own prior work).

So the git-log order is `feat` then `test`, not `test` then `feat`. This is a genuine departure from the standard TDD RED-then-GREEN commit sequence. It reflects this plan's own task structure (each task's action block interleaves schema, production code and tests, verified together by a single conjunctive gate) rather than an oversight caught after the fact — restructuring it into separate RED/GREEN commits per task would have meant either committing intentionally-red tests against not-yet-existing production code within Task 1 (contradicting the plan's own single-gate verification for that task) or splitting Task 1 itself in a way the plan does not describe. Recorded here per the standing instruction rather than silently passed over.

Every individual test, in both commits, was run and observed passing (GREEN) before its commit landed — no test was ever committed in a state its own commit's code could not satisfy.

## Known Stubs

None — every field this plan adds (`store_exists`, `indexing_in_progress`) is wired to real logic (`classifyDegrade`, `query.ResolveCodegraphDir`) on both the success and degrade paths, and both are exercised by tests that assert on their actual values, not on a hardcoded placeholder.

## Deferred: build-tagged integration coverage

Task 3's three exercises against real `codegraph` binaries (below) surfaced no behavior the in-process tests in `internal/uiserver/degrade_test.go` miss. Every RPC's verdict under every holder matched the in-process tests' own expectations exactly: transient collisions (Exercise A) succeeded normally under both `codegraph daemon start` and `codegraph serve --mcp`; the deliberately constructed sustained holder (Exercise B) produced exactly the degrade `TestNonStatusRPCsDegradeWhenAHolderNeverReleases`/`TestStatusDegradesOnLock` already assert; and the post-shutdown open (Exercise C) confirmed no handle was retained, matching `TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete`'s in-process proof. No build-tagged integration test is added by this task, on this or any branch (per the plan's own instruction, this is decided at plan-authoring time, not contingent on what an exercise happens to produce) — this heading is populated with "nothing deferred" rather than omitted, so a future reader searching for it finds a recorded, reasoned absence rather than a missing section.

## Task 3: Multi-Process Evidence

Fixture: `internal/indexer/testdata/gofixture` copied to a scratch directory, `codegraph init` + `codegraph index --force` run against it (4 files, 20 nodes, 22 edges). A throwaway Go client (`uiv1connect.NewUIServiceClient`, deleted before this plan's commits landed — never part of the shipped module) issued all nine RPCs against each running `codegraph ui` instance below.

### Exercise A — the realistic concurrent case

**With `codegraph daemon start` running against the fixture** (foreground, syncing/watching):

```
GetStatus      OK  initialized=true store_exists=true indexing_in_progress=false node_count=20
Search         OK
Files          OK
Callers        OK
Callees        OK
Impact         OK
Affected       OK
GetNodeDetail  OK
Explore        OK
```

**With `codegraph serve --mcp` running against the fixture** (stdin held open via `tail -f /dev/null` piped in, so the process stays alive without ever receiving a request):

```
GetStatus      OK  initialized=true store_exists=true indexing_in_progress=false node_count=20
Search         OK
Files          OK
Callers        OK
Callees        OK
Impact         OK
Affected       OK
GetNodeDetail  OK
Explore        OK
```

All nine RPCs returned a normal successful result under both concurrent holders, exactly as expected. This is recorded as **"not observed, and expected not to be observed"** for the degrade path specifically — neither `codegraph daemon` (opens only inside its flush, then closes) nor `codegraph serve --mcp` (opens per tool call) holds the store long enough to produce a lock-held collision `codegraph ui`'s own `graphstore.Open` retry would even notice; both processes' own opens complete and release well within a single retry attempt. This is the correct, expected result for the transient case, not a gap — Exercise B below is what exercises the degrade path.

### Exercise B — the sustained case, deliberately constructed

**No shipped process holds the store long-term**, so a throwaway program (`graphstore.Open` on the fixture's store directory, then block forever — deleted before this plan's commits landed, never part of the shipped module) was built and started to hold the lock for the whole exercise:

```
GetStatus      OK  initialized=false store_exists=true indexing_in_progress=true node_count=0
Search         unavailable      The index is being rebuilt. Please retry shortly.
Files          unavailable      The index is being rebuilt. Please retry shortly.
Callers        unavailable      The index is being rebuilt. Please retry shortly.
Callees        unavailable      The index is being rebuilt. Please retry shortly.
Impact         unavailable      The index is being rebuilt. Please retry shortly.
Affected       unavailable      The index is being rebuilt. Please retry shortly.
GetNodeDetail  unavailable      The index is being rebuilt. Please retry shortly.
Explore        unavailable      The index is being rebuilt. Please retry shortly.
```

Exactly the expected result: all eight non-`GetStatus` RPCs returned `connect.CodeUnavailable` with the `IndexingInProgress` detail's exact message, and `GetStatus` itself returned a successful, degraded response — `initialized=false`, `store_exists=true` (the `.codegraph/` directory was found), `indexing_in_progress=true`, `node_count=0` (graph-derived counts zeroed). **This holder was built for the exercise and corresponds to no shipped process** — the sustained case is real (a long re-index is exactly it) but is not reproducible from the shipped binaries alone within a short manual exercise window.

### Exercise C — no retained handle

After stopping `codegraph ui` and killing the Exercise B holder, a fresh `graphstore.Open` against the same store directory was attempted immediately:

```
Exercise C: fresh graphstore.Open succeeded (holder process is alive and printed 'holding')
holding
```

The fresh open succeeded immediately (the new holder process started and printed its "holding" line within 0.5s, confirming `graphstore.Open` returned without needing any retry-budget wait), proving `codegraph ui` retained no handle across the exercise — consistent with `TestUIServiceHoldsNoStoreTypedField` (structural, plan 01-01) and `TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete` (behavioral, this plan).

No RPC returned `CodeInternal` in any exercise.

## Issues Encountered

- `task test:unit`'s first run failed one subtest: `TestFrozenTranscriptsMatch/toolslist-repeat` in `test/wireoracle`, with a JSON-RPC response `id` mismatch (`id:3` vs `id:2`, arrival order not content). This is the pre-existing, extensively documented wire-oracle flake recorded in `.planning/todos/pending/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` and disproven-unrelated-to-FIX-01 in `01-CONTEXT.md`/`01-03-SUMMARY.md` (go-sdk's `jsonrpc2.Async` dispatch runs pipelined `tools/list` calls in separate goroutines with no ordering guarantee — a wire-oracle harness defect, not a server defect). Confirmed non-deterministic: re-running the same subtest in isolation immediately after passed, and a full `task test:unit` re-run passed clean end-to-end including `test/wireoracle`. Out of scope for this plan (touches only `internal/uiserver`/`internal/graphstore`/`internal/uiproto`, none of which `test/wireoracle` exercises) — not fixed, per the scope-boundary rule; already tracked in the existing open todo.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 1's degrade path (SRV-04) is complete: `codegraph ui` never surfaces a raw error for a transient or sustained lock, and `GetStatus` answers even when the store cannot be opened at all.
- This was the final plan of the last wave (wave 7) in Phase 1 — the orchestrator now runs phase-level verification.
- Phase 4's index-health verdict (a future phase) can rely on `GetStatus` being reachable exactly when things are wrong, per D-16.

## Self-Check: PASSED

All created/modified files confirmed present on disk; both task commit hashes (`dcfd964`, `d21f05c`) confirmed present in `git log`.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
