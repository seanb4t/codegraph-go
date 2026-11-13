# Design

## Context

See proposal.md for the motivation. The facts below are from `main` at `5920f7ce`.

- `symbolTargets` in `internal/query/overloads.go` turns a `Selector` (symbol and signature) into the start nodes of a traversal. It collects each node with the name. With an overload set and no signature, the start nodes are the members and the set. With no set, it calls `resolveSymbolNode`, which returns the node with the lowest id.
- `CallersSelected`, `CalleesSelected` and `ImpactSelected` in `internal/query/traverse.go` are the only callers of `symbolTargets`.
- `resolveSymbolNode` has one more caller: `resolveNodeForDetail` in `internal/query/node.go`, for `node` with no file. This design does not change that path.
- `targets.overloaded` marks a query that spans overloads. Each row then carries `Location.Signature` or `Location.Ambiguous`, and `callers` lists a caller once for each overload it calls.
- The CLI text adds a suffix to such a row (`query.OverloadSuffix`). If one row has a signature or is ambiguous, the MCP table adds an Overload column (`renderLocationTable` in `internal/query/render_results.go`).
- `node` has `-f/--file` and compares it with `Node.FilePath` for equality. `callers` and `callees` use `-l` for `--limit`, and `impact` uses `-d` for `--depth`. No short flag `-f` is in use on the three commands.
- `CallersArgs`, `CalleesArgs` and `ImpactArgs` in `internal/mcp/tools.go` declare their fields in the order of the JSON names.
- `impact` keeps one visited set for the whole walk. A start node is a row with its overload label. A node that the walk reaches is a row with no label, and it appears once.

One conflict exists between the spec and the code, and this design does not settle it. The requirement "A signature selects one overload" says that `callers`, `callees` and `impact` keep accepting `--line`. Those commands have no `--line` flag, and `-l` is `--limit` on two of them. Only `node` has `--line`.

## Goals / Non-Goals

**Goals:**

- One rule for the start nodes of `callers`, `callees` and `impact`, in one function.
- No dependence on node ids in the answer.
- Output for a name with one definition, and for one overload set alone, that is the same as today byte for byte.

**Non-Goals:**

- A change to `node`, `search`, `explore` or `affected`.
- A change to `resolveSymbolNode` or its use by `node`.
- A ranking of definitions, for example "the non-test file first" or "the last Python definition".
- A normal form for the `--file` value. The match is the equality that `node --file` uses.
- The `--line` conflict above.

## Decisions

### D1: The start nodes are every node with the name

`symbolTargets` returns every node whose name equals the symbol. A node that is a member of a set, the set node, and a node outside any set all go in. The branch that calls `resolveSymbolNode` goes away.

A definition, as the spec uses the word, is such a node that is not an `overload_set` node. The set node stays a start node marked ambiguous, as today.

Alternative: keep one pick and add a note that lists the others. Rejected, because the answer stays wrong by default. An agent acts on "0 callers" and does not read a note.

Alternative: return an error that asks for a file. Rejected, because a query for a common name works in one step today, and the union is a correct answer to it.

Alternative: prefer the last definition of a Python, Ruby or JavaScript name. Rejected as the fix, because it does not cover one name in two files or on two types. It is also not needed as an addition: the union already holds the real definition, and a typing stub adds no row.

### D2: "Spans definitions" is a second mode beside "overloaded"

`targets` gains a field `spans`. If the definitions are not all members of one overload set, it is true. The cases are two or more definitions with no set, a set plus a definition outside it, and two sets.

- `overloaded` keeps its meaning and its effect on `Signature` and `Ambiguous`.
- `spans` adds the definition label. A query over one set alone has `overloaded` true and `spans` false, so its output does not change.
- In `callers` and `callees`, a key removes a repeated row. If `overloaded` or `spans` is true, the key includes the start node. That is the existing rule for overloads.

