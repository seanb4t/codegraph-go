# CodeGraph Go

## What This Is

A ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) — the pre-indexed code knowledge graph for coding agents (Claude Code, Cursor, Codex, Gemini, etc.). One static Go binary replaces the TypeScript version's bundled-Node distribution, delivering the same agent-facing experience (CLI, MCP server, auto-sync) with better performance, a verifiable supply chain, and an architecture designed to grow into team-scale usage.

## Core Value

An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

## Current Milestone: v1.0 Drop-in Parity & Human UX

**Goal:** Close the behavioral and surface gaps between codegraph-go and TS CodeGraph v1.3.x so an existing user can swap binaries with zero change in experience, then cut the first real signed `v1.0.0` release — retiring the "not yet drop-in" caveat v0.1 shipped with.

**Target features:**
- **Agent-output behavioral parity** — `explore` semantic-relevance selection (+ `⚠️ no covering tests` warnings), `node` multi-definition disambiguation, multi-word `<query...>` arity, and richer `status` content (DB size, nodes-by-kind, files-by-language). v0.1's golden test proved template shape but never the selection/relevance algorithms.
- **Watcher-on-MCP by default** — `serve --mcp` runs the fsnotify watcher automatically with a `--no-watch` opt-out, matching TS's live auto-sync (our `install` already writes the byte-identical `serve --mcp` invocation; only the watch default differs).
- **Git/worktree awareness** — borrowed-index detection (compute the currently-inert `worktreeMismatch`: a `status` warning + a compact inline MCP-result notice) plus opt-in git sync hooks (post-commit/merge/checkout). Directly fixes the silent "worktree queries the main branch's graph" correctness gap.
- **Output hygiene** — silence Pebble WAL log noise on stderr; TTY-gate all styling.
- **Human-facing TUI (Charm/bubbletea)** — lipgloss-styled `status`/`files` (plain when piped), an interactive `daemon` picker, `install`/`uninstall` multi-select, and `init`/`index`/`sync` progress. The agent/MCP output path stays plain, stable, parseable text.
- **Surface reconciliation** — systematic per-command flag parity; decide the `search` stance; keep `serve`/`migrate` as documented (`serve` is not a divergence — TS has it too).
- **Behavioral parity test harness** — fixtures for ambiguous names, multi-word queries, and relevance, closing the v0.1 golden-test blind spot.
- **First signed `v1.0.0` release** — closes v0.1's still-pending DIST-02 (real `v*` tag) and PERF-01 (published numbers); audits the new Charm deps via govulncheck/SBOM.

## Requirements

### Validated

