package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWatcherRecursiveAdd proves a file created inside a newly-created
// subdirectory emits an event. fsnotify does not recurse, so on Linux this
// only passes if Run's Create handling re-adds the new subdirectory to the
// watch set (Pattern 3). On macOS it covers the recursive FSEvents stream.
func TestWatcherRecursiveAdd(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	// FSEvents reports real paths (/private/var/..., not /var/...), so the
	// expected event path is built from the resolved root.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
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

	subdir := filepath.Join(root, "newsub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// Give the watcher a moment to observe the Create event and re-add the
	// new subdirectory before writing into it — otherwise the file create
	// below could race the recursive re-add.
	time.Sleep(50 * time.Millisecond)

	newFile := filepath.Join(subdir, "f.txt")
	if err := os.WriteFile(newFile, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	deadline := time.After(2 * time.Second)
	found := false
	for !found {
		select {
		case paths := <-flushed:
			if _, ok := paths[newFile]; ok {
				found = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for an event on %s — recursive add-on-Create did not cover the new subdirectory", newFile)
		}
	}

	cancel()
	<-done
}
