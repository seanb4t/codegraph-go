# Research Summary: v0.12.0 Local Graph UI

**Researched:** 2026-08-22 (consolidated from STACK, FEATURES, ARCHITECTURE, PITFALLS parallel research)  
**Confidence:** HIGH to MEDIUM across all dimensions  
**Status:** Ready for phase roadmap definition

---

## Executive Summary

The v0.12.0 Local Graph UI milestone adds a read-only, browser-based code-intelligence interface to codegraph-go via ConnectRPC + Svelte/shadcn-svelte SPA, with live push updates from the existing watcher. The stack is tightly scoped and proven in precedent — ConnectRPC is validated as a minimal-dependency alternative to gRPC for local loopback use, Cytoscape.js is established as the right rendering choice at file/package scale, and `go:embed`'d committed SPA assets keep the release pipeline pure Go per the project's supply-chain constraints. The architecture reuses `internal/query.Engine` as a third consumer (after CLI and MCP), meaning no new graph-mutation code and significant risk reduction. The critical architectural finding: Pebble's multi-process lock contention is already solved via bounded retry + fresh-per-call semantics from MCP's existing pattern, not a new problem. The single largest new backend work is a fresh `Engine.FileGraph()` method for aggregated file/package rollup — a straightforward full-scan operation following established patterns — and two structured-result methods extracted from Node/Explore's existing fetch pipelines. Supply-chain is the execution frontier: pnpm's strict isolation and default-blocked build scripts add real security benefits but require explicit approval-state pinning and non-vacuous drift guards to avoid silent build degradation.

---

## Key Findings

### From STACK.md: Recommended Technologies

**Core stack is maintainer-directed, versions verified live 2026-08-22:**
- **ConnectRPC** (connect-go v1.20.0, connect-es v2.1.2): 2-dependency Go footprint, server-streaming works over HTTP/1.1 natively, significantly smaller surface than gRPC.
- **Svelte 5.56.10 + shadcn-svelte 1.5.0**: Runes API stable; copy-in-source components (code ownership), but pulls real npm runtime deps (Radix/Bits UI) into pnpm-lock.yaml.
- **Cytoscape.js 3.34.1**: Canvas-based by default, well-performing at file/package scale (hundreds to low-thousands nodes). Layered/hierarchical layout (dagre) correct for DAG structure, not force-directed.
- **pnpm 11.22.0** (project constraint): Pinned via Corepack's `packageManager` field. Strict non-hoisting is pnpm's core value but documented breakage vector for npm-assuming tools.
- **Build tooling isolation**: Go-tool directives for Go plugins; pnpm devDependencies for TS side. buf CLI via pnpm (not `go install`).

**Critical supply-chain additions:**
- `pnpm audit` for JS ecosystem (govulncheck has zero visibility into pnpm-lock.yaml).
- Lifecycle-script approval state pinned in `pnpm-workspace.yaml`, not interactive/per-machine.
- `//go:embed all:dist` with `all:` prefix (excludes dotfiles/underscores by default; Vite outputs `_app`-prefixed chunks).

### From FEATURES.md: What the UI Should Do

**Table-stakes (v1 MVP):**
- Search-as-you-type; jump-to-definition; deep-linkable URLs; syntax-highlighted source; Query Workbench (impact/affected/callers/callees); Index Health (freshness/coverage/staleness/worktree-mismatch).

**Differentiators (v1.x, after MVP validates):**
- Impact/affected blast-radius workbench; aggregated file/package graph with directory grouping + layered layout; live view updates on re-index.

**Rejected anti-features:**
- In-browser editing, multi-user auth, force-directed whole-graph.

**Engine gaps identified (actionable for phase planning):**
- `Engine.NodeDetail()` — structured result, extracted from Node's fetch pipeline
- `Engine.ExploreResult()` — structured result, extracted from Explore's gather pipeline
- `Engine.FileGraph()` — aggregated file/package adjacency with per-kind edge counts

