---
phase: 05-live-tool-catalog-change-notification
plan: 01
subsystem: testing
tags: [mcp, jsonrpc, subscriptions-listen, wire-oracle, go-sdk, spec-09]

# Dependency graph
requires:
  - phase: 03-2026-07-28-spec-compliance
    provides: "Scenario.InitAfterRequest (mid-session codegraph init harness hook), the per-request recheckCatalog/registerTools/unregisterTools re-check that makes the catalog mutate at all, and the wire-oracle's SDK-independent capture architecture"
  - phase: 04
    provides: "the advisory (non-blocking) transcript-freeze CI guard this plan's new transcript passes through cleanly"
provides:
  - "Wire proof (frozen transcript) that an opted-in Modern subscriptions/listen stream receives notifications/tools/list_changed after a real mid-session codegraph init, on the same live connection"
  - "Scenario.AwaitAfterRequest and Scenario.NoResponseRequests harness fields (capture.go), reusable by any future scenario involving a long-lived notification stream"
  - "Four hand-authored spec anchors run against a fresh capture: acknowledgment-echo set equality (D-02 non-vacuity), notification content-freeness, and capability-advertised-true on both negotiation paths"
  - "Two recorded RED mutations (5, 6) proving both new gates are non-vacuous"
affects: []

# Actuals (#2632)
actuals:
  tokens: 13302
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AwaitAfterRequest/NoResponseRequests: waiting on an observed frame method (never a sleep) to make a long-lived, asynchronously-dispatched notification stream capturable deterministically"
    - "Anchors re-capture fresh rather than reading the frozen golden, so a wholesale transcript regeneration cannot launder a regression past the byte comparison"

key-files:
  created:
    - testdata/wireoracle/transcripts/modern-listen-catalog-change.golden
  modified:
    - test/wireoracle/capture.go
    - test/wireoracle/scenarios.go
    - test/wireoracle/oracle_test.go
    - test/wireoracle/anchors.go
    - test/wireoracle/MUTATION-PROOF.md

key-decisions:
  - "NoInitialize with Modern _meta on every request is required for the new scenario, not stylistic — a classic initialize would route the session onto go-sdk's Legacy no-opt-in notification channel and make the proof vacuous (05-CONTEXT D-01/key_links)"
  - "Registered 4 new Anchor entries (ack echo, notification delivery, capability x2) rather than the 5 the plan's Task 2 acceptance criterion names — the plan's own action text describes exactly 4; a 5th was not specified anywhere and inventing one would violate the plan's own calibration note. See Deviations."
  - "MUTATION-PROOF.md's mutation 5 record corrects one detail of the plan's own predicted collateral: Capture never completes under mutation 5 (still blocked waiting for the notification), so assertFramingInvariant's exactly-zero check never gets a chance to run — the plan predicted it would also fire"
  - "assertSubscriptionAckEcho was additionally proven via a temporary, never-committed probe file (mirroring MUTATION-PROOF.md's existing mutation-3 precedent), because mutation 6 also raced away the id-3 tools/list response (go-sdk dispatches tools/list asynchronously too), which masked the anchor's own subtest inside TestSpecAnchorsHold"

patterns-established:
  - "A scenario opening a long-lived, unsolicited-frame-emitting stream must name both an AwaitAfterRequest wait on the LAST request (keeps stdin open past changeAndNotify's 10ms debounce) and, if the request's own response is asymmetric (SubscriptionsListenResult only on graceful teardown), a NoResponseRequests exemption — the two are independent axes and Capture/assertFramingInvariant both read the single expectedResponseIDs() seam"

requirements-completed: [SPEC-09]

