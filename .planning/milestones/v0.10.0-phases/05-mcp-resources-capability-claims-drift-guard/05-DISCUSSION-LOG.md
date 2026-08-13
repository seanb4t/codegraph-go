# Phase 5: MCP Resources Capability & Claims Drift Guard - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-12
**Phase:** 05-mcp-resources-capability-claims-drift-guard
**Areas discussed:** Resource content depth, Resource split — filter vs. index-state, URI naming scheme

---

## Resource content depth

| Option | Description | Selected |
|--------|-------------|----------|
| Fact-sheet only | Params/defaults/return shape, mirrors jsonschema tags as markdown | ✓ |
| Fact-sheet + worked examples | Adds example calls + expected output shape | |
| Fact-sheet + cross-references | Adds "use this instead of X" pointers | |

**User's choice:** Fact-sheet only.

| Option | Description | Selected |
|--------|-------------|----------|
| Leave "empty until indexed" to the separate resource | Keeps tool docs purely mechanical | ✓ |
| One-line pointer in each tool doc | Cross-reference in every tool doc | |

**User's choice:** Leave it to the separate resource.

| Option | Description | Selected |
|--------|-------------|----------|
| No hard budget, short by discipline | No enforced ceiling, unlike `instructions`' 600-byte cap | ✓ |
| Explicit per-resource byte/line cap, test-enforced | Mirrors `instructions` discipline exactly | |

**User's choice:** No hard budget.

| Option | Description | Selected |
|--------|-------------|----------|
| Out of scope | ToolAnnotations already communicates read-only via protocol | ✓ |
| One shared line, once | Single reassurance sentence in a behavior resource | |

**User's choice:** Out of scope.

**Notes:** No follow-up clarifications beyond the selected options.

---

## Resource split — filter vs. index-state

| Option | Description | Selected |
|--------|-------------|----------|
| Two separate resources | Matches roadmap wording, separate sources of truth | ✓ |
| One combined "server behavior" resource | Fewer URIs, but mixes two code paths in one guard | |

**User's choice:** Two — with the added note that "the env var for tool availability is going away in the near future."

**Notes:** This surfaced new information not previously in ROADMAP.md/REQUIREMENTS.md: `CODEGRAPH_MCP_TOOLS` is planned for removal in a future release (timing unspecified, not this milestone). Followed up directly:

| Option | Description | Selected |
|--------|-------------|----------|
| Document as-is, no special caveat | Current behavior only; removal is a future drift-guard event | ✓ |
| Document as-is + "may be removed" caveat | Same content plus explicit forward-looking note | |

**User's choice:** Document as-is, no special caveat.

| Option | Description | Selected |
|--------|-------------|----------|
| Same directory/convention as the 8 tool docs | One embedded FS, one registration loop | ✓ |
| Separate subdirectory for the 2 behavior docs | Signals the distinction in file layout | |

**User's choice:** Same directory, same convention.

---

## URI naming scheme

| Option | Description | Selected |
|--------|-------------|----------|
| Custom scheme: `codegraph://tools/explore` | Dedicated scheme, visually distinct | ✓ |
| Plain descriptive strings: `docs/codegraph_explore` | No custom scheme, path-like | |

**User's choice:** Custom `codegraph://` scheme.

| Option | Description | Selected |
|--------|-------------|----------|
| `codegraph://tools-filter` + `codegraph://index-state` | Short, avoids naming the soon-to-be-removed env var | ✓ |
| `codegraph://env/CODEGRAPH_MCP_TOOLS` + `codegraph://index-state` | Literal env var name in URI | |

**User's choice:** `codegraph://tools-filter` and `codegraph://index-state`.

| Option | Description | Selected |
|--------|-------------|----------|
| `codegraph://tools/explore` (short/companion name) | Matches `companionNames` vocabulary, no redundant prefix | ✓ |
| `codegraph://tools/codegraph_explore` (full tool name) | Matches registered MCP tool name exactly | |

**User's choice:** Short/companion name.

---

## Claude's Discretion

- Exact markdown structure/headers within each fact-sheet.
- How GUARD-01's existing regex+map pattern extends to cover resource markdown claims vs. an alternative pinning mechanism.
- Wire-oracle scenario additions/re-capture mechanics for `resources/list`/`resources/read`.

## Deferred Ideas

None — the `CODEGRAPH_MCP_TOOLS` removal timeline was surfaced as context (shaping D-05/D-06/D-10 in CONTEXT.md) but is not itself a deferred scope item for a future phase; it belongs to whatever milestone actually removes the variable.