### From ARCHITECTURE.md: Integration Points

**System structure:**
- `codegraph ui`: own CLI + separate OS process; loopback bind + Origin/Host validation middleware; handlers open fresh Engine per RPC (**never cache long-lived handle**).
- SPA via `//go:embed all:dist`, fallback to index.html for client routing.
- Live push: fsnotify on `.codegraph/.sync-pending` sidecar (reuses existing watcher signal).

**Protobuf split:**
- Storage format (`codegraph.v1`, private to graphstore) — unchanged.
- Wire API (`codegraph.ui.v1`) — new package, distinct evolution cadence.

**Engine reuse:**
- 8 of 10 methods return JSON-friendly structs; thin field-mapping suffices.
- 2 methods (Node, Explore) return markdown only; need NodeDetail/ExploreResult extracted via pure separation of fetch from render. Frozen goldens untouched.
- FileGraph() computes rolled-up adjacency fresh per call.

**Drift guards (non-vacuous, per rule 84d1gfpywd):**
- `proto_drift_test.go`: regenerate, byte-diff, assert file count > 0.
- `dist_drift_test.go`: hash source tree, compare, assert source count > 0.
- Both demonstrated RED before green CI.

**Pebble multi-process access — VERIFIED FINDING (orchestrator ruling applied):**
- Pitfalls.md claimed "needs IPC-to-daemon" — **FALSE**.
- **VERIFIED**: `internal/graphstore/pebble_store.go::Open()` already has bounded retry (5 attempts, 100ms backoff) + `ErrStoreLocked` sentinel.
- **Precedent**: `internal/mcp/server.go::openEngine()` opens fresh Engine on every MCP call inside long-lived server.
- **Solution**: UI opens fresh per RPC (like MCP), closes before return. Lock held briefly, collisions non-permanent.
- **Real concern**: UI must handle `ErrStoreLocked` as "indexing in progress", not error.
- **Integration test required**: Real daemon + UI concurrently; verify sync flushes don't starve.

### From PITFALLS.md: Critical Risks

**Security (host/origin validation):**
- Pitfalls 1-3: Loopback bind pairs with exact-match Host allowlist in middleware before any handler. CVE-2024-28224 (Ollama), CVE-2025-66414/66416 (MCP SDKs) are precedent.
- Pitfall 4 (CORS): Avoid permissive config in shipped binary. Same-origin needs no CORS. Reserve dev-only build tag.

**Path safety:**
- Pitfall 5: Route all source through Engine accessor, not direct filesystem reads (reuse MCP confinement logic).

**Supply-chain integrity (pnpm-specific):**
- Pitfall 7: Use `//go:embed all:dist`.
- Pitfall 8-9: Pin approval in pnpm-workspace.yaml; enable strictDepBuilds (unapproved = hard CI failure, not silent skip).
- Pitfall 10: Guard must assert file count > 0 before trusting diff.
- Pitfall 11: Structurally enforce zero pnpm/node in .goreleaser.yaml or release.yml.
- Pitfall 13: `pnpm audit` must parse output (not bare exit code) to distinguish "clean" from "check failed".

**Rendering/UX:**
- Pitfall 18-20: Use file/package aggregation + layered layout (not force-directed). Live-push re-renders preserve positions. Client: exponential backoff on reconnect.
- Pitfall 21: Staleness/worktree-mismatch part of live-push or polled, not fetched once.

**Concurrency/correctness:**
- Pitfall 22 (CR-01 precedent): Fix CR-01 early; design live-push: "does server-initiated get distinguished from request-response in lifecycle bookkeeping?"

---

## Implications for Roadmap

### Suggested Phase Structure (9 phases)

**Phase 1: Engine Seam Extraction & Protobuf Scaffolding**
- NodeDetail, ExploreResult extraction; new proto schema split (ui.v1); isolated tool-modfile.
- Unblocks downstream; clean boundary; matches constraints.
- Risk: LOW (extraction, frozen goldens guard).
- Research: STANDARD.

