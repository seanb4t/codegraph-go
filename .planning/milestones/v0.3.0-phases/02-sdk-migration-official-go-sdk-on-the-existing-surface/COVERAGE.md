# API Coverage — Model Context Protocol (via `modelcontextprotocol/go-sdk@v1.7.0`)

> Full coverage by default. Opt-outs are explicit, reasoned decisions.

**Scope note.** The `api-coverage` detector fired on this phase (`wire` + `mcp`).
The thing being adopted is a *library*, not a remote service — but MCP itself has
a real server-side capability surface that codegraph deliberately subsets, so the
matrix below records that surface rather than a "no external API" declaration.
The opt-outs are not new to this phase: they were measured and recorded during
the v0.3.0 scoping work (backlog 999.6) and re-confirmed in Phase 1.

Phase 2 changes **which library implements** these capabilities. It does not
change which capabilities codegraph offers — that invariance is the phase's whole
point (SDK-01).

| capability | decision | reason |
|---|---|---|
| tools | INTEGRATE | The entire product surface. All 8 tools (`codegraph_explore` + 7 companions) migrate; SDK-01 verifies none changed semantically. |
| tools/listChanged | INTEGRATE | Advertised today via mark3labs' `WithToolCapabilities(true)`. Must be preserved explicitly under go-sdk — see CONTEXT.md D-11, where the SDK omits the whole `tools` key at zero registered tools. |
| stdio transport | INTEGRATE | codegraph's sole transport (`serve --mcp`). |
| resources | OPT-OUT | Never implemented; codegraph exposes queries as tools, not as addressable resources. Out of scope for v0.3.0. |
| prompts | OPT-OUT | Never implemented; no prompt templates in the product. |
| logging | OPT-OUT | Deprecated by SEP-2577 in the `2026-07-28` revision. Diagnostics go to stderr as a plain session line (VRFY-03), not through the protocol. |
| sampling | OPT-OUT | Deprecated by SEP-2577. Client-side capability; codegraph never requests model completions. |
| roots | OPT-OUT | Deprecated by SEP-2577. codegraph resolves its own index root from the filesystem. |
| completions | OPT-OUT | Never implemented; no argument-autocomplete surface. |
| server/discover | OPT-OUT | `2026-07-28` SEP-2575. Explicitly **Phase 3** scope, not Phase 2 — this phase inherits negotiation from the dependency and implements none of the revision's obligations. |
| subscriptions/listen | OPT-OUT | Explicitly **Phase 5** scope. Sequenced last and deliberately isolated so slipping it cannot block the milestone. |
| tasks (`io.modelcontextprotocol/tasks`) | OPT-OUT | Not applicable — codegraph's tools are fast read-only queries. Recorded as a deferred requirement (TASK-01) at v1.0 close. |
| elicitation (`resultType: "input_required"`) | OPT-OUT | New product behavior, not protocol currency. Deferred as MRTR-01 at v1.0 close. |
| streamable HTTP transport | OPT-OUT | Deprecated for new use; codegraph is stdio-only, which is why most of the `2026-07-28` deprecation list misses this project entirely. |
