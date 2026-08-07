---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 04
subsystem: testing
tags: [mcp, jsonrpc, stdio, wire-protocol, golden-transcript, mark3labs-mcp-go, spec-anchors]

# Dependency graph
requires:
  - phase: 01-01
    provides: "test/wireoracle package architecture (Capture/Normalize/Scenario/oracle_test.go), the dedicated fixture, and the tracer's handshake-explore scenario this plan extends"
provides:
  - "16 additional frozen pre-migration transcripts (testdata/wireoracle/transcripts/*.golden), bringing the suite to exactly 17 scenarios — all 8 registered MCP tools, all three tools/list variants plus a determinism probe, four error shapes, and one statelessness edge"
  - "Scenario.NoInitialize (test/wireoracle/capture.go) — the struct field the tracer's own downstream invariants (session-line, protocolVersion anchor) key off for a no-initialize scenario"
  - "test/wireoracle/anchors.go — Anchor/Anchors(), the two hand-authored JSON-RPC error-code constants (-32601, -32602), independent of the SDK under test"
  - "TestSpecAnchorsHold, TestToolsListOrderIsDeterministic, TestToolsListExactSets, TestEveryRegisteredToolHasASuccessfulCallScenario"
  - "A documented, verified concurrency-ordering constraint for mark3labs v0.56.0's stdio transport (tools/call is queued async; every other method is synchronous) that any future scenario author (plan 05, plan 07) must respect"
affects: [02-sdk-migration, 05-multi-era-baseline, 07-mutation-matrix]

# Actuals (#2632)
actuals:
  tokens: 13975
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "At most one tools/call request per scenario, always last — mark3labs v0.56.0's stdio transport (server/stdio.go processMessage) dispatches tools/call onto an async worker-pool queue while every other method is handled synchronously inline; violating this ordering produces run-to-run-flaky byte comparisons, not a real server regression (verified by hand: repeated captures of a 5-request script showed the two concurrent tools/call responses swapping order across runs)"
    - "Spec-pinned JSON-RPC error-code anchors live in a package file that stays free of the SDK-under-test's transitive dependency graph; the ProtocolVersion anchor specifically stays in a _test.go file (not the production anchors.go) because internal/mcp's other files import mark3labs/mcp-go and Go resolves imports at the package level — referencing even the one SDK-free symbol from a non-test file would leak the SDK into `go list -deps` for this package"

key-files:
  created:
    - test/wireoracle/anchors.go
    - testdata/wireoracle/transcripts/toolslist-default.golden
    - testdata/wireoracle/transcripts/toolslist-allowlist.golden
    - testdata/wireoracle/transcripts/toolslist-no-index.golden
    - testdata/wireoracle/transcripts/toolslist-repeat.golden
    - testdata/wireoracle/transcripts/call-node.golden
    - testdata/wireoracle/transcripts/call-search.golden
    - testdata/wireoracle/transcripts/call-callers.golden
    - testdata/wireoracle/transcripts/call-callees.golden
    - testdata/wireoracle/transcripts/call-impact.golden
    - testdata/wireoracle/transcripts/call-files.golden
    - testdata/wireoracle/transcripts/call-status.golden
    - testdata/wireoracle/transcripts/error-unknown-method.golden
    - testdata/wireoracle/transcripts/error-unknown-tool.golden
    - testdata/wireoracle/transcripts/error-malformed-args.golden
    - testdata/wireoracle/transcripts/error-confinement-reject.golden
    - testdata/wireoracle/transcripts/edge-call-before-initialize.golden
  modified:
    - test/wireoracle/scenarios.go
    - test/wireoracle/capture.go
    - test/wireoracle/oracle_test.go