coverage:
  - id: D1
    description: "An opted-in Modern subscriptions/listen stream receives notifications/tools/list_changed on the same live connection after a real mid-session codegraph init (SPEC-09 criterion 2)"
    requirement: SPEC-09
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestFrozenTranscriptsMatch/modern-listen-catalog-change"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestSpecAnchorsHold/modern-listen-catalog-change"
        status: pass
    human_judgment: false
  - id: D2
    description: "The server advertises tools.listChanged:true on both the Legacy initialize and Modern server/discover negotiation paths, asserted against a fresh capture (SPEC-09 criterion 1)"
    requirement: SPEC-09
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestSpecAnchorsHold/handshake-explore/capabilities.tools.listChanged_==_true_on_the_Legacy_initialize_path"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestSpecAnchorsHold/modern-discover-explore/capabilities.tools.listChanged_==_true_on_the_Modern_server/discover_path"
        status: pass
    human_judgment: false
  - id: D3
    description: "The acknowledgment echo is asserted by exact set equality (never non-empty), so a dead subscription (D-02's misspelled-opt-in shape) cannot satisfy it — proven non-vacuous by mutation 6"
    requirement: SPEC-09
    verification:
      - kind: unit
        ref: "test/wireoracle/anchors.go#assertSubscriptionAckEcho (registered in TestSpecAnchorsHold)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The 27 pre-existing frozen transcripts stay byte-unchanged, evidencing that a non-opting client observes no session-behavior change (SPEC-09 criterion 3)"
    requirement: SPEC-09
    verification:
      - kind: unit
        ref: "git diff --name-status <phase-base>..HEAD -- testdata/wireoracle/transcripts/ (exactly one A, zero M)"
        status: pass
    human_judgment: false

duration: ~50min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 1: Live Tool-Catalog Change Notification Summary

**Wire proof that an opted-in Modern `subscriptions/listen` stream actually receives `notifications/tools/list_changed` after a real mid-session `codegraph init`, closing SPEC-09 — the milestone's last open requirement.**

## Performance

- **Duration:** ~50 min (approximate — git history spans 17:21-17:33 local across the three commits; additional time was spent on exploration, three-capture determinism proof, and the two-mutation RED demonstration before/around those commits)
- **Started:** ~2026-08-06T21:15Z (approximate)
- **Completed:** 2026-08-06T21:33:25Z
- **Tasks:** 3
- **Files modified:** 6 (5 modified, 1 created)

## Accomplishments

- Froze `testdata/wireoracle/transcripts/modern-listen-catalog-change.golden`: an opted-in Modern `subscriptions/listen` stream observing `notifications/tools/list_changed` on the same live connection after a real mid-session `codegraph init`, captured three consecutive times against a freshly rebuilt `bin/codegraph`, all byte-identical (md5 `e4fba6886d2039b4232a2a61bf054d83`).
- Extended the wire-oracle harness with `Scenario.AwaitAfterRequest` (wait on an observed frame method, never a sleep) and `Scenario.NoResponseRequests` + `Scenario.expectedResponseIDs()` (the one seam both `Capture`'s completion condition and `assertFramingInvariant` read), so a long-lived, asynchronously-dispatched notification stream is capturable deterministically.
- Added four hand-authored spec anchors run against a fresh capture (never the frozen bytes): `assertSubscriptionAckEcho` (D-02 set-equality non-vacuity discriminator), `assertToolsListChangedNotification` (exactly-one, correlated, content-free), and `assertToolsListChangedCapability` registered against both negotiation paths (Legacy `initialize`, Modern `server/discover`).
- Recorded two RED mutations in `MUTATION-PROOF.md` (mutations 5 and 6) proving both new gates fail loudly against confirmed-applied, individually-reverted mutations — including one honest correction to the plan's own predicted collateral and a temporary probe file (mirroring the file's existing mutation-3 precedent) to isolate the ack-echo anchor from a second, independently-discovered race.
- Recorded criterion 3's evidence (the 27 pre-existing transcripts staying byte-unchanged) in the same file, per 05-CONTEXT D-03/D-04, rather than authoring a new no-op scenario.

## Task Commits

1. **Task 1: The wire proof** - `49406ea` (feat)
2. **Task 2: The discriminators** - `3b17fda` (feat)
3. **Task 3: Mutations RED + criterion 3 evidence** - `76d7864` (docs)

_No separate plan-metadata commit was made for the final metadata step at the time this SUMMARY was authored; see `state_updates`/`final_commit` steps below._

