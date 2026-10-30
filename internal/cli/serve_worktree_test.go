package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/wtindex"
)

func realPath(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", path, err)
	}
	return real
}

func pathExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	return err == nil
}

// graphRecords returns the sorted node ids and "source kind target" edges
// of the store at storeDir.
func graphRecords(t *testing.T, storeDir string) []string {
	t.Helper()
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("Open %s: %v", storeDir, err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	var out []string
	nit, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		out = append(out, "node "+nit.Node().Id)
	}
	eit, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		out = append(out, "edge "+e.Source+" "+e.Kind+" "+e.Target)
	}
	if nit.Err() != nil || eit.Err() != nil {
		t.Fatalf("iteration: %v, %v", nit.Err(), eit.Err())
	}
	slices.Sort(out)
	return out
}

// requireRoots fails unless indexRoot and confineRoot resolve to the given
// directories.
func requireRoots(t *testing.T, indexRoot, confineRoot, wantIndex, wantConfine string) {
	t.Helper()
	if got, want := realPath(t, indexRoot), realPath(t, wantIndex); got != want {
		t.Errorf("indexRoot = %s; want %s", got, want)
	}
	if got, want := realPath(t, confineRoot), realPath(t, wantConfine); got != want {
		t.Errorf("confineRoot = %s; want %s", got, want)
	}
}

func TestServeWorktreeIndexOptOut(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flag    bool
		env     string
		wantOff bool
	}{
		{"the flag", true, "", true},
		{"the variable at 1", false, "1", true},
		{"the variable at true", false, "true", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(codegraphNoWorktreeIndexEnv, tc.env)
			wt, main := siblingWorktreeFixture(t)

			var stderr bytes.Buffer
			indexRoot, confineRoot, hasIndex, pending, err := servePrepareIndex(
				context.Background(), wt, worktreeIndexOff(tc.flag), wtindex.DefaultLimits, &stderr,
			)
			if err != nil || !hasIndex || pending != nil {
				t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want an index, nothing pending, no error", hasIndex, pending, err)
			}
			if made := pathExists(t, filepath.Join(wt, codegraphDirName)); made == tc.wantOff {
				t.Errorf("worktree index made = %v; want %v", made, !tc.wantOff)
			}
			if tc.wantOff {
				requireRoots(t, indexRoot, confineRoot, main, wt)
			} else {
				requireRoots(t, indexRoot, confineRoot, wt, wt)
			}
		})
	}
}

func TestServePrepareIndexSiblingWorktree(t *testing.T) {
	wt, main := siblingWorktreeFixture(t)
	mainStore := filepath.Join(main, codegraphDirName, storeDirName)
	before := graphRecords(t, mainStore)

	var stderr bytes.Buffer
	indexRoot, confineRoot, hasIndex, pending, err := servePrepareIndex(context.Background(), wt, false, wtindex.DefaultLimits, &stderr)
	if err != nil || !hasIndex || pending != nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want a ready worktree index", hasIndex, pending, err)
	}
	requireRoots(t, indexRoot, confineRoot, wt, wt)
	if !pathExists(t, filepath.Join(wt, codegraphDirName, storeDirName)) {
		t.Errorf("no store in the worktree index")
	}
	if out := runGitC(t, wt, "status", "--porcelain", "--untracked-files=all"); out != "" {
		t.Errorf("git status in the worktree lists new paths:\n%s", out)
	}
	if after := graphRecords(t, mainStore); !slices.Equal(before, after) {
		t.Errorf("the main checkout's graph changed: %d records before, %d after", len(before), len(after))
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q; want nothing", stderr.String())
	}
}

func TestServePrepareIndexNestedWorktree(t *testing.T) {
	main := copyFixture(t)
	runGitC(t, main, "init")
	runGitC(t, main, "add", "-A")
	runGitC(t, main, "commit", "-m", "init")
	wt := filepath.Join(main, ".worktrees", "wt")
	runGitC(t, main, "worktree", "add", "-b", "wt", wt)
	if _, _, err := execCmd("init", main); err != nil {
		t.Fatalf("init: %v", err)
	}

	var stderr bytes.Buffer
	indexRoot, confineRoot, hasIndex, pending, err := servePrepareIndex(context.Background(), wt, false, wtindex.DefaultLimits, &stderr)
	if err != nil || !hasIndex || pending != nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want a ready worktree index", hasIndex, pending, err)
	}
	requireRoots(t, indexRoot, confineRoot, wt, wt)
}

