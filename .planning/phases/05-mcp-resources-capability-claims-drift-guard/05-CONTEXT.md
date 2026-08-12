# Phase 5: MCP Resources Capability & Claims Drift Guard - Context

**Gathered:** 2026-08-12
**Status:** Ready for planning

<domain>
## Phase Boundary

The server documents itself over the wire: `resources/list` + `resources/read` serve tool-by-tool reference content in any repository, indexed or not, with every stated fact derived from source or pinned by a test that can fire. This phase covers the MCP Resources capability itself (RSRC-01/02/03) and the drift guard that keeps its content honest (GUARD-01/02), plus the wire-oracle re-capture that `capabilities.resources` appearing in `initialize` forces. It does NOT cover SKILL.md authoring (Phase 6), the SessionStart nudge (Phase 6), install/uninstall distribution (Phase 7), or the `instructions` wire-string rewrite (Phase 8) — those name these resource URIs but resolve them in later phases.

</domain>

<decisions>
## Implementation Decisions

### Resource content depth
- **D-01:** Each of the 8 per-tool resource docs is a fact-sheet only — the same params/defaults/return-shape facts already carried in the tool's jsonschema tags (`tools.go`), reformatted as markdown prose. No worked examples, no "use this instead of X" cross-references — those risk duplicating what Phase 6's SKILL.md decision table will state, and each additional claim (an example's expected output, a cross-tool recommendation) is a new fact GUARD-01 has to keep pinned.
- **D-02:** The "empty until indexed" caveat is NOT repeated inside each per-tool doc — it lives solely in the separate index-state-preconditions resource (D-04). Keeps each tool doc mechanical and avoids hand-typing the same caveat 8 times.
- **D-03 [informational]:** No hard byte/line budget is enforced on resource docs (unlike the `instructions` wire-string's 600-byte cap). That cap exists because `instructions` is JSON-encoded into every wire-oracle transcript; these are separate `resources/read` responses with no per-transcript inflation cost. Fact-sheet-only content (D-01) stays short by discipline; no new guard is needed for this. Negative decision (explains an absence, not an action) — no plan implements a byte-budget guard because none should exist.
- **D-04:** The fact-sheets do NOT restate that all 8 tools are read-only/safe. `ToolAnnotations` (in `tools.go`) already communicates this via the protocol's own annotation mechanism, readable directly from `tools/list` — restating it in resource prose would be a second copy of a fact GUARD-01 would have to keep in sync with `tools.go` for no client-facing benefit.

### Resource split — filter semantics vs. index-state preconditions
- **D-05:** `CODEGRAPH_MCP_TOOLS` narrowing-filter semantics and index-state preconditions are two SEPARATE resources, not one combined "server behavior" doc — matches the roadmap's own wording (success criterion 1 names them as distinct items) and keeps each resource's drift-guard surface tied to one source of truth (env-var resolution vs. `hasIndex` gating are different code paths in `server.go`).
- **D-06 — user-supplied fact, load-bearing for scope:** `CODEGRAPH_MCP_TOOLS` is planned for removal in a near-future release (not scheduled within this milestone). Phase 5 documents the filter's CURRENT, real behavior with no special "this may go away" caveat — if/when it's actually removed, GUARD-02 catches the resource going stale at that point; hedging about a future removal now would be documenting a decision that hasn't been made yet.
- **D-07:** The 2 behavior docs (filter semantics, index-state preconditions) live in the same directory/`go:embed` convention as the 8 tool docs — one embedded filesystem, one registration loop, no special-casing.

### URI naming scheme
- **D-08:** Resources use a custom `codegraph://` scheme rather than plain descriptive strings (e.g. `docs/codegraph_explore`) — visually distinct from file/http(s) URIs at a glance, consistent with how other MCP servers expose non-file resources.
- **D-09:** Per-tool URIs are `codegraph://tools/<name>` where `<name>` is the short companion/CLI name (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `files`, `status` — matching `companionNames` in `server.go`), not the full registered tool name (`codegraph_explore`). The `codegraph://` prefix already establishes context; repeating `codegraph_` in every segment is redundant.
- **D-10:** The 2 behavior-doc URIs are `codegraph://tools-filter` and `codegraph://index-state` — deliberately NOT `codegraph://env/CODEGRAPH_MCP_TOOLS`, since that would tie the URI to an env var name that D-06 establishes is going away, forcing a URI rename alongside its eventual removal. `codegraph://tools-filter` names what the doc is about, not the mechanism's current implementation detail.

### Claude's Discretion
- Exact markdown structure/headers within each fact-sheet (beyond "fact-sheet only, no examples, no cross-tool recommendations" from D-01).
- How GUARD-01's existing `engineConstantFor` regex+map pattern (`tools_schema_drift_test.go`) extends to cover claims made in the embedded resource markdown, versus a different pinning mechanism for prose content — this is implementation approach, not a vision decision.
- Wire-oracle scenario additions/re-capture mechanics for `resources/list`/`resources/read` (success criterion 5) — follows the existing `test/wireoracle` reviewed-diff discipline (`COVERAGE-BASELINE.md`, `MUTATION-PROOF.md`); planner/researcher own the specifics.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & requirements
- `.planning/ROADMAP.md` §"Phase 5: MCP Resources Capability & Claims Drift Guard" — goal, 5 success criteria, notes on go-sdk stability and the embed/drift-guard pattern to extend
- `.planning/REQUIREMENTS.md` §"MCP Resources Capability" and §"Claims Drift Guard" — RSRC-01/02/03, GUARD-01/02 full text

### Existing drift-guard pattern (GUARD-01/02 extends this)
- `internal/mcp/tools_schema_drift_test.go` — `TestMCPToolSchemaNumericClaimsMatchEngineConstants`: the `engineConstantFor` regex+map pattern that pins numeric claims ("default N", "max N") in tool schemas to source constants, parsed via `go/parser` from `internal/query/validate.go`
- `internal/mcp/instructions_contract_test.go` — `TestInstructionsDescribesEveryVisibilityMechanism` and related: anchor-based pinning for prose claims (not just numerics), and the `docNamesCompanionsWithoutTheFilter` non-vacuity-proven checker pattern

### Wire-oracle re-capture discipline (success criterion 5)
- `test/wireoracle/COVERAGE-BASELINE.md` — human-readable index of all 29 frozen scenarios; documents the reviewed-diff mechanism (capture before/after, name every changed line's cause) every prior capability change has followed
- `test/wireoracle/MUTATION-PROOF.md` — non-vacuity proof discipline for the oracle itself
- `test/wireoracle/scenarios.go`, `oracle_test.go` — scenario definitions and `TestFrozenTranscriptsMatch`

### Resource registration point
- `internal/mcp/server.go` `BuildServer` (~line 485-529) — where `mcp.ServerCapabilities` is constructed unconditionally (D-11 comment explains why); `Capabilities.Resources` must be added here, unconditionally, mirroring how `Capabilities.Tools` is already set regardless of `hasIndex` — this is what makes RSRC-03 (resources register even with zero index-gated tools) structurally correct rather than incidental

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/mcp/tools.go` — every tool's `Description` and `*Args` struct tags (jsonschema-tagged) are the source-of-truth content the 8 per-tool fact-sheets restate as markdown. `companionNames` (`server.go`) is the existing vocabulary for the 7 filterable tool names — D-09's URI naming reuses it directly.
- `internal/mcp/tools_schema_drift_test.go`'s `parseQueryConstants` — `go/parser`-based extraction of unexported constants from `internal/query/validate.go`, already proven for pinning numeric defaults; likely reusable for whatever facts the resource fact-sheets state.

### Established Patterns
- **Drift-guard-by-source-derivation, not by-hand-authoring:** every existing guard test in `internal/mcp` (`tools_schema_drift_test.go`, `instructions_contract_test.go`) reads BOTH the claimed value and the expected value from their real source (`jsonschema.For[XArgs]`, `go/parser` over `validate.go`, `companionNames`) rather than hardcoding either side. GUARD-01/02 should follow this same shape for resource content, not introduce a parallel mechanism.
- **Unconditional capability advertisement (D-11):** `server.go`'s existing comment on `Capabilities.Tools` explains why capabilities must be set explicitly and unconditionally, never left to whatever the SDK defaults to — the same reasoning applies directly to `Capabilities.Resources` for RSRC-03.
- **`go:embed`'d markdown, one FS, one registration loop:** confirmed by D-07 as the convention to follow for all 10 resource files.

### Integration Points
- Resource registration happens in `BuildServer` (`internal/mcp/server.go`), the same function that already conditionally registers tools — resources register unconditionally, before or independent of the `hasIndex` branch that gates tool registration.
- `resources/read` handlers will need the same confinement/read-only posture as `openEngine`'s tool handlers, though resources are static embedded content (no repo path resolution needed) — simpler than the tool-call path.

</code_context>

<specifics>
## Specific Ideas

No specific content examples or "I want it like X" moments beyond the decisions captured above — the user's one substantive addition was the forward-looking fact about `CODEGRAPH_MCP_TOOLS`'s planned removal (D-06), which shaped the split (D-05) and URI naming (D-10) decisions but was explicitly kept out of this phase's documented content.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. `CODEGRAPH_MCP_TOOLS` removal itself is not deferred as a phase-backlog item here (it belongs to whatever future milestone actually removes it); it was surfaced only as context shaping this phase's decisions.

</deferred>

---

*Phase: 5-MCP Resources Capability & Claims Drift Guard*
*Context gathered: 2026-08-12*
