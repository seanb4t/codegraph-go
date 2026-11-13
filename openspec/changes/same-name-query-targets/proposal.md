# Proposal
Issue: #318

## Why

If several nodes share a name and are not in an overload set, `callers`, `callees` and `impact` answer for one definition. The pick is the node with the lowest id. The id is a hash, so the pick is arbitrary, and the output does not say that other definitions exist.

On the locked `psf/requests` corpus, `callers to_key_val_list` picks a typing stub and prints no caller, although the real function has callers. An agent reads "0 callers" as "safe to change", so this answer causes wrong edits.

## What Changes

- `callers`, `callees` and `impact` answer for every definition of a bare name. This is the rule they already follow for the members of an overload set.
- If more than one definition answers, the output lists the definitions, and each result names the definition it belongs to. The CLI text, the `--json` output and the MCP tool results all carry this.
- `callers`, `callees` and `impact` accept `--file <path>`, which selects the definitions in the files that match the value, as `node --file` matches. The MCP tools `codegraph_callers`, `codegraph_callees` and `codegraph_impact` accept a `file` input with the same meaning. `node` already has this flag.
- A name with one definition gives the same output as today, byte for byte.
- The order of the rows no longer depends on node ids. For a query over one overload set, rows that tie on file, name and line can change their order once.

No change is **BREAKING**. The JSON fields are additive, and they are absent for a name with one definition. A query for a name with several definitions returns more rows than today, because today's answer leaves rows out.

## Out of scope

- `node`, `search` and `explore`. `node` already reports each definition of a name.
- A definition label and a file selector in the graph UI. The UI gets the rows for every definition. design.md D8 gives the reason, and a follow-up issue tracks the label.
- The index. The stored nodes and edges are correct, and this change reads them only.
- A preference for the last definition of a Python, Ruby or JavaScript name that a module defines more than once. The answer for every definition already holds the real callers.
- The `--line` flag that the requirement "A signature selects one overload" names for `callers`, `callees` and `impact`. Those three commands do not have it today. design.md records the conflict, and this change does not settle it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `graph-queries`: adds the answer for a bare name with several definitions that are not one overload set. It also adds how the output names each definition and how `--file` selects one. The capability's purpose line covers overloads only today, so the implementation also widens that line in the main spec.
- `mcp-server`: the table of `codegraph_callers`, `codegraph_callees` and `codegraph_impact` gains a Definition column for a query that spans several definitions.

## Impact

- Code in `internal/query`: the choice of the start nodes of a traversal, the result types and the markdown table.
- Code in `internal/cli`: `callers`, `callees`, `impact` and their text renderers.
- Code in `internal/mcp`: three tool inputs and their descriptions.
- The graph UI calls the same engine functions. Its rows become the union, and its code does not change.
- Output: CLI text, `--json` and MCP results change for a name with several definitions, and in the order of tied rows of an overloaded query. The plain CLI goldens and the wire oracle transcripts that hold such a query change, and each change needs a review.
- The MCP input schemas of three tools gain one optional property. A client that sends no `file` sees no difference in the call.
- No new dependency, no change to the store format, and no change to the index.
