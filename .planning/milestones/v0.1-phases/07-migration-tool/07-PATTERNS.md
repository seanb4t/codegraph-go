# Phase 7: Migration Tool - Pattern Map

**Mapped:** 2026-07-12
**Files analyzed:** 8
**Analogs found:** 7 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/migrate/reader.go` | service (read-only DB reader) | batch/streaming read | `internal/graphstore/export.go` (`exportNamespace`, `Export`) — same "stream rows, check `rows.Err()`/`iter.Error()`, allow-listed namespace" shape | role-match (new tech, `database/sql` vs Pebble iterator, but identical control-flow shape) |
| `internal/migrate/translate.go` | transform (SQL row → proto) | transform | `internal/graphstore/export.go` `importRecord` (kind-switch unmarshal→PutX) + `internal/schema/graph.proto` field defs | role-match |
| `internal/migrate/migrate.go` | service (orchestration) | batch/event-driven (resumable) | `internal/graphstore/export.go` `Import` (drives a Writer batch to completion, tracks `nodeFilePath` for ownerPath) | exact — `Import`'s node-then-edge-with-ownerPath loop is the literal template for D-04 |
| `internal/migrate/progress.go` | model + store integration (progress record) | CRUD (small record) | `internal/graphstore/batch.go` `PutMeta`/`internal/schema/meta.go` `NewMeta` + `internal/graphstore/keys.go` `metaKey` | role-match (additive sibling of the existing single-name meta record) |
| `internal/migrate/validate.go` | service (invariant checks) | batch/transform (read-back + reconcile) | `internal/graphstore/store.go` `Reader` interface (`GetNode`, `IterateNodes`/`IterateEdges`) — used read-only, no direct analog file but the interface itself is the pattern | role-match |
| `internal/migrate/swap.go` | utility (atomic dir replace) | file-I/O | `internal/upgrade/swap.go` (`atomicSwap`, `swapWindows`, `checkWritable`) | exact — same temp-then-rename-with-Windows-fallback shape, just directory instead of single file |
| `internal/cli/migrate.go` | controller (cobra command) | request-response | `internal/cli/sync.go` / `internal/cli/index.go` (`newSyncCmd`/`newIndexCmd`) | exact |
| `internal/migrate/migrate_test.go` + fixtures | test | batch (fixture reconstruction + round-trip) | `internal/graphstore/export_test.go`-style round-trip tests (not read directly this pass — confirm exact test file name before use; see Open Items) + `testdata/golden/ts-schema.sql` / `ts-schema.dump.sql` | partial — no committed aged `.db` fixture exists; harness must be built new (RESEARCH §Fixture Question) |

## Pattern Assignments

### `internal/migrate/reader.go` (service, batch read)

**Analog:** `internal/graphstore/export.go` — `Export` / `exportNamespace`

**Core streaming pattern to copy** (`internal/graphstore/export.go` lines ~92-115, `exportNamespace`):
```go
func exportNamespace(bw *bufio.Writer, snap *pebble.Snapshot, ns byte, kind uint8, newMsg func() proto.Message) error {
	lower := []byte{ns}
	upper := rangeUpperBound(lower)
	iter, err := snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return err
	}
	defer iter.Close()

	for ok := iter.First(); ok; ok = iter.Next() {
		msg := newMsg()
		if err := proto.Unmarshal(iter.Value(), msg); err != nil {
			return err
		}
		if err := writeExportRecord(bw, kind, msg); err != nil {
			return err
		}
	}
	return iter.Error()   // <-- the "check the error after the loop" idiom to replicate as rows.Err()
}
```
**Translate to `database/sql`:** replace `iter.First()/iter.Next()` with `rows.Next()`, replace `iter.Error()` with `rows.Err()` (mandatory per RESEARCH Pitfall 2 — every read loop MUST check `rows.Err()` after the loop, exactly mirroring this `iter.Error()` check). Replace the Pebble prefix-range scan with an explicit allow-listed `SELECT ... FROM nodes|edges|files` (never "enumerate all tables" — Pitfall 3, FTS5 shadow tables).

**Defensive column enumeration** (new pattern, no direct in-repo analog — from RESEARCH Pattern 1, `PRAGMA table_info`):
```go
func presentColumns(db *sql.DB, table string) (map[string]bool, error) {
    rows, err := db.Query("PRAGMA table_info(" + table + ")") // table is a fixed code-controlled identifier — never caller/DB data
    ...
    if err := rows.Err(); err != nil { return nil, fmt.Errorf("migrate: iterate table_info(%s): %w", table, err) }
    return cols, nil
}
```

**Read-only DSN pattern** (RESEARCH §Standard Stack, verified against pkg.go.dev):
```go
import (
    "database/sql"
    _ "modernc.org/sqlite" // driver name is "sqlite", NOT "sqlite3"
)
dsn := "file:" + absPath + "?mode=ro&_pragma=query_only(1)&_txlock=deferred"
db, err := sql.Open("sqlite", dsn)
```

**Error handling pattern:** every error wrapped with a `"migrate: <verb>: %w"` prefix (matches `export.go`'s `fmt.Errorf("export meta: %w", err)` style) — adopt `"migrate: <verb>: %w"` as the package-wide error-wrap convention.

---

### `internal/migrate/translate.go` (transform)

**Analog:** `internal/graphstore/export.go` `importRecord` (kind-switch shape) + field tables in `07-RESEARCH.md` §Field Mapping (authoritative — do not re-derive from the proto alone, the RESEARCH doc already reconciled every TS column against `internal/schema/graph.proto` field numbers).

**Nullable-column scan pattern** (RESEARCH Pattern 2 — copy verbatim, no in-repo analog since graphstore never reads external SQL):
```go
var docstring, signature, visibility, returnType sql.NullString
err := rows.Scan(&id, &kind, &name, &qname, &filePath, &lang,
    &startLine, &endLine, &docstring, &signature, &visibility, &returnType, ...)
