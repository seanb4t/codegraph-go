# Phase 7: Migration Tool - Context

**Gathered:** 2026-07-12
**Status:** Ready for planning
**Mode:** `--auto` (decisions auto-selected to recommended defaults; review before planning)

<domain>
## Phase Boundary

Deliver a single `codegraph migrate` command that converts an existing **TypeScript CodeGraph `.codegraph/` SQLite index** into the new Go format (Pebble LSM store + protobuf records). The conversion is:

- **One step** (MIGR-01) — one command turns a TS index into a new-format index.
- **Resumable + version-stamped** (MIGR-02) — an interrupted run recovers correctly and never leaves a half-baked directory that looks complete.
- **Validated + fail-loud** (MIGR-02) — structural-invariant checks run on the result; corruption fails loudly rather than producing a silently-wrong graph.

**In scope:** the read path over the TS SQLite schema (via the pure-Go `modernc.org/sqlite` reader), faithful record translation into `schema.Node/Edge/File/Meta`, writing through the existing `graphstore.Writer`, resumability/version-stamping, and the validation pass.

**Out of scope:** re-parsing source code (migration converts the *aged index*, not the repo); two-way / round-trip migration (one-way only); any change to the native indexer or query surface; TS-only node attributes the new proto doesn't define (see D-05).
</domain>

<decisions>
## Implementation Decisions

> All decisions below were auto-selected (`--auto`) to the recommended option. They are locked for research/planning unless the user revises them.

### Node/Edge Identity Translation (the central decision)
- **D-01:** **Preserve TS node ids verbatim.** Copy `nodes.id` into `Node.id` and copy `edges.source`/`edges.target` unchanged. Do **not** recompute native `nodeid.NodeID` values.
  - *Why:* TS ids and native ids share the `<kind>:<32-hex>` **shape** but not the **value** — TS uses a 32-hex (MD5-style) digest; `internal/indexer/nodeid/nodeid.go` deliberately uses `sha256(...)[:32]`, which is a different value for the same `(kind, qualified_name, file_path)`. Recomputing would also be **lossy**, because the native Go extractor has documented divergences from TS (call-as-argument edges, WR-01/WR-02, RefKind scoping — see prior-phase memory). A faithful migration preserves the user's *actual* aged graph, keeps every edge self-consistent, and passes structural invariants without pretending to be a native re-index.
  - **Open question for researcher (do not silently ignore):** a graph carrying TS ids will differ from what `codegraph index`/`sync` computes, so a later `sync` could see spurious churn. Decide whether to (a) accept that the next full `index` reconciles, or (b) stamp the migrated `Meta` so `sync`/`index` recognizes a TS-migrated graph and reconciles cleanly. Prefer the least-surprising, no-silent-corruption path.

