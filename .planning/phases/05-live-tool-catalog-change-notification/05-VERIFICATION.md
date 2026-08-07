---
phase: 05-live-tool-catalog-change-notification
verified: 2026-08-06T22:10:00Z
status: passed
score: 3/3 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 5: Live Tool-Catalog Change Notification Verification Report

**Phase Goal:** A client that opts into `subscriptions/listen` is told when codegraph's tool catalog changes, instead of learning about it only on its next poll.
**Verified:** 2026-08-06T22:10:00Z
**Status:** passed
**Re-verification:** No — initial verification

**Method:** RAN — every claim below was independently reproduced (`go test`, `go build`, `git diff`, `rg` against source/golden bytes), not read off SUMMARY.md or MUTATION-PROOF.md prose.

## Goal Achievement

### Observable Truths (ROADMAP Phase 5 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | The server advertises `tools.listChanged: true` in its capabilities | ✓ VERIFIED | `assertToolsListChangedCapability` registered twice in `test/wireoracle/anchors.go` (handshake-explore/id=1 Legacy `initialize`, modern-discover-explore/id=1 Modern `server/discover`), asserted against a **fresh** capture (never the golden). Ran `go test ./test/wireoracle/... -run TestSpecAnchorsHold -count=1 -v`: both subtests present and passing, along with the two new `modern-listen-catalog-change` subtests. |
| 2 | An opted-in `subscriptions/listen` client receives `notifications/tools/list_changed` when the catalog actually changes (e.g. mid-session `codegraph init`) | ✓ VERIFIED | Read `testdata/wireoracle/transcripts/modern-listen-catalog-change.golden` directly: 4 frames — ack (`notifications:{toolsListChanged:true}`), `tools/list` result with empty `tools:[]`, `tools/list` result with `tools:[codegraph_explore]`, then `notifications/tools/list_changed` correlated to subscriptionId 1. Confirmed the scenario (`test/wireoracle/scenarios.go:1079-1098`) is `NoInitialize: true` with every request (`subscriptionsListenRequest`, `modernToolsListRequest` ×2) carrying Modern `_meta` — the vacuity trap (Legacy no-opt-in shared channel) is avoided by construction, verified by reading the actual request builders, not by trusting the doc comment. `go test ./test/wireoracle/...` passes. |
| 3 | A client that does NOT opt in observes no change in session behavior from Phase 3's server | ✓ VERIFIED | `git diff --name-status 36a3efc..HEAD -- testdata/wireoracle/transcripts/` → exactly one `A` (`modern-listen-catalog-change.golden`), zero `M`. All 27 pre-existing frozen transcripts byte-unchanged, confirmed by the same command and by `TestFrozenTranscriptsMatch`'s pass (part of the full suite run). |

