# Wire Oracle Coverage Baseline

**Originally captured:** 2026-08-05 (Phase 1, `01-protocol-scoping-the-sdk-independent-wire-oracle`)
**Last updated:** 2026-08-08 (debug session `mcp-server-one-tool-only` — the `CODEGRAPH_MCP_TOOLS` inversion)
**Scenario count:** 29 (`test/wireoracle.ExpectedScenarioCount`)

This is the human-readable index of the complete, frozen scenario set the wire oracle
(`test/wireoracle`) captures against the real `codegraph` binary. It is **not** a second source of
truth — the structural guarantees it describes already live in code, enforced on every `go test
./test/wireoracle/...` run by four tests:

- `TestScenarioCountIsExact` — the count below is exactly 29, enforced with equality, never a lower
  bound.
- `TestTranscriptSetMatchesScenarioSet` — every scenario named below has exactly one
  `testdata/wireoracle/transcripts/<name>.golden` file, and no orphaned file exists.
- `TestEveryRegisteredToolHasASuccessfulCallScenario` (plan 01-04) — every one of the 8 registered
  MCP tools has a scenario proving a successful `tools/call`.
- `TestLegacyEraBaselineIsDocumented` (plan 01-05) — the six-era Legacy handshake baseline below is
  exactly 6 scenarios, and the four supported revisions negotiate to themselves.

