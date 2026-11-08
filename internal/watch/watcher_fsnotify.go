//go:build !darwin

// This backend uses github.com/fsnotify/fsnotify (inotify on Linux).
// fsnotify does not recurse on its own, so Watcher walks the tree at Open
// time and re-adds any newly-created directory on a Create event.

package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
)

// Watcher wraps a *fsnotify.Watcher covering a repo root recursively. It
// has no graph or storage knowledge of its own — Run's flush callback is
// the caller's seam into indexer.Sync.
type Watcher struct {
	fsw    *fsnotify.Watcher
	closed atomic.Bool
	// dirs holds the directories the watcher watches, so a Rename or
	// Remove event of a file costs no scan of the watch list. Open fills
	// it, and only the Run loop changes it after that.
	dirs map[string]bool
}

// Open opens a native fsnotify watcher rooted at root and walks the tree,
// adding every directory not excluded by discover.ShouldSkipDir to the
// watch set (so .codegraph/, vendor/, and other dot-prefixed directories
// are never watched — the same exclusion discover.go applies, per D-04's
// "watcher and indexer agree on the file set").
func Open(root string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watch: creating fsnotify watcher: %w", err)
	}
	w := &Watcher{fsw: fsw, dirs: make(map[string]bool)}
	if err := addRecursive(w, root, discover.ShouldSkipDir); err != nil {
		_ = fsw.Close()
		return nil, fmt.Errorf("watch: walking %s: %w", root, err)
	}
	return w, nil
}

// addRecursive walks root and calls w.Add on every directory not excluded
// by skip. fsnotify.Add only watches a directory's direct children
// (RESEARCH Pattern 3) — recursive coverage requires this explicit walk at
// startup, plus a re-addRecursive call on any subdirectory discovered
// later via a Create event (see watchLoop).
func addRecursive(w *Watcher, root string, skip func(name string) bool) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p != root && skip(d.Name()) {
			return filepath.SkipDir
		}
		if err := w.fsw.Add(p); err != nil {
			return err
		}
		w.dirs[p] = true
		return nil
	})
}

// addCreatedDir watches a directory that appeared after Open. addRecursive
// never applies skip to its own root, so the created directory's name is
// checked here first: without that check, a new .venv/ or node_modules/
// sibling of a watched directory is walked in full, and kqueue holds one
// file descriptor for every file in it. A path that is not a directory is
// ignored.
func addCreatedDir(w *Watcher, path string, skip func(name string) bool) error {
	if skip(filepath.Base(path)) {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Created and removed before the event was handled.
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return addRecursive(w, path, skip)
}

// removeMoved drops the watch of a directory that moved away from path
// or was deleted, and the watches of every directory below it. The
// removal must happen before the Create event of the directory's new
// path, or of a new directory under the same name, is handled: a watch
// left under the old path is dropped later by fsnotify, and that drop
// takes the fresh watch with it. A path that is not a watched directory,
// such as a file, costs one map lookup.
func removeMoved(w *Watcher, path string) {
	if !w.dirs[path] {
		return
	}
	prefix := path + string(filepath.Separator)
	for p := range w.dirs {
		if p != path && !strings.HasPrefix(p, prefix) {
			continue
		}
		delete(w.dirs, p)
		if err := w.fsw.Remove(p); err != nil && !errors.Is(err, fsnotify.ErrNonExistentWatch) {
			log.Printf("watch: removing moved directory %s: %v", p, err)
		}
	}
}

// Run consumes fsnotify events until ctx is cancelled, feeding each
// changed path into deb so a burst coalesces into one debounced flush
// (Pattern 3). Run returns (and stops deb's pending timer) once ctx is
// done or the watcher's channels are closed.
func (w *Watcher) Run(ctx context.Context, deb *Debouncer) {
	watchLoop(ctx, w, deb)
}

// watchLoop selects on both Events AND Errors in a single loop (Pitfall
// 6): fsnotify's Errors channel must be drained or an internal error (a
// removed watch target, an overflowed event queue, ...) can block
// fsnotify's own goroutine indefinitely. Errors are logged, never treated
// as fatal — a stuck-not-crashed watcher is exactly the failure mode
// Pitfall 6 warns about.
func watchLoop(ctx context.Context, w *Watcher, deb *Debouncer) {
	for {
		select {
		case <-ctx.Done():
			deb.Stop()
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				deb.Stop()
				return
			}
			if ev.Has(fsnotify.Rename) || ev.Has(fsnotify.Remove) {
				// Removed here, before the Create of a directory under
				// the same name is handled: see removeMoved.
				removeMoved(w, ev.Name)
			}
			if !ev.Has(fsnotify.Remove) && !w.dirs[ev.Name] {
				// A directory that is not watched yet: created, moved in,
				// or seen through another event when its create event was
				// not delivered. A file or a gone path costs one stat.
				if err := addCreatedDir(w, ev.Name, discover.ShouldSkipDir); err != nil {
					log.Printf("watch: adding new directory %s: %v", ev.Name, err)
				}
			}
			deb.Add(ev.Name)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				deb.Stop()
				return
			}
			// Logged, not fatal (Pitfall 6): the loop keeps servicing
			// Events after an internal fsnotify error.
			log.Printf("watch: fsnotify error: %v", err)
		}
	}
}

// Close idempotently releases the underlying fsnotify watcher — the same
// atomic.Bool-swap-guarded idiom internal/graphstore's pebbleStore.Close
// uses, since closing an *fsnotify.Watcher a second time can also
// misbehave.
func (w *Watcher) Close() error {
	if w.closed.Swap(true) {
		return nil
	}
	return w.fsw.Close()
}