**Score:** 3/3 truths verified (0 present-but-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `testdata/wireoracle/transcripts/modern-listen-catalog-change.golden` | Frozen wire proof of criterion 2 | ✓ VERIFIED | Exists, 4 lines, decoded and matches claimed shape exactly |
| `test/wireoracle/capture.go` (`AwaitAfterRequest`, `NoResponseRequests`, `expectedResponseIDs`) | Harness support for long-lived notification streams | ✓ VERIFIED | All three present (`capture.go:101,127,159`), single seam read by both `Capture`'s completion condition (`:278`) and `assertFramingInvariant` (`oracle_test.go:534`) |
| `test/wireoracle/scenarios.go` (`modern-listen-catalog-change`, `ExpectedScenarioCount=28`) | New scenario + count bump | ✓ VERIFIED | `ExpectedScenarioCount = 28` at line 481; 28 `.golden` files on disk (`ls testdata/wireoracle/transcripts/*.golden \| wc -l` = 28) — counts agree |
| `test/wireoracle/anchors.go` (4 new anchors) | Criterion-1 capability + D-02 discriminator anchors | ✓ VERIFIED | `assertSubscriptionAckEcho`, `assertToolsListChangedNotification`, `assertToolsListChangedCapability` all present; 9 total `Scenario:` entries in `Anchors()` (5 pre-existing + 4 new), matches SUMMARY's self-reported deviation (see Deviations below) |
| `test/wireoracle/oracle_test.go` (framing-invariant seam, 2 catalog-transition cases) | Set-equality proof of catalog transition | ✓ VERIFIED | `assertFramingInvariant` reads `sc.expectedResponseIDs()` and asserts exactly-zero over `NoResponseRequests` (`oracle_test.go:530-571`) |
| `test/wireoracle/MUTATION-PROOF.md` (mutations 5, 6) | Non-vacuity proof | ✓ VERIFIED | `rg -c '^## Mutation ' test/wireoracle/MUTATION-PROOF.md` = 6 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| Phase 3 `recheckCatalog`→`registerTools`→go-sdk `AddTool`→`changeAndNotify` | Opted-in stream's stdout frame | mid-session `codegraph init` | ✓ WIRED | Golden transcript shows the transition landing on the wire: empty `tools:[]` → `tools:[codegraph_explore]` → `notifications/tools/list_changed`, captured from the real binary, not asserted from source alone |
| `test/wireoracle` package | MCP SDK | import graph | ✓ CLEAN (VRFY-01 preserved) | `go list -deps ./test/wireoracle \| rg -c "modelcontextprotocol/go-sdk"` = 0 |

### Non-Vacuity (Mutation) Verification

| Mutation | Claim | Verified |
|----------|-------|----------|
| 5 — capability off | Reddens the wire gate at capture (`Capture` deadline-exceeded waiting for `notifications/tools/list_changed`) | ✓ Read verbatim recorded failure in MUTATION-PROOF.md; mechanism traced (`allowedSubscriptions` grants nothing → `shouldSendListChangedNotification` false); revert confirmed clean |
| 6 — capability off AND final await removed | Reddens `assertSubscriptionAckEcho` against `{}` (dead-subscription shape) | ✓ Read verbatim recorded failure: `got: "...\"notifications\":{}}"` vs `want: "...\"notifications\":{\"toolsListChanged\":true}}"` — exactly D-02's shape; both mutations individually recorded with diff, confirmed-applied statement, red gate, verbatim failure, and confirmed revert |

Both mutations are recorded distinctly (mutation 5 does not substitute for 6) — confirmed by reading both sections in full; mutation 5 alone cannot reach the ack-echo anchor because `Capture` never completes under it (honest correction the executor recorded rather than restating the plan's original prediction), which is exactly why mutation 6 exists as a separate proof.

### Behavioral Spot-Checks (run directly, not cited from SUMMARY)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full wire-oracle suite | `go test ./test/wireoracle/... -count=1` | `ok ... 19.030s` | ✓ PASS |
| Race detector on new harness waits | `go test ./test/wireoracle/... -count=1 -race` | `ok ... 20.866s` | ✓ PASS |
| Whole-tree build | `go build ./...` | exit 0 | ✓ PASS |
| Spec anchors (fresh-capture) | `go test ./test/wireoracle/... -run TestSpecAnchorsHold -count=1 -v` | `PASS`, all subtests including 4 new SPEC-09 anchors | ✓ PASS |
| Transcript/scenario-count agreement | `ExpectedScenarioCount=28` vs `ls *.golden \| wc -l` | 28 == 28 | ✓ PASS |
| Discriminator: no scope creep into internal/deps | `git diff --name-status 36a3efc..HEAD -- internal/ go.mod go.sum` | empty | ✓ PASS |
| CI transcript-freeze guard untouched | `git diff --name-status 36a3efc..HEAD -- .github/workflows/` | empty | ✓ PASS |
| Runtime capability literal (not mutated residue) | `rg "ListChanged" internal/mcp/server.go` | `Tools: &mcp.ToolCapabilities{ListChanged: true}` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| SPEC-09 | 05-01-PLAN.md | `subscriptions/listen` opt-in notification + `tools.listChanged: true` advertised | ✓ SATISFIED | All three ROADMAP criteria independently verified above; REQUIREMENTS.md traceability table already shows SPEC-09 → Phase 5 → Complete |

No orphaned requirements — SPEC-09 is the only requirement mapped to Phase 5 and it is the only one in scope per 05-CONTEXT.md's explicit phase boundary.

### Anti-Patterns Found

None. Scanned the phase's modified files (`test/wireoracle/{capture,scenarios,oracle_test,anchors}.go`, `test/wireoracle/MUTATION-PROOF.md`) — no `TBD`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` markers, no stub returns, no hardcoded-empty data flowing to assertions. Two temporary probe files (`mutation3_probe_test.go`, `mutation6_probe_test.go`) were used during Task 3 and confirmed never committed (`git status --short` clean at time of SUMMARY, and current `git diff 36a3efc..HEAD` shows no such files added).

### Deviations Examined (self-reported by executor)

1. **Task 2 "five Anchor entries" vs. four implemented.** Verified independently: the plan's own `<action>` prose for Task 2 names exactly four anchors (ack echo, notification delivery, capability×2) — no fifth is named anywhere in `must_haves`, `<behavior>`, or `<action>`. `Anchors()` in the current source has 9 total entries (5 pre-existing + 4 new), confirmed by `rg -n "Scenario: \"" test/wireoracle/anchors.go`. The four anchors implemented fully cover criterion 1 (both negotiation paths) and the D-02 discriminator (criterion 2's non-vacuity requirement). Judgment: **not a gap** — the plan text's "five" was a numbering error inconsistent with its own description, and the four registered anchors are sufficient for what the three ROADMAP criteria require.

2. **Mutation collateral differing from the plan's prediction.** Verified by reading MUTATION-PROOF.md's mutation 5 section in full: the plan predicted `assertFramingInvariant`'s exactly-zero check would also fire; the recorded outcome instead shows `Capture` itself never returns (blocked on `AwaitAfterRequest`'s wait, hits the 30s deadline) — a different, correctly-traced mechanism. This is an honest correction consistent with the recorded verbatim test output, not a restated prediction. Judgment: **not a gap.**

## Gaps Summary

None. All three ROADMAP Phase 5 success criteria are verified by direct reproduction (test runs, byte-level transcript inspection, git diff over the actual commit range), not by trusting SUMMARY.md's narrative. The milestone's SPEC-09 — its final open requirement — is closed.

---

_Verified: 2026-08-06T22:10:00Z_
_Verifier: Claude (gsd-verifier)_
