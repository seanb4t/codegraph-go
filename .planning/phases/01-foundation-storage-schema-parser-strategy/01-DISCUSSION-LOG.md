# Phase 1: Foundation — Storage, Schema & Parser Strategy - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-10
**Phase:** 1-Foundation — Storage, Schema & Parser Strategy
**Mode:** `--auto` (autonomous — recommended defaults auto-selected, no interactive prompts)
**Areas discussed:** Storage engine, Record encoding & schema evolution, Keyspace layout, GraphStore interface & concurrency, Parser spike methodology, Golden-corpus & TS DDL capture

---

## Storage Engine

| Option | Description | Selected |
|--------|-------------|----------|
| Pebble (`cockroachdb/pebble`) | Pure-Go LSM KV; snapshots, range deletes, `IndexedBatch`; lock-free readers + single writer | ✓ |
| Badger (`dgraph-io/badger`) | WiscKey key/value separation; wins for large values | |
| bbolt (`etcd-io/bbolt`) | Simplest, longest track record; single-writer transaction model | |

**Selected:** Pebble (recommended default)
**Notes:** Matches INDX-05's concurrency requirement and the research recommendation in `.claude/CLAUDE.md`. bbolt rejected for single-writer bottleneck; Badger deferred unless profiling shows large-value dominance.

---

## Record Encoding & Schema Evolution

| Option | Description | Selected |
|--------|-------------|----------|
| Protobuf + meta schema_version | Field-numbered records, reserved field numbers for future annotations, unknown-field preservation | ✓ |
| Versioned Go struct + MessagePack | Lighter dep, but weaker field-evolution guarantees | |
| JSON | Human-readable, but no field-number stability / larger on disk | |

**Selected:** Protobuf field-numbered records + `meta` `schema_version` (recommended default)
**Notes:** This is the concrete mechanism satisfying ARCH-01 (future annotations + export without a format break). Additive-only field discipline within a major schema version.

---

## Keyspace Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Prefix-namespaced typed keys | `meta/`, `n/`, `e/<src>/<type>/<dst>`, `f/`, reserved `a/` annotations | ✓ |
| Separate DBs per entity type | Cleaner separation, but loses cross-entity atomic batches | |
| Single flat namespace | Simplest, but no range-scan/range-delete leverage | |

**Selected:** Prefix-namespaced typed keys with reserved annotation namespace (recommended default)
**Notes:** Edge ordering by source enables lock-free callers/callees/impact range scans; a file's whole subgraph is prunable via one range delete (feeds Phase 4).

---

## GraphStore Interface & Concurrency

| Option | Description | Selected |
|--------|-------------|----------|
| Snapshot reads + batched writes, single coordinated writer, interface-only access + bulk export | Lock-free readers via Pebble snapshots; one writer owned by the store; import-graph test guards the boundary | ✓ |
| Mutex-guarded direct engine access | Simpler, but couples every package to the engine | |
| Transaction-per-operation | Correct but high write-amplification at index scale | |

**Selected:** Snapshot reads + batched writes + single writer + interface-only access (recommended default)
**Notes:** The concurrency/architecture test (no package bypasses the interface) is the enforceable form of INDX-05.

---

## Parser Spike Methodology

| Option | Description | Selected |
|--------|-------------|----------|
| CGo-default, spike-to-disprove on Go + 1 external-scanner language, behind a narrow Parser interface | Benchmark throughput / incremental / static-build / crash-isolation; CGo unless wazero proves free | ✓ |
| WASM-default (wazero) | Pure-Go + crash isolation, but builds the grammar-to-WASM pipeline up front with ~2x parse cost unproven | |
| Full 12-language benchmark before deciding | Most thorough, but disproportionate effort for a v1 gate | |

**Selected:** CGo-default disprove-spike on Go + one external-scanner language, behind a narrow parser interface (recommended default; **outcome pending spike**)
**Notes:** Engram founding memory records parser strategy as "unresolved pending research." CONTEXT locks the spike method and decision criteria, not the result. Default lean = CGo (research Option A); adopt wazero only if overhead is invisible against the full pipeline and the WASM pipeline cost is acceptable.

---

## Golden-Corpus & TS DDL Capture

| Option | Description | Selected |
|--------|-------------|----------|
| SQLite `.schema`/`.dump` DDL + JSON snapshots of explore/companion outputs on a pinned corpus, in `testdata/golden/` | Ground truth captured now while live TS v1.3.x is available | ✓ |
| Capture later | Risks the live TS version drifting or being uninstalled | |
| Capture DDL only | Misses tool-output-shape parity needed by Phase 3 (MCP-04) | |

**Selected:** DDL + golden JSON snapshots on a pinned small corpus, version-stamped in `testdata/golden/` (recommended default)
**Notes:** Time-sensitive one-shot; gates parity verification in Phases 3/4/7. Corpus = a compact Go repo + the TS `colbymchenry/codegraph` repo itself.

---

## Claude's Discretion

- Exact node-id / edge-key byte encoding (must preserve range-scan + range-delete properties).
- Precise protobuf message shapes for node/edge/file records (design against captured TS DDL).
- Spike harness structure and the specific benchmark repo (must include Go + one external-scanner language).
- Whether `meta` counts are maintained incrementally or recomputed (Phase 2+ concern).

## Deferred Ideas

- Embeddings / vector search, community detection, graph-viz UI — post-v1; v1 only anticipates them (reserved annotation namespace, forward-compat encoding, bulk export).
- TS→new-format migration converter — Phase 7 (Phase 1 only captures the DDL).
- Indexer / extractors / query commands / MCP tools / file watcher — Phases 2–4.
- Team/server features — v2; architecture must not preclude them.
