// Package migrate — this file is the orchestration capstone (MIGR-01/
// MIGR-02): Run wires the read-only reader (reader.go/translate.go), the
// resumable progress cursor (progress.go), the D-09 invariant pass
// (validate.go), and the atomic directory swap (swap.go) into one call that
// converts a TS CodeGraph SQLite index into a healthy new-format store.
//
// Sequence (07-RESEARCH.md §System Architecture Diagram): detect + guard the
// source -> open/create a deterministic sibling partial store -> resume from
// the durable cursor if present -> stream files, then nodes, then edges
// (nodes-before-edges so PutEdge's ownerPath resolves, D-04) in bounded,
// durably-checkpointed batches -> recompute per-file edge_count from the
// written x/ index (Pitfall 7) -> run validate (D-09) -> stamp Meta.healthy
// only after validate passes (D-10) -> atomically swap the partial store
// into place (D-07). Every I/O error is wrapped and returned — never
// swallowed (this is the highest silent-failure-risk plan in the phase).
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// batchSize bounds how many Puts accumulate in a single Writer batch before
// migrate.go commits it and durably advances the progress cursor (D-06) —
// never one giant unbounded Commit for a monorepo-scale index.
const batchSize = 5000

// partialStoreName is the DETERMINISTIC (not randomly-named) sibling
// directory Run() writes into while migrating, so a re-run after a crash
// finds the SAME partial store and can resume from its durable cursor
// (D-06/D-07) rather than starting a fresh, randomly-named temp dir that
// orphans the interrupted one. This name is fixed per target parent
// directory: running two concurrent migrations into different targets that
// share a parent directory is not supported (single-concurrent-migration
// constraint) — a second concurrent Run would collide on the same partial
// store path.
const partialStoreName = ".codegraph.migrate-partial"

// migrateTableOrder is the fixed files -> nodes -> edges write order
// (nodes-before-edges is required for D-04's ownerPath lookup; files has no
// such dependency and is written first for no reason other than a stable,
// documented order the progress cursor can name unambiguously).
var migrateTableOrder = []string{"files", "nodes", "edges"}

// errTestInterrupted is returned by Run when the test-only testStopAfterBatch
// hook requests a simulated mid-migration interruption. Never set outside
// tests.
var errTestInterrupted = fmt.Errorf("migrate: test-injected interruption")

// testStopAfterBatch, when non-nil, is invoked after every durable cursor
// checkpoint (immediately after advanceCursor's progress-cursor Commit
// succeeds) with the table/rowid the checkpoint just recorded. If it
// returns true, Run aborts immediately with errTestInterrupted — the
// test-only seam the resume-after-interruption test uses to simulate a
// crash between two batches (07-06-PLAN.md's "keep the seam unexported/
// test-only" instruction). Never set outside tests.
var testStopAfterBatch func(table string, rowid int64) bool

// Result summarizes a completed (or resumed-to-completion) migration run:
// final record counts (read back from the migrated store, so they reflect
// this run's writes plus anything already committed from a prior
// interrupted attempt), whether this call resumed a prior in-progress run,
// the stamped Meta.health_message, and the full D-09 validation Report.
type Result struct {
	Nodes, Edges, Files int64
	Resumed             bool
	HealthMessage       string
	Report              Report
}

