# Roadmap: CodeGraph Go

## Overview

CodeGraph Go is a ground-up Go rewrite of TypeScript CodeGraph — the eventual goal is a drop-in, TS-v1.3.x-parity replacement in a single static binary. **v0.1 (Initial Release) shipped 2026-07-14** — all 8 foundation-through-hardening phases complete: the core capabilities (indexing, query, MCP server, sync, migration) work, from a real cosign-signed / SLSA-attested / SBOM'd release, with published head-to-head benchmarks showing the Go binary beating TS 1.3.1 on every measured metric. It is **not yet at full CLI-surface parity** with TS CodeGraph — closing that gap is a goal for a later release.

## Milestones

- ✅ **v0.1 — Initial Release** — Phases 1–8 (shipped 2026-07-14) — core capabilities + signed release; not yet a drop-in parity replacement
- 📋 **Next** — parity gap-closure (CLI surface alignment) and/or Team Scale (central server, CI-distributed indexes) — not yet planned

## Phases

<details>
<summary>✅ v0.1 — Initial Release (Phases 1–8) — SHIPPED 2026-07-14</summary>

Full phase details archived in [`milestones/v0.1-ROADMAP.md`](milestones/v0.1-ROADMAP.md).

- [x] Phase 1: Foundation — Storage, Schema & Parser Strategy (7/7 plans) — completed 2026-07-10
- [x] Phase 2: Go Indexing Pipeline (6/6 plans) — completed 2026-07-11
- [x] Phase 3: Query Engine & MCP Server (9/9 plans) — completed 2026-07-11
- [x] Phase 4: Incremental Sync & File Watcher (9/9 plans) — completed 2026-07-11
- [x] Phase 5: Language Coverage & Resolution Breadth (14/13 plans) — completed 2026-07-12
- [x] Phase 6: Agent Integrations & CLI Lifecycle (6/6 plans) — completed 2026-07-12
- [x] Phase 7: Migration Tool (7/7 plans) — completed 2026-07-13
- [x] Phase 8: Release Hardening & Benchmarks (9/9 plans) — completed 2026-07-14

**Delivered:** the core capabilities of CodeGraph in a single static Go binary — index/query/MCP/sync/migrate — faster and lighter than TS 1.3.1 (6.1×–59.7× indexing throughput, ~3× lower query latency, 3.5×–5.5× lighter peak RSS), from a signed/attested/SBOM'd release verified end-to-end on `v0.0.0-rc.3`. **Known gap:** the CLI surface diverges from TS CodeGraph — v0.1 is not yet a drop-in parity swap.

</details>

### 📋 Next (Planned)

Not yet planned — start with `/gsd-new-milestone`. Candidate scope:

- [ ] **Parity gap-closure** — align the CLI command surface with TS CodeGraph v1.3.x so existing users/agent configs work unchanged (the remaining bar for a "parity" claim / a 1.0)
- [ ] **Team Scale** (carried from v0.1 Out-of-Scope; v0.1's architecture was designed to accommodate it without a rewrite): central graph server (multi-user, remote queries, auth); CI-built shared index distribution/caching
- [ ] Graph annotations the schema reserved space for: embedding vectors, community assignments, bulk export for visualization

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation — Storage, Schema & Parser Strategy | v0.1 | 7/7 | Complete | 2026-07-10 |
| 2. Go Indexing Pipeline | v0.1 | 6/6 | Complete | 2026-07-11 |
| 3. Query Engine & MCP Server | v0.1 | 9/9 | Complete | 2026-07-11 |
| 4. Incremental Sync & File Watcher | v0.1 | 9/9 | Complete | 2026-07-11 |
| 5. Language Coverage & Resolution Breadth | v0.1 | 14/13 | Complete | 2026-07-12 |
| 6. Agent Integrations & CLI Lifecycle | v0.1 | 6/6 | Complete | 2026-07-12 |
| 7. Migration Tool | v0.1 | 7/7 | Complete | 2026-07-13 |
| 8. Release Hardening & Benchmarks | v0.1 | 9/9 | Complete | 2026-07-14 |
