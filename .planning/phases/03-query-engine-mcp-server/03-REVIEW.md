---
phase: 03-query-engine-mcp-server
reviewed: 2026-07-11T00:00:00Z
depth: deep
files_reviewed: 34
files_reviewed_list:
  - internal/graphstore/store.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/iter_test.go
  - internal/query/engine.go
  - internal/query/resolve.go
  - internal/query/validate.go
  - internal/query/search.go
  - internal/query/traverse.go
  - internal/query/files.go
  - internal/query/status.go
  - internal/query/node.go
  - internal/query/explore.go
  - internal/query/render_markdown.go
  - internal/query/engine_test.go
  - internal/query/search_test.go
  - internal/query/traverse_test.go
  - internal/query/files_status_test.go
  - internal/query/render_markdown_test.go
  - internal/mcp/server.go
  - internal/mcp/tools.go
  - internal/mcp/server_test.go
  - internal/cli/root.go
  - internal/cli/query.go
  - internal/cli/search.go
  - internal/cli/callers.go
  - internal/cli/callees.go
  - internal/cli/impact.go
  - internal/cli/affected.go
  - internal/cli/files.go
  - internal/cli/status.go
  - internal/cli/node.go
  - internal/cli/explore.go
  - internal/cli/serve.go
  - internal/cli/query_cli_test.go
  - testdata/golden/golden_parity_test.go
findings:
  critical: 2
  warning: 5
  info: 1
  total: 8
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-07-11T00:00:00Z
**Depth:** deep
**Files Reviewed:** 34
**Status:** issues_found

## Summary

The D-04 reverse-adjacency prerequisite bug (`IterateEdges("")` scanning only
the never-written empty-src slice instead of the whole `e/` namespace) is
correctly fixed and is now pinned by `TestIterateEdgesEmptyPrefixScansEveryEdge`
— confirmed by direct inspection of `pebble_store.go` and the new
`IterateNodes`/`IterateFiles` whole-namespace scans, which follow the same
correct pattern. The `internal/query`/`internal/mcp` packages correctly avoid
importing `pebble` directly (D-04a boundary holds), and MCP handlers do share
Engine-level validation with the CLI (D-08b) for kind/limit/depth/path
confinement.

However, two provable, previously-untested gaps undercut the documented DoS
and trust-boundary posture this phase claims: (1) `query`/`search`/`callers`/
`callees` apply their `MaxLimit=1000` DoS ceiling only when the caller
supplies an *explicit* over-large `--limit`, never as an actual cap on the
default (unset, `limit<=0`) case — meaning the common, no-flag invocation of
every one of these commands (via CLI or MCP) returns an **unbounded** result
set, directly contradicting the `MaxLimit` doc comment's own stated contract;
and (2) every MCP tool accepts a client-supplied `path` argument that is
resolved with no confinement to the server's configured repo root, letting an
MCP client (including a prompt-injected agent) redirect any tool — including
`codegraph_explore`'s verbatim source reads — to an entirely different
`.codegraph/`-indexed project elsewhere on the host filesystem. Both are
classified Critical below. Five further Warnings and one Info item round out
the findings (nil-vs-empty-array JSON inconsistencies, inconsistent negative-
value handling across sibling flags, no symlink confinement in the
fresh-from-disk path-safety gate, dangling-edge fragility, and an
empty-term "matches everything" footgun).

## Critical Issues

### CR-01: `query`/`search`/`callers`/`callees` return an unbounded result set when `--limit`/`limit` is omitted (the default case)

**File:** `internal/query/validate.go:57-67`, `internal/query/search.go:115-137,142-170`, `internal/query/traverse.go:127-163,168-196`

**Issue:** `MaxLimit`'s doc comment states it "bounds how many result rows
query/search/callers/callees may return in one call" (validate.go:24-26), and
`T-03-02-DoS`/V5 both frame this as the mechanism that prevents an untrusted
caller from forcing an unbounded allocation. In practice `validateLimit`
*only* rejects an explicit `n > MaxLimit`; it never supplies a default ceiling
when `n <= 0` (the documented "caller did not set a limit" case,
validate.go:59-61). Every call site then only applies the limit
conditionally:

