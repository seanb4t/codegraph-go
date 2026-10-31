package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// waitFlush blocks until a flush arrives on ch or timeout elapses, failing
// the test on timeout.
func waitFlush(t *testing.T, ch <-chan map[string]struct{}, timeout time.Duration) map[string]struct{} {
	t.Helper()
	select {
	case paths := <-ch:
		return paths
	case <-time.After(timeout):
		t.Fatal("timed out waiting for flush")
		return nil
	}
}

// TestWatcherRecursiveAdd proves a file created inside a newly-created
// subdirectory emits an event: fsnotify does not recurse, so this only
// passes if Run's Create handling re-adds the new subdirectory to the
// watch set (Pattern 3).
func TestWatcherRecursiveAdd(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root := t.TempDir()
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

// TestAddCreatedDirSkipsExcludedDirs covers the Create-event path: a
// directory that ShouldSkipDir excludes and that appears after Open (a
// .venv/ made by `uv sync` in a new worktree) must stay out of the watch
// set, together with everything below it. An ordinary new directory must
// still enter it.
func TestAddCreatedDirSkipsExcludedDirs(t *testing.T) {
	root := t.TempDir()
	w, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer w.Close()

	venvLib := filepath.Join(root, ".venv", "lib")
	plainSub := filepath.Join(root, "pkg", "sub")
	for _, d := range []string{venvLib, plainSub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", d, err)
		}
	}

	for _, d := range []string{filepath.Join(root, ".venv"), filepath.Join(root, "pkg")} {
		if err := addCreatedDir(w.fsw, d, indexer.ShouldSkipDir); err != nil {
			t.Fatalf("addCreatedDir %s: %v", d, err)
		}
	}

	watched := make(map[string]bool)
	for _, p := range w.fsw.WatchList() {
		watched[p] = true
	}
	for _, want := range []string{filepath.Join(root, "pkg"), plainSub} {
		if !watched[want] {
			t.Errorf("%s not watched after addCreatedDir; watch list: %v", want, w.fsw.WatchList())
		}
	}
	for _, unwanted := range []string{filepath.Join(root, ".venv"), venvLib} {
		if watched[unwanted] {
			t.Errorf("%s watched, but ShouldSkipDir excludes .venv", unwanted)
		}
	}
}

// TestWatcherErrorsDrained proves an internal fsnotify error does not
// stall subsequent Events processing (Pitfall 6). fsnotify.Watcher's
// Errors field is a plain (non-directional) chan error, so a synthetic
// error can be sent directly into it without needing to fabricate a real
// OS-level watch failure — exercising the exact same select-loop branch a
// genuine error would.
func TestWatcherErrorsDrained(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root := t.TempDir()
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

	// Induce an internal error. The watch loop must log it and continue
	// servicing Events, not deadlock (Pitfall 6).
	w.fsw.Errors <- errors.New("synthetic induced error")

	newFile := filepath.Join(root, "after-error.txt")
	if err := os.WriteFile(newFile, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	paths := waitFlush(t, flushed, 2*time.Second)
	if _, ok := paths[newFile]; !ok {
		t.Fatalf("event for %s not observed after an induced error — watcher stalled (Pitfall 6)", newFile)
	}

	cancel()
	<-done
}
