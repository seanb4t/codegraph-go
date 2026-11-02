package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// dirNames are the names the generator gives directories. ".venv" is
// excluded by indexer.ShouldSkipDir, so the property also covers a tree
// with excluded parts.
var dirNames = []string{"a", "b", "c", ".venv"}

// watchHarness runs a Watcher over root and collects the paths of its
// flushes.
type watchHarness struct {
	w       *Watcher
	root    string
	flushed chan map[string]struct{}
	seen    map[string]bool
	probes  int
}

func startHarness(t *rapid.T, root string) *watchHarness {
	w, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	h := &watchHarness{w: w, root: root, flushed: make(chan map[string]struct{}, 64), seen: make(map[string]bool)}
	ctx, cancel := context.WithCancel(context.Background())
	deb := NewDebouncer(ctx, DebounceDuration(), func(paths map[string]struct{}) {
		h.flushed <- paths
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx, deb)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
		w.Close()
	})
	return h
}

// waitFor waits until the watcher has flushed every path in paths.
func (h *watchHarness) waitFor(t *rapid.T, paths []string) {
	deadline := time.After(5 * time.Second)
	for _, p := range paths {
		for !h.seen[p] {
			select {
			case batch := <-h.flushed:
				for q := range batch {
					h.seen[q] = true
				}
			case <-deadline:
				t.Fatalf("the watcher reported no event for %s", p)
			}
		}
	}
}

// probe writes a new file in each directory of dirs, relative to the root,
// and returns the files' paths.
func (h *watchHarness) probe(t *rapid.T, dirs []string) []string {
	var paths []string
	for _, d := range dirs {
		h.probes++
		p := filepath.Join(h.root, d, fmt.Sprintf("probe-%d.txt", h.probes))
		if err := os.WriteFile(p, []byte("probe\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		paths = append(paths, p)
	}
	return paths
}

// covered reports whether no directory on the way from the root to rel is
// one that indexer.ShouldSkipDir excludes.
func covered(rel string) bool {
	for _, name := range strings.Split(rel, string(filepath.Separator)) {
		if indexer.ShouldSkipDir(name) {
			return false
		}
	}
	return true
}

// dirTree is the model: the directories under the root, by relative path.
type dirTree map[string]bool

func (m dirTree) sorted() []string {
	out := make([]string, 0, len(m))
	for d := range m {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// freeName draws a name that parent does not hold yet.
func (m dirTree) freeName(t *rapid.T, parent string) string {
	name := rapid.SampledFrom(dirNames).Draw(t, "name")
	for i := 2; m[filepath.Join(parent, name)]; i++ {
		name = fmt.Sprintf("%s%d", strings.TrimRight(name, "0123456789"), i)
	}
	return name
}

// stepResult is what one step changed: the directories it created or
// moved, including the subdirectories of a moved one.
type stepResult struct {
	desc    string
	changed []string
}

func dirStep(t *rapid.T, root string, m dirTree) stepResult {
	ops := []string{"create"}
	if len(m) > 0 {
		ops = append(ops, "rename", "delete")
	}
	parents := append([]string{"."}, m.sorted()...)
	switch rapid.SampledFrom(ops).Draw(t, "op") {
	case "create":
		parent := rapid.SampledFrom(parents).Draw(t, "parent")
		rel := filepath.Join(parent, m.freeName(t, parent))
		if err := os.Mkdir(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		m[rel] = true
		return stepResult{"create " + rel, []string{rel}}
	case "rename":
		from := rapid.SampledFrom(m.sorted()).Draw(t, "from")
		var targets []string
		for _, p := range parents {
			if p != from && !strings.HasPrefix(p, from+string(filepath.Separator)) {
				targets = append(targets, p)
			}
		}
		parent := rapid.SampledFrom(targets).Draw(t, "to parent")
		to := filepath.Join(parent, m.freeName(t, parent))
		if err := os.Rename(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
			t.Fatalf("rename %s to %s: %v", from, to, err)
		}
		var changed []string
		for _, d := range m.sorted() {
			if d == from || strings.HasPrefix(d, from+string(filepath.Separator)) {
				delete(m, d)
				moved := to + strings.TrimPrefix(d, from)
				m[moved] = true
				changed = append(changed, moved)
			}
		}
		return stepResult{"rename " + from + " to " + to, changed}
	default:
		dir := rapid.SampledFrom(m.sorted()).Draw(t, "dir")
		if err := os.RemoveAll(filepath.Join(root, dir)); err != nil {
			t.Fatalf("remove %s: %v", dir, err)
		}
		for _, d := range m.sorted() {
			if d == dir || strings.HasPrefix(d, dir+string(filepath.Separator)) {
				delete(m, d)
			}
		}
		return stepResult{"delete " + dir, nil}
	}
}

// TestWatcherCoversEveryDirectory runs a watcher over a tree, applies a
// generated sequence of directory creates, renames and deletes, and checks
// after each step that the watcher covers every directory that the indexer
// does not exclude (the #208 class). checkCoverage holds the check of each
// backend.
func TestWatcherCoversEveryDirectory(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "5")
	base := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		dir, err := os.MkdirTemp(base, "tree")
		if err != nil {
			t.Fatalf("MkdirTemp: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		// FSEvents reports real paths (/private/var/..., not /var/...).
		root, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("EvalSymlinks: %v", err)
		}
		h := startHarness(t, root)
		m := make(dirTree)

		steps := rapid.IntRange(1, 5).Draw(t, "steps")
		made := 0
		for i := 0; i < steps; i++ {
			res := dirStep(t, root, m)
			made++
			t.Logf("step %d: %s", i+1, res.desc)
			checkCoverage(t, h, m, res)
		}
		if made == 0 {
			t.Fatal("the run made no step")
		}
	})
}
