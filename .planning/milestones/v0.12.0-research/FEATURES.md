# Feature Research: Local Graph UI (v0.12.0)

**Domain:** Local, read-only, human-facing web UI over a precomputed code knowledge graph
**Researched:** 2026-08-22
**Confidence:** MEDIUM (cross-checked against 8+ independent primary/official sources — GitHub repos, official docs, project blogs, MSR research — none behind a paywall or requiring login; no source is single-anecdote-only except where explicitly flagged)

## Scope Note

This research is about the **feature space** the v0.12.0 UI should occupy, not the wire protocol or component library (already decided: ConnectRPC + Svelte/shadcn-svelte, see `PROJECT.md`). Every feature below states which `internal/query.Engine` method backs it, or flags that it needs data the Engine does not yet expose — the latter is a deliberate scope signal for REQ-ID definition, not an oversight.

## Landscape Surveyed

| Tool | Status (verified 2026-08-22) | Relevance |
|---|---|---|
| **SourceTrail** (formerly Coati) | **Archived Dec 14, 2021** by original authors. GPLv3, 16.5k stars. Discontinued from maintainer burnout after their startup closed — Qt (cross-platform) + LLVM-based per-language indexers as separate IPC processes + multi-build-system support was too much surface for volunteer maintenance; the project was also not commercially viable. No successor absorbed it (a KDE-incubator proposal was floated in the archival thread and never happened). | **Closest prior art** — the only tool in this list that did exactly "local, human, browse+visualize a precomputed code graph." Its design history (two rounds of user testing, a v1→v2 pivot) is directly load-bearing for this milestone. |
| **Sourcegraph** | Actively developed. Core is Apache-2.0. But Sourcegraph OSS (self-built from source with `enterprise/` dirs stripped) ships **only universal code search** — it explicitly excludes *all* code-intelligence features (jump-to-definition, code navigation, Batch Changes, Code Insights) behind a paid Enterprise license key, per Sourcegraph's own licensing docs. | Cautionary tale, not a feature model: the thing codegraph-go's whole value proposition rests on (precomputed navigation/call graph) is exactly what Sourcegraph paywalls. Confirms this milestone's local+free code intelligence is a real differentiator, not a "everyone already has this" feature. |
| **go-callvis** / **goexplorer** | Active. Go-specific call-graph visualizer: pointer analysis → Graphviz dot → served as an interactive SVG over a local HTTP server. | Nearest same-language, same-shape prior art for the Graph view: focus-on-click, package/type grouping, stdlib/unexported filtering. Its own README states the goal of "locally store the call graph data and provide quick access... show an interactive map of overall dependencies... then by selecting a package show its call graph" — almost exactly this milestone's Graph view spec. |
| **madge** | Active. JS/TS module dependency grapher. | Simplest possible "useful" convention for a file/package graph: color by structural role (has-deps / leaf / **in-a-cycle**), because cycles are the one thing developers scan a dependency graph specifically to find. |
| **dependency-cruiser** | Active, well-documented. | Most mature prior art for *making a big dependency graph legible*: `focus` (target + N-hop neighbors), `reaches`, `highlight`, and folder/package-level aggregation reporters (`archi`, `ddot`). Its own FAQ states plainly: a monorepo graph with 5,000 modules "will not give you much information" rendered flat. |
| **VS Code Call Hierarchy / Find All References** | Active, mature (shipped 2019, refined since). | The table-stakes UX bar for a callers/callees browsing experience: tree + preview, direction toggle, breadcrumb, double-click-to-navigate. |
| **OpenGrok** | Active (Oracle), release 1.14.13 shipped May 2026 (added dark mode). | Table-stakes bar for the Browse & Inspect view's code-reading ergonomics: syntax-highlighted xref, "what symbols does this file define" panel, "what function am I inside while scrolling" panel. |
| **Graph-hairball literature** | N/A (research + tool ecosystem) | Direct evidence for the Graph view's design constraints — see the dedicated section below. |

## Feature Landscape

### Table Stakes (Users Expect These)

Missing any of these will make the UI feel unfinished next to what developers already use daily in their editor and in every other code browser.

