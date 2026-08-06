package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
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
	deadline := time.Now().Add(testBudget(5 * time.Second))
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

// TestNewPopulatesOpts is the WR-04 regression: New previously never
// assigned its opts parameter to the returned Daemon at all (there wasn't
// even a parameter to assign) — d.opts stayed permanently the zero value,
// making any --workers/--verbose/--quiet customization of the daemon's own
// indexer.Sync calls unreachable. New now threads opts straight through.
func TestNewPopulatesOpts(t *testing.T) {
	root, _, _ := initFixture(t)

	want := indexer.Options{Workers: 3, Verbose: true, Quiet: false}
	d, err := New(root, want)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if d.opts != want {
		t.Fatalf("d.opts = %+v, want %+v — WR-04 regression", d.opts, want)
	}
}

// waitForRegistryRecord polls List() until a record for pid with the given
// repoRoot appears, or the deadline elapses (D-06 — Register happens right
// after acquire, so this mirrors waitForLock's polling shape).
func waitForRegistryRecord(t *testing.T, pid int, repoRoot string) {
	t.Helper()
	deadline := time.Now().Add(testBudget(5 * time.Second))
	for time.Now().Before(deadline) {
		recs, err := List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, r := range recs {
			if r.PID == pid && r.RepoRoot == repoRoot {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for a registry record with pid=%d repoRoot=%s", pid, repoRoot)
}

// assertNoRegistryRecord fails the test if List() still reports a record for
// pid — the D-06 deregister-on-shutdown assertion.
func assertNoRegistryRecord(t *testing.T, pid int) {
	t.Helper()
	recs, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, r := range recs {
		if r.PID == pid {
			t.Fatalf("registry record for pid=%d still present after shutdown: %+v", pid, r)
		}
	}
}

// TestRunRegistersRecordAfterAcquire is the plan's primary D-06 gate: while
// a Run is live, exactly one registry record exists for os.Getpid() naming
// this Daemon's repoRoot.
func TestRunRegistersRecordAfterAcquire(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	withRegistryDir(t)

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)
	waitForRegistryRecord(t, os.Getpid(), root)

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}
}

// TestRunRegistersRecordThatSurvivesPruningOnAgedProcess is the deterministic
// counterpart of the D-06 gate above, covering Run()'s Register() call the way
// the lockfile guard in lock_test.go covers acquire()'s.
//
// Run stamps the registry record it writes for itself, and List() re-derives
// staleness on every read to self-heal records left by crashed daemons.
// Stamping time.Now() rather than this process's actual start time makes the
// record fail that check the moment the daemon is older than
// procStartTimeSlack, so a LIVE daemon's own record is pruned out from under
// it — `codegraph daemon stop` then reports nothing to stop while the daemon
// keeps running and keeps holding the writer.
//
// TestRunRegistersRecordAfterAcquire exercises the same path but only catches
// this once the test binary happens to be old enough, which is what let the
// defect reach CI as an intermittent failure. Forcing the aged condition makes
// it fail every time instead of some of the time.
func TestRunRegistersRecordThatSurvivesPruningOnAgedProcess(t *testing.T) {
	requireAgedProcess(t)

	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	withRegistryDir(t)

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)
	waitForRegistryRecord(t, os.Getpid(), root)

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}
}

// TestRunDeregistersRecordOnCleanShutdown is D-06's other half: after a
// clean ctx-cancel shutdown, the record Register wrote is gone (defer,
// mirroring release()'s shape for the lockfile).
func TestRunDeregistersRecordOnCleanShutdown(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	withRegistryDir(t)

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)
	waitForRegistryRecord(t, os.Getpid(), root)

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}

	assertNoRegistryRecord(t, os.Getpid())
}

// TestRunPolicyDisabledRegistersNothing pins D-06's "registration happens
// only after the lock is held": a policy-disabled Run (watch.DisabledError,
// returned before acquire) must never touch the registry.
func TestRunPolicyDisabledRegistersNothing(t *testing.T) {
	withRegistryDir(t)

	root, _, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{NoWatch: true}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testBudget(2*time.Second))
	defer cancel()

	runErr := d.Run(ctx)
	if !errors.Is(runErr, watch.ErrWatchDisabled) {
		t.Fatalf("Run = %v, want errors.Is(..., watch.ErrWatchDisabled)", runErr)
	}

	recs, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("registry has %d record(s) after a policy-disabled Run, want 0: %+v", len(recs), recs)
	}
}