func TestServePrepareIndexSubmodule(t *testing.T) {
	tmp := t.TempDir()
	child := filepath.Join(tmp, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, "child.go"), []byte("package child\n\nfunc Child() {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGitC(t, child, "init")
	runGitC(t, child, "add", "-A")
	runGitC(t, child, "commit", "-m", "child")
	parent := copyFixture(t)
	runGitC(t, parent, "init")
	runGitC(t, parent, "add", "-A")
	runGitC(t, parent, "commit", "-m", "parent")
	runGitC(t, parent, "submodule", "add", child, "sub")
	runGitC(t, parent, "commit", "-m", "add submodule")
	if _, _, err := execCmd("init", parent); err != nil {
		t.Fatalf("init: %v", err)
	}
	sub := filepath.Join(parent, "sub")

	var stderr bytes.Buffer
	indexRoot, _, hasIndex, pending, err := servePrepareIndex(context.Background(), sub, false, wtindex.DefaultLimits, &stderr)
	if err != nil || !hasIndex || pending != nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want the super-repository's index", hasIndex, pending, err)
	}
	if got, want := realPath(t, indexRoot), realPath(t, parent); got != want {
		t.Errorf("indexRoot = %s; want the super-repository %s", got, want)
	}
	if pathExists(t, filepath.Join(sub, codegraphDirName)) {
		t.Errorf("servePrepareIndex made an index in the submodule")
	}
}

func TestServePrepareIndexMainNotIndexed(t *testing.T) {
	main := copyFixture(t)
	runGitC(t, main, "init")
	runGitC(t, main, "add", "-A")
	runGitC(t, main, "commit", "-m", "init")
	wt := filepath.Join(t.TempDir(), "sibling")
	runGitC(t, main, "worktree", "add", "-b", "sibling", wt)

	var stderr bytes.Buffer
	_, _, hasIndex, pending, err := servePrepareIndex(context.Background(), wt, false, wtindex.DefaultLimits, &stderr)
	if err != nil || hasIndex || pending != nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want no index", hasIndex, pending, err)
	}
	if pathExists(t, filepath.Join(wt, codegraphDirName)) || pathExists(t, filepath.Join(main, codegraphDirName)) {
		t.Errorf("servePrepareIndex made an index although the main checkout has none")
	}
}

func TestServePrepareIndexReadOnlyWorktree(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to a read-only directory")
	}
	wt, main := siblingWorktreeFixture(t)
	if err := os.Chmod(wt, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(wt, 0o755); err != nil {
			t.Errorf("restore the worktree's mode: %v", err)
		}
	})

	var stderr bytes.Buffer
	indexRoot, confineRoot, hasIndex, pending, err := servePrepareIndex(context.Background(), wt, false, wtindex.DefaultLimits, &stderr)
	if err != nil || !hasIndex || pending != nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want the borrowed index and no error", hasIndex, pending, err)
	}
	requireRoots(t, indexRoot, confineRoot, main, wt)
	if !strings.Contains(stderr.String(), "could not make an index for the worktree") || !strings.Contains(stderr.String(), "permission denied") {
		t.Errorf("stderr = %q; want the reason the worktree index could not be made", stderr.String())
	}
	if pathExists(t, filepath.Join(wt, codegraphDirName)) {
		t.Errorf("an index exists in the read-only worktree")
	}
}

// pendingWorktreeBuild returns a sibling worktree with two new files and the
// build servePrepareIndex leaves pending for it at a foreground limit of 1.
func pendingWorktreeBuild(t *testing.T) (wt, indexRoot string, pending *wtindex.Build) {
	t.Helper()
	wt, main := siblingWorktreeFixture(t)
	for name, src := range map[string]string{
		"onlyhere.go": "package fixture\n\nfunc OnlyHere() {}\n",
		"other.go":    "package fixture\n\nfunc Other() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(wt, name), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	var stderr bytes.Buffer
	indexRoot, confineRoot, hasIndex, pending, err := servePrepareIndex(context.Background(), wt, false, wtindex.Limits{Foreground: 1, SeededFraction: 1}, &stderr)
	if err != nil || !hasIndex || pending == nil {
		t.Fatalf("servePrepareIndex = hasIndex %v, pending %v, err %v; want a pending build", hasIndex, pending, err)
	}
	requireRoots(t, indexRoot, confineRoot, main, wt)
	return wt, indexRoot, pending
}

// syncWriter is an io.Writer a test can read while another goroutine
// writes to it. Each write also signals on wrote.
type syncWriter struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	wrote chan struct{}
}

