# Project Research Summary

**Project:** CodeGraph Go — Go port of TypeScript CodeGraph
**Synthesized from:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md
**Date:** 2026-07-10

## Executive Summary

CodeGraph Go is a Go port of TypeScript CodeGraph targeting **full parity plus performance and supply-chain improvements**. Core decisions are settled (Go + tree-sitter + Pebble storage + GraphStore interface boundary), with one open question: CGo vs WASM/wazero for the parser (Phase-1 spike required, ~2x WASM overhead reported but unvalidated for this workload).

Highest implementation risks are **edge-case correctness**: incremental indexing corruption on renames/deletes, file-watcher reliability across platforms, defensive migration tooling. Research identifies **10 critical pitfalls** from comparable projects, with concrete prevention strategies.

**Recommended roadmap:** 9 phases prioritizing foundation work (storage schema, parser strategy, two-pass indexer) → core features (queries, MCP, watcher, agent integrations) → expansion (languages, frameworks, release infrastructure) → validation (benchmarking). Phases 5–7 can run in parallel after Phase 3 MCP server stabilizes.

## Key Findings

**Stack:** Go 1.24+, tree-sitter/go-tree-sitter (v0.25.x + per-language grammars), cockroachdb/pebble (new graph store), mark3labs/mcp-go (MCP server), spf13/cobra (CLI), fsnotify (watcher). Parser strategy (CGo vs WASM/wazero) open question. Release infrastructure required from v1: GoReleaser + cosign + SLSA L3 + Syft SBOM + govulncheck.

**Features:** Full TS CodeGraph v1.3.x parity mandatory. Table stakes: `init/install/index/sync/explore/callers/impact/affected` + 8-agent integrations + auto-sync watcher + migration tool + static binary + benchmarks. Deferred: cross-language synthesizers (v1.x), central server (v2).

**Architecture:** Five-layer standard pattern (access → query → update → extract → store). Two-Pass Indexing (parallel extraction → sequential cross-file resolution). Storage Port (GraphStore interface for v2 server swap without rewrite). Single-Writer/Multi-Reader with lock coordination. Evidence-Based Incremental Invalidation.

**Critical Pitfalls:** (1) CGo breaks cross-compilation story, (2) SQLite WAL ≠ concurrent writes, (3) incremental sync corrupts graph on renames/deletes, (4) watcher reliability gaps across platforms, (5) per-language resolution complexity stalls parity, (6) migration tool not resumable, (7) monorepo-scale memory blowup, (8) goroutine leaks in daemon, (9) MCP output drift from TS original, (10) supply-chain theater (SLSA/reproducibility not verified).

## Implications for Roadmap

**9-Phase Structure:**

1. **Foundation (Parser Strategy, Storage Schema):** Spike CGo vs WASM/wazero. Finalize storage schema. Define GraphStore interface, core domain types, registry factory pattern. *Research flags: parser benchmarks, TS schema DDL.*

2. **Two-Pass Indexer & Go Extraction:** Implement parser interface + Go language. Extractor (symbols + unresolved refs). Content-hash tracker. Validate on real Go repos. *Avoids Pitfalls 3, 7.*

3. **Query Engine & MCP Server:** Graph queries (call paths, blast radius). `explore` algorithm. MCP stdio server + tool registration. CLI commands. **Set up golden-output corpus testing.** *Avoids Pitfall 9.*

4. **Incremental Sync & File Watcher:** fsnotify integration, debounce, event-overflow handling. Dirty-file tracker + dependent re-resolve. Staleness banner. Goroutine-leak detection + soak tests. *Avoids Pitfalls 3, 4, 8.*

5. **Agent Integrations (8-Agent Roster):** Per-agent installer adapters. `install/uninstall/upgrade`. *Parallel with Phases 4–6.*

6. **Language Coverage (Go→Java/C#→Python→TS/JS):** Per-language: (a) static extraction + same-file resolution, (b) cross-file resolution, (c) framework synthesis (v1.x backlog). *Research flags: per-language resolver correctness.* *Parallel with Phase 5.*

7. **Migration Tool (TS SQLite → New Format):** Multi-phase resumable converter. Validate on real aged `.codegraph/` directories. Structural-invariant checks. Version-stamped. *Avoids Pitfall 6.*

8. **Release Engineering (Signing, SLSA, SBOM, Reproducible Builds):** GoReleaser + cosign + SLSA L3 + double-build CI gate + `govulncheck`. *Avoids Pitfall 10.* *Parallel with Phases 6–7.*

9. **Benchmarking & Validation:** Same-repo corpus as TS CodeGraph. Metrics: throughput, latency, RSS. Regression gates. Published results. *Validates Phases 1–8.*

**Ordering:** Phases 1–4 sequential (dependencies). Phases 5–6 parallel after Phase 3. Phase 7 after Phase 1 storage finalized. Phase 8 scaffolds early, validates after Phase 7. Phase 9 last.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM | HIGH on tech choices; MEDIUM on parser strategy (empirical data pending). |
| Features | MEDIUM-HIGH | HIGH on parity list; MEDIUM on numeric benchmarks (directional). |
| Architecture | HIGH | Five-layer converged across 7+ comparable projects. SQLite concurrency, patterns cross-verified. |
| Pitfalls | HIGH | From official docs + real production issues (TS PR #900, comparable projects). |

**Overall:** MEDIUM-HIGH. Ready for phase planning. Gaps: parser-strategy spike, TS schema DDL, per-language resolver validation, platform-specific watcher behavior.

## Sources

- `.planning/research/STACK.md` — technology choices, parser-strategy tradeoff analysis, release tooling
- `.planning/research/FEATURES.md` — TS CodeGraph v1.3.x parity inventory, differentiators, anti-features
- `.planning/research/ARCHITECTURE.md` — component boundaries, data flow, build order, concurrency model
- `.planning/research/PITFALLS.md` — 10 critical pitfalls with prevention strategies and phase mapping