## Files Created/Modified

- `test/wireoracle/capture.go` - `Scenario.AwaitAfterRequest`, `Scenario.NoResponseRequests`, `Scenario.expectedResponseIDs()`, `frameMethod`, and `drainUntil`'s three-way method/id/full-set completion switch
- `test/wireoracle/scenarios.go` - the `modern-listen-catalog-change` scenario, its hand-authored method/opt-in literals (`notificationSubscriptionsAcknowledgedMethod`, `notificationToolsListChangedMethod`, `notificationsOptInFieldName`), `modernToolsListRequest`/`subscriptionsListenRequest` builders, `ExpectedScenarioCount` 27 → 28
- `test/wireoracle/oracle_test.go` - `assertFramingInvariant` reading `expectedResponseIDs()` plus an explicit exactly-zero check over `NoResponseRequests`; two new `TestToolsListExactSets` cases proving the catalog transition by set equality
- `test/wireoracle/anchors.go` - `assertSubscriptionAckEcho`, `assertToolsListChangedNotification`, `assertToolsListChangedCapability`, and four new `Anchor` registrations
- `test/wireoracle/MUTATION-PROOF.md` - mutations 5 and 6, plus criterion 3's evidence subsection
- `testdata/wireoracle/transcripts/modern-listen-catalog-change.golden` - new frozen transcript (created via `test/wireoracle/cmd/wireoracle`, never hand-written)

## Decisions Made

- Followed 05-CONTEXT D-01's corrected measurement exactly: the listen request's own `SubscriptionsListenResult` is never written on stdin EOF, so `NoResponseRequests: []int{1}` plus an exactly-zero framing check is the right shape, not a two-frame expectation.
- `AwaitAfterRequest` maps index 1 to the acknowledgment method (races the next response, per go-sdk's `jsonrpc2.Async`) and index 3 (the last request) to the notification method (keeps stdin open past `changeAndNotify`'s 10ms debounce) — both load-bearing for determinism, documented in the field's own doc comment.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 2's "five Anchor entries" instruction does not match its own description**

- **Found during:** Task 2
- **Issue:** The plan's Task 2 action text says "Register five `Anchor` entries in `Anchors()`" and the acceptance criteria expect "five new anchor subtests," but the action text's own semicolon-separated list describes exactly four: the acknowledgment echo, the notification delivery, the capability check on `handshake-explore`, and the capability check on `modern-discover-explore`. No fifth anchor is named anywhere in `must_haves`, `<behavior>`, or `<action>`.
- **Fix:** Registered the four anchors the description actually specifies. Did not invent a fifth, since doing so would have no textual basis in the plan and would violate this plan's own calibration note ("one wire proof plus its assertions... do not inflate the milestone's last plan").
- **Files modified:** `test/wireoracle/anchors.go` (documented inline in `Anchors()`'s doc comment, immediately above `func Anchors()`)
- **Verification:** `go test ./test/wireoracle/... -run TestSpecAnchorsHold -count=1 -v` shows exactly four new passing anchor subtests beyond the five that existed before this task.
- **Committed in:** `3b17fda` (Task 2 commit)

**2. [Rule 1 - Bug/finding] MUTATION-PROOF.md's mutation 5 record corrects one detail of the plan's own predicted collateral**

- **Found during:** Task 3
- **Issue:** The plan predicted that flipping the tool-capability off would additionally make `assertFramingInvariant`'s exactly-zero `NoResponseRequests` check go red in the same run ("the listen handler no longer blocks and does answer its own request id, so the exactly-zero check ... also go red"). The first half (id 1 answered immediately) is confirmed true. The second half is not what happens: `Capture` itself never returns under this mutation — it stays blocked on `AwaitAfterRequest`'s wait for the notification, which this mutation ensures never arrives — so the 30-second capture deadline fires first and `assertFramingInvariant` is never reached in that run.
- **Fix:** Recorded the actual, traced mechanism in `MUTATION-PROOF.md`'s mutation 5 section rather than restating the plan's prediction verbatim.
- **Files modified:** `test/wireoracle/MUTATION-PROOF.md`
- **Verification:** Verbatim `task test:wireoracle` output recorded in the mutation record; matches the described mechanism exactly (deadline error naming the scenario and the awaited method, "3/2 responses observed").
- **Committed in:** `76d7864` (Task 3 commit)

**3. [Rule 1 - finding, not a defect] Mutation 6 also raced away the id-3 `tools/list` response, masking the ack-echo anchor's own subtest**

- **Found during:** Task 3
- **Issue:** Removing the scenario's `AwaitAfterRequest` entry for the last request (mutation 6, as literally specified) does not only stop the harness waiting for the (never-arriving) notification — go-sdk dispatches `tools/list` asynchronously too (the same `jsonrpc2.Async` fact `AwaitAfterRequest`'s own doc comment cites for the acknowledgment), so the id-3 response itself now races process shutdown and was lost in the recorded run. This made `TestSpecAnchorsHold/modern-listen-catalog-change` fail at `assertFramingInvariant` (an id-3 framing check) before its per-scenario anchor loop ever reached `assertSubscriptionAckEcho`'s own subtest.
- **Fix:** Proved `assertSubscriptionAckEcho` directly and in isolation via a temporary, never-committed probe file (`test/wireoracle/mutation6_probe_test.go`), mirroring `MUTATION-PROOF.md`'s existing mutation-3 precedent for exactly this situation. Deleted immediately after the evidence was captured; confirmed via `git status --short` that it was never staged.
- **Files modified:** none (the probe file was never committed); `test/wireoracle/MUTATION-PROOF.md` records both the masking collateral and the isolated proof.
- **Verification:** The isolated probe's verbatim failure — `acknowledgment params.notifications = map[], want exactly map[toolsListChanged:true]` — matches D-02's dead-subscription shape exactly, recorded in `MUTATION-PROOF.md`.
- **Committed in:** `76d7864` (Task 3 commit; the probe file itself was never committed)

