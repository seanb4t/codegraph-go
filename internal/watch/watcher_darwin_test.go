//go:build darwin

package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/fsnotify/fsevents"
)

func TestBatchPaths(t *testing.T) {
	const root = "/repo"
	batch := []fsevents.Event{
		{Path: "/repo/main.go", Flags: fsevents.ItemModified},
		{Path: "/repo/.gitignore", Flags: fsevents.ItemModified},
		{Path: "/repo/pkg", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
		{Path: "/REPO/Pkg/b.go", Flags: fsevents.ItemCreated},
		{Path: "/repo/.venv", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
		{Path: "/repo/.venv/lib/site.py", Flags: fsevents.ItemCreated},
		{Path: "/repo/vendor/x/y.go", Flags: fsevents.ItemModified},
		{Path: "/repo/.codegraph/store", Flags: fsevents.MustScanSubDirs},
		{Path: "/repository/main.go", Flags: fsevents.ItemModified},
		{Path: "/other/main.go", Flags: fsevents.ItemModified},
		{Path: "/", Flags: fsevents.MustScanSubDirs | fsevents.KernelDropped},
	}
	got := batchPaths(root, batch)
	want := []string{"/repo/main.go", "/repo/.gitignore", "/repo/pkg", "/REPO/Pkg/b.go", root}
	if !slices.Equal(got, want) {
		t.Fatalf("batchPaths =\n  %q\nwant\n  %q", got, want)
	}
}

// TestOpenHoldsNoDescriptorPerFile pins the reason for the FSEvents
// backend: kqueue holds one descriptor per watched file and directory, so
// a 2,000-file tree would add more than 2,000. The write at the end proves
// the stream delivers events, so a low count cannot come from a watcher
// that watches nothing.
func TestOpenHoldsNoDescriptorPerFile(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root := t.TempDir()
	for d := range 20 {
		dir := filepath.Join(root, fmt.Sprintf("d%02d", d))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		for f := range 100 {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%03d.go", f)), nil, 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
		}
	}

	before := openDescriptors(t)
	w, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer w.Close()
	added := openDescriptors(t) - before
	t.Logf("Open added %d descriptors for a 2,000-file tree", added)
	if added > 20 {
		t.Fatalf("Open added %d descriptors for a 2,000-file tree, want at most 20", added)
	}

	flushed := make(chan map[string]struct{}, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deb := NewDebouncer(ctx, DebounceDuration(), func(paths map[string]struct{}) {
		flushed <- paths
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx, deb)
	}()

	target := filepath.Join(w.root, "d07", "f042.go")
	if err := os.WriteFile(target, []byte("package d07\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	deadline := time.After(5 * time.Second)
	for found := false; !found; {
		select {
		case paths := <-flushed:
			_, found = paths[target]
		case <-deadline:
			t.Fatalf("no flush reported %s", target)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	<-done
}

func openDescriptors(t *testing.T) int {
	t.Helper()
	// Names only: ReadDir would stat each entry, and a descriptor that
	// closes during the listing makes that stat fail.
	dir, err := os.Open("/dev/fd")
	if err != nil {
		t.Fatalf("opening /dev/fd: %v", err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		t.Fatalf("reading /dev/fd: %v", err)
	}
	return len(names)
}
