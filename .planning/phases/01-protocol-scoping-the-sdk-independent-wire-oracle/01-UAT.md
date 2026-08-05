---
status: complete
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
source: [01-VERIFICATION.md]
started: 2026-08-05T21:50:00Z
updated: 2026-08-05T22:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. ROADMAP criterion 3 wording decision
expected: A recorded decision. If option (a), the ROADMAP.md edit is made via GSD roadmap tooling, not by hand.
result: pass
decision: |
  Option (a) selected. ROADMAP.md:88 criterion 3 restated from "reads from a
  repo-owned literal" to "is asserted against a repo-owned literal", matching
  REQUIREMENTS.md:35 VRFY-02, which the phase already satisfies. The stricter
  "reads from" (injection) property remains recorded in
  internal/mcp/protocol_version.go:20-22 as landing in Phase 2, when the
  official go-sdk swap supplies a backend a caller can actually supply a
  revision to.
note: |
  The expectation's "via GSD roadmap tooling, not by hand" clause was not
  satisfiable — no such verb exists. `gsd-tools roadmap` offers only
  analyze/get-phase/update-plan-progress/annotate-dependencies/validate/upgrade
  and `gsd-tools phase` only add/add-batch/insert/remove/complete/list-plans;
  none edits a success-criterion's text. The edit was made by hand, changing a
  value inside a shape the parser already emits (no heading or block invented).
  Verified after the edit: `git diff` shows exactly one changed line, and
  getMilestonePhaseFilter still resolves
  ["01-protocol-scoping-the-sdk-independent-wire-oracle"].

### 2. Concurrent/repeated-initialize session-line non-interleaving backstop
expected: Maintainer either accepts the mutex-based, code-review-level assurance as sufficient for this milestone, or requests a follow-up regression test before Phase 2.
result: pass
reported: "fix items 1 and 2"
severity: minor
decision: |
  Maintainer declined the code-review-level assurance and requested both gaps
  closed now rather than deferred to Phase 2. Closed in-session (see gap
  G-01-2 below) — the fix was fully specified at checkpoint time and needed no
  root-cause diagnosis, so no gap-closure plan was spawned.
result_history: |
  Recorded as `issue` when the maintainer declined the as-shipped assurance,
  then reconciled to `pass` once G-01-2 was closed and verified in the same
  session. The reconciliation is deliberate and follows the workflow's
  gap-reconcile semantics (#1921): a gap whose fix has landed is `resolved` and
  is not re-diagnosed. `phase uat-passed` scores test results rather than gap
  statuses, so leaving `issue` here would have reported phase 01 as carrying an
  unresolved defect that no longer exists. The `reported` field, the decision
  above, and the resolved G-01-2 entry below preserve the full history —
  nothing was erased to clear the gate.

## Summary

total: 2
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0
issues_found_and_resolved: 1

## Gaps

- gap_id: G-01-2
  truth: "A concurrent/repeated initialize must never produce a partially-written or interleaved session line, and that property must be enforced by something CI runs."
  status: resolved
  reason: "User reported: fix items 1 and 2"
  severity: minor
  test: 2
  resolved_by: in-session fix during UAT (no gap-closure plan required)
  resolved_at: 2026-08-05
  artifacts:
    - path: "internal/mcp/server.go"
      issue: "The AddAfterInitialize hook's mutex (server.go:180-198) had zero test coverage — every session_line_test.go test drives formatSessionLine on a single goroutine, so deleting the mutex left the package green."
    - path: "Taskfile.yml"
      issue: "test:race scoped -race to ./internal/daemon/... ./internal/watch/... ./internal/cli/... — internal/mcp was excluded, so the race detector was not a fallback net for this seam either."
  missing:
    - "DONE — internal/mcp/session_line_concurrency_test.go adds TestSessionLineSurvivesConcurrentAndRepeatedInitialize: 8 concurrent in-process clients x 4 initializes each against one BuildServer (one mutex), writing through an interleaveProneWriter that chunks its payload and yields between chunks. Asserts exactly 32 lines, each parsing via parseSessionLineFields with an intact client field."
    - "DONE — Taskfile.yml test:race now includes ./internal/mcp/..."
  non_vacuity: |
    Proven RED, not assumed. With the hook's mu.Lock()/defer mu.Unlock() replaced
    by a no-op, the test fails with visibly shredded output (e.g. "h: mcp-sh:
    mcp-session ression requestedequested=2025-11..."). server.go was then
    restored and verified byte-identical to HEAD via `git diff --quiet`.
    The test fails deterministically under plain `go test` — it does not depend
    on -race, which is why both halves of the fix were needed rather than either
    one alone.
  verification: |
    go test ./internal/mcp/ -run TestSessionLineSurvivesConcurrentAndRepeatedInitialize  → ok
    task test:race (all 9 packages, including the two newly-added)              → ok
    go test ./test/wireoracle/... (frozen 23-scenario suite, unaffected)        → ok
    go vet ./internal/mcp/...                                                   → clean

Non-blocking observation carried from 01-07's MUTATION-PROOF.md, recorded here so it is
not lost at phase close (NOT a UAT test — no maintainer action required to pass this phase):

- Mutation 3 (`exploreHandler`'s missing-query error shape) has zero blast radius on the
  frozen 23-scenario suite. No scenario exercises a handler's own required-argument
  validation failure path; the four captured error shapes are all protocol-level. Flagged
  in MUTATION-PROOF.md as input to Phase 2's SDK-04 audit. Extending the frozen set is
  still physically possible today but will be impossible once Phase 2 removes
  `mark3labs/mcp-go` from `go.mod`.
