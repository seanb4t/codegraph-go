---
phase: 03-2026-07-28-spec-compliance
plan: 04
subsystem: mcp
tags: [mcp, go-sdk, wire-oracle, sep-2575, dynamic-tool-catalog, concurrency]

# Dependency graph
requires:
  - phase: 03-2026-07-28-spec-compliance
    provides: "03-01's AddReceivingMiddleware seam (initialize/tools/list/server/discover cases) and modern-discover-explore tracer; 03-03's ExpectedScenarioCount at 26 and its wire-oracle scenario/anchor conventions"
provides:
  - "SPEC-05 closed: a running server's tool catalog now follows .codegraph/'s on-disk presence in both directions, via a per-request re-check inside AddReceivingMiddleware — no restart, no reconnect"
  - "registerTools/unregisterTools: the single factored seam BuildServer's construction-time call and the per-request re-check both go through, so catalog membership can never diverge between the two call sites"
  - "toolCount migrated from a plain int to atomic.Int64 — written from construction and the re-check, read live by the VRFY-03 session line"
  - "Scenario.InitAfterRequest and a restructured Capture that can interleave a real `codegraph init` at a deterministic, response-observed point in a scripted session"
  - "index-appears-mid-session: the wire-level proof — an empty tools/list becomes a one-tool tools/list on the same live connection after a real mid-session `codegraph init`"
  - "ExpectedScenarioCount at 27, with all 26 pre-existing golden transcripts byte-unchanged"
affects: []

# Actuals (#2632)
actuals:
  tokens: 8320
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "One registration seam (registerTools/unregisterTools) shared by construction-time BuildServer and a per-request re-check, so the two call sites cannot silently diverge on what the tool catalog contains"
    - "A response-observed wait (drainUntil, blocking on a specific request id's response appearing in the drain loop) rather than a sleep, to interleave a real subprocess call (codegraph init) at a deterministic point inside a scripted wire-oracle session"

key-files:
  created:
    - testdata/wireoracle/transcripts/index-appears-mid-session.golden
  modified:
    - internal/mcp/server.go
    - internal/mcp/server_test.go
    - test/wireoracle/capture.go
    - test/wireoracle/scenarios.go

key-decisions:
  - "The confinement root (repoPath) passed to registerTools during the re-check is always BuildServer's construction-time value, never re-derived from ResolveCodegraphDir's own return — tightening-only, never widening (T-03-14)."
  - "The re-check resolves query.ResolveCodegraphDir against the server's construction-time startPath only, reading nothing from the request (T-03-13) — pinned by an acceptance-criterion grep of the call site's surrounding lines."
  - "unregisterTools always calls RemoveTools with the FULL possible name set (allToolNames()), not just the allowlisted subset — safe because RemoveTools no-ops on a name that was never registered, and it keeps the register/unregister name sets impossible to drift apart (both derive from exploreTool()/companionTool(), never re-typed literals)."
  - "index-appears-mid-session drives a REAL `codegraph init` subprocess rather than simulating the transition (03-CONTEXT.md's open discretion item) — a simulated transition would only prove the re-check reacts to a test hook, not to the filesystem state SPEC-05 actually promises to track."
  - "Capture's restructuring starts the stdout scanner and response-draining bookkeeping BEFORE the request-write loop and writes requests one at a time, so InitAfterRequest can block on a specific response's arrival without racing a pre-init tools/list serviced out of order."

patterns-established:
  - "A per-request re-check inside AddReceivingMiddleware, running BEFORE next(), as the general mechanism for 'the same call that observes a filesystem-state change already reflects it' — reusable pattern for any future spec requirement with the same shape."
  - "drainUntil(wantID *float64) in test/wireoracle/capture.go: a single drain-loop helper serving both 'wait for one specific response' and 'wait for the full scenario to complete' (nil case), so InitAfterRequest's mid-session wait and Capture's final drain share one implementation rather than two divergent loops."

requirements-completed: [SPEC-05]