```go
// search.go Query/Search
n := len(ranked)
if limit > 0 && limit < n { n = limit }   // limit==0 => no cap at all

// traverse.go Callees/Callers
if limit > 0 && limit < len(locs) { locs = locs[:limit] }  // same gap
```

Every CLI flag defaults `limit` to `0` (`query.go:84`, `search.go:62`,
`callers.go:58`, `callees.go:59`), and every MCP tool defaults it the same
way (`tools.go:166,195,219` — `req.GetInt("limit", 0)`). This means the
*default, most common invocation* of `codegraph query`, `codegraph search`,
`codegraph callers`, `codegraph callees` — and their `codegraph_search`/
`codegraph_callers`/`codegraph_callees` MCP equivalents — returns every
matching node/edge with no cap, for a term that matches broadly (or, per
CR/WR-05 below, an empty term that matches the *entire* graph). On a
large monorepo this is exactly the unbounded-allocation DoS surface `V5`/
`T-03-02-DoS` set out to close, and it is untested: `TestValidateLimit`
(engine_test.go:177-187) only exercises `10`/`MaxLimit`/`MaxLimit+1`, never
the `limit==0` "no explicit limit" path, and every `TestQuery`/`TestSearch`/
`TestCallersCallees` subtest that checks limiting behavior always passes a
non-zero limit.

Compare with `Files`, which *does* apply an unconditional post-filter cap
regardless of any explicit request (`files.go:150-152`,
`if len(entries) > MaxLimit { entries = entries[:MaxLimit] }`) — proving the
intended pattern exists elsewhere in this same package and was simply not
applied to Query/Search/Callers/Callees.

**Fix:** Apply the same unconditional cap Files.go already uses, after
ranking/collection and independent of whether an explicit `--limit` was
given:

```go
// search.go Query/Search, after building `nodes`/`locations`:
if len(nodes) > MaxLimit {
    nodes = nodes[:MaxLimit]
}

// traverse.go Callees/Callers, after building `locs`:
if len(locs) > MaxLimit {
    locs = locs[:MaxLimit]
}
```

### CR-02: MCP tools accept a client-supplied `path` argument with no confinement to the server's configured repo root

**File:** `internal/mcp/tools.go:13-33`, `internal/mcp/server.go:79-92`

**Issue:** `openEngine`/`resolvePath` resolve every tool call's `path`
argument via `req.GetString("path", defaultPath)` with **no validation that
the resolved path is within (or equal to) `defaultPath`** (the server's
configured `repoPath`, itself derived from server-launch cwd in
`serve.go:41-53`). `query.OpenAt(path)` then walks upward from whatever path
the client supplied looking for `.codegraph/` (`resolve.go:31-48`) and, if
found, opens **that** store — including confining `Explore`/`Node`'s
fresh-from-disk verbatim source reads to that *different* project's root
(`engine.go:39-41`, `node.go:19-50`).

Every one of the 8 registered tools (`codegraph_explore` always, plus every
allowlisted companion tool) exposes this same `path` argument
(`tools.go:41,83,91,98,105,112,121,126`). An MCP client — which in this
product's stated threat model is an AI agent that may be processing
attacker-influenced content (prompt injection) — can therefore redirect any
tool call to read and return the *verbatim source code* of an entirely
different, unrelated project on the same host, as long as that project has
ever been `codegraph init`'d (a very common developer-machine condition:
multiple sibling checkouts, each with its own `.codegraph/`). This
undermines the very control `CODEGRAPH_MCP_TOOLS`/D-08a exists to provide
(an operator scoping which tools are exposed for a given repo's MCP server
instance) — the allowlist controls *which* tools are visible, but any
visible tool can still be pointed at *any other indexed project on disk* via
`path`, which is a materially different trust boundary than "query this
repo."