func newSyncWriter() *syncWriter {
	return &syncWriter{wrote: make(chan struct{}, 1)}
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(p)
	select {
	case w.wrote <- struct{}{}:
	default:
	}
	return n, err
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// awaitDaemonLock waits until <root>/.codegraph/daemon.lock names this
// process, watching the filesystem rather than polling. It fails at once
// when stderr reports that the build or the watcher failed, and shows
// stderr on every failure.
func awaitDaemonLock(t *testing.T, root string, stderr *syncWriter) {
	t.Helper()
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("fsnotify: %v", err)
	}
	defer w.Close()
	if err := w.Add(root); err != nil {
		t.Fatalf("watch %s: %v", root, err)
	}
	indexDir := filepath.Join(root, codegraphDirName)
	watchingIndex := false
	start := time.Now()
	var wakes []string
	note := func(format string, a ...any) {
		wakes = append(wakes, time.Since(start).Round(time.Microsecond).String()+": "+fmt.Sprintf(format, a...))
	}
	timeout := time.After(30 * time.Second)
	for {
		if !watchingIndex && pathExists(t, indexDir) {
			if err := w.Add(indexDir); err != nil {
				t.Fatalf("watch %s: %v", indexDir, err)
			}
			watchingIndex = true
			note("watching %s", indexDir)
		}
		data, err := os.ReadFile(filepath.Join(indexDir, "daemon.lock"))
		note("read daemon.lock: %q, error %v", data, err)
		if err == nil {
			var lock struct {
				PID int `json:"pid"`
			}
			if json.Unmarshal(data, &lock) == nil && lock.PID == os.Getpid() {
				return
			}
		}
		if out := stderr.String(); strings.Contains(out, "could not make an index") || strings.Contains(out, "watcher:") {
			t.Fatalf("the build or the watcher failed; stderr:\n%s", out)
		}
		select {
		case ev := <-w.Events:
			note("event %v", ev)
		case <-stderr.wrote:
			note("stderr written")
		case err := <-w.Errors:
			t.Fatalf("fsnotify: %v; stderr:\n%s", err, stderr.String())
		case <-timeout:
			t.Fatalf("no daemon.lock naming this process appeared in %s within 30s; stderr:\n%s\nwake-ups:\n%s\nwatching: %v\n%s",
				indexDir, stderr.String(), strings.Join(wakes, "\n"), w.WatchList(), daemonWaitState(indexDir, start))
		}
	}
}

// daemonWaitState describes what a failed awaitDaemonLock found: the
// entries of indexDir, its daemon.lock if any and when it was written
// relative to start, and the stacks of the goroutines in serve, daemon,
// wtindex, watch and fsnotify.
func daemonWaitState(indexDir string, start time.Time) string {
	var b strings.Builder
	entries, err := os.ReadDir(indexDir)
	fmt.Fprintf(&b, "entries of %s (error %v):", indexDir, err)
	for _, e := range entries {
		fmt.Fprintf(&b, " %s", e.Name())
	}
	lockPath := filepath.Join(indexDir, "daemon.lock")
	data, err := os.ReadFile(lockPath)
	fmt.Fprintf(&b, "\ndaemon.lock: %q (error %v); this process: %d\n", data, err, os.Getpid())
	if fi, err := os.Stat(lockPath); err == nil {
		fmt.Fprintf(&b, "daemon.lock written %v after the wait began\n", fi.ModTime().Sub(start))
	}
	buf := make([]byte, 1<<22)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "internal/daemon.") || strings.Contains(g, "internal/cli.serveWatchStart") ||
			strings.Contains(g, "internal/wtindex.") || strings.Contains(g, "internal/watch.") ||
			strings.Contains(g, "fsnotify.") || strings.Contains(g, "internal/cli.awaitDaemonLock") {
			b.WriteString(g)
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func TestServeWatchStartBuildsThenWatches(t *testing.T) {
	t.Setenv("CODEGRAPH_NO_WATCH", "")
	wt, indexRoot, pending := pendingWorktreeBuild(t)

	released := make(chan struct{})
	release := sync.OnceFunc(func() { close(released) })
	stderr := newSyncWriter()
	cancel, done := serveWatchStart(indexRoot, true, pending, false, true, indexer.Options{Quiet: true}, stderr, func() { <-released })
	defer func() {
		release()
		cancel()
		<-done
	}()
	if pathExists(t, filepath.Join(wt, codegraphDirName)) {
		t.Fatalf("the worktree index exists before the goroutine's work started: the build ran on the handshake path")
	}
	release()

	awaitDaemonLock(t, wt, stderr)
	if !pathExists(t, filepath.Join(wt, codegraphDirName, storeDirName)) {
		t.Errorf("the watcher runs on the worktree, but the worktree index has no store")
	}
	cancel()
	<-done
	out := stderr.String()
	for _, want := range []string{
		"building an index for the worktree " + pending.WorktreeRoot() + " in the background (" + wtindex.ReasonTooManyChanges + ": 2 of ",
		"the index for the worktree " + pending.WorktreeRoot() + " is ready",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("stderr lacks %q:\n%s", want, out)
		}
	}
}
