# Index-state preconditions

A tool registers only when a `.codegraph/` index resolves for the server's
working directory. An unindexed repository advertises zero tools, while
`initialize` still succeeds. The remedy is to run `codegraph init`.

## Live re-check

The catalog is re-checked on every request. An index created part-way
through a session appears without a client restart or reconnect.

## Resources are unaffected

Resources are served regardless of index state. This document, and every
per-tool fact-sheet, remain readable in a repository that has never been
indexed.

## Result

This document is markdown text describing server behavior, not a tool
result.
