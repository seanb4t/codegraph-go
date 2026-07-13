// Package migrate — this file is the D-09 post-migration invariant pass:
// count reconciliation (de-dup aware for edges) plus referential-integrity
// scanning (zero-dangling-edges, file:-endpoint exempt), driving the
// fail-loud-vs-`--drop-dangling` policy that gates Meta.healthy (D-10).
// validate reads the migrated store back through the existing
// graphstore.Reader surface — no new read machinery, per 07-PATTERNS.md.
package migrate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// Options controls behavior shared across internal/migrate's orchestration
// (07-06) and validation phases. This is the package's single definition —
// 07-06's migrate.go reconciles with it rather than redeclaring it.
type Options struct {
	// Force allows overwriting a non-empty target that isn't recognizably a
	// prior migration (D-08). Consumed by the orchestration layer (07-06);
	// validate itself does not read it.
	Force bool

	// DropDangling, when true, makes scanDangling delete each dangling
	// (non-file:) edge and record it in Report.Dropped instead of failing
	// loud — an explicit, opt-in lossy migration (D-09.2).
	DropDangling bool
}

// TableCounts holds one table/kind's source row count next to its migrated
// record count. For Edges, Source is the DISTINCT(source,kind,target) count
// (D-09.1) — never the raw source row count, because the Pebble edge key
// omits line/col and collapses same-triple rows (keys.go's edgeKey doc).
type TableCounts struct {
	Source   int64
	Migrated int64
}

// DanglingEdge identifies a migrated edge whose source and/or target
// endpoint failed to resolve to a migrated node (file:-prefixed endpoints
// are exempt — see isFileEndpoint).
type DanglingEdge struct {
	Source, Kind, Target string

	// MissingSource / MissingTarget record which endpoint(s) failed to
	// resolve; either or both may be true.
	MissingSource, MissingTarget bool
}

// Report is the D-09 structural-invariant pass's result: per-table count
// reconciliation plus the referential-integrity scan's findings. The
// caller (07-06) gates Meta.healthy=true on validate returning a nil error
// (D-10).
type Report struct {
	Nodes TableCounts
	Files TableCounts
	Edges TableCounts

	// Dangling lists every non-exempt edge endpoint that failed to
	// resolve, found before any --drop-dangling deletion.
	Dangling []DanglingEdge

	// Dropped counts dangling edges actually deleted under --drop-dangling.
	Dropped int
}

// isFileEndpoint reports whether id is a file:-prefixed edge endpoint — TS's
// synthetic file-node id shape (e.g. edges.source for a "contains" edge).
// These are exempt from the dangling-edge check: the new format models
// files as File records, not Node records, so a file: endpoint never
// resolves to a migrated node and must not be reported as corruption
// (07-RESEARCH.md Pitfall 4 / CONTEXT Open Question 2, locked to exempt).
func isFileEndpoint(id string) bool {
	return strings.HasPrefix(id, "file:")
}

// validate orchestrates the D-09 post-migration invariant pass: count
// reconciliation (reconcileCounts) then referential-integrity scanning
// (scanDangling), assembling a Report the caller uses to gate
// Meta.healthy=true on a nil error return (D-10). A non-nil error means the
// migrated graph failed a structural invariant — the caller must not swap
// or mark the store healthy.
func validate(src *Source, store graphstore.GraphStore, opts Options) (Report, error) {
	report, err := reconcileCounts(src, store)
	if err != nil {
		return report, err
	}
	if err := scanDangling(store, opts.DropDangling, &report); err != nil {
		return report, err
	}
	return report, nil
}

// reconcileCounts compares migrated Node/Edge/File counts (read back from
// store) against the source's row counts (D-09.1). Edge reconciliation is
// de-dup aware: it compares against src.CountDistinctEdges(), never
// src.CountRows("edges") — a raw-row comparison would fail a correctly-
// collapsing migration, since the Pebble edge key omits line/col.
func reconcileCounts(src *Source, store graphstore.GraphStore) (Report, error) {
	var report Report

	nodeCount, err := countNodes(store)
	if err != nil {
		return report, err
	}
	edgeCount, err := countEdges(store)
	if err != nil {
		return report, err
	}
	fileCount, err := countFiles(store)
	if err != nil {
		return report, err
	}

	srcNodes, err := src.CountRows("nodes")
	if err != nil {
		return report, fmt.Errorf("migrate: validate: %w", err)
	}
	srcFiles, err := src.CountRows("files")
	if err != nil {
		return report, fmt.Errorf("migrate: validate: %w", err)
	}
	srcEdges, err := src.CountDistinctEdges()
	if err != nil {
		return report, fmt.Errorf("migrate: validate: %w", err)
	}

	report.Nodes = TableCounts{Source: srcNodes, Migrated: nodeCount}
	report.Files = TableCounts{Source: srcFiles, Migrated: fileCount}
	report.Edges = TableCounts{Source: srcEdges, Migrated: edgeCount}

	if nodeCount != srcNodes {
		return report, fmt.Errorf("migrate: validate: reconcile nodes: expected %d (source rows), got %d (migrated)", srcNodes, nodeCount)
	}
	if fileCount != srcFiles {
		return report, fmt.Errorf("migrate: validate: reconcile files: expected %d (source rows), got %d (migrated)", srcFiles, fileCount)
	}
	if edgeCount != srcEdges {
		return report, fmt.Errorf("migrate: validate: reconcile edges: expected %d (source DISTINCT source/kind/target), got %d (migrated)", srcEdges, edgeCount)
	}

	return report, nil
}

func countNodes(store graphstore.GraphStore) (int64, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateNodes()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate nodes: %w", err)
	}
	defer it.Close()

	var n int64
	for it.Next() {
		n++
	}
	if err := it.Err(); err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate nodes: %w", err)
	}
	return n, nil
}

