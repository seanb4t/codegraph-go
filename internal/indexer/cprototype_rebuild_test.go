package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// TestCPrototypeRebuildOnce covers a store written before the C prototype
// marks: its Meta record lacks has_c_prototypes. The first sync rebuilds
// every file, and the later syncs reparse only what changed.
func TestCPrototypeRebuildOnce(t *testing.T) {
	root, storeDir := testtree.Write(t, headerPrototypeTree()), t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	clearMetaFlag(t, storeDir, func(m *schema.Meta) { m.HasCPrototypes = false })

	stats, err := Sync(root, storeDir, Options{})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if stats.FilesReparsed != stats.Files || stats.Files != 3 {
		t.Errorf("Sync reparsed %d of %d files, want a full rebuild of 3", stats.FilesReparsed, stats.Files)
	}
	requireSyncEqualsRun(t, root, storeDir)
	if got := callsFrom(t, readGraph(t, storeDir), "main.c", "run"); len(got) != 1 || got[0] != "add.c function add definition" {
		t.Errorf("after the rebuild, calls from run = %q, want the definition in add.c", got)
	}

	for i, src := range []string{
		"#include \"add.h\"\nint run(void) { return add(1, 2) + 0; }\n",
		"#include \"add.h\"\nint run(void) { return add(2, 1); }\n",
	} {
		put(t, root, "main.c", src)
		stats, err := Sync(root, storeDir, Options{})
		if err != nil {
			t.Fatalf("Sync %d: %v", i+2, err)
		}
		if stats.FilesReparsed != 1 {
			t.Errorf("Sync %d reparsed %d files, want 1: the rebuild happens once", i+2, stats.FilesReparsed)
		}
	}
}
