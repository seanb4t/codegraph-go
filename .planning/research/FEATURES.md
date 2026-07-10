# Feature Research

**Domain:** Local-first code knowledge graph / code intelligence MCP server for AI coding agents
**Researched:** 2026-07-10
**Confidence:** MEDIUM-HIGH (official docs + README are HIGH confidence primary sources; exact numeric benchmarks vary slightly release-to-release and should be treated as directional, not exact)

## Sources of truth used

- TS CodeGraph GitHub README — `github.com/colbymchenry/codegraph` (v1.3.1, current as of 2026-07-09, MIT license, ~58k stars)
- Docs site `colbymchenry.github.io/codegraph/` — CLI reference, MCP server reference, Integrations, Languages, Indexing guide, Resolution & Frameworks core-concepts page
- Independent review (andrew.ooo, 2026-05-28) — captures an earlier snapshot (v pre-1.3, Apache-2.0-labeled, 31k stars, tool named `codegraph_context`) — useful for competitive framing and architecture description, but **do not trust its license/tool-naming details over the current docs site**, which is authoritative and more recent.
- Comparative landscape research (Ry Walker "Code Intelligence Tools for AI Agents Compared", Sourcegraph context-compare, probe/Aider/Serena ecosystem pages)

## Feature Landscape

### Table Stakes — the TS CodeGraph v1.3.x parity surface

This is not a hypothetical "what users expect" list — it is the literal feature inventory of TS CodeGraph v1.3.x, since parity with it IS this project's table-stakes bar per PROJECT.md.

