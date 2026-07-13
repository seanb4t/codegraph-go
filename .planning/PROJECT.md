# CodeGraph Go

## What This Is

A ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) — the pre-indexed code knowledge graph for coding agents (Claude Code, Cursor, Codex, Gemini, etc.). One static Go binary replaces the TypeScript version's bundled-Node distribution, delivering the same agent-facing experience (CLI, MCP server, auto-sync) with better performance, a verifiable supply chain, and an architecture designed to grow into team-scale usage.

## Core Value

An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

## Requirements

### Validated

- Schema-versioned Pebble `GraphStore` substrate behind a concurrency-tested interface + benchmarked CGo tree-sitter parser decision — **validated in Phase 1**
- Go indexing pipeline (`init`/`index`/`uninit`, deterministic two-pass parallel-extract → sequential-resolve cross-file resolution) — **validated in Phase 2**
- Read-only query engine + parity stdio MCP server: `query`/`node`/`search`/`callers`/`callees`/`impact`/`affected`/`files`/`status`/`explore` plus `serve --mcp` with `codegraph_explore`-default tool gating (`CODEGRAPH_MCP_TOOLS` allowlist, zero tools without `.codegraph/`); output shapes verified against the TS v1.3.1 golden corpus — **validated in Phase 3**
- Incremental sync + native file watcher: `codegraph sync` (stat→content-hash diff + dependent-file recomputation), correct rename/delete/move pruning via an additive `x/` file-owned secondary index (no orphaned nodes / dangling edges, incl. cross-file `contains` edges), fsnotify debounced watcher + agent-facing staleness banner, MCP-reconnect stat+hash reconcile, shared `codegraph daemon` (single-writer lockfile) + `codegraph unlock`, and a goroutine-leak-free goleak soak; sync-equals-reindex determinism preserved — **validated in Phase 4** (a deep code review caught and fixed 4 concurrency/prune bugs the green suite had missed)
- Multi-language coverage & resolution breadth: the Go-only extractor generalized to a `LanguageSpec` registry + per-language extractor packages + extension→language discovery with per-language `ModuleKey` hooks (Go behavior byte-identical throughout). 14 registered languages — priority-4 (Java/C#/Python/TS-JS) at full extraction + cross-file resolution validated on real repos (Java: temporal `sdk-java` 1223 files; C#: serilog; Python: litellm/types; TS-JS: ccstatusline 13k files); mainstream-6 (Rust/Ruby/PHP/C/C++/Swift/Kotlin) at full-or-documented-partial per the committed D-11 capability matrix. Interface→implementation dispatch synthesized as `implements` edges (Go structural method-set match, Java/C# declared) traversed at query time; every synthesized edge carries `provenance: heuristic` + source location (no SchemaVersion bump — `Edge` fields were pre-reserved). Framework-aware routing (`route` nodes via heuristic `calls` edges) for Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS. The three deferred Go resolution items (WR-01, WR-02) fixed; the call-as-argument gap confirmed already-correct. A latent `proto.Marshal` map-ordering determinism bug was found and fixed store-wide. Two `[SUS]` community grammars (Swift/Kotlin) admitted only through a human-gated supply-chain review — **validated in Phase 5**
- Agent integrations & CLI lifecycle: `codegraph install`/`uninstall` configure all **8** roster agents (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro) via a self-registering `AgentTarget` registry — surgical MCP-config writes (JSON / JSONC-via-`tailscale/hujson` / hand-rolled TOML / YAML) plus marker-fenced (`<!-- CODEGRAPH_START/END -->`) instruction injection for the 4 agents that take one, idempotent + byte-invariant install→uninstall round-trip, `os.Executable()` absolute-path binding, and per-agent quirks (Cursor `--path`, Antigravity no-`type`, Gemini root-level `GEMINI.md`, Codex global-only, opencode comment-preserving JSONC). `codegraph version`/`--version`/`version --json` (ldflags-injected), `codegraph telemetry` (honest zero-passive-phone-home, names `upgrade` as the sole network path), and `codegraph upgrade [version] [--check]` (embedded `sigstore-go` keyless verify-**before**-swap, fail-closed, atomic self-replace). A deep code review caught + fixed 3 Criticals the green TDD suite missed (swallowed install/uninstall I/O errors, Antigravity migration data-loss, Hermes CRLF idempotency) plus security warnings; the security audit then closed 22/22 STRIDE threats. New pure-Go deps `sigstore-go` + `tailscale/hujson` (no new CGo — tree-sitter stays the sole exception). The live MCP handshake was verified end-to-end (install → `codegraph_explore` advertised over stdio → uninstall restores config). Research corrected a stale assumption pre-execution: TS parity covers all 8 agents (not 5), only 4 take instruction files — **validated in Phase 6**
- Migration tool: `codegraph migrate [--from][--to][--force][--drop-dangling]` — a one-way, one-step, resumable, validated converter from a TS CodeGraph `.codegraph/` SQLite index to the new Pebble/protobuf format. Read path via a CGo-free `modernc.org/sqlite` v1.53.0 reader opened read-only (`mode=ro`+`query_only`, source never mutated), confined to `internal/migrate` by an import-graph archtest (no new CGo). Preserves TS node ids verbatim (faithful row-for-row conversion, not a lossy native re-derivation); carries `start_col`/`end_col`; drops FTS/vocab/unresolved_refs/sqlite-internals; builds the Phase-4 `x/` file-index during write (`Meta.has_file_index=true`). Resumable via an additive `PutMigration`/`GetMigration` `m/migration` cursor record + temp-dir→atomic-rename swap (with interrupted-swap self-heal). Fail-loud D-09 validation: de-dup-aware edge-count reconciliation (the new key collapses line/col), `file:`-exempt zero-dangling check (`--drop-dangling` explicit opt-in), schema-version-range guard, gating `Meta.Healthy`. Aged-`.codegraph/` fixtures reconstructed in-Go from the captured TS dump (no external `sqlite3`). A deep code review found 0 Criticals + 5 real Warnings the green TDD suite missed (stale `edge_count` after drop-dangling, un-`Close()`d trailing batch, overwrite-probe mutating the target dir, no interrupted-swap recovery, unescaped DSN path) — all fixed with regression tests. One new pure-Go dep (`modernc.org/sqlite`), no new CGo — **validated in Phase 7**

### Active

- [ ] Index a codebase into a knowledge graph (symbols, edges, files) with structural extraction and cross-file resolution at parity with TS CodeGraph v1.3.x
- [ ] MCP server exposing the same tool surface as TS CodeGraph (`codegraph_explore` and companions) so existing agent configs work unchanged
- [x] CLI with parity commands: `init`, `install`, `uninstall`, `uninit`, `upgrade`, `explore`, plus `migrate` — full command surface shipped (`install`/`uninstall`/`upgrade` Phase 6, `migrate` **shipped Phase 7**)
- [ ] Auto-sync: file watcher keeps the graph current on every change with no manual re-runs
- [x] Agent installer/uninstaller support for the same agent roster (Claude Code, Cursor, Codex, OpenCode, Hermes, Gemini, Antigravity, Kiro) — **shipped Phase 6**
- [ ] 12+ language support at parity, built in priority order: Go → Java/C# → Python → TypeScript/JavaScript → remainder
- [ ] New storage format designed for concurrent access, incremental updates, and monorepo scale — with a migration tool converting existing `.codegraph/` SQLite indexes (migration converter **shipped Phase 7**)
- [ ] Single static binary per platform (macOS/Linux/Windows), minimal audited dependency tree, no bundled runtime
- [ ] Release integrity: cosign-signed artifacts, SLSA build provenance, published SBOM, vuln scanning gating CI, reproducible builds
- [ ] Published benchmarks vs TS CodeGraph on real repos (indexing throughput, query latency, memory)
- [ ] v1 storage format is schema-versioned and anticipates future annotations (embedding vectors, community assignments) and bulk graph export for visualization — future features bolt on without a format break

### Out of Scope

- Central graph server (multi-user, remote queries, auth) — milestone 2; v1 architecture must anticipate it, not implement it
- CI-built shared index distribution/caching — milestone 2, same rule: design for it, don't build it yet
- Hosted platform features (getcodegraph.com-style PR analysis) — different product; not this project's goal
- Bundling any runtime with the binary — antithesis of the design; the static binary IS the distribution
- Feature additions beyond TS parity in v1 — parity plus performance is the bar; new capabilities wait for v2
- Embeddings/vector search, community detection, graph visualization UI — planned future milestones (post-v1); v1 schema must anticipate them (annotations, export), not implement them. Embeddings will be local-model-first; cloud-API embeddings are permanently out

## Context

**Source project:** colbymchenry/codegraph, TypeScript (~4.5 MB source), v1.3.1, ~59k stars, MIT license. Builds a SQLite knowledge graph in `.codegraph/` per project, exposes it via CLI and MCP server, auto-syncs via file watcher, supports ~12 languages with cross-file resolution (framework-aware routes, mixed iOS/React Native bridging). Distributes via `curl | sh` installer or npm, bundling its own Node.js runtime — the supply-chain surface this port eliminates.

**User environment:** Sean runs TS CodeGraph daily as an MCP server across his projects, so parity gaps will be felt immediately — a strength for validation. Work team uses Java/C# heavily (hence its priority position); open-source release is intended from day one, so docs, release engineering, and compatibility promises matter early.

**Repo state:** Phases 1–3 shipped. Phase 1 landed the substrate (Pebble-backed `GraphStore` behind a concurrency-tested interface, protobuf schema-versioned records, CGo tree-sitter parser seam, golden TS ground-truth corpus). Phase 2 shipped the working Go indexer: `codegraph init`/`index`/`uninit` build a correct, cross-file-resolved graph from scratch via a deterministic two-pass (parallel-extract → sequential-resolve) pipeline — validated end-to-end (self-indexing this repo: files=48 nodes=414 edges=660, byte-identical rebuild). Phase 3 opened the graph to agents: the full read-only query command suite plus a stdio MCP server (`serve --mcp`, `mark3labs/mcp-go`), reading through the frozen Phase-2 graph via an additively-extended `Reader` (node/file iterators; query-time reverse adjacency). Output-shape parity with TS v1.3.1 is proven by a live golden-corpus diff (7/7 tools against the pinned `weft` checkout); a deep code review's two Critical findings (default-`--limit` DoS; MCP `path` confinement) were fixed before completion.

**Known open question (RESOLVED in Phase 1):** Parser strategy was decided by a benchmarked spike — **Option A, CGo tree-sitter** (`tree-sitter/go-tree-sitter` + per-language grammars), the single documented CGo exception (DIST-05). See `PARSER-DECISION.md`. wazero WASM remains a monitored future option behind the narrow `parser.Parser` seam.

## Constraints

- **Tech stack**: Go (latest stable), single static binary per platform — the performance and supply-chain story both depend on it
- **Supply chain**: Minimal, audited dependencies; prefer pure Go (CGo only with explicit justification pending parser research); signed + attested + reproducible + SBOM'd releases
- **Compatibility**: Behavioral parity with TS CodeGraph v1.3.x agent-facing surface (MCP tools, CLI semantics); one-way migration from its SQLite format
- **Architecture**: v1 storage and process design must accommodate milestone-2 team features (central server, CI-distributed indexes, concurrent access) without a rewrite
- **Licensing**: Original is MIT — port with attribution

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Full parity in v1 (not core-first) | Drop-in swap is the success bar; partial parity can't validate it | — Pending |
| New index format + migration tool | Optimize schema for concurrency/perf/scale; converter keeps existing users | ✓ Converter shipped (Phase 7): faithful TS-id-preserving one-way migrate, resumable + fail-loud validated |
| Parity v1 → team features v2 | Ship replacement value first; architect so server/CI features bolt on | — Pending |
| Language priority: Go → Java/C# → Python → TS/JS | Matches Sean's daily usage and work-team stack | — Pending |
| Full supply-chain suite from first release | Signing, SLSA, SBOM, reproducibility are the differentiator, not an afterthought | — Pending |
| Parser strategy (CGo tree-sitter vs wazero WASM vs native Go) | Performance vs purity vs sandboxing — needs quantified research | ✓ Resolved (Phase 1): CGo tree-sitter, benchmarked; single documented CGo exception (PARSER-DECISION.md) |
| Plan for embeddings, communities, graph-viz UI as future milestones | Long-term product direction; v1 schema versioned + annotation-ready so they bolt on | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-12 after Phase 7 (Migration Tool) completion*
