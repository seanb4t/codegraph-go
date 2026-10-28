package wtindex

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// copyTree copies the regular files under src to dst, without .git or
// .codegraph.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == ".git" || d.Name() == indexDirName {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
}

// freshGraph indexes a copy of wt from scratch and returns its graph.
func freshGraph(t *testing.T, wt string) (nodes, edges []string) {
	t.Helper()
	copyRoot := t.TempDir()
	copyTree(t, wt, copyRoot)
	storeDir := filepath.Join(t.TempDir(), storeDirName)
	if _, err := indexer.Run(copyRoot, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index the copy: %v", err)
	}
	return graphOf(t, storeDir)
}

// TestSeededIndexEqualsFreshIndex is the change's acceptance test: a worktree
// index made from the main checkout's index holds the same graph as a fresh
// index of the worktree's files.
func TestSeededIndexEqualsFreshIndex(t *testing.T) {
	mainFiles := map[string]string{
		"caller.go":   "package wt\n\nfunc Caller() { Later() }\n",
		"changed.go":  "package wt\n\nfunc Changed() int { return 1 }\n",
		"removed.go":  "package wt\n\nfunc Removed() {}\n\nfunc UsesRemoved() { Removed() }\n",
		"types.go":    "package wt\n\ntype Shape interface{ Area() int }\n\ntype Square struct{}\n",
		"pkg/util.go": "package pkg\n\nfunc Util() {}\n",
	}
	cases := []struct {
		name        string
		lim         Limits
		wantPending bool
		wantRebuilt bool
	}{
		{"foreground sync", Limits{Foreground: 50, SeededFraction: 0}, false, false},
		{"background sync of the copy", Limits{Foreground: 1, SeededFraction: 1}, true, false},
		{"background index from scratch", Limits{Foreground: 1, SeededFraction: 0}, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mainRoot, wt := worktreeFixture(t, mainFiles)
			if err := os.Remove(filepath.Join(wt, "removed.go")); err != nil {
				t.Fatalf("remove: %v", err)
			}
			writeFiles(t, wt, map[string]string{
				"later.go":    "package wt\n\nfunc Later() {}\n",
				"changed.go":  "package wt\n\nimport \"example.com/wt/pkg\"\n\nfunc Changed() int { pkg.Util(); return Added() }\n\nfunc Added() int { return 2 }\n",
				"square.go":   "package wt\n\nfunc (Square) Area() int { return 4 }\n",
				"pkg/util.go": "package pkg\n\nfunc Util() { Helper() }\n\nfunc Helper() {}\n",
			})
			runGit(t, wt, "add", "-A")
			runGit(t, wt, "commit", "-m", "worktree changes")

			b, err := Prepare(context.Background(), wt, mainRoot, tc.lim)
			if err != nil {
				t.Fatalf("Prepare: %v", err)
			}
			if pending := !b.Ready; pending != tc.wantPending {
				t.Fatalf("Prepare pending = %v (%q); want %v", pending, b.Reason, tc.wantPending)
			}
			if exists(t, filepath.Join(wt, indexDirName)) == tc.wantPending {
				t.Fatalf("worktree index exists after Prepare = %v; want %v", !tc.wantPending, !tc.wantPending)
			}
			if err := b.Finish(context.Background()); err != nil {
				t.Fatalf("Finish: %v", err)
			}
			if !b.Ready || b.rebuilt != tc.wantRebuilt {
				t.Fatalf("after Finish: Ready %v, rebuilt %v; want true, %v", b.Ready, b.rebuilt, tc.wantRebuilt)
			}
			if dirs := seedDirs(t, wt); len(dirs) != 0 {
				t.Errorf("seed directories left in the worktree: %v", dirs)
			}

			gotNodes, gotEdges := graphOf(t, filepath.Join(wt, indexDirName, storeDirName))
			wantNodes, wantEdges := freshGraph(t, wt)
			if !slices.Contains(wantEdges, onlyCallerEdge(t)) {
				t.Fatalf("the fresh index lacks Caller -> Later, so this test would not catch a missed edge: %v", wantEdges)
			}
			if !slices.Equal(gotNodes, wantNodes) {
				t.Errorf("nodes differ from a fresh index:\nseeded %v\nfresh  %v", gotNodes, wantNodes)
			}
			if !slices.Equal(gotEdges, wantEdges) {
				t.Errorf("edges differ from a fresh index:\nonly seeded %v\nonly fresh  %v", minus(gotEdges, wantEdges), minus(wantEdges, gotEdges))
			}
		})
	}
}

func onlyCallerEdge(t *testing.T) string {
	t.Helper()
	return nodeID("Caller", "caller.go") + " calls " + nodeID("Later", "later.go")
}

// minus returns the elements of a that are not in b.
func minus(a, b []string) []string {
	var out []string
	for _, s := range a {
		if !slices.Contains(b, s) {
			out = append(out, s)
		}
	}
	return out
}

// pendingBuild returns a build that Prepare left to Finish, because the
// worktree changes two files and the foreground limit is 1.
func pendingBuild(t *testing.T) (b *Build, wt string) {
	t.Helper()
	mainRoot, wt := worktreeFixture(t, map[string]string{"base.go": "package wt\n\nfunc Base() {}\n"})
	writeFiles(t, wt, map[string]string{
		"only.go":  "package wt\n\nfunc OnlyHere() {}\n",
		"other.go": "package wt\n\nfunc Other() {}\n",
	})
	b, err := Prepare(context.Background(), wt, mainRoot, Limits{Foreground: 1, SeededFraction: 1})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if b.Ready {
		t.Fatalf("Prepare finished the index; want a pending build")
	}
	return b, wt
}

func TestFinishYieldsToExistingIndex(t *testing.T) {
	b, wt := pendingBuild(t)
	marker := filepath.Join(wt, indexDirName, "marker")
	writeFiles(t, wt, map[string]string{filepath.Join(indexDirName, "marker"): "theirs"})

	if err := b.Finish(context.Background()); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if !b.Ready {
		t.Errorf("Finish left the build pending; want ready, since the worktree has an index")
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "theirs" {
		t.Errorf("existing index's marker = %q, %v; want it unchanged", got, err)
	}
	if exists(t, filepath.Join(wt, indexDirName, storeDirName)) {
		t.Errorf("Finish moved its store into the existing index")
	}
	if dirs := seedDirs(t, wt); len(dirs) != 0 {
		t.Errorf("seed directories left in the worktree: %v", dirs)
	}
}

func TestFinishCancelled(t *testing.T) {
	b, wt := pendingBuild(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := b.Finish(ctx); err != context.Canceled {
		t.Fatalf("Finish error = %v; want context.Canceled", err)
	}
	<-b.stopped
	if exists(t, filepath.Join(wt, indexDirName)) {
		t.Errorf("Finish placed an index after its context ended")
	}
	if dirs := seedDirs(t, wt); len(dirs) != 0 {
		t.Errorf("seed directories left in the worktree: %v", dirs)
	}
}