| Feature | Why Expected | Complexity | Engine Mapping |
|---|---|---|---|
| Search-as-you-type over symbols and files | Every comparable tool leads with this — SourceTrail's autocomplete popup, OpenGrok's search bar, every IDE's "Go to Symbol." It's the primary entry point, not a nice-to-have. | LOW | `Query`/`Search` already return ranked, limited results — this is a debounced client call, no new Engine surface |
| Syntax-highlighted source display | Table stakes since OpenGrok (2005-era tool) and every editor. Plain `<pre>` text reads as unfinished. | LOW-MEDIUM | `Node` already returns verbatim source; highlighting is a client-side concern (e.g. Shiki/highlight.js by language) |
| Jump-to-definition semantics on symbol references | The single most repeated user behavior across SourceTrail, OpenGrok, VS Code: click a name, land on its definition. | LOW-MEDIUM | `Node` + `Query` resolve a symbol to its definition location; UI wires clicks in rendered source to a client-side route change |
| Click a caller/callee/neighbor and keep going (graph traversal as navigation, not just display) | This is SourceTrail's entire value proposition (click node → activates it → all views update) and VS Code Call Hierarchy's core loop (double-click drills in). A dead-end node view is a regression from what any comparable tool already does. | LOW-MEDIUM | `Callers`/`Callees`/`Node` already return the neighbor set; this is client-side re-navigation on the same calls |
| Deep-linkable, shareable URLs per view/symbol/query | Standard web-app expectation (every SPA with meaningful views does this — see Foxglove/tldraw's "deep links" patterns). Its absence is felt specifically when a developer wants to paste a link in Slack/a PR comment pointing a teammate at "look at this blast radius." | LOW | Pure client-side routing concern — URL encodes `{view, symbol|file, depth, limit}` and re-issues the same Engine-backed RPCs on load. No new Engine surface. |
| Back/forward navigation history | SourceTrail has a dedicated Back/Forward/History/Home toolbar; browsers already give this for free via the History API IF routes are real URLs (see deep-linking above) — this is really the payoff of doing deep-linking right, not a separate feature. | LOW | Free once routing is URL-based; no Engine involvement |
| Copy-path / copy-symbol-name affordance | Small but constantly noticed when missing — every code browser and IDE has a "copy path" context action; developers paste these into terminals, chat, `grep` invocations constantly. | LOW | Pure client-side; data already present in `Node`/`Files` results |
| Keyboard navigation (search focus shortcut, arrow-key result selection, `Esc` to close) | SourceTrail (Ctrl/Cmd+F-style Find On-Screen), OpenGrok, and every IDE all support this; power users will reach for it immediately and be annoyed by a mouse-only UI. | LOW-MEDIUM | Pure client-side |
| "Where am I" context while reading a file (breadcrumb / current-symbol indicator) | OpenGrok's "Scopes" window exists specifically because scrolling a long file loses context of which function you're in — a genuinely load-bearing small feature, not decoration. | MEDIUM | Needs a "which symbol contains this line" answer. `Node`'s file-mode read + the file's symbol list (from `Files`/`Query` scoped to that file) can derive this client-side by line-range containment; **no new Engine method needed**, but it is real client-side computation, not a passthrough |
| Empty/error states for "no index," "stale index," "symbol not found" | Every one of these is a real, frequent state (a repo not yet `codegraph init`'d, a watcher-detected stale graph, a typo'd search). A UI that just shows a blank pane instead of a clear state reads as broken. | LOW | `Status` already exposes `Initialized`/`Stale`; `Query`/`Node` already return not-found errors — UI needs to render these, not compute them |
| Multi-definition disambiguation display | codegraph-go's own `Node` behavior already handles ambiguous symbol names (multi-def rendering, per `render_status.go`/`node.go`); the UI must expose the same disambiguation the CLI/MCP already do, or it silently regresses UX parity with the agent-facing surface. | LOW-MEDIUM | `Node(symbol, file, line)` already resolves ambiguity when `file`/`line` are supplied; UI needs a picker when a bare symbol search is ambiguous — same data `enumerateSymbolDefs` already computes |

### Differentiators (Competitive Advantage)

These are only possible because codegraph-go already has a real precomputed graph with call edges, blast-radius computation, and a semantic-relevance `Explore`. Generic code browsers (grep-based, or LSP-single-file-scoped) cannot do these cheaply or at all.

| Feature | Value Proposition | Complexity | Engine Mapping |
|---|---|---|---|
| One-click blast radius / impact view with tweakable depth | Impact analysis ("what breaks if I change this") is normally a manual, error-prone mental exercise even with an IDE's Find Usages (which only shows depth-1 direct callers). Depth-N impact as a first-class, instant, visual result is the single biggest gap between what an IDE gives you and what this milestone can give for free. | MEDIUM | `Impact(symbol, depth)` — direct mapping, UI adds a depth slider that re-issues the RPC |
| "What breaks if I touch these files" (Affected) as an interactive workbench, not a CI-only check | Most tools that compute file-level blast radius (dependency-cruiser's `reaches`, madge's `depends`) are batch/CLI-only and require re-running a full analysis. Here it's already indexed and instant, so it can be an exploratory tool a developer uses BEFORE opening a PR, not just a CI gate after. | LOW-MEDIUM | `Affected(files, depth)` — direct mapping; UI adds a multi-file picker (likely fed by the file tree from `Files`) |
| Semantic-relevance `Explore` as a natural-language entry point into Browse | Every comparable tool (SourceTrail, OpenGrok, VS Code) requires the user to already know the exact symbol/file name to start from. `Explore`'s query-driven relevance selection (already proven for agents via MCP) is a genuinely novel entry point for a *human* who only has a fuzzy question ("where does auth happen"), not an exact identifier. | MEDIUM | `Explore(query, maxFiles)` already returns verbatim source + call paths + blast radius in one call — the UI's "search" bar can route a natural-language-shaped query here as a fallback/parallel path to symbol search, distinct from `Query`/`Search`'s exact/fuzzy-name matching |
| Aggregated file/package-level graph reading real precomputed edges, not a client-side re-scan | go-callvis/madge/dependency-cruiser all *recompute* the dependency graph at tool-invocation time from source. This project's rollup is "largely reading edges that already exist" (per PROJECT.md's own rationale for choosing file/package granularity) — meaning the graph view can be instant and always current with the index, not a multi-second batch job. | MEDIUM-HIGH (see dedicated Graph View section below) | **Needs new Engine surface**: no existing method returns "distinct file→file (or package→package) edges with aggregated per-kind counts." `Files`, `Node`, and the raw edge iteration used internally by `Status`'s `edgesByKind` scan are the closest existing primitives, but none returns a *rolled-up, cross-file* adjacency list keyed by file/package pairs. This is the single largest genuinely-new piece of backend work this milestone implies — call it out explicitly as a scope item, not an assumption that "the Engine already has this." |
| Live-updating views on re-index (watcher push), not "refresh and hope" | None of the surveyed local tools (SourceTrail, go-callvis, OpenGrok) push live updates to an open browser tab on a background re-index — they all require a manual refresh/re-run. codegraph-go already has a watcher; streaming its re-index events to an open UI (already scoped via ConnectRPC server-streaming, per PROJECT.md) means the UI never quietly goes stale mid-session, which is a real, felt problem in every comparable local tool. | MEDIUM (already scoped at the protocol layer; feature-level work is per-view "apply this delta / show a banner and let the user refresh") | Needs the watcher's existing re-index event source wired to a streaming RPC; no `query.Engine` method changes required, this is plumbing between `internal/watcher` (or wherever fsnotify events currently land) and the new server layer |
| Worktree-mismatch surfaced as a first-class, visual warning (not a buried CLI flag) | Every comparable tool assumes "the index describes the code on disk in front of you." codegraph-go already detects and reports the specific case where that's false (querying a linked worktree against the main checkout's stale graph) — a correctness problem none of the surveyed tools even model. Making this loud and visual in Index Health, rather than a warning line a human has to notice in text output, converts an existing correctness feature into differentiated trust. | LOW | `WorktreeMismatch`/`Status().WorktreeMismatch` — direct mapping, purely a rendering upgrade from the existing MCP/CLI notice |

