package wtindex

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

var testLimits = Limits{Foreground: 50, SeededFraction: 0.10}

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// worktreeFixture makes an indexed main checkout holding files and a linked
// worktree of it, outside the main checkout, on its own branch.
func worktreeFixture(t *testing.T, files map[string]string) (mainRoot, wt string) {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	mainRoot = filepath.Join(parent, "main")
	writeFiles(t, mainRoot, map[string]string{"go.mod": "module example.com/wt\n\ngo 1.26\n"})
	writeFiles(t, mainRoot, files)
	runGit(t, mainRoot, "init")
	runGit(t, mainRoot, "add", "-A")
	runGit(t, mainRoot, "commit", "-m", "init")

	storeDir := filepath.Join(mainRoot, indexDirName, storeDirName)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	writeFiles(t, mainRoot, map[string]string{filepath.Join(indexDirName, ".gitignore"): "*\n"})
	if _, err := indexer.Run(mainRoot, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index the main checkout: %v", err)
	}

	wt = filepath.Join(parent, "wt")
	runGit(t, mainRoot, "worktree", "add", "-b", "wt", wt)
	return mainRoot, wt
}

// graphOf returns the sorted node ids and "source kind target" edges of the
// store at storeDir.
func graphOf(t *testing.T, storeDir string) (nodes, edges []string) {
	t.Helper()
	if !exists(t, storeDir) {
		t.Fatalf("no store at %s", storeDir)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open %s: %v", storeDir, err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	nit, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		nodes = append(nodes, nit.Node().Id)
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("node iteration: %v", err)
	}
	eit, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		edges = append(edges, e.Source+" "+e.Kind+" "+e.Target)
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("edge iteration: %v", err)
	}
	slices.Sort(nodes)
	slices.Sort(edges)
	return nodes, edges
}

