package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// siblingWorktreeFixture builds an indexed main checkout and a linked
// worktree in a different temp directory, so the upward walk from the
// worktree never passes through the main checkout (#139).
func siblingWorktreeFixture(t *testing.T) (worktree, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitC(t, main, "init")
	runGitC(t, main, "add", "-A")
	runGitC(t, main, "commit", "-m", "init")

	wt := filepath.Join(t.TempDir(), "sibling")
	runGitC(t, main, "worktree", "add", "-b", "sibling", wt)

	if _, _, err := execCmd("init", main); err != nil {
		t.Fatalf("init fixture: unexpected error: %v", err)
	}
	return wt, main
}

func TestStatusInSiblingWorktree(t *testing.T) {
	wt, main := siblingWorktreeFixture(t)
	realMain, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", main, err)
	}

	out, _, err := execCmd("status", "-p", wt)
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}
	if !strings.Contains(out, "This CodeGraph index belongs to a different git working tree.") {
		t.Fatalf("status: expected the borrowed-index warning, got: %q", out)
	}
	if !strings.Contains(out, "Index from: "+realMain) {
		t.Fatalf("status: expected %q, got: %q", "Index from: "+realMain, out)
	}

	out, _, err = execCmd("explore", "Alpha", "-p", wt)
	if err != nil {
		t.Fatalf("explore: unexpected error: %v", err)
	}
	if glyph := noticeGlyph(t); !strings.HasPrefix(out, glyph) {
		t.Fatalf("explore: expected output to start with the notice glyph %q, got: %q", glyph, out)
	}
}

func TestWriteCommandsIgnoreWorktreeFallback(t *testing.T) {
	wt, main := siblingWorktreeFixture(t)
	mainIndex := filepath.Join(main, codegraphDirName)
	before := treeSnapshot(t, mainIndex)

	_, _, err := execCmd("sync", wt)
	if !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("sync in the sibling worktree: got err %v, want ErrNotInitialized", err)
	}
	if after := treeSnapshot(t, mainIndex); !equalSnapshots(before, after) {
		t.Fatalf("sync in the sibling worktree changed %s:\nbefore %v\nafter  %v", mainIndex, before, after)
	}

	if _, _, err := execCmd("init", wt); err != nil {
		t.Fatalf("init in the sibling worktree: unexpected error: %v", err)
	}
	if info, err := os.Stat(filepath.Join(wt, codegraphDirName)); err != nil || !info.IsDir() {
		t.Fatalf("init in the sibling worktree: %s/%s not created: %v", wt, codegraphDirName, err)
	}
	out, _, err := execCmd("status", "-p", wt)
	if err != nil {
		t.Fatalf("status: unexpected error: %v", err)
	}
	if strings.Contains(out, "different git working tree") {
		t.Fatalf("status after init in the worktree: expected no borrowed-index warning, got: %q", out)
	}
}

type fileStamp struct {
	size    int64
	modTime time.Time
}

// treeSnapshot records every file under root with its size and
// modification time. A non-empty result is required, so a wrong root
// cannot make two snapshots trivially equal.
func treeSnapshot(t *testing.T, root string) map[string]fileStamp {
	t.Helper()
	snap := map[string]fileStamp{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		snap[p] = fileStamp{size: info.Size(), modTime: info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(snap) == 0 {
		t.Fatalf("walk %s: no files", root)
	}
	return snap
}

func equalSnapshots(a, b map[string]fileStamp) bool {
	if len(a) != len(b) {
		return false
	}
	for p, s := range a {
		if o, ok := b[p]; !ok || o.size != s.size || !o.modTime.Equal(s.modTime) {
			return false
		}
	}
	return true
}
