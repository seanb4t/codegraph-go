# Roadmap: CodeGraph Go

## Overview

CodeGraph Go delivers a drop-in Go replacement for TypeScript CodeGraph: full v1.3.x parity plus performance and supply-chain wins in a single static binary. The journey runs foundation-first — lock the storage schema, the `GraphStore` boundary, and the parser strategy (CGo vs wazero) before anything builds on them — then proves the pipeline end-to-end on Go, opens the query engine and MCP server to agents, makes the graph self-maintaining via the watcher/daemon, broadens language and framework coverage, wires the existing-user adoption path (agent install + migration), and finishes by hardening the release (signing, SLSA, SBOM, reproducibility) and publishing head-to-head benchmarks that validate every prior phase. Golden-output corpus and the TS schema DDL are captured up front, while the live TS version is available, so parity is measured against ground truth rather than memory.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation — Storage, Schema & Parser Strategy** - Schema-versioned graph store behind a `GraphStore` interface, with a benchmarked parser decision and captured TS ground-truth (completed 2026-07-10)
- [x] **Phase 2: Go Indexing Pipeline** - Two-pass indexer proven on Go: `init`/`index` build a correct, cross-file-resolved graph from scratch (completed 2026-07-11)
- [x] **Phase 3: Query Engine & MCP Server** - Full query command suite plus a parity stdio MCP server verified against the golden-output corpus (completed 2026-07-11)
- [x] **Phase 4: Incremental Sync & File Watcher** - Auto-updating graph with correct rename/delete pruning, debounced native watchers, daemon, and leak-free soak (completed 2026-07-11)
- [x] **Phase 5: Language Coverage & Resolution Breadth** - Parity languages, interface-dispatch synthesis with provenance, and framework-aware routing (completed 2026-07-12)
- [ ] **Phase 6: Agent Integrations & CLI Lifecycle** - 8-agent install/uninstall, self-upgrade, and CLI ergonomics for the drop-in swap
- [ ] **Phase 7: Migration Tool** - Resumable, validated converter from TS SQLite `.codegraph/` to the new format
- [ ] **Phase 8: Release Hardening & Benchmarks** - Signed, attested, reproducible static-binary releases with published head-to-head benchmarks and CI regression gates

## Phase Details

### Phase 1: Foundation — Storage, Schema & Parser Strategy

**Goal**: A schema-versioned graph store with a clean interface boundary and a chosen, benchmarked parser approach — the substrate every extractor, query, and migration builds on.
**Depends on**: Nothing (first phase)
**Requirements**: INDX-05, ARCH-01
**Success Criteria** (what must be TRUE):

  1. All graph reads and writes go through a `GraphStore` interface; concurrent readers run lock-free alongside a single coordinated writer, verified by a concurrency test that no package bypasses the interface
  2. The on-disk format is schema-versioned and round-trips future node/edge annotation fields (embedding vectors, community assignments) and a bulk graph export without a format break, verified by a version/export test
  3. The parser strategy (CGo tree-sitter vs wazero WASM) is selected from a head-to-head spike, with parse throughput and static-build impact documented as the basis for the decision
  4. A golden-output corpus and the TS `.codegraph/` schema DDL are captured from the live TS CodeGraph v1.3.x so later phases can measure parity against ground truth

**Plans**: 7/7 plans complete
**Wave 1**

