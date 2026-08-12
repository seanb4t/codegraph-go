# codegraph_search

Lexically search symbol names/qualified names, returning locations only.

## Arguments

- `query` (string, required) — Search term.
- `kind` (string, optional) — Restrict to one node kind.
- `limit` (integer, optional) — Cap on results returned.
- `path` (string, optional) — Repo path (default: server cwd).

## Result

Markdown text.
