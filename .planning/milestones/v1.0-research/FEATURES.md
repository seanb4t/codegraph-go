# Feature Research — v1.0 Drop-in CLI Parity + Human TUI

**Domain:** CLI/MCP behavioral parity — TypeScript CodeGraph v1.3.1 → codegraph-go
**Researched:** 2026-07-14
**Confidence:** HIGH — every claim below is grounded in the installed TS v1.3.1 dist JS (`/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/`) or live `codegraph <cmd> --help` output against the same binary (confirmed `codegraph --version` → `1.3.1`), cross-referenced against our own `internal/query/` and `internal/cli/` source.

> **Supersedes-in-scope note:** this file previously held v0.1's domain-ecosystem research (docs/README-sourced, general "what does a code-graph tool look like" survey). That content shipped in v0.1 and is now historical; this rewrite is v1.0-scoped reverse-engineering of the TS v1.3.1 *reference implementation itself* — the milestone's actual research question (`explore`/`node`/`status` algorithms, exact flag inventory, worktree/git-hooks, watcher defaults). If v0.1's broader domain framing (competitive landscape, MCP ecosystem context) is still needed for roadmap context, it remains in git history at the pre-2026-07-14 commit.

**Method note:** TS's `explore`/`node` CLI commands are NOT separate implementations — `bin/codegraph.js` imports `mcp/tools.js`'s `ToolHandler` and calls `handler.execute('codegraph_explore'|'codegraph_node', args)`, the exact same dispatch path the MCP server uses (confirmed at `bin/codegraph.js:1046-1074`, comment: "The CLI face of the MCP codegraph_explore tool — same handler, same output"). So "MCP tool behavior" and "CLI command behavior" are one algorithm, one spec, for these two commands — including the worktree-notice/staleness-notice cross-cutting wrappers (§5).

---

## 1. `explore` — semantic-relevance selection algorithm

