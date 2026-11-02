# codegraph_callees

List a symbol's forward call targets.

## Arguments

- `symbol` (string, required) — Symbol name.
- `limit` (integer, optional) — Cap on results returned.
- `path` (string, optional) — Repo path (default: server cwd).
- `signature` (string, optional) — Parameter types that select one overload, for example `"String, Throwable"`. Without it, a name that has several overloads reports across all of them, with the overload each row belongs to.

## Result

Markdown text.
