# Requirements — CodeGraph Go v1

Requirements are the contract for v1: full TS CodeGraph v1.3.x parity as a drop-in swap, plus performance proof and supply-chain integrity. Scoping decisions from questioning: interface-dispatch synthesizer in v1 (rest v1.x), priority + mainstream languages (long-tail v1.x), daemon in v1, zero telemetry.

## v1 Requirements

### Indexing Core

- [ ] **INDX-01**: User can run `codegraph init` in any repo to create `.codegraph/` and build the full graph in one step; `uninit [--force]` removes it cleanly
- [ ] **INDX-02**: User can run `codegraph index` for a deterministic from-scratch rebuild with `--force`, `--quiet`, `--verbose`
- [ ] **INDX-03**: User can run `codegraph sync` to incrementally reparse only changed files, using content-hash diffing with dependent-file edge recomputation (not full re-index of a subset)
- [ ] **INDX-04**: File renames, deletes, and moves correctly prune stale symbols and edges — no orphaned nodes or dangling edges after any file operation, verified by a fixture suite
- [x] **INDX-05**: Storage engine (new format) supports concurrent lock-free readers with a single coordinated writer, implemented behind a `GraphStore` interface no other package bypasses
- [ ] **INDX-06**: User can index a 100k+ file monorepo within bounded memory; peak RSS is tracked as a first-class CI metric

### Graph Resolution

