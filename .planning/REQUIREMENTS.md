# Requirements — CodeGraph Go v1

Requirements are the contract for v1: full TS CodeGraph v1.3.x parity as a drop-in swap, plus performance proof and supply-chain integrity. Scoping decisions from questioning: interface-dispatch synthesizer in v1 (rest v1.x), priority + mainstream languages (long-tail v1.x), daemon in v1, zero telemetry.

## v1 Requirements

### Indexing Core

- [x] **INDX-01**: User can run `codegraph init` in any repo to create `.codegraph/` and build the full graph in one step; `uninit [--force]` removes it cleanly
- [x] **INDX-02**: User can run `codegraph index` for a deterministic from-scratch rebuild with `--force`, `--quiet`, `--verbose`
- [x] **INDX-03**: User can run `codegraph sync` to incrementally reparse only changed files, using content-hash diffing with dependent-file edge recomputation (not full re-index of a subset)
- [x] **INDX-04**: File renames, deletes, and moves correctly prune stale symbols and edges — no orphaned nodes or dangling edges after any file operation, verified by a fixture suite
- [x] **INDX-05**: Storage engine (new format) supports concurrent lock-free readers with a single coordinated writer, implemented behind a `GraphStore` interface no other package bypasses
- [x] **INDX-06**: User can index a 100k+ file monorepo within bounded memory; peak RSS is tracked as a first-class CI metric

### Graph Resolution