### Field Mapping — what carries vs. what drops
- **D-02:** **Direct table→record mapping.** `nodes`→`Node`, `edges`→`Edge`, `files`→`File`, plus a single `Meta`. Map every field the target proto defines (`internal/schema/graph.proto`): positions, `signature`, `docstring`, `visibility`, `is_exported`, `return_type`; `files.content_hash/language/modified_at/size` → `File.content_hash/language/mtime_unix_ns/size_bytes`; per-file `node_count`/`edge_count` recomputed or carried. Edge `metadata` JSON → `Edge.metadata` (`map<string,string>`; flatten/stringify non-string values), plus `line`/`col`/`provenance`.
- **D-03:** **Drop (do not translate):** `nodes_fts*` and `name_segment_vocab` (Phase 3 does deterministic lexical search at query time — no persisted FTS), `unresolved_refs` (the new format does not persist unresolved refs), and SQLite internals (`sqlite_sequence`, `sqlite_stat1`, `schema_versions`). Map `project_metadata` into `Meta` where a field exists, else drop.
- **D-04:** **Build the `x/` file-owned secondary index during migration.** Route writes through `graphstore.Writer.PutEdge(e, ownerPath)` with the owning file path so the Phase-4 `x/` index is populated; set `Meta.has_file_index = true`. (Avoids forcing a one-time full re-index backfill on the migrated graph's first `sync`.)
- **D-05:** **Drop TS-only node attributes with no proto field** (`is_async`, `is_static`, `is_abstract`, `decorators`, `type_parameters`, `start_column`/`end_column` if unmodeled). Adding proto fields for them is an additive future change, not migration scope (see Deferred).

### Resumability + Version Stamping (MIGR-02)
- **D-06:** **Batched idempotent writes + a progress record.** Write into the target Pebble store in bounded batches via `Writer.Commit()`. Persist a migration-progress record (source `schema_versions` max, target `SchemaVersion`, cursor = last-committed table + rowid, `status = in_progress|complete`) under a dedicated meta key. On resume, read it and skip completed tables / resume mid-table by rowid. Because writes are keyed by the (preserved) TS id, replaying a partial batch is an idempotent overwrite — safe.
- **D-07:** **Atomic directory swap.** Write the new-format store into a temp directory, then rename into place on success, so an interrupted run never leaves a `.codegraph/` that looks complete but isn't. Stamp `Meta.healthy = true` only after the validation pass (D-09) passes.

### Command Surface + Source Detection + Output (MIGR-01)
- **D-08:** **`codegraph migrate [--from <path>] [--to <path>] [--force]`**, wired via the Phase-6 cobra/cli pattern in a new `internal/migrate` package.
  - `--from` default = the TS `.codegraph/` in cwd; auto-detect the SQLite DB file inside it.
  - `--to` default = new-format `.codegraph/` (in place, via temp→swap per D-07).
  - **Detect a TS source** by opening the SQLite file and checking for the `schema_versions` + `nodes` + `edges` tables (not by filename alone).
  - **Non-destructive to source:** never delete/mutate the source TS DB; refuse to overwrite a non-empty target that isn't recognizably a prior migration without `--force`.

### Validation / Structural-Invariant Checks + Failure Policy (MIGR-02)
- **D-09:** **Post-migration invariant pass, fail loud:**
  1. **Count reconciliation** — migrated `Node`/`Edge`/`File` counts equal source row counts (minus intentionally-dropped tables); report the reconciliation.
  2. **Referential integrity** — every migrated edge's `source` and `target` resolve to a migrated node. Dangling edges are a corruption signal → **fail loudly with a report by default**; offer `--drop-dangling` to instead drop-and-log (an explicit, opt-in lossy migration).
  3. **Readability / schema guard** — reject an unreadable or locked SQLite file, missing required tables, or a source `schema_versions` outside the documented supported range (loud, never silent).
  4. **Aged-DB tolerance** — read columns defensively (`PRAGMA table_info` or explicit column lists with `COALESCE`) so an *aged* index missing later-added columns (`edges.provenance`, `nodes.return_type`, `unresolved_refs.file_path`/`language`) migrates instead of crashing.
- **D-10:** **`Meta.healthy` gates on all checks.** The migration reports success only after D-09 passes end-to-end.

### Locked / Carried Forward (not gray areas)
- **Dependency:** add `modernc.org/sqlite` (pure-Go, CGo-free) as the **read-only** migration reader — mandated by PROJECT.md / CLAUDE.md ("migration reader only, isolated to a one-shot code path"). Open the DB **read-only**. **No new CGo** — tree-sitter stays the sole exception.
- **Reader/writer boundary:** read via `database/sql` + the `modernc.org/sqlite` driver; write exclusively through the existing `graphstore.Writer` batch API — reuse the storage layer, do not reinvent it.
- **Placement:** new `internal/migrate` package; `migrate` subcommand registered on the root cobra command (Phase-6 cli wiring pattern).

### Claude's Discretion
- Exact batch size, progress-record key name, temp-directory naming, and reconciliation-report formatting are implementation details left to planning/execution.
- Whether per-file `node_count`/`edge_count` are carried from TS or recomputed during the write (either is acceptable if counts reconcile).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 7 "Migration Tool" goal + 3 success criteria; depends on Phase 2 (stable storage schema & writer).
- `.planning/REQUIREMENTS.md` — **MIGR-01** (one-step convert) and **MIGR-02** (resumable, version-stamped, validated against real aged dirs, structural-invariant checks).
- `.planning/PROJECT.md` — constraints: one-way migration from TS SQLite; `modernc.org/sqlite` as the isolated read-only reader; no new CGo; "everything works the same or better" core value.

### TS source format (migration reads THIS)
- `testdata/golden/ts-schema.sql` — the TS `.codegraph/` schema DDL (tables `schema_versions`, `nodes`, `edges`, `files`, `unresolved_refs`, `name_segment_vocab`, `project_metadata`, FTS5 virtual tables). **The migration reader's source-of-truth schema.**
- `testdata/golden/ts-schema.dump.sql` — representative `.dump` showing real TS id format (`<kind>:<32-hex>`), null-handling, `unistr(...)` docstrings, epoch-ms timestamps — the concrete shapes the reader must parse.
- `.planning/phases/01-foundation-storage-schema-parser-strategy/01-04-PLAN.md` — the capture harness that produced the fixtures above (shells out to the live TS CLI + `sqlite3`). **How to obtain / regenerate a real aged `.codegraph/` for MIGR-02 validation** — the researcher must confirm whether a committed aged SQLite fixture exists or must be generated.

### Target format (migration writes THIS)
- `internal/schema/graph.proto` — target record shapes: `Node`, `Edge`, `File`, `Meta` (and which TS fields have no home → D-05).
- `internal/schema/meta.go` — `SchemaVersion` (currently `1`), `NewMeta()`, additive-only discipline (version-stamping for MIGR-02).
- `internal/graphstore/store.go` — `GraphStore` / `Reader` / `Writer` interfaces; the write boundary migration must go through.
- `internal/graphstore/batch.go` — `PutNode` / `PutEdge(e, ownerPath)` / `PutFile` / `Commit` semantics + the `x/` file-index population (D-04).
- `internal/indexer/nodeid/nodeid.go` — the native id scheme (`sha256(len-prefixed(kind,qualifiedName,filePath))[:32]`); **the reason preserve-vs-recompute is a real decision** (native value ≠ TS value). Available if the recompute alternative is ever chosen.

### Corpus (validation-target candidates)
- `testdata/golden/corpus/colbymchenry-codegraph` and `testdata/golden/corpus/weft-go` — pinned corpus repos. Note: these are *source repos*; the researcher must confirm whether an actual aged `.codegraph/` SQLite index accompanies them or must be produced via the 01-04 harness.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`graphstore.Writer` batch API** (`internal/graphstore/batch.go`): `PutNode`/`PutEdge(e, ownerPath)`/`PutFile`/`Commit()` — atomic batched writes plus automatic `x/` file-index maintenance. Migration writes through this exclusively.
- **`internal/schema`** protobuf records + `schema.NewMeta()` / `schema.SchemaVersion` for version-stamping.
- **`nodeid.NodeID`** (`internal/indexer/nodeid/nodeid.go`) — only needed if the recompute alternative to D-01 is chosen.
- **Phase-6 cobra/cli wiring** (`internal/cli`, `cmd/codegraph/main.go`) — the pattern for registering the new `migrate` subcommand with flags.
- **Phase-1 fixtures** under `testdata/golden/` — DDL + dump + corpus for the reader and validation.

### Established Patterns
- **Single-writer / atomic commit** (Phase 1/4): all writes go through one `Writer`, committed atomically — migration inherits this (aligns with the temp-dir→swap of D-07).
- **Additive-only schema discipline (D-02a)**: any proto field for TS-only attributes would be an additive change — but that's out of migration scope (D-05).
- **Fail-loud / never-silently-wrong ethos** (prior-phase memory: deep review caught swallowed-I/O + data-loss criticals): D-09's loud-failure policy is the phase-appropriate expression of this. **Recommend a deep `/gsd-code-review` after execution** — this phase is I/O-heavy (SQLite read + Pebble write + partial-run recovery), exactly the profile where TDD-green suites missed real bugs in Phases 4 & 6.

### Integration Points
- **Read side:** `database/sql` + `modernc.org/sqlite` (new dep) against the TS `.codegraph/*.db`, read-only.
- **Write side:** `graphstore.Open` + `Writer` into the new-format target directory.
- **CLI:** new `migrate` command on the root command; must not perturb existing `init`/`index`/`sync`/`serve`/`install` wiring.
- **Flake note (carry forward):** `internal/daemon` `TestSoak` + flush-lock tests are known pre-existing flakes under full-suite parallel load. If `go test ./...` fails there after Phase-7 changes (which touch no daemon code), re-run `go test ./internal/daemon/ -count=1` isolated before treating it as a regression.

</code_context>

<specifics>
## Specific Ideas

- The migration is a **pure data transform of the SQLite bytes** — it does NOT require the source repo tree, because `kind`, `qualified_name`, and `file_path` are all columns in the TS DB. This is the whole value over re-indexing: fast, offline, no grammars.
- TS id shape observed in the dump: `class:1aa9ad9ada394f639ed0f8104462aef5`, `import:daa6c015...`, `constant:0122...` — always `<kind>:<32-hex>`. Preserve verbatim (D-01).
- Aged-index reality: the TS DDL shows columns added over time with `DEFAULT` clauses (`edges.provenance DEFAULT NULL`, `unresolved_refs.file_path DEFAULT ''`, `unresolved_refs.language DEFAULT 'unknown'`). A genuinely aged DB may predate them → defensive column reads are mandatory (D-09.4).

</specifics>

<deferred>
## Deferred Ideas

- **Add proto fields for TS-only node attributes** (`is_async`, `is_static`, `is_abstract`, `decorators`, `type_parameters`) so migration can carry them losslessly. Additive schema change; belongs with whatever downstream feature actually consumes them, not migration. (Ref D-05.)
- **Teach `sync`/`index` to reconcile a TS-migrated graph's ids** without a full churn pass. Related to D-01's open question; a sync-engine concern, potentially Phase 8 / a follow-up, not migration itself.
- **Two-way / export-back-to-TS migration** — explicitly out of scope; the project is one-way only.

</deferred>

---

*Phase: 7-Migration Tool*
*Context gathered: 2026-07-12*