**Phase 2: Transport Layer & Security Foundation**
- CLI command + loopback bind + Host/Origin validation middleware + ConnectRPC mux; security tests.
- Security gates ship in transport, guard all handlers.
- Risk: MODERATE (security boundary, well-documented).
- Research: STANDARD.

**Phase 3: Read-Only RPC Handlers & Query Workbench**
- Handlers for 7 structured methods; Query Workbench forms; SPA skeleton (embed all:dist, fallback).
- Thin field-mapping, zero logic. First UAT checkpoint.
- Risk: LOW.
- Research: STANDARD.

**Phase 4: Browse & Inspect + Index Health**
- NodeDetail RPC + source spans; Index Health (freshness/coverage/staleness/worktree-mismatch as trust banners); syntax highlighting + breadcrumb.
- Reuses Engine; UI-side rendering. Path-confinement regression test (Pitfall 5).
- Risk: LOW-MODERATE.
- Research: STANDARD.

**Phase 5: Deep-Linking & Routing Infrastructure**
- URL-based routing, state encoding, back/forward (free via History API).
- Table-stakes; 3 features depend on it; pure client-side.
- Risk: LOW.
- Research: STANDARD.

**Phase 6: File/Package Graph Rollup**
- Engine.FileGraph() method; Cytoscape with directory grouping, layered layout (dagre), cycle highlighting.
- Largest new backend work; gated on Phase 1, 5 stable. Differentiator, doesn't block MVP.
- Risk: MODERATE-HIGH (rendering scale risk; mitigated by benchmark).
- **Research: MODERATE** — benchmark Cytoscape at project's largest corpora; if degradation past 2k files, spike on Sigma/graphology now.

**Phase 7: Live Push & Index Health Updates**
- fsnotify on .sync-pending; fan-out registry (buffered, non-blocking); streaming RPC; Svelte subscriptions; "stale since when" timestamp.
- Completes differentiator. **MUST integration-test against real daemon+UI concurrently** (Pebble finding).
- Risk: MODERATE (lock interaction requires careful testing).
- **Research: MODERATE** — integration test mandatory before ship.

**Phase 8: Drift Guards & Supply-Chain Hardening**
- proto_drift_test.go + dist_drift_test.go; pnpm audit gate; allowBuilds config; release pipeline Node/pnpm-free.
- Supply-chain non-negotiable; demonstrated RED before GREEN.
- Risk: LOW-MODERATE.
- Research: STANDARD.

**Phase 9: CR-01 Fix (Independent)**
- internal/mcp/server.go::pendingwriter fix; regression test; Phase 7 design review vs. precedent.
- No UI dependency; recommend early.
- Risk: LOW.
- Research: STANDARD.

### Critical Path & Parallelization

```
Phase 1 (Engine + proto)
  ├─→ Phase 2 (Transport + security)
  │    └─→ Phase 3 (Handlers + Query WB)
  │         ├─→ Phase 4 (Browse + Health)
  │         ├─→ Phase 5 (Deep-link + routing)
  │         │    └─→ Phase 6 (FileGraph)
  │         │         └─→ Phase 7 (Live push, integration tested)
  │
  ├─→ Phase 8 (Drift + supply-chain) [parallel]
  └─→ Phase 9 (CR-01) [independent, recommend early]
```

**Ship gates:**
- **MVP**: End Phase 5 (search, browse, query, health, deep-link, history).
- **Differentiators**: End Phase 7 (+ graph, + live push).
- **Supply-chain**: End Phase 8.

### Research Flags

**Phases needing research:**
- **Phase 6 (Graph)**: Benchmark Cytoscape at actual large-repo scale; if degradation past 2k files, spike Sigma/graphology.
- **Phase 7 (Live push)**: Integration testing vs. real daemon non-negotiable.

**Phases with standard patterns (skip):**
- Phase 1-5, 8-9: Established patterns, codebase precedent.

