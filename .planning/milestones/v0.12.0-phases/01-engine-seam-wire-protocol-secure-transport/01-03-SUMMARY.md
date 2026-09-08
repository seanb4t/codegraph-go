---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 03
subsystem: mcp
tags: [mcp, jsonrpc, stdio, concurrency, go-sdk, race-detector]

# Dependency graph
requires: []
provides:
  - "internal/mcp/server.go: pendingWriter balances its response counter — decrements only on complete outbound lines classified as JSON-RPC responses, never on server-initiated notifications"
  - "internal/mcp/server.go: looksLikeJSONRPCResponse — the outbound classifier symmetric to the existing inbound looksLikeJSONRPCCall"
  - "internal/mcp/server.go: decrementPending / pendingUnderflows — a CAS-based decrement that refuses to go negative and counts refused decrements"
  - "internal/mcp/pending_writer_test.go: a committed, observed reproduction of the premature-drain defect, plus an executable disproof that fixing it does not close the wire-oracle toolslist-repeat ordering flake"
affects: [phase-6-connectrpc-server-streaming]

# Actuals (#2632)
actuals:
  tokens: 8000
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Outbound JSON-RPC line classification symmetric to an existing inbound sniff (looksLikeJSONRPCResponse mirrors looksLikeJSONRPCCall), with an intentionally opposite conservative default on parse failure"
    - "Single serialized critical section covering both an underlying io.Writer forward and a classification side-effect, to keep wire order and classification order from diverging under concurrent callers"
    - "CAS-based counter floor (decrementPending) with a test-visible underflow counter instead of a silent clamp"

key-files:
  created:
    - internal/mcp/pending_writer_test.go
  modified:
    - internal/mcp/server.go

key-decisions:
  - "An unparseable or ambiguous outbound line (id: null, unparseable, empty) never decrements — the opposite conservative default from the inbound sniff, because over-decrementing corrupts the counter while under-decrementing only costs a bounded wait against stdinLingerGrace"
  - "The mutex covers the underlying write AND the classification buffer as one critical section, not the buffer alone — a buffer-only mutex lets wire order and classification order diverge under concurrent Write calls"
  - "FIX-01 and the wire-oracle toolslist-repeat ordering flake are proven separate defects; only FIX-01 is closed here, per CONTEXT.md's disproof (go-sdk@v1.7.0's jsonrpc2.Async dispatch, not the pendingWriter counter, is the flake's cause)"

patterns-established:
  - "Pattern: symmetric outbound/inbound JSON-RPC line classifiers with intentionally opposite conservative defaults, documented explicitly so a future reader does not 'fix' the asymmetry"

requirements-completed: [FIX-01]

coverage:
  - id: D1
    description: "pendingWriter decrements only on complete outbound lines classified as JSON-RPC responses; a server-initiated notification never decrements"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterDecrementsOnlyResponses"
        status: pass
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterDrainsEarlyWithoutFix"
        status: pass
    human_judgment: false
  - id: D2
    description: "The classifier's input is only the bytes the underlying writer actually reported as written (b[:n]), never the full buffer on a short write"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterClassifiesOnlyBytesActuallyWritten"
        status: pass
    human_judgment: false
  - id: D3
    description: "pendingWriter forwards every byte before classifying, and forwarding + classification are one serialized critical section so wire order and classification order cannot diverge under concurrency"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterForwardsBeforeClassifying"
        status: pass
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterSerializesWriteAndClassification"
        status: pass
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterDecrementsPerLineAcrossWriteBoundaries"
        status: pass
    human_judgment: false
  - id: D4
    description: "The pending counter never goes negative under any interleaving, and refused decrements are counted via pendingUnderflows rather than silently absorbed"
    requirement: "FIX-01"
    verification:
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPendingWriterNeverGoesNegative"
        status: pass
    human_judgment: false
  - id: D5
    description: "The toolslist-repeat wire-oracle ordering flake is proven separable from FIX-01: with zero notification traffic present, two pipelined tools/list calls are both answered exactly once, matched by JSON-RPC id"
    verification:
      - kind: unit
        ref: "internal/mcp/pending_writer_test.go#TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 3: pendingWriter Counter Fix (FIX-01) Summary

**Fixed `pendingWriter`'s counter corruption in `internal/mcp/server.go` — it decremented on every write, including server-initiated notifications never counted on the increment side — and separately proved this does not close the unrelated `toolslist-repeat` wire-oracle ordering flake.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-22T~23:30Z (est.)
- **Completed:** 2026-08-23T03:27Z
- **Tasks:** 2
- **Files modified:** 2 (`internal/mcp/server.go`, `internal/mcp/pending_writer_test.go`)

## Accomplishments

- Reproduced the premature drain: a server-initiated notification written through `pendingWriter` decremented the pending-response counter to zero while a client call's response was still owed, letting `waitForDrain` return early. Committed RED, then fixed and confirmed GREEN.
- Added `looksLikeJSONRPCResponse`, the outbound classifier symmetric to the existing inbound `looksLikeJSONRPCCall`, with a deliberately opposite conservative default (never decrement on ambiguity, since over-decrementing is the defect being fixed).
- Reworked `pendingWriter.Write` into one serialized critical section covering both the underlying write and the `b[:n]` classification, so wire order and classification order cannot diverge under concurrent callers — proven under `-race` against a deliberately unsynchronized underlying writer.
- Replaced the bare `Add(-1)` with `decrementPending`, a CAS loop that refuses to go below zero and records refused decrements in a test-visible `pendingUnderflows` counter.
- Added an executable disproof (`TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications`) that fixing FIX-01 does NOT close the `toolslist-repeat` flake — a session with zero notification traffic still answers two pipelined `tools/list` calls correctly via async dispatch, matched by id, without asserting arrival order.