- Schema-versioned Pebble `GraphStore` substrate behind a concurrency-tested interface + benchmarked CGo tree-sitter parser decision — **validated in Phase 1**
- Go indexing pipeline (`init`/`index`/`uninit`, deterministic two-pass parallel-extract → sequential-resolve cross-file resolution) — **validated in Phase 2**
- Read-only query engine + parity stdio MCP server: `query`/`node`/`search`/`callers`/`callees`/`impact`/`affected`/`files`/`status`/`explore` plus `serve --mcp` with `codegraph_explore`-default tool gating (`CODEGRAPH_MCP_TOOLS` allowlist, zero tools without `.codegraph/`); output shapes verified against the TS v1.3.1 golden corpus — **validated in Phase 3**
- Incremental sync + native file watcher: `codegraph sync` (stat→content-hash diff + dependent-file recomputation), correct rename/delete/move pruning via an additive `x/` file-owned secondary index (no orphaned nodes / dangling edges, incl. cross-file `contains` edges), fsnotify debounced watcher + agent-facing staleness banner, MCP-reconnect stat+hash reconcile, shared `codegraph daemon` (single-writer lockfile) + `codegraph unlock`, and a goroutine-leak-free goleak soak; sync-equals-reindex determinism preserved — **validated in Phase 4** (a deep code review caught and fixed 4 concurrency/prune bugs the green suite had missed)
- Multi-language coverage & resolution breadth: the Go-only extractor generalized to a `LanguageSpec` registry + per-language extractor packages + extension→language discovery with per-language `ModuleKey` hooks (Go behavior byte-identical throughout). 14 registered languages — priority-4 (Java/C#/Python/TS-JS) at full extraction + cross-file resolution validated on real repos (Java: temporal `sdk-java` 1223 files; C#: serilog; Python: litellm/types; TS-JS: ccstatusline 13k files); mainstream-6 (Rust/Ruby/PHP/C/C++/Swift/Kotlin) at full-or-documented-partial per the committed D-11 capability matrix. Interface→implementation dispatch synthesized as `implements` edges (Go structural method-set match, Java/C# declared) traversed at query time; every synthesized edge carries `provenance: heuristic` + source location (no SchemaVersion bump — `Edge` fields were pre-reserved). Framework-aware routing (`route` nodes via heuristic `calls` edges) for Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS. The three deferred Go resolution items (WR-01, WR-02) fixed; the call-as-argument gap confirmed already-correct. A latent `proto.Marshal` map-ordering determinism bug was found and fixed store-wide. Two `[SUS]` community grammars (Swift/Kotlin) admitted only through a human-gated supply-chain review — **validated in Phase 5**
- Agent integrations & CLI lifecycle: `codegraph install`/`uninstall` configure all **8** roster agents (Claude Code, Cursor, Codex CLI, opencode, Gemini CLI, Hermes, Antigravity, Kiro) via a self-registering `AgentTarget` registry — surgical MCP-config writes (JSON / JSONC-via-`tailscale/hujson` / hand-rolled TOML / YAML) plus marker-fenced (`<!-- CODEGRAPH_START/END -->`) instruction injection for the 4 agents that take one, idempotent + byte-invariant install→uninstall round-trip, `os.Executable()` absolute-path binding, and per-agent quirks (Cursor `--path`, Antigravity no-`type`, Gemini root-level `GEMINI.md`, Codex global-only, opencode comment-preserving JSONC). `codegraph version`/`--version`/`version --json` (ldflags-injected), `codegraph telemetry` (honest zero-passive-phone-home, names `upgrade` as the sole network path), and `codegraph upgrade [version] [--check]` (embedded `sigstore-go` keyless verify-**before**-swap, fail-closed, atomic self-replace). A deep code review caught + fixed 3 Criticals the green TDD suite missed (swallowed install/uninstall I/O errors, Antigravity migration data-loss, Hermes CRLF idempotency) plus security warnings; the security audit then closed 22/22 STRIDE threats. New pure-Go deps `sigstore-go` + `tailscale/hujson` (no new CGo — tree-sitter stays the sole exception). The live MCP handshake was verified end-to-end (install → `codegraph_explore` advertised over stdio → uninstall restores config). Research corrected a stale assumption pre-execution: TS parity covers all 8 agents (not 5), only 4 take instruction files — **validated in Phase 6**
- Migration tool: `codegraph migrate [--from][--to][--force][--drop-dangling]` — a one-way, one-step, resumable, validated converter from a TS CodeGraph `.codegraph/` SQLite index to the new Pebble/protobuf format. Read path via a CGo-free `modernc.org/sqlite` v1.53.0 reader opened read-only (`mode=ro`+`query_only`, source never mutated), confined to `internal/migrate` by an import-graph archtest (no new CGo). Preserves TS node ids verbatim (faithful row-for-row conversion, not a lossy native re-derivation); carries `start_col`/`end_col`; drops FTS/vocab/unresolved_refs/sqlite-internals; builds the Phase-4 `x/` file-index during write (`Meta.has_file_index=true`). Resumable via an additive `PutMigration`/`GetMigration` `m/migration` cursor record + temp-dir→atomic-rename swap (with interrupted-swap self-heal). Fail-loud D-09 validation: de-dup-aware edge-count reconciliation (the new key collapses line/col), `file:`-exempt zero-dangling check (`--drop-dangling` explicit opt-in), schema-version-range guard, gating `Meta.Healthy`. Aged-`.codegraph/` fixtures reconstructed in-Go from the captured TS dump (no external `sqlite3`). A deep code review found 0 Criticals + 5 real Warnings the green TDD suite missed (stale `edge_count` after drop-dangling, un-`Close()`d trailing batch, overwrite-probe mutating the target dir, no interrupted-swap recovery, unescaped DSN path) — all fixed with regression tests. One new pure-Go dep (`modernc.org/sqlite`), no new CGo — **validated in Phase 7**
- Release hardening & benchmarks: single static CGo binary per platform (6 targets, native darwin matrix + zig cross for linux/windows), `release.yml` publishing per-binary cosign keyless signatures + syft SBOMs + SLSA provenance (all Actions SHA-pinned), `ci.yml` gating `govulncheck` + a `-mod=readonly` reproducible double-build + a network-free 120k-file perf-regression gate with an absolute peak-RSS ceiling, minimal audited deps (CGo tree-sitter the sole exception). Proven end-to-end on the real `v0.0.0-rc.3` release (`cosign verify-blob` OK against the shipped verifier's identity, wrong identity rejected). Median-of-3 benchmarks vs installed TS 1.3.1: Go wins every measured metric (indexing throughput 6.1×–59.7×, query latency ~2.3–2.8× lower, peak RSS 3.6×–5.5× lighter, cold start ~6× faster) — **validated in Phase 8**

### Active

**Current milestone — v1.0 (Drop-in Parity & Human UX):** closing the behavioral and surface gaps that keep v0.1 from being a true drop-in replacement (see Current Milestone above), then cutting the first signed `v1.0.0`. The parity work is evidence-scoped from a live TS 1.3.1 vs codegraph-go bake-off (dual-indexed the same tree): the command surface is already a **superset** — no TS command is missing — so the gaps are behavioral (`explore` relevance, `node` multi-def), the watcher-on-MCP default, git/worktree awareness, output hygiene, and human-facing polish, not missing commands.

Deferred to later releases:

- [ ] Central graph server — multi-user, remote queries, auth (Team Scale; v0.1's architecture was designed to accommodate it without a rewrite)
- [ ] CI-built shared index distribution / caching (Team Scale)
- [ ] Graph annotations the v0.1 schema reserved space for — embedding vectors, community assignments, bulk export for visualization
- [ ] Local Svelte web UI for browsing/querying the graph (SEED-001 — triggers once parity lands; the v1.0 bubbletea TUI is a distinct terminal surface)
- [ ] Worktree support **beyond** TS parity — auto-init or `git-common-dir` index sharing (v1.0 ships TS-parity detect+warn+notice only; going further is a deliberate later call)

### Out of Scope

- Hosted platform features (getcodegraph.com-style PR analysis) — different product; not this project's goal
- Bundling any runtime with the binary — antithesis of the design; the static binary IS the distribution
- Embeddings/vector search, community detection, graph visualization UI — planned future milestones; the schema anticipates them (annotations, export) but v0.1 does not implement them. Embeddings will be local-model-first; cloud-API embeddings are permanently out

## Context

**Source project:** colbymchenry/codegraph, TypeScript (~4.5 MB source), v1.3.1, ~59k stars, MIT license. Builds a SQLite knowledge graph in `.codegraph/` per project, exposes it via CLI and MCP server, auto-syncs via file watcher, supports ~12 languages with cross-file resolution (framework-aware routes, mixed iOS/React Native bridging). Distributes via `curl | sh` installer or npm, bundling its own Node.js runtime — the supply-chain surface this port eliminates.

**User environment:** Sean runs TS CodeGraph daily as an MCP server across his projects, so parity gaps will be felt immediately — a strength for validation. Work team uses Java/C# heavily (hence its priority position); open-source release is intended from day one, so docs, release engineering, and compatibility promises matter early.

**Repo state:** **v0.1 (Initial Release) SHIPPED 2026-07-14** — all 8 phases complete. The core capabilities (index/query/MCP/sync/migrate) work in a single static Go binary, from a real cosign-signed / SLSA-attested / SBOM'd release (`v0.0.0-rc.3`, verified end-to-end via `cosign verify-blob` against the shipped verifier's identity) that outperforms TS 1.3.1 on every measured benchmark (median-of-3: 6.1×–59.7× indexing throughput, ~3× lower query latency, 3.5×–5.5× lighter peak RSS). **Not yet drop-in parity** — the CLI command surface diverges from TS CodeGraph; closing that gap is the remaining bar for a 1.0. Earlier milestone history follows. Phase 1 landed the substrate (Pebble-backed `GraphStore` behind a concurrency-tested interface, protobuf schema-versioned records, CGo tree-sitter parser seam, golden TS ground-truth corpus). Phase 2 shipped the working Go indexer: `codegraph init`/`index`/`uninit` build a correct, cross-file-resolved graph from scratch via a deterministic two-pass (parallel-extract → sequential-resolve) pipeline — validated end-to-end (self-indexing this repo: files=48 nodes=414 edges=660, byte-identical rebuild). Phase 3 opened the graph to agents: the full read-only query command suite plus a stdio MCP server (`serve --mcp`, `mark3labs/mcp-go`), reading through the frozen Phase-2 graph via an additively-extended `Reader` (node/file iterators; query-time reverse adjacency). Output-shape parity with TS v1.3.1 is proven by a live golden-corpus diff (7/7 tools against the pinned `weft` checkout); a deep code review's two Critical findings (default-`--limit` DoS; MCP `path` confinement) were fixed before completion.

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
| Full parity in v1 (not core-first) | Drop-in swap is the success bar; partial parity can't validate it | ⚠️ Partial — v0.1 shipped the core capabilities + a signed release, but the CLI command surface diverges from TS, so it is NOT yet a drop-in parity replacement. Versioned 0.1 (not 1.0) to reflect this; full CLI-surface parity is the remaining bar for 1.0. |
| New index format + migration tool | Optimize schema for concurrency/perf/scale; converter keeps existing users | ✓ Converter shipped (Phase 7): faithful TS-id-preserving one-way migrate, resumable + fail-loud validated |
| Parity v1 → team features v2 | Ship replacement value first; architect so server/CI features bolt on | — v0.1 shipped core value; team features (server/CI) still deferred to a later milestone as designed. Note: parity itself is not yet complete (see above), so v0.1 precedes the parity/1.0 bar. |
| Language priority: Go → Java/C# → Python → TS/JS | Matches Sean's daily usage and work-team stack | ✓ Delivered (Phase 5): 14 registered languages, priority-4 at full extraction + resolution on real repos, mainstream-6 full-or-documented-partial |
| Full supply-chain suite from first release | Signing, SLSA, SBOM, reproducibility are the differentiator, not an afterthought | ✓ Shipped (Phase 8): per-binary cosign keyless + SLSA provenance + syft SBOM + govulncheck + reproducible double-build, proven on real `v0.0.0-rc.3` |
| Parser strategy (CGo tree-sitter vs wazero WASM vs native Go) | Performance vs purity vs sandboxing — needs quantified research | ✓ Resolved (Phase 1): CGo tree-sitter, benchmarked; single documented CGo exception (PARSER-DECISION.md) |
| Plan for embeddings, communities, graph-viz UI as future milestones | Long-term product direction; v1 schema versioned + annotation-ready so they bolt on | — Pending |
| Milestone v1.0 = drop-in parity + human UX | v0.1 shipped core value but diverges from TS *behavior*; parity is the honest 1.0 bar. Scope derived from a live dual-indexed bake-off, not docs | — In progress: behavioral parity (`explore`/`node`), watcher-on-MCP default, git/worktree awareness, output hygiene, Charm TUI, then signed v1.0.0 |
| Include bubbletea/Charm TUI in v1.0 | TS's human output is colorized + interactive (`daemon` picker) — prettiness is part of parity, not gold-plating | — Planned; agent/MCP output stays plain/parseable, human path gets lipgloss/bubbletea, all TTY-gated |
| Worktree awareness scoped at parity for v1.0 | TS `sync/worktree.js` only detects a borrowed index + warns; matching that is the 1.0 bar. Auto-init/share is a larger design | — v1.0 = detect+warn+notice; "make worktree support better later" (user-confirmed) |

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
*Last updated: 2026-07-14 — started milestone v1.0 (Drop-in Parity & Human UX)*