---

**Total deviations:** 3 (1 anchor-count clarification, 2 mutation-collateral findings recorded precisely rather than restated)
**Impact on plan:** None affect SPEC-09's substance — all three are either a plan-text inconsistency resolved in favor of what the plan actually specified, or an honestly-recorded measurement correction consistent with this milestone's "measure, don't assume" pattern. No scope creep; no anchor, scenario, or harness field was added beyond what the plan specified.

## Issues Encountered

- The first `task test:wireoracle` run under mutation 5 took ~168s (multiple 30-second capture deadlines across `TestFrozenTranscriptsMatch`, `TestEveryDeclaredFiringRuleActuallyFires`, `TestSpecAnchorsHold`, and both `TestToolsListExactSets` cases each independently re-capturing the now-permanently-blocked scenario) and exceeded the default Bash tool timeout, requiring a background run. Not a defect — an expected consequence of the mutation making every capture of this scenario hit its deadline.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SPEC-09 is closed: all three criteria (capability advertised on both paths, live notification delivery, non-opting-client non-regression) are proven on the wire and via fresh-capture anchors, not merely by frozen bytes.
- This is the milestone's final phase and final plan. No further phases are scheduled by `ROADMAP.md` for this milestone as of this plan.
- `ExpectedScenarioCount` now reads 28; any future plan adding a wire-oracle scenario should follow this plan's `AwaitAfterRequest`/`NoResponseRequests` pattern for any further long-lived-stream proof.

---
*Phase: 05-live-tool-catalog-change-notification*
*Completed: 2026-08-06*

## Self-Check: PASSED

- FOUND: test/wireoracle/capture.go
- FOUND: test/wireoracle/scenarios.go
- FOUND: test/wireoracle/oracle_test.go
- FOUND: test/wireoracle/anchors.go
- FOUND: test/wireoracle/MUTATION-PROOF.md
- FOUND: testdata/wireoracle/transcripts/modern-listen-catalog-change.golden
- FOUND: commit 49406ea (Task 1)
- FOUND: commit 3b17fda (Task 2)
- FOUND: commit 76d7864 (Task 3)