### Confidence Breakdown

| Area | Confidence | Notes |
|------|------------|-------|
| Stack versions & choices | HIGH | Verified 2026-08-22; proven at comparable scale. |
| Feature scope & priorities | HIGH | Landscape survey + Engine capabilities; anti-features coherent. |
| Architecture & integration | HIGH | Engine reuse, MCP precedent, existing signals; Pebble verified upstream. |
| Security mitigations | HIGH | Well-documented (CVE-2024-28224, MCP SDK CVEs); standard practice. |
| pnpm supply-chain | MEDIUM-HIGH | Official docs verified; organizational adoption risk (not technical). |
| Graph rendering at scale | MEDIUM | Cytoscape proven at stated scale; design risk mitigated by Phase 6 benchmark. |
| Live-push multi-process | MEDIUM | Mechanism understood; integration testing mandatory, not yet done. |

### Gaps to Address

1. **FileGraph() aggregation semantics**: Pin exact algorithm in Phase 1 (edge counts by kind? distinct (src-file, dst-file, kind) tuples?).
2. **SPA build determinism**: Prove `pnpm install --frozen-lockfile && vite build` produces byte-identical dist/ twice, or design wrapper.
3. **Index Health "discovered but not indexed"**: Genuine gap. Phase 4: v1 requirement or defer?
4. **Graph-view interaction UX**: "Click neighbor" + "blast radius" exact behavior unspecified. Phase 6 planning.
5. **Live-push event payload granularity**: Does .sync-pending carry summary (change count, timestamp), or does UI re-fetch Status() on every push?
6. **pnpm audit non-vacuity**: CI assertion (JSON parsing, file count) required before Phase 8 ships. Demonstrated-red test.
7. **Cycle highlighting trivial?**: Phase 6 planning should confirm cycle detection on file/package DAG is cheap.

---

## Two CR-01s: Explicit Disambiguation

This milestone touches two findings historically both called CR-01. **Disambiguation:**

1. **THIS MILESTONE'S CR-01** = `internal/mcp/server.go::pendingWriter` counter corruption from server-initiated notifications. **OPEN**, folded into Phase 9 per PROJECT.md ("second long-lived connection-bearing surface, so the existing one being correct matters more").

2. **v0.5.0 PHASE-3 CR-01** = Pebble lock multi-process contention (cockroachdb/pebble#1583). **ALREADY FIXED** by bounded-retry loop + ErrStoreLocked in `internal/graphstore/pebble_store.go`.

Phase 7 design review must reference Pitfall 22 + CR-01 precedent as checklist item, ensuring new ConnectRPC streaming lifecycle doesn't reintroduce equivalent counter-corruption bug.

---

## Sources

### Stack Research
- ConnectRPC, npm registry (verified 2026-08-22)
- pnpm GitHub Releases, official docs/blog (2026-08-22)
- GitHub Issues: cockroachdb/pebble#1583, vitejs/vite#324, golang/go#42328/#43854
- Context7 documentation

### Features Research
- Tool landscape: SourceTrail, go-callvis, madge, dependency-cruiser, VS Code, OpenGrok
- Microsoft Research: "Trimming the Hairball"
- Direct source audit: internal/query/engine.go, PROJECT.md, SEED-001

### Architecture Research
- Direct source: internal/graphstore/pebble_store.go, internal/mcp/server.go, internal/query/engine.go
- cockroachdb/pebble#1583, cockroachdb/pebble#1081
- PROJECT.md (maintainer directives)

### Pitfalls Research
- NCC Group: CVE-2024-28224 (Ollama)
- GitHub Security Advisory: CVE-2025-66414, CVE-2025-66416 (MCP SDKs)
- pnpm docs, Socket.dev (pnpm v10), pnpm issue #10235
- MSR literature: "Trimming the Hairball"
- Direct source: internal/mcp/server.go (CR-01)

---

*Research synthesis completed 2026-08-22. Ready for roadmap phase planning.*
