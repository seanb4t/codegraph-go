# Design

## Context

See proposal.md for the motivation. The facts below are from `main` at `5920f7ce`.

- `symbolTargets` in `internal/query/overloads.go` turns a `Selector` (symbol and signature) into the start nodes of a traversal. It collects each node with the name and sorts them by id. With an overload set and no signature, the start nodes are the members and the set. With no set, it calls `resolveSymbolNode`, which returns the node with the lowest id. The signature branch returns early with `overloaded` true.
- `CallersSelected`, `CalleesSelected` and `ImpactSelected` in `internal/query/traverse.go` are the only callers of `symbolTargets`. The CLI, the MCP tools and the UI handlers in `internal/uiserver/handlers.go` call them, or the wrappers `Callers`, `Callees` and `Impact`.
- `resolveSymbolNode` has one more caller: `resolveNodeForDetail` in `internal/query/node.go`, for `node` with no file. This design does not change that path.
- `enumerateSymbolDefs` in `internal/query/node.go` already returns the nodes of a name that are not `overload_set` nodes. `node` uses it.
- `targets.overloaded` marks a query that spans overloads. Each row then carries `Location.Signature` or `Location.Ambiguous`.
- `callers` removes a repeated row with a key. The key is the caller. If `overloaded` is true, the key also includes the start node. `callees` always uses the start node and the callee as its key.
- `sortLocations` orders rows by file, name and line with a stable sort. Rows that are equal in all three keep the order of the start nodes, which is the order of the node ids.
- The CLI text adds a suffix to an overloaded row (`query.OverloadSuffix`). If one row has a signature or is ambiguous, `renderLocationTable` in `internal/query/render_results.go` adds an Overload column. `RenderCallersMarkdown` and `RenderCalleesMarkdown` return one sentence and no table for a result with no row.
- `node --file` first looks for a node whose `FilePath` equals the value. If the name has several matches, `narrowNodeMatches` keeps the ones whose path contains the value, compared through `normalizeNarrowPath` (lower case, `\` as `/`). The `node` output for several definitions starts with `**N definitions named "X"**` and caps its list at `nodeMultiDefListCap`, which is 20.
- `NodeSelected` takes the file as a parameter beside the `Selector`.
- `callers` and `callees` use `-l` for `--limit`, and `impact` uses `-d` for `--depth`. The three commands each declare the same `--signature` flag. No short flag `-f` is in use on them.
- `impact` keeps one visited set for the whole walk. A start node is a row with its overload label. A node that the walk reaches is a row with no label, and it appears once.
- A node can have an empty `FilePath`: the package node, and a C# partial type whose parts are in several files.
- `task golden:regen` runs `gocapture`, which captures `explore` and `node` output only. The plain CLI goldens come from `go test ./internal/cli -run 'TestPlainGolden$' -update-plain-goldens`. The wire oracle transcripts come from `test/wireoracle/cmd/wireoracle`.

One conflict exists between the spec and the code, and this design does not settle it. The requirement "A signature selects one overload" says that `callers`, `callees` and `impact` keep accepting `--line`. Those commands have no `--line` flag. Only `node` has it.

## Goals / Non-Goals

**Goals:**

- One rule for the start nodes of `callers`, `callees` and `impact`, in one function.
- No dependence on node ids in the rows, in their order, or in their labels.
- Output for a name with one definition that is the same as today byte for byte.
- Output for one overload set alone that has the same rows and labels as today. The order of rows that tie on file, name and line can change once (D4).

**Non-Goals:**

- A change to `node`, `search`, `explore` or `affected`.
- A change to `resolveSymbolNode` or its use by `node`.
- A ranking of definitions, for example "the non-test file first" or "the last Python definition".
- A definition label or a file selector in the graph UI. D8 gives the reason.
- The `--line` conflict above.

## Decisions

### D1: The start nodes are every node with the name

`symbolTargets` gets the definitions from `enumerateSymbolDefs`, so `node` and the three traversals share one meaning of "definition". It adds the `overload_set` nodes of the name as start nodes marked ambiguous, as today. The branch that calls `resolveSymbolNode` goes away.

`symbolTargets` orders the start nodes by file, line, kind and qualified name, and not by id. `enumerateSymbolDefs` has its own order for `node`, which puts generated files last. `symbolTargets` sorts again, because a traversal needs an order that no heuristic changes.

Alternative: keep one pick and add a note that lists the others. Rejected, because the answer stays wrong by default. An agent acts on "0 callers" and does not read a note.

Alternative: return an error that asks for a file. Rejected, because a query for a common name works in one step today, and the union is a correct answer to it.

Alternative: prefer the last definition of a Python, Ruby or JavaScript name. Rejected as the fix, because it does not cover one name in two files or on two types. It is also not needed as an addition: the union already holds the real definition, and a typing stub adds no row.

### D2: "Spans definitions" is a second mode beside "overloaded"

`targets` gains a field `spans`. If the start definitions are not all members of one overload set, it is true. The cases are two or more definitions with no set, a set plus a definition outside it, and two sets.

- One function computes `spans` from the final start nodes. Each return path of `symbolTargets` calls it, the signature branch included. A signature that matches in two files therefore spans definitions.
- `overloaded` keeps its meaning and its effect on `Signature` and `Ambiguous`. A query over one set alone has `overloaded` true and `spans` false, so its output does not change.
- In `callers`, if `overloaded` or `spans` is true, the row key includes the start node. `callees` already includes the start node in its key, so it needs no change.

Alternative: treat every multi-node query as `overloaded` and reuse the signature label. Rejected, because two functions in two files often have the same signature, so the label does not tell them apart.

### D3: The definition label

`Location` gains `Definition string` with the JSON name `definition` and `omitempty`. One function builds the label of a node:

- `<FilePath>:<StartLine>` for a node with a file.
- `<Kind> <QualifiedName>` for a node with no file, for example `package example.com/app/cache`.

A row that comes from the `overload_set` node is marked ambiguous and gets no label. It belongs to the set, and the set is not a definition.

The three result types gain `Definitions []Location` with the JSON name `definitions` and `omitempty`. Each entry carries its own label in `Definition`. The list is in the order of D1, and its length is capped at `MaxLimit`.

If `spans` is false, both fields are empty. `encoding/json` then leaves them out, which gives the byte-for-byte rule of the spec with no second code path.

In `impact`, a start node that is a definition carries its own label. A node that the walk reaches carries none. The walk has one visited set and does not record which start node led to a node. This matches how `impact` treats the overload label today.

Alternative: an object `{filePath, startLine}` in each row. Rejected: the row already has `filePath` and `startLine` of its own. A second pair with the same names in a nested object is easy to misread. A string is also what the text and the table print.

### D4: The order of the rows

`sortLocations` gains two more keys after file, name and line: the definition label, then the signature. Rows that tie today tie only inside an overloaded query, where their order follows the node ids. With the new keys, that order follows the signature. A frozen output of an overloaded query can therefore change its row order once. D7 covers the review of that diff.

Alternative: keep `sortLocations` and sort the start nodes only. Rejected, because the stable sort then still carries an order that the rows do not show, and a reader of the output cannot predict it.

### D5: Text and table rendering

- One function beside `OverloadSuffix` builds the definitions line from a result's `Definitions`. The CLI plain branches, the styled renderers and the markdown renderers all call it. The wording follows `node`: `N definitions named "X"`, then the labels. The line lists at most `nodeMultiDefListCap` labels and then `+K more`.
- One function builds the row suffix of a definition. In the CLI text it follows the overload suffix.
- The definitions line is printed also for a result with no row. In the CLI text it follows the header line. In the markdown it follows the "no callers found" sentence. This is the case of the issue: a name with several definitions and "0 callers".
- `renderLocationTable` builds its header and its cells from two flags, one for the Definition column and one for the Overload column. The Definition column follows Location.
- The frozen outputs pin the exact bytes after the first capture.

### D6: `--file` filters the start nodes

`CallersSelected`, `CalleesSelected` and `ImpactSelected` take the file as a parameter beside the `Selector`, as `NodeSelected` does. The wrappers `Callers`, `Callees` and `Impact` pass an empty file, so the UI handlers need no change.

`symbolTargets` keeps the definitions whose `FilePath` contains the value, compared through `normalizeNarrowPath`. This is the test of `narrowNodeMatches`. The difference is that the filter here is hard: if nothing matches, the query fails. `node` falls back to all matches in that case, because `node` shows each of them. A traversal with that fallback answers for definitions the user excluded.

- A node with no file never matches.
- If no member of a set stays, the set is not a start node.
- The filter runs before the signature filter, and `spans` is computed after both.
- If nothing matches, the error is a not-found error. It names the value and lists each file that holds a definition once, sorted. The form follows the existing error for a signature that matches no overload.

One helper in `internal/cli` registers `--signature` and `--file` (short form `-f`) on the three commands, so the flag text exists once. The MCP input is `file`, placed in the argument structs in the order of the JSON names.

Alternative: a selector by line, as `node --line`. Rejected for this change: `-l` is taken on two commands, and a file is enough to select in each case the issue reports.

Alternative: equality only. Rejected, because `node --file models.py` already finds `src/app/models.py`, and two rules for one flag name on sibling commands is a trap.

### D7: Limits and frozen outputs

`callers` and `callees` sort the rows and then apply `--limit` and `MaxLimit`. A union can pass a limit that one definition did not pass. The definitions line is not subject to the row limit, so the user still sees how many definitions exist and can select with `--file`.

Three kinds of frozen output can change, and each has its own capture command:

- The plain CLI goldens: `go test ./internal/cli -run 'TestPlainGolden$' -update-plain-goldens -count=1`.
- The wire oracle transcripts: the capture command under `test/wireoracle/cmd/wireoracle`. The input schemas of three tools change, so each transcript that holds `tools/list` changes.
- The corpus goldens of `task golden:regen`. They hold `explore` and `node` output, which this change does not touch, so no diff is expected there.

The implementation reads each diff and records it in the pull request. A capture for a name with one definition must not change, except for the row order of an overloaded query (D4) and the schema text.

### D8: The graph UI gets the union and no label

The UI handlers call `Callers`, `Callees` and `Impact`. After this change they return rows for every definition, which is the correct set. The rows carry no definition label in the UI, because `uiv1.Location` has no field for it.

A label in the UI needs a protobuf field, generated code for Go and TypeScript, and a change to the Svelte views. That is a second surface with its own review. A follow-up issue tracks it. Until then a UI row for a name with several definitions shows its own file and line, which is true. It does not show which definition it belongs to.

## Risks / Trade-offs

- [A query for a very common name, such as `get` or `__init__`, returns many rows] → The limit caps the rows. The definitions line shows how many definitions exist, and `--file` selects. Today's answer for such a name is one arbitrary definition, which is worse.
- [A caller of one definition also appears through the dispatch widening of another definition] → The row key includes the start node. The row therefore appears once for each definition, each with its label. The widening itself does not change.
- [`impact` counts change for a name with several definitions] → `nodeCount` and `edgeCount` already count from every start node of an overload set. The same rule now applies to more names. `TestImpact_DispatchTraversal` shows it: `Handle` has two definitions, so its `NodeCount` goes from 2 to 3.
- [An agent parses the CLI text and meets the new definitions line] → The line appears only where today's answer is incomplete. The `--json` shape is the stable surface, and its new fields are additive.
- [A `--file` value matches more files than the user meant] → The definitions line of the result shows what the query started from. A longer value narrows the match.
- [The row order of an overloaded query changes once] → The new order follows the signature and not the node id. The pull request lists each such diff.

## Migration Plan

No data moves. The change ships in a normal release. A rollback is a revert of the pull request, with the frozen outputs captured again.
