// Task 2 (D-09): the CLI-side behavioral confirmation that HYG-01's
// quietLogger (Plan 04-01) actually silences Pebble noise on a real,
// subprocess-driven command's stderr — not just in graphstore's own unit
// tests.
package integration

import (
	"strings"
	"testing"
)

// TestSyncStderrNoPebbleNoise drives a real `init` then `sync` through the
// spawned binary and asserts stderr carries none of Pebble's noise SHAPES
// (the job-line prefix, the WAL-line shape, and the compaction/pickAuto
// keywords — 04-RESEARCH.md's confirmed Infof call sites: open.go:1062,
// compaction_picker.go:1374, obsolete_files.go). `sync` (not `status`) is
// driven deliberately: it exercises strictly more of Pebble's Infof surface
// (flush + possible compaction) than a bare read-only status call would
// (04-RESEARCH.md Open Question #1).
//
// This is absence-of-substring on noise SHAPES ONLY — it must NOT assert
// stderr is empty. Legitimate codegraph warnings (e.g. worktree-mismatch
// notices, watcher diagnostics) may still appear on stderr; D-09 only
// forbids Pebble-shaped noise, not stderr output in general.
func TestSyncStderrNoPebbleNoise(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init: %v: %s", err, stderr)
	}

	_, stderr, err := runBinary(t, dir, nil, "sync")
	if err != nil {
		t.Fatalf("sync: %v: %s", err, stderr)
	}

	for _, noise := range []string{"[JOB ", "WAL ", "compaction", "pickAuto"} {
		if strings.Contains(stderr, noise) {
			t.Errorf("sync stderr contains pebble-shaped noise %q:\n%s", noise, stderr)
		}
	}
}
