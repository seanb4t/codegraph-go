//go:build linux

package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestZZMoveOutOfExcludedDir(t *testing.T) {
	for i := 0; i < 20; i++ {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".venv"), 0o755); err != nil {
			t.Fatal(err)
		}
		w, err := Open(root)
		if err != nil {
			t.Fatal(err)
		}
		flushed := make(chan map[string]struct{}, 64)
		ctx, cancel := context.WithCancel(context.Background())
		deb := NewDebouncer(ctx, 5*time.Millisecond, func(p map[string]struct{}) { flushed <- p })
		done := make(chan struct{})
		go func() { defer close(done); w.Run(ctx, deb) }()

		if err := os.Mkdir(filepath.Join(root, ".venv", "c"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(root, ".venv", "c"), filepath.Join(root, "c")); err != nil {
			t.Fatal(err)
		}
		probe := filepath.Join(root, "probe.txt")
		os.WriteFile(probe, []byte("x"), 0o644)
		deadline := time.After(5 * time.Second)
		seen := false
		for !seen {
			select {
			case b := <-flushed:
				if _, ok := b[probe]; ok {
					seen = true
				}
			case <-deadline:
				t.Fatalf("iteration %d: probe never reported", i)
			}
		}
		watched := map[string]bool{}
		for _, p := range w.fsw.WatchList() {
			watched[p] = true
		}
		if !watched[filepath.Join(root, "c")] || !w.dirs[filepath.Join(root, "c")] {
			t.Errorf("iteration %d: c not watched; list %v dirs %v", i, w.fsw.WatchList(), w.dirs)
		}
		cancel()
		<-done
		w.Close()
	}
}
