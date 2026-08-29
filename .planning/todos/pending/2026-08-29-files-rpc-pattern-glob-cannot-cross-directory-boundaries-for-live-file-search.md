---
created: 2026-08-29T00:00:00.000Z
title: Files RPC's Pattern glob cannot cross directory boundaries — live file search only surfaces root-level matches
area: query
severity: minor
files:

  - web/src/lib/search.ts
  - internal/query/files.go
---

## Problem

**Confirmed live, not assumed** — found during 03-06 Task 3's mandated manual
UAT against this repository's own index (a real `codegraph ui` server, a real
browser via `agent-browser`), not caught by any unit test because the unit
test fixtures never exercised real, nested repository paths.

`internal/query.FilesOptions.Pattern`'s own doc comment states it is "a shell
glob (`path/filepath.Match` semantics, matched against the full
forward-slashed file path)". Go's `path`/`filepath.Match` implements a
`fnmatch`-style dialect where `*` matches any sequence of **non-Separator**
characters — it never crosses a `/`, and there is no recursive `**` wildcard
in the dialect at all. Verified empirically this session:

```go
filepath.Match("*detail*", "internal/query/detail.go")  // false
filepath.Match("*claude*", "claudeassets.go")            // true  (root-level)
filepath.Match("*/*/*detail*", "internal/query/detail.go") // true (exact depth)
```

03-06's `web/src/lib/search.ts` builds `Files`' live-search-as-you-type
pattern as `*` + term + `*`, following the only convention the wire's own
doc comment supports. This works correctly for root-level files
(`claudeassets.go`) but returns **zero matches for any file one or more
directories deep** — which, in a real Go monorepo with `internal/`, `cmd/`,
`web/` etc., is the overwhelming majority of files. Confirmed live against
this repo:

```
Files{pattern: "*detail*"} -> {"format":"flat"}   // empty — internal/query/detail.go exists
Files{pattern: "*claude*"} -> one match: claudeassets.go (root-level)
```

**This is not a client bug and was not silently shipped broken.** No single
glob string can express "term appears anywhere in the path, at any depth" —
this is an architectural property of Go's `path/filepath.Match`, shared by
every caller of `FilesOptions.Pattern` (the CLI, the MCP `files` tool, and
now this RPC), not something introduced by 03-06. 03-06 documented the
limitation directly in `search.ts` and left it unfixed as out of scope,
because the fix is server-side and cross-cutting (any change to
`FilesOptions.Pattern`'s matching semantics affects the CLI and MCP tool's
existing documented behavior, not just this one RPC caller).

**Mitigating context, not a full substitute:** `Engine.Search`'s
`matchNodes` iterates every indexed node — including file-kind pseudo-nodes,
whose `Name`/`QualifiedName` is the file's own path — via
`lexicalMatchTier`, which IS a proper prefix/substring-aware match, not a
whole-path glob. Confirmed live: `Search{term:"detail"}` returned
`internal/query/detail.go`, `internal/query/detail_test.go`, etc., each with
`kind: "file"`. So BRW-01's "search files as you type" intent is
functionally served today — just via the Symbols section (mixed in with
real symbol matches), not via the Files section, for any file not at repo
root.

## Solution

Open question for whoever picks this up — no single obviously-correct fix,
several real options:

1. **Give `Engine.Files` a genuine substring-match mode**, e.g. a
   `Files(opts)` boolean or separate `Pattern` semantic meaning "contains,
   not glob" — a real behavior change to `internal/query`, needs its own
   design pass (does it replace glob matching or add alongside it? does the
   CLI's `--pattern` flag gain a `--contains`-style companion, or is this
   UI-only?).
2. **Split file-kind nodes out of `Search`'s result set into their own
   first-class concept** so the Symbols/Files client-side split maps onto a
   real server-side split instead of an incidental one (Search returning
   file pseudo-nodes was presumably intentional for some other consumer —
   investigate before touching it).
3. **Leave it as-is and reframe the UI copy/expectations** — the "Files"
   section becomes explicitly "files matching this glob-style pattern
   [help]" rather than implying free-text substring search, and the Symbols
   section's incidental file-kind matches are documented as the actual
   arbitrary-depth file-discovery path.
4. **Multi-pattern fan-out client-side** (issue several `Files` calls with
   `*/…` prefixes at a few bounded depths, merge) — rejected in 03-06 as
   over-engineering for a debounced live-search box and a violation of that
   plan's own "issues exactly one Search and one Files" behavior spec; worth
   re-evaluating only if options 1–3 are all rejected.

Whichever direction is chosen, re-verify against a live index with genuinely
nested paths (this repo's own `internal/` tree is a good fixture) — the unit
test suite's flat fixture data never would have caught this.
