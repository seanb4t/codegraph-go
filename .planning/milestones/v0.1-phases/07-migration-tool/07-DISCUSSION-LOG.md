# Phase 7: Migration Tool - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-12
**Phase:** 7-Migration Tool
**Mode:** `--auto` (all gray areas auto-selected to recommended defaults)
**Areas discussed:** Node/edge identity translation, Field mapping, Resumability & version-stamping, Command surface & source detection, Validation & failure policy

---

## Node/Edge Identity Translation

| Option | Description | Selected |
|--------|-------------|----------|
| Preserve TS ids verbatim | Copy `nodes.id` + `edges.source/target` unchanged; faithful row-for-row conversion | ✓ |
| Recompute native ids + remap edges | Compute `nodeid.NodeID(kind, qualified_name, file_path)` per node, build old→new map, rewrite all edges | |
| Re-index from source, migration = trigger + validate | Read TS DB only for repo/paths, then run the native indexer on the source tree | |

**Choice:** Preserve TS ids verbatim.
**Notes:** TS ids and native ids share the `<kind>:<32-hex>` shape but not the value (TS = 32-hex/MD5-style digest; `nodeid.NodeID` = `sha256(...)[:32]`). Recomputing is also lossy — the native Go extractor has documented divergences from TS (call-as-arg edges, WR-01/WR-02, RefKind scoping). Re-index-from-source contradicts the stated design (`modernc.org/sqlite` reader over the aged index, no re-parse) and requires the source tree. Preserving keeps the user's actual aged graph, self-consistent edges, and passes structural invariants. Open question flagged for researcher: whether to stamp the migrated `Meta` so a later `sync`/`index` reconciles cleanly vs. accepting a full-index reconcile.

---

## Field Mapping — carry vs. drop

| Option | Description | Selected |
|--------|-------------|----------|
| Direct table→record mapping + drop derived/aux tables | nodes/edges/files→Node/Edge/File+Meta; drop FTS, name_segment_vocab, unresolved_refs, sqlite internals; build x/ index | ✓ |
| Carry everything (add proto fields for all TS columns) | Additively model is_async/static/abstract/decorators/type_parameters, persist unresolved_refs | |

**Choice:** Direct mapping; drop derived/aux/internal tables; build the `x/` file-index during write; set `Meta.has_file_index = true`.
**Notes:** FTS + name_segment_vocab are regenerable / query-time in the new format; unresolved_refs isn't persisted by the new format. TS-only node attributes with no proto field are dropped (adding fields is out of migration scope — deferred).

---

## Resumability & Version-Stamping (MIGR-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Batched idempotent writes + progress record | Bounded batches; progress record (source ver, cursor, status) in meta; resume by rowid; temp-dir→atomic-swap | ✓ |
| Restart-from-scratch only | Idempotent deterministic keys make a full restart safe; no cursor | |

**Choice:** Batched idempotent writes + progress record + atomic directory swap.
**Notes:** Writes keyed by the preserved TS id → partial-batch replay is an idempotent overwrite. Temp-dir→rename ensures an interrupted run never leaves a `.codegraph/` that looks complete. `Meta.healthy` set only after validation passes.

---

## Command Surface & Source Detection (MIGR-01)

| Option | Description | Selected |
|--------|-------------|----------|
| `codegraph migrate [--from][--to][--force]`, auto-detect + in-place swap | Default from/to = cwd `.codegraph/`; detect TS via schema_versions+nodes; non-destructive to source | ✓ |
| Explicit-paths-required, side-by-side output | Force user to name source and a separate destination dir | |

**Choice:** `codegraph migrate [--from <path>] [--to <path>] [--force]`; auto-detect TS SQLite by table presence; in-place via temp→swap; never mutate/delete source.
**Notes:** One command, one step. New `internal/migrate` package + cobra subcommand (Phase-6 cli pattern).

---

## Validation & Failure Policy (MIGR-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Loud structural-invariant pass | Count reconciliation + zero-dangling-edges (fail loud, `--drop-dangling` opt-out) + readability/schema guard + defensive aged-DB column reads; gate Meta.healthy | ✓ |
| Best-effort, warn-and-continue | Convert whatever parses, log warnings, never fail | |

**Choice:** Loud invariant pass; fail on dangling edges / unreadable source / unsupported schema version by default.
**Notes:** Aligns with the project's fail-loud / never-silently-wrong ethos (deep review caught swallowed-I/O + data-loss criticals in Phases 4 & 6). Defensive column reads (`PRAGMA table_info`/`COALESCE`) tolerate aged DBs missing later-added columns.

---

## Claude's Discretion

- Batch size, progress-record key name, temp-dir naming, reconciliation-report formatting.
- Whether per-file `node_count`/`edge_count` are carried from TS or recomputed on write.

## Deferred Ideas

- Add proto fields for TS-only node attributes (is_async/static/abstract/decorators/type_parameters) — additive, belongs with a consuming feature.
- Teach `sync`/`index` to reconcile a TS-migrated graph's ids without full churn — sync-engine concern, follow-up.
- Two-way / export-back-to-TS migration — out of scope (one-way only).