n := &schema.Node{
    Id: id, Kind: kind, Name: name, QualifiedName: qname, FilePath: filePath,
    Docstring: docstring.String,   // "" when NULL — proto3 zero value
    ...
}
```

**`schema.Node`/`Edge`/`File` target struct fields** — see `internal/schema/graph.proto`: `Node` fields 1-15 (`id, kind, name, qualified_name, file_path, language, start_line, end_line, start_col, end_col, signature, docstring, visibility, is_exported, return_type`, reserved 50-59); confirm `Edge`/`File` field numbers directly in `graph.proto` (`source, target, kind, line, col, provenance, metadata` for Edge; `path, content_hash, language, node_count, edge_count, errors, mtime_unix_ns, size_bytes` for File — per RESEARCH field-mapping tables, cross-checked against the proto source at plan time since this pass did not re-read the Edge/File message bodies verbatim).

**metadata JSON→map flatten** (RESEARCH §Field Mapping, `edges` table — copy verbatim):
```go
var raw map[string]json.RawMessage
if md.Valid && md.String != "" {
    if err := json.Unmarshal([]byte(md.String), &raw); err != nil {
        return nil, fmt.Errorf("migrate: edge %d metadata parse: %w", rowid, err)
    }
    m := make(map[string]string, len(raw))
    for k, v := range raw {
        var s string
        if err := json.Unmarshal(v, &s); err == nil {
            m[k] = s
        } else {
            m[k] = string(v)
        }
    }
    e.Metadata = m
}
```

---

### `internal/migrate/migrate.go` (orchestration)

**Analog:** `internal/graphstore/export.go` — `Import`

**Core pattern to copy — the nodes-before-edges + ownerPath tracking loop** (verbatim template per RESEARCH; this is D-04's exact mechanism):
```go
func Import(dst GraphStore, r io.Reader) error {
	w, err := dst.NewWriter()
	if err != nil {
		return err
	}

	nodeFilePath := make(map[string]string)
	// ... stream records ...
	if err := importRecord(w, kind, data, nodeFilePath); err != nil {
		return err
	}
	// ...
	return w.Commit()
}