**Classification: TABLE STAKES.** This is codegraph's headline differentiator over grep/Read — TS's own code comments call graph-connectivity relevance "codegraph's home turf." Our current lexical name-match is a materially worse product on this axis (it's the exact case the milestone context flagged: leaks `Test*` funcs, misses call-connected non-lexical matches). Must match algorithmically, not just in output shape.

**Source:** `context/index.js` (`ContextBuilder.findRelevantContext`, `extractSymbolsFromQuery`), `mcp/tools.js` (`ToolHandler.handleExplore`, `computeGraphRelevance`, `buildBlastRadiusSection`).

### 1a. Multi-word query tokenization (`extractSymbolsFromQuery`, `context/index.js:64-145`)

Runs 6 regex passes over the raw query string and unions results into a `Set<string>`:
1. CamelCase (`\b([A-Z][a-z]+(?:[A-Z][a-z]*)*|[a-z]+(?:[A-Z][a-z]*)+)\b`), 2+ chars
2. snake_case (`[a-z][a-z0-9]*(?:_[a-z0-9]+)+`), 3+ chars
3. SCREAMING_SNAKE_CASE
4. ALL-CAPS acronyms, 2+ chars (`REST`, `HTTP`, `LRU`)
5. dot.notation (`app.isPackaged` → adds `"app.isPackaged"`, `"app"`, `"isPackaged"` separately)
6. plain lowercase identifiers, 3+ chars (catches `undo`, `render`, `parse`)

Then filters the union against a **hardcoded ~90-word English stopword list** (`the/and/for/with/…/layer/handle/data/flow/level/request/response/implement/interface/class/method/trigger/affected/…`) — this list exists specifically to keep common English words in a prose query from being treated as symbol names. **This stopword list must be ported verbatim** (see file for full contents) — it's load-bearing for query quality, not incidental.

This is the CLI's `<query...>` variadic arity: `queryParts.join(' ')` (bin/codegraph.js:1062) joins all positional args into one space-separated string before tokenization — so `codegraph explore how does auth work` and `codegraph explore "how does auth work"` are identical. **Our CLI must switch `explore` from `cobra.ExactArgs(1)` to variadic + `strings.Join(args, " ")`.**

### 1b. Hybrid search — 5 ranking channels merged (`findRelevantContext`, `context/index.js:433-870`)

1. **Exact-name matches** on extracted symbols (`findNodesByExactName`), with a **co-location boost**: +20×(N-1) score when N≥2 distinct query symbols land in the same file (e.g. query "scrapeLoop run" both matching in `scrape.go`).
2. **Definition-prefix matches**: title-cases each extracted symbol (`rest`→`Rest`) and searches class/interface/struct/trait/protocol/enum/type_alias names starting with it — catches "the user means `RestController`, not a node literally named `rest`." Includes stem-variant expansion (`caching`→`cache`) and a brevity bonus (shorter names score higher — core classes tend to have concise names vs. verbose test/helper classes).
3. **FTS text search** over `extractSearchTerms(query)`, with **test-file deprioritization** (score ×0.3 unless the query itself contains "test"/"spec") and a **core-directory boost**: if one file holds ≥3× the in-file edges of the runner-up ("dominant file" — e.g. `sinatra/base.rb`), sibling files in its directory get +25.
4. **Multi-term co-occurrence re-ranking**: groups query terms that are substrings of each other (stem variants counted once), then for each result counts how many distinct term-groups match its name/directory. ≥2 groups matched → score ×(1 + matchCount×0.5). Exact-name matches on a **distinctive identifier** (camelCase/snake_case/acronym the user typed) are exempt from dampening; exact matches on a **common dictionary word** (query "flat object" hitting constant `FLAT`) are demoted ×0.3; everything else with only 1 term hit is dampened ×0.6.
5. **CamelCase-boundary + compound-term LIKE matching**: finds substring matches at CamelCase/acronym boundaries (FTS tokenizes `TransportSearchAction` as one token and can't find "Search" inside it; a `LIKE '%Search%'` scan can) and scores by path relevance + brevity, then scales aggressively by term-count for multi-term matches.

All channels merge into one `searchResults` list keyed by node id (max score wins on collision), sorted, truncated to `searchLimit×3`, then filtered by `minScore` (0.2 for explore specifically — `handleExplore` passes `minScore: 0.2`, `maxNodes: 200`, vs. the generic default of 0.3/20).

A **LOW-confidence signal** is set when a multi-term prose query (≥2 terms, length≥3) resolves only to isolated common-word matches with no result corroborated by 2+ terms or a distinctive identifier — surfaced as a "best-effort, not a located answer" footer.

### 1c. Graph expansion beyond lexical matches — THE core "semantic relevance" mechanism

This is what makes TS surface graph-neighbors that don't lexically match the query — the exact behavior the milestone context calls out as missing from our lexical matcher:

1. **Type hierarchy expansion** (`context/index.js:921-971`): for class/interface/struct/trait/protocol entry points, pulls in `extends`/`implements` neighbors via `getTypeHierarchy`, capped at `maxNodes/4`. Runs a **second pass** on newly-discovered parent types to find siblings (`InternalEngine → Engine → ReadOnlyEngine`).
2. **BFS traversal** from every entry point (`traverseBFS`, depth=`opts.traversalDepth` — explore's own call passes depth 3), direction `'both'` (both callers and callees), budget `maxNodes / numEntryPoints` per entry point.
3. **"Glue node" injection** (`mcp/tools.js:2439-2467`): pulls direct callers+callees of every root node that live in a file the subgraph *already* includes (capped at 60), independent of lexical match — this is what bridges "App.tsx's `triggerRender` calls the named `triggerUpdate`" without `triggerRender` itself matching any query term.
4. **Named-symbol seeding with overload disambiguation** (`mcp/tools.js:2477-2554`): every token in the raw query that looks like an identifier (regex-validated, ≤16 tokens) is resolved to ALL its exact-name definitions via the direct node-name index (not FTS — FTS's rank cutoff would drop a 50-overload name's wanted definition entirely, mirroring `findSymbolMatches`'s reasoning in §2). A name with ≤3 total definitions injects all of them; an overloaded name (10+ defs) injects only the defs whose containing file/class is ALSO named in the query (PascalCase tokens act as disambiguators, excluding the project's own name), else falls back to the single most-substantive (most callers / most body lines) definition.
5. **Random-Walk-with-Restart / personalized PageRank** (`computeGraphRelevance`, `mcp/tools.js:2321-2386`) — **the ranking signal that makes structural relevance beat lexical relevance**:
   - Build an **undirected** adjacency graph over the already-gathered subgraph, edges restricted to `{calls, references, extends, implements, overrides, instantiates, returns, type_of, imports}`.
   - Restart vector: uniform over query-matched seed node ids present in the subgraph (falls back to uniform-over-all if no seed landed).
   - Restart probability α = **0.25**, **25 power-iteration steps**, dangling nodes (no edges) keep their mass.
   - Result: a per-node walk-mass score. A file whose symbols are call-connected to the matched cluster accrues mass and ranks high; a lone text match that calls nothing in the flow gets only its own restart share (~0) and gets gated out below.
6. **File-level relevance gate** (`mcp/tools.js:~2775`): aggregate each file's nodes' RWR mass; keep a file only if its mass is ≥6% of the max-mass file, OR it's one of the top-2 "central files" (highest mass AND has a textual term hit — prevents an unrelated hub utility from being mistaken for the subject), OR it defines an agent-named symbol (`entryFiles`, from named seeding), OR it's a "buried" type-definition file force-rescued (near-zero mass AND <2 term hits but the symbol was explicitly named — the `grpc DialOption → dialoptions.go` case), OR it independently matches ≥2 distinct query terms. **A lone one-word text match with zero graph connectivity is dropped even though it "matched."** This gate is the concrete mechanism to reimplement — the milestone context's "leaks Test* funcs" symptom is exactly a missing version of this gate (a lexically-matching `TestFoo` with zero graph mass and no query-term corroboration should not survive it).

### 1d. "⚠️ no covering tests" warning (`buildBlastRadiusSection`, `mcp/tools.js:2254-2304`)

Always-on section, computed AFTER selection, per **root** symbol only (not every rendered symbol):
1. Take `subgraph.roots` (the entry points the query actually matched — not glue/hierarchy/named-seed additions), filter to "meaningful" kinds (function/method/class/interface/struct/trait/protocol/enum/type_alias/component/constant/variable/property/field), cap at **5**.
2. For each root, get its **direct callers only** (`cg.getCallers(root.id)`, not transitive/BFS) and dedupe by node id.
3. **If a root has zero callers at all, it is skipped entirely** — no warning, nothing to say (this matters: "no covering tests" only fires when there IS a blast radius but none of it is tests, not for leaf/unused symbols).
4. Split the caller files into test vs. non-test via the shared `isTestFile(path)` heuristic (`search/query-utils.js`).
5. If `testFiles.length === 0` → append `; ⚠️ no covering tests found`. Otherwise list up to 4 test files + `+N more`.
6. Non-test caller files are listed too (up to 4 + `+N more`), independent of the test check.

**Exact reimplementation note:** our existing `internal/query/explore.go:buildBlastEntry` already computes `TestFiles` via the same direct-caller / dedupe / `isTestSymbol` shape (see code excerpt — it's structurally very close already). The gap is (a) it currently doesn't distinguish "zero callers → skip" from "callers but zero test files → warn," and (b) it's applied to every rendered symbol, not just the top-5 query-matched roots. Confirm/fix both against this spec.

### 1e. Rendering (file-grouping, whole-file-vs-skeleton, per-file budget) — lower priority for the algorithm spec, higher for output-shape

TS clusters nearby symbols per file, renders small files whole, skeletonizes "god files" (signatures only, full bodies for on-flow/query-named symbols), and adapts total output budget to project size (`getExploreOutputBudget(fileCount)`). This governs *how much* renders, not *what's* selected — lower priority than §1a-1d for parity since the milestone context's complaint is about selection, not formatting. Flag as **differentiator-adjacent, defer detailed spec to its own phase** if budget-adaptive rendering proves necessary; a fixed-cap version (what we have) is an acceptable interim divergence as long as selection is fixed first.

---

## 2. `node` — multi-definition disambiguation

**Classification: TABLE STAKES.** Directly addresses a named parity gap ("6 definitions named Run — returning 5 in full").

**Source:** `mcp/tools.js:3572-3677` (`handleNode`), `:4193-4220` (`findSymbolMatches`).

### 2a. Candidate selection (`findSymbolMatches`)

- **Unqualified name** (no `.`/`/`/`::` in the query): enumerate **every exact-name definition** via the direct name index (`cg.getNodesByName`, NOT FTS — comment explicitly says FTS's rank cutoff would drop the wanted overload for a 50+-def name like `poll`). Sort: **generated files last** (`isGeneratedFile` boolean sort, stable otherwise — i.e. no other explicit ordering; ties keep index/insertion order). If zero exact matches, fall back to the single top fuzzy search result.
- **Qualified name** (`Session.request`, `stage_apply::run`): FTS search up to limit 50, with a fallback re-search on the bare tail segment if FTS returns nothing (FTS strips `::`).

### 2b. Optional file/line disambiguation (CLI's `-f/--file`; MCP's `file`+`line` args)

Only applied when `matches.length > 1` AND (`file` hint given OR `line` hint given):
1. **File hint**: keep matches whose `filePath` ends-with OR contains (case-insensitive, `\`→`/` normalized) the hint. If nothing matches, hint is ignored (never empties the set).
2. **Line hint** (only if still >1 candidate): prefer definitions whose `[startLine,endLine]` contains the line; else pick the single nearest-`startLine` match.
3. **Narrowing never empties the result** — a hint matching nothing is silently ignored, never an error.

### 2c. Rendering

- **1 match** → render that definition directly (current behavior, unchanged).
- **>1 match, `includeCode`/full-body not requested** → header `**N definitions named "X"**` + a bare list (`name (kind) — file:line`) + a hint to re-query with `includeCode: true`. (CLI always effectively requests full bodies — see below.)
- **>1 match, full bodies requested** (CLI always does, since the CLI text renderer has no separate "listing-only" mode — confirm against `mcp/tools.js:handleNode` args wiring for the CLI path) → pack as many **full bodies** as fit a **12,000-char budget**, hard-capped at **16 rendered definitions**, in `findSymbolMatches`'s sort order (generated-last, otherwise index order — NOT sorted by relevance/callers/anything else). Header: `Returning {rendered} in full{; {overflow} more listed below}`. Overflow beyond the char budget or the 16-cap is listed by `name (kind) — file:line`, capped display at 20 + `… +N more`, with a hint to re-call `node` with `file`/`line` to pin one.
- Separator between rendered bodies: `\n\n---\n\n`.

**Our current `resolveNodeForDetail`** always resolves to exactly ONE node (lowest-Id tie-break) — needs to become "collect all `getNodesByName` matches, apply file/line narrowing if given, render 1 directly or N-in-full+overflow-list per above." The Go analog of "generated-files-last" sort should reuse whatever generated-file detection exists in `internal/indexer` (check for parity with `generated-detection.js`'s heuristics — vendor dirs, `.pb.go`, etc. — before assuming it's a no-op).

---

## 3. `status` — content and computation

**Classification: TABLE STAKES** for the data (DB size, nodes-by-kind, files-by-language, journal/backend health, pending-changes, worktree mismatch, reindex-recommended) — **differentiator-adjacent** for the human-formatted/colorized rendering (that's TUI work, tracked separately). Good news: our `query.StatusResult` **already computes** `NodesByKind` and `Languages` internally (`internal/query/status.go:188-207`) — the gap is almost entirely in the CLI text renderer (`internal/cli/status.go:47-48`, currently one terse `key=value` line), not the data layer.

**Source:** `bin/codegraph.js:800-969` (non-JSON status action), `index.js:862-863` (`getStats`).

### Exact human-format content (line-for-line, in order)

```
CodeGraph Status

Project: <projectPath>
[⚠ worktree mismatch warning, if any]
[⚠ "last index run never finished" / "silently dropped files" / "run failed", if indexState indicates]
[⚠ "<N> references from an interrupted run are awaiting resolution…", if pendingRefs > 0]

Index Statistics:
  Files:     <fileCount, thousands-separated>
  Nodes:     <nodeCount, thousands-separated>
  Edges:     <edgeCount, thousands-separated>
  DB Size:   <dbSizeBytes / 1024 / 1024, 2 decimals> MB
  Backend:   node:sqlite — built-in (full WAL)     [Go analog: "pebble"]
  Journal:   wal   [or a yellow warning "<mode> — WAL inactive; reads can block on writes" if not wal]

Nodes by Kind:
  <kind, left-padded to 15>  <count>     [sorted by count desc, zero-count kinds omitted]

Files by Language:
  <lang, left-padded to 15>  <count>     [sorted by count desc, zero-count langs omitted]

[if pending changes: "Pending Changes:" block — Added:/Modified:/Removed: <N> files, then "Run codegraph sync to update the index"]
[else: "✓ Index is up to date"]

[if reindexRecommended: "⚠ Index was built by v<X>; re-index to pick up this engine's improvements." + "Run codegraph index (full rebuild) or codegraph sync"]
```

### `--json`/`-j` shape

`{initialized, version, projectPath, indexPath, lastIndexed, fileCount, nodeCount, edgeCount, dbSizeBytes, backend, journalMode, nodesByKind, languages, pendingChanges:{added,modified,removed}, worktreeMismatch:{worktreeRoot,indexRoot}|null, index:{builtWithVersion, builtWithExtractionVersion, currentExtractionVersion, reindexRecommended, state, pendingRefs}}`.

**Note our documented, deliberate omission of `dbSizeBytes`:** `internal/query/status.go`'s doc comment states it's dropped "per testdata/golden/README.md's stripping rules — volatile fields never rendered" (a Phase-3 golden-corpus-diff design decision). For v1.0's richer status content, this needs revisiting — `dbSizeBytes` is exactly the "DB size" the milestone context calls out as missing. Computing Pebble's on-disk size (sum of SST/WAL file sizes under `.codegraph/`) is a straightforward addition; the golden-corpus stripping rule should be updated to strip it only in byte-diff tests, not omit it from the live struct.

**`journalMode`/backend framing**: TS's is meaningful because it runs on SQLite with a real WAL/non-WAL distinction (`journalMode !== 'wal'` is a real degraded-mode warning on network mounts/WSL2). Pebble has no equivalent user-facing journal-mode concept — **acceptable divergence**: drop the `Journal:` line (or replace with a Pebble-appropriate health signal, e.g. compaction backlog, if one exists) rather than fabricate a "wal" string with no underlying meaning. `Backend: pebble` alone is fine.

**Worktree mismatch belongs here** — TS's non-JSON status embeds the FULL warning (`worktreeMismatchWarning`, §5) inline, not the compact notice variant. Confirms status is exempted from the compact-notice wrapper (§5) specifically because it has its own richer treatment.

---

## 4. Per-command flag inventory — full diff (TS v1.3.1 live `--help` vs. our `internal/cli/*.go`)

All TS output below is verbatim from `codegraph <cmd> --help` against the installed v1.3.1 binary (`/opt/homebrew/bin/codegraph`).

| Command | TS flags (verbatim) | Our flags | Diff / Classification |
|---|---|---|---|
| `init [path]` | `-i/--index` (deprecated no-op), `-f/--force`, `-v/--verbose` | `-q/--quiet`, `-v/--verbose`, `--workers` | Missing `-f/--force` (home-dir/root guard bypass) and the deprecated-but-accepted `-i/--index`. We have an extra `--quiet`/`--workers` TS's `init` lacks (TS's `index`/`sync` have `-q`; `init` doesn't). **Table stakes:** add `-f/--force`; **acceptable divergence:** keep our extra `--quiet`/`--workers` on `init` (strict superset, harmless), skip re-adding a no-op `-i`. |
| `uninit [path]` | `-f/--force` | `-f/--force` | **Match.** |
| `index [path]` | `-f/--force`, `-q/--quiet`, `-v/--verbose` | `-f/--force`, `-q/--quiet`, `-v/--verbose`, `--workers` | **Match** (our `--workers` is a superset addition). |
| `sync [path]` | `-q/--quiet` | `-q/--quiet`, `-v/--verbose`, `--workers` | **Match** (superset addition). |
| `status [path]` | `-j/--json` | `--json` (no `-j` short) | **Table stakes:** add `-j` short alias. Content gap is the bigger item — see §3. |
| `query <search>` | `-p/--path`, `-l/--limit` (default 10), `-k/--kind`, `-j/--json` | `-p/--path`, `--limit` (default 0/unbounded?), `--kind`, `--json` | **Table stakes:** add `-l`/`-k`/`-j` short aliases; confirm default limit is 10 not unbounded (verify — TS defaults to 10, capping unset `--limit` differently is a behavior change agents will notice). |
| `explore <query...>` | `-p/--path`, `--max-files` | `-p/--path`, `--max-files`; **`ExactArgs(1)`, not variadic** | **Table stakes — the named divergence:** switch to variadic args + `strings.Join(args, " ")` (§1a). |
| `node [name]` | `-p/--path`, `-f/--file`, `--offset`, `--limit`, `--symbols-only` | `-p/--path`, `-f/--file` only | **Table stakes:** add `--offset`/`--limit`/`--symbols-only` for file-read mode (these mirror Read tool semantics — currently our file-mode `Node()` has no windowing at all, always dumps the whole file). Also the disambiguation behavior gap (§2). |
| `files` | `-p/--path`, `--filter <dir>` (dir-scoped filter), `--pattern <glob>`, `--format <tree\|flat\|grouped>` (default tree), `--max-depth <number>`, `--no-metadata`, `-j/--json` | `-p/--path`, `--pattern`, `--filter` (**language filter, not directory filter — semantic collision on the same flag name**), `--depth` (name differs from `--max-depth`), `--json` (no `-j`) | **Real semantic divergence, not just naming:** our `--filter` restricts by *language*; TS's `--filter <dir>` restricts by *directory*. Fix by (a) renaming ours to something else (e.g. `--lang`) and adding a TS-compatible `--filter <dir>`, or (b) reusing `--filter` for directory-scoping and finding another name for language-restriction. Also missing: `grouped` format, `--no-metadata`, `-j` short. Rename `--depth`→`--max-depth` (or alias both). |
| `daemon` \| `daemons` | *(no flags — pure interactive picker: "pick one and press enter to stop it")* | `-p/--path`, `-q/--quiet`, `-v/--verbose`, `--workers` | **NOT a flag diff — a command-identity collision.** See §8.1: our `daemon` is a foreground shared watch/index server process; TS's `daemon` is an interactive picker over *already-running* background MCP daemons, to stop one. These are different commands that happen to share a name. Needs an explicit decision, not a flag patch. |
| `unlock [path]` | *(no flags)* | *(no flags, positional path)* | **Match.** |
| `callers <symbol>` | `-p/--path`, `-l/--limit` (default 20), `-j/--json` | `-p/--path`, `--limit`, `--json` | **Table stakes:** add `-l`/`-j` short aliases; confirm default 20. |
| `callees <symbol>` | same shape as `callers` | `-p/--path`, `--limit`, `--json` | Same as `callers`. |
| `impact <symbol>` | `-p/--path`, `-d/--depth` (**default 2**), `-j/--json` | `-p/--path`, `--depth` (**default 5, comment says "max 50"**) | **Real behavioral divergence, not just a flag:** TS defaults `impact` traversal to depth 2; ours defaults to 5 — materially different (much larger) blast-radius output for the same command with no flag. **Table stakes: change our default to 2** (or confirm intentional and document as divergence — but nothing in PROJECT.md suggests this was deliberate). Add `-d`/`-j` short aliases. |
| `affected [files...]` | `-p/--path`, `--stdin`, `-d/--depth` (default 5), `-f/--filter <glob>`, `-j/--json`, `-q/--quiet` | `-p/--path`, `--json` only | **Table stakes — largest single flag gap found:** missing `--stdin` (git-hook-friendly file-list-from-stdin), `-d/--depth`, `-f/--filter <glob>` (custom test-file pattern, e.g. `e2e/*.spec.ts`), `-q/--quiet` (undecorated path-only output — this is the exact shape a git hook or CI script would pipe). All four are needed for `affected` to be usable in the scripting contexts TS built it for. |
| `install` | `-t/--target <ids>` (comma ids or auto\|all\|none, **default: prompt**), `-l/--location` (global\|local, **default: prompt**), `-y/--yes` (non-interactive: location=global target=auto, **auto-allow ON**), `--no-permissions` (Claude Code only), `--print-config <id>` | `--target` (**default "auto"**, no prompt), `--location` (**default "global"**, no prompt), `--auto-allow` (**default false — opt-in**, inverted from TS's opt-out default) | **Real behavioral divergence:** (1) TS defaults to an *interactive prompt* when `--target`/`--location` are omitted; ours silently defaults to auto/global with no prompt — acceptable divergence IF intentional for a non-interactive-first CLI philosophy (worth an explicit decision, not a silent gap), but note it changes first-run UX materially. (2) TS's non-interactive path (`-y`) auto-allows permissions **by default**, opt-out via `--no-permissions`; ours defaults to **not** auto-allowing, opt-in via `--auto-allow` — an inverted default that changes what a scripted `codegraph install -y`-equivalent produces. (3) Missing `-y/--yes` and `--print-config <id>` entirely. **Table stakes:** add `--print-config` (used by agent-integration scripting) and reconcile the auto-allow default; **acceptable-divergence candidate:** the prompt-vs-flag-default UX split, if intentionally documented. |
| `uninstall` | `-t/--target` (default "all"), `-l/--location` (default: prompt), `-y/--yes` | `--target` (default "all", matches), `--location` (default "global", not prompt) | Missing `-y/--yes`; same prompt-default divergence as `install`. |
| `telemetry [action]` | positional `action` (`status`\|`on`\|`off`) — a real preference toggle, persisted | `Args: cobra.NoArgs` — static print-only statement, no toggle | **Acceptable, ALREADY-DECIDED divergence** (not a gap): our `internal/cli/telemetry.go` doc comment explicitly documents this as D-15/CLI-03 — codegraph-go has zero passive telemetry to toggle, so an on/off preference is meaningless; the command instead prints an honest, auditable "we collect nothing" statement. This is a deliberate, better answer to the same user need (trust), not a missed feature. No action needed beyond confirming this classification stands. |
| `upgrade [version]` | `--check`, `-f/--force` | `--check` only (per grep — verify `-f/--force` truly absent) | **Table stakes if confirmed missing:** add `-f/--force` (reinstall-even-if-current). |
| `version` | *(no flags — prints version, also aliased `-v`/`--version`)* | (confirm `version --json` per our Validated-requirements note) | Confirm TS has no `--json` on `version` (its `--help` shows none) — if ours adds `--json`, that's an **acceptable superset addition**, not a divergence to fix. |
| `serve` | `-p/--path`, `--mcp`, `--no-watch` | (confirm against `internal/cli/serve.go`) | Flag shape likely already matches per PROJECT.md; the **default value** of watch is the real gap — see §7. |
| — | *(TS has no `search`, `migrate` commands)* | `search`, `migrate` | Go-only extras, already flagged as such in milestone context — **acceptable divergence**, not parity gaps. `serve` exists in both (TS hides it from `--help` via `{hidden:true}` but it works when invoked — confirmed live). |

**Unresolved for this research pass (verify against source before planning):** our exact default values for `query --limit`, `callers/callees --limit`, `upgrade -f`, and `version --json` were inferred from grep, not exhaustively read — cheap to confirm with 30 seconds of `internal/cli/*.go` reading during planning, flagged here rather than guessed.

---

## 5. Worktree detection + notices (`sync/worktree.js`)

**Classification: TABLE STAKES.** Directly fixes "the silent worktree queries the main branch's graph" correctness gap the milestone calls out.

### 5a. Detection (`detectWorktreeIndexMismatch(startPath, indexRoot)`)

1. `gitWorktreeRoot(startPath)` = `git rev-parse --show-toplevel` from `startPath`, realpath-resolved, 5s timeout, `stdio:['ignore','pipe','ignore']` (silent on failure). Returns `null` if not a repo / git missing / timeout → **no warning, best-effort only.**
2. If `worktreeRoot === realpath(indexRoot)` → no mismatch (the index genuinely belongs here).
3. **Guard 1:** if `indexRoot` isn't ITSELF a git-worktree-root (`gitWorktreeRoot(realpath(indexRoot)) !== realpath(indexRoot)`) → no mismatch. This distinguishes "borrowed another worktree's index" from "the index just sits in a plain ancestor directory" (monorepo-subdir layouts), and avoids warning at all outside git.
4. **Guard 2 — submodule/embedded-clone exemption:** compare `gitCommonDir` (`git rev-parse --git-common-dir`, resolves the SHARED `.git` all worktrees of one repo point at) between `worktreeRoot` and `indexRoot`. If both resolve and DIFFER → they're genuinely different repositories (a submodule/embedded clone the parent index already covers by walking into it at index time) → **no mismatch, no warning**. Only same-common-dir-but-different-worktree-root counts as a genuine borrowed-worktree situation.
5. Returns `{worktreeRoot, indexRoot: resolvedIndexRoot}` on a real mismatch, else `null`.

### 5b. Two message formats

- **Verbose** (`worktreeMismatchWarning`, used by `status`'s human output): 4-line block — "This CodeGraph index belongs to a different git working tree.\n  Running in: X\n  Index from: Y\nResults reflect that tree's code (often a different branch), not this worktree — symbols changed only here are missing. Run \"codegraph init -i\" in this worktree for a worktree-local index."
- **Compact single-line** (`worktreeMismatchNotice`, used to PREFIX every other read-tool's result — both MCP AND CLI, since CLI shares the dispatch path): `⚠ CodeGraph results below come from a different git worktree (<indexRoot>), not where you're working (<worktreeRoot>) — they may reflect another branch, and symbols changed only here are missing. Run "codegraph init -i" here for a worktree-local index.` Prepended as `${notice}\n\n${originalOutput}`.

### 5c. Where it's wired (`mcp/tools.js:1057-1077, 1225-1254`)

- `codegraph_status` is **excluded** from the compact-notice wrapper (it has its own verbose §3 treatment) and is dispatched specially, never off-loaded to a worker pool.
- **Every other read tool** (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `files`, and by extension their CLI faces via the shared `ToolHandler.execute` dispatch) gets `withWorktreeNotice(result, projectPath)` applied centrally, after the tool's own handler runs, only on success (`!result.isError`).
- The mismatch check itself is **memoized per `(startPath, indexRoot)` pair for the process lifetime** ("that first verdict until restart" — comment references issue #926) — a `git rev-parse` subprocess isn't spawned on every single tool call, only once per distinct path pair.

**Implementation note for Go:** this must live at the query-dispatch/CLI-output layer (wrapping every read command's rendered output), not inside each command's own handler — exactly mirroring TS's centralized wrapper, not a per-command reimplementation. Given our CLI's `explore`/`node`/etc. currently call `eng.Explore()`/`eng.Node()` directly and print the result, the natural seam is a thin wrapper in `internal/cli/` (or `internal/query/Engine`) that all read-command `RunE`s pass through — analogous to TS's `ToolHandler.execute`.

---

## 6. Git sync hooks (`sync/git-hooks.js`) — scope correction from PROJECT.md

**Classification: TABLE STAKES for the mechanism, but narrower trigger scope than PROJECT.md currently implies.**

**Important finding:** PROJECT.md's "Target features" list describes this as "opt-in git sync hooks (post-commit/merge/checkout)" without qualification, which reads as always-offered. **It is not.** TS only offers to install git hooks when `offerWatchFallback()` runs — called from `init`/`index`'s CLI action (`bin/codegraph.js:545,588`) — AND `watchDisabledReason(projectPath)` is non-null, i.e. **only when the live file watcher is being disabled** (WSL2 `/mnt/*` drive, or `CODEGRAPH_NO_WATCH=1`). On a normal macOS/Linux project where the watcher runs fine, TS never mentions git hooks at all. This should be scoped into the v1.0 plan/requirements accordingly — not a standalone `codegraph git-hooks install` command, but a conditional prompt inside `init`/`index` gated on our own watch-disabled detection.

### 6a. Detection: `watchDisabledReason(projectRoot, probe)` (`sync/watch-policy.js:105-118`)

Precedence, first match wins:
1. `CODEGRAPH_NO_WATCH=1` env → disabled (explicit opt-out always wins)
2. `CODEGRAPH_FORCE_WATCH=1` env → enabled (overrides auto-detection)
3. WSL2 detection (`WSL_DISTRO_NAME`/`WSL_INTEROP` env, or `/proc/version` containing "microsoft"/"wsl") **AND** path matches `^/mnt/[a-z](/|$)` (single-letter Windows drive mounts only — deliberately excludes fast Linux mounts like `/mnt/wsl/...`) → disabled
4. Else → enabled (`null` reason)

**Go-relevance:** the WSL2/mnt case is Linux/Windows-specific and may not need a literal port if codegraph-go's fsnotify-based watcher doesn't share the same slow-recursive-watch pathology on WSL2 (verify — this is a Node.js `fs.watch`-specific perf issue, not necessarily one fsnotify inherits). At minimum, honor `CODEGRAPH_NO_WATCH`/`CODEGRAPH_FORCE_WATCH` env-var precedence for scripting/CI parity even if the WSL auto-detection itself doesn't port 1:1.

### 6b. Hook installation (`sync/git-hooks.js`)

- `DEFAULT_SYNC_HOOKS = ['post-commit', 'post-merge', 'post-checkout']` (post-merge covers `git pull`).
- Resolves hooks dir via `git rev-parse --git-path hooks` (honors `core.hooksPath` and worktrees correctly).
- Injects a marker-fenced block (`# >>> codegraph sync hook >>>` … `# <<< codegraph sync hook <<<`) that runs `codegraph sync` **backgrounded and fully silenced** (`( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1`) guarded by `command -v codegraph` (no-ops cleanly if the CLI isn't on PATH — matters for CI runners).
- **Idempotent**: re-install strips any prior marker block before re-appending — never duplicates. Preserves user-authored hook content outside the markers.
- **Uninstall** (`removeGitSyncHook`, wired to `codegraph uninit` in `bin/codegraph.js:625-635`): strips the marker block; deletes the hook file entirely if nothing but a shebang remains, else rewrites the user's surviving content. Also wired automatically into `uninit` — no separate flag needed.
- Interactive prompt at install time (`clack.select`) offers "Sync on git commit/pull/checkout (recommended)" vs. "I'll run `codegraph sync` myself"; `--yes`/non-interactive path defaults to installing hooks.

---

## 7. Watcher-on-MCP default (`serve --mcp`)

**Classification: TABLE STAKES.** Confirmed directly from source, not inference.

`bin/codegraph.js:1560-1577`: `serve` command declares `.option('--no-watch', …)`. Commander.js's `--no-X` convention means the option variable (`options.watch`) is `true` by default and only becomes `false` when `--no-watch` is explicitly passed. The action handler: `if (options.watch === false) { process.env.CODEGRAPH_NO_WATCH = '1'; }` — i.e., **absence of the flag leaves the watcher enabled**; presence of `--no-watch` routes through the same `CODEGRAPH_NO_WATCH` env-var chokepoint `watchDisabledReason` (§6a) already checks. There is no separate "watcher" wiring inside the MCP server path — it's the identical file-watcher module used by `codegraph sync`/`daemon`, just started unconditionally unless opted out.

Our current default is inverted (`--watch` opt-in). **Fix: flip the flag to `--no-watch` (Cobra's `BoolVar` with default `true`, or an explicit inverted-boolean flag) so watcher-on is the default, matching TS exactly** — this is explicitly called out as already-decided in PROJECT.md ("our `install` already writes the byte-identical `serve --mcp` invocation; only the watch default differs").

---

## 8. New/surprising divergence discovered during this research (not in the milestone's known-divergences list)

### 8.1 `daemon` command-name collision — different products, same name

TS's `codegraph daemon` (aliased `daemons`) is a **zero-flag interactive picker** over *already-running* background MCP daemon processes (one per project, spawned lazily by MCP clients — see `mcp/daemon.js`, `mcp/daemon-registry.js`, `mcp/daemon-manager.js`), used to select-and-stop one (`Manage running CodeGraph background daemons — pick one and press enter to stop it`).

Our `codegraph daemon` (`internal/cli/daemon.go`) is a **foreground, user-launched shared watch/index server** (`daemon.Run(ctx)`, blocks until Ctrl-C/SIGTERM) — a different concept entirely, closer to what TS's MCP server does internally when it lazily spawns a daemon, not to TS's CLI `daemon` command.

**This needs an explicit product decision, not a flag patch:**
- Option A: rename our current command (e.g. `codegraph daemon-run` or fold it into `serve`'s daemon mode) and build a TS-shaped `codegraph daemon` picker that lists/stops running instances of whatever daemon concept codegraph-go ends up using.
- Option B: keep our command as-is under the `daemon` name and treat this as a **documented, deliberate divergence** (our single-writer-lockfile daemon model differs enough from TS's per-project background-MCP-daemon model that a literal picker port may not even be meaningful) — but this must be a conscious call, since right now `codegraph daemon --help` produces materially different, surprising behavior for a user coming from TS muscle memory (`codegraph daemon` in TS never blocks the terminal; ours does).

Flag to requirements/roadmap: **resolve this explicitly before v1.0 ships**, since it's exactly the kind of silent surprise "drop-in parity" is supposed to eliminate. Note: PROJECT.md's v1.0 target features already call for "an interactive `daemon` picker" as TUI work — that item should be read as confirming Option A (rebuild `daemon` as a TS-shaped picker), which resolves this collision as a side effect. Worth flagging explicitly in requirements so it isn't lost as "just a TUI nicety."

### 8.2 `files --filter` semantic collision

Already captured in §4's table — worth restating because it's not just a missing flag, it's the **same flag name doing a different thing** (ours: language filter; TS: directory filter). This is the single highest-risk "looks like it works, does something different" landmine in the whole flag audit, because a script ported from TS (`codegraph files --filter src/`) would silently produce wrong (likely empty) output against our binary instead of erroring.

### 8.3 `impact` default depth (2 vs. 5) and `install`/`uninstall` default-prompt vs. default-auto

Both already captured in §4's table; called out again here because they're **behavioral defaults**, not additive flag gaps — a bare `codegraph impact Foo` or `codegraph install` run against both binaries with zero flags produces different results today. These are cheap, high-value parity fixes (one constant + one default value each) relative to the algorithmic work in §1-§2.

---

## 9. Classification summary

| Feature | Classification | Why |
|---|---|---|
| `explore` relevance algorithm (§1) | **Table stakes** | Core differentiator; current lexical-only matcher produces materially worse/wrong results (leaks test funcs, misses graph neighbors) |
| `node` multi-definition disambiguation (§2) | **Table stakes** | Named parity gap; currently silently drops N-1 of N same-named definitions |
| `status` rich content (§3) | **Table stakes** (data) / **differentiator** (human colorized rendering, i.e. TUI polish) | Data mostly already computed internally; renderer is the gap |
| Worktree detection + notices (§5) | **Table stakes** | Fixes a silent correctness bug (queries wrong branch's code) |
| Git sync hooks (§6) | **Table stakes**, narrower scope than PROJECT.md implies | Only triggers when watcher is disabled — scope the plan accordingly, don't build a standalone always-on command |
| Watcher-on-MCP default (§7) | **Table stakes** | One-line flag-default flip, already-decided in PROJECT.md |
| Flag short-aliases (`-l`,`-k`,`-j`,`-d`, etc.) across query/callers/callees/impact/status (§4) | **Table stakes** | Cheap, mechanical, breaks muscle-memory/scripts if skipped |
| `affected --stdin/--depth/--filter/--quiet` (§4) | **Table stakes** | Largest single flag gap; needed for the git-hook/CI scripting use case `affected` exists for |
| `impact` default depth, `install`/`uninstall` default semantics (§8.3) | **Table stakes** | Silent default-value divergences, not additive gaps |
| `files --filter` collision (§8.2) | **Table stakes** | Same name, different meaning — highest silent-failure risk in the audit |
| `daemon` command identity (§8.1) | **Table stakes, resolved by PROJECT.md's planned interactive picker** — flag explicitly in requirements so it isn't lost as pure TUI polish | Different product behind the same name today; the planned TUI daemon picker is the fix, but must be scoped as a parity fix, not just UX polish |
| `telemetry [action]` on/off toggle | **Acceptable divergence (already decided)** | Our zero-telemetry design makes a toggle meaningless; the static honest-statement is a better answer, already documented as D-15/CLI-03 |
| `search`, `migrate` commands (Go-only) | **Acceptable divergence** | Documented supersets, not gaps |
| Explore's adaptive per-project-size output budget, whole-file/skeleton rendering nuance (§1e) | **Differentiator-adjacent, lower priority** | Governs formatting, not selection; a fixed-cap interim is acceptable once selection (§1a-1d) is fixed |
| TS's Pebble-inapplicable `journalMode` field | **Acceptable divergence** | No meaningful Pebble analog; don't fabricate one |
| Install-time interactive prompt default vs. flag-default (`install`/`uninstall`) | **Candidate acceptable divergence, needs explicit sign-off** | Non-interactive-first CLI philosophy is defensible but changes first-run UX; document the choice rather than leave it implicit |

## Feature Dependencies

```
explore relevance rewrite (§1)
    └──requires──> computeGraphRelevance (RWR/PPR) port
    └──requires──> extractSymbolsFromQuery + stopword list port
    └──requires──> named-seed overload disambiguation (shares logic with §2's findSymbolMatches)
    └──enables──> accurate "no covering tests" warning (§1d) — needs root-only, direct-caller-only scoping fixed first

node disambiguation (§2)
    └──shares──> findSymbolMatches "enumerate all exact-name defs, generated-last" logic with explore's named-seeding (§1c.4)

worktree notice wrapper (§5)
    └──requires──> a centralized read-command dispatch seam (all CLI read commands currently print directly; needs a shared wrapper analogous to TS's ToolHandler.execute)
    └──enhances──> status (§3) — status gets the verbose form; every other read command gets the compact form

git sync hooks (§6)
    └──requires──> our own watchDisabledReason-equivalent detection (§6a) to know WHEN to offer them
    └──conflicts with──> assuming git hooks are always-offered (PROJECT.md's current phrasing) — scope correction needed

watcher-on-MCP default flip (§7)
    └──independent──> one flag-default change, no dependencies on the above

daemon picker rebuild (§8.1)
    └──requires──> whatever daemon-registry concept codegraph-go adopts (may need its own small design decision before the picker can be built)
    └──resolves──> the daemon command-identity collision as a side effect
```

## Sources

- TS CodeGraph v1.3.1 installed dist JS: `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/` — `context/index.js`, `mcp/tools.js`, `sync/worktree.js`, `sync/git-hooks.js`, `sync/watch-policy.js`, `bin/codegraph.js`, `index.js` (all read directly, line-cited above). **HIGH confidence — primary source, not documentation.**
- Live `codegraph <cmd> --help` output against the installed v1.3.1 binary (`/opt/homebrew/bin/codegraph`), all 22 shared+hidden commands — captured verbatim in §4. **HIGH confidence — direct tool execution.**
- Our own source: `internal/query/explore.go`, `internal/query/node.go`, `internal/query/status.go`, `internal/cli/{explore,node,status,query,files,callers,callees,impact,affected,install,uninstall,daemon,telemetry,upgrade,unlock}.go` — read directly for the flag-diff table and gap analysis. **HIGH confidence — primary source.**
- `.planning/PROJECT.md` (v1.0 Current Milestone section) — milestone framing and already-known divergences, used as the starting point this research extends/corrects (notably §6's git-hooks scope correction and §8's new findings). **HIGH confidence — project's own source of truth.**

---
*Feature research for: TypeScript CodeGraph v1.3.1 → codegraph-go v1.0 drop-in CLI parity*
*Researched: 2026-07-14*
