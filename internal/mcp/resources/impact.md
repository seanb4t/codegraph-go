# codegraph_impact

Depth-bounded reverse blast radius of a symbol.

## Arguments

- `symbol` (string, required) — Symbol name.
- `file` (string, optional) — File path. Keeps the definitions of the symbol in the files whose path contains it. Without it, a name with several definitions reports across all of them: the result lists the definitions, and a Definition column names the definition of each row.
- `depth` (integer, optional) — BFS depth (default 2, max 50).
- `path` (string, optional) — Repo path (default: server cwd).
- `signature` (string, optional) — Parameter types that select one overload, for example `"String, Throwable"`. Without it, a name that has several overloads reports across all of them, with the overload each row belongs to.

## Result

Markdown text.