Alternative: treat every multi-node query as `overloaded` and reuse the signature label. Rejected, because two functions in two files often have the same signature, so the label does not tell them apart.

### D3: The definition label is "file:line"

`Location` gains `Definition string` with the JSON name `definition` and `omitempty`. Its value is `<FilePath>:<StartLine>` of the definition. The three result types gain `Definitions []Location` with the JSON name `definitions` and `omitempty`. The list holds the definitions in the order of `sortLocations`.

If `spans` is false, both fields are empty. `encoding/json` then leaves them out, which gives the byte-for-byte rule of the spec with no second code path.

In `impact`, a start node that is a definition carries its own label. A node that the walk reaches carries none. The walk has one visited set and does not record which start node led to a node. This matches how `impact` treats the overload label today.

Alternative: an object `{filePath, startLine}` in each row. Rejected: the row already has `filePath` and `startLine` of its own. A second pair with the same names in a nested object is easy to misread. A string is also what the text and the table print.

### D4: Text and table rendering

- CLI text: when `spans` is true, one line follows the header line and lists the definitions, for example `2 definitions: a.py:1, b.py:1`. Each row gets a suffix with its definition. If the row also has an overload suffix, the definition suffix follows it. One function builds the suffix for `callers`, `callees` and `impact`, beside `OverloadSuffix`.
- MCP table: when one row has a definition label, the table gains a Definition column after Location, and after it the Overload column when that one applies. A line before the table lists the definitions.
- The exact words of the two new lines are an implementation detail. The frozen outputs pin them after the first capture.

### D5: `--file` filters the start nodes

`Selector` gains `File`. `symbolTargets` keeps the nodes whose `FilePath` equals it, before the signature filter. If the members of a set span files, the set node has no file. With `File`, a set stays a start node only if one of its members stays.

When no node stays, the error is a not-found error. It names the file and lists the files that hold a definition, sorted. The form follows the existing error for a signature that matches no overload.

`spans` is computed after the filter. A file that holds one definition therefore gives the plain output.

The CLI flag is `--file` with the short form `-f` on all three commands, as on `node`. The MCP input is `file`, placed in the argument structs in the order of the JSON names.

Alternative: a selector by line, as `node --line`. Rejected for this change: `-l` is taken on two commands, and a file is enough to select in each case the issue reports.

### D6: Limits do not change

`callers` and `callees` sort the rows and then apply `--limit` and `MaxLimit`. A union can pass a limit that one definition did not pass. The list of definitions is not subject to the row limit, so the user still sees each definition and can select one with `--file`.

### D7: Frozen outputs

The golden corpus captures and the wire oracle transcripts that query a name with several definitions change. The implementation runs `task golden:regen`, reads each diff, and records in the pull request which names changed and why. A capture for a name with one definition must not change. If one does, that is a defect in the implementation and not a capture to accept.

The MCP input schemas of three tools change, so the transcripts and the resource documents that hold a schema or a tool reference change too.

## Risks / Trade-offs

- [A query for a very common name, such as `get` or `__init__`, returns many rows] → The limit caps the rows. The definitions line shows how many definitions exist, and `--file` selects one. Today's answer for such a name is one arbitrary definition, which is worse.
- [A caller of one definition also appears through the dispatch widening of another definition] → The row key includes the start node. The row therefore appears once for each definition, each with its label. The widening itself does not change.
- [`impact` counts change for a name with several definitions] → `nodeCount` and `edgeCount` already count from every start node of an overload set. The same rule now applies to more names. The pull request states the changed captures.
- [An agent parses the CLI text and meets the new definitions line] → The line appears only where today's answer is incomplete. The `--json` shape is the stable surface, and its new fields are additive.
- [A leading `./` or a backslash stops the match of a `--file` value] → The error lists the files that hold a definition, in the form that matches.

## Migration Plan

No data moves. The change ships in a normal release. A rollback is a revert of the pull request, with the frozen outputs captured again.