// Run converts the TS CodeGraph SQLite index at from into a healthy
// new-format store at to. from may be a TS .codegraph/ directory (the
// source *.db is auto-detected via FindDBFile) or a direct path to the
// *.db file. to is the final new-format .codegraph/ directory path — Run
// writes into a deterministic sibling partial store first and only
// replaces to via an atomic directory swap once validate (D-09) passes
// (D-07/D-10). Every error is wrapped with a "migrate: ..." prefix and
// returned — never swallowed.
func Run(from, to string, opts Options) (Result, error) {
	dbPath, err := resolveSourceDB(from)
	if err != nil {
		return Result{}, err
	}
	src, err := OpenSource(dbPath)
	if err != nil {
		return Result{}, err
	}
	defer src.Close()

	if err := src.DetectTS(); err != nil {
		return Result{}, err
	}
	srcVersion, err := src.SchemaVersion()
	if err != nil {
		return Result{}, err
	}

	target, err := filepath.Abs(to)
	if err != nil {
		return Result{}, fmt.Errorf("migrate: resolve target %q: %w", to, err)
	}
	if err := checkWritableDir(filepath.Dir(target)); err != nil {
		return Result{}, err
	}
	if err := checkTargetOverwrite(target, opts.Force); err != nil {
		return Result{}, err
	}

	tmpDir := partialDir(target)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("migrate: create partial store dir %s: %w", tmpDir, err)
	}
	storeDir := filepath.Join(tmpDir, "store")

	store, err := graphstore.Open(storeDir)
	if err != nil {
		return Result{}, fmt.Errorf("migrate: open partial store %s: %w", storeDir, err)
	}
	defer store.Close() // safe no-op if already explicitly closed below (pebbleStore.Close is idempotent)

	progress, found, err := readProgress(store)
	if err != nil {
		return Result{}, err
	}
	resumed := found && progress.Status == StatusInProgress

	if found && progress.Status == StatusComplete {
		// Data was fully written and validated in a prior invocation; only
		// the final swap didn't happen (the process was interrupted between
		// saveProgress(complete) and atomicSwapDir). Re-read the already-
		// stamped Meta/counts and swap directly — no re-write, no re-validate.
		return finishFromComplete(store, tmpDir, target)
	}

	resumeIdx, resumeAfterRowID := resumePosition(found, progress)

	nodeFilePath, err := seedNodeFilePath(store)
	if err != nil {
		return Result{}, err
	}

	bw, err := newBatchWriter(store)
	if err != nil {
		return Result{}, err
	}

	for i, table := range migrateTableOrder {
		if i < resumeIdx {
			continue
		}
		afterRowID := int64(0)
		if i == resumeIdx {
			afterRowID = resumeAfterRowID
		}

		if err := scanAndWriteTable(src, bw, table, afterRowID, srcVersion, nodeFilePath); err != nil {
			return Result{}, err
		}

		if err := bw.commitData(); err != nil {
			return Result{}, err
		}
		if i+1 < len(migrateTableOrder) {
			nextTable := migrateTableOrder[i+1]
			if err := advanceCursorChecked(bw, nextTable, 0, srcVersion); err != nil {
				return Result{}, err
			}
		}
	}

	if err := recomputeFileEdgeCounts(store); err != nil {
		return Result{}, err
	}

	report, err := validate(src, store, opts)
	if err != nil {
		// D-10: validation failed — the store is never marked healthy and
		// the swap never runs. The partial store is left in place (with its
		// in_progress cursor) so a future --drop-dangling or corrected run
		// can resume/inspect it; the target directory is untouched.
		return Result{Report: report}, err
	}

	nodeCount, err := countNodes(store)
	if err != nil {
		return Result{}, err
	}
	edgeCount, err := countEdges(store)
	if err != nil {
		return Result{}, err
	}
	fileCount, err := countFiles(store)
	if err != nil {
		return Result{}, err
	}

	healthMsg := healthMessage(srcVersion)

	meta := schema.NewMeta()
	meta.NodeCount = nodeCount
	meta.EdgeCount = edgeCount
	meta.HasFileIndex = true
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	meta.HealthMessage = healthMsg
	meta.Healthy = true // ONLY set true here, after validate returned nil (D-10)

	mw, err := store.NewWriter()
	if err != nil {
		return Result{}, fmt.Errorf("migrate: new writer for meta: %w", err)
	}
	if err := mw.PutMeta(meta); err != nil {
		_ = mw.Close()
		return Result{}, fmt.Errorf("migrate: put meta: %w", err)
	}
	if err := saveProgress(mw, Progress{
		SourceSchemaVersion: srcVersion,
		TargetSchemaVersion: schema.SchemaVersion,
		LastTable:           "edges",
		LastRowID:           0,
		Status:              StatusComplete,
	}); err != nil {
		_ = mw.Close()
		return Result{}, err
	}
	if err := mw.Commit(); err != nil {
		return Result{}, fmt.Errorf("migrate: commit meta+complete: %w", err)
	}

	if err := store.Close(); err != nil {
		return Result{}, fmt.Errorf("migrate: close partial store: %w", err)
	}
	if err := atomicSwapDir(tmpDir, target); err != nil {
		return Result{}, err
	}

	return Result{
		Nodes:         nodeCount,
		Edges:         edgeCount,
		Files:         fileCount,
		Resumed:       resumed,
		HealthMessage: healthMsg,
		Report:        report,
	}, nil
}

