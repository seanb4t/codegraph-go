//go:build linux

package watch

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestZZMoveOutOfExcludedDir is a temporary probe for the CI runner: it
// reports the mount that holds the temporary tree and how long a renamed
// directory stays invisible to stat after the rename returns.
func TestZZMoveOutOfExcludedDir(t *testing.T) {
	if mounts, err := os.ReadFile("/proc/mounts"); err == nil {
		tmp := os.TempDir()
		best := ""
		for _, l := range strings.Split(string(mounts), "\n") {
			f := strings.Fields(l)
			if len(f) >= 3 && strings.HasPrefix(tmp, f[1]) && len(f[1]) >= len(best) {
				best = f[1]
				t.Logf("mount for %s: %s", tmp, l)
			}
		}
	}
	for i := 0; i < 40; i++ {
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

		from, to := filepath.Join(root, ".venv", "c"), filepath.Join(root, "c")
		if i%2 == 1 {
			from = filepath.Join(root, "a")
		}
		if err := os.Mkdir(from, 0o755); err != nil {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
		if err := os.Rename(from, to); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		var statErr error
		for {
			_, statErr = os.Stat(to)
			if statErr == nil || time.Since(start) > 2*time.Second {
				break
			}
			time.Sleep(time.Millisecond)
		}
		if statErr != nil || time.Since(start) > 2*time.Millisecond {
			t.Logf("iteration %d (from %s): stat of the destination took %s, final error %v, not-exist %v", i, filepath.Base(filepath.Dir(from)), time.Since(start), statErr, errors.Is(statErr, fs.ErrNotExist))
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
		if !watched[to] || !w.dirs[to] {
			t.Errorf("iteration %d (from %s): c not watched; list %v dirs %v", i, filepath.Base(filepath.Dir(from)), w.fsw.WatchList(), w.dirs)
		}
		cancel()
		<-done
		w.Close()
	}
}