func seedDirs(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), seedPrefix) {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// discard removes a pending build's temporary directory.
func discard(t *testing.T, b *Build) {
	t.Helper()
	if b != nil && b.dir != nil {
		if err := b.dir.remove(); err != nil {
			t.Errorf("remove the seed directory: %v", err)
		}
	}
}

func hasNode(t *testing.T, storeDir, id string) bool {
	t.Helper()
	nodes, _ := graphOf(t, storeDir)
	return slices.Contains(nodes, id)
}

func nodeID(function, file string) string {
	return nodeid.DeclID(goextract.KindFunction, function, file, "()")
}

var onlyHereID = nodeID("OnlyHere", "only.go")

func TestPrepareCommonCase(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	writeFiles(t, wt, map[string]string{
		"only.go": "package wt\n\nfunc OnlyHere() { Base() }\n",
		filepath.Join(seedPrefix+"crashed", lockName): "",
	})
	mainStore := filepath.Join(mainRoot, indexDirName, storeDirName)
	nodesBefore, edgesBefore := graphOf(t, mainStore)

	b, err := Prepare(context.Background(), wt, mainRoot, testLimits)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !b.Ready || b.Reason != "" {
		t.Fatalf("Prepare = Ready %v, Reason %q; want ready", b.Ready, b.Reason)
	}
	if !hasNode(t, filepath.Join(wt, indexDirName, storeDirName), onlyHereID) {
		t.Errorf("the worktree index has no OnlyHere")
	}
	if dirs := seedDirs(t, wt); len(dirs) != 0 {
		t.Errorf("seed directories left in the worktree: %v", dirs)
	}
	nodesAfter, edgesAfter := graphOf(t, mainStore)
	if !slices.Equal(nodesBefore, nodesAfter) || !slices.Equal(edgesBefore, edgesAfter) {
		t.Errorf("the main checkout's graph changed:\nnodes %v -> %v\nedges %v -> %v", nodesBefore, nodesAfter, edgesBefore, edgesAfter)
	}
	if hasNode(t, mainStore, onlyHereID) {
		t.Errorf("the main checkout's index holds OnlyHere")
	}
}

func TestPrepareTooManyChanges(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	writeFiles(t, wt, map[string]string{
		"only.go":  "package wt\n\nfunc OnlyHere() {}\n",
		"other.go": "package wt\n\nfunc Other() {}\n",
	})

	b, err := Prepare(context.Background(), wt, mainRoot, Limits{Foreground: 1, SeededFraction: 1})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer discard(t, b)
	if b.Ready || b.Reason != ReasonTooManyChanges || b.Changed != 2 || b.Total != 3 {
		t.Errorf("Prepare = Ready %v, Reason %q, Changed %d, Total %d; want pending, %q, 2, 3", b.Ready, b.Reason, b.Changed, b.Total, ReasonTooManyChanges)
	}
	if exists(t, filepath.Join(wt, indexDirName)) {
		t.Errorf("Prepare placed an index although too many files changed")
	}
	if !b.seeded || hasNode(t, b.dir.storePath(), onlyHereID) {
		t.Errorf("seeded = %v; want the unsynced copy of the main checkout's index kept for Finish", b.seeded)
	}
}

func TestPrepareMainStoreLocked(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	writeFiles(t, wt, map[string]string{"only.go": "package wt\n\nfunc OnlyHere() {}\n"})
	holder, err := graphstore.Open(filepath.Join(mainRoot, indexDirName, storeDirName))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer holder.Close()

	b, err := Prepare(context.Background(), wt, mainRoot, testLimits)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer discard(t, b)
	if b.Ready || b.Reason != ReasonMainLocked || b.seeded {
		t.Fatalf("Prepare = Ready %v, Reason %q, seeded %v; want pending, %q, not seeded", b.Ready, b.Reason, b.seeded, ReasonMainLocked)
	}
	if exists(t, filepath.Join(wt, indexDirName)) {
		t.Errorf("Prepare placed an index although the main store is locked")
	}

	if err := b.Finish(context.Background()); err != nil {
		t.Fatalf("Finish with the main store still locked: %v", err)
	}
	if !b.Ready || !b.rebuilt || !hasNode(t, filepath.Join(wt, indexDirName, storeDirName), onlyHereID) {
		t.Errorf("after Finish: Ready %v, rebuilt %v; want an index built from the worktree's files", b.Ready, b.rebuilt)
	}
}

func TestPrepareOtherDevice(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	orig := sameDevice
	sameDevice = func(string, string) (bool, error) { return false, nil }
	t.Cleanup(func() { sameDevice = orig })

	b, err := Prepare(context.Background(), wt, mainRoot, testLimits)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	defer discard(t, b)
	if b.Ready || b.Reason != ReasonOtherDevice || b.seeded {
		t.Errorf("Prepare = Ready %v, Reason %q, seeded %v; want pending, %q, not seeded", b.Ready, b.Reason, b.seeded, ReasonOtherDevice)
	}
	if exists(t, b.dir.storePath()) {
		t.Errorf("Prepare copied the main store across devices")
	}
}

func TestPrepareMainOutdated(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func(*schema.Meta)
	}{
		{"a missing sync flag", func(m *schema.Meta) { m.HasMethodSpecs = false }},
		{"an older schema version", func(m *schema.Meta) { m.SchemaVersion = schema.SchemaVersion - 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
			mainStore := filepath.Join(mainRoot, indexDirName, storeDirName)
			func() {
				store, err := graphstore.Open(mainStore)
				if err != nil {
					t.Fatalf("Open: %v", err)
				}
				defer store.Close()
				r, err := store.Snapshot()
				if err != nil {
					t.Fatalf("Snapshot: %v", err)
				}
				meta, err := r.GetMeta()
				r.Close()
				if err != nil {
					t.Fatalf("GetMeta: %v", err)
				}
				tc.alter(meta)
				w, err := store.NewWriter()
				if err != nil {
					t.Fatalf("NewWriter: %v", err)
				}
				if err := w.PutMeta(meta); err != nil {
					t.Fatalf("PutMeta: %v", err)
				}
				if err := w.Commit(); err != nil {
					t.Fatalf("Commit: %v", err)
				}
			}()

			b, err := Prepare(context.Background(), wt, mainRoot, testLimits)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			defer discard(t, b)
			if b.Ready || b.Reason != ReasonMainOutdated || b.seeded {
				t.Errorf("Prepare = Ready %v, Reason %q, seeded %v; want pending, %q, not seeded", b.Ready, b.Reason, b.seeded, ReasonMainOutdated)
			}
			if exists(t, b.dir.storePath()) {
				t.Errorf("Prepare kept a copy that a sync would rebuild")
			}
		})
	}
}

func TestPrepareWorktreeWithIndex(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	writeFiles(t, wt, map[string]string{filepath.Join(indexDirName, "marker"): "theirs"})
	orig := sameDevice
	sameDevice = func(string, string) (bool, error) {
		t.Error("Prepare read the main checkout's store although the worktree has an index")
		return true, nil
	}
	t.Cleanup(func() { sameDevice = orig })

	b, err := Prepare(context.Background(), wt, mainRoot, testLimits)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if !b.Ready {
		t.Errorf("Prepare = pending (%q); want ready, since the worktree has an index", b.Reason)
	}
	if exists(t, filepath.Join(wt, indexDirName, storeDirName)) || len(seedDirs(t, wt)) != 0 {
		t.Errorf("Prepare built an index although the worktree has one")
	}
}

func TestPrepareCancelled(t *testing.T) {
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Prepare(ctx, wt, mainRoot, testLimits); err != context.Canceled {
		t.Fatalf("Prepare error = %v; want context.Canceled", err)
	}
	if exists(t, filepath.Join(wt, indexDirName)) {
		t.Errorf("Prepare placed an index after its context ended")
	}
	if dirs := seedDirs(t, wt); len(dirs) != 0 {
		t.Errorf("seed directories left in the worktree: %v", dirs)
	}
}
