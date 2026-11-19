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
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
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
	got := batchPaths(root, discover.DirFilter{}, batch)
	want := []string{"/repo/main.go", "/repo/.gitignore", "/repo/pkg", "/REPO/Pkg/b.go", root}
	if !slices.Equal(got, want) {
		t.Fatalf("batchPaths =\n  %q\nwant\n  %q", got, want)
	}
}

func TestBatchPathsSkipsTheIndexBehindALink(t *testing.T) {
	for _, tc := range []struct {
		name   string
		layout func(t *testing.T, root string)
		index  []fsevents.Event
	}{
		{"a .codegraph link", func(t *testing.T, root string) {
			mkdirs(t, filepath.Join(root, "cg-real", "store"))
			symlinkT(t, "cg-real", filepath.Join(root, ".codegraph"))
		}, []fsevents.Event{
			{Path: "cg-real", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
			{Path: "cg-real/store/000001.log", Flags: fsevents.ItemModified},
			{Path: "cg-real/store.swap", Flags: fsevents.ItemCreated},
		}},
		{"a store link", func(t *testing.T, root string) {
			mkdirs(t, filepath.Join(root, ".codegraph"), filepath.Join(root, "store-real"))
			symlinkT(t, filepath.Join("..", "store-real"), filepath.Join(root, ".codegraph", "store"))
		}, []fsevents.Event{
			{Path: "store-real", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
			{Path: "store-real/000001.log", Flags: fsevents.ItemModified},
			{Path: "store-real.swap", Flags: fsevents.ItemCreated},
			{Path: "store-real.write", Flags: fsevents.ItemModified},
			{Path: "store-real.scratch-abc", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
			{Path: "store-real.scratch-abc/store/000002.sst", Flags: fsevents.ItemCreated},
			{Path: "store-real.scratch.pending-abc", Flags: fsevents.ItemCreated | fsevents.ItemIsDir},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			tc.layout(t, root)
			filter, err := discover.NewDirFilter(root)
			if err != nil {
				t.Fatalf("NewDirFilter: %v", err)
			}
			batch := []fsevents.Event{
				{Path: filepath.Join(root, "main.go"), Flags: fsevents.ItemModified},
				{Path: filepath.Join(root, "store-real2.swap"), Flags: fsevents.ItemCreated},
			}
			for _, ev := range tc.index {
				ev.Path = filepath.Join(root, ev.Path)
				batch = append(batch, ev)
			}
			got := batchPaths(root, filter, batch)
			want := []string{filepath.Join(root, "main.go"), filepath.Join(root, "store-real2.swap")}
			if !slices.Equal(got, want) {
				t.Fatalf("batchPaths =\n  %q\nwant\n  %q", got, want)
			}
		})
	}
}

func TestBatchPathsSkipsTheIndexWhenTheRootDiffersInCase(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	disk := filepath.Join(base, "repo")
	mkdirs(t, filepath.Join(disk, ".codegraph"), filepath.Join(disk, "store-real"))
	symlinkT(t, filepath.Join("..", "store-real"), filepath.Join(disk, ".codegraph", "store"))
	root := filepath.Join(base, "REPO")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the volume is case-sensitive: %v", err)
	}
	filter, err := discover.NewDirFilter(root)
	if err != nil {
		t.Fatalf("NewDirFilter: %v", err)
	}

	var batch []fsevents.Event
	for _, name := range []string{"main.go", "store-real.swap", "store-real.write", filepath.Join("store-real", "000001.log")} {
		batch = append(batch, fsevents.Event{Path: filepath.Join(disk, name), Flags: fsevents.ItemModified})
	}
	got := batchPaths(root, filter, batch)
	want := []string{filepath.Join(disk, "main.go")}
	if !slices.Equal(got, want) {
		t.Fatalf("batchPaths =\n  %q\nwant\n  %q", got, want)
	}
}

func TestOpenSkipsTheIndexBehindAStoreLink(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root := t.TempDir()
	mkdirs(t, filepath.Join(root, ".codegraph"), filepath.Join(root, "store-real"))
	symlinkT(t, filepath.Join("..", "store-real"), filepath.Join(root, ".codegraph", "store"))
	w, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer w.Close()

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

	lock := filepath.Join(w.root, "store-real.swap")
	source := filepath.Join(w.root, "main.go")
	for _, p := range []string{lock, source} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	deadline := time.After(5 * time.Second)
	for seen := false; !seen; {
		select {
		case paths := <-flushed:
			if _, ok := paths[lock]; ok {
				t.Fatalf("the event of %s reached the debouncer", lock)
			}
			_, seen = paths[source]
		case <-deadline:
			t.Fatalf("no flush reported %s", source)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	<-done
}

func mkdirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
}

func symlinkT(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
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
