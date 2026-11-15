# Node-only line flag

Issue: #323

## Why

The `graph-queries` requirement "A signature selects one overload" says that `node`, `callers`, `callees` and `impact` keep accepting `--line` to select one overload. Only `node` has `--line`. `callers` and `callees` use `-l` for `--limit`, and `impact` has no `--line`, so the spec describes a flag that three commands never had.

## What Changes

- The requirement "A signature selects one overload" stops saying that the four commands keep accepting `--line`.
- A new requirement, "A line narrows the definitions that node shows (NODE-03)", gives `--line` to `node` only. It says what the code does today: `node` keeps the definitions whose lines contain the line, or else the one whose first line is nearest, and `0` means no line. A line narrows any definitions of a name, overloads or not, so the rule gets a requirement of its own.
- `callers`, `callees` and `impact` keep `--signature`, which selects one overload, also between overloads in one file. `--file`, from the requirement "A file selects the definitions in it", selects the definitions in matching files. No flag is added or removed.
- New tests pin the rule through the CLI on the existing `Logger.java` fixture, and at the first and last line of a definition.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `graph-queries`: the requirement "A signature selects one overload" drops `--line`, and a new requirement specifies `node --line`.

## Impact

- Code: tests only. The CLI and the MCP tools already behave as the corrected requirements say. `codegraph_node` takes `line`, and `codegraph_callers`, `codegraph_callees` and `codegraph_impact` do not.
- Output: none.

## Out of scope

- A `--line` flag on `callers`, `callees` or `impact`. Issue #323 lists it as the other option. `--signature` already selects an overload on these commands, and `--file` selects a definition in another file.
- The short form `-l`, which stays `--limit` on `callers` and `callees` and `--line` on `node`.
- Which definition `node --line` keeps when two first lines are equally near, and a line number above the range of a 32-bit integer. Both are current behavior that this change does not specify.