| Feature | Why Expected (parity requirement) | Complexity | Notes |
|---------|--------------|------------|-------|
| `codegraph init [path]` / `uninit [path]` | Creates/removes `.codegraph/` and builds the full graph in one step; `uninit --force` for non-interactive removal | LOW | Directory + first full index run |
| `codegraph install` / `uninstall` (bare `codegraph` launches interactive installer) | Detects installed agents, writes MCP config, injects marker-fenced instructions into `CLAUDE.md`/`AGENTS.md`/`GEMINI.md`; `uninstall` cleanly removes both | MEDIUM | Must special-case each agent's config format (see Integrations below); marker-fencing so re-runs are idempotent and user edits outside markers survive |
| `codegraph index [path]` (full rebuild, `--force`, `--quiet`, `--verbose`) | Deterministic from-scratch graph build | MEDIUM | This is the parser + resolver pipeline entry point |
| `codegraph sync [path]` (incremental, `--quiet`) | "Reparses only what changed" — what the watcher calls on every edit | MEDIUM-HIGH | Requires content-hash/mtime diffing and incremental edge recomputation, not just re-run-full-index-on-subset |
| `codegraph status [path] --json` | Health/statistics check (node/edge counts, last sync time, staleness) | LOW | |
| `codegraph unlock [path]` | Removes stale lock files after crashed/killed daemon | LOW | Signals the design uses a lock file around DB writes — important for concurrent-access architecture goal |
| `codegraph query <search> --kind --limit --json` | Symbol search (name-based) | LOW-MEDIUM | Backed by FTS5 in TS version |
| `codegraph explore <query>` (+ `codegraph_explore` MCP tool) | THE flagship feature: natural-language or symbol/file query → verbatim line-numbered source of relevant symbols grouped by file, PLUS call paths between them and a blast-radius summary, in one round trip | HIGH | This is the single highest-value feature and the reason the tool exists — replaces the grep→Read loop. Must nail this for parity to mean anything |
| `codegraph node <symbol\|file>` (+ `codegraph_node` MCP tool) | Symbol detail (source + caller/callee) or raw file-with-line-numbers read | LOW-MEDIUM | |
| `codegraph search` MCP tool | Locations-only symbol search (no source body) — lighter-weight than explore/node | LOW | |
| `codegraph callers <symbol>` / `codegraph callees <symbol>` (+ MCP equivalents) | Reverse/forward call-graph traversal | MEDIUM | Requires resolved call edges, not just declarations |
| `codegraph impact <symbol> --depth` / `codegraph_impact` | Transitive blast-radius analysis from a symbol | MEDIUM-HIGH | Depends on accurate call graph + depends-on edges |
| `codegraph affected [files...]` | Identify impacted **test** files from a changed-file set | MEDIUM-HIGH | Needs a distinct "test discovers/covers subject" edge type, separate from generic call edges |
| `codegraph files [path] --format --filter --pattern --max-depth --json` (+ `codegraph_files` MCP tool) | Indexed file/directory structure without a filesystem scan | LOW-MEDIUM | Returned from the graph's file table, not `os.ReadDir` — must stay in sync with watcher |
| `codegraph_status` MCP tool | Index health/statistics via MCP (not just CLI) | LOW | |
| Auto-sync file watcher (`codegraph serve --mcp` runs it) | Native FSEvents/inotify/ReadDirectoryChangesW watcher, debounced (default 2000ms, tunable via `CODEGRAPH_WATCH_DEBOUNCE_MS`), consolidates edit bursts into one sync | HIGH | Core "no manual re-runs" promise in PROJECT.md. Must be per-OS native (not polling) for the perf story to hold |
| Per-file staleness banner | While a sync is pending/debouncing, the agent-facing tool output includes a `⚠️` warning that referenced files may be stale, nudging the agent to read the file directly | LOW-MEDIUM | Small but important trust/correctness feature — prevents agents silently acting on stale graph data |
| Connect-time catch-up reconciliation | On MCP server (re)connect, reconciles filesystem state via stat comparison + content hashing to catch changes made while the watcher was offline | MEDIUM | Needed because daemon/session lifecycle is decoupled from editor lifecycle |
| Background daemon (`codegraph daemon`) | Shared indexing/watch server reused across multiple agent sessions/processes; falls back to in-process mode on WSL2/Windows in some cases | HIGH | Directly relevant to the "concurrent access" architecture goal in PROJECT.md |
| MCP server (`codegraph serve --mcp`), tool visibility control via `CODEGRAPH_MCP_TOOLS` env var | Only `codegraph_explore` is exposed by default; 7 more tools (`node`, `search`, `callers`, `callees`, `impact`, `files`, `status`) exist but are unlisted unless the env var allowlists them | MEDIUM | Deliberate context-budget design: fewer tool definitions in the agent's system prompt by default. Server exposes **zero** tools if no `.codegraph/` exists — agent falls back to built-ins gracefully |
| Language coverage: 20-30+ languages via tree-sitter, full vs partial-support tiers | Go, Java, C#, Python, TypeScript/JavaScript, Rust, PHP, Ruby, C/C++, Swift (full), Objective-C (explicitly partial: "`.mm` ObjC++ may parse incompletely"), Kotlin, Scala, Dart, Svelte, Vue, Astro, Liquid, Pascal/Delphi, Lua, Luau, R, plus long-tail (COBOL, Solidity, Terraform/OpenTofu, Nix, Erlang, VB.NET, CUDA, Metal, ArkTS) | VERY HIGH (cumulative) | PROJECT.md's own priority order (Go→Java/C#→Python→TS/JS→rest) matches this project's actual usage, not TS CodeGraph's ordering — sequence a subset first, treat long-tail languages as backlog, not launch-blocking |
| Framework-aware routing (17 frameworks) | Detects routing files/annotations (Django, Flask, FastAPI, Express, NestJS, Laravel, Rails, Spring, Gin, Axum, ASP.NET, Vapor, Rocket, React Router, etc.) and emits `route` nodes linked by `references` edges to handler functions/classes — so "who calls this controller" surfaces the URL pattern | HIGH | Validated per-framework accuracy in TS README ranges ~90-100% on canonical apps for URL-pattern frameworks, dropping for convention-heavy frameworks (Rails ~89.6%) — set expectations accordingly, don't over-promise 100% |
| Cross-language / dynamic-dispatch bridging ("synthesizers") | Bridges what static parsing can't resolve directly: Swift↔Objective-C, React Native legacy bridge/TurboModules/Fabric/Expo Modules, callback/observer patterns, EventEmitter channels, React state→re-render, JSX component wiring, interface→implementation dispatch | VERY HIGH | Each synthesized edge is tagged `provenance: 'heuristic'` with source location — **this provenance tagging itself is a table-stakes feature**, not optional, since it lets downstream consumers (and this port) distinguish ground-truth AST edges from best-effort heuristics |
| SQLite storage + FTS5 full-text search, WAL mode | `.codegraph/codegraph.db`, write-ahead logging for concurrent reader access | MEDIUM | This project explicitly plans a *new* storage format (per PROJECT.md), but WAL-mode concurrent access is the bar the new format must clear or exceed |
| `codegraph upgrade [version]` | Self-update mechanism | LOW-MEDIUM | For a static Go binary this likely becomes "download+swap binary" rather than npm-style upgrade — different mechanism, same user-facing command |
| `codegraph telemetry [on|off]` | Opt-out anonymous aggregated usage stats; explicitly never captures code, paths, symbol names, queries, or IPs | LOW | Privacy posture is part of the product's trust story — replicate the "never captures X" guarantee explicitly, don't just silently drop telemetry |
| `codegraph help [command]` / `codegraph version` | Standard CLI ergonomics | LOW | |
| 8-agent integration roster with agent-specific install logic | Claude Code, Cursor, Codex CLI, opencode, Hermes Agent, Gemini CLI, Antigravity IDE, Kiro — Cursor needs an injected `--path` arg to fix MCP subprocess cwd issues; instruction-file agents get marker-fenced sections | HIGH (breadth) | PROJECT.md commits to the same roster — each integration is individually small but the roster is wide; budget time per-agent, not as one lump task |
| 100% local, no cloud calls, no API keys, no embeddings/vector DB | Core trust proposition — pure structural graph from AST, not semantic/embedding search | N/A (property, not a build item) | This is a *design constraint* achieved by NOT building embeddings, not a feature to implement |
| Migration path for existing indexes | Not a TS CodeGraph feature (TS has no "migrate from an older format" story) but a **hard requirement unique to this port** per PROJECT.md, since the new storage format breaks compatibility with `.codegraph/` SQLite from the TS tool | HIGH | Table stakes *for this project specifically*, not part of TS parity — the one place parity list and this project's requirements diverge from "copy the source" |