key-decisions:
  - "Task 1 blocking checkpoint: human selected full-bar (the D-05 bar exactly as scoped — no additional scenarios). 16 scenarios added, suite at exactly 17, matching ExpectedScenarioCount for this plan's scope."
  - "error-malformed-args uses a different request shape than the plan's original prose ('wrong-type argument on a registered tool'): verified empirically against the built binary that BuildServer never enables WithInputSchemaValidation, so a wrong-type argument value on a real registered tool always resolves to a SUCCESSFUL JSON-RPC response with result.isError=true — never a top-level error. The only request_handler.go path returning mcp.INVALID_PARAMS for tools/call is 'tool not found' (server.go:1936-1942). The scenario now sends params.name=\"\" (empty/omitted) with params.arguments as a bare JSON string instead of an object — a genuinely malformed tools/call request that resolves to \"tool '' not found\" => -32602, matching Task 3's hand-authored anchor. Documented at length in scenarios.go beside the scenario."
  - "Discovered and worked around a load-bearing concurrency constraint: mark3labs v0.56.0's stdio transport queues every tools/call onto a worker pool and returns immediately without waiting, while every other method (initialize, tools/list, unrecognized methods) is handled synchronously inline before the next stdin line is read. A synchronous request queued after an async tools/call can be written to stdout BEFORE that tools/call's own response; two tools/call requests in one scenario race each other. Verified by hand: a 5-request script produced non-deterministic relative ordering between its two tools/call responses across 5 repeated runs. Every scenario in this plan therefore carries at most one tools/call request, always the last request — documented as a load-bearing invariant in scenarios.go for future plans (05, 07) to respect."
  - "assertProtocolVersionAnchor stays in oracle_test.go (a _test.go file), not anchors.go: internal/mcp's other files import mark3labs/mcp-go, so a non-test-file reference to internal/mcp.ProtocolVersion would transitively leak the SDK under test into test/wireoracle's own `go list -deps` output, breaking VRFY-01's dependency guard. Verified empirically (grep on go list -deps output, clean both before and after)."
  - "TestSpecAnchorsHold captures each of the 17 scenarios exactly once (not once per matching anchor) — the framing invariant and any matching named anchor(s) both run against that single capture, keeping subprocess-spawn cost proportional to scenario count, not anchor count."

patterns-established:
  - "Scenario.NoInitialize boolean field (not a scenario-name special case) gates the two invariants (VRFY-03 session line, D-02 protocolVersion anchor) that only make sense downstream of a real initialize handshake."
  - "toolCallRequest/toolsListRequest/initializeRequest helper constructors in scenarios.go for the 16 new scenarios (handshake-explore itself stays hand-inlined and untouched per must_haves)."

requirements-completed: [VRFY-01, VRFY-04]

coverage:
  - id: D1
    description: "16 additional scenarios (all 8 registered MCP tools, three tools/list variants plus a determinism probe, four error shapes, one statelessness edge) frozen against the pre-migration mark3labs v0.56.0 binary — suite at exactly 17 scenarios"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestFrozenTranscriptsMatch"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestEveryRegisteredToolHasASuccessfulCallScenario"
        status: pass
    human_judgment: false
  - id: D2
    description: "Hand-authored spec anchors (method-not-found -32601, invalid-params -32602, protocolVersion) that fail independently of the frozen transcripts, plus exact-set/exact-zero tools/list assertions and a determinism probe for repeated tools/list calls"
    requirement: "VRFY-01"
    verification:
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestSpecAnchorsHold"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestToolsListExactSets"
        status: pass
      - kind: unit
        ref: "test/wireoracle/oracle_test.go#TestToolsListOrderIsDeterministic"
        status: pass
    human_judgment: false

# Metrics
duration: ~90min (continuation session; not precisely timestamped from a checkpoint resume)
completed: 2026-08-05
status: complete
---

# Phase 01 Plan 04: Full D-05 Coverage Bar Summary

**Expanded the wire oracle from 1 to 17 scenarios — all 8 MCP tools, three tools/list variants, four error shapes, and one statelessness edge — frozen against the pre-migration mark3labs v0.56.0 binary, plus hand-authored spec anchors that fail independently of the captured bulk.**

## Performance

- **Tasks:** 3 (Task 1 checkpoint resolved on resume; Tasks 2-3 executed this session)
- **Files modified:** 3 (scenarios.go, capture.go, oracle_test.go)
- **Files created:** 17 (anchors.go + 16 frozen `.golden` transcripts)

