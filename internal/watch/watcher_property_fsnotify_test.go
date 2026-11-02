//go:build !darwin

package watch

import (
	"path/filepath"
	"sort"

	"pgregory.net/rapid"
)

// checkCoverage checks the inotify backend after one step: the watch list
// must hold exactly the root and every directory below it that the
// indexer does not exclude. A file written at the root comes first: once
// the watcher has reported it, its loop has handled every earlier event
// of the step.
func checkCoverage(t *rapid.T, h *watchHarness, m dirTree, _ stepResult) {
	h.waitFor(t, h.probe(t, []string{"."}))

	want := map[string]bool{h.root: true}
	for d := range m {
		if covered(d) {
			want[filepath.Join(h.root, d)] = true
		}
	}
	got := make(map[string]bool)
	for _, p := range h.w.fsw.WatchList() {
		got[p] = true
	}
	var missing, extra []string
	for p := range want {
		if !got[p] {
			missing = append(missing, p)
		}
	}
	for p := range got {
		if !want[p] {
			extra = append(extra, p)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		t.Fatalf("watch list differs from the tree: not watched %v, watched but not in the tree %v", missing, extra)
	}
}
