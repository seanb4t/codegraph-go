# Architecture Research

**Domain:** Local-first code knowledge-graph indexer / code intelligence tool for AI coding agents (Go port of TypeScript CodeGraph)
**Researched:** 2026-07-10
**Confidence:** HIGH (Go/Rust indexer architecture patterns cross-verified across 7+ real open-source implementations); MEDIUM (TS CodeGraph internals — sourced via a single GitHub README fetch, not primary docs)

## Standard Architecture

Every code-graph indexer surveyed (gleann, tessera, code-review-graph-go, code-context, graphindex, arbor, doctree, plus Sourcegraph's SCIP ecosystem) converges on the same five-layer shape. TS CodeGraph itself follows it. This is not incidental — it's the shape forced by the problem (parse → resolve → store → query → serve), and it's the shape that lets you swap the storage layer later without touching the rest.

### System Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         ACCESS LAYER (per-invocation)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────────────┐   │
│  │  CLI (cobra)  │  │ MCP server   │  │ Installer / uninstaller       │   │
│  │  init/index/  │  │ (stdio, long-│  │ (agent config writers:        │   │
│  │  explore/     │  │  running per │  │  Claude Code, Cursor, Codex,  │   │
│  │  migrate      │  │  agent sess.)│  │  OpenCode, Gemini, etc.)      │   │
│  └──────┬───────┘  └──────┬───────┘  └───────────────┬───────────────┘   │
├─────────┼──────────────────┼──────────────────────────┼──────────────────┤
│         │           QUERY ENGINE (read path)          │                  │
│         │  explore / call-paths / blast-radius / search│                 │
│         │  — talks ONLY to the GraphStore port, never  │                 │
│         │    to SQL/driver code directly                │                 │
├─────────┴──────────────────────────────────────────────┴──────────────────┤
│                    INCREMENTAL UPDATE ENGINE (write path)                 │
│  ┌────────────────┐   ┌───────────────────┐   ┌─────────────────────┐   │
│  │ fsnotify watcher│──▶│ Dirty-file tracker │──▶│ Two-pass indexer     │   │
│  │ (debounce 100ms-│   │ (content-hash +    │   │ (local extract, then │   │
│  │  60s window)     │   │  dependency-aware  │   │  cross-file link)    │   │
│  └────────────────┘   │  invalidation)      │   └──────────┬───────────┘   │
│                        └───────────────────┘              │               │
├──────────────────────────────────────────────────────────┼───────────────┤
│                     EXTRACTION PIPELINE                   ▼               │
│  ┌───────────┐   ┌────────────────┐   ┌──────────────────────────────┐  │
│  │ File walker│──▶│ Parser (per-lang│──▶│ Symbol/Edge extractor        │  │
│  │ (gitignore-│   │ tree-sitter,    │   │ (single parse per file,      │  │
│  │  aware)    │   │ behind a Parser │   │  shared AST for symbols+calls│  │
│  │            │   │ interface)      │   │  + framework-route heuristics)│  │
│  └───────────┘   └────────────────┘   └──────────────┬───────────────┘  │
├───────────────────────────────────────────────────────┼───────────────────┤
│                        STORAGE LAYER (GraphStore port)  ▼                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  GraphStore interface: PutSymbols, PutEdges, GetSymbol, Callers,     │ │
│  │  Callees, BlastRadius, Search, DeleteFile, Migrate                   │ │
│  └───────────────────────┬────────────────────────────┬────────────────┘ │
│           ┌───────────────▼──────────────┐   ┌─────────▼──────────────┐  │
│           │ v1: SQLite adapter (WAL,     │   │ v2 (future, same       │  │
│           │ single-writer, FTS5, recursive│   │ interface): gRPC/HTTP  │  │
│           │ CTE traversal)                │   │ client → central       │  │
│           │ .codegraph/index.db           │   │ graph server, CI-built │  │
│           └───────────────────────────────┘   │ shared indexes         │  │
│                                                 └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|-------------------------|
| File walker | Enumerate source files, respect `.gitignore`/ignore config, detect language by extension | `filepath.WalkDir` + gitignore matcher |
| Parser | Turn file bytes into an AST for one language | tree-sitter grammar behind a `Parser` interface — CGo (`smacker/go-tree-sitter`), WASM+wazero (`malivvan/tree-sitter`), or native-Go (`gotreesitter`) implementation swapped per build/per-language |
| Symbol/Edge extractor | Walk the AST once, emit `Symbol` (function/class/type/route) and *local* `Edge` (calls, imports) structs | Tree-sitter S-expression queries, compiled+cached per language; one parse shared by symbol extraction and call extraction |
| Two-pass indexer | Pass 1: per-file local extraction (parallelizable across files). Pass 2: cross-file link/resolve pass that turns unresolved references (e.g. `foo()` call) into resolved edges pointing at a specific symbol ID, using a global symbol table built after pass 1 | Worker pool (goroutines = NumCPU) for pass 1; single-threaded or sharded resolver for pass 2 since it needs global visibility |
| Dirty-file tracker | Decide what needs re-parsing after a change: same-file content hash AND transitive dependents (files that referenced changed symbols) | SHA-256 content hash cache (bbolt/SQLite) + reverse-dependency lookup in the graph itself |
| Watcher | OS-level file change notifications, debounced | `fsnotify` (Linux inotify / macOS FSEvents / Windows ReadDirectoryChangesW under the hood) |
| GraphStore (storage port) | The **only** boundary between "how we ask questions about the graph" and "where the graph physically lives" | Go interface; v1 = SQLite adapter, v2 = remote adapter — see Anti-Patterns for why this boundary is the single most important architectural decision here |
| Query engine | Bounded graph traversals: call paths, blast radius (BFS/DFS to depth N), symbol search, file/module summaries | Recursive CTEs against SQLite in v1; same queries become RPCs against a server in v2 — the query engine's Go code doesn't change, only which GraphStore implementation it's wired to |
| MCP server | Long-lived stdio process per agent session; exposes `codegraph_explore` and companion tools; the natural home for the watcher's writer role while it's running | `mcp-go` or hand-rolled JSON-RPC over stdio; holds the single writer lock for its lifetime |
| CLI | One-shot commands: `init`, `install`, `uninstall`, `uninit`, `upgrade`, `explore`, `migrate` | `cobra`; acquires writer lock only for write commands (`index`, `migrate`), otherwise read-only |
| Installer | Write/remove per-agent config files (`.claude/`, `.cursor/`, etc.) that register the MCP server | Registry of per-agent adapters (one file per agent), not a growing switch statement |
| Migration tool | One-way converter: TS CodeGraph's `codegraph.db` (old schema) → new Go schema | Reads old SQLite read-only, streams rows into new schema via the same `GraphStore.PutSymbols/PutEdges` write path used by the indexer |

## Recommended Project Structure

```
codegraph-go/
├── cmd/
│   └── codegraph/                 # main package, cobra root command wiring
├── internal/
│   ├── types/                     # Symbol, Edge, File, Occurrence — the shared domain model.
│   │                               # Zero dependencies on SQL/tree-sitter/MCP. Everything else depends on this.
│   ├── parser/                    # Parser interface + per-language implementations
│   │   ├── parser.go               # type Parser interface { Parse(ctx, path, content) (*AST, error) }
│   │   ├── golang/                 # first language, per PROJECT.md priority order
│   │   ├── java/ csharp/ python/ typescript/ ...  # added later, same interface
│   │   └── registry.go             # extension → Parser lookup, mirrors storage registry pattern below
│   ├── extract/                    # AST → []Symbol, []Edge (unresolved) — pure functions, easy to unit test
│   ├── resolve/                    # cross-file linking pass: unresolved refs → resolved Edge.TargetSymbolID
│   ├── graph/                      # STORAGE LAYER
│   │   ├── store.go                 # GraphStore interface (the port)
│   │   ├── sqlite/                  # v1 adapter: schema, migrations, WAL config, recursive-CTE queries
│   │   ├── registry.go              # config-driven factory: NewGraphStore(cfg) — same pattern as v2 remote adapter later
│   │   └── migrate/                 # old-TS-schema → new-schema converter, built on the same GraphStore port
│   ├── indexer/                     # walker + two-pass orchestration + content-hash dirty tracking
│   ├── watcher/                     # fsnotify wrapper, debounce, dirty-file propagation, owns the writer role while running
│   ├── query/                       # explore / call-paths / blast-radius / search — talks only to graph.GraphStore
│   ├── mcp/                         # stdio MCP server, tool handlers wrapping query/
│   ├── install/                     # per-agent config-writer adapters (registry pattern, one file per agent)
│   └── cli/                         # cobra command implementations, thin — delegate to indexer/query/install
├── pkg/                             # (only if you want a stable public Go API surface — optional for v1)
└── .codegraph/                      # PER-PROJECT runtime directory (not part of the repo above; created by `codegraph init`)
```

### .codegraph/ Directory Layout (redesigned vs TS CodeGraph)

TS CodeGraph puts everything flat: `codegraph.db` + `-wal`/`-shm` directly in `.codegraph/`, with no version marker and no separation between "local cache" and "the graph itself." That's fine for a single-file SQLite-only design, but it doesn't leave room for milestone-2 (CI-built shared indexes, a remote server pointer, multiple cache types). Recommended layout:

```
.codegraph/
├── VERSION                 # schema version int, read by `codegraph upgrade` and the migration tool
├── config.yaml              # language list, ignore patterns, storage backend selection
├── index.db                 # v1: local SQLite graph (WAL mode) — the source of truth
├── index.db-wal
├── index.db-shm
├── cache/
│   └── filehash.db          # SHA-256 content-hash cache for incremental skip-logic (separate from the graph itself
│                             #  so it can be blown away/rebuilt without touching the graph)
├── writer.lock               # advisory lock file — see Concurrency Model below
└── logs/
    └── watcher.log
```

For milestone 2, `config.yaml` gains a `storage.backend: sqlite|remote` field. When `remote`, `index.db` is absent and a `remote.json` (server URL, project ID, auth token path) replaces it. **No other component needs to know this happened** — `graph.NewGraphStore(cfg)` returns a different adapter, and `internal/query`, `internal/mcp`, `internal/cli` are unchanged. This is the concrete payoff of the storage-port boundary.

### Structure Rationale

- **`internal/types` has zero dependencies.** Every other package (parser, extract, graph, query) depends on it, nothing depends back. This is what makes the two-pass indexer, the storage swap, and unit testing all possible without import cycles.
- **`internal/graph` is the only package that imports a SQL driver or knows the word "SQLite."** `internal/query`, `internal/mcp`, `internal/cli` call `graph.GraphStore` methods only. This is the single boundary the milestone-2 constraint is asking you to protect.
- **`internal/parser` isolates the CGo-vs-WASM-vs-native decision per language.** Because Go/Java/C#/Python/TS all get separate sub-packages behind one `Parser` interface, you can ship Go with a pure-Go WASM backend and revisit CGo for a specific language later without an architecture change — the parser-strategy research question (noted as open in PROJECT.md) is fully decoupled from everything else.
- **`internal/watcher` and `internal/indexer` are separate** even though the watcher's only job is to call the indexer, because the watcher owns process-lifetime concerns (debounce timers, the writer lock, OS-level fsnotify quirks) that have nothing to do with the indexing algorithm itself, and you'll want to unit-test the two-pass indexer without spinning up fsnotify.

## Architectural Patterns

### Pattern 1: Two-Pass Indexing (local extract, then global resolve)

**What:** Every symbol-graph indexer that supports cross-file references (not just single-file outlines) splits indexing into two passes. Pass 1 walks each file independently (fully parallelizable — no file needs to see another file) and emits symbols plus *unresolved* references (`calls("someFunc")`, `imports("pkg/foo")`). Pass 2 runs after all files are parsed, when a complete symbol table exists, and turns those unresolved references into resolved edges (`Edge{From: sym123, To: sym456, Kind: CALLS}`).

**When to use:** Any time symbols in one file can reference symbols in another (which is the entire point of a *cross-file* code graph — this is explicitly a CodeGraph requirement). Single-file-only tools (ctags-style) can skip pass 2.

**Trade-offs:** Pass 1 parallelizes trivially (goroutine pool). Pass 2 is inherently more sequential/coordinated because it needs global state, but it's cheap relative to parsing — it's dictionary lookups, not AST walks. The real cost this pattern imposes is on *incremental* updates: changing one file can only ever need pass-1 re-run on that file, but may require pass-2 re-run on every file that referenced symbols the changed file redefined or removed (see Pitfall below, and the Dirty-file tracker component).

**Example:**
```go
// Pass 1 — parallel, per-file, no cross-file knowledge needed
type LocalExtraction struct {
    Symbols     []types.Symbol
    UnresolvedRefs []types.UnresolvedRef // {FromSymbol, TargetName, Kind}
}

func ExtractFile(ctx context.Context, path string, ast *parser.AST) (LocalExtraction, error)

// Pass 2 — after all files are parsed, symbol table is complete
func Resolve(symbolTable map[string]types.SymbolID, refs []types.UnresolvedRef) []types.Edge
```

### Pattern 2: Storage Port / GraphStore Interface (ports & adapters)

**What:** Define a Go interface (`GraphStore`) covering every operation the rest of the system needs from the graph — writes (`PutSymbols`, `PutEdges`, `DeleteFile`) and reads (`GetSymbol`, `Callers`, `Callees`, `BlastRadius`, `Search`). Every other package — extractor, indexer, query engine, MCP server, CLI — depends on this interface, never on `database/sql`, a SQLite driver, or SQL strings directly.

**When to use:** This is *the* pattern the milestone context calls for explicitly ("storage and process architecture must anticipate milestone-2 team features... without a rewrite"). It's how real projects with a documented local→server migration path (see the `shaktiman` ADR-003 "pluggable storage backends" pattern found in research: registry + config-driven factory, capability-validated backend combinations) avoid a rewrite. The factory (`graph.NewGraphStore(cfg)`) is the one place that knows how to build a concrete adapter from config; everything else takes the interface as a constructor argument.

**Trade-offs:** Slight upfront cost (defining the interface before you have a second implementation feels premature) but the alternative — SQL embedded in query/MCP/CLI code — is the single most common reason these projects need a rewrite for server support. Keep the interface narrow (don't leak SQLite-specific concepts like "connection," "transaction," or "pragma" into it — those stay inside the adapter).

**Example:**
```go
// internal/graph/store.go — the port. No package here imports database/sql.
type GraphStore interface {
    PutSymbols(ctx context.Context, syms []types.Symbol) error
    PutEdges(ctx context.Context, edges []types.Edge) error
    DeleteFile(ctx context.Context, path string) error // cascades symbols+edges for that file

    GetSymbol(ctx context.Context, id types.SymbolID) (types.Symbol, error)
    Callers(ctx context.Context, id types.SymbolID) ([]types.Symbol, error)
    Callees(ctx context.Context, id types.SymbolID) ([]types.Symbol, error)
    BlastRadius(ctx context.Context, id types.SymbolID, depth int) (types.Subgraph, error)
    Search(ctx context.Context, query string, kind types.SymbolKind) ([]types.Symbol, error)
}

// internal/graph/registry.go — config-driven factory, same shape v2's remote adapter will use
func NewGraphStore(cfg Config) (GraphStore, func() error /*close*/, error) {
    switch cfg.Backend {
    case "sqlite":
        return sqlite.Open(cfg.SQLitePath)
    case "remote": // milestone 2 — same interface, different wire
        return remote.Dial(cfg.ServerURL, cfg.AuthToken)
    default:
        return nil, nil, fmt.Errorf("unknown storage backend %q", cfg.Backend)
    }
}
```

### Pattern 3: Single-Writer / Multi-Reader Concurrency, with the Writer Role as a Process Concern (not a storage concern)

**What:** SQLite in WAL mode gives you *many concurrent readers* and *exactly one writer at a time*, enforced at the file level via a shared-memory (`-shm`) wal-index — which is same-host-only by construction (confirmed directly in SQLite's own WAL documentation). Every project surveyed that uses SQLite this way converges on the same shape: one designated writer connection (`BEGIN IMMEDIATE`, `busy_timeout` set, retry with exponential backoff+jitter on `SQLITE_BUSY`), and any number of separate read-only connections/processes.

For CodeGraph specifically, there are *three* processes that might want to write: the CLI (`codegraph index`), the watcher (auto-sync), and — if it's ever made a write path — the MCP server. **Decide the writer role at the process-coordination layer, not by fighting SQLite's model.** Concretely:
- The **watcher** (whether it runs standalone via `codegraph watch` or embedded inside the running MCP server process) is the canonical writer whenever it's alive. It holds `.codegraph/writer.lock` (an `flock`-style advisory lock) for its lifetime.
- **One-shot CLI writes** (`codegraph index`, `codegraph migrate`) try to acquire the same lock; if the watcher already holds it, they either queue a request to the live watcher (best) or fail fast with a clear "watcher is running, changes will be picked up automatically" message (acceptable for v1) rather than racing SQLite's own lock and surfacing a raw `database is locked` error.
- **Reads** (CLI `explore`, MCP tool calls) never need the writer lock — they open the SQLite file read-only (or WAL-mode default) and get SQLite's snapshot-isolation guarantees for free.

**Why this matters for milestone 2:** if "writer" is a role coordinated by a lock file and a config-driven factory rather than something baked into "the CLI process," it transfers cleanly to a world where the writer is a CI job pushing into a central server and every local process (CLI, MCP server) is *only* ever a reader. You are not rearchitecting concurrency for team mode — you are pointing the `GraphStore` factory at a remote adapter and deleting the local writer-lock logic, which was already isolated in one place (`internal/watcher` + `internal/graph/sqlite`).

**Trade-offs:** A lock file adds one more thing that can go stale (crashed process holding a lock) — mitigate with PID-in-lockfile + liveness check, a well-known pattern (same shape as `.git/index.lock`).

### Pattern 4: Content-Hash + Dependency-Aware Incremental Re-Indexing

**What:** On every watcher tick (post-debounce), for each changed file: (1) compute SHA-256, compare to the cached hash — if unchanged (touch/rebuild noise), skip entirely; (2) if changed, delete that file's existing symbols/edges (cascade) and re-run pass 1 on it; (3) **critically**, also identify every symbol that this file used to export/define and re-run pass 2 (resolve) for any file that referenced those symbols, even though *those* files' own content hash didn't change. This second step is what every naive "SHA-256 skip" implementation surveyed gets wrong initially (see Pitfalls) — Tessera's and gleann's public write-ups are explicit that the resolve pass, not just the extract pass, must be re-triggered for dependents.

**When to use:** Always, once cross-file resolution exists — this is not optional for a "code knowledge graph" as opposed to a per-file outline tool.

**Trade-offs:** Requires the graph to answer "who references symbols defined in file X" cheaply (a reverse index on `Edge.TargetSymbolID` grouped by originating file) — build this as a first-class query, not an afterthought, because the dirty-tracker depends on it on every incremental cycle.

## Data Flow

### Indexing Flow (write path)

```
[fsnotify event] or [`codegraph index` CLI invocation]
    ↓
[Watcher: debounce window] (100ms–60s, configurable — mirrors TS CodeGraph's CODEGRAPH_WATCH_DEBOUNCE_MS)
    ↓
[Dirty-file tracker]: hash each touched file → unchanged? skip. changed? mark dirty.
    ↓                                          also: mark dependents of dirty files' exported symbols as "needs re-resolve"
[Indexer Pass 1 — parallel worker pool]:
    file walker → language dispatch → Parser.Parse(content) → *AST
                                            ↓ (single shared AST)
                    extract.Symbols(ast), extract.UnresolvedRefs(ast)
    ↓
[GraphStore.DeleteFile(path)] then [GraphStore.PutSymbols(...)]   ← acquires writer lock
    ↓
[Indexer Pass 2 — resolve, sequential-ish]:
    build/update symbol table → resolve.Resolve(unresolvedRefs, symbolTable) → []Edge
    ↓
[GraphStore.PutEdges(...)]   ← same writer lock, same transaction where possible
    ↓
[index.db updated, WAL checkpointed on schedule]
```

### Query Flow (read path — CLI `explore` or MCP tool call)

```
[Agent tool call: codegraph_explore("someFunc")]
    ↓
[MCP server / CLI] → [Query engine]
    ↓
[GraphStore.Search / GetSymbol / Callers / Callees / BlastRadius]  (read-only connection, no lock needed)
    ↓
[SQLite adapter: recursive CTE for bounded-depth traversal]
    ↓
[Query engine assembles response: verbatim source (re-read from disk by line range) + call paths + blast-radius summary]
    ↓
[Response to agent — one round trip, per TS CodeGraph's design goal]
```

### Key Data Flows

1. **File → Symbol/Edge → Graph:** one-directional, extraction never reads the graph (pass 1 is graph-agnostic; only pass 2 reads the symbol table).
2. **Graph → Query response:** one-directional, read-only; query engine never mutates the store.
3. **Staleness signaling:** while the debounce window is open or pass 2 hasn't finished for a dirty file, query responses for affected symbols should carry a "stale" flag (TS CodeGraph does this) rather than silently returning outdated data — this requires the dirty-tracker's in-flight state to be visible to the query engine, e.g. via a small in-memory "currently indexing" set the watcher updates.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single small repo (< 5k files) | Full reindex on `init` is fine; SQLite default config; no in-memory snapshot needed |
| Larger repo / monorepo (5k–100k files) | Content-hash incremental is mandatory (not full reindex per save); batch writes into large transactions (don't commit per-file); consider periodic `PRAGMA wal_checkpoint` tuning since long-running MCP-server readers can otherwise starve checkpoints (documented SQLite WAL failure mode); consider an in-memory adjacency-list snapshot rebuilt after each index cycle for the hot query path (pattern seen in Tessera's mmap snapshot) — optional, only if recursive-CTE latency becomes a measured problem, not upfront |
| Team scale (milestone 2: shared/CI-built indexes, concurrent multi-agent access across machines) | This is exactly where local SQLite's WAL model structurally stops working — the `-shm` wal-index is same-host-only by SQLite's own design, so a shared team index cannot be "the same SQLite file on a network drive." The `GraphStore` port pattern above is what makes this a config change (swap adapter to a real client/server database with proper cross-host concurrency, e.g. Postgres) rather than a rewrite. CI-built indexes become an upload-to-server step; local agents become pure `GraphStore` readers against the remote adapter |

### Scaling Priorities

1. **First bottleneck (any repo size):** full reindex on every save. Fixed by content-hash + dependency-aware dirty tracking (Pattern 4) — this alone is table stakes, not an optimization.
2. **Second bottleneck (monorepo scale):** SQLite write-transaction granularity. Fixed by batching writes per debounce cycle into one transaction instead of one-per-file, and by keeping the writer role single and lock-coordinated (Pattern 3) so contention never compounds across processes.
3. **Third bottleneck (team scale, explicitly out of v1 scope):** same-host-only SQLite WAL concurrency. Not fixed within SQLite — this is the reason the storage port (Pattern 2) exists at all. Do not attempt to solve this with SQLite tricks (network-mounted DB files, custom VFS) — the research is unambiguous that this is unsafe (SQLite's own docs, and independent write-ups, agree a live network-mounted WAL file corrupts).

## Anti-Patterns

### Anti-Pattern 1: SQL (or any storage detail) leaking outside `internal/graph`

**What people do:** Write `db.Query("SELECT ... FROM symbols WHERE ...")` directly inside the MCP tool handler or the CLI command because "it's just faster for now."
**Why it's wrong:** This is precisely what forces a rewrite when milestone 2 arrives — every call site that assumed "the graph is a local SQLite file" has to be found and changed. It also makes the migration tool and query engine impossible to unit test without a real SQLite file.
**Instead:** Every non-storage package takes a `graph.GraphStore` as a constructor argument. Only `internal/graph/sqlite` (and later `internal/graph/remote`) import a driver.

### Anti-Pattern 2: Parser implementation baked directly into the extractor

**What people do:** Call `sitter.NewParser()` / a specific CGo binding directly inside symbol-extraction code, once per language, copy-pasted.
**Why it's wrong:** The project's own open research question is CGo vs WASM(wazero) vs native-Go per language — a real, measured trade-off (research found ~2x slower than CGo but zero-CGO cross-compilation and crash-isolation for the WASM route; native-Go reimplementations can have real correctness gaps on adversarial input). If the parser call is inline in every language's extractor, you cannot make that call independently per language or change it later without touching extraction logic everywhere.
**Instead:** `Parser` interface in `internal/parser`, one implementation per language, chosen via the same registry/factory pattern as storage. This also directly serves the "single static binary" constraint — the build can decide per-language whether CGo is acceptable for that target.

### Anti-Pattern 3: Full reindex as the only incremental strategy ("just re-run init")

**What people do:** On file change, blow away and rebuild the whole graph because it's simpler than tracking dirty state.
**Why it's wrong:** Explicitly called out as unacceptable by the project's own "auto-sync... with no manual re-runs" requirement, and it's the first thing that breaks at monorepo scale (Java/C# work-team codebase, per PROJECT.md context).
**Instead:** Pattern 4 (content-hash + dependency-aware dirty propagation), with `--full` as an explicit escape hatch (every surveyed project keeps one), not the default path.

### Anti-Pattern 4: Multiple uncoordinated writers to the same SQLite file

**What people do:** Let the watcher, a manually-run `codegraph index`, and the MCP server all open the DB for writes independently, relying on SQLite's `busy_timeout` to smooth over collisions.
**Why it's wrong:** `busy_timeout` has a ceiling; under real contention (a monorepo watcher mid-transaction plus a user manually running `codegraph index`) you get `SQLITE_BUSY` surfaced to a user or agent as an opaque error, and worse, no single process ever "owns" a consistent view of what's currently being written.
**Instead:** Pattern 3 — a single writer-lock file coordinates which process is allowed to write at any moment, independent of SQLite's own locking, so failures are explicit ("watcher is already running") instead of racy.

### Anti-Pattern 5: Growing installer if/else chain instead of a per-agent adapter registry

**What people do:** One `install.go` with a giant switch over agent names, each branch hand-writing config file paths inline.
**Why it's wrong:** The parity requirement lists 8 agents today (Claude Code, Cursor, Codex, OpenCode, Hermes, Gemini, Antigravity, Kiro) and TS CodeGraph's own roster will keep growing — a switch statement becomes an increasingly risky file to touch for any single-agent change.
**Instead:** One small file per agent implementing a common `AgentInstaller` interface (`Install(projectPath) error`, `Uninstall(projectPath) error`, `IsInstalled(projectPath) bool`), registered in a table. Mirrors the same registry pattern already used for parsers and storage backends — one idiom, three uses.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| MCP-speaking agents (Claude Code, Cursor, etc.) | stdio JSON-RPC, long-lived process per session | The MCP server process is the natural home for the watcher/writer role while a session is active — avoid spinning up a *second* independent daemon that also wants write access |
| Original TS CodeGraph installs | One-way migration, read-only access to their `codegraph.db` | Migration tool should not assume it can write back to the old format; open old DB `mode=ro` |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| extract ↔ resolve | In-process Go function calls, `[]UnresolvedRef` / `map[string]SymbolID` | Pass 1 output feeds pass 2 input; no I/O in between for a single indexing run |
| indexer ↔ graph | `GraphStore` interface calls only | The boundary that must survive the v1→v2 storage swap untouched |
| watcher ↔ indexer | Direct function calls (watcher triggers indexer on debounced paths) | Watcher owns timing/locking; indexer owns algorithm — keep them separable for unit testing |
| query ↔ graph | `GraphStore` interface calls only (read methods) | Same interface as indexer's write side — one port, both directions |
| mcp / cli ↔ query | Direct function calls, no network hop (same process) | MCP and CLI are both just callers of the query engine, differing only in transport (stdio JSON-RPC vs terminal output) |

## Build Order Implications

The dependency chain below is the natural phase order — each phase's output is a hard prerequisite for the next, except where marked parallelizable:

1. **Core domain types + `GraphStore` interface** (`internal/types`, `internal/graph/store.go`) — foundation; nothing else compiles meaningfully without it.
2. **SQLite adapter** (`internal/graph/sqlite`) — schema (symbols, edges, files, FTS5), WAL pragmas, writer-lock coordination. Needed before anything can persist data, so it should land before real parsing work, even with a stub/synthetic dataset for testing.
3. **Parser interface + first language (Go)** (`internal/parser`, `internal/parser/golang`) — per PROJECT.md's stated priority order; proves the `Parser` interface shape before adding more languages.
4. **Extractor for Go** (`internal/extract`) — symbols + unresolved refs from a Go AST.
5. **Two-pass indexer** (`internal/indexer`) — wires walker + parser + extractor + resolve + `GraphStore`, single-shot (`codegraph index`, no watch yet). *Depends on 1–4.*
6. **Query engine** (`internal/query`) — call paths, blast radius, search, explore — built and tested against data produced by step 5. *Depends on 2 (interface) and benefits from 5 (real data to query), but the interface contract from step 1 means query code can start against a mocked `GraphStore` in parallel with step 5.*
7. **CLI** (`internal/cli`, `cmd/codegraph`) — `init`, `index`, `explore` wrap steps 5+6. *Depends on 5, 6.*
8. **MCP server** (`internal/mcp`) — wraps step 6's query engine with the stdio tool surface. *Depends on 6; parallelizable with 7 once 6 is stable — CLI and MCP are two independent consumers of the same query engine.*
9. **Watcher + incremental/dirty-file re-index** (`internal/watcher`, content-hash cache, dependency-aware invalidation) — extends step 5 from one-shot to continuous. *Depends on 5 already supporting single-file re-index (delete-then-reinsert per file).*
10. **Additional languages** (Java/C#, Python, TypeScript/JavaScript, remainder per PROJECT.md order) — *depends on 3's `Parser` interface being proven correct on Go; otherwise fully parallelizable across languages once the interface is stable.*
11. **Migration tool** (`internal/graph/migrate`) — TS SQLite schema → new schema. *Depends on 2 (new schema finalized) and requires separately researching the exact TS CodeGraph schema (flagged as its own research task — this document only confirms it's SQLite+FTS5, not the literal column layout).*
12. **Agent installer/uninstaller** (`internal/install`) — mostly independent; can start as soon as the CLI skeleton (step 7) exists to hang `codegraph install <agent>` off of.
13. **Release engineering** (signing, SLSA, SBOM, reproducible builds) — orthogonal to the above; can be scaffolded early (CI pipeline) but only meaningfully validated once steps 1–9 produce a real binary to sign.

**Research flags for later phases:**
- Phase covering the **exact TS CodeGraph SQLite schema** (for the migration tool) needs its own dedicated research pass — this document did not get primary-source access to the literal DDL, only confirmed the general shape (SQLite + FTS5, WAL mode, symbols/edges/files).
- Phase covering **parser strategy per language** (CGo vs WASM/wazero vs native-Go) should re-run the quantified benchmark research per language as it's added, since the trade-off (measured elsewhere as ~2x slower for WASM vs CGo, with correctness gaps reported for at least one native-Go reimplementation on adversarial input) may not hold identically for every grammar.

## Sources

- [SQLite WAL documentation](https://sqlite.org/wal.html) — HIGH confidence, primary source, same-host-only wal-index, single-writer model
- [Bugsink: Single-writer Database Architecture with SQLite](https://www.bugsink.com/blog/database-transactions/) — MEDIUM-HIGH, production write-up, BEGIN IMMEDIATE pattern
- [ChatML: SQLite Concurrency in Go — Desktop AI IDE](https://chatml.com/blog/sqlite-concurrency-in-go-desktop-ai-ide) — MEDIUM, real Go multi-writer war story (retry+backoff+jitter pattern), closely analogous use case (multiple agent processes writing one local SQLite file)
- Tessera `docs/architecture.md` (github.com/iamsaquib8/tessera) — MEDIUM, tree-sitter→SQLite(WAL)→mmap-snapshot pipeline, incremental indexing algorithm
- gleann indexer commit (github.com/tevfik/gleann) — MEDIUM, single-parse-per-file + content-hash incremental cache pattern
- code-review-graph-go (github.com/harshsh-dev/code-review-graph-go) — MEDIUM, Go/CGo/tree-sitter/SQLite-WAL project structure and worker-pool pattern, closest direct analog to this project
- code-context (github.com/sjzsdu/code-context) — MEDIUM, pure-Go SQLite (modernc.org/sqlite), single-binary Go code-graph MCP tool, project layout
- graphindex (pkg.go.dev/github.com/254binaryninja/graphindex) — MEDIUM, minimal Go MCP+SQLite+fsnotify code-graph server, closest architecture diagram analog
- arbor `docs/ARCHITECTURE.md` (github.com/Anandb71/arbor) — MEDIUM, Rust but directly documents debounced watcher + delta engine + surgical vs full re-parse decision, same pattern applies
- shaktiman ADR-003 "pluggable storage backends" (github.com/midhunkrishna/shaktiman) — MEDIUM, directly documents the registry/factory pattern for swapping SQLite→Postgres and the backend-combination-validation approach recommended above
- SCIP Code Intelligence Protocol docs (scip-code.org, github.com/scip-code/scip) — HIGH, primary protocol source; informs the stable-symbol-ID design note and cross-repo symbol addressing scheme for future milestone-2 work
- Sourcegraph "Cross-repository code navigation" (sourcegraph.com/blog) — MEDIUM, context for how SCIP-based systems scale symbol resolution across repo boundaries, relevant to milestone-2 anticipation
- Tree-sitter Go binding trade-off research: dvcdsys/code-index PR #81 (WASM/wazero vs cgo benchmark), tree-sitter/go-tree-sitter issue #16, malivvan/tree-sitter, wazero.io, dev.to "Parsing 11 languages in pure Go without CGO" — MEDIUM (mix of PoC benchmarks and blog write-ups), informs the Parser-interface-isolation anti-pattern rationale
- colbymchenry/codegraph GitHub repository (WebFetch summary) — MEDIUM (single indirect fetch, not primary docs) — confirms `.codegraph/codegraph.db` SQLite+FTS5+WAL layout, tree-sitter-based 30+ language parsing, and the three-layer native-watcher/debounce/staleness-signal auto-sync design that this document's Pattern 4 and "Key Data Flows" staleness note are built on

---
*Architecture research for: local-first code knowledge-graph indexer (Go)*
*Researched: 2026-07-10*