func importRecord(w Writer, kind uint8, data []byte, nodeFilePath map[string]string) error {
	switch kind {
	case exportKindNode:
		var n schema.Node
		proto.Unmarshal(data, &n)
		nodeFilePath[n.GetId()] = n.GetFilePath()   // <-- track id->FilePath as nodes stream past
		w.PutNode(&n)
	case exportKindEdge:
		var e schema.Edge
		proto.Unmarshal(data, &e)
		w.PutEdge(&e, nodeFilePath[e.GetSource()])  // <-- exact D-04 ownerPath lookup pattern
	}
	return nil
}
```
**Migration-specific deviation from this template (per D-04/CONTEXT + RESEARCH §Writer Integration):** TS `edges.source` may be a `file:<path>` pseudo-node id, not a migrated symbol node — for those, `nodeFilePath[e.Source]` will miss; pass `ownerPath=""` in that case (PutEdge tolerates empty — skips the x/ entry) and let the D-09.2 dangling-edge validation exempt `file:`-prefixed endpoints rather than treat them as corruption (RESEARCH Pitfall 4).

**Batching:** bounded batches via repeated `NewWriter()`/`Commit()` cycles (not one giant Commit) — mirrors `internal/indexer`'s own debounce-window batching philosophy referenced in CLAUDE.md ("batch incremental updates ... into a single Pebble Batch/IndexedBatch commit").

---

### `internal/migrate/progress.go` (progress record)

**Analog:** `internal/graphstore/batch.go` `PutMeta` + `internal/graphstore/keys.go` `metaKey` + `internal/schema/meta.go` `NewMeta`

**Existing meta-record pattern to extend additively** (`internal/graphstore/batch.go`):
```go
func (w *pebbleWriter) PutMeta(m *schema.Meta) error {
	data, err := deterministicMarshal(m)
	if err != nil {
		return err
	}
	return w.batch.Set(metaKey(metaRecordName), data, nil)
}
```
**`metaKey` already supports an arbitrary second name with zero format changes** (`internal/graphstore/keys.go`):
```go
func metaKey(name string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(name))
	buf = append(buf, prefixMeta)
	buf = appendSegment(buf, name)
	return buf
}
```
**Recommended additive change (RESEARCH Option A):** add `PutMigration([]byte)`/`GetMigration() ([]byte, error)` to the `Writer`/`Reader` interfaces in `internal/graphstore/store.go`, backed by `pebbleWriter`/`pebbleReader` implementations that call `metaKey("migration")` — same shape as `PutMeta`/`GetMeta`, just a second fixed name. This is a `graphstore` package change, not a `migrate`-package-only file; note it in the plan as touching `internal/graphstore/batch.go`, `store.go`, and whatever `pebbleReader`'s `GetMeta` analog file is (not re-read this pass — locate via `grep -n "func.*GetMeta" internal/graphstore/*.go` at plan time).

**`schema.NewMeta()` version-stamping pattern** (`internal/schema/meta.go`):
```go
func NewMeta() *Meta {
	return &Meta{SchemaVersion: SchemaVersion}
}
```
Migration's final `Meta` write should start from `schema.NewMeta()` then set `HasFileIndex=true` (D-04), counts, and `Healthy` only after D-09 passes (D-10) — do not construct a `Meta{}` literal directly (`meta.go`'s doc comment explicitly asks callers to route through `NewMeta()` so a future `SchemaVersion` bump has one call site).

---

### `internal/migrate/validate.go` (invariant pass)

**Analog:** `internal/graphstore/store.go` `Reader` interface — `GetNode`, `IterateNodes`, `IterateEdges`

**Pattern:** open a `Reader` (via `store.Snapshot()`) against the temp store post-write and drive the D-09 checks through its existing iteration/lookup surface — no new read machinery needed:
```go
// Reader interface (internal/graphstore/store.go) — use as-is:
GetNode(id string) (*schema.Node, error)     // returns ErrNotFound if absent — the dangling-edge check
IterateEdges(srcPrefix string) (EdgeIterator, error)
IterateNodes() (NodeIterator, error)
```
Every iterator here follows the same `for it.Next() { ... }; if err := it.Err(); err != nil { ... }` shape already established by `EdgeIterator`/`NodeIterator`/`FileIterator` in `store.go` — validate.go's own loops should match that shape exactly for consistency (`it.Next()`/`.Node()`or`.Edge()`/`it.Err()`/`it.Close()`).

**Count-reconciliation tolerance (RESEARCH Pitfall/Open Question 3 — must be encoded, not derived from graphstore code):** migrated edge count may legitimately be `<=` source edge count because `edgeKey(src, kind, dst)` (in `keys.go`) intentionally omits line/col — two source rows sharing `(source,kind,target)` collapse to one stored edge. `keys.go`'s own doc comment on `edgeKey` confirms this is deliberate:
> "line/col are intentionally NOT part of this key in v1 ... Two structurally distinct call sites sharing the same (src, kind, dst) collapse to one stored edge here — this is deliberate dedup behavior, not a bug"

---

### `internal/migrate/swap.go` (atomic dir replace)

**Analog:** `internal/upgrade/swap.go` — `atomicSwap`, `swapWindows`, `checkWritable` (full file read above)

**Directly reusable shape (adapt file-swap → directory-swap):**
```go
// checkWritable-equivalent: probe write access to the PARENT dir before
// doing any work (upgrade's own "fail fast before downloading" rationale
// maps directly to "fail fast before running the whole migration").
func checkWritable(targetPath string) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".codegraph-migrate-writable-check-*")
	if err != nil {
		return fmt.Errorf("migrate: %s is not writable...: %w", targetPath, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}
```
**Core swap pattern — temp file to temp DIR, same discipline:**
```go
// atomicSwap in upgrade/swap.go:
// 1. write new content to a temp path IN THE SAME DIRECTORY as target (never os.TempDir() — EXDEV)
// 2. cleanupTemp bool + defer os.Remove(tmpPath), flipped false only after the final rename succeeds
// 3. os.Rename(tmpPath, targetPath) on POSIX — atomic same-fs rename
if err := os.Rename(tmpPath, targetPath); err != nil {
    return fmt.Errorf("upgrade: rename new binary into place: %w", err)
}
cleanupTemp = false
return nil
```
**Migration-specific difference (D-07):** target is a directory (`.codegraph/`), not a single file. `os.Rename` onto an existing *non-empty directory* fails on most platforms — RESEARCH's required three-step swap (not present in `upgrade/swap.go`, since that swaps a single file):
```go
// 1. if dst exists: os.Rename(dst, dst+".old")
// 2. os.Rename(tmp, dst)
// 3. os.RemoveAll(dst+".old")   // log failure here, don't swallow (non-fatal but must be surfaced)
```
This mirrors `swapWindows`'s already-established "rename original aside, rename new into place, remove aside" three-step dance in `upgrade/swap.go` — reuse that exact control flow (including the `WR-04` restore-on-failure discipline: if step 2 fails, attempt to restore from `dst+".old"` and report both errors if restore also fails) rather than reinventing it. Sibling-temp-dir requirement: `<root>/.codegraph.migrate-tmp-<pid-or-rand>/`, never `os.TempDir()` (same EXDEV rationale `upgrade/swap.go`'s own doc comment states for the binary case).

---

### `internal/cli/migrate.go` (cobra command)

**Analog:** `internal/cli/sync.go` (`newSyncCmd`) and `internal/cli/index.go` (`newIndexCmd`) — both read in full above.

**Structure to copy (full pattern from `sync.go`):**
```go
func newMigrateCmd() *cobra.Command {
	var from, to string
	var force, dropDangling bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Convert a TypeScript CodeGraph SQLite index into the new format",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// resolve --from (default: TS .codegraph/ in cwd) / --to (default: new-format .codegraph/, in place)
			// delegate to migrate.Run(from, to, migrate.Options{Force: force, DropDangling: dropDangling})
			stats, err := migrate.Run(from, to, migrate.Options{...})
			if err != nil {
				return err
			}
			// print reconciliation report (mirrors printSyncSummary's post-run report line)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "path to the TS .codegraph/ directory (default: autodetect in cwd)")
	cmd.Flags().StringVar(&to, "to", "", "path to write the new-format .codegraph/ (default: in place)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite a non-empty, unrecognized target without confirming")
	cmd.Flags().BoolVar(&dropDangling, "drop-dangling", false, "drop dangling edges instead of failing loud (D-09.2)")

	return cmd
}
```
**Error/ErrNotInitialized-style sentinel pattern** (`internal/cli/root.go` — copy the shape for a new `ErrNotATSSource` or similar, so `migrate`'s detection failure reads consistently with `ErrNotInitialized`/`ErrAlreadyInitialized`):
```go
var ErrNotInitialized = errors.New("cli: not initialized")
```
**Registration point:** `internal/cli/root.go` `newRootCmd()` — add `newMigrateCmd()` to the `root.AddCommand(...)` call alongside the other subcommands (single line addition, do not perturb existing registrations — RESEARCH's explicit integration-point warning).

**`--force`/confirm pattern** (`internal/cli/index.go` lines ~41-50) — reuse `confirm(cmd, prompt)` for the "refuse to overwrite a non-empty target that isn't recognizably a prior migration without `--force`" D-08 behavior:
```go
if !force {
    ok, err := confirm(cmd, fmt.Sprintf("..."))
    if err != nil { return err }
    if !ok {
        fmt.Fprintln(cmd.OutOrStdout(), "aborted (pass --force to ... without confirming)")
        return nil
    }
}
```

---

### `internal/migrate/migrate_test.go` + fixture harness (test)

**No committed aged `.db` fixture exists** (RESEARCH confirmed via filesystem search) — this is a genuine "no analog" case; the harness itself is new deliverable code, not a copy from an existing test. See `## No Analog Found` below.

**Closest structural analog for round-trip-style assertions:** the existing `Export`/`Import` pair in `internal/graphstore/export.go` has a round-trip test (`TestDeterministicRebuild` referenced in `writeExportRecord`'s doc comment) that asserts "export then import into a fresh store reproduces an identical graph" — `migrate_test.go`'s happy-path test should follow the same shape (`migrate DB → assert counts/spot-check records`, not byte-for-byte since the source format differs). Locate the exact test file (`grep -rl TestDeterministicRebuild internal/graphstore/`) at plan/implementation time to copy the assertion style.

**Fixture reconstruction pattern (RESEARCH §Fixture Question — do in-Go, no `sqlite3` binary dependency):**
```go
// In test: open a fresh DB read-write via modernc.org/sqlite, Exec the DDL
// from testdata/golden/ts-schema.sql, Exec seed INSERTs from
// ts-schema.dump.sql with <EPOCH_MS> substituted for a fixed valid ms
// value, close — then point the migrator at it read-only.
```

## Shared Patterns

### Error wrapping convention
**Source:** `internal/graphstore/export.go` (`fmt.Errorf("export meta: %w", err)`, `fmt.Errorf("import: read kind: %w", err)`)
**Apply to:** every file in `internal/migrate` — adopt `"migrate: <verb>: %w"` prefix consistently (reader.go, translate.go, migrate.go, progress.go, validate.go, swap.go).

### `rows.Err()` / `iter.Error()` post-loop check (fail-loud, never-silent discipline)
**Source:** `internal/graphstore/export.go` `exportNamespace` (`return iter.Error()` after the `for` loop)
**Apply to:** every `internal/migrate/reader.go` read loop — RESEARCH Pitfall 2 explicitly calls this the exact silent-I/O class that deep review caught in Phases 4 & 6; this project's own `iter.Error()` idiom is the in-repo precedent to replicate as `rows.Err()`.

### Additive-only namespace/schema discipline
**Source:** `internal/schema/meta.go` (`SchemaVersion` doc comment) + `internal/graphstore/keys.go` (`prefixFileIndex` doc: "Additive namespace — SchemaVersion stays 1")
**Apply to:** the new `m/migration` progress-record key and any `Writer`/`Reader` interface additions (`PutMigration`/`GetMigration`) — no `SchemaVersion` bump required, matching the project's established precedent for the Phase-4 `x/` namespace addition.

### Sole storage door (D-04a boundary)
**Source:** `internal/graphstore/store.go` doc comment: "No package outside internal/graphstore ... may import the engine directly — archtest.TestNoPackageBypassesGraphStore enforces this"
**Apply to:** `internal/migrate` — all writes MUST go through `graphstore.Writer`/`graphstore.GraphStore`, never a direct Pebble import. RESEARCH also recommends (§Package Legitimacy Audit) considering a parallel archtest guard confining `modernc.org/sqlite` imports to `internal/migrate` only.

### Atomic same-filesystem temp+rename swap
**Source:** `internal/upgrade/swap.go` (`atomicSwap`, full file)
**Apply to:** `internal/migrate/swap.go` — same sibling-temp-dir-never-os.TempDir(), cleanup-flag-until-success, Windows rename-aside-then-in fallback discipline; extended to a 3-step directory version per D-07.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/migrate/reader.go` (SQLite-specific parts: DSN construction, `PRAGMA table_info`, `database/sql` usage) | service | streaming read | No existing code in this repo uses `database/sql` or any SQL driver — `graphstore.Export`'s Pebble-iterator shape is the closest control-flow analog but the concrete API (driver registration, `sql.Rows`, `sql.NullString`) is genuinely new; patterns sourced from RESEARCH.md citations (pkg.go.dev, sqlite.org) instead of an in-repo file. |
| `internal/migrate/migrate_test.go` fixture-reconstruction harness | test | batch/file-I/O | No committed aged TS SQLite `.db` fixture exists anywhere under `testdata/` (verified by RESEARCH via filesystem search) — this harness is new deliverable code per RESEARCH §Fixture Question, not copyable from an existing fixture generator in the repo. |

## Metadata

**Analog search scope:** `internal/graphstore/` (batch.go, export.go, keys.go, store.go), `internal/schema/` (meta.go, graph.proto), `internal/cli/` (root.go, sync.go, index.go), `internal/upgrade/` (swap.go)
**Files scanned:** 8 read in full, 1 partially (graph.proto — Node message only; Edge/File message bodies not re-read this pass, confirm field numbers against the live proto at plan time)
**Pattern extraction date:** 2026-07-12

**Open items for the planner:**
1. Locate the exact `pebbleReader.GetMeta` implementation file (not read this pass) before writing the `PutMigration`/`GetMigration` additive interface change — likely `internal/graphstore/pebble_store.go` or a sibling reader file (`grep -n "func.*GetMeta" internal/graphstore/*.go`).
2. Re-confirm `Edge`/`File` message field numbers directly against `internal/schema/graph.proto` (only `Node`'s field list was read verbatim this pass; the Field Mapping tables in `07-RESEARCH.md` already did this reconciliation and are the authoritative source — cross-check, don't re-derive).
3. Locate the graphstore round-trip test (`TestDeterministicRebuild` or similarly named) referenced in `export.go`'s doc comments, for the exact assertion style to mirror in `migrate_test.go`'s happy-path test.