// TestRunWatchdogCancelsRunOnSimulatedReparent is the primary D-07/D-08
// integration gate: Run starts startWatchdog (07-03) against its own
// derived, cancellable ctx, so a simulated reparent — driven through the
// injectable getppid seam, no forking required — cancels Run itself and
// drives the SAME clean teardown a caller's ctx cancellation would (lock
// released, registry record deregistered) with no new shutdown path.
func TestRunWatchdogCancelsRunOnSimulatedReparent(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	withRegistryDir(t)

	origGetppid := getppid
	t.Cleanup(func() { getppid = origGetppid })
	const original = 424242
	// current is read through the SAME getppid closure both before and
	// after the simulated reparent below — the closure itself is assigned
	// to the package var exactly once, before Run ever starts, so flipping
	// current afterward via atomic.StoreInt32 cannot race against
	// startWatchdog's own read of the getppid var (mirrors
	// watchdog_test.go's TestWatchdogCancelsOnReparent seam pattern).
	var current int32 = original
	getppid = func() int { return int(atomic.LoadInt32(&current)) }

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)
	waitForRegistryRecord(t, os.Getpid(), root)

	// Simulate a reparent: the watchdog's next poll tick observes this and
	// cancels Run's own derived ctx — no external cancel is ever called.
	atomic.StoreInt32(&current, original+1)

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v, want nil — a watchdog-triggered cancel must drive the same clean shutdown as an external ctx cancel (D-08)", err)
		}
	case <-time.After(testBudget(10 * time.Second)):
		// 10s (not the file's usual 5s) — watchdogInterval is a fixed 1s
		// wall-clock ticker inside Run; under heavy full-suite parallel
		// load (many packages' test binaries compiling/running at once)
		// the ticker's next fire after the simulated reparent can be
		// delayed well past 1s, mirroring the load-induced flake class
		// already documented on TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock
		// above.
		t.Fatal("Run did not return after a simulated reparent — watchdog is not wired into Run (D-07/D-08)")
	}

	if _, ok, lerr := readLock(codegraphDir); lerr != nil || ok {
		t.Fatalf("lockfile still present after a watchdog-triggered shutdown (ok=%v err=%v)", ok, lerr)
	}
	assertNoRegistryRecord(t, os.Getpid())
}