### Differentiators (Competitive Advantage for the Go Port)

These are not in TS CodeGraph's feature list — they are where this project earns its "better than the original" claim, matching PROJECT.md's Core Value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Single static binary, no bundled runtime | TS CodeGraph bundles its own Node.js runtime for the `curl | sh` install — a meaningful supply-chain and disk/startup-time cost. A static Go binary eliminates an entire class of "which runtime version shipped" and "npm install spent 40s" complaints | MEDIUM | Directly falsifiable via install-time and binary-size benchmarks vs TS |
| Verifiable supply chain: cosign signing, SLSA provenance, SBOM, reproducible builds, vuln-scan-gated CI | TS CodeGraph has no equivalent public claims; this is a genuine differentiator for security-conscious/enterprise adopters and open-source credibility | HIGH | This is infrastructure work, not a runtime feature — but it's explicitly called out in PROJECT.md as a first-release requirement, not an afterthought |
| Published head-to-head benchmarks vs TS CodeGraph (indexing throughput, query latency, memory) on real repos | Proves the performance claim rather than asserting it — mirrors the credibility play TS CodeGraph itself used (its README benchmark methodology, per the independent review, is what drove its own adoption/trending) | MEDIUM | Reuse TS CodeGraph's own benchmark methodology (`claude -p` headless, N runs, median-of-N, raw per-repo numbers published) for direct comparability and credibility |
| Storage format designed for concurrent multi-process access + incremental updates + monorepo scale from day one | TS CodeGraph's SQLite+WAL is good but was retrofitted (daemon exists partly to work around single-writer SQLite contention, per its own "shared server across sessions" design and the existence of `codegraph unlock` for stale locks). Designing for concurrency up front avoids that retrofit | HIGH | This is the architectural bet called out as an open research question in PROJECT.md — worth a dedicated ARCHITECTURE.md deep-dive, not just a feature bullet |
| v1 architecture that anticipates a central/team server without building it | Nothing comparable exists in TS CodeGraph (single-repo, local-only by design). This sets up milestone 2 (shared/CI-distributed indexes, multi-user queries) without committing to it now | MEDIUM (as v1 constraint) | Table-stakes-adjacent: don't build server features now, but don't paint the storage layer into a local-only corner either |
| Pure-Go (or WASM-sandboxed) parsing avoiding CGo where possible | Improves cross-compilation, static-binary story, and — if WASM/wazero route is chosen — sandboxes third-party grammar code, itself a supply-chain hardening win beyond what TS CodeGraph (plain Node + native tree-sitter bindings) offers | HIGH | This is the open parser-strategy question flagged in PROJECT.md; STACK.md/ARCHITECTURE.md should carry the CGo-vs-WASM-vs-native tradeoff, not FEATURES.md |
| Faster cold-start full index and incremental sync | Go's startup time and lack of JIT-warmup beats Node's cold start; combined with efficient concurrent parsing (goroutines vs Node's single-threaded event loop + worker_threads), incremental sync latency is a believable, benchmarkable edge | MEDIUM-HIGH | This is the actual "better" in "works the same or better" from PROJECT.md's Core Value — treat it as a P1 validation target, not a vague hope |

### Anti-Features (Deliberately Not Building — v1)

| Feature | Why it seems appealing | Why problematic for this project (v1) | What to do instead |
|---------|---------------|------------------|-------------|
| Embeddings / vector search / semantic similarity ranking | Competing tools (Claude Context, grepai, ai-grep) use embeddings for "fuzzy" conceptual queries and get cited as complementary to structural graphs | Explicitly out of character for the product — TS CodeGraph's own positioning is "no embeddings, no vector DB, no API keys, deterministic and lossless." Adding embeddings changes the trust/privacy story (cloud API calls, drift on rename) and duplicates the differentiator this project inherits for free by staying local-structural-only | Stay pure AST/graph-based, as TS CodeGraph does. If semantic/fuzzy search is ever wanted, it's a v2+ decision made deliberately, not smuggled in |
| Central graph server / multi-user remote queries / auth | Natural next step once a graph exists, and PROJECT.md's own milestone-2 note makes it tempting to start early | Explicitly out of scope for v1 per PROJECT.md ("v1 architecture must anticipate it, not implement it") — building it now delays shipping the parity+perf value and adds an auth/network attack surface before the core product is validated | Design storage/process boundaries to not preclude it (e.g., avoid embedding SQLite-file-path assumptions deep in query logic), but don't write server code in v1 |
| CI-built shared index distribution/caching | Would help monorepo/team scenarios and large CI runs | Same milestone-2 deferral as above — requires a distribution/trust model (who signs a shared index? how is staleness detected across machines?) that hasn't been designed yet | Defer; note as an explicit v2 architecture input so v1 doesn't block it |
| Hosted platform / PR analysis service (getcodegraph.com-style) | Adjacent product with revenue potential, tempting scope creep once the core engine works | Different product, different buyer, different distribution model — conflates "index engine" with "SaaS business" and would slow the core parity/perf mission | Not this project's goal; if pursued, it's a separate product built *on* this engine, later |
| Bundling a runtime with the binary | Simpler for some install paths (matches TS CodeGraph's Node-bundling approach) | Directly contradicts the core differentiator — "the static binary IS the distribution." Bundling anything reintroduces the exact supply-chain/size/startup cost this port exists to eliminate | Ship a genuinely dependency-free static binary per platform; if CGo-tree-sitter is chosen, isolate it to build-time linking, not a bundled runtime |
| Auto-editing / refactoring / code modification via the graph | Tools like Serena position themselves as "LSP-grade navigation *and* editing" — tempting to one-up TS CodeGraph by adding write capability | TS CodeGraph itself is read-only/exploration-focused; adding edit/refactor capability is a different trust boundary (agents can now mutate code through this tool, not just read it) and is explicitly outside "full parity" scope | Stay read-only/exploration, matching TS CodeGraph's actual scope; leave editing to the host agent's existing tools |
| New capabilities beyond TS parity in v1 (extra languages, extra MCP tools, new query types) | Once deep in the graph internals, it's easy to see "quick wins" beyond the original tool | PROJECT.md is explicit: "Feature additions beyond TS parity in v1 — parity plus performance is the bar; new capabilities wait for v2" | Track any such ideas in a backlog for v2; resist scope creep during the parity port |
| Polling-based file watching as a fallback-everywhere strategy | Simpler to implement uniformly across OSes than three native watcher APIs | TS CodeGraph deliberately uses native FSEvents/inotify/ReadDirectoryChangesW for the debounced-watcher perf story; polling would regress both latency and CPU usage, undermining the "better" performance claim | Implement native watchers per OS (fsnotify or equivalent in Go covers this well), reserve polling only as an explicit degraded-mode fallback (e.g., certain network filesystems), not the default |

## Feature Dependencies

```
[Tree-sitter/parser layer: symbol + edge extraction]
    └──requires──> [Language grammar per supported language]

[codegraph index (full build)]
    └──requires──> [Parser layer]
    └──requires──> [Storage schema: nodes/edges/files tables]

[codegraph sync (incremental)]
    └──requires──> [codegraph index] (must exist as baseline)
    └──requires──> [Content-hash/mtime diffing]

[File watcher + debounce]
    └──enhances──> [codegraph sync] (triggers it automatically)

[Per-file staleness banner]
    └──requires──> [File watcher + debounce] (needs pending-sync state to report)

[Cross-file resolution: imports, call edges, type inheritance]
    └──requires──> [codegraph index] (full symbol table must exist first)

[codegraph callers / callees]
    └──requires──> [Cross-file resolution] (call edges must be resolved)

[codegraph impact / affected]
    └──requires──> [codegraph callers / callees] (blast radius = graph traversal over call edges)
    └──requires──> [Test-coverage edge type] (affected specifically, beyond generic impact)

[Framework-aware routing]
    └──requires──> [Cross-file resolution] (route→handler is itself a resolved reference edge)

[Cross-language bridging ("synthesizers")]
    └──requires──> [Cross-file resolution]
    └──enhances──> [codegraph explore / impact] (heuristic edges widen blast-radius accuracy for dynamic dispatch)

[codegraph_explore MCP tool]
    └──requires──> [codegraph node/search equivalents] (explore composes symbol lookup + call paths + blast radius in one call)
    └──requires──> [Cross-file resolution]

[MCP server tool-visibility allowlist]
    └──requires──> [All individual MCP tools implemented] (allowlist only gates visibility, not existence)

[Background daemon]
    └──enhances──> [File watcher, MCP server] (shared process avoids redundant watchers/indexes across sessions)
    └──requires──> [Storage format supporting concurrent access] (WAL or equivalent)

[Agent installer/uninstaller per agent]
    └──requires──> [MCP server] (nothing to configure agents to point at otherwise)

[Migration tool (TS SQLite → new format)]
    └──requires──> [New storage schema finalized]
    └──conflicts with──> [starting migration work before schema is stable] (schema churn invalidates migration logic repeatedly)

[Release integrity suite: signing, SLSA, SBOM, reproducible builds]
    └──enhances──> [Single static binary distribution] (the binary is the thing being signed/attested)
    └──independent of──> [core graph/CLI features] (can be built in parallel, doesn't block feature work)
```

### Dependency Notes

- **Everything downstream of `codegraph index` requires cross-file resolution before it's useful**: callers/callees, impact, affected, and framework routing all sit on top of resolved edges, not just raw per-file symbol extraction. Sequence resolution early — a phase that ships raw AST extraction without resolution will not support the majority of table-stakes commands.
- **`codegraph_explore` is a composition, not a primitive**: it's built from the same primitives as `node`/`search`/`callers`/`callees`/`impact`, plus a synthesis step (grouping, call-path assembly, blast-radius summarization). Build the primitives first; `explore` becomes an orchestration layer over them, matching the "unlisted by default" design where `explore` is the only default-visible tool but the others already exist underneath.
- **Migration tooling conflicts with early schema churn**: don't start the TS→Go migration converter until the new storage schema is stable, or it becomes a moving target that gets rewritten repeatedly. Sequence it after storage format lands, even though it's a "table stakes for this project" item.
- **Release integrity (signing/SLSA/SBOM) is architecturally independent of the graph/CLI feature work** and can proceed on a parallel track from day one, per PROJECT.md's "not an afterthought" framing — it should be set up as CI infrastructure early rather than bolted on right before v1 ships.
- **Anti-feature conflicts**: embeddings/vector search would conflict with (undermine) the "100% local, deterministic, no API keys" trust property that is itself a table-stakes/differentiator feature — these are mutually exclusive design directions, not features to combine.

## MVP Definition

### Launch With (v1) — full TS CodeGraph v1.3.x parity, per PROJECT.md

Not a trimmed MVP in the traditional sense — PROJECT.md explicitly rejects "core-first" in favor of full parity, because a drop-in-swap product needs the whole surface to validate. Within that mandate, sequence by dependency order:

- [ ] Parser layer + storage schema (nodes/edges/files) — foundation for everything
- [ ] `codegraph index` / `sync` / `status` / `unlock` — core indexing lifecycle
- [ ] Cross-file resolution (imports, calls, type inheritance) — unlocks callers/callees/impact/explore
- [ ] `codegraph query` / `node` / `search` (CLI + MCP) — basic retrieval
- [ ] `codegraph callers` / `callees` / `impact` / `affected` — dependency analysis surface
- [ ] `codegraph explore` (CLI + `codegraph_explore` MCP, default-visible tool) — the flagship feature; without it, parity is nominal, not real
- [ ] `codegraph files` (CLI + MCP) — structure browsing
- [ ] File watcher + debounce + staleness banner + connect-time catch-up — the "no manual re-runs" promise
- [ ] `codegraph init`/`uninit`, `install`/`uninstall` for the 8-agent roster — without this, nobody can actually adopt it
- [ ] MCP server with `CODEGRAPH_MCP_TOOLS` visibility allowlist — matches TS agent-facing contract exactly
- [ ] Language support in priority order (Go → Java/C# → Python → TS/JS → remainder) — per PROJECT.md, not TS's own ordering
- [ ] Framework-aware routing for the frameworks relevant to priority languages first (Gin/Go, Spring/Java, ASP.NET/C#, Django/Flask/FastAPI/Python, Express/NestJS/TS) — defer long-tail frameworks
- [ ] Migration tool from TS `.codegraph/` SQLite — required for the "uninstall TS, install Go, migrate, everything works" success bar
- [ ] Static binary release for macOS/Linux/Windows, signed + SLSA + SBOM + reproducible — required from first release per PROJECT.md, not deferred
- [ ] Published benchmarks vs TS CodeGraph — required for v1 per PROJECT.md's own requirements list

### Add After Validation (v1.x)

- [ ] Cross-language bridging synthesizers (Swift↔ObjC, React Native bridge/TurboModules/Fabric) — high complexity, lower priority given Go/Java/Python-first language order; add once core languages are solid
- [ ] Long-tail language support (COBOL, Solidity, Terraform/Nix, Pascal/Delphi, etc.) — trigger: real user demand or explicit parity gap reports, not speculative completeness
- [ ] Remaining framework routing coverage (Rails, Vapor, Rocket, Laravel, etc.) — trigger: once priority-language frameworks are solid and accuracy-validated
- [ ] `codegraph daemon` shared-server optimization across sessions — can launch with simpler per-session indexing first if daemon design proves complex, then add shared-process optimization

### Future Consideration (v2+)

- [ ] Central graph server / multi-user remote queries / auth — explicitly milestone 2 per PROJECT.md
- [ ] CI-built shared index distribution/caching — explicitly milestone 2
- [ ] Any capability beyond TS parity (new query types, new MCP tools, semantic/embeddings layer) — explicitly deferred to v2 per PROJECT.md's "parity plus performance is the bar" constraint

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Parser + storage schema | HIGH | HIGH | P1 |
| Cross-file resolution | HIGH | HIGH | P1 |
| `codegraph explore` / `codegraph_explore` MCP | HIGH | HIGH | P1 |
| `index`/`sync`/`status`/`unlock` lifecycle | HIGH | MEDIUM | P1 |
| File watcher + debounce + staleness banner | HIGH | HIGH | P1 |
| `callers`/`callees`/`impact`/`affected` | HIGH | MEDIUM-HIGH | P1 |
| `node`/`search`/`files` (CLI+MCP) | MEDIUM-HIGH | LOW-MEDIUM | P1 |
| Agent install/uninstall (8-agent roster) | HIGH | HIGH (breadth) | P1 |
| MCP tool-visibility allowlist | MEDIUM | LOW | P1 |
| Priority-language coverage (Go/Java/C#/Python/TS/JS) | HIGH | VERY HIGH | P1 |
| Migration tool from TS SQLite | HIGH (for adoption) | HIGH | P1 |
| Static binary + signing/SLSA/SBOM/reproducible builds | HIGH (differentiator) | HIGH | P1 |
| Published benchmarks vs TS | HIGH (proof of Core Value) | MEDIUM | P1 |
| Framework-aware routing (priority-language frameworks) | MEDIUM-HIGH | HIGH | P2 |
| Cross-language bridging synthesizers | MEDIUM | VERY HIGH | P2 |
| Long-tail language support | LOW-MEDIUM | HIGH (cumulative) | P3 |
| Long-tail framework routing | LOW-MEDIUM | MEDIUM (cumulative) | P3 |
| `codegraph daemon` shared-process optimization | MEDIUM | HIGH | P2 |
| Central server / CI-distributed indexes | HIGH (future) | VERY HIGH | P3 (v2) |
| Embeddings/semantic search | N/A (anti-feature) | — | Do not build |

**Priority key:**
- P1: Must have for launch (TS parity core + this project's unique release-integrity/migration requirements)
- P2: Should have, add when possible (broader accuracy/coverage on top of a working core)
- P3: Nice to have, future consideration (long-tail completeness or v2-scoped capabilities)

## Competitor Feature Analysis

| Feature | TS CodeGraph (parity target) | Serena (MCP, LSP-based) | Sourcegraph/SCIP | probe / Aider repo-map | This Project's Approach |
|---------|--------------|--------------|-----------------|------------------------|--------------------------|
| Retrieval model | Structural AST graph, deterministic, no embeddings | LSP-grade symbol navigation + editing | Precise SCIP index, cross-repo, exhaustive | ripgrep speed + tree-sitter AST, lexical/structural hybrid | Same as TS CodeGraph — structural graph, deterministic |
| Scope | Single local repo | Single local repo (per-project LSP) | Org-wide, multi-repo, multi-host | Single local repo | Single local repo (v1); architected to not preclude multi-repo/server later |
| Editing capability | Read-only/exploration | Reads AND edits/refactors | Read/navigate (enterprise tooling around it) | Read-only | Read-only, matching TS CodeGraph scope |
| Infra requirement | None (local SQLite, bundled Node runtime) | Language servers per language | Hosted/self-hosted Sourcegraph instance | None (CLI binary) | None — single static binary, no bundled runtime (improvement over TS) |
| Auto-sync on edits | Yes, native FS watcher + debounce | Depends on LSP server behavior | Requires re-indexing/CI pipeline | No persistent index (stateless per-call) | Yes, matching TS CodeGraph |
| Blast-radius/impact analysis | Yes (`impact`, `affected`) | No (navigation-focused, not graph-analytic) | Partial (via precise refs, not dedicated impact tool) | No | Yes, parity requirement |
| Supply-chain integrity story | None published | N/A | Enterprise trust via hosted product | N/A | Signed, SLSA-attested, SBOM'd, reproducible builds — differentiator |
| Distribution | npm / curl-sh installer bundling Node runtime | pip/uv or similar per ecosystem | Hosted SaaS or self-hosted install | Binary/crate | Single static binary per platform, no runtime bundling — differentiator |
| Team/multi-user features | None (explicitly local-only) | None | Core strength (org-wide) | None | Deferred to v2 by design, architected for in v1 |

## Sources

- [colbymchenry/codegraph GitHub README](https://github.com/colbymchenry/codegraph) — v1.3.1, CLI commands, MCP tools, languages, integrations, benchmark summary (HIGH confidence, primary/official)
- [CodeGraph docs — CLI reference](https://colbymchenry.github.io/codegraph/reference/cli/) (HIGH confidence, primary/official)
- [CodeGraph docs — MCP Server reference](https://colbymchenry.github.io/codegraph/reference/mcp-server/) (HIGH confidence, primary/official)
- [CodeGraph docs — Integrations](https://colbymchenry.github.io/codegraph/reference/integrations/) (HIGH confidence, primary/official)
- [CodeGraph docs — Languages reference](https://colbymchenry.github.io/codegraph/reference/languages/) (HIGH confidence, primary/official)
- [CodeGraph docs — Indexing guide](https://colbymchenry.github.io/codegraph/guides/indexing/) (HIGH confidence, primary/official)
- [CodeGraph docs — Resolution & Frameworks core concepts](https://colbymchenry.github.io/codegraph/core-concepts/resolution/) (HIGH confidence, primary/official)
- [CodeGraph Review — andrew.ooo (2026-05-28)](https://andrew.ooo/posts/codegraph-review-pre-indexed-knowledge-graph-claude-code/) (MEDIUM confidence — independent, but a slightly earlier snapshot than current docs; useful for architecture narrative and competitive framing, cross-checked against official docs for factual details)
- [Code Intelligence Tools for AI Agents Compared — Ry Walker Research](https://rywalker.com/research/code-intelligence-tools) (MEDIUM confidence — third-party competitive landscape synthesis)
- [Sourcegraph — AI Coding Context Tools Compared](https://sourcegraph.com/resources/context-compare) (MEDIUM confidence — vendor-authored but factually grounded comparison)
- [probe (probelabs) GitHub](https://github.com/probelabs/probe) (HIGH confidence for its own feature claims, primary source)
- Web search synthesis on Aider repo-map, ctags/LSIF-based indexers, Serena, ripgrep/ast-grep layering for AI agent code search (MEDIUM confidence, aggregated from multiple independent write-ups)

---
*Feature research for: local-first code knowledge graph / code intelligence MCP tool (Go port of TS CodeGraph)*
*Researched: 2026-07-10*