func countEdges(store graphstore.GraphStore) (int64, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateEdges("")
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate edges: %w", err)
	}
	defer it.Close()

	var n int64
	for it.Next() {
		n++
	}
	if err := it.Err(); err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate edges: %w", err)
	}
	return n, nil
}

func countFiles(store graphstore.GraphStore) (int64, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateFiles()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate files: %w", err)
	}
	defer it.Close()

	var n int64
	for it.Next() {
		n++
	}
	if err := it.Err(); err != nil {
		return 0, fmt.Errorf("migrate: validate: iterate files: %w", err)
	}
	return n, nil
}

// scanDangling builds an in-memory node-id set from store, then scans every
// edge's source/target endpoints for referential integrity (D-09.2).
// file:-prefixed endpoints are exempt (isFileEndpoint). When dropDangling is
// false (the default), a non-empty dangling list is a fail-loud error and
// the store is left unchanged. When true, each dangling edge is deleted via
// a Writer (DeleteEdge, plus a best-effort DeleteFileIndexEdge when the
// edge's source resolved to an owning file) and committed; validate then
// returns nil for this phase with report.Dropped recording the count — an
// explicit, opt-in lossy migration (D-09.2/D-10).
func scanDangling(store graphstore.GraphStore, dropDangling bool, report *Report) error {
	nodeIDs, err := nodeIDSet(store)
	if err != nil {
		return err
	}

	dangling, err := findDangling(store, nodeIDs)
	if err != nil {
		return err
	}
	report.Dangling = dangling

	if len(dangling) == 0 {
		return nil
	}

	if !dropDangling {
		first := dangling[0]
		return fmt.Errorf("migrate: validate: %d dangling edge(s) found, e.g. (%s,%s,%s) (pass --drop-dangling to drop and log)",
			len(dangling), first.Source, first.Kind, first.Target)
	}

	dropped, err := dropDanglingEdges(store, dangling)
	if err != nil {
		return err
	}
	report.Dropped = dropped
	return nil
}

func nodeIDSet(store graphstore.GraphStore) (map[string]struct{}, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("migrate: validate: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateNodes()
	if err != nil {
		return nil, fmt.Errorf("migrate: validate: iterate nodes: %w", err)
	}
	defer it.Close()

	ids := make(map[string]struct{})
	for it.Next() {
		ids[it.Node().GetId()] = struct{}{}
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("migrate: validate: iterate nodes: %w", err)
	}
	return ids, nil
}

func findDangling(store graphstore.GraphStore, nodeIDs map[string]struct{}) ([]DanglingEdge, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("migrate: validate: snapshot: %w", err)
	}
	defer snap.Close()

	it, err := snap.IterateEdges("")
	if err != nil {
		return nil, fmt.Errorf("migrate: validate: iterate edges: %w", err)
	}
	defer it.Close()

	var dangling []DanglingEdge
	for it.Next() {
		e := it.Edge()
		src, dst := e.GetSource(), e.GetTarget()

		missingSrc := false
		if !isFileEndpoint(src) {
			if _, ok := nodeIDs[src]; !ok {
				missingSrc = true
			}
		}
		missingDst := false
		if !isFileEndpoint(dst) {
			if _, ok := nodeIDs[dst]; !ok {
				missingDst = true
			}
		}

		if missingSrc || missingDst {
			dangling = append(dangling, DanglingEdge{
				Source: src, Kind: e.GetKind(), Target: dst,
				MissingSource: missingSrc, MissingTarget: missingDst,
			})
		}
	}
	if err := it.Err(); err != nil {
		return nil, fmt.Errorf("migrate: validate: iterate edges: %w", err)
	}
	return dangling, nil
}

// dropDanglingEdges deletes each dangling edge's e/ record, plus a
// best-effort x/ file-index cleanup. When an edge's source is itself the
// missing endpoint, ownerPath was never populated for that edge in the
// first place (migrate.go's write loop passes ownerPath="" for an
// unresolved source, per 07-PATTERNS.md), so there is no x/ entry to clean
// up. When the source DOES resolve (only the target is dangling), its
// owning file is looked up and the matching x/ entry removed too, so
// IterateFileIndex stays consistent with the dropped edge.
func dropDanglingEdges(store graphstore.GraphStore, dangling []DanglingEdge) (int, error) {
	snap, err := store.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: drop dangling: snapshot: %w", err)
	}
	defer snap.Close()

	w, err := store.NewWriter()
	if err != nil {
		return 0, fmt.Errorf("migrate: validate: drop dangling: new writer: %w", err)
	}
	defer w.Close()

	for _, d := range dangling {
		if err := w.DeleteEdge(d.Source, d.Kind, d.Target); err != nil {
			return 0, fmt.Errorf("migrate: validate: drop dangling edge (%s,%s,%s): %w", d.Source, d.Kind, d.Target, err)
		}

		if d.MissingSource {
			continue
		}
		n, err := snap.GetNode(d.Source)
		switch {
		case err == nil:
			if err := w.DeleteFileIndexEdge(n.GetFilePath(), d.Source, d.Kind, d.Target); err != nil {
				return 0, fmt.Errorf("migrate: validate: drop dangling edge file-index entry (%s,%s,%s): %w", d.Source, d.Kind, d.Target, err)
			}
		case errors.Is(err, graphstore.ErrNotFound):
			// No owner recorded — nothing to clean up.
		default:
			return 0, fmt.Errorf("migrate: validate: drop dangling: look up owner for %s: %w", d.Source, err)
		}
	}

	if err := w.Commit(); err != nil {
		return 0, fmt.Errorf("migrate: validate: drop dangling: commit: %w", err)
	}
	return len(dangling), nil
}