coverage:
  - id: D1
    description: "A running server started before an index exists advertises exactly zero tools; after `codegraph init` runs against its working directory while the server is still alive, the very next tools/list on the same connection advertises exactly the set the allowlist selects — no restart, no reconnect"
    requirement: "SPEC-05"
    verification:
      - kind: unit
        ref: "internal/mcp TestIndexAppearingMidSessionRegistersTools"
        status: pass
      - kind: unit
        ref: "internal/mcp TestIndexAppearingMidSessionHonorsAllowlist"
        status: pass
      - kind: integration
        ref: "test/wireoracle TestFrozenTranscriptsMatch/index-appears-mid-session"
        status: pass
    human_judgment: false
  - id: D2
    description: "The reverse transition holds: removing the index mid-session advertises exactly zero tools on the next tools/list"
    requirement: "SPEC-05"
    verification:
      - kind: unit
        ref: "internal/mcp TestIndexDisappearingMidSessionUnregistersTools"
        status: pass
    human_judgment: false
  - id: D3
    description: "A re-check that finds no state change makes no additional registration call (the flip-guard) — repeated tools/list calls against a steady-state server never duplicate or drift the registered set"
    requirement: "SPEC-05"
    verification:
      - kind: unit
        ref: "internal/mcp TestRepeatedListsDoNotDuplicateTools"
        status: pass
    human_judgment: false
  - id: D4
    description: "The VRFY-03 session line's tools=N value is the tool count observed for the request that produced the line, not a value snapshotted once at construction"
    requirement: "SPEC-05"
    verification:
      - kind: unit
        ref: "internal/mcp TestSessionLineReflectsPostAppearanceToolCount"
        status: pass
    human_judgment: false
  - id: D5
    description: "AddTool/RemoveTools mutation from inside the existing middleware is safe under concurrent request handling — exercised, not merely asserted"
    requirement: "SPEC-05"
    verification:
      - kind: unit
        ref: "go test ./internal/mcp/... -count=1 -race (full package, including the four new tests above)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The three pre-existing set-equality tests (TestDefaultToolVisibility, TestAllowlist, TestNoIndexZeroTools) are unmodified and still pass with exact set equality / exact zero"
    verification:
      - kind: unit
        ref: "internal/mcp (all three, unchanged, in the same -race run)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The per-request index re-check resolves against the server's construction-time startPath only, never a client-supplied path argument"
    requirement: "SPEC-05"
    verification:
      - kind: other
        ref: "rg -n 'ResolveCodegraphDir' -B 6 internal/mcp/server.go — call site's argument is the closure's own startPath, never req"
        status: pass
    human_judgment: false
  - id: D8
    description: "ExpectedScenarioCount moved 26 -> 27 in the same commit as the new scenario and its transcript; all 26 pre-existing goldens are byte-unchanged"
    verification:
      - kind: unit
        ref: "test/wireoracle TestScenarioCountIsExact"
        status: pass
      - kind: unit
        ref: "test/wireoracle TestTranscriptSetMatchesScenarioSet"
        status: pass
      - kind: other
        ref: "git diff --name-status -- testdata/wireoracle/transcripts/ (exactly one A, zero M)"
        status: pass
    human_judgment: false
  - id: D9
    description: "Three consecutive captures of index-appears-mid-session are byte-identical after normalization"
    verification:
      - kind: other
        ref: "three manual `go run ./test/wireoracle/cmd/wireoracle` captures, md5 8e395b1ce16547e4d4e5595173a14d68 on all three"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-08-06
status: complete
---

# Phase 3 Plan 4: SPEC-05 Live Tool Catalog Summary

**A running MCP server's tool catalog now follows the on-disk `.codegraph/` index in both directions via a per-request re-check inside the existing middleware — proven with 5 new exact-set-equality Go tests plus a wire-level transcript driving a real mid-session `codegraph init` against the same live connection, with both re-check branches demonstrated RED against confirmed-applied, individually-reverted mutations.**

## Performance