// healthMessage is the D-01 open-question resolution (accept-and-document):
// a migrated graph carries verbatim TS ids, which differ from what a native
// index/sync computes for the same symbols, so the first post-migration
// sync/index will see every node as "new" and perform a full re-index. This
// is documented here rather than silently absorbed.
func healthMessage(srcVersion int) string {
	return fmt.Sprintf(
		"migrated-from-ts (source schema v%d); the first `codegraph sync`/`index` after this migration will perform a full re-index, because TS node ids differ from native ids",
		srcVersion,
	)
}

// resolveSourceDB accepts either a TS .codegraph/ directory (auto-detects
// the single *.db file via FindDBFile) or a direct path to the *.db file.
func resolveSourceDB(from string) (string, error) {
	info, err := os.Stat(from)
	if err != nil {
		return "", fmt.Errorf("migrate: resolve source %q: %w", from, err)
	}
	if info.IsDir() {
		return FindDBFile(from)
	}
	return from, nil
}

// partialDir returns the deterministic sibling partial-store directory for
// target — always filepath.Dir(target)/partialStoreName, never a randomly
// named or os.TempDir()-rooted path, so (a) a re-run after a crash finds
// the same partial store (D-06) and (b) the later atomicSwapDir rename is
// always same-filesystem (never EXDEV, RESEARCH Pitfall 1).
func partialDir(target string) string {
	return filepath.Join(filepath.Dir(target), partialStoreName)
}

// checkTargetOverwrite enforces D-08: refuse to overwrite a non-empty
// target directory that isn't recognizably a prior healthy migration,
// unless force is set. A missing or empty target is always fine.
func checkTargetOverwrite(target string, force bool) error {
	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("migrate: check target %s: %w", target, err)
	}
	if len(entries) == 0 || force {
		return nil
	}

	if store, openErr := graphstore.Open(filepath.Join(target, "store")); openErr == nil {
		healthy := false
		if r, snapErr := store.Snapshot(); snapErr == nil {
			if m, metaErr := r.GetMeta(); metaErr == nil && m.GetHealthy() {
				healthy = true
			}
			_ = r.Close()
		}
		_ = store.Close()
		if healthy {
			return nil
		}
	}

	return fmt.Errorf("migrate: target %s is a non-empty directory that does not look like a prior healthy migration; pass Options{Force: true} to overwrite (no changes made)", target)
}

// readProgress opens a snapshot on store and loads the durable migration
// cursor, if any.
func readProgress(store graphstore.GraphStore) (Progress, bool, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return Progress{}, false, fmt.Errorf("migrate: read progress: snapshot: %w", err)
	}
	defer snap.Close()
	return loadProgress(snap)
}

// resumePosition translates a loaded Progress record into a
// (tableIndex, afterRowID) starting position for the write loop. A fresh
// run (found=false) or an unrecognized LastTable always starts at index 0,
// rowid 0.
func resumePosition(found bool, p Progress) (idx int, afterRowID int64) {
	if !found || p.Status != StatusInProgress {
		return 0, 0
	}
	for i, t := range migrateTableOrder {
		if t == p.LastTable {
			return i, p.LastRowID
		}
	}
	return 0, 0
}

// finishFromComplete handles the narrow resume case where a prior Run
// finished writing, validating, and stamping Meta.healthy=true, but the
// process was interrupted before the atomic swap ran. No re-write or
// re-validate is needed or performed — just read back the already-final
// counts/health message and swap.
func finishFromComplete(store graphstore.GraphStore, tmpDir, target string) (Result, error) {
	nodeCount, err := countNodes(store)
	if err != nil {
		return Result{}, err
	}
	edgeCount, err := countEdges(store)
	if err != nil {
		return Result{}, err
	}
	fileCount, err := countFiles(store)
	if err != nil {
		return Result{}, err
	}
	meta, err := getStoreMeta(store)
	if err != nil {
		return Result{}, err
	}

	if err := store.Close(); err != nil {
		return Result{}, fmt.Errorf("migrate: close partial store: %w", err)
	}
	if err := atomicSwapDir(tmpDir, target); err != nil {
		return Result{}, err
	}

	return Result{
		Nodes:         nodeCount,
		Edges:         edgeCount,
		Files:         fileCount,
		Resumed:       true,
		HealthMessage: meta.GetHealthMessage(),
	}, nil
}

func getStoreMeta(store graphstore.GraphStore) (*schema.Meta, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("migrate: read meta: snapshot: %w", err)
	}
	defer snap.Close()
	m, err := snap.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("migrate: read meta: %w", err)
	}
	return m, nil
}