// TestDaemonSharedWriter proves the daemon holds exactly one writer: an
// on-disk edit drives a debounced Sync that updates the committed graph
// while the daemon runs, and a second lock acquire fails while it holds
// the lock (INDX-05, SYNC-04).
func TestDaemonSharedWriter(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, storeDir := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true})
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
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

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
	case <-time.After(testBudget(5 * time.Second)):
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
	case <-time.After(testBudget(5 * time.Second)):
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

	d, err := New(root, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)

	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
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
	go func() { runErr2 <- d.Run(ctx2); close(runErr2) }()
	joinDaemonRun(t, cancel2, runErr2)
	waitForLock(t, codegraphDir)
	cancel2()
	select {
	case err := <-runErr2:
		if err != nil {
			t.Fatalf("second Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
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

	d, err := New(root, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	flushStarted := make(chan struct{})
	releaseFlush := make(chan struct{})
	// sync.Once guards the close: under heavy parallel-suite load, fsnotify
	// can deliver the write's events straddling a debounce window boundary,
	// producing a SECOND overlapping fire whose onSyncStart would otherwise
	// panic on a double close (observed as a load-induced flake). The
	// second fire just parks on <-releaseFlush alongside the first; both
	// proceed when it closes, serialized by syncMu.
	var flushStartedOnce sync.Once
	d.onSyncStart = func() {
		flushStartedOnce.Do(func() { close(flushStarted) })
		<-releaseFlush
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)

	writeFixtureFile(t, root, "extra.go", "package main\n\nfunc extra() {}\n")

	select {
	case <-flushStarted:
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("timed out waiting for the debounced flush to start")
	}

	// The flush's fire() goroutine is now blocked mid-flush — the exact
	// CR-01 window. Cancel ctx: the watcher loop's deb.Stop() cannot
	// cancel a timer that has already fired.
	cancel()

	select {
	case err := <-runErr:
		t.Fatalf("Run returned (err=%v) while a debounced flush was still in flight — CR-01 regression", err)
	case <-time.After(testBudget(200 * time.Millisecond)):
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
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after the in-flight flush completed")
	}

	if _, ok, lerr := readLock(codegraphDir); lerr != nil || ok {
		t.Fatalf("lockfile still present after flush completed and Run returned (ok=%v err=%v)", ok, lerr)
	}
}

// TestDaemonFlushLockRequeueGivesUpPerEpisode pins the WR-01/IN-03
// give-up arithmetic in Run's requeue wrapper. The syncFn seam injects
// graphstore.ErrStoreLocked — the exact sentinel the wrapper's errors.Is
// branch keys on — while "contended", so one organic watcher event must
// produce EXACTLY 6 sync attempts (1 organic + maxFlushLockRequeues
// requeues: requeues fire at n=1..5, the give-up branch at n=6) and then
// stop — the give-up branch never requeues. A second fresh event must get
// its own 6 attempts (pinning the counter reset AT give-up, not only on
// success: a broken reset would leave n=6 and give up after a single
// attempt). Once contention "lifts", a third event's sync must succeed —
// exhaustion is per-episode, never terminal for the daemon.
//
// The seam is used instead of holding a real graphstore.Open handle (the
// review's first-choice recipe) because a pebble.Open failing on a held
// LOCK can leak its disk-health ticker goroutine, tripping this package's
// goleak TestMain gate; real ErrStoreLocked propagation through
// indexer.Sync is covered by the integration live-sync test. All
// synchronization is event-driven via the onSync seam.
func TestDaemonFlushLockRequeueGivesUpPerEpisode(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "10")

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var locked atomic.Bool
	locked.Store(true)
	d.syncFn = func(_, _ string, _ indexer.Options) (indexer.Stats, error) {
		if locked.Load() {
			return indexer.Stats{}, fmt.Errorf("%w: injected contention", graphstore.ErrStoreLocked)
		}
		return indexer.Stats{}, nil
	}
	synced := make(chan error, 32)
	d.onSync = func(_ indexer.Stats, err error) { synced <- err }

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)

	// awaitEpisode drives one contention episode: write a fresh file (one
	// organic event) and require exactly `want` consecutive lock-lost sync
	// attempts, followed by a quiet gap proving the give-up branch did NOT
	// requeue a further attempt.
	awaitEpisode := func(rel string, want int) {
		t.Helper()
		writeFixtureFile(t, root, rel, "package main\n")
		for i := 1; i <= want; i++ {
			select {
			case err := <-synced:
				if !errors.Is(err, graphstore.ErrStoreLocked) {
					t.Fatalf("episode %q attempt %d: sync error = %v, want errors.Is(..., graphstore.ErrStoreLocked)", rel, i, err)
				}
			case <-time.After(testBudget(10 * time.Second)):
				t.Fatalf("episode %q: timed out waiting for lock-lost sync attempt %d of %d — the requeue chain stopped early (counter not reset at give-up?)", rel, i, want)
			}
		}
		// The give-up branch must not requeue: a further attempt would
		// arrive one debounce window (~10ms) after the give-up; a 1s quiet
		// gap is a generous margin and cannot flake in the pass direction —
		// with no new events, nothing legitimate can fire.
		select {
		case err := <-synced:
			t.Fatalf("episode %q: a %dth sync attempt fired after exhaustion (err=%v) — the give-up branch requeued", rel, want+1, err)
		case <-time.After(testBudget(1 * time.Second)):
		}
	}

	// Episode 1: 1 organic + maxFlushLockRequeues requeued attempts.
	awaitEpisode("episode1.go", maxFlushLockRequeues+1)
	// Episode 2: the reset at give-up gives a fresh episode its own budget.
	awaitEpisode("episode2.go", maxFlushLockRequeues+1)

	// Contention lifts: the next organic event's sync must succeed.
	locked.Store(false)
	writeFixtureFile(t, root, "episode3.go", "package main\n")
	select {
	case err := <-synced:
		if err != nil {
			t.Fatalf("post-contention sync = %v, want success once the lock is free", err)
		}
	case <-time.After(testBudget(10 * time.Second)):
		t.Fatal("timed out waiting for a successful sync after contention lifted")
	}

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(10 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}
}

// TestRunReturnsErrWatcherClosedAndReleasesLock pins the IN-07 abnormal
// teardown path (WR-01 coverage gap): closing the watcher's event stream
// out from under a running Run — via the onWatchOpen test seam — must make
// Run tear down through the normal join path, RELEASE the daemon lockfile
// (no zombie lock-holder), and return ErrWatcherClosed. Driving it through
// RunWithRetry additionally pins that the error is surfaced immediately:
// it is neither ErrLockLive nor watch.ErrWatchDisabled, so onDeferred must
// never be called and no retry may occur.
func TestRunReturnsErrWatcherClosedAndReleasesLock(t *testing.T) {
	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	watcherCh := make(chan *watch.Watcher, 1)
	d.onWatchOpen = func(w *watch.Watcher) { watcherCh <- w }

	// A real, cancellable ctx (rather than a bare context.Background())
	// gives joinDaemonRun's t.Cleanup an actual escape hatch: the expected
	// path is RunWithRetry returning on its own via ErrWatcherClosed below,
	// but if that path ever fails to fire under contention, cancel() is
	// what lets the join actually unblock the spawned goroutine instead of
	// waiting on a context that can never become Done.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	retryErr := make(chan error, 1)
	go func() {
		retryErr <- RunWithRetry(ctx, d, time.Hour, func() {
			t.Error("onDeferred called — RunWithRetry must surface ErrWatcherClosed immediately, not retry it")
		})
		close(retryErr)
	}()
	joinDaemonRun(t, cancel, retryErr)

	var w *watch.Watcher
	select {
	case w = <-watcherCh:
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("timed out waiting for Run to open its watcher")
	}
	waitForLock(t, codegraphDir)

	// Close the event stream out from under the running watch loop —
	// fsnotify closes Events/Errors, watchLoop returns with ctx still
	// alive: the exact abnormal teardown ErrWatcherClosed exists for.
	if err := w.Close(); err != nil {
		t.Fatalf("closing the watcher out from under Run: %v", err)
	}

	select {
	case err := <-retryErr:
		if !errors.Is(err, ErrWatcherClosed) {
			t.Fatalf("RunWithRetry = %v, want errors.Is(..., ErrWatcherClosed)", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after its watcher's event stream closed — zombie lock-holder (IN-07 regression)")
	}

	if _, ok, lerr := readLock(codegraphDir); lerr != nil || ok {
		t.Fatalf("lockfile still present after ErrWatcherClosed teardown (ok=%v err=%v) — the lock must be released, not held by a dead watcher", ok, lerr)
	}
}

// TestRunPolicyDisabled proves Run enforces watch.WatchDisabledReason as
// its FIRST action, before acquire() (D-11): a Daemon constructed with a
// disabling Probe returns a watch.ErrWatchDisabled-wrapped error and never
// touches the lockfile — the disabled policy check happens strictly before
// lock acquisition, verified here by asserting the lockfile is absent
// after Run returns (the acceptance criteria's companion grep asserts the
// same fact structurally, at the source level).
func TestRunPolicyDisabled(t *testing.T) {
	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{NoWatch: true}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testBudget(2*time.Second))
	defer cancel()

	runErr := d.Run(ctx)
	if !errors.Is(runErr, watch.ErrWatchDisabled) {
		t.Fatalf("Run = %v, want errors.Is(..., watch.ErrWatchDisabled)", runErr)
	}
	if _, ok, lerr := readLock(codegraphDir); lerr != nil || ok {
		t.Fatalf("lockfile present after a policy-disabled Run (ok=%v err=%v) — acquire() must never be reached", ok, lerr)
	}
}

// TestRunHonorsDefaultProbeOnNonWSLHost proves a Daemon constructed via the
// existing New (default zero Probe) on a non-WSL host proceeds to
// acquire()/watch.Open exactly as today — no behavior change for the
// standalone-daemon happy path. Pins IsWSL to false so the assertion holds
// deterministically even if this test ever runs on a real WSL2 host.
func TestRunHonorsDefaultProbeOnNonWSLHost(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- d.Run(ctx); close(runErr) }()
	joinDaemonRun(t, cancel, runErr)

	waitForLock(t, codegraphDir)

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("Run did not return after cancel")
	}
}