**Fix:** Either drop the per-call `path` override for MCP entirely (always
use the server-configured `repoPath`, matching what `BuildServer` already
threads through), or confine it: resolve the client-supplied path and reject
(return an MCP tool error, not open an engine) if it does not resolve to
`repoPath` or a descendant of it.

```go
func openEngine(req mcp.CallToolRequest, repoPath string) (*query.Engine, func() error, error) {
    path := resolvePath(req, repoPath)
    abs, err := filepath.Abs(path)
    if err != nil {
        return nil, nil, err
    }
    rel, err := filepath.Rel(repoPath, abs)
    if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
        return nil, nil, fmt.Errorf("mcp: path %q is outside the server's configured repo root", path)
    }
    eng, closer, err := query.OpenAt(abs)
    ...
}
```

## Warnings

### WR-01: Zero-match results marshal `null` instead of `[]` for several JSON array fields

**File:** `internal/query/traverse.go:143,183,299-300`, `internal/query/files.go:117`

**Issue:** `CalleesResult.Callees`, `CallersResult.Callers`,
`AffectedResult.AffectedTests`, and `FilesResult.Files` are all declared with
`var x []T` and only ever `append`ed to. When zero matches are found (a
symbol with no callers, `files --filter <nothing-matches>`, etc.), the slice
stays `nil`, and `encoding/json` marshals a nil slice as `null`, not `[]`.
This is inconsistent with `Search`/`Query`, which build their result via
`make([]Location, n)` (search.go:132,159) and therefore always marshal `[]`
even for zero matches. A JSON consumer (an agent script doing
`result.callers.map(...)` or `.length`) that reasonably assumes an array
field is always an array will throw/crash on the `null` case. This is
untested — no test asserts JSON shape for a zero-match `callers`/`callees`/
`files`/`affected` call.

**Fix:** Initialize these slices non-nil, e.g. `locs := []Location{}` instead
of `var locs []Location` in `Callees`/`Callers`, and the equivalent in
`Affected`/`Files`.

### WR-02: Inconsistent negative-value handling across sibling flags (`--depth`, `--max-files`, `--limit`)

**File:** `internal/query/validate.go:44-90`, `internal/query/files.go:73-84`

**Issue:** Three different conventions exist for how a negative numeric
input is treated:
- `--limit` (query/search/callers/callees): negative is silently treated
  identically to unset ("no limit" — validate.go:59-61 doc).
- `--depth` (impact) / `--max-files` (explore): negative is silently
  treated identically to unset ("use the default" — `clampDepth`/
  `clampMaxFiles`, validate.go:47-55,82-90).
- `--depth` (files): negative is explicitly **rejected as an error**
  (`validateFilesDepth`, files.go:76-84, `n < 0` → error).

A caller who learns "negative depth is an error" from `codegraph files
--depth -1` will reasonably but incorrectly assume the same holds for
`codegraph impact --depth -1`, which instead silently falls back to
`defaultDepth`. This is a genuine cross-command consistency gap in an
otherwise carefully-designed validation layer.