- **Duration:** ~6 min (commit-to-commit; investigation/reading time not separately tracked)
- **Started:** 2026-08-06T11:56:01-04:00 (first task commit)
- **Completed:** 2026-08-06T12:01:57-04:00 (final task commit)
- **Tasks:** 2
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- Factored `BuildServer`'s construction-time tool registration into `registerTools`/`unregisterTools` in `internal/mcp/server.go` — the single seam both construction and the new per-request re-check go through, deriving tool names from `exploreTool()`/`companionTool()` rather than re-typed literals so the register and unregister sets can never drift apart.
- Replaced `var toolCount int` with `atomic.Int64`, written from two places (construction and the re-check) and read live by the VRFY-03 session-line branch — `tools=N` in the always-on stderr line now reports the count observed for the request that produced it, not a construction-time constant.
- Added the per-request re-check inside the existing `AddReceivingMiddleware` closure, running **before** `next()` and gated to `initialize`/`tools/list`/`tools/call`/`server/discover`. It resolves `query.ResolveCodegraphDir` against the server's construction-time `startPath` only — never a request argument (T-03-13) — compares against the last-observed state under a mutex, and on a state flip calls `registerTools`/`unregisterTools`; a non-`ErrNotInitialized` error leaves the catalog untouched (a transient stat failure must never silently empty a working catalog).
- Left the confinement root (`repoPath`) at its construction-time value on every path — an index appearing at or above `startPath` never widens where handlers may read from (T-03-14).
- Added 5 new tests in `internal/mcp/server_test.go`, all exact set equality: `TestIndexAppearingMidSessionRegistersTools`, `TestIndexAppearingMidSessionHonorsAllowlist`, `TestIndexDisappearingMidSessionUnregistersTools`, `TestRepeatedListsDoNotDuplicateTools` (the flip-guard proof), and `TestSessionLineReflectsPostAppearanceToolCount` (reusing `sendRawInitialize`/`parseSessionLineFields`). The three pre-existing set-equality tests are untouched.
- Demonstrated both re-check branches RED per the standing repo rule: removed the false-to-true branch, confirmed the two appearance tests FAIL, reverted, confirmed green; separately removed the true-to-false branch, confirmed the disappearance test FAILS, reverted, confirmed green — `git diff -- internal/mcp/server.go` byte-clean after each revert.
- Added `Scenario.InitAfterRequest` to `test/wireoracle/capture.go` (zero means never, so all 26 pre-existing scenarios are unaffected) and restructured `Capture`: the stdout scanner and response-draining bookkeeping now start before the request-write loop, requests are written one at a time, and a new `drainUntil` helper blocks (bounded by the existing 30-second `runCtx` deadline) until a specific request's response has been observed before running `binPath init workDir` and continuing to write further requests. No sleep anywhere in the new code.
- Added the `index-appears-mid-session` scenario to `test/wireoracle/scenarios.go`: fixture copied but not indexed, `initialize` + two `tools/list` requests, `InitAfterRequest: 2` runs a real `codegraph init` against the working directory once the id-2 response is observed. Froze the transcript via the oracle's own capture CLI against a freshly rebuilt `bin/codegraph`, captured three consecutive times to prove determinism (all three byte-identical, md5 `8e395b1ce16547e4d4e5595173a14d68`).
- Moved `ExpectedScenarioCount` from 26 to 27 in the same commit as the new scenario and transcript. `git diff --name-status -- testdata/wireoracle/transcripts/` shows exactly one `A` and zero `M` — the 26 pre-existing goldens are byte-unchanged, proving the harness restructuring perturbed no other capture.

## Task Commits

Each task was committed atomically:

1. **Task 1: The live catalog — factored registration, per-request re-check, mutation-safe count** - `d484b43` (feat)
2. **Task 2: Prove it on the wire — a real mid-session `codegraph init`** - `b9b74b4` (test)

_No plan-metadata commit yet — this SUMMARY plus STATE.md/ROADMAP.md/REQUIREMENTS.md updates land in the final `docs(03-04)` commit below._

## Files Created/Modified

