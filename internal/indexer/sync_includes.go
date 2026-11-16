package indexer

import "github.com/seanb4t/codegraph-go/internal/bfs"

// includeDependents returns the files outside inBatch that include one of
// paths, directly or through other headers, in path order (design D6).
// paths are the files that the sync reparses, adds or deletes. A bare C or
// C++ call looks through the files that its file includes, so such a call
// can resolve differently once one of them changes.
func includeDependents(idx *storeIndex, paths []string, inBatch map[string]bool) ([]string, error) {
	if err := idx.load(); err != nil {
		return nil, err
	}
	if len(idx.includers) == 0 {
		return nil, nil
	}
	set := make(map[string]bool)
	next := func(p string) []string { return idx.includers[p] }
	for _, p := range paths {
		bfs.Levels(p, next, func(level []string) bool {
			for _, q := range level {
				if !inBatch[q] {
					set[q] = true
				}
			}
			return false
		})
	}
	return sortedBoolKeys(set), nil
}