**Fix:** Pick one convention (reject negative values outright, matching
`validateFilesDepth`'s stricter V5 posture) and apply it uniformly across
`clampDepth`, `clampMaxFiles`, and `validateLimit`.

### WR-03: `resolveSourcePath`'s repo-root confinement does not resolve symlinks

**File:** `internal/query/node.go:19-50`

**Issue:** `resolveSourcePath` confines `relPath` via `filepath.Clean`/
`filepath.Rel` string manipulation only — it never calls
`filepath.EvalSymlinks` (or equivalent) on the resolved absolute path before
handing it to `os.ReadFile`. If any file or intermediate directory inside
the indexed repo is a symlink pointing outside `repoRoot` (a realistic case
for a repo checked out from an untrusted or compromised source, or one with
a build-tool-generated symlink), `Node`'s file-mode read and `Explore`'s
per-file verbatim-source read will silently follow it and return content
from outside the confined root — defeating the T-03-06-Path defense the
function's own doc comment claims to provide.

**Fix:** After computing `abs`, resolve symlinks and re-verify confinement:

```go
resolved, err := filepath.EvalSymlinks(abs)
if err != nil {
    return "", err
}
if rel, err := filepath.Rel(root, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
    return "", fmt.Errorf("query: path %q escapes the repo root", relPath)
}
```

### WR-04: A single dangling edge/node reference aborts the entire query instead of being skipped

**File:** `internal/query/traverse.go:149-152,185-188,233-236`, `internal/query/explore.go:71-74`, `internal/query/node.go:133-136,149-152`

**Issue:** Every traversal helper (`Callees`, `Callers`'s reverse-adjacency
consumers, `Impact`'s BFS, `Explore`'s blast-radius builder, `Node`'s
calls/called-by trail) calls `e.reader.GetNode(edge.Target)` or
`e.reader.GetNode(edge.Source)` and immediately propagates any error —
including `graphstore.ErrNotFound` — up as a hard failure of the whole
command. Today's single-shot, atomically-committed indexer run makes a
dangling edge unlikely, but this phase's own docs (`DeleteFileSubgraph`,
Phase 4 sync territory) anticipate incremental writes where a node can be
pruned while edges referencing it still exist momentarily. As written, one
stale/missing node reference takes down `callers`/`callees`/`impact`/
`explore`/`node` entirely rather than degrading gracefully (e.g., skipping
the unresolvable edge and continuing).

**Fix:** Treat `graphstore.ErrNotFound` from these lookups as "skip this
edge" rather than a hard error, at minimum in the traversal loops that are
already tolerant of missing data elsewhere (e.g. `buildReverseAdjacency`'s
consumers).

### WR-05: An empty search/query term matches every node in the graph

**File:** `internal/query/search.go:39-58`

**Issue:** `lexicalMatchTier` compares `term` against `name`/`qualifiedName`
via `strings.HasPrefix(field, term)`, which is unconditionally `true` when
`term == ""`. Since the CLI's `query <term>`/`search <term>` args accept an
empty string (`cobra.ExactArgs(1)` only requires one positional arg, not a
non-empty one) and the MCP `codegraph_search` tool's `query` arg is likewise
just `mcp.Required()` (accepts `""`), calling `codegraph search ""` or the
MCP tool with `query: ""` returns **every node in the store** ranked at
`lexicalTierPrefix`, subject only to CR-01's now-documented lack of a
default cap. This is a surprising "dump the whole graph" side effect of an
edge case that looks like it should mean "no matches" or be rejected
outright.

**Fix:** Reject an empty `term` in `Query`/`Search` (and `Explore`'s `query`)
with a clear error, mirroring `ValidateKind`'s "reject before any scan"
posture, rather than letting `HasPrefix`'s degenerate case silently match
everything.

## Info

### IN-01: `DeleteFileSubgraph`'s name does not match its documented behavior

**File:** `internal/graphstore/store.go:148-151`

**Issue:** The `Writer.DeleteFileSubgraph(path string) error` method name
strongly implies it deletes a file's entire subgraph (its nodes and edges),
but its own doc comment says it "stages a single range-delete over path's
own file record" — i.e., only the `schema.File` record itself, not the
nodes/edges defined in that file. If this is accurate, the name is
misleading and will likely mislead Phase 4's rename/delete-pruning
implementation into assuming node/edge cleanup is already handled here when
it is not (compounding WR-04's dangling-reference fragility). If the doc
comment is simply stale/imprecise and the implementation (not in this
phase's file list) does delete the full subgraph, then the comment itself
needs correcting instead.

**Fix:** Reconcile the method name and its doc comment with the actual
(current or Phase-4-planned) implementation — either rename to something
like `DeleteFile` if it truly only removes the file record, or update the
comment to confirm it also range-deletes the file's node/edge namespace.

---

_Reviewed: 2026-07-11T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