- `internal/mcp/server.go` - factored `registerTools`/`unregisterTools`/`allToolNames`, `atomic.Int64` tool counter, the per-request re-check inside `AddReceivingMiddleware`, and the updated VRFY-03 session-line doc comment
- `internal/mcp/server_test.go` - 5 new exact-set-equality tests (see Accomplishments)
- `test/wireoracle/capture.go` - `Scenario.InitAfterRequest` and the restructured `Capture`/`drainUntil`
- `test/wireoracle/scenarios.go` - the `index-appears-mid-session` scenario and `ExpectedScenarioCount` bumped to 27
- `testdata/wireoracle/transcripts/index-appears-mid-session.golden` - new frozen transcript (created via `test/wireoracle/cmd/wireoracle`, never hand-written)

## Decisions Made

- Drove a real `codegraph init` subprocess for the wire-oracle proof rather than simulating the transition (03-CONTEXT.md's open discretion item) — a simulated transition would only prove the re-check reacts to a test hook, not to real filesystem state, which is what SPEC-05 actually promises.
- `unregisterTools` removes the FULL possible tool-name set (`allToolNames()`) rather than tracking exactly which subset was registered — simpler and equally correct since `RemoveTools` is documented as a no-op on a name that was never added, and it keeps the register/unregister sets structurally unable to drift apart (both derive from `exploreTool()`/`companionTool()`).
- Moved `gitmeta.NewCachingDetector()` construction outside the `if hasIndex` block so exactly one detector still exists per server (D-13 unchanged) even when `hasIndex` starts false and the re-check registers tools later in the server's lifetime.
- `drainUntil(wantID *float64)` is one helper serving both the mid-session wait (`wantID` non-nil) and the scenario's final completion drain (`wantID` nil, byte-identical behavior to the pre-restructuring final loop) — avoided writing two separate drain loops that could silently diverge.

## Deviations from Plan

None — plan executed exactly as written. No Rule 1/2/3 auto-fixes were needed; both tasks' `<action>` blocks and acceptance criteria were satisfied without adjustment.

## Issues Encountered

- `go test ./...` (the full repo suite, run once at the end for a broad sanity check beyond this plan's own scope) shows 4 pre-existing failures in `internal/daemon` (`TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonSharedWriter`, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`, `TestDaemonFlushLockRequeueGivesUpPerEpisode`), all timing-sensitive ("timed out waiting for..."). Confirmed out of this plan's scope: `git diff --stat d844e04 HEAD -- internal/` shows only `internal/mcp/server.go` and `internal/mcp/server_test.go` changed — `internal/daemon` is untouched by either task. Per the standing scope-boundary rule, not fixed; logged here rather than in `deferred-items.md` since no file in this plan's scope touches `internal/daemon`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SPEC-05 is closed. `ExpectedScenarioCount` is now 27 and `Scenarios()`/`testdata/wireoracle/transcripts/` agree.
- The `AddReceivingMiddleware` closure now runs a pre-`next()` step in addition to its existing post-`next()` switch — established as the pattern for any future spec requirement needing "the same request that observes a state change already reflects it."
- `registerTools`/`unregisterTools`/`allToolNames` are available for reuse if a future phase (Phase 5, SPEC-09's `subscriptions/listen`) needs the same registration seam — this plan deliberately built nothing beyond what SPEC-05 needs; `changeAndNotify`'s free `notifications/tools/list_changed` delivery to Legacy sessions is a side effect this plan relies on and wrote no code for.
- No blockers. `internal/mcp/archtest`'s VRFY-02 guard was not triggered — no protocol-version literal or SDK constant reference was introduced.

---
*Phase: 03-2026-07-28-spec-compliance*
*Completed: 2026-08-06*

## Self-Check: PASSED

All created/modified files found on disk (`internal/mcp/server.go`, `internal/mcp/server_test.go`, `test/wireoracle/capture.go`, `test/wireoracle/scenarios.go`, `testdata/wireoracle/transcripts/index-appears-mid-session.golden`, this SUMMARY). Both task commits (`d484b43`, `b9b74b4`) found in git log.
