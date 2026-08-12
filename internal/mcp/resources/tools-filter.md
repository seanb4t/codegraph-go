# CODEGRAPH_MCP_TOOLS

`CODEGRAPH_MCP_TOOLS` narrows which of the 7 companion tools register alongside
`codegraph_explore`. `codegraph_explore` is never removable by this filter —
it is always visible whenever an index resolves.

The 7 companion tools it can narrow: `codegraph_node`, `codegraph_search`,
`codegraph_callers`, `codegraph_callees`, `codegraph_impact`,
`codegraph_files`, `codegraph_status`.

## Behavior

- **Unset** — all 7 companion tools register, for 8 tools in total.
- **Set to a comma-separated list** (e.g. `node,status`) — only the companions
  it names register.
- **Set to the empty string** — no companion registers at all, leaving
  `codegraph_explore` alone.

Whether the variable is SET is distinct from what it holds: unset and
set-to-empty are two different answers, which is why the server reads
presence rather than value.

## Entry parsing

Each comma-separated entry is trimmed of surrounding whitespace. An
unrecognized name is ignored, with a warning written to standard error; an
unrecognized name never fails startup.

## Result

This document is markdown text describing server behavior, not a tool
result.