// TestRunWithRetryReturnsNilOnCleanShutdown proves RunWithRetry returns nil
// when d.Run returns nil (a clean ctx-cancelled shutdown) — the common
// case of a single session with no lock contention, where onDeferred must
// never be called.
func TestRunWithRetryReturnsNilOnCleanShutdown(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	root, codegraphDir, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	retryErr := make(chan error, 1)
	go func() {
		retryErr <- RunWithRetry(ctx, d, time.Second, func() {
			t.Error("onDeferred called during a clean, uncontended single-session run")
		})
		close(retryErr)
	}()
	joinDaemonRun(t, cancel, retryErr)

	waitForLock(t, codegraphDir)
	cancel()

	select {
	case err := <-retryErr:
		if err != nil {
			t.Fatalf("RunWithRetry = %v, want nil", err)
		}
	case <-time.After(testBudget(5 * time.Second)):
		t.Fatal("RunWithRetry did not return after ctx cancellation")
	}
}

// TestRunWithRetryReturnsImmediatelyOnDisabled proves RunWithRetry returns
// immediately (no retry) when d.Run returns watch.ErrWatchDisabled, and
// never calls onDeferred — policy is terminal, it does not change
// mid-session.
func TestRunWithRetryReturnsImmediatelyOnDisabled(t *testing.T) {
	root, _, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{NoWatch: true}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	called := false
	err = RunWithRetry(context.Background(), d, time.Hour, func() { called = true })
	if !errors.Is(err, watch.ErrWatchDisabled) {
		t.Fatalf("RunWithRetry = %v, want errors.Is(..., watch.ErrWatchDisabled)", err)
	}
	if called {
		t.Fatal("onDeferred was called on a policy-disabled error — RunWithRetry must return immediately without retrying")
	}
}

