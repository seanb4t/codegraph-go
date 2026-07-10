# CodeGraph Go

## What This Is

A ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) — the pre-indexed code knowledge graph for coding agents (Claude Code, Cursor, Codex, Gemini, etc.). One static Go binary replaces the TypeScript version's bundled-Node distribution, delivering the same agent-facing experience (CLI, MCP server, auto-sync) with better performance, a verifiable supply chain, and an architecture designed to grow into team-scale usage.

## Core Value

An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better — faster, from a single verifiably-built binary.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Index a codebase into a knowledge graph (symbols, edges, files) with structural extraction and cross-file resolution at parity with TS CodeGraph v1.3.x
- [ ] MCP server exposing the same tool surface as TS CodeGraph (`codegraph_explore` and companions) so existing agent configs work unchanged
- [ ] CLI with parity commands: `init`, `install`, `uninstall`, `uninit`, `upgrade`, `explore`, plus migration
- [ ] Auto-sync: file watcher keeps the graph current on every change with no manual re-runs
- [ ] Agent installer/uninstaller support for the same agent roster (Claude Code, Cursor, Codex, OpenCode, Hermes, Gemini, Antigravity, Kiro)
- [ ] 12+ language support at parity, built in priority order: Go → Java/C# → Python → TypeScript/JavaScript → remainder
- [ ] New storage format designed for concurrent access, incremental updates, and monorepo scale — with a migration tool converting existing `.codegraph/` SQLite indexes
- [ ] Single static binary per platform (macOS/Linux/Windows), minimal audited dependency tree, no bundled runtime
- [ ] Release integrity: cosign-signed artifacts, SLSA build provenance, published SBOM, vuln scanning gating CI, reproducible builds
- [ ] Published benchmarks vs TS CodeGraph on real repos (indexing throughput, query latency, memory)

### Out of Scope

- Central graph server (multi-user, remote queries, auth) — milestone 2; v1 architecture must anticipate it, not implement it
- CI-built shared index distribution/caching — milestone 2, same rule: design for it, don't build it yet
- Hosted platform features (getcodegraph.com-style PR analysis) — different product; not this project's goal
- Bundling any runtime with the binary — antithesis of the design; the static binary IS the distribution
- Feature additions beyond TS parity in v1 — parity plus performance is the bar; new capabilities wait for v2

## Context

**Source project:** colbymchenry/codegraph, TypeScript (~4.5 MB source), v1.3.1, ~59k stars, MIT license. Builds a SQLite knowledge graph in `.codegraph/` per project, exposes it via CLI and MCP server, auto-syncs via file watcher, supports ~12 languages with cross-file resolution (framework-aware routes, mixed iOS/React Native bridging). Distributes via `curl | sh` installer or npm, bundling its own Node.js runtime — the supply-chain surface this port eliminates.

**User environment:** Sean runs TS CodeGraph daily as an MCP server across his projects, so parity gaps will be felt immediately — a strength for validation. Work team uses Java/C# heavily (hence its priority position); open-source release is intended from day one, so docs, release engineering, and compatibility promises matter early.

**Repo state:** Greenfield — empty repository, no commits yet.

**Known open question (for research):** Parser strategy is the central performance-vs-purity tension: tree-sitter via CGo (fastest, breaks pure-Go static builds), tree-sitter grammars compiled to WASM run via wazero (pure Go, sandboxes grammar code — itself a supply-chain win, some speed cost), or native Go parsers (huge effort, best control). Research must quantify this before architecture locks.

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
| New index format + migration tool | Optimize schema for concurrency/perf/scale; converter keeps existing users | — Pending |
| Parity v1 → team features v2 | Ship replacement value first; architect so server/CI features bolt on | — Pending |
| Language priority: Go → Java/C# → Python → TS/JS | Matches Sean's daily usage and work-team stack | — Pending |
| Full supply-chain suite from first release | Signing, SLSA, SBOM, reproducibility are the differentiator, not an afterthought | — Pending |
| Parser strategy (CGo tree-sitter vs wazero WASM vs native Go) | Performance vs purity vs sandboxing — needs quantified research | — Pending research |

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
*Last updated: 2026-07-10 after initialization*