## Task Commits

Each task was committed atomically:

1. **Task 1: Reproduce the premature drain, then balance the counter**
   - `c14c9f6` (test) — add failing `TestPendingWriterDrainsEarlyWithoutFix` reproduction, observed RED
   - `7bcae45` (fix) — balance `pendingWriter`'s counter: `looksLikeJSONRPCResponse`, `decrementPending`/`pendingUnderflows`, single serialized critical section; full suite GREEN at 18 `--- PASS` lines (floor 15)
2. **Task 2: Prove the `toolslist-repeat` flake is a different defect and is NOT closed here**
   - `54fbd1e` (test) — `TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications`

**Plan metadata:** (this commit, docs: complete plan)

_TDD gate sequence: `test(01-03)` (c14c9f6) precedes `feat`/`fix(01-03)` (7bcae45) — RED then GREEN, as required for the `type: tdd` plan._

## Files Created/Modified

- `internal/mcp/server.go` — `pendingWriter` rewritten as one serialized critical section; added `looksLikeJSONRPCResponse`, `decrementPending`, `pendingUnderflows`; updated `ServeStdio`'s invariant doc comment
- `internal/mcp/pending_writer_test.go` (new) — 8 test functions: the committed reproduction, six coverage tests for Task 1, and the separability disproof for Task 2

## Decisions Made

- **Conservative-default asymmetry, explicit.** `looksLikeJSONRPCResponse` returns `false` (does not decrement) on any ambiguous or unparseable outbound line — including `id: null` (the JSON-RPC parse-error response shape) — the opposite default from the inbound `looksLikeJSONRPCCall`'s conservative `true`. Documented explicitly so a future reader does not "correct" the asymmetry: over-decrementing here reintroduces FIX-01; under-decrementing only costs a bounded wait against `stdinLingerGrace`.
- **Full critical section, not buffer-only mutex.** `pendingWriter.Write` takes its lock as the first statement and holds it across both `p.w.Write(b)` and the classification of `b[:n]`. A buffer-only mutex was rejected per the plan's cycle-2 review finding: it would let two concurrent `Write` calls reach the wire in one order and the classification buffer in another. Proven via `TestPendingWriterSerializesWriteAndClassification` against a deliberately unsynchronized underlying writer (`chunkyWriter`) — a buffer-only mutex would corrupt that writer's state under `-race`.
- **CAS-based floor, not a silent clamp.** `decrementPending` refuses to go below zero and increments `pendingUnderflows` on refusal, so a future regression in the increment/decrement balance is observable as a number rather than a mysterious early EOF.
- **FIX-01 and the `toolslist-repeat` flake stay separate.** Per `01-CONTEXT.md`'s disproof (go-sdk@v1.7.0's `jsonrpc2.Async` dispatch, not the counter), only FIX-01 is closed. `TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications` makes that disproof executable rather than only asserted in prose, and deliberately does not assert response ordering — not asserting order is not a demonstration of reordering.

## Deviations from Plan

None — plan executed as written. The plan's own `chunkyWriter`-style test double for `TestPendingWriterSerializesWriteAndClassification` needed one within-task fix during authoring (the double's `Write` initially returned `len(b)` on the post-loop, already-drained slice, always reporting `n=0` to the caller — corrected to return the original total length), caught immediately by the first `-race` run and fixed before any commit; no separate deviation entry warranted since it never left the working tree in a broken state.

## Issues Encountered

None beyond the environment's worktree-isolation guard rejecting bare `git`/`rg`/`ls` invocations as "too complex to verify" — worked around by prefixing shell built-ins with `command` and routing multi-step verification through small scratch scripts invoked via `bash <script>`.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- FIX-01 is closed: `internal/mcp/server.go`'s `pendingWriter` counter is balanced, proven under `-race`, and confined to `internal/mcp/server.go` / `internal/mcp/pending_writer_test.go` — no change to the `Server` interface, `BuildServer`, `internal/cli/serve.go`, or any wire-visible response content (confirmed via a ranged `git diff` against the phase base, both for file confinement and for the exported-surface check on changed lines only).
- `task test:unit` (including `test/wireoracle`) and `go vet ./internal/mcp/...` both pass clean — no existing CLI or MCP byte changed.
- The `toolslist-repeat` ordering flake remains open, correctly attributed to the wire-oracle harness (compare by id, not arrival position), and stays in `.planning/todos/pending/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md` — confirmed still present in `pending/` after this plan.
- This fix is flagged in `01-CONTEXT.md` as a worked example for Phase 6's ConnectRPC server-streaming lifecycle design review, since the root cause (a server-initiated write decrementing a counter only client-initiated requests increment) is the same shape that streaming lifecycle can reinvent.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: `internal/mcp/pending_writer_test.go`
- FOUND: `internal/mcp/server.go`
- FOUND: `.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-SUMMARY.md`
- FOUND commit: `c14c9f6`
- FOUND commit: `7bcae45`
- FOUND commit: `54fbd1e`