### Anti-Features (Attractive Traps for This Project's Constraints)

Each of these looks like an obvious win in isolation and is wrong specifically because this UI is **local, read-only, single-binary, no-cloud**.

| Feature | Why It Looks Appealing | Why It's a Trap Here | Alternative |
|---|---|---|---|
| In-browser code editing | Every "modern" code tool eventually grows an editor (Sourcegraph did, GitHub.dev does). Feels like the natural next step from "viewing" source. | codegraph-go's whole surface is deliberately **read-only by construction** (PROJECT.md: "read-only by construction, with bind address and auth kept as explicit seams"). Editing requires write-conflict handling, file-locking against the user's actual editor, and turns a query tool into a second IDE this project has no reason to compete with. | Deep-link/handoff to the user's real editor at a specific file:line (the SourceTrail "Show in IDE" pattern, via a local `vscode://file/...`-style URI or similar), never an in-browser edit surface |
| Multi-user auth / accounts / sharing across machines | Sourcegraph's entire enterprise product is built on this, and it looks like the "obvious" path to make the UI a team tool. | This is explicitly the boundary PROJECT.md draws: "Local-first, no cloud. Hosted platform features stay permanently out of scope," and the milestone scope itself says bind/auth are "explicit seams so widening to `--host` later is a change, not a rewrite" — i.e. deliberately NOT built now. Building auth now is speculative work against a future that may never come, on a tool whose only current transport is loopback. | Loopback-only bind + Origin/Host validation (already the shipped design); if team access is ever wanted, that's a distinct future milestone with a distinct threat model, not a checkbox added here |
| Persisted user state (saved views, pinned bookmarks, per-user dashboards) stored server-side | SourceTrail's bookmark manager (a real, well-loved feature) is the obvious model to copy. | SourceTrail's bookmarks live in a sidecar file *of the SourceTrail project format* — a legitimate local-first pattern, but codegraph-go has no user-account concept and no server-side persistence layer at all today; building one just for bookmarks is new stateful infrastructure (a write path!) bolted onto an otherwise strictly read-only tool, for a feature that duplicates what deep-linkable URLs (table stakes above) already give you almost for free — a saved URL IS a bookmark. | Deep-linkable URLs double as bookmarks (browser bookmark manager, no server state); if a curated "named views" feature is wanted later, it can be a local file the user manages themselves (e.g. exported/imported JSON), never a server-persisted account concept |
| Force-directed "show me the whole graph" view as the default landing experience | It's the most visually impressive demo — a big glowing web of nodes looks like "AI-powered code intelligence." Every dependency visualizer (madge, go-callvis, emerge) *can* render one. | This is the single most-documented failure mode in the literature surveyed: dependency-cruiser's own FAQ says a 5,000-module graph "will not give you much information"; the MSR "Trimming the Hairball" paper is entirely about how force-directed layouts on real-sized graphs become unreadable node/edge soup; `aryx/codegraph` abandoned node-link entirely for large codebases after years of trying. codegraph-go indexes real, possibly-large monorepos (per PROJECT.md's "monorepo scale" architecture constraint) — a whole-symbol-graph force layout would be an instant hairball on any repo big enough to need this tool. This is exactly why PROJECT.md already scoped the Graph view at file/package granularity, not whole-symbol. | File/package-level aggregated graph (already the scoped choice) with focus+context interaction (see next section) as the default; a whole-symbol force graph, if ever offered at all, should be an opt-in, scoped-down view (e.g. only within one already-drilled-into package), never the landing page |
| Real-time collaborative cursors / "who's viewing what" presence | Feels like table stakes for a "modern" tool if you've used Figma/Notion/Google Docs recently. | No multi-user concept exists or is wanted (see auth anti-feature above); this requires exactly the server-side session/identity infrastructure this project has deliberately not built, for a single-developer local tool where "who else is viewing this" has no answer by construction (loopback bind = one machine, typically one person). | None needed — if a team wants shared context, that's Slack-a-deep-link, which table-stakes deep-linking already provides |
| A generic embeddable graph-database query language / power-user query console | Feels like it "future-proofs" the Query Workbench — why build 4 fixed forms (impact/affected/callers/callees) when you could expose one flexible query box? | The milestone's own Query Workbench is explicitly scoped to the 4 existing Engine methods with tweakable depth/limit — a free-form query language is a new query planner, a new security surface (arbitrary traversal cost against a local server with no auth), and duplicates work the schema/Engine team would need to design deliberately, not organically grow from a UI milestone. | Structured forms per existing Engine method (impact/affected/callers/callees), each with its own depth/limit controls — exactly what's already scoped |
| Rich cloud-sync'd metrics/analytics dashboard (churn, complexity, hotspots) | Tools like `emerge` layer git-churn heatmaps and SLOC/fan-out complexity scores on top of their dependency graphs, and it demos well. | None of that data exists in the current schema (`schema.Node`/`schema.Edge`/`schema.Meta` — verified against `internal/query` call sites above — carry structural graph facts, not git-history or complexity metrics). Adding it means a new extraction pass, not a UI change, and risks scope creep from "graph browser" into "code-quality dashboard," a different product. | Index Health (already scoped) stays structural: freshness, coverage, languages, staleness, worktree-mismatch — not code-quality metrics. If churn/complexity is ever wanted, it's a future indexer-extraction milestone with its own schema work |

## What a Genuinely Useful File/Package Dependency Graph View Looks Like

This is the milestone's highest-execution-risk view, per the anti-features section above (default force-directed hairball is the single most well-documented failure mode across every source surveyed). Concretely, from dependency-cruiser, go-callvis, madge, and the hairball literature:

1. **Aggregation is not optional, it's the point.** dependency-cruiser ships three distinct granularities (module/`dot`, folder/`ddot`, architectural/`archi`) precisely because "show everything" never works past a small project. codegraph-go's milestone scope already lands on file/package granularity for the same reason — this is validated, not a novel risk. The Engine gap: today there's no method that returns a rolled-up file↔file (or package↔package) adjacency with aggregated per-kind edge counts; this has to be built new (see the differentiators table above), most cheaply as a read-time aggregation over the same full edge scan `Status`'s `edgesByKind` already performs, keyed by `(srcFile, dstFile, kind)` instead of just `kind`.

2. **Focus + context, not filter-and-lose-orientation.** dependency-cruiser's `focus` (target + N configurable hops of neighbors) is the single most reusable pattern here: a developer drilling into one file wants that file plus its immediate dependency neighborhood, not the whole repo and not an isolated single node with zero context. This maps directly onto the milestone's own stated interaction ("a whole-repo picture... drill into a file for its symbols") — the "whole-repo picture" is the *overview* (aggregated, low-detail), and drilling in is exactly dependency-cruiser's focus+depth move, reusing `Impact`/`Affected`'s existing depth-parameter pattern the Query Workbench already has, for interaction consistency across the two views.

3. **Cycle highlighting is cheap and high-value.** madge's entire visual grammar reduces to one signal that matters most: is this node part of a cycle. A file/package graph is exactly the level at which import cycles are both meaningful (a real architectural smell) and computable from the same edge data the aggregated graph already needs — this should be close to free once the aggregated adjacency exists, and should be a first-class visual distinction (not a stat buried in a table), matching madge's red/blue/green convention.

4. **Grouping/clustering must be structural, not force-simulated.** `emerge`'s Louvain-community-detection-on-force-graph approach exists specifically because raw force-directed layout alone collapses into a "Big Ball of Mud" at real scale — but community detection is itself a new algorithmic capability, not currently in `internal/query`. The cheaper, already-available structural grouping for this milestone is the one the repo itself already has: **directory/package structure**, which requires no new algorithm, just rendering the aggregated file graph with directory boundaries as visual groups (the same idea as dependency-cruiser's `ddot` folder rollup). Community detection is explicitly reserved in the schema for a *future* milestone (per PROJECT.md's Out of Scope: "community detection... the schema anticipates them... but does not implement them") — this milestone should use structural (directory) grouping, not attempt graph-theoretic community detection from scratch.

5. **Layered/hierarchical layout beats force-directed for this specific structure.** File/package dependency graphs are usually near-DAGs (import cycles are the rare, flagged exception per point 3) — a layered layout (top-to-bottom or left-to-right by dependency direction, à la Graphviz `dot`'s default `rankdir`, which go-callvis and dependency-cruiser both default to) reads far more legibly than force-directed for this shape, because direction (what depends on what) is the actual information content, and force layouts discard direction as a first-class visual signal.

## What Makes an Index Health View Load-Bearing, Not a Stats Dump

The Engine's `Status()` already computes everything below in one call — the risk here is entirely about *presentation*, not missing data. A stats dump lists numbers; a load-bearing health view answers "should I trust what I'm about to look at, and if not, why."

1. **Every number needs a trust verdict attached, not just a value.** `Stale: true` and `WorktreeMismatch != nil` are the two facts that mean "everything else on this page might be lying to you" — SourceTrail, OpenGrok, and go-callvis have no equivalent concept at all (they all assume the index and the disk agree), which is exactly the gap this milestone's differentiators table calls out as a real advantage. The health view should surface these as blocking/warning banners *above* the raw counts, not as two more rows in a table — this is a rendering priority decision, not new data (`StatusResult.Stale`, `StatusResult.WorktreeMismatch`).
2. **Coverage needs a denominator, not just a count.** "312 Go files indexed" is a stats-dump number; "312 of 340 discovered Go files indexed, 28 skipped (list why)" is load-bearing. Today's `Status()` reports `FilesByLanguage` (indexed counts) but has no "discovered but not indexed / errored during extraction" concept in its return shape — this is a genuine Engine gap if "why is my file missing" is meant to be answerable from the health view, and should be called out as a scope question for REQ-ID definition rather than assumed available.
3. **Reindex-recommended needs an action, not just a flag.** `Index.ReindexRecommended` (schema-version mismatch) is already computed; a load-bearing view surfaces it as an actionable prompt ("this index predates the current schema, re-run `codegraph index`"), not a boolean buried in a nested JSON-shaped table.
4. **Freshness needs a "how stale, since when" story, not just a boolean.** `Stale` is currently a bool (sidecar file present, or newest-mtime-newer-than-last-sync) with no timestamp of *when* it went stale or *how much* changed — genuinely useful staleness UX (SourceTrail's own weakness: it has no live-staleness concept at all, requiring a manual "Refresh") would want to say "stale since 14:32, 3 files changed," which the live watcher push (already scoped for this milestone) can supply as an event stream even if `Status()`'s point-in-time snapshot cannot. This is a case where the live-push differentiator and the health view are the same feature wearing two names — plan them together.
5. **Language/kind breakdowns should invite drill-down, not just enumerate.** `NodesByKind`/`EdgesByKind`/`FilesByLanguage` are already rich per-kind maps; a stats dump prints them as a table, a load-bearing view makes each row clickable into a pre-filtered `Files`/`Query` view (e.g. click "python: 40 files" → the Browse view's file list pre-filtered to `Filter: "python"`, an existing `FilesOptions.Filter` field) — connecting Index Health to Browse & Inspect rather than leaving it as a terminal dead-end screen.

## Feature Dependencies

```
Search-as-you-type (table stakes)
    └──requires──> Query/Search (existing Engine)

Jump-to-definition + neighbor-click navigation (table stakes)
    └──requires──> Deep-linkable URLs (table stakes) — navigation without shareable state is a worse regression than no navigation at all

Back/forward history (table stakes)
    └──requires──> Deep-linkable URLs (table stakes) — this IS the payoff of real routing, not a separate build

Impact/Affected workbench (differentiator)
    └──requires──> Deep-linkable URLs (table stakes) — a depth/limit/symbol combination is exactly the kind of state worth sharing

File/package Graph view (differentiator)
    └──requires──> NEW Engine surface: aggregated file/package adjacency (does not exist today)
    └──enhances──> Query Workbench's Impact/Affected (clicking a graph edge should be able to seed a workbench query)

Cycle highlighting (Graph view detail)
    └──requires──> File/package Graph view's aggregated adjacency (same new Engine surface)

Live push (differentiator)
    └──requires──> ConnectRPC server-streaming (already decided at protocol layer, PROJECT.md)
    └──enhances──> Index Health's staleness story (freshness needs "since when," which point-in-time Status() alone cannot give)

Worktree-mismatch visual warning (differentiator)
    └──requires──> WorktreeMismatch (existing Engine method) — no new data, pure rendering priority

"Where am I" breadcrumb while reading source (table stakes)
    └──requires──> Node (file mode) + a symbol-list-for-this-file lookup — client-derivable from existing Files/Query scoped by path, no new Engine method, but real client-side logic

In-browser editing (anti-feature)
    └──conflicts──> Read-only-by-construction constraint (PROJECT.md) — do not build

Multi-user auth (anti-feature)
    └──conflicts──> Loopback-only bind + no-cloud constraint (PROJECT.md) — explicit seam for later, not now

Server-persisted bookmarks (anti-feature)
    └──conflicts──> No server-side write path exists or is wanted; deep-linkable URLs (table stakes) already subsume the use case
```

### Dependency Notes

- **Deep-linkable URLs is the single highest-leverage table-stakes item.** Three other table-stakes features (jump-to-definition navigation, back/forward, copy-path-as-shareable-context) and one differentiator (Impact/Affected workbench) all either require it or are meaningfully worse without it. Sequence it early.
- **The aggregated file/package adjacency is the single highest-leverage NEW backend item.** Both the Graph view and its cycle-highlighting detail depend on it, and it does not exist in `internal/query` today (confirmed by reading `engine.go`, `status.go`, `node.go`, `files.go`, `traverse.go` — the closest existing primitive is `Status()`'s unfiltered full edge scan for `edgesByKind`, which tallies by kind only, not by file-pair). Flag this explicitly for REQ-ID scoping rather than assuming it falls out of `Files`/`Impact`/`Affected` as-is.
- **Live push and Index Health staleness are the same feature from two angles.** Building the watcher→ConnectRPC streaming plumbing once and wiring both the "views update in place" differentiator and the health view's "stale since when" story to it avoids building two separate staleness UIs.

## MVP Definition

### Launch With (v1 of this milestone)

- [ ] Search-as-you-type (symbols + files) — the only entry point every comparable tool leads with
- [ ] Browse & Inspect: node view (verbatim source + callers/callees + blast radius), click-through neighbor navigation — this IS the milestone's stated Browse feature and directly reuses `Node`/`Callers`/`Callees`
- [ ] Deep-linkable URLs + back/forward — table stakes that unlocks three other features cheaply; building it last is the expensive order
- [ ] Query Workbench: impact/affected/callers/callees as structured forms with depth/limit — direct mapping onto four existing Engine methods, no new backend work
- [ ] Index Health: freshness/coverage/languages/staleness/worktree-mismatch, rendered with trust-verdict banners (stale/mismatch) above raw counts — almost entirely a `Status()` rendering task
- [ ] Jump-to-definition + copy-path affordances in rendered source — small, expected, low complexity

### Add After Core Validates (v1.x within this milestone)

- [ ] File/package Graph view with directory-structural grouping and layered (not force-directed) layout — gate on the new aggregated-adjacency Engine work landing first; do not let Graph view block shipping Browse+Workbench+Health
- [ ] Cycle highlighting in the Graph view — trivial once the aggregated adjacency exists, so sequence right after the base Graph view
- [ ] Live push wired to Browse/Workbench view updates — protocol-layer work is already scoped; feature-level "apply the delta" work per view can follow once the base views are stable
- [ ] "Where am I" breadcrumb while scrolling a file — real but small client-side logic; nice-to-have relative to the four bullets above

### Explicitly Deferred Beyond This Milestone

- [ ] Community-detection-based clustering in the Graph view — schema explicitly reserves this for later (PROJECT.md Out of Scope); directory-structural grouping is the correct v1 substitute, not a placeholder to feel bad about
- [ ] Coverage denominator ("discovered but not indexed" reporting) — genuine Engine gap; only build if Index Health's "why is my file missing" question is explicitly in scope for this milestone's REQ-IDs
- [ ] Anything from the Anti-Features table — editing, auth, server-persisted state, force-directed whole-graph view, collaborative presence, free-form query console, churn/complexity metrics

## Sources

- [CoatiSoftware/Sourcetrail](https://github.com/coatisoftware/sourcetrail) — GitHub repo, archived status, license, star count (MEDIUM confidence, official/primary)
- [Discontinue Sourcetrail (project blog)](https://vuink.com/post/fbheprgenvy-d-dpbz/blog/discontinue_sourcetrail) — official discontinuation rationale (MEDIUM confidence, official/primary, cross-checked against the GitHub issue thread)
- [Sourcetrail DOCUMENTATION.md](https://github.com/CoatiSoftware/Sourcetrail/blob/master/DOCUMENTATION.md) — full interaction model: search/graph/code triad, activate/expand/collapse/hide, bookmarks, on-screen search bar, tabs (MEDIUM confidence, official/primary)
- [Why working on Chrome made me develop a tool for reading source code (Coati/SourceTrail founder blog)](https://medium.com/@egraether/why-working-on-chrome-made-me-develop-a-tool-for-reading-source-code-7111ba21a6f0) — design history, the v1→v2 user-study-driven pivot from "show everything" to "focus on active symbol," explicit citation of Shneiderman's Visual Information Seeking Mantra (MEDIUM confidence, official/primary)
- [Better code understanding with Sourcetrail — C++ Stories](https://www.cppstories.com/2017/10/sourcetrail/) — independent third-party walkthrough, cross-checks the official documentation (MEDIUM confidence, cross-checked)
- [Sourcegraph self-hosted Terms of Service](https://sourcegraph.com/terms/self-hosted) and [sourcegraph/handbook licensing.md](https://github.com/sourcegraph/handbook/blob/main/content/departments/product/process/gtm/licensing.md) — OSS-vs-Enterprise feature split, license terms (MEDIUM confidence, official/primary)
- [go-callvis](https://github.com/ondrajz/go-callvis) and [go-callvis docs site](https://ondrajz.github.io/go-callvis/) — interactive viewer feature set, flags, goexplorer spinoff goal (MEDIUM confidence, official/primary)
- [pahen/madge](https://github.com/pahen/madge) — circular dependency detection API and color convention (MEDIUM confidence, official/primary)
- [dependency-cruiser options-reference.md](https://github.com/sverweij/dependency-cruiser/blob/main/doc/options-reference.md) and [faq.md](https://github.com/sverweij/dependency-cruiser/blob/main/doc/faq.md) — focus/reaches/highlight/archi/ddot, explicit "5000 modules" readability guidance (MEDIUM confidence, official/primary)
- [VS Code Call Hierarchy feature request #16110](https://github.com/microsoft/vscode/issues/16110) and [UI testing issue #71083](https://github.com/microsoft/vscode/issues/71083) — UX design deliberation and shipped interaction model (MEDIUM confidence, official/primary)
- [C++ Extension Call Hierarchy blog](https://devblogs.microsoft.com/cppblog/c-extension-in-vs-code-1-16-release-call-hierarchy-more/) — shipped-feature description (MEDIUM confidence, official)
- [Find All References improvement suggestions #31720](https://github.com/microsoft/vscode/issues/31720) — community pain points on grouped-vs-flat reference lists (MEDIUM confidence, primary community source)
- [Trimming the Hairball (Microsoft Research)](https://www.microsoft.com/en-us/research/wp-content/uploads/2018/12/TrimmingTheHairball.pdf) — force-directed layout failure mode at scale, edge-cutting mitigation (MEDIUM confidence, peer-reviewed research)
- [aryx/codegraph](https://github.com/aryx/codegraph) — DSM/tabular alternative to node-link graphs for large codebases, explicit rationale (MEDIUM confidence, official/primary)
- [glato/emerge](https://github.com/glato/emerge) — Louvain-community-detection-on-force-graph, "Big Ball of Mud" failure-mode example (MEDIUM confidence, official/primary)
- [oracle/opengrok Features wiki](https://github.com/oracle/opengrok/wiki/Features), [User Interface wiki](https://github.com/oracle/opengrok/wiki/User-Interface), and [release 1.14.13](https://github.com/oracle/opengrok/releases/tag/1.14.13) — active-maintenance status, xref/Navigate/Scopes features (MEDIUM confidence, official/primary)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/{engine,status,node,files,traverse,explore,search}.go` — read directly to ground every Engine-mapping claim in this document in the actual current method signatures and doc comments, not assumption (HIGH confidence, primary source code)
- `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/PROJECT.md` and `.planning/seeds/SEED-001-local-svelte-shadcn-graph-browsing-ui.md` — milestone scope, constraints, and out-of-scope boundaries (HIGH confidence, primary project source)

---
*Feature research for: local code-intelligence/graph-browsing UI, v0.12.0 milestone*
*Researched: 2026-08-22*
