//go:build darwin

package watch

import "pgregory.net/rapid"

// checkCoverage checks the FSEvents backend after one step. One recursive
// stream watches the root, so there is no per-directory watch set to
// compare: the check writes a file in each directory that the step created
// or moved, and waits for the watcher to report each one.
func checkCoverage(t *rapid.T, h *watchHarness, _ dirTree, res stepResult) {
	var dirs []string
	for _, d := range res.changed {
		if covered(d) {
			dirs = append(dirs, d)
		}
	}
	h.waitFor(t, h.probe(t, dirs))
}