// seedNodeFilePath populates the id->FilePath map PutEdge's ownerPath
// lookup needs from whatever nodes are ALREADY committed in store — which,
// on a fresh run, is none (a cheap no-op scan), and on a resume that skips
// or partially covers the nodes table, is exactly the set the write loop's
// own nodes-table callback will not re-visit this run (ScanTable's
// afterRowID filter skips already-committed rows). Combined with the
// callback adding newly-written nodes as they're staged, this guarantees
// nodeFilePath has full coverage by the time the edges table is reached,
// regardless of resume state.
func seedNodeFilePath(store graphstore.GraphStore) (map[string]string, error) {
	m := make(map[string]string)
	snap, err := store.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("migrate: seed node file paths: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateNodes()
	if err != nil {
		return nil, fmt.Errorf("migrate: seed node file paths: iterate nodes: %w", err)
	}
	defer it.Close()

	for it.Next() {
		n := it.Node()
		m[n.GetId()] = n.GetFilePath()
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("migrate: seed node file paths: iterate nodes: %w", err)
	}
	return m, nil
}

// batchWriter tracks the currently-open Writer for the bounded-batch write
// phase, so migrate.go's table-scan callbacks can stage puts, periodically
// commit at batchSize, and durably advance the progress cursor.
type batchWriter struct {
	store graphstore.GraphStore
	w     graphstore.Writer
	n     int
}

func newBatchWriter(store graphstore.GraphStore) (*batchWriter, error) {
	w, err := store.NewWriter()
	if err != nil {
		return nil, fmt.Errorf("migrate: new writer: %w", err)
	}
	return &batchWriter{store: store, w: w}, nil
}

// commitData commits whatever is currently staged (a no-op if nothing is
// staged) and opens a fresh Writer for subsequent puts.
func (bw *batchWriter) commitData() error {
	if bw.n == 0 {
		return nil
	}
	if err := bw.w.Commit(); err != nil {
		return fmt.Errorf("migrate: commit batch: %w", err)
	}
	w, err := bw.store.NewWriter()
	if err != nil {
		return fmt.Errorf("migrate: new writer: %w", err)
	}
	bw.w = w
	bw.n = 0
	return nil
}

// advanceCursor commits any staged data (commitData) and then durably
// persists the progress cursor in its own small committed batch, so the
// cursor never advances ahead of the data it describes (D-06).
func (bw *batchWriter) advanceCursor(table string, rowid int64, srcVersion int) error {
	if err := bw.commitData(); err != nil {
		return err
	}
	pw, err := bw.store.NewWriter()
	if err != nil {
		return fmt.Errorf("migrate: new writer for progress: %w", err)
	}
	if err := saveProgress(pw, Progress{
		SourceSchemaVersion: srcVersion,
		TargetSchemaVersion: schema.SchemaVersion,
		LastTable:           table,
		LastRowID:           rowid,
		Status:              StatusInProgress,
	}); err != nil {
		_ = pw.Close()
		return err
	}
	if err := pw.Commit(); err != nil {
		return fmt.Errorf("migrate: commit progress cursor: %w", err)
	}
	return nil
}

// advanceCursorChecked wraps advanceCursor with the test-only
// testStopAfterBatch seam (never set outside tests).
func advanceCursorChecked(bw *batchWriter, table string, rowid int64, srcVersion int) error {
	if err := bw.advanceCursor(table, rowid, srcVersion); err != nil {
		return err
	}
	if testStopAfterBatch != nil && testStopAfterBatch(table, rowid) {
		return errTestInterrupted
	}
	return nil
}

// scanAndWriteTable streams table via src.ScanTable, translating and
// staging each row through bw, periodically committing at batchSize and
// durably advancing the cursor (D-06). For nodes, it records id->FilePath
// into nodeFilePath (D-04). For edges, ownerPath is nodeFilePath[source]
// for a symbol source, or "" for a file:-prefixed source (isFileEndpoint,
// validate.go) or an unresolved (dangling) source — PutEdge tolerates an
// empty ownerPath by skipping the x/ entry, and the D-09.2 dangling check
// catches a truly-missing symbol source.
func scanAndWriteTable(src *Source, bw *batchWriter, table string, afterRowID int64, srcVersion int, nodeFilePath map[string]string) error {
	switch table {
	case "files":
		return src.ScanTable(table, afterRowID, func(rowid int64, row map[string]any) error {
			f, err := fileFromRow(row)
			if err != nil {
				return err
			}
			if err := bw.w.PutFile(f); err != nil {
				return fmt.Errorf("migrate: put file %s: %w", f.GetPath(), err)
			}
			bw.n++
			if bw.n >= batchSize {
				return advanceCursorChecked(bw, table, rowid, srcVersion)
			}
			return nil
		})
	case "nodes":
		return src.ScanTable(table, afterRowID, func(rowid int64, row map[string]any) error {
			n, err := nodeFromRow(row)
			if err != nil {
				return err
			}
			nodeFilePath[n.GetId()] = n.GetFilePath()
			if err := bw.w.PutNode(n); err != nil {
				return fmt.Errorf("migrate: put node %s: %w", n.GetId(), err)
			}
			bw.n++
			if bw.n >= batchSize {
				return advanceCursorChecked(bw, table, rowid, srcVersion)
			}
			return nil
		})
	case "edges":
		return src.ScanTable(table, afterRowID, func(rowid int64, row map[string]any) error {
			e, err := edgeFromRow(row)
			if err != nil {
				return err
			}
			ownerPath := ""
			if !isFileEndpoint(e.GetSource()) {
				ownerPath = nodeFilePath[e.GetSource()]
			}
			if err := bw.w.PutEdge(e, ownerPath); err != nil {
				return fmt.Errorf("migrate: put edge %s->%s(%s): %w", e.GetSource(), e.GetTarget(), e.GetKind(), err)
			}
			bw.n++
			if bw.n >= batchSize {
				return advanceCursorChecked(bw, table, rowid, srcVersion)
			}
			return nil
		})
	default:
		return fmt.Errorf("migrate: unknown table %q", table)
	}
}

// recomputeFileEdgeCounts sets each File.edge_count from the number of
// edge entries actually present in that file's x/ index (Pitfall 7: TS
// files has no edge_count column, so it cannot be carried — it must be
// recomputed). This reads the x/ index as the source of truth rather than
// any in-memory bookkeeping, so it produces the correct count regardless
// of resume state (entries written in a prior interrupted run are already
// durably in the store and are counted the same as entries written in this
// run).
func recomputeFileEdgeCounts(store graphstore.GraphStore) error {
	snap, err := store.Snapshot()
	if err != nil {
		return fmt.Errorf("migrate: recompute file edge counts: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateFiles()
	if err != nil {
		return fmt.Errorf("migrate: recompute file edge counts: iterate files: %w", err)
	}
	type patch struct {
		path  string
		count int64
	}
	var patches []patch
	for it.Next() {
		path := it.File().GetPath()
		n, cerr := countFileIndexEdges(snap, path)
		if cerr != nil {
			_ = it.Close()
			return cerr
		}
		patches = append(patches, patch{path: path, count: n})
	}
	if err := it.Err(); err != nil {
		_ = it.Close()
		return fmt.Errorf("migrate: recompute file edge counts: iterate files: %w", err)
	}
	_ = it.Close()

	w, err := store.NewWriter()
	if err != nil {
		return fmt.Errorf("migrate: recompute file edge counts: new writer: %w", err)
	}
	n := 0
	for _, p := range patches {
		f, err := snap.GetFile(p.path)
		if err != nil {
			_ = w.Close()
			return fmt.Errorf("migrate: recompute file edge counts: get file %s: %w", p.path, err)
		}
		f.EdgeCount = p.count
		if err := w.PutFile(f); err != nil {
			_ = w.Close()
			return fmt.Errorf("migrate: recompute file edge counts: put file %s: %w", p.path, err)
		}
		n++
		if n >= batchSize {
			if err := w.Commit(); err != nil {
				return fmt.Errorf("migrate: recompute file edge counts: commit: %w", err)
			}
			w, err = store.NewWriter()
			if err != nil {
				return fmt.Errorf("migrate: recompute file edge counts: new writer: %w", err)
			}
			n = 0
		}
	}
	if n > 0 {
		if err := w.Commit(); err != nil {
			return fmt.Errorf("migrate: recompute file edge counts: commit: %w", err)
		}
	} else {
		_ = w.Close()
	}
	return nil
}

func countFileIndexEdges(snap graphstore.Reader, path string) (int64, error) {
	it, err := snap.IterateFileIndex(path)
	if err != nil {
		return 0, fmt.Errorf("migrate: recompute file edge counts: iterate file index %s: %w", path, err)
	}
	defer it.Close()
	var n int64
	for it.Next() {
		if !it.Entry().IsNode {
			n++
		}
	}
	if err := it.Err(); err != nil {
		return 0, fmt.Errorf("migrate: recompute file edge counts: file index %s: %w", path, err)
	}
	return n, nil
}
