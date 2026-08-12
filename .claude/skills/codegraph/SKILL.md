---
name: codegraph
description: Use when the user asks where a symbol is defined, how a function or method works, what calls or is called by a symbol, or what breaks if a signature changes in a repository with a .codegraph/ index — routes those questions to codegraph's call and symbol graph instead of grep, find, or file reads.
---

## Which tool for which question

| Question shape | Reach for this |
|---|---|
| Where is X defined? How does Y work? | `codegraph_explore` (MCP tool), or the `codegraph explore "<query>"` CLI as the shell fallback |
| What calls X? What does X call? | `codegraph_callers` / `codegraph_callees` |
| What breaks if I change X's signature? | `codegraph_impact` |
| I don't know the exact symbol name — fuzzy lookup | `codegraph_search` |
| Show one symbol's full detail | `codegraph_node` |
| Which files are involved? | `codegraph_files` |
| Is the index current? | `codegraph_status` |

## When not to reach for codegraph

With no `.codegraph/` directory at the repository root, skip codegraph entirely — use grep, find, and ordinary file reads instead. Indexing is the user's decision, not something to prompt for.

If the registered tool list looks shorter than the table above, `CODEGRAPH_MCP_TOOLS` is narrowing the companion set to a subset of `codegraph_node`, `codegraph_search`, `codegraph_callers`, `codegraph_callees`, `codegraph_impact`, `codegraph_files`, and `codegraph_status` — `codegraph_explore` is never removable by this filter. Unsetting the variable restores the full set.

## Full reference

Every parameter, default, and return shape deliberately absent above lives in a resource this server serves — read it on demand through your MCP resource-read tool rather than trusting this file to restate it:

- `codegraph_explore` → `codegraph://tools/explore`
- `codegraph_node` → `codegraph://tools/node`
- `codegraph_search` → `codegraph://tools/search`
- `codegraph_callers` → `codegraph://tools/callers`
- `codegraph_callees` → `codegraph://tools/callees`
- `codegraph_impact` → `codegraph://tools/impact`
- `codegraph_files` → `codegraph://tools/files`
- `codegraph_status` → `codegraph://tools/status`
- How `CODEGRAPH_MCP_TOOLS` narrows the companion set → `codegraph://tools-filter`
- Preconditions for tool registration based on index presence → `codegraph://index-state`

All 8 tools are documented this way — nothing above restates a parameter, a default, a maximum, or a return shape that one of these resources already owns.