This file exists so a human (or a later phase's planner) can see the complete set at a glance
without reading `scenarios.go`; if this file and the code ever disagree, the code — and the four
tests above — are authoritative.

## History: how this corpus grew across phases

- **Phase 1** (`01-protocol-scoping-the-sdk-independent-wire-oracle`) captured the original 23
  scenarios below against the real, unmodified `mark3labs/mcp-go@v0.56.0`-backed binary — the
  one-way pre-migration baseline. That exact capture (against mark3labs specifically) is closed:
  once Phase 2 removed `mark3labs/mcp-go` from `go.mod`, none of those 23 handshakes could ever be
  re-*captured* against that pre-migration wire behavior again.
- **Phase 2** (`02-sdk-migration-official-go-sdk-on-the-existing-surface`) re-*froze* (not
  re-captured against mark3labs — captured fresh against the new go-sdk-backed binary) all 23 of
  those transcripts through one reviewed-diff pass (02-05), attributing every changed line to one of
  nine named causes. The scenario **set** did not change size in Phase 2; only the bytes each
  scenario's `.golden` file held.
- **Phase 3** (`03-2026-07-28-spec-compliance`) legitimately *extended* the scenario set for the
  first time since Phase 1 — sanctioned explicitly by Phase 1's own D-03 ("transcripts *must*
  legitimately change there," 03-CONTEXT.md D-06) — adding 4 new scenarios across four plans
  (03-01, 03-03, 03-04, 03-05) and one new request appended to an existing scenario
  (`legacy-2024-11-05`, plan 03-05), bringing the total from 23 to 27. Every addition went through
  the same one-reviewed-diff-pass mechanism (D-06) each time the corpus's frozen bytes moved.
- **Phase 5** (`05-live-tool-catalog-change-notification`) added `modern-listen-catalog-change`,
  taking the count from 27 to 28. This file was not updated at the time and read `27` against a
  constant of `28` until the entry below was written; that gap is corrected here rather than left
  standing.
- **Debug session `mcp-server-one-tool-only`** (2026-08-08) inverted `CODEGRAPH_MCP_TOOLS` from an
  opt-in allowlist into an opt-out narrowing filter and made all eight tools the default surface. It
  added `toolslist-filter-empty` (28 → 29), renamed `toolslist-allowlist` to `toolslist-narrowed`
  (no count change), and moved the frozen bytes of most transcripts in the corpus — see the
  tool-visibility table below.

## The complete scenario set, grouped by coverage category

### Tracer (1)

| Scenario | Covers |
|---|---|
| `handshake-explore` | End-to-end: `initialize` → `tools/list` → `tools/call codegraph_explore` with a real query. The oracle architecture's own proof of life (plan 01-01). |

### `tools/list` variants (5)

`CODEGRAPH_MCP_TOOLS` is a NARROWING filter: unset registers all eight tools, and setting it removes
every companion it does not name. These five scenarios are the complete matrix of that gate crossed
with the index gate.

| Scenario | Covers |
|---|---|
| `toolslist-default` | Variable UNSET, index present — all 8 tools. The default surface, and the exact case the "the mcp server is only showing one tool" report landed on. |
| `toolslist-filter-empty` | Variable SET to the empty string — `codegraph_explore` alone. The operator escape hatch back to the pre-inversion surface, and the only wire shape that distinguishes a set variable from an unset one; paired with `toolslist-default` it is the frozen proof the server reads `os.LookupEnv`, not `os.Getenv`. |
| `toolslist-narrowed` | `CODEGRAPH_MCP_TOOLS=node,status` — explore + 2 companions. Renamed from `toolslist-allowlist`: same value, same answer, reversed mechanism. |
| `toolslist-no-index` | No `.codegraph/` present — zero tools even with a filter set, `initialize` still succeeds (MCP-03). The index gate dominates the filter gate. |
| `toolslist-repeat` | Two consecutive `tools/list` calls in one session — determinism probe (`TestToolsListOrderIsDeterministic`) over the default 8-tool surface. |

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
| `edge-call-before-initialize` | `tools/call` sent with no prior `initialize` — post-migration, go-sdk enforces MCP's session-ordering requirement and REJECTS this (`{"code":0,...}`, tracked as upstream go-sdk#976, not anchored — 02-05 cause #9). Pre-migration this scenario proved the opposite (a permissive mark3labs acceptance); the scenario's name and request shape are unchanged across that flip, only the recorded outcome. |

### Six-era Legacy handshake baseline (6)

The multi-era baseline approved at plan 01-05's Task 1 blocking checkpoint (`six-era` selection): the
four protocol revisions the server recognizes, plus the revision Phase 3 implements
(unsupported before Phase 3), plus a request omitting `protocolVersion` entirely.

| Scenario | Offered | Negotiated | Result |
|---|---|---|---|
| `legacy-2025-11-25` | `2025-11-25` | `2025-11-25` | supported, echoed back |
| `legacy-2025-06-18` | `2025-06-18` | `2025-06-18` | supported, echoed back |
| `legacy-2025-03-26` | `2025-03-26` | `2025-03-26` | supported, echoed back |
| `legacy-2024-11-05` | `2024-11-05` | `2024-11-05` | supported, echoed back — **plus a trailing `tools/call codegraph_explore` (plan 03-05, SPEC-06)**: proves the OLDEST Legacy era, not just negotiation, completes a session AND successfully calls a tool. Paired with `handshake-explore`'s equivalent proof at `2025-11-25`, the NEWEST Legacy era, as the two-endpoint judgment call 03-RESEARCH.md's "SPEC-06 recommendation" flagged LOW confidence. |
| `legacy-unsupported-2026-07-28` | `2026-07-28` | `2025-11-25` | silent coercion to the server's own latest — SUCCESS, no `error` object |
| `legacy-omitted-version` | *(no key)* | `2025-03-26` | silent coercion to the server's older backwards-compat default — SUCCESS, distinct from the row above |

### Modern (2026-07-28) discover + tool-call tracer (1) — plan 03-01

| Scenario | Covers |
|---|---|
| `modern-discover-explore` | A sessionless `server/discover` (SEP-2575 `_meta`-carried protocol version, never `params.protocolVersion`) followed by a sessionless `tools/call codegraph_explore`, both Modern. Proves SPEC-01 (discover answers with capabilities, no tool call first), SPEC-03/SPEC-08 (`resultType:"complete"` and `_meta.io.modelcontextprotocol/serverInfo` on both a discover result and a tool result), and SPEC-04's discover half (`cacheScope:"private"`, `ttlMs:0`, independently anchored by `assertDiscoverCacheControl`). SPEC-07's `instructions` field (plan 03-05) also lands on this scenario's discover result. |

### Modern `_meta` validation failures (2) — plan 03-03

| Scenario | Covers |
|---|---|
| `modern-meta-invalid-params` | A well-formed Modern `_meta` missing `io.modelcontextprotocol/clientCapabilities` — `-32602` invalid-params (SPEC-02, hand-authored anchor at response id 1, a `NoInitialize` sessionless scenario). |
| `modern-meta-unsupported-version` | A well-formed Modern `_meta` offering a supported-shape-but-unrecognized protocol version that sorts lexically after `"2026-07-28"` (`"2099-01-01"`, load-bearing for avoiding go-sdk's lexical-comparison reclassification trap) — `-32022` unsupported-protocol-version (SPEC-02, hand-authored anchor). |

### Live catalog-change notification (1) — plan 05-01

| Scenario | Covers |
|---|---|
| `modern-listen-catalog-change` | An opted-in Modern `subscriptions/listen` stream receives `notifications/tools/list_changed` on the SAME live stdio connection after a real mid-session `codegraph init`. Carries an explicit `CODEGRAPH_MCP_TOOLS=` (empty) filter so exactly one tool registers on the transition and the notification frame count stays a structural property rather than a bet on `changeAndNotify`'s 10ms debounce coalescing eight `AddTool` calls. |

### Dynamic tool catalog (1) — plan 03-04

| Scenario | Covers |
|---|---|
| `index-appears-mid-session` | A server started with NO index present advertises zero tools on its first `tools/list`; a REAL `codegraph init` subprocess then runs against the server's own working directory mid-session (via `InitAfterRequest`, a response-observed wait, never a sleep); the SAME live connection's second `tools/list` advertises the full catalog — no restart, no reconnect. Proves SPEC-05's per-request re-check (`internal/mcp/server.go`'s `recheckCatalog`). |

## Total

1 (tracer) + 5 (tools/list) + 7 (tools/call) + 4 (error shapes) + 1 (statelessness edge)
+ 6 (six-era Legacy baseline) + 1 (Modern discover tracer) + 2 (Modern `_meta` failures)
+ 1 (dynamic tool catalog) + 1 (live catalog-change notification) = **29**, matching
`ExpectedScenarioCount`.

## The original 23 mark3labs captures cannot be re-captured against that backend again

The 23 scenarios in the Tracer/`tools/list`/`tools/call`/Error-shapes/Statelessness/Six-era-Legacy
categories above were originally captured against the real, unmodified
`mark3labs/mcp-go@v0.56.0`-backed binary in Phase 1 — the one-way pre-migration baseline. Once Phase
2 removed `mark3labs/mcp-go` from `go.mod`, none of those 23 handshakes could ever be re-captured
against that exact pre-migration wire behavior again — Phase 2 (02-05) re-froze their `.golden`
bytes against the new go-sdk-backed binary instead, through one reviewed-diff pass. This constraint
applied to Phase 2 specifically (a byte-identity comparison across a backend swap); it does not mean
the scenario **set** itself is closed — Phase 3 (03-01, 03-03, 03-04, 03-05) legitimately added the
4 new scenarios and 1 new request documented above, each through the same reviewed-diff mechanism
(D-06) the corpus uses for every frozen-byte change, pre- or post-migration.

## Instruction for whoever next extends this corpus

Any future plan that adds a scenario, adds a request to an existing scenario, or otherwise moves a
frozen `.golden` file's bytes must: (1) bump `ExpectedScenarioCount` in the same commit as the
scenario addition, per its own doc comment's rule; (2) run the oracle's capture CLI against a freshly
rebuilt binary — never hand-write a `.golden` file; (3) route every byte movement through one
reviewed-diff pass per scenario/plan (D-06's mechanism: capture before and after, read the diff, name
every changed line's cause in the commit message — no ledger file, no sign-off step); and (4) update
this file's category tables and Total line in the same change, so this index does not silently fall
out of date with the constant (`ExpectedScenarioCount`) and the code that guard it.
