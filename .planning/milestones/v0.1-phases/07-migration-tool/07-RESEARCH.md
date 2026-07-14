# Phase 7: Migration Tool - Research

**Researched:** 2026-07-12
**Domain:** One-way SQLite→Pebble data migration (pure-Go SQLite read path + protobuf record translation + resumable/validated write)
**Confidence:** HIGH

## Summary

This phase converts a TypeScript CodeGraph `.codegraph/` SQLite index into the new Go Pebble/protobuf format in one resumable, validated command. The phase is exceptionally well-scoped by CONTEXT.md (10 locked decisions from a `--auto` discuss pass). Research confirms every locked decision is implementable against the existing codebase with no architecture changes: the read path is `database/sql` + `modernc.org/sqlite` (pure Go, CGo-free), the write path is the existing `graphstore.Writer` batch API (unchanged), and the target schema (`internal/schema/graph.proto`) has a direct field home for every TS column the project intends to carry.

The single largest open risk (MIGR-02 success criterion 3, "validated against real aged `.codegraph/` directories") is confirmed: **no committed aged SQLite `.db` fixture exists under `testdata/`** — only the DDL (`ts-schema.sql`) and a determinism-stripped representative dump (`ts-schema.dump.sql`, with `<EPOCH_MS>` placeholders). The planner MUST specify a fixture-reconstruction harness that materializes a real SQLite `.db` from the dump (substituting real epoch-ms integers back for the `<EPOCH_MS>` tokens) so the migrator has something to run against in tests.

This is an I/O-heavy + partial-recovery phase — exactly the profile where prior-phase deep review (Phases 4 & 6) caught silent-I/O-swallow and data-loss criticals that green TDD suites missed. Research surfaces where silent-failure risk concentrates (SQLite `Rows.Err()` after iteration, partial batch commits, `os.Rename` cross-device failure, `sql.NullString` scan of nullable aged columns) so the planner can write explicit error-handling acceptance criteria.

