package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// writeFixtureFile materializes rel under root, creating parent
// directories as needed.
func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// initFixture builds a minimal Go module fixture at a fresh temp root and
// runs a from-scratch indexer.Run to populate .codegraph/store — the
// pre-existing index a Daemon's Run/New expects (mirrors `codegraph init`).
func initFixture(t *testing.T) (root, codegraphDir, storeDir string) {
	t.Helper()
	root = t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module example.com/daemonfixture\n\ngo 1.26\n")
	writeFixtureFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	codegraphDir = filepath.Join(root, codegraphDirName)
	storeDir = filepath.Join(codegraphDir, storeDirName)
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", storeDir, err)
	}
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
	}
	return root, codegraphDir, storeDir
}

// waitForLock polls until codegraphDir's daemon lockfile exists or the
// deadline elapses.
func waitForLock(t *testing.T, codegraphDir string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok, err := readLock(codegraphDir); err == nil && ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to appear", lockPath(codegraphDir))
}

// nodeNames returns the set of node names currently in storeDir's graph.
func nodeNames(t *testing.T, storeDir string) map[string]bool {
	t.Helper()
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()

	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()

	names := map[string]bool{}
	for it.Next() {
		names[it.Node().Name] = true
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateNodes iteration: %v", err)
	}
	return names
}

// TestDaemonSharedWriter proves the daemon holds exactly one writer: an
// on-disk edit drives a debounced Sync that updates the committed graph
// while the daemon runs, and a second lock acquire fails while it holds
// the lock (INDX-05, SYNC-04).
func TestDaemonSharedWriter(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, storeDir := initFixture(t)

	d, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	synced := make(chan indexer.Stats, 4)
	d.onSync = func(stats indexer.Stats, err error) {
		if err != nil {
			t.Errorf("daemon-driven sync failed: %v", err)
			return
		}
		synced <- stats
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx) }()

	waitForLock(t, codegraphDir)

	// The single-writer invariant: a second acquire attempt while the
	// daemon holds the lock must be rejected, never silently succeed.
	if err := acquire(codegraphDir); !errors.Is(err, ErrLockLive) {
		t.Fatalf("acquire while daemon running: got %v, want ErrLockLive", err)
	}

	writeFixtureFile(t, root, "extra.go", "package main\n\nfunc extra() {}\n")

	select {
	case stats := <-synced:
		if stats.FilesReparsed == 0 {
			t.Fatal("daemon-driven Sync ran but reparsed 0 files after adding extra.go")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the daemon-driven sync triggered by the file edit")
	}

	if names := nodeNames(t, storeDir); !names["extra"] {
		t.Fatalf("committed graph after daemon-driven sync = %v, want it to contain \"extra\"", names)
	}

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestDaemonCleanShutdown proves Run returns promptly after ctx
// cancellation, with its spawned watcher goroutine joined (sync.WaitGroup,
// D-07) and the lockfile released — no goroutine or lock outlives Run
// (SYNC-06).
func TestDaemonCleanShutdown(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx) }()

	waitForLock(t, codegraphDir)

	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5s of ctx cancellation — a spawned goroutine did not join (D-07)")
	}

	if _, ok, err := readLock(codegraphDir); err != nil || ok {
		t.Fatalf("lockfile still present after clean shutdown (ok=%v err=%v)", ok, err)
	}

	// Run is safely re-runnable after a clean shutdown (acquire succeeds
	// again once the prior Run released the lock).
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	runErr2 := make(chan error, 1)
	go func() { runErr2 <- d.Run(ctx2) }()
	waitForLock(t, codegraphDir)
	cancel2()
	select {
	case err := <-runErr2:
		if err != nil {
			t.Fatalf("second Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second Run did not return within 5s of ctx cancellation")
	}
}

// TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock is the CR-01
// regression: proves Run does not return — and does not release the daemon
// lock — while a debounce-triggered flush is still in flight, even when ctx
// is cancelled while that flush is running. onSyncStart blocks the flush
// deterministically (a real indexer.Sync against a tiny test fixture
// completes far too fast to reliably race against ctx cancellation
// otherwise), reproducing the exact untracked-goroutine window CR-01
// describes: the debounce timer has already fired, so watchLoop's
// deb.Stop() (called on ctx.Done()) cannot cancel it — only deb.Wait()
// (joined from Run after wg.Wait()) can make Run wait correctly.
func TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "10")

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	flushStarted := make(chan struct{})
	releaseFlush := make(chan struct{})
	d.onSyncStart = func() {
		close(flushStarted)
		<-releaseFlush
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx) }()

	waitForLock(t, codegraphDir)

	writeFixtureFile(t, root, "extra.go", "package main\n\nfunc extra() {}\n")

	select {
	case <-flushStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the debounced flush to start")
	}

	// The flush's fire() goroutine is now blocked mid-flush — the exact
	// CR-01 window. Cancel ctx: the watcher loop's deb.Stop() cannot
	// cancel a timer that has already fired.
	cancel()

	select {
	case err := <-runErr:
		t.Fatalf("Run returned (err=%v) while a debounced flush was still in flight — CR-01 regression", err)
	case <-time.After(200 * time.Millisecond):
		// correct: Run must still be blocked.
	}
	if _, ok, lerr := readLock(codegraphDir); lerr != nil || !ok {
		t.Fatalf("daemon lock released while a flush was still in flight (ok=%v err=%v) — CR-01 regression", ok, lerr)
	}

	close(releaseFlush) // let the flush — and its real indexer.Sync — proceed

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the in-flight flush completed")
	}

	if _, ok, lerr := readLock(codegraphDir); lerr != nil || ok {
		t.Fatalf("lockfile still present after flush completed and Run returned (ok=%v err=%v)", ok, lerr)
	}
}
