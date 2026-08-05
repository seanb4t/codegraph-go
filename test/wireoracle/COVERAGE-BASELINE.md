# Wire Oracle Coverage Baseline — End of Phase 1

**Date:** 2026-08-05
**Phase:** 01-protocol-scoping-the-sdk-independent-wire-oracle
**Scenario count:** 23 (`test/wireoracle.ExpectedScenarioCount`)

This is the human-readable index of the complete, frozen pre-migration scenario set the wire oracle
(`test/wireoracle`) captured against the real, unmodified `mark3labs/mcp-go@v0.56.0`-backed
`codegraph` binary across plans 01, 04, and 05 of this phase. It is **not** a second source of
truth — the structural guarantees it describes already live in code, enforced on every `go test
./test/wireoracle/...` run by four tests:

- `TestScenarioCountIsExact` — the count below is exactly 23, enforced with equality, never a lower
  bound.
- `TestTranscriptSetMatchesScenarioSet` — every scenario named below has exactly one
  `testdata/wireoracle/transcripts/<name>.golden` file, and no orphaned file exists.
- `TestEveryRegisteredToolHasASuccessfulCallScenario` (plan 04) — every one of the 8 registered MCP
  tools has a scenario proving a successful `tools/call`.
- `TestLegacyEraBaselineIsDocumented` (plan 05) — the six-era Legacy handshake baseline below is
  exactly 6 scenarios, and the four supported revisions negotiate to themselves.

This file exists so a human (or a later phase's planner) can see the complete set at a glance
without reading `scenarios.go`; if this file and the code ever disagree, the code — and the four
tests above — are authoritative.

## The complete scenario set, grouped by coverage category

### Tracer (1)

| Scenario | Covers |
|---|---|
| `handshake-explore` | End-to-end: `initialize` → `tools/list` → `tools/call codegraph_explore` with a real query. The oracle architecture's own proof of life (plan 01). |

### `tools/list` variants (4)

| Scenario | Covers |
|---|---|
| `toolslist-default` | No allowlist env — explore-only default (MCP-01). |
| `toolslist-allowlist` | `CODEGRAPH_MCP_TOOLS=node,status` — explore + 2 companions. |
| `toolslist-no-index` | No `.codegraph/` present — zero tools, `initialize` still succeeds (MCP-03). |
| `toolslist-repeat` | Two consecutive `tools/list` calls in one session — determinism probe (`TestToolsListOrderIsDeterministic`), all 8 tools allowlisted. |

### `tools/call` per registered tool (7)

`codegraph_explore` itself is covered by the tracer's `handshake-explore` above —
`TestEveryRegisteredToolHasASuccessfulCallScenario` enforces this structurally rather than by prose.

| Scenario | Tool |
|---|---|
| `call-node` | `codegraph_node` |
| `call-search` | `codegraph_search` |
| `call-callers` | `codegraph_callers` |
| `call-callees` | `codegraph_callees` |
| `call-impact` | `codegraph_impact` |
| `call-files` | `codegraph_files` |
| `call-status` | `codegraph_status` |

### Error shapes (4)

| Scenario | Covers |
|---|---|
| `error-unknown-method` | Unrecognized JSON-RPC method — `-32601` method-not-found (hand-authored anchor). |
| `error-unknown-tool` | `tools/call` naming an unregistered tool — `-32602` invalid-params, "tool not found" path. |
| `error-malformed-args` | Structurally malformed `tools/call` (empty `name`, non-object `arguments`) — `-32602` invalid-params via the same "tool not found" mechanism (hand-authored anchor). |
| `error-confinement-reject` | `codegraph_node` with a `path` argument resolving outside the confinement root — CR-02 trust-boundary rejection. |

### Statelessness edge (1)

| Scenario | Covers |
|---|---|
| `edge-call-before-initialize` | `tools/call` sent with no prior `initialize` — today's server tolerates this (RESEARCH Pitfall 2), locked in as a currently-passing behavior, not an error. |

### Six-era Legacy handshake baseline (6)

The multi-era baseline approved at plan 05's Task 1 blocking checkpoint (`six-era` selection): the
four protocol revisions today's server recognizes, plus the revision Phase 3 will implement
(unsupported today), plus a request omitting `protocolVersion` entirely.

| Scenario | Offered | Negotiated | Result |
|---|---|---|---|
| `legacy-2025-11-25` | `2025-11-25` | `2025-11-25` | supported, echoed back |
| `legacy-2025-06-18` | `2025-06-18` | `2025-06-18` | supported, echoed back |
| `legacy-2025-03-26` | `2025-03-26` | `2025-03-26` | supported, echoed back |
| `legacy-2024-11-05` | `2024-11-05` | `2024-11-05` | supported, echoed back |
| `legacy-unsupported-2026-07-28` | `2026-07-28` | `2025-11-25` | silent coercion to the server's own latest — SUCCESS, no `error` object |
| `legacy-omitted-version` | *(no key)* | `2025-03-26` | silent coercion to the server's older backwards-compat default — SUCCESS, distinct from the row above |

## Total

1 (tracer) + 4 (tools/list) + 7 (tools/call) + 4 (error shapes) + 1 (statelessness edge)
+ 6 (six-era Legacy baseline) = **23**, matching `ExpectedScenarioCount`.

## This set cannot be extended after Phase 2 removes `mark3labs/mcp-go`

Every scenario above was captured against the real, unmodified `mark3labs/mcp-go@v0.56.0`-backed
binary — the one-way pre-migration baseline this milestone's wire oracle exists to freeze. Once
Phase 2 removes `mark3labs/mcp-go` from `go.mod`, none of these 23 handshakes can ever be
re-captured against that exact pre-migration wire behavior again. This set is closed: Phase 2 and
Phase 3 read it as a fixed comparison target (via `TestFrozenTranscriptsMatch`'s byte-for-byte
guard), they do not add new frozen `.golden` transcripts to it. New wire-behavior coverage after the
SDK swap belongs in a new, post-migration comparison mechanism — not in
`testdata/wireoracle/transcripts/`.

## Instruction for whoever plans Phase 2

Phase 2's first plan should declare an explicit dependency on this file and on the four tests named
above (`TestScenarioCountIsExact`, `TestTranscriptSetMatchesScenarioSet`,
`TestEveryRegisteredToolHasASuccessfulCallScenario`, `TestLegacyEraBaselineIsDocumented`) — not on
"Phase 1 complete". A Phase 1 plan cannot create a Phase 2 dependency itself, which is why this
instruction is recorded here rather than implemented as a structural gate.
