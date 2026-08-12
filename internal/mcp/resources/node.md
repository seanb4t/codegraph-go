# codegraph_node

Show a symbol's signature, calls, and callers, or a line-numbered file read.

## Arguments

- `file` (string, optional) — File path — disambiguates symbol, or selects file-mode when symbol is omitted.
- `line` (integer, optional) — Line number — narrows an overloaded symbol to the definition containing (or nearest) this line.
- `path` (string, optional) — Repo path (default: server cwd).
- `symbol` (string, optional) — Symbol name to look up (omit for a file-mode read).

## Result

Markdown text.