- [ ] **RES-01**: Cross-file resolution resolves imports, call edges, and type inheritance across the repo via two-pass indexing (parallel extract → resolve)
- [ ] **RES-02**: Interface→implementation dispatch edges are synthesized (Go implicit interfaces, Java/C# interface implementations) so callers/impact traverse dynamic dispatch
- [ ] **RES-03**: Every heuristic edge carries a `provenance: heuristic` tag with source location, distinguishing it from ground-truth AST edges

### Query & Analysis

- [ ] **QRY-01**: User can run `codegraph query <search>` for symbol search with `--kind`, `--limit`, `--json`
- [ ] **QRY-02**: User can run `codegraph node <symbol|file>` for symbol detail (source + callers/callees) or line-numbered file read
- [ ] **QRY-03**: User can run lightweight locations-only symbol search (`search`) without source bodies
- [ ] **QRY-04**: User can run `codegraph callers <symbol>` / `codegraph callees <symbol>` for reverse/forward call-graph traversal
- [ ] **QRY-05**: User can run `codegraph impact <symbol> --depth` for transitive blast-radius analysis
- [ ] **QRY-06**: User can run `codegraph affected [files...]` to identify impacted test files via a dedicated test-coverage edge type
- [ ] **QRY-07**: User can run `codegraph files` to browse indexed structure from the graph (not a filesystem scan) with format/filter/pattern/depth options
- [ ] **QRY-08**: User can run `codegraph explore <query>` (natural-language or symbol/file) and get verbatim line-numbered source of relevant symbols grouped by file, call paths between them, and a blast-radius summary in one round trip
- [ ] **QRY-09**: User can run `codegraph status --json` for index health, node/edge counts, last-sync time, and staleness

### MCP Server

- [ ] **MCP-01**: Agent gets a stdio MCP server (`codegraph serve --mcp`) with `codegraph_explore` as the only default-visible tool
- [ ] **MCP-02**: User can expose the 7 additional tools (`node`, `search`, `callers`, `callees`, `impact`, `files`, `status`) via the `CODEGRAPH_MCP_TOOLS` allowlist env var
- [ ] **MCP-03**: MCP server exposes zero tools when no `.codegraph/` exists, so agents fall back to built-ins gracefully
- [ ] **MCP-04**: MCP tool output shapes match TS CodeGraph v1.3.x, verified by a golden-output corpus captured from the live TS version before the MCP phase starts

### Auto-Sync & Daemon

- [ ] **SYNC-01**: Graph auto-updates on file changes via native per-OS watchers (FSEvents/inotify/ReadDirectoryChangesW), debounced (default 2000ms, tunable via env), consolidating edit bursts into one sync
- [ ] **SYNC-02**: Agent-facing output includes a staleness warning banner while a sync is pending/debouncing
- [ ] **SYNC-03**: On MCP server (re)connect, the index reconciles filesystem state via stat comparison + content hashing to catch offline changes
- [ ] **SYNC-04**: User can run `codegraph daemon` as a shared indexing/watch server reused across multiple agent sessions, with in-process fallback where unsupported
- [ ] **SYNC-05**: User can run `codegraph unlock` to remove stale lock files after a crashed daemon
- [ ] **SYNC-06**: Long-running watcher/daemon is goroutine-leak-free, verified by soak tests

### CLI & Lifecycle

- [ ] **CLI-01**: User can run `codegraph help [command]` and `codegraph version` with standard CLI ergonomics
- [ ] **CLI-02**: User can run `codegraph upgrade [version]` to self-update via signature-verified binary download-and-swap, with `--check` support
- [ ] **CLI-03**: `codegraph telemetry` reports that this build contains no telemetry; the binary contains zero phone-home code

### Agent Integrations

- [ ] **AGNT-01**: User can run `codegraph install` to detect and configure the 8-agent roster (Claude Code, Cursor, Codex CLI, opencode, Hermes Agent, Gemini CLI, Antigravity IDE, Kiro) — MCP config plus marker-fenced instruction injection, idempotent on re-run
- [ ] **AGNT-02**: User can run `codegraph uninstall` to cleanly reverse everything `install` wrote, preserving user edits outside markers
- [ ] **AGNT-03**: Per-agent quirks are handled (e.g., Cursor's injected `--path` arg for MCP subprocess cwd)

### Language Support

- [ ] **LANG-01**: Go — full structural extraction + cross-file resolution (first language, validates the pipeline)
- [ ] **LANG-02**: Java — full extraction + resolution
- [ ] **LANG-03**: C# — full extraction + resolution
- [ ] **LANG-04**: Python — full extraction + resolution
- [ ] **LANG-05**: TypeScript/JavaScript — full extraction + resolution
- [ ] **LANG-06**: Mainstream tier — Rust, Ruby, PHP, C/C++, Swift, Kotlin at full or documented-partial support
- [ ] **LANG-07**: Framework-aware routing for priority-language frameworks (Gin, Spring, ASP.NET, Django/Flask/FastAPI, Express/NestJS) emitting `route` nodes linked to handlers

### Migration

- [ ] **MIGR-01**: User can run a migration command converting an existing TS `.codegraph/` SQLite index to the new format in one step
- [ ] **MIGR-02**: Migration is resumable, version-stamped, validated against real aged `.codegraph/` directories, and runs structural-invariant checks on the result

### Distribution & Supply Chain

- [ ] **DIST-01**: User can download a single static binary per platform (macOS/Linux/Windows, amd64+arm64) with no bundled runtime and no install-time compilation
- [ ] **DIST-02**: Every release artifact is cosign-signed (keyless) with SLSA build provenance, and users can verify both with documented commands
- [ ] **DIST-03**: Every release publishes an SBOM; `govulncheck` and dependency scanning gate CI
- [ ] **DIST-04**: Builds are reproducible, verified by a double-build comparison gate in CI
- [ ] **DIST-05**: Dependency tree stays minimal and audited; CGo (if the parser spike selects it) is the sole documented exception

### Performance

- [ ] **PERF-01**: Published head-to-head benchmarks vs TS CodeGraph (indexing throughput, query latency, peak RSS, cold start) on real repos using comparable methodology (median-of-N, raw per-repo numbers)
- [ ] **PERF-02**: Performance regression gates run in CI against a benchmark corpus including a 100k+ file monorepo

### Architecture Future-Proofing

- [x] **ARCH-01**: The new storage format and `GraphStore` interface are schema-versioned from v1 and accommodate future node/edge annotations (embedding vectors, community/cluster assignments) and bulk graph export (for visualization) without a format break

## v2 Requirements (Deferred)

### v1.x — after v1 validation

- **SYNTH-DEFER**: Remaining synthesizers — Swift↔ObjC, React Native bridge/TurboModules/Fabric/Expo, EventEmitter/callback patterns, React state→re-render
- **LANG-DEFER**: Long-tail languages (COBOL, Solidity, Terraform/Nix, Pascal/Delphi, Lua, R, etc.) — added on real user demand
- **FRAME-DEFER**: Remaining framework routing (Rails, Laravel, Vapor, Rocket, etc.)

### v2 — next milestone

- **SERVER-01**: Central graph server (multi-user, remote queries, auth)
- **SERVER-02**: CI-built shared index distribution/caching
- **BEYOND-01**: Capabilities beyond TS parity (new query types, new MCP tools)

### v2+ — future milestones (v1 architecture must not preclude; see ARCH-01)

- **EMBED-01**: Embeddings / vector search over graph nodes for semantic queries complementing structural search — local-model-first so the "100% local, no API keys" trust story survives
- **COMM-01**: Community detection / clustering over the graph — module boundaries, god-node identification, architectural summaries
- **VIZ-01**: Graph visualization UI that stays performant regardless of graph size — level-of-detail rendering, community-based aggregation, progressive loading

## Out of Scope

- **Embeddings / vector search in v1** — deferred to a future milestone (EMBED-01), local-model-first to preserve the trust story; v1 stays pure structural. Cloud-API-backed embeddings remain permanently out
- **Hosted platform / PR-analysis SaaS** — different product; this project is the engine
- **Auto-editing / refactoring via the graph** — read-only exploration tool; editing stays with the host agent
- **Bundled runtime of any kind** — the static binary IS the distribution
- **Telemetry collection** — zero phone-home code (decided during scoping; stronger trust story than TS parity here)
- **Polling as default file watching** — native watchers per OS; polling only as explicit degraded fallback

## Traceability

Each v1 requirement maps to exactly one phase. Coverage: 51/51 mapped, no orphans, no duplicates.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INDX-01 | Phase 2 | Pending |
| INDX-02 | Phase 2 | Pending |
| INDX-03 | Phase 4 | Pending |
| INDX-04 | Phase 4 | Pending |
| INDX-05 | Phase 1 | Complete |
| INDX-06 | Phase 8 | Pending |
| RES-01 | Phase 2 | Pending |
| RES-02 | Phase 5 | Pending |
| RES-03 | Phase 5 | Pending |
| QRY-01 | Phase 3 | Pending |
| QRY-02 | Phase 3 | Pending |
| QRY-03 | Phase 3 | Pending |
| QRY-04 | Phase 3 | Pending |
| QRY-05 | Phase 3 | Pending |
| QRY-06 | Phase 3 | Pending |
| QRY-07 | Phase 3 | Pending |
| QRY-08 | Phase 3 | Pending |
| QRY-09 | Phase 3 | Pending |
| MCP-01 | Phase 3 | Pending |
| MCP-02 | Phase 3 | Pending |
| MCP-03 | Phase 3 | Pending |
| MCP-04 | Phase 3 | Pending |
| SYNC-01 | Phase 4 | Pending |
| SYNC-02 | Phase 4 | Pending |
| SYNC-03 | Phase 4 | Pending |
| SYNC-04 | Phase 4 | Pending |
| SYNC-05 | Phase 4 | Pending |
| SYNC-06 | Phase 4 | Pending |
| CLI-01 | Phase 6 | Pending |
| CLI-02 | Phase 6 | Pending |
| CLI-03 | Phase 6 | Pending |
| AGNT-01 | Phase 6 | Pending |
| AGNT-02 | Phase 6 | Pending |
| AGNT-03 | Phase 6 | Pending |
| LANG-01 | Phase 2 | Pending |
| LANG-02 | Phase 5 | Pending |
| LANG-03 | Phase 5 | Pending |
| LANG-04 | Phase 5 | Pending |
| LANG-05 | Phase 5 | Pending |
| LANG-06 | Phase 5 | Pending |
| LANG-07 | Phase 5 | Pending |
| MIGR-01 | Phase 7 | Pending |
| MIGR-02 | Phase 7 | Pending |
| DIST-01 | Phase 8 | Pending |
| DIST-02 | Phase 8 | Pending |
| DIST-03 | Phase 8 | Pending |
| DIST-04 | Phase 8 | Pending |
| DIST-05 | Phase 8 | Pending |
| PERF-01 | Phase 8 | Pending |
| PERF-02 | Phase 8 | Pending |
| ARCH-01 | Phase 1 | Complete |
