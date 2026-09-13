package indexer

import (
	"sort"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// loadStoredExclusions returns every ExcludedFile record currently
// persisted under the c/ namespace, keyed by path — one IterateExcludedFiles
// pass over r, checking Err() after Next() returns false exactly like the
// `deleted` diff above does with fit.Err(). Sync's incremental lifecycle
// (D-07) diffs this snapshot against DiscoverAll's freshly-walked
// Discovery.Excluded list via diffExclusions, mirroring how the existing
// `deleted` diff compares r0.IterateFiles() against the freshly discovered
// file set.
func loadStoredExclusions(r graphstore.Reader) (map[string]*schema.ExcludedFile, error) {
	it, err := r.IterateExcludedFiles()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	stored := make(map[string]*schema.ExcludedFile)
	for it.Next() {
		x := it.ExcludedFile()
		stored[x.GetPath()] = x
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return stored, nil
}

// diffExclusions compares stored (the c/ namespace as it exists in the
// store right now) against fresh (this Sync's freshly-walked
// Discovery.Excluded list) and reports which records to upsert and which
// stored paths to prune — the per-path incremental lifecycle D-07
// requires for the c/ namespace, the same technique Sync's own `deleted`
// diff already uses over r0.IterateFiles().
//
// A fresh record is upserted when its path is absent from stored, or
// present but no longer proto.Equal to the stored record (T-10-13: an
// unchanged record is never re-written, so the no-op fast path stays
// genuinely no-op). A stored path absent from the fresh set is pruned
// (T-10-07: no stale record survives past the disk condition that
// produced it). Both return slices are sorted by path so the caller's
// staging order — and therefore the committed batch — stays deterministic
// run to run (the sync-determinism suite compares export streams).
func diffExclusions(stored map[string]*schema.ExcludedFile, fresh []*schema.ExcludedFile) (upserts []*schema.ExcludedFile, prunes []string) {
	freshPaths := make(map[string]struct{}, len(fresh))
	for _, x := range fresh {
		path := x.GetPath()
		freshPaths[path] = struct{}{}
		existing, ok := stored[path]
		if !ok || !proto.Equal(existing, x) {
			upserts = append(upserts, x)
		}
	}
	for path := range stored {
		if _, ok := freshPaths[path]; !ok {
			prunes = append(prunes, path)
		}
	}

	sort.Slice(upserts, func(i, j int) bool { return upserts[i].GetPath() < upserts[j].GetPath() })
	sort.Strings(prunes)

	return upserts, prunes
}

// stageExclusionDiff applies upserts/prunes to the caller's Writer w —
// it never opens a Writer of its own (T-10-10 pins Sync's own NewWriter
// call-site count at exactly 2). Deletes are staged before puts so a
// path that both loses its old detail and gains a fresh one lands in its
// final upserted state.
func stageExclusionDiff(w graphstore.Writer, upserts []*schema.ExcludedFile, prunes []string) error {
	for _, path := range prunes {
		if err := w.DeleteExcludedFile(path); err != nil {
			return err
		}
	}
	for _, x := range upserts {
		if err := w.PutExcludedFile(x); err != nil {
			return err
		}
	}
	return nil
}