**Primary recommendation:** Add `modernc.org/sqlite v1.53.0` as a read-only migration reader in a new `internal/migrate` package; translate `nodes`/`edges`/`files`→`Node`/`Edge`/`File` verbatim-id (D-01), write through `graphstore.Writer` into a sibling temp store dir, persist a progress record under a new `m/migration` meta key, run the D-09 invariant pass, then atomic `os.Rename` the temp store into place and stamp `Meta.healthy=true`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01: Preserve TS node ids verbatim.** Copy `nodes.id`→`Node.id`; copy `edges.source`/`edges.target` unchanged. Do NOT recompute native `nodeid.NodeID` (TS uses 32-hex MD5-style digest; native uses `sha256(...)[:32]` — different value for same tuple; recompute would also be lossy vs. TS extractor divergences). Open question for researcher: a TS-id graph differs from what `index`/`sync` computes → later `sync` could see spurious churn. Decide (a) accept next full `index` reconciles, or (b) stamp migrated `Meta` so `sync`/`index` recognizes a TS-migrated graph. Prefer least-surprising, no-silent-corruption path.
- **D-02: Direct table→record mapping.** `nodes`→`Node`, `edges`→`Edge`, `files`→`File`, plus one `Meta`. Map every field the target proto defines. Edge `metadata` JSON → `Edge.metadata` (`map<string,string>`; flatten/stringify non-string values), plus `line`/`col`/`provenance`.
- **D-03: Drop (do not translate):** `nodes_fts*`, `name_segment_vocab`, `unresolved_refs`, SQLite internals (`sqlite_sequence`, `sqlite_stat1`, `schema_versions`). Map `project_metadata` into `Meta` where a field exists, else drop.
- **D-04: Build the `x/` file-owned secondary index during migration** via `Writer.PutEdge(e, ownerPath)`; set `Meta.has_file_index = true`.
- **D-05: Drop TS-only node attributes with no proto field** (`is_async`, `is_static`, `is_abstract`, `decorators`, `type_parameters`, `start_column`/`end_column`).
- **D-06: Batched idempotent writes + a progress record.** Bounded batches via `Writer.Commit()`. Persist a migration-progress record (source `schema_versions` max, target `SchemaVersion`, cursor = last-committed table + rowid, `status = in_progress|complete`) under a dedicated meta key. On resume, skip completed tables / resume mid-table by rowid. Writes keyed by preserved TS id → replaying a partial batch is an idempotent overwrite.
- **D-07: Atomic directory swap.** Write into a temp directory, rename into place on success. Stamp `Meta.healthy = true` only after validation (D-09) passes.
- **D-08: `codegraph migrate [--from <path>] [--to <path>] [--force]`**, wired via Phase-6 cobra/cli pattern in a new `internal/migrate` package. `--from` default = TS `.codegraph/` in cwd (auto-detect SQLite DB file inside). `--to` default = new-format `.codegraph/` (in place, temp→swap). Detect TS source by opening the SQLite file and checking for `schema_versions` + `nodes` + `edges` tables (not by filename). Non-destructive to source; refuse to overwrite a non-empty unrecognized target without `--force`.
- **D-09: Post-migration invariant pass, fail loud:** (1) count reconciliation (migrated counts == source row counts minus dropped tables); (2) referential integrity (every migrated edge's source/target resolves to a migrated node; dangling → fail loudly by default; `--drop-dangling` to drop-and-log); (3) readability/schema guard (reject unreadable/locked SQLite, missing required tables, source `schema_versions` outside supported range); (4) aged-DB tolerance (read columns defensively so an aged index missing later-added columns migrates instead of crashing).
- **D-10: `Meta.healthy` gates on all checks.** Success reported only after D-09 passes end-to-end.
- **Locked/carried:** add `modernc.org/sqlite` (pure-Go, CGo-free) as READ-ONLY migration reader; open DB read-only; NO new CGo (tree-sitter stays sole exception); read via `database/sql`, write exclusively through existing `graphstore.Writer`; new `internal/migrate` package; `migrate` subcommand on root cobra command.

### Claude's Discretion

- Exact batch size, progress-record key name, temp-directory naming, and reconciliation-report formatting are implementation details left to planning/execution.
- Whether per-file `node_count`/`edge_count` are carried from TS or recomputed during the write (either is acceptable if counts reconcile).

### Deferred Ideas (OUT OF SCOPE)

- Add proto fields for TS-only node attributes (`is_async`, `is_static`, `is_abstract`, `decorators`, `type_parameters`) — additive schema change, belongs with a downstream consumer, not migration.
- Teach `sync`/`index` to reconcile a TS-migrated graph's ids without a full churn pass — sync-engine concern, potentially Phase 8.
- Two-way / export-back-to-TS migration — explicitly out of scope; one-way only.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MIGR-01 | User can run a migration command converting an existing TS `.codegraph/` SQLite index to the new format in one step | `codegraph migrate` cobra command (§CLI Wiring); read path via `modernc.org/sqlite` (§Standard Stack); write via existing `graphstore.Writer` (§Writer Integration); field mapping table (§Field Mapping) |
| MIGR-02 | Migration is resumable, version-stamped, validated against real aged `.codegraph/` directories, and runs structural-invariant checks on the result | Progress record + resume (§Resumability); `SchemaVersion` stamping via `schema.NewMeta()`; invariant pass (§Validation Invariants); fixture reconstruction harness (§Fixture Question) |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Read TS SQLite schema | Migration reader (`internal/migrate`, `database/sql`+`modernc.org/sqlite`) | — | One-shot read-only path; isolated per PROJECT.md mandate |
| Translate records (SQLite row → proto) | `internal/migrate` (pure transform) | `internal/schema` (proto types) | Pure data transform of DB bytes; needs no source repo tree |
| Persist graph + `x/` index + Meta | `internal/graphstore` (existing `Writer`) | — | Reuse the sole storage door (D-04a boundary); migration must not bypass `GraphStore` |
| Resumability / progress cursor | `internal/migrate` (writes a `m/` meta record via `graphstore`) | `internal/graphstore/keys.go` (needs additive `migrationKey`) | Progress record is store-wide metadata; lives in the meta namespace |
| Atomic directory swap | `internal/migrate` (`os.Rename`) | `internal/cli` (path resolution) | Crash-safety is a filesystem concern above the store engine |
| Validation invariant pass | `internal/migrate` (uses `graphstore.Reader`) | — | Reads back the written store via the `Reader` node-existence API |
| CLI registration | `internal/cli` (`migrate.go`) | `internal/migrate` (business logic) | Mirrors every existing command: thin cobra wrapper delegating to a package |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `modernc.org/sqlite` | v1.53.0 | Pure-Go, CGo-free SQLite `database/sql` driver — the read-only migration reader ONLY | `[VERIFIED: pkg.go.dev — 3,518 importers, BSD-3-Clause, latest v1.53.0 published 2026-06-21, bundles SQLite 3.53.2]`. Mandated by PROJECT.md/CLAUDE.md ("migration reader only, isolated to a one-shot code path"); CGO_ENABLED=0 preserved. `[CITED: pkg.go.dev/modernc.org/sqlite]` |
| `database/sql` (stdlib) | — | Connection + query API against the driver | Standard, no dep |
| `internal/graphstore` (existing) | in-repo | Write path: `Open`, `Writer.PutNode/PutEdge/PutFile/PutMeta/Commit/Close` | The sole storage door (D-04a); reused verbatim |
| `internal/schema` (existing) | in-repo | `Node`/`Edge`/`File`/`Meta` proto types, `NewMeta()`, `SchemaVersion` | Target record shapes + version stamping |
| `github.com/spf13/cobra` | v1.10.2 (in go.mod) | `migrate` subcommand | Every existing command uses it |

### Installation
```bash
go get modernc.org/sqlite@v1.53.0
# Then manually promote to a direct require in go.mod per established project convention
# (do NOT run `go mod tidy` — it strips deliberately pre-pinned, still-unimported deps;
#  see STATE.md Phase-01/02/03/04/05 decisions — every dep add did a manual promote).
```

**Driver registration + open (verified pattern):**
```go
import (
    "database/sql"
    _ "modernc.org/sqlite"  // registers the "sqlite" driver name
)
// driver name is literally "sqlite" (NOT "sqlite3" — that's mattn/go-sqlite3)
db, err := sql.Open("sqlite", dsn)
```
`[CITED: pkg.go.dev/modernc.org/sqlite]` — "Connecting to a SQLite Database" snippet uses `sql.Open("sqlite", dsnURI)`.

**Read-only DSN (defense: never mutate the source, D-08 non-destructive):**
Use a URI DSN with `_pragma=query_only(1)` and `mode=ro`:
```go
dsn := "file:" + absPath + "?mode=ro&_pragma=query_only(1)&_txlock=deferred"
```
- `mode=ro` — SQLite URI filename open mode: read-only, fails if file cannot be opened read-only.
- `_pragma=query_only(1)` — modernc runs each `_pragma` value as a `PRAGMA ...` on connection; `query_only(1)` makes the connection reject all writes at the SQLite layer (constant `SQLITE_QueryOnly = 1048576` exists in the lib). This is belt-and-suspenders with `mode=ro`. `[CITED: pkg.go.dev/modernc.org/sqlite — Driver.Open _pragma query parameter]`
- Because the source is opened read-only, SQLite will NOT create a `-wal`/`-shm` sidecar or a journal in the source `.codegraph/` — critical for D-08's "never delete/mutate the source."

**CGO-free confirmation:** `[VERIFIED: pkg.go.dev]` modernc.org/sqlite is "a CGo-free port of SQLite3" — cross-compiles trivially, no C toolchain. Confirms the "no new CGo" lock (tree-sitter stays the sole exception). No build tags required for the common desktop/CI targets (darwin/linux/windows × amd64/arm64 are all supported pure-Go targets).

### Supporting / stdlib
| Library | Purpose | When to Use |
|---------|---------|-------------|
| `encoding/json` (stdlib) | Parse `edges.metadata` JSON object → flatten to `map[string]string` (D-02) | Every edge with non-NULL metadata |
| `os` (stdlib) | Temp dir create + `os.Rename` atomic swap (D-07) | Final commit step |
| `path/filepath` (stdlib) | `--from`/`--to` resolution, DB file autodetect inside `.codegraph/`, separator normalization | CLI + read path |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `modernc.org/sqlite` | `mattn/go-sqlite3` | Faster (CGo baseline) but requires CGO_ENABLED=1 — **rejected by PROJECT.md** for a one-shot read; would break the static-binary/no-CGo story |
| `_pragma=query_only(1)` + `mode=ro` | opening read-write and just not writing | Rejected — a crashed WAL replay or auto-vacuum could still mutate the source file; read-only is the only D-08-safe posture |
| Additive `migrationKey` in keys.go | overloading the existing `m/schema` Meta record | Progress cursor is transient and updated per-batch; keeping it in a distinct `m/migration` key avoids churning the real Meta record mid-run and lets validation cleanly distinguish "migration in progress" from "healthy graph" |

## Package Legitimacy Audit

> `modernc.org/sqlite` is a Go module; the `gsd-tools package-legitimacy check` seam supports only npm/pypi/crates, so verification was done directly against the Go module proxy + pkg.go.dev.

| Package | Registry | Age | Importers | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `modernc.org/sqlite` | Go proxy (`go list -m -versions` returns 130+ versions v1.0.0→v1.53.0) | ~6 yrs (v1.0.0 → v1.53.0) | 3,518 (pkg.go.dev "Imported by") | gitlab.com/cznic/sqlite (mirror github.com/modernc-org/sqlite) | OK | Approved — pin v1.53.0 |

`[VERIFIED: Go module proxy]` — `go list -m -versions modernc.org/sqlite` returns a continuous 6-year release history ending at v1.53.0; not a slopsquat. `[VERIFIED: pkg.go.dev]` — BSD-3-Clause, 3,518 importers, canonical repo gitlab.com/cznic/sqlite (the well-known `cznic`/Jan Mercl project, same author as the modernc.org toolchain). No postinstall/build-script vector (Go modules have none).

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

**Supply-chain / audit-impact note (project is supply-chain-conscious — DIST-03/DIST-05):** `modernc.org/sqlite` is a *large* pure-Go module — it pulls a transitive tail (`modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory`, and their deps). This materially enlarges the dependency tree and the SBOM. Recommendation for the planner: (1) run `govulncheck ./...` after adding it (call-graph-aware — will only flag reachable vulns, and the migrate path exercises a narrow read-only slice); (2) note in the go.mod / SBOM narrative that this dep is confined to `internal/migrate` and reachable only from the one-shot `migrate` command, not the hot indexing/query/serve paths; (3) the archtest boundary already prevents non-`graphstore` packages from importing pebble — consider whether a parallel guard should keep `modernc.org/sqlite` imports confined to `internal/migrate` so the read-only dep can't leak into runtime paths.

## Architecture Patterns

### System Architecture Diagram

```
                 codegraph migrate [--from][--to][--force][--drop-dangling]
                                        │
                                        ▼
                          internal/cli/migrate.go  (cobra: flags, path resolve)
                                        │  delegates
                                        ▼
        ┌──────────────────── internal/migrate.Run(from, to, opts) ───────────────────┐
        │                                                                              │
        │   1. DETECT      open <from>/.codegraph/<*.db> READ-ONLY                      │
        │      (D-08)      (mode=ro & query_only) → probe schema_versions+nodes+edges   │
        │                        │  (if tables absent → not a TS source, fail loud)    │
        │                        ▼                                                      │
        │   2. GUARD       read max(schema_versions.version); reject if out of          │
        │      (D-09.3)    supported range; PRAGMA table_info per table (D-09.4)         │
        │                        │                                                      │
        │                        ▼                                                      │
        │   3. TARGET      create sibling temp dir <to>/.codegraph.migrate-tmp-XXXX/    │
        │      (D-07)      graphstore.Open(tmp/store)                                    │
        │                        │                                                      │
        │                        ▼                                                      │
        │   4. RESUME?     read m/migration progress record from tmp store              │
        │      (D-06)      → skip completed tables / resume mid-table by rowid           │
        │                        │                                                      │
        │      ┌─────────────────┼──────────────────┬───────────────────┐              │
        │      ▼                 ▼                  ▼                   ▼               │
        │  read files      read nodes         read edges          (progress record     │
        │  → PutFile       → PutNode           → PutEdge(e,owner)    updated per batch)  │
        │      │                 │                  │                                    │
        │      └────── bounded batches → Writer.Commit() (idempotent overwrite) ────────│
        │                        │                                                      │
        │                        ▼                                                      │
        │   5. VALIDATE    count reconciliation + zero-dangling-edges (Reader.GetNode)  │
        │      (D-09)      + (optional --drop-dangling) → PutMeta(healthy per D-10)      │
        │                        │                                                      │
        │                        ▼                                                      │
        │   6. SWAP        store.Close(); os.Rename(tmp, <to>/.codegraph)  (D-07)       │
        │      atomic      (same-parent temp dir → same filesystem → atomic rename)     │
        └──────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
                     new-format .codegraph/  (Pebble store, healthy=true)
```

The migration is a **pure data transform of the SQLite bytes** — it does NOT require the source repo tree, because `kind`, `qualified_name`, and `file_path` are all columns in the TS DB. This is the whole value over re-indexing: fast, offline, no grammars.

### Recommended Project Structure
```
internal/migrate/
├── migrate.go        # Run(from, to string, opts Options) — orchestration
├── reader.go         # open TS DB read-only; table_info probing; row iterators (files/nodes/edges)
├── translate.go      # sqlRow → schema.Node/Edge/File; metadata JSON flatten; ms→ns
├── progress.go       # migration progress record: marshal/read under m/migration key
├── validate.go       # D-09 invariant pass (count reconcile, dangling-edge scan)
└── migrate_test.go   # table-driven tests + fixture reconstruction

internal/cli/
└── migrate.go        # newMigrateCmd() — cobra wrapper, registered in root.go
```
Progress-record persistence needs one **additive** helper in `internal/graphstore/keys.go` (a `migrationKey`/`m/migration` name) plus a `Writer.PutMigration`/`Reader.GetMigration` pair OR (simpler, no interface change) reuse the existing meta namespace by storing the progress bytes under a distinct meta name. See §Writer Integration for the exact fit decision.

### Pattern 1: Read-only, defensive column enumeration (D-09.4 aged-DB tolerance)
**What:** Before reading a table, query `PRAGMA table_info(<table>)` to learn which columns actually exist in *this* aged DB, then build the `SELECT` from the intersection of {columns the migrator wants} ∩ {columns present}. Missing later-added columns (`edges.provenance`, `nodes.return_type`, `unresolved_refs.*` — though unresolved_refs is dropped) map to a Go zero value instead of crashing the query.
**When to use:** Every table read in the migrator.
**Example:**
```go
// Source: SQLite PRAGMA docs (sqlite.org/pragma.html#pragma_table_info) [CITED]
func presentColumns(db *sql.DB, table string) (map[string]bool, error) {
    // table is a fixed code-controlled identifier (one of "nodes"/"edges"/"files"),
    // never caller data — safe to interpolate; table_info is not parameterizable.
    rows, err := db.Query("PRAGMA table_info(" + table + ")")
    if err != nil { return nil, fmt.Errorf("migrate: table_info(%s): %w", table, err) }
    defer rows.Close()
    cols := map[string]bool{}
    for rows.Next() {
        var cid int; var name, ctype string; var notnull, pk int; var dflt sql.NullString
        if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
            return nil, fmt.Errorf("migrate: scan table_info(%s): %w", table, err)
        }
        cols[name] = true
    }
    if err := rows.Err(); err != nil { // MUST check — iteration errors are silent otherwise
        return nil, fmt.Errorf("migrate: iterate table_info(%s): %w", table, err)
    }
    return cols, nil
}
```
`PRAGMA table_info(t)` returns one row per column: `(cid, name, type, notnull, dflt_value, pk)`. `[CITED: sqlite.org/pragma.html]`. A genuinely aged DB may predate `edges.provenance` (`DEFAULT NULL`), `nodes.return_type`, and the `unresolved_refs.file_path`/`language` columns (`DEFAULT ''` / `DEFAULT 'unknown'`) — the DDL shows these were added over time with `DEFAULT` clauses.

### Pattern 2: Nullable-column scanning with `sql.NullString`
**What:** TS `nodes` has nullable `docstring`, `signature`, `visibility`, `return_type`; `edges` has nullable `metadata`, `line`, `col`, `provenance`; `files` has nullable `errors`. Scanning a SQL NULL into a plain `string`/`int` panics. Use `sql.NullString`/`sql.NullInt64` and convert to the proto's zero value when `!Valid`.
**When to use:** Every nullable column.
**Example:**
```go
var docstring, signature, visibility, returnType sql.NullString
var startLine, endLine int64
err := rows.Scan(&id, &kind, &name, &qname, &filePath, &lang,
    &startLine, &endLine, &docstring, &signature, &visibility, &returnType, ...)
n := &schema.Node{
    Id: id, Kind: kind, Name: name, QualifiedName: qname, FilePath: filePath,
    Language: lang, StartLine: int32(startLine), EndLine: int32(endLine),
    Docstring: docstring.String,   // "" when NULL — proto3 zero value
    Signature: signature.String,
    Visibility: visibility.String,
    ReturnType: returnType.String,
}
```

### Anti-Patterns to Avoid
- **Reading `nodes_fts`, `nodes_fts_data`, `nodes_fts_idx`, `nodes_fts_docsize`, `nodes_fts_config` as data.** These are FTS5 virtual/shadow tables (D-03 drop list). Enumerating tables must exclude any table whose name starts with `nodes_fts` and the `sqlite_*` internals. Reading a shadow table's `block BLOB` as records would produce garbage.
- **Recomputing node ids via `nodeid.NodeID`.** Violates D-01 — produces a different value than TS and is lossy vs. TS extractor divergences.
- **Ignoring `rows.Err()` after a `for rows.Next()` loop.** `database/sql` reports iteration/network errors only via `Rows.Err()`, not the loop condition — a truncated read looks like a clean end-of-rows. This is exactly the silent-I/O class that deep review caught in Phases 4 & 6. Every read loop MUST check `rows.Err()`.
- **`os.Rename` from a `/tmp` temp dir into the target.** Cross-device (`EXDEV`, "invalid cross-device link") — see §Resumability. Temp dir MUST be a sibling of the final target (same parent → same filesystem).
- **Writing the source DB open read-write "just to be safe."** A WAL replay on open can mutate the source; D-08 requires the source stay byte-identical.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQLite file parsing | A custom SQLite page reader | `modernc.org/sqlite` + `database/sql` | Pure-Go, handles WAL/format/pragmas; hand-rolling is a multi-month rabbit hole |
| Graph persistence + `x/` index + atomic batch | Direct Pebble writes | existing `graphstore.Writer` | D-04a boundary; `PutEdge(e, ownerPath)` already populates the `x/` index and Meta plumbing exists |
| Atomic dir replace | `os.RemoveAll(dst)` then copy | temp dir + `os.Rename` (D-07) | RemoveAll+copy has a torn-state window; rename is atomic on same fs |
| Deterministic proto marshal | `proto.Marshal` | `graphstore`'s internal `deterministicMarshal` (already used by every `PutX`) | Edge.Metadata map field re-randomizes key order otherwise — the Writer already handles this transparently |
| JSON object flatten | Manual string scanning | `encoding/json` into `map[string]json.RawMessage` then stringify each value | Correct escaping/nesting handling |

**Key insight:** Almost nothing here is new machinery — the migrator is glue between a well-established SQL read (`database/sql`) and an existing, tested write path (`graphstore.Writer`). The genuinely new logic is (a) defensive column reads, (b) the resume cursor, and (c) the validation pass. Concentrate test effort there.

## Field Mapping (D-02, D-05)

Source DDL: `testdata/golden/ts-schema.sql`. Target: `internal/schema/graph.proto`. Confirmed against the real dump `testdata/golden/ts-schema.dump.sql`.

### `nodes` → `schema.Node`
| TS column (`nodes`) | Type / nullable | → proto field | Conversion |
|---------------------|-----------------|---------------|------------|
| `id` | TEXT PK (`<kind>:<32-hex>`) | `Node.id` (1) | Verbatim (D-01) |
| `kind` | TEXT | `Node.kind` (2) | Verbatim |
| `name` | TEXT | `Node.name` (3) | Verbatim |
| `qualified_name` | TEXT | `Node.qualified_name` (4) | Verbatim |
| `file_path` | TEXT | `Node.file_path` (5) | Verbatim (see Pitfall: separator normalization) |
| `language` | TEXT | `Node.language` (6) | Verbatim |
| `start_line` | INTEGER | `Node.start_line` (7) | int64→int32 |
| `end_line` | INTEGER | `Node.end_line` (8) | int64→int32 |
| `start_column` | INTEGER | `Node.start_col` (9) | int64→int32 **— note name mismatch: TS `start_column` → proto `start_col`. Carry it (proto has the field) — NOT a D-05 drop; D-05's "start_column/end_column if unmodeled" caveat resolves to CARRIED, since `start_col`/`end_col` (fields 9/10) exist.** |
| `end_column` | INTEGER | `Node.end_col` (10) | int64→int32 (same note) |
| `signature` | TEXT NULL | `Node.signature` (11) | `sql.NullString`→"" |
| `docstring` | TEXT NULL | `Node.docstring` (12) | `sql.NullString`→"" (see unistr note) |
| `visibility` | TEXT NULL | `Node.visibility` (13) | `sql.NullString`→"" |
| `is_exported` | INTEGER DEFAULT 0 | `Node.is_exported` (14) | int!=0 → bool |
| `return_type` | TEXT NULL (aged: may be absent) | `Node.return_type` (15) | `sql.NullString`→""; column-presence-guarded (D-09.4) |
| `updated_at` | INTEGER (epoch-ms) | — | **DROP** — no per-node timestamp in proto Node |
| `is_async` | INTEGER | — | **DROP (D-05)** |
| `is_static` | INTEGER | — | **DROP (D-05)** |
| `is_abstract` | INTEGER | — | **DROP (D-05)** |
| `decorators` | TEXT (JSON) | — | **DROP (D-05)** |
| `type_parameters` | TEXT (JSON) | — | **DROP (D-05)** |

> **Clarification for planner on the D-05 start/end-column line:** CONTEXT D-05 lists `start_column`/`end_column` "*if unmodeled*." They ARE modeled (`Node.start_col`=9, `Node.end_col`=10 in graph.proto). Therefore they are **CARRIED, not dropped.** Only `is_async`/`is_static`/`is_abstract`/`decorators`/`type_parameters` are the genuine unconditional drops. The planner should state this explicitly so a task doesn't drop position columns the proto defines.

### `edges` → `schema.Edge`
| TS column (`edges`) | Type / nullable | → proto field | Conversion |
|---------------------|-----------------|---------------|------------|
| `id` | INTEGER PK AUTOINCREMENT | — | DROP (rowid is the resume cursor, not persisted) |
| `source` | TEXT | `Edge.source` (1) | Verbatim TS id (D-01) |
| `target` | TEXT | `Edge.target` (2) | Verbatim TS id (D-01) |
| `kind` | TEXT | `Edge.kind` (3) | Verbatim |
| `line` | INTEGER NULL | `Edge.line` (4) | `sql.NullInt64`→int32 (0 when NULL) |
| `col` | INTEGER NULL | `Edge.col` (5) | `sql.NullInt64`→int32 (0 when NULL) |
| `provenance` | TEXT NULL (aged: may be absent) | `Edge.provenance` (6) | `sql.NullString`→""; column-presence-guarded |
| `metadata` | TEXT NULL (JSON object) | `Edge.metadata` (7, `map<string,string>`) | Parse JSON → flatten (see below) |

**`metadata` JSON → `map<string,string>` flatten strategy (D-02):**
```go
// metadata is a JSON object; values may be string/number/bool/null/array/object.
// Flatten each top-level value to a deterministic string.
var raw map[string]json.RawMessage
if md.Valid && md.String != "" {
    if err := json.Unmarshal([]byte(md.String), &raw); err != nil {
        return nil, fmt.Errorf("migrate: edge %d metadata parse: %w", rowid, err) // fail loud, don't swallow
    }
    m := make(map[string]string, len(raw))
    for k, v := range raw {
        var s string
        if err := json.Unmarshal(v, &s); err == nil {
            m[k] = s              // it was a JSON string → use the decoded value
        } else {
            m[k] = string(v)      // number/bool/null/array/object → canonical JSON text, deterministic
        }
    }
    e.Metadata = m
}
```
Rationale: a JSON *string* value should map to its unescaped content (`"foo"` → `foo`); any non-string value (`3`, `true`, `["a"]`) has no natural string form, so preserve its canonical JSON text. `json.RawMessage` gives a stable byte form (no re-marshal reordering for scalars). The existing `graphstore` `deterministicMarshal` handles map key ordering at persist time, so migration output stays byte-deterministic.

### `files` → `schema.File`
| TS column (`files`) | Type / nullable | → proto field | Conversion |
|---------------------|-----------------|---------------|------------|
| `path` | TEXT PK | `File.path` (1) | Verbatim |
| `content_hash` | TEXT | `File.content_hash` (2) | Verbatim (dump shows 64-hex = SHA-256 — matches proto's "MUST be SHA-256" contract) |
| `language` | TEXT | `File.language` (3) | Verbatim |
| `node_count` | INTEGER DEFAULT 0 | `File.node_count` (4) | Carry OR recompute (discretion); if carried, verify reconciliation |
| `modified_at` | INTEGER (epoch-ms) | `File.mtime_unix_ns` (7) | **ms → ns: multiply by 1e6** (`mtime_unix_ns = modified_at * 1_000_000`). See conversion note below. |
| `size` | INTEGER | `File.size_bytes` (8) | Verbatim int64 |
| `indexed_at` | INTEGER (epoch-ms) | — | DROP — no proto home |
| `errors` | TEXT NULL (JSON array) | `File.errors` (6, `repeated string`) | Parse JSON array of strings; NULL→empty |
| *(no TS column)* | — | `File.edge_count` (5) | Recompute during write (TS `files` has no `edge_count` column) OR leave 0 and reconcile from edge counts |

**EPOCH_MS → mtime_unix_ns conversion note:** TS timestamps are epoch **milliseconds** (the dump normalized them to `<EPOCH_MS>`; README confirms "13-digit epoch-ms integers, optionally with a decimal fraction e.g. `1783108606938.7051`"). The proto field is `mtime_unix_ns` (nanoseconds). Convert `ns = round(ms * 1e6)`. **Watch the fractional-ms case** — some rows carry a decimal fraction (sub-ms precision); scan as `float64` or parse the fraction, then `ns = int64(ms_float * 1e6)`. A naive `int64` scan of `1783108606938.7051` will fail or truncate. Since `files.modified_at` is declared `INTEGER NOT NULL`, the fractional form more likely appears in `schema_versions.applied_at`/`nodes.updated_at`; still, scan defensively (`sql.NullFloat64` or scan into `float64`) to avoid a hard parse error on an aged DB.

### `project_metadata` → `schema.Meta` (D-03)
`project_metadata(key, value, updated_at)` is a generic KV table. Map recognized keys into `Meta` where a field exists (e.g. counts), else drop. In practice `Meta` is best constructed fresh via `schema.NewMeta()` (stamps `SchemaVersion=1`) with `node_count`/`edge_count` set from the actual written counts (authoritative), `last_sync_unix_ms` from the source's newest timestamp or migration time, `has_file_index=true` (D-04), and `healthy` gated on D-09 (D-10). Treat `project_metadata` as informational only.

### Dropped tables (D-03)
`nodes_fts`, `nodes_fts_data`, `nodes_fts_idx`, `nodes_fts_docsize`, `nodes_fts_config` (FTS5 shadow), `name_segment_vocab`, `unresolved_refs`, `sqlite_sequence`, `sqlite_stat1`, `schema_versions` (read for the version guard, but not translated to records).

## Writer / x-index Integration (D-04)

`graphstore.Writer` contract (from `internal/graphstore/store.go` + `batch.go`) — reused verbatim, no interface changes needed for the core write:

```go
store, err := graphstore.Open(tmpStoreDir)   // creates if absent; returns GraphStore
w, err := store.NewWriter()                  // one batch per debounce/window; commit once
// per record:
w.PutNode(n)                                 // stages n/<id> + x/<filePath>/node entry (auto)
w.PutEdge(e, ownerPath)                      // stages e/<src>/<kind>/<dst> + x/<ownerPath>/edge entry
w.PutFile(f)                                 // stages f/<path>
w.PutMeta(m)                                 // stages m/schema
w.Commit()                                   // atomic; DO NOT reuse Writer after Commit
w.Close()                                    // release batch if abandoned before Commit (safe no-op after Commit)
```

**`ownerPath` for `PutEdge` (D-04, so `Meta.has_file_index=true` is truthful):** The `x/` index entry for an edge is keyed by the *owning file* — the file that owns the edge's **source** node. `graphstore.Import` already establishes the exact pattern the migrator must follow: track each node's `id → FilePath` as nodes are written, then supply `nodeFilePath[e.Source]` as `ownerPath` when writing edges (see `internal/graphstore/export.go` `importRecord`). **Ordering requirement:** write **nodes before edges** so every edge's source node's FilePath is already known. TS `edges.source` may reference a `file:<path>` pseudo-node (dump shows `file:cmd/weft/main.go` as an edge source for `contains` edges) — the migrator must handle edges whose source is a file-node vs. a symbol-node. For a `contains` edge whose source is `file:<path>`, the ownerPath is that path directly; for symbol-source edges, it's the source node's `file_path`. If a source id is not among the migrated nodes, `ownerPath` is unknown → this is the same condition as a dangling edge (D-09.2). Recommendation: pass `ownerPath=""` (PutEdge tolerates empty — skips the x/ entry) and let the dangling-edge validation catch/report it, rather than fabricating an owner.

**Meta write:** Use `schema.NewMeta()` then set counts + `HasFileIndex=true` + `Healthy` (only after D-09). Written via `w.PutMeta(m)` under the fixed `m/schema` key (`metaRecordName="schema"`, `PutMeta` hard-codes it).

**Batch/commit granularity (D-06 discretion):** Bounded batches — commit every N records (e.g. 5,000–10,000 Puts) rather than one giant batch (a full monorepo index could be millions of records → unbounded memory in one Pebble Batch). Each `Commit` is atomic and durable (`pebble.Sync`). After each `Commit`, write the updated progress record (below) in its *own* small committed batch so the cursor advances durably with the data. Because every write is keyed by the preserved TS id (D-01), re-committing a partially-applied batch on resume is an idempotent overwrite — same key, same bytes, no duplication.

**Progress-record keyspace fit (read `keys.go`):** The meta namespace (`prefixMeta='m'`) currently stores exactly one well-known name, `"schema"` (`metaRecordName`). `metaKey(name)` already length-prefix-encodes an arbitrary name, so a **second** meta record under a distinct name (e.g. `"migration"`) is representable with **zero key-format changes** — the namespace already supports it. However, the `Writer`/`Reader` *interfaces* only expose `PutMeta`/`GetMeta` for the single `"schema"` name. Two clean options for the planner:
- **Option A (additive interface, cleaner):** add `PutMigration([]byte)`/`GetMigration() ([]byte, error)` to the `Writer`/`Reader` interfaces + `pebbleWriter`/`pebbleReader`, keyed via a new unexported `migrationKey = metaKey("migration")`. Small, additive, keeps the progress record inside the D-04a boundary. The progress payload can itself be a small protobuf or JSON blob (source schema_version, target SchemaVersion, last table, last rowid, status).
- **Option B (no interface change):** serialize the progress into a throwaway `Meta`-shaped record — rejected, muddies the real Meta record and its `healthy` semantics.

Recommend **Option A**. It's a bounded, additive storage change consistent with the project's additive-only discipline and keeps validation able to distinguish "in_progress" from a healthy graph.

## Resumability (D-06, D-07)

**Progress/cursor record (D-06):** persist `{ source_schema_version, target_schema_version, last_table, last_rowid, status }` under the `m/migration` key after each committed data batch. On start, read it:
- absent / `status=complete` on a fresh temp dir → start clean (or, if `--to` already holds a *completed* migration, that's the "recognizably a prior migration" case D-08 allows overwriting without `--force`).
- `status=in_progress` → skip tables already fully written; for the partially-written table, resume from `last_rowid + 1` (`SELECT ... WHERE rowid > ? ORDER BY rowid`). `nodes` has a TEXT PK but SQLite still exposes an implicit `rowid` for the resume cursor; `edges` has an explicit INTEGER PK `id` (== rowid). `files` has a TEXT PK `path` — use its implicit `rowid`. Order every read by `rowid` so the cursor is monotonic.
- **Idempotency confirmation:** re-writing a row already written before the crash produces the identical key+bytes (D-01 preserved ids + deterministic marshal) → Pebble `Set` overwrites in place, no duplication, no count inflation. Verified against `batch.go` (`PutNode`/`PutEdge` are `batch.Set`, last-write-wins on a key).

**Atomic directory swap (D-07):** 
- Create the temp store as a **sibling of the final target** — e.g. if `--to` resolves to `<root>/.codegraph`, use `<root>/.codegraph.migrate-tmp-<pid-or-rand>/`. Same parent dir ⇒ same filesystem ⇒ `os.Rename` is atomic on Unix (single `rename(2)`). `[VERIFIED: WebSearch — Go os package + renameio docs]` "rename is atomic when source and destination are on the same filesystem."
- **Cross-device caveat (the landmine):** `os.Rename` across filesystems returns `EXDEV` ("invalid cross-device link"). A temp dir under `/tmp` (or `os.TempDir()`) is frequently a *different* mount than the repo → rename fails. **Mitigation: never use `os.TempDir()`; always place the temp dir beside the destination.** `[VERIFIED: WebSearch — google/renameio TempDir docs, golang/go#8914, go-cloud#3294 all document EXDEV]`. The `google/renameio` package's `TempDir()` helper exists precisely to solve "os.TempDir() resides on a different mount point" — the project doesn't need the dep, just the pattern (sibling temp dir).
- If `--to` already exists (in-place migration, the default): `os.Rename(tmp, dst)` **replaces** a file but **fails if `dst` is a non-empty directory** on most platforms. So the swap for a directory target is: (1) if `dst` exists, `os.Rename(dst, dst+".old")` first, (2) `os.Rename(tmp, dst)`, (3) `os.RemoveAll(dst+".old")`. Each rename is same-parent/atomic; step 3 cleanup failure is non-fatal but must be logged (not swallowed). The planner should specify this three-step directory swap explicitly and give it a rollback-on-failure acceptance criterion.
- Windows note: `os.Rename` onto an existing path is not guaranteed atomic on non-Unix (`[CITED: pkg.go.dev/os]`). v1's primary targets include Windows — flag this as a known-weaker guarantee; the temp+rename still bounds the torn-state window far better than remove-then-copy.

**Failure-mode explicit-error requirements (prior-phase lesson — I/O + partial recovery):** the planner must write acceptance criteria that these are surfaced, never swallowed: SQLite open failure, `Rows.Err()` after every read loop, `Writer.Commit()` error (abort, leave temp dir for resume), progress-record write failure (must abort — a lost cursor breaks resume), `os.Rename` `EXDEV`/error (abort, source untouched, temp dir preserved).

## Validation Invariants (D-09, MIGR-02)

Run after all data is written into the temp store, before the swap and before `Meta.healthy=true` (D-10):

1. **Count reconciliation (D-09.1):** `SELECT count(*)` on source `nodes`/`edges`/`files` (excluding dropped tables) vs. records written. Compare against the migrator's own write counters. Report the reconciliation (counts per table). A mismatch is a corruption signal → fail loud. Note: because the Pebble edge key omits line/col, two TS edges sharing `(source,kind,target)` but differing in `(line,col)` collapse to one stored edge (the TS `idx_edges_identity` unique index keeps `(source,target,kind,line,col)` distinct). **This means migrated edge count may legitimately be ≤ source edge count.** The reconciliation must treat edges with a `<=` tolerance (like Phase 3's edge-dedup tolerance) OR de-dup the source count by `(source,kind,target)` before comparing — otherwise a correct migration fails the check. **This is a real gotcha the planner must encode**, mirroring the documented Phase-1/2 edge-dedup decision (README #1034 note).
2. **Referential integrity / zero-dangling-edges (D-09.2):** every migrated edge's `source` and `target` must resolve to a migrated node. Use `Reader.GetNode(id)` (returns `graphstore.ErrNotFound` if absent) against a snapshot of the temp store. Dangling → fail loud with a report by default; `--drop-dangling` drops-and-logs (explicit opt-in lossy). **Caveat:** TS edges legitimately reference `file:<path>` pseudo-node sources (dump shows `contains` edges from `file:...`). If the migrator does NOT create file pseudo-nodes as `Node` records, those edge sources will be "dangling" by node-existence. Decide: either (a) synthesize `file:<path>` nodes, or (b) exempt `file:`-prefixed endpoints from the dangling check (they're structural, not symbol refs). Recommend (b) — exempt `file:` endpoints and document it — to avoid inventing node records TS didn't have as symbols. The planner must resolve this explicitly; it directly determines whether a faithful migration passes its own invariant.
3. **Readability / schema guard (D-09.3):** reject an unreadable/locked SQLite file (open error), missing required tables (D-08 detection: `schema_versions`+`nodes`+`edges` must exist), or a source `schema_versions` max version outside the documented supported range. The dump shows `schema_versions` rows `(1,...)` and `(7,...,'Initial schema includes all migrations')` — so v7 is a real observed version. Define the supported range as an explicit `[min,max]` constant (observed: 1..7) and fail loud outside it. Never silent.
4. **Aged-DB tolerance (D-09.4):** `PRAGMA table_info` per table (Pattern 1); build SELECT from present columns; absent later-added columns → zero value. Covered above.

**Gate `Meta.healthy` (D-10):** set `m.Healthy=true` (and a `health_message`) only after checks 1–4 pass end-to-end; otherwise `healthy=false` with a diagnostic message, and (per D-09) the command exits non-zero and does NOT perform the atomic swap (leaving the source and any prior target untouched).

## The Real-Aged-`.codegraph/` Fixture Question (MIGR-02 SC-3 — biggest open risk)

**Finding (verified by filesystem search):** **No committed aged SQLite `.db`/`.sqlite` fixture exists anywhere under `testdata/`.** `find testdata -iname '*.db' -o -iname '*.sqlite*'` returns nothing. What exists:
- `testdata/golden/ts-schema.sql` — the DDL only (`.schema` output; `CREATE TABLE`/`CREATE INDEX`/`CREATE TRIGGER`, no data).
- `testdata/golden/ts-schema.dump.sql` — a 17-line **representative, determinism-stripped** `.dump` excerpt: a handful of `INSERT INTO nodes/edges/files/schema_versions` rows with all epoch-ms timestamps replaced by the literal token `<EPOCH_MS>` and `unistr(...)`-wrapped docstrings.
- `testdata/golden/corpus/{weft-go,colbymchenry-codegraph}/` — these are **JSON tool-output fixtures only** (explore/query/callers/callees/impact/node/status `.json`), NOT SQLite indexes. The corpus source trees themselves are not committed. So there is **no aged `.db` accompanying the corpus.**
- `testdata/golden/capture.sh` — the harness that *produced* the DDL/dump by shelling out to the live TS `codegraph` CLI + `sqlite3`. It requires the live TS CLI (v1.3.1) on PATH, which per its own README is a "time-sensitive, one-shot capture" that "may be uninstalled in the future."

**Consequence for the planner:** MIGR-02 SC-3 ("validated against real aged `.codegraph/` directories") cannot rely on a committed real DB — none exists, and re-running `capture.sh` requires the possibly-gone TS CLI. The planner MUST specify a **fixture-reconstruction harness** that builds a real SQLite `.db` the migrator can run against, from the committed artifacts:

```bash
# Reconstruct a real SQLite DB from the committed DDL + dump for tests.
# 1. Substitute real epoch-ms integers back for the <EPOCH_MS> tokens
#    (any fixed valid ms value, e.g. 1700000000000) and decode/keep unistr(...) as-is
#    (SQLite understands unistr()).
sed 's/<EPOCH_MS>/1700000000000/g' testdata/golden/ts-schema.dump.sql > /tmp/seed.sql
# 2. Materialize a DB from schema + seed rows
sqlite3 testdata/golden/ts-index.db < testdata/golden/ts-schema.sql
sqlite3 testdata/golden/ts-index.db < /tmp/seed.sql
```
Better still, do the reconstruction **in Go inside the test** (no `sqlite3` binary dependency in CI) using the same `modernc.org/sqlite` driver the migrator uses: open a fresh DB read-write, `Exec` the DDL from `ts-schema.sql`, `Exec` the seed INSERTs (with `<EPOCH_MS>` substituted), close — then point the migrator at it read-only. This gives a hermetic, committed-artifact-derived fixture with zero external-tool dependency.

**Recommended layered validation fixtures the planner should specify:**
- **F1 — minimal reconstructed DB** from `ts-schema.dump.sql` (above): exercises the happy path + metadata + unistr docstrings + file-source `contains` edges.
- **F2 — aged DB** = F1's DDL with later-added columns removed (`ALTER TABLE nodes DROP COLUMN return_type`, drop `edges.provenance`, etc.) to prove D-09.4 defensive reads. (Or build a reduced DDL variant.)
- **F3 — corruption/dangling DB** = F1 plus an edge referencing a non-existent target id, to prove D-09.2 fails loud (and `--drop-dangling` drops it).
- **F4 — interrupted-run** = drive a migration, kill after the first batch commit, re-run, assert resume-to-completion with correct final counts (D-06).
- **Optional F5 — real full index** = if the TS CLI is still available on a dev machine, `capture.sh`-style produce a real large `.db` and run the migrator against it manually (not committed to CI — too large). Document this as the "real aged directory" spot-check for MIGR-02 SC-3.

**`unistr(...)` docstrings (from the dump):** TS encodes docstrings containing non-ASCII/newlines as `unistr('...
...')` in the `.dump` output. Two things to know: (1) `unistr()` is a SQLite function that decodes `\uXXXX` escapes — it only appears in the **textual dump**, not in the stored data. When the migrator reads via `database/sql` `SELECT docstring FROM nodes`, the **driver returns the already-decoded UTF-8 string** (the real column value), NOT the `unistr(...)` wrapper. So the migrator sees clean text and needs no unistr handling. (2) The wrapper only matters for the **fixture reconstruction** — `sqlite3 db < dump.sql` (or `Exec`) will evaluate `unistr(...)` correctly because it's valid SQLite. No special handling in migration code; a note for whoever writes the fixture seeder.

## Common Pitfalls

### Pitfall 1: Cross-device `os.Rename` (EXDEV)
**What goes wrong:** Temp store in `/tmp`, rename into repo → "invalid cross-device link", swap fails after a fully successful migration.
**Why:** `/tmp` is often a separate mount (tmpfs/overlay) from the repo filesystem.
**How to avoid:** Temp dir MUST be a sibling of `--to` (same parent → same fs). Never `os.TempDir()`.
**Warning signs:** Migration "completes" then errors at the final step; works locally (same fs) but fails in Docker/CI (volume mounts).

### Pitfall 2: Silent `Rows.Err()` after read loops
**What goes wrong:** A truncated/failed SQLite read looks like a clean end-of-rows; migrated graph is silently short.
**Why:** `database/sql` reports iteration errors only via `Rows.Err()`, not `rows.Next()`.
**How to avoid:** `if err := rows.Err(); err != nil { return ... }` after **every** loop. Make it an acceptance criterion.
**Warning signs:** count reconciliation (D-09.1) flags a shortfall — which is exactly why D-09.1 exists; but the read loop should fail loud first.

### Pitfall 3: FTS5 shadow tables read as data
**What goes wrong:** Enumerating all tables and reading `nodes_fts_data(block BLOB)` as records → garbage nodes or a crash.
**Why:** FTS5 creates `nodes_fts`, `nodes_fts_data`, `nodes_fts_idx`, `nodes_fts_docsize`, `nodes_fts_config` shadow tables.
**How to avoid:** Read from an explicit allow-list of tables (`nodes`, `edges`, `files`) — never "enumerate + read all." D-03 drops everything else.

### Pitfall 4: `file:`-source edges look dangling
**What goes wrong:** TS `contains` edges have `file:<path>` sources; if no `file:` node records exist, D-09.2 flags them as dangling and the migration fails its own validation.
**Why:** TS models files as edge endpoints; the new schema stores files as `File` records, not `Node`s.
**How to avoid:** Exempt `file:`-prefixed edge endpoints from the dangling check (recommended), OR synthesize file pseudo-nodes. Decide explicitly.

### Pitfall 5: NULL scan panics on aged nullable columns
**What goes wrong:** Scanning NULL `docstring`/`signature`/`visibility`/`metadata`/`line`/`col`/`provenance` into plain Go types panics.
**How to avoid:** `sql.NullString`/`sql.NullInt64`/`sql.NullFloat64` for every nullable column; convert `!Valid`→proto zero value.

### Pitfall 6: `file_path` separator / relative-vs-absolute mismatch
**What goes wrong:** If TS stored absolute or backslash paths and the new format expects repo-relative forward-slash paths, node/file/x-index keys won't line up with what `sync`/`query` later expect.
**Why:** The `x/` file index and file records key by path; a separator or root mismatch silently fragments the graph.
**How to avoid:** Inspect the dump — TS `file_path` values are repo-relative, forward-slash (`plugin/skills/...`, `internal/plan/plan.go`, `cmd/weft/main.go`), matching the new format's relPath convention. So **verbatim carry is correct for the captured corpus.** But add a defensive normalization pass (`filepath.ToSlash`, strip any leading `./` or absolute prefix) and document the assumption; a Windows-origin TS index could carry backslashes.
**Warning signs:** post-migration `sync` re-indexes everything (path keys don't match) — ties into the D-01 open question below.

### Pitfall 7: `edge_count` has no TS source column
**What goes wrong:** `File.edge_count` (proto field 5) has no corresponding `files` column in TS (only `node_count` exists). Carrying blindly leaves it 0.
**How to avoid:** Either recompute per-file edge counts during the write (count edges whose owner is that file) or accept 0 and note it. Reconciliation (D-09.1) is on totals, so 0 per-file edge_count won't fail the total check — but `status` output would under-report. Recommend recompute.

### Pitfall 8: The D-01 id-scheme open question (sync churn)
**What goes wrong:** A migrated graph carries TS 32-hex ids; a later `codegraph sync`/`index` computes native `sha256[:32]` ids for the same symbols → every node looks "new," triggering a full-churn reconcile (spurious re-index of the whole repo on first sync).
**Why:** D-01 deliberately preserves TS ids (faithful migration) but native id computation differs.
**How to avoid (CONTEXT open question — researcher recommendation):** Prefer the **least-surprising, no-silent-corruption path**: since `sync` already has a `needsFileIndexBackfill`/full-reindex fallback (`internal/indexer/sync.go`) and `Meta.has_file_index` gating, the cleanest v1 answer is **(a) accept that the next full `index` reconciles** — but make it non-silent by setting a distinguishing marker so behavior is predictable. Options for the marker within the *current* schema (no proto change): set `Meta.health_message` to note "migrated-from-ts (schema v7)" and document that the first `sync` after a migration will do a full re-index (the ids won't match, so incremental diffing can't apply). A cleaner future fix (teach `sync` to recognize a TS-migrated graph) is explicitly **Deferred** (out of scope, possibly Phase 8). The planner should (1) document this behavior in the migrate command's help/output ("after migrating, the first `sync` performs a full re-index"), and (2) NOT attempt to make ids match — that would violate D-01 and be lossy. Flag for user confirmation whether option (a) accept-and-document is acceptable vs. option (b) add a `Meta` flag now.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| CGo `mattn/go-sqlite3` for any SQLite | Pure-Go `modernc.org/sqlite` | Mature since ~2021, v1.53.0 now | CGO_ENABLED=0 preserved; the whole reason this dep is acceptable per PROJECT.md |
| `os.TempDir()` + copy for atomic replace | Sibling temp dir + `os.Rename` (or `google/renameio`) | Long-standard | Avoids EXDEV; atomic on same fs |

**Deprecated/outdated:** none relevant. modernc.org/sqlite is actively maintained (v1.53.0 published 2026-06-21).

## Validation Architecture

> `workflow.nyquist_validation` is `true` in `.planning/config.json` — this section is required.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no testify in graphstore/cli; table-driven `t.Run` is the house style) |
| Config file | none — `go test` |
| Quick run command | `go test ./internal/migrate/ -run <name> -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MIGR-01 | One command converts TS SQLite → new format | integration | `go test ./internal/migrate/ -run TestMigrate_HappyPath -count=1` | ❌ Wave 0 |
| MIGR-01 | Node/Edge/File field mapping correct (verbatim ids, ms→ns, metadata flatten) | unit | `go test ./internal/migrate/ -run TestTranslate -count=1` | ❌ Wave 0 |
| MIGR-01 | Read-only: source DB byte-identical after migration | integration | `go test ./internal/migrate/ -run TestMigrate_SourceUnmodified -count=1` | ❌ Wave 0 |
| MIGR-02 | Resumable after interruption (kill mid-run, resume, correct counts) | integration | `go test ./internal/migrate/ -run TestMigrate_Resume -count=1` | ❌ Wave 0 |
| MIGR-02 | Version stamped (`Meta.SchemaVersion`, migration progress record) | unit | `go test ./internal/migrate/ -run TestProgressRecord -count=1` | ❌ Wave 0 |
| MIGR-02 | Count reconciliation fails loud on mismatch | unit | `go test ./internal/migrate/ -run TestValidate_CountReconcile -count=1` | ❌ Wave 0 |
| MIGR-02 | Zero-dangling-edges; `--drop-dangling` opt-in | unit | `go test ./internal/migrate/ -run TestValidate_Dangling -count=1` | ❌ Wave 0 |
| MIGR-02 | Aged-DB tolerance (missing later columns migrates) | integration | `go test ./internal/migrate/ -run TestMigrate_AgedDB -count=1` | ❌ Wave 0 |
| MIGR-02 | Schema-version guard rejects out-of-range | unit | `go test ./internal/migrate/ -run TestValidate_SchemaGuard -count=1` | ❌ Wave 0 |
| MIGR-01 | `codegraph migrate` CLI wiring (flags, detection, non-destructive) | integration | `go test ./internal/cli/ -run TestMigrateCmd -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/migrate/ -run <task-scope> -count=1`
- **Per wave merge:** `go test ./internal/migrate/ ./internal/cli/ -count=1 -race`
- **Phase gate:** `go test ./... -count=1` green before `/gsd-verify-work`. **Known flake (carry-forward):** `internal/daemon` `TestSoak`/flush-lock tests flake under full-suite parallel load; this phase touches no daemon code — if `./...` fails there, re-run `go test ./internal/daemon/ -count=1` isolated before treating it as a regression.

### Wave 0 Gaps
- [ ] `internal/migrate/migrate_test.go` — happy path, resume, aged-DB, source-unmodified (covers MIGR-01/02)
- [ ] `internal/migrate/translate_test.go` — field mapping, metadata flatten, ms→ns
- [ ] `internal/migrate/validate_test.go` — count reconcile, dangling, schema guard
- [ ] `internal/migrate/testdata/` fixture seeder — reconstruct a real SQLite `.db` in-Go from `ts-schema.sql` + `ts-schema.dump.sql` (F1–F4 above); the fixture harness IS a deliverable, not just a test helper
- [ ] `internal/cli/migrate_test.go` — command registration, flag parsing, TS-source detection, non-destructive refusal without `--force`

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1` in config — required.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Local CLI, no auth surface |
| V3 Session Management | no | — |
| V4 Access Control | no | Local filesystem only |
| V5 Input Validation | yes | The SQLite file is untrusted input (arbitrary user's aged index). Validate: (1) open read-only so a malicious DB can't be mutated; (2) `PRAGMA table_info` allow-list of table names — never interpolate a table name derived from DB contents; (3) parse `edges.metadata` JSON with a bounded expectation (map of scalars), fail loud on malformed; (4) treat `file_path`/`id` values as data — they already flow through `keys.go`'s length-prefixed `appendSegment` (T-01-02 mitigation), so a crafted path/id cannot forge a key into another namespace |
| V6 Cryptography | no (read side) | `content_hash` is carried verbatim (already SHA-256, 64-hex in dump); migration does not compute hashes |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malicious/corrupt SQLite file crashes or mutates | Tampering / DoS | Open read-only (`mode=ro`+`query_only`); pure-Go driver (no C memory-safety tail-risk); fail loud on unreadable/locked |
| Crafted `file_path`/`id` forges a Pebble key | Tampering | Existing `keys.go` length-prefixed segments (already mitigated at the storage layer) |
| Source-DB write via WAL replay | Tampering (source integrity, D-08) | Read-only open prevents WAL/journal sidecar creation |
| Path traversal via `file_path` in x-index | Tampering | Paths are keys, not filesystem operations — migration never opens files by `file_path` (pure DB→DB transform); normalize but don't dereference |
| Table-name injection via `PRAGMA table_info` | Injection | Table names are a fixed code constant allow-list, never DB-derived |

## Sources

### Primary (HIGH confidence)
- `internal/schema/graph.proto`, `internal/schema/meta.go` (repo) — target record shapes, `SchemaVersion`, `NewMeta()`; field numbers verified
- `internal/graphstore/store.go`, `batch.go`, `keys.go`, `pebble_store.go`, `export.go` (repo) — Writer/Reader contract, `Open`, `PutEdge(ownerPath)`, meta namespace, Import ownerPath pattern
- `testdata/golden/ts-schema.sql`, `ts-schema.dump.sql`, `README.md`, `ts-version.txt` (repo) — TS DDL, real id/timestamp/unistr shapes, corpus provenance
- `internal/cli/root.go`, `index.go`, `init.go`, `uninit.go` (repo) — cobra registration + flag pattern, `targetRoot`, `confirm`, dir layout constants
- `internal/indexer/sync.go` (repo) — `Meta.HasFileIndex` gating + full-reindex fallback (D-01 open-question basis)

### Secondary (MEDIUM confidence)
- `[CITED: pkg.go.dev/modernc.org/sqlite]` (Context7 `/websites/pkg_go_dev_modernc_org_sqlite`, High reputation) — `sql.Open("sqlite", dsn)`, `_pragma` DSN param, `query_only`, read-only, CGo-free
- `[VERIFIED: pkg.go.dev]` (WebFetch) — modernc.org/sqlite v1.53.0, BSD-3-Clause, 3,518 importers, gitlab.com/cznic/sqlite, SQLite 3.53.2
- `[VERIFIED: Go module proxy]` `go list -m -versions modernc.org/sqlite` — 6-year release history, legitimate package
- `[CITED: sqlite.org/pragma.html]` (WebSearch) — `PRAGMA table_info` column shape, defensive schema detection

### Tertiary (LOW confidence)
- `[CITED: WebSearch]` google/renameio, golang/go#8914, go-cloud#3294 — `os.Rename` EXDEV cross-device caveat + sibling-temp-dir pattern (corroborated across multiple independent sources → treat as effectively MEDIUM)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `modernc.org/sqlite` driver returns already-decoded UTF-8 for `unistr(...)`-encoded docstrings (wrapper is dump-only) | §Fixture Question | LOW — if wrong, docstrings carry literal `unistr(...)` text; caught by a translate unit test on F1 |
| A2 | TS `file_path` values are repo-relative forward-slash (verbatim carry safe) | §Pitfall 6 | MEDIUM — a Windows-origin TS index could carry backslashes; mitigated by a defensive `filepath.ToSlash` normalization the planner should still add |
| A3 | Supported source `schema_versions` range is 1..7 (dump shows v7 "includes all migrations") | §Validation D-09.3 | MEDIUM — if TS shipped a v8+ before the corpus was captured, the guard would wrongly reject; verify against live TS if available, else make the max a easily-bumped constant |
| A4 | `os.Rename` onto an existing non-empty directory fails → three-step swap needed | §Resumability | LOW — standard POSIX behavior; the three-step swap is safe regardless |
| A5 | Progress record fits the meta namespace via an additive `PutMigration`/`GetMigration` (Option A) | §Writer Integration | LOW — additive interface change, no format break; if rejected, Option B fallback exists |
| A6 | The D-01 open question resolves to "accept next full index reconciles + document" (option a) | §Pitfall 8 | MEDIUM — needs user confirmation (see Open Questions); alternative is adding a Meta flag now |

## Open Questions

1. **D-01 id-scheme sync churn (CONTEXT's explicit open question):**
   - What we know: migrated graph carries TS ids; native id computation differs → first `sync` after migration full-reindexes.
   - What's unclear: whether "accept + document" (no code change beyond a `health_message` note) is acceptable to the user, or whether a distinguishing `Meta` marker should be added now.
   - Recommendation: accept + document for v1 (least-surprising, no silent corruption, no D-01 violation); defer a `sync`-reconciliation feature to Phase 8. **Flag for user confirmation in discuss/plan.**
2. **`file:`-source edges in the dangling check (D-09.2):**
   - What we know: TS `contains` edges have `file:<path>` sources; these aren't symbol nodes.
   - What's unclear: exempt `file:` endpoints vs. synthesize file pseudo-nodes.
   - Recommendation: exempt `file:`-prefixed endpoints (don't invent node records) and document it; the planner must lock this since it decides whether a faithful migration passes its own invariant.
3. **Edge count reconciliation tolerance (D-09.1):**
   - What we know: Pebble edge key omits line/col → source edges can legitimately collapse (migrated ≤ source).
   - Recommendation: de-dup source count by `(source,kind,target)` before comparing, OR use `<=` tolerance; planner must encode this so a correct migration doesn't fail its own check.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test | ✓ | go1.26.5 (go.mod `go 1.26.5`) | — |
| `modernc.org/sqlite` | read path | ✗ (not yet in go.mod) | add v1.53.0 | none — required; pure-Go, adds via `go get` |
| `sqlite3` CLI | fixture reconstruction (optional) | ? | 3.54.0 observed at capture time | reconstruct in-Go via `modernc.org/sqlite` (no external binary) — recommended |
| Live TS `codegraph` CLI | real full-index spot-check (F5, optional) | ✗ likely gone | 1.3.1 at capture | committed DDL+dump reconstruction (F1–F4) is the CI path |

**Missing dependencies with no fallback:** `modernc.org/sqlite` — but it's a trivial `go get` (pure-Go).
**Missing dependencies with fallback:** `sqlite3` CLI and live TS CLI — both replaceable by in-Go fixture reconstruction from committed artifacts.

## Metadata

**Confidence breakdown:**
- Standard stack (`modernc.org/sqlite`, read-only DSN, CGo-free): HIGH — verified against pkg.go.dev, Context7, and Go module proxy
- Field mapping: HIGH — read both source DDL and target proto directly; every field accounted for
- Writer integration: HIGH — read the actual `store.go`/`batch.go`/`keys.go`/`export.go`; `Import`'s ownerPath pattern is the exact template
- Resumability + atomic swap: HIGH (mechanism) / MEDIUM (Windows atomicity caveat) — verified EXDEV + same-fs atomicity across multiple sources
- Validation invariants: HIGH — grounded in the Reader API and the observed edge-dedup/`file:`-source realities
- Fixture question: HIGH — verified by filesystem search that no `.db` exists; reconstruction path is concrete
- Pitfalls: HIGH — grounded in actual dump contents + prior-phase silent-I/O lessons

**Research date:** 2026-07-12
**Valid until:** 2026-08-11 (30 days — stable domain; re-verify `modernc.org/sqlite` latest version at plan time)

## RESEARCH COMPLETE
