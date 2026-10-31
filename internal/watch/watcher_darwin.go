//go:build darwin

package watch

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsevents"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// streamLatency is how long FSEvents gathers events before it delivers a
// batch. The Debouncer does the real coalescing, so this stays short.
const streamLatency = 50 * time.Millisecond

// droppedFlags mark a batch in which FSEvents lost events, so the changed
// paths are unknown.
const droppedFlags = fsevents.UserDropped | fsevents.KernelDropped

// Watcher wraps one recursive FSEvents stream over a repo root. FSEvents
// reports changes by path through fseventsd, so the stream holds no file
// descriptor per watched file or directory (kqueue holds one for each),
// and a directory created or renamed into the tree is covered without a
// re-add. It has no graph or storage knowledge of its own — Run's flush
// callback is the caller's seam into indexer.Sync.
type Watcher struct {
	es     *fsevents.EventStream
	root   string
	done   chan struct{}
	closed atomic.Bool
}

// Open starts an FSEvents stream rooted at root. FSEvents reports real
// paths, so root's symlinks are resolved first (fsnotify/fsevents#46):
// a root under /var/folders arrives as /private/var/folders, for example.
func Open(root string) (*Watcher, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("watch: resolving %s: %w", root, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("watch: resolving %s: %w", root, err)
	}
	es := &fsevents.EventStream{
		Paths:   []string{real},
		Flags:   fsevents.FileEvents | fsevents.NoDefer,
		Latency: streamLatency,
		Events:  make(chan []fsevents.Event, 64),
	}
	if err := es.Start(); err != nil {
		return nil, fmt.Errorf("watch: starting FSEvents stream for %s: %w", real, err)
	}
	return &Watcher{es: es, root: real, done: make(chan struct{})}, nil
}

// Run consumes FSEvents batches until ctx is cancelled or Close is called,
// feeding each relevant path into deb so a burst coalesces into one
// debounced flush. Run stops deb's pending timer before it returns.
func (w *Watcher) Run(ctx context.Context, deb *Debouncer) {
	defer deb.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case batch := <-w.es.Events:
			for _, p := range batchPaths(w.root, batch) {
				deb.Add(p)
			}
		}
	}
}

// batchPaths returns the paths in one FSEvents batch that can change the
// index. A batch that lost events yields root: the daemon's sync re-diffs
// the whole repository, so any path is enough to trigger the full check.
func batchPaths(root string, batch []fsevents.Event) []string {
	var paths []string
	for _, ev := range batch {
		switch {
		case ev.Flags&droppedFlags != 0:
			paths = append(paths, root)
		case relevant(root, ev.Path, ev.Flags&(fsevents.ItemIsDir|fsevents.MustScanSubDirs) != 0):
			paths = append(paths, ev.Path)
		}
	}
	return paths
}

// relevant reports whether path is root or lies under it with no directory
// on the way that indexer.ShouldSkipDir excludes, the same rule the indexer
// and the fsnotify backend apply. When isDir is true, path's own name is
// checked too. The prefix comparison ignores case because APFS volumes are
// case-insensitive by default and FSEvents reports the on-disk case.
func relevant(root, path string, isDir bool) bool {
	if len(path) < len(root) || !strings.EqualFold(path[:len(root)], root) {
		return false
	}
	rest := path[len(root):]
	if rest == "" {
		return true
	}
	if rest[0] != filepath.Separator {
		return false
	}
	names := strings.Split(rest[1:], string(filepath.Separator))
	if !isDir {
		names = names[:len(names)-1]
	}
	for _, name := range names {
		if indexer.ShouldSkipDir(name) {
			return false
		}
	}
	return true
}

// Close idempotently stops the stream and ends Run. The fsevents callback
// blocks on its send to es.Events, and Run may already have returned, so
// Close keeps reading the channel until Stop returns. Otherwise a callback
// in flight could hold the stream's dispatch queue forever.
func (w *Watcher) Close() error {
	if w.closed.Swap(true) {
		return nil
	}
	close(w.done)
	stopped := make(chan struct{})
	go func() {
		for {
			select {
			case <-w.es.Events:
			case <-stopped:
				return
			}
		}
	}()
	w.es.Stop()
	close(stopped)
	return nil
}