- [x] **RES-01**: Cross-file resolution resolves imports, call edges, and type inheritance across the repo via two-pass indexing (parallel extract → resolve)
- [x] **RES-02**: Interface→implementation dispatch edges are synthesized (Go implicit interfaces, Java/C# interface implementations) so callers/impact traverse dynamic dispatch
- [x] **RES-03**: Every heuristic edge carries a `provenance: heuristic` tag with source location, distinguishing it from ground-truth AST edges

### Query & Analysis

- [x] **QRY-01**: User can run `codegraph query <search>` for symbol search with `--kind`, `--limit`, `--json`
- [x] **QRY-02**: User can run `codegraph node <symbol|file>` for symbol detail (source + callers/callees) or line-numbered file read
- [x] **QRY-03**: User can run lightweight locations-only symbol search (`search`) without source bodies
- [x] **QRY-04**: User can run `codegraph callers <symbol>` / `codegraph callees <symbol>` for reverse/forward call-graph traversal
- [x] **QRY-05**: User can run `codegraph impact <symbol> --depth` for transitive blast-radius analysis
- [x] **QRY-06**: User can run `codegraph affected [files...]` to identify impacted test files via a dedicated test-coverage edge type
- [x] **QRY-07**: User can run `codegraph files` to browse indexed structure from the graph (not a filesystem scan) with format/filter/pattern/depth options
- [x] **QRY-08**: User can run `codegraph explore <query>` (natural-language or symbol/file) and get verbatim line-numbered source of relevant symbols grouped by file, call paths between them, and a blast-radius summary in one round trip
- [x] **QRY-09**: User can run `codegraph status --json` for index health, node/edge counts, last-sync time, and staleness

### MCP Server

- [x] **MCP-01**: Agent gets a stdio MCP server (`codegraph serve --mcp`) with `codegraph_explore` as the only default-visible tool
- [x] **MCP-02**: User can expose the 7 additional tools (`node`, `search`, `callers`, `callees`, `impact`, `files`, `status`) via the `CODEGRAPH_MCP_TOOLS` allowlist env var
- [x] **MCP-03**: MCP server exposes zero tools when no `.codegraph/` exists, so agents fall back to built-ins gracefully
- [x] **MCP-04**: MCP tool output shapes match TS CodeGraph v1.3.x, verified by a golden-output corpus captured from the live TS version before the MCP phase starts

### Auto-Sync & Daemon

- [x] **SYNC-01**: Graph auto-updates on file changes via native per-OS watchers (FSEvents/inotify/ReadDirectoryChangesW), debounced (default 2000ms, tunable via env), consolidating edit bursts into one sync
- [x] **SYNC-02**: Agent-facing output includes a staleness warning banner while a sync is pending/debouncing
- [x] **SYNC-03**: On MCP server (re)connect, the index reconciles filesystem state via stat comparison + content hashing to catch offline changes
- [x] **SYNC-04**: User can run `codegraph daemon` as a shared indexing/watch server reused across multiple agent sessions, with in-process fallback where unsupported
- [x] **SYNC-05**: User can run `codegraph unlock` to remove stale lock files after a crashed daemon
- [x] **SYNC-06**: Long-running watcher/daemon is goroutine-leak-free, verified by soak tests

### CLI & Lifecycle

- [x] **CLI-01**: User can run `codegraph help [command]` and `codegraph version` with standard CLI ergonomics
- [x] **CLI-02**: User can run `codegraph upgrade [version]` to self-update via signature-verified binary download-and-swap, with `--check` support
- [x] **CLI-03**: `codegraph telemetry` reports that this build contains no telemetry; the binary contains zero phone-home code

### Agent Integrations

- [x] **AGNT-01**: User can run `codegraph install` to detect and configure the 8-agent roster (Claude Code, Cursor, Codex CLI, opencode, Hermes Agent, Gemini CLI, Antigravity IDE, Kiro) — MCP config plus marker-fenced instruction injection, idempotent on re-run
- [x] **AGNT-02**: User can run `codegraph uninstall` to cleanly reverse everything `install` wrote, preserving user edits outside markers
- [x] **AGNT-03**: Per-agent quirks are handled (e.g., Cursor's injected `--path` arg for MCP subprocess cwd)

### Language Support

- [x] **LANG-01**: Go — full structural extraction + cross-file resolution (first language, validates the pipeline)
- [x] **LANG-02**: Java — full extraction + resolution
- [x] **LANG-03**: C# — full extraction + resolution
- [x] **LANG-04**: Python — full extraction + resolution
- [x] **LANG-05**: TypeScript/JavaScript — full extraction + resolution
- [x] **LANG-06**: Mainstream tier — Rust, Ruby, PHP, C/C++, Swift, Kotlin at full or documented-partial support
- [x] **LANG-07**: Framework-aware routing for priority-language frameworks (Gin, Spring, ASP.NET, Django/Flask/FastAPI, Express/NestJS) emitting `route` nodes linked to handlers

### Migration

- [x] **MIGR-01**: User can run a migration command converting an existing TS `.codegraph/` SQLite index to the new format in one step
- [x] **MIGR-02**: Migration is resumable, version-stamped, validated against real aged `.codegraph/` directories, and runs structural-invariant checks on the result

### Distribution & Supply Chain

- [x] **DIST-01**: User can download a single static binary per platform (macOS/Linux/Windows, amd64+arm64) with no bundled runtime and no install-time compilation
- [x] **DIST-02**: Every release artifact is cosign-signed (keyless) with SLSA build provenance, and users can verify both with documented commands _(PROVEN 2026-07-14 on real release v0.0.0-rc.3: all 6 CGo targets built on CI, 6 per-binary `.sigstore.json` + 6 SBOMs + SLSA `multiple.intoto.jsonl` published; downloaded darwin/arm64 binary runs + `cosign verify-blob` returns "Verified OK" against the production identity in verify.go, and correctly REJECTS a wrong identity. Two real bugs caught+fixed by the first live runs: missing linux-only go.sum hash for prometheus/procfs, and SLSA private-repository opt-in. slsa-verifier provenance check left to the user via docs/RELEASE.md — provenance is published + generator succeeded)_
- [x] **DIST-03**: Every release publishes an SBOM; `govulncheck` and dependency scanning gate CI
- [x] **DIST-04**: Builds are reproducible, verified by a double-build comparison gate in CI
- [x] **DIST-05**: Dependency tree stays minimal and audited; CGo (if the parser spike selects it) is the sole documented exception

### Performance

- [x] **PERF-01**: Published head-to-head benchmarks vs TS CodeGraph (indexing throughput, query latency, peak RSS, cold start) on real repos using comparable methodology (median-of-N, raw per-repo numbers) _(RATIFIED 2026-07-13 — median-of-3 runs vs installed TS 1.3.1 published in docs/BENCHMARKS.md; Go wins every metric 6.1×–59.7× throughput, ~2.3–2.8× query, 3.6×–5.5× RSS, ~6× cold start; 3 raw runs committed. Canonical CI-hardware re-run via bench.yml still recommended post-release)_
- [x] **PERF-02**: Performance regression gates run in CI against a benchmark corpus including a 100k+ file monorepo

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
| INDX-01 | Phase 2 | Complete |
| INDX-02 | Phase 2 | Complete |
| INDX-03 | Phase 4 | Complete |
| INDX-04 | Phase 4 | Complete |
| INDX-05 | Phase 1 | Complete |
| INDX-06 | Phase 8 | Complete |
| RES-01 | Phase 2 | Complete |
| RES-02 | Phase 5 | Complete |
| RES-03 | Phase 5 | Complete |
| QRY-01 | Phase 3 | Complete |
| QRY-02 | Phase 3 | Complete |
| QRY-03 | Phase 3 | Complete |
| QRY-04 | Phase 3 | Complete |
| QRY-05 | Phase 3 | Complete |
| QRY-06 | Phase 3 | Complete |
| QRY-07 | Phase 3 | Complete |
| QRY-08 | Phase 3 | Complete |
| QRY-09 | Phase 3 | Complete |
| MCP-01 | Phase 3 | Complete |
| MCP-02 | Phase 3 | Complete |
| MCP-03 | Phase 3 | Complete |
| MCP-04 | Phase 3 | Complete |
| SYNC-01 | Phase 4 | Complete |
| SYNC-02 | Phase 4 | Complete |
| SYNC-03 | Phase 4 | Complete |
| SYNC-04 | Phase 4 | Complete |
| SYNC-05 | Phase 4 | Complete |
| SYNC-06 | Phase 4 | Complete |
| CLI-01 | Phase 6 | Complete |
| CLI-02 | Phase 6 | Complete |
| CLI-03 | Phase 6 | Complete |
| AGNT-01 | Phase 6 | Complete |
| AGNT-02 | Phase 6 | Complete |
| AGNT-03 | Phase 6 | Complete |
| LANG-01 | Phase 2 | Complete |
| LANG-02 | Phase 5 | Complete |
| LANG-03 | Phase 5 | Complete |
| LANG-04 | Phase 5 | Complete |
| LANG-05 | Phase 5 | Complete |
| LANG-06 | Phase 5 | Complete |
| LANG-07 | Phase 5 | Complete |
| MIGR-01 | Phase 7 | Complete |
| MIGR-02 | Phase 7 | Complete |
| DIST-01 | Phase 8 | Complete |
| DIST-02 | Phase 8 | Complete |
| DIST-03 | Phase 8 | Complete |
| DIST-04 | Phase 8 | Complete |
| DIST-05 | Phase 8 | Complete |
| PERF-01 | Phase 8 | Complete |
| PERF-02 | Phase 8 | Complete |
| ARCH-01 | Phase 1 | Complete |