// TestRunWithRetryReturnsImmediatelyOnGenuineError proves RunWithRetry
// returns immediately (no retry) on a genuine, non-ErrLockLive,
// non-ErrWatchDisabled error, and never calls onDeferred. Removing root
// out from under the daemon after New (but before Run) forces acquire()'s
// createLockExclusive to fail on a nonexistent .codegraph/ directory — an
// error distinct from both sentinels.
func TestRunWithRetryReturnsImmediatelyOnGenuineError(t *testing.T) {
	root, _, _ := initFixture(t)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll(root): %v", err)
	}

	called := false
	err = RunWithRetry(context.Background(), d, time.Hour, func() { called = true })
	if err == nil {
		t.Fatal("RunWithRetry = nil, want a genuine error after root was removed out from under the daemon")
	}
	if errors.Is(err, ErrLockLive) || errors.Is(err, watch.ErrWatchDisabled) {
		t.Fatalf("RunWithRetry = %v, want a genuine error distinct from ErrLockLive/ErrWatchDisabled", err)
	}
	if called {
		t.Fatal("onDeferred was called on a genuine non-ErrLockLive error — RunWithRetry must return immediately without retrying")
	}
}

// TestRunWithRetryCtxCancelDuringSleep proves two things at once (the
// behavior block's last two bullets): (1) on ErrLockLive, onDeferred is
// invoked before the retry sleep; (2) a ctx cancellation during that sleep
// returns ctx.Err() promptly, well under the full retry interval — not by
// waiting the interval out. Manually holding the lock (rather than running
// a second Daemon) makes d.Run deterministically return ErrLockLive on
// every attempt.
func TestRunWithRetryCtxCancelDuringSleep(t *testing.T) {
	root, codegraphDir, _ := initFixture(t)

	if err := acquire(codegraphDir); err != nil {
		t.Fatalf("acquire (holding the lock ourselves): %v", err)
	}
	defer release(codegraphDir)

	d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	deferredCount := 0
	onDeferred := func() {
		mu.Lock()
		deferredCount++
		mu.Unlock()
	}

	// Long enough that a prompt ctx.Err() proves the sleep — not the
	// interval — elapsed. interval is RunWithRetry's own retry cadence (an
	// input to the SUT under test, not a test-side wait budget), so it is
	// deliberately left unscaled — the property under test is "returns
	// well under interval", which testBudget scaling interval itself would
	// undermine.
	const interval = 2 * time.Second
	retryErr := make(chan error, 1)
	go func() {
		retryErr <- RunWithRetry(ctx, d, interval, onDeferred)
		close(retryErr)
	}()
	joinDaemonRun(t, cancel, retryErr)

	// Give RunWithRetry time to hit ErrLockLive at least once and enter its
	// retry sleep before cancelling.
	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	cancel()

	select {
	case err := <-retryErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunWithRetry = %v, want context.Canceled", err)
		}
		if elapsed := time.Since(start); elapsed >= interval {
			t.Fatalf("RunWithRetry took %v to return after cancel, want well under the %v interval", elapsed, interval)
		}
	case <-time.After(testBudget(1 * time.Second)):
		t.Fatal("RunWithRetry did not return promptly after ctx cancellation during sleep")
	}

	mu.Lock()
	defer mu.Unlock()
	if deferredCount == 0 {
		t.Fatal("onDeferred was never called — RunWithRetry should have hit ErrLockLive at least once before cancel")
	}
}