- [x] 01-01-PLAN.md — Bootstrap Go module + pinned deps + package skeleton (Wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — Protobuf schema-versioned records + forward-compat round-trip (Wave 2, ARCH-01)
- [x] 01-03-PLAN.md — Narrow Parser interface + CGo tree-sitter backend (Go+Python) with size ceiling (Wave 2)
- [x] 01-04-PLAN.md — Golden corpus + TS SQLite DDL capture from live TS v1.3.1 (Wave 2)
- [x] 01-05-PLAN.md — Typed keyspace encoders + delimiter-injection guard (Wave 2, INDX-05)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-06-PLAN.md — GraphStore interface + pebble/v2 impl + concurrency + bulk export + import-graph boundary test (Wave 3, INDX-05/ARCH-01)
- [x] 01-07-PLAN.md — Parser spike (CGo vs wazero) benchmark + crash isolation + PARSER-DECISION.md (Wave 3)

### Phase 2: Go Indexing Pipeline

**Goal**: A user can index a Go repository from scratch and get a correct, cross-file-resolved, queryable graph — proving the two-pass indexer mechanism on the first language.
**Depends on**: Phase 1
**Requirements**: INDX-01, INDX-02, RES-01, LANG-01
**Success Criteria** (what must be TRUE):

  1. User runs `codegraph init` in a Go repo and gets a `.codegraph/` with a fully built graph in one step; `uninit [--force]` removes it cleanly
  2. User runs `codegraph index` for a deterministic from-scratch rebuild with `--force`, `--quiet`, and `--verbose`
  3. Cross-file resolution links imports, call edges, and type inheritance across a multi-package Go repo via two-pass parallel-extract → sequential-resolve
  4. Indexing a real-world Go project produces symbols and edges that match expected structure for that repo

**Plans**: 6/6 plans complete

**Wave 1**

- [x] 02-01-PLAN.md — Additive schema extension (D-03) + content-hashed node identity (D-02/D-02a) + promote x/sync,x/mod
- [x] 02-02-PLAN.md — Multi-package Go test fixture + file discovery seam (go/build.MatchFile + import-path tagging)

**Wave 2** *(blocked on Wave 1)*

- [x] 02-03-PLAN.md — Pass 1: Go AST→vocabulary mapper (LANG-01) + bounded worker pool (one Parser/worker)

**Wave 3** *(blocked on Wave 2)*

- [x] 02-04-PLAN.md — Pass 2: symbol index + cross-file resolution (RES-01) + deterministic edge collapse + single-writer commit

**Wave 4** *(blocked on Wave 3)*

- [x] 02-05-PLAN.md — Pipeline orchestration + byte-identical determinism gate (INDX-02) + structural fixture-diff

**Wave 5** *(blocked on Wave 4)*

- [x] 02-06-PLAN.md — Cobra CLI lifecycle: init/index/uninit + binary entrypoint (INDX-01, INDX-02)

### Phase 3: Query Engine & MCP Server

**Goal**: Agents and users can interrogate the graph through the full command suite and a parity stdio MCP server whose output shapes match the TS original.
**Depends on**: Phase 2
**Requirements**: QRY-01, QRY-02, QRY-03, QRY-04, QRY-05, QRY-06, QRY-07, QRY-08, QRY-09, MCP-01, MCP-02, MCP-03, MCP-04
**Success Criteria** (what must be TRUE):

  1. User can run `query`, `node`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, and `status` with their documented flags (`--kind`/`--limit`/`--json`/`--depth`/etc.) and get correct results from the graph
  2. User runs `codegraph explore <query>` and gets verbatim line-numbered source of relevant symbols grouped by file, the call paths between them, and a blast-radius summary in one round trip
  3. An agent connects to `codegraph serve --mcp` and sees `codegraph_explore` as the only default tool; the 7 additional tools appear only when listed in `CODEGRAPH_MCP_TOOLS`; the server exposes zero tools when no `.codegraph/` exists
  4. MCP tool output shapes match TS CodeGraph v1.3.x, verified against the golden-output corpus captured in Phase 1

**Plans**: 9/9 plans complete

**Wave 1**

- [x] 03-01-PLAN.md — Additive Reader node/file enumeration (D-03) (Wave 1, QRY-01/03/07/09)

**Wave 2** *(blocked on Wave 1)*

- [x] 03-02-PLAN.md — Query engine foundation: resolve/OpenAt/validation clamps + test harness (Wave 2, QRY-01)

**Wave 3** *(blocked on Wave 2)*

- [x] 03-03-PLAN.md — Deterministic lexical search: query + search (Wave 3, QRY-01/03)
- [x] 03-04-PLAN.md — Traversal: callers/callees/impact/affected over reverse adjacency (Wave 3, QRY-04/05/06)
- [x] 03-05-PLAN.md — files browse + status --json with D-05 remapping (Wave 3, QRY-07/09)

**Wave 4** *(blocked on Wave 3)*

- [x] 03-06-PLAN.md — node + explore markdown templates (D-05a/b) (Wave 4, QRY-02/08)

**Wave 5** *(blocked on Wave 4)*

- [x] 03-07-PLAN.md — Stdio MCP server + tool gating + mcp-go dep (Wave 5, MCP-01/02/03)

**Wave 6** *(blocked on Wave 5)*

- [x] 03-08-PLAN.md — CLI wiring: 11 commands + serve + live-MCP checkpoint (Wave 6, QRY-01..09/MCP-01)

**Wave 7** *(blocked on Wave 6)*

- [x] 03-09-PLAN.md — Golden parity harness vs weft corpus (Wave 7, MCP-04)

### Phase 4: Incremental Sync & File Watcher

**Goal**: The graph stays current automatically as files change — correct pruning on any file operation, debounced native watching, a shared daemon, and no goroutine leaks.
**Depends on**: Phase 3
**Requirements**: INDX-03, INDX-04, SYNC-01, SYNC-02, SYNC-03, SYNC-04, SYNC-05, SYNC-06
**Success Criteria** (what must be TRUE):

  1. User runs `codegraph sync` and only changed files are reparsed via content-hash diffing with dependent-file edge recomputation — not a full re-index of a subset
  2. File renames, deletes, and moves prune stale symbols and edges with no orphaned nodes or dangling edges, verified by a fixture suite
  3. Editing files auto-updates the graph via native per-OS watchers (FSEvents/inotify/ReadDirectoryChangesW), debounced and consolidating edit bursts; agent-facing output shows a staleness banner while a sync is pending
  4. On MCP server (re)connect, offline changes are reconciled via stat comparison plus content hashing
  5. `codegraph daemon` runs a shared watch/index server (with in-process fallback where unsupported), `codegraph unlock` clears stale locks after a crash, and soak tests show the watcher/daemon is goroutine-leak-free

**Plans**: 9/9 plans complete
**Wave 1**

- [x] 04-01-PLAN.md — x/ file-owned secondary index for O(subgraph) prune (Wave 1, INDX-04)
- [x] 04-02-PLAN.md — schema mtime/size/has_file_index fields + export BuildReverseAdjacency (Wave 1, INDX-03)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 04-03-PLAN.md — internal/indexer.Sync() store-seeded incremental engine (Wave 2, INDX-03)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 04-04-PLAN.md — rename/delete/move prune fixtures + sync-equals-reindex determinism (Wave 3, INDX-04/INDX-03)
- [x] 04-05-PLAN.md — internal/watch: fsnotify recursive watcher + debounce (Wave 3, SYNC-01)
- [x] 04-06-PLAN.md — staleness banner + MCP reconnect reconcile (Wave 3, SYNC-02/SYNC-03)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 04-07-PLAN.md — internal/daemon: lockfile + shared single-writer daemon + unlock (Wave 4, SYNC-04/SYNC-05)

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 04-08-PLAN.md — CLI sync/daemon/unlock + root wiring + serve in-process fallback (Wave 5, INDX-03/SYNC-04/SYNC-05)
- [x] 04-09-PLAN.md — goroutine-leak soak (goleak) for watcher + daemon (Wave 5, SYNC-06)

### Phase 5: Language Coverage & Resolution Breadth

**Goal**: Parity language support with cross-file resolution, dynamic-dispatch synthesis carrying provenance, and framework-aware routing for the priority stack.
**Depends on**: Phase 2
**Requirements**: LANG-02, LANG-03, LANG-04, LANG-05, LANG-06, LANG-07, RES-02, RES-03
**Success Criteria** (what must be TRUE):

  1. Java, C#, Python, and TypeScript/JavaScript each get full structural extraction plus cross-file resolution, validated on real repos in that language
  2. The mainstream tier (Rust, Ruby, PHP, C/C++, Swift, Kotlin) is supported at full or explicitly documented-partial coverage
  3. Interface→implementation dispatch edges are synthesized (Go implicit interfaces, Java/C# interface implementations) so callers and impact traverse dynamic dispatch
  4. Every heuristic (synthesized) edge carries a `provenance: heuristic` tag with source location, distinguishing it from ground-truth AST edges
  5. Framework-aware routing for Gin, Spring, ASP.NET, Django/Flask/FastAPI, and Express/NestJS emits `route` nodes linked to their handlers

**Plans**: 13/13 plans complete

**Wave 1 — Foundation seam**

- [x] 05-01-PLAN.md — Language registry + priority-4 parser constructors + grammar pins + additive vocabulary (LANG-02/03/04/05)

**Wave 2 — Pass-1 generalization** *(blocked on Wave 1)*

- [x] 05-02-PLAN.md — Discovery ext→language walker + descriptor/ModuleKey hook + worker-pool per-language fix (Pitfall 1) + pipeline dispatch (LANG-02/03/04/05)

**Wave 3 — Pass-2 generalization** *(blocked on Wave 2)*

- [x] 05-03-PLAN.md — Resolve/symbolindex per-language ModuleKey (Pitfall 2) + D-05 Go fixes (WR-01, WR-02, call-as-argument) (LANG-02/03/04/05)

**Wave 4 — Priority-4 languages + mainstream grammar setup** *(blocked on Wave 3, parallel)*

- [x] 05-04-PLAN.md — Java full extraction + resolution + golden parity (LANG-02)
- [x] 05-05-PLAN.md — C# full extraction + resolution + partial-class decision + golden parity (LANG-03)
- [x] 05-06-PLAN.md — Python full extraction + resolution + golden parity (LANG-04)
- [x] 05-07-PLAN.md — TypeScript/TSX/JavaScript full extraction + resolution + golden parity (LANG-05)
- [x] 05-08-PLAN.md — Mainstream grammar setup: 7 parser constructors + pins + Swift/Kotlin [SUS] human-verify gate (LANG-06)

**Wave 5 — Dispatch synthesis + mainstream extractors** *(blocked on Wave 4, parallel)*

- [x] 05-09-PLAN.md — implements-edge synthesis (Go structural + Java/C# declared) + conformance retry + query-time dispatch traversal + provenance (RES-02, RES-03)
- [x] 05-10-PLAN.md — Mainstream extractors: Rust, Ruby, PHP (LANG-06)
- [x] 05-11-PLAN.md — Mainstream extractors: C/C++ (shared) + Swift/Kotlin (LANG-06)

**Wave 6 — Framework routing** *(blocked on Wave 5)*

- [x] 05-12-PLAN.md — Route detector registry + Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS detectors (LANG-07)

**Wave 7 — Coverage matrix + closeout** *(blocked on Wave 6)*

- [x] 05-13-PLAN.md — D-11 capability matrix (human + machine-readable) + cross-language regression closeout (LANG-06)

### Phase 6: Agent Integrations & CLI Lifecycle

**Goal**: Existing agent users can install CodeGraph Go into their tools, self-upgrade safely, and rely on complete CLI ergonomics — the mechanics of the drop-in swap.
**Depends on**: Phase 3
**Requirements**: AGNT-01, AGNT-02, AGNT-03, CLI-01, CLI-02, CLI-03
**Success Criteria** (what must be TRUE):

  1. User runs `codegraph install` to detect and configure the 8-agent roster (Claude Code, Cursor, Codex CLI, opencode, Hermes, Gemini CLI, Antigravity, Kiro) — MCP config plus marker-fenced instruction injection — and re-running is idempotent
  2. User runs `codegraph uninstall` to cleanly reverse everything `install` wrote while preserving user edits outside the markers
  3. Per-agent quirks are handled correctly (e.g., Cursor's injected `--path` arg for MCP subprocess cwd)
  4. User runs `codegraph help [command]` and `codegraph version` with standard ergonomics, and `codegraph upgrade [version]` self-updates via signature-verified download-and-swap with `--check`
  5. `codegraph telemetry` reports that the build contains no telemetry, and the binary contains zero phone-home code

**Plans**: 6 plans

**Wave 1**

- [ ] 06-01-PLAN.md — internal/agents foundation: AgentTarget interface + surgical write helpers + registry + instructions block + hujson pin (AGNT-01/02)
- [ ] 06-05-PLAN.md — internal/version package + `version`/`telemetry` commands + help ergonomics (CLI-01/03)

**Wave 2** *(blocked on Wave 1)*

- [ ] 06-02-PLAN.md — JSON agent targets: Claude/Cursor/Gemini/Kiro/Antigravity (+ --path, no-type, project-root, self-heal quirks) (AGNT-01/02/03)
- [ ] 06-03-PLAN.md — Format agent targets: Codex TOML / opencode JSONC (hujson) / Hermes YAML surgery (AGNT-01/02/03)
- [ ] 06-06-PLAN.md — `upgrade [version] [--check]`: redirect-trick release resolution + sigstore-go verify-before-swap + atomic self-replace + sigstore-go pin (CLI-02)

**Wave 3** *(blocked on Wave 2)*

- [ ] 06-04-PLAN.md — `install`/`uninstall` commands: --target/--location, os.Executable(), TTY multi-select w/ auto fallback, per-agent status + live-agent checkpoint (AGNT-01/02/03)

### Phase 7: Migration Tool

**Goal**: An existing TS CodeGraph user can convert their aged `.codegraph/` SQLite index to the new format in one resumable, validated step.
**Depends on**: Phase 2 (stable storage schema and writer)
**Requirements**: MIGR-01, MIGR-02
**Success Criteria** (what must be TRUE):

  1. User runs a single migration command that converts an existing TS `.codegraph/` SQLite index to the new format
  2. Migration is resumable after interruption and version-stamped so partial runs recover correctly
  3. Migration is validated against real aged `.codegraph/` directories and runs structural-invariant checks on the result, failing loudly on corruption rather than producing a silently-wrong graph

**Plans**: TBD

### Phase 8: Release Hardening & Benchmarks

**Goal**: A trustworthy, fast v1.0 release — signed, attested, reproducible static binaries with a minimal audited dependency tree, plus published head-to-head benchmarks and CI regression gates that validate the whole project.
**Depends on**: All prior phases (scaffolds early, validates at end)
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, PERF-01, PERF-02, INDX-06
**Success Criteria** (what must be TRUE):

  1. User can download a single static binary per platform (macOS/Linux/Windows, amd64+arm64) with no bundled runtime and no install-time compilation
  2. Every release artifact is cosign-signed (keyless) with SLSA build provenance and publishes an SBOM; users can verify signature and provenance with documented commands; `govulncheck` and dependency scanning gate CI; builds are reproducible, verified by a double-build comparison gate; and the dependency tree stays minimal and audited with CGo (if selected by the parser spike) as the sole documented exception
  3. Published head-to-head benchmarks vs TS CodeGraph (indexing throughput, query latency, peak RSS, cold start) on real repos use comparable methodology with raw per-repo numbers
  4. User can index a 100k+ file monorepo within bounded memory, with peak RSS tracked as a first-class CI metric and performance-regression gates running against a benchmark corpus that includes that monorepo

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8

*Parallelism note (informational, not execution order): Phases 5 and 6 may run in parallel after Phase 3; Phase 7 needs only Phase 2's stable schema; Phase 8 scaffolds early and validates last.*

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation — Storage, Schema & Parser Strategy | 7/7 | Complete    | 2026-07-10 |
| 2. Go Indexing Pipeline | 6/6 | Complete    | 2026-07-11 |
| 3. Query Engine & MCP Server | 9/9 | Complete    | 2026-07-11 |
| 4. Incremental Sync & File Watcher | 9/9 | Complete    | 2026-07-11 |
| 5. Language Coverage & Resolution Breadth | 14/13 | Complete    | 2026-07-12 |
| 6. Agent Integrations & CLI Lifecycle | 0/6 | Planned | - |
| 7. Migration Tool | 0/TBD | Not started | - |
| 8. Release Hardening & Benchmarks | 0/TBD | Not started | - |