## Accomplishments
- Task 1 (blocking checkpoint) closed on the human's `full-bar` selection: 16 scenarios added, no discretionary extras, matching CONTEXT D-05's locked scope.
- 16 new scenarios captured and frozen against the real, unmodified `mark3labs`-backed binary, each read and verified by hand before trusting it: all 8 registered tools (`codegraph_explore` already covered by the tracer's `handshake-explore`), all three `tools/list` variants (explore-only default, node+status allowlist, zero-tools-no-index) plus a repeated-`tools/list` determinism probe, four error shapes (unknown method, unknown tool, malformed args, confinement reject), and one statelessness edge (`tools/call` with no prior `initialize`).
- `len(Scenarios())` is exactly 17 (verified via `rg -c` and via `go test`'s own scenario-count assertion path).
- Hand-authored spec anchors (`anchors.go`): `codeMethodNotFound = -32601`, `codeInvalidParams = -32602`, asserted independently of the frozen transcripts against freshly captured stdout.
- `TestEveryRegisteredToolHasASuccessfulCallScenario` derives the 8 registered tool names from a real capture and structurally proves each has a successful (`isError=false`) `tools/call` scenario — no prose delegation.
- `TestToolsListOrderIsDeterministic` byte-compares two consecutive `tools/list` calls' `result.tools` arrays.
- `TestToolsListExactSets` asserts exact-set/exact-zero for all three `tools/list` variants, mirroring `internal/mcp/server_test.go`'s `equalStrings` shape.
- Discovered, verified by hand, and documented a load-bearing concurrency ordering constraint in mark3labs v0.56.0's stdio transport (tools/call is queued asynchronously; every other method is synchronous) that shapes every scenario's request-list design in this plan and must be respected by future plans.
- Both new failure paths (`TestSpecAnchorsHold`, `TestEveryRegisteredToolHasASuccessfulCallScenario`) proven non-vacuous by deliberate mutation, observed red, then reverted.

## Per-Scenario Captured Outcome

| Scenario | Outcome |
|---|---|
| `toolslist-default` | success — `tools/list` advertises exactly `["codegraph_explore"]` |
| `toolslist-allowlist` | success — exactly `["codegraph_explore","codegraph_node","codegraph_status"]` |
| `toolslist-no-index` | success — `initialize` succeeds, `tools/list` returns `[]` |
| `toolslist-repeat` | success — two `tools/list` calls, byte-identical `result.tools` (8 tools) |
| `call-node` | success — `codegraph_node {symbol:"Alpha"}` |
| `call-search` | success — `codegraph_search {query:"Alpha"}` |
| `call-callers` | success — `codegraph_callers {symbol:"Beta"}` (Alpha is Beta's one caller) |
| `call-callees` | success — `codegraph_callees {symbol:"Alpha"}` (Beta, Helper) |
| `call-impact` | success — `codegraph_impact {symbol:"Beta"}` |
| `call-files` | success — `codegraph_files {}` |
| `call-status` | success — `codegraph_status {}` |
| `error-unknown-method` | named error `-32601` (method-not-found) |
| `error-unknown-tool` | named error `-32602` (tool 'codegraph_bogus_tool' not found) |
| `error-malformed-args` | named error `-32602` (tool '' not found — see Deviations) |
| `error-confinement-reject` | `isError=true`, portable rejected-path message, no host path leaked |
| `edge-call-before-initialize` | success — `codegraph_explore` succeeds with no prior `initialize`, no session line emitted |

## Task Commits

1. **Task 1: One-way gate checkpoint** — resolved on resume, human selected `full-bar`; no commit (decision only, per the plan's checkpoint semantics).
2. **Task 2: Script and freeze the full D-05 scenario set** — `404c999` (feat)
3. **Task 3: Hand-authored spec anchors and exact-set list assertions** — `c32cab9` (test)

## Files Created/Modified
- `test/wireoracle/scenarios.go` — 16 new scenarios, `initializeRequest`/`toolsListRequest`/`toolCallRequest` helpers, the documented concurrency-ordering constraint
- `test/wireoracle/capture.go` — added `Scenario.NoInitialize`
- `test/wireoracle/oracle_test.go` — `NoInitialize` gating in `TestFrozenTranscriptsMatch`, `assertNoSessionLine`, `assertFramingInvariant`, `TestSpecAnchorsHold`, `TestToolsListOrderIsDeterministic`, `toolNamesFromCapture`, `equalStrings`, `TestToolsListExactSets`, `findToolCallRequest`, `isSuccessfulToolCall`, `TestEveryRegisteredToolHasASuccessfulCallScenario`
- `test/wireoracle/anchors.go` (new) — `Anchor`, `Anchors()`, `codeMethodNotFound`/`codeInvalidParams`, `assertErrorCode`
- `testdata/wireoracle/transcripts/*.golden` (16 new files) — frozen pre-migration transcripts

## Decisions Made
See `key-decisions` in frontmatter for the full rationale on each. Summary:
- Human selected `full-bar` at Task 1's checkpoint — exact D-05 scope, no additions.
- `error-malformed-args` redesigned around the server's real -32602 trigger (empty tool name + non-object arguments), not the plan's originally-assumed "wrong-type argument on a registered tool" shape, which never produces a JSON-RPC-level error in this server's configuration.
- Discovered and worked around mark3labs v0.56.0's async tools/call dispatch — every scenario carries at most one tools/call request, always last.
- `assertProtocolVersionAnchor` stays test-only to avoid leaking the SDK-under-test into `test/wireoracle`'s own `go list -deps` (VRFY-01).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] error-malformed-args's originally-planned request shape never produces the required -32602**
- **Found during:** Task 2 (scripting the four error shapes)
- **Issue:** Plan's Task 2 prose specified `error-malformed-args` as "tools/call on a registered tool with an argument of the wrong JSON type," with Task 3 requiring the captured response's `error.code == -32602`. Verified empirically against the built binary (both by direct experimentation and by reading `mark3labs/mcp-go@v0.56.0/server/server.go`/`request_handler.go` source) that `BuildServer` never enables `WithInputSchemaValidation`, so a wrong-type argument value sent against a real registered tool name always resolves to a SUCCESSFUL JSON-RPC response (`result.isError=true`), never a top-level `error` field. The only `request_handler.go` path producing `mcp.INVALID_PARAMS` for `tools/call` is "tool not found" (`server.go:1936-1942`) — the identical mechanism `error-unknown-tool` already exercises.
- **Fix:** Redesigned the scenario's request to `params.name=""` (empty) with `params.arguments` sent as a bare JSON string (`"not-an-object"`) instead of an object — a structurally distinct, still genuinely malformed `tools/call` request that resolves to `"tool '' not found"` => `-32602`, matching Task 3's hand-authored anchor. Verified by hand against the real binary before freezing; documented at length in `scenarios.go` beside the scenario declaration.
- **Files modified:** `test/wireoracle/scenarios.go`, `testdata/wireoracle/transcripts/error-malformed-args.golden`
- **Verification:** `TestFrozenTranscriptsMatch/error-malformed-args` and `TestSpecAnchorsHold/error-malformed-args/error.code_==_invalid-params_(-32602)` both pass.
- **Committed in:** `404c999` (Task 2 commit)

**2. [Rule 3 - Blocking] TestFrozenTranscriptsMatch's unconditional session-line/protocolVersion checks broke for a no-initialize scenario**
- **Found during:** Task 2 (designing edge-call-before-initialize)
- **Issue:** `TestFrozenTranscriptsMatch` unconditionally called `assertSessionLine` and `assertProtocolVersionAnchor` for every scenario. `edge-call-before-initialize` sends no `initialize` request at all (its id=1 response is a `tools/call` result), so both checks would incorrectly fail against a scenario the plan itself commissions as a currently-passing behavior worth locking in.
- **Fix:** Added `Scenario.NoInitialize bool` (in `capture.go`, per the plan's own instruction to add a boolean field rather than special-case by scenario name) and gated both checks on it in `TestFrozenTranscriptsMatch`, adding `assertNoSessionLine` for the negative case.
- **Files modified:** `test/wireoracle/capture.go`, `test/wireoracle/oracle_test.go`
- **Verification:** `TestFrozenTranscriptsMatch/edge-call-before-initialize` passes; all other 16 scenarios' existing assertions unaffected.
- **Committed in:** `404c999` (Task 2 commit)

**3. [Rule 3 - Blocking] Anchors.go importing internal/mcp would have leaked the SDK-under-test into go list -deps**
- **Found during:** Task 3 (writing the protocolVersion anchor into `anchors.go`)
- **Issue:** First draft of `anchors.go` imported `internal/mcp` to reference `internalmcp.ProtocolVersion` for the protocolVersion anchor. `internal/mcp`'s other files (`server.go`, `tools.go`) import `github.com/mark3labs/mcp-go`; Go resolves dependencies at the package level, so this would have transitively pulled the SDK under test into `test/wireoracle`'s own `go list -deps` output — violating Task 3's own acceptance criterion (VRFY-01: "the oracle still never uses the SDK under test as its own decoder").
- **Fix:** Kept `assertProtocolVersionAnchor` in `oracle_test.go` (a `_test.go` file, whose imports are invisible to `go list -deps` on the plain package) instead of moving it into `anchors.go`. `Anchors()` returns only the two numeric-code anchors that need no `internal/mcp` import; `TestSpecAnchorsHold` calls `assertProtocolVersionAnchor` directly for `handshake-explore` rather than through the `Anchors()` slice.
- **Files modified:** `test/wireoracle/anchors.go`, `test/wireoracle/oracle_test.go`
- **Verification:** `go list -deps -f '{{.ImportPath}}' github.com/seanb4t/codegraph-go/test/wireoracle | rg mark3labs` returns no matches (confirmed both before and after the fix).
- **Committed in:** `c32cab9` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (1 bug/verified-behavior mismatch, 2 blocking)
**Impact on plan:** All three fixes were necessary for correctness — a plan-assumed server behavior that doesn't exist, and two structural bugs (a broken test invariant, a dependency-graph leak) that would have made this plan's own acceptance criteria unsatisfiable as originally specified. No scope creep — no scenario count changed (still exactly 17), no architectural change.

## Issues Encountered

**Non-deterministic response ordering discovered via mcp-go's stdio worker pool.** While experimenting with multi-tools/call scenario shapes, discovered mark3labs v0.56.0's stdio transport (`server/stdio.go`) dispatches every `tools/call` onto an async worker-pool queue, while every other method is handled synchronously inline. Two tools/call requests, or a tools/call followed by a later synchronous request, in the same scenario can complete in a run-to-run-varying relative order — confirmed by running an identical 5-request script 5 times and observing the two `tools/call` responses swap position. Resolved by constraining every scenario in this plan to at most one `tools/call` request, always last (mirroring the tracer's own proven shape) — documented prominently in `scenarios.go` as a load-bearing invariant for future scenario authors (plans 05, 07).

## Wall-clock measurement (Task 2 acceptance criterion)

`task test:wireoracle` (== `go test ./test/wireoracle/...`) at 17 scenarios: **~11.8s** (go test's own reported time; ~12.6s wall including `task` CLI overhead), measured via `time task test:wireoracle`.

Plan 01's own recorded 1-scenario baseline (01-01-SUMMARY.md): ~5.8s wall-clock.

Marginal cost: roughly (11.8 - 5.8) / 16 ≈ **0.38s per additional scenario** — consistent with each scenario capture spawning 1-2 short-lived subprocesses (`codegraph init` for `Index: true` scenarios, `codegraph serve --mcp`) plus the additional re-captures performed by `TestSpecAnchorsHold`, `TestToolsListExactSets`, `TestToolsListOrderIsDeterministic`, and `TestEveryRegisteredToolHasASuccessfulCallScenario` (each re-captures a subset of the 17 scenarios independently, per this package's existing "every capture is its own subprocess" discipline — VRFY-01 concurrency edge).

## Mutation-testing proofs (Task 3 acceptance criterion)

**TestSpecAnchorsHold non-vacuity:** Temporarily changed `codeMethodNotFound` from `-32601` to `-99999` in `anchors.go`. Result:
```
oracle_test.go:388: scenario "error-unknown-method": error.code = -32601, want -99999: "{\"jsonrpc\":\"2.0\",\"id\":2,\"error\":{\"code\":-32601,\"message\":\"Method wireoracle/unimplemented-method not found\"}}"
--- FAIL: TestSpecAnchorsHold/error-unknown-method/error.code_==_method-not-found_(-32601)
```
Reverted; `go test ./test/wireoracle/... -run TestSpecAnchorsHold` passes green afterward (confirmed).

**TestEveryRegisteredToolHasASuccessfulCallScenario non-vacuity:** Temporarily renamed `call-node`'s tool-call target from `"codegraph_node"` to `"codegraph_node_MUTATION_TEST"` in `scenarios.go`. Result:
```
oracle_test.go:602: no scenario provides a successful tools/call for: [codegraph_node]
--- FAIL: TestEveryRegisteredToolHasASuccessfulCallScenario
```
Reverted; full suite (`go test ./test/wireoracle/... -count=1`) passes green afterward (confirmed, 12.1s).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The wire oracle now carries the full D-05 pre-migration baseline this plan was scoped to capture (17 of the phase's eventual 23 scenarios; plan 05 adds the remaining 6 multi-era scenarios).
- `ExpectedScenarioCount` (declared in plan 07) must be updated to include plan 05's additional 6 scenarios when that plan lands — this plan's own scenarios and their exact count (17) are locked and frozen.
- The documented concurrency-ordering constraint (at most one `tools/call` per scenario, always last) is a hard requirement for plan 05's own scenario additions.
- No blockers for plan 05 or plan 06/07 identified.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*
